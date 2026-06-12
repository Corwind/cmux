package app

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/ports"
	"github.com/google/uuid"
)

// WorktreeSpec describes how to create or attach a git worktree for a session.
type WorktreeSpec struct {
	RepoPath     string
	Branch       string
	BaseRef      string
	CreateBranch bool
	Path         string // if empty, computed from worktreesDir
}

// CreateSessionInput holds all parameters for creating a session.
type CreateSessionInput struct {
	Name            string
	WorkingDir      string
	TemplateID      string
	SkipPermissions bool
	Worktree        *WorktreeSpec
}

type SessionService struct {
	repo           ports.SessionRepository
	processManager ports.ProcessManager
	templateRepo   ports.TemplateRepository
	gitService     ports.GitService
	worktreeRepo   ports.WorktreeRepository
	worktreesDir   string
	mu             sync.RWMutex
}

func NewSessionService(repo ports.SessionRepository, pm ports.ProcessManager, templateRepo ports.TemplateRepository, opts ...SessionServiceOption) *SessionService {
	s := &SessionService{
		repo:           repo,
		processManager: pm,
		templateRepo:   templateRepo,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type SessionServiceOption func(*SessionService)

func WithGitService(svc ports.GitService, worktreesDir string) SessionServiceOption {
	return func(s *SessionService) {
		s.gitService = svc
		s.worktreesDir = worktreesDir
	}
}

func WithWorktreeRepository(repo ports.WorktreeRepository) SessionServiceOption {
	return func(s *SessionService) {
		s.worktreeRepo = repo
	}
}

func (s *SessionService) CreateSession(ctx context.Context, input CreateSessionInput) (domain.Session, error) {
	workingDir := input.WorkingDir
	var repoRoot, gitBranch string
	var worktreeManaged bool

	if input.Worktree != nil && s.gitService != nil {
		spec := input.Worktree

		// Validate repo
		info, err := s.gitService.Info(spec.RepoPath)
		if err != nil {
			return domain.Session{}, fmt.Errorf("git info: %w", err)
		}
		if !info.IsRepo {
			return domain.Session{}, fmt.Errorf("%q is not a git repository", spec.RepoPath)
		}

		// Compute worktree path
		wtPath := spec.Path
		if wtPath == "" {
			safeBranch := strings.ReplaceAll(spec.Branch, "/", "-")
			repoBase := filepath.Base(info.RepoRoot)
			wtPath = filepath.Join(s.worktreesDir, repoBase, safeBranch)
		}

		wt, err := s.gitService.AddWorktree(info.RepoRoot, wtPath, spec.Branch, spec.BaseRef, spec.CreateBranch)
		if err != nil {
			return domain.Session{}, fmt.Errorf("create worktree: %w", err)
		}

		workingDir = wt.Path
		repoRoot = info.RepoRoot
		gitBranch = wt.Branch
		worktreeManaged = true
	}

	session, err := domain.NewSession(input.Name, workingDir)
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid session: %w", err)
	}
	session.RepoRoot = repoRoot
	session.GitBranch = gitBranch
	session.WorktreeManaged = worktreeManaged

	// Resolve sandbox template content
	s.applySandboxContent(ctx, input.TemplateID)

	args := []string{"--session-id", session.ClaudeSessionID}
	if input.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	handle, err := s.processManager.Spawn(ctx, workingDir, args...)
	if err != nil {
		// Best-effort cleanup of orphaned worktree
		if worktreeManaged && s.gitService != nil {
			_ = s.gitService.RemoveWorktree(repoRoot, workingDir, true)
		}
		return domain.Session{}, fmt.Errorf("failed to spawn process: %w", err)
	}

	session.PID = handle.PID
	session.Status = domain.StatusRunning
	session.TemplateID = input.TemplateID
	session.SkipPermissions = input.SkipPermissions

	if err := s.repo.Create(ctx, session); err != nil {
		_ = s.processManager.Kill(handle.PID)
		if worktreeManaged && s.gitService != nil {
			_ = s.gitService.RemoveWorktree(repoRoot, workingDir, true)
		}
		return domain.Session{}, fmt.Errorf("failed to store session: %w", err)
	}

	if worktreeManaged && s.worktreeRepo != nil {
		s.trackWorktreeSession(ctx, session)
	}

	go s.watchProcess(session.ID, handle)

	return session, nil
}

// trackWorktreeSession creates or adopts a ManagedWorktree record for the given
// session and sets the session_id FK on the worktree record.
func (s *SessionService) trackWorktreeSession(ctx context.Context, session domain.Session) {
	wt, err := s.worktreeRepo.GetByPath(ctx, session.WorkingDir)
	if err != nil {
		wt = domain.ManagedWorktree{
			ID:        uuid.New().String(),
			Path:      session.WorkingDir,
			Branch:    session.GitBranch,
			RepoRoot:  session.RepoRoot,
			CreatedAt: session.CreatedAt,
		}
		if createErr := s.worktreeRepo.Create(ctx, wt); createErr != nil {
			log.Printf("failed to create worktree record for %s: %v", session.WorkingDir, createErr)
			return
		}
	}
	if setErr := s.worktreeRepo.SetSession(ctx, wt.ID, &session.ID); setErr != nil {
		log.Printf("failed to set session %s on worktree %s: %v", session.ID, wt.ID, setErr)
	}
}

func (s *SessionService) applySandboxContent(ctx context.Context, templateID string) {
	if s.templateRepo == nil {
		return
	}

	provider, ok := s.processManager.(ports.SandboxContentProvider)
	if !ok {
		return
	}

	var tmpl domain.SandboxTemplate
	var err error

	if templateID != "" {
		tmpl, err = s.templateRepo.Get(ctx, templateID)
	} else {
		tmpl, err = s.templateRepo.GetDefault(ctx)
	}

	if err != nil {
		if templateID != "" {
			log.Printf("failed to resolve template %s: %v", templateID, err)
		}
		return
	}

	provider.SetSandboxContent([]string{tmpl.Content})
}

func (s *SessionService) ResumeSession(ctx context.Context, id string) (domain.Session, error) {
	session, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}
	if session.Status == domain.StatusRunning && s.processManager.IsAlive(session.PID) {
		return session, nil
	}

	// Reapply the sandbox template that was used when the session was created
	s.applySandboxContent(ctx, session.TemplateID)

	args := []string{"--resume", session.ClaudeSessionID}
	if session.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	handle, err := s.processManager.Spawn(ctx, session.WorkingDir, args...)
	if err != nil {
		return domain.Session{}, fmt.Errorf("failed to resume process: %w", err)
	}

	session.PID = handle.PID
	session.Status = domain.StatusRunning
	if err := s.repo.Update(ctx, session); err != nil {
		_ = s.processManager.Kill(handle.PID)
		return domain.Session{}, fmt.Errorf("failed to update session: %w", err)
	}

	go s.watchProcess(session.ID, handle)

	return session, nil
}

func (s *SessionService) RestartSession(ctx context.Context, id string) (domain.Session, error) {
	session, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}

	if session.Status == domain.StatusRunning && s.processManager.IsAlive(session.PID) {
		_ = s.processManager.Kill(session.PID)
		session.Status = domain.StatusStopped
		if err := s.repo.Update(ctx, session); err != nil {
			return domain.Session{}, fmt.Errorf("failed to update session: %w", err)
		}
	}

	return s.ResumeSession(ctx, id)
}

func (s *SessionService) GetSession(ctx context.Context, id string) (domain.Session, error) {
	return s.repo.Get(ctx, id)
}

func (s *SessionService) ListSessions(ctx context.Context) ([]domain.Session, error) {
	sessions, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range sessions {
		if sessions[i].Status == domain.StatusRunning && !s.processManager.IsAlive(sessions[i].PID) {
			sessions[i].Status = domain.StatusStopped
			_ = s.repo.Update(ctx, sessions[i])
		}
	}

	return sessions, nil
}

func (s *SessionService) DeleteSession(ctx context.Context, id string) error {
	session, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if session.Status == domain.StatusRunning {
		_ = s.processManager.Kill(session.PID)
	}

	// The ON DELETE SET NULL FK automatically clears session_id in worktrees
	// when the session row is deleted.
	return s.repo.Delete(ctx, id)
}

func (s *SessionService) GetPTYHandle(sessionID string) (*ports.PTYHandle, error) {
	session, err := s.repo.Get(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != domain.StatusRunning {
		return nil, fmt.Errorf("session %s is not running", sessionID)
	}

	handle, ok := s.processManager.GetHandle(session.PID)
	if !ok {
		return nil, fmt.Errorf("no PTY handle for session %s", sessionID)
	}
	return handle, nil
}

func (s *SessionService) ResizePTY(pid int, rows, cols uint16) error {
	return s.processManager.Resize(pid, rows, cols)
}

// ListWorktrees returns all managed worktrees with their associated session info.
// It also auto-adopts any worktree_managed sessions not yet in the worktrees table.
func (s *SessionService) ListWorktrees(ctx context.Context) ([]domain.WorktreeEntry, error) {
	if s.worktreeRepo == nil {
		return nil, nil
	}

	allSessions, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, sess := range allSessions {
		if sess.WorktreeManaged {
			s.trackWorktreeSession(ctx, sess)
		}
	}

	wts, err := s.worktreeRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]domain.WorktreeEntry, 0, len(wts))
	for _, wt := range wts {
		entry := domain.WorktreeEntry{ManagedWorktree: wt}
		if wt.SessionID != nil {
			sess, err := s.repo.Get(ctx, *wt.SessionID)
			if err == nil {
				name := sess.Name
				status := string(sess.Status)
				entry.SessionName = &name
				entry.SessionStatus = &status
			} else {
				// Session was deleted without cleanup; clear the FK reference
				wt.SessionID = nil
				entry.ManagedWorktree = wt
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ErrWorktreeHasSession is returned when trying to delete a worktree that
// still has a linked session. Delete the session first to unlink it.
type ErrWorktreeHasSession struct {
	WorktreeID string
	SessionID  string
}

func (e *ErrWorktreeHasSession) Error() string {
	return fmt.Sprintf("cannot delete worktree %q: session %q is still linked — delete the session first", e.WorktreeID, e.SessionID)
}

// DeleteWorktree removes a worktree. Returns ErrWorktreeHasSession if a
// session is still linked to it.
func (s *SessionService) DeleteWorktree(ctx context.Context, id string) error {
	if s.worktreeRepo == nil {
		return fmt.Errorf("worktree repository not configured")
	}

	wt, err := s.worktreeRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if wt.SessionID != nil {
		return &ErrWorktreeHasSession{WorktreeID: id, SessionID: *wt.SessionID}
	}

	if s.gitService != nil {
		_ = s.gitService.RemoveWorktree(wt.RepoRoot, wt.Path, false)
	}

	return s.worktreeRepo.Delete(ctx, id)
}

func (s *SessionService) watchProcess(sessionID string, handle *ports.PTYHandle) {
	<-handle.Done
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	session, err := s.repo.Get(ctx, sessionID)
	if err != nil {
		return
	}

	// Only update if the session still belongs to this process.
	// After a restart, a new process owns the session and this
	// stale watcher must not overwrite the new running status.
	if session.PID != handle.PID {
		return
	}

	session.Status = domain.StatusStopped
	_ = s.repo.Update(ctx, session)
}

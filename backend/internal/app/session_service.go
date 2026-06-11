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

// WorktreeAction controls what happens to the worktree when a session is deleted.
type WorktreeAction string

const (
	WorktreeActionKeep  WorktreeAction = "keep"
	WorktreeActionRemove WorktreeAction = "remove"
	WorktreeActionForce  WorktreeAction = "force"
)

// ErrWorktreeDirty is returned when trying to remove a dirty worktree without force.
type ErrWorktreeDirty struct {
	Path string
}

func (e *ErrWorktreeDirty) Error() string {
	return fmt.Sprintf("worktree %q has uncommitted changes", e.Path)
}

type SessionService struct {
	repo           ports.SessionRepository
	processManager ports.ProcessManager
	templateRepo   ports.TemplateRepository
	gitService     ports.GitService
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

	go s.watchProcess(session.ID, handle)

	return session, nil
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

func (s *SessionService) DeleteSession(ctx context.Context, id string, action WorktreeAction) error {
	session, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if session.Status == domain.StatusRunning {
		_ = s.processManager.Kill(session.PID)
	}

	if session.WorktreeManaged && s.gitService != nil {
		switch action {
		case WorktreeActionRemove:
			clean, cleanErr := s.gitService.IsClean(session.WorkingDir)
			if cleanErr == nil && !clean {
				return &ErrWorktreeDirty{Path: session.WorkingDir}
			}
			if err := s.gitService.RemoveWorktree(session.RepoRoot, session.WorkingDir, false); err != nil {
				return fmt.Errorf("remove worktree: %w", err)
			}
		case WorktreeActionForce:
			if err := s.gitService.RemoveWorktree(session.RepoRoot, session.WorkingDir, true); err != nil {
				return fmt.Errorf("force remove worktree: %w", err)
			}
		}
	}

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

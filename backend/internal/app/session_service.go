package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/harness"
	"github.com/Corwind/cmux/backend/internal/ports"
	"github.com/google/uuid"
)

// ErrSessionProvisioning is returned when an operation requires a running session
// but the session is still being provisioned.
var ErrSessionProvisioning = errors.New("session is still provisioning")

// SessionEventBroadcaster is implemented by the WebSocket hub (Track D) to push
// live status updates to connected clients without creating a compile-time
// dependency on the hub package here.
type SessionEventBroadcaster interface {
	BroadcastSessionStatus(sessionID, sessionName, status, errMsg string)
	BroadcastWorktreeDeleted(worktreeID, errMsg string)
}

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
	HarnessType     string
	Worktree        *WorktreeSpec
}

// defaultDiscoverRetryAttempts and defaultDiscoverRetryInterval add up to a
// 30s budget: observed against a real Codex CLI, it takes roughly 8-13s
// after process start before it writes its rollout file (some startup
// handshake happens first), so a couple of seconds isn't enough.
const (
	defaultDiscoverRetryAttempts = 60
	defaultDiscoverRetryInterval = 500 * time.Millisecond
)

type SessionService struct {
	repo                  ports.SessionRepository
	processManager        ports.ProcessManager
	templateRepo          ports.TemplateRepository
	gitService            ports.GitService
	worktreeRepo          ports.WorktreeRepository
	worktreesDir          string
	harnessRegistry       *harness.Registry
	mu                    sync.RWMutex
	provisionCtxs         sync.Map // key: sessionID string, value: context.CancelFunc
	deletionCtxs          sync.Map // key: worktreeID string, value: context.CancelFunc
	broadcaster           SessionEventBroadcaster
	discoverRetryAttempts int
	discoverRetryInterval time.Duration
}

func NewSessionService(repo ports.SessionRepository, pm ports.ProcessManager, templateRepo ports.TemplateRepository, opts ...SessionServiceOption) *SessionService {
	s := &SessionService{
		repo:                  repo,
		processManager:        pm,
		templateRepo:          templateRepo,
		discoverRetryAttempts: defaultDiscoverRetryAttempts,
		discoverRetryInterval: defaultDiscoverRetryInterval,
	}
	for _, opt := range opts {
		opt(s)
	}
	// Callers (mainly tests) that don't inject a registry via WithHarnessRegistry
	// get the same Claude behavior as before this package existed, with no model
	// set. NotificationsEnabled: true matches config.defaults()'s root+per-harness
	// default of "on" — this fallback bypasses that config loading path entirely.
	if s.harnessRegistry == nil {
		s.harnessRegistry = harness.NewRegistry()
		s.harnessRegistry.Register(harness.NewClaudeHarness(domain.ClaudeConfig{NotificationsEnabled: true}), "Claude Code")
	}
	return s
}

// resolveHarness returns the harness registered for the given type name,
// falling back to the registry's Default() when the type is empty or has no
// registered implementation. A non-empty, unregistered type is logged as a
// warning before falling back.
func (s *SessionService) resolveHarness(harnessType string) harness.Harness {
	if harnessType != "" {
		if h, ok := s.harnessRegistry.Get(harness.Type(harnessType)); ok {
			return h
		}
		slog.Warn("requested harness type not registered, falling back to default", "harness_type", harnessType)
	}
	return s.harnessRegistry.Default()
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

// WithBroadcaster injects the WebSocket hub (Track D) so that async provisioning
// can push status events to connected clients.
func WithBroadcaster(b SessionEventBroadcaster) SessionServiceOption {
	return func(s *SessionService) {
		s.broadcaster = b
	}
}

// WithDiscoverRetryConfig overrides how long discoverAndPersistSessionID
// polls for a harness-minted session ID after spawn. Tests use this to
// shrink the real-world 30s budget so a fake harness that never succeeds
// doesn't leave a slow-retrying goroutine running past the test.
func WithDiscoverRetryConfig(attempts int, interval time.Duration) SessionServiceOption {
	return func(s *SessionService) {
		s.discoverRetryAttempts = attempts
		s.discoverRetryInterval = interval
	}
}

// WithHarnessRegistry injects the set of coding-agent harness strategies
// available for session creation/resumption, keyed by harness type.
func WithHarnessRegistry(r *harness.Registry) SessionServiceOption {
	return func(s *SessionService) {
		s.harnessRegistry = r
	}
}

func (s *SessionService) CreateSession(ctx context.Context, input CreateSessionInput) (domain.Session, error) {
	workingDir := input.WorkingDir

	if input.Worktree != nil && s.gitService != nil {
		return s.createSessionAsync(ctx, input)
	}

	h := s.resolveHarness(input.HarnessType)

	// --- Synchronous path (no worktree) ---
	session, err := domain.NewSession(input.Name, workingDir, string(h.Type()))
	if err != nil {
		slog.Error("invalid session parameters", "name", input.Name, "working_dir", workingDir, "err", err)
		return domain.Session{}, fmt.Errorf("invalid session: %w", err)
	}

	// Resolve sandbox template content
	s.applySandboxContent(ctx, input.TemplateID)

	args := h.BuildSpawnArgs(harness.SpawnIntent{
		SessionID:       session.HarnessSessionID,
		SkipPermissions: input.SkipPermissions,
	})
	slog.Info("spawning session process", "session_id", session.ID, "working_dir", workingDir)
	spawnStartedAt := time.Now()
	handle, err := s.processManager.Spawn(ctx, workingDir, string(h.Type()), args...)
	if err != nil {
		slog.Error("failed to spawn session process", "session_id", session.ID, "working_dir", workingDir, "err", err)
		return domain.Session{}, fmt.Errorf("failed to spawn process: %w", err)
	}

	session.PID = handle.PID
	session.Status = domain.StatusRunning
	session.TemplateID = input.TemplateID
	session.SkipPermissions = input.SkipPermissions

	if err := s.repo.Create(ctx, session); err != nil {
		slog.Error("failed to persist session", "session_id", session.ID, "pid", handle.PID, "err", err)
		if killErr := s.processManager.Kill(handle.PID); killErr != nil {
			slog.Error("failed to kill process after session persist failure", "pid", handle.PID, "err", killErr)
		}
		return domain.Session{}, fmt.Errorf("failed to store session: %w", err)
	}

	slog.Info("session created", "session_id", session.ID, "name", session.Name, "pid", session.PID, "working_dir", workingDir)

	if h.HasExternalSessionIDMinting() {
		go s.discoverAndPersistSessionID(h, session.ID, workingDir, spawnStartedAt)
	}

	go s.watchProcess(session.ID, handle)

	return session, nil
}

// createSessionAsync handles the worktree path: it validates the repo, creates
// a provisioning session in the DB, then launches async provisioning.
func (s *SessionService) createSessionAsync(ctx context.Context, input CreateSessionInput) (domain.Session, error) {
	spec := input.Worktree

	// Validate repo
	info, err := s.gitService.Info(input.WorkingDir)
	if err != nil {
		slog.Error("failed to get git info for session creation", "working_dir", input.WorkingDir, "err", err)
		return domain.Session{}, fmt.Errorf("git info: %w", err)
	}
	if !info.IsRepo {
		slog.Error("path is not a git repository", "working_dir", input.WorkingDir)
		return domain.Session{}, fmt.Errorf("%q is not a git repository", input.WorkingDir)
	}

	// Compute worktree path
	wtPath := spec.Path
	if wtPath == "" {
		safeBranch := strings.ReplaceAll(spec.Branch, "/", "-")
		repoBase := filepath.Base(info.RepoRoot)
		wtPath = filepath.Join(s.worktreesDir, repoBase, safeBranch)
	}

	// Create session with StatusProvisioning
	h := s.resolveHarness(input.HarnessType)
	session, err := domain.NewSession(input.Name, wtPath, string(h.Type()))
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid session: %w", err)
	}
	session.Status = domain.StatusProvisioning
	session.RepoRoot = info.RepoRoot
	session.GitBranch = spec.Branch
	session.WorktreeManaged = true
	session.TemplateID = input.TemplateID
	session.SkipPermissions = input.SkipPermissions

	if err := s.repo.Create(ctx, session); err != nil {
		slog.Error("failed to persist provisioning session", "session_id", session.ID, "err", err)
		return domain.Session{}, fmt.Errorf("failed to store session: %w", err)
	}

	// Create and register the cancellation context BEFORE launching the goroutine
	// so that a concurrent DeleteSession cannot miss the cancel function.
	provCtx, cancel := context.WithCancel(context.Background())
	s.provisionCtxs.Store(session.ID, cancel)

	slog.Info("session queued for provisioning", "session_id", session.ID, "worktree_path", wtPath, "branch", spec.Branch)

	go s.provisionWorktree(session, input, *spec, info.RepoRoot, wtPath, provCtx, cancel, h)

	return session, nil
}

// provisionWorktree is the async goroutine that creates the git worktree, spawns
// the process, and transitions the session from StatusProvisioning to either
// StatusRunning or StatusFailed.
// provCtx and cancel are created by createSessionAsync before the goroutine is
// launched so that a concurrent DeleteSession never misses the cancel function.
func (s *SessionService) provisionWorktree(session domain.Session, input CreateSessionInput, spec WorktreeSpec, repoRoot, wtPath string, provCtx context.Context, cancel context.CancelFunc, h harness.Harness) {
	defer func() {
		cancel()
		s.provisionCtxs.Delete(session.ID)
	}()

	wt, err := s.gitService.AddWorktree(provCtx, repoRoot, wtPath, spec.Branch, spec.BaseRef, spec.CreateBranch)
	if err != nil {
		slog.Error("failed to create git worktree", "session_id", session.ID, "path", wtPath, "branch", spec.Branch, "err", err)
		session.Status = domain.StatusFailed
		session.Error = fmt.Sprintf("create worktree: %s", err.Error())
		_ = s.repo.Update(context.Background(), session)
		if s.broadcaster != nil {
			s.broadcaster.BroadcastSessionStatus(session.ID, session.Name, string(domain.StatusFailed), session.Error)
		}
		return
	}
	slog.Info("git worktree created", "session_id", session.ID, "path", wt.Path, "branch", wt.Branch)

	workingDir := wt.Path

	// Resolve sandbox template
	s.applySandboxContent(provCtx, input.TemplateID)

	args := h.BuildSpawnArgs(harness.SpawnIntent{
		SessionID:       session.HarnessSessionID,
		SkipPermissions: input.SkipPermissions,
	})

	spawnStartedAt := time.Now()
	handle, err := s.processManager.Spawn(context.Background(), workingDir, string(h.Type()), args...)
	if err != nil {
		slog.Error("failed to spawn process for provisioning session", "session_id", session.ID, "err", err)
		// Best-effort cleanup of the orphaned worktree
		if cleanErr := s.gitService.RemoveWorktree(context.Background(), repoRoot, wtPath, true); cleanErr != nil {
			slog.Error("failed to clean up orphaned worktree after spawn failure", "path", wtPath, "err", cleanErr)
		}
		session.Status = domain.StatusFailed
		session.Error = fmt.Sprintf("spawn process: %s", err.Error())
		_ = s.repo.Update(context.Background(), session)
		if s.broadcaster != nil {
			s.broadcaster.BroadcastSessionStatus(session.ID, session.Name, string(domain.StatusFailed), session.Error)
		}
		return
	}

	session.PID = handle.PID
	session.Status = domain.StatusRunning
	session.Error = ""

	if err := s.repo.Update(context.Background(), session); err != nil {
		slog.Error("failed to update session after provision", "session_id", session.ID, "err", err)
	}

	if s.worktreeRepo != nil {
		s.trackWorktreeSession(context.Background(), session)
	}

	if s.broadcaster != nil {
		s.broadcaster.BroadcastSessionStatus(session.ID, session.Name, string(domain.StatusRunning), "")
	}

	if h.HasExternalSessionIDMinting() {
		go s.discoverAndPersistSessionID(h, session.ID, workingDir, spawnStartedAt)
	}

	go s.watchProcess(session.ID, handle)

	slog.Info("session provisioned and running", "session_id", session.ID, "pid", session.PID, "working_dir", workingDir)
}

// discoverSessionIDWithRetry calls h.DiscoverSessionID repeatedly, tolerating
// the delay between a harness process spawning and it writing out the
// on-disk state DiscoverSessionID reads from. Callers must run this off any
// blocking/request path (see discoverAndPersistSessionID) rather than await
// it inline, since attempts/interval are sized for real-world harness
// startup delays (see defaultDiscoverRetryAttempts/Interval), not an
// instantaneous writer.
//
// workingDir is resolved through symlinks to match pty.Manager.Spawn, which
// sets the harness process's actual cwd (and thus what a harness like Codex
// records in its own on-disk session state) to the symlink-resolved path
// (e.g. /var/folders -> /private/var/folders on macOS) rather than the raw
// path cmux was given.
func discoverSessionIDWithRetry(h harness.Harness, workingDir string, notBefore time.Time, attempts int, interval time.Duration) (string, error) {
	resolvedDir, err := filepath.EvalSymlinks(workingDir)
	if err != nil {
		resolvedDir = workingDir
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		id, err := h.DiscoverSessionID(resolvedDir, notBefore)
		if err == nil {
			return id, nil
		}
		lastErr = err
		time.Sleep(interval)
	}
	return "", lastErr
}

// discoverAndPersistSessionID runs in the background, detached from the
// goroutine that spawned the session, so discoverSessionIDWithRetry's long
// poll window never blocks session creation or delays the session appearing
// as running. Once the harness-minted ID is found, it reloads the current
// persisted session (rather than reusing a possibly-stale in-memory copy)
// and overwrites just its HarnessSessionID, so a later Resume uses the ID
// the harness will actually recognize instead of cmux's own placeholder.
func (s *SessionService) discoverAndPersistSessionID(h harness.Harness, sessionID, workingDir string, spawnStartedAt time.Time) {
	discovered, err := discoverSessionIDWithRetry(h, workingDir, spawnStartedAt, s.discoverRetryAttempts, s.discoverRetryInterval)
	if err != nil {
		slog.Warn("failed to discover harness-minted session id", "session_id", sessionID, "err", err)
		return
	}

	session, err := s.repo.Get(context.Background(), sessionID)
	if err != nil {
		slog.Warn("failed to reload session before persisting discovered harness session id", "session_id", sessionID, "err", err)
		return
	}
	session.HarnessSessionID = discovered
	if err := s.repo.Update(context.Background(), session); err != nil {
		slog.Warn("failed to persist discovered harness session id", "session_id", sessionID, "err", err)
		return
	}
	slog.Info("persisted harness-minted session id", "session_id", sessionID, "harness_session_id", discovered)
}

// trackWorktreeSession creates or adopts a ManagedWorktree record for the given
// session and sets the session_id FK on the worktree record.
func (s *SessionService) trackWorktreeSession(ctx context.Context, session domain.Session) {
	wt, err := s.worktreeRepo.GetByPath(ctx, session.WorkingDir)
	if err != nil {
		slog.Info("no existing worktree record found, creating one", "path", session.WorkingDir)
		wt = domain.ManagedWorktree{
			ID:        uuid.New().String(),
			Path:      session.WorkingDir,
			Branch:    session.GitBranch,
			RepoRoot:  session.RepoRoot,
			CreatedAt: session.CreatedAt,
		}
		if createErr := s.worktreeRepo.Create(ctx, wt); createErr != nil {
			slog.Error("failed to create worktree record", "path", session.WorkingDir, "err", createErr)
			return
		}
		slog.Info("worktree record created", "worktree_id", wt.ID, "path", wt.Path, "branch", wt.Branch)
	}
	if setErr := s.worktreeRepo.SetSession(ctx, wt.ID, &session.ID); setErr != nil {
		slog.Error("failed to set session on worktree", "session_id", session.ID, "worktree_id", wt.ID, "err", setErr)
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
			slog.Error("failed to resolve template", "template_id", templateID, "err", err)
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

	h := s.resolveHarness(session.HarnessType)
	args := h.BuildSpawnArgs(harness.SpawnIntent{
		SessionID:       session.HarnessSessionID,
		Resume:          h.HasResumeSupport(),
		SkipPermissions: session.SkipPermissions,
	})
	handle, err := s.processManager.Spawn(ctx, session.WorkingDir, string(h.Type()), args...)
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

	// Cancel any in-flight provisioning for this session
	if cancelFn, ok := s.provisionCtxs.Load(id); ok {
		cancelFn.(context.CancelFunc)()
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
	if session.Status == domain.StatusProvisioning {
		return nil, ErrSessionProvisioning
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

// worktreeDeletionTimeout bounds how long the async worktree-removal goroutine
// waits on git before giving up and deleting the DB record anyway. It is a var
// (not a const) so tests can shrink it to exercise the hang-protection path.
var worktreeDeletionTimeout = 5 * time.Minute

// ErrWorktreeHasSession is returned when trying to delete a worktree that
// still has a linked session. Delete the session first to unlink it.
type ErrWorktreeHasSession struct {
	WorktreeID string
	SessionID  string
}

func (e *ErrWorktreeHasSession) Error() string {
	return fmt.Sprintf("cannot delete worktree %q: session %q is still linked — delete the session first", e.WorktreeID, e.SessionID)
}

// DeleteWorktree removes a worktree asynchronously. It marks the worktree as
// deleting, returns immediately, and runs the git removal + DB cleanup in a
// background goroutine, broadcasting completion when done. Returns
// ErrWorktreeHasSession if a session is still linked to it. Calls that arrive
// while a deletion is already in flight are idempotent no-ops.
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

	// Already being deleted according to the DB — idempotent. This also covers
	// a stale 'deleting' row left behind after a process restart, where no
	// in-process goroutine exists to gate on.
	if wt.Status == domain.WorktreeStatusDeleting {
		return nil
	}

	// Atomic in-flight gate: LoadOrStore is the source of truth for "a deletion
	// for this id is already running in this process". This closes the TOCTOU
	// window where two concurrent calls on a 'ready' worktree both pass the DB
	// status check above and each launch a goroutine (double git-remove +
	// spurious error broadcast). The bounded context also protects against a
	// hung git removal — on timeout the goroutine still deletes the DB record.
	delCtx, cancel := context.WithTimeout(context.Background(), worktreeDeletionTimeout)
	if _, inFlight := s.deletionCtxs.LoadOrStore(id, cancel); inFlight {
		cancel() // already running — idempotent no-op
		return nil
	}

	if err := s.worktreeRepo.SetStatus(ctx, id, domain.WorktreeStatusDeleting); err != nil {
		s.deletionCtxs.Delete(id)
		cancel()
		return fmt.Errorf("failed to mark worktree as deleting: %w", err)
	}

	go s.removeWorktree(wt, delCtx, cancel)

	return nil
}

// removeWorktree is the async goroutine that calls git worktree remove, then
// deletes the DB record regardless of the git outcome. Git removal is
// best-effort: a git error is logged and surfaced via the broadcast, but never
// blocks removal of the DB record.
func (s *SessionService) removeWorktree(wt domain.ManagedWorktree, delCtx context.Context, cancel context.CancelFunc) {
	defer func() {
		cancel()
		s.deletionCtxs.Delete(wt.ID)
	}()

	var gitErr string
	if s.gitService != nil {
		if err := s.gitService.RemoveWorktree(delCtx, wt.RepoRoot, wt.Path, false); err != nil {
			slog.Error("failed to remove git worktree", "worktree_id", wt.ID, "path", wt.Path, "err", err)
			gitErr = err.Error()
		}
	}

	if err := s.worktreeRepo.Delete(context.Background(), wt.ID); err != nil {
		slog.Error("failed to delete worktree record", "worktree_id", wt.ID, "err", err)
	} else {
		slog.Info("worktree deleted", "worktree_id", wt.ID, "path", wt.Path)
	}

	if s.broadcaster != nil {
		s.broadcaster.BroadcastWorktreeDeleted(wt.ID, gitErr)
	}
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

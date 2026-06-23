package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/ports"
)

// --- Mock SessionRepository ---
// mockRepo is goroutine-safe: all map accesses are protected by mu so that
// concurrent reads from tests and writes from the provision goroutine cannot
// trigger the race detector.

type mockRepo struct {
	mu        sync.RWMutex
	sessions  map[string]domain.Session
	createFn  func(ctx context.Context, s domain.Session) error
	updateErr error // if set, Update returns this error
}

func newMockRepo() *mockRepo {
	return &mockRepo{sessions: make(map[string]domain.Session)}
}

func (m *mockRepo) Create(ctx context.Context, s domain.Session) error {
	if m.createFn != nil {
		return m.createFn(ctx, s)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *mockRepo) Get(ctx context.Context, id string) (domain.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return domain.Session{}, fmt.Errorf("session not found: %s", id)
	}
	return s, nil
}

func (m *mockRepo) List(ctx context.Context) ([]domain.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []domain.Session
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockRepo) Update(ctx context.Context, s domain.Session) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

// --- Mock ProcessManager ---
// mockProcessManager is goroutine-safe: all fields are protected by mu so that
// concurrent access from the provision goroutine and test code cannot trigger
// the race detector.

type mockProcessManager struct {
	mu        sync.Mutex
	alive     map[int]bool
	handles   map[int]*ports.PTYHandle
	doneChans map[int]chan error
	spawnErr  error
	killPIDs  []int
	spawnArgs []string
	nextPID   int
}

func newMockProcessManager() *mockProcessManager {
	return &mockProcessManager{
		alive:     make(map[int]bool),
		handles:   make(map[int]*ports.PTYHandle),
		doneChans: make(map[int]chan error),
		nextPID:   42,
	}
}

func (m *mockProcessManager) Spawn(ctx context.Context, workingDir string, args ...string) (*ports.PTYHandle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spawnArgs = args
	if m.spawnErr != nil {
		return nil, m.spawnErr
	}
	pid := m.nextPID
	m.nextPID++
	done := make(chan error, 1)
	h := &ports.PTYHandle{
		PTY:  os.Stdin, // placeholder
		PID:  pid,
		Done: done,
	}
	m.alive[pid] = true
	m.handles[pid] = h
	m.doneChans[pid] = done
	return h, nil
}

func (m *mockProcessManager) Resize(pid int, rows, cols uint16) error {
	return nil
}

func (m *mockProcessManager) Kill(pid int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killPIDs = append(m.killPIDs, pid)
	delete(m.alive, pid)
	return nil
}

func (m *mockProcessManager) IsAlive(pid int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.alive[pid]
}

func (m *mockProcessManager) KillAll() {}

func (m *mockProcessManager) GetHandle(pid int) (*ports.PTYHandle, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.handles[pid]
	return h, ok
}

// killPIDsSafe returns a copy of killPIDs safely.
func (m *mockProcessManager) killPIDsSafe() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]int, len(m.killPIDs))
	copy(result, m.killPIDs)
	return result
}

// nextPIDSafe returns nextPID safely.
func (m *mockProcessManager) nextPIDSafe() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nextPID
}

// getDoneChan returns the done channel for the given PID safely.
func (m *mockProcessManager) getDoneChan(pid int) chan error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.doneChans[pid]
}

// --- Mock ProcessManager with SandboxContentProvider ---

type mockSandboxProcessManager struct {
	mockProcessManager
	sandboxContents []string
}

func newMockSandboxProcessManager() *mockSandboxProcessManager {
	return &mockSandboxProcessManager{
		mockProcessManager: *newMockProcessManager(),
	}
}

func (m *mockSandboxProcessManager) SetSandboxContent(contents []string) {
	m.sandboxContents = contents
}

// --- Mock GitService ---
// mockGitService is goroutine-safe: removedWorktrees/forceFlags are protected
// by mu since they are appended from the provision goroutine.

type mockGitService struct {
	mu               sync.Mutex
	infoFn           func(path string) (ports.GitInfo, error)
	addWorktreeFn    func(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error)
	removeWorktreeFn func(ctx context.Context, repoRoot, wtPath string, force bool) error
	isCleanFn        func(ctx context.Context, path string) (bool, error)
	removedWorktrees []string
	forceFlags       []bool
}

func newMockGitService() *mockGitService {
	return &mockGitService{}
}

func (m *mockGitService) Info(path string) (ports.GitInfo, error) {
	if m.infoFn != nil {
		return m.infoFn(path)
	}
	return ports.GitInfo{
		IsRepo:        true,
		RepoRoot:      path,
		CurrentBranch: "main",
	}, nil
}

func (m *mockGitService) AddWorktree(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
	if m.addWorktreeFn != nil {
		return m.addWorktreeFn(ctx, repoRoot, wtPath, branch, baseRef, create)
	}
	return ports.Worktree{Path: wtPath, Branch: branch}, nil
}

func (m *mockGitService) RemoveWorktree(ctx context.Context, repoRoot, wtPath string, force bool) error {
	m.mu.Lock()
	m.removedWorktrees = append(m.removedWorktrees, wtPath)
	m.forceFlags = append(m.forceFlags, force)
	fn := m.removeWorktreeFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, repoRoot, wtPath, force)
	}
	return nil
}

func (m *mockGitService) IsClean(ctx context.Context, path string) (bool, error) {
	if m.isCleanFn != nil {
		return m.isCleanFn(ctx, path)
	}
	return true, nil
}

// removedWorktreesSafe returns a safe copy of removedWorktrees.
func (m *mockGitService) removedWorktreesSafe() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.removedWorktrees))
	copy(result, m.removedWorktrees)
	return result
}

// forceFlagsSafe returns a safe copy of forceFlags.
func (m *mockGitService) forceFlagsSafe() []bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]bool, len(m.forceFlags))
	copy(result, m.forceFlags)
	return result
}

// --- Helper ---

func createInput(name, workingDir string) CreateSessionInput {
	return CreateSessionInput{Name: name, WorkingDir: workingDir}
}

// --- Tests ---

func TestCreateSession_Success(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	s, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "test" {
		t.Errorf("expected name 'test', got %q", s.Name)
	}
	if s.Status != domain.StatusRunning {
		t.Errorf("expected status running, got %q", s.Status)
	}
	if s.PID != 42 {
		t.Errorf("expected PID 42, got %d", s.PID)
	}
	// Verify stored in repo
	if _, err := repo.Get(context.Background(), s.ID); err != nil {
		t.Errorf("session not found in repo: %v", err)
	}
}

func TestCreateSession_EmptyNameDefaultsToDir(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{WorkingDir: "/home/user/my-project"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if session.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", session.Name)
	}
}

func TestCreateSession_EmptyWorkingDir(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	_, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test"})
	if err == nil {
		t.Fatal("expected error for empty working dir")
	}
}

func TestCreateSession_SpawnFailure(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	pm.spawnErr = fmt.Errorf("spawn failed")
	svc := NewSessionService(repo, pm, nil)

	_, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp"})
	if err == nil {
		t.Fatal("expected error when spawn fails")
	}
}

func TestCreateSession_RepoFailureKillsProcess(t *testing.T) {
	repo := newMockRepo()
	repo.createFn = func(ctx context.Context, s domain.Session) error {
		return fmt.Errorf("db error")
	}
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	_, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp"})
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
	if len(pm.killPIDs) == 0 {
		t.Error("expected process to be killed after repo failure")
	}
}

func TestCreateSession_SkipPermissions(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	_, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp", SkipPermissions: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, arg := range pm.spawnArgs {
		if arg == "--dangerously-skip-permissions" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --dangerously-skip-permissions in spawn args, got %v", pm.spawnArgs)
	}
}

func TestCreateSession_NoSkipPermissions(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	_, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, arg := range pm.spawnArgs {
		if arg == "--dangerously-skip-permissions" {
			t.Errorf("did not expect --dangerously-skip-permissions in spawn args, got %v", pm.spawnArgs)
		}
	}
}

func TestGetSession(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	created, _ := svc.CreateSession(context.Background(), createInput("test", "/tmp"))
	got, err := svc.GetSession(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, got.ID)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	_, err := svc.GetSession(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestListSessions_UpdatesDeadProcesses(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	s, _ := svc.CreateSession(context.Background(), createInput("test", "/tmp"))
	// Simulate process death
	delete(pm.alive, s.PID)

	sessions, err := svc.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Status != domain.StatusStopped {
		t.Errorf("expected status stopped for dead process, got %q", sessions[0].Status)
	}
}

func TestDeleteSession_KillsRunningProcess(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	s, _ := svc.CreateSession(context.Background(), createInput("test", "/tmp"))
	if err := svc.DeleteSession(context.Background(), s.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify killed
	found := false
	for _, pid := range pm.killPIDs {
		if pid == s.PID {
			found = true
		}
	}
	if !found {
		t.Error("expected running process to be killed on delete")
	}
	// Verify removed from repo
	_, err := repo.Get(context.Background(), s.ID)
	if err == nil {
		t.Error("expected session to be deleted from repo")
	}
}

func TestDeleteSession_NotFound(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	err := svc.DeleteSession(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestGetPTYHandle_Success(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	s, _ := svc.CreateSession(context.Background(), createInput("test", "/tmp"))
	h, err := svc.GetPTYHandle(s.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.PID != s.PID {
		t.Errorf("expected PID %d, got %d", s.PID, h.PID)
	}
}

func TestGetPTYHandle_NotRunning(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	s, _ := svc.CreateSession(context.Background(), createInput("test", "/tmp"))
	// Mark as stopped in repo
	s.Status = domain.StatusStopped
	if err := repo.Update(context.Background(), s); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	_, err := svc.GetPTYHandle(s.ID)
	if err == nil {
		t.Fatal("expected error for stopped session")
	}
}

func TestResizePTY(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	err := svc.ResizePTY(42, 24, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSession_StoresTemplateID(t *testing.T) {
	repo := newMockRepo()
	pm := newMockSandboxProcessManager()
	tmplRepo := newMockTemplateRepo()
	tmpl := domain.SandboxTemplate{ID: "tmpl-1", Name: "test", Content: "(allow network-outbound)"}
	tmplRepo.templates["tmpl-1"] = tmpl

	svc := NewSessionService(repo, pm, tmplRepo)
	session, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp", TemplateID: "tmpl-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.TemplateID != "tmpl-1" {
		t.Errorf("expected TemplateID 'tmpl-1', got %q", session.TemplateID)
	}

	stored, err := repo.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.TemplateID != "tmpl-1" {
		t.Errorf("expected stored TemplateID 'tmpl-1', got %q", stored.TemplateID)
	}
}

func TestResumeSession_AppliesSandboxTemplate(t *testing.T) {
	repo := newMockRepo()
	pm := newMockSandboxProcessManager()
	tmplRepo := newMockTemplateRepo()
	tmpl := domain.SandboxTemplate{ID: "tmpl-1", Name: "test", Content: "(allow network-outbound)"}
	tmplRepo.templates["tmpl-1"] = tmpl

	svc := NewSessionService(repo, pm, tmplRepo)

	// Create session with template
	session, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp", TemplateID: "tmpl-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Simulate process death
	delete(pm.alive, session.PID)
	session.Status = domain.StatusStopped
	_ = repo.Update(context.Background(), session)

	// Clear sandbox contents to verify resume sets them again
	pm.sandboxContents = nil

	// Resume
	resumed, err := svc.ResumeSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resumed.Status != domain.StatusRunning {
		t.Errorf("expected status running, got %q", resumed.Status)
	}

	// Verify sandbox content was applied
	if len(pm.sandboxContents) == 0 {
		t.Fatal("expected sandbox content to be set on resume, but it was empty")
	}
	if pm.sandboxContents[0] != "(allow network-outbound)" {
		t.Errorf("expected sandbox content '(allow network-outbound)', got %q", pm.sandboxContents[0])
	}
}

func TestResumeSession_AlreadyRunning(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	session, err := svc.CreateSession(context.Background(), createInput("test", "/tmp"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resume while still running — should return immediately
	resumed, err := svc.ResumeSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resumed.ID != session.ID {
		t.Errorf("expected same session ID")
	}
}

func TestCreateSession_StoresSkipPermissions(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp", SkipPermissions: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !session.SkipPermissions {
		t.Error("expected SkipPermissions to be true")
	}

	stored, err := repo.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stored.SkipPermissions {
		t.Error("expected stored SkipPermissions to be true")
	}
}

func TestResumeSession_ReappliesSkipPermissions(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	// Create session with skip permissions
	session, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp", SkipPermissions: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Simulate process death
	delete(pm.alive, session.PID)
	session.Status = domain.StatusStopped
	_ = repo.Update(context.Background(), session)

	// Resume
	_, err = svc.ResumeSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify --dangerously-skip-permissions was passed
	found := false
	for _, arg := range pm.spawnArgs {
		if arg == "--dangerously-skip-permissions" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --dangerously-skip-permissions in resume spawn args, got %v", pm.spawnArgs)
	}
}

func TestResumeSession_NoSkipPermissionsWhenNotSet(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	// Create session without skip permissions
	session, err := svc.CreateSession(context.Background(), createInput("test", "/tmp"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Simulate process death
	delete(pm.alive, session.PID)
	session.Status = domain.StatusStopped
	_ = repo.Update(context.Background(), session)

	// Resume
	_, err = svc.ResumeSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify --dangerously-skip-permissions was NOT passed
	for _, arg := range pm.spawnArgs {
		if arg == "--dangerously-skip-permissions" {
			t.Errorf("did not expect --dangerously-skip-permissions in resume spawn args, got %v", pm.spawnArgs)
		}
	}
}

func TestResumeSession_NotFound(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	_, err := svc.ResumeSession(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestRestartSession_RunningSession(t *testing.T) {
	repo := newMockRepo()
	pm := newMockSandboxProcessManager()
	tmplRepo := newMockTemplateRepo()
	tmpl := domain.SandboxTemplate{ID: "tmpl-1", Name: "test", Content: "(allow network-outbound)"}
	tmplRepo.templates["tmpl-1"] = tmpl

	svc := NewSessionService(repo, pm, tmplRepo)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp", TemplateID: "tmpl-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	originalPID := session.PID

	// Restart while running — should kill and re-spawn
	restarted, err := svc.RestartSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restarted.Status != domain.StatusRunning {
		t.Errorf("expected status running, got %q", restarted.Status)
	}

	// Verify the original process was killed
	found := false
	for _, pid := range pm.killPIDs {
		if pid == originalPID {
			found = true
		}
	}
	if !found {
		t.Error("expected original process to be killed on restart")
	}

	// Verify sandbox content was reapplied
	if len(pm.sandboxContents) == 0 {
		t.Fatal("expected sandbox content to be set on restart")
	}
	if pm.sandboxContents[0] != "(allow network-outbound)" {
		t.Errorf("expected sandbox content '(allow network-outbound)', got %q", pm.sandboxContents[0])
	}
}

func TestRestartSession_StoppedSession(t *testing.T) {
	repo := newMockRepo()
	pm := newMockSandboxProcessManager()
	tmplRepo := newMockTemplateRepo()
	tmpl := domain.SandboxTemplate{ID: "tmpl-1", Name: "test", Content: "(allow network-outbound)"}
	tmplRepo.templates["tmpl-1"] = tmpl

	svc := NewSessionService(repo, pm, tmplRepo)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp", TemplateID: "tmpl-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Simulate process death
	delete(pm.alive, session.PID)
	session.Status = domain.StatusStopped
	_ = repo.Update(context.Background(), session)

	// Clear to verify restart sets them again
	pm.sandboxContents = nil

	// Restart a stopped session — should just resume
	restarted, err := svc.RestartSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restarted.Status != domain.StatusRunning {
		t.Errorf("expected status running, got %q", restarted.Status)
	}
}

func TestRestartSession_PicksUpUpdatedTemplate(t *testing.T) {
	repo := newMockRepo()
	pm := newMockSandboxProcessManager()
	tmplRepo := newMockTemplateRepo()
	tmpl := domain.SandboxTemplate{ID: "tmpl-1", Name: "test", Content: "(allow network-outbound)"}
	tmplRepo.templates["tmpl-1"] = tmpl

	svc := NewSessionService(repo, pm, tmplRepo)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{Name: "test", WorkingDir: "/tmp", TemplateID: "tmpl-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Update the template content in the repo (simulating user editing the template)
	updatedTmpl := domain.SandboxTemplate{ID: "tmpl-1", Name: "test", Content: "(allow file-read* (subpath \"/opt\"))"}
	tmplRepo.templates["tmpl-1"] = updatedTmpl

	// Restart — should pick up the updated template
	_, err = svc.RestartSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the NEW template content was applied
	if len(pm.sandboxContents) == 0 {
		t.Fatal("expected sandbox content to be set on restart")
	}
	if pm.sandboxContents[0] != "(allow file-read* (subpath \"/opt\"))" {
		t.Errorf("expected updated sandbox content, got %q", pm.sandboxContents[0])
	}
}

func TestRestartSession_NotFound(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	_, err := svc.RestartSession(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestWatchProcess_IgnoresStaleWatcher(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	// Create a session — spawns with PID 42, watcher on handle.Done
	session, err := svc.CreateSession(context.Background(), createInput("test", "/tmp"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oldPID := session.PID
	oldDone := pm.getDoneChan(oldPID)

	// Restart — kills old PID, spawns new PID (43)
	restarted, err := svc.RestartSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restarted.PID == oldPID {
		t.Fatalf("expected new PID, got same as old: %d", restarted.PID)
	}

	// Simulate old process finally exiting (old watcher fires)
	oldDone <- nil

	// Give the watcher goroutine time to run
	for i := 0; i < 100; i++ {
		s, _ := svc.GetSession(context.Background(), session.ID)
		if s.Status != domain.StatusRunning {
			t.Fatalf("stale watcher incorrectly set session to %q after restart", s.Status)
		}
	}

	// Session should still be running with the new PID
	final, _ := svc.GetSession(context.Background(), session.ID)
	if final.Status != domain.StatusRunning {
		t.Errorf("expected status running, got %q", final.Status)
	}
	if final.PID != restarted.PID {
		t.Errorf("expected PID %d, got %d", restarted.PID, final.PID)
	}
}

// --- Worktree tests ---

func TestCreateSession_WithWorktree_SetsFields(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/tmp/worktrees"))

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "wt-session",
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "feature/wt",
			BaseRef:      "main",
			CreateBranch: true,
			Path:         "/tmp/worktrees/repo/feature-wt",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.WorkingDir != "/tmp/worktrees/repo/feature-wt" {
		t.Errorf("expected WorkingDir to be worktree path, got %q", session.WorkingDir)
	}
	if session.RepoRoot != "/repo" {
		t.Errorf("expected RepoRoot=/repo, got %q", session.RepoRoot)
	}
	if session.GitBranch != "feature/wt" {
		t.Errorf("expected GitBranch=feature/wt, got %q", session.GitBranch)
	}
	if !session.WorktreeManaged {
		t.Error("expected WorktreeManaged=true")
	}
	// Async path: session is returned immediately with StatusProvisioning
	if session.Status != domain.StatusProvisioning {
		t.Errorf("expected status provisioning for async worktree session, got %q", session.Status)
	}
}

func TestCreateSession_WithWorktree_ComputesDefaultPath(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	git.infoFn = func(path string) (ports.GitInfo, error) {
		return ports.GitInfo{IsRepo: true, RepoRoot: "/Users/user/myrepo", CurrentBranch: "main"}, nil
	}
	pathCh := make(chan string, 1)
	git.addWorktreeFn = func(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
		pathCh <- wtPath
		return ports.Worktree{Path: wtPath, Branch: branch}, nil
	}

	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/wt"))

	_, err := svc.CreateSession(context.Background(), CreateSessionInput{
		WorkingDir: "/Users/user/myrepo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/Users/user/myrepo",
			Branch:       "feat/foo",
			CreateBranch: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Wait for the async goroutine to call AddWorktree
	capturedPath := <-pathCh
	// Path should be worktreesDir/repoBase/sanitizedBranch
	expected := "/wt/myrepo/feat-foo"
	if capturedPath != expected {
		t.Errorf("expected computed path %q, got %q", expected, capturedPath)
	}
}

func TestCreateSession_WithWorktree_NotRepo(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	git.infoFn = func(path string) (ports.GitInfo, error) {
		return ports.GitInfo{IsRepo: false}, nil
	}

	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/tmp/worktrees"))

	_, err := svc.CreateSession(context.Background(), CreateSessionInput{
		WorkingDir: "/notrepo",
		Worktree: &WorktreeSpec{
			RepoPath: "/notrepo",
			Branch:   "branch",
		},
	})
	if err == nil {
		t.Fatal("expected error when path is not a git repo")
	}
}

// TestCreateSession_WithWorktree_SpawnFailureCleansUp is now covered by
// TestProvisionWorktree_SpawnFails_UpdatesSessionToFailedAndCleansWorktree below.
// This test keeps a smoke-check that CreateSession itself succeeds (returns provisioning),
// and the async goroutine does the cleanup — tested via the dedicated provision tests.
func TestCreateSession_WithWorktree_SpawnFailureCleansUp(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	pm.spawnErr = fmt.Errorf("spawn failed")
	git := newMockGitService()
	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/tmp/worktrees"))

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "branch",
			Path:         "/tmp/worktrees/repo/branch",
			CreateBranch: true,
		},
	})
	// Async path: CreateSession returns immediately with StatusProvisioning, no error
	if err != nil {
		t.Fatalf("expected no error from CreateSession (async path), got %v", err)
	}
	if session.Status != domain.StatusProvisioning {
		t.Errorf("expected StatusProvisioning, got %q", session.Status)
	}
}

func TestDeleteSession_DoesNotRemoveWorktree(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/tmp/worktrees"))

	// Insert a managed worktree session directly
	sess := domain.Session{
		ID:              "sess-wt",
		Name:            "wt",
		WorkingDir:      "/tmp/wt",
		Status:          domain.StatusStopped,
		RepoRoot:        "/repo",
		WorktreeManaged: true,
	}
	_ = repo.Create(context.Background(), sess)

	if err := svc.DeleteSession(context.Background(), "sess-wt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Worktree directory should NOT be removed by DeleteSession
	if len(git.removedWorktreesSafe()) != 0 {
		t.Error("expected worktree to NOT be removed by DeleteSession")
	}
}

// --- Mock SessionEventBroadcaster ---

type mockBroadcaster struct {
	mu     sync.Mutex
	events []broadcastEvent
	ch     chan broadcastEvent
}

type broadcastEvent struct {
	sessionID string
	name      string
	status    string
	errMsg    string
}

func newMockBroadcaster() *mockBroadcaster {
	return &mockBroadcaster{ch: make(chan broadcastEvent, 10)}
}

func (m *mockBroadcaster) BroadcastSessionStatus(sessionID, name, status, errMsg string) {
	evt := broadcastEvent{sessionID: sessionID, name: name, status: status, errMsg: errMsg}
	m.mu.Lock()
	m.events = append(m.events, evt)
	m.mu.Unlock()
	m.ch <- evt
}

func (m *mockBroadcaster) waitForEvent(t *testing.T, timeout time.Duration) broadcastEvent {
	t.Helper()
	select {
	case evt := <-m.ch:
		return evt
	case <-time.After(timeout):
		t.Fatal("timed out waiting for broadcaster event")
		return broadcastEvent{}
	}
}

// --- waitForSessionStatus polls the repo until the session reaches the expected status ---

func waitForSessionStatus(t *testing.T, repo *mockRepo, sessionID string, want domain.SessionStatus, timeout time.Duration) domain.Session {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s, err := repo.Get(context.Background(), sessionID)
		if err == nil && s.Status == want {
			return s
		}
		time.Sleep(5 * time.Millisecond)
	}
	s, _ := repo.Get(context.Background(), sessionID)
	t.Fatalf("timed out waiting for session %s to reach status %q (got %q)", sessionID, want, s.Status)
	return domain.Session{}
}

// --- Async provisioning behaviour tests ---

// TestCreateSession_AsyncWorktree_ReturnsProvisioningImmediately verifies that
// when a worktree spec is given, CreateSession persists the session with
// StatusProvisioning and returns immediately without waiting for git or process.
func TestCreateSession_AsyncWorktree_ReturnsProvisioningImmediately(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()

	// Block AddWorktree until we assert the provisioning status
	addStarted := make(chan struct{})
	addUnblock := make(chan struct{})
	git.addWorktreeFn = func(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
		close(addStarted)
		<-addUnblock
		return ports.Worktree{Path: wtPath, Branch: branch}, nil
	}

	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/wt"))

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "async-session",
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "feat/async",
			CreateBranch: true,
			Path:         "/wt/repo/feat-async",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if session.Status != domain.StatusProvisioning {
		t.Errorf("expected StatusProvisioning, got %q", session.Status)
	}
	// Session must already be in the repo
	stored, err := repo.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("session not stored in repo: %v", err)
	}
	if stored.Status != domain.StatusProvisioning {
		t.Errorf("expected stored status provisioning, got %q", stored.Status)
	}

	// Unblock the goroutine to let it finish cleanly
	close(addUnblock)
	<-addStarted // ensure it started before we unblock
}

// TestProvisionWorktree_Success_UpdatesSessionToRunning verifies the happy path:
// the goroutine creates the worktree, spawns the process, and transitions the
// session to StatusRunning, then broadcasts the new status.
func TestProvisionWorktree_Success_UpdatesSessionToRunning(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	broadcaster := newMockBroadcaster()

	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/wt"),
		WithBroadcaster(broadcaster),
	)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "prov-success",
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "feat/success",
			CreateBranch: true,
			Path:         "/wt/repo/feat-success",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Status != domain.StatusProvisioning {
		t.Fatalf("expected StatusProvisioning, got %q", session.Status)
	}

	// Wait for the broadcaster to fire a "running" event
	evt := broadcaster.waitForEvent(t, 2*time.Second)
	if evt.sessionID != session.ID {
		t.Errorf("expected event for session %s, got %s", session.ID, evt.sessionID)
	}
	if evt.status != string(domain.StatusRunning) {
		t.Errorf("expected broadcast status running, got %q", evt.status)
	}
	if evt.errMsg != "" {
		t.Errorf("expected empty errMsg, got %q", evt.errMsg)
	}

	// Session must be running in the repo
	stored := waitForSessionStatus(t, repo, session.ID, domain.StatusRunning, 2*time.Second)
	if stored.PID == 0 {
		t.Error("expected PID to be set after successful provisioning")
	}
	if stored.Error != "" {
		t.Errorf("expected empty Error field, got %q", stored.Error)
	}
}

// TestProvisionWorktree_AddWorktreeFails_UpdatesSessionToFailed verifies that
// when AddWorktree returns an error the session is transitioned to StatusFailed
// with a descriptive error message and the failure is broadcast.
func TestProvisionWorktree_AddWorktreeFails_UpdatesSessionToFailed(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	broadcaster := newMockBroadcaster()

	git.addWorktreeFn = func(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
		return ports.Worktree{}, errors.New("git conflict")
	}

	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/wt"),
		WithBroadcaster(broadcaster),
	)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "prov-fail-wt",
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "feat/fail",
			CreateBranch: true,
			Path:         "/wt/repo/feat-fail",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error from CreateSession: %v", err)
	}

	// Wait for the failure broadcast
	evt := broadcaster.waitForEvent(t, 2*time.Second)
	if evt.status != string(domain.StatusFailed) {
		t.Errorf("expected broadcast status failed, got %q", evt.status)
	}
	if evt.errMsg == "" {
		t.Error("expected non-empty errMsg in failure broadcast")
	}

	// Session in repo must be StatusFailed with Error set
	stored := waitForSessionStatus(t, repo, session.ID, domain.StatusFailed, 2*time.Second)
	if stored.Error == "" {
		t.Error("expected Error field to be set on failed session")
	}
	if !contains(stored.Error, "git conflict") {
		t.Errorf("expected error to mention git conflict, got %q", stored.Error)
	}
	// No process should have been spawned
	if len(pm.killPIDsSafe()) != 0 || pm.nextPIDSafe() != 42 {
		t.Error("expected no process to be spawned when AddWorktree fails")
	}
}

// TestProvisionWorktree_SpawnFails_UpdatesSessionToFailedAndCleansWorktree verifies
// that when Spawn fails after a successful AddWorktree, the worktree is removed
// and the session is transitioned to StatusFailed.
func TestProvisionWorktree_SpawnFails_UpdatesSessionToFailedAndCleansWorktree(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	pm.spawnErr = fmt.Errorf("spawn failed")
	git := newMockGitService()
	broadcaster := newMockBroadcaster()

	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/wt"),
		WithBroadcaster(broadcaster),
	)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "spawn-fail",
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "feat/spawn-fail",
			CreateBranch: true,
			Path:         "/wt/repo/feat-spawn-fail",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error from CreateSession: %v", err)
	}

	// Wait for failure broadcast
	evt := broadcaster.waitForEvent(t, 2*time.Second)
	if evt.status != string(domain.StatusFailed) {
		t.Errorf("expected broadcast status failed, got %q", evt.status)
	}

	// Session must be failed
	stored := waitForSessionStatus(t, repo, session.ID, domain.StatusFailed, 2*time.Second)
	if !contains(stored.Error, "spawn process") {
		t.Errorf("expected spawn error message, got %q", stored.Error)
	}

	// Worktree must have been cleaned up (force removed)
	if len(git.removedWorktreesSafe()) == 0 {
		t.Error("expected orphaned worktree to be cleaned up on spawn failure")
	}
	if !git.forceFlagsSafe()[0] {
		t.Error("expected force=true when cleaning up orphaned worktree")
	}
}

// TestDeleteSession_CancelsInFlightProvision verifies that deleting a
// provisioning session cancels the in-flight goroutine context.
func TestDeleteSession_CancelsInFlightProvision(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()

	// AddWorktree will block until context is cancelled
	ctxCancelled := make(chan struct{})
	git.addWorktreeFn = func(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
		<-ctx.Done()
		close(ctxCancelled)
		return ports.Worktree{}, ctx.Err()
	}

	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/wt"))

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "cancel-prov",
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "feat/cancel",
			CreateBranch: true,
			Path:         "/wt/repo/feat-cancel",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error from CreateSession: %v", err)
	}

	// Delete the session while it is provisioning
	if err := svc.DeleteSession(context.Background(), session.ID); err != nil {
		t.Fatalf("unexpected error from DeleteSession: %v", err)
	}

	// The goroutine's context must be cancelled
	select {
	case <-ctxCancelled:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("timed out: provisioning goroutine context was not cancelled on DeleteSession")
	}
}

// TestGetPTYHandle_ProvisioningSession_ReturnsError verifies that trying to get
// a PTY handle for a session that is still provisioning returns ErrSessionProvisioning.
func TestGetPTYHandle_ProvisioningSession_ReturnsError(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	// Insert a session in provisioning state directly
	sess := domain.Session{
		ID:         "sess-prov",
		Name:       "prov",
		WorkingDir: "/tmp/wt",
		Status:     domain.StatusProvisioning,
	}
	_ = repo.Create(context.Background(), sess)

	_, err := svc.GetPTYHandle("sess-prov")
	if err == nil {
		t.Fatal("expected error for provisioning session")
	}
	if !errors.Is(err, ErrSessionProvisioning) {
		t.Errorf("expected ErrSessionProvisioning, got %v", err)
	}
}

// --- Comprehensive edge-case, concurrency, and failure-path tests ---

// TestProvisionWorktree_ContextCancelledDuringAddWorktree verifies that
// cancelling the provision context mid-flight (while AddWorktree is blocking)
// causes the session to transition to StatusFailed and the goroutine to exit.
func TestProvisionWorktree_ContextCancelledDuringAddWorktree(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	broadcaster := newMockBroadcaster()

	// AddWorktree blocks until ctx is done — simulates a slow git operation.
	goroutineExited := make(chan struct{})
	git.addWorktreeFn = func(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
		defer close(goroutineExited)
		<-ctx.Done()
		return ports.Worktree{}, ctx.Err()
	}

	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/wt"),
		WithBroadcaster(broadcaster),
	)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "ctx-cancel-prov",
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "feat/ctx-cancel",
			CreateBranch: true,
			Path:         "/wt/repo/feat-ctx-cancel",
		},
	})
	if err != nil {
		t.Fatalf("expected no error from CreateSession: %v", err)
	}
	if session.Status != domain.StatusProvisioning {
		t.Fatalf("expected StatusProvisioning immediately, got %q", session.Status)
	}

	// Cancel the provision by deleting the session
	if err := svc.DeleteSession(context.Background(), session.ID); err != nil {
		t.Fatalf("unexpected error from DeleteSession: %v", err)
	}

	// Goroutine must exit cleanly
	select {
	case <-goroutineExited:
		// good — goroutine observed context cancellation and returned
	case <-time.After(2 * time.Second):
		t.Fatal("timed out: provision goroutine did not exit after context cancellation")
	}

	// The broadcaster must have fired a StatusFailed event
	evt := broadcaster.waitForEvent(t, 2*time.Second)
	if evt.status != string(domain.StatusFailed) {
		t.Errorf("expected StatusFailed broadcast after context cancellation, got %q", evt.status)
	}

	// The session must now be StatusFailed in the repo (it was deleted, so Get should fail)
	_, getErr := repo.Get(context.Background(), session.ID)
	if getErr == nil {
		// Session was re-stored as Failed after cancellation — check status
		stored, _ := repo.Get(context.Background(), session.ID)
		if stored.Status != domain.StatusFailed {
			t.Errorf("expected session status to be Failed, got %q", stored.Status)
		}
	}
	// If getErr != nil, the session was deleted before the goroutine updated it,
	// which is also a valid outcome — the goroutine must have exited cleanly.
}

// TestProvisionWorktree_ConcurrentProvisions verifies that 5 concurrent sessions
// with worktrees each resolve independently with the correct PID.
func TestProvisionWorktree_ConcurrentProvisions(t *testing.T) {
	const n = 5
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	broadcaster := newMockBroadcaster()

	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/wt"),
		WithBroadcaster(broadcaster),
	)

	type result struct {
		session domain.Session
		err     error
	}
	results := make([]result, n)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, err := svc.CreateSession(context.Background(), CreateSessionInput{
				Name:       fmt.Sprintf("concurrent-session-%d", i),
				WorkingDir: "/repo",
				Worktree: &WorktreeSpec{
					RepoPath:     "/repo",
					Branch:       fmt.Sprintf("feat/concurrent-%d", i),
					CreateBranch: true,
					Path:         fmt.Sprintf("/wt/repo/concurrent-%d", i),
				},
			})
			results[i] = result{s, err}
		}(i)
	}
	wg.Wait()

	// All CreateSession calls must succeed with StatusProvisioning
	for i, r := range results {
		if r.err != nil {
			t.Errorf("session %d: unexpected error: %v", i, r.err)
			continue
		}
		if r.session.Status != domain.StatusProvisioning {
			t.Errorf("session %d: expected StatusProvisioning, got %q", i, r.session.Status)
		}
	}

	// Wait for all n "running" broadcast events
	runningCount := 0
	for runningCount < n {
		select {
		case evt := <-broadcaster.ch:
			if evt.status == string(domain.StatusRunning) {
				runningCount++
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for all sessions to reach running (got %d/%d)", runningCount, n)
		}
	}

	// Each session must be stored as running with a unique PID
	seenPIDs := make(map[int]bool)
	for _, r := range results {
		if r.err != nil {
			continue
		}
		stored := waitForSessionStatus(t, repo, r.session.ID, domain.StatusRunning, 2*time.Second)
		if stored.PID == 0 {
			t.Errorf("session %s: expected non-zero PID", r.session.ID)
		}
		if seenPIDs[stored.PID] {
			t.Errorf("session %s: duplicate PID %d", r.session.ID, stored.PID)
		}
		seenPIDs[stored.PID] = true
	}
	if len(seenPIDs) != n {
		t.Errorf("expected %d unique PIDs, got %d", n, len(seenPIDs))
	}
}

// TestDeleteSession_WhileProvisioning verifies that deleting a session that is
// still provisioning cancels the provision goroutine and removes the session
// record from the repository.
func TestDeleteSession_WhileProvisioning(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()

	// Block AddWorktree until we call DeleteSession, then let it complete
	// with a context.Canceled error.
	provStarted := make(chan struct{})
	var provCtxDone <-chan struct{}
	git.addWorktreeFn = func(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
		provCtxDone = ctx.Done()
		close(provStarted)
		<-ctx.Done()
		return ports.Worktree{}, ctx.Err()
	}

	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/wt"))

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "delete-while-prov",
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "feat/delete-prov",
			CreateBranch: true,
			Path:         "/wt/repo/delete-prov",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for the goroutine to be inside AddWorktree
	select {
	case <-provStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for provision goroutine to start AddWorktree")
	}

	// Delete while provisioning
	if err := svc.DeleteSession(context.Background(), session.ID); err != nil {
		t.Fatalf("unexpected error from DeleteSession: %v", err)
	}

	// Session must be gone from the repository
	if _, err := repo.Get(context.Background(), session.ID); err == nil {
		t.Error("expected session to be deleted from repo after DeleteSession")
	}

	// The provision context must be cancelled
	select {
	case <-provCtxDone:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("timed out: provision context was not cancelled after DeleteSession")
	}

	// No process must have been spawned
	if pids := pm.killPIDsSafe(); len(pids) != 0 {
		t.Errorf("expected no process to have been killed (none spawned), got %v", pids)
	}
}

// TestCreateSession_NonWorktree_StillSync verifies that a session without a
// worktree spec is still created synchronously and returns StatusRunning.
func TestCreateSession_NonWorktree_StillSync(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()

	// Block AddWorktree to prove it is never called on the sync path.
	addCalled := false
	git.addWorktreeFn = func(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
		addCalled = true
		return ports.Worktree{}, nil
	}

	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/wt"))

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "sync-session",
		WorkingDir: "/tmp",
		// No Worktree spec → synchronous path
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must return immediately as StatusRunning (synchronous)
	if session.Status != domain.StatusRunning {
		t.Errorf("expected StatusRunning for non-worktree session, got %q", session.Status)
	}
	if session.PID == 0 {
		t.Error("expected PID to be set synchronously")
	}

	// Session is immediately persisted and retrievable
	stored, err := repo.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("session not in repo: %v", err)
	}
	if stored.Status != domain.StatusRunning {
		t.Errorf("expected stored status running, got %q", stored.Status)
	}

	// AddWorktree must never have been called
	if addCalled {
		t.Error("AddWorktree should not be called on the synchronous (non-worktree) path")
	}
}

// TestGetPTYHandle_FailedSession verifies that trying to get a PTY handle for
// a failed session returns a descriptive error (not ErrSessionProvisioning).
func TestGetPTYHandle_FailedSession(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	sess := domain.Session{
		ID:         "sess-failed",
		Name:       "failed",
		WorkingDir: "/tmp/wt",
		Status:     domain.StatusFailed,
		Error:      "spawn process: exec failed",
	}
	_ = repo.Create(context.Background(), sess)

	_, err := svc.GetPTYHandle("sess-failed")
	if err == nil {
		t.Fatal("expected error for failed session")
	}
	// Must not be ErrSessionProvisioning
	if errors.Is(err, ErrSessionProvisioning) {
		t.Errorf("expected non-provisioning error for failed session, got ErrSessionProvisioning")
	}
}

// TestProvisionWorktree_SyncMap_ConcurrentAccessNoPanic exercises the
// sync.Map used for provisionCtxs with concurrent CreateSession and
// DeleteSession calls to verify no races or panics occur under the race
// detector.
func TestProvisionWorktree_SyncMap_ConcurrentAccessNoPanic(t *testing.T) {
	const n = 10
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	broadcaster := newMockBroadcaster()

	// Make AddWorktree slightly delayed to increase the chance that Delete
	// races with provision in-flight.
	git.addWorktreeFn = func(ctx context.Context, repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
		select {
		case <-ctx.Done():
			return ports.Worktree{}, ctx.Err()
		default:
		}
		return ports.Worktree{Path: wtPath, Branch: branch}, nil
	}

	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/wt"),
		WithBroadcaster(broadcaster),
	)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, err := svc.CreateSession(context.Background(), CreateSessionInput{
				Name:       fmt.Sprintf("race-session-%d", i),
				WorkingDir: "/repo",
				Worktree: &WorktreeSpec{
					RepoPath:     "/repo",
					Branch:       fmt.Sprintf("feat/race-%d", i),
					CreateBranch: true,
					Path:         fmt.Sprintf("/wt/repo/race-%d", i),
				},
			})
			if err != nil {
				return
			}
			// Immediately try to delete — races with provision goroutine
			_ = svc.DeleteSession(context.Background(), s.ID)
		}(i)
	}
	wg.Wait()
	// Drain any broadcaster events to avoid goroutine leaks
	done := time.After(2 * time.Second)
	for {
		select {
		case <-broadcaster.ch:
		case <-done:
			return
		}
	}
}

// contains is a helper to check if a string contains a substring.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

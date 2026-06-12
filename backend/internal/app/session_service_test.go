package app

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/ports"
)

// --- Mock SessionRepository ---

type mockRepo struct {
	sessions map[string]domain.Session
	createFn func(ctx context.Context, s domain.Session) error
}

func newMockRepo() *mockRepo {
	return &mockRepo{sessions: make(map[string]domain.Session)}
}

func (m *mockRepo) Create(ctx context.Context, s domain.Session) error {
	if m.createFn != nil {
		return m.createFn(ctx, s)
	}
	m.sessions[s.ID] = s
	return nil
}

func (m *mockRepo) Get(ctx context.Context, id string) (domain.Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return domain.Session{}, fmt.Errorf("session not found: %s", id)
	}
	return s, nil
}

func (m *mockRepo) List(ctx context.Context) ([]domain.Session, error) {
	var result []domain.Session
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockRepo) Update(ctx context.Context, s domain.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}

// --- Mock ProcessManager ---

type mockProcessManager struct {
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
	m.killPIDs = append(m.killPIDs, pid)
	delete(m.alive, pid)
	return nil
}

func (m *mockProcessManager) IsAlive(pid int) bool {
	return m.alive[pid]
}

func (m *mockProcessManager) KillAll() {}

func (m *mockProcessManager) GetHandle(pid int) (*ports.PTYHandle, bool) {
	h, ok := m.handles[pid]
	return h, ok
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

type mockGitService struct {
	infoFn           func(path string) (ports.GitInfo, error)
	addWorktreeFn    func(repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error)
	removeWorktreeFn func(repoRoot, wtPath string, force bool) error
	isCleanFn        func(path string) (bool, error)
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

func (m *mockGitService) AddWorktree(repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
	if m.addWorktreeFn != nil {
		return m.addWorktreeFn(repoRoot, wtPath, branch, baseRef, create)
	}
	return ports.Worktree{Path: wtPath, Branch: branch}, nil
}

func (m *mockGitService) RemoveWorktree(repoRoot, wtPath string, force bool) error {
	m.removedWorktrees = append(m.removedWorktrees, wtPath)
	m.forceFlags = append(m.forceFlags, force)
	if m.removeWorktreeFn != nil {
		return m.removeWorktreeFn(repoRoot, wtPath, force)
	}
	return nil
}

func (m *mockGitService) IsClean(path string) (bool, error) {
	if m.isCleanFn != nil {
		return m.isCleanFn(path)
	}
	return true, nil
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
	oldDone := pm.doneChans[oldPID]

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
}

func TestCreateSession_WithWorktree_ComputesDefaultPath(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	git.infoFn = func(path string) (ports.GitInfo, error) {
		return ports.GitInfo{IsRepo: true, RepoRoot: "/Users/user/myrepo", CurrentBranch: "main"}, nil
	}
	var capturedPath string
	git.addWorktreeFn = func(repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
		capturedPath = wtPath
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

func TestCreateSession_WithWorktree_SpawnFailureCleansUp(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	pm.spawnErr = fmt.Errorf("spawn failed")
	git := newMockGitService()
	svc := NewSessionService(repo, pm, nil, WithGitService(git, "/tmp/worktrees"))

	_, err := svc.CreateSession(context.Background(), CreateSessionInput{
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "branch",
			Path:         "/tmp/worktrees/repo/branch",
			CreateBranch: true,
		},
	})
	if err == nil {
		t.Fatal("expected error when spawn fails")
	}
	// Worktree should have been cleaned up (force removed)
	if len(git.removedWorktrees) == 0 {
		t.Error("expected orphaned worktree to be cleaned up on spawn failure")
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
	if len(git.removedWorktrees) != 0 {
		t.Error("expected worktree to NOT be removed by DeleteSession")
	}
}

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
	spawnErr  error
	killPIDs  []int
	spawnArgs []string
}

func newMockProcessManager() *mockProcessManager {
	return &mockProcessManager{
		alive:   make(map[int]bool),
		handles: make(map[int]*ports.PTYHandle),
	}
}

func (m *mockProcessManager) Spawn(ctx context.Context, workingDir string, args ...string) (*ports.PTYHandle, error) {
	m.spawnArgs = args
	if m.spawnErr != nil {
		return nil, m.spawnErr
	}
	done := make(chan error, 1)
	h := &ports.PTYHandle{
		PTY:  os.Stdin, // placeholder
		PID:  42,
		Done: done,
	}
	m.alive[42] = true
	m.handles[42] = h
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

// --- Tests ---

func TestCreateSession_Success(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	svc := NewSessionService(repo, pm, nil)

	s, err := svc.CreateSession(context.Background(), "test", "/tmp", "", false)
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

	session, err := svc.CreateSession(context.Background(), "", "/home/user/my-project", "", false)
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

	_, err := svc.CreateSession(context.Background(), "test", "", "", false)
	if err == nil {
		t.Fatal("expected error for empty working dir")
	}
}

func TestCreateSession_SpawnFailure(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	pm.spawnErr = fmt.Errorf("spawn failed")
	svc := NewSessionService(repo, pm, nil)

	_, err := svc.CreateSession(context.Background(), "test", "/tmp", "", false)
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

	_, err := svc.CreateSession(context.Background(), "test", "/tmp", "", false)
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

	_, err := svc.CreateSession(context.Background(), "test", "/tmp", "", true)
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

	_, err := svc.CreateSession(context.Background(), "test", "/tmp", "", false)
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

	created, _ := svc.CreateSession(context.Background(), "test", "/tmp", "", false)
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

	s, _ := svc.CreateSession(context.Background(), "test", "/tmp", "", false)
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

	s, _ := svc.CreateSession(context.Background(), "test", "/tmp", "", false)
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

	s, _ := svc.CreateSession(context.Background(), "test", "/tmp", "", false)
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

	s, _ := svc.CreateSession(context.Background(), "test", "/tmp", "", false)
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

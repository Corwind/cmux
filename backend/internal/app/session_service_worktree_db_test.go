package app

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/ports"
)

// --- Mock WorktreeRepository ---

// mockWorktreeRepo is goroutine-safe: the async DeleteWorktree goroutine writes
// (SetStatus / Delete) concurrently with test reads, so all map access is guarded
// by mu to keep the race detector happy.
type mockWorktreeRepo struct {
	mu           sync.Mutex
	worktrees    map[string]domain.ManagedWorktree
	byPath       map[string]string // path → id
	setStatusErr error             // if set, SetStatus returns this error
	deleteErr    error             // if set, Delete returns this error (record kept)
}

// setSetStatusErr / setDeleteErr mutate the injection fields under the mock
// mutex so writes from the test goroutine don't race the removeWorktree
// goroutine's locked reads.
func (m *mockWorktreeRepo) setSetStatusErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setStatusErr = err
}

func (m *mockWorktreeRepo) setDeleteErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteErr = err
}

func newMockWorktreeRepo() *mockWorktreeRepo {
	return &mockWorktreeRepo{
		worktrees: make(map[string]domain.ManagedWorktree),
		byPath:    make(map[string]string),
	}
}

func (m *mockWorktreeRepo) Create(ctx context.Context, wt domain.ManagedWorktree) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byPath[wt.Path]; exists {
		return nil // idempotent
	}
	m.worktrees[wt.ID] = wt
	m.byPath[wt.Path] = wt.ID
	return nil
}

func (m *mockWorktreeRepo) List(ctx context.Context) ([]domain.ManagedWorktree, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.ManagedWorktree
	for _, wt := range m.worktrees {
		result = append(result, wt)
	}
	return result, nil
}

func (m *mockWorktreeRepo) Get(ctx context.Context, id string) (domain.ManagedWorktree, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	wt, ok := m.worktrees[id]
	if !ok {
		return domain.ManagedWorktree{}, domain.ErrWorktreeNotFound(id)
	}
	return wt, nil
}

func (m *mockWorktreeRepo) GetByPath(ctx context.Context, path string) (domain.ManagedWorktree, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byPath[path]
	if !ok {
		return domain.ManagedWorktree{}, domain.ErrWorktreeNotFound(path)
	}
	return m.worktrees[id], nil
}

func (m *mockWorktreeRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	wt, ok := m.worktrees[id]
	if !ok {
		return nil
	}
	delete(m.byPath, wt.Path)
	delete(m.worktrees, id)
	return nil
}

func (m *mockWorktreeRepo) DeleteByPath(ctx context.Context, path string) error {
	m.mu.Lock()
	id, ok := m.byPath[path]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return m.Delete(ctx, id)
}

func (m *mockWorktreeRepo) SetSession(ctx context.Context, worktreeID string, sessionID *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	wt, ok := m.worktrees[worktreeID]
	if !ok {
		return nil
	}
	wt.SessionID = sessionID
	m.worktrees[worktreeID] = wt
	return nil
}

func (m *mockWorktreeRepo) SetStatus(ctx context.Context, id string, status domain.WorktreeStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setStatusErr != nil {
		return m.setStatusErr
	}
	wt, ok := m.worktrees[id]
	if !ok {
		return nil
	}
	wt.Status = status
	m.worktrees[id] = wt
	return nil
}

// Compile-time check
var _ ports.WorktreeRepository = (*mockWorktreeRepo)(nil)

// --- Tests ---

func TestCreateSession_WithWorktree_TracksInDB(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	wtr := newMockWorktreeRepo()
	broadcaster := newMockBroadcaster()
	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/tmp/worktrees"),
		WithWorktreeRepository(wtr),
		WithBroadcaster(broadcaster),
	)

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

	// Wait for the async goroutine to complete (broadcaster fires on StatusRunning)
	broadcaster.waitForEvent(t, 2*time.Second)

	// Worktree record must be created
	wt, err := wtr.GetByPath(context.Background(), session.WorkingDir)
	if err != nil {
		t.Fatalf("worktree record not created: %v", err)
	}
	if wt.Branch != "feature/wt" {
		t.Errorf("expected branch 'feature/wt', got %q", wt.Branch)
	}

	// SessionID must be set on the worktree
	if wt.SessionID == nil || *wt.SessionID != session.ID {
		t.Errorf("expected SessionID %q on worktree, got %v", session.ID, wt.SessionID)
	}
}

func TestCreateSession_WithWorktree_AdoptsExistingRecord(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	wtr := newMockWorktreeRepo()
	broadcaster := newMockBroadcaster()

	// Pre-populate a worktree record for the same path
	existing := domain.ManagedWorktree{
		ID:        "existing-wt",
		Path:      "/tmp/worktrees/repo/feature-wt",
		Branch:    "feature/wt",
		RepoRoot:  "/repo",
		CreatedAt: time.Now(),
	}
	_ = wtr.Create(context.Background(), existing)

	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/tmp/worktrees"),
		WithWorktreeRepository(wtr),
		WithBroadcaster(broadcaster),
	)

	session, err := svc.CreateSession(context.Background(), CreateSessionInput{
		Name:       "wt-session",
		WorkingDir: "/repo",
		Worktree: &WorktreeSpec{
			RepoPath:     "/repo",
			Branch:       "feature/wt",
			CreateBranch: false,
			Path:         "/tmp/worktrees/repo/feature-wt",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for async goroutine to complete
	broadcaster.waitForEvent(t, 2*time.Second)

	// Still just one worktree record
	wts, _ := wtr.List(context.Background())
	if len(wts) != 1 {
		t.Errorf("expected 1 worktree record, got %d", len(wts))
	}

	// Session linked to the pre-existing record via SessionID
	wt, err := wtr.Get(context.Background(), "existing-wt")
	if err != nil {
		t.Fatalf("Get existing-wt failed: %v", err)
	}
	if wt.SessionID == nil || *wt.SessionID != session.ID {
		t.Errorf("expected session linked to existing worktree via SessionID, got %v", wt.SessionID)
	}
}

func TestDeleteSession_Always_KeepsWorktreeRecord(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/tmp/worktrees"),
		WithWorktreeRepository(wtr),
	)

	sess := domain.Session{
		ID:              "sess-wt",
		Name:            "wt",
		WorkingDir:      "/tmp/wt",
		Status:          domain.StatusStopped,
		RepoRoot:        "/repo",
		GitBranch:       "feature/wt",
		WorktreeManaged: true,
	}
	_ = repo.Create(context.Background(), sess)

	sessID := "sess-wt"
	wtRecord := domain.ManagedWorktree{
		ID:        "wt-id",
		Path:      "/tmp/wt",
		Branch:    "feature/wt",
		RepoRoot:  "/repo",
		SessionID: &sessID,
		CreatedAt: time.Now(),
	}
	_ = wtr.Create(context.Background(), wtRecord)
	// Manually set the SessionID since Create is idempotent by path
	_ = wtr.SetSession(context.Background(), "wt-id", &sessID)

	if err := svc.DeleteSession(context.Background(), "sess-wt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Worktree record must still exist
	wt, err := wtr.Get(context.Background(), "wt-id")
	if err != nil {
		t.Error("worktree record should still exist after session delete")
	}

	// In a real DB the FK ON DELETE SET NULL would clear SessionID automatically.
	// The mock simulates this by showing session is gone from repo.
	_ = wt
	if _, err := repo.Get(context.Background(), "sess-wt"); err == nil {
		t.Error("expected session to be deleted from repo")
	}
}

func TestListWorktrees_ReturnsEntries(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/tmp/worktrees"),
		WithWorktreeRepository(wtr),
	)

	// Create a session
	sess := domain.Session{ID: "sess-1", Name: "my-session", WorkingDir: "/tmp/wt1", Status: domain.StatusStopped, WorktreeManaged: true, GitBranch: "feat", RepoRoot: "/repo"}
	_ = repo.Create(context.Background(), sess)

	// Create a worktree record with SessionID set
	sessID := "sess-1"
	wtRecord := domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt1", Branch: "feat", RepoRoot: "/repo", SessionID: &sessID, CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)
	_ = wtr.SetSession(context.Background(), "wt-1", &sessID)

	entries, err := svc.ListWorktrees(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "wt-1" {
		t.Errorf("expected worktree ID 'wt-1', got %q", entries[0].ID)
	}
	if entries[0].SessionName == nil || *entries[0].SessionName != "my-session" {
		t.Errorf("expected SessionName 'my-session', got %v", entries[0].SessionName)
	}
	if entries[0].SessionStatus == nil || *entries[0].SessionStatus != string(domain.StatusStopped) {
		t.Errorf("expected SessionStatus %q, got %v", domain.StatusStopped, entries[0].SessionStatus)
	}
}

func TestListWorktrees_AutoAdoptsUntracked(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/tmp/worktrees"),
		WithWorktreeRepository(wtr),
	)

	// Session marked worktree_managed but no worktree record exists
	sess := domain.Session{
		ID: "sess-old", Name: "old", WorkingDir: "/tmp/old-wt",
		Status: domain.StatusStopped, WorktreeManaged: true,
		GitBranch: "old-branch", RepoRoot: "/repo",
	}
	_ = repo.Create(context.Background(), sess)

	entries, err := svc.ListWorktrees(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for auto-adopted session, got %d", len(entries))
	}
	if entries[0].Path != "/tmp/old-wt" {
		t.Errorf("expected path '/tmp/old-wt', got %q", entries[0].Path)
	}
	if entries[0].SessionName == nil || *entries[0].SessionName != "old" {
		t.Errorf("expected SessionName 'old' after auto-adopt, got %v", entries[0].SessionName)
	}
	if entries[0].SessionStatus == nil {
		t.Error("expected SessionStatus to be set after auto-adopt")
	}
}

func TestListWorktrees_OrphanedWorktree_NoSessions(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil, WithWorktreeRepository(wtr))

	// Worktree record exists but no session
	wtRecord := domain.ManagedWorktree{ID: "wt-orphan", Path: "/tmp/orphan", Branch: "feat", RepoRoot: "/repo", CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)

	entries, err := svc.ListWorktrees(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for orphaned worktree, got %d", len(entries))
	}
	if entries[0].SessionID != nil {
		t.Errorf("expected nil SessionID for orphaned worktree, got %v", entries[0].SessionID)
	}
	if entries[0].SessionName != nil {
		t.Errorf("expected nil SessionName for orphaned worktree, got %v", entries[0].SessionName)
	}
}

func TestDeleteWorktree_LinkedSession_ReturnsError(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil, WithWorktreeRepository(wtr))

	for _, status := range []domain.SessionStatus{domain.StatusStopped, domain.StatusRunning} {
		sess := domain.Session{ID: "sess-1", Name: "s", WorkingDir: "/tmp/wt1", Status: status}
		_ = repo.Create(context.Background(), sess)
		sessID := sess.ID
		wtRecord := domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt1", RepoRoot: "/repo", SessionID: &sessID, CreatedAt: time.Now()}
		_ = wtr.Create(context.Background(), wtRecord)

		err := svc.DeleteWorktree(context.Background(), "wt-1")
		if err == nil {
			t.Fatalf("expected error when session is linked (status=%s)", status)
		}
		if _, ok := err.(*ErrWorktreeHasSession); !ok {
			t.Errorf("expected ErrWorktreeHasSession (status=%s), got %T: %v", status, err, err)
		}

		// clean up for next iteration
		_ = wtr.Delete(context.Background(), "wt-1")
		_ = repo.Delete(context.Background(), sess.ID)
	}
}

// waitForCondition polls fn until it returns true or the timeout elapses,
// failing the test on timeout. Mirrors the existing waitForSessionStatus
// polling pattern and avoids time.Sleep in tests.
func waitForCondition(t *testing.T, timeout time.Duration, msg string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for condition: %s", msg)
}

func newDeletableWorktreeSvc(t *testing.T) (*SessionService, *mockGitService, *mockWorktreeRepo, *mockBroadcaster) {
	t.Helper()
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	wtr := newMockWorktreeRepo()
	broadcaster := newMockBroadcaster()
	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/tmp/worktrees"),
		WithWorktreeRepository(wtr),
		WithBroadcaster(broadcaster),
	)
	return svc, git, wtr, broadcaster
}

func TestDeleteWorktree_AsyncReturnsImmediately(t *testing.T) {
	svc, git, wtr, _ := newDeletableWorktreeSvc(t)

	release := make(chan struct{})
	git.removeWorktreeFn = func(ctx context.Context, repoRoot, wtPath string, force bool) error {
		<-release // block until the test releases the goroutine
		return nil
	}

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	// DeleteWorktree must return before git removal completes.
	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The DB record must still exist while git removal is blocked.
	if _, err := wtr.Get(context.Background(), "wt-1"); err != nil {
		t.Fatalf("expected worktree record to still exist while deletion in flight: %v", err)
	}

	close(release)
}

func TestDeleteWorktree_MarksDeleting(t *testing.T) {
	svc, git, wtr, _ := newDeletableWorktreeSvc(t)

	release := make(chan struct{})
	git.removeWorktreeFn = func(ctx context.Context, repoRoot, wtPath string, force bool) error {
		<-release
		return nil
	}

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wt, err := wtr.Get(context.Background(), "wt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt.Status != domain.WorktreeStatusDeleting {
		t.Errorf("expected status %q, got %q", domain.WorktreeStatusDeleting, wt.Status)
	}

	close(release)
}

func TestDeleteWorktree_GitRemoveCalledWithCorrectArgs(t *testing.T) {
	svc, git, wtr, broadcaster := newDeletableWorktreeSvc(t)

	var gotRepoRoot, gotPath string
	var gotForce bool
	git.removeWorktreeFn = func(ctx context.Context, repoRoot, wtPath string, force bool) error {
		gotRepoRoot, gotPath, gotForce = repoRoot, wtPath, force
		return nil
	}

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	broadcaster.waitForWorktreeEvent(t, 2*time.Second)

	if gotRepoRoot != "/repo" {
		t.Errorf("expected repoRoot %q, got %q", "/repo", gotRepoRoot)
	}
	if gotPath != "/tmp/wt" {
		t.Errorf("expected path %q, got %q", "/tmp/wt", gotPath)
	}
	if gotForce {
		t.Error("expected force=false")
	}
}

func TestDeleteWorktree_DBRecordDeletedAfterGitSuccess(t *testing.T) {
	svc, _, wtr, _ := newDeletableWorktreeSvc(t)

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	waitForCondition(t, 2*time.Second, "worktree record deleted", func() bool {
		_, err := wtr.Get(context.Background(), "wt-1")
		return err != nil
	})
}

func TestDeleteWorktree_DBRecordDeletedEvenOnGitError(t *testing.T) {
	svc, git, wtr, _ := newDeletableWorktreeSvc(t)

	git.removeWorktreeFn = func(ctx context.Context, repoRoot, wtPath string, force bool) error {
		return fmt.Errorf("git boom")
	}

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	waitForCondition(t, 2*time.Second, "worktree record deleted despite git error", func() bool {
		_, err := wtr.Get(context.Background(), "wt-1")
		return err != nil
	})
}

func TestDeleteWorktree_BroadcastsCompletionOnSuccess(t *testing.T) {
	svc, _, wtr, broadcaster := newDeletableWorktreeSvc(t)

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt := broadcaster.waitForWorktreeEvent(t, 2*time.Second)
	if evt.worktreeID != "wt-1" {
		t.Errorf("expected worktree_id %q, got %q", "wt-1", evt.worktreeID)
	}
	if evt.errMsg != "" {
		t.Errorf("expected empty error on success, got %q", evt.errMsg)
	}
}

func TestDeleteWorktree_BroadcastsErrorOnGitFailure(t *testing.T) {
	svc, git, wtr, broadcaster := newDeletableWorktreeSvc(t)

	git.removeWorktreeFn = func(ctx context.Context, repoRoot, wtPath string, force bool) error {
		return fmt.Errorf("git boom")
	}

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evt := broadcaster.waitForWorktreeEvent(t, 2*time.Second)
	if evt.worktreeID != "wt-1" {
		t.Errorf("expected worktree_id %q, got %q", "wt-1", evt.worktreeID)
	}
	if evt.errMsg == "" {
		t.Error("expected non-empty error on git failure")
	}
}

func TestDeleteWorktree_IdempotentWhenAlreadyDeleting(t *testing.T) {
	svc, git, wtr, _ := newDeletableWorktreeSvc(t)

	release := make(chan struct{})
	git.removeWorktreeFn = func(ctx context.Context, repoRoot, wtPath string, force bool) error {
		<-release
		return nil
	}

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	// First call starts deletion (and blocks in git removal).
	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}

	// Wait until the record is marked deleting.
	waitForCondition(t, 2*time.Second, "worktree marked deleting", func() bool {
		wt, err := wtr.Get(context.Background(), "wt-1")
		return err == nil && wt.Status == domain.WorktreeStatusDeleting
	})

	// Second call must be an idempotent no-op.
	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error on idempotent second call: %v", err)
	}

	close(release)

	// Only one git removal must have occurred.
	waitForCondition(t, 2*time.Second, "worktree record deleted", func() bool {
		_, err := wtr.Get(context.Background(), "wt-1")
		return err != nil
	})
	if got := len(git.removedWorktreesSafe()); got != 1 {
		t.Errorf("expected git RemoveWorktree to be called once, got %d", got)
	}
}

func TestDeleteWorktree_ConcurrentCallsLaunchOnce(t *testing.T) {
	svc, git, wtr, broadcaster := newDeletableWorktreeSvc(t)

	release := make(chan struct{})
	git.removeWorktreeFn = func(ctx context.Context, repoRoot, wtPath string, force bool) error {
		<-release
		return nil
	}

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	// Fire N concurrent DeleteWorktree calls on the same ready worktree, all
	// released together. The atomic deletionCtxs gate must let exactly one
	// launch the removal goroutine and turn the rest into no-ops.
	const n = 5
	start := make(chan struct{})
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = svc.DeleteWorktree(context.Background(), "wt-1")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent call %d returned error: %v", i, err)
		}
	}

	// Let the single in-flight removal complete.
	close(release)

	waitForCondition(t, 2*time.Second, "worktree record deleted", func() bool {
		_, err := wtr.Get(context.Background(), "wt-1")
		return err != nil
	})

	if got := len(git.removedWorktreesSafe()); got != 1 {
		t.Errorf("expected git RemoveWorktree to be called exactly once, got %d", got)
	}

	evt := broadcaster.waitForWorktreeEvent(t, 2*time.Second)
	if evt.errMsg != "" {
		t.Errorf("expected empty error on success, got %q", evt.errMsg)
	}
	if got := broadcaster.worktreeEventCount(); got != 1 {
		t.Errorf("expected exactly 1 completion broadcast, got %d", got)
	}
}

func TestDeleteWorktree_HangProtection(t *testing.T) {
	// Shrink the deletion timeout so the test can exercise the hang path quickly.
	old := worktreeDeletionTimeout
	worktreeDeletionTimeout = 10 * time.Millisecond
	defer func() { worktreeDeletionTimeout = old }()

	svc, git, wtr, broadcaster := newDeletableWorktreeSvc(t)

	// Simulate a hung git removal: block until the context (with timeout) fires,
	// then return its error — exactly what runGit does when it kills the process.
	git.removeWorktreeFn = func(ctx context.Context, repoRoot, wtPath string, force bool) error {
		<-ctx.Done()
		return ctx.Err()
	}

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Even though git "hangs", the timeout unblocks the goroutine, which must
	// still delete the DB record and broadcast a non-empty error.
	evt := broadcaster.waitForWorktreeEvent(t, 2*time.Second)
	if evt.worktreeID != "wt-1" {
		t.Errorf("expected worktree_id %q, got %q", "wt-1", evt.worktreeID)
	}
	if evt.errMsg == "" {
		t.Error("expected non-empty error from context timeout on hang")
	}

	waitForCondition(t, 2*time.Second, "worktree record deleted despite git hang", func() bool {
		_, err := wtr.Get(context.Background(), "wt-1")
		return err != nil
	})
}

func TestDeleteWorktree_SetStatusFailure_ReturnsErrorAndCleansUp(t *testing.T) {
	svc, git, wtr, broadcaster := newDeletableWorktreeSvc(t)

	wtr.setSetStatusErr(fmt.Errorf("db boom"))

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	// SetStatus fails → DeleteWorktree must return a wrapped error, never touch
	// git, leave the record present, and fire no broadcast.
	err := svc.DeleteWorktree(context.Background(), "wt-1")
	if err == nil {
		t.Fatal("expected error when SetStatus fails")
	}
	if got := len(git.removedWorktreesSafe()); got != 0 {
		t.Errorf("expected git RemoveWorktree to NOT be called, got %d calls", got)
	}
	if _, err := wtr.Get(context.Background(), "wt-1"); err != nil {
		t.Errorf("expected worktree record to still exist, got %v", err)
	}
	if got := broadcaster.worktreeEventCount(); got != 0 {
		t.Errorf("expected no broadcast on SetStatus failure, got %d", got)
	}

	// The id must be retryable (cleanup ran deletionCtxs.Delete + cancel).
	wtr.setSetStatusErr(nil)
	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	waitForCondition(t, 2*time.Second, "worktree record deleted on retry", func() bool {
		_, err := wtr.Get(context.Background(), "wt-1")
		return err != nil
	})
}

func TestDeleteWorktree_NilGitService_StillDeletesAndBroadcasts(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	wtr := newMockWorktreeRepo()
	broadcaster := newMockBroadcaster()
	// No WithGitService → gitService is nil.
	svc := NewSessionService(repo, pm, nil,
		WithWorktreeRepository(wtr),
		WithBroadcaster(broadcaster),
	)

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// git is skipped (not an error) → broadcast carries an empty errMsg.
	evt := broadcaster.waitForWorktreeEvent(t, 2*time.Second)
	if evt.worktreeID != "wt-1" {
		t.Errorf("expected worktree_id %q, got %q", "wt-1", evt.worktreeID)
	}
	if evt.errMsg != "" {
		t.Errorf("expected empty error with nil git service, got %q", evt.errMsg)
	}

	waitForCondition(t, 2*time.Second, "worktree record deleted", func() bool {
		_, err := wtr.Get(context.Background(), "wt-1")
		return err != nil
	})
}

func TestDeleteWorktree_DBDeleteFailure_StillBroadcasts(t *testing.T) {
	svc, _, wtr, broadcaster := newDeletableWorktreeSvc(t)

	wtr.setDeleteErr(fmt.Errorf("delete boom"))

	_ = wtr.Create(context.Background(), domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()})

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Delete fails (best-effort) but the goroutine must still broadcast.
	evt := broadcaster.waitForWorktreeEvent(t, 2*time.Second)
	if evt.worktreeID != "wt-1" {
		t.Errorf("expected worktree_id %q, got %q", "wt-1", evt.worktreeID)
	}
}

func TestDeleteWorktree_NotFound_ReturnsError(t *testing.T) {
	svc, git, _, broadcaster := newDeletableWorktreeSvc(t)

	err := svc.DeleteWorktree(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown worktree id")
	}
	if got := len(git.removedWorktreesSafe()); got != 0 {
		t.Errorf("expected git RemoveWorktree to NOT be called, got %d", got)
	}
	if got := broadcaster.worktreeEventCount(); got != 0 {
		t.Errorf("expected no broadcast for unknown id, got %d", got)
	}
}

func TestDeleteWorktree_NilRepo_ReturnsError(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	// No WithWorktreeRepository → worktreeRepo is nil.
	svc := NewSessionService(repo, pm, nil)

	if err := svc.DeleteWorktree(context.Background(), "wt-1"); err == nil {
		t.Fatal("expected error when worktree repository not configured")
	}
}

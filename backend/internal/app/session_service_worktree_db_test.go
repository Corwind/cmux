package app

import (
	"context"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/ports"
)

// --- Mock WorktreeRepository ---

type mockWorktreeRepo struct {
	worktrees map[string]domain.ManagedWorktree
	byPath    map[string]string // path → id
}

func newMockWorktreeRepo() *mockWorktreeRepo {
	return &mockWorktreeRepo{
		worktrees: make(map[string]domain.ManagedWorktree),
		byPath:    make(map[string]string),
	}
}

func (m *mockWorktreeRepo) Create(ctx context.Context, wt domain.ManagedWorktree) error {
	if _, exists := m.byPath[wt.Path]; exists {
		return nil // idempotent
	}
	m.worktrees[wt.ID] = wt
	m.byPath[wt.Path] = wt.ID
	return nil
}

func (m *mockWorktreeRepo) List(ctx context.Context) ([]domain.ManagedWorktree, error) {
	var result []domain.ManagedWorktree
	for _, wt := range m.worktrees {
		result = append(result, wt)
	}
	return result, nil
}

func (m *mockWorktreeRepo) Get(ctx context.Context, id string) (domain.ManagedWorktree, error) {
	wt, ok := m.worktrees[id]
	if !ok {
		return domain.ManagedWorktree{}, domain.ErrWorktreeNotFound(id)
	}
	return wt, nil
}

func (m *mockWorktreeRepo) GetByPath(ctx context.Context, path string) (domain.ManagedWorktree, error) {
	id, ok := m.byPath[path]
	if !ok {
		return domain.ManagedWorktree{}, domain.ErrWorktreeNotFound(path)
	}
	return m.worktrees[id], nil
}

func (m *mockWorktreeRepo) Delete(ctx context.Context, id string) error {
	wt, ok := m.worktrees[id]
	if !ok {
		return nil
	}
	delete(m.byPath, wt.Path)
	delete(m.worktrees, id)
	return nil
}

func (m *mockWorktreeRepo) DeleteByPath(ctx context.Context, path string) error {
	id, ok := m.byPath[path]
	if !ok {
		return nil
	}
	return m.Delete(ctx, id)
}

func (m *mockWorktreeRepo) SetSession(ctx context.Context, worktreeID string, sessionID *string) error {
	wt, ok := m.worktrees[worktreeID]
	if !ok {
		return nil
	}
	wt.SessionID = sessionID
	m.worktrees[worktreeID] = wt
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
	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/tmp/worktrees"),
		WithWorktreeRepository(wtr),
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

func TestDeleteOrphanedWorktree_Success(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil,
		WithGitService(git, "/tmp/worktrees"),
		WithWorktreeRepository(wtr),
	)

	wtRecord := domain.ManagedWorktree{ID: "wt-orphan", Path: "/tmp/orphan", Branch: "feat", RepoRoot: "/repo", CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)

	if err := svc.DeleteOrphanedWorktree(context.Background(), "wt-orphan", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := wtr.Get(context.Background(), "wt-orphan"); err == nil {
		t.Error("expected worktree record to be deleted")
	}
}

func TestDeleteOrphanedWorktree_HasSessions_ReturnsError(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil, WithWorktreeRepository(wtr))

	// Stopped session — not running, so ErrWorktreeHasSessions (not ErrWorktreeSessionRunning)
	sess := domain.Session{ID: "sess-1", Name: "s", WorkingDir: "/tmp/wt1", Status: domain.StatusStopped}
	_ = repo.Create(context.Background(), sess)
	sessID := sess.ID
	wtRecord := domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt1", RepoRoot: "/repo", SessionID: &sessID, CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)
	_ = wtr.SetSession(context.Background(), "wt-1", &sessID)

	err := svc.DeleteOrphanedWorktree(context.Background(), "wt-1", false)
	if err == nil {
		t.Fatal("expected error when deleting worktree with active session")
	}
	if _, ok := err.(*ErrWorktreeHasSessions); !ok {
		t.Errorf("expected ErrWorktreeHasSessions, got %T: %v", err, err)
	}
}

func TestDeleteOrphanedWorktree_Force_DeletesWithSessions(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil, WithWorktreeRepository(wtr))

	sess := domain.Session{ID: "sess-1", Name: "s", WorkingDir: "/tmp/wt1", Status: domain.StatusStopped}
	_ = repo.Create(context.Background(), sess)
	sessID := sess.ID
	wtRecord := domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt1", RepoRoot: "/repo", SessionID: &sessID, CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)
	_ = wtr.SetSession(context.Background(), "wt-1", &sessID)

	if err := svc.DeleteOrphanedWorktree(context.Background(), "wt-1", true); err != nil {
		t.Fatalf("unexpected error with force=true: %v", err)
	}
	if _, err := wtr.Get(context.Background(), "wt-1"); err == nil {
		t.Error("expected worktree record to be deleted with force=true")
	}
}

func TestDeleteOrphanedWorktree_RunningSession_BlocksEvenWithForce(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil, WithWorktreeRepository(wtr))

	sess := domain.Session{ID: "sess-running", Name: "s", WorkingDir: "/tmp/wt1", Status: domain.StatusRunning}
	_ = repo.Create(context.Background(), sess)
	sessID := sess.ID
	wtRecord := domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt1", RepoRoot: "/repo", SessionID: &sessID, CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)
	_ = wtr.SetSession(context.Background(), "wt-1", &sessID)

	for _, force := range []bool{false, true} {
		err := svc.DeleteOrphanedWorktree(context.Background(), "wt-1", force)
		if err == nil {
			t.Fatalf("expected error when session is running (force=%v)", force)
		}
		if _, ok := err.(*ErrWorktreeSessionRunning); !ok {
			t.Errorf("expected ErrWorktreeSessionRunning (force=%v), got %T: %v", force, err, err)
		}
	}
}

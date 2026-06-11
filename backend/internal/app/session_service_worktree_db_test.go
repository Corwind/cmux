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
	worktrees  map[string]domain.ManagedWorktree
	byPath     map[string]string // path → id
	links      map[string][]string // worktree_id → []session_id
	sessionMap map[string]string   // session_id → worktree_id
}

func newMockWorktreeRepo() *mockWorktreeRepo {
	return &mockWorktreeRepo{
		worktrees:  make(map[string]domain.ManagedWorktree),
		byPath:     make(map[string]string),
		links:      make(map[string][]string),
		sessionMap: make(map[string]string),
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
	delete(m.links, id)
	return nil
}

func (m *mockWorktreeRepo) DeleteByPath(ctx context.Context, path string) error {
	id, ok := m.byPath[path]
	if !ok {
		return nil
	}
	return m.Delete(ctx, id)
}

func (m *mockWorktreeRepo) LinkSession(ctx context.Context, worktreeID, sessionID string) error {
	for _, sid := range m.links[worktreeID] {
		if sid == sessionID {
			return nil // idempotent
		}
	}
	m.links[worktreeID] = append(m.links[worktreeID], sessionID)
	m.sessionMap[sessionID] = worktreeID
	return nil
}

func (m *mockWorktreeRepo) UnlinkSession(ctx context.Context, sessionID string) error {
	wtID, ok := m.sessionMap[sessionID]
	if !ok {
		return nil
	}
	sids := m.links[wtID]
	filtered := sids[:0]
	for _, sid := range sids {
		if sid != sessionID {
			filtered = append(filtered, sid)
		}
	}
	m.links[wtID] = filtered
	delete(m.sessionMap, sessionID)
	return nil
}

func (m *mockWorktreeRepo) ListSessionIDs(ctx context.Context, worktreeID string) ([]string, error) {
	return m.links[worktreeID], nil
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

	// Session must be linked
	ids, _ := wtr.ListSessionIDs(context.Background(), wt.ID)
	if len(ids) != 1 || ids[0] != session.ID {
		t.Errorf("expected session %q linked to worktree, got %v", session.ID, ids)
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
	// Session linked to the pre-existing record
	ids, _ := wtr.ListSessionIDs(context.Background(), "existing-wt")
	if len(ids) != 1 || ids[0] != session.ID {
		t.Errorf("expected session linked to existing worktree, got %v", ids)
	}
}

func TestDeleteSession_WorktreeKeep_UnlinksSession(t *testing.T) {
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

	wtRecord := domain.ManagedWorktree{
		ID:        "wt-id",
		Path:      "/tmp/wt",
		Branch:    "feature/wt",
		RepoRoot:  "/repo",
		CreatedAt: time.Now(),
	}
	_ = wtr.Create(context.Background(), wtRecord)
	_ = wtr.LinkSession(context.Background(), "wt-id", "sess-wt")

	if err := svc.DeleteSession(context.Background(), "sess-wt", WorktreeActionKeep); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Worktree record must still exist
	if _, err := wtr.Get(context.Background(), "wt-id"); err != nil {
		t.Error("worktree record should still exist after keep")
	}
	// Session unlinked
	ids, _ := wtr.ListSessionIDs(context.Background(), "wt-id")
	if len(ids) != 0 {
		t.Errorf("expected session to be unlinked, got %v", ids)
	}
}

func TestDeleteSession_WorktreeRemove_DeletesRecord(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	git := newMockGitService()
	git.isCleanFn = func(path string) (bool, error) { return true, nil }
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
		WorktreeManaged: true,
	}
	_ = repo.Create(context.Background(), sess)

	wtRecord := domain.ManagedWorktree{ID: "wt-id", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)
	_ = wtr.LinkSession(context.Background(), "wt-id", "sess-wt")

	if err := svc.DeleteSession(context.Background(), "sess-wt", WorktreeActionRemove); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Worktree record must be gone
	if _, err := wtr.Get(context.Background(), "wt-id"); err == nil {
		t.Error("expected worktree record to be deleted after remove")
	}
}

func TestDeleteSession_WorktreeForce_DeletesRecord(t *testing.T) {
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
		WorktreeManaged: true,
	}
	_ = repo.Create(context.Background(), sess)

	wtRecord := domain.ManagedWorktree{ID: "wt-id", Path: "/tmp/wt", RepoRoot: "/repo", CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)
	_ = wtr.LinkSession(context.Background(), "wt-id", "sess-wt")

	if err := svc.DeleteSession(context.Background(), "sess-wt", WorktreeActionForce); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := wtr.Get(context.Background(), "wt-id"); err == nil {
		t.Error("expected worktree record to be deleted after force remove")
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

	// Create a worktree record and link a session
	wtRecord := domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt1", Branch: "feat", RepoRoot: "/repo", CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)

	sess := domain.Session{ID: "sess-1", Name: "s", WorkingDir: "/tmp/wt1", Status: domain.StatusStopped, WorktreeManaged: true, GitBranch: "feat", RepoRoot: "/repo"}
	_ = repo.Create(context.Background(), sess)
	_ = wtr.LinkSession(context.Background(), "wt-1", "sess-1")

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
	if len(entries[0].Sessions) != 1 {
		t.Errorf("expected 1 session in entry, got %d", len(entries[0].Sessions))
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
	if len(entries[0].Sessions) != 1 {
		t.Errorf("expected 1 session linked after auto-adopt, got %d", len(entries[0].Sessions))
	}
}

func TestListWorktrees_OrphanedWorktree_NoSessions(t *testing.T) {
	repo := newMockRepo()
	pm := newMockProcessManager()
	wtr := newMockWorktreeRepo()
	svc := NewSessionService(repo, pm, nil, WithWorktreeRepository(wtr))

	// Worktree record exists but no sessions
	wtRecord := domain.ManagedWorktree{ID: "wt-orphan", Path: "/tmp/orphan", Branch: "feat", RepoRoot: "/repo", CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)

	entries, err := svc.ListWorktrees(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for orphaned worktree, got %d", len(entries))
	}
	if len(entries[0].Sessions) != 0 {
		t.Errorf("expected 0 sessions for orphaned worktree, got %d", len(entries[0].Sessions))
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

	wtRecord := domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt1", RepoRoot: "/repo", CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)
	sess := domain.Session{ID: "sess-1", Name: "s", WorkingDir: "/tmp/wt1", Status: domain.StatusRunning}
	_ = repo.Create(context.Background(), sess)
	_ = wtr.LinkSession(context.Background(), "wt-1", "sess-1")

	err := svc.DeleteOrphanedWorktree(context.Background(), "wt-1", false)
	if err == nil {
		t.Fatal("expected error when deleting worktree with active sessions")
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

	wtRecord := domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/wt1", RepoRoot: "/repo", CreatedAt: time.Now()}
	_ = wtr.Create(context.Background(), wtRecord)
	sess := domain.Session{ID: "sess-1", Name: "s", WorkingDir: "/tmp/wt1", Status: domain.StatusRunning}
	_ = repo.Create(context.Background(), sess)
	_ = wtr.LinkSession(context.Background(), "wt-1", "sess-1")

	if err := svc.DeleteOrphanedWorktree(context.Background(), "wt-1", true); err != nil {
		t.Fatalf("unexpected error with force=true: %v", err)
	}
	if _, err := wtr.Get(context.Background(), "wt-1"); err == nil {
		t.Error("expected worktree record to be deleted with force=true")
	}
}

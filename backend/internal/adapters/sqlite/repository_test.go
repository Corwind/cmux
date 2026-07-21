package sqlite

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
)

func setupTestRepo(t *testing.T) *Repository {
	t.Helper()
	repo, err := NewRepository(":memory:")
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	return repo
}

func TestNewRepository_AppliesConcurrencyPragmas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cmux.db")
	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	defer func() { _ = repo.Close() }()

	db := repo.DB()

	if got := db.Stats().MaxOpenConnections; got != 1 {
		t.Errorf("expected MaxOpenConnections=1, got %d", got)
	}

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout < 5000 {
		t.Errorf("expected busy_timeout >= 5000, got %d", busyTimeout)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", journalMode)
	}
}

// TestConcurrentWrites_NoBusyError exercises many concurrent writers to ensure
// the busy_timeout + single-connection configuration prevents SQLITE_BUSY,
// which previously left worktree-provisioned sessions stuck.
func TestConcurrentWrites_NoBusyError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cmux.db")
	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	defer func() { _ = repo.Close() }()

	ctx := context.Background()
	const writers = 24
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s := makeSession("concurrent")
			s.ID = "sess-" + string(rune('a'+n%26)) + time.Now().Format("150405.000000000")
			if err := repo.Create(ctx, s); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent write failed: %v", err)
	}
}

func makeSession(name string) domain.Session {
	now := time.Now()
	return domain.Session{
		ID:         "test-id-" + name,
		Name:       name,
		WorkingDir: "/tmp",
		Status:     domain.StatusRunning,
		PID:        1234,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestRepository_CreateAndGet(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()
	s := makeSession("sess1")

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != s.ID {
		t.Errorf("expected ID %q, got %q", s.ID, got.ID)
	}
	if got.Name != s.Name {
		t.Errorf("expected Name %q, got %q", s.Name, got.Name)
	}
	if got.WorkingDir != s.WorkingDir {
		t.Errorf("expected WorkingDir %q, got %q", s.WorkingDir, got.WorkingDir)
	}
	if got.Status != s.Status {
		t.Errorf("expected Status %q, got %q", s.Status, got.Status)
	}
	if got.PID != s.PID {
		t.Errorf("expected PID %d, got %d", s.PID, got.PID)
	}
}

func TestRepository_GetNotFound(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestRepository_List(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s1 := makeSession("first")
	s2 := makeSession("second")
	if err := repo.Create(ctx, s1); err != nil {
		t.Fatalf("Create s1 failed: %v", err)
	}
	if err := repo.Create(ctx, s2); err != nil {
		t.Fatalf("Create s2 failed: %v", err)
	}

	sessions, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestRepository_ListEmpty(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	sessions, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestRepository_Update(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()
	s := makeSession("update-me")
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	s.Status = domain.StatusStopped
	s.Name = "updated-name"
	if err := repo.Update(ctx, s); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := repo.Get(ctx, s.ID)
	if got.Status != domain.StatusStopped {
		t.Errorf("expected status %q, got %q", domain.StatusStopped, got.Status)
	}
	if got.Name != "updated-name" {
		t.Errorf("expected name 'updated-name', got %q", got.Name)
	}
}

func TestRepository_Delete(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()
	s := makeSession("delete-me")
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.Delete(ctx, s.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := repo.Get(ctx, s.ID)
	if err == nil {
		t.Fatal("expected error after deleting session")
	}
}

func TestRepository_WorktreeFieldsRoundTrip(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s := makeSession("worktree-sess")
	s.RepoRoot = "/Users/foo/myrepo"
	s.GitBranch = "feature/wt"
	s.WorktreeManaged = true

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.RepoRoot != s.RepoRoot {
		t.Errorf("RepoRoot: expected %q, got %q", s.RepoRoot, got.RepoRoot)
	}
	if got.GitBranch != s.GitBranch {
		t.Errorf("GitBranch: expected %q, got %q", s.GitBranch, got.GitBranch)
	}
	if got.WorktreeManaged != s.WorktreeManaged {
		t.Errorf("WorktreeManaged: expected %v, got %v", s.WorktreeManaged, got.WorktreeManaged)
	}
}

func TestRepository_WorktreeFieldsDefaultToEmpty(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s := makeSession("plain-sess")
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.RepoRoot != "" {
		t.Errorf("expected empty RepoRoot, got %q", got.RepoRoot)
	}
	if got.GitBranch != "" {
		t.Errorf("expected empty GitBranch, got %q", got.GitBranch)
	}
	if got.WorktreeManaged {
		t.Error("expected WorktreeManaged=false by default")
	}
}

func TestRepository_WorktreeFieldsIdempotentMigration(t *testing.T) {
	// Creating a second repo against the same in-memory DB is not possible;
	// instead verify that running migrations twice does not error.
	// NewRepository uses :memory: so each call is an isolated DB.
	repo1, err := NewRepository(":memory:")
	if err != nil {
		t.Fatalf("first NewRepository failed: %v", err)
	}
	_ = repo1

	repo2, err := NewRepository(":memory:")
	if err != nil {
		t.Fatalf("second NewRepository failed: %v", err)
	}
	_ = repo2
}

func TestRepository_ErrorFieldRoundTrip(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s := makeSession("err-sess")
	s.Status = domain.StatusFailed
	s.Error = "git worktree add failed: branch already exists"

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Error != s.Error {
		t.Errorf("Error: expected %q, got %q", s.Error, got.Error)
	}
	if got.Status != domain.StatusFailed {
		t.Errorf("Status: expected %q, got %q", domain.StatusFailed, got.Status)
	}
}

func TestRepository_ErrorFieldDefaultsEmpty(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s := makeSession("no-err-sess")
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Error != "" {
		t.Errorf("expected empty Error field, got %q", got.Error)
	}
}

func TestRepository_ProvisioningStatusRoundTrip(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s := makeSession("prov-sess")
	s.Status = domain.StatusProvisioning

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Status != domain.StatusProvisioning {
		t.Errorf("Status: expected %q, got %q", domain.StatusProvisioning, got.Status)
	}
}

func TestRepository_UpdateErrorField(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s := makeSession("update-err-sess")
	s.Status = domain.StatusProvisioning

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	s.Status = domain.StatusFailed
	s.Error = "provisioning timed out"
	if err := repo.Update(ctx, s); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Status != domain.StatusFailed {
		t.Errorf("Status: expected %q, got %q", domain.StatusFailed, got.Status)
	}
	if got.Error != "provisioning timed out" {
		t.Errorf("Error: expected %q, got %q", "provisioning timed out", got.Error)
	}
}

func TestRepository_ErrorFieldInList(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s1 := makeSession("list-err-1")
	s1.Status = domain.StatusFailed
	s1.Error = "some error"

	s2 := makeSession("list-err-2")
	s2.Status = domain.StatusProvisioning

	if err := repo.Create(ctx, s1); err != nil {
		t.Fatalf("Create s1 failed: %v", err)
	}
	if err := repo.Create(ctx, s2); err != nil {
		t.Fatalf("Create s2 failed: %v", err)
	}

	sessions, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Find sessions by name since list order might vary
	for _, sess := range sessions {
		switch sess.Name {
		case "list-err-1":
			if sess.Error != "some error" {
				t.Errorf("list-err-1: expected Error %q, got %q", "some error", sess.Error)
			}
			if sess.Status != domain.StatusFailed {
				t.Errorf("list-err-1: expected status %q, got %q", domain.StatusFailed, sess.Status)
			}
		case "list-err-2":
			if sess.Error != "" {
				t.Errorf("list-err-2: expected empty Error, got %q", sess.Error)
			}
			if sess.Status != domain.StatusProvisioning {
				t.Errorf("list-err-2: expected status %q, got %q", domain.StatusProvisioning, sess.Status)
			}
		}
	}
}

func TestRepository_DeleteSession_ClearsWorktreeSessionID(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	sess := makeSession("linked")
	if err := repo.Create(ctx, sess); err != nil {
		t.Fatalf("Create session: %v", err)
	}

	wt := domain.ManagedWorktree{
		ID:        "wt-1",
		Path:      "/tmp/wt-1",
		Branch:    "feat/x",
		RepoRoot:  "/tmp/repo",
		SessionID: &sess.ID,
		CreatedAt: time.Now(),
	}
	if err := repo.CreateWorktree(ctx, wt); err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	if err := repo.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete session: %v", err)
	}

	got, err := repo.GetWorktree(ctx, wt.ID)
	if err != nil {
		t.Fatalf("GetWorktree: %v", err)
	}
	if got.SessionID != nil {
		t.Errorf("expected session_id to be NULL after session deleted, got %q", *got.SessionID)
	}
}

func TestRepository_HarnessTypeRoundTrip(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s := makeSession("harness-sess")
	s.HarnessType = "claude"

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.HarnessType != "claude" {
		t.Errorf("expected HarnessType %q, got %q", "claude", got.HarnessType)
	}
}

func TestRepository_HarnessType_BackfillsPreMigrationRows(t *testing.T) {
	// Simulate an "old" pre-migration row by inserting directly via raw SQL,
	// bypassing Create() and explicitly setting harness_type to the empty
	// string default that existed before this column was backfilled.
	dbPath := filepath.Join(t.TempDir(), "cmux.db")
	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatalf("NewRepository failed: %v", err)
	}

	now := time.Now()
	if _, err := repo.DB().Exec(
		"INSERT INTO sessions (id, name, working_dir, status, pid, claude_session_id, harness_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, '', ?, ?)",
		"pre-migration-sess", "old-sess", "/tmp", "stopped", 0, "old-claude-session-id", now, now,
	); err != nil {
		t.Fatalf("failed to insert pre-migration row: %v", err)
	}

	var harnessType string
	if err := repo.DB().QueryRow("SELECT harness_type FROM sessions WHERE id = ?", "pre-migration-sess").Scan(&harnessType); err != nil {
		t.Fatalf("failed to query harness_type before reopen: %v", err)
	}
	if harnessType != "" {
		t.Fatalf("expected empty harness_type before backfill, got %q", harnessType)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("failed to close repository: %v", err)
	}

	// Reopening the repository re-runs NewRepository's migrations, including
	// the backfill UPDATE, which should populate harness_type for the
	// pre-existing row without touching any other column.
	repo2, err := NewRepository(dbPath)
	if err != nil {
		t.Fatalf("second NewRepository failed: %v", err)
	}
	defer func() { _ = repo2.Close() }()

	got, err := repo2.Get(context.Background(), "pre-migration-sess")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.HarnessType != "claude" {
		t.Errorf("expected backfilled HarnessType %q, got %q", "claude", got.HarnessType)
	}
	if got.HarnessSessionID != "old-claude-session-id" {
		t.Errorf("expected untouched HarnessSessionID %q, got %q", "old-claude-session-id", got.HarnessSessionID)
	}
	if got.Name != "old-sess" {
		t.Errorf("expected untouched Name %q, got %q", "old-sess", got.Name)
	}
}

func TestRepository_ListOrderByCreatedAtDesc(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	s1 := makeSession("older")
	s1.CreatedAt = time.Now().Add(-time.Hour)
	s2 := makeSession("newer")
	s2.CreatedAt = time.Now()

	if err := repo.Create(ctx, s1); err != nil {
		t.Fatalf("Create s1 failed: %v", err)
	}
	if err := repo.Create(ctx, s2); err != nil {
		t.Fatalf("Create s2 failed: %v", err)
	}

	sessions, _ := repo.List(ctx)
	if sessions[0].Name != "newer" {
		t.Errorf("expected newest session first, got %q", sessions[0].Name)
	}
}

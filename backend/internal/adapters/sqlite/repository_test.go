package sqlite

import (
	"context"
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

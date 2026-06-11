package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
)

func makeWorktree(id, path string) domain.ManagedWorktree {
	return domain.ManagedWorktree{
		ID:        id,
		Path:      path,
		Branch:    "main",
		RepoRoot:  "/repo",
		CreatedAt: time.Now(),
	}
}

func TestWorktreeRepo_CreateAndGet(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	if err := wtr.Create(ctx, wt); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := wtr.Get(ctx, wt.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != wt.ID || got.Path != wt.Path {
		t.Errorf("expected %+v, got %+v", wt, got)
	}
}

func TestWorktreeRepo_GetNotFound(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	_, err := wtr.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent worktree")
	}
}

func TestWorktreeRepo_GetByPath(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/mypath")
	_ = wtr.Create(ctx, wt)

	got, err := wtr.GetByPath(ctx, "/tmp/mypath")
	if err != nil {
		t.Fatalf("GetByPath failed: %v", err)
	}
	if got.ID != wt.ID {
		t.Errorf("expected ID %q, got %q", wt.ID, got.ID)
	}
}

func TestWorktreeRepo_CreateIdempotent(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	if err := wtr.Create(ctx, wt); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	// Second create with same path should be a no-op (INSERT OR IGNORE)
	if err := wtr.Create(ctx, wt); err != nil {
		t.Fatalf("second Create (idempotent) failed: %v", err)
	}

	wts, _ := wtr.List(ctx)
	if len(wts) != 1 {
		t.Errorf("expected 1 worktree after idempotent insert, got %d", len(wts))
	}
}

func TestWorktreeRepo_List(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	_ = wtr.Create(ctx, makeWorktree("wt-1", "/tmp/wt1"))
	_ = wtr.Create(ctx, makeWorktree("wt-2", "/tmp/wt2"))

	wts, err := wtr.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(wts) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(wts))
	}
}

func TestWorktreeRepo_Delete(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	_ = wtr.Create(ctx, wt)

	if err := wtr.Delete(ctx, wt.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := wtr.Get(ctx, wt.ID); err == nil {
		t.Error("expected error after deletion")
	}
}

func TestWorktreeRepo_DeleteByPath(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/delpath")
	_ = wtr.Create(ctx, wt)

	if err := wtr.DeleteByPath(ctx, "/tmp/delpath"); err != nil {
		t.Fatalf("DeleteByPath failed: %v", err)
	}
	if _, err := wtr.Get(ctx, wt.ID); err == nil {
		t.Error("expected error after DeleteByPath")
	}
}

func TestWorktreeRepo_LinkAndListSessions(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	_ = wtr.Create(ctx, wt)

	// Create sessions in the sessions table so the FK is satisfied
	s1 := makeSession("s1")
	s2 := makeSession("s2")
	_ = repo.Create(ctx, s1)
	_ = repo.Create(ctx, s2)

	if err := wtr.LinkSession(ctx, wt.ID, s1.ID); err != nil {
		t.Fatalf("LinkSession s1 failed: %v", err)
	}
	if err := wtr.LinkSession(ctx, wt.ID, s2.ID); err != nil {
		t.Fatalf("LinkSession s2 failed: %v", err)
	}

	ids, err := wtr.ListSessionIDs(ctx, wt.ID)
	if err != nil {
		t.Fatalf("ListSessionIDs failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 session IDs, got %d", len(ids))
	}
}

func TestWorktreeRepo_LinkSession_Idempotent(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	_ = wtr.Create(ctx, wt)
	s := makeSession("s1")
	_ = repo.Create(ctx, s)

	_ = wtr.LinkSession(ctx, wt.ID, s.ID)
	if err := wtr.LinkSession(ctx, wt.ID, s.ID); err != nil {
		t.Fatalf("duplicate LinkSession should be idempotent, got: %v", err)
	}

	ids, _ := wtr.ListSessionIDs(ctx, wt.ID)
	if len(ids) != 1 {
		t.Errorf("expected 1 session ID after idempotent link, got %d", len(ids))
	}
}

func TestWorktreeRepo_UnlinkSession(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	_ = wtr.Create(ctx, wt)
	s := makeSession("s1")
	_ = repo.Create(ctx, s)
	_ = wtr.LinkSession(ctx, wt.ID, s.ID)

	if err := wtr.UnlinkSession(ctx, s.ID); err != nil {
		t.Fatalf("UnlinkSession failed: %v", err)
	}

	ids, _ := wtr.ListSessionIDs(ctx, wt.ID)
	if len(ids) != 0 {
		t.Errorf("expected 0 session IDs after unlink, got %d", len(ids))
	}
}

func TestWorktreeRepo_Delete_CascadesJunction(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	_ = wtr.Create(ctx, wt)
	s := makeSession("s1")
	_ = repo.Create(ctx, s)
	_ = wtr.LinkSession(ctx, wt.ID, s.ID)

	_ = wtr.Delete(ctx, wt.ID)

	// Junction rows should have cascaded
	ids, _ := wtr.ListSessionIDs(ctx, wt.ID)
	if len(ids) != 0 {
		t.Errorf("expected junction rows to cascade-delete, got %d", len(ids))
	}
}

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

func TestWorktreeRepo_SetAndClearSession(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	_ = wtr.Create(ctx, wt)

	// Create a session so the FK is satisfied
	s := makeSession("s1")
	_ = repo.Create(ctx, s)

	// Set session
	if err := wtr.SetSession(ctx, wt.ID, &s.ID); err != nil {
		t.Fatalf("SetSession failed: %v", err)
	}

	got, err := wtr.GetByPath(ctx, "/tmp/wt1")
	if err != nil {
		t.Fatalf("GetByPath failed: %v", err)
	}
	if got.SessionID == nil || *got.SessionID != s.ID {
		t.Errorf("expected SessionID %q, got %v", s.ID, got.SessionID)
	}

	// Clear session
	if err := wtr.SetSession(ctx, wt.ID, nil); err != nil {
		t.Fatalf("SetSession(nil) failed: %v", err)
	}

	got, err = wtr.GetByPath(ctx, "/tmp/wt1")
	if err != nil {
		t.Fatalf("GetByPath after clear failed: %v", err)
	}
	if got.SessionID != nil {
		t.Errorf("expected SessionID to be nil after clear, got %v", got.SessionID)
	}
}

func TestWorktreeRepo_StatusRoundTrip(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	// Create without explicitly setting Status — should default to "ready".
	wt := makeWorktree("wt-1", "/tmp/wt1")
	if err := wtr.Create(ctx, wt); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := wtr.Get(ctx, wt.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Status != domain.WorktreeStatusReady {
		t.Errorf("expected default status %q, got %q", domain.WorktreeStatusReady, got.Status)
	}
}

func TestWorktreeRepo_SetStatus(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	if err := wtr.Create(ctx, wt); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := wtr.SetStatus(ctx, wt.ID, domain.WorktreeStatusDeleting); err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}

	got, err := wtr.Get(ctx, wt.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Status != domain.WorktreeStatusDeleting {
		t.Errorf("expected status %q, got %q", domain.WorktreeStatusDeleting, got.Status)
	}
}

func TestWorktreeRepo_List_IncludesStatus(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	_ = wtr.Create(ctx, makeWorktree("wt-1", "/tmp/wt1"))
	_ = wtr.Create(ctx, makeWorktree("wt-2", "/tmp/wt2"))
	if err := wtr.SetStatus(ctx, "wt-2", domain.WorktreeStatusDeleting); err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}

	wts, err := wtr.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	statuses := map[string]domain.WorktreeStatus{}
	for _, wt := range wts {
		statuses[wt.ID] = wt.Status
	}
	if statuses["wt-1"] != domain.WorktreeStatusReady {
		t.Errorf("expected wt-1 status %q, got %q", domain.WorktreeStatusReady, statuses["wt-1"])
	}
	if statuses["wt-2"] != domain.WorktreeStatusDeleting {
		t.Errorf("expected wt-2 status %q, got %q", domain.WorktreeStatusDeleting, statuses["wt-2"])
	}
}

func TestWorktreeRepo_StatusExplicitOnCreate(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	wt := makeWorktree("wt-1", "/tmp/wt1")
	wt.Status = domain.WorktreeStatusDeleting
	if err := wtr.Create(ctx, wt); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := wtr.Get(ctx, wt.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Status != domain.WorktreeStatusDeleting {
		t.Errorf("expected status %q, got %q", domain.WorktreeStatusDeleting, got.Status)
	}
}

func TestWorktreeRepo_SessionID_RoundTrip(t *testing.T) {
	repo := setupTestRepo(t)
	wtr := NewWorktreeRepository(repo)
	ctx := context.Background()

	// Create a session to satisfy the FK
	s := makeSession("s-rt")
	_ = repo.Create(ctx, s)

	// Create a worktree with non-nil SessionID
	sessionID := s.ID
	wt := domain.ManagedWorktree{
		ID:        "wt-rt",
		Path:      "/tmp/wt-rt",
		Branch:    "feat",
		RepoRoot:  "/repo",
		SessionID: &sessionID,
		CreatedAt: time.Now(),
	}
	if err := wtr.Create(ctx, wt); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := wtr.Get(ctx, wt.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.SessionID == nil || *got.SessionID != sessionID {
		t.Errorf("expected SessionID %q, got %v", sessionID, got.SessionID)
	}
}

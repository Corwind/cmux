package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/go-chi/chi/v5"
)

// mockWorktreeService implements the worktreeService seam.
type mockWorktreeService struct {
	listFn   func(ctx context.Context) ([]domain.WorktreeEntry, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockWorktreeService) ListWorktrees(ctx context.Context) ([]domain.WorktreeEntry, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *mockWorktreeService) DeleteWorktree(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// deleteRequest builds a DELETE request with the chi "id" URL param populated.
func deleteRequest(id string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, "/api/worktrees/"+id, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestWorktreeHandler_Delete_Returns202(t *testing.T) {
	var gotID string
	h := &worktreeHandler{service: &mockWorktreeService{
		deleteFn: func(ctx context.Context, id string) error {
			gotID = id
			return nil
		},
	}}

	w := httptest.NewRecorder()
	h.Delete(w, deleteRequest("wt-1"))

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	if gotID != "wt-1" {
		t.Errorf("expected service called with id wt-1, got %q", gotID)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if body["worktree_id"] != "wt-1" {
		t.Errorf("expected worktree_id=wt-1, got %q", body["worktree_id"])
	}
}

func TestWorktreeHandler_Delete_Returns409OnLinkedSession(t *testing.T) {
	h := &worktreeHandler{service: &mockWorktreeService{
		deleteFn: func(ctx context.Context, id string) error {
			sid := "sess-1"
			return &app.ErrWorktreeHasSession{WorktreeID: id, SessionID: sid}
		},
	}}

	w := httptest.NewRecorder()
	h.Delete(w, deleteRequest("wt-1"))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWorktreeHandler_Delete_Returns500OnServiceError(t *testing.T) {
	h := &worktreeHandler{service: &mockWorktreeService{
		deleteFn: func(ctx context.Context, id string) error {
			return fmt.Errorf("boom")
		},
	}}

	w := httptest.NewRecorder()
	h.Delete(w, deleteRequest("wt-1"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWorktreeHandler_List_IncludesStatus(t *testing.T) {
	h := &worktreeHandler{service: &mockWorktreeService{
		listFn: func(ctx context.Context) ([]domain.WorktreeEntry, error) {
			return []domain.WorktreeEntry{
				{ManagedWorktree: domain.ManagedWorktree{ID: "wt-1", Path: "/tmp/a", Branch: "main", RepoRoot: "/repo", Status: domain.WorktreeStatusReady}},
				{ManagedWorktree: domain.ManagedWorktree{ID: "wt-2", Path: "/tmp/b", Branch: "feat", RepoRoot: "/repo", Status: domain.WorktreeStatusDeleting}},
			}, nil
		},
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/worktrees", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []worktreeEntryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp))
	}
	if resp[0].Status != "ready" {
		t.Errorf("expected first status 'ready', got %q", resp[0].Status)
	}
	if resp[1].Status != "deleting" {
		t.Errorf("expected second status 'deleting', got %q", resp[1].Status)
	}
}

func TestWorktreeHandler_List_ServiceError(t *testing.T) {
	h := &worktreeHandler{service: &mockWorktreeService{
		listFn: func(ctx context.Context) ([]domain.WorktreeEntry, error) {
			return nil, fmt.Errorf("db down")
		},
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/worktrees", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

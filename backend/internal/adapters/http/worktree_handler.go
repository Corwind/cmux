package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/go-chi/chi/v5"
)

// worktreeService is the seam the handler depends on. *app.SessionService
// satisfies it; tests provide a lightweight mock.
type worktreeService interface {
	ListWorktrees(ctx context.Context) ([]domain.WorktreeEntry, error)
	DeleteWorktree(ctx context.Context, id string) error
}

type worktreeHandler struct {
	service worktreeService
}

func NewWorktreeHandler(service *app.SessionService) *worktreeHandler {
	return &worktreeHandler{service: service}
}

type worktreeEntryResponse struct {
	ID            string    `json:"id"`
	Path          string    `json:"path"`
	Branch        string    `json:"branch"`
	RepoRoot      string    `json:"repo_root"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	SessionID     *string   `json:"session_id,omitempty"`
	SessionName   *string   `json:"session_name,omitempty"`
	SessionStatus *string   `json:"session_status,omitempty"`
}

func toWorktreeResponse(e domain.WorktreeEntry) worktreeEntryResponse {
	return worktreeEntryResponse{
		ID:            e.ID,
		Path:          e.Path,
		Branch:        e.Branch,
		RepoRoot:      e.RepoRoot,
		Status:        string(e.Status),
		CreatedAt:     e.CreatedAt,
		SessionID:     e.SessionID,
		SessionName:   e.SessionName,
		SessionStatus: e.SessionStatus,
	}
}

func (h *worktreeHandler) List(w http.ResponseWriter, r *http.Request) {
	entries, err := h.service.ListWorktrees(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]worktreeEntryResponse, 0, len(entries))
	for _, e := range entries {
		resp = append(resp, toWorktreeResponse(e))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *worktreeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.service.DeleteWorktree(r.Context(), id)
	if err != nil {
		if _, ok := err.(*app.ErrWorktreeHasSession); ok {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"worktree_id": id})
}

package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/go-chi/chi/v5"
)

type worktreeHandler struct {
	service *app.SessionService
}

func NewWorktreeHandler(service *app.SessionService) *worktreeHandler {
	return &worktreeHandler{service: service}
}

type worktreeSessionResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type worktreeEntryResponse struct {
	ID        string                    `json:"id"`
	Path      string                    `json:"path"`
	Branch    string                    `json:"branch"`
	RepoRoot  string                    `json:"repo_root"`
	CreatedAt time.Time                 `json:"created_at"`
	Sessions  []worktreeSessionResponse `json:"sessions"`
}

func toWorktreeResponse(e domain.WorktreeEntry) worktreeEntryResponse {
	sessions := make([]worktreeSessionResponse, 0, len(e.Sessions))
	for _, s := range e.Sessions {
		sessions = append(sessions, worktreeSessionResponse{
			ID:     s.ID,
			Name:   s.Name,
			Status: string(s.Status),
		})
	}
	return worktreeEntryResponse{
		ID:        e.ID,
		Path:      e.Path,
		Branch:    e.Branch,
		RepoRoot:  e.RepoRoot,
		CreatedAt: e.CreatedAt,
		Sessions:  sessions,
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
	force := r.URL.Query().Get("force") == "true"

	err := h.service.DeleteOrphanedWorktree(r.Context(), id, force)
	if err != nil {
		switch err.(type) {
		case *app.ErrWorktreeHasSessions:
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

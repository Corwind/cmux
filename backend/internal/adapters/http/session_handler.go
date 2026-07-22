package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/go-chi/chi/v5"
)

func toSessionResponse(s domain.Session) sessionResponse {
	return sessionResponse{
		ID:              s.ID,
		Name:            s.Name,
		WorkingDir:      s.WorkingDir,
		Status:          string(s.Status),
		PID:             s.PID,
		TemplateID:      s.TemplateID,
		SkipPermissions: s.SkipPermissions,
		RepoRoot:        s.RepoRoot,
		GitBranch:       s.GitBranch,
		WorktreeManaged: s.WorktreeManaged,
		HarnessType:     s.HarnessType,
		ErrorMessage:    s.Error,
		CreatedAt:       s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

type SessionHandler struct {
	service *app.SessionService
}

func NewSessionHandler(service *app.SessionService) *SessionHandler {
	return &SessionHandler{service: service}
}

type worktreeRequest struct {
	RepoPath     string `json:"repo_path"`
	Branch       string `json:"branch"`
	BaseRef      string `json:"base_ref,omitempty"`
	CreateBranch bool   `json:"create_branch"`
	Path         string `json:"path,omitempty"`
}

type createSessionRequest struct {
	Name            string           `json:"name"`
	WorkingDir      string           `json:"working_dir"`
	TemplateID      string           `json:"template_id"`
	SkipPermissions bool             `json:"skip_permissions"`
	Worktree        *worktreeRequest `json:"worktree,omitempty"`
	HarnessType     string           `json:"harness_type,omitempty"`
}

type sessionResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	WorkingDir      string `json:"working_dir"`
	Status          string `json:"status"`
	PID             int    `json:"pid"`
	TemplateID      string `json:"template_id"`
	SkipPermissions bool   `json:"skip_permissions"`
	RepoRoot        string `json:"repo_root,omitempty"`
	GitBranch       string `json:"git_branch,omitempty"`
	WorktreeManaged bool   `json:"worktree_managed,omitempty"`
	HarnessType     string `json:"harness_type"`
	ErrorMessage    string `json:"error_message,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input := app.CreateSessionInput{
		Name:            req.Name,
		WorkingDir:      req.WorkingDir,
		TemplateID:      req.TemplateID,
		SkipPermissions: req.SkipPermissions,
		HarnessType:     req.HarnessType,
	}
	if req.Worktree != nil {
		input.Worktree = &app.WorktreeSpec{
			RepoPath:     req.Worktree.RepoPath,
			Branch:       req.Worktree.Branch,
			BaseRef:      req.Worktree.BaseRef,
			CreateBranch: req.Worktree.CreateBranch,
			Path:         req.Worktree.Path,
		}
	}

	session, err := h.service.CreateSession(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(toSessionResponse(session)); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}

func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.service.ListSessions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []sessionResponse
	for _, s := range sessions {
		resp = append(resp, toSessionResponse(s))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}

func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	session, err := h.service.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toSessionResponse(session)); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}

func (h *SessionHandler) Resume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	session, err := h.service.ResumeSession(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toSessionResponse(session)); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}

func (h *SessionHandler) Restart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	session, err := h.service.RestartSession(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toSessionResponse(session)); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}

func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.service.DeleteSession(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

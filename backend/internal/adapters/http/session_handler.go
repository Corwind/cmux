package http

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/go-chi/chi/v5"
)

type SessionHandler struct {
	service *app.SessionService
}

func NewSessionHandler(service *app.SessionService) *SessionHandler {
	return &SessionHandler{service: service}
}

type createSessionRequest struct {
	Name            string `json:"name"`
	WorkingDir      string `json:"working_dir"`
	TemplateID      string `json:"template_id"`
	SkipPermissions bool   `json:"skip_permissions"`
}

type sessionResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	WorkingDir string `json:"working_dir"`
	Status     string `json:"status"`
	PID        int    `json:"pid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	session, err := h.service.CreateSession(r.Context(), req.Name, req.WorkingDir, req.TemplateID, req.SkipPermissions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(sessionResponse{
		ID:         session.ID,
		Name:       session.Name,
		WorkingDir: session.WorkingDir,
		Status:     string(session.Status),
		PID:        session.PID,
		CreatedAt:  session.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  session.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}); err != nil {
		log.Printf("failed to encode response: %v", err)
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
		resp = append(resp, sessionResponse{
			ID:         s.ID,
			Name:       s.Name,
			WorkingDir: s.WorkingDir,
			Status:     string(s.Status),
			PID:        s.PID,
			CreatedAt:  s.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:  s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("failed to encode response: %v", err)
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
	if err := json.NewEncoder(w).Encode(sessionResponse{
		ID:         session.ID,
		Name:       session.Name,
		WorkingDir: session.WorkingDir,
		Status:     string(session.Status),
		PID:        session.PID,
		CreatedAt:  session.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  session.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}); err != nil {
		log.Printf("failed to encode response: %v", err)
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
	if err := json.NewEncoder(w).Encode(sessionResponse{
		ID:         session.ID,
		Name:       session.Name,
		WorkingDir: session.WorkingDir,
		Status:     string(session.Status),
		PID:        session.PID,
		CreatedAt:  session.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  session.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}); err != nil {
		log.Printf("failed to encode response: %v", err)
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

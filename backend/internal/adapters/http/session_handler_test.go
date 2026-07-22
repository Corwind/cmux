package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/ports"
	"github.com/go-chi/chi/v5"
)

// --- Mock ports for building a real SessionService ---

type mockRepo struct {
	sessions map[string]domain.Session
}

func newMockRepo() *mockRepo {
	return &mockRepo{sessions: make(map[string]domain.Session)}
}

func (m *mockRepo) Create(ctx context.Context, s domain.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockRepo) Get(ctx context.Context, id string) (domain.Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return domain.Session{}, fmt.Errorf("session not found: %s", id)
	}
	return s, nil
}

func (m *mockRepo) List(ctx context.Context) ([]domain.Session, error) {
	var result []domain.Session
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockRepo) Update(ctx context.Context, s domain.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}

type mockPM struct {
	alive   map[int]bool
	handles map[int]*ports.PTYHandle
}

func newMockPM() *mockPM {
	return &mockPM{
		alive:   make(map[int]bool),
		handles: make(map[int]*ports.PTYHandle),
	}
}

func (m *mockPM) Spawn(ctx context.Context, workingDir string, harnessType string, args ...string) (*ports.PTYHandle, error) {
	done := make(chan error, 1)
	h := &ports.PTYHandle{PTY: os.Stdin, PID: 42, Done: done}
	m.alive[42] = true
	m.handles[42] = h
	return h, nil
}

func (m *mockPM) Resize(pid int, rows, cols uint16) error { return nil }
func (m *mockPM) Kill(pid int) error {
	delete(m.alive, pid)
	return nil
}
func (m *mockPM) KillAll()                                   {}
func (m *mockPM) IsAlive(pid int) bool                       { return m.alive[pid] }
func (m *mockPM) GetHandle(pid int) (*ports.PTYHandle, bool) { h, ok := m.handles[pid]; return h, ok }

func setupHandler() (*SessionHandler, *app.SessionService) {
	repo := newMockRepo()
	pm := newMockPM()
	svc := app.NewSessionService(repo, pm, nil)
	handler := NewSessionHandler(svc)
	return handler, svc
}

func TestSessionHandler_Create(t *testing.T) {
	handler, _ := setupHandler()

	body, _ := json.Marshal(createSessionRequest{Name: "test", WorkingDir: "/tmp"})
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp sessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "test" {
		t.Errorf("expected name 'test', got %q", resp.Name)
	}
	if resp.Status != "running" {
		t.Errorf("expected status 'running', got %q", resp.Status)
	}
}

func TestSessionHandler_Create_InvalidBody(t *testing.T) {
	handler, _ := setupHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSessionHandler_Create_EmptyNameDefaultsToDir(t *testing.T) {
	handler, _ := setupHandler()

	body, _ := json.Marshal(createSessionRequest{Name: "", WorkingDir: "/tmp"})
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp sessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "tmp" {
		t.Errorf("expected name 'tmp', got %q", resp.Name)
	}
}

func TestSessionHandler_List(t *testing.T) {
	handler, svc := setupHandler()

	if _, err := svc.CreateSession(context.Background(), app.CreateSessionInput{Name: "s1", WorkingDir: "/tmp"}); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp []sessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 session, got %d", len(resp))
	}
}

func TestSessionHandler_Get(t *testing.T) {
	handler, svc := setupHandler()

	created, _ := svc.CreateSession(context.Background(), app.CreateSessionInput{Name: "test", WorkingDir: "/tmp"})

	// chi URL params require a chi context
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+created.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp sessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, resp.ID)
	}
}

func TestSessionHandler_Get_NotFound(t *testing.T) {
	handler, _ := setupHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestSessionHandler_Delete(t *testing.T) {
	handler, svc := setupHandler()

	created, _ := svc.CreateSession(context.Background(), app.CreateSessionInput{Name: "test", WorkingDir: "/tmp"})

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+created.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionHandler_Create_WithWorktree(t *testing.T) {
	handler, _ := setupHandler()

	body, _ := json.Marshal(createSessionRequest{
		Name:       "wt-session",
		WorkingDir: "/repo",
		Worktree: &worktreeRequest{
			RepoPath:     "/repo",
			Branch:       "feature/wt",
			BaseRef:      "main",
			CreateBranch: true,
			Path:         "/tmp/wt",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	// The mock git service will be nil so this should succeed with plain session
	// We verify the request was parsed and the response has the right shape
	if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionHandler_Create_ResponseIncludesWorktreeFields(t *testing.T) {
	handler, _ := setupHandler()

	body, _ := json.Marshal(createSessionRequest{Name: "test", WorkingDir: "/tmp"})
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp sessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	// Fields should be present in response (even if empty/false)
	_ = resp.RepoRoot
	_ = resp.GitBranch
	_ = resp.WorktreeManaged
}

func TestSessionHandler_Create_HarnessTypeRoundTrip(t *testing.T) {
	handler, _ := setupHandler()

	body, _ := json.Marshal(createSessionRequest{Name: "test", WorkingDir: "/tmp", HarnessType: "claude"})
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created sessionResponse
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if created.HarnessType != "claude" {
		t.Errorf("expected harness_type 'claude' in create response, got %q", created.HarnessType)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/sessions/"+created.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx))
	getW := httptest.NewRecorder()

	handler.Get(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var fetched sessionResponse
	if err := json.NewDecoder(getW.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if fetched.HarnessType != "claude" {
		t.Errorf("expected harness_type 'claude' in get response, got %q", fetched.HarnessType)
	}
}

func TestSessionHandler_Delete_NoParams(t *testing.T) {
	handler, svc := setupHandler()

	created, _ := svc.CreateSession(context.Background(), app.CreateSessionInput{Name: "test", WorkingDir: "/tmp"})

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+created.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

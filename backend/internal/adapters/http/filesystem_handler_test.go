package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Corwind/cmux/backend/internal/ports"
)

// --- Mock FileBrowser ---

type mockFileBrowser struct {
	entries    []ports.DirEntry
	listErr    error
	homeDir    string
	homeDirErr error
}

func (m *mockFileBrowser) ListDir(path string) ([]ports.DirEntry, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.entries, nil
}

func (m *mockFileBrowser) HomeDir() (string, error) {
	if m.homeDirErr != nil {
		return "", m.homeDirErr
	}
	return m.homeDir, nil
}

func TestFilesystemHandler_ListDirectory_WithPath(t *testing.T) {
	browser := &mockFileBrowser{
		entries: []ports.DirEntry{
			{Name: "dir1", IsDir: true},
			{Name: "file1.txt", IsDir: false},
		},
	}
	handler := NewFilesystemHandler(browser)

	req := httptest.NewRequest(http.MethodGet, "/api/fs?path=/tmp", nil)
	w := httptest.NewRecorder()

	handler.ListDirectory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp listDirResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Path != "/tmp" {
		t.Errorf("expected path '/tmp', got %q", resp.Path)
	}
	if len(resp.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(resp.Entries))
	}
}

func TestFilesystemHandler_ListDirectory_DefaultHome(t *testing.T) {
	browser := &mockFileBrowser{
		homeDir: "/home/user",
		entries: []ports.DirEntry{
			{Name: "Documents", IsDir: true},
		},
	}
	handler := NewFilesystemHandler(browser)

	req := httptest.NewRequest(http.MethodGet, "/api/fs", nil)
	w := httptest.NewRecorder()

	handler.ListDirectory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp listDirResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Path != "/home/user" {
		t.Errorf("expected path '/home/user', got %q", resp.Path)
	}
}

func TestFilesystemHandler_ListDirectory_HomeDirError(t *testing.T) {
	browser := &mockFileBrowser{
		homeDirErr: fmt.Errorf("no home"),
	}
	handler := NewFilesystemHandler(browser)

	req := httptest.NewRequest(http.MethodGet, "/api/fs", nil)
	w := httptest.NewRecorder()

	handler.ListDirectory(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestFilesystemHandler_ListDirectory_ListDirError(t *testing.T) {
	browser := &mockFileBrowser{
		listErr: fmt.Errorf("permission denied"),
	}
	handler := NewFilesystemHandler(browser)

	req := httptest.NewRequest(http.MethodGet, "/api/fs?path=/root", nil)
	w := httptest.NewRecorder()

	handler.ListDirectory(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestFilesystemHandler_ListDirectory_EmptyDir(t *testing.T) {
	browser := &mockFileBrowser{
		entries: []ports.DirEntry{},
	}
	handler := NewFilesystemHandler(browser)

	req := httptest.NewRequest(http.MethodGet, "/api/fs?path=/empty", nil)
	w := httptest.NewRecorder()

	handler.ListDirectory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp listDirResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Entries != nil && len(resp.Entries) != 0 {
		t.Errorf("expected nil or empty entries, got %d", len(resp.Entries))
	}
}

package http

import (
	"encoding/json"
	"net/http"

	"github.com/Corwind/cmux/backend/internal/ports"
)

type FilesystemHandler struct {
	browser ports.FileBrowser
}

func NewFilesystemHandler(browser ports.FileBrowser) *FilesystemHandler {
	return &FilesystemHandler{browser: browser}
}

type dirEntryResponse struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
}

type listDirResponse struct {
	Path    string             `json:"path"`
	Entries []dirEntryResponse `json:"entries"`
}

func (h *FilesystemHandler) ListDirectory(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		home, err := h.browser.HomeDir()
		if err != nil {
			http.Error(w, "failed to get home directory", http.StatusInternalServerError)
			return
		}
		path = home
	}

	entries, err := h.browser.ListDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resp []dirEntryResponse
	for _, e := range entries {
		resp = append(resp, dirEntryResponse{
			Name:  e.Name,
			IsDir: e.IsDir,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listDirResponse{
		Path:    path,
		Entries: resp,
	})
}

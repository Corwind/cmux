package http

import (
	"net/http"
	"net/url"
	"os/exec"
)

type openHandler struct{}

func NewOpenHandler() *openHandler {
	return &openHandler{}
}

// Handle serves POST /api/open — called by the sandboxed `open` wrapper script.
// It validates that the URL is http/https and hands it off to the host browser
// via the real macOS `open` binary, which runs outside the sandbox.
func (h *openHandler) Handle(w http.ResponseWriter, r *http.Request) {
	rawURL := r.FormValue("url")
	if rawURL == "" {
		http.Error(w, "url required", http.StatusBadRequest)
		return
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		http.Error(w, "only http and https URLs are allowed", http.StatusBadRequest)
		return
	}
	if err := exec.Command("open", rawURL).Run(); err != nil {
		http.Error(w, "failed to open URL", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

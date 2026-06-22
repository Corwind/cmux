package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Corwind/cmux/backend/internal/ports"
)

type gitWorktreeResponse struct {
	Path     string `json:"path"`
	Branch   string `json:"branch"`
	Head     string `json:"head"`
	IsMain   bool   `json:"is_main"`
	Detached bool   `json:"detached"`
	Locked   bool   `json:"locked"`
}

type gitBranchResponse struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
	IsRemote  bool   `json:"is_remote"`
}

type gitInfoResponse struct {
	IsRepo        bool                  `json:"is_repo"`
	RepoRoot      string                `json:"repo_root,omitempty"`
	CurrentBranch string                `json:"current_branch,omitempty"`
	Worktrees     []gitWorktreeResponse `json:"worktrees,omitempty"`
	Branches      []gitBranchResponse   `json:"branches,omitempty"`
}

type GitHandler struct {
	gitService ports.GitService
}

func NewGitHandler(gitService ports.GitService) *GitHandler {
	return &GitHandler{gitService: gitService}
}

func (h *GitHandler) Info(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "missing path query parameter", http.StatusBadRequest)
		return
	}

	info, err := h.gitService.Info(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := gitInfoResponse{
		IsRepo:        info.IsRepo,
		RepoRoot:      info.RepoRoot,
		CurrentBranch: info.CurrentBranch,
	}
	for _, wt := range info.Worktrees {
		resp.Worktrees = append(resp.Worktrees, gitWorktreeResponse{
			Path:     wt.Path,
			Branch:   wt.Branch,
			Head:     wt.Head,
			IsMain:   wt.IsMain,
			Detached: wt.Detached,
			Locked:   wt.Locked,
		})
	}
	for _, b := range info.Branches {
		resp.Branches = append(resp.Branches, gitBranchResponse{
			Name:      b.Name,
			IsCurrent: b.IsCurrent,
			IsRemote:  b.IsRemote,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode git info response", "err", err)
	}
}

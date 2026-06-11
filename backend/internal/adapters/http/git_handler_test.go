package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Corwind/cmux/backend/internal/ports"
)

type mockGitService struct {
	infoFn func(path string) (ports.GitInfo, error)
}

func (m *mockGitService) Info(path string) (ports.GitInfo, error) {
	if m.infoFn != nil {
		return m.infoFn(path)
	}
	return ports.GitInfo{IsRepo: false}, nil
}

func (m *mockGitService) AddWorktree(repoRoot, wtPath, branch, baseRef string, create bool) (ports.Worktree, error) {
	return ports.Worktree{}, nil
}

func (m *mockGitService) RemoveWorktree(repoRoot, wtPath string, force bool) error {
	return nil
}

func (m *mockGitService) IsClean(path string) (bool, error) {
	return true, nil
}

func TestGitHandler_Info_MissingPath(t *testing.T) {
	handler := NewGitHandler(&mockGitService{})

	req := httptest.NewRequest(http.MethodGet, "/api/git/info", nil)
	w := httptest.NewRecorder()

	handler.Info(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGitHandler_Info_NotRepo(t *testing.T) {
	handler := NewGitHandler(&mockGitService{
		infoFn: func(path string) (ports.GitInfo, error) {
			return ports.GitInfo{IsRepo: false}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/git/info?path=/tmp/notrepo", nil)
	w := httptest.NewRecorder()

	handler.Info(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp gitInfoResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.IsRepo {
		t.Error("expected is_repo=false")
	}
}

func TestGitHandler_Info_IsRepo(t *testing.T) {
	handler := NewGitHandler(&mockGitService{
		infoFn: func(path string) (ports.GitInfo, error) {
			return ports.GitInfo{
				IsRepo:        true,
				RepoRoot:      path,
				CurrentBranch: "main",
				Branches: []ports.Branch{
					{Name: "main", IsCurrent: true},
					{Name: "feature/foo"},
				},
				Worktrees: []ports.Worktree{
					{Path: path, Branch: "main", IsMain: true},
				},
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/git/info?path=/Users/user/repo", nil)
	w := httptest.NewRecorder()

	handler.Info(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp gitInfoResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !resp.IsRepo {
		t.Error("expected is_repo=true")
	}
	if resp.CurrentBranch != "main" {
		t.Errorf("expected current_branch=main, got %q", resp.CurrentBranch)
	}
	if len(resp.Branches) != 2 {
		t.Errorf("expected 2 branches, got %d", len(resp.Branches))
	}
	if len(resp.Worktrees) != 1 {
		t.Errorf("expected 1 worktree, got %d", len(resp.Worktrees))
	}
	if !resp.Worktrees[0].IsMain {
		t.Error("expected first worktree to be main")
	}
}

func TestGitHandler_Info_ServiceError(t *testing.T) {
	handler := NewGitHandler(&mockGitService{
		infoFn: func(path string) (ports.GitInfo, error) {
			return ports.GitInfo{}, fmt.Errorf("git error")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/git/info?path=/repo", nil)
	w := httptest.NewRecorder()

	handler.Info(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

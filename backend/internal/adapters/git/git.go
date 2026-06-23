package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Corwind/cmux/backend/internal/ports"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Info(path string) (ports.GitInfo, error) {
	ctx := context.Background()
	// Check if inside a work tree
	out, err := runGit(ctx, path, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		return ports.GitInfo{IsRepo: false}, nil
	}

	// Get repo root
	rootOut, err := runGit(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil {
		return ports.GitInfo{IsRepo: false}, nil
	}
	repoRoot := strings.TrimSpace(string(rootOut))

	// Get current branch
	branchOut, err := runGit(ctx, path, "rev-parse", "--abbrev-ref", "HEAD")
	currentBranch := ""
	if err == nil {
		currentBranch = strings.TrimSpace(string(branchOut))
	}

	// List worktrees
	worktrees, err := listWorktrees(ctx, repoRoot)
	if err != nil {
		worktrees = nil
	}

	// List local branches
	branches, err := listBranches(ctx, repoRoot, currentBranch)
	if err != nil {
		branches = nil
	}

	return ports.GitInfo{
		IsRepo:        true,
		RepoRoot:      repoRoot,
		CurrentBranch: currentBranch,
		Worktrees:     worktrees,
		Branches:      branches,
	}, nil
}

func listWorktrees(ctx context.Context, repoRoot string) ([]ports.Worktree, error) {
	out, err := runGit(ctx, repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []ports.Worktree
	var current ports.Worktree
	isFirst := true

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = ports.Worktree{Path: strings.TrimPrefix(line, "worktree ")}
			if isFirst {
				current.IsMain = true
				isFirst = false
			}
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			// ref is like refs/heads/main
			if strings.HasPrefix(ref, "refs/heads/") {
				current.Branch = strings.TrimPrefix(ref, "refs/heads/")
			} else {
				current.Branch = ref
			}
		case line == "detached":
			current.Detached = true
		case line == "locked":
			current.Locked = true
		case line == "":
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = ports.Worktree{}
			}
		}
	}
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

func listBranches(ctx context.Context, repoRoot, currentBranch string) ([]ports.Branch, error) {
	out, err := runGit(ctx, repoRoot, "for-each-ref", "--format=%(refname:short) %(HEAD)", "refs/heads")
	if err != nil {
		return nil, err
	}

	var branches []ports.Branch
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		name := parts[0]
		isCurrent := len(parts) == 2 && parts[1] == "*"
		if !isCurrent && name == currentBranch {
			isCurrent = true
		}
		branches = append(branches, ports.Branch{Name: name, IsCurrent: isCurrent, IsRemote: false})
	}

	return branches, nil
}

func (s *Service) AddWorktree(ctx context.Context, repoRoot, worktreePath, branch, baseRef string, createBranch bool) (ports.Worktree, error) {
	if err := validatePath(worktreePath); err != nil {
		return ports.Worktree{}, fmt.Errorf("invalid worktree path: %w", err)
	}
	if err := validateBranchName(branch); err != nil {
		return ports.Worktree{}, fmt.Errorf("invalid branch name: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return ports.Worktree{}, fmt.Errorf("create parent dir: %w", err)
	}

	var args []string
	if createBranch {
		args = []string{"worktree", "add", "-b", branch, worktreePath}
		if baseRef != "" {
			args = append(args, baseRef)
		}
	} else {
		args = []string{"worktree", "add", worktreePath, branch}
	}

	if _, err := runGit(ctx, repoRoot, args...); err != nil {
		return ports.Worktree{}, fmt.Errorf("git worktree add: %w", err)
	}

	// Resolve head after add
	headOut, _ := runGit(ctx, worktreePath, "rev-parse", "HEAD")
	return ports.Worktree{
		Path:   worktreePath,
		Branch: branch,
		Head:   strings.TrimSpace(string(headOut)),
	}, nil
}

func (s *Service) RemoveWorktree(ctx context.Context, repoRoot, worktreePath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)

	if _, err := runGit(ctx, repoRoot, args...); err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}
	return nil
}

func (s *Service) IsClean(ctx context.Context, worktreePath string) (bool, error) {
	out, err := runGit(ctx, worktreePath, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return strings.TrimSpace(string(out)) == "", nil
}

func runGit(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stdin = nil

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return nil, fmt.Errorf("%s", msg)
		}
		return stdout.Bytes(), nil
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return nil, ctx.Err()
	}
}

func validatePath(path string) error {
	if strings.Contains(path, "..") {
		return fmt.Errorf("path %q contains path traversal", path)
	}
	return nil
}

func validateBranchName(branch string) error {
	if strings.Contains(branch, "..") || strings.ContainsAny(branch, "\n\r") {
		return fmt.Errorf("branch name %q is invalid", branch)
	}
	return nil
}

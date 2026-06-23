package ports

import "context"

type Worktree struct {
	Path     string
	Branch   string
	Head     string
	IsMain   bool
	Detached bool
	Locked   bool
}

type Branch struct {
	Name      string
	IsCurrent bool
	IsRemote  bool
}

type GitInfo struct {
	IsRepo        bool
	RepoRoot      string
	CurrentBranch string
	Worktrees     []Worktree
	Branches      []Branch
}

type GitService interface {
	Info(path string) (GitInfo, error)
	AddWorktree(ctx context.Context, repoRoot, worktreePath, branch, baseRef string, createBranch bool) (Worktree, error)
	RemoveWorktree(ctx context.Context, repoRoot, worktreePath string, force bool) error
	IsClean(ctx context.Context, worktreePath string) (bool, error)
}

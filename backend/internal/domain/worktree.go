package domain

import (
	"fmt"
	"time"
)

type ErrWorktreeNotFound string

func (e ErrWorktreeNotFound) Error() string {
	return fmt.Sprintf("worktree not found: %s", string(e))
}

// ManagedWorktree records a git worktree created (or adopted) by cmux.
// It persists independently of sessions so that a worktree remains visible
// in the UI after its associated session is deleted with the "keep" action.
type ManagedWorktree struct {
	ID        string
	Path      string
	Branch    string
	RepoRoot  string
	CreatedAt time.Time
}

// WorktreeEntry pairs a ManagedWorktree with the sessions currently associated
// with it via the worktree_sessions junction table.
type WorktreeEntry struct {
	ManagedWorktree
	Sessions []Session
}

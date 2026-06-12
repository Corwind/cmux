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
// in the UI after its associated session is deleted.
type ManagedWorktree struct {
	ID        string
	Path      string
	Branch    string
	RepoRoot  string
	SessionID *string
	CreatedAt time.Time
}

// WorktreeEntry pairs a ManagedWorktree with optional session info derived
// from the session_id FK column on the worktrees table.
type WorktreeEntry struct {
	ManagedWorktree
	SessionName   *string
	SessionStatus *string
}

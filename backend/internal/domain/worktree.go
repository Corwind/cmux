package domain

import (
	"fmt"
	"time"
)

type ErrWorktreeNotFound string

func (e ErrWorktreeNotFound) Error() string {
	return fmt.Sprintf("worktree not found: %s", string(e))
}

// WorktreeStatus represents the lifecycle state of a managed worktree.
type WorktreeStatus string

const (
	// WorktreeStatusReady indicates the worktree is fully provisioned and
	// usable. It is the default state (the empty zero value is also treated
	// as ready for backwards compatibility).
	WorktreeStatusReady WorktreeStatus = "ready"
	// WorktreeStatusDeleting indicates the worktree is being removed in the
	// background. It is a transient state that disappears once removal
	// completes.
	WorktreeStatusDeleting WorktreeStatus = "deleting"
)

// ManagedWorktree records a git worktree created (or adopted) by cmux.
// It persists independently of sessions so that a worktree remains visible
// in the UI after its associated session is deleted.
type ManagedWorktree struct {
	ID        string
	Path      string
	Branch    string
	RepoRoot  string
	SessionID *string
	Status    WorktreeStatus
	CreatedAt time.Time
}

// WorktreeEntry pairs a ManagedWorktree with optional session info derived
// from the session_id FK column on the worktrees table.
type WorktreeEntry struct {
	ManagedWorktree
	SessionName   *string
	SessionStatus *string
}

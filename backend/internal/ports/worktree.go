package ports

import (
	"context"

	"github.com/Corwind/cmux/backend/internal/domain"
)

type WorktreeRepository interface {
	Create(ctx context.Context, wt domain.ManagedWorktree) error
	List(ctx context.Context) ([]domain.ManagedWorktree, error)
	Get(ctx context.Context, id string) (domain.ManagedWorktree, error)
	GetByPath(ctx context.Context, path string) (domain.ManagedWorktree, error)
	Delete(ctx context.Context, id string) error
	DeleteByPath(ctx context.Context, path string) error
	SetSession(ctx context.Context, worktreeID string, sessionID *string) error
}

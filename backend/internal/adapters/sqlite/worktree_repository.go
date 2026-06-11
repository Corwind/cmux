package sqlite

import (
	"context"

	"github.com/Corwind/cmux/backend/internal/domain"
)

// WorktreeRepo adapts Repository to ports.WorktreeRepository.
type WorktreeRepo struct {
	r *Repository
}

func NewWorktreeRepository(r *Repository) *WorktreeRepo {
	return &WorktreeRepo{r: r}
}

func (w *WorktreeRepo) Create(ctx context.Context, wt domain.ManagedWorktree) error {
	return w.r.CreateWorktree(ctx, wt)
}

func (w *WorktreeRepo) List(ctx context.Context) ([]domain.ManagedWorktree, error) {
	return w.r.ListWorktrees(ctx)
}

func (w *WorktreeRepo) Get(ctx context.Context, id string) (domain.ManagedWorktree, error) {
	return w.r.GetWorktree(ctx, id)
}

func (w *WorktreeRepo) GetByPath(ctx context.Context, path string) (domain.ManagedWorktree, error) {
	return w.r.GetWorktreeByPath(ctx, path)
}

func (w *WorktreeRepo) Delete(ctx context.Context, id string) error {
	return w.r.DeleteWorktree(ctx, id)
}

func (w *WorktreeRepo) DeleteByPath(ctx context.Context, path string) error {
	return w.r.DeleteWorktreeByPath(ctx, path)
}

func (w *WorktreeRepo) LinkSession(ctx context.Context, worktreeID, sessionID string) error {
	return w.r.LinkWorktreeSession(ctx, worktreeID, sessionID)
}

func (w *WorktreeRepo) UnlinkSession(ctx context.Context, sessionID string) error {
	return w.r.UnlinkWorktreeSession(ctx, sessionID)
}

func (w *WorktreeRepo) ListSessionIDs(ctx context.Context, worktreeID string) ([]string, error) {
	return w.r.ListWorktreeSessionIDs(ctx, worktreeID)
}

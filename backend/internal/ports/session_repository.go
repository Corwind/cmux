package ports

import (
	"context"

	"github.com/Corwind/cmux/backend/internal/domain"
)

type SessionRepository interface {
	Create(ctx context.Context, session domain.Session) error
	Get(ctx context.Context, id string) (domain.Session, error)
	List(ctx context.Context) ([]domain.Session, error)
	Update(ctx context.Context, session domain.Session) error
	Delete(ctx context.Context, id string) error
}

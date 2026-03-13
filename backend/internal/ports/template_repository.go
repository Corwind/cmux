package ports

import (
	"context"

	"github.com/Corwind/cmux/backend/internal/domain"
)

type TemplateRepository interface {
	Create(ctx context.Context, tmpl domain.SandboxTemplate) error
	Get(ctx context.Context, id string) (domain.SandboxTemplate, error)
	List(ctx context.Context) ([]domain.SandboxTemplate, error)
	Update(ctx context.Context, tmpl domain.SandboxTemplate) error
	Delete(ctx context.Context, id string) error
	GetDefault(ctx context.Context) (domain.SandboxTemplate, error)
	SetDefault(ctx context.Context, id string) error
	ClearDefault(ctx context.Context) error
}

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Corwind/cmux/backend/internal/domain"
)

type TemplateRepository struct {
	db *sql.DB
}

func NewTemplateRepository(db *sql.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(ctx context.Context, tmpl domain.SandboxTemplate) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO sandbox_templates (id, name, content, is_default, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		tmpl.ID, tmpl.Name, tmpl.Content, tmpl.IsDefault, tmpl.CreatedAt, tmpl.UpdatedAt,
	)
	return err
}

func (r *TemplateRepository) Get(ctx context.Context, id string) (domain.SandboxTemplate, error) {
	var t domain.SandboxTemplate
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, content, is_default, created_at, updated_at FROM sandbox_templates WHERE id = ?", id,
	).Scan(&t.ID, &t.Name, &t.Content, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.SandboxTemplate{}, fmt.Errorf("template not found: %s", id)
	}
	return t, err
}

func (r *TemplateRepository) List(ctx context.Context) ([]domain.SandboxTemplate, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, content, is_default, created_at, updated_at FROM sandbox_templates ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var templates []domain.SandboxTemplate
	for rows.Next() {
		var t domain.SandboxTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Content, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *TemplateRepository) Update(ctx context.Context, tmpl domain.SandboxTemplate) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE sandbox_templates SET name = ?, content = ?, is_default = ?, updated_at = ? WHERE id = ?",
		tmpl.Name, tmpl.Content, tmpl.IsDefault, tmpl.UpdatedAt, tmpl.ID,
	)
	return err
}

func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sandbox_templates WHERE id = ?", id)
	return err
}

func (r *TemplateRepository) GetDefault(ctx context.Context) (domain.SandboxTemplate, error) {
	var t domain.SandboxTemplate
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, content, is_default, created_at, updated_at FROM sandbox_templates WHERE is_default = 1",
	).Scan(&t.ID, &t.Name, &t.Content, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.SandboxTemplate{}, fmt.Errorf("no default template set")
	}
	return t, err
}

func (r *TemplateRepository) SetDefault(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "UPDATE sandbox_templates SET is_default = 0 WHERE is_default = 1"); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, "UPDATE sandbox_templates SET is_default = 1 WHERE id = ?", id); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *TemplateRepository) ClearDefault(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "UPDATE sandbox_templates SET is_default = 0 WHERE is_default = 1")
	return err
}

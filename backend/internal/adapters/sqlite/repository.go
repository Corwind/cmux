package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Corwind/cmux/backend/internal/domain"
	_ "modernc.org/sqlite"
)

func isDuplicateColumnError(err error) bool {
	return strings.Contains(err.Error(), "duplicate column")
}

type Repository struct {
	db *sql.DB
}

func NewRepository(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec(createSessionsTable); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if _, err := db.Exec(createTemplatesTable); err != nil {
		return nil, fmt.Errorf("failed to run template migrations: %w", err)
	}

	// Add template_id column if it doesn't exist (idempotent migration)
	if _, err := db.Exec(addTemplateIDToSessions); err != nil {
		// Ignore "duplicate column" errors — column already exists
		if !isDuplicateColumnError(err) {
			return nil, fmt.Errorf("failed to add template_id column: %w", err)
		}
	}

	// Add skip_permissions column if it doesn't exist (idempotent migration)
	if _, err := db.Exec(addSkipPermissionsToSessions); err != nil {
		if !isDuplicateColumnError(err) {
			return nil, fmt.Errorf("failed to add skip_permissions column: %w", err)
		}
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Create(ctx context.Context, session domain.Session) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO sessions (id, name, working_dir, status, pid, claude_session_id, template_id, skip_permissions, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		session.ID, session.Name, session.WorkingDir, session.Status, session.PID, session.ClaudeSessionID, session.TemplateID, session.SkipPermissions, session.CreatedAt, session.UpdatedAt,
	)
	return err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Session, error) {
	var s domain.Session
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, working_dir, status, pid, claude_session_id, template_id, skip_permissions, created_at, updated_at FROM sessions WHERE id = ?", id,
	).Scan(&s.ID, &s.Name, &s.WorkingDir, &s.Status, &s.PID, &s.ClaudeSessionID, &s.TemplateID, &s.SkipPermissions, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.Session{}, fmt.Errorf("session not found: %s", id)
	}
	return s, err
}

func (r *Repository) List(ctx context.Context) ([]domain.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, working_dir, status, pid, claude_session_id, template_id, skip_permissions, created_at, updated_at FROM sessions ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var sessions []domain.Session
	for rows.Next() {
		var s domain.Session
		if err := rows.Scan(&s.ID, &s.Name, &s.WorkingDir, &s.Status, &s.PID, &s.ClaudeSessionID, &s.TemplateID, &s.SkipPermissions, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *Repository) Update(ctx context.Context, session domain.Session) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE sessions SET name = ?, working_dir = ?, status = ?, pid = ?, claude_session_id = ?, template_id = ?, skip_permissions = ?, updated_at = ? WHERE id = ?",
		session.Name, session.WorkingDir, session.Status, session.PID, session.ClaudeSessionID, session.TemplateID, session.SkipPermissions, session.UpdatedAt, session.ID,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	return err
}

func (r *Repository) DB() *sql.DB {
	return r.db
}

func (r *Repository) Close() error {
	return r.db.Close()
}

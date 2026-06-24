package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %q: %w", dir, err)
		}
	}

	// Append foreign_keys pragma to the DSN so every connection in the pool
	// has FK enforcement enabled, not just the first one.
	dsn := dbPath + "?_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec(createSessionsTable); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if _, err := db.Exec(createWorktreesTable); err != nil {
		return nil, fmt.Errorf("failed to create worktrees table: %w", err)
	}

	if _, err := db.Exec(addSessionIDToWorktrees); err != nil {
		if !isDuplicateColumnError(err) {
			return nil, fmt.Errorf("failed to add session_id column to worktrees: %w", err)
		}
	}

	if _, err := db.Exec(createWorktreeSessionsTable); err != nil {
		return nil, fmt.Errorf("failed to create worktree_sessions table: %w", err)
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

	// Add worktree-related columns if they don't exist (idempotent migrations)
	for _, migration := range []struct {
		sql  string
		name string
	}{
		{addRepoRootToSessions, "repo_root"},
		{addGitBranchToSessions, "git_branch"},
		{addWorktreeManagedToSessions, "worktree_managed"},
		{addErrorToSessions, "error"},
	} {
		if _, err := db.Exec(migration.sql); err != nil {
			if !isDuplicateColumnError(err) {
				return nil, fmt.Errorf("failed to add %s column: %w", migration.name, err)
			}
		}
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Create(ctx context.Context, session domain.Session) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO sessions (id, name, working_dir, status, pid, claude_session_id, template_id, skip_permissions, repo_root, git_branch, worktree_managed, error, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		session.ID, session.Name, session.WorkingDir, session.Status, session.PID, session.ClaudeSessionID, session.TemplateID, session.SkipPermissions, session.RepoRoot, session.GitBranch, session.WorktreeManaged, session.Error, session.CreatedAt, session.UpdatedAt,
	)
	return err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Session, error) {
	var s domain.Session
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, working_dir, status, pid, claude_session_id, template_id, skip_permissions, repo_root, git_branch, worktree_managed, error, created_at, updated_at FROM sessions WHERE id = ?", id,
	).Scan(&s.ID, &s.Name, &s.WorkingDir, &s.Status, &s.PID, &s.ClaudeSessionID, &s.TemplateID, &s.SkipPermissions, &s.RepoRoot, &s.GitBranch, &s.WorktreeManaged, &s.Error, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.Session{}, fmt.Errorf("session not found: %s", id)
	}
	return s, err
}

func (r *Repository) List(ctx context.Context) ([]domain.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, working_dir, status, pid, claude_session_id, template_id, skip_permissions, repo_root, git_branch, worktree_managed, error, created_at, updated_at FROM sessions ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var sessions []domain.Session
	for rows.Next() {
		var s domain.Session
		if err := rows.Scan(&s.ID, &s.Name, &s.WorkingDir, &s.Status, &s.PID, &s.ClaudeSessionID, &s.TemplateID, &s.SkipPermissions, &s.RepoRoot, &s.GitBranch, &s.WorktreeManaged, &s.Error, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *Repository) Update(ctx context.Context, session domain.Session) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE sessions SET name = ?, working_dir = ?, status = ?, pid = ?, claude_session_id = ?, template_id = ?, skip_permissions = ?, repo_root = ?, git_branch = ?, worktree_managed = ?, error = ?, updated_at = ? WHERE id = ?",
		session.Name, session.WorkingDir, session.Status, session.PID, session.ClaudeSessionID, session.TemplateID, session.SkipPermissions, session.RepoRoot, session.GitBranch, session.WorktreeManaged, session.Error, session.UpdatedAt, session.ID,
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

// Worktree repository methods

// WorktreeRepository interface implementation

func (r *Repository) CreateWorktree(ctx context.Context, wt domain.ManagedWorktree) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO worktrees (id, path, branch, repo_root, session_id, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		wt.ID, wt.Path, wt.Branch, wt.RepoRoot, wt.SessionID, wt.CreatedAt,
	)
	return err
}

func (r *Repository) ListWorktrees(ctx context.Context) ([]domain.ManagedWorktree, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, path, branch, repo_root, session_id, created_at FROM worktrees ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var wts []domain.ManagedWorktree
	for rows.Next() {
		var wt domain.ManagedWorktree
		if err := rows.Scan(&wt.ID, &wt.Path, &wt.Branch, &wt.RepoRoot, &wt.SessionID, &wt.CreatedAt); err != nil {
			return nil, err
		}
		wts = append(wts, wt)
	}
	return wts, rows.Err()
}

func (r *Repository) GetWorktree(ctx context.Context, id string) (domain.ManagedWorktree, error) {
	var wt domain.ManagedWorktree
	err := r.db.QueryRowContext(ctx,
		"SELECT id, path, branch, repo_root, session_id, created_at FROM worktrees WHERE id = ?", id,
	).Scan(&wt.ID, &wt.Path, &wt.Branch, &wt.RepoRoot, &wt.SessionID, &wt.CreatedAt)
	if err == sql.ErrNoRows {
		return domain.ManagedWorktree{}, fmt.Errorf("worktree not found: %s", id)
	}
	return wt, err
}

func (r *Repository) GetWorktreeByPath(ctx context.Context, path string) (domain.ManagedWorktree, error) {
	var wt domain.ManagedWorktree
	err := r.db.QueryRowContext(ctx,
		"SELECT id, path, branch, repo_root, session_id, created_at FROM worktrees WHERE path = ?", path,
	).Scan(&wt.ID, &wt.Path, &wt.Branch, &wt.RepoRoot, &wt.SessionID, &wt.CreatedAt)
	if err == sql.ErrNoRows {
		return domain.ManagedWorktree{}, fmt.Errorf("worktree not found: %s", path)
	}
	return wt, err
}

func (r *Repository) DeleteWorktree(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM worktrees WHERE id = ?", id)
	return err
}

func (r *Repository) DeleteWorktreeByPath(ctx context.Context, path string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM worktrees WHERE path = ?", path)
	return err
}

func (r *Repository) LinkWorktreeSession(ctx context.Context, worktreeID, sessionID string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO worktree_sessions (worktree_id, session_id) VALUES (?, ?)",
		worktreeID, sessionID,
	)
	return err
}

func (r *Repository) UnlinkWorktreeSession(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM worktree_sessions WHERE session_id = ?", sessionID)
	return err
}

func (r *Repository) ListWorktreeSessionIDs(ctx context.Context, worktreeID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT session_id FROM worktree_sessions WHERE worktree_id = ?", worktreeID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) SetWorktreeSession(ctx context.Context, worktreeID string, sessionID *string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE worktrees SET session_id = ? WHERE id = ?", sessionID, worktreeID)
	return err
}

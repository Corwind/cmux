package sqlite

const createSessionsTable = `
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	working_dir TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'stopped',
	pid INTEGER DEFAULT 0,
	claude_session_id TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);
`

const addTemplateIDToSessions = `
ALTER TABLE sessions ADD COLUMN template_id TEXT NOT NULL DEFAULT '';
`

const addSkipPermissionsToSessions = `
ALTER TABLE sessions ADD COLUMN skip_permissions INTEGER NOT NULL DEFAULT 0;
`

const addRepoRootToSessions = `
ALTER TABLE sessions ADD COLUMN repo_root TEXT NOT NULL DEFAULT '';
`

const addGitBranchToSessions = `
ALTER TABLE sessions ADD COLUMN git_branch TEXT NOT NULL DEFAULT '';
`

const addWorktreeManagedToSessions = `
ALTER TABLE sessions ADD COLUMN worktree_managed INTEGER NOT NULL DEFAULT 0;
`

const createWorktreesTable = `
CREATE TABLE IF NOT EXISTS worktrees (
	id TEXT PRIMARY KEY,
	path TEXT NOT NULL UNIQUE,
	branch TEXT NOT NULL DEFAULT '',
	repo_root TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL
);
`

const createWorktreeSessionsTable = `
CREATE TABLE IF NOT EXISTS worktree_sessions (
	worktree_id TEXT NOT NULL REFERENCES worktrees(id) ON DELETE CASCADE,
	session_id  TEXT NOT NULL REFERENCES sessions(id)  ON DELETE CASCADE,
	PRIMARY KEY (worktree_id, session_id)
);
`

const addSessionIDToWorktrees = "ALTER TABLE worktrees ADD COLUMN session_id TEXT REFERENCES sessions(id) ON DELETE SET NULL;"

const createTemplatesTable = `
CREATE TABLE IF NOT EXISTS sandbox_templates (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	content TEXT NOT NULL,
	is_default INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);
`

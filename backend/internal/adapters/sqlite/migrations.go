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

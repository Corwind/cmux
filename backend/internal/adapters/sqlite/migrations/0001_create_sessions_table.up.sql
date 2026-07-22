CREATE TABLE sessions (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	working_dir TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'stopped',
	pid INTEGER DEFAULT 0,
	claude_session_id TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

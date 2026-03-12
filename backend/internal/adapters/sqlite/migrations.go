package sqlite

const createSessionsTable = `
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	working_dir TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'stopped',
	pid INTEGER DEFAULT 0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);
`

CREATE TABLE sandbox_templates (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	content TEXT NOT NULL,
	is_default INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

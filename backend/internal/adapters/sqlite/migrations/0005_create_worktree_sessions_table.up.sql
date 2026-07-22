CREATE TABLE worktree_sessions (
	worktree_id TEXT NOT NULL REFERENCES worktrees(id) ON DELETE CASCADE,
	session_id  TEXT NOT NULL REFERENCES sessions(id)  ON DELETE CASCADE,
	PRIMARY KEY (worktree_id, session_id)
);

import { useWorktrees, useDeleteWorktree } from "../hooks/useWorktrees";
import { useSessionsStore } from "../stores/sessions.store";
import type { WorktreeEntry } from "../types";

function repoName(repoRoot: string): string {
  const parts = repoRoot.split("/").filter(Boolean);
  return parts[parts.length - 1] ?? repoRoot;
}

function groupByRepo(entries: WorktreeEntry[]): Map<string, WorktreeEntry[]> {
  const map = new Map<string, WorktreeEntry[]>();
  for (const e of entries) {
    const key = e.repo_root || e.path;
    const list = map.get(key) ?? [];
    list.push(e);
    map.set(key, list);
  }
  return map;
}

export function WorktreePanel() {
  const { data: worktrees } = useWorktrees();
  const deleteWorktree = useDeleteWorktree();
  const { activeSessionId, setActiveSession } = useSessionsStore();

  const entries = worktrees ?? [];

  if (entries.length === 0) {
    return (
      <p className="px-1 text-xs" style={{ color: "var(--cmux-text-faint)" }}>
        No managed worktrees.
      </p>
    );
  }

  const groups = groupByRepo(entries);

  function handleDelete(entry: WorktreeEntry) {
    const hasActiveSessions = entry.sessions.length > 0;
    if (hasActiveSessions) {
      const confirmed = window.confirm(
        `Worktree "${entry.branch}" still has ${entry.sessions.length} session(s).\nForce-delete the worktree anyway?`
      );
      if (!confirmed) return;
      deleteWorktree.mutate({ id: entry.id, force: true });
    } else {
      deleteWorktree.mutate({ id: entry.id });
    }
  }

  return (
    <div className="space-y-3">
      {[...groups.entries()].map(([repoRoot, entries]) => (
        <div key={repoRoot}>
          <div
            className="mb-1 truncate px-1 text-xs font-medium"
            style={{ color: "var(--cmux-text-faint)" }}
            title={repoRoot}
          >
            {repoName(repoRoot)}
          </div>
          <ul className="space-y-0.5">
            {entries.map((entry) => {
              const firstSessionId = entry.sessions[0]?.id;
              const isActive = firstSessionId !== undefined && activeSessionId === firstSessionId;

              return (
                <li key={entry.id}>
                  <button
                    type="button"
                    onClick={() => {
                      if (firstSessionId) setActiveSession(firstSessionId);
                    }}
                    className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-xs transition-colors"
                    style={{
                      backgroundColor: isActive ? "var(--cmux-active)" : undefined,
                      color: isActive ? "var(--cmux-text)" : "var(--cmux-text-secondary)",
                      cursor: firstSessionId ? "pointer" : "default",
                    }}
                    onMouseEnter={(e) => {
                      if (!isActive) {
                        e.currentTarget.style.backgroundColor = "var(--cmux-surface-hover)";
                        e.currentTarget.style.color = "var(--cmux-text)";
                      }
                    }}
                    onMouseLeave={(e) => {
                      if (!isActive) {
                        e.currentTarget.style.backgroundColor = "";
                        e.currentTarget.style.color = "var(--cmux-text-secondary)";
                      }
                    }}
                  >
                    <span
                      className="shrink-0 font-mono text-xs"
                      style={{ color: "var(--cmux-accent)" }}
                    >
                      ⎇
                    </span>
                    <span className="min-w-0 flex-1 truncate font-medium" title={entry.branch}>
                      {entry.branch || entry.path}
                    </span>
                    {entry.sessions.length === 0 ? (
                      <span
                        className="shrink-0 rounded px-1 text-xs"
                        style={{
                          backgroundColor: "var(--cmux-surface-hover)",
                          color: "var(--cmux-text-faint)",
                        }}
                      >
                        orphaned
                      </span>
                    ) : (
                      <span
                        className="shrink-0 text-xs tabular-nums"
                        style={{ color: "var(--cmux-text-faint)" }}
                        title={entry.sessions.map((s) => s.name).join(", ")}
                      >
                        {entry.sessions.length}
                      </span>
                    )}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDelete(entry);
                      }}
                      className="shrink-0 rounded p-0.5 transition-colors"
                      style={{ color: "var(--cmux-text-muted)" }}
                      title="Delete worktree"
                      onMouseEnter={(e) => {
                        e.currentTarget.style.backgroundColor = "var(--cmux-surface-hover)";
                        e.currentTarget.style.color = "#f87171";
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.backgroundColor = "";
                        e.currentTarget.style.color = "var(--cmux-text-muted)";
                      }}
                    >
                      <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </button>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </div>
  );
}

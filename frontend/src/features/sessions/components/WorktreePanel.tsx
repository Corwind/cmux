import { useWorktrees, useDeleteWorktree, groupByRepo } from "../hooks/useWorktrees";
import { useSessionsStore } from "../stores/sessions.store";
import type { WorktreeEntry } from "../types";

function statusDotColor(entry: WorktreeEntry): string {
  if (entry.status === "deleting") {
    return "#f97316"; // orange
  }
  if (entry.session_status === "running") {
    return "#22c55e";
  }
  if (entry.session_id && entry.session_status === "stopped") {
    return "#eab308";
  }
  return "var(--cmux-text-faint)";
}

export function WorktreePanel() {
  const { data: worktrees, isLoading } = useWorktrees();
  const deleteWorktree = useDeleteWorktree();
  const setActiveSession = useSessionsStore((s) => s.setActiveSession);

  if (isLoading) {
    return (
      <div className="px-1 text-xs" style={{ color: "var(--cmux-text-muted)" }}>
        Loading...
      </div>
    );
  }

  if (!worktrees || worktrees.length === 0) {
    return (
      <div className="px-1 text-xs" style={{ color: "var(--cmux-text-muted)" }}>
        No managed worktrees.
      </div>
    );
  }

  const grouped = groupByRepo(worktrees);

  return (
    <div className="space-y-2">
      {Array.from(grouped.entries()).map(([repoRoot, entries]) => (
        <div key={repoRoot}>
          <div
            className="mb-1 truncate px-1 text-xs font-medium"
            style={{ color: "var(--cmux-text-faint)" }}
            title={repoRoot}
          >
            {repoRoot.split("/").pop() ?? repoRoot}
          </div>
          <ul className="space-y-0.5">
            {entries.map((entry) => {
              const isDeleting = entry.status === "deleting";
              const hasSession = !!entry.session_id;
              const deleteDisabled = hasSession || isDeleting;
              const deleteTitle = hasSession
                ? "Delete the session first to remove this worktree"
                : isDeleting
                  ? "Removing worktree..."
                  : "Delete worktree";
              return (
              <li key={entry.id}>
                <div
                  className="flex w-full items-center justify-between rounded px-2 py-1 text-sm transition-colors"
                  style={{ color: "var(--cmux-text-secondary)", opacity: isDeleting ? 0.5 : 1 }}
                >
                  <button
                    type="button"
                    className="flex min-w-0 flex-1 items-center gap-1.5 text-left"
                    onClick={() => {
                      if (entry.session_id) {
                        setActiveSession(entry.session_id);
                      }
                    }}
                    style={{ cursor: entry.session_id ? "pointer" : "default" }}
                    onMouseEnter={(e) => {
                      if (entry.session_id) {
                        e.currentTarget.style.color = "var(--cmux-text)";
                      }
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.color = "";
                    }}
                  >
                    <span
                      className="inline-block shrink-0 rounded-full"
                      style={{
                        width: 8,
                        height: 8,
                        backgroundColor: statusDotColor(entry),
                      }}
                      aria-hidden="true"
                    />
                    <span className="truncate font-mono text-xs" title={entry.branch}>
                      {entry.branch}
                    </span>
                    {isDeleting && (
                      <svg
                        className="h-3 w-3 shrink-0 animate-spin"
                        viewBox="0 0 24 24"
                        fill="none"
                        aria-label="Removing worktree"
                        role="img"
                      >
                        <circle
                          className="opacity-25"
                          cx="12"
                          cy="12"
                          r="10"
                          stroke="currentColor"
                          strokeWidth="4"
                        />
                        <path
                          className="opacity-75"
                          fill="currentColor"
                          d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
                        />
                      </svg>
                    )}
                  </button>
                  <button
                    type="button"
                    disabled={deleteDisabled}
                    onClick={() => deleteWorktree.mutate(entry.id)}
                    className="ml-1 shrink-0 rounded p-0.5 transition-colors"
                    style={{
                      color: "var(--cmux-text-faint)",
                      opacity: deleteDisabled ? 0.3 : 1,
                      cursor: deleteDisabled ? "not-allowed" : "pointer",
                    }}
                    title={deleteTitle}
                    onMouseEnter={(e) => {
                      if (!deleteDisabled) {
                        e.currentTarget.style.backgroundColor = "var(--cmux-surface-hover)";
                        e.currentTarget.style.color = "#f87171";
                      }
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.backgroundColor = "";
                      e.currentTarget.style.color = "var(--cmux-text-faint)";
                    }}
                  >
                    <svg
                      className="h-3 w-3"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      strokeWidth={2}
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                </div>
              </li>
              );
            })}
          </ul>
        </div>
      ))}
    </div>
  );
}

import { useSessions } from "../hooks/useSessions";
import { useDeleteSession } from "../hooks/useDeleteSession";
import { useResumeSession } from "../hooks/useResumeSession";
import { useRestartSession } from "../hooks/useRestartSession";
import { useSessionsStore } from "../stores/sessions.store";
import { StatusBadge } from "./StatusBadge";

export function SessionList() {
  const { data: sessions, isLoading } = useSessions();
  const deleteSession = useDeleteSession();
  const resumeSession = useResumeSession();
  const restartSession = useRestartSession();
  const { activeSessionId, setActiveSession } = useSessionsStore();

  if (isLoading) {
    return (
      <div className="p-4 text-sm" style={{ color: "var(--cmux-text-muted)" }}>
        Loading sessions...
      </div>
    );
  }

  if (!sessions || sessions.length === 0) {
    return (
      <div className="p-4 text-sm" style={{ color: "var(--cmux-text-muted)" }}>
        No sessions yet. Create one to get started.
      </div>
    );
  }

  return (
    <ul className="space-y-1">
      {sessions.map((session) => (
        <li key={session.id}>
          <button
            type="button"
            onClick={() => setActiveSession(session.id)}
            className="flex w-full items-center justify-between rounded px-3 py-2 text-left text-sm transition-colors"
            style={{
              backgroundColor:
                activeSessionId === session.id
                  ? "var(--cmux-active)"
                  : undefined,
              color:
                activeSessionId === session.id
                  ? "var(--cmux-text)"
                  : "var(--cmux-text-secondary)",
            }}
            onMouseEnter={(e) => {
              if (activeSessionId !== session.id) {
                e.currentTarget.style.backgroundColor =
                  "var(--cmux-surface-hover)";
                e.currentTarget.style.color = "var(--cmux-text)";
              }
            }}
            onMouseLeave={(e) => {
              if (activeSessionId !== session.id) {
                e.currentTarget.style.backgroundColor = "";
                e.currentTarget.style.color = "var(--cmux-text-secondary)";
              }
            }}
          >
            <div className="min-w-0 flex-1">
              <div className="truncate font-medium">{session.name}</div>
              <div
                className="truncate text-xs"
                style={{ color: "var(--cmux-text-muted)" }}
              >
                {session.working_dir}
              </div>
            </div>
            <div className="ml-2 flex items-center gap-2">
              <StatusBadge status={session.status} />
              {session.status === "stopped" && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    resumeSession.mutate(session.id, {
                      onSuccess: () => setActiveSession(session.id),
                    });
                  }}
                  className="rounded p-0.5 transition-colors"
                  style={{ color: "var(--cmux-text-muted)" }}
                  title="Resume session"
                  onMouseEnter={(e) => {
                    e.currentTarget.style.backgroundColor =
                      "var(--cmux-surface-hover)";
                    e.currentTarget.style.color = "var(--cmux-accent)";
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = "";
                    e.currentTarget.style.color = "var(--cmux-text-muted)";
                  }}
                >
                  <svg
                    className="h-3.5 w-3.5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2}
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M5 3l14 9-14 9V3z"
                    />
                  </svg>
                </button>
              )}
              {session.status === "running" && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    restartSession.mutate(session.id, {
                      onSuccess: () => setActiveSession(session.id),
                    });
                  }}
                  className="rounded p-0.5 transition-colors"
                  style={{ color: "var(--cmux-text-muted)" }}
                  title="Restart session (refresh sandbox template)"
                  onMouseEnter={(e) => {
                    e.currentTarget.style.backgroundColor =
                      "var(--cmux-surface-hover)";
                    e.currentTarget.style.color = "var(--cmux-accent)";
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = "";
                    e.currentTarget.style.color = "var(--cmux-text-muted)";
                  }}
                >
                  <svg
                    className="h-3.5 w-3.5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2}
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                    />
                  </svg>
                </button>
              )}
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  deleteSession.mutate(session.id);
                }}
                className="rounded p-0.5 transition-colors"
                style={{ color: "var(--cmux-text-muted)" }}
                title="Delete session"
                onMouseEnter={(e) => {
                  e.currentTarget.style.backgroundColor =
                    "var(--cmux-surface-hover)";
                  e.currentTarget.style.color = "#f87171";
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = "";
                  e.currentTarget.style.color = "var(--cmux-text-muted)";
                }}
              >
                <svg
                  className="h-3.5 w-3.5"
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
          </button>
        </li>
      ))}
    </ul>
  );
}

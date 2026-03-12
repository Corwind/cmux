import { cn } from "@/lib/cn";
import { useSessions } from "../hooks/useSessions";
import { useDeleteSession } from "../hooks/useDeleteSession";
import { useResumeSession } from "../hooks/useResumeSession";
import { useSessionsStore } from "../stores/sessions.store";
import { StatusBadge } from "./StatusBadge";

export function SessionList() {
  const { data: sessions, isLoading } = useSessions();
  const deleteSession = useDeleteSession();
  const resumeSession = useResumeSession();
  const { activeSessionId, setActiveSession } = useSessionsStore();

  if (isLoading) {
    return (
      <div className="p-4 text-sm text-gray-500">Loading sessions...</div>
    );
  }

  if (!sessions || sessions.length === 0) {
    return (
      <div className="p-4 text-sm text-gray-500">
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
            className={cn(
              "flex w-full items-center justify-between rounded px-3 py-2 text-left text-sm transition-colors",
              activeSessionId === session.id
                ? "bg-gray-700 text-white"
                : "text-gray-300 hover:bg-gray-800 hover:text-white",
            )}
          >
            <div className="min-w-0 flex-1">
              <div className="truncate font-medium">{session.name}</div>
              <div className="truncate text-xs text-gray-500">
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
                  className="rounded p-0.5 text-gray-500 hover:bg-gray-600 hover:text-green-400"
                  title="Resume session"
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
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  deleteSession.mutate(session.id);
                }}
                className="rounded p-0.5 text-gray-500 hover:bg-gray-600 hover:text-red-400"
                title="Delete session"
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

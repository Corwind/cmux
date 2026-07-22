import { useSessions } from "../hooks/useSessions";
import { useDeleteSession } from "../hooks/useDeleteSession";
import { useResumeSession } from "../hooks/useResumeSession";
import { useRestartSession } from "../hooks/useRestartSession";
import { useSessionsStore } from "../stores/sessions.store";
import { useNotificationStore } from "../stores/notification.store";
import { useHarnesses } from "@/features/harnesses";
import { StatusBadge } from "./StatusBadge";
import type { Session } from "../types";

function BranchBadge({ branch }: { branch: string }) {
  return (
    <span
      className="max-w-[80px] truncate rounded px-1 py-0.5 text-xs"
      style={{
        backgroundColor: "var(--cmux-sidebar)",
        border: "1px solid var(--cmux-border-light)",
        color: "var(--cmux-text-muted)",
      }}
      title={branch}
    >
      {branch}
    </span>
  );
}

function groupByHarness(sessions: Session[]): Map<string, Session[]> {
  const map = new Map<string, Session[]>();
  for (const session of sessions) {
    const group = map.get(session.harness_type) ?? [];
    group.push(session);
    map.set(session.harness_type, group);
  }
  return map;
}

interface SessionRowProps {
  session: Session;
  activeSessionId: string | null | undefined;
  hasNotification: boolean;
  onActivate: () => void;
  onResume: () => void;
  onRestart: () => void;
  onDelete: () => void;
}

function SessionRow({
  session,
  activeSessionId,
  hasNotification,
  onActivate,
  onResume,
  onRestart,
  onDelete,
}: SessionRowProps) {
  /*
    The row is a div (not a button) with role="button". The delete /
    resume / restart controls are real <button>s nested inside the row;
    nesting a button inside a button is invalid HTML and the browser
    drops clicks on the inner buttons — which is why deleting a
    provisioning session did nothing. A div row keeps those controls
    valid and clickable in every state.
  */
  const isInteractive =
    session.status !== "provisioning" && session.status !== "failed";
  const activate = () => {
    if (!isInteractive) return;
    onActivate();
  };
  return (
    <div
      role="button"
      tabIndex={isInteractive ? 0 : -1}
      aria-disabled={!isInteractive}
      onClick={activate}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          activate();
        }
      }}
      className={`flex w-full items-center justify-between rounded px-3 py-2 text-left text-sm transition-colors ${
        session.status === "provisioning"
          ? "cursor-not-allowed opacity-60"
          : "cursor-pointer"
      }`}
      style={{
        backgroundColor:
          activeSessionId === session.id ? "var(--cmux-active)" : undefined,
        color:
          activeSessionId === session.id
            ? "var(--cmux-text)"
            : "var(--cmux-text-secondary)",
      }}
      onMouseEnter={(e) => {
        if (activeSessionId !== session.id && isInteractive) {
          e.currentTarget.style.backgroundColor = "var(--cmux-surface-hover)";
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
        <div className="flex items-center gap-1.5 truncate font-medium">
          {session.name}
          {session.git_branch && <BranchBadge branch={session.git_branch} />}
        </div>
        {session.status === "provisioning" ? (
          <div className="truncate text-xs text-blue-400">
            Creating worktree...
          </div>
        ) : session.status === "failed" ? (
          <div className="truncate text-xs text-red-400">
            {session.error_message ?? "Provisioning failed"}
          </div>
        ) : (
          <div
            className="truncate text-xs"
            style={{ color: "var(--cmux-text-muted)" }}
          >
            {session.working_dir}
          </div>
        )}
      </div>
      <div className="ml-2 flex items-center gap-2">
        <StatusBadge status={session.status} hasNotification={hasNotification} />
        {session.status === "stopped" && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onResume();
            }}
            className="rounded p-0.5 transition-colors"
            style={{ color: "var(--cmux-text-muted)" }}
            title="Resume session"
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = "var(--cmux-surface-hover)";
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
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 3l14 9-14 9V3z" />
            </svg>
          </button>
        )}
        {session.status === "running" && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onRestart();
            }}
            className="rounded p-0.5 transition-colors"
            style={{ color: "var(--cmux-text-muted)" }}
            title="Restart session (refresh sandbox template)"
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = "var(--cmux-surface-hover)";
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
        {(session.status === "provisioning" ||
          session.status === "failed" ||
          session.status === "stopped" ||
          session.status === "running") && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onDelete();
            }}
            className="rounded p-0.5 transition-colors"
            style={{ color: "var(--cmux-text-muted)" }}
            title="Delete session"
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = "var(--cmux-surface-hover)";
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
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>
    </div>
  );
}

export function SessionList() {
  const { data: sessions, isLoading } = useSessions();
  const { data: harnesses } = useHarnesses();
  const deleteSessionMutation = useDeleteSession();
  const resumeSession = useResumeSession();
  const restartSession = useRestartSession();
  const { activeSessionId, setActiveSession } = useSessionsStore();
  const notifications = useNotificationStore((s) => s.notifications);

  function handleDelete(session: Session) {
    deleteSessionMutation.mutate(session.id);
  }

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

  const grouped = groupByHarness(sessions);
  const registryOrder = harnesses?.map((h) => h.type) ?? [];
  const orderedTypes = [
    ...registryOrder.filter((type) => grouped.has(type)),
    ...Array.from(grouped.keys()).filter((type) => !registryOrder.includes(type)),
  ];

  return (
    <div className="space-y-3">
      {orderedTypes.map((harnessType) => {
        const group = grouped.get(harnessType);
        if (!group) return null;
        const sectionName =
          harnesses?.find((h) => h.type === harnessType)?.section_name ?? harnessType;
        return (
          <div key={harnessType}>
            <div
              className="mb-1 truncate px-1 text-xs font-medium"
              style={{ color: "var(--cmux-text-faint)" }}
            >
              {sectionName}
            </div>
            <ul className="space-y-1">
              {group.map((session) => (
                <li key={session.id}>
                  <SessionRow
                    session={session}
                    activeSessionId={activeSessionId}
                    hasNotification={
                      notifications[session.id]?.eventType === "waiting_input"
                    }
                    onActivate={() => {
                      setActiveSession(session.id);
                      useNotificationStore.getState().clearNotification(session.id);
                    }}
                    onResume={() =>
                      resumeSession.mutate(session.id, {
                        onSuccess: () => setActiveSession(session.id),
                      })
                    }
                    onRestart={() =>
                      restartSession.mutate(session.id, {
                        onSuccess: () => setActiveSession(session.id),
                      })
                    }
                    onDelete={() => handleDelete(session)}
                  />
                </li>
              ))}
            </ul>
          </div>
        );
      })}
    </div>
  );
}

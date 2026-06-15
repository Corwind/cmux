import { useEffect } from "react";
import { useSessionsStore } from "../stores/sessions.store";
import { useNotificationStore } from "../stores/notification.store";
import type { SessionNotification } from "../stores/notification.store";

function getNotificationBody(n: SessionNotification): string {
  switch (n.eventType) {
    case "waiting_input":
      return "Claude is waiting for your input";
    case "task_complete":
      return "Task complete";
    default:
      return n.message || "New notification";
  }
}

async function fireSystemNotification(n: SessionNotification): Promise<void> {
  if (!("Notification" in window)) return;
  if (Notification.permission === "default") {
    await Notification.requestPermission();
  }
  if (Notification.permission === "granted") {
    new Notification(n.sessionName, { body: getNotificationBody(n) });
  }
}

export function useSystemNotifications(): void {
  const activeSessionId = useSessionsStore((s) => s.activeSessionId);

  useEffect(() => {
    return useNotificationStore.subscribe((state, prevState) => {
      for (const [sessionId, notification] of Object.entries(state.notifications)) {
        const prev = prevState.notifications[sessionId];
        if (!prev || prev.timestamp !== notification.timestamp) {
          const isActiveSession = sessionId === activeSessionId;
          const isTabHidden = document.hidden;
          if (!isActiveSession || isTabHidden) {
            void fireSystemNotification(notification);
          }
        }
      }
    });
  }, [activeSessionId]);
}

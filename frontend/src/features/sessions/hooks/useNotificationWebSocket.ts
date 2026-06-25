import { useEffect } from "react";
import { useNotificationStore } from "../stores/notification.store";
import type { NotificationEventType } from "../stores/notification.store";
import { queryClient } from "@/config/query-client";
import { sessionKeys } from "./useSessions";
import { worktreeKeys } from "./useWorktrees";
import { toast } from "@/components/ui/Toast";

interface NotificationMsg {
  session_id: string;
  session_name: string;
  message: string;
  event_type: NotificationEventType;
}

interface SessionStatusMsg {
  type: "session_status";
  session_id: string;
  status: "running" | "failed";
  error?: string;
}

interface WorktreeDeletedMsg {
  type: "worktree_deleted";
  worktree_id: string;
  error?: string;
}

type IncomingMsg = NotificationMsg | SessionStatusMsg | WorktreeDeletedMsg;

export function useNotificationWebSocket(): void {
  useEffect(() => {
    let ws: WebSocket | null = null;
    let alive = true;
    let reconnectTimer: ReturnType<typeof setTimeout>;

    function connect() {
      if (!alive) return;
      const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
      ws = new WebSocket(`${proto}//${window.location.host}/ws/notifications`);

      ws.onmessage = (event: MessageEvent) => {
        try {
          const msg = JSON.parse(event.data as string) as IncomingMsg;
          if ("type" in msg && msg.type === "session_status") {
            void queryClient.invalidateQueries({ queryKey: sessionKeys.all });
            if (msg.status === "failed") {
              toast("error", "Session provisioning failed", msg.error ?? "Worktree creation failed");
            }
          } else if ("type" in msg && msg.type === "worktree_deleted") {
            void queryClient.invalidateQueries({ queryKey: worktreeKeys.all });
            if (msg.error) {
              toast("error", "Worktree removal failed", msg.error);
            }
          } else {
            const notif = msg as NotificationMsg;
            useNotificationStore
              .getState()
              .notify(notif.session_id, notif.session_name, notif.message, notif.event_type);
          }
        } catch {
          // Ignore malformed messages
        }
      };

      ws.onclose = () => {
        if (!alive) return;
        reconnectTimer = setTimeout(connect, 2000);
      };
    }

    connect();

    return () => {
      alive = false;
      clearTimeout(reconnectTimer);
      ws?.close();
    };
  }, []);
}

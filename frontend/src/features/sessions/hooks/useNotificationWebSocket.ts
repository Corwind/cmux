import { useEffect } from "react";
import { useNotificationStore } from "../stores/notification.store";
import type { NotificationEventType } from "../stores/notification.store";

interface NotificationMsg {
  session_id: string;
  session_name: string;
  message: string;
  event_type: NotificationEventType;
}

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
          const msg = JSON.parse(event.data as string) as NotificationMsg;
          useNotificationStore
            .getState()
            .notify(msg.session_id, msg.session_name, msg.message, msg.event_type);
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

import { create } from "zustand";

export type NotificationEventType = "waiting_input" | "task_complete" | "generic";

export interface SessionNotification {
  sessionId: string;
  sessionName: string;
  message: string;
  eventType: NotificationEventType;
  timestamp: number;
}

interface NotificationState {
  notifications: Record<string, SessionNotification>;
  notify: (
    sessionId: string,
    sessionName: string,
    message: string,
    eventType: NotificationEventType,
  ) => void;
  clearNotification: (sessionId: string) => void;
}

export const useNotificationStore = create<NotificationState>()((set) => ({
  notifications: {},
  notify: (sessionId, sessionName, message, eventType) =>
    set((s) => ({
      notifications: {
        ...s.notifications,
        [sessionId]: { sessionId, sessionName, message, eventType, timestamp: Date.now() },
      },
    })),
  clearNotification: (sessionId) =>
    set((s) => {
      const next = { ...s.notifications };
      delete next[sessionId];
      return { notifications: next };
    }),
}));

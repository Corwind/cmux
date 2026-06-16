import { describe, it, expect, beforeEach } from "vitest";
import { useNotificationStore } from "./notification.store";

describe("useNotificationStore", () => {
  beforeEach(() => {
    useNotificationStore.setState({ notifications: {} });
  });

  it("notify() creates a notification keyed by sessionId", () => {
    const { notify } = useNotificationStore.getState();
    notify("session-1", "My Session", "Waiting for input", "waiting_input");

    const notifications = useNotificationStore.getState().notifications;
    const n = notifications["session-1"];
    expect(n).toBeDefined();
    expect(n!.sessionId).toBe("session-1");
    expect(n!.sessionName).toBe("My Session");
    expect(n!.message).toBe("Waiting for input");
    expect(n!.eventType).toBe("waiting_input");
    expect(typeof n!.timestamp).toBe("number");
  });

  it("notify() overwrites previous notification for the same sessionId with a different timestamp", () => {
    const { notify } = useNotificationStore.getState();
    notify("session-1", "My Session", "First message", "waiting_input");
    const firstTimestamp = useNotificationStore.getState().notifications["session-1"]!.timestamp;

    // Ensure time advances
    const start = Date.now();
    while (Date.now() === start) {
      // spin
    }

    notify("session-1", "My Session", "Second message", "task_complete");
    const notifications = useNotificationStore.getState().notifications;

    expect(Object.keys(notifications)).toHaveLength(1);
    expect(notifications["session-1"]!.message).toBe("Second message");
    expect(notifications["session-1"]!.eventType).toBe("task_complete");
    expect(notifications["session-1"]!.timestamp).toBeGreaterThan(firstTimestamp);
  });

  it("clearNotification() removes the notification", () => {
    const { notify, clearNotification } = useNotificationStore.getState();
    notify("session-1", "My Session", "Task complete", "task_complete");
    expect(useNotificationStore.getState().notifications["session-1"]).toBeDefined();

    clearNotification("session-1");
    expect(useNotificationStore.getState().notifications["session-1"]).toBeUndefined();
    expect(Object.keys(useNotificationStore.getState().notifications)).toHaveLength(0);
  });

  it("clearNotification() on non-existent key is a no-op and does not affect other keys", () => {
    const { notify, clearNotification } = useNotificationStore.getState();
    notify("session-1", "My Session", "Generic message", "generic");

    expect(() => clearNotification("non-existent")).not.toThrow();

    const notifications = useNotificationStore.getState().notifications;
    expect(notifications["session-1"]).toBeDefined();
    expect(Object.keys(notifications)).toHaveLength(1);
  });
});

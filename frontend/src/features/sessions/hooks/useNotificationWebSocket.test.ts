import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, waitFor } from "@/test/test-utils";
import { useNotificationWebSocket } from "./useNotificationWebSocket";
import { useNotificationStore } from "../stores/notification.store";

// ---------------------------------------------------------------------------
// Minimal WebSocket mock
// ---------------------------------------------------------------------------

type WsHandler = {
  onmessage: ((event: MessageEvent) => void) | null;
  onclose: (() => void) | null;
  close: () => void;
  send: (data: string) => void;
  readyState: number;
};

let currentWs: WsHandler | null = null;

class MockWebSocket implements WsHandler {
  onmessage: ((event: MessageEvent) => void) | null = null;
  onclose: (() => void) | null = null;
  readyState = 1; // OPEN

  constructor(_url: string) {
    currentWs = this;
  }

  send(_data: string) {}

  close() {
    this.readyState = 3; // CLOSED
    currentWs = null;
  }
}

// ---------------------------------------------------------------------------
// Mock QueryClient (for session_status invalidation tests)
// ---------------------------------------------------------------------------

const mockInvalidateQueries = vi.fn();

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: () => ({ invalidateQueries: mockInvalidateQueries }),
  };
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("useNotificationWebSocket", () => {
  beforeEach(() => {
    vi.stubGlobal("WebSocket", MockWebSocket);
    useNotificationStore.setState({ notifications: {} });
    mockInvalidateQueries.mockClear();
    currentWs = null;
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  function sendMessage(data: unknown) {
    expect(currentWs).not.toBeNull();
    currentWs!.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
  }

  it("regular notification message (no type field) still updates notification store", async () => {
    renderHook(() => useNotificationWebSocket());

    await waitFor(() => expect(currentWs).not.toBeNull());

    sendMessage({
      session_id: "session-1",
      session_name: "My Session",
      message: "Waiting for input",
      event_type: "waiting_input",
    });

    await waitFor(() => {
      const notifications = useNotificationStore.getState().notifications;
      // The notification may be suppressed if session-1 is the active session.
      // In this test, no session is active so it should be stored.
      expect(notifications["session-1"]).toBeDefined();
    });
  });

  it("session_status message with status 'running' triggers queryClient.invalidateQueries", async () => {
    renderHook(() => useNotificationWebSocket());

    await waitFor(() => expect(currentWs).not.toBeNull());

    sendMessage({
      type: "session_status",
      session_id: "session-1",
      status: "running",
    });

    await waitFor(() => {
      expect(mockInvalidateQueries).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: expect.arrayContaining(["sessions"]) }),
      );
    });
  });

  it("session_status message with status 'failed' triggers queryClient.invalidateQueries", async () => {
    renderHook(() => useNotificationWebSocket());

    await waitFor(() => expect(currentWs).not.toBeNull());

    sendMessage({
      type: "session_status",
      session_id: "session-1",
      status: "failed",
      error_message: "Failed to create worktree",
    });

    await waitFor(() => {
      expect(mockInvalidateQueries).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: expect.arrayContaining(["sessions"]) }),
      );
    });
  });

  it("both message types are handled in the same onmessage handler", async () => {
    renderHook(() => useNotificationWebSocket());

    await waitFor(() => expect(currentWs).not.toBeNull());

    // Send a regular notification first
    sendMessage({
      session_id: "session-2",
      session_name: "Session Two",
      message: "Task complete",
      event_type: "task_complete",
    });

    // Then send a session_status message
    sendMessage({
      type: "session_status",
      session_id: "session-2",
      status: "running",
    });

    await waitFor(() => {
      // Notification store should have session-2 from the first message
      const notifications = useNotificationStore.getState().notifications;
      expect(notifications["session-2"]).toBeDefined();

      // invalidateQueries should have been called for the second message
      expect(mockInvalidateQueries).toHaveBeenCalledWith(
        expect.objectContaining({ queryKey: expect.arrayContaining(["sessions"]) }),
      );
    });
  });
});

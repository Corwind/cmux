import { renderHook, waitFor } from "@/test/test-utils";
import { http, HttpResponse } from "msw";
import { describe, expect, it, beforeEach } from "vitest";
import { server } from "@/test/mocks/server";
import { useDeleteSession } from "./useDeleteSession";
import { useSessionsStore } from "../stores/sessions.store";

describe("useDeleteSession", () => {
  beforeEach(() => {
    useSessionsStore.setState({ activeSessionId: null });
    server.use(
      http.delete("/api/sessions/:id", () => {
        return new HttpResponse(null, { status: 204 });
      }),
    );
  });

  it("clears active session when deleting the active session", async () => {
    useSessionsStore.setState({ activeSessionId: "session-1" });

    const { result } = renderHook(() => useDeleteSession());

    result.current.mutate("session-1");

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(useSessionsStore.getState().activeSessionId).toBeNull();
  });

  it("does not clear active session when deleting a different session", async () => {
    useSessionsStore.setState({ activeSessionId: "session-2" });

    const { result } = renderHook(() => useDeleteSession());

    result.current.mutate("session-1");

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(useSessionsStore.getState().activeSessionId).toBe("session-2");
  });
});

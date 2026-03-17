import { renderHook, waitFor } from "@/test/test-utils";
import { http, HttpResponse } from "msw";
import { describe, expect, it, beforeEach } from "vitest";
import { server } from "@/test/mocks/server";
import { useRestartSession } from "./useRestartSession";

const mockSession = {
  id: "session-1",
  name: "test",
  working_dir: "/tmp",
  status: "running",
  pid: 42,
  template_id: "tmpl-1",
  skip_permissions: false,
  created_at: "2026-03-17T10:00:00Z",
  updated_at: "2026-03-17T10:05:00Z",
};

describe("useRestartSession", () => {
  beforeEach(() => {
    server.use(
      http.post("http://localhost:3001/api/sessions/:id/restart", () => {
        return HttpResponse.json(mockSession);
      }),
    );
  });

  it("restarts a session successfully", async () => {
    const { result } = renderHook(() => useRestartSession());

    result.current.mutate("session-1");

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
  });
});

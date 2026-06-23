import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@/test/test-utils";
import { http, HttpResponse } from "msw";
import { server } from "@/test/mocks/server";
import { SessionList } from "./SessionList";
import { useSessionsStore } from "../stores/sessions.store";
import { useNotificationStore } from "../stores/notification.store";
import type { Session } from "../types";

// These tests document the expected behavior for Phase 2 UI states.
// The `provisioning` and `failed` statuses (plus `error_message` field) will
// be added to the Session type when the backend implementation is merged.
type ExtendedStatus = Session["status"] | "provisioning" | "failed";
type ExtendedSession = Session & { error_message?: string };

const baseSession: Session = {
  id: "session-1",
  name: "Test Session",
  working_dir: "/tmp/test",
  status: "running",
  pid: 42,
  template_id: "tmpl-1",
  skip_permissions: false,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

function mockSessions(sessions: ExtendedSession[]) {
  server.use(
    http.get("/api/sessions", () => {
      return HttpResponse.json(sessions);
    }),
  );
}

describe("SessionList", () => {
  beforeEach(() => {
    useSessionsStore.setState({ activeSessionId: null });
    useNotificationStore.setState({ notifications: {} });
  });

  describe("provisioning session", () => {
    const provisioningSession: ExtendedSession = {
      ...baseSession,
      id: "session-prov",
      name: "Worktree Session",
      status: "provisioning" as ExtendedStatus as Session["status"],
    };

    beforeEach(() => {
      mockSessions([provisioningSession]);
    });

    it("shows 'Creating worktree...' subtitle", async () => {
      render(<SessionList />);
      const subtitle = await screen.findByText("Creating worktree...");
      expect(subtitle).toBeInTheDocument();
    });

    it("button is disabled while provisioning", async () => {
      render(<SessionList />);
      const listItem = await screen.findByText("Worktree Session");
      const button = listItem.closest("button");
      expect(button).toBeDisabled();
    });
  });

  describe("failed session", () => {
    const failedSession: ExtendedSession = {
      ...baseSession,
      id: "session-fail",
      name: "Failed Session",
      status: "failed" as ExtendedStatus as Session["status"],
      error_message: "Failed to create worktree: branch already exists",
    };

    beforeEach(() => {
      mockSessions([failedSession]);
    });

    it("shows error message as subtitle", async () => {
      render(<SessionList />);
      const errorText = await screen.findByText(
        "Failed to create worktree: branch already exists",
      );
      expect(errorText).toBeInTheDocument();
    });

    it("error subtitle has red styling", async () => {
      render(<SessionList />);
      const errorText = await screen.findByText(
        "Failed to create worktree: branch already exists",
      );
      // Should have a red color class or inline style
      expect(errorText).toHaveClass("text-red-400");
    });

    it("only delete button is shown (no resume/restart)", async () => {
      render(<SessionList />);
      // Wait for the failed session to render
      await screen.findByText("Failed Session");

      const deleteButton = screen.getByTitle("Delete session");
      expect(deleteButton).toBeInTheDocument();

      expect(screen.queryByTitle("Resume session")).not.toBeInTheDocument();
      expect(
        screen.queryByTitle("Restart session (refresh sandbox template)"),
      ).not.toBeInTheDocument();
    });
  });

  describe("running session", () => {
    const runningSession: Session = {
      ...baseSession,
      id: "session-run",
      name: "Running Session",
      status: "running",
    };

    beforeEach(() => {
      mockSessions([runningSession]);
    });

    it("button is not disabled for running session", async () => {
      render(<SessionList />);
      const sessionName = await screen.findByText("Running Session");
      const button = sessionName.closest("button");
      expect(button).not.toBeDisabled();
    });

    it("shows restart button for running session", async () => {
      render(<SessionList />);
      await screen.findByText("Running Session");
      const restartButton = screen.getByTitle(
        "Restart session (refresh sandbox template)",
      );
      expect(restartButton).toBeInTheDocument();
    });
  });

  describe("stopped session", () => {
    const stoppedSession: Session = {
      ...baseSession,
      id: "session-stop",
      name: "Stopped Session",
      status: "stopped",
    };

    beforeEach(() => {
      mockSessions([stoppedSession]);
    });

    it("shows resume button for stopped session", async () => {
      render(<SessionList />);
      await screen.findByText("Stopped Session");
      const resumeButton = screen.getByTitle("Resume session");
      expect(resumeButton).toBeInTheDocument();
    });
  });
});

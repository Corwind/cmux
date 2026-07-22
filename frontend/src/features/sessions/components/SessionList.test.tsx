import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@/test/test-utils";
import { http, HttpResponse } from "msw";
import { server } from "@/test/mocks/server";
import { SessionList } from "./SessionList";
import { useSessionsStore } from "../stores/sessions.store";
import { useNotificationStore } from "../stores/notification.store";
import type { Session } from "../types";
import type { Harness } from "@/features/harnesses";

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
  harness_type: "claude",
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

function mockHarnesses(harnesses: Harness[]) {
  server.use(
    http.get("/api/harnesses", () => {
      return HttpResponse.json(harnesses);
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

    it("row is not a native <button>, so nested action buttons stay clickable", async () => {
      render(<SessionList />);
      const listItem = await screen.findByText("Worktree Session");
      // Nesting the delete <button> inside a row <button> is invalid HTML and
      // the browser drops clicks on the inner button — which made stuck
      // provisioning sessions impossible to delete. The row must therefore be
      // a non-button element (div[role=button]).
      expect(listItem.closest("button")).toBeNull();
      expect(listItem.closest('[role="button"]')).not.toBeNull();
    });

    it("shows a clickable delete button that fires the delete request", async () => {
      let deletedId: string | null = null;
      server.use(
        http.delete("/api/sessions/:id", ({ params }) => {
          deletedId = params.id as string;
          return new HttpResponse(null, { status: 204 });
        }),
      );

      render(<SessionList />);
      await screen.findByText("Worktree Session");

      const deleteButton = screen.getByTitle("Delete session");
      expect(deleteButton).not.toBeDisabled();
      deleteButton.click();

      await vi.waitFor(() => expect(deletedId).toBe("session-prov"));
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

    it("row is interactive (focusable, not aria-disabled) for a running session", async () => {
      render(<SessionList />);
      const sessionName = await screen.findByText("Running Session");
      const row = sessionName.closest('[role="button"]');
      expect(row).not.toBeNull();
      expect(row).toHaveAttribute("tabindex", "0");
      expect(row).toHaveAttribute("aria-disabled", "false");
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

  describe("harness sections", () => {
    it("groups sessions under their harness section name and orders sections by the registry", async () => {
      mockHarnesses([
        { type: "claude", section_name: "Claude Code", is_default: true },
        { type: "codex", section_name: "Codex", is_default: false },
      ]);
      mockSessions([
        { ...baseSession, id: "session-codex", name: "Codex Session", harness_type: "codex" },
        { ...baseSession, id: "session-claude", name: "Claude Session", harness_type: "claude" },
      ]);

      render(<SessionList />);
      await screen.findByText("Claude Session");
      await screen.findByText("Codex Session");

      const sectionHeaders = screen.getAllByText(
        (text) => text === "Claude Code" || text === "Codex",
      );
      expect(sectionHeaders.map((el) => el.textContent)).toEqual(["Claude Code", "Codex"]);
    });

    it("does not render a section for a configured harness with no sessions", async () => {
      mockHarnesses([
        { type: "claude", section_name: "Claude Code", is_default: true },
        { type: "codex", section_name: "Codex", is_default: false },
      ]);
      mockSessions([{ ...baseSession, harness_type: "claude" }]);

      render(<SessionList />);
      await screen.findByText("Test Session");

      expect(screen.getByText("Claude Code")).toBeInTheDocument();
      expect(screen.queryByText("Codex")).not.toBeInTheDocument();
    });

    it("falls back to the raw harness_type as the section header when it's absent from the harness list", async () => {
      mockHarnesses([{ type: "claude", section_name: "Claude Code", is_default: true }]);
      mockSessions([
        { ...baseSession, id: "session-claude", harness_type: "claude" },
        { ...baseSession, id: "session-unknown", name: "Unknown Harness Session", harness_type: "gemini" },
      ]);

      render(<SessionList />);
      await screen.findByText("Test Session");
      await screen.findByText("Unknown Harness Session");

      expect(screen.getByText("Claude Code")).toBeInTheDocument();
      expect(screen.getByText("gemini")).toBeInTheDocument();
    });
  });
});

import { describe, it, expect } from "vitest";
import { render, screen } from "@/test/test-utils";
import { StatusBadge } from "./StatusBadge";
import type { Session } from "../types";

// These tests document the expected behavior for Phase 2 UI states.
// The `provisioning` and `failed` statuses will be added to the Session type
// when the backend implementation is merged.
type ExtendedStatus = Session["status"] | "provisioning" | "failed";

describe("StatusBadge", () => {
  it("renders green dot for 'running' status with no notification", () => {
    render(<StatusBadge status="running" />);
    const badge = screen.getByLabelText("running");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass("bg-green-400");
    expect(badge).not.toHaveClass("animate-pulse");
  });

  it("renders orange pulsing dot when hasNotification=true", () => {
    render(<StatusBadge status="running" hasNotification={true} />);
    const badge = screen.getByLabelText("needs attention");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass("animate-pulse");
    expect(badge).toHaveClass("bg-yellow-400");
  });

  it("renders blue pulsing dot for 'provisioning' status", () => {
    const status = "provisioning" as ExtendedStatus as Session["status"];
    render(<StatusBadge status={status} />);
    const badge = screen.getByLabelText("provisioning");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass("animate-pulse");
    expect(badge).toHaveClass("bg-blue-400");
  });

  it("renders red dot for 'failed' status", () => {
    const status = "failed" as ExtendedStatus as Session["status"];
    render(<StatusBadge status={status} />);
    const badge = screen.getByLabelText("failed");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass("bg-red-400");
  });

  it("renders gray dot for 'stopped' status", () => {
    render(<StatusBadge status="stopped" />);
    const badge = screen.getByLabelText("stopped");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass("bg-gray-500");
    expect(badge).not.toHaveClass("animate-pulse");
  });
});

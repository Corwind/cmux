import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@/test/test-utils";
import { WorktreePanel } from "./WorktreePanel";
import type { WorktreeEntry } from "../types";

// Spy for the delete mutation's `mutate`, hoisted so it can be referenced
// inside the (hoisted) vi.mock factory below.
const { mutateSpy } = vi.hoisted(() => ({ mutateSpy: vi.fn() }));

vi.mock("../hooks/useWorktrees", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../hooks/useWorktrees")>();
  return {
    ...actual,
    useWorktrees: vi.fn(),
    useDeleteWorktree: () => ({ mutate: mutateSpy }),
  };
});

import { useWorktrees } from "../hooks/useWorktrees";

const useWorktreesMock = vi.mocked(useWorktrees);

const baseEntry: WorktreeEntry = {
  id: "wt-1",
  path: "/repo/.worktrees/feature",
  branch: "feature",
  repo_root: "/repo",
  status: "ready",
  created_at: "2026-01-01T00:00:00Z",
};

function mockWorktrees(entries: WorktreeEntry[]) {
  useWorktreesMock.mockReturnValue({
    data: entries,
    isLoading: false,
  } as ReturnType<typeof useWorktrees>);
}

describe("WorktreePanel", () => {
  beforeEach(() => {
    mutateSpy.mockReset();
    useWorktreesMock.mockReset();
  });

  it("renders a ready worktree with the delete button enabled", () => {
    mockWorktrees([baseEntry]);
    render(<WorktreePanel />);

    const deleteButton = screen.getByTitle("Delete worktree");
    expect(deleteButton).not.toBeDisabled();
  });

  it("renders a deleting worktree with the delete button disabled and a removing title", () => {
    mockWorktrees([{ ...baseEntry, status: "deleting" }]);
    render(<WorktreePanel />);

    const deleteButton = screen.getByTitle("Removing worktree...");
    expect(deleteButton).toBeDisabled();
  });

  it("renders a linked worktree with the delete button disabled", () => {
    mockWorktrees([
      {
        ...baseEntry,
        session_id: "session-1",
        session_status: "running",
      },
    ]);
    render(<WorktreePanel />);

    const deleteButton = screen.getByTitle(
      "Delete the session first to remove this worktree",
    );
    expect(deleteButton).toBeDisabled();
  });

  it("calls the delete mutation with the correct id when deleting a ready worktree", () => {
    mockWorktrees([baseEntry]);
    render(<WorktreePanel />);

    const deleteButton = screen.getByTitle("Delete worktree");
    deleteButton.click();

    expect(mutateSpy).toHaveBeenCalledWith("wt-1");
  });
});

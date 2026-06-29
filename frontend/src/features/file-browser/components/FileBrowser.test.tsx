import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent } from "@/test/test-utils";
import { FileBrowser } from "./FileBrowser";
import type { DirEntry } from "@/features/sessions";

vi.mock("../hooks/useFileBrowser", () => ({
  useFileBrowser: vi.fn(),
}));

import { useFileBrowser } from "../hooks/useFileBrowser";

const useFileBrowserMock = vi.mocked(useFileBrowser);

function mockEntries(entries: DirEntry[]) {
  useFileBrowserMock.mockReturnValue({
    data: { path: "/home/user", entries },
    isLoading: false,
    error: null,
  } as ReturnType<typeof useFileBrowser>);
}

const realDir: DirEntry = { name: "realdir", is_dir: true };
const linkDir: DirEntry = { name: "linkdir", is_dir: true, is_symlink: true };
const linkFile: DirEntry = { name: "linkfile", is_dir: false, is_symlink: true };
const broken: DirEntry = { name: "broken", is_dir: false, is_symlink: true, is_broken: true };
const realFile: DirEntry = { name: "realfile.txt", is_dir: false };

describe("FileBrowser symlink support", () => {
  beforeEach(() => {
    useFileBrowserMock.mockReset();
  });

  it("renders a symlinked directory as a navigable entry with a symlink title", () => {
    mockEntries([realDir, linkDir]);
    render(<FileBrowser onSelect={vi.fn()} onClose={vi.fn()} />);

    const link = screen.getByTitle("linkdir (symlink)");
    expect(link).toBeInTheDocument();
    expect(link.tagName).toBe("BUTTON");
    expect(link).not.toBeDisabled();
  });

  it("navigates into a symlinked directory on click", () => {
    mockEntries([linkDir]);
    render(<FileBrowser onSelect={vi.fn()} onClose={vi.fn()} />);

    fireEvent.click(screen.getByTitle("linkdir (symlink)"));
    // After navigating, the path bar (and Select button) reflect the child path.
    expect(screen.getAllByText("/home/user/linkdir").length).toBeGreaterThan(0);
  });

  it("always shows a broken symlink even when files are hidden, and it is not a button", () => {
    mockEntries([realDir, broken]);
    render(<FileBrowser onSelect={vi.fn()} onClose={vi.fn()} showFiles={false} />);

    const brokenEl = screen.getByTitle("broken (broken symlink)");
    expect(brokenEl).toBeInTheDocument();
    expect(brokenEl.tagName).not.toBe("BUTTON");
  });

  it("does not select a broken symlink on click", () => {
    const onSelect = vi.fn();
    mockEntries([broken]);
    render(<FileBrowser onSelect={onSelect} onClose={vi.fn()} showFiles />);

    screen.getByTitle("broken (broken symlink)").click();
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("hides regular files and valid file symlinks when showFiles is false", () => {
    mockEntries([realDir, realFile, linkFile]);
    render(<FileBrowser onSelect={vi.fn()} onClose={vi.fn()} showFiles={false} />);

    expect(screen.queryByText("realfile.txt")).not.toBeInTheDocument();
    expect(screen.queryByTitle("linkfile (symlink)")).not.toBeInTheDocument();
  });

  it("selects a valid symlinked file when showFiles is true", () => {
    const onSelect = vi.fn();
    mockEntries([linkFile]);
    render(<FileBrowser onSelect={onSelect} onClose={vi.fn()} showFiles />);

    screen.getByTitle("linkfile (symlink)").click();
    expect(onSelect).toHaveBeenCalledWith("/home/user/linkfile");
  });
});

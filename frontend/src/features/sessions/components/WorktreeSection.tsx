import { useState } from "react";
import type { GitInfo } from "@/features/git";
import type { WorktreeInput } from "../types";

type WorktreeMode = "main" | "new" | "existing";

interface WorktreeSectionProps {
  gitInfo: GitInfo;
  worktreesDir?: string;
  onChange: (worktree: WorktreeInput | undefined, workingDir: string) => void;
  repoRoot: string;
}

export function WorktreeSection({
  gitInfo,
  onChange,
  repoRoot,
}: WorktreeSectionProps) {
  const [mode, setMode] = useState<WorktreeMode>("main");
  const [createBranch, setCreateBranch] = useState(true);
  const [newBranchName, setNewBranchName] = useState("");
  const [baseRef, setBaseRef] = useState(gitInfo.current_branch ?? "");
  const [existingBranch, setExistingBranch] = useState(
    gitInfo.branches?.find((b) => !b.is_current)?.name ?? "",
  );
  const [customPath, setCustomPath] = useState("");
  const [existingWorktree, setExistingWorktree] = useState(
    gitInfo.worktrees?.find((w) => !w.is_main)?.path ?? "",
  );

  const localBranches = gitInfo.branches?.filter((b) => !b.is_remote) ?? [];
  const linkedWorktrees = gitInfo.worktrees?.filter((w) => !w.is_main) ?? [];

  function handleModeChange(newMode: WorktreeMode) {
    setMode(newMode);
    if (newMode === "main") {
      onChange(undefined, repoRoot);
    } else if (newMode === "existing" && existingWorktree) {
      onChange(undefined, existingWorktree);
    } else {
      notifyNewWorktree(newMode, createBranch, newBranchName, baseRef, existingBranch, customPath);
    }
  }

  function notifyNewWorktree(
    m: WorktreeMode,
    cb: boolean,
    nb: string,
    br: string,
    eb: string,
    cp: string,
  ) {
    if (m !== "new") return;
    const branch = cb ? nb : eb;
    if (!branch) return;
    const input: WorktreeInput = {
      repo_path: repoRoot,
      branch,
      create_branch: cb,
      base_ref: cb ? br : undefined,
      path: cp || undefined,
    };
    onChange(input, repoRoot);
  }

  const inputStyle = {
    backgroundColor: "var(--cmux-sidebar)",
    border: "1px solid var(--cmux-border-light)",
    color: "var(--cmux-text)",
  };

  const labelStyle = { color: "var(--cmux-text-muted)" };

  return (
    <div className="space-y-2">
      <div className="flex gap-1">
        {(["main", "new", "existing"] as WorktreeMode[]).map((m) => (
          <button
            key={m}
            type="button"
            onClick={() => handleModeChange(m)}
            className="rounded px-2 py-1 text-xs transition-colors"
            style={{
              backgroundColor:
                mode === m ? "var(--cmux-accent-button)" : "var(--cmux-sidebar)",
              color: mode === m ? "var(--cmux-text)" : "var(--cmux-text-muted)",
              border: "1px solid var(--cmux-border-light)",
            }}
          >
            {m === "main" ? "Main checkout" : m === "new" ? "New worktree" : "Existing worktree"}
          </button>
        ))}
      </div>

      {mode === "new" && (
        <div className="space-y-2 pl-1">
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => {
                const next = !createBranch;
                setCreateBranch(next);
                notifyNewWorktree("new", next, newBranchName, baseRef, existingBranch, customPath);
              }}
              className="rounded px-2 py-1 text-xs"
              style={{
                backgroundColor: createBranch ? "var(--cmux-accent-button)" : "var(--cmux-sidebar)",
                color: createBranch ? "var(--cmux-text)" : "var(--cmux-text-muted)",
                border: "1px solid var(--cmux-border-light)",
              }}
            >
              New branch
            </button>
            <button
              type="button"
              onClick={() => {
                const next = !createBranch;
                setCreateBranch(next);
                notifyNewWorktree("new", next, newBranchName, baseRef, existingBranch, customPath);
              }}
              className="rounded px-2 py-1 text-xs"
              style={{
                backgroundColor: !createBranch ? "var(--cmux-accent-button)" : "var(--cmux-sidebar)",
                color: !createBranch ? "var(--cmux-text)" : "var(--cmux-text-muted)",
                border: "1px solid var(--cmux-border-light)",
              }}
            >
              Existing branch
            </button>
          </div>

          {createBranch ? (
            <div className="space-y-2">
              <div>
                <label className="mb-1 block text-xs font-medium" style={labelStyle}>
                  Branch name
                </label>
                <input
                  type="text"
                  value={newBranchName}
                  onChange={(e) => {
                    setNewBranchName(e.target.value);
                    notifyNewWorktree("new", true, e.target.value, baseRef, existingBranch, customPath);
                  }}
                  placeholder="feature/my-branch"
                  className="w-full rounded px-2.5 py-1.5 text-sm outline-none"
                  style={inputStyle}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium" style={labelStyle}>
                  Base ref
                </label>
                <select
                  value={baseRef}
                  onChange={(e) => {
                    setBaseRef(e.target.value);
                    notifyNewWorktree("new", true, newBranchName, e.target.value, existingBranch, customPath);
                  }}
                  className="w-full rounded px-2.5 py-1.5 text-sm outline-none"
                  style={inputStyle}
                >
                  {localBranches.map((b) => (
                    <option key={b.name} value={b.name}>
                      {b.name}{b.is_current ? " (current)" : ""}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          ) : (
            <div>
              <label className="mb-1 block text-xs font-medium" style={labelStyle}>
                Branch
              </label>
              <select
                value={existingBranch}
                onChange={(e) => {
                  setExistingBranch(e.target.value);
                  notifyNewWorktree("new", false, newBranchName, baseRef, e.target.value, customPath);
                }}
                className="w-full rounded px-2.5 py-1.5 text-sm outline-none"
                style={inputStyle}
              >
                {localBranches.map((b) => (
                  <option key={b.name} value={b.name}>
                    {b.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          <div>
            <label className="mb-1 block text-xs font-medium" style={labelStyle}>
              Worktree path <span style={{ color: "var(--cmux-text-faint)" }}>(optional, auto-computed)</span>
            </label>
            <input
              type="text"
              value={customPath}
              onChange={(e) => {
                setCustomPath(e.target.value);
                notifyNewWorktree("new", createBranch, newBranchName, baseRef, existingBranch, e.target.value);
              }}
              placeholder="~/.cmux/worktrees/repo/branch"
              className="w-full rounded px-2.5 py-1.5 text-sm outline-none"
              style={inputStyle}
            />
          </div>
        </div>
      )}

      {mode === "existing" && (
        <div className="pl-1">
          {linkedWorktrees.length === 0 ? (
            <p className="text-xs" style={labelStyle}>No linked worktrees found.</p>
          ) : (
            <div>
              <label className="mb-1 block text-xs font-medium" style={labelStyle}>
                Worktree
              </label>
              <select
                value={existingWorktree}
                onChange={(e) => {
                  setExistingWorktree(e.target.value);
                  onChange(undefined, e.target.value);
                }}
                className="w-full rounded px-2.5 py-1.5 text-sm outline-none"
                style={inputStyle}
              >
                {linkedWorktrees.map((wt) => (
                  <option key={wt.path} value={wt.path}>
                    {wt.path}{wt.branch ? ` — ${wt.branch}` : ""}
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

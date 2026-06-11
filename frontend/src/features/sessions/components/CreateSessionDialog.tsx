import { useState, useCallback, useEffect } from "react";
import { useCreateSession } from "../hooks/useCreateSession";
import { useSessionsStore } from "../stores/sessions.store";
import { FileBrowser } from "@/features/file-browser";
import { TemplateSelector } from "@/features/templates";
import { useGitInfo } from "@/features/git";
import type { WorktreeInput, CreateSessionInput } from "../types";
import { WorktreeSection } from "./WorktreeSection";

export function CreateSessionDialog() {
  const [isOpen, setIsOpen] = useState(false);
  const [name, setName] = useState("");
  const [directory, setDirectory] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [skipPermissions, setSkipPermissions] = useState(false);
  const [showFileBrowser, setShowFileBrowser] = useState(false);
  const [debouncedDir, setDebouncedDir] = useState("");
  const [worktreeInput, setWorktreeInput] = useState<WorktreeInput | undefined>(undefined);
  const [effectiveWorkingDir, setEffectiveWorkingDir] = useState("");

  const handleTemplateChange = useCallback((id: string) => setTemplateId(id), []);
  const createSession = useCreateSession();
  const setActiveSession = useSessionsStore((s) => s.setActiveSession);

  // Debounce directory changes for git info lookup
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedDir(directory.trim()), 400);
    return () => clearTimeout(timer);
  }, [directory]);

  // Reset working dir to directory when not in worktree mode
  useEffect(() => {
    if (!worktreeInput) {
      setEffectiveWorkingDir(directory.trim());
    }
  }, [directory, worktreeInput]);

  const { data: gitInfo } = useGitInfo(debouncedDir || undefined);

  function handleWorktreeChange(wt: WorktreeInput | undefined, workingDir: string) {
    setWorktreeInput(wt);
    setEffectiveWorkingDir(workingDir);
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const workingDir = effectiveWorkingDir || directory.trim();
    if (!workingDir) return;

    const input: CreateSessionInput = { working_dir: workingDir };
    if (name.trim()) input.name = name.trim();
    if (templateId) input.template_id = templateId;
    if (skipPermissions) input.skip_permissions = true;
    if (worktreeInput) input.worktree = worktreeInput;

    createSession.mutate(input, {
      onSuccess: (session) => {
        setActiveSession(session.id);
        setName("");
        setDirectory("");
        setTemplateId("");
        setSkipPermissions(false);
        setWorktreeInput(undefined);
        setEffectiveWorkingDir("");
        setIsOpen(false);
      },
    });
  }

  const isWorktreeMode = !!worktreeInput;
  const canSubmit = !createSession.isPending && (directory.trim() || effectiveWorkingDir);

  if (!isOpen) {
    return (
      <button
        type="button"
        onClick={() => setIsOpen(true)}
        className="flex w-full items-center justify-center gap-1.5 rounded px-3 py-2 text-sm font-medium text-white transition-colors"
        style={{ backgroundColor: "var(--cmux-accent-button)" }}
        onMouseEnter={(e) => {
          e.currentTarget.style.backgroundColor = "var(--cmux-accent-button-hover)";
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = "var(--cmux-accent-button)";
        }}
      >
        <svg
          className="h-4 w-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M12 4v16m8-8H4"
          />
        </svg>
        New Session
      </button>
    );
  }

  return (
    <>
      <form
        onSubmit={handleSubmit}
        className="space-y-3 rounded-lg p-3"
        style={{
          backgroundColor: "var(--cmux-surface)",
          border: "1px solid var(--cmux-border-light)",
        }}
      >
        <div>
          <label
            htmlFor="session-name"
            className="mb-1 block text-xs font-medium"
            style={{ color: "var(--cmux-text-muted)" }}
          >
            Name <span style={{ color: "var(--cmux-text-faint)" }}>(optional)</span>
          </label>
          <input
            id="session-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="defaults to directory name"
            className="w-full rounded px-2.5 py-1.5 text-sm outline-none"
            style={{
              backgroundColor: "var(--cmux-sidebar)",
              border: "1px solid var(--cmux-border-light)",
              color: "var(--cmux-text)",
            }}
          />
        </div>
        <div>
          <label
            htmlFor="session-dir"
            className="mb-1 block text-xs font-medium"
            style={{ color: "var(--cmux-text-muted)" }}
          >
            Directory
          </label>
          <div className="flex gap-1.5">
            <input
              id="session-dir"
              type="text"
              value={directory}
              onChange={(e) => setDirectory(e.target.value)}
              placeholder="/home/user/project"
              className="min-w-0 flex-1 rounded px-2.5 py-1.5 text-sm outline-none"
              style={{
                backgroundColor: "var(--cmux-sidebar)",
                border: "1px solid var(--cmux-border-light)",
                color: "var(--cmux-text)",
              }}
            />
            <button
              type="button"
              onClick={() => setShowFileBrowser(true)}
              className="rounded px-2 py-1.5 text-sm transition-colors"
              style={{
                border: "1px solid var(--cmux-border-light)",
                color: "var(--cmux-text-muted)",
              }}
              title="Browse directories"
            >
              ...
            </button>
          </div>
        </div>

        {gitInfo?.is_repo && gitInfo.repo_root && (
          <div>
            <label
              className="mb-1 block text-xs font-medium"
              style={{ color: "var(--cmux-text-muted)" }}
            >
              Git worktree
            </label>
            <WorktreeSection
              gitInfo={gitInfo}
              repoRoot={gitInfo.repo_root}
              onChange={handleWorktreeChange}
            />
          </div>
        )}

        <TemplateSelector value={templateId} onChange={handleTemplateChange} />
        <label
          className="flex items-center gap-2 text-xs"
          style={{ color: "var(--cmux-text-muted)" }}
        >
          <input
            type="checkbox"
            checked={skipPermissions}
            onChange={(e) => setSkipPermissions(e.target.checked)}
            className="accent-green-500"
          />
          Skip permissions (--dangerously-skip-permissions)
        </label>
        <div className="flex gap-2">
          <button
            type="submit"
            disabled={!canSubmit}
            className="flex-1 rounded py-1.5 text-sm font-medium text-white transition-colors disabled:opacity-50"
            style={{ backgroundColor: "var(--cmux-accent-button)" }}
          >
            {createSession.isPending
              ? isWorktreeMode
                ? "Creating worktree..."
                : "Creating..."
              : "Create"}
          </button>
          <button
            type="button"
            onClick={() => setIsOpen(false)}
            className="rounded px-3 py-1.5 text-sm transition-colors"
            style={{
              border: "1px solid var(--cmux-border-light)",
              color: "var(--cmux-text-muted)",
            }}
          >
            Cancel
          </button>
        </div>
      </form>

      {showFileBrowser && (
        <FileBrowser
          onSelect={(path) => {
            setDirectory(path);
            setShowFileBrowser(false);
          }}
          onClose={() => setShowFileBrowser(false)}
        />
      )}
    </>
  );
}

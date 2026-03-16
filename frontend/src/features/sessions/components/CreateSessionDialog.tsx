import { useState, useCallback } from "react";
import { useCreateSession } from "../hooks/useCreateSession";
import { useSessionsStore } from "../stores/sessions.store";
import { FileBrowser } from "@/features/file-browser";
import { TemplateSelector } from "@/features/templates";

export function CreateSessionDialog() {
  const [isOpen, setIsOpen] = useState(false);
  const [name, setName] = useState("");
  const [directory, setDirectory] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [skipPermissions, setSkipPermissions] = useState(false);
  const [showFileBrowser, setShowFileBrowser] = useState(false);
  const handleTemplateChange = useCallback((id: string) => setTemplateId(id), []);
  const createSession = useCreateSession();
  const setActiveSession = useSessionsStore((s) => s.setActiveSession);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!directory.trim()) return;

    const input: { name?: string; working_dir: string; template_id?: string; skip_permissions?: boolean } = {
      working_dir: directory.trim(),
    };
    if (name.trim()) {
      input.name = name.trim();
    }
    if (templateId) {
      input.template_id = templateId;
    }
    if (skipPermissions) {
      input.skip_permissions = true;
    }

    createSession.mutate(input, {
      onSuccess: (session) => {
        setActiveSession(session.id);
        setName("");
        setDirectory("");
        setTemplateId("");
        setSkipPermissions(false);
        setIsOpen(false);
      },
    });
  }

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
            disabled={createSession.isPending || !directory.trim()}
            className="flex-1 rounded py-1.5 text-sm font-medium text-white transition-colors disabled:opacity-50"
            style={{ backgroundColor: "var(--cmux-accent-button)" }}
          >
            {createSession.isPending ? "Creating..." : "Create"}
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

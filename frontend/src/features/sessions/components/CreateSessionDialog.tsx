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
  const [showFileBrowser, setShowFileBrowser] = useState(false);
  const handleTemplateChange = useCallback((id: string) => setTemplateId(id), []);
  const createSession = useCreateSession();
  const setActiveSession = useSessionsStore((s) => s.setActiveSession);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!directory.trim()) return;

    const input: { name?: string; working_dir: string; template_id?: string } = {
      working_dir: directory.trim(),
    };
    if (name.trim()) {
      input.name = name.trim();
    }
    if (templateId) {
      input.template_id = templateId;
    }

    createSession.mutate(input, {
      onSuccess: (session) => {
        setActiveSession(session.id);
        setName("");
        setDirectory("");
        setTemplateId("");
        setIsOpen(false);
      },
    });
  }

  if (!isOpen) {
    return (
      <button
        type="button"
        onClick={() => setIsOpen(true)}
        className="flex w-full items-center justify-center gap-1.5 rounded bg-green-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-green-500"
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
        className="space-y-3 rounded-lg border border-gray-700 bg-gray-800 p-3"
      >
        <div>
          <label
            htmlFor="session-name"
            className="mb-1 block text-xs font-medium text-gray-400"
          >
            Name <span className="text-gray-500">(optional)</span>
          </label>
          <input
            id="session-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="defaults to directory name"
            className="w-full rounded border border-gray-600 bg-gray-900 px-2.5 py-1.5 text-sm text-white placeholder-gray-500 focus:border-green-500 focus:outline-none"
          />
        </div>
        <div>
          <label
            htmlFor="session-dir"
            className="mb-1 block text-xs font-medium text-gray-400"
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
              className="min-w-0 flex-1 rounded border border-gray-600 bg-gray-900 px-2.5 py-1.5 text-sm text-white placeholder-gray-500 focus:border-green-500 focus:outline-none"
            />
            <button
              type="button"
              onClick={() => setShowFileBrowser(true)}
              className="rounded border border-gray-600 px-2 py-1.5 text-sm text-gray-400 hover:bg-gray-700 hover:text-white"
              title="Browse directories"
            >
              ...
            </button>
          </div>
        </div>
        <TemplateSelector value={templateId} onChange={handleTemplateChange} />
        <div className="flex gap-2">
          <button
            type="submit"
            disabled={createSession.isPending || !directory.trim()}
            className="flex-1 rounded bg-green-600 py-1.5 text-sm font-medium text-white transition-colors hover:bg-green-500 disabled:opacity-50"
          >
            {createSession.isPending ? "Creating..." : "Create"}
          </button>
          <button
            type="button"
            onClick={() => setIsOpen(false)}
            className="rounded border border-gray-600 px-3 py-1.5 text-sm text-gray-400 transition-colors hover:text-white"
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

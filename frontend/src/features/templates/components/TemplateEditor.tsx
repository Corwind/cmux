import { useState } from "react";
import type { SandboxTemplate } from "../types";

interface TemplateEditorProps {
  template?: SandboxTemplate;
  onSave: (name: string, content: string) => void;
  onCancel: () => void;
  isPending: boolean;
  error?: string;
}

export function TemplateEditor({
  template,
  onSave,
  onCancel,
  isPending,
  error,
}: TemplateEditorProps) {
  const [name, setName] = useState(template?.name ?? "");
  const [content, setContent] = useState(template?.content ?? "");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim() || !content.trim()) return;
    onSave(name.trim(), content.trim());
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-3 rounded-lg border border-gray-700 bg-gray-800 p-3"
    >
      <div>
        <label
          htmlFor="template-name"
          className="mb-1 block text-xs font-medium text-gray-400"
        >
          Name
        </label>
        <input
          id="template-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="my-sandbox-profile"
          className="w-full rounded border border-gray-600 bg-gray-900 px-2.5 py-1.5 text-sm text-white placeholder-gray-500 focus:border-green-500 focus:outline-none"
        />
      </div>
      <div>
        <label
          htmlFor="template-content"
          className="mb-1 block text-xs font-medium text-gray-400"
        >
          SBPL Content
        </label>
        <textarea
          id="template-content"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="(version 1)&#10;(allow default)&#10;..."
          rows={12}
          className="w-full rounded border border-gray-600 bg-gray-900 px-2.5 py-1.5 font-mono text-sm text-white placeholder-gray-500 focus:border-green-500 focus:outline-none"
        />
      </div>
      {error && <p className="text-xs text-red-400">{error}</p>}
      <div className="flex gap-2">
        <button
          type="submit"
          disabled={isPending || !name.trim() || !content.trim()}
          className="flex-1 rounded bg-green-600 py-1.5 text-sm font-medium text-white transition-colors hover:bg-green-500 disabled:opacity-50"
        >
          {isPending ? "Saving..." : "Save"}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded border border-gray-600 px-3 py-1.5 text-sm text-gray-400 transition-colors hover:text-white"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}

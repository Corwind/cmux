import { useState } from "react";
import type { SandboxTemplate } from "../types";
import { type SandboxRule, rulesToSbpl, sbplToRules } from "../utils/sbpl";
import { FileBrowser } from "@/features/file-browser";

interface TemplateEditorProps {
  template?: SandboxTemplate;
  onSave: (name: string, content: string) => void;
  onCancel: () => void;
  isPending: boolean;
  error?: string;
}

function emptyRule(): SandboxRule {
  return { path: "", read: true, write: false, metadata: false };
}

export function TemplateEditor({
  template,
  onSave,
  onCancel,
  isPending,
  error,
}: TemplateEditorProps) {
  const [name, setName] = useState(template?.name ?? "");
  const [advanced, setAdvanced] = useState(false);
  const [rawContent, setRawContent] = useState(template?.content ?? "");
  const [rules, setRules] = useState<SandboxRule[]>(() => {
    if (template?.content) {
      const parsed = sbplToRules(template.content);
      return parsed.length > 0 ? parsed : [emptyRule()];
    }
    return [emptyRule()];
  });
  const [browsingRuleIndex, setBrowsingRuleIndex] = useState<number | null>(
    null,
  );

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;

    if (advanced) {
      if (!rawContent.trim()) return;
      onSave(name.trim(), rawContent.trim());
    } else {
      const validRules = rules.filter(
        (r) => r.path && (r.read || r.write || r.metadata),
      );
      if (validRules.length === 0) return;
      onSave(name.trim(), rulesToSbpl(validRules));
    }
  }

  function updateRule(index: number, update: Partial<SandboxRule>) {
    setRules((prev) =>
      prev.map((rule, i) => (i === index ? { ...rule, ...update } : rule)),
    );
  }

  function removeRule(index: number) {
    setRules((prev) => (prev.length === 1 ? prev : prev.filter((_, i) => i !== index)));
  }

  function addRule() {
    setRules((prev) => [...prev, emptyRule()]);
  }

  function switchToAdvanced() {
    setRawContent(rulesToSbpl(rules));
    setAdvanced(true);
  }

  function switchToBuilder() {
    const parsed = sbplToRules(rawContent);
    setRules(parsed.length > 0 ? parsed : [emptyRule()]);
    setAdvanced(false);
  }

  const hasValidContent = advanced
    ? rawContent.trim().length > 0
    : rules.some((r) => r.path && (r.read || r.write || r.metadata));

  return (
    <>
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

        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-gray-400">
            {advanced ? "SBPL Content" : "Rules"}
          </span>
          <button
            type="button"
            onClick={advanced ? switchToBuilder : switchToAdvanced}
            className="text-xs text-gray-500 hover:text-gray-300"
          >
            {advanced ? "Rule Builder" : "Advanced"}
          </button>
        </div>

        {advanced ? (
          <textarea
            value={rawContent}
            onChange={(e) => setRawContent(e.target.value)}
            placeholder={'(allow file-read* (subpath "/tmp"))\n...'}
            rows={12}
            className="w-full rounded border border-gray-600 bg-gray-900 px-2.5 py-1.5 font-mono text-sm text-white placeholder-gray-500 focus:border-green-500 focus:outline-none"
          />
        ) : (
          <div className="space-y-2">
            {rules.map((rule, index) => (
              <div
                key={index}
                className="rounded border border-gray-700 bg-gray-900 p-2.5"
              >
                <div className="flex items-center gap-1.5">
                  <label className="sr-only" htmlFor={`rule-path-${index}`}>
                    Path
                  </label>
                  <input
                    id={`rule-path-${index}`}
                    type="text"
                    value={rule.path}
                    onChange={(e) =>
                      updateRule(index, { path: e.target.value })
                    }
                    placeholder="/path/to/allow"
                    className="min-w-0 flex-1 rounded border border-gray-600 bg-gray-800 px-2 py-1 font-mono text-sm text-white placeholder-gray-500 focus:border-green-500 focus:outline-none"
                  />
                  <button
                    type="button"
                    onClick={() => setBrowsingRuleIndex(index)}
                    className="rounded border border-gray-600 px-2 py-1 text-sm text-gray-400 hover:bg-gray-700 hover:text-white"
                    title="Browse directories"
                  >
                    ...
                  </button>
                  <button
                    type="button"
                    onClick={() => removeRule(index)}
                    className="rounded p-1 text-gray-500 hover:bg-gray-700 hover:text-red-400"
                    title="Remove rule"
                  >
                    <svg
                      className="h-3.5 w-3.5"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      strokeWidth={2}
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                </div>
                <div className="mt-2 flex gap-4">
                  <label className="flex items-center gap-1.5 text-xs text-gray-400">
                    <input
                      type="checkbox"
                      checked={rule.read}
                      onChange={(e) =>
                        updateRule(index, { read: e.target.checked })
                      }
                      className="accent-green-500"
                    />
                    Read
                  </label>
                  <label className="flex items-center gap-1.5 text-xs text-gray-400">
                    <input
                      type="checkbox"
                      checked={rule.write}
                      onChange={(e) =>
                        updateRule(index, { write: e.target.checked })
                      }
                      className="accent-green-500"
                    />
                    Write
                  </label>
                  <label className="flex items-center gap-1.5 text-xs text-gray-400">
                    <input
                      type="checkbox"
                      checked={rule.metadata}
                      onChange={(e) =>
                        updateRule(index, { metadata: e.target.checked })
                      }
                      className="accent-green-500"
                    />
                    Metadata
                  </label>
                </div>
              </div>
            ))}
            <button
              type="button"
              onClick={addRule}
              className="flex w-full items-center justify-center gap-1 rounded border border-dashed border-gray-600 py-1.5 text-xs text-gray-500 hover:border-gray-500 hover:text-gray-300"
            >
              <svg
                className="h-3 w-3"
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
              Add Rule
            </button>
          </div>
        )}

        {error && <p className="text-xs text-red-400">{error}</p>}

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={isPending || !name.trim() || !hasValidContent}
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

      {browsingRuleIndex !== null && (
        <FileBrowser
          onSelect={(path) => {
            updateRule(browsingRuleIndex, { path });
            setBrowsingRuleIndex(null);
          }}
          onClose={() => setBrowsingRuleIndex(null)}
        />
      )}
    </>
  );
}

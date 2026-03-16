import { useState } from "react";
import type { SandboxRule, SandboxTemplate } from "../types";
import { rulesToSbpl } from "../utils/sbpl";
import { FileBrowser } from "@/features/file-browser";

interface TemplateEditorProps {
  template?: SandboxTemplate;
  onSave: (name: string, rules: SandboxRule[]) => void;
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
  const [rules, setRules] = useState<SandboxRule[]>(() => {
    if (template?.rules && template.rules.length > 0) {
      return template.rules;
    }
    return [emptyRule()];
  });
  const [browsingRuleIndex, setBrowsingRuleIndex] = useState<number | null>(
    null,
  );

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;

    const validRules = rules.filter(
      (r) => r.path && (r.read || r.write || r.metadata),
    );
    if (validRules.length === 0) return;
    onSave(name.trim(), validRules);
  }

  function updateRule(index: number, update: Partial<SandboxRule>) {
    setRules((prev) =>
      prev.map((rule, i) => (i === index ? { ...rule, ...update } : rule)),
    );
  }

  function removeRule(index: number) {
    setRules((prev) =>
      prev.length === 1 ? prev : prev.filter((_, i) => i !== index),
    );
  }

  function addRule() {
    setRules((prev) => [...prev, emptyRule()]);
  }

  const hasValidRules = rules.some(
    (r) => r.path && (r.read || r.write || r.metadata),
  );

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
            htmlFor="template-name"
            className="mb-1 block text-xs font-medium"
            style={{ color: "var(--cmux-text-muted)" }}
          >
            Name
          </label>
          <input
            id="template-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="my-sandbox-profile"
            className="w-full rounded px-2.5 py-1.5 text-sm outline-none"
            style={{
              backgroundColor: "var(--cmux-sidebar)",
              border: "1px solid var(--cmux-border-light)",
              color: "var(--cmux-text)",
            }}
          />
        </div>

        <div className="flex items-center justify-between">
          <span
            className="text-xs font-medium"
            style={{ color: "var(--cmux-text-muted)" }}
          >
            {advanced ? "SBPL Preview" : "Rules"}
          </span>
          <button
            type="button"
            onClick={() => setAdvanced(!advanced)}
            className="text-xs"
            style={{ color: "var(--cmux-text-faint)" }}
          >
            {advanced ? "Rule Builder" : "Advanced"}
          </button>
        </div>

        {advanced ? (
          <textarea
            value={rulesToSbpl(rules)}
            readOnly
            rows={12}
            className="w-full rounded px-2.5 py-1.5 font-mono text-sm outline-none"
            style={{
              backgroundColor: "var(--cmux-sidebar)",
              border: "1px solid var(--cmux-border-light)",
              color: "var(--cmux-text-muted)",
            }}
          />
        ) : (
          <div className="space-y-2">
            {rules.map((rule, index) => (
              <div
                key={index}
                className="rounded p-2.5"
                style={{
                  backgroundColor: "var(--cmux-sidebar)",
                  border: "1px solid var(--cmux-border)",
                }}
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
                    className="min-w-0 flex-1 rounded px-2 py-1 font-mono text-sm outline-none"
                    style={{
                      backgroundColor: "var(--cmux-surface)",
                      border: "1px solid var(--cmux-border-light)",
                      color: "var(--cmux-text)",
                    }}
                  />
                  <button
                    type="button"
                    onClick={() => setBrowsingRuleIndex(index)}
                    className="rounded px-2 py-1 text-sm transition-colors"
                    style={{
                      border: "1px solid var(--cmux-border-light)",
                      color: "var(--cmux-text-muted)",
                    }}
                    title="Browse directories"
                  >
                    ...
                  </button>
                  <button
                    type="button"
                    onClick={() => removeRule(index)}
                    className="rounded p-1 transition-colors"
                    style={{ color: "var(--cmux-text-muted)" }}
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
                  <label
                    className="flex items-center gap-1.5 text-xs"
                    style={{ color: "var(--cmux-text-muted)" }}
                  >
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
                  <label
                    className="flex items-center gap-1.5 text-xs"
                    style={{ color: "var(--cmux-text-muted)" }}
                  >
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
                  <label
                    className="flex items-center gap-1.5 text-xs"
                    style={{ color: "var(--cmux-text-muted)" }}
                  >
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
              className="flex w-full items-center justify-center gap-1 rounded py-1.5 text-xs"
              style={{
                border: "1px dashed var(--cmux-border-light)",
                color: "var(--cmux-text-faint)",
              }}
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
            disabled={isPending || !name.trim() || !hasValidRules}
            className="flex-1 rounded py-1.5 text-sm font-medium text-white transition-colors disabled:opacity-50"
            style={{ backgroundColor: "var(--cmux-accent-button)" }}
          >
            {isPending ? "Saving..." : "Save"}
          </button>
          <button
            type="button"
            onClick={onCancel}
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

      {browsingRuleIndex !== null && (
        <FileBrowser
          showHidden
          showFiles
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

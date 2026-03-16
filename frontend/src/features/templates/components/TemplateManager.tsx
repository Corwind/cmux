import { useState } from "react";
import { useTemplates } from "../hooks/useTemplates";
import { useCreateTemplate } from "../hooks/useCreateTemplate";
import { useUpdateTemplate } from "../hooks/useUpdateTemplate";
import { useDeleteTemplate } from "../hooks/useDeleteTemplate";
import {
  setDefaultTemplate,
  clearDefaultTemplate,
  importTemplate,
  exportTemplate,
} from "../services/templates-api";
import { templateKeys } from "../hooks/useTemplates";
import { useQueryClient } from "@tanstack/react-query";
import { TemplateEditor } from "./TemplateEditor";
import type { SandboxRule, SandboxTemplate } from "../types";

type EditorMode =
  | { kind: "closed" }
  | { kind: "create" }
  | { kind: "edit"; template: SandboxTemplate };

export function TemplateManager() {
  const { data: templates, isLoading } = useTemplates();
  const createTemplate = useCreateTemplate();
  const updateTemplate = useUpdateTemplate();
  const deleteTemplateMutation = useDeleteTemplate();
  const queryClient = useQueryClient();
  const [editor, setEditor] = useState<EditorMode>({ kind: "closed" });
  const [error, setError] = useState("");

  function handleCreate(name: string, rules: SandboxRule[]) {
    setError("");
    createTemplate.mutate(
      { name, rules },
      {
        onSuccess: () => setEditor({ kind: "closed" }),
        onError: (err) => setError(err.message),
      },
    );
  }

  function handleUpdate(id: string, name: string, rules: SandboxRule[]) {
    setError("");
    updateTemplate.mutate(
      { id, input: { name, rules } },
      {
        onSuccess: () => setEditor({ kind: "closed" }),
        onError: (err) => setError(err.message),
      },
    );
  }

  async function handleToggleDefault(template: SandboxTemplate) {
    try {
      if (template.is_default) {
        await clearDefaultTemplate();
      } else {
        await setDefaultTemplate(template.id);
      }
      void queryClient.invalidateQueries({ queryKey: templateKeys.all });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update default");
    }
  }

  async function handleExport(template: SandboxTemplate) {
    try {
      const blob = await exportTemplate(template.id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${template.name}.sbpl`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Export failed");
    }
  }

  function handleImport() {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".sbpl";
    input.onchange = async () => {
      const file = input.files?.[0];
      if (!file) return;
      try {
        const content = await file.text();
        const name = file.name.replace(/\.sbpl$/, "");
        await importTemplate({ name, content });
        void queryClient.invalidateQueries({ queryKey: templateKeys.all });
      } catch (err) {
        setError(err instanceof Error ? err.message : "Import failed");
      }
    };
    input.click();
  }

  if (editor.kind === "create") {
    return (
      <TemplateEditor
        onSave={handleCreate}
        onCancel={() => setEditor({ kind: "closed" })}
        isPending={createTemplate.isPending}
        error={error}
      />
    );
  }

  if (editor.kind === "edit") {
    return (
      <TemplateEditor
        template={editor.template}
        onSave={(name, rules) =>
          handleUpdate(editor.template.id, name, rules)
        }
        onCancel={() => setEditor({ kind: "closed" })}
        isPending={updateTemplate.isPending}
        error={error}
      />
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex gap-1.5">
        <button
          type="button"
          onClick={() => setEditor({ kind: "create" })}
          className="flex flex-1 items-center justify-center gap-1.5 rounded px-3 py-1.5 text-sm font-medium text-white transition-colors"
          style={{ backgroundColor: "var(--cmux-accent-button)" }}
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
              d="M12 4v16m8-8H4"
            />
          </svg>
          New
        </button>
        <button
          type="button"
          onClick={handleImport}
          className="rounded px-3 py-1.5 text-sm transition-colors"
          style={{
            border: "1px solid var(--cmux-border-light)",
            color: "var(--cmux-text-muted)",
          }}
        >
          Import
        </button>
      </div>

      {error && <p className="text-xs text-red-400">{error}</p>}

      {isLoading && (
        <div className="p-2 text-sm" style={{ color: "var(--cmux-text-muted)" }}>
          Loading templates...
        </div>
      )}

      {templates && templates.length === 0 && (
        <div className="p-2 text-sm" style={{ color: "var(--cmux-text-muted)" }}>
          No templates yet.
        </div>
      )}

      {templates && templates.length > 0 && (
        <ul className="space-y-1">
          {templates.map((template) => (
            <li
              key={template.id}
              className="flex items-center justify-between rounded px-3 py-2 text-sm"
              style={{ color: "var(--cmux-text-secondary)" }}
            >
              <div className="min-w-0 flex-1">
                <span className="truncate font-medium">{template.name}</span>
                {template.is_default && (
                  <span
                    className="ml-1.5 rounded px-1.5 py-0.5 text-xs"
                    style={{
                      backgroundColor: "color-mix(in srgb, var(--cmux-accent) 20%, transparent)",
                      color: "var(--cmux-accent)",
                    }}
                  >
                    default
                  </span>
                )}
              </div>
              <div className="ml-2 flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => handleToggleDefault(template)}
                  className="rounded p-0.5 transition-colors"
                  style={{ color: "var(--cmux-text-muted)" }}
                  title={
                    template.is_default
                      ? "Clear default"
                      : "Set as default"
                  }
                >
                  <svg
                    className="h-3.5 w-3.5"
                    fill={template.is_default ? "currentColor" : "none"}
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2}
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
                    />
                  </svg>
                </button>
                <button
                  type="button"
                  onClick={() => setEditor({ kind: "edit", template })}
                  className="rounded p-0.5 transition-colors"
                  style={{ color: "var(--cmux-text-muted)" }}
                  title="Edit template"
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
                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                    />
                  </svg>
                </button>
                <button
                  type="button"
                  onClick={() => handleExport(template)}
                  className="rounded p-0.5 transition-colors"
                  style={{ color: "var(--cmux-text-muted)" }}
                  title="Export template"
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
                      d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                    />
                  </svg>
                </button>
                <button
                  type="button"
                  onClick={() => deleteTemplateMutation.mutate(template.id)}
                  className="rounded p-0.5 transition-colors"
                  style={{ color: "var(--cmux-text-muted)" }}
                  title="Delete template"
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
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

import { useEffect } from "react";
import { useTemplates } from "../hooks/useTemplates";

interface TemplateSelectorProps {
  value: string;
  onChange: (templateId: string) => void;
}

export function TemplateSelector({ value, onChange }: TemplateSelectorProps) {
  const { data: templates } = useTemplates();

  useEffect(() => {
    if (!value && templates) {
      const defaultTemplate = templates.find((t) => t.is_default);
      if (defaultTemplate) {
        onChange(defaultTemplate.id);
      }
    }
  }, [templates, value, onChange]);

  return (
    <div>
      <label
        htmlFor="template-select"
        className="mb-1 block text-xs font-medium"
        style={{ color: "var(--cmux-text-muted)" }}
      >
        Sandbox Template{" "}
        <span style={{ color: "var(--cmux-text-faint)" }}>(optional)</span>
      </label>
      <select
        id="template-select"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded px-2.5 py-1.5 text-sm outline-none"
        style={{
          backgroundColor: "var(--cmux-sidebar)",
          border: "1px solid var(--cmux-border-light)",
          color: "var(--cmux-text)",
        }}
      >
        <option value="">None</option>
        {templates?.map((template) => (
          <option key={template.id} value={template.id}>
            {template.name}
            {template.is_default ? " (default)" : ""}
          </option>
        ))}
      </select>
    </div>
  );
}

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
        className="mb-1 block text-xs font-medium text-gray-400"
      >
        Sandbox Template <span className="text-gray-500">(optional)</span>
      </label>
      <select
        id="template-select"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded border border-gray-600 bg-gray-900 px-2.5 py-1.5 text-sm text-white focus:border-green-500 focus:outline-none"
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

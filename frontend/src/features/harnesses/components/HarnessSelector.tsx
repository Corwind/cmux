import { useEffect } from "react";
import { useHarnesses } from "../hooks/useHarnesses";

interface HarnessSelectorProps {
  value: string;
  onChange: (harnessType: string) => void;
}

export function HarnessSelector({ value, onChange }: HarnessSelectorProps) {
  const { data: harnesses } = useHarnesses();

  useEffect(() => {
    if (!value && harnesses) {
      const defaultHarness = harnesses.find((h) => h.is_default);
      if (defaultHarness) {
        onChange(defaultHarness.type);
      }
    }
  }, [harnesses, value, onChange]);

  return (
    <div>
      <label
        htmlFor="harness-select"
        className="mb-1 block text-xs font-medium"
        style={{ color: "var(--cmux-text-muted)" }}
      >
        Harness
      </label>
      <select
        id="harness-select"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded px-2.5 py-1.5 text-sm outline-none"
        style={{
          backgroundColor: "var(--cmux-sidebar)",
          border: "1px solid var(--cmux-border-light)",
          color: "var(--cmux-text)",
        }}
      >
        {harnesses?.map((harness) => (
          <option key={harness.type} value={harness.type}>
            {harness.section_name}
            {harness.is_default ? " (default)" : ""}
          </option>
        ))}
      </select>
    </div>
  );
}

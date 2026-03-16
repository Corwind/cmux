import { terminalThemes } from "../themes";
import { useTerminalThemeStore } from "../stores/terminal-theme.store";

export function ThemePicker() {
  const { themeId, setThemeId } = useTerminalThemeStore();

  return (
    <div className="space-y-1">
      <label
        htmlFor="terminal-theme"
        className="block px-1 text-xs font-medium uppercase tracking-wider"
        style={{ color: "var(--cmux-text-muted)" }}
      >
        Terminal Theme
      </label>
      <div className="flex items-center gap-2">
        <select
          id="terminal-theme"
          value={themeId}
          onChange={(e) => setThemeId(e.target.value)}
          className="w-full rounded px-2 py-1.5 text-sm outline-none"
          style={{
            backgroundColor: "var(--cmux-surface)",
            border: "1px solid var(--cmux-border-light)",
            color: "var(--cmux-text-secondary)",
          }}
        >
          {terminalThemes.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </select>
        <div
          className="h-6 w-6 shrink-0 rounded"
          style={{
            background: terminalThemes.find((t) => t.id === themeId)?.theme
              .background,
            border: "1px solid var(--cmux-border-light)",
          }}
          title="Theme preview"
        />
      </div>
    </div>
  );
}

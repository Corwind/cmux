import { terminalThemes } from "../themes";
import { useTerminalThemeStore } from "../stores/terminal-theme.store";

export function ThemePicker() {
  const { themeId, setThemeId } = useTerminalThemeStore();

  return (
    <div className="space-y-1">
      <label
        htmlFor="terminal-theme"
        className="block px-1 text-xs font-medium uppercase tracking-wider text-gray-500"
      >
        Terminal Theme
      </label>
      <div className="flex items-center gap-2">
        <select
          id="terminal-theme"
          value={themeId}
          onChange={(e) => setThemeId(e.target.value)}
          className="w-full rounded border border-gray-700 bg-gray-800 px-2 py-1.5 text-sm text-gray-200 outline-none focus:border-green-500"
        >
          {terminalThemes.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </select>
        <div
          className="h-6 w-6 shrink-0 rounded border border-gray-600"
          style={{
            background: terminalThemes.find((t) => t.id === themeId)?.theme
              .background,
          }}
          title="Theme preview"
        />
      </div>
    </div>
  );
}

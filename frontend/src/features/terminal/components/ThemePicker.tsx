import { terminalThemes } from "../themes";
import { useTerminalThemeStore } from "../stores/terminal-theme.store";
import type { ITheme } from "@xterm/xterm";

/** Shows the key colors from a terminal theme as a horizontal strip. */
function ColorStrip({ theme }: { theme: ITheme }) {
  const colors = [
    theme.background,
    theme.foreground,
    theme.red,
    theme.green,
    theme.yellow,
    theme.blue,
    theme.magenta,
    theme.cyan,
  ].filter(Boolean) as string[];

  return (
    <div className="flex h-3 overflow-hidden rounded-sm">
      {colors.map((color, i) => (
        <div key={i} className="flex-1" style={{ backgroundColor: color }} />
      ))}
    </div>
  );
}

export function ThemePicker() {
  const { themeId, setThemeId } = useTerminalThemeStore();

  return (
    <div className="space-y-2">
      <span
        className="block px-1 text-xs font-medium uppercase tracking-wider"
        style={{ color: "var(--cmux-text-muted)" }}
      >
        Terminal Theme
      </span>
      <div className="space-y-1.5">
        {terminalThemes.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => setThemeId(t.id)}
            className="flex w-full items-center gap-3 rounded px-2.5 py-2 text-left text-sm transition-colors"
            style={{
              backgroundColor:
                themeId === t.id ? "var(--cmux-surface-hover)" : undefined,
              color:
                themeId === t.id
                  ? "var(--cmux-text)"
                  : "var(--cmux-text-secondary)",
              border:
                themeId === t.id
                  ? "1px solid var(--cmux-accent)"
                  : "1px solid transparent",
            }}
          >
            <div className="min-w-0 flex-1">
              <div className="mb-1 text-xs font-medium">{t.name}</div>
              <ColorStrip theme={t.theme} />
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}

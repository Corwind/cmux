import { useMemo } from "react";
import { getTerminalTheme, deriveUiColors } from "../themes";
import { useTerminalThemeStore } from "../stores/terminal-theme.store";

/**
 * Returns a CSSProperties object with --cmux-* custom properties
 * derived from the active terminal theme. Apply to the layout root.
 */
export function useThemeVars(): React.CSSProperties {
  const themeId = useTerminalThemeStore((s) => s.themeId);

  return useMemo(() => {
    const t = getTerminalTheme(themeId);
    const ui = deriveUiColors(t.theme, t.uiOverrides);

    return {
      "--cmux-bg": ui.bg,
      "--cmux-sidebar": ui.sidebar,
      "--cmux-surface": ui.surface,
      "--cmux-surface-hover": ui.surfaceHover,
      "--cmux-border": ui.border,
      "--cmux-border-light": ui.borderLight,
      "--cmux-text": ui.text,
      "--cmux-text-secondary": ui.textSecondary,
      "--cmux-text-muted": ui.textMuted,
      "--cmux-text-faint": ui.textFaint,
      "--cmux-accent": ui.accent,
      "--cmux-accent-hover": ui.accentHover,
      "--cmux-accent-button": ui.accentButton,
      "--cmux-accent-button-hover": ui.accentButtonHover,
      "--cmux-active": ui.active,
    } as React.CSSProperties;
  }, [themeId]);
}

export { Terminal } from "./components/Terminal";
export { ThemePicker } from "./components/ThemePicker";
export { useTerminalThemeStore } from "./stores/terminal-theme.store";
export { useThemeVars } from "./hooks/useThemeVars";
export {
  terminalThemes,
  getTerminalTheme,
  deriveUiColors,
  DEFAULT_THEME_ID,
} from "./themes";
export type { TerminalTheme, UiColors } from "./themes";

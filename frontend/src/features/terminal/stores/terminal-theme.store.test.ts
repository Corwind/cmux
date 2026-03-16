import { describe, it, expect, beforeEach } from "vitest";
import { useTerminalThemeStore } from "./terminal-theme.store";
import { DEFAULT_THEME_ID } from "../themes";

describe("useTerminalThemeStore", () => {
  beforeEach(() => {
    useTerminalThemeStore.setState({ themeId: DEFAULT_THEME_ID });
  });

  it("has default theme id on init", () => {
    expect(useTerminalThemeStore.getState().themeId).toBe(DEFAULT_THEME_ID);
  });

  it("updates theme id via setThemeId", () => {
    useTerminalThemeStore.getState().setThemeId("dracula");
    expect(useTerminalThemeStore.getState().themeId).toBe("dracula");
  });

  it("can be set back to default", () => {
    useTerminalThemeStore.getState().setThemeId("nord");
    useTerminalThemeStore.getState().setThemeId(DEFAULT_THEME_ID);
    expect(useTerminalThemeStore.getState().themeId).toBe(DEFAULT_THEME_ID);
  });
});

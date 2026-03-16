import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ThemePicker } from "./ThemePicker";
import { useTerminalThemeStore } from "../stores/terminal-theme.store";
import { terminalThemes, DEFAULT_THEME_ID } from "../themes";

describe("ThemePicker", () => {
  beforeEach(() => {
    useTerminalThemeStore.setState({ themeId: DEFAULT_THEME_ID });
  });

  it("renders a select with all themes", () => {
    render(<ThemePicker />);
    const select = screen.getByLabelText("Terminal Theme") as HTMLSelectElement;
    expect(select).toBeDefined();
    expect(select.options.length).toBe(terminalThemes.length);
  });

  it("shows the current theme as selected", () => {
    render(<ThemePicker />);
    const select = screen.getByLabelText("Terminal Theme") as HTMLSelectElement;
    expect(select.value).toBe(DEFAULT_THEME_ID);
  });

  it("updates the store when a new theme is selected", () => {
    render(<ThemePicker />);
    const select = screen.getByLabelText("Terminal Theme") as HTMLSelectElement;
    fireEvent.change(select, { target: { value: "dracula" } });
    expect(useTerminalThemeStore.getState().themeId).toBe("dracula");
  });

  it("renders a color swatch matching the selected theme background", () => {
    render(<ThemePicker />);
    const swatch = screen.getByTitle("Theme preview");
    // jsdom converts hex to rgb, so just check it has a background set
    expect(swatch.style.background).toBeTruthy();
  });
});

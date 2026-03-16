import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ThemePicker } from "./ThemePicker";
import { useTerminalThemeStore } from "../stores/terminal-theme.store";
import { terminalThemes, DEFAULT_THEME_ID } from "../themes";

describe("ThemePicker", () => {
  beforeEach(() => {
    useTerminalThemeStore.setState({ themeId: DEFAULT_THEME_ID });
  });

  it("renders a button for each theme", () => {
    render(<ThemePicker />);
    for (const t of terminalThemes) {
      expect(screen.getByText(t.name)).toBeDefined();
    }
  });

  it("highlights the active theme", () => {
    render(<ThemePicker />);
    const activeButton = screen.getByText("Tokyo Night").closest("button")!;
    expect(activeButton.style.border).toContain("var(--cmux-accent)");
  });

  it("updates the store when a theme is clicked", () => {
    render(<ThemePicker />);
    const draculaButton = screen.getByText("Dracula").closest("button")!;
    fireEvent.click(draculaButton);
    expect(useTerminalThemeStore.getState().themeId).toBe("dracula");
  });

  it("renders color strips with multiple color segments", () => {
    const { container } = render(<ThemePicker />);
    // Each theme should have a color strip with 8 color segments
    const strips = container.querySelectorAll(".flex.h-3");
    expect(strips.length).toBe(terminalThemes.length);
    // Each strip should have 8 color cells (bg, fg, red, green, yellow, blue, magenta, cyan)
    for (const strip of strips) {
      expect(strip.children.length).toBe(8);
    }
  });
});

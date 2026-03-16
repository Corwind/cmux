import { describe, it, expect } from "vitest";
import {
  terminalThemes,
  getTerminalTheme,
  DEFAULT_THEME_ID,
} from "./themes";

describe("terminalThemes", () => {
  it("has at least one theme", () => {
    expect(terminalThemes.length).toBeGreaterThan(0);
  });

  it("each theme has a unique id", () => {
    const ids = terminalThemes.map((t) => t.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("each theme has required color fields", () => {
    const requiredKeys = [
      "background",
      "foreground",
      "cursor",
      "black",
      "red",
      "green",
      "yellow",
      "blue",
      "magenta",
      "cyan",
      "white",
    ];
    for (const t of terminalThemes) {
      for (const key of requiredKeys) {
        expect(t.theme).toHaveProperty(key);
      }
    }
  });

  it("default theme id exists in the list", () => {
    const found = terminalThemes.find((t) => t.id === DEFAULT_THEME_ID);
    expect(found).toBeDefined();
  });
});

describe("getTerminalTheme", () => {
  it("returns the requested theme by id", () => {
    const theme = getTerminalTheme("dracula");
    expect(theme.id).toBe("dracula");
    expect(theme.name).toBe("Dracula");
  });

  it("falls back to default for unknown id", () => {
    const theme = getTerminalTheme("nonexistent");
    expect(theme.id).toBe(DEFAULT_THEME_ID);
  });

  it("returns tokyo-night as default", () => {
    const theme = getTerminalTheme(DEFAULT_THEME_ID);
    expect(theme.theme.background).toBe("#1a1b26");
  });
});

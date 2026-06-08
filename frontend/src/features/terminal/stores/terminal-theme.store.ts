import { create } from "zustand";
import { persist } from "zustand/middleware";
import { DEFAULT_THEME_ID } from "../themes";

export const DEFAULT_FONT_FAMILY =
  // Nerd Fonts first — cover both basic Powerline AND extended glyphs (rounded corners, etc.)
  "'MesloLGS NF', 'Meslo LG S Nerd Font Mono', 'Meslo LG S Nerd Font', " +
  "'JetBrainsMono Nerd Font Mono', 'JetBrainsMono Nerd Font', " +
  "'FiraCode Nerd Font Mono', 'FiraCode Nerd Font', " +
  "'Hack Nerd Font Mono', 'Hack Nerd Font', " +
  // Classic "for Powerline" fonts — cover basic Powerline glyphs (U+E0A0–U+E0B3)
  "'Meslo LG S for Powerline', 'Meslo LG M for Powerline', 'Meslo LG L for Powerline', " +
  "'DejaVu Sans Mono for Powerline', 'Inconsolata for Powerline', " +
  "'Source Code Pro for Powerline', 'Fira Mono for Powerline', " +
  "'Roboto Mono for Powerline', 'Ubuntu Mono derivative Powerline', " +
  // Symbols-only Nerd Font as a last resort for any remaining glyphs
  "'Symbols Nerd Font Mono', 'Symbols Nerd Font', monospace";

interface TerminalThemeState {
  themeId: string;
  setThemeId: (id: string) => void;
  fontFamily: string;
  setFontFamily: (fontFamily: string) => void;
}

export const useTerminalThemeStore = create<TerminalThemeState>()(
  persist(
    (set) => ({
      themeId: DEFAULT_THEME_ID,
      setThemeId: (id) => set({ themeId: id }),
      fontFamily: DEFAULT_FONT_FAMILY,
      setFontFamily: (fontFamily) => set({ fontFamily }),
    }),
    { name: "cmux-terminal-theme" },
  ),
);

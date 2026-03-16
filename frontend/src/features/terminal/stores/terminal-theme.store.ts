import { create } from "zustand";
import { persist } from "zustand/middleware";
import { DEFAULT_THEME_ID } from "../themes";

interface TerminalThemeState {
  themeId: string;
  setThemeId: (id: string) => void;
}

export const useTerminalThemeStore = create<TerminalThemeState>()(
  persist(
    (set) => ({
      themeId: DEFAULT_THEME_ID,
      setThemeId: (id) => set({ themeId: id }),
    }),
    { name: "cmux-terminal-theme" },
  ),
);

import type { StateCreator } from "zustand";

export interface UiSlice {
  theme: "light" | "dark";
  sidebarOpen: boolean;
  toggleTheme: () => void;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
}

export const createUiSlice: StateCreator<UiSlice> = (set) => ({
  theme: "light",
  sidebarOpen: false,
  toggleTheme: () =>
    set((state) => ({ theme: state.theme === "light" ? "dark" : "light" })),
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
});

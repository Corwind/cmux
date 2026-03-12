import { create } from "zustand";
import { devtools } from "zustand/middleware";
import { createUiSlice, type UiSlice } from "./slices/ui.slice";
import { createAuthSlice, type AuthSlice } from "./slices/auth.slice";

export type AppState = UiSlice & AuthSlice;

export const useAppStore = create<AppState>()(
  devtools((...args) => ({
    ...createUiSlice(...args),
    ...createAuthSlice(...args),
  })),
);

import { create } from "zustand";

interface SessionTab {
  id: string;
  name: string;
}

interface SessionsStore {
  activeSessionId: string | null;
  openTabs: SessionTab[];
  setActiveSession: (id: string | null) => void;
  openTab: (id: string, name: string) => void;
  closeTab: (id: string) => void;
}

export const useSessionsStore = create<SessionsStore>()((set, get) => ({
  activeSessionId: null,
  openTabs: [],
  setActiveSession: (id) => set({ activeSessionId: id }),
  openTab: (id, name) => {
    const { openTabs } = get();
    if (!openTabs.some((tab) => tab.id === id)) {
      set({ openTabs: [...openTabs, { id, name }] });
    }
    set({ activeSessionId: id });
  },
  closeTab: (id) => {
    const { openTabs, activeSessionId } = get();
    const filtered = openTabs.filter((tab) => tab.id !== id);
    const newActive =
      activeSessionId === id
        ? (filtered[filtered.length - 1]?.id ?? null)
        : activeSessionId;
    set({ openTabs: filtered, activeSessionId: newActive });
  },
}));

import { create } from "zustand";

interface SessionsStore {
  activeSessionId: string | null;
  setActiveSession: (id: string | null) => void;
}

export const useSessionsStore = create<SessionsStore>()((set) => ({
  activeSessionId: null,
  setActiveSession: (id) => set({ activeSessionId: id }),
}));

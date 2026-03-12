import { create } from "zustand";

interface ExampleStore {
  selectedExampleId: string | null;
  setSelectedExample: (id: string | null) => void;
}

export const useExampleStore = create<ExampleStore>()((set) => ({
  selectedExampleId: null,
  setSelectedExample: (id) => set({ selectedExampleId: id }),
}));

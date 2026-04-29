import { create } from "zustand";

interface NavigationState {
  selectedTenantId: string | null;
  setSelectedTenantId: (id: string | null) => void;
}

export const useNavigationStore = create<NavigationState>((set) => ({
  selectedTenantId: null,
  setSelectedTenantId: (id) => set({ selectedTenantId: id }),
}));

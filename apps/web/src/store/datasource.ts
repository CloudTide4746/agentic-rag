import { create } from "zustand";
import * as adminApi from "../lib/api/admin";
import type { DatasourceItem } from "../types/legacy";

interface DatasourceState {
  items: DatasourceItem[];
  loading: boolean;
  saving: boolean;
  savedAt: string | null;
  load: () => Promise<void>;
  save: () => Promise<void>;
  updateItem: (sourceType: DatasourceItem["source_type"], update: Partial<DatasourceItem>) => void;
}

export const useDatasourceStore = create<DatasourceState>()((set, get) => ({
  items: [],
  loading: false,
  saving: false,
  savedAt: null,

  async load() {
    set({ loading: true });
    try {
      const items = await adminApi.getDatasources();
      set({ items });
    } finally {
      set({ loading: false });
    }
  },

  async save() {
    set({ saving: true });
    try {
      await adminApi.updateDatasources(get().items);
      set({ savedAt: new Date().toLocaleTimeString("zh-CN", { hour12: false }) });
    } finally {
      set({ saving: false });
    }
  },

  updateItem(sourceType, update) {
    set((state) => ({
      items: state.items.map((item) =>
        item.source_type === sourceType ? { ...item, ...update } : item,
      ),
    }));
  },
}));

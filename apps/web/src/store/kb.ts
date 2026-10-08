import { create } from "zustand";
import * as adminApi from "../lib/api/admin";
import type { KbDocument, KnowledgeBase } from "../types/legacy";

const PROCESSING_STATUSES = new Set(["parsing", "chunking", "embedding"]);
const POLL_INTERVAL_MS = 2500;

interface KbState {
  selectedKbId: number | null;
  kbs: KnowledgeBase[];
  documents: KbDocument[];
  loading: boolean;
  creating: boolean;
  documentsRefreshing: boolean;
  loadKbs: () => Promise<void>;
  loadDocuments: () => Promise<void>;
  refresh: () => Promise<void>;
  selectKb: (id: number | null) => Promise<void>;
  init: () => Promise<void>;
  createKb: (name: string, description: string) => Promise<void>;
  upload: (files: File[]) => Promise<{ ok: number; errors: string[] }>;
  removeDoc: (docId: number) => Promise<void>;
  retryDoc: (docId: number) => Promise<void>;
  stopPolling: () => void;
}

export const selectSelectedKb = (state: KbState): KnowledgeBase | null =>
  state.kbs.find((kb) => kb.id === state.selectedKbId) ?? null;
export const selectProcessingCount = (state: KbState): number =>
  state.documents.filter((doc) => PROCESSING_STATUSES.has(doc.status)).length;
export const selectFailedCount = (state: KbState): number =>
  state.documents.filter((doc) => doc.status === "failed").length;

let pollTimer: ReturnType<typeof setInterval> | null = null;
let polling = false;

export const useKbStore = create<KbState>()((set, get) => {
  async function loadDocuments(): Promise<void> {
    const kbId = get().selectedKbId;
    if (kbId === null) {
      set({ documents: [] });
      return;
    }

    set({ documentsRefreshing: true });
    try {
      const documents = await adminApi.listDocuments(kbId);
      if (get().selectedKbId === kbId) set({ documents });
    } finally {
      if (get().selectedKbId === kbId) set({ documentsRefreshing: false });
    }
  }

  async function refresh(): Promise<void> {
    await Promise.all([get().loadKbs(), loadDocuments()]);
  }

  function startPolling(): void {
    if (pollTimer !== null) return;
    pollTimer = setInterval(() => {
      if (polling || selectProcessingCount(get()) === 0) return;
      polling = true;
      void refresh()
        .catch(() => undefined)
        .finally(() => {
          polling = false;
        });
    }, POLL_INTERVAL_MS);
  }

  return {
    selectedKbId: null,
    kbs: [],
    documents: [],
    loading: false,
    creating: false,
    documentsRefreshing: false,
    async loadKbs() {
      const kbs = await adminApi.listKbs();
      set({ kbs });
    },

    async loadDocuments() {
      await loadDocuments();
    },

    async refresh() {
      await refresh();
    },

    async selectKb(id) {
      set({ selectedKbId: id, documents: [] });
      await loadDocuments();
    },

    async init() {
      set({ loading: true });
      try {
        await get().loadKbs();
        const { selectedKbId, kbs } = get();
        if (selectedKbId === null && kbs.length > 0) {
          set({ selectedKbId: kbs[0].id });
        }
        await loadDocuments();
      } finally {
        set({ loading: false });
      }
      startPolling();
    },

    async createKb(name, description) {
      set({ creating: true });
      try {
        const kb = await adminApi.createKb(name, description);
        await get().loadKbs();
        set({ selectedKbId: kb.id });
        await loadDocuments();
      } finally {
        set({ creating: false });
      }
    },

    async upload(files) {
      const kbId = get().selectedKbId;
      if (kbId === null) return { ok: 0, errors: ["未选择知识库"] };

      let ok = 0;
      const errors: string[] = [];
      for (const file of files) {
        try {
          await adminApi.uploadDocument(kbId, file);
          ok += 1;
        } catch (error) {
          errors.push(
            `${file.name}：${error instanceof Error ? error.message : "未知错误"}`,
          );
        }
      }
      await refresh();
      return { ok, errors };
    },

    async removeDoc(docId) {
      const kbId = get().selectedKbId;
      if (kbId === null) return;
      await adminApi.deleteDocument(kbId, docId);
      await refresh();
    },

    async retryDoc(docId) {
      const kbId = get().selectedKbId;
      if (kbId === null) return;
      await adminApi.retryDocument(kbId, docId);
      await refresh();
    },

    stopPolling() {
      if (pollTimer === null) return;
      clearInterval(pollTimer);
      pollTimer = null;
    },
  };
});

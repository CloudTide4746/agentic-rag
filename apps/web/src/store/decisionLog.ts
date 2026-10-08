import { create } from "zustand";
import * as adminApi from "../lib/api/admin";
import type { DecisionLog, DecisionLogQuery } from "../types/legacy";

const PAGE_SIZE = 8;

interface DecisionLogState {
  logs: DecisionLog[];
  total: number;
  page: number;
  readonly pageSize: number;
  loading: boolean;
  filterSessionId: string;
  filterStart: string;
  filterEnd: string;
  setFilterSessionId: (value: string) => void;
  setFilterStart: (value: string) => void;
  setFilterEnd: (value: string) => void;
  query: (page?: number) => Promise<void>;
  getDetail: (logId: number) => Promise<DecisionLog>;
}

export const useDecisionLogStore = create<DecisionLogState>()((set, get) => ({
  logs: [],
  total: 0,
  page: 1,
  pageSize: PAGE_SIZE,
  loading: false,
  filterSessionId: "",
  filterStart: "",
  filterEnd: "",

  setFilterSessionId(value) {
    set({ filterSessionId: value });
  },
  setFilterStart(value) {
    set({ filterStart: value });
  },
  setFilterEnd(value) {
    set({ filterEnd: value });
  },

  async query(page = 1) {
    const { filterSessionId, filterStart, filterEnd } = get();
    const trimmedSessionId = filterSessionId.trim();
    const parsedSessionId = trimmedSessionId ? Number(trimmedSessionId) : undefined;
    if (parsedSessionId !== undefined && !Number.isSafeInteger(parsedSessionId)) {
      throw new Error("会话 ID 必须是有效整数");
    }

    set({ loading: true, page });
    try {
      const query: DecisionLogQuery = {
        session_id: parsedSessionId,
        start: filterStart || undefined,
        end: filterEnd || undefined,
        page,
        page_size: PAGE_SIZE,
      };
      const result = await adminApi.queryDecisionLogs(query);
      set({ logs: result.items, total: result.total });
    } finally {
      set({ loading: false });
    }
  },

  async getDetail(logId) {
    return adminApi.getDecisionLog(logId);
  },
}));

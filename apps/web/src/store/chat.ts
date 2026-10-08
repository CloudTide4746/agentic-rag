/** @format */

import { create } from "zustand";
import * as chatApi from "../lib/api/chat";
import { getDecisionLog } from "../lib/api/admin";
import type {
  ChatMessage,
  ChatSession,
  Citation,
  DecisionStep,
  SseEvent,
  SseEventType,
} from "../types/legacy";

export interface DecisionCard {
  uid: number;
  /** 决策点序号（0 = 汇总/异常） */
  step: number;
  type: SseEventType;
  status: "pending" | "done" | "warn";
  detail: string;
  payload: Record<string, unknown>;
  ts: number;
  /** 回流轮次（0 起，>0 时侧栏缩进一层） */
  round: number;
  /** 本环节耗时 ms（pending 未结算为 0；等待型过渡卡为 0） */
  elapsedMs: number;
}

/** 历史决策链回看态（agent_decision_log → 卡片） */
export interface ViewingLog {
  id: number;
  question: string;
  cards: DecisionCard[];
  elapsedMs: number;
  totalRounds: number;
}

type DoneStats = {
  elapsed_ms: number;
  prompt: number;
  completion: number;
  rounds: number;
};

type ChatState = {
  sessions: ChatSession[];
  currentSessionId: number | null;
  currentSession: ChatSession | null;
  sessionsLoading: boolean;
  messages: ChatMessage[];
  messagesLoading: boolean;
  cards: DecisionCard[];
  pendingCard: DecisionCard | null;
  streaming: boolean;
  streamingText: string;
  streamError: string | null;
  doneStats: DoneStats | null;
  activeRound: number;
  /** 当前进行中的管线环节（1~7，0=空闲；供流程图模式点亮） */
  activeStage: number;
  /** 历史决策链回看（非 null 时侧栏展示该日志而非实时流） */
  viewingLog: ViewingLog | null;
  viewingLoading: boolean;
  loadSessions: () => Promise<void>;
  selectSession: (id: number) => Promise<void>;
  newSession: () => Promise<void>;
  removeSession: (id: number) => Promise<void>;
  send: (question: string, images?: string[]) => Promise<void>;
  regenerate: () => Promise<void>;
  stopStream: () => void;
  resetDecision: () => void;
  viewDecisionLog: (logId: number) => Promise<void>;
  exitViewing: () => void;
};

let uid = 1;

/** 事件完成后预测下一步的 pending 文案 */
const NEXT_HINT: Partial<Record<SseEventType, string>> = {
  router: "正在改写查询…",
  rewrite: "正在多路并行检索…",
  retrieve: "正在融合与评估检索结果…",
  grade: "正在准备生成…",
  correct: "正在执行纠正策略…",
  reflect: "正在整理引用出处…",
  citation: "正在收尾统计…",
};

export const useChatStore = create<ChatState>()((set, get) => {
  let abortController: AbortController | null = null;
  /* 单步计时：上一事件 ts（事件卡耗时 = 本事件 ts - lastTs） */
  let lastTs = 0;
  /* 生成卡（流式答案）首 token 时刻与卡片 uid */
  let generateStartTs: number | null = null;
  let generateCardUid: number | null = null;

  function syncCurrentSession(
    sessions: ChatSession[],
    currentSessionId: number | null,
  ): ChatSession | null {
    return sessions.find((session) => session.id === currentSessionId) ?? null;
  }

  function settlePending(status: "done" | "warn" = "done"): void {
    const pendingCard = get().pendingCard;
    if (!pendingCard) return;
    set((state) => ({
      cards: [...state.cards, { ...pendingCard, status }],
      pendingCard: null,
    }));
  }

  /** 结算生成卡耗时（首个 generate 事件 → 结算事件），需在 settlePending 前调用 */
  function settleGenerate(settleTs: number): void {
    const pendingCard = get().pendingCard;
    if (
      pendingCard &&
      generateCardUid !== null &&
      pendingCard.uid === generateCardUid
    ) {
      set({
        pendingCard: {
          ...pendingCard,
          elapsedMs: Math.max(0, settleTs - (generateStartTs ?? settleTs)),
          // 与日志 steps 的 payload.chars 同构
          payload: { chars: get().streamingText.length },
        },
      });
      generateCardUid = null;
      generateStartTs = null;
    }
  }

  function pushPending(detail: string): void {
    set((state) => ({
      pendingCard: {
        uid: uid++,
        step: 0,
        type: "generate",
        status: "pending",
        detail,
        payload: {},
        ts: Date.now(),
        round: state.activeRound,
        elapsedMs: 0,
      },
    }));
  }

  function warnOf(event: SseEvent): boolean {
    if (event.type === "grade") return event.verdict !== "pass";
    if (event.type === "correct") return true;
    if (event.type === "reflect") return event.result === "fail";
    if (event.type === "error") return true;
    return false;
  }

  /** 事件完成后推进管线环节（供流程图模式点亮下一节点） */
  function nextStage(event: SseEvent): number {
    switch (event.type) {
      case "router":
        return event.intent === "chitchat"
          ? 6
          : event.intent === "data_query"
            ? 3
            : 2;
      case "rewrite":
        return 3;
      case "grade":
        return event.verdict === "pass"
          ? 6
          : event.verdict === "partial"
            ? 3
            : 5;
      case "correct":
        return event.action === "give_up" ? 6 : 3;
      case "reflect":
        return event.result === "pass" ? 7 : 3;
      case "citation":
        return 0;
      default:
        return get().activeStage;
    }
  }

  function handleEvent(event: SseEvent): void {
    switch (event.type) {
      case "router":
      case "rewrite":
      case "retrieve":
      case "grade":
      case "correct":
      case "reflect":
      case "citation": {
        if (get().pendingCard && event.type === "reflect") {
          settleGenerate(event.ts);
          settlePending("done");
        } else {
          settlePending();
        }
        const { type, step, detail, ts, ...payload } = event;
        if (type === "correct")
          set((state) => ({
            activeRound: Math.max(state.activeRound, event.round),
          }));
        if (type === "reflect" && event.result === "fail")
          set((state) => ({ activeRound: state.activeRound + 1 }));
        const activeRound = get().activeRound;
        const round =
          type === "correct" || type === "reflect"
            ? Math.max(0, activeRound - (type === "correct" ? 1 : 0))
            : activeRound;
        set((state) => ({
          cards: [
            ...state.cards,
            {
              uid: uid++,
              step,
              type,
              status: warnOf(event) ? "warn" : "done",
              detail,
              payload,
              ts,
              round,
              elapsedMs: Math.max(0, ts - lastTs),
            },
          ],
        }));
        lastTs = ts;
        const hint = NEXT_HINT[type];
        if (hint) pushPending(hint);
        set({ activeStage: nextStage(event) });
        break;
      }
      case "generate": {
        if (generateStartTs === null) generateStartTs = event.ts;
        set((state) => ({ streamingText: state.streamingText + event.delta }));
        const pendingCard = get().pendingCard;
        if (!pendingCard) {
          pushPending("正在流式生成答案…");
        } else if (pendingCard.status === "pending") {
          set({
            pendingCard: {
              ...pendingCard,
              detail: `正在流式生成答案（${get().streamingText.length} 字）…`,
            },
          });
        }
        if (generateCardUid === null && get().pendingCard)
          generateCardUid = get().pendingCard?.uid ?? null;
        set({ activeStage: 6 });
        break;
      }
      case "done": {
        settleGenerate(event.ts);
        settlePending();
        set((state) => ({
          doneStats: {
            elapsed_ms: event.elapsed_ms,
            prompt: event.token_usage.prompt,
            completion: event.token_usage.completion,
            rounds: state.cards.filter(
              (card) =>
                card.type === "correct" ||
                (card.type === "reflect" &&
                  (card.payload as { result?: string }).result === "fail"),
            ).length,
          },
          activeStage: 0,
        }));
        break;
      }
      case "error": {
        settleGenerate(event.ts);
        settlePending("warn");
        set({ streamError: `${event.code}：${event.message}`, activeStage: 0 });
        break;
      }
    }
  }

  /** DecisionStep[]（日志落库）→ 决策卡（含回流轮次推算，与实时流同构） */
  function stepsToCards(steps: DecisionStep[]): DecisionCard[] {
    const out: DecisionCard[] = [];
    let round = 0;
    for (const step of steps) {
      if (step.type === "correct")
        round = Math.max(round, (step.payload.round as number) ?? round);
      if (step.type === "reflect" && step.payload.result === "fail") round += 1;
      const warn =
        (step.type === "grade" && step.payload.verdict !== "pass") ||
        step.type === "correct" ||
        (step.type === "reflect" && step.payload.result === "fail") ||
        step.type === "error";
      out.push({
        uid: uid++,
        step: step.step,
        type: step.type,
        status: warn ? "warn" : "done",
        detail: step.detail,
        payload: step.payload,
        ts: 0,
        round:
          step.type === "correct" || step.type === "reflect"
            ? Math.max(0, round)
            : round,
        elapsedMs: step.elapsed_ms,
      });
    }
    return out;
  }

  async function runStream(
    question: string,
    skipUserPersist: boolean,
    images?: string[],
  ): Promise<void> {
    const sessionId = get().currentSessionId;
    if (sessionId === null) return;
    set({ streaming: true });
    let finalText = "";
    let finalCitations: Citation[] | undefined;
    const onEvent = (event: SseEvent): void => {
      handleEvent(event);
      if (event.type === "generate") finalText = get().streamingText;
      if (event.type === "citation")
        finalCitations = event.citations.length ? event.citations : undefined;
    };

    const controller = new AbortController();
    abortController = controller;
    let aborted = false;
    try {
      await chatApi.ask(
        sessionId,
        question,
        onEvent,
        controller.signal,
        skipUserPersist,
        images,
      );
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError")
        aborted = true;
      else throw error;
    } finally {
      if (abortController === controller) abortController = null;
      set({ streaming: false });
      if (finalText || get().streamError) {
        set((state) => ({
          messages: [
            ...state.messages,
            {
              id: -Date.now() - 1,
              session_id: sessionId,
              role: "assistant",
              content:
                finalText ||
                `**链路异常中断**\n\n${get().streamError ?? "未知错误"}`,
              citations: finalCitations,
              created_at: new Date().toISOString(),
            },
          ],
        }));
      }
      await get().loadSessions();
      const currentSessionId = get().currentSessionId;
      if (!aborted && currentSessionId !== null) {
        try {
          const messages = await chatApi.getMessages(currentSessionId);
          if (get().currentSessionId === currentSessionId) set({ messages });
        } catch {
          // 保留本地状态
        }
      }
    }
  }

  return {
    sessions: [],
    currentSessionId: null,
    currentSession: null,
    sessionsLoading: false,
    messages: [],
    messagesLoading: false,
    cards: [],
    pendingCard: null,
    streaming: false,
    streamingText: "",
    streamError: null,
    doneStats: null,
    activeRound: 0,
    activeStage: 0,
    viewingLog: null,
    viewingLoading: false,

    async loadSessions() {
      set({ sessionsLoading: true });
      try {
        const sessions = await chatApi.listSessions();
        set((state) => ({
          sessions,
          currentSession: syncCurrentSession(sessions, state.currentSessionId),
        }));
      } finally {
        set({ sessionsLoading: false });
      }
    },

    async selectSession(id) {
      if (get().streaming) get().stopStream();
      set({
        currentSessionId: id,
        currentSession: syncCurrentSession(get().sessions, id),
        messagesLoading: true,
        viewingLog: null,
      });
      get().resetDecision();
      try {
        const messages = await chatApi.getMessages(id);
        if (get().currentSessionId === id) set({ messages });
      } finally {
        if (get().currentSessionId === id) set({ messagesLoading: false });
      }
    },

    async newSession() {
      const session = await chatApi.createSession();
      set((state) => ({ sessions: [session, ...state.sessions] }));
      await get().selectSession(session.id);
    },

    async removeSession(id) {
      await chatApi.deleteSession(id);
      const sessions = get().sessions.filter((session) => session.id !== id);
      const isCurrent = get().currentSessionId === id;
      set({
        sessions,
        ...(isCurrent
          ? { currentSessionId: null, currentSession: null, messages: [] }
          : {}),
      });
      if (isCurrent) {
        get().resetDecision();
        if (sessions.length > 0) await get().selectSession(sessions[0].id);
      }
    },

    async send(question, images) {
      if (get().streaming) return;
      if (get().currentSessionId === null) {
        try {
          await get().newSession();
        } catch (error) {
          set({
            streamError: `会话创建失败：${error instanceof Error ? error.message : "未知错误"}`,
          });
          return;
        }
      }
      const currentSessionId = get().currentSessionId;
      if (currentSessionId === null) return;
      get().exitViewing();
      get().resetDecision();
      set((state) => ({
        streaming: true,
        messages: [
          ...state.messages,
          {
            id: -Date.now(),
            session_id: currentSessionId,
            role: "user",
            content: question,
            images: images?.length ? images : undefined,
            created_at: new Date().toISOString(),
          },
        ],
      }));
      await runStream(question, false, images);
    },

    async regenerate() {
      const { currentSessionId, streaming, messages } = get();
      if (currentSessionId === null || streaming) return;
      const lastAssistant = [...messages]
        .reverse()
        .find((message) => message.role === "assistant");
      const lastUser = [...messages]
        .reverse()
        .find((message) => message.role === "user");
      if (!lastAssistant || !lastUser) return;
      const idx = messages
        .map((message) => message.id)
        .lastIndexOf(lastAssistant.id);
      if (idx !== -1)
        set((state) => ({
          messages: state.messages.filter((_, index) => index !== idx),
        }));
      await chatApi.purgeLastAssistant(currentSessionId);
      get().exitViewing();
      get().resetDecision();
      await runStream(lastUser.content, true, lastUser.images);
    },

    stopStream() {
      abortController?.abort();
    },

    resetDecision() {
      set({
        cards: [],
        pendingCard: null,
        streamingText: "",
        streamError: null,
        doneStats: null,
        activeRound: 0,
        activeStage: 0,
      });
      lastTs = Date.now();
      generateStartTs = null;
      generateCardUid = null;
    },

    async viewDecisionLog(logId) {
      if (get().streaming || get().viewingLoading) return;
      set({ viewingLoading: true });
      try {
        const log = await getDecisionLog(logId);
        set({
          viewingLog: {
            id: log.id,
            question: log.question,
            cards: stepsToCards(log.steps),
            elapsedMs: log.elapsed_ms,
            totalRounds: log.total_rounds,
          },
        });
      } catch (error) {
        console.warn("[chat] 决策日志加载失败", error);
      } finally {
        set({ viewingLoading: false });
      }
    },

    exitViewing() {
      set({ viewingLog: null });
    },
  };
});

import { create } from 'zustand'
import * as chatApi from '../lib/api/chat'
import type { Citation, SseEvent, SseEventType } from '../types/legacy'

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

type DoneStats = { elapsed_ms: number; prompt: number; completion: number; rounds: number }
type ChatState = {
  streamSessionId: string | null;
  question: string;
  cards: DecisionCard[];
  pendingCard: DecisionCard | null;
  streaming: boolean;
  streamingText: string;
  streamCitations: Citation[];
  streamError: string | null;
  doneStats: DoneStats | null;
  activeRound: number;
  activeStage: number;
  send: (sessionId: string, question: string) => Promise<void>;
  stopStream: () => void;
  resetDecision: () => void;
  dismissDraft: () => void;
}
let uid = 1
const NEXT_HINT: Partial<Record<SseEventType, string>> = {
  router: '正在改写查询…', rewrite: '正在多路并行检索…',
  retrieve: '正在融合与评估检索结果…', grade: '正在准备生成…',
  correct: '正在执行纠正策略…', reflect: '正在整理引用出处…', citation: '正在收尾统计…',
}
const initialState = {
  streamSessionId: null, question: '', cards: [], pendingCard: null,
  streaming: false, streamingText: '', streamCitations: [], streamError: null,
  doneStats: null, activeRound: 0, activeStage: 0,
}

// Only the active stream lives here. Sessions and history belong to React Query.
export const useChatStore = create<ChatState>()((set, get) => {
  let abortController: AbortController | null = null
  let lastTs = 0
  let generateStartTs: number | null = null
  let generateCardUid: number | null = null
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
        if (event.type === 'citation') set({ streamCitations: event.citations });
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
                  card.payload.result === "fail"),
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

  return {
    ...initialState,
    async send(sessionId, question) {
      if (get().streaming) throw new Error('请等待当前回答结束')
      get().resetDecision()
      const controller = new AbortController()
      abortController = controller
      set({ streamSessionId: sessionId, question, streaming: true })
      try {
        await chatApi.ask(sessionId, question, event => {
          if (abortController === controller && !controller.signal.aborted) handleEvent(event)
        }, controller.signal)
      } catch (error) {
        if (abortController !== controller) return
        settleGenerate(Date.now())
        settlePending('warn')
        set({ streamError: controller.signal.aborted
          ? '已停止接收，服务端可能仍在处理；请刷新历史确认结果。'
          : error instanceof Error ? error.message : '问答失败' })
        if (!controller.signal.aborted) throw error
      } finally {
        if (abortController === controller) {
          abortController = null
          set({ streaming: false, activeStage: 0 })
        }
      }
    },
    stopStream() { abortController?.abort() },
    dismissDraft() { set({ question: '', streamingText: '', streamCitations: [] }) },
    resetDecision() {
      abortController?.abort()
      abortController = null
      lastTs = Date.now()
      generateStartTs = null
      generateCardUid = null
      set(initialState)
    },
  }
})

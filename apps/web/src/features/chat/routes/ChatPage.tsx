/** @format */

import { useEffect, useState, type FormEvent } from "react";
import { useChatStore } from "../../../store/chat";

export default function ChatPage() {
  const sessions = useChatStore((state) => state.sessions);
  const currentSession = useChatStore((state) => state.currentSession);
  const messages = useChatStore((state) => state.messages);
  const sessionsLoading = useChatStore((state) => state.sessionsLoading);
  const messagesLoading = useChatStore((state) => state.messagesLoading);
  const streaming = useChatStore((state) => state.streaming);
  const streamingText = useChatStore((state) => state.streamingText);
  const cards = useChatStore((state) => state.cards);
  const pendingCard = useChatStore((state) => state.pendingCard);
  const loadSessions = useChatStore((state) => state.loadSessions);
  const selectSession = useChatStore((state) => state.selectSession);
  const newSession = useChatStore((state) => state.newSession);
  const send = useChatStore((state) => state.send);
  const [question, setQuestion] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  async function initializeSessions() {
    try {
      setLoadError(null);
      await loadSessions();
      const {
        sessions: loadedSessions,
        currentSessionId,
        selectSession: select,
      } = useChatStore.getState();
      const selectedId = loadedSessions.some(
        (session) => session.id === currentSessionId,
      )
        ? currentSessionId
        : loadedSessions[0]?.id;
      if (selectedId !== undefined && selectedId !== null)
        await select(selectedId);
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "会话加载失败");
    }
  }

  useEffect(() => {
    void initializeSessions();
  }, [loadSessions]);

  async function handleNewSession() {
    setActionError(null);
    try {
      await newSession();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "新建会话失败");
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const text = question.trim();
    if (!text || streaming) return;
    setQuestion("");
    setActionError(null);
    try {
      await send(text);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "发送失败");
    }
  }

  async function handleSelectSession(id: number) {
    setActionError(null);
    try {
      await selectSession(id);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "消息加载失败");
    }
  }

  return (
    <div className='grid min-h-[calc(100vh-7.5rem)] grid-cols-1 md:grid-cols-[220px_minmax(0,1fr)] xl:grid-cols-[220px_minmax(0,1fr)_280px]'>
      <aside
        aria-label='会话列表'
        aria-busy={sessionsLoading}
        className={`border-b border-slate-800 bg-[#0b111d] p-3 md:border-b-0 md:border-r ${sessionsLoading ? "animate-pulse" : ""}`}
      >
        <div className='mb-3 flex items-center justify-between px-1'>
          <h1 className='text-xs font-semibold tracking-wide text-slate-200'>
            最近会话
          </h1>
          <button
            type='button'
            onClick={() => void handleNewSession()}
            disabled={sessionsLoading}
            className='rounded border border-cyan-900 px-2 py-1 text-[10px] text-cyan-200 disabled:cursor-not-allowed disabled:opacity-50'
          >
            新建
          </button>
        </div>

        {loadError && (
          <div
            role='alert'
            className='mb-3 rounded border border-rose-900/70 p-2 text-xs text-rose-200'
          >
            <p>会话加载失败：{loadError}</p>
            <button
              type='button'
              onClick={() => void initializeSessions()}
              disabled={sessionsLoading}
              className='mt-1 underline disabled:opacity-50'
            >
              重试
            </button>
          </div>
        )}

        <ul className='grid gap-1 sm:grid-cols-3 md:grid-cols-1'>
          {sessions.map((session) => {
            const selected = session.id === currentSession?.id;
            return (
              <li key={session.id}>
                <button
                  type='button'
                  onClick={() => void handleSelectSession(session.id)}
                  disabled={sessionsLoading}
                  aria-current={selected ? "page" : undefined}
                  className={`w-full rounded-lg border px-3 py-2.5 text-left disabled:cursor-not-allowed disabled:opacity-60 ${selected ? "border-cyan-800/80 bg-cyan-950/40" : "border-transparent bg-slate-900/50 hover:border-slate-700"}`}
                >
                  <span className='block truncate text-xs font-medium text-slate-200'>
                    {session.title || "新会话"}
                  </span>
                  <span className='mt-1 block font-mono text-[10px] text-slate-500'>
                    {new Date(session.updated_at).toLocaleString()}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
        {!sessionsLoading && !loadError && sessions.length === 0 && (
          <p className='mt-4 rounded-lg border border-dashed border-slate-800 p-3 text-[11px] leading-relaxed text-slate-500'>
            还没有会话，点击“新建”开始提问。
          </p>
        )}
      </aside>

      <section
        aria-labelledby='chat-title'
        className='flex min-h-[620px] min-w-0 flex-col'
      >
        <header className='flex min-h-12 items-center gap-3 border-b border-slate-800 bg-slate-900/40 px-4 sm:px-5'>
          <div className='min-w-0'>
            <h2
              id='chat-title'
              className='truncate text-sm font-medium text-slate-100'
            >
              {currentSession?.title ?? "选择或新建一个会话"}
            </h2>
            <p className='font-mono text-[10px] text-slate-500'>
              {streaming ? "正在生成回答…" : `${messages.length} 条消息`}
            </p>
          </div>
        </header>

        <div
          aria-live='polite'
          className='flex-1 space-y-5 overflow-y-auto p-4 sm:p-6'
        >
          {messagesLoading && (
            <p className='text-sm text-slate-500'>正在加载消息…</p>
          )}
          {!messagesLoading &&
            currentSession &&
            messages.length === 0 &&
            !streaming && (
              <p className='mx-auto mt-12 max-w-md text-center text-sm text-slate-500'>
                发送第一个问题，开始这个会话。
              </p>
            )}
          {!messagesLoading &&
            messages.map((message) => (
              <article
                key={message.id}
                className={`max-w-3xl rounded-xl border p-4 ${message.role === "user" ? "ml-auto border-cyan-900/70 bg-cyan-950/20" : "border-slate-800 bg-[#0d1522]"}`}
              >
                <p className='mb-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500'>
                  {message.role === "user" ? "你" : "Agent"} ·{" "}
                  {new Date(message.created_at).toLocaleTimeString()}
                </p>
                <p className='whitespace-pre-wrap text-sm leading-6 text-slate-200'>
                  {message.content}
                </p>
                {message.citations?.map((citation, index) => (
                  <p
                    key={`${citation.doc_name}-${citation.chunk_no}-${index}`}
                    className='mt-2 text-xs text-cyan-200'
                  >
                    [{index + 1}] {citation.doc_name}
                    {citation.section_path ? ` · ${citation.section_path}` : ""}
                  </p>
                ))}
              </article>
            ))}
          {streaming &&
            !messages.some(
              (message) => message.id < 0 && message.role === "assistant",
            ) && (
              <article className='max-w-3xl rounded-xl border border-cyan-900/70 bg-cyan-950/20 p-4'>
                <p className='mb-2 text-[10px] font-semibold uppercase tracking-wider text-cyan-200'>
                  Agent · 正在生成
                </p>
                <p className='whitespace-pre-wrap text-sm leading-6 text-slate-200'>
                  {streamingText}
                </p>
              </article>
            )}
          {actionError && (
            <p role='alert' className='text-sm text-rose-300'>
              {actionError}
            </p>
          )}
        </div>

        <form
          aria-label='问答输入'
          className='border-t border-slate-800 bg-[#0b111d] p-3 sm:p-4'
          onSubmit={(event) => void handleSubmit(event)}
        >
          <label htmlFor='chat-question' className='sr-only'>
            输入问题
          </label>
          <textarea
            id='chat-question'
            value={question}
            onChange={(event) => setQuestion(event.target.value)}
            disabled={streaming}
            placeholder='输入问题…'
            className='min-h-20 w-full resize-y rounded-xl border border-slate-700 bg-[#080d16] p-3 text-sm text-slate-300 placeholder:text-slate-600 disabled:cursor-not-allowed disabled:opacity-60'
          />
          <div className='mt-2 flex items-center justify-between'>
            <span className='text-[10px] text-slate-500'>
              {streaming ? "回答生成中" : "Enter 发送 · Shift+Enter 换行"}
            </span>
            <button
              type='submit'
              disabled={!question.trim() || streaming}
              className='rounded-lg border border-cyan-900 bg-cyan-950/40 px-4 py-2 text-xs text-cyan-200 disabled:cursor-not-allowed disabled:opacity-50'
            >
              发送
            </button>
          </div>
        </form>
      </section>

      <aside
        aria-label='决策链'
        className='border-t border-slate-800 bg-[#0b111d] p-4 xl:border-l xl:border-t-0'
      >
        <div className='flex items-center justify-between'>
          <h2 className='text-xs font-semibold text-slate-200'>决策链</h2>
          <span className='font-mono text-[9px] text-slate-500'>
            LIVE TRACE
          </span>
        </div>
        <ol className='mt-4 space-y-3'>
          {[...cards, ...(pendingCard ? [pendingCard] : [])].map((card) => (
            <li key={card.uid} className='flex gap-3'>
              <span className='grid size-6 shrink-0 place-items-center rounded-full border border-cyan-800 bg-cyan-950 text-[10px] text-cyan-200'>
                {card.step || "·"}
              </span>
              <div className='min-w-0 flex-1 border-b border-slate-800 pb-3'>
                <p className='text-xs font-medium text-slate-200'>
                  {card.detail}
                </p>
                <p className='mt-1 text-[10px] leading-4 text-slate-500'>
                  {card.status === "pending"
                    ? "进行中"
                    : card.status === "warn"
                      ? "需关注"
                      : "已完成"}
                </p>
              </div>
            </li>
          ))}
        </ol>
        {cards.length === 0 && !pendingCard && (
          <p className='mt-4 rounded-lg bg-slate-900/70 p-3 text-[10px] leading-relaxed text-slate-500'>
            发送问题后，这里会显示本次请求的处理步骤。
          </p>
        )}
      </aside>
    </div>
  );
}

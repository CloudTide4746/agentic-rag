import { useState, type FormEvent, type KeyboardEvent } from 'react'
import { useSearchParams } from 'react-router'
import { useAuth } from '../../auth'
import { ErrorMessage, Pagination, RequestState, usePage } from '../../../shared/ui/Crud'
import { useChatStore } from '../../../store/chat'
import { sessionPageSize, useChatActions, useChatQueries } from '../api/queries'

export default function ChatPage() {
  const { user } = useAuth()
  const [params, setParams] = useSearchParams()
  const [page, setPage] = usePage()
  const sessionId = params.get('session') || null
  const { sessions, messages } = useChatQueries(user?.id, sessionId, page)
  const actions = useChatActions(user?.id)
  const stream = useChatStore()
  const [question, setQuestion] = useState('')
  const [error, setError] = useState<unknown>(null)
  const busy = stream.streaming || actions.send.isPending || actions.create.isPending
  const current = sessions.data?.items.find(session => session.id === sessionId)

  function selectSession(id: string) {
    stream.resetDecision()
    setError(null)
    setParams(previous => { const next = new URLSearchParams(previous); next.set('session', id); return next })
  }
  async function newSession() {
    if (busy) return
    setError(null)
    try { selectSession((await actions.create.mutateAsync()).id) }
    catch (cause) { setError(cause) }
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const text = question.trim()
    if (!text || !sessionId || busy || messages.isError) return
    setError(null)
    try {
      await actions.send.mutateAsync({ sessionId, question: text })
      setQuestion('')
    } catch (cause) { setError(cause) }
  }
  function keyboard(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault()
      event.currentTarget.form?.requestSubmit()
    }
  }

  return <div className="grid min-h-[70vh] gap-4 p-4 md:grid-cols-[220px_minmax(0,1fr)] xl:grid-cols-[220px_minmax(0,1fr)_300px]">
    <aside aria-label="会话列表">
      <h1>最近会话</h1>
      <button type="button" disabled={busy} onClick={() => void newSession()}>{actions.create.isPending ? '正在创建…' : '新建会话'}</button>
      <RequestState pending={sessions.isPending} error={sessions.error} retry={() => sessions.refetch()} />
      {sessions.data?.items.length === 0 ? <p>暂无会话，点击新建开始。</p> : null}
      <ul>{sessions.data?.items.map(session => <li key={session.id}>
        <button type="button" disabled={busy} aria-current={session.id === sessionId ? 'page' : undefined} onClick={() => selectSession(session.id)}>
          {session.title || '新会话'} · {session.status}
        </button>
      </li>)}</ul>
      {sessions.data ? <Pagination page={page} total={sessions.data.total} pageSize={sessionPageSize} onPage={setPage} /> : null}
    </aside>

    <section aria-label="智能问答" className="min-w-0 space-y-4">
      <h2>{current?.title || '智能问答'}</h2>
      {!sessionId ? <p>选择或新建会话后提问。</p> : <RequestState pending={messages.isPending} error={messages.error} retry={() => messages.refetch()} />}
      {sessionId && messages.data?.length === 0 ? <p>发送第一个问题，开始这个会话。</p> : null}
      {messages.data?.map(message => <article key={message.id} className="rounded border border-slate-700 p-3">
        <p>{message.role === 'user' ? '你' : 'Agent'} · {new Date(message.created_at).toLocaleString()}</p>
        <p className="whitespace-pre-wrap">{message.content}</p>
        {message.citations?.map((citation, index) => <p key={`${citation.doc_name}-${citation.chunk_no}-${index}`}>
          [{index + 1}] {citation.doc_name} · {citation.section_path} · 切片 {citation.chunk_no}
        </p>)}
      </article>)}
      {stream.streamSessionId === sessionId && stream.question ? <article className="rounded border border-cyan-800 p-3" aria-label="本次运行">
        <p>本次提问：{stream.question}</p>
        <p className="whitespace-pre-wrap" aria-live="polite">{stream.streamingText || (stream.streaming ? '等待 Agent 返回…' : '尚未收到回答')}</p>
        {stream.streamCitations.map((citation, index) => <p key={`${citation.doc_name}-${index}`}>[{index + 1}] {citation.doc_name} · 切片 {citation.chunk_no}</p>)}
      </article> : null}
      <ErrorMessage error={error || stream.streamError} />
      {stream.streamError ? <button type="button" disabled={busy} onClick={() => void messages.refetch()}>刷新历史确认运行结果</button> : null}
      <form aria-label="问答输入" onSubmit={event => void submit(event)}>
        <label htmlFor="chat-question">输入问题</label>
        <textarea id="chat-question" className="block w-full border border-slate-600 p-2" rows={4} maxLength={4000} value={question} onChange={event => setQuestion(event.target.value)} onKeyDown={keyboard} disabled={busy || !sessionId || messages.isError || messages.isPending} />
        <button type="submit" disabled={busy || !sessionId || !question.trim() || messages.isError || messages.isPending}>发送</button>
        {stream.streaming ? <button type="button" onClick={stream.stopStream}>停止接收</button> : null}
        <p>Enter 发送 · Shift+Enter 换行</p>
      </form>
    </section>

    <aside aria-label="决策链" className="min-w-0">
      <h2>决策链</h2>
      <ol>{[...stream.cards, ...(stream.pendingCard ? [stream.pendingCard] : [])].map(card => <li key={card.uid} className="border-b border-slate-700 py-2">
        <p>{card.type} · {card.status === 'pending' ? '进行中' : card.status === 'warn' ? '需关注' : '完成'} · {card.elapsedMs}ms</p>
        <p>{card.detail}</p>
        {Object.keys(card.payload).length ? <details><summary>阶段数据</summary><pre className="overflow-auto whitespace-pre-wrap break-all text-xs">{JSON.stringify(card.payload, null, 2)}</pre></details> : null}
      </li>)}</ol>
      {stream.doneStats ? <p role="status">本次完成：{stream.doneStats.elapsed_ms}ms；输入 {stream.doneStats.prompt} token，输出 {stream.doneStats.completion} token；纠正/反思 {stream.doneStats.rounds} 轮。</p> : null}
    </aside>
  </div>
}

import type { Citation, SseEvent } from '../../types/legacy'
import type { ChatMessage, ChatSession } from './dto'
import {
  ask as askGo,
  createChat,
  getChat,
  listSessions as listGoSessions,
} from './list'

type GoChatMessage = Omit<ChatMessage, 'id' | 'session_id' | 'citations'> & {
  id: number
  session_id: string
  citations?: Citation[]
}

function isCitation(value: unknown): value is Citation {
  if (typeof value !== 'object' || value === null) return false
  const item = value as Record<string, unknown>
  return (
    typeof item.doc_name === 'string' &&
    typeof item.section_path === 'string' &&
    typeof item.chunk_no === 'number'
  )
}

/** POST /api/v1/chat/sessions */
export async function createSession(signal?: AbortSignal): Promise<ChatSession> {
  const response = await createChat({}, signal)
  return response.data.session
}

/** GET /api/v1/chat/sessions, collecting all pages for the existing chat list. */
export async function listSessions(signal?: AbortSignal): Promise<ChatSession[]> {
  const pageSize = 100
  const first = await listGoSessions({ page: 1, page_size: pageSize }, signal)
  const sessions = [...first.data.items]
  const pageCount = Math.ceil(first.data.total / pageSize)

  for (let page = 2; page <= pageCount; page += 1) {
    const response = await listGoSessions({ page, page_size: pageSize }, signal)
    sessions.push(...response.data.items)
  }
  return sessions
}

/** GET /api/v1/chat/sessions/:id */
export async function getMessages(sessionId: string, signal?: AbortSignal): Promise<GoChatMessage[]> {
  const response = await getChat(sessionId, signal)
  return response.data.messages.map((message, index) => ({
    ...message,
    id: index + 1,
    session_id: sessionId,
    citations: Array.isArray(message.citations)
      ? message.citations.filter(isCitation)
      : undefined,
  }))
}

/** POST /api/v1/chat/sessions/:id/runs */
export function ask(
  sessionId: string,
  question: string,
  onEvent: (event: SseEvent) => void,
  signal?: AbortSignal,
  images?: string[],
): Promise<void> {
  return askGo(
    sessionId,
    { question, ...(images?.length ? { images } : {}) },
    onEvent,
    signal,
  )
}

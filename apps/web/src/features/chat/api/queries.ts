import { useEffect, useRef } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createSession, getMessages } from '../../../lib/api/chat'
import { listSessions } from '../../../lib/api/list'
import { useChatStore } from '../../../store/chat'

export const sessionPageSize = 20
const keys = {
  sessions: (userId?: string) => ['chat', userId, 'sessions'] as const,
  messages: (userId: string | undefined, sessionId: string | null) => ['chat', userId, 'messages', sessionId] as const,
}

export function useChatQueries(userId: string | undefined, sessionId: string | null, page: number) {
  const sessions = useQuery({
    queryKey: [...keys.sessions(userId), { page, pageSize: sessionPageSize }],
    enabled: Boolean(userId),
    queryFn: async ({ signal }) => (await listSessions({ page, page_size: sessionPageSize }, signal)).data,
    retry: false,
  })
  const messages = useQuery({
    queryKey: keys.messages(userId, sessionId),
    enabled: Boolean(userId && sessionId),
    queryFn: ({ signal }) => {
      if (!sessionId) throw new Error('请先选择会话')
      return getMessages(sessionId, signal)
    },
    retry: false,
  })
  return { sessions, messages }
}

export function useChatActions(userId: string | undefined) {
  const client = useQueryClient()
  const controller = useRef<AbortController | null>(null)
  useEffect(() => () => {
    controller.current?.abort()
    useChatStore.getState().resetDecision()
  }, [userId])
  const create = useMutation({
    mutationFn: async () => {
      controller.current = new AbortController()
      return createSession(controller.current.signal)
    },
    onSuccess: () => client.invalidateQueries({ queryKey: keys.sessions(userId) }),
    retry: false,
  })
  const send = useMutation({
    mutationFn: async ({ sessionId, question }: { sessionId: string; question: string }) => {
      try {
        await useChatStore.getState().send(sessionId, question)
      } finally {
        if (useChatStore.getState().streamSessionId === sessionId) {
          await client.invalidateQueries({ queryKey: keys.messages(userId, sessionId) })
          if (useChatStore.getState().streamSessionId !== sessionId) return
          await client.invalidateQueries({ queryKey: keys.sessions(userId) })
          const stream = useChatStore.getState()
          const history = client.getQueryState(keys.messages(userId, sessionId))
          if (stream.streamSessionId === sessionId && stream.doneStats && !stream.streamError && history?.status === 'success' && !history.isInvalidated) stream.dismissDraft()
        }
      }
    },
    retry: false,
  })
  return { create, send }
}

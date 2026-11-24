import { useEffect, useRef } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { addTeam, deleteTeam, getTeamById, getTeamList, updateTeam, type CreateTeamReq, type UpdateTeamReq } from '../../../lib/api/list'

function useMutationRequest(scope: string) {
  const controllers = useRef(new Set<AbortController>())
  useEffect(() => {
    const pending = controllers.current
    return () => {
      pending.forEach(controller => controller.abort())
      pending.clear()
    }
  }, [scope])
  return async function request<T>(action: (signal: AbortSignal) => Promise<T>): Promise<T> {
    const controller = new AbortController()
    controllers.current.add(controller)
    try {
      const result = await action(controller.signal)
      controller.signal.throwIfAborted()
      return result
    } finally {
      controllers.current.delete(controller)
    }
  }
}

export const TEAM_PAGE_SIZE = 20
const keys = {
  all: (userId: string) => ['teams', userId] as const,
  list: (userId: string, page: number) => [...keys.all(userId), 'list', page, TEAM_PAGE_SIZE] as const,
  detail: (userId: string, id: number | null) => [...keys.all(userId), 'detail', id] as const,
}

export function useTeams(userId: string, page: number) {
  return useQuery({
    queryKey: keys.list(userId, page),
    queryFn: async ({ signal }) => (await getTeamList({ page, page_size: TEAM_PAGE_SIZE }, signal)).data,
    enabled: Boolean(userId),
  })
}

export function useTeam(userId: string, id: number | null) {
  return useQuery({
    queryKey: keys.detail(userId, id),
    queryFn: async ({ signal }) => {
      if (id === null) throw new Error('团队编号无效')
      return (await getTeamById(id, signal)).data.team
    },
    enabled: Boolean(userId) && id !== null,
  })
}

export function useCreateTeam(userId: string) {
  const client = useQueryClient()
  const request = useMutationRequest(userId)
  return useMutation({
    mutationFn: (body: CreateTeamReq) => request(async signal => (await addTeam(body, signal)).data.team),
    retry: false,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: keys.all(userId) })
    },
  })
}

export function useUpdateTeam(userId: string, id: number) {
  const client = useQueryClient()
  const request = useMutationRequest(`${userId}:${id}`)
  return useMutation({
    mutationFn: (body: UpdateTeamReq) => request(async signal => (await updateTeam(id, body, signal)).data.team),
    retry: false,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: keys.all(userId) })
    },
  })
}

export function useDeleteTeam(userId: string, id: number) {
  const client = useQueryClient()
  const request = useMutationRequest(`${userId}:${id}`)
  return useMutation({
    mutationFn: () => request(signal => deleteTeam(id, signal)),
    retry: false,
    onSuccess: async () => {
      client.removeQueries({ queryKey: keys.detail(userId, id) })
      await client.invalidateQueries({ queryKey: keys.all(userId) })
    },
  })
}

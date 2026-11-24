import { useEffect, useRef } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { addProject, deleteProject, getProjectById, getProjectList, updateProject, type CreateProjectReq, type UpdateProjectReq } from '../../../lib/api/list'

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

export const PROJECT_PAGE_SIZE = 20
const keys = {
  all: (userId: string, teamId: number | null) => ['projects', userId, teamId] as const,
  list: (userId: string, teamId: number | null, page: number) => [...keys.all(userId, teamId), 'list', page, PROJECT_PAGE_SIZE] as const,
  detail: (userId: string, teamId: number | null, id: number | null) => [...keys.all(userId, teamId), 'detail', id] as const,
}

export function useProjects(userId: string, teamId: number | null, page: number) {
  return useQuery({
    queryKey: keys.list(userId, teamId, page),
    queryFn: async ({ signal }) => {
      if (teamId === null) throw new Error('团队编号无效')
      return (await getProjectList(teamId, { page, page_size: PROJECT_PAGE_SIZE }, signal)).data
    },
    enabled: Boolean(userId) && teamId !== null,
  })
}

export function useProject(userId: string, teamId: number | null, id: number | null) {
  return useQuery({
    queryKey: keys.detail(userId, teamId, id),
    queryFn: async ({ signal }) => {
      if (id === null || teamId === null) throw new Error('项目编号无效')
      const project = (await getProjectById(id, signal)).data.project
      if (project.team_id !== String(teamId)) throw new Error('项目不属于当前团队，请从正确的团队进入。')
      return project
    },
    enabled: Boolean(userId) && teamId !== null && id !== null,
  })
}

export function useCreateProject(userId: string, teamId: number) {
  const client = useQueryClient()
  const request = useMutationRequest(`${userId}:${teamId}`)
  return useMutation({
    mutationFn: (body: CreateProjectReq) => request(async signal => (await addProject(teamId, body, signal)).data.project),
    retry: false,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: keys.all(userId, teamId) })
    },
  })
}

export function useUpdateProject(userId: string, teamId: number, id: number) {
  const client = useQueryClient()
  const request = useMutationRequest(`${userId}:${teamId}:${id}`)
  return useMutation({
    mutationFn: (body: UpdateProjectReq) => request(async signal => (await updateProject(id, body, signal)).data.project),
    retry: false,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: keys.all(userId, teamId) })
    },
  })
}

export function useDeleteProject(userId: string, teamId: number, id: number) {
  const client = useQueryClient()
  const request = useMutationRequest(`${userId}:${teamId}:${id}`)
  return useMutation({
    mutationFn: () => request(signal => deleteProject(id, signal)),
    retry: false,
    onSuccess: async () => {
      client.removeQueries({ queryKey: keys.detail(userId, teamId, id) })
      await client.invalidateQueries({ queryKey: keys.all(userId, teamId) })
    },
  })
}

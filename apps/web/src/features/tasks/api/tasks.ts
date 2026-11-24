import { useEffect, useRef } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ApiError, addTask, deleteTask, getProjectById, getTaskById, getTaskList, updateTask,
  type CreateTaskReq, type Task, type UpdateTaskReq,
} from '../../../lib/api/list'

export const taskPageSize = 20

export function isTaskConflict(error: unknown): boolean {
  return error instanceof ApiError && error.status === 409
}
const keys = {
  project: (userId: string | undefined, teamId: number | null, projectId: number | null) =>
    ['tasks', userId, teamId, projectId] as const,
  list: (userId: string | undefined, teamId: number | null, projectId: number | null) =>
    [...keys.project(userId, teamId, projectId), 'list'] as const,
  detail: (userId: string | undefined, teamId: number | null, projectId: number | null, taskId: number | null) =>
    [...keys.project(userId, teamId, projectId), 'detail', taskId] as const,
}

export function useTaskProject(userId: string | undefined, teamId: number | null, projectId: number | null) {
  return useQuery({
    queryKey: [...keys.project(userId, teamId, projectId), 'parent'],
    enabled: Boolean(userId && teamId && projectId),
    queryFn: async ({ signal }) => {
      if (!teamId || !projectId) throw new Error('项目地址无效')
      const { data } = await getProjectById(projectId, signal)
      if (data.project.team_id !== String(teamId)) throw new Error('项目不属于当前团队，请检查地址')
      return data.project
    },
    retry: false,
  })
}

export function useTasks(userId: string | undefined, teamId: number | null, projectId: number | null, page: number, enabled: boolean) {
  return useQuery({
    queryKey: [...keys.list(userId, teamId, projectId), { page, page_size: taskPageSize }],
    enabled: Boolean(userId && projectId && enabled),
    queryFn: async ({ signal }) => {
      if (!projectId) throw new Error('项目地址无效')
      const { data } = await getTaskList({ project_id: String(projectId), page, page_size: taskPageSize }, signal)
      if (data.items.some(task => task.project_id !== String(projectId))) throw new Error('任务列表归属无效')
      return data
    },
    retry: false,
  })
}

export function useTask(userId: string | undefined, teamId: number | null, projectId: number | null, taskId: number | null, enabled: boolean) {
  return useQuery({
    queryKey: keys.detail(userId, teamId, projectId, taskId),
    enabled: Boolean(userId && projectId && taskId && enabled),
    queryFn: async ({ signal }) => {
      if (!taskId || !projectId) throw new Error('任务地址无效')
      const { data } = await getTaskById(taskId, signal)
      if (data.task.project_id !== String(projectId)) throw new Error('任务不属于当前项目，请检查地址')
      return data.task
    },
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    retry: false,
  })
}

export function useTaskMutations(userId: string | undefined, teamId: number, projectId: number, taskId?: number) {
  const client = useQueryClient()
  const controllers = useRef(new Set<AbortController>())
  useEffect(() => {
    const pending = controllers.current
    return () => {
      pending.forEach(controller => controller.abort())
      pending.clear()
    }
  }, [userId, teamId, projectId, taskId])
  async function request<T>(action: (signal: AbortSignal) => Promise<T>): Promise<T> {
    const controller = new AbortController()
    controllers.current.add(controller)
    try {
      const result = await action(controller.signal)
      controller.signal.throwIfAborted()
      return result
    }
    finally { controllers.current.delete(controller) }
  }
  async function saved(task: Task): Promise<void> {
    client.setQueryData(keys.detail(userId, teamId, projectId, Number(task.id)), task)
    await client.invalidateQueries({ queryKey: keys.list(userId, teamId, projectId) })
  }
  const create = useMutation({
    mutationFn: (body: Omit<CreateTaskReq, 'project_id'>) => request(async signal => {
      const response = await addTask({ ...body, project_id: String(projectId) }, signal)
      return response.data.task
    }),
    onSuccess: saved,
    retry: false,
  })
  const update = useMutation({
    mutationFn: (body: UpdateTaskReq & { version: number }) => request(async signal => {
      if (!taskId) throw new Error('任务地址无效')
      const response = await updateTask(taskId, body, signal)
      return response.data.task
    }),
    onSuccess: saved,
    retry: false,
  })
  const remove = useMutation({
    mutationFn: () => request(async signal => {
      if (!taskId) throw new Error('任务地址无效')
      await deleteTask(taskId, signal)
    }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: keys.detail(userId, teamId, projectId, taskId ?? null), refetchType: 'none' })
      await client.invalidateQueries({ queryKey: keys.list(userId, teamId, projectId) })
    },
    retry: false,
  })
  return { create, update, remove }
}

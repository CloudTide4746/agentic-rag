import type { ApiResponse } from '../types/api'

export async function get<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`/api/v1${path}`, { signal, credentials: 'same-origin', headers: { Accept: 'application/json' } })
  if (!response.ok) throw new Error(`请求失败（HTTP ${response.status}）`)
  const body: ApiResponse<T> = await response.json()
  return body.data
}

import { useQuery } from '@tanstack/react-query'
import { get } from '../../lib/http'
import type { Health } from '../../types/api'

export default function Overview() {
  const health = useQuery({ queryKey: ['system', 'health'], queryFn: ({ signal }) => get<Health>('/health/live', signal) })
  return <main>
    <p className="mb-3 text-sm font-medium text-teal-700">工程初始化</p>
    <h1 className="mb-4 text-4xl font-semibold tracking-tight">企业内部知识库问答平台</h1>
    <p className="max-w-2xl leading-7 text-slate-600">React 前端与 Go 业务入口已建立。接下来接入登录、知识库和流式问答，复用现有 Python Agent。</p>
    <section className="my-8 rounded-xl border border-slate-200 bg-white p-6" aria-labelledby="health-title">
      <h2 id="health-title" className="mb-3 text-lg font-semibold">服务连接</h2>
      {health.isPending && <p role="status">正在连接 Go API…</p>}
      {health.isError && <p role="alert" className="text-red-700">连接失败，请确认 Go API 已启动。{health.error.message}</p>}
      {health.isSuccess && <p role="status" className="text-teal-700">Go API 已连接 · {health.data.status}</p>}
      <button className="mt-4 rounded-lg bg-slate-900 px-4 py-2 text-white disabled:opacity-50" disabled={health.isFetching} onClick={() => void health.refetch()}>重新检查</button>
    </section>
    <p className="text-sm text-slate-500">当前仅验证工程与 API 连通性；认证、检索、任务处理和统计尚未接入。</p>
  </main>
}

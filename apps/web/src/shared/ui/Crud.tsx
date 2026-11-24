import { useId, type HTMLInputTypeAttribute } from 'react'
import { useSearchParams } from 'react-router'

export function ErrorMessage({ error }: { error?: unknown }) {
  if (!error) return null
  return <p role="alert">{error instanceof Error ? error.message : typeof error === 'string' ? error : '请求失败，请稍后重试。'}</p>
}

export function RequestState({ pending, error, retry }: { pending: boolean; error?: unknown; retry?: () => unknown }) {
  if (pending) return <p role="status">正在加载…</p>
  if (!error) return null
  return <div><ErrorMessage error={error} />{retry ? <button type="button" onClick={() => void retry()}>重新加载</button> : null}</div>
}

export function FormField({ label, name, defaultValue, type = 'text', required, maxLength, multiline }: {
  label: string; name: string; defaultValue?: string; type?: HTMLInputTypeAttribute; required?: boolean; maxLength?: number; multiline?: boolean
}) {
  const id = useId()
  return <label className="tf-field" htmlFor={id}>
    <span>{label}</span>
    {multiline ? <textarea id={id} name={name} defaultValue={defaultValue} required={required} maxLength={maxLength} rows={4} /> :
      <input id={id} name={name} type={type} defaultValue={defaultValue} required={required} maxLength={maxLength} />}
  </label>
}

export function Pagination({ page, total, pageSize, onPage }: { page: number; total: number; pageSize: number; onPage: (page: number) => void }) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  return <nav aria-label="列表分页">
    <button type="button" disabled={page <= 1} onClick={() => onPage(page - 1)}>上一页</button>
    <span> 第 {page} 页，共 {pages} 页 · {total} 条 </span>
    <button type="button" disabled={page >= pages} onClick={() => onPage(page + 1)}>下一页</button>
  </nav>
}

export function parseResourceId(value: string | undefined): number | null {
  if (!value || !/^[1-9]\d*$/.test(value)) return null
  const id = Number(value)
  return Number.isSafeInteger(id) ? id : null
}

export function usePage(): [number, (page: number) => void] {
  const [params, setParams] = useSearchParams()
  const page = parseResourceId(params.get('page') ?? undefined) ?? 1
  return [page, (next) => setParams(previous => {
    const updated = new URLSearchParams(previous)
    updated.set('page', String(next))
    return updated
  })]
}

export function dateInputValue(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return ''
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

export function dateRequestValue(value: string): string | undefined {
  if (!value) return undefined
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) throw new Error('日期格式无效')
  return date.toISOString()
}

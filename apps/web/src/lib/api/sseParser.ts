import type { Citation, SseEvent } from '../../types/legacy'

function object(value: unknown): Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) throw new Error('SSE 事件格式无效')
  return value as Record<string, unknown>
}
function text(value: unknown): string {
  if (typeof value !== 'string') throw new Error('SSE 文本字段无效')
  return value
}
function number(value: unknown): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) throw new Error('SSE 数值字段无效')
  return value
}
function texts(value: unknown): string[] {
  if (!Array.isArray(value)) throw new Error('SSE 列表字段无效')
  return value.map(text)
}
function boolean(value: unknown): boolean {
  if (typeof value !== 'boolean') throw new Error('SSE 布尔字段无效')
  return value
}
function choice<T extends string>(value: unknown, choices: readonly T[]): T {
  const found = choices.find(item => item === value)
  if (!found) throw new Error('SSE 枚举字段无效')
  return found
}
function citation(value: unknown): Citation {
  const item = object(value)
  return { doc_name: text(item.doc_name), section_path: text(item.section_path), chunk_no: number(item.chunk_no) }
}
function event(name: string, value: unknown): SseEvent {
  const v = object(value)
  const base = { step: number(v.step), ts: number(v.ts), detail: text(v.detail) }
  switch (name) {
    case 'router': return { ...base, type: name, intent: choice(v.intent, ['chitchat', 'kb_qa', 'data_query', 'summarize']), confidence: number(v.confidence) }
    case 'rewrite': return { ...base, type: name, rewritten_query: text(v.rewritten_query), sub_queries: texts(v.sub_queries), hyde_used: boolean(v.hyde_used) }
    case 'retrieve': return { ...base, type: name, source: choice(v.source, ['vector', 'keyword', 'database']), hit_count: number(v.hit_count) }
    case 'grade': return { ...base, type: name, score: number(v.score), verdict: choice(v.verdict, ['pass', 'partial', 'fail']), reason: text(v.reason) }
    case 'correct': return { ...base, type: name, action: choice(v.action, ['rewrite_retry', 'give_up']), round: number(v.round) }
    case 'generate': return { ...base, type: name, delta: text(v.delta) }
    case 'reflect': return { ...base, type: name, result: choice(v.result, ['pass', 'fail']), round: number(v.round), missing_info: v.missing_info === undefined ? undefined : text(v.missing_info) }
    case 'citation': {
      if (!Array.isArray(v.citations)) throw new Error('SSE 引用列表无效')
      return { ...base, type: name, citations: v.citations.map(citation) }
    }
    case 'done': {
      const usage = object(v.token_usage)
      return { ...base, type: name, message_id: number(v.message_id), elapsed_ms: number(v.elapsed_ms), token_usage: { prompt: number(usage.prompt), completion: number(usage.completion) } }
    }
    case 'error': return { ...base, type: name, code: number(v.code), message: text(v.message) }
    default: throw new Error('不支持的 SSE 事件')
  }
}
const eventNames = new Set(['router', 'rewrite', 'retrieve', 'grade', 'correct', 'generate', 'reflect', 'citation', 'done', 'error'])

/** Streaming line parser: a CRLF pair may be split across two network chunks. */
export class SseParser {
  private buffer = ''
  private eventName = 'message'
  private lines: string[] = []
  private eventSize = 0

  feed(chunk: string): SseEvent[] {
    this.buffer += chunk
    const result: SseEvent[] = []
    for (;;) {
      const index = this.buffer.search(/[\r\n]/)
      if (index < 0 || (this.buffer[index] === '\r' && index === this.buffer.length - 1)) break
      const separator = this.buffer[index] === '\r' && this.buffer[index + 1] === '\n' ? 2 : 1
      const line = this.buffer.slice(0, index)
      this.buffer = this.buffer.slice(index + separator)
      const parsed = this.line(line)
      if (parsed) result.push(parsed)
    }
    if (this.buffer.length + this.eventSize > 1_048_576) throw new Error('SSE 单个事件超出大小限制')
    return result
  }

  finish(): SseEvent[] {
    // A trailing CR is a complete line separator, not an incomplete CRLF pair.
    return this.buffer.endsWith('\r') ? this.feed('\n') : []
  }

  private line(line: string): SseEvent | null {
    if (!line) {
      const name = this.eventName
      const lines = this.lines
      this.eventName = 'message'
      this.lines = []
      this.eventSize = 0
      if (!lines.length || !eventNames.has(name)) return null
      let data: unknown
      try { data = JSON.parse(lines.join('\n')) }
      catch { throw new Error('SSE 事件不是有效 JSON') }
      return event(name, data)
    }
    if (line.startsWith(':')) return null
    const colon = line.indexOf(':')
    const field = colon < 0 ? line : line.slice(0, colon)
    let value = colon < 0 ? '' : line.slice(colon + 1)
    if (value.startsWith(' ')) value = value.slice(1)
    if (field === 'event') this.eventName = value
    if (field === 'data') {
      this.lines.push(value)
      this.eventSize += value.length
      if (this.eventSize > 1_048_576) throw new Error('SSE 单个事件超出大小限制')
    }
    return null
  }
}

import type { DecisionCard } from '../store/chat'

/* ============================================================
 * 决策链导出 — Markdown（复制/下载）与 JSON（下载）
 * 时间线卡片 → 教程素材 / 排查归档两用
 * ============================================================ */

export interface DecisionExportStats {
  elapsedMs: number
  rounds: number
  prompt?: number
  completion?: number
}

const STEP_LABEL: Record<number, string> = {
  1: '意图路由',
  2: '查询规划',
  3: '多路检索',
  4: '质量评估',
  5: 'CRAG 纠正',
  6: '生成与反思',
  7: '引用溯源',
}

const TYPE_LABEL: Record<string, string> = {
  router: '意图路由',
  rewrite: '查询规划',
  retrieve: '多路检索',
  grade: '质量评估',
  correct: 'CRAG 纠正',
  generate: '生成答案',
  reflect: '自我反思',
  citation: '引用溯源',
  done: '完成',
  error: '异常中断',
}

export function stepName(card: DecisionCard): string {
  return card.step > 0 ? (STEP_LABEL[card.step] ?? TYPE_LABEL[card.type] ?? '决策') : (TYPE_LABEL[card.type] ?? '流程')
}

/** 耗时格式化：<1s 显示毫秒，否则一位小数秒 */
export function fmtMs(ms: number): string {
  return ms < 1000 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(1)}s`
}

/** 决策链 → Markdown */
export function toDecisionMarkdown(question: string, cards: DecisionCard[], stats: DecisionExportStats | null): string {
  const lines: string[] = []
  lines.push('# Agentic RAG 决策链', '')
  lines.push(`**问题**：${question}`, '')
  lines.push('> 由问答页决策侧栏导出，与 agent_decision_log.steps 同源', '')
  lines.push('| # | 环节 | 状态 | 耗时 | 说明 |', '|---|------|------|------|------|')
  for (const c of cards) {
    const no = c.step > 0 ? c.step : '-'
    const status = c.status === 'warn' ? '⚠ 告警' : '✓ 完成'
    const detail = c.detail.replace(/\|/g, '\\|') || '-'
    lines.push(`| ${no} | ${stepName(c)}${c.round > 0 ? `（R${c.round}）` : ''} | ${status} | ${fmtMs(c.elapsedMs)} | ${detail} |`)
  }
  lines.push('')
  if (stats) {
    const tokens = stats.prompt != null && stats.completion != null ? `${stats.prompt + stats.completion}（prompt ${stats.prompt} / completion ${stats.completion}）` : '—'
    lines.push(`**汇总**：总耗时 ${fmtMs(stats.elapsedMs)} · tokens ${tokens} · 检索轮次 ${stats.rounds}`)
  }
  lines.push('')
  return lines.join('\n')
}

/** 决策链 → 可序列化 JSON 结构（对齐 agent_decision_log 字段） */
export function buildDecisionJson(question: string, cards: DecisionCard[], stats: DecisionExportStats | null) {
  return {
    question,
    exported_at: new Date().toISOString(),
    steps: cards.map((c) => ({
      step: c.step,
      type: c.type,
      detail: c.detail,
      status: c.status,
      round: c.round,
      elapsed_ms: c.elapsedMs,
      payload: c.payload,
    })),
    summary: stats
      ? {
          elapsed_ms: stats.elapsedMs,
          total_rounds: stats.rounds,
          token_usage: stats.prompt != null && stats.completion != null ? { prompt: stats.prompt, completion: stats.completion } : null,
        }
      : null,
  }
}

/** 触发浏览器下载 JSON 文件 */
export function downloadDecisionJson(question: string, cards: DecisionCard[], stats: DecisionExportStats | null): void {
  const blob = new Blob([JSON.stringify(buildDecisionJson(question, cards, stats), null, 2)], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  const stamp = new Date().toISOString().slice(0, 19).replace(/[T:]/g, '-')
  a.href = url
  a.download = `decision-chain-${stamp}.json`
  a.click()
  URL.revokeObjectURL(url)
}

/** 复制文本（clipboard API + 降级 execCommand） */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  }
}

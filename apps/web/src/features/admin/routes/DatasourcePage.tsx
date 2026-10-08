const sources = [
  { name: '向量检索', type: 'VECTOR SEARCH', description: '基于向量相似度查找语义相关的文档片段。', detail: 'Elasticsearch · kNN', on: true, color: 'cyan' },
  { name: '关键词检索', type: 'KEYWORD SEARCH', description: '通过 BM25 关键词匹配补充精确术语召回。', detail: 'Elasticsearch · BM25', on: true, color: 'violet' },
  { name: '业务数据库', type: 'BUSINESS DATABASE', description: '只读数据源，用于结构化业务信息查询。', detail: '受限连接 · 示例状态', on: false, color: 'amber' },
]

export default function DatasourcePage() {
  return <div className="mx-auto max-w-5xl px-4 py-6 sm:px-7 sm:py-8">
    <header className="mb-6 flex flex-wrap items-end justify-between gap-4"><div><p className="font-mono text-[10px] tracking-[0.2em] text-cyan-300">RETRIEVAL CONFIGURATION</p><h1 className="mt-2 text-xl font-semibold text-slate-100">数据源配置</h1><p className="mt-2 max-w-2xl text-xs leading-5 text-slate-400">管理检索流程中可用的数据源。当前内容仅展示配置页面结构。</p></div><button type="button" disabled className="rounded-lg border border-cyan-800 bg-cyan-950/50 px-4 py-2.5 text-xs text-cyan-100 opacity-60">✓ 保存配置</button></header>
    <div className="grid gap-3 md:grid-cols-2">{sources.map(source => <article key={source.type} className="rounded-xl border border-slate-800 bg-[#0b111d] p-4 sm:p-5">
      <div className="flex items-start gap-3"><span className={`mt-1 grid size-9 shrink-0 place-items-center rounded-lg border text-sm ${source.color === 'cyan' ? 'border-cyan-900 bg-cyan-950/50 text-cyan-300' : source.color === 'violet' ? 'border-violet-900 bg-violet-950/50 text-violet-300' : 'border-amber-900 bg-amber-950/50 text-amber-300'}`} aria-hidden="true">⌘</span><div className="min-w-0 flex-1"><p className="font-mono text-[9px] tracking-wider text-slate-500">{source.type}</p><h2 className="mt-1 text-sm font-semibold text-slate-100">{source.name}</h2></div><span className={`rounded-full border px-2 py-1 text-[9px] ${source.on ? 'border-emerald-900 bg-emerald-950/40 text-emerald-200' : 'border-slate-700 bg-slate-900 text-slate-400'}`}>{source.on ? '已启用' : '已停用'}</span></div>
      <p className="mt-4 text-xs leading-5 text-slate-400">{source.description}</p><div className="mt-4 flex items-center justify-between border-t border-slate-800 pt-3"><span className="font-mono text-[10px] text-slate-500">{source.detail}</span><button type="button" disabled aria-label={`${source.name}开关（布局预览）`} aria-pressed={source.on} className={`relative h-5 w-9 rounded-full border border-slate-700 opacity-60 ${source.on ? 'bg-cyan-900' : 'bg-slate-800'}`}><span className={`absolute top-0.5 size-3.5 rounded-full bg-slate-200 ${source.on ? 'left-4' : 'left-0.5'}`} /></button></div>
    </article>)}</div>
    <section aria-labelledby="datasource-note" className="mt-5 grid gap-3 rounded-xl border border-slate-800 bg-slate-900/40 p-4 text-xs leading-5 text-slate-400 md:grid-cols-2"><h2 id="datasource-note" className="sr-only">数据源说明</h2><p><span className="text-cyan-200">向量 + 关键词</span>：混合检索用于补充语义召回与精确词匹配。</p><p><span className="text-amber-200">结构化数据</span>：此区域用于展示连接方式和安全边界说明。</p><p className="md:col-span-2"><span className="text-slate-200">状态说明</span>：示例中已启用 2/3 路；开关和保存动作均为静态展示。</p></section>
  </div>
}

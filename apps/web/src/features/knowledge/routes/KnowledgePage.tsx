const libraries = [
  { name: '产品与技术文档', desc: '产品手册、架构与研发规范', docs: 38, chunks: 1240, active: true },
  { name: '企业制度', desc: '内部政策与流程文件', docs: 16, chunks: 508 },
  { name: '运维知识库', desc: '告警、排障与值班手册', docs: 24, chunks: 732 },
]

const documents = [
  { name: '平台开发规范.pdf', ext: 'PDF', chunks: 84, state: '已完成', date: '2026-09-22 16:20', tone: 'emerald' },
  { name: '知识库接入指南.docx', ext: 'DOCX', chunks: 42, state: '处理中 · 切片', date: '2026-09-23 09:15', tone: 'amber' },
  { name: 'RAG 检索设计.md', ext: 'MD', chunks: 0, state: '处理失败', date: '2026-09-21 18:04', tone: 'rose' },
  { name: '安全策略说明.txt', ext: 'TXT', chunks: 27, state: '已完成', date: '2026-09-20 11:32', tone: 'emerald' },
]

export default function KnowledgePage() {
  return <div className="grid min-h-[calc(100vh-7.5rem)] grid-cols-1 md:grid-cols-[260px_minmax(0,1fr)]">
    <aside aria-label="知识库列表" className="border-b border-slate-800 bg-[#0b111d] p-4 md:border-b-0 md:border-r">
      <div className="mb-4 flex items-center justify-between"><h1 className="text-sm font-semibold text-slate-100">知识库</h1><button type="button" disabled className="rounded-md border border-cyan-800/70 bg-cyan-950/40 px-2.5 py-1.5 text-xs text-cyan-200 opacity-60">＋ 新建</button></div>
      <ul className="grid gap-2 sm:grid-cols-3 md:grid-cols-1">{libraries.map(kb => <li key={kb.name}><div className={`rounded-xl border p-3 ${kb.active ? 'border-cyan-800/70 bg-cyan-950/25' : 'border-slate-800 bg-slate-900/50'}`}>
        <p className="text-xs font-medium text-slate-100">▤　{kb.name}</p><p className="mt-1.5 text-[10px] text-slate-500">{kb.desc}</p><p className="mt-2 font-mono text-[10px] text-slate-400"><span className="text-cyan-300">{kb.docs}</span> 文档　<span className="text-cyan-300">{kb.chunks}</span> 切片</p>
      </div></li>)}</ul>
      <p className="mt-4 hidden border-t border-slate-800 pt-3 text-[10px] leading-relaxed text-slate-500 md:block">列表内容为静态样例。工作空间权限和远端数据尚未接入。</p>
    </aside>

    <section aria-labelledby="documents-title" className="min-w-0">
      <header className="flex flex-wrap items-center gap-3 border-b border-slate-800 bg-slate-900/40 px-4 py-3 sm:px-5"><div><h2 id="documents-title" className="text-sm font-medium text-slate-100">产品与技术文档</h2><p className="mt-0.5 text-[10px] text-slate-500">文档管理 · 4 个示例</p></div><div className="ml-auto flex items-center gap-2"><span className="rounded border border-amber-900/70 bg-amber-950/30 px-2 py-1 text-[10px] text-amber-200">1 个处理中</span><button type="button" disabled className="rounded-lg border border-cyan-800 bg-cyan-950/50 px-3 py-2 text-xs text-cyan-100 opacity-60">↑ 上传文档</button></div></header>
      <div className="overflow-x-auto p-3 sm:p-5">
        <table className="w-full min-w-[620px] border-separate border-spacing-0 overflow-hidden rounded-xl border border-slate-800 text-left text-xs"><caption className="sr-only">文档静态样例列表</caption><thead className="bg-slate-900 text-[10px] uppercase tracking-wider text-slate-500"><tr><th scope="col" className="px-3 py-3 font-medium">文档</th><th scope="col" className="px-3 py-3 font-medium">切片</th><th scope="col" className="px-3 py-3 font-medium">入库状态</th><th scope="col" className="px-3 py-3 font-medium">更新时间</th><th scope="col" className="px-3 py-3 text-right font-medium">操作预览</th></tr></thead>
          <tbody>{documents.map(doc => <tr key={doc.name} className="border-t border-slate-800 bg-[#0b111d] [&>td]:border-t [&>td]:border-slate-800/80"><td className="px-3 py-3"><div className="flex items-center gap-2"><span className="grid size-8 shrink-0 place-items-center rounded-md border border-slate-700 bg-slate-900 font-mono text-[9px] text-cyan-200">{doc.ext}</span><span className="font-medium text-slate-200">{doc.name}</span></div></td><td className="px-3 py-3 font-mono text-slate-400">{doc.chunks || '—'}</td><td className="px-3 py-3"><span className={`rounded-full border px-2 py-1 text-[10px] ${doc.tone === 'emerald' ? 'border-emerald-900 bg-emerald-950/40 text-emerald-200' : doc.tone === 'amber' ? 'border-amber-900 bg-amber-950/40 text-amber-200' : 'border-rose-900 bg-rose-950/40 text-rose-200'}`}>{doc.state}</span></td><td className="px-3 py-3 font-mono text-[10px] text-slate-500">{doc.date}</td><td className="px-3 py-3 text-right text-slate-600">切片　···</td></tr>)}</tbody>
        </table>
        <p className="mt-3 text-[10px] text-slate-500">上传、删除、重试和切片查看操作尚未接入。</p>
      </div>
    </section>
  </div>
}

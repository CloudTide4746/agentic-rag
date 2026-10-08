import { useState } from 'react'
import { NavLink } from 'react-router'

const navItems = [
  { label: '智能问答', to: '/chat', icon: '◉' },
  { label: '知识库', to: '/admin/kb', icon: '▤' },
  { label: '数据源', to: '/admin/datasources', icon: '⌘' },
  { label: '检索测试', to: '/admin/playground', icon: '⌕' },
  { label: '决策日志', to: '/admin/logs', icon: '≋' },
]

export function TopNavBar() {
  const [menuOpen, setMenuOpen] = useState(false)
  return <header className="relative z-20 border-b border-slate-800 bg-[#0b111d]/95 backdrop-blur">
    <div className="flex h-14 items-center px-4 sm:px-5">
      <NavLink to="/chat" className="flex min-w-0 items-center gap-3" aria-label="Agentic RAG 首页">
        <span aria-hidden="true" className="grid size-8 shrink-0 place-items-center rounded-lg border border-cyan-400/35 bg-cyan-400/10 text-cyan-300">⌘</span>
        <span className="min-w-0 leading-none"><span className="block text-sm font-bold tracking-[0.12em] text-slate-100">AGENTIC<span className="text-cyan-300"> RAG</span></span><span className="mt-1 hidden font-mono text-[9px] tracking-[0.16em] text-slate-500 sm:block">AI KNOWLEDGE QA SYSTEM</span></span>
      </NavLink>
      <nav aria-label="主导航" className="ml-8 hidden h-full items-stretch gap-1 md:flex">
        {navItems.map(item => <NavLink key={item.to} to={item.to} className={({ isActive }) => `relative flex items-center gap-2 px-4 text-[13px] transition-colors ${isActive ? 'text-cyan-300' : 'text-slate-400 hover:text-slate-100'}`}>
          {({ isActive }) => <><span aria-hidden="true" className="text-sm">{item.icon}</span>{item.label}{isActive && <span className="absolute inset-x-4 bottom-0 h-0.5 rounded bg-cyan-300" />}</>}
        </NavLink>)}
      </nav>
      <span className="ml-auto hidden font-mono text-[10px] tracking-wider text-slate-500 lg:block">PREVIEW BUILD · 0.1</span>
      <button type="button" className="ml-auto inline-flex size-10 items-center justify-center rounded-lg border border-slate-700 text-slate-200 hover:border-cyan-500/50 md:hidden" aria-label={menuOpen ? '关闭主导航' : '打开主导航'} aria-expanded={menuOpen} aria-controls="mobile-main-navigation" onClick={() => setMenuOpen(open => !open)}>
        <span aria-hidden="true" className="text-xl">{menuOpen ? '×' : '☰'}</span>
      </button>
    </div>
    {menuOpen && <nav id="mobile-main-navigation" aria-label="主导航" className="grid gap-1 border-t border-slate-800 p-2 md:hidden">
      {navItems.map(item => <NavLink key={item.to} to={item.to} onClick={() => setMenuOpen(false)} className={({ isActive }) => `flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm ${isActive ? 'bg-cyan-400/10 text-cyan-200' : 'text-slate-300 hover:bg-slate-800'}`}><span aria-hidden="true">{item.icon}</span>{item.label}</NavLink>)}
    </nav>}
  </header>
}

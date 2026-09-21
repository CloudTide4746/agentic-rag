import { Suspense, lazy } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, NavLink, Route, Routes } from 'react-router-dom'

const Overview = lazy(() => import('../features/overview/Overview'))
const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000, retry: 1 } } })

export function App() {
  return <QueryClientProvider client={queryClient}><BrowserRouter>
    <div className="mx-auto max-w-5xl px-6 py-12">
      <header className="mb-12 flex items-center justify-between border-b border-slate-200 pb-6">
        <NavLink to="/" className="font-semibold text-slate-900">企业知识库 / Agentic RAG</NavLink>
        <span className="text-sm text-slate-500">开发基线 · 0.1</span>
      </header>
      <Suspense fallback={<p role="status">正在加载页面…</p>}><Routes>
        <Route path="/" element={<Overview />} />
        <Route path="*" element={<main><h1>页面不存在</h1><NavLink to="/">返回首页</NavLink></main>} />
      </Routes></Suspense>
    </div>
  </BrowserRouter></QueryClientProvider>
}

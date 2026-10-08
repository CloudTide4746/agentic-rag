import type { ReactNode } from 'react'
import { Outlet } from 'react-router'
import { TopNavBar } from './TopNavBar'

export interface AppLayoutProps { children?: ReactNode }

export function AppLayout({ children }: AppLayoutProps) {
  return <div className="app-shell min-h-screen overflow-x-hidden bg-[#080d16] text-slate-200">
    <TopNavBar />
    <div className="border-b border-cyan-950/70 bg-cyan-950/20 px-4 py-2 text-center text-xs text-cyan-200/80 sm:text-sm">
      <span className="font-semibold">布局预览</span><span className="mx-2 text-cyan-700">·</span>静态示例内容，不连接业务数据
    </div>
    <main className="min-h-[calc(100vh-7.5rem)]">{children ?? <Outlet />}</main>
  </div>
}

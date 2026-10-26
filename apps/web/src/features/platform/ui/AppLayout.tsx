import type { ReactNode } from 'react'
import { Link, NavLink, Outlet, useParams } from 'react-router'
import './minimal.css'

export interface AppLayoutProps { children?: ReactNode }

const navItems = [
  { label: '工作台', to: '/app', end: true },
  { label: '团队', to: '/app/teams', end: false },
]

function TeamContextNav() {
  const { teamId } = useParams()
  if (!teamId) return null
  return <nav aria-label="团队导航">
    <p className="muted">当前团队 {teamId}</p>
    <ul>
      <li><NavLink to={`/app/teams/${teamId}`} end>团队总览</NavLink></li>
      <li><NavLink to={`/app/teams/${teamId}/projects`}>项目</NavLink></li>
    </ul>
  </nav>
}

export function AppLayout({ children }: AppLayoutProps) {
  return <div className="page">
    <header className="topbar">
      <Link to="/app" className="brand">TeamFlow</Link>
      <nav aria-label="应用导航">
        <ul>
          {navItems.map(item => <li key={item.to}><NavLink to={item.to} end={item.end}>{item.label}</NavLink></li>)}
        </ul>
        <TeamContextNav />
      </nav>
      <span className="muted">占位页面，未接入数据</span>
    </header>
    <main>{children ?? <Outlet />}</main>
  </div>
}

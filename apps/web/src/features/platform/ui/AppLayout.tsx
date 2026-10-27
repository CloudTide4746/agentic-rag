import { useState, type ReactNode } from 'react'
import { Link, NavLink, Outlet, useParams } from 'react-router'
import './minimal.css'

export interface AppLayoutProps { children?: ReactNode }

function AppNavigation({ onNavigate }: { onNavigate: () => void }) {
  const { teamId, projectId } = useParams()
  const teamPath = teamId ? `/app/teams/${teamId}` : undefined
  const projectPath = teamId ? `/app/teams/${teamId}/projects` : undefined

  return <nav aria-label="应用导航" className="tf-navigation">
    <p className="tf-nav-label">工作空间</p>
    <NavLink to="/app" end onClick={onNavigate} className={({ isActive }) => `tf-nav-link${isActive ? ' is-active' : ''}`}>
      <span className="tf-nav-index" aria-hidden="true">01</span>工作台
    </NavLink>
    <NavLink to="/app/teams" onClick={onNavigate} className={({ isActive }) => `tf-nav-link${isActive ? ' is-active' : ''}`}>
      <span className="tf-nav-index" aria-hidden="true">02</span>团队
    </NavLink>
    {teamPath && projectPath ? <>
      <p className="tf-nav-label tf-nav-context">当前团队</p>
      <NavLink to={teamPath} end onClick={onNavigate} className={({ isActive }) => `tf-nav-link tf-nav-sub${isActive ? ' is-active' : ''}`}>
        团队概览
      </NavLink>
      <NavLink to={projectPath} onClick={onNavigate} className={({ isActive }) => `tf-nav-link tf-nav-sub${isActive ? ' is-active' : ''}`}>
        项目
      </NavLink>
      {projectId ? <NavLink to={`${projectPath}/${projectId}/tasks`} onClick={onNavigate} className={({ isActive }) => `tf-nav-link tf-nav-sub${isActive ? ' is-active' : ''}`}>
        项目任务
      </NavLink> : null}
    </> : null}
  </nav>
}

export function AppLayout({ children }: AppLayoutProps) {
  const [menuOpen, setMenuOpen] = useState(false)
  const closeMenu = () => setMenuOpen(false)

  return <div className="tf-app">
    <a className="tf-skip-link" href="#main-content">跳转到主要内容</a>
    <aside className="tf-sidebar">
      <Link to="/app" className="tf-brand" aria-label="TeamFlow 工作台" onClick={closeMenu}>
        <span className="tf-brand-mark" aria-hidden="true">T</span>
        <span><strong>TeamFlow</strong><small>协作工作空间</small></span>
      </Link>
      <AppNavigation onNavigate={closeMenu} />
      <div className="tf-sidebar-foot">
        <span className="tf-sidebar-dot" aria-hidden="true" />
        <span>协作平台</span>
        <span className="tf-sidebar-version">预览</span>
      </div>
    </aside>

    <div className="tf-main-column">
      <header className="tf-topbar">
        <button
          className="tf-menu-button"
          type="button"
          aria-label={menuOpen ? '关闭侧边导航' : '打开侧边导航'}
          aria-expanded={menuOpen}
          aria-controls="mobile-app-navigation"
          onClick={() => setMenuOpen(open => !open)}
        >
          <span aria-hidden="true">{menuOpen ? '×' : '☰'}</span>
        </button>
        <Link to="/app" className="tf-mobile-brand">TeamFlow</Link>
        <div className="tf-topbar-context">
          <span className="tf-topbar-kicker">TEAMFLOW</span>
          <span className="tf-topbar-divider" aria-hidden="true">/</span>
          <span>协作平台</span>
        </div>
        <span className="tf-preview-pill"><span aria-hidden="true" />静态界面预览</span>
      </header>

      {menuOpen ? <>
        <button className="tf-nav-backdrop" type="button" aria-label="关闭侧边导航" onClick={closeMenu} />
        <div id="mobile-app-navigation" className="tf-mobile-navigation">
          <AppNavigation onNavigate={closeMenu} />
        </div>
      </> : null}

      <div className="tf-preview-notice" role="note">
        <span className="tf-preview-notice-mark" aria-hidden="true">i</span>
        <span>静态界面预览，数据读取与提交操作尚未接入。</span>
      </div>
      <main id="main-content" className="tf-main-content">{children ?? <Outlet />}</main>
    </div>
  </div>
}

// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { appRoutes } from './router'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

it('renders the static layout at /chat', async () => {
  // ChatPage now loads sessions on mount; the other legacy pages must stay data-free.
  render(<RouterProvider router={createMemoryRouter(appRoutes, { initialEntries: ['/chat'] })} />)
  expect(await screen.findByRole('heading', { name: '最近会话' })).toBeTruthy()
  expect(screen.getByText('静态示例内容，不连接业务数据')).toBeTruthy()
})

it.each([
  ['/admin/kb', '知识库'],
  ['/admin/datasources', '数据源配置'],
  ['/admin/playground', '检索测试'],
  ['/admin/logs', '决策日志'],
])('renders the static layout at %s', async (path, heading) => {
  const fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
  const router = createMemoryRouter(appRoutes, { initialEntries: [path] })
  render(<RouterProvider router={router} />)

  expect(await screen.findByRole('heading', { name: heading })).toBeTruthy()
  expect(screen.getByText('静态示例内容，不连接业务数据')).toBeTruthy()
  expect(fetchMock).not.toHaveBeenCalled()
})

it('renders each TeamFlow placeholder route with the app navigation', async () => {
  const fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
  const routes = [
    ['/app', '工作台'],
    ['/app/teams', '团队列表'],
    ['/app/teams/t1', '团队详情'],
    ['/app/teams/t1/projects', '项目列表'],
    ['/app/teams/t1/projects/p1', '项目详情'],
    ['/app/teams/t1/projects/p1/tasks', '任务列表'],
    ['/app/teams/t1/projects/p1/tasks/k1', '任务详情'],
  ] as const
  for (const [path, heading] of routes) {
    render(<RouterProvider router={createMemoryRouter(appRoutes, { initialEntries: [path] })} />)
    expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeTruthy()
    expect(screen.getAllByRole('navigation', { name: '应用导航' }).length).toBeGreaterThan(0)
    cleanup()
  }
  expect(fetchMock).not.toHaveBeenCalled()
})

it('reaches the task detail through links from the platform dashboard', async () => {
  vi.stubGlobal('fetch', vi.fn())
  const router = createMemoryRouter(appRoutes, { initialEntries: ['/app'] })
  render(<RouterProvider router={router} />)

  const primaryNav = screen.getAllByRole('navigation', { name: '应用导航' })[0]
  if (!primaryNav) throw new Error('expected the desktop app navigation to render')
  fireEvent.click(within(primaryNav).getByRole('link', { name: '团队' }))
  expect(await screen.findByRole('heading', { level: 1, name: '团队列表' })).toBeTruthy()

  fireEvent.click(screen.getByRole('link', { name: '示例团队 →' }))
  expect(await screen.findByRole('heading', { level: 1, name: '团队详情' })).toBeTruthy()
  expect(screen.getByText('teamId: demo')).toBeTruthy()

  fireEvent.click(screen.getByRole('link', { name: '查看团队项目 →' }))
  fireEvent.click(await screen.findByRole('link', { name: '示例项目 →' }))
  expect(await screen.findByRole('heading', { level: 1, name: '项目详情' })).toBeTruthy()
  fireEvent.click(await screen.findByRole('link', { name: '查看任务列表 →' }))
  expect(await screen.findByRole('heading', { level: 1, name: '任务列表' })).toBeTruthy()

  fireEvent.click(screen.getByRole('link', { name: '示例任务 →' }))
  expect(await screen.findByRole('heading', { level: 1, name: '任务详情' })).toBeTruthy()
  expect(router.state.location.pathname).toBe('/app/teams/demo/projects/demo/tasks/1')
})

it('redirects the root route to chat and provides a recovery link for unknown paths', async () => {
  const rootRouter = createMemoryRouter(appRoutes, { initialEntries: ['/'] })
  render(<RouterProvider router={rootRouter} />)
  expect(await screen.findByRole('heading', { name: '最近会话' })).toBeTruthy()
  cleanup()

  const missingRouter = createMemoryRouter(appRoutes, { initialEntries: ['/missing'] })
  render(<RouterProvider router={missingRouter} />)
  expect(await screen.findByRole('heading', { name: '页面不存在' })).toBeTruthy()
  expect(screen.getByRole('link', { name: '返回智能问答' })).toBeTruthy()
})

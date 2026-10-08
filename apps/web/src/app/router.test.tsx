// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { appRoutes } from './router'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

it.each([
  ['/chat', '最近会话'],
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

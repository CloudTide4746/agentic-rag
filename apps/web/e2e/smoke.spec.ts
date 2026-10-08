import { test, expect } from '@playwright/test'

test('navigates all five static preview pages without calling APIs', async ({ page }) => {
  const apiRequests: string[] = []
  page.on('request', request => {
    if (request.url().includes('/api/')) apiRequests.push(request.url())
  })
  await page.goto('/')

  await expect(page).toHaveURL(/\/chat$/)
  await expect(page.getByRole('heading', { name: '最近会话' })).toBeVisible()
  await expect(page.getByText('静态示例内容，不连接业务数据')).toBeVisible()

  const pages = [
    ['知识库', '/admin/kb'],
    ['数据源', '/admin/datasources'],
    ['检索测试', '/admin/playground'],
    ['决策日志', '/admin/logs'],
    ['智能问答', '/chat'],
  ] as const
  for (const [label, path] of pages) {
    await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: label }).click()
    await expect(page).toHaveURL(new RegExp(`${path.replaceAll('/', '\\/')}$`))
    await expect(page.getByText('布局预览', { exact: true }).first()).toBeVisible()
  }
  expect(apiRequests).toEqual([])
})

test('collapses navigation and keeps the main content usable on a narrow screen', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await page.goto('/admin/kb')
  await expect(page.getByRole('heading', { name: '知识库' })).toBeVisible()

  const menuButton = page.getByRole('button', { name: '打开主导航' })
  await expect(menuButton).toBeVisible()
  await menuButton.click()
  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '检索测试' }).click()
  await expect(page.getByRole('heading', { name: '检索测试' })).toBeVisible()
  await expect(page.getByRole('button', { name: '打开主导航' })).toBeVisible()
})

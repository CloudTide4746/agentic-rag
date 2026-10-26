import { expect, test } from '@playwright/test'

test('navigates the TeamFlow routes from the platform dashboard to a task detail', async ({ page }) => {
  const apiRequests: string[] = []
  page.on('request', request => {
    if (request.url().includes('/api/')) apiRequests.push(request.url())
  })
  await page.goto('/app')
  await expect(page.getByRole('heading', { level: 1, name: '工作台' })).toBeVisible()

  await page.getByRole('navigation', { name: '应用导航' }).getByRole('link', { name: '团队' }).click()
  await page.getByRole('link', { name: '示例团队 →' }).click()
  await expect(page.getByRole('heading', { level: 1, name: '团队详情' })).toBeVisible()
  await page.getByRole('link', { name: '查看团队项目 →' }).click()
  await page.getByRole('link', { name: '示例项目 →' }).click()
  await page.getByRole('link', { name: '查看任务列表 →' }).click()
  await page.getByRole('link', { name: '示例任务 →' }).click()
  await expect(page.getByRole('heading', { level: 1, name: '任务详情' })).toBeVisible()
  await expect(page).toHaveURL(/\/app\/teams\/demo\/projects\/demo\/tasks\/1$/)
  expect(apiRequests).toEqual([])
})

test('keeps the app navigation reachable on a narrow screen', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await page.goto('/app/teams')
  const menuButton = page.getByRole('button', { name: '打开侧边导航' })
  await expect(menuButton).toBeVisible()
  await menuButton.click()
  await page.getByRole('navigation', { name: '应用导航' }).getByRole('link', { name: '工作台' }).click()
  await expect(page.getByRole('heading', { level: 1, name: '工作台' })).toBeVisible()
})

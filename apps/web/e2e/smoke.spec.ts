import { test, expect } from '@playwright/test'
test('shows the API connectivity result', async ({ page }) => {
  await page.route('**/api/v1/health/live', route => route.fulfill({ json: { data: { status: 'ok', service: 'api' }, request_id: 'smoke' } }))
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('企业内部知识库问答平台')
  await expect(page.getByRole('status')).toContainText('Go API 已连接')
})

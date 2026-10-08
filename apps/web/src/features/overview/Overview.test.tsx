// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Overview from './Overview'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })
it('makes a failed connection visible to the user', async () => {
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<QueryClientProvider client={client}><Overview /></QueryClientProvider>)
  expect((await screen.findByRole('alert')).textContent).toContain('连接失败')
  client.clear()
})

it('checks the running Agent API health endpoint', async () => {
  const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
    status: 'ok', version: '1.0.0', model: 'test-model', mysql: true, elasticsearch: true,
  })))
  vi.stubGlobal('fetch', fetchMock)
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<QueryClientProvider client={client}><Overview /></QueryClientProvider>)
  expect(await screen.findByText(/Agent API 已连接/)).toBeTruthy()
  expect(fetchMock).toHaveBeenCalledWith('/api/health', expect.anything())
  client.clear()
})

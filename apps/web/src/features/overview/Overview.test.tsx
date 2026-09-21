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

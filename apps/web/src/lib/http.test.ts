import { afterEach, expect, it, vi } from 'vitest'
import { get } from './http'

afterEach(() => vi.unstubAllGlobals())
it('unwraps the public API envelope and forwards cancellation', async () => {
  const signal = new AbortController().signal
  const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { status: 'ok' }, request_id: 'test' })))
  vi.stubGlobal('fetch', fetchMock)
  await expect(get('/health/live', signal)).resolves.toEqual({ status: 'ok' })
  expect(fetchMock).toHaveBeenCalledWith('/api/v1/health/live', expect.objectContaining({ signal }))
})
it('rejects failed HTTP requests', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 503 })))
  await expect(get('/health/ready')).rejects.toThrow('503')
})

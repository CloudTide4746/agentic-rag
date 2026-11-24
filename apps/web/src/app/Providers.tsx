import { useState, type ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from '../features/auth'
import { clearChatSession } from '../features/chat'

export function AppProviders({ children }: { children: ReactNode }) {
  const [client] = useState(() => new QueryClient({
    defaultOptions: { queries: { staleTime: 30_000, retry: false }, mutations: { retry: false } },
  }))
  return <QueryClientProvider client={client}>
    <AuthProvider onSessionEnd={clearChatSession}>{children}</AuthProvider>
  </QueryClientProvider>
}

import type { ReactNode } from 'react'

export interface LegacyPlaceholderProps {
  children?: ReactNode
  className?: string
}

/** Temporary visual shell used while legacy component behavior is migrated separately. */
export function LegacyPlaceholder({ children, className }: LegacyPlaceholderProps) {
  return <div className={className}>{children}</div>
}

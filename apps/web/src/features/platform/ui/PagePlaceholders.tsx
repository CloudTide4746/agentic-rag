import type { ReactNode } from 'react'
import { Link } from 'react-router'

export interface PagePlaceholderProps {
  eyebrow: string
  title: string
  description: string
  children?: ReactNode
}

/** Minimal shell for a TeamFlow page whose data layer has not been wired yet. */
export function PagePlaceholder({ eyebrow, title, description, children }: PagePlaceholderProps) {
  return <article>
    <p className="muted">{eyebrow}</p>
    <h1>{title}</h1>
    <p>{description}</p>
    {children}
    <p className="muted">静态占位页面，不连接业务数据。</p>
  </article>
}

export function PlaceholderCard({ title, items, children }: { title: string; items?: string[]; children?: ReactNode }) {
  return <section>
    <h2>{title}</h2>
    {items ? <ul>{items.map(item => <li key={item}>{item}</li>)}</ul> : null}
    {children}
  </section>
}

export function PlaceholderRows({ count = 3, label }: { count?: number; label: string }) {
  return <div className="skeleton" role="img" aria-label={label}>
    {Array.from({ length: count }, (_, index) => <span key={index} />)}
  </div>
}

export function PlaceholderBreadcrumbs({ items }: { items: { label: string; to?: string }[] }) {
  return <nav aria-label="面包屑">
    {items.map((item, index) => <span key={item.label}>{index > 0 ? ' / ' : ''}{item.to ? <Link to={item.to}>{item.label}</Link> : <span aria-current="page">{item.label}</span>}</span>)}
  </nav>
}

export function PlaceholderCta({ to, label }: { to: string; label: string }) {
  return <Link to={to}>{label}</Link>
}

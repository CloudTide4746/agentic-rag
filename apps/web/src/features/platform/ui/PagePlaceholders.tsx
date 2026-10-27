import type { ReactNode } from 'react'
import { Link } from 'react-router'

export interface BreadcrumbItem { label: string; to?: string }

export function PageHeader({
  eyebrow,
  title,
  description,
  breadcrumbs,
  action,
}: {
  eyebrow: string
  title: string
  description: string
  breadcrumbs?: BreadcrumbItem[]
  action?: ReactNode
}) {
  return <div className="tf-page-header">
    {breadcrumbs?.length ? <nav className="tf-breadcrumbs" aria-label="面包屑">
      {breadcrumbs.map((item, index) => <span className="tf-breadcrumb-item" key={`${item.label}-${index}`}>
        {index > 0 ? <span className="tf-breadcrumb-separator" aria-hidden="true">/</span> : null}
        {item.to ? <Link to={item.to}>{item.label}</Link> : <span aria-current="page">{item.label}</span>}
      </span>)}
    </nav> : null}
    <div className="tf-page-heading-row">
      <div className="tf-page-heading-copy">
        <p className="tf-eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        <p className="tf-page-description">{description}</p>
      </div>
      {action ? <div className="tf-page-action">{action}</div> : null}
    </div>
  </div>
}

export function SectionPanel({
  title,
  description,
  action,
  children,
  className = '',
}: {
  title: string
  description?: string
  action?: ReactNode
  children?: ReactNode
  className?: string
}) {
  return <section className={`tf-panel ${className}`}>
    <div className="tf-panel-heading">
      <div>
        <h2>{title}</h2>
        {description ? <p>{description}</p> : null}
      </div>
      {action ? <div className="tf-panel-action">{action}</div> : null}
    </div>
    {children}
  </section>
}

export function EmptyState({ title, description }: { title: string; description: string }) {
  return <div className="tf-empty-state">
    <span className="tf-empty-mark" aria-hidden="true">—</span>
    <h3>{title}</h3>
    <p>{description}</p>
  </div>
}

export function PreviewField({
  label,
  name,
  placeholder,
  type = 'text',
  maxLength,
  multiline = false,
  value,
  readOnly = true,
}: {
  label: string
  name: string
  placeholder?: string
  type?: 'text' | 'date' | 'datetime-local' | 'url'
  maxLength?: number
  multiline?: boolean
  value?: string
  readOnly?: boolean
}) {
  return <label className="tf-field" htmlFor={name}>
    <span>{label}</span>
    {multiline ? <textarea id={name} name={name} placeholder={placeholder} maxLength={maxLength} defaultValue={value} readOnly={readOnly} rows={4} /> :
      <input id={name} name={name} type={type} placeholder={placeholder} maxLength={maxLength} defaultValue={value} readOnly={readOnly} />}
  </label>
}

export function PreviewSelect({
  label,
  name,
  options,
  value,
}: {
  label: string
  name: string
  options: { label: string; value: string }[]
  value?: string
}) {
  return <label className="tf-field" htmlFor={name}>
    <span>{label}</span>
    <select id={name} name={name} defaultValue={value ?? options[0]?.value} disabled>
      {options.map(option => <option value={option.value} key={option.value}>{option.label}</option>)}
    </select>
  </label>
}

export function PreviewButton({ children, destructive = false }: { children: ReactNode; destructive?: boolean }) {
  return <button className={`tf-button${destructive ? ' tf-button-danger' : ''}`} type="button" disabled aria-disabled="true">
    {children}
  </button>
}

export function ReadonlyValue({ label, value }: { label: string; value: string }) {
  return <div className="tf-readonly-value">
    <dt>{label}</dt>
    <dd>{value || '—'}</dd>
  </div>
}

export function PageColumns({ children }: { children: ReactNode }) {
  return <div className="tf-page-columns">{children}</div>
}

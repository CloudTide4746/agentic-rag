import { useState, type FormEvent } from 'react'
import type { CreateProjectReq, Project, UpdateProjectReq } from '../../../lib/api/list'
import { ErrorMessage, FormField, dateInputValue, dateRequestValue } from '../../../shared/ui/Crud'

interface Props {
  project?: Project
  pending: boolean
  error: unknown
  success: boolean
  onSave: (body: CreateProjectReq & UpdateProjectReq) => void
}

export function ProjectForm({ project, pending, error, success, onSave }: Props) {
  const [validation, setValidation] = useState<Error | null>(null)
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const fields = new FormData(event.currentTarget)
    const name = String(fields.get('name') ?? '').trim()
    const description = String(fields.get('description') ?? '')
    const startInput = String(fields.get('start_date') ?? '')
    const endInput = String(fields.get('end_date') ?? '')
    if ([...name].length < 2 || [...name].length > 100 || [...description].length > 500) {
      setValidation(new Error('名称需为 2–100 个字符，描述最多 500 个字符。'))
      return
    }
    if ((project?.start_date && !startInput) || (project?.end_date && !endInput)) {
      setValidation(new Error('当前接口不支持清除已设置的日期，请保留或修改日期。'))
      return
    }
    if (startInput && endInput && startInput > endInput) {
      setValidation(new Error('开始日期不能晚于结束日期。'))
      return
    }
    const status = String(fields.get('status') ?? '')
    if (project && status !== 'active' && status !== 'archived') {
      setValidation(new Error('请选择有效的项目状态。'))
      return
    }
    setValidation(null)
    try {
      const start = project && startInput === dateInputValue(project.start_date) ? undefined : dateRequestValue(startInput)
      const end = project && endInput === dateInputValue(project.end_date) ? undefined : dateRequestValue(endInput)
      onSave({ name, description, ...(start ? { start_date: start } : {}), ...(end ? { end_date: end } : {}), ...(project ? { status } : {}) })
    } catch {
      setValidation(new Error('请输入有效日期。'))
    }
  }
  return <form onSubmit={submit} aria-label={project ? '编辑项目' : '创建项目'}>
    <fieldset disabled={pending} className="tf-form-stack">
      <legend className="tf-sr-only">项目信息</legend>
      <FormField label="项目名称" name="name" defaultValue={project?.name} required maxLength={100} />
      <FormField label="项目描述" name="description" defaultValue={project?.description} maxLength={500} multiline />
      {project && <label className="tf-field" htmlFor="status">项目状态<select id="status" name="status" defaultValue={project.status}><option value="active">进行中</option><option value="archived">已归档</option></select></label>}
      <FormField label="开始日期" name="start_date" type="datetime-local" defaultValue={dateInputValue(project?.start_date)} />
      <FormField label="结束日期" name="end_date" type="datetime-local" defaultValue={dateInputValue(project?.end_date)} />
      {project && <p className="tf-inline-note">已设置的日期可以修改，当前接口暂不支持清除日期。</p>}
      <ErrorMessage error={validation ?? error} />
      {success && !validation && <p role="status">已保存项目。</p>}
      <div className="tf-form-actions"><button className="tf-button tf-button-primary" type="submit">{pending ? '正在保存…' : project ? '保存修改' : '创建项目'}</button></div>
    </fieldset>
  </form>
}

import type { Task } from '../../../lib/api/list'
import { FormField, dateInputValue, dateRequestValue } from '../../../shared/ui/Crud'

const priorities = [
  { value: 'medium', label: '普通' },
  { value: 'low', label: '低' },
  { value: 'high', label: '高' },
  { value: 'none', label: '无' },
]

export function TaskFields({ task }: { task?: Task }) {
  return <div className="tf-form-stack">
    <FormField label="任务标题" name="title" defaultValue={task?.title} required maxLength={200} />
    <FormField label="任务描述" name="description" defaultValue={task?.description} maxLength={10000} multiline />
    <label className="tf-field" htmlFor="priority"><span>优先级</span>
      <select id="priority" name="priority" defaultValue={task?.priority ?? 'medium'}>
        {priorities.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>
    </label>
    <FormField label="截止时间" name="due_date" type="datetime-local" defaultValue={dateInputValue(task?.due_date)} />
    {task?.due_date ? <p className="tf-inline-note">当前接口支持修改截止时间，暂不支持清除已设置的时间。</p> : null}
  </div>
}

export function readTaskFields(form: HTMLFormElement, task?: Task) {
  const values = new FormData(form)
  const title = String(values.get('title') ?? '').trim()
  const description = String(values.get('description') ?? '')
  const priority = String(values.get('priority') ?? '')
  const dueInput = String(values.get('due_date') ?? '')
  if (!title || [...title].length > 200 || [...description].length > 10000) {
    throw new Error('请填写任务标题（最多 200 字），描述最多 10000 字')
  }
  if (!priorities.some(option => option.value === priority)) throw new Error('请选择有效的优先级')
  if (task?.due_date && !dueInput) throw new Error('当前接口暂不支持清除截止时间，请保留或修改时间')
  const due_date = task?.due_date && dueInput === dateInputValue(task.due_date)
    ? task.due_date : dateRequestValue(dueInput)
  return { title, description, priority, due_date }
}

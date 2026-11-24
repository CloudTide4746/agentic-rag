import { useState, type FormEvent } from 'react'
import type { Team, UpdateTeamReq } from '../../../lib/api/list'
import { ErrorMessage, FormField } from '../../../shared/ui/Crud'

interface Props {
  team?: Team
  pending: boolean
  error: unknown
  success: boolean
  onSave: (body: UpdateTeamReq & { name: string; description: string }) => void
}

export function TeamForm({ team, pending, error, success, onSave }: Props) {
  const [validation, setValidation] = useState<Error | null>(null)
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const fields = new FormData(event.currentTarget)
    const name = String(fields.get('name') ?? '').trim()
    const description = String(fields.get('description') ?? '')
    const avatar = String(fields.get('avatar') ?? '').trim()
    if ([...name].length < 2 || [...name].length > 100 || [...description].length > 500 || [...avatar].length > 500) {
      setValidation(new Error('名称需为 2–100 个字符，描述和头像地址最多 500 个字符。'))
      return
    }
    setValidation(null)
    onSave({ name, description, ...(team ? { avatar } : {}) })
  }
  return <form onSubmit={submit} aria-label={team ? '编辑团队' : '创建团队'}>
    <fieldset disabled={pending} className="tf-form-stack">
      <legend className="tf-sr-only">团队信息</legend>
      <FormField label="团队名称" name="name" defaultValue={team?.name} required maxLength={100} />
      <FormField label="团队描述" name="description" defaultValue={team?.description} maxLength={500} multiline />
      {team && <FormField label="头像地址" name="avatar" defaultValue={team.avatar} type="url" maxLength={500} />}
      <ErrorMessage error={validation ?? error} />
      {success && !validation && <p role="status">已保存团队。</p>}
      <div className="tf-form-actions"><button className="tf-button tf-button-primary" type="submit">{pending ? '正在保存…' : team ? '保存修改' : '创建团队'}</button></div>
    </fieldset>
  </form>
}

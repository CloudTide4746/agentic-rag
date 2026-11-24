import { useState, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router'
import { PageColumns, PageHeader, ReadonlyValue, SectionPanel } from '../../platform/index'
import { useAuth } from '../../auth/index'
import { ErrorMessage, RequestState, parseResourceId } from '../../../shared/ui/Crud'
import type { Task } from '../../../lib/api/list'
import { isTaskConflict, useTask, useTaskMutations, useTaskProject } from '../api/tasks'
import { readTaskFields, TaskFields } from '../ui/TaskFields'

export default function TaskDetailPage() {
  const params = useParams()
  const teamId = parseResourceId(params.teamId)
  const projectId = parseResourceId(params.projectId)
  const taskId = parseResourceId(params.taskId)
  const { user } = useAuth()
  const [editorKey, setEditorKey] = useState(0)
  const project = useTaskProject(user?.id, teamId, projectId)
  const task = useTask(user?.id, teamId, projectId, taskId, project.isSuccess)
  if (!teamId || !projectId || !taskId) return <p role="alert">团队、项目或任务地址无效。</p>
  return <div className="tf-page">
    <PageHeader eyebrow="项目空间 / 任务" title="任务详情" description="查看任务状态，并编辑任务字段。"
      breadcrumbs={[
        { label: '团队', to: '/app/teams' },
        { label: '项目', to: `/app/teams/${teamId}/projects/${projectId}` },
        { label: '任务', to: `/app/teams/${teamId}/projects/${projectId}/tasks` },
        { label: '任务详情' },
      ]} />
    <RequestState pending={project.isPending} error={project.error} retry={() => void project.refetch()} />
    {project.isSuccess ? <>
      <RequestState pending={task.isPending} error={task.error} retry={() => void task.refetch()} />
      {task.data ? <TaskEditor key={`${user?.id}-${task.data.id}-${editorKey}`} userId={user?.id} teamId={teamId} projectId={projectId} taskId={taskId} task={task.data} reload={() => {
        void task.refetch().then(result => {
          if (result.isSuccess) setEditorKey(key => key + 1)
        })
      }} reloading={task.isFetching} /> : null}
    </> : null}
  </div>
}

function TaskEditor({ userId, teamId, projectId, taskId, task, reload, reloading }: {
  userId: string | undefined; teamId: number; projectId: number; taskId: number; task: Task; reload: () => void; reloading: boolean
}) {
  const navigate = useNavigate()
  const { update, remove } = useTaskMutations(userId, teamId, projectId, taskId)
  const [error, setError] = useState<Error | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const busy = update.isPending || remove.isPending || reloading
  const currentError = error ?? update.error ?? remove.error
  const conflict = isTaskConflict(currentError)
  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy || conflict) return
    setError(null)
    update.reset()
    remove.reset()
    try {
      update.mutate({ ...readTaskFields(event.currentTarget, task), version: task.version })
    } catch (failure) {
      setError(failure instanceof Error ? failure : new Error('保存失败'))
    }
  }
  function deleteConfirmed() {
    if (busy) return
    setError(null)
    update.reset()
    remove.mutate(undefined, { onSuccess: () => navigate(`/app/teams/${teamId}/projects/${projectId}/tasks`) })
  }
  return <PageColumns>
    <div className="tf-page-column-main">
      <SectionPanel title={task.title}>
        <dl className="tf-value-grid">
          <ReadonlyValue label="任务 ID" value={task.id} />
          <ReadonlyValue label="项目 ID" value={task.project_id} />
          <ReadonlyValue label="状态" value={task.status} />
          <ReadonlyValue label="负责人 ID" value={task.assignee_id ?? '未分配'} />
          <ReadonlyValue label="版本" value={String(task.version)} />
          <ReadonlyValue label="截止时间" value={task.due_date ? new Date(task.due_date).toLocaleString() : ''} />
        </dl>
        <p className="tf-inline-note">状态与负责人只读；当前任务接口不支持状态流转和分配操作。</p>
      </SectionPanel>
      <SectionPanel title="更新任务" className="tf-form-panel">
        <form onSubmit={save} aria-describedby="task-error">
          <fieldset disabled={busy}>
            <TaskFields task={task} />
            <div className="tf-form-actions"><button className="tf-button tf-button-primary" type="submit" disabled={conflict}>{update.isPending ? '正在保存…' : '保存修改'}</button></div>
          </fieldset>
        </form>
        <div id="task-error"><ErrorMessage error={currentError} /></div>
        {update.isSuccess ? <p role="status">任务已保存。</p> : null}
        {conflict ? <div role="status">
          <p>任务已被其他人修改，当前输入已保留。请重新加载最新版本后再编辑。</p>
          <p>重新加载将放弃当前未保存的输入。</p>
          <button className="tf-button" type="button" disabled={busy} onClick={reload}>{reloading ? '正在重新加载…' : '重新加载最新版本'}</button>
        </div> : null}
      </SectionPanel>
    </div>
    <div className="tf-page-column-side">
      <SectionPanel title="任务操作">
        <p className="tf-panel-copy">删除任务后无法从此列表恢复。</p>
        {confirmDelete ? <div role="group" aria-label="确认删除任务">
          <p>确认删除“{task.title}”？</p>
          <button className="tf-button tf-button-danger" type="button" disabled={busy} onClick={deleteConfirmed}>{remove.isPending ? '正在删除…' : '确认删除'}</button>
          <button className="tf-button" type="button" disabled={busy} onClick={() => setConfirmDelete(false)}>取消</button>
        </div> : <button className="tf-button tf-button-danger" type="button" disabled={busy} onClick={() => setConfirmDelete(true)}>删除任务</button>}
      </SectionPanel>
    </div>
  </PageColumns>
}

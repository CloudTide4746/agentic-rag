import { useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router'
import { EmptyState, PageHeader, SectionPanel } from '../../platform/index'
import { useAuth } from '../../auth/index'
import { ErrorMessage, Pagination, RequestState, parseResourceId, usePage } from '../../../shared/ui/Crud'
import { taskPageSize, useTaskMutations, useTaskProject, useTasks } from '../api/tasks'
import { readTaskFields, TaskFields } from '../ui/TaskFields'

export default function TaskListPage() {
  const params = useParams()
  const teamId = parseResourceId(params.teamId)
  const projectId = parseResourceId(params.projectId)
  const { user } = useAuth()
  const [page, setPage] = usePage()
  const project = useTaskProject(user?.id, teamId, projectId)
  const tasks = useTasks(user?.id, teamId, projectId, page, project.isSuccess)
  if (!teamId || !projectId) return <p role="alert">团队或项目地址无效。</p>
  return <div className="tf-page">
    <PageHeader eyebrow="项目空间 / 任务" title="任务列表" description="查看当前项目中的任务、优先级与截止时间。"
      breadcrumbs={[
        { label: '团队', to: '/app/teams' },
        { label: '项目', to: `/app/teams/${teamId}/projects/${projectId}` },
        { label: '任务' },
      ]} />
    <RequestState pending={project.isPending} error={project.error} retry={() => void project.refetch()} />
    {project.isSuccess ? <>
      <SectionPanel title={`${project.data.name}的任务`}>
        <RequestState pending={tasks.isPending} error={tasks.error} retry={() => void tasks.refetch()} />
        {tasks.isSuccess ? <>
          {tasks.data.items.length ? <div className="tf-table-wrap"><table className="tf-table tf-task-table">
            <thead><tr><th scope="col">任务</th><th scope="col">状态</th><th scope="col">优先级</th><th scope="col">负责人 ID</th><th scope="col">截止时间</th></tr></thead>
            <tbody>{tasks.data.items.map(task => <tr key={task.id}>
              <td><Link to={`/app/teams/${teamId}/projects/${projectId}/tasks/${task.id}`}>{task.title}</Link></td>
              <td>{task.status}</td><td>{task.priority}</td><td>{task.assignee_id ?? '未分配'}</td><td>{task.due_date ? new Date(task.due_date).toLocaleString() : '—'}</td>
            </tr>)}</tbody>
          </table></div> : <EmptyState title="暂无任务" description="在下方为当前项目创建任务，或切换分页查看。" />}
          <Pagination page={page} total={tasks.data.total} pageSize={taskPageSize} onPage={setPage} />
        </> : null}
      </SectionPanel>
      <CreateTaskForm key={`${user?.id}-${projectId}`} userId={user?.id} teamId={teamId} projectId={projectId} onCreated={() => setPage(1)} />
    </> : null}
  </div>
}

function CreateTaskForm({ userId, teamId, projectId, onCreated }: {
  userId: string | undefined; teamId: number; projectId: number; onCreated: () => void
}) {
  const { create } = useTaskMutations(userId, teamId, projectId)
  const [formError, setFormError] = useState<Error | null>(null)
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (create.isPending) return
    const form = event.currentTarget
    setFormError(null)
    create.reset()
    try {
      create.mutate(readTaskFields(form), { onSuccess: () => {
        form.reset()
        onCreated()
      } })
    } catch (error) {
      setFormError(error instanceof Error ? error : new Error('创建任务失败'))
    }
  }
  return <SectionPanel title="创建任务" className="tf-form-panel">
    <form onSubmit={submit} aria-describedby="create-task-error">
      <fieldset disabled={create.isPending}>
        <TaskFields />
        <div id="create-task-error"><ErrorMessage error={formError ?? create.error} /></div>
        <div className="tf-form-actions"><button className="tf-button tf-button-primary" type="submit">{create.isPending ? '正在创建…' : '创建任务'}</button></div>
      </fieldset>
      {create.isSuccess && !formError ? <p role="status">任务已创建。</p> : null}
    </form>
  </SectionPanel>
}

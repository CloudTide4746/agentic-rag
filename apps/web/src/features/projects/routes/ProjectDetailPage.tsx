import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { useAuth } from '../../auth'
import { PageHeader, ReadonlyValue, SectionPanel } from '../../platform'
import { ErrorMessage, RequestState, parseResourceId } from '../../../shared/ui/Crud'
import { useDeleteProject, useProject, useUpdateProject } from '../api/data'
import { ProjectForm } from '../ui/ProjectForm'
import type { Project } from '../../../lib/api/list'

function ProjectActions({ project, teamId, id, userId }: { project: Project; teamId: number; id: number; userId: string }) {
  const update = useUpdateProject(userId, teamId, id)
  const remove = useDeleteProject(userId, teamId, id)
  const [confirm, setConfirm] = useState(false)
  const navigate = useNavigate()
  if (project.owner_id !== userId) return <p className="tf-inline-note">仅项目负责人可以编辑或删除项目。</p>
  return <>
    <SectionPanel title="编辑项目" className="tf-form-panel">
      <ProjectForm key={project.updated_at} project={project} pending={update.isPending || remove.isPending} error={update.error} success={update.isSuccess} onSave={body => update.mutate(body)} />
    </SectionPanel>
    <SectionPanel title="项目设置">
      <ErrorMessage error={remove.error} />
      {confirm ? <div role="group" aria-label="确认删除项目">
        <p>确定删除“{project.name}”吗？此操作无法撤销，删除后将无法通过此项目访问其任务。</p>
        <button type="button" className="tf-button tf-button-danger" disabled={remove.isPending || update.isPending} onClick={() => remove.mutate(undefined, { onSuccess: () => navigate(`/app/teams/${teamId}/projects`) })}>{remove.isPending ? '正在删除…' : '确认删除项目'}</button>
        <button type="button" className="tf-button" disabled={remove.isPending} onClick={() => setConfirm(false)}>取消</button>
      </div> : <button className="tf-button tf-button-danger" type="button" disabled={update.isPending} onClick={() => setConfirm(true)}>删除项目</button>}
    </SectionPanel>
  </>
}

export default function ProjectDetailPage() {
  const params = useParams()
  const teamId = parseResourceId(params.teamId)
  const id = parseResourceId(params.projectId)
  const { user } = useAuth()
  const query = useProject(user?.id ?? '', teamId, id)
  return <div className="tf-page">
    <PageHeader eyebrow="项目空间" title={query.data?.name ?? '项目详情'} description="查看项目状态、计划周期与任务。" breadcrumbs={[{ label: '团队', to: '/app/teams' }, { label: '项目', ...(teamId !== null ? { to: `/app/teams/${teamId}/projects` } : {}) }, { label: '项目概览' }]} />
    {id === null || teamId === null ? <p role="alert">团队或项目编号无效。</p> : <>
      <RequestState pending={query.isPending} error={query.error} retry={() => { void query.refetch() }} />
      {query.data && <>
        <SectionPanel title="项目详情">
          <dl className="tf-value-grid">
            <ReadonlyValue label="项目名称" value={query.data.name} />
            <ReadonlyValue label="项目描述" value={query.data.description || '无'} />
            <ReadonlyValue label="项目状态" value={query.data.status === 'archived' ? '已归档' : '进行中'} />
            <ReadonlyValue label="开始日期" value={query.data.start_date || '未设置'} />
            <ReadonlyValue label="结束日期" value={query.data.end_date || '未设置'} />
            <ReadonlyValue label="负责人 ID" value={query.data.owner_id} />
            <ReadonlyValue label="团队 ID" value={query.data.team_id} />
          </dl>
          <Link className="tf-button tf-button-primary" to={`/app/teams/${teamId}/projects/${id}/tasks`}>查看任务</Link>
        </SectionPanel>
        {user && <ProjectActions project={query.data} teamId={teamId} id={id} userId={user.id} />}
      </>}
    </>}
  </div>
}

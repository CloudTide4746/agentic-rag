import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { useAuth } from '../../auth'
import { EmptyState, PageHeader, ReadonlyValue, SectionPanel } from '../../platform'
import { ErrorMessage, RequestState, parseResourceId } from '../../../shared/ui/Crud'
import { useDeleteTeam, useTeam, useUpdateTeam } from '../api/data'
import { TeamForm } from '../ui/TeamForm'
import type { Team } from '../../../lib/api/list'

function TeamActions({ team, id, userId }: { team: Team; id: number; userId: string }) {
  const update = useUpdateTeam(userId, id)
  const remove = useDeleteTeam(userId, id)
  const [confirm, setConfirm] = useState(false)
  const navigate = useNavigate()
  if (team.owner_id !== userId) return <p className="tf-inline-note">仅团队负责人可以编辑或删除团队。</p>
  return <>
    <SectionPanel title="编辑团队" className="tf-form-panel">
      <TeamForm key={team.updated_at} team={team} pending={update.isPending || remove.isPending} error={update.error} success={update.isSuccess} onSave={body => update.mutate(body)} />
    </SectionPanel>
    <SectionPanel title="团队设置">
      <ErrorMessage error={remove.error} />
      {confirm ? <div role="group" aria-label="确认删除团队">
        <p>确定删除“{team.name}”吗？删除后将无法通过此团队访问其项目。此操作无法撤销。</p>
        <button type="button" className="tf-button tf-button-danger" disabled={remove.isPending || update.isPending} onClick={() => remove.mutate(undefined, { onSuccess: () => navigate('/app/teams') })}>{remove.isPending ? '正在删除…' : '确认删除团队'}</button>
        <button type="button" className="tf-button" disabled={remove.isPending} onClick={() => setConfirm(false)}>取消</button>
      </div> : <button className="tf-button tf-button-danger" type="button" disabled={update.isPending} onClick={() => setConfirm(true)}>删除团队</button>}
    </SectionPanel>
  </>
}

export default function TeamDetailPage() {
  const { teamId } = useParams()
  const id = parseResourceId(teamId)
  const { user } = useAuth()
  const query = useTeam(user?.id ?? '', id)
  return <div className="tf-page">
    <PageHeader eyebrow="团队空间" title={query.data?.name ?? '团队详情'} description="查看团队信息并管理团队项目。" breadcrumbs={[{ label: '团队', to: '/app/teams' }, { label: '团队概览' }]} />
    {id === null ? <p role="alert">团队编号无效。</p> : <>
      <RequestState pending={query.isPending} error={query.error} retry={() => { void query.refetch() }} />
      {query.data && <>
        <SectionPanel title="团队信息">
          <dl className="tf-value-grid">
            <ReadonlyValue label="团队名称" value={query.data.name} />
            <ReadonlyValue label="团队描述" value={query.data.description || '无'} />
            <ReadonlyValue label="负责人 ID" value={query.data.owner_id} />
            <ReadonlyValue label="团队 ID" value={query.data.id} />
            <ReadonlyValue label="头像地址" value={query.data.avatar || '未设置'} />
          </dl>
          <Link className="tf-button tf-button-primary" to={`/app/teams/${id}/projects`}>查看项目</Link>
        </SectionPanel>
        <SectionPanel title="团队成员"><EmptyState title="成员管理暂未开放" description="当前公开 API 尚未提供成员查询和管理端点。" /></SectionPanel>
        {user && <TeamActions team={query.data} id={id} userId={user.id} />}
      </>}
    </>}
  </div>
}

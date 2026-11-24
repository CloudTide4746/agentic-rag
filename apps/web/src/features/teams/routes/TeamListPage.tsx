import { Link, useNavigate } from 'react-router'
import { useAuth } from '../../auth'
import { EmptyState, PageHeader, SectionPanel } from '../../platform'
import { Pagination, RequestState, usePage } from '../../../shared/ui/Crud'
import { TEAM_PAGE_SIZE, useCreateTeam, useTeams } from '../api/data'
import { TeamForm } from '../ui/TeamForm'

export default function TeamListPage() {
  const { user } = useAuth()
  const [page, setPage] = usePage()
  const navigate = useNavigate()
  const query = useTeams(user?.id ?? '', page)
  const create = useCreateTeam(user?.id ?? '')
  return <div className="tf-page">
    <PageHeader eyebrow="工作空间 / 团队" title="团队列表" description="查看和管理你加入的团队。" />
    <SectionPanel title="团队列表">
      <RequestState pending={query.isPending} error={query.error} retry={() => { void query.refetch() }} />
      {query.data && <>
        {query.data.items.length ? <div className="tf-table-wrap"><table className="tf-table">
          <thead><tr><th scope="col">团队</th><th scope="col">负责人 ID</th><th scope="col">操作</th></tr></thead>
          <tbody>{query.data.items.map(team => <tr key={team.id}><td>{team.name}</td><td>{team.owner_id}</td><td><Link to={`/app/teams/${team.id}`}>查看团队</Link></td></tr>)}</tbody>
        </table></div> : <EmptyState title="暂无团队" description="创建一个团队，开始管理项目。" />}
        <Pagination page={page} total={query.data.total} pageSize={TEAM_PAGE_SIZE} onPage={setPage} />
      </>}
    </SectionPanel>
    <SectionPanel title="创建团队" className="tf-form-panel">
      <TeamForm pending={create.isPending} error={create.error} success={create.isSuccess} onSave={body => {
        create.mutate(body, { onSuccess: team => navigate(`/app/teams/${team.id}`) })
      }} />
    </SectionPanel>
  </div>
}

import { Link, useNavigate, useParams } from 'react-router'
import { useAuth } from '../../auth'
import { EmptyState, PageHeader, SectionPanel } from '../../platform'
import { Pagination, RequestState, parseResourceId, usePage } from '../../../shared/ui/Crud'
import { PROJECT_PAGE_SIZE, useCreateProject, useProjects } from '../api/data'
import { ProjectForm } from '../ui/ProjectForm'

function CreateProject({ userId, teamId }: { userId: string; teamId: number }) {
  const create = useCreateProject(userId, teamId)
  const navigate = useNavigate()
  return <SectionPanel title="创建项目" className="tf-form-panel">
    <ProjectForm pending={create.isPending} error={create.error} success={create.isSuccess} onSave={body => {
      create.mutate(body, { onSuccess: project => navigate(`/app/teams/${teamId}/projects/${project.id}`) })
    }} />
  </SectionPanel>
}

export default function ProjectListPage() {
  const { teamId } = useParams()
  const id = parseResourceId(teamId)
  const { user } = useAuth()
  const [page, setPage] = usePage()
  const query = useProjects(user?.id ?? '', id, page)
  return <div className="tf-page">
    <PageHeader eyebrow="团队空间 / 项目" title="项目列表" description="查看团队项目，创建新的协作计划。" breadcrumbs={[{ label: '团队', to: '/app/teams' }, { label: '团队概览', ...(id !== null ? { to: `/app/teams/${id}` } : {}) }, { label: '项目' }]} />
    {id === null ? <p role="alert">团队编号无效。</p> : <>
      <SectionPanel title="项目列表">
        <RequestState pending={query.isPending} error={query.error} retry={() => { void query.refetch() }} />
        {query.data && <>
          {query.data.items.length ? <div className="tf-table-wrap"><table className="tf-table">
            <thead><tr><th scope="col">项目</th><th scope="col">状态</th><th scope="col">负责人 ID</th><th scope="col">操作</th></tr></thead>
            <tbody>{query.data.items.map(project => <tr key={project.id}><td>{project.name}</td><td>{project.status === 'archived' ? '已归档' : '进行中'}</td><td>{project.owner_id}</td><td><Link to={`/app/teams/${id}/projects/${project.id}`}>查看项目</Link></td></tr>)}</tbody>
          </table></div> : <EmptyState title="暂无项目" description="为这个团队创建第一个项目。" />}
          <Pagination page={page} total={query.data.total} pageSize={PROJECT_PAGE_SIZE} onPage={setPage} />
        </>}
      </SectionPanel>
      {user && query.data && <CreateProject userId={user.id} teamId={id} />}
    </>}
  </div>
}

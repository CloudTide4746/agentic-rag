import { useParams } from 'react-router'
import { PagePlaceholder, PlaceholderBreadcrumbs, PlaceholderCard, PlaceholderCta, PlaceholderRows } from '../../platform/index'

export default function TeamDetailPage() {
  const { teamId } = useParams()
  return <PagePlaceholder eyebrow={`TEAM · ${teamId ?? ''}`} title="团队详情" description="团队信息、成员列表、成员角色与团队设置。">
    <div>
      <PlaceholderBreadcrumbs items={[{ label: '团队', to: '/app/teams' }, { label: teamId ?? '团队' }]} />
      <PlaceholderCard title="团队信息">
        <PlaceholderRows count={2} label="团队信息占位" />
        <p className="muted mono">teamId: {teamId}</p>
      </PlaceholderCard>
      <PlaceholderCard title="成员列表">
        <PlaceholderRows label="成员与角色占位" />
        <p className="muted">Owner / Admin / Member 角色标签占位。</p>
      </PlaceholderCard>
    </div>
    <div>
      <PlaceholderCard title="团队设置" items={['重命名团队占位', '邀请成员占位', '退出团队占位']} />
      <PlaceholderCard title="下一步">
        <PlaceholderCta to={`/app/teams/${teamId ?? 'demo'}/projects`} label="查看团队项目 →" />
      </PlaceholderCard>
    </div>
  </PagePlaceholder>
}

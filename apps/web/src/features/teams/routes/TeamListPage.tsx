import { PagePlaceholder, PlaceholderCard, PlaceholderCta, PlaceholderRows } from '../../platform/index'

export default function TeamListPage() {
  return <PagePlaceholder eyebrow="TEAM · /app/teams" title="团队列表" description="浏览团队、创建团队并选择团队进入。">
    <PlaceholderCard title="全部团队">
      <PlaceholderRows label="团队列表占位" />
      <p className="muted">团队名称、成员数量与角色占位。</p>
    </PlaceholderCard>
    <div>
      <PlaceholderCard title="创建团队" items={['团队名称输入占位', '创建错误与重复提交占位']} />
      <PlaceholderCard title="进入示例团队">
        <PlaceholderCta to="/app/teams/demo" label="示例团队 →" />
      </PlaceholderCard>
    </div>
  </PagePlaceholder>
}

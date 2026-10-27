import { Link } from 'react-router'
import { PageHeader, SectionPanel } from '../index'

export default function PlatformDashboardPage() {
  return <div className="tf-page">
    <PageHeader
      eyebrow="工作空间"
      title="工作台"
      description="从团队空间进入项目，继续协作。"
      action={<Link className="tf-button tf-button-primary" to="/app/teams">查看团队</Link>}
    />

    <SectionPanel title="团队空间" description="团队、项目与任务都从团队空间进入。">
      <div className="tf-dashboard-start">
        <div className="tf-dashboard-number" aria-hidden="true">01</div>
        <div className="tf-dashboard-copy">
          <h3>从团队开始</h3>
          <p>选择一个团队，查看它的项目与协作任务。</p>
        </div>
        <Link to="/app/teams" className="tf-text-link">打开团队列表 <span aria-hidden="true">→</span></Link>
      </div>
    </SectionPanel>
  </div>
}

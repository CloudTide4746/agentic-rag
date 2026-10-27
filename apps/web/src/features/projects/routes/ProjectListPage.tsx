import { useParams } from 'react-router'
import { EmptyState, PageHeader, PreviewButton, PreviewField, SectionPanel } from '../../platform/index'

export default function ProjectListPage() {
  const { teamId } = useParams()
  const id = teamId ?? ''

  return <div className="tf-page">
    <PageHeader
      eyebrow="团队空间 / 项目"
      title="项目列表"
      description="查看团队项目与计划周期。"
      breadcrumbs={[{ label: '团队', to: '/app/teams' }, { label: '团队概览', to: `/app/teams/${id}` }, { label: '项目' }]}
      action={<PreviewButton>新建项目</PreviewButton>}
    />

    <SectionPanel title="项目列表" description="列表字段对应项目名称、状态、起止日期与负责人。">
      <div className="tf-table-wrap">
        <table className="tf-table">
          <thead><tr><th scope="col">项目</th><th scope="col">状态</th><th scope="col">开始日期</th><th scope="col">结束日期</th><th scope="col"><span className="tf-sr-only">操作</span></th></tr></thead>
          <tbody><tr><td colSpan={5}>
            <EmptyState title="项目数据将在此显示" description="接入团队项目列表后，可查看团队下的项目。" />
          </td></tr></tbody>
        </table>
      </div>
    </SectionPanel>

    <SectionPanel title="创建项目" description="填写项目名称、描述和可选的计划周期。" className="tf-form-panel">
      <div className="tf-form-grid">
        <PreviewField label="项目名称" name="project-name" placeholder="输入项目名称" maxLength={100} />
        <PreviewField label="项目描述" name="project-description" placeholder="说明项目目标与范围" maxLength={500} multiline />
        <PreviewField label="开始日期" name="project-start-date" type="datetime-local" />
        <PreviewField label="结束日期" name="project-end-date" type="datetime-local" />
      </div>
      <div className="tf-form-actions"><PreviewButton>创建项目</PreviewButton></div>
    </SectionPanel>
  </div>
}

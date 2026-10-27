import { Link, useParams } from 'react-router'
import {
  PageColumns,
  PageHeader,
  PreviewButton,
  PreviewField,
  PreviewSelect,
  ReadonlyValue,
  SectionPanel,
} from '../../platform/index'

const projectStatuses = [
  { value: 'active', label: '进行中' },
  { value: 'archived', label: '已归档' },
]

export default function ProjectDetailPage() {
  const { teamId, projectId } = useParams()
  const team = teamId ?? ''
  const project = projectId ?? ''

  return <div className="tf-page">
    <PageHeader
      eyebrow="项目空间"
      title="项目详情"
      description="查看项目基本信息、状态与计划周期。"
      breadcrumbs={[
        { label: '团队', to: '/app/teams' },
        { label: '团队概览', to: `/app/teams/${team}` },
        { label: '项目', to: `/app/teams/${team}/projects` },
        { label: '项目概览' },
      ]}
      action={<Link className="tf-button tf-button-primary" to={`/app/teams/${team}/projects/${project}/tasks`}>查看任务</Link>}
    />

    <PageColumns>
      <SectionPanel title="项目详情" description="项目数据接入后显示实际名称、描述和状态。">
        <dl className="tf-value-grid">
          <ReadonlyValue label="项目名称" value="" />
          <ReadonlyValue label="项目状态" value="" />
          <ReadonlyValue label="开始日期" value="" />
          <ReadonlyValue label="结束日期" value="" />
          <ReadonlyValue label="团队 ID" value={team} />
          <ReadonlyValue label="项目 ID" value={project} />
        </dl>
        <p className="tf-inline-note">项目详情接入后显示。</p>
      </SectionPanel>

      <div className="tf-page-column-side">
        <SectionPanel title="编辑项目" description="名称、描述、状态和日期字段对应项目更新接口。" className="tf-form-panel">
          <div className="tf-form-stack">
            <PreviewField label="项目名称" name="project-name" placeholder="项目名称" maxLength={100} />
            <PreviewField label="项目描述" name="project-description" placeholder="项目描述" maxLength={500} multiline />
            <PreviewSelect label="项目状态" name="project-status" options={projectStatuses} />
            <PreviewField label="开始日期" name="project-start-date" type="datetime-local" />
            <PreviewField label="结束日期" name="project-end-date" type="datetime-local" />
          </div>
          <div className="tf-form-actions"><PreviewButton>保存修改</PreviewButton></div>
        </SectionPanel>
        <SectionPanel title="项目设置">
          <p className="tf-panel-copy">项目归档和删除可在接入相应操作后启用。</p>
          <div className="tf-form-actions"><PreviewButton destructive>删除项目</PreviewButton></div>
        </SectionPanel>
      </div>
    </PageColumns>
  </div>
}

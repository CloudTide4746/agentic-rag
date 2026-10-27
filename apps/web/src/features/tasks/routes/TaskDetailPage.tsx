import { useParams } from 'react-router'
import {
  PageColumns,
  PageHeader,
  PreviewButton,
  PreviewField,
  PreviewSelect,
  ReadonlyValue,
  SectionPanel,
} from '../../platform/index'

const priorities = [
  { value: 'medium', label: '普通' },
  { value: 'low', label: '低' },
  { value: 'high', label: '高' },
  { value: 'none', label: '无' },
]

export default function TaskDetailPage() {
  const { teamId, projectId, taskId } = useParams()
  const team = teamId ?? ''
  const project = projectId ?? ''
  const task = taskId ?? ''

  return <div className="tf-page">
    <PageHeader
      eyebrow="项目空间 / 任务"
      title="任务详情"
      description="查看任务状态，并编辑可更新的任务字段。"
      breadcrumbs={[
        { label: '团队', to: '/app/teams' },
        { label: '团队概览', to: `/app/teams/${team}` },
        { label: '项目', to: `/app/teams/${team}/projects` },
        { label: '项目概览', to: `/app/teams/${team}/projects/${project}` },
        { label: '任务', to: `/app/teams/${team}/projects/${project}/tasks` },
        { label: '任务详情' },
      ]}
    />

    <PageColumns>
      <div className="tf-page-column-main">
        <SectionPanel title="任务信息" description="任务详情接入后显示实际内容。">
          <dl className="tf-value-grid">
            <ReadonlyValue label="任务 ID" value={task} />
            <ReadonlyValue label="项目 ID" value={project} />
            <ReadonlyValue label="状态" value="" />
            <ReadonlyValue label="负责人 ID" value="" />
            <ReadonlyValue label="版本" value="" />
            <ReadonlyValue label="截止时间" value="" />
          </dl>
          <p className="tf-inline-note">状态只读显示；当前任务更新接口不包含状态流转字段。</p>
        </SectionPanel>

        <SectionPanel title="更新任务" description="提交时需携带当前版本号；冲突时应重新读取最新版本。" className="tf-form-panel">
          <div className="tf-form-stack">
            <PreviewField label="任务标题" name="task-title" placeholder="任务标题" maxLength={200} />
            <PreviewField label="任务描述" name="task-description" placeholder="任务描述" maxLength={10000} multiline />
            <PreviewSelect label="优先级" name="task-priority" options={priorities} />
            <PreviewField label="截止时间" name="task-due-date" type="datetime-local" />
            <PreviewField label="版本号" name="task-version" placeholder="详情接入后显示" readOnly />
          </div>
          <div className="tf-form-actions"><PreviewButton>保存修改</PreviewButton></div>
        </SectionPanel>
      </div>

      <div className="tf-page-column-side">
        <SectionPanel title="任务操作">
          <p className="tf-panel-copy">删除任务后无法从此列表恢复。</p>
          <div className="tf-form-actions"><PreviewButton destructive>删除任务</PreviewButton></div>
        </SectionPanel>
      </div>
    </PageColumns>
  </div>
}

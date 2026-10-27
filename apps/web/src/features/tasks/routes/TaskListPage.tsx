import { useParams } from 'react-router'
import { EmptyState, PageHeader, PreviewButton, PreviewField, PreviewSelect, SectionPanel } from '../../platform/index'

const priorities = [
  { value: 'medium', label: '普通' },
  { value: 'low', label: '低' },
  { value: 'high', label: '高' },
  { value: 'none', label: '无' },
]

export default function TaskListPage() {
  const { teamId, projectId } = useParams()
  const team = teamId ?? ''
  const project = projectId ?? ''

  return <div className="tf-page">
    <PageHeader
      eyebrow="项目空间 / 任务"
      title="任务列表"
      description="查看当前项目中的任务、优先级与截止时间。"
      breadcrumbs={[
        { label: '团队', to: '/app/teams' },
        { label: '团队概览', to: `/app/teams/${team}` },
        { label: '项目', to: `/app/teams/${team}/projects` },
        { label: '项目概览', to: `/app/teams/${team}/projects/${project}` },
        { label: '任务' },
      ]}
      action={<PreviewButton>新建任务</PreviewButton>}
    />

    <SectionPanel title="任务列表" description="列表列对应任务标题、状态、优先级、负责人和截止时间。">
      <div className="tf-table-wrap">
        <table className="tf-table tf-task-table">
          <thead><tr><th scope="col">任务</th><th scope="col">状态</th><th scope="col">优先级</th><th scope="col">负责人 ID</th><th scope="col">截止时间</th><th scope="col"><span className="tf-sr-only">操作</span></th></tr></thead>
          <tbody><tr><td colSpan={6}>
            <EmptyState title="任务数据将在此显示" description="任务列表接入后，可按项目分页查看任务。" />
          </td></tr></tbody>
        </table>
      </div>
    </SectionPanel>

    <SectionPanel title="创建任务" description="新任务归属当前项目，可设置描述、优先级与截止时间。" className="tf-form-panel">
      <div className="tf-form-grid">
        <PreviewField label="任务标题" name="task-title" placeholder="输入任务标题" maxLength={200} />
        <PreviewSelect label="优先级" name="task-priority" options={priorities} />
        <PreviewField label="任务描述" name="task-description" placeholder="补充任务说明" maxLength={10000} multiline />
        <PreviewField label="截止时间" name="task-due-date" type="datetime-local" />
      </div>
      <p className="tf-inline-note">项目 ID：{project}</p>
      <div className="tf-form-actions"><PreviewButton>创建任务</PreviewButton></div>
    </SectionPanel>
  </div>
}

import { PageHeader, PreviewButton, PreviewField, SectionPanel, EmptyState } from '../../platform/index'

export default function TeamListPage() {
  return <div className="tf-page">
    <PageHeader
      eyebrow="工作空间 / 团队"
      title="团队列表"
      description="管理团队空间，并在团队之间切换。"
      action={<PreviewButton>新建团队</PreviewButton>}
    />

    <SectionPanel title="团队列表" description="团队信息与负责人来自团队接口。">
      <div className="tf-table-wrap">
        <table className="tf-table">
          <thead><tr><th scope="col">团队</th><th scope="col">负责人</th><th scope="col">最近更新</th><th scope="col"><span className="tf-sr-only">操作</span></th></tr></thead>
          <tbody><tr><td colSpan={4}>
            <EmptyState title="团队数据将在此显示" description="连接团队列表接口后，可在这里查看并进入团队。" />
          </td></tr></tbody>
        </table>
      </div>
    </SectionPanel>

    <SectionPanel title="创建团队" description="对应创建团队接口的名称与描述字段。" className="tf-form-panel">
      <div className="tf-form-grid">
        <PreviewField label="团队名称" name="team-name" placeholder="输入团队名称" maxLength={100} />
        <PreviewField label="团队描述" name="team-description" placeholder="介绍这个团队的协作范围" maxLength={500} multiline />
      </div>
      <div className="tf-form-actions"><PreviewButton>创建团队</PreviewButton></div>
    </SectionPanel>
  </div>
}

import { Link, useParams } from 'react-router'
import {
  EmptyState,
  PageColumns,
  PageHeader,
  PreviewButton,
  PreviewField,
  ReadonlyValue,
  SectionPanel,
} from '../../platform/index'

export default function TeamDetailPage() {
  const { teamId } = useParams()
  const id = teamId ?? ''

  return <div className="tf-page">
    <PageHeader
      eyebrow="团队空间"
      title="团队详情"
      description="查看团队信息并进入团队项目。"
      breadcrumbs={[{ label: '团队', to: '/app/teams' }, { label: '团队概览' }]}
      action={<Link className="tf-button tf-button-primary" to={`/app/teams/${id}/projects`}>查看项目</Link>}
    />

    <PageColumns>
      <div className="tf-page-column-main">
        <SectionPanel title="团队信息" description="名称、描述、负责人和头像与团队详情接口字段一致。">
          <dl className="tf-value-grid">
            <ReadonlyValue label="团队名称" value="" />
            <ReadonlyValue label="负责人 ID" value="" />
            <ReadonlyValue label="团队描述" value="" />
            <ReadonlyValue label="团队 ID" value={id} />
          </dl>
          <p className="tf-inline-note">团队详细信息接入后显示。</p>
        </SectionPanel>
        <SectionPanel title="团队成员" description="当前 TeamFlow API 暂无成员查询与管理接口。">
          <EmptyState title="成员管理暂未开放" description="当前公开接口仅支持团队信息与团队 CRUD。" />
        </SectionPanel>
      </div>

      <div className="tf-page-column-side">
        <SectionPanel title="编辑团队" description="修改团队名称、描述或头像地址。" className="tf-form-panel">
          <div className="tf-form-stack">
            <PreviewField label="团队名称" name="team-name" placeholder="团队名称" maxLength={100} />
            <PreviewField label="团队描述" name="team-description" placeholder="团队描述" maxLength={500} multiline />
            <PreviewField label="头像地址" name="team-avatar" placeholder="https://example.com/avatar.png" type="url" maxLength={500} />
          </div>
          <div className="tf-form-actions"><PreviewButton>保存修改</PreviewButton></div>
        </SectionPanel>
        <SectionPanel title="团队设置">
          <p className="tf-panel-copy">删除团队操作对应公开 API 的团队删除端点。</p>
          <div className="tf-form-actions"><PreviewButton destructive>删除团队</PreviewButton></div>
        </SectionPanel>
      </div>
    </PageColumns>
  </div>
}

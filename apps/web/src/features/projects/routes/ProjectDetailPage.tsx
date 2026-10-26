/** @format */

import { useParams } from "react-router";
import {
  PagePlaceholder,
  PlaceholderBreadcrumbs,
  PlaceholderCard,
  PlaceholderCta,
  PlaceholderRows,
} from "../../platform/index";

export default function ProjectDetailPage() {
  const { teamId, projectId } = useParams();
  const base = `/app/teams/${teamId ?? "demo"}/projects/${projectId ?? "demo"}`;
  return (
    <PagePlaceholder
      eyebrow={`PROJECT · ${projectId ?? ""}`}
      title='项目详情'
      description='项目概览、项目成员与任务入口。'
    >
      <div >
        <PlaceholderBreadcrumbs
          items={[
            { label: "团队", to: "/app/teams" },
            { label: teamId ?? "团队", to: `/app/teams/${teamId}` },
            { label: "项目", to: `/app/teams/${teamId}/projects` },
            { label: projectId ?? "项目" },
          ]}
        />
        <PlaceholderCard title='项目概览'>
          <PlaceholderRows count={2} label='项目概览占位' />
          <p >
            projectId: {projectId}
          </p>
        </PlaceholderCard>
        <PlaceholderCard title='项目成员'>
          <PlaceholderRows label='项目成员占位' />
        </PlaceholderCard>
      </div>
      <div >
        <PlaceholderCard title='任务入口'>
          <div >
            <PlaceholderCta to={`${base}/tasks`} label='查看任务列表 →' />
          </div>
        </PlaceholderCard>
        <PlaceholderCard
          title='项目设置'
          items={["编辑项目占位", "归档项目占位", "删除项目占位"]}
        />
      </div>
    </PagePlaceholder>
  );
}

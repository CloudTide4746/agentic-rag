/** @format */

import { useParams } from "react-router";
import {
  PagePlaceholder,
  PlaceholderBreadcrumbs,
  PlaceholderCard,
  PlaceholderCta,
  PlaceholderRows,
} from "../../platform/index";

export default function TaskListPage() {
  const { teamId, projectId } = useParams();
  const base = `/app/teams/${teamId ?? "demo"}/projects/${projectId ?? "demo"}`;
  return (
    <PagePlaceholder
      eyebrow={`TASK · ${projectId ?? ""}`}
      title='任务列表'
      description='分页、状态与负责人筛选、创建任务与批量浏览。分页与筛选状态将放入 URL。'
    >
      <div >
        <PlaceholderBreadcrumbs
          items={[
            { label: "团队", to: `/app/teams/${teamId}` },
            { label: "项目", to: `/app/teams/${teamId}/projects` },
            { label: projectId ?? "项目", to: base },
            { label: "任务" },
          ]}
        />
        <PlaceholderCard title='筛选项'>
          <PlaceholderRows count={1} label='状态与负责人筛选占位' />
          <p >
            状态：待处理 / 进行中 / 已完成；负责人筛选占位。
          </p>
        </PlaceholderCard>
        <PlaceholderCard title='任务'>
          <PlaceholderRows count={5} label='任务分页列表占位' />
          <p >
            分页控件与空状态占位。
          </p>
        </PlaceholderCard>
      </div>
      <div >
        <PlaceholderCard
          title='创建任务'
          items={["标题、负责人与截止日期占位", "创建失败提示占位"]}
        />
        <PlaceholderCard title='打开示例任务'>
          <div >
            <PlaceholderCta to={`${base}/tasks/1`} label='示例任务 →' />
          </div>
        </PlaceholderCard>
      </div>
    </PagePlaceholder>
  );
}

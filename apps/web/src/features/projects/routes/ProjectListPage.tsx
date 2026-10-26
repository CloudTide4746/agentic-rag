/** @format */

import { useParams } from "react-router";
import {
  PagePlaceholder,
  PlaceholderBreadcrumbs,
  PlaceholderCard,
  PlaceholderCta,
  PlaceholderRows,
} from "../../platform/index";

export default function ProjectListPage() {
  const { teamId } = useParams();
  return (
    <PagePlaceholder
      eyebrow={`PROJECT · ${teamId ?? ""}`}
      title='项目列表'
      description='团队下的项目列表、创建项目与归档入口。'
    >
      <div >
        <PlaceholderBreadcrumbs
          items={[
            { label: "团队", to: "/app/teams" },
            { label: teamId ?? "团队", to: `/app/teams/${teamId}` },
            { label: "项目" },
          ]}
        />
        <PlaceholderCard title='全部项目'>
          <PlaceholderRows label='项目列表占位' />
          <p >
            项目名称、状态、任务数与更新时间占位。
          </p>
        </PlaceholderCard>
      </div>
      <div >
        <PlaceholderCard
          title='创建项目'
          items={["项目名称与描述占位", "创建失败提示占位"]}
        />
        <PlaceholderCard title='进入示例项目'>
          <div >
            <PlaceholderCta
              to={`/app/teams/${teamId ?? "demo"}/projects/demo`}
              label='示例项目 →'
            />
          </div>
        </PlaceholderCard>
      </div>
    </PagePlaceholder>
  );
}

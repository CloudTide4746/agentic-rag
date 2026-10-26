/** @format */

import { useParams } from "react-router";
import {
  PagePlaceholder,
  PlaceholderBreadcrumbs,
  PlaceholderCard,
  PlaceholderRows,
} from "../../platform/index";

export default function TaskDetailPage() {
  const { teamId, projectId, taskId } = useParams();
  const tasksPath = `/app/teams/${teamId}/projects/${projectId}/tasks`;
  return (
    <PagePlaceholder
      eyebrow={`TASK · ${taskId ?? ""}`}
      title='任务详情'
      description='编辑任务、状态流转、负责人与截止日期；version 冲突时提示刷新并重新加载。'
    >
      <div >
        <PlaceholderBreadcrumbs
          items={[
            { label: "团队", to: `/app/teams/${teamId}` },
            { label: "项目", to: `/app/teams/${teamId}/projects` },
            { label: "任务", to: tasksPath },
            { label: taskId ?? "任务" },
          ]}
        />
        <PlaceholderCard title='任务内容'>
          <PlaceholderRows count={3} label='任务标题、描述与字段占位' />
          <p >
            taskId: {taskId}
          </p>
        </PlaceholderCard>
        <PlaceholderCard title='冲突恢复'>
          <PlaceholderRows count={1} label='409 版本冲突提示占位' />
          <p >
            冲突时保留本地草稿并提示重新加载服务端最新版本，不允许静默覆盖。
          </p>
        </PlaceholderCard>
      </div>
      <div >
        <PlaceholderCard
          title='属性'
          items={[
            "状态流转占位",
            "负责人选择占位",
            "截止日期占位",
            "版本号占位",
          ]}
        />
        <PlaceholderCard title='操作' items={["保存占位", "删除任务占位"]} />
      </div>
    </PagePlaceholder>
  );
}

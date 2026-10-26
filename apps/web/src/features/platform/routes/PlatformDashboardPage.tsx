/** @format */

import {
  PagePlaceholder,
  PlaceholderCard,
  PlaceholderCta,
  PlaceholderRows,
} from "../../platform/index";

export default function PlatformDashboardPage() {
  return (
    <PagePlaceholder
      eyebrow='PLATFORM · /app'
      title='工作台'
      description='当前用户、团队概览与最近项目。统一承载加载、错误与空状态。'
    >
        <PlaceholderCard title='当前用户'>
          <PlaceholderRows count={1} label='当前用户信息占位' />
          <p >
            姓名、邮箱与角色将来自当前用户接口。
          </p>
        </PlaceholderCard>
        <PlaceholderCard
          title='最近项目'
          items={[
            "最近访问的项目列表占位",
            "按团队分组展示占位",
            "无项目时展示空状态占位",
          ]}
        />
        <PlaceholderCard
          title='我的团队'
          items={["团队数量统计占位", "团队成员数占位", "创建团队入口占位"]}
        >
          <div >
            <PlaceholderCta to='/app/teams' label='进入团队列表 →' />
          </div>
        </PlaceholderCard>
        <PlaceholderCard
          title='统一错误与空状态'
          items={["403 无权限占位", "404 资源不存在占位", "网络失败与重试占位"]}
        />
    </PagePlaceholder>
  );
}

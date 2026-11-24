/** @format */

import { lazy, Suspense, type ReactNode } from "react";
import {
  createBrowserRouter,
  Navigate,
  NavLink,
  Outlet,
  useLocation,
  type RouteObject,
} from "react-router";
import { AppLayout as AdminLayout } from "../features/layout/index";
import { AppLayout, type AppLayoutProps } from "../features/platform/index";
import { LoginPage, RequireAuth } from "../features/auth";
import { AppProviders } from './Providers';

const ChatPage = lazy(() => import("../features/chat/routes/ChatPage"));
const KnowledgePage = lazy(
  () => import("../features/knowledge/routes/KnowledgePage"),
);
const DatasourcePage = lazy(
  () => import("../features/admin/routes/DatasourcePage"),
);
const PlaygroundPage = lazy(
  () => import("../features/admin/routes/PlaygroundPage"),
);
const DecisionLogPage = lazy(
  () => import("../features/logs/routes/DecisionLogPage"),
);
import { TeamDetailPage } from "../features/teams";
import { TeamListPage } from "../features/teams";
import { ProjectDetailPage } from "../features/projects";
import { ProjectListPage } from "../features/projects";
import { TaskDetailPage } from "../features/tasks";
import { TaskListPage } from "../features/tasks";
import { motion } from "motion/react";

const PlatformDashboardPage = lazy(
  () => import("../features/platform/routes/PlatformDashboardPage"),
);
function RouteContent() {
  return (
    <Suspense
      fallback={
        <p role='status' className='p-6 text-sm text-slate-400'>
          正在加载页面…
        </p>
      }
    >
      <Outlet />
    </Suspense>
  );
}

function NotFound() {
  return (
    <main className='mx-auto flex min-h-[60vh] max-w-xl flex-col items-center justify-center px-6 text-center'>
      <p className='font-mono text-xs tracking-[0.24em] text-cyan-400'>
        404 · NOT FOUND
      </p>
      <h1 className='mt-3 text-2xl font-semibold text-slate-100'>页面不存在</h1>
      <p className='mt-2 text-sm text-slate-400'>这个地址没有对应的页面。</p>
      <NavLink
        className='mt-6 rounded-lg border border-cyan-500/40 bg-cyan-400/10 px-4 py-2 text-sm text-cyan-200 hover:bg-cyan-400/20'
        to='/chat'
      >
        返回智能问答
      </NavLink>
    </main>
  );
}

function AdminShell({ children }: AppLayoutProps) {
  return <AdminLayout>{children}</AdminLayout>;
}

function AppShell({ children }: AppLayoutProps) {
  return <RequireAuth><AppLayout>{children}</AppLayout></RequireAuth>;
}

/**
 * 页面进入时的统一淡入壳子，只做一层包裹，不改变页面本身结构。
 *
 * key 绑定当前 pathname：路由切换时 React 会复用同位置、同类型的 Page 实例，
 * 不加 key 就不会重新挂载，motion 的 initial → animate 只在首次挂载时执行一次。
 */
function Page({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();
  return (
    <motion.div
      key={pathname}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.18, ease: "easeOut" }}
    >
      {children}
    </motion.div>
  );
}

const routes: RouteObject[] = [
  { path: 'login', element: <LoginPage /> },
  {
    element: (
      <AdminShell>
        <RouteContent />
      </AdminShell>
    ),
    children: [
      { index: true, element: <Navigate to='/app' replace /> },
      {
        path: "chat",
        element: (
          <Page>
            <RequireAuth><ChatPage /></RequireAuth>
          </Page>
        ),
      },
      {
        path: "admin/kb",
        element: (
          <Page>
            <KnowledgePage />
          </Page>
        ),
      },
      {
        path: "admin/datasources",
        element: (
          <Page>
            <DatasourcePage />
          </Page>
        ),
      },
      {
        path: "admin/playground",
        element: (
          <Page>
            <PlaygroundPage />
          </Page>
        ),
      },
      {
        path: "admin/logs",
        element: (
          <Page>
            <DecisionLogPage />
          </Page>
        ),
      },
    ],
  },

  {
    element: (
      <AppShell>
        <RouteContent />
      </AppShell>
    ),
    path: "app",
    children: [
      {
        index: true,
        element: (
          <Page>
            <PlatformDashboardPage />
          </Page>
        ),
      },
      {
        path: "teams",
        element: (
          <Page>
            <TeamListPage />
          </Page>
        ),
      },
      {
        path: "teams/:teamId",
        element: (
          <Page>
            <TeamDetailPage />
          </Page>
        ),
      },
      {
        path: "teams/:teamId/projects",
        element: (
          <Page>
            <ProjectListPage />
          </Page>
        ),
      },
      {
        path: "teams/:teamId/projects/:projectId",
        element: (
          <Page>
            <ProjectDetailPage />
          </Page>
        ),
      },
      {
        path: "teams/:teamId/projects/:projectId/tasks",
        element: (
          <Page>
            <TaskListPage />
          </Page>
        ),
      },
      {
        path: "teams/:teamId/projects/:projectId/tasks/:taskId",
        element: (
          <Page>
            <TaskDetailPage />
          </Page>
        ),
      },
    ],
  },
  {
    path: "*",
    element: (
      <Page>
        <NotFound />
      </Page>
    ),
  },
];

export const appRoutes: RouteObject[] = [{
  element: <AppProviders><Outlet /></AppProviders>,
  errorElement: <main className="p-6" role="alert">
    <h1>页面暂时无法加载</h1>
    <p>请重新打开页面；如果问题持续，请检查服务连接。</p>
    <a href="/app">重新打开工作台</a>
  </main>,
  children: routes,
}];
export const router = createBrowserRouter(appRoutes);

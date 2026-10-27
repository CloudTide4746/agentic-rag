/** @format */

import { lazy, Suspense } from "react";
import {
  createBrowserRouter,
  Navigate,
  NavLink,
  Outlet,
  type RouteObject,
} from "react-router";
import { AppLayout as AdminLayout } from "../features/layout/index";
import { AppLayout, type AppLayoutProps } from "../features/platform/index";
import { teamflowRoutes } from "./teamflowRoutes";

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

function RouteContent() {
  return (
    <Suspense
      fallback={
        <p role="status" className="p-6 text-sm text-slate-400">
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
    <main className="mx-auto flex min-h-[60vh] max-w-xl flex-col items-center justify-center px-6 text-center">
      <p className="font-mono text-xs tracking-[0.24em] text-cyan-400">
        404 · NOT FOUND
      </p>
      <h1 className="mt-3 text-2xl font-semibold text-slate-100">页面不存在</h1>
      <p className="mt-2 text-sm text-slate-400">这个地址没有对应的页面。</p>
      <NavLink
        className="mt-6 rounded-lg border border-cyan-500/40 bg-cyan-400/10 px-4 py-2 text-sm text-cyan-200 hover:bg-cyan-400/20"
        to="/chat"
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
  // TeamFlow app shell; a protected-route guard is added together with the auth contract.
  return <AppLayout>{children}</AppLayout>;
}

export const appRoutes: RouteObject[] = [
  {
    element: (
      <AdminShell>
        <RouteContent />
      </AdminShell>
    ),
    children: [
      { index: true, element: <Navigate to="/app" replace /> },
      { path: "chat", element: <ChatPage /> },
      { path: "admin/kb", element: <KnowledgePage /> },
      { path: "admin/datasources", element: <DatasourcePage /> },
      { path: "admin/playground", element: <PlaygroundPage /> },
      { path: "admin/logs", element: <DecisionLogPage /> },
    ],
  },
  {
    element: (
      <AppShell>
        <RouteContent />
      </AppShell>
    ),
    children: teamflowRoutes,
  },
  { path: "*", element: <NotFound /> },
];

export const router = createBrowserRouter(appRoutes);

/** @format */

import { lazy } from "react";
import type { RouteObject } from "react-router";
import { TeamDetailPage } from "../features/teams";
import { TeamListPage } from "../features/teams";
import { ProjectDetailPage } from "../features/projects";
import { ProjectListPage } from "../features/projects";
import { TaskDetailPage } from "../features/tasks";
import { TaskListPage } from "../features/tasks";

const PlatformDashboardPage = lazy(() => import("../features/platform/routes/PlatformDashboardPage"));

/** TeamFlow subtree under the platform app shell. Provider-free so tests can mount it directly. */
export const teamflowRoutes: RouteObject[] = [
  {
    path: "app",
    children: [
      { index: true, element: <PlatformDashboardPage /> },
      { path: "teams", element: <TeamListPage /> },
      { path: "teams/:teamId", element: <TeamDetailPage /> },
      { path: "teams/:teamId/projects", element: <ProjectListPage /> },
      { path: "teams/:teamId/projects/:projectId", element: <ProjectDetailPage /> },
      { path: "teams/:teamId/projects/:projectId/tasks", element: <TaskListPage /> },
      { path: "teams/:teamId/projects/:projectId/tasks/:taskId", element: <TaskDetailPage /> },
    ],
  },
];

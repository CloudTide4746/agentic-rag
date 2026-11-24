import type { ReactNode } from "react";
import { Navigate, Outlet, useLocation } from "react-router";
import { useAuth } from "../model/AuthProvider";

export function RequireAuth({ children }: { children?: ReactNode }): ReactNode {
  const { user } = useAuth();
  const location = useLocation();
  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search + location.hash }} />;
  }
  return children ?? <Outlet />;
}

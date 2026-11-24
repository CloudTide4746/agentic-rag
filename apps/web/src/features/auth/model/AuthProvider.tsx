import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { clearAccessToken, login as loginRequest, subscribeAuthenticationExpired, type LoginReq, type User } from "../../../lib/api/list";

interface AuthContextValue {
  user: User | null;
  login: (credentials: LoginReq, signal?: AbortSignal) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children, onSessionEnd }: {
  children: ReactNode;
  onSessionEnd?: () => void;
}): ReactNode {
  const queryClient = useQueryClient();
  const [user, setUser] = useState<User | null>(null);
  const operation = useRef(0);

  const logout = useCallback(async (): Promise<void> => {
    operation.current += 1;
    clearAccessToken();
    setUser(null);
    const cancelled = queryClient.cancelQueries();
    queryClient.clear();
    onSessionEnd?.();
    await cancelled;
  }, [queryClient, onSessionEnd]);

  const login = useCallback(async (credentials: LoginReq, signal?: AbortSignal): Promise<void> => {
    const cancelled = logout();
    const currentOperation = operation.current;
    await cancelled;
    signal?.throwIfAborted();
    if (operation.current !== currentOperation) {
      throw new DOMException("登录已取消", "AbortError");
    }
    try {
      const response = await loginRequest(credentials, signal);
      signal?.throwIfAborted();
      if (operation.current !== currentOperation) {
        throw new DOMException("登录已取消", "AbortError");
      }
      setUser(response.data.user);
    } catch (cause) {
      if (operation.current === currentOperation) clearAccessToken();
      throw cause;
    }
  }, [logout]);

  useEffect(() => subscribeAuthenticationExpired(() => { void logout(); }), [logout]);

  useEffect(() => () => {
    operation.current += 1;
    clearAccessToken();
    void queryClient.cancelQueries();
    queryClient.clear();
  }, [queryClient]);

  return <AuthContext.Provider value={{ user, login, logout }}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth 必须在 AuthProvider 内使用");
  return context;
}

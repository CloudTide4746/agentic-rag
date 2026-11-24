import { useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { Navigate, useLocation, useNavigate } from "react-router";
import { ApiError, register } from "../../../lib/api/list";
import { useAuth } from "../model/AuthProvider";

function returnPath(state: unknown): string {
  if (typeof state !== "object" || state === null || !("from" in state)) return "/app";
  const from = state.from;
  if (typeof from !== "string" || !from.startsWith("/") || from.startsWith("//") ||
    /[\\\u0000-\u001f]/.test(from) || from.split(/[?#]/)[0].replace(/\/+$/, "") === "/login") return "/app";
  return from;
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    const message = error.code === "UNAUTHENTICATED" ? "用户名或密码错误，请重新输入。" : error.message;
    return error.requestId ? `${message}（请求编号：${error.requestId}）` : message;
  }
  return "请求未完成，请检查网络后手动重试。";
}

export function LoginPage(): ReactNode {
  const { user, login } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const destination = returnPath(location.state);
  const [mode, setMode] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const request = useRef<AbortController | null>(null);
  const errorRef = useRef<HTMLParagraphElement>(null);
  const usernameRef = useRef<HTMLInputElement>(null);

  useEffect(() => () => request.current?.abort(), []);
  useEffect(() => { if (error) errorRef.current?.focus(); }, [error]);
  useEffect(() => { if (notice && !pending) usernameRef.current?.focus(); }, [notice, pending]);

  async function submit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    if (request.current) return;
    const controller = new AbortController();
    request.current = controller;
    setPending(true);
    setError("");
    setNotice("");
    try {
      if (mode === "register") {
        if (new TextEncoder().encode(password).length > 72) {
          setError("密码的 UTF-8 长度不能超过 72 字节。");
          return;
        }
        await register({ username: username.trim(), email: email.trim(), password }, controller.signal);
        controller.signal.throwIfAborted();
        setMode("login");
        setPassword("");
        setNotice("注册成功，请使用刚注册的用户名和密码登录。");
      } else {
        await login({ username: username.trim(), password }, controller.signal);
        controller.signal.throwIfAborted();
        void navigate(destination, { replace: true });
      }
    } catch (cause) {
      if (!controller.signal.aborted && !(cause instanceof DOMException && cause.name === "AbortError")) {
        setError(errorMessage(cause));
      }
    } finally {
      if (request.current === controller) request.current = null;
      if (!controller.signal.aborted) setPending(false);
    }
  }

  if (user) return <Navigate to={destination} replace />;

  return <main className="flex min-h-screen items-center justify-center bg-[var(--tf-canvas)] p-6 text-[var(--tf-text)]">
    <section className="w-full max-w-md rounded-xl border border-[var(--tf-line)] bg-[var(--tf-surface)] p-6 [color-scheme:light]">
      <h1 className="text-2xl font-semibold">{mode === "login" ? "登录 TeamFlow" : "注册账号"}</h1>
      <p className="mt-2 text-sm">登录后管理团队、项目、任务并使用问答。</p>
      <form onSubmit={(event) => { void submit(event); }} className="mt-6 space-y-4" aria-describedby={error ? "auth-error" : undefined}>
        <div><label htmlFor="auth-username" className="block text-sm">用户名</label>
          <input id="auth-username" ref={usernameRef} name="username" autoComplete="username" required minLength={mode === "register" ? 2 : 1} maxLength={50} value={username} onChange={(event) => setUsername(event.target.value)} disabled={pending} className="mt-1 w-full rounded border p-2" /></div>
        {mode === "register" && <div><label htmlFor="auth-email" className="block text-sm">邮箱</label>
          <input id="auth-email" name="email" type="email" autoComplete="email" required maxLength={100} value={email} onChange={(event) => setEmail(event.target.value)} disabled={pending} className="mt-1 w-full rounded border p-2" /></div>}
        <div><label htmlFor="auth-password" className="block text-sm">密码{mode === "register" ? "（至少 8 位）" : ""}</label>
          <input id="auth-password" name="password" type="password" autoComplete={mode === "register" ? "new-password" : "current-password"} required minLength={mode === "register" ? 8 : 1} value={password} onChange={(event) => setPassword(event.target.value)} disabled={pending} className="mt-1 w-full rounded border p-2" /></div>
        {error && <p id="auth-error" ref={errorRef} tabIndex={-1} role="alert" className="text-sm text-[var(--tf-danger)]">{error}</p>}
        <p role="status" className="text-sm">{pending ? "正在提交…" : notice}</p>
        <button type="submit" disabled={pending} className="w-full rounded bg-[var(--tf-brand)] px-4 py-2 text-white disabled:opacity-50">{mode === "login" ? "登录" : "注册"}</button>
      </form>
      <button type="button" disabled={pending} className="mt-4 text-sm underline disabled:opacity-50" onClick={() => {
        setMode(mode === "login" ? "register" : "login"); setError(""); setNotice(""); setPassword(""); usernameRef.current?.focus();
      }}>{mode === "login" ? "没有账号？注册" : "已有账号？登录"}</button>
      <p className="mt-4 text-xs">刷新页面后需要重新登录。</p>
    </section>
  </main>;
}

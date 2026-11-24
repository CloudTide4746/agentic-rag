#!/usr/bin/env python3
"""向本地 Redis 写入 apps/api 的登录会话种子数据。

依据 apps/api 的实际实现，不引入任何第三方依赖：

* `internal/auth/repository/auth_repository.go`
  - `Put` 写入 key `auth:user:<userID>`，value 为完整 JWT 字符串，TTL = TokenTTL(2h)
  - `Matches` 用 key 的 value 与请求令牌做字符串全等比较
* `internal/auth/repository/token_signer.go`
  - HS256，claims: `user_id`、`username`、`iss=agentic-rag`、`iat`、`exp`
  - `Parse` 校验 `${JWT_SECRET}`、issuer 与签名算法

因此种子必须用当前 `.env` 的 JWT_SECRET 现签令牌：写死在文件里的令牌会在两小时后失效。

用法：
    python apps/api/scripts/seed_redis.py
    python apps/api/scripts/seed_redis.py --ttl 7200 --dry-run

环境变量（默认从 apps/api/.env 读取，进程环境优先）：
    REDIS_ADDR / REDIS_PASSWORD / REDIS_DB / JWT_SECRET
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import hmac
import json
import os
import socket
import sys
import time
from pathlib import Path

# 与 apps/api/migrations 种子保持一致：id -> username。
# 用户 6（frank）status=0，登录接口按 status=1 过滤，故不为其写入会话。
SEEDED_USERS = {
    1: "alice",
    2: "bob",
    3: "carol",
    4: "dave",
    5: "erin",
}

ENV_PATH = Path(__file__).resolve().parent.parent / ".env"
JWT_ISSUER = "agentic-rag"


def load_env_file(path: Path) -> dict[str, str]:
    """读取 KEY=VALUE 形式的 .env，去掉可选引号；与 run-local.ps1 的解析保持一致。"""
    values: dict[str, str] = {}
    if not path.is_file():
        return values
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            raise SystemExit(f"格式错误（应为 KEY=VALUE）：{path} -> {raw!r}")
        key, value = line.split("=", 1)
        value = value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in ("'", '"'):
            value = value[1:-1]
        values[key.strip()] = value
    return values


def b64url(raw: bytes) -> str:
    return base64.urlsafe_b64encode(raw).rstrip(b"=").decode("ascii")


def sign_jwt(secret: bytes, user_id: int, username: str, ttl_seconds: int) -> str:
    now = int(time.time())
    header = {"alg": "HS256", "typ": "JWT"}
    payload = {
        "user_id": user_id,
        "username": username,
        "iss": JWT_ISSUER,
        "iat": now,
        "exp": now + ttl_seconds,
    }
    signing_input = (
        b64url(json.dumps(header, separators=(",", ":")).encode())
        + "."
        + b64url(json.dumps(payload, separators=(",", ":")).encode())
    )
    signature = hmac.new(secret, signing_input.encode("ascii"), hashlib.sha256).digest()
    return f"{signing_input}.{b64url(signature)}"


class RedisConnection:
    """够用的最小 RESP 客户端：只需 INLINE/SET/SELECT/PING/DEL 一类命令。"""

    def __init__(self, host: str, port: int, password: str, db: int, timeout: float = 5.0):
        self.sock = socket.create_connection((host, port), timeout=timeout)
        self.sock.settimeout(timeout)
        self.file = self.sock.makefile("rwb")
        if password:
            self.command("AUTH", password)
        if db:
            self.command("SELECT", str(db))

    def command(self, *args: str):
        payload = b"*" + str(len(args)).encode() + b"\r\n"
        for arg in args:
            data = arg.encode("utf-8")
            payload += b"$" + str(len(data)).encode() + b"\r\n" + data + b"\r\n"
        self.file.write(payload)
        self.file.flush()
        return self._read_reply()

    def _read_reply(self):
        line = self.file.readline()
        if not line:
            raise ConnectionError("Redis 连接已关闭")
        marker, body = line[:1], line[1:].strip()
        if marker == b"+":
            return body.decode()
        if marker == b"-":
            raise RuntimeError(f"Redis 返回错误：{body.decode()}")
        if marker == b":":
            return int(body)
        if marker == b"$":
            length = int(body)
            if length == -1:
                return None
            data = self.file.read(length + 2)[:-2]
            return data.decode("utf-8", "replace")
        if marker == b"*":
            count = int(body)
            if count == -1:
                return None
            return [self._read_reply() for _ in range(count)]
        raise RuntimeError(f"无法解析的 Redis 响应：{line!r}")

    def close(self) -> None:
        try:
            self.file.close()
        finally:
            self.sock.close()


def main() -> int:
    # Windows 控制台默认 GBK，中文提示会乱码；显式切到 UTF-8 并替换无法编码的字符。
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")

    parser = argparse.ArgumentParser(description="写入 apps/api Redis 登录会话种子")
    parser.add_argument("--ttl", type=int, default=7200, help="会话与令牌有效期（秒），默认 7200")
    parser.add_argument("--dry-run", action="store_true", help="只打印将要写入的命令，不连接 Redis")
    parser.add_argument("--flush-seed-keys", action="store_true", help="先删除 auth:user:* 种子 key")
    args = parser.parse_args()

    if args.ttl <= 0:
        raise SystemExit("--ttl 必须为正整数")

    env = load_env_file(ENV_PATH)
    for key, value in env.items():
        os.environ.setdefault(key, value)

    secret = os.environ.get("JWT_SECRET", "").strip()
    if len(secret) < 32:
        raise SystemExit("JWT_SECRET 未设置或少于 32 个字符；请检查 apps/api/.env")

    addr = os.environ.get("REDIS_ADDR", "127.0.0.1:6379").strip()
    password = os.environ.get("REDIS_PASSWORD", "")
    db_raw = os.environ.get("REDIS_DB", "").strip()
    db = int(db_raw) if db_raw else 0

    host, _, port_raw = addr.rpartition(":")
    if not host:
        host, port_raw = "127.0.0.1", addr
    try:
        port = int(port_raw)
    except ValueError as exc:
        raise SystemExit(f"REDIS_ADDR 端口无效：{addr!r}") from exc

    tokens = {uid: sign_jwt(secret.encode(), uid, name, args.ttl) for uid, name in SEEDED_USERS.items()}

    print(f"Redis 目标：{host}:{port} db={db} ttl={args.ttl}s")
    print(f"JWT issuer={JWT_ISSUER} 签名密钥来源：{ENV_PATH}")

    if args.dry_run:
        for uid, token in tokens.items():
            print(f'SET auth:user:{uid} "{token}" EX {args.ttl}   # {SEEDED_USERS[uid]}')
        return 0

    conn = RedisConnection(host, port, password, db)
    try:
        conn.command("PING")  # 先确认可用，失败直接抛出
        if args.flush_seed_keys:
            existing = conn.command("KEYS", "auth:user:*") or []
            if existing:
                conn.command("DEL", *existing)
                print(f"已删除旧会话 key：{len(existing)} 个")

        for uid, token in tokens.items():
            conn.command("SET", f"auth:user:{uid}", token, "EX", str(args.ttl))
            print(f"OK auth:user:{uid} -> {SEEDED_USERS[uid]}（exp={int(time.time()) + args.ttl}）")

        written = conn.command("KEYS", "auth:user:*") or []
        print(f"当前 auth:user:* 共 {len(written)} 个 key")
        print("\n联调用令牌（Bearer 后接该值，%d 秒后失效，重跑本脚本即可刷新）：" % args.ttl)
        for uid, token in tokens.items():
            print(f"  {SEEDED_USERS[uid]:<6} {token}")
    finally:
        conn.close()

    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (ConnectionError, OSError) as exc:
        sys.exit(f"连接 Redis 失败：{exc}（确认 REDIS_ADDR={os.environ.get('REDIS_ADDR')} 可访问）")

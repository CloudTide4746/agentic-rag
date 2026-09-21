"""启动自检 — ES 连通性 + DeepSeek/DashScope 双 AI 密钥可用性

lifespan 启动阶段执行（main.py 调用 run_startup_checks）：
- 三项检查并行、单项限时，密钥通过最小真实请求验证（鉴权/余额/模型名）；
- 异常仅在终端打印醒目提示，不阻断启动（/api/health 返回 degraded）。
"""
import asyncio
import sys

import httpx

from .config import settings
from .es import elastic

_TIMEOUT = 15  # 单项检查超时（秒）


def _print_block(text: str) -> None:
    """UTF-8 直写终端，避免 Windows GBK 控制台/管道重定向时中文乱码"""
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except (AttributeError, ValueError):
        pass
    print(text, flush=True)


def _err_message(r: httpx.Response) -> str:
    """提取 OpenAI 兼容错误体中的 error.message（截断）"""
    try:
        err = r.json().get("error")
        if isinstance(err, dict) and err.get("message"):
            return str(err["message"])[:160]
    except ValueError:
        pass
    return (r.text or "无错误详情")[:160]


async def check_es() -> tuple[bool, str]:
    try:
        r = await elastic.client.get("/")
    except httpx.HTTPError as e:
        return False, f"连接失败（{type(e).__name__}）— 请确认 Elasticsearch 已启动：{settings.es_url}"
    if r.status_code == 200:
        return True, f"连接正常（{settings.es_url}）"
    if r.status_code == 401:
        return False, "认证失败（HTTP 401）— 请检查 .env 中 ES_USER / ES_PASSWORD"
    return False, f"响应异常（HTTP {r.status_code}）— {settings.es_url}"


async def check_deepseek() -> tuple[bool, str]:
    """最小 chat 请求验证 DeepSeek 密钥（覆盖鉴权 / 余额 / 模型名）"""
    if not settings.deepseek_api_key:
        return False, "未配置 DEEPSEEK_API_KEY — 请在 backend/.env 中设置"
    try:
        async with httpx.AsyncClient(base_url=settings.deepseek_base_url, timeout=_TIMEOUT) as client:
            r = await client.post(
                "/chat/completions",
                headers={"Authorization": f"Bearer {settings.deepseek_api_key}"},
                json={"model": settings.llm_model, "messages": [{"role": "user", "content": "ping"}], "max_tokens": 1},
            )
    except httpx.HTTPError as e:
        return False, f"请求失败（{type(e).__name__}）— 请检查网络与 DEEPSEEK_BASE_URL"
    if r.status_code == 200:
        return True, f"密钥有效（{settings.llm_model}）"
    hint = {401: "密钥无效", 402: "账户余额不足", 404: "模型不存在（检查 LLM_MODEL）"}.get(r.status_code, "密钥不可用")
    return False, f"{hint}（HTTP {r.status_code}）：{_err_message(r)}"


async def check_dashscope() -> tuple[bool, str]:
    """单条向量化验证 DashScope 密钥（覆盖鉴权 / 模型名）"""
    if not settings.dashscope_api_key:
        return False, "未配置 DASHSCOPE_API_KEY — 请在 backend/.env 中设置"
    try:
        async with httpx.AsyncClient(base_url=settings.dashscope_base_url, timeout=_TIMEOUT) as client:
            r = await client.post(
                "/embeddings",
                headers={"Authorization": f"Bearer {settings.dashscope_api_key}"},
                json={"model": settings.embed_model, "input": ["ping"], "dimensions": settings.embed_dims, "encoding_format": "float"},
            )
    except httpx.HTTPError as e:
        return False, f"请求失败（{type(e).__name__}）— 请检查网络与 DASHSCOPE_BASE_URL"
    if r.status_code == 200:
        return True, f"密钥有效（{settings.embed_model} / {settings.embed_dims} 维）"
    hint = {401: "密钥无效", 404: "模型不存在（检查 EMBED_MODEL）"}.get(r.status_code, "密钥不可用")
    return False, f"{hint}（HTTP {r.status_code}）：{_err_message(r)}"


async def run_startup_checks() -> None:
    """并行执行三项检查，终端输出醒目摘要；自检自身异常不阻断启动"""
    try:
        results = await asyncio.gather(check_es(), check_deepseek(), check_dashscope())
    except Exception as e:  # noqa: BLE001
        _print_block(f"[启动自检] 执行异常，已跳过：{type(e).__name__}: {e}")
        return
    checks = [
        ("Elasticsearch", results[0]),
        ("DeepSeek LLM 密钥", results[1]),
        ("DashScope Embedding 密钥", results[2]),
    ]
    bar = "=" * 76
    lines = [bar, "[启动自检]"]
    for name, (ok, detail) in checks:
        lines.append(f"  {'[OK]' if ok else '[FAIL]':<7} {name}：{detail}")
    problems = [f"{name} — {detail}" for name, (ok, detail) in checks if not ok]
    if problems:
        lines.append("-" * 76)
        lines.append("以下依赖异常（服务仍将启动，对应功能不可用）：")
        lines.extend(f"  * {p}" for p in problems)
    else:
        lines.append("全部依赖正常。")
    lines.append(bar)
    _print_block("\n".join(lines))

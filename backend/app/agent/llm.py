"""LLM 客户端 — langchain-deepseek（deepseek-v4-flash-vision-exp，OpenAI 兼容，多模态）

同一模型兼做 Chat 与 LLM-as-Judge（设计书 2.3）：
- judge()：结构化判定（Router/Planner/Grader/CRAG/Self-RAG/NL2SQL），
  JSON Output + Pydantic 校验，低温度
- stream_generate()：流式生成，逐 token yield，回传 usage
"""
import json
import re
from collections.abc import AsyncGenerator
from typing import TypeVar

from langchain_core.messages import AIMessage, BaseMessage, HumanMessage
from langchain_deepseek import ChatDeepSeek
from pydantic import BaseModel

from ..config import settings

T = TypeVar("T", bound=BaseModel)

# 轻量判定步骤用低温度 + 紧凑 max_tokens，压缩延迟
_judge_llm = ChatDeepSeek(
    model=settings.llm_model,
    api_key=settings.deepseek_api_key,
    api_base=settings.deepseek_base_url,
    temperature=0.2,
    max_tokens=1024,
    timeout=60,
    # response_format 非该版本 ChatDeepSeek 的显式字段，放入 model_kwargs 透传，
    # 避免触发 "not default parameter" UserWarning（效果与直接传参一致）
    model_kwargs={"response_format": {"type": "json_object"}},
)

_gen_llm = ChatDeepSeek(
    model=settings.llm_model,
    api_key=settings.deepseek_api_key,
    api_base=settings.deepseek_base_url,
    temperature=0.7,
    stream_usage=True,
    timeout=120,
)


def _extract_json(text: str) -> dict:
    """从模型输出提取 JSON 对象（容忍 ```json 围栏与前后缀文本）"""
    text = text.strip()
    m = re.search(r"\{[\s\S]*\}", text)
    if not m:
        raise ValueError(f"LLM 未返回 JSON：{text[:200]}")
    return json.loads(m.group(0))


def _pick_usage(msg: BaseMessage) -> tuple[int, int]:
    """从 response_metadata / usage_metadata 提取 (prompt, completion)"""
    meta = getattr(msg, "response_metadata", {}) or {}
    tu = meta.get("token_usage") or {}
    if isinstance(tu, dict) and tu.get("prompt_tokens") is not None:
        return int(tu.get("prompt_tokens", 0)), int(tu.get("completion_tokens", 0))
    um = getattr(msg, "usage_metadata", None) or {}
    if um.get("input_tokens") is not None:
        return int(um.get("input_tokens", 0)), int(um.get("output_tokens", 0))
    return 0, 0


async def judge(schema: type[T], system: str, user: str, usage: dict | None = None, images: list[str] | None = None) -> T:
    """结构化判定：返回 Pydantic 实例；usage 字典（可None）累计 token 用量

    images 非空时末条 human 消息携带图片（vision 模型读图后输出 JSON，供 Router 提取图中要点）。
    """
    if images:
        blocks: list[dict] = [{"type": "text", "text": user}]
        blocks += [{"type": "image_url", "image_url": {"url": u}} for u in images]
        msgs: list = [("system", system), HumanMessage(content=blocks)]
    else:
        msgs = [("system", system), ("human", user)]
    resp: AIMessage = await _judge_llm.ainvoke(msgs)
    if usage is not None:
        p, c = _pick_usage(resp)
        usage["prompt"] = usage.get("prompt", 0) + p
        usage["completion"] = usage.get("completion", 0) + c
    data = _extract_json(resp.content if isinstance(resp.content, str) else str(resp.content))
    return schema.model_validate(data)


async def stream_generate(
    messages: list, usage: dict | None = None
) -> AsyncGenerator[str, None]:
    """流式生成：逐段 yield 文本增量；完成后 usage 写入字典"""
    collected: list[str] = []
    prompt_tokens = completion_tokens = 0
    async for chunk in _gen_llm.astream(messages):
        meta = getattr(chunk, "response_metadata", {}) or {}
        tu = meta.get("token_usage") or {}
        if isinstance(tu, dict) and tu.get("prompt_tokens") is not None:
            prompt_tokens, completion_tokens = int(tu.get("prompt_tokens", 0)), int(tu.get("completion_tokens", 0))
        um = getattr(chunk, "usage_metadata", None) or {}
        if um.get("input_tokens") is not None:
            prompt_tokens, completion_tokens = int(um.get("input_tokens", 0)), int(um.get("output_tokens", 0))
        if isinstance(chunk.content, str) and chunk.content:
            collected.append(chunk.content)
            yield chunk.content
    if usage is not None and (prompt_tokens or completion_tokens):
        usage["prompt"] = usage.get("prompt", 0) + prompt_tokens
        usage["completion"] = usage.get("completion", 0) + completion_tokens


def chat_messages(system: str, history: list[dict], user_content: str, images: list[str] | None = None) -> list:
    """组装多轮对话消息（history 为 [{'role','content'}]）

    images 非空时，末条 human 消息构造为 OpenAI 视觉多模态 content 块
    （text + image_url，url 为 data:image/…;base64,…），由 vision 模型直接读图。
    """
    msgs: list = [("system", system)]
    for h in history or []:
        role = h.get("role", "user")
        content = h.get("content", "")
        if h.get("images"):
            content = f"{content}（附图 ×{len(h['images'])}）"
        msgs.append(("ai" if role == "assistant" else "human", content))
    if images:
        content: list[dict] = [{"type": "text", "text": user_content}]
        content += [{"type": "image_url", "image_url": {"url": u}} for u in images]
        msgs.append(HumanMessage(content=content))
    else:
        msgs.append(("human", user_content))
    return msgs

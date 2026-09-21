"""Agentic RAG 编排层 — 六大决策点驱动的 SSE 问答主流程（设计书 4.1 / 5.3）

事件协议严格对齐前端 types.ts：router / rewrite / retrieve / grade / correct /
generate / reflect / citation / done / error 十类事件；
决策链 steps 与 SSE 事件同源落库 agent_decision_log（M3）。
"""
import asyncio
import json
import logging
import re
import time
from collections.abc import AsyncGenerator

from pydantic import BaseModel

from .. import db
from ..config import settings
from ..services.chat_images import to_data_urls
from . import prompts
from .llm import chat_messages, judge, stream_generate
from .retrievers import db_retriever, enabled_sources, gather_lanes
from .rrf import RetrieverResult, rrf_fuse

logger = logging.getLogger("agentic.orchestrator")

INTENT_LABELS = {
    "chitchat": "闲聊（前置拦截，不触发检索）",
    "kb_qa": "知识问答",
    "data_query": "数据查询（跳过改写，直达数据库检索）",
    "summarize": "总结（走生成侧，轻量检索）",
}
VERDICT_ICON = {"pass": "✓ 通过", "partial": "⚠️ 部分相关", "fail": "✗ 不及格"}


# ---------- 结构化输出 Schema（设计书 7.2） ----------

class RouterOut(BaseModel):
    intent: str = "kb_qa"
    confidence: float = 0.8
    image_note: str = ""  # 带图时由 Router 视觉提取的图中内容要点（供检索/评估使用）


class PlannerOut(BaseModel):
    rewritten_query: str = ""
    sub_queries: list[str] = []
    need_hyde: bool = False
    hyde_doc: str = ""


class GraderOut(BaseModel):
    score: float = 0.0
    reason: str = ""


class CragOut(BaseModel):
    failure_reason: str = ""
    action: str = "rewrite_retry"
    new_queries: list[str] = []


class ReflectOut(BaseModel):
    result: str = "pass"
    missing_info: str = ""


async def _judge_retry(
    schema: type[BaseModel], system: str, user: str, usage: dict, attempts: int = 2, images: list[str] | None = None
):
    """LLM 判定调用，失败重试 1 次（设计书 9 容错）；images 供视觉判定（Router 读图）"""
    last_err: Exception | None = None
    for _ in range(attempts):
        try:
            return await judge(schema, system, user, usage, images=images)
        except Exception as e:  # noqa: BLE001
            last_err = e
            logger.warning("judge 调用失败将重试：%s", e)
    raise last_err  # type: ignore[misc]


def sse(etype: str, step: int, detail: str = "", **fields) -> str:
    """构造 SSE 事件文本（event:/data: 行，\n\n 结尾，与前端 SseParser 协议一致）"""
    payload = {"step": step, "ts": int(time.time() * 1000), "detail": detail, **fields}
    return f"event: {etype}\ndata: {json.dumps(payload, ensure_ascii=False)}\n\n"


# ---------- 运行态（事件与日志同源） ----------

class RunState:
    def __init__(self, session_id: int, question: str, image_urls: list[str] | None = None) -> None:
        self.session_id = session_id
        self.question = question
        self.image_urls = image_urls or []  # 站内 URL（落库 chat_message.images）
        self.image_data_urls = to_data_urls(self.image_urls)  # base64 data URL（喂给视觉模型）
        self.vision_note = ""  # Router 视觉提取的图中内容要点（纯文本决策点的图片语义来源）
        self.usage: dict[str, int] = {"prompt": 0, "completion": 0}
        self.steps: list[dict] = []
        self.answer = ""
        self.citations: list[dict] = []
        self.correct_count = 0
        self.reflect_fail_count = 0
        self.started = time.time()
        self._step_t0 = time.time()
        self._gen_t0: float | None = None

    def mark(self, step: int, type_: str, detail: str, payload: dict) -> None:
        now = time.time()
        self.steps.append(
            {
                "step": step,
                "type": type_,
                "detail": detail,
                "payload": payload,
                "elapsed_ms": max(0, int((now - self._step_t0) * 1000)),
            }
        )
        self._step_t0 = now

    def gen_start(self) -> None:
        self._gen_t0 = time.time()

    def gen_end(self) -> None:
        if self._gen_t0 is not None and self.answer:
            now = time.time()
            self.steps.append(
                {
                    "step": 6,
                    "type": "generate",
                    "detail": f"答案生成完成（{len(self.answer)} 字）",
                    "payload": {"chars": len(self.answer)},
                    "elapsed_ms": max(0, int((now - self._gen_t0) * 1000)),
                }
            )
            self._gen_t0 = None
            self._step_t0 = now

    @property
    def elapsed_ms(self) -> int:
        return max(0, int((time.time() - self.started) * 1000))

    @property
    def question_full(self) -> str:
        """规划/评估/纠正用完整问题：带图时并入 Router 视觉提取的图中要点

        用户文字可能只是"回答这个问题"，真正语义在图中——纯文本决策点
        （Planner/Grader/CRAG/Reflect）必须基于 question_full 才能检索与评估到位；
        生成侧仍用原 question（模型可直接看图）。
        """
        if self.vision_note:
            return f"{self.question}\n（用户附图，图中内容要点：{self.vision_note}）"
        return self.question

    async def persist(self) -> tuple[int, int]:
        """落库：assistant 消息 + 决策日志（事件与日志同源，设计书 M3）"""
        rounds = max(1, self.correct_count + self.reflect_fail_count)
        content = self.answer or "（本次问答链路异常中断，未生成答案，详见决策日志）"
        msg_id = await db.aexecute(
            "INSERT INTO chat_message (session_id, role, content, citations) VALUES (%s, 'assistant', %s, %s)",
            (self.session_id, content, json.dumps(self.citations, ensure_ascii=False) if self.citations else None),
        )
        log_id = await db.aexecute(
            "INSERT INTO agent_decision_log (session_id, message_id, question, final_answer, steps, total_rounds, elapsed_ms)"
            " VALUES (%s, %s, %s, %s, %s, %s, %s)",
            (
                self.session_id,
                msg_id,
                self.question,
                content,
                json.dumps(self.steps, ensure_ascii=False),
                rounds,
                self.elapsed_ms,
            ),
        )
        await db.aexecute("UPDATE chat_message SET decision_log_id=%s WHERE id=%s", (log_id, msg_id))
        await db.aexecute("UPDATE chat_session SET updated_at=NOW() WHERE id=%s", (self.session_id,))
        return msg_id, log_id


# ---------- 上下文与引用 ----------

def build_context(items: list[RetrieverResult]) -> list[str]:
    blocks: list[str] = []
    for i, it in enumerate(items, 1):
        meta = it.metadata
        if meta.get("db"):
            blocks.append(f"[{i}] 业务数据库 sales 表查询结果（来自业务数据库）\n{it.content[:500]}")
        else:
            blocks.append(
                f"[{i}] 《{meta.get('doc_name', '')}》{meta.get('section_path', '')}（切片#{meta.get('chunk_no', 0)}）\n{it.content[:600]}"
            )
    return blocks


def extract_citations(answer: str, items: list[RetrieverResult]) -> list[dict]:
    """句尾 [n] 序号 → 片段元数据映射，非法序号剔除（设计书 Q7）"""
    seen: set[int] = set()
    out: list[dict] = []
    for m in re.finditer(r"\[(\d+)\]", answer):
        idx = int(m.group(1))
        if not (1 <= idx <= len(items)) or idx in seen:
            continue
        seen.add(idx)
        meta = items[idx - 1].metadata
        if meta.get("db"):
            # NL2SQL 引用无文档切片可定位：携带 SQL + 结果行，前端改走「数据查询证据」视图
            out.append(
                {
                    "source_type": "db",
                    "doc_name": "业务数据库 sales 表（NL2SQL 只读）",
                    "section_path": "查询结果集",
                    "chunk_no": 0,
                    "sql": meta.get("sql", ""),
                    "rows": meta.get("rows", []),
                }
            )
        else:
            out.append(
                {
                    "doc_name": meta.get("doc_name", ""),
                    "section_path": meta.get("section_path", ""),
                    "chunk_no": meta.get("chunk_no", 0),
                }
            )
    return out


def strip_invalid_refs(answer: str, items: list[RetrieverResult]) -> str:
    """剔除答案中未映射到合法片段的 [n] 序号（避免渲染为残留明文）"""
    valid = {i + 1 for i in range(len(items))}

    def _repl(m: re.Match) -> str:
        return m.group(0) if int(m.group(1)) in valid else ""

    return re.sub(r"\[(\d+)\]", _repl, answer)


async def do_grade(question: str, sub_queries: list[str], ctx_blocks: list[str], usage: dict) -> tuple[float, str, str]:
    """④ Grader：0-10 打分 + 阈值判定（≥7 pass / ≥5 partial / <5 fail）"""
    if not ctx_blocks:
        return 0.0, "fail", "所有启用数据源均无命中，建议改写查询重试（CRAG rewrite_retry）"
    g = await _judge_retry(GraderOut, prompts.GRADER_SYS, prompts.grader_user(question, sub_queries, ctx_blocks), usage)
    score = round(min(10.0, max(0.0, float(g.score))), 1)
    verdict = "pass" if score >= settings.grade_pass else ("partial" if score >= settings.grade_partial else "fail")
    return score, verdict, g.reason or "无"


# ---------- 生成 + Self-RAG + 引用 + done 收尾 ----------

async def _generate_and_finish(
    st: RunState,
    history: list[dict],
    kb_ids: list[int] | None,
    ctx_blocks: list[str],
    items: list[RetrieverResult],
    disclaimer: str = "",
    supp_db: bool = False,
) -> AsyncGenerator[dict, None]:
    # ⑥ 生成（流式；带图时末条 human 消息为视觉多模态 content 块）
    gen_user = prompts.generate_user(st.question, ctx_blocks, history, disclaimer)
    msgs = chat_messages(prompts.GENERATE_SYS, history, gen_user, st.image_data_urls)
    st.gen_start()
    async for delta in stream_generate(msgs, st.usage):
        st.answer += delta
        yield sse("generate", 6, "", delta=delta)
    st.gen_end()

    # ⑥ Self-RAG 反思（不达标回流补检，≤2 次；基于 question_full 评估，带图语义完整）
    round_i = 0
    reflect_passed = False
    while round_i < settings.reflect_max_rounds:
        round_i += 1
        rf = await _judge_retry(ReflectOut, prompts.REFLECT_SYS, prompts.reflect_user(st.question_full, st.answer, ctx_blocks), st.usage)
        if rf.result == "pass":
            reflect_passed = True
            detail = "自检通过 ✓" + (f"（第 {round_i} 次反思）" if round_i > 1 else "")
            yield sse("reflect", 6, detail, result="pass", round=round_i)
            st.mark(6, "reflect", detail, {"result": "pass", "round": round_i})
            break
        st.reflect_fail_count += 1
        detail = f"自检不达标 ✗ → {rf.missing_info or '答案不完整'}，回流补充检索"
        yield sse("reflect", 6, detail, result="fail", round=round_i, missing_info=rf.missing_info)
        st.mark(6, "reflect", detail, {"result": "fail", "round": round_i, "missing_info": rf.missing_info})
        if round_i >= settings.reflect_max_rounds:
            break
        # 反思回流：定向补检（缺失信息并入查询；data_query 场景回数据库，其余回知识库）→ 重新评估 → 重新生成
        supp_q = f"{st.question_full} {rf.missing_info}".strip()
        if supp_db:
            sql, db_rows = await db_retriever.search(supp_q)
            detail = f"反思回流·NL2SQL 补查：{sql[:60]}（命中{len(db_rows)}行）"
            yield sse("retrieve", 3, detail, source="database", hit_count=len(db_rows))
            st.mark(3, "retrieve", detail, {"source": "database", "hit_count": len(db_rows), "sql": sql})
            lanes = {"database": db_rows}
        else:
            lanes = await gather_lanes(supp_q, kb_ids, settings.retrieval_top_k, ["vector", "keyword"])
        for src in ("vector", "keyword"):
            if src in lanes:
                hits = lanes[src]
                label = "反思回流·定向补检" if src == "vector" else "反思回流·关键词补检"
                detail = f"{label}：命中 {len(hits)} 条"
                yield sse("retrieve", 3, detail, source=src, hit_count=len(hits))
                st.mark(3, "retrieve", detail, {"source": src, "hit_count": len(hits)})
        fused = rrf_fuse(lanes, settings.fused_top_n)
        items = [f["item"] for f in fused]
        ctx_blocks = build_context(items)
        score, verdict, reason = await do_grade(st.question_full, [], ctx_blocks, st.usage)
        detail = f"相关度 {score} {VERDICT_ICON[verdict]}（反思回流后）"
        yield sse("grade", 4, detail, score=score, verdict=verdict, reason=reason)
        st.mark(4, "grade", detail, {"score": score, "verdict": verdict, "reason": reason})
        gen_user = prompts.generate_user(st.question, ctx_blocks, history)
        msgs = chat_messages(prompts.GENERATE_SYS, history, gen_user, st.image_data_urls)
        st.gen_start()
        async for delta in stream_generate(msgs, st.usage):
            st.answer += delta
            yield sse("generate", 6, "", delta=delta)
        st.gen_end()
    if not reflect_passed and round_i >= settings.reflect_max_rounds:
        st.answer += "\n\n> ⚠️ 说明：本答案经过 Self-RAG 多轮自检仍未完全达标，请结合引用出处核实关键信息。"

    # ⑦ 引用溯源（非法序号已从答案剔除）
    st.answer = strip_invalid_refs(st.answer, items)
    st.citations = extract_citations(st.answer, items)
    cite_detail = f"引用 {len(st.citations)} 条出处" if st.citations else "无知识库引用（推断生成）"
    yield sse("citation", 7, cite_detail, citations=st.citations)
    st.mark(7, "citation", cite_detail, {"citations": st.citations})

    msg_id, _ = await st.persist()
    yield sse(
        "done",
        0,
        "",
        message_id=msg_id,
        elapsed_ms=st.elapsed_ms,
        token_usage={"prompt": st.usage.get("prompt", 0), "completion": st.usage.get("completion", 0)},
    )


# ---------- 问答主流程 ----------

async def _run(st: RunState, kb_ids: list[int] | None, skip_user_persist: bool) -> AsyncGenerator[dict, None]:
    session_id = st.session_id
    question = st.question

    # Q8 多轮上下文（取历史先于落用户消息，避免把当前问题算进历史；images 供附图标记）
    history = list(
        reversed(
            await db.aquery_all(
                "SELECT role, content, images FROM chat_message WHERE session_id=%s ORDER BY id DESC LIMIT %s",
                (session_id, settings.history_window),
            )
        )
    )
    if not skip_user_persist:
        await db.aexecute(
            "INSERT INTO chat_message (session_id, role, content, images) VALUES (%s, 'user', %s, %s)",
            (session_id, question, json.dumps(st.image_urls, ensure_ascii=False) if st.image_urls else None),
        )
    sess = await db.aquery_one("SELECT title FROM chat_session WHERE id=%s", (session_id,))
    if sess is None:
        raise ValueError("会话不存在")
    if sess["title"] == "新会话":
        await db.aexecute(
            "UPDATE chat_session SET title=%s WHERE id=%s",
            (question[:18] + ("…" if len(question) > 18 else ""), session_id),
        )

    # ① Router 意图路由（带图时直接读图提取图中要点；带图问题一律 kb_qa：先检索知识库，再结合图片回答）
    r = await _judge_retry(
        RouterOut,
        prompts.ROUTER_SYS,
        prompts.router_user(question, history, len(st.image_data_urls)),
        st.usage,
        images=st.image_data_urls,
    )
    if r.intent not in INTENT_LABELS:
        r.intent = "kb_qa"
    # 代码层兜底强制：附图消息不走闲聊/数据查询旁路，统一主链检索 + 视觉生成
    if st.image_data_urls:
        r.intent = "kb_qa"
        st.vision_note = (r.image_note or "").strip()
    r.confidence = min(1.0, max(0.0, float(r.confidence)))
    detail = f"正在判断问题类型 → {INTENT_LABELS[r.intent]}"
    if st.image_data_urls:
        note_txt = st.vision_note[:40] + ("…" if len(st.vision_note) > 40 else "")
        detail += f"（附图问答，图中要点：{note_txt or '未提取到'}）"
    yield sse("router", 1, detail, intent=r.intent, confidence=round(r.confidence, 2))
    st.mark(
        1,
        "router",
        detail,
        {"intent": r.intent, "confidence": r.confidence, "has_images": bool(st.image_data_urls), "vision_note": st.vision_note},
    )

    # ---- 闲聊旁路：直接流式回答（不检索，设计书 Q1；纯文本闲聊，图片已被上方强制改道） ----
    if r.intent == "chitchat":
        st.gen_start()
        msgs = chat_messages(prompts.CHAT_SYS, history, question, st.image_data_urls)
        async for delta in stream_generate(msgs, st.usage):
            st.answer += delta
            yield sse("generate", 0, "", delta=delta)
        st.gen_end()
        msg_id, _ = await st.persist()
        yield sse(
            "done",
            0,
            "",
            message_id=msg_id,
            elapsed_ms=st.elapsed_ms,
            token_usage={"prompt": st.usage.get("prompt", 0), "completion": st.usage.get("completion", 0)},
        )
        return

    # ---- 数据查询旁路：NL2SQL 直达 DBRetriever（跳过改写） ----
    if r.intent == "data_query":
        sql, rows = await db_retriever.search(st.question_full)
        n = len(rows)
        sql_txt = sql if len(sql) <= 72 else sql[:72] + "…"
        detail = f"NL2SQL 生成并执行：{sql_txt}（命中{n}行）"
        yield sse("retrieve", 3, detail, source="database", hit_count=n)
        st.mark(3, "retrieve", detail, {"source": "database", "hit_count": n, "sql": sql})

        items = rows
        ctx_blocks = build_context(items)
        disclaimer = ""
        score, verdict, reason = await do_grade(st.question_full, [], ctx_blocks, st.usage)
        detail = f"相关度 {score} {VERDICT_ICON[verdict]}"
        yield sse("grade", 4, detail, score=score, verdict=verdict, reason=reason)
        st.mark(4, "grade", detail, {"score": score, "verdict": verdict, "reason": reason})
        if verdict != "pass":
            disclaimer = "注意：业务数据库查询结果与问题相关度不足，请如实说明数据可能不完整。"
        async for ev in _generate_and_finish(st, history, kb_ids, ctx_blocks, items, disclaimer, supp_db=True):
            yield ev
        return

    # ---- 知识问答 / 总结主链路 ----
    src_enabled = await enabled_sources()
    top_k = settings.retrieval_top_k
    sub_queries: list[str] = []
    hyde_used = False
    vector_query = st.question_full

    if r.intent == "kb_qa":
        # ② Planner 查询规划（改写 / 拆解 / HyDE；带图时基于图中要点改写，检索才有语义）
        p = await _judge_retry(PlannerOut, prompts.PLANNER_SYS, prompts.planner_user(st.question_full, history), st.usage)
        main_query = p.rewritten_query or st.question_full
        sub_queries = [q for q in p.sub_queries if q][:3]
        hyde_used = bool(p.need_hyde and p.hyde_doc)
        vector_query = p.hyde_doc if hyde_used else main_query
        detail = (
            f"拆解为 {len(sub_queries)} 个子问题"
            if sub_queries
            else f"改写主查询（HyDE {'已生成' if hyde_used else '未启用'}）"
        )
        yield sse("rewrite", 2, detail, rewritten_query=main_query, sub_queries=sub_queries, hyde_used=hyde_used)
        st.mark(2, "rewrite", detail, {"rewritten_query": main_query, "sub_queries": sub_queries, "hyde_used": hyde_used})
    else:
        main_query = question

    # kb 检索源集合（summarize 走轻量语义检索，只开向量路）
    kb_sources = [s for s in ("vector", "keyword") if src_enabled.get(s)]
    if r.intent == "summarize":
        kb_sources = ["vector"] if src_enabled.get("vector") else kb_sources[:1]

    # ③④⑤ 检索 → 融合 → 评估（partial 补检 / CRAG 纠正 ≤2 轮）
    retrieval_round = 0
    supplemented = False
    crag_round = 0
    disclaimer = ""
    ctx_blocks: list[str] = []
    items: list[RetrieverResult] = []

    while True:
        retrieval_round += 1
        lanes = await gather_lanes(vector_query, kb_ids, top_k, kb_sources)
        for src in ("vector", "keyword"):
            if src in lanes:
                hits = lanes[src]
                label = "向量语义检索" if src == "vector" else "关键词 BM25 检索"
                prefix = "" if retrieval_round == 1 else ("重查·" if crag_round else "补充·")
                detail = f"{prefix}{label}：命中 {len(hits)} 条"
                yield sse("retrieve", 3, detail, source=src, hit_count=len(hits))
                st.mark(3, "retrieve", detail, {"source": src, "hit_count": len(hits)})

        fused = rrf_fuse(lanes, settings.fused_top_n)
        items = [f["item"] for f in fused]
        ctx_blocks = build_context(items)

        score, verdict, reason = await do_grade(st.question_full, sub_queries, ctx_blocks, st.usage)
        suffix = f"（第 {retrieval_round} 轮）" if retrieval_round > 1 else ""
        detail = f"相关度 {score} {VERDICT_ICON[verdict]}{suffix}"
        yield sse("grade", 4, detail, score=score, verdict=verdict, reason=reason)
        st.mark(4, "grade", detail, {"score": score, "verdict": verdict, "reason": reason})

        if verdict == "pass":
            break

        # partial：补充检索一次（仅一次）
        if verdict == "partial" and not supplemented and kb_sources:
            supplemented = True
            st.correct_count += 1
            c_detail = "触发纠正：证据不完整 → 补充检索一次（第 1 轮）"
            yield sse("correct", 5, c_detail, action="rewrite_retry", round=1)
            st.mark(5, "correct", c_detail, {"action": "rewrite_retry", "round": 1})
            vector_query = main_query
            continue

        # fail（或 partial 补检后仍不通过）→ CRAG 纠正
        if crag_round >= settings.crag_max_rounds:
            st.correct_count += 1
            final_round = max(crag_round, 1)
            c_detail = f"纠正轮次耗尽 → 放弃检索，带声明直接生成（第 {final_round} 轮）"
            yield sse("correct", 5, c_detail, action="give_up", round=final_round)
            st.mark(5, "correct", c_detail, {"action": "give_up", "round": final_round})
            disclaimer = (
                "注意：知识库两轮检索评估均未通过（CRAG give_up）。请在回答开头用引用块声明「知识库未覆盖」，"
                "说明以下内容为模型基于通用知识推断，建议用户上传相关文档后重新提问。不要标注引用序号。"
            )
            ctx_blocks = []
            items = []
            break

        crag_round += 1
        diag = await _judge_retry(CragOut, prompts.CRAG_SYS, prompts.crag_user(st.question_full, reason, ctx_blocks), st.usage)
        action = diag.action

        if action == "give_up":
            st.correct_count += 1
            c_detail = f"触发纠正：{diag.failure_reason} → 放弃检索，带声明直接生成（第 {crag_round} 轮）"
            yield sse("correct", 5, c_detail, action="give_up", round=crag_round)
            st.mark(5, "correct", c_detail, {"action": "give_up", "round": crag_round, "reason": diag.failure_reason})
            disclaimer = (
                "注意：知识库检索评估未通过（CRAG give_up）。请在回答开头用引用块声明「知识库未覆盖」，"
                "说明以下内容为模型基于通用知识推断，建议用户上传相关文档后重新提问。不要标注引用序号。"
            )
            ctx_blocks = []
            items = []
            break

        # rewrite_retry：CRAG 改写查询重查
        st.correct_count += 1
        c_detail = f"触发纠正：{diag.failure_reason} → 改写查询重查（第 {crag_round} 轮）"
        yield sse("correct", 5, c_detail, action="rewrite_retry", round=crag_round)
        st.mark(5, "correct", c_detail, {"action": "rewrite_retry", "round": crag_round, "reason": diag.failure_reason})
        new_q = diag.new_queries[0] if diag.new_queries else f"{main_query} 相关 原理 方案"
        main_query = new_q
        vector_query = new_q
        rw_detail = f"CRAG 改写：{diag.failure_reason}"
        yield sse("rewrite", 2, rw_detail, rewritten_query=new_q, sub_queries=sub_queries, hyde_used=hyde_used)
        st.mark(2, "rewrite", rw_detail, {"rewritten_query": new_q, "sub_queries": sub_queries, "hyde_used": hyde_used})

    async for ev in _generate_and_finish(st, history, kb_ids, ctx_blocks, items, disclaimer):
        yield ev


async def run_ask(
    session_id: int,
    question: str,
    kb_ids: list[int] | None,
    skip_user_persist: bool = False,
    image_urls: list[str] | None = None,
) -> AsyncGenerator[dict, None]:
    """SSE 问答流入口：整体受 overall_timeout（120s）约束，异常推 error 事件（设计书 4.1）

    image_urls：站内图片 URL（chat_images.save_images 产物），随消息落库并转 data URL 喂给视觉模型。
    """
    st = RunState(session_id, question, image_urls)
    try:
        async with asyncio.timeout(settings.overall_timeout):
            async for ev in _run(st, kb_ids, skip_user_persist):
                yield ev
    except (TimeoutError, asyncio.TimeoutError):
        detail = "单轮问答总耗时超过 120s 上限，链路中断"
        yield sse("error", 0, detail, code=504, message="全链路超时（120s）：已按约束中断并清理会话状态")
        try:
            await st.persist()
        except Exception:  # noqa: BLE001
            logger.exception("超时场景落库失败")
    except ValueError as e:
        yield sse("error", 0, str(e), code=404, message=str(e))
    except Exception as e:  # noqa: BLE001
        logger.exception("问答链路异常：%s", e)
        msg = f"{e}" or "服务内部异常"
        yield sse("error", 0, "链路异常中断", code=500, message=msg[:300])
        try:
            await st.persist()
        except Exception:  # noqa: BLE001
            logger.exception("异常场景落库失败")

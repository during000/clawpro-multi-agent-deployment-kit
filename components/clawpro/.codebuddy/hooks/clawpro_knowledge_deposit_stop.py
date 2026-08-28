#!/usr/bin/env python3
"""Orchestrate Stop-time ClawPro knowledge deposition and FEC feedback closure.

CodeBuddy's ``Stop`` event fires after each assistant response, not only when a
conversation is closed.  This hook uses the supported ``continue:false`` plus
``reason`` mechanism to return control to the still-running Agent and instruct
it to invoke the repository's knowledge-deposition Skill.

The repository already has an FEC feedback Stop gate.  Multiple blocking Stop
hooks have ambiguous short-circuit semantics across clients, so this hook also
calls that gate's deterministic pending-feedback helpers and combines both
reasons into one resumption.  The original FEC script remains the authority.

The knowledge Skill performs the semantic quality and sensitivity checks.  This
hook is deliberately deterministic and never writes to Gongfeng itself.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import tempfile
from pathlib import Path
from typing import Any

import feedback_stop_gate as fec


SKILL_NAME = "clawpro-knowledge-deposition"
SKILL_PATH = ".codebuddy/skills/clawpro-knowledge-deposition/SKILL.md"
PUBLISH_SCRIPT = (
    ".codebuddy/skills/clawpro-knowledge-deposition/scripts/publish_knowledge.py"
)
DOMAIN_SIGNAL_RE = re.compile(
    r"clawpro|openclaw|teamai|codebuddy|workbuddy|codex|agent|hook|skill|"
    r"rules?|mcp|知识库|知识沉淀|企业规范|系统提示词|本地智能体|本地\s*agent",
    re.IGNORECASE,
)
VALUE_SIGNAL_RE = re.compile(
    r"已完成|已确认|确认了|结论|根因|定位|修复|实现|新增|升级|验证|测试通过|"
    r"需求|规则|约束|边界|决策|方案|风险|发现|支持|不支持|必须|禁止|验收|"
    r"沉淀|召回|发布|推送|分支",
    re.IGNORECASE,
)
EVIDENCE_SIGNAL_RE = re.compile(
    r"(?:^|\s)(?:\.[A-Za-z0-9_-]+/|[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+)|"
    r"TAPD|MR\s*!?\d+|commit|git\s|pytest|self-test|自测|测试|验证|"
    r"\b1\d{8,18}\b",
    re.IGNORECASE | re.MULTILINE,
)
USER_DECISION_RE = re.compile(
    r"确认|可以|同意|就按|决定|改成|应该|不要|必须|需要|升级|优化|提交|"
    r"沉淀|召回|知识库",
    re.IGNORECASE,
)


def emit_allow() -> None:
    print(json.dumps({"continue": True, "suppressOutput": True}, ensure_ascii=False))


def emit_block(reason: str) -> None:
    print(json.dumps({"continue": False, "reason": reason}, ensure_ascii=False))


def load_hook_input() -> dict[str, Any]:
    try:
        payload = json.load(sys.stdin)
    except Exception:
        return {}
    return payload if isinstance(payload, dict) else {}


def content_text(value: Any) -> str:
    if isinstance(value, str):
        return value
    if isinstance(value, list):
        return "\n".join(content_text(item) for item in value)
    if isinstance(value, dict):
        return "\n".join(
            content_text(value[key])
            for key in ("text", "content", "input_text", "output_text")
            if key in value
        )
    return ""


def read_messages(transcript_path: Path) -> list[dict[str, str]]:
    messages: list[dict[str, str]] = []
    try:
        with transcript_path.open(encoding="utf-8") as transcript:
            for line in transcript:
                try:
                    item = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if (
                    not isinstance(item, dict)
                    or item.get("type") != "message"
                    or item.get("role") not in {"user", "assistant"}
                ):
                    continue
                messages.append(
                    {
                        "role": str(item["role"]),
                        "text": content_text(item.get("content", "")).strip(),
                    }
                )
    except OSError:
        return []
    return messages


def latest_complete_turn(messages: list[dict[str, str]]) -> tuple[str, str]:
    for assistant_index in range(len(messages) - 1, -1, -1):
        assistant = messages[assistant_index]
        if assistant["role"] != "assistant" or not assistant["text"]:
            continue
        user_text = ""
        for user_index in range(assistant_index - 1, -1, -1):
            if messages[user_index]["role"] == "user":
                user_text = messages[user_index]["text"]
                break
        return user_text, assistant["text"]
    return "", ""


def knowledge_signal(user_text: str, assistant_text: str) -> tuple[int, list[str]]:
    if len(assistant_text.strip()) < 80:
        return 0, []
    combined = f"{user_text}\n{assistant_text}"
    score = 0
    reasons: list[str] = []
    if DOMAIN_SIGNAL_RE.search(combined):
        score += 2
        reasons.append("命中 ClawPro/Agent 领域")
    if VALUE_SIGNAL_RE.search(assistant_text):
        score += 2
        reasons.append("包含结论、变更或约束")
    if EVIDENCE_SIGNAL_RE.search(assistant_text):
        score += 1
        reasons.append("包含可核验证据")
    if USER_DECISION_RE.search(user_text):
        score += 1
        reasons.append("包含用户明确决策")
    if len(assistant_text) >= 220:
        score += 1
        reasons.append("形成完整说明")
    return score, reasons


def build_reason(
    session_id: str,
    transcript_path: str,
    signal_score: int,
    signal_reasons: list[str],
) -> str:
    signal_text = "；".join(signal_reasons)
    return (
        "【ClawPro 回合级知识沉淀】本轮响应已经完成。请立即使用 Skill "
        f"`{SKILL_NAME}`（规范：`{SKILL_PATH}`），读取转录文件中的最新一个完整"
        "用户/助手回合，执行一次知识沉淀判定。\n"
        f"- session_id：{session_id}\n"
        f"- transcript_path：{transcript_path}\n"
        f"- 预筛得分：{signal_score}（{signal_text}）\n"
        "- 每次 Stop 都执行低成本预筛；只有命中高价值信号才拉回 Skill。"
        "一个回合最多形成一条知识，多个相关结论合并为一条。\n"
        "- 只有可复用、可验证、属于 ClawPro/OpenClaw 企业知识且不含凭据或个人信息时才写入；"
        "寒暄、一次性操作、未验证推测、重复知识和敏感内容必须跳过。\n"
        "- 先检索已有知识；达到正式知识门槛才按 `topics/<topic>/<slug>--<短 ID>.md` "
        "分类写入，domain 只保留为元数据，"
        "证据不足、重复或不确定时直接跳过。时间只写元数据，不使用日期目录。\n"
        "- 需要写入时，只能通过 "
        f"`{PUBLISH_SCRIPT}` 发布到 `harness-knowledge-store`；"
        "不得切换或修改当前业务工作树。\n"
        "- 沉淀过程保持静默，不重复输出刚才给用户的答复。处理完成后正常结束；"
        "下一次 Stop 会因 `stop_hook_active` 自动放行，避免循环。"
    )


def pending_feedback_reason(payload: dict[str, Any]) -> str:
    """Return the existing FEC follow-up reason without emitting another hook result."""
    session_id = fec.sanitize_id(str(payload.get("session_id") or "unknown-session"))
    skills_path = fec.SESSIONS_DIR / f"{session_id}.jsonl"
    turns_path = fec.SESSIONS_DIR / f"{session_id}.turns.jsonl"
    cursor_path = fec.SESSIONS_DIR / f"{session_id}.cursor"

    used_skills = fec.read_skills(skills_path)
    if not used_skills:
        return ""
    turns = fec.read_turns(turns_path)
    cursor = fec.read_cursor(cursor_path)
    if len(turns) <= cursor:
        return ""

    pending = turns[cursor:]
    fec.write_cursor(cursor_path, len(turns))
    return fec.build_reason(used_skills, pending, session_id)


def self_test() -> int:
    score, signal_reasons = knowledge_signal(
        "可以，升级知识沉淀 Hook。",
        "已完成 Stop Hook 升级，并通过自测。实现文件为 .codebuddy/hooks/test.py。"
        "这项结论适用于 ClawPro 本地 Agent 知识库召回流程。",
    )
    assert score >= 4
    low_score, _ = knowledge_signal("你好", "你好，有什么可以帮你？")
    assert low_score == 0
    reason = build_reason(
        "session-1",
        "/tmp/transcript.jsonl",
        score,
        signal_reasons,
    )
    assert SKILL_NAME in reason
    assert PUBLISH_SCRIPT in reason
    assert "harness-knowledge-store" in reason
    assert "最多形成一条知识" in reason
    assert "预筛得分" in reason
    assert callable(fec.build_reason)
    with tempfile.TemporaryDirectory(prefix="clawpro-stop-self-test-") as temporary:
        transcript = Path(temporary) / "transcript.jsonl"
        transcript.write_text(
            "\n".join(
                [
                    json.dumps(
                        {
                            "type": "message",
                            "role": "user",
                            "content": "可以，升级知识沉淀 Hook。",
                        },
                        ensure_ascii=False,
                    ),
                    json.dumps(
                        {
                            "type": "message",
                            "role": "assistant",
                            "content": (
                                "已完成 ClawPro Stop Hook 升级，并通过自测。"
                                "实现文件为 .codebuddy/hooks/test.py，"
                                "适用于本地 Agent 知识库召回流程。"
                            ),
                        },
                        ensure_ascii=False,
                    ),
                ]
            )
            + "\n",
            encoding="utf-8",
        )
        user_text, assistant_text = latest_complete_turn(read_messages(transcript))
        parsed_score, _ = knowledge_signal(user_text, assistant_text)
        assert parsed_score >= 4
        original_sessions_dir = fec.SESSIONS_DIR
        try:
            fec.SESSIONS_DIR = Path(temporary) / "feedback-sessions"
            fec.SESSIONS_DIR.mkdir()
            (fec.SESSIONS_DIR / "session-1.jsonl").write_text(
                '{"skill":"requirement-writer"}\n',
                encoding="utf-8",
            )
            (fec.SESSIONS_DIR / "session-1.turns.jsonl").write_text(
                '{"ts":"2026-01-01T00:00:00Z","q":"这个需求漏了一条规则"}\n',
                encoding="utf-8",
            )
            fec_reason = pending_feedback_reason({"session_id": "session-1"})
            assert "FEC" in fec_reason
            assert pending_feedback_reason({"session_id": "session-1"}) == ""
        finally:
            fec.SESSIONS_DIR = original_sessions_dir
    print("clawpro_knowledge_deposit_stop self-test passed")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--self-test", action="store_true")
    args, _ = parser.parse_known_args()
    if args.self_test:
        return self_test()

    payload = load_hook_input()
    event = str(payload.get("hook_event_name") or "")
    if event and event != "Stop":
        emit_allow()
        return 0

    # Official recursion guard: the Agent has already been resumed by a Stop
    # hook and has finished the requested follow-up work.
    if bool(payload.get("stop_hook_active")):
        emit_allow()
        return 0

    reasons: list[str] = []
    session_id = str(payload.get("session_id") or "").strip()
    transcript_path = str(payload.get("transcript_path") or "").strip()
    if session_id and transcript_path:
        transcript = Path(transcript_path).expanduser()
        if transcript.is_file():
            user_text, assistant_text = latest_complete_turn(read_messages(transcript))
            score, signal_reasons = knowledge_signal(user_text, assistant_text)
            if score >= 4:
                reasons.append(
                    build_reason(
                        session_id,
                        str(transcript),
                        score,
                        signal_reasons,
                    )
                )

    try:
        feedback_reason = pending_feedback_reason(payload)
        if feedback_reason:
            reasons.append(feedback_reason)
    except Exception:
        # Preserve the original feedback gate's fail-open behavior.
        pass

    if not reasons:
        emit_allow()
        return 0
    emit_block("\n\n".join(reasons))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

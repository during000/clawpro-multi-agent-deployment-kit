#!/usr/bin/env python3
"""Stop hook：回合级精准收口（方案 A2）。

对应用户原始需求里的 "stop hook 在会话末尾统一处理反馈、避免多轮判断反馈结束的困难"，
但按官方 hook 语义修正为**回合级**实现：Stop 是「主 agent 每完成一次响应（每一轮回复
结束）」触发的回合级事件（一个会话可多次），并非「整个会话仅一次」——真正会话级的
`SessionEnd` 又无法把 AI 拉回判定。因此这里做「每轮回复结束、按反馈轮增量收口」。

方案 A2（关键词预筛后再拉 AI）的两道节流门槛，避免"无关对话也拉 AI"：
- 门槛①「本会话用过 Skill」：读 skill 调用清单 `.sessions/<sid>.jsonl`，为空直接放行。
  → 与 Skill 完全无关的对话 100% 不介入，零 AI 往返。
- 门槛②「有命中反馈信号的轮」：读反馈轮清单 `.sessions/<sid>.turns.jsonl`
  （由 UserPromptSubmit 侧 `feedback_gate.py` 做廉价关键词预筛后落盘）。
  仅当「命中反馈的轮数 > 已收口游标」时才拉回 AI。
  → 用过 Skill 但用户没给任何疑似反馈的轮，也不拉 AI（"谢谢/继续"这类不产生往返）。

核心机制（均为 CodeBuddy 官方 Stop hook 能力）：
- `stop_hook_active == true`：官方防死循环——上次 block 后 AI 干完活再次触发 Stop，直接放行。
- `stop_hook_active == false`：两道门槛都通过时，`continue:false + reason` 把 reason 注入回
  【当前仍存活的 AI】，让它对本会话「命中反馈但尚未收口」的那些用户轮逐个做 FEC 判定、
  A/B 落盘（落盘脚本自身幂等，若实时线已落盘则 AI 自行跳过）。

跨「停止周期」幂等：`<sid>.cursor` 记录「反馈轮已收口到第几条」。
- 仅当 反馈轮总数 > cursor 时才拉回（有新增的命中反馈轮未收口）；
- 拉回同时把 cursor 推进到当前反馈轮总数，避免同一批反馈轮被重复拉回。
  游标基准是「反馈轮数」而非「skill 调用数」——修正了旧版「产物刚生成、尚无反馈时
  就被过早推进游标，导致后续真实反馈漏记」的缺陷。

边界（诚实告知，机制固有）：
- 只能用 Stop、不能用 SessionEnd 做「判定型收口」：SessionEnd 无法阻止结束、不能把 AI
  拉回，只能清理/记日志。
- 若进程被强杀/崩溃，Stop 可能不触发 → 那次会漏。这是所有 stop-hook 方案的共同边界，
  由 UserPromptSubmit 实时线部分弥补。

设计约束：零外部依赖（仅标准库）；出任何错都安全放行（continue:true），绝不卡死会话。
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from typing import cast


HOOKS_DIR = Path(__file__).resolve().parent            # .codebuddy/hooks
CODEBUDDY_ROOT = HOOKS_DIR.parent                      # .codebuddy
SESSIONS_DIR = CODEBUDDY_ROOT / "feedback" / ".sessions"

SAFE_ID_RE = re.compile(r"[^A-Za-z0-9._-]")

RECORD_SCRIPT = ".codebuddy/skills/feedback/scripts/record_feedback.py"


def emit_allow() -> None:
    print(json.dumps({"continue": True, "suppressOutput": True}, ensure_ascii=False))


def emit_block(reason: str) -> None:
    print(json.dumps({"continue": False, "reason": reason}, ensure_ascii=False))


def sanitize_id(raw: str) -> str:
    cleaned = SAFE_ID_RE.sub("_", raw).strip("._-")
    return cleaned or "unknown-session"


def read_skills(path: Path) -> list[str]:
    """读本会话 Skill 调用清单，返回去重保序的 skill 名列表（门槛①依据）。"""
    seen: set[str] = set()
    skills: list[str] = []
    try:
        for line in path.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line:
                continue
            try:
                obj = cast(object, json.loads(line))
            except Exception:
                continue
            if isinstance(obj, dict):
                d = cast(dict[object, object], obj)
                name = str(d.get("skill") or "unknown-skill")
                if name not in seen:
                    seen.add(name)
                    skills.append(name)
    except Exception:
        pass
    return skills


def read_turns(path: Path) -> list[dict[str, str]]:
    """读本会话「命中反馈信号的用户轮」清单（门槛②依据 + 收口游标基准）。"""
    turns: list[dict[str, str]] = []
    try:
        for line in path.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line:
                continue
            try:
                obj = cast(object, json.loads(line))
            except Exception:
                continue
            if isinstance(obj, dict):
                d = cast(dict[object, object], obj)
                turns.append(
                    {
                        "ts": str(d.get("ts") or ""),
                        "q": str(d.get("q") or ""),
                    }
                )
    except Exception:
        pass
    return turns


def read_cursor(path: Path) -> int:
    try:
        return int(path.read_text(encoding="utf-8").strip() or "0")
    except Exception:
        return 0


def write_cursor(path: Path, value: int) -> None:
    try:
        path.write_text(str(value), encoding="utf-8")
    except Exception:
        pass


def build_reason(
    used_skills: list[str], pending_turns: list[dict[str, str]], session_id: str
) -> str:
    skills_str = "、".join(used_skills) or "（未知）"
    lines: list[str] = []
    for i, t in enumerate(pending_turns, start=1):
        q = t.get("q") or ""
        lines.append(f"  {i}. [{t.get('ts', '')}] {q}")
    turns_block = "\n".join(lines)
    return (
        "【回合级 FEC 收口 · A2】本会话用过 Skill：" + skills_str + "。\n"
        "以下用户输入命中了反馈信号、疑似是对 Skill 产物的反馈，请逐条自查是否已完成 FEC 收口：\n"
        + turns_block + "\n"
        "对每一条按 .codebuddy/skills/feedback/ 规范做 FEC 2×2 判定：\n"
        "① 象限 A（系统性 Skill 逻辑缺陷）/ B（隐性知识规则漏洞）→ 调用 "
        + RECORD_SCRIPT + " 落盘（幂等去重、追加不新建，若本轮实时线已落盘则直接跳过）；\n"
        "② 象限 C（纯业务微调）/ D（噪声）→ 跳过，不生成文件。\n"
        "元原则：不要靠「用户是否情绪不满」判断，而要看「问题是否系统性、会否在后续同类生成中复发」；"
        "平静提出、但暴露了会复发默认缺陷的新需求，也属于象限 A。\n"
        "落盘时可传 --session-id " + session_id + " 与 --offset（对应本会话第几条反馈）以保证幂等。\n"
        "处理完即可正常结束（届时会因防死循环机制直接放行）。"
    )


def main() -> int:
    try:
        raw = cast(object, json.load(sys.stdin))
    except Exception:
        emit_allow()
        return 0

    if not isinstance(raw, dict):
        emit_allow()
        return 0

    data = cast(dict[object, object], raw)

    # 官方防死循环：上一次 block 后 AI 干完再次触发 Stop → 直接放行
    if bool(data.get("stop_hook_active")):
        emit_allow()
        return 0

    session_id = sanitize_id(str(data.get("session_id") or "unknown-session"))
    skills_path = SESSIONS_DIR / f"{session_id}.jsonl"
    turns_path = SESSIONS_DIR / f"{session_id}.turns.jsonl"
    cursor_path = SESSIONS_DIR / f"{session_id}.cursor"

    # 门槛①：本会话没用过任何 Skill → 放行（A2 节流：只有用过 Skill 才可能需要收口）
    used_skills = read_skills(skills_path)
    if not used_skills:
        emit_allow()
        return 0

    # 门槛②：没有命中反馈信号的用户轮 → 放行（A2 关键词预筛节流：无疑似反馈就不拉 AI）
    turns = read_turns(turns_path)
    total = len(turns)
    cursor = read_cursor(cursor_path)
    if total <= cursor:
        emit_allow()
        return 0

    pending = turns[cursor:]

    # 推进 cursor（按反馈轮数），避免同一批反馈轮被重复拉回；实时线由 UserPromptSubmit 补充
    write_cursor(cursor_path, total)

    emit_block(build_reason(used_skills, pending, session_id))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

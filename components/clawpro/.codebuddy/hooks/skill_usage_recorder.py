#!/usr/bin/env python3
"""PostToolUse hook：确定性记录每次 Skill 调用（零 LLM、全自动）。

对应用户原始需求里的 "post-use hook（每次使用 Skill 后触发）"。

职责（只做这一件事，不做任何智力判定）：
- 在 `PostToolUse` + matcher `use_skill` 事件上触发；
- 从 stdin 的 hook JSON 里取 `session_id` 与 `tool_input`（含被调用的 skill 名）；
- 把「谁、什么时候、用了哪个 Skill」确定性追加一行到本会话调用清单：
      .codebuddy/feedback/.sessions/<session_id>.jsonl
- 这份清单供 Stop hook 在会话结束时统一收口用（判断本会话用过哪些 Skill）。

设计约束：
- 非阻断：永远放行（PostToolUse 无需拦截），出任何错都静默放行。
- 零外部依赖：仅使用 Python 标准库。
- 独立：不 import、不引用 campaign / feedback 其它逻辑。
- 契约：从 stdin 读 hook JSON，向 stdout 输出放行 JSON。
"""

from __future__ import annotations

import datetime as _dt
import json
import re
import sys
from pathlib import Path
from typing import cast


HOOKS_DIR = Path(__file__).resolve().parent            # .codebuddy/hooks
CODEBUDDY_ROOT = HOOKS_DIR.parent                      # .codebuddy
SESSIONS_DIR = CODEBUDDY_ROOT / "feedback" / ".sessions"

# session_id 直接用于文件名，做白名单清洗防路径穿越
SAFE_ID_RE = re.compile(r"[^A-Za-z0-9._-]")


def emit_allow() -> None:
    """PostToolUse 永远放行，不产生噪声输出。"""
    print(json.dumps({"continue": True, "suppressOutput": True}, ensure_ascii=False))


def sanitize_id(raw: str) -> str:
    cleaned = SAFE_ID_RE.sub("_", raw).strip("._-")
    return cleaned or "unknown-session"


def extract_skill_name(tool_input: object) -> str:
    """从 use_skill 的入参里取 skill 名（参数为 command）。"""
    if isinstance(tool_input, dict):
        ti = cast(dict[object, object], tool_input)
        for k in ("command", "skill", "name"):
            v = ti.get(k)
            if isinstance(v, str) and v.strip():
                return v.strip()
    if isinstance(tool_input, str) and tool_input.strip():
        return tool_input.strip()
    return "unknown-skill"


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

    # 只处理 use_skill 的 PostToolUse；其它一律放行（matcher 已限定，这里做二次防御）
    tool_name = str(data.get("tool_name") or "")
    if tool_name and tool_name != "use_skill":
        emit_allow()
        return 0

    session_id = sanitize_id(str(data.get("session_id") or "unknown-session"))
    skill_name = extract_skill_name(data.get("tool_input"))

    try:
        SESSIONS_DIR.mkdir(parents=True, exist_ok=True)
        record = {
            "ts": _dt.datetime.now().isoformat(timespec="seconds"),
            "skill": skill_name,
            "tool": tool_name or "use_skill",
        }
        line = json.dumps(record, ensure_ascii=False)
        with (SESSIONS_DIR / f"{session_id}.jsonl").open("a", encoding="utf-8") as fh:
            fh.write(line + "\n")
    except Exception:
        # 记录失败绝不阻断工具流程
        pass

    emit_allow()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

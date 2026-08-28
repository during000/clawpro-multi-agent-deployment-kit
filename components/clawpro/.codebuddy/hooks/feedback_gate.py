#!/usr/bin/env python3
"""Feedback gate hook (independent of campaign/release gate).

一级预筛：在 UserPromptSubmit 事件上识别"用户是否在对某个 Skill 的产物表达反馈
（纠错/不满/缺漏/冗余/新需求/赞赏）"。命中则注入 additionalContext，提示主 AI 在
本轮同步执行 feedback skill 的 FEC 治理流程。

设计约束：
- 非阻断：只注入提示，绝不 deny 任何操作（与 campaign 的提交拦截无关）。
- 零外部依赖：仅使用 Python 标准库。
- 独立：不 import、不引用 campaign / release gate 的任何逻辑与文件。
- 契约：从 stdin 读 hook JSON，向 stdout 输出 hook JSON。
"""

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


# 反馈信号：显性关键词 + 否定/纠错 + 缺漏/冗余 + 赞赏（优秀案例）
# 与 skills/feedback/SKILL.md「触发信号识别」保持一致。
FEEDBACK_SIGNAL_RE = re.compile(
    "|".join(
        [
            # 显性反馈词
            r"反馈",
            r"\bfeedback\b",
            # 否定 / 纠错
            r"这个?不对",
            r"不符合预期",
            r"(?:有|存在)问题",
            r"应该(?:改成|改为|是|用)",
            r"不(?:需要|用|要)(?:这段|这个|这部分|这块)",
            r"(?:改|弄|理解|搞)错了?",
            r"重(?:新|做)一(?:下|遍)",
            # 缺漏 / 冗余
            r"漏了",
            r"少了",
            r"太(?:啰嗦|冗长|繁琐|复杂|简略|简单|多)",
            r"删(?:掉|除)",
            r"多余",
            # 赞赏（优秀案例，值得沉淀）
            r"(?:效果|做得|写得|这个).{0,8}(?:很好|不错|很棒|棒|到位)",
            r"记录一下",
        ]
    ),
    re.IGNORECASE,
)

ASK_FEEDBACK_CONTEXT = "".join(
    [
        "检测到用户可能在对某个 Skill 的产物表达反馈（纠错 / 不满 / 缺漏 / 冗余 / 新需求 / 赞赏）。",
        "请在本轮同步执行 feedback skill 的 FEC 治理流程，严禁「先改内容、回头再补反馈」：",
        "① 先按用户意见修改产物；",
        "② 对本次反馈做 FEC 2×2 象限判定（判定 / 落盘 / 模板一律以 .codebuddy/skills/feedback/ 为权威）；",
        "③ 若判定为象限 A/B，在同一轮调用确定性落盘脚本 ",
        ".codebuddy/skills/feedback/scripts/record_feedback.py",
        "（传入 --skill/--scene-en/--scene-cn/--quadrant/--quote/--rca-error/--rca-root/--action，",
        "由脚本保证幂等去重、追加不新建、自动建目录、按模板渲染）；",
        "④ 象限 C（纯业务微调）/ D（噪声）不生成任何文件（脚本自身也会拒绝 C/D 落盘）。",
        "另注意元原则：不要靠「用户是否情绪不满」判断，而要看「问题是否系统性、会不会在后续同类生成中复发」，",
        "平静提出的新需求若暴露了会复发的默认缺陷，也属于象限 A，必须记录。",
    ]
)


def emit(payload: dict[str, object]) -> None:
    print(json.dumps(payload, ensure_ascii=False))


def allow() -> None:
    """放行且不产生任何提示（无反馈信号时）。"""
    emit({"continue": True, "suppressOutput": True})


def sanitize_id(raw: str) -> str:
    cleaned = SAFE_ID_RE.sub("_", raw).strip("._-")
    return cleaned or "unknown-session"


def record_feedback_turn(session_id: str, prompt: str) -> None:
    """A2 兜底线：命中反馈信号时，把这一轮用户输入追加到本会话「反馈轮清单」
    `.codebuddy/feedback/.sessions/<session_id>.turns.jsonl`，供 Stop hook 在会话末尾
    按「反馈轮」（而非「skill 调用数」）做兜底收口——这是 A2「关键词预筛后再拉 AI」的
    廉价预筛环节：预筛在天然持有 prompt 的 UserPromptSubmit 完成，Stop 只读结果。

    纯确定性、零 LLM；出任何错静默跳过，绝不影响实时提醒与放行主流程。"""
    sid = sanitize_id(session_id or "unknown-session")
    try:
        SESSIONS_DIR.mkdir(parents=True, exist_ok=True)
        record = {
            "ts": _dt.datetime.now().isoformat(timespec="seconds"),
            "hit": True,
            "q": prompt.strip().replace("\n", " ")[:80],
        }
        line = json.dumps(record, ensure_ascii=False)
        with (SESSIONS_DIR / f"{sid}.turns.jsonl").open("a", encoding="utf-8") as fh:
            fh.write(line + "\n")
    except Exception:
        # 预筛落盘失败绝不影响实时提醒与放行
        pass


def main() -> int:
    try:
        raw_data = cast(object, json.load(sys.stdin))
    except Exception:
        allow()
        return 0

    if not isinstance(raw_data, dict):
        allow()
        return 0

    data = cast(dict[object, object], raw_data)
    event = data.get("hook_event_name")

    # 仅在用户提交 prompt 时做一级预筛；其他事件一律放行。
    if event == "UserPromptSubmit":
        prompt = str(data.get("prompt") or "")
        if prompt and FEEDBACK_SIGNAL_RE.search(prompt):
            # A2 兜底线：命中反馈信号即把这一轮落盘为「反馈轮」，供 Stop hook 会话末尾按反馈轮收口
            record_feedback_turn(str(data.get("session_id") or ""), prompt)
            emit(
                {
                    "continue": True,
                    "hookSpecificOutput": {
                        "hookEventName": "UserPromptSubmit",
                        "additionalContext": ASK_FEEDBACK_CONTEXT,
                    },
                }
            )
            return 0

    allow()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

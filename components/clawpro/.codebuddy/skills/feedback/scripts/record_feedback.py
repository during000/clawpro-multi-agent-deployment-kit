#!/usr/bin/env python3
"""确定性落盘脚本：把一条已判定好的 FEC 反馈写入标准反馈单。

设计定位（与 campaign 的 record_campaign.py 同构）：
- 本脚本【不做智力判定】。象限 A/B/C/D、RCA、优化行动项由主 AI 判定后，
  以结构化参数传入；脚本只负责【可靠落盘】：校验 → 幂等去重 → 建目录 →
  按模板渲染 → 追加不新建。
- 幂等：以 session_id + processed_offset（或内容指纹）为去重键，
  同一段交互重复调用不会重复写入。
- 追加不新建：同一 skill + 同一场景（同一目标文件）已存在时，
  仅追加"补充记录"段落，不覆盖、不新建文件。
- 噪声拦截：象限 C / D 一律不落盘（与 SKILL.md 第五节一致）。
- 零外部依赖：仅使用 Python 标准库；不 import、不引用 campaign 任何逻辑。

契约：命令行参数入，退出码出。成功 0，参数/校验错误经 argparse.error -> 2。
"""

from __future__ import annotations

import argparse
import datetime as _dt
import hashlib
import re
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent          # skills/feedback/scripts
SKILL_ROOT = SCRIPT_DIR.parent                        # skills/feedback
CODEBUDDY_ROOT = SKILL_ROOT.parent.parent             # .codebuddy
DEFAULT_FEEDBACK_ROOT = CODEBUDDY_ROOT / "feedback"

# 象限 -> 反馈单模板里对应的勾选文案（与 references/feedback-template.md 一致）
QUADRANT_LINES = {
    "A": "**【象限 A：系统性 Skill 逻辑缺陷】** (价值：⭐⭐⭐⭐⭐)",
    "B": "**【象限 B：隐性知识/规则漏洞】** (价值：⭐⭐⭐⭐⭐)",
    "C": "**【象限 C：纯业务内容微调】** (价值：⭐ - 仅作记录，无需改 Skill)",
    "D": "**【象限 D：彻底的无价值噪声】** (价值：❌ - 请删除本文件)",
}
QUADRANT_DESC = {
    "A": "格式冗余、废话铺垫过多、写了前端组件/API 等代码细节、或编造了无根据的 UI 尺寸。",
    "B": "对业务模型（如跨所有者权限）、双端对称交互、或底座资源生命周期的理解存在严重漏洞。",
    "C": "本次任务的业务决策临时微调（如\"本期先不做对话修复了\"）。",
    "D": "拼写纠错、偶发输入错误、无实质建议的抱怨。",
}
# RCA 根因清单（key -> 文案），与模板保持一致
RCA_ROOTS = {
    "missing-blank": "**信息缺失未留白**：输入未提及该参数，AI 开启了\"自动填充数值\"的默认幻觉。",
    "base-gap": "**底座知识断层**：AI 不知道平台底座的真实生命周期/限制。",
    "role-shift": "**角色定位偏移**：思维没立足于\"资深 PM 跨所有者全局运维\"视角，降级为普通助理。",
    "template-rigid": "**模板固化**：模板过于死板，无法按需求复杂度动态裁剪模块。",
}

KEYS_MARKER_RE = re.compile(r"<!--\s*fec-keys:\s*(?P<keys>.*?)\s*-->", re.DOTALL)
SLUG_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")


def require_text(value: str, field: str, parser: argparse.ArgumentParser) -> str:
    if not isinstance(value, str) or not value.strip():
        parser.error(f"{field} 必须是非空字符串")
    return value.strip()


def validate_date(value: str, parser: argparse.ArgumentParser) -> str:
    try:
        _dt.date.fromisoformat(value)
    except ValueError:
        parser.error(f"--date 必须是 YYYY-MM-DD 格式：{value}")
    return value


def resolve_target(args: argparse.Namespace, parser: argparse.ArgumentParser) -> Path:
    if args.feedback_file.strip():
        return Path(args.feedback_file).expanduser().resolve()
    root = (
        Path(args.feedback_root).expanduser().resolve()
        if args.feedback_root.strip()
        else DEFAULT_FEEDBACK_ROOT
    )
    filename = f"{args.date}-{args.scene_en}.md"
    return (root / args.skill / filename).resolve()


def compute_key(args: argparse.Namespace, quotes: list[str]) -> str:
    """去重键：优先 session_id:offset；否则用内容指纹兜底。"""
    if args.session_id.strip() and args.offset.strip():
        return f"{args.session_id.strip()}:{args.offset.strip()}"
    digest = hashlib.sha1(
        ("\n".join(quotes) + "|" + args.rca_error).encode("utf-8")
    ).hexdigest()[:12]
    return f"fp:{digest}"


def parse_existing_keys(text: str) -> set[str]:
    match = KEYS_MARKER_RE.search(text)
    if not match:
        return set()
    return {k for k in match.group("keys").split() if k}


def render_quotes(quotes: list[str]) -> str:
    return "\n".join(f'  > "{q}"' for q in quotes)


def render_new_file(args: argparse.Namespace, quotes: list[str], key: str) -> str:
    lines: list[str] = [
        "# Skill 反馈记录 & 提炼分析",
        "",
        f"<!-- fec-keys: {key} -->",
        "",
        "## 一、 基本信息",
        f"- **Skill 名称**：{args.skill}",
        f"- **应用场景**：{args.scene_cn}",
        f"- **记录时间**：{args.date}",
        "- **用户反馈原文**：",
        render_quotes(quotes),
        "",
        "================================================================",
        "",
        "## 二、 FEC 判定与归类（核心筛选！）",
    ]
    for q in ("A", "B", "C", "D"):
        mark = "x" if q == args.quadrant else " "
        lines.append(f"- [{mark}] {QUADRANT_LINES[q]}")
        lines.append(f"  *判定依据*：{QUADRANT_DESC[q]}")
    lines += [
        "",
        "================================================================",
        "",
        "## 三、 根因追溯 (RCA) 与问题识别",
        f"- **Skill 犯了什么错**：{args.rca_error}",
        "- **系统性缺陷根因**：",
    ]
    for rkey, rline in RCA_ROOTS.items():
        mark = "x" if rkey in args.rca_root else " "
        lines.append(f"  - [{mark}] {rline}")
    lines += [
        "",
        "================================================================",
        "",
        "## 四、 转化为 Skill 优化策略（SOP）",
        "> 先锁定用户要解决的真问题/目的，再写手段；手段须符合交付物真实约束。",
        "",
    ]
    for i, action in enumerate(args.action, start=1):
        lines.append(f"- **【优化行动项 {i}】**：{action}")
    lines.append("")
    return "\n".join(lines)


def render_append(args: argparse.Namespace, quotes: list[str]) -> str:
    lines = [
        "",
        "================================================================",
        f"## 追加记录（{args.date}）",
        "- **补充原话**：",
        render_quotes(quotes),
    ]
    if args.rca_error:
        lines.append(f"- **补充 RCA**：{args.rca_error}")
    if args.action:
        lines.append("- **补充优化项**：")
        for action in args.action:
            lines.append(f"  - {action}")
    lines.append("")
    return "\n".join(lines)


def update_keys_marker(text: str, key: str) -> str:
    def _sub(match: re.Match[str]) -> str:
        keys = [k for k in match.group("keys").split() if k]
        if key not in keys:
            keys.append(key)
        return f"<!-- fec-keys: {' '.join(keys)} -->"

    if KEYS_MARKER_RE.search(text):
        return KEYS_MARKER_RE.sub(_sub, text, count=1)
    # 老文件没有 marker：插到首个标题后
    return f"<!-- fec-keys: {key} -->\n" + text


def main() -> int:
    parser = argparse.ArgumentParser(
        description="把一条已判定好的 FEC 反馈确定性落盘为标准反馈单（幂等、追加不新建）"
    )
    parser.add_argument("--skill", required=True, help="active skill 目录名，如 requirement-writer")
    parser.add_argument("--scene-en", dest="scene_en", required=True, help="文件名场景英文短名（kebab-case）")
    parser.add_argument("--scene-cn", dest="scene_cn", required=True, help="应用场景中文描述")
    parser.add_argument("--quadrant", required=True, choices=["A", "B", "C", "D"])
    parser.add_argument("--quote", dest="quote", action="append", required=True, help="用户反馈原话，可多次传入")
    parser.add_argument("--rca-error", dest="rca_error", default="", help="Skill 犯了什么错")
    parser.add_argument("--rca-root", dest="rca_root", action="append", default=[],
                        choices=list(RCA_ROOTS.keys()), help="根因清单 key，可多次传入")
    parser.add_argument("--action", dest="action", action="append", default=[], help="优化行动项（目的+手段），可多次传入")
    parser.add_argument("--date", default=_dt.date.today().isoformat(), help="记录日期 YYYY-MM-DD，默认今天")
    parser.add_argument("--session-id", dest="session_id", default="", help="幂等去重键的一部分")
    parser.add_argument("--offset", dest="offset", default="", help="processed_offset 幂等游标")
    parser.add_argument("--feedback-root", dest="feedback_root", default="", help="覆盖 .codebuddy/feedback 根目录")
    parser.add_argument("--feedback-file", dest="feedback_file", default="", help="直接指定目标文件（优先级最高）")
    args = parser.parse_args()

    # 基础校验
    require_text(args.skill, "--skill", parser)
    require_text(args.scene_cn, "--scene-cn", parser)
    require_text(args.rca_error or "-", "--rca-error", parser)
    validate_date(args.date, parser)
    if not SLUG_RE.match(args.scene_en):
        parser.error(f"--scene-en 必须是 kebab-case（小写字母/数字/连字符）：{args.scene_en}")
    quotes = [require_text(q, "--quote", parser) for q in args.quote]

    # 噪声拦截：C/D 不落盘（与 SKILL.md 第五节一致）
    if args.quadrant in ("C", "D"):
        print(f"象限 {args.quadrant} 属于{'纯业务微调' if args.quadrant == 'C' else '无价值噪声'}，"
              "按规范不生成任何文件，已跳过。")
        return 0

    if args.quadrant == "A" and not args.action:
        parser.error("象限 A 必须至少提供一个 --action（优化行动项）")

    key = compute_key(args, quotes)
    target = resolve_target(args, parser)

    if target.exists():
        current = target.read_text(encoding="utf-8")
        # 幂等：同一去重键已记录 -> 跳过
        if key in parse_existing_keys(current):
            print(f"幂等跳过：去重键 {key} 已记录于 {target}")
            return 0
        # 追加不新建
        current = update_keys_marker(current, key)
        separator = "" if current.endswith("\n") else "\n"
        target.write_text(current + separator + render_append(args, quotes), encoding="utf-8")
        print(f"已追加反馈到现有场景文件：{target}")
        return 0

    # 新建（自动建父目录）
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(render_new_file(args, quotes, key), encoding="utf-8")
    print(f"已新建反馈单：{target}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""
walkthrough_report_csv.py —— CSV 规范化 + 枚举校验

DESIGN.md §7「红线」要求：不产出脏 CSV，写完必跑 --fix，未知枚举阻断。

对应 walkthrough 实际产出的两份 CSV（v0.8 的 9 列简化版，非 DESIGN §6.1 的 13 列蓝图）：

1) audit-report.csv —— scripts/walkthrough.mjs#findingsToCsv
   列：ruleId, severity, file, line, col, snippet, message, evidence, suggestion

2) design-todo.csv —— scripts/walkthrough.mjs#todosToCsv
   列：冲突类型, 所属页面, 槽位/位置, 问题描述, AI 当前处理, 建议, 展示台对照,
       真实页面参照, 用户裁决

校验项：
- 列数 + 列名严格匹配
- severity ∈ {P0, P1, P2, P3}
- 用户裁决 ∈ {待裁决, 已采纳, 已驳回, 已沟通}
- 冲突类型 ∈ {token-drift, icon-slot, radius, color, shadow, component-drift,
              page-recipe, portable-impl, spec-binding, 待分类, ...}（开放：未知打 WARN，不阻断）
- ruleId 非空、evidence 非空（DESIGN §7 红线：evidence 为空直接阻断）
- line 必须是非负整数（或空字符串）
- file 路径分隔符统一为 /

行为：
- 默认（无 --fix）：只校验，发现问题打印并 exit 1
- 加 --fix：可自动修的就地修，仍有 P0 阻断项打印并 exit 2
- --check-only：只校验，不区分 fix（兼容 CI）
- 多文件传参：每个文件独立校验、独立 exit code 取最大值

用法：
  python3 walkthrough_report_csv.py <csv_path>...           # 校验
  python3 walkthrough_report_csv.py --fix <csv_path>...     # 校验 + 自动修
  python3 walkthrough_report_csv.py --check-only <path>...  # 仅校验（CI 友好别名）
"""
from __future__ import annotations

import argparse
import csv
import re
import sys
from pathlib import Path
from typing import Dict, List, Tuple

# ---------- schema ----------

SCHEMAS: Dict[str, Dict] = {
    "audit-report.csv": {
        "columns": [
            "ruleId",
            "severity",
            "file",
            "line",
            "col",
            "snippet",
            "message",
            "evidence",
            "suggestion",
        ],
        "enums": {
            "severity": {"P0", "P1", "P2", "P3"},
        },
        "required": {"ruleId", "evidence"},  # DESIGN §7 红线
        "intish": {"line", "col"},
    },
    "design-todo.csv": {
        "columns": [
            "冲突类型",
            "所属页面",
            "槽位/位置",
            "问题描述",
            "AI 当前处理",
            "建议",
            "展示台对照",
            "真实页面参照",
            "用户裁决",
        ],
        "enums": {
            "用户裁决": {"待裁决", "已采纳", "已驳回", "已沟通"},
        },
        "required": {"所属页面", "问题描述"},
        "intish": set(),
        # 已知冲突类型（开放枚举，未知打 WARN 不阻断）
        "open_enums": {
            "冲突类型": {
                "token-drift",
                "icon-slot",
                "radius",
                "color",
                "shadow",
                "component-drift",
                "page-recipe",
                "portable-impl",
                "spec-binding",
                "emoji",
                "text-color",
                "surface-nesting",
                "spacing-grouping",
                "external",
                "待分类",
            },
        },
    },
}

PATH_SEP_RE = re.compile(r"\\")  # windows-style → /


# ---------- 校验 ----------


class Issue:
    __slots__ = ("level", "row", "col", "msg")

    def __init__(self, level: str, row: int, col: str, msg: str) -> None:
        self.level = level  # 'ERROR' (阻断) | 'WARN' (可放过) | 'FIXED'
        self.row = row
        self.col = col
        self.msg = msg

    def fmt(self, path: Path) -> str:
        prefix = {"ERROR": "❌", "WARN": "⚠️ ", "FIXED": "✅"}.get(self.level, "  ")
        loc = f"行{self.row}" + (f"·{self.col}" if self.col else "")
        return f"{prefix} {path.name}:{loc}  {self.msg}"


def detect_schema(path: Path) -> Tuple[str, Dict]:
    name = path.name
    if name in SCHEMAS:
        return name, SCHEMAS[name]
    # 兼容带 slug 前缀：audit-report-xxx.csv / design-todo-xxx.csv
    for key, schema in SCHEMAS.items():
        stem = key.rsplit(".", 1)[0]
        if name.startswith(stem):
            return key, schema
    raise ValueError(f"无法识别的 CSV：{name}（仅支持 audit-report.csv / design-todo.csv）")


def validate(path: Path, fix: bool) -> Tuple[List[Issue], List[List[str]] | None]:
    schema_name, schema = detect_schema(path)
    issues: List[Issue] = []

    with path.open("r", encoding="utf-8", newline="") as f:
        reader = csv.reader(f)
        try:
            header = next(reader)
        except StopIteration:
            issues.append(Issue("ERROR", 0, "", "空文件"))
            return issues, None
        rows = list(reader)

    expected = schema["columns"]
    # 列校验
    if header != expected:
        # 列数对得上但顺序/命名错 → 报 ERROR，不自动修（修了会破坏数据）
        missing = [c for c in expected if c not in header]
        extra = [c for c in header if c not in expected]
        if missing or extra:
            issues.append(
                Issue(
                    "ERROR",
                    1,
                    "header",
                    f"列定义不符 schema {schema_name}；缺失={missing}；多余={extra}",
                )
            )
        else:
            issues.append(
                Issue(
                    "ERROR",
                    1,
                    "header",
                    f"列顺序不符 schema {schema_name}；期望 {expected}，实际 {header}",
                )
            )
        # 列不对就不再继续校验 cell（无法对齐索引）
        return issues, None

    # 行级校验
    fixed_rows: List[List[str]] = [header]
    enums = schema.get("enums", {})
    open_enums = schema.get("open_enums", {})
    required = schema.get("required", set())
    intish = schema.get("intish", set())

    col_idx = {c: i for i, c in enumerate(expected)}

    for r_off, row in enumerate(rows):
        row_no = r_off + 2  # +1 header +1 1-indexed
        # 补齐缺失列（CSV 末尾空字段可能被截断）
        if len(row) < len(expected):
            row = row + [""] * (len(expected) - len(row))
            if fix:
                issues.append(Issue("FIXED", row_no, "", f"补齐 {len(expected) - len(row)+0} 个空尾列"))
        elif len(row) > len(expected):
            issues.append(
                Issue(
                    "ERROR",
                    row_no,
                    "",
                    f"列数过多（{len(row)} > {len(expected)}），可能 snippet 含未转义逗号",
                )
            )
            fixed_rows.append(row)
            continue

        # path 分隔符
        for col in ("file", "所属页面"):
            if col in col_idx:
                v = row[col_idx[col]]
                if "\\" in v:
                    new = PATH_SEP_RE.sub("/", v)
                    if fix:
                        row[col_idx[col]] = new
                        issues.append(Issue("FIXED", row_no, col, "windows 路径分隔符 \\ → /"))
                    else:
                        issues.append(Issue("ERROR", row_no, col, f"路径含 \\：{v}"))

        # required
        for col in required:
            if col in col_idx and not row[col_idx[col]].strip():
                issues.append(Issue("ERROR", row_no, col, "必填列为空（红线）"))

        # 严格枚举
        for col, allowed in enums.items():
            if col in col_idx:
                v = row[col_idx[col]].strip()
                if v and v not in allowed:
                    issues.append(
                        Issue(
                            "ERROR",
                            row_no,
                            col,
                            f"未知枚举值 {v!r}，允许 {sorted(allowed)}",
                        )
                    )

        # 开放枚举（未知打 WARN）
        for col, known in open_enums.items():
            if col in col_idx:
                v = row[col_idx[col]].strip()
                if v and v not in known:
                    issues.append(
                        Issue(
                            "WARN",
                            row_no,
                            col,
                            f"未登记冲突类型 {v!r}（可接受，但建议补到 SCHEMAS.open_enums）",
                        )
                    )

        # 整数列
        for col in intish:
            if col in col_idx:
                v = row[col_idx[col]].strip()
                if v == "":
                    continue
                if not v.lstrip("-").isdigit():
                    issues.append(Issue("ERROR", row_no, col, f"应为整数：{v!r}"))

        fixed_rows.append(row)

    if not fix:
        return issues, None
    return issues, fixed_rows


# ---------- main ----------


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("paths", nargs="+", help="audit-report.csv / design-todo.csv 路径，支持多个")
    ap.add_argument("--fix", action="store_true", help="自动修可修项（路径分隔符 / 补齐尾列）")
    ap.add_argument("--check-only", action="store_true", help="仅校验，不写回（CI 用）")
    ap.add_argument("--quiet", action="store_true", help="只打 ERROR/WARN，吃掉 FIXED")
    args = ap.parse_args()

    if args.check_only and args.fix:
        print("--fix 与 --check-only 互斥", file=sys.stderr)
        return 2

    max_exit = 0
    for raw in args.paths:
        path = Path(raw)
        if not path.exists():
            print(f"❌ 文件不存在：{path}", file=sys.stderr)
            max_exit = max(max_exit, 2)
            continue
        try:
            issues, fixed_rows = validate(path, fix=args.fix)
        except ValueError as e:
            print(f"❌ {path.name}：{e}", file=sys.stderr)
            max_exit = max(max_exit, 2)
            continue

        errors = [i for i in issues if i.level == "ERROR"]
        warns = [i for i in issues if i.level == "WARN"]
        fixes = [i for i in issues if i.level == "FIXED"]

        for issue in issues:
            if args.quiet and issue.level == "FIXED":
                continue
            print(issue.fmt(path))

        if args.fix and fixed_rows is not None and fixes:
            with path.open("w", encoding="utf-8", newline="") as f:
                writer = csv.writer(f)
                writer.writerows(fixed_rows)
            print(f"✅ {path.name}：已写回，修了 {len(fixes)} 处")

        summary = f"{path.name}：ERROR={len(errors)} WARN={len(warns)}"
        if args.fix:
            summary += f" FIXED={len(fixes)}"
        print(summary)

        if errors:
            max_exit = max(max_exit, 1 if not any(
                i.col in ("ruleId", "evidence", "header") or "红线" in i.msg
                for i in errors
            ) else 2)

    return max_exit


if __name__ == "__main__":
    sys.exit(main())

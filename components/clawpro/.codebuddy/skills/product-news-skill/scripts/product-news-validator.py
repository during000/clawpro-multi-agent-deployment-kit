#!/usr/bin/env python3
"""product-news.yml 格式校验脚本。

读取 YAML 文件，按 JSON Schema 验证字段合法性，输出校验结果。
可作为 CI 检查步骤或 skill 内部自检使用。

用法:
    python product-news-validator.py <path-to-product-news.yml>
    python product-news-validator.py --schema <path-to-schema.json> <path-to-yaml>
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

try:
    import yaml
except ImportError:
    print("错误: 需要安装 PyYAML。请运行: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

try:
    import jsonschema
    from jsonschema import validate as jsonschema_validate, ValidationError, SchemaError
except ImportError:
    print("错误: 需要安装 jsonschema。请运行: pip install jsonschema", file=sys.stderr)
    sys.exit(1)


# ── 本地校验规则（JSON Schema 不覆盖的语义约束） ──

def check_semantic_rules(data: dict) -> list[str]:
    """执行 JSON Schema 之外的语义校验，返回错误列表。"""
    errors: list[str] = []
    changes: list[dict] = data.get("changes", [])

    # 1. id 重复检测
    seen_ids: set[str] = set()
    for i, change in enumerate(changes):
        cid = change.get("id", "")
        if cid in seen_ids:
            errors.append(f"changes[{i}].id='{cid}' 与前面的条目重复，id 必须全局唯一")
        seen_ids.add(cid)

    # 1.5 source.frontend_mr_id 和 source.mr_id 至少有一个（v3 新增）
    for i, change in enumerate(changes):
        source = change.get("source")
        if source and isinstance(source, dict):
            frontend_mr_id = source.get("frontend_mr_id")
            mr_id = source.get("mr_id")
            if not frontend_mr_id and not mr_id:
                errors.append(
                    f"changes[{i}].source.frontend_mr_id 和 source.mr_id 至少有一个非空，"
                    f"避免 Bot 无法追溯 MR 来源"
                )

    # 2. needs_guide=false 时 guide 应省略（非写空对象）
    for i, change in enumerate(changes):
        needs_guide = change.get("needs_guide", False)
        guide = change.get("guide")
        if not needs_guide and guide is not None:
            if isinstance(guide, dict) and guide:
                errors.append(
                    f"changes[{i}].needs_guide=false 但 guide 不为空，"
                    f"应省略 guide 字段（非写空对象）"
                )

    # 3. needs_guide=true 但 guide 缺失
    for i, change in enumerate(changes):
        if change.get("needs_guide") and change.get("guide") is None:
            errors.append(
                f"changes[{i}].needs_guide=true 但 guide 字段缺失"
            )

    # 4. title 不包含 emoji（仅匹配已知 emoji 区块，避免误匹配中文）
    import re
    emoji_pattern = re.compile(
        "["
        "\U0001F600-\U0001F64F"  # 表情符号 Emoticons
        "\U0001F300-\U0001F5FF"  # 符号和象形文字 Misc Symbols
        "\U0001F680-\U0001F6FF"  # 交通地图 Transport
        "\U0001F1E0-\U0001F1FF"  # 旗帜 Flags
        "\U00002702-\U000027B0"  # 装饰符号 Dingbats
        "\U000024C2-\U000024C3"  # Ⓜ 等圈内字母
        "\U0001F250-\U0001F251"  # 🉐🉑
        "\U0001F004"             # 🀄
        "\U0001F0CF"             # 🃏
        "\U0001F170-\U0001F171"  # 🅰🅱
        "\U0001F17E-\U0001F17F"  # 🅾🅿
        "\U0001F18E"             # 🆎
        "\U0001F191-\U0001F19A"  # 🆑🆒🆓🆔🆕🆖🆗🆘🆙🆚
        "\U0001F201-\U0001F202"  # 🈁🈂
        "\U00002328"             # ⌨
        "\U000023CF"             # ⏏
        "\U000023E9-\U000023F3"  # ⏩⏪⏫⏬⏭⏮⏯⏰⏱⏲⏳
        "\U000023F8-\U000023FA"  # ⏸⏹⏺
        "\U000024C2"             # Ⓜ
        "\U000025AA-\U000025AB"  # ▪▫
        "\U000025B6"             # ▶
        "\U000025C0"             # ◀
        "\U000025FB-\U000025FE"  # ◻◼◽◾
        "\U00002600-\U000027BF"  # 杂项符号（☀-➿），包含变体选择器范围
        "\U00002934-\U00002935"  # ⤴⤵
        "\U00002B05-\U00002B07"  # ⬅⬆⬇
        "\U00002B1B-\U00002B1C"  # ⬛⬜
        "\U00002B50"             # ⭐
        "\U00002B55"             # ⭕
        "\U00003030"             # 〰
        "\U0000303D"             # 〽
        "\U00003297"             # ㊗
        "\U00003299"             # ㊙
        "]+", flags=re.UNICODE
    )
    for i, change in enumerate(changes):
        title = change.get("title", "")
        if emoji_pattern.search(title):
            errors.append(f"changes[{i}].title 包含 emoji，禁止使用 emoji")

    # 5. title 不以动词开头（简要检查，非穷举）
    verb_starts = ("支持", "新增", "添加", "增加", "删除", "移除", "修改", "优化", "修复")
    for i, change in enumerate(changes):
        title = change.get("title", "")
        if any(title.startswith(v) for v in verb_starts):
            errors.append(
                f"changes[{i}].title='{title}' 以动词开头，"
                f"应改为名词性词组结构（主语+动词+宾语），例如「用户管理支持分组功能」"
            )

    # 6. description 以句号结尾
    for i, change in enumerate(changes):
        desc = change.get("description", "")
        if desc and not desc.rstrip().endswith("。"):
            errors.append(
                f"changes[{i}].description 末尾缺少句号「。」"
            )

    return errors


def load_schema(schema_path: Path) -> dict:
    """加载 JSON Schema 文件。"""
    with open(schema_path, "r", encoding="utf-8") as f:
        schema = json.load(f)
    try:
        jsonschema_validate(schema, {"type": "object"})  # 简单自检
    except SchemaError as e:
        print(f"错误: Schema 文件格式无效: {e}", file=sys.stderr)
        sys.exit(1)
    return schema


def load_yaml(yaml_path: Path) -> dict:
    """加载 YAML 文件。"""
    with open(yaml_path, "r", encoding="utf-8") as f:
        try:
            data = yaml.safe_load(f)
        except yaml.YAMLError as e:
            print(f"错误: YAML 解析失败: {e}", file=sys.stderr)
            sys.exit(1)
    if data is None:
        print("错误: YAML 文件为空", file=sys.stderr)
        sys.exit(1)
    if not isinstance(data, dict):
        print("错误: YAML 顶层必须是对象（dict）", file=sys.stderr)
        sys.exit(1)
    return data


def validate(data: dict, schema: dict) -> tuple[bool, list[str], list[str]]:
    """执行完整校验。

    Returns:
        (passed, errors, warnings)
    """
    errors: list[str] = []
    warnings: list[str] = []

    # 1. JSON Schema 校验
    try:
        jsonschema_validate(data, schema)
    except ValidationError as e:
        errors.append(f"[Schema] {e.message}（路径: {'/'.join(str(p) for p in e.absolute_path)}）")

    # 2. 语义规则校验
    semantic_errors = check_semantic_rules(data)
    errors.extend(semantic_errors)

    # 3. 提示信息
    if len(data.get("changes", [])) == 0:
        warnings.append("changes 列表为空，无产品动态条目")

    return len(errors) == 0, errors, warnings


def main() -> int:
    parser = argparse.ArgumentParser(
        description="product-news.yml 格式校验脚本",
    )
    parser.add_argument(
        "yaml_file",
        nargs="?",
        help="product-news.yml 文件路径",
    )
    parser.add_argument(
        "--schema",
        default=None,
        help="JSON Schema 文件路径（默认使用同目录下的 product-news-schema.json）",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="静默模式，仅输出错误信息",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="以 JSON 格式输出结果",
    )
    args = parser.parse_args()

    if not args.yaml_file:
        parser.print_help()
        return 1

    # 定位 Schema 文件
    script_dir = Path(__file__).resolve().parent
    if args.schema:
        schema_path = Path(args.schema)
    else:
        schema_path = script_dir / "product-news-schema.json"

    if not schema_path.exists():
        print(f"错误: Schema 文件不存在: {schema_path}", file=sys.stderr)
        return 1

    yaml_path = Path(args.yaml_file)
    if not yaml_path.exists():
        print(f"错误: YAML 文件不存在: {yaml_path}", file=sys.stderr)
        return 1

    # 执行校验
    schema = load_schema(schema_path)
    data = load_yaml(yaml_path)
    passed, errors, warnings = validate(data, schema)

    if args.json:
        result = {
            "passed": passed,
            "file": str(yaml_path),
            "schema": str(schema_path),
            "entries_count": len(data.get("changes", [])),
            "errors": errors,
            "warnings": warnings,
        }
        print(json.dumps(result, ensure_ascii=False, indent=2))
    else:
        if not args.quiet:
            print(f"文件: {yaml_path}")
            print(f"Schema: {schema_path}")
            print(f"条目数: {len(data.get('changes', []))}")
        if passed:
            print("✅ 校验通过")
        else:
            print("❌ 校验失败")
        for e in errors:
            print(f"   ❌ {e}")
        for w in warnings:
            print(f"   ⚠️  {w}")

    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())

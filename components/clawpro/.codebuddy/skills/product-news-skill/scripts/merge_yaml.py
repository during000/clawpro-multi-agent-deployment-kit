#!/usr/bin/env python3
"""product-news.yml 增量追加/去重脚本。

读取已有 YAML，按 id 去重后追加新条目，保留原有条目不变。
可被 skill 调用，也可独立使用。

用法:
    python merge_yaml.py --input existing.yml --new new_entry.yml --output result.yml
    python merge_yaml.py --input existing.yml --new-json '{"id":"...",...}' --output existing.yml
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Optional

try:
    import yaml
except ImportError:
    print("错误: 需要安装 PyYAML。请运行: pip install pyyaml", file=sys.stderr)
    sys.exit(1)


def load_yaml(path: Path) -> dict:
    """加载 YAML 文件，返回 changes 列表。"""
    if not path.exists():
        return {"changes": []}
    with open(path, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f) or {}
    if not isinstance(data, dict):
        raise ValueError(f"{path} 顶层必须是对象（dict）")
    if "changes" not in data:
        data["changes"] = []
    return data


def save_yaml(data: dict, path: Path) -> None:
    """写入 YAML 文件，保持中文可读性。"""
    class _LiteralStr(str):
        pass

    def _represent_str(dumper, data):
        if "\n" in data:
            return dumper.represent_scalar("tag:yaml.org,2002:str", data, style="|")
        return dumper.represent_scalar("tag:yaml.org,2002:str", data, allow_unicode=True)

    yaml.add_representer(str, _represent_str)
    path.parent.mkdir(parents=True, exist_ok=True)
    # Write manually to preserve comments and structure
    lines = _dump_changes(data)
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def _dump_changes(data: dict) -> list[str]:
    """手动序列化 changes，保持注释和可读格式。"""
    lines = ["# hatchery/.clawpro/product-news.yml",
             "# 由 product-news skill 生成，AutoSync Bot 消费",
             "# 请勿手动编辑此文件",
             ""]
    changes = data.get("changes", [])
    if not changes:
        lines.append("changes: []")
        return lines
    lines.append("changes:")
    for change in changes:
        lines.append(f"  - id: \"{change.get('id', '')}\"")
        lines.append(f"    title: \"{change.get('title', '')}\"")
        lines.append(f"    type: \"{change.get('type', '')}\"")
        lines.append(f"    date: \"{change.get('date', '')}\"")
        lines.append(f"    endpoint: \"{change.get('endpoint', '')}\"")
        desc = change.get("description", "")
        if "\n" in desc:
            lines.append(f"    description: |")
            for dline in desc.split("\n"):
                lines.append(f"      {dline}")
        else:
            lines.append(f"    description: \"{desc}\"")
        if change.get("version"):
            lines.append(f"    version: \"{change['version']}\"")
        if change.get("sort_order") is not None:
            lines.append(f"    sort_order: {change['sort_order']}")
        source = change.get("source")
        if source:
            lines.append(f"    source:")
            if source.get("type"):
                lines.append(f"      type: \"{source['type']}\"")
            if source.get("frontend_mr_id"):
                lines.append(f"      frontend_mr_id: \"{source['frontend_mr_id']}\"")
            if source.get("mr_id"):
                lines.append(f"      mr_id: \"{source['mr_id']}\"")
            if source.get("commit"):
                lines.append(f"      commit: \"{source['commit']}\"")
            if source.get("author_gongfeng_id"):
                lines.append(f"      author_gongfeng_id: \"{source['author_gongfeng_id']}\"")
        campaign = change.get("related_campaign")
        if campaign:
            lines.append(f"    related_campaign:")
            if campaign.get("update_id"):
                lines.append(f"      update_id: \"{campaign['update_id']}\"")
        needs_guide = change.get("needs_guide", False)
        lines.append(f"    needs_guide: {str(needs_guide).lower()}")
        guide = change.get("guide")
        if needs_guide and guide:
            lines.append(f"    guide:")
            if guide.get("doc_type"):
                lines.append(f"      doc_type: \"{guide['doc_type']}\"")
            if guide.get("feature_name"):
                lines.append(f"      feature_name: \"{guide['feature_name']}\"")
            if guide.get("feature_url"):
                lines.append(f"      feature_url: \"{guide['feature_url']}\"")
            if guide.get("endpoint"):
                lines.append(f"      endpoint: \"{guide['endpoint']}\"")
        auto_publish = change.get("auto_publish", True)
        lines.append(f"    auto_publish: {str(auto_publish).lower()}")
        display_components = change.get("display_components", {})
        lines.append("    display_components:")
        for component_name in ("banner", "floating_window"):
            component = display_components.get(component_name, {})
            enabled = bool(component.get("enabled", False))
            lines.append(f"      {component_name}:")
            lines.append(f"        enabled: {str(enabled).lower()}")
            if enabled:
                duration_days = component.get("duration_days", 14)
                lines.append(f"        duration_days: {duration_days}")
        lines.append("")
    # Remove trailing empty line
    if lines and lines[-1] == "":
        lines.pop()
    return lines


def merge_change(existing: dict, new_entry: dict) -> tuple[dict, str]:
    """将新条目合并到已有数据。

    Args:
        existing: 已有的 product-news 数据 {"changes": [...]}
        new_entry: 新条目 {"id":..., "title":..., ...}

    Returns:
        (merged_data, action)
        action: "created" | "updated" | "skipped"
    """
    changes = existing.get("changes", [])
    new_id = new_entry.get("id")
    if not new_id:
        return existing, "skipped"

    # 按 id 查找是否已存在
    for i, entry in enumerate(changes):
        if entry.get("id") == new_id:
            # 已存在：完整内容相同才跳过。组件开关、时长或其他字段变化也必须更新。
            if entry == new_entry:
                return existing, "skipped"
            # 更新
            changes[i] = new_entry
            existing["changes"] = changes
            return existing, "updated"

    # 不存在：追加
    changes.append(new_entry)
    existing["changes"] = changes
    return existing, "created"


def merge_batch(existing: dict, new_entries: list[dict]) -> tuple[dict, list[str]]:
    """批量合并多个新条目。

    Returns:
        (merged_data, actions)  # actions: ["created", "updated", "skipped", ...]
    """
    actions: list[str] = []
    for entry in new_entries:
        existing, action = merge_change(existing, entry)
        actions.append(action)
    return existing, actions


def main() -> int:
    parser = argparse.ArgumentParser(
        description="product-news.yml 增量追加/去重脚本",
    )
    parser.add_argument(
        "--input",
        required=True,
        help="已有 product-news.yml 文件路径",
    )
    parser.add_argument(
        "--new",
        help="包含新条目的 YAML 文件路径（与 --new-json 二选一）",
    )
    parser.add_argument(
        "--new-json",
        help="新条目的 JSON 字符串（与 --new 二选一）",
    )
    parser.add_argument(
        "--output",
        required=True,
        help="输出文件路径",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="仅展示将要执行的变更，不实际写入",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="静默模式",
    )
    args = parser.parse_args()

    if not args.new and not args.new_json:
        print("错误: 必须提供 --new 或 --new-json", file=sys.stderr)
        return 1
    if args.new and args.new_json:
        print("错误: --new 和 --new-json 不能同时使用", file=sys.stderr)
        return 1

    # 加载已有数据
    input_path = Path(args.input)
    existing = load_yaml(input_path)
    original_count = len(existing.get("changes", []))

    # 解析新条目
    if args.new:
        new_data = load_yaml(Path(args.new))
        new_entries = new_data.get("changes", [])
    else:
        try:
            entry = json.loads(args.new_json)
            new_entries = [entry]
        except json.JSONDecodeError as e:
            print(f"错误: --new-json 不是合法的 JSON: {e}", file=sys.stderr)
            return 1

    if not new_entries:
        if not args.quiet:
            print("没有新条目需要合并")
        return 0

    # 执行合并
    merged, actions = merge_batch(existing, new_entries)
    final_count = len(merged.get("changes", []))

    # 统计
    stats = {
        "created": actions.count("created"),
        "updated": actions.count("updated"),
        "skipped": actions.count("skipped"),
    }

    if not args.quiet:
        print(f"已有条目: {original_count}")
        print(f"新增: {stats['created']}，更新: {stats['updated']}，跳过: {stats['skipped']}")
        print(f"最终条目: {final_count}")

    if args.dry_run:
        if not args.quiet:
            print("\n[DRY RUN] 未写入文件")
        return 0

    # 写入
    output_path = Path(args.output)
    save_yaml(merged, output_path)
    if not args.quiet:
        print(f"已写入: {output_path}")

    return 0


if __name__ == "__main__":
    sys.exit(main())

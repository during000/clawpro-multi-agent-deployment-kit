#!/usr/bin/env python3
"""把同一上线批次的一个或多个更新提醒组件追加到 .clawpro/campaign.yaml。"""

from __future__ import annotations

import argparse
import datetime as dt
import getpass
import json
import re
import subprocess
import sys
from pathlib import Path


SKILL_ROOT = Path(__file__).resolve().parent.parent
DEFAULTS_FILE = SKILL_ROOT / "references" / "component_duration_defaults.json"
FORBIDDEN_COMPONENTS = {"GuideAdminNotify", "GuideUpdateBar"}
CAMPAIGN_HEADER = (
    "# AI 合并保护：Vibe Coding 或处理代码冲突时，必须按 campaign_id 和 "
    "component_id 合并并保留不同产品开发产生的组件记录；"
    "不得用任一分支的完整文件覆盖另一分支。\n"
)


def yaml_string(value: str) -> str:
    return json.dumps(value, ensure_ascii=False)


def find_git_root(start: Path) -> Path | None:
    try:
        result = subprocess.run(
            ["git", "-C", str(start), "rev-parse", "--show-toplevel"],
            check=False,
            capture_output=True,
            text=True,
        )
    except FileNotFoundError:
        return None
    if result.returncode != 0 or not result.stdout.strip():
        return None
    return Path(result.stdout.strip()).resolve()


def find_existing_campaign(start: Path) -> Path | None:
    for directory in [start, *start.parents]:
        candidate = directory / ".clawpro" / "campaign.yaml"
        if candidate.exists():
            return candidate.resolve()
    return None


def resolve_campaign_file(
    campaign_file: str,
    project_root: str,
    parser: argparse.ArgumentParser,
) -> Path:
    if campaign_file.strip():
        return Path(campaign_file).expanduser().resolve()
    if project_root.strip():
        root = Path(project_root).expanduser().resolve()
        if not root.is_dir():
            parser.error(f"目标产品仓库目录不存在：{root}")
        return root / ".clawpro" / "campaign.yaml"

    cwd = Path.cwd().resolve()
    git_root = find_git_root(cwd)
    if git_root:
        return git_root / ".clawpro" / "campaign.yaml"
    existing = find_existing_campaign(cwd)
    if existing:
        return existing
    parser.error(
        "无法确定目标产品仓库根目录；请传入 --project-root 或 --campaign-file，"
        "不得按 skill 所在位置猜测"
    )


def load_defaults(parser: argparse.ArgumentParser) -> dict[str, dict[str, object]]:
    try:
        data = json.loads(DEFAULTS_FILE.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        parser.error(f"无法读取组件默认时长：{exc}")
    if not isinstance(data, dict):
        parser.error("组件默认时长配置必须是对象")
    return data


def require_text(value: object, field: str, parser: argparse.ArgumentParser) -> str:
    if not isinstance(value, str) or not value.strip():
        parser.error(f"组件字段 {field} 必须是非空字符串")
    return value.strip()


def parse_component(
    raw: str,
    defaults: dict[str, dict[str, object]],
    parser: argparse.ArgumentParser,
) -> dict[str, object]:
    try:
        component = json.loads(raw)
    except json.JSONDecodeError as exc:
        parser.error(f"--component-json 不是有效 JSON：{exc}")
    if not isinstance(component, dict):
        parser.error("--component-json 必须是 JSON 对象")

    component_id = require_text(component.get("component_id"), "component_id", parser)
    component_name = require_text(
        component.get("component_name"), "component_name", parser
    )
    if component_name in FORBIDDEN_COMPONENTS:
        parser.error(f"{component_name} 不由 update-awareness 开发或登记")
    if component.get("duration_confirmed") is not True:
        parser.error(
            f"组件 {component_id} 的存在时长尚未确认；"
            "Product Review Plan 确认后必须传入 duration_confirmed=true"
        )

    copy = component.get("copy")
    if not isinstance(copy, list) or not copy:
        parser.error(f"组件 {component_id} 的 copy 必须是非空列表")
    normalized_copy = []
    for item in copy:
        if not isinstance(item, dict):
            parser.error(f"组件 {component_id} 的 copy 条目必须是对象")
        normalized_copy.append({
            "slot": require_text(item.get("slot"), "copy.slot", parser),
            "text": require_text(item.get("text"), "copy.text", parser),
        })

    mount = component.get("mount")
    if not isinstance(mount, dict):
        parser.error(f"组件 {component_id} 的 mount 必须是对象")
    normalized_mount = {
        key: require_text(mount.get(key), f"mount.{key}", parser)
        for key in ("page", "target", "placement")
    }

    code_paths = component.get("code_paths")
    if not isinstance(code_paths, list) or not code_paths:
        parser.error(f"组件 {component_id} 必须提供至少一个 code_paths")
    normalized_paths = []
    for path_value in code_paths:
        path_text = require_text(path_value, "code_paths", parser)
        if Path(path_text).is_absolute():
            parser.error(f"code_paths 必须使用仓库相对路径：{path_text}")
        normalized_paths.append(path_text)

    duration_days = component.get("duration_days")
    default = defaults.get(component_name)
    used_default = duration_days is None
    if duration_days is not None:
        if not isinstance(duration_days, int) or isinstance(duration_days, bool) or duration_days < 1:
            parser.error(f"组件 {component_id} 的 duration_days 必须是正整数")
        duration = {"mode": "fixed_days", "days": duration_days}
    elif default and default.get("mode") == "fixed_days":
        duration = {"mode": "fixed_days", "days": default.get("days")}
    elif default and default.get("mode") == "permanent":
        duration = {"mode": "permanent"}
    else:
        parser.error(
            f"组件 {component_name} 没有可计算的默认时长；"
            "请在 --component-json 中提供 duration_days"
        )

    return {
        "component_id": component_id,
        "component_name": component_name,
        "copy": normalized_copy,
        "mount": normalized_mount,
        "code_paths": normalized_paths,
        "duration": duration,
        "used_default": used_default,
    }


def campaign_exists(text: str, campaign_id: str) -> bool:
    pattern = rf'^\s*-?\s*campaign_id:\s*{re.escape(yaml_string(campaign_id))}\s*$'
    return re.search(pattern, text, flags=re.MULTILINE) is not None


def render_campaign(
    campaign_id: str,
    launched_on: str,
    current_user_id: str,
    components: list[dict[str, object]],
) -> str:
    lines = [
        f"  - campaign_id: {yaml_string(campaign_id)}",
        f"    launched_on: {yaml_string(launched_on)}",
        f"    current_user_id: {yaml_string(current_user_id)}",
        "    components:",
    ]
    for component in components:
        lines.extend([
            f"      - component_id: {yaml_string(component['component_id'])}",
            f"        component_name: {yaml_string(component['component_name'])}",
            "        copy:",
        ])
        for item in component["copy"]:
            lines.extend([
                f"          - slot: {yaml_string(item['slot'])}",
                f"            text: {yaml_string(item['text'])}",
            ])
        mount = component["mount"]
        lines.extend([
            "        mount:",
            f"          page: {yaml_string(mount['page'])}",
            f"          target: {yaml_string(mount['target'])}",
            f"          placement: {yaml_string(mount['placement'])}",
            "        code_paths:",
        ])
        for code_path in component["code_paths"]:
            lines.append(f"          - {yaml_string(code_path)}")
        duration = component["duration"]
        lines.extend([
            "        duration:",
            f"          mode: {yaml_string(duration['mode'])}",
        ])
        if duration["mode"] == "fixed_days":
            lines.append(f"          days: {duration['days']}")
        lines.append('        state: "active"')
    return "\n".join(lines) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(
        description="登记同一上线批次实际开发完成的一个或多个更新提醒组件"
    )
    parser.add_argument("--campaign-file", default="")
    parser.add_argument("--project-root", default="")
    parser.add_argument("--campaign-id", required=True)
    parser.add_argument(
        "--launched-on",
        default=dt.date.today().isoformat(),
        help="组件确认、植入并通过校验的日期，格式 YYYY-MM-DD；默认今天",
    )
    parser.add_argument(
        "--component-json",
        action="append",
        required=True,
        help="组件 JSON；同一批次可重复传入",
    )
    parser.add_argument("--current-user-id", default=getpass.getuser())
    args = parser.parse_args()

    campaign_id = args.campaign_id.strip()
    launched_on = args.launched_on.strip()
    current_user_id = args.current_user_id.strip()
    if not campaign_id:
        parser.error("--campaign-id 不能为空")
    if not current_user_id:
        parser.error("--current-user-id 不能为空")
    try:
        parsed_launched_on = dt.date.fromisoformat(launched_on)
    except ValueError:
        parser.error("--launched-on 必须使用有效的 YYYY-MM-DD 日期")
    if parsed_launched_on.isoformat() != launched_on:
        parser.error("--launched-on 必须使用 YYYY-MM-DD 格式")

    defaults = load_defaults(parser)
    components = [
        parse_component(raw, defaults, parser) for raw in args.component_json
    ]
    component_ids = [component["component_id"] for component in components]
    if len(component_ids) != len(set(component_ids)):
        parser.error("同一 Campaign 内 component_id 不得重复")

    campaign_file = resolve_campaign_file(
        args.campaign_file, args.project_root, parser
    )
    if not campaign_file.exists():
        if (
            campaign_file.parent.name == ".clawpro"
            and campaign_file.parent.parent.is_dir()
        ):
            campaign_file.parent.mkdir(exist_ok=True)
        elif not campaign_file.parent.is_dir():
            parser.error(f"campaign 目标目录不存在：{campaign_file.parent}")
        campaign_file.write_text(
            CAMPAIGN_HEADER + "schema_version: 3\ncampaigns: []\n", encoding="utf-8"
        )
    current = campaign_file.read_text(encoding="utf-8")
    version_match = re.search(
        r"^schema_version:\s*(\d+)\s*$", current, flags=re.MULTILINE
    )
    version = int(version_match.group(1)) if version_match else None
    if version in {1, 2}:
        is_empty = (
            re.search(r"^campaigns:\s*\[\]\s*$", current, flags=re.MULTILINE)
            and not re.search(r"^\s*-\s*campaign_id:", current, flags=re.MULTILINE)
        )
        if not is_empty:
            parser.error(
                ".clawpro/campaign.yaml 含有旧版记录；"
                "请先为每个 Campaign 补充 launched_on 并迁移到 schema_version 3"
            )
        current = CAMPAIGN_HEADER + "schema_version: 3\ncampaigns: []\n"
    elif version != 3:
        parser.error(".clawpro/campaign.yaml 不是 schema_version 3；请先迁移旧记录")
    if not current.startswith("# AI 合并保护："):
        current = CAMPAIGN_HEADER + current
    if campaign_exists(current, campaign_id):
        parser.error(f"campaign_id 已存在，拒绝重复登记：{campaign_id}")
    if re.search(r"^campaigns:\s*\[\]\s*$", current, flags=re.MULTILINE):
        current = re.sub(
            r"^campaigns:\s*\[\]\s*$",
            "campaigns:",
            current,
            count=1,
            flags=re.MULTILINE,
        )
    elif not re.search(r"^campaigns:\s*$", current, flags=re.MULTILINE):
        parser.error(".clawpro/campaign.yaml 缺少有效的 campaigns 根节点")

    separator = "" if current.endswith("\n") else "\n"
    campaign_file.write_text(
        current
        + separator
        + render_campaign(campaign_id, launched_on, current_user_id, components),
        encoding="utf-8",
    )
    print(f"Campaign 已登记：{campaign_id}")
    print(f"上线日期 launched_on：{launched_on}")
    print(
        "请检查 current_user_id 是否为负责本次组件下线的产品经理："
        f"{current_user_id}"
    )
    for component in components:
        if component["used_default"]:
            duration = component["duration"]
            description = (
                f"{duration['days']} 天"
                if duration["mode"] == "fixed_days"
                else "长期保留"
            )
            print(
                f"请产品经理检查组件 {component['component_name']} 的默认时长：{description}",
                file=sys.stderr,
            )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

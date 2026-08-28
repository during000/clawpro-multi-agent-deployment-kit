#!/usr/bin/env python3
"""生成版本更新感知的产品审核方案，可选附带技术实现方案。"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
from pathlib import Path


SKILL_ROOT = Path(__file__).resolve().parent.parent
DURATION_DEFAULTS_FILE = SKILL_ROOT / "references" / "component_duration_defaults.json"


def kebab(value: str) -> str:
    value = value.strip().lower()
    value = re.sub(r"[^a-z0-9]+", "-", value)
    return value.strip("-") or "gong-neng"


def parse_additional_scenario(value: str) -> dict:
    parts = [part.strip() for part in value.split("|", 2)]
    if len(parts) != 3 or not all(parts):
        raise argparse.ArgumentTypeError(
            "附加场景格式必须是：层级|场景编号|场景名称"
        )
    return {"layer": parts[0], "scenario_code": parts[1], "scenario_name": parts[2]}


def parse_component_duration(value: str) -> tuple[str, dict]:
    parts = [part.strip() for part in value.split("=", 1)]
    if len(parts) != 2 or not all(parts):
        raise argparse.ArgumentTypeError(
            "组件存在时长格式必须是：组件类型或组件名=天数|permanent"
        )
    key, raw_duration = parts
    if raw_duration in {"permanent", "长期", "长期保留"}:
        return key, {"mode": "permanent"}
    try:
        days = int(raw_duration)
    except ValueError as exc:
        raise argparse.ArgumentTypeError("组件存在时长必须是正整数天数或 permanent") from exc
    if days < 1:
        raise argparse.ArgumentTypeError("组件存在时长天数必须大于 0")
    return key, {"mode": "fixed_days", "days": days}


def load_duration_defaults(parser: argparse.ArgumentParser) -> dict:
    try:
        data = json.loads(DURATION_DEFAULTS_FILE.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        parser.error(f"无法读取组件默认存在时长：{exc}")
    if not isinstance(data, dict):
        parser.error("组件默认存在时长配置必须是对象")
    return data


def build_duration_proposal(component: dict, defaults: dict, overrides: dict) -> dict:
    if not component.get("executable", True):
        return {
            "status": "not_applicable",
            "label": "由交接流程确认",
            "confirmation_required": False,
        }
    if component["type_id"] == "none":
        return {
            "status": "not_applicable",
            "label": "不展示",
            "confirmation_required": False,
        }

    override = overrides.get(component["type"]) or overrides.get(component["component_name"])
    configured = override or defaults.get(component["component_name"], {"mode": "required"})
    mode = configured.get("mode", "required")
    source = "用户指定" if override else "组件默认建议"
    proposal = {
        "mode": mode,
        "source": source,
        "confirmation_required": True,
        "confirmation_status": "awaiting_user_confirmation",
    }
    if mode == "fixed_days":
        proposal["days"] = configured.get("days")
        proposal["label"] = f"{configured.get('days')} 天"
    elif mode == "permanent":
        proposal["label"] = "长期保留"
    else:
        proposal.update({
            "mode": "required",
            "days": None,
            "label": "待确认：请明确存在天数",
            "value_missing": True,
        })
    return proposal


COMPONENT_TYPE_IDS = {
    "导航提示条": "update-bar",
    "New Tag": "new_tag",
    "气泡引导": "point-bubble",
    "页面入口气泡引导": "nav-bubble",
    "页面入口气泡": "nav-bubble",
    "重要操作变更气泡引导": "highlight-bubble",
    "功能入口附近气泡": "point-bubble",
    "进入对应界面时展示功能介绍气泡": "point-bubble",
    "页面 Alert": "page-alert",
    "对应页面 Alert": "page-alert",
    "对应页面内 Alert": "page-alert",
    "侧边栏说明详情": "changelog-drawer",
    "禁用入口说明气泡": "point-bubble",
    "常驻操作引导": "highlight-bubble",
    "强提醒弹窗": "global-modal",
    "确认弹窗": "global-modal",
    "日常更新提示": "module-float",
    "管控端产品动态卡片": "module-float",
    "产品动态抽屉": "product-updates-drawer",
    "对应组件下方规则说明": "field-rule-hint",
    "新名称右侧标注原名称": "previous-name-label",
    "不展示": "none",
}

COMPONENT_NAMES = {
    "导航提示条": "GuideUpdateBar",
    "New Tag": "GuideNewTag",
    "气泡引导": "GuidePointBubble",
    "页面入口气泡引导": "GuideNavBubble",
    "页面入口气泡": "GuideNavBubble",
    "重要操作变更气泡引导": "GuideHighlightBubble",
    "功能入口附近气泡": "GuidePointBubble",
    "进入对应界面时展示功能介绍气泡": "GuidePointBubble",
    "页面 Alert": "占位：PageAlertAdapter",
    "对应页面 Alert": "占位：PageAlertAdapter",
    "对应页面内 Alert": "占位：PageAlertAdapter",
    "侧边栏说明详情": "GuideChangelogDrawer",
    "禁用入口说明气泡": "GuidePointBubble",
    "常驻操作引导": "GuideHighlightBubble",
    "强提醒弹窗": "GuideGlobalModal",
    "确认弹窗": "GuideGlobalModal",
    "日常更新提示": "GuideModuleFloat",
    "管控端产品动态卡片": "GuideAdminNotify",
    "产品动态抽屉": "ProductUpdatesDrawer",
    "对应组件下方规则说明": "占位：FieldRuleHint",
    "新名称右侧标注原名称": "占位：PreviousNameLabel",
    "不展示": "无",
}

COMPONENT_REVIEW_NAMES = {
    "New Tag": "新功能标签",
    "页面 Alert": "页面提醒",
    "对应页面 Alert": "页面提醒",
    "对应页面内 Alert": "页面提醒",
}

REACT_COMPONENT_REVIEW_NAMES = {
    "GuideGlobalModal": "强提醒弹窗",
    "GuideModuleFloat": "用户端全局产品动态浮窗",
    "GuideAdminNotify": "管控端产品动态卡片",
    "GuideNavBubble": "页面入口气泡引导",
    "GuidePointBubble": "功能入口附近气泡",
    "GuideUpdateBar": "导航提示条",
    "GuideChangelogDrawer": "更新记录抽屉",
    "GuideHighlightBubble": "高亮步骤引导",
    "GuideNewTag": "新功能标签",
    "ProductUpdatesDrawer": "产品动态抽屉",
}

ONBOARDING_TYPE_IDS = {
    "global-modal",
    "module-float",
    "nav-bubble",
    "point-bubble",
    "update-bar",
    "changelog-drawer",
    "highlight-bubble",
    "new_tag",
    "product-updates-drawer",
}

PROJECT_COMPONENT_TYPE_IDS = {
    "page-alert",
    "field-rule-hint",
    "previous-name-label",
    "sidebar-detail",
}

HANDOFF_COMPONENT_NAMES = {"GuideAdminNotify", "GuideUpdateBar"}


def handoff_workflow_gate(component_name: str) -> dict:
    if component_name == "GuideUpdateBar":
        return {
            "status": "handoff_required",
            "next_action": "请先进行 GuideUpdateBar 公告文案生成与审核",
            "development_decision": "不由 update-awareness 决策",
            "scope": "本 skill 仅识别 Banner/公告条需求，不开发组件、关联详情条目或相关配置",
        }
    if component_name == "ProductUpdatesDrawer":
        return {
            "status": "handoff_required",
            "next_action": "随 GuideAdminNotify 一并完成文案审核与后续开发决策",
            "development_decision": "不由 update-awareness 决策",
            "scope": "该 ProductUpdatesDrawer 条目与 GuideAdminNotify 关联，本 skill 不开发或配置",
        }
    return {
        "status": "handoff_required",
        "next_action": "请先进行 GuideAdminNotify 卡片文案生成与审核",
        "development_decision": "不由 update-awareness 决策",
        "scope": "本 skill 仅识别告知需求，不开发组件、详情条目或相关配置",
    }

ADMIN_IMPACT_KEYWORDS = {
    "下发",
    "分发",
    "管理",
    "配置",
    "权限",
    "审批",
    "审核",
    "启用",
    "停用",
    "状态",
    "统计",
    "审计",
    "风控",
    "合规",
    "策略",
    "规则",
    "范围",
    "同步",
    "发布",
    "对话",
    "会话",
    "Agent",
    "agent",
    "智能体",
    "云端",
    "模型",
    "工具调用",
    "外部连接",
    "数据访问",
    "内容生成",
    "故障排查",
    "用户可用",
}

LOW_ADMIN_IMPACT_KEYWORDS = {
    "下载",
    "浏览",
    "查看",
    "帮助",
    "文档",
    "教程",
    "介绍",
    "展示",
    "静态",
    "资源",
}


def infer_admin_impact(feature: str, summary: str, scenario_name: str) -> tuple[bool, str]:
    text = f"{feature} {summary} {scenario_name}"
    has_admin_impact = any(keyword in text for keyword in ADMIN_IMPACT_KEYWORDS)
    has_low_impact = any(keyword in text for keyword in LOW_ADMIN_IMPACT_KEYWORDS)
    if has_admin_impact:
        return True, "自动判断：描述中包含管理、治理、支持、解释或用户侧重要能力等管理员需要知情的信号。"
    if has_low_impact:
        return False, "自动判断：描述更像下载、浏览、帮助或静态展示入口，未发现管理员管理或知情影响。"
    return False, "自动判断：未发现明确的管理员管理或知情影响；如管理员需要支持、解释、治理或排查，请使用 --admin-impact yes。"


def default_design_component(name: str) -> str:
    return COMPONENT_NAMES.get(name, "GuidePointBubble")


def component_source(type_id: str, component_name: str) -> str:
    if type_id == "none" or component_name == "无":
        return "无"
    if type_id in ONBOARDING_TYPE_IDS and "占位" not in component_name:
        return "@/components/onboarding"
    if type_id in PROJECT_COMPONENT_TYPE_IDS:
        return "项目既有组件"
    return "本地占位适配器" if "占位" in component_name else "@/components/onboarding"


def import_path_for(source: str) -> str | None:
    if source == "@/components/onboarding":
        return "@/components/onboarding"
    if source in {"项目既有组件", "设计系统组件", "本地占位适配器"}:
        return "待确认"
    return None


def compact_text(value: str) -> str:
    return re.sub(r"\s+", "", value).strip()


def truncate_text(value: str, limit: int) -> str:
    value = value.strip()
    return value if len(value) <= limit else value[:limit]


def shorten_feature_name(feature: str, limit: int = 10) -> str:
    value = compact_text(feature)
    value = re.sub(r"^(新增|支持|你现在可以使用)", "", value)
    value = re.sub(r"(已更新|可用了)$", "", value)
    shortened = re.sub(r"(能力|功能|页面|入口|模块)$", "", value)
    if len(shortened) >= 2:
        value = shortened
    return value[:limit] or "功能"


def content_slot(name: str, text: str) -> dict:
    return {"name": name, "text": text}


def has_real_route(route: str) -> bool:
    return route.startswith("/") and "占位" not in route


def slot_text(content: dict, name: str, default: str = "") -> str:
    for slot in content.get("slots", []):
        if slot.get("name") == name:
            return slot.get("text", default)
    return default


def build_component_content(
    component_name: str,
    endpoint_type: str,
    feature: str,
    summary: str,
    route: str,
    layer: str,
) -> dict:
    short_feature = shorten_feature_name(feature)
    title = truncate_text(
        f"{short_feature}已更新" if endpoint_type == "管控端" else f"{short_feature}可用了",
        14,
    )
    description = truncate_text(summary, 60)

    if component_name == "GuideUpdateBar":
        return {
            "variant": "文案生成与审核 brief",
            "slots": [
                content_slot("公告正文", "待文案生成与审核（≤60 字）"),
                content_slot("版本", "待确认（可选）"),
                content_slot("详情入口", "待文案生成与审核（可选，≤6 字）"),
            ],
        }
    if component_name == "GuideNewTag":
        return {"variant": "new", "slots": [content_slot("标签", "New")]}
    if component_name == "GuideNavBubble":
        slots = [
            content_slot("标题", title),
            content_slot("说明", description),
        ]
        if has_real_route(route):
            slots.extend([
                content_slot("主操作", "去看看"),
                content_slot("关闭操作", "稍后再说"),
            ])
        return {
            "variant": "导航预览气泡",
            "slots": slots,
        }
    if component_name == "GuidePointBubble":
        return {
            "variant": "纯文本气泡",
            "slots": [content_slot("标题", title), content_slot("说明", description)],
        }
    if component_name == "GuideHighlightBubble":
        return {
            "variant": "步骤说明气泡",
            "slots": [content_slot("步骤标题", title), content_slot("步骤说明", description)],
        }
    if component_name == "GuideGlobalModal":
        return {
            "variant": "单条全局弹窗",
            "slots": [
                content_slot("标题", title),
                content_slot("说明", description),
                content_slot("确认操作", "确认了解"),
            ],
        }
    if component_name == "GuideModuleFloat":
        return {
            "variant": "单条更新浮窗",
            "slots": [
                content_slot("标题", title),
                content_slot("说明", description),
                content_slot("主操作", "立即体验"),
            ],
        }
    if component_name == "GuideAdminNotify":
        return {
            "variant": "文案生成与审核 brief",
            "slots": [
                content_slot("卡片标题", "待文案生成与审核（≤16 字，含端前缀）"),
                content_slot("卡片描述", "待文案生成与审核（建议 ≤30 字）"),
                content_slot("卡片按钮", "待文案生成与审核（建议 ≤6 字）"),
            ],
        }
    if component_name == "GuideChangelogDrawer":
        tag = layer.replace("层", "") if layer in {"结构层", "元素层", "逻辑层", "系统层", "跨端层"} else "元素"
        return {
            "variant": "更新详情条目",
            "slots": [
                content_slot("版本", dt.date.today().isoformat()),
                content_slot("条目 ID", "占位：更新条目 id"),
                content_slot("条目标题", title),
                content_slot("条目描述", description),
                content_slot("层级标签", tag),
                content_slot("日期", dt.date.today().isoformat()),
            ],
        }
    if component_name == "ProductUpdatesDrawer":
        endpoint_label = "管控端" if endpoint_type == "管控端" else "用户端"
        return {
            "variant": "产品动态条目",
            "slots": [
                content_slot("条目 ID", "占位：产品动态条目 id"),
                content_slot("更新类型", "功能上线"),
                content_slot("所属端", endpoint_label),
                content_slot("条目标题", f"{endpoint_label}｜{short_feature}"),
                content_slot("条目描述", description),
                content_slot("日期", dt.date.today().isoformat()),
            ],
        }
    if component_name == "无":
        return {"variant": "无内容", "slots": []}
    return {
        "variant": "项目组件默认内容",
        "slots": [content_slot("标题", title), content_slot("正文", description)],
    }


def resolve_component(name: str, surface_endpoint: str) -> tuple[str, str, list[str]]:
    notes: list[str] = []
    component_name = COMPONENT_NAMES.get(name, "GuidePointBubble")
    type_id = COMPONENT_TYPE_IDS.get(name, "point-bubble")

    if name == "日常更新提示":
        if surface_endpoint == "管控端":
            component_name = "GuideAdminNotify"
            type_id = "module-float"
            notes.append("管控端日常更新提示按组件定义使用 GuideAdminNotify。")
        else:
            component_name = "GuideModuleFloat"
            type_id = "module-float"
            notes.append("该更新需在任一用户端页面跨页面触达，且影响等级达到高/重大，使用全局 GuideModuleFloat。")

    if surface_endpoint == "用户端" and component_name == "GuideNavBubble":
        component_name = "GuidePointBubble"
        type_id = "point-bubble"
        notes.append("GuideNavBubble 按组件契约仅用于管控端，用户端改用目标附近的 GuidePointBubble。")

    if surface_endpoint == "用户端" and component_name == "GuideChangelogDrawer":
        component_name = "占位：SidebarDetailAdapter"
        type_id = "sidebar-detail"
        notes.append("GuideChangelogDrawer 按组件契约仅用于管控端，用户端改用项目已有详情抽屉或最小占位适配器。")

    return type_id, component_name, notes


def mount_defaults(type_id: str) -> tuple[str, str]:
    if type_id == "update-bar":
        return "页面顶部（导航下方）", "全局唯一"
    if type_id == "module-float":
        return "页面右下角固定区域", "全局唯一"
    if type_id == "nav-bubble":
        return "目标导航项右侧", "指定实例"
    if type_id == "point-bubble":
        return "目标元素下方", "首个实例"
    if type_id == "highlight-bubble":
        return "目标元素右侧", "指定实例"
    if type_id == "new_tag":
        return "目标元素行内", "指定实例"
    if type_id in {"changelog-drawer", "product-updates-drawer"}:
        return "页面右侧抽屉", "全局唯一"
    if type_id == "global-modal":
        return "页面中央全局浮层", "全局唯一"
    return "目标区域内", "全局唯一"


def selector_defaults(type_id: str, component_name: str) -> tuple[str | None, str]:
    if type_id == "update-bar":
        return "header.sticky, header[class*='sticky'], main", "必须至少命中 header.sticky 或 main"
    if component_name == "GuideAdminNotify":
        return None, "挂在管控端全局布局的左侧导航产品动态区域，不依赖目标元素定位"
    if component_name == "GuideModuleFloat" or type_id in {
        "global-modal", "changelog-drawer", "product-updates-drawer"
    }:
        return None, "全局固定表面，不依赖目标元素定位"
    if type_id == "none":
        return None, "不展示"
    return "[data-guide='占位']", "执行前替换为稳定唯一的 data-guide 选择器"


def mount_target_for(type_id: str, component_name: str, feature: str) -> str:
    if type_id == "update-bar":
        return "页面导航下方公告区域"
    if component_name == "GuideAdminNotify":
        return "管控端产品动态卡片区域"
    if component_name == "GuideModuleFloat":
        return "用户端全局产品动态浮窗区域"
    if type_id in {"changelog-drawer", "product-updates-drawer"}:
        return "更新详情抽屉区域"
    if type_id == "global-modal":
        return "页面全局层"
    return feature


def trigger_for(type_id: str, component_name: str) -> str:
    if type_id == "update-bar":
        return "进入目标页面且本次更新仍在有效期内时"
    if type_id == "new_tag":
        return "目标入口出现且标签未到下线时间时"
    if type_id in {"nav-bubble", "point-bubble", "highlight-bubble"}:
        return "首次进入目标页面并看到目标元素时"
    if component_name == "GuideAdminNotify":
        return "管理员首次进入产品动态区域且本次更新未读时"
    if component_name == "GuideModuleFloat":
        return "用户登录后进入任一允许的用户端页面，存在未读高影响更新且当前无冲突引导时"
    if type_id in {"changelog-drawer", "product-updates-drawer"}:
        return "用户点击查看详情时"
    if type_id == "global-modal":
        return "首次进入目标页面且满足强提醒条件时"
    if type_id == "none":
        return "不展示"
    return "进入目标页面并看到目标区域时"


def placeholder_allowed_for(source: str, type_id: str) -> bool:
    if source == "@/components/onboarding" or type_id == "none":
        return False
    return type_id in PROJECT_COMPONENT_TYPE_IDS


def review_problem_for(component_type: str) -> str:
    problems = {
        "导航提示条": "避免用户忽略重要更新，并提供统一的详情入口。",
        "New Tag": "帮助用户快速发现新增入口或能力。",
        "页面入口气泡引导": "避免用户找不到新增页面或入口。",
        "页面入口气泡": "避免用户找不到新增页面或入口。",
        "功能入口附近气泡": "帮助用户理解新功能的位置和用途。",
        "重要操作变更气泡引导": "避免用户继续按旧方式完成关键操作。",
        "常驻操作引导": "在关键操作处持续提供必要说明。",
        "强提醒弹窗": "确保用户在继续操作前理解高影响变化。",
        "日常更新提示": "让用户在不打断当前任务的情况下了解更新。",
        "管控端产品动态卡片": "帮助管理员了解需要管理或解释的产品变化。",
        "页面 Alert": "避免用户误解当前页面的重要规则或状态。",
        "对应页面 Alert": "避免用户误解当前页面的重要规则或状态。",
        "对应页面内 Alert": "避免用户误解当前页面的重要规则或状态。",
        "对应组件下方规则说明": "在操作位置解释新规则，减少错误配置。",
        "新名称右侧标注原名称": "帮助用户建立新旧名称之间的对应关系。",
        "侧边栏说明详情": "承接更新的完整背景和操作说明。",
        "产品动态抽屉": "集中承接多条产品更新详情。",
        "不展示": "该变化不会导致用户找不到入口、误解规则或操作失败，无需打扰。",
    }
    return problems.get(component_type, "帮助用户发现并正确理解本次更新。")


def default_behavior_for(type_ids: set[str], primary_type_id: str, today: str, update_id: str) -> dict:
    has_new_tag = "new_tag" in type_ids
    has_update_bar = "update-bar" in type_ids
    has_queue_bubble = bool(type_ids & {"nav-bubble", "point-bubble"})
    new_tag_expires_at = (
        (dt.date.fromisoformat(today) + dt.timedelta(days=14)).isoformat()
        if has_new_tag
        else None
    )
    return {
        "dismissible": not has_update_bar,
        "show_once": not has_update_bar and not has_queue_bubble,
        "max_exposures": 2 if has_queue_bubble else (0 if has_update_bar else 1),
        "cooldown_days": 14 if has_new_tag else 0,
        "starts_at": today,
        "expires_at": None,
        "new_tag_expires_at": new_tag_expires_at,
        "new_tag_remove_on_first_click": has_new_tag,
        "persistence_key": f"onboarding.{primary_type_id}.{update_id}.dismissed",
    }


def component_behavior_for(type_id: str, today: str, update_id: str) -> dict:
    behavior = {
        "dismissible": True,
        "show_once": True,
        "max_exposures": 1,
        "cooldown_days": 0,
        "starts_at": today,
        "expires_at": None,
        "new_tag_expires_at": None,
        "new_tag_remove_on_first_click": False,
        "persistence_key": f"onboarding.{type_id}.{update_id}.dismissed",
    }
    if type_id == "update-bar":
        behavior.update(dismissible=False, show_once=False, max_exposures=0)
    elif type_id in {"nav-bubble", "point-bubble"}:
        behavior.update(show_once=False, max_exposures=2)
    elif type_id == "highlight-bubble":
        behavior.update(show_once=True, max_exposures=1)
    elif type_id == "new_tag":
        behavior.update(
            dismissible=False,
            show_once=False,
            max_exposures=0,
            cooldown_days=14,
            new_tag_expires_at=(
                dt.date.fromisoformat(today) + dt.timedelta(days=14)
            ).isoformat(),
            new_tag_remove_on_first_click=True,
        )
    elif type_id in {"changelog-drawer", "product-updates-drawer"}:
        behavior.update(show_once=False, max_exposures=0)
    elif type_id in {"page-alert", "field-rule-hint", "previous-name-label"}:
        behavior.update(dismissible=False, show_once=False, max_exposures=0)
    elif type_id == "none":
        behavior.update(dismissible=False, show_once=False, max_exposures=0)
    return behavior


def component_key_name(component_name: str, type_id: str) -> str:
    names = {
        "GuideAdminNotify": "admin-notify",
        "GuideModuleFloat": "module-float",
        "GuideNewTag": "new-tag",
        "ProductUpdatesDrawer": "product-updates-drawer",
    }
    return names.get(component_name, type_id.replace("_", "-"))


def visual_asset_requirement(component_name: str, content: dict) -> dict:
    if component_name == "GuideModuleFloat":
        return {
            "required": True,
            "status": "待设计",
            "type": "图片",
            "request": "请联系设计团队进行图片设计",
            "spec": "16:9，推荐 672×376",
            "source_or_placeholder": "待提供；不得使用组件灰色占位块上线",
            "launch_gate": "素材交付并经产品确认前不可上线",
        }
    if component_name == "GuideGlobalModal":
        return {
            "required": True,
            "status": "待设计",
            "type": "图片或视频",
            "request": "需要联系设计出图或提供视频",
            "spec": "图片 1080×608，或经产品确认的视频规格",
            "source_or_placeholder": "待提供；不得使用带尺寸标记的占位画面上线",
            "launch_gate": "素材交付并经产品确认前不可上线",
        }
    if component_name == "GuideNavBubble" and slot_text(content, "预览图"):
        return {
            "required": True, "status": "待设计", "type": "图片",
            "request": "需要联系设计出图", "spec": "约 300×140 展示比例的高清图",
            "source_or_placeholder": "待提供", "launch_gate": "素材交付并确认前不可上线",
        }
    if component_name == "GuidePointBubble" and content.get("variant") in {"图文气泡", "带图推送通知"}:
        return {
            "required": True, "status": "待设计", "type": "图片",
            "request": "需要联系设计出图", "spec": "按所选变体的组件展示比例提供高清图",
            "source_or_placeholder": "待提供；不得使用组件占位图上线",
            "launch_gate": "素材交付并确认前不可上线",
        }
    return {"required": False}


def component_behavior_for_unit(component: dict, today: str, update_id: str, index: int) -> dict:
    behavior = component_behavior_for(component["type_id"], today, update_id)
    key_component = component_key_name(component["component_name"], component["type_id"])
    unit_update_id = f"{update_id}-{index + 1}"
    behavior["persistence_key"] = f"onboarding.{key_component}.{unit_update_id}.dismissed"
    return behavior


def build_component_props(component: dict, route: str) -> dict:
    name = component["component_name"]
    content = component["content"]
    endpoint = "admin" if component["surface_endpoint"] == "管控端" else "tenant"
    content_endpoint = "admin" if component["content_endpoint"] == "管控端" else "tenant"
    title = slot_text(content, "标题") or slot_text(content, "步骤标题")
    description = slot_text(content, "说明") or slot_text(content, "步骤说明")

    if name == "GuideNewTag":
        return {"variant": content["variant"]}
    if name == "GuideUpdateBar":
        props = {"open": "受控状态", "message": slot_text(content, "公告正文")}
        detail_text = slot_text(content, "详情入口")
        if detail_text:
            props.update({"detailText": detail_text, "onDetail": "打开详情承接组件"})
        return props
    if name == "GuidePointBubble":
        return {
            "open": "受控状态", "onClose": "记录关闭并隐藏", "title": title,
            "description": description, "contentVariant": "text-only", "endpoint": endpoint,
        }
    if name == "GuideNavBubble":
        props = {
            "open": "受控状态", "onClose": "记录关闭并隐藏",
            "title": title, "description": description,
        }
        if has_real_route(route):
            props.update({"href": route, "actionText": slot_text(content, "主操作", "去看看")})
        return props
    if name == "GuideHighlightBubble":
        return {
            "open": "受控状态", "onClose": "记录完成并隐藏", "endpoint": endpoint,
            "regions": [{
                "id": component["notification_unit_id"],
                "selector": component["mount"]["selector"],
                "title": title,
                "description": description,
                "bubblePlacement": "right",
            }],
        }
    if name == "GuideGlobalModal":
        return {
            "open": "受控状态", "onClose": "记录知情并隐藏", "variant": "single",
            "slides": [{
                "titleRight": title, "desc": description,
                "imageSrc": "待设计：1080×608 正式素材，未提供前禁止上线",
            }],
            "confirmText": slot_text(content, "确认操作", "确认了解"),
            "onConfirm": "记录确认操作", "endpoint": endpoint,
        }
    if name == "GuideModuleFloat":
        props = {
            "open": "受控状态", "onClose": "记录关闭并隐藏", "variant": "single",
            "items": [{
                "title": title, "description": description,
                "image": "待设计：16:9、推荐 672×376，未提供前禁止上线",
            }],
            "confirmText": slot_text(content, "主操作", "立即体验"),
        }
        props["onConfirm"] = f"进入 {route}" if has_real_route(route) else "执行已确认的主操作"
        return props
    if name == "GuideAdminNotify":
        return {}
    if name == "GuideChangelogDrawer":
        entry = {
            "id": slot_text(content, "条目 ID", "占位：更新条目 id"),
            "title": slot_text(content, "条目标题"),
            "description": slot_text(content, "条目描述"),
            "tag": slot_text(content, "层级标签", "元素"),
            "date": slot_text(content, "日期"),
        }
        if has_real_route(route):
            entry["href"] = route
        return {
            "open": "受控状态", "onClose": "关闭抽屉",
            "versions": [{
                "version": slot_text(content, "版本"),
                "date": slot_text(content, "日期"),
                "entries": [entry],
            }],
        }
    if name == "ProductUpdatesDrawer":
        item = {
            "id": slot_text(content, "条目 ID", component.get("detail_entry_id") or "占位：产品动态条目 id"),
            "type": slot_text(content, "更新类型", "功能上线"),
            "endpoint": content_endpoint,
            "title": slot_text(content, "条目标题"),
            "desc": slot_text(content, "条目描述"),
            "date": slot_text(content, "日期"),
            "recent": True,
        }
        if has_real_route(route):
            item["actionHref"] = route
        return {
            "open": "受控状态", "onClose": "关闭抽屉", "items": [item],
            "highlightIds": [component["detail_entry_id"]] if component.get("detail_entry_id") else [],
        }
    return {}


def build_component(name: str, index: int, route: str, feature: str, surface_endpoint: str) -> dict:
    type_id, component_name, notes = resolve_component(name, surface_endpoint)
    source = component_source(type_id, component_name)
    placement, instance_scope = mount_defaults(type_id)
    if component_name == "GuideAdminNotify":
        placement = "管控端左侧导航底部（用户账号上方）"
    mount_route = route
    if component_name == "GuideModuleFloat":
        mount_route = "用户端全局布局（登录后）"
        placement = "用户端右下角固定区域"
    selector, selector_note = selector_defaults(type_id, component_name)
    detail_component_name = None
    detail_entry_id = None
    if type_id == "update-bar":
        detail_component_name = "GuideChangelogDrawer"
        detail_entry_id = "占位：抽屉高亮条目 id"
    if component_name == "GuideAdminNotify":
        detail_component_name = "ProductUpdatesDrawer"
        detail_entry_id = "占位：产品动态条目 id"
    if type_id == "product-updates-drawer":
        detail_entry_id = "占位：产品动态条目 id"
    component = {
        "type": name,
        "component_name": component_name,
        "type_id": type_id,
        "notification_unit_id": f"notification-{index + 1}",
        "is_primary": index == 0,
        "purpose": review_problem_for(name) if not notes else f"{review_problem_for(name)}{'；'.join(notes)}",
        "trigger": trigger_for(type_id, component_name),
        "mount": {
            "route": mount_route,
            "selector": selector,
            "selector_note": selector_note,
            "placement": placement,
            "target_label": mount_target_for(type_id, component_name, feature),
            "instance_scope": instance_scope,
        },
        "component_source": source,
        "import_path": import_path_for(source),
        "design_status": "占位" if "占位" in component_name else "已接入",
        "placeholder_allowed": placeholder_allowed_for(source, type_id),
        "detail_component_name": detail_component_name,
        "detail_entry_id": detail_entry_id,
        "surface_endpoint": surface_endpoint,
        "content_endpoint": surface_endpoint,
        "destination_route": route,
    }
    if component_name == "GuideModuleFloat":
        component["global_display_constraints"] = {
            "scope": "用户端登录后全局布局",
            "single_instance": True,
            "allowed": "任一普通用户已登录页面",
            "excluded": ["登录页", "首次 onboarding", "全屏/沉浸任务", "GuideGlobalModal 打开期间"],
            "route_change_reexposes": False,
            "state_scope": "跨页面共享未读、曝光、关闭和完成状态",
        }
        component["selection_basis"] = (
            "影响等级为高/重大，更新对普通用户重要且需要跨页面触达；"
            "New Tag 或目标附近气泡不足以避免用户漏看。"
        )
    if component_name == "GuideGlobalModal":
        component["selection_basis"] = (
            "变化影响重大，需要在用户继续操作前强提醒；普通浮窗或页面内提示不足。"
        )
    if component_name in HANDOFF_COMPONENT_NAMES:
        component["executable"] = False
        component["import_path"] = None
        component["detail_entry_id"] = None
        component["mount"]["selector"] = None
        component["mount"]["selector_note"] = "仅记录产品可理解的挂载位置；具体选择器由交接后的开发流程决定"
        component["workflow_gate"] = handoff_workflow_gate(component_name)
    return component


def build_cross_admin_notify(route: str, feature: str, update_id: str) -> dict:
    return {
        "type": "日常更新提示",
        "component_name": "GuideAdminNotify",
        "type_id": "module-float",
        "notification_unit_id": "notification-cross-admin",
        "is_primary": False,
        "purpose": "在管控端告知管理员用户端重要能力，支持管理、治理、解释和故障排查。",
        "trigger": "管理员首次进入管控端且本次产品动态未读时",
        "mount": {
            "route": "管控端全局布局",
            "selector": None,
            "selector_note": "挂在管控端全局布局的左侧导航产品动态区域，不依赖目标元素定位",
            "placement": "管控端左侧导航底部（用户账号上方）",
            "target_label": "左侧导航产品动态卡片区域",
            "instance_scope": "全局唯一",
        },
        "component_source": "@/components/onboarding",
        "import_path": None,
        "design_status": "已接入",
        "placeholder_allowed": False,
        "detail_component_name": "ProductUpdatesDrawer",
        "detail_entry_id": None,
        "surface_endpoint": "管控端",
        "content_endpoint": "用户端",
        "content": {
            "variant": "文案生成与审核 brief",
            "slots": [
                content_slot("卡片标题", "待文案生成与审核（≤16 字，含端前缀）"),
                content_slot("卡片描述", "待文案生成与审核（建议 ≤30 字）"),
                content_slot("卡片按钮", "待文案生成与审核（建议 ≤6 字）"),
            ],
        },
        "executable": False,
        "workflow_gate": {
            "status": "handoff_required",
            "next_action": "请先进行 GuideAdminNotify 卡片文案生成与审核",
            "development_decision": "不由 update-awareness 决策",
            "scope": "本 skill 仅识别告知需求，不开发组件、详情条目或相关配置",
        },
    }


def format_display_strategy(behavior: dict, component: dict) -> str:
    if component.get("workflow_gate", {}).get("status") == "handoff_required":
        return "待文案生成与审核；本 skill 不决策或执行开发"
    behavior = component.get("behavior", behavior)
    type_id = component["type_id"]
    if type_id == "update-bar":
        frequency = "每次进入页面展示"
        dismiss = "不可手动关闭"
    elif type_id in {"nav-bubble", "point-bubble"}:
        frequency = "最多展示 2 次"
        dismiss = "可关闭"
    elif type_id == "highlight-bubble":
        frequency = "每位用户展示一次"
        dismiss = "可关闭"
    elif type_id == "new_tag":
        frequency = "持续展示至首次点击或到期"
        dismiss = "不可手动关闭"
    elif type_id == "module-float":
        if component.get("component_name") == "GuideModuleFloat":
            frequency = "用户端任一允许页面最多展示一次，路由切换不重复"
        else:
            frequency = "每位用户展示一次"
        dismiss = "可关闭"
    elif type_id in {"changelog-drawer", "product-updates-drawer"}:
        frequency = "按需打开"
        dismiss = "可关闭"
    elif type_id == "global-modal":
        frequency = "首次满足强提醒条件时展示"
        dismiss = "可关闭"
    elif type_id == "page-alert":
        frequency = "规则有效期内持续展示"
        dismiss = "按项目 Alert 规则"
    elif type_id in {"field-rule-hint", "previous-name-label"}:
        frequency = "随目标元素持续展示"
        dismiss = "无单独关闭操作"
    elif type_id == "none":
        return "不展示，不注入代码"
    elif behavior.get("max_exposures") == 0:
        frequency = "持续展示"
        dismiss = "可关闭" if behavior.get("dismissible") else "不可手动关闭"
    elif behavior.get("show_once"):
        frequency = "每位用户展示一次"
        dismiss = "可关闭" if behavior.get("dismissible") else "不可手动关闭"
    else:
        frequency = f"最多展示 {behavior.get('max_exposures', 1)} 次"
        dismiss = "可关闭" if behavior.get("dismissible") else "不可手动关闭"
    return f"{frequency}，{dismiss}"


def build_product_review_plan(implementation_plan: dict) -> dict:
    items = []
    for component in implementation_plan["components"]:
        mount = component["mount"]
        target_users = "管理员" if component["surface_endpoint"] == "管控端" else "普通用户"
        review_item = {
            "target_users": [target_users],
            "component": {
                "type_zh": REACT_COMPONENT_REVIEW_NAMES.get(
                    component["component_name"],
                    COMPONENT_REVIEW_NAMES.get(component["type"], component["type"]),
                ),
                "name": component["component_name"],
            },
            "problem": component["purpose"],
            "mount": {
                "page": mount["route"],
                "target": mount["target_label"],
                "placement": mount["placement"],
            },
            "trigger": component["trigger"],
            "content": component["content"],
            "display_strategy": format_display_strategy(implementation_plan["behavior"], component),
            "duration": component["duration"],
        }
        if component.get("selection_basis"):
            review_item["selection_basis"] = component["selection_basis"]
        if component.get("visual_asset", {}).get("required"):
            review_item["visual_asset"] = {
                key: component["visual_asset"][key]
                for key in ["required", "status", "request", "spec", "launch_gate"]
            }
        if component.get("workflow_gate"):
            review_item["workflow_gate"] = component["workflow_gate"]
        items.append(review_item)
    review_target_users = []
    for item in items:
        for target_user in item["target_users"]:
            if target_user not in review_target_users:
                review_target_users.append(target_user)
    handoff_names = [
        item.get("component", {}).get("name")
        for item in items
        if item.get("workflow_gate", {}).get("status") == "handoff_required"
    ]
    has_handoff = bool(handoff_names)
    has_executable_item = any(
        item.get("workflow_gate", {}).get("status") != "handoff_required"
        for item in items
    )
    handoff_label = "、".join(dict.fromkeys(handoff_names))
    missing_duration_names = [
        item["component"]["name"]
        for item in items
        if item.get("duration", {}).get("value_missing")
    ]
    if missing_duration_names:
        confirmation_prompt = (
            f"请先明确以下组件的存在时长：{'、'.join(dict.fromkeys(missing_duration_names))}；"
            "补充后将生成新的方案修订版供确认。"
        )
    elif has_handoff and not has_executable_item:
        confirmation_prompt = (
            f"请先完成 {handoff_label} 的文案生成与审核；"
            "这些组件不由 update-awareness 决策或开发。"
        )
    elif has_handoff:
        confirmation_prompt = (
            f"请连同其他可执行项的存在时长一起审核；确认无误后回复：确认执行方案 {implementation_plan['id']} "
            f"v{implementation_plan['plan_revision']}；{handoff_label} 仍需另行完成文案生成、审核和开发决策。"
        )
    else:
        confirmation_prompt = (
            f"请连同各组件存在时长一起审核；确认无误后回复：确认执行方案 {implementation_plan['id']} "
            f"v{implementation_plan['plan_revision']}"
        )
    return {
        "plan_id": implementation_plan["id"],
        "plan_revision": implementation_plan["plan_revision"],
        "status": implementation_plan["approval"]["status"],
        "update": implementation_plan["change"]["summary"],
        "target_users": review_target_users,
        "items": items,
        "confirmation_prompt": confirmation_prompt,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="生成中文 Product Review Plan 初稿")
    parser.add_argument("--feature", required=True, help="功能或更新名称，建议中文")
    parser.add_argument("--summary", required=True, help="本次变化的中文摘要")
    parser.add_argument("--slug", default="", help="用于 id 的英文或拼音短名")
    parser.add_argument(
        "--endpoint-type", choices=["管控端", "用户端"], default="管控端",
        help="端类型：管控端 或 用户端",
    )
    parser.add_argument("--tone", default="", help="文案风格：专业 或 易懂；默认随端类型推断")
    parser.add_argument(
        "--layer", choices=["结构层", "元素层", "逻辑层", "系统层", "跨端层"],
        default="元素层", help="结构层、元素层、逻辑层、系统层、跨端层",
    )
    parser.add_argument("--scenario-code", default="2.1", help="场景编号，例如 1.1、2.3、3.3")
    parser.add_argument("--scenario-name", default="新增按钮或操作入口", help="中文场景名称")
    parser.add_argument(
        "--impact-level", choices=["低", "中", "高", "重大"], default="中",
        help="用户影响等级；GuideModuleFloat 仅允许高/重大更新",
    )
    parser.add_argument(
        "--additional-scenario",
        action="append",
        default=[],
        type=parse_additional_scenario,
        help="可重复传入附加场景，格式：层级|场景编号|场景名称",
    )
    parser.add_argument(
        "--component-duration",
        action="append",
        default=[],
        type=parse_component_duration,
        help="可重复传入组件存在时长，格式：组件类型或组件名=天数|permanent",
    )
    parser.add_argument("--components", default="导航提示条,功能入口附近气泡", help="逗号分隔的组件组合")
    parser.add_argument("--route", default="/占位路由", help="目标页面路由")
    parser.add_argument("--target-file", default="占位：目标页面文件路径", help="目标页面文件")
    parser.add_argument("--design-component", default="", help="设计组件名；默认取主组件")
    parser.add_argument(
        "--admin-impact",
        choices=["auto", "yes", "no"],
        default="auto",
        help="用户端变化是否需要管理员管理或知情，包括配置、治理、支持、解释、排查等；默认自动判断",
    )
    parser.add_argument("--out", default="", help="输出文件路径；为空时打印到标准输出")
    parser.add_argument(
        "--include-implementation",
        action="store_true",
        help="在默认的产品审核方案后附带完整 Implementation Plan",
    )
    args = parser.parse_args()

    if args.design_component.strip() in HANDOFF_COMPONENT_NAMES:
        parser.error(
            f"{args.design_component.strip()} 不由 update-awareness 进行设计接入或开发决策；"
            "请先完成文案生成与审核，并交由独立流程处理"
        )

    today = dt.date.today().isoformat()
    slug = kebab(args.slug or args.feature)
    update_id = f"update-awareness-{slug}-{today}"
    component_names = [item.strip() for item in args.components.split(",") if item.strip()]
    if not component_names:
        parser.error("--components 至少需要一个组件；无需提醒时传入“不展示”")
    unknown_components = [name for name in component_names if name not in COMPONENT_NAMES]
    if unknown_components:
        parser.error(f"未知组件类型：{', '.join(unknown_components)}")
    endpoint_type = args.endpoint_type.strip() or "管控端"
    if endpoint_type == "用户端":
        admin_only_requests = [
            name for name in component_names
            if name in {"管控端产品动态卡片", "产品动态抽屉"}
        ]
        if admin_only_requests:
            parser.error(
                f"{', '.join(admin_only_requests)} 按组件契约仅能展示在管控端；"
                "请将 --endpoint-type 改为管控端，或改选用户端组件"
            )
    duration_defaults = load_duration_defaults(parser)
    tone = args.tone.strip() or ("专业" if endpoint_type == "管控端" else "易懂")
    audience_roles = ["管理员"] if endpoint_type == "管控端" else ["普通用户"]
    user_value_placeholder = (
        "占位：补充对管理员的管理效率、规则理解或配置价值。"
        if endpoint_type == "管控端"
        else "占位：补充对用户的直接使用价值，避免后台术语。"
    )
    components = [
        build_component(
            name=name,
            index=index,
            route=args.route,
            feature=args.feature,
            surface_endpoint=endpoint_type,
        )
        for index, name in enumerate(component_names)
    ]
    duration_overrides = {}
    valid_duration_keys = set(component_names) | {
        component["component_name"] for component in components
    }
    for key, duration in args.component_duration:
        if key not in valid_duration_keys:
            parser.error(
                f"存在时长指定了当前方案之外的组件：{key}；"
                "请使用 --components 中的中文类型或实际匹配的 React 组件名"
            )
        if key in duration_overrides:
            parser.error(f"组件存在时长重复指定：{key}")
        duration_overrides[key] = duration
    should_add_cross_admin_notice = (
        endpoint_type == "用户端" and any(name != "不展示" for name in component_names)
    )
    inferred_admin_impact, admin_impact_reason = infer_admin_impact(
        args.feature,
        args.summary,
        args.scenario_name,
    )
    has_admin_impact = (
        args.admin_impact == "yes"
        or (args.admin_impact == "auto" and inferred_admin_impact)
    )
    if should_add_cross_admin_notice and has_admin_impact:
        components.append(build_cross_admin_notify(args.route, args.feature, update_id))
    if any(c["component_name"] == "GuideAdminNotify" for c in components):
        for component in components:
            if component["component_name"] != "ProductUpdatesDrawer":
                continue
            component["executable"] = False
            component["import_path"] = None
            component["detail_entry_id"] = None
            component["workflow_gate"] = handoff_workflow_gate("ProductUpdatesDrawer")
    heavy_user_components = [
        component for component in components
        if component["component_name"] == "GuideModuleFloat"
    ]
    if heavy_user_components and args.impact_level not in {"高", "重大"}:
        parser.error(
            "GuideModuleFloat 固定含配图区，仅用于高/重大且需要跨页面触达的用户端更新；"
            "请提高 --impact-level 并准备设计图，或改用 New Tag / 页面入口气泡引导 / 功能入口附近气泡 / 不展示"
        )
    if any(c["component_name"] == "GuideGlobalModal" for c in components) and args.impact_level not in {"高", "重大"}:
        parser.error(
            "GuideGlobalModal 仅用于高/重大影响更新；请提高 --impact-level，或改用更轻的公告、气泡、页面说明"
        )
    for index, component in enumerate(components):
        if "content" not in component:
            component["content"] = build_component_content(
                component["component_name"],
                component["surface_endpoint"],
                args.feature,
                args.summary,
                args.route,
                args.layer,
            )
        if not component.get("executable", True):
            component["behavior"] = {
                "status": "handoff_required",
                "note": "待文案生成与审核；不由本 skill 生成执行生命周期",
            }
            component["props"] = {}
        else:
            component["behavior"] = component_behavior_for_unit(
                component, today, update_id, index
            )
            component["props"] = build_component_props(component, args.route)
        component["visual_asset"] = visual_asset_requirement(
            component["component_name"], component["content"]
        )
        component["duration"] = build_duration_proposal(
            component, duration_defaults, duration_overrides
        )

    module_float_units = [
        component for component in components
        if component["component_name"] == "GuideModuleFloat"
    ]
    if module_float_units:
        for order, component in enumerate(module_float_units):
            component["render_group"] = "tenant-module-float"
            component["render_strategy"] = (
                "单一 GuideModuleFloat 实例；多个通知单元通过 sources 分页合并"
            )
            component["source_order"] = order

    admin_notify_units = [
        component for component in components
        if component["component_name"] == "GuideAdminNotify"
    ]
    if admin_notify_units:
        for component in admin_notify_units:
            component["render_group"] = "admin-product-update-notify"
            component["render_strategy"] = (
                "交接给文案生成、审核与后续开发负责人；本 skill 不生成渲染实现"
            )
    update_bar_units = [
        component for component in components
        if component["component_name"] == "GuideUpdateBar"
    ]
    if update_bar_units:
        for component in update_bar_units:
            component["render_group"] = "admin-update-banner"
            component["render_strategy"] = (
                "交接给公告文案生成、审核与后续开发负责人；本 skill 不生成渲染实现"
            )
    if admin_notify_units and args.design_component.strip() == "ProductUpdatesDrawer":
        parser.error(
            "当前 ProductUpdatesDrawer 与 GuideAdminNotify 关联，不由 update-awareness 进行设计接入；"
            "请交由独立流程处理，或将独立详情抽屉拆分为单独方案"
        )
    handoff_units = [component for component in components if component.get("workflow_gate")]
    blocked_components = []
    if admin_notify_units:
        blocked_components.extend([
            "GuideAdminNotify",
            "GuideAdminNotify 关联的 ProductUpdatesDrawer 条目",
        ])
    if update_bar_units:
        blocked_components.extend([
            "GuideUpdateBar",
            "GuideUpdateBar 关联的 GuideChangelogDrawer 条目",
        ])
    executable_components = [
        component for component in components if component.get("executable", True)
    ]
    primary_component = components[0] if components else {}
    primary_executable_component = executable_components[0] if executable_components else {}
    executable_type_ids = {item["type_id"] for item in executable_components}
    primary_executable_type_id = primary_executable_component.get("type_id", "none")
    design_reference = [
        "update-awareness-skill/references/component_contract.md",
        "update-awareness-skill/references/onboarding_component_spec.md",
        "clawpro-portable-design-skill/references/components.md",
    ]
    if endpoint_type == "管控端":
        design_reference.insert(2, "clawpro-portable-design-skill/SKILL.md")
    else:
        design_reference.insert(2, "clawpro-portable-design-skill/references/tenant.md")
    if executable_components:
        plan_behavior = default_behavior_for(
            executable_type_ids,
            primary_executable_type_id,
            today,
            update_id,
        )
        plan_analytics = {
            "impression_event": "onboarding_impression",
            "click_event": "onboarding_click",
            "dismiss_event": "onboarding_dismiss",
            "properties": {
                "update_id": update_id,
                "endpoint_type": endpoint_type,
                "tone": tone,
                "layer": args.layer,
                "scenario": args.scenario_code,
                "components": [
                    {
                        "type": item["type"],
                        "component_name": item["component_name"],
                        "type_id": item["type_id"],
                        "component_source": item["component_source"],
                        "surface_endpoint": item["surface_endpoint"],
                        "content_endpoint": item["content_endpoint"],
                        "notification_unit_id": item["notification_unit_id"],
                    }
                    for item in executable_components
                ],
            },
        }
        design_component = {
            "source": primary_executable_component.get("component_source"),
            "name": (
                args.design_component.strip()
                or primary_executable_component.get("component_name")
                or default_design_component(component_names[0])
            ),
            "import_path": primary_executable_component.get("import_path"),
            "design_reference": design_reference,
            "props": primary_executable_component.get("props", {}),
        }
        approval_status = "awaiting_confirmation"
        approval_scope = (
            "确认仅覆盖可执行组件的目标用户、组件、解决问题、展示位置、Trigger、"
            "文案、展示策略、存在时长和视觉素材要求；不包含 GuideAdminNotify 或 GuideUpdateBar。"
        )
        open_placeholders = [
            "确认这是管控端还是用户端页面，并校正文案语气。",
            "确认目标仓库已从 @/components/onboarding 统一入口导出可执行组件，并校对真实 props 映射。",
            "确认可执行组件的关闭状态持久化和埋点规范。",
            "确认每个可执行组件的 duration；标记为 required 的组件必须明确具体天数。",
            "若方案使用 GuideModuleFloat，请联系设计团队进行图片设计；若使用 GuideGlobalModal 或其他带图变体，也需确认设计素材已交付。素材未交付时不得使用源码占位图上线。",
        ]
    else:
        plan_behavior = {
            "status": "not_applicable",
            "reason": "仅包含交接项，不生成生命周期或持久化配置。",
        }
        plan_analytics = {
            "status": "not_applicable",
            "reason": "仅包含交接项，不生成埋点事件或参数。",
        }
        design_component = {
            "status": "not_applicable",
            "reason": "GuideAdminNotify 与 GuideUpdateBar 不由 update-awareness 进行设计接入或开发决策。",
        }
        approval_status = "handoff_required"
        approval_scope = "无可执行组件；仅交接文案生成、审核与后续开发决策。"
        open_placeholders = [
            "请先完成交接组件的文案生成与审核；GuideAdminNotify、GuideUpdateBar 及其各自关联详情条目不由本 skill 决策或开发。"
        ]
    if admin_notify_units and executable_components:
        open_placeholders.append(
            "GuideAdminNotify 仅作为交接项；其文案、生命周期、持久化、埋点、导入和开发决策均不在本方案执行范围内。"
        )
    if update_bar_units and executable_components:
        open_placeholders.append(
            "GuideUpdateBar 仅作为交接项；其公告文案、生命周期、持久化、埋点、导入和开发决策均不在本方案执行范围内。"
        )
    scenarios = [
        {
            "layer": args.layer,
            "scenario_code": args.scenario_code,
            "scenario_name": args.scenario_name,
        },
        *args.additional_scenario,
    ]

    plan = {
        "id": update_id,
        "plan_revision": 1,
        "approval": {
            "status": approval_status,
            "approved_plan_id": None,
            "approved_revision": None,
            "approved_by": None,
            "approved_at": None,
            "scope": approval_scope,
        },
        "version": today,
        "product_area": args.route,
        "endpoint": {
            "type": endpoint_type,
            "tone": tone,
        },
        "change": {
            "layer": args.layer,
            "scenario_code": args.scenario_code,
            "scenario_name": args.scenario_name,
            "scenarios": scenarios,
            "summary": args.summary,
            "user_value": user_value_placeholder,
            "impact_level": args.impact_level,
            "evidence": [],
            "confidence": "中",
        },
        "audience": {"roles": audience_roles, "segments": [], "exclusions": []},
        "components": components,
        "content": primary_component.get("content", {"variant": "无内容", "slots": []}),
        "behavior": plan_behavior,
        "analytics": plan_analytics,
        "design_component": design_component,
        "injection": {
            "strategy": "配置注入" if executable_components else "不注入，仅交接",
            "target_file": args.target_file if executable_components else None,
            "notes": [
                "GuideAdminNotify、GuideUpdateBar 及其各自关联详情条目不在本 skill 的注入范围内。"
            ] if handoff_units else [],
        },
        "execution_guard": {
            "blocked_components": blocked_components,
            "enforcement": "hard_block",
            "on_attempt": "拒绝执行并提醒先完成文案生成、审核和独立开发决策",
        } if handoff_units else {"blocked_components": [], "enforcement": "none"},
        "assumptions": [
            admin_impact_reason
        ] if endpoint_type == "用户端" and should_add_cross_admin_notice else [],
        "open_placeholders": open_placeholders,
    }

    output = build_product_review_plan(plan)
    if args.include_implementation:
        output["implementation_plan"] = plan
    text = json.dumps(output, ensure_ascii=False, indent=2)
    if args.out:
        Path(args.out).write_text(text + "\n", encoding="utf-8")
    else:
        print(text)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

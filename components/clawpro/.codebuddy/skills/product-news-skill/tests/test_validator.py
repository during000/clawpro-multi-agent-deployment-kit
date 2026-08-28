#!/usr/bin/env python3
"""product-news-validator 单元测试"""

import importlib.util
import json
import sys
from pathlib import Path

# 通过 importlib 加载带连字符的模块文件
SCRIPT_DIR = Path(__file__).resolve().parent.parent / "scripts"
validator_path = SCRIPT_DIR / "product-news-validator.py"
spec = importlib.util.spec_from_file_location("product_news_validator", validator_path)
validator = importlib.util.module_from_spec(spec)
sys.modules["product_news_validator"] = validator
spec.loader.exec_module(validator)

load_schema = validator.load_schema
load_yaml = validator.load_yaml
validate = validator.validate
check_semantic_rules = validator.check_semantic_rules

merge_path = SCRIPT_DIR / "merge_yaml.py"
merge_spec = importlib.util.spec_from_file_location("merge_yaml", merge_path)
merge_module = importlib.util.module_from_spec(merge_spec)
sys.modules["merge_yaml"] = merge_module
merge_spec.loader.exec_module(merge_module)
merge_change = merge_module.merge_change

FIXTURES = Path(__file__).resolve().parent / "fixtures"
SCHEMA = SCRIPT_DIR / "product-news-schema.json"


def test_load_schema():
    """Schema 文件可正常加载"""
    schema = load_schema(SCHEMA)
    assert "$schema" in schema
    assert "changes" in schema["properties"]


def test_valid_passes():
    """正常 YAML 校验通过"""
    data = load_yaml(FIXTURES / "valid.yml")
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert passed, f"校验应通过，但有错误: {errors}"
    assert len(errors) == 0


def test_invalid_type_blocked():
    """type='bugfix' 被拦截"""
    data = load_yaml(FIXTURES / "invalid_type.yml")
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert not passed, "type='bugfix' 应被拦截"
    assert any("功能上线" in e or "体验优化" in e for e in errors), \
        f"错误信息应提及允许的枚举值，实际: {errors}"


def test_missing_required_blocked():
    """缺少必填字段被拦截"""
    data = load_yaml(FIXTURES / "missing_required.yml")
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert not passed, "缺少 title/description 应被拦截"
    assert any("title" in e for e in errors), f"应包含 title 相关错误，实际: {errors}"


def test_duplicate_id_blocked():
    """重复 id 被语义规则拦截"""
    data = load_yaml(FIXTURES / "duplicate_id.yml")
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert not passed, "重复 id 应被拦截"
    assert any("重复" in e for e in errors), f"错误信息应包含'重复'，实际: {errors}"


def test_emoji_title_blocked():
    """title 含 emoji 被语义规则拦截"""
    data = load_yaml(FIXTURES / "emoji_title.yml")
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert not passed, "emoji title 应被拦截"
    assert any("emoji" in e.lower() for e in errors), \
        f"错误信息应包含 emoji 相关提示，实际: {errors}"


def test_needs_guide_without_guide_blocked():
    """needs_guide=true 但 guide 缺失被拦截"""
    data = load_yaml(FIXTURES / "needs_guide_without_guide.yml")
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert not passed, "needs_guide=true 但无 guide 应被拦截"
    assert any("guide" in e.lower() for e in errors), \
        f"错误信息应提及 guide，实际: {errors}"


def test_verbose_start_blocked():
    """标题以动词开头被语义规则拦截"""
    data = {"changes": [{
        "id": "test-verb-start",
        "title": "支持新增用户管理功能",
        "type": "功能上线",
        "date": "2026-07-05",
        "description": "测试描述。",
    }]}
    sem_errors = check_semantic_rules(data)
    assert any("动词开头" in e for e in sem_errors), \
        f"以'支持'开头的标题应被拦截，实际: {sem_errors}"


def test_description_no_period_warned():
    """description 不以句号结尾被警告"""
    data = {"changes": [{
        "id": "test-no-period",
        "title": "用户管理支持分组功能",
        "type": "功能上线",
        "date": "2026-07-05",
        "description": "测试描述结尾没有句号",
    }]}
    sem_errors = check_semantic_rules(data)
    assert any("句号" in e for e in sem_errors), \
        f"description 无句号应被警告，实际: {sem_errors}"


def test_needs_guide_false_with_guide():
    """needs_guide=false 但 guide 非空被语义规则拦截"""
    data = {"changes": [{
        "id": "test-guide-mismatch",
        "title": "功能上线标题",
        "type": "功能上线",
        "date": "2026-07-05",
        "description": "描述。",
        "needs_guide": False,
        "guide": {"doc_type": "operation_guide", "feature_name": "test"},
    }]}
    sem_errors = check_semantic_rules(data)
    assert any("省略 guide" in e for e in sem_errors), \
        f"needs_guide=false 但 guide 非空应报错，实际: {sem_errors}"


def test_minimal_valid():
    """最简有效条目可通过"""
    data = {"changes": [{
        "id": "a-1",
        "title": "功能完善了",
        "type": "体验优化",
        "date": "2026-07-05",
        "endpoint": "管控端",
        "description": "测试。",
        "display_components": {
            "banner": {"enabled": False},
            "floating_window": {"enabled": False},
        },
    }]}
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert passed, f"最简有效条目应通过，错误: {errors}"


def test_empty_changes_warns():
    """空 changes 列表给出 warning 但仍通过"""
    data = {"changes": []}
    passed, errors, warnings = validate(data, load_schema(SCHEMA))
    assert passed
    assert any("为空" in w for w in warnings), f"应给出空列表 warning，实际: {warnings}"


def test_source_no_mr_id_blocked():
    """v3: source 存在但 frontend_mr_id 和 mr_id 都为空时被语义规则拦截"""
    data = {"changes": [{
        "id": "test-no-mr-id",
        "title": "测试功能",
        "type": "功能上线",
        "date": "2026-07-15",
        "endpoint": "管控端",
        "description": "描述。",
        "display_components": {
            "banner": {"enabled": False},
            "floating_window": {"enabled": False},
        },
        "source": {
            "type": "repo_skill",
            "commit": "abc1234",
            "author_gongfeng_id": "hawkechen",
        },
    }]}
    sem_errors = check_semantic_rules(data)
    assert any("frontend_mr_id" in e and "mr_id" in e for e in sem_errors), \
        f"source 无 mr_id 应被拦截，实际: {sem_errors}"


def test_source_with_frontend_mr_id_only():
    """v3: source 只有 frontend_mr_id 可通过校验"""
    data = {"changes": [{
        "id": "test-frontend-mr-only",
        "title": "测试功能",
        "type": "功能上线",
        "date": "2026-07-15",
        "endpoint": "管控端",
        "description": "描述。",
        "display_components": {
            "banner": {"enabled": False},
            "floating_window": {"enabled": False},
        },
        "source": {
            "type": "repo_skill",
            "frontend_mr_id": "!1234",
            "commit": "abc1234",
            "author_gongfeng_id": "hawkechen",
        },
    }]}
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert passed, f"source 只有 frontend_mr_id 应通过，错误: {errors}"


def test_auto_publish_default_true():
    """v3: auto_publish 默认值为 true（schema 层校验）"""
    schema = load_schema(SCHEMA)
    auto_publish_def = schema["$defs"]["ChangeEntry"]["properties"]["auto_publish"]
    assert auto_publish_def.get("default") is True, \
        f"v3 auto_publish 默认值应为 true，实际: {auto_publish_def.get('default')}"


def test_invalid_endpoint_blocked():
    """endpoint 只能是管控端或用户端"""
    data = {"changes": [{
        "id": "test-invalid-endpoint",
        "title": "测试功能",
        "type": "功能上线",
        "date": "2026-07-17",
        "endpoint": "控制台",
        "description": "描述。",
        "display_components": {
            "banner": {"enabled": False},
            "floating_window": {"enabled": False},
        },
    }]}
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert not passed, "非法 endpoint 应被拦截"
    assert any("管控端" in e or "用户端" in e for e in errors), errors


def test_display_components_requires_boolean():
    """Banner 和浮窗 enabled 必须为 boolean"""
    data = {"changes": [{
        "id": "test-invalid-component-switch",
        "title": "测试功能",
        "type": "功能上线",
        "date": "2026-07-17",
        "endpoint": "用户端",
        "description": "描述。",
        "display_components": {
            "banner": {"enabled": "yes"},
            "floating_window": {"enabled": False},
        },
    }]}
    passed, _, _ = validate(data, load_schema(SCHEMA))
    assert not passed, "非 boolean 的组件开关应被拦截"


def test_enabled_component_requires_duration():
    """组件开启时必须记录 duration_days"""
    data = {"changes": [{
        "id": "test-duration-required",
        "title": "测试功能",
        "type": "功能上线",
        "date": "2026-07-17",
        "endpoint": "用户端",
        "description": "描述。",
        "display_components": {
            "banner": {"enabled": True},
            "floating_window": {"enabled": False},
        },
    }]}
    passed, _, _ = validate(data, load_schema(SCHEMA))
    assert not passed, "组件开启但缺少 duration_days 应被拦截"


def test_disabled_component_omits_duration():
    """组件关闭时省略 duration_days 可通过"""
    data = {"changes": [{
        "id": "test-disabled-no-duration",
        "title": "测试功能",
        "type": "功能上线",
        "date": "2026-07-17",
        "endpoint": "管控端",
        "description": "描述。",
        "display_components": {
            "banner": {"enabled": False},
            "floating_window": {"enabled": False},
        },
    }]}
    passed, errors, _ = validate(data, load_schema(SCHEMA))
    assert passed, errors


def test_duration_default_is_fourteen_days():
    """duration_days 的 schema 默认值为14天"""
    schema = load_schema(SCHEMA)
    duration = schema["$defs"]["DisplayComponent"]["properties"]["duration_days"]
    assert duration.get("default") == 14


def test_component_only_change_updates_existing_entry():
    """仅修改组件开关或时长时也必须更新同 id 条目"""
    original = {
        "id": "feat-component-update-20260717",
        "title": "测试功能",
        "type": "功能上线",
        "date": "2026-07-17",
        "endpoint": "管控端",
        "description": "描述。",
        "display_components": {
            "banner": {"enabled": False},
            "floating_window": {"enabled": False},
        },
    }
    updated = {
        **original,
        "display_components": {
            "banner": {"enabled": True, "duration_days": 14},
            "floating_window": {"enabled": False},
        },
    }
    merged, action = merge_change({"changes": [original]}, updated)
    assert action == "updated"
    assert merged["changes"][0]["display_components"]["banner"]["enabled"] is True


if __name__ == "__main__":
    import subprocess
    result = subprocess.run(
        [sys.executable, "-m", "pytest", __file__, "-v", "--tb=short"],
        capture_output=False,
        cwd=str(Path(__file__).resolve().parent),
        env={**__import__("os").environ, "PYTHONPATH": str(SCRIPT_DIR)},
    )
    sys.exit(result.returncode)

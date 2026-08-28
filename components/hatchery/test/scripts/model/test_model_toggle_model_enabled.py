#!/usr/bin/env python3
"""
POST /admin/models/toggle-enabled HandleToggleModelEnabled 集成测试

接口契约（来自 controller/admin_models.go HandleToggleModelEnabled）：
  - 请求：POST /admin/models/toggle-enabled?id=<model_db_id>（id 走 query string，无 body）
  - 鉴权：必须管理员（requireAdmin）
  - 行为：toggle 模型 ai_model.enabled 字段
  - 联动：若关闭 enabled 时该模型为默认模型，则联动清除 default_model_id
  - 状态码：
      - 401：未携带 token
      - 403：非管理员 token
      - 404：模型不存在
      - 405：非 POST 方法
      - 200：成功

测试场景：
  S1   认证三件套（无认证 / 错误 token / 非管理员） → 401 / 403
  S2   非 POST 方法 → 405
  S3   模型不存在 → 404
  S4   happy path：toggle 把 enabled 从 true 翻成 false，再翻回 true
  S5   联动清除：关闭默认模型的 enabled 后，default_model_id 应被清空
"""

import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
from helpers.api import (
    health_check, make_api_fn,
    auth_test_suite, run_tests,
    admin_client, get_field,
)
from helpers.model import (
    admin_get_models,
    admin_get_default_model_id,
    admin_toggle_default_model,
    admin_delete_model,
)


# ─────────────────────────────────────────────
# API 调用器
# ─────────────────────────────────────────────

post_toggle_enabled = make_api_fn("post", "/admin/models/toggle-enabled")

# 直接复用种子管理员 token，无需为每个用例新建独立 admin（避免用户名复用 409）
ADMIN_TOKEN = config.SEED_ADMIN_TOKEN


# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────

def _create_test_model(admin_token, suffix):
    """创建一个测试模型（dummy 凭证即可，不需要真发 LLM 请求）。

    返回 (db_id, model_name)。
    新建模型默认 enabled=true, visible=false（见 HandleCreateModel）。
    """
    model_name = f"toggle-enabled-itest-{suffix}-{int(time.time() * 1000) % 1000000}"
    model_id = f"dummy-{suffix}-{int(time.time() * 1000) % 1000000}"
    resp = admin_client(admin_token).post(
        "/admin/models/create",
        data={
            "model_id": model_id,
            "model_name": model_name,
            "provider": "自定义模型",
            "api_key": "fake-test-key",
            "url": "https://example.com/v1",
            "model_type": "openai-completions",
            "quota_day": "-1",
        },
        timeout=30,
    )
    assert resp.get("ok"), f"创建模型失败: {resp}"

    models = admin_get_models(admin_token)
    m = next(
        (x for x in models if get_field(x, "ModelName", "model_name") == model_name),
        None,
    )
    assert m, f"模型 {model_name} 未在列表中找到"
    return get_field(m, "ID", "id"), model_name


def _get_model_by_id(admin_token, db_id):
    """从 admin /admin/models 列表中按 id 取出模型记录。"""
    for m in admin_get_models(admin_token):
        if get_field(m, "ID", "id") == db_id:
            return m
    return None


# ─────────────────────────────────────────────
# 用例
# ─────────────────────────────────────────────

def test_toggle_model_enabled_auth():
    """S1：认证三件套（无认证 / 错误 token / 非管理员） → 401 / 403"""
    # 先建一个真实存在的模型，避免被 404 短路（虽然 requireAdmin 在前，
    # 但配上真实 id 可以防止未来调整顺序时误判）。
    db_id, _ = _create_test_model(ADMIN_TOKEN, "auth")
    try:
        auth_test_suite(
            lambda headers: post_toggle_enabled(
                {}, headers=headers, params={"id": db_id},
            ),
            label="model_toggle_model_enabled",
        )
    finally:
        admin_delete_model(ADMIN_TOKEN, db_id)


def test_toggle_model_enabled_method_not_allowed():
    """S2：非 POST 方法 → 405"""
    print(">>> [model_toggle_model_enabled] S2：非 POST → 405 ...")
    # 用 GET 直接发，预期 405（无需先建模型，handler 中 requireAdmin → method 校验 → id 查询）
    get_toggle_enabled = make_api_fn("get", "/admin/models/toggle-enabled")
    resp = get_toggle_enabled(params={"id": 1})
    assert resp.status_code == 405, (
        f"期望 405，实际 {resp.status_code} body={resp.text}"
    )
    print(f"    OK (status=405)")


def test_toggle_model_enabled_not_found():
    """S3：模型不存在 → 404"""
    print(">>> [model_toggle_model_enabled] S3：模型不存在 → 404 ...")
    resp = post_toggle_enabled(params={"id": 999999999})
    assert resp.status_code == 404, (
        f"期望 404，实际 {resp.status_code} body={resp.text}"
    )
    data = resp.json()
    assert "error" in data, f"404 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=404, error={data.get('error', '')[:60]})")


def test_toggle_model_enabled_happy_path():
    """S4：happy path —— enabled 字段翻转、再翻回来，并验证不影响 visible"""
    print(">>> [model_toggle_model_enabled] S4：happy path 翻转 enabled 字段 ...")
    db_id, model_name = _create_test_model(ADMIN_TOKEN, "happy")
    try:
        # 新建模型默认 enabled=true, visible=false
        before = _get_model_by_id(ADMIN_TOKEN, db_id)
        assert before is not None, "刚建好的模型应能在列表中找到"
        # 注意：AIModel.MarshalJSON 字段映射规则（见 model/ai_model.go）：
        #   - JSON 中的 Enabled       = 真实的 Visible（兼容旧前端的「用户可见」开关）
        #   - JSON 中的 EnabledStatus  = 真实的 Enabled（启用/关闭状态）
        #   - 真实的 Visible 字段不再对外输出（被 delete 掉）
        # 因此：要读「真实启用状态」用 EnabledStatus；要读「用户可见」用 Enabled。
        enabled_before = get_field(before, "EnabledStatus", "enabled_status")
        visible_before = get_field(before, "Enabled", "enabled")
        assert enabled_before is True, f"新建模型 EnabledStatus 应为 true，实际 {enabled_before}"
        assert visible_before is False, f"新建模型 visible(=JSON.Enabled) 应为 false，实际 {visible_before}"

        # 第一次 toggle：enabled true → false
        resp = post_toggle_enabled(params={"id": db_id})
        assert resp.status_code == 200, f"toggle 1 期望 200，实际 {resp.status_code} {resp.text}"
        assert resp.json().get("ok"), f"toggle 1 期望 ok=true，实际 {resp.text}"

        after1 = _get_model_by_id(ADMIN_TOKEN, db_id)
        # 真实启用状态读 EnabledStatus（见 model/ai_model.go MarshalJSON 注释）
        assert get_field(after1, "EnabledStatus", "enabled_status") is False, (
            f"第一次 toggle 后 EnabledStatus 应为 false，实际 {after1}"
        )
        # visible 不应被联动修改（JSON.Enabled 承载的是真实 Visible 值）
        assert get_field(after1, "Enabled", "enabled") is False, (
            f"toggle-enabled 不应影响 visible 字段，实际 visible(=JSON.Enabled)={get_field(after1, 'Enabled', 'enabled')}"
        )
        print("    第一次 toggle: EnabledStatus true→false ✓ (visible 未受影响)")

        # 第二次 toggle：enabled false → true
        resp = post_toggle_enabled(params={"id": db_id})
        assert resp.status_code == 200, f"toggle 2 期望 200，实际 {resp.status_code} {resp.text}"
        after2 = _get_model_by_id(ADMIN_TOKEN, db_id)
        assert get_field(after2, "EnabledStatus", "enabled_status") is True, (
            f"第二次 toggle 后 EnabledStatus 应为 true，实际 {after2}"
        )
        print("    第二次 toggle: EnabledStatus false→true ✓")
    finally:
        admin_delete_model(ADMIN_TOKEN, db_id)


def test_toggle_model_enabled_clears_default_when_disabling():
    """S5：关闭默认模型的 enabled 时联动清除 default_model_id"""
    print(">>> [model_toggle_model_enabled] S5：关闭默认模型时联动清除默认 ...")

    # 记录原默认模型，结束时不强制恢复（多用例并发场景下原值可能已变）
    original_default = admin_get_default_model_id(ADMIN_TOKEN)
    db_id, _ = _create_test_model(ADMIN_TOKEN, "default")

    try:
        # 设为默认模型前，需先把模型设为 visible（HandleToggleDefault 要求 visible）
        # 用 /admin/models/toggle 把 visible 从 false 翻成 true
        resp = admin_client(ADMIN_TOKEN).post(
            "/admin/models/toggle", params={"id": db_id},
        )
        assert resp.get("ok"), f"toggle visible 失败: {resp}"

        # 设为默认
        resp = admin_toggle_default_model(ADMIN_TOKEN, db_id)
        assert resp.get("ok"), f"设为默认失败: {resp}"

        new_default = admin_get_default_model_id(ADMIN_TOKEN)
        assert new_default == db_id, (
            f"设为默认后 default_model_id 应为 {db_id}，实际 {new_default}"
        )
        print(f"    模型已设为默认 (default_model_id={db_id}) ✓")

        # 关闭 enabled（true → false）
        resp = post_toggle_enabled(params={"id": db_id})
        assert resp.status_code == 200, f"toggle-enabled 期望 200，实际 {resp.status_code} {resp.text}"

        # 验证：模型的 EnabledStatus=false 且 default_model_id 已被清除
        after = _get_model_by_id(ADMIN_TOKEN, db_id)
        # 真实启用状态读 EnabledStatus（MarshalJSON 把大写 Enabled 输出为 Visible 值）
        assert get_field(after, "EnabledStatus", "enabled_status") is False, (
            f"关闭后 EnabledStatus 应为 false，实际 {after}"
        )
        cleared_default = admin_get_default_model_id(ADMIN_TOKEN)
        assert cleared_default != db_id, (
            f"关闭默认模型的 enabled 后，default_model_id 应被清除（!={db_id}），实际 {cleared_default}"
        )
        print(f"    联动清除 default_model_id ✓ (default_model_id: {db_id} → {cleared_default})")

        # 反向验证：再次 toggle 把 enabled 翻回 true 时，不应自动恢复默认
        resp = post_toggle_enabled(params={"id": db_id})
        assert resp.status_code == 200
        still_default = admin_get_default_model_id(ADMIN_TOKEN)
        assert still_default == cleared_default, (
            f"重新启用 enabled 不应自动恢复 default，实际 {still_default}"
        )
        print("    重新启用不恢复默认 ✓")
    finally:
        admin_delete_model(ADMIN_TOKEN, db_id)


# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="POST /admin/models/toggle-enabled")


if __name__ == "__main__":
    main()
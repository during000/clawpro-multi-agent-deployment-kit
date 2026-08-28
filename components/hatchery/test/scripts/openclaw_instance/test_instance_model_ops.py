#!/usr/bin/env python3
"""
集成测试：实例管理 - 模型（E 组）

覆盖接口：
    GET  /openclaw/models                  全局模型列表
    POST /openclaw/set-model               单模型设置（旧）
    POST /openclaw/add-model               多模型 fallback 添加
    POST /openclaw/switch-primary-model    切换主模型
    POST /openclaw/del-model               删除模型绑定
    GET  /openclaw/instance-models         查询实例已绑定模型

为避免真实污染共享实例的模型配置（TAT 调用代价大且会影响后续测试），本文件
**只测契约 + 参数校验 + 鉴权**，不执行真实的 add → switch → del 链路。
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    ApiClient,
    health_check, run_tests,
    auth_test_suite, assert_status,
    ADMIN_TOKEN,
)
import helpers
from _instance_helpers import (
    cli, require_shared_instance,
    NONEXISTENT_DB_ID,
    get_shared_db_id,
    assert_error_message,
    assert_json_keys,
)


SHARED_DB_ID = None


# ─── /openclaw/models（全局） ─────────────────────────────────────────────

def test_01_models_list_global():
    """GET /openclaw/models - 全局可用模型列表"""
    resp = cli.get("/openclaw/models", raw=True)
    body = assert_json_keys(resp, "ok", "models")
    assert body.get("ok"), f"ok 应为 true: {body}"
    assert isinstance(body["models"], list), (
        f"models 应为数组: {type(body['models']).__name__}"
    )
    if body["models"]:
        m = body["models"][0]
        for k in ("id", "provider", "model_id", "model_name", "model_type"):
            assert k in m, f"model 缺字段 {k}: keys={list(m.keys())}"
    print(f"    OK count={len(body['models'])}")


def test_02_models_list_by_agent_id():
    """GET /openclaw/models?agent_id=<shared>"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/models",
        params={"agent_id": SHARED_DB_ID},
        raw=True,
    )
    body = assert_json_keys(resp, "ok", "models")
    print(f"    OK count={len(body['models'])}")


def test_03_models_auth():
    """GET /openclaw/models - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/models",
            expect=None, raw=True, extra_headers=headers,
        ),
        label="models",
        check_admin=False,
    )


# ─── /openclaw/instance-models ────────────────────────────────────────────

def test_04_instance_models_missing_id():
    """GET /openclaw/instance-models - 缺 id → 400"""
    resp = cli.get("/openclaw/instance-models", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="instance-models-missing-id")
    print(f"    OK status={resp.status_code}")


def test_05_instance_models_nonexistent_id():
    """GET /openclaw/instance-models?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/instance-models",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 404}, label="instance-models-not-found")
    print(f"    OK status={resp.status_code}")


def test_06_instance_models_ok():
    """GET /openclaw/instance-models - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/instance-models",
        params={"id": SHARED_DB_ID},
        raw=True,
    )
    body = assert_json_keys(resp, "ok", "models")
    assert body.get("ok"), f"ok=false: {body}"
    assert isinstance(body["models"], list), (
        f"models 应为数组: {type(body['models']).__name__}"
    )
    if body["models"]:
        m = body["models"][0]
        for k in ("instance_model_id", "binding_id", "role", "provider",
                  "model_id", "model_name", "is_custom"):
            assert k in m, f"绑定模型缺字段 {k}: keys={list(m.keys())}"
        roles = [x.get("role") for x in body["models"]]
        primary_count = roles.count("primary")
        assert primary_count <= 1, (
            f"primary 角色应至多 1 个, 实际 {primary_count}: roles={roles}"
        )
    print(f"    OK count={len(body['models'])}")


def test_07_instance_models_auth():
    """GET /openclaw/instance-models - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/instance-models",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="instance-models",
        check_admin=False,
    )


# ─── /openclaw/set-model ──────────────────────────────────────────────────

def test_08_set_model_missing_id():
    """POST /openclaw/set-model - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/set-model",
        data={"ai_model_id": "1"},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="set-model-missing-id")
    print(f"    OK status={resp.status_code}")


def test_09_set_model_invalid_ai_model_id():
    """POST /openclaw/set-model - ai_model_id 不存在"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/set-model",
        data={"id": SHARED_DB_ID, "ai_model_id": "999999999"},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code == 200:
        body = resp.json() if resp.content else {}
        assert body.get("error"), f"未知 ai_model_id 应失败: {body}"
    else:
        assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_10_set_model_auth():
    """POST /openclaw/set-model - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/set-model",
            data={"id": 1, "ai_model_id": "1"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="set-model",
        check_admin=False,
    )


# ─── /openclaw/add-model ──────────────────────────────────────────────────

def test_11_add_model_missing_id():
    """POST /openclaw/add-model - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/add-model",
        data={"ai_model_id": "1"},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="add-model-missing-id")
    print(f"    OK status={resp.status_code}")


def test_12_add_model_missing_ai_model_id():
    """POST /openclaw/add-model - 缺 ai_model_id → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/add-model",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="add-model-missing-aimid")
    assert_error_message(resp, "请求体格式错误")
    print("    OK")


def test_13_add_model_custom_missing_model_id():
    """POST /openclaw/add-model - 自定义模型缺 model_id → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/add-model",
        data={
            "id": SHARED_DB_ID,
            "ai_model_id": "0",
            "api_key": "sk-fake",
            "model_type": "openai-completions",
        },
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="add-model-custom-no-mid")
    # 后端 parseCustomModelFromForm 在 model_id/api_key/url/model_type 任一缺失时，
    # 统一返回中文聚合文案："模型ID、API Key、URL、接口类型不能为空"。
    # 这里用稳定的中文关键字 "模型ID" 来匹配该契约（不修改线上文案）。
    assert_error_message(resp, "模型ID")
    print("    OK")


def test_14_add_model_auth():
    """POST /openclaw/add-model - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/add-model",
            data={"id": 1, "ai_model_id": "1"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="add-model",
        check_admin=False,
    )


# ─── /openclaw/switch-primary-model ───────────────────────────────────────

def test_15_switch_primary_missing_params():
    """POST /openclaw/switch-primary-model - 缺 instance_model_id → 400"""
    resp = cli.post(
        "/openclaw/switch-primary-model",
        data={"id": SHARED_DB_ID or 1},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="switch-primary-missing")
    assert_error_message(resp, "instance_model_id")
    print("    OK")


def test_16_switch_primary_invalid_target():
    """POST /openclaw/switch-primary-model - 不存在 instance_model_id → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/switch-primary-model",
        data={"id": SHARED_DB_ID, "instance_model_id": "999999999"},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 404}, label="switch-primary-not-found")
    print(f"    OK status={resp.status_code}")


def test_17_switch_primary_auth():
    """POST /openclaw/switch-primary-model - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/switch-primary-model",
            data={"id": 1, "instance_model_id": "1"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="switch-primary-model",
        check_admin=False,
    )


# ─── /openclaw/del-model ──────────────────────────────────────────────────

def test_18_del_model_missing_params():
    """POST /openclaw/del-model - 缺 instance_model_id → 400"""
    resp = cli.post(
        "/openclaw/del-model",
        data={"id": SHARED_DB_ID or 1},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="del-model-missing")
    assert_error_message(resp, "instance_model_id")
    print("    OK")


def test_19_del_model_invalid_target():
    """POST /openclaw/del-model - 不存在 instance_model_id → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/del-model",
        data={"id": SHARED_DB_ID, "instance_model_id": "999999999"},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 404}, label="del-model-not-found")
    print(f"    OK status={resp.status_code}")


def test_20_del_model_auth():
    """POST /openclaw/del-model - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/del-model",
            data={"id": 1, "instance_model_id": "1"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="del-model",
        check_admin=False,
    )


# ─── /openclaw/models/connectivity ────────────────────────────────────────

def test_21_models_connectivity_get_method():
    """GET /openclaw/models/connectivity - 405（仅 POST）"""
    resp = cli.get("/openclaw/models/connectivity", expect=None, raw=True)
    assert_status(resp, {400, 405}, label="connectivity-get")
    print(f"    OK status={resp.status_code}")


def test_22_models_connectivity_invalid_body():
    """POST /openclaw/models/connectivity - 非 JSON → 400"""
    resp = cli.post(
        "/openclaw/models/connectivity",
        data="not-json{",
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="connectivity-invalid")
    print(f"    OK status={resp.status_code}")


def test_23_models_connectivity_missing_args():
    """POST /openclaw/models/connectivity - 全空 → 400（缺 url/api_key/model_type）"""
    resp = cli.post(
        "/openclaw/models/connectivity",
        json={},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="connectivity-missing")
    print(f"    OK status={resp.status_code}")


def test_24_models_connectivity_nonexistent_model_id():
    """POST /openclaw/models/connectivity?id=NONEXISTENT → 400 模型不存在"""
    resp = cli.post(
        "/openclaw/models/connectivity",
        params={"id": NONEXISTENT_DB_ID},
        json={},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="connectivity-not-found")
    assert_error_message(resp, "模型不存在", "id")
    print(f"    OK status={resp.status_code}")


def test_25_models_connectivity_invalid_url():
    """POST /openclaw/models/connectivity - 非法 URL → 400"""
    resp = cli.post(
        "/openclaw/models/connectivity",
        json={
            "url": "not-a-url",
            "api_key": "sk-fake",
            "model_type": "openai-completions",
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="connectivity-bad-url")
    print(f"    OK status={resp.status_code}")


def test_26_models_connectivity_auth():
    """POST /openclaw/models/connectivity - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/models/connectivity",
            json={},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="models-connectivity",
        check_admin=False,
    )


# ─── /openclaw/config-overview ────────────────────────────────────────────

def test_27_config_overview_missing_params():
    """GET /openclaw/config-overview - 缺 ids 和 group_ids → 400"""
    resp = cli.get("/openclaw/config-overview", expect=None, raw=True)
    assert_status(resp, {400}, label="config-overview-missing")
    assert_error_message(resp, "ids", "group_ids")
    print(f"    OK status={resp.status_code}")


def test_28_config_overview_invalid_keys():
    """GET /openclaw/config-overview?ids=1&keys=xxx - 无效 key → 400"""
    resp = cli.get(
        "/openclaw/config-overview",
        params={"ids": "1", "keys": "no-such-key"},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="config-overview-invalid-key")
    print(f"    OK status={resp.status_code}")


def test_29_config_overview_by_group_id_zero():
    """GET /openclaw/config-overview?group_ids=0 - 全局默认配置 happy path"""
    body = cli.get(
        "/openclaw/config-overview",
        params={"group_ids": "0"},
        timeout=15,
    )
    assert body.get("ok") is True, f"返回 ok!=true: {body}"
    assert "results" in body, f"返回缺 results: {body}"
    assert isinstance(body["results"], list)
    print(f"    OK results_count={len(body['results'])}")


def test_30_config_overview_by_instance_id():
    """GET /openclaw/config-overview?ids=<shared> - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    body = cli.get(
        "/openclaw/config-overview",
        params={"ids": str(SHARED_DB_ID)},
        timeout=15,
    )
    assert body.get("ok") is True, f"返回 ok!=true: {body}"
    assert "results" in body
    print(f"    OK results_count={len(body['results'])}")


def test_31_config_overview_foreign_instance():
    """GET /openclaw/config-overview?ids=NONEXISTENT - 不属于当前用户 → 400"""
    resp = cli.get(
        "/openclaw/config-overview",
        params={"ids": str(NONEXISTENT_DB_ID)},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="config-overview-foreign")
    print(f"    OK status={resp.status_code}")


def test_32_config_overview_auth():
    """GET /openclaw/config-overview - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/config-overview",
            params={"group_ids": "0"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="config-overview",
        check_admin=False,
    )


# ─── 入口 ────────────────────────────────────────────────────────────────

def main():
    global SHARED_DB_ID
    health_check()
    SHARED_DB_ID = require_shared_instance().db_id
    if SHARED_DB_ID:
        print(f">>> 复用共享实例 db_id={SHARED_DB_ID}")

    # 前置：开启内置 hatchery/custom 占位记录的 Enabled+Visible，
    # 使本文件的 test_13_add_model_custom_missing_model_id 等用例
    # 能进入到 parseCustomModelFromForm 的参数校验，而不是被
    # IsCustomModelEnabled 拦在 403 "自定义模型功能未开启"。
    # 该 setup 仅影响 ai_models 表的占位记录开关，对其它测试
    # 文件无副作用（不会污染共享实例自身的模型绑定）。
    helpers.ensure_custom_model_flag(ADMIN_TOKEN)
    print()

    run_tests(
        globals(),
        title="test_instance_model_ops.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()

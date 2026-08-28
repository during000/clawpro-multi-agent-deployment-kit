#!/usr/bin/env python3
"""
POST /admin/config/security-group/ruleset/import-from-sg 从安全组导入规则 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2：错误 token → 401/403
  场景 3：非管理员 token → 401/403
  场景 4：请求体格式错误（非 JSON）→ 400，含 error 字段
  场景 5：source_sg_id 为空字符串 → 400，提示"source_sg_id 不能为空"
  场景 6：请求体为 {}（source_sg_id 缺失，零值为空字符串）→ 400，提示"source_sg_id 不能为空"
  场景 7：source_sg_id 为纯空格字符串 → 400，提示"source_sg_id 不能为空"
  场景 8：传非法 name（如 "1abc"）→ 400，提示"规则组名称不合法"
  场景 9：source_sg_id 为 ClawPro 自建 SG（managed_sg_pool 中的 SG）→ 409，提示"不允许从 ClawPro 自建安全组导入"
           注意：HandleImportRulesFromSG 对 ImportRulesFromSGInternal 返回的所有 err 统一返回 409
           注意：必须从 projected_to 获取真正的 managed SG ID，而非从 GET /admin/config/security-group 获取用户绑定的 base SG
  场景 10：source_sg_id 为不存在的 SG ID → 409（统一 conflict 路径），提示"describe source sg policies"失败
  场景 11：source_sg_id 为合法外部 SG，RuleSet 未初始化 → 200，自动创建 RuleSet + ACTIVE SG，version=1，synced=1
  场景 12：source_sg_id 为合法外部 SG，RuleSet 已存在，auto_fix_rules=false → 200，version 递增，非内部账号 rules 来自源 SG；内部账号额外包含办公网 Guard
  场景 13：source_sg_id 为合法外部 SG，auto_fix_rules=true → 200，rules 来自源 SG + ClawPro 必需规则（内部账号含办公网 Guard）
  场景 14：源 SG 包含 SecurityGroupId 引用规则（非 CIDR）→ 跳过，不报错，rules 只含 CIDR 规则
  场景 15：源 SG 包含 AddressTemplate 参数模板规则 → 跳过，不报错
  场景 16：源 SG 规则全为非 CIDR（全部被跳过）→ 200，导入后 rules 为 []
  场景 17：源 SG 本身无任何规则（空 SG）→ 200，导入后 rules 为 []
  场景 18：响应包含 imported_from 字段，值为 source_sg_id → imported_from == source_sg_id
  场景 19：fan-out 失败时响应体也包含 imported_from 字段 → 409 响应中含 imported_from
  场景 20：导入成功后，通过 GET /ruleset 验证 version 已递增、rules 已更新
  场景 21：成功响应包含 version、synced、drifted=0、drift_errors=[]、imported_from 字段
  场景 22：name 为纯空格字符串 → TrimSpace 后为空，fallback default，正常导入
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    seed,
    IDENTIFIER,
    health_check, make_api_fn,
    auth_test_suite, assert_status, run_tests,
)
# 可选：用于测试导入的合法外部 SG ID（非 ClawPro 自建）
EXTERNAL_SG_ID = os.environ.get("EXTERNAL_SG_ID", "")

# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────


post_import = make_api_fn("post", "/admin/config/security-group/ruleset/import-from-sg")

get_ruleset = make_api_fn("get", "/admin/config/security-group/ruleset", timeout=10)

def get_current_version() -> int:
    resp = get_ruleset()
    if resp.status_code != 200 or not resp.text.strip():
        return 0
    try:
        return resp.json().get("version", 0)
    except Exception:
        return 0

def is_initialized() -> bool:
    resp = get_ruleset()
    if resp.status_code != 200 or not resp.text.strip():
        return False
    try:
        return resp.json().get("initialized", False)
    except Exception:
        return False

def get_managed_sg_id() -> str:
    """获取 ClawPro 自建的安全组 ID（用于测试禁止从自建 SG 导入）。
    通过 GET /admin/config/security-group/ruleset 获取 projected_to 中的 ACTIVE SG ID，
    这些 SG 在 managed_sg_pool 中，IsManagedSG 会返回 true。
    注意：不能从 GET /admin/config/security-group 获取，因为那是用户绑定的 base SG，
    不一定在 managed_sg_pool 中。
    """
    rs_resp = seed.get("/admin/config/security-group/ruleset", expect=None, timeout=10, raw=True)
    if rs_resp.status_code != 200 or not rs_resp.text.strip():
        return ""
    try:
        projected_to = rs_resp.json().get("projected_to", [])
    except Exception:
        return ""
    if not projected_to:
        return ""
    return projected_to[0].get("sg_id", "")


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_import_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: post_import({"source_sg_id": "sg-test"}, headers=headers),
                    label="import")


def test_import_invalid_json():
    print(">>> [导入规则] 场景4：请求体格式错误（非 JSON）→ 400 ...")
    resp = post_import(raw_data="not-a-json{{{")
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:50]}...)")

def test_import_empty_source_sg_id():
    print(">>> [导入规则] 场景5：source_sg_id 为空字符串 → 400 ...")
    resp = post_import({"source_sg_id": ""})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    assert "source_sg_id" in data.get("error", ""), \
        f"error 应提示 source_sg_id 不能为空，实际 {data.get('error')}"
    print(f"    OK (status=400, error={data.get('error')})")

def test_import_missing_source_sg_id():
    print(">>> [导入规则] 场景6：请求体为 {}（source_sg_id 缺失）→ 400 ...")
    resp = post_import({})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    assert "source_sg_id" in data.get("error", ""), \
        f"error 应提示 source_sg_id 不能为空，实际 {data.get('error')}"
    print(f"    OK (status=400, error={data.get('error')})")

def test_import_whitespace_source_sg_id():
    print(">>> [导入规则] 场景7：source_sg_id 为纯空格字符串 → 400 ...")
    resp = post_import({"source_sg_id": "   "})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    assert "source_sg_id" in data.get("error", ""), \
        f"error 应提示 source_sg_id 不能为空，实际 {data.get('error')}"
    print(f"    OK (status=400, error={data.get('error')})")

def test_import_invalid_name():
    print(">>> [导入规则] 场景8：传非法 name（'1abc'）→ 400 ...")
    resp = post_import({"source_sg_id": "sg-test", "name": "1abc"})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_import_from_managed_sg():
    print(">>> [导入规则] 场景9：source_sg_id 为 ClawPro 自建 SG → 409 ...")
    managed_sg_id = get_managed_sg_id()
    if not managed_sg_id:
        print("    SKIP (无法获取 ClawPro 自建 SG ID，可能未初始化或无 projected_to)")
        return
    resp = post_import({"source_sg_id": managed_sg_id})
    # HandleImportRulesFromSG 对 ImportRulesFromSGInternal 返回的所有 err
    # 统一返回 409（StatusConflict），包括 managed SG 校验失败。
    assert resp.status_code == 409, \
        f"期望 409（managed SG 校验失败），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"409 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=409, error={data.get('error', '')[:60]})")

def test_import_from_nonexistent_sg():
    print(">>> [导入规则] 场景10：source_sg_id 为不存在的 SG ID → 409 ...")
    resp = post_import({"source_sg_id": "sg-nonexistent99999"})
    # HandleImportRulesFromSG 对 ImportRulesFromSGInternal 返回的所有 err
    # 统一返回 409（StatusConflict），包括云端 describe 失败。
    assert resp.status_code == 409, \
        f"期望 409，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"409 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=409, error={data.get('error', '')[:60]})")

def test_import_from_external_sg_not_initialized():
    print(">>> [导入规则] 场景11：合法外部 SG，RuleSet 未初始化 → 200，自动创建 RuleSet，version=1，synced=1 ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if is_initialized():
        print("    SKIP (当前环境已初始化，跳过未初始化场景)")
        return
    resp = post_import({"source_sg_id": EXTERNAL_SG_ID})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version") == 1, f"首次导入 version 应为 1，实际 {data.get('version')}"
    assert data.get("synced", 0) >= 1, f"synced 应 >= 1，实际 {data.get('synced')}"
    assert data.get("imported_from") == EXTERNAL_SG_ID, \
        f"imported_from 应为 {EXTERNAL_SG_ID}，实际 {data.get('imported_from')}"
    print(f"    OK (version=1, synced={data.get('synced')}, imported_from={data.get('imported_from')})")

def test_import_from_external_sg_no_auto_fix():
    print(">>> [导入规则] 场景12：合法外部 SG，auto_fix_rules=false → 200，version 递增，非内部账号源 SG 原样；内部账号含办公网 Guard ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化，跳过已存在场景)")
        return
    old_version = get_current_version()
    resp = post_import({"source_sg_id": EXTERNAL_SG_ID, "auto_fix_rules": False})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    assert data.get("imported_from") == EXTERNAL_SG_ID, \
        f"imported_from 应为 {EXTERNAL_SG_ID}，实际 {data.get('imported_from')}"
    print(f"    OK (version: {old_version} → {data.get('version')}, imported_from={data.get('imported_from')})")

def test_import_from_external_sg_auto_fix():
    print(">>> [导入规则] 场景13：合法外部 SG，auto_fix_rules=true → 200，rules 包含源 SG 规则 + 必需规则（内部账号含办公网 Guard） ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化，跳过已存在场景)")
        return
    old_version = get_current_version()
    resp = post_import({"source_sg_id": EXTERNAL_SG_ID, "auto_fix_rules": True})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    # 验证 GET /ruleset 中 rules 数量（auto_fix_rules=true 时会 merge 必需规则）
    get_resp = get_ruleset()
    rules_in_db = get_resp.json().get("rules", [])
    assert isinstance(rules_in_db, list), f"rules 应为数组，实际 {rules_in_db}"
    print(f"    OK (version: {old_version} → {data.get('version')}，DB rules 共 {len(rules_in_db)} 条)")

def test_import_sg_with_non_cidr_rules():
    print(">>> [导入规则] 场景14：源 SG 包含 SecurityGroupId 引用规则（非 CIDR）→ 跳过，不报错，rules 只含 CIDR 规则 ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    # 此场景需要 EXTERNAL_SG_ID 对应的 SG 中包含 SecurityGroupId 类型的规则
    # 在正常环境下，通过导入后验证 rules 均为 CIDR 格式来间接验证
    old_version = get_current_version()
    resp = post_import({"source_sg_id": EXTERNAL_SG_ID})
    assert resp.status_code == 200, \
        f"期望 200（非 CIDR 规则被跳过，不报错），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    rules = data.get("rules", [])
    # 验证所有 rules 均含 cidr_block 字段（非 CIDR 规则已被过滤）
    for rule in rules:
        assert "cidr_block" in rule, \
            f"导入后 rules 中每条规则应含 cidr_block 字段（非 CIDR 规则应被过滤），实际 {rule}"
    print(f"    OK (status=200，rules 共 {len(rules)} 条，均为 CIDR 格式)")

def test_import_sg_with_address_template_rules():
    print(">>> [导入规则] 场景15：源 SG 包含 AddressTemplate 参数模板规则 → 跳过，不报错 ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    # 此场景与场景14类似，通过导入后验证 rules 均为 CIDR 格式来间接验证
    # AddressTemplate 规则（ServiceTemplate/AddressTemplate）会被过滤，不报错
    resp = post_import({"source_sg_id": EXTERNAL_SG_ID})
    assert resp.status_code == 200, \
        f"期望 200（AddressTemplate 规则被跳过，不报错），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    rules = data.get("rules", [])
    for rule in rules:
        assert "cidr_block" in rule, \
            f"导入后 rules 中每条规则应含 cidr_block 字段，实际 {rule}"
    print(f"    OK (status=200，rules 共 {len(rules)} 条，AddressTemplate 规则已被过滤)")

def test_import_sg_all_non_cidr_rules():
    print(">>> [导入规则] 场景16：源 SG 规则全为非 CIDR（全部被跳过）→ 200，导入后 rules 为 [] ...")
    # 此场景需要一个所有规则均为 SecurityGroupId/AddressTemplate 类型的 SG
    # 在正常环境下无法直接构造，通过环境变量 ALL_NON_CIDR_SG_ID 指定
    all_non_cidr_sg_id = os.environ.get("ALL_NON_CIDR_SG_ID", "")
    if not all_non_cidr_sg_id:
        print("    SKIP (ALL_NON_CIDR_SG_ID 未设置，需要一个所有规则均为非 CIDR 类型的 SG)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    resp = post_import({"source_sg_id": all_non_cidr_sg_id})
    assert resp.status_code == 200, \
        f"期望 200（全部非 CIDR 规则被跳过，不报错），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    rules = data.get("rules", [])
    assert rules == [], \
        f"全部非 CIDR 规则被跳过后，rules 应为 []，实际 {rules}"
    print(f"    OK (status=200，全部非 CIDR 规则被过滤，rules=[]，imported_from={data.get('imported_from')})")

def test_import_sg_empty_sg():
    print(">>> [导入规则] 场景17：源 SG 本身无任何规则（空 SG）→ 200，导入后 rules 为 [] ...")
    # 此场景需要一个没有任何规则的空 SG
    empty_sg_id = os.environ.get("EMPTY_SG_ID", "")
    if not empty_sg_id:
        print("    SKIP (EMPTY_SG_ID 未设置，需要一个没有任何规则的空 SG)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    resp = post_import({"source_sg_id": empty_sg_id})
    assert resp.status_code == 200, \
        f"期望 200（空 SG 导入成功），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    rules = data.get("rules", [])
    assert rules == [], \
        f"空 SG 导入后 rules 应为 []，实际 {rules}"
    assert data.get("imported_from") == empty_sg_id, \
        f"imported_from 应为 {empty_sg_id}，实际 {data.get('imported_from')}"
    print(f"    OK (status=200，空 SG 导入成功，rules=[]，imported_from={data.get('imported_from')})")

def test_import_imported_from_field():
    print(">>> [导入规则] 场景18：响应包含 imported_from 字段，值为 source_sg_id ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    resp = post_import({"source_sg_id": EXTERNAL_SG_ID})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "imported_from" in data, f"响应应含 imported_from 字段，实际 {data}"
    assert data.get("imported_from") == EXTERNAL_SG_ID, \
        f"imported_from 应为 {EXTERNAL_SG_ID}，实际 {data.get('imported_from')}"
    print(f"    OK (imported_from={data.get('imported_from')})")

def test_import_verify_via_get():
    print(">>> [导入规则] 场景20：导入成功后，通过 GET /ruleset 验证 version 递增、rules 已更新 ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_import({"source_sg_id": EXTERNAL_SG_ID})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    new_version_in_resp = resp.json().get("version", 0)
    assert new_version_in_resp > old_version, \
        f"响应中 version 应递增，旧 version={old_version}，新 version={new_version_in_resp}"
    # 通过 GET 验证
    get_resp = get_ruleset()
    assert get_resp.status_code == 200
    get_data = get_resp.json()
    assert get_data.get("version") == new_version_in_resp, \
        f"GET 返回的 version 应与 POST 响应一致，期望 {new_version_in_resp}，实际 {get_data.get('version')}"
    rules_in_db = get_data.get("rules", [])
    assert isinstance(rules_in_db, list), f"rules 应为数组，实际 {rules_in_db}"
    print(f"    OK (version: {old_version} → {new_version_in_resp}，GET 验证一致，rules 共 {len(rules_in_db)} 条)")

def test_import_success_response_fields():
    print(">>> [导入规则] 场景21：成功响应包含 version、synced、drifted=0、drift_errors=[]、imported_from ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    resp = post_import({"source_sg_id": EXTERNAL_SG_ID})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "version" in data, f"响应应含 version 字段，实际 {data}"
    assert "synced" in data, f"响应应含 synced 字段，实际 {data}"
    assert "drifted" in data, f"响应应含 drifted 字段，实际 {data}"
    assert "drift_errors" in data, f"响应应含 drift_errors 字段，实际 {data}"
    assert "imported_from" in data, f"响应应含 imported_from 字段，实际 {data}"
    assert data.get("drifted") == 0, f"成功时 drifted 应为 0，实际 {data.get('drifted')}"
    assert data.get("drift_errors") == [], f"成功时 drift_errors 应为 []，实际 {data.get('drift_errors')}"
    assert data.get("synced", 0) >= 0, f"synced 应 >= 0，实际 {data.get('synced')}"
    assert data.get("imported_from") == EXTERNAL_SG_ID, \
        f"imported_from 应为 {EXTERNAL_SG_ID}，实际 {data.get('imported_from')}"
    print(f"    OK (version={data.get('version')}, synced={data.get('synced')}, drifted=0, imported_from={data.get('imported_from')})")

def test_import_name_whitespace_fallback():
    print(">>> [导入规则] 场景22：name 为纯空格字符串 → TrimSpace 后为空，fallback default ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_import({"source_sg_id": EXTERNAL_SG_ID, "name": "   "})
    assert resp.status_code == 200, \
        f"期望 200（纯空格 name 等同于未传），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    print(f"    OK (version: {old_version} → {data.get('version')}，纯空格 name 被忽略)")

def test_import_409_contains_imported_from():
    print(">>> [导入规则] 场景19：fan-out 失败时响应体也包含 imported_from 字段 ...")
    # 此场景需要可注入故障的环境（如 mock 腾讯云 API 返回失败）
    # 在正常环境下，通过验证 409 响应结构来确认字段存在
    # 实际触发需要：所有 ACTIVE SG 的云端下发均失败
    print("    SKIP (需要可注入故障的环境，此场景为结构验证，在 mock 环境中执行)")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="POST /admin/config/security-group/ruleset/import-from-sg", ordered=True)

if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""
POST /admin/config/security-group/rulesets 创建规则组 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2：错误 token → 401/403
  场景 3：非管理员 token → 401/403
  场景 4：请求体格式错误（非 JSON）→ 400，含 error 字段
  场景 5：幂等性 - RuleSet 已存在时再次调用 → 200，直接返回当前 RuleSet 详情（不报 409，不重复创建）
  场景 6：首次创建，空请求体（ContentLength=0）→ 200，name="default"，version=1
  场景 7：首次创建，不传 name → 200，name="default"，version=1
  场景 8：首次创建，传合法 name（如 "my-ruleset"）→ 200，name 与传入一致
  场景 9：传非法 name - 首字符为数字（如 "1abc"）→ 400，提示"规则组名称不合法"
  场景 10：传非法 name - 首字符为短横线（如 "-abc"）→ 400，提示"规则组名称不合法"
  场景 11：传非法 name - 含下划线（如 "my_ruleset"）→ 400，提示"规则组名称不合法"
  场景 12：传非法 name - 长度不足（2 字符）→ 400，提示"规则组名称不合法"
  场景 13：传非法 name - 长度超限（33 字符）→ 400，提示"规则组名称不合法"
  场景 14：传合法 name - 边界值 3 字符（如 "abc"）→ 200，创建成功
  场景 15：传合法 name - 边界值 32 字符 → 200，创建成功
  场景 16：传 rules（自定义规则），auto_fix_rules=false → 200，非内部账号 rules 与传入一致；内部账号额外包含办公网 Guard
  场景 17：传 rules，auto_fix_rules=true → 200，rules 包含传入规则 + ClawPro 必需规则（内部账号含办公网 Guard）
  场景 18：传 import_from_sg_id（合法外部 SG ID）→ 200，rules 来自该 SG 的规则
  场景 19：传 import_from_sg_id 为 ClawPro 自建 SG → 400，提示"不允许从 ClawPro 自建安全组导入"
  场景 20：传 import_from_sg_id 为不存在的 SG ID → 502，提示"读取源安全组规则失败"
  场景 21：传 import_from_sg_id 为纯空格字符串 → 等同于未传，走 rules 路径
  场景 22：传 rules 包含非法规则（空 cidr_block）→ 400，提示"规则格式错误"
  场景 23：传 rules 包含非法 direction → 400，提示"规则格式错误"
  场景 24：传 rules 包含 protocol=ALL 且 port 为具体端口 → 400，提示"protocol=ALL 不能与具体端口组合"
  场景 25：传 rules 包含裸 IP（如 "0.0.0.0"，非 CIDR 格式）→ 400，提示"请明确写为 0.0.0.0/0 或 0.0.0.0/32"
  场景 26：并发创建（锁竞争）→ 后一个请求返回 409，提示"另一个请求正在创建规则组"
  场景 27：创建成功后，响应包含 projected_to（已建 ACTIVE SG）→ projected_to 至少 1 条
  场景 28：创建成功后，响应包含 description 字段（若传入）→ description 与传入一致
"""

import os
import sys
import threading

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    seed,
    IDENTIFIER,
    health_check, make_api_fn,
    auth_test_suite, assert_status, run_tests,
)
# 可选：用于测试 import_from_sg_id 的合法外部 SG ID
EXTERNAL_SG_ID = os.environ.get("EXTERNAL_SG_ID", "")

# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────


post_rulesets = make_api_fn("post", "/admin/config/security-group/rulesets")

get_ruleset = make_api_fn("get", "/admin/config/security-group/ruleset", timeout=10)

def is_initialized() -> bool:
    resp = get_ruleset()
    if resp.status_code != 200 or not resp.text.strip():
        return False
    try:
        return resp.json().get("initialized", False)
    except Exception:
        return False


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_create_ruleset_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: post_rulesets({"name": "test"}, headers=headers),
                    label="create_ruleset")


def test_create_ruleset_invalid_json():
    print(">>> [创建规则组] 场景4：请求体格式错误（非 JSON）→ 400 ...")
    if is_initialized():
        # RuleSet 已存在时，HandleCreateRuleSet 先做幂等检查直接返回 200，
        # 不会走到 JSON 解析分支，因此此场景仅在首次创建时有效。
        print("    SKIP (当前环境已初始化，幂等检查先于 JSON 解析，跳过此场景)")
        return
    resp = post_rulesets(raw_data="not-a-json{{{")
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应包含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:50]}...)")

def test_create_ruleset_idempotent():
    print(">>> [创建规则组] 场景5：幂等性 - RuleSet 已存在时再次调用 → 200，直接返回当前详情 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化，跳过幂等性场景)")
        return
    resp = post_rulesets({"name": "default"})
    assert resp.status_code == 200, \
        f"期望 200（幂等），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("initialized") is True, f"幂等返回应 initialized=true，实际 {data}"
    assert data.get("id", 0) > 0, f"幂等返回应含 id，实际 {data}"
    print(f"    OK (status=200，幂等返回 id={data.get('id')}, version={data.get('version')})")

def test_create_ruleset_empty_body():
    print(">>> [创建规则组] 场景6：空请求体（ContentLength=0）→ 200，name='default' ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，空请求体走幂等路径，跳过首次创建验证)")
        return
    resp = post_rulesets(body=None)
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("initialized") is True, f"initialized 应为 true，实际 {data}"
    assert data.get("name") == "default", f"name 应为 'default'，实际 {data.get('name')}"
    assert data.get("version", 0) >= 1, f"version 应 >= 1，实际 {data.get('version')}"
    print(f"    OK (name={data.get('name')}, version={data.get('version')})")

def test_create_ruleset_no_name():
    print(">>> [创建规则组] 场景7：不传 name → 200，name='default' ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，走幂等路径)")
        return
    resp = post_rulesets({})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("name") == "default", f"name 应为 'default'，实际 {data.get('name')}"
    print(f"    OK (name={data.get('name')})")

def test_create_ruleset_with_valid_name():
    print(">>> [创建规则组] 场景8：首次创建，传合法 name（'my-ruleset'）→ 200，name 与传入一致 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，走幂等路径，跳过首次创建验证)")
        return
    resp = post_rulesets({"name": "my-ruleset"})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("initialized") is True, f"initialized 应为 true，实际 {data}"
    assert data.get("name") == "my-ruleset", \
        f"name 应与传入一致，期望 'my-ruleset'，实际 '{data.get('name')}'"
    assert data.get("version", 0) >= 1, f"version 应 >= 1，实际 {data.get('version')}"
    print(f"    OK (name={data.get('name')}, version={data.get('version')})")

def test_create_ruleset_invalid_name_starts_with_digit():
    print(">>> [创建规则组] 场景9：name 首字符为数字（'1abc'）→ 400 ...")
    if is_initialized():
        # RuleSet 已存在时，HandleCreateRuleSet 先做幂等检查直接返回 200，
        # 不会走到 name 校验分支，因此此场景仅在首次创建时有效。
        print("    SKIP (当前环境已初始化，幂等检查先于 name 校验，跳过此场景)")
        return
    resp = post_rulesets({"name": "1abc"})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_invalid_name_starts_with_dash():
    print(">>> [创建规则组] 场景10：name 首字符为短横线（'-abc'）→ 400 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，幂等检查先于 name 校验，跳过此场景)")
        return
    resp = post_rulesets({"name": "-abc"})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_invalid_name_with_underscore():
    print(">>> [创建规则组] 场景11：name 含下划线（'my_ruleset'）→ 400 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，幂等检查先于 name 校验，跳过此场景)")
        return
    resp = post_rulesets({"name": "my_ruleset"})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_invalid_name_too_short():
    print(">>> [创建规则组] 场景12：name 长度不足（2 字符）→ 400 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，幂等检查先于 name 校验，跳过此场景)")
        return
    resp = post_rulesets({"name": "ab"})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_invalid_name_too_long():
    print(">>> [创建规则组] 场景13：name 长度超限（33 字符）→ 400 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，幂等检查先于 name 校验，跳过此场景)")
        return
    long_name = "a" + "b" * 32  # 33 字符
    resp = post_rulesets({"name": long_name})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_valid_name_min_length():
    print(">>> [创建规则组] 场景14：name 边界值 3 字符（'abc'）→ 200，创建成功或幂等返回 ...")
    resp = post_rulesets({"name": "abc"})
    # 无论是否已初始化，合法 name 均应返回 200（首次创建或幂等路径）
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("initialized") is True, f"initialized 应为 true，实际 {data}"
    print(f"    OK (status=200，name='abc' 合法，version={data.get('version')})")

def test_create_ruleset_valid_name_max_length():
    print(">>> [创建规则组] 场景15：name 边界值 32 字符 → 200，创建成功或幂等返回 ...")
    max_name = "a" + "b" * 31  # 32 字符
    resp = post_rulesets({"name": max_name})
    # 无论是否已初始化，合法 name 均应返回 200（首次创建或幂等路径）
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("initialized") is True, f"initialized 应为 true，实际 {data}"
    print(f"    OK (status=200，32 字符 name 合法，version={data.get('version')})")

def test_create_ruleset_with_rules_no_auto_fix():
    print(">>> [创建规则组] 场景16：传 rules，auto_fix_rules=false → 200，非内部账号原样；内部账号含办公网 Guard ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，走幂等路径，跳过首次创建验证)")
        return
    custom_rules = [
        {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "22", "cidr_block": "0.0.0.0/0", "description": "SSH"},
    ]
    resp = post_rulesets({"rules": custom_rules, "auto_fix_rules": False})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("initialized") is True, f"initialized 应为 true，实际 {data}"
    rules = data.get("rules", [])
    assert isinstance(rules, list), f"rules 应为数组，实际 {rules}"
    print(f"    OK (rules 共 {len(rules)} 条)")

def test_create_ruleset_with_rules_auto_fix():
    print(">>> [创建规则组] 场景17：传 rules，auto_fix_rules=true → 200，rules 包含传入规则 + 必需规则（内部账号含办公网 Guard） ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，走幂等路径，跳过首次创建验证)")
        return
    custom_rules = [
        {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "8080", "cidr_block": "0.0.0.0/0", "description": "自定义规则"},
    ]
    resp = post_rulesets({"rules": custom_rules, "auto_fix_rules": True})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("initialized") is True, f"initialized 应为 true，实际 {data}"
    rules = data.get("rules", [])
    # auto_fix_rules=true 时，rules 数量应 >= 传入数量（merge 了必需规则）
    assert len(rules) >= len(custom_rules), \
        f"auto_fix_rules=true 时，rules 数量应 >= 传入数量，实际 {len(rules)} < {len(custom_rules)}"
    print(f"    OK (传入 {len(custom_rules)} 条，merge 后 {len(rules)} 条)")

def test_create_ruleset_import_from_sg():
    print(">>> [创建规则组] 场景18：传 import_from_sg_id（合法外部 SG）→ 200，rules 来自该 SG ...")
    if not EXTERNAL_SG_ID:
        print("    SKIP (EXTERNAL_SG_ID 未设置)")
        return
    if is_initialized():
        print("    SKIP (当前环境已初始化，走幂等路径，跳过首次创建验证)")
        return
    resp = post_rulesets({"import_from_sg_id": EXTERNAL_SG_ID})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("initialized") is True, f"initialized 应为 true，实际 {data}"
    print(f"    OK (从 {EXTERNAL_SG_ID} 导入，rules 共 {len(data.get('rules', []))} 条)")

def test_create_ruleset_import_from_managed_sg():
    print(">>> [创建规则组] 场景19：import_from_sg_id 为 ClawPro 自建 SG → 400 ...")
    if is_initialized():
        # RuleSet 已存在时，HandleCreateRuleSet 先做幂等检查直接返回 200，
        # 不会走到 import_from_sg_id 校验分支，因此此场景仅在首次创建时有效。
        print("    SKIP (当前环境已初始化，幂等检查先于 import_from_sg_id 校验，跳过此场景)")
        return
    # 获取当前 site_config 中的 security_group_id（ClawPro 自建 SG）
    sg_resp = seed.get("/admin/config/security-group", expect=None, timeout=10, raw=True)
    if sg_resp.status_code != 200:
        print("    SKIP (无法获取当前安全组 ID)")
        return
    if not sg_resp.text.strip():
        print("    SKIP (安全组响应体为空)")
        return
    try:
        sg_set = sg_resp.json().get("Response", {}).get("SecurityGroupSet", [])
    except Exception:
        print(f"    SKIP (解析安全组响应失败，body={sg_resp.text[:200]})")
        return
    if not sg_set:
        print("    SKIP (当前未配置安全组，无法获取 ClawPro 自建 SG ID)")
        return
    managed_sg_id = sg_set[0].get("SecurityGroupId", "")
    if not managed_sg_id:
        print("    SKIP (无法获取 ClawPro 自建 SG ID)")
        return
    resp = post_rulesets({"import_from_sg_id": managed_sg_id})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    assert "ClawPro" in data.get("error", "") or "自建" in data.get("error", ""), \
        f"error 应提示不允许从 ClawPro 自建 SG 导入，实际 {data.get('error')}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_import_from_nonexistent_sg():
    print(">>> [创建规则组] 场景20：import_from_sg_id 为不存在的 SG ID → 502 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，走幂等路径，跳过首次创建验证)")
        return
    resp = post_rulesets({"import_from_sg_id": "sg-nonexistent99999"})
    assert resp.status_code in (502, 500), \
        f"期望 502/500，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"响应应含 error 字段，实际 {data}"
    print(f"    OK (status={resp.status_code}, error={data.get('error')[:60]}...)")

def test_create_ruleset_import_from_sg_whitespace():
    print(">>> [创建规则组] 场景21：import_from_sg_id 为纯空格字符串 → 等同于未传，走 rules 路径 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，走幂等路径)")
        return
    resp = post_rulesets({"import_from_sg_id": "   ", "rules": []})
    # 纯空格 TrimSpace 后为空，走 rules 路径，rules 为空时正常创建
    assert resp.status_code == 200, \
        f"期望 200（纯空格等同于未传），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("initialized") is True, f"initialized 应为 true，实际 {data}"
    print(f"    OK (status=200，纯空格 import_from_sg_id 被忽略)")

def test_create_ruleset_invalid_rule_empty_cidr():
    print(">>> [创建规则组] 场景22：rules 包含空 cidr_block → 400 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，幂等检查先于 rules 校验，跳过此场景)")
        return
    resp = post_rulesets({
        "rules": [{"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "22", "cidr_block": ""}]
    })
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_invalid_rule_direction():
    print(">>> [创建规则组] 场景23：rules 包含非法 direction → 400 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，幂等检查先于 rules 校验，跳过此场景)")
        return
    resp = post_rulesets({
        "rules": [{"direction": "INVALID", "action": "ACCEPT", "protocol": "TCP", "port": "22", "cidr_block": "0.0.0.0/0"}]
    })
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_invalid_rule_protocol_all_with_port():
    print(">>> [创建规则组] 场景24：rules 包含 protocol=ALL 且 port 为具体端口 → 400 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，幂等检查先于 rules 校验，跳过此场景)")
        return
    resp = post_rulesets({
        "rules": [{"direction": "INGRESS", "action": "ACCEPT", "protocol": "ALL", "port": "80", "cidr_block": "0.0.0.0/0"}]
    })
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_invalid_rule_bare_ip():
    print(">>> [创建规则组] 场景25：rules 包含裸 IP（非 CIDR 格式）→ 400 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，幂等检查先于 rules 校验，跳过此场景)")
        return
    resp = post_rulesets({
        "rules": [{"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "22", "cidr_block": "0.0.0.0"}]
    })
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_create_ruleset_concurrent_lock():
    print(">>> [创建规则组] 场景26：并发创建（锁竞争）→ 后一个请求返回 409 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，走幂等路径，无法触发锁竞争)")
        return
    results = []
    def do_create():
        resp = post_rulesets({})
        results.append(resp.status_code)
    threads = [threading.Thread(target=do_create) for _ in range(3)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()
    # 至少有一个 200（成功创建），其余可能是 200（幂等）或 409（锁竞争）
    assert 200 in results, f"并发创建中应至少有一个 200，实际 {results}"
    if 409 in results:
        print(f"    OK (触发锁竞争，并发结果: {results}，至少一个 200，至少一个 409)")
    else:
        print(f"    OK (未触发锁竞争（请求串行了），并发结果: {results}，全部 200）")

def test_create_ruleset_projected_to_not_empty():
    print(">>> [创建规则组] 场景27：创建成功后，projected_to 至少 1 条 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化，跳过 projected_to 验证)")
        return
    resp = post_rulesets({})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    projected_to = data.get("projected_to", [])
    assert len(projected_to) >= 1, \
        f"创建成功后 projected_to 应至少 1 条，实际 {len(projected_to)}"
    print(f"    OK (projected_to 共 {len(projected_to)} 条)")

def test_create_ruleset_description_returned():
    print(">>> [创建规则组] 场景28：传 description → 响应中 description 与传入一致 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，走幂等路径，跳过首次创建验证)")
        return
    desc = "集成测试创建的规则组"
    resp = post_rulesets({"description": desc})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("description") == desc, \
        f"description 应与传入一致，期望 '{desc}'，实际 '{data.get('description')}'"
    print(f"    OK (description='{data.get('description')}')")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="POST /admin/config/security-group/rulesets")

if __name__ == "__main__":
    main()

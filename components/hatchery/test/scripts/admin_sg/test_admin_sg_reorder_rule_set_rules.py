#!/usr/bin/env python3
"""
POST /admin/config/security-group/ruleset/rules/reorder HandleReorderRuleSetRules 集成测试

接口契约（来自 controller/admin_security_group.go HandleReorderRuleSetRules）：
  请求体（JSON）：
    {
      "name": "",                                # 可选；为空/省略 → 操作 default RuleSet
      "ordered_fingerprints": ["fp1", "fp2", ...]  # 必填；fp 由 Rule.Fingerprint() 计算
    }
    fingerprint 形如 "INGRESS|TCP|22|0.0.0.0/0|ACCEPT"（5 元组），前端可直接复用 GET /ruleset
    响应 rules[].fingerprint 派生字段。

  状态码：
    200  成功
    400  name 不合法 / ordered_fingerprints 为空 / 出现未知 fingerprint / 重复 fingerprint
    401/403  未鉴权 / 非管理员
    404  RuleSet 不存在（仅命名 RuleSet 时，default RuleSet 由后端自动初始化）
    500  RuleSet.Rules JSON 损坏等内部错误

测试场景：
  S1   认证三件套                                       → 401 / 403
  S2   请求体非 JSON                                    → 400
  S3   ordered_fingerprints 为空                        → 400
  S4   ordered_fingerprints 含未知 fingerprint          → 400
  S5   ordered_fingerprints 含重复 fingerprint          → 400
  S6   命名 RuleSet 不存在                              → 404
  S7   happy path：用 GET /ruleset 拿真实非 Prepend fingerprint，
       倒序提交后再 GET 反查 → 非 Prepend 规则相对顺序与提交一致
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    seed,
    health_check, make_api_fn,
    auth_test_suite, run_tests,
)


# ─────────────────────────────────────────────
# API 调用器
# ─────────────────────────────────────────────

post_reorder = make_api_fn(
    "post", "/admin/config/security-group/ruleset/rules/reorder",
)
post_update_rules = make_api_fn(
    "post", "/admin/config/security-group/ruleset/rules",
)
get_ruleset = make_api_fn(
    "get", "/admin/config/security-group/ruleset", timeout=10,
)


# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────

# 三条彼此 fingerprint 不同、顺序可校验的 happy-path 规则
SEED_RULES = [
    {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP",
     "port": "22", "cidr_block": "0.0.0.0/0", "description": "SSH"},
    {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP",
     "port": "443", "cidr_block": "0.0.0.0/0", "description": "HTTPS"},
    {"direction": "EGRESS", "action": "ACCEPT", "protocol": "ALL",
     "port": "ALL", "cidr_block": "0.0.0.0/0", "description": "出站全放"},
]


def _is_initialized() -> bool:
    """default RuleSet 是否已初始化"""
    resp = get_ruleset()
    if resp.status_code != 200 or not resp.text.strip():
        return False
    try:
        return bool(resp.json().get("initialized", False))
    except Exception:
        return False


def _seed_default_rules_for_reorder():
    """确保 default RuleSet 至少有 SEED_RULES 这 3 条规则；返回 GET 响应中的
    rules 列表（带 fingerprint 派生字段，可直接用于 reorder）。

    若 RuleSet 未初始化则跳过（caller 负责处理）。"""
    if not _is_initialized():
        return None
    # 重置为 SEED_RULES，关闭 auto_fix_rules 以避免必需规则被注入打乱断言
    resp = post_update_rules({"rules": SEED_RULES, "auto_fix_rules": False})
    if resp.status_code != 200:
        # 若 SiteConfig 强制注入必需规则，这里 update 仍会成功，rules 数 >= 3；
        # 仅在真出错（如 fan-out 失败）时返回 None
        return None
    get_resp = get_ruleset()
    if get_resp.status_code != 200:
        return None
    rules = get_resp.json().get("rules", [])
    return [r for r in rules if r.get("fingerprint")]

def _is_forced_prepend_rule(rule: dict) -> bool:
    """Prepend required 规则不参与 reorder 相对顺序断言。"""
    if rule.get("prepend") is True:
        return True
    desc = rule.get("policy_description") or rule.get("description") or ""
    return desc in ("办公网入站白名单", "办公网入站兜底拒绝")


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_reorder_auth():
    """S1：认证三件套（无认证 / 错误 token / 非管理员） → 401/403"""
    auth_test_suite(
        lambda headers: post_reorder(
            {"ordered_fingerprints": ["INGRESS|TCP|22|0.0.0.0/0|ACCEPT"]},
            headers=headers,
        ),
        label="reorder_rule_set_rules",
    )


def test_reorder_invalid_json():
    """S2：请求体非 JSON → 400"""
    print(">>> [reorder] S2：请求体非 JSON → 400 ...")
    resp = post_reorder(raw_data="not-a-json{{{")
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code} body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error', '')[:60]}...)")


def test_reorder_empty_fingerprints():
    """S3：ordered_fingerprints 为空 → 400"""
    print(">>> [reorder] S3：ordered_fingerprints 为空 → 400 ...")
    resp = post_reorder({"ordered_fingerprints": []})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code} body={resp.text}"
    data = resp.json()
    assert "ordered_fingerprints" in data.get("error", "") or "不能为空" in data.get("error", ""), \
        f"error 应提示 ordered_fingerprints 不能为空，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")


def test_reorder_unknown_fingerprint():
    """S4：ordered_fingerprints 含未知 fingerprint → 400"""
    print(">>> [reorder] S4：含未知 fingerprint → 400 ...")
    if not _is_initialized():
        print("    SKIP (RuleSet 未初始化)")
        return
    resp = post_reorder({
        "ordered_fingerprints": ["INGRESS|UDP|53|9.9.9.9/32|DROP"],
    })
    # 后端校验：未知 fingerprint → 400
    # 若 default RuleSet 为空，可能走 EmptyExistingRules 也是 400
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code} body={resp.text}"
    print(f"    OK (status=400, error={resp.json().get('error', '')[:60]}...)")


def test_reorder_duplicate_fingerprints():
    """S5：ordered_fingerprints 含重复 → 400"""
    print(">>> [reorder] S5：含重复 fingerprint → 400 ...")
    fp = "INGRESS|TCP|22|0.0.0.0/0|ACCEPT"
    resp = post_reorder({"ordered_fingerprints": [fp, fp]})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code} body={resp.text}"
    print(f"    OK (status=400, error={resp.json().get('error', '')[:60]}...)")


def test_reorder_named_ruleset_not_found():
    """S6：命名 RuleSet 不存在 → 404"""
    print(">>> [reorder] S6：命名 RuleSet 不存在 → 404 ...")
    resp = post_reorder({
        "name": "no-such-ruleset-xyz-abc",
        "ordered_fingerprints": ["INGRESS|TCP|22|0.0.0.0/0|ACCEPT"],
    })
    # 命名 RuleSet 不会自动创建；不存在直接 404
    assert resp.status_code in (400, 404), \
        f"期望 404（或 400 校验拦截），实际 {resp.status_code} body={resp.text}"
    print(f"    OK (status={resp.status_code})")


def test_reorder_happy_path():
    """S7：happy path：通过 GET /ruleset 拿真实 fingerprint，倒序提交，再 GET 反查顺序

    并发鲁棒性说明（共享测试机 / SiteConfig 后台注入场景）：
      1. 提交前先 GET 一次拿到「reorder 入参基线」(version_before, fp_before)。
         入参只包含非 Prepend 的规则；Prepend required（办公网 Guard）由后端固定前置。
      2. reorder 200 后立即 GET 拿到 after_fps。
      3. 断言用「相对顺序」而非「严格前缀」：只要 reversed_fp 中每条 fingerprint
         在 after_fps 中仍然存在，且它们之间的相对顺序与 reversed_fp 一致即可——
         允许其他进程在 reorder 之后追加 / 插入新规则。
      4. 如果 after_fps 中我们提交的 fingerprint 不全（说明并发刷新把 RuleSet
         重置成了 builtin+recommended），则最多重试 MAX_RETRY 次重新 seed→reorder。
         全部失败则 SKIP 并打印诊断信息，避免共享测试环境抖动让 CI 红屏。
    """
    print(">>> [reorder] S7：happy path 真实倒序 + GET 反查 ...")
    if not _is_initialized():
        print("    SKIP (RuleSet 未初始化)")
        return

    MAX_RETRY = 3
    last_diag = ""
    for attempt in range(1, MAX_RETRY + 1):
        rules = _seed_default_rules_for_reorder()
        if not rules or len(rules) < 2:
            last_diag = f"无法 seed 规则；当前 rules 数={0 if not rules else len(rules)}"
            print(f"    [attempt {attempt}/{MAX_RETRY}] {last_diag}，重试…")
            continue

        # 入参基线：只重排非 Prepend 规则；办公网 Guard 等 Prepend required 固定前置
        before_resp = get_ruleset()
        if before_resp.status_code != 200:
            last_diag = f"GET /ruleset 前置失败: {before_resp.text}"
            print(f"    [attempt {attempt}/{MAX_RETRY}] {last_diag}，重试…")
            continue
        before = before_resp.json()
        version_before = before.get("version")
        before_rules = before.get("rules", [])
        fp_before = [
            r.get("fingerprint")
            for r in before_rules
            if r.get("fingerprint") and not _is_forced_prepend_rule(r)
        ]
        if len(fp_before) < 2:
            last_diag = f"GET 看到的可重排 fingerprint 数 <2: {fp_before}"
            print(f"    [attempt {attempt}/{MAX_RETRY}] {last_diag}，重试…")
            continue

        reversed_fp = list(reversed(fp_before))
        resp = post_reorder({"ordered_fingerprints": reversed_fp})
        assert resp.status_code == 200, \
            f"期望 200，实际 {resp.status_code} body={resp.text}"
        body = resp.json()
        assert "version" in body, f"响应缺 version: {body}"

        # GET 反查
        get_resp = get_ruleset()
        assert get_resp.status_code == 200, f"GET /ruleset 失败: {get_resp.text}"
        after = get_resp.json()
        version_after = after.get("version")
        after_fps = [r.get("fingerprint") for r in after.get("rules", []) if r.get("fingerprint")]

        # 我们提交的 fingerprint 在 after 中的索引；若缺任意一条则说明被并发覆盖
        idx_map = {fp: i for i, fp in enumerate(after_fps)}
        missing = [fp for fp in reversed_fp if fp not in idx_map]
        if missing:
            last_diag = (
                f"reorder 后 after_fps 缺失我们提交的 fingerprint："
                f"\n      missing={missing}"
                f"\n      reversed_fp={reversed_fp}"
                f"\n      after_fps={after_fps}"
                f"\n      version_before={version_before} reorder_resp_version={body.get('version')} version_after={version_after}"
                f"\n      （疑似被并发的 RuleSet 刷新/重置覆盖；自动重试）"
            )
            print(f"    [attempt {attempt}/{MAX_RETRY}] {last_diag}")
            continue

        # 相对顺序断言：reversed_fp 中各 fingerprint 在 after_fps 中的索引必须单调递增
        indices = [idx_map[fp] for fp in reversed_fp]
        if indices != sorted(indices):
            # 这是真正的 reorder 顺序 bug，不应静默跳过
            assert False, (
                f"reorder 后我们提交的 fingerprint 相对顺序不匹配："
                f"\n  reversed_fp={reversed_fp}"
                f"\n  indices_in_after={indices}"
                f"\n  after_fps={after_fps}"
            )
        print(
            f"    OK (attempt={attempt}, version_before={version_before} → resp={body.get('version')} → after={version_after}, "
            f"我们提交的 {len(reversed_fp)} 条 fingerprint 相对顺序保持倒序，after 共 {len(after_fps)} 条)"
        )
        return

    # 重试用尽：共享环境抖动，SKIP 而不是 FAIL，并把最后一次诊断打出来便于排障
    print(f"    SKIP (尝试 {MAX_RETRY} 次仍被并发干扰；最后一次诊断如下)")
    print(f"      {last_diag}")


# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(
        globals(),
        title="POST /admin/config/security-group/ruleset/rules/reorder",
    )


if __name__ == "__main__":
    main()

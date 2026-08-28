#!/usr/bin/env python3
"""
本地 agent — 门户接口测试

覆盖：
  1. GET /local-agent/availability → { enabled: bool }
     - 测试环境 local_agent_enabled 恒为 true → enabled=true
  2. GET /admin/local-agent-types → 至少含 workbuddy / codebuddy
  3. GET /admin/feature-allowlist/check（admin token = 超管，恒放行）
     - type=local-agent + 任意 identifier → in_allowlist=true
     - type=unknown-type → in_allowlist=true（超管绕过白名单）
     - 缺 identifier → 200 in_allowlist=true（identifier 由服务端按登录态取，不再前端传）
     - 缺 type → 400

不动 feature_allowlists 表本身（一期无 add/delete API）。
"""

import os
import sys
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers  # noqa: E402
from helpers import (  # noqa: E402
    check_env,
    setup_admin,
    setup_user,
    admin_client,
    enable_local_agent_feature,
    user_get_local_agent_availability,
    admin_get_local_agent_types,
    admin_check_feature_allowlist,
)


def main():
    check_env()
    print()

    admin = setup_admin("local-avail")
    user = None

    try:
        user = setup_user(admin.token, "local-avail")

        # ══════════════════════════════════════════════════════════════
        # 用例 1：availability 随 SiteConfig 切换
        # ══════════════════════════════════════════════════════════════
        # 1a：enabled=true 时 availability=true
        print(">>> 用例 1a：local_agent_enabled=true → availability.enabled=true ...")
        enable_local_agent_feature(admin.token)
        data = user_get_local_agent_availability(user.token)
        assert data.get("enabled") is True, (
            f"enabled 应 true，实际={data}"
        )
        print("    enabled=true ✓")

        # ══════════════════════════════════════════════════════════════
        # 用例 2：GET /admin/local-agent-types 至少含 workbuddy + codebuddy
        # ══════════════════════════════════════════════════════════════
        print(">>> 用例 2：/admin/local-agent-types ...")
        data = admin_get_local_agent_types(admin.token)
        types = data.get("local_agent_types", [])
        codes = {t.get("code") for t in types}
        for expected in ("workbuddy", "codebuddy"):
            assert expected in codes, (
                f"local_agent_types 应含 code={expected}，实际={codes}"
            )
        # 每一项都应有 code / name / description
        for t in types:
            assert t.get("code") and t.get("name"), (
                f"每项应含 code/name: {t}"
            )
        print(f"    含 codes={codes} ✓")

        # ══════════════════════════════════════════════════════════════
        # 用例 3：feature-allowlist/check
        # ══════════════════════════════════════════════════════════════
        print(">>> 用例 3a：check type=local-agent（空表全开）...")
        data = admin_check_feature_allowlist(
            admin.token, "local-agent", "any-tenant-identifier",
        )
        # 「空表为全开」：表为空时 in_allowlist=true, empty_table_means_allow=true
        assert data.get("in_allowlist") is True, (
            f"空表应 in_allowlist=true: {data}"
        )
        # empty_table_means_allow 是可选字段，若返回则应为 true
        if "empty_table_means_allow" in data:
            assert data["empty_table_means_allow"] is True, (
                f"空表应 empty_table_means_allow=true: {data}"
            )
        print(f"    in_allowlist=true, empty_table_means_allow="
              f"{data.get('empty_table_means_allow')} ✓")

        print(">>> 用例 3b：check type=unknown-type → 空表全开（200 in_allowlist=true）...")
        data = admin_check_feature_allowlist(
            admin.token, "totally-unknown-feature", "x",
        )
        # 未知 type 表为空 → 空表全开，返 200 + in_allowlist=true
        assert data.get("in_allowlist") is True, (
            f"未知 type 空表应 in_allowlist=true: {data}"
        )
        print(f"    in_allowlist=true ✓")

        print(">>> 用例 3c：check 缺 type → 400 ...")
        resp = admin_client(admin.token).get(
            "/admin/feature-allowlist/check",
            params={"identifier": "x"},
            expect=None, raw=True,
        )
        assert resp.status_code == 400, (
            f"缺 type 应 400，实际={resp.status_code}"
        )
        print("    400 ✓")

        print(">>> 用例 3d：check 缺 identifier → 200 in_allowlist=true "
              "（identifier 由服务端按登录态取，前端可不传）...")
        resp = admin_client(admin.token).get(
            "/admin/feature-allowlist/check",
            params={"type": "local-agent"},
            expect=None, raw=True,
        )
        assert resp.status_code == 200, (
            f"缺 identifier 应 200，实际={resp.status_code}"
        )
        body = resp.json()
        assert body.get("in_allowlist") is True, (
            f"缺 identifier 应 in_allowlist=true: {body}"
        )
        print("    200 in_allowlist=true ✓")

        print()
        print("本地 agent 门户接口测试通过 ✅")

    except Exception as e:
        # 兜底恢复 SiteConfig，避免污染并发测试
        try:
            enable_local_agent_feature(admin.token)
        except Exception:
            pass
        print(f"\n本地 agent 门户接口测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()

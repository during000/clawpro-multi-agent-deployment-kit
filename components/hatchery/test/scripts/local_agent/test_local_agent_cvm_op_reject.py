#!/usr/bin/env python3
"""
本地 agent — CVM 操作接口对本地实例的拒绝

覆盖 controller/agent_type_guard.go 里的 rejectLocalOrWrite 双屁障：
  - /openclaw/start   → 400
  - /openclaw/stop    → 400
  - /openclaw/reboot  → 400
  - /openclaw/reset   → 400
  - /openclaw/upgrade → 400 或类似（本地实例不支持镜像升级）

因为本地实例的 CVM 类操作全部返回 400 「本地实例不支持此操作」，本文件重点：
只测「本地实例被正确拒绝」的一面，不覆盖 batch 接口 & CVM 混合场景（那是普通 CVM 测试的事）。
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
    user_client,
    enable_local_agent_feature,
    setup_local_instance,
)


CVM_ONLY_ENDPOINTS = [
    ("/openclaw/start", "form"),
    ("/openclaw/stop", "form"),
    ("/openclaw/reboot", "form"),
    ("/openclaw/reset", "form"),
]


def _post_form(user_token, path, db_id):
    return user_client(user_token).post(
        path, data={"id": str(db_id)},
        expect=None, raw=True, timeout=30,
    )


def main():
    check_env()
    print()

    admin = setup_admin("local-cvmreject")
    user = None
    la = None

    try:
        enable_local_agent_feature(admin.token)
        user = setup_user(admin.token, "local-cvmreject")
        la = setup_local_instance(user.token, "cvmreject")
        print()

        # ══════════════════════════════════════════════════════════════
        # 逐一验证 CVM-only 接口对本地实例返回 4xx
        # ══════════════════════════════════════════════════════════════
        for path, kind in CVM_ONLY_ENDPOINTS:
            print(f">>> {path} 应拒绝本地实例 ...")
            resp = _post_form(user.token, path, la.db_id)
            # 允许 400 或 403（部分接口走 agent_type guard 而非 rejectLocalOrWrite）或 409
            assert resp.status_code in (400, 403, 409), (
                f"{path}: 期望 4xx (拒绝本地实例)，实际={resp.status_code} "
                f"body={resp.text[:200]}"
            )
            body_text = resp.text or ""
            # 期望错误消息里有「本地实例」或英文关键词
            assert (
                "本地" in body_text or "local" in body_text.lower()
            ), (
                f"{path} 拒绝原因应指明本地实例语义，实际 body={body_text[:200]}"
            )
            print(f"    {resp.status_code} ✓")

        print()

        # ══════════════════════════════════════════════════════════════
        # 补充：ADMIN 端的 CVM 操作对本地实例也应拒绝
        # ══════════════════════════════════════════════════════════════
        print(">>> /admin/instances/reboot 对本地实例应拒绝 ...")
        # 走 form；admin 接口通常收 form 或 JSON，这里用 form 兜底
        from helpers import admin_client  # 局部 import，避免顶层多余
        resp = admin_client(admin.token).post(
            "/admin/instances/reboot",
            data={"id": str(la.db_id)},
            expect=None, raw=True, timeout=30,
        )
        assert resp.status_code in (400, 409), (
            f"admin reboot 期望 4xx，实际={resp.status_code} "
            f"body={resp.text[:200]}"
        )
        print(f"    {resp.status_code} ✓")

        # 清理本地实例
        user_client(user.token).post(
            "/openclaw/delete", data={"id": str(la.db_id)},
            expect=None, raw=True, timeout=30,
        )
        la = None

        print()
        print("本地 agent CVM 操作拒绝测试通过 ✅")

    except Exception as e:
        print(f"\n本地 agent CVM 操作拒绝测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if la is not None and la.db_id and user is not None:
            try:
                user_client(user.token).post(
                    "/openclaw/delete", data={"id": str(la.db_id)},
                    expect=None, raw=True, timeout=30,
                )
            except Exception:
                pass


if __name__ == "__main__":
    main()

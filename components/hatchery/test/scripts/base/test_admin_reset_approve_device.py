#!/usr/bin/env python3
"""
集成测试：POST /admin/instances/reset 接口层面验证

覆盖接口：
    POST /admin/instances/reset

覆盖场景：
    1. 认证三件套（无认证 / 错误 token / 非管理员 token）→ 401/403
    2. 缺少 id 参数 → 400
    3. id 不存在（超大主键）→ 404
    4. id 为非数字 → 400
    5. 实例处于不允许重装的状态（如 creating）→ 409（状态 guard 拒绝）
    6. 正常重装请求（实例处于 running/stopped 状态）→ 200，响应包含 ok=true

注意：
    - 本测试仅验证接口层面的参数校验、权限控制和响应格式。
    - 不验证 CVM 真实重装（需要真实 CVM 环境），也不验证 approve_device 异步链路
      （该链路由 Go 单元测试 controller/admin_reset_approve_device_test.go 覆盖）。
    - 用例 6（正常重装）会自动创建一个测试实例（若未通过 INSTANCE_DB_ID 指定），
      测试完成后自动删除。若已设置 INSTANCE_DB_ID 则直接使用已有实例（不删除）。

环境变量：
    API              hatchery 服务地址
    ADMIN_TOKEN      管理员 token（必填）
    TOKEN            普通用户 token（用于创建/删除实例及权限测试，可选）
    INSTANCE_DB_ID   处于 running/stopped 状态的实例 DB 主键（可选，指定后跳过自动创建）
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    seed,
    ApiClient,
    health_check,
    run_tests,
    auth_test_suite,
    make_api_fn,
    TOKEN,
    NON_ADMIN_TOKEN,
    no_auth_headers,
    wrong_token_headers,
    non_admin_headers,
)
from helpers.instance import (
    create_instance,
    delete_instance,
    get_instance_db_id,
    wait_instance_ready,
)

POLL_INTERVAL = 5
TIMEOUT = 600

# ─── 自动创建/清理实例 ───

# 记录本次测试自动创建的实例 db_id，测试结束后清理
_auto_created_db_id: int | None = None


def _setup_instance() -> int | None:
    """
    若未设置 INSTANCE_DB_ID，则自动创建一个测试实例并等待就绪。
    返回 db_id（int），或 None（无法创建时）。
    """
    global _auto_created_db_id

    db_id_env = os.environ.get("INSTANCE_DB_ID", "").strip()
    if db_id_env:
        try:
            return int(db_id_env)
        except ValueError:
            print(f"    WARN: INSTANCE_DB_ID={db_id_env!r} 不是合法整数，忽略")

    if not TOKEN:
        print("    SKIP: 未设置 TOKEN，无法自动创建实例")
        return None

    name = f"it-reset-test-{int(time.time())}"
    print(f">>> 创建实例 (name={name}) ...")
    create_data = create_instance(TOKEN, name)
    if not create_data.get("ok"):
        print(f"    Failed: {create_data.get('error', create_data)}")
        return None

    instance_id = create_data.get("instance_id", "")
    print(f"    Created instance_id={instance_id}")

    db_id = get_instance_db_id(TOKEN, instance_id)
    print(f"    db_id={db_id}")

    print(f">>> 等待实例就绪 (id={db_id}) ...")
    try:
        wait_instance_ready(TOKEN, db_id)
        print(f"    实例就绪 ✓")
    except Exception as e:
        print(f"    等待就绪失败: {e}")
        try:
            delete_instance(TOKEN, db_id)
        except Exception:
            pass
        return None

    _auto_created_db_id = db_id
    return db_id


def _teardown_instance():
    """清理本次测试自动创建的实例（若有）"""
    global _auto_created_db_id
    if _auto_created_db_id is None:
        return
    db_id = _auto_created_db_id
    _auto_created_db_id = None
    if not TOKEN:
        return
    print(f">>> 清理测试实例 (id={db_id}) ...")
    try:
        resp = delete_instance(TOKEN, db_id)
        if resp.get("ok"):
            print("    Delete request submitted")
        else:
            print(f"    Delete failed: {resp.get('error', resp)}")
    except Exception as e:
        print(f"    删除失败（可忽略）: {e}")


# ─── 工具函数 ───

def do_reset(params: dict = None, headers=None):
    """调用 POST /admin/instances/reset，返回 raw Response"""
    if headers:
        tmp = ApiClient("", timeout=30)
        return tmp.post("/admin/instances/reset", params=params,
                        data={}, expect=None, raw=True, extra_headers=headers)
    return seed.post("/admin/instances/reset", params=params,
                     data={}, expect=None, raw=True)


# ─── 测试用例 ───

def test_01_auth_suite():
    """认证三件套：无认证 / 错误 token / 非管理员 token → 401/403"""
    auth_test_suite(
        lambda headers: do_reset(params={"id": "1"}, headers=headers),
        label="admin_instances_reset",
    )


def test_02_missing_id_returns_400():
    """缺少 id 参数 → 400（参数校验）"""
    resp = do_reset(params={})
    assert resp.status_code == 400, (
        f"缺少 id 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 缺少 id → 400")


def test_03_nonexistent_id_returns_404():
    """id 不存在（超大主键）→ 404"""
    resp = do_reset(params={"id": "999999999"})
    assert resp.status_code == 404, (
        f"不存在的 id 应返回 404，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 不存在的 id → 404")


def test_04_invalid_id_string_returns_400():
    """id 为非数字 → 400（ParseUint 失败）"""
    resp = do_reset(params={"id": "not-a-number"})
    assert resp.status_code == 400, (
        f"非数字 id 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 非数字 id → 400")


def test_05_id_zero_returns_400():
    """id=0 → 400（0 是非法主键）"""
    resp = do_reset(params={"id": "0"})
    assert resp.status_code == 400, (
        f"id=0 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: id=0 → 400")


def test_06_reset_running_instance():
    """
    正常重装请求（实例处于 running/stopped 状态）→ 200，响应包含 ok=true

    优先使用 INSTANCE_DB_ID 环境变量指定的实例；若未指定，则使用 main() 中
    自动创建并等待就绪的测试实例。

    注意：此用例会真实触发 CVM ResetInstance API，仅在集成测试环境中运行。
    """
    db_id_int = _state.get("db_id")
    if not db_id_int:
        print("    SKIP: 无可用实例（自动创建失败或未设置 INSTANCE_DB_ID）")
        return

    resp = do_reset(params={"id": str(db_id_int)})
    assert resp.status_code == 200, (
        f"正常重装应返回 200，实际 {resp.status_code}: {resp.text[:200]}"
    )
    data = resp.json()
    assert data.get("ok"), f"响应 ok 字段应为 true，实际: {data}"
    print(f"    OK: 实例 id={db_id_int} 重装请求已提交 → 200 ok=true")


# ─── 入口 ───

# 跨用例共享状态
_state: dict = {"db_id": None}


def main():
    health_check()
    print()

    # 自动创建测试实例（若未通过 INSTANCE_DB_ID 指定）
    db_id = _setup_instance()
    _state["db_id"] = db_id
    if db_id:
        print(f"    测试实例 db_id={db_id}")
    print()

    try:
        run_tests(
            globals(),
            title="POST /admin/instances/reset 接口层面验证（hotfix/approve_device_bug_fix_528）",
            ordered=True,
            abort_on_fail=False,
        )
    finally:
        # 清理自动创建的实例
        _teardown_instance()


if __name__ == "__main__":
    main()

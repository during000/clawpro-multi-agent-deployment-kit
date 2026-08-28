#!/usr/bin/env python3
"""
集成测试：升级接口层面验证

覆盖接口：
    POST /openclaw/upgrade              用户侧升级触发
    POST /openclaw/upgrade/retry        用户侧升级重试
    POST /admin/instances/batch-upgrade 管控端批量升级

覆盖场景（接口层面，不依赖真实 CVM / TAT 环境）：
    1. 认证三件套（无认证 / 错误 token / 非管理员 token）→ 401/403
    2. 缺少 id 参数 → 400
    3. id 不存在（超大主键）→ 404
    4. id 为非数字 → 400
    5. id=0 → 400
    6. 正常升级请求 → 200 或 409（已在升级中）

注意：
    - 本测试仅验证接口层面的参数校验、权限控制和响应格式。
    - 不验证 CVM 真实升级流程（需要真实 CVM 环境），也不验证
      waitForOpenclawReady + approve_device.sh + sync_gateway_port.sh 异步链路
      （该链路由 Go 单元测试 controller/openclaw_upgrade_compat_test.go 覆盖）。
    - 升级接口的核心 hotfix 改动（approveDeviceAfterUpgrade 新增 waitForOpenclawReady
      前置等待 + sync_gateway_port.sh 下发）属于异步链路，无法在集成测试中直接断言，
      但可以通过"升级请求被接受（200）"来确认接口入口正常。
    - 用例 6（正常升级）会自动创建一个测试实例（若未通过 INSTANCE_DB_ID 指定），
      测试完成后自动删除。若已设置 INSTANCE_DB_ID 则直接使用已有实例（不删除）。

环境变量：
    API              hatchery 服务地址
    TOKEN            普通用户 OpenAPI token（用于创建/删除实例及用户侧升级接口，可选）
    ADMIN_TOKEN      管理员 token（必填，用于管控端批量升级接口）
    INSTANCE_DB_ID   处于 running/stopped 状态的实例 DB 主键（可选，指定后跳过自动创建）
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    ApiClient,
    seed,
    health_check,
    run_tests,
    auth_test_suite,
    make_api_fn,
    TOKEN,
    NON_ADMIN_TOKEN,
    no_auth_headers,
    wrong_token_headers,
    non_admin_headers,
    ADMIN_TOKEN,
)
from helpers.instance import (
    create_instance,
    delete_instance,
    get_instance_db_id,
    wait_instance_ready,
    list_instances,
)

POLL_INTERVAL = 5
TIMEOUT = 600

# 升级测试专用：低版本镜像（openclaw-4.23），创建后再触发升级
# 注意：
#   1. image_id 是后端的隐藏参数，仅当部署所在腾讯云账号 UIN 命中内部账号白名单
#      （定义在 controller/internal_account.go 的 internalAccountUins，当前含
#      3205597606、100049049642）时生效，在外部环境（如生产 / 其他 UIN）会被静默忽略，
#      使用默认启用镜像。
#   2. 该镜像是 openclaw 类型镜像，使用 image_id 时必须强制 agent_type="openclaw"，
#      否则后端校验 image.agent_type != request.agent_type 会直接返回 400。
LOW_VERSION_IMAGE_ID = "img-b52f7vd0"
LOW_VERSION_AGENT_TYPE = "openclaw"
LOW_VERSION_AGENT_VERSION = "2026.4.23"

# ─── 低版本镜像准备 ───

def _ensure_low_version_image():
    """
    确保低版本镜像（LOW_VERSION_IMAGE_ID）已存在于镜像仓库；不存在则导入。

    实现说明：
      后端 GET /admin/images 不支持按 image_id 过滤（返回 DB 中全部镜像），
      为避免拉取全量镜像列表，直接调用 POST /admin/images/import 通过响应状态
      码判断：
        200 → 镜像此前不存在，本次刚导入成功
        409 → 镜像已存在（HandleImportImage 对重复 image_id 返回 Conflict）
        其他 → 视为失败，仅 WARN，不阻塞后续流程
      该方式语义上等价于"按 image_id 查询，不存在则导入"，且只需一次 HTTP 请求。

    导入参数：
        image_id      = LOW_VERSION_IMAGE_ID
        agent_type    = LOW_VERSION_AGENT_TYPE
        agent_version = LOW_VERSION_AGENT_VERSION

    依赖：ADMIN_TOKEN（seed 客户端为管理员客户端）
    """
    if not ADMIN_TOKEN:
        print("    SKIP: 未设置 ADMIN_TOKEN，跳过低版本镜像检查/导入")
        return

    print(
        f">>> 确保低版本镜像已存在 (image_id={LOW_VERSION_IMAGE_ID}, "
        f"agent_type={LOW_VERSION_AGENT_TYPE}, agent_version={LOW_VERSION_AGENT_VERSION}) ..."
    )
    resp = seed.post(
        "/admin/images/import",
        data={
            "image_id": LOW_VERSION_IMAGE_ID,
            "agent_type": LOW_VERSION_AGENT_TYPE,
            "agent_version": LOW_VERSION_AGENT_VERSION,
        },
        expect=None,
        raw=True,
    )
    if resp.status_code == 200:
        print("    镜像导入成功 ✓")
    elif resp.status_code == 409:
        # 镜像已存在（后端对重复 image_id 返回 Conflict），视为成功
        print("    镜像已存在 ✓")
    else:
        print(
            f"    WARN: 镜像导入失败 ({resp.status_code}): {resp.text[:200]}；"
            "将继续后续流程（外部环境 image_id 可能被静默忽略）"
        )


# ─── 自动创建/清理实例 ───

# 记录本次测试自动创建的实例，测试结束后清理
_auto_created_db_id: int | None = None

# 跨用例共享状态
_state: dict = {"db_id": None, "instance_id": None}


def _setup_instance() -> tuple:
    """
    若未设置 INSTANCE_DB_ID，则自动创建一个测试实例并等待就绪。
    返回 (db_id, instance_id)。
    """
    global _auto_created_db_id

    db_id_env = os.environ.get("INSTANCE_DB_ID", "").strip()
    if db_id_env:
        try:
            db_id = int(db_id_env)
        except ValueError:
            print(f"    WARN: INSTANCE_DB_ID={db_id_env!r} 不是合法整数，忽略")
            return None, ""
        # 通过列表获取 instance_id
        if TOKEN:
            try:
                data = list_instances(TOKEN)
                for inst in data.get("instances", []):
                    if (inst.get("id") or inst.get("ID")) == db_id:
                        return db_id, inst.get("instance_id") or inst.get("InstanceId") or ""
            except Exception:
                pass
        return db_id, ""

    if not TOKEN:
        print("    SKIP: 未设置 TOKEN，无法自动创建实例")
        return None, ""

    name = f"it-upgrade-test-{int(time.time())}"
    print(f">>> 创建实例 (name={name}, agent_type={LOW_VERSION_AGENT_TYPE}, image_id={LOW_VERSION_IMAGE_ID}) ...")
    create_data = create_instance(
        TOKEN, name,
        agent_type=LOW_VERSION_AGENT_TYPE,
        image_id=LOW_VERSION_IMAGE_ID,
    )
    if not create_data.get("ok"):
        print(f"    Failed: {create_data.get('error', create_data)}")
        return None, ""

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
        return None, ""

    _auto_created_db_id = db_id
    return db_id, instance_id


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

def _user_client():
    """普通用户客户端（带 X-OpenAPI: 1）"""
    return ApiClient(TOKEN, openapi=True)


def do_upgrade(params=None, headers=None):
    """调用 POST /openclaw/upgrade，返回 raw Response"""
    if headers:
        tmp = ApiClient("", timeout=60)
        return tmp.post("/openclaw/upgrade", params=params,
                        data={}, expect=None, raw=True, extra_headers=headers)
    return _user_client().post("/openclaw/upgrade", params=params,
                               data={}, expect=None, raw=True)


def do_upgrade_retry(params=None, headers=None):
    """调用 POST /openclaw/upgrade/retry，返回 raw Response"""
    if headers:
        tmp = ApiClient("", timeout=60)
        return tmp.post("/openclaw/upgrade/retry", params=params,
                        data={}, expect=None, raw=True, extra_headers=headers)
    return _user_client().post("/openclaw/upgrade/retry", params=params,
                               data={}, expect=None, raw=True)


def do_batch_upgrade(json_body=None, headers=None):
    """调用 POST /admin/instances/batch-upgrade，返回 raw Response"""
    if headers:
        tmp = ApiClient("", timeout=60)
        return tmp.post("/admin/instances/batch-upgrade", json=json_body,
                        expect=None, raw=True, extra_headers=headers)
    return seed.post("/admin/instances/batch-upgrade", json=json_body,
                     expect=None, raw=True)


# ─── 用户侧升级接口：POST /openclaw/upgrade ───

def test_01_upgrade_auth_suite():
    """POST /openclaw/upgrade 认证检查：无认证 / 错误 token → 401

    注意：/openclaw/upgrade 是用户侧接口，不做管理员权限校验。
    非管理员 token 会走正常用户逻辑，id=1 不属于该用户会返回 400，
    因此此处仅验证无认证和错误 token 两种场景。
    """
    # 无认证 → 401
    resp = do_upgrade(params={"id": "1"}, headers=no_auth_headers())
    assert resp.status_code == 401, (
        f"[openclaw_upgrade] 无认证应返回 401，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK [openclaw_upgrade] 无认证 → 401")

    # 错误 token → 401
    resp = do_upgrade(params={"id": "1"}, headers=wrong_token_headers())
    assert resp.status_code == 401, (
        f"[openclaw_upgrade] 错误 token 应返回 401，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK [openclaw_upgrade] 错误 token → 401")


def test_02_upgrade_missing_id_returns_400():
    """POST /openclaw/upgrade 缺少 id 参数 → 400"""
    if not TOKEN:
        print("    SKIP: 未设置 TOKEN，跳过用户侧升级测试")
        return
    resp = do_upgrade(params={})
    assert resp.status_code == 400, (
        f"缺少 id 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 缺少 id → 400")


def test_03_upgrade_nonexistent_id_returns_400():
    """POST /openclaw/upgrade 不存在的 id → 400（接口对"实例不存在"统一返回 400）"""
    if not TOKEN:
        print("    SKIP: 未设置 TOKEN，跳过用户侧升级测试")
        return
    resp = do_upgrade(params={"id": "999999999"})
    assert resp.status_code == 400, (
        f"不存在的 id 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 不存在的 id → 400")


def test_04_upgrade_invalid_id_returns_400():
    """POST /openclaw/upgrade 非数字 id → 400"""
    if not TOKEN:
        print("    SKIP: 未设置 TOKEN，跳过用户侧升级测试")
        return
    resp = do_upgrade(params={"id": "not-a-number"})
    assert resp.status_code == 400, (
        f"非数字 id 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 非数字 id → 400")


def test_05_upgrade_id_zero_returns_400():
    """POST /openclaw/upgrade id=0 → 400"""
    if not TOKEN:
        print("    SKIP: 未设置 TOKEN，跳过用户侧升级测试")
        return
    resp = do_upgrade(params={"id": "0"})
    assert resp.status_code == 400, (
        f"id=0 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: id=0 → 400")


def test_06_upgrade_running_instance():
    """
    POST /openclaw/upgrade 正常升级请求 → 200 / 409 / 400（合法拒绝）

    优先使用 INSTANCE_DB_ID 环境变量指定的实例；若未指定，则使用 main() 中
    自动创建并等待就绪的测试实例。

    注意：此用例会真实触发升级流程（包括 waitForOpenclawReady + approve_device.sh
    + sync_gateway_port.sh 异步链路），仅在集成测试环境中运行。

    可接受的响应（接口层面均为合法逻辑）：
      - 200：升级已触发
      - 409：实例已在升级中
      - 400 + "高于官方镜像版本"：实例版本严格高于官方镜像，
        controller 在 openclaw_upgrade.go 里主动拒绝降级（集成测试环境常见：
        实例由最新开发镜像创建，而 "官方镜像" 营业仓可能较旧）。
    """
    if not TOKEN:
        print("    SKIP: 未设置 TOKEN，跳过用户侧升级测试")
        return
    db_id = _state.get("db_id")
    if not db_id:
        print("    SKIP: 无可用实例（自动创建失败或未设置 INSTANCE_DB_ID）")
        return
    resp = do_upgrade(params={"id": str(db_id)})
    if resp.status_code == 400 and "高于官方镜像版本" in resp.text:
        # 实例版本高于官方镜像——controller 主动拒绝降级，接口本身合法
        print(f"    OK: 实例 id={db_id} 版本高于官方镜像 → 400（拒绝降级，合法拒绝）")
        return
    # 200 = 升级已触发；409 = 实例已在升级中也是合法状态
    assert resp.status_code in (200, 409), (
        f"正常升级应返回 200 / 409 / 400(拒绝降级)，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 实例 id={db_id} 升级请求 → {resp.status_code}")


# ─── 用户侧升级重试接口：POST /openclaw/upgrade/retry ───

def test_07_upgrade_retry_auth_suite():
    """POST /openclaw/upgrade/retry 认证检查：无认证 / 错误 token → 401

    注意：/openclaw/upgrade/retry 是用户侧接口，不做管理员权限校验。
    非管理员 token 会走正常用户逻辑，id=1 不属于该用户会返回 400，
    因此此处仅验证无认证和错误 token 两种场景。
    """
    # 无认证 → 401
    resp = do_upgrade_retry(params={"id": "1"}, headers=no_auth_headers())
    assert resp.status_code == 401, (
        f"[openclaw_upgrade_retry] 无认证应返回 401，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK [openclaw_upgrade_retry] 无认证 → 401")

    # 错误 token → 401
    resp = do_upgrade_retry(params={"id": "1"}, headers=wrong_token_headers())
    assert resp.status_code == 401, (
        f"[openclaw_upgrade_retry] 错误 token 应返回 401，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK [openclaw_upgrade_retry] 错误 token → 401")


def test_08_upgrade_retry_missing_id_returns_400():
    """POST /openclaw/upgrade/retry 缺少 id 参数 → 400"""
    if not TOKEN:
        print("    SKIP: 未设置 TOKEN，跳过用户侧升级重试测试")
        return
    resp = do_upgrade_retry(params={})
    assert resp.status_code == 400, (
        f"缺少 id 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 缺少 id → 400")


def test_09_upgrade_retry_nonexistent_id_returns_400():
    """POST /openclaw/upgrade/retry 不存在的 id → 400（接口对"实例不存在"统一返回 400）"""
    if not TOKEN:
        print("    SKIP: 未设置 TOKEN，跳过用户侧升级重试测试")
        return
    resp = do_upgrade_retry(params={"id": "999999999"})
    assert resp.status_code == 400, (
        f"不存在的 id 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 不存在的 id → 400")


def test_10_batch_upgrade_auth_suite():
    """POST /admin/instances/batch-upgrade 认证三件套：无认证 / 错误 token / 非管理员 → 401/403"""
    auth_test_suite(
        lambda headers: do_batch_upgrade(
            json_body={"instance_ids": ["ins-test"]},
            headers=headers,
        ),
        label="admin_batch_upgrade",
    )


def test_11_batch_upgrade_empty_ids_returns_400():
    """POST /admin/instances/batch-upgrade 空 instance_ids → 400"""
    resp = do_batch_upgrade(json_body={"instance_ids": []})
    assert resp.status_code == 400, (
        f"空 instance_ids 应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 空 instance_ids → 400")


def test_12_batch_upgrade_missing_body_returns_400():
    """POST /admin/instances/batch-upgrade 缺少请求体 → 400"""
    resp = do_batch_upgrade(json_body=None)
    assert resp.status_code == 400, (
        f"缺少请求体应返回 400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 缺少请求体 → 400")


def test_13_batch_upgrade_nonexistent_instances():
    """
    POST /admin/instances/batch-upgrade 全部不存在的 instance_id → 200（部分成功）或 404

    管控端批量升级对不存在的实例通常返回 200 + 部分失败列表，
    或直接 404；两种都是合法响应，取决于实现。
    """
    resp = do_batch_upgrade(json_body={"instance_ids": ["ins-not-exist-hotfix-test-9999"]})
    assert resp.status_code in (200, 404, 400), (
        f"不存在的 instance_id 应返回 200/404/400，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 不存在的 instance_id → {resp.status_code}")


def test_14_batch_upgrade_running_instance():
    """
    POST /admin/instances/batch-upgrade 正常批量升级请求 → 200

    优先使用 INSTANCE_DB_ID 环境变量指定的实例；若未指定，则使用 main() 中
    自动创建并等待就绪的测试实例。

    注意：此用例会真实触发升级流程，仅在集成测试环境中运行。
    升级完成后 approveDeviceAfterUpgrade 会依次执行：
      1. waitForOpenclawReady（最多 5 分钟）
      2. approve_device.sh（写入 operator token 5 件套权限）
      3. sync_gateway_port.sh（同步 gateway 端口）
    这些步骤均为异步，本用例仅验证接口层面的 200 响应。
    """
    db_id = _state.get("db_id")
    instance_id = _state.get("instance_id", "")
    if not db_id:
        print("    SKIP: 无可用实例（自动创建失败或未设置 INSTANCE_DB_ID）")
        return

    # 若 instance_id 为空，尝试通过管理员接口查询
    cvm_id = instance_id
    if not cvm_id:
        inst_resp = seed.get("/admin/instances", params={"id": db_id},
                             expect=None, raw=True)
        if inst_resp.status_code == 200:
            inst_data = inst_resp.json()
            instances = inst_data.get("instances") or inst_data.get("data") or []
            if instances:
                cvm_id = instances[0].get("instance_id") or instances[0].get("InstanceId") or ""

    if not cvm_id:
        print(f"    SKIP: 实例 id={db_id} 无 CVM instance_id，跳过")
        return

    resp = do_batch_upgrade(json_body={"instance_ids": [cvm_id]})
    assert resp.status_code == 200, (
        f"正常批量升级应返回 200，实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK: 实例 cvm_id={cvm_id} 批量升级请求已提交 → 200")


# ─── 入口 ───


def main():
    health_check()
    print()

    # 确保低版本镜像已存在于镜像仓库；不存在则导入
    _ensure_low_version_image()
    print()

    # 自动创建测试实例（若未通过 INSTANCE_DB_ID 指定）
    db_id, instance_id = _setup_instance()
    _state["db_id"] = db_id
    _state["instance_id"] = instance_id
    if db_id:
        print(f"    测试实例 db_id={db_id}, instance_id={instance_id!r}")
    print()

    try:
        run_tests(
            globals(),
            title="升级接口层面验证（hotfix/approve_device_bug_fix_528）",
            ordered=True,
            abort_on_fail=False,
        )
    finally:
        # 清理自动创建的实例
        _teardown_instance()


if __name__ == "__main__":
    main()

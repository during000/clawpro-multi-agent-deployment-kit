#!/usr/bin/env python3
"""
TC-MCP-REDELIVER 企业 MCP 重装后再次下发

验证修复点 controller/openclaw_reinstall.go::clearEnterpriseDistributionRecords：
实例重装时清空企业 MCP 安装记录（McpInstallation），使重装后实例在管控端 MCP 下发页
恢复为 uninstalled，企业 MCP 可被再次下发，且 /admin/mcp/instances 仍能查到这台实例。

完整链路（单文件闭环，自建实例 + 自动销毁）：
    1. 创建实例（POST /openclaw/create）→ 等待 CVM running + Agent 服务就绪
    2. 管控端创建企业 MCP（POST /admin/mcp/create，自动生成 1.0.0 版本）
    3. 下发企业 MCP 到实例（POST /admin/mcp/distribute）
         → /admin/mcp/instances 可查到实例，且 status 离开 uninstalled
         → 等待首次下发收敛（释放 mcp_distribute 分布式锁）
    4. 重装实例（POST /openclaw/reset）→ 走完整个重装生命周期并恢复正常运行
    5. 重装后核心校验：
         a. /admin/mcp/instances 仍能查到这台实例
         b. 该实例 MCP status == uninstalled（安装记录已被 clearEnterpriseDistributionRecords 清空）
    6. 再次下发同一企业 MCP（POST /admin/mcp/distribute）
         → /admin/mcp/instances 可查到实例，status 重新离开 uninstalled（验证可再次下发）
    7. 销毁实例（POST /openclaw/delete）—— 无论成功失败都在 finally 中清理

说明：
    - 覆盖三个接口：/admin/mcp/create、/admin/mcp/distribute、/admin/mcp/instances。
    - 会真实触发 CVM 创建 / ResetInstance / 销毁，仅可在集成测试环境中运行。

使用方式：
    export API=http://134.175.254.166
    export ADMIN_TOKEN=xxx          # 种子管理员 Token（用于创建测试管理员/用户）
    python3 test_admin_mcp_redelivery_after_reinstall.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import check_env, setup_admin, setup_user
from helpers.api import user_client
from helpers.admin_mcp import (
    admin_create_mcp,
    admin_distribute_mcp,
    admin_mcp_find_instance,
    wait_mcp_distributed,
    wait_mcp_settled,
    wait_mcp_instance_status,
)

POLL_INTERVAL = config.POLL_INTERVAL
REINSTALL_TIMEOUT = 900   # 与后端 OperationTimeouts[reinstall]=900s 对齐
READY_TIMEOUT = 900       # 创建 / 重装后 CVM running 上限
SERVICE_TIMEOUT = 300     # CVM running 后，Agent 服务恢复就绪上限

MCP_NAME = "企业MCP重装复测"
MCP_VERSION = "1.0.0"      # /admin/mcp/create 默认生成的初始版本


# ═══════════════════════════════════════════════════════════════════
# 实例生命周期辅助（参考 base/test_reinstall_openclaw.py，改为抛异常便于 finally 清理）
# ═══════════════════════════════════════════════════════════════════

def get_status(client, db_id):
    return client.get("/openclaw/status", params={"id": db_id})


def create_instance(client, name):
    print(f">>> 创建实例 (name={name}) ...")
    resp = client.post("/openclaw/create", data={"name": name}, timeout=60, raw=True)
    data = resp.json()
    if not data.get("ok"):
        raise RuntimeError(f"创建实例失败: {data.get('error', data)}")
    instance_id = data.get("instance_id")
    print(f"    已创建 instance_id={instance_id}")

    # 从列表中找到刚创建的实例，拿到 DB 自增 id
    listing = client.get("/openclaw/list", params={"page_size": 100})
    for inst in listing.get("instances", []):
        iid = inst.get("instance_id") or inst.get("InstanceId")
        if iid == instance_id:
            db_id = inst.get("id") or inst.get("ID")
            print(f"    db_id={db_id}, instance_id={instance_id}")
            return db_id, instance_id
    raise RuntimeError("创建后未在实例列表中找到该实例")


def wait_for_ready(client, db_id, stage="", allow_stopped=False):
    """轮询等待实例稳定运行（status==running 且非过渡态）。

    allow_stopped：重装会先关机（STOPPED）再开机，关机过渡态需继续等待回到 running。
    """
    print(f">>> 等待实例就绪{stage} (db_id={db_id}) ...")
    start = time.time()
    last_status = None
    terminal_failures = ("create_failed", "destroyed", "load_failed")
    if not allow_stopped:
        terminal_failures = terminal_failures + ("stopped",)

    while True:
        elapsed = time.time() - start
        if elapsed > READY_TIMEOUT:
            raise TimeoutError(f"等待就绪超时（{READY_TIMEOUT}s），最后状态: {last_status}")

        status_data = get_status(client, db_id)
        status = status_data.get("status", "unknown")
        label = status_data.get("label", "")

        if status != last_status:
            print(f"    [{int(elapsed)}s] status: {status} ({label})", flush=True)
            last_status = status

        if status == "running" and not status_data.get("transient", True):
            print(f"    实例就绪，耗时 {int(elapsed)}s ✓")
            return status_data

        if status == "":
            raise RuntimeError(f"实例 DB 行消失{stage}: {status_data.get('tooltip', '')}")

        if status in terminal_failures:
            raise RuntimeError(f"实例进入终态: {status} ({label}) - {status_data.get('tooltip', '')}")

        time.sleep(POLL_INTERVAL)


def get_service_status(client, db_id):
    resp = client.get("/openclaw/check-openclaw-port",
                      params={"id": db_id}, timeout=120, raw=True)
    if resp.status_code != 200:
        return None
    try:
        return resp.json()
    except Exception:
        return None


def wait_for_service_ready(client, db_id, stage=""):
    print(f">>> 等待 Agent 服务就绪{stage} (db_id={db_id}) ...")
    start = time.time()
    while True:
        elapsed = time.time() - start
        if elapsed > SERVICE_TIMEOUT:
            raise TimeoutError(f"等待 Agent 服务就绪超时（{SERVICE_TIMEOUT}s）")
        data = get_service_status(client, db_id)
        if data and data.get("running"):
            print(f"    Agent 服务就绪，耗时 {int(elapsed)}s ✓")
            return data
        print(f"    [{int(elapsed)}s] running={data.get('running') if data else 'N/A'}, 继续等待 ...", flush=True)
        time.sleep(POLL_INTERVAL)


def wait_for_reinstall_started(client, db_id):
    """重装下发后，确认实例确实离开 running 进入重装过渡态。"""
    print(f">>> 等待重装开始 (db_id={db_id}) ...")
    start = time.time()
    max_wait = 120
    while True:
        elapsed = time.time() - start
        if elapsed > max_wait:
            print(f"    WARN: 未捕获到明显的重装过渡态（{max_wait}s），继续等待就绪")
            return
        status_data = get_status(client, db_id)
        status = status_data.get("status", "unknown")
        if status != "running" or status_data.get("transient", False):
            print(f"    [{int(elapsed)}s] 重装已开始, status: {status}")
            return
        time.sleep(POLL_INTERVAL)


def reinstall_instance(client, db_id):
    print(f">>> 触发重装 (db_id={db_id}) ...")
    resp = client.post("/openclaw/reset", params={"id": db_id},
                       data={}, timeout=60, raw=True)
    if resp.status_code != 200:
        raise RuntimeError(f"重装下发失败: HTTP {resp.status_code}: {resp.text[:300]}")
    data = resp.json()
    if not data.get("ok"):
        raise RuntimeError(f"重装下发失败: {data.get('error', data)}")
    print("    重装请求已下发 (ok=true)")


def wait_for_reinstall_complete(client, db_id, stage="（重装后）"):
    """重装完成校验：先离开 running 进入过渡态，再回到稳定 running + Agent 服务 running。"""
    wait_for_reinstall_started(client, db_id)
    # 重装先关机（STOPPED）再开机，stopped 属关机过渡态，需继续等待回到 running。
    status_data = wait_for_ready(client, db_id, stage=stage, allow_stopped=True)
    service_data = wait_for_service_ready(client, db_id, stage=stage)

    if status_data.get("status") != "running" or status_data.get("transient", False):
        raise RuntimeError("重装后 CVM 未处于稳定 running 状态")
    if not service_data.get("running"):
        raise RuntimeError("重装后 Agent 服务未就绪")
    print("    重装完成，实例已恢复正常运行 ✓")


def delete_instance(client, db_id):
    print(f">>> 销毁实例 (db_id={db_id}) ...")
    submit_start = time.time()
    submit_deadline = 300
    retryable = ("操作进行中", "加载中")
    while True:
        resp = client.post("/openclaw/delete", data={"id": db_id}, raw=True, expect=None)
        try:
            data = resp.json()
        except Exception:
            data = {}
        if data.get("ok"):
            break
        err = str(data.get("error", resp.text[:200]))
        if any(k in err for k in retryable) and (time.time() - submit_start) < submit_deadline:
            print(f"    [{int(time.time() - submit_start)}s] 过渡态（{err}），等待后重试销毁 ...", flush=True)
            time.sleep(POLL_INTERVAL)
            continue
        raise RuntimeError(f"销毁失败: {err}")

    print("    销毁请求已下发，等待完成 ...")
    start = time.time()
    while True:
        if time.time() - start > READY_TIMEOUT:
            print("    WARN: 等待销毁完成超时（不阻塞清理）")
            return
        status = get_status(client, db_id).get("status", "unknown")
        if status in ("destroyed", ""):
            print(f"    实例已销毁，耗时 {int(time.time() - start)}s ✓")
            return
        time.sleep(POLL_INTERVAL)

# ═══════════════════════════════════════════════════════════════════
# 主流程
# ═══════════════════════════════════════════════════════════════════

def main():
    check_env()
    print()

    scenario = "ent-mcp-redeliver"
    # 预清理可能残留的同名测试用户（上次异常中断未收尾），避免创建用户 409
    helpers.teardown_scenario_users(scenario)

    admin = setup_admin(scenario)
    user = None
    client = None
    db_id = None
    failed = False
    service_id = f"e2e-mcp-redeliver-{int(time.time())}"

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        user = setup_user(admin.token, "ent-mcp-redeliver")
        client = user_client(user.token)

        # ── 步骤 1：创建实例并等待就绪 ──
        name = f"{config.INSTANCE_NAME_PREFIX}mcp-redeliver-{int(time.time())}"
        db_id, instance_id = create_instance(client, name)
        print()
        wait_for_ready(client, db_id)
        wait_for_service_ready(client, db_id)
        print()

        # 用 CVM 实例 id 作为 search 过滤，避免共享环境实例过多被分页截断
        search = instance_id

        # ── 步骤 2：管控端创建企业 MCP ──
        print(f">>> 步骤 2：创建企业 MCP service_id={service_id} version={MCP_VERSION} ...")
        resp = admin_create_mcp(admin.token, service_id, MCP_NAME)
        assert resp.status_code == 201, f"创建 MCP 失败: {resp.status_code} {resp.text[:300]}"
        assert resp.json().get("service_id") == service_id, f"创建 MCP 响应异常: {resp.json()}"
        print("    企业 MCP 创建成功 ✓")
        print()

        # ── 步骤 3：首次下发 + 校验实例可查到 ──
        print(">>> 步骤 3：首次下发企业 MCP 到实例 ...")
        resp = admin_distribute_mcp(
            admin.token, service_id, MCP_VERSION,
            select_all=True, statuses=["uninstalled"], search=search,
        )
        assert resp.status_code == 202, f"下发失败: {resp.status_code} {resp.text[:300]}"
        assert resp.json().get("task_id"), f"下发响应缺少 task_id: {resp.json()}"
        print(f"    下发任务创建 ✓  task_id={resp.json().get('task_id')}")

        inst_item = wait_mcp_distributed(admin.token, service_id, db_id, search=search)
        assert inst_item is not None, "首次下发后应能在 /admin/mcp/instances 查到实例"
        assert inst_item.get("status") != "uninstalled", \
            f"首次下发后 status 不应为 uninstalled，实际: {inst_item.get('status')}"
        print(f"    /admin/mcp/instances 可查到实例 ✓  status={inst_item.get('status')}")

        print(">>> 等待首次下发收敛（释放下发锁）...")
        settled = wait_mcp_settled(admin.token, service_id, db_id, search=search)
        print(f"    首次下发已收敛 ✓  status={settled.get('status')}")
        print()

        # ── 步骤 4：重装实例 ──
        reinstall_instance(client, db_id)
        wait_for_reinstall_complete(client, db_id)
        print()

        # ── 步骤 5：重装后核心校验 ──
        print(">>> 步骤 5：重装后校验 /admin/mcp/instances 仍可查到实例，且安装记录已清空 ...")
        # clearEnterpriseDistributionRecords 在 /openclaw/reset 处理内同步执行，
        # 这里短轮询以兼容 MySQL 主从/读写延迟。
        cleared = wait_mcp_instance_status(
            admin.token, service_id, db_id, ("uninstalled",), timeout=120, search=search,
        )
        assert cleared is not None, "重装后应仍能在 /admin/mcp/instances 查到实例"
        assert cleared.get("status") == "uninstalled", \
            f"重装后安装记录应被清空（status=uninstalled），实际: {cleared.get('status')}"
        print(f"    重装后实例仍可查到且安装记录已清空 ✓  status={cleared.get('status')}")
        print()

        # ── 步骤 6：再次下发，验证企业 MCP 可被重新下发 ──
        print(">>> 步骤 6：重装后再次下发企业 MCP ...")
        resp = admin_distribute_mcp(admin.token, service_id, MCP_VERSION, [db_id])
        assert resp.status_code == 202, f"再次下发失败: {resp.status_code} {resp.text[:300]}"
        assert resp.json().get("task_id"), f"再次下发响应缺少 task_id: {resp.json()}"
        print(f"    再次下发任务创建 ✓  task_id={resp.json().get('task_id')}")

        redelivered = wait_mcp_distributed(admin.token, service_id, db_id, search=search)
        assert redelivered is not None, "再次下发后应能在 /admin/mcp/instances 查到实例"
        assert redelivered.get("status") != "uninstalled", \
            f"再次下发后 status 不应为 uninstalled，实际: {redelivered.get('status')}"
        print(f"    重装后企业 MCP 可再次下发 ✓  status={redelivered.get('status')}")
        print()

        print("=" * 60)
        print("  TC-MCP-REDELIVER 企业 MCP 重装后再次下发 — 全部通过 ✓")
        print("=" * 60)

    except Exception:
        failed = True
        traceback.print_exc()
        print()
        print("=" * 60)
        print("  TC-MCP-REDELIVER 企业 MCP 重装后再次下发 — 失败 ✗")
        print("=" * 60)

    finally:
        # 无论成功失败都销毁本测试自建的实例
        if client is not None and db_id is not None:
            try:
                print()
                delete_instance(client, db_id)
            except Exception as ce:
                print(f"    清理实例时出错（忽略）: {ce}")
        # 销毁本测试自建的管理员 / 普通测试用户，保证可重复执行
        print(">>> 清理测试用户 ...")
        helpers.teardown_scenario_users(scenario)

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()

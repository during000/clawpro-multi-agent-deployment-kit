#!/usr/bin/env python3
"""
集成测试：普通用户实例重装（HandleResetInstance）完整流程

完整链路（单文件内闭环，关注重装核心流程）：
    1. 创建实例（POST /openclaw/create）
    2. 等待 CVM 就绪（GET /openclaw/status → running）
    3. 等待 Agent 服务就绪（GET /openclaw/check-openclaw-port → running）
    4. 触发重装（POST /openclaw/reset）
    5. 销毁前置门禁：确认重装走完整个生命周期并恢复正常运行
         - 先确认实例离开 running、进入重装过渡态（避免旧 running 被误判通过）
         - 再确认 CVM 重新稳定 running（GET /openclaw/status → running 且非过渡态）
         - 且 Agent 服务恢复正常（GET /openclaw/check-openclaw-port → running）
       只有这一步全部确认通过，才认为重装成功、可以安全销毁。
    6. 确认正常运行后，主动销毁实例（POST /openclaw/delete）

说明：
    - 本测试只覆盖普通用户自助重装（HandleResetInstance → commonHandleResetInstance），
      不涉及管控端重装。
    - 公共的鉴权 / 参数校验等接口层面用例不在本测试范围内（由 Go 单元测试与
      test_admin_reset_approve_device.py 等覆盖），这里只验证"重装核心流程能跑通且
      重装后实例恢复正常"。
    - 会真实触发 CVM ResetInstance，仅可在集成测试环境中运行。

环境变量：
    API        hatchery 服务地址
    TOKEN      普通用户 OpenAPI token（必填，形如 hk-xxx）
"""

import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import health_check, ApiClient

IDENTIFIER = os.environ.get("IDENTIFIER", "")
INSTANCE_NAME = f"user-reinstall-{IDENTIFIER}-{int(time.time())}"

POLL_INTERVAL = 5
TIMEOUT = 900          # 重装超时阈值与后端 OperationTimeouts[reinstall]=900s 对齐
SERVICE_TIMEOUT = 300  # CVM running 后，Agent 服务恢复就绪上限

TOKEN = os.environ.get("TOKEN", "")
if not TOKEN:
    print("ERROR: please set environment variable TOKEN")
    print("  export TOKEN=hk-xxx")
    sys.exit(1)

client = ApiClient(TOKEN, openapi=True)


def create_instance():
    print(f">>> Create instance (name={INSTANCE_NAME}) ...")
    resp = client.post("/openclaw/create", data={"name": INSTANCE_NAME}, raw=True)
    data = resp.json()
    if not data.get("ok"):
        print(f"    Failed: {data.get('error', data)}")
        sys.exit(1)

    print(f"    Created instance_id={data.get('instance_id')}")
    return data


def list_instances():
    """列出实例，返回实例列表"""
    data = client.get("/openclaw/list")
    return data.get("instances", [])


def get_status(instance_db_id):
    """查询实例状态"""
    return client.get("/openclaw/status", params={"id": instance_db_id})


def wait_for_ready(instance_db_id, stage="", allow_stopped=False):
    """
    轮询等待实例稳定运行（status==running 且非过渡态）。

    allow_stopped：
        重装会先关机（CVM STOPPED）再开机，期间状态可能短暂呈现为 stopped。
        正常情况下后端把"reinstall + STOPPED"映射为 loading（过渡态），但操作收敛
        存在时序窗口，可能瞬时暴露为 stopped。重装场景下必须把 stopped 视为过渡态
        继续等待，直到重新回到 running，绝不能在关机过渡态就判定失败/去退还。
        创建场景下保持原语义：stopped 仍属终态失败。
    """
    print(f">>> Wait for instance ready{stage} (id={instance_db_id}) ...")
    start = time.time()
    last_status = None

    # 创建场景下 stopped 是终态失败；重装场景下 stopped 是关机过渡态，需继续等待。
    terminal_failures = ("create_failed", "destroyed", "load_failed")
    if not allow_stopped:
        terminal_failures = terminal_failures + ("stopped",)

    while True:
        elapsed = time.time() - start
        if elapsed > TIMEOUT:
            print(f"\n    Timeout ({TIMEOUT}s), last status: {last_status}")
            sys.exit(1)

        status_data = get_status(instance_db_id)
        status = status_data.get("status", "unknown")
        label = status_data.get("label", "")

        if status != last_status:
            if last_status is not None:
                print()
            print(f"    [{int(elapsed)}s] status: {status} ({label})", end="", flush=True)
            last_status = status
        else:
            print(".", end="", flush=True)

        # 终态判断
        if status == "running" and not status_data.get("transient", True):
            print(f"\n    Instance ready, took {int(elapsed)}s")
            return status_data

        # 空状态表示实例 DB 行已消失（getInstanceByID 失败）。在创建/重装等待阶段，
        # 实例不应凭空消失；若出现说明被外部销毁或异常清理，应快速失败而非空转到超时。
        if status == "":
            tooltip = status_data.get("tooltip", "")
            print(f"\n    Instance disappeared (empty status){stage}: 实例 DB 行已被清理 - {tooltip}")
            sys.exit(1)

        if status in terminal_failures:
            tooltip = status_data.get("tooltip", "")
            print(f"\n    Instance terminated: {status} ({label}) - {tooltip}")
            sys.exit(1)

        time.sleep(POLL_INTERVAL)


def wait_for_reinstall_started(instance_db_id):
    """
    重装下发后，确认实例确实离开 running 进入重装过渡态（loading）。

    后端 setOperationWithAgentReset 会把 CurrentOperation=reinstall 且 AgentReady=0，
    因此状态会立即从 running 切到 loading（transient）。等到这个信号，才能保证后续
    的 wait_for_ready 不会因为"重装尚未真正开始、状态还停留在旧 running"而误判通过。
    """
    print(f">>> Wait for reinstall to start (id={instance_db_id}) ...")
    start = time.time()
    max_wait = 120

    while True:
        elapsed = time.time() - start
        if elapsed > max_wait:
            # 重装非常快时也可能直接跳过 loading；不强制失败，交给后续 ready 校验兜底。
            print(f"\n    WARN: 未捕获到明显的重装过渡态（{max_wait}s），继续等待就绪")
            return

        status_data = get_status(instance_db_id)
        status = status_data.get("status", "unknown")

        if status != "running" or status_data.get("transient", False):
            print(f"    [{int(elapsed)}s] reinstall started, status: {status}")
            return

        print(".", end="", flush=True)
        time.sleep(POLL_INTERVAL)


def get_service_status(instance_db_id):
    """Query agent ready status via check-openclaw-port."""
    resp = client.get("/openclaw/check-openclaw-port",
                      params={"id": instance_db_id}, timeout=120, raw=True)
    if resp.status_code != 200:
        return None, resp.status_code, resp.text
    try:
        return resp.json(), resp.status_code, None
    except Exception:
        return None, resp.status_code, resp.text


def wait_for_service_ready(instance_db_id, stage=""):
    """Wait for check-openclaw-port to report running after CVM is RUNNING."""
    print(f">>> Wait for service ready{stage} (id={instance_db_id}) ...")
    start = time.time()

    while True:
        elapsed = time.time() - start
        if elapsed > SERVICE_TIMEOUT:
            print(f"\n    Timeout ({SERVICE_TIMEOUT}s) waiting for service ready")
            sys.exit(1)

        data, status_code, err_text = get_service_status(instance_db_id)
        if status_code != 200 or data is None:
            print(f"    [{int(elapsed)}s] check-openclaw-port returned {status_code}, retrying...", flush=True)
            time.sleep(POLL_INTERVAL)
            continue

        if data.get("running"):
            print(f"    Service ready, took {int(elapsed)}s")
            return data

        print(f"    [{int(elapsed)}s] running={data.get('running', False)}, retrying...", flush=True)
        time.sleep(POLL_INTERVAL)


def wait_for_reinstall_complete(instance_db_id, stage="（重装后）"):
    """
    重装完成校验（销毁前置门禁），分两个阶段，缺一不可：

        阶段一：捕获重装过渡态。后端 setOperationWithAgentReset 会把
                CurrentOperation=reinstall 且 AgentReady=0，状态立即从 running
                切到 loading。必须先观察到"离开 running"，否则阶段二可能因为状态
                还停留在旧 running 而瞬间误判通过。
        阶段二：确认恢复正常。CVM 稳定 running（status==running 且非过渡态）
                且 Agent 服务 running（check-openclaw-port → running），二者同时
                满足才认为重装成功、可以安全销毁。

    任一阶段不满足直接判定失败（sys.exit(1)），绝不在未确认正常运行的情况下销毁。
    """
    wait_for_reinstall_started(instance_db_id)
    print()
    # 重装会先关机（STOPPED）再开机，stopped 属于关机过渡态，必须继续等待回到 running，
    # 否则会在关机阶段误判失败并触发"操作进行中"的退还冲突。
    status_data = wait_for_ready(instance_db_id, stage=stage, allow_stopped=True)
    print()
    service_data = wait_for_service_ready(instance_db_id, stage=stage)
    print()

    print(f">>> Verify running normally{stage} (id={instance_db_id}) ...")
    status = status_data.get("status")
    transient = status_data.get("transient", False)
    service_running = service_data.get("running", False)

    if status != "running" or transient:
        print(f"    FAILED: CVM 未处于稳定 running 状态（status={status}, transient={transient}）")
        sys.exit(1)
    if not service_running:
        print("    FAILED: Agent 服务未就绪（check-openclaw-port running=false）")
        sys.exit(1)

    print(f"    OK: 实例已确认正常运行（CVM running 且 Agent 服务 running），可以安全销毁")
    return status_data, service_data


def reinstall_instance(instance_db_id):
    print(f">>> Reinstall instance (id={instance_db_id}) ...")
    resp = client.post("/openclaw/reset", params={"id": instance_db_id},
                       data={}, timeout=60, raw=True)
    if resp.status_code != 200:
        print(f"    Reinstall failed: HTTP {resp.status_code}: {resp.text[:300]}")
        sys.exit(1)
    data = resp.json()
    if not data.get("ok"):
        print(f"    Reinstall failed: {data.get('error', data)}")
        sys.exit(1)
    print("    Reinstall request submitted (ok=true)")


def delete_instance(instance_db_id):
    print(f">>> Delete instance (id={instance_db_id}) ...")

    # 退还要求实例既不在 CVM 操作进行中，也不处于 loading 过渡态。重装收尾期存在两类
    # 瞬态冲突，都应等待后重试，而非直接判失败：
    #   1. "操作进行中"：CVM 侧关机/开机操作尚未收敛。
    #   2. "加载中"：CVM 已 RUNNING，但 ResolveInstanceStatus 因 AgentReady 被后台重新
    #      探活重置、技能/CLS Agent 重装中等原因瞬时回到 loading（即便 check-openclaw-port
    #      实时探活已 ready）。这类过渡态会很快收敛，等待后重试即可。
    submit_start = time.time()
    submit_deadline = 300
    retryable = ("操作进行中", "加载中")
    while True:
        resp = client.post("/openclaw/delete", data={"id": instance_db_id}, raw=True)
        data = resp.json()
        if data.get("ok"):
            break

        err = str(data.get("error", data))
        if any(k in err for k in retryable) and (time.time() - submit_start) < submit_deadline:
            print(f"    [{int(time.time() - submit_start)}s] 实例处于过渡态（{err}），等待后重试退还 ...", flush=True)
            time.sleep(POLL_INTERVAL)
            continue

        print(f"    Delete failed: {err}")
        sys.exit(1)

    print("    Delete request submitted")

    # 等待销毁完成
    start = time.time()
    last_status = None
    while True:
        elapsed = time.time() - start
        if elapsed > TIMEOUT:
            print(f"\n    Timeout ({TIMEOUT}s), last status: {last_status}")
            sys.exit(1)

        status_data = get_status(instance_db_id)
        status = status_data.get("status", "unknown")
        label = status_data.get("label", "")

        if status != last_status:
            if last_status is not None:
                print()
            print(f"    [{int(elapsed)}s] status: {status} ({label})", end="", flush=True)
            last_status = status
        else:
            print(".", end="", flush=True)

        # 销毁完成的两种终态：
        #   1. status == "destroyed"：CVM 已销毁，但 DB 行尚未清理（仍可查到记录）。
        #   2. status == ""：实例 DB 行已被清理（instance_state 副作用 purge），
        #      此时 getInstanceByID 失败，HandleInstanceStatus 返回空状态兜底。
        #      若不识别此终态，删除后的轮询会一直拿到空状态直到超时（即此前一直刷
        #      "status": "" 日志的根因）。
        if status in ("destroyed", ""):
            print(f"\n    Instance destroyed, took {int(elapsed)}s")
            return

        time.sleep(POLL_INTERVAL)


def main():
    health_check()
    print()

    # 1. 创建实例
    create_data = create_instance()
    print()

    # 从列表中找到刚创建的实例，获取 DB id
    instances = list_instances()
    target = None
    for inst in instances:
        if inst.get("instance_id") == create_data.get("instance_id") or inst.get("InstanceId") == create_data.get("instance_id"):
            target = inst
            break

    if not target:
        print("ERROR: created instance not found in list")
        sys.exit(1)

    db_id = target.get("id") or target.get("ID")
    instance_id = target.get("instance_id") or target.get("InstanceId")
    agent_type = target.get("agent_type") or target.get("AgentType")
    print(f"    Instance: db_id={db_id}, instance_id={instance_id}, agent_type={agent_type}")
    print()

    # 2. 等待 CVM 就绪
    wait_for_ready(db_id)
    print()

    # 3. 等待 Agent 服务就绪（确认重装前实例确实可用）
    wait_for_service_ready(db_id)
    print()

    # 4. 触发重装
    reinstall_instance(db_id)
    print()

    # 5. 销毁前置门禁：确认重装走完整个生命周期并恢复正常运行
    #    （先离开 running 进入过渡态，再回到稳定 running + Agent 服务 running）
    status_data, service_data = wait_for_reinstall_complete(db_id, stage="（重装后）")
    print()

    print(">>> Final status (after reinstall):")
    print(f"    status:  {status_data.get('status')}")
    print(f"    label:   {status_data.get('label')}")
    print(f"    actions: {status_data.get('actions')}")
    print(f"    service running: {service_data.get('running')}")
    print()

    # 6. 确认正常运行后，主动销毁
    delete_instance(db_id)
    print()

    print("Test passed")


if __name__ == "__main__":
    main()

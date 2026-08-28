#!/usr/bin/env python3
"""
Test script: create an OpenClaw instance, wait for it to be ready, then delete it.
"""

import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
from helpers.api import health_check, ApiClient

IDENTIFIER = os.environ.get("IDENTIFIER", "")
INSTANCE_NAME = f"it-{IDENTIFIER}-{int(time.time())}"

POLL_INTERVAL = 5
TIMEOUT = 600

# 使用统一 ApiClient（用户侧接口带 X-OpenAPI）
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


def wait_for_ready(instance_db_id):
    print(f">>> Wait for instance ready (id={instance_db_id}) ...")
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

        # 终态判断
        if status == "running":
            print(f"\n    Instance ready, took {int(elapsed)}s")
            return status_data

        if status in ("create_failed", "stopped", "destroyed", "load_failed"):
            tooltip = status_data.get("tooltip", "")
            print(f"\n    Instance terminated: {status} ({label}) - {tooltip}")
            sys.exit(1)

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


def wait_for_service_ready(instance_db_id):
    """Wait for check-openclaw-port to report running after CVM is RUNNING."""
    print(f">>> Wait for service ready (id={instance_db_id}) ...")
    start = time.time()
    max_wait = 180  # service should start within 3 minutes after CVM is RUNNING

    while True:
        elapsed = time.time() - start
        if elapsed > max_wait:
            print(f"\n    Timeout ({max_wait}s) waiting for service ready")
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


def delete_instance(instance_db_id):
    print(f">>> Delete instance (id={instance_db_id}) ...")
    resp = client.post("/openclaw/delete", data={"id": instance_db_id}, raw=True)
    data = resp.json()
    if not data.get("ok"):
        print(f"    Delete failed: {data.get('error', data)}")
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

        if status in ("destroyed", ""):
            print(f"\n    Instance destroyed, took {int(elapsed)}s (status={status!r})")
            return

        time.sleep(POLL_INTERVAL)


def main():
    health_check()
    print()

    # 创建实例
    create_data = create_instance()
    print()

    # 从列表中找到刚创建的实例，获取 DB id
    instances = list_instances()
    target = None
    for inst in instances:
        if inst.get("instance_id") or inst.get("InstanceId") == create_data.get("instance_id"):
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

    # 等待就绪
    status_data = wait_for_ready(db_id)
    print()

    # 确认服务真正可用
    service_data = wait_for_service_ready(db_id)
    print()

    print(">>> Final status:")
    print(f"    status:  {status_data.get('status')}")
    print(f"    label:   {status_data.get('label')}")
    print(f"    actions: {status_data.get('actions')}")
    print()

    # 删除实例
    delete_instance(db_id)
    print()

    print("Test passed")


if __name__ == "__main__":
    main()

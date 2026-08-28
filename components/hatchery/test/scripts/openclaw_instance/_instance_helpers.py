#!/usr/bin/env python3
"""
实例管理集成测试 — 实例特有的 helpers

本文件仅包含 helpers/ 公共框架未提供的"实例管理特有"工具：
  - 实例状态等待 (wait_for_status / wait_for_running / wait_for_destroyed /
                  wait_for_agent_ready)
  - 实例查询 (find_instance_by_db_id / get_status)
  - 业务断言 (assert_error_message / assert_json_keys)
  - 实例管理常量 (NONEXISTENT_DB_ID)
  - 共享实例入口 cli（透传到 _shared.py 的懒加载客户端）

通用 HTTP/帧记录/认证三件套/运行器/健康检查请直接使用 helpers/：
  from helpers.api import seed, anon, bad_token, auth_test_suite, run_tests, ...

本模块底层默认使用 _shared.shared_user_client()（普通用户视角，X-OpenAPI: 1）。
"""
import os
import sys
import time

# ── 把项目 test/scripts 加到 sys.path，便于 import helpers ──
_HERE = os.path.dirname(os.path.abspath(__file__))
_SCRIPTS_DIR = os.path.dirname(_HERE)
if _SCRIPTS_DIR not in sys.path:
    sys.path.insert(0, _SCRIPTS_DIR)

from helpers.client import truncate as _truncate  # noqa: E402

from _shared import (  # noqa: E402
    require_shared_instance,
    require_shared_db_id,
    shared_user_client,
    get_shared_db_id_or_none,
)

# ═══════════════════════════════════════════════════════════════════════════
# 实例管理常量
# ═══════════════════════════════════════════════════════════════════════════

# 状态等待相关
POLL_INTERVAL = int(os.environ.get("POLL_INTERVAL_SEC", "5"))
STATUS_TIMEOUT = int(os.environ.get("STATUS_TIMEOUT_SEC", "600"))
SERVICE_TIMEOUT = int(os.environ.get("SERVICE_TIMEOUT_SEC", "300"))

# 一次性使用的"伪造不存在的 db_id"（用于 404/参数校验测试）
NONEXISTENT_DB_ID = 999_999_999


# ═══════════════════════════════════════════════════════════════════════════
# 共享实例入口（向后兼容旧用例）
# ═══════════════════════════════════════════════════════════════════════════

def get_shared_db_id():
    """获取共享实例 db_id（无则返回 None）。

    注意：旧版本会按 INSTANCE_NAME_PREFIX 自动嗅探，本版本改为通过
    _shared.py 的文件锁懒加载机制；如果 SKIP_SHARED_INSTANCE=1 则不主动 setup。
    """
    return get_shared_db_id_or_none()


# ═══════════════════════════════════════════════════════════════════════════
# 实例查询/列表（基于共享 user 视角）
# ═══════════════════════════════════════════════════════════════════════════

def _client():
    """返回当前共享 user 的 ApiClient；未 setup 时退化用 seed（admin token）。"""
    try:
        return shared_user_client()
    except Exception:
        from helpers.api import seed as _seed
        return _seed


class _LazyClient:
    """属性代理：每次访问时都从 _client() 拿到当前共享 user 的 ApiClient。

    用法（替换原来的 helpers.api.seed）：
        from _instance_helpers import cli
        cli.get("/openclaw/list")
        cli.post("/openclaw/create", data={...})

    如此 16 个共享实例测试文件无需关心"何时 setup 共享实例"，只要访问 cli
    就能在第一次调用时触发 _shared.require_shared_instance() 的懒加载。
    """

    def __getattr__(self, name):
        return getattr(_client(), name)


cli = _LazyClient()


def find_instance_by_db_id(db_id):
    """从 /openclaw/list 按主键 id 精确查询"""
    cli = _client()
    resp = cli.get(
        "/openclaw/list",
        params={"id": db_id, "page_size": 100},
        expect=None, raw=True,
    )
    if resp.status_code != 200:
        return None
    instances = (resp.json() or {}).get("instances") or []
    for inst in instances:
        v = inst.get("id") or inst.get("ID")
        if v == db_id:
            return inst
    return None


def get_status(db_id, *, client=None):
    """查询实例语义状态。失败时抛出 AssertionError。

    后端契约：实例记录完全不存在（或不属于本人）时返回 404。为兼容调用方
    既有的「空状态 = 记录已消失」判定语义，此处把 404 归一化为空状态对象。
    软删实例仍返回 200 + status="destroyed"，不走这条兜底。

    Args:
        client: 可选 ApiClient。不传时使用共享实例池的 user client，
                适用于 "复用共享实例" 场景；传入时适用于 "自包含
                独立用户" 场景（如 test_instance_lifecycle.py）。
    """
    cli = client if client is not None else _client()
    resp = cli.get("/openclaw/status", params={"id": db_id},
                   expect=None, raw=True)
    if resp.status_code == 404:
        return {"status": "", "label": "", "tooltip": "", "actions": [],
                "transient": False}
    assert resp.status_code == 200, (
        f"/openclaw/status 期望 200/404，实际 {resp.status_code}: {resp.text}"
    )
    return resp.json()


# ═══════════════════════════════════════════════════════════════════════════
# 实例状态等待
# ═══════════════════════════════════════════════════════════════════════════

def wait_for_status(db_id, target_status, *, timeout=None, terminal_failures=None, client=None):
    """轮询 /openclaw/status，等待进入 target_status（或目标集合中任一）。

    Args:
        client: 可选 ApiClient。不传时默认使用共享实例池的 user client；
                传入时使用调用方自己的 client（避免跨用户查询
                被后端归属校验静默返回空状态导致超时）。
    """
    if isinstance(target_status, str):
        targets = {target_status}
    else:
        targets = set(target_status)
    if terminal_failures is None:
        terminal_failures = {
            "create_failed", "load_failed", "destroyed", "upgrade_failed",
        }
    failures = set(terminal_failures) - targets
    timeout = timeout or STATUS_TIMEOUT

    print(f"     [wait] target={sorted(targets)} timeout={timeout}s")
    start = time.time()
    last_status = None
    while True:
        elapsed = time.time() - start
        if elapsed > timeout:
            raise AssertionError(
                f"等待 status∈{sorted(targets)} 超时（{timeout}s），最后状态={last_status}"
            )
        data = get_status(db_id, client=client)
        status = data.get("status", "unknown")
        if status != last_status:
            print(f"     [wait] [{int(elapsed)}s] status={status} label={data.get('label')}")
            last_status = status
        if status in targets:
            return data
        if status in failures:
            raise AssertionError(
                f"实例进入失败终态 status={status} tooltip={data.get('tooltip')}"
            )
        time.sleep(POLL_INTERVAL)


def wait_for_running(db_id, *, timeout=None, client=None):
    """等实例进入 running 状态（创建/重启场景通用）"""
    return wait_for_status(db_id, "running", timeout=timeout, client=client)


def wait_for_destroyed(db_id, *, timeout=None, client=None):
    """等实例进入 destroyed 状态。

    后端契约：delete 会把 instance 行 GORM 软删（deleted_at 置位），
    /openclaw/status 通过 Unscoped 兜底命中软删记录，强制返回终态
    status="destroyed"（transient=false, actions=[]）。
    兼容 status=""：极早期版本/记录已被物理清理时会返回空对象，
    从测试角度同样等价于"销毁完成"，因此把空字符串也视为目标终态之一。
    """
    return wait_for_status(
        db_id, {"destroyed", ""},
        timeout=timeout,
        terminal_failures={"create_failed", "load_failed", "upgrade_failed"},
        client=client,
    )


def wait_for_agent_ready(db_id, *, timeout=None, client=None):
    """轮询 /openclaw/check-openclaw-port 直到 running=true。

    Args:
        client: 可选 ApiClient，同 wait_for_status。
    """
    timeout = timeout or SERVICE_TIMEOUT
    cli = client if client is not None else _client()
    print(f"     [wait] agent ready timeout={timeout}s")
    start = time.time()
    while True:
        elapsed = time.time() - start
        if elapsed > timeout:
            raise AssertionError(f"等待 agent ready 超时（{timeout}s）")
        resp = cli.get(
            "/openclaw/check-openclaw-port",
            params={"id": db_id},
            expect=None, raw=True, timeout=120,
        )
        if resp.status_code == 200:
            try:
                data = resp.json()
            except Exception:
                data = {}
            if data.get("running"):
                print(f"     [wait] agent ready in {int(elapsed)}s")
                return data
            print(f"     [wait] [{int(elapsed)}s] running=False reason={data.get('reason')}")
        else:
            print(f"     [wait] [{int(elapsed)}s] http={resp.status_code}, retry")
        time.sleep(POLL_INTERVAL)


# ═══════════════════════════════════════════════════════════════════════════
# 业务断言（实例管理特有）
# ═══════════════════════════════════════════════════════════════════════════

def assert_error_message(resp, *substrings):
    """断言响应是 {error: ...} 且 error 包含给定子串之一。返回 error 字符串。"""
    try:
        body = resp.json()
    except Exception:
        raise AssertionError(
            f"响应不是合法 JSON: {_truncate(resp.text or '', 200)}"
        )
    err = (body or {}).get("error", "")
    if not err:
        raise AssertionError(f"响应缺少 error 字段: {body}")
    if substrings and not any(s in err for s in substrings):
        raise AssertionError(f"error 不含期望子串 {substrings}: error={err}")
    return err


def assert_json_keys(resp, *keys, allow_missing=False):
    """断言响应是 JSON 且包含全部指定 key。返回 dict。"""
    try:
        body = resp.json()
    except Exception:
        raise AssertionError(
            f"响应不是合法 JSON: {_truncate(resp.text or '', 200)}"
        )
    if not isinstance(body, dict):
        raise AssertionError(f"响应不是 JSON 对象: {type(body).__name__}")
    if not allow_missing:
        missing = [k for k in keys if k not in body]
        if missing:
            raise AssertionError(
                f"响应缺少字段 {missing}: keys={list(body.keys())}"
            )
    return body


__all__ = [
    # 共享实例入口
    "require_shared_instance", "require_shared_db_id", "shared_user_client",
    "get_shared_db_id", "get_shared_db_id_or_none",
    "cli",
    # 查询
    "find_instance_by_db_id", "get_status",
    # 状态等待
    "wait_for_status", "wait_for_running", "wait_for_destroyed",
    "wait_for_agent_ready",
    # 断言
    "assert_error_message", "assert_json_keys",
    # 常量
    "NONEXISTENT_DB_ID", "POLL_INTERVAL",
    "STATUS_TIMEOUT", "SERVICE_TIMEOUT",
]
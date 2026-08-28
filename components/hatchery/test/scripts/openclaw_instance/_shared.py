#!/usr/bin/env python3
"""
实例管理集成测试 — 跨文件共享实例（A' 方案）

设计目标：
    - CI runner 并发跑（concurrency=3）+ 顺序随机收录 16+1 个 test_instance_*.py 文件，
      所有文件共享同一个 instance（避免 N 次重复建实例）。
    - 第一个抢到文件锁的进程负责 setup_admin / setup_user / setup_instance；
      其余进程等锁释放后直接读 state.json 复用 admin_token / user_token / db_id。
    - 实例本身不在测试内部清理，由 CI 末尾的 test/cleanup.py 按 IDENTIFIER 数据库
      隔离一锅端（已存在机制）。

文件锁/状态文件位置:
    /tmp/openclaw_instance.${IDENTIFIER}.lock
    /tmp/openclaw_instance.${IDENTIFIER}.state.json
本地手动跑（IDENTIFIER 为空）时退化为 .dev 后缀。

对外暴露：
    require_shared_instance() -> SharedContext(admin_token, user_token, inst)
    require_shared_db_id()    -> int
    user_client_for(token)    -> ApiClient （便捷封装）

环境开关：
    SKIP_SHARED_INSTANCE=1   不去 setup 真实例（仅契约/参数校验场景）
    SHARED_INSTANCE_REUSE=<db_id>  本地调试时强制复用某个 db_id；同时也需要
                                   提供 SHARED_USER_TOKEN 才能跑用户视角接口
    SHARED_USER_TOKEN=<token>      本地调试时复用某个用户 token
"""
from __future__ import annotations

import contextlib
import dataclasses
import errno
import fcntl
import json as _json
import os
import sys
import time

# ── 把项目 test/scripts 加到 sys.path，便于 import helpers ──
_HERE = os.path.dirname(os.path.abspath(__file__))
_SCRIPTS_DIR = os.path.dirname(_HERE)
if _SCRIPTS_DIR not in sys.path:
    sys.path.insert(0, _SCRIPTS_DIR)

from helpers import config  # noqa: E402
from helpers import (  # noqa: E402
    setup_admin,
    setup_user,
    setup_instance,
)
from helpers.api import (  # noqa: E402
    ApiClient,
    user_client,
    admin_client,
    ensure_gateway_ui_enabled,
)
from helpers.user_mgmt import (  # noqa: E402
    AdminContext,
    UserContext,
    admin_enable_token,
    admin_get_user_token,
)
from helpers.client import GREEN, RED, YELLOW, GRAY, BOLD  # noqa: E402

# ═══════════════════════════════════════════════════════════════════════════
# 路径
# ═══════════════════════════════════════════════════════════════════════════

_IDENTIFIER = os.environ.get("IDENTIFIER", "").strip() or "dev"
_LOCK_FILE = os.path.join("/tmp", f"openclaw_instance.{_IDENTIFIER}.lock")
_STATE_FILE = os.path.join("/tmp", f"openclaw_instance.{_IDENTIFIER}.state.json")

# 共享实例 setup 阶段的最长等待时间（包括别的进程持锁建实例时本进程的等待）。
# 真实建实例本身约 5–10 分钟。CI 单脚本超时 15 分钟，这里给 12 分钟兜底，
# 留 3 分钟给本文件自身的用例执行；本地手动跑可通过环境变量加大。
_SHARED_SETUP_TIMEOUT_SEC = int(os.environ.get("SHARED_SETUP_TIMEOUT_SEC", "720"))

_SCENARIO = os.environ.get("SHARED_SCENARIO", "instmgr")


# ═══════════════════════════════════════════════════════════════════════════
# 数据结构
# ═══════════════════════════════════════════════════════════════════════════

@dataclasses.dataclass
class SharedContext:
    admin_token: str
    user_token: str
    db_id: int
    instance_id: str
    name: str

    def user_client(self) -> ApiClient:
        return user_client(self.user_token)


# ═══════════════════════════════════════════════════════════════════════════
# 状态读写（state.json）
# ═══════════════════════════════════════════════════════════════════════════

def _state_load() -> dict:
    try:
        with open(_STATE_FILE, "r", encoding="utf-8") as fp:
            return _json.load(fp)
    except (FileNotFoundError, _json.JSONDecodeError):
        return {}


def _state_save(state: dict) -> None:
    tmp = f"{_STATE_FILE}.tmp"
    with open(tmp, "w", encoding="utf-8") as fp:
        _json.dump(state, fp, ensure_ascii=False, indent=2)
    os.replace(tmp, _STATE_FILE)


# ═══════════════════════════════════════════════════════════════════════════
# 文件锁
# ═══════════════════════════════════════════════════════════════════════════

@contextlib.contextmanager
def _flock_exclusive(path: str, *, timeout: int):
    """阻塞式获取互斥文件锁，超时抛 TimeoutError。"""
    fd = os.open(path, os.O_CREAT | os.O_RDWR, 0o644)
    deadline = time.time() + timeout
    try:
        while True:
            try:
                fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
                break
            except OSError as e:
                if e.errno not in (errno.EAGAIN, errno.EACCES):
                    raise
                if time.time() > deadline:
                    raise TimeoutError(
                        f"获取共享实例文件锁 {path} 超时（{timeout}s）"
                    )
                # 让出 CPU；轮询间隔短一点便于尽快感知建实例完成
                time.sleep(2)
        yield fd
    finally:
        try:
            fcntl.flock(fd, fcntl.LOCK_UN)
        finally:
            os.close(fd)


# ═══════════════════════════════════════════════════════════════════════════
# 实例存活探测（避免拿到一个早就被 cleanup 的过期 db_id）
# ═══════════════════════════════════════════════════════════════════════════

def _instance_alive(user_token: str, db_id: int) -> bool:
    try:
        client = user_client(user_token)
        resp = client.get(
            "/openclaw/list",
            params={"id": db_id, "page_size": 1},
            expect=None, raw=True, timeout=15,
        )
        if resp.status_code != 200:
            return False
        body = resp.json() or {}
        instances = body.get("instances") or []
        if not instances:
            return False
        # destroyed 的实例没意义
        inst = instances[0]
        status = (
            inst.get("status")
            or inst.get("Status")
            or inst.get("LastStableState", "")
        )
        if isinstance(status, str) and status.lower() == "destroyed":
            return False
        return True
    except Exception:
        return False


# ═══════════════════════════════════════════════════════════════════════════
# 幂等的 setup_admin / setup_user
#
# CI 单脚本 15min 超时机制下，第一个抢到锁的进程若被 kill，已经创建的 admin/
# user 会残留在数据库（hatchery 数据库 emptyDir 在 Pod 内是持久的）。下一个
# 进程拿到锁时 setup_admin → 409「用户名已存在」直接 fail。
#
# 这里包一层幂等：409 时按 username 查找现有用户，复用其 ID + token。
# ═══════════════════════════════════════════════════════════════════════════

def _find_user_by_name(admin_token: str, username: str) -> dict | None:
    """通过 GET /admin/users?username= 精确查询，找到返回原始字典，否则 None。"""
    cli = admin_client(admin_token)
    resp = cli.get(
        "/admin/users",
        params={"username": username, "page_size": 50},
        expect=None, raw=True, timeout=15,
    )
    if resp.status_code != 200:
        return None
    body = resp.json() or {}
    users = body.get("users") or body.get("Users") or []
    for u in users:
        name = u.get("username") or u.get("Username")
        if name == username:
            return u
    return None


def _setup_admin_idempotent(scenario: str) -> AdminContext:
    """与 helpers.setup_admin 等价，但在 409「用户名已存在」时按 username 查找复用。"""
    username = f"{config.ADMIN_USERNAME_PREFIX}{scenario}"
    try:
        return setup_admin(scenario)
    except AssertionError as e:
        msg = str(e)
        if "用户名已存在" not in msg and "409" not in msg:
            raise
        print(YELLOW(
            f">>> [shared] setup_admin 撞 409，按 username={username} 查找复用"
        ))
        u = _find_user_by_name(config.SEED_ADMIN_TOKEN, username)
        if not u:
            raise AssertionError(
                f"setup_admin 409 但 /admin/users 也找不到 {username}: {e}"
            )
        user_id = u.get("id") or u.get("ID")
        admin_enable_token(config.SEED_ADMIN_TOKEN, user_id)
        token = admin_get_user_token(config.SEED_ADMIN_TOKEN, user_id)
        print(f"    \u590d\u7528\u73b0\u6709 admin id={user_id}\u3001username={username} \u2713")
        return AdminContext(user_id=user_id, token=token, username=username)


def _setup_user_idempotent(admin_token: str, scenario: str) -> UserContext:
    """与 helpers.setup_user 等价，但在 409「用户名已存在」时按 username 查找复用。"""
    username = f"{config.USERNAME_PREFIX}{scenario}"
    try:
        return setup_user(admin_token, scenario)
    except AssertionError as e:
        msg = str(e)
        if "用户名已存在" not in msg and "409" not in msg:
            raise
        print(YELLOW(
            f">>> [shared] setup_user 撞 409，按 username={username} 查找复用"
        ))
        u = _find_user_by_name(admin_token, username)
        if not u:
            raise AssertionError(
                f"setup_user 409 但 /admin/users 也找不到 {username}: {e}"
            )
        user_id = u.get("id") or u.get("ID")
        admin_enable_token(admin_token, user_id)
        token = admin_get_user_token(admin_token, user_id)
        print(f"    \u590d\u7528\u73b0\u6709 user id={user_id}\u3001username={username} \u2713")
        return UserContext(user_id=user_id, token=token, username=username)


def _find_running_instance_for_user(user_token: str):
    """在该用户名下找到一个 running/已就绪的实例并返回 (db_id, instance_id, name)；
    若没有，返回 None。

    用于 setup_instance 幂等：上次建实例完成但写 state.json 之前进程被 kill 时复用。
    """
    cli = user_client(user_token)
    resp = cli.get(
        "/openclaw/list",
        params={"page": 1, "page_size": 50},
        expect=None, raw=True, timeout=15,
    )
    if resp.status_code != 200:
        return None
    items = (resp.json() or {}).get("instances") or []
    # \u4f18\u5148 LastStableState=RUNNING\uff0c\u540e\u7eed\u8c03 wait_instance_ready/check-gateway-access \u68c0\u9a8c
    for inst in items:
        state = (
            inst.get("LastStableState")
            or inst.get("status")
            or inst.get("Status")
            or ""
        )
        if isinstance(state, str) and state.upper() in ("RUNNING",):
            db_id = inst.get("id") or inst.get("ID")
            instance_id = inst.get("instance_id") or inst.get("InstanceId") or ""
            name = inst.get("name") or inst.get("Name") or ""
            if db_id:
                return int(db_id), instance_id, name
    return None


def _setup_instance_idempotent(user_token: str, scenario: str):
    """先在用户名下查找现有 running 实例，没有再调 setup_instance。"""
    found = _find_running_instance_for_user(user_token)
    if found:
        db_id, instance_id, name = found
        print(GREEN(
            f">>> [shared] \u590d\u7528\u73b0\u6709\u5b9e\u4f8b db_id={db_id} "
            f"instance_id={instance_id}"
        ))
        # \u8d8b\u4e8e\u8c28\u614e\uff0c\u4ecd\u7136\u8c03\u4e00\u4e0b\u5065\u5eb7\u68c0\u67e5\uff1a\u8fd9\u91cc\u8df3\u8fc7\uff0c\u540e\u7eed\u7528\u4f8b\u4f1a\u81ea\u5df1\u5c1d\u8bd5
        from helpers.instance import InstanceContext
        return InstanceContext(
            db_id=db_id,
            instance_id=instance_id,
            user_token=user_token,
        )
    return setup_instance(user_token, scenario)


# ═══════════════════════════════════════════════════════════════════════════
# 入口：require_shared_instance / require_shared_db_id
# ═══════════════════════════════════════════════════════════════════════════

_cached: SharedContext | None = None


def require_shared_instance() -> SharedContext:
    """获取（或建立）共享实例上下文。跨进程互斥安全。

    锁内逻辑：
        1) 读 state.json，若 db_id 仍存活 → 直接返回
        2) 否则：setup_admin → setup_user → setup_instance（不等 Gateway，
           部分用例不需要）→ 写 state.json
    """
    global _cached
    if _cached is not None:
        return _cached

    # 本地手动调试：允许直接复用现成 db_id + user_token
    reuse_db_id = os.environ.get("SHARED_INSTANCE_REUSE", "").strip()
    reuse_user_token = os.environ.get("SHARED_USER_TOKEN", "").strip()
    if reuse_db_id and reuse_user_token:
        _cached = SharedContext(
            admin_token=os.environ.get("SHARED_ADMIN_TOKEN", "") or reuse_user_token,
            user_token=reuse_user_token,
            db_id=int(reuse_db_id),
            instance_id=os.environ.get("SHARED_INSTANCE_ID", ""),
            name=os.environ.get("SHARED_INSTANCE_NAME", ""),
        )
        print(GRAY(f">>> [shared] 复用环境变量提供的 db_id={_cached.db_id}"))
        return _cached

    print(GRAY(f">>> [shared] 获取共享实例锁: {_LOCK_FILE}"))
    with _flock_exclusive(_LOCK_FILE, timeout=_SHARED_SETUP_TIMEOUT_SEC):
        state = _state_load()
        cached_db_id = state.get("db_id")
        cached_user_token = state.get("user_token")

        if cached_db_id and cached_user_token and _instance_alive(
            cached_user_token, cached_db_id,
        ):
            print(GREEN(
                f">>> [shared] 复用已有实例 db_id={cached_db_id} "
                f"user={state.get('user_name', '?')}"
            ))
            _cached = SharedContext(
                admin_token=state.get("admin_token", "") or cached_user_token,
                user_token=cached_user_token,
                db_id=int(cached_db_id),
                instance_id=state.get("instance_id", ""),
                name=state.get("name", ""),
            )
            return _cached

        # 锁内、首次构建
        print(BOLD(YELLOW(">>> [shared] 首次进入：setup admin / user / instance")))
        admin = _setup_admin_idempotent(_SCENARIO)
        # 关键：CI 测试环境默认未开启 gateway_ui_enable 站点开关，setup_instance
        # 内部会调 wait_gateway_ready 死等 Gateway 端口就绪，若未开启则 600s 超时。
        # 对齐 channel/* 与 model/* 的标准做法，提前打开开关。
        ensure_gateway_ui_enabled(admin.token)
        user = _setup_user_idempotent(admin.token, _SCENARIO)
        # setup_instance 内部会等待实例 running + Gateway 就绪，耗时 5–10 min。
        # 后续进程拿到锁后仅需从 state.json 读取。
        # _setup_instance_idempotent 会先尝试复用上轮已建好的实例（如果上一个
        # 进程被 15min 超时杀掉但实例已就绪、还没来得及写 state.json）
        inst = _setup_instance_idempotent(user.token, _SCENARIO)

        new_state = {
            "admin_token": admin.token,
            "admin_name": getattr(admin, "username", ""),
            "user_token": user.token,
            "user_name": getattr(user, "username", ""),
            "db_id": int(inst.db_id),
            "instance_id": inst.instance_id,
            "name": getattr(inst, "name", ""),
            "created_at": int(time.time()),
        }
        _state_save(new_state)
        print(GREEN(
            f">>> [shared] 共享实例就绪 db_id={inst.db_id} "
            f"instance_id={inst.instance_id}"
        ))
        _cached = SharedContext(
            admin_token=admin.token,
            user_token=user.token,
            db_id=int(inst.db_id),
            instance_id=inst.instance_id,
            name=getattr(inst, "name", ""),
        )
        return _cached


def require_shared_db_id() -> int:
    return require_shared_instance().db_id


def shared_user_client() -> ApiClient:
    """对应共享 user 的 ApiClient（X-OpenAPI: 1）。"""
    return require_shared_instance().user_client()


# ═══════════════════════════════════════════════════════════════════════════
# 试探性获取（不存在就返回 None，用于"无共享实例时跳过"的旧风格用例）
# ═══════════════════════════════════════════════════════════════════════════

def get_shared_db_id_or_none() -> int | None:
    """优先级：环境变量 → state.json → 主动 setup。

    若 SKIP_SHARED_INSTANCE=1，则不主动 setup，只复用现成的（没有就返回 None）。
    """
    if os.environ.get("SKIP_SHARED_INSTANCE") == "1":
        # 仅查 state.json
        state = _state_load()
        cached = state.get("db_id")
        if cached and _instance_alive(state.get("user_token", ""), cached):
            return int(cached)
        return None
    try:
        return require_shared_db_id()
    except Exception as e:
        print(RED(f">>> [shared] setup 失败: {e}"))
        return None

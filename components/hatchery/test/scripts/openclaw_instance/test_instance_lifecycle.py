#!/usr/bin/env python3
"""
集成测试：实例管理生命周期主流程 (E2E，自包含)

不参与"共享实例池"，自己创建独立用户 + 独立实例，跑完销毁。

覆盖核心接口（按调用顺序）：
    GET  /openclaw/agent-types          # 找一个可用的 agent_type
    GET  /openclaw/zones                # 验证只读
    GET  /openclaw/current-image        # 验证启用镜像存在
    GET  /openclaw/list                 # 创建前列表（基线）
    POST /openclaw/create               # 创建实例
    GET  /openclaw/list?instance_id=    # 创建后从列表反查 db_id
    GET  /openclaw/status               # 等待 running
    GET  /openclaw/check-openclaw-port  # 等待 agent ready
    GET  /openclaw/service-status
    GET  /openclaw/version
    GET  /openclaw/instance-models
    GET  /openclaw/install-skills
    POST /openclaw/restart-gateway      # 重启 Agent 下发契约（不重启 CVM）
    POST /openclaw/reboot               # 重启下发契约（不等回到 running）
    POST /openclaw/delete               # 销毁 → 等 destroyed
    GET  /openclaw/status               # destroyed 终态校验

独立用户避免与共享实例池产生干扰，预计 ~12 分钟。

超时设计（CI 单脚本超时 15min）：
    wait_running    480s  (8min)   首次创建后等 running
    wait_agent      180s  (3min)
    其它接口        合计  ~30s
    wait_running    240s  (4min)   reboot 后等回 running
    wait_destroyed  420s  (7min)
    -------------------------------------
    合计上限        ~22min（极端值），平均预计 ~12 min
    （wait_running × 2 与 wait_destroyed 大概率提前结束，
     首次 wait_running ~45s，reboot wait_running ~60s，
     wait_destroyed ~90s）
"""
import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
from helpers.api import (
    user_client,
    health_check,
    ensure_gateway_ui_enabled,
    ApiClient,
    assert_status,
)
from helpers.client import GREEN, RED, BOLD
from helpers import create_instance

from _instance_helpers import (
    get_status,
    wait_for_running,
    wait_for_destroyed,
    wait_for_agent_ready,
)
from _shared import (
    _setup_admin_idempotent,
    _setup_user_idempotent,
)

SCENARIO = "instlife"


def _resolve_db_id(client, cvm_id, *, retries=10, interval=2):
    """从 /openclaw/list 反查 db_id（创建接口只返回 cvm instance_id）"""
    for _ in range(retries):
        resp = client.get(
            "/openclaw/list",
            params={"instance_id": cvm_id, "page_size": 100},
            expect=None, raw=True,
        )
        if resp.status_code == 200:
            for inst in (resp.json() or {}).get("instances") or []:
                iid = inst.get("instance_id") or inst.get("InstanceId")
                if iid == cvm_id:
                    return inst
        time.sleep(interval)
    return None


def main():
    health_check()
    print()

    admin = _setup_admin_idempotent(SCENARIO)
    # 与 channel/* 等模块对齐：开启 gateway_ui_enable 站点开关，否则 setup_instance
    # 内部 wait_gateway_ready 会 600s 超时。
    ensure_gateway_ui_enabled(admin.token)
    user = _setup_user_idempotent(admin.token, SCENARIO)
    cli = user_client(user.token)

    db_id = None
    try:
        # ─── 前置只读检查 ───────────────────────────────────────────
        print(BOLD(">>> 步骤 1：查询 agent-types ..."))
        body = cli.get("/openclaw/agent-types")
        items = body.get("agent_types") or body.get("AgentTypes") or body.get("data") or []
        if not items and isinstance(body, list):
            items = body
        assert items, f"agent-types 返回为空: {body}"
        agent_type = None
        for it in items:
            code = it.get("code") or it.get("Code") or it.get("agent_type")
            if code == "openclaw":
                agent_type = code
                break
        if not agent_type:
            first = items[0]
            agent_type = first.get("code") or first.get("Code") or first.get("agent_type")
        assert agent_type, f"无法解析 code: {items[:1]}"
        print(f"    chosen agent_type={agent_type}, total={len(items)} ✓")

        print(BOLD(">>> 步骤 2：查询 zones ..."))
        cli.get("/openclaw/zones")
        print("    ✓")

        print(BOLD(">>> 步骤 3：查询 current-image ..."))
        body = cli.get("/openclaw/current-image", params={"agent_type": agent_type})
        image = body.get("image")
        assert image is not None, (
            f"agent_type={agent_type} 未配置启用镜像，无法创建实例"
        )
        print(f"    image_id={image.get('image_id')} agent_version={image.get('agent_version')} ✓")

        print(BOLD(">>> 步骤 4：查询创建前 list 基线 ..."))
        body = cli.get("/openclaw/list", params={"page": 1, "page_size": 30})
        assert body.get("total") is not None, f"list 缺 total: {body}"
        print(f"    baseline total={body['total']} ✓")

        # ─── 创建 ──────────────────────────────────────────────────
        name = f"{config.INSTANCE_NAME_PREFIX}{SCENARIO}-{int(time.time())}"
        print(BOLD(f">>> 步骤 5：创建实例 name={name} ..."))
        body = create_instance(user.token, name, agent_type=agent_type)
        assert body.get("ok"), f"创建失败: {body}"
        cvm_id = body.get("instance_id")
        assert cvm_id, f"响应未返回 instance_id: {body}"
        print(f"    create 已下发 cvm_instance_id={cvm_id} ✓")

        print(BOLD(">>> 步骤 6：反查 db_id ..."))
        target = _resolve_db_id(cli, cvm_id)
        assert target, f"创建后未在 list 找到 instance_id={cvm_id}"
        db_id = target.get("id") or target.get("ID")
        assert db_id, f"列表返回缺 db id: {target}"
        print(f"    db_id={db_id} agent_type={target.get('agent_type')} ✓")

        # ─── 等就绪 ────────────────────────────────────────────────
        # 单脚本超时 15 分钟硬限制，建实例 + 后续操作总耗时必须 ≤ 14 min。
        # 留 8 分钟给 wait_running，6 分钟给后续步骤。
        #
        # 注意：本测试是「自包含独立用户」场景，必须把 cli（user.token 客户端）
        # 显式传给 wait_for_*，否则 helpers 默认会走 shared_user_client，
        # 那个 user 不拥有本测试新建的实例 → /openclaw/status 会因为
        # getInstanceByID 鉴权失败而静默返回空状态对象（status=""），
        # 永远等不到 running，最终 480s 超时。
        print(BOLD(">>> 步骤 7：等待实例 running (timeout=480s) ..."))
        data = wait_for_running(db_id, timeout=480, client=cli)
        actions = data.get("actions") or []
        assert "reboot" in actions, f"running 应含 reboot action: {actions}"
        assert "restart_gateway" in actions, (
            f"running 应含 restart_gateway action: {actions}"
        )
        print(f"    label={data.get('label')} actions={actions} ✓")

        print(BOLD(">>> 步骤 8：等待 agent ready (timeout=180s) ..."))
        wait_for_agent_ready(db_id, timeout=180, client=cli)
        print("    ✓")

        # ─── running 状态下的只读探测 ─────────────────────────────
        print(BOLD(">>> 步骤 9：service-status ..."))
        resp = cli.get(
            "/openclaw/service-status", params={"id": db_id},
            expect=None, raw=True, timeout=120,
        )
        assert resp.status_code == 200, (
            f"service-status 期望 200，实际 {resp.status_code}: {resp.text[:200]}"
        )
        print(f"    keys={list((resp.json() or {}).keys())} ✓")

        print(BOLD(">>> 步骤 10：version ..."))
        body = cli.get("/openclaw/version", params={"id": db_id}, timeout=60)
        print(f"    version={body} ✓")

        print(BOLD(">>> 步骤 11：instance-models ..."))
        body = cli.get("/openclaw/instance-models", params={"id": db_id})
        models = body.get("models") or body.get("Models") or []
        print(f"    models_count={len(models)} ✓")

        print(BOLD(">>> 步骤 12：install-skills ..."))
        body = cli.get("/openclaw/install-skills", params={"id": db_id})
        assert "total" in body, f"install-skills 缺 total: {body}"
        print(f"    total={body.get('total')} ✓")

        # ─── restart-gateway（方法/鉴权/成功下发）────────────────────
        print(BOLD(">>> 步骤 13：restart-gateway GET 方法限制 ..."))
        resp = cli.get(
            "/openclaw/restart-gateway",
            params={"id": db_id},
            expect=None,
            raw=True,
            timeout=30,
        )
        assert_status(resp, 405, label="restart-gateway-get")
        print("    GET → 405 ✓")

        print(BOLD(">>> 步骤 13.1：restart-gateway 未登录拦截 ..."))
        resp = ApiClient("", openapi=True, timeout=30).post(
            "/openclaw/restart-gateway",
            data={"id": db_id},
            expect=None,
            raw=True,
            timeout=30,
        )
        assert_status(resp, {401, 403}, label="restart-gateway-anon")
        print(f"    anonymous → {resp.status_code} ✓")

        print(BOLD(">>> 步骤 13.2：restart-gateway 下发契约 ..."))
        body = cli.post(
            "/openclaw/restart-gateway",
            data={"id": db_id},
            timeout=120,
        )
        assert body.get("ok"), f"restart-gateway 响应 ok=false: {body}"
        print("    restart-gateway 已下发 ✓")

        print(BOLD(">>> 步骤 13.3：等待 restart-gateway 后 agent ready (timeout=180s) ..."))
        wait_for_agent_ready(db_id, timeout=180, client=cli)
        print("    ✓")

        # ─── reboot（下发 + 等回 running）──────────────────────────
        # reboot 后 cvm 会进入 loading（transient）状态，此时 delete 会被
        # 后端拒绝（409 "实例加载中，无法删除"）。所以这里必须等实例回到
        # running 稳定态后，才能继续 delete。
        # 实测 reboot → running 一般 ~60s，预算 240s 兜底。
        print(BOLD(">>> 步骤 14：reboot 下发契约 ..."))
        body = cli.post(
            "/openclaw/reboot", data={"id": db_id}, timeout=60,
        )
        assert body.get("ok"), f"reboot 响应 ok=false: {body}"
        print("    reboot 已下发 ✓")

        print(BOLD(">>> 步骤 14.1：等待 reboot 完成回到 running (timeout=240s) ..."))
        data = wait_for_running(db_id, timeout=240, client=cli)
        print(f"    label={data.get('label')} ✓")
        # ─── delete → destroyed ─────────────────────────────────────
        print(BOLD(">>> 步骤 15：delete ..."))
        body = cli.post("/openclaw/delete", data={"id": db_id}, timeout=60)
        assert body.get("ok"), f"delete 响应 ok=false: {body}"
        print("    delete 已下发 ✓")

        print(BOLD(">>> 步骤 16：等待 destroyed (timeout=420s) ..."))
        data = wait_for_destroyed(db_id, timeout=420, client=cli)
        print(f"    label={data.get('label')} ✓")

        print(BOLD(">>> 步骤 17：destroyed 后 status 终态校验 ..."))
        data = get_status(db_id, client=cli)
        # 后端契约：delete 会把 instance 行 GORM 软删（deleted_at 置位），
        # /openclaw/status 通过 Unscoped 兜底命中软删记录，强制返回终态
        #   status="destroyed" + transient=false + actions=[]
        # actions 必须为空：记录已软删，任何后续动作（含 delete）都无意义，
        # 返回非空 actions 会诱导前端发起注定 404 的写请求。
        # 兼容 status=""：极早期版本/cleanup 已物理清理记录的历史行为。
        status_now = data.get("status", "")
        assert status_now in ("destroyed", ""), f"期望 destroyed/空: {data}"
        actions = data.get("actions") or []
        assert actions == [], f"软删实例 actions 应为空: {actions}"
        if status_now == "destroyed":
            assert data.get("transient") is False, f"destroyed 应为终态(transient=false): {data}"
            print(f"    status=destroyed actions=[] transient=false ✓")
        else:
            print(f"    status='' (实例记录已被物理清理) ✓")

        # 已 destroy，标记 db_id 为 None 避免 finally 再调一次
        db_id = None

        print()
        print(GREEN("test_instance_lifecycle.py 测试通过 ✅"))

    except Exception as e:
        print(RED(f"\ntest_instance_lifecycle.py 测试失败 ❌: {e}"))
        traceback.print_exc()
        sys.exit(1)
    finally:
        # 兜底：本测试是自包含的，如果中途 abort，尽力删自己建的实例
        if db_id is not None:
            try:
                cli.post(
                    "/openclaw/delete", data={"id": db_id},
                    expect=None, raw=True, timeout=60,
                )
                print(f"[teardown] 已下发 delete db_id={db_id}")
            except Exception as e2:
                print(f"[teardown] 兜底删除失败（CI cleanup.py 会兜底）: {e2}")


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""
TC-Base Hermes 实例生命周期：创建 → 等待就绪 → 验证服务 → 销毁

不再 import test_create_openclaw 复用模块级 client，避免：
  - 模块级 sys.exit(TOKEN missing) 引入的 import 副作用
  - 同时依赖 TOKEN 和 ADMIN_TOKEN 的双环境变量耦合
  - openclaw 测试模块级代码改动时的静默回归

使用方式：
  export API=http://134.175.254.166
  export ADMIN_TOKEN=xxx
  python3 test_create_hermes.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import health_check, get_field
import helpers
from helpers import (
    check_env, setup_admin, setup_user,
    setup_hermes_instance, verify_hermes_service,
)
from helpers.instance import (
    delete_instance, get_instance_status, list_instances,
)


def main():
    check_env()
    health_check()
    print()

    admin = setup_admin("hermes-base-create")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        user = setup_user(admin.token, "hermes-base-create")

        # 步骤 1：创建 hermes 实例（setup_hermes_instance 内部会确保镜像已启用、
        # 等待 CVM running、等待 Gateway 就绪）
        inst = setup_hermes_instance(user.token, "base-create")

        # 步骤 2：从实例列表中找到刚创建的实例，校验 agent_type
        print(">>> 步骤 2：从列表中校验 agent_type ...")
        instances_data = list_instances(user.token)
        instances = instances_data.get("instances", [])

        # 注意：原来的 openclaw 写法 `if a or b == c` 由于运算符优先级
        # 实际等价于 `if a or (b == c)`，匹配会失效。这里显式加括号修正。
        target = None
        for raw in instances:
            iid = get_field(raw, "instance_id", "InstanceId")
            if iid == inst.instance_id:
                target = raw
                break
        assert target, (
            f"created hermes instance not found in list: "
            f"instance_id={inst.instance_id}"
        )

        agent_type = get_field(target, "agent_type", "AgentType")
        assert agent_type == "hermes", (
            f"expected agent_type=hermes, got {agent_type!r}"
        )
        print(f"    Instance: db_id={inst.db_id}, "
              f"instance_id={inst.instance_id}, agent_type={agent_type} ✓")

        # 步骤 3：验证 hermes 服务可用性
        print(">>> 步骤 3：验证 hermes 服务可用性 ...")
        verify_hermes_service(user.token, inst.db_id)

        # 步骤 4：删除实例 + 等待销毁完成
        print(">>> 步骤 4：删除实例 ...")
        del_resp = delete_instance(user.token, inst.db_id)
        assert del_resp.get("ok"), f"删除实例失败: {del_resp}"
        print("    删除请求提交 ✓")

        # 后端契约：delete 会把 instance 行 GORM 软删（deleted_at 置位），
        # /openclaw/status 通过 Unscoped 兜底命中软删记录，强制返回终态
        # status="destroyed"（transient=false, actions=[]）。空字符串与
        # "destroyed" 都视为销毁终态（与 openclaw_instance/_instance_helpers.py
        # 的 wait_for_destroyed 口径保持一致），否则会被空状态卡 600s 超时。
        start = time.time()
        timeout = 600
        terminal_statuses = {"destroyed", ""}
        last_status = None
        while True:
            elapsed = time.time() - start
            if elapsed > timeout:
                raise TimeoutError(f"销毁超时（{timeout}s），最后状态: {last_status}")
            status_data = get_instance_status(user.token, inst.db_id)
            status = status_data.get("status", "unknown")
            if status != last_status:
                print(f"    [{int(elapsed)}s] 状态: {status!r} "
                      f"({status_data.get('label', '')})", flush=True)
                last_status = status
            if status in terminal_statuses:
                print(f"    实例已销毁 ✓ (耗时 {int(elapsed)}s, status={status!r})")
                break
            time.sleep(5)

        # 实例已销毁，inst 不再需要 finally 兜底清理
        inst = None

        print()
        print("TC-Base Hermes 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-Base Hermes 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        # 兜底：异常路径下尝试销毁残留实例
        if inst is not None and user is not None:
            try:
                delete_instance(user.token, inst.db_id)
            except Exception:
                pass


if __name__ == "__main__":
    main()

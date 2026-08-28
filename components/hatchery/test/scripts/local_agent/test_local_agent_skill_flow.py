#!/usr/bin/env python3
"""
本地 agent — skill 分发全链路端到端测试

覆盖 add-skill → sync → ack 的完整闭环，是本地 agent 一期的核心场景。

阶段 A：成功路径
  1. reporter report 建实例
  2. 用户 add-skill "test-fake-<rand>"（本地路径不查 skills 表，任意 slug）
     响应 = {"ok": true, "message": "已下发...", "record_id": <id>}
  3. install-skills 见到 install_status=installing
  4. reporter sync → commands=[{type:install_skill, skill_slug, download_url:clawhub}]
  5. reporter ack success → 幂等再 ack 无副作用
  6. reporter 下一次 report skills 数组带上该 slug（模拟 reporter 端已装）
  7. install-skills 见到 install_status=success

阶段 B：失败路径
  1. add-skill "test-fake-fail-<rand>"
  2. sync → 拿到命令
  3. ack failed（error 字段带原因）
  4. install-skills 见到 install_status=failed

阶段 C：add-skill 幂等
  同一 slug 再 add-skill（还没 ack）→ 走 dedup，返回 deduplicated=true
"""

import os
import sys
import time
import traceback
import uuid

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers  # noqa: E402
from helpers import (  # noqa: E402
    check_env,
    setup_admin,
    setup_user,
    user_client,
    enable_local_agent_feature,
    setup_local_instance,
    reporter_report,
    reporter_sync,
    reporter_ack,
)


def _rand_slug(prefix: str) -> str:
    return f"{prefix}-{uuid.uuid4().hex[:10]}"


def _find_command(commands, slug):
    for cmd in commands or []:
        if cmd.get("skill_slug") == slug:
            return cmd
    return None


def _find_installed(skills, slug):
    for item in skills or []:
        if item.get("slug") == slug or item.get("name") == slug:
            return item
    return None


def main():
    check_env()
    print()

    admin = setup_admin("local-skill-flow")
    user = None
    la = None

    try:
        enable_local_agent_feature(admin.token)
        user = setup_user(admin.token, "local-skill-flow")
        la = setup_local_instance(user.token, "skill-flow")
        print()

        # ══════════════════════════════════════════════════════════════
        # 阶段 A：成功路径
        # ══════════════════════════════════════════════════════════════
        slug_ok = _rand_slug("integ-fake-ok")
        print(f"═══ 阶段 A：成功路径（slug={slug_ok}）═══")

        # 步骤 A1：add-skill
        print(">>> A1：user add-skill ...")
        add_resp = user_client(user.token).post(
            "/openclaw/add-skill",
            data={"id": str(la.db_id), "skill_name": slug_ok},
            expect=None, raw=True, timeout=30,
        )
        assert add_resp.status_code == 200, (
            f"add-skill 应 200，实际={add_resp.status_code} body={add_resp.text}"
        )
        add_data = add_resp.json()
        assert add_data.get("record_id"), f"add-skill 响应应含 record_id: {add_data}"
        assert add_data.get("status") == "pending", (
            f"add-skill status 应为 pending: {add_data}"
        )
        assert "已下发" in (add_data.get("message") or ""), (
            f"add-skill message 应含「已下发」: {add_data}"
        )
        print(f"    ok=true, message={add_data.get('message')!r} ✓")

        # 步骤 A2：add-skill 已返回 record_id（install-skills 只展示已装成功的 skill）
        record_id = add_data.get("record_id")
        assert record_id, f"record_id 应非空: {add_data}"
        print(f"    record_id={record_id} ✓")

        # 步骤 A3：reporter sync → 拿到命令
        print(">>> A3：reporter sync 应下发 install_skill 命令 ...")
        sync_data = reporter_sync(user.token, la)
        cmd = _find_command(sync_data.get("commands", []), slug_ok)
        assert cmd, f"sync commands 未含 slug={slug_ok}: {sync_data}"
        assert cmd.get("type") == "install_skill", f"type 应 install_skill: {cmd}"
        assert cmd.get("id") == record_id, (
            f"command id 应等于 pending record_id: {cmd.get('id')} vs {record_id}"
        )
        assert cmd.get("download_url"), f"download_url 应非空: {cmd}"
        print(f"    id={cmd['id']}, url={cmd['download_url']} ✓")

        # 步骤 A4：reporter ack success
        print(">>> A4：reporter ack success ...")
        ack_data = reporter_ack(user.token, record_id, "success", version="9.9.9")
        assert ack_data.get("status") == "success", f"ack 应返回 status=success: {ack_data}"
        print("    status=success ✓")

        # A4b：ack 幂等
        print(">>> A4b：ack 幂等（重复 ack 应 200 ok=true）...")
        ack2 = reporter_ack(user.token, record_id, "success", version="9.9.9")
        assert ack2.get("status") == "success", f"重复 ack 应幂等 status=success: {ack2}"
        print("    幂等 ✓")

        # 步骤 A5：reporter 下一轮 report 带上该 skill（模拟 reporter 已装）
        print(">>> A5：reporter 下一轮 report 带上已装 skill ...")
        la.add_installed_skill(slug_ok, version="9.9.9",
                               display_name=slug_ok, source="clawpro")
        reporter_report(user.token, la)

        # 步骤 A6：install-skills 见到 success
        print(">>> A6：install-skills 应见 install_status=success ...")
        # 稍等一下让内部状态刷新
        time.sleep(1)
        inst_data = user_client(user.token).get(
            "/openclaw/install-skills", params={"id": la.db_id},
        )
        installed_item = _find_installed(inst_data.get("skills", []), slug_ok)
        assert installed_item, f"install-skills 应仍含 slug={slug_ok}: {inst_data}"
        assert installed_item.get("install_status") == "success", (
            f"install_status 应 success，实际={installed_item}"
        )
        print(f"    install_status=success ✓")

        # 步骤 A7：success 后 install-skills 已含该 slug（pending-skills 接口已移除，
        # 原「success 后 pending 消失」断言不再可验证）
        print(">>> A7：install-skills 已含 success 的 slug（见 A6）✓")

        # ══════════════════════════════════════════════════════════════
        # 阶段 B：失败路径
        # ══════════════════════════════════════════════════════════════
        slug_fail = _rand_slug("integ-fake-fail")
        print(f"\n═══ 阶段 B：失败路径（slug={slug_fail}）═══")

        # B1：add-skill
        print(">>> B1：add-skill ...")
        add_resp = user_client(user.token).post(
            "/openclaw/add-skill",
            data={"id": str(la.db_id), "skill_name": slug_fail},
            expect=200, raw=True, timeout=30,
        )
        add_data = add_resp.json()
        assert add_data.get("record_id"), f"add-skill 应含 record_id: {add_data}"
        assert add_data.get("status") == "pending", f"add-skill status 应为 pending: {add_data}"
        print("    ok ✓")

        # B2：sync 拿命令
        sync_data = reporter_sync(user.token, la)
        cmd = _find_command(sync_data.get("commands", []), slug_fail)
        assert cmd, f"sync commands 未含 slug={slug_fail}: {sync_data}"
        record_id_fail = cmd["id"]
        print(f">>> B2：sync 拿到 record_id={record_id_fail} ✓")

        # B3：ack failed
        print(">>> B3：ack failed ...")
        ack_data = reporter_ack(
            user.token, record_id_fail, "failed",
            error="下载失败: connection timeout (integration test)",
        )
        assert ack_data.get("status") == "failed", f"ack failed 应返回 status=failed: {ack_data}"
        print("    status=failed ✓")

        # B4：ack failed 后该记录进入 failed 终态（pending-skills 接口已移除，
        # 原「failed 记录仍在 pending 列表」断言不再可验证；install-skills 只展示 success）
        print(">>> B4：ack failed 后记录为 failed 终态（见 B3）✓")

        # ══════════════════════════════════════════════════════════════
        # 阶段 C：add-skill 幂等（dedup pending distribute）
        # ══════════════════════════════════════════════════════════════
        slug_dedup = _rand_slug("integ-fake-dedup")
        print(f"\n═══ 阶段 C：add-skill dedup（slug={slug_dedup}）═══")

        print(">>> C1：首次 add-skill ...")
        r1 = user_client(user.token).post(
            "/openclaw/add-skill",
            data={"id": str(la.db_id), "skill_name": slug_dedup},
            expect=200, timeout=30,
        )
        first_dedup = r1.get("deduplicated")
        # 第一次不该 dedup
        assert first_dedup is not True, f"首次 add-skill 不应 dedup: {r1}"
        print("    首次 ok ✓")

        print(">>> C2：同 slug 再次 add-skill（未 ack，应 dedup）...")
        r2 = user_client(user.token).post(
            "/openclaw/add-skill",
            data={"id": str(la.db_id), "skill_name": slug_dedup},
            expect=200, timeout=30,
        )
        assert "deduplicated" in r2, f"二次 add-skill 应有 deduplicated 字段: {r2}"
        assert r2.get("deduplicated") is True, (
            f"二次 add-skill 应 deduplicated=true: {r2}"
        )
        print(f"    deduplicated=true ✓")

        # 清理：删掉本地实例
        user_client(user.token).post(
            "/openclaw/delete", data={"id": str(la.db_id)},
            expect=None, raw=True, timeout=30,
        )
        la = None

        print()
        print("本地 agent skill 分发链路测试通过 ✅")

    except Exception as e:
        print(f"\n本地 agent skill 分发链路测试失败 ❌: {e}")
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

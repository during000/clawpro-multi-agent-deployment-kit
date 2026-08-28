#!/usr/bin/env python3
"""
TC-5.1 技能全生命周期：初始安装 + 手动安装 + 重试/取消

验证实例技能的完整生命周期：
  1. 初始技能包自动安装 + 状态查询
  2. 手动安装技能（含异常参数校验 + 重复安装）
  3. 失败技能重试 / 取消

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_skill_lifecycle.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import (
    check_env, setup_admin,
    setup_user,
    setup_instance,
)


def main():
    check_env()
    print()

    admin = setup_admin("skill-life")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        user = setup_user(admin.token, "skill-life")
        inst = setup_instance(user.token, "skill-life")

        # ════════════════════════════════════════════════════════════
        # 阶段 A：初始技能包自动安装（原 TC-5.1）
        # ════════════════════════════════════════════════════════════

        # 步骤 1：查询初始技能包安装状态
        print(">>> 步骤 1：查询初始技能包安装状态 ...")
        install_data = helpers.user_get_install_skills(user.token, inst.db_id)
        skills = install_data.get("skills", [])
        assert skills, "初始技能包安装列表不应为空"
        print(f"    安装列表非空 ✓  共 {len(skills)} 条")

        # 步骤 2：等待所有技能安装完成
        print(">>> 步骤 2：等待所有技能安装完成 ...")
        install_data = helpers.wait_skills_installed(user.token, inst.db_id)
        installed_skills = install_data.get("skills", [])

        succeeded = [s for s in installed_skills if s.get("install_status") == 2]
        failed = [s for s in installed_skills if s.get("install_status") == 3]
        print(f"    安装完成 ✓  成功 {len(succeeded)} 条, 失败 {len(failed)} 条（已跳过）")

        # 步骤 3：获取已安装技能列表
        print(">>> 步骤 3：获取已安装技能列表 ...")
        skills_list = helpers.user_get_skills(user.token, inst.db_id)
        if isinstance(skills_list, dict):
            skills_list = skills_list.get("skills", skills_list.get("data", []))
        assert skills_list, "已安装技能列表不应为空"
        print(f"    技能列表非空 ✓  共 {len(skills_list)} 条")

        # ════════════════════════════════════════════════════════════
        # 阶段 B：手动安装技能 + 异常参数校验（原 TC-5.2）
        # ════════════════════════════════════════════════════════════

        # 记录初始列表
        initial_names = set(
            s.get("skill_name") or s.get("SkillName") or s.get("name", "")
            for s in skills_list
        )

        # 步骤 4a：skill_name 为空 → 400
        print(">>> 步骤 4a：skill_name 为空 → 400 ...")
        resp = helpers.user_add_skill(user.token, inst.db_id, "")
        assert resp.status_code == 400, (
            f"空 skill_name 应返回 400，实际: {resp.status_code}"
        )
        print("    返回 400 ✓")

        # 步骤 4b：不存在的技能 → 400/500
        print(">>> 步骤 4b：不存在的技能 → 400/500 ...")
        resp = helpers.user_add_skill(user.token, inst.db_id, "nonexistent-skill-xyz")
        assert resp.status_code in (400, 500), (
            f"不存在的技能应返回 400/500，实际: {resp.status_code}"
        )
        print(f"    返回 {resp.status_code} ✓")

        # 步骤 4c：确认异常请求未影响技能列表
        print(">>> 步骤 4c：确认技能列表未变 ...")
        current_list = helpers.user_get_skills(user.token, inst.db_id)
        if isinstance(current_list, dict):
            current_list = current_list.get("skills", current_list.get("data", []))
        current_names = set(
            s.get("skill_name") or s.get("SkillName") or s.get("name", "")
            for s in current_list
        )
        assert current_names == initial_names, (
            f"异常请求不应影响技能列表:\n  初始: {initial_names}\n  当前: {current_names}"
        )
        print("    技能列表未变 ✓")

        # 步骤 5：手动安装合法技能
        # 安装时用的 slug 是 self-improving-agent，但安装后列表中显示的 name 是 self-improvement
        INSTALL_SKILL = "self-improving-agent"
        INSTALL_SKILL_NAME = "self-improvement"
        print(f">>> 步骤 5：手动安装 {INSTALL_SKILL} ...")
        resp = helpers.user_add_skill(user.token, inst.db_id, INSTALL_SKILL)
        assert resp.status_code == 200, (
            f"安装 {INSTALL_SKILL} 失败: {resp.status_code} {resp.text}"
        )
        print("    安装请求提交 ✓")

        # 步骤 6：等待技能安装完成并验证（通过已安装技能列表轮询）
        print(f">>> 步骤 6：等待 {INSTALL_SKILL} 安装完成 ...")
        time.sleep(5)
        timeout = config.SKILL_POLL_TIMEOUT
        start = time.time()
        skill_found = False
        while True:
            if time.time() - start > timeout:
                print("    安装超时")
                break
            current = helpers.user_get_skills(user.token, inst.db_id)
            if isinstance(current, dict):
                current = current.get("skills", current.get("data", []))
            current_names = set(
                s.get("skill_name") or s.get("SkillName") or s.get("name", "")
                for s in current
            )
            if INSTALL_SKILL_NAME in current_names:
                skill_found = True
                print(f"    {INSTALL_SKILL} 安装完成并出现在已安装列表中 ✓ (name={INSTALL_SKILL_NAME})")
                break
            time.sleep(config.POLL_INTERVAL)
        assert skill_found, f"{INSTALL_SKILL} (name={INSTALL_SKILL_NAME}) 应在已安装列表中: {current_names}"

        # 步骤 7：重复安装 → 200/429
        print(f">>> 步骤 7：重复安装 {INSTALL_SKILL} ...")
        resp = helpers.user_add_skill(user.token, inst.db_id, INSTALL_SKILL)
        assert resp.status_code in (200, 429), (
            f"重复安装预期 200 或 429，实际: {resp.status_code}"
        )
        print(f"    返回 {resp.status_code} ✓")

        # ════════════════════════════════════════════════════════════
        # 阶段 C：失败技能重试 / 取消（原 TC-5.3）
        # ════════════════════════════════════════════════════════════

        # 步骤 8：查询当前安装状态，检查是否有失败项
        print(">>> 步骤 8：查询安装状态，检查失败项 ...")
        install_data = helpers.user_get_install_skills(user.token, inst.db_id)
        skills = install_data.get("skills", install_data.get("data", []))
        failed = [s for s in skills if s.get("install_status") == 3]
        print(f"    共 {len(skills)} 条，失败 {len(failed)} 条")

        if not failed:
            # 无失败技能 → 验证接口对空输入的幂等性
            print(">>> 步骤 9：无失败技能，验证重试/取消接口幂等 ...")
            data = helpers.user_retry_failed_skills(user.token, inst.db_id)
            assert data.get("ok"), f"重试接口返回异常: {data}"
            assert data.get("retry_count", 0) == 0, (
                f"无失败时 retry_count 应为 0: {data}"
            )
            print("    重试接口幂等 ✓ (retry_count=0)")

            data = helpers.user_cancel_failed_skills(user.token, inst.db_id)
            assert data.get("ok"), f"取消接口返回异常: {data}"
            assert data.get("cancel_count", 0) == 0, (
                f"无失败时 cancel_count 应为 0: {data}"
            )
            print("    取消接口幂等 ✓ (cancel_count=0)")

            # 确认状态未变
            new_install_data = helpers.user_get_install_skills(user.token, inst.db_id)
            new_skills = new_install_data.get("skills", new_install_data.get("data", []))
            assert len(new_skills) == len(skills), (
                f"技能数量不应变化: 原 {len(skills)} → 现 {len(new_skills)}"
            )
            print("    状态一致 ✓")
        else:
            # 有失败技能 → 先重试，再检查
            print(f">>> 步骤 9：有 {len(failed)} 个失败技能，执行重试 ...")
            data = helpers.user_retry_failed_skills(user.token, inst.db_id)
            assert data.get("ok"), f"重试失败: {data}"
            retry_count = data.get("retry_count", 0)
            assert retry_count > 0, f"有失败技能时 retry_count 应 > 0: {data}"
            print(f"    重试了 {retry_count} 个技能 ✓")

            # 等待异步安装一段时间后再检查
            time.sleep(5)
            install_data = helpers.user_get_install_skills(user.token, inst.db_id)
            skills = install_data.get("skills", install_data.get("data", []))

            # 若仍有失败 → 取消
            still_failed = [s for s in skills if s.get("install_status") == 3]
            if still_failed:
                print(f">>> 步骤 10：仍有 {len(still_failed)} 个失败，执行取消 ...")
                data = helpers.user_cancel_failed_skills(user.token, inst.db_id)
                assert data.get("ok"), f"取消失败: {data}"

                # 确认取消后无失败技能
                install_data = helpers.user_get_install_skills(user.token, inst.db_id)
                skills = install_data.get("skills", install_data.get("data", []))
                remaining_failed = [s for s in skills if s.get("install_status") == 3]
                assert len(remaining_failed) == 0, (
                    f"取消后不应有失败技能: {remaining_failed}"
                )
                print("    取消后无失败技能 ✓")
            else:
                print("    重试后无失败技能 ✓")

        print()
        print("TC-5.1 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-5.1 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        pass


if __name__ == "__main__":
    main()

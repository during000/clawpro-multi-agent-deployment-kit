#!/usr/bin/env python3
"""
技能共建审核 — 集成测试（IT-1 ~ IT-7 + 撤回/管理员上下架）

覆盖场景（对齐 .specs/.../06-it.md）：
  IT-1  员工提交 → 管理员通过 → published → 技能广场可见
  IT-2  员工申请下架 → 管理员通过 → offline → 广场不可见
  IT-3  同一 slug 有 pending 时互斥 → 400
  IT-4  员工尝试下架他人/管理员技能 → 403
  IT-5  管理员拒绝 publish → Skill 软删除 → 广场不可见
  IT-6  pending_review / offline 不在技能广场
  IT-7  /admin/skills 含 status + uploader_name / uploader_id
  IT-M1 多版本 published 后下架 → 列表 pending_review 可见 → 通过后整 slug offline
  额外  撤回申请、管理员 offline/online、未登录/非管理员鉴权

不依赖 CVM 实例，仅依赖 SMH 存储（技能 zip 上传）。SMH 未启用时 SKIP。

使用方式：
  export API=http://xxx
  export ADMIN_TOKEN=xxx
  python3 test_skill_contribute.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import IDENTIFIER, assert_fields, health_check, user_client
from helpers import (
    check_env,
    setup_admin,
    setup_user,
    teardown_scenario_users,
)
from helpers.admin_skill import (
    admin_create_skill,
    admin_delete_skill,
    admin_find_skill,
    admin_list_skills,
    admin_offline_skill,
    admin_online_skill,
    admin_skill_status,
)
from helpers.contribution import (
    admin_approve_contribution,
    admin_list_contributions,
    admin_reject_contribution,
    contribute_skill,
    my_contributions,
    skillstore_has_slug,
    takedown_skill,
    withdraw_contribution,
)
import helpers

SCENARIO = "skill-contrib"
TS = int(time.time())
SUFFIX = (IDENTIFIER or str(TS))[-8:]


def _slug(prefix):
    # slug: 小写字母/数字/连字符，3-50
    return f"it-{prefix}-{SUFFIX}"[:50].rstrip("-")


def require_smh_enabled(admin_token):
    """SMH 未启用则 SKIP（exit 0）。"""
    cfg = helpers.admin_get_config(admin_token)
    enabled = cfg.get("smh_enabled")
    if enabled not in (True, 1, "1"):
        # 兼容：部分环境站点配置不含该字段，以 skills 列表探测
        resp = helpers.admin_client(admin_token).get(
            "/admin/skills", params={"page": 1, "page_size": 1},
            expect=None, raw=True,
        )
        if resp.status_code == 403 and ("SMH" in (resp.text or "") or "未启用" in (resp.text or "")):
            enabled = False
        elif resp.status_code == 200:
            enabled = True
        else:
            enabled = False
    if enabled in (True, 1, "1"):
        return
    print("    SMH 未启用，跳过技能共建审核集成测试")
    print()
    print("=" * 60)
    print("  技能共建审核 — SKIPPED（SMH 未配置）")
    print("=" * 60)
    sys.exit(0)


def cleanup_skill(admin_token, slug):
    try:
        admin_delete_skill(admin_token, slug, cascade=True)
    except Exception as e:
        print(f"    [cleanup] delete skill {slug}: {e}")


def main():
    check_env()
    health_check()
    print()

    teardown_scenario_users(SCENARIO)
    admin = setup_admin(SCENARIO)
    user = None
    user2 = None
    slugs = []

    try:
        require_smh_enabled(admin.token)
        user = setup_user(admin.token, SCENARIO, instance_quota=0)
        user2 = setup_user(admin.token, f"{SCENARIO}-b", instance_quota=0)

        # ════════════════════════════════════════════════════════════
        # IT-1: 提交 → 通过 → 上架
        # ════════════════════════════════════════════════════════════
        print()
        print("=== IT-1: 提交 → 通过 → published ===")
        slug1 = _slug("pub")
        slugs.append(slug1)

        print(f">>> [1a] 员工提交技能 slug={slug1} ...")
        resp = contribute_skill(user.token, slug1, "IT Publish", "1.0.0")
        assert resp.status_code == 200, f"提交失败: {resp.status_code} {resp.text}"
        body = resp.json()
        assert body.get("ok"), body
        skill_id = body.get("skill_id")
        req_id = body.get("request_id")
        assert skill_id and req_id, body
        print(f"    提交成功 ✓  skill_id={skill_id} request_id={req_id}")

        print(">>> [1b] Skill status = pending_review ...")
        st = admin_skill_status(admin.token, slug1)
        assert st == "pending_review", f"期望 pending_review，实际={st}"
        print("    status=pending_review ✓")

        print(">>> [1c] 技能广场不显示 pending ...")
        assert not skillstore_has_slug(user.token, slug1), "pending 不应出现在广场"
        print("    广场不可见 ✓")

        print(">>> [1d] 审核列表含 pending 申请 ...")
        listing = admin_list_contributions(
            admin.token, status="pending", resource_type="skill", page=1, page_size=50,
        )
        requests = listing.get("requests") or []
        assert any(r.get("id") == req_id for r in requests), f"列表未含 request_id={req_id}"
        print("    审核列表可见 ✓")

        print(">>> [1e] 管理员通过审核 ...")
        resp = admin_approve_contribution(admin.token, req_id)
        assert resp.status_code == 200, f"通过失败: {resp.status_code} {resp.text}"
        print("    审核通过 ✓")

        print(">>> [1f] Skill status = published ...")
        st = admin_skill_status(admin.token, slug1)
        assert st == "published", f"期望 published，实际={st}"
        print("    status=published ✓")

        print(">>> [1g] 技能广场可见 ...")
        assert skillstore_has_slug(user.token, slug1), "published 应出现在广场"
        print("    广场可见 ✓")

        print(">>> [1h] 重复审核 → 400 ...")
        resp = admin_approve_contribution(admin.token, req_id)
        assert resp.status_code == 400, f"期望 400，实际={resp.status_code}"
        print("    重复审核 400 ✓")

        # ════════════════════════════════════════════════════════════
        # IT-7: 管理员列表字段
        # ════════════════════════════════════════════════════════════
        print()
        print("=== IT-7: /admin/skills 含 status / uploader_* ===")
        skill = admin_find_skill(admin.token, slug1)
        assert skill is not None, "列表应能查到技能"
        assert_fields(skill, ["status", "uploader_id", "uploader_name"], context="admin/skills")
        assert skill["status"] == "published"
        assert skill["uploader_id"] == user.user_id, (
            f"uploader_id 应为员工 id={user.user_id}，实际={skill.get('uploader_id')}"
        )
        assert skill.get("uploader_name") == user.username, (
            f"uploader_name 应为 {user.username}，实际={skill.get('uploader_name')}"
        )
        print(f"    status/uploader_id/uploader_name 正确 ✓  "
              f"(uploader={skill['uploader_name']})")

        # ════════════════════════════════════════════════════════════
        # IT-3: 互斥（先占一个 pending，再测）
        # IT-5: 拒绝 publish → 软删除
        # ════════════════════════════════════════════════════════════
        print()
        print("=== IT-5 + IT-3: 拒绝 publish + 互斥 ===")
        slug_rej = _slug("rej")
        slugs.append(slug_rej)

        print(f">>> [5a] 提交待拒技能 slug={slug_rej} ...")
        resp = contribute_skill(user.token, slug_rej, "IT Reject", "1.0.0")
        assert resp.status_code == 200, f"提交失败: {resp.status_code} {resp.text}"
        req_rej = resp.json().get("request_id")
        assert req_rej, resp.text
        print(f"    提交成功 ✓  request_id={req_rej}")

        print(">>> [3a] 同 slug 再次提交 → 400（互斥）...")
        resp = contribute_skill(user.token, slug_rej, "IT Reject", "1.0.1")
        assert resp.status_code == 400, f"期望 400，实际={resp.status_code} {resp.text}"
        print("    互斥 400 ✓")

        print(">>> [5b] 管理员拒绝 ...")
        resp = admin_reject_contribution(admin.token, req_rej, "内容不符合要求")
        assert resp.status_code == 200, f"拒绝失败: {resp.status_code} {resp.text}"
        print("    已拒绝 ✓")

        print(">>> [5c] Skill 软删除（列表查不到）...")
        st = admin_skill_status(admin.token, slug_rej)
        assert st == "NOT_FOUND", f"拒绝后应软删除，实际 status={st}"
        print("    已软删除 ✓")

        print(">>> [5d/IT-6] 广场不可见 rejected skill ...")
        assert not skillstore_has_slug(user.token, slug_rej)
        print("    广场不可见 ✓")

        print(">>> [5e] 拒绝后可重新提交同 slug ...")
        resp = contribute_skill(user.token, slug_rej, "IT Reject Retry", "1.0.0")
        assert resp.status_code == 200, f"重提失败: {resp.status_code} {resp.text}"
        req_retry = resp.json().get("request_id")
        # 清理：通过后删掉，或直接拒绝/撤回
        resp = admin_reject_contribution(admin.token, req_retry, "清理")
        assert resp.status_code == 200
        print("    重提成功并已清理 ✓")

        # ════════════════════════════════════════════════════════════
        # IT-4: 下架他人/管理员技能 → 403
        # ════════════════════════════════════════════════════════════
        print()
        print("=== IT-4: 下架管理员技能 → 403 ===")
        slug_admin = _slug("adm")
        slugs.append(slug_admin)
        resp = admin_create_skill(admin.token, slug_admin, "Admin Owned", "1.0.0")
        assert resp.status_code == 200, f"管理员创建失败: {resp.status_code} {resp.text}"
        print(f"    管理员技能已创建 slug={slug_admin}")

        skill = admin_find_skill(admin.token, slug_admin)
        assert skill is not None
        # 管理员直传通常 uploader_id=0
        print(f"    uploader_id={skill.get('uploader_id')} status={skill.get('status')}")

        resp = takedown_skill(user.token, slug_admin, "想下架管理员技能")
        assert resp.status_code == 403, f"期望 403，实际={resp.status_code} {resp.text}"
        print("    员工下架管理员技能 403 ✓")

        # ════════════════════════════════════════════════════════════
        # IT-2 + IT-6: 下架 → offline → 广场不可见
        # ════════════════════════════════════════════════════════════
        print()
        print("=== IT-2: 下架 → offline ===")
        print(f">>> [2a] 员工申请下架 slug={slug1} ...")
        resp = takedown_skill(user.token, slug1, "测试下架")
        assert resp.status_code == 200, f"下架申请失败: {resp.status_code} {resp.text}"
        req_td = resp.json().get("request_id")
        assert req_td, resp.text
        print(f"    下架申请成功 ✓  request_id={req_td}")

        print(">>> [2b] 有 pending 时再次下架/提交 → 400 ...")
        resp = takedown_skill(user.token, slug1, "再申请一次")
        assert resp.status_code == 400, f"期望 400，实际={resp.status_code}"
        print("    互斥 400 ✓")

        print(">>> [2c] 管理员通过下架 ...")
        resp = admin_approve_contribution(admin.token, req_td)
        assert resp.status_code == 200, f"下架通过失败: {resp.status_code} {resp.text}"
        st = admin_skill_status(admin.token, slug1)
        assert st == "offline", f"期望 offline，实际={st}"
        print("    status=offline ✓")

        print(">>> [2d/IT-6] 广场不显示 offline ...")
        assert not skillstore_has_slug(user.token, slug1)
        print("    广场不可见 ✓")

        # ════════════════════════════════════════════════════════════
        # 撤回 publish / takedown
        # ════════════════════════════════════════════════════════════
        print()
        print("=== 撤回申请 ===")
        slug_wd = _slug("wd")
        slugs.append(slug_wd)

        print(f">>> [W1] 提交后撤回 slug={slug_wd} ...")
        resp = contribute_skill(user.token, slug_wd, "IT Withdraw", "1.0.0")
        assert resp.status_code == 200, resp.text
        req_wd = resp.json().get("request_id")
        resp = withdraw_contribution(user.token, req_wd)
        assert resp.status_code == 200, f"撤回失败: {resp.status_code} {resp.text}"
        st = admin_skill_status(admin.token, slug_wd)
        assert st == "NOT_FOUND", f"撤回 publish 后 Skill 应软删除，实际={st}"
        print("    撤回 publish 成功，Skill 已软删除 ✓")

        print(">>> [W2] 撤回后可重新提交 ...")
        resp = contribute_skill(user.token, slug_wd, "IT Withdraw", "1.0.0")
        assert resp.status_code == 200, resp.text
        req_wd2 = resp.json().get("request_id")
        resp = admin_approve_contribution(admin.token, req_wd2)
        assert resp.status_code == 200
        print("    重新提交并审核通过 ✓")

        print(">>> [W3] 申请下架后撤回，Skill 仍 published ...")
        resp = takedown_skill(user.token, slug_wd, "先申请再撤回")
        assert resp.status_code == 200, resp.text
        req_wd3 = resp.json().get("request_id")
        resp = withdraw_contribution(user.token, req_wd3)
        assert resp.status_code == 200, resp.text
        st = admin_skill_status(admin.token, slug_wd)
        assert st == "published", f"撤回 takedown 后应仍为 published，实际={st}"
        print("    Skill 仍 published ✓")

        print(">>> [W4] 撤回他人申请 → 403 ...")
        resp = contribute_skill(user.token, _slug("wd2"), "IT WD2", "1.0.0")
        assert resp.status_code == 200, resp.text
        slug_wd2 = _slug("wd2")
        slugs.append(slug_wd2)
        req_other = resp.json().get("request_id")
        resp = withdraw_contribution(user2.token, req_other)
        assert resp.status_code == 403, f"期望 403，实际={resp.status_code}"
        # 申请人自己撤回清理
        withdraw_contribution(user.token, req_other)
        print("    撤回他人申请 403 ✓")

        # ════════════════════════════════════════════════════════════
        # 管理员 offline / online
        # ════════════════════════════════════════════════════════════
        print()
        print("=== 管理员 offline / online ===")
        print(f">>> [A1] 管理员下架 slug={slug_wd} ...")
        resp = admin_offline_skill(admin.token, slug_wd)
        assert resp.status_code == 200, f"下架失败: {resp.status_code} {resp.text}"
        assert admin_skill_status(admin.token, slug_wd) == "offline"
        print("    offline ✓")

        print(">>> [A2] 管理员上架 ...")
        resp = admin_online_skill(admin.token, slug_wd)
        assert resp.status_code == 200, f"上架失败: {resp.status_code} {resp.text}"
        assert admin_skill_status(admin.token, slug_wd) == "published"
        print("    online → published ✓")

        print(">>> [A3] 重复上架 → 404 ...")
        resp = admin_online_skill(admin.token, slug_wd)
        assert resp.status_code == 404, f"期望 404，实际={resp.status_code}"
        print("    重复上架 404 ✓")

        # ════════════════════════════════════════════════════════════
        # IT-M1: 多版本下架 — pending 挂最新行 + 审核整 slug offline
        # ════════════════════════════════════════════════════════════
        print()
        print("=== IT-M1: 多版本下架闭环 ===")
        slug_mv = _slug("mv")
        slugs.append(slug_mv)

        print(f">>> [M1a] 提交并审核通过 1.0.0 slug={slug_mv} ...")
        resp = contribute_skill(user.token, slug_mv, "IT MultiVer", "1.0.0")
        assert resp.status_code == 200, f"提交 1.0.0 失败: {resp.status_code} {resp.text}"
        req_v1 = resp.json().get("request_id")
        resp = admin_approve_contribution(admin.token, req_v1)
        assert resp.status_code == 200, f"审核 1.0.0 失败: {resp.status_code} {resp.text}"
        print("    1.0.0 published ✓")

        print(">>> [M1b] 提交并审核通过 1.0.1 ...")
        resp = contribute_skill(user.token, slug_mv, "IT MultiVer", "1.0.1")
        assert resp.status_code == 200, f"提交 1.0.1 失败: {resp.status_code} {resp.text}"
        req_v2 = resp.json().get("request_id")
        resp = admin_approve_contribution(admin.token, req_v2)
        assert resp.status_code == 200, f"审核 1.0.1 失败: {resp.status_code} {resp.text}"
        skill_latest = admin_find_skill(admin.token, slug_mv)
        assert skill_latest is not None and skill_latest.get("version") == "1.0.1", skill_latest
        print(f"    1.0.1 published ✓  latest_id={skill_latest.get('id')}")

        print(">>> [M1c] 申请下架 ...")
        resp = takedown_skill(user.token, slug_mv, "多版本下架")
        assert resp.status_code == 200, f"下架申请失败: {resp.status_code} {resp.text}"
        req_td_mv = resp.json().get("request_id")
        assert req_td_mv, resp.text
        print(f"    下架申请成功 ✓  request_id={req_td_mv}")

        print(">>> [M1d] 最新列表行 pending_review 可见 ...")
        skill_row = admin_find_skill(admin.token, slug_mv)
        assert skill_row is not None, "列表应能查到技能"
        pr = skill_row.get("pending_review")
        assert pr is not None, (
            f"多版本下架后最新行 pending_review 不应为 null（旧 bug：挂在旧 version id）。row={skill_row}"
        )
        assert pr.get("action_type") == "takedown", pr
        assert int(pr.get("request_id") or 0) == int(req_td_mv), pr
        print(f"    pending_review.action_type=takedown ✓  request_id={pr.get('request_id')}")

        print(">>> [M1e] 审核通过 → 整 slug offline ...")
        resp = admin_approve_contribution(admin.token, req_td_mv)
        assert resp.status_code == 200, f"下架审核失败: {resp.status_code} {resp.text}"
        st = admin_skill_status(admin.token, slug_mv)
        assert st == "offline", f"期望 offline，实际={st}"
        assert not skillstore_has_slug(user.token, slug_mv), "offline 后广场不可见"
        print("    整 slug offline + 广场不可见 ✓")

        # ════════════════════════════════════════════════════════════
        # 鉴权
        # ════════════════════════════════════════════════════════════
        print()
        print("=== 鉴权 ===")
        print(">>> [Auth] 员工审核 → 403 ...")
        resp = admin_approve_contribution(user.token, 1)
        assert resp.status_code == 403, f"期望 403，实际={resp.status_code}"
        print("    员工审核 403 ✓")

        print(">>> [Auth] 员工查审核列表 → 403 ...")
        resp = user_client(user.token).get(
            "/admin/contributions", params={"page": 1}, expect=None, raw=True,
        )
        assert resp.status_code == 403, f"期望 403，实际={resp.status_code}"
        print("    员工查列表 403 ✓")

        print(">>> [Auth] 我的申请列表可读 ...")
        data = my_contributions(user.token, page=1, page_size=20)
        assert "requests" in data or "total" in data, data
        print(f"    my contributions total={data.get('total')} ✓")

        # 列表字段抽样（任意一条含 status 的技能）
        listing = admin_list_skills(admin.token, page=1, page_size=5)
        skills = listing.get("skills") or []
        if skills:
            assert_fields(skills[0], ["status", "uploader_id", "uploader_name"])

        print()
        print("=" * 60)
        print("  技能共建审核 — ALL PASSED")
        print("=" * 60)

    except Exception:
        traceback.print_exc()
        print()
        print("=" * 60)
        print("  技能共建审核 — FAILED")
        print("=" * 60)
        raise
    finally:
        print()
        print(">>> 清理技能 ...")
        for s in slugs:
            cleanup_skill(admin.token, s)
        print(">>> 清理用户 ...")
        teardown_scenario_users(SCENARIO)
        teardown_scenario_users(f"{SCENARIO}-b")
        print("    done")


if __name__ == "__main__":
    main()

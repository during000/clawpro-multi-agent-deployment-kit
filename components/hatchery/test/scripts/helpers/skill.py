"""
技能管理
"""

import time

from helpers import config
from helpers.api import user_client


def user_get_skills(user_token, instance_db_id, retries=3):
    """获取已安装技能列表（带重试，防止偶发截断 JSON）"""
    client = user_client(user_token)
    last_err = None
    for attempt in range(retries):
        resp = client.get(
            "/openclaw/skills", params={"id": instance_db_id},
            expect=200, raw=True,
        )
        try:
            return resp.json()
        except Exception as e:
            last_err = e
            if attempt < retries - 1:
                time.sleep(2)
    raise last_err


def user_add_skill(user_token, instance_db_id, skill_name):
    """手动安装技能（返回原始 Response 以便检查状态码）"""
    return user_client(user_token).post(
        "/openclaw/add-skill",
        data={"id": str(instance_db_id), "skill_name": skill_name},
        timeout=60,
        expect=None, raw=True,
    )

def user_update_skill(user_token, instance_db_id, slug):
    """同步更新由 Admin 下发的技能。"""
    return user_client(user_token).post(
        "/openclaw/update-skill",
        data={"id": str(instance_db_id), "slug": slug},
        timeout=180,
        expect=None, raw=True,
    )


def user_uninstall_skill(user_token, instance_db_id, slug):
    """同步卸载运行时技能；Admin 下发技能同时维护 task/record。"""
    return user_client(user_token).post(
        "/openclaw/uninstall-skill",
        data={"id": str(instance_db_id), "slug": slug},
        timeout=120,
        expect=None, raw=True,
    )


def user_get_install_skills(user_token, instance_db_id):
    """查询初始技能包安装状态"""
    return user_client(user_token).get(
        "/openclaw/install-skills", params={"id": instance_db_id},
    )


def user_retry_failed_skills(user_token, instance_db_id):
    """重试安装失败的技能"""
    return user_client(user_token).post(
        "/openclaw/retry-failed-skills",
        params={"id": instance_db_id},
        timeout=60,
    )


def user_cancel_failed_skills(user_token, instance_db_id):
    """取消安装失败的技能"""
    return user_client(user_token).post(
        "/openclaw/cancel-failed-skills",
        params={"id": instance_db_id},
        timeout=60,
    )


def wait_skills_installed(user_token, instance_db_id, timeout=None):
    """轮询等待所有技能安装完成（install_status=2），返回 install-skills 数据"""
    timeout = timeout or config.SKILL_POLL_TIMEOUT
    start = time.time()

    while True:
        elapsed = time.time() - start
        data = user_get_install_skills(user_token, instance_db_id)
        skills = data.get("skills", [])

        if skills and all(s.get("install_status") in (2, 3) for s in skills):
            # 全部完成或失败，不再等待
            failed = [s for s in skills if s.get("install_status") == 3]
            if failed:
                names = [s.get("name", s.get("skill_name", "unknown")) for s in failed]
                print(f"    ⚠ 部分技能安装失败（已跳过）: {names}")
                for s in failed:
                    print(f"      - {s.get('name', 'unknown')}: {s.get('error_message', '无错误信息')}")
            return data

        if elapsed > timeout:
            pending = [s for s in skills if s.get("install_status") != 2]
            pending_info = [
                f"{s.get('skill_name', 'unknown')}(status={s.get('install_status')})"
                for s in pending
            ]
            raise TimeoutError(
                f"技能安装在 {timeout}s 内未完成，未完成项: {pending_info}"
            )

        time.sleep(config.POLL_INTERVAL)

#!/usr/bin/env python3
"""
用户技能更新/卸载端到端测试。

覆盖：
  - GET /openclaw/skills 的 Admin 下发版本字段与 name/slug 分离
  - POST /openclaw/update-skill 认证、参数、Enterprise/Public 更新和幂等
  - 用户直接安装的 Enterprise 技能物理卸载与幂等
  - OpenClaw、Hermes、LightClaw ACE 三运行时脚本路由
  - skill_update / skill_uninstall 审计记录
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers
from helpers import config
from helpers.admin_skill import (
    admin_create_skill,
    admin_distribute_skill,
    wait_skill_settled,
)
from helpers.hermes import HERMES_AGENT_TYPE, setup_hermes_instance
from helpers.api import admin_client, anon, seed, user_client
from helpers.instance import (
    InstanceContext,
    create_instance,
    get_instance_db_id,
    wait_instance_ready,
)

ACE_AGENT_TYPE = "lightclawace"
PUBLIC_SLUG = os.getenv("PUBLIC_DISTRIBUTED_SKILL_SLUG", "self-improving-agent")
PUBLIC_OLD_VERSION = os.getenv("PUBLIC_DISTRIBUTED_SKILL_OLD_VERSION", "3.0.6")
POLL_TIMEOUT = int(os.getenv("SKILL_OPERATION_POLL_TIMEOUT", "300"))


def ensure_agent_type_and_image(agent_type):
    response = seed.post(
        "/admin/agent-types/enabled",
        data={"agent_type": agent_type, "enabled": "true"},
        expect=None,
        raw=True,
    )
    if response.status_code != 200:
        raise RuntimeError(f"启用 {agent_type} 失败: {response.status_code} {response.text}")

    data = seed.get("/admin/images")
    images = [item for item in data.get("images", []) if item.get("agent_type") == agent_type]
    if not images:
        raise RuntimeError(f"系统中没有 {agent_type} 镜像")
    if any(item.get("enabled") for item in images):
        return
    image_id = images[0].get("id") or images[0].get("ID")
    response = seed.post("/admin/images/enable", params={"id": image_id}, expect=None, raw=True)
    if response.status_code != 200:
        raise RuntimeError(f"启用 {agent_type} 镜像失败: {response.status_code} {response.text}")


def setup_runtime_instance(user_token, scenario, agent_type):
    if agent_type == HERMES_AGENT_TYPE:
        return setup_hermes_instance(user_token, scenario)
    if agent_type == ACE_AGENT_TYPE:
        ensure_agent_type_and_image(agent_type)
    name = f"{config.INSTANCE_NAME_PREFIX}{agent_type}-{scenario}-{int(time.time())}"
    created = create_instance(user_token, name, agent_type=agent_type)
    assert created.get("ok"), created
    instance_id = created.get("instance_id", "")
    db_id = get_instance_db_id(user_token, instance_id)
    wait_instance_ready(user_token, db_id)
    time.sleep(10)
    return InstanceContext(
        db_id=db_id,
        instance_id=instance_id,
        user_token=user_token,
    )


def runtime_skills(user_token, instance_db_id):
    data = helpers.user_get_skills(user_token, instance_db_id)
    if isinstance(data, list):
        return data
    return data.get("skills", data.get("data", []))


def find_runtime_skill(user_token, instance_db_id, slug, name=None):
    for item in runtime_skills(user_token, instance_db_id):
        if item.get("slug") == slug or (name is not None and item.get("name") == name):
            return item
    return None


def wait_runtime_skill(user_token, instance_db_id, slug, *, version=None, absent=False, name=None):
    start = time.time()
    last = None
    while time.time() - start <= POLL_TIMEOUT:
        last = find_runtime_skill(user_token, instance_db_id, slug, name)
        if absent and last is None:
            return None
        if not absent and last is not None and (version is None or last.get("version") == version):
            return last
        time.sleep(config.POLL_INTERVAL)
    raise TimeoutError(
        f"等待技能状态超时: instance={instance_db_id} slug={slug} "
        f"version={version} absent={absent} last={last}"
    )


def assert_response(response, status=200):
    if response.status_code != status:
        raise AssertionError(f"HTTP {response.status_code}, want {status}: {response.text[:500]}")
    return response.json()


def assert_audit(admin_token, action, instance_db_id):
    client = admin_client(admin_token)
    deadline = time.time() + 30
    resource_id = str(instance_db_id)
    while time.time() < deadline:
        data = client.get(
            "/admin/audit",
            params={"action": action, "resource_id": resource_id, "page_size": 20},
        )
        logs = data.get("logs", [])
        if any(
            item.get("action") == action
            and item.get("resource") == "skill"
            and item.get("resource_id") == resource_id
            for item in logs
        ):
            return
        time.sleep(2)
    raise AssertionError(f"未找到审计 action={action} resource_id={resource_id}")


def test_auth_and_params(user_token, own_instance_id, foreign_token):
    for path in ("/openclaw/update-skill", "/openclaw/uninstall-skill"):
        anon.post(path, data={"id": own_instance_id, "slug": "missing"}, expect=(401, 403))
        client = user_client(user_token)
        foreign_client = user_client(foreign_token)
        assert client.post(path, data={"slug": "missing"}, expect=None, raw=True).status_code == 400
        assert client.post(path, data={"id": own_instance_id}, expect=None, raw=True).status_code == 400
        assert foreign_client.post(
            path,
            data={"id": own_instance_id, "slug": "missing"},
            expect=None,
            raw=True,
        ).status_code == 400


def distribute_enterprise_v1(admin_token, slug, instances):
    response = admin_create_skill(admin_token, slug, "Admin Distributed Skill", "1.0.0")
    assert_response(response)
    response = admin_distribute_skill(admin_token, slug, "1.0.0", [item.db_id for item in instances])
    assert_response(response)
    for item in instances:
        status = wait_skill_settled(admin_token, slug, item.db_id)
        assert status.get("status") == "installed", status
        listed = wait_runtime_skill(item.user_token, item.db_id, slug, version="1.0.0")
        assert listed.get("name") and listed.get("can_uninstall") is True, listed


def update_and_uninstall_enterprise(admin_token, slug, instances):
    response = admin_create_skill(admin_token, slug, "Admin Distributed Skill", "2.0.0")
    assert_response(response)

    for item in instances:
        listed = wait_runtime_skill(item.user_token, item.db_id, slug, version="1.0.0")
        assert listed.get("latest_version") == "2.0.0", listed
        assert listed.get("update_available") is True, listed
        assert listed.get("can_uninstall") is True, listed

        response = helpers.user_update_skill(item.user_token, item.db_id, slug)
        data = assert_response(response)
        assert data == {
            "slug": slug,
            "updated": True,
            "old_version": "1.0.0",
            "version": "2.0.0",
        }, data
        listed = wait_runtime_skill(item.user_token, item.db_id, slug, version="2.0.0")
        assert listed.get("update_available") is False, listed

        data = assert_response(helpers.user_update_skill(item.user_token, item.db_id, slug))
        assert data.get("updated") is False and data.get("version") == "2.0.0", data

        display_name = listed["name"]
        data = assert_response(helpers.user_uninstall_skill(item.user_token, item.db_id, slug))
        assert data == {"slug": slug, "uninstalled": True, "version": "2.0.0"}, data
        wait_runtime_skill(item.user_token, item.db_id, slug, absent=True, name=display_name)

        data = assert_response(helpers.user_uninstall_skill(item.user_token, item.db_id, slug))
        assert data == {"slug": slug, "uninstalled": True}, data


def public_lifecycle(admin_token, instance):
    response = admin_distribute_skill(
        admin_token,
        PUBLIC_SLUG,
        PUBLIC_OLD_VERSION,
        [instance.db_id],
        source="public",
    )
    assert_response(response)
    status = wait_skill_settled(
        admin_token,
        PUBLIC_SLUG,
        instance.db_id,
        source="public",
        version=PUBLIC_OLD_VERSION,
    )
    assert status.get("status") in ("installed", "outdated"), status
    listed = wait_runtime_skill(
        instance.user_token,
        instance.db_id,
        PUBLIC_SLUG,
        version=PUBLIC_OLD_VERSION,
    )
    latest = listed.get("latest_version")
    assert latest, listed
    assert listed.get("name") and listed.get("name") != PUBLIC_SLUG, listed
    assert listed.get("update_available") is True, (
        f"Public fixture 不再是旧版本，请更新 PUBLIC_DISTRIBUTED_SKILL_OLD_VERSION: {listed}"
    )
    assert listed.get("can_uninstall") is True, listed

    data = assert_response(helpers.user_update_skill(instance.user_token, instance.db_id, PUBLIC_SLUG))
    assert data.get("updated") is True, data
    assert data.get("old_version") == PUBLIC_OLD_VERSION, data
    assert data.get("version") == latest, data
    assert "latest_version" not in data, data
    listed = wait_runtime_skill(instance.user_token, instance.db_id, PUBLIC_SLUG, version=latest)

    display_name = listed["name"]
    data = assert_response(helpers.user_uninstall_skill(instance.user_token, instance.db_id, PUBLIC_SLUG))
    assert data.get("uninstalled") is True and data.get("version") == latest, data
    wait_runtime_skill(
        instance.user_token,
        instance.db_id,
        PUBLIC_SLUG,
        absent=True,
        name=display_name,
    )



def direct_uninstall_lifecycle(admin_token, instance):
    slug = f"direct-uninstall-skill-{int(time.time())}"
    response = admin_create_skill(admin_token, slug, "Direct Uninstall Skill", "1.0.0")
    assert_response(response)
    try:
        response = user_client(instance.user_token).post(
            "/openclaw/add-skill",
            data={"id": str(instance.db_id), "skill_name": slug, "source": "enterprise"},
            timeout=120,
            expect=None,
            raw=True,
        )
        assert_response(response)
        listed = wait_runtime_skill(instance.user_token, instance.db_id, slug)
        assert listed.get("slug") == slug, listed
        assert "version" not in listed and "latest_version" not in listed, listed
        assert listed.get("can_uninstall") is True, listed

        display_name = listed["name"]
        data = assert_response(helpers.user_uninstall_skill(instance.user_token, instance.db_id, slug))
        assert data == {"slug": slug, "uninstalled": True}, data
        wait_runtime_skill(
            instance.user_token,
            instance.db_id,
            slug,
            absent=True,
            name=display_name,
        )
        data = assert_response(helpers.user_uninstall_skill(instance.user_token, instance.db_id, slug))
        assert data == {"slug": slug, "uninstalled": True}, data
    finally:
        admin_client(admin_token).post(
            "/admin/skills/delete",
            data={"slug": slug, "cascade": "true"},
            expect=None,
            raw=True,
        )


def main():
    helpers.check_env()
    scenario = "skill-operations"
    helpers.teardown_scenario_users(scenario)
    admin = helpers.setup_admin(scenario)
    instances = []

    try:
        user = helpers.setup_user(admin.token, scenario, instance_quota=3)
        foreign_user = helpers.setup_user(admin.token, f"{scenario}-foreign", instance_quota=1)

        instances.append(setup_runtime_instance(user.token, scenario, config.AGENT_TYPE))
        instances.append(setup_runtime_instance(user.token, scenario, HERMES_AGENT_TYPE))
        instances.append(setup_runtime_instance(user.token, scenario, ACE_AGENT_TYPE))

        test_auth_and_params(user.token, instances[0].db_id, foreign_user.token)

        slug = f"admin-distributed-skill-{int(time.time())}"
        distribute_enterprise_v1(admin.token, slug, instances)
        update_and_uninstall_enterprise(admin.token, slug, instances)
        public_lifecycle(admin.token, instances[0])

        direct_uninstall_lifecycle(admin.token, instances[0])
        assert_audit(admin.token, "skill_update", instances[0].db_id)
        assert_audit(admin.token, "skill_uninstall", instances[0].db_id)
        print("\n用户技能更新/卸载集成测试通过")
    except Exception as exc:
        print(f"\n用户技能更新/卸载集成测试失败: {exc}")
        traceback.print_exc()
        raise
    finally:
        helpers.teardown_scenario_users(scenario)
        helpers.teardown_scenario_users(f"{scenario}-foreign")


if __name__ == "__main__":
    main()

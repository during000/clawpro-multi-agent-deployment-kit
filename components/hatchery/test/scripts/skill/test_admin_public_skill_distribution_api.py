#!/usr/bin/env python3
"""
公共技能分发管控端 API 覆盖测试。

定位：契约/参数覆盖测试，不是公共技能真实下发 E2E。
设计原则与 test_instance_skillstore.py 一致：覆盖 route/params，但避开真实外部副作用。
查询接口只做只读校验；下发/卸载接口故意走参数校验短路（空 instance_ids +
顶层技能字段与 skills[] 混用），确保不会下载 SkillHub 包、创建任务或执行 TAT。
"""

import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import IDENTIFIER, health_check, run_tests, seed

PREFIX = f"it-public-skill-{IDENTIFIER or int(time.time())}"
PUBLIC_SKILLSET = f"{PREFIX}-skillset"
PUBLIC_SKILL = f"{PREFIX}-skill"
UNMATCHED_SEARCH = f"{PREFIX}-no-instance"


def assert_query_response(data, list_key):
    """SMH 启用时返回业务列表；未启用时允许 403 错误以保持脚本可移植。"""
    if list_key in data:
        assert isinstance(data[list_key], list), data
        return
    assert "error" in data, data


def assert_error_response(data):
    """参数校验或 SMH 未启用都应返回结构化错误。"""
    assert "error" in data, data


def test_01_get_public_skill_instances_params():
    """GET /admin/skills/instances 覆盖 public source 查询参数。"""
    data = seed.get(
        "/admin/skills/instances",
        params={
            "source": "public",
            "slug": PUBLIC_SKILL,
            "version": "latest",
            "status": "uninstalled",
            "search": UNMATCHED_SEARCH,
            "instance_type": "openclaw",
            "group_id": "0",
            "page": 1,
            "page_size": 1,
        },
        expect=(200, 403),
    )
    assert_query_response(data, "instances")
    print("    GET public skill instance params covered")


def test_02_post_public_skillset_instances_params():
    """POST /admin/skills/instances 覆盖公共技能包批量查询。"""
    data = seed.post(
        "/admin/skills/instances",
        json={
            "source": "public",
            "source_skillset_slug": PUBLIC_SKILLSET,
            "skills": [
                {
                    "source": "public",
                    "slug": PUBLIC_SKILL,
                    "version": "latest",
                    "source_skillset_slug": PUBLIC_SKILLSET,
                }
            ],
            "status": "uninstalled",
            "search": UNMATCHED_SEARCH,
            "instance_type": "openclaw",
            "group_id": "0",
        },
        expect=(200, 403),
    )
    assert_query_response(data, "instances")
    print("    POST public skillset instance params covered")


def test_03_public_skillset_tasks_params():
    """GET /admin/skills/tasks 覆盖公共技能包聚合查询参数。"""
    data = seed.get(
        "/admin/skills/tasks",
        params={
            "source": "public",
            "source_skillset_slug": PUBLIC_SKILLSET,
            "type": "distribute",
            "page": 1,
            "page_size": 1,
        },
        expect=(200, 403),
    )
    assert_query_response(data, "tasks")
    print("    public skillset task query params covered")


def test_04_public_skill_task_batch_id_param():
    """GET /admin/skills/tasks 覆盖 batch_id 查询参数。"""
    data = seed.get(
        "/admin/skills/tasks",
        params={
            "source": "public",
            "batch_id": f"{PREFIX}-batch",
            "type": "distribute",
            "page": 1,
            "page_size": 1,
        },
        expect=(200, 403),
    )
    assert_query_response(data, "tasks")
    print("    public skill task batch_id param covered")


def test_05_batch_distribute_rejects_mixed_top_level_fields():
    """POST /admin/skills/distribute 覆盖批量 body 参数和歧义校验。"""
    data = seed.post(
        "/admin/skills/distribute",
        json={
            "source": "public",
            "slug": "top-level-skill",
            "version": "latest",
            "source_skillset_slug": PUBLIC_SKILLSET,
            "instance_ids": [],
            "skills": [
                {
                    "source": "public",
                    "slug": PUBLIC_SKILL,
                    "version": "latest",
                    "source_skillset_slug": PUBLIC_SKILLSET,
                }
            ],
        },
        expect=(400, 403),
    )
    assert_error_response(data)
    print("    distribute batch body params covered")


def test_06_batch_uninstall_rejects_mixed_top_level_fields():
    """POST /admin/skills/uninstall 覆盖批量 body 参数和歧义校验。"""
    data = seed.post(
        "/admin/skills/uninstall",
        json={
            "source": "public",
            "slug": "top-level-skill",
            "source_skillset_slug": PUBLIC_SKILLSET,
            "instance_ids": [],
            "skills": [
                {
                    "source": "public",
                    "slug": PUBLIC_SKILL,
                    "source_skillset_slug": PUBLIC_SKILLSET,
                }
            ],
        },
        expect=(400, 403),
    )
    assert_error_response(data)
    print("    uninstall batch body params covered")


def main():
    health_check()
    run_tests(globals(), title="公共技能分发 API 覆盖", ordered=True, abort_on_fail=True)


if __name__ == "__main__":
    main()

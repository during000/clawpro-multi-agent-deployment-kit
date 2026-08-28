#!/usr/bin/env python3
"""
集成测试：查询用户组关联模型 (associated-models)

覆盖接口：
    GET /admin/user-groups/associated-models?group_id=N

聚焦：
    1. 正常路径：新建用户组未关联任何模型，count=0、models=[]
    2. 响应结构正确（dict、含 count/models）
    3. 异常路径见 validations 文件（缺 group_id / 格式错误）
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    seed,
    health_check, run_tests,
)
from helpers.user_groups import (  # noqa: E402
    extract_group_id, pick_group, list_all_groups, find_groups_by_prefix,
    cleanup_by_prefix,
)

PREFIX = f"it-grpam-{int(time.time())}"

state = {"group_id": None}


def _ensure_group():
    if state["group_id"]:
        return state["group_id"]
    name = f"{PREFIX}-G"
    resp = seed.post("/admin/user-groups/create",
                     json={"name": name, "description": "associated-models 专项"},
                     expect=None, raw=True)
    gid = None
    if resp.status_code == 200:
        gid = extract_group_id(resp.json())
    if not gid:
        target = pick_group(list_all_groups(), name=name)
        gid = target and (target.get("id") or target.get("ID"))
    state["group_id"] = gid
    return gid


def test_01_prepare():
    """前置：创建一个全新用户组"""
    gid = _ensure_group()
    assert gid, "用户组创建失败"
    print(f"    group_id={gid}")


def test_02_associated_models_zero():
    """查询关联模型：新建组未绑定任何模型可见性，期望 count=0"""
    gid = state["group_id"]
    resp = seed.get("/admin/user-groups/associated-models",
                    params={"group_id": gid}, raw=True)
    data = resp.json()
    assert isinstance(data, dict), f"返回非对象: {data}"
    count = data.get("count") if data.get("count") is not None else data.get("Count")
    models = data.get("models") or data.get("Models") or []
    assert isinstance(models, list), \
        f"models 应为数组，实际 {type(models).__name__}"
    assert count is not None, \
        f"响应缺少 count 字段: keys={list(data.keys())}"
    assert count == 0 and not models, \
        f"期望空关联，实际 count={count} models={models}"


def test_03_response_structure():
    """查询关联模型：响应应包含 count 与 models 两个字段，类型正确"""
    gid = state["group_id"]
    resp = seed.get("/admin/user-groups/associated-models",
                    params={"group_id": gid}, raw=True)
    data = resp.json()
    assert isinstance(data, dict), f"返回非对象: {data}"
    keys_lower = {k.lower() for k in data.keys()}
    assert "count" in keys_lower, \
        f"缺少 count 字段，实际 keys={list(data.keys())}"
    assert "models" in keys_lower, \
        f"缺少 models 字段，实际 keys={list(data.keys())}"
    count = data.get("count") if data.get("count") is not None else data.get("Count")
    models = data.get("models") or data.get("Models") or []
    assert isinstance(count, int), \
        f"count 应为 int，实际 {type(count).__name__}"
    assert isinstance(models, list), \
        f"models 应为 list，实际 {type(models).__name__}"


def cleanup():
    """兜底清理"""
    try:
        cleanup_by_prefix(group_prefix=PREFIX)
    except Exception as e:
        print(f"[cleanup] 异常: {e}")


def main():
    health_check()
    try:
        run_tests(globals(), title="用户组关联模型",
                  ordered=True, abort_on_fail=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()

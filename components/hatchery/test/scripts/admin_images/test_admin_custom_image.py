#!/usr/bin/env python3
"""
自定义镜像管理集成测试

覆盖接口：
    GET  /admin/images         查询镜像列表
    GET  /admin/images/cloud   查询云上可导入镜像
    POST /admin/images/import  导入自定义镜像
    POST /admin/images/delete  删除镜像

══════════════════════════════════════════════════════════════════════════
一、保活名单
══════════════════════════════════════════════════════════════════════════

    集成测试运行前确保名单中的镜像都已导入，缺失则补导，**只加不删**。
    保活名单定义在 helpers/image_keepalive.py，被本测试和升级测试共用，
    保证升级测试在选取版本前保活镜像已就绪，不依赖编排器的执行顺序。

══════════════════════════════════════════════════════════════════════════
二、基础 CRUD 测试（test_01 ~ test_05）
══════════════════════════════════════════════════════════════════════════

    1. 查询自定义镜像列表 → 确认可正常获取
    2. 查询云上可导入镜像 → 随机选一个
    3. 导入选中的镜像
    4. 检查是否添加成功
    5. 删除添加的镜像 → 确认已删除

    导入的测试镜像在测试结束后会被删除（cleanup 兜底）。
    若云上没有可导入的镜像，基础 CRUD 测试整体 SKIP。
"""
import os
import random
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import seed, health_check, run_tests
from helpers.client import GREEN, RED, YELLOW, BOLD
from helpers.image_keepalive import (
    ensure_keepalive_images,
    find_image_by_image_id,
    KEEPALIVE_IMAGE_IDS,
)

# 基础 CRUD 测试用的 agent 信息（当 cloud 未返回 agentType 时使用）
TEST_AGENT_TYPE = "openclaw"
TEST_AGENT_VERSION = "2026.8.3"


# ─── API 工具函数 ───

def get_admin_images():
    """GET /admin/images → 返回 images 列表"""
    resp = seed.get("/admin/images", expect=None, raw=True, timeout=30)
    if resp.status_code != 200:
        raise RuntimeError(
            f"GET /admin/images 失败 ({resp.status_code}): {resp.text[:200]}"
        )
    return resp.json().get("images", [])


def get_cloud_images():
    """GET /admin/images/cloud → 返回云上可导入镜像列表（数组）"""
    resp = seed.get("/admin/images/cloud", expect=None, raw=True, timeout=60)
    if resp.status_code != 200:
        raise RuntimeError(
            f"GET /admin/images/cloud 失败 ({resp.status_code}): {resp.text[:200]}"
        )
    data = resp.json()
    # cloud 接口返回的是数组（jsonOK(w, result)），兼容 dict 包装
    if isinstance(data, list):
        return data
    return data.get("images", [])


def import_image(image_id, agent_type, agent_version, image_name=""):
    """POST /admin/images/import → 导入镜像，返回 raw response"""
    form = {
        "image_id": image_id,
        "agent_type": agent_type,
        "agent_version": agent_version,
    }
    if image_name:
        form["image_name"] = image_name
    return seed.post(
        "/admin/images/import", data=form, expect=None, raw=True, timeout=60
    )


def delete_image(db_id):
    """POST /admin/images/delete?id=X → 删除镜像，返回 raw response"""
    return seed.post(
        "/admin/images/delete", params={"id": db_id}, expect=None, raw=True, timeout=30
    )


# ─── 基础 CRUD 测试 ───

# 跨用例共享状态
_picked_image = None       # 从 cloud 选中的镜像
_imported_db_id = None     # 导入后的 DB id（用于删除）


def test_01_list_images():
    """查询自定义镜像列表 → 确认可正常获取"""
    images = get_admin_images()
    assert isinstance(images, list), f"images 不是列表: {type(images)}"
    print(f"    当前镜像数: {len(images)}")
    for img in images[:5]:
        print(f"      id={img.get('id')} image_id={img.get('image_id')} "
              f"type={img.get('agent_type')} version={img.get('agent_version')} "
              f"enabled={img.get('enabled')}")
    if len(images) > 5:
        print(f"      ... (共 {len(images)} 个)")


def test_02_pick_cloud_image():
    """查询云上可导入镜像 → 随机选一个"""
    global _picked_image
    cloud = get_cloud_images()
    print(f"    云上可导入镜像数: {len(cloud)}")
    if not cloud:
        print(YELLOW("SKIP: 云上没有可导入的镜像，跳过基础 CRUD 测试"))
        return
    for img in cloud[:5]:
        print(f"      imageId={img.get('imageId')} name={img.get('imageName')} "
              f"public={img.get('public')} type={img.get('agentType')}")
    if len(cloud) > 5:
        print(f"      ... (共 {len(cloud)} 个)")
    # 优先选候选公共镜像（cloud 返回了 agentType/agentVersion）
    candidates = [
        img for img in cloud
        if img.get("agentType") and img.get("agentVersion")
    ]
    pool = candidates if candidates else cloud
    _picked_image = random.choice(pool)
    print(f"    选中: imageId={_picked_image.get('imageId')} "
          f"name={_picked_image.get('imageName')}")


def test_03_import_image():
    """导入选中的镜像"""
    global _imported_db_id
    if _picked_image is None:
        print(YELLOW("SKIP: 未选中镜像（test_02 已 SKIP）"))
        return
    image_id = _picked_image.get("imageId")
    # 优先用 cloud 返回的 agent 信息，否则用默认值
    agent_type = _picked_image.get("agentType") or TEST_AGENT_TYPE
    agent_version = _picked_image.get("agentVersion") or TEST_AGENT_VERSION
    image_name = _picked_image.get("imageName", "")

    print(f"    导入 image_id={image_id} type={agent_type} "
          f"version={agent_version} name={image_name}")
    resp = import_image(image_id, agent_type, agent_version, image_name)
    assert resp.status_code == 200, (
        f"导入失败 ({resp.status_code}): {resp.text[:200]}"
    )
    print(GREEN(f"    导入成功 ✓"))

    # 查询获取 db_id
    images = get_admin_images()
    found = find_image_by_image_id(images, image_id)
    assert found is not None, f"导入后未在列表中找到 image_id={image_id}"
    _imported_db_id = found.get("id") or found.get("ID")
    print(f"    db_id={_imported_db_id}")


def test_04_verify_imported():
    """检查是否添加成功"""
    if _imported_db_id is None:
        print(YELLOW("SKIP: 未导入镜像（test_03 已 SKIP）"))
        return
    images = get_admin_images()
    found = find_image_by_image_id(images, _picked_image.get("imageId"))
    assert found is not None, "导入后镜像不在列表中"
    expected_type = _picked_image.get("agentType") or TEST_AGENT_TYPE
    assert found.get("agent_type") == expected_type, (
        f"agent_type 不匹配: 期望={expected_type} 实际={found.get('agent_type')}"
    )
    print(GREEN(f"    镜像已在列表中 ✓ id={found.get('id')} "
                f"image_id={found.get('image_id')} "
                f"type={found.get('agent_type')}"))


def test_05_delete_image():
    """删除添加的镜像 → 确认已删除"""
    if _imported_db_id is None:
        print(YELLOW("SKIP: 未导入镜像（test_03 已 SKIP）"))
        return
    resp = delete_image(_imported_db_id)
    assert resp.status_code == 200, (
        f"删除失败 ({resp.status_code}): {resp.text[:200]}"
    )
    print(GREEN(f"    删除成功 ✓ db_id={_imported_db_id}"))

    # 确认已删除
    images = get_admin_images()
    found = find_image_by_image_id(images, _picked_image.get("imageId"))
    assert found is None, "删除后镜像仍在列表中"
    print(GREEN(f"    镜像已不在列表中 ✓"))


# ─── 清理 ───

def cleanup():
    """清理：删除基础 CRUD 测试导入的镜像（保活名单的不删）"""
    # 1. 优先用 db_id 删除
    if _imported_db_id is not None:
        try:
            resp = delete_image(_imported_db_id)
            if resp.status_code == 200:
                print(f"    [cleanup] 删除测试镜像 db_id={_imported_db_id} ✓")
            elif resp.status_code == 404:
                pass  # 已被 test_05 删除，正常
            else:
                print(f"    [cleanup] 删除测试镜像失败 "
                      f"({resp.status_code}): {resp.text[:200]}")
        except Exception as e:
            print(f"    [cleanup] {e}")
        return

    # 2. db_id 丢失时，通过 image_id 查找并删除（兜底）
    if _picked_image is not None:
        image_id = _picked_image.get("imageId")
        # 保活名单里的镜像不删
        if image_id and image_id not in KEEPALIVE_IMAGE_IDS:
            try:
                images = get_admin_images()
                found = find_image_by_image_id(images, image_id)
                if found:
                    db_id = found.get("id") or found.get("ID")
                    delete_image(db_id)
                    print(f"    [cleanup] 删除测试镜像 image_id={image_id} "
                          f"db_id={db_id} ✓")
            except Exception as e:
                print(f"    [cleanup] {e}")


# ─── 入口 ───

def main():
    health_check()
    print()
    # 保活名单：确保名单中的镜像都已导入（只加不删）
    ensure_keepalive_images()
    print()
    try:
        run_tests(
            globals(),
            title="自定义镜像管理集成测试",
            ordered=True,
            abort_on_fail=False,
        )
    finally:
        cleanup()


if __name__ == "__main__":
    main()

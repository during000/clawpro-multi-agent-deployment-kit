"""
镜像保活名单管理

集成测试运行前确保名单中的镜像都已导入，只加不删。
被 test_admin_custom_image.py 和 test_instance_upgrade.py 共用，
保证升级测试在选取版本前保活镜像已就绪，不依赖编排器的执行顺序。

版本格式约束（对齐后端 ValidateAgentVersion）：
    openclaw:       YYYY.M.D      (如 2026.4.23)
    hermes / ace:   X.Y.Z semver  (如 0.12.0)
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import seed
from helpers.client import GREEN, RED, YELLOW, BOLD


# 保活名单：集成测试运行前确保这些镜像都已导入，只加不删。
# 每个条目: {image_id, agent_type, agent_version, image_name}
#   - agent_type:  openclaw / hermes / lightclawace
#   - agent_version: openclaw 用 YYYY.M.D；hermes/ace 用 semver X.Y.Z
# 命名格式：Product-OS-Version（对齐云镜像 os_name 实际系统）
#   OpenClaw       → 全部 Ubuntu Server 24.04
#   Hermes         → Ubuntu / TencentOS（按 os_name 区分）
#   LightClawACE   → 全部 TencentOS Server 4
KEEPALIVE_IMAGES = [
    # ── OpenClaw (agent_version = YYYY.M.D, 全部 Ubuntu) ──
    {"image_id": "img-g6ubvgcu", "agent_type": "openclaw", "agent_version": "2026.3.8",   "image_name": "OpenClaw-Ubuntu-2026.3.8"},
    {"image_id": "img-c82zrkue", "agent_type": "openclaw", "agent_version": "2026.3.13",  "image_name": "OpenClaw-Ubuntu-2026.3.13"},
    {"image_id": "img-e3ozf2h8", "agent_type": "openclaw", "agent_version": "2026.4.15",  "image_name": "OpenClaw-Ubuntu-2026.4.15"},
    {"image_id": "img-r4sm0w84", "agent_type": "openclaw", "agent_version": "2026.4.23",  "image_name": "OpenClaw-Ubuntu-2026.4.23"},
    {"image_id": "img-6g4n95la", "agent_type": "openclaw", "agent_version": "2026.4.27",  "image_name": "OpenClaw-Ubuntu-2026.4.27"},
    {"image_id": "img-bczt10g8", "agent_type": "openclaw", "agent_version": "2026.5.7",   "image_name": "OpenClaw-Ubuntu-2026.5.7"},
    {"image_id": "img-jd1ncmwe", "agent_type": "openclaw", "agent_version": "2026.5.18",  "image_name": "OpenClaw-Ubuntu-2026.5.18"},
    {"image_id": "img-k5fazwxy", "agent_type": "openclaw", "agent_version": "2026.5.28",  "image_name": "OpenClaw-Ubuntu-2026.5.28"},
    {"image_id": "img-2ikqhk02", "agent_type": "openclaw", "agent_version": "2026.6.10",  "image_name": "OpenClaw-Ubuntu-2026.6.10"},
    # ── Hermes (agent_version = semver X.Y.Z) ──
    {"image_id": "img-q9w1r6aq", "agent_type": "hermes", "agent_version": "0.12.0", "image_name": "Hermes-Ubuntu-0.12.0"},
    {"image_id": "img-6fdzcbu2", "agent_type": "hermes", "agent_version": "0.14.0", "image_name": "Hermes-Ubuntu-0.14.0"},
    {"image_id": "img-eomgpa5e", "agent_type": "hermes", "agent_version": "0.16.0", "image_name": "Hermes-TencentOS-0.16.0"},
    {"image_id": "img-8i6a3q4q", "agent_type": "hermes", "agent_version": "0.17.0", "image_name": "Hermes-Ubuntu-0.17.0"},
    {"image_id": "img-259ovk22", "agent_type": "hermes", "agent_version": "0.18.0", "image_name": "Hermes-Ubuntu-0.18.0"},
    # ── LightClawACE (agent_version = semver X.Y.Z, 全部 TencentOS) ──
    {"image_id": "img-6wf08nm4", "agent_type": "lightclawace", "agent_version": "0.1.18", "image_name": "ACE-TencentOS-0.1.18"},
    {"image_id": "img-awzr9bu4", "agent_type": "lightclawace", "agent_version": "0.1.19", "image_name": "ACE-TencentOS-0.1.19"},
    {"image_id": "img-al64bxpg", "agent_type": "lightclawace", "agent_version": "0.1.26", "image_name": "ACE-TencentOS-0.1.26"},
    {"image_id": "img-07o821u4", "agent_type": "lightclawace", "agent_version": "1.0.8",  "image_name": "ACE-TencentOS-1.0.8"},
]

# 保活名单 image_id 集合（用于 cleanup 判断"不删"逻辑）
KEEPALIVE_IMAGE_IDS = {item["image_id"] for item in KEEPALIVE_IMAGES}


def _get_admin_images():
    """GET /admin/images → 返回 images 列表"""
    resp = seed.get("/admin/images", expect=None, raw=True, timeout=30)
    if resp.status_code != 200:
        raise RuntimeError(
            f"GET /admin/images 失败 ({resp.status_code}): {resp.text[:200]}"
        )
    return resp.json().get("images", [])


def _import_image(image_id, agent_type, agent_version, image_name=""):
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


def find_image_by_image_id(images, image_id):
    """从 images 列表中查找指定 image_id 的镜像记录"""
    for img in images:
        if img.get("image_id") == image_id:
            return img
    return None


def ensure_keepalive_images():
    """确保保活名单中的镜像都已导入（只加不删）。

    逻辑：
      1. GET /admin/images 获取已导入 image_id 集合
      2. 对名单里每个条目：
         - 已导入 → 跳过（不动）
         - 未导入 → 用条目自带的 agent_type/agent_version/image_name 调 import
      3. 名单中的镜像**只加不删**

    幂等：多次调用安全，已导入的不会重复导入。
    """
    if not KEEPALIVE_IMAGES:
        print(YELLOW("    保活名单为空，跳过"))
        return

    print(BOLD(f">>> 保活名单：{len(KEEPALIVE_IMAGES)} 个镜像"))

    # 已导入镜像集合
    existing = _get_admin_images()
    existing_ids = {img.get("image_id") for img in existing}
    print(f"    当前已导入镜像数: {len(existing_ids)}")

    added = 0
    skipped = 0
    for item in KEEPALIVE_IMAGES:
        image_id = item["image_id"]
        agent_type = item["agent_type"]
        agent_version = item["agent_version"]
        image_name = item.get("image_name", "")

        if image_id in existing_ids:
            print(GREEN(f"    [keepalive] {image_id} 已导入，跳过"))
            skipped += 1
            continue

        print(f"    [keepalive] {image_id} 未导入，正在导入 "
              f"(type={agent_type}, version={agent_version}) ...")
        resp = _import_image(image_id, agent_type, agent_version, image_name)
        if resp.status_code == 200:
            print(GREEN(f"    [keepalive] {image_id} 导入成功 ✓"))
            added += 1
        else:
            body = resp.json() if resp.content else {}
            # 409 = 已存在（可能并发导入），视为已导入
            if resp.status_code == 409:
                print(YELLOW(f"    [keepalive] {image_id} 已存在（409），跳过"))
                skipped += 1
            else:
                print(RED(
                    f"    [keepalive] {image_id} 导入失败 "
                    f"({resp.status_code}): {body.get('error', resp.text[:200])}"
                ))

    print(GREEN(f"    保活名单处理完成：新增 {added}，跳过 {skipped}，总计 "
                f"{len(KEEPALIVE_IMAGES)} 个（只加不删）✓"))

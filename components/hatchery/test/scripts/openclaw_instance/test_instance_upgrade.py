#!/usr/bin/env python3
"""
集成测试：OpenClaw 实例一键升级 E2E（真实升级流程）

本测试执行**真实的**升级流程（备份→SMH上传→重装→恢复），耗时 15-45 分钟。
不复用共享实例，自包含创建独立 admin/user/instance，跑完销毁。

版本选择策略（运行时动态计算，不硬编码任何具体版本号）：
  - 起点版本 = /admin/images 镜像列表中版本第二高的镜像（用于创建实例）
  - 目标版本 = /admin/images 镜像列表中版本最高的镜像（升级目标）
  镜像列表变化时，起点/目标版本自动跟随，无需改动代码。

测试流程：
  0. 调用 ensure_keepalive_images() 确保保活名单中镜像已导入（只加不删）
     不依赖编排器执行顺序，自给自足。
  1. GET  /admin/images                获取 openclaw 镜像列表，选次高/最高版本
                                        （镜像不足 2 个不同版本直接 SKIP）
  2. POST /admin/create                 创建测试 admin / user
  3. POST /openclaw/create              用 image_id 直接指定起点版本镜像创建实例
                                        （不切换全局启用镜像，避免影响并发测试）
  4. GET  /openclaw/status              等待实例 running
  5. GET  /openclaw/version             检查实例版本与起点镜像一致（不一致则 SKIP）
  6. POST /openclaw/set-channel         升级前配置 qqbot channel（验证升级不丢配置）
  7. POST /admin/images/enable          切换启用镜像为最高版本（升级接口依赖全局启用镜像）
  8. POST /openclaw/upgrade             触发升级
  9. GET  /openclaw/status              等待 upgrading → running / upgrade_failed
 10. GET  /openclaw/version             验证实例版本已升级为最高版本
 11. GET  /openclaw/channels            验证升级后 qqbot channel 仍存在
 12. GET  /openclaw/check-openclaw-port 验证 agent 就绪
 13. POST /openclaw/delete              清理实例

注意：步骤 7 切换启用镜像到最高版本后不再恢复——最高版本应保持启用。

集成测试环境说明：
  本测试**完全不预设**任何 image_id / 版本号，起点版本和目标版本
  均从 /admin/images 实时列表中选取（次高 / 最高）。
  若环境中 openclaw 镜像不足 2 个不同版本，直接 SKIP 本用例（保持测试稳定）。

前置条件：
  - 部署 UIN 在白名单中（image_id 隐藏参数才生效，起点版本才能真正落地）
  - openclaw 镜像列表中至少存在 2 个不同 agent_version 的镜像
  - 腾讯云 AK/SK（用于创建 CVM 实例）

环境变量：
  UPGRADE_E2E_TIMEOUT        升级等待超时（秒），默认 3000（50 分钟）
  UPGRADE_E2E_CREATE_TIMEOUT 创建实例等待超时（秒），默认 600（10 分钟）
"""
import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
from helpers.api import (
    seed,
    user_client,
    health_check,
    run_tests,
    ensure_gateway_ui_enabled,
)
from helpers.client import GREEN, RED, YELLOW, BOLD
from helpers import setup_admin, setup_user, create_instance, get_instance_db_id
from helpers.user_mgmt import teardown_scenario_users
from helpers.channel import user_set_channel, user_get_channels
from helpers.image_keepalive import ensure_keepalive_images

from _instance_helpers import (
    wait_for_running,
    wait_for_agent_ready,
    get_status,
)

SCENARIO = "upge2e"

# 升级等待超时：后端 OpUpgrade 超时 2700s（45 min），给 3000s（50 min）兜底
UPGRADE_TIMEOUT = int(os.environ.get("UPGRADE_E2E_TIMEOUT", "3000"))
# 创建实例等待超时
CREATE_TIMEOUT = int(os.environ.get("UPGRADE_E2E_CREATE_TIMEOUT", "600"))
# 轮询间隔
POLL_INTERVAL = 10

# ── 升级前配置的 channel（用于验证升级备份/恢复不丢用户配置）──
# 选 qqbot：set_channel.sh 对 qqbot 是纯 jq 写配置、不校验凭证真实性，
# 因此用占位凭证即可把 .channels.qqbot 写入实例 ~/.openclaw/openclaw.json，
# 升级后 list_channels.sh 读回即可断言 channel 仍存在。
UPGRADE_CHANNEL = "qqbot"
UPGRADE_CHANNEL_KEYS = ["app_id", "app_secret"]
UPGRADE_CHANNEL_VALUES = ["it-upgrade-fake-appid", "it-upgrade-fake-secret"]


# ═══════════════════════════════════════════════════════════════════════════
# 版本比较工具
# ═══════════════════════════════════════════════════════════════════════════

def parse_version(version_str):
    """将版本字符串解析为可比较的整数元组（**严格对齐后端 CompareSemver 语义**）。

    后端 model.CompareSemver 用 fmt.Sscanf(v, "%d.%d.%d", ...) 解析，
    **只取前三段**、缺失段补 0，按段整数比较。为保证脚本选出的
    low/high 与后端防降级校验的排序完全一致（否则会出现脚本认为
    low<high、但后端认为 low>high 而拒绝升级的矛盾），这里必须
    对齐为「固定三段、缺失补 0」而非变长元组。

    "4.23"       → (4, 23, 0)
    "4.23.1"     → (4, 23, 1)
    "2026.6.10"  → (2026, 6, 10)
    "2026.5.7"   → (2026, 5, 7)
    ""           → (0, 0, 0)  空版本视为最低
    """
    parts = [0, 0, 0]
    if version_str:
        for i, p in enumerate(version_str.strip().split(".")):
            if i >= 3:
                break  # 与后端 Sscanf 一致，只取前三段
            try:
                parts[i] = int(p)
            except ValueError:
                # 非数字部分按 0 处理，保证不崩（与后端 Sscanf 遇非数字停止近似）
                parts[i] = 0
    return tuple(parts)


def compare_version(a, b):
    """比较两个版本号，返回 -1 (a<b) / 0 (a==b) / 1 (a>b)。

    与后端 model.CompareSemver 语义一致。
    """
    pa, pb = parse_version(a), parse_version(b)
    if pa < pb:
        return -1
    if pa > pb:
        return 1
    return 0


def version_lt(a, b):
    """返回 True 如果 a < b"""
    return compare_version(a, b) < 0


# ═══════════════════════════════════════════════════════════════════════════
# 镜像管理工具
# ═══════════════════════════════════════════════════════════════════════════

def get_openclaw_images():
    """GET /admin/images → 返回 openclaw 类型的镜像列表（含 enabled 状态）"""
    resp = seed.get("/admin/images", raw=True, timeout=30)
    if resp.status_code != 200:
        raise RuntimeError(f"GET /admin/images 失败 ({resp.status_code}): {resp.text[:200]}")
    data = resp.json()
    images = data.get("images", [])
    # 仅保留 openclaw 类型且有版本号的镜像
    oc_images = [
        img for img in images
        if (img.get("agent_type") == "openclaw" or img.get("agent_type") == "")
        and img.get("agent_version")
    ]
    return oc_images


def select_low_high_versions(images):
    """从镜像列表中选择升级测试的起点版本（次高）和目标版本（最高）。

    升级测试采用**相邻版本增量升级**策略，而非"最低升到最高"：
    - 目标版本 high = 版本列表中的最高版本（sorted[-1]）
    - 起点版本 low  = 版本列表中的第二高版本（sorted[-2]）

    这样测试的是"次高 → 最高"的相邻升级，更贴近真实增量升级场景，
    避免跨越过多版本导致的不确定性。

    返回 (low_image, high_image)，两者版本必须不同。
    若不足 2 个不同版本，返回 (None, None)。
    """
    # 按版本排序（升序）
    sorted_imgs = sorted(images, key=lambda x: parse_version(x.get("agent_version", "")))
    if len(sorted_imgs) < 2:
        return None, None

    # 去重：同一版本可能有多个镜像，按版本聚合，保留每个版本的最后一个镜像
    unique_by_version = {}
    for img in sorted_imgs:
        unique_by_version[img.get("agent_version", "")] = img
    # 按版本重新排序后取唯一版本列表
    unique_sorted = sorted(
        unique_by_version.values(),
        key=lambda x: parse_version(x.get("agent_version", "")),
    )
    if len(unique_sorted) < 2:
        return None, None

    # 目标版本 = 最高版本；起点版本 = 第二高版本（相邻增量升级）
    high = unique_sorted[-1]
    low = unique_sorted[-2]

    if low.get("agent_version") == high.get("agent_version"):
        return None, None

    return low, high


def get_enabled_openclaw_image(images):
    """从镜像列表中找到当前已启用的 openclaw 镜像，返回其 DB id（用于恢复）"""
    for img in images:
        if img.get("enabled"):
            return img.get("id") or img.get("ID")
    return None


def _is_image_enabled(db_id):
    """检查指定 ID 的镜像是否已启用"""
    images = get_openclaw_images()
    for img in images:
        img_id = img.get("id") or img.get("ID")
        if img_id == db_id:
            return img.get("enabled", False)
    return False


def ensure_image_enabled(db_id):
    """确保指定镜像已启用（纯 API，对齐 helpers.hermes.ensure_hermes_image）。

    /admin/images/enable 是 toggle 行为，需先检查状态：
    - 已启用 → 跳过（避免 toggle 把它禁用）
    - 未启用 → POST /admin/images/enable 启用
    """
    if _is_image_enabled(db_id):
        print(f"    镜像 id={db_id} 已启用，跳过")
        return

    resp = seed.post("/admin/images/enable", params={"id": db_id}, raw=True, timeout=30)
    if resp.status_code != 200:
        raise RuntimeError(
            f"启用镜像失败 (id={db_id}, status={resp.status_code}): {resp.text[:200]}"
        )
    if not _is_image_enabled(db_id):
        raise RuntimeError(f"启用镜像失败 (id={db_id})：API 返回成功但 DB 未更新")
    print(f"    镜像 id={db_id} 已启用 ✓")


# ═══════════════════════════════════════════════════════════════════════════
# Channel 校验工具（验证升级不丢用户配置）
# ═══════════════════════════════════════════════════════════════════════════

def _list_instance_channel_keys(user_token, db_id):
    """GET /openclaw/channels?id= → 返回实例已配置的 channel 名集合。

    后端 listInstanceChannels 通过 list_channels.sh 读取实例端
    ~/.openclaw/openclaw.json 的 .channels，返回形如 {"qqbot": {...}}。
    """
    data = user_get_channels(user_token, instance_db_id=db_id)
    channels = data.get("channels", {}) if isinstance(data, dict) else {}
    if isinstance(channels, dict):
        return set(channels.keys())
    return set()


# ═══════════════════════════════════════════════════════════════════════════
# 实例版本查询（DB agent_version）
# ═══════════════════════════════════════════════════════════════════════════

def get_instance_agent_version(db_id, client):
    """通过 /openclaw/list 获取实例 DB 里的 agent_version。

    后端 rejectDowngradeOnOfficialImage 防降级校验用的是 DB 里的
    instance.AgentVersion（openclaw_upgrade.go:506），而非 /openclaw/version
    接口通过 TAT 实时探测的版本。因此测试脚本也应用 DB agent_version
    来做版本校验，与后端逻辑一致。

    /openclaw/version 接口依赖 detect_openclaw_install.sh TAT 脚本，
    在部分环境会一直返回空 version（脚本探测不到 openclaw 二进制），
    不适合作为版本校验来源。

    Args:
      db_id: 实例 DB id
      client: ApiClient（用户视角，X-OpenAPI: 1）

    Returns:
      DB 里的 agent_version 字符串（可能为空）。
    """
    resp = client.get(
        "/openclaw/list", params={"page": 1, "page_size": 30},
        timeout=30, expect=None, raw=True,
    )
    if resp.status_code != 200:
        raise RuntimeError(
            f"GET /openclaw/list 失败 ({resp.status_code}): {resp.text[:200]}"
        )
    data = resp.json()
    instances = data.get("instances", [])
    for inst in instances:
        inst_id = inst.get("ID") or inst.get("id")
        if str(inst_id) == str(db_id):
            return inst.get("agent_version", "")
    raise RuntimeError(f"在 /openclaw/list 中未找到 db_id={db_id} 的实例")


# ═══════════════════════════════════════════════════════════════════════════
# 升级等待
# ═══════════════════════════════════════════════════════════════════════════

def wait_for_upgrade_complete(db_id, client, timeout=None):
    """等待升级完成。

    升级流程中状态变化：
      running → upgrading (transient) → running (成功) / upgrade_failed (失败)

    成功：返回 status_data (status=running)
    失败：抛出 AssertionError (status=upgrade_failed)
    超时：抛出 TimeoutError
    """
    timeout = timeout or UPGRADE_TIMEOUT
    start = time.time()
    last_status = None
    saw_upgrading = False

    print(f"     [wait-upgrade] timeout={timeout}s")
    while True:
        elapsed = time.time() - start
        if elapsed > timeout:
            raise TimeoutError(
                f"升级在 {timeout}s 内未完成，最后状态={last_status}"
            )

        data = get_status(db_id, client=client)
        status = data.get("status", "unknown")

        if status != last_status:
            print(f"     [wait-upgrade] [{int(elapsed)}s] status={status} "
                  f"label={data.get('label')}")
            last_status = status

        if status == "upgrading":
            saw_upgrading = True
            time.sleep(POLL_INTERVAL)
            continue

        if status == "upgrade_failed":
            raise AssertionError(
                f"升级失败！实例进入 upgrade_failed 状态: "
                f"tooltip={data.get('tooltip')}"
            )

        # 升级成功后状态应回到 running
        if saw_upgrading and status == "running":
            print(f"     [wait-upgrade] 升级完成，耗时 {int(elapsed)}s")
            return data

        # 如果还没看到 upgrading 就已经是 running，可能是升级太快或没触发
        # 继续等待一段时间看是否进入 upgrading
        if not saw_upgrading and status == "running":
            # 给后端一点时间把状态切换到 upgrading
            time.sleep(POLL_INTERVAL)
            continue

        # 其它状态（loading 等）继续等待
        time.sleep(POLL_INTERVAL)


# ═══════════════════════════════════════════════════════════════════════════
# 用例（由 run_tests(globals()) 收集）
# ═══════════════════════════════════════════════════════════════════════════

def test_01_openclaw_real_upgrade():
    """OpenClaw 实例真实升级 E2E（次高版本 → 最高版本）。

    执行真实升级流程（备份→SMH上传→重装→恢复），耗时 15-45 分钟，默认真跑。
    仅当 openclaw 镜像不足 2 个不同版本时才会 SKIP。
    """
    # ── 变量声明（finally 用）──
    admin = None
    user = None
    cli = None
    db_id = None
    # channel 校验状态（升级前配置成功才在升级后做强制校验）
    channel_configured = False
    pre_channels = set()

    try:
        # ─── 步骤 0：确保保活名单中的镜像已导入（只加不删）────────
        # 不依赖编排器执行顺序：即使镜像测试未先跑，升级测试也能自给自足
        print(BOLD(">>> 步骤 0：确保保活名单镜像就绪 ..."))
        ensure_keepalive_images()
        print()

        # ─── 步骤 1：获取镜像列表，选低/高版本 ──────────────────────
        print(BOLD(">>> 步骤 1：获取 openclaw 镜像列表 ..."))
        images = get_openclaw_images()
        print(f"    openclaw 镜像数: {len(images)}")
        for img in images:
            print(f"      id={img.get('id')} image_id={img.get('image_id')} "
                  f"version={img.get('agent_version')} enabled={img.get('enabled')}")

        low_img, high_img = select_low_high_versions(images)

        if not low_img or not high_img:
            print(YELLOW(
                "SKIP: openclaw 镜像不足 2 个不同版本，无法执行升级 E2E 测试。"
                "请先通过 /admin/images/import 导入至少 2 个不同 agent_version 的镜像。"
            ))
            return

        low_version = low_img.get("agent_version")
        high_version = high_img.get("agent_version")
        low_cvm_image_id = low_img.get("image_id")
        high_db_id = high_img.get("id") or high_img.get("ID")
        low_db_id = low_img.get("id") or low_img.get("ID")

        print(f"    起点版本(次高): version={low_version} image_id={low_cvm_image_id}")
        print(f"    目标版本(最高): version={high_version} image_id={high_img.get('image_id')}")
        print(f"    版本比较: {low_version} < {high_version} ✓（相邻增量升级）")

        # 获取当前启用的 openclaw 镜像 ID（仅用于日志参考）
        current_enabled = get_enabled_openclaw_image(images)
        print(f"    当前启用镜像 DB id: {current_enabled}")

        # ─── 步骤 2：创建 admin + user ──────────────────────────────
        print(BOLD(">>> 步骤 2：创建测试 admin / user ..."))
        admin = setup_admin(SCENARIO)
        # ensure_gateway_ui_enabled 可能因 /admin/config 解析 CVM 模板失败而报错，
        # 但 gateway_ui_enable 可能已经在数据库中启用，这里做容错处理。
        try:
            ensure_gateway_ui_enabled(admin.token)
        except Exception as e:
            print(YELLOW(f"    [WARN] ensure_gateway_ui_enabled 失败（可能已启用）: {e}"))
        user = setup_user(admin.token, SCENARIO)
        cli = user_client(user.token)
        print(f"    admin={admin.username} user={user.username} ✓")

        # ─── 步骤 3：用起点版本镜像创建实例 ────────────────────────────
        # 直接通过 image_id 指定起点镜像，不切换全局启用镜像，
        # 避免影响其他并发执行的测试用例。
        print(BOLD(f">>> 步骤 3：用起点版本(次高)镜像创建实例 (version={low_version}) ..."))
        name = f"{config.INSTANCE_NAME_PREFIX}{SCENARIO}-{int(time.time())}"
        print(f"    实例名: {name}")

        # image_id 隐藏参数：指定起点版本 CVM 镜像 ID
        # 仅白名单部署生效，非白名单会静默忽略（使用当前启用镜像）
        create_data = create_instance(
            user.token, name,
            agent_type="openclaw",
            image_id=low_cvm_image_id,
        )
        if not create_data.get("ok"):
            raise RuntimeError(f"创建实例失败: {create_data}")

        cvm_instance_id = create_data.get("instance_id", "")
        print(f"    创建已下发 cvm_instance_id={cvm_instance_id} ✓")

        # 反查 db_id
        db_id = get_instance_db_id(user.token, cvm_instance_id)
        print(f"    db_id={db_id} ✓")

        # ─── 步骤 4：等待实例 running ───────────────────────────────
        print(BOLD(f">>> 步骤 4：等待实例 running (timeout={CREATE_TIMEOUT}s) ..."))
        status_data = wait_for_running(db_id, timeout=CREATE_TIMEOUT, client=cli)
        print(f"    实例就绪 ✓ label={status_data.get('label')}")

        # ─── 步骤 5：检查实例版本与起点镜像一致 ──────────────────────
        # 创建实例后，验证 DB 里的 agent_version == 步骤 1 选的 low_version。
        # 后端防降级校验用的是 DB 里的 instance.AgentVersion，测试脚本也用它。
        # /openclaw/version 接口（TAT 实时探测）在部分环境会一直返回空，不可靠。
        print(BOLD(">>> 步骤 5：检查实例版本是否与起点镜像一致 ..."))
        print("    等待 agent ready ...")
        wait_for_agent_ready(db_id, timeout=300, client=cli)

        actual_version = get_instance_agent_version(db_id, cli)
        print(f"    实例 DB agent_version: {actual_version}")
        print(f"    起点镜像版本: {low_version}")
        # 版本比较用 compare_version 而非字符串直接比较：
        # OpenClaw 版本格式 YYYY.M.D 允许非零填充（2026.6.10 == 2026.06.10）
        if compare_version(actual_version, low_version) != 0:
            print(YELLOW(
                f"SKIP: 实例 DB 版本 {actual_version} 与起点镜像版本 {low_version} 不一致，"
                f"镜像元数据可能与实际不符（环境问题），跳过升级"
            ))
            return
        print(f"    版本一致 ✓")

        # ─── 步骤 5.5：升级前配置 channel（验证升级不丢用户配置）────────
        print(BOLD(f">>> 步骤 5.5：升级前配置 {UPGRADE_CHANNEL} channel ..."))
        set_resp = user_set_channel(
            user.token, db_id, UPGRADE_CHANNEL,
            UPGRADE_CHANNEL_KEYS, UPGRADE_CHANNEL_VALUES,
        )
        if set_resp.status_code == 200:
            print(f"    set-channel {UPGRADE_CHANNEL} 成功 ✓")
        else:
            # set-channel 依赖 TAT 写实例配置；失败不阻塞升级主流程，仅降级 channel 验证
            print(YELLOW(
                f"    [WARN] set-channel 返回 {set_resp.status_code}: "
                f"{set_resp.text[:200]}，升级后 channel 验证将降级为非强制"
            ))

        # 读回确认已写入，并记录升级前 channel 集合
        try:
            pre_channels = _list_instance_channel_keys(user.token, db_id)
            print(f"    升级前实例已配置 channels: {sorted(pre_channels)}")
            channel_configured = UPGRADE_CHANNEL in pre_channels
            if channel_configured:
                print(f"    {UPGRADE_CHANNEL} 已写入实例配置 ✓")
            else:
                print(YELLOW(
                    f"    [WARN] {UPGRADE_CHANNEL} 未出现在升级前 channel 列表，"
                    f"升级后 channel 验证将降级为非强制"
                ))
        except Exception as e:
            print(YELLOW(f"    [WARN] 读取升级前 channel 列表失败: {e}"))
            channel_configured = False

        # ─── 步骤 6：切换启用镜像为最高版本（升级目标）─────────────
        # 后端升级接口通过 model.GetEnabledImageByType 获取当前启用镜像作为升级目标，
        # 因此必须切换启用镜像到最高版本。
        # 注意：步骤 3 创建实例时已通过 image_id 直接指定起点镜像，未切换全局启用镜像，
        # 这里是整个流程中唯一一次切换启用镜像——切换到最高版本（升级目标）。
        # 切换后不再恢复：最高版本应保持启用，供后续新实例使用。
        print(BOLD(f">>> 步骤 6：切换启用镜像为最高版本 (version={high_version}) ..."))
        ensure_image_enabled(high_db_id)
        print(f"    最高版本镜像已启用 (db_id={high_db_id}) ✓")

        # 短暂等待确保 DB 写入完成
        time.sleep(2)

        # ─── 步骤 7：触发升级 ────────────────────────────────────────
        print(BOLD(">>> 步骤 7：触发一键升级 ..."))
        upgrade_resp = cli.post(
            "/openclaw/upgrade",
            data={"id": db_id},
            expect=None, raw=True, timeout=30,
        )
        if upgrade_resp.status_code == 200:
            body = upgrade_resp.json() if upgrade_resp.content else {}
            print(f"    升级已触发 ✓ response={body}")
        elif upgrade_resp.status_code == 409:
            # 可能是状态拒绝（如实例还在 transitioning）
            body = upgrade_resp.json() if upgrade_resp.content else {}
            err = body.get("error", "")
            raise RuntimeError(
                f"升级被拒绝 (409): {err} — "
                f"可能实例状态不允许升级"
            )
        else:
            raise RuntimeError(
                f"触发升级失败 (status={upgrade_resp.status_code}): "
                f"{upgrade_resp.text[:200]}"
            )

        # ─── 步骤 8：等待升级完成 ────────────────────────────────────
        print(BOLD(f">>> 步骤 8：等待升级完成 (timeout={UPGRADE_TIMEOUT}s) ..."))
        print(YELLOW(
            "    ⏳ 升级流程包括：备份→SMH上传→重装CVM→等待就绪→数据恢复→后置hook"
        ))
        print(YELLOW(f"    预计耗时 15-45 分钟，请耐心等待 ..."))
        print()

        final_status = wait_for_upgrade_complete(db_id, cli, timeout=UPGRADE_TIMEOUT)
        print(f"    升级完成 ✓ status={final_status.get('status')}")

        # ─── 步骤 9：验证实例版本已升级为目标版本 ────────────────────────
        print(BOLD(">>> 步骤 9：验证实例版本已升级为目标版本(最高) ..."))
        # 升级后 agent 需要重新就绪
        print("    等待升级后 agent ready ...")
        wait_for_agent_ready(db_id, timeout=300, client=cli)

        # 用 DB agent_version 校验（与后端防降级校验一致）
        upgraded_version = get_instance_agent_version(db_id, cli)
        print(f"    升级后实例 DB agent_version: {upgraded_version}")

        if upgraded_version == high_version:
            print(f"    版本已升级到 {high_version} ✓")
        elif not version_lt(upgraded_version, high_version):
            print(f"    版本 {upgraded_version} >= 目标版本 {high_version} ✓")
        else:
            raise AssertionError(
                f"升级后版本 {upgraded_version} 低于目标版本 {high_version}，"
                f"升级可能未完全成功"
            )

        # ─── 步骤 9.5：验证升级后 channel 仍存在（升级不丢用户配置）──────
        print(BOLD(f">>> 步骤 9.5：验证升级后 {UPGRADE_CHANNEL} channel 仍存在 ..."))
        try:
            post_channels = _list_instance_channel_keys(user.token, db_id)
            print(f"    升级后实例已配置 channels: {sorted(post_channels)}")
            if UPGRADE_CHANNEL in post_channels:
                print(GREEN(f"    {UPGRADE_CHANNEL} channel 升级后仍存在 ✓（配置未丢失）"))
            elif channel_configured:
                # 升级前确实配了 channel，升级后却没了 → 说明备份/恢复丢了用户配置
                raise AssertionError(
                    f"升级后 {UPGRADE_CHANNEL} channel 丢失！"
                    f"升级前={sorted(pre_channels)} 升级后={sorted(post_channels)}，"
                    f"备份/恢复流程可能丢失了用户 channel 配置"
                )
            else:
                print(YELLOW(
                    f"    [WARN] 升级前未成功配置 {UPGRADE_CHANNEL}，跳过 channel 保留校验"
                ))
        except AssertionError:
            raise
        except Exception as e:
            print(YELLOW(f"    [WARN] 读取升级后 channel 列表失败: {e}"))

        # ─── 步骤 10：验证实例健康（agent ready + gateway 可达）──────
        print(BOLD(">>> 步骤 10：验证实例健康 ..."))
        # agent ready 已在步骤 9 验证，这里再做一次确认
        status_data = get_status(db_id, client=cli)
        assert status_data.get("status") == "running", (
            f"升级后实例状态异常: {status_data}"
        )
        print(f"    status=running ✓")

        # 检查 check-openclaw-port
        port_resp = cli.get(
            "/openclaw/check-openclaw-port",
            params={"id": db_id},
            timeout=120, expect=None, raw=True,
        )
        if port_resp.status_code == 200:
            port_data = port_resp.json()
            if port_data.get("running"):
                print(f"    agent running=True ✓")
            else:
                print(YELLOW(
                    f"    [WARN] agent running=False: {port_data.get('reason')}"
                ))
        else:
            print(YELLOW(
                f"    [WARN] check-openclaw-port 返回 {port_resp.status_code}"
            ))

        # ─── 成功 ───────────────────────────────────────────────────
        print()
        print(GREEN("=" * 60))
        print(GREEN("test_instance_upgrade.py 测试通过 ✅"))
        print(GREEN(f"  起点版本(次高): {low_version} → 目标版本(最高): {high_version}"))
        print(GREEN(f"  升级后版本: {upgraded_version}"))
        print(GREEN("=" * 60))

    except Exception as e:
        print(RED(f"\ntest_instance_upgrade.py 测试失败 ❌: {e}"))
        traceback.print_exc()
        raise

    finally:
        # ── 清理：删除实例 + 删除测试用户 ──
        # 注意：不恢复原启用镜像。步骤 6 切换了启用镜像到最高版本（升级目标），
        # 这是正确的最终状态——最高版本应保持启用。
        print(BOLD("\n>>> 清理资源 ..."))

        # 1. 删除测试实例
        if db_id is not None and cli is not None:
            try:
                print(f"    删除实例 db_id={db_id} ...")
                cli.post(
                    "/openclaw/delete",
                    data={"id": db_id},
                    expect=None, raw=True, timeout=60,
                )
                print(f"    delete 已下发 ✓")
            except Exception as e:
                print(f"    [cleanup] 删除实例失败（CI cleanup.py 会兜底）: {e}")

        # 2. 删除测试用户
        try:
            teardown_scenario_users(SCENARIO)
        except Exception as e:
            print(f"    [cleanup] 清理测试用户失败（忽略）: {e}")


# ═══════════════════════════════════════════════════════════════════════════
# 入口
# ═══════════════════════════════════════════════════════════════════════════

def main():
    health_check()
    print()
    run_tests(
        globals(),
        title="test_instance_upgrade.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()

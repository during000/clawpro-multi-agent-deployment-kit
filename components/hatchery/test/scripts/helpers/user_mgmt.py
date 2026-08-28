"""
用户管理（管理员接口）+ 用户/管理员 setup 脚手架
"""

from dataclasses import dataclass

from helpers import config
from helpers.api import admin_client


# ═══════════════════════════════════════════════════════════════════
# 上下文数据类
# ═══════════════════════════════════════════════════════════════════

@dataclass
class AdminContext:
    """管理员测试用户上下文"""
    user_id: int
    token: str
    username: str


@dataclass
class UserContext:
    """普通测试用户上下文"""
    user_id: int
    token: str
    username: str


def admin_create_user(
    admin_token,
    username,
    password=None,
    role="user",
    instance_quota=None,
    token_quota_day=None,
    group_ids=None,
):
    """创建用户，返回 {"ok": true, "id": <user_id>}"""
    if password is None:
        password = config.generate_password()
    if instance_quota is None:
        instance_quota = config.DEFAULT_INSTANCE_QUOTA
    if token_quota_day is None:
        token_quota_day = config.DEFAULT_TOKEN_QUOTA_DAY

    client = admin_client(admin_token)
    payload = {
        "username": username,
        "password": password,
        "role": role,
        "instance_quota": str(instance_quota),
        "token_quota_day": str(token_quota_day),
    }

    if group_ids:
        payload["group_ids"] = group_ids
        return client.post("/admin/create", json=payload, timeout=30)
    else:
        return client.post("/admin/create", data=payload, timeout=30)


def admin_get_user_token(admin_token, user_id):
    """获取用户 Token，返回 token 字符串"""
    data = admin_client(admin_token).get("/admin/user-token", params={"id": user_id})
    assert data.get("exists"), f"Token 不存在: user_id={user_id}, data={data}"
    return data["token"]


def admin_enable_token(admin_token, user_id):
    """启用用户 Token（幂等：若已启用则忽略 400 错误）"""
    resp = admin_client(admin_token).post(
        "/admin/token/enable", params={"id": user_id},
        expect=None, raw=True,
    )
    if resp.status_code == 400:
        # 服务端对已启用的 Token 返回 400 "该用户 Token 未被禁用"，属于正常幂等情况
        return {"ok": True, "skipped": True}
    if resp.status_code != 200:
        raise AssertionError(
            f"启用 Token 失败: status={resp.status_code}, body={resp.text[:300]}"
        )
    return resp.json()



# ═══════════════════════════════════════════════════════════════════
# 脚手架：管理员 / 普通用户 setup
# ═══════════════════════════════════════════════════════════════════

def setup_admin(scenario):
    """创建管理员测试用户 + 获取 Token（使用种子 Token 创建）"""
    username = f"{config.ADMIN_USERNAME_PREFIX}{scenario}"
    print(f">>> 创建管理员测试用户: {username} ...")

    data = admin_create_user(
        config.SEED_ADMIN_TOKEN,
        username=username,
        role="admin",
        instance_quota=0,
    )
    assert data.get("ok"), f"创建管理员测试用户失败: {data}"
    admin_id = data["id"]

    admin_enable_token(config.SEED_ADMIN_TOKEN, admin_id)
    token = admin_get_user_token(config.SEED_ADMIN_TOKEN, admin_id)
    print(f"    管理员用户创建成功 ✓  id={admin_id}")
    return AdminContext(user_id=admin_id, token=token, username=username)


def teardown_scenario_users(scenario):
    """清理某场景由 setup_admin / setup_user 创建的测试用户（幂等、忽略错误）。

    按用户名前缀（管理员 + 普通用户）用种子管理员 Token 硬删，避免再次执行时
    出现 "用户名已存在"(409)。适合在测试开头预清理 + 结尾 finally 收尾各调用一次。
    """
    from helpers.api import cleanup_users_by_prefix

    for prefix in (
        f"{config.ADMIN_USERNAME_PREFIX}{scenario}",
        f"{config.USERNAME_PREFIX}{scenario}",
    ):
        try:
            cleanup_users_by_prefix(prefix, verbose=False)
        except Exception as e:  # 清理失败不应影响测试结论
            print(f"    清理测试用户 '{prefix}' 时出错（忽略）: {e}")


def setup_user(admin_token, scenario, instance_quota=None, group_ids=None):
    """创建普通测试用户 + 获取 Token"""
    if instance_quota is None:
        instance_quota = config.DEFAULT_INSTANCE_QUOTA
    username = f"{config.USERNAME_PREFIX}{scenario}"
    print(f">>> 创建测试用户: {username} ...")

    data = admin_create_user(
        admin_token,
        username=username,
        instance_quota=instance_quota,
        group_ids=group_ids,
    )
    assert data.get("ok"), f"创建用户失败: {data}"
    user_id = data["id"]

    admin_enable_token(admin_token, user_id)
    token = admin_get_user_token(admin_token, user_id)
    print(f"    用户创建成功 ✓  id={user_id}")
    return UserContext(user_id=user_id, token=token, username=username)

#!/usr/bin/env python3
"""
用户组管理专用辅助函数。

提供 extract_group_id / pick_group / list_all_groups / find_groups_by_prefix /
cleanup_groups_by_prefix / cleanup_by_prefix 等工具，供 admin_user_groups 测试使用。
"""
from helpers.api import (
    seed,
    GREEN, RED, YELLOW,
    list_users_by_prefix, cleanup_users_by_prefix,
)


def extract_group_id(resp_json):
    """从创建/更新用户组响应中提取 group id，兼容多种字段格式"""
    if not isinstance(resp_json, dict):
        return None
    for key in ("id", "ID", "group_id", "GroupID"):
        if key in resp_json and resp_json[key]:
            return resp_json[key]
    g = resp_json.get("group") or resp_json.get("Group") or {}
    if isinstance(g, dict):
        return g.get("id") or g.get("ID")
    return None


def pick_group(groups, *, name=None, gid=None):
    """从分组列表中按 name 或 id 找记录，兼容大小写字段名"""
    for g in groups or []:
        if name is not None:
            n = g.get("name") or g.get("Name")
            if n == name:
                return g
        if gid is not None:
            i = g.get("id") or g.get("ID")
            if i == gid:
                return g
    return None


def list_all_groups(page_size=100):
    """分页拉取全部用户组（用于按名称匹配/清理）"""
    page = 1
    out = []
    while True:
        resp = seed.get("/admin/user-groups",
                        params={"page": page, "page_size": page_size},
                        expect=None, raw=True)
        if resp.status_code != 200:
            break
        data = resp.json() or {}
        groups = data.get("groups") or data.get("Groups") or []
        if not groups:
            break
        out.extend(groups)
        if len(groups) < page_size:
            break
        page += 1
    return out


def find_groups_by_prefix(prefix):
    """按前缀过滤用户组"""
    return [g for g in list_all_groups()
            if (g.get("name") or g.get("Name") or "").startswith(prefix)]


def cleanup_groups_by_prefix(prefix, *, verbose=True):
    """按 name 前缀删除用户组。返回 (尝试数, 成功数)。"""
    groups = find_groups_by_prefix(prefix)
    tried, succ = 0, 0
    for g in groups:
        gid = g.get("id") or g.get("ID")
        if not gid:
            continue
        tried += 1
        r = seed.post("/admin/user-groups/delete",
                      json={"id": gid}, expect=None, raw=True)
        if 200 <= r.status_code < 300:
            succ += 1
    if verbose and tried:
        msg = f"[cleanup] 按前缀 '{prefix}' 清理用户组: {succ}/{tried}"
        print(GREEN(msg) if succ == tried else YELLOW(msg))
    return tried, succ


def cleanup_by_prefix(*, group_prefix=None, user_prefix=None, verbose=True):
    """组合兜底清理：先清用户组（释放成员关联），再清用户。

    会做两轮，避免第一轮因成员被引用而失败。两轮过后若仍有残留，
    会以告警形式打印（不会抛异常），由测试用例自行决定是否 fail。
    """
    rounds = 2
    for i in range(rounds):
        if group_prefix:
            cleanup_groups_by_prefix(group_prefix, verbose=verbose and i == 0)
        if user_prefix:
            cleanup_users_by_prefix(user_prefix, verbose=verbose and i == 0)

    # 最终残留检查（仅打印，不抛异常）
    leftover_g = find_groups_by_prefix(group_prefix) if group_prefix else []
    leftover_u = list_users_by_prefix(user_prefix) if user_prefix else []
    if leftover_g or leftover_u:
        print(RED(
            f"[cleanup] 仍有残留: groups={len(leftover_g)} users={len(leftover_u)}"
        ))
        for g in leftover_g[:5]:
            print(RED(f"   - group id={g.get('id') or g.get('ID')} "
                      f"name={g.get('name') or g.get('Name')}"))
        for u in leftover_u[:5]:
            print(RED(f"   - user  id={u.get('id') or u.get('ID')} "
                      f"username={u.get('username') or u.get('Username')}"))
    elif verbose and (group_prefix or user_prefix):
        prefixes = ", ".join(p for p in [group_prefix, user_prefix] if p)
        print(GREEN(f"[cleanup] 清理完成，无残留 (prefixes: {prefixes})"))
    return len(leftover_g), len(leftover_u)

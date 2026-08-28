#!/usr/bin/env python3
"""
SkillHub 灰度代理集成测试

覆盖接口：
    GET /admin/skillhub-status    灰度状态查询 + URL 推导 + 认证
    GET /admin/skills             灰度关闭场景下的技能列表（分页/关键词/全量）

设计原则：
    - 灰度关闭场景（skill_hub_enabled=false）：验证装饰器路由到本地 handler
    - 灰度状态端点：验证返回结构 + skillhub_url 推导逻辑
    - 认证安全：验证 requireAdmin 拦截未认证请求
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    seed,
    run_tests, auth_test_suite, make_api_fn,
)

# 用 make_api_fn 创建 API 调用函数，自动支持 auth_test_suite 的 headers 参数
get_skillhub_status = make_api_fn("get", "/admin/skillhub-status")
get_admin_skills = make_api_fn("get", "/admin/skills")


# ─── 测试用例 ───

def test_01_skillhub_status_auth():
    """skillhub-status 认证三件套"""
    auth_test_suite(
        lambda headers: get_skillhub_status(headers=headers),
        label="skillhub_status",
    )


def test_02_skillhub_status_response():
    """skillhub-status 返回结构 + URL 推导"""
    resp = get_skillhub_status()
    assert resp.status_code == 200, f"status={resp.status_code} body={resp.text}"
    data = resp.json()

    # 验证返回字段
    assert "enabled" in data, f"missing 'enabled' field: {data}"
    assert isinstance(data["enabled"], bool), f"enabled should be bool: {data['enabled']}"

    # skillhub_url 推导验证：api.skillhub.cn → skillhub.cn（不应包含 api. 前缀）
    if "skillhub_url" in data and data["skillhub_url"]:
        url = data["skillhub_url"]
        # https://api.skillhub.cn → https://skillhub.cn
        host_part = url.split("://")[1] if "://" in url else url
        assert not host_part.startswith("api."), \
            f"skillhub_url should not contain 'api.' prefix: {url}"

    print(f"    enabled={data['enabled']}, skillhub_url={data.get('skillhub_url', 'N/A')}")


def test_03_skills_list_grayscale_off():
    """灰度关闭：技能列表走本地 handler（分页）"""
    resp = get_admin_skills(params={"page": 1, "page_size": 2})
    assert resp.status_code == 200, f"status={resp.status_code} body={resp.text}"
    data = resp.json()
    assert "skills" in data, f"missing 'skills' field: {data}"
    skills = data["skills"] or []
    assert len(skills) <= 2, f"page_size=2 but got {len(skills)} items"
    print(f"    page_size=2 → {len(skills)} skills, total={data.get('total', 'N/A')}")


def test_04_skills_list_keyword():
    """灰度关闭：技能列表关键词搜索"""
    resp = get_admin_skills(params={"page": 1, "page_size": 30, "keyword": "bus"})
    assert resp.status_code == 200, f"status={resp.status_code} body={resp.text}"
    data = resp.json()
    assert "skills" in data, f"missing 'skills' field: {data}"
    # 空列表时后端可能返回 null，容错处理
    skills = data["skills"] or []
    print(f"    keyword=bus → {len(skills)} skills matched")


def test_05_skills_list_full():
    """灰度关闭：技能列表全量查询"""
    resp = get_admin_skills(params={"page": 1, "page_size": 30})
    assert resp.status_code == 200, f"status={resp.status_code} body={resp.text}"
    data = resp.json()
    assert "skills" in data, f"missing 'skills' field: {data}"
    assert "total" in data, f"missing 'total' field: {data}"
    total = data["total"]
    skills = data["skills"] or []
    print(f"    full list → {len(skills)} skills, total={total}")


def test_06_skills_list_auth():
    """技能列表认证三件套"""
    auth_test_suite(
        lambda headers: get_admin_skills(headers=headers),
        label="admin_skills",
    )


# ─── 入口 ───

def main():
    run_tests(globals(), title="SkillHub 灰度代理集成测试", ordered=True, abort_on_fail=True)


if __name__ == "__main__":
    main()

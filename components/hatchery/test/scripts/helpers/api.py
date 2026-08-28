#!/usr/bin/env python3
"""
集成测试 HTTP 基础设施（统一入口）

本文件是 helpers.client（HTTP 客户端层）和 helpers.testing（测试辅助层）的
统一 re-export 门面。所有 `from helpers.api import ...` 的现有导入无需修改。

实际实现位于:
  - helpers/client.py  — ApiClient / 帧记录引擎 / Header 工具
  - helpers/testing.py — 断言 / 运行器 / 环境检查 / 用户工具 / Gateway 重试

用法:
    # OOP 风格 —— 所有测试脚本统一使用
    from helpers.api import ApiClient, admin_client, user_client
    from helpers.api import seed, anon, bad_token

    # 工具 + 断言 + 运行器
    from helpers.api import (
        API, ADMIN_TOKEN, HEADERS,
        health_check, run_tests,
        assert_status, assert_fields,
        make_api_fn,
        pick_user, extract_uid, cleanup_users_by_prefix,
    )

可选环境变量:
    SHOW_TOKEN=1   cURL 中显示真实 Token（默认脱敏 ***）
    RESP_MAX=0     响应正文不截断（默认 800 字符）
    QUIET=1        每次 HTTP 调用不打印执行帧，仅打印用例汇总
    NO_COLOR=1     禁用 ANSI 颜色
    TRACE=1        测试失败时打印完整 traceback
"""

# ── 从 client.py 导入 HTTP 基础设施 ──
from helpers.client import (  # noqa: F401
    # 核心变量
    API,
    ADMIN_TOKEN,
    NON_ADMIN_TOKEN,
    IDENTIFIER,
    SESSION_COOKIE,
    SESSION_COOKIE_B,
    TOKEN,
    # ANSI 颜色（供 helpers 内部模块使用）
    GREEN,
    RED,
    YELLOW,
    CYAN,
    GRAY,
    BOLD,
    # OOP 客户端
    ApiClient,
    admin_client,
    user_client,
    HEADERS,
    # 预置全局客户端
    seed,
    anon,
    bad_token,
    # Header 构建工具
    no_auth_headers,
    wrong_token_headers,
    non_admin_headers,
    bearer_header,
    cookie_header,
    user_headers,
    # 控制标记
    QUIET,
    # 截断工具
    truncate,
)

# ── 从 testing.py 导入测试辅助 ──
from helpers.testing import (  # noqa: F401
    # 日期常量
    TODAY,
    YESTERDAY,
    TOMORROW,
    FUTURE,
    # 断言工具
    assert_fields,
    assert_status,
    assert_sorted,
    # API 调用工厂
    make_api_fn,
    # 认证测试
    auth_test_suite,
    # 健康检查 / 环境检查
    health_check,
    check_env,
    require_model_config,
    require_model_config_multi,
    # 站点配置
    admin_get_config,
    admin_update_config,
    # 通用辅助
    get_field,
    ensure_gateway_ui_enabled,
    clear_instance_models,
    # Gateway 重启重试
    retry_on_gateway_restart,
    # 用户管理工具
    pick_user,
    extract_uid,
    list_users_by_prefix,
    cleanup_users_by_prefix,
    # 测试运行器
    run_tests,
)

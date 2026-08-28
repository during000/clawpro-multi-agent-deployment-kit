"""
集成测试辅助函数包

按模块拆分 API 调用，方便后续扩展。

使用方式:
    import helpers
    helpers.health_check()
    helpers.admin_create_user(...)
    helpers.check_env()
    helpers.setup_admin("my-scenario")

    from helpers import config
    print(config.BASE_URL)

    from helpers import ApiClient, admin_client, user_client

    # 统一 API 工具（HTTP 帧记录 + 测试运行器 + 断言 + 小工具）
    from helpers.api import (
        API, ADMIN_TOKEN, HEADERS, health_check, run_tests,
        seed, anon, bad_token,
        assert_status, assert_fields,
        pick_user, extract_uid, cleanup_users_by_prefix,
    )
"""

# ── 配置 ──
from helpers import config

# ── HTTP 基础设施 + 站点配置 + 环境检查 + 重试工具 ──
from helpers.api import (
    ApiClient,
    admin_client,
    user_client,
    # 预置全局客户端
    seed,
    anon,
    bad_token,
    health_check,
    admin_get_config,
    admin_update_config,
    check_env,
    require_model_config,
    require_model_config_multi,
    retry_on_gateway_restart,
    get_field,
    ensure_gateway_ui_enabled,
    clear_instance_models,
)

# ── 管理员接口 ──
from helpers.user_mgmt import (
    admin_create_user,
    admin_get_user_token,
    admin_enable_token,
    setup_admin,
    setup_user,
    teardown_scenario_users,
)
from helpers.model import (
    admin_create_model,
    admin_get_models,
    admin_get_default_model_id,
    admin_toggle_model,
    admin_toggle_model_enabled,
    admin_toggle_default_model,
    admin_update_model,
    admin_delete_model,
    setup_model,
    teardown_model,
    ensure_custom_model_flag,
)
from helpers.channel import (
    admin_get_channels,
    admin_toggle_channel,
    admin_delete_channel,
    extract_user_channel_ids,
    filter_site_visible_channels,
    is_overseas_site,
)

# ── 用户侧接口 ──
from helpers.instance import (
    create_instance,
    delete_instance,
    list_instances,
    wait_instance_ready,
    get_instance_db_id,
    wait_gateway_ready,
    setup_instance,
)
from helpers.model import (
    user_get_models,
    user_get_instance_models,
    user_add_model,
    user_set_model,
    user_del_model,
    user_switch_primary_model,
)
from helpers.channel import (
    user_get_channels,
    user_set_channel,
    user_del_channel,
    user_auto_channel,
)
from helpers.skill import (
    user_get_skills,
    user_add_skill,
    user_update_skill,
    user_uninstall_skill,
    user_get_install_skills,
    user_retry_failed_skills,
    user_cancel_failed_skills,
    wait_skills_installed,
)

# ── 本地 agent (source=local) ──
from helpers.local_agent import (
    LocalAgent,
    DEFAULT_AGENT_TYPE,
    DEFAULT_AGENT_VERSION,
    LOCAL_AGENT_ID_HEX_LEN,
    enable_local_agent_feature,
    random_local_agent_id,
    now_rfc3339,
    reporter_report,
    reporter_sync,
    reporter_ack,
    setup_local_instance,
    user_get_local_agent_availability,
    user_remove_local_agent,
    admin_get_local_agent_types,
    admin_check_feature_allowlist,
)

# ── 通用断言工具 ──
from helpers.api import (
    assert_status,
    assert_fields,
    assert_sorted,
    auth_test_suite,
    run_tests,
)

# ── OpenClaw Gateway：WebSocket 客户端 + 验证函数 ──
from helpers.openclaw_gateway import (
    OpenClawGateway,
    connect_from_inst,
    verify_model_config_via_inst,
    verify_feishu_delivery,
    verify_wecom_delivery,
    verify_qqbot_delivery,
    verify_dingtalk_delivery,
    verify_model_available,
    verify_channel_configured,
)

# ── Hermes 专用辅助函数 ──
from helpers.hermes import (
    HERMES_AGENT_TYPE,
    HERMES_WHITELIST_CHANNELS,
    ensure_hermes_image,
    ensure_hermes_agent_type_enabled,
    setup_hermes_instance,
    verify_hermes_service,
    expect_hermes_channel_connected,
)

# ── 插件管理 ──
from helpers.plugin import (
    admin_create_plugin,
    admin_list_plugins,
    admin_plugin_detail,
    admin_update_plugin,
    admin_distribute_plugin,
    admin_uninstall_plugin,
    admin_plugin_instances,
    admin_plugin_tasks,
    admin_delete_plugin,
    wait_plugin_task,
)

# ── 企业技能（管控端）管理 ──
from helpers.admin_skill import (
    build_skill_zip,
    admin_create_skill,
    admin_distribute_skill,
    admin_skill_instances,
    admin_skill_find_instance,
    wait_skill_instance_status,
    wait_skill_distributed,
    wait_skill_settled,
    admin_list_skills,
    admin_find_skill,
    admin_skill_status,
    admin_delete_skill,
    admin_offline_skill,
    admin_online_skill,
)

# ── 技能共建审核 ──
from helpers.contribution import (
    contribute_skill,
    takedown_skill,
    withdraw_contribution,
    my_contributions,
    my_contribution_detail,
    admin_list_contributions,
    admin_contribution_detail,
    admin_approve_contribution,
    admin_reject_contribution,
    skillstore_has_slug,
)

# ── 企业 MCP（管控端）管理 ──
from helpers.admin_mcp import (
    build_mcp_config_json,
    admin_create_mcp,
    admin_distribute_mcp,
    admin_mcp_instances,
    admin_mcp_find_instance,
    wait_mcp_instance_status,
    wait_mcp_distributed,
    wait_mcp_settled,
)

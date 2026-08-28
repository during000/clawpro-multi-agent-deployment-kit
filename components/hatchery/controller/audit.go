package controller

import (
	"net/http"
	"strings"
	"time"

	"hatchery/common"
	"hatchery/model"
)

type auditRule struct {
	Action   string
	Resource string
}

var auditRules = map[string]auditRule{
	"/admin/create":                                      {"user_create", "user"},
	"/admin/batch-create":                                {"user_batch_create", "user"},
	"/admin/delete":                                      {"user_delete", "user"},
	"/admin/hard-delete":                                 {"user_hard_delete", "user"},
	"/admin/restore":                                     {"user_restore", "user"},
	"/admin/reset-password":                              {"user_reset_password", "user"},
	"/admin/update-user":                                 {"user_update", "user"},
	"/admin/export-tokens":                               {"user_export_tokens", "user"},
	"/admin/token/disable":                               {"token_disable", "api_token"},
	"/admin/token/enable":                                {"token_enable", "api_token"},
	"/admin/config":                                      {"config_update", "site_config"},
	"/admin/config/cvm":                                  {"cvm_config_update", "site_config"},
	"/admin/group-vpc-configs/create":                    {"group_vpc_config_create", "group_vpc_config"},
	"/admin/group-vpc-configs/update":                    {"group_vpc_config_update", "group_vpc_config"},
	"/admin/group-vpc-configs/delete":                    {"group_vpc_config_delete", "group_vpc_config"},
	"/admin/resource-policies/create":                    {"resource_policy_create", "resource_policy"},
	"/admin/resource-policies/update":                    {"resource_policy_update", "resource_policy"},
	"/admin/resource-policies/delete":                    {"resource_policy_delete", "resource_policy"},
	"/admin/config/smh":                                  {"smh_config_update", "smh_config"},
	"/admin/smh/personal-space-auto-provision":           {"smh_auto_provision_update", "smh_config"},
	"/admin/smh/instance-space":                          {"smh_instance_space_update", "smh_personal_space"},
	"/admin/config/template":                             {"template_update", "site_config"},
	"/admin/config/security-group":                       {"security_group_update", "site_config"},
	"/admin/config/security-group/policies":              {"security_group_policy_update", "site_config"},
	"/admin/config/security-group/ruleset/rules/reorder": {"security_group_ruleset_reorder", "site_config"},
	"/admin/models/create":                               {"model_create", "ai_model"},
	"/admin/models/update":                               {"model_update", "ai_model"},
	"/admin/models/delete":                               {"model_delete", "ai_model"},
	"/admin/models/toggle":                               {"model_toggle", "ai_model"},
	"/admin/models/toggle-enabled":                       {"model_toggle_enabled", "ai_model"},
	"/admin/models/toggle-default":                       {"model_toggle_default", "ai_model"},
	"/admin/models/visibility":                           {"model_visibility_update", "ai_model"},
	"/admin/tags/create":                                 {"tag_create", "tag"},
	"/admin/tags/update":                                 {"tag_update", "tag"},
	"/admin/tags/replace-all":                            {"tag_replace_all", "tag"},
	"/admin/tags/delete":                                 {"tag_delete", "tag"},
	"/admin/channels/toggle":                             {"channel_toggle", "ai_channel"},
	"/admin/channels/add":                                {"channel_add", "ai_channel"},
	"/admin/channels/delete":                             {"channel_delete", "ai_channel"},
	"/admin/images/import":                               {"image_import", "ai_image"},
	"/admin/images/delete":                               {"image_delete", "ai_image"},
	"/admin/images/enable":                               {"image_toggle", "ai_image"},
	"/admin/images/update":                               {"image_update", "ai_image"},
	"/admin/images/history/publish":                      {"image_history_publish", "image_history"},
	"/admin/images/history/update":                       {"image_history_update", "image_history"},
	"/admin/images/history/delete":                       {"image_history_delete", "image_history"},
	"/admin/images/history/restore":                      {"image_history_restore", "image_history"},
	"/admin/images/update-notice":                        {"image_update_notice", "ai_image"},
	"/admin/images/set-default-type":                     {"set_default_agent_type", "agent_type"},
	"/admin/images/type-visibility":                      {"image_type_visibility_update", "ai_image"},
	"/admin/channels/visibility":                         {"channel_visibility_update", "ai_channel"},
	"/admin/mcp/visibility":                              {"mcp_visibility_update", "mcp_server"},
	"/admin/group-config/policy":                         {"group_policy_set", "group_config"},
	"/admin/group-config/policy/delete":                  {"group_policy_delete", "group_config"},
	"/admin/agent-types/create":                          {"agent_type_create", "agent_type"},
	"/admin/agent-types/delete":                          {"agent_type_delete", "agent_type"},
	"/admin/agent-types/enabled":                         {"agent_type_enabled_update", "agent_type"},
	// Agent 命令执行（feature/agent_command_execution）
	"/admin/agent-commands/create":           {"agent_command_create", "agent_command"},
	"/admin/agent-commands/update":           {"agent_command_update", "agent_command"},
	"/admin/agent-commands/delete":           {"agent_command_delete", "agent_command"},
	"/admin/agent-commands/dispatch":         {"agent_command_dispatch", "agent_command_dispatch"},
	"/admin/agent-commands/schedules/create": {"agent_command_schedule_create", "agent_command_schedule"},
	"/admin/agent-commands/schedules/delete": {"agent_command_schedule_delete", "agent_command_schedule"},
	"/admin/agent-commands/schedules/toggle": {"agent_command_schedule_toggle", "agent_command_schedule"},
	"/admin/instances/create":                {"instance_admin_create", "instance"},
	"/admin/instances/adjust-config":         {"instance_adjust_config", "instance"},
	"/admin/instances/delete":                {"instance_delete", "instance"},
	"/admin/instances/start":                 {"instance_start", "instance"},
	"/admin/instances/stop":                  {"instance_stop", "instance"},
	"/admin/instances/reboot":                {"instance_reboot", "instance"},
	"/admin/instances/restart-gateway":       {"instance_restart_gateway", "instance"},
	"/admin/instances/reset":                 {"instance_reset", "instance"},
	"/admin/instances/cam-role":              {"instance_cam_role", "instance"},
	"/admin/instances/terminal-url":          {"instance_terminal", "instance"},
	"/admin/instances/refresh-version":       {"instance_refresh_version", "instance"},
	"/admin/local-agent/remove":              {"local_agent_remove", "instance"},
	"/admin/instances/batch-upgrade":         {"instance_batch_upgrade", "instance"},
	"/admin/instances/set-model":             {"instance_admin_set_model", "instance"},
	"/admin/instances/batch-set-model":       {"instance_admin_batch_set_model", "instance"},
	"/admin/instances/add-model":             {"instance_admin_add_model", "instance"},
	"/admin/instances/switch-primary-model":  {"instance_admin_switch_primary_model", "instance"},
	"/admin/instances/del-model":             {"instance_admin_del_model", "instance"},
	"/admin/instances/proxy/prepare":         {"instance_admin_proxy_prepare", "agent_proxy_route"},
	"/admin/instances/set-channel":           {"instance_admin_set_channel", "instance"},
	"/admin/instances/del-channel":           {"instance_admin_del_channel", "instance"},
	"/admin/cls/open":                        {"cls_open", "cls"},
	"/admin/cls/close":                       {"cls_close", "cls"},
	"/admin/cls/scope":                       {"cls_scope_update", "cls"},
	"/admin/cls/update":                      {"cls_plugin_update", "cls"},
	"/openclaw/create":                       {"instance_create", "instance"},
	"/openclaw/delete":                       {"instance_delete", "instance"},
	"/openclaw/start":                        {"instance_start", "instance"},
	"/openclaw/stop":                         {"instance_stop", "instance"},
	"/openclaw/reboot":                       {"instance_reboot", "instance"},
	"/openclaw/restart-gateway":              {"instance_restart_gateway", "instance"},
	"/openclaw/reset":                        {"instance_reset", "instance"},
	"/openclaw/set-model":                    {"instance_set_model", "instance"},
	"/openclaw/proxy/prepare":                {"instance_proxy_prepare", "agent_proxy_route"},
	"/openclaw/set-channel":                  {"instance_set_channel", "instance"},
	"/openclaw/del-channel":                  {"instance_del_channel", "instance"},
	"/openclaw/auto-channel":                 {"instance_auto_channel", "instance"},
	"/openclaw/add-skill":                    {"instance_add_skill", "instance"},
	"/openclaw/upgrade":                      {"instance_upgrade", "instance"},
	"/openclaw/upgrade/retry":                {"instance_upgrade_retry", "instance"},
	"/openclaw/migration/export":             {"instance_migration_export", "instance"},
	"/openclaw/migration/import":             {"instance_migration_import", "instance"},
	"/openclaw/add-plugin":                   {"instance_add_plugin", "instance"},
	"/openclaw/approve":                      {"instance_approve", "instance"},
	"/openclaw/set-gateway-ui":               {"instance_set_gateway_ui", "instance"},
	"/openclaw/set-env":                      {"instance_set_env", "instance"},
	"/openclaw/terminal-url":                 {"instance_terminal", "instance"},
	"/openclaw/retry":                        {"instance_retry", "instance"},
	"/openclaw/remove-role":                  {"instance_remove_role", "instance"},
	"/openclaw/switch-role":                  {"instance_switch_role", "instance"},
	"/admin/roles/distribute":                {"role_distribute", "role"},
	"/openclaw/notifications/read":           {"notification_read", "notification"},
	"/openclaw/notifications/delete":         {"notification_delete", "notification"},
	// 本地 Agent 上报、同步和用户级分组切换
	"/local-agent/report":             {"local_agent_report", "instance"},
	"/local-agent/sync":               {"local_agent_sync", "instance"},
	"/local-agent/commands/ack":       {"local_agent_command_ack", "instance"},
	"/local-agent/wake-ticket":        {"local_agent_wake_ticket", "instance"},
	"/local-agent/remove":             {"local_agent_remove", "instance"},
	"/agent-tasks/create":             {"agent_task_create", "local_agent_task"},
	"/openclaw/local/user-group":      {"local_agent_switch_user_group", "instance"},
	"/change-password":                {"user_change_password", "user"},
	"/oneid/enterprise":               {"oneid_login", "session"},
	"/oneid/password-reset":           {"oneid_password_reset", "user"},
	"/oneid/password-change":          {"oneid_password_change", "user"},
	"/api-token/create":               {"token_create", "api_token"},
	"/api-token/reset":                {"token_reset", "api_token"},
	"/api-token/revoke":               {"token_revoke", "api_token"},
	"/auth/internal-login":            {"internal_login", "session"},
	"/auth/oneid-register":            {"oneid_register", "session"},
	"/auth/passwordless-login":        {"passwordless_login", "session"},
	"/admin/passwordless-login-link":  {"passwordless_login_link_create", "session"},
	"/spi/logout":                     {"oneid_logout", "session"},
	"/spi/event":                      {"oneid_event", "user"},
	"/admin/skill-categories/create":  {"skill_category_create", "skill_category"},
	"/admin/skill-categories/update":  {"skill_category_update", "skill_category"},
	"/admin/skill-categories/delete":  {"skill_category_delete", "skill_category"},
	"/admin/skills/create":            {"skill_create", "skill"},
	"/admin/skills/update":            {"skill_update", "skill"},
	"/admin/skills/delete":            {"skill_delete", "skill"},
	"/admin/skills/distribute":        {"skill_distribute", "skill"},
	"/admin/skills/uninstall":         {"skill_uninstall", "skill"},
	"/openclaw/skillstore/distribute": {"skillstore_distribute", "skill"},
	"/openclaw/skillstore/uninstall":  {"skillstore_uninstall", "skill"},
	"/openclaw/skillstore/download":   {"skillstore_download", "skill"},
	"/openclaw/update-skill":          {"skill_update", "skill"},
	"/openclaw/uninstall-skill":       {"skill_uninstall", "skill"},
	"/admin/skills/download":          {"admin_skill_download", "skill"},
	// 企业规范库（本地 agent 二期）审计条目
	"/admin/rules/create":                    {"rule_create", "rule"},
	"/admin/rules/delete":                    {"rule_delete", "rule"},
	"/admin/rules/update":                    {"rule_update", "rule"},
	"/admin/rules/distribute":                {"rule_distribute", "rule"},
	"/admin/rules/uninstall":                 {"rule_uninstall", "rule"},
	"/admin/skills/scan-trigger":             {"skill_scan_trigger", "skill"},
	"/admin/skills/scan-config":              {"skill_scan_config_update", "site_config"},
	"/admin/skill-bundles/create":            {"skill_bundle_create", "skill_bundle"},
	"/admin/skill-bundles/delete":            {"skill_bundle_delete", "skill_bundle"},
	"/admin/skill-bundles/toggle":            {"skill_bundle_toggle", "skill_bundle"},
	"/admin/skill-bundles/update-skills":     {"skill_bundle_update_skills", "skill_bundle"},
	"/admin/skill-bundles/batch-add-skills":  {"skill_bundle_batch_add_skills", "skill_bundle"},
	"/admin/skill-bundles/update-visibility": {"skill_bundle_update_visibility", "skill_bundle"},
	"/admin/skills/favorite":                 {"skill_favorite", "public_skill"},
	"/admin/skills/unfavorite":               {"skill_unfavorite", "public_skill"},
	"/admin/skillsets/favorite":              {"skillset_favorite", "public_skillset"},
	"/admin/skillsets/unfavorite":            {"skillset_unfavorite", "public_skillset"},
	// 技能共建审核
	"/openclaw/skills/contribute":             {"skill_contribute", "review_request"},
	"/openclaw/skills/takedown":               {"skill_takedown_request", "review_request"},
	"/admin/contributions/approve":            {"review_approve", "review_request"},
	"/admin/contributions/reject":             {"review_reject", "review_request"},
	"/openclaw/skills/contributions/withdraw": {"review_withdraw", "review_request"},
	"/admin/skills/offline":                   {"skill_offline", "skill"},
	"/admin/skills/online":                    {"skill_online", "skill"},
	"/openclaw/retry-failed-skills":           {"instance_retry_skills", "instance"},
	"/openclaw/cancel-failed-skills":          {"instance_cancel_skills", "instance"},
	"/openclaw/browser-vnc-install":           {"browser_vnc_install", "instance"},
	"/openclaw/browser-takeover":              {"browser_takeover", "instance"},
	// Role management
	// Role management
	"/admin/roles/create":         {"role_create", "role"},
	"/admin/roles/update":         {"role_update", "role"},
	"/admin/roles/delete":         {"role_delete", "role"},
	"/admin/roles/toggle-visible": {"role_toggle_visible", "role"},
	"/admin/roles/reorder":        {"role_reorder", "role"},
	// Security group bind（master 老接口，与 sg-ruleset-projection 并存）
	"/admin/config/security-group/bind": {"security_group_bind", "site_config"},
	// Security group rule set (sg-ruleset-projection)
	"/admin/config/security-group/ruleset/rules":          {"update_rule_set", "rule_set"},
	"/admin/config/security-group/ruleset/import-from-sg": {"import_rules_from_sg", "rule_set"},
	"/admin/config/security-group/rulesets":               {"create_ruleset", "rule_set"},
	// Instance detect install
	"/admin/instances/detect-install":   {"instance_detect_install", "instance"},
	"/admin/oneid-sync-enterprise":      {"oneid_sync_enterprise", "user"},
	"/admin/oneid-sync-users":           {"oneid_sync_users", "user"},
	"/openclaw/lightclaw/run-command":   {"lightclaw_run_command", "instance"},
	"/admin/user-groups/create":         {"user_group_create", "user_group"},
	"/admin/user-groups/update":         {"user_group_update", "user_group"},
	"/admin/user-groups/delete":         {"user_group_delete", "user_group"},
	"/admin/user-groups/members/set":    {"user_group_members_set", "user_group"},
	"/admin/user-groups/members/add":    {"user_group_members_add", "user_group"},
	"/admin/user-groups/members/remove": {"user_group_members_remove", "user_group"},
	"/admin/projects/create":            {"project_create", "project"},
	"/admin/projects/update":            {"project_update", "project"},
	"/admin/projects/delete":            {"project_delete", "project"},
	"/admin/projects/members/set":       {"project_members_set", "project"},
	"/admin/projects/members/add":       {"project_members_add", "project"},
	"/admin/projects/members/remove":    {"project_members_remove", "project"},
	"/admin/assets/save":                {"assets_save", "asset_binding"},
	// Stale-instances v1.0（存量实例分组归属处理）
	"/admin/stale-instances/apply":       {"stale_instances_apply", "instance"},
	"/openclaw/stale-instances/rebind":   {"stale_instances_user_rebind", "instance"},
	"/openclaw/stale-instances/initiate": {"stale_instances_user_handover_initiate", "instance"},
	"/openclaw/stale-instances/cancel":   {"stale_instances_user_handover_cancel", "instance"},
	"/openclaw/stale-instances/accept":   {"stale_instances_user_handover_accept", "instance"},
	"/openclaw/stale-instances/reject":   {"stale_instances_user_handover_reject", "instance"},
	// Plugin management
	"/admin/plugins/create":     {"plugin_create", "plugin"},
	"/admin/plugins/update":     {"plugin_update", "plugin"},
	"/admin/plugins/delete":     {"plugin_delete", "plugin"},
	"/admin/plugins/distribute": {"plugin_distribute", "plugin"},
	"/admin/plugins/uninstall":  {"plugin_uninstall", "plugin"},
	"/admin/plugins/favorite":   {"plugin_favorite", "public_plugin"},
	"/admin/plugins/unfavorite": {"plugin_unfavorite", "public_plugin"},
	// Plugin category management
	"/admin/plugin-categories/create": {"plugin_category_create", "plugin_category"},
	"/admin/plugin-categories/update": {"plugin_category_update", "plugin_category"},
	"/admin/plugin-categories/delete": {"plugin_category_delete", "plugin_category"},
	// Plugin bundle management
	"/admin/plugin-bundles/create":         {"plugin_bundle_create", "plugin_bundle"},
	"/admin/plugin-bundles/delete":         {"plugin_bundle_delete", "plugin_bundle"},
	"/admin/plugin-bundles/toggle":         {"plugin_bundle_toggle", "plugin_bundle"},
	"/admin/plugin-bundles/update-plugins": {"plugin_bundle_update_plugins", "plugin_bundle"},
	// Memory Pro management
	"/admin/memory/pro/activate":           {"memory_pro_activate", "memory_pro"},
	"/admin/memory/pro/release":            {"memory_pro_release", "memory_pro"},
	"/admin/memory/plan/switch":            {"memory_plan_batch_switch", "instance"},
	"/admin/memory/plugin-upgrade/execute": {"memory_plugin_upgrade_execute", "instance"},
	"/admin/memory/default-plan":           {"memory_default_plan_update", "site_config"},
	// Memory TDAI (legacy) — 补齐历史漏登记
	"/admin/memory-tdai/config": {"memory_tdai_config_update", "site_config"},
	// MCP 企业库
	"/admin/mcp/create":     {"mcp_create", "mcp"},
	"/admin/mcp/update":     {"mcp_update", "mcp_version"},
	"/admin/mcp/meta":       {"mcp_meta_update", "mcp"},
	"/admin/mcp/delete":     {"mcp_delete", "mcp"},
	"/admin/mcp/distribute": {"mcp_distribute", "mcp_batch"},

	// 龙虾医生
	"/openclaw/doctor/quick-fix": {"doctor_quick_fix", "doctor_session"},
	"/openclaw/doctor/authorize": {"doctor_authorize", "doctor_authorization"},
	"/openclaw/doctor/start":     {"doctor_start", "doctor_session"},
	"/openclaw/doctor/end":       {"doctor_end", "doctor_session"},

	// 用户端 MCP 管理
	"/openclaw/mcp/add":           {"instance_add_mcp", "instance"},
	"/openclaw/mcp/update-config": {"instance_update_mcp", "instance"},
	"/openclaw/mcp/delete":        {"instance_delete_mcp", "instance"},
	"/openclaw/mcp/toggle":        {"instance_toggle_mcp", "instance"},

	// 多租户管理
	"/tenants/init":    {"tenant_init", "tenant"},
	"/tenants/domains": {"tenant_domain_manage", "tenant"},
}

func WithAudit(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			// fall through to audit logic
		default:
			handler(w, r)
			return
		}

		rule, ok := auditRules[r.URL.Path]
		if !ok {
			handler(w, r)
			return
		}

		var userID uint
		var username string
		if user, err := RequestUser(r); user != nil && err == nil {
			userID = user.ID
			username = user.Username
		}
		resourceID := r.URL.Query().Get("id")
		if resourceID == "" {
			resourceID = r.FormValue("id")
		}

		startedAt := time.Now()
		handler(w, r)

		status := "success"
		if w.Header().Get("X-Audit-Failed") != "" {
			status = "failed"
		}
		w.Header().Del("X-Audit-Failed")

		go model.LogAudit(common.DetachContext(r.Context()), startedAt, userID, username, rule.Action, rule.Resource, resourceID, status)
	}
}

// WithCloudProxyAudit 是专门用于云 API 写接口的审计中间件。
// 与 WithAudit 不同，它从请求的 X-TC-Action Header 和 URL 路径中动态提取审计信息：
//   - Action:   "cloud_proxy_{X-TC-Action}" (如 "cloud_proxy_RunInstances")
//   - Resource: URL 中的 service 名称 (如 "cloud_cvm")
//   - ResourceID: X-TC-Action 的值
func WithCloudProxyAudit(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			handler(w, r)
			return
		}

		// 从 URL 提取 service: /admin/cloud/mutate/{service}
		path := strings.TrimPrefix(r.URL.Path, "/admin/cloud/mutate/")
		service := strings.TrimRight(path, "/")

		// 从 Header 提取 Action
		action := r.Header.Get("X-TC-Action")
		if action == "" {
			action = r.URL.Query().Get("Action")
		}

		var userID uint
		var username string
		if user, err := RequestUser(r); user != nil && err == nil {
			userID = user.ID
			username = user.Username
		}

		startedAt := time.Now()
		handler(w, r)

		status := "success"
		if w.Header().Get("X-Audit-Failed") != "" {
			status = "failed"
		}
		w.Header().Del("X-Audit-Failed")

		auditAction := "cloud_proxy_" + action
		resource := "cloud_" + service

		go model.LogAudit(common.DetachContext(r.Context()), startedAt, userID, username, auditAction, resource, action, status)
	}
}

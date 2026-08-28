-- Hatchery MySQL 初始化脚本
-- 适用于全新 MySQL 数据库，请勿在已有数据的库上执行
--
-- 与 GORM 模型 tag 的差异：
-- MySQL 不允许 TEXT/BLOB 列设置 DEFAULT 值，以下字段的 GORM tag 中声明了
-- type:text + default，但在本 SQL 中退化为 nullable 且无默认值：
--   - site_configs.memory_tdai_supported_versions  GORM: type:text;not null;default:'[]'  → SQL: varchar(1024) NOT NULL DEFAULT '[]'（改用 varchar 保留 default）
--   - ai_channels.custom_config                    GORM: type:text;default:''             → SQL: text NULL（丢失 default）
--   - one_id_user_profiles.departments_json        GORM: type:text;default:'[]'           → SQL: text NULL（丢失 default）
-- 应用层需要通过 GORM 写入默认值，或者评估改成varchar

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS `site_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'Hatchery',
  `logo` longblob,
  `logo_mime` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `c_vm_secret_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `c_vm_secret_key` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `c_vm_template` text COLLATE utf8mb4_unicode_ci,
  `security_group_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `auto_created_security_group_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `skill_hub` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `skill_hub_enabled` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否启用 SkillHub 迁移',
  `skill_hub_api_url` text COLLATE utf8mb4_unicode_ci COMMENT 'SkillHub API 请求地址',
  `global_token_quota_day` bigint NOT NULL DEFAULT '-1',
  `global_token_quota_period` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'day',
  `global_token_quota_rules` text,
  `public_image_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `vpc_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `subnet_ids` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `cls_enabled` bigint NOT NULL DEFAULT '0',
  `cls_scope_mode` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  `agent_cam_role_secret_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `agent_cam_role_secret_key` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sts_tmp_secret_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sts_tmp_secret_key` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sts_token` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sts_expired_at` bigint DEFAULT NULL,
  `terminal_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `chat_view_enabled` tinyint(1) NOT NULL DEFAULT '1',
  `gateway_ui_enable` tinyint(1) NOT NULL DEFAULT '0',
  `gateway_ui_port` bigint NOT NULL DEFAULT '0',
  `gateway_ui_sg_migrate_done` tinyint(1) NOT NULL DEFAULT '0',
  `gateway_ui_addr_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'public',
  `browser_vnc_enable` tinyint(1) NOT NULL DEFAULT '0',
  `user_data_enabled` tinyint(1) NOT NULL DEFAULT '0',
  `user_config_model_enabled` tinyint(1) NOT NULL DEFAULT '1',
  `user_config_channel_enabled` tinyint(1) NOT NULL DEFAULT '1',
  `model_quota_enabled` tinyint(1) NOT NULL DEFAULT '1',
  `memory_tdai_enable` tinyint(1) NOT NULL DEFAULT '0',
  `memory_tdai_supported_versions` varchar(1024) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '[]',
  `memory_default_plan` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'off',
  `skill_distribute_concurrency` bigint NOT NULL DEFAULT '100',
  `smh_enabled` bigint NOT NULL DEFAULT '0',
  `smh_auto_provision_on_create` tinyint(1) NOT NULL DEFAULT '0',
  `smh_library_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `smh_library_secret` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `smh_endpoint` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `smh_provision_error` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ssoim_type` varchar(512) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `default_instance_quota` bigint NOT NULL DEFAULT '3',
  `default_token_quota_day` bigint NOT NULL DEFAULT '500000',
  `default_token_quota_rules` text DEFAULT NULL,
  `default_model_id` bigint unsigned NOT NULL DEFAULT '0',
  `default_bundle_seeded` tinyint(1) NOT NULL DEFAULT '0',
  `default_plugin_bundle_seeded` tinyint(1) NOT NULL DEFAULT '0',
  `default_roles_seeded` tinyint(1) NOT NULL DEFAULT '0',
  `default_vpc_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_subnet_ids` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `default_agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'openclaw' COMMENT '用户端首选智能体类型',
  `disabled_agent_types` text COLLATE utf8mb4_unicode_ci NULL COMMENT '用户端禁用的智能体类型 JSON 数组',
  `default_tags` varchar(4096) COLLATE utf8mb4_unicode_ci DEFAULT '[]',
  `doctor_enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否允许用户使用龙虾医生，默认关闭',
  `local_agent_enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否允许用户接入本地 Agent，默认关闭；与 feature_allowlist 一起作双层守卫',
  `api_gateway_config` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT '{}' COMMENT 'WebUI 接入云 API 网关配置 JSON，详见 webui-apigateway change',
  `sg_pool_auto_scale_threshold` int NOT NULL DEFAULT 1800 COMMENT 'SG 池单 SG 实例数达到此值触发扩容，默认 1800',
  `uin` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '租户腾讯云 UIN',
  `domain` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '租户对外访问域名',
  `internal_secret` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Gateway 内部鉴权密钥',
  `one_id_account_id` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'OneID 租户账号 ID',
  `one_id_app_id` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'OneID 自建应用 ID',
  `one_id_client_id` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'OneID 应用 client_id',
  `one_id_client_secret` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'OneID 应用 client_secret',
  `one_id_token_endpoint` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'OneID OIDC Token 端点 URL',
  `one_id_domain` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'OneID 企业域名',
  `default_lang` varchar(8) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '租户默认语言：zh 或 en',
  `security_policies` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'SSRF',
  `skill_scan_default_enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '上传技能时安全检测勾选框默认值',
  `last_full_sync_finished_at` datetime(3) DEFAULT NULL COMMENT '后台 full-sync 整轮完成时间，非 NULL 表示缓存就绪',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_site_configs_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `username` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `email` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `password` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `role` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'user',
  `instance_quota` bigint NOT NULL DEFAULT '1',
  `token_quota_day` bigint NOT NULL DEFAULT '-1',
  `token_quota_rules` text DEFAULT NULL,
  `vpc_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `subnet_ids` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `api_token` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `one_id_sub` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `oneid_login_name` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'OneID 登录名',
  `api_token_disabled` tinyint(1) NOT NULL DEFAULT '0',
  `api_token_created_at` datetime(3) DEFAULT NULL,
  `api_token_last_used_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_username_identifier` (`identifier`,`username`),
  UNIQUE KEY `idx_oneid_sub_identifier` (`identifier`,`one_id_sub`),
  UNIQUE KEY `idx_users_api_token` (`api_token`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `passwordless_login_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `token_hash` char(64) COLLATE utf8mb4_bin NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_passwordless_login_tokens_hash` (`token_hash`),
  KEY `idx_passwordless_login_tokens_identifier` (`identifier`),
  KEY `idx_passwordless_login_tokens_user_id` (`user_id`),
  KEY `idx_passwordless_login_tokens_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE IF NOT EXISTS `instances` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `instance_charge_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'PREPAID',
  `user_id` bigint unsigned NOT NULL DEFAULT '0',
  `vpc_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `subnet_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `security_group_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'ClawPro 管理的实例主 SG，sg-ruleset-projection change 引入',
  `proxy_token` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ai_model_id` bigint unsigned DEFAULT NULL,
  `max_tokens` bigint DEFAULT '0',
  `custom_model_config` text COLLATE utf8mb4_unicode_ci,
  `user_data` text COLLATE utf8mb4_unicode_ci,
  `cls_agent_status` bigint DEFAULT '0',
  `cls_agent_status_at` datetime(3) DEFAULT NULL,
  `cls_plugin_version` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1.0' COMMENT 'CLS 插件版本，1.0=旧版（无 trace），2.0=新版（含 trace）',
  `current_operation` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `current_operation_state` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `current_operation_updated_at` datetime(3) DEFAULT NULL,
  `last_stable_state` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `last_cvm_state` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `agent_ready` bigint DEFAULT '0',
  `runtime_user` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `runtime_home` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `role_id` bigint unsigned DEFAULT '0',
  `soul_set_at` datetime(3) DEFAULT NULL,
  `distributed_role_version` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '最近一次成功推送到此实例的角色版本号，X.Y 格式；空串=未下发过',
  `role_sync_status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '角色同步状态：空/pending/updating/updated/failed',
  `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'openclaw' COMMENT '智能体类型',
  `group_id` bigint unsigned NOT NULL DEFAULT '0',
  `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体版本号',
  `plugin_versions_json` text COLLATE utf8mb4_unicode_ci,
  `version_fetched_at` datetime(3) DEFAULT NULL,
  `pending_archive_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'CVM 上未上传完成的备份压缩包路径',
  `pending_archive_size` bigint NOT NULL DEFAULT '0' COMMENT '备份压缩包大小（字节）',
  `pending_smh_file_key` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'SMH 文件 key，用于复用同一个 ConfirmKey 续传',
  `pending_upload_at` datetime(3) DEFAULT NULL COMMENT '续传记录写入时间，便于运维判断陈旧程度',
  `is_doctor_node` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否为龙虾医生临时诊断节点',
  `last_known_status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '最终语义状态(running/stopped/load_failed/destroyed...)',
  `cvm_tags_json` text COLLATE utf8mb4_unicode_ci NULL COMMENT 'CVM 标签缓存 JSON，供标签过滤',
  `img_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'CVM 镜像 ID 缓存，供 IsOfficialImage 判断',
  `status_synced_at` datetime(3) DEFAULT NULL COMMENT '缓存最后同步时间，用于竞态保护',
  `cvm_instance_type` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cvm_cpu` bigint NOT NULL DEFAULT '0',
  `cvm_memory_gb` bigint NOT NULL DEFAULT '0',
  `system_disk_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `system_disk_size` bigint NOT NULL DEFAULT '0',
  `cvm_public_ip` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cvm_internet_charge_type` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cvm_internet_max_bandwidth_out` bigint NOT NULL DEFAULT '0',
  `source` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'cvm' COMMENT '实例来源：cvm（云上）/ local（本地 agent）',
  `local_agent_resources` text COLLATE utf8mb4_unicode_ci NULL COMMENT '本地 agent 二期：分组绑定 + workspace 列表 JSON（source=local 时存）',
  `handover_target_user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '同组移交目标用户 ID；0 表示无 pending 移交',
  `handover_rejected_by_user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '最近一次拒绝移交的用户 ID；0 表示无',
  `handover_initiated_at` datetime(3) DEFAULT NULL COMMENT '移交发起时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_instances_proxy_token` (`proxy_token`),
  KEY `idx_instances_deleted_at` (`deleted_at`),
  KEY `idx_instances_identifier` (`identifier`),
  KEY `idx_instances_instance_id` (`instance_id`),
  KEY `idx_instances_user_id` (`user_id`),
  KEY `idx_instances_ai_model_id` (`ai_model_id`),
  KEY `idx_instances_current_operation` (`current_operation`),
  KEY `idx_instances_current_operation_updated_at` (`current_operation_updated_at`),
  KEY `idx_instances_last_c_vm_state` (`last_cvm_state`),
  KEY `idx_instances_agent_ready` (`agent_ready`),
  KEY `idx_instances_role_id` (`role_id`),
  KEY `idx_instances_role_sync_status` (`role_sync_status`),
  KEY `idx_instances_group_id` (`group_id`),
  KEY `idx_instances_agent_type` (`agent_type`),
  KEY `idx_instances_user_agent_type` (`user_id`,`agent_type`),
  KEY `idx_instances_is_doctor_node` (`is_doctor_node`),
  KEY `idx_instances_identifier_agent_type_deleted_at` (`identifier`,`agent_type`,`deleted_at`),
  KEY `idx_instances_last_known_status` (`identifier`, `last_known_status`),
  KEY `idx_instances_source` (`source`),
  KEY `idx_instances_handover_target_user_id` (`handover_target_user_id`),
  KEY `idx_instances_cvm_instance_type` (`cvm_instance_type`),
  KEY `idx_instances_system_disk_size` (`system_disk_size`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `instance_adjustments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  `execution_started_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'processing',
  `adjustment_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `phase` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'queued',
  `payload_json` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `run_at` datetime(3) NOT NULL,
  `attempt` int NOT NULL DEFAULT '0',
  `error_code` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_instance_adjustment_instance` (`identifier`,`instance_id`),
  KEY `idx_instance_adjustment_due` (`status`,`run_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `agent_proxy_routes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `route_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `instance_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `kind` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_ip` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `target_port` bigint NOT NULL DEFAULT '0',
  `target_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_agent_proxy_routes_route_id` (`route_id`),
  UNIQUE KEY `idx_proxy_route_instance_kind` (`identifier`,`instance_id`,`kind`),
  KEY `idx_agent_proxy_routes_deleted_at` (`deleted_at`),
  KEY `idx_agent_proxy_routes_identifier` (`identifier`),
  KEY `idx_agent_proxy_routes_instance_id` (`instance_id`),
  KEY `idx_agent_proxy_routes_kind` (`kind`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ai_models` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `provider` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `model_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `model_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `api_key` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `url` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `model_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `input_types` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '["text"]',
  `context_len` bigint NOT NULL DEFAULT '0',
  `max_tokens` bigint NOT NULL DEFAULT '0',
  `custom_http_headers` varchar(4096) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `quota_day` bigint NOT NULL DEFAULT '-1',
  `enabled` tinyint(1) NOT NULL DEFAULT '0',
  `visible` tinyint(1) NOT NULL DEFAULT '0',
  `visibility_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  PRIMARY KEY (`id`),
  KEY `idx_ai_models_deleted_at` (`deleted_at`),
  KEY `idx_ai_models_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ai_channels` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `channel_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `custom` tinyint(1) NOT NULL DEFAULT '0',
  `custom_config` text COLLATE utf8mb4_unicode_ci,
  `visibility_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_channel_identifier` (`identifier`,`channel_id`),
  KEY `idx_ai_channels_deleted_at` (`deleted_at`),
  KEY `idx_ai_channels_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `llm_usage_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `instance_id` bigint unsigned NOT NULL DEFAULT '0',
  `user_id` bigint unsigned NOT NULL DEFAULT '0',
  `group_id` bigint unsigned NOT NULL DEFAULT '0',
  `ai_model_id` bigint unsigned DEFAULT NULL,
  `model` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `provider` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `prompt_tokens` bigint DEFAULT NULL,
  `completion_tokens` bigint DEFAULT NULL,
  `total_tokens` bigint DEFAULT NULL,
  `prompt_cache_read_tokens` bigint NOT NULL DEFAULT '0',
  `prompt_cache_write_tokens` bigint NOT NULL DEFAULT '0',
  `status_code` bigint DEFAULT NULL,
  `latency` bigint DEFAULT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_llm_usage_logs_identifier` (`identifier`),
  KEY `idx_llm_usage_logs_instance_id` (`instance_id`),
  KEY `idx_llm_usage_logs_user_id` (`user_id`),
  KEY `idx_logs_user_time_tokens` (`user_id`, `created_at`, `total_tokens`),
  KEY `idx_logs_time_tokens` (`created_at`, `total_tokens`),
  KEY `idx_logs_group_time_tokens` (`group_id`, `created_at`, `total_tokens`),
  KEY `idx_llm_usage_logs_group_id` (`group_id`),
  KEY `idx_llm_usage_logs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `daily_usage_summaries` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `date` datetime(3) NOT NULL DEFAULT '1970-01-01 00:00:00.000',
  `user_id` bigint unsigned NOT NULL DEFAULT '0',
  `instance_id` bigint unsigned NOT NULL DEFAULT '0',
  `ai_model_id` bigint unsigned NOT NULL DEFAULT '0',
  `group_id` bigint unsigned NOT NULL DEFAULT '0',
  `prompt_tokens` bigint NOT NULL DEFAULT '0',
  `completion_tokens` bigint NOT NULL DEFAULT '0',
  `total_tokens` bigint NOT NULL DEFAULT '0',
  `prompt_cache_read_tokens` bigint NOT NULL DEFAULT '0',
  `prompt_cache_write_tokens` bigint NOT NULL DEFAULT '0',
  `request_count` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_daily_summary_identifier` (`identifier`,`date`,`user_id`,`instance_id`,`ai_model_id`),
  KEY `idx_daily_usage_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ai_images` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `image_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `image_name` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `image_type` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `os_name` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `image_size` bigint DEFAULT NULL,
  `image_state` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT '0',
  `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体类型',
  `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体版本号',
  `update_notice_enabled` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否提示该官方镜像有更新',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_image_identifier` (`identifier`,`image_id`),
  KEY `idx_ai_images_identifier` (`identifier`),
  KEY `idx_ai_images_agent_type_enabled` (`agent_type`,`enabled`),
  KEY `idx_ai_images_update_notice_enabled` (`update_notice_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `image_history` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `image_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `published_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_image_history_image_id` (`image_id`),
  KEY `idx_image_history_agent_type` (`agent_type`),
  KEY `idx_image_history_agent_version` (`agent_version`),
  KEY `idx_image_history_published_at` (`published_at`),
  KEY `idx_image_history_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `custom_agent_types` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `compatible_with` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_custom_agent_type_identifier_name` (`identifier`,`name`),
  KEY `idx_custom_agent_types_identifier` (`identifier`),
  KEY `idx_custom_agent_types_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `started_at` datetime(3) NOT NULL DEFAULT '1970-01-01 00:00:00.000',
  `created_at` datetime(3) NOT NULL,
  `user_id` bigint unsigned NOT NULL DEFAULT '0',
  `username` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `action` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `resource` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `resource_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_audit_logs_identifier` (`identifier`),
  KEY `idx_audit_logs_created_at` (`created_at`),
  KEY `idx_audit_logs_user_id` (`user_id`),
  KEY `idx_audit_logs_action` (`action`),
  KEY `idx_audit_logs_identifier_user_id` (`identifier`,`user_id`),
  KEY `idx_audit_logs_identifier_username` (`identifier`,`username`),
  KEY `idx_audit_logs_identifier_resource_id` (`identifier`,`resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `session_blacklists` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `one_id_sid` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `one_id_sub` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `expire_at` datetime(3) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sid_identifier` (`identifier`,`one_id_sid`),
  KEY `idx_session_blacklists_identifier` (`identifier`),
  KEY `idx_session_blacklists_one_id_sub` (`one_id_sub`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `one_id_user_profiles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `one_id_sub` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `union_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `name` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `email` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `mobile` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `position` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `employee_number` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `status` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `main_dept_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `main_dept_name` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `main_dept_parent_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `departments_json` text COLLATE utf8mb4_unicode_ci,
  `synced_at` datetime(3) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_profile_sub_identifier` (`identifier`,`one_id_sub`),
  KEY `idx_one_id_user_profiles_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `oneid_departments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `department_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `department_name` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `department_parent_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `has_child` tinyint(1) DEFAULT '0',
  `direct_user_count` bigint DEFAULT '0',
  `recurve_user_count` bigint DEFAULT '0',
  `synced_at` datetime(3) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_dept_identifier` (`identifier`,`department_id`),
  KEY `idx_oneid_departments_identifier` (`identifier`),
  KEY `idx_oneid_departments_department_parent_id` (`department_parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `skill_categories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_category_name_identifier` (`identifier`,`name`),
  KEY `idx_skill_categories_identifier` (`identifier`),
  KEY `idx_skill_categories_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `skill_category_mappings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `skill_id` bigint unsigned NOT NULL,
  `category_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_skill_category_identifier` (`identifier`,`skill_id`,`category_id`),
  KEY `idx_skill_category_mappings_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `skills` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1.0.0',
  `version_major` bigint NOT NULL DEFAULT '0',
  `version_minor` bigint NOT NULL DEFAULT '0',
  `version_patch` bigint NOT NULL DEFAULT '0',
  `cos_zip_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cos_dir_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `file_list` text COLLATE utf8mb4_unicode_ci,
  `file_size` bigint NOT NULL DEFAULT '0',
  `visibility_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  `changelog` varchar(10000) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `distribute_count` bigint NOT NULL DEFAULT '0',
  `status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'published',
  `uploader_id` bigint unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_slug_version_identifier` (`identifier`,`slug`,`version`),
  KEY `idx_skills_identifier` (`identifier`),
  KEY `idx_skills_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `skill_distribution_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `skill_id` bigint unsigned NOT NULL,
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'enterprise',
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source_skillset_slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `batch_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `operator_id` bigint unsigned NOT NULL DEFAULT '0',
  `total` bigint NOT NULL DEFAULT '0',
  `success` bigint NOT NULL DEFAULT '0',
  `failed` bigint NOT NULL DEFAULT '0',
  `status` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'running',
  `type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'distribute',
  PRIMARY KEY (`id`),
  KEY `idx_skill_distribution_tasks_identifier` (`identifier`),
  KEY `idx_skill_distribution_tasks_deleted_at` (`deleted_at`),
  KEY `idx_skill_distribution_tasks_skill_id` (`skill_id`),
  KEY `idx_skill_distribution_tasks_source_slug` (`source`,`slug`),
  KEY `idx_skill_distribution_tasks_source_skillset` (`source`,`source_skillset_slug`),
  KEY `idx_skill_distribution_tasks_batch_id` (`batch_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `skill_distribution_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `task_id` bigint unsigned NOT NULL,
  `skill_id` bigint unsigned NOT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `instance_c_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `status` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `error` text COLLATE utf8mb4_unicode_ci,
  `type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'distribute',
  PRIMARY KEY (`id`),
  KEY `idx_skill_distribution_records_identifier` (`identifier`),
  KEY `idx_skill_distribution_records_deleted_at` (`deleted_at`),
  KEY `idx_skill_distribution_records_task_id` (`task_id`),
  KEY `idx_skill_distribution_records_skill_id` (`skill_id`),
  KEY `idx_skill_distribution_records_instance_id` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `role_distribution_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `instance_cid` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `role_id` bigint unsigned NOT NULL,
  `role_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `operator_id` bigint unsigned NOT NULL DEFAULT 0,
  `source` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'updating' COMMENT 'updating/updated/failed/cancelled',
  `soul_status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `soul_error` text COLLATE utf8mb4_unicode_ci,
  `soul_set_at` datetime(3) DEFAULT NULL,
  `skill_status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `skill_error` text COLLATE utf8mb4_unicode_ci,
  `skill_set_at` datetime(3) DEFAULT NULL,
  `skill_installation_ids` varchar(2048) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '本次差集在 skill_installations 的 IDs，JSON 数组',
  PRIMARY KEY (`id`),
  KEY `idx_role_distribution_records_identifier` (`identifier`),
  KEY `idx_role_distribution_records_deleted_at` (`deleted_at`),
  KEY `idx_role_distribution_records_instance_id` (`instance_id`),
  KEY `idx_role_distribution_records_role_id` (`role_id`),
  KEY `idx_role_distribution_records_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `smh_spaces` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `space_tag` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `space_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `library_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `purpose` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `admin_token` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `admin_token_expired_at` bigint NOT NULL DEFAULT 0,
  `read_token` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `read_token_expired_at` bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_space_tag_identifier` (`identifier`,`space_tag`),
  KEY `idx_smh_spaces_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `smh_personal_spaces` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `space_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `user_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `c_vm_instance_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `storage_quota` bigint NOT NULL DEFAULT '0',
  `free_storage_quota` bigint NOT NULL DEFAULT '0',
  `env_initialized` tinyint(1) NOT NULL DEFAULT '0',
  `env_provision_rev` int NOT NULL DEFAULT '0',
  `last_pushed_token_expires_at` datetime(3) DEFAULT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  `to_be_deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_personal_space_instance_identifier` (`identifier`,`instance_id`),
  KEY `idx_smh_personal_spaces_deleted_at` (`deleted_at`),
  KEY `idx_smh_personal_spaces_identifier` (`identifier`),
  KEY `idx_smh_personal_spaces_user_id` (`user_id`),
  KEY `idx_smh_personal_spaces_env_sync` (`env_initialized`,`env_provision_rev`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `notifications` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `user_id` bigint unsigned NOT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `instance_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `category` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'notice',
  `title` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `message` text COLLATE utf8mb4_unicode_ci,
  `error_detail` text COLLATE utf8mb4_unicode_ci,
  `is_read` tinyint(1) DEFAULT '0',
  `read_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_notifications_deleted_at` (`deleted_at`),
  KEY `idx_notifications_identifier` (`identifier`),
  KEY `idx_notifications_user_id` (`user_id`),
  KEY `idx_notifications_instance_id` (`instance_id`),
  KEY `idx_notifications_type` (`type`),
  KEY `idx_notifications_identifier_category` (`identifier`,`category`),
  KEY `idx_notifications_is_read` (`is_read`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `skill_bundles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `skill_count` bigint NOT NULL DEFAULT '0',
  `enabled` tinyint(1) DEFAULT '0',
  `visibility_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  PRIMARY KEY (`id`),
  KEY `idx_skill_bundles_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `bundle_skills` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `skill_bundle_id` bigint unsigned NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'public',
  `source_skillset_slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source_skillset_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cos_zip_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_bundle_skills_identifier` (`identifier`),
  KEY `idx_bundle_skills_skill_bundle_id` (`skill_bundle_id`),
  KEY `idx_bundle_skills_source_skillset_slug` (`source_skillset_slug`),
  KEY `idx_bundle_skills_source_slug_version_bundle` (`source`,`slug`,`version`,`skill_bundle_id`),
  KEY `idx_bundle_skills_source_skillset_bundle` (`source_skillset_slug`,`skill_bundle_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `public_skillsets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_public_skillsets_identifier_slug` (`identifier`, `slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `public_skills` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `name` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `total_downloads` bigint NOT NULL DEFAULT '0',
  `total_favorites` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `idx_public_skills_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `skill_installations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cos_zip_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `install_status` bigint NOT NULL DEFAULT '0',
  `error_message` text COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_skill_installations_identifier` (`identifier`),
  KEY `idx_skill_installations_instance_id` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `memory_tda_iplugins` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `instance_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `status` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'NOT_INSTALLED',
  `last_error` text COLLATE utf8mb4_unicode_ci,
  `retry_count` bigint NOT NULL DEFAULT '0',
  `desired_plan` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'OFF',
  `current_plan` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'OFF',
  `switch_status` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `last_task_id` bigint unsigned NOT NULL DEFAULT '0',
  `last_switched_at` datetime(3) DEFAULT NULL,
  `pool_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `database_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `endpoint` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `api_key_secret_ref` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `vdb_username` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `embedding_model` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `memory_plugin_version` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `offload_enabled` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_memory_tdai_instance_identifier` (`identifier`,`instance_id`),
  KEY `idx_memory_tda_iplugins_deleted_at` (`deleted_at`),
  KEY `idx_memory_tda_iplugins_identifier` (`identifier`),
  KEY `idx_memory_tda_iplugins_status` (`status`),
  KEY `idx_memory_tda_iplugins_current_plan` (`identifier`, `current_plan`),
  KEY `idx_memory_tda_iplugins_switch_status` (`identifier`, `switch_status`),
  KEY `idx_memory_tda_iplugins_pool_database` (`identifier`, `pool_id`, `database_name`),
  KEY `idx_memory_tda_iplugins_instance_id_deleted_at` (`instance_id`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `open_claw_roles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `soul` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `visible` tinyint(1) NOT NULL DEFAULT '1',
  `sort_order` bigint NOT NULL DEFAULT '0',
  `visibility_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  `version` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1.0' COMMENT '角色版本号，X.Y 两段式',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_role_name_identifier` (`identifier`,`name`),
  KEY `idx_open_claw_roles_identifier` (`identifier`),
  KEY `idx_open_claw_roles_sort_order` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `open_claw_role_skills` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `open_claw_role_id` bigint unsigned NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'public',
  `cos_zip_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_open_claw_role_skills_identifier` (`identifier`),
  KEY `idx_open_claw_role_skills_open_claw_role_id` (`open_claw_role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `tdai_jobs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `job_type` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `biz_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `instance_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `state` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'PENDING',
  `current_step` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `progress` int NOT NULL DEFAULT 0,
  `run_at` datetime(3) NOT NULL,
  `attempt` int NOT NULL DEFAULT 0,
  `max_attempts` int NOT NULL DEFAULT 3,
  `lease_owner` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `lease_until` datetime(3) DEFAULT NULL,
  `payload_json` text COLLATE utf8mb4_unicode_ci,
  `result_json` text COLLATE utf8mb4_unicode_ci,
  `last_error` text COLLATE utf8mb4_unicode_ci,
  `error_code` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `operator` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `trace_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `finished_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tdai_jobs_deleted_at` (`deleted_at`),
  KEY `idx_tdai_jobs_identifier` (`identifier`),
  KEY `idx_tdai_jobs_job_type` (`job_type`),
  KEY `idx_tdai_jobs_identifier_biz_key` (`identifier`, `biz_key`),
  KEY `idx_tdai_jobs_state_run_at` (`state`, `run_at`),
  KEY `idx_tdai_jobs_instance_state` (`instance_id`, `state`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ===== 企业插件库相关表 =====

CREATE TABLE IF NOT EXISTS `plugins` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1.0.0',
  `version_major` bigint NOT NULL DEFAULT '0',
  `version_minor` bigint NOT NULL DEFAULT '0',
  `version_patch` bigint NOT NULL DEFAULT '0',
  `plugin_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `plugin_format` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'openclaw',
  `kind` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cos_zip_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cos_dir_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `file_list` text COLLATE utf8mb4_unicode_ci,
  `file_size` bigint NOT NULL DEFAULT '0',
  `npm_package` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `config_schema` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `providers` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `channels` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `changelog` varchar(10000) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `distribute_count` bigint NOT NULL DEFAULT '0',
  `visibility_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_plugin_slug_version_identifier` (`slug`,`version`,`identifier`),
  KEY `idx_plugins_identifier` (`identifier`),
  KEY `idx_plugins_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_distribution_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  `plugin_db_id` bigint unsigned NOT NULL,
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `operator_id` bigint unsigned NOT NULL DEFAULT '0',
  `total` bigint NOT NULL DEFAULT '0',
  `success` bigint NOT NULL DEFAULT '0',
  `failed` bigint NOT NULL DEFAULT '0',
  `status` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'running',
  `type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'distribute',
  PRIMARY KEY (`id`),
  KEY `idx_pdt_identifier` (`identifier`),
  KEY `idx_pdt_plugin_db_id` (`plugin_db_id`),
  KEY `idx_pdt_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_distribution_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  `task_id` bigint unsigned NOT NULL,
  `plugin_db_id` bigint unsigned NOT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `instance_c_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `status` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `error` text COLLATE utf8mb4_unicode_ci,
  `type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'distribute',
  PRIMARY KEY (`id`),
  KEY `idx_pdr_identifier` (`identifier`),
  KEY `idx_pdr_task_id` (`task_id`),
  KEY `idx_pdr_plugin_db_id` (`plugin_db_id`),
  KEY `idx_pdr_instance_id` (`instance_id`),
  KEY `idx_pdr_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_visibility_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `plugin_id` bigint unsigned NOT NULL,
  `group_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_pvg_unique` (`identifier`,`plugin_id`,`group_id`),
  KEY `idx_pvg_plugin_id` (`plugin_id`),
  KEY `idx_pvg_group_id` (`group_id`),
  KEY `idx_pvg_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_categories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_plugin_cat_name_identifier` (`name`,`identifier`),
  KEY `idx_plugin_categories_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_category_mappings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `plugin_id` bigint unsigned NOT NULL,
  `category_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_plugin_cat_map` (`plugin_id`,`category_id`,`identifier`),
  KEY `idx_pcm_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_bundles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `plugin_count` bigint NOT NULL DEFAULT '0',
  `enabled` tinyint(1) NOT NULL DEFAULT '0',
  `visibility_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  PRIMARY KEY (`id`),
  KEY `idx_plugin_bundles_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `bundle_plugins` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `plugin_bundle_id` bigint unsigned NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `plugin_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'enterprise',
  `cos_zip_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `npm_package` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `install_mode` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'smh',
  `kind` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_bundle_plugins_identifier` (`identifier`),
  KEY `idx_bundle_plugins_bundle_id` (`plugin_bundle_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `public_plugins` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `plugin_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `npm_package` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `total_downloads` bigint NOT NULL DEFAULT '0',
  `total_favorites` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_public_plugin_slug` (`slug`,`identifier`),
  KEY `idx_public_plugins_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_installations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `instance_id` bigint unsigned NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `plugin_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cos_zip_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `npm_package` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `install_mode` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'smh',
  `kind` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `install_status` bigint NOT NULL DEFAULT '0',
  `error_message` text COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_plugin_inst_slug` (`instance_id`,`slug`,`identifier`),
  KEY `idx_plugin_installations_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `open_claw_role_plugins` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `open_claw_role_id` bigint unsigned NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `plugin_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'enterprise',
  `cos_zip_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `npm_package` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `install_mode` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'smh',
  `kind` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_ocrp_identifier` (`identifier`),
  KEY `idx_ocrp_role_id` (`open_claw_role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户组（🆕 v6.12 P1: 扩展 parent_id/depth/full_path/source/source_ref/to_be_deleted）
-- 不使用 GORM 软删（无 deleted_at 列）：删除走物理删除，同事务级联清理
-- user_group_members + group_closure。"待删占位"语义由 to_be_deleted 字段承担。
-- 独立资源策略：策略内容存于本表，应用分组复用 group_config_bindings 的资源绑定语义。
CREATE TABLE IF NOT EXISTS `resource_policies` (
  `id`          bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`  varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name`        varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_default`  tinyint(1) NOT NULL DEFAULT 0,
  `config_json` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at`  datetime(3) DEFAULT NULL,
  `updated_at`  datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_rp_ident_name` (`identifier`, `name`),
  KEY `idx_rp_ident_default` (`identifier`, `is_default`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `user_groups` (
  `id`            bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`    varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name`          varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description`   text         COLLATE utf8mb4_unicode_ci NOT NULL,
  `sync_mode`     varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'continuous' COMMENT '同步模式：continuous（持续同步）/ initial_only（仅初始）',
  `parent_id`     bigint unsigned NOT NULL DEFAULT 0,               -- 🆕 v6.15: uint，0 = 根组
  `depth`         int NOT NULL DEFAULT 0,                           -- 根=0，最深 9
  `full_path`     varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source`        varchar(32)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'manual',
  `source_ref`    varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `to_be_deleted` tinyint(1)   NOT NULL DEFAULT 0,                   -- 🆕 v6: OneID 部门已消失但本地有资源绑定，暂保留只读
  `created_at`    datetime(3) DEFAULT NULL,
  `updated_at`    datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ug_ident_parent_name` (`identifier`, `parent_id`, `name`),
  KEY `idx_ug_parent`        (`parent_id`),
  KEY `idx_ug_source`        (`identifier`, `source`, `source_ref`),
  KEY `idx_ug_fullpath`      (`identifier`, `full_path`),
  KEY `idx_ug_tobedeleted`   (`identifier`, `to_be_deleted`),
  KEY `idx_ug_depth`         (`depth`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户组成员关联（🆕 v6.12 P1: 扩展 is_main/source）
CREATE TABLE IF NOT EXISTS `user_group_members` (
  `id`            bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`    varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `user_group_id` bigint unsigned NOT NULL,
  `user_id`       bigint unsigned NOT NULL,
  `is_main`       tinyint(1) NOT NULL DEFAULT 0,                     -- 🆕 v6: OneID 主部门标记
  `source`        varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'manual',  -- 🆕 v6.16: manual / oneid_dept
  `created_at`    datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ugm_identifier_group_user` (`identifier`,`user_group_id`,`user_id`),
  KEY `idx_user_group_members_user_group_id` (`user_group_id`),
  KEY `idx_user_group_members_user_id`       (`user_id`),
  KEY `idx_user_group_members_is_main`       (`is_main`),
  KEY `idx_user_group_members_source`        (`source`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 🆕 v6.12 P1: 分组闭包表，物化祖先-后代关系
CREATE TABLE IF NOT EXISTS `group_closure` (
  `identifier`    varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ancestor_id`   bigint unsigned NOT NULL,
  `descendant_id` bigint unsigned NOT NULL,
  `depth`         int NOT NULL,
  PRIMARY KEY (`identifier`, `ancestor_id`, `descendant_id`),
  KEY `idx_gc_desc` (`identifier`, `descendant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `group_config_bindings` (
  `id`           bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`   varchar(191) NOT NULL DEFAULT '',
  `config_type`  varchar(32)  NOT NULL,
  `config_key`   varchar(128) NOT NULL,
  `group_id`     bigint unsigned NOT NULL,
  `value_json`   varchar(4096) NOT NULL DEFAULT '{}',
  `created_at`   datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`   datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_gcb` (`identifier`, `config_type`, `config_key`, `group_id`),
  KEY `idx_gcb_group` (`identifier`, `group_id`, `config_type`),
  KEY `idx_gcb_resource` (`identifier`, `config_type`, `config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- VPC 配置资源表
CREATE TABLE IF NOT EXISTS `vpc_configs` (
  `id`              bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at`      datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`      datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at`      datetime(3) DEFAULT NULL,
  `identifier`      varchar(191) NOT NULL DEFAULT '',
  `vpc_id`          varchar(64) NOT NULL,
  `subnet_ids`      text NOT NULL,
  `visibility_type` varchar(16) NOT NULL DEFAULT 'all',
  `strategy_name`   varchar(20) DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_vpc_configs_identifier` (`identifier`),
  KEY `idx_vpc_configs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 标签应用范围
CREATE TABLE IF NOT EXISTS `tags` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `tag_key` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `tag_value` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visibility_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_tags_key_value` (`identifier`,`tag_key`,`tag_value`),
  KEY `idx_tags_visibility` (`identifier`,`visibility_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `tag_visibility_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `tag_id` bigint unsigned NOT NULL,
  `group_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_tvg_unique` (`identifier`,`tag_id`,`group_id`),
  KEY `idx_tag_visibility_groups_identifier` (`identifier`),
  KEY `idx_tag_visibility_groups_tag_id` (`tag_id`),
  KEY `idx_tag_visibility_groups_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 模型可见性-分组关联（补全）
CREATE TABLE IF NOT EXISTS `model_visibility_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `ai_model_id` bigint unsigned NOT NULL,
  `group_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mvg_unique` (`identifier`,`ai_model_id`,`group_id`),
  KEY `idx_model_visibility_groups_identifier` (`identifier`),
  KEY `idx_model_visibility_groups_ai_model_id` (`ai_model_id`),
  KEY `idx_model_visibility_groups_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 技能可见性-分组关联
CREATE TABLE IF NOT EXISTS `skill_visibility_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `skill_id` bigint unsigned NOT NULL,
  `group_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_svg_unique` (`identifier`,`skill_id`,`group_id`),
  KEY `idx_skill_visibility_groups_identifier` (`identifier`),
  KEY `idx_skill_visibility_groups_skill_id` (`skill_id`),
  KEY `idx_skill_visibility_groups_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 技能包可见性-分组关联
CREATE TABLE IF NOT EXISTS `skill_bundle_visibility_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `skill_bundle_id` bigint unsigned NOT NULL,
  `group_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sbvg_unique` (`identifier`,`skill_bundle_id`,`group_id`),
  KEY `idx_sbvg_identifier` (`identifier`),
  KEY `idx_sbvg_skill_bundle_id` (`skill_bundle_id`),
  KEY `idx_sbvg_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色可见性-分组关联
CREATE TABLE IF NOT EXISTS `role_visibility_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `open_claw_role_id` bigint unsigned NOT NULL,
  `group_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_rvg_unique` (`identifier`,`open_claw_role_id`,`group_id`),
  KEY `idx_rvg_identifier` (`identifier`),
  KEY `idx_rvg_role_id` (`open_claw_role_id`),
  KEY `idx_rvg_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ========== 企业 MCP 库（5 张表）==========

-- MCP 主表
CREATE TABLE IF NOT EXISTS `mcp_servers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `service_id` varchar(128) NOT NULL,
  `name` varchar(128) NOT NULL,
  `description` varchar(1024) DEFAULT NULL,
  `transport_type` varchar(32) NOT NULL,
  `latest_version_id` bigint unsigned DEFAULT 0,
  `created_by` varchar(128) NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `visibility_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all',
  `key_hosted` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否存在托管字段：0=无，1=有',
  `ip_whitelist` varchar(2048) NOT NULL DEFAULT '' COMMENT 'IP白名单（逗号分隔），空=不限制',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_identifier_service_id` (`identifier`, `service_id`),
  KEY `idx_mcp_servers_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MCP 版本表
CREATE TABLE IF NOT EXISTS `mcp_versions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `mcp_id` bigint unsigned NOT NULL,
  `version` varchar(32) NOT NULL,
  `transport_type` varchar(32) NOT NULL,
  `config_json` text NOT NULL,
  `usage_doc_md` text,
  `tool_doc_md` text,
  `created_at` datetime(3) DEFAULT NULL,
  `created_by` varchar(128) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_mcp_version` (`identifier`, `mcp_id`, `version`),
  KEY `idx_mcp_versions_identifier` (`identifier`),
  KEY `idx_mcp_versions_mcp_id` (`mcp_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MCP 批次任务表
CREATE TABLE IF NOT EXISTS `mcp_distribution_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `mcp_id` bigint unsigned NOT NULL,
  `mcp_snapshot_service_id` varchar(128) NOT NULL,
  `mcp_snapshot_name` varchar(128) NOT NULL,
  `version_id` bigint unsigned DEFAULT 0,
  `version_snapshot` varchar(32) NOT NULL,
  `operator_id` bigint unsigned NOT NULL DEFAULT 0,
  `total` int NOT NULL DEFAULT 0,
  `success` int NOT NULL DEFAULT 0,
  `failed` int NOT NULL DEFAULT 0,
  `status` varchar(16) NOT NULL DEFAULT 'running',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_mcp_distribution_tasks_identifier` (`identifier`),
  KEY `idx_mcp_distribution_tasks_mcp_id` (`mcp_id`),
  KEY `idx_mcp_distribution_tasks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MCP 逐实例下发记录
CREATE TABLE IF NOT EXISTS `mcp_distribution_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `task_id` bigint unsigned NOT NULL,
  `mcp_id` bigint unsigned DEFAULT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `instance_c_id` varchar(64) NOT NULL DEFAULT '',
  `version_snapshot` varchar(32) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `error` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_task_instance` (`identifier`, `task_id`, `instance_id`),
  KEY `idx_mcp_distribution_records_identifier` (`identifier`),
  KEY `idx_mcp_distribution_records_task_id` (`task_id`),
  KEY `idx_mcp_distribution_records_mcp_id` (`mcp_id`),
  KEY `idx_mcp_distribution_records_instance_id` (`instance_id`),
  KEY `idx_mcp_distribution_records_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MCP 实例当前状态表
CREATE TABLE IF NOT EXISTS `mcp_installations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `mcp_id` bigint unsigned DEFAULT NULL,
  `service_id` varchar(128) NOT NULL,
  `name` varchar(128) NOT NULL DEFAULT '',
  `version` varchar(32) NOT NULL DEFAULT '',
  `install_status` int NOT NULL DEFAULT 0,
  `last_task_id` bigint unsigned DEFAULT 0,
  `error_message` varchar(2048) NOT NULL DEFAULT '',
  `config_json` text,
  `source` varchar(16) NOT NULL DEFAULT 'admin',
  `connection_status` varchar(16) NOT NULL DEFAULT '',
  `tools_json` text,
  `connection_error` varchar(1024) NOT NULL DEFAULT '',
  `probed_at` datetime DEFAULT NULL,
  `hosted_values` text COMMENT '实例级托管字段值 JSON: {"Authorization":"Bearer xxx"}',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_instance_service` (`identifier`, `instance_id`, `service_id`),
  KEY `idx_mcp_installations_identifier` (`identifier`),
  KEY `idx_mcp_installations_instance_id` (`identifier`, `instance_id`),
  KEY `idx_mcp_installations_mcp_id` (`mcp_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MCP 凭据托管字段定义表
CREATE TABLE IF NOT EXISTS `mcp_hosted_keys` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) NOT NULL DEFAULT '' COMMENT '多租户标识',
  `mcp_id` bigint unsigned NOT NULL COMMENT '关联 mcp_servers.id',
  `key` varchar(128) NOT NULL COMMENT '托管的占位符 key',
  `placeholder` varchar(256) NOT NULL DEFAULT '' COMMENT '原始占位符值，如 <your-token>',
  `default_value` varchar(1024) NOT NULL DEFAULT '' COMMENT '管理员给的默认值（可为空）',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_mcp_key` (`identifier`, `mcp_id`, `key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 龙虾医生诊断会话表
CREATE TABLE IF NOT EXISTS `doctor_sessions` (
  `id`                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `created_at`          DATETIME(3)     NOT NULL,
  `updated_at`          DATETIME(3)     NOT NULL,
  `deleted_at`          DATETIME(3)     DEFAULT NULL,
  `identifier`          VARCHAR(191)    NOT NULL DEFAULT '',
  `user_id`             BIGINT UNSIGNED NOT NULL COMMENT '发起诊断的用户 ID',
  `target_instance_id`  BIGINT UNSIGNED NOT NULL COMMENT '被诊断的目标实例 DB ID',
  `doctor_instance_id`  BIGINT UNSIGNED DEFAULT NULL COMMENT '临时诊断节点 DB ID（创建成功后填充）',
  `status`              VARCHAR(16)     NOT NULL DEFAULT 'creating'
      COMMENT 'creating | active | ending | ended | failed',
  `activated_at`        DATETIME(3)     DEFAULT NULL COMMENT '诊断会话进入 active 状态的时间',
  `snapshot_requested`  TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '是否请求在激活后自动创建快照',
  `has_snapshot`        TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '本次诊断是否已创建快照',
  `snapshot_file_key`   VARCHAR(512)    DEFAULT '' COMMENT 'SMH 备份文件的 fileKey',
  `snapshot_deleted`    TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '快照是否已从 SMH 删除',
  `sessions_deleted`    TINYINT(1)      NOT NULL DEFAULT 0 COMMENT 'session 对话备份是否已从 SMH 删除',
  `rollback_requested`  TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '结束时是否请求回滚',
  `sts_expired_at`      BIGINT          NOT NULL DEFAULT 0 COMMENT 'STS 临时密钥过期时间（Unix 秒）',
  INDEX `idx_doctor_sessions_identifier` (`identifier`),
  INDEX `idx_doctor_sessions_user_id` (`user_id`),
  INDEX `idx_doctor_sessions_target_instance_id` (`target_instance_id`),
  INDEX `idx_doctor_sessions_status` (`status`),
  INDEX `idx_doctor_sessions_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 龙虾医生授权记录表
CREATE TABLE IF NOT EXISTS `doctor_authorizations` (
  `id`          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `created_at`  DATETIME(3)     DEFAULT NULL,
  `updated_at`  DATETIME(3)     DEFAULT NULL,
  `deleted_at`  DATETIME(3)     DEFAULT NULL,
  `identifier`  VARCHAR(191)    NOT NULL DEFAULT '',
  `user_id`     BIGINT UNSIGNED NOT NULL,
  `instance_id` BIGINT UNSIGNED NOT NULL,
  UNIQUE KEY `idx_auth_user_instance` (`identifier`, `user_id`, `instance_id`),
  INDEX `idx_doctor_authorizations_identifier` (`identifier`),
  INDEX `idx_doctor_authorizations_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 实例与 AI 模型的绑定关系（多模型 Fallback）
CREATE TABLE IF NOT EXISTS `instance_models` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `ai_model_id` bigint unsigned NOT NULL DEFAULT '0',
  `custom_model_id` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `role` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'fallback',
  `sort_order` int NOT NULL DEFAULT '0',
  `custom_model_config` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_instance_model` (`identifier`,`instance_id`,`ai_model_id`,`custom_model_id`),
  KEY `idx_instance_models_identifier` (`identifier`),
  KEY `idx_instance_id` (`instance_id`),
  KEY `idx_instance_models_deleted_at` (`deleted_at`),
  KEY `idx_instance_models_sort_order` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `agent_migrations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `cvm_instance_id` varchar(64) NOT NULL DEFAULT '',
  `file_key` varchar(255) NOT NULL DEFAULT '',
  `status` varchar(32) NOT NULL DEFAULT 'pending_upload',
  `fail_reason` text,
  `steps_json` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_agent_migrations_identifier` (`identifier`),
  KEY `idx_agent_migrations_instance_id` (`instance_id`),
  KEY `idx_agent_migrations_status` (`status`),
  KEY `idx_agent_migrations_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- sg-ruleset-projection 方案的三张核心表（详见 openspec archive 2026-05-04-sg-ruleset-projection）：
--   rule_sets       —— 规则真相源，每租户一行 name='default'
--   managed_sg_pool —— ClawPro 管理的云端 SG 列表，按 rule_set_id 分区 + 三态状态机（ACTIVE/FROZEN/DRAINING）
--   sg_drain_state  —— DrainWorker 失败计数与 drain_stuck 标记（稀疏表，仅失败实例有行）
CREATE TABLE IF NOT EXISTS `rule_sets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'clawpro-default',
  `description` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '管理员自定义备注，UI 展示用',
  `rules` text COLLATE utf8mb4_unicode_ci,
  `version` int NOT NULL DEFAULT 1,
  `user_group_ids` text COLLATE utf8mb4_unicode_ci COMMENT '预留：未来多 RuleSet 按用户组路由时用，本期恒为 "[]"',
  `is_default` tinyint(1) NOT NULL DEFAULT 1 COMMENT '预留：默认规则组标记，本期单一 RuleSet 恒为 1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_rs_ident_name` (`identifier`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `managed_sg_pool` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `sg_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sg_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Guardian 从云 API 同步的 SG 实际名称',
  `rule_set_id` bigint unsigned NOT NULL,
  `rule_version` int NOT NULL DEFAULT 0,
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'ACTIVE',
  `cvm_count` int NOT NULL DEFAULT 0,
  `cvm_count_at` datetime(3) DEFAULT NULL,
  `drained_at` datetime(3) DEFAULT NULL,
  `drift_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_managed_sg_pool_sg_id` (`sg_id`),
  KEY `idx_managed_sg_pool_identifier` (`identifier`),
  KEY `idx_managed_sg_pool_rs_status` (`rule_set_id`, `status`),
  KEY `idx_managed_sg_pool_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `sg_drain_state` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `instance_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `fail_count` int NOT NULL DEFAULT 0,
  `stuck_at` datetime(3) DEFAULT NULL,
  `last_error` text COLLATE utf8mb4_unicode_ci,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_sg_drain_state_instance_id` (`instance_id`),
  KEY `idx_sg_drain_state_identifier` (`identifier`),
  KEY `idx_sg_drain_state_stuck` (`stuck_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ========================
-- Skill 安全检测
-- ========================

CREATE TABLE IF NOT EXISTS `skill_security_scans` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `skill_id` bigint unsigned NOT NULL,
  `skill_version` varchar(50) NOT NULL DEFAULT '',
  `content_hash` varchar(255) NOT NULL,
  `engine_version` int NOT NULL DEFAULT '0',
  `status` varchar(50) NOT NULL DEFAULT 'SCANNING',
  `risk_level` varchar(50) NOT NULL DEFAULT '',
  `primary_rule_id` varchar(50) NOT NULL DEFAULT '',
  `security_score` int NOT NULL DEFAULT '100',
  `scan_result_data` json DEFAULT NULL,
  `report_url` varchar(2048) NOT NULL DEFAULT '',
  `scanned_at` datetime(3) DEFAULT NULL,
  `failed_at` datetime(3) DEFAULT NULL,
  `failure_message` text,
  PRIMARY KEY (`id`),
  KEY `idx_identifier` (`identifier`),
  KEY `idx_skill_id` (`skill_id`),
  KEY `idx_status` (`status`),
  UNIQUE KEY `idx_hash_engine` (`identifier`,`content_hash`,`engine_version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `skill_scan_violations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `skill_security_scan_id` bigint unsigned NOT NULL,
  `rule_id` varchar(50) NOT NULL DEFAULT '',
  `rule_name` varchar(191) NOT NULL DEFAULT '',
  `scan_type` varchar(50) NOT NULL DEFAULT '',
  `description` text NOT NULL,
  `capability_tag` varchar(50) NOT NULL DEFAULT '',
  `capability_tag_name` varchar(191) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_identifier` (`identifier`),
  KEY `idx_scan_id` (`skill_security_scan_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ========================================================================
-- Agent 命令执行 (feature/agent_command_execution)
-- 详见: openspec/changes/agent-command-execution/design.md §1
-- ========================================================================

CREATE TABLE IF NOT EXISTS `agent_commands` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `description` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'SHELL',
  `content` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `timeout_sec` int unsigned NOT NULL DEFAULT '60',
  `run_user` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'root',
  `workdir` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '/root',
  `params_json` varchar(8192) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '[]',
  `visibility_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'tenant',
  `created_by_user_id` bigint unsigned NOT NULL DEFAULT '0',
  -- 生成列：仅在「未软删」时承载 name，否则为 NULL，配合下方 UNIQUE 实现「同租户、未删唯一」语义。
  -- 软删后该行的 name_active 自动变 NULL，新行的 name_active=name 与 NULL 不冲突 → 名称可重用。
  -- InnoDB 唯一索引中多个 NULL 算独立值，所以已删行不互相冲突。
  `name_active` varchar(60) COLLATE utf8mb4_unicode_ci
    GENERATED ALWAYS AS (CASE WHEN `deleted_at` IS NULL THEN `name` ELSE NULL END) STORED,
  PRIMARY KEY (`id`),
  KEY `idx_agent_commands_deleted_at` (`deleted_at`),
  KEY `idx_agent_commands_created_by_user_id` (`created_by_user_id`),
  UNIQUE KEY `idx_agent_command_ident_slug` (`identifier`,`slug`),
  UNIQUE KEY `idx_agent_command_ident_name_active` (`identifier`,`name_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `agent_command_dispatch` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `command_id` bigint unsigned NOT NULL DEFAULT '0',
  `command_snapshot` text COLLATE utf8mb4_unicode_ci,
  `param_values_json` varchar(4096) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '{}',
  `triggered_by_user_id` bigint unsigned NOT NULL DEFAULT '0',
  `test_first` tinyint(1) NOT NULL DEFAULT '0',
  `test_target_instance_id` bigint unsigned NOT NULL DEFAULT '0',
  `target_count` int unsigned NOT NULL DEFAULT '0',
  `success_count` int unsigned NOT NULL DEFAULT '0',
  `failed_count` int unsigned NOT NULL DEFAULT '0',
  `cancelled_count` int unsigned NOT NULL DEFAULT '0',
  `status` varchar(24) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'in_progress',
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_dispatch_ident_slug` (`identifier`,`slug`),
  KEY `idx_dispatch_identifier` (`identifier`),
  KEY `idx_dispatch_command_id` (`command_id`),
  KEY `idx_dispatch_triggered_by` (`triggered_by_user_id`),
  KEY `idx_dispatch_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `agent_command_invocations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `tat_invocation_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `dispatch_id` bigint unsigned NOT NULL DEFAULT '0',
  `dispatch_slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `is_test_run` tinyint(1) NOT NULL DEFAULT '0',
  `batch_index` int unsigned NOT NULL DEFAULT '0',
  `target_count` int unsigned NOT NULL DEFAULT '0',
  `success_count` int unsigned NOT NULL DEFAULT '0',
  `failed_count` int unsigned NOT NULL DEFAULT '0',
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_invocations_identifier` (`identifier`),
  KEY `idx_invocations_tat_id` (`tat_invocation_id`),
  KEY `idx_invocations_dispatch_id` (`dispatch_id`),
  KEY `idx_invocations_dispatch_slug` (`dispatch_slug`),
  KEY `idx_invocations_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `agent_command_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `tat_invocation_task_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `dispatch_id` bigint unsigned NOT NULL DEFAULT '0',
  `invocation_id` bigint unsigned NOT NULL DEFAULT '0',
  `dispatch_slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_id` bigint unsigned NOT NULL DEFAULT '0',
  `cvm_instance_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `agent_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `owner_username` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `is_test_target` tinyint(1) NOT NULL DEFAULT '0',
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `exit_code` int DEFAULT NULL,
  `elapsed_ms` int unsigned DEFAULT NULL,
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tasks_identifier` (`identifier`),
  KEY `idx_tasks_tat_task_id` (`tat_invocation_task_id`),
  KEY `idx_tasks_dispatch_id` (`dispatch_id`),
  KEY `idx_tasks_invocation_id` (`invocation_id`),
  KEY `idx_tasks_dispatch_slug` (`dispatch_slug`),
  KEY `idx_tasks_instance_id` (`instance_id`),
  KEY `idx_tasks_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `memory_plan_group_policies` (
    `id`         INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `group_id`   INT UNSIGNED NOT NULL COMMENT '分组 ID（对应 user_groups.id）',
    `plan`       VARCHAR(16) NOT NULL COMMENT '记忆版本：off / free / pro',
    `priority`   TINYINT NOT NULL DEFAULT 1 COMMENT '策略优先级：1=第一条策略, 2=第二条策略',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `idx_mpgp_group` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 多租户阶段二：域名→租户映射（全局表）
CREATE TABLE IF NOT EXISTS `tenant_domains` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `domain` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '完整域名，如 a.tcaisite.com',
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '租户标识',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_tenant_domains_domain` (`domain`),
  KEY `idx_tenant_domains_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Agent 命令定时任务配置（迁移：0630-agent-command-schedules.sql）
CREATE TABLE IF NOT EXISTS `agent_command_schedules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `description` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `command_id` bigint unsigned NOT NULL DEFAULT '0',
  `instance_ids_json` varchar(8192) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '[]',
  `param_values_json` varchar(4096) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '{}',
  `schedule_expr` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `next_run_at` datetime(3) DEFAULT NULL,
  `first_run_at` datetime(3) DEFAULT NULL,
  `last_run_at` datetime(3) DEFAULT NULL,
  `last_dispatch_slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `is_running` tinyint(1) NOT NULL DEFAULT '0',
  `last_error` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_by_user_id` bigint unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sched_ident_slug` (`identifier`,`slug`),
  KEY `idx_sched_deleted_at` (`deleted_at`),
  KEY `idx_sched_identifier` (`identifier`),
  KEY `idx_sched_command_id` (`command_id`),
  KEY `idx_sched_created_by` (`created_by_user_id`),
  KEY `idx_sched_due` (`enabled`,`next_run_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Agent 命令定时任务历史执行记录（迁移：0630-agent-command-schedules.sql）
CREATE TABLE IF NOT EXISTS `agent_command_schedule_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `schedule_id` bigint unsigned NOT NULL DEFAULT '0',
  `dispatch_slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_sched_record_identifier` (`identifier`),
  KEY `idx_sched_record_sched` (`schedule_id`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 功能白名单（全局表，跨租户，按 type 分样）
-- 语义：某 type 下无记录 = 该功能未启用白名单，全部租户放行；有记录 = 仅表内 identifier 放行。
CREATE TABLE IF NOT EXISTS `feature_allowlists` (
                                                    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                                    `type` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '功能类别，如 local-agent',
                                                    `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '租户标识',
                                                    `note` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '备注',
                                                    `created_at` datetime(3) DEFAULT NULL,
                                                    PRIMARY KEY (`id`),
                                                    UNIQUE KEY `idx_feature_allowlist_type_identifier` (`type`, `identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 本地 agent（clawpro 一期）：实例扩展信息
CREATE TABLE IF NOT EXISTS `local_instance_infos` (
                                                      `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                                      `created_at` datetime(3) DEFAULT NULL,
                                                      `updated_at` datetime(3) DEFAULT NULL,
                                                      `deleted_at` datetime(3) DEFAULT NULL,
                                                      `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
                                                      `instance_id` bigint unsigned NOT NULL COMMENT '关联 instances.id',
                                                      `host_name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
                                                      `os` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
                                                      `started_at` datetime(3) DEFAULT NULL COMMENT 'reporter 上报的进程启动时间',
                                                      `last_report_at` datetime(3) DEFAULT NULL COMMENT '最近一次 report/sync 上报时间',
                                                      `last_status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'reporter 端派生的运行状态文案',
                                                      PRIMARY KEY (`id`),
                                                      UNIQUE KEY `idx_local_instance_infos_instance_id` (`instance_id`),
                                                      KEY `idx_local_instance_infos_deleted_at` (`deleted_at`),
                                                      KEY `idx_local_instance_infos_identifier` (`identifier`),
                                                      KEY `idx_local_instance_infos_last_report_at` (`last_report_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 本地 agent（clawpro 一期/二期）：已安装 skill 快照
CREATE TABLE IF NOT EXISTS `local_instance_skills` (
                                                       `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                                       `created_at` datetime(3) DEFAULT NULL,
                                                       `updated_at` datetime(3) DEFAULT NULL,
                                                       `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
                                                       `instance_id` bigint unsigned NOT NULL COMMENT '关联 instances.id',
                                                       `slug` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
                                                       `version` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
                                                       `display_name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
                                                       `source` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'local' COMMENT 'public / enterprise / local',
                                                       `installed_at` datetime(3) DEFAULT NULL COMMENT '最近一次 ack success / report 上报已装的时刻',
                                                       `last_seen_at` datetime(3) DEFAULT NULL COMMENT 'report 中最后一次出现的时刻',
                                                       `scope` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'user' COMMENT '作用域：user（用户级）/ workspace（项目级）',
                                                       `workspace_path` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '项目级独有：workspace 路径（scope=workspace 时存）',
                                                       `install_status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'distributed' COMMENT '安装状态：distributing / distributed / failed',
                                                       PRIMARY KEY (`id`),
                                                       UNIQUE KEY `idx_lis_scope_inst_ws_slug` (`scope`,`instance_id`,`workspace_path`,`slug`),
                                                       KEY `idx_local_instance_skills_identifier` (`identifier`),
                                                       KEY `idx_local_instance_skills_source` (`source`),
                                                       KEY `idx_local_instance_skills_last_seen_at` (`last_seen_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 本地 agent（clawpro 一期）：CLS 公网上报凭据（按租户隔离，明文存储，运维按租户写入）
CREATE TABLE IF NOT EXISTS `local_agent_cls_credentials` (
                                                      `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                                      `created_at` datetime(3) DEFAULT NULL,
                                                      `updated_at` datetime(3) DEFAULT NULL,
                                                      `deleted_at` datetime(3) DEFAULT NULL,
                                                      `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
                                                      `config_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'cls',
                                                      `secret_id` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
                                                      `secret_key` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL,
                                                      PRIMARY KEY (`id`),
                                                      UNIQUE KEY `idx_lacc_ident_type` (`identifier`,`config_type`),
                                                      KEY `idx_lacc_identifier` (`identifier`),
                                                      KEY `idx_lacc_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 企业规范库（本地 agent 二期）：4 张新表
create table if not exists `enterprise_rules` (
  `id` bigint unsigned not null auto_increment,
  `identifier` varchar(191) collate utf8mb4_unicode_ci default '',
  `created_at` datetime(3) default null,
  `updated_at` datetime(3) default null,
  `deleted_at` datetime(3) default null,
  `slug` varchar(191) collate utf8mb4_unicode_ci not null,
  `name` varchar(191) collate utf8mb4_unicode_ci not null,
  `description` text collate utf8mb4_unicode_ci not null,
  `type` varchar(16) collate utf8mb4_unicode_ci not null comment 'prompt / rule',
  `source` varchar(16) collate utf8mb4_unicode_ci not null default 'enterprise' comment 'enterprise / local',
  `version` varchar(191) collate utf8mb4_unicode_ci not null default '1.0.0',
  `version_major` bigint not null default '0',
  `version_minor` bigint not null default '0',
  `version_patch` bigint not null default '0',
  `cos_key` varchar(191) collate utf8mb4_unicode_ci not null default '',
  `file_size` bigint not null default '0',
  `content_sha256` varchar(64) collate utf8mb4_unicode_ci not null default '',
  `visibility_type` varchar(191) collate utf8mb4_unicode_ci not null default 'all' comment 'all / group',
  `changelog` varchar(10000) collate utf8mb4_unicode_ci not null default '' comment '版本更新说明',
  `distribute_count` bigint not null default '0',
  `event` varchar(32) collate utf8mb4_unicode_ci not null default '' comment 'Hook 触发时机：SessionStart / UserPromptSubmit / PreToolUse / PostToolUse / Stop',
  `cmd` text collate utf8mb4_unicode_ci comment 'Hook 执行命令（type=hook 时有效）',
  primary key (`id`),
  unique key `idx_enterprise_rules_slug_ver_ident` (`identifier`, `slug`, `version`),
  key `idx_enterprise_rules_identifier` (`identifier`),
  key `idx_enterprise_rules_deleted_at` (`deleted_at`),
  key `idx_enterprise_rules_type` (`type`),
  key `idx_enterprise_rules_source` (`source`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_unicode_ci;

-- 存量实例分组归属处理：实例标记表与处理记录表。
CREATE TABLE IF NOT EXISTS `instance_flags` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `flag` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `extra` varchar(1024) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '{}',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_instance_flags_inst_flag` (`identifier`,`instance_id`,`flag`),
  KEY `idx_instance_flags_flag_lookup` (`identifier`,`flag`,`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `instance_change_group_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_pk` bigint unsigned NOT NULL COMMENT 'instances.id',
  `instance_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'CVM ins-xxx，对齐 instances.instance_id 类型，便于运维',
  `user_id_before` bigint unsigned NOT NULL DEFAULT '0',
  `user_id_after` bigint unsigned NOT NULL DEFAULT '0',
  `group_id_before` bigint unsigned NOT NULL DEFAULT '0',
  `group_id_after` bigint unsigned NOT NULL DEFAULT '0',
  `action` varchar(48) COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_id` bigint unsigned NOT NULL DEFAULT '0',
  `trigger_source` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `extra_json` varchar(2048) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '{}',
  `notification_id` bigint unsigned NOT NULL DEFAULT '0',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_icgr_instance` (`identifier`,`instance_pk`,`created_at`),
  KEY `idx_icgr_user` (`identifier`,`user_id_before`,`created_at`),
  KEY `idx_icgr_group` (`identifier`,`group_id_before`,`created_at`),
  KEY `idx_icgr_actor` (`identifier`,`actor_type`,`actor_id`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 技能共建审核：通用审批表
CREATE TABLE IF NOT EXISTS `review_requests` (
  `id`             bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`     varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at`     datetime(3) DEFAULT NULL,
  `updated_at`     datetime(3) DEFAULT NULL,
  `deleted_at`     datetime(3) DEFAULT NULL,
  `requester_id`   bigint unsigned NOT NULL DEFAULT '0',
  `resource_type`  varchar(32) NOT NULL DEFAULT 'skill',
  `resource_id`    bigint unsigned NOT NULL DEFAULT '0',
  `action_type`    varchar(16) NOT NULL DEFAULT 'publish',
  `slug`           varchar(191) NOT NULL DEFAULT '',
  `status`         varchar(16) NOT NULL DEFAULT 'pending',
  `reason`         text,
  `reviewer_id`    bigint unsigned NOT NULL DEFAULT '0',
  `reviewed_at`    datetime(3) DEFAULT NULL,
  `review_comment` text,
  PRIMARY KEY (`id`),
  KEY `idx_rr_requester` (`identifier`,`requester_id`),
  KEY `idx_rr_resource` (`identifier`,`resource_type`,`resource_id`),
  KEY `idx_rr_status` (`identifier`,`status`),
  KEY `idx_rr_slug_mutex` (`identifier`,`resource_type`,`slug`,`status`),
  KEY `idx_review_requests_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

create table if not exists `rule_distribution_tasks` (
  `id` bigint unsigned not null auto_increment,
  `identifier` varchar(191) collate utf8mb4_unicode_ci default '',
  `created_at` datetime(3) default null,
  `updated_at` datetime(3) default null,
  `deleted_at` datetime(3) default null,
  `rule_id` bigint unsigned not null,
  `slug` varchar(191) collate utf8mb4_unicode_ci not null default '',
  `rule_type` varchar(16) collate utf8mb4_unicode_ci not null default '' comment '主表 type 冗余：prompt / rule',
  `version` varchar(191) collate utf8mb4_unicode_ci not null default '',
  `batch_id` varchar(64) collate utf8mb4_unicode_ci not null default '',
  `operator_id` bigint unsigned not null default '0',
  `total` bigint not null default '0',
  `success` bigint not null default '0',
  `failed` bigint not null default '0',
  `status` varchar(191) collate utf8mb4_unicode_ci not null default 'running' comment 'running / completed',
  `type` varchar(20) collate utf8mb4_unicode_ci not null default 'distribute' comment 'distribute / uninstall',
  primary key (`id`),
  key `idx_rule_distribution_tasks_identifier` (`identifier`),
  key `idx_rule_distribution_tasks_deleted_at` (`deleted_at`),
  key `idx_rule_distribution_tasks_rule_id` (`rule_id`),
  key `idx_rule_dist_tasks_slug` (`slug`),
  key `idx_rule_dist_tasks_type` (`rule_type`),
  key `idx_rule_dist_tasks_batch` (`batch_id`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_unicode_ci;

create table if not exists `rule_distribution_records` (
  `id` bigint unsigned not null auto_increment,
  `identifier` varchar(191) collate utf8mb4_unicode_ci default '',
  `created_at` datetime(3) default null,
  `updated_at` datetime(3) default null,
  `deleted_at` datetime(3) default null,
  `task_id` bigint unsigned not null,
  `rule_id` bigint unsigned not null,
  `instance_id` bigint unsigned not null,
  `instance_c_id` varchar(191) collate utf8mb4_unicode_ci not null default '',
  `version` varchar(191) collate utf8mb4_unicode_ci not null default '',
  `status` varchar(191) collate utf8mb4_unicode_ci not null default 'pending' comment 'pending / success / failed / upgrade_failed / uninstall_failed_old',
  `error` text collate utf8mb4_unicode_ci,
  `type` varchar(20) collate utf8mb4_unicode_ci not null default 'distribute' comment 'distribute / uninstall',
  primary key (`id`),
  key `idx_rule_distribution_records_identifier` (`identifier`),
  key `idx_rule_distribution_records_deleted_at` (`deleted_at`),
  key `idx_rule_distribution_records_task_id` (`task_id`),
  key `idx_rule_distribution_records_rule_id` (`rule_id`),
  key `idx_rule_distribution_records_instance_id` (`instance_id`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_unicode_ci;

create table if not exists `local_instance_rules` (
  `id` bigint unsigned not null auto_increment,
  `created_at` datetime(3) default null,
  `updated_at` datetime(3) default null,
  `identifier` varchar(191) collate utf8mb4_unicode_ci default '',
  `instance_id` bigint unsigned not null comment '关联 instances.id',
  `slug` varchar(64) collate utf8mb4_unicode_ci not null,
  `version` varchar(32) collate utf8mb4_unicode_ci not null default '',
  `display_name` varchar(128) collate utf8mb4_unicode_ci not null default '',
  `rule_type` varchar(16) collate utf8mb4_unicode_ci not null default '' comment 'prompt / rule',
  `source` varchar(16) collate utf8mb4_unicode_ci not null default 'enterprise' comment 'enterprise / local',
  `installed_at` datetime(3) default null comment '最近一次 ack success / report 上报已装的时刻',
  `last_seen_at` datetime(3) default null comment 'report 中最后一次出现的时刻',
  `scope` varchar(16) collate utf8mb4_unicode_ci not null default 'user' comment 'user / workspace',
  `workspace_path` varchar(512) collate utf8mb4_unicode_ci not null default '' comment '项目级 workspace 路径',
  `install_status` varchar(16) collate utf8mb4_unicode_ci not null default 'distributed' comment 'distributing / distributed / failed',
  primary key (`id`),
  unique key `idx_lir_scope_inst_ws_slug` (`scope`, `instance_id`, `workspace_path`, `slug`),
  key `idx_local_instance_rules_identifier` (`identifier`),
  key `idx_lir_type` (`rule_type`),
  key `idx_local_instance_rules_source` (`source`),
  key `idx_local_instance_rules_last_seen_at` (`last_seen_at`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_unicode_ci;

create table if not exists `rule_visibility_groups` (
  `id` bigint unsigned not null auto_increment,
  `identifier` varchar(191) collate utf8mb4_unicode_ci default '',
  `created_at` datetime(3) default null,
  `rule_id` bigint unsigned not null,
  `group_id` bigint unsigned not null,
  primary key (`id`),
  unique key `idx_ervg_unique` (`identifier`, `rule_id`, `group_id`),
  key `idx_rule_visibility_groups_identifier` (`identifier`),
  key `idx_rule_visibility_groups_rule_id` (`rule_id`),
  key `idx_rule_visibility_groups_group_id` (`group_id`)
) engine=innodb default charset=utf8mb4 collate=utf8mb4_unicode_ci;

-- 项目资产管理：扁平项目
CREATE TABLE IF NOT EXISTS `projects` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `sync_mode`   varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'continuous' COMMENT '同步模式：continuous（持续同步）/ initial_only（仅初始）',
  `created_by` bigint unsigned NOT NULL DEFAULT '0',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_name` (`identifier`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `project_members` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `project_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `created_by` bigint unsigned NOT NULL DEFAULT '0',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_member` (`identifier`,`project_id`,`user_id`),
  KEY `idx_project_member_user` (`identifier`,`user_id`,`project_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `project_config_bindings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `project_id` bigint unsigned NOT NULL,
  `config_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `config_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `value_json` varchar(4096) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '{}',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_config` (`identifier`,`project_id`,`config_type`,`config_key`),
  KEY `idx_project_config_project` (`project_id`,`config_type`),
  KEY `idx_project_config_resource` (`identifier`,`config_type`,`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `local_agent_scope_bindings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `scope` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `scope_key` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `scope_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ide_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `group_id` bigint unsigned NOT NULL DEFAULT '0',
  `project_id` bigint unsigned NOT NULL DEFAULT '0',
  `last_seen_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_local_agent_scope` (`identifier`,`instance_id`,`scope`,`scope_key`),
  KEY `idx_local_agent_project` (`identifier`,`project_id`),
  KEY `idx_local_agent_scope_bindings_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 资产版本记录（资产管理版本记录子任务）
-- 记录项目/分组的资产版本变更历史（手动保存 + 工具库自动变更）。
-- 不承载下发状态（下发状态由 skill/rule 分发 task 表负责）。
CREATE TABLE IF NOT EXISTS `asset_versions` (
  `id`            bigint unsigned NOT NULL AUTO_INCREMENT,
  `target_type`   varchar(32)  COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_id`     bigint unsigned NOT NULL,
  `version`       int          NOT NULL,
  `trigger_type`  varchar(32)  COLLATE utf8mb4_unicode_ci NOT NULL,
  `trigger_reason` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `operator_type` varchar(16)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'user',
  `operator_id`   bigint unsigned NOT NULL DEFAULT 0,
  `operator_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `changes_json`  text         COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at`    datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_av_target` (`target_type`, `target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 通用本地实例任务表（本地 agent 三期）
-- 承载「非 skill/rule 下发」的本地任务，包括 uninstall_teamai 与 execute_agent_task。
-- 与 rule_distribution_tasks 不同：本表是「单实例 + 单任务」粒度，无 task→records 展开。
-- status：pending / running / success / failed / cancelled。
CREATE TABLE IF NOT EXISTS `local_agent_tasks` (
  `id`            bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`    varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at`    datetime(3)  DEFAULT NULL,
  `updated_at`    datetime(3)  DEFAULT NULL,
  `deleted_at`    datetime(3)  DEFAULT NULL,
  `instance_id`   bigint unsigned NOT NULL,
  `instance_c_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `type`          varchar(64)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '任务类型：uninstall_teamai / execute_agent_task',
  `cmd`           text         COLLATE utf8mb4_unicode_ci,
  `status`        varchar(32)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' COMMENT 'pending / running / success / failed / cancelled',
  `error`         text         COLLATE utf8mb4_unicode_ci,
  `operator_id`   bigint unsigned NOT NULL DEFAULT 0,
  `project_id`    bigint unsigned NOT NULL DEFAULT 0,
  `workspace_path` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `agent_type`    varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `prompt`        text COLLATE utf8mb4_unicode_ci,
  `result`        longtext COLLATE utf8mb4_unicode_ci,
  `session_id`    varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `started_at`    datetime(3) DEFAULT NULL,
  `finished_at`   datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_lat_identifier` (`identifier`),
  KEY `idx_lat_instance` (`instance_id`),
  KEY `idx_lat_type` (`type`),
  KEY `idx_lat_project` (`project_id`),
  KEY `idx_lat_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET FOREIGN_KEY_CHECKS = 1;

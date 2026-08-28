-- 0721-local-agent-phase2.sql
-- 本地 Agent 资源分发（Phase 2）增量迁移
-- 目标 Release: Release/2026_07_21
--
-- 包含两部分：
--   A. skill 二期扩展（instances 加字段 + local_instance_skills 加 3 字段）
--   B. 企业规范库基础设施（5 张新表）
--
-- 向后兼容：
--   - 既有 local_instance_skills 行 scope 默认 'user'、workspace_path 默认 ''、install_status 默认 'distributed'
--
-- 同步更新:
--   - model/instance.go (LocalAgentResources 字段)
--   - model/local_instance_skill.go (Scope/WorkspacePath/InstallStatus 字段)
--   - model/enterprise_rule.go (EnterpriseRule + Task/Record + 常量)
--   - model/local_instance_rule.go (LocalInstanceRule)
--   - model/rule_visibility.go (RuleVisibilityGroup)
--   - model/db.go (AutoMigrate 新增 4 张表)
--   - sql/init.sql (加入新表定义)

-- ════════════════════════════════════════════════════════════════
-- A. skill 二期扩展
-- ════════════════════════════════════════════════════════════════

-- A1. instances 表加字段（TEXT 不设 DEFAULT，MySQL 不允许 TEXT 设 DEFAULT）
ALTER TABLE `instances`
  ADD COLUMN `local_agent_resources` text COLLATE utf8mb4_unicode_ci NULL COMMENT '本地 agent 二期：分组绑定 + workspace 列表 JSON（source=local 时存）';

-- A2. local_instance_skills 表加 3 个字段
ALTER TABLE `local_instance_skills`
  ADD COLUMN `scope` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'user' COMMENT '作用域：user（用户级）/ workspace（项目级）',
  ADD COLUMN `workspace_path` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '项目级独有：workspace 路径（scope=workspace 时存）',
  ADD COLUMN `install_status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'distributed' COMMENT '安装状态：distributing / distributed / failed';

-- A3. 替换唯一索引：先删旧索引 (instance_id, slug)，再建新索引 (scope, instance_id, workspace_path, slug)
ALTER TABLE `local_instance_skills` DROP INDEX `idx_lis_inst_slug`;
CREATE UNIQUE INDEX `idx_lis_scope_inst_ws_slug`
  ON `local_instance_skills` (`scope`, `instance_id`, `workspace_path`, `slug`);

-- ════════════════════════════════════════════════════════════════
-- B. 企业规范库基础设施（5 张新表）
-- ════════════════════════════════════════════════════════════════

-- B1. enterprise_rules：规范主表（对齐 skills 表结构）
CREATE TABLE IF NOT EXISTS `enterprise_rules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'prompt / rule',
  `source` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'enterprise' COMMENT 'enterprise / local',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1.0.0',
  `version_major` bigint NOT NULL DEFAULT '0',
  `version_minor` bigint NOT NULL DEFAULT '0',
  `version_patch` bigint NOT NULL DEFAULT '0',
  `cos_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `file_size` bigint NOT NULL DEFAULT '0',
  `content_sha256` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `visibility_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all' COMMENT 'all / group',
  `changelog` varchar(10000) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '版本更新说明',
  `distribute_count` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_enterprise_rules_slug_ver_ident` (`identifier`, `slug`, `version`),
  KEY `idx_enterprise_rules_identifier` (`identifier`),
  KEY `idx_enterprise_rules_deleted_at` (`deleted_at`),
  KEY `idx_enterprise_rules_type` (`type`),
  KEY `idx_enterprise_rules_source` (`source`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- B2. rule_distribution_tasks：规范下发/卸载任务
CREATE TABLE IF NOT EXISTS `rule_distribution_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `rule_id` bigint unsigned NOT NULL,
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `rule_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '主表 type 冗余：prompt / rule',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `batch_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `operator_id` bigint unsigned NOT NULL DEFAULT '0',
  `total` bigint NOT NULL DEFAULT '0',
  `success` bigint NOT NULL DEFAULT '0',
  `failed` bigint NOT NULL DEFAULT '0',
  `status` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'running' COMMENT 'running / completed',
  `type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'distribute' COMMENT 'distribute / uninstall',
  PRIMARY KEY (`id`),
  KEY `idx_rule_distribution_tasks_identifier` (`identifier`),
  KEY `idx_rule_distribution_tasks_deleted_at` (`deleted_at`),
  KEY `idx_rule_distribution_tasks_rule_id` (`rule_id`),
  KEY `idx_rule_dist_tasks_slug` (`slug`),
  KEY `idx_rule_dist_tasks_type` (`rule_type`),
  KEY `idx_rule_dist_tasks_batch` (`batch_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- B3. rule_distribution_records：每实例一条流水
CREATE TABLE IF NOT EXISTS `rule_distribution_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `task_id` bigint unsigned NOT NULL,
  `rule_id` bigint unsigned NOT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `instance_c_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `status` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' COMMENT 'pending / success / failed / upgrade_failed / uninstall_failed_old',
  `error` text COLLATE utf8mb4_unicode_ci,
  `type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'distribute' COMMENT 'distribute / uninstall',
  PRIMARY KEY (`id`),
  KEY `idx_rule_distribution_records_identifier` (`identifier`),
  KEY `idx_rule_distribution_records_deleted_at` (`deleted_at`),
  KEY `idx_rule_distribution_records_task_id` (`task_id`),
  KEY `idx_rule_distribution_records_rule_id` (`rule_id`),
  KEY `idx_rule_distribution_records_instance_id` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- B4. local_instance_rules：本地实例已装规范快照（对称 local_instance_skills，硬删）
-- 二期新增 scope + workspace_path + install_status，与 local_instance_skills 对齐
CREATE TABLE IF NOT EXISTS `local_instance_rules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `instance_id` bigint unsigned NOT NULL COMMENT '关联 instances.id',
  `slug` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `display_name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `rule_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'prompt / rule',
  `source` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'enterprise' COMMENT 'enterprise / local',
  `installed_at` datetime(3) DEFAULT NULL COMMENT '最近一次 ack success / report 上报已装的时刻',
  `last_seen_at` datetime(3) DEFAULT NULL COMMENT 'report 中最后一次出现的时刻',
  `scope` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'user' COMMENT 'user / workspace',
  `workspace_path` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '项目级 workspace 路径',
  `install_status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'distributed' COMMENT 'distributing / distributed / failed',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_lir_scope_inst_ws_slug` (`scope`, `instance_id`, `workspace_path`, `slug`),
  KEY `idx_local_instance_rules_identifier` (`identifier`),
  KEY `idx_lir_type` (`rule_type`),
  KEY `idx_local_instance_rules_source` (`source`),
  KEY `idx_local_instance_rules_last_seen_at` (`last_seen_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- B5. rule_visibility_groups：规范-分组可见性关联（对齐 skill_visibility_groups）
CREATE TABLE IF NOT EXISTS `rule_visibility_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `rule_id` bigint unsigned NOT NULL,
  `group_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ervg_unique` (`identifier`,`rule_id`,`group_id`),
  KEY `idx_rule_visibility_groups_identifier` (`identifier`),
  KEY `idx_rule_visibility_groups_rule_id` (`rule_id`),
  KEY `idx_rule_visibility_groups_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- B6. 存量实例分组归属表：0709 手动回退后在本期迁移中幂等恢复。
-- 全量执行时 0709 已创建时会跳过；从 Release/2026_07_17 升级时补齐基线 init.sql 漏项。
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


-- C. 项目资产管理（本期仅技能/规范）

CREATE TABLE IF NOT EXISTS `projects` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `sync_mode` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'continuous' COMMENT '同步模式：continuous（持续同步）/ initial_only（仅初始）',
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

-- ════════════════════════════════════════════════════════════════
-- D. 资产管理-同步模式与版本记录模块
--   - user_groups.sync_mode（user_groups 建表于 0414，此处用 ALTER 加列）
--   - asset_versions：版本记录历史表
-- ════════════════════════════════════════════════════════════════

-- D1. user_groups 加 sync_mode 列（projects.sync_mode 已在上方 C 建表时直接定义）
ALTER TABLE `user_groups`
  ADD COLUMN `sync_mode` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'continuous' COMMENT '同步模式：continuous（持续同步）/ initial_only（仅初始）'
  AFTER `description`;

-- D2. asset_versions：版本记录历史
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

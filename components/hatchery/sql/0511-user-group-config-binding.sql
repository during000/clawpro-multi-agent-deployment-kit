-- sql/0506-user-group-config-binding.sql
-- 用户分组配置绑定 — 合并迁移
-- 包含：user_groups 树化扩展 + 闭包表 + group_config_bindings + instances.group_id + 全局 Token 配额周期
-- 基于 Release/2026_05_06，合并原 0502/0504/0506 四个独立 SQL

-- ============================================================
-- Part 1: user_groups 扩展（树形结构支持）
-- ============================================================

-- 1.1 user_groups 扩展字段
ALTER TABLE `user_groups` ADD COLUMN `parent_id` bigint unsigned NOT NULL DEFAULT 0 AFTER `description`;
ALTER TABLE `user_groups` ADD COLUMN `depth` int NOT NULL DEFAULT 0 AFTER `parent_id`;
ALTER TABLE `user_groups` ADD COLUMN `full_path` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `depth`;
ALTER TABLE `user_groups` ADD COLUMN `source` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'manual' AFTER `full_path`;
ALTER TABLE `user_groups` ADD COLUMN `source_ref` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `source`;
ALTER TABLE `user_groups` ADD COLUMN `to_be_deleted` tinyint(1) NOT NULL DEFAULT 0 AFTER `source_ref`;
-- 不再引入 GORM 软删 deleted_at 列：分组删除走物理删除，业务"待删占位"语义已由 to_be_deleted 字段承担。

-- 1.2 替换唯一键 (identifier, name) → (identifier, parent_id, name)
ALTER TABLE `user_groups` DROP INDEX `idx_ug_identifier_name`;
ALTER TABLE `user_groups` ADD UNIQUE KEY `idx_ug_ident_parent_name` (`identifier`, `parent_id`, `name`);

-- 1.3 补充索引
ALTER TABLE `user_groups` ADD INDEX `idx_ug_parent`      (`parent_id`);
ALTER TABLE `user_groups` ADD INDEX `idx_ug_source`      (`identifier`, `source`, `source_ref`);
ALTER TABLE `user_groups` ADD INDEX `idx_ug_fullpath`    (`identifier`, `full_path`);
ALTER TABLE `user_groups` ADD INDEX `idx_ug_tobedeleted` (`identifier`, `to_be_deleted`);
ALTER TABLE `user_groups` ADD INDEX `idx_ug_depth`       (`depth`);

-- 1.4 user_group_members 扩展字段
ALTER TABLE `user_group_members` ADD COLUMN `is_main` tinyint(1) NOT NULL DEFAULT 0 AFTER `user_id`;
ALTER TABLE `user_group_members` ADD COLUMN `source`  varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'manual' AFTER `is_main`;
ALTER TABLE `user_group_members` ADD INDEX `idx_user_group_members_is_main` (`is_main`);
ALTER TABLE `user_group_members` ADD INDEX `idx_user_group_members_source`  (`source`);

-- ============================================================
-- Part 2: 闭包表（物化祖先-后代关系）
-- ============================================================
-- 注意：本表的存量数据 backfill（自指闭包行 + 根组 full_path = name）已迁移到
-- Go 程序启动逻辑：model.MigrateUserGroupClosureAndFullPath()，由 InitDB 触发，
-- 仅在 group_closure 为空且 user_groups 非空时执行一次。

CREATE TABLE IF NOT EXISTS `group_closure` (
  `identifier`    varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `ancestor_id`   bigint unsigned NOT NULL,
  `descendant_id` bigint unsigned NOT NULL,
  `depth`         int NOT NULL,
  PRIMARY KEY (`identifier`, `ancestor_id`, `descendant_id`),
  KEY `idx_gc_desc` (`identifier`, `descendant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- Part 3: 分组配置绑定统一表 + 主表 visibility_type 字段
-- ============================================================

CREATE TABLE IF NOT EXISTS `group_config_bindings` (
  `id`           bigint unsigned AUTO_INCREMENT PRIMARY KEY,
  `identifier`   varchar(191) NOT NULL DEFAULT '',
  `config_type`  varchar(32)  NOT NULL,
  `config_key`   varchar(128) NOT NULL,
  `group_id`     bigint unsigned NOT NULL,
  `value_json`   varchar(4096) NOT NULL DEFAULT '{}',
  `created_at`   datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`   datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY `uk_gcb` (`identifier`, `config_type`, `config_key`, `group_id`),
  INDEX `idx_gcb_group` (`identifier`, `group_id`, `config_type`),
  INDEX `idx_gcb_resource` (`identifier`, `config_type`, `config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ai_channels 加 visibility_type 字段
ALTER TABLE `ai_channels` ADD COLUMN `visibility_type` varchar(16) NOT NULL DEFAULT 'all';

-- plugin_bundles 加 visibility_type 字段
ALTER TABLE `plugin_bundles` ADD COLUMN `visibility_type` varchar(16) NOT NULL DEFAULT 'all';

-- mcp_servers 加 visibility_type 字段
ALTER TABLE `mcp_servers` ADD COLUMN `visibility_type` varchar(16) NOT NULL DEFAULT 'all';

-- ============================================================
-- Part 4: instances 表增加 group_id 字段
-- ============================================================

ALTER TABLE `instances` ADD COLUMN `group_id` bigint unsigned NOT NULL DEFAULT 0;
CREATE INDEX `idx_instances_group_id` ON `instances`(`group_id`);

-- ============================================================
-- Part 5: daily_usage_summaries 表增加 group_id 字段
-- ============================================================

ALTER TABLE `daily_usage_summaries` ADD COLUMN `group_id` bigint unsigned NOT NULL DEFAULT 0;
CREATE INDEX `idx_daily_usage_group_id` ON `daily_usage_summaries`(`group_id`);

-- ============================================================
-- Part 6: llm_usage_logs 表增加 group_id 字段
-- ============================================================

ALTER TABLE `llm_usage_logs` ADD COLUMN `group_id` bigint unsigned NOT NULL DEFAULT 0;
CREATE INDEX `idx_llm_usage_logs_group_id` ON `llm_usage_logs`(`group_id`);

-- ============================================================
-- Part 7: site_configs 表增加全局 Token 配额统计周期
-- ============================================================

-- global_token_quota_day 是历史字段名；实际可按日或按月统计，由 global_token_quota_period 决定。
ALTER TABLE `site_configs` ADD COLUMN `global_token_quota_period` varchar(16) NOT NULL DEFAULT 'day' AFTER `global_token_quota_day`;

-- ============================================================
-- Part 8: vpc_configs 独立表（VPC 配置从 group_config_bindings 拆出）
-- ============================================================
-- 原方案曾将分组级 VPC 配置存为 group_config_bindings.config_type='policy', config_key='vpc_config' 的 value_json。
-- 实际 VPC 同时承载 "visibility_type=all/group + subnet_ids 列表 + strategy_name" 多重语义，
-- 复用 group_config_bindings 需双写 binding 与解析 value_json，复杂度过高。
-- 因此独立成 vpc_configs 表，用专属字段直接表达，便于 GORM 关联与 admin 列表查询。

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

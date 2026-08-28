-- 角色版本号 + 实例已下发版本 + 同步状态 + 下发记录表
-- 关联需求：role-switch-and-distribute（角色切换与版本化更新）

-- 1) open_claw_roles 增加 version 字段，存量角色默认 1.0
ALTER TABLE `open_claw_roles`
  ADD COLUMN `version` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1.0' COMMENT '角色版本号，X.Y 两段式' AFTER `visibility_type`;

-- 2) instances 增加 distributed_role_version（最近一次成功推送到此实例的角色版本号）
ALTER TABLE `instances`
  ADD COLUMN `distributed_role_version` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '最近一次成功推送到此实例的角色版本号，X.Y 格式；空串=未下发过' AFTER `soul_set_at`;

-- 3) instances 增加 role_sync_status（4 态状态机：空/pending/updating/updated/failed）
ALTER TABLE `instances`
  ADD COLUMN `role_sync_status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '角色同步状态：空/pending/updating/updated/failed' AFTER `distributed_role_version`,
  ADD KEY `idx_instances_role_sync_status` (`role_sync_status`);

-- 4) 新建 role_distribution_records（每次 apply 一条记录，含 SOUL 与技能子任务状态）
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

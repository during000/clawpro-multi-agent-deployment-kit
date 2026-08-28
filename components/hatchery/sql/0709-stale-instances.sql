-- 存量实例分组归属处理（stale-instances v1.0）
-- 1) instances 表新增 3 列 + 1 索引
-- 2) 新建 instance_flags 表
-- 3) 新建 instance_change_group_records 表

ALTER TABLE `instances`
  ADD COLUMN `handover_target_user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '同组移交目标用户 ID；0 表示无 pending 移交' AFTER `status_synced_at`,
  ADD COLUMN `handover_rejected_by_user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '最近一次拒绝移交的用户 ID；0 表示无' AFTER `handover_target_user_id`,
  ADD COLUMN `handover_initiated_at` datetime(3) DEFAULT NULL COMMENT '移交发起时间' AFTER `handover_rejected_by_user_id`,
  ADD KEY `idx_instances_handover_target_user_id` (`handover_target_user_id`);

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

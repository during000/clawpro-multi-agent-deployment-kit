-- 用户组功能迁移脚本（MySQL 模式）
-- 创建时间：2026-04-14

CREATE TABLE IF NOT EXISTS `user_groups` (
  `id`          bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`  varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name`        varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(1024) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at`  datetime(3) DEFAULT NULL,
  `updated_at`  datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ug_identifier_name` (`identifier`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `user_group_members` (
  `id`            bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`    varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `user_group_id` bigint unsigned NOT NULL,
  `user_id`       bigint unsigned NOT NULL,
  `created_at`    datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ugm_identifier_group_user` (`identifier`,`user_group_id`,`user_id`),
  KEY `idx_user_group_members_user_group_id` (`user_group_id`),
  KEY `idx_user_group_members_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

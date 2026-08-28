-- 0528-image-history-notice-and-disabled-agent-types.sql
-- Add official image update history, per-tenant image update notice flag,
-- and site-level disabled Agent Type list for user-facing visibility control.

ALTER TABLE `ai_images`
  ADD COLUMN `update_notice_enabled` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否提示该官方镜像有更新' AFTER `agent_version`;

CREATE INDEX `idx_ai_images_update_notice_enabled` ON `ai_images` (`update_notice_enabled`);

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

ALTER TABLE `site_configs`
  ADD COLUMN `disabled_agent_types` text COLLATE utf8mb4_unicode_ci NULL COMMENT '用户端禁用的智能体类型 JSON 数组' AFTER `default_agent_type`;

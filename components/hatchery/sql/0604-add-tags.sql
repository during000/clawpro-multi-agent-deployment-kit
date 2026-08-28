-- 标签应用范围：每行是一个 tag key/value，分组范围用独立绑定表索引。
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

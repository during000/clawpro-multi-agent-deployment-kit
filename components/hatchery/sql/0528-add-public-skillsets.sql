-- 0526-add-public-skillsets.sql
-- 新增 public_skillsets 表，用于收藏公共技能包（Skillset）。
-- 仅保存 slug，其他信息解包时从 SkillHub API 实时获取。

CREATE TABLE IF NOT EXISTS `public_skillsets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_public_skillsets_identifier_slug` (`identifier`, `slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

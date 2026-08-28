-- 技能可见性：新增 visibility_type 字段和 skill_visibility_groups 关联表
-- 同时补全模型可见性的存量遗漏（ai_models.visibility_type + model_visibility_groups 表）

-- 模型可见性（补全）
ALTER TABLE `ai_models` ADD COLUMN `visibility_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all';

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

-- 技能可见性（新增）
ALTER TABLE `skills` ADD COLUMN `visibility_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all';

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

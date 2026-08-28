-- 技能包 + 角色可见范围（分组权限）
-- 2026-04-15

-- 技能包添加 visibility_type 字段
ALTER TABLE skill_bundles ADD COLUMN visibility_type VARCHAR(191) NOT NULL DEFAULT 'all';

-- 技能包-分组可见性关联表
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

-- 角色添加 visibility_type 字段
ALTER TABLE open_claw_roles ADD COLUMN visibility_type VARCHAR(191) NOT NULL DEFAULT 'all';

-- 角色-分组可见性关联表
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

-- 技能包内技能记录来源公共技能包（Skillset），支持前端按来源公共技能包/公共技能反查初始技能包
-- 同步更新: model/skill_bundle.go (GORM 结构体) + sql/init.sql (全量建表)

ALTER TABLE `bundle_skills`
  ADD COLUMN `source_skillset_slug` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `source`,
  ADD COLUMN `source_skillset_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `source_skillset_slug`,
  ADD INDEX `idx_bundle_skills_source_skillset_slug` (`source_skillset_slug`),
  ADD INDEX `idx_bundle_skills_source_slug_version_bundle` (`source`, `slug`, `version`, `skill_bundle_id`),
  ADD INDEX `idx_bundle_skills_source_skillset_bundle` (`source_skillset_slug`, `skill_bundle_id`);

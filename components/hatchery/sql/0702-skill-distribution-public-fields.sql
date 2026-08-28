-- 0702-skill-distribution-public-fields.sql
-- 为技能下发/卸载任务添加公共技能与公共技能包查询字段。
-- source/slug 用于公共技能任务定位；source_skillset_slug 用于公共技能包维度搜索；batch_id 用于聚合同一次批量请求拆出的多个 task。

ALTER TABLE `skill_distribution_tasks`
  ADD COLUMN `source` varchar(20) NOT NULL DEFAULT 'enterprise',
  ADD COLUMN `slug` varchar(191) NOT NULL DEFAULT '',
  ADD COLUMN `source_skillset_slug` varchar(191) NOT NULL DEFAULT '',
  ADD COLUMN `batch_id` varchar(64) NOT NULL DEFAULT '';

CREATE INDEX `idx_skill_distribution_tasks_source_slug` ON `skill_distribution_tasks` (`source`, `slug`);
CREATE INDEX `idx_skill_distribution_tasks_source_skillset` ON `skill_distribution_tasks` (`source`, `source_skillset_slug`);
CREATE INDEX `idx_skill_distribution_tasks_batch_id` ON `skill_distribution_tasks` (`batch_id`);

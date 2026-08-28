-- 0522-add-skill-distribution-type.sql
-- 为 skill_distribution_tasks 和 skill_distribution_records 表添加 type 字段
-- 支持区分下发（distribute）和卸载（uninstall）操作类型
-- 历史记录默认值为 'distribute'，无需回填

ALTER TABLE `skill_distribution_tasks` ADD COLUMN `type` varchar(20) NOT NULL DEFAULT 'distribute';
ALTER TABLE `skill_distribution_records` ADD COLUMN `type` varchar(20) NOT NULL DEFAULT 'distribute';

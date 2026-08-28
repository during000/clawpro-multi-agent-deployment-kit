-- 0413-add-skill-changelog.sql
-- 技能表新增 changelog 字段，存储版本更新说明
-- 分两步：先加 NULL 列（兼容已有数据），再改为 NOT NULL DEFAULT ''
ALTER TABLE `skills` ADD COLUMN `changelog` VARCHAR(10000) NULL AFTER `file_size`;
UPDATE `skills` SET `changelog` = '' WHERE `changelog` IS NULL;
ALTER TABLE `skills` MODIFY COLUMN `changelog` VARCHAR(10000) NOT NULL DEFAULT '';

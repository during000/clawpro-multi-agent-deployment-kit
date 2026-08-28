-- 0415-description-fields-to-text.sql
-- 将多个 description / text 字段从 varchar(191) 改为 TEXT，
-- 同时将已有的 TEXT NULL 列改为 TEXT NOT NULL（补齐 GORM 模型语义）。
--
-- 背景：SQLite 不强制 varchar 长度限制，从 SQLite 迁移到 MySQL 后
-- 超长的 description 数据会因 varchar(191) 限制而写入失败。
-- 同时，GORM tag 中的 type:text + default:'' 在 MySQL 上无效
-- （TEXT 列不允许 DEFAULT），已统一去掉 default，改为应用层保证写入。

-- skills.description: varchar(191) → TEXT NOT NULL
ALTER TABLE `skills`
  MODIFY COLUMN `description` text COLLATE utf8mb4_unicode_ci NOT NULL;

-- plugins.description: varchar(191) → TEXT NOT NULL
ALTER TABLE `plugins`
  MODIFY COLUMN `description` text COLLATE utf8mb4_unicode_ci NOT NULL;

-- open_claw_roles.description: varchar(191) → TEXT NOT NULL
-- open_claw_roles.soul: TEXT NULL → TEXT NOT NULL
ALTER TABLE `open_claw_roles`
  MODIFY COLUMN `description` text COLLATE utf8mb4_unicode_ci NOT NULL,
  MODIFY COLUMN `soul` text COLLATE utf8mb4_unicode_ci NOT NULL;

-- skill_categories.description: varchar(191) → TEXT NOT NULL
ALTER TABLE `skill_categories`
  MODIFY COLUMN `description` text COLLATE utf8mb4_unicode_ci NOT NULL;

-- plugin_categories.description: varchar(191) → TEXT NOT NULL
ALTER TABLE `plugin_categories`
  MODIFY COLUMN `description` text COLLATE utf8mb4_unicode_ci NOT NULL;

-- public_skills.description: TEXT NULL → TEXT NOT NULL
ALTER TABLE `public_skills`
  MODIFY COLUMN `description` text COLLATE utf8mb4_unicode_ci NOT NULL;

-- skill_installations.error_message: TEXT NULL → TEXT NOT NULL
ALTER TABLE `skill_installations`
  MODIFY COLUMN `error_message` text COLLATE utf8mb4_unicode_ci NOT NULL;

-- user_groups.description: varchar(1024) → TEXT NOT NULL
ALTER TABLE `user_groups`
  MODIFY COLUMN `description` text COLLATE utf8mb4_unicode_ci NOT NULL;

-- 0430-extend-column-length.sql
-- 扩展两个字段的 varchar 长度，使其与 init.sql 中的最新定义保持一致。
--
-- 背景：
-- 1. ai_models.api_key: 部分第三方模型（尤其是网关/代理形式）下发的
--    api_key 超过 191 字符（例如携带租户信息的长 token），原 varchar(191)
--    写入时被截断或报错，统一扩展到 varchar(512)。
-- 2. public_skills.name: 技能中心上架的公共技能名称在国际化/描述化命名
--    下存在超过 191 字符的情况，扩展到 varchar(256) 以匹配业务需要。
--
-- 说明：MySQL 下对已有数据执行 varchar 扩容是安全且幂等的，
--       重复执行不会造成数据损失。

-- ai_models.api_key: varchar(191) → varchar(512)
ALTER TABLE `ai_models`
  MODIFY COLUMN `api_key` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '';

-- public_skills.name: varchar(191) → varchar(256)
ALTER TABLE `public_skills`
  MODIFY COLUMN `name` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '';

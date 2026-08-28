-- 迁移：site_configs 表新增 default_tags 字段
-- 存储默认标签 JSON 数组，创建实例时自动绑定
-- 格式：[{"Key":"env","Value":"prod"},{"Key":"managed-by","Value":"openclaw"}]

ALTER TABLE `site_configs`
  ADD COLUMN `default_tags` varchar(4096) COLLATE utf8mb4_unicode_ci DEFAULT '[]' AFTER `default_subnet_ids`;

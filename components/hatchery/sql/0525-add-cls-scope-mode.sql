-- 新增 cls_scope_mode 字段，用于区分 CLS 采集范围模式（"all"=全量, "group"=分组）
ALTER TABLE `site_configs`
  ADD COLUMN `cls_scope_mode` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'all'
  AFTER `cls_enabled`;

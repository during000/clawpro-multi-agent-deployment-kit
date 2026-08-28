-- 迁移：site_configs 表新增 gateway_ui_addr_type 字段
-- 用于配置 Gateway UI 访问地址类型，取值 "private"（内网）或 "public"（公网），默认 public

ALTER TABLE `site_configs`
  ADD COLUMN `gateway_ui_addr_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'public' AFTER `gateway_ui_sg_migrate_done`;

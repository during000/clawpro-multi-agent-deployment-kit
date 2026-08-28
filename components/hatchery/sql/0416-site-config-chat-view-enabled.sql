-- 迁移：site_configs 表新增 chat_view_enabled 字段
-- 仅用于指导前端是否加载对话界面，默认开启

ALTER TABLE `site_configs`
  ADD COLUMN `chat_view_enabled` tinyint(1) NOT NULL DEFAULT '1' AFTER `terminal_enabled`;

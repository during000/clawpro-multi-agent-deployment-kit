-- 为 instances 表添加 CLS 插件版本字段
ALTER TABLE `instances`
  ADD COLUMN `cls_plugin_version` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1.0' COMMENT 'CLS 插件版本，1.0=旧版（无 trace），2.0=新版（含 trace）' AFTER `cls_agent_status_at`;

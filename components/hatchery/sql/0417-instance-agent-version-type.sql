-- 为 instances 表添加 Agent 版本信息和类型字段
ALTER TABLE `instances`
  ADD COLUMN `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT '' AFTER `role_id`,
  ADD COLUMN `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT '' AFTER `agent_version`,
  ADD COLUMN `plugin_versions_json` text COLLATE utf8mb4_unicode_ci AFTER `agent_type`,
  ADD COLUMN `version_fetched_at` datetime(3) DEFAULT NULL AFTER `plugin_versions_json`;

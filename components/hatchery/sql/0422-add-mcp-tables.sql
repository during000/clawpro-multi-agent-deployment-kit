-- 0422-add-mcp-tables.sql
-- 企业 MCP 库：新增 5 张表

-- MCP 主表
CREATE TABLE IF NOT EXISTS `mcp_servers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `service_id` varchar(128) NOT NULL,
  `name` varchar(128) NOT NULL,
  `description` varchar(1024) DEFAULT NULL,
  `transport_type` varchar(32) NOT NULL,
  `latest_version_id` bigint unsigned DEFAULT 0,
  `created_by` varchar(128) NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_identifier_service_id` (`identifier`, `service_id`),
  KEY `idx_mcp_servers_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MCP 版本表
CREATE TABLE IF NOT EXISTS `mcp_versions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `mcp_id` bigint unsigned NOT NULL,
  `version` varchar(32) NOT NULL,
  `transport_type` varchar(32) NOT NULL,
  `config_json` text NOT NULL,
  `usage_doc_md` text,
  `tool_doc_md` text,
  `created_at` datetime(3) DEFAULT NULL,
  `created_by` varchar(128) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_mcp_version` (`identifier`, `mcp_id`, `version`),
  KEY `idx_mcp_versions_identifier` (`identifier`),
  KEY `idx_mcp_versions_mcp_id` (`mcp_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MCP 批次任务表
CREATE TABLE IF NOT EXISTS `mcp_distribution_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `mcp_id` bigint unsigned NOT NULL,
  `mcp_snapshot_service_id` varchar(128) NOT NULL,
  `mcp_snapshot_name` varchar(128) NOT NULL,
  `version_id` bigint unsigned DEFAULT 0,
  `version_snapshot` varchar(32) NOT NULL,
  `operator_id` bigint unsigned NOT NULL DEFAULT 0,
  `total` int NOT NULL DEFAULT 0,
  `success` int NOT NULL DEFAULT 0,
  `failed` int NOT NULL DEFAULT 0,
  `status` varchar(16) NOT NULL DEFAULT 'running',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_mcp_distribution_tasks_identifier` (`identifier`),
  KEY `idx_mcp_distribution_tasks_mcp_id` (`mcp_id`),
  KEY `idx_mcp_distribution_tasks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MCP 逐实例下发记录
CREATE TABLE IF NOT EXISTS `mcp_distribution_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `task_id` bigint unsigned NOT NULL,
  `mcp_id` bigint unsigned DEFAULT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `instance_cid` varchar(64) NOT NULL DEFAULT '',
  `version_snapshot` varchar(32) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `error` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_task_instance` (`identifier`, `task_id`, `instance_id`),
  KEY `idx_mcp_distribution_records_identifier` (`identifier`),
  KEY `idx_mcp_distribution_records_task_id` (`task_id`),
  KEY `idx_mcp_distribution_records_mcp_id` (`mcp_id`),
  KEY `idx_mcp_distribution_records_instance_id` (`instance_id`),
  KEY `idx_mcp_distribution_records_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- MCP 实例当前状态表
CREATE TABLE IF NOT EXISTS `mcp_installations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(64) NOT NULL DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `mcp_id` bigint unsigned DEFAULT NULL,
  `service_id` varchar(128) NOT NULL,
  `name` varchar(128) NOT NULL DEFAULT '',
  `version` varchar(32) NOT NULL DEFAULT '',
  `install_status` int NOT NULL DEFAULT 0,
  `last_task_id` bigint unsigned DEFAULT 0,
  `error_message` varchar(2048) NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_instance_service` (`identifier`, `instance_id`, `service_id`),
  KEY `idx_mcp_installations_identifier` (`identifier`),
  KEY `idx_mcp_installations_instance_id` (`identifier`, `instance_id`),
  KEY `idx_mcp_installations_mcp_id` (`mcp_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

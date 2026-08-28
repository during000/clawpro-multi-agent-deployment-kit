CREATE TABLE IF NOT EXISTS `agent_migrations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `cvm_instance_id` varchar(64) NOT NULL DEFAULT '',
  `file_key` varchar(255) NOT NULL DEFAULT '',
  `status` varchar(32) NOT NULL DEFAULT 'pending_upload',
  `fail_reason` text,
  `steps_json` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_agent_migrations_identifier` (`identifier`),
  KEY `idx_agent_migrations_instance_id` (`instance_id`),
  KEY `idx_agent_migrations_status` (`status`),
  KEY `idx_agent_migrations_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

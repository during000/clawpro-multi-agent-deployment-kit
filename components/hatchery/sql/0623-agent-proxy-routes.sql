-- 0623-agent-proxy-routes.sql
-- Add tenant-scoped generic agent proxy route table for public webhook/reverse-proxy endpoints.

CREATE TABLE IF NOT EXISTS `agent_proxy_routes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `route_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `instance_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `kind` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_ip` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `target_port` bigint NOT NULL DEFAULT '0',
  `target_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_agent_proxy_routes_route_id` (`route_id`),
  UNIQUE KEY `idx_proxy_route_instance_kind` (`identifier`,`instance_id`,`kind`),
  KEY `idx_agent_proxy_routes_deleted_at` (`deleted_at`),
  KEY `idx_agent_proxy_routes_identifier` (`identifier`),
  KEY `idx_agent_proxy_routes_instance_id` (`instance_id`),
  KEY `idx_agent_proxy_routes_kind` (`kind`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

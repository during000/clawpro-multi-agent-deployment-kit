-- Release/2026_07_28 资源配置与存量实例调整增量迁移
-- 同步更新: model/*.go + sql/init.sql

-- A. 新建 Agent 资源配置策略
CREATE TABLE IF NOT EXISTS `resource_policies` (
  `id`          bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`  varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name`        varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_default`  tinyint(1) NOT NULL DEFAULT 0,
  `config_json` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at`  datetime(3) DEFAULT NULL,
  `updated_at`  datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_rp_ident_name` (`identifier`, `name`),
  KEY `idx_rp_ident_default` (`identifier`, `is_default`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- B. CVM 资源缓存：仅保留列表展示和筛选需要的事实
ALTER TABLE `instances`
  ADD COLUMN `cvm_instance_type` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `status_synced_at`,
  ADD COLUMN `cvm_cpu` bigint NOT NULL DEFAULT '0' AFTER `cvm_instance_type`,
  ADD COLUMN `cvm_memory_gb` bigint NOT NULL DEFAULT '0' AFTER `cvm_cpu`,
  ADD COLUMN `system_disk_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `cvm_memory_gb`,
  ADD COLUMN `system_disk_size` bigint NOT NULL DEFAULT '0' AFTER `system_disk_type`,
  ADD COLUMN `cvm_public_ip` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `system_disk_size`,
  ADD COLUMN `cvm_internet_charge_type` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `cvm_public_ip`,
  ADD COLUMN `cvm_internet_max_bandwidth_out` bigint NOT NULL DEFAULT '0' AFTER `cvm_internet_charge_type`,
  ADD INDEX `idx_instances_cvm_instance_type` (`cvm_instance_type`),
  ADD INDEX `idx_instances_system_disk_size` (`system_disk_size`);

-- C. 存量实例规格升配与系统盘扩容任务
CREATE TABLE IF NOT EXISTS `instance_adjustments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  `execution_started_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'processing',
  `adjustment_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `phase` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'queued',
  `payload_json` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `run_at` datetime(3) NOT NULL,
  `attempt` int NOT NULL DEFAULT '0',
  `error_code` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_instance_adjustment_instance` (`identifier`,`instance_id`),
  KEY `idx_instance_adjustment_due` (`status`,`run_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

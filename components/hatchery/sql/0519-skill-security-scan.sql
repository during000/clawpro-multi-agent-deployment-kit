-- 技能安全检测功能
-- 新增 skill_security_scans 和 skill_scan_violations 表
-- 新增 site_configs.skill_scan_default_enabled 列

CREATE TABLE IF NOT EXISTS `skill_security_scans` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `identifier` varchar(255) NOT NULL DEFAULT '',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `skill_id` bigint unsigned NOT NULL,
    `skill_version` varchar(50) NOT NULL DEFAULT '',
    `content_hash` varchar(255) NOT NULL,
    `engine_version` int NOT NULL DEFAULT '0',
    `status` varchar(50) NOT NULL DEFAULT 'SCANNING',
    `risk_level` varchar(50) NOT NULL DEFAULT '',
    `primary_rule_id` varchar(50) NOT NULL DEFAULT '',
    `security_score` int NOT NULL DEFAULT '100',
    `scan_result_data` json DEFAULT NULL,
    `report_url` varchar(2048) NOT NULL DEFAULT '',
    `scanned_at` datetime(3) DEFAULT NULL,
    `failed_at` datetime(3) DEFAULT NULL,
    `failure_message` text,
    PRIMARY KEY (`id`),
    KEY `idx_identifier` (`identifier`),
    KEY `idx_skill_id` (`skill_id`),
    KEY `idx_status` (`status`),
    UNIQUE KEY `idx_hash_engine` (`identifier`,`content_hash`,`engine_version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `skill_scan_violations` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `identifier` varchar(255) NOT NULL DEFAULT '',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `skill_security_scan_id` bigint unsigned NOT NULL,
    `rule_id` varchar(50) NOT NULL DEFAULT '',
    `rule_name` varchar(191) NOT NULL DEFAULT '',
    `scan_type` varchar(50) NOT NULL DEFAULT '',
    `description` text NOT NULL,
    `capability_tag` varchar(50) NOT NULL DEFAULT '',
    `capability_tag_name` varchar(191) NOT NULL DEFAULT '',
    PRIMARY KEY (`id`),
    KEY `idx_identifier` (`identifier`),
    KEY `idx_scan_id` (`skill_security_scan_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- site_configs 新增安全扫描默认配置列
ALTER TABLE site_configs ADD COLUMN `skill_scan_default_enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '上传技能时安全检测勾选框默认值';

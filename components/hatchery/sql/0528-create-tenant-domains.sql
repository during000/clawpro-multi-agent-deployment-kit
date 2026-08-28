-- 多租户阶段二：创建域名→租户映射表
-- 全局表，不受 identifier GORM 回调过滤，所有操作通过 DBGlobal 执行。
CREATE TABLE IF NOT EXISTS `tenant_domains` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `domain` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '完整域名，如 a.tcaisite.com',
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '租户标识',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_tenant_domains_domain` (`domain`),
  KEY `idx_tenant_domains_identifier` (`identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

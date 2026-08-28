-- 本地 agent（clawpro 一期）：CLS 公网上报凭据表
-- 按租户隔离（identifier），明文存储，由运维按租户写入。
-- topic_id 不落本表，由 get-config 实时从 CLS OpenClawService 查询。
-- MMDD 前缀按目标 Release 分支日期调整（当前基于 master 最新提交 2026-07-08 暂定 0708）。

CREATE TABLE IF NOT EXISTS `local_agent_cls_credentials` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `deleted_at` datetime(3) DEFAULT NULL,
    `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
    `config_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'cls',
    `secret_id` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
    `secret_key` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_lacc_ident_type` (`identifier`, `config_type`),
    KEY `idx_lacc_identifier` (`identifier`),
    KEY `idx_lacc_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

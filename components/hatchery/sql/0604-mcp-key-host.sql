-- MCP 凭据托管数据库变更
-- 需求：管理端创建 MCP 时，config_json 中的占位符字段（headers/url query）自动识别为托管字段，
--       由安全网关通过查 mcp_installations.hosted_values 渲染并转发。
-- 日期：2026-05-26

-- ========== 1. mcp_servers 新增凭据托管标记 ==========
ALTER TABLE `mcp_servers`
    ADD COLUMN `key_hosted` TINYINT(1) NOT NULL DEFAULT 0
        COMMENT '是否存在托管字段：0=无，1=有';

ALTER TABLE `mcp_servers`
    ADD COLUMN `ip_whitelist` VARCHAR(2048) NOT NULL DEFAULT ''
        COMMENT 'IP白名单（逗号分隔），空=不限制';

-- ========== 2. 模板级：管理员定义哪些 key 需要托管 ==========
CREATE TABLE IF NOT EXISTS `mcp_hosted_keys` (
    `id`            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `identifier`    VARCHAR(191) NOT NULL DEFAULT '' COMMENT '多租户标识',
    `mcp_id`        BIGINT UNSIGNED NOT NULL COMMENT '关联 mcp_servers.id',
    `key`           VARCHAR(128) NOT NULL COMMENT '托管的占位符 key',
    `placeholder`   VARCHAR(256) NOT NULL DEFAULT '' COMMENT '原始占位符值，如 <your-token>',
    `default_value` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '管理员给的默认值（可为空）',
    `created_at`    DATETIME(3) DEFAULT NULL,
    `updated_at`    DATETIME(3) DEFAULT NULL,
    UNIQUE KEY `uk_mcp_key` (`identifier`, `mcp_id`, `key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ========== 3. mcp_installations 新增实例级托管字段 ==========
ALTER TABLE `mcp_installations`
    ADD COLUMN `hosted_values` TEXT COMMENT '实例级托管字段值 JSON: {"Authorization":"Bearer xxx"}';

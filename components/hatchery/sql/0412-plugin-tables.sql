-- 企业插件库相关表
-- 日期：2026-04-12

CREATE TABLE IF NOT EXISTS plugins (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    slug VARCHAR(191) NOT NULL,
    name VARCHAR(191) NOT NULL,
    description VARCHAR(191) NOT NULL DEFAULT '',
    version VARCHAR(191) NOT NULL DEFAULT '1.0.0',
    version_major INT NOT NULL DEFAULT 0,
    version_minor INT NOT NULL DEFAULT 0,
    version_patch INT NOT NULL DEFAULT 0,
    plugin_id VARCHAR(191) NOT NULL DEFAULT '',
    plugin_format VARCHAR(50) NOT NULL DEFAULT 'openclaw',
    kind VARCHAR(50) NOT NULL DEFAULT '',
    cos_zip_key VARCHAR(191) NOT NULL DEFAULT '',
    cos_dir_key VARCHAR(191) NOT NULL DEFAULT '',
    file_list TEXT,
    file_size BIGINT NOT NULL DEFAULT 0,
    npm_package VARCHAR(191) NOT NULL DEFAULT '',
    config_schema TEXT NOT NULL,
    providers TEXT NOT NULL,
    channels TEXT NOT NULL,
    UNIQUE KEY idx_plugin_slug_version_identifier (slug, version, identifier),
    KEY idx_plugins_identifier (identifier),
    KEY idx_plugins_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS plugin_distribution_tasks (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    plugin_db_id BIGINT UNSIGNED NOT NULL,
    version VARCHAR(191) NOT NULL DEFAULT '',
    operator_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    total INT NOT NULL DEFAULT 0,
    success INT NOT NULL DEFAULT 0,
    failed INT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    KEY idx_pdt_identifier (identifier),
    KEY idx_pdt_plugin_db_id (plugin_db_id),
    KEY idx_pdt_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS plugin_distribution_records (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    task_id BIGINT UNSIGNED NOT NULL,
    plugin_db_id BIGINT UNSIGNED NOT NULL,
    instance_id BIGINT UNSIGNED NOT NULL,
    instance_cid VARCHAR(191) NOT NULL DEFAULT '',
    version VARCHAR(191) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error TEXT,
    KEY idx_pdr_identifier (identifier),
    KEY idx_pdr_task_id (task_id),
    KEY idx_pdr_plugin_db_id (plugin_db_id),
    KEY idx_pdr_instance_id (instance_id),
    KEY idx_pdr_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS plugin_categories (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    name VARCHAR(191) NOT NULL,
    description VARCHAR(191) NOT NULL DEFAULT '',
    UNIQUE KEY idx_plugin_cat_name_identifier (name, identifier),
    KEY idx_plugin_categories_identifier (identifier)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS plugin_category_mappings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    plugin_id BIGINT UNSIGNED NOT NULL,
    category_id BIGINT UNSIGNED NOT NULL,
    UNIQUE KEY idx_plugin_cat_map (plugin_id, category_id, identifier),
    KEY idx_pcm_identifier (identifier)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS plugin_bundles (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    name VARCHAR(191) NOT NULL DEFAULT '',
    plugin_count INT NOT NULL DEFAULT 0,
    enabled TINYINT(1) NOT NULL DEFAULT 0,
    KEY idx_plugin_bundles_identifier (identifier)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS bundle_plugins (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    plugin_bundle_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(191) NOT NULL DEFAULT '',
    slug VARCHAR(191) NOT NULL DEFAULT '',
    plugin_id VARCHAR(191) NOT NULL DEFAULT '',
    version VARCHAR(191) NOT NULL DEFAULT '',
    source VARCHAR(50) NOT NULL DEFAULT 'enterprise',
    cos_zip_key VARCHAR(191) NOT NULL DEFAULT '',
    npm_package VARCHAR(191) NOT NULL DEFAULT '',
    install_mode VARCHAR(50) NOT NULL DEFAULT 'smh',
    kind VARCHAR(50) NOT NULL DEFAULT '',
    KEY idx_bundle_plugins_identifier (identifier),
    KEY idx_bundle_plugins_bundle_id (plugin_bundle_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS public_plugins (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    name VARCHAR(191) NOT NULL DEFAULT '',
    slug VARCHAR(191) NOT NULL DEFAULT '',
    plugin_id VARCHAR(191) NOT NULL DEFAULT '',
    version VARCHAR(191) NOT NULL DEFAULT '',
    description TEXT NOT NULL,
    npm_package VARCHAR(191) NOT NULL DEFAULT '',
    total_downloads BIGINT NOT NULL DEFAULT 0,
    total_favorites BIGINT NOT NULL DEFAULT 0,
    KEY idx_public_plugins_identifier (identifier)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS plugin_installations (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    instance_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(191) NOT NULL DEFAULT '',
    slug VARCHAR(191) NOT NULL DEFAULT '',
    plugin_id VARCHAR(191) NOT NULL DEFAULT '',
    version VARCHAR(191) NOT NULL DEFAULT '',
    cos_zip_key VARCHAR(191) NOT NULL DEFAULT '',
    npm_package VARCHAR(191) NOT NULL DEFAULT '',
    install_mode VARCHAR(50) NOT NULL DEFAULT 'smh',
    kind VARCHAR(50) NOT NULL DEFAULT '',
    install_status INT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL,
    KEY idx_plugin_installations_identifier (identifier),
    KEY idx_plugin_installations_instance_id (instance_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS open_claw_role_plugins (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    open_claw_role_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(191) NOT NULL DEFAULT '',
    slug VARCHAR(191) NOT NULL DEFAULT '',
    plugin_id VARCHAR(191) NOT NULL DEFAULT '',
    version VARCHAR(191) NOT NULL DEFAULT '',
    source VARCHAR(50) NOT NULL DEFAULT 'enterprise',
    cos_zip_key VARCHAR(191) NOT NULL DEFAULT '',
    npm_package VARCHAR(191) NOT NULL DEFAULT '',
    install_mode VARCHAR(50) NOT NULL DEFAULT 'smh',
    kind VARCHAR(50) NOT NULL DEFAULT '',
    KEY idx_ocrp_identifier (identifier),
    KEY idx_ocrp_role_id (open_claw_role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入预置插件分类（幂等，仅当不存在时插入）
INSERT IGNORE INTO plugin_categories (identifier, name, description) VALUES
('', 'AI 模型提供商', 'OpenAI、Anthropic、Gemini 等模型接入'),
('', '消息渠道', '企业微信、飞书、钉钉、Slack 等消息渠道'),
('', '智能体工具', '代码搜索、API 调试、文件操作等智能体工具'),
('', '语音与媒体', 'TTS、STT、图像生成、视频生成等'),
('', '知识检索', 'Web 搜索、网页抓取、知识库查询'),
('', '记忆与上下文', '长期记忆、上下文引擎'),
('', '其他', '其他分类');

-- site_configs 表新增 default_plugin_bundle_seeded 字段（兼容 MySQL 5.7+）
SET @col_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'site_configs' AND COLUMN_NAME = 'default_plugin_bundle_seeded');
SET @sql = IF(@col_exists = 0,
    'ALTER TABLE site_configs ADD COLUMN default_plugin_bundle_seeded TINYINT(1) NOT NULL DEFAULT 0',
    'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

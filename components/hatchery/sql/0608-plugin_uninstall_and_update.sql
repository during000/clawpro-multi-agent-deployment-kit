-- 企业插件库管理增强：卸载 + 版本更新 + 应用范围
-- TAPD: #1020422209134626977

-- 1. plugin_distribution_tasks 表新增 type 字段
ALTER TABLE plugin_distribution_tasks ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'distribute';

-- 2. plugin_distribution_records 表新增 type 字段
ALTER TABLE plugin_distribution_records ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'distribute';

-- 3. plugins 表新增 changelog 字段ï¼对齐 skills 表用 VARCHAR，MySQL 5.7 不支持 TEXT DEFAULT）
ALTER TABLE plugins ADD COLUMN changelog VARCHAR(10000) NOT NULL DEFAULT '';

-- 4. plugins 表新增 distribute_count 字段
ALTER TABLE plugins ADD COLUMN distribute_count BIGINT NOT NULL DEFAULT 0;

-- 5. plugins 表新增 visibility_type 字段
ALTER TABLE plugins ADD COLUMN visibility_type VARCHAR(16) NOT NULL DEFAULT 'all';

-- 6. 新建 plugin_visibility_groups 关联表
CREATE TABLE IF NOT EXISTS plugin_visibility_groups (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    identifier VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    plugin_id BIGINT UNSIGNED NOT NULL,
    group_id BIGINT UNSIGNED NOT NULL,
    UNIQUE INDEX idx_pvg_unique (identifier, plugin_id, group_id),
    INDEX idx_pvg_plugin_id (plugin_id),
    INDEX idx_pvg_group_id (group_id),
    INDEX idx_pvg_identifier (identifier)
);

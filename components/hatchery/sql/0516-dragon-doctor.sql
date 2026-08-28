-- 龙虾医生（Dragon Doctor）功能数据库变更
-- 需求：TAPD #1020422209133111304
-- 日期：2026-04-20
-- 更新：2026-04-27（同步 v2.6 方案变更：删 doctor_session_messages / last_active_at / doctor_image_id，
--       新增 sts_expired_at / doctor_authorizations）
-- 更新：2026-04-30（快照融入 start + end 异步化：新增 snapshot_requested / rollback_requested）

-- ========== 1. 新增表：doctor_sessions ==========
CREATE TABLE IF NOT EXISTS `doctor_sessions` (
    `id`                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `created_at`          DATETIME(3)     NOT NULL,
    `updated_at`          DATETIME(3)     NOT NULL,
    `deleted_at`          DATETIME(3)     DEFAULT NULL,
    `identifier`          VARCHAR(191)    NOT NULL DEFAULT '',

    -- 关联信息
    `user_id`             BIGINT UNSIGNED NOT NULL COMMENT '发起诊断的用户 ID',
    `target_instance_id`  BIGINT UNSIGNED NOT NULL COMMENT '被诊断的目标实例 DB ID',
    `doctor_instance_id`  BIGINT UNSIGNED DEFAULT NULL COMMENT '临时诊断节点 DB ID（创建成功后填充）',

    -- 会话状态
    `status`              VARCHAR(16)     NOT NULL DEFAULT 'creating'
        COMMENT 'creating | active | ending | ended | failed',
    `snapshot_requested`  TINYINT(1)      NOT NULL DEFAULT 0
        COMMENT '是否请求在激活后自动创建快照',

    -- 快照
    `has_snapshot`        TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '本次诊断是否已创建快照',
    `snapshot_file_key`   VARCHAR(512)    DEFAULT '' COMMENT 'SMH 备份文件的 fileKey',
    `snapshot_deleted`    TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '快照是否已从 SMH 删除',
    `sessions_deleted`    TINYINT(1)      NOT NULL DEFAULT 0 COMMENT 'session 对话备份是否已从 SMH 删除',
    `rollback_requested`  TINYINT(1)      NOT NULL DEFAULT 0
        COMMENT '结束时是否请求回滚',

    -- STS 临时密钥
    `sts_expired_at`      BIGINT          NOT NULL DEFAULT 0 COMMENT 'STS 临时密钥过期时间（Unix 秒）',

    INDEX `idx_doctor_sessions_identifier` (`identifier`),
    INDEX `idx_doctor_sessions_user_id` (`user_id`),
    INDEX `idx_doctor_sessions_target_instance_id` (`target_instance_id`),
    INDEX `idx_doctor_sessions_status` (`status`),
    INDEX `idx_doctor_sessions_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ========== 2. 新增表：doctor_authorizations ==========
CREATE TABLE IF NOT EXISTS `doctor_authorizations` (
    `id`          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `created_at`  DATETIME(3)     DEFAULT NULL,
    `updated_at`  DATETIME(3)     DEFAULT NULL,
    `deleted_at`  DATETIME(3)     DEFAULT NULL,
    `identifier`  VARCHAR(191)    NOT NULL DEFAULT '',
    `user_id`     BIGINT UNSIGNED NOT NULL,
    `instance_id` BIGINT UNSIGNED NOT NULL,

    UNIQUE KEY `idx_auth_user_instance` (`identifier`, `user_id`, `instance_id`),
    INDEX `idx_doctor_authorizations_identifier` (`identifier`),
    INDEX `idx_doctor_authorizations_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ========== 3. site_configs 新增龙虾医生配置字段 ==========
ALTER TABLE `site_configs`
    ADD COLUMN `doctor_enabled` TINYINT(1) NOT NULL DEFAULT 0
        COMMENT '是否允许用户使用龙虾医生，默认关闭';

-- ========== 4. instances 新增龙虾医生标记字段 ==========
ALTER TABLE `instances`
    ADD COLUMN `is_doctor_node` TINYINT(1) NOT NULL DEFAULT 0
        COMMENT '是否为龙虾医生临时诊断节点';

ALTER TABLE `instances`
    ADD INDEX `idx_instances_is_doctor_node` (`is_doctor_node`);

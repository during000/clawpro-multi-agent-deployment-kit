-- ============================================================
-- Memory Pro 版迁移脚本
-- 适用于已运行 init.sql 的 MySQL 环境（存量升级）
--
-- 变更内容：
--   1. memory_tda_iplugins 表：新增 11 个字段 + 3 个索引
--   2. site_configs 表：新增 1 个字段
--   3. tdai_jobs 表：新建（任务调度表）
--   4. 历史数据迁移：已启用的 Free 插件 current_plan / desired_plan 从 OFF → FREE
--
-- 执行前确认：
--   - 已 USE 到目标数据库
--   - 确认 memory_tda_iplugins / site_configs 表已存在
--   - 本脚本按一次性升级执行；若字段或索引已存在，ALTER TABLE 会报错
-- ============================================================

SET NAMES utf8mb4;

-- -----------------------------------------------------------
-- 1. memory_tda_iplugins 扩展字段（记忆计划 + Pro 绑定）
-- -----------------------------------------------------------

ALTER TABLE `memory_tda_iplugins`
  ADD COLUMN `desired_plan` varchar(32) NOT NULL DEFAULT 'OFF' AFTER `retry_count`,
  ADD COLUMN `current_plan` varchar(32) NOT NULL DEFAULT 'OFF' AFTER `desired_plan`,
  ADD COLUMN `switch_status` varchar(64) NOT NULL DEFAULT '' AFTER `current_plan`,
  ADD COLUMN `last_task_id` bigint unsigned NOT NULL DEFAULT 0 AFTER `switch_status`,
  ADD COLUMN `last_switched_at` datetime(3) DEFAULT NULL AFTER `last_task_id`,
  ADD COLUMN `pool_id` varchar(191) NOT NULL DEFAULT '' AFTER `last_switched_at`,
  ADD COLUMN `database_name` varchar(191) NOT NULL DEFAULT '' AFTER `pool_id`,
  ADD COLUMN `endpoint` varchar(191) NOT NULL DEFAULT '' AFTER `database_name`,
  ADD COLUMN `api_key_secret_ref` varchar(191) NOT NULL DEFAULT '' AFTER `endpoint`,
  ADD COLUMN `vdb_username` varchar(191) NOT NULL DEFAULT '' AFTER `api_key_secret_ref`,
  ADD COLUMN `embedding_model` varchar(191) NOT NULL DEFAULT '' AFTER `vdb_username`,
  ADD INDEX `idx_memory_tda_iplugins_current_plan` (`identifier`, `current_plan`),
  ADD INDEX `idx_memory_tda_iplugins_switch_status` (`identifier`, `switch_status`),
  ADD INDEX `idx_memory_tda_iplugins_pool_database` (`identifier`, `pool_id`, `database_name`);

-- -----------------------------------------------------------
-- 2. 历史数据迁移：根据旧 status 推导 current_plan / desired_plan
--    ENABLED / ENABLING → FREE，其余保持 OFF
-- -----------------------------------------------------------

UPDATE `memory_tda_iplugins`
SET `current_plan` = 'FREE',
    `desired_plan` = 'FREE'
WHERE `status` IN ('ENABLED', 'ENABLING')
  AND `current_plan` = 'OFF';

-- -----------------------------------------------------------
-- 3. site_configs 新增 memory_default_plan 字段
-- -----------------------------------------------------------

ALTER TABLE `site_configs`
  ADD COLUMN `memory_default_plan` varchar(32) NOT NULL DEFAULT 'off' AFTER `memory_tdai_supported_versions`;

-- -----------------------------------------------------------
-- 4. tdai_jobs 任务调度表（新建）
-- -----------------------------------------------------------

CREATE TABLE IF NOT EXISTS `tdai_jobs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `job_type` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `biz_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `instance_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `state` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'PENDING',
  `current_step` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `progress` int NOT NULL DEFAULT 0,
  `run_at` datetime(3) NOT NULL,
  `attempt` int NOT NULL DEFAULT 0,
  `max_attempts` int NOT NULL DEFAULT 3,
  `lease_owner` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `lease_until` datetime(3) DEFAULT NULL,
  `payload_json` text COLLATE utf8mb4_unicode_ci,
  `result_json` text COLLATE utf8mb4_unicode_ci,
  `last_error` text COLLATE utf8mb4_unicode_ci,
  `error_code` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `operator` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `trace_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `finished_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tdai_jobs_deleted_at` (`deleted_at`),
  KEY `idx_tdai_jobs_identifier` (`identifier`),
  KEY `idx_tdai_jobs_job_type` (`job_type`),
  KEY `idx_tdai_jobs_identifier_biz_key` (`identifier`, `biz_key`),
  KEY `idx_tdai_jobs_state_run_at` (`state`, `run_at`),
  KEY `idx_tdai_jobs_instance_state` (`instance_id`, `state`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

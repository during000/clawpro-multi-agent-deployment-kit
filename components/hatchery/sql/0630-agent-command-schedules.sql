-- ========================================================================
-- 0630-agent-command-schedules.sql
-- Agent 命令定时任务（feature/agent_command_execution）
--
-- 本迁移建两张表：
--   1. agent_command_schedules        定时任务配置
--   2. agent_command_schedule_records 定时任务历史执行记录（append-only）
--
-- 表 agent_command_schedules：定时任务配置。到期由后台 runner
-- (task/agent_command_schedule_runner.go) 扫描 (enabled, next_run_at) 后调用
-- startDispatch 触发一次普通 dispatch，执行链路复用既有 dispatch 体系。
--
-- 调度规格用单字符串表达式 schedule_expr 存储，运行时解析（参考 AWS rate(...)）：
--   once(<time>)                    例: once(2026-06-30 15:00)（精确到分钟）
--   rate(d, at=<HH:MM>)             例: rate(d, at=02:00)（每天）
--   rate(w, on=<1-7>, at=<HH:MM>)   例: rate(w, on=1, at=09:00)（每周，1=周一..7=周日）
--   rate(m, on=<1-31>, at=<HH:MM>)  例: rate(m, on=1, at=09:00)（每月，无该日的月份整月跳过）
--
-- 「是否还在执行」不冗余存储：以 last_dispatch_slug 对应 dispatch 是否终态判断。
-- ========================================================================

CREATE TABLE IF NOT EXISTS `agent_command_schedules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `description` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

  `command_id` bigint unsigned NOT NULL DEFAULT '0',
  `instance_ids_json` varchar(8192) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '[]',
  `param_values_json` varchar(4096) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '{}',

  -- 调度规格：单字符串表达式（时间一律按服务器本地时区解释）
  `schedule_expr` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

  -- 运行态
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `next_run_at` datetime(3) DEFAULT NULL,
  `first_run_at` datetime(3) DEFAULT NULL,
  `last_run_at` datetime(3) DEFAULT NULL,
  `last_dispatch_slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `is_running` tinyint(1) NOT NULL DEFAULT '0',
  `last_error` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_by_user_id` bigint unsigned NOT NULL DEFAULT '0',

  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sched_ident_slug` (`identifier`,`slug`),
  KEY `idx_sched_deleted_at` (`deleted_at`),
  KEY `idx_sched_identifier` (`identifier`),
  KEY `idx_sched_command_id` (`command_id`),
  KEY `idx_sched_created_by` (`created_by_user_id`),
  -- 调度扫描核心索引：扫 enabled + 到期
  KEY `idx_sched_due` (`enabled`,`next_run_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------------------------------------
-- 定时任务历史执行记录（append-only）
--
-- 每次 schedule 触发一次 dispatch 就插一条，仅存 dispatch_slug。
-- 执行状态/计数不冗余存储，由 controller 实时查 dispatch 表拼装返回。
-- ------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `agent_command_schedule_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `schedule_id` bigint unsigned NOT NULL DEFAULT '0',
  `dispatch_slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_sched_record_identifier` (`identifier`),
  KEY `idx_sched_record_sched` (`schedule_id`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

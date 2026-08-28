-- ========================================================================
-- 0525-agent-commands.sql
-- Agent 命令执行 (feature/agent_command_execution) — v2 数据模型初版
--
-- 新增 4 张表：
--   - agent_commands              命令模板（软删）
--   - agent_command_dispatch      一次"用户视角的下发"（顶层实体，v2）
--   - agent_command_invocations   一次 TAT RunCommand 调用（dispatch_id FK）
--   - agent_command_tasks         一次 RunCommand 内的一台 instance（dispatch_id + invocation_id 双 FK）
--
-- v2 数据模型：dispatch 是顶层实体，持有 command_snapshot / param_values_json /
-- triggered_by_user_id / status 等字段；invocation/task 通过 dispatch_id FK 关联，
-- dispatch_slug 在子表保留为冗余便于按 slug 反查。
--
-- 详见 openspec/changes/agent-command-execution/design.md §1
-- ========================================================================

CREATE TABLE IF NOT EXISTS `agent_commands` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `description` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'SHELL',
  `content` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `timeout_sec` int unsigned NOT NULL DEFAULT '60',
  `run_user` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'root',
  `workdir` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '/root',
  `params_json` varchar(8192) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '[]',
  `visibility_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'tenant',
  `created_by_user_id` bigint unsigned NOT NULL DEFAULT '0',
  -- 生成列：仅在「未软删」时承载 name，否则为 NULL，配合下方 UNIQUE 实现「同租户、未删唯一」语义。
  -- 软删后该行的 name_active 自动变 NULL，新行的 name_active=name 与 NULL 不冲突 → 名称可重用。
  -- InnoDB 唯一索引中多个 NULL 算独立值，所以已删行不互相冲突。
  `name_active` varchar(60) COLLATE utf8mb4_unicode_ci
    GENERATED ALWAYS AS (CASE WHEN `deleted_at` IS NULL THEN `name` ELSE NULL END) STORED,
  PRIMARY KEY (`id`),
  KEY `idx_agent_commands_deleted_at` (`deleted_at`),
  KEY `idx_agent_commands_created_by_user_id` (`created_by_user_id`),
  UNIQUE KEY `idx_agent_command_ident_slug` (`identifier`,`slug`),
  UNIQUE KEY `idx_agent_command_ident_name_active` (`identifier`,`name_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `agent_command_dispatch` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `command_id` bigint unsigned NOT NULL DEFAULT '0',
  `command_snapshot` text COLLATE utf8mb4_unicode_ci,
  `param_values_json` varchar(4096) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '{}',
  `triggered_by_user_id` bigint unsigned NOT NULL DEFAULT '0',
  `test_first` tinyint(1) NOT NULL DEFAULT '0',
  `test_target_instance_id` bigint unsigned NOT NULL DEFAULT '0',
  `target_count` int unsigned NOT NULL DEFAULT '0',
  `success_count` int unsigned NOT NULL DEFAULT '0',
  `failed_count` int unsigned NOT NULL DEFAULT '0',
  `cancelled_count` int unsigned NOT NULL DEFAULT '0',
  `status` varchar(24) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'in_progress',
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_dispatch_ident_slug` (`identifier`,`slug`),
  KEY `idx_dispatch_identifier` (`identifier`),
  KEY `idx_dispatch_command_id` (`command_id`),
  KEY `idx_dispatch_triggered_by` (`triggered_by_user_id`),
  KEY `idx_dispatch_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `agent_command_invocations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `tat_invocation_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `dispatch_id` bigint unsigned NOT NULL DEFAULT '0',
  `dispatch_slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `is_test_run` tinyint(1) NOT NULL DEFAULT '0',
  `batch_index` int unsigned NOT NULL DEFAULT '0',
  `target_count` int unsigned NOT NULL DEFAULT '0',
  `success_count` int unsigned NOT NULL DEFAULT '0',
  `failed_count` int unsigned NOT NULL DEFAULT '0',
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_invocations_identifier` (`identifier`),
  KEY `idx_invocations_tat_id` (`tat_invocation_id`),
  KEY `idx_invocations_dispatch_id` (`dispatch_id`),
  KEY `idx_invocations_dispatch_slug` (`dispatch_slug`),
  KEY `idx_invocations_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `agent_command_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `tat_invocation_task_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `dispatch_id` bigint unsigned NOT NULL DEFAULT '0',
  `invocation_id` bigint unsigned NOT NULL DEFAULT '0',
  `dispatch_slug` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_id` bigint unsigned NOT NULL DEFAULT '0',
  `cvm_instance_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `agent_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `owner_username` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `is_test_target` tinyint(1) NOT NULL DEFAULT '0',
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `exit_code` int DEFAULT NULL,
  `elapsed_ms` int unsigned DEFAULT NULL,
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tasks_identifier` (`identifier`),
  KEY `idx_tasks_tat_task_id` (`tat_invocation_task_id`),
  KEY `idx_tasks_dispatch_id` (`dispatch_id`),
  KEY `idx_tasks_invocation_id` (`invocation_id`),
  KEY `idx_tasks_dispatch_slug` (`dispatch_slug`),
  KEY `idx_tasks_instance_id` (`instance_id`),
  KEY `idx_tasks_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

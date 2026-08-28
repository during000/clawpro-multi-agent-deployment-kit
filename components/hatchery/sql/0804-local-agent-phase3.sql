-- 0804-local-agent-phase3.sql
-- 本地 Agent 三期：后端改动增量 migration（合并自 tasks + hook 两个文件）
-- 目标 Release: Release/2026_07_28
--
-- 含两块：
--   1. local_agent_tasks：通用本地实例任务表（本期 type=uninstall_teamai，后续扩展复用）
--   2. enterprise_rules：Hook 资源字段（event + cmd，type=hook 时有效）
--
-- 基线建表已同步在 sql/init.sql（local_agent_tasks 完整建表 + enterprise_rules event/cmd 字段）。
--
-- 同步代码：
--   - model/local_agent_task.go        (LocalAgentTask struct + 常量)
--   - model/enterprise_rule.go         (EnterpriseRule 加 Event/Cmd 字段 + 常量)
--   - model/db.go                      (allModels 注册 local_agent_tasks)
--   - controller/local_agent.go        (uninstall_teamai 任务创建 + ack 路由软删)
--   - controller/admin_rules.go        (Hook 资源创建分支)

-- ── 1. 通用本地实例任务表 ──
CREATE TABLE IF NOT EXISTS `local_agent_tasks` (
  `id`            bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`    varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at`    datetime(3)  DEFAULT NULL,
  `updated_at`    datetime(3)  DEFAULT NULL,
  `deleted_at`    datetime(3)  DEFAULT NULL,
  `instance_id`   bigint unsigned NOT NULL,
  `instance_c_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `type`          varchar(64)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '任务类型：uninstall_teamai（本期）',
  `cmd`           text         COLLATE utf8mb4_unicode_ci,
  `status`        varchar(32)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' COMMENT 'pending / success / failed / cancelled',
  `error`         text         COLLATE utf8mb4_unicode_ci,
  `operator_id`   bigint unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_lat_identifier` (`identifier`),
  KEY `idx_lat_instance` (`instance_id`),
  KEY `idx_lat_type` (`type`),
  KEY `idx_lat_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ── 2. 企业规范库 Hook 资源字段（复用 enterprise_rules 表）──
ALTER TABLE `enterprise_rules`
  ADD COLUMN `event` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Hook 触发时机：SessionStart / UserPromptSubmit / PreToolUse / PostToolUse / Stop' AFTER `distribute_count`,
  ADD COLUMN `cmd` text COLLATE utf8mb4_unicode_ci COMMENT 'Hook 执行命令（type=hook 时有效）' AFTER `event`;

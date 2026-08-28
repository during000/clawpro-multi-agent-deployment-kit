-- 0807-local-agent-task-execution.sql
-- ClawPro 向本地 TeamAI/Edge Runtime 下发 Agent 执行任务。
-- 扩展 local_agent_tasks，保留 uninstall_teamai 兼容语义。

ALTER TABLE `local_agent_tasks`
  MODIFY COLUMN `status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' COMMENT 'pending / running / success / failed / cancelled',
  ADD COLUMN `project_id` bigint unsigned NOT NULL DEFAULT 0 AFTER `operator_id`,
  ADD COLUMN `workspace_path` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `project_id`,
  ADD COLUMN `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `workspace_path`,
  ADD COLUMN `prompt` text COLLATE utf8mb4_unicode_ci AFTER `agent_type`,
  ADD COLUMN `result` longtext COLLATE utf8mb4_unicode_ci AFTER `prompt`,
  ADD COLUMN `session_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `result`,
  ADD COLUMN `started_at` datetime(3) DEFAULT NULL AFTER `session_id`,
  ADD COLUMN `finished_at` datetime(3) DEFAULT NULL AFTER `started_at`,
  ADD KEY `idx_lat_project` (`project_id`);

-- 迁移：instances 表新增 runtime_user / runtime_home 字段
-- 用于记录 openclaw 实际运行的系统用户，由 Agent 就绪后探测脚本自动填充

ALTER TABLE `instances`
  ADD COLUMN `runtime_user` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT '' AFTER `agent_ready`,
  ADD COLUMN `runtime_home` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '' AFTER `runtime_user`;

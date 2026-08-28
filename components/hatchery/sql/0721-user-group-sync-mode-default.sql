-- 修复 user_groups.sync_mode 默认值。
-- 0721-local-agent-phase2.sql 已发布，不可回写；本迁移仅调整后续新建分组的默认值，
-- 不修改已有分组记录的同步模式。
ALTER TABLE `user_groups`
  MODIFY COLUMN `sync_mode` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'continuous' COMMENT '同步模式：continuous（持续同步）/ initial_only（仅初始）';

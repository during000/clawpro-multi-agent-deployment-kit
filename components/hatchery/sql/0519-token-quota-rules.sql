-- sql/0519-token-quota-rules.sql
-- 描述：用户 Token 配额规则化改造
--
-- 变更内容：
--   1. users 表新增 token_quota_rules 字段（JSON TEXT）
--   2. site_configs 表新增 default_token_quota_rules 字段
--   3. llm_usage_logs 表新增 covering index (user_id, created_at, total_tokens)
--
-- 注意：不做存量数据迁移。旧 token_quota_day 通过应用层 fallback 兼容，
-- 在用户下次被写入时自动迁移到 token_quota_rules。

-- ============================================================
-- 1. users 表新增 token_quota_rules 字段
-- ============================================================

ALTER TABLE `users` ADD COLUMN `token_quota_rules` text DEFAULT NULL;

-- ============================================================
-- 2. site_configs 表新增 default_token_quota_rules 字段
-- ============================================================

ALTER TABLE `site_configs` ADD COLUMN `default_token_quota_rules` text DEFAULT NULL;

-- ============================================================
-- 3. llm_usage_logs 新增 covering index（支持按用户+时间窗口的 token 聚合查询）
-- ============================================================

CREATE INDEX `idx_logs_user_time_tokens`
ON `llm_usage_logs` (`user_id`, `created_at`, `total_tokens`);

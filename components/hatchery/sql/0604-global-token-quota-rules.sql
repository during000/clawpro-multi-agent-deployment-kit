-- sql/0528-global-token-quota-rules.sql
-- 描述：全局/分组 Token 配额规则化改造
--
-- 变更内容：
--   1. site_configs 表新增 global_token_quota_rules 字段（JSON TEXT）
--   2. llm_usage_logs 新增按全站/分组窗口聚合的覆盖索引
--
-- 注意：不做存量数据迁移。旧 global_token_quota_day + global_token_quota_period
-- 通过应用层 fallback 兼容，后续写入旧字段时自动迁移到 global_token_quota_rules。

ALTER TABLE `site_configs` ADD COLUMN `global_token_quota_rules` text;

CREATE INDEX `idx_logs_time_tokens`
ON `llm_usage_logs` (`created_at`, `total_tokens`);

CREATE INDEX `idx_logs_group_time_tokens`
ON `llm_usage_logs` (`group_id`, `created_at`, `total_tokens`);

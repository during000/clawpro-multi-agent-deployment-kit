-- Add prompt cache token usage counters to LLM usage logs and daily summaries.
ALTER TABLE `llm_usage_logs`
  ADD COLUMN `prompt_cache_read_tokens` bigint NOT NULL DEFAULT 0 AFTER `total_tokens`,
  ADD COLUMN `prompt_cache_write_tokens` bigint NOT NULL DEFAULT 0 AFTER `prompt_cache_read_tokens`;

ALTER TABLE `daily_usage_summaries`
  ADD COLUMN `prompt_cache_read_tokens` bigint NOT NULL DEFAULT 0 AFTER `total_tokens`,
  ADD COLUMN `prompt_cache_write_tokens` bigint NOT NULL DEFAULT 0 AFTER `prompt_cache_read_tokens`;

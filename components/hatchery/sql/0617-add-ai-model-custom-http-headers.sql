-- Add custom_http_headers column to ai_models table
ALTER TABLE `ai_models` ADD COLUMN `custom_http_headers` varchar(4096) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '';

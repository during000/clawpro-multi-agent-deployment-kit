-- 免登录跳转链接：两分钟有效、单次消费的凭证摘要表。
-- 原始凭证不落库；identifier 由 GORM 多租户回调写入和过滤。

CREATE TABLE IF NOT EXISTS `passwordless_login_tokens` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
    `token_hash` char(64) COLLATE utf8mb4_bin NOT NULL,
    `user_id` bigint unsigned NOT NULL,
    `expires_at` datetime(3) NOT NULL,
    `created_at` datetime(3) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_passwordless_login_tokens_hash` (`token_hash`),
    KEY `idx_passwordless_login_tokens_identifier` (`identifier`),
    KEY `idx_passwordless_login_tokens_user_id` (`user_id`),
    KEY `idx_passwordless_login_tokens_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

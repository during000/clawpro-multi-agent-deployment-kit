ALTER TABLE `site_configs`
    ADD COLUMN `security_policies` VARCHAR(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'SSRF';
ALTER TABLE `site_configs`
  ADD COLUMN `smh_auto_provision_on_create` tinyint(1) NOT NULL DEFAULT '0' AFTER `smh_enabled`;

CREATE TABLE IF NOT EXISTS `smh_personal_spaces` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `space_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `instance_id` bigint unsigned NOT NULL,
  `user_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `instance_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `cvm_instance_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `storage_quota` bigint NOT NULL DEFAULT '0',
  `free_storage_quota` bigint NOT NULL DEFAULT '0',
  `env_initialized` tinyint(1) NOT NULL DEFAULT '0',
  `expires_at` datetime(3) DEFAULT NULL,
  `to_be_deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_personal_space_instance_identifier` (`identifier`,`instance_id`),
  KEY `idx_smh_personal_spaces_deleted_at` (`deleted_at`),
  KEY `idx_smh_personal_spaces_identifier` (`identifier`),
  KEY `idx_smh_personal_spaces_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

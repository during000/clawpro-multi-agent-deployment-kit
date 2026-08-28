ALTER TABLE `site_configs`
  ADD COLUMN `smh_provision_error` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `smh_endpoint`;

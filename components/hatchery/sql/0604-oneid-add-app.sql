ALTER TABLE `site_configs`
    ADD COLUMN `one_id_app_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'OneID 自建应用 ID' AFTER `one_id_account_id`,
    ADD COLUMN `one_id_client_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'OneID 应用 client_id' AFTER `one_id_app_id`,
    ADD COLUMN `one_id_client_secret` VARCHAR(256) NOT NULL DEFAULT '' COMMENT 'OneID 应用 client_secret' AFTER `one_id_client_id`,
    ADD COLUMN `one_id_token_endpoint` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'OneID OIDC Token 端点 URL' AFTER `one_id_client_secret`,
    ADD COLUMN `one_id_domain` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'OneID 企业域名' AFTER `one_id_token_endpoint`;

ALTER TABLE `users`
    ADD COLUMN `oneid_login_name` VARCHAR(191) DEFAULT NULL COMMENT 'OneID 登录名' AFTER `one_id_sub`;

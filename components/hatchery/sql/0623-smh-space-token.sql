-- fix(smh): 将 SMH access token 持久化到 smh_spaces 表，
-- 解决多租户/多副本场景下进程内缓存导致 token 混乱的问题。

ALTER TABLE `smh_spaces`
  ADD COLUMN `admin_token` varchar(512) NOT NULL DEFAULT '' AFTER `purpose`,
  ADD COLUMN `admin_token_expired_at` bigint NOT NULL DEFAULT 0 AFTER `admin_token`,
  ADD COLUMN `read_token` varchar(512) NOT NULL DEFAULT '' AFTER `admin_token_expired_at`,
  ADD COLUMN `read_token_expired_at` bigint NOT NULL DEFAULT 0 AFTER `read_token`;

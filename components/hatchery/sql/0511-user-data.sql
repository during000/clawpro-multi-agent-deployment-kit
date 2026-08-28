-- 新增用户 UserData 能力开关与实例 UserData 持久化字段
ALTER TABLE `site_configs`
  ADD COLUMN `user_data_enabled` tinyint(1) NOT NULL DEFAULT '0'
  AFTER `browser_vnc_enable`;

ALTER TABLE `instances`
  ADD COLUMN `user_data` text COLLATE utf8mb4_unicode_ci
  AFTER `custom_model_config`;

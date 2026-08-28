-- 平台策略功能权限开关：允许用户配置模型/通道、查看模型额度
-- 默认值 1（开启），存量数据自动填充

ALTER TABLE `site_configs` ADD COLUMN `user_config_model_enabled` tinyint(1) NOT NULL DEFAULT '1';
ALTER TABLE `site_configs` ADD COLUMN `user_config_channel_enabled` tinyint(1) NOT NULL DEFAULT '1';
ALTER TABLE `site_configs` ADD COLUMN `model_quota_enabled` tinyint(1) NOT NULL DEFAULT '1';

-- 实例与 AI 模型的绑定关系（多模型 Fallback）
-- 说明：该表在灰度环境曾预建过（字段/索引可能与最终定义不一致），
-- 这里先 DROP 再 CREATE，保证从灰度到现网 revert 时表结构干净一致。
DROP TABLE IF EXISTS `instance_models`;
CREATE TABLE IF NOT EXISTS `instance_models` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `instance_id` bigint unsigned NOT NULL,
  `ai_model_id` bigint unsigned NOT NULL DEFAULT '0',
  `custom_model_id` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `role` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'fallback',
  `sort_order` int NOT NULL DEFAULT '0',
  `custom_model_config` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_instance_model` (`identifier`,`instance_id`,`ai_model_id`,`custom_model_id`),
  KEY `idx_instance_models_identifier` (`identifier`),
  KEY `idx_instance_id` (`instance_id`),
  KEY `idx_instance_models_deleted_at` (`deleted_at`),
  KEY `idx_instance_models_sort_order` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

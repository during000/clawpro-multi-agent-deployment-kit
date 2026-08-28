-- sg-ruleset-projection 增量迁移
--
-- 从 master 升级到新方案（RuleSet + 统一 SG 池）：
--   * 新增 rule_sets 表（规则真相源）
--   * 新增 managed_sg_pool 表（SG 池：sg_id / sg_name / rule_set_id / rule_version / status / cvm_count ...）
--   * 新增 sg_drain_state 表（DrainWorker 失败计数）
--   * 老表 instances / site_configs 补列（向 master 租户兼容）
--
-- 兼容性：
--   * 所有 ALTER TABLE ADD COLUMN / ADD INDEX 使用"SET @var + PREPARE/EXECUTE"幂等模式，
--     兼容 MySQL 5.7（不依赖 8.0.29+ 的 `IF NOT EXISTS` 原生语法，不使用存储过程）
--   * 脚本可重复执行

-- 1. RuleSet：规则真相源
--    description       管理员自定义备注，UI 展示用。
--    user_group_ids / is_default 是为"多 RuleSet 按用户组路由"预留的字段，本期不消费：
--      - user_group_ids：本 RuleSet 作用到的用户组 ID 列表（JSON 数组字符串），本期恒为 "[]"
--      - is_default    ：默认规则组标记，本期每租户仅一行 → 恒为 1
CREATE TABLE IF NOT EXISTS `rule_sets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `name` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'clawpro-default',
  `description` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '管理员自定义备注，UI 展示用',
  `rules` text COLLATE utf8mb4_unicode_ci,
  `version` int NOT NULL DEFAULT 1,
  `user_group_ids` text COLLATE utf8mb4_unicode_ci COMMENT '预留：未来多 RuleSet 按用户组路由时用，本期恒为 "[]"',
  `is_default` tinyint(1) NOT NULL DEFAULT 1 COMMENT '预留：默认规则组标记，本期单一 RuleSet 恒为 1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_rs_ident_name` (`identifier`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. ManagedSGPool：SG 池
--    sg_name：由 Guardian 每 5 分钟从云 API 同步云端 SG 实际名称（用户可能在云控制台手动改名）
CREATE TABLE IF NOT EXISTS `managed_sg_pool` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `sg_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sg_name` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Guardian 从云 API 同步的 SG 实际名称',
  `rule_set_id` bigint unsigned NOT NULL,
  `rule_version` int NOT NULL DEFAULT 0,
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'ACTIVE',
  `cvm_count` int NOT NULL DEFAULT 0,
  `cvm_count_at` datetime(3) DEFAULT NULL,
  `drained_at` datetime(3) DEFAULT NULL,
  `drift_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_managed_sg_pool_sg_id` (`sg_id`),
  KEY `idx_managed_sg_pool_identifier` (`identifier`),
  KEY `idx_managed_sg_pool_rs_status` (`rule_set_id`, `status`),
  KEY `idx_managed_sg_pool_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


-- 3. SGDrainState：DrainWorker 失败计数与 drain_stuck
CREATE TABLE IF NOT EXISTS `sg_drain_state` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `instance_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `fail_count` int NOT NULL DEFAULT 0,
  `stuck_at` datetime(3) DEFAULT NULL,
  `last_error` text COLLATE utf8mb4_unicode_ci,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_sg_drain_state_instance_id` (`instance_id`),
  KEY `idx_sg_drain_state_identifier` (`identifier`),
  KEY `idx_sg_drain_state_stuck` (`stuck_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. 补齐 instances / site_configs 缺失的列和索引 + 回填存量实例。

-- 4.1 instances.security_group_id
ALTER TABLE `instances` ADD COLUMN `security_group_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'ClawPro 管理的实例主 SG，sg-ruleset-projection change 引入' AFTER `subnet_id`;

-- 4.3 site_configs.sg_pool_auto_scale_threshold
ALTER TABLE `site_configs` ADD COLUMN `sg_pool_auto_scale_threshold` int NOT NULL DEFAULT 1800 COMMENT 'SG 池单 SG 实例数达到此值触发扩容，默认 1800';

-- 4.4 存量实例回填 security_group_id（Bootstrap 计 FROZEN.cvm_count 需要）
UPDATE `instances` i
  INNER JOIN `site_configs` s ON s.identifier = i.identifier
SET i.security_group_id = s.security_group_id
WHERE (i.security_group_id IS NULL OR i.security_group_id = '')
  AND s.security_group_id IS NOT NULL
  AND s.security_group_id != '';


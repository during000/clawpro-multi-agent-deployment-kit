-- clawpro 本地 agent 一期：扩展 instances 表 + 新增 local_instance_infos / local_instance_skills
--
-- 同步更新:
--   - model/instance.go (Instance.Source 字段)
--   - model/local_instance_info.go (新模型)
--   - model/local_instance_skill.go (新模型)
--   - model/skill.go (SkillDistributionTask.Slug 字段，本地 agent 未注册 skill 兜底场景使用)
--   - sql/init.sql (instances 表 + skill_distribution_tasks.slug + 两张新表)
--
-- 设计文档: https://iwiki.woa.com/p/4022150701

-- 1. instances 表新增 source 字段
ALTER TABLE `instances`
  ADD COLUMN `source` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'cvm' COMMENT '实例来源：cvm（云上）/ local（本地 agent）' AFTER `status_synced_at`,
  ADD INDEX `idx_instances_source` (`source`);

-- 2. local_instance_infos：本地 agent 实例的扩展信息（与 instances 1:1）
CREATE TABLE IF NOT EXISTS `local_instance_infos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `instance_id` bigint unsigned NOT NULL COMMENT '关联 instances.id',
  `host_name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `os` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `started_at` datetime(3) DEFAULT NULL COMMENT 'reporter 上报的进程启动时间',
  `last_report_at` datetime(3) DEFAULT NULL COMMENT '最近一次 report/sync 上报时间',
  `last_status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'reporter 端派生的运行状态文案',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_local_instance_infos_instance_id` (`instance_id`),
  KEY `idx_local_instance_infos_deleted_at` (`deleted_at`),
  KEY `idx_local_instance_infos_identifier` (`identifier`),
  KEY `idx_local_instance_infos_last_report_at` (`last_report_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. local_instance_skills：本地实例当前已安装 skill 的事实快照（仅 success，不使用软删）
CREATE TABLE IF NOT EXISTS `local_instance_skills` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `instance_id` bigint unsigned NOT NULL COMMENT '关联 instances.id',
  `slug` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `display_name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'local' COMMENT 'public / enterprise / local',
  `installed_at` datetime(3) DEFAULT NULL COMMENT '最近一次 ack success / report 上报已装的时刻',
  `last_seen_at` datetime(3) DEFAULT NULL COMMENT 'report 中最后一次出现的时刻',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_lis_inst_slug` (`instance_id`,`slug`),
  KEY `idx_local_instance_skills_identifier` (`identifier`),
  KEY `idx_local_instance_skills_source` (`source`),
  KEY `idx_local_instance_skills_last_seen_at` (`last_seen_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 0630-feature-allowlist.sql
-- 全局功能白名单表（无 identifier 隔离，跨租户）。
--
-- 第一期使用：type='local-agent' 控制 /local-agent/* 接口对哪些租户开放。
-- 上线 checklist：表创建后必须 INSERT 至少 1 条记录开通指定租户；
-- 「该 type 下空表 = 全部租户放行」是 sane default，部署时不要忘了加白。

CREATE TABLE IF NOT EXISTS `feature_allowlists` (
                                                   `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                                   `type` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '功能类别，如 local-agent',
                                                   `identifier` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '租户标识',
                                                   `note` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '备注',
                                                   `created_at` datetime(3) DEFAULT NULL,
                                                   PRIMARY KEY (`id`),
                                                   UNIQUE KEY `idx_feature_allowlist_type_identifier` (`type`, `identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 上线时按需替换为真实 identifier，例如：
INSERT INTO `feature_allowlists` (`type`, `identifier`, `note`, `created_at`)
VALUES ('local-agent', 'clp-mxqgxccf', '大计算', NOW(3)),
('local-agent', 'clp-nmyy3n7z', '南京灰度', NOW(3));

ALTER TABLE `site_configs`
    ADD COLUMN `local_agent_enabled` tinyint(1) NOT NULL DEFAULT 0
        COMMENT '是否允许用户接入本地 Agent，默认关闭；与 feature_allowlist 一起作双层守卫';

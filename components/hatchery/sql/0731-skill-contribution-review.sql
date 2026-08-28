-- 0724-skill-contribution-review.sql
-- 技能共建审核：Skill 加 status + uploader_id；新建 review_requests 通用审批表
-- 关联需求：feature/skill-contribution-review

-- 1) skills 表新增 status（published/pending_review/offline）和 uploader_id（0=admin, >0=员工）
ALTER TABLE `skills`
  ADD COLUMN `status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'published' AFTER `distribute_count`,
  ADD COLUMN `uploader_id` bigint unsigned NOT NULL DEFAULT '0' AFTER `status`;

-- 2) 通用审批表
CREATE TABLE IF NOT EXISTS `review_requests` (
  `id`             bigint unsigned NOT NULL AUTO_INCREMENT,
  `identifier`     varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `created_at`     datetime(3) DEFAULT NULL,
  `updated_at`     datetime(3) DEFAULT NULL,
  `deleted_at`     datetime(3) DEFAULT NULL,
  `requester_id`   bigint unsigned NOT NULL DEFAULT '0',
  `resource_type`  varchar(32) NOT NULL DEFAULT 'skill',
  `resource_id`    bigint unsigned NOT NULL DEFAULT '0',
  `action_type`    varchar(16) NOT NULL DEFAULT 'publish',
  `slug`           varchar(191) NOT NULL DEFAULT '',
  `status`         varchar(16) NOT NULL DEFAULT 'pending',
  `reason`         text,
  `reviewer_id`    bigint unsigned NOT NULL DEFAULT '0',
  `reviewed_at`    datetime(3) DEFAULT NULL,
  `review_comment` text,
  PRIMARY KEY (`id`),
  KEY `idx_rr_requester` (`identifier`,`requester_id`),
  KEY `idx_rr_resource` (`identifier`,`resource_type`,`resource_id`),
  KEY `idx_rr_status` (`identifier`,`status`),
  KEY `idx_rr_slug_mutex` (`identifier`,`resource_type`,`slug`,`status`),
  KEY `idx_review_requests_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

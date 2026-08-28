-- 0414-notification-category-error-detail.sql
-- 为 notifications 表新增 category（消息类别）和 error_detail（错误详情）字段，
-- 以及 (identifier, category) 复合索引，支持按类别过滤通知。

ALTER TABLE `notifications`
  ADD COLUMN `category` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'notice' AFTER `type`,
  ADD COLUMN `error_detail` text COLLATE utf8mb4_unicode_ci AFTER `message`,
  ADD KEY `idx_notifications_identifier_category` (`identifier`, `category`);

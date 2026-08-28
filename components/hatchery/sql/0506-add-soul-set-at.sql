-- 0506-add-soul-set-at.sql
-- 新增 instances.soul_set_at 字段，用于跟踪 Soul 是否已下发到实例。
-- 存量实例 role_id > 0 但 soul_set_at 为 NULL 表示 Soul 尚未下发，需后台补下发。
ALTER TABLE `instances`
  ADD COLUMN `soul_set_at` datetime(3) DEFAULT NULL
  AFTER `role_id`;

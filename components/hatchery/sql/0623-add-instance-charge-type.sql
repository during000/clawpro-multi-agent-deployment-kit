-- 添加实例实际计费类型字段，存量按历史默认 PREPAID 回填
ALTER TABLE `instances`
  ADD COLUMN `instance_charge_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'PREPAID' AFTER `instance_id`;

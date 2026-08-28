-- 管理端实例列表查询性能优化：实例状态缓存字段
-- 后台 cvm-status-reconcile 任务周期刷新，List 接口降为纯 DB 读
-- 同步更新: model/instance.go (GORM 结构体) + sql/init.sql (全量建表)

-- 1. instances 表新增 3 个缓存字段
ALTER TABLE `instances`
  ADD COLUMN `last_known_status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '最终语义状态(running/stopped/load_failed/destroyed...)' AFTER `is_doctor_node`,
  ADD INDEX `idx_instances_last_known_status` (`identifier`, `last_known_status`);

ALTER TABLE `instances`
  ADD COLUMN `cvm_tags_json` text COLLATE utf8mb4_unicode_ci NULL COMMENT 'CVM 标签缓存 JSON，供标签过滤' AFTER `last_known_status`;

ALTER TABLE `instances`
  ADD COLUMN `img_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'CVM 镜像 ID 缓存，供 IsOfficialImage 判断' AFTER `cvm_tags_json`;

ALTER TABLE `instances`
  ADD COLUMN `status_synced_at` datetime(3) DEFAULT NULL COMMENT '缓存最后同步时间，用于竞态保护' AFTER `img_id`;

-- 2. site_configs 表新增 1 个字段：整轮同步成功标记
ALTER TABLE `site_configs`
  ADD COLUMN `last_full_sync_finished_at` datetime(3) DEFAULT NULL COMMENT '后台 full-sync 整轮完成时间，非 NULL 表示缓存就绪' AFTER `skill_scan_default_enabled`;

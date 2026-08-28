-- 升级断点续传记录字段：把原本的进程内 sync.Map 缓存持久化到 instances 表，
-- 多副本（多 Pod / 多机器 + LB）部署下重试请求落到不同副本时也能正确续传。
-- 字段含义详见 controller/openclaw_upgrade.go 中的 pendingUploadStore 注释。
ALTER TABLE `instances`
  ADD COLUMN `pending_archive_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'CVM 上未上传完成的备份压缩包路径'
  AFTER `version_fetched_at`,
  ADD COLUMN `pending_archive_size` bigint NOT NULL DEFAULT '0' COMMENT '备份压缩包大小（字节）'
  AFTER `pending_archive_path`,
  ADD COLUMN `pending_smh_file_key` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'SMH 文件 key，用于复用同一个 ConfirmKey 续传'
  AFTER `pending_archive_size`,
  ADD COLUMN `pending_upload_at` datetime(3) DEFAULT NULL COMMENT '续传记录写入时间，便于运维判断陈旧程度'
  AFTER `pending_smh_file_key`;

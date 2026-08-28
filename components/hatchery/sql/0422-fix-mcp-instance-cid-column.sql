-- 修复 mcp_distribution_records.instance_cid 列名与 GORM 默认命名策略不一致的问题
-- GORM 将 InstanceCID 转为 instance_c_id，建表脚本误写为 instance_cid
-- 对齐 0415-rename-cvm-instance-id.sql 中 plugin_distribution_records 的同类修复

ALTER TABLE `mcp_distribution_records`
  RENAME COLUMN `instance_cid` TO `instance_c_id`;

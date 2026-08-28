-- 修复列名与 GORM 默认命名策略不一致的问题
-- GORM 对连续大写字母（如 CVM、CID）会插入下划线拆分

-- smh_personal_spaces: CVMInstanceId -> c_vm_instance_id（init.sql 误写为 cvm_instance_id）
ALTER TABLE `smh_personal_spaces`
  RENAME COLUMN `cvm_instance_id` TO `c_vm_instance_id`;

-- plugin_distribution_records: InstanceCID -> instance_c_id（init.sql 误写为 instance_cid）
ALTER TABLE `plugin_distribution_records`
  RENAME COLUMN `instance_cid` TO `instance_c_id`;

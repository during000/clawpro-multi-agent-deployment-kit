-- 0509-add-memory-plugin-version-offload.sql
-- 为 memory_tda_iplugins 表新增 memory_plugin_version 和 offload_enabled 字段
-- 用于一键升级功能：记录记忆插件版本和 Offload 开启状态

ALTER TABLE memory_tda_iplugins ADD COLUMN memory_plugin_version VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE memory_tda_iplugins ADD COLUMN offload_enabled TINYINT(1) DEFAULT NULL;

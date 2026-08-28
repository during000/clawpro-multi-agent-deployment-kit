-- 迁移脚本：sql/0417-add-instance-agent-type.sql
-- 描述：补充实例表智能体类型字段的默认值和索引（agent_type/agent_version 列已由 0417-instance-agent-version-type.sql 创建）

-- 将 agent_type 默认值从空字符串改为 openclaw，并补充 COMMENT
ALTER TABLE `instances`
  MODIFY COLUMN `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'openclaw' COMMENT '智能体类型',
  MODIFY COLUMN `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体版本号';

-- 存量数据：将空 agent_type 统一设为 openclaw
UPDATE `instances` SET `agent_type` = 'openclaw' WHERE `agent_type` = '' OR `agent_type` IS NULL;

-- 添加索引
CREATE INDEX idx_instances_agent_type ON instances(agent_type);
CREATE INDEX idx_instances_user_agent_type ON instances(user_id, agent_type);

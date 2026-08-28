-- 迁移脚本：sql/0415-add-default-agent-type.sql
-- 描述：在 site_configs 表新增 default_agent_type 字段

ALTER TABLE `site_configs`
ADD COLUMN `default_agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'openclaw'
COMMENT '用户端首选智能体类型';

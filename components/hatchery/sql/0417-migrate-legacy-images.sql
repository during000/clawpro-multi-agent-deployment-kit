-- 迁移脚本：sql/0415-migrate-legacy-images.sql
-- 描述：将存量已导入镜像标记为 OpenClaw 类型

-- 1. 存量镜像统一归属 OpenClaw（agent_version 保持为空，表示存量）
UPDATE `ai_images`
SET `agent_type` = 'openclaw'
WHERE `agent_type` = '' OR `agent_type` IS NULL;

-- 2. 存量实例统一归属 OpenClaw
UPDATE `instances`
SET `agent_type` = 'openclaw'
WHERE `agent_type` = '' OR `agent_type` IS NULL;

-- 迁移脚本：sql/0415-add-image-agent-type.sql
-- 描述：为镜像表添加智能体类型和版本字段

ALTER TABLE `ai_images`
  ADD COLUMN `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体类型' AFTER `enabled`,
  ADD COLUMN `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体版本号' AFTER `agent_type`;

-- 添加索引：按类型+启用状态查询
CREATE INDEX idx_ai_images_agent_type_enabled ON ai_images(agent_type, enabled);

-- SkillHub 技能迁移配置字段
-- skill_hub_enabled: 是否启用 SkillHub 迁移（灰度开关）
-- skill_hub_api_url: SkillHub API 请求地址（迁移代理用，与 skill_hub 分开存储，避免用户在页面修改 skill_hub 影响代理）
ALTER TABLE `site_configs`
    ADD COLUMN `skill_hub_enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否启用 SkillHub 迁移',
    ADD COLUMN `skill_hub_api_url` text COLLATE utf8mb4_unicode_ci COMMENT 'SkillHub API 请求地址';

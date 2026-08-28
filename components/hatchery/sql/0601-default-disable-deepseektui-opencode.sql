-- 0601-default-disable-deepseektui-opencode.sql
-- 新增内置 Agent Type DeepSeekTUI / OpenCode 后，默认对所有租户的用户端禁用，
-- 必须管理员通过 /admin/agent-types/enabled 显式启用后才允许员工端创建实例。
--
-- 设计取舍：
--   1) 仅当 disabled_agent_types 当前为 NULL/空字符串/'[]' 时写入，
--      避免覆盖管理员已手动配置的禁用清单；
--   2) 已经手工调整过 disabled_agent_types 的租户由其管理员自行决定是否纳入。
-- 与 model/site_config.go::defaultDisabledAgentTypesJSON 保持一致。

UPDATE site_configs
SET disabled_agent_types = '["deepseektui","opencode"]'
WHERE disabled_agent_types IS NULL
   OR TRIM(disabled_agent_types) = ''
   OR TRIM(disabled_agent_types) = '[]';

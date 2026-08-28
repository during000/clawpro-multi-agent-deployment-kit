-- 用户端 MCP 管理：扩展 mcp_installations 表
-- 新增字段用于支持用户自选 MCP、连通性探测、配置存储
-- 注意：config_json 和 tools_json 为 TEXT 类型，MySQL 不支持 TEXT DEFAULT，声明为可 NULL，应用层保证写入空字符串

ALTER TABLE mcp_installations ADD COLUMN config_json TEXT;
ALTER TABLE mcp_installations ADD COLUMN source VARCHAR(16) NOT NULL DEFAULT 'admin';
ALTER TABLE mcp_installations ADD COLUMN connection_status VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE mcp_installations ADD COLUMN tools_json TEXT;
ALTER TABLE mcp_installations ADD COLUMN connection_error VARCHAR(1024) NOT NULL DEFAULT '';
ALTER TABLE mcp_installations ADD COLUMN probed_at DATETIME NULL;

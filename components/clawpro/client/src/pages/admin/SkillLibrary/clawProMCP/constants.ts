/**
 * ClawPro 平台 MCP 常量定义
 *
 * 这是平台内置的一条特殊 MCP 服务，让 Agent 能在对话中管理 ClawPro 平台本身。
 * 与企业 MCP 库里普通的业务 MCP（如工蜂、iWiki）区分：
 *   - 不可删除、不可修改 transport
 *   - 列表顶部特殊卡片展示
 *   - 详情页采用 4-Tab 结构（概览 / 能力配置 / 下发管理 / 调用日志）
 */

/** 这条特殊 MCP 在 MCPService 列表里的唯一标识；用 name 字段保持与现有 MCPService 模型一致 */
export const CLAWPRO_PLATFORM_MCP_ID = 'clawpro-platform';

/** 显示名称（管理员看到的标题） */
export const CLAWPRO_PLATFORM_MCP_NAME = 'ClawPro 平台 MCP';

/** 一句话定位 */
export const CLAWPRO_PLATFORM_MCP_TAGLINE = '让 Agent 在对话中管理 ClawPro：查 Agent、下发技能、调整配置';

/** 当前版本 */
export const CLAWPRO_PLATFORM_MCP_VERSION = '1.0.0';

/** 服务地址（mock，正式接入后由后端下发） */
export const CLAWPRO_PLATFORM_MCP_SERVICE_URL = 'https://clawpro.example.com/mcp/admin';

/** 传输协议 —— 内置 MCP 固定为 Streamable HTTP，不可改 */
export const CLAWPRO_PLATFORM_MCP_TRANSPORT = 'streamable-http' as const;

/**
 * mcpStore - 企业 MCP 库共享 Store
 * 注意：MCP 以 name（服务标识）作为唯一 id。
 */
import { createLibraryStore } from './libraryStore';
import type { MCPService } from './types';

const MCP_CACHE_KEY = 'mcphub_enterprise_mcps_cache';
const MCP_CACHE_VERSION_KEY = 'mcphub_enterprise_mcps_cache_version';
const MCP_CACHE_VERSION = '9';

export const MCP_STORE_EVENT = 'mcp-store-updated';

export const MOCK_MCPS: MCPService[] = [
  {
    name: 'gongfeng',
    displayName: '工蜂 MCP 服务',
    description: '通过 MCP 协议连接工蜂代码仓库，支持代码搜索、文件浏览、PR 管理、Issue 查询等操作，让 AI 智能体能够直接与工蜂平台交互。',
    version: '1.0.0',
    versions: ['1.0.0'],
    transport: 'sse',
    configJson: JSON.stringify({
      mcp: {
        servers: {
          gongfeng: {
            url: 'https://gongfeng.example.com/mcp/sse',
            transport: 'sse',
            headers: { 'Authorization': '<your-gongfeng-token>' },
            timeout: 60,
          },
        },
      },
    }, null, 2),
    usageDoc: '# 工蜂 MCP 使用说明\n\n## 前置条件\n\n1. 需要工蜂个人访问令牌（Private Token）\n2. 确保网络可访问工蜂服务\n\n## 使用方式\n\n将配置中的 `<your-gongfeng-token>` 替换为你的工蜂 Token 即可。',
    toolDoc: '# 工具列表\n\n## search_projects\n搜索工蜂项目\n\n## get_blob_content\n获取文件内容\n\n## create_merge_request\n创建合并请求',
    scope: 'public',
    groupIds: [],
    createdAt: new Date('2025-10-15'),
    updatedAt: new Date('2025-10-15'),
  },
  {
    name: 'iwiki',
    displayName: 'iWiki 文档服务',
    description: '连接 iWiki 知识库平台，支持文档搜索、内容获取、评论管理等操作，帮助 AI 智能体快速获取企业知识。',
    version: '2.1.0',
    versions: ['1.0.0', '2.0.0', '2.1.0'],
    transport: 'streamable-http',
    configJson: JSON.stringify({
      mcp: {
        servers: {
          iwiki: {
            url: 'https://iwiki.example.com/mcp',
            transport: 'streamable-http',
            headers: { 'Authorization': '<your-iwiki-token>' },
            timeout: 60,
          },
        },
      },
    }, null, 2),
    usageDoc: '# iWiki MCP 使用说明\n\n连接 iWiki 后可以搜索和获取企业文档内容。\n\n## 注意事项\n\n- 需要 iWiki 访问权限\n- Token 请从 iWiki 个人设置中获取',
    scope: 'public',
    groupIds: [],
    createdAt: new Date('2025-11-20'),
    updatedAt: new Date('2025-11-20'),
  },
  {
    name: 'filesystem',
    displayName: '本地文件系统',
    description: '通过 STDIO 方式连接本地文件系统 MCP 服务，支持文件读写、目录浏览等基础操作。',
    version: '1.2.0',
    versions: ['1.0.0', '1.1.0', '1.2.0'],
    transport: 'stdio',
    configJson: JSON.stringify({
      mcp: {
        servers: {
          filesystem: {
            command: 'npx',
            args: ['-y', '@anthropic-ai/mcp-filesystem'],
            transport: 'stdio',
            env: { HOME: '/home/user' },
            timeout: 30,
          },
        },
      },
    }, null, 2),
    scope: 'public',
    groupIds: [],
    createdAt: new Date('2025-12-01'),
    updatedAt: new Date('2025-12-01'),
  },
  {
    name: 'tapd',
    displayName: 'TAPD 项目管理',
    description: '连接 TAPD 项目管理平台，支持需求查询、缺陷管理、迭代跟踪、任务操作等功能，让 AI 智能体可以直接参与项目管理流程。支持按语义搜索需求和缺陷，自动创建和更新工作项。',
    version: '1.0.0',
    versions: ['0.9.0', '1.0.0'],
    transport: 'sse',
    configJson: JSON.stringify({
      mcp: {
        servers: {
          tapd: {
            url: 'https://tapd.example.com/mcp/sse',
            transport: 'sse',
            headers: { 'Authorization': 'Bearer <your-tapd-token>' },
            timeout: 90,
          },
        },
      },
    }, null, 2),
    usageDoc: '# TAPD MCP 使用说明\n\n## 快速开始\n\n1. 获取 TAPD API Token\n2. 将 Token 填入配置\n3. 连接后即可使用\n\n## 常用操作\n\n- 查询需求列表\n- 创建/更新缺陷\n- 查看迭代进度',
    toolDoc: '# 工具列表\n\n## stories_get\n查询需求列表\n\n| 参数 | 类型 | 说明 |\n|------|------|------|\n| workspace_id | string | 项目ID |\n| status | string | 状态 |\n\n## bugs_create\n创建缺陷\n\n## iterations_get\n查询迭代信息',
    scope: 'public',
    groupIds: [],
    createdAt: new Date('2025-09-28'),
    updatedAt: new Date('2025-10-05'),
  },
];

export const mcpStore = createLibraryStore<MCPService>({
  cacheKey: MCP_CACHE_KEY,
  versionKey: MCP_CACHE_VERSION_KEY,
  version: MCP_CACHE_VERSION,
  initialData: MOCK_MCPS,
  getId: (m) => m.name,
  eventName: MCP_STORE_EVENT,
  reviver: (m: any): MCPService => ({
    ...m,
    createdAt: new Date(m.createdAt),
    updatedAt: new Date(m.updatedAt),
  }),
});

/**
 * ClawPro 平台 MCP 的 mock 初始数据
 *
 * 第一期：9 个分组（管控端 8 组 + 用户端 1 组），共 53 个工具。
 * - 管控端（roleScope='admin'）：走 /admin/* 接口，41 个
 * - 用户端（roleScope='member'）：走 /openclaw/* 接口，12 个
 */

import type {
  Capability,
  CapabilityToggles,
  UserToken,
  DistributedAgent,
  AutoInstallPolicy,
  ModuleDef,
} from './types';

// ────────────────────────────────────────────────────────────
// 模块定义（去掉 M 编号，用 slug + 中文名）
// ────────────────────────────────────────────────────────────

export const MODULES: ModuleDef[] = [
  // ── 管控端（走 /admin/*）──
  { id: 'enterprise-skills', label: '企业技能库', description: '技能的增删查改、下发卸载', roleScope: 'admin' },
  { id: 'enterprise-rules', label: '企业规范库', description: '规范的增删查改、下发卸载', roleScope: 'admin' },
  { id: 'enterprise-mcp', label: '企业 MCP 库', description: 'MCP 服务的增删查改、下发', roleScope: 'admin' },
  { id: 'admin-instances', label: 'Agent 实例', description: '查看实例列表、状态、已装工具', roleScope: 'admin' },
  { id: 'usage-monitor', label: '用量监控', description: 'Token 用量统计与调用明细', roleScope: 'admin' },
  { id: 'audit-log', label: '操作审计', description: '查看操作审计日志', roleScope: 'admin' },
  { id: 'user-management', label: '用户管理', description: '用户列表、创建、更新、重置密码', roleScope: 'admin' },
  { id: 'unified-assets', label: '资产管理', description: '分组/项目级资产绑定与同步', roleScope: 'admin' },
  // ── 用户端（走 /openclaw/*）──
  { id: 'my-instances', label: '我的 Agent', description: '用户端 Agent 实例查询与生命周期操作（管理员和用户端角色均可用）', roleScope: 'member' },
];

// ────────────────────────────────────────────────────────────
// 能力清单（第一期，53 个工具：管控端 41 + 用户端 12）
// ────────────────────────────────────────────────────────────

export const CAPABILITIES: Capability[] = [

  // ═══ 企业技能库（管理员，10）═══
  { id: 'list_skills', label: '查看技能列表', description: '查看企业技能库中所有技能', category: 'query', risk: 'low', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'GET /admin/skills' },
  { id: 'get_skill_detail', label: '查看技能详情', description: '查看技能文件列表、安全扫描结果', category: 'query', risk: 'low', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'GET /admin/skills/detail' },
  { id: 'create_skill', label: '上传技能', description: '上传 ZIP 包创建新技能', category: 'distribute', risk: 'medium', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'POST /admin/skills/create' },
  { id: 'update_skill', label: '更新技能', description: '更新技能信息或上传新版本', category: 'config', risk: 'medium', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'POST /admin/skills/update' },
  { id: 'delete_skill', label: '删除技能', description: '从企业技能库删除技能', category: 'danger', risk: 'high', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'POST /admin/skills/delete' },
  { id: 'distribute_skill', label: '下发技能', description: '将技能下发到指定 Agent 实例', category: 'distribute', risk: 'medium', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'POST /admin/skills/distribute' },
  { id: 'uninstall_skill', label: '卸载技能', description: '从指定 Agent 实例卸载技能', category: 'distribute', risk: 'medium', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'POST /admin/skills/uninstall' },
  { id: 'download_skill', label: '下载技能', description: '下载技能 ZIP 包', category: 'query', risk: 'low', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'GET /admin/skills/download' },
  { id: 'get_skill_install_status', label: '查看技能安装情况', description: '查看技能已安装到哪些实例', category: 'query', risk: 'low', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'GET /admin/skills/instances' },
  { id: 'list_distribute_tasks', label: '查看下发任务', description: '查看下发/卸载任务状态', category: 'query', risk: 'low', phase: 1, module: 'enterprise-skills', roleScope: 'admin', backendApi: 'GET /admin/skills/tasks' },

  // ═══ 企业规范库（管理员，10）═══
  { id: 'list_standards', label: '查看规范列表', description: '查看企业规范库中所有规范', category: 'query', risk: 'low', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'GET /admin/rules' },
  { id: 'get_standard_detail', label: '查看规范详情', description: '查看规范详情', category: 'query', risk: 'low', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'GET /admin/rules/detail' },
  { id: 'create_standard', label: '创建规范', description: '创建新规范', category: 'config', risk: 'medium', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'POST /admin/rules/create' },
  { id: 'update_standard', label: '更新规范', description: '更新规范内容', category: 'config', risk: 'medium', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'POST /admin/rules/update' },
  { id: 'delete_standard', label: '删除规范', description: '删除企业规范', category: 'danger', risk: 'high', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'POST /admin/rules/delete' },
  { id: 'distribute_standard', label: '下发规范', description: '将规范下发到指定 Agent 实例', category: 'distribute', risk: 'medium', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'POST /admin/rules/distribute' },
  { id: 'uninstall_standard', label: '卸载规范', description: '从指定 Agent 实例卸载规范', category: 'distribute', risk: 'medium', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'POST /admin/rules/uninstall' },
  { id: 'get_standard_install_status', label: '查看规范安装情况', description: '查看规范已安装到哪些实例', category: 'query', risk: 'low', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'GET /admin/rules/instances' },
  { id: 'list_standard_tasks', label: '查看规范下发任务', description: '查看规范下发/卸载任务状态', category: 'query', risk: 'low', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'GET /admin/rules/tasks' },
  { id: 'get_standard_files', label: '查看规范文件列表', description: '查看规范包含的文件', category: 'query', risk: 'low', phase: 1, module: 'enterprise-rules', roleScope: 'admin', backendApi: 'GET /admin/rules/files' },

  // ═══ 企业 MCP 库（管理员，7）═══
  { id: 'list_mcp_services', label: '查看 MCP 列表', description: '查看企业 MCP 库中所有 MCP 服务', category: 'query', risk: 'low', phase: 1, module: 'enterprise-mcp', roleScope: 'admin', backendApi: 'GET /admin/mcp' },
  { id: 'get_mcp_detail', label: '查看 MCP 详情', description: '查看 MCP 版本、配置、安装情况', category: 'query', risk: 'low', phase: 1, module: 'enterprise-mcp', roleScope: 'admin', backendApi: 'GET /admin/mcp/detail' },
  { id: 'create_mcp_service', label: '创建 MCP', description: '创建新的 MCP 服务', category: 'config', risk: 'medium', phase: 1, module: 'enterprise-mcp', roleScope: 'admin', backendApi: 'POST /admin/mcp/create' },
  { id: 'update_mcp_meta', label: '更新 MCP 信息', description: '更新 MCP 名称、描述等元数据', category: 'config', risk: 'medium', phase: 1, module: 'enterprise-mcp', roleScope: 'admin', backendApi: 'POST /admin/mcp/meta' },
  { id: 'delete_mcp_service', label: '删除 MCP', description: '从企业 MCP 库删除 MCP 服务', category: 'danger', risk: 'high', phase: 1, module: 'enterprise-mcp', roleScope: 'admin', backendApi: 'POST /admin/mcp/delete' },
  { id: 'distribute_mcp_service', label: '下发 MCP', description: '将 MCP 下发到指定 Agent 实例', category: 'distribute', risk: 'medium', phase: 1, module: 'enterprise-mcp', roleScope: 'admin', backendApi: 'POST /admin/mcp/distribute' },
  { id: 'set_mcp_visibility', label: '设置 MCP 可见范围', description: '设置 MCP 对哪些分组可见', category: 'config', risk: 'medium', phase: 1, module: 'enterprise-mcp', roleScope: 'admin', backendApi: 'POST /admin/mcp/visibility' },

  // ═══ Agent 实例（管理员，3）═══
  { id: 'list_instances', label: '查看 Agent 列表', description: '查看所有 Agent 实例及状态', category: 'query', risk: 'low', phase: 1, module: 'admin-instances', roleScope: 'admin', backendApi: 'GET /admin/instances' },
  { id: 'get_instance_detail', label: '查看 Agent 详情', description: '查看实例状态、模型、通道等', category: 'query', risk: 'low', phase: 1, module: 'admin-instances', roleScope: 'admin', backendApi: 'GET /admin/instances/status' },
  { id: 'list_instance_skills', label: '查看 Agent 已装工具', description: '查看实例已安装的技能', category: 'query', risk: 'low', phase: 1, module: 'admin-instances', roleScope: 'admin', backendApi: 'GET /admin/instances/skills' },

  // ═══ 用量监控（管理员，2）═══
  { id: 'get_org_usage', label: '查看用量统计', description: '查看全企业 Token 用量统计', category: 'query', risk: 'low', phase: 1, module: 'usage-monitor', roleScope: 'admin', backendApi: 'GET /admin/usage/data' },
  { id: 'get_usage_logs', label: '查看用量明细', description: '查看 Token 调用明细记录', category: 'query', risk: 'low', phase: 1, module: 'usage-monitor', roleScope: 'admin', backendApi: 'GET /admin/usage/logs' },

  // ═══ 操作审计（管理员，1）═══
  { id: 'query_audit_log', label: '查看操作记录', description: '查看操作审计日志', category: 'query', risk: 'low', phase: 1, module: 'audit-log', roleScope: 'admin', backendApi: 'GET /admin/audit' },

  // ═══ 用户管理（管理员，4）═══
  { id: 'list_users', label: '查看用户列表', description: '查看企业所有用户', category: 'query', risk: 'low', phase: 1, module: 'user-management', roleScope: 'admin', backendApi: 'GET /admin/users' },
  { id: 'create_user', label: '创建用户', description: '创建新用户账号', category: 'config', risk: 'medium', phase: 1, module: 'user-management', roleScope: 'admin', backendApi: 'POST /admin/create' },
  { id: 'update_user', label: '更新用户', description: '更新用户信息、配额', category: 'config', risk: 'medium', phase: 1, module: 'user-management', roleScope: 'admin', backendApi: 'POST /admin/update-user' },
  { id: 'reset_user_password', label: '重置密码', description: '重置用户密码', category: 'danger', risk: 'high', phase: 1, module: 'user-management', roleScope: 'admin', backendApi: 'POST /admin/reset-password' },

  // ═══ 资产管理（管理员，4）═══
  { id: 'get_asset_detail', label: '查看已绑定资产', description: '查询分组/项目已绑定的资产', category: 'query', risk: 'low', phase: 1, module: 'unified-assets', roleScope: 'admin', backendApi: 'GET /admin/assets/detail' },
  { id: 'get_asset_candidates', label: '查看资产候选列表', description: '查询可绑定的资产候选', category: 'query', risk: 'low', phase: 1, module: 'unified-assets', roleScope: 'admin', backendApi: 'GET /admin/assets/candidates' },
  { id: 'get_asset_versions', label: '查看资产版本历史', description: '查询资产版本变更记录', category: 'query', risk: 'low', phase: 1, module: 'unified-assets', roleScope: 'admin', backendApi: 'GET /admin/assets/versions' },
  { id: 'save_asset_version', label: '保存资产绑定', description: '保存资产绑定（全量替换，含同步模式）', category: 'config', risk: 'medium', phase: 1, module: 'unified-assets', roleScope: 'admin', backendApi: 'POST /admin/assets/save' },

  // ═══ 我的 Agent（用户端，12）═══
  { id: 'list_my_instances', label: '查看我的实例列表', description: '查看当前用户名下的实例列表', category: 'query', risk: 'low', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'GET /openclaw/list' },
  { id: 'get_instance_status', label: '查看实例运行状态', description: '查询指定实例的云服务器运行状态', category: 'query', risk: 'low', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'GET /openclaw/status' },
  { id: 'get_instance_config_overview', label: '查看实例配置总览', description: '查询实例分组配置总览（含技能/规范/MCP）', category: 'query', risk: 'low', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'GET /openclaw/config-overview' },
  { id: 'get_current_image', label: '查看当前镜像', description: '查询实例当前使用的镜像信息', category: 'query', risk: 'low', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'GET /openclaw/current-image' },
  { id: 'get_personal_space_status', label: '查看个人空间状态', description: '查询实例个人空间使用状态', category: 'query', risk: 'low', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'GET /openclaw/personal-space' },
  { id: 'get_version_info', label: '查看版本信息', description: '获取实例版本信息', category: 'query', risk: 'low', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'GET /openclaw/version' },
  { id: 'create_instance', label: '创建实例', description: '创建一个新 Agent 实例（会创建关联的云服务器）', category: 'config', risk: 'medium', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'POST /openclaw/create' },
  { id: 'start_instance', label: '开机实例', description: '开机指定实例的云服务器', category: 'ops', risk: 'medium', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'POST /openclaw/start' },
  { id: 'stop_instance', label: '关机实例', description: '关机指定实例的云服务器', category: 'ops', risk: 'medium', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'POST /openclaw/stop' },
  { id: 'reboot_instance', label: '重启实例', description: '重启指定实例的云服务器', category: 'ops', risk: 'medium', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'POST /openclaw/reboot' },
  { id: 'rename_instance', label: '修改实例名称', description: '修改指定实例的显示名称', category: 'config', risk: 'medium', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'POST /openclaw/rename' },
  { id: 'delete_instance', label: '删除实例', description: '删除指定实例，同时销毁关联的云服务器', category: 'danger', risk: 'high', phase: 1, module: 'my-instances', roleScope: 'member', backendApi: 'POST /openclaw/delete' },
];

// ────────────────────────────────────────────────────────────
// 默认能力开关
//
// 权限模型（管理员是超集）：
// - 管控端工具（走 /admin/*）：管理员可用；用户端不可用（Token 层 403）
// - 用户端工具（走 /openclaw/*）：管理员和用户端都可用（管理员也有自己的实例）
//
// 具体到默认值：
// - admin=true 对所有工具（管理员是超集，全部默认开）
// - member=true 仅对用户端工具（用户端只能用 /openclaw/*）
//
// 高风险工具默认亦开启，由 get_usage_guide 提示用户二次确认。
// ────────────────────────────────────────────────────────────

export const DEFAULT_CAPABILITY_TOGGLES: CapabilityToggles = CAPABILITIES.reduce(
  (acc, cap) => {
    if (cap.phase === 1) {
      acc[cap.id] = {
        admin: true, // 管理员是超集：所有工具默认开
        member: cap.roleScope === 'member', // 用户端仅能用用户端工具
      };
    } else {
      acc[cap.id] = { admin: false, member: false };
    }
    return acc;
  },
  {} as CapabilityToggles,
);

// ────────────────────────────────────────────────────────────
// 初始 Token 列表
// ────────────────────────────────────────────────────────────

export const INITIAL_USER_TOKENS: UserToken[] = [
  {
    userId: 'alice@acompany.com',
    userName: 'Alice',
    role: 'admin',
    token: 'hk-admin-ax8f-9f3c-7d11-mock',
    tokenMask: 'hk-***ax8f',
    status: 'active',
    createdAt: '2026-06-18 10:30:00',
    lastUsedAt: '2026-06-23 16:18:00',
  },
  {
    userId: 'lisi@a-company.com',
    userName: 'lisi',
    role: 'member',
    token: 'hk-member-zq3p-2a55-91bd-mock',
    tokenMask: 'hk-***zq3p',
    status: 'active',
    createdAt: '2026-06-20 14:10:00',
    lastUsedAt: '2026-06-23 15:42:00',
  },
];

// ────────────────────────────────────────────────────────────
// 初始已下发 Agent
// ────────────────────────────────────────────────────────────

export const INITIAL_DISTRIBUTED_AGENTS: DistributedAgent[] = [
  {
    agentId: 'oc-001',
    agentName: 'Alice 的运营助手',
    ownerUserId: 'alice@acompany.com',
    ownerUserName: 'Alice',
    ownerRole: 'admin',
    injectedTokenMask: 'hk-***ax8f',
    source: 'manual',
    distributedAt: '2026-06-21 11:20:00',
  },
  {
    agentId: 'oc-007',
    agentName: 'Alice 的测试 Agent',
    ownerUserId: 'alice@acompany.com',
    ownerUserName: 'Alice',
    ownerRole: 'admin',
    injectedTokenMask: 'hk-***ax8f',
    source: 'auto',
    distributedAt: '2026-06-22 09:15:00',
  },
  {
    agentId: 'oc-003',
    agentName: 'lisi 的工作助手',
    ownerUserId: 'lisi@a-company.com',
    ownerUserName: 'lisi',
    ownerRole: 'member',
    injectedTokenMask: 'hk-***zq3p',
    source: 'manual',
    distributedAt: '2026-06-22 14:05:00',
  },
];

// ────────────────────────────────────────────────────────────
// 默认自动装载策略
// ────────────────────────────────────────────────────────────

export const DEFAULT_AUTO_INSTALL_POLICY: AutoInstallPolicy = {
  enabled: false,
  scope: 'adminOnly',
  autoGenerateToken: true,
  excludeUserIds: [],
  excludeAgentIds: [],
  updatedAt: '2026-06-23 16:00:00',
};

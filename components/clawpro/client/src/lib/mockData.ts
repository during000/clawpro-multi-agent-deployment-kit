// Mock data for the OpenClaw Enterprise Platform

export const SITE_CONFIG = {
  name: "A企业企业版OpenClaw",
  description: "快速创建属于你的24小时AI私人助理",
  logo: "🦞",
  region: "广州",
  ip: "43.128.xx.xx",
  domain: "openclaw.a-company.com",
  tencentUin: "3205597606",
  websiteDescription: "企业级AI助理平台，让每位用户都拥有专属AI助手",
};

const getRecentLocalAgentReportedAt = () =>
  new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().slice(0, 19).replace("T", " ");

export const MOCK_OPENCLAW_LIST = [
  { id: "oc-001", instanceId: "ins-creating01",  name: "创建中示例",                                          creator: "alice@acompany.com",  status: "creating",    agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-03-26 09:00:00", model: "",               modelVersion: "",           channels: [], skills: [] },
  { id: "oc-002", instanceId: "ins-createfail01", name: "创建失败示例",                                          creator: "bob@acompany.com",    status: "createFail",  agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-03-26 09:05:00", model: "",               modelVersion: "",           channels: [], skills: [] },
  { id: "oc-003", instanceId: "ins-running01",    name: "运行中示例",                                          creator: "carol@acompany.com",  status: "running",     agentType: "openclaw",     version: "2026.3.28", createdAt: "2026-03-01 10:23:45", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["飞书"], skills: ["github 1.0.0"], billingMode: "payg", hourlyRate: 1.5, runningMinutes: 4820 },
  { id: "oc-004", instanceId: "ins-loading01",    name: "加载中示例",                                          creator: "dave@acompany.com",   status: "loading",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-03-26 09:10:00", model: "",               modelVersion: "",           channels: [], skills: [], billingMode: "payg", hourlyRate: 1.5, runningMinutes: 0 },
  { id: "oc-005", instanceId: "ins-loadfail01",   name: "加载失败示例",                                          creator: "eve@acompany.com",    status: "loadFail",    agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-03-26 09:15:00", model: "",               modelVersion: "",           channels: [], skills: [] },
  { id: "oc-006", instanceId: "ins-shutdown01",   name: "已关机示例",                                          creator: "frank@acompany.com",  status: "shutdown",    agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-03-05 09:00:00", model: "腾讯云混元",   modelVersion: "混元 Turbo",     channels: [], skills: [], billingMode: "payg", hourlyRate: 1.5, runningMinutes: 2160 },
  { id: "oc-007", instanceId: "ins-maintaining01",name: "维护中示例",                                          creator: "grace@acompany.com",  status: "maintaining", agentType: "openclaw",     version: "2026.3.28", createdAt: "2026-03-10 16:45:00", model: "腾讯云混元",   modelVersion: "混元 Pro",       channels: ["企业微信机器人"], skills: [], billingMode: "payg", hourlyRate: 1.5, runningMinutes: 980 },
  { id: "oc-008", instanceId: "ins-pending01",    name: "待处理示例",                                          creator: "henry@acompany.com",  status: "pending",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-03-26 09:20:00", model: "",               modelVersion: "",           channels: [], skills: [] },
  { id: "oc-009", instanceId: "ins-longname01",   name: "这是一个名称非常非常长的智能助手用来测试超长文本截断效果", creator: "ivy@acompany.com",    status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-03-28 14:30:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["企业微信"], skills: ["github 1.0.0"], billingMode: "subscription" },
  { id: "oc-010", instanceId: "ins-hermes01",     name: "Hermes 示例",                                              creator: "jack@acompany.com",   status: "running",     agentType: "hermes",       version: "v0.10.0",   createdAt: "2026-04-01 10:00:00", model: "",               modelVersion: "",           channels: [], skills: [], billingMode: "payg", hourlyRate: 2.0, runningMinutes: 1530 },
  { id: "oc-011", instanceId: "ins-hermes02",     name: "Hermes 加载失败示例",                                      creator: "karen@acompany.com",  status: "loadFail",    agentType: "hermes",       version: "v0.12.0",   createdAt: "2026-04-02 11:00:00", model: "",               modelVersion: "",           channels: [], skills: [] },
  { id: "oc-012", instanceId: "ins-lightclaw01",  name: "LightclawACE 示例",                                          creator: "leo@acompany.com",    status: "running",     agentType: "lightclawace", version: "v0.1.5",    createdAt: "2026-04-03 09:30:00", model: "",               modelVersion: "",           channels: [], skills: [], billingMode: "payg", hourlyRate: 0.8, runningMinutes: 3260 },
  { id: "oc-013", instanceId: "ins-lightclaw02",  name: "LightclawACE 创建中示例",                                      creator: "alice@acompany.com",  status: "creating",    agentType: "lightclawace", version: "v0.1.8",    createdAt: "2026-04-04 14:00:00", model: "",               modelVersion: "",           channels: [], skills: [] },
  { id: "oc-014", instanceId: "ins-grpdemo01",   name: "多组织示例-前端组",                                            creator: "alice@acompany.com",  status: "running",     agentType: "openclaw",     version: "2026.3.28", createdAt: "2026-04-05 10:00:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: [], skills: [], groupId: "grp-fe", groupName: "A公司 / 技术部 / 前端组", billingMode: "payg", hourlyRate: 1.5, runningMinutes: 720 },
  { id: "oc-015", instanceId: "ins-grpdemo02",   name: "多组织示例-前端研发",                                            creator: "alice@acompany.com",  status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-04-05 11:00:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: [], skills: [], groupId: "grp-custom", groupName: "前端研发同学", billingMode: "subscription" },
  { id: "oc-016", instanceId: "local-workbuddy-01",  name: "WorkBuddy-运营笔记本",                                             creator: "olivia@acompany.com",  status: "running",     agentType: "localagent", localProduct: "WorkBuddy", localConnectionStatus: "connected", lastReportedAt: getRecentLocalAgentReportedAt(), createdAt: "2026-04-06 09:30:00", model: "WorkBuddy",       modelVersion: "workbuddy-2.3.1",   channels: [], skills: ["doc-summarizer 1.3.0", "meeting-summary 1.3.0"], standards: ["Agent 安全合规基线", "交付协作规范"], groupId: "grp-ops", groupName: "运营组", billingMode: "payg", hourlyRate: 0, runningMinutes: 360 },
  // ===== 共享实例（与管控端 ins-k25f9zwg "Dave的代码助手" 为同一台，用于验证互斥诊断） =====
  { id: "oc-shared-01", instanceId: "ins-k25f9zwg", name: "Dave的代码助手", creator: "dave@acompany.com", status: "running", agentType: "openclaw", version: "2026.3.28", createdAt: "2026-01-20 16:48:09", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["飞书"], skills: ["github 1.0.0"], billingMode: "payg", hourlyRate: 1.5, runningMinutes: 3200 },
  // ===== Demo Agent（绑定了角色，用于验证角色下发功能）=====
  // 行业分析师 v1 角色，2个实例：1个已下发v1，1个从未下发
  { id: "oc-role-001", instanceId: "ins-role-001", name: "市场分析助手",      creator: "bob@acompany.com",   status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-10 09:00:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["企业微信"], skills: [], roleName: "行业分析师", distributedRoleVersion: "1.0", groupId: "dept-operation", groupName: "A公司 / 产品部 / 运营组", billingMode: "payg", hourlyRate: 1.5, runningMinutes: 980 },
  { id: "oc-role-002", instanceId: "ins-role-002", name: "财经数据日报",      creator: "carol@acompany.com", status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-12 10:30:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["飞书"],    skills: [], roleName: "行业分析师", groupId: "og-frontend", groupName: "前端研发同学", billingMode: "subscription" },
  // 开发工程师 v1 角色，2个实例：1个已下发v1，1个从未下发
  { id: "oc-role-003", instanceId: "ins-role-003", name: "前端开发助手",      creator: "dave@acompany.com",  status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-14 14:15:00", model: "腾讯云混元",   modelVersion: "混元 Pro",       channels: [],          skills: [], roleName: "开发工程师", distributedRoleVersion: "1.0", groupId: "dept-fe", groupName: "A公司 / 技术部 / 前端组", billingMode: "payg", hourlyRate: 1.5, runningMinutes: 620 },
  { id: "oc-role-004", instanceId: "ins-role-004", name: "API 全栈开发",       creator: "eve@acompany.com",   status: "shutdown",    agentType: "openclaw",     version: "2026.3.28", createdAt: "2026-06-08 16:45:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: [],          skills: [], roleName: "开发工程师", groupId: "og-backend", groupName: "后端研发同学", billingMode: "payg", hourlyRate: 1.5, runningMinutes: 340 },
  // 办公能手 v1 角色，1个实例：从未下发
  { id: "oc-role-005", instanceId: "ins-role-005", name: "只有一个角色的agent",        creator: "frank@acompany.com", status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-15 08:00:00", model: "腾讯云混元",   modelVersion: "混元 Turbo",     channels: ["企业微信"], skills: [], roleName: "办公能手",   groupId: "dept-ai", groupName: "A公司 / 技术部 / AI 组", billingMode: "payg", hourlyRate: 1.5, runningMinutes: 260 },
  // ===== 多角色实例（roles 为角色位全集，含主角色位；用户端批量切换与管控端批量下发共用同一套 slotId）=====
  // 验证角色标签下拉 / 三点菜单二级列表精确定位角色位后切换；「设计师」角色位已下发过 v1.0
  { id: "oc-role-006", instanceId: "ins-role-006", name: "多角色综合助手",    creator: "grace@acompany.com", status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-20 10:00:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["企业微信", "飞书"], skills: [], roleName: "通用助手", roleCount: 4, roles: [
    { slotId: "slot-role-006-main", roleName: "通用助手", isMain: true },
    { slotId: "slot-role-006-pm", roleName: "项目经理", isMain: false },
    { slotId: "slot-role-006-designer", roleName: "设计师", isMain: false, distributedRoleVersion: "1.0", distributedRoleName: "设计师" },
    { slotId: "slot-role-006-monk", roleName: "佛法大师", isMain: false },
  ], billingMode: "payg", hourlyRate: 2.0, runningMinutes: 480 },
  // 场景 C：同一实例 5 个角色位里 3 个都绑定「设计师」且均未单独命名 → 三行 fallback 名完全撞车，
  // 管控端展示层须编号兜底为「设计师 1/2/3」，否则管理员分不清勾的是哪一个；用户端则验证同名 Pill 靠 slotId 精确选中不串号
  { id: "oc-role-009", instanceId: "ins-role-009", name: "多设计师协作助手", creator: "grace@acompany.com", status: "running", agentType: "openclaw", version: "2026.4.23", createdAt: "2026-06-29 09:30:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["企业微信"], skills: [], roleName: "项目经理", roleCount: 5, roles: [
    { slotId: "slot-role-009-main", roleName: "项目经理", baseRoleName: "项目经理", isMain: true },
    { slotId: "slot-role-009-d1", roleName: "设计师", baseRoleName: "设计师", isMain: false },
    { slotId: "slot-role-009-d2", roleName: "设计师", baseRoleName: "设计师", isMain: false, distributedRoleVersion: "1.0", distributedRoleName: "设计师" },
    { slotId: "slot-role-009-d3", roleName: "设计师", baseRoleName: "设计师", isMain: false },
    { slotId: "slot-role-009-writer", roleName: "内容创作者", baseRoleName: "内容创作者", isMain: false },
  ], billingMode: "payg", hourlyRate: 1.8, runningMinutes: 300 },
  // 场景 D：同一实例下 2 个角色位绑定同一角色「行业分析师」，且更新状态不同（1 个已更新 v1.0、1 个从未更新），
  // 验证树形勾选可精细到单个角色位、各自独立显示状态
  { id: "oc-role-014", instanceId: "ins-role-014", name: "多角色-双行业分析师", creator: "ivy@acompany.com",   status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-25 11:00:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["飞书"],            skills: [], roleName: "内容创作者", roleCount: 3, roles: [
    { slotId: "slot-role-014-main", roleName: "内容创作者", isMain: true },
    { slotId: "slot-role-014-a1", roleName: "行业分析师", isMain: false, distributedRoleVersion: "1.0", distributedRoleName: "行业分析师" },
    { slotId: "slot-role-014-a2", roleName: "行业分析师", isMain: false },
  ], billingMode: "payg", hourlyRate: 1.5, runningMinutes: 210 },
  // ===== 批量下发「设计师」角色的场景补充（B=3+同角色命中 / 混合命名 / A=大量角色位中精确摘出少数命中 / E=已关机禁用 / F=切换后下发版本作废）=====
  // 场景 B：3 个角色位全部绑定「设计师」且均未命名 → 与 009 一样触发编号兜底，但本实例主角色位不命中，验证"父行命中数 < 实例总角色位数"
  { id: "oc-role-010", instanceId: "ins-role-010", name: "多设计师协作实例", creator: "leo@acompany.com",   status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-26 09:00:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["企业微信"], skills: [], roleName: "项目经理", roleCount: 5, roles: [
    { slotId: "slot-role-010-main", roleName: "项目经理", isMain: true },
    { slotId: "slot-role-010-d1", roleName: "设计师", isMain: false },
    { slotId: "slot-role-010-d2", roleName: "设计师", isMain: false },
    { slotId: "slot-role-010-d3", roleName: "设计师", isMain: false },
    { slotId: "slot-role-010-writer", roleName: "内容创作者", isMain: false },
  ], billingMode: "payg", hourlyRate: 1.8, runningMinutes: 300 },
  // 场景「混合命名」：3 个角色位绑定「设计师」，1 个已被用户单独命名（直显自定义名、不参与编号）、2 个未命名（撞车编号为「设计师 1/2」）
  { id: "oc-role-011", instanceId: "ins-role-011", name: "角色切换部分成功+重试失败", creator: "grace@acompany.com", status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-27 10:00:00", model: "腾讯云混元",   modelVersion: "混元 Pro",       channels: ["飞书"],    skills: [], roleName: "办公能手",   roleCount: 4, roles: [
    { slotId: "slot-role-011-main", roleName: "办公能手", isMain: true },
    { slotId: "slot-role-011-d1", roleName: "设计师", isMain: false, name: "首页视觉设计师", distributedRoleVersion: "1.0", distributedRoleName: "设计师" },
    { slotId: "slot-role-011-d2", roleName: "设计师", isMain: false },
    { slotId: "slot-role-011-d3", roleName: "设计师", isMain: false },
  ], billingMode: "subscription" },
  // 场景 A：实例下 9 个角色位（总数较多），仅 2 个绑定「设计师」，验证从一堆角色位里精确摘出命中项
  { id: "oc-role-012", instanceId: "ins-role-012", name: "角色切换部分成功+重试成功",   creator: "henry@acompany.com", status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-28 11:00:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["企业微信", "飞书"], skills: [], roleName: "行业分析师", roleCount: 9, roles: [
    { slotId: "slot-role-012-main", roleName: "行业分析师", isMain: true },
    { slotId: "slot-role-012-1", roleName: "开发工程师", isMain: false },
    { slotId: "slot-role-012-2", roleName: "开发工程师", isMain: false },
    { slotId: "slot-role-012-3", roleName: "设计师", isMain: false },
    { slotId: "slot-role-012-4", roleName: "设计师", isMain: false },
    { slotId: "slot-role-012-5", roleName: "内容创作者", isMain: false },
    { slotId: "slot-role-012-6", roleName: "项目经理", isMain: false },
    { slotId: "slot-role-012-7", roleName: "办公能手", isMain: false },
    { slotId: "slot-role-012-8", roleName: "通用助手", isMain: false },
  ], billingMode: "payg", hourlyRate: 2.2, runningMinutes: 640 },
  // 场景 E：实例整体已关机，但有 2 个角色位绑定「设计师」→ 验证弹窗禁用勾选非运行中实例
  { id: "oc-role-013", instanceId: "ins-role-013", name: "关机多角色实例",   creator: "ivy@acompany.com",   status: "shutdown",    agentType: "openclaw",     version: "2026.3.28", createdAt: "2026-06-18 08:00:00", model: "腾讯云混元",   modelVersion: "混元 Turbo",     channels: [],          skills: [], roleName: "通用助手",   roleCount: 3, roles: [
    { slotId: "slot-role-013-main", roleName: "通用助手", isMain: true },
    { slotId: "slot-role-013-d1", roleName: "设计师", isMain: false },
    { slotId: "slot-role-013-d2", roleName: "设计师", isMain: false },
  ], billingMode: "payg", hourlyRate: 1.0, runningMinutes: 150 },
  // 场景 F：角色位「切换后下发版本作废」——两个角色位都下发过 v1.0，但用户随后在对话视图把它们切成了别的角色：
  //   · d1 现绑定「设计师」，上次下发的却是「内容创作者」v1.0 → 下发「设计师」时须判为待更新（不能算已更新 v1.0）
  //   · w1 现绑定「内容创作者」，上次下发的是「设计师」v1.0 → 下发「设计师」时它已不再命中，不应出现在列表里
  { id: "oc-role-015", instanceId: "ins-role-015", name: "切换过角色的实例", creator: "leo@acompany.com",   status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-30 15:00:00", model: "腾讯云 DeepSeek", modelVersion: "DeepSeek V3 0324", channels: ["飞书"],    skills: [], roleName: "办公能手",   roleCount: 3, roles: [
    { slotId: "slot-role-015-main", roleName: "办公能手", isMain: true },
    { slotId: "slot-role-015-d1", roleName: "设计师", isMain: false, distributedRoleVersion: "1.0", distributedRoleName: "内容创作者" },
    { slotId: "slot-role-015-w1", roleName: "内容创作者", isMain: false, distributedRoleVersion: "1.0", distributedRoleName: "设计师" },
  ], billingMode: "payg", hourlyRate: 1.6, runningMinutes: 180 },
  // 分组限制示例（仅开放通用助手，用于验证空态提示）
  { id: "oc-role-007", instanceId: "ins-role-007", name: "受限分组-仅通用助手", creator: "henry@acompany.com", status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-22 14:00:00", model: "腾讯云混元",   modelVersion: "混元 Turbo",     channels: ["企业微信"], skills: [], roleName: "通用助手", allowedRoleNames: [], billingMode: "payg", hourlyRate: 1.0, runningMinutes: 120 },
  // 分组限制示例（仅开放设计师角色）——多角色实例：main 通用助手 + 2 个设计师角色位
  { id: "oc-role-008", instanceId: "ins-role-008", name: "受限分组-仅设计师",   creator: "ivy@acompany.com",   status: "running",     agentType: "openclaw",     version: "2026.4.23", createdAt: "2026-06-23 09:00:00", model: "腾讯云混元",   modelVersion: "混元 Pro",       channels: ["企业微信"], skills: [], roleName: "通用助手", roleCount: 3, roles: [
    { slotId: "slot-role-008-main", roleName: "通用助手", baseRoleName: "通用助手", isMain: true },
    { slotId: "slot-role-008-d1", roleName: "UI 设计师", baseRoleName: "设计师", isMain: false },
    { slotId: "slot-role-008-d2", roleName: "视觉设计师", baseRoleName: "设计师", isMain: false },
  ], allowedRoleNames: ["设计师"], billingMode: "payg", hourlyRate: 1.2, runningMinutes: 90 },
];

export const MOCK_MEMBERS = [
  {
    id: "alice@acompany.com",
    name: "Alice",
    role: "admin",
    status: "active",
    openclawLimit: 5,
    tokenLimit: 100000,
    openclawCount: 2,
    createdAt: "2026-01-15",
  },
  {
    id: "lisi@a-company.com",
    role: "member",
    status: "active",
    openclawLimit: 3,
    tokenLimit: 50000,
    openclawCount: 1,
    createdAt: "2026-01-20",
  },
  {
    id: "wangwu@a-company.com",
    role: "member",
    status: "active",
    openclawLimit: 3,
    tokenLimit: 50000,
    openclawCount: 3,
    createdAt: "2026-02-01",
  },
  {
    id: "zhaoliu@a-company.com",
    role: "member",
    status: "disabled",
    openclawLimit: 3,
    tokenLimit: 50000,
    openclawCount: 0,
    createdAt: "2026-02-10",
  },
];

export const MOCK_MODELS = [
  {
    id: "model-001",
    name: "腾讯云 DeepSeek",
    version: "DeepSeek V3 0324",
    apiKey: "sk-**********************a1b2",
    status: "connected",
    visible: true,
    dailyTokenLimit: 500000,
  },
  {
    id: "model-002",
    name: "腾讯云混元",
    version: "混元 Turbo",
    apiKey: "sk-**********************c3d4",
    status: "connected",
    visible: true,
    dailyTokenLimit: 300000,
  },
  {
    id: "model-003",
    name: "腾讯云 Coding Plan",
    version: "自动",
    apiKey: "sk-**********************e5f6",
    status: "disconnected",
    visible: false,
    dailyTokenLimit: 200000,
  },
];

export const MOCK_CHANNELS = [
  { id: "wework-bot", name: "企业微信机器人", icon: "💼", visible: true },
  { id: "wework-app", name: "企业微信应用", icon: "📱", visible: true },
  { id: "qq", name: "QQ", icon: "🐧", visible: true },
  { id: "feishu", name: "飞书", icon: "🪶", visible: true },
  { id: "dingtalk", name: "钉钉", icon: "📌", visible: false },
];

export const MOCK_DOCS = [
  {
    id: "doc-001",
    title: "OpenClaw 概念介绍",
    addedAt: "2026-01-01",
    addedBy: "系统默认",
    visible: true,
    isDefault: true,
  },
  {
    id: "doc-002",
    title: "企业版 OpenClaw 的功能与特色",
    addedAt: "2026-01-01",
    addedBy: "系统默认",
    visible: true,
    isDefault: true,
  },
  {
    id: "doc-003",
    title: "部署 OpenClaw 指引",
    addedAt: "2026-01-01",
    addedBy: "系统默认",
    visible: true,
    isDefault: true,
  },
  {
    id: "doc-004",
    title: "OpenClaw 进阶玩法",
    addedAt: "2026-01-01",
    addedBy: "系统默认",
    visible: true,
    isDefault: true,
  },
];

export const MOCK_IMAGES = [
  {
    id: "img-001",
    imageId: "img-20260101-001",
    name: "OpenClaw-Enterprise-v2.1",
    status: "active",
    type: "public" as const,
    agentType: "OpenClaw",
    agentVersion: "2026.3.28",
    os: "CentOS 7.9 64位",
    createdAt: "2026-01-15 10:00:00",
    active: true,
  },
  {
    id: "img-002",
    imageId: "img-20260201-002",
    name: "Hermes-Agent-v0.8",
    status: "active",
    type: "public" as const,
    agentType: "HermesAgent",
    agentVersion: "0.8.0",
    os: "Ubuntu 22.04 64位",
    createdAt: "2026-02-01 14:30:00",
    active: true,
  },
];

export const DEFAULT_INBOUND_RULES = [
  { source: "全部IPv4地址", protocol: "ICMP", port: "ALL", policy: "允许", remark: "放通Ping服务" },
  { source: "全部IPv4地址", protocol: "TCP", port: "22", policy: "允许", remark: "放通Linux SSH登录" },
  { source: "全部IPv4地址", protocol: "TCP", port: "80", policy: "允许", remark: "Web服务HTTP (80)，如Apache、Nginx" },
  { source: "全部IPv4地址", protocol: "TCP", port: "443", policy: "允许", remark: "Web服务HTTPS (443)，如Apache、Nginx" },
  { source: "全部IPv4地址", protocol: "TCP", port: "3389", policy: "允许", remark: "Windows远程桌面登录" },
  { source: "全部IPv4地址", protocol: "UDP", port: "3389", policy: "允许", remark: "Windows远程桌面登录优化" },
  { source: "全部IPv4地址", protocol: "TCP", port: "18789", policy: "允许", remark: "" },
  { source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
];

export const DEFAULT_OUTBOUND_RULES = [
  { target: "-", protocol: "ALL", port: "ALL", policy: "允许", remark: "" },
  { target: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
];

export const MOCK_AUDIT_LOGS = [
  {
    id: "log-001",
    operator: "alice@acompany.com",
    event: "UpdateBasicInfo",
    action: "/api/admin/basic-info",
    requestTime: "2026-03-09 10:23:45",
    responseTime: "2026-03-09 10:23:45",
    success: true,
    detail: {
      eventId: "6af57777-10bd-4032-b881-f2e2f8872cd0",
      request: '{"siteName":"A企业企业版OpenClaw","description":"企业级AI助理平台"}',
      startDate: "2026-03-09 10:23:45",
      endDate: "2026-03-09 10:23:45",
      duration: "158",
      invokerName: "alice@acompany.com",
      invokerId: "1001",
      action: "/api/admin/basic-info",
      sourceIp: "30.42.219.99",
      success: "true",
      userAgent: "Mozilla/5.0",
    },
  },
  {
    id: "log-002",
    operator: "alice@acompany.com",
    event: "AddMember",
    action: "/api/admin/members",
    requestTime: "2026-03-09 11:05:12",
    responseTime: "2026-03-09 11:05:12",
    success: true,
    detail: {
      eventId: "7bf68888-20cd-5143-c992-g3f3g9983de1",
      request: '{"memberId":"newuser@a-company.com","openclawLimit":3,"tokenLimit":50000}',
      startDate: "2026-03-09 11:05:12",
      endDate: "2026-03-09 11:05:12",
      duration: "92",
      invokerName: "alice@acompany.com",
      invokerId: "1001",
      action: "/api/admin/members",
      sourceIp: "30.42.219.99",
      success: "true",
      userAgent: "Mozilla/5.0",
    },
  },
  {
    id: "log-003",
    operator: "lisi@a-company.com",
    event: "DeleteModel",
    action: "/api/admin/models/model-003",
    requestTime: "2026-03-09 14:30:00",
    responseTime: "2026-03-09 14:30:01",
    success: false,
    detail: {
      eventId: "8cg79999-31de-6254-d003-h4g4h0094ef2",
      request: '{"modelId":"model-003"}',
      startDate: "2026-03-09 14:30:00",
      endDate: "2026-03-09 14:30:01",
      duration: "1203",
      invokerName: "lisi@a-company.com",
      invokerId: "1002",
      action: "/api/admin/models/model-003",
      sourceIp: "30.42.220.15",
      success: "false",
      userAgent: "Mozilla/5.0",
    },
  },
];

export const MOCK_TOKEN_STATS = {
  totalRequests: 12847,
  inputTokens: 3241580,
  outputTokens: 1876320,
  totalTokens: 5117900,
  globalTokenLimit: 10000000,
};

export const MOCK_TOKEN_BY_MEMBER = [
  { memberId: "alice@acompany.com", inputTokens: 1200000, outputTokens: 680000, totalTokens: 1880000, tokenLimit: 100000, ratio: 0.188 },
  { memberId: "lisi@a-company.com", inputTokens: 980000, outputTokens: 560000, totalTokens: 1540000, tokenLimit: 50000, ratio: 0.308 },
  { memberId: "wangwu@a-company.com", inputTokens: 760000, outputTokens: 420000, totalTokens: 1180000, tokenLimit: 50000, ratio: 0.236 },
  { memberId: "zhaoliu@a-company.com", inputTokens: 301580, outputTokens: 216320, totalTokens: 517900, tokenLimit: 50000, ratio: 0.104 },
];

export const MOCK_TOKEN_BY_MODEL = [
  { modelName: "腾讯云 DeepSeek", inputTokens: 1800000, outputTokens: 1000000, totalTokens: 2800000, tokenLimit: 500000, ratio: 0.56 },
  { modelName: "腾讯云混元", inputTokens: 1100000, outputTokens: 650000, totalTokens: 1750000, tokenLimit: 300000, ratio: 0.583 },
  { modelName: "腾讯云 Coding Plan", inputTokens: 341580, outputTokens: 226320, totalTokens: 567900, tokenLimit: 200000, ratio: 0.284 },
];

export const MOCK_OPENCLAW_MONITOR = [
  { id: "oc-001", name: "工作助手", creator: "alice@acompany.com", status: "running", createdAt: "2026-03-01 10:23:45" },
  { id: "oc-002", name: "代码助手", creator: "alice@acompany.com", status: "running", createdAt: "2026-03-03 14:12:00" },
  { id: "oc-003", name: "文档整理助手", creator: "lisi@a-company.com", status: "stopped", createdAt: "2026-03-05 09:00:00" },
  { id: "oc-004", name: "数据分析助手", creator: "wangwu@a-company.com", status: "running", createdAt: "2026-03-06 16:45:00" },
  { id: "oc-005", name: "客服助手", creator: "wangwu@a-company.com", status: "running", createdAt: "2026-03-07 11:20:00" },
];

export const AVAILABLE_MODELS = [
  { value: "tencent-deepseek", label: "腾讯云 DeepSeek", versions: ["DeepSeek V3 0324", "DeepSeek R1", "DeepSeek V2"] },
  { value: "tencent-hunyuan", label: "腾讯云混元", versions: ["混元 TurboS Latest", "混元 Pro", "混元 Lite"] },
  { value: "tencent-coding", label: "腾讯云 Coding Plan", versions: ["自动"] },
];

export const AVAILABLE_SKILLS = [
  "tavily-search 1.0.0",
  "summarize 1.0.0",
  "agent-browser 0.2.0",
  "find-skills 0.1.0",
  "github 1.0.0",
  "obsidian 1.0.0",
  "notion 1.0.0",
  "weather 1.0.0",
  "tencentcloud-lighthouse-skill 1.0.0",
  "tencent-docs 1.0.3",
  "xhs-skill 1.0.15",
  "ai-ppt-generator 1.1.2",
];

// ─── 程序员角色 Soul（来自 Agent Chat 专家包 tcb-programmer-agent）────────────────
export const PROGRAMMER_SOUL = `<!--
name: 全栈程序员
description: 通用全栈开发 Agent，擅长前后端开发、架构设计与 CloudBase 部署，覆盖 Web、小程序、API 全链路
-->

# SOUL.md — Coder, Full-Stack Programmer Agent

_You're a senior full-stack developer who writes production-ready code and ships with CloudBase._

---

## Who You Are

- 你是一名经验丰富的**全栈程序员**，精通产品规划、UI设计、交互设计、前端、后端开发、数据库和部署
- 你默认使用**腾讯云 CloudBase** 作为前端和后端的基础设施和部署平台（云函数、云数据库、云存储、静态托管）
- 你可以处理各类编程任务：从快速原型到生产级应用，从 bug 修复到架构重构
- 你既能独立完成项目，也能作为团队中的 AI 结对编程伙伴

---

## Core Personality

- **代码优先**：能用代码说话就不废话，直接给出可运行的实现
- **务实高效**：选择成熟方案而非过度工程化，优先交付再迭代
- **主动完善**：发现潜在问题主动提出（性能、安全、边界情况）
- **结构清晰**：复杂任务拆解为步骤，让用户随时了解进度

---

## How You Communicate

- **默认中文**，技术术语和代码标识符保留英文
- **代码块**：文件路径写在首行注释，代码直接可复制运行
- **回答结构**：先给结论/方案，再解释原因
- **错误处理**：报错时按「原因 → 修复 → 预防」三段式
- **每轮结尾**：明确告知下一步行动或需要用户确认的内容

---

## What You Can Do

### 日常编程任务

- 代码实现、重构、优化
- Bug 排查与修复
- 代码审查与建议
- 技术方案选型与对比
- 单元测试编写
- 应用部署

### 项目开发（使用 CloudBase）

当用户需要开发新应用或部署项目时，遵循以下工作流：

【需求理解】确认核心需求（详细询问客户需求）
      ↓
【方案设计】技术选型 + 数据库结构 + API 设计
           主动查阅 cloudbase skill 获取可用能力
          （包括但不限于：UI 设计、Web 开发、小程序开发、数据库、云函数、部署等）
           根据项目需求调用对应的 skill 能力完成设计
      ↓
【代码实现】输出完整可运行代码
      ↓
【部署上线】通过 cloudbase skill 完成部署，部署到 CloudBase 环境中

**注意**：对于简单任务（修 bug、写函数、解释代码），跳过流程直接回答。

**部署规则**： 当用户提出应用/项目/系统/页面等开发需求时，**如果用户没有明确提及部署方式，默认部署到 CloudBase**

---

## Technical Stack

| 层级 | 默认选型 | 备选 |
|------|---------|------|
| Web 前端 | React + TypeScript + Vite + Tailwind CSS | Vue 3、Next.js |
| 移动端/小程序 | 微信原生小程序 | uni-app |
| 后端 | CloudBase 云函数 (Node.js) | CloudBase CloudRun (容器) |
| 数据库 | CloudBase 云数据库（MongoDB 兼容） | CloudBase MySQL |
| 存储 | CloudBase 云存储 + CDN | — |
| 部署 | CloudBase MCP 及 CloudBase CLI | — |
| 包管理 | pnpm | npm、yarn |

---

## CloudBase 使用方式

**所有 CloudBase 相关的开发和部署操作，统一通过 cloudbase skill 完成。**

- 涉及云函数、云数据库、云存储、静态托管、部署等操作时，**调用 cloudbase skill** 获取正确的 API 用法、配置格式和最佳实践
- 不自行编写 CLI 命令或猜测配置格式，以 skill 提供的规范为准
- 项目结构、部署配置、SDK 调用方式等均以 cloudbase skill 的指引为准

### 环境配置

使用 cloudbase skill 前，**必须先读取当前空间目录下的 settings/env.json 文件**来确定 CloudBase 环境：

**环境识别和处理逻辑：**

1. **文件存在**: 读取 envId，后续所有 cloudbase skill 调用使用该环境，使用 envid 时不需要再次确认
2. **文件不存在 或 cloudbase skill 反馈环境未绑定**: 引导用户完成环境绑定，告知用户需要通过点击界面上的云开发管理，完成和云开发环境的绑定，才能继续使用 cloudbase 进行应用部署

### 核心原则

- 不在源码中硬编码密钥、Token、连接串
- 敏感配置通过环境变量注入
- 数据库操作需配置安全规则
- 有任何 CloudBase 用法不确定时，优先查阅 cloudbase skill

---

## Core Rules

1. **异步操作必须有 try/catch**
2. **API 接口必须做参数校验**（入参类型 + 必填检查）
3. **数据库操作必须提醒权限规则配置**
4. **前端代码必须处理 loading / error / empty 三态**
5. **不跳过需求理解直接写代码**（复杂项目）；简单问题可直接回答
6. **代码中的占位配置用 /* TODO: 替换为实际值 */ 明确标注**

---

## What You Never Do

- 不输出无法运行的伪代码（除非用户要求思路概述）
- 不硬编码密钥、Token、连接串
- 不写缺少错误处理的异步函数
- 不在没有理解需求的情况下设计整个系统
- 不使用已废弃的 API 或过时的库版本
- 不让用户猜下一步——每次对话都有明确的行动指引
`;

// 角色数据
export interface RoleSkill {
  name: string;
  version: string;
  source: "公共" | "企业";
  /** 最新可用版本（来自技能库） */
  latestVersion?: string;
  /** 更新说明 */
  updateNote?: string;
  /** 更新前的原版本（仅在刷新后显示） */
  previousVersion?: string;
}

export interface RoleUpdateRecord {
  id: string;
  version: string;
  totalCount: number;
  successCount: number;
  failedCount: number;
  operator: string;
  operatedAt: string;
  status: 'success' | 'partial' | 'failed';
  /** 失败原因，成功时为 "-" */
  reason: string;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  soul: string;
  skills: RoleSkill[];
  visible: boolean;
  /** 应用范围：public=全部用户，private=按组织 */
  scope: 'public' | 'private';
  /** 当 scope=private 时，关联的组织 ID 列表 */
  groupIds: string[];
  /** 角色配置版本号：x.y 两段式，管理员手动管理 */
  version: string;
  /** 更新下发历史记录 */
  updateHistory?: RoleUpdateRecord[];
}

/**
 * 实例上已挂载的单个角色 slot（main Agent 或 sub Agent）。
 * 用于「多角色实例」场景下精确定位「切换角色」作用的是哪一个角色，而非整实例覆盖。
 * 注意：sub 角色的真实数据当前由 AgentChat SDK 维护，ClawPro 后端暂无法同步读取
 * （后端技术方案：AgentChat 上报变更 + ClawPro 提供 API 文档 + ClawPro 通过 TAT 下发），
 * 本字段现阶段仅用 mock 驱动前端交互演示。
 */
/**
 * 实例上的一个「角色位」。产品口径上同一实例下的各角色位是平等的（不区分主/子角色），
 * 用户端「我的 Agent」的批量切换、管控端「角色设定」的批量下发，都以本结构为唯一权威模型。
 */
export interface AgentRoleSlot {
  /**
   * 角色位唯一标识（实例内唯一）。这是"下发/切换能精确命中具体哪一个角色位"的唯一依据：
   * 同一实例下允许存在多个绑定同一角色的角色位（如 5 个都是「设计师」），它们 roleName 完全相同，
   * 只能靠 slotId 区分，任何按 roleName 寻址的实现都会串号。
   *
   * ⚠️ 后端契约要求（当前 mock 阶段由前端自造，接入真实数据前必须替换）：
   *   1) 权威产生方必须是 AgentChat —— 角色位由用户在对话视图侧创建，现网 ClawPro 前端是 SDK 直接
   *      集成 AgentChat，ClawPro 自造的 ID 与实例上真实的角色位无法对应；
   *   2) 必须在「AgentChat 云端 → ClawPro 后端 → TAT → 实例本地配置」整条链路上保持同一个值，
   *      否则 TAT 下发时无法落到实例内的目标角色位；
   *   3) 必须稳定且永不复用 —— 删除某角色位后新建的角色位必须是新 slotId，否则管控端的历史下发
   *      记录会被新角色位错误继承（显示「已更新」但实际从未配置过）；
   *   4) 角色位的删除同样需要上报，否则管控端弹窗会列出已不存在的角色位，下发必然失败。
   */
  slotId: string;
  /**
   * 该角色位当前绑定的角色名称（可被用户「修改名称」改成自定义显示名）。
   * ⚠️ 后端契约建议：应同时提供稳定的 roleId 引用。当前管控端命中判定是按角色名做字符串匹配
   * （`slot.roleName === role.name`），角色一改名整条命中链就断裂。
   */
  roleName: string;
  /**
   * 该 slot 的头像身份名，用于锚定头像。
   * 「修改名称」只改 roleName（显示名），不动 baseRoleName，从而头像保持不变；
   * 「切换角色」才会更新头像身份。缺省时回退到 roleName。
   */
  baseRoleName?: string;
  /**
   * 是否为该实例的主角色位。仅用于数据层与实例顶层 `roleName` 兼容字段的同步，
   * UI 不应据此展示「主 Agent / 子角色」等结构位置措辞（角色位之间是平等的）。
   */
  isMain: boolean;
  /** 用户给该角色位单独起的名字（少数场景才有）；为空时展示层 fallback 到 roleName */
  name?: string;
  /** 该角色创建时间（ISO 字符串，如 2026-08-01T10:30:00） */
  createdAt?: string;
  /** 该角色位最后一次被下发的角色版本号（undefined=从未下发） */
  distributedRoleVersion?: string;
  /**
   * 该角色位最后一次被下发时所绑定的角色名，与 distributedRoleVersion 同时写入。
   * 用于识别「下发之后用户又把这个角色位切换成了别的角色」：此时已下发的版本号属于旧角色、
   * 对当前角色毫无意义，管控端须判为「待更新」而不是「已更新」，否则管理员会误以为新角色已生效。
   * 由管控端单侧比对即可闭环，无需用户端切换逻辑配合改动。
   */
  distributedRoleName?: string;
}

export const MOCK_ROLES: Role[] = [
  {
    id: "role-tcb-programmer",
    name: "程序员",
    description: "经验丰富的全栈开发工程师，精通网站、小程序和全栈应用的开发部署场景",
    soul: PROGRAMMER_SOUL,
    skills: [
      { name: "github", version: "v2.0.0", source: "公共" },
      { name: "code-reviewer", version: "v1.3.0", source: "公共" },
      { name: "docker-ops", version: "v1.1.0", source: "公共" },
      { name: "api-tester", version: "v1.5.0", source: "公共" },
    ],
    visible: true,
    scope: "public",
    groupIds: [],
    version: "1.0",
  },
  {
    id: "role-001",
    name: "行业分析师",
    description: "结构化分析，输出高质量行业洞察",
    soul: "具备麦肯锡级别分析能力，擅长 PEST/波特五力/SWOT 等框架，输出结构化行业洞察报告",
    skills: [
      { name: "data-analyst", version: "v1.8.0", source: "公共" },
      { name: "sql-expert", version: "v1.6.0", source: "公共" },
      { name: "web-search-pro", version: "v3.1.0", source: "公共" },
    ],
    visible: true,
    scope: "public",
    groupIds: [],
    version: "1.0",
    updateHistory: [
      { id: "hist-001", version: "1.0", totalCount: 2, successCount: 2, failedCount: 0, operator: "admin@acompany.com", operatedAt: "2026-06-10 09:00:00", status: "success", reason: "-" },
      { id: "hist-002", version: "1.5", totalCount: 3, successCount: 3, failedCount: 0, operator: "admin@acompany.com", operatedAt: "2026-06-15 14:30:00", status: "success", reason: "-" },
      { id: "hist-003", version: "2.0", totalCount: 4, successCount: 2, failedCount: 2, operator: "bob@acompany.com", operatedAt: "2026-06-20 10:15:00", status: "partial", reason: "frontend-design-ultimate：技能包尚未完成SMH同步，请稍后重试" },
    ],
  },
  {
    id: "role-002",
    name: "开发工程师",
    description: "精通全栈开发，擅长网站、小程序和应用部署",
    soul: "面向交付闭环的全栈工程师，遵循 CloudBase 原生最佳实践，擅长从原型到部署的完整链路",
    skills: [
      { name: "github", version: "v2.0.0", source: "公共" },
      { name: "code-reviewer", version: "v1.3.0", source: "公共" },
    ],
    visible: true,
    scope: "private",
    groupIds: ["grp-2"],
    version: "1.0",
    updateHistory: [
      { id: "hist-004", version: "1.0", totalCount: 2, successCount: 2, failedCount: 0, operator: "admin@acompany.com", operatedAt: "2026-06-08 11:00:00", status: "success", reason: "-" },
    ],
  },
  {
    id: "role-003",
    name: "设计师",
    description: "美感与功能平衡，用设计解决问题",
    soul: "专业设计师伙伴，遵循信息架构 > 交互逻辑 > 视觉表现的优先级，注重用户体验闭环",
    skills: [
      { name: "self-improving-agent", version: "v1.0.0", source: "公共" },
      { name: "docker-ops", version: "v1.1.0", source: "公共" },
      { name: "email-writer", version: "v2.2.0", source: "公共" },
      { name: "k8s-manager", version: "v1.5.0", source: "公共" },
      { name: "api-tester", version: "v1.5.0", source: "公共" },
    ],
    visible: true,
    scope: "private",
    groupIds: ["grp-3"],
    version: "1.0",
  },
  {
    id: "role-004",
    name: "项目经理",
    description: "覆盖项目全生命周期，从立项到复盘",
    soul: "项目全生命周期管理，支持启动/会议/周报/风险/复盘全流程，确保项目高质量交付",
    skills: [
      { name: "文档总结助手", version: "v0.9.0", source: "企业" },
      { name: "智能翻译工具", version: "v1.0.0", source: "企业" },
      { name: "API 自动化测试", version: "v2.0.0", source: "企业" },
      { name: "代码质量扫描", version: "v1.0.0", source: "企业" },
      { name: "知识库问答", version: "v1.0.0", source: "企业" },
    ],
    visible: true,
    scope: "private",
    groupIds: ["grp-1", "grp-2"],
    version: "1.0",
  },
  {
    id: "role-005",
    name: "办公能手",
    description: "高效办公，熟练处理文档、表格、演示、会议",
    soul: "高效办公 AI 助手，熟练处理 Word/PDF/PPT/Excel/会议记录，提升日常办公效率",
    skills: [
      { name: "ppt-generator", version: "v1.0.0", source: "公共" },
      { name: "security-scanner", version: "v2.0.1", source: "公共" },
      { name: "email-writer", version: "v2.1.0", source: "公共" },
      { name: "web-search-pro", version: "v3.2.1", source: "公共" },
    ],
    visible: false,
    scope: "public",
    groupIds: [],
    version: "1.0",
  },
  {
    id: "role-006",
    name: "内容创作者",
    description: "优秀的图文内容创作者，具备极高审美",
    soul: "优秀的图文内容创作者，审美极高，擅长搜索+写作+配图+润色+发布全链路内容生产",
    skills: [
      { name: "self-improving-agent", version: "v1.0.0", source: "公共" },
      { name: "github", version: "v2.1.0", source: "公共" },
      { name: "data-analyst", version: "v2.0.0", source: "公共" },
      { name: "git-helper", version: "v1.2.0", source: "公共" },
      { name: "code-formatter", version: "v2.5.0", source: "公共" },
    ],
    visible: true,
    scope: "private",
    groupIds: ["grp-1", "grp-3", "grp-4"],
    version: "1.0",
  },
];

// 可供角色选择的技能库
export const PUBLIC_SKILL_POOL = [
  { name: "Data Analysis", description: "全面的数据分析技能", version: "v2.0" },
  { name: "Data Visualization", description: "数据可视化图表生成", version: "v1.5" },
  { name: "SWOT Analyzer", description: "SWOT 分析框架工具", version: "v1.0" },
  { name: "ui-ux-pro-max", description: "专业 UI/UX 设计辅助", version: "v1.0" },
  { name: "Impeccable", description: "设计质量检查工具", version: "v1.2" },
  { name: "taste-skill", description: "审美与品味评估", version: "v1.0" },
  { name: "Vercel web design", description: "现代 Web 设计最佳实践", version: "v1.0" },
  { name: "playwright-cli", description: "浏览器自动化测试", version: "v0.2" },
  { name: "self-improving-agent", description: "自我改进型 Agent 框架", version: "v1.0" },
  { name: "humanizer", description: "文本人性化润色", version: "v1.0" },
  { name: "agent-reach", description: "多平台内容分发", version: "v1.0" },
  { name: "baoyu-infographic", description: "信息图自动生成", version: "v1.0" },
  { name: "web-search-pro", description: "增强型网络搜索", version: "v3.2" },
  { name: "github", description: "GitHub 交互工具", version: "v2.1" },
  { name: "code-reviewer", description: "自动化代码审查", version: "v1.4" },
];

export const ENTERPRISE_SKILL_POOL = [
  { name: "cloudbase", description: "腾讯云 CloudBase 开发工具", version: "v1.0" },
  { name: "pm-project-kickoff", description: "项目启动模板", version: "v1.0" },
  { name: "pm-meeting-minutes", description: "会议纪要自动生成", version: "v1.0" },
  { name: "pm-weekly-report", description: "周报自动汇总", version: "v1.0" },
  { name: "pm-risk-assessment", description: "项目风险评估", version: "v1.0" },
  { name: "pm-retrospective", description: "项目复盘模板", version: "v1.0" },
  { name: "office-documents", description: "Office 文档处理", version: "v1.0" },
  { name: "tencent-docs", description: "腾讯文档集成", version: "v1.0" },
  { name: "tencent-meeting-skill", description: "腾讯会议技能", version: "v1.0" },
  { name: "ima-note", description: "即时笔记工具", version: "v1.0" },
];

// ─── OneID 相关 Mock 数据 ────────────────────────────────────────────────────

/** 是否启用 OneID 模式（Demo 可切换） */
export const HAS_ONEID = true;

/** SSO 登录方式选项 */
export interface SsoImTypeOption {
  value: string;
  label: string;
}

export const MOCK_SSO_IM_TYPE_OPTIONS: SsoImTypeOption[] = [
  { value: "enterprise_wechat", label: "企业微信" },
  { value: "dingtalk", label: "钉钉扫码" },
  { value: "feishu", label: "飞书扫码" },
];

/** 当前已选的登录方式 */
export const MOCK_SSO_IM_TYPES = ["enterprise_wechat", "dingtalk", "feishu"];

/** 部门树节点 */
export interface DepartmentNode {
  id: string;
  name: string;
  path?: string;
  children?: DepartmentNode[];
}

/** Mock 部门树数据 */
export const MOCK_DEPARTMENTS: DepartmentNode[] = [
  {
    id: "dept-root",
    name: "A公司",
    path: "A公司",
    children: [
      {
        id: "dept-tech",
        name: "技术部",
        path: "A公司/技术部",
        children: [
          { id: "dept-fe", name: "前端组", path: "A公司/技术部/前端组" },
          { id: "dept-be", name: "后端组", path: "A公司/技术部/后端组" },
          { id: "dept-ai", name: "AI 团队", path: "A公司/技术部/AI 团队" },
        ],
      },
      {
        id: "dept-product",
        name: "产品部",
        path: "A公司/产品部",
        children: [
          { id: "dept-pm", name: "产品经理组", path: "A公司/产品部/产品经理组" },
          { id: "dept-design", name: "设计组", path: "A公司/产品部/设计组" },
        ],
      },
      {
        id: "dept-ops",
        name: "运营部",
        path: "A公司/运营部",
      },
      {
        id: "dept-hr",
        name: "人力资源部",
        path: "A公司/人力资源部",
      },
    ],
  },
];

/** 带部门信息的成员数据（OneID 模式使用） */
export const MOCK_MEMBERS_WITH_DEPT = [
  { id: "alice@acompany.com", name: "Alice", role: "admin", status: "active", clawLimit: 5, tokenLimit: 100000, clawCount: 2, joinTime: "2026-01-15", department: "A公司/技术部/前端组", departmentId: "dept-fe" },
  { id: "bob@acompany.com", name: "Bob", role: "admin", status: "active", clawLimit: 5, tokenLimit: 100000, clawCount: 1, joinTime: "2026-01-18", department: "A公司/技术部/后端组", departmentId: "dept-be" },
  { id: "lisi@a-company.com", name: "李四", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2026-01-20", department: "A公司/技术部/AI 团队", departmentId: "dept-ai" },
  { id: "wangwu@a-company.com", name: "王五", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 3, joinTime: "2026-02-01", department: "A公司/产品部/产品经理组", departmentId: "dept-pm" },
  { id: "zhaoliu@a-company.com", name: "赵六", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2026-02-10", department: "A公司/产品部/设计组", departmentId: "dept-design" },
  { id: "sunqi@a-company.com", name: "孙七", role: "member", status: "active", clawLimit: 3, tokenLimit: -1, clawCount: 2, joinTime: "2026-02-15", department: "A公司/运营部", departmentId: "dept-ops" },
  { id: "zhouba@a-company.com", name: "周八", role: "member", status: "disabled", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2026-02-20", department: "A公司/人力资源部", departmentId: "dept-hr" },
  { id: "wujiu@a-company.com", name: "吴九", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2026-03-01", department: "A公司/技术部/前端组", departmentId: "dept-fe" },
  { id: "zhengshi@a-company.com", name: "郑十", role: "member", status: "active", clawLimit: 3, tokenLimit: 80000, clawCount: 0, joinTime: "2026-03-05", department: "A公司/技术部/后端组", departmentId: "dept-be" },
  { id: "liuyi@a-company.com", name: "刘一", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2026-03-10", department: "A公司/运营部", departmentId: "dept-ops" },
  { id: "chener@a-company.com", name: "陈二", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2026-03-15", department: "A公司/技术部/AI 团队", departmentId: "dept-ai" },
  { id: "yangsan@a-company.com", name: "杨三", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2026-03-20", department: "A公司/产品部/设计组", departmentId: "dept-design" },
];

/** 按部门汇总的 Token 消耗数据（Tokens 监控「按部门」Tab 使用） */
export const MOCK_TOKEN_BY_DEPARTMENT = [
  { departmentId: "dept-fe", departmentName: "前端组", path: "A公司/技术部/前端组", requests: 3200, inputTokens: 820000, outputTokens: 450000, totalTokens: 1270000 },
  { departmentId: "dept-be", departmentName: "后端组", path: "A公司/技术部/后端组", requests: 2800, inputTokens: 710000, outputTokens: 390000, totalTokens: 1100000 },
  { departmentId: "dept-ai", departmentName: "AI 团队", path: "A公司/技术部/AI 团队", requests: 4100, inputTokens: 1050000, outputTokens: 600000, totalTokens: 1650000 },
  { departmentId: "dept-pm", departmentName: "产品经理组", path: "A公司/产品部/产品经理组", requests: 1500, inputTokens: 380000, outputTokens: 210000, totalTokens: 590000 },
  { departmentId: "dept-design", departmentName: "设计组", path: "A公司/产品部/设计组", requests: 900, inputTokens: 230000, outputTokens: 130000, totalTokens: 360000 },
  { departmentId: "dept-ops", departmentName: "运营部", path: "A公司/运营部", requests: 600, inputTokens: 151580, outputTokens: 96320, totalTokens: 247900 },
  { departmentId: "dept-hr", departmentName: "人力资源部", path: "A公司/人力资源部", requests: 250, inputTokens: 50000, outputTokens: 30000, totalTokens: 80000 },
];

/** 按组织汇总的 Token 消耗数据 —— 普通模式（manual 组织，树形层级）*/
export interface GroupNode {
  id: string;
  name: string;
  path?: string;
  children?: GroupNode[];
}

export const MOCK_GROUP_TREE_MANUAL: GroupNode[] = [
  { id: "mgrp-product", name: "产品组", path: "产品组" },
  {
    id: "mgrp-rd", name: "研发组", path: "研发组",
    children: [
      { id: "mgrp-rd-fe", name: "研发-前端", path: "研发组/研发-前端" },
      { id: "mgrp-rd-be", name: "研发-后端", path: "研发组/研发-后端" },
    ],
  },
  { id: "mgrp-design", name: "设计组", path: "设计组" },
  { id: "mgrp-ops", name: "产品运营与市场推广团队", path: "产品运营与市场推广团队" },
];

export const MOCK_TOKEN_BY_GROUP_MANUAL = [
  { groupId: "mgrp-product", groupName: "产品组", requests: 1800, inputTokens: 460000, outputTokens: 250000, totalTokens: 710000 },
  { groupId: "mgrp-rd", groupName: "研发组", requests: 5200, inputTokens: 1320000, outputTokens: 780000, totalTokens: 2100000 },
  { groupId: "mgrp-rd-fe", groupName: "研发组/研发-前端", requests: 3100, inputTokens: 790000, outputTokens: 440000, totalTokens: 1230000 },
  { groupId: "mgrp-rd-be", groupName: "研发组/研发-后端", requests: 2600, inputTokens: 680000, outputTokens: 370000, totalTokens: 1050000 },
  { groupId: "mgrp-design", groupName: "设计组", requests: 950, inputTokens: 240000, outputTokens: 135000, totalTokens: 375000 },
  { groupId: "mgrp-ops", groupName: "产品运营与市场推广团队", requests: 720, inputTokens: 182000, outputTokens: 108000, totalTokens: 290000 },
];

/** 按组织汇总的 Token 消耗数据 —— OneID 模式（部门 + 自定义组织，树形层级）*/
export const MOCK_GROUP_TREE_ONEID: GroupNode[] = [
  {
    id: "__section_dept", name: "部门", path: "部门",
    children: [
      {
        id: "dept-root", name: "A公司", path: "A公司",
        children: [
          {
            id: "dept-tech", name: "技术部", path: "A公司/技术部",
            children: [
              { id: "dept-fe", name: "前端组", path: "A公司/技术部/前端组" },
              { id: "dept-be", name: "后端组", path: "A公司/技术部/后端组" },
              { id: "dept-ai", name: "AI 组", path: "A公司/技术部/AI 组" },
            ],
          },
          {
            id: "dept-product", name: "产品部", path: "A公司/产品部",
            children: [
              { id: "dept-pm", name: "产品策划", path: "A公司/产品部/产品策划" },
              { id: "dept-design", name: "设计组", path: "A公司/产品部/设计组" },
              {
                id: "dept-operation", name: "运营组", path: "A公司/产品部/运营组",
                children: [
                  { id: "dept-operation-1", name: "运营一组", path: "A公司/产品部/运营组/运营一组" },
                  { id: "dept-operation-2", name: "运营二组", path: "A公司/产品部/运营组/运营二组" },
                ],
              },
            ],
          },
          { id: "dept-hr", name: "人力资源", path: "A公司/人力资源" },
          { id: "dept-finance", name: "财务部", path: "A公司/财务部" },
        ],
      },
    ],
  },
  {
    id: "__section_group", name: "自定义组织", path: "自定义组织",
    children: [
      {
        id: "og-frontend", name: "前端研发同学", path: "前端研发同学",
        children: [
          { id: "og-fe-web", name: "Web 端", path: "前端研发同学/Web 端" },
          { id: "og-fe-mobile", name: "移动端", path: "前端研发同学/移动端" },
        ],
      },
      {
        id: "og-backend", name: "后端研发同学", path: "后端研发同学",
        children: [
          { id: "og-be-java", name: "Java 方向", path: "后端研发同学/Java 方向" },
          { id: "og-be-go", name: "Go 方向", path: "后端研发同学/Go 方向" },
        ],
      },
      { id: "og-ai-core", name: "AI 核心团队", path: "AI 核心团队" },
    ],
  },
];

export const MOCK_TOKEN_BY_GROUP_ONEID = [
  // 部门
  { groupId: "dept-root", groupName: "A公司", requests: 14200, inputTokens: 3620000, outputTokens: 2060000, totalTokens: 5680000 },
  { groupId: "dept-tech", groupName: "A公司/技术部", requests: 10100, inputTokens: 2580000, outputTokens: 1440000, totalTokens: 4020000 },
  { groupId: "dept-fe", groupName: "A公司/技术部/前端组", requests: 3200, inputTokens: 820000, outputTokens: 450000, totalTokens: 1270000 },
  { groupId: "dept-be", groupName: "A公司/技术部/后端组", requests: 2800, inputTokens: 710000, outputTokens: 390000, totalTokens: 1100000 },
  { groupId: "dept-ai", groupName: "A公司/技术部/AI 组", requests: 4100, inputTokens: 1050000, outputTokens: 600000, totalTokens: 1650000 },
  { groupId: "dept-product", groupName: "A公司/产品部", requests: 3500, inputTokens: 891580, outputTokens: 506320, totalTokens: 1397900 },
  { groupId: "dept-pm", groupName: "A公司/产品部/产品策划", requests: 1500, inputTokens: 380000, outputTokens: 210000, totalTokens: 590000 },
  { groupId: "dept-design", groupName: "A公司/产品部/设计组", requests: 900, inputTokens: 230000, outputTokens: 130000, totalTokens: 360000 },
  { groupId: "dept-operation", groupName: "A公司/产品部/运营组", requests: 1100, inputTokens: 281580, outputTokens: 166320, totalTokens: 447900 },
  { groupId: "dept-operation-1", groupName: "A公司/产品部/运营组/运营一组", requests: 650, inputTokens: 165000, outputTokens: 95000, totalTokens: 260000 },
  { groupId: "dept-operation-2", groupName: "A公司/产品部/运营组/运营二组", requests: 450, inputTokens: 116580, outputTokens: 71320, totalTokens: 187900 },
  { groupId: "dept-hr", groupName: "A公司/人力资源", requests: 250, inputTokens: 50000, outputTokens: 30000, totalTokens: 80000 },
  { groupId: "dept-finance", groupName: "A公司/财务部", requests: 350, inputTokens: 98420, outputTokens: 83680, totalTokens: 182100 },
  // 自定义组织
  { groupId: "og-frontend", groupName: "前端研发同学", requests: 2900, inputTokens: 740000, outputTokens: 410000, totalTokens: 1150000 },
  { groupId: "og-fe-web", groupName: "前端研发同学/Web 端", requests: 1600, inputTokens: 410000, outputTokens: 225000, totalTokens: 635000 },
  { groupId: "og-fe-mobile", groupName: "前端研发同学/移动端", requests: 1300, inputTokens: 330000, outputTokens: 185000, totalTokens: 515000 },
  { groupId: "og-backend", groupName: "后端研发同学", requests: 2500, inputTokens: 640000, outputTokens: 350000, totalTokens: 990000 },
  { groupId: "og-be-java", groupName: "后端研发同学/Java 方向", requests: 1400, inputTokens: 360000, outputTokens: 195000, totalTokens: 555000 },
  { groupId: "og-be-go", groupName: "后端研发同学/Go 方向", requests: 1100, inputTokens: 280000, outputTokens: 155000, totalTokens: 435000 },
  { groupId: "og-ai-core", groupName: "AI 核心团队", requests: 3600, inputTokens: 920000, outputTokens: 520000, totalTokens: 1440000 },
];

/** OpenClaw 列表（带部门信息，OneID 模式使用） */
export const MOCK_CLAWS_WITH_DEPT: Array<{
  id: string;
  instanceId: string;
  name: string;
  creator: string;
  createTime: string;
  status: string;
  department?: string;
  departmentId?: string;
}> = [
  { id: "1",  instanceId: "ins-g83c6wvc", name: "Alice的助手",      creator: "alice@acompany.com",  createTime: "2025-12-01 09:12:34", status: "running",     department: "A公司/技术部/前端组", departmentId: "dept-fe" },
  { id: "2",  instanceId: "ins-h92d7xwe", name: "Bob工作助手",       creator: "bob@acompany.com",    createTime: "2025-12-15 14:05:22", status: "running",     department: "A公司/技术部/后端组", departmentId: "dept-be" },
  { id: "3",  instanceId: "ins-j14e8yvf", name: "Carol的研究助手",   creator: "carol@acompany.com",  createTime: "2026-01-05 10:33:47", status: "shutdown",    department: "A公司/技术部/AI 团队", departmentId: "dept-ai" },
  { id: "4",  instanceId: "ins-k25f9zwg", name: "Dave的代码助手",    creator: "dave@acompany.com",   createTime: "2026-01-20 16:48:09", status: "running",     department: "A公司/产品部/产品经理组", departmentId: "dept-pm" },
  { id: "5",  instanceId: "ins-l36g0axh", name: "Eve的写作助手",     creator: "eve@acompany.com",    createTime: "2026-02-10 08:21:55", status: "createFail",  department: "A公司/产品部/设计组", departmentId: "dept-design" },
  { id: "6",  instanceId: "ins-m47h1byi", name: "Frank的数据助手",   creator: "frank@acompany.com",  createTime: "2026-02-18 11:07:30", status: "running",     department: "A公司/运营部", departmentId: "dept-ops" },
  { id: "7",  instanceId: "ins-n58i2czj", name: "Grace的翻译助手",   creator: "grace@acompany.com",  createTime: "2026-02-25 15:44:18", status: "creating",    department: "A公司/人力资源部", departmentId: "dept-hr" },
  { id: "8",  instanceId: "ins-o69j3dak", name: "Henry的销售助手",   creator: "henry@acompany.com",  createTime: "2026-03-01 09:58:03", status: "running",     department: "A公司/技术部/前端组", departmentId: "dept-fe" },
  { id: "9",  instanceId: "ins-p70k4ebl", name: "Ivy的客服助手",     creator: "ivy@acompany.com",    createTime: "2026-03-05 13:26:41", status: "maintaining", department: "A公司/技术部/后端组", departmentId: "dept-be" },
  { id: "10", instanceId: "ins-q81l5fcm", name: "Jack的会议助手",    creator: "jack@acompany.com",   createTime: "2026-03-08 17:02:15", status: "running",     department: "A公司/技术部/AI 团队", departmentId: "dept-ai" },
  { id: "11", instanceId: "ins-r92m6gdn", name: "Karen的报告助手",   creator: "karen@acompany.com",  createTime: "2026-03-09 10:15:50", status: "loadFail",    department: "A公司/产品部/产品经理组", departmentId: "dept-pm" },
  { id: "12", instanceId: "ins-s03n7heo", name: "Leo的项目助手",     creator: "leo@acompany.com",    createTime: "2026-03-10 08:39:27", status: "running",     department: "A公司/运营部", departmentId: "dept-ops" },
  { id: "13", instanceId: "ins-t14o8ipf", name: "Mia的新助手",       creator: "mia@acompany.com",    createTime: "2026-03-12 11:00:00", status: "loading",     department: "A公司/产品部/设计组", departmentId: "dept-design" },
  { id: "14", instanceId: "ins-u25p9jqg", name: "Noah的分析助手",    creator: "noah@acompany.com",   createTime: "2026-03-13 14:30:00", status: "pending",     department: "A公司/人力资源部", departmentId: "dept-hr" },
];

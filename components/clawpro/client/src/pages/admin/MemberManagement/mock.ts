/**
 * MemberManagement 子模块 mock 数据（PRD v2.0）
 *
 * 组织体系（OneID 模式演示）：
 *   - OneID 组织架构（源 oneid-dept，只读，多层级）：
 *       dept-root
 *         ├─ dept-tech
 *         │    ├─ dept-fe
 *         │    ├─ dept-be
 *         │    └─ dept-ai
 *         ├─ dept-product
 *         │    ├─ dept-pm
 *         │    └─ dept-design
 *         ├─ dept-hr
 *         └─ dept-finance
 *   - OneID 用户组（源 oneid-group，只读，单层）：
 *       og-frontend / og-backend / og-ai-core
 *   - ClawPro 自建（源 manual，普通模式唯一；OneID 模式里**不存在**此类，mock 提供一个示例说明历史切换场景）
 *
 * 为了覆盖核心业务场景：
 *   - 用户多归属（OneID 同时在多个部门和多个用户组）
 *   - 多组织对同唯一型资源冲突（VPC）
 *   - 主部门失效（primaryGroupValid=false）
 */
import type {
  UserGroup,
  UserOrg,
  ResourceItem,
  UserOverrideInfo,
  EffectiveConfig,
  ConfigEntry,
  ConfigCategory,
  AnomalousGroup,
  SyncResult,
} from "./types";

// ─── 组织（多层级 + 多来源） ──────────────────────────────
export const MOCK_GROUPS: UserGroup[] = [
  // ── OneID 组织架构（只读） ──
  {
    id: "dept-root",
    name: "A公司",
    parentId: null,
    source: "oneid-dept",
    readonly: true,
    externalId: "dept-root",
    syncBatchId: "oneid-org",
    createdAt: "2025-01-01",
  },
  { id: "dept-tech", name: "技术部", parentId: "dept-root", source: "oneid-dept", readonly: true, externalId: "dept-tech", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-fe", name: "前端架构与跨平台体验设计组", parentId: "dept-tech", source: "oneid-dept", readonly: true, externalId: "dept-fe", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-be", name: "后端组", parentId: "dept-tech", source: "oneid-dept", readonly: true, externalId: "dept-be", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-ai", name: "AI 组", parentId: "dept-tech", source: "oneid-dept", readonly: true, externalId: "dept-ai", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-product", name: "产品部", parentId: "dept-root", source: "oneid-dept", readonly: true, externalId: "dept-product", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-pm", name: "产品策划", parentId: "dept-product", source: "oneid-dept", readonly: true, externalId: "dept-pm", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-design", name: "设计组", parentId: "dept-product", source: "oneid-dept", readonly: true, externalId: "dept-design", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-operation", name: "运营组", parentId: "dept-product", source: "oneid-dept", readonly: true, externalId: "dept-operation", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-operation-1", name: "运营一组", parentId: "dept-operation", source: "oneid-dept", readonly: true, externalId: "dept-operation-1", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-operation-2", name: "运营二组", parentId: "dept-operation", source: "oneid-dept", readonly: true, externalId: "dept-operation-2", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-hr", name: "人力资源", parentId: "dept-root", source: "oneid-dept", readonly: true, externalId: "dept-hr", syncBatchId: "oneid-org", createdAt: "2025-01-01" },
  { id: "dept-finance", name: "财务部", parentId: "dept-root", source: "oneid-dept", readonly: true, externalId: "dept-finance", syncBatchId: "oneid-org", createdAt: "2025-01-01" },

  // ── OneID 用户组（管理员自建，多层级） ──
  { id: "og-frontend", name: "前端基础架构与工程效能研发协作组", parentId: null, source: "oneid-group", readonly: false, createdAt: "2025-03-01" },
  { id: "og-fe-web", name: "Web 端", parentId: "og-frontend", source: "oneid-group", readonly: false, createdAt: "2025-03-05" },
  { id: "og-fe-mobile", name: "移动端", parentId: "og-frontend", source: "oneid-group", readonly: false, createdAt: "2025-03-05" },
  { id: "og-backend", name: "后端研发同学", parentId: null, source: "oneid-group", readonly: false, createdAt: "2025-03-01" },
  { id: "og-be-java", name: "Java 方向", parentId: "og-backend", source: "oneid-group", readonly: false, createdAt: "2025-03-08" },
  { id: "og-be-go", name: "Go 方向", parentId: "og-backend", source: "oneid-group", readonly: false, createdAt: "2025-03-08" },
  { id: "og-ai-core", name: "AI 核心团队", parentId: null, source: "oneid-group", readonly: false, createdAt: "2025-03-01" },
];

// ─── 普通模式的自建组织（mock，供 hasOneid=false 场景使用） ──
export const MOCK_MANUAL_GROUPS: UserGroup[] = [
  { id: "mgrp-product", name: "产品组", parentId: null, source: "manual", readonly: false, createdAt: "2025-06-01" },
  { id: "mgrp-rd", name: "研发组", parentId: null, source: "manual", readonly: false, createdAt: "2025-06-05" },
  { id: "mgrp-rd-fe", name: "研发-前端", parentId: "mgrp-rd", source: "manual", readonly: false, createdAt: "2025-06-08" },
  { id: "mgrp-rd-be", name: "研发-后端", parentId: "mgrp-rd", source: "manual", readonly: false, createdAt: "2025-06-08" },
  { id: "mgrp-design", name: "设计组", parentId: null, source: "manual", readonly: false, createdAt: "2025-07-10" },
  { id: "mgrp-ops", name: "产品运营与市场推广团队", parentId: null, source: "manual", readonly: false, createdAt: "2025-08-15" },
  { id: "mgrp-qa", name: "质量保证组", parentId: null, source: "manual", readonly: false, createdAt: "2025-09-01" },
];

// ─── 用户（18 人） ───────────────────────────────────────
// OneID 模式：groupIds = OneID 部门（主+兼任）+ OneID 用户组
// primaryGroupId = OneID 主部门对应的组织 id
export const MOCK_USERS: UserOrg[] = [
  // ceo 直接挂在全公司根节点（高管，不隶属任何部门/小组）
  { userId: "ceo@acompany.com", displayName: "Ken (CEO)", groupIds: ["dept-root"], primaryGroupId: "dept-root", primaryGroupValid: true },
  // tim 技术部 VP（直挂在 dept-tech，不在任何小组）
  { userId: "tim@acompany.com", displayName: "Tim (技术 VP)", groupIds: ["dept-tech"], primaryGroupId: "dept-tech", primaryGroupValid: true },
  // peter 产品部 VP（直挂在 dept-product）
  { userId: "peter@acompany.com", displayName: "Peter (产品 VP)", groupIds: ["dept-product"], primaryGroupId: "dept-product", primaryGroupValid: true },
  // alice 前端组（主）+ 兼任 AI 组，加入 前端研发同学 用户组；参与 AI 中台建设项目-模型服务组
  { userId: "alice@acompany.com", displayName: "alice", groupIds: ["dept-fe", "dept-ai", "og-frontend", "proj-ai-platform-model"], primaryGroupId: "dept-fe", primaryGroupValid: true },
  // bob 后端组，加入 后端研发同学；参与 AI 中台建设项目-数据治理组
  { userId: "bob@acompany.com", displayName: "bob", groupIds: ["dept-be", "og-backend", "proj-ai-platform-data"], primaryGroupId: "dept-be", primaryGroupValid: true },
  // carol AI 组，加入 AI 核心团队 + 后端研发同学；同时参与 AI 中台建设项目 及 模型服务组（多项目）
  { userId: "carol@acompany.com", displayName: "carol", groupIds: ["dept-ai", "og-ai-core", "og-backend", "proj-ai-platform", "proj-ai-platform-model"], primaryGroupId: "dept-ai", primaryGroupValid: true },
  // david 产品策划 + 兼任运营一组
  { userId: "david@acompany.com", displayName: "david", groupIds: ["dept-pm", "dept-operation-1"], primaryGroupId: "dept-pm", primaryGroupValid: true },
  // eve 设计组 + 兼任运营一组
  { userId: "eve@acompany.com", displayName: "eve", groupIds: ["dept-design", "dept-operation-1"], primaryGroupId: "dept-design", primaryGroupValid: true },
  // frank 前端组 + 兼任后端组，前端+后端用户组；参与云原生迁移项目
  { userId: "frank@acompany.com", displayName: "frank", groupIds: ["dept-fe", "dept-be", "og-frontend", "og-backend", "proj-cloud-migration"], primaryGroupId: "dept-fe", primaryGroupValid: true },
  // grace 后端组；参与云原生迁移项目
  { userId: "grace@acompany.com", displayName: "grace", groupIds: ["dept-be", "og-backend", "proj-cloud-migration"], primaryGroupId: "dept-be", primaryGroupValid: true },
  // henry 人力资源 → 主部门失效（OneID 侧已删除）
  { userId: "henry@acompany.com", displayName: "henry", groupIds: [], primaryGroupId: "dept-hr", primaryGroupValid: false },
  // iris AI 组 + 加入 AI 核心团队 & 前端研发同学（冲突候选：og-ai-core vpc-ai 和 og-frontend vpc-fe）；参与增长实验项目
  { userId: "iris@acompany.com", displayName: "iris", groupIds: ["dept-ai", "og-ai-core", "og-frontend", "proj-growth"], primaryGroupId: "dept-ai", primaryGroupValid: true },
  // jack 财务部 + 兼任运营二组
  { userId: "jack@acompany.com", displayName: "jack", groupIds: ["dept-finance", "dept-operation-2"], primaryGroupId: "dept-finance", primaryGroupValid: true },
  // kate 前端组
  { userId: "kate@acompany.com", displayName: "kate", groupIds: ["dept-fe", "og-frontend"], primaryGroupId: "dept-fe", primaryGroupValid: true },
  // leo 产品策划 + 兼任设计组 + 运营一组
  { userId: "leo@acompany.com", displayName: "leo", groupIds: ["dept-pm", "dept-design", "dept-operation-1"], primaryGroupId: "dept-pm", primaryGroupValid: true },
  // mike 后端组 → 无用户组
  { userId: "mike@acompany.com", displayName: "mike", groupIds: ["dept-be"], primaryGroupId: "dept-be", primaryGroupValid: true },
  // nina 设计组 + 兼任前端组
  { userId: "nina@acompany.com", displayName: "nina", groupIds: ["dept-design", "dept-fe", "og-frontend"], primaryGroupId: "dept-design", primaryGroupValid: true },
  // oscar 财务部 + 兼任运营二组
  { userId: "oscar@acompany.com", displayName: "oscar", groupIds: ["dept-finance", "dept-operation-2"], primaryGroupId: "dept-finance", primaryGroupValid: true },
];

// ─── 项目（特殊组织，独立分组；source 均为 'project'） ──────
// 项目为单层级结构（不支持子项目），拥有成员 / 项目资产能力，并在「用户管理」与「项目资产管理」两页共享同步。
export const MOCK_PROJECTS: UserGroup[] = [
  { id: "proj-ai-platform", name: "AI 中台建设项目", parentId: null, source: "project", readonly: false, createdAt: "2026-01-10" },
  { id: "proj-ai-platform-model", name: "模型服务项目", parentId: null, source: "project", readonly: false, createdAt: "2026-01-12" },
  { id: "proj-ai-platform-data", name: "数据治理项目", parentId: null, source: "project", readonly: false, createdAt: "2026-01-12" },
  { id: "proj-cloud-migration", name: "云原生迁移项目", parentId: null, source: "project", readonly: false, createdAt: "2026-02-20" },
  { id: "proj-growth", name: "增长实验项目", parentId: null, source: "project", readonly: false, createdAt: "2026-03-05" },
];

// ─── 普通模式专用用户集（hasOneid=false） ─────────────────
// 仅用于演示自建组织：groupIds 仅引用 MOCK_MANUAL_GROUPS 的 id；
// primaryGroupId = null（普通模式没有主部门概念，列表列显示 "—"）
export const MOCK_USERS_MANUAL: UserOrg[] = [
  // ── 产品组（mgrp-product）：3 人 ──
  { userId: "anna@acompany.com", displayName: "Anna", groupIds: ["mgrp-product", "proj-ai-platform"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },
  { userId: "bill@acompany.com", displayName: "Bill", groupIds: ["mgrp-product"], primaryGroupId: null, primaryGroupValid: true, role: "admin", status: "active" },
  // cara 兼任设计组（用于验证 CreateAgentDialog 里 2 个分组的场景）+ 增长项目（验证项目归属）
  { userId: "cara@acompany.com", displayName: "Cara", groupIds: ["mgrp-product", "mgrp-design", "proj-growth"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },

  // ── 研发组（mgrp-rd）直挂：2 人（技术总监 + 架构师） ──
  { userId: "daniel@acompany.com", displayName: "Daniel (研发总监)", groupIds: ["mgrp-rd", "proj-ai-platform", "proj-cloud-migration"], primaryGroupId: null, primaryGroupValid: true, role: "admin", status: "active" },
  { userId: "eric@acompany.com", displayName: "Eric (架构师)", groupIds: ["mgrp-rd", "proj-cloud-migration"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },

  // ── 研发-前端（mgrp-rd-fe）：4 人 ──
  { userId: "fiona@acompany.com", displayName: "Fiona", groupIds: ["mgrp-rd-fe", "proj-ai-platform-model"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },
  { userId: "george@acompany.com", displayName: "George", groupIds: ["mgrp-rd-fe"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "disabled" },
  // helen 兼任产品组（跨 2 个平级分组，验证跨块归属）
  { userId: "helen@acompany.com", displayName: "Helen", groupIds: ["mgrp-rd-fe", "mgrp-product"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },
  { userId: "ivan@acompany.com", displayName: "Ivan", groupIds: ["mgrp-rd-fe"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },

  // ── 研发-后端（mgrp-rd-be）：3 人 ──
  { userId: "jason@acompany.com", displayName: "Jason", groupIds: ["mgrp-rd-be", "proj-ai-platform-data"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },
  { userId: "kelly@acompany.com", displayName: "Kelly", groupIds: ["mgrp-rd-be"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },
  // lucas 兼任前端，演示多归属
  { userId: "lucas@acompany.com", displayName: "Lucas", groupIds: ["mgrp-rd-be", "mgrp-rd-fe"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },

  // ── 设计组（mgrp-design）：2 人 ──
  { userId: "mia@acompany.com", displayName: "Mia", groupIds: ["mgrp-design"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },
  { userId: "nick@acompany.com", displayName: "Nick", groupIds: ["mgrp-design"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "disabled" },

  // ── 产品运营与市场推广团队（mgrp-ops）：3 人 ──
  // olivia 兼任设计组 + QA，共 3 个分组，验证 CreateAgentDialog 多分组换行渲染
  { userId: "olivia@acompany.com", displayName: "Olivia", groupIds: ["mgrp-ops", "mgrp-design", "mgrp-qa"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },
  { userId: "paul@acompany.com", displayName: "Paul", groupIds: ["mgrp-ops"], primaryGroupId: null, primaryGroupValid: true, role: "admin", status: "active" },
  // quinn 兼任研发-后端（用于运营↔研发跨大类分组验证）
  { userId: "quinn@acompany.com", displayName: "Quinn", groupIds: ["mgrp-ops", "mgrp-rd-be"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },

  // ── 质量保证组（mgrp-qa）：1 人（移出后无任何组织，原组织也无其他用户，存量实例处理弹窗的"回退"变体演示） ──
  { userId: "tina@acompany.com", displayName: "Tina", groupIds: ["mgrp-qa"], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },

  // ── 未分配组织：2 人（不在任何自建组织） ──
  { userId: "ryan@acompany.com", displayName: "Ryan", groupIds: [], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },
  { userId: "susan@acompany.com", displayName: "Susan", groupIds: [], primaryGroupId: null, primaryGroupValid: true, role: "member", status: "active" },
];

// ─── 资源池 ──────────────────────────────────────────────
export const MOCK_RESOURCES: ResourceItem[] = [
  // ── 模型（加法型） ──
  { id: "m-default", kind: "model", name: "腾讯云混元 - 混元 TurboS Latest", scope: { type: "all" }, createdAt: "2025-01-01" },
  { id: "m-tech", kind: "model", name: "Claude Sonnet 4", scope: { type: "filtered", groupIds: ["dept-tech"] }, createdAt: "2025-02-10" },
  { id: "m-fe", kind: "model", name: "GPT-4o", scope: { type: "filtered", groupIds: ["dept-fe"] }, createdAt: "2025-02-15" },
  { id: "m-ai", kind: "model", name: "DeepSeek V3", scope: { type: "filtered", groupIds: ["dept-ai", "og-ai-core"] }, createdAt: "2025-03-05" },
  { id: "m-product", kind: "model", name: "腾讯云 DeepSeek - DeepSeek V3 0324", scope: { type: "filtered", groupIds: ["dept-product"] }, createdAt: "2025-02-20" },

  // ── 通道（加法型） ──
  { id: "c-default", kind: "channel", name: "默认通道", scope: { type: "all" }, createdAt: "2025-01-01" },
  { id: "c-tech", kind: "channel", name: "技术部专用通道", scope: { type: "filtered", groupIds: ["dept-tech"] }, createdAt: "2025-02-10" },
  { id: "c-og-fe", kind: "channel", name: "前端研发同学 高级通道", scope: { type: "filtered", groupIds: ["og-frontend"] }, createdAt: "2025-03-01" },
  { id: "c-og-be", kind: "channel", name: "后端研发同学 专属通道", scope: { type: "filtered", groupIds: ["og-backend"] }, createdAt: "2025-03-05" },
  { id: "c-product", kind: "channel", name: "产品部通道", scope: { type: "filtered", groupIds: ["dept-product"] }, createdAt: "2025-02-20" },
  { id: "c-fe", kind: "channel", name: "前端组专用通道", scope: { type: "filtered", groupIds: ["dept-fe"] }, createdAt: "2025-02-15" },

  // ── 安全组（唯一型） ──
  { id: "sg-default", kind: "securityGroup", name: "sg-default-enterprise", scope: { type: "filtered", groupIds: ["dept-root"] }, isPlatformDefault: true, createdAt: "2025-01-01" },
  { id: "sg-tech", kind: "securityGroup", name: "sg-tech-internal", scope: { type: "filtered", groupIds: ["dept-tech"] }, createdAt: "2025-02-10" },
  { id: "sg-product", kind: "securityGroup", name: "sg-product-internal", scope: { type: "filtered", groupIds: ["dept-product"] }, createdAt: "2025-02-20" },
  { id: "sg-fe", kind: "securityGroup", name: "sg-frontend", scope: { type: "filtered", groupIds: ["dept-fe"] }, createdAt: "2025-02-15" },

  // ── VPC（唯一型；构造冲突） ──
  { id: "vpc-default", kind: "vpc", name: "vpc-default", scope: { type: "all" }, isPlatformDefault: true, createdAt: "2025-01-01" },
  { id: "vpc-og-ai", kind: "vpc", name: "vpc-ai-core", scope: { type: "filtered", groupIds: ["og-ai-core"] }, createdAt: "2025-03-20" },
  { id: "vpc-og-fe", kind: "vpc", name: "vpc-fe-team", scope: { type: "filtered", groupIds: ["og-frontend"] }, createdAt: "2025-02-25" },

  // ── 记忆（唯一型） ──
  { id: "mem-default", kind: "memory", name: "mem-default", scope: { type: "all" }, isPlatformDefault: true, createdAt: "2025-01-01" },

  // ── 镜像（唯一型） ──
  { id: "img-coder", kind: "image", name: "coder-v2", scope: { type: "all" }, isPlatformDefault: true, createdAt: "2025-01-01" },

  // ── 产品组专属配置（演示删除组织时无法删除） ──
  { id: "m-product-exclusive", kind: "model", name: "腾讯云混元 - 混元 Pro", scope: { type: "filtered", groupIds: ["mgrp-product"] }, createdAt: "2025-06-10" },
  { id: "c-product-exclusive", kind: "channel", name: "产品组专属通道", scope: { type: "filtered", groupIds: ["mgrp-product"] }, createdAt: "2025-06-10" },
  { id: "sg-product-exclusive", kind: "securityGroup", name: "sg-product-group", scope: { type: "filtered", groupIds: ["mgrp-product"] }, createdAt: "2025-06-10" },

  // ── 研发-后端专属模型（演示删除组织时无法删除） ──
  { id: "m-rd-be-exclusive", kind: "model", name: "DeepSeek V3", scope: { type: "filtered", groupIds: ["mgrp-rd-be"] }, createdAt: "2025-06-12" },

  // ── 云原生迁移项目专属配置（演示删除项目时因有专属配置而无法删除；项目目前仅支持「工具」） ──
  { id: "at-proj-cloud-migration", kind: "agentTool", name: "云原生迁移专属技能", scope: { type: "filtered", groupIds: ["proj-cloud-migration"] }, createdAt: "2026-02-22" },
];

// ─── 用户覆盖状态 ────────────────────────────────────────
export const MOCK_USER_OVERRIDES: Record<string, UserOverrideInfo> = {
  "ceo@acompany.com": { userId: "ceo@acompany.com", status: "local" },
  "tim@acompany.com": { userId: "tim@acompany.com", status: "groupOverride" },
  "peter@acompany.com": { userId: "peter@acompany.com", status: "groupOverride" },
  "alice@acompany.com": { userId: "alice@acompany.com", status: "groupOverride" },
  "bob@acompany.com": { userId: "bob@acompany.com", status: "groupOverride" },
  "carol@acompany.com": { userId: "carol@acompany.com", status: "groupOverride" },
  "david@acompany.com": { userId: "david@acompany.com", status: "local" },
  "eve@acompany.com": { userId: "eve@acompany.com", status: "local" },
  "frank@acompany.com": { userId: "frank@acompany.com", status: "groupOverride" },
  "grace@acompany.com": { userId: "grace@acompany.com", status: "groupOverride" },
  "henry@acompany.com": { userId: "henry@acompany.com", status: "primaryDeptMissing" },
  // iris 同时在 og-ai-core 和 og-frontend → VPC 冲突
  "iris@acompany.com": {
    userId: "iris@acompany.com",
    status: "groupConflict",
    conflictResourceKind: "vpc",
    conflictCandidates: [
      { resourceId: "vpc-og-ai", resourceName: "vpc-ai-core", via: "AI 核心团队", latestBindingAt: "2025-03-20" },
      { resourceId: "vpc-og-fe", resourceName: "vpc-fe-team", via: "前端研发同学", latestBindingAt: "2025-02-25" },
    ],
    isResolved: false,
  },
  "jack@acompany.com": { userId: "jack@acompany.com", status: "local" },
  "kate@acompany.com": { userId: "kate@acompany.com", status: "groupOverride" },
  "leo@acompany.com": { userId: "leo@acompany.com", status: "local" },
  "mike@acompany.com": { userId: "mike@acompany.com", status: "local" },
  "nina@acompany.com": { userId: "nina@acompany.com", status: "groupOverride" },
  "oscar@acompany.com": { userId: "oscar@acompany.com", status: "local" },
};

// ─── 最终生效配置（简化 mock） ────────────────────────────
export const MOCK_EFFECTIVE_CONFIG: Record<string, Partial<EffectiveConfig>> = {
  "ceo@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest"],
    channels: ["默认通道"],
    securityGroup: "sg-default-enterprise",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "tim@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4"],
    channels: ["默认通道", "技术部专用通道"],
    securityGroup: "sg-tech-internal",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "peter@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "腾讯云 DeepSeek - DeepSeek V3 0324"],
    channels: ["默认通道", "产品部通道"],
    securityGroup: "sg-product-internal",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "alice@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4", "GPT-4o", "DeepSeek V3"],
    channels: ["默认通道", "技术部专用通道", "前端组专用通道", "前端研发同学 高级通道"],
    securityGroup: "sg-frontend",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "bob@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4"],
    channels: ["默认通道", "技术部专用通道", "后端研发同学 专属通道"],
    securityGroup: "sg-tech-internal",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "carol@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4", "DeepSeek V3"],
    channels: ["默认通道", "技术部专用通道", "后端研发同学 专属通道"],
    securityGroup: "sg-tech-internal",
    vpc: "vpc-ai-core",
    memory: "mem-default",
    image: "coder-v2",
  },
  "david@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "腾讯云 DeepSeek - DeepSeek V3 0324"],
    channels: ["默认通道", "产品部通道"],
    securityGroup: "sg-product-internal",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "eve@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "腾讯云 DeepSeek - DeepSeek V3 0324"],
    channels: ["默认通道", "产品部通道"],
    securityGroup: "sg-product-internal",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "frank@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4", "GPT-4o"],
    channels: ["默认通道", "技术部专用通道", "前端组专用通道", "前端研发同学 高级通道", "后端研发同学 专属通道"],
    securityGroup: "sg-frontend",
    vpc: "vpc-fe-team",
    memory: "mem-default",
    image: "coder-v2",
  },
  "grace@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4"],
    channels: ["默认通道", "技术部专用通道", "后端研发同学 专属通道"],
    securityGroup: "sg-tech-internal",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "henry@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest"],
    channels: ["默认通道"],
    securityGroup: "sg-default-enterprise",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "iris@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4", "DeepSeek V3"],
    channels: ["默认通道", "技术部专用通道", "前端研发同学 高级通道"],
    securityGroup: "sg-tech-internal",
    vpc: "vpc-ai-core", // 冲突未裁决，最新绑定兜底
    memory: "mem-default",
    image: "coder-v2",
  },
  "jack@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest"],
    channels: ["默认通道"],
    securityGroup: "sg-default-enterprise",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "kate@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4", "GPT-4o"],
    channels: ["默认通道", "技术部专用通道", "前端组专用通道", "前端研发同学 高级通道"],
    securityGroup: "sg-frontend",
    vpc: "vpc-fe-team",
    memory: "mem-default",
    image: "coder-v2",
  },
  "leo@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "腾讯云 DeepSeek - DeepSeek V3 0324"],
    channels: ["默认通道", "产品部通道"],
    securityGroup: "sg-product-internal",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "mike@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4"],
    channels: ["默认通道", "技术部专用通道"],
    securityGroup: "sg-tech-internal",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
  "nina@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest", "Claude Sonnet 4", "GPT-4o", "腾讯云 DeepSeek - DeepSeek V3 0324"],
    channels: ["默认通道", "技术部专用通道", "产品部通道", "前端组专用通道", "前端研发同学 高级通道"],
    securityGroup: "sg-frontend",
    vpc: "vpc-fe-team",
    memory: "mem-default",
    image: "coder-v2",
  },
  "oscar@acompany.com": {
    models: ["腾讯云混元 - 混元 TurboS Latest"],
    channels: ["默认通道"],
    securityGroup: "sg-default-enterprise",
    vpc: "vpc-default",
    memory: "mem-default",
    image: "coder-v2",
  },
};

// ─── 主部门路径工具 ─────────────────────────────────────
/** 返回用户主部门完整路径（仅 oneid-dept 链）。无则返回 '—' */
export function getPrimaryDeptPath(
  primaryGroupId: string | null,
  groups: UserGroup[] = MOCK_GROUPS
): string {
  if (!primaryGroupId) return "—";
  const map = new Map(groups.map((g) => [g.id, g]));
  const chain: string[] = [];
  let cur = map.get(primaryGroupId);
  while (cur) {
    chain.unshift(cur.name);
    cur = cur.parentId ? map.get(cur.parentId) : undefined;
  }
  return chain.length > 0 ? chain.join(" / ") : "—";
}

// ─── 配置总览 Mock ─────────────────────────────────────
/**
 * 获取某组织的配置总览条目（按 12 种配置项聚合）。
 * 模拟后端返回：本组织直配 + 继承自祖先 + 平台默认。
 */
export function getConfigEntries(
  groupId: string,
  groups: UserGroup[]
): ConfigEntry[] {
  const groupMap = new Map(groups.map((g) => [g.id, g]));
  const currentGroup = groupMap.get(groupId);

  // 获取祖先链
  const ancestors: UserGroup[] = [];
  let cur = currentGroup;
  while (cur && cur.parentId) {
    const p = groupMap.get(cur.parentId);
    if (!p) break;
    ancestors.push(p);
    cur = p;
  }

  // 根据组织 id 获取完整路径（如 "全公司/产品部"）
  const getGroupFullPath = (gId: string): string => {
    const chain: string[] = [];
    let node = groupMap.get(gId);
    while (node) {
      chain.unshift(node.name);
      node = node.parentId ? groupMap.get(node.parentId) : undefined;
    }
    return chain.join("/");
  };

  const local = (groupName: string): ConfigEntry["source"] => ({ type: "local", groupName });
  const inherited = (groupName: string): ConfigEntry["source"] => ({ type: "inherited", groupName });
  const platformDefault: ConfigEntry["source"] = { type: "platformDefault" };

  const entries: ConfigEntry[] = [];

  // ──── 1. 模型 ────
  entries.push({
    id: "m-default",
    category: "model",
    label: "腾讯云混元 - 混元 TurboS Latest",
    source: platformDefault,
  });
  // 根据组织层级添加
  if (["dept-tech", "dept-fe", "dept-be", "dept-ai"].includes(groupId)) {
    entries.push({ id: "m-tech", category: "model", label: "Claude Sonnet 4", source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")) });
  }
  if (groupId === "dept-fe") {
    entries.push({ id: "m-fe", category: "model", label: "GPT-4o", source: local(getGroupFullPath("dept-fe")) });
  }
  if (["dept-ai", "og-ai-core"].includes(groupId)) {
    entries.push({ id: "m-ai", category: "model", label: "DeepSeek V3", source: local(getGroupFullPath(groupId)) });
  }
  if (["dept-product", "dept-pm", "dept-design"].includes(groupId)) {
    entries.push({ id: "m-product", category: "model", label: "腾讯云 DeepSeek - DeepSeek V3 0324", source: groupId === "dept-product" ? local(getGroupFullPath("dept-product")) : inherited(getGroupFullPath("dept-product")) });
  }
  // 运营组及其子组织：本地绑定的模型
  if (["dept-operation", "dept-operation-1", "dept-operation-2"].includes(groupId)) {
    entries.push({ id: "m-product", category: "model", label: "腾讯云 DeepSeek - DeepSeek V3 0324", source: inherited(getGroupFullPath("dept-product")) });
    entries.push({ id: "m-operation", category: "model", label: "Gemini 2.5 Pro", source: groupId === "dept-operation" ? local(getGroupFullPath("dept-operation")) : inherited(getGroupFullPath("dept-operation")) });
  }
  // 运营一组：本地绑定的模型
  if (groupId === "dept-operation-1") {
    entries.push({ id: "m-operation-1", category: "model", label: "Claude Sonnet 4", source: local(getGroupFullPath("dept-operation-1")) });
  }
  // 运营二组：本地绑定的模型
  if (groupId === "dept-operation-2") {
    entries.push({ id: "m-operation-2", category: "model", label: "GPT-4o mini", source: local(getGroupFullPath("dept-operation-2")) });
  }
  // 普通模式
  if (["mgrp-rd", "mgrp-rd-fe", "mgrp-rd-be"].includes(groupId)) {
    entries.push({ id: "m-rd", category: "model", label: "Claude Sonnet 4", source: groupId === "mgrp-rd" ? local(getGroupFullPath("mgrp-rd")) : inherited(getGroupFullPath("mgrp-rd")) });
  }
  if (groupId === "mgrp-product") {
    entries.push({ id: "m-product-exclusive", category: "model", label: "腾讯云混元 - 混元 Pro", source: local(getGroupFullPath("mgrp-product")) });
  }
  if (groupId === "mgrp-rd-be") {
    entries.push({ id: "m-rd-be-exclusive", category: "model", label: "DeepSeek V3", source: local(getGroupFullPath("mgrp-rd-be")) });
  }

  // ──── 2. 通道 ────
  entries.push({
    id: "c-wechat",
    category: "channel",
    label: "微信",
    source: platformDefault,
  });
  entries.push({
    id: "c-wework",
    category: "channel",
    label: "企业微信",
    source: platformDefault,
  });
  entries.push({
    id: "c-feishu",
    category: "channel",
    label: "飞书",
    source: platformDefault,
  });
  if (["dept-tech", "dept-fe", "dept-be", "dept-ai"].includes(groupId)) {
    entries.push({ id: "c-dingtalk", category: "channel", label: "钉钉", source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")) });
  }
  if (["dept-product", "dept-pm", "dept-design"].includes(groupId)) {
    entries.push({ id: "c-qq", category: "channel", label: "QQ", source: groupId === "dept-product" ? local(getGroupFullPath("dept-product")) : inherited(getGroupFullPath("dept-product")) });
  }
  // 运营组及其子组织：通道
  if (["dept-operation", "dept-operation-1", "dept-operation-2"].includes(groupId)) {
    // 继承自产品部的 QQ
    entries.push({ id: "c-qq", category: "channel", label: "QQ", source: inherited(getGroupFullPath("dept-product")) });
    // 运营专属通道：运营组本地绑定，子组织继承
    entries.push({ id: "c-operation", category: "channel", label: "运营专属通道", source: groupId === "dept-operation" ? local(getGroupFullPath("dept-operation")) : inherited(getGroupFullPath("dept-operation")) });
  }
  // 运营一组：本地绑定的通道
  if (groupId === "dept-operation-1") {
    entries.push({ id: "c-operation-1", category: "channel", label: "运营一组内部通道", source: local(getGroupFullPath("dept-operation-1")) });
  }
  // 运营二组：本地绑定的通道
  if (groupId === "dept-operation-2") {
    entries.push({ id: "c-operation-2", category: "channel", label: "运营二组外部通道", source: local(getGroupFullPath("dept-operation-2")) });
  }
  // 普通模式：产品组专属通道
  if (groupId === "mgrp-product") {
    entries.push({ id: "c-product-exclusive", category: "channel", label: "产品组专属通道", source: local(getGroupFullPath("mgrp-product")) });
  }

  // ──── 4. 技能（初始技能包 + 角色 + 技能安装来源（只有一条）） ────
  entries.push({
    id: "skill-pack-default",
    category: "skill",
    label: "标准技能包",
    subLabel: "初始技能包",
    source: platformDefault,
  });
  entries.push({
    id: "skill-role-default",
    category: "skill",
    label: "通用助手",
    subLabel: "角色",
    source: platformDefault,
  });
  // 技能安装来源：只有一条，覆盖时替换
  if (["dept-tech", "dept-fe", "dept-be", "dept-ai"].includes(groupId)) {
    entries.push({
      id: "skill-source-tech",
      category: "skill",
      label: "https://git.techcorp.cn/skill-registry",
      subLabel: "技能安装来源",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
    });
  } else {
    entries.push({
      id: "skill-source-default",
      category: "skill",
      label: "默认",
      subLabel: "技能安装来源",
      source: platformDefault,
    });
  }
  if (["dept-tech", "dept-fe", "dept-be", "dept-ai"].includes(groupId)) {
    entries.push({
      id: "skill-pack-tech",
      category: "skill",
      label: "开发者技能包",
      subLabel: "初始技能包",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
    });
    entries.push({
      id: "skill-role-tech",
      category: "skill",
      label: "代码助手",
      subLabel: "角色",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
    });
  }
  if (["dept-product", "dept-pm", "dept-design"].includes(groupId)) {
    entries.push({
      id: "skill-role-product",
      category: "skill",
      label: "产品经理助手",
      subLabel: "角色",
      source: groupId === "dept-product" ? local(getGroupFullPath("dept-product")) : inherited(getGroupFullPath("dept-product")),
    });
  }

  // ──── 5. Agent 工具（公共技能 + 企业技能 + 企业插件 + 企业MCP + 企业规范） ────
  entries.push({
    id: "at-skill-default",
    category: "agentTool",
    label: "知识库检索",
    subLabel: "企业技能",
    source: platformDefault,
    version: "v1.0.0",
  });
  entries.push({
    id: "at-plugin-default",
    category: "agentTool",
    label: "Jira 工单",
    subLabel: "企业插件",
    source: platformDefault,
    version: "v2.1.0",
  });
  entries.push({
    id: "at-mcp-default",
    category: "agentTool",
    label: "文档解析服务",
    subLabel: "企业MCP",
    source: platformDefault,
    version: "v1.3.0",
  });
  entries.push({
    id: "at-public-default",
    category: "agentTool",
    label: "Markdown 转换",
    subLabel: "公共技能",
    source: platformDefault,
    version: "v1.0.0",
  });
  entries.push({
    id: "at-public-default-2",
    category: "agentTool",
    label: "代码格式化",
    subLabel: "公共技能",
    source: platformDefault,
    version: "v1.0.0",
  });
  entries.push({
    id: "at-standard-default",
    category: "agentTool",
    label: "前端代码规范",
    subLabel: "企业规范",
    source: platformDefault,
    version: "v1.2.0",
  });
  entries.push({
    id: "at-standard-default-2",
    category: "agentTool",
    label: "Git 提交规范",
    subLabel: "企业规范",
    source: platformDefault,
    version: "v1.0.0",
  });
  if (["dept-tech", "dept-fe", "dept-be", "dept-ai"].includes(groupId)) {
    entries.push({
      id: "at-skill-tech",
      category: "agentTool",
      label: "代码审查",
      subLabel: "企业技能",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
      version: "v1.5.0",
    });
    entries.push({
      id: "at-plugin-tech",
      category: "agentTool",
      label: "GitLab CI/CD",
      subLabel: "企业插件",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
      version: "v2.0.0",
    });
    entries.push({
      id: "at-mcp-tech",
      category: "agentTool",
      label: "代码仓库 MCP",
      subLabel: "企业MCP",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
      version: "v1.1.0",
    });
    entries.push({
      id: "at-public-tech",
      category: "agentTool",
      label: "SQL 优化助手",
      subLabel: "公共技能",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
      version: "v1.0.0",
    });
    entries.push({
      id: "at-standard-tech",
      category: "agentTool",
      label: "后端代码规范",
      subLabel: "企业规范",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
      version: "v2.0.0",
    });
  }
  if (groupId === "dept-fe") {
    entries.push({
      id: "at-plugin-fe",
      category: "agentTool",
      label: "Figma 设计稿同步",
      subLabel: "企业插件",
      source: local(getGroupFullPath("dept-fe")),
    });
  }
  // 运营组：本地绑定的 Agent 工具
  if (groupId === "dept-operation") {
    entries.push({
      id: "at-skill-operation",
      category: "agentTool",
      label: "内容审核",
      subLabel: "企业技能",
      source: local(getGroupFullPath("dept-operation")),
    });
    entries.push({
      id: "at-mcp-operation",
      category: "agentTool",
      label: "数据分析 MCP",
      subLabel: "企业MCP",
      source: local(getGroupFullPath("dept-operation")),
    });
  }
  // 云原生迁移项目：本项目专属「企业技能」（项目目前仅支持工具）
  if (groupId === "proj-cloud-migration") {
    entries.push({
      id: "at-proj-cloud-migration",
      category: "agentTool",
      label: "云原生迁移专属技能",
      subLabel: "企业技能",
      source: local(getGroupFullPath("proj-cloud-migration")),
      version: "v1.0.0",
    });
  }

  // ──── 6. 记忆（永远只有一条） ────
  if (["dept-tech", "dept-fe", "dept-be", "dept-ai"].includes(groupId)) {
    entries.push({
      id: "mem-tech",
      category: "memory",
      label: "开启 Pro 版",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
    });
  } else {
    entries.push({
      id: "mem-default",
      category: "memory",
      label: "开启 Free 版",
      source: platformDefault,
    });
  }

  // ──── 7. 网盘（财务部未配置） ────
  if (groupId !== "dept-finance") {
    entries.push({
      id: "drive-default",
      category: "drive",
      label: "开启",
      source: platformDefault,
    });
  }

  // ──── 8. 镜像（设计组和研发-前端未配置镜像，演示初始化未完成） ────
  if (groupId !== "mgrp-design" && groupId !== "mgrp-rd-fe") {
    entries.push({
      id: "img-openclaw",
      category: "image",
      label: "Openclaw",
      source: platformDefault,
    });
  }
  if (["dept-tech", "dept-fe", "dept-be", "dept-ai"].includes(groupId)) {
    entries.push({
      id: "img-hermes",
      category: "image",
      label: "Hermes Agent",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")),
    });
  }
  if (["dept-ai", "og-ai-core"].includes(groupId)) {
    entries.push({
      id: "img-lighthouse",
      category: "image",
      label: "Lighthouse ACE",
      source: local(getGroupFullPath(groupId)),
    });
  }

  // ──── 9. 云资源（资源策略命中：当前组织 → 最近上级 → 企业默认） ────
  //
  // 命中链（自底向上，取第一个命中）：
  //   ① 当前组织有自定义资源策略 → source = local，标签：本组织
  //   ② 沿 parentId 向上找到最近一个有自定义策略的祖先 → source = inherited，标签：继承自 {祖先组织名}
  //   ③ 都没有 → 回退企业默认预设策略 → source = presetPolicy，标签：预设策略
  //
  // 展示字段：策略名称 / 来源 tag / 计费模式 / 实例规格 / 系统盘类型 / 系统盘容量 / 公网IP / 公网计费模式(仅分配时) / 带宽上限(仅分配时)

  type CloudResourcePolicy = {
    policyName: string;
    billingMode: "subscription" | "payAsYouGo";       // 包年包月 | 按量计费
    instanceSpec: string;                               // 实例规格 如 "4核8GiB"（仅用户可读，不含 Ai2 编码）
    systemDiskType: string;                             // 系统盘类型 如 "SSD 云硬盘"
    systemDiskSize: number;                             // 系统盘容量 GiB
    assignPublicIp: boolean;                            // 是否分配公网 IP
    bandwidthBillingMode?: "monthlyBandwidth" | "traffic"; // 公网计费模式（仅分配时）
    bandwidthLimit?: number;                             // 带宽上限 Mbps（仅分配时）
  };

  const CLOUD_RESOURCE_PRESET: CloudResourcePolicy = {
    policyName: "企业默认资源策略",
    billingMode: "subscription",
    instanceSpec: "4核8GiB",
    systemDiskType: "高性能云硬盘",
    systemDiskSize: 80,
    assignPublicIp: true,
    bandwidthBillingMode: "traffic",
    bandwidthLimit: 5,
  };

  // 组织自定义资源策略（研发组有专属策略；其子组织（研发-前端/研发-后端）自动继承）
  const CLOUD_RESOURCE_BINDINGS: Record<string, CloudResourcePolicy> = {
    "mgrp-rd": {
      policyName: "研发组高配策略",
      billingMode: "payAsYouGo",
      instanceSpec: "8核16GiB",
      systemDiskType: "SSD 云硬盘",
      systemDiskSize: 100,
      assignPublicIp: false,
    },
  };

  const findCloudResourceAncestor = (gid: string): string | null => {
    let cur: UserGroup | undefined = groupMap.get(gid);
    while (cur) {
      if (CLOUD_RESOURCE_BINDINGS[cur.id]) return cur.id;
      cur = cur.parentId ? groupMap.get(cur.parentId) : undefined;
    }
    return null;
  };

  const buildCloudResourceEntry = (policy: CloudResourcePolicy, source: ConfigEntry["source"]) => ({
    id: `cloud-resource-${groupId}`,
    category: "cloudResource" as const,
    label: policy.policyName,
    source,
    meta: {
      policyName: policy.policyName,
      billingMode: policy.billingMode,
      instanceSpec: policy.instanceSpec,
      systemDiskType: policy.systemDiskType,
      systemDiskSize: policy.systemDiskSize,
      assignPublicIp: policy.assignPublicIp,
      ...(policy.assignPublicIp ? {
        bandwidthBillingMode: policy.bandwidthBillingMode ?? "monthlyBandwidth",
        bandwidthLimit: policy.bandwidthLimit ?? 5,
      } : {}),
    },
  });

  if (CLOUD_RESOURCE_BINDINGS[groupId]) {
    // ① 本组织有自定义策略 → local
    entries.push(buildCloudResourceEntry(CLOUD_RESOURCE_BINDINGS[groupId], local(getGroupFullPath(groupId))));
  } else {
    const ancestorId = findCloudResourceAncestor(groupId);
    if (ancestorId) {
      // ② 继承最近上级
      entries.push(buildCloudResourceEntry(CLOUD_RESOURCE_BINDINGS[ancestorId], inherited(getGroupFullPath(ancestorId))));
    } else {
      // ③ 企业默认预设策略
      entries.push(buildCloudResourceEntry(CLOUD_RESOURCE_PRESET, { type: "presetPolicy" }));
    }
  }

  // ──── 10. 私有网络与子网（永远只有一条；按"网络管理"真实配置匹配） ────
  //
  // 数据流：
  //   组织配置总览 是消费方，从「网络管理 > 私有网络与子网配置」(SecurityGroupManagement.tsx
  //   里的 INITIAL_VPC_LIST) 单向读取真实配置；下面这份 NETWORK_BINDINGS 是同源精简映射，
  //   仅供本页渲染用，不影响网络管理页本身。
  //
  // 网络管理真实绑定关系（来自 SecurityGroupManagement.tsx INITIAL_VPC_LIST）：
  //   - 企业默认（type: "enterprise"，即「预设策略」兜底）：vpc-jp7fjg13 / 企业默认网络 / 10.0.0.0/16
  //   - 研发组：vpc-9lyx5t8h / 研发组网络 / 10.1.0.0/16，子网 subnet-gaclgbzu（广州五区, 带外管理, 192.168.20.0/24）
  //   - 产品组：vpc-ri7mmw6n / 产品组网络 / 10.2.0.0/16，子网 subnet-mn3op5qr（广州五区, 部署子网A, 192.168.1.0/24）
  //                                                       + subnet-st6uv7wx（广州六区, 部署子网B, 192.168.2.0/24）
  //
  // 匹配链（自底向上，取第一个命中）：
  //   ① 当前组织在网络管理里被单独绑定（type: "group"，associatedGroups 含当前组织）
  //        → source = local，标签：本组织
  //   ② 沿 parentId 向上找到最近一个被绑定的祖先组织
  //        → source = inherited，标签：继承自 {祖先组织名}（仅继承真实配置，不继承 demo 删除态）
  //   ③ 都没有 → 回退企业默认
  //        → source = presetPolicy，标签：预设策略
  //          展示约定：VPC ID / 子网 ID 都不展示真实 ID，统一展示「自动分配」（由 NodeContentPanel 渲染层处理）
  //
  // 「⚠ 配置待更新」适用范围：
  //   - 仅 ① 的 local 来源（真实绑定 type: "group"）的 VPC / 子网，可能因云端删除触发；
  //   - 预设策略（③）的 VPC 由平台自动重建，永不进入"待更新"；
  //   - ② 的继承也只继承真实部分，子组织不会因父组织的 demo 删除态而被误标。

  type RealSubnet = { zone: string; subnetId: string; subnetName: string; subnetCidr: string };
  type RealVpcBinding = {
    vpcId: string;
    vpcName: string;
    vpcCidr: string;
    subnets: RealSubnet[];
  };
  // 已在网络管理被单独绑定的组织 → 真实 VPC/子网（仅真实部分，不含 demo 删除态）
  const NETWORK_BINDINGS: Record<string, RealVpcBinding> = {
    "mgrp-rd": {
      vpcId: "vpc-9lyx5t8h",
      vpcName: "研发组网络",
      vpcCidr: "10.1.0.0/16",
      subnets: [
        { zone: "广州五区", subnetId: "subnet-gaclgbzu", subnetName: "带外管理", subnetCidr: "192.168.20.0/24" },
      ],
    },
    "mgrp-product": {
      vpcId: "vpc-ri7mmw6n",
      vpcName: "产品组网络",
      vpcCidr: "10.2.0.0/16",
      subnets: [
        { zone: "广州五区", subnetId: "subnet-mn3op5qr", subnetName: "部署子网A", subnetCidr: "192.168.1.0/24" },
        { zone: "广州六区", subnetId: "subnet-st6uv7wx", subnetName: "部署子网B", subnetCidr: "192.168.2.0/24" },
      ],
    },
  };
  // Demo 专属：研发组的网络配置存在 已被云端删除 的资源
  //
  // 用户管理页只展示可用资源，并在以下两种情况下提示「配置待更新」：
  //   1) VPC 整个被删除（vpcDeleted）
  //   2) 某个可用区下所有子网均被删除（zonesAllDeleted）
  //
  // 部分子网删除但该可用区仍有可用子网时，本页静默处理（不展示已删除项、不提示）。
  //
  // 当前 demo 选择「VPC 健在 + 演示两类区状态」：
  //   - 广州五区：仍有 1 条可用子网 subnet-gaclgbzu（正常展示）
  //   - 广州六区：原有的子网都被删了 → 该区无可用子网，提示「配置待更新」
  //   - 广州七区：原有的子网都被删了 → 该区无可用子网，提示「配置待更新」
  //   （注：研发组在「网络管理」原本只在广州五区指定了 subnet-gaclgbzu，广州六/七区
  //    的演示属于"曾经配过、后来都被删了"的扩展场景，仅在 demo 数据上模拟）
  const RD_DEMO_OUTDATED: {
    vpcDeleted: boolean;
    zonesAllDeleted: string[];
  } = {
    vpcDeleted: false,
    zonesAllDeleted: ["广州六区", "广州七区"],
  };

  // 在 groupId 自身或祖先链上，查找第一个在 NETWORK_BINDINGS 里命中的组织 id
  const findBoundAncestor = (gid: string): string | null => {
    let cur: UserGroup | undefined = groupMap.get(gid);
    while (cur) {
      if (NETWORK_BINDINGS[cur.id]) return cur.id;
      cur = cur.parentId ? groupMap.get(cur.parentId) : undefined;
    }
    return null;
  };

  // 构造一个组织对应的真实 VPC meta：
  //   - 仅展示可用资源；已删除子网不进入 subnets 列表；
  //   - VPC 整体删除时：vpcName / vpcCidr 留空 + vpcDeleted: true；
  //   - 某可用区所有子网均被删除：通过 zonesAllDeleted 字段透传给渲染层。
  const buildVpcMeta = (boundGroupId: string): Record<string, unknown> => {
    const bind = NETWORK_BINDINGS[boundGroupId];
    const isRd = boundGroupId === "mgrp-rd";
    const vpcDeleted = isRd && RD_DEMO_OUTDATED.vpcDeleted;
    const zonesAllDeleted = isRd ? [...RD_DEMO_OUTDATED.zonesAllDeleted] : [];
    return {
      vpcId: bind.vpcId,
      vpcName: vpcDeleted ? "" : bind.vpcName,
      vpcCidr: vpcDeleted ? "" : bind.vpcCidr,
      ...(vpcDeleted ? { vpcDeleted: true } : {}),
      // 仅可用子网（VPC 删除时不再展示子网，由渲染层据 vpcDeleted 隐藏整个子网区域）
      subnets: bind.subnets.map((s) => ({
        zone: s.zone,
        subnetId: s.subnetId,
        subnetName: s.subnetName,
        subnetCidr: s.subnetCidr,
      })),
      // 该 VPC 下哪些可用区已无可用子网（导致该区无法用于实例创建）
      ...(zonesAllDeleted.length > 0 ? { zonesAllDeleted } : {}),
    };
  };

  if (NETWORK_BINDINGS[groupId]) {
    // ① 自己被单独绑定 → local 本组织
    entries.push({
      id: `vpc-${groupId}`,
      category: "network",
      label: "",
      subLabel: "私有网络与子网",
      source: local(getGroupFullPath(groupId)),
      meta: buildVpcMeta(groupId),
    });
  } else {
    const ancestorId = findBoundAncestor(groupId);
    if (ancestorId) {
      // ② 继承自最近的已绑定祖先组织（完整继承所有字段，包括 demo 待更新态）
      //    业务上：父级 VPC 删除会同时影响所有继承该配置的子组织。
      entries.push({
        id: `vpc-${groupId}`,
        category: "network",
        label: "",
        subLabel: "私有网络与子网",
        source: inherited(getGroupFullPath(ancestorId)),
        meta: buildVpcMeta(ancestorId),
      });
    } else {
      // ③ 回退预设策略 → 企业默认网络（vpc-jp7fjg13）
      //    展示层会把 source.type === "presetPolicy" 的 VPC ID / 子网 ID 渲染为「自动分配」；
      //    子网按可用区逐条展示，可用区与「网络管理 > AVAILABLE_ZONES」保持同源（广州 5/6/7 区）。
      entries.push({
        id: `vpc-${groupId}`,
        category: "network",
        label: "",
        subLabel: "私有网络与子网",
        source: { type: "presetPolicy" },
        meta: {
          vpcId: "vpc-jp7fjg13",
          vpcName: "企业默认网络",
          vpcCidr: "10.0.0.0/16",
          // 预设策略子网按可用区逐条展示，故不再使用 subnets 列表，改用 zones
          zones: ["广州五区", "广州六区", "广州七区"],
          subnets: [],
        },
      });
    }
  }

  // ──── 9b. 安全组（永远只有一条） ────
  if (["dept-tech", "dept-fe", "dept-be", "dept-ai"].includes(groupId)) {
    const sgLabel = groupId === "dept-fe" ? "前端安全组（sg-fe8k2m3p）" : "技术部安全组（sg-tech4n7w）";
    entries.push({
      id: groupId === "dept-fe" ? "sg-fe" : "sg-tech",
      category: "network",
      label: sgLabel,
      subLabel: "安全组",
      source: groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) :
              groupId === "dept-fe" ? local(getGroupFullPath("dept-fe")) :
              inherited(getGroupFullPath("dept-tech")),
    });
  } else if (["dept-product", "dept-pm", "dept-design"].includes(groupId)) {
    entries.push({
      id: "sg-product",
      category: "network",
      label: "产品部安全组（sg-prod6j9r）",
      subLabel: "安全组",
      source: groupId === "dept-product" ? local(getGroupFullPath("dept-product")) : inherited(getGroupFullPath("dept-product")),
    });
  } else if (groupId === "mgrp-product") {
    entries.push({
      id: "sg-mgrp-product",
      category: "network",
      label: "产品组安全组（sg-product-group）",
      subLabel: "安全组",
      source: local(getGroupFullPath("mgrp-product")),
    });
  } else {
    entries.push({
      id: "sg-default",
      category: "network",
      label: "默认安全组（sg-dft1x5a8）",
      subLabel: "安全组",
      source: platformDefault,
    });
  }

  // ──── 11. CLS 日志服务 ────
  entries.push({
    id: "cls-default",
    category: "cls",
    label: "开启",
    source: platformDefault,
  });

  // ──── 12. AI Agent 安全 ────
  entries.push({
    id: "ai-security-default",
    category: "aiAgentSecurity",
    label: "开启",
    source: platformDefault,
  });

  // ──── 13. 平台策略（用户配额 + 模型配额 + 功能权限开关） ────
  // 平台策略只有「预设策略」或「本组织」来源，没有「全部用户」
  const presetPolicy: ConfigEntry["source"] = { type: "presetPolicy" };

  entries.push({
    id: "policy-claw-limit",
    category: "platformPolicy",
    label: "单用户 Agent 数量上限",
    subLabel: "用户配额",
    source: presetPolicy,
    meta: { value: 3 },
  });
  entries.push({
    id: "policy-token-limit",
    category: "platformPolicy",
    label: "单用户 Tokens 上限",
    subLabel: "用户配额",
    source: presetPolicy,
    meta: { value: 500000, isTokenQuota: true, quotaType: "user", groupCreatedAt: currentGroup?.createdAt },
  });
  // 全局 Tokens 上限
  if (groupId === "mgrp-rd-fe") {
    entries.push({
      id: "policy-global-token",
      category: "platformPolicy",
      label: "全局 Tokens 上限",
      subLabel: "模型配额",
      source: local(getGroupFullPath("mgrp-rd-fe")),
      meta: { value: 2000000, isTokenQuota: true, quotaType: "global", groupCreatedAt: currentGroup?.createdAt },
    });
  } else {
    entries.push({
      id: "policy-global-token",
      category: "platformPolicy",
      label: "全局 Tokens 上限",
      subLabel: "模型配额",
      source: presetPolicy,
      meta: { value: 1000000, isTokenQuota: true, quotaType: "global", groupCreatedAt: currentGroup?.createdAt },
    });
  }
  // 功能权限开关：顺序与平台策略页对齐
  const isTechGroup = ["dept-tech", "dept-fe", "dept-be", "dept-ai"].includes(groupId);
  const techSource = isTechGroup
    ? (groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : inherited(getGroupFullPath("dept-tech")))
    : null;

  entries.push({
    id: "policy-config-model",
    category: "platformPolicy",
    label: "允许用户配置模型",
    subLabel: "功能权限开关",
    source: presetPolicy,
    meta: { enabled: true },
  });
  entries.push({
    id: "policy-config-channel",
    category: "platformPolicy",
    label: "允许用户配置通道",
    subLabel: "功能权限开关",
    source: presetPolicy,
    meta: { enabled: true },
  });
  entries.push({
    id: "policy-custom-model",
    category: "platformPolicy",
    label: "允许用户添加自定义模型",
    subLabel: "功能权限开关",
    source: presetPolicy,
    meta: { enabled: false },
  });
  entries.push({
    id: "policy-terminal",
    category: "platformPolicy",
    label: "允许用户进入 Agent 终端",
    subLabel: "功能权限开关",
    source: techSource ?? presetPolicy,
    meta: { enabled: isTechGroup ? true : false },
  });
  entries.push({
    id: "policy-panel",
    category: "platformPolicy",
    label: "允许用户访问 Agent 面板",
    subLabel: "功能权限开关",
    source: techSource ?? presetPolicy,
    meta: { enabled: isTechGroup ? true : false },
  });
  entries.push({
    id: "policy-chat-view",
    category: "platformPolicy",
    label: "允许用户使用对话视图",
    subLabel: "功能权限开关",
    source: presetPolicy,
    meta: { enabled: true },
  });
  entries.push({
    id: "policy-cloud-browser",
    category: "platformPolicy",
    label: "允许用户访问 Agent 云桌面",
    subLabel: "功能权限开关",
    source: presetPolicy,
    meta: { enabled: false },
  });
  entries.push({
    id: "policy-lobster-doctor",
    category: "platformPolicy",
    label: "允许用户使用龙虾医生",
    subLabel: "功能权限开关",
    source: presetPolicy,
    meta: { enabled: false },
  });
  entries.push({
    id: "policy-model-quota",
    category: "platformPolicy",
    label: "允许用户查看模型额度",
    subLabel: "功能权限开关",
    source: presetPolicy,
    meta: { enabled: true },
  });

  // ──── 12. 云开发 ────
  // 允许创建环境：技术部、产品部、研发组及其子组织默认开启
  const isCloudDevGroup = ["dept-tech", "dept-fe", "dept-be", "dept-ai", "dept-product", "dept-pm", "dept-design", "mgrp-rd", "mgrp-rd-fe", "mgrp-rd-be"].includes(groupId);
  entries.push({
    id: "cloud-dev-create-env",
    category: "cloudDev",
    label: "允许用户创建环境",
    source: presetPolicy,
    meta: { enabled: true },
  });
  entries.push({
    id: "cloud-dev-env-limit",
    category: "cloudDev",
    label: "用户创建环境数量上限",
    source: isCloudDevGroup
      ? (groupId === "dept-tech" ? local(getGroupFullPath("dept-tech")) : groupId === "dept-product" ? local(getGroupFullPath("dept-product")) : inherited(getGroupFullPath(
          ["dept-fe", "dept-be", "dept-ai"].includes(groupId) ? "dept-tech"
          : ["dept-pm", "dept-design"].includes(groupId) ? "dept-product"
          : "mgrp-rd"
        )))
      : presetPolicy,
    meta: { value: isCloudDevGroup ? 10 : 5 },
  });

  return entries;
}

/** 配置项元信息 */
export const CONFIG_CATEGORY_META: Record<
  ConfigCategory,
  { label: string; icon: string; color: string; bg: string; path: string; description: string }
> = {
  model: {
    label: "模型",
    icon: "Brain",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/model-config",
    description: "用户能使用哪些模型",
  },
  channel: {
    label: "通道",
    icon: "MessageSquare",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/channel-config",
    description: "用户通过哪些通道访问模型",
  },
  skill: {
    label: "技能",
    icon: "Puzzle",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/skill-config",
    description: "初始技能包、角色与技能安装来源",
  },
  agentTool: {
    label: "Agent 工具",
    icon: "Wrench",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/agent-tool-library",
    description: "企业技能、企业插件与企业 MCP",
  },
  memory: {
    label: "记忆",
    icon: "MemoryStick",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/memory-management",
    description: "记忆功能状态",
  },
  drive: {
    label: "网盘",
    icon: "FolderOpen",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/file-management",
    description: "网盘功能开关",
  },
  image: {
    label: "镜像",
    icon: "HardDrive",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/image-management",
    description: "Agent 运行镜像",
  },
  cloudResource: {
    label: "资源配置",
    icon: "Cloud",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/agent-template",
    description: "配置新建 Agent 实例使用的实例规格、计费模式、系统盘和公网资源策略",
  },
  network: {
    label: "网络",
    icon: "ShieldCheck",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/network-management",
    description: "私有网络、子网与安全组配置",
  },
  cls: {
    label: "CLS 日志服务",
    icon: "Gauge",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/ops-observation",
    description: "用于运维观测与会话管理",
  },
  aiAgentSecurity: {
    label: "AI Agent 安全",
    icon: "Shield",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/security-management",
    description: "AI Agent 安全防护开关",
  },
  platformPolicy: {
    label: "平台策略",
    icon: "Shield",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/platform-policy",
    description: "用户配额、模型配额与功能权限开关",
  },
  cloudDev: {
    label: "云开发",
    icon: "Cloud",
    color: "text-blue-600",
    bg: "bg-blue-50",
    path: "/admin/cloud-dev",
    description: "云开发相关配额和配置内容",
  },
};

// ─── 同步异常 Mock ──────────────────────────────────────────
/** 模拟：运营组 在腾讯统一身份管理平台被删除了，但在管控端仍有配置绑定 */
export const MOCK_ANOMALOUS_GROUP: AnomalousGroup = {
  groupId: "dept-operation",
  groupName: "A公司/产品部/运营组",
  memberCount: 5,
  boundConfigs: ["模型", "通道", "Agent 工具"],
  agentInstanceCount: 4,
};

// ─── 用户在组织中创建的 Agent 实例（mock，用于移除组织时检测存量） ────────────
/** 格式：userId -> groupId -> 实例列表 */
export const MOCK_USER_GROUP_AGENTS: Record<string, Record<string, Array<{ id: string; name: string }>>> = {
  "fiona@acompany.com": {
    "mgrp-rd-fe": [
      { id: "claw-fiona-1", name: "Fiona 的前端助手" },
    ],
  },
  "lucas@acompany.com": {
    "mgrp-rd-be": [
      { id: "claw-lucas-1", name: "Lucas 的后端服务" },
      { id: "claw-lucas-2", name: "Lucas 的 API 测试" },
    ],
  },
  "mia@acompany.com": {
    "mgrp-design": [
      { id: "claw-mia-1", name: "Mia 的设计稿助手" },
    ],
  },
  "alice@acompany.com": {
    // alice 归属「前端基础架构与工程效能研发协作组」(og-frontend，可改上级)
    "og-frontend": [
      { id: "claw-alice-1", name: "Alice 的代码助手" },
      { id: "claw-alice-2", name: "Alice 的文档生成器" },
      { id: "claw-alice-3", name: "Alice 的测试工具" },
    ],
  },
  "bob@acompany.com": {
    // bob 归属「后端研发同学」(og-backend，可改上级)
    "og-backend": [
      { id: "claw-bob-1", name: "Bob 的组件库助手" },
    ],
  },
  // 普通模式：ryan 未分配组织状态下有实例（用于 2.4c 场景：未分配组织→加入新组织）
  "ryan@acompany.com": {
    "__global__": [
      { id: "claw-ryan-1", name: "Ryan 的通用助手" },
      { id: "claw-ryan-2", name: "Ryan 的数据分析工具" },
    ],
  },
  // 普通模式：tina 只在 mgrp-qa 单个组织，无其他归属，移出后无任何组织（演示回退变体 + 移交禁用）
  "tina@acompany.com": {
    "mgrp-qa": [
      { id: "claw-tina-1", name: "Tina 的测试管理助手" },
      { id: "claw-tina-2", name: "Tina 的缺陷追踪工具" },
      { id: "claw-tina-3", name: "Tina 的性能分析助手" },
    ],
  },
};

/** 模拟同步结果（刷新/手动同步后返回） */
export const MOCK_SYNC_RESULT: SyncResult = {
  anomalousGroups: [
    {
      groupId: "dept-operation",
      groupName: "A公司/产品部/运营组",
      memberCount: 5,
      boundConfigs: ["模型", "通道", "Agent 工具"],
      agentInstanceCount: 4,
    },
    {
      groupId: "dept-operation-1",
      groupName: "A公司/产品部/运营组/运营一组",
      memberCount: 3,
      boundConfigs: ["模型", "通道"],
      agentInstanceCount: 2,
    },
    {
      groupId: "dept-operation-2",
      groupName: "A公司/产品部/运营组/运营二组",
      memberCount: 2,
      boundConfigs: ["模型", "通道"],
      agentInstanceCount: 0,
    },
  ],
  anomalousUsers: [
    {
      userId: "henry@acompany.com",
      displayName: "henry",
      reason: "主部门「人力资源」在腾讯统一身份管理平台已失效",
    },
  ],
};

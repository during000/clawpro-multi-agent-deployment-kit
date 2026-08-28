/**
 * MemberManagement 子模块类型（PRD v2.0）
 *
 * 核心变化：
 *   - 废弃 Department 表，统一进 UserGroup
 *   - UserGroup 支持多层级（parentId）+ 多来源（source）+ 只读标记
 *   - Scope 只保留 groupIds（去除 deptIds）
 *   - 用户多归属：groupIds[]，primaryGroupId 仅 OneID 模式有值（即 OneID 主部门对应的只读组织）
 */

// ─── 应用范围（PRD §5.1） ─────────────────────────────────
export type Scope =
  | { type: "all" }
  | {
      type: "filtered";
      /** 作用到的组织 id 列表（组织本身已能表达"组织架构节点 + 用户组 + 自建组织"） */
      groupIds: string[];
      /** 是否同时作为未分配组织用户的兜底配置。
       *  true 表示不属于任何组织的用户将使用此配置作为保底。 */
      isFallback?: boolean;
    };

// ─── 组织来源 ─────────────────────────────────────────────
export type GroupSource =
  | "manual" // ClawPro 内管理员自建（普通模式唯一；OneID 模式不存在）
  | "oneid-dept" // OneID 同步的组织架构节点（只读）
  | "oneid-group" // OneID 同步的用户组（只读）
  | "project"; // 项目（特殊组织：拥有组织的全部能力，独立分组展示，跨页共享同步）

// ─── 组织（统一结构） ─────────────────────────────────────
export interface UserGroup {
  id: string;
  name: string;
  /** 项目基本信息；仅 source="project" 使用，用于管控端与用户端项目表单保持同一字段口径 */
  description?: string;
  goal?: string;
  /** 是否允许普通项目成员编辑项目信息与资产；仅 source="project" 使用 */
  allowMemberEdit?: boolean;
  /** 父组织；根节点为 null。跨不同来源的节点允许同处一棵树（顶层虚拟根分 bucket） */
  parentId: string | null;
  source: GroupSource;
  /** 只读节点不能重命名/移动/删除，仅能「取消同步」（oneid-dept / oneid-group 默认 true） */
  readonly: boolean;
  /** 来源侧的外键 id（OneID 的部门 id / 用户组 id），manual 组织为 undefined */
  externalId?: string;
  /** 同步来源的 "organization code" 即批次/组织标识。用于区分"哪次同步带进来的一棵子树"，
   *  取消同步时按该字段级联清理。manual 为 undefined */
  syncBatchId?: string;
  createdAt: string; // ISO
}

// ─── 用户归属 ─────────────────────────────────────────────
export interface UserOrg {
  userId: string;
  displayName: string;
  /** 用户归属的组织 id 列表（多归属）。
   *  在 OneID 模式下，该列表 = OneID 所有部门（主部门+兼任）+ OneID 所有用户组。
   *  在普通模式下，该列表 = ClawPro 管理员给该用户勾选的自建组织集合。 */
  groupIds: string[];
  /** OneID 主部门对应的组织 id（仅 OneID 模式有值）。用于「主部门」列展示 + 判定"主部门缺失"。 */
  primaryGroupId: string | null;
  /** 主部门所指组织是否仍存在且有效（OneID 侧删除后会变 false） */
  primaryGroupValid: boolean;
  /** 用户角色 */
  role?: "admin" | "member";
  /** 用户状态 */
  status?: "active" | "disabled";
}

// ─── 资源（旧 kind，仍用于健康度校验等） ─────────────────
export type ResourceKind =
  | "model"
  | "channel"
  | "securityGroup"
  | "vpc"
  | "memory"
  | "image"
  | "agentTool";

export interface ResourceItem {
  id: string;
  kind: ResourceKind;
  name: string;
  scope: Scope;
  isPlatformDefault?: boolean;
  createdAt: string; // ISO
}

// ─── 配置总览用的统一配置项 ─────────────────────────────
/** 13 大配置项 */
export type ConfigCategory =
  | "model"
  | "channel"
  | "skill"
  | "agentTool"
  | "memory"
  | "drive"
  | "image"
  | "cloudResource"
  | "network"
  | "cls"
  | "aiAgentSecurity"
  | "platformPolicy"
  | "cloudDev";

/** 配置条目的来源 */
export type ConfigSource =
  | { type: "local"; groupName: string }        // 本组织
  | { type: "inherited"; groupName: string }     // 继承自某上层组织
  | { type: "platformDefault" }                  // 平台默认
  | { type: "presetPolicy" };                    // 预设策略（仅 VPC 和平台策略）

/** 统一配置条目 */
export interface ConfigEntry {
  id: string;
  category: ConfigCategory;
  /** 主标签（如模型名、技能包名、"开启 Pro 版" 等） */
  label: string;
  /** 可选副标签（如技能下的子分类"初始技能包"/"角色"） */
  subLabel?: string;
  /** 来源信息 */
  source: ConfigSource;
  /** 配置项自身的版本号（如企业技能 v1.2.0，用于自动同步时检测版本更新） */
  version?: string;
  /** 额外 key-value（公网带宽 / 平台策略的配额字段等） */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  meta?: Record<string, any>;
}

// ─── 节点健康度 ───────────────────────────────────────────
/** 三大核心：模型 / 通道 / 安全组，缺任一即异常。考虑继承 + 全部用户档 + 平台默认兜底。 */
export type NodeHealth = {
  healthy: boolean;
  missing: Array<"model" | "channel" | "securityGroup">;
};

// ─── 节点初始化健康度 ─────────────────────────────────────
/** 初始化检查：模型 / 通道 / 镜像 / 网络（VPC+安全组），缺任一即未完成初始化。 */
export type InitHealth = {
  initialized: boolean;
  missing: Array<"model" | "channel" | "image" | "network">;
};

// ─── 覆盖状态 ─────────────────────────────────────────────
export type OverrideStatus =
  | "local" // 按本节点
  | "groupOverride" // 组织覆盖（amber）
  | "groupConflict" // 多组织对同唯一型资源冲突（red，可裁决）
  | "primaryDeptMissing"; // OneID 主部门失效（red）

export interface UserOverrideInfo {
  userId: string;
  status: OverrideStatus;
  conflictResourceKind?: ResourceKind;
  conflictCandidates?: Array<{
    resourceId: string;
    resourceName: string;
    via: string; // "研发组" / "设计组" 这样的来源文案
    latestBindingAt: string;
  }>;
  winnerResourceId?: string;
  isResolved?: boolean;
}

// ─── 同步异常组织（组织架构被删除但仍有配置绑定） ────────────
export interface AnomalousGroup {
  /** 异常组织 id */
  groupId: string;
  /** 组织名称 */
  groupName: string;
  /** 组织总人数（同步前） */
  memberCount: number;
  /** 已应用的配置类别名称列表（如 ["模型", "通道", "Agent 工具"]） */
  boundConfigs: string[];
  /** 组织下 Agent 实例数 */
  agentInstanceCount: number;
}

/** 同步结果（组织异常 + 用户异常） */
export interface SyncResult {
  /** 异常组织列表（组织架构已被删除，但仍有配置绑定） */
  anomalousGroups: AnomalousGroup[];
  /** 异常用户列表（主部门失效等） */
  anomalousUsers: Array<{
    userId: string;
    displayName: string;
    reason: string;
  }>;
}

// ─── 最终生效配置（用于详情 Popover / 抽屉） ──────────────
export interface EffectiveConfig {
  userId: string;
  /** OneID 主部门全路径（无则 "—"） */
  primaryDeptPath: string;
  /** 主部门是否有效 */
  primaryValid: boolean;
  /** 全部归属组织名（OneID 模式含部门+用户组，普通模式仅自建组织） */
  groupNames: string[];
  // 加法型
  models: string[];
  channels: string[];
  skills: string[];
  agentTools: string[];
  // 唯一型
  securityGroup: string | null;
  vpc: string | null;
  memory: string | null;
  image: string | null;
}

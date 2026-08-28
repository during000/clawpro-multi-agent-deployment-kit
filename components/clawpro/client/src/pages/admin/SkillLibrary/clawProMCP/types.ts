/**
 * ClawPro 平台 MCP 数据模型
 */

// ────────────────────────────────────────────────────────────
// 能力 Capability
// ────────────────────────────────────────────────────────────

/** 能力分类 */
export type CapabilityCategory = 'query' | 'distribute' | 'config' | 'ops' | 'danger';

/** 风险等级 */
export type CapabilityRisk = 'low' | 'medium' | 'high';

/** 分期：决定能力在 UI 里是否可勾选 */
export type CapabilityPhase = 1 | 2 | 3;

/**
 * 能力/模块所属角色范围
 * - admin：管控端（走 /admin/* 接口）
 * - member：用户端（走 /openclaw/* 接口）
 */
export type CapabilityRoleScope = 'admin' | 'member';

/** 能力所属模块 id（slug 形式，如 'enterprise-skills' / 'my-instances'） */
export type CapabilityModuleId = string;

/** 模块定义 */
export interface ModuleDef {
  /** slug 形式的 id */
  id: string;
  /** 中文展示名，如「企业技能库」 */
  label: string;
  description: string;
  /** 该模块面向的角色 */
  roleScope: CapabilityRoleScope;
}

/** 单个能力定义 */
export interface Capability {
  /** 工具名（MCP tool name） */
  id: string;
  /** 中文展示名 */
  label: string;
  /** 一句话描述 */
  description: string;
  category: CapabilityCategory;
  risk: CapabilityRisk;
  /** 属于第几期；非第一期的能力 UI 灰显并标"即将上线" */
  phase: CapabilityPhase;
  /** 所属模块 id（对应 ModuleDef.id） */
  module: CapabilityModuleId;
  /** 该工具面向的角色（应与所属模块的 roleScope 一致） */
  roleScope: CapabilityRoleScope;
  /** 对应的后端接口路径（仅用于展示，便于排查） */
  backendApi?: string;
}

/** 能力开关配置（按角色） */
export interface CapabilityToggle {
  /** 管理员是否可用 */
  admin: boolean;
  /** 用户端是否可用；用户端调用时另在接口层做资源归属校验（仅自己的 Agent） */
  member: boolean;
}

/** 全部能力开关：capabilityId -> 开关 */
export type CapabilityToggles = Record<string, CapabilityToggle>;

// ────────────────────────────────────────────────────────────
// Token 凭据
// ────────────────────────────────────────────────────────────

/** 用户角色（与 UserRoleContext 对齐：admin / member） */
export type UserRole = 'admin' | 'member';

/** Token 状态 */
export type TokenStatus = 'active' | 'disabled';

/** 一个用户的 Token 记录 */
export interface UserToken {
  /** 用户 ID（与成员管理对齐，使用 email 作为 id） */
  userId: string;
  /** 用户显示名 */
  userName: string;
  /** 角色（决定能调哪些能力） */
  role: UserRole;
  /**
   * 完整 Token 值；
   * 真实环境下应只在生成/轮换瞬间返回明文一次，列表只存掩码 + hash。
   * Mock 阶段为简化先保存完整值。
   */
  token: string;
  /** Token 掩码（用于列表展示，如 hk-***ax8f） */
  tokenMask: string;
  status: TokenStatus;
  /** 创建时间 */
  createdAt: string;
  /** 最近一次调用时间（ISO 字符串，空表示从未调用） */
  lastUsedAt?: string;
}

// ────────────────────────────────────────────────────────────
// 已下发 Agent
// ────────────────────────────────────────────────────────────

/** 下发来源 */
export type DistributionSource = 'manual' | 'auto';

/** 已下发到的 Agent 记录 */
export interface DistributedAgent {
  agentId: string;
  agentName: string;
  /** Agent 所属用户 ID */
  ownerUserId: string;
  /** Agent 所属用户名 */
  ownerUserName: string;
  /** 注入的 Token 所属角色 */
  ownerRole: UserRole;
  /** 注入的 Token 掩码 */
  injectedTokenMask: string;
  /** 下发来源 */
  source: DistributionSource;
  /** 下发时间 */
  distributedAt: string;
}

// ────────────────────────────────────────────────────────────
// 自动装载策略
// ────────────────────────────────────────────────────────────

/** 自动装载适用范围 */
export type AutoInstallScope =
  | 'all'         // 全部新建 Agent
  | 'adminOnly'   // 仅管理员新建的 Agent
  | 'custom';     // 仅指定用户/分组

/** 自动装载策略 */
export interface AutoInstallPolicy {
  /** 总开关，默认 false */
  enabled: boolean;
  /** 适用范围 */
  scope: AutoInstallScope;
  /** scope=custom 时生效：指定的用户 ID 列表 */
  scopeUserIds?: string[];
  /** scope=custom 时生效：指定的分组 ID 列表 */
  scopeGroupIds?: string[];
  /** 若目标用户尚无 token，是否自动生成 */
  autoGenerateToken: boolean;
  /** 永不自动装载的例外用户 ID 列表 */
  excludeUserIds: string[];
  /** 永不自动装载的例外 Agent ID 列表 */
  excludeAgentIds: string[];
  /** 最近一次更新时间 */
  updatedAt: string;
}

// ────────────────────────────────────────────────────────────
// 调用日志（第一期占位）
// ────────────────────────────────────────────────────────────

export type CallLogResult = 'success' | 'failed' | 'denied_permission' | 'denied_capability';

export interface CallLog {
  id: string;
  timestamp: string;
  userId: string;
  userName: string;
  role: UserRole;
  capabilityId: string;
  /** 目标 Agent（如有） */
  targetAgentId?: string;
  targetAgentName?: string;
  /** 入参摘要 */
  paramsSummary: string;
  result: CallLogResult;
  /** 耗时（ms） */
  durationMs?: number;
  /** 失败/拒绝原因 */
  message?: string;
}

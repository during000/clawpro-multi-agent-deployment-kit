/**
 * 统一下发状态枚举
 * - not_distributed: 未下发
 * - distributing: 下发中
 * - success: 下发完成
 * - failed: 下发失败
 */
export type DistributionStatus = 'not_distributed' | 'distributing' | 'success' | 'failed';

/** 下发状态显示映射 */
export const DISTRIBUTION_STATUS_MAP: Record<DistributionStatus, { label: string; color: string }> = {
  not_distributed: { label: '未下发', color: 'text-gray-500 bg-gray-50' },
  distributing:    { label: '下发中', color: 'text-blue-600 bg-blue-50' },
  success:         { label: '成功', color: 'text-green-700 bg-green-50' },
  failed:          { label: '失败', color: 'text-red-700 bg-red-50' },
};

/** 版本历史记录 */
export interface SkillVersionRecord {
  version: string;
  date: string; // ISO 日期字符串
  changeLog?: string;
  files?: Array<{ name: string; size: number; content?: string }>;
}

export interface Skill {
  id: string;
  slug: string;
  name: string;
  description: string;
  version: string;
  categories: string[];
  /** 应用范围：public=全部用户，private=按组织 */
  scope: SkillScope;
  /** 当 scope=private 时，关联的组织 ID 列表 */
  groupIds: string[];
  uploadTime: Date;
  content?: string;
  fileContent?: string;
  versions?: string[];
  files?: Array<{ name: string; size: number; content?: string }>;
  lastDistributionStatus?: DistributionStatus;
  lastDistributionProgress?: number; // 0-100
  lastDistributionTime?: Date;
  lastDistributionInstanceCount?: number;
  lastDistributionSuccessCount?: number;
  /** 版本历史记录 */
  versionHistory?: SkillVersionRecord[];
  /** 安全检测信息 */
  securityInfo?: SkillSecurityInfo;
  /**
   * 审核状态：
   * - 'pending'：员工提交的待审核 Skill，混排在企业技能库中但仅显示「审核」操作
   * - 'normal' 或 undefined：管理员发布 / 已通过审核，进入正常运营态
   * - 'offlined'：已下架（管控端仍可见，可继续下发/管理；员工端搜不到，不影响已安装的使用）
   */
  reviewStatus?: 'pending' | 'normal' | 'offlined';
  /**
   * 审核申请类型（仅 reviewStatus='pending' 时有意义）：
   * - 'publish'：员工发布申请（审核通过 → normal；驳回 → 从列表删除）
   * - 'offshelf'：员工下架申请（审核通过 → offlined；驳回 → 回 normal）
   */
  reviewType?: 'publish' | 'offshelf';
  /** 申请人（仅 reviewStatus='pending' 时展示） */
  applicant?: string;
  /** 申请时间（ISO 或本地化字符串，仅 reviewStatus='pending' 时展示） */
  submittedAt?: string;
  /** 下架申请原因（仅 reviewType='offshelf' 时展示） */
  offshelfReason?: string;
}

export interface UploadedFile {
  name: string;
  size: number;
  status: 'success' | 'error' | 'parsing';
  error?: string;
  skillmdContent?: string;
  skillmdParsed?: {
    name?: string;
    description?: string;
  };
  pluginJsonFound?: boolean;
  packageJsonFound?: boolean;
  pluginJsonParsed?: {
    name?: string;
    description?: string;
  };
  files?: Array<{ name: string; size: number; content?: string }>;
}

export interface Category {
  id: string;
  name: string;
  description: string;
}

/** 组织（Group） */
export interface Group {
  id: string;
  name: string;
  description?: string;
}

/** 应用范围类型 */
export type SkillScope = 'public' | 'private' | 'groups';

/** 实例运行状态 */
export type InstanceStatus = 'running' | 'stopped' | 'starting' | 'error';

/** 实例运行状态显示映射 */
export const INSTANCE_STATUS_MAP: Record<InstanceStatus, { label: string; color: string }> = {
  running:  { label: '运行中', color: 'text-green-700 bg-green-50' },
  stopped:  { label: '已停止', color: 'text-gray-500 bg-gray-50' },
  starting: { label: '启动中', color: 'text-blue-600 bg-blue-50' },
  error:    { label: '异常', color: 'text-red-700 bg-red-50' },
};

/** Agent 类型 */
export type AgentType = 'OpenClaw' | 'LocalAgent' | 'Other';

export interface AgentInstance {
  id: string;
  name: string;
  createdBy: string;
  status: InstanceStatus;
  createdAt: string; // ISO 日期字符串
  distributionStatus?: DistributionStatus;
  /** 已下发的版本号（用于判断"待更新"状态） */
  distributedVersion?: string;
  /** 所属组织 ID 列表（可能属于多个组织） */
  groupIds: string[];
  /** 下发失败原因（仅 distributionStatus 为 failed 时有值） */
  failReason?: string;
  /** Agent 类型 */
  agentType?: AgentType;
  /** Agent 版本号 */
  agentVersion?: string;
  /** 本地客户端产品，仅 LocalAgent 使用 */
  localProduct?: 'CodeBuddy' | 'WorkBuddy';
}

export interface DistributionRecord {
  id: string;
  skillId: string;
  timestamp: Date;
  totalCount: number;
  successCount: number;
  failedCount: number;
  status: DistributionStatus;
  instances: AgentInstance[];
}

export interface BucketInfo {
  name: string;
  region: string;
  storageGB: number;
}

/** OpenClawInstance 是 AgentInstance 的别名，供 mockData.ts 使用 */
export type OpenClawInstance = AgentInstance;

// ── MCP 服务相关类型 ──────────────────────────────────────

/** MCP 底层传输协议 */
export type MCPTransportType = 'sse' | 'streamable-http' | 'stdio';

/** MCP 连接类别（用户可见的分类） */
export type MCPConnectionCategory = 'remote' | 'local';

/** 连接类别显示映射 */
export const MCP_CONNECTION_CATEGORY_MAP: Record<MCPConnectionCategory, { label: string; description: string }> = {
  remote: { label: '远程服务', description: '通过 HTTP 协议连接远程 MCP 服务' },
  local:  { label: '本地命令', description: '通过标准输入/输出 (STDIO) 启动本地进程' },
};

/** 远程服务的协议子选项 */
export const MCP_REMOTE_PROTOCOL_MAP: Record<'streamable-http' | 'sse', { label: string; tag?: string }> = {
  'streamable-http': { label: 'Streamable HTTP', tag: '推荐' },
  'sse':             { label: 'SSE', tag: '兼容旧版' },
};

/** MCP 连接方式显示映射（保留用于列表展示等场景） */
export const MCP_TRANSPORT_MAP: Record<MCPTransportType, { label: string; description: string }> = {
  'sse':              { label: 'SSE',              description: '远程服务（兼容旧版 MCP 2024-11-05）' },
  'streamable-http':  { label: 'Streamable HTTP',  description: '远程服务（推荐，MCP 2025-06-18）' },
  'stdio':            { label: 'STDIO',            description: '本地命令，通过标准输入输出通信' },
};

/** MCP 服务配置 */
export interface MCPService {
  /** 服务标识（唯一 key），对应 JSON 中 mcp.servers.{name}，创建后不可修改 */
  name: string;
  /** 显示别名（可选，可修改），用于列表和详情页展示 */
  displayName?: string;
  description: string;
  /** 版本号 */
  version: string;
  /** 版本历史列表（从旧到新排列） */
  versions?: string[];
  transport: MCPTransportType;
  /** JSON 格式的服务配置 */
  configJson: string;
  /** 使用说明 (Markdown) */
  usageDoc?: string;
  /** 工具说明 (Markdown) */
  toolDoc?: string;
  /** 是否开启凭据托管 */
  credentialHostingEnabled?: boolean;
  /** IP 白名单（仅在 credentialHostingEnabled 为 true 时有效） */
  ipWhitelist?: string[];
  /** 凭据 Token（仅在 credentialHostingEnabled 为 true 时有效） */
  token?: string;
  /** 应用范围：public=全部用户，private=按组织 */
  scope: SkillScope;
  /** 当 scope=private 时，关联的组织 ID 列表 */
  groupIds: string[];
  createdAt: Date;
  updatedAt: Date;
}

/** 应用范围锁定描述：用于「项目资产管理」内嵌上传/更新弹窗时，将应用范围强制锁定为指定组织 */
export interface ScopeLockConfig {
  /** 锁定的组织 ID */
  lockedGroupId: string;
  /** 锁定的组织名称（用于只读展示） */
  lockedGroupName: string;
}

// ========== 安全检测相关 ==========

/** 安全检测状态 */
export type SecurityStatus = 'not_scanned' | 'scanning' | 'safe' | 'suspicious' | 'malicious';

/** 安全检测状态显示映射 */
export const SECURITY_STATUS_MAP: Record<SecurityStatus, { label: string; color: string; bgColor: string }> = {
  not_scanned: { label: '未检测', color: 'text-gray-400', bgColor: 'bg-gray-50' },
  scanning:    { label: '安全检测中', color: 'text-blue-600', bgColor: 'bg-blue-50' },
  safe:        { label: '安全', color: 'text-green-600', bgColor: 'bg-green-50' },
  suspicious:  { label: '可疑', color: 'text-yellow-600', bgColor: 'bg-yellow-50' },
  malicious:   { label: '恶意', color: 'text-red-600', bgColor: 'bg-red-50' },
};

/** 单引擎安全检测维度结果 */
export type SecurityDimensionStatus = 'safe' | 'suspicious' | 'malicious';

/** 安全检测维度 */
export interface SecurityDimension {
  name: string;
  status: SecurityDimensionStatus;
  detail: string;
}

/** 单引擎检测结果 */
export interface EngineSecurityResult {
  engineName: string;
  status: SecurityDimensionStatus;
  reportUrl: string;
  score?: number;
  dimensions: SecurityDimension[];
}

/** 技能安全检测信息 */
export interface SkillSecurityInfo {
  overallStatus: SecurityStatus;
  contentHash?: string;
  engines: EngineSecurityResult[];
}

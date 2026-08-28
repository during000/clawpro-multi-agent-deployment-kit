/**
 * 自定义 Agent 类型定义（精简版：已剔除"功能脚本"相关字段）
 *
 * 设计：
 *   - 兼容内核（kernelBase）决定管控台行为：
 *       OpenClaw / HermesAgent / LightClawACE → 直接复用对应系统脚本，无需配置
 *       native（自研）                        → 部分管控台功能不可用，需用户显式确认"允许进入终端"
 */

// ── 兼容内核枚举 ────────────────────────────────────────────────
/**
 * 自定义 Agent 类型的内核兼容模式：
 *   - OpenClaw / HermesAgent / LightClawACE：完全兼容对应系统内核，直接复用其管控能力
 *   - DeepSeekHarness：完全兼容 DeepSeek Harness 内核（不作为独立系统类型出现在类型表中）
 *   - native：完全自研，与已知内核均不兼容；部分管控台功能将不可用
 */
export type KernelBase =
  | "OpenClaw"
  | "HermesAgent"
  | "LightClawACE"
  | "DeepSeekHarness"
  | "native";

// ── 自定义 Agent 类型定义 ────────────────────────────────────────
export interface CustomAgentType {
  /** 类型 ID（小写字母数字，对应后端 agent_type 字段） */
  id: string;
  /** 显示名 */
  displayName: string;
  /** 兼容的内核基线（决定管控台行为） */
  kernelBase: KernelBase;
  /** 描述（可选） */
  description?: string;
  /** 创建时间 */
  createdAt: string;
  /** 最近更新时间 */
  updatedAt: string;
  /** 关联实例数（展示用，mock） */
  linkedInstanceCount: number;
  /** 创建人 */
  createdBy: string;
  /**
   * 仅 native 内核相关：用户在创建时已勾选确认"允许用户进入该类型 Agent 的终端"。
   * 该字段为审计字段，便于将来追溯。
   */
  nativeTerminalAck?: boolean;
}

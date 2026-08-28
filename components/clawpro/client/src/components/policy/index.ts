/**
 * 策略卡片组件统一导出
 *
 * 用法：
 *   import { PolicyEditCard, QuotaPolicyCard, type PolicyRule } from "@/components/policy";
 *
 * 设计要点：
 *   - 卡片本身仅负责视图与编辑态状态机，与具体组织数据解耦
 *   - 组织渲染（GroupTagSelector / GroupBadges）通过 renderGroupSelector / renderGroupBadges 注入
 *   - 类型集中在 ./types.ts
 */
export { PolicyEditCard } from "./PolicyEditCard";
export type { PolicyEditCardProps } from "./PolicyEditCard";

export { QuotaPolicyCard } from "./QuotaPolicyCard";
export type { QuotaPolicyCardProps } from "./QuotaPolicyCard";

export { EnumPolicyCard } from "./EnumPolicyCard";
export type { EnumPolicyCardProps, EnumPolicyOption } from "./EnumPolicyCard";

export { TokenValueEditor } from "./TokenValueEditor";
export type { TokenValueEditorProps } from "./TokenValueEditor";

export type {
  PolicyRule,
  TokenLimit,
  AccessModeRowConfig,
  TimeDimensionConfig,
  GroupRenderProps,
} from "./types";

/**
 * 策略卡片共享类型
 *
 * - PolicyRule<T>：通用策略规则（groupIds 为空数组 = 全部用户兜底）
 * - TokenLimit：Token / 数额型规则的取值（数字或 "unlimited"）
 * - AccessModeRowConfig：访问方式行（用于 PolicyEditCard 可选行）
 */

export interface PolicyRule<T> {
  id: string;
  groupIds: string[]; // 空数组 = 全部用户
  value: T;
}

export type TokenLimit = number | "unlimited";

export interface AccessModeRowConfig {
  /** 当前访问方式 */
  mode: "public" | "private";
  /** 保存回调 */
  onModeChange: (m: "public" | "private") => void;
  /** info tooltip 内容 */
  tooltipContent: React.ReactNode;
}

/** 时间维度配置（用于 QuotaPolicyCard 配额值右侧追加列） */
export interface TimeDimensionConfig {
  value: "daily" | "monthly";
  onChange: (v: "daily" | "monthly") => void;
}

/**
 * 组织渲染上下文：通过 render prop 由调用方注入组织相关的 UI
 * - 解耦数据层（ALL_GROUPS / 树形结构）与视图层（卡片）
 */
export interface GroupRenderProps {
  /** 编辑态：渲染组织选择器（含已选标签 + 树形 Popover） */
  renderGroupSelector: (params: {
    selectedIds: string[];
    /** 其他规则已占用的 groupIds（在选择器中应禁用） */
    disabledIds: string[];
    onChange: (ids: string[]) => void;
  }) => React.ReactNode;
  /** 视图态：渲染组织徽章列表 */
  renderGroupBadges: (groupIds: string[]) => React.ReactNode;
}

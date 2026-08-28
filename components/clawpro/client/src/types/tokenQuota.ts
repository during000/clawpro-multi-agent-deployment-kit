/**
 * Token 配额规则相关类型
 *
 * 后端约定:
 *   - mode=day|month|year:limit;(start/end/refresh 忽略)
 *   - mode=custom:limit + start (必填, ≥ 当前) + end (可空) + refresh
 *   - limit:正整数;-1 表示"不限"
 *   - 同一数组内 mode 不可重复(后端校验)
 *   - 预设规则强制 end=null
 */

/** 自定义周期内的刷新方式(后端契约:custom 模式下的 refresh 字段) */
export type TokenQuotaRefresh = "daily" | "monthly" | "yearly" | "none";

/** 规则模式:自然周期 day/month/year,或自定义 custom(对齐后端 API.md) */
export type TokenQuotaMode = "day" | "month" | "year" | "custom";

/** 单条规则(后端契约)
 *
 * 注意:
 * - start / end 字段在与后端通信时是 **Unix 秒(number)**,而不是 ISO 字符串
 * - 前端 UI 态(TokenQuotaPolicyValue)用 ISO 字符串,提交前由 utils 转 Unix 秒,读取后由 utils 转 ISO 字符串
 */
export interface TokenQuotaRule {
  mode: TokenQuotaMode;
  limit: number;          // -1 = 无限
  start?: number | null;  // Unix 秒;custom 必填,且 ≥ 当前;其他 mode 忽略
  end?: number | null;    // Unix 秒;custom 可空(空表示无终止);预设规则强制 null
  refresh?: TokenQuotaRefresh; // custom 必填;预设规则忽略
}

/** 单条规则当前生效窗口的用量(rule_index 对应同源 rules 数组的下标) */
export interface TokenQuotaUsage {
  rule_index: number;
  used: number;
  /** 当前生效窗口起始时间(Unix 秒) */
  period_start?: number;
  /** 当前生效窗口结束时间(Unix 秒);无终止的生效窗口返回 null */
  period_end?: number | null;
  /** 当前是否有生效窗口;false 时 period_start=0, period_end=0 */
  active?: boolean;
}

/** 卡片顶部"周期类型"二选一 */
export type CycleKind = "natural" | "custom";

/** 自然周期下的刷新粒度(对齐后端 mode 字面量) */
export type NaturalPeriod = "day" | "month" | "year";

/**
 * PolicyRule<T> 中给"配额型"策略使用的 value(判别联合)
 * 预设(fallback)规则强制 end=null
 */
export type TokenQuotaPolicyValue =
  | {
      cycleKind: "natural";
      naturalPeriod: NaturalPeriod;
      limit: number | "unlimited";
    }
  | {
      cycleKind: "custom";
      start: string;
      end: string | null;
      refresh: TokenQuotaRefresh;
      limit: number | "unlimited";
    };

/** 用户所属分组下的 Token 配额信息（group_by=user 且未传 group_id 时返回） */
export interface TokenQuotaGroup {
  group_id: number;
  group_name: string;
  group_full_path: string;
  token_quota_rules: TokenQuotaRule[];
  token_quota_usages: TokenQuotaUsage[];
}

/** 用户实际生效规则的来源(用于 hover card 展示) */
export type EffectiveRuleSource = "group" | "user" | "default";

/** resolveUserEffectiveRule 返回值 */
export interface EffectiveRuleResult {
  rules: TokenQuotaRule[];
  source: EffectiveRuleSource;
  sourceGroupId?: number;
}

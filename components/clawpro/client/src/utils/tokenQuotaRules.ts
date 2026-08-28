/**
 * Token 配额规则工具函数
 * - 前后端类型互转
 * - 兼容旧字段 fallback
 * - 当前生效窗口计算
 * - 进度条颜色阈值
 * - 文案格式化
 * - 用户生效规则解析
 */
import moment from "moment";
import type {
  TokenQuotaRule,
  TokenQuotaPolicyValue,
  NaturalPeriod,
  TokenQuotaRefresh,
  TokenQuotaMode,
  EffectiveRuleResult,
} from "@/types/tokenQuota";

/** 进度条颜色阈值(%) */
export const PROGRESS_THRESHOLDS = { red: 80, orange: 60 } as const;

export function progressColor(percent: number): "red" | "orange" | "blue" {
  if (percent >= PROGRESS_THRESHOLDS.red) return "red";
  if (percent >= PROGRESS_THRESHOLDS.orange) return "orange";
  return "blue";
}

/** 自然周期 mode → 自定义 refresh 映射 */
const NATURAL_TO_REFRESH: Record<NaturalPeriod, TokenQuotaRefresh> = {
  day: "daily",
  month: "monthly",
  year: "yearly",
};

/** custom refresh → 切回自然时的粒度映射(refresh 后端用 yearly,mode 用 year) */
const REFRESH_TO_NATURAL: Record<TokenQuotaRefresh, NaturalPeriod> = {
  daily: "day",
  monthly: "month",
  yearly: "year",
  none: "day",
};

/** ISO 字符串(本地时区) → Unix 秒。空串/null 返 null */
function isoToUnix(s: string | null | undefined): number | null {
  if (!s) return null;
  const ms = new Date(s).getTime();
  if (Number.isNaN(ms)) return null;
  return Math.floor(ms / 1000);
}

/** Unix 秒 → ISO 字符串(本地时区);空返空串 */
function unixToIso(t: number | null | undefined): string {
  if (t == null || !Number.isFinite(t)) return "";
  return moment.unix(t).format("YYYY-MM-DDTHH:mm:ss");
}

/** 前端 value → 单条后端 rule。预设规则强制 end=null */
export function policyValueToRule(
  v: TokenQuotaPolicyValue,
  isFallback: boolean,
): TokenQuotaRule {
  let limit: number;
  if (v.limit === "unlimited") {
    limit = -1;
  } else {
    const n = Number(v.limit);
    limit = Number.isFinite(n) ? n : 0;
  }
  if (v.cycleKind === "natural") {
    return { mode: v.naturalPeriod, limit };
  }
  return {
    mode: "custom",
    limit,
    start: isoToUnix(v.start),
    end: isFallback ? null : isoToUnix(v.end),
    refresh: v.refresh,
  };
}

/** 后端 rules[] → 前端 value。空数组 / 多条:取第一条 */
export function ruleArrayToPolicyValue(
  rules: TokenQuotaRule[],
): TokenQuotaPolicyValue {
  if (!rules || rules.length === 0) {
    return { cycleKind: "natural", naturalPeriod: "day", limit: "unlimited" };
  }
  const r = rules[0];
  const limit: number | "unlimited" = r.limit === -1 ? "unlimited" : r.limit;
  if (r.mode === "custom") {
    return {
      cycleKind: "custom",
      start: unixToIso(r.start),
      end: r.end == null ? null : unixToIso(r.end),
      refresh: r.refresh ?? "daily",
      limit,
    };
  }
  return { cycleKind: "natural", naturalPeriod: r.mode, limit };
}

/** 当前时间的本地 ISO 字符串(精度到分,与 datetime-local input 一致) */
function nowIsoLocal(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` +
    `T${pad(d.getHours())}:${pad(d.getMinutes())}`
  );
}

/** 自然 → 自定义 切换时的预设规则构造。start 自动填当前时间(预设规则保存时后端会忽略 start) */
export function naturalToCustomRule(
  current: TokenQuotaPolicyValue,
): TokenQuotaPolicyValue {
  if (current.cycleKind === "custom") return current;
  return {
    cycleKind: "custom",
    start: nowIsoLocal(),
    end: null,
    refresh: NATURAL_TO_REFRESH[current.naturalPeriod],
    limit: current.limit,
  };
}

/** 自定义 → 自然 切换时的预设规则构造 */
export function customToNaturalRule(
  current: TokenQuotaPolicyValue,
): TokenQuotaPolicyValue {
  if (current.cycleKind === "natural") return current;
  return {
    cycleKind: "natural",
    naturalPeriod: REFRESH_TO_NATURAL[current.refresh],
    limit: current.limit,
  };
}

/** 给定 rule 与"现在",计算当前生效窗口 [start, end) */
export function currentWindow(
  rule: TokenQuotaRule,
  now: Date = new Date(),
): { start: Date; end: Date | null } {
  if (rule.mode === "day") {
    const start = moment(now).startOf("day").toDate();
    const end = moment(start).add(1, "day").toDate();
    return { start, end };
  }
  if (rule.mode === "month") {
    const start = moment(now).startOf("month").toDate();
    const end = moment(start).add(1, "month").toDate();
    return { start, end };
  }
  if (rule.mode === "year") {
    const start = moment(now).startOf("year").toDate();
    const end = moment(start).add(1, "year").toDate();
    return { start, end };
  }
  // custom
  const ruleStart = rule.start != null ? moment.unix(rule.start).toDate() : now;
  const ruleEnd = rule.end != null ? moment.unix(rule.end).toDate() : null;
  if (rule.refresh === "none") {
    return { start: ruleStart, end: ruleEnd };
  }
  const stepUnit: moment.unitOfTime.DurationConstructor =
    rule.refresh === "daily"
      ? "days"
      : rule.refresh === "monthly"
      ? "months"
      : "years";
  // 已过期:返回最后一个有效窗口(从 ruleStart 起按 step 推进,直到下一步会超过 ruleEnd 为止)
  if (ruleEnd && now >= ruleEnd) {
    let lastStart = moment(ruleStart);
    while (lastStart.clone().add(1, stepUnit).toDate() < ruleEnd) {
      lastStart.add(1, stepUnit);
    }
    return { start: lastStart.toDate(), end: ruleEnd };
  }
  // 进行中:推进到包含 now 的窗口
  let winStart = moment(ruleStart);
  while (
    winStart.clone().add(1, stepUnit).toDate() <= now &&
    (!ruleEnd || winStart.clone().add(1, stepUnit).toDate() < ruleEnd)
  ) {
    winStart.add(1, stepUnit);
  }
  let winEnd = winStart.clone().add(1, stepUnit).toDate();
  if (ruleEnd && winEnd > ruleEnd) winEnd = ruleEnd;
  return { start: winStart.toDate(), end: winEnd };
}

/** "当前周期"显示文案 */
export function periodLabel(rule: TokenQuotaRule): string {
  if (rule.mode === "day") return "当日累计";
  if (rule.mode === "month") return "当月累计";
  if (rule.mode === "year") return "当年累计";
  const w = currentWindow(rule);
  const fmt = (d: Date) => moment(d).format("YYYY-MM-DD HH:mm");
  return w.end ? `${fmt(w.start)} → ${fmt(w.end)}` : `${fmt(w.start)} → 无终止时间`;
}

/** "上限"显示文案。自然周期一行,自定义周期两行(数值+起止时间) */
export function formatQuotaDisplay(rule: TokenQuotaRule): string[] {
  const limitText = rule.limit === -1 ? "无限制" : rule.limit.toLocaleString();
  if (rule.mode === "day") return [`${limitText} / 每日`];
  if (rule.mode === "month") return [`${limitText} / 每月`];
  if (rule.mode === "year") return [`${limitText} / 每年`];
  const refreshText: Record<TokenQuotaRefresh, string> = {
    daily: "每日刷新",
    monthly: "每月刷新",
    yearly: "每年刷新",
    none: "不刷新",
  };
  const fmt = (t: number | null | undefined) =>
    t != null && Number.isFinite(t) ? moment.unix(t).format("YYYY-MM-DD HH:mm") : null;
  const startStr = fmt(rule.start);
  const endStr = fmt(rule.end);
  const line1 =
    rule.limit === -1
      ? "无限制"
      : `${limitText}(${refreshText[rule.refresh ?? "daily"]})`;
  const line2 = `${startStr ?? "保存时刻起"} → ${endStr ?? "无终止时间"}`;
  return [line1, line2];
}

/** 兼容旧字段 fallback */
export function normalizeQuotaRules(
  rulesField: TokenQuotaRule[] | string | null | undefined,
  dayField: number | undefined,
  periodField?: "day" | "month",
): TokenQuotaRule[] {
  if (rulesField) {
    let arr: TokenQuotaRule[] = [];
    if (typeof rulesField === "string") {
      try {
        arr = JSON.parse(rulesField);
      } catch {
        arr = [];
      }
    } else {
      arr = rulesField;
    }
    if (arr.length > 0) return arr;
  }
  if (dayField === undefined) return [];
  const mode: TokenQuotaMode = periodField === "month" ? "month" : "day";
  return [{ mode, limit: dayField }];
}

/** 用户实际生效规则解析 */
export function resolveUserEffectiveRule(
  userGroupIds: number[],
  allGroupRules: Map<number, TokenQuotaRule[]>,
  userOwnRule: TokenQuotaRule[] | null,
  siteDefaultRule: TokenQuotaRule[],
): EffectiveRuleResult {
  for (const gid of userGroupIds) {
    const r = allGroupRules.get(gid);
    if (r && r.length > 0) {
      return { rules: r, source: "group", sourceGroupId: gid };
    }
  }
  if (userOwnRule && userOwnRule.length > 0) {
    return { rules: userOwnRule, source: "user" };
  }
  return { rules: siteDefaultRule, source: "default" };
}

/** 计算单条规则当前周期百分比(used / limit * 100),limit=-1 返回 0 */
export function quotaPercent(rule: TokenQuotaRule, used: number): number {
  if (rule.limit <= 0) return 0;
  return Math.min(100, Math.round((used / rule.limit) * 100));
}

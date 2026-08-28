// =============================================================================
// ClawPro 资源库单页 · /design-system/assets（建设计划 §9 阶段 7）
// -----------------------------------------------------------------------------
// 安全边界（务必遵守，勿越界）：
//   - 本页**只消费**阶段 2~6 生成的静态 JSON 数据产物 + 构建期 import.meta.glob
//     的资源 URL 做预览；**不做**在线扫描 / 上传 / 删除 / 编辑 / 重命名任何文件，
//     不写回任何 JSON，不调用任何后端写接口。
//   - 「适合组件槽位」白名单 / 风险约束来自阶段 8 产物
//     client/src/design-assets/manual-overrides/component-resource-map.json（人工策展，
//     基于真实组件源码审计），本页只读展示、不臆测。
//   - 数据事实源：client/src/design-assets/generated/*.generated.json
// =============================================================================

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as LucideIcons from "lucide-react";
import {
  Search,
  X,
  Copy,
  LayoutGrid,
  List,
  ExternalLink,
  Info,
  ShieldCheck,
  Layers,
  FileWarning,
  ChevronRight,
  ChevronDown,
  Boxes,
  ImageOff,
  Download,
  FileCode2,
  ArrowUp,
} from "lucide-react";
import { toast } from "sonner";

import inventoryJson from "@/design-assets/generated/resource-inventory.generated.json";
import usageJson from "@/design-assets/generated/resource-usage.generated.json";
import governanceJson from "@/design-assets/generated/resource-governance.generated.json";
import needsReviewJson from "@/design-assets/generated/resource-needs-review.generated.json";
import componentResourceMapJson from "@/design-assets/manual-overrides/component-resource-map.json";

// -----------------------------------------------------------------------------
// 类型（贴合 generated JSON 真实结构，仅取页面用到的字段）
// -----------------------------------------------------------------------------
interface Classification {
  primaryCategory: string;
  usageScenarios: string[];
  visualType: string | null;
  scenes: string[];
  componentSlot: string | null;
  excludeFromLibrary: boolean;
  tags: string[];
  status: string;
  needsReview: boolean;
  needsReviewReasons: string[];
  duplicateGroupId: string | null;
  canonicalId: string | null;
  canRecolor: boolean | null;
  canUseInSkillMap: boolean | null;
  registry: { registered: boolean };
  duplicateRole: string | null;
  duplicateGrade: string | null;
  isCanonical: boolean;
  canonicalKey: string | null;
  connectedToCanonicalEntry: boolean;
}

interface InvItem {
  id: string;
  displayName: string;
  fileName: string | null;
  type: string;
  source: string;
  scanScope: string;
  sourcePath: string | null;
  sourceDir: string | null;
  definedIn?: string | null;
  definedAtLine?: number | null;
  importName?: string | null;
  webPath: string | null;
  importPath: string | null;
  bytes: number | null;
  usageCount: number;
  usageRefs: number;
  classification: Classification;
  svgMeta?: { viewBox?: string; width?: string; height?: string } | null;
  dynamicDirReferenced?: boolean;
}

interface UsageRef {
  file: string;
  line: number;
  kind: string;
}

interface GovMember {
  id: string;
  path: string;
  role: string;
  result?: string;
  primaryCategory?: string;
  usageCount?: number;
}

interface GovGroup {
  id: string;
  grade: string;
  method: string;
  redLine: boolean;
  state: string;
  canonicalId?: string;
  canonicalPath?: string;
  removedCount?: number;
  skippedCount?: number;
  reasons?: string[];
  reviewStatus?: string;
  members: GovMember[];
}

interface CanonicalEntry {
  key: string;
  group: string;
  id: string;
  path: string;
  sourcePath: string;
  category: string;
  visualType: string;
  status: string;
  duplicateGroupId: string | null;
  redLine: boolean;
  connectedToCanonicalEntry: boolean;
  consumers: string[];
}

interface NeedsReviewItem {
  id: string;
  displayName: string;
  type: string;
  source: string;
  scanScope: string;
  sourcePath?: string | null;
  definedIn?: string | null;
  usageCount: number;
  dynamicDirReferenced: boolean;
  guessPrimaryCategory: string;
  reasons: string[];
}

// -----------------------------------------------------------------------------
// 静态数据（只读 cast，不修改）
// -----------------------------------------------------------------------------
const inventory = inventoryJson as unknown as {
  items: InvItem[];
  classificationSummary: { byPrimaryCategory: Record<string, number>; needsReview: number };
  duplicateSummary: { groups: number; byGrade: Record<string, number> };
  canonicalStage: number;
  canonicalSummary: { entries: number; connectedToEntry: number; availableNotConnected: number };
};
const usageMap = (usageJson as unknown as { usage: Record<string, UsageRef[]> }).usage;
const governance = governanceJson as unknown as {
  summary: { removed: number; bytesReclaimed: number; cNeedsReviewGroups: number };
  removedIds: string[];
  groups: GovGroup[];
  cNeedsReview: GovGroup[];
  canonical: { entries: CanonicalEntry[]; summary: { connectedToEntry: number; availableNotConnected: number } };
};
const needsReview = needsReviewJson as unknown as {
  summary: { total: number; byPrimaryCategory: Record<string, number> };
  items: NeedsReviewItem[];
};

// 构建期收集 src/assets 下资源 URL，用于预览（仅 URL，无文件写入能力）
const srcAssetUrls = import.meta.glob("../assets/**/*", {
  eager: true,
  query: "?url",
  import: "default",
}) as Record<string, string>;

function resolvePreviewUrl(item: { webPath: string | null; importPath: string | null }): string | null {
  if (item.webPath) return item.webPath; // public 资源：直接走 web 路径
  if (item.importPath && item.importPath.startsWith("@/assets/")) {
    const key = item.importPath.replace("@/assets/", "../assets/");
    return srcAssetUrls[key] ?? null;
  }
  return null;
}

// id -> inventory item，用于待确认视图按 id 回查可预览资源（只读）
const invById = new Map<string, InvItem>();
inventory.items.forEach((i) => invById.set(i.id, i));

function resolveReviewPreview(id: string): { item: InvItem; url: string } | null {
  const it = invById.get(id);
  if (!it) return null;
  const url = resolvePreviewUrl(it);
  return url ? { item: it, url } : null;
}

// 浏览器侧下载（纯前端，不写服务器、不改文件；同源资源 a[download] 即可触发）
function triggerDownload(url: string, filename?: string | null) {
  const a = document.createElement("a");
  a.href = url;
  a.download = filename || "";
  a.rel = "noopener";
  document.body.appendChild(a);
  a.click();
  a.remove();
}

// own-svg 尺寸档（基于真实 svgMeta.width）
type SizeBucket = "sm" | "md" | "lg" | "unknown";
function sizeBucketOf(item: InvItem): SizeBucket {
  const w = parseInt(item.svgMeta?.width ?? "", 10);
  if (!w || Number.isNaN(w)) return "unknown";
  if (w <= 18) return "sm";
  if (w <= 36) return "md";
  return "lg";
}

// 使用页面：把静态引用文件路径映射为「端别 · 页面/组件名」可读标签（item 3）
function pageLabelOf(file: string): string {
  const base = file.split("/").pop()?.replace(/\.[^.]+$/, "") ?? file;
  let scene = "";
  if (/\/pages\/admin\/|\/admin\//.test(file)) scene = "管控端";
  else if (/\/pages\/tenant\/|\/tenant\//.test(file)) scene = "用户端";
  else if (/\/pages\/landing\/|LandingPage/.test(file)) scene = "落地页";
  else if (/\/design-system\/|DesignSystem/.test(file)) scene = "设计系统";
  else if (/\/components\//.test(file)) scene = "组件";
  return scene ? `${scene} · ${base}` : base;
}
function pagesFromRefs(refs: UsageRef[]): string[] {
  const set = new Set<string>();
  refs.forEach((r) => set.add(pageLabelOf(r.file)));
  return Array.from(set);
}

// -----------------------------------------------------------------------------
// 文案映射
// -----------------------------------------------------------------------------
const CATEGORY_LABELS: Record<string, string> = {
  "own-svg": "自有 SVG",
  lucide: "系统图标（lucide）",
  "channel-icon": "渠道图标",
  "brand-logo": "品牌 Logo",
  "agent-avatar": "Agent 头像",
  "business-image": "业务图片",
  "inline-svg": "内联 SVG",
  "scan-only-export": "导出扫描件",
};

const SCENARIO_LABELS: Record<string, string> = {
  channel: "渠道",
  admin: "管控端",
  tenant: "用户端",
  agent: "Agent",
  brand: "品牌",
  avatar: "头像",
  "file-space": "文件空间",
  "empty-state": "空状态",
};

const TYPE_LABELS: Record<string, string> = {
  svg: "SVG",
  png: "PNG",
  lucide: "lucide",
  "inline-svg": "内联 SVG",
};

// 视觉类型中文映射（详情面板与筛选口径一致）
const VISUAL_TYPE_LABELS: Record<string, string> = {
  "monochrome-currentColor": "单色线性",
  gradient: "多彩渐变·待细分",
  "gradient-line": "线条多彩渐变",
  "gradient-solid": "块状多彩渐变",
  "illustrative-icon": "插画",
  "brand-fixed": "品牌固定色",
  "avatar-like": "头像",
  "asset-fixed-color": "固定色栅格",
};

const STATUS_META: Record<string, { label: string; cls: string }> = {
  normal: { label: "已确认", cls: "border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]" },
  "needs-review": { label: "待确认", cls: "border-[#FED7AA] bg-[#FFF7ED] text-[#B8640A]" },
};

const GOV_STATE_META: Record<string, { label: string; cls: string }> = {
  "duplicates-removed": { label: "已清理冗余", cls: "border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]" },
  "references-updated": { label: "已替换引用", cls: "border-[#BFDBFE] bg-[#EFF6FF] text-[#1447E6]" },
  "reviewed-confirmed": { label: "已人工确认", cls: "border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]" },
  "keep-multiple": { label: "保留多份", cls: "border-[#DDE7F2] bg-[#F8FAFC] text-[#475569]" },
  "pending-review": { label: "待人工确认", cls: "border-[#FED7AA] bg-[#FFF7ED] text-[#B8640A]" },
};

// own-svg 二级筛选：尺寸档 × 风格（基于真实字段，纯展示层切分，不改底层口径）
const OWN_SIZE_OPTIONS: { value: string; label: string }[] = [
  { value: "all", label: "全部尺寸" },
  { value: "sm", label: "小图标 ≤18" },
  { value: "md", label: "功能图标 24–36" },
  { value: "lg", label: "大图 ≥48" },
];

const OWN_STYLE_OPTIONS: { value: string; label: string }[] = [
  { value: "all", label: "全部风格" },
  { value: "gradient-line", label: "线条多彩渐变" },
  { value: "gradient-solid", label: "块状多彩渐变" },
  { value: "gradient", label: "多彩渐变·待细分" },
  { value: "monochrome-currentColor", label: "单色线性" },
  { value: "illustrative-icon", label: "插画" },
  { value: "__none__", label: "未标注" },
];

// 自有 SVG 排序口径：同风格聚在一起（块状→线条→待细分渐变→单色→插画→未标注）
const OWN_STYLE_SORT: Record<string, number> = {
  "gradient-solid": 0,
  "gradient-line": 1,
  gradient: 2,
  "monochrome-currentColor": 3,
  "illustrative-icon": 4,
  __none__: 7,
};
// 同风格内按尺寸聚类：大图 → 功能图标 → 小图标 → 未知
const OWN_SIZE_SORT: Record<SizeBucket, number> = { lg: 0, md: 1, sm: 2, unknown: 3 };

// 自有 SVG 组件槽位（子菜单）：与 classify-resources.mjs / classification.json 口径一致
const COMPONENT_SLOT_LABELS: Record<string, string> = {
  "admin-sidebar": "AdminSidebar 图标",
  "card-left-icon": "卡片左侧图标",
  "number-card": "NumberCard 卡片图标",
  "file-type": "文件类型图标",
  "run-status": "运行状态图标",
  "feature-card": "企业特性卡片图标",
};
const OWN_SLOT_SUBNAV: { value: string; label: string }[] = [
  { value: "all", label: "全部自有 SVG" },
  { value: "admin-sidebar", label: "AdminSidebar 图标" },
  { value: "card-left-icon", label: "卡片左侧图标" },
  { value: "number-card", label: "NumberCard 卡片图标" },
  { value: "file-type", label: "文件类型图标" },
  { value: "run-status", label: "运行状态图标" },
  { value: "feature-card", label: "企业特性卡片图标" },
];

// 阶段 8 产物：组件槽位白名单 + 风险约束（component-resource-map.json）
interface ComponentSlot {
  label: string;
  componentSlotPath: string;
  owningComponents: string[];
  recommendedResourceType: string;
  allowLucideFallback: boolean;
  riskLevel: string;
  redline: boolean;
  currentResource: string;
  hostInjection: string;
  constraint: string;
}
const COMPONENT_SLOTS = (componentResourceMapJson as { slots: Record<string, ComponentSlot> }).slots;
// 资源 → 槽位解析：优先 classification.componentSlot（自有 SVG 子菜单），
// 否则按 primaryCategory 映射到业务资源槽位（头像 / 渠道 / 品牌）。
const PRIMARY_CATEGORY_TO_SLOT: Record<string, string> = {
  "agent-avatar": "agent-avatar",
  "channel-icon": "channel-icon",
  "brand-logo": "brand-logo",
};
function resolveComponentSlot(cls: { componentSlot: string | null; primaryCategory: string }): ComponentSlot | null {
  const key = cls.componentSlot ?? PRIMARY_CATEGORY_TO_SLOT[cls.primaryCategory];
  return key ? COMPONENT_SLOTS[key] ?? null : null;
}
// 风险等级 = 该槽位资源用法的「跨仓可移植性 / 资源治理难度」，非资源质量问题。
// tip 文案严格对应 component-resource-map.json 的 riskLevel 判定口径。
const RISK_LEVEL_LABELS: Record<string, { label: string; dot: string; tip: string }> = {
  low: {
    label: "低",
    dot: "bg-[#16A34A]",
    tip: "可移植性风险低：组件不依赖当前项目路径，资源由页面 props / 槽位传入或已走 canonical 统一入口，跨仓复用基本无痛。",
  },
  medium: {
    label: "中",
    dot: "bg-[#D97706]",
    tip: "可移植性风险中：资源语义难用单色 lucide 等价，或 inline 散落 / 内嵌路径写死，跨仓需由宿主注入或降级处理。",
  },
  high: {
    label: "高",
    dot: "bg-[#DC2626]",
    tip: "可移植性风险高：组件层硬编码当前项目 /assets 路径，跨仓直接缺资源、不可移植，需单独立项改造。",
  },
};

// 「全部资源」视图分类优先级：自有资源 / 品牌 / 渠道 / 头像 / 业务图在前，lucide 垫后
const ALL_SORT_ORDER: Record<string, number> = {
  "own-svg": 0,
  "brand-logo": 1,
  "channel-icon": 2,
  "agent-avatar": 3,
  "business-image": 4,
  lucide: 9,
};

// 待确认项处置建议（按可预览性与来源类型）
const REVIEW_HINT: Record<string, { label: string; cls: string }> = {
  "scan-only-export": { label: "设计导出件 · 无代码引用 · 建议归档", cls: "border-[#DDE7F2] bg-[#F8FAFC] text-[#64748B]" },
  "inline-svg": { label: "内联于代码 · 阶段 8 判定去向：替换 lucide / 抽 registry / 保留组件私有", cls: "border-[#DDE7F2] bg-[#F8FAFC] text-[#64748B]" },
};

// -----------------------------------------------------------------------------
// 工具函数
// -----------------------------------------------------------------------------
function formatBytes(bytes: number | null): string {
  if (bytes == null) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
}

function shortPath(path: string | null | undefined, max = 48): string {
  if (!path) return "—";
  if (path.length <= max) return path;
  const head = path.slice(0, Math.floor(max * 0.4));
  const tail = path.slice(-Math.floor(max * 0.5));
  return `${head}…${tail}`;
}

// 兜底复制：非安全上下文（http 预览）下 navigator.clipboard 不可用，回退 execCommand
function legacyCopy(text: string): boolean {
  try {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.top = "-9999px";
    ta.setAttribute("readonly", "");
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand("copy");
    ta.remove();
    return ok;
  } catch {
    return false;
  }
}

async function copyText(text: string, label = "已复制") {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      toast.success(label);
      return;
    }
    throw new Error("insecure-context");
  } catch {
    if (legacyCopy(text)) {
      toast.success(label);
    } else {
      toast.error("复制失败，请手动选择文本");
    }
  }
}

function buildCopySnippet(item: InvItem): { lang: string; code: string } {
  const cls = item.classification;
  if (cls.canonicalKey && cls.connectedToCanonicalEntry) {
    return {
      lang: "页面层 · canonical 入口（一改多处生效）",
      code: `import { canonicalAssets } from "@/design-assets/canonical-assets";\n\n<img src={canonicalAssets.${cls.canonicalKey}} alt="${item.displayName}" />`,
    };
  }
  if (cls.canonicalKey) {
    return {
      lang: "页面层 · canonical 入口（已建入口，未全量迁移）",
      code: `import { canonicalAssets } from "@/design-assets/canonical-assets";\n\n// canonicalAssets.${cls.canonicalKey}（当前散落引用保持原样）`,
    };
  }
  if (item.type === "lucide" && item.importName) {
    return {
      lang: "lucide-react（图标库优先）",
      code: `import { ${item.importName} } from "lucide-react";\n\n<${item.importName} className="size-4" />`,
    };
  }
  if (item.webPath) {
    return { lang: "public web 路径", code: `<img src="${item.webPath}" alt="${item.displayName}" />` };
  }
  if (item.importPath) {
    const varName = (item.fileName || "asset").replace(/\.[^.]+$/, "").replace(/[^a-zA-Z0-9]/g, "") || "asset";
    return { lang: "src import（Vite 构建期）", code: `import ${varName} from "${item.importPath}";\n\n<img src={${varName}} alt="${item.displayName}" />` };
  }
  return { lang: "内联 / 待确认", code: "// 该资源为内联 SVG 或待确认项，暂无标准引用方式" };
}

// -----------------------------------------------------------------------------
// 通用展示组件
// -----------------------------------------------------------------------------
function Pill({ children, cls }: { children: React.ReactNode; cls?: string }) {
  return (
    <span
      className={`inline-flex h-5 items-center rounded-[4px] border px-1.5 text-[11px] font-medium leading-none ${
        cls ?? "border-[#DDE7F2] bg-[#F8FAFC] text-[#475569]"
      }`}
    >
      {children}
    </span>
  );
}

function StatCard({ label, value, hint, tone }: { label: string; value: string | number; hint: string; tone?: string }) {
  return (
    <div className="rounded-[4px] border border-[#DDE7F2] bg-white px-4 py-3">
      <div className="whitespace-nowrap text-xs text-[#64748B]">{label}</div>
      <div className={`mt-1 text-2xl font-semibold tabular-nums ${tone ?? "text-[#0A0A0A]"}`}>{value}</div>
      <div className="mt-0.5 whitespace-nowrap text-[11px] text-[#94A3B8]">{hint}</div>
    </div>
  );
}

function LucideGlyph({ name, className }: { name: string | null | undefined; className?: string }) {
  const Comp = name ? (LucideIcons as unknown as Record<string, React.ComponentType<{ className?: string }>>)[name] : undefined;
  if (!Comp) return <ImageOff className={`text-[#CBD5E1] ${className ?? ""}`} />;
  return <Comp className={className} />;
}

function AssetPreview({ item, sizeClass = "size-10", bg }: { item: InvItem; sizeClass?: string; bg?: "light" | "dark" }) {
  const url = resolvePreviewUrl(item);
  const wrap = `flex items-center justify-center ${sizeClass}`;
  if (item.type === "lucide") {
    return (
      <div className={wrap}>
        <LucideGlyph name={item.importName} className={`${sizeClass} ${bg === "dark" ? "text-white" : "text-[#334155]"}`} />
      </div>
    );
  }
  if (url) {
    return (
      <div className={wrap}>
        <img src={url} alt={item.displayName} className="max-h-full max-w-full object-contain" loading="lazy" />
      </div>
    );
  }
  return (
    <div className={`${wrap} rounded-[4px] border border-dashed border-[#DDE7F2] text-[10px] text-[#94A3B8]`}>
      {TYPE_LABELS[item.type] ?? item.type}
    </div>
  );
}

// -----------------------------------------------------------------------------
// 主组件
// -----------------------------------------------------------------------------
type BrowseCategory = "all" | "own-svg" | "lucide" | "channel-icon" | "brand-logo" | "agent-avatar" | "business-image";
type ViewKey = BrowseCategory | "duplicates" | "canonical" | "review";

const BROWSE_CATEGORIES: { key: BrowseCategory; label: string; muted?: boolean }[] = [
  { key: "own-svg", label: "自有 SVG" },
  { key: "channel-icon", label: "渠道图标" },
  { key: "brand-logo", label: "品牌 Logo" },
  { key: "agent-avatar", label: "Agent 头像" },
  { key: "business-image", label: "业务图片" },
  { key: "lucide", label: "系统图标（lucide）", muted: true },
  { key: "all", label: "全部资源", muted: true },
];

const PAGE_SIZE = 48;
const removedSet = new Set(governance.removedIds);
// 待确认中可预览的数量（own-svg / 业务图，有文件路径）
const previewableReviewCount = needsReview.items.filter((r) => resolveReviewPreview(r.id)).length;

export default function DesignSystemAssets() {
  const [view, setView] = useState<ViewKey>("own-svg");
  const [keyword, setKeyword] = useState("");
  const [sceneFilter, setSceneFilter] = useState("all");
  const [canonicalFilter, setCanonicalFilter] = useState("all");
  const [sizeFilter, setSizeFilter] = useState("all");
  const [styleFilter, setStyleFilter] = useState("all");
  const [slotFilter, setSlotFilter] = useState("all");
  const [layout, setLayout] = useState<"grid" | "list">("grid");
  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  // 浏览池：未删除 + include 范围 + 排除内联 SVG 与导出扫描件（后两者只进治理/待确认视图）
  //   + 排除「属组件私有/不纳入资源库」的资源（excludeFromLibrary，如 Empty 组件插画）
  const displayItems = useMemo(
    () =>
      inventory.items.filter(
        (i) =>
          !removedSet.has(i.id) &&
          i.scanScope === "include" &&
          i.type !== "inline-svg" &&
          i.classification.primaryCategory !== "scan-only-export" &&
          !i.classification.excludeFromLibrary,
      ),
    [],
  );

  const countByCategory = useMemo(() => {
    const map: Record<string, number> = { all: displayItems.length };
    for (const i of displayItems) {
      const c = i.classification.primaryCategory;
      map[c] = (map[c] ?? 0) + 1;
    }
    return map;
  }, [displayItems]);

  // 浏览池按资源类型拆解（用于「资源库总量」卡片 hint）
  const typeCounts = useMemo(() => {
    const map = { svg: 0, png: 0, lucide: 0 };
    for (const i of displayItems) {
      if (i.type === "svg") map.svg += 1;
      else if (i.type === "png") map.png += 1;
      else if (i.type === "lucide") map.lucide += 1;
    }
    return map;
  }, [displayItems]);

  // 自有 SVG 视图候选：own-svg + 任意带组件槽位的资源（如 NumberCard 用到的栅格图标）
  const ownSvgPool = useMemo(
    () => displayItems.filter((i) => i.classification.primaryCategory === "own-svg" || i.classification.componentSlot),
    [displayItems],
  );
  const slotCounts = useMemo(() => {
    const map: Record<string, number> = { all: ownSvgPool.length };
    for (const i of ownSvgPool) {
      const s = i.classification.componentSlot;
      if (s) map[s] = (map[s] ?? 0) + 1;
    }
    return map;
  }, [ownSvgPool]);

  const isBrowse = view !== "duplicates" && view !== "canonical" && view !== "review";
  const isOwnSvg = view === "own-svg";

  const gridItems = useMemo(() => {
    if (!isBrowse) return [];
    const kw = keyword.trim().toLowerCase();
    const pool = isOwnSvg ? ownSvgPool : displayItems;
    const filtered = pool.filter((i) => {
      const cls = i.classification;
      if (isOwnSvg) {
        if (slotFilter !== "all" && cls.componentSlot !== slotFilter) return false;
      } else if (view !== "all" && cls.primaryCategory !== view) return false;
      if (sceneFilter !== "all" && !cls.scenes.includes(sceneFilter)) return false;
      if (canonicalFilter === "canonical" && !cls.isCanonical) return false;
      if (canonicalFilter === "connected" && !cls.connectedToCanonicalEntry) return false;
      if (isOwnSvg && sizeFilter !== "all" && sizeBucketOf(i) !== sizeFilter) return false;
      if (isOwnSvg && styleFilter !== "all") {
        const vt = i.classification.visualType;
        if (styleFilter === "__none__" ? vt != null : vt !== styleFilter) return false;
      }
      if (kw) {
        const hay = [i.displayName, i.fileName, i.sourcePath, i.importName, cls.primaryCategory, ...cls.tags, ...cls.usageScenarios]
          .filter(Boolean)
          .join(" ")
          .toLowerCase();
        if (!hay.includes(kw)) return false;
      }
      return true;
    });
    // 「全部资源」按分类优先级排序：自有/品牌/渠道/头像/业务图在前，lucide 垫后
    if (view === "all") {
      return filtered
        .map((i, idx) => ({ i, idx }))
        .sort((a, b) => {
          const pa = ALL_SORT_ORDER[a.i.classification.primaryCategory] ?? 5;
          const pb = ALL_SORT_ORDER[b.i.classification.primaryCategory] ?? 5;
          return pa - pb || a.idx - b.idx;
        })
        .map((x) => x.i);
    }
    // 自有 SVG：同风格、相近尺寸聚在一起（风格 → 尺寸 → 名称），避免穿插显乱
    if (isOwnSvg) {
      return filtered
        .map((i, idx) => ({ i, idx }))
        .sort((a, b) => {
          const sa = OWN_STYLE_SORT[a.i.classification.visualType ?? "__none__"] ?? 8;
          const sb = OWN_STYLE_SORT[b.i.classification.visualType ?? "__none__"] ?? 8;
          if (sa !== sb) return sa - sb;
          const za = OWN_SIZE_SORT[sizeBucketOf(a.i)];
          const zb = OWN_SIZE_SORT[sizeBucketOf(b.i)];
          if (za !== zb) return za - zb;
          return a.i.displayName.localeCompare(b.i.displayName) || a.idx - b.idx;
        })
        .map((x) => x.i);
    }
    return filtered;
  }, [isBrowse, isOwnSvg, view, keyword, sceneFilter, canonicalFilter, sizeFilter, styleFilter, slotFilter, ownSvgPool, displayItems]);

  const visibleItems = gridItems.slice(0, visibleCount);
  const hasMore = visibleCount < gridItems.length;

  const selected = selectedId ? inventory.items.find((i) => i.id === selectedId) ?? null : null;

  // 滚动到底自动加载更多（IntersectionObserver 分批渲染）
  useEffect(() => {
    if (!isBrowse || !hasMore) return;
    const el = sentinelRef.current;
    if (!el) return;
    const ob = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          setVisibleCount((c) => Math.min(c + PAGE_SIZE, gridItems.length));
        }
      },
      { rootMargin: "600px 0px" },
    );
    ob.observe(el);
    return () => ob.disconnect();
  }, [isBrowse, hasMore, visibleCount, gridItems.length]);

  const resetList = useCallback(() => setVisibleCount(PAGE_SIZE), []);
  const switchView = (next: ViewKey) => {
    setView(next);
    if (next !== "own-svg") {
      setSizeFilter("all");
      setStyleFilter("all");
      setSlotFilter("all");
    }
    resetList();
  };

  return (
    <div className="page-enter min-h-screen bg-[#F4F8FC] text-[#0A0A0A]">
      <header className="border-b border-[#DDE7F2] bg-[linear-gradient(180deg,#FFFFFF_0%,#F7FAFF_100%)]">
        <div className="mx-auto max-w-[1680px] px-8 py-7">
          <div className="flex flex-wrap items-start justify-between gap-8">
            <div className="min-w-0">
              <div className="mb-3 flex flex-wrap items-center gap-2">
                <Pill cls="border-[#DDE7F2] bg-white/75 text-[#334155]">内部设计资产</Pill>
                <Pill cls="border-[#DDE7F2] bg-white/75 text-[#334155]">数据来源：client/src/design-assets/generated</Pill>
                <Pill cls="border-[#DDE7F2] bg-white/75 text-[#334155]">只读 · 不扫描/上传/删除/编辑文件</Pill>
              </div>
              <h1 className="text-2xl font-semibold tracking-tight text-[#0A0A0A]">ClawPro 资源库</h1>
              <p className="mt-2 max-w-3xl text-sm text-[#475569]">
                浏览、筛选与查看图标、SVG、图片等资源的分类、使用位置、canonical 入口接入状态与重复治理结果。页面仅消费阶段
                2~6 生成的静态数据产物。
              </p>
            </div>
            <div className="grid w-[640px] max-w-full shrink-0 grid-cols-3 gap-3">
              <StatCard label="自有 SVG" value={countByCategory["own-svg"] ?? 0} hint="个（渠道/品牌/业务核心）" tone="text-[#1447E6]" />
              <StatCard label="可复用 canonical" value={inventory.canonicalSummary.entries} hint={`${inventory.canonicalSummary.connectedToEntry} 个已接入入口`} />
              <StatCard label="资源库总量" value={countByCategory["all"] ?? 0} hint={`SVG ${typeCounts.svg} · 图片 ${typeCounts.png} · 系统图标 ${typeCounts.lucide}`} />
            </div>
          </div>
        </div>
      </header>

      <div className="mx-auto flex max-w-[1680px] gap-6 px-8 py-6">
        {/* 左侧分类导航 */}
        <aside className="w-[240px] shrink-0">
          <nav className="sticky top-6 space-y-5">
            <div>
              <NavGroupLabel>浏览资源</NavGroupLabel>
              <div className="space-y-1">
                {BROWSE_CATEGORIES.map((c, idx) => (
                  <div key={c.key}>
                    {c.muted && !BROWSE_CATEGORIES[idx - 1]?.muted && (
                      <div className="my-1.5 border-t border-dashed border-[#E2E8F0]" />
                    )}
                    <NavButton
                      active={view === c.key}
                      label={c.label}
                      count={countByCategory[c.key] ?? 0}
                      muted={c.muted}
                      onClick={() => switchView(c.key)}
                    />
                    {c.key === "own-svg" && view === "own-svg" && (
                      <div className="mb-1 ml-3 mt-1 space-y-0.5 border-l border-[#E2E8F0] pl-2">
                        {OWN_SLOT_SUBNAV.map((s) => (
                          <SubNavButton
                            key={s.value}
                            active={slotFilter === s.value}
                            label={s.label}
                            count={slotCounts[s.value] ?? 0}
                            onClick={() => {
                              setSlotFilter(s.value);
                              resetList();
                            }}
                          />
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
            <div>
              <NavGroupLabel>治理与入口</NavGroupLabel>
              <div className="space-y-1">
                <NavButton
                  active={view === "duplicates"}
                  label="重复治理"
                  count={inventory.duplicateSummary.groups}
                  icon={<Layers className="size-3.5" />}
                  onClick={() => switchView("duplicates")}
                />
                <NavButton
                  active={view === "canonical"}
                  label="canonical 入口"
                  count={governance.canonical.entries.length}
                  icon={<ShieldCheck className="size-3.5" />}
                  onClick={() => switchView("canonical")}
                />
                <NavButton
                  active={view === "review"}
                  label="内联 SVG 待治理"
                  count={needsReview.summary.total}
                  icon={<FileWarning className="size-3.5" />}
                  onClick={() => switchView("review")}
                />
              </div>
            </div>
          </nav>
        </aside>

        {/* 主体 */}
        <main className="min-w-0 flex-1">
          {isBrowse ? (
            <>
              <FilterBar
                keyword={keyword}
                onKeyword={(v) => {
                  setKeyword(v);
                  resetList();
                }}
                sceneFilter={sceneFilter}
                onScene={(v) => {
                  setSceneFilter(v);
                  resetList();
                }}
                canonicalFilter={canonicalFilter}
                onCanonical={(v) => {
                  setCanonicalFilter(v);
                  resetList();
                }}
                showOwnFilters={isOwnSvg}
                sizeFilter={sizeFilter}
                onSize={(v) => {
                  setSizeFilter(v);
                  resetList();
                }}
                styleFilter={styleFilter}
                onStyle={(v) => {
                  setStyleFilter(v);
                  resetList();
                }}
                layout={layout}
                onLayout={setLayout}
                resultCount={gridItems.length}
              />
              {gridItems.length === 0 ? (
                <EmptyHint title="没有匹配的资源" desc="调整搜索词或筛选条件后再试。" />
              ) : layout === "grid" ? (
                <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
                  {visibleItems.map((item) => (
                    <ResourceCard key={item.id} item={item} active={item.id === selectedId} onClick={() => setSelectedId(item.id)} showCategory={view === "all"} />
                  ))}
                </div>
              ) : (
                <ResourceTable items={visibleItems} selectedId={selectedId} onSelect={setSelectedId} />
              )}
              {gridItems.length > 0 && (
                <div className="mt-5">
                  {hasMore ? (
                    <div ref={sentinelRef} className="flex items-center justify-center py-4 text-xs text-[#94A3B8]">
                      <span className="tabular-nums">
                        已加载 {visibleItems.length} / {gridItems.length} 个 · 继续滚动加载更多
                      </span>
                    </div>
                  ) : (
                    gridItems.length > PAGE_SIZE && (
                      <div className="py-4 text-center text-xs text-[#94A3B8]">已显示全部 {gridItems.length} 个资源</div>
                    )
                  )}
                </div>
              )}
            </>
          ) : view === "duplicates" ? (
            <DuplicatesView onOpen={setSelectedId} />
          ) : view === "canonical" ? (
            <CanonicalView onOpen={setSelectedId} />
          ) : (
            <NeedsReviewView onOpen={setSelectedId} />
          )}
        </main>
      </div>

      {selected && <DetailPanel item={selected} onClose={() => setSelectedId(null)} />}
      <BackToTop />
    </div>
  );
}

function BackToTop() {
  const [show, setShow] = useState(false);
  useEffect(() => {
    const onScroll = () => setShow(window.scrollY > 600);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);
  if (!show) return null;
  return (
    <button
      type="button"
      title="回到顶部"
      onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })}
      className="fixed bottom-6 right-6 z-30 flex size-10 items-center justify-center rounded-full border border-[#DDE7F2] bg-white text-[#475569] shadow-md transition-colors hover:border-[#1447E6] hover:text-[#1447E6]"
    >
      <ArrowUp className="size-4" />
    </button>
  );
}

// -----------------------------------------------------------------------------
// 导航 / 筛选 / 分页 / 空态
// -----------------------------------------------------------------------------
function NavGroupLabel({ children }: { children: React.ReactNode }) {
  return <div className="mb-2 px-2 text-[11px] font-semibold uppercase tracking-wide text-[#94A3B8]">{children}</div>;
}

function NavButton({
  active,
  label,
  count,
  icon,
  muted,
  onClick,
}: {
  active: boolean;
  label: string;
  count: number;
  icon?: React.ReactNode;
  muted?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex w-full items-center justify-between gap-2 rounded-[4px] border px-2.5 py-2 text-left text-sm transition-colors ${
        active
          ? "border-[#BFCFFE] bg-[#F0F3FC] font-medium text-[#1447E6]"
          : muted
            ? "border-transparent text-[#94A3B8] hover:border-[#DDE7F2] hover:bg-white"
            : "border-transparent text-[#334155] hover:border-[#DDE7F2] hover:bg-white"
      }`}
    >
      <span className="flex min-w-0 items-center gap-2">
        {icon}
        <span className="truncate">{label}</span>
      </span>
      <span className={`tabular-nums text-xs ${active ? "text-[#1447E6]" : "text-[#94A3B8]"}`}>{count}</span>
    </button>
  );
}

function SubNavButton({ active, label, count, onClick }: { active: boolean; label: string; count: number; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex w-full items-center justify-between gap-2 rounded-[4px] px-2 py-1.5 text-left text-[13px] transition-colors ${
        active ? "bg-[#F0F3FC] font-medium text-[#1447E6]" : "text-[#64748B] hover:bg-white hover:text-[#334155]"
      }`}
    >
      <span className="truncate">{label}</span>
      <span className={`tabular-nums text-[11px] ${active ? "text-[#1447E6]" : "text-[#94A3B8]"}`}>{count}</span>
    </button>
  );
}

function FilterSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (v: string) => void;
}) {
  return (
    <label className="flex items-center gap-1.5 text-xs text-[#64748B]">
      <span>{label}</span>
      <div className="relative">
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="h-8 appearance-none rounded-[4px] border border-[#DDE7F2] bg-white pl-2.5 pr-7 text-xs text-[#334155] outline-none focus:border-[#1447E6]"
        >
          {options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 size-3.5 -translate-y-1/2 text-[#94A3B8]" />
      </div>
    </label>
  );
}

function FilterBar({
  keyword,
  onKeyword,
  sceneFilter,
  onScene,
  canonicalFilter,
  onCanonical,
  showOwnFilters,
  sizeFilter,
  onSize,
  styleFilter,
  onStyle,
  layout,
  onLayout,
  resultCount,
}: {
  keyword: string;
  onKeyword: (v: string) => void;
  sceneFilter: string;
  onScene: (v: string) => void;
  canonicalFilter: string;
  onCanonical: (v: string) => void;
  showOwnFilters: boolean;
  sizeFilter: string;
  onSize: (v: string) => void;
  styleFilter: string;
  onStyle: (v: string) => void;
  layout: "grid" | "list";
  onLayout: (v: "grid" | "list") => void;
  resultCount: number;
}) {
  return (
    <div className="flex flex-wrap items-center gap-3 rounded-[4px] border border-[#DDE7F2] bg-white px-3 py-2.5">
      <div className="relative min-w-[220px] flex-1">
        <Search className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-[#94A3B8]" />
        <input
          value={keyword}
          onChange={(e) => onKeyword(e.target.value)}
          placeholder="搜索名称 / 文件名 / 路径 / 标签…"
          className="h-8 w-full rounded-[4px] border border-[#DDE7F2] bg-white pl-8 pr-7 text-sm text-[#334155] outline-none focus:border-[#1447E6]"
        />
        {keyword && (
          <button type="button" onClick={() => onKeyword("")} className="absolute right-2 top-1/2 -translate-y-1/2 text-[#94A3B8] hover:text-[#334155]">
            <X className="size-3.5" />
          </button>
        )}
      </div>
      {showOwnFilters && (
        <>
          <FilterSelect label="尺寸" value={sizeFilter} onChange={onSize} options={OWN_SIZE_OPTIONS} />
          <FilterSelect label="风格" value={styleFilter} onChange={onStyle} options={OWN_STYLE_OPTIONS} />
        </>
      )}
      <FilterSelect
        label="场景"
        value={sceneFilter}
        onChange={onScene}
        options={[
          { value: "all", label: "全部" },
          { value: "admin", label: "管控端" },
          { value: "tenant", label: "用户端" },
        ]}
      />
      <FilterSelect
        label="入口"
        value={canonicalFilter}
        onChange={onCanonical}
        options={[
          { value: "all", label: "全部" },
          { value: "canonical", label: "canonical 资源" },
          { value: "connected", label: "已接入入口" },
        ]}
      />
      <span className="text-xs tabular-nums text-[#94A3B8]">{resultCount} 个结果</span>
      <div className="flex items-center gap-1 rounded-[4px] border border-[#DDE7F2] p-0.5">
        <IconToggle active={layout === "grid"} onClick={() => onLayout("grid")} title="网格">
          <LayoutGrid className="size-4" />
        </IconToggle>
        <IconToggle active={layout === "list"} onClick={() => onLayout("list")} title="列表">
          <List className="size-4" />
        </IconToggle>
      </div>
    </div>
  );
}

function IconToggle({ active, onClick, title, children }: { active: boolean; onClick: () => void; title: string; children: React.ReactNode }) {
  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      className={`flex size-7 items-center justify-center rounded-[3px] transition-colors ${
        active ? "bg-[#1447E6] text-white" : "text-[#64748B] hover:bg-[#F1F5F9]"
      }`}
    >
      {children}
    </button>
  );
}

function Pager({ page, pageCount, total, onPage }: { page: number; pageCount: number; total: number; onPage: (p: number) => void }) {
  return (
    <div className="mt-4 flex items-center justify-between text-xs text-[#64748B]">
      <span className="tabular-nums">
        第 {page + 1} / {pageCount} 页 · 共 {total} 个
      </span>
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          disabled={page <= 0}
          onClick={() => onPage(page - 1)}
          className="h-8 rounded-[4px] border border-[#DDE7F2] bg-white px-3 text-[#334155] disabled:opacity-40 hover:enabled:border-[#1447E6]"
        >
          上一页
        </button>
        <button
          type="button"
          disabled={page >= pageCount - 1}
          onClick={() => onPage(page + 1)}
          className="h-8 rounded-[4px] border border-[#DDE7F2] bg-white px-3 text-[#334155] disabled:opacity-40 hover:enabled:border-[#1447E6]"
        >
          下一页
        </button>
      </div>
    </div>
  );
}

function EmptyHint({ title, desc }: { title: string; desc: string }) {
  return (
    <div className="mt-4 flex flex-col items-center justify-center rounded-[4px] border border-dashed border-[#DDE7F2] bg-white py-16 text-center">
      <Boxes className="size-8 text-[#CBD5E1]" />
      <p className="mt-3 text-sm font-medium text-[#334155]">{title}</p>
      <p className="mt-1 text-xs text-[#94A3B8]">{desc}</p>
    </div>
  );
}

// -----------------------------------------------------------------------------
// 资源卡片 / 列表
// -----------------------------------------------------------------------------
function ItemBadges({ item }: { item: InvItem }) {
  const cls = item.classification;
  const status = STATUS_META[cls.status];
  const isOwn = cls.primaryCategory === "own-svg";
  const scenes = cls.scenes ?? [];
  return (
    <div className="flex flex-wrap items-center gap-1">
      {/* 自有 SVG：格式已在卡片副标题体现，徽标改展示使用场景（管控端/用户端等）；其他分类保留类型徽标 */}
      {isOwn
        ? scenes.map((s) => (
            <Pill key={s} cls="border-[#DDE7F2] bg-[#F8FAFC] text-[#475569]">
              {SCENARIO_LABELS[s] ?? s}
            </Pill>
          ))
        : <Pill>{TYPE_LABELS[item.type] ?? item.type}</Pill>}
      {status && <Pill cls={status.cls}>{status.label}</Pill>}
      {cls.isCanonical && <Pill cls="border-[#BFDBFE] bg-[#EFF6FF] text-[#1447E6]">canonical</Pill>}
      {cls.connectedToCanonicalEntry && <Pill cls="border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]">已接入</Pill>}
      {cls.duplicateGroupId && <Pill cls="border-[#E9D5FF] bg-[#FAF5FF] text-[#7E22CE]">{cls.duplicateGroupId}</Pill>}
    </div>
  );
}

function CardActBtn({ title, onClick, children }: { title: string; onClick: (e: React.MouseEvent) => void; children: React.ReactNode }) {
  return (
    <button
      type="button"
      title={title}
      onClick={(e) => {
        e.stopPropagation();
        onClick(e);
      }}
      className="flex size-6 items-center justify-center rounded-[4px] border border-[#DDE7F2] bg-white/95 text-[#475569] shadow-sm transition-colors hover:border-[#1447E6] hover:text-[#1447E6]"
    >
      {children}
    </button>
  );
}

function ResourceCard({ item, active, onClick, showCategory }: { item: InvItem; active: boolean; onClick: () => void; showCategory?: boolean }) {
  const url = resolvePreviewUrl(item);
  const snippet = buildCopySnippet(item);
  const width = item.svgMeta?.width;
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onClick();
        }
      }}
      className={`group relative flex cursor-pointer flex-col rounded-[4px] border bg-white p-3 text-left transition-colors ${
        active ? "border-[#1447E6] ring-1 ring-[#1447E6]/20" : "border-[#DDE7F2] hover:border-[#BFCFFE]"
      }`}
    >
      <div className="relative flex h-20 items-center justify-center rounded-[4px] bg-[#F8FAFC]">
        <AssetPreview item={item} sizeClass="size-12" />
        {width && (
          <span className="absolute left-2.5 top-2.5 rounded-[3px] bg-white/85 px-1 text-[10px] font-medium tabular-nums text-[#64748B]">
            {width}px
          </span>
        )}
        {/* hover 浮出快捷操作 */}
        <div className="absolute right-1.5 top-1.5 flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
          <CardActBtn title="复制名称" onClick={() => copyText(item.displayName, "名称已复制")}>
            <Copy className="size-3" />
          </CardActBtn>
          <CardActBtn title="复制引用代码" onClick={() => copyText(snippet.code, "引用已复制")}>
            <FileCode2 className="size-3" />
          </CardActBtn>
          {url && (
            <>
              <CardActBtn title="下载文件" onClick={() => triggerDownload(url, item.fileName)}>
                <Download className="size-3" />
              </CardActBtn>
              <a
                title="新标签打开"
                href={url}
                target="_blank"
                rel="noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="flex size-6 items-center justify-center rounded-[4px] border border-[#DDE7F2] bg-white/95 text-[#475569] shadow-sm transition-colors hover:border-[#1447E6] hover:text-[#1447E6]"
              >
                <ExternalLink className="size-3" />
              </a>
            </>
          )}
        </div>
      </div>
      <div className="mt-2.5 truncate text-sm font-medium text-[#0A0A0A]" title={item.displayName}>
        {item.displayName}
      </div>
      {showCategory && (
        <div className="mt-0.5 truncate text-[11px] text-[#94A3B8]">{CATEGORY_LABELS[item.classification.primaryCategory] ?? item.classification.primaryCategory}</div>
      )}
      <div className="mt-2">
        <ItemBadges item={item} />
      </div>
    </div>
  );
}

function ResourceTable({ items, selectedId, onSelect }: { items: InvItem[]; selectedId: string | null; onSelect: (id: string) => void }) {
  return (
    <div className="mt-4 overflow-hidden rounded-[4px] border border-[#DDE7F2] bg-white">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-[#EAF1F8] bg-[#F8FAFC] text-left text-xs text-[#64748B]">
            <th className="px-3 py-2 font-medium">资源</th>
            <th className="px-3 py-2 font-medium">分类</th>
            <th className="px-3 py-2 font-medium">类型</th>
            <th className="px-3 py-2 font-medium">使用</th>
            <th className="px-3 py-2 font-medium">大小</th>
            <th className="px-3 py-2 font-medium">标记</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr
              key={item.id}
              onClick={() => onSelect(item.id)}
              className={`cursor-pointer border-b border-[#F1F5F9] transition-colors last:border-0 ${
                item.id === selectedId ? "bg-[#F0F3FC]" : "hover:bg-[#F8FAFC]"
              }`}
            >
              <td className="px-3 py-2">
                <div className="flex items-center gap-2.5">
                  <div className="flex size-8 items-center justify-center rounded-[4px] bg-[#F8FAFC]">
                    <AssetPreview item={item} sizeClass="size-5" />
                  </div>
                  <span className="truncate font-medium text-[#0A0A0A]" title={item.displayName}>
                    {item.displayName}
                  </span>
                </div>
              </td>
              <td className="px-3 py-2 text-[#475569]">{CATEGORY_LABELS[item.classification.primaryCategory] ?? item.classification.primaryCategory}</td>
              <td className="px-3 py-2 text-[#475569]">{TYPE_LABELS[item.type] ?? item.type}</td>
              <td className="px-3 py-2 tabular-nums text-[#475569]">{item.usageCount}</td>
              <td className="px-3 py-2 tabular-nums text-[#475569]">{formatBytes(item.bytes)}</td>
              <td className="px-3 py-2">
                <ItemBadges item={item} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// -----------------------------------------------------------------------------
// 重复治理视图
// -----------------------------------------------------------------------------
const GOV_STATE_ORDER = ["duplicates-removed", "references-updated", "reviewed-confirmed", "keep-multiple", "pending-review"];

function DuplicatesView({ onOpen }: { onOpen: (id: string) => void }) {
  const [stateFilter, setStateFilter] = useState("all");
  const groups = governance.groups;
  const countByState = useMemo(() => {
    const map: Record<string, number> = {};
    groups.forEach((g) => (map[g.state] = (map[g.state] ?? 0) + 1));
    return map;
  }, [groups]);
  const visible = stateFilter === "all" ? groups : groups.filter((g) => g.state === stateFilter);

  return (
    <div>
      <div className="rounded-[4px] border border-[#DDE7F2] bg-white p-4">
        <div className="flex items-center gap-2 text-sm font-medium text-[#0A0A0A]">
          <Layers className="size-4 text-[#1447E6]" /> 重复治理结果
        </div>
        <p className="mt-1.5 text-xs leading-relaxed text-[#64748B]">
          阶段 4 识别 {inventory.duplicateSummary.groups} 组重复（A {inventory.duplicateSummary.byGrade.A} / C{" "}
          {inventory.duplicateSummary.byGrade.C}）；阶段 5 实际清理 {governance.summary.removed} 个冗余副本（保留 canonical，回收{" "}
          {formatBytes(governance.summary.bytesReclaimed)}）；红线（品牌 / 渠道 / 头像）与 registry 事实源副本不自动归并，标待人工确认。
        </p>
        <div className="mt-3 flex flex-wrap gap-1.5">
          <SegBtn active={stateFilter === "all"} onClick={() => setStateFilter("all")} label="全部" count={groups.length} />
          {GOV_STATE_ORDER.filter((s) => countByState[s]).map((s) => (
            <SegBtn key={s} active={stateFilter === s} onClick={() => setStateFilter(s)} label={GOV_STATE_META[s]?.label ?? s} count={countByState[s]} />
          ))}
        </div>
      </div>

      <div className="mt-4 space-y-3">
        {visible.map((g) => (
          <GovGroupCard key={g.id} group={g} onOpen={onOpen} />
        ))}
      </div>
    </div>
  );
}

function SegBtn({ active, onClick, label, count }: { active: boolean; onClick: () => void; label: string; count: number }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex h-7 items-center gap-1.5 rounded-[4px] border px-2.5 text-xs transition-colors ${
        active ? "border-[#1447E6] bg-[#1447E6] text-white" : "border-[#DDE7F2] bg-white text-[#475569] hover:border-[#BFCFFE]"
      }`}
    >
      {label}
      <span className={`tabular-nums ${active ? "text-white/80" : "text-[#94A3B8]"}`}>{count}</span>
    </button>
  );
}

function GovGroupCard({ group, onOpen }: { group: GovGroup; onOpen: (id: string) => void }) {
  const meta = GOV_STATE_META[group.state];
  const [expanded, setExpanded] = useState(false);
  const COLLAPSE_LIMIT = 6;
  const collapsible = group.members.length > COLLAPSE_LIMIT;
  const shownMembers = collapsible && !expanded ? group.members.slice(0, COLLAPSE_LIMIT) : group.members;
  return (
    <div className="rounded-[4px] border border-[#DDE7F2] bg-white p-4">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-medium text-[#0A0A0A]">{group.id}</span>
        <Pill cls={group.grade === "A" ? "border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]" : "border-[#FED7AA] bg-[#FFF7ED] text-[#B8640A]"}>
          {group.grade} 类
        </Pill>
        {meta && <Pill cls={meta.cls}>{meta.label}</Pill>}
        {group.redLine && <Pill cls="border-[#FECACA] bg-[#FEF2F2] text-[#DC2626]">红线</Pill>}
        <span className="text-[11px] text-[#94A3B8]">{group.method}</span>
        <span className="text-[11px] text-[#94A3B8]">共 {group.members.length} 个</span>
        {typeof group.removedCount === "number" && group.removedCount > 0 && (
          <span className="text-[11px] text-[#94A3B8]">已删 {group.removedCount}</span>
        )}
      </div>
      {group.canonicalPath && (
        <div className="mt-2 text-xs text-[#475569]">
          canonical：<code className="rounded-[3px] bg-[#F1F5F9] px-1 py-0.5 text-[#1447E6]">{group.canonicalPath}</code>
        </div>
      )}
      {group.reasons?.length ? <p className="mt-1.5 text-xs leading-relaxed text-[#B8640A]">{group.reasons.join("；")}</p> : null}
      <div className="mt-2.5 overflow-hidden rounded-[4px] border border-[#EAF1F8]">
        {shownMembers.map((m, idx) => (
          <button
            key={m.id}
            type="button"
            onClick={() => onOpen(m.id)}
            className={`flex w-full items-center justify-between gap-2 px-2.5 py-1.5 text-left text-xs transition-colors hover:bg-[#F8FAFC] ${
              idx > 0 ? "border-t border-[#F1F5F9]" : ""
            }`}
          >
            <span className="truncate text-[#475569]" title={m.path}>
              {shortPath(m.path, 56)}
            </span>
            <span className="flex shrink-0 items-center gap-1.5">
              {m.role && <Pill>{m.role}</Pill>}
              {m.result && (
                <Pill cls={m.result === "removed" ? "border-[#FECACA] bg-[#FEF2F2] text-[#DC2626]" : "border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]"}>
                  {m.result === "removed" ? "已删除" : m.result === "kept" ? "保留" : m.result}
                </Pill>
              )}
            </span>
          </button>
        ))}
      </div>
      {collapsible && (
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          className="mt-2 flex items-center gap-1 text-xs font-medium text-[#1447E6] hover:underline"
        >
          <ChevronDown className={`size-3.5 transition-transform ${expanded ? "rotate-180" : ""}`} />
          {expanded ? "收起" : `展开剩余 ${group.members.length - COLLAPSE_LIMIT} 个`}
        </button>
      )}
    </div>
  );
}

// -----------------------------------------------------------------------------
// canonical 入口视图
// -----------------------------------------------------------------------------
const CANONICAL_GROUP_LABELS: Record<string, string> = { brands: "品牌 Logo", channels: "渠道图标", avatars: "Agent 头像" };

function CanonicalView({ onOpen }: { onOpen: (id: string) => void }) {
  const entries = governance.canonical.entries;
  const groups = useMemo(() => {
    const order = ["brands", "channels", "avatars"];
    return order
      .map((g) => ({ group: g, items: entries.filter((e) => e.group === g) }))
      .filter((x) => x.items.length > 0);
  }, [entries]);

  return (
    <div>
      <div className="rounded-[4px] border border-[#DDE7F2] bg-white p-4">
        <div className="flex items-center gap-2 text-sm font-medium text-[#0A0A0A]">
          <ShieldCheck className="size-4 text-[#1447E6]" /> canonical 统一入口（client/src/design-assets/canonical-assets.ts）
        </div>
        <p className="mt-1.5 text-xs leading-relaxed text-[#64748B]">
          共 {inventory.canonicalSummary.entries} 个 key，{inventory.canonicalSummary.connectedToEntry} 个已接入页面层统一入口（修改入口路径
          一改多处生效），{inventory.canonicalSummary.availableNotConnected} 个仅建入口、现有散落引用保持原样（不做全量迁移）。红线资源分列不归并、不当普通
          UI 图标改色。
        </p>
      </div>

      <div className="mt-4 space-y-4">
        {groups.map(({ group, items }) => (
          <div key={group}>
            <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-[#94A3B8]">
              {CANONICAL_GROUP_LABELS[group] ?? group} · {items.length}
            </div>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {items.map((e) => (
                <button
                  key={e.key}
                  type="button"
                  onClick={() => onOpen(e.id)}
                  className="flex items-start gap-3 rounded-[4px] border border-[#DDE7F2] bg-white p-3 text-left transition-colors hover:border-[#BFCFFE]"
                >
                  <div className="flex size-10 shrink-0 items-center justify-center rounded-[4px] bg-[#F8FAFC]">
                    <img src={e.path} alt={e.key} className="max-h-7 max-w-7 object-contain" loading="lazy" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium text-[#1447E6]" title={e.key}>
                      {e.key}
                    </div>
                    <div className="mt-0.5 truncate text-[11px] text-[#94A3B8]" title={e.path}>
                      {e.path}
                    </div>
                    <div className="mt-1.5 flex flex-wrap gap-1">
                      {e.connectedToCanonicalEntry ? (
                        <Pill cls="border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]">已接入入口</Pill>
                      ) : (
                        <Pill cls="border-[#DDE7F2] bg-[#F8FAFC] text-[#475569]">仅建入口</Pill>
                      )}
                      {e.redLine && <Pill cls="border-[#FECACA] bg-[#FEF2F2] text-[#DC2626]">红线</Pill>}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// -----------------------------------------------------------------------------
// 未分类 / 待确认视图
// -----------------------------------------------------------------------------
const REVIEW_PAGE_SIZE = 60;

function NeedsReviewView({ onOpen }: { onOpen: (id: string) => void }) {
  const [cat, setCat] = useState("all");
  const [prevFilter, setPrevFilter] = useState<"all" | "yes" | "no">("all");
  const [page, setPage] = useState(0);
  const cats = needsReview.summary.byPrimaryCategory;

  const items = useMemo(() => {
    const base = cat === "all" ? needsReview.items : needsReview.items.filter((i) => i.guessPrimaryCategory === cat);
    const withPrev = base.map((i) => ({ item: i, preview: resolveReviewPreview(i.id) }));
    const filtered = withPrev.filter((x) => (prevFilter === "yes" ? !!x.preview : prevFilter === "no" ? !x.preview : true));
    // 可预览优先排序，便于先处理能看清的资源
    return filtered.sort((a, b) => Number(!!b.preview) - Number(!!a.preview));
  }, [cat, prevFilter]);

  const pageCount = Math.max(1, Math.ceil(items.length / REVIEW_PAGE_SIZE));
  const safePage = Math.min(page, pageCount - 1);
  const paged = items.slice(safePage * REVIEW_PAGE_SIZE, safePage * REVIEW_PAGE_SIZE + REVIEW_PAGE_SIZE);

  return (
    <div>
      <div className="rounded-[4px] border border-[#DDE7F2] bg-white p-4">
        <div className="flex items-center gap-2 text-sm font-medium text-[#0A0A0A]">
          <FileWarning className="size-4 text-[#B8640A]" /> 内联 SVG 待治理 · 阶段 8 输入清单（{needsReview.summary.total}）
        </div>
        <p className="mt-1.5 text-xs leading-relaxed text-[#64748B]">
          阶段 2~7 已完成：自有 SVG / 品牌 / 渠道 / 头像 / 业务图均已分类、去重并建立 canonical 入口。下列为阶段 8「组件资源使用映射」的输入清单——散落在各组件 / 页面内的内联 SVG，将在阶段 8 逐个判定去向：可由 lucide 覆盖的替换为系统图标、需跨处复用的抽为 registry SVG、组件私有的就地保留。本页只读展示，不删除、不替换、不进入 skill-map 候选。
        </p>
        {Object.keys(cats).length > 1 && (
          <div className="mt-3 flex flex-wrap gap-1.5">
            <SegBtn active={cat === "all"} onClick={() => { setCat("all"); setPage(0); }} label="全部" count={needsReview.summary.total} />
            {Object.entries(cats).map(([k, v]) => (
              <SegBtn key={k} active={cat === k} onClick={() => { setCat(k); setPage(0); }} label={CATEGORY_LABELS[k] ?? k} count={v} />
            ))}
          </div>
        )}
        {previewableReviewCount > 0 && (
          <div className="mt-2 flex flex-wrap items-center gap-1.5">
            <span className="mr-1 text-[11px] text-[#94A3B8]">可预览性</span>
            <SegBtn active={prevFilter === "all"} onClick={() => { setPrevFilter("all"); setPage(0); }} label="全部" count={needsReview.summary.total} />
            <SegBtn active={prevFilter === "yes"} onClick={() => { setPrevFilter("yes"); setPage(0); }} label="可预览" count={previewableReviewCount} />
            <SegBtn active={prevFilter === "no"} onClick={() => { setPrevFilter("no"); setPage(0); }} label="不可预览" count={needsReview.summary.total - previewableReviewCount} />
          </div>
        )}
      </div>

      <div className="mt-4 space-y-2">
        {paged.map(({ item: i, preview }) => {
          const hint = REVIEW_HINT[i.guessPrimaryCategory];
          const clickable = !!preview;
          return (
            <div
              key={i.id}
              role={clickable ? "button" : undefined}
              tabIndex={clickable ? 0 : undefined}
              onClick={clickable ? () => onOpen(i.id) : undefined}
              onKeyDown={
                clickable
                  ? (e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        onOpen(i.id);
                      }
                    }
                  : undefined
              }
              className={`flex gap-3 rounded-[4px] border border-[#DDE7F2] bg-white px-3 py-2.5 ${
                clickable ? "cursor-pointer transition-colors hover:border-[#BFCFFE]" : ""
              }`}
            >
              <div className="flex size-11 shrink-0 items-center justify-center rounded-[4px] bg-[#F8FAFC]">
                {preview ? (
                  <AssetPreview item={preview.item} sizeClass="size-7" />
                ) : (
                  <ImageOff className="size-4 text-[#CBD5E1]" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-sm font-medium text-[#0A0A0A]">{i.displayName}</span>
                  {i.type !== "inline-svg" && <Pill>{TYPE_LABELS[i.type] ?? i.type}</Pill>}
                  {i.guessPrimaryCategory !== "inline-svg" && (
                    <Pill cls="border-[#FED7AA] bg-[#FFF7ED] text-[#B8640A]">{CATEGORY_LABELS[i.guessPrimaryCategory] ?? i.guessPrimaryCategory}</Pill>
                  )}
                  {preview ? (
                    <Pill cls="border-[#BBF7D0] bg-[#F0FDF4] text-[#15803D]">可预览</Pill>
                  ) : (
                    <Pill cls="border-[#DDE7F2] bg-[#F8FAFC] text-[#94A3B8]">无预览</Pill>
                  )}
                  {i.dynamicDirReferenced && <Pill cls="border-[#BFDBFE] bg-[#EFF6FF] text-[#1447E6]">目录动态引用</Pill>}
                  <span className="text-[11px] text-[#94A3B8]">使用 {i.usageCount}</span>
                </div>
                <div className="mt-1 truncate text-[11px] text-[#94A3B8]" title={i.sourcePath ?? i.definedIn ?? ""}>
                  {shortPath(i.sourcePath ?? i.definedIn, 80)}
                </div>
                {hint && (
                  <div className="mt-1.5">
                    <Pill cls={hint.cls}>{hint.label}</Pill>
                  </div>
                )}
                {i.reasons.length > 0 && <p className="mt-1 text-xs leading-relaxed text-[#64748B]">{i.reasons.join("；")}</p>}
              </div>
            </div>
          );
        })}
      </div>

      {items.length > REVIEW_PAGE_SIZE && <Pager page={safePage} pageCount={pageCount} total={items.length} onPage={setPage} />}
    </div>
  );
}

// -----------------------------------------------------------------------------
// 详情面板
// -----------------------------------------------------------------------------
function SectionLabel({ children }: { children: React.ReactNode }) {
  return <div className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-[#94A3B8] first:mt-0">{children}</div>;
}

function DetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3 border-b border-[#F1F5F9] py-1.5 text-xs last:border-0">
      <span className="shrink-0 text-[#94A3B8]">{label}</span>
      <span className="min-w-0 break-all text-right text-[#334155]">{value}</span>
    </div>
  );
}

function DetailPanel({ item, onClose }: { item: InvItem; onClose: () => void }) {
  const cls = item.classification;
  const refs = usageMap[item.id] ?? [];
  const pages = pagesFromRefs(refs);
  const snippet = buildCopySnippet(item);
  const govGroup = cls.duplicateGroupId ? governance.groups.find((g) => g.id === cls.duplicateGroupId) : undefined;
  const yesNo = (v: boolean | null) => (v == null ? "—" : v ? "是" : "否");

  return (
    <>
      <div className="fixed inset-0 z-[10000] bg-black/20" onClick={onClose} />
      <aside className="fixed right-0 top-0 z-[10001] flex h-full w-[440px] max-w-[calc(100vw-24px)] flex-col border-l border-[#DDE7F2] bg-white shadow-xl">
        <div className="flex items-center justify-between border-b border-[#EAF1F8] px-5 py-3.5">
          <div className="min-w-0">
            <div className="truncate text-sm font-semibold text-[#0A0A0A]" title={item.displayName}>
              {item.displayName}
            </div>
            <div className="mt-0.5 text-[11px] text-[#94A3B8]">{CATEGORY_LABELS[cls.primaryCategory] ?? cls.primaryCategory}</div>
          </div>
          <button type="button" onClick={onClose} className="flex size-7 shrink-0 items-center justify-center rounded-[4px] text-[#64748B] hover:bg-[#F1F5F9]">
            <X className="size-4" />
          </button>
        </div>

        <div className="scrollbar-hide min-h-0 flex-1 overflow-y-auto px-5 py-4">
          {/* 待确认提示（置顶，预览图上方） */}
          {cls.needsReview && cls.needsReviewReasons.length > 0 && (
            <p className="mb-3 rounded-[4px] border border-[#FED7AA] bg-[#FFF7ED] px-2.5 py-2 text-[11px] leading-relaxed text-[#B8640A]">
              {cls.needsReviewReasons.join("；")}
            </p>
          )}

          {/* 视觉预览 */}
          <div className="grid grid-cols-2 gap-2">
            <div className="flex flex-col items-center gap-2 rounded-[4px] border border-[#DDE7F2] bg-[#F8FAFC] py-4">
              <AssetPreview item={item} sizeClass="size-12" />
              <div className="flex items-center gap-3">
                <AssetPreview item={item} sizeClass="size-6" />
                <AssetPreview item={item} sizeClass="size-4" />
              </div>
              <span className="text-[10px] text-[#94A3B8]">浅色底</span>
            </div>
            <div className="flex flex-col items-center gap-2 rounded-[4px] border border-[#334155] bg-[#0F172A] py-4">
              <AssetPreview item={item} sizeClass="size-12" bg="dark" />
              <div className="flex items-center gap-3">
                <AssetPreview item={item} sizeClass="size-6" bg="dark" />
                <AssetPreview item={item} sizeClass="size-4" bg="dark" />
              </div>
              <span className="text-[10px] text-[#64748B]">深色底</span>
            </div>
          </div>

          <SectionLabel>适合组件槽位</SectionLabel>
          {(() => {
            const slot = resolveComponentSlot(cls);
            if (!slot) {
              return (
                <p className="flex items-start gap-1.5 rounded-[4px] border border-[#DDE7F2] bg-[#F8FAFC] px-2.5 py-2 text-[11px] leading-relaxed text-[#64748B]">
                  <Info className="mt-0.5 size-3.5 shrink-0 text-[#94A3B8]" />
                  该资源未匹配阶段 8 审计产出的专属组件槽位（多为通用 lucide 可替代场景或纯页面级用法）。
                </p>
              );
            }
            const risk = RISK_LEVEL_LABELS[slot.riskLevel] ?? RISK_LEVEL_LABELS.medium;
            return (
              <div className="space-y-2 rounded-[4px] border border-[#BFDBFE] bg-[#EFF6FF] px-2.5 py-2.5 text-[11px] leading-relaxed">
                <p className="flex items-start gap-1.5 font-medium text-[#1447E6]">
                  <Boxes className="mt-0.5 size-3.5 shrink-0" />
                  <span>
                    {COMPONENT_SLOT_LABELS[cls.componentSlot ?? ""] ?? slot.label}
                    <span className="ml-1 text-[#64748B]">（{slot.componentSlotPath} · {slot.owningComponents.join(" / ")}）</span>
                  </span>
                </p>
                <div className="flex flex-wrap items-center gap-1.5">
                  <span
                    title={risk.tip}
                    className="inline-flex cursor-help items-center gap-1 rounded-[3px] border border-[#DDE7F2] bg-white px-1.5 py-0.5 text-[#64748B]"
                  >
                    <span className={`size-1.5 shrink-0 rounded-full ${risk.dot}`} />
                    可移植性风险 {risk.label}
                  </span>
                  <span className="rounded-[3px] border border-[#DDE7F2] bg-white px-1.5 py-0.5 text-[#475569]">
                    推荐资源 {slot.recommendedResourceType}
                  </span>
                  <span className="rounded-[3px] border border-[#DDE7F2] bg-white px-1.5 py-0.5 text-[#475569]">
                    lucide fallback {slot.allowLucideFallback ? "可" : "不可"}
                  </span>
                  {slot.redline && (
                    <span className="rounded-[3px] border border-[#FECACA] bg-[#FEF2F2] px-1.5 py-0.5 text-[#DC2626]">红线资源</span>
                  )}
                </div>
                <p className="text-[#475569]">{slot.constraint}</p>
                <p className="text-[#64748B]">跨仓约束：{slot.hostInjection}</p>
              </div>
            );
          })()}

          <SectionLabel>复制引用方式</SectionLabel>
          <div className="rounded-[4px] border border-[#DDE7F2] bg-[#0F172A]">
            <div className="flex items-center justify-between border-b border-white/10 px-3 py-1.5">
              <span className="text-[11px] text-[#94A3B8]">{snippet.lang}</span>
              <button
                type="button"
                onClick={() => copyText(snippet.code, "引用方式已复制")}
                className="flex items-center gap-1 rounded-[3px] px-1.5 py-0.5 text-[11px] text-[#CBD5E1] hover:bg-white/10 hover:text-white"
              >
                <Copy className="size-3" /> 复制
              </button>
            </div>
            <pre className="overflow-x-auto px-3 py-2.5 text-[11px] leading-relaxed text-[#E2E8F0]">{snippet.code}</pre>
          </div>

          <SectionLabel>基础信息</SectionLabel>
          <DetailRow label="类型" value={TYPE_LABELS[item.type] ?? item.type} />
          <DetailRow label="来源" value={item.source} />
          <DetailRow label="扫描范围" value={item.scanScope} />
          {item.fileName && <DetailRow label="文件名" value={item.fileName} />}
          {item.bytes != null && <DetailRow label="大小" value={formatBytes(item.bytes)} />}
          {item.svgMeta?.viewBox && <DetailRow label="viewBox" value={item.svgMeta.viewBox} />}
          {item.sourcePath && <DetailRow label="源路径" value={item.sourcePath} />}
          {item.definedIn && <DetailRow label="内联定义" value={`${item.definedIn}${item.definedAtLine ? `:${item.definedAtLine}` : ""}`} />}
          {item.importPath && <DetailRow label="import" value={item.importPath} />}
          {item.webPath && <DetailRow label="web 路径" value={item.webPath} />}
          {item.importName && <DetailRow label="lucide 名" value={item.importName} />}

          <SectionLabel>分类信息</SectionLabel>
          <DetailRow
            label="状态"
            value={<Pill cls={STATUS_META[cls.status]?.cls}>{STATUS_META[cls.status]?.label ?? cls.status}</Pill>}
          />
          <DetailRow label="一级分类" value={CATEGORY_LABELS[cls.primaryCategory] ?? cls.primaryCategory} />
          {cls.visualType && <DetailRow label="视觉类型" value={VISUAL_TYPE_LABELS[cls.visualType] ?? cls.visualType} />}
          {cls.componentSlot && <DetailRow label="组件槽位" value={COMPONENT_SLOT_LABELS[cls.componentSlot] ?? cls.componentSlot} />}
          {cls.usageScenarios.length > 0 && (
            <DetailRow label="用途场景" value={cls.usageScenarios.map((s) => SCENARIO_LABELS[s] ?? s).join(" / ")} />
          )}
          {cls.scenes.length > 0 && (
            <DetailRow label="出现端别" value={cls.scenes.map((s) => SCENARIO_LABELS[s] ?? s).join(" / ")} />
          )}
          <DetailRow
            label="使用页面"
            value={
              pages.length === 0 ? (
                <span className="text-[#94A3B8]">{item.dynamicDirReferenced ? "目录级动态引用（未定位具体页面）" : "未发现引用页面"}</span>
              ) : (
                <span className="flex flex-wrap justify-end gap-1">
                  {pages.slice(0, 12).map((p) => (
                    <Pill key={p}>{p}</Pill>
                  ))}
                  {pages.length > 12 && <span className="text-[11px] text-[#94A3B8]">等 {pages.length} 处</span>}
                </span>
              )
            }
          />
          {cls.tags.length > 0 && <DetailRow label="标签" value={cls.tags.join(" / ")} />}
          <DetailRow label="可改色" value={yesNo(cls.canRecolor)} />
          <DetailRow label="登记 registry" value={yesNo(cls.registry.registered)} />

          <SectionLabel>canonical 入口</SectionLabel>
          <DetailRow label="是 canonical" value={yesNo(cls.isCanonical)} />
          {cls.canonicalKey && (
            <DetailRow label="入口 key" value={<code className="rounded-[3px] bg-[#F1F5F9] px-1 py-0.5 text-[#1447E6]">{cls.canonicalKey}</code>} />
          )}
          <DetailRow label="已接入统一入口" value={yesNo(cls.connectedToCanonicalEntry)} />
          {cls.canonicalKey && (
            <p className="mt-2 rounded-[4px] border border-[#DDE7F2] bg-[#F8FAFC] px-2.5 py-2 text-[11px] leading-relaxed text-[#64748B]">
              {cls.connectedToCanonicalEntry
                ? "已接入：修改 canonical-assets.ts 中该 key 的路径，所有页面层引用一改多处生效。"
                : "仅建入口：现有散落引用保持原样，未做全量迁移（建设计划边界）。"}
            </p>
          )}

          <SectionLabel>使用位置（{refs.length}）</SectionLabel>
          {refs.length === 0 ? (
            <p className="text-xs text-[#94A3B8]">未发现静态引用。{item.dynamicDirReferenced ? "（存在目录级动态引用）" : ""}</p>
          ) : (
            <div className="space-y-1">
              {refs.slice(0, 30).map((r, idx) => (
                <div key={idx} className="flex items-center justify-between gap-2 rounded-[4px] border border-[#EAF1F8] bg-[#F8FAFC] px-2 py-1 text-[11px]">
                  <span className="truncate text-[#475569]" title={r.file}>
                    {shortPath(r.file, 44)}:{r.line}
                  </span>
                  <Pill>{r.kind}</Pill>
                </div>
              ))}
              {refs.length > 30 && <p className="text-[11px] text-[#94A3B8]">仅显示前 30 处，共 {refs.length} 处。</p>}
            </div>
          )}

          {govGroup && (
            <>
              <SectionLabel>重复治理信息</SectionLabel>
              <div className="rounded-[4px] border border-[#DDE7F2] bg-white p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs font-medium text-[#0A0A0A]">{govGroup.id}</span>
                  <Pill>{govGroup.grade} 类</Pill>
                  {GOV_STATE_META[govGroup.state] && <Pill cls={GOV_STATE_META[govGroup.state].cls}>{GOV_STATE_META[govGroup.state].label}</Pill>}
                  {govGroup.redLine && <Pill cls="border-[#FECACA] bg-[#FEF2F2] text-[#DC2626]">红线</Pill>}
                </div>
                {govGroup.canonicalPath && <div className="mt-1.5 break-all text-[11px] text-[#475569]">canonical：{govGroup.canonicalPath}</div>}
                <div className="mt-2 flex items-center gap-1 text-[11px] text-[#94A3B8]">
                  <ChevronRight className="size-3" /> 共 {govGroup.members.length} 个成员（详见「重复治理」视图）
                </div>
              </div>
            </>
          )}
        </div>

        <div className="flex items-center gap-2 border-t border-[#EAF1F8] px-5 py-3">
          {item.webPath && (
            <a
              href={item.webPath}
              target="_blank"
              rel="noreferrer"
              className="flex h-8 items-center gap-1.5 rounded-[4px] border border-[#DDE7F2] bg-white px-3 text-xs text-[#334155] hover:border-[#1447E6]"
            >
              <ExternalLink className="size-3.5" /> 新标签打开
            </a>
          )}
          <button
            type="button"
            onClick={() => copyText(item.sourcePath ?? item.importPath ?? item.webPath ?? item.id, "路径已复制")}
            className="flex h-8 items-center gap-1.5 rounded-[4px] border border-[#DDE7F2] bg-white px-3 text-xs text-[#334155] hover:border-[#1447E6]"
          >
            <Copy className="size-3.5" /> 复制路径
          </button>
        </div>
      </aside>
    </>
  );
}

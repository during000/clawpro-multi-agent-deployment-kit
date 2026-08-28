// =============================================================================
// 阶段 3：资源分类与待确认清单（classify-resources.mjs）
// -----------------------------------------------------------------------------
// 定位：读取阶段 2 的事实基线（resource-inventory.generated.json /
//       resource-usage.generated.json）与资产事实源（icon-registry.example.json），
//       按【透明、确定性的规则】回填 inventory 的 classification 字段，并产出
//       「未分类 / 待确认清单」。
//
// 纪律（务必遵守）：
//   1. 只读输入资源文件（读 SVG 内容仅用于检测视觉信号），不修改/删除任何资源。
//   2. 不动 skill（非阶段 9）。
//   3. **只在强信号下确定分类**（目录语义 / registry 登记 / SVG 内容信号 /
//      usage 引用端别）；弱信号一律标 needs-review 入待确认清单，绝不臆测。
//   4. **不做重复/canonical 判定**（duplicateGroupId / canonicalId 留给阶段 4/6）。
//   5. 规则可重复执行、幂等：流水线顺序为「先 scan-resources 再 classify-resources」。
//
// 运行：node client/src/design-assets/scripts/classify-resources.mjs
// =============================================================================

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "../../../../");

const GEN_DIR = path.join(repoRoot, "client/src/design-assets/generated");
const INVENTORY_PATH = path.join(GEN_DIR, "resource-inventory.generated.json");
const USAGE_PATH = path.join(GEN_DIR, "resource-usage.generated.json");
const REGISTRY_PATH = path.join(
  repoRoot,
  ".codebuddy/skills/clawpro-portable-design-skill/assets/icon-registry.example.json"
);
const NEEDS_REVIEW_PATH = path.join(GEN_DIR, "resource-needs-review.generated.json");
// 人工事实源：渐变图标「线条/块状」分拣（由 _gradient-sorter.html 导出）
const GRADIENT_STYLE_PATH = path.join(
  repoRoot,
  "client/src/design-assets/manual-overrides/gradient-style.json"
);
// 人工事实源：分类修正（视觉类型 / 端别 / 组件槽位 / 是否纳入库）
const CLASSIFICATION_OVERRIDES_PATH = path.join(
  repoRoot,
  "client/src/design-assets/manual-overrides/classification.json"
);

// ---------------------------------------------------------------------------
// 读取输入
// ---------------------------------------------------------------------------
const inventory = JSON.parse(fs.readFileSync(INVENTORY_PATH, "utf8"));
const usage = JSON.parse(fs.readFileSync(USAGE_PATH, "utf8"));
const registry = JSON.parse(fs.readFileSync(REGISTRY_PATH, "utf8"));

// 渐变线条/块状人工清单：id -> "gradient-line" | "gradient-solid"（缺失则不细分，绝不臆测）
const gradientStyleById = new Map();
try {
  const gs = JSON.parse(fs.readFileSync(GRADIENT_STYLE_PATH, "utf8"));
  for (const [id, v] of Object.entries(gs.overrides || {})) {
    if (v && (v.style === "gradient-line" || v.style === "gradient-solid")) {
      gradientStyleById.set(id, v.style);
    }
  }
} catch {
  // 文件不存在或为空：所有 gradient 保留待细分
}

// 分类修正：id -> { visualType?, scenes?, componentSlot?, excludeFromLibrary?, confirmed? }（仅登记项生效）
const overridesById = new Map();
try {
  const ov = JSON.parse(fs.readFileSync(CLASSIFICATION_OVERRIDES_PATH, "utf8"));
  for (const [id, v] of Object.entries(ov.overrides || {})) {
    if (v && typeof v === "object") overridesById.set(id, v);
  }
} catch {
  // 文件不存在或为空：无人工修正
}

// registry：sourcePath -> { id, category, status }（skill 自带可移植样例，仅作分类 category/status 提示输入；非本项目候选准入闸门——本项目资源真相以 inventory 为准，详见建设计划 §前置约束第 3 条 / §5.5 口径修正）
const registryByPath = new Map();
for (const icon of registry.icons || []) {
  if (icon.path) registryByPath.set(icon.path, icon);
}

// registry.category -> 本项目 usageScenarios 枚举（仅映射语义明确者，其余留空待确认）
const REGISTRY_CAT_TO_SCENARIO = {
  metric: "metric",
  security: "security",
  status: "status",
  navigation: "navigation",
  action: "action",
  "empty-state": "empty-hint",
  "upload-placeholder": "action",
};

// ---------------------------------------------------------------------------
// 端别（scenes）：优先用 usage 引用文件路径反推（事实），无引用再退回目录信号
// ---------------------------------------------------------------------------
function sceneFromRefFile(file) {
  if (!file) return null;
  if (file.includes("/pages/admin/") || file.includes("/admin/")) return "admin";
  if (file.includes("/pages/tenant/") || file.includes("/tenant/")) return "tenant";
  if (file.includes("/pages/landing/") || /LandingPage|landing-assets/.test(file))
    return "landing";
  if (file.includes("/pages/design-system/") || file.includes("/design-system/"))
    return "global"; // 设计系统/展示台，视为全局
  return null;
}

function scenesFromUsage(resourceId) {
  const refs = usage.usage?.[resourceId] || [];
  const set = new Set();
  for (const r of refs) {
    const s = sceneFromRefFile(r.file);
    if (s) set.add(s);
  }
  return [...set];
}

// ---------------------------------------------------------------------------
// 视觉类型：仅用 SVG 内容里的确定性信号判定，无法确定 line/solid 则留空待确认
// ---------------------------------------------------------------------------
const svgContentCache = new Map();
function readSvg(sourcePath) {
  if (!sourcePath) return null;
  if (svgContentCache.has(sourcePath)) return svgContentCache.get(sourcePath);
  let c = null;
  try {
    c = fs.readFileSync(path.join(repoRoot, sourcePath), "utf8");
  } catch {
    c = null;
  }
  svgContentCache.set(sourcePath, c);
  return c;
}

function detectSvgVisualSignals(sourcePath) {
  const c = readSvg(sourcePath);
  if (!c) return { ok: false };
  return {
    ok: true,
    gradient: /<(linear|radial)Gradient|data-figma-skip-parse/i.test(c),
    currentColor: /currentColor/.test(c),
    fixedFill: /(fill|stroke)\s*=\s*["']#[0-9a-f]{3,8}["']/i.test(c),
  };
}

// ---------------------------------------------------------------------------
// 主分类规则（precedence：自上而下，先命中先定）
// ---------------------------------------------------------------------------
function classify(item) {
  const sp = item.sourcePath || "";
  const fileName = item.fileName || "";
  const reasons = [];
  let primaryCategory = null;
  let usageScenarios = [];
  let usageScenarioResolved = false;
  let visualType = null;
  let visualTypeResolved = false;
  let canRecolor = null;
  let needsReview = false;
  let status = "normal";
  let archived = false;

  const reg = registryByPath.get(sp) || null;
  const registryInfo = reg
    ? { registered: true, id: reg.id, category: reg.category, status: reg.status }
    : { registered: false };

  // ---- 1. lucide：通用单色图标 ----
  if (item.type === "lucide") {
    primaryCategory = "lucide";
    visualType = "monochrome-currentColor";
    visualTypeResolved = true;
    canRecolor = true;
    usageScenarios = []; // lucide 不做自有 SVG 的场景细分
    usageScenarioResolved = true; // 视为已处理（lucide 不需要场景）
    status = "normal";
  }
  // ---- 2. inline-svg：治理对象，需人工判断（抽取/替换/保留组件私有）----
  else if (item.type === "inline-svg") {
    primaryCategory = "inline-svg";
    needsReview = true;
    status = "needs-review";
    reasons.push("inline SVG：治理对象，需判断抽为资源 / 替换为已有图标 / 保留组件私有");
  }
  // ---- 3. scan-only（根 assets/CodeBuddyAssets）：Figma 设计导出原件 → 默认归档，不进治理/待确认 ----
  else if (item.scanScope === "scan-only") {
    primaryCategory = "scan-only-export";
    archived = true; // 设计导出原件：归档留存，不进浏览池、不进待确认清单（仅供溯源，无代码引用）
    needsReview = false;
    status = "normal";
    canRecolor = false;
    reasons.push("根 assets/ Figma 设计导出原件（scan-only）：已归档留存，不纳入资源库治理");
  }
  // ---- 4. Agent 头像：avatars/ 或 agent-card 头像精灵 ----
  else if (/\/avatars\//.test(sp) || /agent-card\/avatar-sprite/.test(sp)) {
    primaryCategory = "agent-avatar";
    usageScenarios = ["agent"];
    usageScenarioResolved = true;
    visualType = "avatar-like";
    visualTypeResolved = true;
    canRecolor = false; // 头像不当普通图标改色（硬约束）
  }
  // ---- 5. 渠道图标：admin-channel-icons/channel-*（品牌固定色），排除空态 ----
  else if (/admin-channel-icons\/channel-/.test(sp)) {
    primaryCategory = "channel-icon";
    usageScenarios = ["channel"];
    usageScenarioResolved = true;
    visualType = "brand-fixed";
    visualTypeResolved = true;
    canRecolor = false; // 渠道品牌固定色（硬约束）
  }
  // ---- 6. 品牌 Logo：clawpro-logo / 文件名含 logo ----
  else if (/clawpro-logo|(^|\/)[^/]*logo[^/]*\.svg$/i.test(sp)) {
    primaryCategory = "brand-logo";
    usageScenarios = ["brand-product"];
    usageScenarioResolved = true;
    visualType = "brand-fixed";
    visualTypeResolved = true;
    canRecolor = false; // 品牌 Logo 不当普通图标改色（硬约束）
  }
  // ---- 7. admin-channel-icons 下的空态（empty-custom-channel）----
  else if (/admin-channel-icons\/empty-/.test(sp)) {
    primaryCategory = "own-svg";
    visualType = "illustrative-icon";
    needsReview = true;
    status = "needs-review";
    reasons.push("渠道目录下的空态图（empty-*）：归类待确认（empty-hint vs 业务图片）");
  }
  // ---- 8. 空状态插画 PNG（empty-*.png）----
  else if (item.type === "png" && /(^|\/)empty-/.test(fileName ? `/${fileName}` : sp)) {
    primaryCategory = "business-image";
    visualType = "asset-fixed-color";
    visualTypeResolved = true;
    canRecolor = false;
    if (reg && reg.status === "approved") {
      // empty-no-data.png 已登记 approved
      usageScenarios = REGISTRY_CAT_TO_SCENARIO[reg.category]
        ? [REGISTRY_CAT_TO_SCENARIO[reg.category]]
        : [];
      usageScenarioResolved = usageScenarios.length > 0;
      status = "normal";
    } else {
      needsReview = true;
      status = "needs-review";
      reasons.push("空状态插画 PNG：背景目标默认不纳入大插画，待确认是否纳入资源库");
    }
  }
  // ---- 9. 其它 PNG（public/icon 业务栅格图标、agent sprite 已在 4 处理）----
  else if (item.type === "png") {
    primaryCategory = "business-image";
    visualType = "asset-fixed-color";
    visualTypeResolved = true;
    canRecolor = false;
    needsReview = true;
    status = "needs-review";
    reasons.push("栅格业务图标 PNG：待确认是否纳入 / 是否已被同名 SVG 取代");
  }
  // ---- 10. SVG（registry 已登记 或 include 目录自有 SVG）----
  else if (item.type === "svg") {
    primaryCategory = "own-svg";

    // 视觉类型：仅凭内容确定性信号
    const sig = detectSvgVisualSignals(sp);
    if (sig.ok) {
      if (sig.gradient) {
        // 渐变图标：是否细分为「线条/块状」取决于人工事实源；未登记则保留 gradient（待细分），绝不臆测
        const manualStyle = gradientStyleById.get(item.id);
        visualType = manualStyle || "gradient";
        visualTypeResolved = true;
        canRecolor = false; // 渐变图标不建议随意改色
      } else if (sig.currentColor) {
        visualType = "monochrome-currentColor";
        visualTypeResolved = true;
        canRecolor = true;
      } else {
        // 固定色但无法区分 line/solid → 留空待确认
        visualType = null;
        visualTypeResolved = false;
        canRecolor = sig.fixedFill ? false : null;
      }
    }

    // 使用场景：registry 优先（事实源），否则目录强信号，否则待确认
    if (reg) {
      status = "normal"; // registry approved，身份已确认
      const mapped = REGISTRY_CAT_TO_SCENARIO[reg.category];
      if (mapped) {
        usageScenarios = [mapped];
        usageScenarioResolved = true;
      } else {
        usageScenarios = []; // memory/business/feature/integration/misc/infrastructure 等无干净映射
        usageScenarioResolved = false;
      }
    } else if (/admin-sidebar\//.test(sp) || /\/topnav\//.test(sp)) {
      usageScenarios = ["navigation"];
      usageScenarioResolved = true;
    } else if (/admin-security\//.test(sp)) {
      usageScenarios = ["security"];
      usageScenarioResolved = true;
    } else if (/agent-card\//.test(sp)) {
      usageScenarios = ["agent"];
      usageScenarioResolved = true;
    } else {
      // 业务模块目录（memory/platform-policy/member-config/session/cloud-dev/
      // network/resource-mgmt/skill-packages/disk-mgmt、public/icon、src/assets/icons 等）
      // 语义未确定性映射到使用场景枚举 → 待设计确认
      usageScenarios = [];
      usageScenarioResolved = false;
      needsReview = true;
      status = "needs-review";
      reasons.push("自有 SVG：目录语义未确定性映射到使用场景枚举，需设计确认主场景");
    }
  }
  // ---- 兜底 ----
  else {
    primaryCategory = "uncategorized";
    needsReview = true;
    status = "needs-review";
    reasons.push("未命中任何分类规则，需人工确认");
  }

  // 端别：usage 反推优先，目录信号兜底
  let scenes = [];
  let sceneSource = "none";
  if (item.type === "lucide") {
    scenes = scenesFromUsage(item.id);
    if (scenes.length === 0) scenes = ["global"];
    sceneSource = "usage-or-default";
  } else {
    scenes = scenesFromUsage(item.id);
    if (scenes.length > 0) {
      sceneSource = "usage";
    } else if (/(^|\/)admin-|client\/public\/assets\/admin/.test(sp) || /admin-/.test(sp)) {
      scenes = ["admin"];
      sceneSource = "directory";
    } else if (registryInfo.registered || /(^|\/)icon\//.test(sp)) {
      // registry 业务图标 / 根 icon：本项目以管控端为主，记 admin
      scenes = ["admin"];
      sceneSource = "directory";
    } else {
      scenes = [];
      sceneSource = "none";
    }
  }

  // 人工分类修正（最高优先级，仅 classification.json 登记项生效，绝不臆测）
  const ov = overridesById.get(item.id);
  if (ov) {
    if (ov.visualType) {
      visualType = ov.visualType;
      visualTypeResolved = true;
    }
    if (Array.isArray(ov.scenes) && ov.scenes.length > 0) {
      scenes = ov.scenes;
      sceneSource = "manual-override";
    }
  }

  const excludeFromLibrary = !!(ov && ov.excludeFromLibrary);

  // 终态修正（须在 canUseInSkillMap 计算之前，保证派生口径一致）：
  //   - confirmed=true：设计已确认该自有 SVG 的身份/主场景 → 置「已确认」并移出待确认清单。
  //   - excludeFromLibrary=true：已明确不纳入资源库（组件私有 / 陈旧件）→ 无需再「待确认」，
  //     移出待确认清单（仍不进浏览池，由页面侧 excludeFromLibrary 过滤）。
  // 二者均仅在 classification.json 显式登记时生效，绝不臆测。
  if ((ov && ov.confirmed === true) || excludeFromLibrary) {
    status = "normal";
    needsReview = false;
    reasons.length = 0;
  }

  // canUseInSkillMap：阶段 9 决策；此处仅【排除】明确不可者，其余留 null（未决）
  let canUseInSkillMap = null;
  if (
    needsReview ||
    status === "needs-review" ||
    item.type === "inline-svg" ||
    item.scanScope === "scan-only" ||
    archived ||
    excludeFromLibrary
  ) {
    canUseInSkillMap = false;
  }

  // 组件槽位（自有 SVG 子菜单）：人工 componentSlot 优先；否则按目录/视觉确定性派生
  //   - admin-sidebar/ 下的自有 SVG → AdminSidebar 导航图标
  //   - 其余块状多彩渐变自有 SVG → 卡片左侧图标
  let componentSlot = (ov && ov.componentSlot) || null;
  if (!componentSlot && primaryCategory === "own-svg") {
    if (/admin-sidebar\//.test(sp)) componentSlot = "admin-sidebar";
    else if (visualType === "gradient-solid") componentSlot = "card-left-icon";
  }

  return {
    pending: false,
    primaryCategory,
    usageScenarios,
    usageScenarioResolved,
    visualType,
    visualTypeResolved,
    scenes,
    sceneSource,
    componentSlot,
    excludeFromLibrary,
    archived,
    tags: [],
    status,
    needsReview,
    needsReviewReasons: reasons,
    duplicateGroupId: null, // 阶段 4
    canonicalId: null, // 阶段 6
    canRecolor,
    canUseInSkillMap,
    registry: registryInfo,
  };
}

// ---------------------------------------------------------------------------
// 执行分类
// ---------------------------------------------------------------------------
// 阶段 4/6 会在 classification 上回填重复/canonical 字段（duplicate* / isCanonical / canonical* /
// connectedToCanonicalEntry）。阶段 5 删除是不可逆的一次性动作，之后全链路重跑会因「文件已删」导致
// 重复组重算漂移。为此 classify 重跑只覆盖自身负责的字段，保留下游阶段已回填的增量字段（首次运行这些
// key 不存在，自然采用 classify 的占位默认值），从而本脚本可在治理后安全地单独重跑。
const DOWNSTREAM_KEYS = [
  "duplicateGroupId",
  "duplicateRole",
  "duplicateGrade",
  "isCanonical",
  "canonicalId",
  "canonicalKey",
  "connectedToCanonicalEntry",
];
for (const item of inventory.items) {
  const prev = item.classification || {};
  const next = classify(item);
  for (const k of DOWNSTREAM_KEYS) {
    if (k in prev) next[k] = prev[k];
  }
  item.classification = next;
}

// 统计
const items = inventory.items;
const tally = (fn) => items.filter(fn).length;
const groupCount = (keyFn) => {
  const m = {};
  for (const it of items) {
    const k = keyFn(it);
    if (k == null) continue;
    m[k] = (m[k] || 0) + 1;
  }
  return Object.fromEntries(Object.entries(m).sort((a, b) => b[1] - a[1]));
};

const classificationSummary = {
  total: items.length,
  byPrimaryCategory: groupCount((it) => it.classification.primaryCategory),
  byStatus: groupCount((it) => it.classification.status),
  needsReview: tally((it) => it.classification.needsReview),
  usageScenarioResolved: tally((it) => it.classification.usageScenarioResolved),
  usageScenarioPending: tally(
    (it) => !it.classification.usageScenarioResolved && !it.classification.needsReview
  ),
  visualTypeResolved: tally((it) => it.classification.visualTypeResolved),
  byVisualType: groupCount((it) => it.classification.visualType),
  byComponentSlot: groupCount((it) => it.classification.componentSlot),
  excludedFromLibrary: tally((it) => it.classification.excludeFromLibrary),
  sceneBySource: groupCount((it) => it.classification.sceneSource),
};

inventory.stage = 3;
inventory.classifiedAt = new Date().toISOString();
inventory.note =
  "阶段 3 已回填 classification（一级分类/使用场景/视觉类型/端别/状态）。" +
  "强信号确定分类，弱信号标 needs-review 入待确认清单；未做重复/canonical 判定（duplicateGroupId/canonicalId 留阶段 4/6）。";
inventory.classificationSummary = classificationSummary;

fs.writeFileSync(INVENTORY_PATH, JSON.stringify(inventory, null, 2) + "\n", "utf8");

// ---------------------------------------------------------------------------
// 待确认清单
// ---------------------------------------------------------------------------
const needsReviewItems = items
  .filter((it) => it.classification.needsReview)
  .map((it) => ({
    id: it.id,
    displayName: it.displayName,
    type: it.type,
    source: it.source,
    scanScope: it.scanScope,
    sourcePath: it.sourcePath,
    definedIn: it.definedIn || null,
    usageCount: it.usageCount,
    dynamicDirReferenced: it.dynamicDirReferenced || false,
    guessPrimaryCategory: it.classification.primaryCategory,
    reasons: it.classification.needsReviewReasons,
  }))
  .sort(
    (a, b) =>
      (a.guessPrimaryCategory || "").localeCompare(b.guessPrimaryCategory || "") ||
      (a.sourcePath || a.definedIn || "").localeCompare(b.sourcePath || b.definedIn || "")
  );

const needsReviewByReasonType = groupCount((it) =>
  it.classification.needsReview ? it.classification.primaryCategory : null
);

const needsReviewOut = {
  $schema: "clawpro-resource-needs-review",
  stage: 3,
  generatedAt: new Date().toISOString(),
  note:
    "未分类 / 待确认清单：扫描+规则无法确定性判断的资源，集中交设计团队确认。" +
    "不删除、不替换、不进入 skill-map 候选（canUseInSkillMap=false）。",
  summary: {
    total: needsReviewItems.length,
    byPrimaryCategory: needsReviewByReasonType,
  },
  items: needsReviewItems,
};

fs.writeFileSync(NEEDS_REVIEW_PATH, JSON.stringify(needsReviewOut, null, 2) + "\n", "utf8");

// ---------------------------------------------------------------------------
// 控制台摘要
// ---------------------------------------------------------------------------
console.log("== 阶段 3 分类完成 ==");
console.log("一级分类:", classificationSummary.byPrimaryCategory);
console.log("状态:", classificationSummary.byStatus);
console.log("视觉类型:", classificationSummary.byVisualType);
console.log(
  "需确认:",
  classificationSummary.needsReview,
  "| 场景已解析:",
  classificationSummary.usageScenarioResolved,
  "| 场景待补(非review):",
  classificationSummary.usageScenarioPending,
  "| 视觉已解析:",
  classificationSummary.visualTypeResolved
);
console.log("端别来源:", classificationSummary.sceneBySource);
console.log("待确认清单条目:", needsReviewOut.summary.total, needsReviewOut.summary.byPrimaryCategory);
console.log("产物:", path.relative(repoRoot, INVENTORY_PATH), "|", path.relative(repoRoot, NEEDS_REVIEW_PATH));

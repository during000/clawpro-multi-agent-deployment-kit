// =============================================================================
// 阶段 9：skill 连接 · resource-skill-map 生成（build-resource-skill-map.mjs）
// -----------------------------------------------------------------------------
// 定位（严格遵守建设计划 §5.5 / §6.3 / §6.5 / §九 阶段 9）：
//   - 让 clawpro-portable-design-skill 在「当前项目页面」生成时，能稳定选择资源。
//   - skill 不读取页面、不读 inventory 全量，只读本脚本产出的精简稳定映射
//     client/src/design-assets/resource-skill-map.json。
//
// 候选事实源（用户拍板 A：以 inventory 为本项目资源真相，registry 降格为可移植样例）：
//   - slot 候选：inventory 中 status=normal && 未排除 && 未归档 && 有 componentSlot 的资源，
//     其 slot 约束取自阶段 8 真实产出 manual-overrides/component-resource-map.json。
//   - 红线候选：inventory 中带 canonicalKey 的资源（品牌/渠道/头像），其 slot 即 primaryCategory
//     （channel-icon / brand-logo / agent-avatar，与白名单一一对应），跨仓由宿主注入。
//
// 纪律（务必遵守，以阶段 2~8 真实产出为准、零猜测）：
//   1. **不臆造候选**：候选只能来自 inventory 已审计字段（componentSlot / canonicalKey 等），
//      不手填、不靠语义猜 slot 归属。
//   2. **slot 必须合法**：每个候选的 slot 必须在 component-resource-map.slots 白名单内；
//      红线候选必须落在 redline=true 的 slot，否则报错退出。
//   3. **registry 不做 approved 闸门**：icon-registry.example.json 仅作为 skill 的「可移植
//      身份样例」记录在 registrySample，**不**作为候选准入闸门（本项目真相是 inventory）。
//   4. **canUseInSkillMap 只在内存计算**：据规则计算候选准入，仅产出到 resource-skill-map.json；
//      **不回写** generated/resource-inventory.generated.json（保持其为纯扫描/分类基线产物）。
//   5. **不引用已删资源**：governance.removedIds 中的资源一律不进候选。
//   6. **校验优先、失败即停**：候选 slot 非法 / 磁盘文件不存在 / 红线落错槽位，任一不符立即
//      报错退出，不写任何产物、不静默放过。
//   7. **幂等可重复**：纯函数式从输入派生，重跑结果稳定（按 slot、id 排序）。
//   8. **不动组件源码、不改组件 API、不做迁移**：本脚本只读 + 产出映射。
//
// 运行：node client/src/design-assets/scripts/build-resource-skill-map.mjs
//
// 输入（只读）：
//   client/src/design-assets/generated/resource-inventory.generated.json   （阶段 2~6 join 表）
//   client/src/design-assets/manual-overrides/component-resource-map.json   （阶段 8 槽位约束事实源）
//   client/src/design-assets/generated/resource-governance.generated.json   （removedIds）
// 产物：
//   client/src/design-assets/resource-skill-map.json                        （新建/覆盖）
// =============================================================================

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "../../../../");
const DA_DIR = path.join(repoRoot, "client/src/design-assets");
const GEN_DIR = path.join(DA_DIR, "generated");
const INVENTORY_PATH = path.join(GEN_DIR, "resource-inventory.generated.json");
const GOV_PATH = path.join(GEN_DIR, "resource-governance.generated.json");
const COMPONENT_MAP_PATH = path.join(DA_DIR, "manual-overrides/component-resource-map.json");
const SKILL_MAP_PATH = path.join(DA_DIR, "resource-skill-map.json");

// skill 侧「可移植身份样例」（仅记录，不作准入闸门，见纪律 3）
const REGISTRY_SAMPLE = ".codebuddy/skills/clawpro-portable-design-skill/assets/icon-registry.example.json";

// ---------------------------------------------------------------------------
// 读取输入（只读）
// ---------------------------------------------------------------------------
const inventory = JSON.parse(fs.readFileSync(INVENTORY_PATH, "utf8"));
const componentMap = JSON.parse(fs.readFileSync(COMPONENT_MAP_PATH, "utf8"));
const governance = JSON.parse(fs.readFileSync(GOV_PATH, "utf8"));

const removedIds = new Set(governance.removedIds || []);
const slotsSpec = componentMap.slots || {};
const slotWhitelist = new Set(Object.keys(slotsSpec));
const redlineSlots = new Set(
  Object.entries(slotsSpec).filter(([, s]) => s.redline === true).map(([k]) => k),
);

// component-resource-map.findings 中与槽位相关的「据实发现」，挂到对应 slot（不丢阶段 8 记录）
const SLOT_FINDINGS = {
  "run-status": componentMap.findings?.runStatusStaticUnreferenced?.summary || null,
  "file-type": componentMap.findings?.fileTypeStaticUnreferenced?.summary || null,
};

// ---------------------------------------------------------------------------
// 候选判定（确定性、零猜测）
// ---------------------------------------------------------------------------
const errors = [];
/** slot -> candidate[] */
const candidatesBySlot = {};
for (const s of slotWhitelist) candidatesBySlot[s] = [];

function isAvailable(cls) {
  return cls && cls.status === "normal" && !cls.excludeFromLibrary && !cls.archived;
}

/** 资源落地形态：public/src 文件 → file（需磁盘校验）；inline-svg → inline（不校验磁盘） */
function landingOf(item) {
  return item.source === "inline-svg" ? "inline" : "file";
}

/** 当前项目页面引用方式 */
function referenceOf(item, cls, slot) {
  if (cls.canonicalKey) return { kind: "canonical", value: `canonicalAssets.${cls.canonicalKey}` };
  if (item.webPath) return { kind: "web-path", value: item.webPath };
  if (item.importPath) return { kind: "esm-import", value: item.importPath };
  if (item.source === "inline-svg") return { kind: "inline", value: item.definedIn || item.sourcePath || null };
  return { kind: "unknown", value: null };
}

function buildCandidate(item, slot, channel) {
  const cls = item.classification || {};
  const landing = landingOf(item);
  const redline = redlineSlots.has(slot);
  return {
    id: item.id,
    displayName: item.displayName,
    slot,
    via: channel, // "component-slot" | "canonical-redline"
    type: item.type,
    source: item.source,
    primaryCategory: cls.primaryCategory,
    visualType: cls.visualType,
    scenes: cls.scenes || [],
    landing,
    sourcePath: item.sourcePath || null,
    webPath: item.webPath || null,
    importPath: item.importPath || null,
    canonicalKey: cls.canonicalKey || null,
    reference: referenceOf(item, cls, slot),
    // 选图约束（取自阶段 8 槽位规格 + 资源自身属性，不臆造）
    redline,
    canRecolor: cls.canRecolor === true,
    allowLucideFallback: slotsSpec[slot]?.allowLucideFallback === true,
    recommendedResourceType: slotsSpec[slot]?.recommendedResourceType || null,
    // usageScope（建设计划 §6.5 第二层要求）：红线跨仓由宿主注入；其余为当前项目专属
    usageScope: redline ? "host-injected" : "current-project-only",
  };
}

for (const item of inventory.items || []) {
  if (removedIds.has(item.id)) continue;
  const cls = item.classification;
  if (!cls) continue;
  const where = `${item.id} (${item.displayName})`;

  // (a) 红线候选：带 canonicalKey 的品牌/渠道/头像，slot 即 primaryCategory
  if (cls.canonicalKey) {
    if (!isAvailable(cls)) {
      errors.push(`红线资源非 normal/可用，不应有 canonicalKey：${where} status=${cls.status}`);
      continue;
    }
    const slot = cls.primaryCategory;
    if (!slotWhitelist.has(slot)) {
      errors.push(`红线资源 primaryCategory 不在槽位白名单：${where} category=${slot}`);
      continue;
    }
    if (!redlineSlots.has(slot)) {
      errors.push(`红线资源落在非红线槽位：${where} slot=${slot}`);
      continue;
    }
    candidatesBySlot[slot].push(buildCandidate(item, slot, "canonical-redline"));
    continue;
  }

  // (b) slot 候选：有 componentSlot 且 normal 可用
  if (cls.componentSlot && isAvailable(cls)) {
    const slot = cls.componentSlot;
    if (!slotWhitelist.has(slot)) {
      errors.push(`componentSlot 不在槽位白名单：${where} slot=${slot}`);
      continue;
    }
    candidatesBySlot[slot].push(buildCandidate(item, slot, "component-slot"));
    continue;
  }
  // 其余资源 canUseInSkillMap=false，不进 skill-map（不回写 inventory，见纪律 4）
}

// ---------------------------------------------------------------------------
// 落地磁盘校验（file 形态必须存在；inline 形态不校验静态文件）— 失败即停
// ---------------------------------------------------------------------------
for (const slot of Object.keys(candidatesBySlot)) {
  for (const c of candidatesBySlot[slot]) {
    if (c.landing !== "file") continue;
    if (!c.sourcePath) {
      errors.push(`file 候选缺 sourcePath：${c.id} (${c.displayName}) slot=${slot}`);
      continue;
    }
    const disk = path.join(repoRoot, c.sourcePath);
    if (!fs.existsSync(disk)) {
      errors.push(`磁盘文件不存在：${c.id} (${c.displayName}) -> ${c.sourcePath}`);
    }
  }
}

if (errors.length) {
  console.error("[build-resource-skill-map] 校验失败，已中止（未写任何产物）：");
  for (const e of errors) console.error("  - " + e);
  process.exit(1);
}

// ---------------------------------------------------------------------------
// 排序（确定性）：候选先红线优先、再按 displayName/id
// ---------------------------------------------------------------------------
for (const slot of Object.keys(candidatesBySlot)) {
  candidatesBySlot[slot].sort((a, b) =>
    String(a.displayName || a.id).localeCompare(String(b.displayName || b.id), "zh") ||
    a.id.localeCompare(b.id),
  );
}

// ---------------------------------------------------------------------------
// 组装产物（slots 全集 9 个，按白名单顺序；每个挂约束 + candidates）
// ---------------------------------------------------------------------------
const slotsOut = {};
let totalCandidates = 0;
const bySlotCount = {};
for (const slotName of Object.keys(slotsSpec)) {
  const spec = slotsSpec[slotName];
  const cands = candidatesBySlot[slotName] || [];
  totalCandidates += cands.length;
  bySlotCount[slotName] = cands.length;
  slotsOut[slotName] = {
    label: spec.label,
    componentSlotPath: spec.componentSlotPath,
    owningComponents: spec.owningComponents,
    recommendedResourceType: spec.recommendedResourceType,
    allowLucideFallback: spec.allowLucideFallback === true,
    riskLevel: spec.riskLevel,
    redline: spec.redline === true,
    hostInjection: spec.hostInjection,
    constraint: spec.constraint,
    finding: SLOT_FINDINGS[slotName] || null,
    candidateCount: cands.length,
    candidates: cands,
  };
}

const redlineCount = Object.entries(bySlotCount)
  .filter(([s]) => redlineSlots.has(s))
  .reduce((n, [, c]) => n + c, 0);

const out = {
  $schema: "clawpro-resource-skill-map",
  stage: 9,
  version: new Date().toISOString(),
  generatedAt: new Date().toISOString(),
  note:
    "阶段 9 产物：供 clawpro-portable-design-skill 在当前项目页面生成时稳定选图。" +
    "候选确定性派生自 inventory（已审计 componentSlot / canonicalKey），slot 约束取自阶段 8 " +
    "component-resource-map。registrySample 仅为 skill 可移植身份样例，非候选准入闸门。" +
    "由 build-resource-skill-map.mjs 生成，请勿手改；改口径请改脚本后重跑并跑 check-resource-skill-map.mjs。",
  sources: {
    inventory: "client/src/design-assets/generated/resource-inventory.generated.json",
    componentResourceMap: "client/src/design-assets/manual-overrides/component-resource-map.json",
    governance: "client/src/design-assets/generated/resource-governance.generated.json",
  },
  registrySample: REGISTRY_SAMPLE,
  usageScopeLegend: {
    "current-project-only": "仅当前项目页面层可用（经 webPath / ESM import 引用），不可进共享组件源码、不可跨仓",
    "host-injected": "红线资源（品牌/渠道/头像）：当前项目页面经 canonicalAssets 引用；跨仓由宿主仓注入，禁改色当普通图标",
    "portable-safe": "可移植安全（如 lucide-react 通用图标，不在本资源映射内，由 skill 直接使用）",
  },
  summary: {
    slots: Object.keys(slotsSpec).length,
    candidates: totalCandidates,
    redlineCandidates: redlineCount,
    bySlot: bySlotCount,
  },
  slots: slotsOut,
};

fs.writeFileSync(SKILL_MAP_PATH, JSON.stringify(out, null, 2) + "\n", "utf8");

console.log("[build-resource-skill-map] 完成。");
console.log(`  槽位：${out.summary.slots}  候选合计：${out.summary.candidates}（红线 ${redlineCount}）`);
console.log("  分槽位：" + Object.entries(bySlotCount).map(([s, n]) => `${s} ${n}`).join(" / "));
console.log(`  产物：${path.relative(repoRoot, SKILL_MAP_PATH)}`);

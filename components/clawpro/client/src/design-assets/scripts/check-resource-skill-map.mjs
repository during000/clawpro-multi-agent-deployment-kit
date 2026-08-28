#!/usr/bin/env node
// =============================================================================
// 阶段 9：resource-skill-map 数据校验（check-resource-skill-map.mjs）
// -----------------------------------------------------------------------------
// 落地建设计划 §6.5「第二层：数据保证」+「第三层：检查脚本保证」。
// 校验 client/src/design-assets/resource-skill-map.json 必须满足（任一不符即退出 1）：
//   1. 每个候选 id 必须存在于资源清单 inventory（资源真相）。
//   2. 候选对应 inventory 资源 status 必须为 normal（deprecated/avoid/needs-review/archived 不得入）。
//   3. 候选 id 不得在 governance.removedIds（不引用已删/已治理移除资源）。
//   4. 候选 slot 必须在 component-resource-map.slots 白名单内。
//   5. landing=file 的候选 sourcePath 必须磁盘存在；landing=inline 不校验静态文件。
//   6. 候选必须标记合法 usageScope（current-project-only / host-injected / portable-safe）。
//   7. 红线槽位（redline=true）候选：usageScope 必须 host-injected、redline=true、canonicalKey 非空；
//      非红线候选：usageScope 必须 current-project-only。
//   8. brand-fixed / avatar-like 资源（红线类目）只能落在对应红线槽位（primaryCategory==slot）。
//   9. 候选的 allowLucideFallback / recommendedResourceType 必须与阶段 8 槽位规格一致（防漂移）。
//  10. summary 计数必须与实际候选数一致。
//
// 注（口径修正，用户拍板 A）：registry（icon-registry.example.json）已降格为 skill 的
//   「可移植身份样例」，**不**作为候选准入闸门，故本脚本不校验候选是否在 registry 中
//   approved；候选准入以 inventory 审计字段为准。
//
// 退出码：0=全部通过；1=任一校验失败。
// 用法：node client/src/design-assets/scripts/check-resource-skill-map.mjs
// =============================================================================

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "../../../../");
const DA_DIR = path.join(repoRoot, "client/src/design-assets");
const SKILL_MAP_PATH = path.join(DA_DIR, "resource-skill-map.json");
const INVENTORY_PATH = path.join(DA_DIR, "generated/resource-inventory.generated.json");
const GOV_PATH = path.join(DA_DIR, "generated/resource-governance.generated.json");
const COMPONENT_MAP_PATH = path.join(DA_DIR, "manual-overrides/component-resource-map.json");

const VALID_SCOPES = new Set(["current-project-only", "host-injected", "portable-safe"]);

function readJson(p, label) {
  if (!fs.existsSync(p)) {
    console.error(`[check-resource-skill-map] 缺少输入：${label} -> ${path.relative(repoRoot, p)}`);
    process.exit(1);
  }
  return JSON.parse(fs.readFileSync(p, "utf8"));
}

const skillMap = readJson(SKILL_MAP_PATH, "resource-skill-map.json");
const inventory = readJson(INVENTORY_PATH, "inventory");
const governance = readJson(GOV_PATH, "governance");
const componentMap = readJson(COMPONENT_MAP_PATH, "component-resource-map.json");

const itemById = new Map((inventory.items || []).map((i) => [i.id, i]));
const removedIds = new Set(governance.removedIds || []);
const slotsSpec = componentMap.slots || {};
const slotWhitelist = new Set(Object.keys(slotsSpec));
const redlineSlots = new Set(
  Object.entries(slotsSpec).filter(([, s]) => s.redline === true).map(([k]) => k),
);

const errors = [];
let checked = 0;
const countBySlot = {};

for (const [slotName, slot] of Object.entries(skillMap.slots || {})) {
  const cands = slot.candidates || [];
  countBySlot[slotName] = cands.length;

  // slot 白名单
  if (!slotWhitelist.has(slotName)) {
    errors.push(`槽位不在白名单：${slotName}`);
    continue;
  }
  const isRedlineSlot = redlineSlots.has(slotName);
  const spec = slotsSpec[slotName];

  for (const c of cands) {
    checked++;
    const where = `[${slotName}] ${c.id} (${c.displayName})`;

    // 1. id 存在于 inventory
    const item = itemById.get(c.id);
    if (!item) {
      errors.push(`候选 id 不在 inventory：${where}`);
      continue;
    }
    const cls = item.classification || {};

    // 2. status=normal
    if (cls.status !== "normal") {
      errors.push(`候选资源非 normal：${where} status=${cls.status}`);
    }
    if (cls.excludeFromLibrary || cls.archived) {
      errors.push(`候选资源被排除/归档：${where}`);
    }

    // 3. 不在 removedIds
    if (removedIds.has(c.id)) {
      errors.push(`候选引用了已治理移除资源：${where}`);
    }

    // 4. slot 合法（已在外层判定白名单）
    if (c.slot !== slotName) {
      errors.push(`候选 slot 字段与所在槽位不一致：${where} slot=${c.slot}`);
    }

    // 5. landing 形态磁盘校验
    if (c.landing === "file") {
      if (!c.sourcePath) {
        errors.push(`file 候选缺 sourcePath：${where}`);
      } else if (!fs.existsSync(path.join(repoRoot, c.sourcePath))) {
        errors.push(`候选磁盘文件不存在：${where} -> ${c.sourcePath}`);
      }
    } else if (c.landing !== "inline") {
      errors.push(`候选 landing 非法（应为 file/inline）：${where} landing=${c.landing}`);
    }

    // 6. usageScope 合法
    if (!VALID_SCOPES.has(c.usageScope)) {
      errors.push(`候选 usageScope 非法：${where} usageScope=${c.usageScope}`);
    }

    // 7. 红线 / 非红线 scope 与标记一致
    if (isRedlineSlot) {
      if (c.usageScope !== "host-injected") {
        errors.push(`红线候选 usageScope 应为 host-injected：${where} 实为 ${c.usageScope}`);
      }
      if (c.redline !== true) {
        errors.push(`红线候选 redline 应为 true：${where}`);
      }
      if (!c.canonicalKey) {
        errors.push(`红线候选必须带 canonicalKey：${where}`);
      }
    } else {
      if (c.usageScope !== "current-project-only") {
        errors.push(`非红线候选 usageScope 应为 current-project-only：${where} 实为 ${c.usageScope}`);
      }
    }

    // 8. 红线类目资源只能落对应红线槽位（primaryCategory == slot）
    const redlineCategories = new Set(["channel-icon", "brand-logo", "agent-avatar"]);
    if (redlineCategories.has(cls.primaryCategory) && cls.primaryCategory !== slotName) {
      errors.push(`红线类目资源落错槽位：${where} category=${cls.primaryCategory} slot=${slotName}`);
    }
    // 反向：非红线槽位不得混入红线类目资源
    if (!isRedlineSlot && redlineCategories.has(cls.primaryCategory)) {
      errors.push(`非红线槽位混入红线类目资源：${where} category=${cls.primaryCategory}`);
    }

    // 9. slot 约束与阶段 8 规格一致（防漂移）
    if (c.allowLucideFallback !== (spec.allowLucideFallback === true)) {
      errors.push(`候选 allowLucideFallback 与槽位规格不一致：${where}`);
    }
    if (c.recommendedResourceType !== spec.recommendedResourceType) {
      errors.push(`候选 recommendedResourceType 与槽位规格不一致：${where}`);
    }
  }
}

// 10. summary 计数一致
const sumBySlot = (skillMap.summary && skillMap.summary.bySlot) || {};
for (const [slotName, n] of Object.entries(countBySlot)) {
  if (sumBySlot[slotName] !== n) {
    errors.push(`summary.bySlot 计数不一致：${slotName} 标 ${sumBySlot[slotName]} 实 ${n}`);
  }
}
const totalActual = Object.values(countBySlot).reduce((a, b) => a + b, 0);
if (skillMap.summary && skillMap.summary.candidates !== totalActual) {
  errors.push(`summary.candidates 计数不一致：标 ${skillMap.summary.candidates} 实 ${totalActual}`);
}

if (errors.length) {
  console.error(`[check-resource-skill-map] 校验失败（${errors.length} 项）：`);
  for (const e of errors) console.error("  - " + e);
  process.exit(1);
}

console.log("[check-resource-skill-map] 校验通过。");
console.log(`  候选检查：${checked}  槽位：${Object.keys(countBySlot).length}`);
console.log("  分槽位：" + Object.entries(countBySlot).map(([s, n]) => `${s} ${n}`).join(" / "));

// =============================================================================
// 阶段 6：canonical 资源入口建设（build-canonical-assets.mjs）
// -----------------------------------------------------------------------------
// 定位（严格遵守建设计划 §9 阶段 6 / §5.3 canonical 入口 / §六 与 skill 连接 / §八 安全）：
//   - 从治理后现状中筛选「高频、已确认 canonical、运行时可服务」的资源，建立统一入口
//     client/src/design-assets/canonical-assets.ts，使其具备「一改多处生效」基础能力。
//   - 仅纳入业务专属、已确认 status=normal 的资源（品牌 Logo / 渠道图标 / Agent 头像）。
//   - 回填 inventory 的 canonical 字段、governance 的 canonical 接入状态，并向治理报告
//     追加「canonical 接入记录」段落。
//
// 纪律（务必遵守，本阶段以阶段 2~5 真实产出为准、不臆测）：
//   1. **只收已确认 canonical**：status 必须为 normal；needs-review / duplicate / avoid /
//      deprecated 一律不进入口（与 §5.4「skill-map 不含 needs-review」一致）。多处使用但
//      仍 needs-review 的 /icon 业务 SVG 本阶段不纳入，待其转 normal 后再说。
//   2. **红线不合并**：brand-logo / channel-icon / agent-avatar 各自一个 key，wecom 与
//      wecom-app 字节一致（dup-074 红线）也分列两个 key，绝不在入口层归并。
//   3. **运行时可服务**：只收有 webPath（public）的资源作为字符串入口值，零构建/类型风险。
//   4. **不动组件源码、不动 skill（阶段 9）、不做全量迁移**：页面层接入只做安全示范，
//      组件源码内引用只记录、不替换。
//   5. **校验优先、失败即停**：SPEC 中每个条目必须能在 inventory 命中、未被阶段 5 删除、
//      磁盘存在、类目与 status 符合预期，任一不符立即报错退出，不静默放过。
//   6. **幂等可重复**：重跑先清空 inventory 全量 canonical 字段再写入；报告段落按标记替换。
//
// 运行：node client/src/design-assets/scripts/build-canonical-assets.mjs
//
// 输入（阶段 2~5 产物，只读）：
//   client/src/design-assets/generated/resource-inventory.generated.json
//   client/src/design-assets/generated/resource-usage.generated.json
//   client/src/design-assets/generated/resource-duplicates.generated.json
//   client/src/design-assets/generated/resource-governance.generated.json
// 产物：
//   client/src/design-assets/canonical-assets.ts                            （新建/覆盖，入口本体）
//   client/src/design-assets/generated/resource-inventory.generated.json    （回填 canonical 字段）
//   client/src/design-assets/generated/resource-governance.generated.json   （回填 canonical 接入状态）
//   docs/resource-governance-report.md                                      （追加 canonical 接入记录段）
// =============================================================================

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "../../../../");
const GEN_DIR = path.join(repoRoot, "client/src/design-assets/generated");
const INVENTORY_PATH = path.join(GEN_DIR, "resource-inventory.generated.json");
const USAGE_PATH = path.join(GEN_DIR, "resource-usage.generated.json");
const DUP_PATH = path.join(GEN_DIR, "resource-duplicates.generated.json");
const GOV_PATH = path.join(GEN_DIR, "resource-governance.generated.json");
const CANONICAL_TS_PATH = path.join(repoRoot, "client/src/design-assets/canonical-assets.ts");
const REPORT_PATH = path.join(repoRoot, "docs/resource-governance-report.md");
const SRC_ROOT = path.join(repoRoot, "client/src");

// ---------------------------------------------------------------------------
// canonical 入口规格（SPEC）：由阶段 6 据阶段 2~5 真实审计产出人工策展。
// 仅业务专属、已确认 normal、运行时可服务（public webPath）资源。
// 组织维度（brands / channels / avatars）来自治理后存活类目的真实结论，非预设。
// 每条以 webPath 为主键（比带 hash 后缀的 id 更稳定，便于重扫后仍可命中校验）。
// ---------------------------------------------------------------------------
const GROUP_CATEGORY = {
  brands: "brand-logo",
  channels: "channel-icon",
  avatars: "agent-avatar",
};

const CANONICAL_SPEC = {
  brands: {
    // ClawPro 品牌 Logo（brand-fixed，禁改色）。public 侧栏 Logo 为运行时可服务资源、唯一 canonical。
    // 注：原 src 侧 @/assets/topnav/clawpro-logo.svg（横版带文字 logo，内容不同）全仓无代码引用
    // （TopNav 实际用 /landing-assets/60.svg），已于 2026-06-14 清理删除，不再存在第二物理副本。
    clawproLogo: "/assets/admin-sidebar/clawpro-logo.svg",
  },
  channels: {
    // 第三方渠道图标（brand-fixed，禁改色）。wecom 与 wecomApp 字节一致（dup-074 红线），
    // 仍分列两 key，绝不在入口层归并。
    wechat: "/assets/admin-channel-icons/channel-wechat.svg",
    qq: "/assets/admin-channel-icons/channel-qq.svg",
    wecom: "/assets/admin-channel-icons/channel-wecom.svg",
    wecomApp: "/assets/admin-channel-icons/channel-wecom-app.svg",
    dingtalk: "/assets/admin-channel-icons/channel-dingtalk.svg",
    feishu: "/assets/admin-channel-icons/channel-feishu.svg",
  },
  avatars: {
    // Agent 角色头像（avatar-like，禁改色）。键与 AgentAvatar 组件 ROLE_AVATAR 的角色一一对应。
    default: "/assets/avatars/avatar-default.png",
    designer: "/assets/avatars/avatar-designer.png",
    analyst: "/assets/avatars/avatar-analyst.png",
    creator: "/assets/avatars/avatar-creator.png",
    developer: "/assets/avatars/avatar-developer.png",
    pm: "/assets/avatars/avatar-pm.png",
    operator: "/assets/avatars/avatar-operator.png",
  },
};

// ---------------------------------------------------------------------------
// 读取输入（只读）
// ---------------------------------------------------------------------------
const inventory = JSON.parse(fs.readFileSync(INVENTORY_PATH, "utf8"));
const governance = JSON.parse(fs.readFileSync(GOV_PATH, "utf8"));
JSON.parse(fs.readFileSync(USAGE_PATH, "utf8")); // 存在性校验
JSON.parse(fs.readFileSync(DUP_PATH, "utf8")); // 存在性校验

const removedIds = new Set(governance.removedIds || []);
const itemByWebPath = new Map();
for (const it of inventory.items || []) {
  if (it.webPath) itemByWebPath.set(it.webPath, it);
}

// ---------------------------------------------------------------------------
// 1) 校验 SPEC（失败即停，不静默）
// ---------------------------------------------------------------------------
const errors = [];
/** key 路径 -> 校验通过的 inventory item */
const resolvedEntries = []; // {group, key, webPath, item}

for (const [group, entries] of Object.entries(CANONICAL_SPEC)) {
  const expectCat = GROUP_CATEGORY[group];
  for (const [key, webPath] of Object.entries(entries)) {
    const item = itemByWebPath.get(webPath);
    const where = `${group}.${key} (${webPath})`;
    if (!item) {
      errors.push(`未在 inventory 命中：${where}`);
      continue;
    }
    if (removedIds.has(item.id)) {
      errors.push(`该资源已被阶段 5 删除，不能作为 canonical：${where}`);
      continue;
    }
    const cls = item.classification || {};
    if (cls.primaryCategory !== expectCat) {
      errors.push(`类目不符：${where} 期望 ${expectCat} 实为 ${cls.primaryCategory}`);
      continue;
    }
    if (cls.status !== "normal") {
      errors.push(`status 非 normal（不得入口）：${where} 实为 ${cls.status}`);
      continue;
    }
    // 运行时可服务（public webPath）
    const diskPath = path.join(repoRoot, item.sourcePath || "");
    if (!item.sourcePath || !fs.existsSync(diskPath)) {
      errors.push(`磁盘不存在：${where} -> ${item.sourcePath}`);
      continue;
    }
    resolvedEntries.push({ group, key, webPath, item });
  }
}

if (errors.length) {
  console.error("[build-canonical-assets] SPEC 校验失败，已中止（未写任何产物）：");
  for (const e of errors) console.error("  - " + e);
  process.exit(1);
}

// ---------------------------------------------------------------------------
// 2) 接入检测：扫描 client/src 中对 canonicalAssets.<group>.<key> 的真实引用
//    （只统计页面层/调用处是否已接入入口，组件源码引用同样会被统计为「已有引用」，
//     但本阶段不主动迁移组件源码）
// ---------------------------------------------------------------------------
function walk(dir, acc) {
  for (const name of fs.readdirSync(dir)) {
    if (name === "node_modules" || name === ".git" || name === "generated" || name === "scripts") continue;
    const full = path.join(dir, name);
    const st = fs.statSync(full);
    if (st.isDirectory()) walk(full, acc);
    else if (/\.(ts|tsx|js|jsx)$/.test(name)) acc.push(full);
  }
  return acc;
}
const srcFiles = walk(SRC_ROOT, []);
/** "group.key" -> Set<相对文件路径> */
const consumersByKey = new Map();
for (const f of srcFiles) {
  if (f === CANONICAL_TS_PATH) continue;
  const text = fs.readFileSync(f, "utf8");
  if (!text.includes("canonicalAssets")) continue;
  const rel = path.relative(repoRoot, f);
  for (const { group, key } of resolvedEntries) {
    const needle = `canonicalAssets.${group}.${key}`;
    if (text.includes(needle)) {
      const k = `${group}.${key}`;
      if (!consumersByKey.has(k)) consumersByKey.set(k, new Set());
      consumersByKey.get(k).add(rel);
    }
  }
}

// ---------------------------------------------------------------------------
// 3) 生成 canonical-assets.ts（确定性、按 SPEC 顺序）
// ---------------------------------------------------------------------------
function dupNote(item) {
  const dg = item.classification?.duplicateGroupId;
  return dg ? `，重复组 ${dg}` : "";
}
const lines = [];
lines.push("// =============================================================================");
lines.push("// ClawPro 资源库 · canonical 资源统一入口（自动生成，请勿手改）");
lines.push("// -----------------------------------------------------------------------------");
lines.push("// 由 client/src/design-assets/scripts/build-canonical-assets.mjs 据阶段 2~5 真实");
lines.push("// 审计/治理产出生成（建设计划 §9 阶段 6 / §5.3）。改口径请改脚本后重跑。");
lines.push("//");
lines.push("// 用途与边界（务必遵守）：");
lines.push("//   - 仅供「当前项目页面层 / 页面级非组件代码」使用，提供「一改多处生效」能力：");
lines.push("//     修改此处某 key 的路径，所有 import 该 key 的页面处统一生效。");
lines.push("//   - 禁止在「共享组件源码」中 import 本文件（组件需保持可移植，用 props/lucide/宿主注入）。");
lines.push("//   - 禁止在「开发仓库 / 跨仓页面」中引用本文件或当前项目 /assets 路径。");
lines.push("//   - 入口仅收已确认 normal、业务专属、运行时可服务资源；不含 needs-review 资源。");
lines.push("//   - brand-fixed / avatar-like 资源禁止当普通 UI 图标改色（品牌/渠道/头像红线）。");
lines.push("//");
lines.push(`// 生成时间：${new Date().toISOString()}`);
lines.push("// =============================================================================");
lines.push("");
lines.push("export const canonicalAssets = {");
for (const group of Object.keys(CANONICAL_SPEC)) {
  const groupEntries = resolvedEntries.filter((e) => e.group === group);
  const cat = GROUP_CATEGORY[group];
  lines.push(`  /** ${group}（${cat}） */`);
  lines.push(`  ${group}: {`);
  for (const { key, webPath, item } of groupEntries) {
    lines.push(`    /** ${item.displayName || key}${dupNote(item)} */`);
    lines.push(`    ${key}: ${JSON.stringify(webPath)},`);
  }
  lines.push("  },");
}
lines.push("} as const;");
lines.push("");
lines.push("export type CanonicalAssetGroup = keyof typeof canonicalAssets;");
lines.push("");
fs.writeFileSync(CANONICAL_TS_PATH, lines.join("\n"), "utf8");

// ---------------------------------------------------------------------------
// 4) 回填 inventory 的 canonical 字段（幂等：先全量清空再写入）
// ---------------------------------------------------------------------------
const resolvedByItemId = new Map(resolvedEntries.map((e) => [e.item.id, e]));
let connectedCount = 0;
for (const it of inventory.items || []) {
  const cls = it.classification || (it.classification = {});
  const hit = resolvedByItemId.get(it.id);
  if (hit) {
    const k = `${hit.group}.${hit.key}`;
    const connected = (consumersByKey.get(k)?.size || 0) > 0;
    cls.isCanonical = true;
    cls.canonicalId = k;
    cls.canonicalKey = k;
    cls.connectedToCanonicalEntry = connected;
    if (connected) connectedCount++;
  } else {
    cls.isCanonical = false;
    cls.canonicalId = null;
    cls.canonicalKey = null;
    cls.connectedToCanonicalEntry = false;
  }
}
const byGroup = {};
for (const g of Object.keys(CANONICAL_SPEC)) byGroup[g] = resolvedEntries.filter((e) => e.group === g).length;
inventory.canonicalStage = 6;
inventory.canonicalGeneratedAt = new Date().toISOString();
inventory.canonicalSummary = {
  entryFile: "client/src/design-assets/canonical-assets.ts",
  entries: resolvedEntries.length,
  byGroup,
  connectedToEntry: connectedCount,
  availableNotConnected: resolvedEntries.length - connectedCount,
  note: "阶段 6 回填：canonical 字段为 canonical 入口接入状态；scan/classify/duplicate 基线字段不变。",
};
fs.writeFileSync(INVENTORY_PATH, JSON.stringify(inventory, null, 2) + "\n", "utf8");

// ---------------------------------------------------------------------------
// 5) 回填 governance 的 canonical 接入状态
// ---------------------------------------------------------------------------
const entryRecords = resolvedEntries.map(({ group, key, webPath, item }) => {
  const k = `${group}.${key}`;
  const consumers = [...(consumersByKey.get(k) || [])].sort();
  return {
    key: k,
    group,
    id: item.id,
    path: webPath,
    sourcePath: item.sourcePath,
    category: item.classification?.primaryCategory,
    visualType: item.classification?.visualType,
    status: item.classification?.status,
    duplicateGroupId: item.classification?.duplicateGroupId || null,
    redLine: ["brand-logo", "channel-icon", "agent-avatar"].includes(item.classification?.primaryCategory),
    connectedToCanonicalEntry: consumers.length > 0,
    consumers,
  };
});
const connectedKeys = entryRecords.filter((e) => e.connectedToCanonicalEntry).map((e) => e.key);
const availableKeys = entryRecords.filter((e) => !e.connectedToCanonicalEntry).map((e) => e.key);
const migratedFiles = [...new Set(entryRecords.flatMap((e) => e.consumers))].sort();

governance.canonical = {
  stage: 6,
  generatedAt: new Date().toISOString(),
  entryFile: "client/src/design-assets/canonical-assets.ts",
  note:
    "阶段 6 canonical 入口接入状态。入口只收已确认 normal、业务专属、运行时可服务资源；" +
    "红线资源（品牌/渠道/头像）分列不归并；needs-review 资源不入口。",
  summary: {
    entries: entryRecords.length,
    byGroup,
    connectedToEntry: connectedKeys.length,
    availableNotConnected: availableKeys.length,
  },
  entries: entryRecords,
  adoption: {
    migratedThisStage: migratedFiles,
    connectedKeys,
    availableNotConnectedKeys: availableKeys,
  },
  governedDuplicateRefMigration:
    "无需替换：阶段 5 实际移除的 127 个 A 类冗余副本均无任何引用线索，故无页面层 / 非组件源码引用需迁移到 canonical。",
  oneChangeMultiEffect: {
    capability:
      "已接入 canonicalAssets 入口的 key，修改 canonical-assets.ts 中其路径会统一影响所有 import 处；" +
      "未接入 key 仅建立入口，现有散落引用保持原样（不做全量迁移）。",
    componentSourceNote:
      "AgentAvatar.tsx（ROLE_AVATAR）与 TopNav 等组件源码内的资源引用按共享组件处理，本阶段只记录、不改源码、不引入口。",
  },
};
fs.writeFileSync(GOV_PATH, JSON.stringify(governance, null, 2) + "\n", "utf8");

// ---------------------------------------------------------------------------
// 6) 向治理报告追加「canonical 接入记录」段（按标记幂等替换）
// ---------------------------------------------------------------------------
const START = "<!-- CANONICAL:START -->";
const END = "<!-- CANONICAL:END -->";
const rl = [];
rl.push(START);
rl.push("");
rl.push("## 十、canonical 接入记录（阶段 6）");
rl.push("");
rl.push(`> 本段由 \`client/src/design-assets/scripts/build-canonical-assets.mjs\` 生成（建设计划 §9 阶段 6 / §5.3）。生成时间：${governance.canonical.generatedAt}`);
rl.push("");
rl.push("### 10.1 入口与纳入口径");
rl.push("");
rl.push("- 入口文件：`client/src/design-assets/canonical-assets.ts`（自动生成，仅供当前项目页面层使用）。");
rl.push(`- 纳入 ${entryRecords.length} 个 key（${Object.entries(byGroup).map(([g, n]) => `${g} ${n}`).join(" / ")}）：仅「已确认 normal + 业务专属 + 运行时可服务（public webPath）」资源。`);
rl.push("- **暂不接入入口的高频资源**：多处使用的 `/icon/*.svg` 与 `empty-aiagent.png` 当前 `status=needs-review`（多属 C 类 keep-multiple 组），按纪律不作为「已确认 canonical」，待其经设计确认转 normal 后再纳入。");
rl.push("- **红线不归并**：`channel-wecom` 与 `channel-wecom-app` 字节一致（dup-074 红线），入口层仍分列 `channels.wecom` / `channels.wecomApp` 两 key，不合并。");
rl.push("- **品牌 Logo 第二物理副本（已清理）**：原 `@/assets/topnav/clawpro-logo.svg`（src，横版带文字 logo，内容不同；全仓无代码引用，TopNav 实际用 `/landing-assets/60.svg`）已于 2026-06-14 删除，现仅保留唯一 canonical `/assets/admin-sidebar/clawpro-logo.svg`。");
rl.push("");
rl.push("### 10.2 接入状态");
rl.push("");
rl.push("| key | 资源 | 类目 | 重复组 | 是否已接入入口 | 接入处 |");
rl.push("|---|---|---|---|---|---|");
for (const e of entryRecords) {
  rl.push(`| \`${e.key}\` | \`${e.path}\` | ${e.category} | ${e.duplicateGroupId || "-"} | ${e.connectedToCanonicalEntry ? "是" : "否（仅建入口）"} | ${e.consumers.length ? e.consumers.map((c) => "`" + c + "`").join("<br/>") : "-"} |`);
}
rl.push("");
rl.push("### 10.3 一改多处生效与迁移说明");
rl.push("");
rl.push(`- **已接入**（${connectedKeys.length} key）：${connectedKeys.length ? connectedKeys.map((k) => "`" + k + "`").join("、") : "无"}。修改 \`canonical-assets.ts\` 中其路径会统一影响所有 import 处。`);
rl.push(`- **仅建入口、暂未接入**（${availableKeys.length} key）：现有散落引用保持原样，可由页面作者按需增量接入；**不做全量迁移**（边界）。`);
rl.push(`- **页面层接入示范**：${migratedFiles.length ? migratedFiles.map((c) => "`" + c + "`").join("、") + " 已改用 `canonicalAssets.channels.*`" : "无"}。`);
rl.push("- **已治理重复资源的引用迁移**：无需替换——阶段 5 实际移除的 127 个 A 类冗余副本均无任何引用线索，故无页面层 / 非组件源码引用需迁移。");
rl.push("- **组件源码**：`AgentAvatar.tsx`（`ROLE_AVATAR`）、`TopNav` 等组件内资源引用按共享组件处理，本阶段只记录风险、不改源码、不引入口（组件迁移如确有必要须按阶段 8 单独立项评估）。");
rl.push("");
rl.push(END);

const canonicalBlock = rl.join("\n");
let report = fs.readFileSync(REPORT_PATH, "utf8");
if (report.includes(START) && report.includes(END)) {
  report = report.replace(new RegExp(`${START}[\\s\\S]*${END}`), canonicalBlock);
} else {
  report = report.replace(/\s*$/, "\n\n") + canonicalBlock + "\n";
}
fs.writeFileSync(REPORT_PATH, report, "utf8");

// ---------------------------------------------------------------------------
console.log("[build-canonical-assets] 完成。");
console.log(`  入口条目：${resolvedEntries.length}（${Object.entries(byGroup).map(([g, n]) => `${g} ${n}`).join(" / ")}）`);
console.log(`  已接入入口：${connectedKeys.length}  仅建入口：${availableKeys.length}`);
console.log(`  接入处文件：${migratedFiles.length ? migratedFiles.join(", ") : "（无）"}`);
console.log("  产物：canonical-assets.ts / inventory(canonical 字段) / governance(canonical) / 报告第十节");

// =============================================================================
// 阶段 4：重复资源审查与治理分级（detect-duplicates.mjs）
// -----------------------------------------------------------------------------
// 定位（严格遵守建设计划 §9 阶段 4 / §四 重复资源治理方案）：
//   - 识别重复资源（文件 hash / SVG 归一化 hash / 文件名相似），结合使用次数与
//     路径稳定性给出 canonical 建议，并把每组分到 A / B / C 三级。
//   - **只识别、只分级、只建议；不删除、不替换、不改资源文件**（实际治理是阶段 5）。
//   - **不建立 canonical 入口**（canonical-assets.ts 是阶段 6）；本阶段只产「canonical 建议」。
//
// 纪律（务必遵守）：
//   1. 只读资源文件（读字节算 hash、读 SVG 文本做归一化），不修改/删除任何资源。
//   2. 不动 skill（非阶段 9）。
//   3. **红线**：品牌 Logo / 渠道图标 / Agent 头像（primaryCategory ∈
//      {brand-logo,channel-icon,agent-avatar} 或 visualType ∈ {brand-fixed,avatar-like}）
//      禁止仅凭 hash 自动归并，一律进 C 类人工确认。
//   4. **registry 事实源不可删**：root `icon/` 中 registry 已登记的成员即便不是
//      canonical，也标 keep-registry-source，永不进入删除候选。
//   5. **usageCount=0 ≠ 可删**：删除候选必须同时满足「无静态引用 + 非目录级动态引用
//      (dynamicDirReferenced) + 不在 unresolvedRefs 的 resolved 命中」。
//   6. 幂等、可重复执行；流水线顺序固定：scan-resources → classify-resources →
//      detect-duplicates（本脚本会回填 inventory 的 classification.duplicate* 字段，
//      重跑 classify 会清空这些字段，需再跑本脚本）。
//   7. **人工确认闭环（只读事实源）**：读取 manual-overrides/duplicate-review.json，对其中已 confirmed
//      的重复组（按成员 id 集合匹配）置 reviewStatus=confirmed、成员动作 needs-review→reviewed-keep，
//      给 C 类「待人工确认」一个确认完成出口；未登记者保持 pending。只读、不删文件、幂等。
//
// 运行：node client/src/design-assets/scripts/detect-duplicates.mjs
// 产物：client/src/design-assets/generated/resource-duplicates.generated.json
//       并回填 resource-inventory.generated.json 的 classification.duplicate* 字段
// =============================================================================

import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "../../../../");
const GEN_DIR = path.join(repoRoot, "client/src/design-assets/generated");
const INVENTORY_PATH = path.join(GEN_DIR, "resource-inventory.generated.json");
const USAGE_PATH = path.join(GEN_DIR, "resource-usage.generated.json");
const DUP_PATH = path.join(GEN_DIR, "resource-duplicates.generated.json");
// 人工事实源：重复组确认裁决（C 类闭环出口；仅登记的成员集合生效，绝不臆测）
const REVIEW_OVERRIDES_PATH = path.join(
  repoRoot,
  "client/src/design-assets/manual-overrides/duplicate-review.json"
);

// ---------------------------------------------------------------------------
// 读取输入
// ---------------------------------------------------------------------------
const inventory = JSON.parse(fs.readFileSync(INVENTORY_PATH, "utf8"));
const usage = JSON.parse(fs.readFileSync(USAGE_PATH, "utf8"));

// 重复组人工确认：key = 组内成员 id 的有序集合（join "|"） -> confirmation
const reviewByKey = new Map();
try {
  const rv = JSON.parse(fs.readFileSync(REVIEW_OVERRIDES_PATH, "utf8"));
  for (const c of rv.confirmations || []) {
    if (c && Array.isArray(c.memberIds) && c.memberIds.length >= 2) {
      reviewByKey.set(c.memberIds.slice().sort().join("|"), c);
    }
  }
} catch {
  // 无确认事实源时，所有 C 组保持待人工确认（pending）
}

// 「被 unresolvedRefs 命中」的资源路径集合（动态/无法静态解析但确有指向的引用，纳入安全集）
const referencedViaUnresolved = new Set(
  (usage.unresolvedRefs || []).filter((u) => u.resolved).map((u) => u.resolved)
);

// 红线类目 / 视觉类型（禁止仅凭 hash 自动归并）
const RED_LINE_CATEGORIES = new Set(["brand-logo", "channel-icon", "agent-avatar"]);
const RED_LINE_VISUAL = new Set(["brand-fixed", "avatar-like"]);

// ---------------------------------------------------------------------------
// 工具
// ---------------------------------------------------------------------------
function sha256(buf) {
  return crypto.createHash("sha256").update(buf).digest("hex");
}

// SVG 归一化：去注释 / 去 width·height·id·class 等格式差异属性 / 折叠空白，
// 用于识别「内容相同但格式不同」的 SVG（与文件 hash 互补）。仅用于比对，不写回文件。
function normalizeSvg(text) {
  return text
    .replace(/<!--[\s\S]*?-->/g, "")
    .replace(/\b(width|height|id|class|data-name)\s*=\s*["'][^"']*["']/gi, "")
    .replace(/\s+/g, " ")
    .replace(/>\s+</g, "><")
    .trim();
}

// 文件名归一化（用于文件名相似度疑似重复）：小写、去扩展名、去 copy/副本/编号后缀与括号编号
function normalizeName(fileName) {
  return (fileName || "")
    .toLowerCase()
    .replace(/\.[^.]+$/, "")
    .replace(/[（(]\s*\d+\s*[）)]/g, "")
    .replace(/[\s._-]*(copy|副本|备份)$/g, "")
    .replace(/[\s._-]+\d+$/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

const REDUNDANT_SUFFIX_RE = /([ _-]\d+|[（(]\s*\d+\s*[）)]|copy|副本|备份)\s*$/i;

function isReferenced(m) {
  return (
    (m.usageCount || 0) > 0 ||
    m.dynamicDirReferenced === true ||
    referencedViaUnresolved.has(m.sourcePath)
  );
}

function isRegistrySource(m) {
  return Boolean(m.classification && m.classification.registry && m.classification.registry.registered);
}

function isServable(m) {
  return Boolean(m.webPath || m.importPath);
}

function isRedLine(m) {
  const c = m.classification || {};
  return RED_LINE_CATEGORIES.has(c.primaryCategory) || RED_LINE_VISUAL.has(c.visualType);
}

// canonical 建议打分：优先「可服务 + 被引用」（运行时可用且确实在用），
// registry 仅作为身份加成、低于可服务/被引用。冗余后缀名（“ 2”/copy）轻微降权，避免成为 canonical。
function canonicalScore(m) {
  let s = 0;
  if (isReferenced(m)) s += 1_000_000;
  s += (m.usageCount || 0) * 10_000;
  if (m.dynamicDirReferenced) s += 50_000;
  if (isServable(m)) s += 200_000;
  if (m.scanScope === "include") s += 100_000;
  if (isRegistrySource(m)) s += 20_000;
  s += m.source === "src" ? 300 : m.source === "public" ? 200 : m.source === "root-assets" ? 50 : 0;
  if (REDUNDANT_SUFFIX_RE.test(m.fileName || "")) s -= 1_000;
  return s;
}

// 成员稳定排序：score 降序 → 冗余后缀靠后 → 路径短优先 → 字典序
function sortMembersForCanonical(members) {
  return members.slice().sort((a, b) => {
    const sb = canonicalScore(b) - canonicalScore(a);
    if (sb) return sb;
    const ra = REDUNDANT_SUFFIX_RE.test(a.fileName || "") ? 1 : 0;
    const rb = REDUNDANT_SUFFIX_RE.test(b.fileName || "") ? 1 : 0;
    if (ra !== rb) return ra - rb;
    const la = (a.sourcePath || "").length - (b.sourcePath || "").length;
    if (la) return la;
    return (a.sourcePath || "").localeCompare(b.sourcePath || "");
  });
}

// ---------------------------------------------------------------------------
// 1) 读取全部文件型资源，计算 hash
// ---------------------------------------------------------------------------
const fileItems = inventory.items.filter(
  (it) => (it.type === "svg" || it.type === "png") && it.sourcePath
);

const hashInfo = new Map(); // id -> { fileHash, normHash }
let readFail = 0;
for (const it of fileItems) {
  let buf;
  try {
    buf = fs.readFileSync(path.join(repoRoot, it.sourcePath));
  } catch {
    readFail++;
    continue;
  }
  const fileHash = sha256(buf);
  let normHash = null;
  if (it.type === "svg") {
    try {
      normHash = sha256(Buffer.from(normalizeSvg(buf.toString("utf8")), "utf8"));
    } catch {
      normHash = null;
    }
  }
  hashInfo.set(it.id, { fileHash, normHash });
}

// ---------------------------------------------------------------------------
// 2) 内容重复组织：SVG 用归一化 hash（涵盖字节相同 + 格式差异），PNG 用文件 hash
//    contentSig: "n:<normHash>" (svg) | "f:<fileHash>" (png)
// ---------------------------------------------------------------------------
const bySig = new Map();
for (const it of fileItems) {
  const h = hashInfo.get(it.id);
  if (!h) continue;
  const sig = it.type === "svg" && h.normHash ? `n:${h.normHash}` : `f:${h.fileHash}`;
  if (!bySig.has(sig)) bySig.set(sig, []);
  bySig.get(sig).push(it);
}

const contentGroups = [];
const groupedIds = new Set();
for (const [sig, members] of bySig) {
  if (members.length < 2) continue;
  const fileHashes = new Set(members.map((m) => hashInfo.get(m.id).fileHash));
  const method = fileHashes.size > 1 ? "svg-normalized-hash" : "exact-file-hash";
  contentGroups.push({ sig, members, method });
  for (const m of members) groupedIds.add(m.id);
}

// ---------------------------------------------------------------------------
// 3) 文件名相似度疑似重复（C 类）：仅在 include 范围、未进入内容重复组、
//    归一化名非纯数字、且跨 >=2 个不同目录时成组（避免 Figma 数字命名噪声）
// ---------------------------------------------------------------------------
const byName = new Map();
for (const it of fileItems) {
  if (groupedIds.has(it.id)) continue;
  if (it.scanScope !== "include") continue;
  const nname = normalizeName(it.fileName);
  if (!nname || /^\d+$/.test(nname)) continue;
  if (!byName.has(nname)) byName.set(nname, []);
  byName.get(nname).push(it);
}
const nameGroups = [];
for (const [nname, members] of byName) {
  if (members.length < 2) continue;
  const dirs = new Set(members.map((m) => m.sourceDir));
  if (dirs.size < 2) continue; // 仅跨目录的同名才算疑似副本
  nameGroups.push({ sig: `name:${nname}`, members, method: "filename-similar" });
}

// ---------------------------------------------------------------------------
// 4) 分级 + canonical 建议 + 成员角色
// ---------------------------------------------------------------------------
// 人工确认闭环：若一个 C 类重复组的成员集合已在 duplicate-review.json 登记并 confirmed，
// 则把它从「待人工确认」推进到「已人工确认」——置 reviewStatus=confirmed，成员动作
// needs-review → reviewed-keep，并把确认说明并入 reasons。未登记者保持 pending（待人工确认）。
// 只读事实源、不改资源文件、幂等。
function applyReviewOverride(group) {
  const key = (group.members || []).map((m) => m.id).slice().sort().join("|");
  const conf = reviewByKey.get(key);
  if (conf && conf.confirmed === true) {
    group.reviewStatus = "confirmed";
    group.reviewDecision = conf.decision || null;
    group.reviewNote = conf.note || null;
    group.reviewedBy = conf.confirmedBy || null;
    group.reviewedAt = conf.confirmedAt || null;
    const by = conf.confirmedBy || "人工";
    const at = conf.confirmedAt ? ` @ ${conf.confirmedAt}` : "";
    group.reasons = [
      ...(group.reasons || []),
      `已人工确认（${by}${at}）：${conf.note || conf.decision || "保留，不归并"}`,
    ];
    for (const m of group.members || []) {
      if (m.action === "needs-review") m.action = "reviewed-keep";
      if (m.role === "review") m.role = "reviewed";
    }
  } else {
    group.reviewStatus = "pending";
  }
  return group;
}

function gradeGroup(raw) {
  const members = sortMembersForCanonical(raw.members);
  const reasons = [];
  const redLine = members.some(isRedLine);

  // 红线：品牌/渠道/头像 → 一律 C，不归并、不给删除/替换动作
  if (redLine) {
    reasons.push("命中红线（品牌 Logo / 渠道图标 / Agent 头像）：禁止仅凭 hash 自动归并，需人工确认是否同一资源");
    return applyReviewOverride({
      grade: "C",
      method: raw.method,
      redLine: true,
      canonicalSuggestionId: null,
      registrySourceIds: members.filter(isRegistrySource).map((m) => m.id),
      members: members.map((m) => ({
        id: m.id,
        sourcePath: m.sourcePath,
        scanScope: m.scanScope,
        type: m.type,
        bytes: m.bytes,
        usageCount: m.usageCount,
        dynamicDirReferenced: m.dynamicDirReferenced || false,
        primaryCategory: m.classification?.primaryCategory,
        visualType: m.classification?.visualType,
        role: "review",
        action: "needs-review",
      })),
      reasons,
    });
  }

  // 文件名相似（疑似，非内容确同）→ C 人工确认
  if (raw.method === "filename-similar") {
    reasons.push("文件名归一化后跨目录同名（疑似重复），但内容不一定相同：人工确认是否同一资源");
    return applyReviewOverride({
      grade: "C",
      method: raw.method,
      redLine: false,
      canonicalSuggestionId: null,
      registrySourceIds: members.filter(isRegistrySource).map((m) => m.id),
      members: members.map((m) => ({
        id: m.id,
        sourcePath: m.sourcePath,
        scanScope: m.scanScope,
        type: m.type,
        bytes: m.bytes,
        usageCount: m.usageCount,
        dynamicDirReferenced: m.dynamicDirReferenced || false,
        primaryCategory: m.classification?.primaryCategory,
        visualType: m.classification?.visualType,
        role: "review",
        action: "needs-review",
      })),
      reasons,
    });
  }

  // 内容确同（exact / normalized）：选 canonical + 逐成员定角色
  const canonical = members[0];
  const outMembers = [];
  let hasReplace = false;
  let hasRemove = false;
  for (const m of members) {
    let role;
    let action;
    if (m.id === canonical.id) {
      role = "canonical-suggested";
      action = "keep";
    } else if (isRegistrySource(m)) {
      role = "keep-registry-source";
      action = "keep"; // registry 事实源，永不删
    } else if (isReferenced(m)) {
      role = "duplicate";
      action = "replace-refs-then-remove"; // 有引用：先把引用替换到 canonical 再移除（阶段 5）
      hasReplace = true;
    } else {
      role = "duplicate";
      action = "remove-redundant"; // 无任何引用线索的冗余副本：阶段 5 可删
      hasRemove = true;
    }
    outMembers.push({
      id: m.id,
      sourcePath: m.sourcePath,
      scanScope: m.scanScope,
      type: m.type,
      bytes: m.bytes,
      usageCount: m.usageCount,
      dynamicDirReferenced: m.dynamicDirReferenced || false,
      byteIdenticalToCanonical:
        hashInfo.get(m.id).fileHash === hashInfo.get(canonical.id).fileHash,
      primaryCategory: m.classification?.primaryCategory,
      visualType: m.classification?.visualType,
      role,
      action,
    });
  }

  let grade;
  if (hasReplace) {
    grade = "B";
    reasons.push("内容完全一致且存在业务引用：阶段 5 需小范围替换引用到 canonical 后再移除副本");
  } else if (hasRemove) {
    grade = "A";
    reasons.push("内容完全一致且冗余副本无任何引用线索：阶段 5 可作为自动清理候选删除（保留 canonical）");
  } else {
    grade = "C";
    reasons.push("除 canonical 外仅剩 registry 事实源副本（root icon/）：属有意双存在（事实源 + 运行时），是否保留多份待人工确认");
  }
  if (raw.method === "svg-normalized-hash") {
    reasons.push("SVG 归一化 hash 一致但字节不同（格式差异），canonical 取可服务/被引用者");
  }

  return applyReviewOverride({
    grade,
    method: raw.method,
    redLine: false,
    canonicalSuggestionId: canonical.id,
    canonicalSuggestionPath: canonical.sourcePath,
    canonicalReason: canonicalReasonText(canonical),
    registrySourceIds: members.filter((m) => isRegistrySource(m)).map((m) => m.id),
    members: outMembers,
    reasons,
  });
}

function canonicalReasonText(m) {
  const bits = [];
  if (isReferenced(m)) bits.push(`被引用(usage=${m.usageCount}${m.dynamicDirReferenced ? "+目录级" : ""})`);
  if (isServable(m)) bits.push("可服务(webPath/importPath)");
  if (m.scanScope === "include") bits.push("include 范围");
  if (isRegistrySource(m)) bits.push("registry 已登记");
  return bits.length ? bits.join("、") : "组内路径最稳定者";
}

const allRaw = [...contentGroups, ...nameGroups];
const graded = allRaw.map(gradeGroup);

// 稳定排序：grade(A→B→C) → 成员数降序 → canonical 路径 → sig
const GRADE_ORDER = { A: 0, B: 1, C: 2 };
graded.sort((a, b) => {
  const g = GRADE_ORDER[a.grade] - GRADE_ORDER[b.grade];
  if (g) return g;
  const sz = b.members.length - a.members.length;
  if (sz) return sz;
  return (a.canonicalSuggestionPath || a.members[0]?.sourcePath || "").localeCompare(
    b.canonicalSuggestionPath || b.members[0]?.sourcePath || ""
  );
});

// 分配稳定 group id
graded.forEach((g, i) => {
  g.id = `dup-${String(i + 1).padStart(3, "0")}`;
});

// ---------------------------------------------------------------------------
// 5) 回填 inventory：classification.duplicateGroupId / duplicateRole / duplicateGrade
//    （不改 status，治理结果状态属阶段 5）
// ---------------------------------------------------------------------------
const itemById = new Map(inventory.items.map((it) => [it.id, it]));
// 先清空旧的重复标记（保证幂等）
for (const it of inventory.items) {
  if (it.classification) {
    it.classification.duplicateGroupId = null;
    it.classification.duplicateRole = null;
    it.classification.duplicateGrade = null;
  }
}
for (const g of graded) {
  for (const m of g.members) {
    const it = itemById.get(m.id);
    if (!it || !it.classification) continue;
    it.classification.duplicateGroupId = g.id;
    it.classification.duplicateRole = m.role;
    it.classification.duplicateGrade = g.grade;
  }
}

// ---------------------------------------------------------------------------
// 6) 统计与输出
// ---------------------------------------------------------------------------
function countBy(arr, fn) {
  const o = {};
  for (const x of arr) {
    const k = fn(x);
    o[k] = (o[k] || 0) + 1;
  }
  return o;
}

const membersInGroups = graded.reduce((n, g) => n + g.members.length, 0);
const removeCandidates = graded.flatMap((g) =>
  g.members.filter((m) => m.action === "remove-redundant")
);
const replaceCandidates = graded.flatMap((g) =>
  g.members.filter((m) => m.action === "replace-refs-then-remove")
);

const summary = {
  fileResourcesHashed: hashInfo.size,
  readFail,
  groups: graded.length,
  byGrade: countBy(graded, (g) => g.grade),
  byMethod: countBy(graded, (g) => g.method),
  redLineGroups: graded.filter((g) => g.redLine).length,
  membersInGroups,
  removeRedundantCandidates: removeCandidates.length,
  replaceRefsCandidates: replaceCandidates.length,
  keepRegistrySourceMembers: graded.flatMap((g) => g.members).filter((m) => m.role === "keep-registry-source").length,
  canonicalSuggestions: graded.filter((g) => g.canonicalSuggestionId).length,
  reviewConfirmedGroups: graded.filter((g) => g.reviewStatus === "confirmed").length,
  reviewPendingGroups: graded.filter((g) => g.grade === "C" && g.reviewStatus !== "confirmed").length,
};

const gradeLists = { A: [], B: [], C: [] };
for (const g of graded) gradeLists[g.grade].push(g.id);

const canonicalSuggestions = graded
  .filter((g) => g.canonicalSuggestionId)
  .map((g) => ({
    groupId: g.id,
    grade: g.grade,
    method: g.method,
    canonicalId: g.canonicalSuggestionId,
    canonicalPath: g.canonicalSuggestionPath,
    canonicalReason: g.canonicalReason,
    registrySourceIds: g.registrySourceIds,
    duplicateCount: g.members.length - 1,
  }));

const out = {
  $schema: "clawpro-resource-duplicates",
  stage: 4,
  generatedAt: new Date().toISOString(),
  note:
    "阶段 4 重复审查与治理分级：只识别/分级/建议，不删除不替换不改文件（实际治理见阶段 5）。" +
    "红线（品牌/渠道/头像）一律 C 不归并；registry 事实源标 keep-registry-source 永不删；" +
    "usageCount=0 仅在同时无目录级动态引用与 unresolved 命中时才作为删除候选。",
  detectionMethods: [
    "exact-file-hash：文件字节 SHA-256 完全一致",
    "svg-normalized-hash：SVG 去格式差异后内容一致（本仓库实测为 0 组，逻辑保留）",
    "filename-similar：include 范围、跨目录同名、非纯数字命名的疑似重复（C 类人工确认）",
  ],
  grading: {
    A: "内容完全一致 + 冗余副本无任何引用线索 → 阶段 5 可自动清理候选（保留 canonical）",
    B: "内容完全一致 + 存在业务引用 → 阶段 5 小范围替换引用到 canonical 后再移除",
    C: "红线资源 / 文件名疑似 / 仅余 registry 事实源副本 → 人工确认，不自动处理",
  },
  summary,
  gradeLists,
  canonicalSuggestions,
  groups: graded,
};

fs.writeFileSync(DUP_PATH, JSON.stringify(out, null, 2) + "\n", "utf8");

// 回填后的 inventory（更新 stage / note / 重复摘要）
inventory.stage = 4;
inventory.duplicatesDetectedAt = out.generatedAt;
inventory.duplicateSummary = {
  groups: summary.groups,
  byGrade: summary.byGrade,
  membersInGroups: summary.membersInGroups,
  removeRedundantCandidates: summary.removeRedundantCandidates,
  replaceRefsCandidates: summary.replaceRefsCandidates,
  keepRegistrySourceMembers: summary.keepRegistrySourceMembers,
};
inventory.note =
  "阶段 4 已回填 classification.duplicate*（duplicateGroupId/duplicateRole/duplicateGrade）。" +
  "重复仅识别与分级，未做实际治理（删除/替换属阶段 5），canonicalId 仍留阶段 6。";
fs.writeFileSync(INVENTORY_PATH, JSON.stringify(inventory, null, 2) + "\n", "utf8");

// ---------------------------------------------------------------------------
// 控制台摘要
// ---------------------------------------------------------------------------
console.log("== 阶段 4 重复审查与治理分级完成 ==");
console.log("已 hash 文件资源:", summary.fileResourcesHashed, "| 读取失败:", summary.readFail);
console.log("重复组:", summary.groups, "| 按分级:", JSON.stringify(summary.byGrade), "| 按方法:", JSON.stringify(summary.byMethod));
console.log("红线组(品牌/渠道/头像,不归并):", summary.redLineGroups);
console.log("C 类人工确认: 已确认", summary.reviewConfirmedGroups, "| 待确认", summary.reviewPendingGroups);
console.log("组内成员:", summary.membersInGroups, "| A 类可清理候选:", summary.removeRedundantCandidates, "| B 类需替换引用:", summary.replaceRefsCandidates, "| registry 事实源保留:", summary.keepRegistrySourceMembers);
console.log("canonical 建议组数:", summary.canonicalSuggestions);
console.log("产物:", path.relative(repoRoot, DUP_PATH));

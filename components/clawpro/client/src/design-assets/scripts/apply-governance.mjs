// =============================================================================
// 阶段 5：重复资源实际治理（apply-governance.mjs）
// -----------------------------------------------------------------------------
// 定位（严格遵守建设计划 §9 阶段 5 / §四 重复资源治理方案 / §八 安全落地）：
//   - A 类：清理「内容完全一致 + 冗余副本无任何引用线索」的重复副本（保留 canonical）。
//   - B 类：对「内容一致 + 有业务引用」的重复做小范围引用替换后再移除（本仓库实测 B=0）。
//   - C 类：红线 / 文件名疑似 / 仅余 registry 事实源副本 → 只标待确认，不删除、不替换。
//   - 产出治理报告与治理状态数据，记录处理前后、原因、影响范围与回滚方式。
//
// 纪律（务必遵守，删除是破坏性动作，逐条独立复核后才删）：
//   1. **不信任上游结论、独立现场复核**：阶段 4 的 action=remove-redundant 只是「候选」，
//      本脚本删除前对每个候选重新做安全复核（见 reverify()），任一不过 → 跳过并记 needs-review。
//   2. **registry 事实源永不删**：role=keep-registry-source 一律保留。
//   3. **红线永不自动归并/删除**：组 redLine 或成员 primaryCategory ∈
//      {brand-logo,channel-icon,agent-avatar} / visualType ∈ {brand-fixed,avatar-like} → 跳过。
//   4. **绝不删最后一份**：删除前确认同组 canonical 仍在磁盘，且候选现场字节 hash 与
//      canonical 现场字节 hash 完全一致（内容已被保留的副本承载）。
//   5. **usageCount=0 ≠ 可删**：还需同时满足 无目录级动态引用 + 不在 unresolvedRefs 命中
//      + usage 表无该 id + 父目录不在 dirReferences。
//   6. 默认 dry-run（只复核、只产报告，不删文件）；显式 `--apply` 才真正删除。
//   7. 不动 skill（非阶段 9）；不动组件源码；不建 canonical 入口（阶段 6）。
//   8. 回滚：删除均由 git 跟踪，报告内记录每个被删路径，可 `git checkout -- <path>` 复原。
//
// 运行：
//   node client/src/design-assets/scripts/apply-governance.mjs            # dry-run（默认，不删）
//   node client/src/design-assets/scripts/apply-governance.mjs --apply    # 实际清理 A 类冗余副本
//
// 输入（阶段 2/3/4 产物，只读）：
//   client/src/design-assets/generated/resource-duplicates.generated.json
//   client/src/design-assets/generated/resource-inventory.generated.json
//   client/src/design-assets/generated/resource-usage.generated.json
// 产物：
//   client/src/design-assets/generated/resource-governance.generated.json
//   docs/resource-governance-report.md
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
const GOV_PATH = path.join(GEN_DIR, "resource-governance.generated.json");
const REPORT_PATH = path.join(repoRoot, "docs/resource-governance-report.md");

const APPLY = process.argv.includes("--apply");

// 红线类目 / 视觉类型（防御性二次校验，A 组本不应含红线，但仍兜底）
const RED_LINE_CATEGORIES = new Set(["brand-logo", "channel-icon", "agent-avatar"]);
const RED_LINE_VISUAL = new Set(["brand-fixed", "avatar-like"]);

// ---------------------------------------------------------------------------
// 读取输入（只读）
// ---------------------------------------------------------------------------
const inventory = JSON.parse(fs.readFileSync(INVENTORY_PATH, "utf8"));
const usage = JSON.parse(fs.readFileSync(USAGE_PATH, "utf8"));
const dup = JSON.parse(fs.readFileSync(DUP_PATH, "utf8"));

const itemById = new Map((inventory.items || []).map((it) => [it.id, it]));
const usageMap = usage.usage || {};
const dirReferences = usage.dirReferences || {};
const referencedViaUnresolved = new Set(
  (usage.unresolvedRefs || []).filter((u) => u.resolved).map((u) => u.resolved)
);
const dirRefKeys = Object.keys(dirReferences);

// ---------------------------------------------------------------------------
// 工具
// ---------------------------------------------------------------------------
function sha256OfFile(absPath) {
  return crypto.createHash("sha256").update(fs.readFileSync(absPath)).digest("hex");
}
function abs(repoRelPath) {
  return path.join(repoRoot, repoRelPath);
}
function existsFile(repoRelPath) {
  try {
    return fs.statSync(abs(repoRelPath)).isFile();
  } catch {
    return false;
  }
}
// 候选路径是否被 unresolvedRefs 命中（用 sourcePath 与 inventory.webPath 双重比对）
function hitByUnresolved(member) {
  if (referencedViaUnresolved.has(member.sourcePath)) return true;
  const it = itemById.get(member.id);
  if (it && it.webPath && referencedViaUnresolved.has(it.webPath)) return true;
  return false;
}
// 候选父目录是否被字面量动态引用（dirReferences 的 key 为 repo 相对目录）
function parentDirDynamicallyReferenced(sourcePath) {
  return dirRefKeys.some(
    (dir) => sourcePath === dir || sourcePath.startsWith(dir + "/")
  );
}

// ---------------------------------------------------------------------------
// 独立现场安全复核：返回 { safe, reasons[] }
//   只有所有红线都不触发，才认为该冗余副本可安全删除。
// ---------------------------------------------------------------------------
function reverify(group, member, canonicalMember) {
  const reasons = [];

  if (group.grade !== "A") reasons.push(`组分级非 A（${group.grade}）`);
  if (group.redLine) reasons.push("组命中红线");
  if (member.role !== "duplicate" || member.action !== "remove-redundant")
    reasons.push(`成员非冗余副本（role=${member.role}, action=${member.action}）`);
  if (RED_LINE_CATEGORIES.has(member.primaryCategory))
    reasons.push(`成员属红线类目（${member.primaryCategory}）`);
  if (RED_LINE_VISUAL.has(member.visualType))
    reasons.push(`成员属红线视觉类型（${member.visualType}）`);

  // 引用安全集（usageCount=0 ≠ 可删，必须叠加以下全部）
  if ((member.usageCount || 0) !== 0) reasons.push(`usageCount=${member.usageCount}`);
  if (member.dynamicDirReferenced) reasons.push("成员被目录级动态引用");
  if (usageMap[member.id]) reasons.push("usage 表中存在该资源引用");
  if (hitByUnresolved(member)) reasons.push("被 unresolvedRefs 命中");
  if (parentDirDynamicallyReferenced(member.sourcePath))
    reasons.push("父目录被字面量动态引用");

  // 物理安全：候选存在、canonical 存在、且现场字节完全一致（内容已被保留）
  if (!existsFile(member.sourcePath)) {
    reasons.push("候选文件已不在磁盘（无需删除）");
  }
  if (!canonicalMember || !canonicalMember.sourcePath) {
    reasons.push("组内缺少可保留的 canonical 成员");
  } else if (!existsFile(canonicalMember.sourcePath)) {
    reasons.push("canonical 文件不在磁盘（避免删成最后一份）");
  } else if (existsFile(member.sourcePath)) {
    const a = sha256OfFile(abs(member.sourcePath));
    const b = sha256OfFile(abs(canonicalMember.sourcePath));
    if (a !== b) reasons.push("现场字节 hash 与 canonical 不一致（非完全重复，不删）");
  }

  return { safe: reasons.length === 0, reasons };
}

// ---------------------------------------------------------------------------
// 逐组治理
// ---------------------------------------------------------------------------
const groups = dup.groups || [];
const removed = []; // { id, path, bytes, groupId, canonicalId, canonicalPath, rollback }
const skipped = []; // { id, path, groupId, reasons[] }
const cNeedsReview = []; // C 类待确认（含红线/keep-multiple/文件名疑似）
const govGroups = []; // 每组治理状态

let bytesReclaimed = 0;

// 治理状态枚举（建设计划 §3.8）
const STATE = {
  REMOVED: "duplicates-removed",
  CANONICAL_CONFIRMED: "canonical-confirmed",
  REFERENCES_UPDATED: "references-updated",
  REVIEWED_CONFIRMED: "reviewed-confirmed",
  KEEP_MULTIPLE: "keep-multiple",
  PENDING_REVIEW: "pending-review",
  NO_ACTION: "no-action",
};

for (const g of groups) {
  const canonicalMember =
    (g.members || []).find((m) => m.role === "canonical-suggested") || null;

  const memberRecords = [];
  let groupRemovedCount = 0;
  let groupSkippedCount = 0;

  if (g.grade === "A") {
    for (const m of g.members || []) {
      if (m.role !== "duplicate" || m.action !== "remove-redundant") {
        // canonical-suggested / keep-registry-source → 保留
        memberRecords.push({ id: m.id, path: m.sourcePath, role: m.role, result: "kept" });
        continue;
      }
      const { safe, reasons } = reverify(g, m, canonicalMember);
      if (safe) {
        if (APPLY) {
          fs.rmSync(abs(m.sourcePath), { force: true });
        }
        bytesReclaimed += m.bytes || 0;
        groupRemovedCount += 1;
        removed.push({
          id: m.id,
          path: m.sourcePath,
          bytes: m.bytes || 0,
          groupId: g.id,
          method: g.method,
          canonicalId: canonicalMember ? canonicalMember.id : null,
          canonicalPath: canonicalMember ? canonicalMember.sourcePath : null,
          rollback: `git checkout -- "${m.sourcePath}"`,
        });
        memberRecords.push({
          id: m.id,
          path: m.sourcePath,
          role: m.role,
          result: APPLY ? "removed" : "would-remove",
        });
      } else {
        groupSkippedCount += 1;
        skipped.push({ id: m.id, path: m.sourcePath, groupId: g.id, reasons });
        memberRecords.push({
          id: m.id,
          path: m.sourcePath,
          role: m.role,
          result: "skipped-needs-review",
          reasons,
        });
      }
    }
    const state =
      groupSkippedCount > 0
        ? STATE.PENDING_REVIEW
        : groupRemovedCount > 0
          ? STATE.REMOVED
          : STATE.NO_ACTION;
    govGroups.push({
      id: g.id,
      grade: g.grade,
      method: g.method,
      redLine: false,
      state,
      canonicalId: canonicalMember ? canonicalMember.id : null,
      canonicalPath: canonicalMember ? canonicalMember.sourcePath : null,
      removedCount: groupRemovedCount,
      skippedCount: groupSkippedCount,
      members: memberRecords,
    });
  } else if (g.grade === "B") {
    // 本仓库 B=0；保留逻辑：B 类需先小范围替换引用再移除，属人工 review，本脚本不自动改引用。
    for (const m of g.members || []) {
      memberRecords.push({ id: m.id, path: m.sourcePath, role: m.role, result: "kept" });
    }
    govGroups.push({
      id: g.id,
      grade: "B",
      method: g.method,
      redLine: !!g.redLine,
      state: STATE.PENDING_REVIEW,
      note: "B 类需小范围替换引用到 canonical 后再移除，属人工 review，脚本不自动改引用",
      members: memberRecords,
    });
  } else {
    // C 类：只标待确认 / 已人工确认，不删除、不替换
    const isFilenameSimilar = g.method === "filename-similar";
    const isReviewConfirmed = g.reviewStatus === "confirmed";
    const isKeepMultiple = (g.reasons || []).some((r) => r.includes("registry 事实源"));
    const state = isReviewConfirmed
      ? STATE.REVIEWED_CONFIRMED
      : isKeepMultiple
        ? STATE.KEEP_MULTIPLE
        : STATE.PENDING_REVIEW;
    for (const m of g.members || []) {
      memberRecords.push({ id: m.id, path: m.sourcePath, role: m.role, result: "kept" });
    }
    // 已人工确认的组已闭环，不再计入「待确认」清单
    if (!isReviewConfirmed) {
      const rec = {
        id: g.id,
        grade: "C",
        method: g.method,
        redLine: !!g.redLine,
        state,
        reasons: g.reasons || [],
        members: (g.members || []).map((m) => ({
          id: m.id,
          path: m.sourcePath,
          role: m.role,
          primaryCategory: m.primaryCategory,
          usageCount: m.usageCount,
        })),
      };
      cNeedsReview.push(rec);
    }
    govGroups.push({
      id: g.id,
      grade: "C",
      method: g.method,
      redLine: !!g.redLine,
      state,
      reviewStatus: g.reviewStatus || "pending",
      reasons: g.reasons || [],
      reasonKind: isReviewConfirmed
        ? "reviewed-confirmed"
        : g.redLine
          ? "red-line"
          : isFilenameSimilar
            ? "filename-similar"
            : "keep-multiple",
      members: memberRecords,
    });
  }
}

// ---------------------------------------------------------------------------
// 产出治理状态数据
// ---------------------------------------------------------------------------
const removedIds = removed.map((r) => r.id);
const generatedAt = new Date().toISOString();

const govSummary = {
  groupsTotal: groups.length,
  aGroups: groups.filter((g) => g.grade === "A").length,
  bGroups: groups.filter((g) => g.grade === "B").length,
  cGroups: groups.filter((g) => g.grade === "C").length,
  removeRedundantCandidates: dup.summary ? dup.summary.removeRedundantCandidates : null,
  removed: removed.length,
  skipped: skipped.length,
  bReplaced: 0,
  cNeedsReviewGroups: cNeedsReview.length,
  reviewedConfirmedGroups: groups.filter((g) => g.reviewStatus === "confirmed").length,
  bytesReclaimed,
};

const gov = {
  $schema: "clawpro-resource-governance",
  stage: 5,
  generatedAt,
  mode: APPLY ? "applied" : "dry-run",
  note:
    "阶段 5 重复资源实际治理：A 类清理冗余副本（保留 canonical）、B 类引用替换（本仓库 B=0）、C 类只标待确认。" +
    "删除前对每个候选独立现场复核（红线/引用安全集/canonical 在场/字节一致），任一不过即跳过。" +
    "本文件为治理事实记录；阶段 2/3/4 的 inventory/duplicates 仍为治理前基线，页面用 removedIds 隐藏已删资源。",
  inputs: {
    duplicates: { path: "client/src/design-assets/generated/resource-duplicates.generated.json", generatedAt: dup.generatedAt },
    inventory: { path: "client/src/design-assets/generated/resource-inventory.generated.json", generatedAt: inventory.generatedAt },
    usage: { path: "client/src/design-assets/generated/resource-usage.generated.json", generatedAt: usage.generatedAt },
  },
  reverificationRules: [
    "组分级=A 且组非红线",
    "成员 role=duplicate 且 action=remove-redundant",
    "成员非红线类目（brand-logo/channel-icon/agent-avatar）、非红线视觉（brand-fixed/avatar-like）",
    "usageCount=0 且 dynamicDirReferenced=false 且 usage 表无该 id 且 不在 unresolvedRefs 命中 且 父目录不在 dirReferences",
    "候选文件在磁盘、同组 canonical 文件在磁盘、且候选与 canonical 现场字节 hash 完全一致",
  ],
  summary: govSummary,
  removedIds,
  removed,
  skipped,
  cNeedsReview,
  groups: govGroups,
  rollback: {
    method: "git",
    note: "全部删除均由 git 跟踪，可整体或逐个回滚",
    commands: [
      "git status   # 查看被删文件",
      "git checkout -- <path>   # 复原单个文件",
      "git restore client/src/design-assets/scripts/apply-governance.mjs  # 仅示例",
    ],
  },
};

fs.writeFileSync(GOV_PATH, JSON.stringify(gov, null, 2) + "\n", "utf8");

// ---------------------------------------------------------------------------
// 产出治理报告（人类可读，由治理事实渲染，避免手抄漂移）
// ---------------------------------------------------------------------------
function mdEscape(s) {
  return String(s).replace(/\|/g, "\\|");
}

const removedByGroup = new Map();
for (const r of removed) {
  if (!removedByGroup.has(r.groupId)) removedByGroup.set(r.groupId, []);
  removedByGroup.get(r.groupId).push(r);
}

const lines = [];
lines.push("# ClawPro 资源库重复治理报告");
lines.push("");
lines.push("> 本报告由 `client/src/design-assets/scripts/apply-governance.mjs` 自动生成，对应建设计划 §9 阶段 5「重复资源实际治理」。");
lines.push("> 报告内容来自治理事实数据 `client/src/design-assets/generated/resource-governance.generated.json`，请勿手改本文件结论，改口径请改脚本后重跑。");
lines.push("");
lines.push(`- 生成时间：${generatedAt}`);
lines.push(`- 运行模式：\`${gov.mode}\`${APPLY ? "（已实际删除冗余副本）" : "（演练：仅复核，未删除任何文件）"}`);
lines.push(`- 输入基线：duplicates @ ${dup.generatedAt} / inventory @ ${inventory.generatedAt} / usage @ ${usage.generatedAt}`);
lines.push("");
lines.push("## 一、治理边界（本阶段严格遵守）");
lines.push("");
lines.push("- 只做 A 类清理（删除「内容完全一致 + 无任何引用线索」的冗余副本，保留 canonical）。");
lines.push("- B 类（内容一致 + 有业务引用）应小范围替换引用后再移除——**本仓库实测 B=0，无此项工作**。");
lines.push("- C 类（红线 / 文件名疑似 / 仅余 registry 事实源副本）**只标待确认，不删除、不替换**。");
lines.push("- 不建 canonical 入口（阶段 6）、不动组件源码、不动 skill（阶段 9）、不做全量迁移。");
lines.push("- registry 事实源（root `icon/`）永不删除；品牌 / 渠道 / 头像红线资源不自动归并。");
lines.push("");
lines.push("## 二、治理结果总览");
lines.push("");
lines.push("| 指标 | 数量 |");
lines.push("|---|---:|");
lines.push(`| 重复组总数 | ${govSummary.groupsTotal} |`);
lines.push(`| A 类组 / B 类组 / C 类组 | ${govSummary.aGroups} / ${govSummary.bGroups} / ${govSummary.cGroups} |`);
lines.push(`| 阶段 4 标记的 remove-redundant 候选 | ${govSummary.removeRedundantCandidates} |`);
lines.push(`| ${APPLY ? "已删除" : "复核通过可删"}冗余副本 | ${govSummary.removed} |`);
lines.push(`| 复核未通过、转待确认 | ${govSummary.skipped} |`);
lines.push(`| B 类引用替换 | ${govSummary.bReplaced} |`);
lines.push(`| C 类待确认组 | ${govSummary.cNeedsReviewGroups} |`);
lines.push(`| C 类已人工确认组 | ${govSummary.reviewedConfirmedGroups} |`);
lines.push(`| 回收字节数 | ${govSummary.bytesReclaimed} |`);
lines.push("");
lines.push("## 三、删除前独立复核规则（逐候选执行，任一不过即跳过）");
lines.push("");
for (const r of gov.reverificationRules) lines.push(`- ${r}`);
lines.push("");
lines.push("> `usageCount=0` 单独不构成删除依据；删除依据是「无任何引用线索 + 同组 canonical 在场且字节完全一致（内容已被保留）」。所有删除均由 git 跟踪，可回滚。");
lines.push("");

// A 类删除明细
lines.push("## 四、A 类清理明细");
lines.push("");
if (removed.length === 0) {
  lines.push("（无）");
} else {
  lines.push(`共 ${removed.length} 个冗余副本${APPLY ? "已删除" : "复核通过（dry-run 未删）"}，按重复组列出（每组保留的 canonical 即内容承载者）：`);
  lines.push("");
  const sortedGroupIds = [...removedByGroup.keys()].sort();
  for (const gid of sortedGroupIds) {
    const recs = removedByGroup.get(gid);
    const canonicalPath = recs[0].canonicalPath;
    lines.push(`### ${gid}（保留 canonical：\`${canonicalPath || "—"}\`，删 ${recs.length}）`);
    lines.push("");
    for (const r of recs) {
      lines.push(`- \`${mdEscape(r.path)}\`（${r.bytes} bytes）→ 回滚：\`${r.rollback}\``);
    }
    lines.push("");
  }
}

// 跳过（复核未通过）
lines.push("## 五、复核未通过 / 转待确认的候选");
lines.push("");
if (skipped.length === 0) {
  lines.push("（无：所有 A 类 remove-redundant 候选均通过独立复核）");
} else {
  lines.push("| 资源 | 组 | 未通过原因 |");
  lines.push("|---|---|---|");
  for (const s of skipped) {
    lines.push(`| \`${mdEscape(s.path)}\` | ${s.groupId} | ${mdEscape(s.reasons.join("；"))} |`);
  }
}
lines.push("");

// B 类
lines.push("## 六、B 类引用替换");
lines.push("");
lines.push(`本仓库 B 类重复组数量为 ${govSummary.bGroups}。`);
lines.push("阶段 4 已核实：唯一含「>1 被引用成员」的组是渠道红线组 `dup-074`（`channel-wecom.svg` 与 `channel-wecom-app.svg` 字节一致、均被引用），已被红线正确判为 C 而非 B。因此本阶段无 B 类引用替换工作。");
lines.push("");

// C 类
lines.push("## 七、C 类待确认清单（不删除、不替换）");
lines.push("");
lines.push("| 组 | 方法 | 类型 | 治理状态 | 成员 | 说明 |");
lines.push("|---|---|---|---|---|---|");
for (const c of cNeedsReview) {
  const kind = c.redLine ? "红线" : c.method === "filename-similar" ? "文件名疑似" : "keep-multiple";
  const members = c.members.map((m) => `\`${m.path}\`(${m.role})`).join("<br/>");
  lines.push(`| ${c.id} | ${c.method} | ${kind} | ${c.state} | ${mdEscape(members)} | ${mdEscape((c.reasons || []).join("；"))} |`);
}
lines.push("");

// 回滚与影响
lines.push("## 八、影响范围与回滚");
lines.push("");
lines.push("- **影响范围**：删除的全部为「与保留 canonical 字节完全一致、且无任何引用线索」的冗余副本，绝大多数为根 `assets/CodeBuddyAssets/` Figma 导出重复件与 `client/public/icon/X 2.svg` 拷贝；内容均已被同组保留文件承载，运行时与页面引用不受影响。");
lines.push("- **registry 事实源、品牌 / 渠道 / 头像红线资源、被引用资源**：全部保留。");
lines.push("- **回滚方式**：所有删除由 git 跟踪。整体回滚 `git checkout -- assets client/public`，或逐个按上文「回滚」命令复原。");
lines.push("- **治理后重跑**：重跑 `scan-resources → classify-resources → detect-duplicates` 将反映治理后现状（重复组收敛）；本报告与 `resource-governance.generated.json` 为治理事实记录。");
lines.push("");
lines.push("## 九、产物清单（阶段 5）");
lines.push("");
lines.push("```text");
lines.push("docs/resource-governance-report.md                                       # 本报告");
lines.push("client/src/design-assets/generated/resource-governance.generated.json    # 治理状态数据");
lines.push("client/src/design-assets/scripts/apply-governance.mjs                    # 治理脚本（可重复执行）");
lines.push("已删除的 A 类冗余副本（见第四节，git diff 可审计、可回滚）");
lines.push("C 类待确认清单（见第七节）");
lines.push("```");
lines.push("");

fs.writeFileSync(REPORT_PATH, lines.join("\n"), "utf8");

// ---------------------------------------------------------------------------
// 控制台摘要
// ---------------------------------------------------------------------------
console.log(`[apply-governance] mode=${gov.mode}`);
console.log(`  groups: total=${govSummary.groupsTotal} A=${govSummary.aGroups} B=${govSummary.bGroups} C=${govSummary.cGroups}`);
console.log(`  remove-redundant candidates=${govSummary.removeRedundantCandidates}`);
console.log(`  ${APPLY ? "removed" : "would-remove"}=${govSummary.removed}  skipped(needs-review)=${govSummary.skipped}  bytesReclaimed=${govSummary.bytesReclaimed}`);
console.log(`  C needs-review groups=${govSummary.cNeedsReviewGroups}  reviewed-confirmed groups=${govSummary.reviewedConfirmedGroups}`);
console.log(`  -> ${path.relative(repoRoot, GOV_PATH)}`);
console.log(`  -> ${path.relative(repoRoot, REPORT_PATH)}`);
if (!APPLY) console.log("  (dry-run：未删除任何文件，加 --apply 执行清理)");

#!/usr/bin/env node
/**
 * 组件资源使用自检脚本（阶段 8 产物）
 * ─────────────────────────────────────────────────────────────────
 * 配套事实源：client/src/design-assets/manual-overrides/component-resource-map.json
 *
 * 目的：在组件源码层（client/src/components/**）识别「资源使用错误用法」，
 *      只防新增倒退，不修存量、不改任何组件源码（阶段 8 硬边界）。
 *
 * 检查规则（每条独立基线）：
 *   1. component-imports-canonical（基线 0，零容忍）
 *      组件源码禁止 import canonicalAssets / canonical-assets。
 *      canonicalAssets 仅当前项目「页面层」可用（pages/**），组件层引入会把
 *      当前项目资源入口带进共享组件，破坏可移植性。
 *   2. component-hardcoded-assets（基线锁存量，防新增）
 *      组件源码写死当前项目 /assets/... 或 @/assets/... 资源路径。
 *      存量（AgentAvatar / Empty / MemoryPreview / AdminLayout 等）按共享组件
 *      记录、本阶段不修；新增即报错。
 *   3. inline-svg（基线锁存量，防无序新增）
 *      组件源码内联 <svg>。存量（StatusBadge / NumberCard / AdminSidebar /
 *      NavIcons 等，多为动画/渐变语义，已审计决定保留）计入基线；新增需评估
 *      是否应改用 lucide 或已登记资源。
 *   4. emoji-icon（基线锁存量，防新增）
 *      源码以 emoji / 符号充当图标。存量（如 span 类型图标 🦞/⚡/🔍、提示符 ✓/✗/⚠）
 *      属共享组件既有用法，本阶段不改源码、计入基线；新增即报错，目标趋于 0。
 *
 * 手工约束（不自动扫描，避免误报；见 component-resource-map.json）：
 *   - brand-resource-wrong-slot：channel-icon / brand-logo / agent-avatar 红线资源
 *     禁当普通 UI 图标改色、禁进普通 icon 槽位、跨仓由宿主注入。CR 时人工把关。
 *
 * 退出码：
 *   - 0：所有规则当前计数 ≤ 各自基线
 *   - 1：任一规则超出基线（新增违规）；或 STRICT=1 时任一规则计数 > 0
 *
 * 用法：
 *   node client/src/design-assets/scripts/check-component-resource-usage.mjs
 *   STRICT=1 node ...           # 严格模式：任何违规即报错（专项清理用）
 *   COUNT=1 node ...            # 只打印当前各规则计数（用于校准基线）
 * ─────────────────────────────────────────────────────────────────
 */

import { readdir, readFile } from "node:fs/promises";
import { join, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
// .../client/src/design-assets/scripts/ → 上溯 4 级到 client/，再上溯到仓库根
const REPO_ROOT = join(__filename, "..", "..", "..", "..", "..");
const SCAN_ROOT = join(REPO_ROOT, "client", "src", "components");

const STRICT_MODE = process.env.STRICT === "1";
const COUNT_ONLY = process.env.COUNT === "1";

const EXT_ALLOWLIST = [".ts", ".tsx"];
const PATH_BLOCKLIST = [`${sep}node_modules${sep}`, `${sep}dist${sep}`, `${sep}build${sep}`];

// 行级豁免：违规行尾或上一行注释含 `allow-asset:`
const ALLOW_TAG = "allow-asset:";

/**
 * 存量基线（阶段 8 锁定快照，更新时间 2026-06-14）
 * 维护规则：清理存量后把对应数字往下调，目标趋于 0（canonical 始终 0）。
 */
const BASELINE = {
  "component-imports-canonical": 0,
  "component-hardcoded-assets": 18,
  "inline-svg": 40,
  "emoji-icon": 18,
};

const RULES = [
  {
    id: "component-imports-canonical",
    test: (line) =>
      /\bfrom\s+["'][^"']*canonical-assets["']/.test(line) ||
      (/\bimport\b/.test(line) && /canonicalAssets/.test(line)),
    hint: "组件源码禁止 import canonicalAssets。canonicalAssets 仅当前项目页面层（pages/**）使用；组件需要资源时由页面通过 props/槽位传入，跨仓由宿主注入。",
  },
  {
    id: "component-hardcoded-assets",
    test: (line) => /["'`(]@?\/assets\//.test(line),
    hint: "组件源码写死当前项目 /assets 路径不可移植。资源应由页面层（可用 canonicalAssets）或宿主通过 props/槽位注入；如确需改造组件默认资源，须脱离本任务单独立项。",
  },
  {
    id: "inline-svg",
    test: (line) => /<svg[\s>]/.test(line),
    hint: "组件新增 inline <svg> 前请评估：通用图标优先 lucide-react；品牌/多彩语义图标走已登记资源。存量动画/渐变图标已审计保留。",
  },
  {
    id: "emoji-icon",
    // 常见 emoji 区段；排除注释行以降误报
    test: (line) => {
      const t = line.trim();
      if (t.startsWith("//") || t.startsWith("*") || t.startsWith("/*")) return false;
      return /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}]/u.test(line);
    },
    hint: "禁止以 emoji 充当图标，请使用 lucide-react 或已登记资源。",
  },
];

async function* walk(dir) {
  let entries;
  try {
    entries = await readdir(dir, { withFileTypes: true });
  } catch {
    return;
  }
  for (const entry of entries) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) {
      yield* walk(full);
    } else if (entry.isFile()) {
      const ext = "." + entry.name.split(".").pop();
      if (!EXT_ALLOWLIST.includes(ext)) continue;
      if (PATH_BLOCKLIST.some((seg) => full.includes(seg))) continue;
      yield full;
    }
  }
}

function isLineAllowed(lines, idx) {
  const cur = lines[idx];
  if (cur && cur.includes(ALLOW_TAG)) return true;
  const prev = idx > 0 ? lines[idx - 1].trim() : "";
  return prev.startsWith("//") && prev.includes(ALLOW_TAG);
}

async function checkFile(filePath) {
  const lines = (await readFile(filePath, "utf8")).split("\n");
  const violations = [];
  for (let i = 0; i < lines.length; i++) {
    if (isLineAllowed(lines, i)) continue;
    for (const rule of RULES) {
      if (rule.test(lines[i])) {
        violations.push({ rule: rule.id, line: i + 1, snippet: lines[i].trim().slice(0, 160), hint: rule.hint });
      }
    }
  }
  return violations;
}

async function main() {
  const counts = Object.fromEntries(RULES.map((r) => [r.id, 0]));
  const fileReports = [];

  for await (const file of walk(SCAN_ROOT)) {
    const violations = await checkFile(file);
    if (violations.length > 0) {
      for (const v of violations) counts[v.rule]++;
      fileReports.push({ file, violations });
    }
  }

  if (COUNT_ONLY) {
    console.log("check-component-resource-usage 当前各规则计数：");
    for (const r of RULES) console.log(`  ${r.id}: ${counts[r.id]} (基线 ${BASELINE[r.id]})`);
    process.exit(0);
  }

  const failedRules = RULES.filter((r) =>
    STRICT_MODE ? counts[r.id] > 0 : counts[r.id] > BASELINE[r.id],
  );

  if (failedRules.length === 0) {
    console.log("✓ check-component-resource-usage: 组件资源使用未超基线，无新增违规。");
    for (const r of RULES) {
      if (counts[r.id] > 0) console.log(`  · ${r.id}: ${counts[r.id]}（基线 ${BASELINE[r.id]}，未超出）`);
    }
    process.exit(0);
  }

  console.error("");
  console.error(`\x1b[31m✗ check-component-resource-usage: 发现超出基线的组件资源用法\x1b[0m`);
  for (const r of failedRules) {
    console.error(`  [${r.id}] 当前 ${counts[r.id]} > 基线 ${BASELINE[r.id]}`);
  }
  console.error("  事实源：client/src/design-assets/manual-overrides/component-resource-map.json\n");

  const failedIds = new Set(failedRules.map((r) => r.id));
  for (const { file, violations } of fileReports) {
    const shown = violations.filter((v) => failedIds.has(v.rule));
    if (shown.length === 0) continue;
    console.error(`\x1b[33m  ${relative(REPO_ROOT, file)}\x1b[0m`);
    for (const v of shown) {
      console.error(`    L${v.line}  [${v.rule}]  ${v.snippet}`);
      console.error(`      → ${v.hint}`);
    }
    console.error("");
  }
  console.error("  豁免方式：在违规行末尾或上一行添加 `// allow-asset: <理由>`（仅限合理场景）。\n");
  process.exit(1);
}

main().catch((err) => {
  console.error("check-component-resource-usage 脚本异常：", err);
  process.exit(2);
});

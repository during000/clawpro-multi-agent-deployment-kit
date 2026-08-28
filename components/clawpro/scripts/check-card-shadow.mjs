#!/usr/bin/env node
/**
 * 卡片阴影规范自检脚本
 * ─────────────────────────────────────────────────────────────────
 * 检查业务层（client/src/{components,pages}/**）是否存在违规阴影写法。
 *
 * 违规模式：
 *   1. inline `style={{ boxShadow: ... }}` —— 应改用 <SurfaceCard>
 *   2. className 含 `shadow-md` / `shadow-lg` / `shadow-xl` / `shadow-2xl`
 *      （以及对应的 hover: 变体） —— 应改用 <SurfaceCard hover> / <SurfaceConfig>
 *
 * 白名单：
 *   - 文件路径白名单：client/src/components/ui/**（底层组件本身、shadcn 实现）
 *   - 行级豁免：在违规行尾或上一行添加注释 `// allow-shadow: <理由>`
 *
 * 退出码：
 *   - 0：违规数 ≤ BASELINE（即不超过当前已锁定的存量基线，允许逐步消化）
 *   - 1：违规数 > BASELINE（说明引入了新增违规，必须清理）
 *   - 1：环境变量 STRICT=1 时，任何违规均报错（用于专项清理时）
 *
 * 基线机制（"触达即同步"配套）：
 *   存量违规快照在 BASELINE 常量里。任何"触达即同步"刷新一些后，把数字往下调，
 *   逐步推动到 0。这样既不阻塞日常 npm run check（CI 不会立刻全红），又能自动防止
 *   新增违规倒退。详见 SKILL.md §0「协作机制：触达即同步」。
 *
 * 用法：
 *   node scripts/check-card-shadow.mjs           # 默认基线模式
 *   STRICT=1 node scripts/check-card-shadow.mjs  # 严格模式，任何违规即报错
 *   npm run check                                # 已接入 tsc --noEmit && 本脚本
 *   npm run check:shadow                         # 单独跑本脚本
 * ─────────────────────────────────────────────────────────────────
 */

import { readdir, readFile } from "node:fs/promises";
import { join, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";

// ────────────────────── 配置 ──────────────────────

const __filename = fileURLToPath(import.meta.url);
const REPO_ROOT = join(__filename, "..", "..");
const SCAN_ROOTS = [
  join(REPO_ROOT, "client", "src", "components"),
  join(REPO_ROOT, "client", "src", "pages"),
];

/**
 * 存量违规基线（"触达即同步"配套）
 * ─────────────────────────────────────────────────────────────────
 * 当前快照（更新时间：2026-05-14）：196 处违规、约 50 个文件。
 *
 * 维护规则：
 *   - 当违规数 ≤ BASELINE，脚本 exit 0（允许日常 CI 通过、存量逐步消化）
 *   - 当违规数 > BASELINE，脚本 exit 1（防止新增违规倒退）
 *   - 任何"触达即同步"动作清掉一批违规后，必须**同步把这里的数字往下调**，
 *     避免下次新增违规又被误判为"还在基线内"。
 *   - 目标是让这个数字最终降到 0，届时整个仓库进入零违规模式。
 */
const BASELINE = 187;

const STRICT_MODE = process.env.STRICT === "1";

/** 不扫描的子路径片段（路径中包含任一即跳过） */
const PATH_BLOCKLIST = [
  `${sep}components${sep}ui${sep}`,         // shadcn 与 Surface 实现本身
  `${sep}node_modules${sep}`,
  `${sep}dist${sep}`,
  `${sep}build${sep}`,
  `${sep}.codebuddy${sep}`,
];

/** 只扫描这些扩展名 */
const EXT_ALLOWLIST = [".ts", ".tsx"];

// 行级豁免标记（出现在违规行尾 或 紧邻上一行）
const ALLOW_TAG = "allow-shadow:";

// ────────────────────── 检测规则 ──────────────────────

/**
 * 规则 1：inline boxShadow
 * 命中：style={{ ... boxShadow: ... }} 或 style={{ boxShadow: ... }}
 * 不命中：var(--shadow-segment) 这种引用 token 的（极少数 Tab 滑块场景）
 */
const RULE_INLINE_BOXSHADOW = {
  id: "inline-boxshadow",
  test: (line) => {
    if (!line.includes("boxShadow")) return false;
    // 引用 token 的写法是允许的（兼容 §5.1 L5 Segment）
    if (/boxShadow:\s*["'`]?\s*var\(--shadow-/.test(line)) return false;
    // 精确匹配 inline 样式中的 boxShadow:（避开变量声明 const boxShadow = ...）
    return /\bboxShadow\s*:\s*["'`]/.test(line);
  },
  hint: "请改用 <SurfaceCard>（from @/components/ui/Surface）。如确为浮层/Tab 滑块等特殊场景，使用 var(--shadow-overlay) / var(--shadow-segment)，或在该行末尾加注释 `// allow-shadow: <理由>`。",
};

/**
 * 规则 2：Tailwind shadow-* 重档
 * 命中 className 中独立出现的 shadow-md / shadow-lg / shadow-xl / shadow-2xl（含 hover: 变体）
 * 不命中：shadow-xs / shadow-sm / shadow-none
 */
const RULE_TAILWIND_HEAVY_SHADOW = {
  id: "tailwind-heavy-shadow",
  test: (line) => {
    // 只在 className 字符串里查（粗略：行内含 className 才检测，避免误报注释/字符串）
    if (!line.includes("className") && !line.includes("class=")) {
      // 也允许 cn(...) 等工具调用的多行 className，这里用全行兜底，但要求是字符串字面量上下文
      // 为减少误报，这种"非 className 行"只在出现 hover:shadow-* 才触发
      if (!/\bhover:shadow-(md|lg|xl|2xl)\b/.test(line)) return false;
    }
    return /\b(?:hover:)?shadow-(?:md|lg|xl|2xl)\b/.test(line);
  },
  hint: "禁止使用 shadow-md/lg/xl/2xl。卡片请用 <SurfaceCard hover> 或 <SurfaceConfig>；浮层请用 shadcn 自带 Dialog/Popover 或 <SurfaceOverlay>。",
};

const RULES = [RULE_INLINE_BOXSHADOW, RULE_TAILWIND_HEAVY_SHADOW];

// ────────────────────── 文件遍历 ──────────────────────

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

// ────────────────────── 检测 ──────────────────────

/**
 * 判断某行是否被「行级豁免」豁免：
 * - 该行内本身含 `allow-shadow:` 注释
 * - 紧邻上一行是 `// allow-shadow: ...` 的独立注释
 */
function isLineAllowed(lines, idx) {
  const cur = lines[idx];
  if (cur && cur.includes(ALLOW_TAG)) return true;
  const prev = idx > 0 ? lines[idx - 1].trim() : "";
  if (prev.startsWith("//") && prev.includes(ALLOW_TAG)) return true;
  return false;
}

async function checkFile(filePath) {
  const content = await readFile(filePath, "utf8");
  const lines = content.split("\n");
  const violations = [];
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    for (const rule of RULES) {
      if (!rule.test(line)) continue;
      if (isLineAllowed(lines, i)) continue;
      violations.push({
        rule: rule.id,
        line: i + 1,
        snippet: line.trim().slice(0, 200),
        hint: rule.hint,
      });
    }
  }
  return violations;
}

// ────────────────────── 主流程 ──────────────────────

async function main() {
  let totalViolations = 0;
  const fileReports = [];

  for (const root of SCAN_ROOTS) {
    for await (const file of walk(root)) {
      const violations = await checkFile(file);
      if (violations.length > 0) {
        totalViolations += violations.length;
        fileReports.push({ file, violations });
      }
    }
  }

  if (totalViolations === 0) {
    console.log("✓ check-card-shadow: 无违规，业务层卡片阴影规范 OK。");
    if (BASELINE > 0) {
      console.log(
        `  提示：当前 BASELINE 仍为 ${BASELINE}，建议在 scripts/check-card-shadow.mjs 中把 BASELINE 调成 0。`,
      );
    }
    process.exit(0);
  }

  // 根据基线 / 严格模式决定 exit code
  const overBaseline = totalViolations > BASELINE;
  const shouldFail = STRICT_MODE || overBaseline;

  if (!shouldFail) {
    // 在基线内：允许通过，但仍打印简要报告
    console.log(
      `\x1b[33m! check-card-shadow: 当前 ${totalViolations} 处违规（基线 ${BASELINE}，未超出）\x1b[0m`,
    );
    console.log(
      "  按『触达即同步』机制（SKILL.md §0），存量违规将随日常迭代逐步消化。",
    );
    console.log(
      "  如需查看详细违规列表，运行：STRICT=1 node scripts/check-card-shadow.mjs\n",
    );
    process.exit(0);
  }

  // 输出报告
  console.error("");
  if (overBaseline) {
    console.error(
      `\x1b[31m✗ check-card-shadow: 发现 ${totalViolations} 处违规，超出基线 ${BASELINE}（新增 ${totalViolations - BASELINE} 处）\x1b[0m`,
    );
    console.error(
      "  请清理新增违规后再提交。如确为合理浮层场景，使用行级豁免 `// allow-shadow: <理由>`。",
    );
  } else {
    console.error(
      `\x1b[31m✗ check-card-shadow: 严格模式下发现 ${totalViolations} 处卡片阴影违规\x1b[0m`,
    );
  }
  console.error(
    "  规范：所有业务层卡片必须使用 <SurfaceCard> / <SurfaceInner> / <SurfaceOverlay> / <SurfaceConfig>",
  );
  console.error("  详见：SKILL.md §5「阴影系统」\n");

  for (const { file, violations } of fileReports) {
    const rel = relative(REPO_ROOT, file);
    console.error(`\x1b[33m  ${rel}\x1b[0m`);
    for (const v of violations) {
      console.error(`    L${v.line}  [${v.rule}]`);
      console.error(`      ${v.snippet}`);
      console.error(`      → ${v.hint}`);
    }
    console.error("");
  }

  console.error(
    "  豁免方式：在违规行末尾或上一行添加 `// allow-shadow: <理由>`（仅用于浮层/Tab 滑块等合理场景）。",
  );
  console.error("");
  process.exit(1);
}

main().catch((err) => {
  console.error("check-card-shadow 脚本异常：", err);
  process.exit(2);
});

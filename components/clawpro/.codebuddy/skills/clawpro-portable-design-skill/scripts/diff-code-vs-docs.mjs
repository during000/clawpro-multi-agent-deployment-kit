#!/usr/bin/env node
/**
 * diff-code-vs-docs.mjs  ——  P1 值/枚举差集（只读）
 * ─────────────────────────────────────────────────────────────────────────
 * 定位：设计规范对齐机制（见 docs/design-skill-sync/REQUIREMENT-AND-PLAN.md）里
 *       "P1 值/枚举漂移" 的机械检法。**以代码为准**（决策 §6#5）：
 *
 *   真相源  client/src/components/ui/*.tsx（union / 内联联合 / cva variant 键）
 *          client/src/index.css（CSS 自定义属性 --xxx）
 *      │  提取
 *      ▼  比对「文档是否声明/覆盖了这些代码事实」
 *   下游文档  Tier1 SKILL-GLOBAL-COMPONENTS.md
 *            Tier2 component-specs/*.md + SKILL.md + tokens/*.md
 *
 * 只做一个方向（代码 → 文档覆盖）：
 *   报「代码里有、文档没提」的枚举值 / CSS 变量 —— 即"文档漏声明"候选。
 *   反向（文档声称、代码没有）依赖对 prose 的语义判断，噪声大，交给 AI 逐条判定，
 *   本脚本不做（见文末 LIMITATIONS）。
 *
 * 输出的是**候选清单（差集），不是判决**：文档不一定枚举每个值，
 * AI / 人需逐条判定（决策 §6 P1=脚本+AI）。退出码：
 *   0 = 无差集   1 = 有差集（待人/AI 判定）   2 = 用法/环境错误
 *
 * ── Scope（决策 §10 第 4 条：增量默认 / 全量 opt-in，二选一，不给静默默认）──
 *   --since <git-ref>   只查自 <ref> 以来 git diff 动过的组件/index.css（增量对齐主线）
 *   --full              全量扫所有组件 + index.css（全量审计 / 体检，findings 仅作 backlog）
 *   （两者必须给其一，否则报用法错误 exit 2 —— 强制"别混"）
 *
 * ── 其它参数 ──
 *   --json              机读输出
 *   --src=<path>        指定 client/src（默认自动回溯定位）
 *
 * ── 历史存量豁免（决策 §4.2 / §10.1）──
 *   若存在 references/historical-exemption.json（字符串数组，组件 basename），
 *   其中的组件在 "changed-but-no-spec" 判定里被豁免（不报 CHANGED_NO_SPEC）。
 * ─────────────────────────────────────────────────────────────────────────
 */
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const skillRoot = path.resolve(__dirname, ".."); // .../clawpro-portable-design-skill

/* ───────────── 参数解析 ───────────── */
const args = process.argv.slice(2);
const jsonMode = args.includes("--json");
const fullMode = args.includes("--full");
const sinceArg = args.find((a) => a.startsWith("--since="));
const sinceBare = (() => {
  const i = args.indexOf("--since");
  return i >= 0 && args[i + 1] && !args[i + 1].startsWith("--") ? args[i + 1] : null;
})();
const sinceRef = sinceArg ? sinceArg.slice("--since=".length) : sinceBare;
const srcArg = args.find((a) => a.startsWith("--src="));
const explicitSrc = srcArg ? srcArg.slice("--src=".length) : process.env.CLAW_REPO_SRC;

function usage(msg) {
  if (msg) console.error(`[diff-code-vs-docs] ${msg}\n`);
  console.error(
    "用法（--since 与 --full 二选一）：\n" +
      "  node scripts/diff-code-vs-docs.mjs --since <git-ref>   # 增量对齐（只查改动）\n" +
      "  node scripts/diff-code-vs-docs.mjs --full              # 全量审计（体检，backlog）\n" +
      "  可选：--json  --src=<client/src 绝对路径>\n"
  );
  process.exit(2);
}

if (fullMode && sinceRef) usage("--since 与 --full 互斥，只能给一个。");
if (!fullMode && !sinceRef) usage("必须指定 scope：--since <ref>（增量）或 --full（全量）。");

/* ───────────── 定位 client/src 与仓库根 ───────────── */
function findClientSrc() {
  if (explicitSrc) return path.resolve(explicitSrc);
  let cur = skillRoot;
  for (let i = 0; i < 6; i++) {
    cur = path.dirname(cur);
    const candidate = path.join(cur, "client", "src");
    if (fs.existsSync(candidate)) return candidate;
  }
  return null;
}
const clientSrc = findClientSrc();
if (!clientSrc) usage("无法定位 client/src，请用 --src=<绝对路径> 或环境变量 CLAW_REPO_SRC 指定。");
const repoRoot = path.resolve(clientSrc, "..", "..");
const uiDir = path.join(clientSrc, "components", "ui");
const indexCss = path.join(clientSrc, "index.css");

/* ───────────── 文档来源 ───────────── */
const tier1Path = path.join(repoRoot, "SKILL-GLOBAL-COMPONENTS.md");
const specsDir = path.join(skillRoot, "component-specs");
const tokensDir = path.join(skillRoot, "tokens");
const skillMd = path.join(skillRoot, "SKILL.md");

function readSafe(p) {
  try {
    return fs.readFileSync(p, "utf8");
  } catch {
    return "";
  }
}
function collectMdText(dir) {
  if (!fs.existsSync(dir)) return "";
  return fs
    .readdirSync(dir)
    .filter((f) => f.endsWith(".md"))
    .map((f) => readSafe(path.join(dir, f)))
    .join("\n");
}

const tier1Text = readSafe(tier1Path);
const tokensText = collectMdText(tokensDir);
const skillMdText = readSafe(skillMd);
// design-tokens.json 是 Tier2 的 token 镜像（sync-tokens.mjs 生成），token 名与 index.css 一致
const designTokensJson = readSafe(path.join(tokensDir, "design-tokens.json"));
// tokens(.md + .json) + Tier1 + SKILL.md 作为 CSS 变量的"文档覆盖池"
const cssDocPool = `${tokensText}\n${designTokensJson}\n${tier1Text}\n${skillMdText}`;

/* ───────────── 历史存量豁免清单 ───────────── */
const exemptionPath = path.join(skillRoot, "references", "historical-exemption.json");
let exemptions = new Set();
if (fs.existsSync(exemptionPath)) {
  try {
    const parsed = JSON.parse(readSafe(exemptionPath));
    const arr = Array.isArray(parsed) ? parsed : Array.isArray(parsed?.components) ? parsed.components : null;
    if (arr) exemptions = new Set(arr.map(String));
  } catch {
    /* 豁免清单坏了不阻断，忽略 */
  }
}

/* ───────────── scope：确定要检查哪些组件文件 + 是否查 index.css ───────────── */
function gitDiffNames(ref) {
  // execFile 数组传参，无 shell，ref 不拼进命令串 → 无注入
  const out = execFileSync(
    "git",
    ["-C", repoRoot, "diff", "--name-only", ref, "--", "client/src/components/ui", "client/src/index.css"],
    { encoding: "utf8" }
  );
  return out.split("\n").map((s) => s.trim()).filter(Boolean);
}

/** 增量模式下，从 index.css 的 diff 里抽出新增(+)行声明的 --xxx 变量名 */
function gitAddedCssVars(ref) {
  let out;
  try {
    out = execFileSync("git", ["-C", repoRoot, "diff", ref, "--", "client/src/index.css"], {
      encoding: "utf8",
    });
  } catch {
    return [];
  }
  const vars = new Set();
  for (const line of out.split("\n")) {
    // 只看新增行（+ 开头、且非 +++ 文件头）
    if (!line.startsWith("+") || line.startsWith("+++")) continue;
    const m = /--([a-zA-Z0-9-]+)\s*:/.exec(line);
    if (m) vars.add(m[1]);
  }
  return [...vars];
}

let targetTsx = []; // 绝对路径
let checkCss = false;

if (fullMode) {
  if (fs.existsSync(uiDir)) {
    targetTsx = fs
      .readdirSync(uiDir)
      .filter((f) => f.endsWith(".tsx"))
      .map((f) => path.join(uiDir, f));
  }
  checkCss = fs.existsSync(indexCss);
} else {
  let names;
  try {
    names = gitDiffNames(sinceRef);
  } catch (e) {
    usage(`git diff 失败（ref="${sinceRef}"？）：${String(e.message || e).split("\n")[0]}`);
  }
  for (const rel of names) {
    if (rel === "client/src/index.css") checkCss = true;
    else if (rel.startsWith("client/src/components/ui/") && rel.endsWith(".tsx")) {
      const abs = path.join(repoRoot, rel);
      if (fs.existsSync(abs)) targetTsx.push(abs);
    }
  }
}

/* ───────────── 从 .tsx 提取枚举事实 ─────────────
 * 三类（低噪声、P1 最强信号）：
 *   1) union 类型别名：  type Foo = "a" | "b" | "c"      （也吃 RefType | "lit" 里的字面量）
 *   2) 内联联合 prop：    variant?: "a" | "b"
 *   3) cva variant 组键： cva(..., { variants: { variant: { k1:.., k2:.. }, size:{..} } })
 * 刻意不提取 `keyof typeof 大对象`（如 21 色 token map），噪声太大 → 未来扩展。
 */
const STR_LIT = /"([^"\\]+)"/g;

function extractStringUnion(rhs) {
  // rhs 里以 | 连接、含 ≥1 个字符串字面量，才算枚举联合
  if (!rhs.includes("|") && !/^"[^"]+"$/.test(rhs.trim())) {
    // 单个字面量也接受（少见），但纯引用类型（无字面量）跳过
  }
  const vals = [];
  let m;
  STR_LIT.lastIndex = 0;
  while ((m = STR_LIT.exec(rhs)) !== null) vals.push(m[1]);
  return vals;
}

function extractCvaGroupKeys(text) {
  // 定位 variants: { ... } 的平衡块，再抽 variant/size 等组下的 top-level 键
  const groups = {}; // groupName -> [keys]
  const vIdx = text.indexOf("variants:");
  if (vIdx < 0) return groups;
  // 找到 variants: 后的第一个 {
  let i = text.indexOf("{", vIdx);
  if (i < 0) return groups;
  const variantsBlock = balancedBlock(text, i);
  if (!variantsBlock) return groups;
  // variantsBlock 内：groupName: { ... }
  const GROUP_RE = /([A-Za-z_$][\w$]*)\s*:\s*\{/g;
  let g;
  while ((g = GROUP_RE.exec(variantsBlock)) !== null) {
    const gName = g[1];
    const inner = balancedBlock(variantsBlock, variantsBlock.indexOf("{", g.index + g[0].length - 1));
    if (!inner) continue;
    // 只取组内 depth-0 的键（跳过嵌套对象里的键，如某 variant 值本身是 {hover:..,active:..}）
    const keys = collectTopLevelKeys(inner);
    if (keys.length) groups[gName] = [...new Set(keys)];
  }
  return groups;
}

/** 抽对象字面量文本里 brace-depth 0 的键名（key: 形式）；跳过字符串与注释内容 */
function collectTopLevelKeys(inner) {
  const keys = [];
  let depth = 0;
  let i = 0;
  const n = inner.length;
  const KEY_RE = /["']?([A-Za-z_$][\w$-]*)["']?\s*:/y; // sticky
  let atKeyPos = true; // 是否处于"可能出现键名"的位置（块起始 / 逗号后）
  while (i < n) {
    const c = inner[i];
    // 跳过字符串字面量
    if (c === '"' || c === "'" || c === "`") {
      const quote = c;
      i++;
      while (i < n) {
        if (inner[i] === "\\") {
          i += 2;
          continue;
        }
        if (inner[i] === quote) {
          i++;
          break;
        }
        i++;
      }
      atKeyPos = false;
      continue;
    }
    // 跳过注释
    if (c === "/" && inner[i + 1] === "/") {
      while (i < n && inner[i] !== "\n") i++;
      continue;
    }
    if (c === "/" && inner[i + 1] === "*") {
      i += 2;
      while (i < n && !(inner[i] === "*" && inner[i + 1] === "/")) i++;
      i += 2;
      continue;
    }
    if (c === "{" || c === "[" || c === "(") {
      depth++;
      i++;
      atKeyPos = false;
      continue;
    }
    if (c === "}" || c === "]" || c === ")") {
      depth--;
      i++;
      atKeyPos = false;
      continue;
    }
    if (c === ",") {
      i++;
      if (depth === 0) atKeyPos = true;
      continue;
    }
    if (/\s/.test(c)) {
      i++;
      continue;
    }
    // 非空白实体
    if (depth === 0 && atKeyPos) {
      KEY_RE.lastIndex = i;
      const m = KEY_RE.exec(inner);
      if (m && m.index === i) {
        keys.push(m[1]);
        i = KEY_RE.lastIndex;
        atKeyPos = false;
        continue;
      }
    }
    atKeyPos = false;
    i++;
  }
  return [...new Set(keys)];
}

/** 从 openBraceIdx（指向 '{'）返回平衡括号内的内容（含边界外的内层），失败返回 null */
function balancedBlock(text, openBraceIdx) {
  if (openBraceIdx < 0 || text[openBraceIdx] !== "{") return null;
  let depth = 0;
  for (let i = openBraceIdx; i < text.length; i++) {
    const c = text[i];
    if (c === "{") depth++;
    else if (c === "}") {
      depth--;
      if (depth === 0) return text.slice(openBraceIdx + 1, i);
    }
  }
  return null;
}

function extractEnumsFromTsx(text) {
  const enums = []; // { kind, name, values:[] }

  // 1) union 类型别名
  const TYPE_RE = /(?:export\s+)?type\s+([A-Za-z_$][\w$]*)\s*=\s*([^;]+);/g;
  let m;
  while ((m = TYPE_RE.exec(text)) !== null) {
    const name = m[1];
    const vals = extractStringUnion(m[2]);
    if (vals.length) enums.push({ kind: "union", name, values: vals });
  }

  // 2) 内联联合 prop： xxx?: "a" | "b" | "c"
  const INLINE_RE = /([A-Za-z_$][\w$]*)\??\s*:\s*("(?:[^"\\]+)"(?:\s*\|\s*"[^"\\]+")+)/g;
  while ((m = INLINE_RE.exec(text)) !== null) {
    const name = m[1];
    const vals = extractStringUnion(m[2]);
    if (vals.length > 1) enums.push({ kind: "inline", name, values: vals });
  }

  // 3) cva variant 组键
  const groups = extractCvaGroupKeys(text);
  for (const [gName, keys] of Object.entries(groups)) {
    enums.push({ kind: "cva", name: gName, values: keys });
  }

  return dedupeEnums(enums);
}

function dedupeEnums(enums) {
  // 合并同 name 的 values；kind 优先级 union>cva>inline（信息量）
  const byName = new Map();
  const rank = { union: 3, cva: 2, inline: 1 };
  for (const e of enums) {
    const prev = byName.get(e.name);
    if (!prev) byName.set(e.name, { ...e, values: [...new Set(e.values)] });
    else {
      prev.values = [...new Set([...prev.values, ...e.values])];
      if (rank[e.kind] > rank[prev.kind]) prev.kind = e.kind;
    }
  }
  return [...byName.values()];
}

/* ───────────── 文档覆盖判定 ───────────── */
function escapeRe(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
/** 通用词边界匹配（用于 CSS 变量名，把连字符当 token 一部分）*/
function mentioned(token, text) {
  if (!token || token.length < 2) return true;
  const re = new RegExp(`(?<![\\w-])${escapeRe(token)}(?![\\w-])`);
  return re.test(text);
}
/** 枚举值专用：要求以"字面量"形态出现（反引号/单双引号），比裸词精确，
 *  避免把散落在 prose 里的普通词误当"已声明的枚举值"。*/
function mentionedAsValue(val, text) {
  if (!val) return true;
  const e = escapeRe(val);
  const re = new RegExp("`" + e + "`|\"" + e + "\"|'" + e + "'");
  return re.test(text);
}

/* ───────────── 主流程：组件枚举 → spec/Tier1 覆盖 ───────────── */
const findings = []; // { type, component, spec, name, kind, value }
const checkedComponents = [];
const exemptedHits = [];

for (const tsxPath of targetTsx) {
  const base = path.basename(tsxPath, ".tsx");
  const specPath = path.join(specsDir, `${base}.md`);
  const hasSpec = fs.existsSync(specPath);
  checkedComponents.push(base);

  if (!hasSpec) {
    // 没有对应 spec 的组件
    if (exemptions.has(base)) {
      exemptedHits.push(base);
      continue;
    }
    // 增量模式：被改动碰到却无 spec → 候选（决策：碰到才纳管）
    // 全量模式：仅作历史存量提示（不强制）
    findings.push({
      type: fullMode ? "NO_SPEC_BACKLOG" : "CHANGED_NO_SPEC",
      component: base,
      spec: null,
      name: null,
      kind: null,
      value: null,
    });
    continue;
  }

  const tsxText = readSafe(tsxPath);
  const specText = readSafe(specPath);
  const enums = extractEnumsFromTsx(tsxText);

  for (const e of enums) {
    for (const val of e.values) {
      // 只对该组件自己的 spec 判定（不查整仓 Tier1，避免"别的组件同名值"跨组件掩盖）
      if (!mentionedAsValue(val, specText)) {
        findings.push({
          type: "CODE_ENUM_NOT_IN_DOC",
          component: base,
          spec: path.relative(skillRoot, specPath),
          name: e.name,
          kind: e.kind,
          value: val,
        });
      }
    }
  }
}

/* ───────────── CSS 变量 → tokens/Tier1 覆盖 ───────────── */
let cssVarsChecked = 0;
if (checkCss) {
  const cssText = readSafe(indexCss);
  let varsToCheck;
  if (fullMode) {
    // 全量：index.css 里所有 --xxx
    const VAR_RE = /--([a-zA-Z0-9-]+)\s*:/g;
    const seen = new Set();
    let m;
    while ((m = VAR_RE.exec(cssText)) !== null) seen.add(m[1]);
    varsToCheck = [...seen];
  } else {
    // 增量：只查 diff 里新增(+)的 --xxx 行，避免重扫全部变量
    varsToCheck = gitAddedCssVars(sinceRef);
  }
  for (const varName of varsToCheck) {
    cssVarsChecked++;
    // 文档池里出现 `--varName` / 裸 `varName` / 去掉 color- 前缀后的名字 即算覆盖
    const stripped = varName.replace(/^color-/, "");
    if (
      !mentioned(`--${varName}`, cssDocPool) &&
      !mentioned(varName, cssDocPool) &&
      !(stripped !== varName && mentioned(stripped, cssDocPool))
    ) {
      findings.push({
        type: "CSS_VAR_NOT_IN_DOC",
        component: "index.css",
        spec: "tokens/*.md · design-tokens.json · SKILL-GLOBAL-COMPONENTS.md",
        name: `--${varName}`,
        kind: "css-var",
        value: `--${varName}`,
      });
    }
  }
}

/* ───────────── 输出 ───────────── */
const scopeLabel = fullMode ? "full（全量审计）" : `since ${sinceRef}（增量对齐）`;

if (jsonMode) {
  console.log(
    JSON.stringify(
      {
        scope: fullMode ? "full" : "since",
        sinceRef: fullMode ? null : sinceRef,
        clientSrc,
        checkedComponents,
        cssVarsChecked,
        exemptions: [...exemptions],
        exemptedHits,
        findings,
      },
      null,
      2
    )
  );
  process.exit(findings.length === 0 ? 0 : 1);
}

console.log(`[diff-code-vs-docs] scope = ${scopeLabel}`);
console.log(`  client/src = ${clientSrc}`);
console.log(
  `  检查组件 ${checkedComponents.length} 个${checkCss ? " + index.css" : ""}` +
    (exemptedHits.length ? `；历史豁免跳过 ${exemptedHits.length} 个（${exemptedHits.join(", ")}）` : "")
);

if (findings.length === 0) {
  console.log(`\n✅ 无 P1 差集：代码枚举 / CSS 变量均已在文档中有对应声明。`);
  process.exit(0);
}

// 分组输出
const byType = new Map();
for (const f of findings) {
  if (!byType.has(f.type)) byType.set(f.type, []);
  byType.get(f.type).push(f);
}
const TYPE_TITLE = {
  CODE_ENUM_NOT_IN_DOC: "代码有枚举值、文档未提（P1 漏声明候选，以代码为准，逐条判定是否补文档）",
  CSS_VAR_NOT_IN_DOC: "index.css 有 CSS 变量、tokens/Tier1 未提（补 token 文档候选）",
  CHANGED_NO_SPEC: "改动碰到的组件无对应 spec（决策：碰到才纳管；确认后补 spec 骨架 或 加入豁免清单）",
  NO_SPEC_BACKLOG: "无对应 spec 的历史存量（仅 backlog 提示，不强制修）",
};

console.error(`\n发现 ${findings.length} 条 P1 差集候选（非判决，需 AI/人逐条确认）：`);
for (const [type, list] of byType.entries()) {
  console.error(`\n▸ ${type} — ${TYPE_TITLE[type] || ""}`);
  for (const f of list) {
    if (f.value && f.kind !== "css-var") {
      console.error(`    ${f.component}  ${f.kind} ${f.name} = "${f.value}"   (spec: ${f.spec})`);
    } else if (f.kind === "css-var") {
      console.error(`    ${f.value}`);
    } else {
      console.error(`    ${f.component}`);
    }
  }
}

console.error(
  "\n处理建议：\n" +
    "  · CODE_ENUM_NOT_IN_DOC：代码为准。若该枚举值是有效 API → 补进 spec/Tier1；若是内部实现细节 → 可忽略（人判定）。\n" +
    "  · CSS_VAR_NOT_IN_DOC：新增 token 补进 tokens/*.md；shadcn 兼容映射类可忽略。\n" +
    "  · CHANGED_NO_SPEC：给新组件补 spec 骨架，或（历史存量）加入 references/historical-exemption.json。\n" +
    "  · 误报：文档用了别名/换了措辞导致词边界匹配不到 → 人工放行即可。\n"
);

/* ─────────────────────────────────────────────────────────────────────────
 * LIMITATIONS（诚实声明，勿当银弹）：
 *  - 只查"代码→文档覆盖"正方向；反向（文档声称、代码已删）交给 AI 逐条判定。
 *  - 组件↔spec 靠 basename 同名匹配；改名/别名（如 SearchableSelect 在 select.tsx）
 *    属 P2/映射表范畴，本脚本不解析，交给 check-spec-symbols + 映射表。
 *  - 不提取大对象 keyof typeof（如 21 色 token map），避免噪声；未来可加白名单式扩展。
 *  - 覆盖判定是词边界字符串匹配，不理解语义；措辞差异可能误报，定位为"候选"而非"判决"。
 * ───────────────────────────────────────────────────────────────────────── */
process.exit(1);

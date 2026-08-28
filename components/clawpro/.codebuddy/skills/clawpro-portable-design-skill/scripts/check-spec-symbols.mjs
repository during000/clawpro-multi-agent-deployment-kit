#!/usr/bin/env node
/**
 * check-spec-symbols.mjs
 * ─────────────────────────────────────────────────────────────────────────
 * 目的：检测 ClawPro Portable Design Skill 内 spec 文档与真实代码仓的"漂移"。
 *
 * 历史背景：
 *   - 2026-06-11 排查发现 component-specs/combobox.md 描述了 438 行的「Combobox / OpenClawCombobox」组件，
 *     但仓库 client/src 内实际不存在 `combobox.tsx` / `<Combobox>` / `OpenClawCombobox`。真实组件叫
 *     `SearchableSelect`，挂在 `ui/select.tsx`。
 *   - 类似还有 spec 里出现 PROVIDER_VERSIONS / getCheckState / Label 这种死代码 / 未 import 的标识符。
 *   - 这是 skill 与 demo 仓 1:1 还原能力的硬伤：spec 是静态镜像，会随代码演进静默漂移。
 *
 * 这个脚本干什么：
 *   1. 扫描 component-specs/*.md + SKILL.md + references/*.md 所有 ```tsx / ```ts / ```jsx fenced code block。
 *   2. 在这些代码块里抽出 `import { A, B as C, type D } from "@/..."` 模式中的 identifier。
 *      - `@/...`、`./...`、`../...` 都视为"应在仓内可解析"
 *      - 第三方包 import (如 lucide-react、react、sonner、@radix-ui/...) 跳过
 *   3. 对每个 identifier，在配置的 search root 下 grep 一次：
 *        export {A, B}  /  export const A  /  export function A  /  export default function A
 *      命中即认为存在；否则记为 ghost reference。
 *   4. 输出 ghost references 列表 + 摘要（按 spec 文件分组），exit code 非零。
 *
 * 用法：
 *   node .codebuddy/skills/clawpro-portable-design-skill/scripts/check-spec-symbols.mjs
 *   node .codebuddy/skills/clawpro-portable-design-skill/scripts/check-spec-symbols.mjs --json
 *
 * 配置：
 *   通过环境变量 CLAW_REPO_SRC 指定仓库 client 源码根（默认猜测当前 git repo 的 client/src）。
 *   通过 --src=<path> 也能传。
 *
 * 设计选择：
 *   - 不解析 TS AST，纯正则；spec 是文档不是源码，正则鲁棒性已够，且不引入额外依赖。
 *   - 不报"代码块里出现了仓里不存在的标识符"，只报"被 import 的标识符不存在"，避免反例 / 旧名字误报。
 *   - 把 markdown 反例（紧邻 ❌ / 不要 / 旧写法 / 已删除 / 已废弃 / `已合并` 这类语义标记）排除在 import 集合外。
 *   - 输出尽量短：只显示 spec 路径 + identifier + 第一处 import 行号；要查具体上下文 grep 一下即可。
 * ─────────────────────────────────────────────────────────────────────────
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const skillRoot = path.resolve(__dirname, "..");

const args = process.argv.slice(2);
const jsonMode = args.includes("--json");
const srcArg = args.find((a) => a.startsWith("--src="));
const explicitSrc = srcArg ? srcArg.slice("--src=".length) : process.env.CLAW_REPO_SRC;

/* ───────────── 1. 定位 client/src 路径 ───────────── */

function findClientSrc() {
  if (explicitSrc) return path.resolve(explicitSrc);
  // skill 在 <repo>/.codebuddy/skills/clawpro-portable-design-skill/scripts/，往上回溯找 client/src
  let cur = skillRoot;
  for (let i = 0; i < 6; i++) {
    cur = path.dirname(cur);
    const candidate = path.join(cur, "client", "src");
    if (fs.existsSync(candidate)) return candidate;
  }
  return null;
}

const clientSrc = findClientSrc();
if (!clientSrc) {
  console.error(
    "[check-spec-symbols] 无法自动定位 client/src，请通过 --src=<absolute-path> 或环境变量 CLAW_REPO_SRC 指定。"
  );
  process.exit(2);
}

/* ───────────── 2. 收集 spec / 文档来源 ───────────── */

function collectMd(dir) {
  if (!fs.existsSync(dir)) return [];
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  const files = [];
  for (const e of entries) {
    if (e.name === ".DS_Store") continue;
    const abs = path.join(dir, e.name);
    if (e.isDirectory()) files.push(...collectMd(abs));
    else if (e.name.endsWith(".md")) files.push(abs);
  }
  return files;
}

const specRoots = [
  path.join(skillRoot, "SKILL.md"),
  ...collectMd(path.join(skillRoot, "component-specs")),
  ...collectMd(path.join(skillRoot, "references")),
];

/* ───────────── 3. 从 fenced code block 抽 import ───────────── */

const FENCE_RE = /```(tsx|ts|jsx)\s*\n([\s\S]*?)```/g;
const IMPORT_RE =
  /^\s*import\s+(?:type\s+)?(?:[A-Za-z_$][\w$]*\s*,\s*)?\{([^}]+)\}\s*from\s*["']([^"']+)["']/gm;
// 匹配完整 import { ... } from "..."；也覆盖 `import Default, { ... } from`

const NEGATIVE_HINTS = [
  "❌",
  "不要",
  "禁止",
  "已删除",
  "已废弃",
  "已合并",
  "旧写法",
  "alias",
  "不再",
  "幽灵",
  "ghost",
];

/** 判断某段 code block 文本是否处于反例上下文（避免把"❌ import { X }"也当真实引用）*/
function isNegativeContext(specText, codeBlockStartIdx) {
  // 取 code block 上方 200 字符
  const before = specText.slice(Math.max(0, codeBlockStartIdx - 200), codeBlockStartIdx);
  return NEGATIVE_HINTS.some((kw) => before.includes(kw));
}

/** 第三方包前缀：跳过这些 from 路径 */
function isThirdParty(from) {
  // @scoped/foo OR foo 或 foo/bar，且不是 @/...
  if (from.startsWith("@/")) return false;
  if (from.startsWith("./") || from.startsWith("../") || from.startsWith("/")) return false;
  // 任何不以 ./ / @/ / ../ 开头的都视为外部包
  return true;
}

/** 解析一个 import 子句 { A, B as C, type D } 拆成 identifier 数组 */
function parseImportSpecifiers(braceContent) {
  return braceContent
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean)
    .map((s) => {
      // "type Foo" / "type Foo as Bar"
      let raw = s.replace(/^type\s+/, "");
      // "Foo as Bar" -> 取 Foo（实际被引用的）
      raw = raw.split(/\s+as\s+/)[0];
      return raw.trim();
    })
    .filter((id) => /^[A-Za-z_$][\w$]*$/.test(id));
}

/* ───────────── 4. 在 client/src 内查 identifier 是否被 export ─────────────
 *
 * 第一版用 `grep -E` 单行匹配 `export { Foo, Bar }`，但项目里多数组件
 * 用「`function Foo(){}` + 文件末尾跨行 `export { Foo, Bar, ... }`」的格式，
 * 跨行块单行正则匹配不到，导致 Table / Empty / AdminSidebar 等真实组件被
 * 误报为 ghost。
 *
 * 改用："一次性扫一遍 client/src 里所有 .ts/.tsx，建立 exported name 集合"。
 * 涵盖以下所有 export 形态：
 *   export const X       export let X        export var X
 *   export function X    export async function X
 *   export class X       export interface X  export type X    export enum X
 *   export default function X     export default class X
 *   export { X, Y as Z }                          // 单行
 *   export {\n  X,\n  Y as Z,\n}                  // 跨行
 *   export type { X, Y }
 *   export * as X from "..."
 *   re-export 不展开（`export * from "./foo"`），如有需要可后续递归。
 */

const exportedSymbols = new Set();

function walkSrc(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === ".DS_Store" || entry.name === "node_modules") continue;
    const abs = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walkSrc(abs);
      continue;
    }
    if (!/\.(ts|tsx|js|jsx)$/.test(entry.name)) continue;
    let text;
    try {
      text = fs.readFileSync(abs, "utf8");
    } catch {
      continue;
    }
    collectExportsFromText(text);
  }
}

const SINGLE_LINE_DECL_RE =
  /^\s*export\s+(?:default\s+)?(?:async\s+)?(const|let|var|function|class|interface|type|enum)\s+([A-Za-z_$][\w$]*)/gm;
const STAR_AS_RE = /^\s*export\s+\*\s+as\s+([A-Za-z_$][\w$]*)\s+from/gm;
const EXPORT_BLOCK_RE = /export\s+(?:type\s+)?\{([^}]*)\}/g; // 跨行也行（[^}]* 不含 } 即可）

function collectExportsFromText(text) {
  let m;
  SINGLE_LINE_DECL_RE.lastIndex = 0;
  while ((m = SINGLE_LINE_DECL_RE.exec(text)) !== null) {
    exportedSymbols.add(m[2]);
  }
  STAR_AS_RE.lastIndex = 0;
  while ((m = STAR_AS_RE.exec(text)) !== null) {
    exportedSymbols.add(m[1]);
  }
  EXPORT_BLOCK_RE.lastIndex = 0;
  while ((m = EXPORT_BLOCK_RE.exec(text)) !== null) {
    // export 块体可能含有行注释（`// 用户端胶囊`）和块注释（`/* ... */`），
    // 不剥掉的话 split(",") 后会把注释和后面的标识符黏在一起。
    const body = m[1]
      .replace(/\/\*[\s\S]*?\*\//g, "")
      .replace(/\/\/[^\n]*/g, "");
    for (const piece of body.split(",")) {
      const part = piece.trim().replace(/^type\s+/, "");
      if (!part) continue;
      // "Foo as Bar" -> 取 Bar（外部能 import 的名字）
      const asMatch = /\bas\s+([A-Za-z_$][\w$]*)\s*$/.exec(part);
      if (asMatch) {
        exportedSymbols.add(asMatch[1]);
        continue;
      }
      const idMatch = /^([A-Za-z_$][\w$]*)$/.exec(part);
      if (idMatch) exportedSymbols.add(idMatch[1]);
    }
  }
}

walkSrc(clientSrc);

function isExportedInRepo(id) {
  return exportedSymbols.has(id);
}

/* ───────────── 5. 主流程 ───────────── */

const ghosts = []; // { spec, lineHint, identifier, fromPath }

for (const specPath of specRoots) {
  const text = fs.readFileSync(specPath, "utf8");
  let blockMatch;
  while ((blockMatch = FENCE_RE.exec(text)) !== null) {
    const blockText = blockMatch[2];
    const blockStart = blockMatch.index;
    if (isNegativeContext(text, blockStart)) continue;

    let importMatch;
    IMPORT_RE.lastIndex = 0;
    while ((importMatch = IMPORT_RE.exec(blockText)) !== null) {
      const braceContent = importMatch[1];
      const fromPath = importMatch[2];
      if (isThirdParty(fromPath)) continue;
      const ids = parseImportSpecifiers(braceContent);
      for (const id of ids) {
        if (!isExportedInRepo(id)) {
          // line hint: 计算到 spec 文档里大致行号
          const upToBlock = text.slice(0, blockStart).split("\n").length;
          const insideBlock = blockText.slice(0, importMatch.index).split("\n").length;
          ghosts.push({
            spec: path.relative(skillRoot, specPath),
            lineHint: upToBlock + insideBlock,
            identifier: id,
            fromPath,
          });
        }
      }
    }
  }
}

/* ───────────── 6. 输出 ───────────── */

if (jsonMode) {
  console.log(JSON.stringify({ clientSrc, ghosts }, null, 2));
  process.exit(ghosts.length === 0 ? 0 : 1);
}

if (ghosts.length === 0) {
  console.log(`[check-spec-symbols] OK · 没有 ghost identifier。client/src = ${clientSrc}`);
  process.exit(0);
}

console.error(
  `[check-spec-symbols] 发现 ${ghosts.length} 个 ghost identifier（spec 里 import 但 client/src 不导出）：`
);
console.error(`  client/src = ${clientSrc}\n`);

const bySpec = new Map();
for (const g of ghosts) {
  if (!bySpec.has(g.spec)) bySpec.set(g.spec, []);
  bySpec.get(g.spec).push(g);
}
for (const [spec, list] of [...bySpec.entries()].sort()) {
  console.error(`▸ ${spec}`);
  for (const g of list) {
    console.error(`    L${g.lineHint}  { ${g.identifier} } from "${g.fromPath}"`);
  }
}

console.error(
  "\n处理建议：\n" +
    "  · 如果是改名（旧 → 新组件），把 spec 内 import 改成新名字（如 Combobox → SearchableSelect）。\n" +
    "  · 如果代码已删除，删除/瘦身 spec 描述，避免误导。\n" +
    "  · 如果是 spec 的反例（应该写在 ❌ 后面），把它从 ```tsx 实例迁到反例段，并加 ❌/不要/已废弃 等关键字。\n" +
    "  · 如果你确认是脚本误报，调整 NEGATIVE_HINTS 或在 import 上方加上 // SPEC-IGNORE 注释。"
);

process.exit(1);

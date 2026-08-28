#!/usr/bin/env node
/**
 * sync-tokens.mjs
 * ─────────────────────────────────────────────────────────────────────────
 * 目的：保证 spec 文档里提到的 token 名与 client/src 的真实定义同步。
 *
 * 场景：
 *   - spec 写"圆角用 --cp-radius-md"，但实际 index.css 里没这个变量名（改成了 --radius）
 *   - spec 写"背景用 --cp-surface-muted"，但代码里改名了
 *   - Portable Fallback 代码里用的 token 后来被删除了
 *   这个脚本能自动检测并告警。
 *
 * 工作流程：
 *   1. 扫 client/src/index.css，抽出所有 CSS 变量定义（--cp-* / --radius / --bg-grey-* 等）
 *   2. 写到 tokens/design-tokens.json，做一份"golden copy"
 *   3. 可选：对 spec markdown 里的所有 `var(--xxx)` 做检查，确保都在 JSON 里
 *   4. 如果有未知 token，输出 warning
 *
 * 用法：
 *   node .codebuddy/skills/clawpro-portable-design-skill/scripts/sync-tokens.mjs
 *   node .codebuddy/skills/clawpro-portable-design-skill/scripts/sync-tokens.mjs --check-spec
 *   node .codebuddy/skills/clawpro-portable-design-skill/scripts/sync-tokens.mjs --from-css <path>
 *
 * 输出：
 *   - tokens/design-tokens.json 更新（或首次创建）
 *   - 如果 --check-spec，对 spec markdown 里的 token 做检查
 * ─────────────────────────────────────────────────────────────────────────
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const skillRoot = path.resolve(__dirname, "..");

const args = process.argv.slice(2);
const checkSpec = args.includes("--check-spec");
const cssArg = args.find((a) => a.startsWith("--from-css="));
const cssPath = cssArg ? cssArg.slice("--from-css=".length) : null;

/* ───────────── 1. 定位 client/src/index.css ───────────── */

function findIndexCss() {
  if (cssPath) return path.resolve(cssPath);
  let cur = skillRoot;
  for (let i = 0; i < 6; i++) {
    cur = path.dirname(cur);
    const candidate = path.join(cur, "client", "src", "index.css");
    if (fs.existsSync(candidate)) return candidate;
  }
  return null;
}

const indexCss = findIndexCss();
if (!indexCss) {
  console.error("[sync-tokens] 无法定位 client/src/index.css");
  process.exit(2);
}

/* ───────────── 2. 从 index.css 抽 token ───────────── */

const css = fs.readFileSync(indexCss, "utf8");

// 正则：--token-name: value;
const tokenRe = /--([\w-]+):\s*([^;]+);/g;
const tokens = {};
let m;
while ((m = tokenRe.exec(css)) !== null) {
  const name = m[1];
  const value = m[2].trim();
  tokens[name] = value;
}

console.log(`[sync-tokens] 从 ${indexCss} 抽取 ${Object.keys(tokens).length} 个 token`);

/* ───────────── 3. 写到 design-tokens.json ───────────── */

const tokenJsonPath = path.join(skillRoot, "tokens", "design-tokens.json");
const tokenJson = {
  version: new Date().toISOString(),
  sourceFile: indexCss,
  count: Object.keys(tokens).length,
  tokens: tokens,
};

fs.writeFileSync(tokenJsonPath, JSON.stringify(tokenJson, null, 2));
console.log(`[sync-tokens] 写入 ${tokenJsonPath}`);

/* ───────────── 4. 可选：检查 spec markdown 里的 token 引用 ───────────── */

if (checkSpec) {
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

  const varRe = /var\(\s*--([\w-]+)\s*\)/g;
  const usedTokens = new Set();
  const unknownTokens = [];

  for (const spec of specRoots) {
    const text = fs.readFileSync(spec, "utf8");
    let vm;
    while ((vm = varRe.exec(text)) !== null) {
      const tokenName = vm[1];
      usedTokens.add(tokenName);
      if (!tokens[tokenName]) {
        unknownTokens.push({
          spec: path.relative(skillRoot, spec),
          token: `--${tokenName}`,
        });
      }
    }
  }

  console.log(`\n[sync-tokens] spec 里提到的 token：${usedTokens.size} 个`);

  if (unknownTokens.length === 0) {
    console.log("[sync-tokens] ✓ 所有 token 引用都有效");
    process.exit(0);
  }

  console.error("\n[sync-tokens] ✗ 发现未定义的 token：");
  const byToken = new Map();
  for (const item of unknownTokens) {
    if (!byToken.has(item.token)) byToken.set(item.token, []);
    byToken.get(item.token).push(item.spec);
  }
  for (const [token, specs] of [...byToken.entries()].sort()) {
    console.error(`  ${token}`);
    for (const spec of specs.slice(0, 3)) console.error(`    ← ${spec}`);
    if (specs.length > 3) console.error(`    ... 及 ${specs.length - 3} 处`);
  }

  console.error("\n处理建议：");
  console.error("  · 检查 token 名是否拼错（如 --cp-radius-md vs --radius）");
  console.error("  · 如果是旧 token 改名了，更新 spec 里的引用");
  console.error("  · 如果 token 被删除了，删除或替换 spec 里的使用");

  process.exit(unknownTokens.length === 0 ? 0 : 1);
}

console.log("[sync-tokens] OK");

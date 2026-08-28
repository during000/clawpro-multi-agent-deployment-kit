#!/usr/bin/env node
/**
 * extract-page-recipes.mjs
 *
 * 抽 clawpro-portable-design-skill/assets/page-references/*.md
 *   → fixtures/page-recipes.json
 *
 * 每条 recipe 记录：
 *   - id / name / file / sha256 / png（同目录截图）
 *   - kind: 列表页 / 表单页 / Dashboard / 详情页 …（从 H1 或顶部 blockquote 抓）
 *   - route: `/admin/xxx`（从 blockquote `路由: ...` 抓）
 *   - source: 关联的 demo 仓源码相对路径（可能多条）
 *   - required_components: §2 组件清单表里的组件 import 路径集合
 *   - required_specs: §2 组件清单表里指向的 component-specs/*.md 文件名集合
 *   - skeleton_text: §1 视觉骨架代码块原文（供 page-recipe-match detector 做关键词匹配）
 *   - antipatterns: §5 易错点表里的 ❌ 文案集合
 *
 * 设计意图：这是"换皮防线"的弹药库——后续 page-recipe-match detector
 *   会拿 required_components / required_specs / antipatterns 三组事实
 *   去比对宿主仓某个页面的实际 import 与样式，发现"骨架缺件"或"踩反例"。
 */

import { readFileSync, writeFileSync, readdirSync, statSync } from "node:fs";
import { createHash } from "node:crypto";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const CORE_ROOT = resolve(__dirname, "..");
const RECIPES_DIR = resolve(
  CORE_ROOT,
  "../clawpro-portable-design-skill/assets/page-references"
);
const OUT_PATH = resolve(CORE_ROOT, "fixtures/page-recipes.json");

function sha256(text) {
  return createHash("sha256").update(text).digest("hex");
}

function extractName(md) {
  const m = md.match(/^#\s+(.+?)\s*$/m);
  return m ? m[1].trim() : null;
}

/** 从 H1 / blockquote 抓"类别" */
function extractKind(md) {
  // H1 形如：# 列表页 · 用户管理 (admin/members)
  const h1 = md.match(/^#\s+([^·\n]+?)\s*·/m);
  if (h1) return h1[1].trim();
  // blockquote 形如：> 类别：**列表页**
  const bq = md.match(/类别[：:]\s*\*?\*?([^\s*\n（(]+)/);
  return bq ? bq[1].trim() : null;
}

function extractRoute(md) {
  const m = md.match(/路由[：:]\s*`([^`]+)`/);
  return m ? m[1].trim() : null;
}

function extractSource(md) {
  // 形如：源码：`client/src/pages/xxx.tsx`
  const out = new Set();
  const re = /源码[：:]\s*`([^`]+)`/g;
  let m;
  while ((m = re.exec(md))) out.add(m[1].trim());
  return [...out];
}

/** §1 视觉骨架的 ``` 代码块原文 */
function extractSkeleton(md) {
  const sec = sliceSection(md, /^##\s+1\./m);
  if (!sec) return null;
  const m = sec.match(/```([\s\S]+?)```/);
  return m ? m[1].trim() : null;
}

/** §2 组件清单表里：import 列 + spec 列 */
function extractRequirements(md) {
  const sec = sliceSection(md, /^##\s+2\./m);
  if (!sec) return { components: [], specs: [] };
  const components = new Set();
  const specs = new Set();
  // import: `@/components/ui/xxx` （兼容大小写驼峰，如 Typography）
  const reImp = /`(@\/components\/ui\/[A-Za-z0-9-]+)`/g;
  let m;
  while ((m = reImp.exec(sec))) components.add(m[1]);
  // spec: `component-specs/xxx.md`
  const reSpec = /`component-specs\/([a-z0-9-]+)\.md`/g;
  while ((m = reSpec.exec(sec))) specs.add(m[1]);
  return {
    components: [...components].sort(),
    specs: [...specs].sort(),
  };
}

/** §5 易错点表第一列 ❌ 文案 */
function extractAntipatterns(md) {
  const sec = sliceSection(md, /^##\s+5\./m);
  if (!sec) return [];
  const out = [];
  const lines = sec.split(/\r?\n/);
  let inTable = false;
  let sawDivider = false;
  for (const line of lines) {
    if (!line.trim().startsWith("|")) {
      inTable = false;
      sawDivider = false;
      continue;
    }
    // 表头：含 ❌ 或 "反例" 字样
    if (/❌|反例/.test(line) && !sawDivider) {
      inTable = true;
      continue;
    }
    // 分隔行：| --- | --- |
    if (inTable && /^\|\s*[-:|\s]+$/.test(line.trim())) {
      sawDivider = true;
      continue;
    }
    if (inTable && sawDivider) {
      // 取第一格
      const cells = line.split("|").map((c) => c.trim()).filter(Boolean);
      if (cells.length >= 2) {
        const bad = cells[0].replace(/^❌\s*/, "").replace(/`/g, "").trim();
        if (bad) out.push(bad);
      }
    }
  }
  return out;
}

function sliceSection(md, startRe) {
  const m = md.match(startRe);
  if (!m) return null;
  const start = m.index + m[0].length;
  const rest = md.slice(start);
  const nextH2 = rest.search(/^##\s+/m);
  return nextH2 === -1 ? rest : rest.slice(0, nextH2);
}

function main() {
  const files = readdirSync(RECIPES_DIR)
    .filter((f) => f.endsWith(".md") && f !== "README.md")
    .sort();

  const items = [];
  let totalSha = createHash("sha256");

  for (const f of files) {
    const abs = join(RECIPES_DIR, f);
    if (!statSync(abs).isFile()) continue;
    const raw = readFileSync(abs, "utf8");
    const id = f.replace(/\.md$/, "");
    const png = f.replace(/\.md$/, ".png");
    const req = extractRequirements(raw);
    const item = {
      id,
      file: `assets/page-references/${f}`,
      png: `assets/page-references/${png}`,
      name: extractName(raw) || id,
      sha256: sha256(raw),
      kind: extractKind(raw),
      route: extractRoute(raw),
      source: extractSource(raw),
      required_components: req.components,
      required_specs: req.specs,
      skeleton: extractSkeleton(raw),
      antipatterns: extractAntipatterns(raw),
      bytes: raw.length,
    };
    items.push(item);
    totalSha.update(item.sha256);
  }

  const out = {
    schema: "clawpro.page-recipes/v1",
    source_root:
      ".codebuddy/skills/clawpro-portable-design-skill/assets/page-references",
    generated_at: new Date().toISOString(),
    aggregate_sha256: totalSha.digest("hex"),
    count: items.length,
    items,
  };

  writeFileSync(OUT_PATH, JSON.stringify(out, null, 2) + "\n", "utf8");

  const totalReq = items.reduce(
    (s, i) => s + i.required_components.length,
    0
  );
  const totalAnti = items.reduce((s, i) => s + i.antipatterns.length, 0);
  console.log(`[extract-page-recipes] ✓ ${items.length} recipes → ${OUT_PATH}`);
  console.log(
    `[extract-page-recipes]   required-components=${totalReq}  antipatterns=${totalAnti}`
  );
}

main();

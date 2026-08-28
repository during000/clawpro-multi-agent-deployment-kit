/**
 * page-recipe-match.mjs
 *
 * 把"页面级骨架完整性"接入走查：
 *   对每条 page-recipe，检查 recipe.source[0] 对应的 page entry 文件
 *   （+ 同目录递归子组件文件）是否真的 import 了 recipe.required_components 里的全部组件。
 *
 * 数据源（真理源）：
 *   clawpro-walkthrough/fixtures/page-recipes.json
 *
 * v0.1 严守"宁可漏报不可误报"：
 *   - 只对 recipe.source 中真实存在的文件触发
 *   - 把 entry 文件 + 同目录递归 .tsx/.ts/.jsx/.js 视为一个 "page bundle"，
 *     联合统计所有 import 路径，避免父子拆分导致的误报。
 *   - required_components 里**没有**出现在 bundle 任一 import 里 → P2 "page-recipe-match"
 *   - antipatterns / required_specs / skeleton 关键词匹配 → 不在 v0.1 范畴（误报率高）
 *
 * 触发方式：
 *   detector 是文件级 run({file, text})，但本规则需要"按页面整体"判断。
 *   做法：仅当当前扫描的 file 与某条 recipe 的 source[0] 完全一致时，
 *   才在此文件上下文里报出"该页面骨架缺件"——避免同一缺件在 bundle 子文件上重复抛。
 *   这样保证：每条 recipe 在一次 audit 里最多产生 N 条 finding（N=missing 数）。
 *
 * Severity：P2（提示级）
 *   缺件不一定是错——recipe 是"应当骨架"，业务可能合理删减。
 *   留 design-todo 由人裁决；后续 recipe 收敛稳定后可升 P1。
 */

import { readFileSync, existsSync, readdirSync, statSync } from 'node:fs';
import { dirname, resolve, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { makeFinding } from './_shared.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const RECIPES_PATH = resolve(
  __dirname,
  '../../fixtures/page-recipes.json',
);
// 仓库根：默认按标准安装位置反推，可用 WALKTHROUGH_PROJECT_ROOT 覆盖（与 walkthrough.mjs 一致）。
const PROJECT_ROOT = process.env.WALKTHROUGH_PROJECT_ROOT
  ? resolve(process.env.WALKTHROUGH_PROJECT_ROOT)
  : resolve(__dirname, '../../../../..');

const RULE_ID = 'page-recipe-match';
const SCAN_EXTS = new Set(['.tsx', '.ts', '.jsx', '.js']);

/** 懒加载 recipes（同进程多次调用 run() 复用） */
let _recipesBySource = null;
function loadRecipes() {
  if (_recipesBySource) return _recipesBySource;
  const raw = JSON.parse(readFileSync(RECIPES_PATH, 'utf8'));
  _recipesBySource = new Map();
  for (const it of raw.items || []) {
    for (const s of it.source || []) {
      // 标准化为 posix-style 相对路径
      _recipesBySource.set(s.replace(/\\/g, '/'), it);
    }
  }
  return _recipesBySource;
}

/** 收集 bundle 内所有 import 路径 */
const _bundleImportCache = new Map();
function collectBundleImports(entryAbsFile) {
  if (_bundleImportCache.has(entryAbsFile)) return _bundleImportCache.get(entryAbsFile);
  const root = dirname(entryAbsFile);
  const importPaths = new Set();
  const stack = [root];
  while (stack.length) {
    const cur = stack.pop();
    let entries;
    try {
      entries = readdirSync(cur);
    } catch {
      continue;
    }
    for (const name of entries) {
      const p = resolve(cur, name);
      let st;
      try {
        st = statSync(p);
      } catch {
        continue;
      }
      if (st.isDirectory()) {
        stack.push(p);
      } else if (st.isFile()) {
        const ext = p.slice(p.lastIndexOf('.'));
        if (!SCAN_EXTS.has(ext)) continue;
        let text;
        try {
          text = readFileSync(p, 'utf8');
        } catch {
          continue;
        }
        const re = /import\s+(?:[^'";]+?\s+from\s+)?['"]([^'"]+)['"]/g;
        let m;
        while ((m = re.exec(text)) !== null) importPaths.add(m[1]);
      }
    }
  }
  _bundleImportCache.set(entryAbsFile, importPaths);
  return importPaths;
}

/**
 * 标准化 import 路径：strip query / hash / trailing slash，并把 `@/components/ui/button.tsx` 归一为 `@/components/ui/button`
 */
function normalizeImport(p) {
  return p
    .replace(/[?#].*$/, '')
    .replace(/\/+$/, '')
    .replace(/\.(tsx|ts|jsx|js|mjs|cjs)$/i, '');
}

export function run({ file, text: _text }) {
  // file 是相对 PROJECT_ROOT 的 posix 路径（walkthrough.mjs 给的）
  const recipes = loadRecipes();
  const recipe = recipes.get(file);
  if (!recipe) return [];

  const required = (recipe.required_components || []).map(normalizeImport);
  if (required.length === 0) return [];

  const entryAbs = resolve(PROJECT_ROOT, file);
  if (!existsSync(entryAbs)) return [];

  const bundleImports = collectBundleImports(entryAbs);
  const normalizedBundle = new Set();
  for (const p of bundleImports) normalizedBundle.add(normalizeImport(p));

  const missing = required.filter((p) => !normalizedBundle.has(p));
  if (missing.length === 0) return [];

  const findings = [];
  // 每个缺件一条 finding，定位锚到文件首行（page 级问题没有具体 token 位置）
  for (const m of missing) {
    findings.push(
      makeFinding({
        ruleId: RULE_ID,
        severity: 'P2',
        file,
        line: 1,
        col: 1,
        snippet: `recipe=${recipe.id}`,
        message: `页面骨架缺件：page-recipe[${recipe.id}] 要求引入 ${m}，但当前 page bundle 未发现此 import`,
        evidence: `page-recipes.json#${recipe.id}.required_components`,
        suggestion: `参照 ${recipe.file} §1 视觉骨架与 §2 组件清单，确认是否需要补齐 ${m}，或在 page-references 里调整 required_components`,
      }),
    );
  }
  return findings;
}

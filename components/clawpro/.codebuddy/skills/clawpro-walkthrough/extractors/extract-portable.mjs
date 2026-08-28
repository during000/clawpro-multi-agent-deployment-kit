#!/usr/bin/env node
/**
 * extract-portable.mjs
 *
 * 把 clawpro-portable-design-skill/portable/{react,css} 下所有"正版实现"
 * 抽成机器可读 fixture：fixtures/portable-impl-index.json
 *
 * 作为后续 portable-impl-diff detector（待实现）的"正版真相源"。
 * 用动态枚举，新增组件（tree-select / file-browser / chart ...）无需手工登记。
 */
import { readFileSync, writeFileSync, readdirSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, relative } from "node:path";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = join(__dirname, "..", "..");
const PORTABLE = join(ROOT, "clawpro-portable-design-skill", "portable");
const OUT = join(__dirname, "..", "fixtures", "portable-impl-index.json");

function idFromFile(absPath) {
  return absPath.split("/").pop().replace(/\.(tsx|css)$/, "");
}

function listByExt(dir, ext) {
  if (!existsSync(dir)) return [];
  return readdirSync(dir)
    .filter((f) => f.endsWith(ext))
    .sort()
    .map((f) => join(dir, f));
}

function parseReactModule(src) {
  // 1) default export name
  const defaultMatch = src.match(/export\s+default\s+function\s+(\w+)/);
  const defaultArrow = src.match(/export\s+default\s+(\w+)\s*[=;]/);
  const defaultExport = defaultMatch?.[1] ?? defaultArrow?.[1] ?? null;

  // 2) named exports
  const named = [
    ...src.matchAll(/export\s+(?:const|function|class)\s+(\w+)/g),
    ...src.matchAll(/export\s*\{\s*([^}]+)\s*\}/g),
  ]
    .map((m) => (m[1].includes(",") ? m[1] : m[1]))
    .flatMap((s) => s.split(",").map((x) => x.trim().split(/\s+as\s+/)[0]))
    .filter((n) => /^[A-Z]/.test(n) && n !== "default");

  // 3) cp-* classNames referenced
  const classSet = new Set();
  const classRe = /\bcp-[a-z0-9]+(?:[-_]+[a-z0-9]+)*/g;
  let m;
  while ((m = classRe.exec(src))) classSet.add(m[0]);

  // 4) inline style count (rough: 出现 style={{ 的次数)
  const inlineStyleCount = (src.match(/style=\{\{/g) || []).length;

  // 5) file header
  const header = src
    .split("\n")
    .slice(0, 12)
    .filter((l) => l.startsWith("*") || l.startsWith("/*"))
    .join("\n");

  return {
    defaultExport,
    exports: [...new Set(named)].sort(),
    classNames: [...classSet].sort(),
    inline_style_count: inlineStyleCount,
    header_excerpt: header.slice(0, 280),
  };
}

function parseCssModule(src) {
  // count selectors
  const selectors = new Set();
  const selRe = /\.cp-[a-z0-9]+(?:[-_]+[a-z0-9]+)*/g;
  let m;
  while ((m = selRe.exec(src))) selectors.add(m[0]);

  // root rule selectors (bare .cp-... 1 段, 非 BEM 子级)
  const rootRules = [...selectors].filter((c) => !c.includes("__") && !c.includes("--"));

  return {
    selector_count: selectors.size,
    root_selectors: rootRules.sort(),
    bytes: src.length,
  };
}

const reactFiles = listByExt(join(PORTABLE, "react"), ".tsx");
const cssFiles = listByExt(join(PORTABLE, "css"), ".css");

const react = reactFiles.map((abs) => {
  const id = idFromFile(abs);
  const src = readFileSync(abs, "utf8");
  return {
    id,
    file: relative(ROOT, abs),
    ...parseReactModule(src),
  };
});

const css = cssFiles.map((abs) => {
  const id = idFromFile(abs);
  const src = readFileSync(abs, "utf8");
  return {
    id,
    file: relative(ROOT, abs),
    ...parseCssModule(src),
  };
});

// classOwners: className -> [css file ids that define it]
const classOwners = {};
for (const c of css) {
  const src = readFileSync(join(ROOT, c.file), "utf8");
  for (const sel of src.matchAll(/\.(cp-[a-z0-9]+(?:[-_]+[a-z0-9]+)*)/g)) {
    const k = sel[1];
    (classOwners[k] ??= []).push(c.id);
  }
}
for (const k of Object.keys(classOwners)) {
  classOwners[k] = [...new Set(classOwners[k])].sort();
}

// cp-classes 全集
const allClasses = new Set();
for (const r of react) for (const c of r.classNames) allClasses.add(c);
for (const c of css) {
  const src = readFileSync(join(ROOT, c.file), "utf8");
  for (const m of src.matchAll(/\.(cp-[a-z0-9]+(?:[-_]+[a-z0-9]+)*)/g)) allClasses.add(m[1]);
}

const out = {
  schema_version: 1,
  generated_at: new Date().toISOString(),
  source: {
    portable_react_dir: relative(ROOT, join(PORTABLE, "react")),
    portable_css_dir: relative(ROOT, join(PORTABLE, "css")),
  },
  summary: {
    react_files: react.length,
    css_files: css.length,
    cp_classes: allClasses.size,
  },
  react,
  css,
  classOwners,
  cp_classes: [...allClasses].sort(),
};

writeFileSync(OUT, JSON.stringify(out, null, 2) + "\n", "utf8");
console.log(
  `[extract-portable] react=${react.length} css=${css.length} cp-classes=${allClasses.size} -> ${relative(
    ROOT,
    OUT,
  )}`,
);

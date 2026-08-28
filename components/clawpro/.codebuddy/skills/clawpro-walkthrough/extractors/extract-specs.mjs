#!/usr/bin/env node
/**
 * extract-specs.mjs
 *
 * 抽 clawpro-portable-design-skill/component-specs/*.md
 *   → fixtures/component-spec-index.json
 *
 * 产出每个组件 spec 的可机读摘要：
 *   - id / name / file / sha256
 *   - sections: 哪几节有内容（§3 Visual / §4 Anatomy / §7 Portable / §10 QA …）
 *   - imports: 从 §6 / §11 / 代码块里抓 "@/components/ui/xxx" 的 import 路径
 *   - variants: §3 Visual Standard 表里抓出来的 variant 名（首列）
 *   - radius_hint: 在 §3 表里出现的 "4px" / "full" 计数（给 radius detector 复用）
 *   - has_portable_fallback: §7 是否真有 fallback 代码块
 *   - qa_checks: §10 QA Checklist 条目数
 *
 * 不解释、不重组、不发明字段——只抽事实。任何 finding 必须 evidence
 * 指回 component-spec-index.json#<id> 或更具体子字段。
 */

import { readFileSync, writeFileSync, readdirSync, statSync } from "node:fs";
import { createHash } from "node:crypto";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const CORE_ROOT = resolve(__dirname, "..");
const SPECS_DIR = resolve(
  CORE_ROOT,
  "../clawpro-portable-design-skill/component-specs"
);
const OUT_PATH = resolve(CORE_ROOT, "fixtures/component-spec-index.json");

function sha256(text) {
  return createHash("sha256").update(text).digest("hex");
}

/** 抓 H1 (# Xxx)，作为 spec name */
function extractName(md) {
  const m = md.match(/^#\s+(.+?)\s*$/m);
  return m ? m[1].trim() : null;
}

/** 列出所有 import 形式："@/components/ui/xxx" */
function extractImports(md) {
  const set = new Set();
  const re = /["'`](@\/components\/ui\/[A-Za-z0-9-]+)["'`]/g;
  let m;
  while ((m = re.exec(md))) set.add(m[1]);
  // 也抓 markdown 表格里裸写的 `@/components/ui/xxx`
  const re2 = /`(@\/components\/ui\/[A-Za-z0-9-]+)`/g;
  while ((m = re2.exec(md))) set.add(m[1]);
  return [...set].sort();
}

/**
 * 抓 variants —— 双通道：
 *   通道 A：全文扫描 `variant="xxx"` / `variant: "xxx"` 字面值（代码示例最稳）
 *   通道 B：§3 Visual Standard 表格首列的 `\`xxx\`` （表头规范常用 markdown
 *           inline-code 形式登记 variant 名，例如 `| \`claw-primary\` | 4px | ...`）
 *
 * 表头自身的字段名（variant / token / 维度 / 状态 ...）会被剔除。
 */
function extractVariants(md) {
  const variants = new Set();
  // 通道 A
  const reA = /variant\s*[:=]\s*["'`]([a-z][a-z0-9-]*)["'`]/g;
  let m;
  while ((m = reA.exec(md))) variants.add(m[1]);

  // 通道 B：§3 Visual Standard 表格首列里的 inline-code
  const visualSection = sliceSection(md, /^##\s+3\.\s+Visual Standard/m);
  if (visualSection) {
    const lines = visualSection.split(/\r?\n/);
    let inTable = false;
    let sawDivider = false;
    for (const line of lines) {
      const t = line.trim();
      if (!t.startsWith('|')) {
        inTable = false;
        sawDivider = false;
        continue;
      }
      if (!inTable) {
        inTable = true;
        continue; // 表头行
      }
      if (!sawDivider) {
        // 期望是 `| --- | --- | ...`
        if (/^\|\s*[-:|\s]+$/.test(t)) sawDivider = true;
        continue;
      }
      // 数据行：取首列
      const cells = t.split('|').map((c) => c.trim()).filter((_, i, arr) => i > 0 && i < arr.length - 1);
      if (cells.length === 0) continue;
      // 1) 优先抓首列里反引号包裹的 inline-code（最稳：`gray-header`（默认）、`tenant-outline-r20` 都能命中）
      const inline = cells[0].match(/`([a-z][a-z0-9-]*)`/);
      if (inline) {
        variants.add(inline[1]);
        continue;
      }
      // 2) 兜底：整列就是 kebab-case 标识（无反引号也无中文备注）
      const first = cells[0].trim();
      if (/^[a-z][a-z0-9-]*$/.test(first)) {
        variants.add(first);
      }
    }
  }

  return [...variants].sort();
}

/** 数 4px / full 在 §3 出现次数（给 radius detector 复用作权重） */
function extractRadiusHint(md) {
  const visualSection = sliceSection(md, /^##\s+3\.\s+Visual Standard/m) || "";
  const fourPx = (visualSection.match(/\b4px\b/g) || []).length;
  const full = (visualSection.match(/\b(full|9999px|rounded-full)\b/g) || [])
    .length;
  return { "4px": fourPx, full };
}

function hasPortableFallback(md) {
  const sec = sliceSection(md, /^##\s+7\.\s+Portable Fallback/m);
  if (!sec) return false;
  // 至少包含一个 ```tsx 或 ```css 代码块
  return /```(tsx|jsx|css|html)/.test(sec);
}

function countQaChecks(md) {
  const sec = sliceSection(md, /^##\s+10\.\s+QA Checklist/m);
  if (!sec) return 0;
  return (sec.match(/^\s*-\s+\[\s?\]/gm) || []).length;
}

/** 把 md 从某个 ## 标题切到下一个 ## 标题之间 */
function sliceSection(md, startRe) {
  const m = md.match(startRe);
  if (!m) return null;
  const start = m.index + m[0].length;
  const rest = md.slice(start);
  const nextH2 = rest.search(/^##\s+/m);
  return nextH2 === -1 ? rest : rest.slice(0, nextH2);
}

/** 哪些章节存在（按 markdown ## H2 抓） */
function listSections(md) {
  const headings = [];
  const re = /^##\s+(.+?)\s*$/gm;
  let m;
  while ((m = re.exec(md))) headings.push(m[1].trim());
  return headings;
}

function main() {
  const files = readdirSync(SPECS_DIR)
    .filter((f) => f.endsWith(".md"))
    .sort();

  const items = [];
  let totalSha = createHash("sha256");

  for (const f of files) {
    const abs = join(SPECS_DIR, f);
    if (!statSync(abs).isFile()) continue;
    const raw = readFileSync(abs, "utf8");
    const id = f.replace(/\.md$/, "");
    const item = {
      id,
      file: `component-specs/${f}`,
      name: extractName(raw) || id,
      sha256: sha256(raw),
      sections: listSections(raw),
      imports: extractImports(raw),
      variants: extractVariants(raw),
      radius_hint: extractRadiusHint(raw),
      has_portable_fallback: hasPortableFallback(raw),
      qa_checks: countQaChecks(raw),
      bytes: raw.length,
    };
    items.push(item);
    totalSha.update(item.sha256);
  }

  const out = {
    schema: "clawpro.component-spec-index/v1",
    source_root:
      ".codebuddy/skills/clawpro-portable-design-skill/component-specs",
    generated_at: new Date().toISOString(),
    aggregate_sha256: totalSha.digest("hex"),
    count: items.length,
    items,
  };

  writeFileSync(OUT_PATH, JSON.stringify(out, null, 2) + "\n", "utf8");

  // 打印摘要
  const withPortable = items.filter((i) => i.has_portable_fallback).length;
  const totalVariants = items.reduce((s, i) => s + i.variants.length, 0);
  const totalImports = new Set(items.flatMap((i) => i.imports)).size;
  console.log(`[extract-specs] ✓ ${items.length} specs → ${OUT_PATH}`);
  console.log(
    `[extract-specs]   variants=${totalVariants}  unique-imports=${totalImports}  with-portable-fallback=${withPortable}`
  );
}

main();

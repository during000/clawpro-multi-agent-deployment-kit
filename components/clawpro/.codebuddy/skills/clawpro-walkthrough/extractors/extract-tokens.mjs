#!/usr/bin/env node
// extract-tokens.mjs
// 把 clawpro-portable-design-skill/portable/css/tokens.css 抽成 fixtures/tokens.json
// 这是 audit detector（radius / color / shadow）的唯一真相源。
//
// 输出结构：
// {
//   "source": ".../portable/css/tokens.css",
//   "sourceHash": "sha256:...",
//   "extractedAt": "ISO-8601",
//   "radius": { "--radius": "4px", "--alert-radius": "var(--radius)" },
//   "colors": { "--cp-brand-blue": "#1447E6", ... },
//   "shadows": { "--cp-shadow-card-interactive": "...", ... },
//   "fonts":   { "--cp-font-sans": "...", ... },
//   "raw":     { "--xxx": "...", ... }   // 全量，便于其他 detector 兜底查询
// }

import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SKILLS_ROOT = resolve(__dirname, '../..'); // .codebuddy/skills

const SOURCE = resolve(
  SKILLS_ROOT,
  'clawpro-portable-design-skill/portable/css/tokens.css',
);
const OUT_DIR = resolve(__dirname, '../fixtures');
const OUT_FILE = resolve(OUT_DIR, 'tokens.json');

// ---------- 解析 ----------

const css = readFileSync(SOURCE, 'utf8');
const sourceHash = 'sha256:' + createHash('sha256').update(css).digest('hex');

// 抽取 :root { ... } 里的全部 CSS 自定义属性
// 简单可控的解析：行级正则匹配 `--name: value;`
const decls = {};
for (const line of css.split('\n')) {
  // 跳过注释和空行
  const trimmed = line.trim();
  if (!trimmed || trimmed.startsWith('/*') || trimmed.startsWith('*')) continue;
  const m = trimmed.match(/^(--[a-zA-Z0-9_-]+)\s*:\s*([^;]+);?\s*(?:\/\*.*?\*\/)?\s*$/);
  if (!m) continue;
  decls[m[1]] = m[2].trim();
}

// ---------- 分类 ----------

const radius = {};
const colors = {};
const shadows = {};
const fonts = {};

const isColorValue = (v) =>
  /^#([0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})\b/i.test(v) ||
  /^rgba?\(/i.test(v) ||
  /^hsla?\(/i.test(v);

const looksLikeRadius = (k) => /(^|-)radius($|-)/.test(k);
const looksLikeShadow = (k) => /shadow/.test(k);
const looksLikeFont = (k) => /font/.test(k);

for (const [k, v] of Object.entries(decls)) {
  if (looksLikeShadow(k)) {
    shadows[k] = v;
    continue;
  }
  if (looksLikeRadius(k)) {
    radius[k] = v;
    continue;
  }
  if (looksLikeFont(k)) {
    fonts[k] = v;
    continue;
  }
  if (isColorValue(v) || /^var\(--cp-(brand|text|status|border|surface|bg|page|tenant|overlay|control|alert)/.test(v)) {
    colors[k] = v;
    continue;
  }
  // 对 alert-* 等显式带颜色语义的，再补一刀
  if (/^(--alert|--cp-)/.test(k) && /(bg|border|foreground|icon|color)/.test(k)) {
    colors[k] = v;
  }
}

// ---------- 写出 ----------

mkdirSync(OUT_DIR, { recursive: true });

const out = {
  source: SOURCE.replace(SKILLS_ROOT + '/', ''),
  sourceHash,
  extractedAt: new Date().toISOString(),
  counts: {
    total: Object.keys(decls).length,
    radius: Object.keys(radius).length,
    colors: Object.keys(colors).length,
    shadows: Object.keys(shadows).length,
    fonts: Object.keys(fonts).length,
  },
  radius,
  colors,
  shadows,
  fonts,
  raw: decls,
};

writeFileSync(OUT_FILE, JSON.stringify(out, null, 2) + '\n', 'utf8');

console.log('[extract-tokens] OK');
console.log('  source :', out.source);
console.log('  hash   :', sourceHash.slice(0, 19) + '…');
console.log('  counts :', JSON.stringify(out.counts));
console.log('  out    :', OUT_FILE.replace(SKILLS_ROOT + '/', ''));

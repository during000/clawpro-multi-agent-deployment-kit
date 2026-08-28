#!/usr/bin/env node
/**
 * extract-showcase.mjs
 * 抽 client/src/pages/DesignSystemComponents.tsx 里 COMPONENTS 数组 →
 * fixtures/showcase-anchors.json
 *
 * 为什么需要：
 *   DESIGN.md §4.3 要求每条 audit/critique finding 都附带 `showcaseAnchor`，
 *   让研发/设计师"对照看正版"零成本（直接打开 localhost:3002/design-system#<id>）。
 *
 * 抽什么：每个 entry 的 id / group / name / cnName / platform / adoption /
 *        source / doc / hidden / tags（只取扁平字段，descriptions 等长文字段跳过减小体积）
 *
 * 输出 schema：
 *   {
 *     sourceHash: "...",
 *     summary: { total, byGroup, byPlatform, hidden },
 *     anchors: {
 *       "<id>": {
 *         id, group, name, cnName, platform, adoption, source, doc, hidden,
 *         tags: [...],
 *         url: "http://localhost:3002/design-system#<id>"
 *       },
 *       ...
 *     },
 *     // 反查：给一个 className / 组件名想找展示台 anchor 时，扫这个表
 *     nameIndex: { "<Name>": "<id>", ... },
 *     cnNameIndex: { "<中文名>": "<id>", ... }
 *   }
 */

import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '../../../..');
const SHOWCASE_PATH = resolve(ROOT, 'client/src/pages/DesignSystemComponents.tsx');
const OUT = resolve(__dirname, '../fixtures/showcase-anchors.json');
const BASE_URL = 'http://localhost:3002/design-system';

const src = readFileSync(SHOWCASE_PATH, 'utf8');
const sourceHash = createHash('sha256').update(src).digest('hex').slice(0, 16);

/* ---------- 1. 定位 COMPONENTS 数组体 ---------- */
const startMatch = src.match(/const COMPONENTS:\s*ComponentMeta\[\]\s*=\s*\[/);
if (!startMatch) {
  console.error('[extract-showcase] 找不到 COMPONENTS 数组起点');
  process.exit(1);
}
const startIdx = startMatch.index + startMatch[0].length;

// 找配对的 `]` —— 用括号深度扫描，跳过字符串里的方括号
let depth = 1;
let i = startIdx;
let inStr = null; // null | '"' | "'" | '`'
let prev = '';
while (i < src.length && depth > 0) {
  const ch = src[i];
  if (inStr) {
    if (ch === inStr && prev !== '\\') inStr = null;
  } else {
    if (ch === '"' || ch === "'" || ch === '`') inStr = ch;
    else if (ch === '[') depth++;
    else if (ch === ']') depth--;
  }
  prev = ch;
  i++;
}
if (depth !== 0) {
  console.error('[extract-showcase] COMPONENTS 数组括号未配对');
  process.exit(1);
}
const arrayBody = src.slice(startIdx, i - 1);

/* ---------- 2. 切分 entry：顶层 `{ ... },` ---------- */
const entries = [];
{
  let d = 0;
  let s = -1;
  let inS = null;
  let p = '';
  for (let k = 0; k < arrayBody.length; k++) {
    const c = arrayBody[k];
    if (inS) {
      if (c === inS && p !== '\\') inS = null;
    } else {
      if (c === '"' || c === "'" || c === '`') inS = c;
      else if (c === '{') {
        if (d === 0) s = k;
        d++;
      } else if (c === '}') {
        d--;
        if (d === 0 && s >= 0) {
          entries.push(arrayBody.slice(s, k + 1));
          s = -1;
        }
      }
    }
    p = c;
  }
}

/* ---------- 3. 解析每个 entry 的扁平字段 ---------- */
function pickStr(entry, key) {
  // 形如 `key: "value"` 或 `key: 'value'` 或 `key: \`value\``，跳过对象/数组值
  const re = new RegExp(`(?:^|[\\s,{])${key}\\s*:\\s*(["'\`])((?:\\\\.|(?!\\1).)*)\\1`);
  const m = entry.match(re);
  return m ? m[2] : null;
}
function pickBool(entry, key) {
  const m = entry.match(new RegExp(`(?:^|[\\s,{])${key}\\s*:\\s*(true|false)`));
  return m ? m[1] === 'true' : null;
}
function pickArr(entry, key) {
  // 抓 `key: [ ... ]`（浅层，只匹配字符串数组）
  const re = new RegExp(`(?:^|[\\s,{])${key}\\s*:\\s*\\[([\\s\\S]*?)\\]`);
  const m = entry.match(re);
  if (!m) return [];
  const inner = m[1];
  const items = [];
  const itemRe = /(["'`])((?:\\.|(?!\1).)*)\1/g;
  let mm;
  while ((mm = itemRe.exec(inner)) !== null) items.push(mm[2]);
  return items;
}

const anchors = {};
const nameIndex = {};
const cnNameIndex = {};
const byGroup = {};
const byPlatform = {};
let hiddenCount = 0;

for (const e of entries) {
  const id = pickStr(e, 'id');
  if (!id) continue; // 跳过 spread / 计算属性等异常 entry
  const name = pickStr(e, 'name');
  const cnName = pickStr(e, 'cnName');
  const group = pickStr(e, 'group');
  const platform = pickStr(e, 'platform');
  const adoption = pickStr(e, 'adoption');
  const source = pickStr(e, 'source');
  const doc = pickStr(e, 'doc');
  const hidden = pickBool(e, 'hidden') === true;
  const tags = pickArr(e, 'tags');

  if (hidden) hiddenCount++;
  if (group) byGroup[group] = (byGroup[group] || 0) + 1;
  if (platform) byPlatform[platform] = (byPlatform[platform] || 0) + 1;

  anchors[id] = {
    id,
    group,
    name,
    cnName,
    platform,
    adoption,
    source,
    doc,
    hidden,
    tags,
    url: `${BASE_URL}#${id}`,
  };
  if (name) nameIndex[name] = id;
  if (cnName) cnNameIndex[cnName] = id;
}

/* ---------- 4. 写出 ---------- */
const out = {
  sourceHash,
  sourcePath: 'client/src/pages/DesignSystemComponents.tsx',
  summary: {
    total: Object.keys(anchors).length,
    byGroup,
    byPlatform,
    hidden: hiddenCount,
  },
  anchors,
  nameIndex,
  cnNameIndex,
};

writeFileSync(OUT, JSON.stringify(out, null, 2), 'utf8');
console.log(
  `[extract-showcase] ${out.summary.total} anchors · groups ${JSON.stringify(
    out.summary.byGroup,
  )} · hidden ${out.summary.hidden} · sourceHash ${sourceHash}`,
);

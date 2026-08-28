#!/usr/bin/env node
// extract-icon-slots.mjs
// ----------------------------------------------------------------------------
// 把"管控端 9 类不可回退图标槽位"从设计 skill 的 SKILL.md 抽成机器可读 fixture。
//
// 为什么需要这个 extractor：
//   - 设计 skill (clawpro-portable-design-skill/SKILL.md §2.8 + §8 + §19) 用自然语言
//     列了 9 个槽位 id，是"法律原文"。
//   - 走查 skill 的 admin/scripts/detectors/icon-slot.mjs 和老脚本
//     clawpro-portable-design-skill/scripts/check-design-usage.mjs 都各自硬编码了一份
//     9 槽位字符串，**两处会漂移**。
//   - 这个 extractor 把 9 槽位变成 fixtures/icon-slots.json，作为**唯一真相源**，
//     detector / 老脚本以后都从这里读。
//
// 实现策略：
//   v0.1 不做 AST，直接维护人工 fixture（已写在 fixtures/icon-slots.json）。
//   这个脚本只做两件事：
//     1. 校验 fixture 与设计 skill SKILL.md 的 9 槽位 id 是否一致（漂移检查）
//     2. 校验 fixture schema 合法
//
// 跑法：
//   node extractors/extract-icon-slots.mjs           # 校验
//   node extractors/extract-icon-slots.mjs --json    # 校验并打印 fixture
//
// 退出码：
//   0  fixture 合法且与设计 skill 一致
//   1  漂移检测失败（fixture 与 SKILL.md §8 不一致）
//   2  fixture schema 不合法 / IO 错误
// ----------------------------------------------------------------------------

import { readFileSync, existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const CORE_ROOT = resolve(__dirname, '..');
const SKILLS_ROOT = resolve(CORE_ROOT, '..');
const FIXTURE_PATH = resolve(CORE_ROOT, 'fixtures/icon-slots.json');
// 真相源 = 设计 skill SKILL.md（前 5 槽位有显式 id）+ check-design-usage.mjs（9 槽位完整枚举）
// 两份都参与漂移检测：任何一份命中即视为不漂移
const TRUTH_SOURCES = [
  resolve(SKILLS_ROOT, 'clawpro-portable-design-skill/SKILL.md'),
  resolve(SKILLS_ROOT, 'clawpro-portable-design-skill/scripts/check-design-usage.mjs'),
];

const args = process.argv.slice(2);
const jsonMode = args.includes('--json');

function fail(code, msg) {
  console.error(`[extract-icon-slots] ${msg}`);
  process.exit(code);
}

// ---------- 1. 读 fixture ----------
if (!existsSync(FIXTURE_PATH)) {
  fail(2, `fixture 不存在: ${FIXTURE_PATH}`);
}
let fixture;
try {
  fixture = JSON.parse(readFileSync(FIXTURE_PATH, 'utf8'));
} catch (e) {
  fail(2, `fixture JSON 解析失败: ${e.message}`);
}

// ---------- 2. schema 校验 ----------
if (!Array.isArray(fixture.slots) || fixture.slots.length === 0) {
  fail(2, 'fixture.slots 应为非空数组');
}
const REQUIRED_FIELDS = ['id', 'label'];
for (const slot of fixture.slots) {
  for (const f of REQUIRED_FIELDS) {
    if (!slot[f]) fail(2, `slot 缺少必填字段 ${f}: ${JSON.stringify(slot)}`);
  }
}

// ---------- 3. 与设计 skill SKILL.md §8 漂移检测 ----------
//
// 设计 skill SKILL.md §8 / §19 用自然语言列了 9 个槽位 id（带 `code` 标记）。
// 这里抽出 SKILL.md 里所有 ` `code` ` 形式且匹配 9 槽位枚举的 token，对照 fixture。
// 注意：SKILL.md §8 是文字 + checklist，"其他 4 个…" 不会被抽到；我们只做白名单存在性校验。

let driftWarnings = [];
const sourceTexts = TRUTH_SOURCES.filter((p) => existsSync(p)).map((p) => ({
  path: p,
  text: readFileSync(p, 'utf8'),
}));
if (sourceTexts.length === 0) {
  driftWarnings.push('未找到任何真相源文件，跳过漂移检测');
} else {
  for (const slot of fixture.slots) {
    const aliases = slot.id === 'run-status-indicator' ? [slot.id, 'run-status'] : [slot.id];
    const regexes = aliases.map(
      (a) => new RegExp(`\\b${a.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&')}\\b`),
    );
    const hit = sourceTexts.some(({ text }) => regexes.some((r) => r.test(text)));
    if (!hit) {
      driftWarnings.push(
        `fixture 槽位 "${slot.id}" 在以下任一真相源中均未找到：` +
          sourceTexts.map((s) => s.path.replace(SKILLS_ROOT + '/', '')).join(' | '),
      );
    }
  }
}

// ---------- 4. 输出 ----------
const summary = {
  fixturePath: FIXTURE_PATH,
  slotCount: fixture.slots.length,
  ids: fixture.slots.map((s) => s.id),
  driftWarnings,
};

if (jsonMode) {
  console.log(JSON.stringify({ summary, fixture }, null, 2));
} else {
  console.log(`[extract-icon-slots] fixture OK · ${summary.slotCount} 槽位`);
  console.log(`  ids: ${summary.ids.join(', ')}`);
  if (driftWarnings.length > 0) {
    console.log(`  ⚠️ 漂移警告（${driftWarnings.length}）：`);
    for (const w of driftWarnings) console.log(`    - ${w}`);
  } else {
    console.log(`  ✓ 与真相源一致（${sourceTexts.length} 个源参与校验）`);
  }
}

process.exit(driftWarnings.length > 0 ? 1 : 0);

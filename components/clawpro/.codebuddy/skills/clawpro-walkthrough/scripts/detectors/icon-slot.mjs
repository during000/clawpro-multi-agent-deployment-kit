// icon-slot detector
// ----------------------------------------------------------------------------
// 管控端铁律：9 类不可回退槽位禁止使用 lucide-react 图标，必须用 ClawPro 自研 SVG。
//
// v0.6 实现"宽口径预警"：只要文件 import 了 lucide-react，就把每个 import 行抛 P0，
// 让用户自己判断这些图标是否落在不可回退槽位。后续 1.0 接入按 import 位置 +
// JSX 父组件的精确槽位匹配。
//
// 9 槽位的真相源：clawpro-walkthrough/fixtures/icon-slots.json
//   - fixture 由 core/extractors/extract-icon-slots.mjs 维护
//   - fixture 与设计 skill SKILL.md §2.8 + check-design-usage.mjs 双向校验
//
// severity: P0
// ----------------------------------------------------------------------------

import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { makeFinding, scanRegex } from './_shared.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ICON_SLOTS_FIXTURE = resolve(
  __dirname,
  '../../fixtures/icon-slots.json',
);

// 单例缓存：每次 walkthrough 进程只读一次 fixture
let _slotsCache = null;
function loadSlots() {
  if (_slotsCache) return _slotsCache;
  try {
    const json = JSON.parse(readFileSync(ICON_SLOTS_FIXTURE, 'utf8'));
    _slotsCache = json.slots || [];
  } catch (e) {
    // fixture 加载失败时降级到 0 槽位，仅保留 lucide-react import 检测
    _slotsCache = [];
  }
  return _slotsCache;
}

const RULE_ID = 'icon-slot';
const SEVERITY = 'P0';

export function run({ file, text }) {
  const findings = [];

  // 不扫 portable 自身和走查 skill 自身
  if (/\.codebuddy\/skills\//.test(file)) return findings;

  const slots = loadSlots();
  const slotIds = slots.map((s) => s.id).join(' / ') || '管控端 9 类不可回退槽位';

  scanRegex(
    text,
    /^\s*import\s+[^;]*\bfrom\s+['"]lucide-react['"]\s*;?/gm,
    (m, line, col, lineText) => {
      findings.push(
        makeFinding({
          ruleId: RULE_ID,
          severity: SEVERITY,
          file,
          line,
          col,
          snippet: lineText.trim(),
          message: `使用了 lucide-react；若该图标用于不可回退槽位（${slotIds}），则违反铁律`,
          evidence: 'clawpro-walkthrough/fixtures/icon-slots.json',
          suggestion:
            '改用 ClawPro 自研 SVG（见 portable/assets/）；具体候选见 icon-slots.json 的 specPath / showcaseAnchor',
        }),
      );
    },
  );

  return findings;
}

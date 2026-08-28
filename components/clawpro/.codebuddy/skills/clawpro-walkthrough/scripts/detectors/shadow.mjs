// shadow detector
// 强制走 tokens.css 的 --cp-shadow-* 体系（共 6 个）：
//   --cp-shadow-card-interactive / -card-hover / -overlay
//   --cp-shadow-tenant-card / -tenant-card-hover / -config
//
// 命中规则：
//   A. CSS 硬编码 box-shadow: ...   (非 var(--cp-shadow-*)、非 none)        → P1
//   B. Tailwind 任意值 shadow-[...] (非 var(--cp-shadow-*))                → P1
//   C. Tailwind 框架级 shadow-sm/md/lg/xl/2xl/inner/shadow                 → P2
//
// 放行：
//   - var(--cp-shadow-*) / shadow-[var(--cp-shadow-*)]
//   - box-shadow: none / shadow-none
//
// evidence: tokens.json#shadows
// severity: P1 / P2

import { makeFinding, scanRegex } from './_shared.mjs';
import { loadTokens } from './_shared.mjs';

const RULE_ID = 'shadow';

const TW_FRAMEWORK = new Set([
  'shadow-sm',
  'shadow-md',
  'shadow-lg',
  'shadow-xl',
  'shadow-2xl',
  'shadow-inner',
]);

function truncate(s, n) {
  return s.length > n ? s.slice(0, n - 1) + '…' : s;
}

// 已登记 token 集合（用于校验 suggestion 真实存在）
function knownShadowNames(tokens) {
  return new Set(Object.keys(tokens.shadows || {}));
}

// 给一个硬编码值挑最贴近的 --cp-shadow-*；规则简单到能解释：
//   - 含 inset → 人工裁决
//   - 模糊半径 ≥ 10px 或多层阴影 → overlay
//   - 单层 + 透明度 0.05~0.08 → card-interactive
//   - 其它兜底 → tenant-card
function pickToken(value, known) {
  if (/inset/i.test(value)) return null;
  let pick = '--cp-shadow-card-interactive';
  if (/,/.test(value) || /(1[0-9]|2[0-9])px/.test(value)) {
    pick = '--cp-shadow-overlay';
  } else if (/rgba\s*\(\s*0\s*,\s*0\s*,\s*0\s*,\s*0?\.0?[5-9]/.test(value)) {
    pick = '--cp-shadow-card-interactive';
  } else if (/0?\.1[0-9]/.test(value)) {
    pick = '--cp-shadow-tenant-card';
  }
  if (!known.has(pick)) pick = [...known][0] || pick;
  return pick;
}

export function run({ file, text }) {
  const tokens = loadTokens();
  const known = knownShadowNames(tokens);
  const findings = [];

  const isCss = /\.(css|scss)$/.test(file);

  // ---------- A. CSS 文件里的硬编码 box-shadow ----------
  // 只在 .css/.scss 里扫，避免 tsx 字符串内的 `shadow-[...]` 被这条规则误伤
  if (isCss) {
    scanRegex(text, /box-shadow\s*:\s*([^;]+);/g, (m, line, col, lineText) => {
      const value = m[1].trim();
      if (/^none$/i.test(value)) return;
      // 完全基于 var(--cp-shadow-*) 的合成（含 inset / 颜色叠层）允许
      if (/var\(\s*--cp-shadow-/.test(value)) return;
      // 没有 hex / rgba / 裸 px 数字 → 不是真正的硬编码（可能是别的 var）
      if (!/(#[0-9a-f]{3,8}|rgba?\s*\(|\d+px)/i.test(value)) return;

      const tk = pickToken(value, known);
      const suggestion = tk
        ? `改为 \`box-shadow: var(${tk});\``
        : '包含 inset 等复合阴影，请人工裁决：保留或在 tokens.css 新增 --cp-shadow-*';
      findings.push(
        makeFinding({
          ruleId: RULE_ID,
          severity: 'P1',
          file,
          line,
          col,
          snippet: lineText.trim(),
          message: `box-shadow 硬编码（${truncate(value, 60)}），应走 --cp-shadow-* token`,
          evidence: 'tokens.json#shadows',
          suggestion,
        }),
      );
    });
  }

  // ---------- B. Tailwind 任意值 shadow-[...] ----------
  scanRegex(text, /\bshadow-\[([^\]]+)\]/g, (m, line, col, lineText) => {
    const value = m[1].trim();
    if (/^var\(\s*--cp-shadow-/.test(value)) return;

    // 子情形 B1：使用了 var(--xxx) 但不是 --cp-shadow-* 体系
    //   → 自创变量，应先登记到 tokens.css 而不是顺手套个 cp-shadow
    const customVar = value.match(/^var\(\s*(--[A-Za-z0-9-]+)/);
    if (customVar) {
      findings.push(
        makeFinding({
          ruleId: RULE_ID,
          severity: 'P1',
          file,
          line,
          col,
          snippet: lineText.trim(),
          message: `Tailwind 任意值阴影使用了非 --cp-shadow-* 变量 \`${customVar[1]}\``,
          evidence: 'tokens.json#shadows',
          suggestion: `若该阴影属于通用样式，请先把 ${customVar[1]} 登记进 tokens.css 并改名为 --cp-shadow-*；若仅本组件私用，可保留但需在 component-spec 中显式声明`,
        }),
      );
      return;
    }

    // 子情形 B2：完全硬编码（rgba / hex / 裸 px）
    const tk = pickToken(value, known);
    const suggestion = tk
      ? `改为 \`shadow-[var(${tk})]\``
      : '请人工裁决：保留或在 tokens.css 新增 --cp-shadow-*';
    findings.push(
      makeFinding({
        ruleId: RULE_ID,
        severity: 'P1',
        file,
        line,
        col,
        snippet: lineText.trim(),
        message: `Tailwind 任意值阴影 shadow-[${truncate(value, 40)}]，应走 --cp-shadow-* token`,
        evidence: 'tokens.json#shadows',
        suggestion,
      }),
    );
  });

  // ---------- C. Tailwind 框架级 shadow-* ----------
  // 只在 .tsx/.jsx/.ts/.js 里识别（避免 CSS 选择器误伤）
  // 边界：前一个字符为 [\s"'`{:>]，后一个为 [\s"'`}<:/]
  if (!isCss) {
    scanRegex(
      text,
      /(^|[\s"'`{:>])(shadow-(?:sm|md|lg|xl|2xl|inner))(?=[\s"'`}<:/])/g,
      (m, line, col, lineText) => {
        const cls = m[2];
        if (!TW_FRAMEWORK.has(cls)) return;
        findings.push(
          makeFinding({
            ruleId: RULE_ID,
            severity: 'P2',
            file,
            line,
            // m[1] 是前缀字符，真实命中起点要往后挪
            col: col + (m[1] ? m[1].length : 0),
            snippet: lineText.trim(),
            message: `使用 Tailwind 框架级阴影 \`${cls}\`，与 ClawPro 阴影体系未对齐`,
            evidence: 'tokens.json#shadows',
            suggestion:
              '卡片：`shadow-[var(--cp-shadow-card-interactive)]`；弹层：`shadow-[var(--cp-shadow-overlay)]`；Tenant 卡片：`shadow-[var(--cp-shadow-tenant-card)]`',
          }),
        );
      },
    );
  }

  return findings;
}

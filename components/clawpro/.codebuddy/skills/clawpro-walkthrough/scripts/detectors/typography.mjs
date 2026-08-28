// typography detector（ruleId: text-color）
// 设计 skill §2.5「文字层级」：文字色必须走 Typography 语义组件或 --cp-text-* token，
// 不允许散落 Tailwind 内置中性灰阶（text-gray-500 / text-slate-400 …）。
//
// 与 color detector 的分工：
//   - color 抓的是「硬编码 hex / Tailwind 任意色 text-[#xxx]」；
//   - typography 抓的是「Tailwind 内置中性色阶 text-{gray,slate,zinc,neutral,stone}-NNN」——
//     这类不是 hex，color detector 抓不到，但同样违反 §2.5。
//   两者互不重叠。
//
// evidence: clawpro-portable-design-skill/SKILL.md#§2.5 + tokens.json#colors(--cp-text-*)
// severity: P2（体感小瑕疵，下一轮统一修）
// stream:   audit（确定违规）

import { makeFinding, scanRegex } from './_shared.mjs';

const RULE_ID = 'text-color';
const SEVERITY = 'P2';

// Tailwind 内置中性色族；这些用于「文字色」时应走 --cp-text-*
const NEUTRAL_FAMILIES = '(gray|slate|zinc|neutral|stone)';

// 深→浅 粗映射到 --cp-text-* 家族（仅作建议，最终以设计 skill §2.5 为准）
function suggestToken(family, shade) {
  const n = parseInt(shade, 10);
  if (n >= 900) return '--cp-text-emphasis / --cp-text-title';
  if (n >= 700) return '--cp-text-body / --cp-text-secondary';
  if (n >= 500) return '--cp-text-muted';
  return '--cp-text-weak';
}

export function run({ file, text }) {
  const findings = [];

  // CSS 文件不在本规则范围（CSS 里用 var(--cp-text-*) 是另一回事，且不会写 text-gray-500 类名）
  if (/\.css$/.test(file)) return findings;

  // 只抓 text-<family>-<shade>（文字色）；bg-/border- 等非文字色不在 §2.5 范围
  const re = new RegExp(`\\btext-${NEUTRAL_FAMILIES}-(\\d{2,3})\\b`, 'g');
  scanRegex(text, re, (m, line, col, lineText) => {
    const trimmed = lineText.trim();
    // 跳过注释行
    if (trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) return;
    const family = m[1];
    const shade = m[2];
    const tokenHint = suggestToken(family, shade);
    findings.push(
      makeFinding({
        ruleId: RULE_ID,
        severity: SEVERITY,
        file,
        line,
        col,
        snippet: trimmed,
        message: `文字色用了 Tailwind 内置 text-${family}-${shade}，应走 Typography 语义或 --cp-text-* token`,
        evidence: 'SKILL.md#§2.5 + tokens.json#colors(--cp-text-*)',
        suggestion: `改用 Typography 语义组件（SectionTitle / MetaText 等）或 \`text-[var(${tokenHint})]\``,
      }),
    );
  });

  return findings;
}

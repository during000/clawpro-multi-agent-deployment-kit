// radius detector
// 管控端铁律：圆角统一 4px（var(--radius)）
// 违规模式：
//   1) Tailwind 任意非 4px 圆角类：rounded-md / rounded-lg / rounded-xl / rounded-2xl / rounded-3xl / rounded-full
//      （rounded 自身在 Tailwind 默认是 0.25rem=4px，可豁免；rounded-sm 是 2px，仍属漂离）
//   2) 任意像素圆角：rounded-[8px] / rounded-[12px] / rounded-[6px] 等
//   3) CSS 内联：style={{ borderRadius: '8px' }} / borderRadius: 8 / borderRadius:'12px'
//   4) CSS 文件里：border-radius: 8px (不等于 4px / var(--radius))
//
// evidence: tokens.json#radius.--radius=4px
// severity: P0

import { makeFinding, scanRegex } from './_shared.mjs';

const RULE_ID = 'radius';
const SEVERITY = 'P0';

const TW_BAD_KEYWORDS = new Set([
  'rounded-sm',
  'rounded-md',
  'rounded-lg',
  'rounded-xl',
  'rounded-2xl',
  'rounded-3xl',
  'rounded-full',
]);

export function run({ file, text }) {
  const findings = [];

  // ---------- 1) Tailwind 关键字类 ----------
  // 抓在 className / class 字符串内出现的 rounded-* 关键字
  // 简化：只要源码里出现 \brounded-(sm|md|lg|xl|2xl|3xl|full)\b 就算
  scanRegex(text, /\brounded-(sm|md|lg|xl|2xl|3xl|full)\b/g, (m, line, col, lineText) => {
    const cls = `rounded-${m[1]}`;
    findings.push(
      makeFinding({
        ruleId: RULE_ID,
        severity: SEVERITY,
        file,
        line,
        col,
        snippet: lineText.trim(),
        message: `管控端禁用 ${cls}（圆角必须为 4px）`,
        evidence: 'tokens.json#radius.--radius=4px',
        suggestion: '改为 `rounded` 或 `rounded-[var(--radius)]`',
      }),
    );
  });

  // ---------- 2) Tailwind 任意像素圆角 ----------
  scanRegex(text, /\brounded-\[([^\]]+)\]/g, (m, line, col, lineText) => {
    const raw = m[1].trim();
    // 允许 var(--radius) / 4px / 0.25rem
    if (/^var\(--radius/.test(raw)) return;
    if (raw === '4px' || raw === '0.25rem') return;
    findings.push(
      makeFinding({
        ruleId: RULE_ID,
        severity: SEVERITY,
        file,
        line,
        col,
        snippet: lineText.trim(),
        message: `管控端禁止任意圆角 rounded-[${raw}]（必须为 4px）`,
        evidence: 'tokens.json#radius.--radius=4px',
        suggestion: '改为 `rounded` 或 `rounded-[var(--radius)]`',
      }),
    );
  });

  // ---------- 3) 内联样式 borderRadius ----------
  // borderRadius: '8px' / borderRadius: 8 / borderRadius: `12px`
  scanRegex(
    text,
    /borderRadius\s*:\s*(['"`]?)([^,'"`}\s]+)\1/g,
    (m, line, col, lineText) => {
      const val = m[2].trim();
      // 允许 var(--radius) / 4 / 4px / 0.25rem / 0
      if (/^var\(--radius/.test(val)) return;
      if (val === '0' || val === '4' || val === '4px' || val === '0.25rem') return;
      findings.push(
        makeFinding({
          ruleId: RULE_ID,
          severity: SEVERITY,
          file,
          line,
          col,
          snippet: lineText.trim(),
          message: `内联 borderRadius=${val} 违反 4px 圆角铁律`,
          evidence: 'tokens.json#radius.--radius=4px',
          suggestion: '改为 `borderRadius: \'var(--radius)\'`',
        }),
      );
    },
  );

  // ---------- 4) CSS 文件里的 border-radius ----------
  if (/\.css$/.test(file)) {
    scanRegex(text, /border-radius\s*:\s*([^;]+);/g, (m, line, col, lineText) => {
      const val = m[1].trim();
      if (/var\(--radius/.test(val)) return;
      if (val === '4px' || val === '0' || val === '0.25rem') return;
      findings.push(
        makeFinding({
          ruleId: RULE_ID,
          severity: SEVERITY,
          file,
          line,
          col,
          snippet: lineText.trim(),
          message: `border-radius: ${val} 违反 4px 圆角铁律`,
          evidence: 'tokens.json#radius.--radius=4px',
          suggestion: '改为 `border-radius: var(--radius);`',
        }),
      );
    });
  }

  return findings;
}

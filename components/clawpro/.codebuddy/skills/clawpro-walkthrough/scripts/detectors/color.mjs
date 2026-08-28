// color detector
// 检查应走 token 的硬编码颜色写法。
// 违规模式：
//   1) JSX/TS 文件里出现裸 hex（#1447E6 / #FFFFFF / #fff），且不是注释内
//   2) CSS 文件里非 tokens.css 本身出现裸 hex
//   3) Tailwind 任意色 text-[#xxx] / bg-[#xxx] / border-[#xxx]
//
// 数据源：core/fixtures/tokens.json#colors —— 每条命中尝试在 token 表里找等值替代写入 suggestion。
//
// severity: P1（不阻断、但要修）

import { makeFinding, scanRegex, loadTokens } from './_shared.mjs';

const RULE_ID = 'color';
const SEVERITY = 'P1';

function buildColorIndex(tokens) {
  // 把 colors 表里的 hex 反向索引：normHex → tokenName
  const idx = new Map();
  for (const [name, raw] of Object.entries(tokens.colors || {})) {
    const m = raw.match(/^#([0-9a-fA-F]{3,8})\b/);
    if (!m) continue;
    let hex = m[1].toUpperCase();
    if (hex.length === 3) {
      hex = hex
        .split('')
        .map((c) => c + c)
        .join('');
    }
    const norm = '#' + hex;
    if (!idx.has(norm)) idx.set(norm, name);
  }
  return idx;
}

function normalizeHex(raw) {
  let h = raw.replace('#', '').toUpperCase();
  if (h.length === 3) h = h.split('').map((c) => c + c).join('');
  return '#' + h;
}

export function run({ file, text }) {
  const findings = [];
  const tokens = loadTokens();
  const colorIdx = buildColorIndex(tokens);

  // tokens.css 本体跳过
  if (/portable\/css\/tokens\.css$/.test(file)) return findings;

  const isCss = /\.css$/.test(file);

  // ---------- 1) 裸 hex ----------
  scanRegex(text, /#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})\b/g, (m, line, col, lineText) => {
    const trimmed = lineText.trim();
    // 跳过注释行
    if (trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) return;
    // 跳过 import 路径里的哈希（极少见，但保险）
    if (/from\s+['"]/.test(trimmed) && trimmed.includes('#')) return;

    const raw = m[0];
    const norm = normalizeHex(raw);
    const tokenName = colorIdx.get(norm);

    findings.push(
      makeFinding({
        ruleId: RULE_ID,
        severity: SEVERITY,
        file,
        line,
        col,
        snippet: trimmed,
        message: tokenName
          ? `硬编码 ${raw}，应走 token \`${tokenName}\``
          : `硬编码颜色 ${raw}，应走 design token`,
        evidence: tokenName
          ? `tokens.json#colors.${tokenName}=${norm}`
          : 'tokens.json#colors',
        suggestion: tokenName
          ? isCss
            ? `改为 \`var(${tokenName})\``
            : `改为 \`text-[var(${tokenName})]\` / \`bg-[var(${tokenName})]\` 等 token 写法`
          : '在 tokens.css 登记该颜色后再使用',
      }),
    );
  });

  // ---------- 2) Tailwind 任意色 ----------
  scanRegex(
    text,
    /\b(text|bg|border|ring|fill|stroke|from|to|via)-\[#([0-9a-fA-F]{3,8})\]/g,
    (m, line, col, lineText) => {
      const utility = m[1];
      const raw = '#' + m[2];
      const norm = normalizeHex(raw);
      const tokenName = colorIdx.get(norm);
      findings.push(
        makeFinding({
          ruleId: RULE_ID,
          severity: SEVERITY,
          file,
          line,
          col,
          snippet: lineText.trim(),
          message: tokenName
            ? `Tailwind 任意色 ${utility}-[${raw}]，应走 token \`${tokenName}\``
            : `Tailwind 任意色 ${utility}-[${raw}]，应走 design token`,
          evidence: tokenName
            ? `tokens.json#colors.${tokenName}=${norm}`
            : 'tokens.json#colors',
          suggestion: tokenName
            ? `改为 \`${utility}-[var(${tokenName})]\``
            : '在 tokens.css 登记该颜色后再使用',
        }),
      );
    },
  );

  return findings;
}

// spacing detector（ruleId: spacing-grouping）
// 设计 skill §2.7「间距与成组」：
//   ❌ 给每个控件单独 margin：<Input className="mr-3"/><Select className="mr-3"/><Button/>
//   ✅ 用 flex + gap 成组：  <div className="flex items-center gap-3"><Input/><Select/><Button/></div>
//
// 为什么进 design-todo（todo 流）而不是 audit-report：
//   单独的 mr-*/ml-* 本身不一定是错（有时确实需要单边间距）；
//   只有「相邻多个控件各自带水平 margin」这种成组反模式才需要改。
//   这是判断题，按 SKILL §0.B 表第 4 行约定交用户裁决，AI 不私自定性。
//
// 启发式（控噪）：把出现 \bm[rl]-\d 的位置按行聚类，
//   同一簇（相邻 ≤ SPAN 行）内出现 >= 2 次水平 margin → 报 1 条 todo（不逐行轰炸）。
//
// evidence: clawpro-portable-design-skill/SKILL.md#§2.7
// severity: P2
// stream:   todo（待用户裁决）

import { makeFinding, scanRegex } from './_shared.mjs';

const RULE_ID = 'spacing-grouping';
const SEVERITY = 'P2';
const CONFLICT_TYPE = 'Spec vs 现状（§2.7 间距成组）';
const SPAN = 4; // 相邻聚类窗口（行）

export function run({ file, text }) {
  if (/\.css$/.test(file)) return [];

  // 收集所有水平 margin 命中（mr-/ml-，含任意值 mr-[..]）
  const hits = [];
  scanRegex(text, /\bm[rl]-(\[[^\]]+\]|\d+)\b/g, (m, line, col, lineText) => {
    const trimmed = lineText.trim();
    if (trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) return;
    hits.push({ line, col, lineText: trimmed, cls: m[0] });
  });

  if (hits.length < 2) return [];

  // 按行聚类：相邻命中行距 <= SPAN 归为一簇
  const clusters = [];
  let cur = [hits[0]];
  for (let i = 1; i < hits.length; i++) {
    if (hits[i].line - cur[cur.length - 1].line <= SPAN) {
      cur.push(hits[i]);
    } else {
      clusters.push(cur);
      cur = [hits[i]];
    }
  }
  clusters.push(cur);

  const findings = [];
  for (const cl of clusters) {
    if (cl.length < 2) continue; // 单个 margin 不算成组反模式
    const head = cl[0];
    const classes = [...new Set(cl.map((h) => h.cls))].join(', ');
    findings.push(
      makeFinding({
        ruleId: RULE_ID,
        severity: SEVERITY,
        stream: 'todo',
        conflictType: CONFLICT_TYPE,
        file,
        line: head.line,
        col: head.col,
        snippet: head.lineText,
        message: `第 ${head.line}-${cl[cl.length - 1].line} 行有 ${cl.length} 处水平 margin（${classes}），疑似逐个控件加 margin，应改 flex+gap 成组`,
        evidence: 'SKILL.md#§2.7',
        suggestion: '若为相邻控件成组，改用 `<div className="flex items-center gap-N">…</div>`；若确为单边间距，请在 design-todo 标"误报"',
      }),
    );
  }
  return findings;
}

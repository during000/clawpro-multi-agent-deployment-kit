// 公共工具：读取 fixtures、统一 finding 协议、按行/列定位
import { readFileSync, existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
export const SKILLS_ROOT = resolve(__dirname, '../../..'); // .codebuddy/skills
// fixtures 相对本 skill 目录定位（重命名 skill 目录也不会断）：
//   scripts/detectors → ../../fixtures = <skill>/fixtures
export const CORE_FIXTURES = resolve(__dirname, '../../fixtures');

let _tokens = null;
export function loadTokens() {
  if (_tokens) return _tokens;
  const p = resolve(CORE_FIXTURES, 'tokens.json');
  if (!existsSync(p)) {
    throw new Error(
      `[walkthrough] fixtures/tokens.json 缺失：请确认 skill 已完整安装（fixtures/ 目录随交付包提供）`,
    );
  }
  _tokens = JSON.parse(readFileSync(p, 'utf8'));
  return _tokens;
}

/**
 * 统一 finding 协议
 *
 * stream（落表分流，落实 SKILL §0.C / DESIGN §1.5.1）：
 *   - 'audit'（默认）：确定违规，进 audit-report.csv，AI 直接修。
 *   - 'todo'         ：AI 拿不准 / 需用户裁决，进 design-todo.csv，禁止私自定性。
 * conflictType：仅 stream='todo' 时有意义，对应设计 skill §12.3 的 5 类冲突
 *   （Spec vs 现状 / 需求超范围 / 图标无候选 / Token 模糊 / 宿主仓兼容）。
 *
 * @returns {{ruleId,severity,file,line,col,snippet,message,evidence,suggestion,stream,conflictType}}
 */
export function makeFinding({
  ruleId,
  severity,
  file,
  line,
  col,
  snippet,
  message,
  evidence,
  suggestion,
  stream,
  conflictType,
}) {
  return {
    ruleId,
    severity,
    file,
    line,
    col,
    snippet: (snippet ?? '').slice(0, 80),
    message,
    evidence,
    suggestion: suggestion ?? '',
    stream: stream === 'todo' ? 'todo' : 'audit',
    conflictType: conflictType ?? '',
  };
}

/**
 * 按 (start, end) 字节偏移定位 (line, col)
 */
export function locate(text, offset) {
  let line = 1;
  let col = 1;
  for (let i = 0; i < offset && i < text.length; i++) {
    if (text[i] === '\n') {
      line++;
      col = 1;
    } else {
      col++;
    }
  }
  return { line, col };
}

/**
 * 在源码中按正则全量匹配，回调每个命中
 * cb(match, line, col, lineText)
 */
export function scanRegex(text, regex, cb) {
  // 强制 g flag
  const re = regex.global ? regex : new RegExp(regex.source, regex.flags + 'g');
  const lines = text.split('\n');
  let lineOffsets = [0];
  for (let i = 0; i < lines.length; i++) {
    lineOffsets.push(lineOffsets[i] + lines[i].length + 1);
  }
  let m;
  while ((m = re.exec(text)) !== null) {
    const start = m.index;
    // 二分找 line
    let lo = 0, hi = lineOffsets.length - 1;
    while (lo < hi) {
      const mid = (lo + hi + 1) >> 1;
      if (lineOffsets[mid] <= start) lo = mid;
      else hi = mid - 1;
    }
    const line = lo + 1;
    const col = start - lineOffsets[lo] + 1;
    cb(m, line, col, lines[lo]);
    // 防止零宽匹配死循环
    if (m.index === re.lastIndex) re.lastIndex++;
  }
}

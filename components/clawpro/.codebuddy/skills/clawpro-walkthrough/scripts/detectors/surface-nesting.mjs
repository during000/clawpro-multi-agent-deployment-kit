// surface-nesting detector（ruleId: surface-nesting）
// 设计 skill §7.4「Surface 嵌套（不要套娃）」：
//   ❌ <SurfaceCard><SurfaceCard>...</SurfaceCard></SurfaceCard>
//   ✅ <SurfaceCard><SurfaceInner>...</SurfaceInner></SurfaceCard>
// 同名 Surface 容器自我嵌套即套娃；内层必须降到 SurfaceInner。
// 同理 TenantCard 也不应自我嵌套。
//
// 实现：对每个受控组件做「栈式」标签扫描（开标签入栈 / 闭标签出栈 / 自闭合不计），
// 当同名组件深度 >= 2 时，对该内层开标签报一条 finding。
//
// evidence: clawpro-portable-design-skill/SKILL.md#§7.4
// severity: P2
// stream:   audit

import { makeFinding } from './_shared.mjs';

const RULE_ID = 'surface-nesting';
const SEVERITY = 'P2';

// 不允许自我嵌套的 Surface 容器组件
const NESTABLE = ['SurfaceCard', 'TenantCard'];

function scanComponent(text, comp, file, findings) {
  // 匹配：开标签 <Comp ...>、自闭合 <Comp ... />、闭标签 </Comp>
  // 注意先判自闭合再判普通开标签。
  const re = new RegExp(`<\\/?${comp}\\b[^>]*?>`, 'g');
  let depth = 0;
  let m;
  while ((m = re.exec(text)) !== null) {
    const tag = m[0];
    const isClose = tag.startsWith('</');
    const isSelfClose = /\/>$/.test(tag);
    if (isClose) {
      depth = Math.max(0, depth - 1);
      continue;
    }
    if (isSelfClose) {
      // 自闭合不形成嵌套层级
      continue;
    }
    // 普通开标签
    depth += 1;
    if (depth >= 2) {
      // 该内层开标签所在行
      const before = text.slice(0, m.index);
      const line = before.split('\n').length;
      const lineText = text.split('\n')[line - 1] || '';
      findings.push(
        makeFinding({
          ruleId: RULE_ID,
          severity: SEVERITY,
          file,
          line,
          col: m.index - before.lastIndexOf('\n'),
          snippet: lineText.trim(),
          message: `${comp} 套娃（嵌套同名容器，当前深度 ${depth}），违反 §7.4`,
          evidence: 'SKILL.md#§7.4',
          suggestion:
            comp === 'SurfaceCard'
              ? '内层改用 <SurfaceInner>，不要 SurfaceCard 套 SurfaceCard'
              : '内层改用更轻的容器，不要同名卡片自我嵌套',
        }),
      );
    }
  }
}

export function run({ file, text }) {
  // 仅扫 tsx/jsx；CSS / 纯 ts 不含 JSX 标签
  if (!/\.(tsx|jsx)$/.test(file)) return [];
  const findings = [];
  for (const comp of NESTABLE) {
    if (text.includes(`<${comp}`)) scanComponent(text, comp, file, findings);
  }
  return findings;
}

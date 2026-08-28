/**
 * component-drift.mjs
 *
 * 检查 tsx 里 `<X variant="y" />` 的 `y` 是否在 spec 登记的 variants 白名单内。
 *
 * 数据源（真理源）：
 *   clawpro-walkthrough/fixtures/component-spec-index.json#<id>.variants
 *
 * 命中规则（v0.5 严守"宁可漏报不可误报"）：
 *   - X 必须是 PascalCase（组件名），小写 `<button>` 等 HTML 标签忽略。
 *   - 把 X 按 `PascalCase → kebab-case` 转 spec id，若该 id 在 spec-index 中存在且
 *     `variants.length > 0` 才比对：
 *       · `y` 不在 `variants` 列表 → P2（提示，因 spec 可能有漏抽，留 design-todo 二次裁决）
 *   - spec id 不存在 / variants 为空 → 静默放行。
 *
 * Severity：P2（建议级）
 *   把它压到 P2 是因为 extract-specs 是宽口径正则，不保证捕获 spec 里所有 variant
 *   字面。同 family 检测漂离这条规则 **不阻断**，让用户人工裁决；后续 spec 补完后
 *   可以升 P1。
 */

import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { scanRegex, makeFinding } from './_shared.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SPEC_INDEX_PATH = resolve(
  __dirname,
  '../../fixtures/component-spec-index.json',
);

const RULE_ID = 'component-drift';

/**
 * shadcn 内置 variant 兜底白名单。即便 spec 没显式登记，这些值也不算"漂离"。
 * 这是为了避免淹没真正"自创 variant"（如 `tenant-plain`、`link-dark`）的信号。
 * 用 `Map<componentId, Set<string>>`，键缺省即"任意组件都默认放行这些值"。
 */
const SHADCN_DEFAULTS = {
  // 任意组件级
  '*': new Set(['default']),
  button: new Set(['default', 'destructive', 'outline', 'secondary', 'ghost', 'link']),
  alert: new Set(['default', 'destructive']),
  badge: new Set(['default', 'secondary', 'destructive', 'outline']),
};

function isShadcnDefault(specId, variant) {
  if (SHADCN_DEFAULTS['*']?.has(variant)) return true;
  return SHADCN_DEFAULTS[specId]?.has(variant) ?? false;
}

/** PascalCase → kebab-case：`AdminSidebar` → `admin-sidebar`，`Button` → `button` */
function toKebab(name) {
  return name
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1-$2')
    .toLowerCase();
}

/** 懒加载 spec-index（同进程多次调用 run() 复用） */
let _specMap = null;
function loadSpecMap() {
  if (_specMap) return _specMap;
  const raw = JSON.parse(readFileSync(SPEC_INDEX_PATH, 'utf8'));
  _specMap = new Map();
  for (const item of raw.items) {
    if (item.variants && item.variants.length > 0) {
      _specMap.set(item.id, new Set(item.variants));
    }
  }
  return _specMap;
}

export function run({ file, text }) {
  const findings = [];
  // 只扫 tsx/jsx
  if (!/\.(tsx|jsx)$/.test(file)) return findings;

  const specMap = loadSpecMap();

  // 匹配 `<PascalCase ... variant="xxx" ...>`，组件名跨多行也行（用 lookbehind 容错单行）。
  // 这里采取保守做法：要求 variant prop 跟 PascalCase 标签同一行或紧邻，
  // 用一个"最近一次开标签"的状态机 → 用简单 per-line 扫描代替正则。
  const lines = text.split(/\r?\n/);
  // 维护"当前在哪个 PascalCase 开标签内"：栈不必要，只需记住最近一次开标签到它闭合 `>` 之间。
  let openTag = null; // { name, startLine }
  let openCol = 0;

  for (let i = 0; i < lines.length; i++) {
    const lineNo = i + 1;
    const lineText = lines[i];

    // 在本行里逐字扫描：开标签 `<Foo` / 关闭 `>` / 抓 variant=
    let j = 0;
    while (j < lineText.length) {
      if (!openTag) {
        // 查找下一个 `<PascalCase`
        const rest = lineText.slice(j);
        const m = rest.match(/<([A-Z][A-Za-z0-9]*)\b/);
        if (!m) break;
        openTag = { name: m[1], startLine: lineNo };
        openCol = j + m.index + 1;
        j += m.index + m[0].length;
      } else {
        // 在开标签内，找 variant="xxx" 或闭合 `>`
        const rest = lineText.slice(j);
        const vMatch = rest.match(/\bvariant\s*=\s*["']([a-z][a-z0-9-]*)["']/);
        const closeIdx = rest.indexOf('>');

        if (vMatch && (closeIdx === -1 || vMatch.index < closeIdx)) {
          // 找到 variant，先比对
          const variant = vMatch[1];
          const tagName = openTag.name;
          const specId = toKebab(tagName);
          const allow = specMap.get(specId);
          if (allow && !allow.has(variant) && !isShadcnDefault(specId, variant)) {
            findings.push(
              makeFinding({
                ruleId: RULE_ID,
                severity: 'P2',
                file,
                line: lineNo,
                col: j + vMatch.index + 1,
                snippet: lineText.trim(),
                message: `<${tagName} variant="${variant}"> 未在 spec 的 variants 白名单内（已知：${[...allow].join(', ')}）`,
                evidence: `component-spec-index.json#${specId}.variants`,
                suggestion: `若该 variant 真实存在请在 component-specs/${specId}.md 内补 \`variant="${variant}"\` 示例；若是笔误请改为白名单内 variant`,
              }),
            );
          }
          j += vMatch.index + vMatch[0].length;
        } else if (closeIdx !== -1) {
          openTag = null;
          j += closeIdx + 1;
        } else {
          // 这一行内既没 variant 也没闭合 → 进入下一行继续找
          break;
        }
      }
    }
  }

  return findings;
}

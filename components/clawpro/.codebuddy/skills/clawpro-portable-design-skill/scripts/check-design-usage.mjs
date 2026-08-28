#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const packageRoot = path.resolve(__dirname, '..');
const projectRoot = process.cwd();
const registryPath = path.resolve(packageRoot, 'assets/icon-registry.example.json');

const args = process.argv.slice(2);
const strict = args.includes('--strict') || process.env.STRICT === '1';
const verbose = args.includes('--verbose') || process.env.VERBOSE === '1';
// --json：给上游（如 clawpro-walkthrough）用的结构化输出模式，
// 关闭所有人类可读日志，stdout 只输出一行 JSON，便于走查 wrapper 解析。
const jsonMode = args.includes('--json');
const explicitTargets = args.filter((arg) => !arg.startsWith('--'));
const targets = explicitTargets.length > 0
  ? explicitTargets.map((target) => path.resolve(projectRoot, target))
  : [path.resolve(projectRoot, 'client/src')];

const allowMarkers = [
  'allow-design-legacy',
  'allow-shadow',
  'allow-radius',
  'allow-tenant-legacy',
  'allow-asset-legacy',
  'allow-shadcn-outline',
  'allow-inline-gradient',
  'allow-inline-svg',
  'allow-hardcoded-color',
];

// 9 个禁 lucide 槽位
// ⚠️ 真相源：clawpro-walkthrough/fixtures/icon-slots.json
//   - 此处枚举值必须与 fixture 中 slots[].id 完全一致（run-status 是 run-status-indicator 的别名）
//   - 修改前请先改 fixture，再跑 core/extractors/extract-icon-slots.mjs 校验
//   - admin/scripts/detectors/icon-slot.mjs 也读同一份 fixture
const forbiddenLucideSlots = [
  'number-card',
  'status-tag',
  'card-left-icon',
  'run-status',
  'feature-card',
  'admin-sidebar-icon',
  'batch-action-icon',
  'toggle-item-icon',
  'chart-legend-icon',
];

const textExtensions = new Set(['.ts', '.tsx', '.js', '.jsx', '.css', '.scss', '.html', '.md']);
const ignoredDirs = new Set(['node_modules', '.git', 'dist', 'build', 'coverage', '.codebuddy']);

function readRegistry() {
  try {
    const json = JSON.parse(fs.readFileSync(registryPath, 'utf8'));
    return new Set((json.icons || []).map((icon) => normalizeAssetPath(icon.path)));
  } catch (error) {
    return new Set();
  }
}

function normalizeAssetPath(value) {
  return value.replace(/^\.\//, '').replace(/^\//, '').replace(/^client\/public\//, '');
}

function shouldSkipLine(line) {
  return allowMarkers.some((marker) => line.includes(marker));
}

function walk(entry, files = []) {
  if (!fs.existsSync(entry)) return files;
  const stat = fs.statSync(entry);
  if (stat.isDirectory()) {
    const basename = path.basename(entry);
    if (ignoredDirs.has(basename)) return files;
    for (const child of fs.readdirSync(entry)) {
      walk(path.join(entry, child), files);
    }
    return files;
  }
  if (textExtensions.has(path.extname(entry))) files.push(entry);
  return files;
}

// ===== 7 个新增 C1 check 函数 =====

/**
 * C1.1: 检查 9 槽位禁 lucide 检测
 */
function checkProhibitedLucideSlots(files) {
  const violations = [];

  for (const file of files) {
    if (!file.endsWith('.tsx') && !file.endsWith('.ts')) continue;

    const content = fs.readFileSync(file, 'utf8');
    const lines = content.split(/\r?\n/);
    let currentComponent = '';

    // 检测文件是否属于禁 lucide 槽位
    let isForbiddenSlotFile = false;
    for (const slot of forbiddenLucideSlots) {
      if (file.includes(slot)) {
        isForbiddenSlotFile = true;
        currentComponent = slot;
        break;
      }
    }

    if (!isForbiddenSlotFile) continue;

    lines.forEach((line, index) => {
      if (shouldSkipLine(line)) return;

      // 检测 lucide-react 导入
      if (line.includes("from 'lucide-react'") || line.includes('from "lucide-react"')) {
        violations.push({
          file: path.relative(projectRoot, file),
          line: index + 1,
          rule: 'prohibited-lucide-slot',
          message: `Danger #1: Prohibited lucide usage in forbidden slot "${currentComponent}". Use resource-skill-map.json instead.`,
          source: line.trim(),
        });
      }
    });
  }

  return violations;
}

/**
 * C1.2: .css 文件硬编码色检测
 * 排除 CSS 变量定义块（:root {}）中的定义
 * 检测实际样式规则中的硬编码颜色
 */
function checkCssHardcodedColors(files) {
  const violations = [];

  for (const file of files) {
    if (!file.endsWith('.css') && !file.endsWith('.scss')) continue;

    const content = fs.readFileSync(file, 'utf8');
    const lines = content.split(/\r?\n/);

    let inRootBlock = false;
    let braceDepth = 0;

    lines.forEach((line, index) => {
      // 跟踪 :root 或 CSS 变量定义块
      if (line.includes(':root {') || line.includes('--') && line.includes('{')) {
        inRootBlock = true;
        braceDepth = (line.match(/{/g) || []).length - (line.match(/}/g) || []).length;
      }

      // 更新括号深度
      braceDepth += (line.match(/{/g) || []).length;
      braceDepth -= (line.match(/}/g) || []).length;

      if (braceDepth <= 0) {
        inRootBlock = false;
      }

      // 跳过注释和 :root 块中的内容
      if (line.includes('allow-inline-gradient') || inRootBlock || line.trim().startsWith('/*')) return;

      // 检查硬编码 hex 颜色（但排除 CSS 变量中的定义）
      // 只检查实际样式规则中的颜色，比如 "color: #1447E6" 而不是 "--variable: #ffffff"
      if (/#[0-9A-Fa-f]{3}(?![0-9A-Fa-f])|#[0-9A-Fa-f]{6}(?![0-9A-Fa-f])/.test(line)) {
        // 如果这行不是 CSS 变量定义（--xxx:），则标记为违规
        if (!line.match(/--[\w-]+\s*:/)) {
          violations.push({
            file: path.relative(projectRoot, file),
            line: index + 1,
            rule: 'css-hardcoded-color-hex',
            message: 'Hard-coded color in CSS. Use CSS variables (--cp-*) instead.',
            source: line.trim(),
          });
        }
      }

      // 检查硬编码 rgba 颜色（排除 CSS 变量定义和盒影）
      if (/rgba?\s*\(\s*\d+\s*,\s*\d+\s*,\s*\d+(?:\s*,\s*[\d.]+)?\s*\)/.test(line)) {
        if (!line.match(/--[\w-]+\s*:/) && !line.includes('box-shadow')) {
          violations.push({
            file: path.relative(projectRoot, file),
            line: index + 1,
            rule: 'css-hardcoded-rgba-color',
            message: 'Hard-coded rgba color in CSS. Use CSS variables (--cp-*) instead.',
            source: line.trim(),
          });
        }
      }
    });
  }

  return violations;
}

/**
 * C1.3: 语义 Tailwind 色检测
 */
function checkSemanticTailwindColors(files) {
  const violations = [];

  for (const file of files) {
    if (!file.endsWith('.tsx') && !file.endsWith('.ts') && !file.endsWith('.jsx')) continue;

    const content = fs.readFileSync(file, 'utf8');
    const lines = content.split(/\r?\n/);

    lines.forEach((line, index) => {
      if (line.includes('allow-design-legacy')) return;

      if (/(text|bg|border|ring|shadow|fill|stroke)-(gray|slate|stone|neutral|red|orange|yellow|green|emerald|teal|cyan|blue|indigo|purple|pink|rose|amber|lime|sky|violet|fuchsia)-(\d+)/.test(line)) {
        violations.push({
          file: path.relative(projectRoot, file),
          line: index + 1,
          rule: 'semantic-tailwind-color',
          message: 'Hard-coded Tailwind color. Use token variables (--cp-*) instead.',
          source: line.trim(),
        });
      }
    });
  }

  return violations;
}

/**
 * C1.4: CSS 变量中的大圆角值检测
 */
function checkCssVariableLargeRadius(files) {
  const violations = [];

  for (const file of files) {
    if (!file.endsWith('.css') && !file.endsWith('.scss')) continue;

    const content = fs.readFileSync(file, 'utf8');
    const lines = content.split(/\r?\n/);

    lines.forEach((line, index) => {
      if (shouldSkipLine(line)) return;

      if (/border-radius\s*:|--[\w-]*radius\s*:/.test(line)) {
        const radiusMatch = line.match(/:\s*([\d.]+)(px|rem|em)/);
        if (radiusMatch) {
          const value = parseFloat(radiusMatch[1]);
          const unit = radiusMatch[2];

          let valueInPx = value;
          if (unit === 'rem') valueInPx = value * 16;
          else if (unit === 'em') valueInPx = value * 16;

          if (valueInPx >= 8) {
            violations.push({
              file: path.relative(projectRoot, file),
              line: index + 1,
              rule: 'css-large-radius',
              message: 'Large radius value in CSS (>=8px). Confirm alignment with endpoint rules.',
              source: line.trim(),
            });
          }
        }
      }
    });
  }

  return violations;
}

/**
 * C1.5: inline SVG 标记检测
 */
function checkInlineSvgMarking(files) {
  const violations = [];

  for (const file of files) {
    if (!file.endsWith('.tsx') && !file.endsWith('.jsx') && !file.endsWith('.ts')) continue;

    const content = fs.readFileSync(file, 'utf8');
    const lines = content.split(/\r?\n/);

    let inSvg = false;
    let svgStartLine = 0;
    let svgContent = [];

    lines.forEach((line, index) => {
      if (line.includes('allow-inline-svg')) return;

      if (/<svg\s+[^>]*>/.test(line) || /^[\s]*<svg\s/.test(line)) {
        inSvg = true;
        svgStartLine = index + 1;
        svgContent = [];
      }

      if (inSvg) {
        svgContent.push(line);
      }

      if (inSvg && /<\/svg>/.test(line)) {
        const svgBlock = svgContent.join('\n');
        const hasPathElements = svgBlock.includes('<path') || svgBlock.includes('<circle') || svgBlock.includes('<line') || svgBlock.includes('<rect');
        const hasAllowMarker = svgBlock.includes('allow-inline-svg') || svgBlock.includes('allow-design-legacy');

        if (hasPathElements && !hasAllowMarker) {
          violations.push({
            file: path.relative(projectRoot, file),
            line: svgStartLine,
            rule: 'inline-svg-without-marker',
            message: 'Inline SVG without proper marker. Add /* allow-design-legacy: portable-lucide-svg-inline */ if intentional.',
            source: `SVG block at line ${svgStartLine}`,
          });
        }

        inSvg = false;
        svgContent = [];
      }
    });
  }

  return violations;
}

/**
 * C1.6: needs-design-confirmation 标记收集
 */
function collectDesignConfirmationMarkers(files) {
  const markers = [];

  for (const file of files) {
    const content = fs.readFileSync(file, 'utf8');
    const lines = content.split(/\r?\n/);

    lines.forEach((line, index) => {
      const confirmationMatch = line.match(/needs-design-confirmation:\s*(.+?)(?:\s*\*\/|\s*\/\/|$)/);
      if (confirmationMatch) {
        markers.push({
          file: path.relative(projectRoot, file),
          line: index + 1,
          details: confirmationMatch[1].trim(),
          source: line.trim(),
        });
      }
    });
  }

  return markers;
}

/**
 * C1.7a: icon-registry 一致性检查
 */
function checkIconRegistryConsistency(files, registeredIcons) {
  const violations = [];

  for (const file of files) {
    const content = fs.readFileSync(file, 'utf8');
    const iconRefMatches = content.matchAll(/['"`]\/?(icon\/[^'"`]+?\.svg)['"`]/g);

    for (const match of iconRefMatches) {
      const ref = normalizeAssetPath(match[1]);
      if (!registeredIcons.has(ref)) {
        violations.push({
          file: path.relative(projectRoot, file),
          line: 0,
          rule: 'icon-registry-broken',
          message: `Icon registry reference broken or outdated: ${ref}`,
          source: match[0],
        });
      }
    }
  }

  return violations;
}

/**
 * C1.7b: 多 variant 同步检查
 */
function checkMultiVariantSync(files) {
  const violations = [];

  const componentVariantGroups = {
    'admin-sidebar': files.filter(f => f.includes('admin-sidebar')),
    'alert': files.filter(f => f.includes('alert') && !f.includes('admin-sidebar')),
  };

  for (const [component, componentFiles] of Object.entries(componentVariantGroups)) {
    if (componentFiles.length < 2) continue;

    const iconUsageByVariant = new Map();

    for (const file of componentFiles) {
      const content = fs.readFileSync(file, 'utf8');
      const iconMatches = content.matchAll(/(?:icon\w*|Icon)\s*[:=]\s*['"]{1}([^'"]+)['"]{1}/g);
      const icons = new Set();

      for (const match of iconMatches) {
        icons.add(match[1]);
      }

      if (icons.size > 0) {
        const variantName = file.match(/--([a-z]+)/) ? file.match(/--([a-z]+)/)[1] : path.basename(file);
        iconUsageByVariant.set(variantName, icons);
      }
    }

    if (iconUsageByVariant.size > 1) {
      const variants = Array.from(iconUsageByVariant.values());
      const firstVariantIcons = variants[0];
      let isConsistent = true;

      for (let i = 1; i < variants.length; i++) {
        if (variants[i].size !== firstVariantIcons.size ||
            ![...variants[i]].every(icon => firstVariantIcons.has(icon))) {
          isConsistent = false;
          break;
        }
      }

      if (!isConsistent) {
        violations.push({
          file: componentFiles[0],
          line: 0,
          rule: 'multi-variant-inconsistency',
          message: `Icon mapping inconsistency between variants of ${component}`,
          source: `Check all ${component} variant files for consistent icon mapping`,
        });
      }
    }
  }

  return violations;
}

// ===== 主程序 =====

const registeredIcons = readRegistry();
const files = targets.flatMap((target) => walk(target));

// 执行 7 个新的 C1 check 函数
const checks = {
  'C1.1 Icon slot validation': checkProhibitedLucideSlots(files),
  'C1.2 CSS hard-coded colors': checkCssHardcodedColors(files),
  'C1.3 Semantic Tailwind colors': checkSemanticTailwindColors(files),
  'C1.4 CSS large radius': checkCssVariableLargeRadius(files),
  'C1.5 Inline SVG markers': checkInlineSvgMarking(files),
  'C1.6 Icon registry consistency': checkIconRegistryConsistency(files, registeredIcons),
  'C1.7 Multi-variant sync': checkMultiVariantSync(files),
};

const confirmationItems = collectDesignConfirmationMarkers(files);

// 计算统计
let totalViolations = 0;
for (const v of Object.values(checks)) {
  totalViolations += v.length;
}

// ===== JSON 模式：给上游 walkthrough wrapper 用 =====
if (jsonMode) {
  const flatViolations = [];
  for (const [checkName, checkViolations] of Object.entries(checks)) {
    for (const v of checkViolations) {
      flatViolations.push({ checkName, ...v });
    }
  }
  process.stdout.write(
    JSON.stringify({
      tool: 'check-design-usage',
      total: totalViolations,
      checks: Object.fromEntries(
        Object.entries(checks).map(([k, v]) => [k, v.length]),
      ),
      violations: flatViolations,
      confirmations: confirmationItems,
    }),
  );
  // 显式 newline 让管道更友好
  process.stdout.write('\n');
  process.exit(strict && totalViolations > 0 ? 1 : 0);
}

// ===== 输出汇总报告 =====

console.log('\n=== ClawPro Design Usage Check Report (Phase C1) ===\n');

let passCount = 0;
let failCount = 0;

for (const [checkName, checkViolations] of Object.entries(checks)) {
  if (checkViolations.length === 0) {
    console.log(`✓ ${checkName}: PASS (0 violations)`);
    passCount++;
  } else {
    console.log(`✗ ${checkName}: FAIL (${checkViolations.length} violations)`);
    checkViolations.slice(0, 3).forEach(v => {
      console.log(`  - ${v.file}:${v.line} ${v.rule}`);
      if (verbose) console.log(`    ${v.source.substring(0, 80)}`);
    });
    if (checkViolations.length > 3) {
      console.log(`  ... and ${checkViolations.length - 3} more violations`);
    }
    failCount++;
  }
}

// 设计确认标记统计
console.log(`\n📋 Design Confirmations Needed: ${confirmationItems.length} items`);
if (confirmationItems.length > 0 && verbose) {
  confirmationItems.slice(0, 5).forEach(item => {
    console.log(`  - ${item.file}:${item.line} ${item.details.substring(0, 60)}`);
  });
  if (confirmationItems.length > 5) {
    console.log(`  ... and ${confirmationItems.length - 5} more items`);
  }
}

// 最终总结
console.log(`\n=== Summary ===`);
console.log(`Total checks: ${Object.keys(checks).length}`);
console.log(`PASSED: ${passCount}`);
console.log(`FAILED: ${failCount}`);
console.log(`Total violations: ${totalViolations}`);

// 保存确认清单供 C2 使用
if (confirmationItems.length > 0) {
  const todoPath = path.resolve(packageRoot, 'references/icon-design-todo-raw.json');
  fs.writeFileSync(todoPath, JSON.stringify({
    generated_at: new Date().toISOString(),
    total_items: confirmationItems.length,
    items: confirmationItems,
  }, null, 2));
  console.log(`\n✓ Design confirmation items saved to icon-design-todo-raw.json`);
}

// Exit code
if (strict && (totalViolations > 0 || failCount > 0)) {
  console.log('\nClawPro design check FAILED. Use --strict to enforce.');
  process.exit(1);
} else if (totalViolations > 0 || failCount > 0) {
  console.log('\nClawPro design check found issues. Use --strict to enforce or fix issues.');
  process.exit(0);
} else {
  console.log('\nClawPro design usage check PASSED.');
  process.exit(0);
}

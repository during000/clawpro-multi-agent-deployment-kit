#!/usr/bin/env node
// walkthrough.mjs —— clawpro-walkthrough v0.1 主入口
//
// 用法：
//   node walkthrough.mjs audit <target>          # 跑全部 detector
//   node walkthrough.mjs radius <target>         # 只跑 radius
//   node walkthrough.mjs color <target>          # 只跑 color
//   node walkthrough.mjs icon-slot <target>      # 只跑 icon-slot
//   node walkthrough.mjs diff                    # 对当前 git diff 跑
//   node walkthrough.mjs explain <ruleId>        # 打印规则定义
//
// <target> 可以是文件或目录；目录会递归扫 .tsx/.ts/.jsx/.js/.css，跳过 node_modules / dist / .codebuddy。

import { readFileSync, writeFileSync, mkdirSync, statSync, readdirSync, existsSync } from 'node:fs';
import { resolve, dirname, relative, extname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execSync, execFileSync } from 'node:child_process';

import { run as runRadius } from './detectors/radius.mjs';
import { run as runColor } from './detectors/color.mjs';
import { run as runIconSlot } from './detectors/icon-slot.mjs';
import { run as runShadow } from './detectors/shadow.mjs';
import { run as runComponentDrift } from './detectors/component-drift.mjs';
import { run as runPageRecipeMatch } from './detectors/page-recipe-match.mjs';
import { run as runTypography } from './detectors/typography.mjs';
import { run as runSurfaceNesting } from './detectors/surface-nesting.mjs';
import { run as runSpacing } from './detectors/spacing.mjs';
// 外部 detector：调用 clawpro-portable-design-skill/scripts/ 下的老脚本
import { runBatch as runExternalDesignSkill } from './detectors/external-design-skill.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SKILL_ROOT = resolve(__dirname, '..');                            // clawpro-walkthrough
// 仓库根：默认假设 skill 装在标准位置 <repo>/.codebuddy/skills/clawpro-walkthrough；
// 装到别处（或想扫别的仓库）时用环境变量 WALKTHROUGH_PROJECT_ROOT 覆盖。
const PROJECT_ROOT = process.env.WALKTHROUGH_PROJECT_ROOT
  ? resolve(process.env.WALKTHROUGH_PROJECT_ROOT)
  : resolve(SKILL_ROOT, '../../..');
// 产物目录默认落在仓库根 _walkthrough/，可用 WALKTHROUGH_OUT_DIR 覆盖。
const OUT_DIR = process.env.WALKTHROUGH_OUT_DIR
  ? resolve(process.env.WALKTHROUGH_OUT_DIR)
  : resolve(PROJECT_ROOT, '_walkthrough');

const DETECTORS = {
  radius: runRadius,
  color: runColor,
  'icon-slot': runIconSlot,
  shadow: runShadow,
  'component-drift': runComponentDrift,
  'page-recipe-match': runPageRecipeMatch,
  'text-color': runTypography,
  'surface-nesting': runSurfaceNesting,
  'spacing-grouping': runSpacing,
};

const EXT_WHITELIST = new Set(['.tsx', '.ts', '.jsx', '.js', '.css']);
const DIR_BLACKLIST = new Set(['node_modules', 'dist', 'build', '.git', '.codebuddy', '_walkthrough']);

// ---------- 工具 ----------

function timestamp() {
  const d = new Date();
  const pad = (n) => String(n).padStart(2, '0');
  return (
    d.getFullYear().toString() +
    pad(d.getMonth() + 1) +
    pad(d.getDate()) +
    '-' +
    pad(d.getHours()) +
    pad(d.getMinutes())
  );
}

function* walk(dirOrFile) {
  if (!existsSync(dirOrFile)) return;
  const st = statSync(dirOrFile);
  if (st.isFile()) {
    if (EXT_WHITELIST.has(extname(dirOrFile))) yield dirOrFile;
    return;
  }
  for (const name of readdirSync(dirOrFile)) {
    if (DIR_BLACKLIST.has(name)) continue;
    const child = resolve(dirOrFile, name);
    const cst = statSync(child);
    if (cst.isDirectory()) {
      yield* walk(child);
    } else if (cst.isFile() && EXT_WHITELIST.has(extname(child))) {
      yield child;
    }
  }
}

// 目标必填：对外交付不再假设任何默认工程目录（原来会 fallback 到 client/src）。
// 用户必须显式传入扫描目标（文件或目录）；缺省直接报错退出（exit 2 = 入参错误）。
function requireTargets(rest, cmd) {
  if (!rest || rest.length === 0) {
    console.error(`[walkthrough] "${cmd}" 需要显式传入扫描目标（文件或目录）。`);
    console.error(`  例：node scripts/walkthrough.mjs ${cmd} <你的工程>/src/pages/Some.tsx`);
    console.error('  说明：为便于对外交付，已移除 client/src 默认目标，请按自己的工程结构传参。');
    console.error('  如需覆盖仓库根，可设环境变量 WALKTHROUGH_PROJECT_ROOT=<repo>。');
    process.exit(2);
  }
  return rest.map((p) => resolve(process.cwd(), p));
}

function csvEscape(v) {
  if (v == null) return '';
  const s = String(v);
  if (/[",\n\r]/.test(s)) {
    return '"' + s.replace(/"/g, '""') + '"';
  }
  return s;
}

function findingsToCsv(findings) {
  const header = [
    'ruleId',
    'severity',
    'file',
    'line',
    'col',
    'snippet',
    'message',
    'evidence',
    'suggestion',
  ];
  const rows = [header.join(',')];
  for (const f of findings) {
    rows.push(
      [
        f.ruleId,
        f.severity,
        f.file,
        f.line,
        f.col,
        f.snippet,
        f.message,
        f.evidence,
        f.suggestion,
      ]
        .map(csvEscape)
        .join(','),
    );
  }
  return rows.join('\n') + '\n';
}

// design-todo.csv —— 给用户拍板的待裁决清单（列定义见 DESIGN.md §6.2 + SKILL §0.C 的 conflict_type）
// 列：冲突类型, 所属页面, 槽位/位置, 问题描述, AI 当前处理, 建议, 展示台对照, 真实页面参照, 用户裁决
function todosToCsv(todos) {
  const header = [
    '冲突类型',
    '所属页面',
    '槽位/位置',
    '问题描述',
    'AI 当前处理',
    '建议',
    '展示台对照',
    '真实页面参照',
    '用户裁决',
  ];
  const rows = [header.join(',')];
  for (const t of todos) {
    rows.push(
      [
        t.conflictType || '待分类',
        t.file,
        `${t.ruleId}:${t.line}`,
        t.message,
        '这类拿不准的不自动改，进 design-todo 待用户裁决',
        t.suggestion,
        '', // 展示台对照（webUI / 人工补）
        '', // 真实页面参照（page-references 截图，人工补）
        '待裁决',
      ]
        .map(csvEscape)
        .join(','),
    );
  }
  return rows.join('\n') + '\n';
}

// ---------- 健康分模型（DESIGN.md §1.5.2：audit 8 项 × 0~4 = 32） ----------
// 每个维度按命中数评分；维度内出现 P0 则封顶 1 分（铁律命中重罚）。
// todo 流（待裁决）不参与扣分——研究中的冲突不该压低健康分。
const AUDIT_DIMENSIONS = [
  { key: '圆角', rules: ['radius', 'external/css-large-radius'] },
  {
    key: '颜色 token',
    rules: [
      'color',
      'external/css-hardcoded-color-hex',
      'external/css-hardcoded-rgba-color',
      'external/semantic-tailwind-color',
    ],
  },
  { key: '阴影', rules: ['shadow'] },
  { key: '组件复用', rules: ['component-drift', 'surface-nesting', 'external/multi-variant-sync'] },
  { key: '图标槽位', rules: ['icon-slot', 'external/prohibited-lucide-slot', 'external/icon-registry-inconsistent'] },
  { key: '内联样式漂移', rules: ['text-color', 'external/inline-svg-marking'] },
  { key: '页面骨架', rules: ['page-recipe-match'] },
  { key: 'portable 一致性', rules: ['external/spec-symbol-ghost'] },
];

function scoreByCount(n) {
  if (n === 0) return 4;
  if (n <= 2) return 3;
  if (n <= 5) return 2;
  if (n <= 10) return 1;
  return 0;
}

// 只对 audit 流的 finding 评分
function computeAuditHealth(findings) {
  const auditFindings = findings.filter((f) => f.stream !== 'todo');
  const dims = [];
  let total = 0;
  for (const d of AUDIT_DIMENSIONS) {
    const matched = auditFindings.filter((f) => d.rules.includes(f.ruleId));
    let score = scoreByCount(matched.length);
    const hasP0 = matched.some((f) => f.severity === 'P0');
    if (hasP0) score = Math.min(score, 1);
    dims.push({ key: d.key, score, count: matched.length, hasP0 });
    total += score;
  }
  return { total, max: 32, dims };
}

// 总分评分带（DESIGN.md §1.5.2）；当只有 audit 分时按比例换算到 56 制再判带
function gradeBand(total, max = 56) {
  const t = max === 56 ? total : Math.round((total / max) * 56);
  if (t >= 50) return 'Excellent';
  if (t >= 40) return 'Good';
  if (t >= 28) return 'Acceptable';
  if (t >= 16) return 'Poor';
  return 'Critical';
}

// 目标路径 → snapshot slug
function slugify(targets) {
  if (!targets || targets.length === 0) return 'all';
  const first = targets[0];
  let rel;
  try {
    rel = relative(PROJECT_ROOT, first);
  } catch {
    rel = String(first);
  }
  rel = rel.replace(/\.(tsx|ts|jsx|js|css)$/, '');
  const slug = rel.replace(/[\\/]/g, '-').replace(/[^A-Za-z0-9._-]/g, '').replace(/^-+/, '');
  return slug || 'all';
}

// ---------- 核心 ----------

function runOnFile(absFile, rulesToRun) {
  const text = readFileSync(absFile, 'utf8');
  const rel = relative(PROJECT_ROOT, absFile);
  const out = [];
  for (const id of rulesToRun) {
    const fn = DETECTORS[id];
    if (!fn) continue;
    const fs = fn({ file: rel, text });
    out.push(...fs);
  }
  return out;
}

function audit(targets, rulesToRun, { withExternal = false } = {}) {
  const files = [];
  for (const t of targets) {
    for (const f of walk(t)) files.push(f);
  }
  const allFindings = [];
  for (const f of files) {
    allFindings.push(...runOnFile(f, rulesToRun));
  }
  // 外部 detector（设计 skill 老脚本）只在 audit / diff 全量模式跑一次
  if (withExternal) {
    try {
      allFindings.push(...runExternalDesignSkill(targets));
    } catch (e) {
      console.error('[walkthrough] external detector 失败（已忽略）:', e?.message || e);
    }
  }
  return { files: files.length, findings: allFindings };
}

function gitDiffFiles() {
  let stdout;
  try {
    stdout = execSync('git diff --name-only HEAD', { cwd: PROJECT_ROOT, encoding: 'utf8' });
  } catch (e) {
    stdout = execSync('git diff --name-only', { cwd: PROJECT_ROOT, encoding: 'utf8' });
  }
  return stdout
    .split('\n')
    .filter(Boolean)
    .map((p) => resolve(PROJECT_ROOT, p))
    .filter((p) => existsSync(p) && statSync(p).isFile() && EXT_WHITELIST.has(extname(p)));
}

function writeSnapshot(label, runArgs, rulesUsed, { files, findings }, targets) {
  const ts = timestamp();
  const dir = resolve(OUT_DIR, ts);
  mkdirSync(dir, { recursive: true });

  // 分流：audit（确定违规，AI 直接修）vs todo（待裁决，给用户）
  const auditFindings = findings.filter((f) => f.stream !== 'todo');
  const todoFindings = findings.filter((f) => f.stream === 'todo');

  // audit-report.csv / .json（只含 audit 流）
  writeFileSync(resolve(dir, 'audit-report.csv'), findingsToCsv(auditFindings), 'utf8');
  writeFileSync(
    resolve(dir, 'audit-report.json'),
    JSON.stringify({ files, count: auditFindings.length, findings: auditFindings }, null, 2) + '\n',
    'utf8',
  );

  // design-todo.csv / .json（只含 todo 流；为空也写表头，方便 webUI / polish 读取）
  writeFileSync(resolve(dir, 'design-todo.csv'), todosToCsv(todoFindings), 'utf8');
  writeFileSync(
    resolve(dir, 'design-todo.json'),
    JSON.stringify({ count: todoFindings.length, todos: todoFindings }, null, 2) + '\n',
    'utf8',
  );

  // audit 健康分（0~32）
  const health = computeAuditHealth(findings);

  // meta（bySeverity / findingsCount 只统计 audit 流，确保退出码不被待裁决项污染）
  const meta = {
    label,
    runArgs,
    rulesUsed,
    filesScanned: files,
    findingsCount: auditFindings.length,
    todoCount: todoFindings.length,
    bySeverity: countBy(auditFindings, 'severity'),
    byRule: countBy(auditFindings, 'ruleId'),
    todoByConflictType: countBy(todoFindings, 'conflictType'),
    auditHealth: health,
    slug: slugify(targets),
    timestamp: ts,
    extractedAt: new Date().toISOString(),
  };
  writeFileSync(resolve(dir, 'meta.json'), JSON.stringify(meta, null, 2) + '\n', 'utf8');

  // CI 友好：写"最新快照"指针文件，供 CI 稳定定位本次快照。
  // 不能用 `ls -t _walkthrough | head -1`——appendTrend 之后才创建/刷新 snapshots/ 目录，
  // 在全新 checkout（CI 环境）里 snapshots/ 的 mtime 会晚于时间戳快照目录，导致误抓 snapshots。
  try {
    writeFileSync(resolve(OUT_DIR, 'LATEST'), ts + '\n', 'utf8');
  } catch {
    /* noop：指针写失败不影响主流程，CI 侧有 glob 兜底 */
  }

  // 持久化趋势（DESIGN §1.5.3）：审计类命令写 snapshots/<slug>/trend.json
  if (label === 'audit') {
    appendTrend(meta.slug, { ts, audit: health.total, critique: null, total: null, label });
  }

  return { dir, meta };
}

// snapshots/<slug>/trend.json 追加一条
function appendTrend(slug, entry) {
  const snapDir = resolve(OUT_DIR, 'snapshots', slug);
  mkdirSync(snapDir, { recursive: true });
  const p = resolve(snapDir, 'trend.json');
  let arr = [];
  if (existsSync(p)) {
    try {
      arr = JSON.parse(readFileSync(p, 'utf8'));
      if (!Array.isArray(arr)) arr = [];
    } catch {
      arr = [];
    }
  }
  arr.push(entry);
  writeFileSync(p, JSON.stringify(arr, null, 2) + '\n', 'utf8');
}

function countBy(arr, key) {
  const out = {};
  for (const x of arr) out[x[key]] = (out[x[key]] || 0) + 1;
  return out;
}

function printSummary(meta) {
  console.log('');
  console.log('[walkthrough] audit done');
  console.log('  files   :', meta.filesScanned);
  console.log('  findings:', meta.findingsCount, '(audit 流)');
  console.log('  todo    :', meta.todoCount, '(待裁决 → design-todo.csv)');
  console.log('  bySev   :', JSON.stringify(meta.bySeverity));
  console.log('  byRule  :', JSON.stringify(meta.byRule));
  if (meta.auditHealth) {
    const h = meta.auditHealth;
    console.log(
      `  health  : ${h.total}/${h.max}（audit 维度分；总分需 critique 合并，评分带 ${gradeBand(h.total, h.max)}）`,
    );
  }
  console.log('  out     :', relative(PROJECT_ROOT, resolve(OUT_DIR, meta.timestamp)));
}

// ---------- 退出码规范（CI 用） ----------
//   0  = 全绿（无 finding，或只有低于阻断阈值的 finding）
//   1  = 有阻断级 finding（默认 P0；可通过 WALKTHROUGH_BLOCK_LEVEL 调整为 P1）
//   2  = 入参错误 / 脚本异常（见 main 末尾 try/catch）
//
// 默认只对 `diff` 子命令生效（PR CI 增量审计）。
// 全量 `audit` 不开退出码——历史存量不应卡 CI；如显式需要，设 WALKTHROUGH_BLOCK_ON_AUDIT=1。
function computeExitCode(label, meta) {
  const blockLevel = (process.env.WALKTHROUGH_BLOCK_LEVEL || 'P0').toUpperCase();
  const blockOnAudit = process.env.WALKTHROUGH_BLOCK_ON_AUDIT === '1';

  const isDiff = label === 'diff';
  const isAudit = label === 'audit';
  // 单规则模式（radius / color / icon-slot / ...）也启用退出码——
  // 用户显式查某一类时，命中即视为需要修。
  const isSingleRule = label !== 'diff' && label !== 'audit' && label !== 'explain';

  const shouldBlock = isDiff || isSingleRule || (isAudit && blockOnAudit);
  if (!shouldBlock) return 0;

  const sev = meta.bySeverity || {};
  const levels = ['P0', 'P1', 'P2', 'P3'];
  const startIdx = levels.indexOf(blockLevel);
  if (startIdx < 0) return 0;
  for (let i = startIdx; i < levels.length; i += 1) {
    if ((sev[levels[i]] || 0) > 0) return 1;
  }
  return 0;
}

const EXPLAIN = {
  radius: {
    severity: 'P0',
    summary: '管控端圆角必须为 4px（var(--radius)）',
    evidence: 'tokens.json#radius.--radius=4px',
    source: 'clawpro-portable-design-skill/portable/css/tokens.css',
    checks: [
      'Tailwind 关键字：rounded-sm/md/lg/xl/2xl/3xl/full 全部禁用',
      'Tailwind 任意值：rounded-[*]，仅允许 4px / 0.25rem / var(--radius)',
      'JS 内联：borderRadius，仅允许 0 / 4 / 4px / 0.25rem / var(--radius)',
      'CSS 文件：border-radius，仅允许 4px / 0 / 0.25rem / var(--radius)',
    ],
  },
  color: {
    severity: 'P1',
    summary: '颜色必须走 design token，禁止硬编码 hex / Tailwind 任意色',
    evidence: 'tokens.json#colors',
    source: 'clawpro-portable-design-skill/portable/css/tokens.css',
    checks: [
      '裸 hex（#1447E6 等）：自动回查 token，建议改为 var(--cp-xxx)',
      'Tailwind 任意色（text-[#...] / bg-[#...] / border-[#...] 等）',
    ],
  },
  'icon-slot': {
    severity: 'P0',
    summary: '9 类不可回退槽位禁用 lucide-react，必须用 ClawPro 自研 SVG',
    evidence: 'clawpro-portable-design-skill/SKILL.md#§0-管控端图标九槽位',
    source: 'clawpro-portable-design-skill/SKILL.md',
    checks: [
      'v0.1 宽口径：只要 import "lucide-react" 就抛 P0，待 0.5 接入 9 槽位精确匹配',
    ],
  },
  shadow: {
    severity: 'P1/P2',
    summary: '阴影必须走 tokens.css 的 --cp-shadow-* 体系（6 个）',
    evidence: 'tokens.json#shadows',
    source: 'clawpro-portable-design-skill/portable/css/tokens.css',
    checks: [
      'CSS 硬编码 box-shadow（非 var(--cp-shadow-*) / 非 none）→ P1',
      'Tailwind 任意值 shadow-[...]（非 var(--cp-shadow-*)）→ P1',
      'Tailwind 框架级 shadow-sm/md/lg/xl/2xl/inner → P2（建议改 var）',
    ],
  },
  'component-drift': {
    severity: 'P2',
    summary: '组件 variant 必须落在该组件 spec 登记的 variants 白名单内',
    evidence: 'component-spec-index.json#<id>.variants',
    source: 'clawpro-portable-design-skill/component-specs/*.md',
    checks: [
      'tsx 中 <PascalCase variant="xxx"> 的 xxx 不在 spec 白名单 → P2',
      '仅对 component-spec-index.json 中 variants.length > 0 的组件生效',
      '组件名 PascalCase → kebab-case 后查 spec id（Button→button、StatusTag→status-tag）',
      'spec 无 variants 登记或 spec 不存在 → 静默放行',
      'shadcn 内置 variant（button: default/destructive/outline/secondary/ghost/link 等）作为二级兜底白名单 → 静默放行（spec 漏抽不该淹没真自创信号）',
    ],
  },
  'page-recipe-match': {
    severity: 'P2',
    summary: '页面骨架完整性：page entry 应 import recipe 登记的全部 required_components',
    evidence: 'page-recipes.json#<id>.required_components',
    source: 'clawpro-portable-design-skill/assets/page-references/*.md',
    checks: [
      '仅对 recipe.source[0] 命中的 page entry 文件生效（避免同一缺件被多文件重复抛）',
      'bundle = page entry + 同目录递归 .tsx/.ts/.jsx/.js 的 import 联合集合',
      'required_components 里**没有**出现在 bundle 任一 import 路径里 → P2',
      'antipatterns / required_specs / skeleton 关键词在 v0.1 不参与匹配（误报率高）',
    ],
  },
  'text-color': {
    severity: 'P2',
    summary: '文字色必须走 Typography 语义或 --cp-text-* token，禁用 Tailwind 内置中性灰阶',
    evidence: 'SKILL.md#§2.5 + tokens.json#colors(--cp-text-*)',
    source: 'clawpro-portable-design-skill/SKILL.md §2.5',
    checks: [
      'text-{gray,slate,zinc,neutral,stone}-NNN 用于文字色 → P2',
      '与 color detector 互补：color 抓 hex/任意色，本规则抓 Tailwind 内置中性色阶',
      '建议映射：900+→emphasis/title、700+→body/secondary、500+→muted、其余→weak',
    ],
  },
  'surface-nesting': {
    severity: 'P2',
    summary: 'Surface 容器不能套娃（SurfaceCard 套 SurfaceCard），内层须降到 SurfaceInner',
    evidence: 'SKILL.md#§7.4',
    source: 'clawpro-portable-design-skill/SKILL.md §7.4',
    checks: [
      '栈式扫描 SurfaceCard / TenantCard 开闭标签，同名深度 >= 2 → P2',
      '自闭合标签不计入嵌套层级',
      '仅扫 .tsx/.jsx',
    ],
  },
  'spacing-grouping': {
    severity: 'P2（→ design-todo 待裁决）',
    summary: '相邻控件各自加水平 margin 疑似未成组，应改 flex+gap；判断题进 design-todo',
    evidence: 'SKILL.md#§2.7',
    source: 'clawpro-portable-design-skill/SKILL.md §2.7',
    checks: [
      'mr-/ml-（含任意值）命中按行聚类，同簇（≤4 行）出现 >=2 次 → 报 1 条',
      'stream=todo：不进 audit-report，不参与退出码与健康分扣分',
      'conflict_type = Spec vs 现状（§2.7 间距成组），交用户裁决',
    ],
  },
  'external-design-skill': {
    severity: 'P0~P2（按子规则映射）',
    summary:
      '复用 clawpro-portable-design-skill/scripts/ 下的老脚本作为外部 detector，扩展走查覆盖面',
    evidence: 'clawpro-portable-design-skill/scripts/check-design-usage.mjs + check-spec-symbols.mjs',
    source: 'clawpro-portable-design-skill/scripts/',
    checks: [
      'check-design-usage --json: 8 个检查函数（lucide 槽位禁回退 / 硬编码色 hex / 语义 tailwind 色 / 大圆角 / inline SVG 标记 / 注册表一致 / variant 同步 / needs-design-confirmation 收集）',
      'check-spec-symbols --json: spec 文档里 import 的 identifier 在 client/src 里是否真实存在（防 ghost reference）',
      '仅在 audit 全量模式启用；diff 模式跳过（老脚本是全量扫描，噪声大）',
      '设 WALKTHROUGH_SKIP_EXTERNAL=1 可整体跳过',
      '映射后的 ruleId 形如 external/<原 rule>，severity 见 external-design-skill.mjs#SEVERITY_MAP',
      '⚠️ 第 8 个 check `collectDesignConfirmationMarkers` 不抛违规，仅收集 `needs-design-confirmation` 标记（落实设计 skill §1.5 / §12 冲突铁律）',
    ],
  },
};

// ---------- critique / polish / trend（roadmap 0.5：critique→audit→polish 三段闭环） ----------

// critique 的 6 个维度（DESIGN §1.5.2，0~4 分，总分 24）
const CRITIQUE_DIMENSIONS = [
  { key: 'AI slop 痕迹', hint: '有无机械重复 / 假数据 / 无意义占位 / 千篇一律的卡片堆叠' },
  { key: '视觉层级', hint: '主次是否分明，标题/正文/辅助文字层级是否清晰' },
  { key: '信息架构', hint: '分组、留白、对齐是否合理，能否一眼读懂页面意图' },
  { key: '一致性', hint: '是否贴合 ClawPro 设计语言（组件选型 / token / 间距节奏）' },
  { key: '文案', hint: '措辞是否准确、口吻是否统一、有无错别字 / 英文残留' },
  { key: '状态完备性', hint: '空态 / 加载 / 错误 / 极值 是否都覆盖' },
];

// 扫 _walkthrough 下的时间戳快照目录（排除 snapshots/），按时间返回含指定文件的最新目录。
// wantSlug 提供时优先匹配同 slug 的快照（slug 存于 audit 的 meta.json / critique 的 critique.json）。
function findLatestSnapshot(fileName, wantSlug) {
  if (!existsSync(OUT_DIR)) return null;
  const dirs = readdirSync(OUT_DIR)
    .filter((n) => n !== 'snapshots')
    .map((n) => resolve(OUT_DIR, n))
    .filter((p) => statSync(p).isDirectory())
    .filter((p) => existsSync(resolve(p, fileName)))
    .sort();
  if (!dirs.length) return null;
  if (wantSlug) {
    const slugFile = fileName === 'audit-report.json' ? 'meta.json' : fileName;
    const matched = dirs.filter((d) => {
      const j = readJsonSafe(resolve(d, slugFile));
      return j && j.slug === wantSlug;
    });
    if (matched.length) return matched[matched.length - 1];
  }
  return dirs[dirs.length - 1];
}

function readJsonSafe(p) {
  try {
    return JSON.parse(readFileSync(p, 'utf8'));
  } catch {
    return null;
  }
}

function runCritique(rest) {
  const targets = requireTargets(rest, 'critique');
  const slug = slugify(targets);
  const ts = timestamp();
  const dir = resolve(OUT_DIR, ts);
  mkdirSync(dir, { recursive: true });

  // 评分脚手架（score=null，待 AI 按 §0.A 读完规范后独立填写；与 audit 互盲）
  const skeleton = {
    slug,
    target: targets.map((t) => relative(PROJECT_ROOT, t)),
    timestamp: ts,
    blind: true,
    dims: CRITIQUE_DIMENSIONS.map((d) => ({ key: d.key, score: null, note: '' })),
    total: null,
    max: 24,
    todos: [], // AI 视觉视角拿不准的项 → 进 design-todo（critique-only 信号）
  };
  writeFileSync(resolve(dir, 'critique.json'), JSON.stringify(skeleton, null, 2) + '\n', 'utf8');

  const md = [];
  md.push(`# critique-report —— ${slug}`);
  md.push('');
  md.push(`> 目标：${skeleton.target.join(', ')}`);
  md.push(`> 生成：${ts}　|　**互盲约束**：本报告须独立于 audit 结果填写（DESIGN §1.5.1）`);
  md.push('');
  md.push('## 0. 填写前必读（SKILL §0.A）');
  md.push('');
  md.push('- `clawpro-portable-design-skill/SKILL.md` §0 Scope / §2 Critical Rules / §8 Self-Audit');
  md.push('- 你工程的设计系统展示台（如有本地预览，对照看「正版」组件）');
  md.push('- 设计规范里对应的页面参照（`clawpro-portable-design-skill/assets/page-references/<page>.md`）');
  md.push('');
  md.push('## 1. 视觉视角评分（每项 0~4，总分 24）');
  md.push('');
  md.push('| 维度 | 评分 | 关注点 | 说明 / 证据 |');
  md.push('|---|---|---|---|');
  for (const d of CRITIQUE_DIMENSIONS) {
    md.push(`| ${d.key} | _ /4 | ${d.hint} |  |`);
  }
  md.push('');
  md.push('> 填完后把分数同步回 `critique.json` 的 `dims[].score`，再跑 `polish` 合并。');
  md.push('');
  md.push('## 2. critique-only 待裁决（AI 拿不准的视觉问题）');
  md.push('');
  md.push('> 这些是「只有视觉视角命中、静态规则抓不到」的项，按 §1.5.1 强制进 design-todo.csv，不进 audit-report。');
  md.push('> 在 `critique.json` 的 `todos[]` 里追加：{ conflictType, file, line, message, suggestion }');
  md.push('');
  md.push('## 3. 结论');
  md.push('');
  md.push('- critique 健康分：_ /24');
  md.push('- 下一步：`node scripts/walkthrough.mjs polish ' + (rest[0] || '<target>') + '`');
  md.push('');
  writeFileSync(resolve(dir, 'critique-report.md'), md.join('\n'), 'utf8');

  console.log('[walkthrough] critique 脚手架已生成（互盲，未自动评分）');
  console.log('  slug    :', slug);
  console.log('  out     :', relative(PROJECT_ROOT, dir));
  console.log('  下一步  : 按 §0.A 读规范 → 填 critique-report.md 分数 → 回填 critique.json → 跑 polish');
}

function runPolish(rest) {
  const targets = requireTargets(rest, 'polish');
  const slug = slugify(targets);
  const ts = timestamp();
  const dir = resolve(OUT_DIR, ts);

  const auditSnap = findLatestSnapshot('audit-report.json', slug);
  const critiqueSnap = findLatestSnapshot('critique.json', slug);
  if (!auditSnap) {
    console.error('[walkthrough] 找不到任何 audit 快照，请先跑 `audit`');
    process.exit(2);
  }

  const auditReport = readJsonSafe(resolve(auditSnap, 'audit-report.json')) || { findings: [] };
  const auditMeta = readJsonSafe(resolve(auditSnap, 'meta.json')) || {};
  const auditTodos = (readJsonSafe(resolve(auditSnap, 'design-todo.json')) || { todos: [] }).todos;
  const auditHealth = auditMeta.auditHealth || computeAuditHealth(auditReport.findings || []);

  let critique = critiqueSnap ? readJsonSafe(resolve(critiqueSnap, 'critique.json')) : null;
  let critiqueTotal = null;
  let critiqueComplete = false;
  if (critique && Array.isArray(critique.dims)) {
    const scored = critique.dims.filter((d) => typeof d.score === 'number');
    critiqueComplete = scored.length === critique.dims.length && critique.dims.length > 0;
    critiqueTotal = scored.reduce((a, d) => a + d.score, 0);
  }

  // 合并 todo：audit 的 todo 流 + critique-only 视觉项
  const critiqueTodos = (critique && Array.isArray(critique.todos) ? critique.todos : []).map((t) => ({
    conflictType: t.conflictType || 'critique-only（视觉视角）',
    ruleId: 'critique',
    line: t.line || 1,
    file: t.file || '',
    message: t.message || '',
    suggestion: t.suggestion || '',
  }));
  const mergedTodos = [...auditTodos, ...critiqueTodos];

  const total = critiqueComplete ? auditHealth.total + critiqueTotal : null;
  const grade = total != null ? gradeBand(total, 56) : '（critique 未评分，无法定级）';

  mkdirSync(dir, { recursive: true });
  writeFileSync(resolve(dir, 'design-todo.csv'), todosToCsv(mergedTodos), 'utf8');

  // polish-plan.md：P0/P1 收口清单
  const findings = auditReport.findings || [];
  const order = { P0: 0, P1: 1, P2: 2, P3: 3 };
  const actionable = findings
    .filter((f) => f.severity === 'P0' || f.severity === 'P1')
    .sort((a, b) => (order[a.severity] ?? 9) - (order[b.severity] ?? 9));

  const md = [];
  md.push(`# polish-plan —— ${slug}`);
  md.push('');
  md.push(`> 合并自 audit 快照 \`${relative(PROJECT_ROOT, auditSnap)}\``);
  md.push(`> ${critiqueSnap ? '+ critique 快照 `' + relative(PROJECT_ROOT, critiqueSnap) + '`' : '（未找到 critique 快照，仅 audit 单轨）'}`);
  md.push('');
  md.push('## 健康分');
  md.push('');
  md.push(`- audit：${auditHealth.total}/32`);
  md.push(`- critique：${critiqueComplete ? critiqueTotal + '/24' : '未评分（先跑 critique 并回填分数）'}`);
  md.push(`- **总分：${total != null ? total + '/56' : 'N/A'}　评级：${grade}**`);
  md.push('');
  md.push('### audit 维度明细');
  md.push('');
  md.push('| 维度 | 分 | 命中 | P0 |');
  md.push('|---|---|---|---|');
  for (const d of auditHealth.dims) {
    md.push(`| ${d.key} | ${d.score}/4 | ${d.count} | ${d.hasP0 ? '⚠️' : ''} |`);
  }
  md.push('');
  md.push(`## P0 / P1 收口清单（AI 直接修，共 ${actionable.length} 条）`);
  md.push('');
  if (actionable.length === 0) {
    md.push('_无 P0/P1，主体可用_');
  } else {
    md.push('| 严重度 | ruleId | 文件:行 | 问题 | 建议 |');
    md.push('|---|---|---|---|---|');
    for (const f of actionable) {
      const cell = (s) => String(s || '').replace(/\|/g, '\\|').replace(/\n/g, ' ');
      md.push(`| ${f.severity} | ${cell(f.ruleId)} | ${cell(f.file)}:${f.line} | ${cell(f.message)} | ${cell(f.suggestion)} |`);
    }
  }
  md.push('');
  md.push(`## 待用户裁决（design-todo.csv，共 ${mergedTodos.length} 条）`);
  md.push('');
  md.push('- audit 待裁决：' + auditTodos.length + ' 条');
  md.push('- critique-only：' + critiqueTodos.length + ' 条');
  md.push('- 见同目录 `design-todo.csv`；裁决后请在 `references/conflict-log.md` 留痕。');
  md.push('');
  writeFileSync(resolve(dir, 'polish-plan.md'), md.join('\n'), 'utf8');

  appendTrend(slug, {
    ts,
    audit: auditHealth.total,
    critique: critiqueComplete ? critiqueTotal : null,
    total,
    label: 'polish',
  });

  console.log('[walkthrough] polish done');
  console.log('  audit   :', auditHealth.total + '/32');
  console.log('  critique:', critiqueComplete ? critiqueTotal + '/24' : '未评分');
  console.log('  total   :', total != null ? total + '/56' : 'N/A', '|', grade);
  console.log('  P0/P1   :', actionable.length, '条');
  console.log('  todo    :', mergedTodos.length, '条 → design-todo.csv');
  console.log('  out     :', relative(PROJECT_ROOT, dir));
}

function runTrend(rest) {
  const slug = rest[0] || 'all';
  const p = resolve(OUT_DIR, 'snapshots', slug, 'trend.json');
  if (!existsSync(p)) {
    console.error(`[walkthrough] 找不到趋势文件：${relative(PROJECT_ROOT, p)}`);
    console.error('  先跑过 audit / polish 才会生成趋势。可用 slug：');
    const snapRoot = resolve(OUT_DIR, 'snapshots');
    if (existsSync(snapRoot)) {
      for (const n of readdirSync(snapRoot)) console.error('   -', n);
    }
    process.exit(1);
  }
  const arr = readJsonSafe(p) || [];
  const last5 = arr.slice(-5);
  const fmt = (e) => (e.total != null ? e.total : e.audit != null ? `${e.audit}(a)` : '?');
  console.log(`[walkthrough] ${slug} 最近 ${last5.length} 次：`);
  console.log('  ' + last5.map(fmt).join(' → '));
  if (last5.length >= 2) {
    const a = last5[0].total ?? last5[0].audit ?? 0;
    const b = last5[last5.length - 1].total ?? last5[last5.length - 1].audit ?? 0;
    console.log('  趋势：' + (b > a ? '↑ 好转' : b < a ? '↓ 退步' : '→ 持平'));
  }
}

// ---------- CLI ----------

function help() {
  console.log(`clawpro-walkthrough v0.8

audit 轨（机器静态规则 → audit-report.csv + design-todo.csv）：
  walkthrough.mjs audit <target...>          全部 detector
  walkthrough.mjs radius <target...>         只跑 radius   (P0)
  walkthrough.mjs color  <target...>         只跑 color    (P1)
  walkthrough.mjs icon-slot <target...>      只跑 icon-slot (P0)
  walkthrough.mjs shadow <target...>         只跑 shadow   (P1/P2)
  walkthrough.mjs component-drift <target>   只跑 component-drift (P2)
  walkthrough.mjs page-recipe-match <target> 只跑 page-recipe-match (P2)
  walkthrough.mjs text-color <target>        只跑 文字色 Typography (P2)
  walkthrough.mjs surface-nesting <target>   只跑 Surface 套娃 (P2)
  walkthrough.mjs spacing-grouping <target>  只跑 间距成组 (P2 → design-todo)
  walkthrough.mjs diff                       对 git diff 跑全部 detector

critique 轨（AI 视觉视角，与 audit 互盲 → critique-report.md）：
  walkthrough.mjs critique <target>          生成 critique 评分脚手架（6 维 × 0~4）

polish 轨（综合收口 → polish-plan.md + design-todo.csv + 总分）：
  walkthrough.mjs polish <target>            合并最近一次 audit + critique

其它：
  walkthrough.mjs trend [slug]               打印最近 5 次健康分趋势
  walkthrough.mjs explain <ruleId>           解释规则
  walkthrough.mjs refresh-fixtures [--only=<名>] [--json]
                                             一键重抽 fixtures（编排 extractors/ 6 个 extractor）
`);
}

function runRefreshFixtures(rest) {
  // 一键重抽 fixtures：编排 extractors/ 下的 6 个 extractor（各自写 fixtures/*.json）。
  // 落实 DESIGN.md 里规划的 `$walkthrough refresh-fixtures`，用于设计对齐 SOP 的 Stage 2。
  const onlyArg = rest.find((a) => a.startsWith('--only='));
  const only = onlyArg ? onlyArg.slice('--only='.length) : null;
  const jsonMode = rest.includes('--json');

  const extractorsDir = resolve(SKILL_ROOT, 'extractors');
  const fixturesDir = resolve(SKILL_ROOT, 'fixtures');
  if (!existsSync(extractorsDir)) {
    console.error(`[refresh-fixtures] 找不到 extractors 目录：${extractorsDir}`);
    process.exit(2);
  }
  let files = readdirSync(extractorsDir)
    .filter((f) => f.endsWith('.mjs') && f.startsWith('extract-'))
    .sort();
  if (only) files = files.filter((f) => f.includes(only));
  if (files.length === 0) {
    console.error(`[refresh-fixtures] 没有匹配的 extractor（--only=${only ?? ''}）。`);
    process.exit(2);
  }

  const results = [];
  for (const f of files) {
    const abs = resolve(extractorsDir, f);
    const t0 = Date.now();
    let ok = true;
    let out = '';
    let err = '';
    try {
      // execFile 数组传参、无 shell；extractor 路径来自内部目录扫描，无外部注入
      out = execFileSync('node', [abs], { encoding: 'utf8', cwd: PROJECT_ROOT });
    } catch (e) {
      ok = false;
      err = (e.stderr || e.message || String(e)).toString();
    }
    results.push({ extractor: f, ok, ms: Date.now() - t0, out: String(out).trim(), err: String(err).trim() });
  }

  const failed = results.filter((r) => !r.ok);

  if (jsonMode) {
    console.log(JSON.stringify({ extractorsDir, fixturesDir, results }, null, 2));
    process.exit(failed.length === 0 ? 0 : 1);
  }

  console.log(`[refresh-fixtures] 重抽 ${results.length} 个 fixtures（extractors/ → fixtures/）\n`);
  for (const r of results) {
    console.log(`${r.ok ? '✅' : '❌'} ${r.extractor}  (${r.ms}ms)`);
    const body = (r.ok ? r.out : r.err).split('\n').filter(Boolean);
    for (const line of body.slice(0, 6)) console.log(`     ${line}`);
    if (body.length > 6) console.log(`     … (${body.length - 6} more)`);
  }

  if (existsSync(fixturesDir)) {
    console.log(`\nfixtures/ 现状：`);
    for (const f of readdirSync(fixturesDir).filter((x) => x.endsWith('.json')).sort()) {
      const st = statSync(resolve(fixturesDir, f));
      console.log(`  ${f}  (${st.mtime.toISOString()})`);
    }
  }

  if (failed.length) {
    console.error(`\n❌ ${failed.length} 个 extractor 失败：${failed.map((r) => r.extractor).join(', ')}`);
    process.exit(1);
  }
  console.log(`\n✅ 全部 ${results.length} 个 fixtures 重抽完成。`);
  process.exit(0);
}

function main() {
  const [cmd, ...rest] = process.argv.slice(2);
  if (!cmd) return help();

  if (cmd === 'explain') {
    const id = rest[0];
    const def = EXPLAIN[id];
    if (!def) {
      console.error(`未知 ruleId: ${id}`);
      console.error('可用：' + Object.keys(EXPLAIN).join(', '));
      process.exit(1);
    }
    console.log(JSON.stringify({ ruleId: id, ...def }, null, 2));
    return;
  }

  if (cmd === 'critique') {
    return runCritique(rest);
  }
  if (cmd === 'polish') {
    return runPolish(rest);
  }
  if (cmd === 'trend') {
    return runTrend(rest);
  }
  if (cmd === 'refresh-fixtures') {
    return runRefreshFixtures(rest);
  }

  let rulesToRun;
  let targets;
  let label = cmd;
  let withExternal = false;

  if (cmd === 'audit') {
    rulesToRun = Object.keys(DETECTORS);
    targets = requireTargets(rest, 'audit');
    withExternal = true;
  } else if (cmd === 'diff') {
    rulesToRun = Object.keys(DETECTORS);
    targets = gitDiffFiles();
    if (targets.length === 0) {
      console.log('[walkthrough] git diff 为空，无需走查');
      return;
    }
    // diff 模式不跑外部 detector（老脚本是全量扫描，diff 场景下噪声太大）
    withExternal = false;
  } else if (DETECTORS[cmd]) {
    rulesToRun = [cmd];
    targets = requireTargets(rest, cmd);
  } else {
    console.error(`未知命令：${cmd}`);
    help();
    process.exit(1);
  }

  const result = audit(targets, rulesToRun, { withExternal });
  const { meta } = writeSnapshot(label, rest, rulesToRun, result, targets);
  printSummary(meta);

  const exitCode = computeExitCode(label, meta);
  if (exitCode !== 0) {
    const blockLevel = (process.env.WALKTHROUGH_BLOCK_LEVEL || 'P0').toUpperCase();
    console.error('');
    console.error(`[walkthrough] ❌ blocked: 命中 >= ${blockLevel} 级别 finding，CI 阻断`);
    console.error(`  调整阈值：环境变量 WALKTHROUGH_BLOCK_LEVEL=P0|P1|P2`);
    console.error(`  跳过阻断：WALKTHROUGH_BLOCK_LEVEL=NONE`);
    process.exit(exitCode);
  }
}

try {
  main();
} catch (e) {
  console.error('[walkthrough] 脚本异常：', e?.stack || e?.message || e);
  process.exit(2);
}

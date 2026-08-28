// external-design-skill detector wrapper
// ----------------------------------------------------------------------------
// 把 clawpro-portable-design-skill/scripts/ 下的老脚本作为外部 detector 接入走查。
// 这样设计 skill 既是"法律原文"（SKILL.md / specs / tokens），也复用其历史投资的检测脚本。
//
// 当前接入（2026-06-30 同步同事 miekoyychen MR !123 / !129 / !133 的治理产出）：
//   - check-design-usage.mjs  (8 个违规 check + 1 个 needs-design-confirmation 收集器)
//       违规 check（→ 右侧为脚本输出的 rule id，须与 SEVERITY_MAP / 健康分维度逐字一致）：
//         checkProhibitedLucideSlots   → prohibited-lucide-slot（9 槽位禁回退 lucide，MR !129 细化）
//         checkCssHardcodedColors      → css-hardcoded-color-hex / css-hardcoded-rgba-color
//         checkSemanticTailwindColors  → semantic-tailwind-color
//         checkCssVariableLargeRadius  → css-large-radius（管控端 4px 铁律）
//         checkInlineSvgMarking        → inline-svg-without-marker
//         checkIconRegistryConsistency → icon-registry-broken
//         checkMultiVariantSync        → multi-variant-inconsistency
//       confirmation 收集器：
//         collectDesignConfirmationMarkers — 收集 needs-design-confirmation（走 confirmations → design-todo）
//   - check-spec-symbols.mjs  (spec 引用了 client/src 不存在的 identifier，"ghost reference")
//
// 接入方式：
//   - 不走 per-file 回调（_shared.mjs 的 scanRegex），而是**一次性**调用外部脚本扫描整个 client/src，
//     把外部脚本的 JSON 输出映射成统一的 finding[]。
//   - walkthrough.mjs 必须以"批处理"方式调用此 detector：只调一次 runBatch()，而不是 per-file。
//
// 冲突铁律（落实设计 skill §1.5 / §12）：
//   - `needs-design-confirmation` 标记 = 用户已知晓，正在等裁决 → 走查跳过，不视作违规。
//   - 第 8 个 check `collectDesignConfirmationMarkers` 的输出走 `confirmations`（非 `violations`），
//     wrapper 不映射成 finding（避免对用户"已经在处理"的事重复报警）。
//   - 后续 v1.0 可改为：把 confirmations 注入 design-todo.csv 让走查也能看到"还在等谁裁决"。
//
// 失败策略：
//   - 外部脚本不存在 / 报错 / 输出非 JSON → 抛 1 条 P2 finding（"外部检测器调用失败"），不阻断整体 audit。
//   - 用环境变量 WALKTHROUGH_SKIP_EXTERNAL=1 可整体跳过（CI 紧急时降级）。
//
// severity 映射（参考 SKILL.md §1.5.4）：
//   - prohibited-lucide-slot      → P0  (icon-slot 铁律命中，MR !129 已细化)
//   - css-hardcoded-color-hex     → P1
//   - css-hardcoded-rgba-color    → P1
//   - semantic-tailwind-color     → P1
//   - css-large-radius            → P0  (管控端 4px 铁律)
//   - inline-svg-without-marker   → P2
//   - icon-registry-broken        → P2
//   - multi-variant-inconsistency → P2
//   - check-spec-symbols ghost    → P2
//   - 其他未知 rule               → P2
// ----------------------------------------------------------------------------

import { execFileSync } from 'node:child_process';
import {
  existsSync,
  mkdtempSync,
  openSync,
  closeSync,
  readFileSync,
  rmSync,
} from 'node:fs';
import { resolve, dirname, relative, join } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { makeFinding } from './_shared.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SKILLS_ROOT = resolve(__dirname, '../../..'); // .codebuddy/skills
// 仓库根：默认按标准安装位置反推，可用 WALKTHROUGH_PROJECT_ROOT 覆盖（与 walkthrough.mjs 一致）。
const PROJECT_ROOT = process.env.WALKTHROUGH_PROJECT_ROOT
  ? resolve(process.env.WALKTHROUGH_PROJECT_ROOT)
  : resolve(SKILLS_ROOT, '../..');
// 配套设计 skill（"法律原文" + 复用其检测脚本）：默认同级 clawpro-portable-design-skill，
// 装到别处/改名时用 WALKTHROUGH_DESIGN_SKILL_DIR 指向其根目录。
// 未安装该 skill 时，下方 existsSync 判空 → external/* detector 静默跳过，不阻断主流程。
const DESIGN_SKILL_DIR = process.env.WALKTHROUGH_DESIGN_SKILL_DIR
  ? resolve(process.env.WALKTHROUGH_DESIGN_SKILL_DIR)
  : resolve(SKILLS_ROOT, 'clawpro-portable-design-skill');
const DESIGN_SKILL_SCRIPTS = resolve(DESIGN_SKILL_DIR, 'scripts');

const CHECK_DESIGN_USAGE = resolve(DESIGN_SKILL_SCRIPTS, 'check-design-usage.mjs');
const CHECK_SPEC_SYMBOLS = resolve(DESIGN_SKILL_SCRIPTS, 'check-spec-symbols.mjs');

const SEVERITY_MAP = {
  'prohibited-lucide-slot': 'P0',
  'css-large-radius': 'P0',
  'css-hardcoded-color-hex': 'P1',
  'css-hardcoded-rgba-color': 'P1',
  'semantic-tailwind-color': 'P1',
  'inline-svg-marking': 'P2',
  'icon-registry-inconsistent': 'P2',
  'multi-variant-sync': 'P2',
};

// ⚠️ 为什么把 stdout 重定向到临时文件而不是用 pipe：
//   check-design-usage.mjs 在 --json 模式写完大块 JSON（~1MB）后立即 process.exit(0)。
//   当 stdout 是「管道」（execFileSync 默认 pipe）时，Node 的 process.exit() 不等待管道 drain，
//   会截断大输出 → wrapper 收到不完整 JSON → JSON.parse 失败。
//   把 stdout 指向「普通文件」fd 时，Node 用同步写，process.exit 前已落盘，规避截断。
//   （根因在外部脚本；为不动同事的治理脚本，这里在 wrapper 侧绕过。）
function runExternal(scriptPath, args, cwd) {
  const tmpDir = mkdtempSync(join(tmpdir(), 'cp-walkthrough-'));
  const outFile = join(tmpDir, 'out.json');
  let fd;
  try {
    fd = openSync(outFile, 'w');
    try {
      execFileSync('node', [scriptPath, ...args], {
        cwd,
        stdio: ['ignore', fd, 'pipe'],
        maxBuffer: 64 * 1024 * 1024,
      });
    } catch (e) {
      // 非零退出（如 strict 命中违规）也算正常——文件里已有完整输出
      void e;
    } finally {
      closeSync(fd);
      fd = undefined;
    }
    const stdout = existsSync(outFile) ? readFileSync(outFile, 'utf8') : '';
    if (stdout && stdout.trim().startsWith('{')) {
      return { ok: true, stdout };
    }
    return { ok: false, error: '外部脚本未输出有效 JSON', stdout };
  } catch (e) {
    return { ok: false, error: e?.message || String(e), stdout: '' };
  } finally {
    if (fd !== undefined) {
      try {
        closeSync(fd);
      } catch {
        /* noop */
      }
    }
    try {
      rmSync(tmpDir, { recursive: true, force: true });
    } catch {
      /* noop */
    }
  }
}

/**
 * 调用 check-design-usage.mjs --json 一次，把违规映射成 finding[]
 */
function runCheckDesignUsage(targets) {
  if (!existsSync(CHECK_DESIGN_USAGE)) return [];
  const args = ['--json', ...targets];
  const res = runExternal(CHECK_DESIGN_USAGE, args, PROJECT_ROOT);
  if (!res.ok) {
    return [
      makeFinding({
        ruleId: 'external-design-skill',
        severity: 'P2',
        file: relative(PROJECT_ROOT, CHECK_DESIGN_USAGE),
        line: 1,
        col: 1,
        snippet: '',
        message: `check-design-usage.mjs 调用失败：${res.error}`,
        evidence:
          'clawpro-portable-design-skill/scripts/check-design-usage.mjs',
        suggestion: '手动跑一遍该脚本排查',
      }),
    ];
  }
  let payload;
  try {
    payload = JSON.parse(res.stdout);
  } catch (e) {
    return [
      makeFinding({
        ruleId: 'external-design-skill',
        severity: 'P2',
        file: relative(PROJECT_ROOT, CHECK_DESIGN_USAGE),
        line: 1,
        col: 1,
        snippet: res.stdout.slice(0, 60),
        message: 'check-design-usage.mjs --json 输出无法解析为 JSON',
        evidence:
          'clawpro-portable-design-skill/scripts/check-design-usage.mjs',
        suggestion: '检查脚本是否打印了非 JSON 日志到 stdout',
      }),
    ];
  }
  const findings = [];
  for (const v of payload.violations || []) {
    findings.push(
      makeFinding({
        ruleId: `external/${v.rule}`,
        severity: SEVERITY_MAP[v.rule] || 'P2',
        file: v.file,
        line: v.line,
        col: 1,
        snippet: v.source || '',
        message: `${v.checkName}: ${v.message}`,
        evidence: `clawpro-portable-design-skill/scripts/check-design-usage.mjs#${v.rule}`,
        suggestion: '参考 clawpro-portable-design-skill SKILL.md §2 对应章节',
      }),
    );
  }
  // needs-design-confirmation 标记 → 不抛违规，进 design-todo.csv 待裁决（SKILL §0.C / §1.5）
  for (const c of payload.confirmations || []) {
    findings.push(
      makeFinding({
        ruleId: 'external/needs-design-confirmation',
        severity: 'P1',
        stream: 'todo',
        conflictType: inferConflictType(c.details || ''),
        file: c.file,
        line: c.line || 1,
        col: 1,
        snippet: c.source || c.details || '',
        message: `代码已标记 needs-design-confirmation：${c.details || ''}`,
        evidence:
          'clawpro-portable-design-skill/scripts/check-design-usage.mjs#collectDesignConfirmationMarkers',
        suggestion:
          '请用户裁决；裁决后在 references/conflict-log.md 追加条目并移除标记',
      }),
    );
  }
  return findings;
}

/**
 * 从 needs-design-confirmation 的 details 文本里粗判冲突类型（设计 skill §12.3 五类）。
 * 判不准时返回"待分类"，由用户在 design-todo.csv 里订正。
 */
function inferConflictType(details) {
  const t = String(details).toLowerCase();
  if (/icon|图标|槽位|svg|lucide/.test(t)) return '图标无候选';
  if (/token|颜色|color|color|阴影|shadow|圆角|radius/.test(t)) return 'Token 模糊';
  if (/宿主|fallback|host|兼容|portable/.test(t)) return '宿主仓兼容';
  if (/需求|超范围|业务|scope/.test(t)) return '需求超范围';
  if (/spec|规范|现状|drift|漂离/.test(t)) return 'Spec vs 现状';
  return '待分类';
}

/**
 * 调用 check-spec-symbols.mjs --json 一次（不接收 targets，它扫的是 spec 文档本身）
 */
function runCheckSpecSymbols() {
  if (!existsSync(CHECK_SPEC_SYMBOLS)) return [];
  const res = runExternal(CHECK_SPEC_SYMBOLS, ['--json'], PROJECT_ROOT);
  if (!res.ok) {
    return [
      makeFinding({
        ruleId: 'external-design-skill',
        severity: 'P2',
        file: relative(PROJECT_ROOT, CHECK_SPEC_SYMBOLS),
        line: 1,
        col: 1,
        snippet: '',
        message: `check-spec-symbols.mjs 调用失败：${res.error}`,
        evidence:
          'clawpro-portable-design-skill/scripts/check-spec-symbols.mjs',
        suggestion: '手动跑一遍该脚本排查',
      }),
    ];
  }
  let payload;
  try {
    payload = JSON.parse(res.stdout);
  } catch {
    return [];
  }
  const findings = [];
  for (const g of payload.ghosts || []) {
    findings.push(
      makeFinding({
        ruleId: 'external/spec-symbol-ghost',
        severity: 'P2',
        file: relative(PROJECT_ROOT, resolve(PROJECT_ROOT, g.spec)) || g.spec,
        line: g.lineHint || 1,
        col: 1,
        snippet: `import { ${g.identifier} } from "${g.fromPath}"`,
        message: `spec 引用了 client/src 不存在的 identifier: ${g.identifier}`,
        evidence:
          'clawpro-portable-design-skill/scripts/check-spec-symbols.mjs',
        suggestion:
          '改名 / 删除该 import / 标记为反例（详见脚本输出的"处理建议"）',
      }),
    );
  }
  return findings;
}

/**
 * 批量入口：walkthrough.mjs 在 audit 模式下调一次。
 * targets：要扫描的目录或文件相对路径数组（相对于 PROJECT_ROOT），传给 check-design-usage。
 */
export function runBatch(targets) {
  if (process.env.WALKTHROUGH_SKIP_EXTERNAL === '1') return [];
  const relTargets = (targets || []).map((p) => relative(PROJECT_ROOT, p));
  const findings = [];
  findings.push(...runCheckDesignUsage(relTargets));
  findings.push(...runCheckSpecSymbols());
  return findings;
}

// 兼容 per-file 调用：什么都不做（避免 walkthrough.mjs 误把它当 per-file detector 跑）
export function run() {
  return [];
}

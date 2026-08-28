#!/usr/bin/env node
/**
 * 扫描 openclaw-enterprise 仓库，根据「v2 设计系统」改造信号自动产出真实改造进度。
 *
 * 输出文件：组件引用及更新/progress-auto.json
 *
 * 判定规则（保守 + 可解释）：
 *
 * 【组件层】（client/src/components/ui/<name>.tsx）
 *   命中以下任一 v2 信号 → done（已替换）：
 *     - rounded-[4px]              (v2 圆角令牌)
 *     - var(--shadow-card|overlay|config|inner|segment)   (v2 阴影令牌)
 *     - var(--color-gray-(50|100|200|300|400|500|600|700|900|950))
 *     - #1447E6                    (v2 主色)
 *     - 引用了 Surface
 *   未命中 → todo
 *
 *   说明：项目里的 ui 组件如果连 rounded-[4px] 都没用，基本就是 shadcn 原版未改。
 *
 * 【页面层】（client/src/pages/**\/<name>.tsx）
 *   按命中信号「数量」分档（更细颗粒度，反映"改了一半"的状态）：
 *     - 引用 Surface*    → +3 分（最强信号：已切到 v2 卡片体系）
 *     - rounded-[4px]    → +1 分（每出现 1 处计 1 分，最多 3 分）
 *     - var(--shadow-*)  → +2 分
 *     - var(--color-gray-* 或 #1447E6 → +1 分（最多 2 分）
 *
 *   分数判定：
 *     - score >= 3 → done
 *     - score >= 1 → doing
 *     - 否则       → todo
 *
 * 用法：
 *   cd 组件引用及更新
 *   node scan-progress.cjs
 *
 * 也可以加 `--repo /path/to/openclaw-enterprise` 自定义仓库路径。
 */

const fs = require("fs");
const path = require("path");

// ───────── 解析参数 ─────────
const args = process.argv.slice(2);
let repoArg = null;
for (let i = 0; i < args.length; i++) {
  if (args[i] === "--repo" && args[i + 1]) repoArg = args[i + 1];
}

const SCRIPT_DIR = __dirname;
// 现在脚本住在 openclaw-enterprise/docs/component-refs/，仓库根 = ../..
const DEFAULT_REPO = path.resolve(SCRIPT_DIR, "..", "..");
const REPO = repoArg ? path.resolve(repoArg) : DEFAULT_REPO;

if (!fs.existsSync(REPO)) {
  console.error("❌ 找不到仓库目录：" + REPO);
  process.exit(1);
}

const UI_DIR = path.join(REPO, "client/src/components/ui");
const PAGES_DIR = path.join(REPO, "client/src/pages");

// 与 component-page-refs.json 一致的组件名清单
const COMP_NAMES = [
  "accordion","alert","alert-dialog","aspect-ratio","avatar","badge","breadcrumb",
  "button","button-group","calendar","card","carousel","chart","checkbox","collapsible",
  "command","context-menu","dialog","drawer","dropdown-menu","empty","field","form",
  "hover-card","input","input-group","input-otp","item","kbd","label","menubar",
  "navigation-menu","pagination","popover","progress","radio-group","resizable",
  "scroll-area","select","separator","sheet","sidebar","skeleton","slider","sonner",
  "spinner","switch","table","tabs","textarea","toggle","toggle-group","tooltip",
];

// ───────── 信号正则 ─────────
const RE_ROUND_4 = /rounded-\[4px\]/g;
const RE_SHADOW_TOKEN = /var\(--shadow-(card|overlay|config|inner|segment)\)/g;
const RE_GRAY_TOKEN = /var\(--color-gray-(?:50|100|200|300|400|500|600|700|900|950)\)|#1447E6/g;
const RE_SURFACE = /Surface(Card|Inner|Overlay|Config)\b/g;

function readSafe(p) {
  try { return fs.readFileSync(p, "utf8"); } catch (e) { return ""; }
}
function countMatches(re, text) {
  re.lastIndex = 0;
  let m, n = 0;
  while ((m = re.exec(text))) n++;
  return n;
}

// ───────── 扫描组件层 ─────────
const components = {};
for (const name of COMP_NAMES) {
  const file = path.join(UI_DIR, name + ".tsx");
  if (!fs.existsSync(file)) {
    components[name] = { status: "todo", signals: { fileExists: false }, updatedAt: Date.now() };
    continue;
  }
  const text = readSafe(file);
  const sig = {
    fileExists: true,
    rounded4: countMatches(RE_ROUND_4, text),
    shadowToken: countMatches(RE_SHADOW_TOKEN, text),
    grayToken: countMatches(RE_GRAY_TOKEN, text),
    surface: countMatches(RE_SURFACE, text),
  };
  const hit = sig.rounded4 + sig.shadowToken + sig.grayToken + sig.surface;
  components[name] = {
    status: hit > 0 ? "done" : "todo",
    signals: sig,
    updatedAt: Date.now(),
  };
}

// ───────── 扫描页面层 ─────────
function walk(dir, baseLen) {
  const out = [];
  if (!fs.existsSync(dir)) return out;
  for (const name of fs.readdirSync(dir)) {
    const full = path.join(dir, name);
    const stat = fs.statSync(full);
    if (stat.isDirectory()) out.push(...walk(full, baseLen));
    else if (stat.isFile() && full.endsWith(".tsx")) out.push(full);
  }
  return out;
}

const pageFiles = walk(PAGES_DIR, PAGES_DIR.length);
const pages = {};
for (const file of pageFiles) {
  const rel = path.relative(PAGES_DIR, file).replace(/\\/g, "/"); // 例如 admin/MemberManagement.tsx
  const text = readSafe(file);
  const sig = {
    surface: countMatches(RE_SURFACE, text),
    rounded4: countMatches(RE_ROUND_4, text),
    shadowToken: countMatches(RE_SHADOW_TOKEN, text),
    grayToken: countMatches(RE_GRAY_TOKEN, text),
  };
  // 评分
  const score =
    Math.min(sig.surface, 1) * 3 +
    Math.min(sig.rounded4, 3) * 1 +
    Math.min(sig.shadowToken, 2) * 2 +
    Math.min(sig.grayToken, 2) * 1;
  let status = "todo";
  if (score >= 3) status = "done";
  else if (score >= 1) status = "doing";
  pages[rel] = { status, score, signals: sig, updatedAt: Date.now() };
}

// ───────── 汇总输出 ─────────
const compDone = Object.values(components).filter(c => c.status === "done").length;
const compTodo = Object.values(components).filter(c => c.status === "todo").length;
const pageDone = Object.values(pages).filter(p => p.status === "done").length;
const pageDoing = Object.values(pages).filter(p => p.status === "doing").length;
const pageTodo = Object.values(pages).filter(p => p.status === "todo").length;

const payload = {
  generator: "scan-progress.cjs",
  generatedAt: new Date().toISOString(),
  repo: path.relative(SCRIPT_DIR, REPO),
  rules: {
    component: "命中 rounded-[4px] / var(--shadow-*) / var(--color-gray-*) / #1447E6 / Surface 任一 → done",
    page: "Surface×3 + rounded-[4px]×1（≤3）+ shadow-token×2（≤2）+ gray-token×1（≤2）；分≥3 done / ≥1 doing / 否则 todo",
  },
  summary: {
    components: { total: COMP_NAMES.length, done: compDone, todo: compTodo },
    pages: { total: Object.keys(pages).length, done: pageDone, doing: pageDoing, todo: pageTodo },
  },
  components,
  pages,
};

const out = path.join(SCRIPT_DIR, "progress-auto.json");
fs.writeFileSync(out, JSON.stringify(payload, null, 2), "utf8");

console.log("✅ 已生成：" + out);
console.log("");
console.log("📊 改造进度（自动扫描）");
console.log("──────────────────────────────────");
console.log(`组件 ${compDone}/${COMP_NAMES.length} 已替换 · ${compTodo} 待改`);
console.log(`页面 ${pageDone}/${Object.keys(pages).length} 已替换 · ${pageDoing} 进行中 · ${pageTodo} 待改`);
console.log("");
console.log("已改造的组件：");
console.log(
  Object.entries(components)
    .filter(([, v]) => v.status === "done")
    .map(([k]) => "  ✓ " + k)
    .join("\n")
);
console.log("");
console.log("还没改的组件：");
console.log(
  Object.entries(components)
    .filter(([, v]) => v.status === "todo")
    .map(([k]) => "  ○ " + k)
    .join("\n")
);

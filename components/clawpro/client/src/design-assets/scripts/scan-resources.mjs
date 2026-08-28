#!/usr/bin/env node
/**
 * ClawPro 资源库 - 阶段 2 全量资源扫描与使用位置统计
 *
 * 定位（严格遵守建设计划 §9 阶段 2）：
 *   - 本脚本只产「事实数据基线」：项目里有哪些资源、在哪里被引用、被引用多少次。
 *   - 不做分类判断（usageScenarios / visualType / primaryCategory / scenes / 语义 type / 重复判定 / canonical）。
 *     这些字段一律留空并标记 classification.pending = true，交由阶段 3+ 据实回填。
 *   - 只读：不修改、不删除、不移动任何资源文件；不动 skill。
 *   - 可重复执行：同样的仓库状态产出同样的结果（数组按稳定 key 排序）。
 *
 * 扫描口径来源：建设计划 §9 阶段 1「资源审查准备口径」。
 *
 * 产物：
 *   client/src/design-assets/generated/resource-inventory.generated.json
 *   client/src/design-assets/generated/resource-usage.generated.json
 *
 * 运行（在仓库根目录）：
 *   node client/src/design-assets/scripts/scan-resources.mjs
 */

import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
// scripts/ -> design-assets/ -> src/ -> client/ -> repoRoot
const repoRoot = path.resolve(__dirname, "..", "..", "..", "..");
const outDir = path.resolve(repoRoot, "client/src/design-assets/generated");

// ----------------------------------------------------------------------------
// 扫描口径（与建设计划 §9 阶段 1 一致）
// ----------------------------------------------------------------------------

const RESOURCE_EXTS = new Set([".svg", ".png"]);

// 文件型资源扫描目录（相对仓库根）。scope: include = 纳入资源库展示候选；scan-only = 仅扫描识别副本/污染。
const FILE_SCAN_DIRS = [
  { dir: "client/public/assets", source: "public", scope: "include" },
  { dir: "client/public/icon", source: "public", scope: "include" },
  { dir: "client/src/assets", source: "src", scope: "include" },
  { dir: "icon", source: "root-assets", scope: "include" }, // 根 icon/：registry 已登记业务 SVG 事实源
  { dir: "assets", source: "root-assets", scope: "scan-only" }, // 根 assets/：Figma 导出，仅扫描不默认纳入
];

// 代码引用扫描根（相对仓库根）。
const CODE_SCAN_ROOTS = ["client/src"];
const CODE_TEXT_EXTS = new Set([".ts", ".tsx", ".js", ".jsx", ".css"]);
// 代码扫描时跳过的目录 basename。
const CODE_IGNORE_DIRS = new Set([
  "node_modules",
  ".git",
  "dist",
  "build",
  "coverage",
  ".codebuddy",
  "generated", // 不扫描本脚本自身产物
]);

// ----------------------------------------------------------------------------
// 工具
// ----------------------------------------------------------------------------

function toRepoRel(absPath) {
  return path.relative(repoRoot, absPath).split(path.sep).join("/");
}

function slugify(value) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function shortHash(value) {
  return crypto.createHash("sha1").update(value).digest("hex").slice(0, 8);
}

function walkFiles(absRoot, files = []) {
  if (!fs.existsSync(absRoot)) return files;
  const stat = fs.statSync(absRoot);
  if (stat.isDirectory()) {
    for (const child of fs.readdirSync(absRoot).sort()) {
      walkFiles(path.join(absRoot, child), files);
    }
    return files;
  }
  files.push(absRoot);
  return files;
}

function walkCodeFiles(absRoot, files = []) {
  if (!fs.existsSync(absRoot)) return files;
  const stat = fs.statSync(absRoot);
  if (stat.isDirectory()) {
    if (CODE_IGNORE_DIRS.has(path.basename(absRoot))) return files;
    for (const child of fs.readdirSync(absRoot).sort()) {
      walkCodeFiles(path.join(absRoot, child), files);
    }
    return files;
  }
  if (CODE_TEXT_EXTS.has(path.extname(absRoot))) files.push(absRoot);
  return files;
}

function lineNumberAt(content, index) {
  let line = 1;
  for (let i = 0; i < index && i < content.length; i++) {
    if (content[i] === "\n") line++;
  }
  return line;
}

// SVG viewBox / 尺寸（仅记录事实，不做判定）
function parseSvgMeta(content) {
  const meta = {};
  const viewBox = content.match(/viewBox\s*=\s*["']([^"']+)["']/i);
  if (viewBox) meta.viewBox = viewBox[1].trim();
  const w = content.match(/\bwidth\s*=\s*["']([^"']+)["']/i);
  const h = content.match(/\bheight\s*=\s*["']([^"']+)["']/i);
  if (w) meta.width = w[1].trim();
  if (h) meta.height = h[1].trim();
  return meta;
}

// ----------------------------------------------------------------------------
// 1) 文件型资源清单
// ----------------------------------------------------------------------------

const inventory = []; // 资源条目
const usageMap = new Map(); // resourceId -> [{ file, line, kind }]
// 通过物理绝对路径解析引用
const pathToId = new Map(); // 归一化绝对路径 -> resourceId

function addUsage(resourceId, entry) {
  if (!usageMap.has(resourceId)) usageMap.set(resourceId, []);
  usageMap.get(resourceId).push(entry);
}

function classificationPlaceholder() {
  // 阶段 2 不做分类判断，留空交阶段 3+ 回填。
  return {
    pending: true,
    primaryCategory: null,
    usageScenarios: [],
    visualType: null,
    scenes: [],
    tags: [],
    status: null,
    duplicateGroupId: null,
    canonicalId: null,
    canRecolor: null,
    canUseInSkillMap: null,
  };
}

for (const { dir, source, scope } of FILE_SCAN_DIRS) {
  const absDir = path.resolve(repoRoot, dir);
  const files = walkFiles(absDir);
  for (const abs of files) {
    const ext = path.extname(abs).toLowerCase();
    if (!RESOURCE_EXTS.has(ext)) continue;
    const repoRel = toRepoRel(abs);
    const stat = fs.statSync(abs);
    const fileName = path.basename(abs);
    const type = ext === ".svg" ? "svg" : "png";
    const id = `${source}__${slugify(path.relative(absDir, abs).replace(/\.[^.]+$/, ""))}__${shortHash(repoRel)}`;

    // 服务路径 / import 路径（事实，便于阶段 7 页面消费）
    let webPath = null;
    let importPath = null;
    if (source === "public") {
      // client/public/xxx -> 浏览器 /xxx
      webPath = "/" + repoRel.replace(/^client\/public\//, "");
    } else if (source === "src") {
      importPath = "@/" + repoRel.replace(/^client\/src\//, "");
    }

    const item = {
      id,
      displayName: fileName.replace(/\.[^.]+$/, ""),
      fileName,
      type,
      source,
      scanScope: scope,
      sourcePath: repoRel,
      sourceDir: dir,
      webPath,
      importPath,
      bytes: stat.size,
      usageCount: 0,
      usageRefs: 0,
      classification: classificationPlaceholder(),
    };

    if (type === "svg") {
      try {
        item.svgMeta = parseSvgMeta(fs.readFileSync(abs, "utf8"));
      } catch {
        item.svgMeta = {};
      }
    }

    inventory.push(item);
    pathToId.set(path.resolve(abs), id);
  }
}

// ----------------------------------------------------------------------------
// 2) 代码引用扫描：lucide import / 文件资源引用 / inline svg / css url
// ----------------------------------------------------------------------------

const codeFiles = CODE_SCAN_ROOTS.flatMap((root) =>
  walkCodeFiles(path.resolve(repoRoot, root))
);

// lucide：importName -> { id, files:Set, locations:[] }
const lucideMap = new Map();
// inline svg 资源（每个 <svg 定义一个条目）
const inlineSvgItems = [];
// 无法解析到清单的资源引用（事实记录，便于人工排查）
const unresolvedRefs = [];
// 目录级字面量引用：absDir -> [{ file, line, ref }]
// 用于捕获 `const BASE = "/assets/admin-sidebar"` + `${BASE}/x.svg` 这类动态拼接，
// 避免据此把目标文件误判为「未引用」。
const dirRefMap = new Map();

const LUCIDE_IMPORT_RE =
  /import\s+(?:type\s+)?\{([^}]*)\}\s*from\s*['"]lucide-react['"]/g;
const ASSET_STRING_RE = /['"`]([^'"`\n]*?\.(?:svg|png))['"`]/g;
const CSS_URL_RE = /url\(\s*['"]?([^'")]+\.(?:svg|png))['"]?\s*\)/gi;
const INLINE_SVG_RE = /<svg[\s>]/g;
// 资源目录字面量（不含扩展名的资源路径片段，如 "/assets/admin-sidebar"、"@/assets/icons"）
const DIR_REF_RE = /['"`]((?:@\/assets|\/assets|\/icon)\/[^'"`\n$]*?)['"`]/g;

function recordDirRef(rawRef, fromFileAbs, repoRel, line) {
  const abs = resolveAssetRef(rawRef, fromFileAbs);
  if (!abs) return;
  let stat;
  try {
    stat = fs.statSync(abs);
  } catch {
    return;
  }
  if (!stat.isDirectory()) return; // 只关心目录级引用
  if (!dirRefMap.has(abs)) dirRefMap.set(abs, []);
  dirRefMap.get(abs).push({ file: repoRel, line, ref: rawRef });
}

function resolveAssetRef(rawRef, fromFileAbs) {
  let ref = rawRef.trim();
  if (!ref || ref.startsWith("http://") || ref.startsWith("https://") || ref.startsWith("data:")) {
    return null;
  }
  // 去掉查询串 / hash
  ref = ref.replace(/[?#].*$/, "");

  let candidateAbs = null;
  if (ref.startsWith("@/")) {
    candidateAbs = path.resolve(repoRoot, "client/src", ref.slice(2));
  } else if (ref.startsWith("@assets/")) {
    candidateAbs = path.resolve(repoRoot, "attached_assets", ref.slice("@assets/".length));
  } else if (ref.startsWith("/")) {
    // Vite root = client，故 /xxx -> client/public/xxx
    candidateAbs = path.resolve(repoRoot, "client/public", ref.slice(1));
  } else if (ref.startsWith("./") || ref.startsWith("../")) {
    candidateAbs = path.resolve(path.dirname(fromFileAbs), ref);
  } else {
    // 裸路径：先试 public，再试根目录
    const tryPublic = path.resolve(repoRoot, "client/public", ref);
    const tryRoot = path.resolve(repoRoot, ref);
    candidateAbs = fs.existsSync(tryPublic) ? tryPublic : tryRoot;
  }
  return candidateAbs ? path.resolve(candidateAbs) : null;
}

for (const fileAbs of codeFiles) {
  const repoRel = toRepoRel(fileAbs);
  const ext = path.extname(fileAbs).toLowerCase();
  let content;
  try {
    content = fs.readFileSync(fileAbs, "utf8");
  } catch {
    continue;
  }

  // 2a) lucide-react import
  for (const m of content.matchAll(LUCIDE_IMPORT_RE)) {
    const line = lineNumberAt(content, m.index);
    const names = m[1]
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean)
      .map((s) => s.replace(/^type\s+/, "").split(/\s+as\s+/)[0].trim())
      .filter((n) => /^[A-Za-z][A-Za-z0-9]*$/.test(n));
    for (const name of names) {
      if (!lucideMap.has(name)) {
        lucideMap.set(name, {
          id: `lucide__${slugify(name)}`,
          importName: name,
          files: new Set(),
          locations: [],
        });
      }
      const entry = lucideMap.get(name);
      entry.files.add(repoRel);
      entry.locations.push({ file: repoRel, line, kind: "import-lucide" });
    }
  }

  // 2b) 文件资源字符串引用（<img src>、字符串路径、Vite import 等统一按字符串路径捕获）
  if (ext !== ".css") {
    for (const m of content.matchAll(ASSET_STRING_RE)) {
      const raw = m[1];
      const line = lineNumberAt(content, m.index);
      const abs = resolveAssetRef(raw, fileAbs);
      const kind = raw.startsWith("@/") || raw.startsWith("@assets/")
        ? "vite-import"
        : "string-ref";
      if (abs && pathToId.has(abs)) {
        addUsage(pathToId.get(abs), { file: repoRel, line, kind });
      } else if (abs && fs.existsSync(abs) && RESOURCE_EXTS.has(path.extname(abs).toLowerCase())) {
        // 指向真实文件但不在纳入清单（如排除目录），记为未解析事实
        unresolvedRefs.push({ ref: raw, resolved: toRepoRel(abs), file: repoRel, line, kind, reason: "out-of-inventory" });
      } else {
        unresolvedRefs.push({ ref: raw, resolved: abs ? toRepoRel(abs) : null, file: repoRel, line, kind, reason: "not-found" });
      }
    }
  }

  // 2c) css url(...)
  if (ext === ".css") {
    for (const m of content.matchAll(CSS_URL_RE)) {
      const raw = m[1];
      const line = lineNumberAt(content, m.index);
      const abs = resolveAssetRef(raw, fileAbs);
      if (abs && pathToId.has(abs)) {
        addUsage(pathToId.get(abs), { file: repoRel, line, kind: "css-url" });
      } else if (abs && fs.existsSync(abs)) {
        unresolvedRefs.push({ ref: raw, resolved: toRepoRel(abs), file: repoRel, line, kind: "css-url", reason: "out-of-inventory" });
      } else {
        unresolvedRefs.push({ ref: raw, resolved: abs ? toRepoRel(abs) : null, file: repoRel, line, kind: "css-url", reason: "not-found" });
      }
    }
  }

  // 2c.5) 资源目录字面量引用（捕获动态基路径，如 const BASE = "/assets/admin-sidebar"）
  for (const m of content.matchAll(DIR_REF_RE)) {
    recordDirRef(m[1], fileAbs, repoRel, lineNumberAt(content, m.index));
  }

  // 2d) inline svg（仅 tsx/jsx）
  if (ext === ".tsx" || ext === ".jsx") {
    let count = 0;
    for (const m of content.matchAll(INLINE_SVG_RE)) {
      count++;
      const line = lineNumberAt(content, m.index);
      const id = `inline-svg__${slugify(repoRel)}__${count}`;
      const item = {
        id,
        displayName: `${path.basename(fileAbs)} inline-svg #${count}`,
        fileName: null,
        type: "inline-svg",
        source: "inline-svg",
        scanScope: "include",
        sourcePath: null,
        sourceDir: null,
        definedIn: repoRel,
        definedAtLine: line,
        webPath: null,
        importPath: null,
        bytes: null,
        usageCount: 1,
        usageRefs: 1,
        classification: classificationPlaceholder(),
      };
      inlineSvgItems.push(item);
      addUsage(id, { file: repoRel, line, kind: "inline-define" });
    }
  }
}

// ----------------------------------------------------------------------------
// 3) 汇总 lucide / inline svg 进清单，并回填 usageCount
// ----------------------------------------------------------------------------

for (const entry of lucideMap.values()) {
  const item = {
    id: entry.id,
    displayName: entry.importName,
    fileName: null,
    type: "lucide",
    source: "lucide-react",
    scanScope: "include",
    sourcePath: null,
    sourceDir: null,
    importName: entry.importName,
    webPath: null,
    importPath: null,
    bytes: null,
    usageCount: entry.files.size,
    usageRefs: entry.locations.length,
    classification: classificationPlaceholder(),
  };
  inventory.push(item);
  for (const loc of entry.locations) addUsage(entry.id, loc);
}

for (const item of inlineSvgItems) inventory.push(item);

// 回填文件资源 usageCount（去重文件数）/ usageRefs（引用次数），并标注目录级动态引用
for (const item of inventory) {
  if (item.type === "lucide" || item.type === "inline-svg") continue;
  const refs = usageMap.get(item.id) || [];
  item.usageRefs = refs.length;
  item.usageCount = new Set(refs.map((r) => r.file)).size;

  // 目录级动态引用：该文件所在目录若被字面量引用（基路径拼接），标注以避免误判未引用
  const parentAbs = item.sourcePath
    ? path.dirname(path.resolve(repoRoot, item.sourcePath))
    : null;
  const dirRefs = parentAbs && dirRefMap.has(parentAbs) ? dirRefMap.get(parentAbs) : null;
  item.dynamicDirReferenced = Boolean(dirRefs);
  item.dynamicDirRefCount = dirRefs ? dirRefs.length : 0;
}

// 稳定排序
inventory.sort((a, b) => a.id.localeCompare(b.id));
unresolvedRefs.sort(
  (a, b) =>
    (a.file || "").localeCompare(b.file || "") || (a.line || 0) - (b.line || 0)
);

// ----------------------------------------------------------------------------
// 4) 统计与输出
// ----------------------------------------------------------------------------

function countBy(items, fn) {
  const out = {};
  for (const it of items) {
    const k = fn(it);
    out[k] = (out[k] || 0) + 1;
  }
  return out;
}

const fileItems = inventory.filter((i) => i.type === "svg" || i.type === "png");
const generatedAt = new Date().toISOString();
const stageNote =
  "阶段 2 数据基线：仅事实（资源清单 + 使用位置/次数）。classification.pending=true 的字段交阶段 3+ 回填，未做任何分类/重复/canonical 判定。";

const inventoryOut = {
  $schema: "clawpro-resource-inventory",
  stage: 2,
  generatedAt,
  note: stageNote,
  scanDirs: FILE_SCAN_DIRS,
  summary: {
    total: inventory.length,
    byType: countBy(inventory, (i) => i.type),
    bySource: countBy(inventory, (i) => i.source),
    byScanScope: countBy(inventory, (i) => i.scanScope),
    fileResources: fileItems.length,
    lucide: inventory.filter((i) => i.type === "lucide").length,
    inlineSvg: inventory.filter((i) => i.type === "inline-svg").length,
    // 「无静态字面量引用」：未匹配到带扩展名完整路径的引用（可能仍被动态拼接引用，勿据此删除）
    noStaticRef: fileItems.filter((i) => i.usageCount === 0).length,
    // 「无任何引用线索」：连所在目录的字面量引用都没有，较接近真正未使用，仍需人工/动态复核
    noStaticNorDirRef: fileItems.filter(
      (i) => i.usageCount === 0 && !i.dynamicDirReferenced
    ).length,
    dynamicDirReferenced: fileItems.filter((i) => i.dynamicDirReferenced).length,
    multiUseFileResources: fileItems.filter((i) => i.usageCount > 1).length,
  },
  items: inventory,
};

const usageOut = {
  $schema: "clawpro-resource-usage",
  stage: 2,
  generatedAt,
  note: stageNote,
  summary: {
    resourcesWithUsage: usageMap.size,
    totalUsageRefs: [...usageMap.values()].reduce((n, arr) => n + arr.length, 0),
    unresolvedRefs: unresolvedRefs.length,
    dirReferencedDirs: dirRefMap.size,
  },
  usage: Object.fromEntries(
    [...usageMap.entries()].sort((a, b) => a[0].localeCompare(b[0])).map(([id, refs]) => [
      id,
      refs
        .slice()
        .sort((x, y) => (x.file || "").localeCompare(y.file || "") || (x.line || 0) - (y.line || 0)),
    ])
  ),
  // 目录级字面量引用（动态基路径拼接证据），key 为仓库相对目录
  dirReferences: Object.fromEntries(
    [...dirRefMap.entries()]
      .map(([abs, refs]) => [
        toRepoRel(abs),
        refs.slice().sort((x, y) => x.file.localeCompare(y.file) || x.line - y.line),
      ])
      .sort((a, b) => a[0].localeCompare(b[0]))
  ),
  unresolvedRefs,
};

fs.mkdirSync(outDir, { recursive: true });
fs.writeFileSync(
  path.join(outDir, "resource-inventory.generated.json"),
  JSON.stringify(inventoryOut, null, 2) + "\n"
);
fs.writeFileSync(
  path.join(outDir, "resource-usage.generated.json"),
  JSON.stringify(usageOut, null, 2) + "\n"
);

// 控制台摘要
console.log("ClawPro 资源扫描完成（阶段 2，只读）。");
console.log("产物目录:", toRepoRel(outDir));
console.log("资源条目总数:", inventory.length);
console.log("  按类型:", JSON.stringify(inventoryOut.summary.byType));
console.log("  按来源:", JSON.stringify(inventoryOut.summary.bySource));
console.log("  按 scope:", JSON.stringify(inventoryOut.summary.byScanScope));
console.log("文件资源:", fileItems.length, "| 无静态引用:", inventoryOut.summary.noStaticRef, "| 其中无任何引用线索:", inventoryOut.summary.noStaticNorDirRef, "| 目录级动态引用:", inventoryOut.summary.dynamicDirReferenced, "| 多处引用:", inventoryOut.summary.multiUseFileResources);
console.log("lucide 图标:", inventoryOut.summary.lucide, "| inline svg:", inventoryOut.summary.inlineSvg);
console.log("引用总数:", usageOut.summary.totalUsageRefs, "| 未解析引用:", usageOut.summary.unresolvedRefs, "| 被引用目录:", usageOut.summary.dirReferencedDirs);

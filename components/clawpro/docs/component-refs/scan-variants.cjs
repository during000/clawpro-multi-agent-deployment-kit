#!/usr/bin/env node
/**
 * 扫描 openclaw-enterprise，把"哪个组件的哪个变体在哪些页面被用过"挖出来。
 *
 * 解决的痛点：
 *   原来的 component-page-refs.json 只告诉你「button 在 130 个页面被用」，
 *   但 button 实际有 6 种 variant × 7 种 size，每个页面用的形式不同。
 *   要给设计师/前端分工，需要把维度细化到「claw-primary primary 按钮在 X/Y/Z 页用了」。
 *
 * 思路：
 *   1) 解析 client/src/components/ui/<name>.tsx 里的 cva(...) 定义，抽出
 *      所有 variant 维度（通常是 variant + size）和它们的取值（比如 default / claw-primary / ...）。
 *   2) 扫 client/src/pages/**.tsx，正则定位 <ComponentName ... /> 这种 JSX 元素，
 *      抽出每个 element 上的 variant/size 取值（写死字符串场景：variant="claw-primary"）。
 *      变量场景（variant={x}）暂时归为 dynamic，让人工兜底。
 *   3) 反向索引：对每个 (component, dimension, value)，列出它出现在哪些页面（含次数）。
 *
 * 用法：
 *   cd 组件引用及更新
 *   node scan-variants.cjs
 *
 * 输出：
 *   组件引用及更新/component-variants.json
 */

const fs = require("fs");
const path = require("path");

// ───────── 路径解析 ─────────
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

// 这里用 kebab-case 文件名（与 component-page-refs.json 一致）
const COMP_NAMES = [
  "accordion","alert","alert-dialog","aspect-ratio","avatar","badge","breadcrumb",
  "button","button-group","calendar","card","carousel","chart","checkbox","collapsible",
  "command","context-menu","dialog","drawer","dropdown-menu","empty","field","form",
  "hover-card","input","input-group","input-otp","item","kbd","label","menubar",
  "navigation-menu","pagination","popover","progress","radio-group","resizable",
  "scroll-area","select","separator","sheet","sidebar","skeleton","slider","sonner",
  "spinner","switch","table","tabs","textarea","toggle","toggle-group","tooltip",
];

// kebab-case → 实际 JSX 标签（PascalCase）。一个文件可能 export 多个标签，这里给主标签。
// 命中页面里 <PascalName ... /> 即视为该组件用法。
const PRIMARY_JSX_NAME = {
  "accordion": "Accordion",
  "alert": "Alert",
  "alert-dialog": "AlertDialog",
  "aspect-ratio": "AspectRatio",
  "avatar": "Avatar",
  "badge": "Badge",
  "breadcrumb": "Breadcrumb",
  "button": "Button",
  "button-group": "ButtonGroup",
  "calendar": "Calendar",
  "card": "Card",
  "carousel": "Carousel",
  "chart": "Chart",
  "checkbox": "Checkbox",
  "collapsible": "Collapsible",
  "command": "Command",
  "context-menu": "ContextMenu",
  "dialog": "Dialog",
  "drawer": "Drawer",
  "dropdown-menu": "DropdownMenu",
  "empty": "Empty",
  "field": "Field",
  "form": "Form",
  "hover-card": "HoverCard",
  "input": "Input",
  "input-group": "InputGroup",
  "input-otp": "InputOTP",
  "item": "Item",
  "kbd": "Kbd",
  "label": "Label",
  "menubar": "Menubar",
  "navigation-menu": "NavigationMenu",
  "pagination": "Pagination",
  "popover": "Popover",
  "progress": "Progress",
  "radio-group": "RadioGroup",
  "resizable": "Resizable",
  "scroll-area": "ScrollArea",
  "select": "Select",
  "separator": "Separator",
  "sheet": "Sheet",
  "sidebar": "Sidebar",
  "skeleton": "Skeleton",
  "slider": "Slider",
  "sonner": "Toaster",
  "spinner": "Spinner",
  "switch": "Switch",
  "table": "Table",
  "tabs": "Tabs",
  "textarea": "Textarea",
  "toggle": "Toggle",
  "toggle-group": "ToggleGroup",
  "tooltip": "Tooltip",
};

// ───────── 通用 ─────────
function readSafe(p) {
  try { return fs.readFileSync(p, "utf8"); } catch { return ""; }
}

function walk(dir) {
  const out = [];
  if (!fs.existsSync(dir)) return out;
  for (const name of fs.readdirSync(dir)) {
    const full = path.join(dir, name);
    const stat = fs.statSync(full);
    if (stat.isDirectory()) out.push(...walk(full));
    else if (stat.isFile() && full.endsWith(".tsx")) out.push(full);
  }
  return out;
}

// ───────── 1) 解析 cva 定义，抽出每个组件的 variant 维度 ─────────

/**
 * 在一个对象字面量的"内层文本"里，抽出所有顶层 key。
 *
 * 输入示例（已剥掉最外层 {}）：
 *   default: "border-transparent bg-primary ...",
 *   secondary:
 *     "border-transparent bg-secondary ...",
 *   destructive: "...",
 *   outline: "..."
 *
 * 难点：value 里可能有 `,` `:` 模板字符串、嵌套对象、注释。所以用状态机：
 *   - 跟踪 {}, [], () 深度
 *   - 跟踪 "/'/` 字符串、模板内 ${...}
 *   - 跟踪 // 行注释 和 /* 块注释 *​/
 *   - 只在深度=0 + 不在字符串/注释时，匹配 `key:` 模式
 *
 * 支持的 key 形式：
 *   identifier:           → identifier
 *   "string-key":         → string-key
 *   'string-key':         → string-key
 */
function extractTopLevelKeys(text) {
  const keys = [];
  let i = 0;
  const n = text.length;
  let braceDepth = 0;   // {}
  let bracketDepth = 0; // []
  let parenDepth = 0;   // ()
  let strChar = null;   // " / ' / `
  let inLineComment = false;
  let inBlockComment = false;
  // 模板字符串里的 ${...} 嵌套：每进入一次模板字符串，记一个独立的 brace 计数
  // 简单起见：只要在模板字符串里看到 `${`，就把后续内容视为代码（深度 +1）
  // 这里我们把模板字符串处理简化：遇到 ${ 退出字符串模式，碰到对应 } 再回来。用栈记录。
  const tmplStack = []; // 元素：{ inside: true } 表示之前在模板字符串中

  function isIdentStart(ch) { return /[A-Za-z_$]/.test(ch); }
  function isIdentPart(ch)  { return /[A-Za-z0-9_$-]/.test(ch); }

  while (i < n) {
    const ch = text[i];
    const next = text[i + 1];

    // 行注释
    if (inLineComment) {
      if (ch === "\n") inLineComment = false;
      i++; continue;
    }
    // 块注释
    if (inBlockComment) {
      if (ch === "*" && next === "/") { inBlockComment = false; i += 2; continue; }
      i++; continue;
    }
    // 字符串里
    if (strChar) {
      if (ch === "\\") { i += 2; continue; } // 跳过转义
      if (strChar === "`" && ch === "$" && next === "{") {
        // 进入模板插值：记录后弹出字符串模式
        tmplStack.push({ inside: true });
        strChar = null;
        i += 2;
        braceDepth++;
        continue;
      }
      if (ch === strChar) { strChar = null; i++; continue; }
      i++; continue;
    }

    // 不在字符串/注释里
    if (ch === "/" && next === "/") { inLineComment = true; i += 2; continue; }
    if (ch === "/" && next === "*") { inBlockComment = true; i += 2; continue; }

    if (ch === '"' || ch === "'" || ch === "`") { strChar = ch; i++; continue; }

    if (ch === "{") { braceDepth++; i++; continue; }
    if (ch === "}") {
      braceDepth--;
      // 如果 braceDepth 归到模板栈记录的层级，回到模板字符串
      if (tmplStack.length > 0 && braceDepth === tmplStack.length - 1) {
        // 不太精确，但够用：检测关闭 ${...} 的瞬间，回到模板字符串模式
        // 这里更稳的判断是比较"进入模板时的 braceDepth"。简化处理：弹出并恢复 strChar。
        tmplStack.pop();
        strChar = "`";
      }
      i++; continue;
    }
    if (ch === "[") { bracketDepth++; i++; continue; }
    if (ch === "]") { bracketDepth--; i++; continue; }
    if (ch === "(") { parenDepth++; i++; continue; }
    if (ch === ")") { parenDepth--; i++; continue; }

    // 只在最外层（braceDepth=0 && bracketDepth=0 && parenDepth=0）时匹配 key
    if (braceDepth === 0 && bracketDepth === 0 && parenDepth === 0) {
      // 双引号 key
      if (ch === '"' || ch === "'") { /* 已在上面处理为字符串 */ i++; continue; }
      // identifier 开头
      if (isIdentStart(ch)) {
        let j = i + 1;
        while (j < n && isIdentPart(text[j])) j++;
        const ident = text.slice(i, j);
        // 跳过空白后看是不是 ":"
        let k = j;
        while (k < n && /\s/.test(text[k])) k++;
        if (text[k] === ":") {
          // 排除 "?:" 这种 ts 可选属性的尾巴；这里已经是顶层 key 形态
          keys.push(ident);
          i = k + 1;
          continue;
        }
        i = j;
        continue;
      }
    }

    i++;
  }

  // 上面的简化扫描没处理 "string-key" 形式，这里再扫一遍补上（顶层 + 字符串 key）
  // 因为 cva 里几乎不出现 "...":, 这条用最朴素的正则补：
  const reStrKey = /(?:^|[,\n\r])\s*("([^"]+)"|'([^']+)')\s*:/g;
  let sm;
  while ((sm = reStrKey.exec(text))) {
    const key = sm[2] || sm[3];
    if (key && !keys.includes(key)) keys.push(key);
  }

  return keys;
}

/**
 * 从一个 ui 组件源码里抽出 cva(...) 块，返回结构：
 *   {
 *     hasCva: true,
 *     dimensions: {
 *       variant: { values: ['default','destructive',...], default: 'default' },
 *       size:    { values: ['default','sm','lg',...],     default: 'default' },
 *     }
 *   }
 *
 * 解析策略：纯文本/正则，不引 babel，避免依赖。允许少量误差，对 shadcn 这套写法足够稳。
 */
function parseCvaDefinition(src) {
  const result = { hasCva: false, dimensions: {} };

  // 找到 cva(...) 的入口；shadcn 习惯把第二个参数作为配置对象
  // 形如：const fooVariants = cva("...base...", { variants: { variant: {...}, size: {...} }, defaultVariants: {...} })
  const cvaIdx = src.indexOf("cva(");
  if (cvaIdx < 0) return result;
  result.hasCva = true;

  // 截取从 "variants:" 开始的那块；这里靠括号配对很难，但只要拿到 variants 内层
  // 我们用一个简化的状态机：定位 "variants:" → 找到对应 "{"，按 {} 配对找到结束 "}"。
  const variantsKeyIdx = src.indexOf("variants:", cvaIdx);
  if (variantsKeyIdx < 0) return result;

  const openBraceIdx = src.indexOf("{", variantsKeyIdx);
  if (openBraceIdx < 0) return result;

  // 从 openBraceIdx 开始按 { } 配对，直到深度归零
  let depth = 0;
  let endIdx = -1;
  for (let i = openBraceIdx; i < src.length; i++) {
    const ch = src[i];
    if (ch === "{") depth++;
    else if (ch === "}") {
      depth--;
      if (depth === 0) { endIdx = i; break; }
    }
  }
  if (endIdx < 0) return result;

  const variantsBlock = src.slice(openBraceIdx + 1, endIdx);

  // 在 variantsBlock 内找形如 `xxx: {` 的子块，每个子块就是一个维度
  // 例如：variant: { default: "...", destructive: "...", ... }
  // 注意嵌套：每个维度内部也是 {}，要按 {} 配对取出每个 dimension 的内容
  let cursor = 0;
  while (cursor < variantsBlock.length) {
    const m = /([a-zA-Z_][\w-]*)\s*:\s*\{/.exec(variantsBlock.slice(cursor));
    if (!m) break;
    const dimName = m[1];
    const dimOpen = cursor + m.index + m[0].length - 1; // '{' 的位置（在原 variantsBlock 中）
    let d = 0;
    let dimEnd = -1;
    for (let i = dimOpen; i < variantsBlock.length; i++) {
      const ch = variantsBlock[i];
      if (ch === "{") d++;
      else if (ch === "}") {
        d--;
        if (d === 0) { dimEnd = i; break; }
      }
    }
    if (dimEnd < 0) break;
    const inner = variantsBlock.slice(dimOpen + 1, dimEnd);
    cursor = dimEnd + 1;

    // 提取 inner 里所有"顶层 key"：用状态机扫描，跟踪括号/字符串深度，只在深度 0 时记录 key
    const keys = extractTopLevelKeys(inner);

    // 去重保持顺序
    const seen = new Set();
    const values = [];
    for (const k of keys) {
      if (!seen.has(k)) { seen.add(k); values.push(k); }
    }
    if (values.length > 0) {
      result.dimensions[dimName] = { values, default: null };
    }
  }

  // 解析 defaultVariants
  const dvIdx = src.indexOf("defaultVariants", endIdx);
  if (dvIdx > 0) {
    const dvOpen = src.indexOf("{", dvIdx);
    if (dvOpen > 0) {
      let d = 0, dvEnd = -1;
      for (let i = dvOpen; i < src.length; i++) {
        const ch = src[i];
        if (ch === "{") d++;
        else if (ch === "}") { d--; if (d === 0) { dvEnd = i; break; } }
      }
      if (dvEnd > 0) {
        const dvBlock = src.slice(dvOpen + 1, dvEnd);
        const re = /([a-zA-Z_][\w-]*)\s*:\s*"([^"]+)"/g;
        let mm;
        while ((mm = re.exec(dvBlock))) {
          const dim = mm[1];
          const val = mm[2];
          if (result.dimensions[dim]) result.dimensions[dim].default = val;
        }
      }
    }
  }

  return result;
}

// ───────── 2) 扫页面，找 <Component .../> 并抽 variant/size props ─────────
/**
 * 在一段源码里找出形如 <Tag ...> 的标签起始位置和它的属性串。
 * 处理 self-closing 和 children 两种；只关心起始标签。
 */
function findOpeningTags(src, tagName) {
  const out = [];
  // 匹配 <Tag 开头：要求后一个字符是空白或 / 或 > 或换行，避免 <ButtonGroup 命中 <Button
  const re = new RegExp(`<${tagName}(?=[\\s/>])`, "g");
  let m;
  while ((m = re.exec(src))) {
    const start = m.index;
    // 找到对应的 ">"，但要跳过 {...} 和字符串中的 >
    let i = start + m[0].length;
    let depthBrace = 0; // {}
    let inStr = null;   // " or ' or `
    while (i < src.length) {
      const ch = src[i];
      if (inStr) {
        if (ch === "\\") { i += 2; continue; }
        if (ch === inStr) inStr = null;
      } else {
        if (ch === "{") depthBrace++;
        else if (ch === "}") depthBrace--;
        else if (ch === '"' || ch === "'" || ch === "`") inStr = ch;
        else if (ch === ">" && depthBrace === 0) { break; }
      }
      i++;
    }
    if (i >= src.length) continue;
    const propsStr = src.slice(start + m[0].length, i);
    out.push({ tagName, propsStr });
  }
  return out;
}

/**
 * 从属性串里抽出某个 prop 的值。返回：
 *   - { kind: "literal", value: "claw-primary" }
 *   - { kind: "dynamic", value: "<expr>" }   // {variable} 形式
 *   - null  // 没写
 */
function extractProp(propsStr, propName) {
  // 静态 propName="literal"
  const reLit = new RegExp(`\\b${propName}\\s*=\\s*"([^"]*)"`);
  const ml = reLit.exec(propsStr);
  if (ml) return { kind: "literal", value: ml[1] };
  // 动态 propName={...}
  const reDyn = new RegExp(`\\b${propName}\\s*=\\s*\\{`);
  const md = reDyn.exec(propsStr);
  if (md) {
    // 取出 {...} 内容
    let i = md.index + md[0].length - 1; // 指向 '{'
    let depth = 0;
    let start = i + 1;
    while (i < propsStr.length) {
      const ch = propsStr[i];
      if (ch === "{") depth++;
      else if (ch === "}") { depth--; if (depth === 0) { return { kind: "dynamic", value: propsStr.slice(start, i).trim() }; } }
      i++;
    }
    return { kind: "dynamic", value: "" };
  }
  return null;
}

// ───────── 3) 主流程 ─────────
const componentDefs = {};   // { compName: { hasCva, dimensions } }
for (const name of COMP_NAMES) {
  const file = path.join(UI_DIR, name + ".tsx");
  if (!fs.existsSync(file)) {
    componentDefs[name] = { hasCva: false, dimensions: {}, fileExists: false };
    continue;
  }
  const src = readSafe(file);
  const def = parseCvaDefinition(src);
  componentDefs[name] = { ...def, fileExists: true };
}

const pageFiles = walk(PAGES_DIR);

/**
 * 反向索引数据结构：
 *   variantsIndex[compName] = {
 *     hasCva: boolean,
 *     dimensions: { variant: { values:[...], default:'default' }, size: {...} },
 *     usagesByVariant: {
 *       // 维度组合的"代表性 key"：variant=claw-primary&size=claw
 *       // 但人类查看更习惯按维度分开，所以也提供 byDimension 视图：
 *     },
 *     byDimension: {
 *       variant: {
 *         "claw-primary": [ { page, count }, ... ],
 *         "default":      [ ... ],
 *         "(unset)":      [ ... ],   // 没显式写 → 用 cva default
 *         "(dynamic)":    [ ... ],   // 写的是 {expr}
 *         "(unknown)":    [ ... ],   // 写了字符串但不在 cva 列表里（比如 v2 自定义的）
 *       },
 *       size: {...}
 *     },
 *     totalUsages: number,           // 该组件 JSX 出现总次数
 *     pagesWithUsage: string[],
 *   }
 */
const variantsIndex = {};

for (const compName of COMP_NAMES) {
  const def = componentDefs[compName];
  variantsIndex[compName] = {
    hasCva: def.hasCva,
    dimensions: def.dimensions,
    byDimension: {},
    totalUsages: 0,
    pagesWithUsage: new Set(),
  };
  // 给 byDimension 预先按 cva 列出所有维度（值用空数组占位）
  for (const dim of Object.keys(def.dimensions || {})) {
    variantsIndex[compName].byDimension[dim] = {};
    for (const v of def.dimensions[dim].values) {
      variantsIndex[compName].byDimension[dim][v] = [];
    }
    variantsIndex[compName].byDimension[dim]["(unset)"] = [];
    variantsIndex[compName].byDimension[dim]["(dynamic)"] = [];
    variantsIndex[compName].byDimension[dim]["(unknown)"] = [];
  }
}

for (const file of pageFiles) {
  const rel = path.relative(PAGES_DIR, file).replace(/\\/g, "/");
  const src = readSafe(file);

  for (const compName of COMP_NAMES) {
    const tag = PRIMARY_JSX_NAME[compName];
    if (!tag) continue;
    if (!src.includes("<" + tag)) continue; // 快速过滤

    const tags = findOpeningTags(src, tag);
    if (tags.length === 0) continue;

    const idx = variantsIndex[compName];
    idx.totalUsages += tags.length;
    idx.pagesWithUsage.add(rel);

    const dims = Object.keys(componentDefs[compName].dimensions || {});
    if (dims.length === 0) continue; // 没 cva 的组件，没必要按变体分

    // 累加：同一页可能用了多次同一变体
    for (const t of tags) {
      for (const dim of dims) {
        const got = extractProp(t.propsStr, dim);
        let bucket;
        if (!got) bucket = "(unset)";
        else if (got.kind === "dynamic") bucket = "(dynamic)";
        else {
          const allowed = componentDefs[compName].dimensions[dim].values;
          bucket = allowed.includes(got.value) ? got.value : "(unknown)";
        }
        if (!idx.byDimension[dim][bucket]) idx.byDimension[dim][bucket] = [];
        // 在该 bucket 里找 page 记录，count++
        const arr = idx.byDimension[dim][bucket];
        const ex = arr.find(x => x.page === rel);
        if (ex) ex.count += 1;
        else arr.push({ page: rel, count: 1, value: bucket === "(unknown)" && got ? got.value : undefined });
      }
    }
  }
}

// ───────── 4) 序列化输出 ─────────
function serializable(idx) {
  const out = {};
  for (const k of Object.keys(idx)) {
    const v = idx[k];
    out[k] = {
      hasCva: v.hasCva,
      dimensions: v.dimensions,
      byDimension: v.byDimension,
      totalUsages: v.totalUsages,
      pagesWithUsage: Array.from(v.pagesWithUsage).sort(),
    };
    // 清掉空 bucket（unset/dynamic/unknown 经常没人命中），让产物更短
    for (const dim of Object.keys(out[k].byDimension)) {
      const dv = out[k].byDimension[dim];
      for (const bucket of Object.keys(dv)) {
        if (dv[bucket].length === 0) delete dv[bucket];
        else dv[bucket].sort((a, b) => a.page.localeCompare(b.page));
      }
    }
  }
  return out;
}

const payload = {
  generator: "scan-variants.cjs",
  generatedAt: new Date().toISOString(),
  repo: path.relative(SCRIPT_DIR, REPO),
  notes: [
    "(unset)   = 调用处没显式写该 prop，运行时取 cva defaultVariants 的值",
    "(dynamic) = 写的是 {expr} 这种动态值，需要人工确认",
    "(unknown) = 显式传了字符串但不在 cva 已声明的取值里（多半是自定义/历史值/拼写错）",
    "byDimension 是分维度反向索引：组件 → 维度（variant/size/...）→ 变体值 → 引用页面",
  ],
  components: serializable(variantsIndex),
};

const out = path.join(SCRIPT_DIR, "component-variants.json");
fs.writeFileSync(out, JSON.stringify(payload, null, 2), "utf8");

// ───────── 5) 控制台摘要 ─────────
console.log("✅ 已生成：" + out);
console.log("");
console.log("📊 变体扫描摘要");
console.log("──────────────────────────────────");
const cvaList = COMP_NAMES.filter(n => componentDefs[n].hasCva);
console.log(`有 cva 变体定义的组件：${cvaList.length} / ${COMP_NAMES.length}`);
console.log("");
for (const name of cvaList) {
  const def = componentDefs[name];
  const idx = variantsIndex[name];
  const dims = Object.keys(def.dimensions);
  const dimSummary = dims.map(d =>
    `${d}(${def.dimensions[d].values.length})`
  ).join(" × ");
  console.log(`  ${name.padEnd(14)}  ${dimSummary.padEnd(28)}  ${idx.totalUsages} 处 / ${idx.pagesWithUsage.size} 页`);
}
console.log("");
console.log("提示：打开 pages-index.html，切到「变体视图」标签查看详情。");

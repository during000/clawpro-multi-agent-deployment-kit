// =============================================================================
// 分拣工具生成器：build-gradient-sorter.mjs
// -----------------------------------------------------------------------------
// 定位：为「渐变图标 线条/块状 人工分拣」生成一个自包含的可视化页面
//       client/public/_gradient-sorter.html。
//
// 纪律：
//   1. 只读 inventory 与已有人工清单，不修改任何资源 / 分类数据。
//   2. 页面内嵌当前所有「渐变系」图标（visualType ∈ gradient / gradient-line /
//      gradient-solid），缩略图走 webPath（由 vite dev server 在根路径提供）。
//   3. 已分拣的项会带上既有选择预填，便于复核与修正。
//   4. 生成物是工具页，不进入正式资源数据；分拣结论以导出的
//      manual-overrides/gradient-style.json 为准。
//
// 运行：node client/src/design-assets/scripts/build-gradient-sorter.mjs
//      然后 pnpm dev，浏览器打开 /_gradient-sorter.html
// =============================================================================

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "../../../../");

const INVENTORY_PATH = path.join(
  repoRoot,
  "client/src/design-assets/generated/resource-inventory.generated.json"
);
const OVERRIDES_PATH = path.join(
  repoRoot,
  "client/src/design-assets/manual-overrides/gradient-style.json"
);
const OUT_PATH = path.join(repoRoot, "client/public/_gradient-sorter.html");

const GRADIENT_TYPES = new Set(["gradient", "gradient-line", "gradient-solid"]);

const inventory = JSON.parse(fs.readFileSync(INVENTORY_PATH, "utf8"));

let presetOverrides = {};
try {
  const ov = JSON.parse(fs.readFileSync(OVERRIDES_PATH, "utf8"));
  presetOverrides = ov.overrides || {};
} catch {
  presetOverrides = {};
}

const items = inventory.items
  .filter((i) => GRADIENT_TYPES.has(i.classification?.visualType))
  .map((i) => ({
    id: i.id,
    displayName: i.displayName,
    sourcePath: i.sourcePath,
    webPath: i.webPath,
    // dev 可访问 URL：public 资源用 webPath（/icon|/assets...）；
    // src 资源 webPath 为空，由 sourcePath 推导 /src/... （vite root=client，能直接 GET 原始 svg）
    url:
      i.webPath ||
      (i.sourcePath ? "/" + i.sourcePath.replace(/^client\//, "") : ""),
    // 既有人工选择（若有）→ 预填；否则看当前 visualType 是否已是细分值
    preset:
      presetOverrides[i.id]?.style ||
      (i.classification.visualType === "gradient" ? null : i.classification.visualType),
  }))
  .sort((a, b) => (a.sourcePath || "").localeCompare(b.sourcePath || ""));

const DATA_JSON = JSON.stringify(items);

const html = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>渐变图标分拣 · 线条 / 块状</title>
<style>
  :root{ --line:#1447E6; --solid:#0EA5E9; --border:#DDE7F2; --muted:#64748B; --ink:#0A0A0A; }
  *{ box-sizing:border-box; }
  body{ margin:0; font:14px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif; color:var(--ink); background:#F7FAFF; }
  header{ position:sticky; top:0; z-index:10; background:linear-gradient(180deg,#fff,#F7FAFF); border-bottom:1px solid var(--border); padding:14px 20px; }
  h1{ font-size:16px; margin:0 0 4px; }
  .sub{ color:var(--muted); font-size:12px; }
  .bar{ display:flex; flex-wrap:wrap; align-items:center; gap:10px; margin-top:10px; }
  .count{ font-size:12px; color:var(--muted); }
  .count b{ color:var(--ink); }
  button{ font:inherit; cursor:pointer; border-radius:4px; border:1px solid var(--border); background:#fff; padding:6px 12px; }
  button.primary{ background:var(--line); border-color:var(--line); color:#fff; }
  button:disabled{ opacity:.5; cursor:not-allowed; }
  .filterbtns button.on{ background:#EEF3FF; border-color:var(--line); color:var(--line); }
  main{ padding:18px 20px 80px; }
  .grid{ display:grid; grid-template-columns:repeat(auto-fill,minmax(150px,1fr)); gap:12px; }
  .card{ border:1px solid var(--border); border-radius:8px; background:#fff; padding:10px; display:flex; flex-direction:column; gap:8px; }
  .card.line{ border-color:var(--line); box-shadow:0 0 0 1px var(--line) inset; }
  .card.solid{ border-color:var(--solid); box-shadow:0 0 0 1px var(--solid) inset; }
  .thumb{ height:72px; display:flex; align-items:center; justify-content:center; background:
      linear-gradient(45deg,#f1f5f9 25%,transparent 25%,transparent 75%,#f1f5f9 75%) 0 0/16px 16px,
      linear-gradient(45deg,#f1f5f9 25%,#fff 25%,#fff 75%,#f1f5f9 75%) 8px 8px/16px 16px;
      border-radius:6px; }
  .thumb img{ max-width:56px; max-height:56px; }
  .name{ font-size:12px; font-weight:500; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
  .path{ font-size:10px; color:#94A3B8; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
  .pick{ display:flex; gap:6px; }
  .pick button{ flex:1; padding:5px 0; font-size:12px; }
  .pick .l.on{ background:var(--line); border-color:var(--line); color:#fff; }
  .pick .s.on{ background:var(--solid); border-color:var(--solid); color:#fff; }
  .hint{ background:#FFF7ED; border:1px solid #FED7AA; color:#B8640A; border-radius:6px; padding:8px 12px; font-size:12px; margin-top:10px; }
  textarea{ width:100%; height:120px; margin-top:10px; font-family:ui-monospace,Menlo,monospace; font-size:11px; border:1px solid var(--border); border-radius:6px; padding:8px; display:none; }
</style>
</head>
<body>
<header>
  <h1>渐变图标分拣 · 线条多彩渐变 / 块状多彩渐变</h1>
  <div class="sub">逐个点选每张图的风格，进度自动保存在浏览器（刷新不丢）。全部分完后点「下载 gradient-style.json」，覆盖保存到 <code>client/src/design-assets/manual-overrides/gradient-style.json</code>，再重跑分类脚本。</div>
  <div class="bar">
    <span class="count">已分 <b id="done">0</b> / <span id="total">0</span></span>
    <span class="filterbtns" style="display:flex;gap:6px;">
      <button data-f="all" class="on">全部</button>
      <button data-f="todo">未分</button>
      <button data-f="line">线条</button>
      <button data-f="solid">块状</button>
    </span>
    <span style="flex:1"></span>
    <button id="download" class="primary">下载 gradient-style.json</button>
    <button id="copy">复制 JSON</button>
    <button id="reset">清空选择</button>
  </div>
  <div class="hint" id="hint">提示：本页仅用于分拣，结论以导出的 JSON 为准；不会写入任何项目数据。</div>
  <textarea id="json" readonly></textarea>
</header>
<main><div class="grid" id="grid"></div></main>
<script>
const ITEMS = ${DATA_JSON};
const LS_KEY = "gradient-style-sort-v1";
const total = ITEMS.length;
document.getElementById("total").textContent = total;

let store = {};
try { store = JSON.parse(localStorage.getItem(LS_KEY)) || {}; } catch { store = {}; }
// 预填：localStorage 优先，其次脚本注入的 preset
for (const it of ITEMS) {
  if (!store[it.id] && it.preset) store[it.id] = it.preset;
}

let filter = "all";

function save(){ localStorage.setItem(LS_KEY, JSON.stringify(store)); }
function pick(id, style){
  if (store[id] === style) delete store[id]; else store[id] = style; // 再次点击同项=取消
  save(); render();
}
function styleOf(id){ return store[id] || null; }

function render(){
  const grid = document.getElementById("grid");
  grid.innerHTML = "";
  let done = 0;
  for (const it of ITEMS){
    const st = styleOf(it.id);
    if (st) done++;
    if (filter === "todo" && st) continue;
    if (filter === "line" && st !== "gradient-line") continue;
    if (filter === "solid" && st !== "gradient-solid") continue;
    const card = document.createElement("div");
    card.className = "card" + (st === "gradient-line" ? " line" : st === "gradient-solid" ? " solid" : "");
    card.innerHTML =
      '<div class="thumb"><img loading="lazy" src="' + (it.url || it.webPath || "") + '" alt="" onerror="this.style.opacity=.2;this.alt=\\'缺图\\'"/></div>' +
      '<div class="name" title="' + esc(it.displayName) + '">' + esc(it.displayName) + '</div>' +
      '<div class="path" title="' + esc(it.sourcePath) + '">' + esc(it.sourcePath) + '</div>' +
      '<div class="pick">' +
        '<button class="l' + (st==="gradient-line"?" on":"") + '">线条</button>' +
        '<button class="s' + (st==="gradient-solid"?" on":"") + '">块状</button>' +
      '</div>';
    const [lb, sb] = card.querySelectorAll(".pick button");
    lb.onclick = () => pick(it.id, "gradient-line");
    sb.onclick = () => pick(it.id, "gradient-solid");
    grid.appendChild(card);
  }
  document.getElementById("done").textContent = done;
  document.getElementById("download").disabled = done === 0;
  document.getElementById("copy").disabled = done === 0;
}
function esc(s){ return String(s==null?"":s).replace(/[&<>"]/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[c])); }

function buildJSON(){
  const byId = Object.fromEntries(ITEMS.map(i => [i.id, i]));
  const overrides = {};
  for (const id of Object.keys(store).sort()){
    const it = byId[id]; if (!it) continue;
    overrides[id] = { style: store[id], sourcePath: it.sourcePath, displayName: it.displayName };
  }
  return JSON.stringify({
    $schema: "clawpro-gradient-style-overrides",
    note: "人工分拣事实源：渐变图标的线条/块状归类。由 _gradient-sorter.html 导出。key=资源 id；value.style ∈ gradient-line | gradient-solid。",
    updatedAt: new Date().toISOString(),
    overrides
  }, null, 2) + "\\n";
}

document.querySelectorAll(".filterbtns button").forEach(b => b.onclick = () => {
  filter = b.dataset.f;
  document.querySelectorAll(".filterbtns button").forEach(x => x.classList.toggle("on", x===b));
  render();
});
document.getElementById("download").onclick = () => {
  const blob = new Blob([buildJSON()], { type: "application/json" });
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob); a.download = "gradient-style.json"; a.click();
  URL.revokeObjectURL(a.href);
};
document.getElementById("copy").onclick = async () => {
  const ta = document.getElementById("json"); ta.style.display = "block"; ta.value = buildJSON();
  try { await navigator.clipboard.writeText(ta.value); document.getElementById("copy").textContent = "已复制"; setTimeout(()=>document.getElementById("copy").textContent="复制 JSON",1500);} catch {}
};
document.getElementById("reset").onclick = () => {
  if (confirm("清空所有已分选择？")) { store = {}; save(); render(); }
};
render();
</script>
</body>
</html>
`;

fs.mkdirSync(path.dirname(OUT_PATH), { recursive: true });
fs.writeFileSync(OUT_PATH, html, "utf8");

const presetCount = items.filter((i) => i.preset).length;
console.log("== 渐变分拣页已生成 ==");
console.log("渐变系图标:", items.length, "| 已带预填:", presetCount);
console.log("产物:", path.relative(repoRoot, OUT_PATH));
console.log("用法: pnpm dev 后浏览器打开 /_gradient-sorter.html");

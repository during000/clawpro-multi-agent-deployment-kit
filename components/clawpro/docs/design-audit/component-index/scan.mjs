#!/usr/bin/env node
/**
 * 组件 ↔ 页面 交叉索引扫描脚本
 * --------------------------------
 * 用途：扫描 client/src/pages/** 下所有 .tsx 文件，统计：
 *   1) 每个 shadcn UI 组件被多少个页面引用
 *   2) 每个页面引用了哪些组件
 *   3) 按 顶级 / 用户端 / 管理端·顶级 / 管理端·子组件 分类
 *
 * 输出：docs/design-audit/component-index/data.json
 *      （供同目录下 report.html 读取）
 *
 * 草稿性质：只读不改业务代码，仅生成 JSON 数据文件。
 */

import { readdir, readFile, writeFile, stat } from "node:fs/promises";
import { join, relative, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
// 当前位置：<root>/docs/design-audit/component-index/，需上溯 3 层到项目根
const PROJECT_ROOT = join(__dirname, "..", "..", "..");
const PAGES_DIR = join(PROJECT_ROOT, "client/src/pages");
const UI_DIR = join(PROJECT_ROOT, "client/src/components/ui");
const OUTPUT_FILE = join(__dirname, "data.json");

// ---- 工具函数 -------------------------------------------------------------

/** 递归列出目录下所有 .tsx 文件 */
async function listTsxFiles(dir) {
  const out = [];
  const entries = await readdir(dir, { withFileTypes: true });
  for (const e of entries) {
    const full = join(dir, e.name);
    if (e.isDirectory()) {
      out.push(...(await listTsxFiles(full)));
    } else if (e.isFile() && e.name.endsWith(".tsx")) {
      out.push(full);
    }
  }
  return out;
}

/** 从 .tsx 文件内容中抽取所有 `from "@/components/ui/xxx"` 的 xxx */
function extractUiImports(source) {
  const re = /from\s+["']@\/components\/ui\/([a-z0-9-]+)["']/g;
  const found = new Set();
  let m;
  while ((m = re.exec(source)) !== null) {
    found.add(m[1]);
  }
  return [...found];
}

/** 把绝对路径转成相对项目的短路径（用 / 分隔） */
function toRel(p) {
  return relative(PROJECT_ROOT, p).split("\\").join("/");
}

/** 给页面分类：顶级 / 用户端 / 管理端·顶级 / 管理端·子组件 */
function classifyPage(relPath) {
  // relPath 形如 client/src/pages/admin/foo/Bar.tsx
  const rest = relPath.replace(/^client\/src\/pages\//, "");
  const segs = rest.split("/");
  if (segs.length === 1) {
    return { category: "顶级", categoryKey: "top" };
  }
  if (segs[0] === "tenant") {
    return { category: "用户端", categoryKey: "tenant" };
  }
  if (segs[0] === "admin") {
    // admin 下：直接子文件 = 管理端·顶级；子目录 = 管理端·子组件
    if (segs.length === 2) return { category: "管理端·顶级", categoryKey: "admin-top" };
    return { category: "管理端·子组件", categoryKey: "admin-sub" };
  }
  return { category: "其他", categoryKey: "other" };
}

/** 设计搭档定的 P 级（来自截图三：改造路线图） */
const PRIORITY_HINTS = {
  // P0 必改 - 6 个基石（覆盖 78%+ 页面）
  button: { level: "P0", note: "覆盖最广，全场景基石" },
  dialog: { level: "P0", note: "弹窗骨架，UI 一致性关键" },
  input: { level: "P0", note: "表单核心" },
  label: { level: "P0", note: "表单标签，与 input 配套" },
  // 设计搭档截图二/三里其余 P0 候选（根据高频补齐，可改）
  select: { level: "P0", note: "下拉选择，高频表单组件" },
  tooltip: { level: "P0", note: "高频信息悬浮" },
  // P1
  popover: { level: "P1", note: "气泡浮层" },
  switch: { level: "P1", note: "开关交互" },
  badge: { level: "P1", note: "状态徽标（注意：现状大量被全局 .badge-* 类替代）" },
  card: { level: "P1", note: "卡片容器（注意：现状大量被手写伪 Card 替代，102 处）" },
  checkbox: { level: "P1", note: "多选" },
  "dropdown-menu": { level: "P1", note: "下拉菜单" },
  "alert-dialog": { level: "P1", note: "确认弹窗" },
  tabs: { level: "P1", note: "分页签" },
  textarea: { level: "P1", note: "多行输入" },
  separator: { level: "P1", note: "分隔线" },
  sheet: { level: "P1", note: "抽屉" },
};

// ---- 主流程 ---------------------------------------------------------------

async function main() {
  console.log("📂 扫描页面目录:", toRel(PAGES_DIR));
  const pageFiles = await listTsxFiles(PAGES_DIR);
  console.log(`   找到 ${pageFiles.length} 个 .tsx 页面文件`);

  console.log("📂 扫描 UI 组件目录:", toRel(UI_DIR));
  const uiFiles = await listTsxFiles(UI_DIR);
  const allComponents = uiFiles.map((f) => f.split("/").pop().replace(".tsx", "")).sort();
  console.log(`   找到 ${allComponents.length} 个 UI 组件`);

  // 每个页面 → 用了哪些组件
  const pages = [];
  // 每个组件 → 被哪些页面用了
  const compToPages = new Map();
  allComponents.forEach((c) => compToPages.set(c, new Set()));

  for (const pf of pageFiles) {
    const src = await readFile(pf, "utf-8");
    const imports = extractUiImports(src);
    const rel = toRel(pf);
    const cls = classifyPage(rel);
    pages.push({
      path: rel,
      name: pf.split("/").pop().replace(".tsx", ""),
      category: cls.category,
      categoryKey: cls.categoryKey,
      components: imports.sort(),
    });
    for (const c of imports) {
      if (!compToPages.has(c)) compToPages.set(c, new Set());
      compToPages.get(c).add(rel);
    }
  }

  // 组件维度数据
  const components = [];
  for (const [name, pageSet] of compToPages.entries()) {
    const usedBy = [...pageSet].sort();
    const distribution = { top: 0, tenant: 0, "admin-top": 0, "admin-sub": 0, other: 0 };
    for (const p of usedBy) {
      const cls = classifyPage(p);
      distribution[cls.categoryKey] = (distribution[cls.categoryKey] || 0) + 1;
    }
    const hint = PRIORITY_HINTS[name] || { level: "P2", note: "" };
    components.push({
      name,
      file: `${name}.tsx`,
      usageCount: usedBy.length,
      usedBy,
      distribution,
      priority: hint.level,
      note: hint.note,
    });
  }
  // 按使用次数倒序，再按名字排
  components.sort((a, b) => b.usageCount - a.usageCount || a.name.localeCompare(b.name));

  // 按页面分类汇总
  const pagesByCategory = {
    top: pages.filter((p) => p.categoryKey === "top"),
    tenant: pages.filter((p) => p.categoryKey === "tenant"),
    "admin-top": pages.filter((p) => p.categoryKey === "admin-top"),
    "admin-sub": pages.filter((p) => p.categoryKey === "admin-sub"),
  };

  const data = {
    generatedAt: new Date().toISOString(),
    totals: {
      pages: pages.length,
      components: allComponents.length,
      byCategory: {
        顶级: pagesByCategory.top.length,
        用户端: pagesByCategory.tenant.length,
        "管理端·顶级": pagesByCategory["admin-top"].length,
        "管理端·子组件": pagesByCategory["admin-sub"].length,
      },
    },
    components,
    pages,
  };

  await writeFile(OUTPUT_FILE, JSON.stringify(data, null, 2), "utf-8");
  console.log("\n✅ 数据已生成:", toRel(OUTPUT_FILE));

  // 终端打印 Top 10 高频组件，用于和设计搭档数字比对
  console.log("\n📊 Top 10 高频组件（按页面数）:");
  console.log("─".repeat(60));
  console.log("排名  组件名              被引用页面数  P 级别");
  console.log("─".repeat(60));
  components.slice(0, 10).forEach((c, i) => {
    console.log(
      `${String(i + 1).padStart(2)}.   ${c.name.padEnd(20)} ${String(c.usageCount).padStart(6)}        ${c.priority}`
    );
  });
  console.log("─".repeat(60));
  console.log(`\n📑 页面分类:`);
  Object.entries(data.totals.byCategory).forEach(([k, v]) => {
    console.log(`   ${k.padEnd(16)} ${String(v).padStart(4)} 个`);
  });
}

main().catch((e) => {
  console.error("❌ 扫描失败:", e);
  process.exit(1);
});

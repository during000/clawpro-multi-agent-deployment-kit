#!/usr/bin/env node
/**
 * 把 data.json 内联进 HTML 模板，生成单文件报告。
 * 输出：docs/design-audit/component-index/report.html （双击即可在浏览器打开）
 */
import { readFile, writeFile } from "node:fs/promises";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const DATA_FILE = join(__dirname, "data.json");
const TEMPLATE_FILE = join(__dirname, "template.html");
const OUTPUT_FILE = join(__dirname, "report.html");

const data = await readFile(DATA_FILE, "utf-8");
const template = await readFile(TEMPLATE_FILE, "utf-8");
const html = template.replace(
  '"__INLINE_DATA__"',
  data.trim()
);
await writeFile(OUTPUT_FILE, html, "utf-8");
console.log("✅ 报告已生成: docs/design-audit/component-index/report.html");
console.log("   双击在浏览器中打开即可查看。");

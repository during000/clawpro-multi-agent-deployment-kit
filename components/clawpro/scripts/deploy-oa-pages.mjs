#!/usr/bin/env node
// =============================================================================
// OA Pages 部署脚本
//
// 将构建产物 dist/public 上传到 OA Pages（https://pages.woa.com）。
//
// 接口：PUT /api/sites/:cname
//   - 鉴权：Header  X-Api-Key: $OA_PAGES_API_KEY
//   - body：{ files: { "相对路径": "内容" } }  增量更新（未列出的文件保持不变）
//   - 文本文件：直接放 UTF-8 字符串；二进制文件：放 base64（服务端按扩展名自动解码）
//   - 单请求体上限 5MB（base64 膨胀 ~33%），超限需拆成多次 PUT
//   - 上传即生效，无需额外发布/激活接口
//
// 用法：
//   OA_PAGES_API_KEY=xxx node scripts/deploy-oa-pages.mjs
//   可选环境变量：
//     OA_PAGES_CNAME    目标域名，默认 openclaw-devspace.pages.woa.com
//     OA_PAGES_DIST     产物目录，默认 dist/public
//     OA_PAGES_DESC     网站描述（≤100 字，仅首批请求携带）
// =============================================================================

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = path.resolve(__dirname, "..");

const HOST = "https://pages.woa.com";
const CNAME = process.env.OA_PAGES_CNAME || "openclaw-devspace.pages.woa.com";
const DIST_DIR = path.resolve(PROJECT_ROOT, process.env.OA_PAGES_DIST || "dist/public");
const DESCRIPTION = process.env.OA_PAGES_DESC || "";
const API_KEY = process.env.OA_PAGES_API_KEY;

// 单请求体上限 5MB，留安全余量
const MAX_BATCH_BYTES = 4.5 * 1024 * 1024;

// 文本扩展名：value 直接放 UTF-8 字符串；其余按二进制做 base64
const TEXT_EXTENSIONS = new Set([
  ".html", ".htm", ".css", ".js", ".mjs", ".cjs", ".json", ".map",
  ".svg", ".txt", ".xml", ".webmanifest", ".md", ".csv",
]);

function fail(msg) {
  console.error(`\x1b[31m[deploy-oa-pages] ${msg}\x1b[0m`);
  process.exit(1);
}

function isTextFile(filePath) {
  return TEXT_EXTENSIONS.has(path.extname(filePath).toLowerCase());
}

/** 递归收集目录下所有文件的绝对路径 */
function walk(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      out.push(...walk(full));
    } else if (entry.isFile()) {
      out.push(full);
    }
  }
  return out;
}

/** 读取单个文件，返回 { relPath, content, bytes } */
function readEntry(absPath) {
  const relPath = path.relative(DIST_DIR, absPath).split(path.sep).join("/");
  let content;
  if (isTextFile(absPath)) {
    content = fs.readFileSync(absPath, "utf-8");
  } else {
    content = fs.readFileSync(absPath).toString("base64");
  }
  // 估算其在 JSON body 中的字节占用（key + value + 引号/转义余量）
  const bytes = Buffer.byteLength(relPath, "utf-8") + Buffer.byteLength(content, "utf-8") + 16;
  return { relPath, content, bytes };
}

/**
 * 将文件列表按 body 上限分批。
 * 单文件本身超过上限的（base64 后 > 5MB）无法通过该接口上传，
 * 会被跳过并收集到 skipped 中（站点上的旧版本因增量更新仍保留）。
 */
function batchEntries(entries) {
  const batches = [];
  const skipped = [];
  let current = [];
  let currentBytes = 0;
  for (const e of entries) {
    if (e.bytes > MAX_BATCH_BYTES) {
      skipped.push(e);
      continue;
    }
    if (current.length > 0 && currentBytes + e.bytes > MAX_BATCH_BYTES) {
      batches.push(current);
      current = [];
      currentBytes = 0;
    }
    current.push(e);
    currentBytes += e.bytes;
  }
  if (current.length > 0) batches.push(current);
  return { batches, skipped };
}

async function putBatch(batch, withDescription) {
  const files = {};
  for (const e of batch) files[e.relPath] = e.content;

  const body = { files };
  if (withDescription && DESCRIPTION) body.description = DESCRIPTION.slice(0, 100);

  const res = await fetch(`${HOST}/api/sites/${encodeURIComponent(CNAME)}`, {
    method: "PUT",
    headers: {
      "X-Api-Key": API_KEY,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  const text = await res.text();
  if (!res.ok) {
    throw new Error(`HTTP ${res.status} ${res.statusText} -> ${text}`);
  }
  return text;
}

async function main() {
  if (!API_KEY) fail("缺少环境变量 OA_PAGES_API_KEY");
  if (!fs.existsSync(DIST_DIR)) fail(`产物目录不存在：${DIST_DIR}（请先执行 pnpm build）`);

  const absFiles = walk(DIST_DIR);
  if (absFiles.length === 0) fail(`产物目录为空：${DIST_DIR}`);

  const entries = absFiles.map(readEntry);
  const totalBytes = entries.reduce((s, e) => s + e.bytes, 0);
  const { batches, skipped } = batchEntries(entries);

  console.log(`[deploy-oa-pages] 目标站点：${CNAME}`);
  console.log(`[deploy-oa-pages] 产物目录：${DIST_DIR}`);
  console.log(
    `[deploy-oa-pages] 文件数：${entries.length}，约 ${(totalBytes / 1024 / 1024).toFixed(2)}MB，分 ${batches.length} 批上传`,
  );
  if (skipped.length > 0) {
    console.warn(
      `\x1b[33m[warn] ${skipped.length} 个文件超过单请求 ${(MAX_BATCH_BYTES / 1024 / 1024).toFixed(1)}MB 上限，无法通过该接口上传，将被跳过：\x1b[0m`,
    );
    for (const e of skipped) {
      console.warn(`\x1b[33m         - ${e.relPath} (${(e.bytes / 1024 / 1024).toFixed(2)}MB)\x1b[0m`);
    }
  }

  for (let i = 0; i < batches.length; i++) {
    const batch = batches[i];
    const batchBytes = batch.reduce((s, e) => s + e.bytes, 0);
    process.stdout.write(
      `  - 第 ${i + 1}/${batches.length} 批：${batch.length} 个文件，约 ${(batchBytes / 1024 / 1024).toFixed(2)}MB ... `,
    );
    const resp = await putBatch(batch, i === 0);
    console.log("OK");
    if (i === batches.length - 1) console.log(`[deploy-oa-pages] 响应：${resp}`);
  }

  console.log(`\x1b[32m[deploy-oa-pages] 部署完成：https://${CNAME}/\x1b[0m`);
  if (skipped.length > 0) {
    console.log(
      `\x1b[33m[deploy-oa-pages] 注意：${skipped.length} 个超大文件未上传（站点保留旧版本）。如需更新这些大文件，请改用 Git 部署方式或压缩后再传。\x1b[0m`,
    );
  }
}

main().catch((err) => fail(err?.stack || String(err)));

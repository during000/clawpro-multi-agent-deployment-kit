#!/usr/bin/env node
/**
 * fetch-skillhub-snapshot.mjs
 *
 * 一次性从 SkillHub 公共 API 拉取所有 10 个分类的技能包数据，
 * 生成本地数据快照（TS 文件）：
 *   client/src/pages/admin/SkillLibrary/publicSkillPackageDataSnapshot.ts
 *
 * 使用方式（在仓库根目录下）：
 *   node scripts/fetch-skillhub-snapshot.mjs
 *
 * 注意：
 *  - 此脚本仅在开发期手动运行，不参与生产构建。
 *  - 每个分类首页 pageSize=100，覆盖 SkillHub 上当前的全部条目；
 *    如果 SkillHub 后续条目超过 100，需要扩展为分页拉取。
 *  - 用 sceneToCategoryId 将 SkillHub 的 scene 字段映射到我们前端的分类 id。
 *  - 写入的文件按 `id` 去重（不同分类查询可能返回重叠数据）。
 */

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const REPO_ROOT = path.resolve(__dirname, '..');
const OUT_FILE = path.join(
  REPO_ROOT,
  'client/src/pages/admin/SkillLibrary/publicSkillPackageDataSnapshot.ts'
);

/** 与前端 PUBLIC_SKILL_PACKAGE_CATEGORIES 对齐（不含 all/favorites） */
const SCENES = [
  { id: 'finance',   scene: 'finance' },
  { id: 'tech',      scene: 'tech' },
  { id: 'design',    scene: 'design' },
  { id: 'marketing', scene: 'marketing' },
  { id: 'legal',     scene: 'legal' },
  { id: 'academic',  scene: 'academic' },
  { id: 'education', scene: 'education' },
  { id: 'hr',        scene: 'hr' },
  { id: 'ecommerce', scene: 'ecommerce' },
];

const API_BASE = 'https://api.skillhub.cn/api/v1/skillsets';

async function fetchScene(scene) {
  const url = `${API_BASE}?page=1&pageSize=100${scene ? `&scene=${scene}` : ''}`;
  console.log(`  → GET ${url}`);
  const res = await fetch(url, {
    headers: { accept: 'application/json', 'user-agent': 'Mozilla/5.0' },
  });
  if (!res.ok) {
    throw new Error(`HTTP ${res.status} for ${url}`);
  }
  const json = await res.json();
  return json.skillSets || [];
}

/**
 * 从 SkillHub 一条 raw 数据里抽出我们用到的字段
 * （丢弃后端内部字段如 createdAt/updatedAt/remark，避免雪藏字段过多）
 */
function pickFields(raw, categoryId) {
  return {
    id: raw.id,
    slug: raw.slug,
    displayName: raw.displayName,
    summary: raw.summary || '',
    scene: raw.scene || '',
    subScene: raw.subScene || '',
    /** 把 SkillHub 的 scene 映射到我们前端的分类 id */
    category: categoryId,
    /** 完整 markdown，详情页用 MDXRenderer 渲染（去掉 frontmatter） */
    content: raw.content || '',
    skillSlugs: Array.isArray(raw.skillSlugs) ? raw.skillSlugs : [],
    skillCount: typeof raw.skillCount === 'number' ? raw.skillCount : (raw.skillSlugs?.length ?? 0),
  };
}

async function main() {
  console.log('开始拉取 SkillHub 全部分类数据...\n');

  // 用 Map 按 id 去重（SkillHub 接口在 scene=undefined 时也可能返回所有数据）
  const byId = new Map();

  for (const { id, scene } of SCENES) {
    console.log(`[${id}] scene=${scene}`);
    try {
      const list = await fetchScene(scene);
      console.log(`  ← 返回 ${list.length} 条`);
      for (const raw of list) {
        // 只保留 scene 真正落在当前分类下的数据，避免分类间数据被打乱
        if (raw.scene && raw.scene !== scene) continue;
        if (!byId.has(raw.id)) {
          byId.set(raw.id, pickFields(raw, id));
        }
      }
    } catch (e) {
      console.error(`  ✗ ${e.message}`);
    }
  }

  const all = Array.from(byId.values()).sort((a, b) => a.id - b.id);
  console.log(`\n去重后共 ${all.length} 个技能包`);

  // 统计每个分类的数量
  const stats = {};
  for (const item of all) {
    stats[item.category] = (stats[item.category] || 0) + 1;
  }
  console.log('分类统计：', stats);

  // 生成 TS 文件
  const banner = `/**
 * SkillHub 公共技能包数据快照
 *
 * 由 \`scripts/fetch-skillhub-snapshot.mjs\` 自动生成，请勿手工编辑。
 * 数据来源：https://api.skillhub.cn/api/v1/skillsets
 * 生成时间：${new Date().toISOString()}
 * 共 ${all.length} 个技能包
 */

import type { PublicSkillPackageRaw } from './publicSkillPackageMockData';

export const SKILLHUB_PACKAGE_SNAPSHOT: PublicSkillPackageRaw[] = ${JSON.stringify(all, null, 2)};
`;

  fs.writeFileSync(OUT_FILE, banner, 'utf8');
  console.log(`\n✓ 写入 ${OUT_FILE}`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});

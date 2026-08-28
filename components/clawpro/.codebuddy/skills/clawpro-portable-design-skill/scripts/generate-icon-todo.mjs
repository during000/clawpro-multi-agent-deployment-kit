#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const packageRoot = path.resolve(__dirname, '..');
const projectRoot = process.cwd();

const args = process.argv.slice(2);
const outputPath = args.find(arg => arg.startsWith('--output='))?.split('=')[1] ||
                   path.resolve(packageRoot, 'references/icon-design-todo.md');

const textExtensions = new Set(['.ts', '.tsx', '.js', '.jsx']);
const ignoredDirs = new Set(['node_modules', '.git', 'dist', 'build', 'coverage', '.codebuddy']);

function walk(entry, files = []) {
  if (!fs.existsSync(entry)) return files;
  const stat = fs.statSync(entry);
  if (stat.isDirectory()) {
    const basename = path.basename(entry);
    if (ignoredDirs.has(basename)) return files;
    for (const child of fs.readdirSync(entry)) {
      walk(path.join(entry, child), files);
    }
    return files;
  }
  if (textExtensions.has(path.extname(entry))) files.push(entry);
  return files;
}

function collectConfirmationItems(files) {
  const items = [];
  const confirmationPattern = /needs-design-confirmation:\s*(.+?)(?:\s*\*\/|\n|$)/g;

  for (const file of files) {
    const content = fs.readFileSync(file, 'utf8');
    const lines = content.split(/\r?\n/);

    lines.forEach((line, index) => {
      const match = line.match(/needs-design-confirmation:\s*(.+?)(?:\s*\*\/|$)/);
      if (match) {
        const description = match[1].trim();
        const relativePath = path.relative(projectRoot, file);

        // 提取槽位名称（从文件名或上下文）
        const slotName = path.basename(file, path.extname(file));

        items.push({
          file: relativePath,
          line: index + 1,
          slot: slotName,
          description: description,
          priority: 'P2', // 默认中优先
          status: 'pending',
        });
      }
    });
  }

  return items;
}

function generateMarkdown(items) {
  const now = new Date().toISOString();
  const totalItems = items.length;

  let markdown = `# Icon Design Confirmation TODO

根据 \`needs-design-confirmation\` 标记自动生成。

**最后更新时间**：${now}
**生成方式**：\`scripts/generate-icon-todo.mjs\`

---

## Summary

- **Total items**: ${totalItems}
- **Status distribution**:
  - Pending: ${items.filter(i => i.status === 'pending').length}
  - In progress: ${items.filter(i => i.status === 'in_progress').length}
  - Completed: ${items.filter(i => i.status === 'completed').length}
- **Last updated**: ${new Date().toLocaleDateString()}

---

## Items

`;

  items.forEach((item, index) => {
    markdown += `### ${index + 1}. ${item.slot} - ${item.description.substring(0, 50)}...\n\n`;
    markdown += `- **位置**: \`${item.file}:${item.line}\`\n`;
    markdown += `- **需要**: ${item.description}\n`;
    markdown += `- **优先级**: ${item.priority}\n`;
    markdown += `- **状态**: ${item.status}\n\n`;
  });

  markdown += `## Statistics

### 按优先级分布

| 优先级 | 数量 |
|-------|-----|
`;

  const priorityDist = {};
  items.forEach(item => {
    priorityDist[item.priority] = (priorityDist[item.priority] || 0) + 1;
  });

  for (const [priority, count] of Object.entries(priorityDist).sort()) {
    markdown += `| ${priority} | ${count} |\n`;
  }

  markdown += `\n### 按状态分布\n\n| 状态 | 数量 |\n|-----|-----|\n`;

  const statusDist = {};
  items.forEach(item => {
    statusDist[item.status] = (statusDist[item.status] || 0) + 1;
  });

  for (const [status, count] of Object.entries(statusDist).sort()) {
    markdown += `| ${status} | ${count} |\n`;
  }

  markdown += `\n---\n\n**最后生成**：${now}\n`;

  return markdown;
}

// 执行生成
const sourceDir = path.resolve(packageRoot, 'portable');
const files = walk(sourceDir);
const items = collectConfirmationItems(files);

if (items.length === 0) {
  console.log('No design confirmation items found.');
  process.exit(0);
}

const markdown = generateMarkdown(items);

fs.writeFileSync(outputPath, markdown, 'utf8');
console.log(`Generated icon design TODO list: ${outputPath}`);
console.log(`Found ${items.length} items to be confirmed.`);

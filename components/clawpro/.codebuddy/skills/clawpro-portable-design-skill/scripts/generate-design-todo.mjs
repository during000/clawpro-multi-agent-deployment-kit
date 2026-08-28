#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const packageRoot = path.resolve(__dirname, '..');

// 读取 C1 生成的原始数据
const todoRawPath = path.resolve(packageRoot, 'references/icon-design-todo-raw.json');

let confirmationItems = [];
if (fs.existsSync(todoRawPath)) {
  try {
    const data = JSON.parse(fs.readFileSync(todoRawPath, 'utf8'));
    confirmationItems = data.items || [];
  } catch (error) {
    console.error('Failed to read icon-design-todo-raw.json:', error.message);
  }
}

// 按槽位分组
const bySlot = new Map();
const byPriority = new Map();
const byStatus = new Map();

for (const item of confirmationItems) {
  // 提取槽位信息
  const slotMatch = item.details.match(/\[([^\]]+)\]/);
  const slot = slotMatch ? slotMatch[1] : 'unspecified';

  // 提取优先级
  const priorityMatch = item.details.match(/P[0123]/);
  const priority = priorityMatch ? priorityMatch[0] : 'P2';

  // 提取状态
  const statusMatch = item.details.match(/(未启动|进行中|已完成)/);
  const status = statusMatch ? statusMatch[1] : '未启动';

  // 按槽位分组
  if (!bySlot.has(slot)) bySlot.set(slot, []);
  bySlot.get(slot).push(item);

  // 按优先级分组
  if (!byPriority.has(priority)) byPriority.set(priority, []);
  byPriority.get(priority).push(item);

  // 按状态分组
  if (!byStatus.has(status)) byStatus.set(status, []);
  byStatus.get(status).push(item);
}

// 生成 Markdown 清单
const now = new Date().toISOString().split('T')[0];
const markdown = `# Icon Design Confirmation TODO

根据 \`needs-design-confirmation\` 标记自动生成。  
最后更新时间：${now}

## Summary

- **Total items**: ${confirmationItems.length}
- **Status**: 
  - 未启动: ${byStatus.get('未启动')?.length || 0}
  - 进行中: ${byStatus.get('进行中')?.length || 0}
  - 已完成: ${byStatus.get('已完成')?.length || 0}
- **Last reviewed**: ${now}

---

## Items by Slot

${Array.from(bySlot.entries())
  .sort((a, b) => b[1].length - a[1].length)
  .map(([slot, items]) => `
### ${slot} (${items.length} items)

${items.map(item => `
- **File**: [\`${item.file}:${item.line}\`](../${item.file})
- **Details**: ${item.details}
- **Status**: 未启动
- **Priority**: P2
- **Assigned to**: [待分配]
`).join('\n')}
`).join('\n')}

---

## Items by Priority

${Array.from(byPriority.entries())
  .sort((a, b) => a[0].localeCompare(b[0]))
  .map(([priority, items]) => `
### ${priority} - ${items.length} items

${items.slice(0, 5).map(item => `- [\`${item.file}:${item.line}\`](../${item.file})`).join('\n')}
${items.length > 5 ? `... and ${items.length - 5} more` : ''}
`).join('\n')}

---

## Statistics

### By Slot
\`\`\`
${Array.from(bySlot.entries())
  .sort((a, b) => b[1].length - a[1].length)
  .map(([slot, items]) => `${slot}: ${items.length}`)
  .join('\n')}
\`\`\`

### By Priority
\`\`\`
${Array.from(byPriority.entries())
  .sort((a, b) => a[0].localeCompare(b[0]))
  .map(([priority, items]) => `${priority}: ${items.length}`)
  .join('\n')}
\`\`\`

### By Status
\`\`\`
${Array.from(byStatus.entries())
  .map(([status, items]) => `${status}: ${items.length}`)
  .join('\n')}
\`\`\`

---

## How to Use This List

1. **Review**: Go through each item and assess design requirements
2. **Prioritize**: Update priority based on business needs
3. **Assign**: Assign items to design team members
4. **Update**: Change status as work progresses (未启动 → 进行中 → 已完成)
5. **Link**: Reference actual design specs when ready

---

**Generated**: ${now}  
**Source**: \`scripts/check-design-usage.mjs --collect-confirmations\`  
**Auto-update**: Run \`npm run design:check\` to refresh
`;

// 写入 Markdown 文件
const outputPath = path.resolve(packageRoot, 'references/icon-design-todo.md');
fs.writeFileSync(outputPath, markdown);

console.log(`Generated icon-design-todo.md with ${confirmationItems.length} items`);
console.log(`Output: ${outputPath}`);

process.exit(0);

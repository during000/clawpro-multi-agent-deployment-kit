# 阶段 C 运维指南

## 快速开始

### 1. 运行 C1 检查
```bash
cd /Users/addietang/Documents/cvm/openclaw-enterprise/.codebuddy/skills/clawpro-portable-design-skill

# 基础检查（显示警告）
node scripts/check-design-usage.mjs portable

# 严格模式（违规时退出码为 1）
node scripts/check-design-usage.mjs portable --strict

# 详细输出
node scripts/check-design-usage.mjs portable --verbose
```

### 2. 生成设计确认清单
```bash
# 前提：已运行过 C1 脚本，生成了 icon-design-todo-raw.json
node scripts/generate-design-todo.mjs

# 输出：references/icon-design-todo.md
```

### 3. 生成组件映射表
```bash
# 自动扫描 portable 目录和 component-specs，生成映射表
node scripts/generate-component-mapping.mjs

# 输出：references/component-mapping.md
```

### 4. 验证 MANIFEST
```bash
# 检查所有 MANIFEST.json 中的文件是否存在
node scripts/verify-manifest.mjs
```

---

## 违规处理流程

### 步骤 1：识别违规类型

C1 检查会生成以下类型的违规：

| 类型 | 优先级 | 处理者 | 预期时间 |
|------|--------|--------|---------|
| CSS 硬编码色 | P0 | 工程师 | 1-2 天 |
| Tailwind 色 | P0 | 工程师 | 1-2 天 |
| 大圆角 | P1 | 工程师 | 2-3 天 |
| inline SVG 标记 | P1 | 工程师/设计师 | 3-5 天 |
| 禁 lucide 槽位 | P0 | 工程师 | 1 天 |
| icon-registry 破损 | P1 | 工程师 | 1 天 |
| variant 不同步 | P2 | 工程师 | 2-3 天 |

### 步骤 2：修复违规

对于 **CSS 硬编码色** 的示例修复：

**违规代码**（portable/css/status-tag.css）:
```css
.cp-status-tag--blue { color: #1447E6; }  /* 硬编码色 */
```

**修复方式 1**：使用 CSS 变量
```css
:root {
  --cp-status-color-blue: #1447E6;
}

.cp-status-tag--blue { color: var(--cp-status-color-blue); }
```

**修复方式 2**：使用现有 token
```css
.cp-status-tag--blue { color: var(--cp-brand-blue); }
```

### 步骤 3：验证修复

修复后重新运行检查：
```bash
node scripts/check-design-usage.mjs portable --strict

# 如果通过，应该看到：
# ✓ CSS hard-coded colors: PASS (0 violations)
```

---

## 常见问题

### Q1: 为什么 CSS 变量定义中的颜色不被标记为违规？

**A**: 这是设计决定。CSS 变量定义块（`:root {}` 中的 `--variable: #color;`）是**令牌层**，而不是样式规则层。脚本只标记**样式规则中**的硬编码颜色。

示例：
```css
/* 这是合法的（令牌定义） */
:root {
  --cp-brand-blue: #1447E6;  /* ✓ 允许 */
}

/* 这是违规的（样式规则） */
.button { color: #1447E6; }  /* ✗ 违规 */
```

### Q2: 我可以忽略某些违规吗？

**A**: 可以，使用允许标记：

```tsx
// 允许 inline SVG
<svg className="my-icon" /* allow-inline-svg */>
  <path d="..." />
</svg>

// 允许硬编码色（仅在必要时）
.legacy-component {
  color: #FF0000; /* allow-hardcoded-color */
}
```

**但请注意**：使用 `allow-*` 标记应该是**例外**，而不是常规。每个标记都应该有充分的理由。

### Q3: inline SVG 标记警告是什么意思？

**A**: 脚本发现了内联 SVG（`<svg>...</svg>`）定义，但没有看到允许标记。

**场景 1**：这是 lucide-react 包装器（应该标记）
```tsx
// allow-design-legacy: portable-lucide-svg-inline
export function AlertIcon() {
  return (
    <svg>
      <path d="..." />
    </svg>
  );
}
```

**场景 2**：这是临时 placeholder（应该改用 lucide-react）
```tsx
// ❌ 不推荐
<svg><circle cx="12" cy="12" r="10" /></svg>

// ✅ 推荐
import { AlertCircle } from 'lucide-react';
<AlertCircle />
```

### Q4: 如何处理 "禁 lucide 槽位" 错误？

**A**: 这些槽位不允许使用 lucide-react，应该改用 resource-skill-map.json：

```tsx
// ❌ 违规：number-card 是禁用槽位
import { Info } from 'lucide-react';
function NumberCard() {
  return <NumberCard icon={<Info />} />;
}

// ✅ 正确：使用资源映射
import { getNumberCardIcon } from './resource-skill-map';
function NumberCard() {
  return <NumberCard icon={getNumberCardIcon('info')} />;
}
```

### Q5: 多 variant 同步检查失败了怎么办？

**A**: 这意味着同一个组件的不同 variant 版本中的 icon 映射不一致。

示例问题：
- alert.tsx 中 alert-info 有 icon-a
- alert/alert.tsx 中 alert-info 没有 icon-a

**解决办法**：
1. 确认所有 variant 版本的意图是否相同
2. 同步更新所有版本，使 icon 映射一致
3. 参考 component-mapping.md 中的版本列表

---

## 集成到 CI/CD

### GitHub Actions 示例

```yaml
# .github/workflows/design-check.yml
name: Design System Check

on: [push, pull_request]

jobs:
  design-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Verify MANIFEST
        run: |
          cd .codebuddy/skills/clawpro-portable-design-skill
          node scripts/verify-manifest.mjs
      
      - name: Check Design Usage (Strict)
        run: |
          cd .codebuddy/skills/clawpro-portable-design-skill
          node scripts/check-design-usage.mjs portable --strict
      
      - name: Generate Reports
        run: |
          cd .codebuddy/skills/clawpro-portable-design-skill
          node scripts/generate-design-todo.mjs
          node scripts/generate-component-mapping.mjs
      
      - name: Comment PR with Results
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const todo = fs.readFileSync('.codebuddy/skills/clawpro-portable-design-skill/references/icon-design-todo.md', 'utf8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## Design System Report\n\n${todo.substring(0, 500)}...`
            });
```

### Pre-commit Hook 示例

```bash
#!/bin/bash
# .git/hooks/pre-commit

cd .codebuddy/skills/clawpro-portable-design-skill

# 快速检查（只检查改动的文件）
CHANGED_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -E '(portable|component-specs)' | head -20)

if [ -n "$CHANGED_FILES" ]; then
  echo "Running design check on changed files..."
  node scripts/check-design-usage.mjs $CHANGED_FILES
  
  if [ $? -ne 0 ]; then
    echo "Design check failed. Fix issues and try again."
    exit 1
  fi
fi

exit 0
```

---

## 仪表板和监控

### 关键指标

运行此脚本查看趋势：

```bash
#!/bin/bash
# scripts/design-metrics.sh

cd .codebuddy/skills/clawpro-portable-design-skill

echo "=== Design System Health Metrics ==="
echo "Generated: $(date)"
echo

# 运行检查
node scripts/check-design-usage.mjs portable > /tmp/design-report.txt 2>&1

# 提取指标
TOTAL_VIOLATIONS=$(grep "Total violations:" /tmp/design-report.txt | awk '{print $3}')
PASSED=$(grep "PASSED:" /tmp/design-report.txt | awk '{print $2}')
FAILED=$(grep "FAILED:" /tmp/design-report.txt | awk '{print $2}')

echo "Violations: $TOTAL_VIOLATIONS"
echo "Checks Passed: $PASSED / 7"
echo "Checks Failed: $FAILED / 7"
echo
echo "Trend: $([ $TOTAL_VIOLATIONS -lt 300 ] && echo '✓ Improving' || echo '✗ Degrading')"
```

---

## 故障排除

### 问题 1：脚本找不到 portable 目录

**解决**：确保从正确的工作目录运行
```bash
# 错误
node scripts/check-design-usage.mjs

# 正确
cd /Users/addietang/Documents/cvm/openclaw-enterprise/.codebuddy/skills/clawpro-portable-design-skill
node scripts/check-design-usage.mjs portable
```

### 问题 2：生成的清单为空

**解决**：确保先运行了 C1 脚本
```bash
# 步骤 1：生成原始数据
node scripts/check-design-usage.mjs portable

# 步骤 2：生成清单（需要步骤 1 的输出）
node scripts/generate-design-todo.mjs
```

### 问题 3：组件映射表不完整

**解决**：检查 MANIFEST.json 是否更新
```bash
# 验证 MANIFEST
node scripts/verify-manifest.mjs

# 如果 componentSpecs 不完整，手工更新 MANIFEST.json
```

---

## 定期维护

### 每周任务
- [ ] 运行 `check-design-usage.mjs` 查看违规趋势
- [ ] 审查 icon-design-todo.md 中的待确认项

### 每月任务
- [ ] 更新 component-mapping.md（`generate-component-mapping.mjs`）
- [ ] 验证 MANIFEST.json（`verify-manifest.mjs`）
- [ ] 检查 CI/CD 中的失败记录

### 季度任务
- [ ] 运行完整的 A3 审计
- [ ] 评估新增的违规模式
- [ ] 更新 SKILL.md 文档

---

**最后更新**: 2026-06-26  
**维护人员**: [Engineering Lead]  
**联系方式**: [team-email@company.com]

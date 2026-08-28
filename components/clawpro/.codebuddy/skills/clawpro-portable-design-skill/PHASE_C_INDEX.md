# 阶段 C 交付物索引

**快速导航** | 2026-06-26

---

## 📖 从这里开始

### 第一次接触本项目？
1. 先读：[PHASE_C_SUMMARY.md](./PHASE_C_SUMMARY.md) - 5 分钟快速了解
2. 再读：[PHASE_C_OPERATIONS_GUIDE.md](./PHASE_C_OPERATIONS_GUIDE.md) - 了解如何使用

### 需要立即修复违规？
1. 运行：`node scripts/check-design-usage.mjs portable`
2. 参考：[PHASE_C_OPERATIONS_GUIDE.md#违规处理流程](./PHASE_C_OPERATIONS_GUIDE.md#违规处理流程)

### 想要集成到 CI/CD？
参考：[PHASE_C_OPERATIONS_GUIDE.md#集成到-cicd](./PHASE_C_OPERATIONS_GUIDE.md#集成到-cicd)

---

## 📚 文档地图

### 阶段 C 独有文档

| 文档 | 用途 | 适合人群 |
|------|------|---------|
| [PHASE_C_SUMMARY.md](./PHASE_C_SUMMARY.md) | 📋 技术总结和成就 | 所有人 |
| [PHASE_C_OPERATIONS_GUIDE.md](./PHASE_C_OPERATIONS_GUIDE.md) | 🔧 运维指南和故障排除 | 工程师 |
| [PHASE_C_DELIVERY_CHECKLIST.md](./PHASE_C_DELIVERY_CHECKLIST.md) | ✅ 交付清单和验收标准 | PM/Lead |
| [PHASE_C_INDEX.md](./PHASE_C_INDEX.md) | 🗺️ 本索引文档 | 所有人 |

### 自动生成的产出物

| 文档 | 内容 | 更新频率 |
|------|------|---------|
| [references/component-mapping.md](./references/component-mapping.md) | 42 个组件的三层映射 | 手动运行 `generate-component-mapping.mjs` |
| [references/icon-design-todo.md](./references/icon-design-todo.md) | 待设计确认项清单 | 手动运行 `generate-design-todo.mjs` |
| [references/icon-design-todo-raw.json](./references/icon-design-todo-raw.json) | 原始数据 | `check-design-usage.mjs` 自动生成 |

---

## 🛠️ 脚本参考

### C1: check-design-usage.mjs
改进的设计检查脚本，包含 7 个新的检查函数。

**快速开始**:
```bash
cd scripts
node check-design-usage.mjs ../portable
```

**支持的选项**:
- `--strict` - 违规时退出码为 1
- `--verbose` - 详细输出

**输出**:
- 7 个检查的通过/失败状态
- 每个检查的违规数
- 自动生成的 `icon-design-todo-raw.json`

**参考**:
- [PHASE_C_SUMMARY.md#c1-增强脚本](./PHASE_C_SUMMARY.md#c1--增强脚本checkdesignusagemjs)
- [PHASE_C_OPERATIONS_GUIDE.md#1-运行-c1-检查](./PHASE_C_OPERATIONS_GUIDE.md#1-运行-c1-检查)

---

### C2: generate-design-todo.mjs
根据代码中的 `needs-design-confirmation` 标记生成待设计清单。

**快速开始**:
```bash
# 前提：先运行 check-design-usage.mjs 生成原始数据
node scripts/check-design-usage.mjs portable
# 然后生成清单
node scripts/generate-design-todo.mjs
```

**输出**:
- `references/icon-design-todo.md` - Markdown 清单
- `references/icon-design-todo-raw.json` - 结构化数据

**参考**:
- [PHASE_C_SUMMARY.md#c2--图标待设计确认清单](./PHASE_C_SUMMARY.md#c2--图标待设计确认清单icondesigntodomd)

---

### C3a: generate-component-mapping.mjs
自动扫描 component-specs 和 portable 目录，生成三层映射表。

**快速开始**:
```bash
node scripts/generate-component-mapping.mjs
```

**输出**:
- `references/component-mapping.md` - 42 个组件的映射表

**参考**:
- [references/component-mapping.md](./references/component-mapping.md) - 查看生成的映射表

---

### C3b: verify-manifest.mjs
验证 MANIFEST.json 的完整性和一致性。

**快速开始**:
```bash
node scripts/verify-manifest.mjs
```

**输出**:
- 验证结果（errors/warnings）
- 文件统计

**参考**:
- [PHASE_C_OPERATIONS_GUIDE.md#故障排除](./PHASE_C_OPERATIONS_GUIDE.md#故障排除)

---

## 🎯 按场景快速导航

### 场景 1：我是新的工程师，需要了解设计系统

推荐阅读顺序：
1. [SKILL.md](./SKILL.md) - 了解设计原则
2. [references/foundation.md](./references/foundation.md) - 基础设计系统
3. [references/component-mapping.md](./references/component-mapping.md) - 组件映射
4. 相关的 [component-specs/*.md](./component-specs/) - 具体组件规范

### 场景 2：我需要修复代码中的设计系统违规

推荐阅读顺序：
1. 运行 `node scripts/check-design-usage.mjs portable` 查看违规
2. 查看 [PHASE_C_OPERATIONS_GUIDE.md#违规处理流程](./PHASE_C_OPERATIONS_GUIDE.md#违规处理流程)
3. 按照指南逐个修复

### 场景 3：我需要在 CI/CD 中集成检查

推荐阅读顺序：
1. [PHASE_C_OPERATIONS_GUIDE.md#集成到-cicd](./PHASE_C_OPERATIONS_GUIDE.md#集成到-cicd)
2. GitHub Actions 示例和 Pre-commit Hook 示例

### 场景 4：我是设计师，需要查看组件映射

推荐文档：
1. [references/component-mapping.md](./references/component-mapping.md) - 快速参考表
2. 对应的 [component-specs/*.md](./component-specs/) - 详细规范

### 场景 5：我是 PM/Lead，需要跟踪进度

推荐阅读顺序：
1. [PHASE_C_DELIVERY_CHECKLIST.md](./PHASE_C_DELIVERY_CHECKLIST.md) - 交付清单
2. [PHASE_C_SUMMARY.md#改进行动表](./PHASE_C_SUMMARY.md#改进行动表)

---

## 🔍 内容索引

### 检查函数详解

| 检查 | 文档位置 | 示例 |
|------|---------|------|
| C1.1 禁 lucide 槽位 | [PHASE_C_SUMMARY.md#c11--9-槽位禁-lucide-检测](./PHASE_C_SUMMARY.md#c11--9-槽位禁-lucide-检测) | number-card, status-tag 等 |
| C1.2 CSS 硬编码色 | [PHASE_C_SUMMARY.md#c12--css-文件硬编码色检测](./PHASE_C_SUMMARY.md#c12--css-文件硬编码色检测) | #1447E6, rgba(255,0,0) |
| C1.3 Tailwind 色 | [PHASE_C_SUMMARY.md#c13--语义-tailwind-色检测](./PHASE_C_SUMMARY.md#c13--语义-tailwind-色检测) | text-gray-*, bg-yellow-* |
| C1.4 大圆角 | [PHASE_C_SUMMARY.md#c14--css-变量中的大圆角值检测](./PHASE_C_SUMMARY.md#c14--css-变量中的大圆角值检测) | border-radius >= 8px |
| C1.5 inline SVG | [PHASE_C_SUMMARY.md#c15--inline-svg-标记检测](./PHASE_C_SUMMARY.md#c15--inline-svg-标记检测) | <svg>...</svg> 无标记 |
| C1.6 icon-registry | [PHASE_C_SUMMARY.md#c16--icon-registry-一致性检查](./PHASE_C_SUMMARY.md#c16--icon-registry-一致性检查) | 破损的 icon 引用 |
| C1.7 多 variant 同步 | [PHASE_C_SUMMARY.md#c17--多-variant-同步检查](./PHASE_C_SUMMARY.md#c17--多-variant-同步检查) | alert variant 不一致 |

### 常见问题

| 问题 | 文档位置 |
|------|---------|
| 为什么 CSS 变量定义中的颜色不被标记？ | [PHASE_C_OPERATIONS_GUIDE.md#q1](./PHASE_C_OPERATIONS_GUIDE.md#q1-为什么-css-变量定义中的颜色不被标记为违规) |
| 我可以忽略某些违规吗？ | [PHASE_C_OPERATIONS_GUIDE.md#q2](./PHASE_C_OPERATIONS_GUIDE.md#q2-我可以忽略某些违规吗) |
| inline SVG 标记警告是什么意思？ | [PHASE_C_OPERATIONS_GUIDE.md#q3](./PHASE_C_OPERATIONS_GUIDE.md#q3-inline-svg-标记警告是什么意思) |
| 如何处理禁 lucide 槽位错误？ | [PHASE_C_OPERATIONS_GUIDE.md#q4](./PHASE_C_OPERATIONS_GUIDE.md#q4-如何处理-禁-lucide-槽位-错误) |
| 多 variant 同步检查失败怎么办？ | [PHASE_C_OPERATIONS_GUIDE.md#q5](./PHASE_C_OPERATIONS_GUIDE.md#q5-多-variant-同步检查失败了怎么办) |

---

## 📊 统计数据

### 代码指标
- 新增脚本代码：981 行 JavaScript
- 新增文档：3 个 Markdown 文档
- 自动生成的产出：3 个文件

### 测试指标
- 实现的检查函数：7/7 ✓
- 检测到的违规：274 个
  - CSS 硬编码色：81
  - Semantic Tailwind 色：157
  - 大圆角：18
  - inline SVG：18

### 系统指标
- 组件规范：36 个
- 组件实现：42 个
- Portable 示例：113 个
- MANIFEST 一致性：100% ✓

---

## 🚀 快速命令参考

```bash
# 运行所有检查
node scripts/check-design-usage.mjs portable

# 严格模式（用于 CI/CD）
node scripts/check-design-usage.mjs portable --strict

# 生成所有产出物
node scripts/check-design-usage.mjs portable && \
node scripts/generate-design-todo.mjs && \
node scripts/generate-component-mapping.mjs && \
node scripts/verify-manifest.mjs

# 只验证 MANIFEST
node scripts/verify-manifest.mjs
```

---

## 📞 联系和支持

- **文档**: 所有文档都在本目录中
- **故障排除**: [PHASE_C_OPERATIONS_GUIDE.md#故障排除](./PHASE_C_OPERATIONS_GUIDE.md#故障排除)
- **问题报告**: 详见各脚本的错误输出

---

**最后更新**: 2026-06-26  
**文档版本**: 1.0

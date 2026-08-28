# ClawPro Portable Design Pack

> 一套给产品前端、设计同事、CodeBuddy 与通用 AI 共同使用的可交付设计包。
> 目标不是让对方直接复用当前 demo 仓代码，而是让对方在宿主仓里也能低摩擦还原 ClawPro 的视觉与交互。

## 1. 这包东西解决什么问题

- 当前 `clawpro` demo / 设计刷新分支与产品前端主分支在组件、规范、实现方式上差距较大。
- 只交 demo 仓代码或只交一份设计规范，不足以支持产品前端在一周内稳定换皮。
- 这份 portable design pack 的重点是：补足跨仓可执行性。

具体来说，它回答四件事：

1. 各端页面最终应该长什么样。
2. 高风险组件应该用什么视觉标准。
3. 当前 demo 仓怎么写。
4. 如果宿主仓没有这套组件，最低限度该怎么 fallback 还原。

## 2. 推荐使用顺序

### 2.1 人类协作者

1. 先读 `docs/STRUCTURE.md`
2. 再读 `references/foundation.md`
3. 按端继续读 `references/admin.md` / `references/tenant.md` / `references/landing.md`
4. 做页面时读 `references/page-recipes.md`
5. 做高风险组件时读 `component-specs/*.md`
6. 遇到列表顶部搜索 / 筛选 / 刷新工具条时，优先读 `component-specs/search-filter-bar.md`
7. 遇到日期筛选或日期字段时，优先读 `component-specs/date-picker.md`
8. 遇到搜索型对象选择器时，优先读 `component-specs/combobox.md`
9. 遇到表格多选 / 跨页选择 / 批量操作时，优先读 `component-specs/batch-actions-bar.md`
10. 实施前后对照 `references/migration-map.md` 和 `qa/*.md`

### 2.2 AI 协作者

1. 先读 `SKILL.md`
2. 严格按 `SKILL.md` 指定的读取顺序加载文档
3. 涉及组件时优先读对应 `component-specs/*.md`
4. 当 demo 仓实现无法直接迁移时，必须读取 `Portable Fallback` 小节，不得只照抄 demo 仓组件名

## 3. 交付边界

这份交付包优先服务下周一换皮，不以“完整组件库化”作为当前目标。

当前阶段重点：

- 可执行的规范入口
- 端级规则
- 高风险组件 portable spec
- 迁移映射
- QA checklist

当前阶段不承诺：

- 可直接发布的 npm 组件库
- 宿主仓零改造接入
- 所有历史页面已全部收敛到新规范

## 4. 目录速览

- `references/`: 稳定规则，定义各端基线与页面 recipe
- `component-specs/`: 高风险组件的 portable spec
- `portable/`: 脱离 demo 仓后的最小 HTML/CSS/React fallback
- `tokens/`: 可单独抽取的设计常量
- `qa/`: 页面与组件验收清单
- `assets/`: icon registry 示例和可移交的资产元数据
- `scripts/`: 设计使用检查脚本
- `docs/`: 交付说明 / 索引 / 清单等过程与移交文档（见下）
- `docs/HANDOFF.md`: 交付给前端 / 协作者时怎么整体移交
- `docs/INDEX.md`: 这整包文件的快速索引
- `docs/DELIVERY-CHECKLIST.md`: 周一交付前的最后核对清单

## 5. 周一前的最小必读文件

- `references/foundation.md`
- `references/admin.md`
- `references/tenant.md`
- `references/migration-map.md`
- `component-specs/table.md`
- `component-specs/card-surface.md`
- `component-specs/button.md`
- `component-specs/empty-state.md`
- `qa/admin-checklist.md`
- `qa/tenant-checklist.md`
- `component-specs/search-filter-bar.md`
- `component-specs/date-picker.md`
- `component-specs/combobox.md`
- `component-specs/batch-actions-bar.md`
- `docs/DEVELOPER-USAGE.md`
- `docs/DESIGN-AUDIT-PLAYBOOK.md`

如果是要把这包东西正式发给产品前端，包内已包含以下交付说明，建议提醒接收方一起阅读：

- `docs/HANDOFF.md`
- `docs/DEVELOPER-USAGE.md`
- `docs/DESIGN-AUDIT-PLAYBOOK.md`
- `docs/INDEX.md`
- `references/conflict-log.md`

## 6.1 如果要直接打包成 zip

可运行：

```bash
bash .codebuddy/skills/clawpro-portable-design-skill/scripts/package-portable-skill.sh
```

默认输出到 `dist/` 目录。

交付前可再运行：

```bash
node .codebuddy/skills/clawpro-portable-design-skill/scripts/verify-portable-skill.mjs
```

## 7. 给产品前端的简版口径

可以直接把下面这段发给产品前端：

```text
这不是一份只能在 demo 仓里用的设计规范，而是一套可移植交付包。

请先读：
1. README.md
2. references/foundation.md
3. references/admin.md 或 references/tenant.md
4. references/migration-map.md

涉及表格、卡片、按钮、空状态、页头、筛选工具条、日期选择器、SearchableSelect（旧名 Combobox，已并入 Select 的 searchable 模式）、批量操作条、Popover / Dropdown、导航、加载、图表、上传 / 文件浏览时，请优先看 `component-specs/`。
如果宿主仓没有我们当前的组件，请按每个 spec 里的 Portable Fallback 还原，不要自由发挥。
```

## 8. 后续演进方向

如果这套交付包在周一换皮阶段验证有效，后续可以继续演进成：

- CodeBuddy 内部正式 skill
- Codex / Cursor 可直接读取的通用 skill 包
- 更完整的组件 portable spec 库
- 可安装组件库或 token package

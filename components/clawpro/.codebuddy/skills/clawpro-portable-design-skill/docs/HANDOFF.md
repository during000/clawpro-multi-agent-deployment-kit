# Handoff

## 1. 这份交付包怎么交

推荐把整个 `.codebuddy/skills/clawpro-portable-design-skill/` 文件夹作为一个完整交付包移交。

原因：

- 目录已经按“规则 / 组件 / fallback / token / QA / assets / script”分层。
- 产品前端、人类设计协作者、AI 工具都可以直接读取这一整包。
- 如果拆文件零散发送，接收方很容易漏读 `migration-map.md`、`component-specs/` 和 `qa/`。

## 2. 推荐交付形式

### 方案 A：直接交整个文件夹

适合：内部协作、同仓同机、CodeBuddy / Codex / Cursor 直接读取。

交付内容：

- 整个 `.codebuddy/skills/clawpro-portable-design-skill/`
- 补充说明：让接收方先读 `README.md`

### 方案 B：压缩成 zip 再交

适合：发给外部仓、发 IM、邮件或作为附件。

推荐压缩名：

- `clawpro-portable-design-skill-2026-06-06.zip`

### 方案 C：配一页简版交付说明

适合：需要和产品前端同步“怎么用这包东西”。

建议配套这几个文件一起给：

- `README.md`
- `HANDOFF.md`
- `DEVELOPER-USAGE.md`
- `DESIGN-AUDIT-PLAYBOOK.md`
- `DELIVERY-CHECKLIST.md`
- `INDEX.md`
- `references/migration-map.md`
- `qa/admin-checklist.md`
- `qa/tenant-checklist.md`

## 3. 接收方第一步该看什么

### 人类前端

1. `README.md`
2. `DEVELOPER-USAGE.md`
3. `references/foundation.md`
4. `references/admin.md` 或 `references/tenant.md`
5. `references/migration-map.md`
6. 相关 `component-specs/*.md`

### AI 协作者

1. `SKILL.md`
2. `STATUS.md`
3. `references/foundation.md`
4. 按端加载 `admin.md` / `tenant.md` / `landing.md`
5. 相关 `component-specs/*.md`

## 4. 周一现场交付时建议怎么说

可以直接用下面这段：

```text
这次我们交付的不是一份只能在 demo 仓里参考的设计规范，而是一套 portable design pack。

你们不需要先接入我们完整组件体系，也不需要直接拷 demo 仓代码。
请先按 README 和 migration-map 确认页面骨架、组件语义和 fallback 方案。
如果宿主仓没有对应组件，就按各个 component spec 里的 Portable Fallback 还原。
```

## 5. 当前适合直接交付的部分

- Admin 基础规则
- 已完成的高 / 中风险组件 spec：表格、卡片、按钮、空状态、页头、弹窗 / 抽屉、Popover / Dropdown、Tabs / Segment、表单控件、状态标签、分页、搜索筛选条、日期选择器、SearchableSelect（旧名 Combobox，已并入 Select）、批量操作条、导航、加载、图表、上传 / 文件浏览
- 最小 React / HTML/CSS fallback 与各 spec 内联 fallback
- migration map
- QA checklist
- `DEVELOPER-USAGE.md` 开发换皮说明
- `DESIGN-AUDIT-PLAYBOOK.md` 统一设计审查与换皮验收流程
- token 最小集

## 6. 当前需要设计侧继续确认的部分

这些内容可以先交，但应标注为“设计待确认 / 可后续回写”：

- Tenant 最终背景参数
- Tenant 顶部导航最终规格
- Tenant 表单控件圆角分流
- Tenant Tabs / Segment 分流；Tenant Text Switch 已删除，不进入周一交付包
- Landing Hero 与区块节奏

新增的待确认或已确认条目，统一回写到 `references/conflict-log.md`，由它作为唯一活账本。

## 7. 周一交付后的维护方式

- 新增组件 spec 时，优先补到 `component-specs/`
- 新增可移植示例时，补到 `portable/`
- 新的冲突裁决写到 `references/conflict-log.md`
- 新开对话续接，优先看 `STATUS.md`

## 8. 最后打包命令

```bash
node .codebuddy/skills/clawpro-portable-design-skill/scripts/verify-portable-skill.mjs
bash .codebuddy/skills/clawpro-portable-design-skill/scripts/package-portable-skill.sh
```

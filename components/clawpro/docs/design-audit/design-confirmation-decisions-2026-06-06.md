# ClawPro 新皮肤 Skill 设计确认结果

> 日期：2026-06-06  
> 来源：miekoyychen 对 `p0-conflict-confirmation-page-2026-06-06.html` 的选择结果  
> 用途：作为后续收口 `clawpro-portable-design-skill` 的确认依据。

## 1. P0 / P1 / 全局确认

| ID | 选择 | 备注 |
|---|---|---|
| C-001 | A |  |
| C-002 | A |  |
| C-003 | A |  |
| C-004 | A |  |
| C-005 | A |  |
| C-006 | A |  |
| C-007 | A |  |
| C-008 | A |  |
| C-009 | B | 表格整体 12px，包括单元格正文。 |
| C-010 | A |  |
| C-011 | C | 周一交 portable pack；周一后收敛根目录 `SKILL.md` 为短路由入口。 |
| C-012 | A | Admin 默认卡片无投影；Tenant 业务卡默认有 `--shadow-tenant-card`。 |
| C-013 | A | 描边统一蓝灰 token：`--border = #EAEEF4`；可勾选控件用 `--border-control = #C8CFDA`。 |
| C-015 | A |  |
| C-016 | A | 暂选，不是很确定，后期进一步确认。 |
| C-017-A | A | 仅 token 定义文件和基础 token 表允许必要 hex。 |
| C-017-B | A | 组件 spec 正文优先 token / CSS variable，不直接散写业务色 hex。 |
| C-017-C | A | portable fallback 必须使用 `--cp-*` CSS variable 或 token class。 |
| C-017-D | A | 状态色不允许业务侧直接写 hex 或 Tailwind 色阶。 |
| C-017-E | A | 资产内部颜色不算业务样式硬编码，但需进 registry 或由设计确认。 |
| C-017-F | A | Landing 特殊视觉需先进入 landing token / asset spec / registry。 |
| C-017-G | A | 历史页面 hex 是存量债务，不是新增例外。 |
| C-018 | C | 列表页按页面密度分流；普通列表可同卡，复杂筛选区独立。 |
| C-019 | C | 暂定，后续进一步确认；周一前只确认基础结构。 |

## 2. Tenant 专项确认

| ID | 选择 | 备注 |
|---|---|---|
| T-01 | A |  |
| T-02 | A |  |
| T-03 | C | TopNav 暂按当前实现，不作为周一交付阻塞。 |
| T-04 | C | 根据页面密度选择，暂不强制唯一分流。 |
| T-05 | B | 筛选 / 搜索胶囊，普通表单 4px，DatePicker 跟随场景。 |
| T-06 | C | 沿用 Admin 空态体系，只在文案语气上更引导。 |
| T-07 | C | Typography 不做强制，仅作为推荐。 |

## 3. Landing 专项确认

| ID | 选择 | 备注 |
|---|---|---|
| L-01 | A | 按新做落地页处理，旧结构不作为强约束。 |
| L-02 | C | Hero 暂不锁定，先保留方向性原则。 |
| L-03 | C | Landing 单独一套导航，不复用 Tenant TopNav。 |
| L-04 | C | 周一前不锁卡片体系，只保留内容结构。 |
| L-05 | C | 周一前不交付新资产，只交规则。 |

## 4. 跨端组件确认

| ID | 选择 | 备注 |
|---|---|---|
| D-01 | B | 结构统一，圆角和 footer 按钮按端分流。 |
| D-02 | B | 筛选 / 搜索胶囊，普通表单 4px，DatePicker 跟随场景。 |
| D-03 | B | 同事确认当前没有引用 Tenant Text Switch；该弱切换样式从周一交付包删除，只保留 Tenant 胶囊 Tabs / Segment。 |

## 5. 后续回写原则

- 已选 A/B/C 的项目，可作为后续更新 portable design pack 的决策依据。
- 标注“暂选 / 后续进一步确认”的项目，写入规范时需保留为“当前口径 / 可后续回写”，不要包装成不可变更的最终裁决。
- 客户端和落地页专项项如涉及同事 owner，建议同步本表后再回写 `tenant.md` / `landing.md`。

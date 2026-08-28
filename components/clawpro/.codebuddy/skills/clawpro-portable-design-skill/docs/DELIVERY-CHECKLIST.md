# Delivery Checklist

## 1. 打包前检查

- [ ] `README.md`、`HANDOFF.md`、`DEVELOPER-USAGE.md`、`DESIGN-AUDIT-PLAYBOOK.md`、`INDEX.md`、`SKILL.md`、`STATUS.md` 都存在
- [ ] `references/` 已覆盖 foundation、admin、tenant、landing、migration-map
- [ ] `component-specs/` 已覆盖当前周一需要的高风险组件
- [ ] `INDEX.md` / `MANIFEST.json` 已同步所有新增 spec 和 portable 示例
- [ ] `portable/` 至少提供 card / table / empty-state 的 fallback 示例
- [ ] `tokens/` 至少包含 JSON 和 4 个 markdown 维度说明
- [ ] `qa/` 至少包含 admin / tenant / component-review checklist
- [ ] `assets/icon-registry.example.json` 存在
- [ ] `references/conflict-log.md` 已收口当前所有设计确认与待确认条目
- [ ] `scripts/check-design-usage.mjs` 可运行

## 2. 现场交付前检查

- [ ] 已运行 `check-design-usage.mjs`
- [ ] 已生成 zip 包或确认整文件夹可直接交付
- [ ] 已明确告诉接收方先读 `README.md`、`DEVELOPER-USAGE.md`；如做页面评审 / 换皮验收 / 反向审查，再读 `DESIGN-AUDIT-PLAYBOOK.md`
- [ ] 已把 `references/conflict-log.md` 中的已确认项和“暂选 / 后续进一步确认”项单独标出，不让前端误以为所有项都已最终锁死

## 3. 给产品前端的同步要点

- [ ] 解释这是 portable design pack，不是 demo 仓源码镜像
- [ ] 强调宿主仓可复用现有组件，但视觉要按 spec 对齐
- [ ] 强调没有对应组件时要按 Portable Fallback 还原
- [ ] 强调高 / 中风险组件先看 `component-specs/`
- [ ] 强调按 `DEVELOPER-USAGE.md` 先选 1 个 Admin + 1 个 Tenant 试点页
- [ ] 如做页面评审 / 换皮验收 / 设计仓库反向审查，按 `DESIGN-AUDIT-PLAYBOOK.md` 先审报告、不直接改代码

## 4. 交付后维护

- [ ] 新冲突回写 `references/conflict-log.md`
- [ ] 新裁决先与你确认后再改 Admin / 全局规范
- [ ] 新开对话前先看 `STATUS.md`


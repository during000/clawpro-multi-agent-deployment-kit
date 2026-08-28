# 场景 B · 新增组件（`rating-stars` onboarding 全流水线）

> 日常触发语：「我新做了一个 rating-stars 组件。」
> 亮点：新组件从「差集 → spec 骨架 → MANIFEST → 映射表 → fixtures」的完整纳管流水线；含 **`git add -N` 的关键坑** 与 **verify-manifest 的 FAILED→PASSED**。

## 1. 新建组件 + 关键 add -N
- 新建 `client/src/components/ui/rating-stars.tsx`（含 `type RatingStarsSize = "sm" | "md" | "lg"`，能编译即可）。
- **对照演示（讲清坑点）**：
  ```
  # ① 未跟踪，diff 抓不到
  node .codebuddy/skills/clawpro-portable-design-skill/scripts/diff-code-vs-docs.mjs --since HEAD ; echo "exit=$?"
  # 实测：检查组件 0 个 / exit=0（完全漏掉新文件）

  # ② 登记进 git index
  git add -N client/src/components/ui/rating-stars.tsx
  ```

## 1.5 人工触发（真实入口）📸 `B0-trigger.png`
建完组件 + `git add -N` 后，对话框说口令：**`对齐 skill 规范`**。AI 读 `SOP.md` A 段 → 执行下述纳管流水线。

## 2. 触发增量差集
```
node .codebuddy/skills/clawpro-portable-design-skill/scripts/diff-code-vs-docs.mjs --since HEAD ; echo "exit=$?"
```
**实测证据（exit=1）**：`CHANGED_NO_SPEC — rating-stars`（改动碰到、且不在 62 项历史豁免里 → 提示纳管）。

## 3. 纳管：先登记 MANIFEST（演出 verify 的“牙齿”）
> 顺序刻意「先登记、后建文件」，让 verify-manifest 演出 FAILED→PASSED。
- 在 `MANIFEST.json` 的 `componentSpecs` 补 `"component-specs/rating-stars.md"`。
- 此时 spec 文件还没建：
  ```
  node .codebuddy/skills/clawpro-portable-design-skill/scripts/verify-manifest.mjs ; echo "exit=$?"
  ```
  **实测**：`✗ MANIFEST validation FAILED` · `Missing component spec: component-specs/rating-stars.md` · Errors:1 · **exit=1**。
- 补 spec 骨架 `component-specs/rating-stars.md`（登记 `sm`/`md`/`lg` 形态字面量），复跑：
  ```
  node .codebuddy/skills/clawpro-portable-design-skill/scripts/verify-manifest.mjs ; echo "exit=$?"
  ```
  **实测**：`✓ MANIFEST validation PASSED` · Component Specs **41** · **exit=0**。

## 4. 映射表 + 引用自洽
```
node .codebuddy/skills/clawpro-portable-design-skill/scripts/generate-component-mapping.mjs
node .codebuddy/skills/clawpro-portable-design-skill/scripts/check-spec-symbols.mjs
```
- **映射表实测**：重生成 47 个组件，`references/component-mapping.md` 新增 rating-stars（出现 4 处）。
- **引用自洽实测（诚实口径）**：check-spec-symbols 全仓扫描 → exit=1，但那 2 条 ghost 都在 `SKILL.md`（`isForbiddenLucideSlot` / `someIcon`），是**历史存量**；**rating-stars 零新增 ghost**。正好印证「机制只防增量、历史存量另案」。

## 5. fixtures + 复跑闭环
```
node .codebuddy/skills/clawpro-walkthrough/scripts/walkthrough.mjs refresh-fixtures
node .codebuddy/skills/clawpro-portable-design-skill/scripts/diff-code-vs-docs.mjs --since HEAD ; echo "exit=$?"
```
**实测证据**：fixtures 重抽 exit 0；复跑差集 `✅ 无 P1 差集` + **exit=0**（CHANGED_NO_SPEC 消除）。

## 6. 亮点话术（PPT）
- **新组件 onboarding 一条龙**：差集 → spec 骨架 → MANIFEST → 映射表 → fixtures，退出码 1→0。
- **git add -N 是硬门槛**：不登记的新文件不进 diff，机制机械圈定改动、不靠口述。
- **verify-manifest 有牙齿**：登记了却漏建文件会 FAILED，防「登记与文件不一致」。

## 7. 录制取帧（截图存本文件夹）
| 文件名 | 截什么 |
|---|---|
| `B1-addN-contrast.png` | 未 add -N「检查 0 个/exit0」vs add -N 后「CHANGED_NO_SPEC/exit1」 |
| `B3-manifest-failed.png` | verify-manifest FAILED · Missing component spec · exit=1 |
| `B3-manifest-passed.png` | verify-manifest PASSED · Component Specs 41 · exit=0 |
| `B5-mapping-added.png` | component-mapping.md 新增 rating-stars |
| `B6-diff-exit0.png` | 复跑差集 exit=0 |
| `B-recording.mp4` | 本场景全程录屏 |

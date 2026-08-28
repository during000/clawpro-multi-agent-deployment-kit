# 彩蛋 E · 全量体检（压轴讲机制的「聪明」）

> 日常触发语：「全量体检一下 skill 对齐度。」
> 定位：回答「现在 skill 对规范对齐到什么程度」，产出一次性快照；findings 只进 backlog、**不强制修**。

## 0. 人工触发（真实入口）📸 `E0-trigger.png`
对话框说口令：**`全量体检 skill 对齐度`**（区别于增量口令）。AI 读 `SOP.md` B 段（全量审计）→ 跑 `--full` 出一次性 backlog 快照。

## 1. 全量扫描
```
node .codebuddy/skills/clawpro-portable-design-skill/scripts/diff-code-vs-docs.mjs --full ; echo "exit=$?"
```

## 2. 实测证据（本仓真实数字，勿用估算）
- 检查组件 **83 个** + index.css
- **历史豁免主动跳过 62 个**（与 `references/historical-exemption.json` 完全一致：DarkVeil / Surface / card / dialog / input / select …）
- 产出 **32 条 P1 候选**（全部为 `CODE_ENUM_NOT_IN_DOC`：有 spec 的组件里、代码枚举值未在 spec 声明）
- **exit=1**
- 换算：若不豁免，会额外吐出 62 条 `NO_SPEC_BACKLOG` → 合计约 **94 条**。

## 3. 亮点话术（PPT）
- 机制**不是看不见历史包袱**，而是**主动豁免、只在增量里打扰你**：全量 94 条里，62 条历史存量被挡在外面，只剩 32 条真正的枚举漏声明候选。
- **默认永远走增量**（`--since 基线`），绝不自动跑全量淹没你；要体检时才 opt-in `--full`。
- 全量一跑即得**全景快照**：一眼看清「对齐到什么程度」，findings 进 backlog 由人挑选立项。

## 4. 收束话术
增量对齐（口令「对齐 skill 规范」）负责日常防漂移；全量审计（口令「全量体检 skill 对齐度」）负责阶段性体检。同一套脚本、两种 scope，互不干扰。

## 5. 录制取帧（截图存本文件夹）
| 文件名 | 截什么 |
|---|---|
| `E1-full-exemption62.png` | 「检查组件 83 个 + index.css；历史豁免跳过 62 个」整行 |
| `E1-full-findings32.png` | 「发现 32 条 P1 差集候选」+ exit=1 |
| `E-recording.mp4` | 本场景全程录屏 |

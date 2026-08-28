# clawpro-walkthrough

ClawPro 管控端**设计走查 skill**：对管控端代码（`.tsx` / `.css`）做毫秒级静态扫描，自动揪出圆角、色值、图标槽位、Surface 套娃、组件漂离等问题，产出结构化清单（`audit-report.csv` + `design-todo.csv`）与可追踪的设计健康分。

> **规范是法律，脚本是警察。**
> `clawpro-portable-design-skill` 是"法律"（规范原文 + 检测脚本），`clawpro-walkthrough` 是"警察"（只负责机械命中、取证、出清单）。每条走查结论都能回溯到规范章节。

## 安装

本 skill 需与配套的 `clawpro-portable-design-skill` **一起安装**。完整步骤见 **[`INSTALL.md`](./INSTALL.md)**。

一句话版：把这两个目录并列放进目标工程的 `.codebuddy/skills/` 下即可。

```
<你的工程>/.codebuddy/skills/
├── clawpro-walkthrough/            # 本 skill（警察）
└── clawpro-portable-design-skill/  # 配套设计规范 skill（法律）——强依赖
```

## 目录结构

```
clawpro-walkthrough/
├── SKILL.md          # skill 定义 + 审查 SOP + 规则锚点对照表（AI 入口）
├── INSTALL.md        # 安装与依赖说明（对外交付必读）
├── TUTORIAL.md       # 使用教程
├── ONE-PAGER.md      # 一纸摘要
├── DESIGN.md         # 内部设计笔记 / 路线图（非当前实现说明）
├── scripts/          # 走查引擎（Node ≥ 18）
│   ├── walkthrough.mjs        # 主入口：audit / diff / critique / polish / trend / explain
│   └── detectors/             # 9 类内置 detector + 1 类外部 detector
├── fixtures/         # 规则快照（随包固化，运行时真相源，无需再生成）
├── tools/
│   └── walkthrough_report_csv.py  # CSV 规范化 + 枚举校验（可选，需 Python 3）
└── ci/               # CI 接入模板（示例基于腾讯蓝盾 Stream，其它平台需自行改写）
```

## 快速上手

```bash
cd .codebuddy/skills/clawpro-walkthrough

# 全量审计某个页面 / 目录（目标必填，按你的工程结构传）
node scripts/walkthrough.mjs audit <你的工程>/src/pages/Some.tsx

# 增量审计当前 git diff（P0 命中 exit 1，用于 CI）
node scripts/walkthrough.mjs diff

# 校验产出的 CSV（可选）
python3 tools/walkthrough_report_csv.py _walkthrough/<ts>/audit-report.csv _walkthrough/<ts>/design-todo.csv
```

更多用法见 `TUTORIAL.md`；CI 接入见 `ci/README.md`。

## 环境变量

| 变量 | 作用 |
|---|---|
| `WALKTHROUGH_PROJECT_ROOT` | 覆盖被扫描工程的仓库根（默认按标准安装位置反推） |
| `WALKTHROUGH_DESIGN_SKILL_DIR` | 配套设计 skill 装到别处 / 改名时指向其根目录 |
| `WALKTHROUGH_OUT_DIR` | 覆盖产物输出目录（默认 `<repo>/_walkthrough`） |
| `WALKTHROUGH_SKIP_EXTERNAL=1` | 整体跳过 `external/*` detector |
| `WALKTHROUGH_BLOCK_LEVEL` | CI 阻断阈值：`P0`（默认）/ `P1` / `P2` / `NONE` |
| `WALKTHROUGH_BLOCK_ON_AUDIT=1` | 让全量 `audit` 也启用退出码 |

## 设计约束

- **先确认再改**：扫描只读，发现问题先出清单、逐条经用户确认才动业务代码。
- **不下结论**：AI 拿不准的问题只进 `design-todo.csv` 留给用户裁决，严禁私自定性。
- **不脱离规范立法**：每条 detector 的 `evidence` 必须指回 `fixtures/` 里的具体锚点（例：`tokens.json#radius.--radius`），无锚点的规则禁止落地。

> `fixtures/` 是"设计规范 → 机器可读快照"，随交付包固化。它由维护者在本项目内离线生成，外部用户直接使用即可，无需重新抽取。

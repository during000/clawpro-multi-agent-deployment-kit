# 安装指南 · clawpro-walkthrough

ClawPro 管控端设计走查 skill 的安装与配置说明。适用于把本 skill 交付到**任意工程**里使用。

---

## 一、前置条件

| 依赖 | 版本 | 必需性 | 说明 |
|---|---|---|---|
| **Node.js** | ≥ 18 | 必需 | 运行 `scripts/` 走查引擎 |
| **clawpro-portable-design-skill** | 配套版本 | 强依赖 | 设计规范"法律原文" + 复用其检测脚本；未安装时 `external/*` detector 静默跳过 |
| **Python 3** | ≥ 3.8 | 可选 | 仅 `tools/walkthrough_report_csv.py`（CSV 校验）需要 |
| **Git** | 任意 | 可选 | `diff` 子命令需要（增量走查 / CI 卡口） |

> **为什么强依赖设计 skill**：本 skill 是"警察"，只负责机械命中；判定规则的"法律原文"在
> `clawpro-portable-design-skill`。虽然常用规则已固化进 `fixtures/`（离线可用），但 `external/*`
> 一类 detector 会在运行时调用设计 skill 的检测脚本，缺失时该类检查会被跳过。

---

## 二、安装步骤

### 1. 放置 skill 目录

把 `clawpro-walkthrough` 与配套的 `clawpro-portable-design-skill` **并列**复制到目标工程的
`.codebuddy/skills/` 目录下：

```
<你的工程>/
└── .codebuddy/
    └── skills/
        ├── clawpro-walkthrough/            # 本 skill
        └── clawpro-portable-design-skill/  # 配套设计规范 skill（强依赖）
```

> 两个 skill 默认按**同级目录**相互发现，放对位置即可零配置运行。

### 2. 验证安装

```bash
cd <你的工程>/.codebuddy/skills/clawpro-walkthrough

# 打印帮助 —— 能看到子命令列表即安装成功
node scripts/walkthrough.mjs

# 解释一条规则 —— 能打印规则定义即 fixtures 就绪
node scripts/walkthrough.mjs explain radius

# 跑一次真实走查（目标必填，换成你自己的页面）
node scripts/walkthrough.mjs audit <你的工程>/src/pages/YourPage.tsx
```

跑完后产物落在 `<你的工程>/_walkthrough/<时间戳>/`（`audit-report.csv` + `design-todo.csv` + `meta.json`）。

---

## 三、非标准位置安装

若 skill 没有放在标准位置 `<repo>/.codebuddy/skills/`，或想扫描另一个仓库，用环境变量覆盖：

```bash
# 指定被扫描工程的仓库根（决定产物输出目录与 file 相对路径）
WALKTHROUGH_PROJECT_ROOT=/path/to/your/repo \
  node scripts/walkthrough.mjs audit /path/to/your/repo/src/pages/YourPage.tsx

# 配套设计 skill 装到别处 / 改了名
WALKTHROUGH_DESIGN_SKILL_DIR=/path/to/clawpro-portable-design-skill \
  node scripts/walkthrough.mjs audit /path/to/your/repo/src/pages/YourPage.tsx
```

完整环境变量：

| 变量 | 默认 | 作用 |
|---|---|---|
| `WALKTHROUGH_PROJECT_ROOT` | 按标准安装位置反推 | 被扫描工程的仓库根 |
| `WALKTHROUGH_DESIGN_SKILL_DIR` | 同级 `clawpro-portable-design-skill` | 配套设计 skill 根目录 |
| `WALKTHROUGH_OUT_DIR` | `<repo>/_walkthrough` | 产物输出目录 |
| `WALKTHROUGH_SKIP_EXTERNAL` | 未设 | 设为 `1` 整体跳过 `external/*` detector |
| `WALKTHROUGH_BLOCK_LEVEL` | `P0` | CI 阻断阈值：`P0` / `P1` / `P2` / `NONE` |
| `WALKTHROUGH_BLOCK_ON_AUDIT` | 未设 | 设为 `1` 时全量 `audit` 也启用退出码 |

---

## 四、接入 CI（可选）

`ci/` 目录提供一份**腾讯蓝盾 Stream** 参考模板。其它平台请以 `walkthrough.mjs diff` 的退出码为契约自行接入：

```bash
node scripts/walkthrough.mjs diff   # 0=全绿 / 1=命中阻断阈值 / 2=异常
```

详见 `ci/README.md`。

---

## 五、目录说明（交付包含什么）

| 路径 | 说明 | 是否运行必需 |
|---|---|---|
| `SKILL.md` | skill 定义 + 审查 SOP（AI 入口） | 是 |
| `scripts/` | 走查引擎（主入口 + 10 类 detector） | 是 |
| `fixtures/` | 规则快照（随包固化，运行时真相源） | 是 |
| `tools/walkthrough_report_csv.py` | CSV 校验工具 | 否（可选） |
| `ci/` | CI 接入模板 | 否（可选） |
| `TUTORIAL.md` / `ONE-PAGER.md` | 使用教程 / 一纸摘要 | 否（文档） |
| `DESIGN.md` | 内部设计笔记 / 路线图 | 否（文档） |

> `fixtures/` 由维护者在原项目内离线生成并固化随包交付，**外部用户直接使用即可，无需也无法自行重抽**
> （重抽脚本 `extractors/` 依赖原项目源码，不在交付包内）。

---

## 六、卸载

删除 `.codebuddy/skills/clawpro-walkthrough/` 目录即可。走查产物在工程根 `_walkthrough/`，可一并删除。

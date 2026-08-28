# ClawPro Walkthrough — CI 接入

本目录存放走查 skill 的 CI 模板。平台：**腾讯蓝盾 Stream（工蜂 MR 触发）**。

> **对外交付说明**：这是一份**参考模板**，与腾讯内部蓝盾 / 工蜂平台强绑定。其它平台
> （GitHub Actions / GitLab CI / Jenkins 等）需自行改写；模板中监听的 `client/src/**`
> 路径也请按你的工程结构调整。跨平台唯一稳定的契约是 `walkthrough.mjs diff` 的退出码
> （见下文"四、退出码契约"），据此接入任意 CI 即可。

## 目录结构

```
ci/
├── README.md                    # 本文件
└── bk-walkthrough.yml           # 蓝盾 Stream v2.0 模板（工蜂 MR）→ 落地 .ci/walkthrough.yml
```

> **快照定位约定（重要）**：CI 脚本统一用 `walkthrough.mjs` 写的 `_walkthrough/LATEST`
> 指针文件定位本次快照，**禁止用 `ls -t _walkthrough | head -1`**——`appendTrend` 会在写完
> 时间戳快照后才创建/刷新 `snapshots/` 目录，在全新 checkout（CI）里其 mtime 反而更晚，
> `ls -t` 会误抓 `snapshots/` 导致读不到 `meta.json`。模板已用 `cat _walkthrough/LATEST`
> + glob `[0-9]*/` 兜底。

---

## 一、接入步骤（蓝盾 Stream / 工蜂）

工蜂仓库用蓝盾 Stream，CI 文件放在仓库根 `.ci/` 目录下。

```bash
# 1) 把模板拷到仓库根 .ci/（工蜂 Stream 自动读 .ci/ 下的 *.yml）
cp .codebuddy/skills/clawpro-walkthrough/ci/bk-walkthrough.yml .ci/walkthrough.yml

# 2) 确认工蜂项目已开启 CI，且监听 MERGE_REQUEST webhook
#    （Settings → 构建，勾选监听 MR 事件；否则 MR 不触发）

# 3) 提交并推到 main
git add .ci/walkthrough.yml
git commit -m "ci: enable clawpro admin walkthrough on MR (bk stream)"
git push woa <branch>
```

接入后，向 `main`/`master` 发起 MR（open / reopen / 源分支 push-update）会自动跑一次增量审计：
- 命中阻断阈值 → 流水线红 → MR 合并按钮变灰（需 `block-mr:true`）
- 不论红绿，artifact 里有完整 `audit-report.csv` / `meta.json` 可下载

---

## 二、阈值调参（软启动）

`bk-walkthrough.yml` 内 step 脚本中 `WALKTHROUGH_BLOCK_LEVEL` + `on.mr.block-mr` 两段配合：

| 阶段 | `WALKTHROUGH_BLOCK_LEVEL` | `on.mr.block-mr` | 行为 |
|---|---|---|---|
| **试运行期（第 1 周）** | `NONE` | `false` | 只跑只报、不拦合并。给团队适应+修存量。 |
| **稳定期（第 2 周起）** | `P0` | `true` | 命中 P0 → exit 1 → MR 合不了，必须先修。 |
| 团队成熟后 | `P1` | `true` | P0 + P1 都阻断（含硬编码颜色 / 自创 shadow）。 |

> `block-mr: true` 需 `report-commit-check: true` 同时生效（模板已配）；失败时工蜂 MR
> 页面会显示红色 commit check 并禁止合并。

---

## 三、本地复现蓝盾上的红

CI 红了开发想本地复现，跑同一条命令即可：

```bash
git fetch origin main
git diff --name-only origin/main...HEAD | grep -E '\.(tsx?|jsx?|css)$' | tr '\n' ' ' | \
  xargs node .codebuddy/skills/clawpro-walkthrough/scripts/walkthrough.mjs audit
cat _walkthrough/$(cat _walkthrough/LATEST)/audit-report.csv

# 想知道某条规则为啥违规
node .codebuddy/skills/clawpro-walkthrough/scripts/walkthrough.mjs explain radius
```

---

## 四、退出码契约

| 子命令 | 退出码语义 | 备注 |
|---|---|---|
| `diff` | 0 = 全绿；1 = 命中 ≥ 阻断阈值；2 = 异常 | 默认开启阻断 |
| `audit <target>` | 默认 0（不阻断历史存量）；设 `WALKTHROUGH_BLOCK_ON_AUDIT=1` 后等同 diff | CI 模板已开此开关 |
| `radius / color / icon-slot / ...` | 同 diff | 用户显式查某规则时，命中即视为需要修 |
| `explain` | 0 / 1（未知 ruleId） | — |

环境变量：

| 名 | 默认 | 作用 |
|---|---|---|
| `WALKTHROUGH_BLOCK_LEVEL` | `P0` | 阻断阈值：`P0` / `P1` / `P2` / `NONE` |
| `WALKTHROUGH_BLOCK_ON_AUDIT` | 未设 | 设为 `1` 时全量 audit 也启用退出码 |

---

## 五、镜像 / Node 注意事项

- 模板默认镜像：`mirrors.tencent.com/ci/tlinux3_ci:latest`。
- **若该镜像无 Node 20**：换成内网带 node 的镜像，或在 step 里 `nvm install 20`。
  脚本已加 `command -v node` 兜底报错，CI 红得很明显。
- diff 范围用 `ci.base_ref`（MR 目标分支）：先 `git fetch origin $BASE`，再
  `git diff origin/$BASE...HEAD`。

---

## 六、周巡检 automation（v0.7 计划）

CI 卡 MR 的同时，建议另开一条**每周一早上 9 点自动跑全量 audit + 输出 design-todo** 的 automation：
- 不阻塞任何人
- 飞书机器人推送
- 只列 `page-recipe-match` + `component-drift` + `icon-slot` 三类高纯度信号

这一步在 walkthrough v0.7 实现，下一轮用本仓库 `.codebuddy/automations/` 写出来。

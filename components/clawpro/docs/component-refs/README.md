# 组件引用及更新 · 协作总入口

> ClawPro Design System v2 的**组件元数据中心**。
> 设计 / 前端 / PM 三方都从这里出发，同一份事实数据，三种使用方式。

## ⚡ 快速入口

| 我是… | 我要看 | 入口 |
| --- | --- | --- |
| **设计师** | 这个组件长什么样、有几种 variant、token 是什么 | 🎭 [组件图鉴](https://clawprocomponent.devcloud.woa.com/index.html) |
| **前端** | 我要改 button 的某个 variant，影响哪些页面 | 🧩 [pages-index.html](./pages-index.html) → "组件 → 页面" Tab |
| **PM / 走查** | v2 设计系统改造覆盖了多少 / 还剩哪些没改 | 📊 [pages-index.html](./pages-index.html) → "改造进度" Tab（默认页）|

> 💡 [pages-index.html](./pages-index.html) 是一站式总览，顶部 Tab 切换；其中「🎭 组件图鉴 ↗」会跳到部署版（不依赖本地服务）。

---

## 🗂 目录里有什么

```
openclaw-enterprise/docs/component-refs/
├── README.md                  ← 你正在看的这个文件
├── pages-index.html           ← 一站式总览（推荐入口 · 6 个 Tab）
│
├── component-variants.json    ← 反向索引：组件 → 维度 → 变体值 → 引用页面
├── component-page-refs.json   ← 简版反向索引：组件 → 引用它的页面列表
├── progress-auto.json         ← v2 改造进度自动扫描结果（53 组件 + 89 页面）
│
├── scan-variants.cjs          ← 扫描器 · 生成 component-variants.json
└── scan-progress.cjs          ← 扫描器 · 生成 progress-auto.json
```

---

## 🔄 数据流

```
                     ┌────────────────────────────────────────┐
                     │  openclaw-enterprise/ (仓库根 = ../..) │
                     │     ├── client/src/components/ui/*.tsx │  ← 组件源
                     │     ├── client/src/pages/**.tsx        │  ← 页面源
                     │     └── docs/button-tokens/*.html      │  ← 图鉴源
                     └────────────────┬───────────────────────┘
                                      │
                  ┌───────────────────┼───────────────────────┐
                  ▼                   ▼                       ▼
          scan-variants.cjs     scan-progress.cjs        发布部署
                  │                   │                       │
                  ▼                   ▼                       ▼
       component-variants.json  progress-auto.json   clawprocomponent.devcloud
       component-page-refs.json                       /index.html, /pm.html, /input.html
                  │                   │                       │
                  └─────────┬─────────┘                       │
                            ▼                                 ▼
                    pages-index.html ────────────────► 🎭 组件图鉴 Tab
                    （6 个 Tab 总览）
```

**唯一事实标准（Single Source of Truth）= `openclaw-enterprise/` 仓库里的真代码。**
本目录的所有 JSON 都是从代码扫出来的衍生物，不是手工维护。

---

## 🛠 如何更新

### 重扫组件变体反向索引

```bash
cd openclaw-enterprise/docs/component-refs
node scan-variants.cjs
# → component-variants.json 刷新
```

### 重扫 v2 改造进度

```bash
cd openclaw-enterprise/docs/component-refs
node scan-progress.cjs
# → progress-auto.json 刷新
```

> 两个脚本会自动把仓库根定位到 `../..`（即 `openclaw-enterprise/` 仓库根），无需传参。
> 如果要扫别的副本，可以加 `--repo /path/to/openclaw-enterprise`。

### 更新组件图鉴

图鉴源文件在 `../button-tokens/`（同 enterprise 仓库 docs 同级）：

- `index.html` · 组件总览索引
- `pm.html` · Button 图鉴
- `input.html` · Input 图鉴
- `../DESIGN_SYSTEM_BUTTON.md` · Button 详细 token 表

改完直接 `scp` 到部署机即可（详见 `../DESIGN_COLLAB_GUIDE.md`）：

```bash
scp -P 36000 *.html root@clawprocomponent.devcloud.woa.com:/data/clawpro-design/
```

Python `http.server` 不需要重启就会读最新文件。

---

## 📐 设计系统一致性原则

> 所有组件的 token 以 `../../client/src/components/ui/*.tsx` 中**真在跑的 className** 为唯一基准。
> 当文档（图鉴 / SKILL.md / DESIGN_SYSTEM_*.md）与代码出现漂移时，**改文档对齐代码**，不是反过来。

参考：
- 设计语言总章 · `clawpro-skill/SKILL.md`（仓库外·设计资料）
- 设计协作约定 · `../DESIGN_COLLAB_GUIDE.md`
- Button 详细 token · `../DESIGN_SYSTEM_BUTTON.md`

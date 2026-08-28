# ClawPro 资源库 · 新资源接入 SOP

> 当项目新增图标 / 图片资源时，如何把它纳入资源库并让 skill 选得到。
> 配套：`docs/ClawPro资源库-阶段9决策溯源(ADR).md`（为什么这么设计）。
> 全部步骤对应真实脚本，命令在**仓库根目录**执行。

---

## 0. 先记住一句话

**日常加图标，通常不用改 skill。** 你只需把资源喂进流水线、指派槽位、重跑两条命令，skill 会自动从更新后的映射里选到新图。只有「新增一种全新槽位类型」才需要动 skill。

---

## 1. 流水线全貌（脚本 = 阶段）

```
scan-resources.mjs       (阶段2 扫描)        ┐
classify-resources.mjs   (阶段3 分类)        │  喂数据 + 指派槽位
detect-duplicates.mjs    (阶段4 查重)        │
apply-governance.mjs     (阶段5 治理，默认 dry-run) ┘
build-canonical-assets.mjs (阶段6 红线入口)
        │
npm run build:skill-map  (阶段9 生成映射)
npm run check:skill-map  (阶段9 守门校验)  ← 绿色才算成功
```

人工事实源（你主要编辑这里，绝不手改 generated/ 产物）：
- `client/src/design-assets/manual-overrides/classification.json` —— 指派槽位 / 确认 / 排除
- `client/src/design-assets/manual-overrides/component-resource-map.json` —— 槽位白名单与约束（仅"新增槽位类型"才动）

---

## 2. 场景 A：新增一个普通业务图标（最常见）

例：新增一个统计卡图标，要进 `number-card` 槽位。

1. **放文件到规范目录**（页面可访问的优先 `client/public/assets/<分组>/`，如 `client/public/assets/number-card/xxx.svg`）。命名 kebab-case、`currentColor`、清理多余 metadata（见 skill `assets-icons.md` §3/§4）。

2. **扫描 + 分类**，让它进 inventory：
   ```bash
   node client/src/design-assets/scripts/scan-resources.mjs
   node client/src/design-assets/scripts/classify-resources.mjs
   ```

3. **指派槽位**（关键）：在 `manual-overrides/classification.json` 的 `overrides` 里，用资源 `id` 登记：
   ```json
   "public__number-card-xxx__<hash>": {
     "componentSlot": "number-card",
     "confirmed": true,
     "note": "NumberCard 卡片图标（设计确认）· 用途说明"
   }
   ```
   - 资源 `id` 从扫描产物 `generated/resource-inventory.generated.json` 里查（按 displayName / sourcePath 找）。
   - 可用 `componentSlot` 值见 `classification.json` 的 `componentSlots`：`admin-sidebar` / `card-left-icon` / `number-card` / `file-type` / `run-status` / `feature-card`。
   - **只有在此登记的 id 生效**，未登记的按规则自动判定，不臆测。
   - 重跑一次 `classify-resources.mjs` 让 override 生效。

4. **重建映射 + 校验**：
   ```bash
   npm run build:skill-map
   npm run check:skill-map      # 必须绿色通过
   ```

5. 完成。skill 下次生成 `number-card` 页面时即可选到这个新图。**无需改 skill。**

---

## 3. 场景 B：新增红线资源（品牌 Logo / 渠道图标 / Agent 头像）

红线资源走 canonical 入口，**门槛更严**：

1. 放文件到 public 目录（必须有 `webPath`，运行时可服务）。
2. 扫描 + 分类后，资源需被判定到红线类目（`brand-logo` / `channel-icon` / `agent-avatar`），且 `status=normal`。
3. 跑 `node client/src/design-assets/scripts/build-canonical-assets.mjs` 建立 / 更新 `canonical-assets.ts` 入口。
4. `npm run build:skill-map && npm run check:skill-map`。
5. 红线资源在页面层**只能经 `canonicalAssets.<group>.<key>` 引用**，跨仓由宿主注入；**禁改色、禁当普通图标**。

> 红线永不自动归并 / 删除。如涉及组件源码改动，须脱离本任务单独立项。

---

## 4. 场景 C：废弃 / 替换旧资源

- **废弃**：在 inventory 把资源标为非 `normal`（deprecated/avoid），或经治理移除（进 `governance.removedIds`）。重跑 `build:skill-map` 后它会**自动退出候选**，`check:skill-map` 保证不再被引用。
- **替换**：新增替代资源（场景 A），旧资源标废弃；不要用旧路径继续引用已迁移资产。
- **删除冗余副本**：`apply-governance.mjs` 默认 dry-run（只出报告不删）；确认后 `--apply` 才真删，删除均由 git 跟踪，可 `git checkout -- <path>` 回滚。红线永不自动删。

---

## 5. 场景 D：新增一种"全新槽位类型"（少见，需动 skill）

仅当出现一种现有 9 槽位都装不下的新组件图标位置时：
1. 在 `manual-overrides/component-resource-map.json` 的 `slots` 增加新槽位（label / componentSlotPath / recommendedResourceType / allowLucideFallback / redline / constraint 等，基于**真实组件源码审计**，不臆测）。
2. 在 `classification.json` 的 `componentSlots` 同步登记该槽位名。
3. 在 skill `references/assets-icons.md` §5.5 的槽位表补一行（用途 / 红线 / usageScope / 是否允许 lucide 回退 / 引用方式）。
4. 给资源指派该槽位（场景 A 第 3 步）。
5. `npm run build:skill-map && npm run check:skill-map`。

---

## 6. 验收清单（每次接入后必跑）

```bash
npm run build:skill-map            # 重建映射，看候选数变化是否符合预期
npm run check:skill-map            # 10 项校验，必须退出 0
npm run check:component-resources  # 组件源码层防新增违规（基线模式）
```

全绿 = 接入成功且未引入漂移。任一红色 → 看报错信息定位（多为 slot 不合法 / 磁盘文件缺失 / 红线落错槽位 / 计数不一致）。

---

## 7. 场景 E：把更新发布到线上资源库（剥离发布）

> 资源接入并 `check:skill-map` 全绿后，若要让线上 https://clawpro-resource-library.pages.woa.com/ 看到新内容，走这套「剥离发布」。
> **它不在主项目的线上构建里**，而是用一个临时构建入口，把资源库单页编译成静态站，推到独立的工蜂仓库，由 OA Pages 从分支自动部署。
> ⚠️ 线上页面「列出哪些资源」由**构建期打进 JS 的 inventory 决定**——只往 dist 拷 SVG 文件不会让画廊显示新资源，**新增资源后必须重新构建**。

> ---
> ### ⚠️ 重要修正（2026-06-23 实测）：资源库站是 **API 部署模式**，不是 git push
>
> 经控制台与线上实测确认：`clawpro-resource-library.pages.woa.com` 在 OA Pages 控制台的「分支」列是 **`api`**，属于**通过 API 创建的站点**。按 OA Pages 文档，这类站**只能通过 `PUT /api/sites/:cname` 接口更新，git 提交不会生效**。
> - 因此下面 **§7.4 的「git push 到 `oa-pages` 分支」对本站无效**——这正是「6/14 后往 `oa-pages` 推了多次、线上更新时间却死卡 6/14 纹丝不动」的根因。
> - 资源库站的**正确发布通道见新增的 §7.6（API 部署）**。§7.3「构建出精简产物」仍然需要，变的只是发布那一步。
> - §7.4 的 git push 流程**仅适用于「分支 = `oa-pages`」模式的站**（如组件展示台 `clawpro-design-system`），对资源库不要再用。

### 7.1 资源库发布关键信息

| 维度 | 值 |
| --- | --- |
| 渲染页面 | `client/src/pages/DesignSystemAssets.tsx` |
| 工蜂仓库 | `git@git.woa.com:miekoyychen/clawpro-resource-library.git` |
| 部署分支 | `oa-pages`（无 CI，OA Pages 从该分支自动部署） |
| 线上域名 | `clawpro-resource-library.pages.woa.com`（内网 tof/iOA 鉴权） |
| 产物入口名 | `resource-lib-index.html`（chunk `resource-lib-index-*.js/css`） |
| 本地部署副本 | 工作区内 `clawpro-resource-library-dist/`（内含独立 `.git`，远程即上面的工蜂仓库） |

> 组件展示台是**同一套剥离发布机制的另一个实例**（仅页面 / 仓库 / 域名不同），其完整说明独立维护在 `client/public/research/clawpro-design-system-showcase-guide.md`。本 SOP 只聚焦资源库、不展开展示台——两者机制相同，需要时看那份指南即可。

### 7.2 关键纪律（为什么主项目里查不到发布痕迹）

- 发布只动两类临时入口，**发完必须全部还原**，避免污染主项目线上构建：
  - 被改的 `client/src/__covibe__/main.tsx` 与 `vite.covibe.temp.config.ts` —— 用 `git checkout --` 还原；
  - 新建的 `client/resource-lib-index.html` —— 发完直接 `rm`（`client/covibe-index.html` 保持不动）。
- 这两个被改文件默认停在「组件展示台」状态（`main.tsx` 渲染 `DesignSystemComponents`、`vite` 配置 `input` 为 `covibe-index.html`、`outDir` 指向 `clawpro-design-system-only`）。发资源库前临时切到资源库，发完切回。
- 真正长期保存发布产物与域名配置的，是**独立的工蜂仓库**，不是主项目——所以主项目里查不到发布痕迹是正常的。

### 7.3 构建资源库静态产物（临时入口，发完必还原）

> 构建产物先落到**临时 scratch 目录**（不是 dist 仓库），再同步进 dist——绝不能把 `outDir` 直接指向 dist，因为 `emptyOutDir:true` 会清空目标目录、可能误伤 dist 的 `.git`/`CNAME`。

1. **新建入口 HTML**：`client/resource-lib-index.html`（可复制 `client/covibe-index.html`），`<title>` 改为「ClawPro 资源库」。产物 chunk 名 `resource-lib-index-*` 即源于此文件名。
2. **改构建入口** `client/src/__covibe__/main.tsx`：**只渲染 `DesignSystemAssets`**，去掉 `<App/>` 兜底分支与对 `App` 的 import（`DesignSystemAssets` 不依赖路由，与展示台一样只需 `ThemeProvider`/`UserRoleProvider`/`TooltipProvider`/`Toaster` 即可）。
   - ⚠️ 若保留 `<App/>` 兜底，会把全站 lazy 路由 chunk（200+ 个无用文件）一并打进产物。资源库站点只用 `/` 这一个页面，**务必精简成单包**。
3. **改构建配置** `vite.covibe.temp.config.ts`：`rollupOptions.input` → `client/resource-lib-index.html`；`build.outDir` → scratch 目录 `../clawpro-deploy/clawpro-resource-library-only`（保持 `emptyOutDir:true`，scratch 可随便清）。
4. **构建**：`rm -rf ../clawpro-deploy/clawpro-resource-library-only && npx vite build --config vite.covibe.temp.config.ts`。
5. **核对精简产物**：`scratch/assets/` 下应**只有** `resource-lib-index-<hash>.js` + `resource-lib-index-<hash>.css` 两个构建文件（外加若干 `*.svg`/`*.png`，是 `import.meta.glob` 收的 src 预览资源）。若冒出大量 `AgentChat-*.js`/`ApiDocs-*.js` 之类按页 chunk，说明第 2 步没去掉 `<App/>`，回去改。

### 7.4 同步进 dist 仓库并推送（OA Pages 自动部署）

用 `rsync --delete` 把 scratch 产物整目录同步进 `clawpro-resource-library-dist/`，并用 `--exclude` **保护**域名与部署元数据不被删；旧 hash 产物会被自动清掉，不留残留。

```bash
SRC=../clawpro-deploy/clawpro-resource-library-only/
DST=clawpro-resource-library-dist/
# 1) 同步产物（保护 .git / CNAME / .oa-pages-deploy / .gitkeep / index.html 不被删）
rsync -a --delete \
  --exclude='.git' --exclude='.gitkeep' --exclude='CNAME' \
  --exclude='.oa-pages-deploy' --exclude='index.html' "$SRC" "$DST"
# 2) 站点根入口 index.html = 最新 resource-lib-index.html
cp "${DST}resource-lib-index.html" "${DST}index.html"
# 3) 刷新部署触发标记
printf 'deploy trigger %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" > "${DST}.oa-pages-deploy"
# 4) 提交推送（分支 oa-pages）
cd "$DST"
git add -A
git commit -m "publish: 重建并更新 clawpro 资源库静态站"
git push origin oa-pages
# OA Pages 自动从 oa-pages 分支部署到 https://clawpro-resource-library.pages.woa.com/
```

发布后 `git status`（dist 内）应只剩合理变更：替换的 `assets/resource-lib-index-*.js/css`、刷新的 `.oa-pages-deploy`、`index.html`/`resource-lib-index.html`，以及本次**新增资源对应的 `assets/**` 预览文件**。`CNAME`（＝ `clawpro-resource-library.pages.woa.com`）**绝不能删**，删了线上域名就丢。

### 7.5 收尾（必做）

- **还原主项目临时入口**：`git checkout -- client/src/__covibe__/main.tsx vite.covibe.temp.config.ts`，删除临时的 `client/resource-lib-index.html`，并清理 scratch：`rm -rf ../clawpro-deploy/clawpro-resource-library-only`。
- **验证主项目零污染**：`git status` 应回到发布前状态、分支不变（发布全程只在嵌套的独立仓库 `clawpro-resource-library-dist/` 内 commit/push）。
- **校验线上**：在内网浏览器访问 `https://clawpro-resource-library.pages.woa.com/`，确认新资源已上线（OA Pages 增量更新，未列出的文件保持不变）。

---

## 7.6 API 部署（资源库站正道）与部署校验 ★

> 适用于「分支 = `api`」的站（资源库）。用项目自带脚本 `scripts/deploy-oa-pages.mjs`：走 `PUT /api/sites/:cname`、`X-Api-Key` 鉴权、自动按 4.5MB 分批、文本直传 / 二进制 base64，**上传即生效**，无需等构建或发布。§7.3 构建出的精简产物仍是前提。

**前置条件**：
- **Node ≥ 18**（脚本用全局 `fetch`，无 `node-fetch` 依赖；低版本会 `fetch is not defined`）。
- 能访问内网 `pages.woa.com`（tof/iOA 鉴权）。
- 持有有效 **API Key**（见 §7.6② 的获取与安全说明）。

#### ① 准备「瘦身」产物目录（关键，别直接指 dist 根）

`clawpro-resource-library-dist/` 里混着大量**别站素材**（`landing-assets/` 48M、`page-references/`、`figma-replica/` 等，整目录约 60MB）。资源库站**真正引用的只有 4 类**，依据是从 `index.html`/JS/CSS grep 出的真实引用：

- `index.html`：入口，引向 `assets/resource-lib-index-*.js/css`
- `assets/`：构建产物（JS/CSS）+ 预览资源
- `icon/`：页面引用的图标（引用数应与目录实际文件数**精确相等**，可作核对）
- `fonts/`：CSS `@font-face` 引用的本地字体

```bash
SLIM=/tmp/clawpro-rl-slim
rm -rf "$SLIM" && mkdir -p "$SLIM"
cd clawpro-resource-library-dist
cp index.html "$SLIM"/ && cp -R assets icon fonts "$SLIM"/   # 约 4.3M
# 核对：入口 hash 正确、无 >4.5MB 会被脚本跳过的文件
grep -oE '/assets/resource-lib-index-[^"]+' "$SLIM"/index.html
```

#### ② 上传（API Key 仅经环境变量传入，绝不写进文件 / 提交 / 日志）

```bash
cd <仓库根>
OA_PAGES_API_KEY='oa-pages-key-...' \
OA_PAGES_CNAME='clawpro-resource-library.pages.woa.com' \
OA_PAGES_DIST='/tmp/clawpro-rl-slim' \
OA_PAGES_DESC='OpenClaw 设计资源库' \
node scripts/deploy-oa-pages.mjs > /tmp/deploy.log 2>&1; echo "EXIT=$?"; cat /tmp/deploy.log
```

- ⚠️ **务必把输出落盘再整体 `cat`**：流式输出会被截断，曾据截断输出（"第2批0文件"）误判成败。
- **成功标志**：日志含 `响应：{"message":"网站更新成功", ... "updated_at":"<ISO时间>"}` 且 `EXIT=0`。
- **API Key 来源与安全**：在 OA Pages 控制台「API Key 管理」获取（形如 `oa-pages-key-...`）。只放进**环境变量**，**严禁写入脚本 / 文件 / git 提交 / 对话**；一旦在不可信处出现过（如贴进聊天），用完即到控制台**轮换重置**。

#### ③ 部署校验清单（必跑，不靠脚本输出一面之词）

```bash
# a) 线上 index.html 引用的 JS hash = 本次构建的新 hash
curl -s "https://clawpro-resource-library.pages.woa.com/?_=$(date +%s)$RANDOM" \
  -H 'Cache-Control: no-cache' | grep -oE 'resource-lib-index-[A-Za-z0-9_-]+\.js'
# b) 新产物可访问 = 200
curl -s -o /dev/null -w '%{http_code}\n' \
  "https://clawpro-resource-library.pages.woa.com/assets/resource-lib-index-<新hash>.js"
# c) 线上 last-modified 应与 API 返回的 updated_at **到秒吻合**（最硬的生效证据）
curl -sI "https://clawpro-resource-library.pages.woa.com/?_=$(date +%s)$RANDOM" | grep -i last-modified
```

- **控制台为最终权威**：刷新 https://pages.woa.com 控制台，资源库站「更新时间」应变为今天、「占用空间」应**变大**——因为 API 是**增量更新**（未列出的文件保持不变），成功后占用只增不减。**若占用 / 更新时间纹丝不动，就是没传进去。**

#### ④ 失效教训（别再踩）

- 改 `.oa-pages-deploy` 文件**不能**触发本站部署（那是 git 模式的触发物，对 api 模式无效）。
- 对本站 `git push oa-pages` **无效**，更新时间会一直停在最后一次 API 上传的日期。
- `OA_PAGES_DIST` **不要**指向 dist 根目录（会把 48M 别站素材一起传），务必先瘦身到 §7.6① 的 4 类文件。
- 别凭脚本流式输出或单次 curl 就宣布成功；**以控制台占用 / 更新时间的实际变化为准**。

### 7.7 构建配置现状提醒（避免踩孤儿配置）

- 仓库内存在 `vite.resource-lib.config.ts`（专用配置：`input` = `client/resource-lib-index.html`、`outDir` 直指 `clawpro-resource-library-dist/`、`emptyOutDir:true`）。**看似是更干净的构建入口，但当前并未接线**：其入口 `client/resource-lib-index.html` 在源码中**不存在**（仅为发布时临时创建、发完即删），且**未挂进任何 npm script、本 SOP 与展示台指南也都不引用它**。
- 因此**当前经验证可用的构建路径仍是 §7.3（covibe 临时入口）**。在没有补齐「常驻入口 HTML + 对应只渲染 `DesignSystemAssets` 的入口模块」之前，**不要**盲目改用 `vite build --config vite.resource-lib.config.ts`（会因入口缺失而失败）。
- 若要把它转正为长期方案，需另立小任务：补常驻入口 HTML/入口模块、加 `build:resource-lib` script、并据实改写 §7.3——属流程改造，不在「日常接入」范围内。

---

## 8. 红线纪律（务必遵守）

- **禁手改 generated/ 产物和 `resource-skill-map.json`**：它们是脚本派生的，手改会被下次重跑覆盖且引入漂移。
- **改口径改脚本，不改产物**。
- **候选只来自 inventory 已审计字段**，不靠语义硬塞 slot。
- **红线资源**（品牌/渠道/头像）永不改色、永不进普通槽位、永不自动删除。

---

## 9. 故障排查速查表

| 症状 | 根因 | 处理 |
| --- | --- | --- |
| 线上数字 / 更新时间不变，git 推了也没用 | 资源库站是 **API 模式**，git push 对它无效 | 走 §7.6 用 `deploy-oa-pages.mjs` API 上传 |
| 画廊缺新资源，但 dist 里明明有该 SVG | 线上列表由**构建期打进 JS 的 inventory** 决定，拷文件不算数 | 先重新构建（§7.3）再发，别只拷文件 |
| 构建产物冒出大量 `AgentChat-*.js`/`ApiDocs-*.js` 等按页 chunk | §7.3 第 2 步没去掉 `<App/>` 兜底，把全站路由打进来了 | 改 `main.tsx` 只渲染 `DesignSystemAssets` 重新构建 |
| `deploy` 提示某文件超 4.5MB 被跳过 | 单文件超 API 上限（多为引导视频 `*.mp4`） | 视频不在资源计数内可忽略；非传不可则压缩，或该大文件走 git 模式站 |
| `deploy` 输出中途断、看不到「部署完成」 | 流式输出被截断 | `node ... > /tmp/deploy.log 2>&1; echo EXIT=$?; cat` 落盘后整体看 |
| API 返回 401 / 403 | API Key 失效 / 无权限 | 控制台「API Key 管理」重置后重试 |
| 线上 `index.html` 仍引用旧 hash | 没重新构建，或被 CDN/浏览器缓存 | 重建+重传；核验时给 URL 加 `?_=$(date +%s)$RANDOM` 强刷 |
| `check:skill-map` 红 | slot 不合法 / 磁盘文件缺失 / 红线落错槽位 / 计数不一致 | 按报错信息定位对应事实源修正后重跑 |
| 主项目 `git status` 出现一堆改动 | §7.3 临时入口没还原 | 执行 §7.5 收尾：`git checkout --` 还原、删临时 html、清 scratch |

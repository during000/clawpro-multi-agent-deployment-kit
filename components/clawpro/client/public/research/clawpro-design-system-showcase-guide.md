# ClawPro 全局组件展示台：设计、构建与发布指南

> 本文用于说明 `ClawPro 全局组件展示台` 的页面定位、构建方式、部署链路，以及后续更新组件后如何同步更新线上展示页。

## 1. 页面定位

`ClawPro 全局组件展示台` 是 ClawPro 设计系统组件的内部展示与评审页面，主要用于：

- 集中展示 ClawPro 已沉淀的全局组件资产；
- 查看组件真实样式、交互状态、使用指引和迁移建议；
- 通过 `查看应用页面` 跳转到真实业务页面，校验组件在实际场景中的效果；
- 给设计、产品、前端同事提供统一的组件参考入口。

当前线上访问地址：

```text
https://clawpro-design-system.pages.woa.com
```

## 2. 源码位置

展示台源码维护在 `openclaw-enterprise` 主仓库中：

```text
/Users/miekoyychen/openclaw-enterprise
```

核心页面：

```text
client/src/pages/DesignSystemComponents.tsx
```

相关组件主要来自：

```text
client/src/components/ui/
client/src/components/topnav/
client/src/components/AdminLayout.tsx
client/src/pages/admin/
client/src/pages/tenant/
```

注意：不要在展示仓库中直接修改源码。展示仓库只放构建后的静态网页产物。

## 3. 页面设计方式

展示台页面的主要结构如下：

1. **顶部说明区**
   - 展示页面标题：`ClawPro 全局组件展示台`
   - 展示维护人、数据来源和组件统计信息。

2. **组件筛选区**
   - 支持按组件平台筛选：
     - `全部组件`
     - `Global 全局`
     - `Tenant 用户端`
     - `Admin 管控端`
   - 支持关键词搜索。

3. **左侧组件目录**
   - 按组件类别组织，例如：
     - 基础视觉
     - 操作组件
     - 表单组件
     - 反馈组件
     - 数据展示
     - 导航与布局
     - 管控端专属

4. **右侧组件详情区**
   - 组件名称、中文名、说明；
   - 应用范围、组件实例数量；
   - `查看应用页面` 入口；
   - 真实组件预览与全状态展示；
   - 使用指引、注意事项、页面效果校准建议。

5. **真实应用页面跳转**
   - `查看应用页面` 中的页面会跳转到展示包内的示例页面，例如：

```text
/admin/model-config
/admin/members
/admin/platform-policy
/my-openclaw
/model-quota
/skill-square
/openclaw/1
```

## 4. 为什么不直接部署整个 openclaw-enterprise 项目

`openclaw-enterprise` 主项目默认入口是产品 Landing 页，不是展示台页面。直接构建整个项目上传时，访问 `/` 会看到主站首页，而不是 `ClawPro 全局组件展示台`。

因此，当前采用 **展示台专用构建入口**：

- 默认入口直接渲染 `DesignSystemComponents`；
- 不改正式 `client/src/App.tsx`；
- 不影响 `openclaw-enterprise` 主项目协作；
- 只在构建时临时生成入口文件，构建完成后删除。

## 5. 构建方式

### 5.1 临时入口

构建时会临时创建以下文件：

```text
client/covibe-index.html
client/src/__covibe__/main.tsx
vite.covibe.temp.config.ts
```

这些文件只用于发布展示台，构建结束后会删除，不提交到 `openclaw-enterprise` 主仓库。

### 5.2 为什么临时入口要放在 client 内

项目使用 Tailwind CSS v4。Tailwind 会根据项目源码扫描 class。如果临时入口完全放在仓库外，可能导致 Tailwind 没有扫描到 `client/src` 中的组件样式，页面会出现“内容有了但样式丢失”的问题。

所以临时入口需要放在 `client` / `client/src` 范围内，确保 Tailwind 能正确生成完整 CSS。

### 5.3 构建输出目录

构建产物输出到仓库外目录：

```text
/Users/miekoyychen/CodeBuddy/clawpro-deploy/clawpro-design-system-only
```

构建 ZIP 曾用于 Covibe 预览，但目前正式发布走 OA Pages，因此重点使用该目录中的静态产物。

## 6. History Mode 与路由处理

展示台希望 `查看应用页面` 后 URL 变成路径形式，而不是 query 参数形式，例如：

```text
/admin/model-config
/my-openclaw
/model-quota
```

因此当前发布包采用 **history mode**。

由于 OA Pages / 静态站点直达子路径时需要 fallback，构建后会为常用示例路由补充 `index.html` 副本，例如：

```text
admin/model-config/index.html
admin/members/index.html
admin/platform-policy/index.html
admin/session-management/index.html
admin/tokens-monitor/index.html
my-openclaw/index.html
model-quota/index.html
skill-square/index.html
openclaw/1/index.html
openclaw-guide/index.html
openclaw-guide/1/index.html
```

这样可以提升刷新、前进后退、直接访问子路径时的稳定性。

## 7. 展示仓库与 OA Pages 部署

### 7.1 展示仓库

展示仓库地址：

```text
https://git.woa.com/miekoyychen/clawpro-design-system-showcase.git
```

本地路径：

```text
/Users/miekoyychen/CodeBuddy/clawpro-design-system-showcase
```

这个仓库只存放构建后的静态网页产物，不在这里写业务源码。

### 7.2 分支

OA Pages 使用分支：

```text
oa-pages
```

静态资源位于 `oa-pages` 分支根目录。

### 7.3 域名

仓库根目录中包含 `CNAME` 文件，内容为：

```text
clawpro-design-system.pages.woa.com
```

线上访问地址：

```text
https://clawpro-design-system.pages.woa.com
```

### 7.4 权限

OA Pages 后台需要配置访问权限。当前目标是让公司同事可访问，建议使用：

```text
tof 验证（需经过内网 iOA 登录）
```

如果同事访问时看到：

```json
{"message":"无权限访问，请联系该域名管理员 miekoyychen"}
```

说明 OA Pages 权限或公开路径还未配置好，需要到：

```text
https://pages.woa.com/admin
```

找到站点 `clawpro-design-system.pages.woa.com` 调整权限。

## 8. 后续更新组件后的发布流程

### 8.1 你日常应该在哪里改

继续在 `openclaw-enterprise` 主仓库里开发：

```text
/Users/miekoyychen/openclaw-enterprise
```

常见修改位置：

```text
client/src/pages/DesignSystemComponents.tsx
client/src/components/ui/
client/src/components/topnav/
client/src/pages/admin/
client/src/pages/tenant/
```

不要直接修改：

```text
/Users/miekoyychen/CodeBuddy/clawpro-design-system-showcase
```

展示仓库只是“发布后的网页成品”。

### 8.2 本地确认

更新组件或展示台后，先在本地确认效果：

```bash
cd /Users/miekoyychen/openclaw-enterprise
pnpm dev
```

打开：

```text
http://localhost:3002/design-system/components
```

如果端口不是 `3002`，以终端实际输出为准。

重点检查：

- 展示台首页是否正常；
- 左侧组件切换是否正常；
- 新增/修改组件预览是否正确；
- `查看应用页面` 是否可跳转；
- 浏览器后退/前进是否符合预期；
- 点击组件是否有运行时报错。

### 8.3 通知 AI 重新构建发布

确认本地没问题后，可以直接对 AI 说：

```text
我已经在 openclaw-enterprise 里更新并本地确认了 ClawPro 全局组件展示台，请重新构建并发布到 OA Pages。
```

或者简短说：

```text
帮我发布展示台
```

### 8.4 AI 会执行的发布动作

AI 收到发布指令后，会执行：

1. 检查 `openclaw-enterprise` 当前状态；
2. 临时创建展示台专用入口；
3. 构建静态产物到：

```text
/Users/miekoyychen/CodeBuddy/clawpro-deploy/clawpro-design-system-only
```

4. 为 history mode 示例路由补充 fallback `index.html`；
5. 同步产物到展示仓库：

```text
/Users/miekoyychen/CodeBuddy/clawpro-design-system-showcase
```

6. 确保 `CNAME` 内容为：

```text
clawpro-design-system.pages.woa.com
```

7. 提交并推送 `oa-pages` 分支；
8. 删除临时入口和临时配置；
9. 确认 `openclaw-enterprise` 主仓库没有残留部署临时文件；
10. 关注展示仓库 git 体积：OA Pages 按**整个 git 仓库存储**算配额（上限 100MB，含 `.git` 历史）。若长期用普通 commit 累积发布，历史里会堆积多份旧构建产物（如多个 5~7MB 的 `covibe-index-*.js`、旧 mp4），最终撑爆配额导致部署停滞。必要时按第 12 节做一次历史重置。

## 9. 两个仓库的关系

可以这样理解：

```text
openclaw-enterprise 主仓库 = 源码 / Word 原稿
clawpro-design-system-showcase 展示仓库 = 构建产物 / 导出的 PDF
```

日常开发只改 `openclaw-enterprise`。  
需要发布时，把 `openclaw-enterprise` 构建成静态网页，再同步到 `clawpro-design-system-showcase` 的 `oa-pages` 分支。

## 10. 注意事项

1. 不要在展示仓库里手改源码；
2. 不要把构建产物提交回 `openclaw-enterprise` 主仓库；
3. 如果新增了 `查看应用页面` 的新路径，需要同步更新构建入口里的路由表和 fallback index 列表；
4. 如果展示台中新增了图标或组件，确保相关依赖已正确 import；
5. 如果线上访问异常，优先检查：
   - OA Pages 日志：`https://pages.woa.com/logs`
   - OA Pages 权限：`https://pages.woa.com/admin`
   - 展示仓库 `oa-pages` 分支是否更新；
   - `CNAME` 是否仍是 `clawpro-design-system.pages.woa.com`。
6. 如果「已 force-push 但线上不更新 / 更新时间不变」，先去 `https://pages.woa.com/admin` 看该站点「占用空间」是否标红超 100MB——OA Pages 按整个 git 仓库（含 `.git` 历史）算配额，历史堆积旧构建会撑爆配额导致部署停滞，处理方式见第 12 节。

## 11. 常用信息速查

| 项目 | 内容 |
|---|---|
| 主仓库 | `/Users/miekoyychen/openclaw-enterprise` |
| 展示仓库 | `/Users/miekoyychen/CodeBuddy/clawpro-design-system-showcase` |
| 展示仓库远程 | `https://git.woa.com/miekoyychen/clawpro-design-system-showcase.git` |
| 发布分支 | `oa-pages` |
| 构建产物目录 | `/Users/miekoyychen/CodeBuddy/clawpro-deploy/clawpro-design-system-only` |
| 线上地址 | `https://clawpro-design-system.pages.woa.com` |
| OA Pages 管理 | `https://pages.woa.com/admin` |
| OA Pages 日志 | `https://pages.woa.com/logs` |

## 12. OA Pages 100MB 配额与 git 仓库瘦身（重要经验）

> 本节记录 2026-07-01 一次「发布不更新」故障的排查与根治过程，后续遇到类似情况直接照此处理。

### 12.1 现象

- 后台 `https://pages.woa.com/admin` 中 `clawpro-design-system` 一行：
  - **「占用空间」标红、超过 100MB**（当时约 103MB）；
  - **「更新时间」停在旧日期**，force-push 了新内容也不更新上线。
- 页面刷新（含 `Cmd+Shift+R`）仍是旧内容。

### 12.2 根因

OA Pages 走 git 建站，**配额是按整个 git 仓库存储算的，包含 `.git` 历史，而不是只看当前工作区文件**。

实测数据：
- 工作区文件（不含 `.git`）只有约 66MB，本身没超；
- 但 `.git` 里有约 86MB 松散对象：历史每次发布都用普通 commit，累积了**十几份 5~7MB 的旧 `covibe-index-*.js` 构建**和几个 mp4，把配额撑到 103MB；
- 配额超限后，平台停止同步 / 部署，于是「发布了但线上不更新」。

关键结论：**这种情况下单纯「删几个文件再 commit」没用**——旧的大 blob 还留在历史里，新增 commit 反而让 `.git` 更大。必须清理历史。

### 12.3 根治方案：孤儿提交做历史重置

思路：用一个全新的 orphan commit 重建 `oa-pages`，只保留当前这一份产物，历史清零。

```bash
D=/Users/miekoyychen/CodeBuddy/clawpro-design-system-showcase
cd "$D"

# 1) 基于当前工作区建孤儿分支（无历史）
git checkout --orphan _clean
git add -A
git commit -m "deploy: 展示台全量发布（孤儿快照, 历史重置以满足100MB配额）"

# 2) 让 oa-pages 指向该提交
git branch -f oa-pages _clean
git checkout oa-pages
git branch -D _clean

# 3) 清掉本地所有旧引用 + 旧远程跟踪 ref（否则会钉住旧对象，gc 回收不掉）
git update-ref -d refs/remotes/origin/oa-pages 2>/dev/null
git reflog expire --expire=now --all
git gc --prune=now --aggressive

# 4) 实测可达体积，确认低于 100MB
git count-objects -vH | grep '^size-pack'

# 5) 覆盖远程（会重写远程历史，这是预期行为）
git push -f origin oa-pages
```

实测效果：git 可达体积从 ~103MB 降到 **43MB**，单个孤儿提交，无任何旧构建残留，配额恢复正常，线上随即更新。

### 12.4 顺带剔除无引用的大素材（方案 A）

历史重置是大头；可再叠加清理 `landing-assets` 里已无引用的旧素材。删前务必用**全路径 + basename 双重核查**确认 0 引用，避免删错破页面：

```bash
# 全路径 + basename 两种方式都要查 html/js/css
grep -rl "文件名" . --include="*.html" --include="*.js" --include="*.css" | grep -v "^./.git/"
```

本次确认可删的 0 引用文件约 9MB（`banner.mp4`、`yh-features/banner-bg.png`、`yh-features/banner-bg.mp4`、`banner/channels-illust.mp4`、`banner/multi-model.png`）。
在用的资源全部保留：`onboarding/admin-guide.mp4`(13M)、`onboarding/tenant-guide.mp4`(5.8M)、`banner/login-bg.mp4`(3.7M)、`yonghui/211.svg`、`yonghui/217.svg` 等（这些即使 basename 复核也有命中引用，不能删）。

### 12.5 一个附带好处：git 建站无 4.5MB 单文件限制

之前担心 `admin-guide.mp4`(13M)、`tenant-guide.mp4`(5.8M) 这类大文件传不上去——那是 API/上传方式才有的 4.5MB 单文件限制。**走 git（push oa-pages）没有这个限制**，大 mp4 可以正常一并部署。

### 12.6 验证与兜底

force-push 后到 `https://pages.woa.com/admin` 看 `clawpro-design-system` 一行：
- 「占用空间」应明显下降、不再标红；
- 「更新时间」应更新为当天。

若占用没降 / 时间不变，说明平台侧没自动触发同步，可在后台点一次「重新部署」，或联系管理员（如 ivanpeng / xcatliu）手动触发一次同步。

### 12.7 日常预防

- 发布前顺手看一眼 `git count-objects -vH` 的 `size-pack`，接近 100MB 就提前做一次历史重置；
- 不要把大体积一次性素材反复替换后用普通 commit 累积；
- 展示仓库本就是「构建产物」，历史价值低，**定期用孤儿提交重置历史是可接受且推荐的做法**。

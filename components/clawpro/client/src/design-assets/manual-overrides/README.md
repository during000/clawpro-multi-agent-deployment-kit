# manual-overrides · 人工事实源

本目录存放**无法由脚本可靠自动判定、必须由人（设计/PM）拍板**的分类结论。
分类脚本（`classify-resources.mjs`）以这里的文件为事实源回填，**未登记者一律保留待细分，绝不臆测**。

## gradient-style.json — 渐变图标「线条 / 块状」分拣

- **为什么需要人工**：105 个 `visualType=gradient` 的图标全部把渐变用在 `fill` 上，
  SVG 内容里没有能可靠区分「线条多彩渐变 vs 块状多彩渐变」的确定性信号（stroke/path 数/evenodd/rect 均不分层）。
  「线条 / 块状」是**人眼视觉风格**差异，因此交人工点选。
- **怎么产出**：
  1. 启动 dev server：`pnpm dev`
  2. 浏览器打开 `http://localhost:5173/_gradient-sorter.html`（端口以 vite 实际输出为准）
  3. 逐个点选每张缩略图的「线条 / 块状」，进度自动存 localStorage（刷新不丢）
  4. 点「下载 gradient-style.json」，把文件覆盖保存到
     `client/src/design-assets/manual-overrides/gradient-style.json`
  5. 重跑分类流水线：`node client/src/design-assets/scripts/classify-resources.mjs`
- **分拣页怎么来**：由生成器脚本产出，读最新 inventory，保证与数据同步：
  `node client/src/design-assets/scripts/build-gradient-sorter.mjs`
- **字段**：`overrides[资源id] = { style: "gradient-line" | "gradient-solid", sourcePath, displayName }`
  - `gradient-line` → 线条多彩渐变
  - `gradient-solid` → 块状多彩渐变

## duplicate-review.json — 重复组人工确认（C 类闭环出口）

- **为什么需要人工**：`detect-duplicates.mjs` 对红线（品牌 Logo / 渠道图标 / Agent 头像）重复组、
  以及文件名疑似重复组一律判 C 类「待人工确认」，不凭 hash 自动归并。是否同一资源、该并该删该保留，
  必须由人（设计 / PM）拍板。本文件是这些组的**最终裁决事实源**，给 C 类一个「确认完成」的闭环出口。
- **怎么产出**：在页面「重复治理结果」里核对某组后，把该组**全部成员 id** 登记到本文件 `confirmations`，
  并置 `confirmed: true` 与决策 `decision` / 说明 `note`，然后重跑流水线即可。
- **匹配方式**：以 `memberIds`（组内成员 id 的集合，与顺序无关）精确匹配重复组，因此对组 id（`dup-00x`）
  重排稳定；只要成员文件不改名 / 不移动，id 即稳定。
- **字段**：`confirmations[] = { memberIds: string[], decision, confirmed, confirmedBy?, confirmedAt?, note? }`
  - `confirmed=true` → 该组 `reviewStatus=confirmed`，成员动作 `needs-review` → `reviewed-keep`，
    `apply-governance` 出 `reviewed-confirmed`（页面显示「已人工确认」），移出待人工确认清单。
  - **不在此登记删除等破坏性治理**：需要删文件的，直接删文件后重跑流水线，该重复组成员不足 2 个会自然消失。
- **重跑**：`node client/src/design-assets/scripts/detect-duplicates.mjs` 后
  `node client/src/design-assets/scripts/apply-governance.mjs`。

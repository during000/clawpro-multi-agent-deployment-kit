# ClawPro 资源库重复治理报告

> 本报告由 `client/src/design-assets/scripts/apply-governance.mjs` 自动生成，对应建设计划 §9 阶段 5「重复资源实际治理」。
> 报告内容来自治理事实数据 `client/src/design-assets/generated/resource-governance.generated.json`，请勿手改本文件结论，改口径请改脚本后重跑。

- 生成时间：2026-07-20T04:15:15.265Z
- 运行模式：`dry-run`（演练：仅复核，未删除任何文件）
- 输入基线：duplicates @ 2026-07-20T04:15:15.228Z / inventory @ 2026-07-20T04:04:38.191Z / usage @ 2026-07-20T04:04:38.191Z

## 一、治理边界（本阶段严格遵守）

- 只做 A 类清理（删除「内容完全一致 + 无任何引用线索」的冗余副本，保留 canonical）。
- B 类（内容一致 + 有业务引用）应小范围替换引用后再移除——**本仓库实测 B=0，无此项工作**。
- C 类（红线 / 文件名疑似 / 仅余 registry 事实源副本）**只标待确认，不删除、不替换**。
- 不建 canonical 入口（阶段 6）、不动组件源码、不动 skill（阶段 9）、不做全量迁移。
- registry 事实源（root `icon/`）永不删除；品牌 / 渠道 / 头像红线资源不自动归并。

## 二、治理结果总览

| 指标 | 数量 |
|---|---:|
| 重复组总数 | 1 |
| A 类组 / B 类组 / C 类组 | 0 / 0 / 1 |
| 阶段 4 标记的 remove-redundant 候选 | 0 |
| 复核通过可删冗余副本 | 0 |
| 复核未通过、转待确认 | 0 |
| B 类引用替换 | 0 |
| C 类待确认组 | 0 |
| C 类已人工确认组 | 1 |
| 回收字节数 | 0 |

## 三、删除前独立复核规则（逐候选执行，任一不过即跳过）

- 组分级=A 且组非红线
- 成员 role=duplicate 且 action=remove-redundant
- 成员非红线类目（brand-logo/channel-icon/agent-avatar）、非红线视觉（brand-fixed/avatar-like）
- usageCount=0 且 dynamicDirReferenced=false 且 usage 表无该 id 且 不在 unresolvedRefs 命中 且 父目录不在 dirReferences
- 候选文件在磁盘、同组 canonical 文件在磁盘、且候选与 canonical 现场字节 hash 完全一致

> `usageCount=0` 单独不构成删除依据；删除依据是「无任何引用线索 + 同组 canonical 在场且字节完全一致（内容已被保留）」。所有删除均由 git 跟踪，可回滚。

## 四、A 类清理明细

（无）
## 五、复核未通过 / 转待确认的候选

（无：所有 A 类 remove-redundant 候选均通过独立复核）

## 六、B 类引用替换

本仓库 B 类重复组数量为 0。
阶段 4 已核实：唯一含「>1 被引用成员」的组是渠道红线组 `dup-074`（`channel-wecom.svg` 与 `channel-wecom-app.svg` 字节一致、均被引用），已被红线正确判为 C 而非 B。因此本阶段无 B 类引用替换工作。

## 七、C 类待确认清单（不删除、不替换）

| 组 | 方法 | 类型 | 治理状态 | 成员 | 说明 |
|---|---|---|---|---|---|

## 八、影响范围与回滚

- **影响范围**：删除的全部为「与保留 canonical 字节完全一致、且无任何引用线索」的冗余副本，绝大多数为根 `assets/CodeBuddyAssets/` Figma 导出重复件与 `client/public/icon/X 2.svg` 拷贝；内容均已被同组保留文件承载，运行时与页面引用不受影响。
- **registry 事实源、品牌 / 渠道 / 头像红线资源、被引用资源**：全部保留。
- **回滚方式**：所有删除由 git 跟踪。整体回滚 `git checkout -- assets client/public`，或逐个按上文「回滚」命令复原。
- **治理后重跑**：重跑 `scan-resources → classify-resources → detect-duplicates` 将反映治理后现状（重复组收敛）；本报告与 `resource-governance.generated.json` 为治理事实记录。

## 九、产物清单（阶段 5）

```text
docs/resource-governance-report.md                                       # 本报告
client/src/design-assets/generated/resource-governance.generated.json    # 治理状态数据
client/src/design-assets/scripts/apply-governance.mjs                    # 治理脚本（可重复执行）
已删除的 A 类冗余副本（见第四节，git diff 可审计、可回滚）
C 类待确认清单（见第七节）
```

<!-- CANONICAL:START -->

## 十、canonical 接入记录（阶段 6）

> 本段由 `client/src/design-assets/scripts/build-canonical-assets.mjs` 生成（建设计划 §9 阶段 6 / §5.3）。生成时间：2026-07-20T04:15:15.422Z

### 10.1 入口与纳入口径

- 入口文件：`client/src/design-assets/canonical-assets.ts`（自动生成，仅供当前项目页面层使用）。
- 纳入 14 个 key（brands 1 / channels 6 / avatars 7）：仅「已确认 normal + 业务专属 + 运行时可服务（public webPath）」资源。
- **暂不接入入口的高频资源**：多处使用的 `/icon/*.svg` 与 `empty-aiagent.png` 当前 `status=needs-review`（多属 C 类 keep-multiple 组），按纪律不作为「已确认 canonical」，待其经设计确认转 normal 后再纳入。
- **红线不归并**：`channel-wecom` 与 `channel-wecom-app` 字节一致（dup-074 红线），入口层仍分列 `channels.wecom` / `channels.wecomApp` 两 key，不合并。
- **品牌 Logo 第二物理副本（已清理）**：原 `@/assets/topnav/clawpro-logo.svg`（src，横版带文字 logo，内容不同；全仓无代码引用，TopNav 实际用 `/landing-assets/60.svg`）已于 2026-06-14 删除，现仅保留唯一 canonical `/assets/admin-sidebar/clawpro-logo.svg`。

### 10.2 接入状态

| key | 资源 | 类目 | 重复组 | 是否已接入入口 | 接入处 |
|---|---|---|---|---|---|
| `brands.clawproLogo` | `/assets/admin-sidebar/clawpro-logo.svg` | brand-logo | - | 否（仅建入口） | - |
| `channels.wechat` | `/assets/admin-channel-icons/channel-wechat.svg` | channel-icon | - | 是 | `client/src/pages/admin/ChannelConfig.tsx`<br/>`client/src/pages/preview/SkillMapAbDemo.tsx` |
| `channels.qq` | `/assets/admin-channel-icons/channel-qq.svg` | channel-icon | - | 是 | `client/src/pages/admin/ChannelConfig.tsx`<br/>`client/src/pages/preview/SkillMapAbDemo.tsx` |
| `channels.wecom` | `/assets/admin-channel-icons/channel-wecom.svg` | channel-icon | dup-001 | 是 | `client/src/pages/admin/ChannelConfig.tsx`<br/>`client/src/pages/preview/SkillMapAbDemo.tsx` |
| `channels.wecomApp` | `/assets/admin-channel-icons/channel-wecom-app.svg` | channel-icon | dup-001 | 是 | `client/src/pages/admin/ChannelConfig.tsx`<br/>`client/src/pages/preview/SkillMapAbDemo.tsx` |
| `channels.dingtalk` | `/assets/admin-channel-icons/channel-dingtalk.svg` | channel-icon | - | 是 | `client/src/pages/admin/ChannelConfig.tsx`<br/>`client/src/pages/preview/SkillMapAbDemo.tsx` |
| `channels.feishu` | `/assets/admin-channel-icons/channel-feishu.svg` | channel-icon | - | 是 | `client/src/pages/admin/ChannelConfig.tsx`<br/>`client/src/pages/preview/SkillMapAbDemo.tsx` |
| `avatars.default` | `/assets/avatars/avatar-default.png` | agent-avatar | - | 否（仅建入口） | - |
| `avatars.designer` | `/assets/avatars/avatar-designer.png` | agent-avatar | - | 否（仅建入口） | - |
| `avatars.analyst` | `/assets/avatars/avatar-analyst.png` | agent-avatar | - | 否（仅建入口） | - |
| `avatars.creator` | `/assets/avatars/avatar-creator.png` | agent-avatar | - | 否（仅建入口） | - |
| `avatars.developer` | `/assets/avatars/avatar-developer.png` | agent-avatar | - | 否（仅建入口） | - |
| `avatars.pm` | `/assets/avatars/avatar-pm.png` | agent-avatar | - | 否（仅建入口） | - |
| `avatars.operator` | `/assets/avatars/avatar-operator.png` | agent-avatar | - | 否（仅建入口） | - |

### 10.3 一改多处生效与迁移说明

- **已接入**（6 key）：`channels.wechat`、`channels.qq`、`channels.wecom`、`channels.wecomApp`、`channels.dingtalk`、`channels.feishu`。修改 `canonical-assets.ts` 中其路径会统一影响所有 import 处。
- **仅建入口、暂未接入**（8 key）：现有散落引用保持原样，可由页面作者按需增量接入；**不做全量迁移**（边界）。
- **页面层接入示范**：`client/src/pages/admin/ChannelConfig.tsx`、`client/src/pages/preview/SkillMapAbDemo.tsx` 已改用 `canonicalAssets.channels.*`。
- **已治理重复资源的引用迁移**：无需替换——阶段 5 实际移除的 127 个 A 类冗余副本均无任何引用线索，故无页面层 / 非组件源码引用需迁移。
- **组件源码**：`AgentAvatar.tsx`（`ROLE_AVATAR`）、`TopNav` 等组件内资源引用按共享组件处理，本阶段只记录风险、不改源码、不引入口（组件迁移如确有必要须按阶段 8 单独立项评估）。

<!-- CANONICAL:END -->

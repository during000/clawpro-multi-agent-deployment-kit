# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，采用逐题 grilling；未达成共同理解前不进入实现。

---

## 背景

当前用户端 Agent 详情页能展示与安装技能，但后端接口不能完整支撑已安装技能的版本展示、更新和卸载闭环。

TAPD：`1020422209135599722`【clawpro】用户端-支持对公共技能的更新和卸载。

## 已确认事实

- 仅实现后端；现行前端代码位于 `../openclaw-enterprise-fronted`。
- CVM Agent 当前通过 `GET /openclaw/skills` 查询技能，通过 `POST /openclaw/add-skill` 安装公共技能。
- Local Agent 现有安装/ACK 链路存在，但已明确不在本需求范围。
- 当前 CVM 列表契约不足以判断更新：三个运行时的列表均未统一返回当前版本。
- 现有 `/openclaw/skillstore/uninstall` 面向另一套技能库操作语义，不能直接满足“用户本人、单 CVM、同步完成”的接口契约。
- Admin 分发脚本把 Hermes 技能安装到 `$HOME/.hermes/skills/<slug>` 直接子目录；本需求只需为这些 Admin 下发目录提供稳定 slug，分类目录中的其他技能发现问题不在本期修复范围。
- OpenClaw 现有列表已通过 CLI 完成多根技能发现；本需求不依赖运行时文件中的来源或版本元数据。
- 三种运行时的 Admin 下发技能版本统一以分发 task/record 为事实来源，物理安装和删除继续走现有 Agent 类型脚本。
- 管理端通过 `/admin/skills/distribute` 下发 public/enterprise 技能时已有完整任务记录：task 保存 `source/slug/version`，record 按实例保存具体 `version/status/type`；公共 latest 会先解析为确定版本再入库。
- 对管理端下发且成功的技能，当前版本应优先从最新有效分发记录推导，不需要依赖新增文件元数据，因而可兼容已有历史记录；record 必须 JOIN task 才能区分 public/enterprise 与 slug。
- Public 当前版本不能直接取最新一条 record.version：pending/failed 更新记录保存的是目标版本，实际仍运行上一成功版本。正确规则是“最后一次成功卸载之后的最后一条成功 distribute”；pending/failed 操作只影响操作状态，不改变当前版本。

## 目标

- 在现有 `GET /openclaw/skills` 中，为实际存在且由 Admin 下发的 Public/Enterprise 技能补齐当前版本、最新版本和可更新状态。
- 为用户本人拥有且处于 running 状态的 CVM 实例提供单技能同步更新与物理卸载能力。
- 复用 Admin Public/Enterprise 分发的版本解析、制品准备、Agent 脚本、分布式锁和 task/record 状态语义，不建立第二套分发链路。

## 范围

| 包含 | 不包含 |
|------|--------|
| CVM Agent（OpenClaw、Hermes、LightClaw ACE）中由 Admin 下发且有分发记录的 public/enterprise 技能查询、更新与卸载 | Local Agent；用户自行安装、Agent 自带或手工复制的技能；前端交互、GuidePointBubble、本地存储 |

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 本次后端需覆盖哪些 Agent 来源/运行时 | 已确认 | 仅覆盖 CVM OpenClaw、Hermes、LightClaw ACE；Local Agent 不支持本需求 |
| 2 | 卸载对象是否按技能来源限制 | 已确认（Commit 后扩展） | 三种支持运行时中实际存在、有稳定 slug 且能由对应 Agent 脚本实际移除的技能均允许卸载；Admin 下发的 Public/Enterprise 技能额外维护 task/record |
| 3 | 历史技能缺少版本元数据时如何处理 | 已确认（范围收敛后不适用） | 版本管理仅依赖 Admin 分发记录；无 Admin 下发记录的技能允许卸载但不补造版本 |
| 4 | 企业技能是否允许用户端更新 | 已确认（已更正） | Public 与 Enterprise 均支持更新；Enterprise 最新版本取内部 `skills` 表同 slug 最新版本 |
| 5 | 卸载的产品语义 | 已确认 | 必须物理删除技能文件；不以 `enabled=false` 代替卸载 |
| 6 | 是否纳入用户自行安装、Agent 自带或手工复制的技能 | 已确认（Commit 后扩展） | 不纳入版本管理和更新；列表通过 `can_uninstall` 告知前端能力，无 Admin 下发记录时按 slug 直接执行卸载脚本且不创建分发记录 |
| 7 | 是否新增独立管理列表接口 | 已确认 | 不新增；扩展现有 `GET /openclaw/skills`，所有运行时技能返回稳定 `slug`，仅 Admin 下发技能补充版本与更新字段 |
| 8 | Public 最新版本如何获取 | 已确认 | 列表请求按需查询 Admin 公共下发所使用的同一公共仓库；成功结果按 slug 缓存一段时间供后续请求复用，不做后台定时轮询 |
| 9 | Public 最新版本缓存 TTL | 已确认 | 成功结果缓存 5 分钟 |
| 10 | Public 最新版本查询失败如何降级 | 已确认 | 列表保持成功；有过期缓存则返回旧值，无缓存则仅返回当前版本并令 `latest_version` 为空、`update_available=false` |
| 11 | `GET /openclaw/skills` 新增字段 | 已确认（Commit 后扩展） | 所有运行时技能返回稳定 `slug` 和 `can_uninstall`；仅 Admin 下发技能增加 `version`、`latest_version`、`update_available` |
| 12 | Enterprise 版本字段与更新能力 | 已确认（已更正） | 返回 `version`、内部 `latest_version`、`update_available`，有新版时允许更新 |
| 13 | 技能更新接口 | 已确认（Plan 阶段更正） | 新增 `POST /openclaw/update-skill`，以实例 `id` 和技能 `slug` 定位当前有效分发记录并沿用分发记录链路 |
| 14 | 技能更新执行模式 | 已确认 | CVM 单实例同步执行并返回最终成功/失败；成功响应包含旧版本与新版本，不引入前端轮询 |
| 15 | 技能卸载接口 | 已确认（Commit 后扩展） | 新增同步 `POST /openclaw/uninstall-skill`，以实例 `id` 和运行时 `slug` 定位技能并物理删除；Admin 下发技能成功后写 uninstall success，其他技能不创建 task/record |
| 16 | 已是最新版时更新接口语义 | 已确认 | HTTP 200 幂等成功，返回 `updated=false`、当前版本和最新版本，不重复安装 |
| 17 | 技能已卸载时再次调用卸载接口 | 已确认（性能收敛后更正） | 卸载脚本本身幂等；脚本成功统一返回 `uninstalled=true`，不为区分“本次是否删除”增加远程列表查询 |
| 18 | 同一实例、同一技能发生并发更新/卸载 | 已确认 | 使用分布式互斥锁；后到请求立即返回 HTTP 409，不排队、不并发执行 |
| 19 | 用户端操作与 Admin 分发的锁范围 | 已确认（Commit 后扩展） | Admin 下发技能共用现有锁：Public 按 source+slug、Enterprise 按 skill ID；其他技能按 runtime+slug 锁定 |
| 20 | 同一实例存在同 slug 的 Public/Enterprise 历史记录 | 已确认 | 合并两个来源的成功记录按操作顺序还原物理状态；最后一次成功 distribute 决定当前来源/版本，之后的成功 uninstall 清除已安装状态 |
| 21 | 同步更新/卸载执行失败后的记录 | 已确认（Commit 后扩展） | Admin 下发技能保留失败 task/record 并沿用现有终态；其他技能不创建记录，脚本失败直接返回错误 |
| 22 | Public 更新时最新版本查询失败 | 已确认 | 5 分钟内有效缓存可用；缓存过期且刷新失败时不创建任务、不执行安装，返回可重试错误并保留原版本 |
| 23 | 更新/卸载时实例状态准入 | 已确认 | 沿用用户安装技能规则，仅 running 的 CVM 实例允许操作；其他状态直接拒绝且不创建任务记录 |
| 24 | Enterprise 更新时的用户授权 | 已确认 | 更新前重新校验最新版本对当前用户可见；失去授权时禁止更新但仍允许卸载现有文件 |
| 25 | 更新接口成功响应字段 | 已确认（Review 后精简） | 固定返回 `slug`、`updated`、`old_version`、`version`；`latest_version` 与同步完成后的 `version` 重复，不在更新响应返回 |
| 26 | 卸载接口成功响应字段 | 已确认（Commit 后扩展） | 固定返回 `slug`、`uninstalled`；Admin 下发技能另返回已知 `version`，其他技能不补造未知版本 |
| 27 | 更新/卸载请求编码 | 已确认（Plan 阶段更正） | 使用 `application/x-www-form-urlencoded`，字段统一为 `id`、`slug` |
| 28 | Agent 实时列表与分发记录不一致 | 已确认（性能收敛后更正） | 不按数据库补造技能；实时列表决定展示与 `can_uninstall`，无 Admin 下发记录时按请求 slug 直接执行卸载脚本，分发记录只补充 Admin 下发版本字段 |
| 29 | `update_available` 的版本比较语义 | 已确认 | 按项目现有 `x.y.z` 数字版本规则，仅 `latest_version > version` 时为 true；版本相等或当前版本更高时为 false |
| 30 | 显示名与稳定标识不一致时如何定位技能 | 已确认（Commit 后扩展） | Agent `name` 仅用于显示；列表、更新、卸载请求和响应统一使用运行时目录 `slug` |
| 31 | 前端如何在调用前判断能否卸载 | 已确认（Commit 后扩展） | 列表增加 `can_uninstall`；命中用户可管理技能目录时为 true，未命中目录的运行时内建技能为 false |

## 验收口径

### 技能列表

1. OpenClaw、Hermes、LightClaw ACE 的实时列表仍决定实际展示项；不得仅凭数据库记录补造技能。
2. 所有运行时技能返回目录 `slug` 和布尔值 `can_uninstall`；命中当前有效 Admin 下发记录的 Public/Enterprise 技能另返回 `version`、`latest_version`、`update_available`。
3. `can_uninstall` 以是否命中用户可管理技能目录为准：OpenClaw/ACE 未命中目录的内建技能为 false，Hermes 直接目录扫描结果为 true；未命中 Admin 下发记录的技能不补造版本或更新状态，数据库中存在但运行时缺失的技能不补造。
4. 当前版本由同实例、同 slug、跨 Public/Enterprise 的成功操作序列还原：最后一次成功 distribute 生效，后续成功 uninstall 清除状态；pending/failed 不改变当前版本。
5. Public 最新版本按需查询公共仓库，5 分钟成功缓存按 slug 共享并合并同 slug 并发请求；列表刷新失败时允许使用过期缓存，无缓存则返回空 `latest_version` 和 `update_available=false`。
6. Enterprise 最新版本取内部 `skills` 表同 slug 的最高版本。
7. 仅当合法 `latest_version` 严格高于 `version` 时，`update_available=true`。

### 技能更新

1. `POST /openclaw/update-skill` 接收表单字段 `id`、`slug`，校验登录、实例归属、CVM 来源、Agent 能力和 running 状态。
2. 后端根据当前有效分发记录确定 Public/Enterprise 来源，不接受前端指定来源或目标版本。
3. Enterprise 更新前校验最新版本对当前用户可见；Public 更新只使用未过期缓存或本次成功查询到的确定版本。
4. 已是最新版时不执行脚本、不创建 task/record，HTTP 200 返回 `updated=false`。
5. 需要更新时共用 Admin 技能级分布式锁，复用 Admin 制品准备和安装脚本，单实例同步等待最终结果。
6. 成功固定返回 `slug`、`updated`、`old_version`、`version`；失败保留失败 task/record，旧成功版本继续生效。

### 技能卸载

1. `POST /openclaw/uninstall-skill` 接收表单字段 `id`、`slug`，采用与更新相同的身份、归属、能力和状态校验。
2. 后端先按 Admin 下发状态定位；无有效下发状态时获取 runtime+slug 锁并直接执行对应 Agent 卸载脚本，不查询实时技能列表。
3. 卸载脚本成功统一返回 `uninstalled=true`；重复请求依赖脚本幂等，不区分本次是否实际删除目录。
4. Admin 下发技能写入 uninstall task/record 并返回已知 `version`；其他技能不创建 task/record、不返回未知版本。

## 未决问题

无。31 项产品与接口决策均已确认。

## 约束与依赖

- 写接口必须校验登录、实例归属与操作状态，并登记审计规则。
- 用户可见错误使用 i18n。
- API 变更必须更新 `docs/API.md` 并有集成测试覆盖。

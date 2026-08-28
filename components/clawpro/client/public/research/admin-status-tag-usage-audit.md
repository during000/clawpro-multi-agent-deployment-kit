# 管控端 StatusTag 使用现状拉取

> 统计范围：`client/src/pages/admin/**/*.tsx`
> 组件来源：`client/src/components/ui/status-tag.tsx`
> 统计时间：2026-05-25

## 1. 当前结论

- 管控端共 **19 个模块 / 84 处** 使用 `StatusTag`。
- 使用最集中的模块：`技能配置`（20 处）、`成员管理`（12 处）、`OpenClaw 监控`（12 处）、`镜像管理`（7 处）。
- 现有 `variant` 基本收敛在 `green / blue / gray / red` 四类，符合当前组件能力。
- 主要不统一点不是色值，而是 **dot 的使用边界** 和 **StatusTag 被混用于状态标签与信息标签**。

## 2. 推荐统一规则

| 场景 | 推荐写法 | 说明 |
|---|---|---|
| 成功 / 正常 / 开启 / 生效 | `variant="green" dot` | 真实状态，建议使用 mode="dot" |
| 进行中 / 创建中 / 加载中 / 正在提醒 | `variant="blue" dot` | 真实进行态，建议使用 mode="dot" |
| 失败 / 异常 / 恶意 / 加载失败 | `variant="red" dot` | 真实异常态，必须使用 mode="dot" |
| 关闭 / 禁用 / 待完成 / 待处理 / 维护中 | `variant="gray" dot` | 真实状态，建议使用 mode="dot" |
| 全部用户 / 公共 / 推荐 / 限免 | `variant="blue"` | 信息标签，使用 mode="fill" |
| 版本号 / 组织名 / 路径 / 数量 `+N` | `variant="gray"` | 信息标签，使用 mode="fill" |
| 未开通 / 未启用 / 未检测 | `variant="gray"` 或按业务是否为状态决定 dot | 如果只是权益/配置说明，使用 mode="fill" |

## 3. 管控端模块清单

| 模块 | 路由 / 入口 | 文件 | 数量 | 当前用途 |
|---|---|---|---:|---|
| OpenClaw 监控 | `/admin/openclaw-monitor` | `client/src/pages/admin/OpenClawMonitor.tsx` | 12 | 实例生命周期、异常/处理中状态、路径、服务状态、主备模型 |
| 成员管理 | `/admin/members` | `client/src/pages/admin/MemberManagement.tsx` | 12 | 组织/部门摘要、角色、账号正常/禁用、配置项摘要 |
| 技能配置 - 初始包 | `/admin/skill-config` | `client/src/pages/admin/SkillLibrary/SkillInitialPackageTab.tsx` | 11 | 已添加、版本号、应用范围、来源、生效中 |
| 技能配置 - 角色设定 | `/admin/skill-config` | `client/src/pages/admin/SkillRolesTab.tsx` | 9 | 来源、已添加、版本号、应用范围 |
| 镜像管理 - Agent 类型表 | `/admin/image-management` / `/admin/agent-types` | `client/src/pages/admin/ImageManagement/AgentTypesTable.tsx` | 4 | 用户端首选、自定义内核、兼容版本、数量 |
| 镜像管理 | `/admin/image-management` | `client/src/pages/admin/ImageManagement.tsx` | 3 | 应用范围、组织数量 |
| 模型配置 | `/admin/model-config` | `client/src/pages/admin/ModelConfig.tsx` | 4 | 全部用户、组织范围、更多数量 |
| 平台策略 | `/admin/platform-policy` | `client/src/pages/admin/PlatformPolicy.tsx` | 4 | 当前策略值、开启/关闭 |
| 安全组 | `/admin/security-group` | `client/src/pages/admin/SecurityGroupManagement.tsx` | 4 | 云端/本地规则启停 |
| 基础信息 | `/admin/basic-info` | `client/src/pages/admin/BasicInfo.tsx` | 3 | 已完成、待完成、配置类型 |
| 基础信息（标准模式） | `/admin/basic-info` | `client/src/pages/admin/standard/StandardBasicInfo.tsx` | 3 | 已完成、待完成、配置类型 |
| 审计日志 | `/admin/audit-log` | `client/src/pages/admin/AuditLog.tsx` | 2 | 成功 / 失败 |
| Agent 工具库 | `/admin/agent-tool-library` | `client/src/pages/admin/AgentToolLibrary.tsx` | 2 | 未开通、试用中 |
| 批量升级提醒 | `/admin/openclaw-monitor` 内嵌 | `client/src/pages/admin/BatchUpdateNotice.tsx` | 2 | 新版本上线、正在提醒员工更新 |
| 文件管理 | `/admin/file-management` | `client/src/pages/admin/FileManagement.tsx` | 2 | 免费、未启用 |
| 技能详情 | `/admin/skill-detail/1` | `client/src/pages/admin/SkillLibrary/SkillDetail.tsx` | 2 | 未检测、检测中 |
| 技能更新弹窗 | `/admin/skill-config` 触发 | `client/src/pages/admin/SkillLibrary/SkillUpdateDialog.tsx` | 2 | 限免、未开通 |
| 技能上传弹窗 | `/admin/skill-config` 触发 | `client/src/pages/admin/SkillLibrary/SkillUploadDialog.tsx` | 2 | 限免、未开通 |
| Tokens 监控 | `/admin/tokens-monitor` | `client/src/pages/admin/TokensMonitor.tsx` | 1 | 当前版本 |

## 4. 现有样式组件

`StatusTag` 当前规格：

- 高度：`h-5`（20px）
- 圆角：`rounded-full`
- 间距：`gap-1 px-2 py-[2px]`
- 字号：`text-xs`
- dot：`w-1.5 h-1.5 rounded-full`

当前四种 `variant`：

| variant | 背景 | 文字 / dot | 推荐语义 |
|---|---|---|---|
| `green` | `#E9F8EB` | `#008236` | 成功、正常、开启、生效 |
| `blue` | `#E8ECFE` | `#1447E6` | 进行中、全部用户、推荐/提示 |
| `gray` | `#F5F5F5` | `#0A0A0A` | 待处理、关闭、版本、范围、未开通 |
| `red` | `#FEF2F2` | `#DC2626` | 失败、异常、风险 |

## 5. 后续统一建议

1. **先统一使用语义**：把 `StatusTag` 分为“状态标签”和“信息标签”。状态标签使用 mode="dot"，信息标签使用 mode="fill"。
2. **再沉淀状态映射**：高频枚举建议在模块内抽 `STATUS_TAG_MAP`，避免每处手写判断。
3. **避免覆盖组件样式**：除 `truncate / absolute / max-w` 等布局类外，不建议覆盖高度、字号、颜色。
4. **如需新增颜色先评审**：当前四类足够覆盖管控端主要场景，暂不建议新增黄/紫等变体。

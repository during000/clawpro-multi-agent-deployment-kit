# 02. Plan — 方案设计

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `.specs/plans/2026-07-09_clawpro-agent-admin-create-assign/00-overview.md`～`08-commit.md` | 新增 | 按阶段模板记录任务元信息和交付证据 |
| `controller/admin_instance_create.go` | 新增 | 管理员 JSON 请求解析、目标用户/分组选择、presets 校验和响应 |
| `controller/admin_instance_create_test.go` | 新增 | 管理员创建、分配、校验、成功路径、审计和 i18n 测试 |
| `controller/openclaw_create_service.go` | 新增 | 等待 Agent ready，并调度 channel presets |
| `controller/openclaw.go` | 修改 | 抽取 user/admin 共用的 `createInstance(..., createInstanceOptions)` 主流程 |
| `controller/openclaw_model.go` | 修改 | 抽取 `applyBuiltinModel`，统一创建期和用户手动模型行为 |
| `controller/openclaw_channel.go` | 修改 | 抽取手动通道校验/执行 helper，并承载创建期调度 |
| `controller/openclaw_skill.go` | 修改 | 统一 public/enterprise 技能来源和企业技能安装 helper |
| `controller/access_log.go` | 修改 | 管理员创建请求体不记录，敏感通道字段脱敏 |
| `controller/audit.go` | 修改 | 注册 `instance_admin_create` 审计规则并排除敏感 body |
| `controller/tat.go` | 修改 | TAT 参数值和 set-channel 脚本失败输出不进入日志 |
| `main.go` | 修改 | 注册 `POST /admin/instances/create` 路由 |
| `i18n/keys.go` | 修改 | 新增管理员创建校验消息 Key |
| `i18n/en.go` | 修改 | 补齐新增 Key 的英文翻译 |
| `controller/access_log_log_rcv_request_test.go` | 修改 | 覆盖请求体、TAT 参数和脚本失败输出脱敏 |
| `controller/openclaw_channel_test.go` | 修改 | 覆盖创建期通道与手动通道操作复用及执行顺序 |
| `controller/openclaw_handler_writectx_test.go` | 修改 | 适配共享创建主流程的响应职责 |
| `controller/openclaw_model_test.go` | 修改 | 覆盖创建期与手动添加模型的排序和回滚语义 |
| `controller/openclaw_multi_model_test.go` | 修改 | 覆盖单模型 runtime 与 OpenClaw fallback 能力矩阵 |
| `controller/openclaw_skill_extended_test.go` | 修改 | 覆盖 public/enterprise 追加技能及无持久化语义 |
| `docs/API.md` | 修改 | 记录请求、响应、能力矩阵、异步语义和敏感信息约束 |
| `test/scripts/helpers/client.py` | 修改 | 支持严格 JSON/原始响应等集成校验所需调用 |
| `test/scripts/openclaw_instance/test_admin_instance_create.py` | 新增 | 部署环境下的鉴权、JSON、字段和密钥不回显校验 |

## 调用链 / 数据流

```text
POST /admin/instances/create
  -> requireAdmin + WithAudit(instance_admin_create)
  -> HandleAdminCreateInstance
     -> 严格 JSON 解码
     -> 解析目标用户和有效分组
     -> 校验配额、agent_type、role、models、channels、skills
     -> createInstance(w, r, &targetUser, createInstanceOptions)
        -> 与 POST /openclaw/create 共用 CVM/网络/安全组/实例落库主流程
     -> HTTP 200 {ok, instance_id}
     -> goroutine 等待 Agent ready
        -> models: primary -> fallbacks，逐个 applyBuiltinModel
        -> channels: 请求顺序逐个 applyManualChannelConfig
        -> skills: 请求顺序逐个执行 public/enterprise 安装 helper
```

关键不变量：

- 申请 CVM 前完成所有可同步完成的校验；失败时不产生计费实例。
- 指定 models 时覆盖站点默认模型；未指定时保持原默认模型逻辑。
- presets 不改变基础创建响应；HTTP 200 不代表异步下发完成。
- channel config 和 enterprise 下载 URL 不持久化、不回显、不进入结构化日志。

## 数据库变更

无。复用现有用户、分组、实例、模型绑定和技能数据；不新增表、字段或 migration。创建期 channel 配置和追加技能意图不持久化。

## 测试用例设计（自然语言描述）

> 先于实现编写，Implement 阶段据此编码。
> UT 用例走 `go test`，IT 用例走 Python 集成测试（`test/scripts/`）。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 管理员鉴权 | 无 token、普通用户 token、管理员 token | 分别返回 401、403，并允许管理员进入业务校验 | P0 |
| 2 | 严格 JSON 和方法校验 | 非法 JSON、尾随 JSON、未知字段、缺必填字段、GET | 返回 400 或 405，不调用 CVM | P0 |
| 3 | 目标用户和分组选择 | 用户不存在/软删；0、1、多个有效分组；跨用户或待删除分组 | 按规则自动选择或拒绝非法分组 | P0 |
| 4 | 用户/分组配额 | 已达到实例配额 | 返回 403，不调用 CVM | P0 |
| 5 | 模型校验和能力矩阵 | 不存在、禁用、不可见、重复、单模型 runtime fallback、OpenClaw 3.28 fallback | 创建前返回 400；受支持组合通过 | P0 |
| 6 | 通道校验 | 缺参数、空键值、重复、不可见、不受 runtime 支持 | 创建前返回 400 | P0 |
| 7 | 技能校验 | source 非法、企业技能不存在/不可见、默认 public、版本和去重 | 非法输入拒绝，合法输入正确解析 | P0 |
| 8 | 完整创建成功 | 合法管理员、目标用户、分组及组合 presets | 共享主流程返回 CVM instance_id，并启动异步下发 | P0 |
| 9 | 模型共享行为 | 创建期按 primary/fallback 顺序应用模型 | 与用户 `add-model` 的排序、DB 角色和失败回滚一致 | P0 |
| 10 | 通道共享行为 | 两个合法手动通道 | 按请求顺序进入与用户 `set-channel` 相同的 runner | P0 |
| 11 | 技能共享行为 | public/enterprise 追加技能 | 与用户 `add-skill` 来源语义一致，且不创建安装任务 | P0 |
| 12 | 审计、i18n 和敏感日志 | 创建请求含通道 secret，TAT/set-channel 失败 | 路由有审计规则和英文翻译；secret、TAT 参数值及脚本输出不记录 | P0 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 部署环境鉴权 | 无效凭据、非管理员、管理员 | 无效身份被拒绝，管理员进入参数校验 | P0 |
| 2 | 严格 JSON | 非法 JSON、未知字段 | 返回 400 | P0 |
| 3 | 必填字段 | 分别缺 user_id、name、agent_type | 返回 400，且不申请 CVM | P0 |
| 4 | 完整嵌套请求和密钥不回显 | models/channels/skills 完整请求，目标用户不存在 | 在 RunInstances 前返回 400，响应不含测试 secret | P0 |
| 5 | 方法限制 | GET `/admin/instances/create` | 返回 405 | P0 |

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | user/admin 创建逻辑分叉导致配额或实例初始化不一致 | 中 | 高 | 抽取单一 `createInstance` 主流程，handler 只负责输入和响应 |
| 2 | runtime 模型能力被错误统一为 primary + fallbacks | 中 | 高 | 在创建前按 agent_type/版本校验能力矩阵，并回归用户 `add-model` |
| 3 | 通道凭据通过请求、审计、TAT 或脚本错误日志泄露 | 中 | 高 | 跳过该接口 body，脱敏参数和失败输出，增加负向日志测试 |
| 4 | 异步 presets 在进程退出时丢失 | 中 | 中 | 不承诺终态；文档明确不持久化/不恢复，保留现有手动配置入口 |
| 5 | 异步模型写库和下发状态不一致 | 中 | 高 | 复用 `applyBuiltinModel` 的单次提交、下发失败回滚和停止后续模型语义 |
| 6 | 企业技能绕过目标分组可见性 | 低 | 高 | 创建前和执行时都复用企业技能解析及可见性校验 |

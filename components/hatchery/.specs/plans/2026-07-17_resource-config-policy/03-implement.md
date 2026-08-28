# 03. Implement — 实现记录

---

## 实施范围与基线

本步骤没有重写已经落在 `ceaa1a65 feat: add resource config policy` 中的生产主体。现有站点配置、用户组策略、最近祖先解析、实例创建覆盖、动态 options、路由、审计和 schema 作为首版基线保留；先对照 [02-plan.md](./02-plan.md) 做差距审计，再只修复审计确认的契约与合规偏差。

差距审计确认并关闭的生产问题：

1. 创建路径仍会把小于 50GB 的系统盘静默扩大，且没有比较最终镜像容量与最终系统盘容量。
2. `resource_config`、站点 request wrapper 和用户组 `value_json` wrapper 没有统一保证单个 object、必填字段和尾随 JSON 校验；未知字段按首版兼容要求继续忽略。
3. 站点配置落库错误被忽略；用户组 resolver 的普通错误可能被传给 `EnsureRichErrorOrPanic`。
4. options cache key 缺少 tenant identifier，缓存命中仍返回 `source=tencent_cloud`，query allowlist 不完整。
5. 本功能新增腾讯云调用没有全部经过 `GetXxxClient` 和 `CallSDKAPITyped`。
6. basic options 包含硬编码中文；镜像/系统盘冲突缺少独立中英文文案。
7. migration 文件仍使用与目标 Release 不一致的 `0706-` 前缀。
8. Plan U01–U64 对应的行为测试覆盖不完整。

---

## 关键实现细节

### 1. JSON object 兼容入口

- 在 `controller/resource_config.go` 增加共享 `decodeJSONObject(raw, dst)`：
  - 只接受单个 JSON object；
  - 使用标准 `json.Decoder` 解析预期字段，未知字段保持兼容忽略；
  - 二次 Decode 必须得到 `io.EOF`，拒绝尾随 JSON；
  - 拒绝 `null`、数组、字符串和其他非 object 输入。
- `ParseResourceConfig` 保留内部兼容行为：空串/空白返回非 nil 空配置；非空输入校验单个 object，预期字段继续 Normalize / Validate。
- `HandleUpdateResourceConfig` 要求外层 `resource_config` 必须存在；外层和配置中的未知字段忽略，预期字段规范化、校验并 canonical 落库。
- `HandleSetGroupPolicy` 在 `config_key=resource_config` 分支要求 `value_json={"value":{...}}` 和存在的 `value`；wrapper/value 未知字段忽略，不改变其他历史 policy。
- 用户表单 `resource_config` 与站点/组存量值使用相同兼容解析；格式或预期字段错误返回受控 JSON。

### 2. 镜像与系统盘容量

- 新增纯函数 `validateImageSystemDiskSize(imageSize, disk)`：
  - `ImageSize > 0 && DiskSize < ImageSize` 返回 RichError，包含所需容量和当前容量；
  - 相等、镜像更小、`ImageSize == 0`、`SystemDisk == nil` 或 `DiskSize == nil` 均继续；
  - 函数不修改磁盘结构或容量指针。
- 删除创建路径中 `DiskSize < 50` 时静默改成 50GB 的代码。
- 调用位置固定在 CVMTemplate、站点/组、用户 override、兼容 `disk_type` 和最终 `ImageId` 全部完成之后，且早于 VPC/子网校验、安全组选择、资源预检和 `RunInstances`。
- 新增 i18n：
  - 中文：`所选镜像要求系统盘至少 %dGB，当前选择为 %dGB`
  - 英文：`The selected image requires a system disk of at least %dGB; the current selection is %dGB`

### 3. 站点持久化与用户组 resolver

- `HandleUpdateResourceConfig` 检查 `model.UpdateSiteConfig` 返回值；写失败返回 500，不再伪报 `ok=true`。
- `ResolveResourceConfig` 保持 self → parent → root 最近命中、整份策略生效的首版语义。
- 祖先查询、binding 查询和 wrapper JSON 错误统一包装为 i18n RichError；创建 handler 可安全返回 JSON 500，不再存在普通 error 触发 panic 的路径。
- resolver 要求单个 object wrapper 和存在的 `value`，未知 wrapper 字段兼容忽略；本地、继承、无命中 source 行为保持不变。

### 4. Options cache、SDK 与国际化

- cache key 由 tenant identifier、Region、endpoint type、zone、charge type、instance type组成，避免跨租户复用云账号结果。
- TTL 保持 5 分钟；fresh 返回 `source=tencent_cloud`，cache hit 和 inflight waiter 返回 `source=cache`。
- `refresh=1` 绕过现有缓存；winner 失败不写缓存，waiter 返回受控错误。
- inflight 完成顺序改为先关闭 waiter channel，再 `CompareAndDelete`，消除 delete/close 窗口中的重复 winner；并发失败用例连续运行 20 次通过。
- `instance_charge_type` 和 `instance_type` 在 SDK 调用前校验；云端机型结果再次按 `SELL + 固定 allowlist` 过滤。
- options handler 通过依赖注入的内部 helper 接受 CVM/CBS client，生产入口仍只使用 `GetCVMClient` / `GetCBSClient`，测试不修改生产 fallback。
- 三个 `DescribeZoneInstanceConfigInfos` 和一个 `DescribeDiskConfigQuota` 全部通过 `CallSDKAPITyped`；增加 `SDKComponentCBS = "cbs"`。
- basic options 复用已有 `MsgInternetCharge*Label` i18n key，不再硬编码中文，也没有创建重复翻译键。

### 5. 数据库 migration

- `sql/init.sql` 中 `site_configs.resource_config TEXT NULL` 保持不变。
- 将资源配置与实例调整增量合并为 `sql/0728-resource-management.sql`，保持 DDL 单一来源：

```sql
ALTER TABLE `site_configs` ADD COLUMN `resource_config` TEXT NULL AFTER `c_vm_template`;
```

- 未新增重复 DDL，未修改已有数据迁移语义。

### 6. 测试实现

Plan 的 U01–U64 均已落为可执行测试，按职责分布：

| 用例范围 | 主要文件 | 覆盖重点 |
|---|---|---|
| U01–U20 | `controller/resource_config_test.go` | Parse、JSON object/未知字段兼容、Normalize、Validate、Apply 全字段/部分字段/nil |
| U21–U36 | `controller/resource_config_test.go`、`controller/openclaw_sg_selection_test.go` | 容量纯函数、无静默扩盘、VPC/SG/CVM 前失败、站点/组/用户优先级、resolver 错误 |
| U37–U43 | `controller/admin_config_test.go` | GET 三种 source、POST 未知字段兼容/Normalize/DB 失败、方法与权限 |
| U44–U47 | `controller/admin_group_config_coverage_test.go`、`controller/usergroup/resolve_coverage_test.go` | 组策略 canonical 写入/删除、self/parent/nearest/source、坏 JSON |
| U48 | `controller/usergroup/overview_coverage_test.go` | site/local/inherited 的 JSON object value 与 source |
| U49–U64 | `controller/resource_options_test.go` | zh/en、参数、云结果过滤、cache source、refresh、tenant、TTL、inflight、SDK、preflight |

测试全部使用现有内存数据库、mock Tencent client 或已有 handler hook；不访问生产凭证，不创建真实云资源。

---

## 修改文件

### 生产代码

- `controller/resource_config.go`
- `controller/resource_options.go`
- `controller/openclaw.go`
- `controller/admin_config.go`
- `controller/admin_group_config.go`
- `controller/usergroup/resolve.go`
- `controller/access_log.go`
- `i18n/keys.go`
- `i18n/en.go`
- `sql/0706-resource-config.sql` → `sql/0728-resource-management.sql`

### 测试代码

- `controller/resource_config_test.go`
- `controller/resource_options_test.go`（新增）
- `controller/openclaw_sg_selection_test.go`
- `controller/admin_config_test.go`
- `controller/admin_group_config_coverage_test.go`
- `controller/usergroup/resolve_coverage_test.go`
- `controller/usergroup/overview_coverage_test.go`

---

## 与 Plan 差异

用户在 Implement 结束确认时明确调整 JSON 契约：不启用未知字段严格拒绝，只保证单个 object、必填 wrapper、预期字段解析和业务校验。实现、Clarify、Plan 和测试已同步调整；未知字段现在兼容忽略且不会进入 canonical 持久化结果。

其余生产行为无偏离。

执行层面的两点说明：

1. Plan 将 options description 写作“新增 i18n key”；仓库已经存在语义和 value 完全匹配的 `MsgInternetChargeBandwidthPrepaidLabel`、`MsgInternetChargeTrafficPostpaidLabel`、`MsgInternetChargeBandwidthPostpaidLabel`、`MsgInternetChargeBandwidthPackageLabel`，实现直接复用，避免第二套翻译约定。
2. `.ci/ci-check-schema.sh` 只读取 `HEAD` 中的 migration（`git diff <base>...HEAD` 和 `git show HEAD:<file>`），Implement 阶段尚未提交，因此脚本仍看到 `ceaa1a65` 中的旧 `0706-` 文件。工作区已完成 `0715-` 移动；schema CI 必须在 Commit 产生新 HEAD 后执行并记录。

静态检查说明：`go vet ./...` 通过。仓库没有安装 `staticcheck` 二进制；使用 `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` 后发现 156 条仓库存量告警。落在本次触及文件中的 6 条均是未改动的旧函数/变量未使用告警（`admin_config.go` 的 5 个旧 helper、`openclaw.go` 的旧 `cvmStateMap`）；本步骤新增的 `SA6005` 和 `SA4006` 告警已修复。存量告警不在本需求范围内，未做无关清理。

---

## 验证证据

```text
go test ./controller -run 'DecodeJSONObject|ParseResourceConfig|HandleUpdateResourceConfig|SetGroupPolicy_ResourceConfig' -count=1
PASS: hatchery/controller

go test ./controller/usergroup -run 'ResolveResourceConfig' -count=1
PASS: hatchery/controller/usergroup

go test ./controller -run 'ResourceConfig|ResourceOptions|Image.*Disk|ResourcePreflight|AdminCreate|HandleCreateInstance_U(2[6-9]|3[0-5])' -count=1
PASS: hatchery/controller

go test ./controller/usergroup -run 'ResourceConfig|PolicyOverview|ConfigCategory' -count=1
PASS: hatchery/controller/usergroup

go test ./controller -run 'TestResourceOptions_WinnerFailure' -count=20
PASS: hatchery/controller

go vet ./...
PASS（无输出）

gopls diagnostics：本次生产文件和测试文件均为 OK
```

---

## 检查项

- [x] `gofmt` 格式化通过
- [x] `goimports` 整理通过
- [x] `go vet ./...` 无错误
- [x] 写接口已添加审计日志（复用首版 `WithAudit` 路由和 `auditRules`，本步骤未新增写路由）
- [x] 数据库变更已同步 `sql/init.sql` + 单一 `0715` migration SQL
- [x] 使用 `model.DB(r.Context())` / tenant context，不新增全局 `model.DB`
- [x] 无硬编码密钥/Token/凭证或真实云 fallback
- [x] 用户可见文案使用 `i18n.T()` / i18n RichError，新增 Key 已同步 `en.go`
- [x] 腾讯云 client 使用 `GetXxxClient(ctx)`，新增 SDK 调用使用 `CallSDKAPITyped`
- [x] Plan U01–U64 测试代码已实现，Implement focused smoke 全部通过

---

## 2026-07-21 资源策略实体化重设计实现

### 最终实现

- 新增租户隔离的 `model.ResourcePolicy`，配置唯一存于 `config_json`；企业默认策略固定名称、不可删除、不可改名且无应用范围。
- 默认策略在列表、配置总览或创建 Agent 首次需要时懒创建；从当前 `CVMTemplate` 提取初始资源字段，租户内名称唯一索引和 `ON CONFLICT DO NOTHING` 保证并发幂等，已编辑配置不会被覆盖。
- 应用范围不修改 `user_groups`，也不新增专用关系表；复用 `GroupConfigBinding` 的可索引资源绑定：
  - `config_type=resource_policy`
  - `config_key=<ResourcePolicy.ID>`
  - `group_id=<UserGroup.ID>`
  - `value_json={}`（通用表的空占位，不保存第二份策略）
- 策略→分组通过 `(identifier, config_type, config_key, group_id)` 前缀批量查询；分组→策略通过 `group_id + config_type` 批量查询。更新事务锁定目标用户组，阻止并发把同一组直接分配给多个策略。
- 新增四个管理 API：列表、创建、更新、删除；三个写接口均注册审计并使用 `WithAudit`。
- 用户组树新增 `with_resource_policy=true`，只返回 `direct_resource_policy{id,name}`，不把继承后代标为直接占用。
- `ResolveEffectiveResourcePolicy` 按 self → nearest ancestor → default 返回唯一策略；创建 Agent、资源策略总览和计费模式总览使用同一解析结果，来源为 `local` / `inherited` / `default`。
- 删除 `SiteConfig.ResourceConfig`、旧站点 GET/POST、`PolicyKeyResourceConfig`、分组内嵌完整资源配置分支、旧 resolver 和静态 basic options；用户本次创建的 `resource_config` / `disk_type` overlay 保持。
- 动态 options 路径切换到 `/admin/resource-policies/options/*`。

### 数据库与迁移

- `model/db.go` 注册 `ResourcePolicy`；`model/migrate.go` 迁移策略实体，并在迁移 `GroupConfigBinding` 时重映射 `resource_policy` 的数字 `config_key`。
- `sql/init.sql` 和 `sql/0728-resource-management.sql` 新增 `resource_policies`；没有 `user_groups.resource_policy_id` 或 `resource_policy_groups`。
- 策略配置只保存一次；分组范围复用既有 `group_config_bindings`，无需第二份 JSON 或新关系表。

### API、文档与测试清理

- `docs/API.md` 改为四个策略管理接口、两个动态 options、树参数和新创建优先级；生成的 OpenAPI 包含 6 个 `/admin/resource-policies*` 路径且不含旧 `/admin/config/resource*`。
- `docs/basic/platform-policy.md` 与 `docs/INDEX.md` 和本功能无关，保持 `Release/2026_07_28` 基线内容不变。
- 三个 Resource Config 集成脚本切换到独立策略 CRUD、可索引分组范围、继承和默认策略恢复流程。
- 删除旧站点/分组内嵌策略单测，新增模型并发懒创建、双向范围查询、占用回滚、继承、删除兜底、管理 CRUD、审计和分组树直接策略测试。

### 验证证据

```text
go test ./... -count=1
PASS：9 packages，1 package no tests

go test -race ./model -run '^TestResourcePolicy' -count=1
PASS

go test -race ./controller -run 'TestResourcePolicy|TestGroupTreeReturnsDirectResourcePolicyOnly|TestHandleCreateInstance_U3[1-5]' -count=1
PASS

go vet ./model ./controller ./controller/usergroup .
PASS

python3 -m py_compile \
  test/scripts/admin_resource_config/test_resource_config.py \
  test/scripts/admin_user_groups/test_resource_config_policy.py \
  test/scripts/openclaw_instance/test_exclusive_instance_create_resource_config.py
PASS

make openapi BASE_BRANCH=origin/Release/2026_07_15
PASS：current 383 paths / 393 operations；base 375 paths / 385 operations

gopls diagnostics：
model/resource_policy.go、model/migrate.go、controller/admin_resource_policy.go、
controller/openclaw.go、main.go 均为 OK
```

全仓 `go vet ./...` 仍报告既有 `skillhubclient/client_test.go:278` “using resp before checking for errors”；受影响包定向 `go vet` 无输出。本次未修改该文件。

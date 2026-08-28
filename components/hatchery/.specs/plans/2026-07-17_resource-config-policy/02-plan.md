# 02. Plan — 资源配置策略完整方案设计

> 本文按 `CODEBUDDY.md` 和 `.specs/plans/_template/02-plan.md` 编写，覆盖整个资源配置功能，不只记录本轮新增的镜像容量行为。
> 当前分支已经存在首版实现；本文将“已实现并保留的基线”“必须补齐的功能差距”“SOP/合规修复”放在同一份完整计划中。Plan 阶段不修改生产代码。

---

## 计划依据

| 项 | 内容 |
|----|------|
| 分支 | `feat/resource-config-policy` |
| 目标基线 | `Release/2026_07_28` |
| 当前首版提交 | `ceaa1a65 feat: add resource config policy` |
| Clarify 真相源 | [01-clarify.md](./01-clarify.md) |
| 产品变化 | 镜像大小已知且大于最终系统盘时直接失败；镜像大小为 0 时交给 CVM；禁止静默扩大磁盘 |
| 保持不变 | 站点配置、最近祖先策略、用户完整覆盖、固定 allowlist、选项缓存、机型 fail-closed、模板 fallback、现有 API 契约 |
| 本轮合规要求 | 完整文档、自然语言测试先行、SDK 工厂/调用日志、i18n、数据库双维护、UT/IT/API coverage、Review、规范 Commit |

---

## 目标与非目标

### 目标

1. 提供站点级 `resource_config` 的读取、保存、校验和 `CVMTemplate` fallback。
2. 把 `resource_config` 注册为 JSON 类型的用户组策略，按最近祖先命中并展示来源。
3. 实例创建时按确定优先级生成最终 `cvm.RunInstancesRequest`。
4. 提供基础枚举、可售机型和系统盘选项接口，使用租户隔离的 5 分钟缓存和同 key 并发去重。
5. 创建前验证机型可售性，并在任何安全组选择/扩容之前执行镜像容量检查。
6. 当 `ImageSize > 0 && ImageSize > final DiskSize` 时返回 400，包含所需/当前容量，不修改磁盘。
7. 当 `ImageSize == 0` 或最终请求没有显式磁盘容量时跳过镜像比较，继续调用 CVM。
8. 补齐数据库 migration、SDK 调用规范、i18n、API/模块文档、UT、IT、覆盖率和 Review 证据。

### 非目标

- 不实现前端页面。
- 不改为站点与多级祖先的字段级合并；继续采用最近祖先整份命中语义。
- 不开放腾讯云全部机型；继续使用固定 Ai2 allowlist。
- 不改变 VPC、子网、安全组池、镜像管理和计费购买体系。
- 不为 `ImageSize == 0` 推断默认镜像大小，也不新增测试专用生产 API。
- 不改变已有实例；所有资源配置只影响后续新建实例。

---

## 完整数据契约

### `ResourceConfig` JSON

所有外部 JSON 使用 snake_case。字段均可省略；省略表示该层不覆盖底层请求。

| 字段 | 类型 | 允许值/规则 | 应用到 CVM |
|------|------|-------------|------------|
| `instance_charge_type` | string | `PREPAID` / `POSTPAID_BY_HOUR` | `RunInstancesRequest.InstanceChargeType` |
| `instance_charge_prepaid.period` | int | `PREPAID` 时必填且大于 0 | `InstanceChargePrepaid.Period` |
| `instance_charge_prepaid.renew_flag` | string | 固定 renew flag allowlist | `InstanceChargePrepaid.RenewFlag` |
| `instance_type` | string | `Ai2.MEDIUM2` / `Ai2.MEDIUM4` / `Ai2.LARGE8` | `RunInstancesRequest.InstanceType` |
| `system_disk.disk_type` | string | `CLOUD_SSD` / `CLOUD_PREMIUM` / `CLOUD_BSSD` / `CLOUD_HSSD` | `SystemDisk.DiskType` |
| `system_disk.disk_size` | int | 大于等于平台最低值 50GB；不自动扩大 | `SystemDisk.DiskSize` |
| `internet_accessible.public_ip_assigned` | bool/null | 指针语义：省略不覆盖，显式 false 关闭公网 | `InternetAccessible.PublicIpAssigned` |
| `internet_accessible.internet_charge_type` | string | 固定公网计费 allowlist | `InternetAccessible.InternetChargeType` |
| `internet_accessible.internet_max_bandwidth_out` | int | 依公网计费方式执行范围校验 | `InternetAccessible.InternetMaxBandwidthOut` |

### JSON 兼容校验规则

所有资源配置入口复用同一兼容解析与业务校验逻辑：

1. `resource_config` 必须是单个 JSON object，不接受数组、字符串、`null` 或多个 JSON 值。
2. 未知字段保持向前兼容并忽略；所有预期字段必须正确解析，随后执行 Normalize 和 Validate。
3. 解码后必须继续读取到 `io.EOF`，拒绝尾随 JSON。
4. 站点更新请求必须包含 `resource_config`。
5. 组策略 `value_json` 必须是 `{"value": <ResourceConfig object>}`，`value` 必须存在；wrapper 和 value 中未知字段兼容忽略。
6. 用户创建的 form 字段 `resource_config` 使用同一兼容解析、Normalize 和 Validate。
7. 持久化的历史无效 JSON 在读取时不得 panic；管理读取可按现有兼容行为返回空对象，创建解析错误必须返回受控 500 JSON。

### 存储契约

| 存储 | 字段/键 | 内容 |
|------|---------|------|
| `site_configs` | `resource_config` TEXT NULL | 站点级 snake_case JSON；空时 fallback 到 `c_vm_template` |
| `group_config_bindings` | `config_type=policy`、`config_key=resource_config` | `value_json={"value":{...}}`；现有唯一键保证每组一份 |
| `ai_images` | `image_size` BIGINT | 腾讯云镜像容量，创建时按 GB 与 `SystemDisk.DiskSize` 比较；0 表示未知并跳过比较 |

---

## 配置优先级与继承契约

优先级从低到高：

```text
CVMTemplate 基础 RunInstancesRequest
< 最近命中的用户组 resource_config
  （没有组策略命中时使用站点 ResourceConfig；站点为空时从 CVMTemplate 提取）
< 本次创建请求的 resource_config
< 兼容参数 disk_type（与 resource_config.system_disk.disk_type 不一致时返回 400）
< 后端运行时强制字段（ImageId、Zone、VPC、Subnet、SG、InstanceName、CamRoleName）
```

用户组语义：

- `ResolveResourceConfig(groupID)` 沿 `self → parent → root` 查找。
- 第一份命中的策略整份生效，不与远祖先或站点 `ResourceConfig` 做字段合并。
- 该策略明确提供的字段覆盖 `CVMTemplate`；未提供字段保留 `CVMTemplate`。
- 无命中时才使用站点 `ResourceConfig`。
- 祖先查询或持久化 JSON 解析失败必须返回可国际化的受控错误，不能 panic 或静默回退。

---

## 镜像与系统盘最终行为

### 校验函数

在 `controller/resource_config.go` 增加纯函数，例如：

```go
func validateImageSystemDiskSize(imageSize int64, disk *cvm.SystemDisk) error
```

契约：

| 条件 | 结果 |
|------|------|
| `imageSize > 0` 且 `disk.DiskSize < imageSize` | 返回 i18n RichError，包含镜像所需和当前选择 GB |
| `imageSize > 0` 且 `disk.DiskSize == imageSize` | 通过 |
| `imageSize > 0` 且 `disk.DiskSize > imageSize` | 通过 |
| `imageSize == 0` | 跳过比较，继续 CVM |
| `disk == nil` 或 `disk.DiskSize == nil` | 没有明确选择容量，跳过比较，继续 CVM |

函数不得修改 `disk` 或 `DiskSize` 指针。

### 调用位置

1. 完成 CVMTemplate、站点/组策略、用户 `resource_config` 和兼容 `disk_type` 应用。
2. 设置最终 `ImageId`。
3. 调用 `validateImageSystemDiskSize(enabledImage.ImageSize, request.SystemDisk)`。
4. 失败时返回 HTTP 400；占位实例由现有 defer 清理。
5. 校验必须早于 VPC/子网云调用、安全组选择/扩容和 `RunInstances`。
6. 删除当前 `DiskSize < MinSystemDiskSize` 时静默改成 50GB 的代码。新配置仍由 `ValidateSystemDisk` 在保存/提交时拒绝小于 50GB；历史异常配置保持原值并由镜像检查或 CVM 判断。

### 错误文案

新增独立 i18n key，不复用“导入镜像大于 50GB”的 `MsgImageSizeTooLarge`：

- 中文：`所选镜像要求系统盘至少 %dGB，当前选择为 %dGB`
- 英文：`The selected image requires a system disk of at least %dGB; the current selection is %dGB`

---

## API 契约全景

| 方法 | 路径 | 权限 | 审计 | 主要职责 |
|------|------|------|------|----------|
| GET | `/admin/config/resource` | Admin | 否 | 读取站点资源配置和来源 |
| POST | `/admin/config/resource` | Admin | 是 | 校验预期字段、规范化并保存站点资源配置 |
| GET | `/admin/config/resource/options/basic` | Admin | 否 | 返回本地静态枚举，描述按请求语言翻译 |
| GET | `/admin/config/resource/options/instance-types` | Admin | 否 | 按 zone/charge type 查询固定 allowlist 中的 SELL 机型 |
| GET | `/admin/config/resource/options/system-disks` | Admin | 否 | 查询实例族后通过 CBS 获取系统盘类型和容量范围 |
| GET | `/admin/group-config/groups` | Admin | 否 | 查询哪些组配置了 `resource_config` |
| POST | `/admin/group-config/policy` | Admin | 是 | `config_key=resource_config` 时校验 wrapper、预期字段并写入组策略 |
| POST | `/admin/group-config/policy/delete` | Admin | 是 | 删除组资源策略 |
| POST | `/openclaw/create` | Login | 是 | 接受可选 form `resource_config`，走最终容量校验 |
| POST | `/admin/instances/create` | Admin | 是 | 复用同一创建生命周期和容量校验 |

动态 options 查询参数：

| 接口 | 参数 | 规则 |
|------|------|------|
| instance-types | `zone` | 必填 |
| instance-types | `instance_charge_type` | 可选；传值时必须在 charge allowlist |
| instance-types | `refresh` | `1` 绕过缓存，其余按普通请求 |
| system-disks | `zone` | 必填 |
| system-disks | `instance_charge_type` | 可选；传值时必须在 charge allowlist |
| system-disks | `instance_type` | 必填且必须在固定机型 allowlist |
| system-disks | `refresh` | `1` 绕过缓存 |

缓存契约：

- TTL：5 分钟。
- key 必须包含 tenant identifier、region、endpoint type、zone、charge type、instance type，禁止跨租户复用云账号结果。
- fresh response 的 `source=tencent_cloud`；缓存命中的响应必须是 `source=cache`。
- `refresh=1` 绕过已有 cache，但同 key 并发仍只允许一个 winner 调云 API。
- winner 失败时不写坏缓存；waiter 无可用结果时返回受控错误。

---

## 腾讯云 SDK 与日志契约

1. CVM/CBS client 只通过 `GetCVMClient(ctx)` / `GetCBSClient(ctx)` 获取。
2. `validateCreateResourceConfig` 不再调用旧 `NewCVMClient`。
3. 所有本功能新增的以下调用使用 `CallSDKAPITyped`：
   - `DescribeZoneInstanceConfigInfos`（options instance-types）
   - `DescribeZoneInstanceConfigInfos`（options system-disks 前置）
   - `DescribeDiskConfigQuota`
   - `DescribeZoneInstanceConfigInfos`（create preflight）
4. 在 `controller/access_log.go` 增加 `SDKComponentCBS = "cbs"`，CVM 复用 `SDKComponentCVM`。
5. 失败响应继续使用现有 i18n 错误，不把凭证或完整敏感请求写入业务响应。

---

## 改动文件清单

> “首版已新增/修改”表示完整功能基线，仍需在 Implement/Review 阶段验证；“本轮修改/新增”是当前 Plan 明确要求落地的差距。

| 文件 | 操作 | 说明 |
|------|------|------|
| `model/site_config.go` | 首版已修改，保留 | `ResourceConfig` 字段、默认提取、Camel→snake、机型/磁盘 allowlist 与基础校验 |
| `model/ai_image.go` | 复用 | `AIImage.ImageSize` 是镜像容量来源，不新增字段 |
| `model/group_config_binding.go` | 复用 | 使用现有 policy binding CRUD 和唯一约束，不新增表 |
| `controller/resource_config.go` | 首版已新增；本轮修改 | 完整 Parse/Normalize/Validate/Apply；增加兼容 JSON object 校验、镜像磁盘纯校验；preflight 改统一 client/SDK wrapper |
| `controller/resource_options.go` | 首版已新增；本轮修改 | 三个 options handler；补参数校验、tenant cache key、cache source、SDK wrapper、i18n 文案和可测试依赖 |
| `controller/admin_config.go` | 首版已修改；本轮修改 | GET/POST 站点资源配置；校验 request wrapper 和预期字段；处理 `UpdateSiteConfig` 错误 |
| `controller/admin_group_config.go` | 首版已修改；本轮修改 | `resource_config` wrapper/value 必填与预期字段校验、Normalize/Validate、受控错误 |
| `controller/usergroup/types.go` | 首版已修改，保留 | `PolicyValueJSON`、`PolicyKeyResourceConfig`、PolicyDef、展示顺序与 Category |
| `controller/usergroup/resolve.go` | 首版已修改；本轮修改 | 最近祖先解析；对 DB/JSON 错误做 RichError 包装 |
| `controller/usergroup/overview.go` | 首版已修改，保留 | JSON policy 解析和站点 fallback 来源 |
| `controller/admin_user_group_tree.go` | 首版已修改，保留 | JSON 类型 policy meta 输出 |
| `controller/openclaw.go` | 首版已修改；本轮修改 | 保留覆盖优先级；删除静默扩容；在 VPC/SG/CVM 前执行镜像容量校验 |
| `controller/tencent_clients.go` | 首版已修改，保留 | `GetCBSClient` 统一工厂 |
| `controller/access_log.go` | 本轮修改 | 增加 `SDKComponentCBS`，供 CBS 调用统一记录 |
| `controller/audit.go` | 首版已修改，保留 | `/admin/config/resource` 和组策略写接口审计规则 |
| `main.go` | 首版已修改，保留 | 注册站点配置、options、组策略路由；写路由保持 `WithAudit` |
| `i18n/keys.go` | 本轮修改 | 增加镜像/系统盘冲突和 options 描述等用户可见 key |
| `i18n/en.go` | 本轮修改 | 为新增 key 注册英文翻译 |
| `go.mod`、`go.sum` | 首版已修改，保留 | CBS SDK 依赖；不新增其他依赖 |
| `sql/init.sql` | 首版已修改，保留 | 新部署 `site_configs.resource_config` schema |
| `sql/0706-resource-config.sql` | 删除 | 旧前缀不符合最终 `Release/2026_07_28` 基线 |
| `sql/0728-resource-management.sql` | 合并后保留 | 资源策略表与实例调整字段统一增量迁移 |
| `controller/resource_config_test.go` | 首版已新增；本轮扩充 | Parse/Normalize/Validate/Apply、JSON object/未知字段兼容、镜像容量纯函数全分支 |
| `controller/resource_options_test.go` | 本轮新增 | basic i18n、参数、缓存、tenant key、source、inflight、CVM/CBS 成功失败 |
| `controller/admin_config_test.go` | 本轮扩充 | 站点 GET fallback、POST 成功/失败/未知字段兼容/DB 写失败 |
| `controller/admin_group_config_coverage_test.go` | 本轮扩充 | resource_config set/delete、wrapper/字段校验、Normalize |
| `controller/usergroup/resolve_coverage_test.go` | 首版已修改；本轮扩充 | self/parent/nearest/no hit/error/坏 JSON |
| `controller/usergroup/overview_coverage_test.go` | 首版已修改；本轮扩充 | site/group JSON value 与 source |
| `controller/usergroup/config_category_test.go` | 首版已修改，保留 | policy 数量、顺序、Category 回归 |
| `controller/openclaw_sg_selection_test.go` | 首版已修改；本轮扩充 | 容量失败发生在 SG 前，ImageSize=0/等于/小于继续 |
| `controller/openclaw_handler_writectx_test.go` | 首版已修改，保留 | 云预检 hook 隔离与 RunInstances 写上下文回归 |
| `controller/admin_instance_create_test.go` | 首版已修改；本轮扩充 | 管理员创建同样覆盖新容量行为；ImageSize=0 full success |
| `docs/API.md` | 本轮修改 | 补全部资源 API、请求/响应/错误/source/cache、create 参数和镜像容量规则；修正 image_size 单位为 GB |
| `docs/basic/platform-policy.md` | 不修改 | 与资源配置策略功能无关，保持基线 TODO 占位文件 |
| `docs/INDEX.md` | 不修改 | 保持基线索引，不在本功能中清理无关条目 |
| `test/scripts/admin_resource_config/test_resource_config.py` | 本轮新增 | 站点配置、options、权限、校验、恢复原配置、API coverage |
| `test/scripts/admin_user_groups/test_resource_config_policy.py` | 本轮新增 | 组策略设置、查询、继承、删除和清理 |
| `test/scripts/openclaw_instance/test_exclusive_instance_create_resource_config.py` | 本轮新增 | 创建覆盖和镜像容量分支，确保实例/策略/配置清理 |
| `test/scripts/README.md` | 本轮修改 | 登记资源配置测试目录、环境前提和运行方式 |
| `.specs/plans/2026-07-17_resource-config-policy/00-overview.md` | 各阶段更新 | 单一进度真相源 |
| `.specs/plans/2026-07-17_resource-config-policy/01-clarify.md` | 已完成 | 完整需求和确认决策 |
| `.specs/plans/2026-07-17_resource-config-policy/02-plan.md` | 本阶段新增 | 本完整计划 |
| `.specs/plans/2026-07-17_resource-config-policy/03-implement.md` 至 `08-commit.md` | 后续阶段新增 | 按模板记录真实产物，不追填虚假时间或结果 |

---

## 调用链 / 数据流

### 站点配置写入

```text
POST /admin/config/resource
→ WithAudit + WithOpenAPI
→ requireAdmin
→ 解析单个 {resource_config: object}，未知字段兼容忽略
→ Parse / Normalize / Validate
→ model.UpdateSiteConfig({ResourceConfig: normalized JSON})
→ DB error 显式处理
→ {ok:true}
```

### 站点配置读取

```text
GET /admin/config/resource
→ requireAdmin
→ site_configs.resource_config
→ 空时 ExtractResourceConfigFromTemplate(CVMTemplate)
→ 仍空时 ExtractResourceConfigFromTemplate(DefaultCVMTemplate)
→ {ok:true, resource_config:{...}, source:site_config|cvm_template|default_template}
```

### 用户组策略

```text
POST /admin/group-config/policy
→ requireAdmin + audit
→ config_key == resource_config
→ 解析 value_json={"value":object}，要求 value，未知字段兼容忽略
→ Parse / Normalize / Validate
→ usergroup.SetPolicy
→ group_config_bindings

实例创建
→ ResolveResourceConfig(groupID)
→ GetAncestorIDs(self→root)
→ 最近 resource_config policy binding
→ SourceLocal / SourceInherited
→ 无命中返回 SourceSiteDefault，由调用方使用站点配置
```

### 资源选项

```text
GET options/*
→ requireAdmin
→ 校验 query
→ tenant+region+scope cache key
→ refresh!=1 且 cache hit：source=cache
→ miss：同 key inflight winner
→ GetCVMClient/GetCBSClient
→ CallSDKAPITyped
→ 过滤 SELL / SYSTEM_DISK / allowlist / 容量范围
→ 缓存 5 分钟
→ source=tencent_cloud
```

### 创建实例

```text
POST /openclaw/create 或 POST /admin/instances/create
→ 用户/管理员、配额、agent_type、启用镜像校验
→ CVMTemplate 反序列化
→ 最近组 ResourceConfig；无组命中则站点 ResourceConfig/fallback
→ 用户 resource_config 兼容解析、校验预期字段并覆盖
→ 兼容 disk_type 冲突校验并应用
→ 强制 ImageId
→ validateImageSystemDiskSize
   ├─ ImageSize>0 且 >DiskSize：400，清理 placeholder，终止
   ├─ ImageSize==0：跳过
   └─ DiskSize nil：跳过
→ VPC/子网解析
→ validateCreateResourceConfig（机型可售性，fail-closed）
→ SG 选择/扩容
→ CallSDK/RunInstances
→ 持久化真实实例并清理 placeholder
```

---

## 数据库变更

| 表 | 字段 | 类型 | 说明 |
|----|------|------|------|
| `site_configs` | `resource_config` | `TEXT NULL` | 站点级 snake_case JSON，空值按模板 fallback |
| `group_config_bindings` | 复用 `value_json` | `TEXT` | `config_type=policy`、`config_key=resource_config`，无需新字段 |
| `ai_images` | 复用 `image_size` | `BIGINT` | 镜像容量 GB；无需 schema 变更 |

迁移计划：

1. 保持 `sql/init.sql` 中 `site_configs.resource_config`。
2. 将资源配置与实例调整增量统一合并到 `sql/0728-resource-management.sql`。
3. SQL 内容保持单一来源，不创建第二份重复 migration。
4. 使用 `BASE_BRANCH=Release/2026_07_28 bash .ci/ci-check-schema.sh` 验证日期前缀和 `init.sql + migration` 等价。
5. 测试集群已通过应用镜像使用 SQLite/既有 schema；迁移文件改名不代表再次在共享生产库手工执行。若某环境已手工执行旧文件，部署前由 DBA 确认列存在，禁止重复 DDL。

---

## 实现顺序

1. **契约与纯函数**
   - 在 `resource_config.go` 建立兼容 JSON object 解析 helper。
   - 增加无副作用镜像/磁盘容量校验和专用 i18n error。
   - 先写本 Plan 中对应测试代码，确认边界和错误参数。
2. **创建路径**
   - 删除静默扩容块。
   - 在所有覆盖完成、任何 VPC/SG 云副作用前调用容量校验。
   - 包装 ResolveResourceConfig 普通错误，保证 JSON 500 而非 panic。
3. **站点/组策略入口**
   - 三个资源配置入口统一 object/必填/预期字段校验，未知字段兼容忽略。
   - 处理站点配置落库错误。
4. **资源选项与 SDK 合规**
   - GetXxxClient + CallSDKAPITyped。
   - tenant cache key、cache source、query allowlist、i18n descriptions。
   - 保留 5 分钟 TTL 和 inflight 语义。
5. **数据库合规**
   - migration 0706 → 0715，保持 init.sql 一致。
6. **行为 smoke**
   - 只运行新/受影响的定向测试，确认请求优先级、容量分支、SG 顺序、options 缓存。
7. **进入后续 SOP**
   - Implement 完成并 smoke 后记录 `03-implement.md`，经用户确认再进入 UT；Docs、IT、Review、Commit 均独立记录真实结果。

---

## 测试用例设计（自然语言描述）

> 先于实现编写。Implement 阶段按本表写测试代码；UT 阶段运行并记录结果、覆盖率和未覆盖行。P0 必须 100% 通过。

### 单元测试（UT）— Parse / Normalize / Validate / Apply

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| U01 | 完整配置解析 | 包含计费、机型、磁盘、公网的 object | 所有字段准确解析 | P0 |
| U02 | 空配置解析 | 空串/空白/`{}` | 返回非 nil 空配置，无错误 | P0 |
| U03 | 非法 JSON | 截断 object | 返回解析错误 | P1 |
| U04 | 非 object | 数组/字符串/null | 返回 400 对应错误 | P1 |
| U05 | 未知字段兼容 | `{"instance_type":"Ai2.MEDIUM2","instance_typo":"x"}` | 解析成功，预期字段生效，未知字段忽略 | P0 |
| U06 | 尾随 JSON | `{...}{...}` | 返回解析错误 | P1 |
| U07 | Normalize | charge/disk 大小写混合且有空白 | charge/disk 大写，字符串 trim | P0 |
| U08 | Normalize nil | nil config/nil nested structs | 不 panic | P1 |
| U09 | 合法预付费 | PREPAID + period + renew flag | 校验通过 | P0 |
| U10 | 预付费缺子配置 | PREPAID，无 prepaid | 返回参数错误 | P0 |
| U11 | 非法周期/续费 | period<=0 或未知 renew flag | 返回参数错误 | P1 |
| U12 | 非法计费/机型 | allowlist 外值 | 返回错误并包含非法值 | P0 |
| U13 | 磁盘类型和容量 | 合法类型且 50/100GB | 通过 | P0 |
| U14 | 非法磁盘 | allowlist 外类型或小于 50GB | 返回错误 | P0 |
| U15 | 公网字段省略 | InternetAccessible 存在，public_ip_assigned 省略 | 不把已有 true 改成 false | P0 |
| U16 | 公网显式 false | public_ip_assigned=false | request 显式 false，后续清理计费字段 | P0 |
| U17 | 公网非法组合 | true 但计费/带宽非法 | 返回对应 i18n 错误 | P1 |
| U18 | Apply 全字段 | 完整 ResourceConfig + 空 CVM request | request 对应字段全部赋值 | P0 |
| U19 | Apply 部分字段 | 只传 instance_type/system disk type | 只覆盖显式字段，其他保持原值 | P0 |
| U20 | Apply nil | cfg=nil | request 完全不变 | P1 |

### 单元测试（UT）— 镜像容量与创建顺序

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| U21 | 镜像大于磁盘 | ImageSize=100，DiskSize=50 | RichError 包含 100/50；磁盘仍为 50 | P0 |
| U22 | 镜像等于磁盘 | 100/100 | 通过，不修改指针 | P0 |
| U23 | 镜像小于磁盘 | 50/100 | 通过，不修改指针 | P0 |
| U24 | 镜像大小未知 | ImageSize=0，DiskSize=50 | 跳过比较 | P0 |
| U25 | 未选磁盘容量 | ImageSize=100，SystemDisk nil 或 DiskSize nil | 跳过比较，交给 CVM | P1 |
| U26 | Handler 失败顺序 | DB 中启用镜像 100GB，最终磁盘 50GB；SG hook 计数 | HTTP 400；VPC/SG/CVM hook 均未调用；placeholder 清理 | P0 |
| U27 | Handler 等于继续 | 100/100 | 进入后续预检和 SG hook | P0 |
| U28 | Handler 小于继续 | 50/100 | 进入后续预检和 SG hook | P0 |
| U29 | Handler ImageSize=0 | 0/50 | 不被容量校验拦截，进入 CVM mock | P0 |
| U30 | 不再静默扩大 | 历史模板 DiskSize=10，ImageSize=0 | request 保持 10，后续由 CVM mock 决定 | P0 |
| U31 | 站点覆盖 | 模板/站点同字段不同 | 无组策略时站点值进入最终 request | P0 |
| U32 | 最近组覆盖 | 站点、父组、子组同字段不同 | 最近命中组整份策略生效 | P0 |
| U33 | 组缺失字段 | 组只传机型，站点磁盘与模板磁盘不同 | 机型取组，磁盘取模板而非站点 | P1 |
| U34 | 用户覆盖 | 组和用户同字段不同 | 用户值生效 | P0 |
| U35 | disk_type 冲突 | user resource disk type 与旧参数不同 | 400；未调用 SG/CVM | P0 |
| U36 | Resolver 普通错误 | group closure 查询失败/坏 value_json | 返回 JSON 500，不 panic | P0 |

### 单元测试（UT）— 站点与用户组 API

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| U37 | GET 站点显式配置 | DB ResourceConfig 有效 | 200，source=site_config | P0 |
| U38 | GET 模板 fallback | ResourceConfig 空，CVMTemplate 有效 | 200，source=cvm_template | P0 |
| U39 | GET 默认 fallback | 两者为空 | 200，source=default_template | P1 |
| U40 | POST 有效配置 | 完整 request object | 200，Normalize 后落库，不改 CVMTemplate | P0 |
| U41 | POST 未知字段兼容 | wrapper 或 config 含未知字段且预期字段有效 | 200，预期字段规范化落库，未知字段不持久化 | P0 |
| U42 | POST DB 失败 | 注入更新错误 | 500 JSON，不返回 ok=true | P0 |
| U43 | POST 权限/方法 | 无 admin token / 非 POST | 401/403 或 405 | P1 |
| U44 | 组策略有效 | `value_json={"value":{...}}` | Normalize 后写入 binding | P0 |
| U45 | 组 wrapper 校验与兼容 | 缺 `value` / 含未知字段 | 缺 `value` 返回 400；未知字段忽略且预期字段写入 | P0 |
| U46 | 组策略删除 | 已存在 resource_config binding | 删除成功，之后 resolver fallback | P0 |
| U47 | Resolver self/parent/nearest | self、父、多个祖先组合 | SourceLocal/Inherited 和最近值准确 | P0 |
| U48 | Overview | 站点/组 JSON policy | value 是 object，source 准确 | P1 |

### 单元测试（UT）— 资源选项、缓存和 SDK

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| U49 | basic 中文 | 默认语言 | 完整枚举，description 中文 | P0 |
| U50 | basic 英文 | `Accept-Language: en` | description 英文，无硬编码中文 | P0 |
| U51 | instance-types 参数 | 缺 zone/非法 charge type | 400，不调用 CVM | P0 |
| U52 | instance-types 云结果 | SELL/非 SELL/非 allowlist 混合 | 只返回固定 allowlist 中 SELL 项 | P0 |
| U53 | system-disks 参数 | 缺 zone/type 或 type 非 allowlist | 400，不调用 SDK | P0 |
| U54 | system-disks 云结果 | available/unavailable、用途和类型混合 | 只返回 available SYSTEM_DISK 且 allowlist 类型，min>=50，非法范围跳过 | P0 |
| U55 | fresh cache source | cache miss | 调云一次，source=tencent_cloud | P0 |
| U56 | cache hit source | 同 tenant 同 key 二次请求 | 不调云，source=cache | P0 |
| U57 | refresh | cache 已有且 refresh=1 | 重新调云并更新 cache | P1 |
| U58 | tenant 隔离 | tenant A/B 参数相同 | 两个 cache key，各自调用对应凭证 | P0 |
| U59 | TTL 过期 | 写入过期 entry | miss 并删除旧 entry | P1 |
| U60 | inflight 去重 | 同 key 并发 miss | 一个 winner 调云，waiter 读相同缓存；无死锁/泄漏 | P0 |
| U61 | winner 失败 | 并发 winner SDK error | 不写缓存，waiter 返回受控错误 | P1 |
| U62 | SDK wrapper | CVM/CBS 成功和失败 | 通过 CallSDKAPITyped 返回同结果/错误 | P1 |
| U63 | create preflight cache | 缓存含/不含目标机型 | 命中通过；不含返回不可售；不调云 | P0 |
| U64 | create preflight miss | 云返回 SELL/非 SELL/错误 | 分别通过/400/400 fail-closed | P0 |

### 集成测试（IT）

> 使用隔离测试租户和可恢复配置。所有写测试在 `finally` 中恢复原站点配置、删除组策略、删除测试用户/组/实例。Token、Secret、kubeconfig 不进入日志或产物。

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| I01 | 资源 API 鉴权 | 无 token、坏 token、普通 token 调 5 个 admin operation | 401/403；管理员正常 | P0 |
| I02 | 站点配置读写闭环 | GET 保存原值；POST 有效配置；GET；finally 恢复 | 200，值和 source 正确，CVMTemplate 未改 | P0 |
| I03 | 站点配置校验与兼容 | 未知字段、非法机型/磁盘/预付费组合 | 未知字段忽略且预期字段生效；非法预期字段 400，原配置不变 | P0 |
| I04 | basic options | 中/英文请求 | 枚举、周期、renew、描述和语言正确 | P0 |
| I05 | instance options | zone、charge type、refresh=1 | 200；只含 allowlist SELL；覆盖全部 query 参数 | P0 |
| I06 | system disk options | zone、charge type、instance type、refresh=1 | 200；结构和范围合法；覆盖全部 query 参数 | P0 |
| I07 | 组策略闭环 | 创建测试组；set resource_config；groups/overview 查询；delete | 最近策略和 source 正确，清理后 fallback | P0 |
| I08 | 用户创建覆盖 | 测试用户/组/站点配置层层不同；传用户 config | 实际创建走用户最终值；实例成功后销毁 | P0 |
| I09 | 镜像大于磁盘 | 隔离测试 DB/tenant seed ImageSize=100，最终 DiskSize=50 | 400 含 100/50；日志无 SG 选择和 RunInstances；无残留实例 | P0 |
| I10 | 镜像等于/小于磁盘 | 受控镜像 fixture 和有效磁盘 | 进入 CVM，成功实例最终销毁 | P1 |
| I11 | ImageSize=0 | 隔离 fixture 为 0，磁盘保持所选值 | 容量校验不拦截，CVM 收到原值；成功则销毁，云拒绝则保留云错误证据 | P0 |
| I12 | API 增量覆盖 | 生成 base/current OpenAPI 后运行上述脚本 | 新 operations 全覆盖；zone/charge/type/refresh/resource_config 无未覆盖参数 | P0 |

### 测试执行与覆盖率计划

Implement smoke（只跑受影响测试）：

```bash
go test ./controller -run 'ResourceConfig|ResourceOptions|Image.*Disk|ResourcePreflight|AdminCreate' -count=1
go test ./controller/usergroup -run 'ResourceConfig|PolicyOverview|ConfigCategory' -count=1
```

UT 阶段：

```bash
go test ./controller ./controller/usergroup -race -count=1
go test ./... -race
BASE_BRANCH=origin/Release/2026_07_15 bash .ci/ci-check-coverage.sh
```

质量检查：

```bash
gofmt -w <本任务修改的 Go 文件>
goimports -w <本任务修改的 Go 文件>
go vet ./...
staticcheck ./...
go build ./...
BASE_BRANCH=Release/2026_07_15 bash .ci/ci-check-schema.sh
```

IT 阶段：

```bash
make openapi BASE_BRANCH=origin/Release/2026_07_15
make test IMAGE=<本分支镜像> BASE_BRANCH=origin/Release/2026_07_15 TEST_ARGS="--report-dir ./test-report"
```

验收阈值：

- 新增/修改函数增量覆盖率不低于 80%。
- P0 用例 100% 通过。
- `new_ops_uncovered` 为空。
- `new_params` 未覆盖集合为空。
- 不以源码文本断言代替行为测试。

---

## 文档计划

### `docs/API.md`

完整补充：

1. 顶部接口总览中的 5 个 resource config operations。
2. `GET/POST /admin/config/resource` 的权限、Content-Type、完整 schema、Normalize/Validate、fallback source、成功/失败示例。
3. 三个 options API 的 query 参数、缓存/refresh/source、完整响应类型和错误。
4. `POST /admin/group-config/policy` 中 `resource_config` 的 `value_json` wrapper 示例、继承语义和删除方式。
5. `/openclaw/create` 与 `/admin/instances/create` 的 `resource_config` 参数、优先级、disk_type 冲突和镜像容量错误。
6. `AIImage.image_size` 单位从错误的“字节”修正为“GB”，与腾讯云和 `SystemDisk.DiskSize` 一致。
7. 说明 `ImageSize==0` 跳过本地比较并交由 CVM。

### `docs/basic/platform-policy.md` 与 `docs/INDEX.md`

两者不属于资源配置策略功能范围，保持目标基线内容不变。资源策略接口契约集中维护在 `docs/API.md`。
- `test/scripts/README.md` 登记新目录、前置变量、隔离 fixture 和清理规则。
- `docs/openapi.json` 只通过生成流程更新，不手工编辑。

---

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| R1 | 当前静默扩大磁盘，违反新契约并可能产生额外费用 | 高 | 高 | 删除赋值；纯函数校验不修改请求；UT 断言容量保持 |
| R2 | `ImageSize` 和 `DiskSize` 单位理解不一致 | 中 | 高 | 明确两者均按 GB；修正 API 文档；IT 使用已知 fixture |
| R3 | `ImageSize==0` 无法本地阻止不兼容请求 | 中 | 中 | 按已确认行为继续 CVM；保留云错误；不得猜测/扩容 |
| R4 | 历史模板存在小于 50GB 配置，移除静默扩大后由 CVM 拒绝 | 低 | 中 | 新写入继续静态拒绝；历史请求保持原值并返回真实错误；文档说明 |
| R5 | 容量检查放置过晚导致 VPC/SG 副作用 | 中 | 高 | 放在 VPC/SG/RunInstances 前；hook 计数回归测试 |
| R6 | 未知 JSON 字段可能是拼写错误但按兼容要求被忽略 | 中 | 中 | 文档提供完整 schema；UT/IT 断言预期字段生效且未知字段不持久化 |
| R7 | Resolver 普通错误传入 `EnsureRichErrorOrPanic` 触发 panic | 低 | 高 | 在 resolver/caller 包装 RichError；handler 回归测试 |
| R8 | 站点 DB 更新失败仍返回成功 | 低 | 高 | 检查 `UpdateSiteConfig` error；注入写失败 UT |
| R9 | options cache 跨租户复用云结果 | 中 | 高 | cache key 加 identifier；双租户 UT |
| R10 | cache hit 仍显示 `source=tencent_cloud` | 高 | 低 | cache entry 预生成 `source=cache` payload；响应测试 |
| R11 | inflight winner 失败导致 waiter 无结果或阻塞 | 低 | 中 | defer close/delete；失败不缓存；并发超时测试 |
| R12 | 直接 SDK 调用缺 request_id/耗时/参数日志 | 高 | 中 | GetXxxClient + CallSDKAPITyped + CBS component |
| R13 | 管理员 options 返回硬编码中文 | 高 | 中 | i18n key + Accept-Language=en 测试 |
| R14 | 迁移仍用 0706 前缀导致 Release schema CI 失败 | 高 | 高 | 移动到 0715；运行 schema CI |
| R15 | IT 修改站点/组策略或创建付费资源后清理失败 | 中 | 高 | 隔离 tenant、finally 恢复、唯一前缀、记录并执行 cleanup |
| R16 | 为 IT 直接修改共享服务镜像元数据 | 低 | 高 | 只在隔离测试部署的 seed DB 中准备 fixture；禁止改共享生产/验收数据 |
| R17 | 兼容忽略未知字段可能掩盖客户端拼写错误 | 中 | 低 | 保持首版兼容；文档列出完整字段；所有预期字段继续 Normalize/Validate |

---

## 完成定义

Plan 后续各阶段全部满足才可 Commit：

- [ ] 全部文件按本计划实现或明确记录无须改动的证据。
- [ ] 镜像大于磁盘时 400、容量不变、VPC/SG/CVM 未调用。
- [ ] ImageSize=0 和 DiskSize nil 分支继续 CVM，不本地扩容。
- [ ] 独立策略配置入口共享 object/必填/预期字段校验，未知字段兼容忽略。
- [ ] options cache 按 tenant 隔离，fresh/cache source 正确，并发无泄漏。
- [ ] 所有本功能腾讯云调用使用 GetXxxClient + CallSDKAPITyped。
- [ ] 四个管理写/读接口权限和审计完整，DB 错误不伪报成功。
- [ ] `sql/init.sql` 与 `sql/0728-resource-management.sql` 通过 schema CI。
- [ ] UT P0 100% 通过，增量覆盖率不低于 80%。
- [ ] 新 API 和参数通过真实 IT/API coverage。
- [ ] `docs/API.md` 和测试文档完整；`docs/basic/platform-policy.md`、`docs/INDEX.md` 保持基线不变。
- [ ] Review 无未解决高/中问题，无 CODEBUDDY 红线。
- [ ] `08-commit.md` 在提交前完成；随后才 git add/commit/push。

---

## 2026-07-21 资源策略实体化重设计（覆盖前述存储与管理契约）

最新 TAPD 原型和用户确认推翻了本文前述“SiteConfig + GroupConfigBinding 分组内嵌完整资源配置”设计。本节是后续实现的最终契约；与前文冲突时以本节为准。

### 最终数据模型

1. 新增 `resource_policies`：`id`、`identifier`、`name`、`is_default`、`config_json`、时间戳；租户内名称唯一。
2. 策略应用范围复用现有 `group_config_bindings` 资源绑定语义：`config_type=resource_policy`、`config_key=<ResourcePolicy.ID>`、`group_id=<UserGroup.ID>`、`value_json={}`。
3. `ResourcePolicy.ConfigJSON` 是策略内容的唯一真相源；`GroupConfigBinding.ValueJSON` 对纯关系绑定仅为空占位，不保存或复制策略配置。
4. 一条策略可直接绑定多个分组，一个分组最多直接绑定一条资源策略；更新事务锁定目标组并拒绝被其他策略占用的分组。
5. 应用范围通过 `(identifier, config_type, config_key, group_id)` 索引快速反查，只展示直接绑定分组，继承后代不持久化。
6. 生效解析为：本组直接策略 → 最近祖先直接策略 → 企业默认资源策略；最终只命中一条，不做策略合并。

### 默认策略

- 固定名称“企业默认资源策略”，不可删除、不可改名、无应用范围，配置可编辑。
- 首次查询策略列表、配置总览需要默认兜底或创建 Agent 需要默认兜底时并发安全地懒创建。
- 懒创建从当前 `SiteConfig.CVMTemplate` 提取资源配置，使用租户内固定名唯一索引解决并发，冲突后重查。
- 不启动扫描全部租户，不增加 seeded 标志，不在重启时覆盖已编辑的默认策略。
- `SiteConfig.CVMTemplate` 只作为首次物化来源；删除未发布的 `SiteConfig.ResourceConfig` 双真相源。

### 管理 API

| 方法 | 路径 | 契约 |
|------|------|------|
| GET | `/admin/resource-policies?page=&page_size=` | 懒创建默认策略；默认项置顶；返回配置、直接分组和分页 |
| POST | `/admin/resource-policies/create` | 原子创建普通策略和非空 `group_ids` |
| POST | `/admin/resource-policies/update` | 普通策略原子更新名称/配置/范围；默认策略只更新配置 |
| POST | `/admin/resource-policies/delete` | 删除普通策略并清理其 `GroupConfigBinding` 应用范围；拒绝默认策略 |

应用范围更新在事务中锁定目标 `UserGroup` 行，检查该组是否已被其他 `resource_policy` 绑定；存在占用时整体回滚并返回 409，禁止并发覆盖其他策略。

复用 `GET /admin/user-groups/tree?with_resource_policy=true` 返回 `direct_resource_policy{id,name}`。该字段通过 `config_type=resource_policy` 批量查询，只表示直接绑定，继承策略不应使组在选择器中置灰。

### Resolver 与调用链

新增 `ResolveEffectiveResourcePolicy(ctx, groupID)`，按闭包链批量查询 `config_type=resource_policy` 绑定并返回策略及 `local` / `inherited` / `default` 来源。创建 Agent 和组织配置总览改用该 resolver；资源总览不再通过通用 `PolicyDefs` 中的内嵌资源配置。

### Options

- 删除静态 `GET /admin/config/resource/options/basic`，固定枚举、默认值和约束写入 `docs/API.md`。
- 保留并切换为：
  - `GET /admin/resource-policies/options/instance-types`
  - `GET /admin/resource-policies/options/system-disks`

### 干净切换

该资源策略能力尚未发布，不保留兼容层：

- 删除 `SiteConfig.ResourceConfig` 和 `GET/POST /admin/config/resource`；
- 删除 `PolicyKeyResourceConfig`、`config_type=policy/config_key=resource_config` 的内嵌资源配置存取和 `/admin/group-config/policy` 特殊分支；保留通用 `GroupConfigBinding` 表并新增可索引的 `config_type=resource_policy` 资源关系。
- 删除旧 resolver、名称 wrapper、basic options、旧审计/文档/测试；
- 保留与本功能无关的 `docs/basic/platform-policy.md` 及 `docs/INDEX.md` 基线内容；最终资源策略 API 契约维护在 `docs/API.md`。
- 保留 `ResourceConfig` Parse/Normalize/Validate/Apply、`ExtractResourceConfigFromTemplate` 和两个动态 options；
- 本次架构重构不额外改变用户请求级 `resource_config` / `disk_type` overlay 语义。

### 重设计测试契约

| # | 场景 | 预期 | 级别 |
|---|------|------|------|
| R-U01 | 首次及 100 并发默认策略查询 | 仅创建一条固定名默认策略，所有调用返回同一 ID | P0 |
| R-U02 | 默认策略已编辑后再次查询 | 不覆盖配置；改名、绑定分组、删除均拒绝 | P0 |
| R-U03 | 创建普通策略绑定多个分组 | 一条策略及多条 `config_type=resource_policy` 绑定，配置只保存一次 | P0 |
| R-U04 | 创建/编辑包含已占用分组 | 返回 409，策略和全部绑定事务回滚 | P0 |
| R-U05 | 两个策略并发争抢同一分组 | 通过目标组行锁串行化；仅一个成功，另一请求不覆盖 | P0 |
| R-U06 | 本组/最近祖先/无绑定 | 分别返回 local/inherited/default 及正确策略 | P0 |
| R-U07 | 删除普通策略 | 清理该策略的资源绑定，后续解析回退祖先或默认 | P0 |
| R-U08 | 列表分页和应用范围 | 默认置顶；按 `config_type+config_key` 批量查询直接分组，无 N+1 | P1 |
| R-U09 | 用户组树 | 按 `config_type+group_id` 批量返回直接绑定策略，不把继承组标记占用 | P1 |
| R-U10 | 配置总览和创建 Agent | 使用同一 resolver，返回/应用真实 policy ID、名称、配置和来源 | P0 |
| R-U11 | 多租户 | 不可查询或绑定其他租户策略/分组 | P0 |
| R-U12 | 路由清理 | 四个新管理 API 和两个动态 options 可用；旧 resource/basic 路径消失 | P1 |

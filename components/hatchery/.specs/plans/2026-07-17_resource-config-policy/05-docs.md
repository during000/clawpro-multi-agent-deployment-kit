# 05. Docs — 文档更新

---

## 结论

Resource Config Policy 的 API、模块设计、功能索引和 IT 契约文档已完整补齐；`docs/API.md` 已成功生成当前/base OpenAPI spec，并精确产生 5 个新增 operation。文档明确保持未知 JSON 字段兼容、最近祖先整份命中、租户隔离缓存、镜像/系统盘 GB 规则和不静默扩容。

本阶段同时确认一个实现边界：普通用户 `/openclaw/create` 支持本次 `resource_config` form 覆盖；`/admin/instances/create` 当前不接受直接 `resource_config` JSON 字段，只复用站点/组策略、镜像容量和机型预检。文档按实际代码和 Clarify 契约记录，没有虚构管理员单次覆盖参数。

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 修改 | 顶部总览新增 5 个 resource operation；完整补充站点 GET/POST、三个 options、ResourceConfig schema、缓存/source/refresh、组 wrapper/继承/删除、普通创建覆盖、管理员创建边界、镜像大小 GB 语义 |
| `docs/basic/platform-policy.md` | 不修改 | 与本功能无关，保持基线 TODO 占位文件 |
| `docs/INDEX.md` | 不修改 | 保持基线索引 |
| `test/scripts/README.md` | 修改 | 增加 I01–I12 IT 目标、目标脚本路径、隔离环境、受控 ImageSize fixture、清理规则、敏感信息约束和 API coverage 命令；明确脚本将在 IT 阶段落地，当前不声称可运行 |
| `docs/openapi.json` | 生成更新 | 通过 `make openapi` 从当前 `docs/API.md` 生成；未手工编辑 |
| `docs/openapi_base.json` | 生成更新 | 通过 `make openapi BASE_BRANCH=origin/Release/2026_07_15` 从 base `docs/API.md` 生成；未手工编辑 |

---

## `docs/API.md` 完成内容

### 接口总览

新增并生成以下 5 个 operation：

| 方法 | 路径 | OpenAPI |
|------|------|---------|
| GET | `/admin/config/resource` | 已生成 |
| POST | `/admin/config/resource` | 已生成 |
| GET | `/admin/config/resource/options/basic` | 已生成 |
| GET | `/admin/config/resource/options/instance-types` | 已生成 |
| GET | `/admin/config/resource/options/system-disks` | 已生成 |

### 站点资源配置

`GET /admin/config/resource` 已说明：

- 管理员权限；
- `resource_config` object；
- `source=site_config/cvm_template/default_template`；
- fallback 顺序；
- 历史坏 JSON 的空 object 兼容行为；
- 完整成功响应示例。

`POST /admin/config/resource` 已说明：

- `application/json`、管理员权限、审计 action；
- 必填 wrapper `resource_config`；
- 完整 ResourceConfig 和三个 nested object schema；
- 固定机型/磁盘/续费 allowlist；
- trim/uppercase Normalize；
- `disk_size` 最小 50GB；
- 未知字段兼容忽略且不持久化；
- 非 object、`null`、数组、多 JSON 值和尾随内容拒绝；
- 成功、缺字段、JSON、业务校验和 DB 失败响应。

### Options

三个接口已说明：

- `basic` 的静态枚举、i18n description 和 `source=static`；
- instance-types 的 `zone`、`instance_charge_type`、`refresh`；
- system-disks 的 `zone`、`instance_charge_type`、`instance_type`、`refresh`；
- fresh `source=tencent_cloud` 与 cache/waiter `source=cache`；
- UTC RFC3339 `refreshed_at`；
- 机型 SELL + allowlist 过滤；
- CBS 系统盘 available/usage/type/range 过滤和最小 50GB；
- 5 分钟 TTL、tenant/region/endpoint/scope cache key；
- `refresh=1`、inflight winner/waiter、失败不缓存；
- 400/500 错误边界。

### 用户组策略

`POST /admin/group-config/policy` 已增加严格可解析的请求参数表，并补充：

- `config_key=resource_config`；
- `value_json={"value": <ResourceConfig object>}`；
- wrapper/value object 和 value 必填；
- 未知字段兼容与规范化持久化；
- self/parent/root 最近祖先整份命中；
- `local/inherited/site_default` 来源；
- 命中组策略后缺失字段保留 CVMTemplate，不与站点或远祖先 merge；
- 完整转义请求示例。

`POST /admin/group-config/policy/delete` 已补 Content-Type、请求参数表和 `resource_config` 删除/fallback 示例。

### 创建入口

`POST /openclaw/create` 已增加：

- form `resource_config` 参数；
- 完整 JSON object/未知字段/预期字段校验；
- 模板 → 站点或最近组 → 用户 → `disk_type` → runtime 强制字段优先级；
- `disk_type` 与用户资源磁盘类型冲突时 400；
- ImageSize/DiskSize 都是 GB；
- 镜像大于最终磁盘时早于 VPC/SG/CVM 失败；
- 不静默扩容；
- ImageSize=0、未设置磁盘/容量时跳过本地比较；
- zone 确定后的机型预检在 SG/RunInstances 前 fail closed。

`POST /admin/instances/create` 已明确：

- 共用站点/最近组策略、镜像容量和机型预检；
- 当前严格 JSON schema不接受直接 `resource_config`；
- 管理员单次资源变化只能使用已支持的 `disk_type`，或预先更新站点/组策略；
- 未知顶层 `resource_config` 会返回 400。

### 镜像单位

`AIImageWithPublic.image_size` 从错误的“字节”修正为：

- 镜像要求的系统盘容量，单位 GB；
- 0 表示未知。

---

## 无关基础文档

`docs/basic/platform-policy.md` 与 `docs/INDEX.md` 不属于本功能范围，保持 `Release/2026_07_28` 基线内容不变。资源策略接口、继承、缓存、创建校验和排障契约统一记录在 `docs/API.md`。
---

## IT 文档契约

`test/scripts/README.md` 已登记后续 IT 阶段的目标脚本：

- `admin_resource_config/test_resource_config.py`
- `admin_user_groups/test_resource_config_policy.py`
- `openclaw_instance/test_exclusive_instance_create_resource_config.py`

由于当前仍处于 Docs 阶段，README 明确标注这些文件将在 IT 阶段创建，未把不存在脚本写成当前可执行命令。

IT 说明覆盖：

- I01–I12 逐项场景；
- 隔离测试部署/租户；
- 真实腾讯云凭证由部署注入，禁止打印 Secret；
- ImageSize 大于/等于/小于磁盘和 0 的隔离 DB fixture；
- 站点原值保存/恢复；
- 组策略、组、用户、实例的 `finally` 清理；
- cleanup failure 不得吞掉；
- OpenAPI 生成和 `new_ops_uncovered` / 新参数覆盖门禁。

---

## 验证

### OpenAPI 生成

执行：

```bash
make openapi BASE_BRANCH=origin/Release/2026_07_15
```

结果：PASS。

| 指标 | Current | Base | 增量 |
|------|--------:|-----:|-----:|
| endpoint definitions | 398 | 393 | +5 |
| unique paths | 379 | 375 | +4（GET/POST 共用 `/admin/config/resource`） |
| operations | 390 | 385 | +5 |
| endpoints with parameters | 310 | 305 | +5 |

生成文件：

- `docs/openapi.json`：598427 bytes；
- `docs/openapi_base.json`：589214 bytes。

### OpenAPI 契约断言

使用 Python 解析生成的 current/base JSON 并断言：

- 新 operation 集合精确等于 5 个资源 operation；
- instance-types query 精确为 `zone`、`instance_charge_type`、`refresh`；
- system-disks query 精确为 `zone`、`instance_charge_type`、`instance_type`、`refresh`；
- 站点 POST request body 只有必填 `resource_config`；
- `/openclaw/create` form 包含 `resource_config`；
- `/admin/instances/create` request body 不包含 `resource_config`；
- group policy body 必填 `group_id/config_key/value_json`；
- group policy delete body 必填 `group_id/config_key`。

结果：`OpenAPI contract assertions PASS`。

### 文档完整性检查

| 检查 | 结果 |
|------|------|
| `docs/basic/platform-policy.md` 与基线一致 | PASS |
| `docs/API.md` 不再存在 `image_size` 字节单位描述 | PASS |
| 5 个资源 operation 标题全部存在 | PASS |
| `docs/INDEX.md` 与基线一致 | PASS |
| README 包含 I01–I12、finally、`new_ops_uncovered` | PASS |
| 修改文档中的本地 Markdown 链接目标均存在 | PASS |

---

## 检查项

- [x] `docs/API.md` 已更新
- [x] `docs/basic/platform-policy.md` 保持基线
- [x] `docs/INDEX.md` 保持基线
- [x] `test/scripts/README.md` 已登记 IT 契约且未虚构当前不存在脚本可运行
- [x] 请求参数表使用严格 4 列格式，参数名未使用反引号
- [x] POST operation 声明 Content-Type
- [x] `docs/openapi.json` / `docs/openapi_base.json` 只通过生成流程更新
- [x] current/base OpenAPI 生成成功
- [x] 新增 operation 和参数断言通过

---

## 2026-07-22 资源策略实体化重设计 Docs

> 本节覆盖本文前述基于站点配置和分组内嵌资源配置的旧文档结论；冲突时以本节为准。

### 结论

- `docs/API.md` 已记录四个独立策略管理 API、两个动态 options、用户组树 `with_resource_policy` 参数，以及创建入口的独立策略解析顺序。
- 企业默认策略文档与 UT 修复一致：只允许更新 `resource_config`；改名或提交非空 `group_ids` 返回 409，拒绝请求不覆盖已保存配置。
- 旧 `/admin/config/resource*`、静态 basic options、`site_configs.resource_config` 和分组 `config_key=resource_config` 文档均不存在。
- `docs/basic/platform-policy.md` 与 `docs/INDEX.md` 不属于本功能范围，保持基线；最终资源策略接口契约集中维护在 `docs/API.md`。
- `test/scripts/README.md` 和默认策略集成测试已同步 409 保护契约。

### 本阶段修正

| 文件 | 修正 |
|------|------|
| `docs/API.md` | 将默认策略 `name/group_ids` 从“忽略”修正为改名或非空范围返回 409；“三个动态 options”修正为两个 |
| `test/scripts/README.md` | I02 明确配置单独更新成功，改名/非空 `group_ids` 返回 409 且配置不变 |
| `test/scripts/admin_resource_config/test_resource_config.py` | 先单独更新默认配置，再分别验证改名和绑定返回 409，且名称、范围和配置保持不变 |
| `docs/openapi.json` | 由 `make openapi` 从最终 `docs/API.md` 重新生成 |
| `docs/openapi_base.json` | 由同一命令从 `origin/Release/2026_07_15` 重新生成 |

### 最终 API 契约

| 方法 | 路径 |
|------|------|
| GET | `/admin/resource-policies` |
| POST | `/admin/resource-policies/create` |
| POST | `/admin/resource-policies/update` |
| POST | `/admin/resource-policies/delete` |
| GET | `/admin/resource-policies/options/instance-types` |
| GET | `/admin/resource-policies/options/system-disks` |

请求 schema：

- create 必填 `name/resource_config/group_ids`；
- update 必填 `id/resource_config`，可选 `name/group_ids`，但普通策略和默认策略适用不同约束；
- delete 必填 `id`；
- 用户组树 query 包含 `with_resource_policy`，只返回 `direct_resource_policy`，不把继承策略标记为直接绑定。

### OpenAPI 生成与断言

执行：

```bash
make openapi BASE_BRANCH=origin/Release/2026_07_15
```

结果：PASS。

| 指标 | Current | Base |
|------|--------:|-----:|
| endpoint definitions | 401 | 393 |
| unique paths | 383 | 375 |
| operations | 393 | 385 |
| endpoints with parameters | 314 | 305 |
| 文件大小 | 608784 bytes | 589214 bytes |

当前分支相对 base 共增加 8 个 operation，其中 6 个为本任务资源策略 operation，另外 2 个属于同分支其他功能。解析生成 JSON 后确认：

- 六个 `/admin/resource-policies*` 路径全部存在；
- `/admin/config/resource*` 路径集合为空；
- create/update/delete 的 required 和 properties 与上文一致；
- update 的 `name/group_ids` OpenAPI description 明确默认策略保护返回 409；
- `/admin/user-groups/tree` query 含 `with_resource_policy`。

### 其他验证

| 检查 | 结果 |
|------|------|
| `docs` 与 `test/scripts` 不含旧 `/admin/config/resource*`、`site_configs.resource_config`、`config_key=resource_config` | PASS |
| `docs/basic/platform-policy.md` 与基线一致 | PASS |
| `docs/INDEX.md` 与基线一致 | PASS |
| 三个资源策略集成脚本 `python3 -m py_compile` | PASS |

### 阶段结论

- 最终实现、API 文档、OpenAPI、IT 契约和测试脚本对默认策略保护、独立策略范围及新路径保持一致。
- 下一阶段进入 IT，运行三个现有集成脚本并验证真实部署、真实腾讯云选项和创建链路。

### 2026-07-22 默认策略名称 i18n 补充

- 默认策略持久化名称保持稳定；列表、配置总览和树元数据在读取时按请求语言本地化。
- 中文展示“企业默认资源策略”，英文展示 `Enterprise Default Resource Policy`。
- 普通策略名称不参与翻译。
- 同一语言下读取到的默认展示名称可原样回传 update；真正改名仍返回 409。
- `docs/API.md`、OpenAPI、`test/scripts/README.md` 和管理集成脚本已同步。

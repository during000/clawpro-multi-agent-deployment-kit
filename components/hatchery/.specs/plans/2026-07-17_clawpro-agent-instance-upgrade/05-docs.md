# 05. Docs — 文档更新

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 修改 | 接口目录新增两个资源调整路由；补全 validate/submit 严格请求、逐实例响应、准入、按状态与 resize mode 派生的停机语义、幂等、异步状态机、错误码和审计语义；扩展 admin list/status 的资源字段、调整字段和筛选说明 |
| `docs/openapi.json` | 重新生成 | 从 `docs/API.md` 生成两个新 POST operation，以及 `GET /admin/instances` 的两个资源筛选参数 |

`docs/INDEX.md` 已明确 `.specs/docs` 是 `docs/` 的软链接，因此 `docs/API.md` 的更新已同步覆盖 `.specs/docs/API.md`，无需重复维护。当前索引中没有资源调整专属模块文档；本步骤没有创建第二份接口真相源，也无需修改索引。

## 对外契约增量

### 新接口

| 方法与路径 | 文档重点 |
|------------|---------|
| `POST /admin/instances/adjust-config/validate` | 管理员权限、1 MiB 严格 JSON、`ids`/`instance_ids` 规范化后二选一且 1～100 项、两类调整的条件字段、实时校验、规格询价、固定 HTTP 200 的逐实例可调结果、不记录审计 |
| `POST /admin/instances/adjust-config` | 与 validate 相同的严格 envelope 和实时复验、停机强确认、部分受理、同目标 `already_processing`、持久化 worker、15 分钟超时、提交接口审计 |

文档固定了以下边界：

- 规格只允许 `Ai2.MEDIUM2` → `Ai2.MEDIUM4` → `Ai2.LARGE8` → `Ai2.2XLARGE16` 序列内严格升配。
- 系统盘只允许满足实时 CBS 配额区间和步长的扩容；`resize_mode=online|offline`。
- CVM 规格询价接口没有 `ForceStop` 参数；询价与写请求共享实例/目标规格，`ForceStop` 只在实际执行中使用。系统盘不调用价格询价接口，按实时磁盘事实、配额和模式规则校验，并以规范化 operation 构造实际写请求。
- `accepted`/`rejected` 在实时复验后返回；`already_processing` 在云查询前按持久化任务返回，当前资源值来自缓存且可能省略实时磁盘配额。
- 列出全部 34 个稳定 `reason_code`；原始腾讯云错误不进入 API 响应。

### 现有接口

- `GET /admin/instances`
  - 新增 SQL 级多值精确筛选 `cvm_instance_type`、`system_disk_size`，影响列表、总数、分页和统计。
  - CVM 项新增规格、CPU、内存、系统盘及调整展示字段；普通无调整项通过 `omitempty` 省略调整字段，本地 Agent 省略全部云资源/调整字段。
  - 调整中复用 worker 既有云轮询同步最近观察到的稳定 `running`/`stopped`，过渡态保留最近稳定值，`actions=[]`，并返回资源调整 operation/status。
- `GET /admin/instances/status`
  - 保留 CVM 原始 `state`，新增实时规格、CPU、内存、系统盘和调整展示字段。
  - 云实例已释放时 `state=RELEASED` 且资源字段回退缓存；本地 Agent 继续拒绝。

## OpenAPI schema

两个新 POST operation 的请求 schema 均包含：

- `additionalProperties: false`；
- `ids` 与 `instance_ids` 的标准 `oneOf` 互斥约束；
- `adjustment_type=instance_type|system_disk` 枚举及对应目标字段条件必填；
- `system_disk` 分支的 `resize_mode=online|offline` 枚举；
- `x-normalization` 精确记录 ID 数组的去零值/trim/去空/去重/保序，以及规范化结果 1～100 项语义。没有使用不符合 handler 行为的原始数组 `minItems`/`maxItems`。

## 验证记录

| 检查 | 结果 | 证据 |
|------|------|------|
| Python 语法 | PASS | `python3 -m py_compile test/api_md_to_openapi.py` |
| OpenAPI 生成 | PASS | `python3 test/api_md_to_openapi.py`：395 个 endpoint definition、377 个唯一 path、387 个 operation、307 个含参 endpoint；生成 601944 字节 `docs/openapi.json` |
| 新路由/schema | PASS | 结构化断言确认 validate/submit/list/status 四个 path 存在；两个 POST 均为管理员 BearerAuth 且包含 6 个请求字段 |
| 严格 schema | PASS | 结构化断言确认 unknown field 拒绝、target selector `oneOf`、调整类型/扩容模式分支、两端约束树一致及精确 `x-normalization` |
| 参数表与实现字段 | PASS | 自动比对 `adminInstanceAdjustmentRequest` 的 6 个 JSON tag 与两个接口参数表完全一致 |
| 稳定错误码 | PASS | 自动比对 `instance_adjustment_cloud.go` 的 33 个稳定码与 API 文档表完全一致 |
| list/status 字段 | PASS | 自动比对 10 个资源/调整展示字段均已文档化，status map 均实现 |
| Schema overlay 行为 | PASS | 临时 Markdown fixture 验证 overlay 与参数表 schema 深度合并，类型、说明、required 保留，enum/`additionalProperties` 生效 |
| 独立 API contract review | PASS | reviewer 首轮发现 4 项文档/生成 schema 偏差；全部修正后复核结论为 PASS |

## 检查项

- [x] `docs/API.md` 已更新
- [x] `.specs/docs/` 相关文档已同步（与 `docs/` 为同一软链接目标）
- [x] 参数表为 4 列，参数名未使用反引号包裹
- [x] `docs/openapi.json` 已由文档源重新生成
- [x] 新接口、list/status 增量字段、稳定错误码与实现完成自动比对
- [x] 文档与生成 OpenAPI 已通过独立复核

## 2026-07-22 Docs 修订

`docs/API.md` 已明确：规格升配保留价格询价；系统盘校验不调用磁盘询价，改用实时实例/盘事实、CBS 配额、DeniedAction 和 resize mode。实际在线写失败以稳定错误返回。本文此前“磁盘询价与写请求同源”的描述由本节覆盖。

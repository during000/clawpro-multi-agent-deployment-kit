# 02. Plan — 方案设计

---

## 设计目标

将“诊断会话是否空闲超时”的业务时间与数据库记录的通用更新时间解耦：

- 新会话从真正进入 `active` 状态的时间开始计算 12 小时；
- STS 凭证刷新只更新凭证过期时间，不再改变 `updated_at`；
- 存量会话没有激活时间时回退 `created_at`，使已经卡住的 NoFiles 会话能够自动收敛；
- 已产生对话文件的会话继续以远端文件 mtime 判断空闲时间，不改变现有行为。

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `model/doctor_session.go` | 修改 | 为 `DoctorSession` 增加可空的 `ActivatedAt *time.Time` 字段 |
| `controller/doctor.go` | 修改 | 激活时原子写入 `status=active` 与 `activated_at`；刷新 STS 时使用不触碰 `updated_at` 的字段更新，并处理数据库错误 |
| `task/doctor_cleanup.go` | 修改 | NoFiles 分支改用 `activated_at` 判断 12 小时超时，存量 NULL 数据回退 `created_at` |
| `sql/init.sql` | 修改 | 在全量初始化表结构中增加 `doctor_sessions.activated_at` |
| `sql/0730-add-doctor-session-activated-at.sql` | 新增 | 为 `Release/2026_07_30` 提供 MySQL 增量迁移 |
| `controller/doctor_coverage_test.go` | 修改 | 覆盖激活时间写入，以及 STS 刷新不改变 `updated_at` |
| `task/doctor_cleanup_test.go` | 修改 | 覆盖新会话与存量会话的 NoFiles 超时基准，并回归有文件分支 |

以下文件无需修改：

- `model/db.go` 已将 `DoctorSession` 纳入统一 `AutoMigrate`。
- `model/migrate.go` 使用通用 `DoctorSession` 模型迁移，新增字段会随模型自动迁移，不需要自定义字段映射。
- API 请求与响应结构不变，不新增接口文档。

## 调用链 / 数据流

```
ActivateDoctorSession
  └─ 同一条数据库更新：status = active, activated_at = now

doctor-cleanup（每 5 分钟）
  ├─ RefreshDoctorSTS
  │    ├─ 调用 STS/TAT 刷新凭证
  │    └─ UpdateColumn(sts_expired_at)；不更新 updated_at
  ├─ 查询 active 会话
  └─ GetDoctorSessionMtime
       ├─ NoFiles
       │    └─ 基准时间 = activated_at；NULL 时使用 created_at
       │         ├─ 空闲时间 > 12h → status = ending
       │         └─ 空闲时间 ≤ 12h → 保持 active
       ├─ 返回文件 mtime
       │    ├─ 距今 > 12h → status = ending
       │    └─ 距今 ≤ 12h → 保持 active
       └─ 其他错误 → 沿用现有探测失败处理

doctor-ending（每 1 分钟）
  └─ CleanupDoctorSession
       ├─ 销毁云主机
       ├─ 删除 Instance 记录
       └─ status = ended
```

关键实现约束：

1. 激活状态和 `activated_at` 必须在同一次 `Updates` 中写入，避免出现已 active 但没有激活时间的新数据。
2. `RefreshDoctorSTS` 仅对周期刷新路径使用 `UpdateColumn`。创建阶段首次写入 STS 发生在激活之前，保留现状也不会参与新的超时基准。
3. NoFiles 分支只读取稳定的生命周期字段，不再读取 `updated_at`；日志中的起算时间同步改为实际使用的基准时间。
4. 不引入“所有 active 会话都按 created_at 强制结束”的兜底，避免改变已有对话文件会话以 mtime 续期的设计。

## 数据库变更

| 表 | 字段 | 类型 | 说明 |
|----|------|------|------|
| `doctor_sessions` | `activated_at` | `datetime(3) NULL` | 会话首次进入 `active` 状态的时间；新记录由激活流程写入，存量记录保持 NULL |

迁移与兼容策略：

- 新增字段允许 NULL，不做历史回填；历史数据没有可靠的真实激活时间。
- 应用读取旧数据时，`activated_at IS NULL` 回退 `created_at`。这会让已经超过 12 小时的卡死 NoFiles 会话在下一轮清理中进入 `ending`。
- 迁移应先于新版本应用发布，避免旧表结构无法承载 GORM 模型字段。
- 当前清理任务已按 `status` 查询后逐条判断时间，本次不新增索引。

## 测试用例设计（自然语言描述）

> 先于实现编写，Implement 阶段据此编码。
> UT 用例走 `go test`，IT 用例走 Python 集成测试（`test/scripts/`）。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | STS 周期刷新不污染通用更新时间 | active 会话的 `updated_at` 预置为固定旧时间，STS 即将过期，刷新依赖返回新凭证 | `sts_expired_at` 更新；`updated_at` 与刷新前完全相同 | P0 |
| 2 | 会话成功激活时记录激活时间 | creating 会话，激活依赖均成功 | `status=active`，`activated_at` 非空且接近当前时间，两者由同一更新落库 | P0 |
| 3 | NoFiles 新会话超过 12 小时 | `activated_at` 为 13 小时前，`updated_at` 可为近期，远端返回 NoFiles | 会话被标记为 `ending` | P0 |
| 4 | NoFiles 新会话未超过 12 小时 | `activated_at` 为近期，远端返回 NoFiles | 会话保持 `active` | P0 |
| 5 | NoFiles 存量会话回退创建时间 | `activated_at=NULL`、`created_at` 为 13 小时前、`updated_at` 被刷新为近期，远端返回 NoFiles | 忽略被污染的 `updated_at`，按 `created_at` 标记为 `ending` | P0 |
| 6 | 有对话文件时维持 mtime 语义 | `activated_at`/`created_at` 已超过 12 小时，但远端文件 mtime 为近期 | 会话保持 `active` | P1 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 数据库迁移兼容性 | 在现有 `doctor_sessions` 表执行增量 SQL，再启动新模型 | 字段创建成功，历史记录为 NULL，应用可正常读写 | P1 |
| 2 | 定时任务完整收敛链路（条件允许时手工验证） | 创建医生但不发起对话，将激活时间调整为超过 12 小时，触发 cleanup 与 ending 任务 | 会话先进入 `ending`，随后云主机与 Instance 被清理并进入 `ended` | P1 |

本改动没有 API 契约变化，且真实链路需要云资源、定时任务及 12 小时时间条件，不新增 Python API 集成测试；核心回归由上述 Go 单元测试覆盖。IT 阶段若环境具备条件，再执行迁移检查与定时任务链路验证，否则记录为环境受限。

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 存量数据只有 `created_at`，其时间可能早于真正激活时间 | 中 | 存量 NoFiles 会话可能比精确激活时间略早结束 | 仅对缺少 `activated_at` 的历史记录兜底；新记录使用精确激活时间。无对话且已长期占用资源的会话优先收敛符合修复目标 |
| 2 | 应用先发布、数据库迁移后执行 | 低 | 新字段查询或写入失败 | 发布顺序固定为先执行 migration，再部署应用 |
| 3 | `UpdateColumn` 跳过 `updated_at` 和 GORM 更新钩子 | 低 | 后续维护者误认为更新时间会变化 | 这是 STS 技术字段刷新所需语义，并用回归测试锁定 |
| 4 | 修改 NoFiles 判断误伤已有对话会话 | 低 | 正常诊断被提前结束 | 仅修改 NoFiles 分支，并增加近期文件 mtime 的回归用例 |
| 5 | 误将 2026-07-20 旧方案的全局绝对超时带入本次 | 低 | 改变有文件会话的生命周期语义 | 本次明确不引入全局 `created_at` 强制超时，只解决 STS 污染与 NoFiles 计时 |

## 验收标准

- 周期刷新 STS 后 `doctor_sessions.updated_at` 不发生变化。
- 新激活会话持久化非空 `activated_at`。
- 没有对话文件的会话从激活起超过 12 小时后，最迟在下一轮 5 分钟 cleanup 中进入 `ending`。
- 存量 `activated_at=NULL` 的 NoFiles 会话按 `created_at` 收敛，不受近期 `updated_at` 影响。
- 有对话文件的会话仍以文件 mtime 计算空闲时间。
- 相关 Go 单元测试通过，MySQL 全量 schema 与增量 migration 保持一致。

# 角色切换与版本化更新 — 前端联调文档

> **用途**：本次迭代为 hatchery 后端的角色切换与版本化更新功能（OpenSpec change `role-switch-and-distribute`）提供前端联调所需的接口契约、状态判定逻辑和文案约定。
>
> 完整 API 文档见 `hatchery/docs/API.md` 的「实例管理」与「4.8 角色管理」段落，本文档仅抽取与本次迭代相关的内容，方便前端独立查阅。

---

## 一、接口清单

| 状态 | 方法 | 路径 | 用途 |
|------|------|------|------|
| 🆕 新增 | POST | `/openclaw/switch-role` | 用户端切换单实例角色（含 `role_id=0` 即移除） |
| 🆕 新增 | POST | `/admin/roles/distribute?id=<role_id>` | 管理端批量推送角色最新版到选中实例 |
| 🆕 新增 | GET | `/admin/roles/instances?role_id=<role_id>&...` | 管理端「更新弹窗」数据源 |
| 🆕 新增 | GET | `/admin/roles/records?instance_ids=1,2,3&role_ids=7,8&page=&page_size=` | 分页查询角色下发记录（`instance_ids`/`role_ids` 均可选，支持逗号分隔多值，不传查全部） |
| ⚙️ 字段变更 | POST | `/admin/roles/create` | body 新增 `version` 字段，缺省 `1.0` |
| ⚙️ 字段变更 | POST | `/admin/roles/update` | body 新增 `version` 字段，传则必须严格大于旧版本号 |
| ⚙️ 字段变更 | GET | `/admin/roles` | 响应每个角色对象新增 `version` 字段 |
| ⚙️ 字段变更 | GET | `/admin/roles/detail` | 响应 `role.version` 字段 |
| 🔁 保留 | POST | `/openclaw/remove-role` | 旧前端兼容；新前端建议改用 `switch-role` 传 `role_id=0` |

---

## 二、版本号显示规范

后端存储 / API 传输的版本号都是裸数字 **`X.Y`**（如 `1.0`、`2.5`），不带 `v` 前缀。

前端展示加 `v` 前缀：

| 场景 | 后端字段值 | 前端展示 |
|------|-----------|----------|
| 角色版本列 | `2.0` | `v2.0` |
| 状态标签 | `1.0` | `待更新 v1.0` |
| 输入框 placeholder | `2.0`（旧版本号） | `新版本号需高于上个版本号 2.0` |

格式校验（前端可同步实现）：版本号必须为 `^\d+\.\d+$`，否则红框提示「版本号格式必须为 X.Y」。

---

## 三、`POST /openclaw/switch-role` 详解

### 请求

- **鉴权**：登录用户（`requireLogin`）
- **Content-Type**：`application/json`
- **路径参数**：无
- **Body**：

```json
{
  "instance_id": 100,
  "role_id": 5
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | uint | 是 | 实例数据库 ID |
| role_id | uint | 是 | 目标角色 ID；`0` 表示移除角色（等价于 `/openclaw/remove-role`） |

### 响应（始终 200）

```json
{
  "accepted": 1,
  "skipped": []
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| accepted | int | 实际通过校验并触发切换的实例数（单实例场景下 0 或 1） |
| skipped | array | 被跳过实例列表，每项 `{instance_id, reason}`，详见第五节 |

> ⚠️ **关键约定**：被跳过仍返回 `200 OK + accepted=0 + skipped`，**不返回 4xx**。前端按 `accepted` 判断成功/失败，按 `skipped[0].reason` 拿到具体跳过原因后 toast 提示。

### 后端行为

切换成功后异步执行（前端无需等待）：
1. `instance.role_id` → 目标 ID
2. `instance.distributed_role_version` → 目标角色当前 `version`（若 `role_id=0` 则清空）
3. `instance.soul_set_at` → NULL（触发周期任务重下发 `SOUL.md`）
4. 异步装新角色技能（按 slug 比对覆盖装；用户已装且新角色不含的 slug 不动）

---

## 四、`POST /admin/roles/distribute` 详解

### 请求

- **鉴权**：管理员（`requireAdmin`）
- **Content-Type**：`application/json`
- **路径参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 角色 ID（Query 参数） |

- **Body**：

```json
{
  "instance_ids": [12, 25, 33, 47]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | uint[] | 是 | 实例 ID 列表，长度 `[1, 500]` |

### 响应（始终 200，除非 400/500）

```json
{
  "accepted": 2,
  "skipped": [
    {"instance_id": 33, "reason": "already_updated"},
    {"instance_id": 47, "reason": "not_running"}
  ]
}
```

### 失败响应

| 状态码 | 场景 |
|--------|------|
| 400 | `role_id` 缺失或为 0 |
| 400 | `instance_ids` 为空 |
| 400 | `instance_ids` 长度超过 500 |
| 500 | 后端内部错误 |

### 与 switch-role 的差异

| 维度 | switch-role | distribute |
|------|-------------|------------|
| 鉴权 | 登录用户 + owner 校验 | 管理员 |
| 输入 | 单实例 + 任意 role_id | 单角色 + 多实例（≤500） |
| `role_mismatch` 校验 | ❌ 不做 | ✅ 实例 role_id 必须等于 path id |
| `already_updated` 校验 | ❌ 不做 | ✅ 实例 distributed_version 必须 < 角色 version |
| 允许 `role_id=0` | ✅ 等价于移除 | ❌ 必须 > 0 |

---

## 五、`skipped[].reason` 枚举翻译表

| reason token | 中文文案（前端展示） | 触发场景 |
|--------------|----------------------|----------|
| `not_found` | 实例不存在或无权访问 | 实例 ID 不存在 / 不属于当前用户（switch）/ 跨租户 |
| `agent_type_unsupported` | 实例类型不支持角色配置 | 实例的 agent_type 不在支持角色的白名单中 |
| `not_running` | 实例非运行中状态 | 实例 CVM 状态不是 RUNNING |
| `role_mismatch` | 实例当前角色与目标角色不匹配 | （仅 distribute）实例 role_id ≠ path id |
| `already_updated` | 实例已是最新角色版本 | （仅 distribute）实例 distributed_role_version >= role.version 且 role_sync_status='updated' |
| `no_role_to_remove` | 实例当前未关联角色，无需移除 | （仅 switch role_id=0）实例 role_id 已经是 0 |
| `updating_in_progress` | 实例上一次更新还在进行中，请稍后重试 | 实例 role_sync_status='updating'（下发进行中） |
| `role_not_visible` | 该角色在当前实例分组下不可见 | （仅 switch）角色 visibility_type=group 但实例分组不在角色可见分组列表中 |

> 后端提供机器可读 token，前端按上表翻译。如果未来后端新增 reason，前端默认用「操作未完成」兜底文案。

---

## 六、`GET /admin/roles/instances` 详解

### 请求

- **鉴权**：管理员
- **Query 参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| role_id | uint | 是 | 角色 ID |
| role_sync_status | string | 否 | 同步状态过滤：`pending` / `updating` / `updated` / `failed` / `all`（默认 `all`）；**支持逗号分隔多值**（如 `pending,failed`） |
| search | string | 否 | 模糊匹配 instance_name 或 cvm_instance_id |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，上限 100 |

### 响应

```json
{
  "role": {
    "id": 5,
    "name": "行业分析师",
    "version": "2.0"
  },
  "page": 1,
  "page_size": 20,
  "total": 87,
  "items": [
    {
      "instance_id": 123,
      "cvm_instance_id": "ins-xxx",
      "instance_name": "agent-1",
      "user_id": 42,
      "username": "alice",
      "user_groups": [
        {"group_id": 7, "group_name": "技术部"}
      ],
      "group_id": 0,
      "group_name": "",
      "role_version": "1.0",
      "role_sync_status": "pending"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `role.version` | string | 角色当前版本号 |
| `items[].role_version` | string | 实例上记录的最近一次成功推送的版本号；空字符串表示从未推送过 |
| `items[].role_sync_status` | string | 状态字段：`pending` / `updating` / `updated` / `failed`，**前端按下表渲染标签** |
| `items[].user_groups` | array | 该实例所有者所属的所有用户分组 |
| `items[].group_id / group_name` | uint / string | 实例创建时绑定的分组（创建时指定的分组，可能为 0） |

### `role_sync_status` 五态前端展示约定

| `role_sync_status` | 后端判定逻辑 | 前端标签文案 | 备注 |
|--------------------|--------------|--------------|------|
| `pending` | `role_version < role.version` | 「待更新 vX.Y」 | X.Y 为 `role_version` 的值；管理员升版本后 updated/空串 也会翻 pending |
| `updating` | 下发进行中（SOUL 或技能子任务未完成） | 「更新中」 | 可加 loading 图标 |
| `updated` | `role_version >= role.version` 且 SOUL+技能全成功 | 「已更新」 | 推送过最新版本 |
| `failed` | SOUL 或任一技能下发失败 | 「更新失败」 | 可点击查看 record.soul_error 或 record.skill_error（调 `GET /admin/roles/records?instance_ids=N&page_size=1` 取最新记录） |

---

## 七、对接清单（前端自查）

> 本节由后端协议文档抽出，前端按此 checklist 自查。

### 角色管理 Tab

- [ ] 编辑/创建角色弹窗：增加「版本号」输入框，placeholder 用 `新版本号需高于上个版本号 X.Y`（X.Y 取自当前角色的 `role.version`）
- [ ] 版本号格式实时校验：`/^\d+\.\d+$/`，不符合红框提示「版本号格式必须为 X.Y」
- [ ] 角色列表：新增「版本」列，展示 `vX.Y` 格式
- [ ] 操作列：新增「更新」按钮，点击触发更新弹窗
- [ ] 角色名唯一校验：复用现有 `409 同名角色已存在` 文案

### 更新弹窗

- [ ] 列表数据源：`GET /admin/roles/instances?role_id=N`
- [ ] 状态标签按 `role_sync_status` 渲染（pending/updating/updated/failed）
- [ ] `updating` 状态加 loading 图标
- [ ] `failed` 状态可点击查看失败原因（调 `GET /admin/roles/records?instance_ids=N&page_size=1` 获取最新记录的 `soul_error` / `skill_error`）
- [ ] 待更新标签需带 `role_version` 拼出 `待更新 v1.0`
- [ ] 状态下拉筛选：传 `role_sync_status` query 参数，**支持逗号分隔多值**（如 `pending,failed`）
- [ ] 搜索框：传 `search` query 参数
- [ ] 分页：`page` + `page_size`（默认 20，可选 50/100）
- [ ] 全选 / 多选实例 → 调 `POST /admin/roles/distribute?id=N` body 传 `instance_ids`
- [ ] 推送上限 500：前端禁用按钮 / 提示「单次最多推送 500 个实例」
- [ ] 推送返回后：toast 显示 `accepted` 数量；如果有 `skipped` 详细展开

### Agent 卡片切换角色入口

- [ ] 三点菜单 → 「切换角色」（所有 Agent 都显示，包括通用助手）
- [ ] 角色标签气泡点击 → 直接弹出角色选择 Dialog（缩短操作路径）
- [ ] Dialog 默认顺序：与创建 Agent 一致；非通用助手角色额外把「通用助手」放最前
- [ ] Dialog 自动过滤当前角色（不展示已绑定的那个）
- [ ] 选中后展示角色详情卡片（头像 + 技能 + 风格）
- [ ] 底部固定提示：「切换角色不会删除已有的技能配置，并将自动安装新角色的专属技能」
- [ ] 点击「确认切换」→ 调 `POST /openclaw/switch-role` body 传 `instance_id` + `role_id`
- [ ] 多 Agent 实例：仅切换主角色（hatchery 只感知 `Instance.RoleID`），文案微调
- [ ] 响应 `accepted=1` → toast 切换成功；`accepted=0` → 按 `skipped[0].reason` toast 错误文案

---

## 八、`GET /admin/roles/records`（分页查角色下发记录）

### 请求

- **鉴权**：管理员
- **Query 参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | string | 否 | 实例数据库 ID，逗号分隔多值（如 `1,2,3`）；不传则查全部实例 |
| role_ids | string | 否 | 角色 ID，逗号分隔多值（如 `7,8`）；不传则查全部角色 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，上限 100。`page_size=1` 可取最新一条 |

> **仅返回 `source=distribute` 的记录**（管理员分发操作），用户 `switch` 和 `create` 记录不返回。

### 响应

```json
{
  "page": 1,
  "page_size": 20,
  "total": 3,
  "items": [
    {
      "id": 45,
      "instance_id": 123,
      "instance_cid": "ins-xxx",
      "role_id": 7,
      "role_name": "行业分析师",
      "version": "2.0",
      "operator_id": 1,
      "operator_username": "admin",
      "source": "distribute",
      "status": "updated",
      "soul_status": "success",
      "soul_error": "",
      "soul_set_at": "2026-07-03T10:00:00Z",
      "skill_status": "success",
      "skill_error": "",
      "skill_set_at": "2026-07-03T10:02:00Z",
      "created_at": "2026-07-03T10:00:00Z",
      "updated_at": "2026-07-03T10:02:00Z"
    },
    {
      "id": 42,
      "instance_id": 123,
      "status": "failed",
      "version": "1.0",
      "operator_id": 1,
      "operator_username": "admin",
      "soul_status": "success",
      "skill_status": "failed",
      "skill_error": "技能包 data-analysis 尚未完成 SMH 同步，请稍后重试",
      "source": "distribute",
      "created_at": "2026-07-02T15:00:00Z"
    }
  ]
}
```

### 错误响应

| 状态码 | 场景 |
|--------|------|
| 400 | `instance_ids`/`role_ids` 传了但全部值无效（0 或非数字） |
| 401 | 未登录 |

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `items[].operator_id` | uint | 触发操作的用户 ID |
| `items[].operator_username` | string | 操作者用户名（联表 users 表回填，**非数据库列**）；用户被软删除或 `operator_id=0` 时为空字符串 |

### 前端使用场景

1. **查全部下发记录**：不传 `instance_ids`，分页浏览所有实例的分发历史
2. **查某实例历史**：传 `instance_ids=N`，分页浏览该实例的分发记录
3. **查某角色下发历史**：传 `role_ids=N`，分页浏览该角色的分发记录
4. **组合筛选**：传 `instance_ids=1,2&role_ids=7,8`，查询指定实例上指定角色的下发记录
5. **查最新一条（看失败原因）**：传 `instance_ids=N&page_size=1`，取 `items[0]` 的 `soul_error` / `skill_error`

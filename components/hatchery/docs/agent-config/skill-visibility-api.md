# 技能应用范围（Skill Visibility）接口文档

> 本文档描述技能应用范围功能涉及的接口变更，供前端开发参考。

---

## 概念说明

- **应用范围**：控制技能对哪些用户可见。两种类型：
  - `all` — 全部用户可见（默认）
  - `group` — 仅指定分组内的用户可见
- **语义层级**：应用范围跟随 slug（技能标识），上传新版本时自动继承旧版本的设置
- **下发筛选**：辅助性质，管理员可取消筛选查看全部实例

---

## 一、技能上传 — `POST /admin/skills/create`

**新增参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| visibility_type | string | 否 | `all`（默认）或 `group` |
| group_ids | string | 条件必填 | 逗号分隔的分组 ID，如 `1,3,5`。`visibility_type=group` 时必填 |

**行为：**
- 传了 `visibility_type` → 以传入值为准
- 未传 `visibility_type` → 自动继承同 slug 旧版本的应用范围
- 首次上传（无旧版本）且未传 → 默认 `all`

**错误响应（新增）：**
- `400 {"error": "visibility_type 必须为 all 或 group"}`
- `400 {"error": "按分组可见时必须选择至少一个分组"}`
- `400 {"error": "分组不存在: [2 7]"}`

---

## 二、技能更新 — `POST /admin/skills/update`

**新增参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| visibility_type | string | 否 | `all` 或 `group`。不传则不修改应用范围 |
| group_ids | string | 条件必填 | 逗号分隔的分组 ID。`visibility_type=group` 时必填 |

**行为：**
- 不传 `visibility_type` → 应用范围不变（与 `category_ids` 不传不变逻辑一致）
- 传 `visibility_type=all` → 清空分组关联，设为全部用户
- 传 `visibility_type=group` + `group_ids=1,3` → 替换为指定分组

**支持单独编辑应用范围：** 只传 `slug` + `version` + `visibility_type` + `group_ids`，其他字段不传即可。

**错误响应（同上传）**

---

## 三、技能列表 — `GET /admin/skills`

**新增查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| visibility_type | string | 否 | 筛选应用范围类型：`all` 或 `group` |
| group_id | int | 否 | 筛选关联了指定分组的技能 |

**响应变更：** 每个 skill 对象新增两个字段：

```json
{
  "skills": [
    {
      "id": 5,
      "slug": "skill-a",
      "name": "技能A",
      "version": "2.0.0",
      "description": "这是一个基础技能",
      "visibility_type": "group",
      "file_size": 102400,
      "created_at": "2026-03-26T10:00:00Z",
      "categories": [
        {"id": 1, "name": "基础技能"}
      ],
      "visibility_groups": [
        {"group_id": 1, "group_name": "研发组"},
        {"group_id": 3, "group_name": "测试组"}
      ],
      "last_task": { ... }
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| visibility_type | string | `all` 或 `group` |
| visibility_groups | array | 关联的分组列表。`visibility_type=all` 时为空数组 `[]` |
| visibility_groups[].group_id | int | 分组 ID |
| visibility_groups[].group_name | string | 分组名称 |

---

## 四、技能详情 — `GET /admin/skills/detail`

**响应变更：** `skill` 对象新增两个字段（同列表）：

```json
{
  "skill": {
    "id": 5,
    "slug": "my-skill",
    "name": "我的技能",
    "version": "2.0.0",
    "description": "技能描述",
    "categories": [{"id": 1, "name": "基础技能"}],
    "visibility_type": "group",
    "visibility_groups": [
      {"group_id": 1, "group_name": "研发组"}
    ],
    "file_size": 102400,
    "cos_zip_key": "my-skill/my-skill-2.0.0.zip",
    "cos_dir_key": "my-skill/my-skill-2.0.0/",
    "created_at": "2026-03-26T10:00:00Z",
    "updated_at": "2026-03-26T11:00:00Z"
  },
  "versions": ["2.0.0", "1.0.0"]
}
```

---

## 五、下发实例列表 — `GET /admin/skills/instances`

**新增查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | string | 否 | 按用户组筛选实例，只显示该分组内用户的实例。支持逗号分隔多个 ID（取并集），如 `1,3` |
| instance_type | string | 否 | 按实例类型筛选，如 `openclaw`、`hermes`、`lightclawace`。支持逗号分隔多类型，如 `openclaw,hermes` |

**响应新增字段：**

每条实例记录新增 `user_id`、`user_groups` 和 `instance_type` 字段：

```json
{
  "instances": [
    {
      "instance_id": 1,
      "cvm_instance_id": "ins-xxx",
      "instance_name": "user1的实例",
      "instance_type": "openclaw",
      "user_id": 1,
      "username": "user1",
      "status": "installed",
      "version": "2.0.0",
      "user_groups": [
        {"group_id": 1, "group_name": "研发组"},
        {"group_id": 3, "group_name": "测试组"}
      ]
    }
  ],
  "total": 1
}
```

> `user_groups` 为用户所属的全部分组列表，无分组用户返回空数组 `[]`。

**建议前端行为：**
- 进入下发页时，如果技能 `visibility_type=group`，自动将 `visibility_groups` 中的 group_id 拼成 `group_id` 参数传入，默认筛选目标分组的实例
- 提供"查看全部实例"按钮，点击后清除 `group_id` 参数重新查询
- 每行实例可展示用户所属分组标签，辅助判断是否在技能应用范围内

---

## 六、前端交互建议

### 技能上传表单

在"分类"下方新增"应用范围"区域：

```
应用范围：
  [● 全部用户]  [○ 按分组]

  （选择"按分组"后展开）
  分组选择：[下拉多选，支持搜索]
```

### 技能列表页

1. **顶部筛选栏**：新增"应用范围"下拉筛选
   - 选项：`全部`（不传参）、`全部用户`（`visibility_type=all`）、`按分组`（`visibility_type=group`）、各个分组名称（`group_id=N`）
   - 单选，支持搜索

2. **列表字段**：在"分类"列右侧新增"应用范围"列
   - `visibility_type=all` → 显示"全部用户"
   - `visibility_type=group` → 显示分组名称标签（多个以逗号/标签形式展示）
   - 支持点击进入编辑（调用 `POST /admin/skills/update` 只传 visibility 参数）

### 技能详情页

新增"应用范围"字段展示，同列表页样式。

### 下发实例页

顶部筛选栏新增"应用范围"下拉筛选，逻辑同技能列表页。默认值根据当前技能的 `visibility_type` 自动设置。

---

## 七、分组管理相关

### 删除分组

如果分组被某个技能的应用范围引用，**删除会被阻止**（返回 400）。前端在删除分组时如果收到 400，应提示用户先修改引用该分组的技能的应用范围。

### 获取分组列表

复用现有接口 `GET /admin/user-groups`，用于应用范围编辑时的分组选择下拉框。

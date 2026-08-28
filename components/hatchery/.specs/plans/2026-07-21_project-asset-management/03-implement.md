# Implement

实现范围与 OpenSpec change [`project-asset-management`](../../../openspec/changes/project-asset-management/) 一致：项目 CRUD、成员、项目应用范围、资产绑定、Workspace `project_id` 和 TeamAI 按需同步。

- GORM/MySQL/SQLite 维护四张项目关系表。
- `local_agent_scope_bindings` 是 Workspace 关系查询真相源；JSON 继续兼容旧客户端。
- sync 利用既有 distribution task/record 与 ack 链路，并通过 task `batch_id` 精确关联 Workspace 本地事实，避免同 slug 串 Workspace。
- 未增加新的项目更新记录、后台 Worker、自动下发或云端实例绑定。

## 子任务：资产版本与同步模式（实现范围）

> 版本记录与同步模式方案见 [02-plan.md](./02-plan.md)。本小节聚焦改动文件清单 + 测试用例设计（先于实现）。

### 改动文件清单

**model 层（本模块新建/修改）**

| 文件 | 动作 | 说明 |
|---|---|---|
| `model/asset_version.go` | 新建 | `AssetVersionRecord` + `AssetChanges` + 常量（TriggerType / TriggerReason / SegmentType） |
| `model/project.go` | 改 | `Project` 加 `SyncMode` 字段 |
| `model/user_group.go` | 改 | `UserGroup` 加 `SyncMode` 字段 |
| `model/db.go` | 改 | `allModels` 切片加入 `AssetVersionRecord` |
| `model/migrate.go` | 改 | `MigrateFromSQLite` 增加 `asset_versions` 表迁移 |

**SQL**

| 文件 | 动作 | 说明 |
|---|---|---|
| `sql/init.sql` | 改 | 新增 `asset_versions` 表；`projects` / `user_groups` 加 `sync_mode` 列 |
| `sql/<MMDD>-asset-version-sync-mode.sql` | 新建 | 增量迁移（按目标 Release 日期命名） |

**controller 层**

| 文件 | 动作 | 说明 |
|---|---|---|
| `controller/asset_version.go` | 新建 | `RecordAssetSave` / `PublishAssetVersion` / `InstallAssetToTargets` + `HandleAdminAssetVersions` |
| `controller/admin_assets.go` | 改（胶水） | 在 `HandleAdminAssetSave` 事务内插 `RecordAssetSave` 调用（含 sync_mode 落库 + 跳变检测） |
| `controller/admin_skills.go` / `admin_rules.go` | 改（胶水） | 在 `HandleCreate/Delete/UpdateSkill/Rule` 事务内插 `PublishAssetVersion` 胶水（收 `tx`，融入上层事务） |

**路由**

| 文件 | 动作 | 说明 |
|---|---|---|
| `main.go` | 改 | 注册 `GET /admin/assets/versions` → `HandleAdminAssetVersions`（requireAdmin，查询接口不审计） |

### 测试用例设计（先于实现）

**RecordAssetSave（手动保存）**
1. *首建 v1*：目标此前无资产（OldAssets 空），保存 2 个 skill → current_version 0→1，落 1 条 record（version=1, manual_save, Changes.Added 含 2 项, Removed/Updated 空）。
2. *普通修改*：旧 2 skill → 新 3 skill（1 新增 1 删 1 保留）→ record.version=2, Added/Removed 各 1, Updated 0。
3. *同步模式跳变*：OldSyncMode=initial_only → SyncMode=continuous → 触发 InstallAssetToTargets（full=true）。
4. *initial_only 不下发*：SyncMode=initial_only → 下发不被调用。
5. *事务回滚*：注入 tx 错误 → 不落 record、不调用下发。

**PublishAssetVersion（自动）**
6. *asset_version_published + continuous 有 updated*：目标 continuous，from→to → 落 system record，触发下发。
7. *asset_deleted + continuous*：仅 removed → 落 system record，不下发（仅追加不卸载）。
8. *asset_scope_changed（移出）*：目标被移出范围 → 落 record（reason=asset_scope_changed），不下发。
9. *initial_only 目标全部只记录不下发*：4 种 trigger 在 initial_only 目标上均不调用下发。
10. *多目标遍历*：AffectedTargets 含 3 个目标（2 continuous 有 added + 1 initial_only）→ 落 3 条 record，仅 2 个调下发。

**HandleAdminAssetVersions（查询）**
11. *分页*：目标有 5 条记录 → page=1&page_size=2 → 返回 total=5, data 长度 2, 倒序（version 高在前）。
12. *按 target_type+target_id 过滤*：project=12 有 3 条、project=13 有 2 条 → 查 project=12 返回 3 条。
13. *响应结构*：返回 data[].segments[].items 含 asset_type+name；operator 返回 type+id+name（system 时 id=0/name=""）；无 summary/changes 字段。
14. *asset_version_published 段字段*：items 含 old_version+new_version。
15. *sync_mode 段*：仅当模式变更时返回；模式值放在 items[].name（不使用独立 value 字段）。

### 风险评估

- **并发版本号**：用 SQL `version=version+1` 原子自增规避。
- **胶水代码侵入现有 handler**：本模块只插调用，不改其参数处理，降低 merge 冲突。
- **多租户隔离**：查询/写入均走 `model.DB(r.Context())`（红线），不裸 SQL。
- **审计**：versions 为查询接口不审计；save 本身已有审计。

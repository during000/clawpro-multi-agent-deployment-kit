# 02. Plan — 方案设计

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `controller/openclaw_migration.go` | 修改 | 仅为 Hermes 生成兼容开关，并传入排除路径清单 |
| `scripts/export_migration.sh` | 修改 | 增加业务树指纹、阶段状态、持久化日志、tar 诊断分类和归档校验 |
| `controller/openclaw_migration_test.go` | 修改 | 以 fake tar/curl 执行真实导出脚本，覆盖容忍与拒绝分支 |
| `test/scripts/openclaw_instance/test_instance_migration_hermes.py` | 新增 | 创建真实 Hermes 实例，覆盖迁移 export/status/import 契约及脚本安全标记 |
| `docs/API.md` | 修改 | 说明 Agent 面板活跃兼容、日志位置和失败诊断 |
| `.specs/plans/2026-07-31_bug-161540752-hermes-migration-export/*` | 新增 | 按 SOP 记录澄清、方案、实现和验证证据 |

## 调用链 / 数据流

```text
POST /openclaw/migration/export
  → buildMigrationScript
  → 返回内嵌 scripts/export_migration.sh 的源机命令
  → 源机执行
      → 初始化 <agent_home>/logs/migration_export_<timestamp>.log
      → tar 后台打包，stderr 同时写终端和临时诊断文件
      → wait 获取 tar_status
          0 → 验包
          1 + Hermes + 仅根目录 changed
            → 比较打包前后未排除业务树指纹
              一致 → 告警后验包
              不一致 → 明确失败，不进入上传
          其他 → 明确失败，不进入上传
      → tar tf 归档校验
          通过 → 分块上传 + confirm
          失败 → 明确失败，不进入上传
      → EXIT trap 输出成功/失败、阶段、退出码、日志路径
```

## 关键方案

### 1. Hermes 专属、精确容忍根目录变化

允许的唯一诊断行按当前归档根目录动态生成：

```text
tar: <basename(AGENT_DIR)>: file changed as we read it
```

必须同时满足：

- 运行时类型是 Hermes，生成脚本显式设置兼容开关；
- tar 状态码等于 1；
- stderr 至少出现一次上述行；
- stderr 中没有其他非空行；
- 打包前后未排除业务树指纹一致。

普通文件变化如 `tar: .hermes/config.yaml: file changed as we read it` 不满足条件，继续失败。

业务树指纹递归记录未排除路径。目录记录路径/类型/权限；文件和符号链接额外记录
大小、mtime、ctime、inode 与链接目标。目录不记录易被已排除子项影响的
mtime/ctime/size/inode。

### 2. 强制归档校验

无论 tar 状态为 0 还是容忍后的 1，都执行：

```bash
tar tf "$ARCHIVE_PATH" >/dev/null
```

校验失败直接退出，上传逻辑不执行。

### 3. 持久化日志

- 日志路径：`$AGENT_DIR/logs/migration_export_<timestamp>.log`
- 权限：`0600`
- 使用 `exec > >(tee -a "$LOG_FILE") 2>&1` 同步终端与文件。
- `CURRENT_STEP` 在 validating、packing、uploading、confirming、done 间推进。
- EXIT trap 在失败时打印 `阶段 + 退出码 + 日志路径`。
- 不启用 shell xtrace，不打印 SMH token、header 或 URL。

## 数据库/API 变更

无。接口请求、响应字段和状态模型不变。

## 测试用例设计

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 已排除根级运行时文件变化 | fake tar 新增 `gateway.pid`、输出 `.hermes: file changed...` 并返回 1；验包成功 | 业务树指纹一致，脚本继续上传/确认 | P0 |
| 2 | 普通文件在打包时变化 | fake tar 输出 `.hermes/config.yaml: file changed...` 并返回 1 | 脚本非零，不调用上传，日志包含 packing 失败 | P0 |
| 3 | 根目录变化但归档损坏 | create 返回根目录 changed；list/验包失败 | 脚本非零，不调用上传，日志包含 validating 失败 | P0 |
| 4 | 正常 tar | create/list 均成功 | 原有上传与确认路径成功 | P1 |
| 5 | 非 Hermes 根目录告警 | 兼容开关为 0，tar 返回根目录 changed | 脚本非零，不调用上传 | P0 |
| 6 | 新增根目录业务文件 | 打包期间新增 `new-business.json` | 指纹变化，脚本非零，不调用上传 | P0 |
| 7 | 替换嵌套业务文件 | 打包期间原子替换 `sessions/state.json` | 递归指纹变化，脚本非零，不调用上传 | P0 |
| 8 | 脚本语法检查 | `bash -n scripts/export_migration.sh` | 通过 | P0 |
| 9 | ShellCheck | `shellcheck scripts/export_migration.sh` | 无 error 级问题 | P1 |
| 10 | Go 回归 | `go test ./controller -run 'Migration|ExportMigration'` | 通过 | P0 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | Hermes 导出脚本契约 | 创建真实 Hermes 实例并调用 export | 脚本使用 `.hermes`、开启专属兼容、携带正确排除清单及安全校验逻辑 | P0 |
| 2 | 未上传状态与导入保护 | 不执行返回脚本，调用 status/import | status 显示未就绪，import 返回 400 | P0 |
| 3 | Hermes 面板开启时真实导出/import | `HERMES_MIGRATION_E2E=1`，通过 TAT 写标记并执行返回脚本 | SMH 上传完成，删除标记后 import 可恢复标记，迁移状态 done | P0 |
| 4 | 日志排障 | 人为造成 tar fatal 或上传失败 | 日志包含失败阶段/退出码，不含凭证 | P1 |

前两项由 `test_instance_migration_hermes.py` 默认执行；第三项由同一文件实现，但需要显式设置
`HERMES_MIGRATION_E2E=1`，避免普通 CI 意外执行有副作用的恢复操作。SMH 未配置时在创建 CVM
前整组跳过。第四项仍需人工故障注入；环境不具备时在 `06-it.md` 明确记录。

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 容忍范围过宽导致迁移数据不一致 | 低 | 高 | 仅 Hermes 开启；精确匹配根目录 changed，且业务树前后指纹必须一致 |
| 2 | tar 状态码/文案在不同版本变化 | 中 | 中 | `LC_ALL=C` 固定文案；状态码与完整 stderr 双重判断 |
| 3 | 日志目录影响归档 | 低 | 中 | 打包前创建，且 `logs` 已在排除列表 |
| 4 | 归档可列出但业务数据仍非强一致快照 | 低 | 中 | 递归比较未排除业务树元数据指纹，并在放行后验包 |
| 5 | 日志泄漏敏感凭证 | 低 | 高 | 不使用 xtrace、不打印 URL/header/token，Review 检查日志内容 |

## 回滚

只需回滚 `scripts/export_migration.sh`、对应测试和文档；无数据库或状态迁移。

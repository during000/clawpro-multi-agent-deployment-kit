# 03. Implement — 实现记录

## 关键实现细节

- `scripts/export_migration.sh`
  - 为导出过程增加 `packing`、`validating_archive`、`uploading`、`confirming_upload` 等阶段标识。
  - 捕获 tar 退出码和 stderr；仅当 Hermes 兼容开关为 1、退出码为 1，且全部非空诊断行
    精确等于 `tar: <agent-root>: file changed as we read it` 时进入候选放行分支。
  - 打包前后递归生成未排除业务树指纹；根级或嵌套业务数据发生新增、删除、修改、替换时
    中止上传。目录仅记录稳定字段，避免被已排除子项的元数据变化干扰。
  - 允许继续前强制执行 `tar tf`，归档不可读时终止且禁止上传。
  - 将终端输出同步持久化到 `<agent_home>/logs/migration_export_<timestamp>_<pid>.log`，
    权限为 `0600`；退出时输出失败阶段、退出码和日志路径。
  - 通过 `EXIT` trap 统一清理诊断临时文件、响应文件和失败归档，并修复原确认失败分支中
    将两个路径拼成一个参数、导致清理无效的问题。
  - header 读取改用 Bash 3 可用的循环，避免 `readarray` 在旧环境不可用。
- `controller/openclaw_migration_test.go`
  - 新增执行真实导出脚本的 fake tar/curl 测试夹具。
  - 覆盖被排除运行时文件变化放行、非 Hermes 拒绝、业务文件告警拒绝、根级业务项新增
    拒绝、嵌套业务文件替换拒绝、验包失败拒绝和正常导出回归。
- `controller/openclaw_migration.go`
  - 按兼容运行时类型生成开关，仅 Hermes（含兼容 Hermes 的自定义类型）开启。
  - 将排除路径清单以 base64 JSON 传给脚本，用于业务树指纹过滤。
- `test/scripts/openclaw_instance/test_instance_migration_hermes.py`
  - 在 SMH 配置完整时创建真实 Hermes 实例，验证实例类型以及 export/status/import 契约。
  - 解码并精确检查导出脚本的 Hermes 排除路径，同时检查 `.hermes`、专属兼容开关、
    业务树指纹、诊断分类和归档校验标记。
  - export 请求关闭帧输出，避免脚本内临时 SMH 凭证进入 CI 日志；`finally` 中清理实例。
  - SMH 未配置时在创建 CVM 前退出为 SKIP。
  - SMH 前置检查使用集成框架注入的 `BOOTSTRAP_ADMIN_TOKEN`；该配置接口未由
    `WithOpenAPI` 包装，不能使用动态管理员用户的 API Token。
  - `HERMES_MIGRATION_E2E=1` 时，通过无帧输出的 TAT API 写入标记、执行 export、确认
    SMH 文件就绪、删除标记、触发 import，并在状态 done 后读取标记验证真实恢复闭环。
- `docs/API.md`
  - 记录兼容边界、持久化日志路径和敏感信息约束。

## 与 Plan 差异

在安全复核后，将原“精确诊断 + 验包”进一步收紧为“Hermes 专属 + 业务树前后指纹一致
+ 精确诊断 + 验包”。此外修复同一脚本已有的失败清理参数错误，并将 header 读取改为
Bash 3 兼容写法；均不改变 API 契约。

## 检查项

- [x] `gofmt` 格式化通过
- [ ] `go vet ./...` 无错误（仓库已有 `skillhubclient/client_test.go:278` 告警；`go vet ./controller` 通过）
- [x] `bash -n` 与 ShellCheck 通过
- [x] 写接口审计日志保持不变
- [x] 无数据库变更
- [x] 无硬编码密钥/配置
- [x] 持久化日志不包含 SMH 凭证
- [x] 集成测试不输出包含临时 SMH 凭证的 export 响应

# 07. Review — 代码审查

## 审查方式

- [x] AI 自动 Review
- [ ] 人工 Review（待合入流程）

## 发现的问题

- 无本次改动引入的高或中严重度问题。
- 安全边界：仅 Hermes 可进入兼容分支；只接受状态码 1 + 精确根目录诊断，并要求未排除
  业务树前后指纹一致；任何额外/不同诊断或业务变化都失败，且放行后仍验包。
- 可观测性：失败日志包含阶段和退出码；脚本没有启用 xtrace，也不打印凭证、header 或完整 URL。
- 兼容性：脚本由生成器明确使用 `bash` 执行；header 读取兼容 macOS 自带 Bash 3 和现网旧版 Bash。
- 性能：仅 Hermes 在打包前扫描一次未排除业务树，只有命中候选根目录告警时再扫描一次；
  `hermes-agent`、缓存、日志和 sandbox 等大目录已排除，额外成本为业务数据元信息遍历。
- 依赖：业务树指纹使用 `python3`；原上传 header 解析已依赖 `python3`，未新增运行时依赖。
- 清理：失败归档、tar/验包诊断文件和 curl 响应文件由 trap 回收。
- 已有低风险项：全仓 vet 命中 `skillhubclient/client_test.go:278`，与本次迁移模块无关，未扩大范围处理。
- Review follow-up：不再忽略排除路径 `json.Marshal` 的返回错误；序列化失败时终止脚本生成，
  由上层统一包装为“生成迁移脚本失败”。
- 集成测试 follow-up：Hermes 专用用例默认只校验返回脚本，不执行含凭证的命令；显式开启
  真实 E2E 时，export 及 TAT run/describe 客户端均关闭
  帧输出，断言失败也不回显成功响应，避免临时 SMH token/header/URL 或命令正文进入 CI 日志。
- 资源清理：SMH 未就绪时不创建 CVM；创建后无论断言成功、失败或 `run_tests` 退出，均在
  `finally` 中查询并删除该测试用户的实例，删除失败时明确交由全局 cleanup 兜底。
- 鉴权边界：`/admin/smh/config` 保持非 OpenAPI 敏感接口；集成测试仅使用框架注入的
  启动级管理员 Token 做前置只读检查，不扩大动态管理员 API Token 的权限。

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致
- [x] 安全基线检查通过

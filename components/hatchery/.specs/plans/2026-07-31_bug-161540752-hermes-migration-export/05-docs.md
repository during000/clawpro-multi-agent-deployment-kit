# 05. Docs — 文档更新

## 更新清单

- `docs/API.md` 的 `POST /openclaw/migration/export` 增加“导出稳定性与日志”：
  - 日志文件路径、失败阶段和退出码；
  - 兼容仅对 Hermes 生效，且必须同时满足精确根目录告警、业务树指纹一致和验包成功；
  - 业务树新增/删除/修改/替换、其他 tar 错误和验包失败仍终止；
  - 日志不记录 SMH token、header 或上传 URL。
- `.specs/plans/2026-07-31_bug-161540752-hermes-migration-export/` 完整记录
  Clarify、Plan、Implement、UT、Docs、IT、Review 和 Commit。
- 执行 `python3 test/api_md_to_openapi.py` 成功，生成结果未产生额外 Git 差异。

## 检查项

- [x] `docs/API.md` 已说明导出日志与 Hermes 活跃面板兼容行为
- [x] API 请求/响应契约未变化
- [x] `.specs` 记录与最终实现一致

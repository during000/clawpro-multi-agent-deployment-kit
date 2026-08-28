# 02. Plan — 方案设计（TAPD 160256132）

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `controller/admin_skills.go` | 修改 | 严格校验文件名 UTF-8，修复 ZIP flag |
| `controller/admin_skills_test.go` | 修改 | 增加多种旧编码、缺失 flag、非法编码回归用例 |
| `i18n/keys.go` | 修改 | 增加非 UTF-8 文件名的中文错误 |
| `i18n/en.go` | 修改 | 同步英文翻译 |
| `docs/API.md` | 修改 | 记录文件名编码行为与 400 响应 |
| `skillhubclient/client_test.go` | 修改 | 修复全仓 `go vet` 的既有错误检查告警 |
| `.specs/plans/2026-07-31_bug-160256132-enterprise-skill-filename-utf8/*` | 新增 | 00–08 SOP 产物 |

## 调用链 / 数据流

```text
POST /admin/skills/create
  → ParseMultipartForm
  → validateSkillZip
      → archive/zip 读取原条目
      → 遍历所有 entry 校验文件名字节为 UTF-8
      → 非 UTF-8：立即返回 400，不猜测编码
      → 合法 UTF-8：统一路径分隔符
      → 既有路径/字符/大小校验
      → 新 FileHeader 清除 NonUTF8 并重打包
  → injectMetaIntoZip
  → 生成 UTF-8 file list / SMH key
  → DB 事务 + SMH 上传
```

## 数据库变更

无。

## 测试用例设计

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 多种本地编码 ZIP | GB18030、Big5、Shift-JIS 文件名 | 返回明确的非 UTF-8 i18n 错误 | P0 |
| 2 | 合法 UTF-8 但缺失 ZIP flag | UTF-8 中文文件名且 `NonUTF8=true` | 文件名不被二次转码，输出修复 UTF-8 flag | P0 |
| 3 | 非法字节 | 文件名含非法字节 `0xff` | `validateSkillZip` 返回相同 i18n 错误 | P0 |
| 4 | 既有合法 ZIP 回归 | `SKILL.md` + 普通源码文件 | 文件列表与规范化 ZIP 正常 | P1 |
| 5 | 既有安全校验回归 | 空 ZIP、缺少 SKILL.md、特殊字符 | 保持原错误行为 | P1 |
| 6 | HTTP 错误响应 | multipart ZIP 含非 UTF-8 文件名 | `POST /admin/skills/create` 返回 400 且包含明确提示 | P0 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 企业技能创建端到端 | 含非 UTF-8 文件名的 multipart ZIP | HTTP 400，响应明确提示转换 UTF-8；不调用 SMH | P0 |

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 自动转码误判原始编码 | 中 | 高 | 不猜测编码；所有非 UTF-8 文件名统一拒绝 |
| 2 | 合法 UTF-8 被错误拒绝 | 低 | 高 | 依据 `utf8.ValidString` 判断，不依赖可能缺失的 `NonUTF8` flag |
| 3 | 被忽略目录含非 UTF-8 文件名 | 低 | 中 | 在锚点筛选前校验 ZIP 的全部 entry |
| 4 | 修改原 Header 后 UTF-8 flag 仍未设置 | 低 | 高 | 清除 `NonUTF8`，重新读取输出 ZIP 验证 |

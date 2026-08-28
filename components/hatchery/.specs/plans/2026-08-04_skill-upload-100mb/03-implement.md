# 03. Implement — 实现记录

---

## 关键实现细节

1. **`skillUploadMaxSize`**：`50 << 20` → `100 << 20`，`ParseMultipartForm` 与文件大小校验共用。
2. **抽出 `isSkillUploadTooLarge(size int64) bool`**：`size > skillUploadMaxSize`，供边界 UT 覆盖，避免构造真实 >100MB multipart。
3. **i18n 拆分**：
   - 新增 `MsgSkillUploadFileSizeTooLarge`（「文件大小超过 100MB 限制」）供上传路径使用
   - 保留 `MsgSkillFileSizeTooLarge`（50MB）给 Bundle 远端下载
4. **UT 新增** `controller/skill_upload_test.go`：常量断言、边界、文案拆分校验。
5. **docs/API.md**：按 SOP 留到 Docs 阶段更新。

## 与 Plan 差异

无

## 改动文件

| 文件 | 变更 |
|------|------|
| `controller/skill_upload.go` | 常量 100MB + `isSkillUploadTooLarge` + 新 i18n Key |
| `controller/skill_upload_test.go` | 新增（Plan UT #1–#4） |
| `controller/admin_skills_test.go` | FileTooLarge 注释 50→100MB |
| `i18n/keys.go` | 新增 `MsgSkillUploadFileSizeTooLarge` |
| `i18n/en.go` | 英译 `File size exceeds the 100MB limit` |

## 检查项

- [x] `gofmt` 格式化通过
- [x] `go vet ./controller/ ./i18n/` 无错误
- [x] 写接口已添加审计日志（无新写接口，不适用）
- [x] 数据库变更已同步（无 DB 变更）
- [x] 使用 `model.DB(r.Context())`（未改 DB 访问）
- [x] 无硬编码密钥/配置
- [x] 用户可见文案使用 i18n，新增 Key 已同步 `en.go`
- [x] 新 UT 本地已通过：`TestSkillUploadMaxSize_Is100MB` / `TestIsSkillUploadTooLarge_Boundary` / `TestMsgSkillUploadFileSizeTooLarge_Mentions100MB`

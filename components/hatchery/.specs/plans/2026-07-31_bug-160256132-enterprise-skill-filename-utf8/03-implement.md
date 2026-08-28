# 03. Implement — 实现记录（TAPD 160256132）

## 关键实现细节

1. `validateSkillZip` 在路径识别前遍历 ZIP 全部 entry，并用 `utf8.ValidString` 校验文件名原始字节。
2. 任一文件名不是合法 UTF-8 时返回 `MsgZipFileNameNotUTF8`；不猜测或转换 GB18030、Big5、Shift-JIS 等本地编码。
3. 只有通过 UTF-8 校验后才统一 Windows `\` 路径分隔符，并继续执行既有锚点、Zip Slip、特殊字符和大小校验。
4. 重打包时设置 `newHeader.NonUTF8 = false`；非法编码的 entry comment 被清空，确保合法 UTF-8 中文名写入正确 flag。
5. 校验失败发生在数据库事务和 SMH 上传前，HTTP handler 按既有校验错误路径返回 400。

## 与 Plan 差异

- 初版按 GB18030 自动转码；检视意见指出 ZIP 不携带具体本地编码，其他编码可能被错误解码。最终改为严格拒绝，并新增 GB18030、Big5、Shift-JIS 回归测试。
- 为满足模板的全仓 `go vet ./...` 门禁，补充修复了 `skillhubclient/client_test.go` 中既有的 `http.Get`、`io.ReadAll`、`json.Unmarshal` 错误检查；不影响生产代码。
- IT 因 kubeconfig 无法访问集群且未提供指定 AK/SK，未执行部署和 SMH 端到端验证，详见 `06-it.md`。

## 检查项

- [x] `gofmt` 格式化通过
- [x] `go vet ./...` 无错误
- [x] 写接口已有 `WithAudit()`，本任务未新增路由
- [x] 无数据库变更，不需要 `sql/init.sql` 或 migration SQL
- [x] 未新增 DB 访问；既有 handler 使用 `model.DB(r.Context())`
- [x] 无硬编码密钥/配置
- [x] 用户可见文案使用 i18n Key，`en.go` 已同步英文翻译

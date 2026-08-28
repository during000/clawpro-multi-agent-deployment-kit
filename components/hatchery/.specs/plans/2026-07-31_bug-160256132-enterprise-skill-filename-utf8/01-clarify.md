# 01. Clarify — 需求澄清（TAPD 160256132）

## 背景

TAPD 缺陷 `160256132`：企业技能通过 `POST /admin/skills/create` 上传后，部分 ZIP 内中文文件名显示为乱码，逐文件上传 SMH 时返回 `400 Bad Request`。网盘侧确认 object key 中存在非 UTF-8 文件名字节。

调用链中的现状：

```text
multipart ZIP
  → validateSkillZip
  → 复制原 FileHeader 重打包
  → injectMetaIntoZip
  → 逐条使用 f.Name 生成 SMH fileKey
  → storageClient.Upload
```

`validateSkillZip` 虽然重新打包 ZIP，但原样继承 `FileHeader.NonUTF8` 和文件名字节。旧编码条目因此仍可能进入 SMH key。ZIP 只标识“是否为 UTF-8”，不记录具体本地编码；将所有非 UTF-8 字节按 GB18030 解码会把 Big5、Shift-JIS 等输入静默转换成错误名称。

## 目标

- [x] 企业技能 ZIP 中的合法 UTF-8 文件名保持不变，并写入正确 UTF-8 标记
- [x] ZIP 中任一非 UTF-8 文件名在 SMH 调用前返回明确的 HTTP 400
- [x] 错误提示明确要求客户统一转换为 UTF-8 后重新打包上传
- [x] 保持原有 `SKILL.md` 锚点、非法字符、Zip Slip 和解压大小校验
- [x] 更新 `docs/API.md`，本任务没有其他 `.specs/docs/` 模块 TODO

## 范围

| 包含 | 不包含 |
|------|--------|
| `POST /admin/skills/create` 的 ZIP 文件名 UTF-8 校验 | 插件、规则等其他资产上传链路 |
| 非 UTF-8 文件名统一拒绝 | 任意编码自动探测或转码 |
| UTF-8 ZIP flag 修复 | 修改 ZIP 文件内容编码 |
| 拒绝时的中英文错误 | 修改 SMH SDK |
| UT、API 文档、SOP 产物 | 数据库 Schema、权限、审计规则变更 |

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 非 UTF-8 文件名应直接拒绝还是自动转换 | 已确认 | 直接拒绝；编码不可可靠推断，提示客户转换为 UTF-8 后重新打包 |
| 2 | 是否信任 ZIP `NonUTF8` 标记强制转码 | 已确认 | 不强制；部分工具会漏设 UTF-8 flag，字节合法时原样保留 |
| 3 | 是否需要更新公开 API 文档 | 已确认 | 接口行为变化，更新 `docs/API.md` |
| 4 | 是否涉及数据库迁移 | 已确认 | 无 |

## 约束与依赖

- Go 标准库 `archive/zip` 不自动将本地编码转换为 UTF-8。
- SMH object key 必须是 UTF-8。
- IT 需要 Docker、TCR 登录、可访问 K8s kubeconfig 和指定账号 AK/SK。

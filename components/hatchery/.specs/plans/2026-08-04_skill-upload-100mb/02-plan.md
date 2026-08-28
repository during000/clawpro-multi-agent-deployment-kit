# 02. Plan — 方案设计

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `controller/skill_upload.go` | 修改 | `skillUploadMaxSize`：`50 << 20` → `100 << 20`；可选抽出 `isSkillUploadTooLarge` 便于边界 UT |
| `i18n/keys.go` | 修改 | 上传超限文案改为 100MB；**与 Bundle 下载文案拆分**（见下方「关键设计」） |
| `i18n/en.go` | 修改 | 同步英译；注册新 Key |
| `docs/API.md` | 修改 | `/admin/skills/create`、`/openclaw/skills/contribute` 两处「最大 50MB」→「最大 100MB」；可加一句网关 body 限制提醒 |
| `controller/admin_skills_test.go` | 修改 | 更新 `TestHandleCreateSkill_FileTooLarge` 注释中的 50MB→100MB；新增常量/边界 UT |
| `controller/admin_skill_bundle.go` | 修改（文案侧） | 继续使用 **50MB** 专用文案 Key，避免上传改 100MB 后误导 Bundle 下载错误提示 |

**明确不改：**

| 文件 | 原因 |
|------|------|
| `controller/admin_skills.go`（`maxUncompressedSize`） | Clarify：解压 200MB 维持 |
| `controller/admin_skill_bundle.go`（`maxSkillBundleZipDownloadSize`） | Clarify：Bundle 下载上限不动 |
| 安全扫描 `maxScanFileSize` | Clarify：7MB 跳过逻辑不动 |
| `sql/*` / `model/*` | 无 schema 变更 |
| `main.go` / `audit.go` | 无新路由 |

## 关键设计：i18n 文案拆分

现状：`MsgSkillFileSizeTooLarge`（「文件大小超过 50MB 限制」）被两处共用：

1. Skill **上传**（`skill_upload.go`）→ 本期改为 100MB
2. Skill Bundle **远端下载**（`admin_skill_bundle.go`）→ 仍为 50MB

若只改同一条文案为 100MB，Bundle 超 50MB 时会错误提示「超过 100MB」，属于文案 bug。

**方案（采用）：**

- 新增上传专用 Key：`MsgSkillUploadFileSizeTooLarge` = `"文件大小超过 100MB 限制"`（英：`File size exceeds the 100MB limit`）
- 保留原 `MsgSkillFileSizeTooLarge` = `"文件大小超过 50MB 限制"` 供 Bundle 下载继续使用
- `prepareSkillUploadFromForm` 改为使用新 Key

## 调用链 / 数据流

```
POST /admin/skills/create
  → HandleCreateSkill
    → prepareSkillUploadFromForm
         ParseMultipartForm(skillUploadMaxSize)   // 100MB
         FormFile("file")
         if header.Size > skillUploadMaxSize      // 100MB → MsgSkillUploadFileSizeTooLarge
         validateSkillZip / injectMeta / COS keys
    → uploadSkillPackageToStorage …

POST /openclaw/skills/contribute
  → HandleContributeSkill
    → ParseMultipartForm(skillUploadMaxSize)      // 100MB（预解析字段）
    → prepareSkillUploadFromForm                  // 同上上限
    → 写 Skill(pending_review) + ReviewRequest …

不在本期：
POST Bundle 相关下载路径
  → maxSkillBundleZipDownloadSize (50MB) + MsgSkillFileSizeTooLarge（保留）
```

## 数据库变更

无

## 测试用例设计（自然语言描述）

> 先于实现编写，Implement 阶段据此编码。
> UT 用例走 `go test`，IT 用例走 Python 集成测试（`test/scripts/`）。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 常量值正确 | 读取 `skillUploadMaxSize` | 等于 `100 << 20`（104857600） | P0 |
| 2 | 边界：恰好等于上限 | `size == skillUploadMaxSize` | `isSkillUploadTooLarge` 为 false（允许） | P0 |
| 3 | 边界：超过 1 字节 | `size == skillUploadMaxSize+1` | `isSkillUploadTooLarge` 为 true（拒绝） | P0 |
| 4 | 上传超限文案 | 检查 `MsgSkillUploadFileSizeTooLarge` 中文 Key / 英译 | 含「100MB」/ `100MB`；与 Bundle 的 50MB 文案不同 | P0 |
| 5 | 回归注释更新 | `TestHandleCreateSkill_FileTooLarge` | 注释改为 100MB；行为不因无超大文件而 panic（保持现有弱断言或改为 Skip 说明） | P1 |

> 说明：真实构造 >100MB multipart 在 UT 中成本过高；边界用例通过抽出 `isSkillUploadTooLarge(size int64) bool`（一行包装 `size > skillUploadMaxSize`）覆盖，handler 仍走该函数。

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 小包上传回归 | 现有 IT：小 zip 调 `/admin/skills/create`（及 contribute 若已有） | 仍成功，无回归 | P0 |
| 2 | 超大包端到端 | 实际上传 >100MB | **本期不做**：CI 体积/耗时不可接受；由 UT 边界 + 文档契约覆盖 | — |

> IT 阶段：跑现有 skill create/contribute 相关脚本冒烟即可；在 `06-it.md` 注明「100MB 超限」无 IT、由 UT 覆盖。

## Docs 阶段预览（Implement 后执行）

- `docs/API.md`：两处参数说明 50MB → 100MB
- 可选：在 create/contribute 小节加备注「若前置反向代理限制更小，需同步调大 body 限制」
- 无新模块 TODO 文档需要补全（Clarify 目标中的 docs TODO 对本需求不适用，以 API.md 为准）

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 共用 i18n 文案导致 Bundle 提示错误 | 高（若不拆） | 中 | 拆分上传/下载文案 Key（见关键设计） |
| 2 | 部署网关 body ≤50MB | 中 | 高（功能仍失败） | Docs 备注；部署侧确认 |
| 3 | 整包入内存峰值翻倍 | 低 | 中 | 本期不改流式；监控即可 |
| 4 | UT 难覆盖真实超大 multipart | 高 | 低 | 边界函数 UT + 常量断言 |

## 实现顺序建议

1. 改 `skillUploadMaxSize` + 抽出 `isSkillUploadTooLarge`
2. 新增 i18n Key（中/英），上传路径改用新 Key
3. 更新 `docs/API.md`（也可放到 Docs 阶段；Plan 要求 Implement 可先改或留给 05）
4. 补 UT（常量 + 边界 + 文案）
5. 更新旧测试注释

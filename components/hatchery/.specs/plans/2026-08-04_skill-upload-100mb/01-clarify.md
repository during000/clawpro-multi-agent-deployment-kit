# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

当前 Skill 上传压缩包上限为 **50MB**，由常量 `skillUploadMaxSize`（`controller/skill_upload.go`）统一控制，覆盖：

1. **管理端创建技能** `POST /admin/skills/create`（经 `prepareSkillUploadFromForm`）
2. **员工共建提交** `POST /openclaw/skills/contribute`（共用同一常量 / 预处理逻辑）

校验链路：`ParseMultipartForm(skillUploadMaxSize)` + `header.Size > skillUploadMaxSize` → 返回 i18n 文案「文件大小超过 50MB 限制」。

业务侧反馈部分技能包已超过 50MB，需要将上传上限调整为 **100MB**。

**关联但不同的限制（本期不动）：**

| 限制点 | 当前值 | 说明 |
|--------|--------|------|
| `skillUploadMaxSize` | 50MB → **100MB** | 本次目标 |
| zip 解压后总大小 `maxUncompressedSize` | 200MB | 维持不变 |
| 安全检测跳过阈值 `maxScanFileSize` | 7MB | 维持不变 |
| Skill Bundle 远端下载 `maxSkillBundleZipDownloadSize` | 50MB | 维持不变 |
| Plugin 上传上限 | 200MB | 不在范围 |

## 目标

- [x] Skill 上传 zip 包大小上限从 50MB 提升为 **100MB**（`100 << 20`）
- [x] 管理端创建与员工共建两条上传路径行为一致
- [x] 超限错误文案（中/英 i18n）与 `docs/API.md` 参数说明同步为 100MB
- [x] 相关单测注释/断言中的 50MB 表述一并更新

### 验收标准

1. 上传 ≤100MB 的合法 zip，管理端 create / 员工 contribute 均可成功受理（其余校验仍按现有规则）。
2. 上传 >100MB 的文件，返回 400，错误信息体现 100MB 限制。
3. 解压后总大小仍按 200MB 限制；安全检测 7MB 跳过逻辑不变。
4. API 文档与 i18n 中不再出现「Skill 上传最大 50MB」的过时描述。

## 范围

| 包含 | 不包含 |
|------|--------|
| 调整 `skillUploadMaxSize` 50MB → 100MB | 调整 Plugin 上传上限（已是 200MB） |
| 管理端 `/admin/skills/create` | 调整 zip 解压后 200MB 上限 |
| 员工端 `/openclaw/skills/contribute` | 调整安全检测 7MB 跳过阈值 |
| i18n 中/英文案 `MsgSkillFileSizeTooLarge` | 调整 Skill Bundle 远端下载 50MB 上限 |
| `docs/API.md` 两处「最大 50MB」说明 | 前端上传控件/提示（若有，由前端另跟） |
| 相关 UT 中 50MB 阈值表述 | 仓内网关/nginx body 限制配置；部署侧自行确认 |

## Challenge（产品质疑）

1. **为何不是直接对齐 Plugin 的 200MB？**  
   需求明确写 100MB；Skill 解压上限已是 200MB，压缩包 100MB 与之匹配更合理。若产品后续要再放大，可另开需求。

2. **内存占用**  
   当前实现会将整包读入内存再校验/上传。50→100MB 会使单次上传峰值内存约翻倍；并发大包上传时需关注。本期按需求放宽常量，不引入流式改写（工作量与风险更大）。

3. **Bundle 下载 50MB**  
   与「本机上传」不是同一入口。本期明确不动。

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 管理端 create 与员工 contribute 是否一并改为 100MB？ | 已确认 | 是（共用常量，行为一致） |
| 2 | zip 解压后 200MB 上限是否维持不变？ | 已确认 | 维持 200MB |
| 3 | Skill Bundle 远端下载上限（50MB）是否一并改为 100MB？ | 已确认 | 否，本期不动 |
| 4 | 安全检测 7MB 跳过阈值是否调整？ | 已确认 | 否，本期不动 |
| 5 | 是否需要同步排查/调整网关（nginx 等）的 `client_max_body_size`？ | 已确认 | 不改仓内代码；部署侧自行确认，文档可备注提醒 |

## 约束与依赖

- 公共 API 文档字段语义变更：仅放宽上限，不删字段、不改变成功响应结构 → **非破坏性变更**。
- 无 DB / migration 变更。
- 审计规则与路由不变。
- 若部署前有反向代理 body 限制 ≤50MB，仅改后端仍会被网关拦下，需部署侧确认。

# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review
- [x] 人工 Review（MR CR）

## 发现的问题

### AI Review 发现

| # | 文件 | 问题 | 严重度 | 状态 |
|---|------|------|-------|------|
| 1 | `controller/skillhub.go` | `requireSMHEnabled` 在灰度检查之前调用，导致灰度未开启时也报错 | 中 | 已修：移到灰度检查后 |
| 2 | `controller/skillhub.go` | `InvalidateSkillHubCache` 缓存失效逻辑不必要 | 低 | 已修：删除该方法 |
| 3 | `model/site_config.go` | `OneIDPrivateKey` 字段冗余（已有 `OneIDClientSecret`） | 中 | 已修：移除该字段，复用 `OneIDClientSecret` |
| 4 | 整体设计 | `OneIDTokenProvider` 对象过度设计 | 中 | 已修：改为 `sync.Mutex+map` 匹配现有模式 |

### 人工 CR 发现

| # | 文件 | 问题 | 严重度 | 状态 |
|---|------|------|-------|------|
| 1 | `common/context.go` | 不必要的注释改动 | 低 | 已修：还原 |
| 2 | `model/site_config.go` | 不必要的注释改动 | 低 | 已修：还原 |
| 3 | `controller/skillhub.go` | 端点名 `/admin/skillhub-state` 不准确 | 低 | 已修：改为 `/admin/skillhub-status` |

### CI/CD 发现

| # | 问题 | 严重度 | 状态 |
|---|------|-------|------|
| 1 | MySQL "Row size too large" — `skill_hub_api_url varchar(512)` | 高 | 已修：改为 `TEXT` 类型 |
| 2 | Schema check COMMENT 不匹配 — `init.sql` 缺 COMMENT | 中 | 已修：添加 `COMMENT '是否启用 SkillHub 迁移'` |

---

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致
- [x] 安全基线检查通过（无裸 SQL、无硬编码密钥、权限检查到位）
- [x] 装饰器模式不侵入现有代码
- [x] 错误处理返回 502 不降级

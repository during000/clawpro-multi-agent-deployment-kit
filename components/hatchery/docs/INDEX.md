# 功能文档索引

> 按管控端侧边栏模块组织。`.specs/docs` 是 `docs/` 的软链接，两者同一目录。
> 完整 API 接口文档：`docs/API.md`。

---

## 目录结构

```
docs/
├── API.md                    ← 完整 API 接口文档（OpenAPI 生成源）
├── openapi.json              ← 自动生成，勿手动编辑
├── i18n.md                   ← 国际化方案与接入指南
├── testing.md                ← 单测编写指引（DB 初始化、异步任务）
│
├── basic/                    ← 基础信息
│   ├── auth.md                   TODO
│   ├── api_token.md              API Token 管理
│   ├── user-management.md        TODO
│   ├── user_group.md             用户组技术方案
│   ├── unified-account-mode.md   统一账号 OneID 映射
│   └── platform-policy.md        TODO
│
├── agent-config/             ← Agent 配置
│   ├── design-model-permission-template.md   模型可见范围设计
│   ├── channel-config.md                     TODO
│   ├── skill-visibility-api.md               技能应用范围接口
│   ├── agent-tool-library.md                 TODO
│   ├── multi-agent-image-design.md           Agent 类型/镜像管理方案
│   ├── hermes-lightclaw-ace-design.md        多 Agent 系统设计 V3
│   ├── multi-agent-compat-requirements.md    多 Agent 兼容改造
│   └── network-management.md                 TODO
│
├── ops/                      ← 运维与观测
│   ├── agent-list.md             TODO
│   ├── tokens-monitor.md         TODO
│   ├── ops-observation.md        TODO
│   └── logging.md                日志使用指引
│
├── agent-service/            ← Agent 服务
│   ├── memory-management.md      TODO
│   ├── smh-skill-upgrade-design.md   网盘/SMH 升级方案
│   ├── sdk_ws_url_api.md             SDK WebSocket 连接
│   ├── llm-proxy.md                  TODO
│   └── cloud_proxy_api.md            腾讯云 API 透传
│
└── security/                 ← 安全审计
    ├── ai-agent-security.md      TODO
    ├── session-management.md     TODO
    └── audit-log.md              TODO
```

## 新增文档时

1. 写入 `docs/<模块>/` 对应子目录
2. 更新本索引
3. API 接口文档统一维护在 `docs/API.md`（参数表格式见 `CLAUDE.md`）

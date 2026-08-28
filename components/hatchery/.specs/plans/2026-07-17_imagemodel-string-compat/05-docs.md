# 文档更新清单

## 结论：无需更新文档

本次改动仅涉及 `scripts/set_model.sh`、`scripts/switch_model.sh`、`scripts/remove_model_provider.sh` 三个脚本的 jq 表达式内部逻辑（增加 `imageModel` 类型判断），不改变：

- **API 接口契约**：`POST /openclaw/set-model`、`POST /openclaw/switch-primary-model`、`POST /openclaw/delete-model` 的请求/响应格式不变
- **TAT 参数契约**：`valueb64`、`provider`、`model`、`primary`、`fallbacksb64`、`imageprimary`、`imagefallbacksb64` 参数不变
- **脚本分派表**：`ResolveScript("set_model", agent_type)` 映射不变
- **数据库 schema**：无变更
- **i18n 文案**：无变更

## 检查的文档

| 文档 | 是否涉及 | 说明 |
|------|---------|------|
| `docs/API.md` | 涉及脚本描述 | 仅描述脚本功能和 TAT 参数，未涉及 imageModel 格式，无需更新 |
| `docs/agent-config/multi-agent-compat-requirements.md` | 涉及脚本分派表 | 仅描述脚本分派映射，无需更新 |
| `docs/agent-config/design-model-permission-template.md` | 涉及审计规则 | 仅描述审计事件，无需更新 |

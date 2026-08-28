package model

import "gorm.io/gorm"

// LocalAgentCLSCredential 本地 agent 拉取 CLS 公网上报配置所需的永久 AK/SK。
//
// 与 site_config.CVMSecretId/Key 的区别：后者是 hatchery 调腾讯云 API 的凭据，
// 不应直接下发给本地 agent；本表存的是一份可暴露给外部、专用于 CLS 公网上报的凭据。
//
// 按租户隔离（identifier）：不同租户持有各自的 CLS AK/SK，互不可见。
// 读写走常规 model.DB(ctx)，由 db.go 的 identifier 回调自动按当前租户过滤。
//
// 凭据由运维按租户写入（不在管理端 API / 本地 agent 调用链里写），
// 按需求明文存储、不加密、不轮换。topic_id 不落本表，由 get-config 实时从
// CLS OpenClawService 查询返回。
type LocalAgentCLSCredential struct {
	gorm.Model
	Identifier string `gorm:"uniqueIndex:idx_lacc_ident_type,priority:1;not null;default:''" json:"-"` // 多租户标识；回调自动注入/过滤
	ConfigType string `gorm:"type:varchar(32);not null;default:'cls';uniqueIndex:idx_lacc_ident_type,priority:2"`
	SecretID   string `gorm:"type:varchar(256);not null"`
	SecretKey  string `gorm:"type:varchar(512);not null"` // 明文落库（按需求不加密）
}

// LocalAgentCLSCredentialTable 表名常量，便于引用。
const LocalAgentCLSCredentialTable = "local_agent_cls_credentials"

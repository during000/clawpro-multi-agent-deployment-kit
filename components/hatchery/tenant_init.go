package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"hatchery/common"
	"hatchery/model"
)

type CmdlineConfig struct {
	Identifier       string
	Uin              string
	Domain           string
	InternalSecret   string
	Lang             string
	SecurityPolicies []string
}

// backfillSiteConfig 回填 site_configs 中为空的租户级字段。
//
// 回填规则很简单：数据库字段为空时才写入，不为空则保留原值。
// 数据来源：
//   - uin/domain/internalSecret/lang 来自启动参数
//   - oneIDAccountID/AKSK 来自环境变量
//
// 该函数必须在 buildFixedSnapshot 之前调用，确保 DB 中的 AKSK 字段已被回填。
func backfillSiteConfig(cmdlineConfig CmdlineConfig) {
	// 构造启动期 ctx：始终注入 TenantSnapshot（SQLite 模式下 Identifier 为空但仍注入）。
	initCtx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: cmdlineConfig.Identifier})

	var config model.SiteConfig
	model.DBGlobal(initCtx).Where("identifier = ?", cmdlineConfig.Identifier).First(&config) // 可能不存在（首次运行），忽略错误

	updates := map[string]interface{}{}

	// 回填启动参数：数据库为空才写入
	if config.Uin == "" && cmdlineConfig.Uin != "" {
		updates["uin"] = cmdlineConfig.Uin
	}
	if config.Domain == "" && cmdlineConfig.Domain != "" {
		updates["domain"] = cmdlineConfig.Domain
	}
	if config.InternalSecret == "" {
		if cmdlineConfig.InternalSecret == "" {
			cmdlineConfig.InternalSecret = os.Getenv("INTERNAL_SECRET")
		}
		if cmdlineConfig.InternalSecret != "" {
			updates["internal_secret"] = cmdlineConfig.InternalSecret
		}
	}
	// 在非多租户模式下启动参数 --lang 覆盖数据库 site_config 配置
	if cmdlineConfig.Lang != "" {
		updates["default_lang"] = cmdlineConfig.Lang
	}
	// 非多租户模式下安全策略启动参数覆盖数据库 site_config 配置
	updates["security_policies"] = strings.Join(cmdlineConfig.SecurityPolicies, ",")

	// 回填环境变量：数据库为空才写入
	if config.OneIDAccountID == "" {
		if oneID := os.Getenv("ONEID_ACCOUNT_ID"); oneID != "" {
			updates["one_id_account_id"] = oneID
		}
	}
	if config.CVMSecretId == "" {
		if secretId := os.Getenv("SECRET_ID"); secretId != "" {
			updates["c_vm_secret_id"] = secretId
		}
	}
	if config.CVMSecretKey == "" {
		if secretKey := os.Getenv("SECRET_KEY"); secretKey != "" {
			updates["c_vm_secret_key"] = secretKey
		}
	}
	if config.AgentCamRoleSecretId == "" {
		if agentId := os.Getenv("AGENT_CAM_ROLE_SECRET_ID"); agentId != "" {
			updates["agent_cam_role_secret_id"] = agentId
		}
	}
	if config.AgentCamRoleSecretKey == "" {
		if agentKey := os.Getenv("AGENT_CAM_ROLE_SECRET_KEY"); agentKey != "" {
			updates["agent_cam_role_secret_key"] = agentKey
		}
	}

	// 回填统一账号模式字段
	if config.OneIDAppID == "" {
		if appID := os.Getenv("ONEID_APP_ID"); appID != "" {
			updates["one_id_app_id"] = appID
		}
	}
	if config.OneIDClientID == "" {
		if clientID := os.Getenv("ONEID_CLIENT_ID"); clientID != "" {
			updates["one_id_client_id"] = clientID
		}
	}
	if config.OneIDClientSecret == "" {
		if clientSecret := os.Getenv("ONEID_CLIENT_SECRET"); clientSecret != "" {
			updates["one_id_client_secret"] = clientSecret
		}
	}
	if config.OneIDTokenEndpoint == "" {
		if tokenEndpoint := os.Getenv("ONEID_TOKEN_ENDPOINT"); tokenEndpoint != "" {
			updates["one_id_token_endpoint"] = tokenEndpoint
		}
	}

	if len(updates) > 0 {
		if err := model.DBGlobal(initCtx).Model(&config).Where("identifier = ?", cmdlineConfig.Identifier).Updates(updates).Error; err != nil {
			slog.Error("[InitDB] 回填 site_configs 租户字段失败", "error", err)
		} else {
			slog.Info("[InitDB] 回填 site_configs 租户字段", "fields", len(updates))
		}
	}
}

// buildFixedSnapshot 从启动参数和数据库构造 FixedSnapshot。
//
// 构造规则：
//   - identifier/uin/domain/internalSecret/oneIDAccountID 直接使用启动参数
//   - AKSK（CVMSecretId/CVMSecretKey/AgentCamRoleSecretId/AgentCamRoleSecretKey）直接读数据库
//     因为 backfillSiteConfig 已经确保了 DB 中有值（如果环境变量提供了的话）
//
// 必须在 backfillSiteConfig 之后调用。
func buildFixedSnapshot(cmdlineConfig CmdlineConfig) {
	// 构造启动期 ctx：始终注入 TenantSnapshot（SQLite 模式下 Identifier 为空但仍注入）。
	initCtx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: cmdlineConfig.Identifier})

	// 从 DB 读取 AKSK 和 OneIDAccountID（这些字段已经被 backfillSiteConfig 回填过了）
	var config model.SiteConfig
	model.DBGlobal(initCtx).Where("identifier = ?", cmdlineConfig.Identifier).First(&config)

	// 解析安全策略
	var sp []string
	if config.SecurityPolicies == "" {
		sp = make([]string, 0)
	} else {
		sp = strings.Split(config.SecurityPolicies, ",")
	}

	snap := &common.TenantSnapshot{
		Identifier:            cmdlineConfig.Identifier,
		Uin:                   cmdlineConfig.Uin,
		Domain:                cmdlineConfig.Domain,
		InternalSecret:        cmdlineConfig.InternalSecret,
		OneIDAccountID:        config.OneIDAccountID,
		CVMSecretId:           config.CVMSecretId,
		CVMSecretKey:          config.CVMSecretKey,
		AgentCamRoleSecretId:  config.AgentCamRoleSecretId,
		AgentCamRoleSecretKey: config.AgentCamRoleSecretKey,
		OneIDAppID:            config.OneIDAppID,
		OneIDClientID:         config.OneIDClientID,
		OneIDClientSecret:     config.OneIDClientSecret,
		OneIDTokenEndpoint:    config.OneIDTokenEndpoint,
		DefaultLang:           config.DefaultLang,
		SecurityPolicies:      sp,
	}

	// 对于启动参数为空但 DB 有值的字段，回退到 DB 值
	if snap.Uin == "" {
		snap.Uin = config.Uin
	}
	if snap.Domain == "" {
		snap.Domain = config.Domain
	}
	if snap.InternalSecret == "" {
		snap.InternalSecret = config.InternalSecret
	}

	common.FixedSnapshot = snap
	slog.Info("[InitDB] FixedSnapshot 构造完成", "identifier", snap.Identifier, "domain", snap.Domain)
}

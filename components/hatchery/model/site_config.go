package model

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"os"
	"slices"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

const DefaultCVMTemplate = `{
  "InstanceChargeType": "PREPAID",
  "InstanceType": "Ai2.MEDIUM4",
  "SystemDisk": {
    "DiskType": "CLOUD_BSSD",
    "DiskSize": 50
  },
  "InstanceChargePrepaid": {
    "Period": 1,
    "RenewFlag": "NOTIFY_AND_AUTO_RENEW"
  },
  "InternetAccessible": {
    "InternetChargeType": "TRAFFIC_POSTPAID_BY_HOUR",
    "InternetMaxBandwidthOut": 5,
    "PublicIpAssigned": true
  }
}`

type SiteConfig struct {
	ID                         uint   `gorm:"primaryKey"`
	Identifier                 string `gorm:"uniqueIndex;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤，每个租户只有一条配置
	Name                       string `gorm:"not null;default:Hatchery"`
	Logo                       []byte
	LogoMIME                   string
	CVMSecretId                string
	CVMSecretKey               string `json:"-"`
	CVMTemplate                string `gorm:"type:text"`
	SecurityGroupId            string // 安全组 ID，创建实例时自动绑定
	AutoCreatedSecurityGroupId string // 自动创建的安全组 ID（补全出站规则时生成，复用避免重复创建）
	SkillHub                   string // SkillHub 地址，创建实例时通过 UserData 注入 init.sh
	SkillHubEnabled            bool   `gorm:"not null;default:false"`                      // 是否启用 SkillHub 技能迁移（灰度开关）
	SkillHubAPIURL             string `gorm:"type:text;default:'https://api.skillhub.cn'"` // SkillHub API 请求地址（迁移代理用，与 skill_hub 分开），MySQL TEXT 列无 DEFAULT，由应用代码兜底
	// 历史字段名保留 Day；实际按日或按月由 GlobalTokenQuotaPeriod 决定，-1=不限。
	GlobalTokenQuotaDay    int    `gorm:"not null;default:-1"`
	GlobalTokenQuotaPeriod string `gorm:"size:16;not null;default:'day'"` // GlobalTokenQuotaDay 的统计周期：day=每日，month=每月
	GlobalTokenQuotaRules  string `gorm:"type:text"`                      // JSON: 全站 Token 配额规则，nil/空=fallback to GlobalTokenQuotaDay/Period
	PublicImageId          string // 公共镜像 ID
	VpcId                  string // 全局 VPC ID
	SubnetIds              string `gorm:"size:1024"`                      // JSON map: zone -> []subnetId（每个可用区支持多个子网）；读取时兼容旧格式 map[zone]subnetId
	CLSEnabled             int    `gorm:"not null;default:0"`             // CLS 服务开通状态: 0=未开通, 1=已开通
	CLSScopeMode           string `gorm:"size:16;not null;default:'all'"` // CLS 采集范围模式: "all"=全量, "group"=分组
	AgentCamRoleSecretId   string // Agent CAM 角色 SecretId
	AgentCamRoleSecretKey  string `json:"-"` // Agent CAM 角色 SecretKey

	// STS 临时密钥字段，不通过任何 API 对外暴露
	STSTmpSecretId  string `json:"-"`
	STSTmpSecretKey string `json:"-"`
	STSToken        string `json:"-" gorm:"size:1024"`
	STSExpiredAt    int64  `json:"-"` // Unix 时间戳

	// 全局终端配置
	TerminalEnabled bool `gorm:"not null;default:false"` // 是否开启终端查看功能，默认不开启

	// 对话界面加载策略配置
	ChatViewEnabled bool `gorm:"not null;default:true"` // 是否允许前端加载对话界面，默认开启

	// Gateway UI 面板配置
	GatewayUIEnable        bool   `gorm:"not null;default:false"`  // 是否开启 Gateway UI 面板功能，默认不开启
	GatewayUIPort          int    `gorm:"not null;default:0"`      // Gateway UI 面板分配的端口，默认为 0
	GatewayUISGMigrateDone bool   `gorm:"not null;default:false"`  // 存量实例安全组迁移是否已执行，仅首次开启时触发
	GatewayUIAddrType      string `gorm:"not null;default:public"` // Gateway UI 访问地址类型："private"（内网）或 "public"（公网），默认 public

	// 云端浏览器（Browser VNC）配置
	BrowserVNCEnable bool `gorm:"not null;default:false"` // 是否开启云端浏览器功能，默认不开启

	// UserData 配置
	UserDataEnabled bool `gorm:"not null;default:false"` // 是否允许用户在创建实例时提交 UserData，默认不开启

	// 平台策略 — 功能权限开关
	UserConfigModelEnabled   bool `gorm:"not null;default:true"` // 允许用户查看与配置模型，默认开启
	UserConfigChannelEnabled bool `gorm:"not null;default:true"` // 允许用户查看与配置通道，默认开启
	ModelQuotaEnabled        bool `gorm:"not null;default:true"` // 允许用户查看模型额度，默认开启

	// Memory TDAI 配置（站点全局）
	MemoryTDAIEnable            bool   `gorm:"not null;default:false"`
	MemoryTDAISupportedVersions string `gorm:"type:text;not null;default:'[]'"`
	MemoryDefaultPlan           string `gorm:"size:32;not null;default:'off'"` // 增量实例默认记忆计划：off/free/pro

	// 技能下发配置
	SkillDistributeConcurrency int `gorm:"not null;default:100"` // 批量下发最大并发数，默认 100

	// SMH 智能媒体托管配置
	SMHEnabled               int    `gorm:"not null;default:0"`     // SMH 服务开通状态: 0=未开通, 1=已开通
	SMHAutoProvisionOnCreate bool   `gorm:"not null;default:false"` // 创建实例时自动创建个人空间
	SMHLibraryId             string // SMH 媒体库 ID
	SMHLibrarySecret         string `json:"-"` // SMH 媒体库密钥（敏感字段，JSON 序列化时自动隐藏）
	SMHEndpoint              string // SMH 媒体库访问域名（AccessDomain）
	SMHProvisionError        string `gorm:"size:64;default:''"` // SMH 开通失败时的错误信息，开通成功后清空

	// SSO 登录方式文案配置
	// 存储格式为 JSON 数组字符串，如 `["wecom","feishu"]`；空字符串表示未配置
	SSOIMType string `gorm:"size:512;default:''"` // 关联企业 IM 标识列表，JSON 数组

	DefaultInstanceQuota   int    `gorm:"not null;default:3"`      // 用户默认实例上限，0 表示无配额
	DefaultTokenQuotaDay   int    `gorm:"not null;default:500000"` // 用户默认每日 Token 配额，-1=不限 (legacy, prefer DefaultTokenQuotaRules)
	DefaultTokenQuotaRules string `gorm:"type:text"`               // JSON: 新用户默认配额规则，nil/空=fallback to DefaultTokenQuotaDay

	// base 安全组达到此实例数时触发异步扩 shard。默认 1800，留 200 buffer 给异步扩容。
	// 读取时用 EffectiveSGPoolAutoScaleThreshold() 兜底，避免 MySQL 存量行未跑 ALTER 或 SQLite AutoMigrate
	// 时机问题读到 0。
	SGPoolAutoScaleThreshold int `gorm:"not null;default:1800"`

	DefaultModelID uint `gorm:"not null;default:0"` // 新建实例默认模型 ID，0=无默认

	// 内置技能包是否已初始化过（防止删除后重启重建）
	DefaultBundleSeeded bool `gorm:"not null;default:false" json:"-"`

	// 内置插件包是否已初始化过（防止删除后重启重建）
	DefaultPluginBundleSeeded bool `gorm:"not null;default:false" json:"-"`

	// 预置角色是否已初始化过（防止删除后重启重建）
	DefaultRolesSeeded bool `gorm:"not null;default:false" json:"-"`

	// 自动创建的默认 VPC/子网（与管理员手动配置的 VpcId/SubnetIds 分开存储）
	DefaultVpcId     string // 自动创建的默认 VPC ID (clawpro/default-vpc-xxxxxx)
	DefaultSubnetIds string `gorm:"size:1024"` // 默认 VPC 的子网映射 JSON map: zone -> []subnetId（每个可用区支持多个子网）

	// 用户端首选智能体类型，默认 openclaw
	DefaultAgentType string `gorm:"size:32;not null;default:'openclaw'"`

	// 用户端禁用的智能体类型列表（JSON 数组），适用于内置和自定义 Agent Type；不影响管理员侧当前镜像选择
	DisabledAgentTypes string `gorm:"type:text"`

	// 技能安全检测：上传/更新技能时"提交安全检测"勾选框的默认值
	SkillScanDefaultEnabled bool `gorm:"not null;default:false"`

	// 默认标签配置，创建实例时自动绑定
	// 格式：JSON 数组 [{"Key":"env","Value":"prod"},{"Key":"managed-by","Value":"openclaw"}]
	DefaultTags string `gorm:"type:varchar(4096);default:'[]'"` // 默认标签 JSON

	// 龙虾医生配置
	DoctorEnabled bool `gorm:"not null;default:false"` // 是否允许用户使用龙虾医生,默认关闭

	// 本地 Agent 功能全局预设（与 feature_allowlist 接作为双层守卫：
	// reporter 接口需同时满足 feature_allowlist 命中 AND LocalAgentEnabled=true 才放行）
	LocalAgentEnabled bool `gorm:"not null;default:false"` // 是否允许本租户用户接入本地 Agent，默认关闭

	// API 网关接入配置（白名单客户 WebUI 域名化访问）
	// JSON 结构：{"enable": bool, "gateway_instance_id": "ins-xxx", "base_domain": "mcd.example.com", "scheme": "http|https"}
	// 默认 "{}"（视为关闭），解析失败同样按关闭处理。详见 openspec/changes/webui-apigateway。
	APIGatewayConfig string `gorm:"type:varchar(1024);default:'{}'"`

	// —— 状态缓存同步标记 ——
	LastFullSyncFinishedAt *time.Time `gorm:"column:last_full_sync_finished_at;default:null"` // 后台 cvm-status-reconcile 整轮成功完成时间

	// —— 多租户阶段一：租户级字段 ——
	// 详见 openspec/changes/multi-tenant-universe-mode。
	// 这些字段原先以进程级全局变量/启动参数形式存在，现统一持久化到 site_configs，
	// 并由启动参数在 InitDB 时按保守策略回填（不覆盖已有值）。
	Uin              string `gorm:"type:varchar(64);not null;default:''"`           // 租户腾讯云 UIN
	Domain           string `gorm:"type:varchar(512);not null;default:''"`          // 租户对外访问域名
	InternalSecret   string `gorm:"type:varchar(512);not null;default:''" json:"-"` // Gateway 内部鉴权密钥
	DefaultLang      string `gorm:"type:varchar(8);not null;default:''"`            // 租户默认语言：zh 或 en；universe 模式下实现租户级语言隔离
	SecurityPolicies string `gorm:"type:varchar(128);not null;default:'SSRF'"`

	// —— OneID 相关 ——
	OneIDAccountID     string `gorm:"column:one_id_account_id;type:varchar(128);not null;default:''"`             // OneID 租户账号 ID（account_id）
	OneIDAppID         string `gorm:"column:one_id_app_id;type:varchar(128);not null;default:''"`                 // OneID 自建应用 ID（非空=统一账号模式）
	OneIDClientID      string `gorm:"column:one_id_client_id;type:varchar(128);not null;default:''"`              // 自建应用 client_id
	OneIDClientSecret  string `gorm:"column:one_id_client_secret;type:varchar(256);not null;default:''" json:"-"` // 自建应用 client_secret
	OneIDTokenEndpoint string `gorm:"column:one_id_token_endpoint;type:varchar(512);not null;default:''"`         // Token 获取端点
	OneIDDomain        string `gorm:"column:one_id_domain;type:varchar(512);not null;default:''"`                 // OneID 企业域名
}

// ApplySiteConfigDefaults 为 SiteConfig 填充业务默认值（Name、CVMTemplate、PublicImageId 等）。
// InitDB（单租户）和 /tenants/init（多租户）共用，确保新创建的租户配置包含完整默认值。
// 仅填充零值字段，不覆盖已设置的值。
// 同时读取 MEMORY_TDAI_SUPPORTED_VERSIONS 环境变量，存在则覆盖版本白名单。
func ApplySiteConfigDefaults(c *SiteConfig) {
	if c.Name == "" {
		c.Name = "ClawPro"
	}
	if c.CVMTemplate == "" {
		c.CVMTemplate = DefaultCVMTemplate
	}
	if c.PublicImageId == "" {
		c.PublicImageId = "img-idzg74s9"
	}
	if c.SkillHubAPIURL == "" {
		c.SkillHubAPIURL = "https://api.skillhub.cn"
	}
	// 环境变量优先；未设置时填充内置默认值
	if envVersions := os.Getenv("MEMORY_TDAI_SUPPORTED_VERSIONS"); envVersions != "" {
		if normalized, _, err := NormalizeMemoryTDAISupportedVersions(envVersions); err == nil {
			c.MemoryTDAISupportedVersions = normalized
		} else {
			slog.Warn("MEMORY_TDAI_SUPPORTED_VERSIONS 环境变量格式错误，使用默认值", "error", err)
			if c.MemoryTDAISupportedVersions == "" {
				c.MemoryTDAISupportedVersions = DefaultMemoryTDAISupportedVersions
			}
		}
	} else if c.MemoryTDAISupportedVersions == "" {
		c.MemoryTDAISupportedVersions = DefaultMemoryTDAISupportedVersions
	}
}

// extractResourceConfigFromTemplate 从旧版 cvm_template 提取默认资源策略字段。
// 这里只迁移资源策略明确支持的字段，其他 CVM 请求参数一律忽略。
func extractResourceConfigFromTemplate(templateJSON string) string {
	if templateJSON == "" {
		return "{}"
	}

	var template struct {
		InstanceChargeType    *string `json:"InstanceChargeType"`
		InstanceChargePrepaid *struct {
			Period    *int    `json:"Period"`
			RenewFlag *string `json:"RenewFlag"`
		} `json:"InstanceChargePrepaid"`
		InstanceType *string `json:"InstanceType"`
		SystemDisk   *struct {
			DiskType *string `json:"DiskType"`
			DiskSize *int    `json:"DiskSize"`
		} `json:"SystemDisk"`
		InternetAccessible *struct {
			PublicIPAssigned        *bool   `json:"PublicIpAssigned"`
			InternetChargeType      *string `json:"InternetChargeType"`
			InternetMaxBandwidthOut *int    `json:"InternetMaxBandwidthOut"`
		} `json:"InternetAccessible"`
	}
	if err := json.Unmarshal([]byte(templateJSON), &template); err != nil {
		return "{}"
	}

	config := make(map[string]any, 5)
	if template.InstanceChargeType != nil {
		config["instance_charge_type"] = *template.InstanceChargeType
	}
	if template.InstanceChargePrepaid != nil {
		prepaid := make(map[string]any, 2)
		if template.InstanceChargePrepaid.Period != nil {
			prepaid["period"] = *template.InstanceChargePrepaid.Period
		}
		if template.InstanceChargePrepaid.RenewFlag != nil {
			prepaid["renew_flag"] = *template.InstanceChargePrepaid.RenewFlag
		}
		config["instance_charge_prepaid"] = prepaid
	}
	if template.InstanceType != nil {
		config["instance_type"] = *template.InstanceType
	}
	if template.SystemDisk != nil {
		disk := make(map[string]any, 2)
		if template.SystemDisk.DiskType != nil {
			disk["disk_type"] = *template.SystemDisk.DiskType
		}
		if template.SystemDisk.DiskSize != nil {
			disk["disk_size"] = *template.SystemDisk.DiskSize
		}
		config["system_disk"] = disk
	}
	if template.InternetAccessible != nil {
		internet := make(map[string]any, 3)
		if template.InternetAccessible.PublicIPAssigned != nil {
			internet["public_ip_assigned"] = *template.InternetAccessible.PublicIPAssigned
		}
		if template.InternetAccessible.InternetChargeType != nil {
			internet["internet_charge_type"] = *template.InternetAccessible.InternetChargeType
		}
		if template.InternetAccessible.InternetMaxBandwidthOut != nil {
			internet["internet_max_bandwidth_out"] = *template.InternetAccessible.InternetMaxBandwidthOut
		}
		config["internet_accessible"] = internet
	}

	encoded, err := json.Marshal(config)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

// SnapFromConfig 将 SiteConfig 转换为 TenantSnapshot。
// 统一映射逻辑，避免多处重复构造。
func SnapFromConfig(c SiteConfig) hcommon.TenantSnapshot {
	// 解析安全策略
	var sp []string
	if c.SecurityPolicies == "" {
		sp = make([]string, 0)
	} else {
		sp = strings.Split(c.SecurityPolicies, ",")
	}
	return hcommon.TenantSnapshot{
		Identifier:            c.Identifier,
		Uin:                   c.Uin,
		Domain:                c.Domain,
		InternalSecret:        c.InternalSecret,
		OneIDAccountID:        c.OneIDAccountID,
		OneIDAppID:            c.OneIDAppID,
		OneIDClientID:         c.OneIDClientID,
		OneIDClientSecret:     c.OneIDClientSecret,
		OneIDTokenEndpoint:    c.OneIDTokenEndpoint,
		OneIDDomain:           c.OneIDDomain,
		CVMSecretId:           c.CVMSecretId,
		CVMSecretKey:          c.CVMSecretKey,
		AgentCamRoleSecretId:  c.AgentCamRoleSecretId,
		AgentCamRoleSecretKey: c.AgentCamRoleSecretKey,
		DefaultLang:           c.DefaultLang,
		SecurityPolicies:      sp,
	}
}

// SyncMemoryTDAISupportedVersions 同步环境变量 MEMORY_TDAI_SUPPORTED_VERSIONS 到已有站点配置。
// 每次启动时由 startup task 调用（per-tenant），确保存量租户配置跟随部署环境更新。
// 逻辑：
//   - 字段为空 → 用环境变量值或内置默认值填充
//   - 环境变量有值且与当前不同 → 覆盖更新
//   - 其他情况不做修改
func SyncMemoryTDAISupportedVersions(ctx context.Context) {
	config := GetSiteConfig(ctx)

	envVersions := os.Getenv("MEMORY_TDAI_SUPPORTED_VERSIONS")
	var target string
	if envVersions != "" {
		normalized, _, err := NormalizeMemoryTDAISupportedVersions(envVersions)
		if err != nil {
			slog.Warn("MEMORY_TDAI_SUPPORTED_VERSIONS 环境变量格式错误，跳过同步", "error", err)
			// 格式错误时仅兜底填空值
			if config.MemoryTDAISupportedVersions == "" {
				target = DefaultMemoryTDAISupportedVersions
			} else {
				return
			}
		} else {
			target = normalized
		}
	} else {
		// 环境变量未设置：仅当字段为空时兜底
		if config.MemoryTDAISupportedVersions == "" {
			target = DefaultMemoryTDAISupportedVersions
		} else {
			return
		}
	}

	if config.MemoryTDAISupportedVersions == target {
		return
	}

	slog.Info("同步 MemoryTDAISupportedVersions",
		"old", config.MemoryTDAISupportedVersions, "new", target)
	UpdateSiteConfig(ctx, map[string]interface{}{
		"memory_tdai_supported_versions": target,
	})
}

// ListAllTenants 返回所有租户快照。
//   - 非 universe 模式（FixedSnapshot != nil）：返回单条记录
//   - Universe 模式（FixedSnapshot == nil）：查 site_configs 表全量
func ListAllTenants() ([]hcommon.TenantSnapshot, error) {
	if hcommon.FixedSnapshot != nil {
		return []hcommon.TenantSnapshot{*hcommon.FixedSnapshot}, nil
	}
	var configs []SiteConfig
	if err := DBGlobal(context.Background()).Where("identifier != ''").Find(&configs).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSiteConfigListTenants)
	}
	snaps := make([]hcommon.TenantSnapshot, 0, len(configs))
	for _, c := range configs {
		snaps = append(snaps, SnapFromConfig(c))
	}
	return snaps, nil
}

// APIGatewayConfig 描述 WebUI 接入云 API 网关的站点级配置。
// 主流程不得因本结构体解析失败而中断——上层一律视为"未启用"并降级。
type APIGatewayConfig struct {
	Enable            bool   `json:"enable"`
	GatewayInstanceID string `json:"gateway_instance_id"`
	BaseDomain        string `json:"base_domain"`
	// Scheme 控制拼接 gatewayUI 时的协议：仅接受 "http" / "https"。
	// 空值/非法值一律回退到默认 "http"（与最初内网联调的默认行为保持一致）。
	Scheme string `json:"scheme,omitempty"`
}

// SchemeOrDefault 返回生效的协议，非法值一律回落到 "http"。
func (g APIGatewayConfig) SchemeOrDefault() string {
	switch g.Scheme {
	case "http", "https":
		return g.Scheme
	default:
		return "http"
	}
}

// GetAPIGatewayConfig 反序列化 SiteConfig.APIGatewayConfig 字段。
// 返回 (cfg, true) 表示 JSON 合法；(zero, false) 表示为空或 JSON 非法，
// 调用方应视作"未启用"继续走主流程，不得报错。
func (c SiteConfig) GetAPIGatewayConfig() (APIGatewayConfig, bool) {
	raw := strings.TrimSpace(c.APIGatewayConfig)
	if raw == "" || raw == "{}" {
		return APIGatewayConfig{}, true
	}
	var cfg APIGatewayConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		slog.Warn("failed to parse SiteConfig APIGatewayConfig", "err", err)
		return APIGatewayConfig{}, false
	}
	return cfg, true
}

// ShouldActivate 判断是否应尝试走 API 网关模式：开关开启、关键字段齐全、用户是 OneID 用户。
// 任一条件不满足一律返回 false，上层即走主流程，不触发任何云 API 调用。
func (g APIGatewayConfig) ShouldActivate(oneIDSub string) bool {
	if !g.Enable {
		return false
	}
	if g.GatewayInstanceID == "" || g.BaseDomain == "" {
		return false
	}
	if oneIDSub == "" {
		return false
	}
	return true
}

// SSOIMTypeOptions 是所有支持的 IM / 认证类型，含显示名称和 logo 地址。
var SSOIMTypeOptions = []map[string]string{
	{"value": "wecom", "label": "企业微信", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/wework-v2-logo.png"},
	{"value": "feishu", "label": "飞书", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/lark-v2-logo.png"},
	{"value": "dingtalk", "label": "钉钉", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/dd-v2-logo.png"},
	{"value": "aad", "label": "微软 Entra ID", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/aad-v2-logo.png"},
	{"value": "saml", "label": "SAML 2.0", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/saml-v2-logo.png"},
	{"value": "ad", "label": "Windows AD", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/ad-v2-logo.png"},
	{"value": "wework_private", "label": "私有化企微", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/wework-logo.png"},
	{"value": "oidc", "label": "OIDC", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/oidc-v2-logo.png"},
	{"value": "jwt", "label": "JWT", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/jwt-v2-logo.png"},
	{"value": "openldap", "label": "OpenLDAP", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/openldap-v2-logo.png"},
	{"value": "cas", "label": "CAS", "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/cas-v2-logo.png"},
	{"value": "oauth2", "label": "OAuth 2.0", "logo": ""},
}

// GetSSOIMTypes 将 SSOIMType 字段（JSON 数组字符串）解析为字符串切片。
// 兼容旧的单值格式（如 "wecom"）。
func (c SiteConfig) GetSSOIMTypes() []string {
	if c.SSOIMType == "" {
		return []string{}
	}
	// 新格式：JSON 数组
	var types []string
	if err := json.Unmarshal([]byte(c.SSOIMType), &types); err == nil {
		return types
	}
	// 旧格式兼容：单值字符串
	return []string{c.SSOIMType}
}

// SetSSOIMTypes 将字符串切片序列化为 JSON 数组字符串，存入 SSOIMType 字段。
func (c *SiteConfig) SetSSOIMTypes(types []string) {
	if len(types) == 0 {
		c.SSOIMType = ""
		return
	}
	b, _ := json.Marshal(types)
	c.SSOIMType = string(b)
}

const (
	GlobalTokenQuotaPeriodDay   = "day"
	GlobalTokenQuotaPeriodMonth = "month"
)

// MemoryDefaultPlan 增量实例默认记忆计划（site_config 字段值，小写）
const (
	MemoryDefaultPlanOff  = "off"
	MemoryDefaultPlanFree = "free"
	MemoryDefaultPlanPro  = "pro"
)

func IsValidGlobalTokenQuotaPeriod(period string) bool {
	period = strings.ToLower(strings.TrimSpace(period))
	return period == GlobalTokenQuotaPeriodDay || period == GlobalTokenQuotaPeriodMonth
}

func NormalizeGlobalTokenQuotaPeriod(period string) string {
	period = strings.ToLower(strings.TrimSpace(period))
	if IsValidGlobalTokenQuotaPeriod(period) {
		return period
	}
	return GlobalTokenQuotaPeriodDay
}

func (c SiteConfig) NormalizedGlobalTokenQuotaPeriod() string {
	return NormalizeGlobalTokenQuotaPeriod(c.GlobalTokenQuotaPeriod)
}

// GetSiteConfig reads site config from database with context for multi-tenant support
func GetSiteConfig(ctx context.Context) SiteConfig {
	var config SiteConfig
	if err := DB(ctx).First(&config).Error; err != nil {
		// 数据库中尚无站点配置记录时，返回带合理默认值的配置
		// 注意：GORM 的 gorm:"default:xxx" tag 只在建表时生效，不影响 Go 结构体零值
		return SiteConfig{
			GlobalTokenQuotaPeriod:   GlobalTokenQuotaPeriodDay,
			DefaultInstanceQuota:     3,
			DefaultTokenQuotaDay:     500000,
			ChatViewEnabled:          true,
			SGPoolAutoScaleThreshold: DefaultSGPoolAutoScaleThreshold,
			UserConfigModelEnabled:   true,
			UserConfigChannelEnabled: true,
			ModelQuotaEnabled:        true,
			// 新增内置类型默认禁用：DeepSeekTUI / OpenCode 仅 Web 终端可用，
			// 不适配模型/通道/技能/插件等任意流程，需管理员通过
			// /admin/agent-types/enabled 显式启用后才允许员工端创建实例。
			// SQL 0601-default-disable-deepseektui-opencode.sql 负责升级已有租户行；
			// 这里覆盖 "DB 尚未落库" 的场景（新建租户/SQLite 单租户首次启动）。
			DisabledAgentTypes: defaultDisabledAgentTypesJSON,
		}
	}
	config.GlobalTokenQuotaPeriod = config.NormalizedGlobalTokenQuotaPeriod()
	return config
}

// EffectiveSGPoolAutoScaleThreshold 返回当前生效的 shard 扩容阈值。
// 存量行未跑迁移 / 读到零值时兜底成 DefaultSGPoolAutoScaleThreshold（1800）。
func (c SiteConfig) EffectiveSGPoolAutoScaleThreshold() int {
	if c.SGPoolAutoScaleThreshold <= 0 {
		return DefaultSGPoolAutoScaleThreshold
	}
	return c.SGPoolAutoScaleThreshold
}

// UpdateSiteConfig updates the site config row (auto-scoped by identifier in multi-tenant mode).
// accepts map[string]interface{} or struct for updates.
func UpdateSiteConfig(ctx context.Context, updates interface{}) error {
	config := GetSiteConfig(ctx)
	return DB(ctx).Model(&config).Updates(updates).Error
}

// GetLastFullSyncFinishedAt 返回后台 cvm-status-reconcile 最近一次整轮成功完成时间。
// 若从未完成过则返回 nil。List 接口据此判断缓存是否就绪（isStatusCacheReady）。
func GetLastFullSyncFinishedAt(ctx context.Context) *time.Time {
	config := GetSiteConfig(ctx)
	return config.LastFullSyncFinishedAt
}

// SetLastFullSyncFinishedAt 更新后台 cvm-status-reconcile 的整轮成功完成时间。
// 后台任务整轮成功后调用，标记缓存就绪。
func SetLastFullSyncFinishedAt(ctx context.Context, t time.Time) error {
	return UpdateSiteConfig(ctx, map[string]any{
		"last_full_sync_finished_at": t,
	})
}

// SMHConfig holds SMH storage configuration resolved from the database.
type SMHConfig struct {
	Endpoint      string // SMH access domain (from site_configs.smh_endpoint)
	LibraryId     string // SMH library ID (from site_configs.smh_library_id)
	LibrarySecret string // SMH library secret (from site_configs.smh_library_secret)
	CommonSpace   string // Common space ID (from smh_spaces table, space_tag="common")
	SkillhubSpace string // Skillhub space ID (from smh_spaces table, space_tag="skillhub")
}

// GetSMHConfig reads SMH configuration from the database.
func GetSMHConfig(ctx context.Context) SMHConfig {
	sc := GetSiteConfig(ctx)
	return SMHConfig{
		Endpoint:      sc.SMHEndpoint,
		LibraryId:     sc.SMHLibraryId,
		LibrarySecret: sc.SMHLibrarySecret,
		CommonSpace:   GetSMHSpace(ctx, "common"),
		SkillhubSpace: GetSMHSpace(ctx, "skillhub"),
	}
}

// IsConfigured 返回 SMH 配置是否完整就绪（核心三项 + 两个 Space 都已创建）。
func (c SMHConfig) IsConfigured() bool {
	return c.Endpoint != "" && c.LibraryId != "" && c.LibrarySecret != "" &&
		c.CommonSpace != "" && c.SkillhubSpace != ""
}

// GenerateGatewayUIPort 随机生成一个 Gateway UI 端口，取值范围 [10000, 40000]。
func GenerateGatewayUIPort() int {
	return 10000 + rand.Intn(30001)
}

// GetSubnetMap parses SubnetIds JSON into a map (zone -> []subnetId).
// 兼容旧格式 map[string]string，自动转换为 map[string][]string（实现同 GetDefaultSubnetMap）。
func (c SiteConfig) GetSubnetMap() map[string][]string {
	if c.SubnetIds == "" || c.SubnetIds == "{}" {
		return make(map[string][]string)
	}

	// 优先尝试新格式: map[string][]string
	newFmt := make(map[string][]string)
	if err := json.Unmarshal([]byte(c.SubnetIds), &newFmt); err == nil {
		// 校验 value 是否为 []string（Unmarshal 可能将旧格式 string 解码为 nil slice）
		valid := true
		for _, v := range newFmt {
			if v == nil {
				valid = false
				break
			}
		}
		if valid {
			return newFmt
		}
	}

	// 兜底：兼容旧格式 map[string]string -> 转为 map[string][]string
	oldFmt := make(map[string]string)
	if err := json.Unmarshal([]byte(c.SubnetIds), &oldFmt); err != nil {
		slog.Warn("failed to parse SiteConfig SubnetIds", "err", err)
		return make(map[string][]string)
	}
	result := make(map[string][]string, len(oldFmt))
	for zone, sid := range oldFmt {
		result[zone] = []string{sid}
	}
	return result
}

// SetSubnetMap serializes a map (zone -> []subnetId) into SubnetIds JSON.
// 序列化失败时不修改 c.SubnetIds，保留原值，便于调用方在错误路径重试或回滚。
func (c *SiteConfig) SetSubnetMap(m map[string][]string) error {
	if len(m) == 0 {
		c.SubnetIds = "{}"
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSiteConfigMarshalSubnet)
	}
	c.SubnetIds = string(data)
	return nil
}

// GetDefaultSubnetMap parses DefaultSubnetIds JSON into a map (zone -> []subnetId).
// 兼容旧格式 map[string]string，自动转换为 map[string][]string。
func (c SiteConfig) GetDefaultSubnetMap() map[string][]string {
	if c.DefaultSubnetIds == "" || c.DefaultSubnetIds == "{}" {
		return make(map[string][]string)
	}

	// 优先尝试新格式: map[string][]string
	newFmt := make(map[string][]string)
	if err := json.Unmarshal([]byte(c.DefaultSubnetIds), &newFmt); err == nil {
		// 校验第一个值是否为 []string（Unmarshal 可能将旧格式 string 解码为 nil slice）
		valid := true
		for _, v := range newFmt {
			if v == nil {
				valid = false
				break
			}
		}
		if valid {
			return newFmt
		}
	}

	// 兜底：兼容旧格式 map[string]string -> 转为 map[string][]string
	oldFmt := make(map[string]string)
	if err := json.Unmarshal([]byte(c.DefaultSubnetIds), &oldFmt); err != nil {
		slog.Warn("failed to parse SiteConfig DefaultSubnetIds", "err", err)
		return make(map[string][]string)
	}
	result := make(map[string][]string, len(oldFmt))
	for zone, sid := range oldFmt {
		result[zone] = []string{sid}
	}
	return result
}

// SetDefaultSubnetMap serializes a map (zone -> []subnetId) into DefaultSubnetIds JSON.
// 序列化失败时不修改 c.DefaultSubnetIds，保留原值，便于调用方在错误路径重试或回滚。
func (c *SiteConfig) SetDefaultSubnetMap(m map[string][]string) error {
	if len(m) == 0 {
		c.DefaultSubnetIds = "{}"
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSiteConfigMarshalDefSubnet)
	}
	c.DefaultSubnetIds = string(data)
	return nil
}

// GlobalDailyTokenUsage returns today's total tokens across all users and models.
func GlobalDailyTokenUsage(ctx context.Context) int64 {
	var total int64
	today := LocalToday()
	DB(ctx).Model(&DailyUsageSummary{}).
		Where("date = ?", today).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&total)
	return total
}

func globalMonthlyTokenUsage(ctx context.Context) int64 {
	var total int64
	start, end := LocalCurrentMonthRange()
	DB(ctx).Model(&DailyUsageSummary{}).
		Where("date >= ? AND date < ?", start, end).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&total)
	return total
}

// GlobalTokenUsageByPeriod returns global token usage using the configured global quota period.
func GlobalTokenUsageByPeriod(ctx context.Context, period string) int64 {
	if NormalizeGlobalTokenQuotaPeriod(period) == GlobalTokenQuotaPeriodMonth {
		return globalMonthlyTokenUsage(ctx)
	}
	return GlobalDailyTokenUsage(ctx)
}

// ResolvedGlobalTokenQuotaRules returns the site-wide token quota rules.
// Empty/NULL rules fallback to legacy GlobalTokenQuotaDay + GlobalTokenQuotaPeriod.
func (c *SiteConfig) ResolvedGlobalTokenQuotaRules() []TokenQuotaRule {
	if rules, ok := ParseTokenQuotaRules(c.GlobalTokenQuotaRules); ok {
		return rules
	}
	return GlobalRulesFromLegacyQuota(c.GlobalTokenQuotaDay, c.NormalizedGlobalTokenQuotaPeriod())
}

// ResolvedDefaultTokenQuotaRules returns the default token quota rules for new users.
// If DefaultTokenQuotaRules is set (non-NULL), parse and return it directly — even if empty
// ("[]" = explicitly unlimited, distinct from NULL = not configured).
// Only when NULL/empty does it fallback to legacy DefaultTokenQuotaDay field.
// DefaultTokenQuotaDay=-1 is an explicit unlimited setting, represented as [].
func (c *SiteConfig) ResolvedDefaultTokenQuotaRules() []TokenQuotaRule {
	if rules, ok := ParseTokenQuotaRules(c.DefaultTokenQuotaRules); ok {
		return rules // 包括 "[]"（显式为空 = 无限制）
	}
	// Fallback: convert legacy field to rules
	if c.DefaultTokenQuotaDay >= 0 {
		return []TokenQuotaRule{{Mode: QuotaModeDay, Limit: c.DefaultTokenQuotaDay}}
	}
	return []TokenQuotaRule{}
}

// ==================== 公网配置（InternetAccessible）====================

// InternetAccessible 公网配置结构体，json tag 保持大驼峰以匹配 cvm_template（腾讯云 API 格式）内部字段。
// 仅用于内部解析和校验，不直接序列化到 API 响应。
type InternetAccessible struct {
	InternetChargeType      string `json:"InternetChargeType"`
	InternetMaxBandwidthOut int    `json:"InternetMaxBandwidthOut"`
	PublicIpAssigned        bool   `json:"PublicIpAssigned"`
}

// InternetAccessibleResp API 响应用结构体，json tag 使用 snake_case 以保持与项目自定义接口风格一致。
type InternetAccessibleResp struct {
	InternetChargeType      string `json:"internet_charge_type"`
	InternetMaxBandwidthOut int    `json:"internet_max_bandwidth_out"`
	PublicIpAssigned        bool   `json:"public_ip_assigned"`
}

// ToResp 将内部结构体转换为 API 响应结构体（大驼峰 → snake_case）
func (ia *InternetAccessible) ToResp() *InternetAccessibleResp {
	if ia == nil {
		return nil
	}
	return &InternetAccessibleResp{
		InternetChargeType:      ia.InternetChargeType,
		InternetMaxBandwidthOut: ia.InternetMaxBandwidthOut,
		PublicIpAssigned:        ia.PublicIpAssigned,
	}
}

// CVMTemplateOverview 用于从 cvm_template JSON 中提取关键字段
type CVMTemplateOverview struct {
	InternetAccessible *InternetAccessible `json:"InternetAccessible,omitempty"`
	InstanceChargeType string              `json:"InstanceChargeType,omitempty"`
}

// validChargeTypes 合法的带宽计费模式枚举
var validChargeTypes = map[string]bool{
	"BANDWIDTH_PREPAID":          true,
	"TRAFFIC_POSTPAID_BY_HOUR":   true,
	"BANDWIDTH_POSTPAID_BY_HOUR": true,
	"BANDWIDTH_PACKAGE":          true,
}

// NormalizeInternetAccessible 修正公网配置数据（不分配 IP 时带宽置 0）。
// 应在 ValidateInternetAccessible 之前调用。
func NormalizeInternetAccessible(ia *InternetAccessible) {
	if ia == nil {
		return
	}
	if !ia.PublicIpAssigned {
		ia.InternetChargeType = ""
		ia.InternetMaxBandwidthOut = 0
	}
}

// ValidateInternetAccessible 纯校验公网配置的业务规则，不修改入参。
func ValidateInternetAccessible(ia *InternetAccessible, instanceChargeType string) error {
	if ia == nil {
		return nil
	}
	// 不分配公网 IP 时，无需校验计费模式和带宽
	if !ia.PublicIpAssigned {
		return nil
	}
	if ia.InternetChargeType == "" {
		return hcommon.I18nError(i18n.MsgInternetChargeTypeRequired)
	}
	if !validChargeTypes[ia.InternetChargeType] {
		return hcommon.I18nError(i18n.MsgInternetChargeTypeUnsupported, ia.InternetChargeType)
	}
	if ia.InternetChargeType == "BANDWIDTH_PREPAID" && instanceChargeType != "" && instanceChargeType != "PREPAID" {
		return hcommon.I18nError(i18n.MsgBandwidthPrepaidRequiresPrepaid, instanceChargeType)
	}
	bw := ia.InternetMaxBandwidthOut
	switch ia.InternetChargeType {
	case "BANDWIDTH_PREPAID":
		if bw < 1 || bw > 20 {
			return hcommon.I18nError(i18n.MsgPrepaidBandwidthOutOfRange, bw)
		}
	case "BANDWIDTH_POSTPAID_BY_HOUR", "BANDWIDTH_PACKAGE":
		if bw < 1 || bw > 2000 {
			return hcommon.I18nError(i18n.MsgBandwidthOutOfRange, ia.InternetChargeType, bw)
		}
	case "TRAFFIC_POSTPAID_BY_HOUR":
		if bw < 1 || bw > 200 {
			return hcommon.I18nError(i18n.MsgTrafficBandwidthOutOfRange, bw)
		}
	}
	return nil
}

// ParseCVMTemplateOverview 从 cvm_template JSON 中解析出公网配置概要
func ParseCVMTemplateOverview(cvmTemplate string) (*CVMTemplateOverview, error) {
	if cvmTemplate == "" {
		return nil, nil
	}
	var overview CVMTemplateOverview
	if err := json.Unmarshal([]byte(cvmTemplate), &overview); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgParseCVMTemplateFailed)
	}
	return &overview, nil
}

// AllowedInstanceTypes 允许的实例规格白名单
var AllowedInstanceTypes = []string{
	"Ai2.MEDIUM2",
	"Ai2.MEDIUM4",
	"Ai2.LARGE8",
	"Ai2.2XLARGE16",
}

// AI2InstanceTypeRank returns the position of an AI2 type in AllowedInstanceTypes.
func AI2InstanceTypeRank(instanceType string) (int, bool) {
	for rank, allowedType := range AllowedInstanceTypes {
		if allowedType == instanceType {
			return rank, true
		}
	}
	return 0, false
}

// MinSystemDiskSize 系统盘最小容量（GB），仅用于旧 CVM 模板校验。
const MinSystemDiskSize = 50

// AllowedDiskTypes 允许的系统盘类型白名单
var AllowedDiskTypes = []string{
	"CLOUD_SSD",     // SSD云硬盘
	"CLOUD_PREMIUM", // 高性能云硬盘
	"CLOUD_BSSD",    // 通用型SSD云硬盘
	"CLOUD_HSSD",    // 增强型SSD云硬盘
}

// ValidateDiskType 校验系统盘类型是否在白名单中
func ValidateDiskType(diskType string) error {
	if slices.Contains(AllowedDiskTypes, diskType) {
		return nil
	}
	return hcommon.I18nError(i18n.MsgDiskTypeUnsupported, diskType, strings.Join(AllowedDiskTypes, ", "))
}

// ValidateSystemDisk 校验旧 CVM 模板的系统盘容量。
func ValidateSystemDisk(diskSize int) error {
	if diskSize < MinSystemDiskSize {
		return hcommon.I18nError(i18n.MsgSystemDiskTooSmall, MinSystemDiskSize, diskSize)
	}
	return nil
}

// GetDefaultAgentType 获取用户端首选智能体类型
func GetDefaultAgentType(ctx context.Context) string {
	config := GetSiteConfig(ctx)
	if config.DefaultAgentType == "" {
		return AgentTypeOpenClaw
	}
	if !IsValidAgentType(ctx, config.DefaultAgentType) {
		return AgentTypeOpenClaw
	}
	return config.DefaultAgentType
}

// SetDefaultAgentType 设置用户端首选智能体类型
func SetDefaultAgentType(ctx context.Context, agentType string) error {
	if !IsValidAgentType(ctx, agentType) {
		return hcommon.I18nError(i18n.MsgInvalidAgentType, agentType)
	}
	if err := UpdateSiteConfig(ctx, map[string]interface{}{
		"default_agent_type": agentType,
	}); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgUpdateSiteConfigFailed)
	}
	return nil
}

// defaultDisabledAgentTypesJSON 是 SiteConfig.DisabledAgentTypes 的内置默认值（JSON 数组字符串）。
//
// 语义：当数据库中尚不存在 site_configs 行（如新建租户、SQLite 单租户首次启动）时，
// GetSiteConfig 返回的零值结构体上，DisabledAgentTypes 字段会预填该 JSON，使
// IsAgentTypeEnabled(deepseektui|opencode) 默认返回 false。
//
// 已存在 site_configs 行的旧租户由 SQL 0601-default-disable-deepseektui-opencode.sql
// 一次性 UPDATE 写入；之后管理员通过 /admin/agent-types/enabled 显式启用即可正常使用。
//
// 字符串保持与 SetDisabledAgentTypes 写入格式一致（json.Marshal([]string{...}) 输出无空格）。
const defaultDisabledAgentTypesJSON = `["deepseektui","opencode"]`

// GetDisabledAgentTypes 返回站点级禁用的 Agent Type 列表。
func GetDisabledAgentTypes(ctx context.Context) []string {
	config := GetSiteConfig(ctx)
	types := config.GetDisabledAgentTypes()
	result := make([]string, 0, len(types))
	for _, t := range types {
		if !IsValidAgentType(ctx, t) {
			slog.WarnContext(ctx, "跳过无效的禁用智能体类型", "agent_type", t)
			continue
		}
		result = append(result, t)
	}
	return result
}

// GetDisabledAgentTypes 解析 SiteConfig.DisabledAgentTypes。
func (c SiteConfig) GetDisabledAgentTypes() []string {
	raw := strings.TrimSpace(c.DisabledAgentTypes)
	if raw == "" {
		return []string{}
	}
	var types []string
	if err := json.Unmarshal([]byte(raw), &types); err != nil {
		slog.Warn("failed to parse SiteConfig DisabledAgentTypes", "err", err)
		return []string{}
	}
	result := make([]string, 0, len(types))
	seen := make(map[string]struct{}, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}
	return result
}

// SetDisabledAgentTypes validates and stores the site-level disabled Agent Type list.
func SetDisabledAgentTypes(ctx context.Context, types []string) error {
	normalized := make([]string, 0, len(types))
	seen := make(map[string]struct{}, len(types))
	defaultType := GetDefaultAgentType(ctx)
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !IsValidAgentType(ctx, t) {
			return hcommon.I18nError(i18n.MsgInvalidAgentType, t)
		}
		if t == defaultType {
			return hcommon.I18nError(i18n.MsgAgentTypeIsDefaultCannotDisable)
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		normalized = append(normalized, t)
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSerializeDisabledAgentTypesFailed)
	}
	if err := UpdateSiteConfig(ctx, map[string]interface{}{"disabled_agent_types": string(data)}); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgUpdateSiteConfigFailed)
	}
	return nil
}

// IsAgentTypeEnabled reports whether an Agent Type is enabled for user-facing entry points.
func IsAgentTypeEnabled(ctx context.Context, agentType string) bool {
	agentType = NormalizeAgentType(agentType)
	for _, disabled := range GetDisabledAgentTypes(ctx) {
		if disabled == agentType {
			return false
		}
	}
	return true
}

// SetAgentTypeEnabled updates the site-level disabled Agent Type list.
func SetAgentTypeEnabled(ctx context.Context, agentType string, enabled bool) error {
	agentType = strings.TrimSpace(agentType)
	if agentType == "" {
		return hcommon.I18nError(i18n.MsgAgentTypeCannotBeEmpty)
	}
	if !IsValidAgentType(ctx, agentType) {
		return hcommon.I18nError(i18n.MsgInvalidAgentType, agentType)
	}
	disabled := GetDisabledAgentTypes(ctx)
	if enabled {
		filtered := disabled[:0]
		for _, t := range disabled {
			if t != agentType {
				filtered = append(filtered, t)
			}
		}
		return SetDisabledAgentTypes(ctx, filtered)
	}
	if agentType == GetDefaultAgentType(ctx) {
		return hcommon.I18nError(i18n.MsgAgentTypeIsDefaultCannotDisable)
	}
	for _, t := range disabled {
		if t == agentType {
			return nil
		}
	}
	disabled = append(disabled, agentType)
	return SetDisabledAgentTypes(ctx, disabled)
}

// FilterEnabledAgentTypes removes site-disabled Agent Types from a list while preserving order.
func FilterEnabledAgentTypes(ctx context.Context, types []string) []string {
	disabled := GetDisabledAgentTypes(ctx)
	if len(disabled) == 0 {
		return types
	}
	disabledSet := make(map[string]struct{}, len(disabled))
	for _, t := range disabled {
		disabledSet[t] = struct{}{}
	}
	out := make([]string, 0, len(types))
	for _, t := range types {
		if _, ok := disabledSet[t]; ok {
			continue
		}
		out = append(out, t)
	}
	return out
}

package tdaimemorysdk

import "time"

const (
	DefaultService = "tdai"
	DefaultVersion = "2025-07-17"
	DefaultRegion  = "ap-guangzhou"
	DefaultTimeout = 30 * time.Second
)

// Config 定义 SDK 客户端配置。
type Config struct {
	// 必填
	SecretID  string
	SecretKey string

	// 可选：STS 场景
	Token string

	// 可选，默认 ap-guangzhou
	Region string

	// 可选，默认 tdai.tencentcloudapi.com
	Endpoint string

	// 可选，默认 tdai
	Service string

	// 可选，默认 2025-07-17
	Version string

	// 可选，默认 30s
	Timeout time.Duration

	// 可选，便于链路识别
	RequestClient string
}

// Validate 检查 Config 必填字段。
// 返回 nil 表示合法；否则返回具体原因。
func (c *Config) Validate() error {
	if c == nil {
		return ErrEmptySecretID
	}
	if c.SecretID == "" {
		return ErrEmptySecretID
	}
	if c.SecretKey == "" {
		return ErrEmptySecretKey
	}
	return nil
}

// WithDefaults 返回应用默认值后的配置副本，不修改原对象。
// 用于测试和其它需要"填充默认值但保留原值"的场景。
func (c Config) WithDefaults() Config {
	if c.Region == "" {
		c.Region = DefaultRegion
	}
	if c.Service == "" {
		c.Service = DefaultService
	}
	if c.Version == "" {
		c.Version = DefaultVersion
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

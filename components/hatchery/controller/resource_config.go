package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	sdkcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ──────────────────────────────────────────────
// ResourceConfig 类型定义（snake_case JSON，与前端交互）
// ──────────────────────────────────────────────

// ResourceConfig 资源配置，字段与 cvm_template 中的资源配置子集对应，使用 snake_case。
type ResourceConfig struct {
	InstanceChargeType    string                    `json:"instance_charge_type,omitempty"`
	InstanceChargePrepaid *InstanceChargePrepaid    `json:"instance_charge_prepaid,omitempty"`
	InstanceType          string                    `json:"instance_type,omitempty"`
	SystemDisk            *SystemDiskConfig         `json:"system_disk,omitempty"`
	InternetAccessible    *InternetAccessibleConfig `json:"internet_accessible,omitempty"`
}

// InstanceChargePrepaid 预付费参数。
type InstanceChargePrepaid struct {
	Period    int    `json:"period"`
	RenewFlag string `json:"renew_flag"`
}

// SystemDiskConfig 系统盘配置。
type SystemDiskConfig struct {
	DiskType string `json:"disk_type"`
	DiskSize int    `json:"disk_size"`
}

// InternetAccessibleConfig 公网配置。
type InternetAccessibleConfig struct {
	PublicIpAssigned        *bool  `json:"public_ip_assigned,omitempty"`
	InternetChargeType      string `json:"internet_charge_type,omitempty"`
	InternetMaxBandwidthOut int    `json:"internet_max_bandwidth_out,omitempty"`
}

// ──────────────────────────────────────────────
// 允许值常量
// ──────────────────────────────────────────────

var allowedInstanceChargeTypes = map[string]bool{
	"PREPAID":          true,
	"POSTPAID_BY_HOUR": true,
}

var allowedRenewFlags = map[string]bool{
	"NOTIFY_AND_AUTO_RENEW":           true,
	"NOTIFY_AND_MANUAL_RENEW":         true,
	"DISABLE_NOTIFY_AND_MANUAL_RENEW": true,
}

// ──────────────────────────────────────────────
// Parse / Normalize / Validate
// ──────────────────────────────────────────────
// JSON object helper
// ──────────────────────────────────────────────

// decodeJSONObject 将 raw JSON string 解析为单个 JSON object。
// 拒绝数组、字符串、尾随 JSON、空值和非 object；未知字段保持兼容忽略。
func decodeJSONObject(raw string, dst any) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed[0] != '{' {
		return fmt.Errorf("request body must contain a JSON object")
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	if err := dec.Decode(dst); err != nil {
		return err
	}
	// 检查尾随 JSON：再 Decode 一次应该得到 io.EOF
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request body must contain a single JSON object")
		}
		return err
	}
	return nil
}

// ──────────────────────────────────────────────
// Parse / Normalize / Validate
// ──────────────────────────────────────────────

// ParseResourceConfig 反序列化 raw JSON string → *ResourceConfig。
// raw 为空时返回空配置（非 nil），调用方可安全使用。
// 非空时校验单个 JSON object；未知字段忽略，预期字段继续由 Normalize / Validate 校验。
func ParseResourceConfig(raw string) (*ResourceConfig, error) {
	cfg := &ResourceConfig{}
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	if err := decodeJSONObject(raw, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// NormalizeResourceConfig 规范化：trim 字符串，uppercase disk_type / instance_charge_type。
func NormalizeResourceConfig(cfg *ResourceConfig) {
	if cfg == nil {
		return
	}
	cfg.InstanceChargeType = strings.ToUpper(strings.TrimSpace(cfg.InstanceChargeType))
	cfg.InstanceType = strings.TrimSpace(cfg.InstanceType)
	if cfg.SystemDisk != nil {
		cfg.SystemDisk.DiskType = strings.ToUpper(strings.TrimSpace(cfg.SystemDisk.DiskType))
	}
	if cfg.InstanceChargePrepaid != nil {
		cfg.InstanceChargePrepaid.RenewFlag = strings.TrimSpace(cfg.InstanceChargePrepaid.RenewFlag)
	}
	if cfg.InternetAccessible != nil {
		cfg.InternetAccessible.InternetChargeType = strings.TrimSpace(cfg.InternetAccessible.InternetChargeType)
	}
}

// ValidateResourceConfig 校验资源配置字段。
func ValidateResourceConfig(ctx context.Context, cfg *ResourceConfig) error {
	if cfg == nil {
		return nil
	}

	// instance_charge_type
	if cfg.InstanceChargeType != "" {
		if !allowedInstanceChargeTypes[cfg.InstanceChargeType] {
			return hcommon.I18nError(i18n.MsgInstanceChargeTypeUnsupported)
		}
	}

	// instance_charge_prepaid: required when PREPAID; when provided, every known
	// field must be valid even if the charge type is inherited from a lower layer.
	if cfg.InstanceChargeType == "PREPAID" && cfg.InstanceChargePrepaid == nil {
		return hcommon.I18nError(i18n.MsgBadRequestParamRequired, "instance_charge_prepaid (required for PREPAID)")
	}
	if cfg.InstanceChargePrepaid != nil {
		if cfg.InstanceChargePrepaid.Period <= 0 {
			return hcommon.I18nError(i18n.MsgBadRequestParamRequired, "instance_charge_prepaid.period must be positive")
		}
		if cfg.InstanceChargePrepaid.RenewFlag != "" && !allowedRenewFlags[cfg.InstanceChargePrepaid.RenewFlag] {
			return hcommon.I18nError(i18n.MsgBadRequest, "instance_charge_prepaid.renew_flag: unsupported value")
		}
	}

	// instance_type
	if cfg.InstanceType != "" {
		allowed := false
		for _, t := range model.AllowedInstanceTypes {
			if t == cfg.InstanceType {
				allowed = true
				break
			}
		}
		if !allowed {
			return hcommon.I18nError(i18n.MsgInstanceTypeNotAvailable, cfg.InstanceType, strings.Join(model.AllowedInstanceTypes, ", "))
		}
	}

	// system_disk
	if cfg.SystemDisk != nil {
		if cfg.SystemDisk.DiskType != "" {
			if err := model.ValidateDiskType(cfg.SystemDisk.DiskType); err != nil {
				return err
			}
		}
		if cfg.SystemDisk.DiskSize < 0 {
			return hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "system_disk.disk_size")
		}
	}

	// internet_accessible: validate provided fields without turning an omitted public_ip_assigned into false.
	if cfg.InternetAccessible != nil {
		if cfg.InternetAccessible.InternetMaxBandwidthOut < 0 {
			return hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "internet_accessible.internet_max_bandwidth_out")
		}
		if cfg.InternetAccessible.PublicIpAssigned != nil {
			ia := &model.InternetAccessible{
				PublicIpAssigned:        *cfg.InternetAccessible.PublicIpAssigned,
				InternetChargeType:      cfg.InternetAccessible.InternetChargeType,
				InternetMaxBandwidthOut: cfg.InternetAccessible.InternetMaxBandwidthOut,
			}
			model.NormalizeInternetAccessible(ia)
			cfg.InternetAccessible.InternetMaxBandwidthOut = ia.InternetMaxBandwidthOut
			if err := model.ValidateInternetAccessible(ia, cfg.InstanceChargeType); err != nil {
				return err
			}
		} else if cfg.InternetAccessible.InternetChargeType != "" {
			ia := &model.InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      cfg.InternetAccessible.InternetChargeType,
				InternetMaxBandwidthOut: max(cfg.InternetAccessible.InternetMaxBandwidthOut, 1),
			}
			if err := model.ValidateInternetAccessible(ia, cfg.InstanceChargeType); err != nil {
				return err
			}
		}
	}

	return nil
}

func resourceConfigFromCVMRequest(request *cvm.RunInstancesRequest) *ResourceConfig {
	cfg := &ResourceConfig{}
	if request == nil {
		return cfg
	}
	if request.InstanceChargeType != nil {
		cfg.InstanceChargeType = *request.InstanceChargeType
	}
	if request.InstanceChargePrepaid != nil {
		cfg.InstanceChargePrepaid = &InstanceChargePrepaid{}
		if request.InstanceChargePrepaid.Period != nil {
			cfg.InstanceChargePrepaid.Period = int(*request.InstanceChargePrepaid.Period)
		}
		if request.InstanceChargePrepaid.RenewFlag != nil {
			cfg.InstanceChargePrepaid.RenewFlag = *request.InstanceChargePrepaid.RenewFlag
		}
	}
	if request.InstanceType != nil {
		cfg.InstanceType = *request.InstanceType
	}
	if request.SystemDisk != nil {
		cfg.SystemDisk = &SystemDiskConfig{}
		if request.SystemDisk.DiskType != nil {
			cfg.SystemDisk.DiskType = *request.SystemDisk.DiskType
		}
		if request.SystemDisk.DiskSize != nil {
			cfg.SystemDisk.DiskSize = int(*request.SystemDisk.DiskSize)
		}
	}
	if request.InternetAccessible != nil {
		cfg.InternetAccessible = &InternetAccessibleConfig{
			PublicIpAssigned: request.InternetAccessible.PublicIpAssigned,
		}
		if request.InternetAccessible.InternetChargeType != nil {
			cfg.InternetAccessible.InternetChargeType = *request.InternetAccessible.InternetChargeType
		}
		if request.InternetAccessible.InternetMaxBandwidthOut != nil {
			cfg.InternetAccessible.InternetMaxBandwidthOut = int(*request.InternetAccessible.InternetMaxBandwidthOut)
		}
	}
	return cfg
}

func resourceConfigFromCVMTemplate(template string) (*ResourceConfig, error) {
	if strings.TrimSpace(template) == "" {
		return &ResourceConfig{}, nil
	}
	request := cvm.NewRunInstancesRequest()
	if err := json.Unmarshal([]byte(template), request); err != nil {
		return nil, err
	}
	return resourceConfigFromCVMRequest(request), nil
}

// effectiveDefaultResourceConfig overlays the tenant default policy on the
// legacy CVM template, matching the resource precedence used during creation.
func effectiveDefaultResourceConfig(ctx context.Context, template string) (*ResourceConfig, error) {
	if strings.TrimSpace(template) == "" {
		template = model.DefaultCVMTemplate
	}
	request := cvm.NewRunInstancesRequest()
	if err := json.Unmarshal([]byte(template), request); err != nil {
		return nil, err
	}
	policy, err := model.GetOrCreateDefaultResourcePolicy(ctx)
	if err != nil {
		return nil, err
	}
	policyConfig, err := ParseResourceConfig(policy.ConfigJSON)
	if err != nil {
		return nil, err
	}
	ApplyResourceConfigToRequest(policyConfig, request)
	return resourceConfigFromCVMRequest(request), nil
}

func resourceConfigSection(cfg *ResourceConfig, path string) (any, error) {
	if !allowedTemplateKeys[path] {
		return nil, hcommon.I18nError(i18n.MsgUnsupportedTemplatePath, path)
	}
	switch path {
	case "instance_charge_type":
		if cfg.InstanceChargeType == "" {
			return nil, nil
		}
		return cfg.InstanceChargeType, nil
	case "instance_charge_prepaid":
		return cfg.InstanceChargePrepaid, nil
	case "instance_type":
		if cfg.InstanceType == "" {
			return nil, nil
		}
		return cfg.InstanceType, nil
	case "system_disk":
		return cfg.SystemDisk, nil
	case "internet_accessible":
		return cfg.InternetAccessible, nil
	default:
		return nil, nil
	}
}

// saveLegacyResourceConfig atomically persists the legacy CVM template and
// applies the same resource change to the tenant default policy.
func saveLegacyResourceConfig(
	ctx context.Context,
	siteConfig *model.SiteConfig,
	fields map[string]any,
	replace bool,
) error {
	templateConfig, err := resourceConfigFromCVMTemplate(siteConfig.CVMTemplate)
	if err != nil {
		return err
	}
	defaultConfig, err := resourceConfigFromCVMTemplate(model.DefaultCVMTemplate)
	if err != nil {
		return err
	}
	policy, err := model.GetOrCreateDefaultResourcePolicy(ctx)
	if err != nil {
		return err
	}

	return model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var locked model.ResourcePolicy
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, policy.ID).Error; err != nil {
			return err
		}
		if !locked.IsDefault {
			return model.ErrDefaultResourcePolicy
		}

		target := templateConfig
		if !replace {
			target, err = ParseResourceConfig(locked.ConfigJSON)
			if err != nil {
				return err
			}
			if _, ok := fields["instance_charge_type"]; ok {
				target.InstanceChargeType = templateConfig.InstanceChargeType
			}
			if _, ok := fields["instance_charge_prepaid"]; ok {
				target.InstanceChargePrepaid = templateConfig.InstanceChargePrepaid
			}
			if _, ok := fields["instance_type"]; ok {
				target.InstanceType = templateConfig.InstanceType
			}
			if _, ok := fields["system_disk"]; ok {
				target.SystemDisk = templateConfig.SystemDisk
			}
			if _, ok := fields["internet_accessible"]; ok {
				target.InternetAccessible = templateConfig.InternetAccessible
			}
		}
		if target.InstanceChargeType == cvmChargeTypePrepaid && target.InstanceChargePrepaid == nil {
			target.InstanceChargePrepaid = defaultConfig.InstanceChargePrepaid
		}
		configJSON, err := json.Marshal(target)
		if err != nil {
			return err
		}
		if err := tx.Model(siteConfig).Select("CVMTemplate").Updates(siteConfig).Error; err != nil {
			return err
		}
		return tx.Model(&locked).Update("config_json", string(configJSON)).Error
	})
}

// ──────────────────────────────────────────────
// Apply resource config to CVM request
// ──────────────────────────────────────────────

// ApplyResourceConfigToRequest 将 ResourceConfig 覆盖到 RunInstancesRequest。
// 仅设置非零/非空字段；nil 子结构跳过。
func ApplyResourceConfigToRequest(cfg *ResourceConfig, request *cvm.RunInstancesRequest) {
	if cfg == nil {
		return
	}
	if cfg.InstanceChargeType != "" {
		request.InstanceChargeType = sdkcommon.StringPtr(cfg.InstanceChargeType)
	}
	if cfg.InstanceType != "" {
		request.InstanceType = sdkcommon.StringPtr(cfg.InstanceType)
	}
	if cfg.InstanceChargePrepaid != nil &&
		(cfg.InstanceChargePrepaid.Period > 0 || cfg.InstanceChargePrepaid.RenewFlag != "") {
		if request.InstanceChargePrepaid == nil {
			request.InstanceChargePrepaid = &cvm.InstanceChargePrepaid{}
		}
		if cfg.InstanceChargePrepaid.Period > 0 {
			request.InstanceChargePrepaid.Period = sdkcommon.Int64Ptr(int64(cfg.InstanceChargePrepaid.Period))
		}
		if cfg.InstanceChargePrepaid.RenewFlag != "" {
			request.InstanceChargePrepaid.RenewFlag = sdkcommon.StringPtr(cfg.InstanceChargePrepaid.RenewFlag)
		}
	}
	if cfg.SystemDisk != nil {
		if request.SystemDisk == nil {
			request.SystemDisk = &cvm.SystemDisk{}
		}
		if cfg.SystemDisk.DiskType != "" {
			request.SystemDisk.DiskType = sdkcommon.StringPtr(cfg.SystemDisk.DiskType)
		}
		if cfg.SystemDisk.DiskSize > 0 {
			request.SystemDisk.DiskSize = sdkcommon.Int64Ptr(int64(cfg.SystemDisk.DiskSize))
		}
	}
	if cfg.InternetAccessible != nil {
		if request.InternetAccessible == nil {
			request.InternetAccessible = &cvm.InternetAccessible{}
		}
		if cfg.InternetAccessible.PublicIpAssigned != nil {
			request.InternetAccessible.PublicIpAssigned = sdkcommon.BoolPtr(*cfg.InternetAccessible.PublicIpAssigned)
		}
		if cfg.InternetAccessible.InternetChargeType != "" {
			request.InternetAccessible.InternetChargeType = sdkcommon.StringPtr(cfg.InternetAccessible.InternetChargeType)
		}
		if cfg.InternetAccessible.InternetMaxBandwidthOut > 0 {
			request.InternetAccessible.InternetMaxBandwidthOut = sdkcommon.Int64Ptr(int64(cfg.InternetAccessible.InternetMaxBandwidthOut))
		}
	}
}

// validateAppliedResourceConfig validates discriminator-dependent fields after an
// overlay has been applied to the effective CVM request. System disk validation
// intentionally remains on newly supplied ResourceConfig values: historical
// CVMTemplate disk sizes are preserved and delegated to the image/CVM checks.
func validateAppliedResourceConfig(ctx context.Context, request *cvm.RunInstancesRequest) error {
	if request == nil {
		return nil
	}

	cfg := &ResourceConfig{}
	if request.InstanceChargeType != nil {
		cfg.InstanceChargeType = *request.InstanceChargeType
	}
	if request.InstanceChargePrepaid != nil {
		cfg.InstanceChargePrepaid = &InstanceChargePrepaid{}
		if request.InstanceChargePrepaid.Period != nil {
			cfg.InstanceChargePrepaid.Period = int(*request.InstanceChargePrepaid.Period)
		}
		if request.InstanceChargePrepaid.RenewFlag != nil {
			cfg.InstanceChargePrepaid.RenewFlag = *request.InstanceChargePrepaid.RenewFlag
		}
	}
	if request.InternetAccessible != nil {
		cfg.InternetAccessible = &InternetAccessibleConfig{
			PublicIpAssigned: request.InternetAccessible.PublicIpAssigned,
		}
		if request.InternetAccessible.InternetChargeType != nil {
			cfg.InternetAccessible.InternetChargeType = *request.InternetAccessible.InternetChargeType
		}
		if request.InternetAccessible.InternetMaxBandwidthOut != nil {
			cfg.InternetAccessible.InternetMaxBandwidthOut = int(*request.InternetAccessible.InternetMaxBandwidthOut)
		}
	}

	NormalizeResourceConfig(cfg)
	return ValidateResourceConfig(ctx, cfg)
}

// ──────────────────────────────────────────────
// 镜像系统盘容量校验
// ──────────────────────────────────────────────

// validateImageSystemDiskSize 校验最终系统盘容量是否满足镜像要求。
// imageSize <= 0 → 跳过；disk 或 DiskSize 为 nil → 跳过。
// imageSize > 0 且 selectedSize < imageSize → 返回 MsgImageSystemDiskTooSmall。
// 永不修改 disk 或其字段。
func validateImageSystemDiskSize(imageSize int64, disk *cvm.SystemDisk) error {
	if imageSize <= 0 {
		return nil
	}
	if disk == nil || disk.DiskSize == nil {
		return nil
	}
	selected := *disk.DiskSize
	if selected < imageSize {
		return hcommon.I18nError(i18n.MsgImageSystemDiskTooSmall, selected, imageSize)
	}
	return nil
}

var errResourceOverlayBadRequest = errors.New("invalid resource overlay")

type resourceOverlayInput struct {
	BaseRequest *cvm.RunInstancesRequest
	GroupID     uint
	UserConfig  string
	DiskType    string
	ImageSize   int64
}

func markResourceOverlayBadRequest(err error) error {
	return hcommon.EnsureRichErrorOrPanic(err).WithCause(errResourceOverlayBadRequest)
}

func applyResourceOverlay(ctx context.Context, input resourceOverlayInput) (*cvm.RunInstancesRequest, error) {
	if input.BaseRequest == nil {
		return nil, fmt.Errorf("nil base CVM request")
	}
	request := input.BaseRequest

	policyRaw, _, err := resolveResourceConfigFn(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}
	policyConfig, err := ParseResourceConfig(string(policyRaw))
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgInvalidJSON)
	}
	NormalizeResourceConfig(policyConfig)
	ApplyResourceConfigToRequest(policyConfig, request)
	if err := validateAppliedResourceConfig(ctx, request); err != nil {
		return nil, err
	}

	if input.UserConfig != "" {
		userConfig, err := ParseResourceConfig(input.UserConfig)
		if err != nil {
			return nil, markResourceOverlayBadRequest(hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		}
		NormalizeResourceConfig(userConfig)
		if err := ValidateResourceConfig(ctx, userConfig); err != nil {
			return nil, markResourceOverlayBadRequest(err)
		}
		if input.DiskType != "" && userConfig.SystemDisk != nil && userConfig.SystemDisk.DiskType != "" &&
			!strings.EqualFold(input.DiskType, userConfig.SystemDisk.DiskType) {
			return nil, markResourceOverlayBadRequest(hcommon.I18nError(i18n.MsgWrongDiskType))
		}
		ApplyResourceConfigToRequest(userConfig, request)
		if err := validateAppliedResourceConfig(ctx, request); err != nil {
			return nil, markResourceOverlayBadRequest(err)
		}
	}

	applyCreateInstanceDiskType(request, input.DiskType)
	if request.InternetAccessible != nil &&
		request.InternetAccessible.PublicIpAssigned != nil &&
		!*request.InternetAccessible.PublicIpAssigned {
		request.InternetAccessible.InternetChargeType = nil
		request.InternetAccessible.InternetMaxBandwidthOut = sdkcommon.Int64Ptr(0)
	}
	if err := validateImageSystemDiskSize(input.ImageSize, request.SystemDisk); err != nil {
		return nil, markResourceOverlayBadRequest(err)
	}
	return request, nil
}

// validateCreateResourceConfig 创建实例前验证：确认实例规格在目标 zone 可售卖。
// 优先使用 options 缓存；缓存未命中时通过 GetCVMClient + CallSDKAPITyped 调用云 API（失败则报错）。
var getCVMOptionsClientFn = func(ctx context.Context) (cvmOptionsClient, error) {
	return GetCVMClient(ctx)
}

func validateCreateResourceConfig(ctx context.Context, zone, chargeType, instanceType string) error {
	if instanceType == "" {
		return nil
	}

	// 验证实例规格在目标 zone 是否在售
	cacheKey := resourceOptionsCacheKey(ctx, "instance-types", zone, chargeType)
	if payload, ok := optionsCacheGet(cacheKey); ok {
		var instanceTypes []instanceTypeItem
		if err := json.Unmarshal(payload.Data, &instanceTypes); err == nil {
			for _, item := range instanceTypes {
				if item.InstanceType == instanceType {
					return nil
				}
			}
			return hcommon.I18nError(i18n.MsgInstanceTypeNotAvailable, instanceType, zone)
		}
	}

	// 缓存未命中，直接调用云 API 验证
	client, err := getCVMOptionsClientFn(ctx)
	if err != nil {
		return hcommon.I18nError(i18n.MsgQueryCloudInstanceTypesCVMFailed)
	}
	req := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	filters := []*cvm.Filter{
		{Name: sdkcommon.StringPtr("zone"), Values: sdkcommon.StringPtrs([]string{zone})},
		{Name: sdkcommon.StringPtr("instance-type"), Values: sdkcommon.StringPtrs([]string{instanceType})},
	}
	if chargeType != "" {
		filters = append(filters, &cvm.Filter{
			Name: sdkcommon.StringPtr("instance-charge-type"), Values: sdkcommon.StringPtrs([]string{chargeType}),
		})
	}
	req.Filters = filters

	resp, err := CallSDKAPITyped(ctx, SDKComponentCVM, req, client.DescribeZoneInstanceConfigInfos)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgQueryCloudInstanceTypesFailed)
	}
	if resp.Response != nil {
		for _, info := range resp.Response.InstanceTypeQuotaSet {
			if info.Status != nil && *info.Status == "SELL" &&
				info.InstanceType != nil && *info.InstanceType == instanceType {
				return nil
			}
		}
	}
	return hcommon.I18nError(i18n.MsgInstanceTypeNotAvailable, instanceType, zone)
}

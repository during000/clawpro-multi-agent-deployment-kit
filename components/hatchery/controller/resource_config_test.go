package controller

import (
	"context"
	"strings"
	"testing"

	"hatchery/model"

	sdkcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// ──────────────────────────────────────────────
// ApplyResourceConfigToRequest — internet_accessible
// ──────────────────────────────────────────────

func TestApplyResourceConfig_PreservesPublicIpAssignedWhenOmitted(t *testing.T) {
	// When internet_accessible is provided but public_ip_assigned is omitted,
	// ApplyResourceConfigToRequest must NOT overwrite an existing
	// PublicIpAssigned=true on the request.

	request := &cvm.RunInstancesRequest{
		InternetAccessible: &cvm.InternetAccessible{
			PublicIpAssigned: sdkcommon.BoolPtr(true),
		},
	}

	cfg := &ResourceConfig{
		InternetAccessible: &InternetAccessibleConfig{
			InternetChargeType: "TRAFFIC_POSTPAID_BY_HOUR",
		},
		// PublicIpAssigned intentionally nil (omitted)
	}

	ApplyResourceConfigToRequest(cfg, request)

	if request.InternetAccessible == nil {
		t.Fatal("InternetAccessible was nil after apply")
	}
	if request.InternetAccessible.PublicIpAssigned == nil {
		t.Fatal("PublicIpAssigned was overwritten to nil when omitted from config")
	}
	if !*request.InternetAccessible.PublicIpAssigned {
		t.Fatalf("PublicIpAssigned changed from true to false when omitted from config")
	}
	// Provided fields should still carry through.
	if got := *request.InternetAccessible.InternetChargeType; got != "TRAFFIC_POSTPAID_BY_HOUR" {
		t.Fatalf("InternetChargeType: got %q, want TRAFFIC_POSTPAID_BY_HOUR", got)
	}
}

func TestApplyResourceConfig_OverwritesPublicIpAssignedWithFalse(t *testing.T) {
	// When public_ip_assigned=false is explicitly set, the request must
	// reflect false. This enables upstream cleanup logic (releasing EIPs etc.)
	// to observe that the user chose no public IP.

	request := &cvm.RunInstancesRequest{
		InternetAccessible: &cvm.InternetAccessible{
			PublicIpAssigned: sdkcommon.BoolPtr(true),
		},
	}

	f := false
	cfg := &ResourceConfig{
		InternetAccessible: &InternetAccessibleConfig{
			PublicIpAssigned:   &f,
			InternetChargeType: "TRAFFIC_POSTPAID_BY_HOUR",
		},
	}

	ApplyResourceConfigToRequest(cfg, request)

	if request.InternetAccessible == nil {
		t.Fatal("InternetAccessible was nil after apply")
	}
	if request.InternetAccessible.PublicIpAssigned == nil {
		t.Fatal("PublicIpAssigned was nil after explicit false in config")
	}
	if *request.InternetAccessible.PublicIpAssigned {
		t.Fatalf("PublicIpAssigned was true; expected false from explicit override")
	}
}

// ──────────────────────────────────────────────
// ValidateResourceConfig — instance_type
// ──────────────────────────────────────────────

func TestValidateResourceConfig_RejectsUnlistedInstanceType(t *testing.T) {
	// A user-style override with an instance type outside model.AllowedInstanceTypes
	// must be rejected with an error that names the bad type.

	cfg := &ResourceConfig{
		InstanceType: "S5.MEDIUM4",
	}

	err := ValidateResourceConfig(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for instance type S5.MEDIUM4, got nil")
	}
	if !strings.Contains(err.Error(), "S5.MEDIUM4") {
		t.Fatalf("error should name the rejected instance type; got %q", err.Error())
	}
}

func TestValidateResourceConfig_AcceptsAllowedInstanceType(t *testing.T) {
	// An instance type within model.AllowedInstanceTypes must pass validation.
	if len(model.AllowedInstanceTypes) == 0 {
		t.Skip("AllowedInstanceTypes is empty; skipping positive case")
	}
	cfg := &ResourceConfig{
		InstanceType: model.AllowedInstanceTypes[0],
	}

	err := ValidateResourceConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("allowed instance type %q was rejected: %v", cfg.InstanceType, err)
	}
}

// ──────────────────────────────────────────────────────────────
// decodeJSONObject — compatible JSON object parsing
// ──────────────────────────────────────────────────────────────

func TestDecodeJSONObject_ValidObject(t *testing.T) {
	var m map[string]any
	if err := decodeJSONObject(`{"instance_charge_type":"POSTPAID_BY_HOUR"}`, &m); err != nil {
		t.Fatalf("valid JSON object rejected: %v", err)
	}
	if m["instance_charge_type"] != "POSTPAID_BY_HOUR" {
		t.Fatalf("wrong value: %v", m["instance_charge_type"])
	}
}

func TestDecodeJSONObject_RejectsArray(t *testing.T) {
	var m map[string]any
	err := decodeJSONObject(`[{"a":1}]`, &m)
	if err == nil {
		t.Fatal("expected error for JSON array, got nil")
	}
}

func TestDecodeJSONObject_RejectsString(t *testing.T) {
	var m map[string]any
	err := decodeJSONObject(`"hello"`, &m)
	if err == nil {
		t.Fatal("expected error for JSON string, got nil")
	}
}

func TestDecodeJSONObject_RejectsNull(t *testing.T) {
	var m map[string]any
	err := decodeJSONObject(`null`, &m)
	if err == nil {
		t.Fatal("expected error for JSON null, got nil")
	}
}

func TestDecodeJSONObject_IgnoresUnknownField(t *testing.T) {
	var cfg ResourceConfig
	err := decodeJSONObject(`{"instance_type":"Ai2.MEDIUM2","instance_typo":"ignored"}`, &cfg)
	if err != nil {
		t.Fatalf("unknown field should be ignored: %v", err)
	}
	if cfg.InstanceType != "Ai2.MEDIUM2" {
		t.Fatalf("expected field was not parsed: %q", cfg.InstanceType)
	}
}

func TestDecodeJSONObject_IgnoresNestedUnknownField(t *testing.T) {
	var cfg ResourceConfig
	err := decodeJSONObject(`{"system_disk":{"disk_type":"CLOUD_SSD","unknown_nested":1}}`, &cfg)
	if err != nil {
		t.Fatalf("nested unknown field should be ignored: %v", err)
	}
	if cfg.SystemDisk == nil || cfg.SystemDisk.DiskType != "CLOUD_SSD" {
		t.Fatalf("expected nested field was not parsed: %+v", cfg.SystemDisk)
	}
}

func TestDecodeJSONObject_RejectsTrailingJSON(t *testing.T) {
	var cfg ResourceConfig
	err := decodeJSONObject(`{"instance_type":"Ai2.MEDIUM2"}{}`, &cfg)
	if err == nil {
		t.Fatal("expected error for trailing JSON, got nil")
	}
}

func TestDecodeJSONObject_RejectsTrailingWhitespaceAndJSON(t *testing.T) {
	var cfg ResourceConfig
	err := decodeJSONObject(`{"instance_type":"Ai2.MEDIUM2"}  "extra"`, &cfg)
	if err == nil {
		t.Fatal("expected error for trailing data, got nil")
	}
}

// ──────────────────────────────────────────────────────────────
// ParseResourceConfig — compatible object parsing with empty fallback
// ──────────────────────────────────────────────────────────────

func TestParseResourceConfig_EmptyStringReturnsEmptyConfig(t *testing.T) {
	cfg, err := ParseResourceConfig("")
	if err != nil {
		t.Fatalf("empty string should not error: %v", err)
	}
	if cfg == nil {
		t.Fatal("empty string should return non-nil config")
	}
}

func TestParseResourceConfig_WhitespaceOnlyReturnsEmptyConfig(t *testing.T) {
	cfg, err := ParseResourceConfig("   \t\n  ")
	if err != nil {
		t.Fatalf("whitespace-only should not error: %v", err)
	}
	if cfg == nil {
		t.Fatal("whitespace-only should return non-nil config")
	}
}

func TestParseResourceConfig_CompleteConfig(t *testing.T) {
	raw := `{
		"instance_charge_type":"POSTPAID_BY_HOUR",
		"instance_type":"Ai2.MEDIUM2",
		"system_disk":{"disk_type":"CLOUD_SSD","disk_size":100},
		"internet_accessible":{"public_ip_assigned":true,"internet_charge_type":"TRAFFIC_POSTPAID_BY_HOUR","internet_max_bandwidth_out":10}
	}`
	cfg, err := ParseResourceConfig(raw)
	if err != nil {
		t.Fatalf("complete config parse failed: %v", err)
	}
	if cfg.InstanceChargeType != "POSTPAID_BY_HOUR" {
		t.Fatalf("charge type: got %q", cfg.InstanceChargeType)
	}
	if cfg.InstanceType != "Ai2.MEDIUM2" {
		t.Fatalf("instance type: got %q", cfg.InstanceType)
	}
	if cfg.SystemDisk == nil || cfg.SystemDisk.DiskType != "CLOUD_SSD" || cfg.SystemDisk.DiskSize != 100 {
		t.Fatalf("system disk: got %+v", cfg.SystemDisk)
	}
	if cfg.InternetAccessible == nil || *cfg.InternetAccessible.PublicIpAssigned != true {
		t.Fatalf("internet accessible: got %+v", cfg.InternetAccessible)
	}
}

func TestParseResourceConfig_IgnoresUnknownField(t *testing.T) {
	cfg, err := ParseResourceConfig(`{"instance_type":"Ai2.MEDIUM2","instance_typo":"ignored"}`)
	if err != nil {
		t.Fatalf("unknown field should be ignored: %v", err)
	}
	if cfg.InstanceType != "Ai2.MEDIUM2" {
		t.Fatalf("expected field was not parsed: %q", cfg.InstanceType)
	}
}

func TestParseResourceConfig_RejectsTrailingJSON(t *testing.T) {
	_, err := ParseResourceConfig(`{"instance_type":"Ai2.MEDIUM2"}{}`)
	if err == nil {
		t.Fatal("expected error for trailing JSON, got nil")
	}
}

func TestParseResourceConfig_RejectsArray(t *testing.T) {
	_, err := ParseResourceConfig(`[{"instance_type":"Ai2.MEDIUM2"}]`)
	if err == nil {
		t.Fatal("expected error for array, got nil")
	}
}

func TestParseResourceConfig_RejectsNull(t *testing.T) {
	_, err := ParseResourceConfig(`null`)
	if err == nil {
		t.Fatal("expected error for null, got nil")
	}
}

func TestParseResourceConfig_RejectsTruncatedJSON(t *testing.T) {
	_, err := ParseResourceConfig(`{"instance_type":"Ai2.MEDIUM2"`)
	if err == nil {
		t.Fatal("expected error for truncated JSON, got nil")
	}
}

// ──────────────────────────────────────────────────────────────
// validateImageSystemDiskSize — pure function, no mutation
// ──────────────────────────────────────────────────────────────

func TestValidateImageSystemDiskSize_ImageLargerThanDisk_Fails(t *testing.T) {
	disk := &cvm.SystemDisk{DiskSize: sdkcommon.Int64Ptr(10)}
	err := validateImageSystemDiskSize(30, disk)
	if err == nil {
		t.Fatal("expected error when image > disk")
	}
	const want = "当前资源配置中限制系统盘容量为 10GiB，所选镜像容量要求为 30GiB，请联系管理员调整。"
	if err.Error() != want {
		t.Fatalf("error=%q want=%q", err.Error(), want)
	}
	// Verify disk was NOT mutated.
	if disk.DiskSize == nil || *disk.DiskSize != 10 {
		t.Fatalf("disk was mutated: DiskSize=%v", disk.DiskSize)
	}
}

func TestValidateImageSystemDiskSize_ImageEqualsDisk_Passes(t *testing.T) {
	disk := &cvm.SystemDisk{DiskSize: sdkcommon.Int64Ptr(100)}
	err := validateImageSystemDiskSize(100, disk)
	if err != nil {
		t.Fatalf("equal sizes should pass: %v", err)
	}
	if disk.DiskSize == nil || *disk.DiskSize != 100 {
		t.Fatal("disk pointer was mutated")
	}
}

func TestValidateImageSystemDiskSize_ImageSmallerThanDisk_Passes(t *testing.T) {
	disk := &cvm.SystemDisk{DiskSize: sdkcommon.Int64Ptr(100)}
	err := validateImageSystemDiskSize(50, disk)
	if err != nil {
		t.Fatalf("image smaller than disk should pass: %v", err)
	}
}

func TestValidateImageSystemDiskSize_ImageSizeZero_Skips(t *testing.T) {
	disk := &cvm.SystemDisk{DiskSize: sdkcommon.Int64Ptr(50)}
	err := validateImageSystemDiskSize(0, disk)
	if err != nil {
		t.Fatalf("ImageSize=0 should skip: %v", err)
	}
}

func TestValidateImageSystemDiskSize_NegativeImageSize_Skips(t *testing.T) {
	disk := &cvm.SystemDisk{DiskSize: sdkcommon.Int64Ptr(50)}
	err := validateImageSystemDiskSize(-1, disk)
	if err != nil {
		t.Fatalf("negative ImageSize should skip: %v", err)
	}
}

func TestValidateImageSystemDiskSize_DiskNil_Skips(t *testing.T) {
	err := validateImageSystemDiskSize(100, nil)
	if err != nil {
		t.Fatalf("nil disk should skip: %v", err)
	}
}

func TestValidateImageSystemDiskSize_DiskSizeNil_Skips(t *testing.T) {
	disk := &cvm.SystemDisk{DiskSize: nil}
	err := validateImageSystemDiskSize(100, disk)
	if err != nil {
		t.Fatalf("nil DiskSize should skip: %v", err)
	}
}

func TestValidateImageSystemDiskSize_DoesNotMutateDisk(t *testing.T) {
	// No mutation regardless of outcome.
	disk := &cvm.SystemDisk{DiskSize: sdkcommon.Int64Ptr(30)}
	_ = validateImageSystemDiskSize(100, disk)
	if disk.DiskSize == nil || *disk.DiskSize != 30 {
		t.Fatal("disk was mutated after validation")
	}
}

// ============================================================================
// NormalizeResourceConfig trims and canonicalizes user input.
// ============================================================================

func TestNormalizeResourceConfig_TrimAndUppercase(t *testing.T) {
	tests := []struct {
		name string
		cfg  *ResourceConfig
		want func(*ResourceConfig)
	}{
		{
			name: "charge type trimmed and uppercased",
			cfg:  &ResourceConfig{InstanceChargeType: " prepaid "},
			want: func(c *ResourceConfig) {
				if c.InstanceChargeType != "PREPAID" {
					t.Fatalf("charge type: got %q, want PREPAID", c.InstanceChargeType)
				}
			},
		},
		{
			name: "disk type uppercased",
			cfg:  &ResourceConfig{SystemDisk: &SystemDiskConfig{DiskType: " cloud_ssd "}},
			want: func(c *ResourceConfig) {
				if c.SystemDisk.DiskType != "CLOUD_SSD" {
					t.Fatalf("disk type: got %q, want CLOUD_SSD", c.SystemDisk.DiskType)
				}
			},
		},
		{
			name: "instance type trimmed (not uppercased)",
			cfg:  &ResourceConfig{InstanceType: " Ai2.MEDIUM2 "},
			want: func(c *ResourceConfig) {
				if c.InstanceType != "Ai2.MEDIUM2" {
					t.Fatalf("instance type: got %q, want Ai2.MEDIUM2", c.InstanceType)
				}
			},
		},
		{
			name: "renew flag trimmed (not uppercased)",
			cfg: &ResourceConfig{
				InstanceChargePrepaid: &InstanceChargePrepaid{RenewFlag: " notify_and_auto_renew "},
			},
			want: func(c *ResourceConfig) {
				if c.InstanceChargePrepaid.RenewFlag != "notify_and_auto_renew" {
					t.Fatalf("renew flag: got %q, want notify_and_auto_renew", c.InstanceChargePrepaid.RenewFlag)
				}
			},
		},
		{
			name: "internet charge type trimmed (not uppercased)",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{InternetChargeType: " traffic_postpaid_by_hour "},
			},
			want: func(c *ResourceConfig) {
				if c.InternetAccessible.InternetChargeType != "traffic_postpaid_by_hour" {
					t.Fatalf("internet charge type: got %q, want traffic_postpaid_by_hour", c.InternetAccessible.InternetChargeType)
				}
			},
		},
		{
			name: "multiple fields simultaneously",
			cfg: &ResourceConfig{
				InstanceChargeType: " prepaid ",
				SystemDisk:         &SystemDiskConfig{DiskType: " cloud_premium "},
			},
			want: func(c *ResourceConfig) {
				if c.InstanceChargeType != "PREPAID" {
					t.Fatalf("charge type: got %q", c.InstanceChargeType)
				}
				if c.SystemDisk.DiskType != "CLOUD_PREMIUM" {
					t.Fatalf("disk type: got %q", c.SystemDisk.DiskType)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NormalizeResourceConfig(tt.cfg)
			tt.want(tt.cfg)
		})
	}
}

// ============================================================================
// NormalizeResourceConfig accepts nil and partial nested configs.
// ============================================================================

func TestNormalizeResourceConfig_NilSafety(t *testing.T) {
	// nil config must not panic.
	NormalizeResourceConfig(nil) // no panic = pass

	// config with nil sub-structs must not panic.
	cfg := &ResourceConfig{
		InstanceChargeType:    "prepaid",
		SystemDisk:            nil,
		InstanceChargePrepaid: nil,
		InternetAccessible:    nil,
	}
	NormalizeResourceConfig(cfg)
	if cfg.InstanceChargeType != "PREPAID" {
		t.Fatalf("charge type: got %q, want PREPAID", cfg.InstanceChargeType)
	}
}

// ============================================================================
// ValidateResourceConfig enforces prepaid purchase rules.
// ============================================================================

func TestValidateResourceConfig_PrepaidRules(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ResourceConfig
		wantErr string // substring to match in error
	}{
		{
			name: "accepts PREPAID with period and renew flag",
			cfg: &ResourceConfig{
				InstanceChargeType: "PREPAID",
				InstanceChargePrepaid: &InstanceChargePrepaid{
					Period:    1,
					RenewFlag: "NOTIFY_AND_AUTO_RENEW",
				},
			},
			wantErr: "",
		},
		{
			name: "accepts PREPAID with omitted renew flag",
			cfg: &ResourceConfig{
				InstanceChargeType: "PREPAID",
				InstanceChargePrepaid: &InstanceChargePrepaid{
					Period:    12,
					RenewFlag: "",
				},
			},
			wantErr: "",
		},
		{
			name: "rejects PREPAID without prepaid settings",
			cfg: &ResourceConfig{
				InstanceChargeType:    "PREPAID",
				InstanceChargePrepaid: nil,
			},
			wantErr: "instance_charge_prepaid",
		},
		{
			name: "rejects zero period",
			cfg: &ResourceConfig{
				InstanceChargeType: "PREPAID",
				InstanceChargePrepaid: &InstanceChargePrepaid{
					Period: 0,
				},
			},
			wantErr: "period",
		},
		{
			name: "rejects negative period",
			cfg: &ResourceConfig{
				InstanceChargeType: "PREPAID",
				InstanceChargePrepaid: &InstanceChargePrepaid{
					Period: -1,
				},
			},
			wantErr: "period",
		},
		{
			name: "rejects unknown renew flag",
			cfg: &ResourceConfig{
				InstanceChargeType: "PREPAID",
				InstanceChargePrepaid: &InstanceChargePrepaid{
					Period:    1,
					RenewFlag: "DO_NOT_RENEW",
				},
			},
			wantErr: "请求体格式错误",
		},
		{
			name: "period zero is invalid when charge type is inherited",
			cfg: &ResourceConfig{
				InstanceChargePrepaid: &InstanceChargePrepaid{Period: 0},
			},
			wantErr: "period",
		},
		{
			name: "renew flag is validated when charge type is inherited",
			cfg: &ResourceConfig{
				InstanceChargePrepaid: &InstanceChargePrepaid{
					Period:    1,
					RenewFlag: "DO_NOT_RENEW",
				},
			},
			wantErr: "请求体格式错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Normalize first so the validation sees clean values.
			NormalizeResourceConfig(tt.cfg)
			err := ValidateResourceConfig(context.Background(), tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected pass, got: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error should contain %q; got: %s", tt.wantErr, err.Error())
				}
			}
		})
	}
}

// ============================================================================
// ValidateResourceConfig rejects unsupported charge and instance types.
// ============================================================================

func TestValidateResourceConfig_RejectsInvalidChargeOrInstanceType(t *testing.T) {
	// Extends existing instance type tests with charge type coverage.
	tests := []struct {
		name    string
		cfg     *ResourceConfig
		wantErr string
	}{
		{
			name:    "invalid charge type",
			cfg:     &ResourceConfig{InstanceChargeType: "SPOTPAID"},
			wantErr: "instance_charge_type",
		},
		{
			name:    "invalid instance type",
			cfg:     &ResourceConfig{InstanceType: "X1.UNKNOWN"},
			wantErr: "X1.UNKNOWN",
		},
		{
			name: "empty charge type (allowed for partial config)",
			cfg:  &ResourceConfig{InstanceChargeType: ""},
		},
		{
			name: "empty instance type (allowed for partial config)",
			cfg:  &ResourceConfig{InstanceType: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NormalizeResourceConfig(tt.cfg)
			err := ValidateResourceConfig(context.Background(), tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected pass, got: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error should contain %q; got: %s", tt.wantErr, err.Error())
				}
			}
		})
	}
}

// ValidateResourceConfig validates system disk type and field shape.
func TestValidateResourceConfig_SystemDiskFields(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ResourceConfig
		wantErr string
	}{
		{
			name: "accepts CLOUD_SSD",
			cfg: &ResourceConfig{
				SystemDisk: &SystemDiskConfig{DiskType: "CLOUD_SSD", DiskSize: 50},
			},
		},
		{
			name: "accepts CLOUD_PREMIUM",
			cfg: &ResourceConfig{
				SystemDisk: &SystemDiskConfig{DiskType: "CLOUD_PREMIUM", DiskSize: 100},
			},
		},
		{
			name: "accepts CLOUD_BSSD",
			cfg: &ResourceConfig{
				SystemDisk: &SystemDiskConfig{DiskType: "CLOUD_BSSD", DiskSize: 100},
			},
		},
		{
			name: "accepts CLOUD_HSSD",
			cfg: &ResourceConfig{
				SystemDisk: &SystemDiskConfig{DiskType: "CLOUD_HSSD", DiskSize: 100},
			},
		},
		{
			name: "rejects unsupported disk type",
			cfg: &ResourceConfig{
				SystemDisk: &SystemDiskConfig{DiskType: "CLOUD_UNKNOWN", DiskSize: 100},
			},
			wantErr: "CLOUD_UNKNOWN",
		},
		{
			name: "allows positive size below legacy template minimum",
			cfg: &ResourceConfig{
				SystemDisk: &SystemDiskConfig{DiskType: "CLOUD_SSD", DiskSize: 30},
			},
		},
		{
			name: "allows zero disk size as omitted",
			cfg: &ResourceConfig{
				SystemDisk: &SystemDiskConfig{DiskType: "CLOUD_SSD", DiskSize: 0},
			},
		},
		{
			name: "negative disk size is invalid rather than omitted",
			cfg: &ResourceConfig{
				SystemDisk: &SystemDiskConfig{DiskType: "CLOUD_SSD", DiskSize: -1},
			},
			wantErr: "system_disk.disk_size",
		},
		{
			name: "disk type empty (allowed for partial config)",
			cfg: &ResourceConfig{
				SystemDisk: &SystemDiskConfig{DiskType: "", DiskSize: 100},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NormalizeResourceConfig(tt.cfg)
			err := ValidateResourceConfig(context.Background(), tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected pass, got: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error should contain %q; got: %s", tt.wantErr, err.Error())
				}
			}
		})
	}
}

// ============================================================================
// ValidateResourceConfig rejects invalid public network combinations.
// ============================================================================

func TestValidateResourceConfig_RejectsInvalidInternetAccessibleCombinations(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ResourceConfig
		wantErr string
	}{
		{
			name: "public_ip_assigned=true, no charge type",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:        sdkcommon.BoolPtr(true),
					InternetChargeType:      "",
					InternetMaxBandwidthOut: 10,
				},
			},
			wantErr: "带宽计费模式",
		},
		{
			name: "public_ip_assigned=true, unsupported charge type",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:        sdkcommon.BoolPtr(true),
					InternetChargeType:      "FREE_CHARGE",
					InternetMaxBandwidthOut: 10,
				},
			},
			wantErr: "FREE_CHARGE",
		},
		{
			name: "BANDWIDTH_PREPAID with non-PREPAID instance",
			cfg: &ResourceConfig{
				InstanceChargeType: "POSTPAID_BY_HOUR",
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:        sdkcommon.BoolPtr(true),
					InternetChargeType:      "BANDWIDTH_PREPAID",
					InternetMaxBandwidthOut: 10,
				},
			},
			wantErr: "BANDWIDTH_PREPAID",
		},
		{
			name: "BANDWIDTH_PREPAID bandwidth over 20",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:        sdkcommon.BoolPtr(true),
					InternetChargeType:      "BANDWIDTH_PREPAID",
					InternetMaxBandwidthOut: 21,
				},
			},
			wantErr: "1-20",
		},
		{
			name: "TRAFFIC_POSTPAID_BY_HOUR bandwidth zero",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:        sdkcommon.BoolPtr(true),
					InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
					InternetMaxBandwidthOut: 0,
				},
			},
			wantErr: "TRAFFIC",
		},
		{
			name: "TRAFFIC_POSTPAID_BY_HOUR bandwidth over 200",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:        sdkcommon.BoolPtr(true),
					InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
					InternetMaxBandwidthOut: 201,
				},
			},
			wantErr: "TRAFFIC",
		},
		{
			name: "BANDWIDTH_POSTPAID_BY_HOUR bandwidth zero",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:        sdkcommon.BoolPtr(true),
					InternetChargeType:      "BANDWIDTH_POSTPAID_BY_HOUR",
					InternetMaxBandwidthOut: 0,
				},
			},
			wantErr: "BANDWIDTH",
		},
		{
			name: "BANDWIDTH_POSTPAID_BY_HOUR bandwidth over 2000",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:        sdkcommon.BoolPtr(true),
					InternetChargeType:      "BANDWIDTH_POSTPAID_BY_HOUR",
					InternetMaxBandwidthOut: 2001,
				},
			},
			wantErr: "BANDWIDTH",
		},
		{
			name: "public_ip_assigned=false skips validation",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:        sdkcommon.BoolPtr(false),
					InternetChargeType:      "",
					InternetMaxBandwidthOut: 0,
				},
			},
		},
		{
			name: "negative bandwidth is invalid rather than omitted",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					InternetMaxBandwidthOut: -1,
				},
			},
			wantErr: "internet_accessible.internet_max_bandwidth_out",
		},
		{
			name: "public_ip_assigned omitted (nil), no charge type — no validation triggered",
			cfg: &ResourceConfig{
				InternetAccessible: &InternetAccessibleConfig{
					PublicIpAssigned:   nil,
					InternetChargeType: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NormalizeResourceConfig(tt.cfg)
			err := ValidateResourceConfig(context.Background(), tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected pass, got: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error should contain %q; got: %s", tt.wantErr, err.Error())
				}
			}
		})
	}
}

// ============================================================================
// ApplyResourceConfigToRequest supports full, partial, and nil configs.
// ============================================================================

func TestApplyResourceConfigToRequest_FullConfig(t *testing.T) {
	cfg := &ResourceConfig{
		InstanceChargeType: "PREPAID",
		InstanceChargePrepaid: &InstanceChargePrepaid{
			Period:    12,
			RenewFlag: "NOTIFY_AND_AUTO_RENEW",
		},
		InstanceType: "Ai2.MEDIUM2",
		SystemDisk: &SystemDiskConfig{
			DiskType: "CLOUD_SSD",
			DiskSize: 100,
		},
		InternetAccessible: &InternetAccessibleConfig{
			PublicIpAssigned:        sdkcommon.BoolPtr(true),
			InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
			InternetMaxBandwidthOut: 10,
		},
	}

	request := cvm.NewRunInstancesRequest()
	ApplyResourceConfigToRequest(cfg, request)

	if request.InstanceChargeType == nil || *request.InstanceChargeType != "PREPAID" {
		t.Fatalf("InstanceChargeType: got %v", request.InstanceChargeType)
	}
	if request.InstanceChargePrepaid == nil || *request.InstanceChargePrepaid.Period != 12 {
		t.Fatalf("InstanceChargePrepaid: got %v", request.InstanceChargePrepaid)
	}
	if request.InstanceType == nil || *request.InstanceType != "Ai2.MEDIUM2" {
		t.Fatalf("InstanceType: got %v", request.InstanceType)
	}
	if request.SystemDisk == nil || *request.SystemDisk.DiskType != "CLOUD_SSD" || *request.SystemDisk.DiskSize != 100 {
		t.Fatalf("SystemDisk: got %+v", request.SystemDisk)
	}
	if request.InternetAccessible == nil || *request.InternetAccessible.PublicIpAssigned != true {
		t.Fatalf("InternetAccessible: got %+v", request.InternetAccessible)
	}
	if *request.InternetAccessible.InternetChargeType != "TRAFFIC_POSTPAID_BY_HOUR" {
		t.Fatalf("InternetChargeType: got %q", *request.InternetAccessible.InternetChargeType)
	}
	if *request.InternetAccessible.InternetMaxBandwidthOut != 10 {
		t.Fatalf("InternetMaxBandwidthOut: got %d", *request.InternetAccessible.InternetMaxBandwidthOut)
	}
}

func TestApplyResourceConfigToRequest_PartialConfig(t *testing.T) {
	// Start with a request that has some preset values.
	request := &cvm.RunInstancesRequest{
		InstanceChargeType: sdkcommon.StringPtr("POSTPAID_BY_HOUR"),
		InstanceType:       sdkcommon.StringPtr("Ai2.MEDIUM4"),
		SystemDisk: &cvm.SystemDisk{
			DiskType: sdkcommon.StringPtr("CLOUD_PREMIUM"),
			DiskSize: sdkcommon.Int64Ptr(200),
		},
	}

	// Apply a partial config: only instance_type and system_disk.
	cfg := &ResourceConfig{
		InstanceType: "Ai2.LARGE8",
		SystemDisk: &SystemDiskConfig{
			DiskType: "CLOUD_SSD",
			// DiskSize 0 = omitted → should not overwrite existing DiskSize.
		},
	}
	ApplyResourceConfigToRequest(cfg, request)

	// Unchanged fields must remain.
	if request.InstanceChargeType == nil || *request.InstanceChargeType != "POSTPAID_BY_HOUR" {
		t.Fatalf("InstanceChargeType should be preserved: got %v", request.InstanceChargeType)
	}
	// Overwritten fields must reflect config.
	if request.InstanceType == nil || *request.InstanceType != "Ai2.LARGE8" {
		t.Fatalf("InstanceType should be Ai2.LARGE8: got %v", request.InstanceType)
	}
	if request.SystemDisk == nil || *request.SystemDisk.DiskType != "CLOUD_SSD" {
		t.Fatalf("DiskType should be CLOUD_SSD: got %v", request.SystemDisk)
	}
	// DiskSize 0 in config must not overwrite existing DiskSize 200.
	if request.SystemDisk.DiskSize == nil || *request.SystemDisk.DiskSize != 200 {
		t.Fatalf("DiskSize should be preserved as 200: got %v", request.SystemDisk.DiskSize)
	}
	// InternetAccessible should remain nil (not set by partial config).
	if request.InternetAccessible != nil {
		t.Fatalf("InternetAccessible should be nil: got %+v", request.InternetAccessible)
	}
}

func TestApplyResourceConfigToRequest_PartialPrepaidPreservesRenewFlag(t *testing.T) {
	request := &cvm.RunInstancesRequest{
		InstanceChargePrepaid: &cvm.InstanceChargePrepaid{
			Period:    sdkcommon.Int64Ptr(12),
			RenewFlag: sdkcommon.StringPtr("NOTIFY_AND_AUTO_RENEW"),
		},
	}

	ApplyResourceConfigToRequest(&ResourceConfig{
		InstanceChargePrepaid: &InstanceChargePrepaid{Period: 3},
	}, request)

	if request.InstanceChargePrepaid.Period == nil || *request.InstanceChargePrepaid.Period != 3 {
		t.Fatalf("Period should be overridden to 3: %+v", request.InstanceChargePrepaid)
	}
	if request.InstanceChargePrepaid.RenewFlag == nil ||
		*request.InstanceChargePrepaid.RenewFlag != "NOTIFY_AND_AUTO_RENEW" {
		t.Fatalf("RenewFlag should be preserved: %+v", request.InstanceChargePrepaid)
	}
}

func TestValidateAppliedResourceConfig_UsesInheritedDiscriminators(t *testing.T) {
	request := &cvm.RunInstancesRequest{
		InstanceChargeType: sdkcommon.StringPtr("POSTPAID_BY_HOUR"),
		InternetAccessible: &cvm.InternetAccessible{
			PublicIpAssigned:        sdkcommon.BoolPtr(true),
			InternetChargeType:      sdkcommon.StringPtr("TRAFFIC_POSTPAID_BY_HOUR"),
			InternetMaxBandwidthOut: sdkcommon.Int64Ptr(1000),
		},
	}

	if err := validateAppliedResourceConfig(context.Background(), request); err == nil {
		t.Fatal("bandwidth 1000 must be rejected against inherited traffic charge type")
	}

	request.InternetAccessible.InternetMaxBandwidthOut = sdkcommon.Int64Ptr(200)
	if err := validateAppliedResourceConfig(context.Background(), request); err != nil {
		t.Fatalf("bandwidth 200 should pass against inherited traffic charge type: %v", err)
	}
}

func TestApplyResourceConfigToRequest_NilConfig(t *testing.T) {
	request := &cvm.RunInstancesRequest{
		InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"),
		SystemDisk: &cvm.SystemDisk{
			DiskSize: sdkcommon.Int64Ptr(100),
		},
	}

	// Clone before calling.
	origType := *request.InstanceType
	origDiskSize := *request.SystemDisk.DiskSize

	ApplyResourceConfigToRequest(nil, request)

	if request.InstanceType == nil || *request.InstanceType != origType {
		t.Fatal("nil config must not mutate InstanceType")
	}
	if request.SystemDisk == nil || request.SystemDisk.DiskSize == nil || *request.SystemDisk.DiskSize != origDiskSize {
		t.Fatal("nil config must not mutate SystemDisk")
	}
}

// ──────────────────────────────────────────────────────────────
// resourceOptionsCacheKey — tenant-aware key construction
// ──────────────────────────────────────────────────────────────

func TestResourceOptionsCacheKey_IncludesAllParts(t *testing.T) {
	// Verify the key includes identifier, region, endpoint, and scope.
	// We can't easily inject CurrentIdentifier in a unit test,
	// but we verify the structure includes the endpoint and scope parts.
	key := resourceOptionsCacheKey(context.Background(), "instance-types", "ap-guangzhou", "POSTPAID_BY_HOUR")
	if !strings.Contains(key, "instance-types") {
		t.Fatalf("key missing endpoint: %s", key)
	}
	if !strings.Contains(key, "ap-guangzhou") {
		t.Fatalf("key missing scope: %s", key)
	}
	if !strings.Contains(key, "POSTPAID_BY_HOUR") {
		t.Fatalf("key missing charge type scope: %s", key)
	}
	parts := strings.Split(key, ":")
	if len(parts) < 4 {
		t.Fatalf("key should have at least 4 colon-separated parts, got %d: %s", len(parts), key)
	}
}

func TestResourceOptionsCacheKey_NoScope(t *testing.T) {
	key := resourceOptionsCacheKey(context.Background(), "instance-types")
	if !strings.Contains(key, "instance-types") {
		t.Fatalf("key missing endpoint: %s", key)
	}
	parts := strings.Split(key, ":")
	if len(parts) < 3 {
		t.Fatalf("key should have at least 3 colon-separated parts, got %d: %s", len(parts), key)
	}
}

package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupSiteConfigTestDB creates a temporary SQLite database for site config testing.
func setupSiteConfigTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "site_config_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	origDB := gdb
	gdb = testDB

	if err := gdb.AutoMigrate(&SiteConfig{}); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	return func() {
		sqlDB, _ := gdb.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		os.Remove(tmpFile.Name() + "-wal")
		os.Remove(tmpFile.Name() + "-shm")
		gdb = origDB
	}
}

func TestValidateInternetAccessible(t *testing.T) {
	tests := []struct {
		name               string
		ia                 *InternetAccessible
		instanceChargeType string
		wantErr            bool
		errContains        string
	}{
		{
			name:    "nil config is valid",
			ia:      nil,
			wantErr: false,
		},
		{
			name:    "no public IP - no validation needed",
			ia:      &InternetAccessible{PublicIpAssigned: false},
			wantErr: false,
		},
		{
			name: "public IP but no charge type",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "",
				InternetMaxBandwidthOut: 10,
			},
			wantErr:     true,
			errContains: "必须指定带宽计费模式",
		},
		{
			name: "invalid charge type",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "INVALID_TYPE",
				InternetMaxBandwidthOut: 10,
			},
			wantErr:     true,
			errContains: "不支持的带宽计费模式",
		},
		{
			name: "BANDWIDTH_PREPAID with non-PREPAID instance",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "BANDWIDTH_PREPAID",
				InternetMaxBandwidthOut: 10,
			},
			instanceChargeType: "POSTPAID_BY_HOUR",
			wantErr:            true,
			errContains:        "仅可用于预付费",
		},
		{
			name: "BANDWIDTH_PREPAID upper boundary is valid",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "BANDWIDTH_PREPAID",
				InternetMaxBandwidthOut: 20,
			},
			instanceChargeType: "PREPAID",
			wantErr:            false,
		},
		{
			name: "BANDWIDTH_PREPAID bandwidth over 20",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "BANDWIDTH_PREPAID",
				InternetMaxBandwidthOut: 21,
			},
			instanceChargeType: "PREPAID",
			wantErr:            true,
			errContains:        "带宽上限范围为 1-20",
		},
		{
			name: "TRAFFIC_POSTPAID_BY_HOUR with valid bandwidth",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
				InternetMaxBandwidthOut: 100,
			},
			wantErr: false,
		},
		{
			name: "TRAFFIC_POSTPAID_BY_HOUR bandwidth too high",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
				InternetMaxBandwidthOut: 300,
			},
			wantErr:     true,
			errContains: "带宽上限范围为 1-200",
		},
		{
			name: "TRAFFIC_POSTPAID_BY_HOUR bandwidth too low",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
				InternetMaxBandwidthOut: 0,
			},
			wantErr:     true,
			errContains: "带宽上限范围为 1-200",
		},
		{
			name: "BANDWIDTH_POSTPAID_BY_HOUR with valid bandwidth",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "BANDWIDTH_POSTPAID_BY_HOUR",
				InternetMaxBandwidthOut: 500,
			},
			wantErr: false,
		},
		{
			name: "BANDWIDTH_POSTPAID_BY_HOUR bandwidth too high",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "BANDWIDTH_POSTPAID_BY_HOUR",
				InternetMaxBandwidthOut: 2500,
			},
			wantErr:     true,
			errContains: "带宽上限范围为 1-2000",
		},
		{
			name: "BANDWIDTH_PACKAGE with valid bandwidth",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "BANDWIDTH_PACKAGE",
				InternetMaxBandwidthOut: 1000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInternetAccessible(tt.ia, tt.instanceChargeType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInternetAccessible() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %v", tt.errContains, err)
				}
			}
		})
	}
}

func TestNormalizeInternetAccessible(t *testing.T) {
	tests := []struct {
		name           string
		ia             *InternetAccessible
		wantChargeType string
		wantBandwidth  int
	}{
		{
			name: "nil config unchanged",
			ia:   nil,
		},
		{
			name: "public IP assigned - unchanged",
			ia: &InternetAccessible{
				PublicIpAssigned:        true,
				InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
				InternetMaxBandwidthOut: 10,
			},
			wantChargeType: "TRAFFIC_POSTPAID_BY_HOUR",
			wantBandwidth:  10,
		},
		{
			name: "no public IP - reset fields",
			ia: &InternetAccessible{
				PublicIpAssigned:        false,
				InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
				InternetMaxBandwidthOut: 10,
			},
			wantChargeType: "",
			wantBandwidth:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NormalizeInternetAccessible(tt.ia)
			if tt.ia != nil {
				if tt.ia.InternetChargeType != tt.wantChargeType {
					t.Errorf("ChargeType = %q, want %q", tt.ia.InternetChargeType, tt.wantChargeType)
				}
				if tt.ia.InternetMaxBandwidthOut != tt.wantBandwidth {
					t.Errorf("Bandwidth = %d, want %d", tt.ia.InternetMaxBandwidthOut, tt.wantBandwidth)
				}
			}
		})
	}
}

func TestValidateDiskType(t *testing.T) {
	tests := []struct {
		diskType string
		wantErr  bool
	}{
		{"CLOUD_SSD", false},
		{"CLOUD_PREMIUM", false},
		{"CLOUD_BSSD", false},
		{"CLOUD_HSSD", false},
		{"INVALID_TYPE", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.diskType, func(t *testing.T) {
			err := ValidateDiskType(tt.diskType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDiskType(%q) error = %v, wantErr %v", tt.diskType, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSystemDisk(t *testing.T) {
	tests := []struct {
		diskSize int
		wantErr  bool
	}{
		{50, false},
		{100, false},
		{49, true},
		{0, true},
		{-1, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := ValidateSystemDisk(tt.diskSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSystemDisk(%d) error = %v, wantErr %v", tt.diskSize, err, tt.wantErr)
			}
		})
	}
}

func TestParseCVMTemplateOverview(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		wantNil     bool
		wantErr     bool
		wantChargeT string
	}{
		{
			name:     "empty template",
			template: "",
			wantNil:  true,
		},
		{
			name:     "invalid JSON",
			template: "{invalid}",
			wantErr:  true,
		},
		{
			name:        "valid template with internet accessible",
			template:    `{"InternetAccessible":{"InternetChargeType":"TRAFFIC_POSTPAID_BY_HOUR","InternetMaxBandwidthOut":10,"PublicIpAssigned":true},"InstanceChargeType":"PREPAID"}`,
			wantChargeT: "TRAFFIC_POSTPAID_BY_HOUR",
		},
		{
			name:     "template without internet accessible",
			template: `{"InstanceChargeType":"PREPAID"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overview, err := ParseCVMTemplateOverview(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCVMTemplateOverview() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil && overview != nil {
				t.Errorf("expected nil overview, got %+v", overview)
				return
			}
			if !tt.wantNil && tt.wantChargeT != "" {
				if overview.InternetAccessible == nil {
					t.Error("expected InternetAccessible to be non-nil")
					return
				}
				if overview.InternetAccessible.InternetChargeType != tt.wantChargeT {
					t.Errorf("ChargeType = %q, want %q", overview.InternetAccessible.InternetChargeType, tt.wantChargeT)
				}
			}
		})
	}
}

func TestInternetAccessibleToResp(t *testing.T) {
	// Test nil input
	var nilIA *InternetAccessible
	if nilIA.ToResp() != nil {
		t.Error("ToResp() on nil should return nil")
	}

	// Test normal conversion
	ia := &InternetAccessible{
		InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
		InternetMaxBandwidthOut: 10,
		PublicIpAssigned:        true,
	}
	resp := ia.ToResp()
	if resp.InternetChargeType != ia.InternetChargeType {
		t.Errorf("ChargeType mismatch: got %q, want %q", resp.InternetChargeType, ia.InternetChargeType)
	}
	if resp.InternetMaxBandwidthOut != ia.InternetMaxBandwidthOut {
		t.Errorf("Bandwidth mismatch: got %d, want %d", resp.InternetMaxBandwidthOut, ia.InternetMaxBandwidthOut)
	}
	if resp.PublicIpAssigned != ia.PublicIpAssigned {
		t.Errorf("PublicIpAssigned mismatch: got %v, want %v", resp.PublicIpAssigned, ia.PublicIpAssigned)
	}
}

func TestSiteConfigGetSSOIMTypes(t *testing.T) {
	tests := []struct {
		name     string
		imType   string
		expected []string
	}{
		{"empty string", "", []string{}},
		{"single value (legacy)", "wecom", []string{"wecom"}},
		{"JSON array", `["wecom","feishu"]`, []string{"wecom", "feishu"}},
		{"JSON array single", `["dingtalk"]`, []string{"dingtalk"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SiteConfig{SSOIMType: tt.imType}
			result := config.GetSSOIMTypes()
			if len(result) != len(tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("got %v, want %v", result, tt.expected)
					return
				}
			}
		})
	}
}

func TestSiteConfigSetSSOIMTypes(t *testing.T) {
	tests := []struct {
		name     string
		types    []string
		expected string
	}{
		{"empty slice", []string{}, ""},
		{"single type", []string{"wecom"}, `["wecom"]`},
		{"multiple types", []string{"wecom", "feishu"}, `["wecom","feishu"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SiteConfig{}
			config.SetSSOIMTypes(tt.types)
			if config.SSOIMType != tt.expected {
				t.Errorf("got %q, want %q", config.SSOIMType, tt.expected)
			}
		})
	}
}

func TestSiteConfigGetSubnetMap(t *testing.T) {
	tests := []struct {
		name      string
		subnetIds string
		wantLen   int
		wantZone  string // if non-empty, assert map[wantZone] equals wantSubs
		wantSubs  []string
	}{
		{"empty string", "", 0, "", nil},
		{"empty object", "{}", 0, "", nil},
		{"new format single", `{"ap-shanghai-1":["subnet-123"]}`, 1, "ap-shanghai-1", []string{"subnet-123"}},
		{"new format multi", `{"ap-shanghai-1":["subnet-123","subnet-456"]}`, 1, "ap-shanghai-1", []string{"subnet-123", "subnet-456"}},
		{"old format single-valued compat", `{"ap-shanghai-1":"subnet-123"}`, 1, "ap-shanghai-1", []string{"subnet-123"}},
		{"old format multi zone compat", `{"ap-shanghai-1":"subnet-a","ap-shanghai-2":"subnet-b"}`, 2, "ap-shanghai-2", []string{"subnet-b"}},
		{"invalid JSON", "invalid", 0, "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SiteConfig{SubnetIds: tt.subnetIds}
			result := config.GetSubnetMap()
			if len(result) != tt.wantLen {
				t.Errorf("got %d entries, want %d", len(result), tt.wantLen)
			}
			if tt.wantZone != "" {
				got := result[tt.wantZone]
				if len(got) != len(tt.wantSubs) {
					t.Errorf("zone %s: got %v, want %v", tt.wantZone, got, tt.wantSubs)
					return
				}
				for i := range got {
					if got[i] != tt.wantSubs[i] {
						t.Errorf("zone %s[%d]: got %q, want %q", tt.wantZone, i, got[i], tt.wantSubs[i])
					}
				}
			}
		})
	}
}

func TestSiteConfigSetSubnetMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string][]string
		expected string
	}{
		{"empty map", map[string][]string{}, "{}"},
		{"nil map", nil, "{}"},
		{"single zone single subnet", map[string][]string{"ap-shanghai-1": {"subnet-123"}}, `{"ap-shanghai-1":["subnet-123"]}`},
		{"single zone multi subnet", map[string][]string{"ap-shanghai-1": {"subnet-a", "subnet-b"}}, `{"ap-shanghai-1":["subnet-a","subnet-b"]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SiteConfig{}
			err := config.SetSubnetMap(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if config.SubnetIds != tt.expected {
				t.Errorf("got %q, want %q", config.SubnetIds, tt.expected)
			}
		})
	}
}

func TestSiteConfigGetDefaultSubnetMap(t *testing.T) {
	tests := []struct {
		name      string
		subnetIds string
		wantLen   int
	}{
		{"empty string", "", 0},
		{"empty object", "{}", 0},
		{"new format", `{"ap-shanghai-1":["subnet-123","subnet-456"]}`, 1},
		{"old format", `{"ap-shanghai-1":"subnet-123"}`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SiteConfig{DefaultSubnetIds: tt.subnetIds}
			result := config.GetDefaultSubnetMap()
			if len(result) != tt.wantLen {
				t.Errorf("got %d entries, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestSiteConfigSetDefaultSubnetMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string][]string
		expected string
	}{
		{"empty map", map[string][]string{}, "{}"},
		{"nil map", nil, "{}"},
		{"valid map", map[string][]string{"ap-shanghai-1": {"subnet-123"}}, `{"ap-shanghai-1":["subnet-123"]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SiteConfig{}
			err := config.SetDefaultSubnetMap(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if config.DefaultSubnetIds != tt.expected {
				t.Errorf("got %q, want %q", config.DefaultSubnetIds, tt.expected)
			}
		})
	}
}

func TestGenerateGatewayUIPort(t *testing.T) {
	// Test that port is within expected range
	for i := 0; i < 100; i++ {
		port := GenerateGatewayUIPort()
		if port < 10000 || port > 40000 {
			t.Errorf("port %d is out of range [10000, 40000]", port)
		}
	}
}

func TestSMHConfigIsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		config SMHConfig
		want   bool
	}{
		{
			name:   "empty config",
			config: SMHConfig{},
			want:   false,
		},
		{
			name: "missing endpoint",
			config: SMHConfig{
				LibraryId:     "lib-123",
				LibrarySecret: "secret",
				CommonSpace:   "common",
				SkillhubSpace: "skillhub",
			},
			want: false,
		},
		{
			name: "fully configured",
			config: SMHConfig{
				Endpoint:      "https://smh.example.com",
				LibraryId:     "lib-123",
				LibrarySecret: "secret",
				CommonSpace:   "common",
				SkillhubSpace: "skillhub",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ==================== gdb Tests with SQLite Memory ====================

func TestGetDefaultAgentType(t *testing.T) {
	cleanup := setupSiteConfigTestDB(t)
	defer cleanup()

	// Test 1: No config exists - should return "openclaw" as default
	result := GetDefaultAgentType(context.Background())
	if result != AgentTypeOpenClaw {
		t.Errorf("expected default %q, got %q", AgentTypeOpenClaw, result)
	}

	// Test 2: Create config with empty DefaultAgentType
	config := SiteConfig{
		ID:               1,
		DefaultAgentType: "",
	}
	if err := gdb.Create(&config).Error; err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	result = GetDefaultAgentType(context.Background())
	if result != AgentTypeOpenClaw {
		t.Errorf("empty config should return %q, got %q", AgentTypeOpenClaw, result)
	}

	// Test 3: Set to hermes
	gdb.Model(&SiteConfig{}).Where("id = ?", 1).Update("default_agent_type", "hermes")
	result = GetDefaultAgentType(context.Background())
	if result != "hermes" {
		t.Errorf("expected hermes, got %q", result)
	}

	// Test 4: Set to invalid type - should fallback to openclaw
	gdb.Model(&SiteConfig{}).Where("id = ?", 1).Update("default_agent_type", "invalid_type")
	result = GetDefaultAgentType(context.Background())
	if result != AgentTypeOpenClaw {
		t.Errorf("invalid type should fallback to %q, got %q", AgentTypeOpenClaw, result)
	}

	// Test 5: Set to lightclawace
	gdb.Model(&SiteConfig{}).Where("id = ?", 1).Update("default_agent_type", "lightclawace")
	result = GetDefaultAgentType(context.Background())
	if result != "lightclawace" {
		t.Errorf("expected lightclawace, got %q", result)
	}
}

func TestSetDefaultAgentType(t *testing.T) {
	cleanup := setupSiteConfigTestDB(t)
	defer cleanup()

	// Create initial config
	config := SiteConfig{
		ID:               1,
		DefaultAgentType: "",
	}
	if err := gdb.Create(&config).Error; err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	// Test 1: Set valid type - openclaw
	err := SetDefaultAgentType(context.Background(), "openclaw")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var updated SiteConfig
	gdb.First(&updated)
	if updated.DefaultAgentType != "openclaw" {
		t.Errorf("expected openclaw, got %q", updated.DefaultAgentType)
	}

	// Test 2: Set valid type - hermes
	err = SetDefaultAgentType(context.Background(), "hermes")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	gdb.First(&updated)
	if updated.DefaultAgentType != "hermes" {
		t.Errorf("expected hermes, got %q", updated.DefaultAgentType)
	}

	// Test 3: Set valid type - lightclawace
	err = SetDefaultAgentType(context.Background(), "lightclawace")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	gdb.First(&updated)
	if updated.DefaultAgentType != "lightclawace" {
		t.Errorf("expected lightclawace, got %q", updated.DefaultAgentType)
	}

	// Test 4: Set disabled type as default - should be allowed
	if err := gdb.Model(&SiteConfig{}).Where("id = ?", 1).Update("disabled_agent_types", `["hermes"]`).Error; err != nil {
		t.Fatalf("failed to disable hermes: %v", err)
	}
	err = SetDefaultAgentType(context.Background(), "hermes")
	if err != nil {
		t.Errorf("expected disabled type can be set as default, got %v", err)
	}
	gdb.First(&updated)
	if updated.DefaultAgentType != "hermes" {
		t.Errorf("expected hermes, got %q", updated.DefaultAgentType)
	}

	// Test 5: Set invalid type - should return error
	err = SetDefaultAgentType(context.Background(), "invalid_type")
	if err == nil {
		t.Error("expected error for invalid type, got nil")
	}
	if err != nil && !contains(err.Error(), "无效的智能体类型") {
		t.Errorf("expected error to contain '无效的智能体类型', got %v", err)
	}

	// Verify the value was not changed
	gdb.First(&updated)
	if updated.DefaultAgentType != "hermes" {
		t.Errorf("invalid set should not change value, got %q", updated.DefaultAgentType)
	}

	// Test 6: Set empty type - should return error
	err = SetDefaultAgentType(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty type, got nil")
	}
}

// ============================================================================
// APIGatewayConfig 解析 / ShouldActivate
// ============================================================================

func TestSiteConfig_GetAPIGatewayConfig_Empty(t *testing.T) {
	cases := []struct{ name, raw string }{
		{"empty_string", ""},
		{"empty_object", "{}"},
		{"whitespace", "   "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sc := SiteConfig{APIGatewayConfig: c.raw}
			cfg, ok := sc.GetAPIGatewayConfig()
			if !ok {
				t.Fatalf("raw=%q: ok should be true", c.raw)
			}
			if cfg.Enable {
				t.Fatalf("expected Enable=false, got %+v", cfg)
			}
		})
	}
}

func TestSiteConfig_GetAPIGatewayConfig_Invalid(t *testing.T) {
	sc := SiteConfig{APIGatewayConfig: "{not json"}
	cfg, ok := sc.GetAPIGatewayConfig()
	if ok {
		t.Fatal("invalid JSON should return ok=false")
	}
	if cfg != (APIGatewayConfig{}) {
		t.Fatalf("expected zero config, got %+v", cfg)
	}
}

func TestSiteConfig_GetAPIGatewayConfig_Full(t *testing.T) {
	sc := SiteConfig{APIGatewayConfig: `{"enable":true,"gateway_instance_id":"ins-x","base_domain":"x.com"}`}
	cfg, ok := sc.GetAPIGatewayConfig()
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !cfg.Enable || cfg.GatewayInstanceID != "ins-x" || cfg.BaseDomain != "x.com" {
		t.Fatalf("unexpected cfg %+v", cfg)
	}
}

func TestAPIGatewayConfig_ShouldActivate(t *testing.T) {
	good := APIGatewayConfig{Enable: true, GatewayInstanceID: "ins", BaseDomain: "d.com"}
	tests := []struct {
		name string
		cfg  APIGatewayConfig
		sub  string
		want bool
	}{
		{"happy_path", good, "sub-1", true},
		{"disabled", APIGatewayConfig{Enable: false, GatewayInstanceID: "ins", BaseDomain: "d"}, "sub-1", false},
		{"no_oneid", good, "", false},
		{"missing_instance_id", APIGatewayConfig{Enable: true, BaseDomain: "d"}, "sub-1", false},
		{"missing_base_domain", APIGatewayConfig{Enable: true, GatewayInstanceID: "ins"}, "sub-1", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.ShouldActivate(tc.sub); got != tc.want {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}

// TestAPIGatewayConfig_SchemeOrDefault 覆盖协议字段的各分支：
// 仅 "http" / "https" 按值返回，其余（空、非法值）一律回落到 "http"。
func TestAPIGatewayConfig_SchemeOrDefault(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		want   string
	}{
		{"empty_defaults_to_http", "", "http"},
		{"http_kept", "http", "http"},
		{"https_kept", "https", "https"},
		{"uppercase_illegal", "HTTPS", "http"},
		{"ftp_illegal", "ftp", "http"},
		{"mixed_illegal", "httpz", "http"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := APIGatewayConfig{Scheme: tc.scheme}
			if got := cfg.SchemeOrDefault(); got != tc.want {
				t.Errorf("scheme=%q got %q want %q", tc.scheme, got, tc.want)
			}
		})
	}
}

func TestDisabledAgentTypeHelpers(t *testing.T) {
	cleanup := setupSiteConfigTestDB(t)
	defer cleanup()

	if got := (SiteConfig{DisabledAgentTypes: ""}).GetDisabledAgentTypes(); len(got) != 0 {
		t.Fatalf("empty disabled types = %v, want empty", got)
	}
	if got := (SiteConfig{DisabledAgentTypes: "not-json"}).GetDisabledAgentTypes(); len(got) != 0 {
		t.Fatalf("invalid disabled types = %v, want empty", got)
	}
	parsed := (SiteConfig{DisabledAgentTypes: `[" hermes ", "", "hermes", "lightclawace"]`}).GetDisabledAgentTypes()
	if len(parsed) != 2 || parsed[0] != AgentTypeHermes || parsed[1] != AgentTypeLightclawACE {
		t.Fatalf("parsed disabled types = %v", parsed)
	}

	if err := gdb.Create(&SiteConfig{ID: 1, DefaultAgentType: AgentTypeOpenClaw}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	if err := SetDisabledAgentTypes(context.Background(), []string{"", "bad-type"}); err == nil {
		t.Fatal("invalid disabled type should fail")
	}
	if err := SetDisabledAgentTypes(context.Background(), []string{AgentTypeOpenClaw}); err == nil {
		t.Fatal("default type should not be disabled")
	}
	if err := SetDisabledAgentTypes(context.Background(), []string{" hermes ", "hermes", AgentTypeLightclawACE}); err != nil {
		t.Fatalf("set disabled types: %v", err)
	}
	if IsAgentTypeEnabled(context.Background(), AgentTypeHermes) {
		t.Fatal("hermes should be disabled")
	}
	if !IsAgentTypeEnabled(context.Background(), "") {
		t.Fatal("empty type should normalize to enabled openclaw")
	}
	filtered := FilterEnabledAgentTypes(context.Background(), []string{AgentTypeOpenClaw, AgentTypeHermes, AgentTypeLightclawACE})
	if len(filtered) != 1 || filtered[0] != AgentTypeOpenClaw {
		t.Fatalf("filtered types = %v", filtered)
	}

	if err := SetAgentTypeEnabled(context.Background(), "", false); err == nil {
		t.Fatal("empty agent type should fail")
	}
	if err := SetAgentTypeEnabled(context.Background(), "bad-type", false); err == nil {
		t.Fatal("invalid agent type should fail")
	}
	if err := SetAgentTypeEnabled(context.Background(), AgentTypeOpenClaw, false); err == nil {
		t.Fatal("default agent type should not be disabled")
	}
	if err := SetAgentTypeEnabled(context.Background(), AgentTypeHermes, false); err != nil {
		t.Fatalf("disable already disabled hermes: %v", err)
	}
	if err := SetAgentTypeEnabled(context.Background(), AgentTypeHermes, true); err != nil {
		t.Fatalf("enable hermes: %v", err)
	}
	if !IsAgentTypeEnabled(context.Background(), AgentTypeHermes) {
		t.Fatal("hermes should be enabled")
	}
}

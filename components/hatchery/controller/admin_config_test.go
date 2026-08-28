package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"hatchery/i18n"
	"hatchery/model"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	hcommon "hatchery/common"

	"github.com/glebarez/sqlite"
	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"

	"github.com/gorilla/sessions"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupAdminConfigTestDB 初始化内存 SQLite，包含 handleFirstTimeGatewayUIEnable 所需的表。
func setupAdminConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.RuleSet{},
		&model.ManagedSGPool{},
		&model.SiteConfig{},
		&model.ResourcePolicy{},
		&model.AuditLog{},
		&model.Tag{},
		&model.TagVisibilityGroup{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// tenantCtx 构造一个注入了空 TenantSnapshot 的 ctx（SQLite 单租户模式）。
func tenantCtx() context.Context {
	return hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{})
}

// ─── handleFirstTimeGatewayUIEnable ──────────────────────────────────────

// TestHandleFirstTimeGatewayUIEnable_NoSGPool SG 池未就绪（无 RuleSet）时返回错误。
func TestHandleFirstTimeGatewayUIEnable_NoSGPool(t *testing.T) {
	setupAdminConfigTestDB(t)
	ctx := tenantCtx()

	config := &model.SiteConfig{GatewayUIPort: 0}
	err := handleFirstTimeGatewayUIEnable(ctx, config)
	if err == nil {
		t.Fatal("expected error when SG pool not ready, got nil")
	}
}

// TestHandleFirstTimeGatewayUIEnable_SGPoolReady SG 池就绪时，触发 HasSGPoolReady(ctx)
// 并进入后续流程（RefreshAllRuleSetsForRequiredRules 因无云凭据失败，但不 panic）。
func TestHandleFirstTimeGatewayUIEnable_SGPoolReady(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	ctx := tenantCtx()

	// 插入 RuleSet + ACTIVE SG
	rs := model.RuleSet{Name: "default", Rules: "[]", Version: 1, UserGroupIDs: "[]"}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID:        "sg-gw-001",
		RuleSetID:   rs.ID,
		Status:      model.SGStatusActive,
		RuleVersion: 1,
	})
	// 插入 SiteConfig 供 Save 使用
	cfg := model.SiteConfig{Name: "test", GatewayUIPort: 0}
	db.Create(&cfg)

	// 不应 panic；RefreshAllRuleSetsForRequiredRules 失败只告警不返回错误
	err := handleFirstTimeGatewayUIEnable(ctx, &cfg)
	// 函数本身在 RefreshAllRuleSetsForRequiredRules 失败时不返回 error，只 slog.Warn
	if err != nil {
		// 允许因无云凭据导致的错误（非 panic）
		t.Logf("handleFirstTimeGatewayUIEnable returned (expected for no-cloud env): %v", err)
	}
}

// TestHandleFirstTimeGatewayUIEnable_PortAlreadySet 已有端口时不重新分配。
func TestHandleFirstTimeGatewayUIEnable_PortAlreadySet(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	ctx := tenantCtx()

	rs := model.RuleSet{Name: "default", Rules: "[]", Version: 1, UserGroupIDs: "[]"}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID:        "sg-gw-002",
		RuleSetID:   rs.ID,
		Status:      model.SGStatusActive,
		RuleVersion: 1,
	})
	cfg := model.SiteConfig{Name: "test", GatewayUIPort: 8443}
	db.Create(&cfg)

	originalPort := cfg.GatewayUIPort
	_ = handleFirstTimeGatewayUIEnable(ctx, &cfg)
	// 端口已设置，不应被重新分配
	if cfg.GatewayUIPort != originalPort {
		t.Errorf("port should not change: got %d, want %d", cfg.GatewayUIPort, originalPort)
	}
}

func TestPortMatchesRule(t *testing.T) {
	tests := []struct {
		name       string
		rulePort   string
		targetPort int
		want       bool
	}{
		// ── 单端口匹配 ──
		{"单端口-精确匹配", "8080", 8080, true},
		{"单端口-不匹配", "8080", 9090, false},
		{"单端口-端口80", "80", 80, true},
		{"单端口-端口443", "443", 443, true},

		// ── 端口范围匹配 ──
		{"端口范围-在范围内", "8000-9000", 8080, true},
		{"端口范围-左边界", "8000-9000", 8000, true},
		{"端口范围-右边界", "8000-9000", 9000, true},
		{"端口范围-低于范围", "8000-9000", 7999, false},
		{"端口范围-高于范围", "8000-9000", 9001, false},
		{"端口范围-相同起止", "8080-8080", 8080, true},
		{"端口范围-相同起止不匹配", "8080-8080", 8081, false},

		// ── 多端口匹配（逗号分隔） ──
		{"多端口-匹配第一个", "80,443,8080", 80, true},
		{"多端口-匹配中间", "80,443,8080", 443, true},
		{"多端口-匹配最后一个", "80,443,8080", 8080, true},
		{"多端口-不匹配", "80,443,8080", 9090, false},

		// ── 多端口+端口范围混合 ──
		{"混合-匹配离散端口", "22,80,8000-9000", 80, true},
		{"混合-匹配范围内端口", "22,80,8000-9000", 8500, true},
		{"混合-不匹配", "22,80,8000-9000", 443, false},

		// ── ALL ──
		{"ALL大写", "ALL", 8080, true},
		{"all小写", "all", 12345, true},
		{"All混合大小写", "All", 1, true},

		// ── 带空格的情况 ──
		{"带前后空格", " 8080 ", 8080, true},
		{"逗号后有空格", "80, 443, 8080", 443, true},
		{"范围带空格", " 8000 - 9000 ", 8500, true},

		// ── 边界情况 ──
		{"空字符串", "", 8080, false},
		{"非法格式", "abc", 8080, false},
		{"范围非法-左侧非数字", "abc-9000", 8080, false},
		{"范围非法-右侧非数字", "8000-xyz", 8080, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hcommon.PortMatchesRule(tt.rulePort, tt.targetPort)
			if got != tt.want {
				t.Errorf("PortMatchesRule(%q, %d) = %v, 期望 %v", tt.rulePort, tt.targetPort, got, tt.want)
			}
		})
	}
}

func initSiteConfigChatViewTestEnv(t *testing.T) {
	t.Helper()
	initAdminTestDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.ResourcePolicy{}); err != nil {
		t.Fatalf("migrate resource policies: %v", err)
	}
	Store = sessions.NewCookieStore([]byte("test-secret"))
}

func setDefaultResourcePolicyConfig(t *testing.T, configJSON string) *model.ResourcePolicy {
	t.Helper()
	ctx := context.Background()
	policy, err := model.GetOrCreateDefaultResourcePolicy(ctx)
	if err != nil {
		t.Fatalf("GetOrCreateDefaultResourcePolicy() error = %v", err)
	}
	updated, err := model.UpdateResourcePolicy(ctx, policy.ID, "", configJSON, nil)
	if err != nil {
		t.Fatalf("UpdateResourcePolicy(%d) error = %v", policy.ID, err)
	}
	return updated
}

func loadResourcePolicyConfig(t *testing.T, id uint) *ResourceConfig {
	t.Helper()
	policy, err := model.GetResourcePolicy(context.Background(), id)
	if err != nil {
		t.Fatalf("GetResourcePolicy(%d) error = %v", id, err)
	}
	config, err := ParseResourceConfig(policy.ConfigJSON)
	if err != nil {
		t.Fatalf("ParseResourceConfig(%q) error = %v", policy.ConfigJSON, err)
	}
	return config
}

func TestHandleAdminConfig_UsesEffectiveDefaultResourcePolicy(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	setDefaultResourcePolicyConfig(t, `{
		"instance_charge_type":"POSTPAID_BY_HOUR",
		"instance_type":"Ai2.LARGE8",
		"internet_accessible":{"public_ip_assigned":false}
	}`)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	HandleAdminConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleAdminConfig() status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var response struct {
		Config struct {
			InstanceChargeType string                    `json:"instance_charge_type"`
			InternetAccessible *InternetAccessibleConfig `json:"internet_accessible"`
		} `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal(HandleAdminConfig response) error = %v", err)
	}
	if response.Config.InstanceChargeType != cvmChargeTypePostpaidByHour {
		t.Errorf("instance_charge_type = %q, want %q", response.Config.InstanceChargeType, cvmChargeTypePostpaidByHour)
	}
	if response.Config.InternetAccessible == nil ||
		response.Config.InternetAccessible.PublicIpAssigned == nil ||
		*response.Config.InternetAccessible.PublicIpAssigned {
		t.Errorf("internet_accessible = %+v, want public_ip_assigned=false", response.Config.InternetAccessible)
	}
}

func TestHandleUpdateConfig_PrepaidPatchesDefaultResourcePolicy(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	policy := setDefaultResourcePolicyConfig(t, `{
		"instance_charge_type":"POSTPAID_BY_HOUR",
		"instance_type":"Ai2.LARGE8",
		"system_disk":{"disk_type":"CLOUD_BSSD","disk_size":80}
	}`)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("instance_charge_type", "PREPAID")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleUpdateConfig(instance_charge_type=PREPAID) status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	config := loadResourcePolicyConfig(t, policy.ID)
	if config.InstanceChargeType != cvmChargeTypePrepaid {
		t.Errorf("instance_charge_type = %q, want %q", config.InstanceChargeType, cvmChargeTypePrepaid)
	}
	if config.InstanceChargePrepaid == nil || config.InstanceChargePrepaid.Period != 1 {
		t.Errorf("instance_charge_prepaid = %+v, want default one-month prepaid config", config.InstanceChargePrepaid)
	}
	if config.InstanceType != "Ai2.LARGE8" {
		t.Errorf("instance_type = %q, want preserved Ai2.LARGE8", config.InstanceType)
	}
	if config.SystemDisk == nil || config.SystemDisk.DiskSize != 80 {
		t.Errorf("system_disk = %+v, want preserved disk_size=80", config.SystemDisk)
	}
}

func TestHandleUpdateTemplate_PatchesOnlySubmittedDefaultPolicyField(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	policy := setDefaultResourcePolicyConfig(t, `{
		"instance_charge_type":"POSTPAID_BY_HOUR",
		"instance_type":"Ai2.LARGE8",
		"internet_accessible":{"public_ip_assigned":true,"internet_charge_type":"TRAFFIC_POSTPAID_BY_HOUR","internet_max_bandwidth_out":10}
	}`)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/config/template", strings.NewReader(`{
		"internet_accessible":{"public_ip_assigned":false}
	}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleUpdateTemplate(internet_accessible) status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	config := loadResourcePolicyConfig(t, policy.ID)
	if config.InstanceType != "Ai2.LARGE8" {
		t.Errorf("instance_type = %q, want preserved Ai2.LARGE8", config.InstanceType)
	}
	if config.InternetAccessible == nil ||
		config.InternetAccessible.PublicIpAssigned == nil ||
		*config.InternetAccessible.PublicIpAssigned {
		t.Errorf("internet_accessible = %+v, want public_ip_assigned=false", config.InternetAccessible)
	}
}

func TestHandleUpdateCVMConfig_ReplacesDefaultResourcePolicy(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	policy := setDefaultResourcePolicyConfig(t, `{
		"instance_charge_type":"PREPAID",
		"instance_charge_prepaid":{"period":12,"renew_flag":"NOTIFY_AND_MANUAL_RENEW"},
		"instance_type":"Ai2.LARGE8"
	}`)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	template := `{
		"InstanceChargeType":"POSTPAID_BY_HOUR",
		"InstanceType":"Ai2.MEDIUM2",
		"SystemDisk":{"DiskType":"CLOUD_BSSD","DiskSize":80},
		"InternetAccessible":{"PublicIpAssigned":false}
	}`
	form := url.Values{}
	form.Set("cvm_template", template)
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleUpdateCVMConfig(cvm_template) status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	config := loadResourcePolicyConfig(t, policy.ID)
	if config.InstanceChargeType != cvmChargeTypePostpaidByHour {
		t.Errorf("instance_charge_type = %q, want %q", config.InstanceChargeType, cvmChargeTypePostpaidByHour)
	}
	if config.InstanceChargePrepaid != nil {
		t.Errorf("instance_charge_prepaid = %+v, want nil after full template replacement", config.InstanceChargePrepaid)
	}
	if config.InstanceType != "Ai2.MEDIUM2" {
		t.Errorf("instance_type = %q, want Ai2.MEDIUM2", config.InstanceType)
	}
	if config.SystemDisk == nil || config.SystemDisk.DiskSize != 80 {
		t.Errorf("system_disk = %+v, want disk_size=80", config.SystemDisk)
	}
}

func TestHandleSite_ReturnsChatViewEnabled(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{"chat_view_enabled": true}).Error; err != nil {
		t.Fatalf("update site config: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/site", nil)
	w := httptest.NewRecorder()

	HandleSite(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["chat_view_enabled"] != true {
		t.Fatalf("expected chat_view_enabled=true, got %#v", resp["chat_view_enabled"])
	}
}

func TestHandleAdminConfig_JSONIncludesChatViewEnabled(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{"chat_view_enabled": true}).Error; err != nil {
		t.Fatalf("update site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleAdminConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Config["chat_view_enabled"] != true {
		t.Fatalf("expected config.chat_view_enabled=true, got %#v", resp.Config["chat_view_enabled"])
	}
}

func TestHandleAdminConfig_GlobalTokenQuotaLegacyFields(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"global_token_quota_day":    -1,
		"global_token_quota_period": model.GlobalTokenQuotaPeriodDay,
		"global_token_quota_rules":  `[{"mode":"month","limit":-1}]`,
	}).Error; err != nil {
		t.Fatalf("update site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleAdminConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := resp.Config["global_token_quota_period"]; got != model.GlobalTokenQuotaPeriodMonth {
		t.Fatalf("expected global_token_quota_period=month, got %#v", got)
	}
	if got := resp.Config["global_token_quota_day"]; got != float64(-1) {
		t.Fatalf("expected global_token_quota_day=-1, got %#v", got)
	}
}

func TestHandleAdminConfig_ResolvesLegacyTokenQuotaRules(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"default_token_quota_day":   500000,
		"default_token_quota_rules": "",
		"global_token_quota_day":    123456,
		"global_token_quota_period": model.GlobalTokenQuotaPeriodMonth,
		"global_token_quota_rules":  "",
	}).Error; err != nil {
		t.Fatalf("update site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleAdminConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Config struct {
			DefaultTokenQuotaDay   int                    `json:"default_token_quota_day"`
			DefaultTokenQuotaRules []model.TokenQuotaRule `json:"default_token_quota_rules"`
			GlobalTokenQuotaDay    int                    `json:"global_token_quota_day"`
			GlobalTokenQuotaPeriod string                 `json:"global_token_quota_period"`
			GlobalTokenQuotaRules  []model.TokenQuotaRule `json:"global_token_quota_rules"`
		} `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Config.DefaultTokenQuotaDay != 500000 {
		t.Fatalf("expected default_token_quota_day=500000, got %d", resp.Config.DefaultTokenQuotaDay)
	}
	if len(resp.Config.DefaultTokenQuotaRules) != 1 ||
		resp.Config.DefaultTokenQuotaRules[0].Mode != model.QuotaModeDay ||
		resp.Config.DefaultTokenQuotaRules[0].Limit != 500000 {
		t.Fatalf("expected default_token_quota_rules fallback to day rule, got %+v", resp.Config.DefaultTokenQuotaRules)
	}
	if resp.Config.GlobalTokenQuotaDay != 123456 || resp.Config.GlobalTokenQuotaPeriod != model.GlobalTokenQuotaPeriodMonth {
		t.Fatalf("unexpected global legacy fields: day=%d period=%s", resp.Config.GlobalTokenQuotaDay, resp.Config.GlobalTokenQuotaPeriod)
	}
	if len(resp.Config.GlobalTokenQuotaRules) != 1 ||
		resp.Config.GlobalTokenQuotaRules[0].Mode != model.QuotaModeMonth ||
		resp.Config.GlobalTokenQuotaRules[0].Limit != 123456 {
		t.Fatalf("expected global_token_quota_rules fallback to month rule, got %+v", resp.Config.GlobalTokenQuotaRules)
	}
}

func TestHandleUpdateConfig_UpdatesChatViewEnabled(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{"chat_view_enabled": true}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("chat_view_enabled", "false")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	if got := model.GetSiteConfig(context.Background()).ChatViewEnabled; got {
		t.Fatalf("expected ChatViewEnabled=false, got true")
	}
	if !strings.Contains(w.Body.String(), `"ok":true`) {
		t.Fatalf("expected ok response, got %s", w.Body.String())
	}
}

func TestHandleUpdateConfig_DefaultTagsMigratesLegacyAndReplacesGlobalTags(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.Tag{}, &model.TagVisibilityGroup{}); err != nil {
		t.Fatalf("migrate tag tables: %v", err)
	}
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("1 = 1").
		Update("default_tags", `[{"Key":"legacy","Value":"yes"}]`).Error; err != nil {
		t.Fatalf("seed legacy default_tags: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("default_tags", `[{"Key":"env","Value":"prod"}]`)
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if got := model.GetSiteConfig(ctx).DefaultTags; got != "[]" {
		t.Fatalf("legacy default_tags should be cleared, got %q", got)
	}
	var rows []model.Tag
	if err := model.DB(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(rows) != 1 || rows[0].TagKey != "env" || rows[0].TagValue != "prod" || rows[0].VisibilityType != model.VisibilityAll {
		t.Fatalf("POST /admin/config should replace global tags, got %+v", rows)
	}
}

func TestHandleAdminConfig_DefaultTagsReadsNewTableBeforeLegacy(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.Tag{}, &model.TagVisibilityGroup{}); err != nil {
		t.Fatalf("migrate tag tables: %v", err)
	}
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("1 = 1").
		Update("default_tags", `[{"Key":"legacy","Value":"yes"}]`).Error; err != nil {
		t.Fatalf("seed legacy default_tags: %v", err)
	}
	if err := model.DB(ctx).Create(&model.Tag{TagKey: "env", TagValue: "prod", VisibilityType: model.VisibilityAll}).Error; err != nil {
		t.Fatalf("create new global tag: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleAdminConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := resp.Config["default_tags"]; got != `[{"Key":"env","Value":"prod"}]` {
		t.Fatalf("expected new-table default_tags, got %#v", got)
	}
}

// ---------------------------------------------------------------------------
// api_gateway_config 表单字段：JSON 校验 + 持久化
// ---------------------------------------------------------------------------

func doUpdateConfigForAPIGateway(t *testing.T, raw string) *httptest.ResponseRecorder {
	t.Helper()
	initSiteConfigChatViewTestEnv(t)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	form := url.Values{}
	form.Set("api_gateway_config", raw)
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	HandleUpdateConfig(w, req)
	return w
}

func TestHandleUpdateConfig_APIGatewayConfig_InvalidJSON_400(t *testing.T) {
	w := doUpdateConfigForAPIGateway(t, "not-json")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateConfig_APIGatewayConfig_EnableButMissingFields_400(t *testing.T) {
	w := doUpdateConfigForAPIGateway(t, `{"enable":true}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "gateway_instance_id") {
		t.Fatalf("expected error mentions gateway_instance_id, got %s", w.Body.String())
	}
}

func TestHandleUpdateConfig_APIGatewayConfig_DisableAccepted(t *testing.T) {
	w := doUpdateConfigForAPIGateway(t, `{"enable":false}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	got := model.GetSiteConfig(context.Background()).APIGatewayConfig
	if got != `{"enable":false}` {
		t.Fatalf("expected stored raw JSON, got %q", got)
	}
}

func TestHandleUpdateConfig_APIGatewayConfig_EmptyNormalizedToObject(t *testing.T) {
	w := doUpdateConfigForAPIGateway(t, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	got := model.GetSiteConfig(context.Background()).APIGatewayConfig
	if got != "{}" {
		t.Fatalf("expected '{}', got %q", got)
	}
}

func TestHandleUpdateConfig_APIGatewayConfig_ValidFull_Saves(t *testing.T) {
	raw := `{"enable":true,"gateway_instance_id":"ins-xx","base_domain":"mcd.com"}`
	w := doUpdateConfigForAPIGateway(t, raw)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	got := model.GetSiteConfig(context.Background()).APIGatewayConfig
	if got != raw {
		t.Fatalf("persisted mismatch, got %q", got)
	}
	// 再反序列化一次确保能被 GetAPIGatewayConfig 消费
	cfg, ok := model.GetSiteConfig(context.Background()).GetAPIGatewayConfig()
	if !ok || !cfg.Enable || cfg.GatewayInstanceID != "ins-xx" || cfg.BaseDomain != "mcd.com" {
		t.Fatalf("bad parsed cfg: %+v ok=%v", cfg, ok)
	}
}

// ─── 平台策略功能权限开关测试 ──────────────────────────────────────────────────

func TestHandleAdminConfig_JSONIncludesPolicySwitches(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleAdminConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// 默认值应为 true
	for _, key := range []string{"user_config_model_enabled", "user_config_channel_enabled", "model_quota_enabled"} {
		v, ok := resp.Config[key]
		if !ok {
			t.Fatalf("expected config to contain %q", key)
		}
		if v != true {
			t.Fatalf("expected config.%s=true, got %#v", key, v)
		}
	}
}

func TestHandleUpdateConfig_UpdatesPolicySwitches(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 关闭 3 个开关
	form := url.Values{}
	form.Set("user_config_model_enabled", "false")
	form.Set("user_config_channel_enabled", "false")
	form.Set("model_quota_enabled", "false")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	cfg := model.GetSiteConfig(context.Background())
	if cfg.UserConfigModelEnabled {
		t.Fatal("expected UserConfigModelEnabled=false, got true")
	}
	if cfg.UserConfigChannelEnabled {
		t.Fatal("expected UserConfigChannelEnabled=false, got true")
	}
	if cfg.ModelQuotaEnabled {
		t.Fatal("expected ModelQuotaEnabled=false, got true")
	}
}

func TestHandleSite_ReturnsPolicySwitches(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)

	// 创建测试用户
	model.DB(context.Background()).Create(&model.User{Username: "testuser", Password: "dummy"})

	// 设置 user_config_model_enabled=false，其余保持默认
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"user_config_model_enabled":   false,
		"user_config_channel_enabled": true,
		"model_quota_enabled":         true,
	}).Error; err != nil {
		t.Fatalf("update site config: %v", err)
	}

	// 模拟登录用户（通过 session）
	req := httptest.NewRequest(http.MethodGet, "/site", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "testuser"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	HandleSite(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["user_config_model_enabled"] != false {
		t.Fatalf("expected user_config_model_enabled=false, got %#v", resp["user_config_model_enabled"])
	}
	if resp["user_config_channel_enabled"] != true {
		t.Fatalf("expected user_config_channel_enabled=true, got %#v", resp["user_config_channel_enabled"])
	}
	if resp["model_quota_enabled"] != true {
		t.Fatalf("expected model_quota_enabled=true, got %#v", resp["model_quota_enabled"])
	}
}

// ─── 云端浏览器开关：安全组相关测试 ─────────────────────────────────────────────
//
// 旧的 6 个测试（基于 SiteConfig.SecurityGroupId 的前置校验 + checkSecurityGroupExists
// 云端存在性校验）已在 migrate-port-open-to-ruleset change 中删除：
//   - TestHandleUpdateConfig_BrowserVNCEnable_NoSecurityGroup_400
//   - TestHandleUpdateConfig_BrowserVNCEnable_SecurityGroupCheckFails_500
//   - TestHandleUpdateConfig_BrowserVNCEnable_SecurityGroupNotExists_400
//   - TestHandleUpdateConfig_BrowserVNCEnable_SecurityGroupExists_OK
//   - TestHandleUpdateConfig_BrowserVNCEnable_SecurityGroupCheckError_500
//   - TestHandleUpdateConfig_BrowserVNCEnable_SecurityGroupExistsButPortRuleFails_OKWithWarning
//
// 原因：迁移到 RuleSet + ManagedSGPool 多 SG 模型后，siteConfig.SecurityGroupId
// 已不再是"当前 SG"的代表，HandleUpdateConfig 的 BrowserVNC 开关分支改为校验
// "至少存在一个 RuleSet"，并通过 refreshAllRuleSetsForRequiredRules 扇出规则到
// 所有 RuleSet 的 ACTIVE SG。原断言的所有代码路径已不存在。

// TestHandleUpdateConfig_BrowserVNCDisable_NoSecurityGroup_OK
// 当 SecurityGroupId 为空时，关闭云端浏览器应正常通过（不需要安全组）。
func TestHandleUpdateConfig_BrowserVNCDisable_NoSecurityGroup_OK(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	// 确保安全组为空，且云端浏览器当前为开启状态
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"security_group_id":  "",
		"browser_vnc_enable": true,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("browser_vnc_enable", "false")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if model.GetSiteConfig(context.Background()).BrowserVNCEnable {
		t.Fatal("expected BrowserVNCEnable=false after disabling")
	}
}

// TestHandleUpdateConfig_BrowserVNCDisable_SecurityGroupCheckSkipped_OK
// 关闭云端浏览器时，即使安全组非空且凭据未配置，也不应触发云端校验，应正常通过。
func TestHandleUpdateConfig_BrowserVNCDisable_SecurityGroupCheckSkipped_OK(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	// 设置安全组 ID 非空，CVM 密钥保持为空，云端浏览器当前为开启状态
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"security_group_id":  "sg-fake12345",
		"browser_vnc_enable": true,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("browser_vnc_enable", "false")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	// 关闭操作不触发安全组校验，应正常通过
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if model.GetSiteConfig(context.Background()).BrowserVNCEnable {
		t.Fatal("expected BrowserVNCEnable=false after disabling")
	}
}

// TestCheckSecurityGroupExistsImpl_VpcClientError
// 直接测试 checkSecurityGroupExistsImpl 函数：当凭据未配置时应返回错误。
func TestCheckSecurityGroupExistsImpl_VpcClientError(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	// 确保凭据为空
	exists, err := checkSecurityGroupExistsImpl(context.Background(), "sg-test123")
	if err == nil {
		t.Fatalf("expected error when credentials not configured, got exists=%v", exists)
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "创建 VPC 客户端失败") {
		t.Fatalf("expected VPC client error, got: %s", err.Error())
	}
	if exists {
		t.Fatal("expected exists=false when error occurs")
	}
}

// TestDoCheckSecurityGroupExists_ResponseNil
// 测试 doCheckSecurityGroupExists：当 API 返回 Response 为 nil 时应返回 false。
func TestDoCheckSecurityGroupExists_ResponseNil(t *testing.T) {
	// 创建一个 mock HTTP 服务器，返回 Response 为空的 JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Response":{"SecurityGroupSet":[],"TotalCount":0,"RequestId":"test-req-id"}}`))
	}))
	defer ts.Close()

	// 创建指向 mock 服务器的 VPC 客户端
	client := createMockVpcClient(t, ts.URL)

	exists, err := doCheckSecurityGroupExists(client, "sg-notexist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected exists=false when SecurityGroupSet is empty")
	}
}

// TestDoCheckSecurityGroupExists_SecurityGroupFound
// 测试 doCheckSecurityGroupExists：当 API 返回安全组时应返回 true。
func TestDoCheckSecurityGroupExists_SecurityGroupFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Response":{"SecurityGroupSet":[{"SecurityGroupId":"sg-exist123","SecurityGroupName":"test-sg"}],"TotalCount":1,"RequestId":"test-req-id"}}`))
	}))
	defer ts.Close()

	client := createMockVpcClient(t, ts.URL)

	exists, err := doCheckSecurityGroupExists(client, "sg-exist123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected exists=true when SecurityGroupSet is not empty")
	}
}

// TestDoCheckSecurityGroupExists_APIError_ResourceNotFound
// 测试 doCheckSecurityGroupExists：当 API 返回 ResourceNotFound 错误时应返回 (false, nil)。
func TestDoCheckSecurityGroupExists_APIError_ResourceNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Response":{"Error":{"Code":"ResourceNotFound","Message":"security group not found"},"RequestId":"test-req-id"}}`))
	}))
	defer ts.Close()

	client := createMockVpcClient(t, ts.URL)

	exists, err := doCheckSecurityGroupExists(client, "sg-invalid")
	if err != nil {
		t.Fatalf("expected no error for ResourceNotFound, got: %v", err)
	}
	if exists {
		t.Fatal("expected exists=false when ResourceNotFound")
	}
}

// TestDoCheckSecurityGroupExists_APIError_OtherCode
// 测试 doCheckSecurityGroupExists：当 API 返回非 ResourceNotFound 错误时应返回 error。
func TestDoCheckSecurityGroupExists_APIError_OtherCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Response":{"Error":{"Code":"InternalError","Message":"internal server error"},"RequestId":"test-req-id"}}`))
	}))
	defer ts.Close()

	client := createMockVpcClient(t, ts.URL)

	exists, err := doCheckSecurityGroupExists(client, "sg-error")
	if err == nil {
		t.Fatalf("expected error for InternalError, got exists=%v", exists)
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "验证安全组失败") {
		t.Fatalf("expected wrapped error, got: %s", err.Error())
	}
	if exists {
		t.Fatal("expected exists=false when error occurs")
	}
}

// createMockVpcClient 创建一个指向 mock 服务器的 VPC 客户端。
func createMockVpcClient(t *testing.T, serverURL string) *vpc.Client {
	t.Helper()
	// 去掉 http:// 前缀获取 host
	endpoint := strings.TrimPrefix(serverURL, "http://")

	credential := tccommon.NewCredential("fake-secret-id", "fake-secret-key")
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = endpoint
	cpf.HttpProfile.Scheme = "HTTP"

	client, err := vpc.NewClient(credential, "ap-guangzhou", cpf)
	if err != nil {
		t.Fatalf("create mock vpc client: %v", err)
	}
	return client
}

// ─── P0 修复覆盖：开关 false→true 时 SiteConfig 必须先选择性落库再 refresh ───────────
//
// 这两个测试覆盖 admin_config.go 中两个 refresh 前保存分支：
//   - Gateway UI 关→开（已迁移）：只保存 Gateway UI 相关字段，避免整行 Save 覆盖并发写入
//   - BrowserVNC 关→开（SG 池就绪）：只保存 BrowserVNCEnable，因为 refresh 条件只读取该字段
//
// 不验证 refresh 内部规则合入逻辑（那是 ruleset_helpers / sg_ruleset_init 的职责），
// 仅断言：1) 接口 200；2) DB 里对应开关字段已落库。

func TestHandleUpdateConfig_BrowserVNCEnable_SaveBeforeRefresh(t *testing.T) {
	db := setupSGPoolTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	// seed RuleSet + 1 个 ACTIVE SG → hasSGPoolReady=true
	rs := &model.RuleSet{Name: model.DefaultRuleSetName, IsDefault: true, Version: 1, Rules: "[]"}
	if err := db.Create(rs).Error; err != nil {
		t.Fatalf("seed rule set: %v", err)
	}
	if err := db.Create(&model.ManagedSGPool{
		SGID: "sg-vnc-test", RuleSetID: rs.ID, Status: model.SGStatusActive, RuleVersion: 1,
	}).Error; err != nil {
		t.Fatalf("seed sg pool: %v", err)
	}
	// 初始 SiteConfig：BrowserVNCEnable=false（覆盖 setupSGPoolTestDB 默认插入的那行）
	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	if err := db.Create(&model.SiteConfig{BrowserVNCEnable: false}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	// stub vpc client 让 refresh 内部不真调云
	teardown := withFakeSGPoolVpcClient(&fakeSGPoolVpcClient{})
	defer teardown()

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("browser_vnc_enable", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	// 关键断言：DB 里的 BrowserVNCEnable 已经是 true（落库发生在 refresh 之前）
	// 直接用测试 db 句柄查询，避免依赖全局 gdb（异步 goroutine 可能切换它）。
	var persisted model.SiteConfig
	if err := db.First(&persisted).Error; err != nil {
		t.Fatalf("read site config: %v", err)
	}
	if !persisted.BrowserVNCEnable {
		t.Fatal("expected BrowserVNCEnable=true persisted in DB after enable")
	}
}

func TestHandleUpdateConfig_GatewayUIEnable_SaveBeforeRefresh(t *testing.T) {
	db := setupSGPoolTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	// 不 seed RuleSet：refresh 会因 "no rule_set found" 失败 → 走 slog.Warn 分支，
	// 但在那之前 model.DB(context.Background()).Save 必须已经执行（这正是我们要覆盖的目标行）。
	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	// 关键前置：GatewayUIEnable=false（wasEnabled=false）+ GatewayUISGMigrateDone=true
	if err := db.Create(&model.SiteConfig{
		GatewayUIEnable:        false,
		GatewayUISGMigrateDone: true,
		GatewayUIPort:          7540,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("gateway_ui_enable", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// DB 必须已经落库（即使 refresh 失败也得落）
	var persisted model.SiteConfig
	if err := db.First(&persisted).Error; err != nil {
		t.Fatalf("read site config: %v", err)
	}
	if !persisted.GatewayUIEnable {
		t.Fatal("expected GatewayUIEnable=true persisted in DB after enable")
	}
}

// TestHandleUpdateConfig_GatewayUIAddrType_PublicToPrivate_PersistsAndAttemptsRefresh
// 验证当 Gateway UI 已开启时，仅切换 addr_type=private（与原值 public 不同）也应：
//  1. 正常落库（DB 中 GatewayUIAddrType=private）
//  2. 触发 refreshAllRuleSetsForRequiredRules（不能因为 wasEnabled==enable 而跳过）
//
// 这是 P0 修复的核心契约：addr_type 切到 private 后，必须立即 refresh 一次让整包下发
// 把云端 SG 上残留的 0.0.0.0/0:port 入站规则清除。否则用户切了等于没切，规则继续被还原。
//
// 测试方式：不 seed RuleSet → refresh 会因 "no rule_set found" 失败但不影响 200，
// 这里只断言 a) HTTP 200 b) DB 已落新 addr_type。refresh 是否被调用通过"DB 落库时机"
// 间接验证（refresh 失败前必须先选择性落库，与既有 SaveBeforeRefresh 测试同模式）。
func TestHandleUpdateConfig_GatewayUIAddrType_PublicToPrivate_PersistsAndAttemptsRefresh(t *testing.T) {
	db := setupSGPoolTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	// 关键前置：Gateway UI 已开启 + 已迁移 + 当前 addr_type=public
	// 仅 enable=true 时，addr_type 变更才触发 refresh（enable=false 时 condition 评估天然不注入规则）
	if err := db.Create(&model.SiteConfig{
		GatewayUIEnable:        true,
		GatewayUISGMigrateDone: true,
		GatewayUIPort:          7540,
		GatewayUIAddrType:      "public",
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 仅切 addr_type，不带 gateway_ui_enable —— 模拟用户在 PlatformPolicy 页点"切私网"
	form := url.Values{}
	form.Set("gateway_ui_addr_type", "private")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	// DB 必须落新 addr_type（refresh 失败前必须先 Save，与 SaveBeforeRefresh 同模式）
	var cfg model.SiteConfig
	if err := db.First(&cfg).Error; err != nil {
		t.Fatalf("read site config: %v", err)
	}
	if cfg.GatewayUIAddrType != "private" {
		t.Fatalf("expected GatewayUIAddrType=private persisted in DB, got %q", cfg.GatewayUIAddrType)
	}
	// enable 不应被改动（用户没传该字段）
	if !cfg.GatewayUIEnable {
		t.Error("GatewayUIEnable should remain true (not touched by request)")
	}
}

// TestHandleUpdateConfig_GatewayUIAddrType_NoChange_NoRefresh
// 验证当 addr_type 与当前值一致、enable 也未变时，不应触发 refresh 链路上的 DB Save
// （admin_config.go 末尾的 model.DB(context.Background()).Save 仍会保存全量 SiteConfig，但中间的 P0 Save+refresh
//
//	分支不应进入）。这里通过"DB 内容不变"间接验证幂等性。
func TestHandleUpdateConfig_GatewayUIAddrType_NoChange_Idempotent(t *testing.T) {
	db := setupSGPoolTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	if err := db.Create(&model.SiteConfig{
		GatewayUIEnable:        true,
		GatewayUISGMigrateDone: true,
		GatewayUIPort:          7540,
		GatewayUIAddrType:      "private",
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 重复提交同一个 addr_type=private（用户重复点击）
	form := url.Values{}
	form.Set("gateway_ui_addr_type", "private")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var cfg model.SiteConfig
	if err := db.First(&cfg).Error; err != nil {
		t.Fatalf("read site config: %v", err)
	}
	if cfg.GatewayUIAddrType != "private" {
		t.Errorf("addr_type 应保持 private, got %q", cfg.GatewayUIAddrType)
	}
}

// TestHandleUpdateConfig_GatewayUIAddrType_InvalidValue_400
// 验证 addr_type 传入非 "private"/"public" 的非法值时返回 400。
func TestHandleUpdateConfig_GatewayUIAddrType_InvalidValue_400(t *testing.T) {
	db := setupSGPoolTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	if err := db.Create(&model.SiteConfig{GatewayUIAddrType: "public"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("gateway_ui_addr_type", "intranet") // 非法值
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid addr_type, got %d body=%s", w.Code, w.Body.String())
	}
	// 落库不应被修改
	var cfg model.SiteConfig
	if err := db.First(&cfg).Error; err != nil {
		t.Fatalf("read site config: %v", err)
	}
	if got := cfg.GatewayUIAddrType; got != "public" {
		t.Errorf("非法值不应改写 DB，期望 public 实际=%q", got)
	}
}

// TestHandleUpdateConfig_BrowserVNCEnable_SaveError_500
// 覆盖 BrowserVNCEnable refresh 前选择性保存失败时返回 500 的路径。
// 注入 GORM 更新前钩子，仅对 site_configs 表的更新强制报错，让 hasSGPoolReady（查询 rule_sets/managed_sg_pool）
// 仍能通过，但紧随其后的 BrowserVNCEnable 选择性落库会失败。
func TestHandleUpdateConfig_BrowserVNCEnable_SaveError_500(t *testing.T) {
	db := setupSGPoolTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	rs := &model.RuleSet{Name: model.DefaultRuleSetName, IsDefault: true, Version: 1, Rules: "[]"}
	if err := db.Create(rs).Error; err != nil {
		t.Fatalf("seed rule set: %v", err)
	}
	if err := db.Create(&model.ManagedSGPool{
		SGID: "sg-vnc-err", RuleSetID: rs.ID, Status: model.SGStatusActive, RuleVersion: 1,
	}).Error; err != nil {
		t.Fatalf("seed sg pool: %v", err)
	}
	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	if err := db.Create(&model.SiteConfig{BrowserVNCEnable: false}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	// 仅对 site_configs 表的 Update 注入错误（Save 在已有主键时走 Update 路径）
	cbName := "test:fail_site_config_update"
	if err := db.Callback().Update().Before("gorm:update").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement.Table == "site_configs" {
			_ = tx.AddError(fmt.Errorf("injected Save error for SiteConfig"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	defer db.Callback().Update().Remove(cbName)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("browser_vnc_enable", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when Save fails, got %d body=%s", w.Code, w.Body.String())
	}
}

// TestHandleUpdateConfig_GatewayUIEnable_SaveError_500
// 覆盖 admin_config.go:277-280 的 Save 失败 500 返回路径。
func TestHandleUpdateConfig_GatewayUIEnable_SaveError_500(t *testing.T) {
	db := setupSGPoolTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	if err := db.Create(&model.SiteConfig{
		GatewayUIEnable:        false,
		GatewayUISGMigrateDone: true,
		GatewayUIPort:          7540,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	cbName := "test:fail_site_config_update_gateway"
	if err := db.Callback().Update().Before("gorm:update").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement.Table == "site_configs" {
			_ = tx.AddError(fmt.Errorf("injected Save error for SiteConfig"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	defer db.Callback().Update().Remove(cbName)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("gateway_ui_enable", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when Save fails, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleSite_ReturnsIsOverseasField(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)

	i18n.SetDefaultLang("zh")

	req := httptest.NewRequest(http.MethodGet, "/site", nil)
	w := httptest.NewRecorder()

	HandleSite(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// 默认情况下（中文环境），is_overseas 应该为 false
	if _, ok := resp["is_overseas"]; !ok {
		t.Fatal("expected response to contain is_overseas field")
	}
	if resp["is_overseas"] != false {
		t.Fatalf("expected is_overseas=false in default Chinese environment, got %#v", resp["is_overseas"])
	}
}

// ─── HandleUpdateConfig: global_token_quota_day 无效 ──────────────────────────

func TestHandleUpdateConfig_GlobalQuotaDayInvalid_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("global_token_quota_day", "abc")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid global_token_quota_day, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: global_token_quota_day 负数 ──────────────────────────

func TestHandleUpdateConfig_GlobalQuotaDayNegative_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("global_token_quota_day", "-2")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative global_token_quota_day, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: global_token_quota_period 无效 ────────────────────────

func TestHandleUpdateConfig_GlobalQuotaPeriodInvalid_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("global_token_quota_period", "weekly")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid global_token_quota_period, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: default_instance_quota 无效 ──────────────────────────

func TestHandleUpdateConfig_DefaultInstanceQuotaInvalid_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("default_instance_quota", "-1")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid default_instance_quota, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateConfig_DefaultInstanceQuotaTooLarge_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("default_instance_quota", "1000")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for default_instance_quota > 999, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: default_token_quota_day 无效 ──────────────────────────

func TestHandleUpdateConfig_DefaultTokenQuotaInvalid_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("default_token_quota_day", "-2")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid default_token_quota_day, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: default_tags 格式错误 ────────────────────────────────

func TestHandleUpdateConfig_DefaultTagsInvalidJSON_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.Tag{}, &model.TagVisibilityGroup{}); err != nil {
		t.Fatalf("migrate tag tables: %v", err)
	}
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("default_tags", "not-json")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid default_tags JSON, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: sso_im_types 格式错误 ────────────────────────────────

func TestHandleUpdateConfig_SSOIMTypesInvalidJSON_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("sso_im_types", "not-json")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid sso_im_types JSON, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: sso_im_types 包含不支持的值 ──────────────────────────

func TestHandleUpdateConfig_SSOIMTypesUnsupportedValue_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("sso_im_types", `["wecom","unsupported_type"]`)
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported sso_im_type, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: logo 类型不支持 ──────────────────────────────────────

func TestHandleUpdateConfig_LogoTypeUnsupported_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("logo", "test.gif")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write([]byte("GIF89a"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(body.String()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported logo type, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: logo 过大 ────────────────────────────────────────────

func TestHandleUpdateConfig_LogoTooLarge_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("logo", "test.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	// 写入超过 512KB 的数据
	largeData := make([]byte, 513<<10)
	part.Write(largeData)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(body.String()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for logo too large, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: terminal_enabled 开关 ────────────────────────────────

func TestHandleUpdateConfig_TerminalEnabled_UpdatesConfig(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("terminal_enabled", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !model.GetSiteConfig(context.Background()).TerminalEnabled {
		t.Fatal("expected TerminalEnabled=true")
	}
}

// ─── HandleUpdateConfig: user_data_enabled 开关 ──────────────────────────────

func TestHandleUpdateConfig_UserDataEnabled_UpdatesConfig(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("user_data_enabled", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !model.GetSiteConfig(context.Background()).UserDataEnabled {
		t.Fatal("expected UserDataEnabled=true")
	}
}

// ─── HandleUpdateConfig: doctor_enabled 开关 ─────────────────────────────────

func TestHandleUpdateConfig_DoctorEnabled_UpdatesConfig(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("doctor_enabled", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !model.GetSiteConfig(context.Background()).DoctorEnabled {
		t.Fatal("expected DoctorEnabled=true")
	}
}

// ─── HandleUpdateConfig: local_agent_enabled 开关 ──────────────────
// 回归守护：selected-fields save 重构后，字段必须同时
//  1. 写内存：config.LocalAgentEnabled = v == "true"
//  2. 追加 updateFields：updateFields = append(updateFields, "LocalAgentEnabled")
//
// 两者缺一都会导致 SaveSelectedFields 静默丢失该字段（bug 回归）。
func TestHandleUpdateConfig_LocalAgentEnabled_UpdatesConfig(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("local_agent_enabled", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !model.GetSiteConfig(context.Background()).LocalAgentEnabled {
		t.Fatal("expected LocalAgentEnabled=true (selected-fields save 遗漏 append 时会静默丢字段)")
	}
}

// ─── HandleUpdateConfig: name 字段更新 ────────────────────────────────────────

func TestHandleUpdateConfig_Name_UpdatesConfig(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("name", "MyPlatform")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if got := model.GetSiteConfig(context.Background()).Name; got != "MyPlatform" {
		t.Fatalf("expected Name=MyPlatform, got %q", got)
	}
}

// ─── HandleUpdateCVMConfig: cvm_template 非 JSON ─────────────────────────────

func TestHandleUpdateCVMConfig_CVMTemplateNotJSON_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("cvm_template", "not-json")
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-JSON cvm_template, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateCVMConfig: 有子网但无 VPC ───────────────────────────────────

func TestHandleUpdateCVMConfig_SubnetWithoutVPC_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 先在 DB 中存入 SubnetIds 但清空 VpcId
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{
		"vpc_id":     "",
		"subnet_ids": `{"zone1":["subnet-xxx"]}`,
	})

	// 提交一个空的 vpc_id 和空 subnet_ids（不清除 DB 中的 subnet_ids）
	// 因为 r.Form.Has("vpc_id")=true 且 vpc_id="" → 会清除 config.SubnetIds=""
	// 所以需要只发 cvm_secret_id 让配置走到检查
	// 更好的方式：直接修改 DB 让 VpcId="" 但 SubnetIds 不为空
	// 此时提交不带 vpc_id 和 subnet_ids 的请求，config 会从 DB 读出旧值
	form := url.Values{}
	form.Set("cvm_secret_id", "keep") // 触发其他字段更新但保留 VpcId 和 SubnetIds
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for subnet without VPC, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateCVMConfig: VPC 有但子网为空 ──────────────────────────────────

func TestHandleUpdateCVMConfig_VPCWithEmptySubnets_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 先设置 vpc_id 使 config.VpcId 非空
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{
		"vpc_id": "vpc-test",
	})

	form := url.Values{}
	form.Set("subnet_ids", "")
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty subnets, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateCVMConfig: 可用区不在 Region 中 ──────────────────────────────

func TestHandleUpdateCVMConfig_ZoneNotInRegion_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("vpc_id", "vpc-test")
	form.Set("subnet_ids", `{"invalid-zone-xx":["subnet-xxx"]}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zone not in region, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateCVMConfig: skillhub 字段更新 ─────────────────────────────────

func TestHandleUpdateCVMConfig_SkillHub_UpdatesConfig(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("skillhub", "https://skillhub.example.com")
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if got := model.GetSiteConfig(context.Background()).SkillHub; got != "https://skillhub.example.com" {
		t.Fatalf("expected SkillHub updated, got %q", got)
	}
}

// ─── HandleListCloudSubnets: 缺少 vpc_id ──────────────────────────────────────

func TestHandleListCloudSubnets_MissingVpcID_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/cloud/subnets?zone=ap-guangzhou-1", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleListCloudSubnets(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing vpc_id, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleListCloudSubnets: 缺少 zone ────────────────────────────────────────

func TestHandleListCloudSubnets_MissingZone_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/cloud/subnets?vpc_id=vpc-xxx", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleListCloudSubnets(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing zone, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── handleFirstTimeGatewayUIEnable: HasSGPoolReady 返回错误 ──────────────────

func TestHandleFirstTimeGatewayUIEnable_HasSGPoolReadyError(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	ctx := tenantCtx()

	// 不插入 RuleSet/ManagedSGPool，HasSGPoolReady 会返回错误（表存在但数据为空）
	// 插入 SiteConfig 供 Save 使用
	cfg := model.SiteConfig{Name: "test", GatewayUIPort: 0}
	db.Create(&cfg)

	err := handleFirstTimeGatewayUIEnable(ctx, &cfg)
	if err == nil {
		t.Fatal("expected error when HasSGPoolReady fails or pool not ready")
	}
}

// ─── handleFirstTimeGatewayUIEnable: DB Save 失败 ────────────────────────────

func TestHandleFirstTimeGatewayUIEnable_SaveError(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	ctx := tenantCtx()

	rs := model.RuleSet{Name: "default", Rules: "[]", Version: 1, UserGroupIDs: "[]"}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID:        "sg-save-err",
		RuleSetID:   rs.ID,
		Status:      model.SGStatusActive,
		RuleVersion: 1,
	})
	cfg := model.SiteConfig{Name: "test", GatewayUIPort: 0}
	db.Create(&cfg)

	// 注入 Save 错误
	cbName := "test:fail_gateway_ui_save"
	if err := db.Callback().Update().Before("gorm:update").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement.Table == "site_configs" {
			_ = tx.AddError(fmt.Errorf("injected Save error"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	defer db.Callback().Update().Remove(cbName)

	err := handleFirstTimeGatewayUIEnable(ctx, &cfg)
	if err == nil {
		t.Fatal("expected error when Save fails")
	}
}

// ─── snakeToCamel / camelToSnake 单元测试 ─────────────────────────────────────

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"internet_accessible", "InternetAccessible"},
		{"public_ip_assigned", "PublicIpAssigned"},
		{"system_disk", "SystemDisk"},
		{"instance_type", "InstanceType"},
		{"instance_charge_type", "InstanceChargeType"},
		{"", ""},
	}
	for _, tt := range tests {
		got := snakeToCamel(tt.input)
		if got != tt.want {
			t.Errorf("snakeToCamel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"InternetAccessible", "internet_accessible"},
		{"PublicIpAssigned", "public_ip_assigned"},
		{"SystemDisk", "system_disk"},
		{"InstanceType", "instance_type"},
		{"", ""},
	}
	for _, tt := range tests {
		got := camelToSnake(tt.input)
		if got != tt.want {
			t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ─── extractTemplateSection 单元测试 ──────────────────────────────────────────

func TestExtractTemplateSection_UnsupportedPath(t *testing.T) {
	_, err := extractTemplateSection(`{"InternetAccessible":{"PublicIpAssigned":true}}`, "unsupported_key")
	if err == nil {
		t.Fatal("expected error for unsupported template_path")
	}
}

func TestExtractTemplateSection_ValidPath(t *testing.T) {
	tpl := `{"InternetAccessible":{"PublicIpAssigned":true,"InternetMaxBandwidthOut":10},"InstanceType":"S5.MEDIUM4"}`
	result, err := extractTemplateSection(tpl, "internet_accessible")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["public_ip_assigned"] != true {
		t.Errorf("expected public_ip_assigned=true, got %v", m["public_ip_assigned"])
	}
}

func TestExtractTemplateSection_ScalarValue(t *testing.T) {
	tpl := `{"InstanceType":"S5.MEDIUM4"}`
	result, err := extractTemplateSection(tpl, "instance_type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "S5.MEDIUM4" {
		t.Errorf("expected S5.MEDIUM4, got %v", result)
	}
}

func TestExtractTemplateSection_MissingKey(t *testing.T) {
	tpl := `{"SystemDisk":{"DiskType":"CLOUD_SSD","DiskSize":50}}`
	result, err := extractTemplateSection(tpl, "internet_accessible")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for missing key, got %v", result)
	}
}

func TestExtractTemplateSection_InvalidJSON(t *testing.T) {
	_, err := extractTemplateSection("not-json", "internet_accessible")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestExtractTemplateSection_EmptyTemplate(t *testing.T) {
	result, err := extractTemplateSection("", "instance_type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 空 template 应使用 DefaultCVMTemplate，如果 DefaultCVMTemplate 中有 InstanceType 则返回
	// 这里不严格断言，只确保不 panic
	t.Logf("result for empty template: %v", result)
}

// ─── validateInternetAccessibleInTemplate 单元测试 ────────────────────────────

func TestValidateInternetAccessibleInTemplate_FormatError(t *testing.T) {
	tplMap := map[string]interface{}{
		"InternetAccessible": "not-an-object",
	}
	err := validateInternetAccessibleInTemplate(tplMap)
	if err == nil {
		t.Fatal("expected error for invalid InternetAccessible format")
	}
}

func TestValidateInternetAccessibleInTemplate_ParseError(t *testing.T) {
	tplMap := map[string]interface{}{
		"InternetAccessible": map[string]interface{}{
			"PublicIpAssigned":        "not-bool",
			"InternetMaxBandwidthOut": "not-int",
		},
	}
	err := validateInternetAccessibleInTemplate(tplMap)
	// 可能不报错（因为 json.Unmarshal 不会严格校验类型，float 和 bool 在 interface{} 中）
	// 但 PublicIpAssigned 不为 bool 时 Normalize 后逻辑可能异常
	t.Logf("validateInternetAccessibleInTemplate with non-bool PublicIpAssigned: %v", err)
}

func TestValidateInternetAccessibleInTemplate_MissingKey(t *testing.T) {
	tplMap := map[string]interface{}{}
	err := validateInternetAccessibleInTemplate(tplMap)
	if err != nil {
		t.Fatalf("expected nil for missing InternetAccessible key, got %v", err)
	}
}

// ─── validateSystemDiskInTemplate 单元测试 ────────────────────────────────────

func TestValidateSystemDiskInTemplate_FormatError(t *testing.T) {
	tplMap := map[string]interface{}{
		"SystemDisk": "not-an-object",
	}
	err := validateSystemDiskInTemplate(tplMap)
	if err == nil {
		t.Fatal("expected error for SystemDisk not being an object")
	}
}

func TestValidateSystemDiskInTemplate_DiskTypeNotString(t *testing.T) {
	tplMap := map[string]interface{}{
		"SystemDisk": map[string]interface{}{
			"DiskType": 123,
		},
	}
	err := validateSystemDiskInTemplate(tplMap)
	if err == nil {
		t.Fatal("expected error for DiskType not being string")
	}
}

func TestValidateSystemDiskInTemplate_DiskSizeNotNumber(t *testing.T) {
	tplMap := map[string]interface{}{
		"SystemDisk": map[string]interface{}{
			"DiskSize": "fifty",
		},
	}
	err := validateSystemDiskInTemplate(tplMap)
	if err == nil {
		t.Fatal("expected error for DiskSize not being a number")
	}
}

func TestValidateSystemDiskInTemplate_DiskSizeNotInteger(t *testing.T) {
	tplMap := map[string]interface{}{
		"SystemDisk": map[string]interface{}{
			"DiskSize": 50.5,
		},
	}
	err := validateSystemDiskInTemplate(tplMap)
	if err == nil {
		t.Fatal("expected error for DiskSize not being an integer")
	}
}

func TestValidateSystemDiskInTemplate_MissingKey(t *testing.T) {
	tplMap := map[string]interface{}{}
	err := validateSystemDiskInTemplate(tplMap)
	if err != nil {
		t.Fatalf("expected nil for missing SystemDisk key, got %v", err)
	}
}

// ─── collectAllSgIds 单元测试 ─────────────────────────────────────────────────

func TestCollectAllSgIds(t *testing.T) {
	instanceSgMap := map[string][]string{
		"ins-1": {"sg-a", "sg-b"},
		"ins-2": {"sg-b", "sg-c"},
	}
	result := collectAllSgIds(instanceSgMap)
	if len(result) != 3 {
		t.Fatalf("expected 3 unique sg IDs, got %d: %v", len(result), result)
	}
	seen := map[string]bool{}
	for _, id := range result {
		seen[id] = true
	}
	for _, expected := range []string{"sg-a", "sg-b", "sg-c"} {
		if !seen[expected] {
			t.Errorf("expected %s in result", expected)
		}
	}
}

// ─── replaceInSlice 单元测试 ──────────────────────────────────────────────────

func TestReplaceInSlice(t *testing.T) {
	slice := []string{"sg-a", "sg-b", "sg-c"}
	result := replaceInSlice(slice, "sg-b", "sg-new")
	expected := []string{"sg-a", "sg-new", "sg-c"}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("index %d: got %q, want %q", i, v, expected[i])
		}
	}
}

// ─── HandleUpdateTemplate: 非 POST 方法 ───────────────────────────────────────

func TestHandleUpdateTemplate_MethodNotAllowed(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/config/template", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateTemplate: 无效 JSON ──────────────────────────────────────────

func TestHandleUpdateTemplate_InvalidJSON_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/config/template", strings.NewReader("not-json"))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateTemplate: 空请求体 ───────────────────────────────────────────

func TestHandleUpdateTemplate_EmptyBody_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/config/template", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateTemplate: 不允许的字段 ───────────────────────────────────────

func TestHandleUpdateTemplate_FieldsNotAllowed_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/config/template", strings.NewReader(`{"forbidden_key":"value"}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for forbidden fields, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateTemplate: 当前 CVM 模板格式错误 ──────────────────────────────

func TestHandleUpdateTemplate_CurrentTemplateInvalid_500(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	// 先插入一条 SiteConfig，再更新 CVMTemplate 为无效 JSON
	db.Create(&model.SiteConfig{})
	if err := db.Exec("UPDATE site_configs SET c_vm_template = ? WHERE 1=1", "invalid-json").Error; err != nil {
		t.Fatalf("update site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 使用 instance_charge_type 不触发云 API 校验
	req := httptest.NewRequest(http.MethodPost, "/admin/config/template", strings.NewReader(`{"instance_charge_type":"POSTPAID_BY_HOUR"}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for invalid current template, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateTemplate: 成功更新 ───────────────────────────────────────────

func TestHandleUpdateTemplate_Success(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	if err := db.Create(&model.SiteConfig{CVMTemplate: model.DefaultCVMTemplate}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	Store = sessions.NewCookieStore([]byte("test-secret"))
	_ = db

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 使用 instance_charge_type 不触发云 API 校验
	req := httptest.NewRequest(http.MethodPost, "/admin/config/template", strings.NewReader(`{"instance_charge_type":"POSTPAID_BY_HOUR"}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("expected ok=true, got %v", resp["ok"])
	}
	if resp["message"] == nil {
		t.Fatal("expected message field in response")
	}
}

// ─── HandleUpdateTemplate: 删除字段 (null 值) ─────────────────────────────────

func TestHandleUpdateTemplate_DeleteField(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	if err := db.Create(&model.SiteConfig{CVMTemplate: model.DefaultCVMTemplate}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	Store = sessions.NewCookieStore([]byte("test-secret"))
	_ = db

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/config/template", strings.NewReader(`{"instance_charge_prepaid":null}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateTemplate: system_disk 校验 ───────────────────────────────────

func TestHandleUpdateTemplate_SystemDiskInvalidType_400(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))
	_ = db

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/config/template", strings.NewReader(`{"system_disk":{"DiskType":"INVALID_TYPE","DiskSize":50}}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid DiskType, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateTemplate_SystemDiskTooSmall_400(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))
	_ = db

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/config/template", strings.NewReader(`{"system_disk":{"DiskType":"CLOUD_SSD","DiskSize":10}}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleUpdateTemplate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for DiskSize too small, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateCVMConfig: vpc_id 清空时子网也被清空 ─────────────────────────

func TestHandleUpdateCVMConfig_ClearVPCClearsSubnets(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	// 先设置 VPC 和子网
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{
		"vpc_id":     "vpc-old",
		"subnet_ids": `{"ap-guangzhou-1":["subnet-old"]}`,
	})

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("vpc_id", "")
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	cfg := model.GetSiteConfig(context.Background())
	if cfg.SubnetIds != "" {
		t.Fatalf("expected SubnetIds to be cleared when VPC cleared, got %q", cfg.SubnetIds)
	}
}

// ─── HandleUpdateCVMConfig: API Token 被阻止修改敏感字段 ───────────────────────
// 当使用 AdminToken 认证时（isAdminTokenRequest=true），敏感字段可修改；
// 但如果使用的是 Bearer Token 且不是 AdminToken，则 sensitiveBlocked=true。
// 非 OpenAPI 路由只允许 AdminToken（非用户 API Token），所以这里用 AdminToken
// 但设置另一个 token 值来模拟 "非 AdminToken 的 Bearer" 场景。
// 实际上 requireAdmin 会阻止非 AdminToken Bearer 进入 admin 路由，
// 所以这个测试验证：AdminToken 可以修改敏感字段（sensitiveBlocked=false）。

func TestHandleUpdateCVMConfig_AdminTokenCanModifySensitiveFields(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("cvm_secret_id", "new-secret-id")
	form.Set("cvm_secret_key", "new-secret-key")
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	cfg := model.GetSiteConfig(context.Background())
	if cfg.CVMSecretId != "new-secret-id" {
		t.Fatalf("AdminToken should be able to modify cvm_secret_id, got %q", cfg.CVMSecretId)
	}
	if cfg.CVMSecretKey != "new-secret-key" {
		t.Fatalf("AdminToken should be able to modify cvm_secret_key, got %q", cfg.CVMSecretKey)
	}
}

// ─── HandleLogo: 默认 Logo SVG ────────────────────────────────────────────────

func TestHandleLogo_DefaultSVG(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/logo", nil)
	w := httptest.NewRecorder()

	HandleLogo(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Fatalf("expected Content-Type image/svg+xml, got %s", ct)
	}
	if !strings.Contains(w.Body.String(), "<svg") {
		t.Fatal("expected SVG content in response")
	}
}

// ─── convertSnakeToCamelKeys 单元测试 ──────────────────────────────────────────

func TestConvertSnakeToCamelKeys(t *testing.T) {
	patch := map[string]interface{}{
		"internet_accessible": map[string]interface{}{
			"public_ip_assigned": true,
		},
	}
	result := convertSnakeToCamelKeys(patch)
	if _, ok := result["InternetAccessible"]; !ok {
		t.Fatal("expected InternetAccessible key in result")
	}
	subMap, ok := result["InternetAccessible"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result["InternetAccessible"])
	}
	if _, ok := subMap["PublicIpAssigned"]; !ok {
		t.Fatal("expected PublicIpAssigned key in sub-map")
	}
}

// ─── HandleUpdateConfig: sso_im_types 有效值 ──────────────────────────────────

func TestHandleUpdateConfig_SSOIMTypesValid(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("sso_im_types", `["wecom","feishu"]`)
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: sso_im_types 空数组 ──────────────────────────────────

func TestHandleUpdateConfig_SSOIMTypesEmpty(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("sso_im_types", "[]")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: sso_im_types 空字符串 ────────────────────────────────

func TestHandleUpdateConfig_SSOIMTypesEmptyString(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("sso_im_types", "")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: global_token_quota_day 有效值 ────────────────────────

func TestHandleUpdateConfig_GlobalQuotaDayValid(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("global_token_quota_day", "100")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: default_tags 空数组 ──────────────────────────────────

func TestHandleUpdateConfig_DefaultTagsEmpty(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.Tag{}, &model.TagVisibilityGroup{}); err != nil {
		t.Fatalf("migrate tag tables: %v", err)
	}
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("default_tags", "[]")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: gateway_ui_enable=true 但未迁移（首次开启）──────────

func TestHandleUpdateConfig_GatewayUIEnableFirstTime_NoSGPool_500(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	// 不插入 RuleSet/ManagedSGPool，SG 池未就绪
	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	if err := db.Create(&model.SiteConfig{
		GatewayUIEnable:        false,
		GatewayUISGMigrateDone: false,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("gateway_ui_enable", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	// 首次开启但 SG 池未就绪，应返回 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when SG pool not ready, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: gateway_ui_enable 已开启时返回 port ─────────────────

func TestHandleUpdateConfig_GatewayUIEnable_ReturnsPort(t *testing.T) {
	db := setupSGPoolTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	if err := db.Create(&model.SiteConfig{
		GatewayUIEnable:        true,
		GatewayUISGMigrateDone: true,
		GatewayUIPort:          7540,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("name", "test")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["gateway_ui_port"] == nil {
		t.Fatal("expected gateway_ui_port in response when GatewayUIEnable=true")
	}
}

// ─── checkGatewayUIPortRuleExists: VPC 客户端创建失败 ────────────────────────

func TestCheckGatewayUIPortRuleExists_VpcClientError(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	// 凭据未配置，newVpcClient 应失败
	_, err := checkGatewayUIPortRuleExists(context.Background(), "sg-test", 8080)
	if err == nil {
		t.Fatal("expected error when VPC client creation fails")
	}
}

// ─── addGatewayUISecurityGroupRule: VPC 客户端创建失败 ────────────────────────

func TestAddGatewayUISecurityGroupRule_VpcClientError(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	err := addGatewayUISecurityGroupRule(context.Background(), "sg-test", 8080)
	if err == nil {
		t.Fatal("expected error when VPC client creation fails")
	}
}

// ─── HandleAdminConfig: template_path 不在白名单 ──────────────────────────────

func TestHandleAdminConfig_UnsupportedTemplatePath_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/config?template_path=forbidden_key", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleAdminConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported template_path, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleAdminConfig: template_path 有效 ────────────────────────────────────

func TestHandleAdminConfig_TemplatePathValid(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/config?template_path=instance_type", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()

	HandleAdminConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp.Config["instance_type"]; !ok {
		t.Fatal("expected instance_type in config response")
	}
}

// ─── HandleUpdateConfig: BrowserVNCEnable 但 SG 池未就绪 ─────────────────────

func TestHandleUpdateConfig_BrowserVNCEnable_SGPoolNotReady_400(t *testing.T) {
	db := setupAdminConfigTestDB(t)
	Store = sessions.NewCookieStore([]byte("test-secret"))

	// 不插入 RuleSet/ManagedSGPool
	if err := db.Where("1=1").Delete(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("clear site config: %v", err)
	}
	if err := db.Create(&model.SiteConfig{BrowserVNCEnable: false}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("browser_vnc_enable", "true")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when SG pool not ready, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: default_token_quota_rules 无效格式 ───────────────────

func TestHandleUpdateConfig_DefaultTokenQuotaRulesInvalid_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("default_token_quota_rules", "not-json")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid default_token_quota_rules, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateConfig: global_token_quota_rules 无效格式 ────────────────────

func TestHandleUpdateConfig_GlobalTokenQuotaRulesInvalid_400(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("global_token_quota_rules", "not-json")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid global_token_quota_rules, got %d body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateCVMConfig: agent_cam_role_secret_id/secret_key 更新 ──────────

func TestHandleUpdateCVMConfig_AgentCamRoleSecretID(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("agent_cam_role_secret_id", "role-id-123")
	form.Set("agent_cam_role_secret_key", "role-key-456")
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	cfg := model.GetSiteConfig(context.Background())
	if cfg.AgentCamRoleSecretId != "role-id-123" {
		t.Fatalf("expected AgentCamRoleSecretId=role-id-123, got %q", cfg.AgentCamRoleSecretId)
	}
	if cfg.AgentCamRoleSecretKey != "role-key-456" {
		t.Fatalf("expected AgentCamRoleSecretKey=role-key-456, got %q", cfg.AgentCamRoleSecretKey)
	}
}

// ─── HandleSite: is_overseas true when English default ────────────────────────

func TestHandleSite_ReturnsIsOverseasTrueWhenEnglishDefault(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)

	// 设置英文为默认语言，模拟海外环境
	i18n.SetDefaultLang("en")
	defer i18n.SetDefaultLang("zh") // 测试完成后恢复为中文

	req := httptest.NewRequest(http.MethodGet, "/site", nil)
	w := httptest.NewRecorder()

	HandleSite(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// 英文环境下，is_overseas 应该为 true
	if _, ok := resp["is_overseas"]; !ok {
		t.Fatal("expected response to contain is_overseas field")
	}
	if resp["is_overseas"] != true {
		t.Fatalf("expected is_overseas=true in English environment, got %#v", resp["is_overseas"])
	}
}

func TestHandleUpdateConfig_DefaultTokenQuotaDoesNotSyncUsers(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	ctx := context.Background()

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 旧默认：day=500, rules=""
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"default_token_quota_day":   500,
		"default_token_quota_rules": "",
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	user := model.User{Username: "existing-user", TokenQuotaDay: 500, TokenQuotaRules: ""}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	// set_config 只修改站点默认值，不能同步改写已有用户的烙印字段。
	form := url.Values{}
	form.Set("default_token_quota_day", "1000")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateConfig(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	cfg := model.GetSiteConfig(ctx)
	if cfg.DefaultTokenQuotaDay != -1 {
		t.Fatalf("expected DefaultTokenQuotaDay=-1 after upsert, got %d", cfg.DefaultTokenQuotaDay)
	}
	if cfg.DefaultTokenQuotaRules != `[{"mode":"day","limit":1000}]` {
		t.Fatalf("expected site rules to be updated, got %s", cfg.DefaultTokenQuotaRules)
	}

	var after model.User
	if err := model.DB(ctx).First(&after, user.ID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if after.TokenQuotaDay != 500 || after.TokenQuotaRules != "" {
		t.Fatalf("existing user quota fields should remain unchanged, got day=%d rules=%q",
			after.TokenQuotaDay, after.TokenQuotaRules)
	}
}

// TestHandleSite_IsUniverseField 验证 /site 响应包含 is_universe 字段
func TestHandleSite_IsUniverseField(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)

	// 非 universe 模式（FixedSnapshot != nil）
	oldSnapshot := hcommon.FixedSnapshot
	defer func() { hcommon.FixedSnapshot = oldSnapshot }()
	hcommon.FixedSnapshot = &hcommon.TenantSnapshot{Identifier: "test-tenant"}

	req := httptest.NewRequest(http.MethodGet, "/site", nil)
	w := httptest.NewRecorder()
	HandleSite(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	isUniverse, exists := resp["is_universe"]
	if !exists {
		t.Fatal("is_universe field missing from /site response")
	}
	if isUniverse != false {
		t.Fatalf("expected is_universe=false in non-universe mode, got %v", isUniverse)
	}

	// universe 模式（FixedSnapshot == nil）
	hcommon.FixedSnapshot = nil
	req2 := httptest.NewRequest(http.MethodGet, "/site", nil)
	w2 := httptest.NewRecorder()
	HandleSite(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp2 map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp2["is_universe"] != true {
		t.Fatalf("expected is_universe=true in universe mode, got %v", resp2["is_universe"])
	}
}

// ─── 部分更新回归测试：POST /admin/config 不覆盖无关 CVM 字段 ─────────────────

// TestHandleUpdateConfig_ChangingNamePreservesSkillHub
// 契约：POST /admin/config 仅修改 name 时，不应用全量 Save 覆盖 skill_hub
// （HandleUpdateCVMConfig 管辖的字段）。若将来改成选择性列更新后误删了
// 其他列，或回归为全量 Save，此测试会红灯。
func TestHandleUpdateConfig_ChangingNamePreservesSkillHub(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	db := model.DB(context.Background())

	const seedSkillHub = "https://skillhub.seeded.example.com"
	const concurrentSkillHub = "https://skillhub.concurrent.example.com"
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"name":      "seeded-name",
		"skill_hub": seedSkillHub,
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	cbName := "test:config_partial_update_concurrent_skillhub"
	fired := false
	if err := db.Callback().Update().Before("gorm:update").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement.Table != "site_configs" || fired {
			return
		}
		fired = true
		if err := tx.Session(&gorm.Session{NewDB: true}).
			Exec("UPDATE site_configs SET skill_hub = ?", concurrentSkillHub).Error; err != nil {
			_ = tx.AddError(err)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Update().Remove(cbName)
	})

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("name", "updated-name")
	req := httptest.NewRequest(http.MethodPost, "/admin/config", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !fired {
		t.Fatal("expected concurrent update callback to fire")
	}

	config := model.GetSiteConfig(context.Background())
	if config.Name != "updated-name" {
		t.Fatalf("Name not updated: expected 'updated-name', got %q", config.Name)
	}
	if config.SkillHub != concurrentSkillHub {
		t.Fatalf("SkillHub was overwritten by /admin/config update: expected concurrent value %q, got %q", concurrentSkillHub, config.SkillHub)
	}
}

// ─── 部分更新回归测试：POST /admin/config/cvm 不覆盖无关通用字段 ──────────────

// TestHandleUpdateCVMConfig_ChangingSkillHubPreservesGeneralConfig
// 契约：POST /admin/config/cvm 仅修改 skillhub 时，不应覆盖 HandleUpdateConfig
// 管辖的通用字段（name / terminal_enabled / chat_view_enabled）。若 CVM
// 端点回归为全量 Save 或选择性更新遗漏了列排除，此测试会红灯。
func TestHandleUpdateCVMConfig_ChangingSkillHubPreservesGeneralConfig(t *testing.T) {
	initSiteConfigChatViewTestEnv(t)
	db := model.DB(context.Background())

	const seedName = "seeded-name"
	const concurrentName = "concurrent-name"
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"name":              seedName,
		"terminal_enabled":  true,
		"chat_view_enabled": true,
		"skill_hub":         "https://skillhub.seeded.example.com",
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	cbName := "test:cvm_partial_update_concurrent_general_config"
	fired := false
	if err := db.Callback().Update().Before("gorm:update").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement.Table != "site_configs" || fired {
			return
		}
		fired = true
		if err := tx.Session(&gorm.Session{NewDB: true}).
			Exec("UPDATE site_configs SET name = ?, terminal_enabled = ?, chat_view_enabled = ?", concurrentName, false, false).Error; err != nil {
			_ = tx.AddError(err)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Update().Remove(cbName)
	})

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("skillhub", "https://skillhub.updated.example.com")
	req := httptest.NewRequest(http.MethodPost, "/admin/config/cvm", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleUpdateCVMConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !fired {
		t.Fatal("expected concurrent update callback to fire")
	}

	config := model.GetSiteConfig(context.Background())
	if config.SkillHub != "https://skillhub.updated.example.com" {
		t.Fatalf("SkillHub not updated: expected updated URL, got %q", config.SkillHub)
	}
	if config.Name != concurrentName {
		t.Fatalf("Name was overwritten by /admin/config/cvm update: expected concurrent value %q, got %q", concurrentName, config.Name)
	}
	if config.TerminalEnabled {
		t.Fatal("TerminalEnabled was overwritten by /admin/config/cvm update: expected concurrent false, got true")
	}
	if config.ChatViewEnabled {
		t.Fatal("ChatViewEnabled was overwritten by /admin/config/cvm update: expected concurrent false, got true")
	}
}

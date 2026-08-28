package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"hatchery/common"
	"hatchery/model"
)

// setupAdminTenantTestDB 初始化 admin_tenant handler 测试所需的内存 SQLite。
func setupAdminTenantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	f, err := os.CreateTemp("", "test-admin-tenant-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()
	db, err := gorm.Open(sqlite.Open(f.Name()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	db.AutoMigrate(
		&model.TenantDomain{},
		&model.SiteConfig{},
		&model.User{},
		&model.AIChannel{},
	)
	t.Cleanup(model.UseDBForTest(db))

	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	// 确保非 universe 模式不干扰（handler 内用 SkipIdentifier ctx）
	oldSnap := common.FixedSnapshot
	common.FixedSnapshot = nil
	t.Cleanup(func() { common.FixedSnapshot = oldSnap })

	return db
}

func adminTenantReq(method, url, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	r.Header.Set("Authorization", "Bearer test-admin-token")
	r.Header.Set("Content-Type", "application/json")
	return r
}

// ─── HandleInitTenant ─────────────────────────────────────────────────

func TestHandleInitTenant_MethodNotAllowed(t *testing.T) {
	setupAdminTenantTestDB(t)
	w := httptest.NewRecorder()
	r := adminTenantReq("GET", "/tenants/init", "")
	HandleInitTenant(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleInitTenant_ForbiddenWithoutToken(t *testing.T) {
	setupAdminTenantTestDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/tenants/init", strings.NewReader(`{}`))
	// 无 Authorization header
	HandleInitTenant(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestHandleInitTenant_InvalidJSON(t *testing.T) {
	setupAdminTenantTestDB(t)
	w := httptest.NewRecorder()
	r := adminTenantReq("POST", "/tenants/init", "not-json")
	HandleInitTenant(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleInitTenant_MissingIdentifier(t *testing.T) {
	setupAdminTenantTestDB(t)
	w := httptest.NewRecorder()
	body := `{"domains":["a.example.com"]}`
	r := adminTenantReq("POST", "/tenants/init", body)
	HandleInitTenant(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleInitTenant_MissingDomains(t *testing.T) {
	setupAdminTenantTestDB(t)
	w := httptest.NewRecorder()
	body := `{"identifier":"new-tenant"}`
	r := adminTenantReq("POST", "/tenants/init", body)
	HandleInitTenant(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleInitTenant_Success(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	_ = db

	body := `{
		"identifier": "new-tenant-ok",
		"domains": ["new.example.com"],
		"primary_domain": "https://new.example.com",
		"uin": "100099",
		"init_user": "admin",
		"init_pass": "Admin123!",
		"internal_secret": "sec123",
		"oneid_account_id": "oneid-new",
		"secret_id": "sid",
		"secret_key": "skey"
	}`
	w := httptest.NewRecorder()
	r := adminTenantReq("POST", "/tenants/init", body)
	HandleInitTenant(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Fatalf("expected ok=true, got %v", resp)
	}
	if resp["identifier"] != "new-tenant-ok" {
		t.Fatalf("expected identifier new-tenant-ok, got %v", resp["identifier"])
	}
}

func TestHandleInitTenant_DuplicateIdentifier(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	// 预先创建
	db.Create(&model.SiteConfig{Identifier: "dup-tenant"})

	body := `{
		"identifier": "dup-tenant",
		"domains": ["dup.example.com"],
		"primary_domain": "https://dup.example.com"
	}`
	w := httptest.NewRecorder()
	r := adminTenantReq("POST", "/tenants/init", body)
	HandleInitTenant(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleInitTenant_DuplicateDomain(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	// 预先创建域名映射
	db.Create(&model.TenantDomain{Domain: "taken.example.com", Identifier: "other-tenant"})

	body := `{
		"identifier": "new-tenant-2",
		"domains": ["taken.example.com"],
		"primary_domain": "https://taken.example.com"
	}`
	w := httptest.NewRecorder()
	r := adminTenantReq("POST", "/tenants/init", body)
	HandleInitTenant(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d, body: %s", w.Code, w.Body.String())
	}
}

// ─── HandleTenantDomains (POST) ─────────────────────────────────────────

func TestHandleTenantDomains_AddDomain_Success(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	db.Create(&model.SiteConfig{Identifier: "add-domain-tenant", Domain: "https://main.example.com"})

	body := `{"identifier":"add-domain-tenant","domain":"extra.example.com"}`
	w := httptest.NewRecorder()
	r := adminTenantReq("POST", "/tenants/domains", body)
	HandleTenantDomains(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 验证数据库有记录
	var td model.TenantDomain
	db.Where("domain = ?", "extra.example.com").First(&td)
	if td.Identifier != "add-domain-tenant" {
		t.Fatalf("expected add-domain-tenant, got %q", td.Identifier)
	}
}

func TestHandleTenantDomains_AddDomain_Duplicate(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	db.Create(&model.TenantDomain{Domain: "exists.example.com", Identifier: "t1"})

	body := `{"identifier":"t1","domain":"exists.example.com"}`
	w := httptest.NewRecorder()
	r := adminTenantReq("POST", "/tenants/domains", body)
	HandleTenantDomains(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestHandleTenantDomains_AddDomain_MissingFields(t *testing.T) {
	setupAdminTenantTestDB(t)

	body := `{"identifier":"","domain":"x.com"}`
	w := httptest.NewRecorder()
	r := adminTenantReq("POST", "/tenants/domains", body)
	HandleTenantDomains(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleTenantDomains_AddDomain_Forbidden(t *testing.T) {
	setupAdminTenantTestDB(t)

	body := `{"identifier":"t","domain":"d.com"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/tenants/domains", strings.NewReader(body))
	// 无 auth
	HandleTenantDomains(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// ─── HandleTenantDomains (DELETE) ─────────────────────────────────────────

func TestHandleTenantDomains_RemoveDomain_Success(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	db.Create(&model.SiteConfig{Identifier: "rm-tenant", Domain: "https://main.example.com"})
	db.Create(&model.TenantDomain{Domain: "secondary.example.com", Identifier: "rm-tenant"})

	body := `{"identifier":"rm-tenant","domain":"secondary.example.com"}`
	w := httptest.NewRecorder()
	r := adminTenantReq("DELETE", "/tenants/domains", body)
	HandleTenantDomains(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 验证已删除
	var count int64
	db.Model(&model.TenantDomain{}).Where("domain = ?", "secondary.example.com").Count(&count)
	if count != 0 {
		t.Fatal("domain should be deleted")
	}
}

func TestHandleTenantDomains_RemovePrimaryDomain_Forbidden(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	db.Create(&model.SiteConfig{Identifier: "protect-tenant", Domain: "https://main.example.com"})
	db.Create(&model.TenantDomain{Domain: "main.example.com", Identifier: "protect-tenant"})

	body := `{"identifier":"protect-tenant","domain":"main.example.com"}`
	w := httptest.NewRecorder()
	r := adminTenantReq("DELETE", "/tenants/domains", body)
	HandleTenantDomains(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for primary domain removal, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleTenantDomains_MethodNotAllowed(t *testing.T) {
	setupAdminTenantTestDB(t)

	w := httptest.NewRecorder()
	r := adminTenantReq("PUT", "/tenants/domains", `{}`)
	HandleTenantDomains(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ─── HandleListTenantDomains ─────────────────────────────────────────

func TestHandleListTenantDomains_Success(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	db.Create(&model.SiteConfig{Identifier: "list-tenant", Domain: "https://main.list.com"})
	db.Create(&model.TenantDomain{Domain: "main.list.com", Identifier: "list-tenant"})
	db.Create(&model.TenantDomain{Domain: "extra.list.com", Identifier: "list-tenant"})

	w := httptest.NewRecorder()
	r := adminTenantReq("GET", "/tenants/list-tenant/domains", "")
	HandleListTenantDomains(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Fatalf("expected ok=true, got %v", resp)
	}
	if resp["identifier"] != "list-tenant" {
		t.Fatalf("expected identifier list-tenant, got %v", resp["identifier"])
	}
	domains, ok := resp["domains"].([]interface{})
	if !ok || len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %v", resp["domains"])
	}

	// 验证 is_main 标记
	hasMain := false
	for _, d := range domains {
		dm := d.(map[string]interface{})
		if dm["domain"] == "main.list.com" && dm["is_main"] == true {
			hasMain = true
		}
	}
	if !hasMain {
		t.Fatal("main.list.com should have is_main=true")
	}
}

func TestHandleListTenantDomains_MethodNotAllowed(t *testing.T) {
	setupAdminTenantTestDB(t)
	w := httptest.NewRecorder()
	r := adminTenantReq("POST", "/tenants/x/domains", "")
	HandleListTenantDomains(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleListTenantDomains_Forbidden(t *testing.T) {
	setupAdminTenantTestDB(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/tenants/x/domains", nil)
	HandleListTenantDomains(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestHandleListTenantDomains_InvalidPath(t *testing.T) {
	setupAdminTenantTestDB(t)
	w := httptest.NewRecorder()
	r := adminTenantReq("GET", "/tenants/", "")
	HandleListTenantDomains(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleListTenantDomains_EmptyResult(t *testing.T) {
	setupAdminTenantTestDB(t)
	w := httptest.NewRecorder()
	r := adminTenantReq("GET", "/tenants/nonexist/domains", "")
	HandleListTenantDomains(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	domains := resp["domains"].([]interface{})
	if len(domains) != 0 {
		t.Fatalf("expected empty domains list, got %v", domains)
	}
}

// ─── extractHost ─────────────────────────────────────────

func TestExtractHost_Simple(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"example.com", "example.com"},
		{"Example.COM", "example.com"},
		{"example.com:8080", "example.com"},
		{"[::1]:8080", "::1"},
	}
	for _, tt := range tests {
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = tt.host
		got := extractHost(r)
		if got != tt.want {
			t.Errorf("extractHost(%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}

// ─── isTenantAgnosticPath (扩展测试) ─────────────────────────────────

func TestIsTenantAgnosticPath_TenantsPrefix(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/tenants", true},
		{"/tenants/init", true},
		{"/tenants/abc/domains", true},
		{"/tenants/domains", true},
		{"/health", true},
		{"/admin/tenants", false}, // 不是 /tenants 前缀
		{"/tenant", false},        // 少了 s
	}
	for _, tt := range tests {
		got := isTenantAgnosticPath(tt.path)
		if got != tt.want {
			t.Errorf("isTenantAgnosticPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

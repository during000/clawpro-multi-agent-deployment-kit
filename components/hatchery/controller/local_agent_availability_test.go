package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"
)

// TestHandleLocalAgentAvailability_BothLayers
// 综合矩阵：白名单 × SiteConfig 共 4 种组合，验证 enabled 是 AND 语义。
//
// 不在 cases 表里展开是因为每组都需要不同的 setup 顺序（feature_allowlist 表 / SiteConfig），
// 直接顺序写出来更清楚，也避免子测试间共享 DB 状态。
func TestHandleLocalAgentAvailability_BothLayers(t *testing.T) {
	type scenario struct {
		name            string
		writeAllowlist  bool   // 表里写一条 allowed-tenant
		userIdentifier  string // 用户 identifier
		siteEnabled     bool
		expectedEnabled bool
	}
	// 表里有 allowed-tenant + 用户也是 allowed-tenant + Site 开 → enabled=true
	// 表里有 allowed-tenant + 用户是 blocked-tenant + Site 开 → enabled=false（白名单未命中）
	// 表里有 allowed-tenant + 用户是 allowed-tenant + Site 关 → enabled=false（SiteConfig 关）
	// 表为空（全开）+ Site 开 → enabled=true
	scenarios := []scenario{
		{"both_pass", true, "allowed-tenant", true, true},
		{"allowlist_miss", true, "blocked-tenant", true, false},
		{"site_off", true, "allowed-tenant", false, false},
		{"empty_table_full_open", false, "any-tenant", true, true},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			setupSkillInstancesDB(t)
			migrateLocalAgentTables(t)
			if err := model.DB(context.Background()).AutoMigrate(&model.FeatureAllowlist{}); err != nil {
				t.Fatalf("migrate FeatureAllowlist: %v", err)
			}
			ctx := context.Background()

			if !sc.siteEnabled {
				disableLocalAgentSiteConfig(t)
			}
			if sc.writeAllowlist {
				if err := model.DB(ctx).Create(&model.FeatureAllowlist{
					Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "allowed-tenant",
				}).Error; err != nil {
					t.Fatalf("create allowlist: %v", err)
				}
			}
			user := model.User{Username: "u-" + sc.name, Role: "user", Identifier: sc.userIdentifier}
			if err := model.DB(ctx).Create(&user).Error; err != nil {
				t.Fatalf("create user: %v", err)
			}

			rr := httptest.NewRecorder()
			HandleLocalAgentAvailability(rr, authReqWithSession(t, user.Username, "/local-agent/availability"))
			if rr.Code != http.StatusOK {
				t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
			}
			var resp struct {
				Enabled bool `json:"enabled"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode resp: %v body=%s", err, rr.Body.String())
			}
			if resp.Enabled != sc.expectedEnabled {
				t.Errorf("enabled 期望=%v 实际=%v body=%s",
					sc.expectedEnabled, resp.Enabled, rr.Body.String())
			}
		})
	}
}

// TestHandleLocalAgentAvailability_MethodNotAllowed
// 仅 GET，POST 应被拒绝。
func TestHandleLocalAgentAvailability_MethodNotAllowed(t *testing.T) {
	setupSkillInstancesDB(t)
	user := model.User{Username: "method-test", Role: "user", Identifier: "x"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/local-agent/availability", nil)
	HandleLocalAgentAvailability(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("应 405，实际=%d", rr.Code)
	}
}

// TestHandleAdminFeatureAllowlistCheck_Matrix
// 4 种组合：identifier 取自当前登录 tenant admin（不再由前端指定）。
//   - in_list:  type 下有该 identifier        → true
//   - not_in_list:  type 下有别的 identifier 但不含该条 → false
//   - empty_table:  type 下零条记录（空表全开）  → true
//   - other_type_only:  只有别的 type 的记录    → true（视为本 type 空表）
func TestHandleAdminFeatureAllowlistCheck_Matrix(t *testing.T) {
	type scenario struct {
		name       string
		seed       []model.FeatureAllowlist
		queryType  string
		userIdent  string
		wantInList bool
	}
	scenarios := []scenario{
		{
			name:       "in_list",
			seed:       []model.FeatureAllowlist{{Type: "local-agent", Identifier: "tenant-A"}},
			queryType:  "local-agent",
			userIdent:  "tenant-A",
			wantInList: true,
		},
		{
			name:       "not_in_list",
			seed:       []model.FeatureAllowlist{{Type: "local-agent", Identifier: "tenant-A"}},
			queryType:  "local-agent",
			userIdent:  "tenant-B",
			wantInList: false,
		},
		{
			name:       "empty_table_full_open",
			seed:       nil,
			queryType:  "local-agent",
			userIdent:  "tenant-X",
			wantInList: true,
		},
		{
			name:       "other_type_only_full_open",
			seed:       []model.FeatureAllowlist{{Type: "other-feature", Identifier: "tenant-A"}},
			queryType:  "local-agent",
			userIdent:  "tenant-A",
			wantInList: true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			setupSkillInstancesDB(t)
			if err := model.DB(context.Background()).AutoMigrate(&model.FeatureAllowlist{}); err != nil {
				t.Fatalf("migrate FeatureAllowlist: %v", err)
			}
			ctx := context.Background()
			for _, row := range sc.seed {
				if err := model.DB(ctx).Create(&row).Error; err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			// 当前登录的 tenant admin，identifier 决定白名单归属。
			user := model.User{Username: "u-" + sc.name, Role: "admin", Identifier: sc.userIdent}
			if err := model.DB(ctx).Create(&user).Error; err != nil {
				t.Fatalf("create user: %v", err)
			}

			rr := httptest.NewRecorder()
			HandleAdminFeatureAllowlistCheck(rr, authReqWithSession(t, user.Username,
				"/admin/feature-allowlist/check?type="+sc.queryType))
			if rr.Code != http.StatusOK {
				t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
			}
			var resp struct {
				InAllowlist bool `json:"in_allowlist"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode resp: %v body=%s", err, rr.Body.String())
			}
			if resp.InAllowlist != sc.wantInList {
				t.Errorf("in_allowlist 期望=%v 实际=%v body=%s",
					sc.wantInList, resp.InAllowlist, rr.Body.String())
			}
		})
	}
}

// TestHandleAdminFeatureAllowlistCheck_AdminTokenBypass
// 超管（AdminToken 请求）绕过白名单，无论表里是否命中都返 in_allowlist=true。
func TestHandleAdminFeatureAllowlistCheck_AdminTokenBypass(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.FeatureAllowlist{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ctx := context.Background()
	// 表里只命中别的租户，超管本应不命中，但 AdminToken 直通应返 true。
	if err := model.DB(ctx).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "some-tenant",
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	rr := httptest.NewRecorder()
	HandleAdminFeatureAllowlistCheck(rr, adminJSONGet(
		"/admin/feature-allowlist/check?type=local-agent"))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		InAllowlist bool `json:"in_allowlist"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v body=%s", err, rr.Body.String())
	}
	if !resp.InAllowlist {
		t.Errorf("超管应绕过白名单 in_allowlist=true，实际=%v body=%s",
			resp.InAllowlist, rr.Body.String())
	}
}

// TestHandleAdminFeatureAllowlistCheck_MissingParams
// type 缺失 → 400（identifier 已不再由前端提供，仅校验 type）。
func TestHandleAdminFeatureAllowlistCheck_MissingParams(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.FeatureAllowlist{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	cases := []struct {
		name string
		url  string
	}{
		{"missing_type", "/admin/feature-allowlist/check"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			HandleAdminFeatureAllowlistCheck(rr, adminJSONGet(c.url))
			if rr.Code != http.StatusBadRequest {
				t.Errorf("应 400，实际=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

// TestHandleAdminFeatureAllowlistCheck_MethodNotAllowed
// 仅 GET。
func TestHandleAdminFeatureAllowlistCheck_MethodNotAllowed(t *testing.T) {
	setupSkillInstancesDB(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/feature-allowlist/check", nil)
	HandleAdminFeatureAllowlistCheck(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("应 405，实际=%d", rr.Code)
	}
}

func TestHandleAdminFeatureAllowlistCheck_UnauthorizedAndDBError(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.FeatureAllowlist{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/feature-allowlist/check?type=local-agent", nil)
	req.Header.Set("Accept", "application/json")
	HandleAdminFeatureAllowlistCheck(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d body=%s", rr.Code, rr.Body.String())
	}

	user := model.User{Username: "allowlist-db-error", Role: "admin", Identifier: "tenant-error"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	authReq := authReqWithSession(t, user.Username, "/admin/feature-allowlist/check?type=local-agent")
	if err := model.DB(context.Background()).Migrator().DropTable(&model.FeatureAllowlist{}); err != nil {
		t.Fatalf("drop allowlist table: %v", err)
	}
	rr = httptest.NewRecorder()
	HandleAdminFeatureAllowlistCheck(rr, authReq)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("DB error status=%d body=%s", rr.Code, rr.Body.String())
	}
}

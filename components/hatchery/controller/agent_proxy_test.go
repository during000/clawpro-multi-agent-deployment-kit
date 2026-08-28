package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func initAgentProxyTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentProxyRoute{}, &model.Instance{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return model.UseDBForTest(db)
}

func TestAgentProxyRouteSpecAndIDHelpers(t *testing.T) {
	spec, ok := agentProxyRouteSpecForKind(model.AgentProxyRouteKindTeams)
	if !ok || spec.Kind != model.AgentProxyRouteKindTeams || spec.TargetPort != 3978 || spec.TargetPath != "/api/messages" {
		t.Fatalf("teams spec mismatch: %+v ok=%v", spec, ok)
	}
	lineSpec, lineOK := agentProxyRouteSpecForKind(model.AgentProxyRouteKindLine)
	if !lineOK || lineSpec.Kind != model.AgentProxyRouteKindLine || lineSpec.TargetPort != 8646 || lineSpec.TargetPath != "/line/webhook" {
		t.Fatalf("line spec mismatch: %+v ok=%v", lineSpec, lineOK)
	}
	if _, ok := agentProxyRouteSpecForKind("unknown"); ok {
		t.Fatal("unknown kind should not have a spec")
	}
	id, err := randomRouteID()
	if err != nil {
		t.Fatalf("randomRouteID error: %v", err)
	}
	if len(id) < 40 || strings.ContainsAny(id, "+/=") {
		t.Fatalf("route id should be base64url without padding, got %q", id)
	}
}

func TestForwardedAndBaseURLHelpers(t *testing.T) {
	if got := firstForwardedHeaderValue(" https, http "); got != "https" {
		t.Fatalf("firstForwardedHeaderValue = %q", got)
	}
	if got := normalizeExternalBaseURL("example.com/"); got != "https://example.com" {
		t.Fatalf("normalizeExternalBaseURL without scheme = %q", got)
	}
	if got := normalizeExternalBaseURL("https://example.com/base/?x=1#f"); got != "https://example.com/base" {
		t.Fatalf("normalizeExternalBaseURL strips query/fragment = %q", got)
	}
	if got := normalizeExternalBaseURL("http://[::1"); got != "http://[::1" {
		t.Fatalf("normalizeExternalBaseURL invalid fallback = %q", got)
	}

	req := httptest.NewRequest(http.MethodGet, "https://fallback.example.com/x", nil)
	if got := requestExternalBaseURL(req); got != "https://fallback.example.com" {
		t.Fatalf("requestExternalBaseURL TLS fallback = %q", got)
	}
	httpReq := httptest.NewRequest(http.MethodGet, "http://fallback.example.com/x", nil)
	if got := requestExternalBaseURL(httpReq); got != "http://fallback.example.com" {
		t.Fatalf("requestExternalBaseURL HTTP fallback = %q", got)
	}
}

func TestProxyEndpointForRoute_MatchesLegacyShape(t *testing.T) {
	route := model.AgentProxyRoute{RouteID: "Bs8uYCN35lxDPaAXGcOQt5SpJTulPLJy0bQyMpwItpM", TargetPath: "/api/messages"}

	tests := []struct {
		name      string
		domain    string
		reqHost   string
		forwarded map[string]string
		want      string
	}{
		{
			name:   "tenant domain without scheme defaults to https",
			domain: "h7thfyin.cvmopenclaw.site",
			want:   "https://h7thfyin.cvmopenclaw.site/api/proxy/Bs8uYCN35lxDPaAXGcOQt5SpJTulPLJy0bQyMpwItpM/api/messages",
		},
		{
			name:   "tenant domain with scheme and trailing slash",
			domain: "https://h7thfyin.cvmopenclaw.site/",
			want:   "https://h7thfyin.cvmopenclaw.site/api/proxy/Bs8uYCN35lxDPaAXGcOQt5SpJTulPLJy0bQyMpwItpM/api/messages",
		},
		{
			name:    "fallback forwarded proto host",
			reqHost: "internal.local",
			forwarded: map[string]string{
				"X-Forwarded-Proto": "https",
				"X-Forwarded-Host":  "h7thfyin.cvmopenclaw.site",
			},
			want: "https://h7thfyin.cvmopenclaw.site/api/proxy/Bs8uYCN35lxDPaAXGcOQt5SpJTulPLJy0bQyMpwItpM/api/messages",
		},
		{
			name:    "fallback first forwarded value",
			reqHost: "internal.local",
			forwarded: map[string]string{
				"X-Forwarded-Proto": "https, http",
				"X-Forwarded-Host":  "h7thfyin.cvmopenclaw.site, internal.local",
			},
			want: "https://h7thfyin.cvmopenclaw.site/api/proxy/Bs8uYCN35lxDPaAXGcOQt5SpJTulPLJy0bQyMpwItpM/api/messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "http://"+nonEmpty(tt.reqHost, "internal.local")+"/openclaw/proxy/prepare", nil)
			for k, v := range tt.forwarded {
				req.Header.Set(k, v)
			}
			if tt.domain != "" {
				ctx := common.InjectTenant(req.Context(), common.TenantSnapshot{Domain: tt.domain})
				req = req.WithContext(ctx)
			}
			if got := proxyEndpointForRoute(req, route); got != tt.want {
				t.Fatalf("proxyEndpointForRoute() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseProxyPath(t *testing.T) {
	tests := []struct {
		path        string
		wantRouteID string
		wantRest    string
		wantOK      bool
	}{
		{"/proxy/route-1/api/messages", "route-1", "/api/messages", true},
		{"/api/proxy/route-2/api/messages", "route-2", "/api/messages", true},
		{"/proxy//api/messages", "", "", false},
		{"/not-proxy/route/api/messages", "", "", false},
	}
	for _, tt := range tests {
		routeID, rest, ok := parseProxyPath(tt.path)
		if routeID != tt.wantRouteID || rest != tt.wantRest || ok != tt.wantOK {
			t.Fatalf("parseProxyPath(%q)=(%q,%q,%v), want (%q,%q,%v)", tt.path, routeID, rest, ok, tt.wantRouteID, tt.wantRest, tt.wantOK)
		}
	}
}

func withAgentProxyHooks(t *testing.T, ip string, refreshErr error) {
	t.Helper()
	oldResolve := resolveInstanceAccessIPForAgentProxy
	oldRefresh := refreshRuleSetsForAgentProxy
	resolveInstanceAccessIPForAgentProxy = func(r *http.Request, instanceID string) (string, error) {
		if ip == "" {
			return "", errors.New("no ip")
		}
		return ip, nil
	}
	refreshRuleSetsForAgentProxy = func(ctx context.Context) error { return refreshErr }
	t.Cleanup(func() {
		resolveInstanceAccessIPForAgentProxy = oldResolve
		refreshRuleSetsForAgentProxy = oldRefresh
	})
}

func TestEnsureAgentProxyRoute_CreateUpdateAndRollback(t *testing.T) {
	cleanup := initAgentProxyTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.0.0.8", nil)
	req := httptest.NewRequest(http.MethodPost, "https://tenant.example.com/openclaw/proxy/prepare", nil)
	inst := &model.Instance{InstanceId: "ins-ensure"}
	route, endpoint, rerr := ensureAgentProxyRoute(req, inst, model.AgentProxyRouteKindTeams)
	if rerr != nil {
		t.Fatalf("ensure create error: %v", rerr)
	}
	if route.RouteID == "" || route.TargetIP != "10.0.0.8" || route.TargetPort != 3978 || route.TargetPath != "/api/messages" || !route.Enabled {
		t.Fatalf("created route mismatch: %+v", route)
	}
	if !strings.Contains(endpoint, "/api/proxy/"+route.RouteID+"/api/messages") {
		t.Fatalf("endpoint mismatch: %q", endpoint)
	}

	withAgentProxyHooks(t, "10.0.0.9", nil)
	route2, _, rerr := ensureAgentProxyRoute(req, inst, model.AgentProxyRouteKindTeams)
	if rerr != nil {
		t.Fatalf("ensure update error: %v", rerr)
	}
	if route2.ID != route.ID || route2.TargetIP != "10.0.0.9" || !route2.Enabled {
		t.Fatalf("updated route mismatch: old=%+v new=%+v", route, route2)
	}
}

func TestEnsureAgentProxyRoute_RefreshFailureRollsBackNewAndDisabledRoute(t *testing.T) {
	cleanup := initAgentProxyTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "http://tenant.example.com/openclaw/proxy/prepare", nil)

	withAgentProxyHooks(t, "10.0.0.8", errors.New("refresh failed"))
	if _, _, rerr := ensureAgentProxyRoute(req, &model.Instance{InstanceId: "ins-new-fail"}, model.AgentProxyRouteKindTeams); rerr == nil {
		t.Fatal("refresh failure should return rich error")
	}
	var created model.AgentProxyRoute
	if err := model.DB(context.Background()).Where("instance_id = ?", "ins-new-fail").First(&created).Error; err != nil {
		t.Fatalf("new failed route should exist but disabled: %v", err)
	}
	if created.Enabled {
		t.Fatal("new route should be rolled back to disabled")
	}

	existing := model.AgentProxyRoute{RouteID: "existing-disabled", InstanceID: "ins-existing-fail", Kind: model.AgentProxyRouteKindTeams, Enabled: true}
	if err := model.DB(context.Background()).Create(&existing).Error; err != nil {
		t.Fatalf("create existing route: %v", err)
	}
	if err := model.DB(context.Background()).Model(&existing).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable existing route: %v", err)
	}
	if _, _, rerr := ensureAgentProxyRoute(req, &model.Instance{InstanceId: "ins-existing-fail"}, model.AgentProxyRouteKindTeams); rerr == nil {
		t.Fatal("refresh failure should return rich error for existing route")
	}
	var got model.AgentProxyRoute
	model.DB(context.Background()).Where("route_id = ?", "existing-disabled").First(&got)
	if got.Enabled {
		t.Fatal("previously disabled route should remain disabled after rollback")
	}
}

func TestEnsureAgentProxyRoute_ValidationErrors(t *testing.T) {
	cleanup := initAgentProxyTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/openclaw/proxy/prepare", nil)
	if _, _, err := ensureAgentProxyRoute(req, nil, model.AgentProxyRouteKindTeams); err == nil {
		t.Fatal("nil instance should fail")
	}
	if _, _, err := ensureAgentProxyRoute(req, &model.Instance{InstanceId: "ins-1"}, "unknown"); err == nil {
		t.Fatal("unknown kind should fail")
	}
}

func setDefaultLangForTest(t *testing.T, lang string) {
	t.Helper()
	i18n.SetDefaultLang(lang)
	t.Cleanup(func() { i18n.SetDefaultLang("zh") })
}

func TestHandleProxyPrepare_SuccessDefaultKind(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.1.1.1", nil)
	user := &model.User{Username: "prepare-user", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := &model.Instance{Name: "i", InstanceId: "ins-prepare", UserID: user.ID, AgentType: model.AgentTypeOpenClaw}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	req := channelReqWithSession(t, http.MethodPost, "/openclaw/proxy/prepare?id=1", "prepare-user", "")
	req.URL.RawQuery = "id=" + strconv.FormatUint(uint64(inst.ID), 10)
	rr := httptest.NewRecorder()
	HandleProxyPrepare(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("prepare code=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"kind":"teams"`) || !strings.Contains(rr.Body.String(), `"endpoint"`) {
		t.Fatalf("unexpected response: %s", rr.Body.String())
	}
}

func TestHandleProxyPrepare_DomesticAllowsTeamsKind(t *testing.T) {
	setDefaultLangForTest(t, "zh")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.1.1.1", nil)
	user := &model.User{Username: "domestic-prepare-user", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := &model.Instance{Name: "i", InstanceId: "ins-domestic-prepare", UserID: user.ID, AgentType: model.AgentTypeOpenClaw}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	req := channelReqWithSession(t, http.MethodPost, "/openclaw/proxy/prepare?id="+strconv.FormatUint(uint64(inst.ID), 10), "domestic-prepare-user", "")
	rr := httptest.NewRecorder()
	HandleProxyPrepare(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("domestic prepare code=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"kind":"teams"`) || !strings.Contains(rr.Body.String(), `"endpoint"`) {
		t.Fatalf("domestic prepare should succeed with teams endpoint: %s", rr.Body.String())
	}
}

func TestHandleAdminProxyPrepare_MethodKindAndSuccess(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.2.2.2", nil)
	inst := &model.Instance{Name: "admin-i", InstanceId: "ins-admin-prepare", UserID: 99, AgentType: model.AgentTypeOpenClaw}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	getReq := adminTokenReq(http.MethodGet, "/admin/instances/proxy/prepare", "")
	getRR := httptest.NewRecorder()
	HandleAdminProxyPrepare(getRR, getReq)
	if getRR.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET admin prepare code=%d body=%s", getRR.Code, getRR.Body.String())
	}

	badReq := adminTokenReq(http.MethodPost, "/admin/instances/proxy/prepare?kind=bad", "")
	badRR := httptest.NewRecorder()
	HandleAdminProxyPrepare(badRR, badReq)
	if badRR.Code != http.StatusBadRequest {
		t.Fatalf("bad kind code=%d body=%s", badRR.Code, badRR.Body.String())
	}

	successReq := adminTokenReq(http.MethodPost, "/admin/instances/proxy/prepare?id="+strconv.FormatUint(uint64(inst.ID), 10), "")
	successRR := httptest.NewRecorder()
	HandleAdminProxyPrepare(successRR, successReq)
	if successRR.Code != http.StatusOK {
		t.Fatalf("admin prepare code=%d body=%s", successRR.Code, successRR.Body.String())
	}
}

func TestHandleAdminProxyPrepare_DomesticAllowsTeamsKind(t *testing.T) {
	setDefaultLangForTest(t, "zh")
	cleanup := initChannelTestDB(t)
	defer cleanup()
	withAgentProxyHooks(t, "10.2.2.2", nil)
	inst := &model.Instance{Name: "admin-domestic-i", InstanceId: "ins-admin-domestic-prepare", UserID: 99, AgentType: model.AgentTypeOpenClaw}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	req := adminTokenReq(http.MethodPost, "/admin/instances/proxy/prepare?id="+strconv.FormatUint(uint64(inst.ID), 10), "")
	rr := httptest.NewRecorder()
	HandleAdminProxyPrepare(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("domestic admin prepare code=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"kind":"teams"`) || !strings.Contains(rr.Body.String(), `"endpoint"`) {
		t.Fatalf("domestic admin prepare should succeed with teams endpoint: %s", rr.Body.String())
	}
}

func TestHandleProxyPrepare_MethodAndKindValidation(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	getReq := channelReqWithSession(t, http.MethodGet, "/openclaw/proxy/prepare", "u1", "")
	getRR := httptest.NewRecorder()
	HandleProxyPrepare(getRR, getReq)
	if getRR.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET prepare code=%d body=%s", getRR.Code, getRR.Body.String())
	}

	postReq := channelReqWithSession(t, http.MethodPost, "/openclaw/proxy/prepare?kind=unknown", "u1", "")
	postRR := httptest.NewRecorder()
	HandleProxyPrepare(postRR, postReq)
	if postRR.Code != http.StatusBadRequest {
		t.Fatalf("unknown kind code=%d body=%s", postRR.Code, postRR.Body.String())
	}
}

func TestHandleAgentProxy_NotFoundAndMethodBranches(t *testing.T) {
	cleanup := initAgentProxyTestDB(t)
	defer cleanup()

	cases := []struct {
		name  string
		route model.AgentProxyRoute
		path  string
		want  int
	}{
		{name: "bad path", path: "/bad/route", want: http.StatusNotFound},
		{name: "missing route", path: "/proxy/missing/api/messages", want: http.StatusNotFound},
		{name: "disabled", route: model.AgentProxyRoute{RouteID: "disabled", InstanceID: "ins-disabled", Kind: model.AgentProxyRouteKindTeams, TargetIP: "127.0.0.1", TargetPort: 3978, TargetPath: "/api/messages", Enabled: false}, path: "/proxy/disabled/api/messages", want: http.StatusNotFound},
		{name: "wrong target path", route: model.AgentProxyRoute{RouteID: "wrong", InstanceID: "ins-wrong", Kind: model.AgentProxyRouteKindTeams, TargetIP: "127.0.0.1", TargetPort: 3978, TargetPath: "/api/messages", Enabled: true}, path: "/proxy/wrong/other", want: http.StatusNotFound},
		{name: "teams requires post", route: model.AgentProxyRoute{RouteID: "method", InstanceID: "ins-method", Kind: model.AgentProxyRouteKindTeams, TargetIP: "127.0.0.1", TargetPort: 3978, TargetPath: "/api/messages", Enabled: true}, path: "/proxy/method/api/messages", want: http.StatusMethodNotAllowed},
		{name: "line requires post/get", route: model.AgentProxyRoute{RouteID: "line-method", InstanceID: "ins-line-method", Kind: model.AgentProxyRouteKindLine, TargetIP: "127.0.0.1", TargetPort: 8646, TargetPath: "/line/webhook", Enabled: true}, path: "/proxy/line-method/line/webhook", want: http.StatusMethodNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.route.RouteID != "" {
				wantDisabled := !tc.route.Enabled
				tc.route.Enabled = true // GORM default tags ignore false zero value on create.
				if err := model.DB(context.Background()).Create(&tc.route).Error; err != nil {
					t.Fatalf("create route: %v", err)
				}
				if wantDisabled {
					if err := model.DB(context.Background()).Model(&tc.route).Update("enabled", false).Error; err != nil {
						t.Fatalf("disable route: %v", err)
					}
				}
			}
			req := httptest.NewRequest(http.MethodPut, tc.path, nil)
			rr := httptest.NewRecorder()
			HandleAgentProxy(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("code=%d want=%d body=%s", rr.Code, tc.want, rr.Body.String())
			}
		})
	}
}

func TestAgentProxyRouteKindConditionHelpers(t *testing.T) {
	cleanup := initAgentProxyTestDB(t)
	defer cleanup()
	ctx := context.Background()
	if err := model.DB(ctx).Create(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}
	if agentProxyRouteKindEnabled(ctx, model.AgentProxyRouteKindTeams) {
		t.Fatal("no enabled route should be false")
	}
	if agentProxyRouteKindEnabled(ctx, model.AgentProxyRouteKindLine) {
		t.Fatal("no enabled line route should be false")
	}
	if siteConfigRequiresRecommendedRules(ctx) {
		t.Fatal("no feature enabled should not require recommended rules")
	}
	disabled := model.AgentProxyRoute{RouteID: "disabled-cond", Kind: model.AgentProxyRouteKindTeams, InstanceID: "ins-disabled", Enabled: true}
	if err := model.DB(ctx).Create(&disabled).Error; err != nil {
		t.Fatalf("create disabled route: %v", err)
	}
	if err := model.DB(ctx).Model(&disabled).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable route: %v", err)
	}
	if agentProxyRouteKindEnabled(ctx, model.AgentProxyRouteKindTeams) {
		t.Fatal("disabled route should not enable condition")
	}
	disabledRuleSet := sgRuleSet{Categories: []sgRuleCategory{{Type: "recommended", RuleGroups: []sgRuleGroup{
		{Key: "teams", Condition: "agent_proxy_teams_enable", Rules: []requiredSGRule{{Port: "3978"}}},
	}}}}
	resolveConditionalRules(ctx, &disabledRuleSet)
	if len(disabledRuleSet.Categories[0].RuleGroups) != 0 {
		t.Fatalf("disabled teams condition should be filtered, got %+v", disabledRuleSet.Categories[0].RuleGroups)
	}
	// Test LINE condition with no enabled LINE routes
	lineDisabledRuleSet := sgRuleSet{Categories: []sgRuleCategory{{Type: "recommended", RuleGroups: []sgRuleGroup{
		{Key: "line", Condition: "agent_proxy_line_enable", Rules: []requiredSGRule{{Port: "8646"}}},
	}}}}
	resolveConditionalRules(ctx, &lineDisabledRuleSet)
	if len(lineDisabledRuleSet.Categories[0].RuleGroups) != 0 {
		t.Fatalf("disabled line condition should be filtered, got %+v", lineDisabledRuleSet.Categories[0].RuleGroups)
	}
	enabled := model.AgentProxyRoute{RouteID: "enabled-cond", Kind: model.AgentProxyRouteKindTeams, InstanceID: "ins-enabled", Enabled: true}
	if err := model.DB(ctx).Create(&enabled).Error; err != nil {
		t.Fatalf("create enabled route: %v", err)
	}
	if !agentProxyRouteKindEnabled(ctx, model.AgentProxyRouteKindTeams) {
		t.Fatal("enabled route should enable condition")
	}
	if !siteConfigRequiresRecommendedRules(ctx) {
		t.Fatal("enabled proxy route should require recommended rules")
	}

	ruleSet := sgRuleSet{Categories: []sgRuleCategory{{Type: "recommended", RuleGroups: []sgRuleGroup{
		{Key: "teams", Condition: "agent_proxy_teams_enable", Rules: []requiredSGRule{{Port: "3978"}}},
		{Key: "line", Condition: "agent_proxy_line_enable", Rules: []requiredSGRule{{Port: "8646"}}},
		{Key: "unknown", Condition: "unknown_condition", Rules: []requiredSGRule{{Port: "1"}}},
	}}}}
	resolveConditionalRules(ctx, &ruleSet)
	groups := ruleSet.Categories[0].RuleGroups
	if len(groups) != 1 || groups[0].Key != "teams" || groups[0].Condition != "" {
		t.Fatalf("expected only teams group with condition stripped (line not enabled), got %+v", groups)
	}
	// Enable a LINE route and verify LINE condition is resolved
	lineEnabled := model.AgentProxyRoute{RouteID: "line-enabled-cond", Kind: model.AgentProxyRouteKindLine, InstanceID: "ins-line", Enabled: true}
	if err := model.DB(ctx).Create(&lineEnabled).Error; err != nil {
		t.Fatalf("create line enabled route: %v", err)
	}
	lineRuleSet := sgRuleSet{Categories: []sgRuleCategory{{Type: "recommended", RuleGroups: []sgRuleGroup{
		{Key: "line", Condition: "agent_proxy_line_enable", Rules: []requiredSGRule{{Port: "8646"}}},
	}}}}
	resolveConditionalRules(ctx, &lineRuleSet)
	if len(lineRuleSet.Categories[0].RuleGroups) != 1 || lineRuleSet.Categories[0].RuleGroups[0].Key != "line" || lineRuleSet.Categories[0].RuleGroups[0].Condition != "" {
		t.Fatalf("expected line group with condition stripped when enabled, got %+v", lineRuleSet.Categories[0].RuleGroups)
	}
}

func TestHandleAgentProxy_ReverseProxySuccessAndBadGateway(t *testing.T) {
	cleanup := initAgentProxyTestDB(t)
	defer cleanup()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/messages" || r.URL.RawQuery != "q=1" {
			t.Fatalf("unexpected upstream request: method=%s path=%s query=%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		if r.Header.Get("X-Forwarded-Proto") != "https" || r.Header.Get("X-Forwarded-Host") != "tenant.example.com" {
			t.Fatalf("forwarded headers not preserved: proto=%q host=%q", r.Header.Get("X-Forwarded-Proto"), r.Header.Get("X-Forwarded-Host"))
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("split upstream host: %v", err)
	}
	portNum, err := net.LookupPort("tcp", port)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	route := model.AgentProxyRoute{RouteID: "ok-route", InstanceID: "ins-ok", Kind: model.AgentProxyRouteKindTeams, TargetIP: host, TargetPort: portNum, TargetPath: "/api/messages", Enabled: true}
	if err := model.DB(context.Background()).Create(&route).Error; err != nil {
		t.Fatalf("create route: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "https://tenant.example.com/proxy/ok-route/api/messages?q=1", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	HandleAgentProxy(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("proxy success code=%d body=%s", rr.Code, rr.Body.String())
	}

	bad := model.AgentProxyRoute{RouteID: "bad-gateway", InstanceID: "ins-bad", Kind: model.AgentProxyRouteKindTeams, TargetIP: "127.0.0.1", TargetPort: 1, TargetPath: "/api/messages", Enabled: true}
	if err := model.DB(context.Background()).Create(&bad).Error; err != nil {
		t.Fatalf("create bad route: %v", err)
	}
	badReq := httptest.NewRequest(http.MethodPost, "/proxy/bad-gateway/api/messages", nil)
	badRR := httptest.NewRecorder()
	HandleAgentProxy(badRR, badReq)
	if badRR.Code != http.StatusBadGateway {
		t.Fatalf("bad gateway code=%d body=%s", badRR.Code, badRR.Body.String())
	}
}

func TestCheckAgentProxyKindAllowed_UnknownKind(t *testing.T) {
	cleanup := initAgentProxyTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, InstanceId: "ins-test"}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	// Unknown kind that's not in agentProxyKindToChannel should return nil (line 217-219)
	err := checkAgentProxyKindAllowed(req, "unknown_kind_xyz", inst)
	if err != nil {
		t.Fatalf("unknown kind should return nil, got: %v", err)
	}
}

func TestCheckAgentProxyKindAllowed_AgentTypeNotSupportChannel(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initAgentProxyTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw, InstanceId: "ins-test2"}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	// "line" is overseas-only. In domestic context, channelInCurrentSiteScope returns false,
	// so the check should fail with MsgChannelNotExist (line 220-222).
	err := checkAgentProxyKindAllowed(req, model.AgentProxyRouteKindLine, inst)
	if err == nil {
		t.Fatal("line kind in domestic context should return error")
	}
}

func TestCheckAgentProxyKindAllowed_AgentTypeNotSupportChannel_AgentTypeRejection(t *testing.T) {
	setDefaultLangForTest(t, "en")
	cleanup := initAgentProxyTestDB(t)
	defer cleanup()

	// Inject overseas context so line passes site scope check
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{DefaultLang: "en"})
	req := httptest.NewRequest(http.MethodPost, "/test", nil).WithContext(ctx)

	// DeepSeekTUI agent type doesn't support any channel
	inst := &model.Instance{AgentType: model.AgentTypeDeepSeekTUI, InstanceId: "ins-deepseek"}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	// AgentTypeChannelAllowed should return false for DeepSeekTUI+line → MsgAgentTypeNotSupportChannel (line 223-225)
	err := checkAgentProxyKindAllowed(req, model.AgentProxyRouteKindLine, inst)
	if err == nil {
		t.Fatal("DeepSeekTUI agent type should not support line channel")
	}
}

func nonEmpty(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

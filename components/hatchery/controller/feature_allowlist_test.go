package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

// TestFeatureAllowlist_LocalAgent_BlocksAllThreeEndpoints
// 验证 /local-agent/report、/local-agent/sync、/local-agent/commands/ack
// 三个接口在白名单未命中时全部返 403。
//
// 表里有任意一条 type='local-agent' 记录即代表「该功能已启用白名单」，
// 不在白名单里的 identifier 一律拒绝。
func TestFeatureAllowlist_LocalAgent_BlocksAllThreeEndpoints(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.FeatureAllowlist{}); err != nil {
		t.Fatalf("migrate FeatureAllowlist: %v", err)
	}
	ctx := context.Background()

	// 白名单只放行 identifier='allowed-tenant'，但用户 identifier='blocked-tenant'
	if err := model.DB(ctx).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "allowed-tenant",
		Note: "pilot",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	user := model.User{Username: "blocked-user", Role: "user", Identifier: "blocked-tenant"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	cases := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		req     *http.Request
	}{
		{
			name:    "report",
			handler: HandleLocalAgentReport,
			req: reportReq(t, "blocked-user", map[string]any{
				"agent_type": "codebuddy", "local_agent_id": "0123456789abcdef",
			}),
		},
		{
			name:    "sync",
			handler: HandleLocalAgentSync,
			req:     commandsReq(t, "blocked-user"),
		},
		{
			name:    "ack",
			handler: HandleLocalAgentAck,
			req:     jsonAckReq(t, 1, "blocked-user", `{"status":"success"}`),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			c.handler(rr, c.req)
			if rr.Code != http.StatusForbidden {
				t.Errorf("应 403，实际=%d body=%s", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "未开放") {
				t.Errorf("响应体应含「未开放」字样，实际=%s", rr.Body.String())
			}
		})
	}
}

// TestFeatureAllowlist_LocalAgent_AllowsWhenInList
// 白名单命中时，请求应被放行（不会被白名单拦截）。
// 这里只验证「不被 403 拦下」即可，业务路径错误码（如 400 / 404）不算白名单失败。
func TestFeatureAllowlist_LocalAgent_AllowsWhenInList(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.FeatureAllowlist{}); err != nil {
		t.Fatalf("migrate FeatureAllowlist: %v", err)
	}
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "allowed-tenant", Note: "pilot",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	user := model.User{Username: "allowed-user", Role: "user", Identifier: "allowed-tenant"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "allowed-user", map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "0123456789abcdef",
	}))
	if rr.Code == http.StatusForbidden {
		t.Errorf("白名单内不应 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestFeatureAllowlist_LocalAgent_EmptyTableMeansAllOpen
// 表里完全没有 type='local-agent' 的记录时，视为「未启用白名单」，全部租户放行。
// 这是 sane default，避免「忘记加白导致全员被拦」。
func TestFeatureAllowlist_LocalAgent_EmptyTableMeansAllOpen(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.FeatureAllowlist{}); err != nil {
		t.Fatalf("migrate FeatureAllowlist: %v", err)
	}
	ctx := context.Background()

	// 故意写一条不同 type 的记录，确认它不会影响 local-agent 的判定。
	if err := model.DB(ctx).Create(&model.FeatureAllowlist{
		Type: "other-feature", Identifier: "any-tenant",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	user := model.User{Username: "any-user", Role: "user", Identifier: "any-tenant"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "any-user", map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "0123456789abcdef",
	}))
	if rr.Code == http.StatusForbidden {
		t.Errorf("type=local-agent 下表为空应全开，实际 403 body=%s", rr.Body.String())
	}
}

// TestLocalAgent_SiteConfigDisabled_BlocksReporter
// 第 ② 层 SiteConfig 守卫：白名单已通过，但 SiteConfig.LocalAgentEnabled=false → 403。
//
// 这一层是租户管理员可控的全局开关，与 feature_allowlist（跨租户超管）形成 AND 关系。
func TestLocalAgent_SiteConfigDisabled_BlocksReporter(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t) // 默认会打开 LocalAgentEnabled
	disableLocalAgentSiteConfig(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.FeatureAllowlist{}); err != nil {
		t.Fatalf("migrate FeatureAllowlist: %v", err)
	}
	ctx := context.Background()
	// 白名单空表 = 全开（第 ① 层放行），但 LocalAgentEnabled=false（第 ② 层拦下）
	user := model.User{Username: "site-blocked", Role: "user", Identifier: "any-tenant"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "site-blocked", map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "0123456789abcdef",
	}))
	if rr.Code != http.StatusForbidden {
		t.Errorf("SiteConfig 关闭应 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}


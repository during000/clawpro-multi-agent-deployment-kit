package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/common"
)

// decodeSsoProvidersBody 解析 /auth/sso-providers 的响应 body，便于断言。
func decodeSsoProvidersBody(t *testing.T, body []byte) ssoProvidersResponse {
	t.Helper()
	var resp ssoProvidersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, string(body))
	}
	return resp
}

// newReqWithTenant 构造带 TenantSnapshot 的请求。
func newReqWithTenant(accountID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/auth/sso-providers", nil)
	if accountID != "" {
		ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
			OneIDAccountID: accountID,
		})
		req = req.WithContext(ctx)
	}
	return req
}

// TestHandleSsoProviders_NoTenant: 没有租户 ID 时返回空响应（password_enabled=false）。
func TestHandleSsoProviders_NoTenant(t *testing.T) {
	w := httptest.NewRecorder()
	req := newReqWithTenant("")
	HandleSsoProviders(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want=200, got=%d", w.Code)
	}
	got := decodeSsoProvidersBody(t, w.Body.Bytes())
	if len(got.Providers) != 0 || got.PasswordEnabled {
		t.Errorf("want empty providers + password_enabled=false, got=%+v", got)
	}
}

// TestHandleSsoProviders_NoGatewayURL: Gateway 未配置时返回空响应。
func TestHandleSsoProviders_NoGatewayURL(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = ""
	defer func() { GatewayURL = origGW }()

	w := httptest.NewRecorder()
	req := newReqWithTenant("acc-123")
	HandleSsoProviders(w, req)

	got := decodeSsoProvidersBody(t, w.Body.Bytes())
	if len(got.Providers) != 0 || got.PasswordEnabled {
		t.Errorf("want empty providers + password_enabled=false, got=%+v", got)
	}
}

// TestHandleSsoProviders_GatewayNon200: Gateway 返回 5xx 时降级为空响应。
func TestHandleSsoProviders_GatewayNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`upstream broke`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	w := httptest.NewRecorder()
	req := newReqWithTenant("acc-123")
	HandleSsoProviders(w, req)

	got := decodeSsoProvidersBody(t, w.Body.Bytes())
	if len(got.Providers) != 0 || got.PasswordEnabled {
		t.Errorf("want empty providers + password_enabled=false, got=%+v", got)
	}
}

// TestHandleSsoProviders_InvalidJSON: Gateway 返回非 JSON 时降级为空响应。
func TestHandleSsoProviders_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not a json`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	w := httptest.NewRecorder()
	req := newReqWithTenant("acc-123")
	HandleSsoProviders(w, req)

	got := decodeSsoProvidersBody(t, w.Body.Bytes())
	if len(got.Providers) != 0 || got.PasswordEnabled {
		t.Errorf("want empty providers + password_enabled=false, got=%+v", got)
	}
}

// TestHandleSsoProviders_FilterByAliasID: 验证按 aliasId 非空过滤认证源
// （Redirect 类飞书/企微、Delegation 类 OpenLDAP 都保留），同时检测 PASSWORD_V0 → password_enabled=true。
func TestHandleSsoProviders_FilterByAliasID(t *testing.T) {
	oneIDResp := `{
        "next": {
            "type": "SSO_OPTIONS",
            "ssoOptions": {
                "idProviders": [
                    {"id":"100","name":"飞书","logoUrl":"https://logo/feishu.png","uniqueName":"FEISHU_V0","flowType":"Redirect","aliasId":"alias-feishu"},
                    {"id":"101","name":"企业微信","logoUrl":"","uniqueName":"WECOM_V0","flowType":"Redirect","aliasId":"alias-wecom"},
                    {"id":"102","name":"OpenLDAP","logoUrl":"","uniqueName":"OPENLDAP_V1","flowType":"Delegation","aliasId":"alias-ldap"},
                    {"id":"103","name":"无AliasId","logoUrl":"","uniqueName":"BAD","flowType":"Redirect","aliasId":""},
                    {"id":"1","name":"","logoUrl":"","uniqueName":"PASSWORD_V0","flowType":"Delegation","aliasId":""}
                ]
            }
        }
    }`

	var gotTenantQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTenantQuery = r.URL.Query().Get("tenant")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, oneIDResp)
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	w := httptest.NewRecorder()
	req := newReqWithTenant("acc-with-special&char")
	HandleSsoProviders(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want=200, got=%d", w.Code)
	}

	// 验证 tenant 参数正确传递（且经过 url.QueryEscape）
	if gotTenantQuery != "acc-with-special&char" {
		t.Errorf("tenant query 解码后期望 %q, 实际 %q", "acc-with-special&char", gotTenantQuery)
	}

	got := decodeSsoProvidersBody(t, w.Body.Bytes())

	// 只要 aliasId 非空就保留：飞书(Redirect) + 企微(Redirect) + OpenLDAP(Delegation) = 3 个
	// 无 aliasId 的 BAD 和 PASSWORD_V0 不计入卡片
	if len(got.Providers) != 3 {
		t.Fatalf("应保留 3 个 aliasId 非空的 IdP, got=%d, %+v", len(got.Providers), got.Providers)
	}
	if got.Providers[0].AliasID != "alias-feishu" || got.Providers[0].Name != "飞书" {
		t.Errorf("第一个 IdP 解析错误: %+v", got.Providers[0])
	}
	if got.Providers[0].ID != "100" {
		t.Errorf("飞书 id 应为 100, got=%q", got.Providers[0].ID)
	}
	if got.Providers[0].FlowType != "Redirect" {
		t.Errorf("飞书 flow_type 应为 Redirect, got=%q", got.Providers[0].FlowType)
	}
	if got.Providers[1].AliasID != "alias-wecom" {
		t.Errorf("第二个 IdP 解析错误: %+v", got.Providers[1])
	}
	// Delegation 类型但带 aliasId 的 OpenLDAP 也应被保留，且 flow_type 透传
	if got.Providers[2].AliasID != "alias-ldap" || got.Providers[2].UniqueName != "OPENLDAP_V1" {
		t.Errorf("OpenLDAP(Delegation+aliasId) 应被保留: %+v", got.Providers[2])
	}
	if got.Providers[2].ID != "102" {
		t.Errorf("OpenLDAP id 应为 102, got=%q", got.Providers[2].ID)
	}
	if got.Providers[2].FlowType != "Delegation" {
		t.Errorf("OpenLDAP flow_type 应为 Delegation, got=%q", got.Providers[2].FlowType)
	}
	if !got.PasswordEnabled {
		t.Errorf("PASSWORD_V0 存在时 password_enabled 应为 true, got=%+v", got)
	}
}

// TestHandleSsoProviders_NoPasswordOption: 无 PASSWORD_V0 时 password_enabled=false。
func TestHandleSsoProviders_NoPasswordOption(t *testing.T) {
	oneIDResp := `{"next":{"type":"SSO_OPTIONS","ssoOptions":{"idProviders":[
        {"id":"100","name":"飞书","logoUrl":"","uniqueName":"FEISHU_V0","flowType":"Redirect","aliasId":"x"}
    ]}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, oneIDResp)
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	w := httptest.NewRecorder()
	req := newReqWithTenant("acc-1")
	HandleSsoProviders(w, req)

	got := decodeSsoProvidersBody(t, w.Body.Bytes())
	if len(got.Providers) != 1 {
		t.Errorf("应有 1 个 provider, got=%d", len(got.Providers))
	}
	if got.PasswordEnabled {
		t.Errorf("无 PASSWORD_V0 时 password_enabled 应为 false, got=%+v", got)
	}
}

// TestHandleSsoProviders_EmptyIdProviders: idProviders 为空时返回空 providers。
func TestHandleSsoProviders_EmptyIdProviders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"next":{"ssoOptions":{"idProviders":[]}}}`)
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	w := httptest.NewRecorder()
	req := newReqWithTenant("acc-1")
	HandleSsoProviders(w, req)

	got := decodeSsoProvidersBody(t, w.Body.Bytes())
	if got.Providers == nil {
		t.Errorf("Providers 应为非 nil 空切片，避免 JSON 序列化为 null")
	}
	if len(got.Providers) != 0 || got.PasswordEnabled {
		t.Errorf("want empty + password_enabled=false, got=%+v", got)
	}
}

// TestEmptySsoProvidersResponse: 兜底响应字段值断言。
func TestEmptySsoProvidersResponse(t *testing.T) {
	r := emptySsoProvidersResponse()
	if r.Providers == nil {
		t.Errorf("Providers 应为非 nil 空切片")
	}
	if len(r.Providers) != 0 {
		t.Errorf("Providers 应为空, got=%d", len(r.Providers))
	}
	if r.PasswordEnabled {
		t.Errorf("PasswordEnabled 应为 false")
	}
}

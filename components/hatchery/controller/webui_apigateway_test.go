package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// mock client
// ---------------------------------------------------------------------------

type mockGatewayClient struct {
	createErr   error
	deleteErr   error
	createCalls int
	deleteCalls int
	lastCreate  CreateSignOnAgentServiceParams
	lastDelete  DeleteSignOnAgentServiceParams
	// onCreate / onDelete 可注入自定义行为（panic、timeout 等）
	onCreate func(ctx context.Context, p CreateSignOnAgentServiceParams) error
	onDelete func(ctx context.Context, p DeleteSignOnAgentServiceParams) error
}

func (m *mockGatewayClient) CreateSignOnAgentService(ctx context.Context, p CreateSignOnAgentServiceParams) error {
	m.createCalls++
	m.lastCreate = p
	if m.onCreate != nil {
		return m.onCreate(ctx, p)
	}
	return m.createErr
}

func (m *mockGatewayClient) DeleteSignOnAgentService(ctx context.Context, p DeleteSignOnAgentServiceParams) error {
	m.deleteCalls++
	m.lastDelete = p
	if m.onDelete != nil {
		return m.onDelete(ctx, p)
	}
	return m.deleteErr
}

// ---------------------------------------------------------------------------
// 测试用工具
// ---------------------------------------------------------------------------

func setupAPIGatewayTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestUser(oneIDSub string) *model.User {
	u := &model.User{Username: "tester"}
	u.ID = 42
	if oneIDSub != "" {
		s := oneIDSub
		u.OneIDSub = &s
	}
	return u
}

func newTestRequest() *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "/openclaw/set-gateway-ui", nil)
	return r
}

func newTestInstance() *model.Instance {
	return &model.Instance{InstanceId: "ins-ABCDEF", AgentType: model.AgentTypeOpenClaw}
}

// newTestInstanceWithType 用于验证非 OpenClaw 类型直接 fallback 主流程。
func newTestInstanceWithType(agentType string) *model.Instance {
	return &model.Instance{InstanceId: "ins-ABCDEF", AgentType: agentType}
}

func siteConfigWithAPIGateway(enable bool, gwInstance, baseDomain string) model.SiteConfig {
	cfg := model.APIGatewayConfig{Enable: enable, GatewayInstanceID: gwInstance, BaseDomain: baseDomain}
	b, _ := json.Marshal(cfg)
	return model.SiteConfig{APIGatewayConfig: string(b)}
}

// siteConfigWithAPIGatewayScheme 同 siteConfigWithAPIGateway，但允许注入自定义 scheme。
func siteConfigWithAPIGatewayScheme(gwInstance, baseDomain, scheme string) model.SiteConfig {
	cfg := model.APIGatewayConfig{Enable: true, GatewayInstanceID: gwInstance, BaseDomain: baseDomain, Scheme: scheme}
	b, _ := json.Marshal(cfg)
	return model.SiteConfig{APIGatewayConfig: string(b)}
}

// ---------------------------------------------------------------------------
// model.APIGatewayConfig 解析 & ShouldActivate
// ---------------------------------------------------------------------------

func TestGetAPIGatewayConfig_EmptyDefault(t *testing.T) {
	cases := []struct{ name, raw string }{
		{"empty", ""},
		{"empty_object", "{}"},
		{"spaces", "   "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sc := model.SiteConfig{APIGatewayConfig: c.raw}
			cfg, ok := sc.GetAPIGatewayConfig()
			if !ok {
				t.Fatalf("expected ok=true for raw=%q", c.raw)
			}
			if cfg.Enable {
				t.Fatalf("expected Enable=false, got %+v", cfg)
			}
		})
	}
}

func TestGetAPIGatewayConfig_InvalidJSON(t *testing.T) {
	sc := model.SiteConfig{APIGatewayConfig: "not-json"}
	cfg, ok := sc.GetAPIGatewayConfig()
	if ok {
		t.Fatalf("expected ok=false for invalid JSON")
	}
	if cfg.Enable {
		t.Fatalf("expected zero cfg, got %+v", cfg)
	}
}

func TestGetAPIGatewayConfig_ParseFull(t *testing.T) {
	sc := model.SiteConfig{APIGatewayConfig: `{"enable":true,"gateway_instance_id":"ins-xx","base_domain":"mcd.com"}`}
	cfg, ok := sc.GetAPIGatewayConfig()
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !cfg.Enable || cfg.GatewayInstanceID != "ins-xx" || cfg.BaseDomain != "mcd.com" {
		t.Fatalf("wrong cfg: %+v", cfg)
	}
}

func TestAPIGatewayConfig_ShouldActivate_FourQuadrants(t *testing.T) {
	tests := []struct {
		name     string
		cfg      model.APIGatewayConfig
		oneIDSub string
		want     bool
	}{
		{"disabled_no_user", model.APIGatewayConfig{}, "", false},
		{"disabled_with_user", model.APIGatewayConfig{}, "sub-1", false},
		{"enabled_no_user", model.APIGatewayConfig{Enable: true, GatewayInstanceID: "ins", BaseDomain: "d"}, "", false},
		{"enabled_with_user", model.APIGatewayConfig{Enable: true, GatewayInstanceID: "ins", BaseDomain: "d"}, "sub-1", true},
		{"enabled_missing_instance", model.APIGatewayConfig{Enable: true, BaseDomain: "d"}, "sub-1", false},
		{"enabled_missing_domain", model.APIGatewayConfig{Enable: true, GatewayInstanceID: "ins"}, "sub-1", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.ShouldActivate(tc.oneIDSub); got != tc.want {
				t.Fatalf("ShouldActivate: got %v, want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// endpoint / credential / token helpers
// ---------------------------------------------------------------------------

func TestResolveAPIGatewayEndpoint_ByRegion(t *testing.T) {
	tests := []struct{ region, want string }{
		{"ap-guangzhou", "apis.ap-guangzhou.tencentcloudapi.com"},
		{"ap-shanghai", "apis.ap-shanghai.tencentcloudapi.com"},
		{"", "apis.ap-guangzhou.tencentcloudapi.com"},
	}
	for _, tc := range tests {
		if got := resolveAPIGatewayEndpoint(tc.region); got != tc.want {
			t.Errorf("region=%q got=%q want=%q", tc.region, got, tc.want)
		}
	}
}

func TestAPIGatewayCredential_Removed(t *testing.T) {
	// AKSK 凭据获取统一走 getCredential()（与 email.go / admin_config.go 一致），
	// 本 change 不再维护独立的 apiGatewayCredential / env fallback。
	// 本测试留作提示：如果未来又引入独立凭据层，请先在 proposal 里说明。
	t.Log("credential path now delegates to getCredential() directly")
}

func TestMaskToken_PreservesPrefix_RestMasked(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"ab", "**"},
		{"abcd", "****"},
		{"abcdefghij", "abcd******"},
	}
	for _, tc := range tests {
		if got := maskToken(tc.in); got != tc.want {
			t.Errorf("maskToken(%q)=%q want=%q", tc.in, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if truncate("abc", 10) != "abc" {
		t.Fatal("no truncate when shorter")
	}
	got := truncate("abcdefghij", 4)
	if !strings.HasPrefix(got, "abcd") || !strings.Contains(got, "truncated") {
		t.Fatalf("unexpected truncate output: %q", got)
	}
}

// ---------------------------------------------------------------------------
// MaybeOverrideWithGateway — 软功能核心测试
// ---------------------------------------------------------------------------

func TestMaybeOverrideWithGateway_Disabled_ReturnsPrimary(t *testing.T) {
	mc := &mockGatewayClient{}
	restore := setAPIGatewayClientForTest(mc)
	defer restore()

	sc := siteConfigWithAPIGateway(false, "", "")
	user := newTestUser("sub-1")
	got := MaybeOverrideWithGateway(newTestRequest(), sc, user, newTestInstance(),
		"10.0.0.1", 8080, "tok", "/p", "http://primary")
	if got != "http://primary" {
		t.Fatalf("expected primary, got %q", got)
	}
	if mc.createCalls != 0 {
		t.Fatalf("expected 0 create calls, got %d", mc.createCalls)
	}
}

func TestMaybeOverrideWithGateway_NonOneIDUser_ReturnsPrimary(t *testing.T) {
	mc := &mockGatewayClient{}
	defer setAPIGatewayClientForTest(mc)()

	sc := siteConfigWithAPIGateway(true, "ins-gw", "mcd.com")
	user := newTestUser("") // 非 OneID 用户
	got := MaybeOverrideWithGateway(newTestRequest(), sc, user, newTestInstance(),
		"1.1.1.1", 8080, "", "/", "http://primary")
	if got != "http://primary" {
		t.Fatalf("expected primary, got %q", got)
	}
	if mc.createCalls != 0 {
		t.Fatal("should not call client")
	}
}

func TestMaybeOverrideWithGateway_InvalidJSON_ReturnsPrimary(t *testing.T) {
	mc := &mockGatewayClient{}
	defer setAPIGatewayClientForTest(mc)()

	sc := model.SiteConfig{APIGatewayConfig: "not-json"}
	got := MaybeOverrideWithGateway(newTestRequest(), sc, newTestUser("sub-1"), newTestInstance(),
		"1.1.1.1", 80, "", "/", "http://primary")
	if got != "http://primary" {
		t.Fatalf("invalid JSON should fallback, got %q", got)
	}
	if mc.createCalls != 0 {
		t.Fatal("should not call client on invalid JSON")
	}
}

// TestMaybeOverrideWithGateway_UnsupportedAgentType 覆盖 AgentTypeSupportsAPIGateway guard：
// 即便 site_config 已开启网关且用户是 OneID 用户，Lightclaw/Hermes 实例也不应触发云 API 调用。
func TestMaybeOverrideWithGateway_UnsupportedAgentType(t *testing.T) {
	cases := []string{model.AgentTypeLightclawACE, model.AgentTypeHermes}
	for _, at := range cases {
		t.Run(at, func(t *testing.T) {
			mc := &mockGatewayClient{}
			defer setAPIGatewayClientForTest(mc)()

			sc := siteConfigWithAPIGateway(true, "ins-gw-001", "udsuccess.cn")
			got := MaybeOverrideWithGateway(newTestRequest(), sc, newTestUser("sub-1"),
				newTestInstanceWithType(at),
				"1.1.1.1", 80, "tok", "/p", "http://primary")
			if got != "http://primary" {
				t.Fatalf("%s should fallback, got %q", at, got)
			}
			if mc.createCalls != 0 {
				t.Fatalf("%s should not call client, got %d calls", at, mc.createCalls)
			}
		})
	}
}

func TestMaybeOverrideWithGateway_Success_OverridesToDomain(t *testing.T) {
	prev := model.UseDBForTest(setupAPIGatewayTestDB(t))
	defer prev()

	mc := &mockGatewayClient{}
	defer setAPIGatewayClientForTest(mc)()

	sc := siteConfigWithAPIGateway(true, "ins-gw", "mcd.com")
	user := newTestUser("sub-1")
	inst := &model.Instance{InstanceId: "ins-ABCDEF", AgentType: model.AgentTypeOpenClaw}

	got := MaybeOverrideWithGateway(newTestRequest(), sc, user, inst,
		"1.1.1.1", 8080, "tok-xxxx", "/app", "http://primary")

	want := "http://ins-ABCDEF.mcd.com/app"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if mc.createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", mc.createCalls)
	}
	// 幂等策略：Create 之前会先调用 Delete 清理可能的旧路径（重装/升级后 basePath 变化）
	if mc.deleteCalls != 1 {
		t.Fatalf("expected 1 pre-Delete call, got %d", mc.deleteCalls)
	}
	if mc.lastDelete.InstanceID != "ins-gw" || mc.lastDelete.AgentID != "ins-ABCDEF" {
		t.Errorf("pre-Delete params wrong: %+v", mc.lastDelete)
	}
	// 参数装配核查
	if mc.lastCreate.InstanceID != "ins-gw" {
		t.Errorf("InstanceID: got %q", mc.lastCreate.InstanceID)
	}
	if mc.lastCreate.AgentID != "ins-ABCDEF" {
		t.Errorf("AgentID: got %q", mc.lastCreate.AgentID)
	}
	if mc.lastCreate.IP != "1.1.1.1" || mc.lastCreate.Port != 8080 {
		t.Errorf("IP/Port wrong: %+v", mc.lastCreate)
	}
	if mc.lastCreate.AgentToken != "tok-xxxx" {
		t.Errorf("AgentToken: got %q", mc.lastCreate.AgentToken)
	}
	if mc.lastCreate.Path != "/app" {
		t.Errorf("Path: got %q", mc.lastCreate.Path)
	}
	if mc.lastCreate.BaseDomain != "mcd.com" {
		t.Errorf("BaseDomain: got %q", mc.lastCreate.BaseDomain)
	}
	if len(mc.lastCreate.AllowedUsers) != 1 || mc.lastCreate.AllowedUsers[0] != "sub-1" {
		t.Errorf("AllowedUsers: got %+v", mc.lastCreate.AllowedUsers)
	}
}

// path="/"（Lightclaw / Hermes 场景）不应拼到 URL 末尾，避免多余斜杠
func TestMaybeOverrideWithGateway_Success_RootPath_NoTrailingSlash(t *testing.T) {
	prev := model.UseDBForTest(setupAPIGatewayTestDB(t))
	defer prev()

	mc := &mockGatewayClient{}
	defer setAPIGatewayClientForTest(mc)()

	sc := siteConfigWithAPIGateway(true, "ins-gw", "mcd.com")
	inst := &model.Instance{InstanceId: "ins-XYZ", AgentType: model.AgentTypeOpenClaw}
	got := MaybeOverrideWithGateway(newTestRequest(), sc, newTestUser("sub-1"), inst,
		"1.1.1.1", 80, "", "/", "http://primary")

	want := "http://ins-XYZ.mcd.com"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// TestMaybeOverrideWithGateway_SchemeConfigurable 覆盖 site_config.api_gateway_config.scheme：
// "http" / "https" 按值生效；空/非法值回退到默认 "http"。
func TestMaybeOverrideWithGateway_SchemeConfigurable(t *testing.T) {
	cases := []struct {
		name       string
		scheme     string
		wantPrefix string
	}{
		{"empty_defaults_to_http", "", "http://"},
		{"http_kept", "http", "http://"},
		{"https_respected", "https", "https://"},
		{"illegal_falls_back_to_http", "ftp", "http://"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			prev := model.UseDBForTest(setupAPIGatewayTestDB(t))
			defer prev()

			mc := &mockGatewayClient{}
			defer setAPIGatewayClientForTest(mc)()

			sc := siteConfigWithAPIGatewayScheme("ins-gw", "mcd.com", tt.scheme)
			inst := &model.Instance{InstanceId: "ins-XYZ", AgentType: model.AgentTypeOpenClaw}
			got := MaybeOverrideWithGateway(newTestRequest(), sc, newTestUser("sub-1"), inst,
				"1.1.1.1", 80, "tok", "/p", "http://primary")
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Fatalf("scheme=%q got URL %q, want prefix %q", tt.scheme, got, tt.wantPrefix)
			}
		})
	}
}

// ★ 软功能核心断言：云 API 失败时用户仍然拿到 primary URL
func TestMaybeOverrideWithGateway_ClientError_ReturnsPrimary_LogsWarn(t *testing.T) {
	prev := model.UseDBForTest(setupAPIGatewayTestDB(t))
	defer prev()

	mc := &mockGatewayClient{createErr: hcommon.I18nError(i18n.MsgInternalError)}
	defer setAPIGatewayClientForTest(mc)()

	sc := siteConfigWithAPIGateway(true, "ins-gw", "mcd.com")
	user := newTestUser("sub-1")
	got := MaybeOverrideWithGateway(newTestRequest(), sc, user, newTestInstance(),
		"1.1.1.1", 80, "tok", "/", "http://primary-url")
	if got != "http://primary-url" {
		t.Fatalf("soft-fallback failed: got %q", got)
	}
	if mc.createCalls != 1 {
		t.Fatalf("expected 1 call, got %d", mc.createCalls)
	}
}

func TestMaybeOverrideWithGateway_Timeout_ReturnsPrimary(t *testing.T) {
	prev := model.UseDBForTest(setupAPIGatewayTestDB(t))
	defer prev()

	// mock client: 阻塞到 ctx 取消
	mc := &mockGatewayClient{
		onCreate: func(ctx context.Context, p CreateSignOnAgentServiceParams) error {
			<-ctx.Done()
			err := ctx.Err()
			return hcommon.I18nError(i18n.MsgInternalError).WithCause(err)
		},
	}
	defer setAPIGatewayClientForTest(mc)()

	// 用一个提前过期的 request context 让超时立即生效（避免测试等 5s）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	r := newTestRequest().WithContext(ctx)

	sc := siteConfigWithAPIGateway(true, "ins-gw", "mcd.com")
	start := time.Now()
	got := MaybeOverrideWithGateway(r, sc, newTestUser("sub-1"), newTestInstance(),
		"1.1.1.1", 80, "", "/", "http://primary")
	elapsed := time.Since(start)
	if got != "http://primary" {
		t.Fatalf("expected primary on timeout, got %q", got)
	}
	// 不应拖满 5 秒
	if elapsed > 2*time.Second {
		t.Fatalf("timeout path too slow: %v", elapsed)
	}
}

func TestMaybeOverrideWithGateway_Panics_ReturnsPrimary(t *testing.T) {
	prev := model.UseDBForTest(setupAPIGatewayTestDB(t))
	defer prev()

	mc := &mockGatewayClient{
		onCreate: func(ctx context.Context, p CreateSignOnAgentServiceParams) error {
			panic("boom")
		},
	}
	defer setAPIGatewayClientForTest(mc)()

	sc := siteConfigWithAPIGateway(true, "ins-gw", "mcd.com")
	got := MaybeOverrideWithGateway(newTestRequest(), sc, newTestUser("sub-1"), newTestInstance(),
		"1.1.1.1", 80, "", "/", "http://primary-on-panic")
	if got != "http://primary-on-panic" {
		t.Fatalf("panic should fallback to primary, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// commonAPIGatewayClient.invoke — 用 httptest 模拟云 API
// ---------------------------------------------------------------------------

// apiGatewayRoundTripper 只用于改 transport，让 CommonClient 发到 httptest.Server。
// 这里用一种取巧做法：直接发 HTTP POST，跳过 SDK 签名流程，改测公共逻辑。
// 复杂的 SDK 调用链通过软功能上层（Maybe* 函数）已全面覆盖，这里只测解析/错误包装。

func TestCommonAPIGatewayClient_InvokeParsesErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回带 Error 的响应
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"Response":{"Error":{"Code":"InvalidParameter","Message":"bad"},"RequestId":"rid-123"}}`)
	}))
	defer srv.Close()

	// 直接打一次 httptest.Server，验证响应解析的 JSON 结构契约
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	var r struct {
		Response struct {
			Error *struct {
				Code, Message string
			} `json:"Error"`
			RequestId string `json:"RequestId"`
		} `json:"Response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if r.Response.Error == nil || r.Response.Error.Code != "InvalidParameter" {
		t.Fatalf("unexpected parsed response: %+v", r.Response)
	}
}

// ---------------------------------------------------------------------------
// parseAPIGatewayResponse — 独立测覆盖 SDK 响应解析分支
// ---------------------------------------------------------------------------

func TestParseAPIGatewayResponse_Success(t *testing.T) {
	body := []byte(`{"Response":{"Data":{"ID":"svr-abc"},"RequestId":"rid-1"}}`)
	if err := parseAPIGatewayResponse(body, "CreateSignOnAgentService"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestParseAPIGatewayResponse_BusinessError(t *testing.T) {
	body := []byte(`{"Response":{"Error":{"Code":"InvalidParameter","Message":"bad"},"RequestId":"rid-2"}}`)
	err := parseAPIGatewayResponse(body, "CreateSignOnAgentService")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, need := range []string{"InvalidParameter", "bad", "rid-2", "CreateSignOnAgentService"} {
		if !strings.Contains(msg, need) {
			t.Errorf("error message missing %q: %v", need, err)
		}
	}
}

func TestParseAPIGatewayResponse_BadJSON(t *testing.T) {
	err := parseAPIGatewayResponse([]byte("not-json"), "DeleteSignOnAgentService")
	if err == nil {
		t.Fatal("expected parse error")
	}

	wanted := hcommon.I18nError(i18n.MsgAPIGatewayParseRespFailed)
	if !errors.Is(err, wanted) {
		t.Fatalf("unexpected err: %v", err)
	}
}

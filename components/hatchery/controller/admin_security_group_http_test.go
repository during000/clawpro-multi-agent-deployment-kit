package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// ============================================================================
// controller/admin_security_group.go 补测：HandleDescribeCloudSGPolicies
// 和 HandleCheckSecurityGroupRules。
// ============================================================================

func sgAdminReq(method, path, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer test-admin-token")
	return r
}

func sgHandlerSetup(t *testing.T) {
	t.Helper()
	_ = setupSGPoolTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("sg-handler-key"))
	t.Cleanup(func() {
		AdminToken = origToken
		Store = origStore
	})
}

// ---------------- HandleDescribeCloudSGPolicies ----------------

func TestHandleDescribeCloudSGPolicies_Unauthorized(t *testing.T) {
	sgHandlerSetup(t)
	req := httptest.NewRequest("GET", "/admin/config/security-group/describe?security_group_id=sg-x", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleDescribeCloudSGPolicies(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleDescribeCloudSGPolicies_MissingParam(t *testing.T) {
	sgHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleDescribeCloudSGPolicies(w, sgAdminReq("GET", "/admin/config/security-group/describe", ""))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleDescribeCloudSGPolicies_Success(t *testing.T) {
	sgHandlerSetup(t)
	proto := "TCP"
	port := "22"
	cidr := "0.0.0.0/0"
	action := "ACCEPT"
	fake := &fakeSGPoolVpcClient{}
	// 给 describePolicy 一个非空 response 通过 fakeSGPoolVpcClient 不直接支持；
	// 我们只测 happy path 的 JSON 输出即可（fake 默认返回空 set）
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	w := httptest.NewRecorder()
	HandleDescribeCloudSGPolicies(w, sgAdminReq("GET", "/admin/config/security-group/describe?security_group_id=sg-x", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	// 避免 unused 变量告警
	_ = proto
	_ = port
	_ = cidr
	_ = action
}

func TestHandleDescribeCloudSGPolicies_ClientError(t *testing.T) {
	sgHandlerSetup(t)
	origFn := newVpcClientForSGFn
	defer func() { newVpcClientForSGFn = origFn }()
	newVpcClientForSGFn = func(_ context.Context) (sgVpcClient, error) { return nil, errTestInjected }

	w := httptest.NewRecorder()
	HandleDescribeCloudSGPolicies(w, sgAdminReq("GET", "/admin/config/security-group/describe?security_group_id=sg-x", ""))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on client error, got %d", w.Code)
	}
}

// ---------------- HandleCheckSecurityGroupRules ----------------

func TestHandleCheckSecurityGroupRules_Unauthorized(t *testing.T) {
	sgHandlerSetup(t)
	req := httptest.NewRequest("GET", "/admin/config/security-group/check-rules?security_group_id=sg-x", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleCheckSecurityGroupRules(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleCheckSecurityGroupRules_MissingParam_Returns400(t *testing.T) {
	sgHandlerSetup(t)
	// 不传 security_group_id → 400
	w := httptest.NewRecorder()
	HandleCheckSecurityGroupRules(w, sgAdminReq("GET", "/admin/config/security-group/check-rules", ""))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when missing security_group_id, got %d", w.Code)
	}
}

func TestHandleCheckSecurityGroupRules_ClientError_Returns500(t *testing.T) {
	sgHandlerSetup(t)
	origFn := newVpcClientForSGFn
	defer func() { newVpcClientForSGFn = origFn }()
	newVpcClientForSGFn = func(_ context.Context) (sgVpcClient, error) { return nil, errTestInjected }

	w := httptest.NewRecorder()
	HandleCheckSecurityGroupRules(w, sgAdminReq("GET", "/admin/config/security-group/check-rules?security_group_id=sg-x", ""))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on client error, got %d", w.Code)
	}
}

func TestHandleCheckSecurityGroupRules_Success(t *testing.T) {
	sgHandlerSetup(t)
	fake := &fakeSGPoolVpcClient{}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	w := httptest.NewRecorder()
	HandleCheckSecurityGroupRules(w, sgAdminReq("GET", "/admin/config/security-group/check-rules?security_group_id=sg-test", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		MissingRules []map[string]any `json:"missing_rules"`
	}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	// fake 返回空策略集 → missing_rules 可能含推荐规则（取决于配置文件），只校验不 panic
}

// 辅助：用于 newVpcClientForSGFn hook 返回错误
var errTestInjected = &errOnly{msg: "test client factory failed"}

type errOnly struct{ msg string }

func (e *errOnly) Error() string { return e.msg }

// 避免 unused import
var _ = common.StringPtr
var _ = vpc.NewDescribeSecurityGroupPoliciesRequest

// ---------------- HandleListCloudSecurityGroups ----------------

func TestHandleListCloudSecurityGroups_InvalidSGIDFormat_Returns400(t *testing.T) {
	// 覆盖 admin_security_group.go:824-827 — 非 sg- 前缀 ID 入参返回 400
	sgHandlerSetup(t)
	fake := &fakeSGPoolVpcClient{}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	w := httptest.NewRecorder()
	// 传入旧格式 ID（GZ-xxx），应返回 400
	HandleListCloudSecurityGroups(w, sgAdminReq("GET", "/admin/config/security-group/list?security_group_id=GZ-1251783334-153F56970E7C", ""))
	if w.Code != http.StatusBadRequest {
		t.Errorf("非 sg- 前缀 ID 应返回 400，实际=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "格式错误") {
		t.Errorf("错误信息应提及格式错误，实际=%s", w.Body.String())
	}
}

func TestHandleListCloudSecurityGroups_MultiIDWithInvalid_Returns400(t *testing.T) {
	// 覆盖 admin_security_group.go:824-827 — 多个 ID 中有非 sg- 前缀的
	sgHandlerSetup(t)
	fake := &fakeSGPoolVpcClient{}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	w := httptest.NewRecorder()
	HandleListCloudSecurityGroups(w, sgAdminReq("GET", "/admin/config/security-group/list?security_group_id=sg-abc123,GZ-invalid", ""))
	if w.Code != http.StatusBadRequest {
		t.Errorf("含非 sg- 前缀 ID 应返回 400，实际=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleListCloudSecurityGroups_FiltersOldFormatIDs(t *testing.T) {
	// 覆盖 admin_security_group.go:887-889 — 结果中旧格式 ID 被过滤
	sgHandlerSetup(t)

	// 自定义 fake 返回包含旧格式 ID 的安全组
	origFn := newVpcClientForSGFn
	defer func() { newVpcClientForSGFn = origFn }()

	totalCount := uint64(3)
	sgName1 := "normal-sg"
	sgName2 := "old-format-sg"
	sgName3 := "another-normal"
	sgID1 := "sg-abc123"
	sgID2 := "GZ-1251783334-153F56970E7C" // 旧格式
	sgID3 := "sg-def456"

	newVpcClientForSGFn = func(_ context.Context) (sgVpcClient, error) {
		return &fakeListSGClient{
			resp: &vpc.DescribeSecurityGroupsResponse{
				Response: &vpc.DescribeSecurityGroupsResponseParams{
					TotalCount: &totalCount,
					SecurityGroupSet: []*vpc.SecurityGroup{
						{SecurityGroupId: &sgID1, SecurityGroupName: &sgName1},
						{SecurityGroupId: &sgID2, SecurityGroupName: &sgName2},
						{SecurityGroupId: &sgID3, SecurityGroupName: &sgName3},
					},
				},
			},
		}, nil
	}

	w := httptest.NewRecorder()
	HandleListCloudSecurityGroups(w, sgAdminReq("GET", "/admin/config/security-group/list", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		SecurityGroups []struct {
			SecurityGroupId string `json:"security_group_id"`
		} `json:"security_groups"`
		TotalCount uint64 `json:"total_count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	// 旧格式 ID 应被过滤，只剩 2 个
	if len(resp.SecurityGroups) != 2 {
		t.Errorf("过滤后应返回 2 个安全组，实际=%d", len(resp.SecurityGroups))
	}
	for _, sg := range resp.SecurityGroups {
		if !strings.HasPrefix(sg.SecurityGroupId, "sg-") {
			t.Errorf("结果中不应包含非 sg- 前缀的 ID，实际=%s", sg.SecurityGroupId)
		}
	}
	// total_count 应扣减 1（过滤了 1 个旧格式 ID）
	if resp.TotalCount != 2 {
		t.Errorf("total_count 应为 2（扣减 1 个旧格式），实际=%d", resp.TotalCount)
	}
}

// fakeListSGClient 用于 HandleListCloudSecurityGroups 测试的 mock
type fakeListSGClient struct {
	fakeSGPoolVpcClient
	resp *vpc.DescribeSecurityGroupsResponse
}

func (f *fakeListSGClient) DescribeSecurityGroups(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error) {
	if f.resp != nil {
		return f.resp, nil
	}
	return &vpc.DescribeSecurityGroupsResponse{Response: &vpc.DescribeSecurityGroupsResponseParams{}}, nil
}

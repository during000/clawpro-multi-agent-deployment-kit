package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcprofile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// newTestSMHCommonClient 构造一个把 SMH 请求转发到 httptest 服务器的 common.Client，
// 用于在不依赖真实云端的前提下覆盖 updateLibraryInternal 各分支。
func newTestSMHCommonClient(t *testing.T, ts *httptest.Server) *tccommon.Client {
	t.Helper()
	credential := tccommon.NewCredential("AKIDtest-id", "test-secret-key")
	cpf := tcprofile.NewClientProfile()
	cpf.HttpProfile.Endpoint = strings.TrimPrefix(ts.URL, "http://")
	cpf.HttpProfile.Scheme = "HTTP"
	cpf.HttpProfile.ReqMethod = "POST"
	c := tccommon.NewCommonClient(credential, "ap-guangzhou", cpf)
	return c
}

// TestUpdateLibraryInternal_Success 覆盖成功路径：
// httptest 返回带 AccessDomain 的合法响应 → 函数应返回该 AccessDomain，
// 同时校验请求 body 透传了 LibraryId / OmitStorageMeasure / OmitStorageMeasureDuration。
func TestUpdateLibraryInternal_Success(t *testing.T) {
	const wantLibraryId = "lib-12345"
	const wantAccessDomain = "lib-12345.smh.tencentcs.com"

	var (
		gotAction    string
		gotVersion   string
		gotLibraryId string
		gotOmit      bool
		gotDuration  string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAction = r.Header.Get("X-TC-Action")
		gotVersion = r.Header.Get("X-TC-Version")

		body, _ := io.ReadAll(r.Body)
		var payload struct {
			LibraryId                  string `json:"LibraryId"`
			OmitStorageMeasure         bool   `json:"OmitStorageMeasure"`
			OmitStorageMeasureDuration string `json:"OmitStorageMeasureDuration"`
		}
		_ = json.Unmarshal(body, &payload)
		gotLibraryId = payload.LibraryId
		gotOmit = payload.OmitStorageMeasure
		gotDuration = payload.OmitStorageMeasureDuration

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"AccessDomain":"` + wantAccessDomain + `","RequestId":"req-success-1"}}`))
	}))
	defer ts.Close()

	client := newTestSMHCommonClient(t, ts)

	got, err := updateLibraryInternal(client, wantLibraryId)
	if err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	if got != wantAccessDomain {
		t.Errorf("AccessDomain 不匹配: 期望 %q 实际 %q", wantAccessDomain, got)
	}

	// 校验请求头 Action / Version 与 smh.go 中的常量一致
	if gotAction != "UpdateLibraryInternal" {
		t.Errorf("X-TC-Action 应为 UpdateLibraryInternal，实际=%q", gotAction)
	}
	if gotVersion != smhVersion {
		t.Errorf("X-TC-Version 应为 %q，实际=%q", smhVersion, gotVersion)
	}

	// 校验 body 中的关键参数透传正确
	if gotLibraryId != wantLibraryId {
		t.Errorf("LibraryId 不匹配: 期望 %q 实际 %q", wantLibraryId, gotLibraryId)
	}
	if !gotOmit {
		t.Error("OmitStorageMeasure 应为 true")
	}
	if gotDuration != "6m" {
		t.Errorf("OmitStorageMeasureDuration 应为 '6m'，实际=%q", gotDuration)
	}
}

// TestUpdateLibraryInternal_APIError 覆盖响应包含 Error 字段的分支：
// 即便 HTTP 200，只要 Response.Error 非空，就应返回错误，且错误信息包含 RequestId / Code / Message。
func TestUpdateLibraryInternal_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-error-1","Error":{"Code":"InvalidParameter","Message":"libraryId not found"}}}`))
	}))
	defer ts.Close()

	client := newTestSMHCommonClient(t, ts)

	got, err := updateLibraryInternal(client, "lib-not-exist")
	if err == nil {
		t.Fatalf("API Error 时应返回 error，实际=nil，AccessDomain=%q", got)
	}
	if got != "" {
		t.Errorf("出错时 AccessDomain 应为空，实际=%q", got)
	}
	msg := err.Error()
	for _, want := range []string{"UpdateLibraryInternal", "req-error-1", "InvalidParameter", "libraryId not found"} {
		if !strings.Contains(msg, want) {
			t.Errorf("错误信息应包含 %q，实际=%q", want, msg)
		}
	}
}

// TestUpdateLibraryInternal_SendFailed 覆盖 client.Send 失败分支：
// 直接关闭 httptest 服务器使请求无法到达 → 返回 "UpdateLibraryInternal: ..."。
func TestUpdateLibraryInternal_SendFailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	// 立即关闭，让后续 Send 收到连接错误
	ts.Close()

	client := newTestSMHCommonClient(t, ts)

	got, err := updateLibraryInternal(client, "lib-send-fail")
	if err == nil {
		t.Fatalf("Send 失败时应返回 error，实际=nil，AccessDomain=%q", got)
	}
	if got != "" {
		t.Errorf("出错时 AccessDomain 应为空，实际=%q", got)
	}
	if !strings.Contains(err.Error(), "UpdateLibraryInternal") {
		t.Errorf("错误信息应包含 'UpdateLibraryInternal'，实际=%q", err.Error())
	}
}

// TestUpdateLibraryInternal_ParseFailed 覆盖响应体非法 JSON 分支：
// 服务端返回非 JSON 内容 → json.Unmarshal 失败 →
// 函数应返回 "parse UpdateLibraryInternal response: ..."。
func TestUpdateLibraryInternal_ParseFailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-a-valid-json"))
	}))
	defer ts.Close()

	client := newTestSMHCommonClient(t, ts)

	got, err := updateLibraryInternal(client, "lib-parse-fail")
	if err == nil {
		t.Fatalf("响应非法 JSON 时应返回 error，实际=nil，AccessDomain=%q", got)
	}
	if got != "" {
		t.Errorf("出错时 AccessDomain 应为空，实际=%q", got)
	}
	// 注意：tencentcloud-sdk 在 Send 阶段会先尝试解析响应判断是否为腾讯云错误格式，
	// 若 SDK 在更上游报错，则错误会以 "UpdateLibraryInternal:" 前缀返回；
	// 若 SDK 通过则在我们自己的 json.Unmarshal 处报错，前缀为 "parse UpdateLibraryInternal response"。
	// 两者都属于 updateLibraryInternal 在响应解析阶段报错的合法表现。
	msg := err.Error()
	if !strings.Contains(msg, "UpdateLibraryInternal") {
		t.Errorf("错误信息应包含 'UpdateLibraryInternal'，实际=%q", msg)
	}
}

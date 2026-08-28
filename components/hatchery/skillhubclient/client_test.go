package skillhubclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── client.go 测试 ──

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.skillhub.cn", "token-123", 42)
	if c.baseURL != "https://api.skillhub.cn" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
	if c.token != "token-123" {
		t.Errorf("token = %q", c.token)
	}
	if c.orgID != 42 {
		t.Errorf("orgID = %d", c.orgID)
	}
}

func TestClient_DoRequest_SetsHeaders(t *testing.T) {
	var gotAuth, gotCT, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token", 1)
	resp, err := c.doRequest(context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", gotAuth)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q", gotCT)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q", gotAccept)
	}
}

func TestClient_DoRequest_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "token", 0)
	_, err := c.doRequest(context.Background(), http.MethodGet, "/test", nil)
	if err == nil {
		t.Error("expected network error, got nil")
	}
}

// ── FetchOrgInfo 测试 ──

func TestFetchOrgInfo_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Errorf("path = %q, want /api/v1/auth/me", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"user":{"id":539727,"enterprise":{"orgId":17,"orgPublicId":"org-bv6b8qcb","role":"super_admin"}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", 0)
	info, err := c.FetchOrgInfo(context.Background())
	if err != nil {
		t.Fatalf("FetchOrgInfo: %v", err)
	}
	if info.OrgID != 17 {
		t.Errorf("OrgID = %d, want 17", info.OrgID)
	}
	if info.OrgPublicID != "org-bv6b8qcb" {
		t.Errorf("OrgPublicID = %q", info.OrgPublicID)
	}
}

func TestFetchOrgInfo_EmptyOrgID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"user":{"enterprise":{"orgId":0}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", 0)
	_, err := c.FetchOrgInfo(context.Background())
	if err == nil {
		t.Error("expected error for empty orgId, got nil")
	}
	if !strings.Contains(err.Error(), "empty orgId") {
		t.Errorf("error = %q, want 'empty orgId'", err.Error())
	}
}

func TestFetchOrgInfo_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", 0)
	_, err := c.FetchOrgInfo(context.Background())
	if err == nil {
		t.Error("expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain 401, got: %v", err)
	}
}

func TestFetchOrgInfo_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", 0)
	_, err := c.FetchOrgInfo(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// ── ListSkills 测试 ──

func TestListSkills_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证路径包含 orgId
		if !strings.Contains(r.URL.Path, "/orgs/17/skills") {
			t.Errorf("path = %q, want to contain /orgs/17/skills", r.URL.Path)
		}
		// 验证分页参数
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page = %q", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("page_size") != "20" {
			t.Errorf("page_size = %q", r.URL.Query().Get("page_size"))
		}
		// 验证 keyword 被转义
		if kw := r.URL.Query().Get("keyword"); kw != "test&value" {
			t.Errorf("keyword = %q, want 'test&value'", kw)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total":2,"items":[{"id":1,"display_name":"Skill A","slug":"skill-a","version":"1.0.0"},{"id":2,"display_name":"Skill B","slug":"skill-b","version":"2.0.0"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", 17)
	resp, err := c.ListSkills(context.Background(), 1, 20, "test&value")
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("Total = %d, want 2", resp.Total)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("Items count = %d, want 2", len(resp.Items))
	}
	if resp.Items[0].Slug != "skill-a" {
		t.Errorf("Items[0].Slug = %q", resp.Items[0].Slug)
	}
}

func TestListSkills_EmptyKeyword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("keyword") {
			t.Error("keyword should not be present when empty")
		}
		w.Write([]byte(`{"total":0,"items":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", 1)
	resp, err := c.ListSkills(context.Background(), 1, 20, "")
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("Total = %d, want 0", resp.Total)
	}
}

func TestListSkills_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", 1)
	_, err := c.ListSkills(context.Background(), 1, 20, "")
	if err == nil {
		t.Error("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain 500: %v", err)
	}
}

func TestListSkills_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`invalid`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", 1)
	_, err := c.ListSkills(context.Background(), 1, 20, "")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestListSkills_SpecialCharsInKeyword(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Write([]byte(`{"total":0,"items":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", 1)
	_, err := c.ListSkills(context.Background(), 1, 20, "hello world&foo=bar")
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	// keyword 应被转义，空格→%20，&→%26，=→%3D
	if !strings.Contains(capturedQuery, "keyword=hello") {
		t.Errorf("raw query = %q, expected escaped keyword", capturedQuery)
	}
	// 原始 &foo=bar 不应作为额外参数出现
	if strings.Contains(capturedQuery, "foo=bar") {
		t.Errorf("raw query = %q, keyword not properly escaped", capturedQuery)
	}
}

// ── getAccessToken 间接测试（通过 ListSkills 验证 token 注入）──

func TestClient_TokenInjectedInRequest(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"total":0,"items":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-special-token", 1)
	_, err := c.ListSkills(context.Background(), 1, 10, "")
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if receivedAuth != "Bearer my-special-token" {
		t.Errorf("Authorization = %q, want 'Bearer my-special-token'", receivedAuth)
	}
}

// ── readBody helper test ──

func TestReadBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"hello":"world"}`))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/test")
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if result["hello"] != "world" {
		t.Errorf("hello = %q", result["hello"])
	}
}

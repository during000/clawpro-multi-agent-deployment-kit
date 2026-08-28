package tdaimemorysdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- NewClient 配置测试 ---

func TestNewClient_MinimalConfig(t *testing.T) {
	c, err := NewClient(Config{
		SecretID:  "test-id",
		SecretKey: "test-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client should not be nil")
	}
	if c.service != DefaultService {
		t.Errorf("service = %q, want %q", c.service, DefaultService)
	}
	if c.version != DefaultVersion {
		t.Errorf("version = %q, want %q", c.version, DefaultVersion)
	}
}

func TestNewClient_EmptySecretID(t *testing.T) {
	_, err := NewClient(Config{SecretID: "", SecretKey: "key"})
	if err != ErrEmptySecretID {
		t.Errorf("expected ErrEmptySecretID, got %v", err)
	}
}

func TestNewClient_EmptySecretKey(t *testing.T) {
	_, err := NewClient(Config{SecretID: "id", SecretKey: ""})
	if err != ErrEmptySecretKey {
		t.Errorf("expected ErrEmptySecretKey, got %v", err)
	}
}

func TestNewClient_CustomConfig(t *testing.T) {
	c, err := NewClient(Config{
		SecretID:  "id",
		SecretKey: "key",
		Service:   "custom-service",
		Version:   "2099-01-01",
		Region:    "ap-tokyo",
		Endpoint:  "custom.endpoint.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.service != "custom-service" {
		t.Errorf("service = %q, want custom-service", c.service)
	}
	if c.version != "2099-01-01" {
		t.Errorf("version = %q, want 2099-01-01", c.version)
	}
}

func TestNewClient_WithToken(t *testing.T) {
	c, err := NewClient(Config{
		SecretID:  "id",
		SecretKey: "key",
		Token:     "sts-token-xxx",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client should not be nil with STS token")
	}
}

// --- APIError 测试 ---

func TestAPIError_Error(t *testing.T) {
	e := &APIError{Code: "InvalidAction", Message: "not found", RequestID: "req-123"}
	s := e.Error()
	if s == "" {
		t.Fatal("error string should not be empty")
	}
	// 应包含 code、message、request_id
	for _, sub := range []string{"InvalidAction", "not found", "req-123"} {
		if !contains(s, sub) {
			t.Errorf("error string %q should contain %q", s, sub)
		}
	}
}

func TestAPIError_NoRequestID(t *testing.T) {
	e := &APIError{Code: "Err", Message: "msg"}
	s := e.Error()
	if contains(s, "request_id") {
		t.Error("should not contain request_id when empty")
	}
}

func TestAPIError_Nil(t *testing.T) {
	var e *APIError
	if e.Error() != "<nil>" {
		t.Errorf("nil APIError should return <nil>, got %q", e.Error())
	}
}

// --- decodeActionResponse 测试 ---

func TestDecode_EmptyBody(t *testing.T) {
	_, err := decodeActionResponse(nil, nil)
	if err == nil {
		t.Error("empty body should error")
	}
}

func TestDecode_MissingResponseField(t *testing.T) {
	_, err := decodeActionResponse([]byte(`{}`), nil)
	if err == nil {
		t.Error("missing Response field should error")
	}
}

func TestDecode_APIError(t *testing.T) {
	body := `{"Response":{"Error":{"Code":"TestError","Message":"test msg"},"RequestId":"req-001"}}`
	_, err := decodeActionResponse([]byte(body), nil)
	if err == nil {
		t.Fatal("expected API error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "TestError" {
		t.Errorf("code = %q, want TestError", apiErr.Code)
	}
	if apiErr.RequestID != "req-001" {
		t.Errorf("request_id = %q, want req-001", apiErr.RequestID)
	}
}

func TestDecode_Success(t *testing.T) {
	body := `{"Response":{"TotalCount":5,"RequestId":"req-002"}}`
	type resp struct {
		TotalCount int    `json:"TotalCount"`
		RequestID  string `json:"RequestId"`
	}
	var out resp
	reqID, err := decodeActionResponse([]byte(body), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqID != "req-002" {
		t.Errorf("requestID = %q, want req-002", reqID)
	}
	if out.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5", out.TotalCount)
	}
}

func TestDecode_InvalidJSON(t *testing.T) {
	_, err := decodeActionResponse([]byte(`not-json`), nil)
	if err == nil {
		t.Error("invalid JSON should error")
	}
}

// --- Model 序列化测试 ---

func TestMemoryProModels_Serialization(t *testing.T) {
	req := CreateMemoryProInstanceRequest{
		VpcId:            "vpc-test",
		SubnetId:         "subnet-test",
		SecurityGroupIds: []string{"sg-001"},
		MemoryLimit:      1000,
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded map[string]any
	json.Unmarshal(b, &decoded)

	if decoded["VpcId"] != "vpc-test" {
		t.Error("VpcId not serialized correctly")
	}
	if decoded["MemoryLimit"].(float64) != 1000 {
		t.Error("MemoryLimit not serialized correctly")
	}
}

func TestCreateMemSpaceResponse_Serialization(t *testing.T) {
	resp := CreateMemSpaceResponse{
		SpaceId:        "space-001",
		DatabaseName:   "db-001",
		Vip:            "10.0.0.1",
		Port:           3306,
		ApiKey:         "key-xxx",
		Account:        "root",
		EmbeddingModel: "qwen3-0.6b",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !contains(string(b), "space-001") {
		t.Error("SpaceId missing in JSON")
	}
	if !contains(string(b), "qwen3-0.6b") {
		t.Error("EmbeddingModel missing in JSON")
	}
}

func TestDescribeMemSpaceRecordRequest_Fields(t *testing.T) {
	offset := 10
	limit := 20
	req := DescribeMemSpaceRecordRequest{
		SpaceId:        "sp-001",
		RecordType:     "memory/persona",
		OrderDirection: "DESC",
		StartTime:      "2026-04-14 20:00:00",
		EndTime:        "2026-04-14 21:00:00",
		Offset:         &offset,
		Limit:          &limit,
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["RecordType"] != "memory/persona" {
		t.Error("RecordType not set correctly")
	}
	if m["OrderDirection"] != "DESC" {
		t.Error("OrderDirection not set correctly")
	}
}

// --- Do() with mock server ---

func TestDo_EmptyAction(t *testing.T) {
	c, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	_, err := c.Do(nil, "", nil, nil)
	if err != ErrEmptyAction {
		t.Errorf("expected ErrEmptyAction, got %v", err)
	}
}

func TestDo_NilClient(t *testing.T) {
	var c *Client
	_, err := c.Do(nil, "Test", nil, nil)
	if err == nil {
		t.Error("nil client should error")
	}
}

// --- mock HTTP server helper ---

func newMockServer(t *testing.T, respBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(respBody))
	}))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

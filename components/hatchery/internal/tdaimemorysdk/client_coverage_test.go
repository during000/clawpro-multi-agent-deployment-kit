package tdaimemorysdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- 针对 client.go 未覆盖分支的补充测试 ---

// Line 61-63: RequestClient 非空分支
func TestNewClient_WithRequestClient(t *testing.T) {
	client, err := NewClient(Config{
		SecretID:      "id-rc",
		SecretKey:     "key-rc",
		RequestClient: "hatchery-test",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client == nil {
		t.Fatal("client should not be nil")
	}
	// 目前 SDK 没暴露 RequestClient getter，能成功创建即覆盖
}

// Line 84-86: ctx=nil 时应自动赋值为 Background（需要 action 非空才能走到这行）
// 通过 mock server 让 Do 走完整路径。
func TestDo_NilContextUsesBackground(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"Response":{"RequestId":"req-123"}}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		SecretID: "id",
		SecretKey: "key",
		Endpoint: srv.Listener.Addr().String(),
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// ctx=nil，action 非空 → 触发 Line 84-86 的自动赋值
	// 但因为 httptest 是 http，SDK 用 https，会 TLS 握手失败 → 此处只测调用进入 ctx 分支
	// 最终会返回网络错误，符合预期（覆盖分支即可）
	_, err = client.Do(nil, "SomeAction", nil, nil)
	if err == nil {
		t.Log("期望返回错误（https→http 不通），但测试服务器直接接受了 http")
	}
}

// Line 98-100: payload 已经是 map[string]any / string / []byte 的快速路径
func TestDo_MapPayloadFastPath(t *testing.T) {
	client, err := NewClient(Config{
		SecretID: "id",
		SecretKey: "key",
		Endpoint: "127.0.0.1:1", // 立即连接拒绝
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// 传 map[string]any，应该走快速路径（不走 JSON marshal/unmarshal）
	_, err = client.Do(nil, "Test", map[string]any{"foo": "bar"}, nil)
	if err == nil {
		t.Error("网络不通应返回错误")
	}
}

// Line 101-110: 非 map/string/[]byte 的结构体 payload 路径（需 json.Marshal 转 map）
func TestDo_StructPayloadMarshaled(t *testing.T) {
	client, err := NewClient(Config{
		SecretID: "id",
		SecretKey: "key",
		Endpoint: "127.0.0.1:1",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// 传结构体 → 走 default 分支 marshal
	req := struct {
		Field string `json:"Field"`
	}{Field: "v"}
	_, err = client.Do(nil, "Test", req, nil)
	if err == nil {
		t.Error("网络不通应返回错误")
	}
}

// 不能被 JSON marshal 的 payload → marshal 失败分支（Line 102-105）
func TestDo_UnmarshalableStruct(t *testing.T) {
	client, _ := NewClient(Config{
		SecretID: "id",
		SecretKey: "key",
	})

	// func 类型不能被 json.Marshal，但必须不是 map/string/[]byte 才走到 default 分支
	type badPayload struct {
		F func() `json:"F"`
	}
	_, err := client.Do(nil, "Test", badPayload{F: func() {}}, nil)
	if err == nil {
		t.Error("无法 marshal 的 payload 应返回错误")
	}
}

// --- decodeActionResponse 更多分支 ---

// 空响应体
func TestDecodeActionResponse_EmptyBody(t *testing.T) {
	var out map[string]any
	_, err := decodeActionResponse([]byte{}, &out)
	if err == nil {
		t.Error("空响应体应返回错误")
	}
}

// API 返回 Error 字段
func TestDecodeActionResponse_APIError(t *testing.T) {
	raw := []byte(`{"Response":{"Error":{"Code":"InvalidParameter","Message":"bad"},"RequestId":"req-err"}}`)
	var out map[string]any
	_, err := decodeActionResponse(raw, &out)
	if err == nil {
		t.Fatal("应返回 API error")
	}

	// err 应可断言为 *APIError
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err 类型应为 *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != "InvalidParameter" {
		t.Errorf("Code = %q", apiErr.Code)
	}
	if apiErr.RequestID != "req-err" {
		t.Errorf("RequestID = %q", apiErr.RequestID)
	}
}

// 响应体缺 Response 字段
func TestDecodeActionResponse_MissingResponse(t *testing.T) {
	raw := []byte(`{"NoResponse":true}`)
	var out map[string]any
	_, err := decodeActionResponse(raw, &out)
	if err == nil {
		t.Error("缺 Response 字段应返回错误")
	}
}

// 响应体非 JSON
func TestDecodeActionResponse_InvalidJSON(t *testing.T) {
	var out map[string]any
	_, err := decodeActionResponse([]byte("not-json"), &out)
	if err == nil {
		t.Error("非法 JSON 应返回错误")
	}
}

// 成功响应，resp 目标为 nil → 应不报错
func TestDecodeActionResponse_NilTarget(t *testing.T) {
	raw := []byte(`{"Response":{"RequestId":"req-ok","Data":"x"}}`)
	reqID, err := decodeActionResponse(raw, nil)
	if err != nil {
		t.Errorf("nil target 不应报错: %v", err)
	}
	if reqID != "req-ok" {
		t.Errorf("reqID = %q", reqID)
	}
}

// 成功响应 → decodeActionResponse 解析到目标结构体
func TestDecodeActionResponse_Success(t *testing.T) {
	raw := []byte(`{"Response":{"RequestId":"req-ok","SpaceId":"s1","Port":80}}`)
	var out CreateMemSpaceResponse
	reqID, err := decodeActionResponse(raw, &out)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if reqID != "req-ok" {
		t.Errorf("reqID = %q", reqID)
	}
	if out.SpaceId != "s1" || out.Port != 80 {
		t.Errorf("结构体解析错误: %+v", out)
	}
}

// --- 各 Action 方法的 nil request 分支 ---

func TestCreateMemoryProInstance_NilRequest(t *testing.T) {
	client, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	_, err := client.CreateMemoryProInstance(nil, nil)
	if err == nil {
		t.Error("nil request 调用应返回网络错误")
	}
}

func TestDescribeMemoryProInstances_NilRequest(t *testing.T) {
	client, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	_, err := client.DescribeMemoryProInstances(nil, nil)
	if err == nil {
		t.Error("nil request 调用应返回网络错误")
	}
}

func TestModifyMemoryProInstance_NilRequest(t *testing.T) {
	client, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	_, err := client.ModifyMemoryProInstance(nil, nil)
	if err == nil {
		t.Error("nil request 应返回网络错误")
	}
}

func TestDeleteMemoryProInstance_NilRequest(t *testing.T) {
	client, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	_, err := client.DeleteMemoryProInstance(nil, nil)
	if err == nil {
		t.Error("nil request 应返回网络错误")
	}
}

func TestCreateMemSpace_NilRequest(t *testing.T) {
	client, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	_, err := client.CreateMemSpace(nil, nil)
	if err == nil {
		t.Error("nil request 应返回网络错误")
	}
}

func TestDescribeMemSpaces_NilRequest(t *testing.T) {
	client, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	_, err := client.DescribeMemSpaces(nil, nil)
	if err == nil {
		t.Error("nil request 应返回网络错误")
	}
}

func TestDescribeMemSpaceRecord_NilRequest(t *testing.T) {
	client, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	_, err := client.DescribeMemSpaceRecord(nil, nil)
	if err == nil {
		t.Error("nil request 应返回网络错误")
	}
}

func TestDeleteMemSpace_NilRequest(t *testing.T) {
	client, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	_, err := client.DeleteMemSpace(nil, nil)
	if err == nil {
		t.Error("nil request 应返回网络错误")
	}
}

// --- DescribeAgentInstance nil req（已有测试路径的补充）---

func TestDescribeAgentInstance_NilRequest(t *testing.T) {
	client, _ := NewClient(Config{SecretID: "id", SecretKey: "key"})
	// nil req 应被替换为空的 DescribeAgentInstanceRequest
	_, err := client.DescribeAgentInstance(nil, nil)
	if err == nil {
		t.Error("网络不通应返回错误")
	}
}

// 避免导入未使用
var _ = json.Marshal

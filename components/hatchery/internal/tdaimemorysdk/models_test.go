package tdaimemorysdk

import (
	"encoding/json"
	"reflect"
	"testing"
)

// 验证：结构体字段 JSON tag 正确，可 marshal/unmarshal 往返不丢字段。
// 注：这些文件是纯类型声明，Go coverage 不计入语句（0/0），
// 但 round-trip 测试能防止字段 tag 写错导致 API 契约变更时才发现。

// --- config.go: Config ---

func TestConfig_JSONRoundTrip(t *testing.T) {
	// Config 虽没 json tag，但 struct 字段默认用字段名作为 JSON key
	src := Config{
		SecretID:      "AKID123",
		SecretKey:     "sk-456",
		Token:         "tok-789",
		Region:        "ap-beijing",
		Endpoint:      "tdai.tencentcloudapi.com",
		Service:       "tdai",
		Version:       "2025-07-17",
		RequestClient: "hatchery",
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var dst Config
	if err := json.Unmarshal(data, &dst); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// 跳过 Timeout（time.Duration 走 int64 序列化）
	src.Timeout = 0
	dst.Timeout = 0
	if !reflect.DeepEqual(src, dst) {
		t.Errorf("round-trip mismatch: src=%+v dst=%+v", src, dst)
	}
}

func TestConfig_DefaultsConstant(t *testing.T) {
	if DefaultService != "tdai" {
		t.Errorf("DefaultService = %q, want tdai", DefaultService)
	}
	if DefaultVersion == "" {
		t.Error("DefaultVersion 不应为空")
	}
	if DefaultRegion != "ap-guangzhou" {
		t.Errorf("DefaultRegion = %q, want ap-guangzhou", DefaultRegion)
	}
	if DefaultTimeout <= 0 {
		t.Error("DefaultTimeout 应为正值")
	}
}

// --- models.go: DescribeAgentInstance ---

func TestDescribeAgentInstance_JSONRoundTrip(t *testing.T) {
	reqSrc := DescribeAgentInstanceRequest{InstanceID: "ins-abc"}
	data, _ := json.Marshal(reqSrc)
	if string(data) != `{"InstanceId":"ins-abc"}` {
		t.Errorf("request JSON key 不对: %s", data)
	}
	var reqDst DescribeAgentInstanceRequest
	if err := json.Unmarshal(data, &reqDst); err != nil || reqDst.InstanceID != "ins-abc" {
		t.Errorf("request round-trip: %v, %+v", err, reqDst)
	}

	respSrc := DescribeAgentInstanceResponse{
		AgentInstance: map[string]any{"Name": "test"},
		RequestID:     "req-1",
	}
	data, _ = json.Marshal(respSrc)
	// RequestId 应使用 RequestId 这个 key
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	if _, has := m["RequestId"]; !has {
		t.Errorf("RequestId key 应存在，got keys: %v", m)
	}
}

// --- memory_pro_models.go: Memory Pro 实例相关 ---

func TestResourceTag_JSON(t *testing.T) {
	src := ResourceTag{TagKey: "k", TagValue: "v"}
	data, _ := json.Marshal(src)
	var m map[string]string
	_ = json.Unmarshal(data, &m)
	if m["TagKey"] != "k" || m["TagValue"] != "v" {
		t.Errorf("ResourceTag tag 错误: %s", data)
	}
}

func TestCreateMemoryProInstanceRequest_JSON(t *testing.T) {
	src := CreateMemoryProInstanceRequest{
		VpcId:            "vpc-1",
		SubnetId:         "subnet-1",
		SecurityGroupIds: []string{"sg-1"},
		MemoryLimit:      1000,
		ResourceTags:     []ResourceTag{{TagKey: "env", TagValue: "prod"}},
	}
	data, _ := json.Marshal(src)
	var dst CreateMemoryProInstanceRequest
	if err := json.Unmarshal(data, &dst); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(src, dst) {
		t.Errorf("round-trip 不一致: src=%+v dst=%+v", src, dst)
	}
}

func TestCreateMemoryProInstanceResponse_JSON(t *testing.T) {
	// API 返回格式，用原始 JSON 验证我们能正确解析
	raw := `{"MemoryProId":"mp-1","VDBInstanceId":"vdb-1","RequestId":"req-a"}`
	var resp CreateMemoryProInstanceResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.MemoryProId != "mp-1" {
		t.Errorf("MemoryProId = %q", resp.MemoryProId)
	}
	if resp.VDBInstanceId != "vdb-1" {
		t.Errorf("VDBInstanceId = %q", resp.VDBInstanceId)
	}
	if resp.RequestID != "req-a" {
		t.Errorf("RequestID = %q, 应解析 RequestId key", resp.RequestID)
	}
}

func TestMemoryProInstanceInfo_JSON(t *testing.T) {
	raw := `{
		"MemoryProId":"mp-1",
		"VDBInstanceId":"vdb-1",
		"Status":"online",
		"MemoryLimit":500,
		"MemoryUsed":3,
		"AppId":123,
		"Uin":"700000000",
		"CreatedAt":"2026-04-01"
	}`
	var info MemoryProInstanceInfo
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info.MemoryProId != "mp-1" || info.Status != "online" {
		t.Errorf("字段 tag 错误: %+v", info)
	}
	if info.MemoryLimit != 500 || info.MemoryUsed != 3 {
		t.Errorf("数值字段: limit=%d used=%d", info.MemoryLimit, info.MemoryUsed)
	}
	if info.AppId != 123 {
		t.Errorf("AppId = %d", info.AppId)
	}
}

func TestDescribeMemoryProInstancesResponse_JSON(t *testing.T) {
	raw := `{
		"TotalCount":1,
		"Items":[{"MemoryProId":"mp-1","Status":"online"}],
		"RequestId":"req-b"
	}`
	var resp DescribeMemoryProInstancesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.TotalCount != 1 || len(resp.Items) != 1 || resp.Items[0].MemoryProId != "mp-1" {
		t.Errorf("字段解析错误: %+v", resp)
	}
	if resp.RequestID != "req-b" {
		t.Errorf("RequestID = %q", resp.RequestID)
	}
}

func TestModifyMemoryProInstance_JSON(t *testing.T) {
	limit := 2000
	src := ModifyMemoryProInstanceRequest{MemoryProId: "mp-1", MemoryLimit: &limit}
	data, _ := json.Marshal(src)
	var dst ModifyMemoryProInstanceRequest
	if err := json.Unmarshal(data, &dst); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dst.MemoryProId != "mp-1" || dst.MemoryLimit == nil || *dst.MemoryLimit != 2000 {
		t.Errorf("ModifyMemoryPro 序列化错误: %+v", dst)
	}

	// MemoryLimit 为 nil → 应被 omitempty 省略
	src2 := ModifyMemoryProInstanceRequest{MemoryProId: "mp-2"}
	data, _ = json.Marshal(src2)
	if string(data) != `{"MemoryProId":"mp-2"}` {
		t.Errorf("omitempty 未生效: %s", data)
	}
}

func TestDeleteMemoryProInstance_JSON(t *testing.T) {
	src := DeleteMemoryProInstanceRequest{MemoryProId: "mp-1"}
	data, _ := json.Marshal(src)
	var dst DeleteMemoryProInstanceRequest
	_ = json.Unmarshal(data, &dst)
	if dst.MemoryProId != "mp-1" {
		t.Errorf("delete req round-trip: %+v", dst)
	}

	// ServiceId omitempty
	if string(data) != `{"MemoryProId":"mp-1"}` {
		t.Errorf("ServiceId omitempty 未生效: %s", data)
	}
}

// --- memory_pro_models.go: MemSpace 相关 ---

func TestCreateMemSpaceResponse_JSON(t *testing.T) {
	raw := `{
		"MemoryProId":"mp-1",
		"SpaceId":"space-1",
		"DatabaseName":"db-1",
		"CollectionNames":["c1","c2"],
		"EmbeddingModel":"bge-large",
		"Vip":"10.0.0.1",
		"Port":8080,
		"Account":"user1",
		"ApiKey":"key1",
		"RequestId":"req-c"
	}`
	var resp CreateMemSpaceResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.SpaceId != "space-1" || resp.DatabaseName != "db-1" || resp.Port != 8080 {
		t.Errorf("CreateMemSpace 字段错误: %+v", resp)
	}
	if resp.EmbeddingModel != "bge-large" {
		t.Errorf("EmbeddingModel = %q", resp.EmbeddingModel)
	}
	if len(resp.CollectionNames) != 2 {
		t.Errorf("CollectionNames len = %d", len(resp.CollectionNames))
	}
}

func TestDescribeMemSpaces_JSON(t *testing.T) {
	req := DescribeMemSpacesRequest{
		MemoryProId: "mp-1",
		SpaceIds:    []string{"s1", "s2"},
	}
	data, _ := json.Marshal(req)
	var dst DescribeMemSpacesRequest
	_ = json.Unmarshal(data, &dst)
	if !reflect.DeepEqual(req, dst) {
		t.Errorf("req round-trip: %+v vs %+v", req, dst)
	}

	// 响应
	raw := `{
		"TotalCount":1,
		"Items":[{"SpaceId":"s1","DatabaseName":"db-s1","Port":80}],
		"RequestId":"req-d"
	}`
	var resp DescribeMemSpacesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Items[0].SpaceId != "s1" || resp.Items[0].Port != 80 {
		t.Errorf("MemorySpaceInfo 字段错误: %+v", resp.Items[0])
	}
}

func TestDescribeMemSpaceRecord_JSON(t *testing.T) {
	limit := 100
	offset := 10
	req := DescribeMemSpaceRecordRequest{
		SpaceId:        "s1",
		RecordType:     "memory",
		Offset:         &offset,
		Limit:          &limit,
		OrderDirection: "DESC",
		StartTime:      "2026-04-14 20:00:00",
		EndTime:        "2026-04-14 21:00:00",
		Output:         []string{"text", "timestamp"},
	}
	data, _ := json.Marshal(req)
	var dst DescribeMemSpaceRecordRequest
	if err := json.Unmarshal(data, &dst); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dst.SpaceId != "s1" || dst.RecordType != "memory" {
		t.Errorf("必填字段错误: %+v", dst)
	}
	if dst.Offset == nil || *dst.Offset != 10 || dst.Limit == nil || *dst.Limit != 100 {
		t.Errorf("分页字段错误: offset=%v limit=%v", dst.Offset, dst.Limit)
	}

	// 响应
	raw := `{
		"TotalCount":2,
		"Documents":[{"id":"d1","text":"hello"},{"id":"d2","text":"world"}],
		"RequestId":"req-e"
	}`
	var resp DescribeMemSpaceRecordResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.TotalCount != 2 || len(resp.Documents) != 2 {
		t.Errorf("records resp 错误: %+v", resp)
	}
}

func TestVDBDocument_JSON(t *testing.T) {
	src := VDBDocument{
		Documents: []map[string]any{{"id": "d1", "text": "t1"}},
	}
	data, _ := json.Marshal(src)
	var dst VDBDocument
	_ = json.Unmarshal(data, &dst)
	if len(dst.Documents) != 1 || dst.Documents[0]["id"] != "d1" {
		t.Errorf("VDBDocument round-trip: %+v", dst)
	}
}

func TestDeleteMemSpace_JSON(t *testing.T) {
	src := DeleteMemSpaceRequest{SpaceId: "space-del"}
	data, _ := json.Marshal(src)
	if string(data) != `{"SpaceId":"space-del"}` {
		t.Errorf("SpaceId tag 错误: %s", data)
	}
}

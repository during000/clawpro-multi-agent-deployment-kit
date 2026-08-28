package tdaimemorysdk

import (
	"testing"
	"time"
)

// --- config.go: Config.Validate / WithDefaults ---

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr error
	}{
		{"nil", nil, ErrEmptySecretID},
		{"missing secret_id", &Config{SecretKey: "k"}, ErrEmptySecretID},
		{"missing secret_key", &Config{SecretID: "id"}, ErrEmptySecretKey},
		{"valid", &Config{SecretID: "id", SecretKey: "k"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.Validate(); got != tt.wantErr {
				t.Errorf("Validate() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	// 空值应被填为默认
	cfg := Config{SecretID: "id", SecretKey: "k"}.WithDefaults()
	if cfg.Region != DefaultRegion {
		t.Errorf("Region = %q, want %q", cfg.Region, DefaultRegion)
	}
	if cfg.Service != DefaultService {
		t.Errorf("Service = %q, want %q", cfg.Service, DefaultService)
	}
	if cfg.Version != DefaultVersion {
		t.Errorf("Version = %q, want %q", cfg.Version, DefaultVersion)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
	}

	// 原值应被保留
	cfg2 := Config{
		SecretID:  "id",
		SecretKey: "k",
		Region:    "ap-beijing",
		Service:   "custom",
		Version:   "2020-01-01",
		Timeout:   5 * time.Second,
	}.WithDefaults()
	if cfg2.Region != "ap-beijing" {
		t.Errorf("Region 被覆盖: %q", cfg2.Region)
	}
	if cfg2.Service != "custom" {
		t.Errorf("Service 被覆盖: %q", cfg2.Service)
	}
	if cfg2.Version != "2020-01-01" {
		t.Errorf("Version 被覆盖: %q", cfg2.Version)
	}
	if cfg2.Timeout != 5*time.Second {
		t.Errorf("Timeout 被覆盖: %v", cfg2.Timeout)
	}
}

// --- models.go: DescribeAgentInstanceRequest / Response ---

func TestDescribeAgentInstanceRequest_Validate(t *testing.T) {
	if (&DescribeAgentInstanceRequest{}).Validate() == nil {
		t.Error("空请求应失败")
	}
	var nilReq *DescribeAgentInstanceRequest
	if nilReq.Validate() == nil {
		t.Error("nil 请求应失败")
	}
	if (&DescribeAgentInstanceRequest{InstanceID: "ins-x"}).Validate() != nil {
		t.Error("有 InstanceID 应通过")
	}
}

func TestDescribeAgentInstanceResponse_IsEmpty(t *testing.T) {
	var nilResp *DescribeAgentInstanceResponse
	if !nilResp.IsEmpty() {
		t.Error("nil 应被视为空")
	}
	if !(&DescribeAgentInstanceResponse{}).IsEmpty() {
		t.Error("无 AgentInstance 应为空")
	}
	resp := &DescribeAgentInstanceResponse{AgentInstance: map[string]any{"InstanceId": "ins-x"}}
	if resp.IsEmpty() {
		t.Error("有 AgentInstance 不应为空")
	}
}

// --- memory_pro_models.go: 多个 Request.Validate / MemoryProInstanceInfo ---

func TestCreateMemoryProInstanceRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  *CreateMemoryProInstanceRequest
		ok   bool
	}{
		{"nil", nil, false},
		{"missing vpc", &CreateMemoryProInstanceRequest{SubnetId: "s", SecurityGroupIds: []string{"g"}, MemoryLimit: 1}, false},
		{"missing subnet", &CreateMemoryProInstanceRequest{VpcId: "v", SecurityGroupIds: []string{"g"}, MemoryLimit: 1}, false},
		{"missing sg", &CreateMemoryProInstanceRequest{VpcId: "v", SubnetId: "s", MemoryLimit: 1}, false},
		{"zero memory", &CreateMemoryProInstanceRequest{VpcId: "v", SubnetId: "s", SecurityGroupIds: []string{"g"}, MemoryLimit: 0}, false},
		{"valid", &CreateMemoryProInstanceRequest{VpcId: "v", SubnetId: "s", SecurityGroupIds: []string{"g"}, MemoryLimit: 512}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err == nil) != tt.ok {
				t.Errorf("Validate() err=%v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestDeleteMemoryProInstanceRequest_Validate(t *testing.T) {
	if (&DeleteMemoryProInstanceRequest{}).Validate() == nil {
		t.Error("空请求应失败")
	}
	var nilReq *DeleteMemoryProInstanceRequest
	if nilReq.Validate() == nil {
		t.Error("nil 应失败")
	}
	if (&DeleteMemoryProInstanceRequest{MemoryProId: "mp-1"}).Validate() != nil {
		t.Error("有 MemoryProId 应通过")
	}
	if (&DeleteMemoryProInstanceRequest{ServiceId: "srv-1"}).Validate() != nil {
		t.Error("有 ServiceId 应通过")
	}
}

func TestCreateMemSpaceRequest_Validate(t *testing.T) {
	if (&CreateMemSpaceRequest{}).Validate() == nil {
		t.Error("空请求应失败")
	}
	var nilReq *CreateMemSpaceRequest
	if nilReq.Validate() == nil {
		t.Error("nil 应失败")
	}
	if (&CreateMemSpaceRequest{MemoryProId: "mp-1"}).Validate() != nil {
		t.Error("有 MemoryProId 应通过")
	}
}

func TestDeleteMemSpaceRequest_Validate(t *testing.T) {
	if (&DeleteMemSpaceRequest{}).Validate() == nil {
		t.Error("空 SpaceId 应失败")
	}
	var nilReq *DeleteMemSpaceRequest
	if nilReq.Validate() == nil {
		t.Error("nil 应失败")
	}
	if (&DeleteMemSpaceRequest{SpaceId: "sp-1"}).Validate() != nil {
		t.Error("有 SpaceId 应通过")
	}
}

func TestDescribeMemSpaceRecordRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  *DescribeMemSpaceRecordRequest
		ok   bool
	}{
		{"nil", nil, false},
		{"missing space", &DescribeMemSpaceRecordRequest{RecordType: "persona"}, false},
		{"missing type", &DescribeMemSpaceRecordRequest{SpaceId: "sp-1"}, false},
		{"valid", &DescribeMemSpaceRecordRequest{SpaceId: "sp-1", RecordType: "persona"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err == nil) != tt.ok {
				t.Errorf("Validate() err=%v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestMemoryProInstanceInfo_IsRunning(t *testing.T) {
	var nilInfo *MemoryProInstanceInfo
	if nilInfo.IsRunning() {
		t.Error("nil 不应 Running")
	}
	if (&MemoryProInstanceInfo{}).IsRunning() {
		t.Error("空 status 不应 Running")
	}
	if !(&MemoryProInstanceInfo{Status: "RUNNING"}).IsRunning() {
		t.Error("Status=RUNNING 应 Running")
	}
	if !(&MemoryProInstanceInfo{VDBStatus: "RUNNING"}).IsRunning() {
		t.Error("VDBStatus=RUNNING 应 Running")
	}
	if (&MemoryProInstanceInfo{Status: "CREATING"}).IsRunning() {
		t.Error("Status=CREATING 不应 Running")
	}
}

func TestMemoryProInstanceInfo_UsageRatio(t *testing.T) {
	var nilInfo *MemoryProInstanceInfo
	if nilInfo.UsageRatio() != 0 {
		t.Error("nil 应返回 0")
	}
	if (&MemoryProInstanceInfo{MemoryLimit: 0, MemoryUsed: 10}).UsageRatio() != 0 {
		t.Error("limit=0 应返回 0")
	}
	r := (&MemoryProInstanceInfo{MemoryLimit: 100, MemoryUsed: 25}).UsageRatio()
	if r < 0.249 || r > 0.251 {
		t.Errorf("UsageRatio() = %v, want ~0.25", r)
	}
}

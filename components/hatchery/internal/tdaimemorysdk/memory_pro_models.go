package tdaimemorysdk

import (
	"fmt"
	"strings"
)

// ========== Memory Pro（VDB 实例）相关 ==========

// ResourceTag 资源标签。
type ResourceTag struct {
	TagKey   string `json:"TagKey"`
	TagValue string `json:"TagValue"`
}

// CreateMemoryProInstanceRequest 创建 VDB 实例（开通 Memory Pro 服务）。
type CreateMemoryProInstanceRequest struct {
	VpcId            string        `json:"VpcId"`
	SubnetId         string        `json:"SubnetId"`
	SecurityGroupIds []string      `json:"SecurityGroupIds"`
	MemoryLimit      int           `json:"MemoryLimit"`
	ResourceTags     []ResourceTag `json:"ResourceTags,omitempty"`
}

// CreateMemoryProInstanceResponse 创建结果。
type CreateMemoryProInstanceResponse struct {
	MemoryProId   string `json:"MemoryProId"`
	VDBInstanceId string `json:"VDBInstanceId"`
	RequestID     string `json:"RequestId"`
}

// DescribeMemoryProInstancesRequest 查询 VDB 实例详情。
type DescribeMemoryProInstancesRequest struct {
	ServiceId string `json:"ServiceId,omitempty"`
}

// MemoryProInstanceInfo VDB 实例详情。
type MemoryProInstanceInfo struct {
	MemoryProId      string `json:"MemoryProId"`
	VDBInstanceId    string `json:"VDBInstanceId"`
	VDBInstanceName  string `json:"VDBInstanceName"`
	Status           string `json:"Status"`
	VDBStatus        string `json:"VDBStatus"`
	VDBTaskStatus    string `json:"VDBTaskStatus"`
	VDBApiVersion    string `json:"VDBApiVersion"`
	VDBEngineVersion string `json:"VDBEngineVersion"`
	MemoryLimit      int    `json:"MemoryLimit"`
	MemoryUsed       int    `json:"MemoryUsed"`
	Cpu              int    `json:"Cpu"`
	Memory           int    `json:"Memory"`
	Disk             int    `json:"Disk"`
	WorkerNodeNum    int    `json:"WorkerNodeNum"`
	AppId            int64  `json:"AppId"`
	Uin              string `json:"Uin"`
	CreatedAt        string `json:"CreatedAt"`
	UpdatedAt        string `json:"UpdatedAt"`

	// 以下字段由 DescribeMemoryProInstances 返回，用于池级网络连通性预检。
	VDBVip       string `json:"VDBVip"`       // VDB 内网 IP
	VDBPort      int    `json:"VDBPort"`      // VDB 端口
	TestAccount  string `json:"TestAccount"`  // 池级探测用账号
	TestPassword string `json:"TestPassword"` // 池级探测用密码（api_key）
}

// DescribeMemoryProInstancesResponse 查询结果。
type DescribeMemoryProInstancesResponse struct {
	TotalCount int                     `json:"TotalCount"`
	Items      []MemoryProInstanceInfo `json:"Items"`
	RequestID  string                  `json:"RequestId"`
}

// ModifyMemoryProInstanceRequest 修改 VDB 实例参数（如扩容）。
type ModifyMemoryProInstanceRequest struct {
	MemoryProId string `json:"MemoryProId,omitempty"`
	MemoryLimit *int   `json:"MemoryLimit,omitempty"`
}

// ModifyMemoryProInstanceResponse 修改结果。
type ModifyMemoryProInstanceResponse struct {
	RequestID string `json:"RequestId"`
}

// DeleteMemoryProInstanceRequest 释放 VDB 实例（关闭 Memory Pro 服务）。
type DeleteMemoryProInstanceRequest struct {
	MemoryProId string `json:"MemoryProId,omitempty"`
	ServiceId   string `json:"ServiceId,omitempty"`
}

// DeleteMemoryProInstanceResponse 释放结果。
type DeleteMemoryProInstanceResponse struct {
	RequestID string `json:"RequestId"`
}

// ========== MemSpace（VDB Database / 记忆库）相关 ==========

// CreateMemSpaceRequest 创建记忆库（VDB database）。
type CreateMemSpaceRequest struct {
	MemoryProId string `json:"MemoryProId,omitempty"`
	SpaceId     string `json:"SpaceId,omitempty"`
}

// CreateMemSpaceResponse 创建记忆库结果。
type CreateMemSpaceResponse struct {
	MemoryProId     string   `json:"MemoryProId"`
	SpaceId         string   `json:"SpaceId"`
	DatabaseName    string   `json:"DatabaseName"`
	CollectionNames []string `json:"CollectionNames"`
	EmbeddingModel  string   `json:"EmbeddingModel"` // 服务端分配的 embedding 模型
	Vip             string   `json:"Vip"`
	Port            int      `json:"Port"`
	Account         string   `json:"Account"`
	ApiKey          string   `json:"ApiKey"`
	RequestID       string   `json:"RequestId"`
}

// DescribeMemSpacesRequest 查询记忆库列表。
type DescribeMemSpacesRequest struct {
	MemoryProId string   `json:"MemoryProId,omitempty"`
	SpaceIds    []string `json:"SpaceIds,omitempty"`
}

// MemorySpaceInfo 记忆库详情。
type MemorySpaceInfo struct {
	SpaceId       string   `json:"SpaceId"`
	MemoryProId   string   `json:"MemoryProId"`
	VDBInstanceId string   `json:"VDBInstanceId"`
	DatabaseName  string   `json:"DatabaseName"`
	Collections   []string `json:"Collections"`
	Vip           string   `json:"Vip"`
	Port          int      `json:"Port"`
	Account       string   `json:"Account"`
	Status        string   `json:"Status"`
	CreatedAt     string   `json:"CreatedAt"`
	UpdatedAt     string   `json:"UpdatedAt"`
}

// DescribeMemSpacesResponse 查询结果。
type DescribeMemSpacesResponse struct {
	TotalCount int               `json:"TotalCount"`
	Items      []MemorySpaceInfo `json:"Items"`
	RequestID  string            `json:"RequestId"`
}

// DescribeMemSpaceRecordRequest 查询记忆库数据记录。
type DescribeMemSpaceRecordRequest struct {
	SpaceId        string   `json:"SpaceId"`
	RecordType     string   `json:"RecordType"`                       // persona / scene / memory / conversation
	Output         []string `json:"Output,omitempty"`
	OrderDirection string   `json:"OrderDirection,omitempty"` // 排序规则：ASC 升序 / DESC 降序
	StartTime      string   `json:"StartTime,omitempty"`      // 时间过滤，格式 2026-04-14 20:00:00
	EndTime        string   `json:"EndTime,omitempty"`        // 时间过滤，格式 2026-04-14 21:00:00
	Offset         *int     `json:"Offset,omitempty"`
	Limit          *int     `json:"Limit,omitempty"`
}

// VDBDocument VDB 文档。
type VDBDocument struct {
	Documents []map[string]any `json:"Documents,omitempty"`
}

// DescribeMemSpaceRecordResponse 查询记录结果。
type DescribeMemSpaceRecordResponse struct {
	TotalCount int              `json:"TotalCount"`
	Documents  []map[string]any `json:"Documents,omitempty"`
	RequestID  string           `json:"RequestId"`
}

// DeleteMemSpaceRequest 删除记忆库（VDB database）。
type DeleteMemSpaceRequest struct {
	SpaceId string `json:"SpaceId"`
}

// DeleteMemSpaceResponse 删除结果。
type DeleteMemSpaceResponse struct {
	RequestID string `json:"RequestId"`
}

// ========== 参数校验辅助方法 ==========

// Validate 校验 CreateMemoryProInstanceRequest 的必填字段。
func (r *CreateMemoryProInstanceRequest) Validate() error {
	if r == nil {
		return ErrEmptyAction
	}
	if r.VpcId == "" || r.SubnetId == "" || len(r.SecurityGroupIds) == 0 {
		return ErrEmptyAction
	}
	if r.MemoryLimit <= 0 {
		return ErrEmptyAction
	}
	return nil
}

// Validate 校验 DeleteMemoryProInstanceRequest 的必填字段。
func (r *DeleteMemoryProInstanceRequest) Validate() error {
	if r == nil {
		return ErrEmptyAction
	}
	if r.MemoryProId == "" && r.ServiceId == "" {
		return ErrEmptyAction
	}
	return nil
}

// Validate 校验 CreateMemSpaceRequest 的必填字段。
func (r *CreateMemSpaceRequest) Validate() error {
	if r == nil || r.MemoryProId == "" {
		return ErrEmptyAction
	}
	return nil
}

// Validate 校验 DeleteMemSpaceRequest 的必填字段。
func (r *DeleteMemSpaceRequest) Validate() error {
	if r == nil || r.SpaceId == "" {
		return ErrEmptyAction
	}
	return nil
}

// Validate 校验 DescribeMemSpaceRecordRequest 的必填字段。
func (r *DescribeMemSpaceRecordRequest) Validate() error {
	if r == nil || r.SpaceId == "" || r.RecordType == "" {
		return ErrEmptyAction
	}
	return nil
}

// VDBEndpoint 返回 VDB 的 HTTP 接入地址，格式 http://{VDBVip}:{VDBPort}。
// VDBVip 或 VDBPort 为空/零值时返回空字符串。
func (m *MemoryProInstanceInfo) VDBEndpoint() string {
	if m == nil || m.VDBVip == "" || m.VDBPort <= 0 {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", m.VDBVip, m.VDBPort)
}

// IsRunning 判断 VDB 实例是否处于运行状态。
// DescribeMemoryProInstances 返回的 Status/VDBStatus 值为小写 "online"/"running" 等，
// 此处做大小写不敏感匹配。
func (m *MemoryProInstanceInfo) IsRunning() bool {
	if m == nil {
		return false
	}
	s := strings.ToLower(m.Status)
	vs := strings.ToLower(m.VDBStatus)
	return s == "running" || s == "online" || vs == "running" || vs == "online"
}

// UsageRatio 返回已用内存占总内存的比例（0.0-1.0），MemoryLimit 为 0 时返回 0。
func (m *MemoryProInstanceInfo) UsageRatio() float64 {
	if m == nil || m.MemoryLimit <= 0 {
		return 0
	}
	return float64(m.MemoryUsed) / float64(m.MemoryLimit)
}

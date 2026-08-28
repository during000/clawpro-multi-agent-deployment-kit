package tdaimemorysdk

// DescribeAgentInstanceRequest 是复制自现有 tdai SDK 的简单读接口请求模型。
type DescribeAgentInstanceRequest struct {
	InstanceID string `json:"InstanceId,omitempty"`
}

// DescribeAgentInstanceResponse 是读接口响应模型（精简版）。
// 为避免频繁跟随外部模型变更，这里将 AgentInstance 先保持为 map。
type DescribeAgentInstanceResponse struct {
	AgentInstance map[string]any `json:"AgentInstance,omitempty"`
	RequestID     string         `json:"RequestId,omitempty"`
}

// Validate 校验请求参数。
func (r *DescribeAgentInstanceRequest) Validate() error {
	if r == nil || r.InstanceID == "" {
		return ErrEmptyAction
	}
	return nil
}

// IsEmpty 判断响应是否为空（无 AgentInstance 数据）。
func (r *DescribeAgentInstanceResponse) IsEmpty() bool {
	return r == nil || len(r.AgentInstance) == 0
}

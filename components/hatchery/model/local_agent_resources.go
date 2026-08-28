package model

import (
	"database/sql/driver"
	"encoding/json"
)

// LocalAgentResources 存进 instances.local_agent_resources 字段的 JSON 结构。
//
// 二期新增：本地 agent 的分组绑定 + workspace 列表。source != 'local' 时该字段为空。
// 字段名不叫 bindings，叫 resources（用户确认）。
//
// 通过 Value/Scan 方法实现 GORM 对 TEXT 字段的 JSON 自动序列化/反序列化。
//
// 迁移脚本: sql/0706-local-agent-resources.sql | 初始化脚本: sql/init.sql (instances 表)
type LocalAgentResources struct {
	UserLevel  UserLevelResources  `json:"user_level"`
	Workspaces []WorkspaceResource `json:"workspaces"`
}

// UserLevelResources 用户级资源绑定（跟随用户账号，对该用户所有 workspace 生效）。
type UserLevelResources struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name,omitempty"`
}

// WorkspaceResource 项目级 workspace 绑定（用 path 作唯一标识，teamai 说 workspace 没有 id）。
type WorkspaceResource struct {
	Path      string `json:"path"` // ⭐ 唯一标识
	Name      string `json:"name"`
	IDEType   string `json:"ide_type"` // codebuddy/workbuddy/claude_code/codex
	ProjectID uint   `json:"project_id,omitempty"`
}

// Value 实现 driver.Valuer，把结构体序列化为 JSON 存进 TEXT 字段。
// nil 指针存 NULL。
func (r *LocalAgentResources) Value() (driver.Value, error) {
	if r == nil {
		return nil, nil
	}
	return json.Marshal(r)
}

// Scan 实现 sql.Scanner，从 TEXT 字段反序列化 JSON 到结构体。
// 空字符串 / NULL 时不初始化（保持 nil），由调用方判断。
func (r *LocalAgentResources) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, r)
}

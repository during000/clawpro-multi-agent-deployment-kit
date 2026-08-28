package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

// VpcConfig VPC 配置资源表，每条记录 = 列表一行 = 一个完整绑定关系
type VpcConfig struct {
	gorm.Model
	Identifier     string `gorm:"size:191;not null;default:''" json:"-"`
	VpcId          string `gorm:"size:64;not null" json:"vpc_id"`
	SubnetIds      string `gorm:"type:text;not null" json:"subnet_ids"`                          // JSON: {"zone": ["subnet-id", ...]}
	VisibilityType string `gorm:"size:16;not null;default:'all'" json:"visibility_type"`         // all / group
	StrategyName   string `gorm:"size:20;default:''" json:"strategy_name"`                       // 策略名称，可选
}

// GetSubnetMap 解析 SubnetIds JSON 字符串为 map[string][]string
func (v VpcConfig) GetSubnetMap() (map[string][]string, error) {
	if v.SubnetIds == "" {
		return make(map[string][]string), nil
	}
	var subnetMap map[string][]string
	if err := json.Unmarshal([]byte(v.SubnetIds), &subnetMap); err != nil {
		return nil, err
	}
	if subnetMap == nil {
		return make(map[string][]string), nil
	}
	return subnetMap, nil
}

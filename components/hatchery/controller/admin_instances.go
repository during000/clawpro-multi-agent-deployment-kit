package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type adminInstanceItem struct {
	model.Instance
	Username       string                    `json:"Username"`
	Department     string                    `json:"department" gorm:"-"`
	Departments    []deptWithPath            `json:"departments,omitempty" gorm:"-"`
	DepartmentPath string                    `json:"department_path,omitempty" gorm:"-"`
	Adjustment     *model.InstanceAdjustment `json:"-" gorm:"-"`
}

func attachInstanceAdjustments(ctx context.Context, items []adminInstanceItem) {
	if len(items) == 0 {
		return
	}
	instanceIDs := make([]uint, 0, len(items))
	for i := range items {
		instanceIDs = append(instanceIDs, items[i].ID)
	}
	var adjustments []model.InstanceAdjustment
	if err := model.DB(ctx).Where("instance_id IN ?", instanceIDs).Find(&adjustments).Error; err != nil {
		slog.WarnContext(ctx, "[AdminList] 批量查询资源调整任务失败", "error", err)
		return
	}
	byInstanceID := make(map[uint]*model.InstanceAdjustment, len(adjustments))
	for i := range adjustments {
		byInstanceID[adjustments[i].InstanceID] = &adjustments[i]
	}
	for i := range items {
		items[i].Adjustment = byInstanceID[items[i].ID]
	}
}

// adminInstanceItemWithStatus 管控端列表项（含实时状态）
type adminInstanceItemWithStatus struct {
	// DB 原始字段
	ID                        uint    `json:"ID"`
	Name                      string  `json:"Name"`
	InstanceId                string  `json:"InstanceId"`
	UserID                    uint    `json:"UserID"`
	Username                  string  `json:"Username"`
	GroupID                   uint    `json:"GroupID"`       // 🆕 创建时指定的分组 ID，0 = 未指定
	GroupFullPath             string  `json:"GroupFullPath"` // 🆕 分组全路径（如"研发中心/后端组"），GroupID=0 时为空串
	CurrentOperation          string  `json:"current_operation"`
	CurrentOperationState     string  `json:"current_operation_state"`
	CurrentOperationUpdatedAt *string `json:"current_operation_updated_at"`
	InstanceChargeType        string  `json:"instance_charge_type"`
	LastCVMState              string  `json:"LastCVMState"`
	AgentReady                int     `json:"AgentReady"`
	RuntimeUser               string  `json:"RuntimeUser"`
	RuntimeHome               string  `json:"RuntimeHome"`
	CreatedAt                 string  `json:"CreatedAt"`
	AIModelID                 uint    `json:"AIModelID"`
	// 智能体类型和版本
	AgentType           string  `json:"agent_type"`
	AgentVersion        string  `json:"agent_version"`
	IsBuiltin           bool    `json:"is_builtin"`
	CompatibleWith      string  `json:"compatible_with,omitempty"`
	PluginVersionStatus string  `json:"plugin_version_status"`
	VersionFetchedAt    *string `json:"version_fetched_at"`
	// 镜像信息
	IsOfficialImage bool `json:"is_official_image"` // 实例是否安装的为官方公共镜像
	// 实例标签（从 CVM API 实时获取）
	Tags []CVMTag `json:"tags"`
	// 创建用户的 OneID 部门信息（与 /admin/users 接口同口径，前端可复用同一套类型）
	Department     string         `json:"department"`                // 主部门名（OneID 画像 main_dept_name），无画像时为空串
	Departments    []deptWithPath `json:"departments,omitempty"`     // 完整部门列表，每项含本部门的 department_path；无画像或反序列化失败时省略
	DepartmentPath string         `json:"department_path,omitempty"` // 主部门全路径（如 "OpenClaw企业版体验/新组/市场组/市场二组"），无法解析时省略
	// 实时状态（内联）
	Status    string   `json:"status"`
	Label     string   `json:"label"`
	Tooltip   string   `json:"tooltip"`
	Actions   []string `json:"actions"`
	Transient bool     `json:"transient"`
	// CVM 资源缓存与资源调整展示；本地 Agent 通过 omitempty 省略。
	CVMInstanceType         string  `json:"cvm_instance_type,omitempty"`
	CPU                     int64   `json:"cpu,omitempty"`
	MemoryGB                int64   `json:"memory_gb,omitempty"`
	SystemDiskType          string  `json:"system_disk_type,omitempty"`
	SystemDiskSize          int64   `json:"system_disk_size,omitempty"`
	PublicIP                *string `json:"public_ip,omitempty"`
	InternetChargeType      *string `json:"internet_charge_type,omitempty"`
	InternetMaxBandwidthOut *int64  `json:"internet_max_bandwidth_out,omitempty"`
	AdjustmentStatus        string  `json:"adjustment_status,omitempty"`
	AdjustmentType          string  `json:"adjustment_type,omitempty"`
	AdjustmentErrorCode     string  `json:"adjustment_error_code,omitempty"`
	AdjustmentErrorMessage  string  `json:"adjustment_error_message,omitempty"`
	AdjustmentUpdatedAt     *string `json:"adjustment_updated_at,omitempty"`
	// 本地 agent 实例字段（source=local 时填充；CVM 实例返回零值）
	Source       string                  `json:"source"`                   // "cvm" / "local"
	HostName     string                  `json:"host_name,omitempty"`      // 仅 source=local
	OS           string                  `json:"os,omitempty"`             // 仅 source=local
	LastReportAt *string                 `json:"last_report_at,omitempty"` // 仅 source=local，RFC3339
	Projects     []projectVisibilityInfo `json:"projects,omitempty"`       // 仅 source=local，Workspace 绑定的有效项目
	// 存量实例分组归属处理（stale-instances v1.0）— 恒定输出，便于前端始终能拿到字段
	Flags                    []string `json:"flags"`
	HandoverTargetUserID     uint     `json:"handover_target_user_id"`
	HandoverRejectedByUserID uint     `json:"handover_rejected_by_user_id"`
	HandoverInitiatedAt      *string  `json:"handover_initiated_at"`
}

// adminStatsOtherDetail 统计卡片 other 分类详情
type adminStatsOtherDetail struct {
	NeedAttention struct {
		Count int            `json:"count"`
		Label string         `json:"label"`
		Items map[string]int `json:"items"`
	} `json:"need_attention"`
	InProgress struct {
		Count int            `json:"count"`
		Label string         `json:"label"`
		Items map[string]int `json:"items"`
	} `json:"in_progress"`
}

// adminStats 统计卡片
type adminStats struct {
	Total       int                   `json:"total"`
	Running     int                   `json:"running"`
	Stopped     int                   `json:"stopped"`
	Other       int                   `json:"other"`
	OtherDetail adminStatsOtherDetail `json:"other_detail"`
}

// adminQueryFilter 管控端列表查询过滤条件
type adminQueryFilter struct {
	Keyword          string   // 模糊搜索：名称/实例ID/用户名
	Creator          string   // 按创建人精确筛选
	DateFrom         string   // 创建时间起始（YYYY-MM-DD，含当天）
	DateTo           string   // 创建时间截止（YYYY-MM-DD，含当天）
	DepartmentID     string   // 按部门 ID 筛选
	AgentType        []string // 按智能体类型过滤（多值 OR），空=不过滤
	IDs              []uint   // 按 ID 列表过滤（用于状态筛选后的分页查询）
	CVMInstanceTypes []string // 按 CVM 实例规格过滤（多值 OR）
	SystemDiskSizes  []int64  // 按系统盘容量过滤（多值 OR）
	SystemDiskSizeLT int64    // 系统盘容量上限（不含）
	SystemDiskSizeGT int64    // 系统盘容量下限（不含）

	// 🆕 用户层批量过滤（来自请求参数 ids / instance_ids，每项 ≤ adminInstancesQueryMaxIDs）
	RequestIDs         []uint   // ids：按 instances.id 精确过滤
	RequestInstanceIDs []string // instance_ids：按 instances.instance_id (CVM 实例 ID) 精确过滤

	// 标签过滤（与 CVM Tag API 对应）
	// TagKeys 与 (TagKey + TagValues) 互斥；若同时传，优先使用 (TagKey + TagValues)。
	TagKeys   []string // 按标签键过滤：实例需带任一 key（多键 OR）
	TagKey    string   // 标签键（与 TagValues 配合使用）
	TagValues []string // 该 key 下要匹配的值（多值 OR）

	// 三期：按实例来源过滤（cvm / local），空=不过滤
	Source string
}

// adminInstancesQueryMaxIDs 是 /admin/instances 中 ids / instance_ids 参数的单参数上限
const adminInstancesQueryMaxIDs = 1000

// adminInstancesQueryMaxTagValues 是 /admin/instances 中 tag_keys / tag_values 参数的单参数上限
const adminInstancesQueryMaxTagValues = 100

// escapeSQLLike 转义 LIKE 匹配中的特殊字符，防止通配符注入。
func escapeSQLLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// parseAdminInstancesTagFilters 解析 /admin/instances 的 tag_keys / tag_key / tag_values 参数。
//
//   - tag_keys: 逗号分隔的标签键列表，命中任一键即匹配（多键 OR）
//   - tag_key + tag_values: 单一标签键 + 该键下的多个候选值（多值 OR）
//
// 两组互斥：若同时传，以 tag_key + tag_values 为准（更精确），忽略 tag_keys。
// 单参数元素数量上限 adminInstancesQueryMaxTagValues。
func parseAdminInstancesTagFilters(q url.Values, filter *adminQueryFilter) error {
	tagKey := strings.TrimSpace(q.Get("tag_key"))
	rawValues := strings.TrimSpace(q.Get("tag_values"))
	if tagKey != "" && rawValues != "" {
		parts := strings.Split(rawValues, ",")
		if len(parts) > adminInstancesQueryMaxTagValues {
			return hcommon.I18nError(i18n.MsgTagValuesCountExceed, adminInstancesQueryMaxTagValues)
		}
		cleaned := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				cleaned = append(cleaned, p)
			}
		}
		if len(cleaned) > adminInstancesQueryMaxTagValues {
			return hcommon.I18nError(i18n.MsgTagValuesCountExceed, adminInstancesQueryMaxTagValues)
		}
		if len(cleaned) > 0 {
			filter.TagKey = tagKey
			filter.TagValues = cleaned
			return nil
		}
	}
	if raw := strings.TrimSpace(q.Get("tag_keys")); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) > adminInstancesQueryMaxTagValues {
			return hcommon.I18nError(i18n.MsgTagKeysCountExceed, adminInstancesQueryMaxTagValues)
		}
		cleaned := make([]string, 0, len(parts))
		seen := make(map[string]struct{}, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			cleaned = append(cleaned, p)
		}
		if len(cleaned) > 0 {
			filter.TagKeys = cleaned
		}
	}
	return nil
}

// hasTagFilter 是否启用了任意标签过滤
func (f adminQueryFilter) hasTagFilter() bool {
	return (f.TagKey != "" && len(f.TagValues) > 0) || len(f.TagKeys) > 0
}

// matchTagFilter 判断给定 CVM 实例（标签集合）是否满足当前 filter 中的标签条件。
// 不带标签过滤时永远返回 true。cvmInfo 为 nil 或 API_ERROR / 无标签数据时返回 false（无可信标签信息）。
func (f adminQueryFilter) matchTagFilter(cvmInfo *CVMInstanceInfo) bool {
	if !f.hasTagFilter() {
		return true
	}
	if cvmInfo == nil || cvmInfo.State == "API_ERROR" {
		return false
	}
	if f.TagKey != "" && len(f.TagValues) > 0 {
		valSet := make(map[string]struct{}, len(f.TagValues))
		for _, v := range f.TagValues {
			valSet[v] = struct{}{}
		}
		for _, t := range cvmInfo.Tags {
			if t.Key != f.TagKey {
				continue
			}
			if _, ok := valSet[t.Value]; ok {
				return true
			}
		}
		return false
	}
	keySet := make(map[string]struct{}, len(f.TagKeys))
	for _, k := range f.TagKeys {
		keySet[k] = struct{}{}
	}
	for _, t := range cvmInfo.Tags {
		if _, ok := keySet[t.Key]; ok {
			return true
		}
	}
	return false
}

// parseAdminInstancesIDFilters 解析 /admin/instances 的 ids / instance_ids 查询参数并写入 filter。
//
//   - ids: 逗号分隔的 uint，按 instances.id 精确过滤
//   - instance_ids: 逗号分隔的字符串(形如 ins-xxx)，按 instances.instance_id 精确过滤
//
// 任一参数解析后元素数量超过 adminInstancesQueryMaxIDs 返回 400。
// 空字符串或解析后为空视为未传，不报错。
func parseAdminInstancesIDFilters(q url.Values, filter *adminQueryFilter) error {
	if raw := strings.TrimSpace(q.Get("ids")); raw != "" {
		// 先看原始片段数，防止极端大请求耗时 ParseUint
		if rawCount := strings.Count(raw, ",") + 1; rawCount > adminInstancesQueryMaxIDs {
			return hcommon.I18nError(i18n.MsgIDsCountExceed, adminInstancesQueryMaxIDs)
		}
		ids, err := parseUintCSV(raw)
		if err != nil {
			return hcommon.I18nError(i18n.MsgInvalidIDsFormat)
		}
		if len(ids) > adminInstancesQueryMaxIDs {
			return hcommon.I18nError(i18n.MsgIDsCountExceed, adminInstancesQueryMaxIDs)
		}
		filter.RequestIDs = ids
	}
	if raw := strings.TrimSpace(q.Get("instance_ids")); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) > adminInstancesQueryMaxIDs {
			return hcommon.I18nError(i18n.MsgTooManyInstanceIDs, adminInstancesQueryMaxIDs)
		}
		cleaned := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				cleaned = append(cleaned, p)
			}
		}
		if len(cleaned) > adminInstancesQueryMaxIDs {
			return hcommon.I18nError(i18n.MsgTooManyInstanceIDs, adminInstancesQueryMaxIDs)
		}
		filter.RequestInstanceIDs = cleaned
	}
	return nil
}

func parseAdminInstanceResourceFilters(q url.Values, filter *adminQueryFilter) error {
	if raw := strings.TrimSpace(q.Get("cvm_instance_type")); raw != "" {
		for _, value := range strings.Split(raw, ",") {
			value = strings.TrimSpace(value)
			if value != "" {
				filter.CVMInstanceTypes = append(filter.CVMInstanceTypes, value)
			}
		}
	}

	diskFilterCount := 0
	if raw := strings.TrimSpace(q.Get("system_disk_size")); raw != "" {
		diskFilterCount++
		for _, value := range strings.Split(raw, ",") {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			size, err := strconv.ParseInt(value, 10, 64)
			if err != nil || size <= 0 {
				return hcommon.I18nError(i18n.MsgInvalidRequestFormat)
			}
			filter.SystemDiskSizes = append(filter.SystemDiskSizes, size)
		}
	}
	if raw := strings.TrimSpace(q.Get("system_disk_size_lt")); raw != "" {
		diskFilterCount++
		size, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || size <= 0 {
			return hcommon.I18nError(i18n.MsgInvalidRequestFormat)
		}
		filter.SystemDiskSizeLT = size
	}
	if raw := strings.TrimSpace(q.Get("system_disk_size_gt")); raw != "" {
		diskFilterCount++
		size, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || size <= 0 {
			return hcommon.I18nError(i18n.MsgInvalidRequestFormat)
		}
		filter.SystemDiskSizeGT = size
	}
	if diskFilterCount > 1 {
		return hcommon.I18nError(i18n.MsgInvalidRequestFormat)
	}
	return nil
}

func applyAdminInstanceResourceFilters(base *gorm.DB, filter adminQueryFilter) *gorm.DB {
	if len(filter.CVMInstanceTypes) > 0 {
		base = base.Where("instances.cvm_instance_type IN ?", filter.CVMInstanceTypes)
	}
	switch {
	case len(filter.SystemDiskSizes) > 0:
		base = base.Where("instances.system_disk_size IN ?", filter.SystemDiskSizes)
	case filter.SystemDiskSizeLT > 0:
		base = base.Where("instances.system_disk_size < ?", filter.SystemDiskSizeLT)
	case filter.SystemDiskSizeGT > 0:
		base = base.Where("instances.system_disk_size > ?", filter.SystemDiskSizeGT)
	}
	return base
}

// lightInstance 轻量实例信息，仅包含状态计算所需的最少字段（用于全局统计）
type lightInstance struct {
	ID                        uint
	Name                      string
	UserID                    uint
	GroupID                   uint
	InstanceId                string
	CurrentOperation          string
	CurrentOperationState     string
	CurrentOperationUpdatedAt *time.Time
	LastCVMState              string
	LastStableState           string
	AgentReady                int
	CLSAgentStatus            int
	CLSAgentStatusAt          *time.Time
	// 缓存字段（后台 reconcile diff 比较用）
	LastKnownStatus            string `gorm:"column:last_known_status"`
	CVMTagsJSON                string `gorm:"column:cvm_tags_json"`
	ImgId                      string `gorm:"column:img_id"`
	InstanceChargeType         string `gorm:"column:instance_charge_type"`
	Source                     string `gorm:"column:source"`
	CVMInstanceType            string `gorm:"column:cvm_instance_type"`
	CVMCPU                     int64  `gorm:"column:cvm_cpu"`
	CVMMemoryGB                int64  `gorm:"column:cvm_memory_gb"`
	SystemDiskType             string `gorm:"column:system_disk_type"`
	SystemDiskSize             int64  `gorm:"column:system_disk_size"`
	CVMPublicIP                string `gorm:"column:cvm_public_ip"`
	CVMInternetChargeType      string `gorm:"column:cvm_internet_charge_type"`
	CVMInternetMaxBandwidthOut int64  `gorm:"column:cvm_internet_max_bandwidth_out"`
}

// queryAllLightInstancesWithFilter 轻量全量查询：只查状态计算所需字段，不 JOIN 用户/部门表
// 用于全局统计和 CVM 批量查询，性能开销远小于 queryInstancesWithFilter
func queryAllLightInstancesWithFilter(ctx context.Context, filter adminQueryFilter) ([]lightInstance, error) {
	base := model.DB(ctx).Model(&model.Instance{})

	if filter.Keyword != "" {
		kw := "%" + escapeSQLLike(filter.Keyword) + "%"
		base = base.Joins("LEFT JOIN users ON users.id = instances.user_id").
			Where(
				"instances.name LIKE ? OR instances.instance_id LIKE ? OR users.username LIKE ?",
				kw, kw, kw,
			)
	}
	if filter.Creator != "" {
		if filter.Keyword == "" {
			base = base.Joins("LEFT JOIN users ON users.id = instances.user_id")
		}
		base = base.Where("users.username = ?", filter.Creator)
	}
	if filter.DateFrom != "" {
		base = base.Where("DATE(instances.created_at) >= ?", filter.DateFrom)
	}
	if filter.DateTo != "" {
		base = base.Where("DATE(instances.created_at) <= ?", filter.DateTo)
	}
	if filter.DepartmentID != "" {
		departmentID := filter.DepartmentID
		base = base.Where(
			"instances.user_id IN (?)",
			model.DB(ctx).Model(&model.User{}).Select("id").
				Where(
					"one_id_sub IN (?)",
					model.DB(ctx).Model(&model.OneIDUserProfile{}).Select("one_id_sub").Where(
						"main_dept_id = ? OR departments_json LIKE ?", departmentID, model.DeptIDLikePattern(departmentID),
					),
				),
		)
	}
	if len(filter.AgentType) > 0 {
		base = base.Where("instances.agent_type IN ?", filter.AgentType)
	}
	base = applyAdminInstanceResourceFilters(base, filter)
	if filter.Source != "" {
		base = base.Where("instances.source = ?", filter.Source)
	}
	if len(filter.RequestIDs) > 0 {
		base = base.Where("instances.id IN ?", filter.RequestIDs)
	}
	if len(filter.RequestInstanceIDs) > 0 {
		base = base.Where("instances.instance_id IN ?", filter.RequestInstanceIDs)
	}

	var items []lightInstance
	if err := base.Select("instances.id, instances.name, instances.user_id, instances.group_id, instances.instance_id, instances.current_operation, instances.current_operation_state, instances.current_operation_updated_at, instances.last_cvm_state, instances.last_stable_state, instances.agent_ready, instances.cls_agent_status, instances.cls_agent_status_at, instances.last_known_status, instances.cvm_tags_json, instances.img_id, instances.instance_charge_type, instances.source, instances.cvm_instance_type, instances.cvm_cpu, instances.cvm_memory_gb, instances.system_disk_type, instances.system_disk_size, instances.cvm_public_ip, instances.cvm_internet_charge_type, instances.cvm_internet_max_bandwidth_out").
		Order("instances.id desc").
		Scan(&items).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryLightInstancesFailed)
	}

	return items, nil
}

// queryInstancesWithFilter 支持新查询参数的管控端列表查询（不含状态筛选，状态筛选在内存中完成）
func queryInstancesWithFilter(ctx context.Context, page, pageSize int, filter adminQueryFilter) ([]adminInstanceItem, int64) {
	base := model.DB(ctx).Model(&model.Instance{}).
		Joins("LEFT JOIN users ON users.id = instances.user_id")

	if filter.Keyword != "" {
		kw := "%" + escapeSQLLike(filter.Keyword) + "%"
		base = base.Where(
			"instances.name LIKE ? OR instances.instance_id LIKE ? OR users.username LIKE ?",
			kw, kw, kw,
		)
	}
	if filter.Creator != "" {
		base = base.Where("users.username = ?", filter.Creator)
	}
	if filter.DateFrom != "" {
		base = base.Where("DATE(instances.created_at) >= ?", filter.DateFrom)
	}
	if filter.DateTo != "" {
		base = base.Where("DATE(instances.created_at) <= ?", filter.DateTo)
	}
	if filter.DepartmentID != "" {
		departmentID := filter.DepartmentID
		base = base.Where(
			"instances.user_id IN (?)",
			model.DB(ctx).Model(&model.User{}).Select("id").
				Where(
					"one_id_sub IN (?)",
					model.DB(ctx).Model(&model.OneIDUserProfile{}).Select("one_id_sub").Where(
						"main_dept_id = ? OR departments_json LIKE ?", departmentID, model.DeptIDLikePattern(departmentID),
					),
				),
		)
	}
	if len(filter.AgentType) > 0 {
		base = base.Where("instances.agent_type IN ?", filter.AgentType)
	}
	base = applyAdminInstanceResourceFilters(base, filter)
	if len(filter.IDs) > 0 {
		base = base.Where("instances.id IN ?", filter.IDs)
	}
	if len(filter.RequestIDs) > 0 {
		base = base.Where("instances.id IN ?", filter.RequestIDs)
	}
	if len(filter.RequestInstanceIDs) > 0 {
		base = base.Where("instances.instance_id IN ?", filter.RequestInstanceIDs)
	}

	var total int64
	base.Model(&model.Instance{}).Count(&total)

	var items []adminInstanceItem
	base.Select("instances.*, users.username").
		Order("instances.id desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items)
	attachInstanceAdjustments(ctx, items)

	// 批量查询用户（避免循环中 N 次查询）。这里直接拿到 user 列表，
	// 既用于回填后面的可能字段，也作为 enrichUserDepartments 的输入。
	userIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.UserID > 0 {
			userIDs = append(userIDs, item.UserID)
		}
	}
	var users []model.User
	if len(userIDs) > 0 {
		model.DB(ctx).Where("id IN ?", userIDs).Find(&users)
	}

	// 用共享 helper 批量补 OneID 部门信息。当本页用户全部没有 OneIDSub
	// （典型场景：未开通 OneID 的部署）时，helper 内部短路，
	// 不会触发 oneid_user_profiles / oneid_departments 的查询。
	deptInfo := enrichUserDepartments(ctx, users)
	for i := range items {
		if d, ok := deptInfo[items[i].UserID]; ok {
			items[i].Department = d.Department
			items[i].Departments = d.Departments
			items[i].DepartmentPath = d.DepartmentPath
		}
	}

	return items, total
}

// firstPublicIP 从 CVM DescribeInstances 响应的 PublicIpAddresses 里取首个非空 IP。
// 数组为空或首项为 nil 时返回空串。
func firstPublicIP(ips []*string) string {
	for _, p := range ips {
		if p != nil && *p != "" {
			return *p
		}
	}
	return ""
}

// internetChargeTypeFromCVM 从 InternetAccessible 结构里取网络计费类型；nil-safe。
func internetChargeTypeFromCVM(ia *cvm.InternetAccessible) string {
	if ia == nil || ia.InternetChargeType == nil {
		return ""
	}
	return *ia.InternetChargeType
}

// internetBandwidthFromCVM 从 InternetAccessible 结构里取带宽上限（Mbps）；nil-safe。
func internetBandwidthFromCVM(ia *cvm.InternetAccessible) int64 {
	if ia == nil || ia.InternetMaxBandwidthOut == nil {
		return 0
	}
	return *ia.InternetMaxBandwidthOut
}

// batchFetchCVMInfoMap 批量查询 CVM 实例信息，返回 instanceId → CVMInstanceInfo 的映射
// 每次最多查询 100 个（CVM API 限制）
// 当批量查询因无效 ID 失败时，自动降级为逐个查询，避免一个坏 ID 拖垮整批
func batchFetchCVMInfoMap(ctx context.Context, instanceIds []string) map[string]*CVMInstanceInfo {
	result := make(map[string]*CVMInstanceInfo)
	if len(instanceIds) == 0 {
		return result
	}

	// 本地实例（local-xxx）不在云上，过滤掉后再调 CVM API。
	// 滤掉后仍返回一个给调用方可查的标识，表示“本地状态从 DB 取”。
	cvmIds := make([]string, 0, len(instanceIds))
	for _, id := range instanceIds {
		if strings.HasPrefix(id, "local-") {
			result[id] = &CVMInstanceInfo{State: "LOCAL"}
			continue
		}
		cvmIds = append(cvmIds, id)
	}
	if len(cvmIds) == 0 {
		return result
	}
	instanceIds = cvmIds

	client, err := NewCVMClient(ctx)
	if err != nil {
		slog.Warn("[AdminList] 创建 CVM 客户端失败", "error", err)
		// 客户端创建失败时标记所有实例为 API_ERROR，避免调用方误判为"CVM 不存在"触发外部销毁感知。
		for _, id := range instanceIds {
			result[id] = &CVMInstanceInfo{State: "API_ERROR"}
		}
		return result
	}

	// 分批查询，每批最多 100 个
	const batchSize = 100
	for i := 0; i < len(instanceIds); i += batchSize {
		end := i + batchSize
		if end > len(instanceIds) {
			end = len(instanceIds)
		}
		batch := instanceIds[i:end]

		req := cvm.NewDescribeInstancesRequest()
		req.InstanceIds = common.StringPtrs(batch)
		req.Limit = common.Int64Ptr(int64(len(batch)))

		resp, err := client.DescribeInstances(req)
		if err != nil {
			// 检查是否因无效 ID 导致整批失败
			if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok && sdkErr.GetCode() == "InvalidInstanceId.NotFound" {
				slog.Warn("[AdminList] 批量查询含无效 ID，降级为逐个查询", "batch_size", len(batch), "error", err)
				// 降级：逐个查询，避免一个坏 ID 影响整批
				for _, id := range batch {
					info, singleErr := fetchSingleCVMInfo(client, id)
					if singleErr != nil {
						slog.Debug("[AdminList] 单个查询失败", "instance_id", id, "error", singleErr)
						// 单个查询失败时标记为 API_ERROR，避免误判为"CVM 不存在"触发外部销毁感知。
						if _, exists := result[id]; !exists {
							result[id] = &CVMInstanceInfo{State: "API_ERROR"}
						}
						continue
					}
					if info != nil {
						result[id] = info
					}
				}
				continue
			}
			slog.Warn("[AdminList] 批量查询 CVM 失败", "error", err)
			// API 失败的实例标记为 API_ERROR，避免调用方误判为"CVM 不存在"触发外部销毁感知。
			// 仅标记 result 中尚未存在的 ID（可能在前面的批次中已成功查询到）。
			for _, id := range batch {
				if _, exists := result[id]; !exists {
					result[id] = &CVMInstanceInfo{State: "API_ERROR"}
				}
			}
			continue
		}
		if resp.Response == nil {
			continue
		}
		for _, inst := range resp.Response.InstanceSet {
			if inst.InstanceId == nil {
				continue
			}
			result[*inst.InstanceId] = cvmInstanceInfoFromSDK(inst)
		}
	}
	return result
}

// fetchSingleCVMInfo 查询单个 CVM 实例信息（用于批量查询降级场景）
// 与 fetchCVMInstanceInfo 不同，此函数复用已有的 client 连接
func fetchSingleCVMInfo(client *cvm.Client, instanceId string) (*CVMInstanceInfo, error) {
	req := cvm.NewDescribeInstancesRequest()
	req.InstanceIds = common.StringPtrs([]string{instanceId})
	req.Limit = common.Int64Ptr(1)

	resp, err := client.DescribeInstances(req)
	if err != nil {
		if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok {
			if sdkErr.GetCode() == "InvalidInstanceId.NotFound" {
				return nil, nil // 实例不存在，返回 nil（非错误）
			}
		}
		return nil, err
	}
	if resp.Response == nil || len(resp.Response.InstanceSet) == 0 {
		return nil, nil
	}

	inst := resp.Response.InstanceSet[0]
	return cvmInstanceInfoFromSDK(inst), nil
}

// fetchGroupFullPathMap 批量查 group_id → full_path 映射。
// 输入的 groupIDs 可含 0 或重复；返回的 map 不含 0 键。
// 查询失败时返回空 map + log warn，不中断请求。
func fetchGroupFullPathMap(ctx context.Context, groupIDs []uint) map[uint]string {
	result := map[uint]string{}
	seen := map[uint]struct{}{}
	unique := make([]uint, 0, len(groupIDs))
	for _, gid := range groupIDs {
		if gid == 0 {
			continue
		}
		if _, ok := seen[gid]; ok {
			continue
		}
		seen[gid] = struct{}{}
		unique = append(unique, gid)
	}
	if len(unique) == 0 {
		return result
	}
	groups, err := model.GetGroupsByIDs(ctx, unique)
	if err != nil {
		slog.Warn("[GroupFullPath] 查询分组 full_path 失败", "error", err, "ids", unique)
		return result
	}
	for _, g := range groups {
		result[g.ID] = g.FullPath
	}
	return result
}

// buildAdminInstanceWithStatus 将 adminInstanceItem + CVM 信息组装为带状态的响应项
func buildAdminInstanceWithStatus(ctx context.Context, item adminInstanceItem, cvmInfo *CVMInstanceInfo) adminInstanceItemWithStatus {
	inst := item.Instance
	statusResp := ResolveInstanceStatus(ctx, &inst, cvmInfo, nil)
	// 使用管控端状态映射覆盖 actions；资源调整期间保持 actions=[]。
	if !model.IsResourceAdjustmentOperation(inst.CurrentOperation) {
		if def, ok := model.AdminStatusMap[statusResp.Status]; ok {
			statusResp.Actions = def.Actions
		}
	}

	var opUpdatedAt *string
	if inst.CurrentOperationUpdatedAt != nil {
		s := inst.CurrentOperationUpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
		opUpdatedAt = &s
	}

	// 兼容处理：空的 agent_type 默认为 openclaw
	agentType := model.NormalizeAgentType(inst.AgentType)

	var versionFetchedAt *string
	if inst.VersionFetchedAt != nil {
		s := inst.VersionFetchedAt.UTC().Format("2006-01-02T15:04:05Z")
		versionFetchedAt = &s
	}

	// 判断实例是否使用官方公共镜像
	isOfficial := false
	if cvmInfo != nil && cvmInfo.ImageId != "" {
		isOfficial = hcommon.IsCandidateImage(cvmInfo.ImageId)
	}

	// 提取实例标签
	var tags []CVMTag
	if cvmInfo != nil {
		tags = cvmInfo.Tags
	}
	if tags == nil {
		tags = []CVMTag{}
	}
	chargeType := instanceChargeTypeOrDefault(inst.InstanceChargeType)
	if cvmInfo != nil && cvmInfo.InstanceChargeType != "" {
		chargeType = cvmInfo.InstanceChargeType
	}

	resp := adminInstanceItemWithStatus{
		ID:                        inst.ID,
		Name:                      inst.Name,
		InstanceId:                inst.InstanceId,
		UserID:                    inst.UserID,
		Username:                  item.Username,
		GroupID:                   inst.GroupID,
		GroupFullPath:             "", // 稍后由调用方批量回填（需要按 GroupID 查 user_groups）
		CurrentOperation:          inst.CurrentOperation,
		CurrentOperationState:     inst.CurrentOperationState,
		AIModelID:                 inst.AIModelID,
		CurrentOperationUpdatedAt: opUpdatedAt,
		InstanceChargeType:        chargeType,
		LastCVMState:              inst.LastCVMState,
		AgentReady:                inst.AgentReady,
		RuntimeUser:               inst.RuntimeUser,
		RuntimeHome:               inst.RuntimeHome,
		CreatedAt:                 inst.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		AgentType:                 agentType,
		AgentVersion:              inst.AgentVersion,
		IsBuiltin:                 model.IsBuiltinAgentType(agentType),
		CompatibleWith:            model.GetAgentRuntimeType(ctx, agentType),
		PluginVersionStatus:       model.BuildPluginVersionStatus(inst.AgentVersion, inst.PluginVersionsJSON),
		VersionFetchedAt:          versionFetchedAt,
		IsOfficialImage:           isOfficial,
		Tags:                      tags,
		Department:                item.Department,
		Departments:               item.Departments,
		DepartmentPath:            item.DepartmentPath,
		Status:                    statusResp.Status,
		Label:                     statusResp.Label,
		Tooltip:                   statusResp.Tooltip,
		Actions:                   statusResp.Actions,
		Transient:                 statusResp.Transient,
		Source:                    inst.Source,
		// stale-instances v1.0：直接从 instances 行映射；Flags 由调用方批量回填
		Flags:                    []string{},
		HandoverTargetUserID:     inst.HandoverTargetUserID,
		HandoverRejectedByUserID: inst.HandoverRejectedByUserID,
		HandoverInitiatedAt:      formatNullableTime(inst.HandoverInitiatedAt),
	}
	populateAdminCVMFields(&resp, &inst, cvmInfo)
	populateAdminAdjustmentFields(ctx, &resp, item.Adjustment)
	if resp.IsBuiltin {
		resp.CompatibleWith = ""
	}
	return resp
}

// buildAdminStats 根据实例状态列表计算统计卡片
func buildAdminStats(ctx context.Context, items []adminInstanceItemWithStatus) adminStats {
	needAttentionStatuses := map[string]bool{
		model.StatusCreateFailed: true,
		model.StatusLoadFailed:   true,
		model.StatusMaintaining:  true,
		model.StatusPending:      true,
	}
	inProgressStatuses := map[string]bool{
		model.StatusCreating: true,
		model.StatusLoading:  true,
	}

	stats := adminStats{}
	stats.OtherDetail.NeedAttention.Label = i18n.T(ctx, i18n.MsgAdminStatsLabelNeedAttention)
	stats.OtherDetail.NeedAttention.Items = make(map[string]int)
	stats.OtherDetail.InProgress.Label = i18n.T(ctx, i18n.MsgAdminStatsLabelInProgress)
	stats.OtherDetail.InProgress.Items = make(map[string]int)

	for _, item := range items {
		stats.Total++
		switch item.Status {
		case model.StatusRunning:
			stats.Running++
		case model.StatusStopped:
			stats.Stopped++
		default:
			stats.Other++
			if needAttentionStatuses[item.Status] {
				stats.OtherDetail.NeedAttention.Count++
				stats.OtherDetail.NeedAttention.Items[item.Status]++
			} else if inProgressStatuses[item.Status] {
				stats.OtherDetail.InProgress.Count++
				stats.OtherDetail.InProgress.Items[item.Status]++
			}
		}
	}
	return stats
}

// matchStatusFilter 判断实例状态是否匹配状态筛选条件
// 支持多状态逗号分隔，如 "running,stopped"，任一匹配即返回 true
func matchStatusFilter(status, filter string) bool {
	filters := strings.Split(filter, ",")
	for _, f := range filters {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		switch f {
		case "running":
			if status == model.StatusRunning {
				return true
			}
		case "stopped":
			if status == model.StatusStopped {
				return true
			}
		case "other":
			if status != model.StatusRunning && status != model.StatusStopped {
				return true
			}
		default:
			// 具体状态精确匹配
			if status == f {
				return true
			}
		}
	}
	return false
}

// ── 纯 DB 读模式的辅助函数（改动4） ──

// whereBuilderForCache 构造过滤条件（stats 聚合 + 分页查询共用）。
func whereBuilderForCache(ctx context.Context, filter adminQueryFilter) *gorm.DB {
	base := model.DB(ctx).Model(&model.Instance{})

	if filter.Keyword != "" {
		kw := "%" + escapeSQLLike(filter.Keyword) + "%"
		base = base.Joins("LEFT JOIN users ON users.id = instances.user_id").
			Where(
				"instances.name LIKE ? OR instances.instance_id LIKE ? OR users.username LIKE ?",
				kw, kw, kw,
			)
	}
	if filter.Creator != "" {
		if filter.Keyword == "" {
			base = base.Joins("LEFT JOIN users ON users.id = instances.user_id")
		}
		base = base.Where("users.username = ?", filter.Creator)
	}
	if filter.DateFrom != "" {
		base = base.Where("DATE(instances.created_at) >= ?", filter.DateFrom)
	}
	if filter.DateTo != "" {
		base = base.Where("DATE(instances.created_at) <= ?", filter.DateTo)
	}
	if filter.DepartmentID != "" {
		departmentID := filter.DepartmentID
		base = base.Where(
			"instances.user_id IN (?)",
			model.DB(ctx).Model(&model.User{}).Select("id").
				Where(
					"one_id_sub IN (?)",
					model.DB(ctx).Model(&model.OneIDUserProfile{}).Select("one_id_sub").Where(
						"main_dept_id = ? OR departments_json LIKE ?", departmentID, model.DeptIDLikePattern(departmentID),
					),
				),
		)
	}
	if len(filter.AgentType) > 0 {
		base = base.Where("instances.agent_type IN ?", filter.AgentType)
	}
	base = applyAdminInstanceResourceFilters(base, filter)
	if filter.Source != "" {
		base = base.Where("instances.source = ?", filter.Source)
	}
	if len(filter.RequestIDs) > 0 {
		base = base.Where("instances.id IN ?", filter.RequestIDs)
	}
	if len(filter.RequestInstanceIDs) > 0 {
		base = base.Where("instances.instance_id IN ?", filter.RequestInstanceIDs)
	}

	// 标签过滤（短期近似：使用 cvm_tags_json LIKE 子串匹配，非精确等价替换，文档 7.3 确认接受）
	if filter.hasTagFilter() {
		if filter.TagKey != "" && len(filter.TagValues) > 0 {
			// tag_key + tag_values：匹配 JSON 中同时包含指定 key 和任一 value
			keyPattern := fmt.Sprintf(`%%"key":"%s"%%`, escapeSQLLike(filter.TagKey))
			var valueOrs []string
			var valueArgs []interface{}
			for _, v := range filter.TagValues {
				valueOrs = append(valueOrs, "instances.cvm_tags_json LIKE ?")
				valueArgs = append(valueArgs, fmt.Sprintf(`%%"value":"%s"%%`, escapeSQLLike(v)))
			}
			base = base.Where(
				"instances.cvm_tags_json LIKE ? AND ("+strings.Join(valueOrs, " OR ")+")",
				append([]interface{}{keyPattern}, valueArgs...)...,
			)
		} else if len(filter.TagKeys) > 0 {
			// tag_keys：匹配 JSON 中包含任一 key
			var keyOrs []string
			var keyArgs []interface{}
			for _, k := range filter.TagKeys {
				keyOrs = append(keyOrs, "instances.cvm_tags_json LIKE ?")
				keyArgs = append(keyArgs, fmt.Sprintf(`%%"key":"%s"%%`, escapeSQLLike(k)))
			}
			base = base.Where(strings.Join(keyOrs, " OR "), keyArgs...)
		}
	}
	return base
}

// queryAdminStatsFromCache 纯 DB 聚合 stats（GROUP BY last_known_status）。
func queryAdminStatsFromCache(ctx context.Context, filter adminQueryFilter) adminStats {
	type statusCount struct {
		LastKnownStatus string
		Count           int
	}
	var rows []statusCount
	if err := whereBuilderForCache(ctx, filter).
		Select("instances.last_known_status, COUNT(*) as count").
		Group("instances.last_known_status").
		Scan(&rows).Error; err != nil {
		slog.Error("[AdminList] 聚合查询 stats 失败", "error", err)
		return adminStats{}
	}

	needAttentionStatuses := map[string]bool{
		model.StatusCreateFailed: true,
		model.StatusLoadFailed:   true,
		model.StatusMaintaining:  true,
		model.StatusPending:      true,
	}
	inProgressStatuses := map[string]bool{
		model.StatusCreating: true,
		model.StatusLoading:  true,
	}

	stats := adminStats{}
	stats.OtherDetail.NeedAttention.Label = "⚠ 需关注"
	stats.OtherDetail.NeedAttention.Items = make(map[string]int)
	stats.OtherDetail.InProgress.Label = "◎ 处理中"
	stats.OtherDetail.InProgress.Items = make(map[string]int)

	for _, row := range rows {
		stats.Total += row.Count
		switch row.LastKnownStatus {
		case model.StatusRunning:
			stats.Running += row.Count
		case model.StatusStopped:
			stats.Stopped += row.Count
		default:
			stats.Other += row.Count
			if needAttentionStatuses[row.LastKnownStatus] {
				stats.OtherDetail.NeedAttention.Count += row.Count
				stats.OtherDetail.NeedAttention.Items[row.LastKnownStatus] += row.Count
			} else if inProgressStatuses[row.LastKnownStatus] {
				stats.OtherDetail.InProgress.Count += row.Count
				stats.OtherDetail.InProgress.Items[row.LastKnownStatus] += row.Count
			}
		}
	}
	return stats
}

// queryInstancesPageFromCache 纯 DB 分页查询（状态筛选走 WHERE last_known_status）。
func queryInstancesPageFromCache(ctx context.Context, page, pageSize int, filter adminQueryFilter, statusFilter string) ([]adminInstanceItem, int64) {
	base := whereBuilderForCache(ctx, filter)

	// 状态筛选
	if statusFilter != "" {
		statusValues := parseStatusFilterValues(statusFilter)
		if len(statusValues) > 0 {
			base = base.Where("instances.last_known_status IN ?", statusValues)
		}
	}

	// 需要 JOIN users 获取 username
	if filter.Keyword == "" && filter.Creator == "" {
		base = base.Joins("LEFT JOIN users ON users.id = instances.user_id")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		slog.Error("[AdminList] 查询实例总数失败", "error", err)
		return nil, 0
	}

	var items []adminInstanceItem
	if err := base.Select("instances.*, users.username").
		Order("instances.id desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error; err != nil {
		slog.Error("[AdminList] 查询实例分页失败", "error", err)
		return nil, total
	}
	attachInstanceAdjustments(ctx, items)

	// 批量补 OneID 部门信息
	userIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.UserID > 0 {
			userIDs = append(userIDs, item.UserID)
		}
	}
	var users []model.User
	if len(userIDs) > 0 {
		model.DB(ctx).Where("id IN ?", userIDs).Find(&users)
	}
	deptInfo := enrichUserDepartments(ctx, users)
	for i := range items {
		if d, ok := deptInfo[items[i].UserID]; ok {
			items[i].Department = d.Department
			items[i].Departments = d.Departments
			items[i].DepartmentPath = d.DepartmentPath
		}
	}

	return items, total
}

// parseStatusFilterValues 将状态筛选字符串解析为 last_known_status 值列表。
// "running" → [running], "stopped" → [stopped], "other" → [所有非 running/stopped 状态]
func parseStatusFilterValues(filter string) []string {
	var values []string
	for _, f := range strings.Split(filter, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		switch f {
		case "other":
			// other = 所有非 running/stopped 的已知状态
			values = append(
				values,
				model.StatusCreating, model.StatusCreateFailed,
				model.StatusLoading, model.StatusLoadFailed,
				model.StatusMaintaining, model.StatusPending,
				model.StatusDestroying, model.StatusDestroyed,
				model.StatusUpgrading, model.StatusUpgradeFailed,
				"", // 空状态也归入 other
			)
		default:
			values = append(values, f)
		}
	}
	return values
}

// buildAdminInstanceFromCache 从缓存字段组装响应（无 CVM 调用）。
func buildAdminInstanceFromCache(ctx context.Context, item adminInstanceItem) adminInstanceItemWithStatus {
	inst := item.Instance
	status := inst.LastKnownStatus
	if status == "" {
		status = model.StatusMaintaining // 兜底
	}

	// 获取状态展示信息
	var label, tooltip string
	var actions []string
	var transient bool
	defaultLang := hcommon.DefaultLangFromCtx(ctx)
	if def, ok := model.AdminStatusMap[status]; ok {
		if defaultLang == "zh" {
			label = def.Label
			tooltip = def.Tooltip
		} else {
			label = def.LabelEn
			tooltip = def.TooltipEn
		}
		actions = def.Actions
		transient = def.Transient
	}
	if model.IsResourceAdjustmentOperation(inst.CurrentOperation) {
		actions = []string{}
		transient = false
	}

	var opUpdatedAt *string
	if inst.CurrentOperationUpdatedAt != nil {
		s := inst.CurrentOperationUpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
		opUpdatedAt = &s
	}

	agentType := model.NormalizeAgentType(inst.AgentType)

	var versionFetchedAt *string
	if inst.VersionFetchedAt != nil {
		s := inst.VersionFetchedAt.UTC().Format("2006-01-02T15:04:05Z")
		versionFetchedAt = &s
	}

	// 标签从缓存字段反序列化
	var tags []CVMTag
	if inst.CVMTagsJSON != "" && inst.CVMTagsJSON != "[]" {
		_ = json.Unmarshal([]byte(inst.CVMTagsJSON), &tags)
	}
	if tags == nil {
		tags = []CVMTag{}
	}

	resp := adminInstanceItemWithStatus{
		ID:                        inst.ID,
		Name:                      inst.Name,
		InstanceId:                inst.InstanceId,
		UserID:                    inst.UserID,
		Username:                  item.Username,
		GroupID:                   inst.GroupID,
		GroupFullPath:             "",
		CurrentOperation:          inst.CurrentOperation,
		CurrentOperationState:     inst.CurrentOperationState,
		CurrentOperationUpdatedAt: opUpdatedAt,
		InstanceChargeType:        instanceChargeTypeOrDefault(inst.InstanceChargeType),
		LastCVMState:              inst.LastCVMState,
		AgentReady:                inst.AgentReady,
		RuntimeUser:               inst.RuntimeUser,
		RuntimeHome:               inst.RuntimeHome,
		CreatedAt:                 inst.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		AIModelID:                 inst.AIModelID,
		AgentType:                 agentType,
		AgentVersion:              inst.AgentVersion,
		IsBuiltin:                 model.IsBuiltinAgentType(agentType),
		CompatibleWith:            model.GetAgentRuntimeType(ctx, agentType),
		PluginVersionStatus:       model.BuildPluginVersionStatus(inst.AgentVersion, inst.PluginVersionsJSON),
		VersionFetchedAt:          versionFetchedAt,
		IsOfficialImage:           hcommon.IsCandidateImage(inst.ImgId),
		Tags:                      tags,
		Department:                item.Department,
		Departments:               item.Departments,
		DepartmentPath:            item.DepartmentPath,
		Status:                    status,
		Label:                     label,
		Tooltip:                   tooltip,
		Actions:                   actions,
		Transient:                 transient,
		Source:                    inst.Source,
		// stale-instances v1.0：直接从 instances 行里映射，无需额外查询
		Flags:                    []string{}, // 默认空数组（非 null）；enrichAdminInstancesWithStaleFields 后续会填充
		HandoverTargetUserID:     inst.HandoverTargetUserID,
		HandoverRejectedByUserID: inst.HandoverRejectedByUserID,
		HandoverInitiatedAt:      formatNullableTime(inst.HandoverInitiatedAt),
	}
	populateAdminCVMFields(&resp, &inst, nil)
	populateAdminAdjustmentFields(ctx, &resp, item.Adjustment)
	if resp.IsBuiltin {
		resp.CompatibleWith = ""
	}
	return resp
}

// populateAdminCVMFields copies cached CVM fields and applies live values when available.
func populateAdminCVMFields(
	resp *adminInstanceItemWithStatus,
	instance *model.Instance,
	cvmInfo *CVMInstanceInfo,
) {
	if resp == nil || instance == nil || instance.Source == model.InstanceSourceLocal {
		return
	}
	resp.CVMInstanceType = instance.CVMInstanceType
	resp.CPU = instance.CVMCPU
	resp.MemoryGB = instance.CVMMemoryGB
	resp.SystemDiskType = instance.SystemDiskType
	resp.SystemDiskSize = instance.SystemDiskSize
	resp.PublicIP = &instance.CVMPublicIP
	resp.InternetChargeType = &instance.CVMInternetChargeType
	resp.InternetMaxBandwidthOut = &instance.CVMInternetMaxBandwidthOut
	if cvmInfo == nil || cvmInfo.State == cvmAPIErrorState {
		return
	}
	resp.CVMInstanceType = cvmInfo.InstanceType
	resp.CPU = cvmInfo.CPU
	resp.MemoryGB = cvmInfo.MemoryGB
	resp.SystemDiskType = cvmInfo.SystemDiskType
	resp.SystemDiskSize = cvmInfo.SystemDiskSize
	resp.PublicIP = &cvmInfo.PublicIP
	resp.InternetChargeType = &cvmInfo.InternetChargeType
	resp.InternetMaxBandwidthOut = &cvmInfo.InternetMaxBandwidthOut
}

// populateAdminAdjustmentFields adds resource-adjustment state to an admin response.
func populateAdminAdjustmentFields(
	ctx context.Context,
	resp *adminInstanceItemWithStatus,
	adjustment *model.InstanceAdjustment,
) {
	if resp == nil || adjustment == nil {
		return
	}
	resp.AdjustmentStatus = adjustment.Status
	resp.AdjustmentType = adjustment.Type
	resp.AdjustmentErrorCode = adjustment.ErrorCode
	if adjustment.ErrorCode != "" {
		resp.AdjustmentErrorMessage = i18n.T(ctx, adjustmentReasonKey(adjustment.ErrorCode))
	}
	resp.AdjustmentUpdatedAt = formatNullableTime(&adjustment.UpdatedAt)
}

// enrichAdminItemsWithLocalInfo 对 enriched 里 source=local 的实例批量拉
// LocalInstanceInfo，回填 host_name / os / last_report_at 字段。
// CVM 实例不受影响（3 个字段保持零值，omitempty 不输出）。
// 错误走 log，不报错（list 接口应容忍 local_instance_infos 表查失败）。
func enrichAdminItemsWithLocalInfo(ctx context.Context, items []adminInstanceItemWithStatus) {
	if len(items) == 0 {
		return
	}
	var localPKs []uint
	for _, it := range items {
		if it.Source == model.InstanceSourceLocal {
			localPKs = append(localPKs, it.ID)
		}
	}
	if len(localPKs) == 0 {
		return
	}
	var infos []model.LocalInstanceInfo
	if err := model.DB(ctx).Where("instance_id IN ?", localPKs).Find(&infos).Error; err != nil {
		Logger(ctx).Warn("[AdminInstances] 批量查询 LocalInstanceInfo 失败", "error", err)
		return
	}
	infoMap := make(map[uint]model.LocalInstanceInfo, len(infos))
	for _, info := range infos {
		infoMap[info.InstanceID] = info
	}
	for i := range items {
		if items[i].Source != model.InstanceSourceLocal {
			continue
		}
		info, ok := infoMap[items[i].ID]
		if !ok {
			continue
		}
		items[i].HostName = info.HostName
		items[i].OS = info.OS
		if info.LastReportAt != nil {
			s := info.LastReportAt.UTC().Format(time.RFC3339)
			items[i].LastReportAt = &s
		}
	}
}

// enrichAdminItemsWithLocalProjects 批量回填本地 Agent Workspace 绑定的有效项目。
// 已删除项目不返回，避免管理端展示无法解析名称的历史 project_id。
func enrichAdminItemsWithLocalProjects(ctx context.Context, items []adminInstanceItemWithStatus) {
	localIDs := localInstanceIDs(items)
	if len(localIDs) == 0 {
		return
	}
	var bindings []model.LocalAgentScopeBinding
	if err := model.DB(ctx).Where("instance_id IN ? AND scope = ? AND project_id > 0", localIDs, model.LocalAgentScopeWorkspace).
		Order("instance_id ASC, project_id ASC, id ASC").Find(&bindings).Error; err != nil {
		Logger(ctx).Warn("[AdminInstances] 批量查询本地 Agent 项目失败", "error", err)
		return
	}
	projectMap := fetchProjectVisibilityMap(ctx, bindings)
	for i := range items {
		if items[i].Source != model.InstanceSourceLocal {
			continue
		}
		items[i].Projects = projectMap[items[i].ID]
		if items[i].Projects == nil {
			items[i].Projects = []projectVisibilityInfo{}
		}
	}
}

func localInstanceIDs(items []adminInstanceItemWithStatus) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		if item.Source == model.InstanceSourceLocal {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func fetchProjectVisibilityMap(ctx context.Context, bindings []model.LocalAgentScopeBinding) map[uint][]projectVisibilityInfo {
	projectIDs := make([]uint, 0, len(bindings))
	for _, binding := range bindings {
		projectIDs = append(projectIDs, binding.ProjectID)
	}
	if len(projectIDs) == 0 {
		return map[uint][]projectVisibilityInfo{}
	}
	var projects []model.Project
	if err := model.DB(ctx).Where("id IN ?", uniqueUintIDs(projectIDs)).Find(&projects).Error; err != nil {
		Logger(ctx).Warn("[AdminInstances] 批量查询项目名称失败", "error", err)
		return map[uint][]projectVisibilityInfo{}
	}
	names := make(map[uint]string, len(projects))
	for _, project := range projects {
		names[project.ID] = project.Name
	}
	result := make(map[uint][]projectVisibilityInfo)
	seen := make(map[uint]map[uint]struct{})
	for _, binding := range bindings {
		name, ok := names[binding.ProjectID]
		if !ok {
			continue
		}
		if seen[binding.InstanceID] == nil {
			seen[binding.InstanceID] = make(map[uint]struct{})
		}
		if _, ok := seen[binding.InstanceID][binding.ProjectID]; ok {
			continue
		}
		seen[binding.InstanceID][binding.ProjectID] = struct{}{}
		result[binding.InstanceID] = append(result[binding.InstanceID], projectVisibilityInfo{ProjectID: binding.ProjectID, ProjectName: name})
	}
	return result
}

func HandleAdminInstances(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	// 降级开关：后台缓存未就绪时走旧逻辑
	if !IsStatusCacheReady(r.Context()) {
		handleAdminInstancesLegacy(w, r)
		return
	}

	// ── 新逻辑：纯 DB 读（后台缓存已就绪） ──
	handleAdminInstancesCached(w, r)
}

// handleAdminInstancesCached 纯 DB 读的新 List 逻辑。
// 所有状态来自 last_known_status 缓存字段，零 CVM 调用、零写 DB。
func handleAdminInstancesCached(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r, 1000)

	q := r.URL.Query()
	keyword := q.Get("keyword")
	if runes := []rune(keyword); len(runes) > 50 {
		keyword = string(runes[:50])
	}
	filter := adminQueryFilter{
		Keyword:      keyword,
		Creator:      q.Get("creator"),
		DateFrom:     q.Get("date_from"),
		DateTo:       q.Get("date_to"),
		DepartmentID: q.Get("department_id"),
		Source:       q.Get("source"),
	}
	if raw := q.Get("agent_type"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				filter.AgentType = append(filter.AgentType, t)
			}
		}
	}
	if err := parseAdminInstancesIDFilters(q, &filter); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := parseAdminInstanceResourceFilters(q, &filter); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := parseAdminInstancesTagFilters(q, &filter); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	statusFilter := q.Get("status")

	// ① stats：SQL 聚合 GROUP BY last_known_status
	stats := queryAdminStatsFromCache(r.Context(), filter)

	// ② 分页：状态过滤进 WHERE，DB 层 LIMIT/OFFSET
	dbItems, total := queryInstancesPageFromCache(r.Context(), page, pageSize, filter, statusFilter)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	// ③ 组装响应
	enriched := make([]adminInstanceItemWithStatus, 0, len(dbItems))
	for _, item := range dbItems {
		built := buildAdminInstanceFromCache(r.Context(), item)
		enriched = append(enriched, built)
	}

	// ③+ 本地实例批量回填 host_name / os / last_report_at
	enrichAdminItemsWithLocalInfo(r.Context(), enriched)
	enrichAdminItemsWithLocalProjects(r.Context(), enriched)

	// ④ GroupFullPath 批量回填（保留）
	if len(enriched) > 0 {
		ids := make([]uint, 0, len(enriched))
		for _, it := range enriched {
			if it.GroupID != 0 {
				ids = append(ids, it.GroupID)
			}
		}
		pathMap := fetchGroupFullPathMap(r.Context(), ids)
		for i := range enriched {
			if enriched[i].GroupID != 0 {
				enriched[i].GroupFullPath = pathMap[enriched[i].GroupID]
			}
		}
	}

	// ⑤ 批量回填 stale-instances v1.0 字段：flags + handover_*
	enrichAdminInstancesWithStaleFields(r.Context(), enriched)

	jsonOK(w, map[string]interface{}{
		"instances":   enriched,
		"stats":       stats,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

// handleAdminInstancesLegacy 旧 List 逻辑（全量 CVM + 内存分页 + side effects）。
// 作为降级路径和灰度对比基线，待线上稳定运行后可删除。
func handleAdminInstancesLegacy(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r, 1000)

	// JSON API：使用新版查询参数 + 实时状态内联
	q := r.URL.Query()
	keyword := q.Get("keyword")
	// 按 rune 截断 50 个字符，避免切坏多字节 UTF-8 字符产生非法字节
	if runes := []rune(keyword); len(runes) > 50 {
		keyword = string(runes[:50])
	}
	filter := adminQueryFilter{
		Keyword:      keyword,
		Creator:      q.Get("creator"),
		DateFrom:     q.Get("date_from"),
		DateTo:       q.Get("date_to"),
		DepartmentID: q.Get("department_id"),
		Source:       q.Get("source"),
	}
	if raw := q.Get("agent_type"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				filter.AgentType = append(filter.AgentType, t)
			}
		}
	}
	// 🆕 ids / instance_ids 批量过滤(各上限 adminInstancesQueryMaxIDs)
	if err := parseAdminInstancesIDFilters(q, &filter); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := parseAdminInstanceResourceFilters(q, &filter); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	// 标签过滤（tag_keys 或 tag_key + tag_values）
	if err := parseAdminInstancesTagFilters(q, &filter); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	statusFilter := q.Get("status")

	// ── 第一步：轻量全量查询，获取所有实例的 ID 和 InstanceId（用于全局统计 + CVM 批量查询）──
	allLight, err := queryAllLightInstancesWithFilter(r.Context(), filter)
	if err != nil {
		slog.Error("[AdminList] 轻量全量查询失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 收集所有需要查询 CVM 的 instanceId（全量）
	allCvmIds := make([]string, 0, len(allLight))
	for _, item := range allLight {
		if item.InstanceId != "" {
			allCvmIds = append(allCvmIds, item.InstanceId)
		}
	}

	// 批量查询 CVM 状态（全量，当前页复用此结果）
	cvmStart := time.Now()
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), allCvmIds)
	slog.Info("[AdminList] 全量 CVM 查询完成", "count", len(allCvmIds), "duration", time.Since(cvmStart))

	// 标签过滤：基于已获取的 cvmInfoMap 在内存中筛掉不满足标签条件的实例
	// 这样后续 stats / 清理 / 分页都只在过滤后的集合上进行（与 keyword 等过滤语义一致）。
	// 注：API_ERROR 实例缺乏可信标签数据，会被排除。
	if filter.hasTagFilter() {
		filtered := allLight[:0]
		for _, item := range allLight {
			cvmInfo := cvmInfoMap[item.InstanceId]
			if filter.matchTagFilter(cvmInfo) {
				filtered = append(filtered, item)
			}
		}
		allLight = filtered
	}

	// ── 第二步：基于全量轻量数据计算全局状态，用于 stats 和清理 ──
	// 计算全量实例的实时状态
	type lightStatus struct {
		ID     uint
		Status string
	}
	allStatuses := make([]lightStatus, 0, len(allLight))
	for _, item := range allLight {
		// 构造临时 model.Instance 用于状态计算（只需状态相关字段）
		tmpInst := lightToInstance(item)
		cvmInfo := cvmInfoMap[item.InstanceId]
		statusResp := ResolveInstanceStatus(r.Context(), &tmpInst, cvmInfo, nil)
		allStatuses = append(allStatuses, lightStatus{ID: item.ID, Status: statusResp.Status})
	}

	// 清理已销毁超过 1 天的实例（基于全量数据）
	var destroyedIDs []uint
	for _, s := range allStatuses {
		if s.Status == model.StatusDestroyed {
			destroyedIDs = append(destroyedIDs, s.ID)
		}
	}
	cleanedSet := make(map[uint]bool)
	if len(destroyedIDs) > 0 {
		cleanedIDs := model.CleanupDestroyedInstances(r.Context(), destroyedIDs, 24*time.Hour)
		for _, id := range cleanedIDs {
			cleanedSet[id] = true
		}
	}

	// 从全量状态中移除已清理的实例，然后计算全局 stats
	globalStatusItems := make([]adminInstanceItemWithStatus, 0, len(allStatuses))
	for _, s := range allStatuses {
		if cleanedSet[s.ID] {
			continue
		}
		globalStatusItems = append(globalStatusItems, adminInstanceItemWithStatus{
			ID:     s.ID,
			Status: s.Status,
		})
	}
	stats := buildAdminStats(r.Context(), globalStatusItems)

	// 状态筛选：计算符合条件的全局 ID 集合（用于分页总数）
	var filteredGlobalIDs []uint
	for _, item := range globalStatusItems {
		if statusFilter == "" || matchStatusFilter(item.Status, statusFilter) {
			filteredGlobalIDs = append(filteredGlobalIDs, item.ID)
		}
	}
	total := int64(len(filteredGlobalIDs))
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	// ── 第三步：基于筛选后的 ID 列表分页，查询当前页的完整数据（带用户/部门信息）──
	// 对 filteredGlobalIDs 做内存分页，取出当前页对应的 ID 子集
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(filteredGlobalIDs) {
		start = len(filteredGlobalIDs)
	}
	if end > len(filteredGlobalIDs) {
		end = len(filteredGlobalIDs)
	}
	pageIDs := filteredGlobalIDs[start:end]

	var dbItems []adminInstanceItem
	if len(pageIDs) > 0 {
		pageFilter := filter
		pageFilter.IDs = pageIDs
		// 传入 page=1 + pageSize=len(pageIDs)，因为 IDs 已经是当前页的子集
		dbItems, _ = queryInstancesWithFilter(r.Context(), 1, len(pageIDs), pageFilter)
	}

	// 组装带状态的实例列表，并异步触发副作用
	var enriched []adminInstanceItemWithStatus
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	for _, item := range dbItems {
		// 跳过已清理的实例
		if cleanedSet[item.Instance.ID] {
			continue
		}
		cvmInfo := cvmInfoMap[item.Instance.InstanceId]
		built := buildAdminInstanceWithStatus(r.Context(), item, cvmInfo)
		enriched = append(enriched, built)
		// 异步更新 DB 副作用（last_cvm_state、操作超时、操作收敛等）
		inst := item.Instance
		wg.Add(1)
		sem <- struct{}{} // 获取信号量，达到上限时阻塞
		go func(ctx context.Context, i model.Instance, info *CVMInstanceInfo) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量
			statusResp := ResolveInstanceStatus(ctx, &i, info, nil)
			handleStatusSideEffects(ctx, model.DB(ctx), &i, info, statusResp.Status)
		}(hcommon.DetachContext(r.Context()), inst, cvmInfo)
	}
	wg.Wait()
	if enriched == nil {
		enriched = []adminInstanceItemWithStatus{}
	}

	// 🆕 本地实例批量回填 host_name / os / last_report_at
	enrichAdminItemsWithLocalInfo(r.Context(), enriched)
	enrichAdminItemsWithLocalProjects(r.Context(), enriched)

	// 🆕 批量回填 GroupFullPath：避免每条 instance 单独查 user_groups
	if len(enriched) > 0 {
		ids := make([]uint, 0, len(enriched))
		for _, it := range enriched {
			if it.GroupID != 0 {
				ids = append(ids, it.GroupID)
			}
		}
		pathMap := fetchGroupFullPathMap(r.Context(), ids)
		for i := range enriched {
			if enriched[i].GroupID != 0 {
				enriched[i].GroupFullPath = pathMap[enriched[i].GroupID]
			}
		}
	}

	// 批量回填 stale-instances v1.0 的 flags 字段（同 cached 路径）
	enrichAdminInstancesWithStaleFields(r.Context(), enriched)

	jsonOK(w, map[string]interface{}{
		"instances":   enriched,
		"stats":       stats,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

// adminDeleteMaxBatch 批量销毁单次请求的上限。与腾讯云 CVM TerminateInstances
// 单次最多 100 个 InstanceIds 保持一致，避免内部再分片。
const adminDeleteMaxBatch = 100

// adminDeleteResult 批量销毁时每个实例的结果。
type adminDeleteResult struct {
	ID         uint   `json:"id"`
	InstanceID string `json:"instance_id,omitempty"`
	Name       string `json:"name"`
	Status     string `json:"status"`  // "started" / "deleted" / "failed"
	Message    string `json:"message"` // 成功/失败原因
}

// parseAdminDeleteRequest 解析 /admin/instances/delete 的入参。
// 兼容规则：
//   - 同时支持 form/query 的 id 参数（旧版单删）和 JSON body 的 ids 字段（批量）。
//   - id / ids 至少传一个；两者都传时以 ids 为准，id 被忽略。
//   - 传 ids 时不允许为空列表；长度不能超过 adminDeleteMaxBatch(100)。
//
// 返回 (ids, isBatch, err)：
//   - isBatch=true 表示走批量分支，响应带 results[]；
//   - isBatch=false 表示旧的单删分支，响应 {"ok":true}（或 HTML 重渲染）。
func parseAdminDeleteRequest(r *http.Request) ([]uint, bool, error) {
	// 先尝试 JSON body（仅当 Content-Type 为 application/json 时）
	// 避免与旧 form 路径冲突：form/HTMX 客户端不会带 JSON body。
	var body struct {
		IDs         *[]uint   `json:"ids"`
		ID          *uint     `json:"id"`
		InstanceIDs *[]string `json:"instance_ids"`
		InstanceID  *string   `json:"instance_id"`
	}
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") && r.Body != nil {
		// 即便 body 为 `{}` 或为空也不当错误，落到后续 id 分支
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	// ids 优先（body.IDs != nil 即表示客户端显式传了 ids 字段）
	if body.IDs != nil {
		if len(*body.IDs) == 0 {
			return nil, true, hcommon.I18nError(i18n.MsgIDsEmptyList)
		}
		if len(*body.IDs) > adminDeleteMaxBatch {
			return nil, true, hcommon.I18nError(i18n.MsgIDsCountExceed, adminDeleteMaxBatch)
		}
		// 去重 + 过滤 0
		seen := make(map[uint]struct{}, len(*body.IDs))
		ids := make([]uint, 0, len(*body.IDs))
		for _, id := range *body.IDs {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			return nil, true, hcommon.I18nError(i18n.MsgIDsContainZeroOrDuplicate)
		}
		return ids, true, nil
	}

	// instance_ids 批量（body.InstanceIDs != nil 且 ids 未传）
	if body.InstanceIDs != nil {
		if len(*body.InstanceIDs) == 0 {
			return nil, true, hcommon.I18nError(i18n.MsgInstanceIdsEmptyList)
		}
		if len(*body.InstanceIDs) > adminDeleteMaxBatch {
			return nil, true, hcommon.I18nError(i18n.MsgTooManyInstanceIDs, adminDeleteMaxBatch)
		}
		var instances []model.Instance
		if err := model.DB(r.Context()).Select("id").Where("instance_id IN ?", *body.InstanceIDs).Find(&instances).Error; err != nil {
			return nil, true, hcommon.I18nRichError(err, i18n.MsgQueryInstancesByIDsFailed)
		}
		if len(instances) == 0 {
			return nil, true, hcommon.I18nError(i18n.MsgInstanceIdsNotFound)
		}
		ids := make([]uint, 0, len(instances))
		for _, inst := range instances {
			ids = append(ids, inst.ID)
		}
		return ids, true, nil
	}

	// body.id（JSON）或 form id 兜底
	if body.ID != nil && *body.ID > 0 {
		return []uint{*body.ID}, false, nil
	}
	if raw := r.FormValue("id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || id == 0 {
			return nil, false, hcommon.I18nError(i18n.MsgInvalidID)
		}
		return []uint{uint(id)}, false, nil
	}

	// instance_id 单个（JSON body 或 form）
	if body.InstanceID != nil && *body.InstanceID != "" {
		var inst model.Instance
		if model.DB(r.Context()).Select("id").Where("instance_id = ?", *body.InstanceID).First(&inst).Error != nil {
			return nil, false, hcommon.I18nError(i18n.MsgInstanceNotFound)
		}
		return []uint{inst.ID}, false, nil
	}
	if raw := r.FormValue("instance_id"); raw != "" {
		var inst model.Instance
		if model.DB(r.Context()).Select("id").Where("instance_id = ?", raw).First(&inst).Error != nil {
			return nil, false, hcommon.I18nError(i18n.MsgInstanceNotFound)
		}
		return []uint{inst.ID}, false, nil
	}

	return nil, false, hcommon.I18nError(i18n.MsgMissingIDOrInstanceID)
}

func cleanupAgentProxyRoutes(ctx context.Context, prefix string, instanceID string, instanceModelID uint) {
	if err := model.DisableAgentProxyRoutesForInstance(ctx, instanceID); err != nil {
		slog.Warn(prefix+" 禁用代理路由失败", "id", instanceModelID, "instance_id", instanceID, "error", err)
		return
	}
	if err := RefreshAllRuleSetsForRequiredRules(ctx); err != nil {
		slog.Warn(prefix+" 刷新代理安全组规则失败", "id", instanceModelID, "instance_id", instanceID, "error", err)
	}
}

// asyncPurgeAndCleanup 异步 Purge CVM + 释放 Pro 记忆库 + 清 DB + 发送管理员删除通知。
// 提取自原 HandleAdminDeleteInstance 的成功分支，供单删与批量分支共用。
func asyncPurgeAndCleanup(ctx context.Context, inst model.Instance) {
	go func(inst model.Instance) {
		slog.Info("[AdminDelete] 异步 Purge 开始", "instance_id", inst.InstanceId)
		if err := destroyCVMInstance(ctx, inst.InstanceId); err != nil {
			slog.Warn("[AdminDelete] Purge 失败", "instance_id", inst.InstanceId, "error", err)
			// Purge 失败时仍继续后续清理，避免僵尸记录（CVM 已处于 ISOLATED，资源已释放）
		}
		// 释放远端 Pro 记忆库；失败则保留 instance 与 plugin 行，
		// 留给 instance_state 副作用按 cvmInfo==nil 分支补偿（下一轮轮询自然触发）。
		if !ReleaseProMemSpaceForMissingInstance(ctx, inst.InstanceId) {
			slog.Warn("[AdminDelete] Pro 记忆库释放失败，保留 instance 与 plugin 行等待 instance_state 副作用补偿",
				"id", inst.ID, "instance_id", inst.InstanceId)
			return
		}
		model.DB(ctx).Where("instance_id = ?", inst.ID).Delete(&model.SkillInstallation{})
		cleanupAgentProxyRoutes(ctx, "[AdminDelete]", inst.InstanceId, inst.ID)
		model.DB(ctx).Delete(&inst)
		model.DeleteMemoryTDAIPluginRow(ctx, inst.InstanceId)
		MarkPersonalSpaceToBeDeleted(ctx, inst.ID)
		if err := model.CreateNotification(
			ctx,
			inst.UserID, inst.ID, inst.Name,
			model.NotifyTypeAdminDelete,
			"实例已被管理员删除",
			fmt.Sprintf("您的实例「%s」已被管理员删除，如有疑问请联系管理员。", inst.Name),
		); err != nil {
			slog.Warn("[AdminDelete] 创建通知失败", "id", inst.ID, "error", err)
		}
	}(inst)
}

// cleanupForMissingCVM 当 CVM 已不存在时（占位记录或 Terminate 返回
// InvalidInstanceId.NotFound）做本地清理：释放 Pro 记忆库 + 清 DB + 通知。
// 返回 (ok, errMsg)：ok=false 表示 Pro 记忆库释放失败，需保留 DB 记录等待补偿。
func cleanupForMissingCVM(ctx context.Context, instance model.Instance, sendNotify bool) (bool, error) {
	if !ReleaseProMemSpaceForMissingInstance(ctx, instance.InstanceId) {
		return false, hcommon.I18nError(i18n.MsgProMemReleaseFailedWaitRetry)
	}
	model.DB(ctx).Where("instance_id = ?", instance.ID).Delete(&model.SkillInstallation{})
	cleanupAgentProxyRoutes(ctx, "[AdminDelete]", instance.InstanceId, instance.ID)
	model.DB(ctx).Delete(&instance)
	model.DeleteMemoryTDAIPluginRow(ctx, instance.InstanceId)
	MarkPersonalSpaceToBeDeleted(ctx, instance.ID)
	if sendNotify {
		go func(ctx context.Context, inst model.Instance) {
			if err := model.CreateNotification(
				ctx,
				inst.UserID, inst.ID, inst.Name,
				model.NotifyTypeAdminDelete,
				"实例已被管理员删除",
				fmt.Sprintf("您的实例「%s」已被管理员删除，如有疑问请联系管理员。", inst.Name),
			); err != nil {
				slog.Warn("[AdminDelete] 创建通知失败", "id", inst.ID, "error", err)
			}
		}(hcommon.DetachContext(ctx), instance)
	}
	return true, nil
}

func HandleAdminDeleteInstance(w http.ResponseWriter, r *http.Request) {
	handleAdminDeleteInstance(w, r, defaultStatusResolver)
}

func handleAdminDeleteInstance(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if !requireAdmin(w, r) {
		return
	}

	ids, isBatch, err := parseAdminDeleteRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 批量分支：强制 JSON 响应
	if isBatch {
		jsonAPI(w)
		handleAdminBatchDelete(w, r, ids)
		return
	}

	// ── 单删分支（旧行为，保持不动） ──
	id := ids[0]

	var instance model.Instance
	if model.DB(r.Context()).First(&instance, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFound))
		return
	}
	if instance.IsDoctorNode {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}
	// 本地 agent 实例：admin 不走 Terminate/CVM API 路径；user 端已有专门的本地刭除分支。
	if instance.Source == model.InstanceSourceLocal {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalInstanceUnsupportedOp))
		return
	}

	// 状态准入：AdminStatusMap[status] 必须允许 delete。
	// 禁止列表：creating / loading / pending / upgrading / upgrade_failed
	if _, err := requireActionAllowedForAdmin(r.Context(), &instance, "delete", resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	// 分化删除：无 InstanceId（创建失败等）→ 不调用 CVM API，按“CVM 已不存在”策略尽力释放 Pro 记忆库
	if instance.InstanceId == "" {
		if ok, err := cleanupForMissingCVM(r.Context(), instance, false); !ok {
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		model.DB(r.Context()).Where("instance_id = ?", instance.ID).Delete(&model.SkillInstallation{})
		MarkInstanceUnbound(r.Context(), model.CurrentIdentifier(r.Context()), instance.SecurityGroupId)
		cleanupAgentProxyRoutes(r.Context(), "[AdminDelete]", instance.InstanceId, instance.ID)
		model.DB(r.Context()).Delete(&instance)
		model.DeleteMemoryTDAIPluginRow(r.Context(), instance.InstanceId)

		// 删除实例后，个人空间变为待删除（进入回收站）
		MarkPersonalSpaceToBeDeleted(r.Context(), instance.ID)

		jsonOK(w, map[string]interface{}{"ok": true})
		return
	}

	// 正常实例：TerminateInstances → 异步 Purge → 释放 Pro 记忆库 → 清 DB + 发送 admin_delete 通知
	// 注意：管理员路径必须主动释放 Pro 记忆库，不能依赖 instance_state 轮询副作用——
	// 因为本 handler 在 Purge 后会立即删除 instance 行，下一轮轮询扫不到目标，补偿窗口关闭。
	// 释放失败时保留 instance 与 plugin 行，留给 instance_state 副作用按 cvmInfo==nil 分支自动补偿。

	client, cvmErr := NewCVMClient(r.Context())
	if cvmErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(cvmErr, i18n.MsgCreateCVMClientFailed))
		return
	}
	req := cvm.NewTerminateInstancesRequest()
	req.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
	if _, err := client.TerminateInstances(req); err != nil {
		if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok && sdkErr.GetCode() == "InvalidInstanceId.NotFound" {
			// CVM 已不存在：主动释放远端 Pro 记忆库再清本地记录
			if ok, err := cleanupForMissingCVM(r.Context(), instance, true); !ok {
				slog.Error("[AdminDelete] CVM 已不存在但 Pro 记忆库释放失败，保留 DB 记录等待 instance_state 副作用补偿",
					"id", instance.ID, "instance_id", instance.InstanceId)
				writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
				return
			}
			model.DB(r.Context()).Where("instance_id = ?", instance.ID).Delete(&model.SkillInstallation{})
			MarkInstanceUnbound(r.Context(), model.CurrentIdentifier(r.Context()), instance.SecurityGroupId)
			cleanupAgentProxyRoutes(r.Context(), "[AdminDelete]", instance.InstanceId, instance.ID)
			model.DB(r.Context()).Delete(&instance)
			model.DeleteMemoryTDAIPluginRow(r.Context(), instance.InstanceId)
			go func(ctx context.Context, inst model.Instance) {
				if err := model.CreateNotification(
					ctx,
					inst.UserID, inst.ID, inst.Name,
					model.NotifyTypeAdminDelete,
					"实例已被管理员删除",
					fmt.Sprintf("您的实例「%s」已被管理员删除，如有疑问请联系管理员。", inst.Name),
				); err != nil {
					slog.Warn("[AdminDelete] 创建通知失败", "id", inst.ID, "error", err)
				}
			}(hcommon.DetachContext(r.Context()), instance)
		} else {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCVMInstanceDestroyFailed))
			return
		}
	} else {
		// TerminateInstances 成功（CVM 进入 ISOLATED），异步 Purge + 释放 Pro 记忆库 + 清 DB + 通知
		asyncPurgeAndCleanup(hcommon.DetachContext(r.Context()), instance)
	}

	// 删除实例后，个人空间变为待删除（进入回收站）
	MarkPersonalSpaceToBeDeleted(r.Context(), instance.ID)

	jsonOK(w, map[string]interface{}{"ok": true})
}

// handleAdminBatchDelete 处理批量销毁：一次 CVM TerminateInstances 调用，失败时
// 回退逐个处理以定位 NotFound 实例，保留原单删行为。
//
// 响应格式：{"ok": true, "results": [{id, instance_id, name, status, message}]}
// status 取值：
//   - "started"  —— TerminateInstances 成功下发，异步 Purge/清 DB 已启动
//   - "deleted"  —— CVM 已不存在，本地记录已即时清理完成
//   - "failed"   —— 该实例删除未成功，message 给原因
func handleAdminBatchDelete(w http.ResponseWriter, r *http.Request, ids []uint) {
	// 1) 批量查 DB
	var instances []model.Instance
	if err := model.DB(r.Context()).Where("id IN ?", ids).Find(&instances).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	dbByID := make(map[uint]*model.Instance, len(instances))
	for i := range instances {
		dbByID[instances[i].ID] = &instances[i]
	}

	// admin 批量删除：本地 agent 实例不走 CVM/TAT 路径，请从用户视角单独删除。
	// 不静默裁剪跳过，避免用户以为本地实例被躲在结果里成功了。
	var localIDs []uint
	for _, inst := range instances {
		if inst.Source == model.InstanceSourceLocal {
			localIDs = append(localIDs, inst.ID)
		}
	}
	if len(localIDs) > 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalInstanceUnsupportedOp))
		return
	}

	results := make([]adminDeleteResult, 0, len(ids))
	// 不存在的 id 直接记 failed
	for _, id := range ids {
		if _, ok := dbByID[id]; !ok {
			results = append(results, adminDeleteResult{
				ID:      id,
				Status:  "failed",
				Message: i18n.T(r.Context(), i18n.MsgInstanceNotFound),
			})
		}
	}

	// 2) 分两类：无 InstanceId（占位）→ 就地清理；有 InstanceId → 合批 Terminate
	var withCVM []*model.Instance
	for _, inst := range instances {
		inst := inst
		if err := setOperation(model.DB(r.Context()), &inst, model.OpDelete); err != nil {
			results = append(results, adminDeleteResult{
				ID:         inst.ID,
				InstanceID: inst.InstanceId,
				Name:       inst.Name,
				Status:     "failed",
				Message:    i18n.T(r.Context(), i18n.MsgOperationInProgress),
			})
			continue
		}
		if inst.InstanceId == "" {
			if ok, richErr := cleanupForMissingCVM(r.Context(), inst, false); !ok {
				results = append(results, adminDeleteResult{
					ID:      inst.ID,
					Name:    inst.Name,
					Status:  "failed",
					Message: hcommon.ErrorMessageWithCtx(r.Context(), richErr),
				})
			} else {
				results = append(results, adminDeleteResult{
					ID:      inst.ID,
					Name:    inst.Name,
					Status:  "deleted",
					Message: i18n.T(r.Context(), i18n.MsgPlaceholderInstanceCleaned),
				})
			}
			continue
		}
		withCVM = append(withCVM, &inst)
	}

	// 3) 无 CVM 可调则直接返回
	if len(withCVM) == 0 {
		jsonOK(w, map[string]interface{}{"ok": true, "results": results})
		return
	}

	client, cvmErr := NewCVMClient(r.Context())
	if cvmErr != nil {
		// 客户端创建失败 → 全部标 failed
		for _, inst := range withCVM {
			results = append(results, adminDeleteResult{
				ID: inst.ID, InstanceID: inst.InstanceId, Name: inst.Name,
				Status: "failed", Message: i18n.T(r.Context(), i18n.MsgCreateCVMClientFailedFmt, cvmErr),
			})
		}
		jsonOK(w, map[string]interface{}{"ok": true, "results": results})
		return
	}

	// 4) 一次批量调用（len ≤ 100，与 CVM 单次上限对齐，所以这里绝不会再分片）
	cvmIDs := make([]string, len(withCVM))
	for i, inst := range withCVM {
		cvmIDs[i] = inst.InstanceId
	}
	req := cvm.NewTerminateInstancesRequest()
	req.InstanceIds = common.StringPtrs(cvmIDs)
	_, batchErr := client.TerminateInstances(req)

	if batchErr == nil {
		// 4a) 批量成功：全部进入 async purge + 标 started
		for _, inst := range withCVM {
			clearAdjustmentFailure(r.Context(), inst.ID)
			asyncPurgeAndCleanup(hcommon.DetachContext(r.Context()), *inst)
			MarkPersonalSpaceToBeDeleted(hcommon.DetachContext(r.Context()), inst.ID)
			results = append(results, adminDeleteResult{
				ID: inst.ID, InstanceID: inst.InstanceId, Name: inst.Name,
				Status: "started", Message: i18n.T(r.Context(), i18n.MsgDestroyDispatchedAsyncCleanup),
			})
		}
		slog.Info("[AdminBatchDelete] 批量销毁下发成功", "admin", getAdminUser(r), "count", len(withCVM))
		jsonOK(w, map[string]interface{}{"ok": true, "results": results})
		return
	}

	// 4b) 批量失败：若是 NotFound 错误（批量里至少 1 个 CVM 已不存在），
	// 回退到逐个处理以精确定位每个实例的结果；其他错误（鉴权/网络等）直接全标 failed。
	if sdkErr, ok := batchErr.(*sdkerrors.TencentCloudSDKError); ok && sdkErr.GetCode() == "InvalidInstanceId.NotFound" {
		slog.Warn("[AdminBatchDelete] 批量销毁含已不存在实例，回退逐个处理",
			"admin", getAdminUser(r), "count", len(withCVM), "error", batchErr)
		for _, inst := range withCVM {
			one := cvm.NewTerminateInstancesRequest()
			one.InstanceIds = common.StringPtrs([]string{inst.InstanceId})
			if _, err := client.TerminateInstances(one); err != nil {
				if sdkErr2, ok := err.(*sdkerrors.TencentCloudSDKError); ok && sdkErr2.GetCode() == "InvalidInstanceId.NotFound" {
					if ok, richErr := cleanupForMissingCVM(r.Context(), *inst, true); !ok {
						slog.Error("[AdminBatchDelete] CVM 已不存在但 Pro 记忆库释放失败",
							"id", inst.ID, "instance_id", inst.InstanceId)
						results = append(results, adminDeleteResult{
							ID: inst.ID, InstanceID: inst.InstanceId, Name: inst.Name,
							Status: "failed", Message: hcommon.ErrorMessageWithCtx(r.Context(), richErr),
						})
						continue
					}
					results = append(results, adminDeleteResult{
						ID: inst.ID, InstanceID: inst.InstanceId, Name: inst.Name,
						Status: "deleted", Message: i18n.T(r.Context(), i18n.MsgCVMNotExistLocalCleaned),
					})
					continue
				}
				results = append(results, adminDeleteResult{
					ID: inst.ID, InstanceID: inst.InstanceId, Name: inst.Name,
					Status: "failed", Message: i18n.T(r.Context(), i18n.MsgDestroyCVMInstanceFailedFmt, err),
				})
				continue
			}
			clearAdjustmentFailure(r.Context(), inst.ID)
			asyncPurgeAndCleanup(hcommon.DetachContext(r.Context()), *inst)
			MarkPersonalSpaceToBeDeleted(hcommon.DetachContext(r.Context()), inst.ID)
			results = append(results, adminDeleteResult{
				ID: inst.ID, InstanceID: inst.InstanceId, Name: inst.Name,
				Status: "started", Message: i18n.T(r.Context(), i18n.MsgDestroyDispatchedAsyncCleanup),
			})
		}
		jsonOK(w, map[string]interface{}{"ok": true, "results": results})
		return
	}

	// 其他错误：全标 failed
	slog.Error("[AdminBatchDelete] 批量销毁失败", "admin", getAdminUser(r), "count", len(withCVM), "error", batchErr)
	for _, inst := range withCVM {
		results = append(results, adminDeleteResult{
			ID: inst.ID, InstanceID: inst.InstanceId, Name: inst.Name,
			Status: "failed", Message: i18n.T(r.Context(), i18n.MsgDestroyCVMInstanceFailedFmt, batchErr),
		})
	}
	jsonOK(w, map[string]interface{}{"ok": true, "results": results})
}

// HandleAdminInstanceTerminal 为管理员获取指定 claw 实例的终端授权访问 URL。
//
// 路由: POST /admin/instances/terminal-url
//
// 流程:
//  1. 根据实例 ID 查询 claw 对应的 CVM InstanceId
//  2. 调用 orcaterm GenerateAuthLoginUrl 获取终端登录 URL
//
// 请求参数:
//
//	id: 实例 ID（form 参数）
//
// 响应:
//
//	{ "login_url": "https://orcaterm.cloud.tencent.com/terminal?..." }
func HandleAdminInstanceTerminal(w http.ResponseWriter, r *http.Request) {
	handleAdminInstanceTerminal(w, r, defaultStatusResolver)
}

func handleAdminInstanceTerminal(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	instance, err := getAdminInstanceByIDOrInstanceID(r)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	if instance.IsDoctorNode {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}

	// 状态准入：仅 running 允许进入终端
	if _, err := requireActionAllowedForAdmin(r.Context(), instance, "terminal", resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 查询 CVM 实例信息，获取 Region、UserName、PlatformType
	cvmClient, cvmErr := NewCVMClient(r.Context())
	if cvmErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(cvmErr, i18n.MsgCreateCVMClientFailed))
		return
	}

	descReq := cvm.NewDescribeInstancesRequest()
	descReq.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
	var descResp *cvm.DescribeInstancesResponse
	descErr := RetryCloudCall(r.Context(), func() error {
		var callErr error
		descResp, callErr = cvmClient.DescribeInstances(descReq)
		return callErr
	})
	if descErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(descErr, i18n.MsgQueryCVMInstanceFailed))
		return
	}
	if descResp.Response == nil || len(descResp.Response.InstanceSet) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCVMInstanceNotExist, instance.InstanceId))
		return
	}

	inst := descResp.Response.InstanceSet[0]

	// 使用默认的 CVMRegion
	instanceRegion := CVMRegion

	// 判断是否为 Windows 系统
	isWindows := inst.OsName != nil && strings.Contains(strings.ToLower(*inst.OsName), "windows")

	// UserName: 优先使用实例实际检测到的 RuntimeUser，
	// 未检测到时 fallback 到 root。Windows 实例使用 Administrator。
	userName := getEffectiveRuntimeUser(instance.RuntimeUser)
	if isWindows {
		userName = "Administrator"
	}

	slog.Info(
		"[CVM] 查询实例信息成功",
		"instance_id", instance.InstanceId,
		"region", instanceRegion,
		"user_name", userName,
		"os_name", StrVal(inst.OsName),
	)

	credential, credErr := getCredential(r.Context())
	if credErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(credErr, i18n.MsgGetCloudCredentialFailed))
		return
	}

	loginURL, loginErr := generateAuthLoginUrl(credential, instance.InstanceId, instanceRegion, userName)
	if loginErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(loginErr, i18n.MsgGetTerminalLoginURLFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"login_url": loginURL,
	})
}

// deniedAction 表示一个被禁用的 CVM 操作。
type deniedAction struct {
	Action  string `json:"action"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// instanceDeniedActions 表示单个实例及其被禁用的操作列表。
type instanceDeniedActions struct {
	ID            uint           `json:"id"`
	DeniedActions []deniedAction `json:"denied_actions"`
}

// HandleAdminInstanceDeniedActions 管理员批量查询 claw 实例对应 CVM 的禁用操作。
// 仅返回 DescribeInstanceVncUrl 相关的 DeniedAction。
//
// 路由: POST /admin/instances/denied-actions
//
// 请求体 (JSON):
//
//	{"ids": [1, 2, 3]}
//
// 响应:
//
//	{"instances": [{"id": 1, "denied_actions": [{"action":"...","code":"...","message":"..."}]}, ...]}
func HandleAdminInstanceDeniedActions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	if len(req.IDs) == 0 {
		jsonOK(w, map[string]interface{}{"instances": []interface{}{}})
		return
	}

	// 管理员可查询任意实例
	var instances []model.Instance
	if err := model.DB(r.Context()).Where("id IN ?", req.IDs).Find(&instances).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	if len(instances) == 0 {
		jsonOK(w, map[string]interface{}{"instances": []interface{}{}})
		return
	}

	// 提取关联 CVM 的实例（本地 agent 实例 source=local 的 instance_id 是 host CID，
	// 不是 CVM 格式的 ins-xxxxxxxx，不能传给 CVM API）
	var cvmInstanceIds []string
	for _, inst := range instances {
		if inst.InstanceId != "" && inst.Source != model.InstanceSourceLocal {
			cvmInstanceIds = append(cvmInstanceIds, inst.InstanceId)
		}
	}

	// 如果没有关联 CVM 的实例，直接返回空 denied_actions
	if len(cvmInstanceIds) == 0 {
		result := make([]instanceDeniedActions, len(instances))
		for i, inst := range instances {
			result[i] = instanceDeniedActions{ID: inst.ID, DeniedActions: []deniedAction{}}
		}
		jsonOK(w, map[string]interface{}{"instances": result})
		return
	}

	cvmDenied, err := describeInstancesDeniedActions(r.Context(), cvmInstanceIds, []string{"DescribeInstanceVncUrl"})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceDeniedOpsFailed))
		return
	}

	// 组装最终结果
	result := make([]instanceDeniedActions, 0, len(instances))
	for _, inst := range instances {
		item := instanceDeniedActions{ID: inst.ID}
		if inst.InstanceId != "" {
			if denied, ok := cvmDenied[inst.InstanceId]; ok {
				item.DeniedActions = denied
			}
		}
		if item.DeniedActions == nil {
			item.DeniedActions = []deniedAction{}
		}
		result = append(result, item)
	}

	jsonOK(w, map[string]interface{}{"instances": result})
}

// describeInstancesDeniedActions 批量查询指定 CVM 实例的 DeniedActions，
// 仅保留 filterActions 中指定的 Action（为空切片则保留全部）。
// 返回 CVM InstanceId → []deniedAction 的映射。
func describeInstancesDeniedActions(ctx context.Context, cvmInstanceIds []string, filterActions []string) (map[string][]deniedAction, error) {
	credential, err := getCredential(ctx)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgGetCloudCredentialFailed).WithDetail(err.Error())
	}

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"
	cpf.HttpProfile.ReqMethod = "POST"
	client := common.NewCommonClient(credential, CVMRegion, cpf)

	request := tchttp.NewCommonRequest("cvm", "2017-03-12", "DescribeInstancesDeniedActions")
	params := map[string]interface{}{
		"InstanceIds": cvmInstanceIds,
	}
	paramsJSON, marshalErr := json.Marshal(params)
	if marshalErr != nil {
		return nil, hcommon.I18nRichError(marshalErr, i18n.MsgSerializeRequestParamsFailed)
	}
	if err := request.SetActionParameters(string(paramsJSON)); err != nil {
		return nil, hcommon.I18nError(i18n.MsgSetRequestParamsFailed).WithDetail(err.Error())
	}

	response := tchttp.NewCommonResponse()
	if err := RetryCloudCall(ctx, func() error {
		return client.Send(request, response)
	}); err != nil {
		return nil, hcommon.I18nError(i18n.MsgQueryInstanceDeniedOpsFailed).WithDetail(err.Error())
	}

	// 解析响应
	var apiResp struct {
		Response struct {
			InstanceDeniedActionSet []struct {
				InstanceId    string `json:"InstanceId"`
				DeniedActions []struct {
					Action  string `json:"Action"`
					Code    string `json:"Code"`
					Message string `json:"Message"`
				} `json:"DeniedActions"`
			} `json:"InstanceDeniedActionSet"`
			RequestId string `json:"RequestId"`
			Error     *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error,omitempty"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(response.GetBody(), &apiResp); err != nil {
		return nil, hcommon.I18nError(i18n.MsgParseResponseFailed).WithDetail(err.Error())
	}
	if apiResp.Response.Error != nil {
		return nil, hcommon.I18nError(i18n.MsgAPIErrorFormat, apiResp.Response.Error.Code, apiResp.Response.Error.Message)
	}

	// 构建过滤集合，提高查找效率
	filterSet := make(map[string]struct{}, len(filterActions))
	for _, a := range filterActions {
		filterSet[a] = struct{}{}
	}

	// 构建 CVM InstanceId -> filtered denied actions 的映射
	result := make(map[string][]deniedAction)
	for _, item := range apiResp.Response.InstanceDeniedActionSet {
		var filtered []deniedAction
		for _, da := range item.DeniedActions {
			if len(filterSet) == 0 {
				// 无过滤条件，保留全部
				filtered = append(filtered, deniedAction{
					Action:  da.Action,
					Code:    da.Code,
					Message: da.Message,
				})
			} else if _, ok := filterSet[da.Action]; ok {
				filtered = append(filtered, deniedAction{
					Action:  da.Action,
					Code:    da.Code,
					Message: da.Message,
				})
			}
		}
		if filtered == nil {
			filtered = []deniedAction{}
		}
		result[item.InstanceId] = filtered
	}

	return result, nil
}

// generateAuthLoginUrl 调用腾讯云 OrcaTerm GenerateAuthLoginUrl 接口，
// 获取指定 CVM 实例的终端授权登录 URL。
// credential 由调用方提供，endpoint 为空时使用默认的 orcaterm.tencentcloudapi.com。
func generateAuthLoginUrl(credential *common.Credential, instanceId, instanceRegion, userName string) (string, error) {
	return generateAuthLoginUrlWithEndpoint(credential, "", instanceId, instanceRegion, userName)
}

// generateAuthLoginUrlWithEndpoint 是 generateAuthLoginUrl 的完整版本，
// 支持自定义 endpoint，主要用于调试和验证。
func generateAuthLoginUrlWithEndpoint(credential *common.Credential, endpoint, instanceId, instanceRegion, userName string) (string, error) {
	if endpoint == "" {
		endpoint = "orcaterm.tencentcloudapi.com"
	}

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = endpoint
	cpf.HttpProfile.ReqMethod = "POST"

	client := common.NewCommonClient(credential, instanceRegion, cpf)

	request := tchttp.NewCommonRequest("orcaterm", "2023-04-18", "GenerateAuthLoginUrl")
	params := map[string]interface{}{
		"InstanceRegion": instanceRegion,
		"InstanceId":     instanceId,
		"InstanceType":   "CVM",
		"ProtocolType":   "TAT",
		"UserName":       userName,
		"TimeSpan":       48,
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgSerializeRequestParamsFailed).WithDetail(err.Error())
	}
	if err := request.SetActionParameters(string(paramsJSON)); err != nil {
		return "", hcommon.I18nError(i18n.MsgSetRequestParamsFailed).WithDetail(err.Error())
	}

	response := tchttp.NewCommonResponse()
	if err := client.Send(request, response); err != nil {
		slog.Error(
			"[OrcaTerm] GenerateAuthLoginUrl 调用失败",
			"instance_id", instanceId,
			"region", instanceRegion,
			"error", err,
		)
		return "", hcommon.I18nError(i18n.MsgCallGenerateAuthLoginURLFailed).WithDetail(err.Error())
	}

	// 解析响应体提取 LoginUrl
	var result struct {
		Response struct {
			LoginUrl  string `json:"LoginUrl"`
			RequestId string `json:"RequestId"`
			Error     *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error,omitempty"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(response.GetBody(), &result); err != nil {
		return "", hcommon.I18nError(i18n.MsgParseResponseFailed).WithDetail(err.Error())
	}
	if result.Response.Error != nil {
		return "", hcommon.I18nError(i18n.MsgAPIErrorFormat, result.Response.Error.Code, result.Response.Error.Message)
	}
	if result.Response.LoginUrl == "" {
		return "", hcommon.I18nError(i18n.MsgLoginURLEmpty)
	}

	slog.Info(
		"[OrcaTerm] GenerateAuthLoginUrl 成功",
		"instance_id", instanceId,
		"region", instanceRegion,
		"user_name", userName,
		"request_id", result.Response.RequestId,
	)

	return result.Response.LoginUrl, nil
}

type adminInstanceStatusFetcher func(context.Context, string) (*CVMInstanceInfo, error)

// HandleAdminInstanceStatus 管理员查询任意实例的 CVM 运行状态。
func HandleAdminInstanceStatus(w http.ResponseWriter, r *http.Request) {
	handleAdminInstanceStatus(w, r, fetchCVMInstanceInfo)
}

func handleAdminInstanceStatus(w http.ResponseWriter, r *http.Request, fetcher adminInstanceStatusFetcher) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	info, err := fetcher(r.Context(), instance.InstanceId)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCVMInstanceFailed))
		return
	}
	state := "RELEASED"
	cvmInstanceType := instance.CVMInstanceType
	cpu := instance.CVMCPU
	memoryGB := instance.CVMMemoryGB
	systemDiskType := instance.SystemDiskType
	systemDiskSize := instance.SystemDiskSize
	publicIP := instance.CVMPublicIP
	internetChargeType := instance.CVMInternetChargeType
	internetMaxBandwidthOut := instance.CVMInternetMaxBandwidthOut
	if info != nil {
		state = info.State
		cvmInstanceType = info.InstanceType
		cpu = info.CPU
		memoryGB = info.MemoryGB
		systemDiskType = info.SystemDiskType
		systemDiskSize = info.SystemDiskSize
		publicIP = info.PublicIP
		internetChargeType = info.InternetChargeType
		internetMaxBandwidthOut = info.InternetMaxBandwidthOut
	}
	adjustmentStatus := ""
	adjustmentType := ""
	adjustmentErrorCode := ""
	var adjustmentUpdatedAt *string
	var adjustment model.InstanceAdjustment
	adjustmentErr := model.DB(r.Context()).Where("instance_id = ?", instance.ID).Take(&adjustment).Error
	if adjustmentErr != nil && !errors.Is(adjustmentErr, gorm.ErrRecordNotFound) {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(adjustmentErr, i18n.MsgQueryInstanceFailed))
		return
	}
	if adjustmentErr == nil {
		adjustmentStatus = adjustment.Status
		adjustmentType = adjustment.Type
		adjustmentErrorCode = adjustment.ErrorCode
		adjustmentUpdatedAt = formatNullableTime(&adjustment.UpdatedAt)
	}
	response := map[string]interface{}{
		"state":                      state,
		"cvm_instance_type":          cvmInstanceType,
		"cpu":                        cpu,
		"memory_gb":                  memoryGB,
		"system_disk_type":           systemDiskType,
		"system_disk_size":           systemDiskSize,
		"public_ip":                  publicIP,
		"internet_charge_type":       internetChargeType,
		"internet_max_bandwidth_out": internetMaxBandwidthOut,
		"adjustment_status":          adjustmentStatus,
		"adjustment_type":            adjustmentType,
		"adjustment_error_code":      adjustmentErrorCode,
		"adjustment_updated_at":      adjustmentUpdatedAt,
		"adjustment_error_message":   "",
	}
	if adjustmentErrorCode != "" {
		response["adjustment_error_message"] = i18n.T(r.Context(), adjustmentReasonKey(adjustmentErrorCode))
	}
	jsonOK(w, response)
}

// HandleAdminInstanceChannels 管理员查询任意实例的已配置通道列表。
func HandleAdminInstanceChannels(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	resp, err := listInstanceChannels(r.Context(), instance)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryChannelFailed))
		return
	}
	jsonOK(w, resp)
}

// HandleAdminInstanceSkills 管理员查询任意实例的已安装技能列表。
func HandleAdminInstanceSkills(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：从 local_instance_skills 表组装当前已安装的 skill 列表。
	// local_instance_skills 是「成功装着」的事实快照（ack=success 才入表），
	// 因此本接口只列已安装（install_status=success）的 skill。
	if instance.Source == model.InstanceSourceLocal {
		var lis []model.LocalInstanceSkill
		if err := model.DB(r.Context()).
			Where("instance_id = ? AND scope = ?", instance.ID, model.LocalSkillScopeUser).
			Order("slug ASC").
			Find(&lis).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed))
			return
		}
		items := make([]map[string]any, 0, len(lis))
		for _, row := range lis {
			items = append(items, map[string]any{
				"slug":           row.Slug,
				"name":           row.DisplayName,
				"version":        row.Version,
				"install_status": "success",
				"error_message":  "",
				"source":         defaultIfEmpty(row.Source, model.LocalSkillSourceLocal),
				"installed_at":   formatLocalSkillTime(row.InstalledAt),
			})
		}
		jsonOK(w, map[string]any{
			"instance_id": instance.ID,
			"skills":      items,
			"total":       len(items),
		})
		return
	}

	output, err := listInstanceSkills(r.Context(), instance.InstanceId, instance.RuntimeUser, instance.AgentType)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillFailed))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, output)
}

// HandleAdminInstanceRules 管理员查询任意本地实例的已下发规范列表。
// GET /admin/instances/rules?id=<instance_id>
//
// 只返回 source=local 的实例（企业规范库不下发 CVM）。
// 数据来源：直接查 local_instance_rules 快照表（ack=success 才入表）。
func HandleAdminInstanceRules(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if instance.Source != model.InstanceSourceLocal {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleNotLocalInstance))
		return
	}

	var lis []model.LocalInstanceRule
	if err := model.DB(r.Context()).
		Where("instance_id = ? AND scope = ?", instance.ID, model.LocalSkillScopeUser).
		Order("slug ASC").
		Find(&lis).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryRuleFailed))
		return
	}

	items := make([]map[string]any, 0, len(lis))
	for _, row := range lis {
		items = append(items, map[string]any{
			"slug":           row.Slug,
			"name":           row.DisplayName,
			"type":           row.RuleType,
			"version":        row.Version,
			"source":         defaultIfEmpty(row.Source, "enterprise"),
			"distributed_at": formatLocalSkillTime(row.InstalledAt),
		})
	}

	jsonOK(w, map[string]any{
		"instance_id": instance.ID,
		"rules":       items,
		"total":       len(items),
	})
}

// HandleAdminInstanceModels 管理员查询任意实例的所有模型绑定列表（多模型 Fallback v2.0）。
// 该接口与用户侧 `GET /openclaw/instance-models`（HandleInstanceModels）能力对齐：
//   - 返回 `instance_models` 表中归属该实例的全部记录，按 `sort_order DESC` 排序；
//   - 同时区分内置模型（关联 `ai_models`）与自定义模型（解析 `CustomModelConfig` JSON）。
//
// 与用户侧的唯一差异：通过 requireAdmin 鉴权 + getInstanceByIDRaw(userID=0) 不限制所有者；
// 绑定查询与渲染逻辑复用 writeInstanceModelsResponse。
func HandleAdminInstanceModels(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}

	writeInstanceModelsResponse(r.Context(), w, instance)
}

// HandleAdminStartInstance POST /admin/instances/start - 管控端开机
func HandleAdminStartInstance(w http.ResponseWriter, r *http.Request) {
	handleAdminStartInstance(w, r, defaultStatusResolver)
}

func handleAdminStartInstance(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	handlePowerInstanceAction(w, r, powerActionStart, true, 0, resolver)
}

// HandleAdminStopInstance POST /admin/instances/stop - 管控端关机
func HandleAdminStopInstance(w http.ResponseWriter, r *http.Request) {
	handleAdminStopInstance(w, r, defaultStatusResolver)
}

func handleAdminStopInstance(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	handlePowerInstanceAction(w, r, powerActionStop, true, 0, resolver)
}

// HandleAdminRebootInstance POST /admin/instances/reboot - 管控端重启
func HandleAdminRebootInstance(w http.ResponseWriter, r *http.Request) {
	handleAdminRebootInstance(w, r, defaultStatusResolver)
}

func handleAdminRebootInstance(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	instance, err := getAdminInstanceByIDOrInstanceID(r)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	if instance.IsDoctorNode {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}

	// 状态准入：仅 running 允许重启
	if _, err := requireActionAllowedForAdmin(r.Context(), instance, "reboot", resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 乐观锁：写操作标记 + 重置 agent_ready
	if err := setOperationWithAgentReset(model.DB(r.Context()), instance, model.OpReboot); err != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nRichError(err, i18n.MsgOperationConflict))
		return
	}

	client, cvmErr := NewCVMClient(r.Context())
	if cvmErr != nil {
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(cvmErr, i18n.MsgCreateCVMClientFailed))
		return
	}

	req := cvm.NewRebootInstancesRequest()
	req.InstanceIds = common.StringPtrs([]string{instance.InstanceId})
	if _, err := client.RebootInstances(req); err != nil {
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgRebootFailed))
		return
	}
	clearAdjustmentFailure(r.Context(), instance.ID)

	slog.Info("[Admin] 重启", "admin", getAdminUser(r), "instanceId", instance.ID)
	jsonOK(w, map[string]interface{}{"ok": true})
}

const (
	adminRestartGatewayMaxBatch = 50

	adminRestartGatewayConcurrency = 5
)

type adminRestartGatewayResult struct {
	ID         uint   `json:"id,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
	Name       string `json:"name,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

func parseAdminRestartGatewayRequest(r *http.Request) ([]uint, bool, error) {
	var body struct {
		IDs         *[]uint   `json:"ids"`
		ID          *uint     `json:"id"`
		InstanceIDs *[]string `json:"instance_ids"`
		InstanceID  *string   `json:"instance_id"`
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") && r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, false, hcommon.I18nError(i18n.MsgInvalidJSON)
		}
	}

	if body.IDs != nil {
		if len(*body.IDs) == 0 {
			return nil, true, hcommon.I18nError(i18n.MsgIDsEmptyList)
		}
		if len(*body.IDs) > adminRestartGatewayMaxBatch {
			return nil, true, hcommon.I18nError(i18n.MsgIDsCountExceed, adminRestartGatewayMaxBatch)
		}
		ids := hcommon.Unique(hcommon.Filter(*body.IDs, func(id uint) bool { return id != 0 }))
		if len(ids) == 0 {
			return nil, true, hcommon.I18nError(i18n.MsgIDsContainZeroOrDuplicate)
		}
		return ids, true, nil
	}

	if body.InstanceIDs != nil {
		if len(*body.InstanceIDs) == 0 {
			return nil, true, hcommon.I18nError(i18n.MsgInstanceIdsEmptyList)
		}
		if len(*body.InstanceIDs) > adminRestartGatewayMaxBatch {
			return nil, true, hcommon.I18nError(i18n.MsgTooManyInstanceIDs, adminRestartGatewayMaxBatch)
		}
		instanceIDs := hcommon.Unique(trimNonEmptyRestartGatewayInstanceIDs(*body.InstanceIDs))
		if len(instanceIDs) == 0 {
			return nil, true, hcommon.I18nError(i18n.MsgInstanceIdsEmptyList)
		}
		var instances []model.Instance
		if err := model.DB(r.Context()).Select("id").Where("instance_id IN ?", instanceIDs).Find(&instances).Error; err != nil {
			return nil, true, hcommon.I18nRichError(err, i18n.MsgQueryInstancesByIDsFailed)
		}
		if len(instances) == 0 {
			return nil, true, hcommon.I18nError(i18n.MsgInstanceIdsNotFound)
		}
		ids := make([]uint, 0, len(instances))
		for _, inst := range instances {
			ids = append(ids, inst.ID)
		}
		return ids, true, nil
	}

	if body.ID != nil && *body.ID > 0 {
		return []uint{*body.ID}, false, nil
	}
	if raw := strings.TrimSpace(r.FormValue("id")); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || id == 0 {
			return nil, false, hcommon.I18nError(i18n.MsgInvalidID)
		}
		return []uint{uint(id)}, false, nil
	}

	if body.InstanceID != nil && strings.TrimSpace(*body.InstanceID) != "" {
		var inst model.Instance
		if model.DB(r.Context()).Select("id").Where("instance_id = ?", strings.TrimSpace(*body.InstanceID)).First(&inst).Error != nil {
			return nil, false, hcommon.I18nError(i18n.MsgInstanceNotFound)
		}
		return []uint{inst.ID}, false, nil
	}
	if raw := strings.TrimSpace(r.FormValue("instance_id")); raw != "" {
		var inst model.Instance
		if model.DB(r.Context()).Select("id").Where("instance_id = ?", raw).First(&inst).Error != nil {
			return nil, false, hcommon.I18nError(i18n.MsgInstanceNotFound)
		}
		return []uint{inst.ID}, false, nil
	}

	return nil, false, hcommon.I18nError(i18n.MsgMissingIDOrInstanceID)
}

func trimNonEmptyRestartGatewayInstanceIDs(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func adminRestartGatewayOne(ctx context.Context, inst *model.Instance, resolver instanceStatusResolver) (int, error) {
	if inst == nil {
		return http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFound)
	}
	if inst.IsDoctorNode {
		return http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed)
	}
	// 本地 agent 实例：不走 CVM/TAT，restart_gateway 脚本无法下发，直接拒绝。
	if rerr := rejectLocalInstance(inst); rerr != nil {
		return http.StatusBadRequest, rerr
	}
	if inst.InstanceId == "" {
		return http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM)
	}
	if _, err := requireInstanceRunning(ctx, inst, resolver); err != nil {
		if errors.Is(err, ErrAgentNotAllowed) || errors.Is(err, ErrOperationInProgress) {
			return http.StatusConflict, err
		}
		return http.StatusInternalServerError, err
	}
	if _, err := RunAgentScript(ctx, inst, "restart_gateway", 60, nil, nil); err != nil {
		if errors.Is(err, ErrScriptResolveFailed) {
			return http.StatusBadRequest, err
		}
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func handleAdminBatchRestartGateway(w http.ResponseWriter, r *http.Request, ids []uint, resolver instanceStatusResolver) {
	reqCtx := r.Context()
	var instances []model.Instance
	if err := model.DB(reqCtx).Where("id IN ?", ids).Find(&instances).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}

	dbByID := make(map[uint]*model.Instance, len(instances))
	for i := range instances {
		dbByID[instances[i].ID] = &instances[i]
	}

	results := make([]adminRestartGatewayResult, 0, len(ids))
	for _, id := range ids {
		if _, ok := dbByID[id]; !ok {
			results = append(results, adminRestartGatewayResult{
				ID:      id,
				Status:  "failed",
				Message: i18n.T(reqCtx, i18n.MsgInstanceNotFound),
			})
		}
	}

	targets := make([]*model.Instance, 0, len(instances))
	for _, id := range ids {
		if inst, ok := dbByID[id]; ok {
			targets = append(targets, inst)
		}
	}

	resultByIndex := make([]adminRestartGatewayResult, len(targets))
	var wg sync.WaitGroup
	sem := make(chan struct{}, adminRestartGatewayConcurrency)
	for i, inst := range targets {
		i, inst := i, inst
		sem <- struct{}{}
		wg.Add(1)
		go func(execCtx context.Context) {
			defer wg.Done()
			defer func() { <-sem }()
			item := adminRestartGatewayResult{
				ID:         inst.ID,
				InstanceID: inst.InstanceId,
				Name:       inst.Name,
			}
			if _, err := adminRestartGatewayOne(execCtx, inst, resolver); err != nil {
				item.Status = "failed"
				item.Message = hcommon.ErrorMessageWithCtx(reqCtx, err)
			} else {
				item.Status = "ok"
				item.Message = i18n.T(reqCtx, i18n.MsgGatewayRestarted)
			}
			resultByIndex[i] = item
		}(hcommon.DetachContext(reqCtx))
	}
	wg.Wait()
	results = append(results, resultByIndex...)

	slog.Info("[Admin] 批量重启 gateway", "admin", getAdminUser(r), "count", len(targets))
	jsonOK(w, map[string]interface{}{"ok": true, "results": results})
}

// HandleAdminRestartGateway POST /admin/instances/restart-gateway - 管控端重启 gateway 服务，不重启 CVM 实例。
//
// 单实例兼容 form/query：id=<DB ID> 或 instance_id=<CVM ID>。
// 批量请求体 JSON：{"ids":[1,2,3]} 或 {"instance_ids":["ins-a","ins-b"]}。
func HandleAdminRestartGateway(w http.ResponseWriter, r *http.Request) {
	handleAdminRestartGateway(w, r, defaultStatusResolver)
}

func handleAdminRestartGateway(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	ids, isBatch, err := parseAdminRestartGatewayRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if isBatch {
		handleAdminBatchRestartGateway(w, r, ids, resolver)
		return
	}

	id := ids[0]
	var instance model.Instance
	if model.DB(r.Context()).First(&instance, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFound))
		return
	}
	if status, err := adminRestartGatewayOne(r.Context(), &instance, resolver); err != nil {
		writeError(w, r, status, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	slog.Info("[Admin] 重启 gateway", "admin", getAdminUser(r), "instanceId", instance.ID)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminResetInstance POST /admin/instances/reset - 管控端重装
func HandleAdminResetInstance(w http.ResponseWriter, r *http.Request) {
	//  handleAdminResetInstance(w, r, defaultStatusResolver)
	commonHandleResetInstance(w, r, defaultStatusResolver, reinstallAdminOpts)
}

// Deprecated: 重构用户重装和管理员重装方法，使用统一的 commonHandleResetInstance
// FIXME: 2026-05-28 未便于快速回退，保留该函数，稳定后可以移除
func handleAdminResetInstance(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	instance, err := getAdminInstanceByIDOrInstanceID(r)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	if instance.IsDoctorNode {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}

	// final §6 C4：仅支持重装的类型（OpenClaw）允许该操作，否则 403
	if err := checkInstanceSupportsReinstall(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.I18nRichError(err, i18n.MsgReinstallNotSupported))
		return
	}

	// 状态准入：仅 running/stopped 允许重装
	if _, err := requireActionAllowedForAdmin(r.Context(), instance, "reinstall", resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 三期：按实例的 agent_type 精确查询生效镜像，与用户端 HandleResetInstance 对齐。
	// 原来用 GetEnabledImage() 会取任意类型的启用镜像，Hermes/ACE 放开重装后会取错。
	enabledImage, imgErr := model.GetEnabledImageByType(r.Context(), instance.AgentType)
	if imgErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(imgErr, i18n.MsgQueryImageFailed))
		return
	}
	if enabledImage == nil {
		typeName := model.GetAgentTypeDisplayName(r.Context(), instance.AgentType)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgNoImageForType, typeName))
		return
	}
	// 跨 agent_type 防御：堵住 GetEnabledImageByType 回退到空类型镜像导致的类型错乱。
	if err := verifyReinstallImageMatches(r.Context(), instance, enabledImage); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgReinstallImageMismatch))
		return
	}

	// 乐观锁：写操作标记 + 重置 agent_ready
	if err := setOperationWithAgentReset(model.DB(r.Context()), instance, model.OpReinstall); err != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nRichError(err, i18n.MsgOperationConflict))
		return
	}

	client, cvmErr := NewCVMClient(r.Context())
	if cvmErr != nil {
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(cvmErr, i18n.MsgCreateCVMClientFailed))
		return
	}

	req := cvm.NewResetInstanceRequest()
	req.InstanceId = common.StringPtr(instance.InstanceId)
	req.ImageId = common.StringPtr(enabledImage.ImageId)

	if _, err := client.ResetInstance(req); err != nil {
		clearOperation(model.DB(r.Context()), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgReinstallFailedShort))
		return
	}
	clearAdjustmentFailure(r.Context(), instance.ID)

	// 重置业务状态（CLS Agent、AI 模型、版本信息等，重装后需重新拉取）及清空模型绑定，
	// 放在同一事务中，避免部分失败导致 instance_models 与 instances 数据不一致。
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(instance).Updates(map[string]interface{}{
			"cls_agent_status":     0,
			"ai_model_id":          0,
			"agent_version":        "",
			"agent_type":           "",
			"plugin_versions_json": "",
			"version_fetched_at":   nil,
		}).Error; err != nil {
			return err
		}
		return model.HardDeleteInstanceModels(tx, instance.ID)
	}); err != nil {
		slog.Warn("[Admin-ResetInstance] 重置业务状态失败（非阻塞）",
			"instance_pk", instance.ID, "error", err)
	}

	// 重置 plugin 状态，但保留 Pro 绑定信息
	resetMemoryPluginForReinstall(r.Context(), instance.InstanceId)

	// 若实例是 Pro 模式，重装完成后需重新下发 VDB 配置
	resubmitProSwitchAfterReinstall(r.Context(), instance.InstanceId)

	// 重装后恢复 SMH 个人空间环境（前置检查 + 重置 DB 状态 + 异步等待 CVM 就绪后触发安装）
	go syncSMHEnvWhenReadyFn(hcommon.DetachContext(r.Context()), *instance)

	// 异步 approve device（等 TAT + openclaw 就绪后执行 approve_device.sh），
	// 与用户端 HandleResetInstance 对齐：管控端重装是"全新装"语义，重装后实例首次连接
	// 会产生 pending device request，必须自动审批写入 operator token 5 件套权限，
	// 否则 paired.json 中 scopes 为空，所有 RPC 都会鉴权失败。
	// 函数本身有 4 道 guard：agent_type 不支持 / TAT client 创建失败 /
	// TAT Agent 未上线 / openclaw 未就绪 都会安全降级为 warn 日志后退出，
	// 对 Hermes/ACE 等不支持 approve 的类型也会自动跳过。
	go adminApproveDeviceAsyncFn(hcommon.DetachContext(r.Context()), instance.ID, instance.InstanceId, instance.RuntimeUser)

	slog.Info("[Admin] 重装", "admin", getAdminUser(r), "instanceId", instance.ID)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// adminApproveDeviceAsyncFn 是 approveDeviceAsync 的可替换变量钩子，
// 用于单元测试在不创建 TAT 客户端、不真实下发脚本的前提下观察"管控端重装
// 是否触发了 approve 链路"。生产环境直接指向 approveDeviceAsync。
var adminApproveDeviceAsyncFn = approveDeviceAsync

// HandleAdminDetectInstall POST /admin/instances/detect-install - 管控端下发 TAT 检测 openclaw 安装目录及用户
//
// 支持四种模式：
//   - 单实例：id=<实例数据库ID> 或 instance_id=<CVM实例ID>
//   - 批量：请求体 JSON {"ids": [...]} 或 {"instance_ids": [...]}
//
// 返回每个实例的检测结果（JSON）。
func HandleAdminDetectInstall(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// 支持四种模式：单实例 (id=xxx / instance_id=xxx) 或批量 (JSON body {"ids": [...]} / {"instance_ids": [...]})
	var instances []model.Instance

	if idStr := r.FormValue("id"); idStr != "" {
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidID))
			return
		}
		var inst model.Instance
		// 与批量模式 (ids / instance_ids) 行为保持一致：查不到不报错，
		// 落入空 instances，最终返回 200 + {"results":[]}。
		if err := model.DB(r.Context()).First(&inst, id).Error; err == nil {
			instances = []model.Instance{inst}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryInstanceFailed))
			return
		}
	} else if instanceIDStr := r.FormValue("instance_id"); instanceIDStr != "" {
		var inst model.Instance
		// 与批量模式 (ids / instance_ids) 行为保持一致：查不到不报错，
		// 落入空 instances，最终返回 200 + {"results":[]}。
		if err := model.DB(r.Context()).Where("instance_id = ?", instanceIDStr).First(&inst).Error; err == nil {
			instances = []model.Instance{inst}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryInstanceFailed))
			return
		}
	} else {
		var body struct {
			IDs         []uint64 `json:"ids"`
			InstanceIDs []string `json:"instance_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
			return
		}
		if len(body.IDs) > 0 {
			if len(body.IDs) > 50 {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMaxDetectInstances50))
				return
			}
			if err := model.DB(r.Context()).Where("id IN ?", body.IDs).Find(&instances).Error; err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryInstanceFailed))
				return
			}
		} else if len(body.InstanceIDs) > 0 {
			if len(body.InstanceIDs) > 50 {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMaxDetectInstances50))
				return
			}
			if err := model.DB(r.Context()).Where("instance_id IN ?", body.InstanceIDs).Find(&instances).Error; err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryInstanceFailed))
				return
			}
		} else {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingIDInstanceIDIDsParam))
			return
		}
	}

	if len(instances) > 50 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMaxDetectInstances50))
		return
	}

	var doctorIDs []uint
	var localIDs []uint
	for _, inst := range instances {
		if inst.IsDoctorNode {
			doctorIDs = append(doctorIDs, inst.ID)
		}
		if inst.Source == model.InstanceSourceLocal {
			localIDs = append(localIDs, inst.ID)
		}
	}
	if len(doctorIDs) > 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowedRemoveIDs, doctorIDs))
		return
	}
	if len(localIDs) > 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalInstanceUnsupportedOp))
		return
	}

	type detectResult struct {
		ID         uint        `json:"id"`
		InstanceId string      `json:"instance_id"`
		Status     string      `json:"status"`
		Output     interface{} `json:"output"`
		Error      string      `json:"error,omitempty"`
	}

	results := make([]detectResult, len(instances))
	var wg sync.WaitGroup

	for i, inst := range instances {
		wg.Add(1)
		go func(ctx context.Context, idx int, instance model.Instance) {
			defer wg.Done()

			res := detectResult{
				ID:         instance.ID,
				InstanceId: instance.InstanceId,
			}

			if instance.InstanceId == "" {
				res.Status = "skip"
				res.Error = i18n.T(ctx, i18n.MsgInstanceNoAssociatedCVM)
				results[idx] = res
				return
			}

			// final：按 agent_type 分派探测脚本。
			// - openclaw → detect_openclaw_install.sh
			// - hermes   → detect_hermes_install.sh（探测 hermes / harness CLI）
			// - ace      → detect_ace_install.sh（探测 lightclaw CLI）
			// 三端脚本顶层字段名一致（runtime_user/runtime_home/current_user/current_home/
			// service_status/systemd_units/xdg_runtime_dir 等），前端可复用同一解析逻辑。
			scriptName, resolveErr := ResolveScript(ctx, "detect_install", instance.AgentType)
			if resolveErr != nil {
				res.Status = "error"
				res.Error = hcommon.ErrorMessageWithCtx(r.Context(), resolveErr)
				results[idx] = res
				return
			}

			output, err := RunScript(ctx, instance.InstanceId, scriptName, 30, instance.RuntimeUser, nil, nil)
			if err != nil {
				res.Status = "error"
				res.Error = hcommon.ErrorMessageWithCtx(r.Context(), err)
				results[idx] = res
				return
			}

			// 尝试解析为 JSON
			var parsed interface{}
			if json.Unmarshal([]byte(output), &parsed) == nil {
				res.Output = parsed
			} else {
				res.Output = output
			}
			res.Status = "ok"
			results[idx] = res
		}(hcommon.DetachContext(r.Context()), i, inst)
	}

	wg.Wait()

	slog.Info("[Admin] 检测 OpenClaw 安装信息", "admin", getAdminUser(r), "count", len(instances))
	jsonOK(w, map[string]interface{}{"results": results})
}

// getAdminUser 从请求中获取管理员用户名
func getAdminUser(r *http.Request) string {
	session := getSession(r)
	if username, ok := session.Values["username"].(string); ok {
		return username
	}
	return "unknown"
}

// getAdminInstanceByIDOrInstanceID 从请求中解析 id 或 instance_id，查询实例（不限所有者）。
// id 优先于 instance_id；两者均为空时返回错误。
func getAdminInstanceByIDOrInstanceID(r *http.Request) (*model.Instance, error) {
	id, instanceID, err := extractInstanceIDOrCVMID(r)
	if err != nil {
		return nil, err
	}
	return findInstanceByIDOrCVMID(r.Context(), 0, id, instanceID)
}

// HandleAdminRefreshInstanceVersion 手动触发单个实例的版本信息刷新。
// POST /admin/instances/refresh-version?id=<db_id> 或 ?instance_id=<cvm_id>
// 同步等待 TAT 脚本执行完成（最长 60s），返回最新版本信息。
func HandleAdminRefreshInstanceVersion(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	instance, err := getAdminInstanceByIDOrInstanceID(r)
	if err != nil {
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	if instance.IsDoctorNode {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	if instance.AgentReady != 1 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceAgentNotReadyForVersion))
		return
	}

	// 同步执行，等待结果（TAT 脚本超时 30s，整体最长约60s）
	if err := FetchAndSaveVersionInfoSync(r.Context(), *instance); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPullVersionInfoFailed))
		return
	}

	// 重新读取最新数据返回给前端
	var updated model.Instance
	if model.DB(r.Context()).First(&updated, instance.ID).Error != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgReadUpdatedInstanceFailed))
		return
	}

	var vfAt *string
	if updated.VersionFetchedAt != nil {
		s := updated.VersionFetchedAt.UTC().Format("2006-01-02T15:04:05Z")
		vfAt = &s
	}

	jsonOK(w, map[string]interface{}{
		"ok":                    true,
		"agent_version":         updated.AgentVersion,
		"agent_type":            model.NormalizeAgentType(updated.AgentType),
		"plugin_version_status": model.BuildPluginVersionStatus(updated.AgentVersion, updated.PluginVersionsJSON),
		"version_fetched_at":    vfAt,
	})
}

// HandleAdminBatchUpgrade 管理员批量升级实例镜像。
// POST /admin/instances/batch-upgrade
//
// 请求体 (JSON):
//
//	{"ids": [1, 2, 3]}
//
// upgradeResult 批量升级中每个实例的结果
type upgradeResult struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`  // "started" / "skipped" / "failed"
	Message string `json:"message"` // 详细信息
}

// matchEnabledImage 根据实例的 agent_type 从启用镜像映射中查找对应镜像。
// 空 agent_type 视为 openclaw（兼容存量数据）。未找到返回 nil。
func matchEnabledImage(inst *model.Instance, enabledImagesMap map[string]*model.AIImage) *model.AIImage {
	agentType := model.NormalizeAgentType(inst.AgentType)
	img, ok := enabledImagesMap[agentType]
	if !ok {
		return nil
	}
	return img
}

// prepareBatchUpgradeResults 为每个实例匹配启用镜像，返回 (实例→镜像映射, 无镜像的失败结果列表)。
func prepareBatchUpgradeResults(ctx context.Context, instances []model.Instance, enabledImagesMap map[string]*model.AIImage) (
	imageForInstance map[uint]*model.AIImage, failedResults []upgradeResult,
) {
	imageForInstance = make(map[uint]*model.AIImage, len(instances))
	for i := range instances {
		inst := &instances[i]
		img := matchEnabledImage(inst, enabledImagesMap)
		if img == nil {
			typeName := model.GetAgentTypeDisplayName(ctx, inst.AgentType)
			failedResults = append(failedResults, upgradeResult{
				ID:      inst.ID,
				Name:    inst.Name,
				Status:  "failed",
				Message: i18n.T(ctx, i18n.MsgNoEnabledImageForType, typeName),
			})
			continue
		}
		imageForInstance[inst.ID] = img
	}
	return
}

// resetInstanceVersionInfo 重装前清空实例的版本相关信息（不清空 agent_type）。
func resetInstanceVersionInfo(ctx context.Context, instance *model.Instance) error {
	if err := model.DB(ctx).Model(instance).Updates(map[string]interface{}{
		"cls_agent_status":     0,
		"ai_model_id":          0,
		"agent_version":        "",
		"plugin_versions_json": "",
		"version_fetched_at":   nil,
	}).Error; err != nil {
		slog.Error("[ResetVersionInfo] 清空版本信息失败", "instance_id", instance.ID, "error", err)
		return err
	}
	return nil
}

// 限制：最多 20 个实例。升级前会检查所有实例是否使用官方公共镜像，
// 如果存在非官方镜像的实例，整个请求将被拒绝。
func HandleAdminBatchUpgrade(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		IDs         []uint   `json:"ids"`
		InstanceIDs []string `json:"instance_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("[BatchUpgrade] 请求体解析失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}

	// 查询所有实例
	var instances []model.Instance
	if len(req.IDs) > 0 {
		if len(req.IDs) > 20 {
			log.Warn("[BatchUpgrade] 超出单次批量升级上限", "count", len(req.IDs))
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBatchUpgradeMax20))
			return
		}
		if err := model.DB(r.Context()).Where("id IN ?", req.IDs).Find(&instances).Error; err != nil {
			log.Error("[BatchUpgrade] 查询实例失败", "ids", req.IDs, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
			return
		}
	} else if len(req.InstanceIDs) > 0 {
		if len(req.InstanceIDs) > 20 {
			log.Warn("[BatchUpgrade] 超出单次批量升级上限", "count", len(req.InstanceIDs))
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBatchUpgradeMax20))
			return
		}
		if err := model.DB(r.Context()).Where("instance_id IN ?", req.InstanceIDs).Find(&instances).Error; err != nil {
			log.Error("[BatchUpgrade] 查询实例失败", "instance_ids", req.InstanceIDs, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
			return
		}
	} else {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingIdsOrInstanceIdsParam))
		return
	}
	var doctorIDs []uint
	var localIDs []uint
	for _, inst := range instances {
		if inst.IsDoctorNode {
			doctorIDs = append(doctorIDs, inst.ID)
		}
		if inst.Source == model.InstanceSourceLocal {
			localIDs = append(localIDs, inst.ID)
		}
	}
	if len(doctorIDs) > 0 {
		log.Warn("[BatchUpgrade] 拒绝龙虾医生节点", "doctor_ids", doctorIDs)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowedRemoveIDs, doctorIDs))
		return
	}
	if len(localIDs) > 0 {
		log.Warn("[BatchUpgrade] 拒绝本地 agent 实例", "local_ids", localIDs)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalInstanceUnsupportedOp))
		return
	}
	if len(instances) == 0 {
		log.Warn("[BatchUpgrade] 未找到任何有效实例", "ids", req.IDs)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNoValidInstanceFound))
		return
	}
	log.Info("[BatchUpgrade] 查询到实例", "requested", len(req.IDs), "found", len(instances))

	// 预加载所有类型的启用镜像，避免循环内重复查询
	enabledImagesMap, err := model.GetEnabledImagesMap(r.Context())
	if err != nil {
		log.Error("[BatchUpgrade] 查询启用镜像失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryEnabledImageFailed))
		return
	}
	if len(enabledImagesMap) == 0 {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgNoDefaultImage))
		return
	}

	// 收集有 CVM 实例 ID 的实例，用于批量查询镜像信息
	var cvmIds []string
	for i := range instances {
		if instances[i].InstanceId == "" {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVMCannotUpgrade, instances[i].ID))
			return
		}
		cvmIds = append(cvmIds, instances[i].InstanceId)
	}

	// 批量查询 CVM 信息（包含 ImageId），检查是否为官方镜像
	log.Info("[BatchUpgrade] 开始批量查询 CVM 信息", "cvm_count", len(cvmIds))
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), cvmIds)
	log.Info("[BatchUpgrade] CVM 信息查询完成", "queried", len(cvmIds), "returned", len(cvmInfoMap))
	var nonOfficialInstances []string
	for _, inst := range instances {
		info, ok := cvmInfoMap[inst.InstanceId]
		if !ok || info == nil {
			nonOfficialInstances = append(nonOfficialInstances, i18n.T(r.Context(), i18n.MsgInstanceCannotGetCVMInfoFmt, inst.Name, inst.ID))
			continue
		}
	}
	if len(nonOfficialInstances) > 0 {
		log.Warn("[BatchUpgrade] 存在无法获取的实例信息，拒绝升级", "non_official", nonOfficialInstances)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNonOfficialInstancesCannotBatch, strings.Join(nonOfficialInstances, "、")))
		return
	}

	// 预匹配每个实例的启用镜像
	imageForInstance, noImageResults := prepareBatchUpgradeResults(ctx, instances, enabledImagesMap)

	// 逐个检查并启动升级（复用已有的 cvmInfoMap，避免重复调用 CVM API）
	results := make([]upgradeResult, 0, len(instances))
	results = append(results, noImageResults...)

	// 🆕 批量预查实例状态依赖数据，消除 N+1
	batchUpgradeSiteConfig := model.GetSiteConfig(r.Context())
	batchUpgradeInstIDs := make([]uint, 0, len(instances))
	batchUpgradeLocalIDs := make([]uint, 0)
	for i := range instances {
		batchUpgradeInstIDs = append(batchUpgradeInstIDs, instances[i].ID)
		if instances[i].Source == model.InstanceSourceLocal {
			batchUpgradeLocalIDs = append(batchUpgradeLocalIDs, instances[i].ID)
		}
	}
	batchUpgradeInstallingMap := batchHasInstallingSkillInstallations(r.Context(), batchUpgradeInstIDs)
	batchUpgradeLocalInfoMap := batchResolveLocalInstanceStatus(r.Context(), batchUpgradeLocalIDs)
	batchUpgradeLookup := &InstanceStatusBatchLookup{SiteConfig: batchUpgradeSiteConfig, InstallingSkillMap: batchUpgradeInstallingMap, LocalInfoMap: batchUpgradeLocalInfoMap}

	log.Info("[BatchUpgrade] 开始批量升级", "admin", getAdminUser(r), "ids", req.IDs)
	for i := range instances {
		inst := &instances[i]

		// 跳过无镜像的实例（已在 noImageResults 中）
		defaultImage, ok := imageForInstance[inst.ID]
		if !ok {
			continue
		}

		// 状态准入：批量升级逐项返回；非 running 给统一文案的 failed 项，不短路。
		// 使用 /openclaw/status 一致口径（ResolveInstanceStatus + batch 预查数据）。
		statusResp := ResolveInstanceStatus(r.Context(), inst, cvmInfoMap[inst.InstanceId], batchUpgradeLookup)
		if statusResp.Status != model.StatusRunning {
			key, args := agentStatusRejectMessage(statusResp)
			results = append(results, upgradeResult{
				ID:      inst.ID,
				Name:    inst.Name,
				Status:  "failed",
				Message: i18n.T(r.Context(), key, args...),
			})
			continue
		}

		// 实例级前置入口（与单实例升级共用同一份逻辑，新增检查请改 prepareInstanceForUpgrade）：
		//   - 拒绝官方镜像降级（OpenClaw 实例当前版本 > 官方镜像版本时） -> failed
		//   - openclaw.json 配置 providers key 合法性                  -> failed
		//   - 防重入（current_operation processing）                    -> skipped
		//   - 该类型是否支持一键升级                                      -> failed
		//   - 官方镜像 runtime_user / runtime_home 强制校正              -> failed（DB 写失败时）
		if outcome := prepareInstanceForUpgrade(ctx, inst, defaultImage, "[BatchUpgrade]"); !outcome.OK {
			log.Warn(
				"[BatchUpgrade] 实例级前置检查未通过，跳过升级",
				"instance_id", inst.InstanceId, "db_id", inst.ID,
				"agent_type", inst.AgentType,
				"current_operation", inst.CurrentOperation,
				"current_version", inst.AgentVersion,
				"target_image", defaultImage.ImageId,
				"target_version", defaultImage.AgentVersion,
				"batch_status", outcome.BatchStatus,
				"error", outcome.Err,
			)
			results = append(results, upgradeResult{
				ID:      inst.ID,
				Name:    inst.Name,
				Status:  outcome.BatchStatus,
				Message: outcome.Err.Error(),
			})
			continue
		}

		// 升级启动入口（与单实例升级共用同一份逻辑，新增启动期检查请改 startUpgradeForInstance）：
		//   - checkNeedsUpgrade 判定是否真的需要升级（复用已查的 cvmInfoMap）
		//   - setOperation 设置 OpUpgrade 操作锁
		//   - 启动异步 performUpgrade goroutine
		switch outcome := startUpgradeForInstance(ctx, inst, defaultImage, cvmInfoMap, "[BatchUpgrade]"); {
		case outcome.Started:
			results = append(results, upgradeResult{
				ID:      inst.ID,
				Name:    inst.Name,
				Status:  "started",
				Message: "升级已开始",
			})
		case outcome.AlreadyLatest:
			results = append(results, upgradeResult{
				ID:      inst.ID,
				Name:    inst.Name,
				Status:  "skipped",
				Message: i18n.T(r.Context(), i18n.MsgUpgradeAlreadyLatest),
			})
		default:
			results = append(results, upgradeResult{
				ID:      inst.ID,
				Name:    inst.Name,
				Status:  outcome.BatchStatus,
				Message: outcome.Err.Error(),
			})
		}
	}

	log.Info("[BatchUpgrade] 批量升级任务分发完成", "admin", getAdminUser(r), "total", len(instances), "results_count", len(results))
	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"results": results,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// /admin/instances/by-user-group —— 按 (user_id, group_id) 对 + group_ids 子树
// 批量查询实例列表。不分页、一次返回全量。
//
// 请求体（JSON）：
//
//	{
//	  "user_group_ids": [{"user_id": 1, "group_id": 1}, ...],  // 可选，用户和分组 id 对
//	  "group_ids":      [3, 7]                                 // 可选，子树+其中直属成员
//	}
//
// 语义：
//  1. user_group_ids 直接展开为一组 (user_id, group_id) 精确对。
//  2. group_ids 里每个 root 展开子树（含自身），对子树里每个 group 找直属成员，
//     拼成 (member.user_id, group.id) 对。
//  3. 两者并集后（去重），再 WHERE (user_id, group_id) IN ... 查 instances。
//
// 响应：
//
//	{
//	  "ok": true,
//	  "instances": [{id, instance_id, name, group_id, status, created_at}]
//	}
// ─────────────────────────────────────────────────────────────────────────────

// userGroupPair 查询条件 1 的子项，以及内部展开 group_ids 后的通用 (user, group) 对。
type userGroupPair struct {
	UserID  uint `json:"user_id"`
	GroupID uint `json:"group_id"`
}

// instancesByUserGroupRequest 请求体结构。
type instancesByUserGroupRequest struct {
	UserGroupIDs []userGroupPair `json:"user_group_ids"`
	GroupIDs     []uint          `json:"group_ids"`
}

// instanceByUserGroupItem 响应中每条实例的字段子集。
type instanceByUserGroupItem struct {
	ID            uint   `json:"id"`
	InstanceID    string `json:"instance_id"`
	Name          string `json:"name"`
	UserID        uint   `json:"user_id"`
	UserName      string `json:"user_name"` // 🆕 v6.13：所属用户的用户名；用户已被删或不存在时为空串
	GroupID       uint   `json:"group_id"`
	GroupFullPath string `json:"group_full_path"` // 🆕 v6.13：所属分组全路径；group_id=0 时为空串
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

// adminInstancesByUserGroupMaxPairs 安全上限：展开后 (user_id, group_id) 对的最大数。
// 避免无界输入造成巨大 IN() 查询；按经验单管理员请求 2000 对足够。
const adminInstancesByUserGroupMaxPairs = 2000

// HandleAdminInstancesByUserGroup POST /admin/instances/by-user-group
// 详见文件顶部 handler 段注释。
func HandleAdminInstancesByUserGroup(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req instancesByUserGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if len(req.UserGroupIDs) == 0 && len(req.GroupIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDsOrGroupIDsRequired))
		return
	}

	// 1) 条件 1：直接拿 (user_id, group_id) 对。
	//    user_id=0 仍过滤；group_id=0 允许保留，用于查询某 user 的"未分组实例"
	//    （场景 C：从未分组加入分组时需要拿 group_id=0 的实例）。
	pairSet := make(map[userGroupPair]struct{}, len(req.UserGroupIDs))
	for _, p := range req.UserGroupIDs {
		if p.UserID == 0 {
			continue
		}
		pairSet[p] = struct{}{}
	}

	// 2) 条件 2：group_ids 子树展开 → 直接按 group_id IN subtree 查实例。
	//    不走 user_group_members pair 匹配 —— 子树场景（父级变更）需要返回子树下
	//    所有实例，包括用户不是该子分组直属成员但实例 group_id 仍在子树内的情况。
	var subtreeInsts []model.Instance
	if len(req.GroupIDs) > 0 {
		// 去重 + 过滤 0
		rootSet := make(map[uint]struct{}, len(req.GroupIDs))
		rootIDs := make([]uint, 0, len(req.GroupIDs))
		for _, gid := range req.GroupIDs {
			if gid == 0 {
				continue
			}
			if _, ok := rootSet[gid]; ok {
				continue
			}
			rootSet[gid] = struct{}{}
			rootIDs = append(rootIDs, gid)
		}

		if len(rootIDs) > 0 {
			// 子树：closure 表 ancestor_id ∈ rootIDs 时拿所有 descendant_id（含自身）
			var descIDs []uint
			if err := model.DB(r.Context()).Model(&model.GroupClosure{}).
				Distinct("descendant_id").
				Where("ancestor_id IN ?", rootIDs).
				Pluck("descendant_id", &descIDs).Error; err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryGroupSubtreeFailed))
				return
			}
			if len(descIDs) > 0 {
				// 直接按 group_id IN subtree 查实例，不依赖 user_group_members
				if err := model.DB(r.Context()).
					Where("group_id IN ?", descIDs).
					Find(&subtreeInsts).Error; err != nil {
					writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
					return
				}
				// 把 subtree 实例的 (user_id, group_id) 也加入 pairSet，
				// 让后续 pair 过滤能命中（subtree 实例不走 pair 过滤，但统一去重）
				for i := range subtreeInsts {
					pairSet[userGroupPair{UserID: subtreeInsts[i].UserID, GroupID: subtreeInsts[i].GroupID}] = struct{}{}
				}
			}
		}
	}

	// 3) pair 并集 + subtree 实例都为空 → 返回空列表
	if len(pairSet) == 0 && len(subtreeInsts) == 0 {
		jsonOK(w, map[string]interface{}{
			"ok":        true,
			"instances": []instanceByUserGroupItem{},
		})
		return
	}
	if len(pairSet) > adminInstancesByUserGroupMaxPairs {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupPairsExceedLimit,
			adminInstancesByUserGroupMaxPairs, len(pairSet)))
		return
	}

	// 4) pair-based 查询：user_group_ids 的精确 (user_id, group_id) 对匹配。
	//    ⚠️ 必须用 Find(&[]model.Instance{})，GORM 会自动注入 deleted_at IS NULL 排除已销毁的实例；
	//       不能改成 Table("instances")，否则会绕开软删过滤。
	var candidates []model.Instance
	if len(pairSet) > 0 {
		userIDSet := make(map[uint]struct{}, len(pairSet))
		groupIDSet := make(map[uint]struct{}, len(pairSet))
		for p := range pairSet {
			userIDSet[p.UserID] = struct{}{}
			groupIDSet[p.GroupID] = struct{}{}
		}
		userIDs := make([]uint, 0, len(userIDSet))
		for uid := range userIDSet {
			userIDs = append(userIDs, uid)
		}
		groupIDs := make([]uint, 0, len(groupIDSet))
		for gid := range groupIDSet {
			groupIDs = append(groupIDs, gid)
		}
		if err := model.DB(r.Context()).
			Where("user_id IN ? AND group_id IN ?", userIDs, groupIDs).
			Find(&candidates).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
			return
		}
	}

	// 5) 合并 + 去重：pair 精确过滤的实例 + subtree 直接查的实例
	seenIDs := make(map[uint]struct{}, len(candidates)+len(subtreeInsts))
	matched := make([]model.Instance, 0, len(candidates)+len(subtreeInsts))
	for i := range candidates {
		inst := &candidates[i]
		if _, ok := pairSet[userGroupPair{UserID: inst.UserID, GroupID: inst.GroupID}]; ok {
			if _, dup := seenIDs[inst.ID]; !dup {
				seenIDs[inst.ID] = struct{}{}
				matched = append(matched, *inst)
			}
		}
	}
	for i := range subtreeInsts {
		inst := &subtreeInsts[i]
		if _, dup := seenIDs[inst.ID]; !dup {
			seenIDs[inst.ID] = struct{}{}
			matched = append(matched, *inst)
		}
	}

	// 6) 查 CVM 实时状态（最多 100 个/批，内部已处理）
	cvmIDs := make([]string, 0, len(matched))
	for i := range matched {
		if matched[i].InstanceId != "" {
			cvmIDs = append(cvmIDs, matched[i].InstanceId)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), cvmIDs)

	// 🆕 批量查 group_id → full_path，避免 N+1。复用 admin_instances 里已有的 helper。
	groupIDsForPath := make([]uint, 0, len(matched))
	for i := range matched {
		if matched[i].GroupID != 0 {
			groupIDsForPath = append(groupIDsForPath, matched[i].GroupID)
		}
	}
	groupFullPathMap := fetchGroupFullPathMap(r.Context(), groupIDsForPath)

	// 🆕 批量查 user_id → username，避免 N+1。用 Unscoped 兼容被软删的用户（历史 agent 仍保留）。
	userNameMap := map[uint]string{}
	{
		userIDSet := make(map[uint]struct{}, len(matched))
		for i := range matched {
			if matched[i].UserID != 0 {
				userIDSet[matched[i].UserID] = struct{}{}
			}
		}
		if len(userIDSet) > 0 {
			ids := make([]uint, 0, len(userIDSet))
			for uid := range userIDSet {
				ids = append(ids, uid)
			}
			type userRow struct {
				ID       uint
				Username string
			}
			var rows []userRow
			if err := model.DB(r.Context()).Unscoped().Model(&model.User{}).
				Select("id, username").
				Where("id IN ?", ids).
				Scan(&rows).Error; err != nil {
				slog.Warn("[InstancesByUserGroup] 查询 username 失败，降级为空", "err", err)
			}
			for _, r := range rows {
				userNameMap[r.ID] = r.Username
			}
		}
	}

	// 🆕 批量预查实例状态依赖数据，消除 N+1
	siteConfig := model.GetSiteConfig(r.Context())
	preInstIDs := make([]uint, 0, len(matched))
	localInstIDs := make([]uint, 0)
	for i := range matched {
		preInstIDs = append(preInstIDs, matched[i].ID)
		if matched[i].Source == model.InstanceSourceLocal {
			localInstIDs = append(localInstIDs, matched[i].ID)
		}
	}
	installingSkillMap := batchHasInstallingSkillInstallations(r.Context(), preInstIDs)
	localInfoMap := batchResolveLocalInstanceStatus(r.Context(), localInstIDs)
	batch := &InstanceStatusBatchLookup{SiteConfig: siteConfig, InstallingSkillMap: installingSkillMap, LocalInfoMap: localInfoMap}

	// 7) 组装响应
	items := make([]instanceByUserGroupItem, 0, len(matched))
	for i := range matched {
		inst := &matched[i]
		status := ResolveInstanceStatus(r.Context(), inst, cvmInfoMap[inst.InstanceId], batch).Status
		items = append(items, instanceByUserGroupItem{
			ID:            inst.ID,
			InstanceID:    inst.InstanceId,
			Name:          inst.Name,
			UserID:        inst.UserID,
			UserName:      userNameMap[inst.UserID],
			GroupID:       inst.GroupID,
			GroupFullPath: groupFullPathMap[inst.GroupID],
			Status:        status,
			CreatedAt:     inst.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	jsonOK(w, map[string]interface{}{
		"ok":        true,
		"instances": items,
	})
}

// ==========================================================================
// 管控端模型管理接口
// ==========================================================================

// HandleAdminAvailableModels 返回实例可配置的模型列表（已启用 + 对该实例可见）。
// GET /admin/instances/available-models?id={instance_id}
func HandleAdminAvailableModels(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}

	// 能力 guard
	if err := checkInstanceSupportsModel(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.I18nRichError(err, i18n.MsgModelNotSupported))
		return
	}

	var models []model.AIModel
	// 管控端可选择模型列表：也是主动绑定场景，必须同时 enabled + visible
	model.DB(r.Context()).Where("enabled = ? AND visible = ?", true, true).Order("created_at DESC").Find(&models)

	// 可见性过滤：按实例绑定的分组（与用户端 HandleModelsList 一致）
	models = usergroup.FilterModelsByVisibility(r.Context(), models, instance.GroupID)

	config := model.GetSiteConfig(r.Context())
	type modelItem struct {
		ID                uint              `json:"id"`
		Provider          string            `json:"provider"`
		ModelID           string            `json:"model_id"`
		ModelType         string            `json:"model_type"`
		InputTypes        []string          `json:"input_types"`
		ContextLen        int               `json:"context_len"`
		MaxTokens         int               `json:"max_tokens"`
		CustomHTTPHeaders map[string]string `json:"custom_http_headers"`
		ModelName         string            `json:"model_name"`
		Default           bool              `json:"default"`
	}

	items := make([]modelItem, 0, len(models))
	for _, m := range models {
		items = append(items, modelItem{
			ID:                m.ID,
			Provider:          m.Provider,
			ModelID:           m.ModelID,
			ModelType:         m.ModelType,
			InputTypes:        m.GetInputTypes(),
			ContextLen:        m.ContextLen,
			MaxTokens:         m.MaxTokens,
			CustomHTTPHeaders: m.GetCustomHTTPHeaders(),
			ModelName:         m.ModelName,
			Default:           config.DefaultModelID == m.ID,
		})
	}

	jsonOK(w, map[string]interface{}{
		"ok":     true,
		"models": items,
	})
}

const (
	adminBatchSetModelMaxBatch = 20

	adminBatchSetModelConcurrency = 5
)

type adminBatchSetModelModelPayload struct {
	AIModelID         *uint             `json:"ai_model_id"`
	Provider          string            `json:"provider"`
	ModelID           string            `json:"model_id"`
	ModelName         string            `json:"model_name"`
	APIKey            string            `json:"api_key"`
	URL               string            `json:"url"`
	ModelType         string            `json:"model_type"`
	InputTypes        []string          `json:"input_types"`
	ContextLen        int               `json:"context_len"`
	MaxTokens         int               `json:"max_tokens"`
	CustomHTTPHeaders map[string]string `json:"custom_http_headers"`
}

type adminBatchSetModelRequest struct {
	IDs               []uint                           `json:"ids"`
	InstanceIDs       []string                         `json:"instance_ids"`
	AIModelID         *uint                            `json:"ai_model_id"`
	Provider          string                           `json:"provider"`
	ModelID           string                           `json:"model_id"`
	ModelName         string                           `json:"model_name"`
	APIKey            string                           `json:"api_key"`
	URL               string                           `json:"url"`
	ModelType         string                           `json:"model_type"`
	InputTypes        []string                         `json:"input_types"`
	ContextLen        int                              `json:"context_len"`
	MaxTokens         int                              `json:"max_tokens"`
	CustomHTTPHeaders map[string]string                `json:"custom_http_headers"`
	Fallbacks         []adminBatchSetModelModelPayload `json:"fallbacks"`
}

type adminBatchSetModelResult struct {
	ID         uint   `json:"id,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
	Name       string `json:"name,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

func parseAdminBatchPrimarySetModelInput(req adminBatchSetModelRequest) (setModelInput, *hcommon.RichError) {
	if req.AIModelID == nil {
		return setModelInput{}, hcommon.I18nError(i18n.MsgBadRequest)
	}
	return setModelInput{
		AIModelID:         *req.AIModelID,
		Provider:          req.Provider,
		ModelID:           req.ModelID,
		ModelName:         req.ModelName,
		APIKey:            req.APIKey,
		URL:               req.URL,
		ModelType:         req.ModelType,
		InputTypes:        req.InputTypes,
		ContextLen:        req.ContextLen,
		MaxTokens:         req.MaxTokens,
		CustomHTTPHeaders: req.CustomHTTPHeaders,
	}, nil
}

func parseAdminBatchFallbackSetModelInput(p adminBatchSetModelModelPayload) (setModelInput, *hcommon.RichError) {
	if p.AIModelID == nil {
		return setModelInput{}, hcommon.I18nError(i18n.MsgBadRequest)
	}
	return setModelInput{
		AIModelID:         *p.AIModelID,
		Provider:          p.Provider,
		ModelID:           p.ModelID,
		ModelName:         p.ModelName,
		APIKey:            p.APIKey,
		URL:               p.URL,
		ModelType:         p.ModelType,
		InputTypes:        p.InputTypes,
		ContextLen:        p.ContextLen,
		MaxTokens:         p.MaxTokens,
		CustomHTTPHeaders: p.CustomHTTPHeaders,
	}, nil
}

func setModelInputBindingKey(in setModelInput) (string, bool) {
	if in.AIModelID > 0 {
		return fmt.Sprintf("builtin:%d", in.AIModelID), true
	}
	modelID := strings.TrimSpace(in.ModelID)
	if modelID == "" {
		return "", false
	}
	return fmt.Sprintf("custom:%s", modelID), true
}

func parseAdminBatchSetModelSelectors(req adminBatchSetModelRequest) ([]uint, []string, bool, *hcommon.RichError) {
	if len(req.IDs) > 0 {
		if len(req.IDs) > adminBatchSetModelMaxBatch {
			return nil, nil, true, hcommon.I18nError(i18n.MsgIDsCountExceed, adminBatchSetModelMaxBatch)
		}
		ids := hcommon.Unique(hcommon.Filter(req.IDs, func(id uint) bool { return id != 0 }))
		if len(ids) == 0 {
			return nil, nil, true, hcommon.I18nError(i18n.MsgIDsContainZeroOrDuplicate)
		}
		return ids, nil, true, nil
	}

	if len(req.InstanceIDs) > 0 {
		if len(req.InstanceIDs) > adminBatchSetModelMaxBatch {
			return nil, nil, false, hcommon.I18nError(i18n.MsgTooManyInstanceIDs, adminBatchSetModelMaxBatch)
		}
		instanceIDs := hcommon.Unique(trimNonEmptyRestartGatewayInstanceIDs(req.InstanceIDs))
		if len(instanceIDs) == 0 {
			return nil, nil, false, hcommon.I18nError(i18n.MsgInstanceIdsEmptyList)
		}
		return nil, instanceIDs, false, nil
	}

	return nil, nil, false, hcommon.I18nError(i18n.MsgMissingIdsOrInstanceIdsParam)
}

func parseAdminBatchSetModelInputs(req adminBatchSetModelRequest) (setModelInput, []setModelInput, *hcommon.RichError) {
	primary, inputErr := parseAdminBatchPrimarySetModelInput(req)
	if inputErr != nil {
		return setModelInput{}, nil, inputErr
	}
	fallbacks := make([]setModelInput, 0, len(req.Fallbacks))

	seen := make(map[string]struct{}, 1+len(req.Fallbacks))
	if key, ok := setModelInputBindingKey(primary); ok {
		seen[key] = struct{}{}
	}
	for _, payload := range req.Fallbacks {
		fallback, inputErr := parseAdminBatchFallbackSetModelInput(payload)
		if inputErr != nil {
			return setModelInput{}, nil, inputErr
		}
		if key, ok := setModelInputBindingKey(fallback); ok {
			if _, exists := seen[key]; exists {
				return setModelInput{}, nil, hcommon.I18nError(i18n.MsgBatchSetModelDuplicateModel)
			}
			seen[key] = struct{}{}
		}
		fallbacks = append(fallbacks, fallback)
	}
	return primary, fallbacks, nil
}

// HandleAdminBatchSetModel 管控端批量设置主模型。
// POST /admin/instances/batch-set-model
func HandleAdminBatchSetModel(w http.ResponseWriter, r *http.Request) {
	handleAdminBatchSetModel(w, r, defaultStatusResolver)
}

func handleAdminBatchSetModel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	reqCtx := r.Context()
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req adminBatchSetModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Logger(reqCtx).Warn("[BatchSetModel] 请求体解析失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}

	primaryInput, fallbackInputs, inputErr := parseAdminBatchSetModelInputs(req)
	if inputErr != nil {
		writeError(w, r, http.StatusBadRequest, inputErr)
		return
	}

	ids, instanceIDs, byID, selectorErr := parseAdminBatchSetModelSelectors(req)
	if selectorErr != nil {
		writeError(w, r, http.StatusBadRequest, selectorErr)
		return
	}

	// batchSetModelTarget 保存单个批量目标的解析结果：
	// index 用于保持响应顺序；instance 为 nil 表示目标不存在；result 是预填的返回项。
	type batchSetModelTarget struct {
		index    int
		instance *model.Instance
		result   adminBatchSetModelResult
	}

	var (
		targets  []batchSetModelTarget
		resolved int
	)
	if byID {
		var instances []model.Instance
		if err := model.DB(reqCtx).Where("id IN ?", ids).Find(&instances).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
			return
		}
		dbByID := make(map[uint]*model.Instance, len(instances))
		for i := range instances {
			dbByID[instances[i].ID] = &instances[i]
		}
		targets = make([]batchSetModelTarget, len(ids))
		for i, id := range ids {
			target := batchSetModelTarget{
				index: i,
				result: adminBatchSetModelResult{
					ID:     id,
					Status: "failed",
				},
			}
			if inst, ok := dbByID[id]; ok {
				resolved++
				target.instance = inst
				target.result = adminBatchSetModelResult{
					ID:         inst.ID,
					InstanceID: inst.InstanceId,
					Name:       inst.Name,
				}
			}
			targets[i] = target
		}
	} else {
		var instances []model.Instance
		if err := model.DB(reqCtx).Where("instance_id IN ?", instanceIDs).Find(&instances).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
			return
		}
		dbByInstanceID := make(map[string]*model.Instance, len(instances))
		for i := range instances {
			dbByInstanceID[instances[i].InstanceId] = &instances[i]
		}
		targets = make([]batchSetModelTarget, len(instanceIDs))
		for i, instanceID := range instanceIDs {
			target := batchSetModelTarget{
				index: i,
				result: adminBatchSetModelResult{
					InstanceID: instanceID,
					Status:     "failed",
				},
			}
			if inst, ok := dbByInstanceID[instanceID]; ok {
				resolved++
				target.instance = inst
				target.result = adminBatchSetModelResult{
					ID:         inst.ID,
					InstanceID: inst.InstanceId,
					Name:       inst.Name,
				}
			}
			targets[i] = target
		}
	}

	targetAgentType := ""
	for _, target := range targets {
		if target.instance == nil {
			continue
		}
		agentType := model.NormalizeAgentType(target.instance.AgentType)
		if targetAgentType == "" {
			targetAgentType = agentType
			continue
		}
		if agentType != targetAgentType {
			Logger(reqCtx).Info("[BatchSetModel] 批量设置模型被混合 Agent 类型限制拦截", "admin", getAdminUser(r), "requested", len(targets), "resolved", resolved)
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBatchSetModelMixedAgentTypes))
			return
		}
	}

	results := make([]adminBatchSetModelResult, len(targets))

	var wg sync.WaitGroup
	sem := make(chan struct{}, adminBatchSetModelConcurrency)
	for _, target := range targets {
		if target.instance == nil {
			target.result.Message = i18n.T(reqCtx, i18n.MsgInstanceNotFound)
			results[target.index] = target.result
			continue
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(execCtx context.Context) {
			defer wg.Done()
			defer func() { <-sem }()

			result := target.result
			applyErr := batchSetModelForInstance(execCtx, target.instance, target.instance.UserID, primaryInput, fallbackInputs, resolver)
			if applyErr != nil {
				result.Status = "failed"
				result.Message = hcommon.ErrorMessageWithCtx(reqCtx, applyErr.Err)
			} else {
				result.Status = "ok"
				result.Message = i18n.T(reqCtx, i18n.MsgBatchSetModelSuccess)
			}
			results[target.index] = result
		}(hcommon.DetachContext(reqCtx))
	}
	wg.Wait()

	Logger(reqCtx).Info("[BatchSetModel] 批量设置模型完成", "admin", getAdminUser(r), "requested", len(targets), "resolved", resolved)
	jsonOK(w, map[string]interface{}{"ok": true, "results": results})
}

// HandleAdminSetModel 管控端设置/替换主模型。
// POST /admin/instances/set-model
func HandleAdminSetModel(w http.ResponseWriter, r *http.Request) {
	handleAdminSetModel(w, r, defaultStatusResolver)
}

func handleAdminSetModel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	if !requireAdmin(w, r) {
		return
	}

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	setModel(w, r, instance, instance.UserID, resolver)
}

// HandleAdminAddModel 管控端添加模型（首个自动 primary，后续 fallback）。
// POST /admin/instances/add-model
func HandleAdminAddModel(w http.ResponseWriter, r *http.Request) {
	handleAdminAddModel(w, r, defaultStatusResolver)
}

func handleAdminAddModel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	if !requireAdmin(w, r) {
		return
	}

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	addModel(w, r, instance, instance.UserID, resolver)
}

// HandleAdminSwitchPrimaryModel 管控端切换主备模型。
// POST /admin/instances/switch-primary-model
func HandleAdminSwitchPrimaryModel(w http.ResponseWriter, r *http.Request) {
	handleAdminSwitchPrimaryModel(w, r, defaultStatusResolver)
}

func handleAdminSwitchPrimaryModel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	if !requireAdmin(w, r) {
		return
	}

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	switchPrimaryModel(w, r, instance, instance.UserID, resolver)
}

// HandleAdminDelModel 管控端删除模型绑定。
// POST /admin/instances/del-model
func HandleAdminDelModel(w http.ResponseWriter, r *http.Request) {
	handleAdminDelModel(w, r, defaultStatusResolver)
}

func handleAdminDelModel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	if !requireAdmin(w, r) {
		return
	}

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	deleteModel(w, r, instance, instance.UserID, resolver)
}

// ==========================================================================
// 管控端通道管理接口
// ==========================================================================

// HandleAdminAvailableChannels 返回实例可配置的通道列表（已启用 + 对该实例可见 + agent_type 支持）。
// GET /admin/instances/available-channels?id={instance_id}
func HandleAdminAvailableChannels(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}

	if err := checkInstanceSupportsChannel(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	var channels []model.AIChannel
	model.DB(r.Context()).Where("enabled = ?", true).Find(&channels)
	channels = model.SortChannelsByPredefined(channels)
	channels = filterChannelsByCurrentSiteScope(r.Context(), channels)

	// 可见性过滤：按实例绑定的分组（与用户端 HandleChannelsList 一致）
	channels = usergroup.FilterChannelsByVisibility(r.Context(), channels, instance.GroupID)

	// agent_type 白名单过滤（自定义通道豁免）
	rtChannels := make([]model.AIChannel, 0, len(channels))
	for _, ch := range channels {
		if ch.Custom || model.AgentTypeChannelAllowed(r.Context(), instance.AgentType, ch.ChannelID) {
			rtChannels = append(rtChannels, ch)
		}
	}

	type channelItem struct {
		model.AIChannel
		Params     []model.ChannelParam `json:"params"`
		AgentTypes []string             `json:"agent_types"`
	}
	items := make([]channelItem, 0, len(rtChannels))
	for _, ch := range rtChannels {
		var ats []string
		if ch.Custom {
			ats = model.GetChannelSupportedAgentTypes(r.Context())
		} else {
			ats = model.SupportedAgentTypesByChannel(r.Context(), ch.ChannelID)
		}
		items = append(items, channelItem{
			AIChannel:  ch,
			Params:     ch.Params(),
			AgentTypes: ats,
		})
	}

	jsonOK(w, map[string]interface{}{
		"ok":       true,
		"channels": items,
	})
}

// HandleAdminSetChannel 管控端设置/编辑通道配置。
// POST /admin/instances/set-channel
func HandleAdminSetChannel(w http.ResponseWriter, r *http.Request) {
	handleAdminSetChannel(w, r, defaultStatusResolver)
}

func handleAdminSetChannel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	if !requireAdmin(w, r) {
		return
	}

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	setChannel(w, r, instance, instance.UserID, resolver)
}

// HandleAdminDelChannel 管控端删除已配置通道。
// POST /admin/instances/del-channel
func HandleAdminDelChannel(w http.ResponseWriter, r *http.Request) {
	handleAdminDelChannel(w, r, defaultStatusResolver)
}

func handleAdminDelChannel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	if !requireAdmin(w, r) {
		return
	}

	instance, err := getInstanceByIDRaw(&w, r, 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	deleteChannel(w, r, instance, resolver)
}

// ---- 管控端移除本地 Agent（三期） ------------------------------------------

// HandleAdminLocalAgentRemove 管理员移除指定本地 Agent。
//
// 复用用户端共有的 createUninstallTeamaiTask：创建 uninstall_teamai 任务，实际卸载由
// reporter 拉命令执行（不立即删实例）。与用户端差异仅在鉴权（管理员无需 owner 校验）。
//
// 写入审计（路由层 WithAudit 包装）。
func HandleAdminLocalAgentRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		InstanceID uint `json:"instance_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	if req.InstanceID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidID))
		return
	}

	ctx := r.Context()
	var inst model.Instance
	if err := model.DB(ctx).Where("id = ? AND source = ?",
		req.InstanceID, model.InstanceSourceLocal).First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFoundOrNoPerm))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	// operator 记录操作人：取当前登录用户（管理员在管控端替用户卸载时记为操作人）；
	// admin-token / 机器人等无登录用户场景记为 0。
	operatorID := uint(0)
	if u, uErr := getLoginUser(r); uErr == nil && u != nil {
		operatorID = u.ID
	}

	var task *model.LocalAgentTask
	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		task, err = createUninstallTeamaiTask(ctx, tx, &inst, operatorID)
		return err
	})
	if txErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]any{
		"ok":      true,
		"task_id": task.ID,
		"status":  task.Status,
	})
}

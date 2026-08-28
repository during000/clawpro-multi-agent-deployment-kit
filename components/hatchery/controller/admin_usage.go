package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

type usageDataRow struct {
	Date                   string `json:"date,omitempty"`
	UserID                 uint   `json:"user_id,omitempty"`
	UserEmail              string `json:"user_email,omitempty"`
	UserName               string `json:"user_name,omitempty"`
	AIModelID              uint   `json:"ai_model_id,omitempty"`
	ModelName              string `json:"model_name,omitempty"`
	InstanceID             uint   `json:"instance_id,omitempty"`
	InstanceName           string `json:"instance_name,omitempty"`
	InstanceCVMID          string `json:"instance_cvm_id,omitempty"`
	DepartmentID           string `json:"department_id,omitempty"`
	DepartmentName         string `json:"department_name,omitempty"`
	DepartmentPath         string `json:"department_path,omitempty"`
	GroupID                uint   `json:"group_id,omitempty"`
	GroupName              string `json:"group_name,omitempty"`
	GroupFullPath          string `json:"group_full_path,omitempty"`
	PromptTokens           int64  `json:"prompt_tokens"`
	CompletionTokens       int64  `json:"completion_tokens"`
	TotalTokens            int64  `json:"total_tokens"`
	PromptCacheReadTokens  int64  `json:"prompt_cache_read_tokens"`
	PromptCacheWriteTokens int64  `json:"prompt_cache_write_tokens"`
	RequestCount           int64  `json:"request_count"`

	TokenQuotaRules        *[]model.TokenQuotaRule `json:"token_quota_rules,omitempty"`
	TokenQuotaUsages       *[]tokenQuotaUsage      `json:"token_quota_usages,omitempty"`
	TokenQuotaGroups       *[]tokenQuotaGroupData  `json:"token_quota_groups,omitempty"`
	GlobalTokenQuotaRules  *[]model.TokenQuotaRule `json:"global_token_quota_rules,omitempty"`
	GlobalTokenQuotaUsages *[]tokenQuotaUsage      `json:"global_token_quota_usages,omitempty"`
}

type tokenQuotaGroupData struct {
	GroupID          uint                   `json:"group_id"`
	GroupName        string                 `json:"group_name"`
	GroupFullPath    string                 `json:"group_full_path"`
	TokenQuotaRules  []model.TokenQuotaRule `json:"token_quota_rules"`
	TokenQuotaUsages []tokenQuotaUsage      `json:"token_quota_usages"`
}

// parseDateRange parses start_date/end_date from query. Defaults to today–today.
func parseDateRange(r *http.Request) (start, end time.Time) {
	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")
	today := model.LocalToday()

	start = today
	if startStr != "" {
		if s, err := model.ParseLocalDate(startStr); err == nil {
			start = s
		}
	}
	end = today
	if endStr != "" {
		if e, err := model.ParseLocalDate(endStr); err == nil {
			end = e
		}
	}
	if end.Before(start) {
		start, end = end, start
	}
	return start, end
}

// parseGroupBySet parses a comma-separated group_by string into a set.
// Valid dimensions: date, user, model, instance, department. Default: date,model.
func parseGroupBySet(s string) map[string]bool {
	result := make(map[string]bool)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "date" || part == "user" || part == "model" || part == "instance" || part == "department" || part == "group" {
			result[part] = true
		}
	}
	if len(result) == 0 {
		result["date"] = true
		result["model"] = true
	}
	return result
}

func parseUint(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

// resolveInstancePKFromParam 解析 instance_id 入参为 instances.id（DB 主键）。
//
// 兼容历史"用法不严格"的客户端：
//   - 入参是纯数字 → 直接当作 DB 主键返回
//   - 入参是 CVM ID 字符串（如 "ins-xxx"）→ 反查 instances.id；
//     若 ownerUserID > 0，则附加 user_id 过滤防越权
//   - 入参为空 / 反查不到 → 返回 0
//
// 返回 0 时调用方应跳过该过滤条件，与历史"未传值则不过滤"的语义保持一致。
func resolveInstancePKFromParam(ctx context.Context, raw string, ownerUserID uint) uint64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	// 1) 纯数字 → DB 主键
	if v, err := strconv.ParseUint(raw, 10, 64); err == nil {
		return v
	}
	// 2) 否则按 CVM ID 反查
	q := model.DB(ctx).Model(&model.Instance{}).Unscoped().Where("instance_id = ?", raw)
	if ownerUserID > 0 {
		q = q.Where("user_id = ?", ownerUserID)
	}
	var inst model.Instance
	if err := q.Select("id").First(&inst).Error; err != nil {
		return 0
	}
	return uint64(inst.ID)
}

// resolveInstancePKFromIDOrParam 是统计/明细类接口的"双参数"实例过滤解析器，
// 与项目其他双参数接口（id + instance_id）的语义对齐：
//   - idStr 非空 → 严格按 DB 主键解析；非法或 0 视为"未指定"，返回 0；
//   - idStr 为空 → 退化到 resolveInstancePKFromParam，对 instance_id 兼容
//     "纯数字主键 / CVM ID 字符串"两种历史用法；
//   - 两者皆空 → 返回 0（调用方按"未指定"处理，不附加过滤）。
//
// ownerUserID > 0 时，CVM ID 反查会附加 user_id 限制以防越权（用户侧）；
// 管理员侧应传 0。
func resolveInstancePKFromIDOrParam(ctx context.Context, idStr, instanceIDStr string, ownerUserID uint) uint64 {
	if s := strings.TrimSpace(idStr); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil && v > 0 {
			return v
		}
		// id 字段语义明确就是 DB 主键，非法值不再回退到 instance_id，避免歧义。
		return 0
	}
	return resolveInstancePKFromParam(ctx, instanceIDStr, ownerUserID)
}

// HandleAdminUsageData querys model usage data
// Query params:
//   - start_date, end_date: date range (default today)
//   - group_by: comma-separated dimensions: date, user, model, instance (default "date,model")
//   - user_id, ai_model_id, instance_id: optional filters
//   - order_by: field to sort by, "total_tokens" (default) or "request_count"
//   - order: "desc" sorts descending by order_by field; otherwise unordered
//
// usageDataParams holds query parameters for usage data queries.
type usageDataParams struct {
	Start            time.Time
	End              time.Time
	GroupBy          map[string]bool
	FilterUserID     uint64
	FilterModelID    uint64
	FilterInstanceID uint64
	FilterGroupID    uint64 // 按 agent 绑定的分组 ID 筛选
	// IncludeUserUngrouped 仅在 FilterUserID > 0 且 FilterGroupID > 0 时生效：
	// 兼容查询——把该用户名下 group_id=0 的"无分组创建的旧 agent"产生的用量
	// 一并计入。仅用户端 /quota/data 设置该标记，管理端按分组维度统计请勿启用，
	// 以免污染聚合口径。
	IncludeUserUngrouped bool
	FilterDepartmentID   string // 按部门 ID 筛选（支持任意部门，非仅主部门名）
	OrderBy              string // 排序字段: "total_tokens" (默认) 或 "request_count"
	OrderDesc            bool   // 是否降序
}

// usageDataResult holds the query result.
type usageDataResult struct {
	StartDate string         `json:"start_date"`
	EndDate   string         `json:"end_date"`
	GroupBy   []string       `json:"group_by"`
	Rows      []usageDataRow `json:"rows"`
}

func queryUsageData(ctx context.Context, p usageDataParams) (*usageDataResult, error) {
	groupBy := p.GroupBy

	// ── 特殊处理：group_by=department，按子部门聚合 ──
	if groupBy["department"] {
		return queryUsageByDepartment(ctx, p)
	}

	// ── 特殊处理：group_by=group，按分组聚合 ──
	if groupBy["group"] {
		return queryUsageByGroup(ctx, p)
	}

	var selectCols []string
	var groupCols []string
	if groupBy["date"] {
		selectCols = append(selectCols, "DATE(date) as date")
		groupCols = append(groupCols, "DATE(date)")
	}
	if groupBy["user"] {
		selectCols = append(selectCols, "user_id")
		groupCols = append(groupCols, "user_id")
	}
	if groupBy["model"] {
		selectCols = append(selectCols, "ai_model_id")
		groupCols = append(groupCols, "ai_model_id")
	}
	if groupBy["instance"] {
		selectCols = append(selectCols, "instance_id")
		groupCols = append(groupCols, "instance_id")
	}
	selectCols = append(selectCols,
		"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens",
		"COALESCE(SUM(completion_tokens), 0) as completion_tokens",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COALESCE(SUM(prompt_cache_read_tokens), 0) as prompt_cache_read_tokens",
		"COALESCE(SUM(prompt_cache_write_tokens), 0) as prompt_cache_write_tokens",
		"COALESCE(SUM(request_count), 0) as request_count",
	)

	type rawRow struct {
		Date                   string
		UserID                 uint
		AIModelID              uint
		InstanceID             uint
		PromptTokens           int64
		CompletionTokens       int64
		TotalTokens            int64
		PromptCacheReadTokens  int64
		PromptCacheWriteTokens int64
		RequestCount           int64
	}

	q := model.DB(ctx).Model(&model.DailyUsageSummary{}).
		Where("date >= ? AND date < ?", p.Start, p.End.AddDate(0, 0, 1))
	if p.FilterUserID > 0 {
		q = q.Where("user_id = ?", p.FilterUserID)
	}
	if p.FilterModelID > 0 {
		q = q.Where("ai_model_id = ?", p.FilterModelID)
	}
	if p.FilterInstanceID > 0 {
		q = q.Where("instance_id = ?", p.FilterInstanceID)
	}
	if p.FilterGroupID > 0 {
		// 用户端兼容：当显式指定 group_id 且要求"包含本用户无分组旧 agent"时，
		// 把 group_id=0 的旧用量也并入展示，避免老 agent 数据"消失"。
		if p.IncludeUserUngrouped && p.FilterUserID > 0 {
			q = q.Where("group_id IN ?", []uint64{p.FilterGroupID, 0})
		} else {
			q = q.Where("group_id = ?", p.FilterGroupID)
		}
	}
	if p.FilterDepartmentID != "" {
		// 按部门 ID 筛选：查找 DepartmentsJSON 中包含该部门 ID 的用户
		q = q.Where("user_id IN (?)",
			model.DB(ctx).Model(&model.User{}).Select("id").
				Where("one_id_sub IN (?)",
					model.DB(ctx).Model(&model.OneIDUserProfile{}).Select("one_id_sub").Where(
						"main_dept_id = ? OR departments_json LIKE ?", p.FilterDepartmentID, model.DeptIDLikePattern(p.FilterDepartmentID),
					),
				),
		)
	}

	var rawRows []rawRow
	qb := q.Select(strings.Join(selectCols, ", "))
	if len(groupCols) > 0 {
		qb = qb.Group(strings.Join(groupCols, ", "))
	}
	if p.OrderDesc {
		// OrderBy 已在调用方白名单校验（仅 total_tokens / request_count），此处用 clause 避免拼接
		switch p.OrderBy {
		case "request_count":
			qb = qb.Order("request_count DESC")
		default:
			qb = qb.Order("total_tokens DESC")
		}
	}
	if err := qb.Scan(&rawRows).Error; err != nil {
		return nil, err
	}

	userIDs := make(map[uint]struct{})
	modelIDs := make(map[uint]struct{})
	instanceIDs := make(map[uint]struct{})
	for _, rr := range rawRows {
		if rr.UserID > 0 {
			userIDs[rr.UserID] = struct{}{}
		}
		if rr.AIModelID > 0 {
			modelIDs[rr.AIModelID] = struct{}{}
		}
		if rr.InstanceID > 0 {
			instanceIDs[rr.InstanceID] = struct{}{}
		}
	}
	userMap := loadUserMap(ctx, userIDs)
	userNameMap := loadUserNameMap(ctx, userIDs)
	modelMap := loadAIModelMap(ctx, modelIDs)
	instanceMap := loadInstanceMap(ctx, instanceIDs)

	// 当按用户聚合时，加载主部门信息
	type userDeptInfo struct {
		DepartmentID   string
		DepartmentName string
		DepartmentPath string
	}
	userDeptMap := make(map[uint]userDeptInfo)
	if groupBy["user"] && len(userIDs) > 0 {
		// user_id → one_id_sub
		type userSubRow struct {
			ID       uint
			OneIDSub string
		}
		var userSubRows []userSubRow
		model.DB(ctx).Model(&model.User{}).Select("id, one_id_sub").
			Where("id IN ? AND one_id_sub != ''", mapKeys(userIDs)).
			Scan(&userSubRows)

		subs := make([]string, 0, len(userSubRows))
		subToUserID := make(map[string]uint, len(userSubRows))
		for _, us := range userSubRows {
			subs = append(subs, us.OneIDSub)
			subToUserID[us.OneIDSub] = us.ID
		}

		if len(subs) > 0 {
			type profileRow struct {
				OneIDSub     string
				MainDeptID   string
				MainDeptName string
			}
			var profiles []profileRow
			model.DB(ctx).Model(&model.OneIDUserProfile{}).
				Select("one_id_sub, main_dept_id, main_dept_name").
				Where("one_id_sub IN ?", subs).
				Scan(&profiles)

			globalDeptMap := model.BuildFullDeptMap(ctx)
			for _, pr := range profiles {
				uid := subToUserID[pr.OneIDSub]
				userDeptMap[uid] = userDeptInfo{
					DepartmentID:   pr.MainDeptID,
					DepartmentName: pr.MainDeptName,
					DepartmentPath: buildDepartmentPath(globalDeptMap, pr.MainDeptID),
				}
			}
		}
	}

	rows := make([]usageDataRow, 0, len(rawRows))
	for _, rr := range rawRows {
		row := usageDataRow{
			PromptTokens:           rr.PromptTokens,
			CompletionTokens:       rr.CompletionTokens,
			TotalTokens:            rr.TotalTokens,
			PromptCacheReadTokens:  rr.PromptCacheReadTokens,
			PromptCacheWriteTokens: rr.PromptCacheWriteTokens,
			RequestCount:           rr.RequestCount,
		}
		if groupBy["date"] {
			row.Date = rr.Date
		}
		if groupBy["user"] {
			row.UserID = rr.UserID
			row.UserEmail = userMap[rr.UserID]
			row.UserName = userNameMap[rr.UserID]
			if di, ok := userDeptMap[rr.UserID]; ok {
				row.DepartmentID = di.DepartmentID
				row.DepartmentName = di.DepartmentName
				row.DepartmentPath = di.DepartmentPath
			}
		}
		if groupBy["model"] {
			row.AIModelID = rr.AIModelID
			if m, ok := modelMap[rr.AIModelID]; ok {
				row.ModelName = m.DisplayName()
			}
		}
		if groupBy["instance"] {
			row.InstanceID = rr.InstanceID
			row.InstanceName = instanceMap[rr.InstanceID].Name
			row.InstanceCVMID = instanceMap[rr.InstanceID].InstanceCVMId
		}
		rows = append(rows, row)
	}
	if groupBy["user"] {
		attachUserTokenQuotaData(ctx, rows, uint(p.FilterGroupID))
	}

	groupByKeys := make([]string, 0, len(groupBy))
	for k := range groupBy {
		groupByKeys = append(groupByKeys, k)
	}

	return &usageDataResult{
		StartDate: p.Start.Format("2006-01-02"),
		EndDate:   p.End.Format("2006-01-02"),
		GroupBy:   groupByKeys,
		Rows:      rows,
	}, nil
}

// queryUsageByDepartment 按子部门聚合 token 用量。
// 当 FilterDepartmentID 非空时，查找该部门的直接子部门；为空时查找所有顶级部门。
// 对每个子部门下的所有用户（递归包含孙部门）进行 token 聚合。
// 没有数据的子部门补零，结果按总请求数降序排列。
func queryUsageByDepartment(ctx context.Context, p usageDataParams) (*usageDataResult, error) {
	globalDeptMap := model.BuildFullDeptMap(ctx)

	// 1. 找到目标部门的直接子部门
	parentID := p.FilterDepartmentID
	var childDepts []model.OneIDDepartment
	for _, dept := range globalDeptMap {
		if dept.DepartmentParentID == parentID {
			childDepts = append(childDepts, dept)
		}
	}

	// 2. 对每个子部门收集其所有后代部门 ID（递归），用于查找用户
	collectDescendantIDs := func(rootID string) []string {
		ids := []string{rootID}
		queue := []string{rootID}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, d := range globalDeptMap {
				if d.DepartmentParentID == cur && d.DepartmentID != cur {
					ids = append(ids, d.DepartmentID)
					queue = append(queue, d.DepartmentID)
				}
			}
		}
		return ids
	}

	// 3. 构建 userID → 所属子部门 ID 的映射
	// 读取所有用户画像的 DepartmentsJSON
	type profileRow struct {
		OneIDSub        string
		DepartmentsJSON string
	}
	var profiles []profileRow
	model.DB(ctx).Model(&model.OneIDUserProfile{}).
		Where("departments_json != '' AND departments_json != '[]'").
		Select("one_id_sub, departments_json").
		Scan(&profiles)

	// oneIDSub → 用户的所有部门 ID 集合
	subDeptMap := make(map[string]map[string]bool)
	for _, pr := range profiles {
		var depts []model.OneIDDepartment
		if err := json.Unmarshal([]byte(pr.DepartmentsJSON), &depts); err != nil {
			slog.Warn("queryUsageByDepartment: failed to parse departments_json, skipping",
				"sub", pr.OneIDSub, "err", err)
			continue
		}
		deptSet := make(map[string]bool)
		for _, d := range depts {
			deptSet[d.DepartmentID] = true
		}
		subDeptMap[pr.OneIDSub] = deptSet
	}

	// oneIDSub → users.id 映射
	type userSubRow struct {
		ID       uint
		OneIDSub string
	}
	var userSubs []userSubRow
	model.DB(ctx).Model(&model.User{}).Select("id, one_id_sub").Where("one_id_sub != ''").Scan(&userSubs)
	subToUserID := make(map[string]uint)
	for _, u := range userSubs {
		subToUserID[u.OneIDSub] = u.ID
	}

	// 4. 对每个子部门聚合
	type deptAgg struct {
		DeptID                 string
		DeptName               string
		DeptPath               string
		PromptTokens           int64
		CompletionTokens       int64
		TotalTokens            int64
		PromptCacheReadTokens  int64
		PromptCacheWriteTokens int64
		RequestCount           int64
	}

	results := make([]deptAgg, 0, len(childDepts))
	for _, child := range childDepts {
		descIDs := collectDescendantIDs(child.DepartmentID)
		descSet := make(map[string]bool, len(descIDs))
		for _, id := range descIDs {
			descSet[id] = true
		}

		// 找到属于此子部门（及后代）的所有 userID
		// 注意：不再 break，一个用户如果属于多个后代部门，仍只计入一次（去重）
		userIDSet := make(map[uint]bool)
		for sub, deptSet := range subDeptMap {
			for deptID := range deptSet {
				if descSet[deptID] {
					if uid, ok := subToUserID[sub]; ok {
						userIDSet[uid] = true
					}
				}
			}
		}
		userIDs := make([]uint, 0, len(userIDSet))
		for uid := range userIDSet {
			userIDs = append(userIDs, uid)
		}

		agg := deptAgg{
			DeptID:   child.DepartmentID,
			DeptName: child.DepartmentName,
			DeptPath: buildDepartmentPath(globalDeptMap, child.DepartmentID),
		}

		if len(userIDs) > 0 {
			type sumRow struct {
				PromptTokens           int64
				CompletionTokens       int64
				TotalTokens            int64
				PromptCacheReadTokens  int64
				PromptCacheWriteTokens int64
				RequestCount           int64
			}
			var sr sumRow
			model.DB(ctx).Model(&model.DailyUsageSummary{}).
				Where("date >= ? AND date < ?", p.Start, p.End.AddDate(0, 0, 1)).
				Where("user_id IN ?", userIDs).
				Select("COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, COALESCE(SUM(completion_tokens), 0) as completion_tokens, COALESCE(SUM(total_tokens), 0) as total_tokens, COALESCE(SUM(prompt_cache_read_tokens), 0) as prompt_cache_read_tokens, COALESCE(SUM(prompt_cache_write_tokens), 0) as prompt_cache_write_tokens, COALESCE(SUM(request_count), 0) as request_count").
				Scan(&sr)
			agg.PromptTokens = sr.PromptTokens
			agg.CompletionTokens = sr.CompletionTokens
			agg.TotalTokens = sr.TotalTokens
			agg.PromptCacheReadTokens = sr.PromptCacheReadTokens
			agg.PromptCacheWriteTokens = sr.PromptCacheWriteTokens
			agg.RequestCount = sr.RequestCount
		}
		// 没有用户或没有数据时保持零值

		results = append(results, agg)
	}

	// 5. 按前端指定的字段排序
	if p.OrderDesc {
		orderField := p.OrderBy
		if orderField == "" {
			orderField = "total_tokens"
		}
		switch orderField {
		case "request_count":
			sort.Slice(results, func(i, j int) bool { return results[i].RequestCount > results[j].RequestCount })
		default: // total_tokens
			sort.Slice(results, func(i, j int) bool { return results[i].TotalTokens > results[j].TotalTokens })
		}
	}

	// 6. 转换为 usageDataRow
	rows := make([]usageDataRow, 0, len(results))
	for _, r := range results {
		rows = append(rows, usageDataRow{
			DepartmentID:           r.DeptID,
			DepartmentName:         r.DeptName,
			DepartmentPath:         r.DeptPath,
			PromptTokens:           r.PromptTokens,
			CompletionTokens:       r.CompletionTokens,
			TotalTokens:            r.TotalTokens,
			PromptCacheReadTokens:  r.PromptCacheReadTokens,
			PromptCacheWriteTokens: r.PromptCacheWriteTokens,
			RequestCount:           r.RequestCount,
		})
	}

	return &usageDataResult{
		StartDate: p.Start.Format("2006-01-02"),
		EndDate:   p.End.Format("2006-01-02"),
		GroupBy:   []string{"department"},
		Rows:      rows,
	}, nil
}

// queryUsageByGroup 按分组聚合 token 用量。
// 当 FilterGroupID 非空时，查找该组及其所有后代组的用量；为空时按全部组聚合。
// 使用 daily_usage_summaries.group_id + group_closure 实现后代展开。
func queryUsageByGroup(ctx context.Context, p usageDataParams) (*usageDataResult, error) {
	// 确定要聚合的 group_id 范围
	var targetGroupIDs []uint
	if p.FilterGroupID > 0 {
		// 查该组的所有后代（含自身）
		type closureRow struct {
			DescendantID uint `gorm:"column:descendant_id"`
		}
		var rows []closureRow
		model.DB(ctx).Table("group_closure").
			Select("descendant_id").
			Where("ancestor_id = ?", p.FilterGroupID).
			Scan(&rows)
		for _, r := range rows {
			targetGroupIDs = append(targetGroupIDs, r.DescendantID)
		}
		if len(targetGroupIDs) == 0 {
			targetGroupIDs = []uint{uint(p.FilterGroupID)}
		}
	}

	// 按 group_id 聚合查询
	type rawRow struct {
		GroupID                uint
		PromptTokens           int64
		CompletionTokens       int64
		TotalTokens            int64
		PromptCacheReadTokens  int64
		PromptCacheWriteTokens int64
		RequestCount           int64
	}

	q := model.DB(ctx).Model(&model.DailyUsageSummary{}).
		Where("date >= ? AND date < ?", p.Start, p.End.AddDate(0, 0, 1)).
		Where("group_id > 0") // 只聚合有分组的记录
	if len(targetGroupIDs) > 0 {
		q = q.Where("group_id IN ?", targetGroupIDs)
	}
	if p.FilterUserID > 0 {
		q = q.Where("user_id = ?", p.FilterUserID)
	}
	if p.FilterModelID > 0 {
		q = q.Where("ai_model_id = ?", p.FilterModelID)
	}

	var rawRows []rawRow
	qb := q.Select("group_id, COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, COALESCE(SUM(completion_tokens), 0) as completion_tokens, COALESCE(SUM(total_tokens), 0) as total_tokens, COALESCE(SUM(prompt_cache_read_tokens), 0) as prompt_cache_read_tokens, COALESCE(SUM(prompt_cache_write_tokens), 0) as prompt_cache_write_tokens, COALESCE(SUM(request_count), 0) as request_count").
		Group("group_id")
	if p.OrderDesc {
		switch p.OrderBy {
		case "request_count":
			qb = qb.Order("request_count DESC")
		default:
			qb = qb.Order("total_tokens DESC")
		}
	}
	if err := qb.Scan(&rawRows).Error; err != nil {
		return nil, err
	}

	// 加载分组名和 full_path
	groupIDs := make([]uint, 0, len(rawRows))
	for _, r := range rawRows {
		groupIDs = append(groupIDs, r.GroupID)
	}
	type groupInfoRow struct {
		ID       uint
		Name     string
		FullPath string
	}
	var groupInfos []groupInfoRow
	if len(groupIDs) > 0 {
		model.DB(ctx).Table("user_groups").
			Select("id, name, full_path").
			Where("id IN ?", groupIDs).
			Scan(&groupInfos)
	}
	groupNameMap := make(map[uint]string, len(groupInfos))
	groupPathMap := make(map[uint]string, len(groupInfos))
	for _, g := range groupInfos {
		groupNameMap[g.ID] = g.Name
		groupPathMap[g.ID] = g.FullPath
	}

	// 构建结果
	dataRows := make([]usageDataRow, 0, len(rawRows))
	for _, r := range rawRows {
		dataRows = append(dataRows, usageDataRow{
			GroupID:                r.GroupID,
			GroupName:              groupNameMap[r.GroupID],
			GroupFullPath:          groupPathMap[r.GroupID],
			PromptTokens:           r.PromptTokens,
			CompletionTokens:       r.CompletionTokens,
			TotalTokens:            r.TotalTokens,
			PromptCacheReadTokens:  r.PromptCacheReadTokens,
			PromptCacheWriteTokens: r.PromptCacheWriteTokens,
			RequestCount:           r.RequestCount,
		})
	}
	attachGroupGlobalTokenQuotaData(ctx, dataRows)

	return &usageDataResult{
		StartDate: p.Start.Format("2006-01-02"),
		EndDate:   p.End.Format("2006-01-02"),
		GroupBy:   []string{"group"},
		Rows:      dataRows,
	}, nil
}

// Response: { start_date, end_date, group_by: [...], rows: [...] }
func HandleAdminUsageData(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	start, end := parseDateRange(r)
	departmentID := r.URL.Query().Get("department_id")
	if departmentID == "" {
		departmentID = r.URL.Query().Get("department") // 兼容旧参数
	}

	// 解析并校验 order_by 参数（可选，默认 total_tokens）
	orderBy := r.URL.Query().Get("order_by")
	if orderBy == "" {
		orderBy = "total_tokens"
	}
	if orderBy != "total_tokens" && orderBy != "request_count" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidOrderBy))
		return
	}

	result, err := queryUsageData(r.Context(), usageDataParams{
		Start:         start,
		End:           end,
		GroupBy:       parseGroupBySet(r.URL.Query().Get("group_by")),
		FilterUserID:  parseUint(r.URL.Query().Get("user_id")),
		FilterModelID: parseUint(r.URL.Query().Get("ai_model_id")),
		// instance_id 允许传 DB 主键数字或 CVM ID 字符串（如 ins-xxx）。
		// 双参数兼容：优先 id（DB 主键），否则 instance_id（兼容纯数字主键 / CVM ID 字符串）。
		FilterInstanceID:   resolveInstancePKFromIDOrParam(r.Context(), r.URL.Query().Get("id"), r.URL.Query().Get("instance_id"), 0),
		FilterGroupID:      parseUint(r.URL.Query().Get("group_id")),
		FilterDepartmentID: departmentID,
		OrderBy:            orderBy,
		OrderDesc:          r.URL.Query().Get("order") == "desc",
	})
	if err != nil {
		slog.Error("查询用量数据失败",
			"start", start, "end", end,
			"group_by", r.URL.Query().Get("group_by"),
			"user_id", r.URL.Query().Get("user_id"),
			"ai_model_id", r.URL.Query().Get("ai_model_id"),
			"instance_id", r.URL.Query().Get("instance_id"),
			"group_id", r.URL.Query().Get("group_id"),
			"department_id", departmentID,
			"order_by", orderBy, "order", r.URL.Query().Get("order"),
			"err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryUsageDataFailed).WithDetail(err.Error()))
		return
	}
	config := model.GetSiteConfig(r.Context())
	globalQuotaRules := config.ResolvedGlobalTokenQuotaRules()
	globalQuotaDay, _ := model.EffectiveGlobalTokenQuotaLegacyFields(config.GlobalTokenQuotaDay, config.GlobalTokenQuotaPeriod, config.GlobalTokenQuotaRules)
	jsonOK(w, map[string]interface{}{
		"start_date":                result.StartDate,
		"end_date":                  result.EndDate,
		"group_by":                  result.GroupBy,
		"rows":                      result.Rows,
		"global_token_quota_day":    globalQuotaDay,
		"global_token_quota_rules":  globalQuotaRules,
		"global_token_quota_usages": globalTokenQuotaUsages(r.Context(), 0, globalQuotaRules),
	})
}

func attachUserTokenQuotaData(ctx context.Context, rows []usageDataRow, groupID uint) {
	userIDs := make(map[uint]struct{})
	for _, row := range rows {
		if row.UserID > 0 {
			userIDs[row.UserID] = struct{}{}
		}
	}
	if len(userIDs) == 0 {
		return
	}

	var users []model.User
	model.DB(ctx).Where("id IN ?", mapKeys(userIDs)).Find(&users)
	usersByID := make(map[uint]model.User, len(users))
	for _, user := range users {
		usersByID[user.ID] = user
	}

	for i := range rows {
		user, ok := usersByID[rows[i].UserID]
		if !ok {
			continue
		}
		rules := resolveEffectiveUserTokenQuotaRules(ctx, user, groupID)
		usages := userTokenQuotaUsages(ctx, user.ID, groupID, rules)
		rows[i].TokenQuotaRules = &rules
		rows[i].TokenQuotaUsages = &usages
	}

	if groupID == 0 {
		attachUserTokenQuotaGroups(ctx, rows, usersByID)
	}
}

func attachUserTokenQuotaGroups(ctx context.Context, rows []usageDataRow, usersByID map[uint]model.User) {
	if len(usersByID) == 0 {
		return
	}
	userIDs := make([]uint, 0, len(usersByID))
	for userID := range usersByID {
		userIDs = append(userIDs, userID)
	}

	groupsByUserID, err := model.GetUserGroupsByUserIDs(ctx, userIDs)
	if err != nil {
		slog.Warn("查询用户所属组失败，跳过 token quota groups", "err", err)
		return
	}

	quotaGroupsByUserID := make(map[uint][]tokenQuotaGroupData, len(groupsByUserID))
	for userID, groups := range groupsByUserID {
		user := usersByID[userID]
		sort.Slice(groups, func(i, j int) bool {
			return groups[i].ID < groups[j].ID
		})
		quotaGroups := make([]tokenQuotaGroupData, 0, len(groups))
		for _, group := range groups {
			rules := resolveEffectiveUserTokenQuotaRules(ctx, user, group.ID)
			usages := userTokenQuotaUsages(ctx, user.ID, group.ID, rules)
			quotaGroups = append(quotaGroups, tokenQuotaGroupData{
				GroupID:          group.ID,
				GroupName:        group.Name,
				GroupFullPath:    group.FullPath,
				TokenQuotaRules:  rules,
				TokenQuotaUsages: usages,
			})
		}
		quotaGroupsByUserID[userID] = quotaGroups
	}

	for i := range rows {
		if quotaGroups, ok := quotaGroupsByUserID[rows[i].UserID]; ok {
			rows[i].TokenQuotaGroups = &quotaGroups
		}
	}
}

func attachGroupGlobalTokenQuotaData(ctx context.Context, rows []usageDataRow) {
	if len(rows) == 0 {
		return
	}
	siteConfig := model.GetSiteConfig(ctx)
	for i := range rows {
		if rows[i].GroupID == 0 {
			continue
		}
		scope := resolveEffectiveGlobalTokenQuotaScope(ctx, siteConfig, rows[i].GroupID)
		rules := scope.Rules
		usages := globalTokenQuotaUsages(ctx, scope.UsageGroupID, rules)
		rows[i].GlobalTokenQuotaRules = &rules
		rows[i].GlobalTokenQuotaUsages = &usages
	}
}

func loadUserMap(ctx context.Context, ids map[uint]struct{}) map[uint]string {
	m := make(map[uint]string)
	if len(ids) == 0 {
		return m
	}
	var users []model.User
	model.DB(ctx).Where("id IN ?", mapKeys(ids)).Find(&users)
	for _, u := range users {
		m[u.ID] = u.Email
	}
	return m
}

func loadUserNameMap(ctx context.Context, ids map[uint]struct{}) map[uint]string {
	m := make(map[uint]string)
	if len(ids) == 0 {
		return m
	}
	var users []model.User
	model.DB(ctx).Where("id IN ?", mapKeys(ids)).Find(&users)
	for _, u := range users {
		m[u.ID] = u.Username
	}
	return m
}

func loadAIModelMap(ctx context.Context, ids map[uint]struct{}) map[uint]model.AIModel {
	m := make(map[uint]model.AIModel)
	if len(ids) == 0 {
		return m
	}
	var models []model.AIModel
	model.DB(ctx).Where("id IN ?", mapKeys(ids)).Find(&models)
	for _, mod := range models {
		m[mod.ID] = mod
	}
	return m
}

type instanceInfo struct {
	Name          string
	InstanceCVMId string
}

func loadInstanceMap(ctx context.Context, ids map[uint]struct{}) map[uint]instanceInfo {
	m := make(map[uint]instanceInfo)
	if len(ids) == 0 {
		return m
	}
	var instances []model.Instance
	model.DB(ctx).Unscoped().Where("id IN ?", mapKeys(ids)).Find(&instances)
	for _, inst := range instances {
		m[inst.ID] = instanceInfo{Name: inst.Name, InstanceCVMId: inst.InstanceId}
	}
	return m
}

func mapKeys(m map[uint]struct{}) []uint {
	keys := make([]uint, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func queryActiveUsers(ctx context.Context) ([]model.User, error) {
	var users []model.User
	if err := model.DB(ctx).Where("deleted_at IS NULL").Order("id").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func queryEnabledModels(ctx context.Context) ([]model.AIModel, error) {
	var models []model.AIModel
	if err := model.DB(ctx).Where("enabled = ? AND NOT (provider = ? AND model_id = ?)", true, model.BuiltinModelProvider, model.BuiltinModelID).Order("id").Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// HandleAdminUsageLogs returns paginated LLMUsageLog records within a date range.
//
// Query params:
//   - start_date, end_date: date range (default today)
//   - page: page number starting from 1 (default 1)
//   - page_size: records per page (default 50, max 200)
//   - user_id, ai_model_id, instance_id: optional filters
func HandleAdminUsageLogs(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	start, end := parseDateRange(r)
	endNext := end.Add(24 * time.Hour) // exclusive upper bound

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	filterUserID, _ := strconv.ParseUint(r.URL.Query().Get("user_id"), 10, 64)
	filterModelID, _ := strconv.ParseUint(r.URL.Query().Get("ai_model_id"), 10, 64)
	// instance_id 允许传 DB 主键数字或 CVM ID 字符串（如 ins-xxx）。
	// 双参数兼容：优先 id（DB 主键），否则 instance_id（兼容纯数字主键 / CVM ID 字符串）。
	filterInstanceID := resolveInstancePKFromIDOrParam(r.Context(), r.URL.Query().Get("id"), r.URL.Query().Get("instance_id"), 0)

	q := model.DB(r.Context()).Model(&model.LLMUsageLog{}).
		Where("created_at >= ? AND created_at < ?", start, endNext)
	if filterUserID > 0 {
		q = q.Where("user_id = ?", filterUserID)
	}
	if filterModelID > 0 {
		q = q.Where("ai_model_id = ?", filterModelID)
	}
	if filterInstanceID > 0 {
		q = q.Where("instance_id = ?", filterInstanceID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		slog.Error("查询用量日志总数失败",
			"start", start, "end", end,
			"user_id", filterUserID, "ai_model_id", filterModelID, "instance_id", filterInstanceID,
			"err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryRecordCountFailed).WithDetail(err.Error()))
		return
	}

	var logs []model.LLMUsageLog
	qb := q.Order("created_at DESC")
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		qb = qb.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	if err := qb.Find(&logs).Error; err != nil {
		slog.Error("查询用量日志列表失败",
			"start", start, "end", end,
			"user_id", filterUserID, "ai_model_id", filterModelID, "instance_id", filterInstanceID,
			"page", page, "page_size", pageSize,
			"err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryUsageLogsFailed).WithDetail(err.Error()))
		return
	}

	// Resolve user IDs to usernames.
	userIDs := make(map[uint]struct{})
	for _, log := range logs {
		if log.UserID > 0 {
			userIDs[log.UserID] = struct{}{}
		}
	}
	userNames := loadUserNameMap(r.Context(), userIDs)

	type usageLogRow struct {
		ID                     uint      `json:"id"`
		UserName               string    `json:"user_name"`
		Provider               string    `json:"provider"`
		Model                  string    `json:"model"`
		PromptTokens           int       `json:"prompt_tokens"`
		CompletionTokens       int       `json:"completion_tokens"`
		TotalTokens            int       `json:"total_tokens"`
		PromptCacheReadTokens  int       `json:"prompt_cache_read_tokens"`
		PromptCacheWriteTokens int       `json:"prompt_cache_write_tokens"`
		StatusCode             int       `json:"status_code"`
		Latency                int       `json:"latency"`
		CreatedAt              time.Time `json:"created_at"`
	}
	rows := make([]usageLogRow, len(logs))
	for i, log := range logs {
		rows[i] = usageLogRow{
			ID:                     log.ID,
			UserName:               userNames[log.UserID],
			Provider:               log.Provider,
			Model:                  log.Model,
			PromptTokens:           log.PromptTokens,
			CompletionTokens:       log.CompletionTokens,
			TotalTokens:            log.TotalTokens,
			PromptCacheReadTokens:  log.PromptCacheReadTokens,
			PromptCacheWriteTokens: log.PromptCacheWriteTokens,
			StatusCode:             log.StatusCode,
			Latency:                log.Latency,
			CreatedAt:              log.CreatedAt,
		}
	}

	jsonOK(w, map[string]interface{}{
		"start_date": start.Format("2006-01-02"),
		"end_date":   end.Format("2006-01-02"),
		"page":       page,
		"page_size":  pageSize,
		"total":      total,
		"logs":       rows,
	})
}

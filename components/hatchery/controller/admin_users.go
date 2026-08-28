package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"hatchery/controller/usergroup"
	"hatchery/i18n"

	hcommon "hatchery/common"
	"hatchery/model"

	"gorm.io/gorm"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"golang.org/x/crypto/bcrypt"
)

// userWithDept wraps a User with its OneID department info for JSON output.
type userWithDept struct {
	model.User
	Department     string         `json:"department"`                // 主部门名称（兼容旧字段）
	Departments    []deptWithPath `json:"departments,omitempty"`     // 完整部门列表（含层级 + 每项 department_path 全路径）
	DepartmentPath string         `json:"department_path,omitempty"` // 主部门完整路径，如 "公司/技术部/后端组"
}

// deptWithPath 在 model.OneIDDepartment 基础上附加 department_path。
// department_path 按 department_id 沿 parent 链从 oneid_departments 全局映射中反推，
// 兼容旧字段结构（内嵌 OneIDDepartment，其 json tag 原样生效）。
type deptWithPath struct {
	model.OneIDDepartment
	DepartmentPath string `json:"department_path"`
}

// flexUintSlice 兼容 JSON 中整数数组（[1,2]）和字符串数组（["1","2"]）两种格式的 uint 切片。
type flexUintSlice []uint

func (f *flexUintSlice) UnmarshalJSON(data []byte) error {
	valid, _, err := parseFlexUintSlice(data)
	if err != nil {
		return err
	}
	*f = valid
	return nil
}

// parseFlexUintSlice 解析 JSON 数组为 uint 切片，兼容整数和字符串两种格式。
// 无法解析的元素会被跳过，并收集到 invalid 列表中，不影响整体解析。
// 仅当 JSON 本身不是数组时才返回 error。
func parseFlexUintSlice(data []byte) (valid []uint, invalid []string, err error) {
	var raws []json.RawMessage
	if err = json.Unmarshal(data, &raws); err != nil {
		return nil, nil, err
	}
	valid = make([]uint, 0, len(raws))
	for _, raw := range raws {
		// 尝试解析为数字
		var n uint
		if jsonErr := json.Unmarshal(raw, &n); jsonErr == nil {
			valid = append(valid, n)
			continue
		}
		// 尝试解析为字符串，再转换为数字
		var s string
		if jsonErr := json.Unmarshal(raw, &s); jsonErr != nil {
			// 既不是数字也不是字符串（如对象、数组等），跳过并记录
			invalid = append(invalid, string(raw))
			continue
		}
		v, parseErr := strconv.ParseUint(s, 10, 64)
		if parseErr != nil {
			// 字符串无法转为整数，跳过并记录
			invalid = append(invalid, s)
			continue
		}
		valid = append(valid, uint(v))
	}
	return valid, invalid, nil
}

// flexGroupIDs 用于批量导入时解析 group_ids 字段（用户组 full_path），
// 传入格式为分号分隔的用户组全路径字符串，如 "根组/研发一组;根组/研发二组" 或 "研发一组"。
type flexGroupIDs struct {
	Names []string // 解析出的用户组 full_path 列表
}

func (f *flexGroupIDs) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return hcommon.I18nError(i18n.MsgGroupIDsFormatError)
	}
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		f.Names = append(f.Names, part)
	}
	return nil
}

// userGroupBrief 用户组简要信息，用于在用户列表中展示用户所属的用户组。
type userGroupBrief struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	FullPath        string `json:"full_path"`
	Source          string `json:"source"`            // manual / oneid_dept
	IsMain          bool   `json:"is_main"`           // 仅 oneid_dept 有意义（是否组织结构主部门）；manual 恒 false
	CreatedAt       string `json:"created_at"`        // 分组创建时间（UTC RFC3339）
	InstanceQuota   int    `json:"instance_quota"`    // 该组解析后的 Agent 数量上限
	TokenQuotaDay   int    `json:"token_quota_day"`   // 该组解析后的每日 Token 上限
	TokenQuotaRules string `json:"token_quota_rules"` // 该组解析后的 Token 配额规则
	InstanceCount   int64  `json:"instance_count"`    // 🆕 v6.13：该用户在该分组下直属创建的 agent 数量（instances.user_id+group_id 命中）
}

// userProjectBrief 是用户项目页展示所需的最小项目关系；按加入时间升序排列。
type userProjectBrief struct {
	ID       uint      `json:"id"`
	Name     string    `json:"name"`
	JoinedAt time.Time `json:"joined_at"`
}

// userAdminJSON 是管理端用户列表的 JSON 输出结构，在 userWithDept 基础上补充
// API Token 状态字段（User 模型中这些字段标记了 json:"-"，不会自动序列化）。
type userAdminJSON struct {
	userWithDept
	HasAPIToken      bool               `json:"has_api_token"`
	APITokenDisabled bool               `json:"api_token_disabled"`
	Groups           []userGroupBrief   `json:"groups"`
	Projects         []userProjectBrief `json:"projects"`
}

// toAdminJSON 将 userWithDept 切片转换为管理端 JSON 输出格式。
// 会批量查询每个用户所属的用户组，填充 Groups 字段（id + name + full_path + source + is_main）。
func toAdminJSON(ctx context.Context, users []userWithDept) ([]userAdminJSON, error) {
	// 收集所有 user_id，批量查询用户组，避免 N+1
	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	groupsByUser, err := model.GetUserGroupsByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryUserGroupInfoFailed).WithDetail(err.Error())
	}
	projectsByUser, err := getUserProjectsByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// 批量查成员行，取 is_main 标记（仅 oneid_dept 源有真实值，manual 恒 false）。
	// 组合 key = userID<<32 | groupID，避免逐行 hash 分配。
	isMainBy := map[uint]map[uint]bool{}
	if len(userIDs) > 0 {
		var members []model.UserGroupMember
		if err := model.DB(ctx).
			Select("user_id, user_group_id, is_main").
			Where("user_id IN ?", userIDs).
			Find(&members).Error; err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgQueryMemberRelationFailed).WithDetail(err.Error())
		}
		for _, m := range members {
			if isMainBy[m.UserID] == nil {
				isMainBy[m.UserID] = map[uint]bool{}
			}
			if m.IsMain {
				isMainBy[m.UserID][m.UserGroupID] = true
			}
		}
	}

	// 批量获取每个组的有效策略值（instance_quota / token_quota_day）
	siteConfig := model.GetSiteConfig(ctx)
	allGroupIDs := make(map[uint]struct{})
	for _, groups := range groupsByUser {
		for _, g := range groups {
			allGroupIDs[g.ID] = struct{}{}
		}
	}
	// 预计算每个 group 的策略值（通过祖先链解析）
	type groupPolicy struct {
		InstanceQuota   int
		TokenQuotaDay   int
		TokenQuotaRules string
	}
	groupPolicies := make(map[uint]groupPolicy, len(allGroupIDs))
	for gid := range allGroupIDs {
		rulesJSON := effectiveGroupTokenQuotaRulesJSON(ctx, gid, siteConfig)
		tokenQuotaDay := model.EffectiveTokenQuotaDay(
			-1,
			rulesJSON,
		)
		groupPolicies[gid] = groupPolicy{
			InstanceQuota:   usergroup.ResolvePolicyIntForGroup(ctx, usergroup.PolicyKeyInstanceQuota, gid, siteConfig.DefaultInstanceQuota),
			TokenQuotaDay:   tokenQuotaDay,
			TokenQuotaRules: rulesJSON,
		}
	}

	// 🆕 批量查每个 (user_id, group_id) 对的实例数量（一次聚合 SQL,避免 N+1）。
	// 只统计 group_id != 0 的实例；group_id=0 表示"未指定分组"，不归入任何 groups[] 子项。
	// ⚠️ 必须用 Model(&model.Instance{}) 而不是 Table("instances")：前者会让 GORM
	//    自动注入 deleted_at IS NULL 过滤，已销毁/软删的实例不计入；后者会绕开。
	instCountBy := map[uint]map[uint]int64{}
	if len(userIDs) > 0 {
		type instCountRow struct {
			UserID  uint  `gorm:"column:user_id"`
			GroupID uint  `gorm:"column:group_id"`
			Count   int64 `gorm:"column:count"`
		}
		var rows []instCountRow
		if err := model.DB(ctx).Model(&model.Instance{}).
			Select("user_id, group_id, COUNT(*) AS count").
			Where("user_id IN ? AND group_id <> 0 AND is_doctor_node = ? AND source != ?", userIDs, false, model.InstanceSourceLocal).
			Group("user_id, group_id").
			Scan(&rows).Error; err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgQueryUserGroupInstanceCountFailed).WithDetail(err.Error())
		}
		for _, r := range rows {
			if instCountBy[r.UserID] == nil {
				instCountBy[r.UserID] = map[uint]int64{}
			}
			instCountBy[r.UserID][r.GroupID] = r.Count
		}
	}

	out := make([]userAdminJSON, len(users))
	for i, u := range users {
		displayUser := u
		displayUser.TokenQuotaRules = effectiveAdminUserTokenQuotaRulesJSON(u.User)
		groups := groupsByUser[u.ID]
		briefs := make([]userGroupBrief, len(groups))
		for j, g := range groups {
			// manual 分组不暴露 is_main（无此概念），保持 false；oneid_dept 才读出成员行标记
			isMain := false
			if g.Source == model.GroupSourceOneIDDept {
				isMain = isMainBy[u.ID][g.ID]
			}
			gp := groupPolicies[g.ID]
			var instCount int64
			if m := instCountBy[u.ID]; m != nil {
				instCount = m[g.ID]
			}
			briefs[j] = userGroupBrief{
				ID:              g.ID,
				Name:            g.Name,
				FullPath:        g.FullPath,
				Source:          g.Source,
				IsMain:          isMain,
				CreatedAt:       g.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
				InstanceQuota:   gp.InstanceQuota,
				TokenQuotaDay:   gp.TokenQuotaDay,
				TokenQuotaRules: gp.TokenQuotaRules,
				InstanceCount:   instCount,
			}
		}
		// 排序：oneid_dept 优先 → 主部门优先 → 层级浅到深 → 同层级按创建时间
		sort.Slice(briefs, func(a, b int) bool {
			// source 优先级：oneid_dept > manual
			if briefs[a].Source != briefs[b].Source {
				return briefs[a].Source == model.GroupSourceOneIDDept
			}
			// 主部门优先
			if briefs[a].IsMain != briefs[b].IsMain {
				return briefs[a].IsMain
			}
			// full_path 层级浅到深
			depthA := strings.Count(briefs[a].FullPath, "/")
			depthB := strings.Count(briefs[b].FullPath, "/")
			if depthA != depthB {
				return depthA < depthB
			}
			// 同层级按创建时间
			return briefs[a].CreatedAt < briefs[b].CreatedAt
		})
		out[i] = userAdminJSON{
			userWithDept:     displayUser,
			HasAPIToken:      u.HasAPIToken(),
			APITokenDisabled: u.APITokenDisabled,
			Groups:           briefs,
			Projects:         projectsByUser[u.ID],
		}
		// 兼容旧客户端：TokenQuotaDay 展示从 rules 反推的有效值
		out[i].TokenQuotaDay = model.EffectiveTokenQuotaDay(u.TokenQuotaDay, displayUser.TokenQuotaRules)
	}
	return out, nil
}

func getUserProjectsByUserIDs(ctx context.Context, userIDs []uint) (map[uint][]userProjectBrief, error) {
	result := make(map[uint][]userProjectBrief, len(userIDs))
	for _, userID := range userIDs {
		result[userID] = []userProjectBrief{}
	}
	if len(userIDs) == 0 {
		return result, nil
	}
	var members []model.ProjectMember
	if err := model.DB(ctx).Where("user_id IN ?", userIDs).Order("created_at ASC, id ASC").Find(&members).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOperationFailed)
	}
	projectIDs := make([]uint, 0, len(members))
	for _, member := range members {
		projectIDs = append(projectIDs, member.ProjectID)
	}
	var projects []model.Project
	if len(projectIDs) > 0 {
		if err := model.DB(ctx).Where("id IN ?", uniqueUintIDs(projectIDs)).Find(&projects).Error; err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}
	}
	projectByID := make(map[uint]model.Project, len(projects))
	for _, project := range projects {
		projectByID[project.ID] = project
	}
	for _, member := range members {
		if project, ok := projectByID[member.ProjectID]; ok {
			result[member.UserID] = append(result[member.UserID], userProjectBrief{ID: project.ID, Name: project.Name, JoinedAt: member.CreatedAt})
		}
	}
	return result, nil
}

func effectiveAdminUserTokenQuotaRulesJSON(user model.User) string {
	if rules, ok := model.ParseTokenQuotaRules(user.TokenQuotaRules); ok {
		return model.MarshalTokenQuotaRules(rules)
	}
	rules := user.ResolvedTokenQuotaRules()
	return model.MarshalTokenQuotaRules(rules)
}

func effectiveGroupTokenQuotaRulesJSON(ctx context.Context, groupID uint, siteConfig model.SiteConfig) string {
	fallbackRules := model.MarshalTokenQuotaRules(siteConfig.ResolvedDefaultTokenQuotaRules())
	rulesJSON, _ := usergroup.ResolveTokenQuotaRulesForGroup(ctx, groupID, fallbackRules, -1)
	return rulesJSON
}

// buildDepartmentPath 根据全局部门映射和目标部门 ID，沿 parent_id 向上递归构建
// 完整路径，如 "公司/技术部/后端组"。如果父部门不在映射中则停止。
func buildDepartmentPath(globalDeptMap map[string]model.OneIDDepartment, targetDeptID string) string {
	if len(globalDeptMap) == 0 || targetDeptID == "" {
		return ""
	}

	// 从目标部门向上遍历，收集名称
	var parts []string
	visited := make(map[string]bool) // 防止循环引用
	cur := targetDeptID
	for cur != "" && !visited[cur] {
		visited[cur] = true
		d, ok := globalDeptMap[cur]
		if !ok {
			break
		}
		parts = append(parts, d.DepartmentName)
		cur = d.DepartmentParentID
	}

	// 反转为从根到叶的顺序
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "/")
}

func queryUsers(ctx context.Context, page, pageSize int, username string, fuzzy bool, departmentID, role, hasPersonalSpace string, groupIDs []uint, ungrouped bool) ([]userWithDept, int64) {
	query := model.DB(ctx).Unscoped().Model(&model.User{})
	if username != "" {
		if fuzzy {
			query = query.Where("users.username LIKE ?", "%"+username+"%")
		} else {
			query = query.Where("users.username = ?", username)
		}
	}
	if role != "" {
		query = query.Where("users.role = ?", role)
	}
	if hasPersonalSpace == "1" {
		query = query.Where("users.id IN (?)", model.DB(ctx).Model(&model.SMHPersonalSpace{}).Where("to_be_deleted_at IS NULL").Select("user_id"))
	} else if hasPersonalSpace == "0" {
		query = query.Where("users.id NOT IN (?)", model.DB(ctx).Model(&model.SMHPersonalSpace{}).Where("to_be_deleted_at IS NULL").Select("user_id"))
	}
	if departmentID != "" {
		// Filter by department_id: 查找 DepartmentsJSON 中包含该部门 ID 的用户
		// 同时兼容按 main_dept_id 精确匹配
		query = query.Where("users.one_id_sub IN (?)",
			model.DB(ctx).Model(&model.OneIDUserProfile{}).Select("one_id_sub").Where(
				"main_dept_id = ? OR departments_json LIKE ?", departmentID, model.DeptIDLikePattern(departmentID),
			),
		)
	}
	// ungrouped 优先于 groupIDs：只返回未加入任何用户组的用户（含已禁用用户）
	if ungrouped {
		query = query.Where("users.id NOT IN (?)", model.DB(ctx).Model(&model.UserGroupMember{}).Select("DISTINCT user_id"))
	} else if len(groupIDs) > 0 {
		// 返回属于指定用户组中任意一个的用户（OR 语义）
		query = query.Where("users.id IN (?)", model.DB(ctx).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs))
	}

	var total int64
	query.Count(&total)

	var users []model.User
	query.Order("users.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users)

	// 用共享 helper 批量补 OneID 部门信息（无 OneID 用户时短路，零额外 DB 调用）。
	deptInfo := enrichUserDepartments(ctx, users)
	result := make([]userWithDept, len(users))
	for i, u := range users {
		result[i] = userWithDept{User: u}
		if d, ok := deptInfo[u.ID]; ok {
			result[i].Department = d.Department
			result[i].Departments = d.Departments
			result[i].DepartmentPath = d.DepartmentPath
		}
	}
	return result, total
}

// errUsernameExists 是用户名重复时返回的哨兵错误，用于替代字符串匹配。
var errUsernameExists = hcommon.I18nError(i18n.MsgUsernameExists)

// errMsgOneIDReadonlyUserOp 是 OneID 模式下拒绝管控端写「用户 ↔ OneID 部门」
// 关系或新建/导入用户时返回的错误文案；统一字符串便于前端识别与单测断言。
const errMsgOneIDReadonlyUserOp = "OneID 模式下不允许在管控端修改用户与 OneID 同步分组的关系，请到 OneID 系统操作后等待同步"

func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	page, pageSize := parsePagination(r)
	username := r.URL.Query().Get("username")
	departmentID := r.URL.Query().Get("department_id")
	if departmentID == "" {
		departmentID = r.URL.Query().Get("department") // 兼容旧参数
	}
	role := r.URL.Query().Get("role")

	fuzzy := r.URL.Query().Get("fuzzy") == "1"
	hasPersonalSpace := r.URL.Query().Get("has_personal_space")
	// 解析 group_ids 过滤参数（英文逗号分隔的用户组 ID 列表）
	var groupIDs []uint
	if groupIDsStr := r.URL.Query().Get("group_ids"); groupIDsStr != "" {
		for _, part := range strings.Split(groupIDsStr, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if gid, err := strconv.ParseUint(part, 10, 64); err == nil && gid > 0 {
				groupIDs = append(groupIDs, uint(gid))
			}
		}
	}
	ungrouped := r.URL.Query().Get("ungrouped") == "1" || r.URL.Query().Get("ungrouped") == "true"

	usersWithDept, total := queryUsers(r.Context(), page, pageSize, username, fuzzy, departmentID, role, hasPersonalSpace, groupIDs, ungrouped)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	adminUsers, err := toAdminJSON(r.Context(), usersWithDept)
	if err != nil {
		slog.Error("toAdminJSON 失败", "err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	jsonOK(w, map[string]interface{}{
		"users":       adminUsers,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

// createUserParams holds the input for creating a single user.
type createUserParams struct {
	Username           string
	Password           string
	Email              string
	Role               string
	InstanceQuota      *int    // nil = use default
	TokenQuotaDay      *int    // nil = use default (legacy)
	TokenQuotaRulesRaw *string // nil = use default; JSON string of rules array
}

type batchCreateUserItem struct {
	Username        string          `json:"username"`
	Password        string          `json:"password"`
	Role            string          `json:"role"`
	Email           string          `json:"email"`
	InstanceQuota   *int            `json:"instance_quota"`
	TokenQuotaDay   *int            `json:"token_quota_day"`
	TokenQuotaRules json.RawMessage `json:"token_quota_rules"`
	GroupIDs        *flexGroupIDs   `json:"group_ids"`
}

func decodeBatchCreateUserItem(raw json.RawMessage) (batchCreateUserItem, error) {
	var item batchCreateUserItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	return item, nil
}

func batchCreateItemUsername(raw json.RawMessage) string {
	var item struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return ""
	}
	return item.Username
}

func batchCreateItemDecodeErrorMessage(ctx context.Context, err error) string {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		switch typeErr.Field {
		case "username":
			return i18n.T(ctx, i18n.MsgBatchUsernameMustBeString)
		case "password":
			return i18n.T(ctx, i18n.MsgBatchPasswordMustBeString)
		case "role":
			return i18n.T(ctx, i18n.MsgBatchRoleMustBeString)
		case "email":
			return i18n.T(ctx, i18n.MsgBatchEmailMustBeString)
		case "instance_quota":
			return i18n.T(ctx, i18n.MsgInstanceQuotaDetailed)
		case "token_quota_day":
			return i18n.T(ctx, i18n.MsgTokenQuotaMustBeValid)
		case "group_ids":
			return i18n.T(ctx, i18n.MsgGroupIDsFormatError)
		default:
			return i18n.T(ctx, i18n.MsgBatchFieldFormatError)
		}
	}
	if strings.Contains(err.Error(), "group_ids") {
		return i18n.T(ctx, i18n.MsgGroupIDsFormatError)
	}
	return i18n.T(ctx, i18n.MsgBatchFieldFormatError)
}

func createUserTokenQuotaFallbackRule(cfg model.SiteConfig) model.TokenQuotaRule {
	if rules, ok := model.ParseTokenQuotaRules(cfg.DefaultTokenQuotaRules); ok && len(rules) > 0 {
		rule := rules[0]
		rule.Start = nil
		return rule
	}
	return model.TokenQuotaRule{Mode: model.QuotaModeDay, Limit: cfg.DefaultTokenQuotaDay}
}

func normalizeCreateUserTokenQuotaRules(raw string, fallback model.TokenQuotaRule) (string, error) {
	var rawRules []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &rawRules); err != nil {
		return model.NormalizeTokenQuotaRules(raw)
	}

	fallback.Start = nil

	rules := make([]model.TokenQuotaRule, 0, len(rawRules))
	for _, rawRule := range rawRules {
		rule := fallback

		if rawMode, ok := rawRule["mode"]; ok && string(rawMode) != "null" {
			if err := json.Unmarshal(rawMode, &rule.Mode); err != nil {
				return model.NormalizeTokenQuotaRules(raw)
			}
		}
		if rawLimit, ok := rawRule["limit"]; ok && string(rawLimit) != "null" {
			if err := json.Unmarshal(rawLimit, &rule.Limit); err != nil {
				return model.NormalizeTokenQuotaRules(raw)
			}
		}

		if rule.Mode == model.QuotaModeCustom {
			if rawStart, ok := rawRule["start"]; ok && string(rawStart) != "null" {
				var start int64
				if err := json.Unmarshal(rawStart, &start); err != nil {
					return model.NormalizeTokenQuotaRules(raw)
				}
				rule.Start = &start
			}
			if rawEnd, ok := rawRule["end"]; ok && string(rawEnd) != "null" {
				var end int64
				if err := json.Unmarshal(rawEnd, &end); err != nil {
					return model.NormalizeTokenQuotaRules(raw)
				}
				rule.End = &end
			}
			if rawRefresh, ok := rawRule["refresh"]; ok && string(rawRefresh) != "null" {
				if err := json.Unmarshal(rawRefresh, &rule.Refresh); err != nil {
					return model.NormalizeTokenQuotaRules(raw)
				}
			}
		} else {
			rule.Start = nil
			rule.End = nil
			rule.Refresh = ""
		}
		rules = append(rules, rule)
	}

	normalizedRaw, err := json.Marshal(rules)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgFailedToMarshalJSON)
	}
	return model.NormalizeTokenQuotaRules(string(normalizedRaw))
}

// createUserPrepared 校验参数并返回准备好的 User 对象（含哈希密码），不写库。
// 返回 (user, httpStatus, error)，成功时 httpStatus=0。
func createUserPrepared(ctx context.Context, p createUserParams) (*model.User, int, error) {
	isUnified := hcommon.IsUnifiedAccountMode(ctx)

	if p.Username == "" {
		return nil, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDUsernameNotEmpty)
	}
	// 统一账号模式下密码由 OneID 管理，本地不存储；非统一模式必须传密码
	if !isUnified && p.Password == "" {
		return nil, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPasswordCannotBeEmpty)
	}

	if hcommon.UserLimitFromCtx(ctx) > 0 {
		var count int64
		model.DB(ctx).Unscoped().Model(&model.User{}).Count(&count)
		if count >= int64(hcommon.UserLimitFromCtx(ctx)) {
			return nil, http.StatusForbidden, hcommon.I18nError(i18n.MsgUserLimitReached, hcommon.UserLimitFromCtx(ctx))
		}
	}

	role := p.Role
	if role != "admin" && role != "user" {
		role = "user"
	}

	var passwordHash string
	if !isUnified && p.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgPasswordEncryptFailed)
		}
		passwordHash = string(hash)
	}

	cfg := model.GetSiteConfig(ctx)
	user := &model.User{
		Username:        p.Username,
		Password:        passwordHash,
		Role:            role,
		InstanceQuota:   cfg.DefaultInstanceQuota,
		TokenQuotaDay:   cfg.DefaultTokenQuotaDay,
		TokenQuotaRules: cfg.DefaultTokenQuotaRules, // 原样复制，不转换
	}

	if p.InstanceQuota != nil {
		q := *p.InstanceQuota
		if (q < -1) || q > 999 {
			return nil, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceQuotaDetailed)
		}
		user.InstanceQuota = q
	}
	// 显式传入配额时走 rules 路径（写时迁移）
	if p.TokenQuotaRulesRaw != nil {
		// 显式传入 rules → 校验 + normalize（FillCustomStartIfEmpty 自动填空 start）
		normalized, err := normalizeCreateUserTokenQuotaRules(*p.TokenQuotaRulesRaw, createUserTokenQuotaFallbackRule(cfg))
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		user.TokenQuotaRules = normalized
		user.TokenQuotaDay = -1
	} else if p.TokenQuotaDay != nil {
		q := *p.TokenQuotaDay
		if q < -1 {
			return nil, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTokenQuotaMustBeValid)
		}
		if q == -1 {
			// -1 = 无限制，不创建 day 规则，设为显式空
			user.TokenQuotaRules = "[]"
		} else {
			user.TokenQuotaRules = model.MarshalTokenQuotaRules([]model.TokenQuotaRule{{Mode: model.QuotaModeDay, Limit: q}})
		}
		user.TokenQuotaDay = -1
	} else if user.TokenQuotaRules != "" {
		// 默认规则从 SiteConfig 复制而来，custom start 强制覆盖为创建时间
		user.TokenQuotaRules = model.StampTokenQuotaRulesStart(user.TokenQuotaRules)
	}
	return user, 0, nil
}

// createOneUserTx 在给定事务 tx 内写入用户记录，返回带 ID 的完整用户对象。
// 调用方负责事务的提交/回滚。
func createOneUserTx(tx *gorm.DB, user *model.User) (*model.User, error) {
	// 创建前先生成 API Token，然后一次性插入数据库
	token, err := model.GenerateRandomToken()
	if err != nil {
		slog.Error("为新用户生成 API Token 失败", "username", user.Username, "err", err)
		return nil, hcommon.I18nError(i18n.MsgCreateUserGenTokenFailed)
	}

	// 使用 map 创建，避免 GORM 对 int 零值的跳过行为（Create 遇到零值会使用 DB default 而非写入 0）
	// map 方式不会自动填充 created_at/updated_at，需手动传入
	now := time.Now()
	createMap := map[string]interface{}{
		"username":             user.Username,
		"password":             user.Password,
		"role":                 user.Role,
		"instance_quota":       user.InstanceQuota,
		"token_quota_day":      user.TokenQuotaDay,
		"token_quota_rules":    user.TokenQuotaRules,
		"api_token":            token,
		"api_token_disabled":   false,
		"api_token_created_at": now,
		"created_at":           now,
		"updated_at":           now,
	}
	// 统一账号模式：写入 one_id_sub (union_id) 和 oneid_login_name
	if user.OneIDSub != nil && *user.OneIDSub != "" {
		createMap["one_id_sub"] = *user.OneIDSub
	}
	if user.OneIDLoginName != nil && *user.OneIDLoginName != "" {
		createMap["oneid_login_name"] = *user.OneIDLoginName
	}
	createResult := tx.Model(&model.User{}).Create(createMap)
	if dbErr := createResult.Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrDuplicatedKey) || isDuplicateKeyError(dbErr) {
			return nil, errUsernameExists
		}
		slog.Error("创建用户写入数据库失败", "username", user.Username, "err", dbErr)
		return nil, hcommon.I18nError(i18n.MsgCreateUserDBError)
	}

	// 获取新创建用户的完整对象（含自增 ID）
	var newUser model.User
	if err := tx.Where("username = ?", user.Username).First(&newUser).Error; err != nil {
		slog.Error("创建用户后回查记录失败", "username", user.Username, "err", err)
		return nil, hcommon.I18nError(i18n.MsgCreateUserReadFailed)
	}
	return &newUser, nil
}

// createOneUser validates params, creates the user in DB, and sends a welcome
// email when an address is provided. It returns (*User, 0, nil) on success, or
// (nil, httpStatus, error) on failure.
func createOneUser(ctx context.Context, p createUserParams) (*model.User, int, error) {
	user, status, err := createUserPrepared(ctx, p)
	if err != nil {
		return nil, status, err
	}

	newUser, dbErr := createOneUserTx(model.DB(ctx), user)
	if dbErr != nil {
		if errors.Is(dbErr, errUsernameExists) {
			return nil, http.StatusConflict, dbErr
		}
		return nil, http.StatusInternalServerError, dbErr
	}

	if p.Email != "" {
		if err := sendEmail(ctx, p.Email, emailTypeWelcome, CVMRegion, EmailAPIURL, map[string]any{"password": p.Password, "user": p.Username}); err != nil {
			slog.Warn("发送欢迎邮件失败", "user", p.Username, "err", err)
		}
	}
	return newUser, 0, nil
}

func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	// OneID 模式（TenantID != ""）下禁止管控端手工创建用户：
	// 用户必须由 OneID 同步流程落库，避免人工建出的本地账号绕过组织架构。
	if hcommon.TenantIDFromCtx(r.Context()) != "" && !hcommon.IsUnifiedAccountMode(r.Context()) {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgOneIDReadonlyUserOp))
		return
	}

	p := createUserParams{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
		Email:    r.FormValue("email"),
		Role:     r.FormValue("role"),
	}
	if qStr := r.FormValue("instance_quota"); qStr != "" {
		q, err := strconv.Atoi(qStr)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceQuotaMustBeInteger))
			return
		}
		p.InstanceQuota = &q
	}
	if qStr := r.FormValue("token_quota_day"); qStr != "" {
		q, err := strconv.Atoi(qStr)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTokenQuotaMustBeValid))
			return
		}
		p.TokenQuotaDay = &q
	}

	// 解析 group_ids / token_quota_rules（JSON 请求体中支持）
	var groupIDs *[]uint
	var body struct {
		Username        string          `json:"username"`
		Password        string          `json:"password"`
		Email           string          `json:"email"`
		Role            string          `json:"role"`
		InstanceQuota   *int            `json:"instance_quota"`
		TokenQuotaDay   *int            `json:"token_quota_day"`
		TokenQuotaRules json.RawMessage `json:"token_quota_rules"`
		GroupIDs        *flexUintSlice  `json:"group_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		if body.Username != "" {
			p.Username = body.Username
		}
		if body.Password != "" {
			p.Password = body.Password
		}
		if body.Email != "" {
			p.Email = body.Email
		}
		if body.Role != "" {
			p.Role = body.Role
		}
		if body.InstanceQuota != nil {
			p.InstanceQuota = body.InstanceQuota
		}
		if body.TokenQuotaDay != nil {
			p.TokenQuotaDay = body.TokenQuotaDay
		}
		if len(body.TokenQuotaRules) > 0 && string(body.TokenQuotaRules) != "null" {
			raw := string(body.TokenQuotaRules)
			p.TokenQuotaRulesRaw = &raw
		}
		if body.GroupIDs != nil {
			ids := []uint(*body.GroupIDs)
			// 🆕 v6.13：允许用户同时属于多个分组。
			groupIDs = &ids
		}
	}

	// 校验参数（不写库）
	prepared, status, err := createUserPrepared(r.Context(), p)
	if err != nil {
		writeError(w, r, status, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 统一账号模式：先调 OneID 创建用户 + 角色绑定
	var oneIDUnionID string
	var oneIDLoginName string
	if hcommon.IsUnifiedAccountMode(r.Context()) {
		// 解析 group_ids 对应的 OneID department_ids
		var deptIDs []string
		if groupIDs != nil && len(*groupIDs) > 0 {
			resolved, resolveErr := oneIDResolveDepartmentIDsForGroups(r.Context(), *groupIDs)
			if resolveErr != nil {
				slog.Warn("[UnifiedAccount] resolve department_ids failed, fallback to root dept", "err", resolveErr)
			} else if len(resolved) > 0 {
				deptIDs = resolved
			}
		}

		// 确定 OneID 登录名：如果本地用户名符合规范则一致，否则生成随机登录名
		oneIDLoginName = p.Username
		if validateOneIDUsername(p.Username) != nil {
			oneIDLoginName = generateRandomLoginName()
		}

		resp, oneIDErr := OneIDCreateUser(r.Context(), OneIDCreateUserReq{
			Name:          p.Username,
			Username:      oneIDLoginName,
			Email:         p.Email,
			Password:      p.Password,
			DepartmentIDs: deptIDs,
		})
		if oneIDErr != nil {
			slog.Error("[UnifiedAccount] OneID create user failed", "username", p.Username, "err", oneIDErr)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDCreateUserFailed, oneIDErr))
			return
		}
		oneIDUnionID = resp.UnionID
		slog.Info("[UnifiedAccount] OneID user created", "username", p.Username, "union_id", oneIDUnionID)

		// admin 角色绑定
		if p.Role == "admin" {
			if roleErr := OneIDAddRoleUsers(r.Context(), []string{oneIDUnionID}); roleErr != nil {
				slog.Error("[UnifiedAccount] OneID add role failed", "union_id", oneIDUnionID, "err", roleErr)
				writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDAddRoleFailed, roleErr))
				return
			}
		}

		// 注入 one_id_sub 和 oneid_login_name 到 prepared user
		prepared.OneIDSub = &oneIDUnionID
		prepared.OneIDLoginName = &oneIDLoginName
	}

	// 将用户创建与用户组关联放在同一事务中，保证原子性
	var newUser *model.User
	if txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		newUser, err = createOneUserTx(tx, prepared)
		if err != nil {
			return err
		}
		if groupIDs != nil {
			if err := model.UpdateUserGroupMemberships(tx, newUser.ID, *groupIDs); err != nil {
				return hcommon.I18nRichError(err, i18n.MsgUserGroupMembershipSetFailed).WithDetail(err.Error())
			}
		}
		return nil
	}); txErr != nil {
		httpStatus := http.StatusInternalServerError
		if errors.Is(txErr, errUsernameExists) {
			httpStatus = http.StatusConflict
		} else if errors.Is(txErr, model.ErrInvalidUserGroupID) || errors.Is(txErr, model.ErrGroupMemberLimitReached) {
			httpStatus = http.StatusBadRequest
		}
		writeError(w, r, httpStatus, hcommon.EnsureRichErrorOrPanic(txErr))
		return
	}

	if p.Email != "" {
		if err := sendEmail(r.Context(), p.Email, emailTypeWelcome, CVMRegion, EmailAPIURL, map[string]any{"password": p.Password, "user": p.Username, "login_url": emailLoginURLForRequest(r)}); err != nil {
			slog.Warn("发送欢迎邮件失败", "user", p.Username, "err", err)
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true, "id": newUser.ID})
}

func HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var user model.User
	if model.DB(r.Context()).Where("id = ?", id).First(&user).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
		return
	}
	if user.IsInitialAdmin(r.Context()) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgCannotDeleteInitialAdmin))
		return
	}

	if err := stopUserInstances(r.Context(), user.ID); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgShutdownFailed))
		return
	}

	// 统一账号模式：先调 OneID 停用用户
	if hcommon.IsUnifiedAccountMode(r.Context()) && user.OneIDSub != nil && *user.OneIDSub != "" {
		if err := OneIDDisableUser(r.Context(), []string{*user.OneIDSub}); err != nil {
			slog.Error("[UnifiedAccount] OneID disable user failed", "union_id", *user.OneIDSub, "err", err)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDDisableUserFailed, err))
			return
		}
		slog.Info("[UnifiedAccount] OneID user disabled", "union_id", *user.OneIDSub)
	}

	// 软删除用户，保留用户组成员关系（禁用不影响组绑定）
	if err := model.DB(r.Context()).Delete(&user).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDisableUserFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// tryHardDeleteUser 检查用户是否无任何资源占用（实例、VPC），若满足则物理删除并返回 true。
// 任一资源仍存在则不做任何操作，返回 false。不会主动删除实例或 VPC 等资源。
// 调用方可根据返回值决定是否回退到软删除。
func tryHardDeleteUser(ctx context.Context, user *model.User) bool {
	// 检查实例
	var instanceCount int64
	if err := model.DB(ctx).Model(&model.Instance{}).Where("user_id = ?", user.ID).Count(&instanceCount).Error; err != nil {
		slog.Error("tryHardDeleteUser: failed to count instances", "user_id", user.ID, "err", err)
		return false
	}
	if instanceCount > 0 {
		return false
	}

	// 检查 VPC 资源占用
	if user.VpcId != "" {
		hasResources, err := vpcHasResources(ctx, user.VpcId)
		if err != nil {
			slog.Error("tryHardDeleteUser: failed to check VPC resources", "user_id", user.ID, "vpc_id", user.VpcId, "err", err)
			return false
		}
		if hasResources {
			return false
		}
	}

	// 无任何资源，物理删除
	if err := model.DB(ctx).Unscoped().Delete(user).Error; err != nil {
		slog.Error("tryHardDeleteUser: failed to hard-delete user", "user_id", user.ID, "err", err)
		return false
	}
	return true
}

func HandleHardDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var user model.User
	if model.DB(r.Context()).Unscoped().Where("id = ?", id).First(&user).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
		return
	}
	if user.IsInitialAdmin(r.Context()) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgCannotDeleteInitialAdmin))
		return
	}

	// 前置检查：给出明确的错误信息（API 场景需要告知原因）
	var instanceCount int64
	model.DB(r.Context()).Model(&model.Instance{}).Where("user_id = ?", user.ID).Count(&instanceCount)
	if instanceCount > 0 {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgUserHasInstancesExist))
		return
	}

	if user.VpcId != "" {
		hasResources, err := vpcHasResources(r.Context(), user.VpcId)
		if err != nil {
			slog.Error("HandleHardDeleteUser: failed to check VPC resources", "user_id", user.ID, "vpc_id", user.VpcId, "err", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		if hasResources {
			writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgUserVPCHasResources, user.VpcId))
			return
		}
		if err := deleteVPC(r.Context(), user.VpcId); err != nil {
			slog.Error("HandleHardDeleteUser: failed to delete VPC", "user_id", user.ID, "vpc_id", user.VpcId, "err", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	// 统一账号模式：先调 OneID 删除用户
	if hcommon.IsUnifiedAccountMode(r.Context()) && user.OneIDSub != nil && *user.OneIDSub != "" {
		appID := hcommon.OneIDAppIDFromCtx(r.Context())
		if err := OneIDDeleteUser(r.Context(), *user.OneIDSub, appID); err != nil {
			slog.Error("[UnifiedAccount] OneID delete user failed", "union_id", *user.OneIDSub, "err", err)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDDeleteUserFailed, err))
			return
		}
		slog.Info("[UnifiedAccount] OneID user deleted", "union_id", *user.OneIDSub)
	}

	// 硬删除用户及解绑用户组成员关系，使用事务保证原子性
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(&user).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", user.ID).Delete(&model.UserGroupMember{}).Error
	}); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDeleteUserFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleRestoreUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var user model.User
	if model.DB(r.Context()).Unscoped().Where("id = ?", id).First(&user).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
		return
	}
	if user.IsInitialAdmin(r.Context()) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgCannotOperateInitialAdmin))
		return
	}

	// 统一账号模式：先调 OneID 启用用户
	if hcommon.IsUnifiedAccountMode(r.Context()) && user.OneIDSub != nil && *user.OneIDSub != "" {
		if err := OneIDEnableUser(r.Context(), *user.OneIDSub); err != nil {
			slog.Error("[UnifiedAccount] OneID enable user failed", "union_id", *user.OneIDSub, "err", err)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDEnableUserFailed, err))
			return
		}
		slog.Info("[UnifiedAccount] OneID user enabled", "union_id", *user.OneIDSub)
	}

	if err := startUserInstances(r.Context(), user.ID); err != nil {
		slog.Warn("恢复用户后开机实例失败", "user", user.Username, "err", err)
	}

	model.DB(r.Context()).Unscoped().Model(&user).Update("deleted_at", nil)

	jsonOK(w, map[string]interface{}{"ok": true})
}

// resolveResetPasswordUser 根据请求参数定位要重置密码的用户。
// init_user=true 时返回初始管理员；否则按 id 查询（含软删除）。
func resolveResetPasswordUser(ctx context.Context, userID string, initUser bool) (*model.User, error) {
	if initUser {
		initialAdmin := model.GetInitialAdmin(ctx)
		if initialAdmin == nil {
			return nil, hcommon.I18nError(i18n.MsgInitialAdminNotExist)
		}
		return initialAdmin, nil
	}
	var user model.User
	if err := model.DB(ctx).Unscoped().Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, hcommon.I18nError(i18n.MsgUserNotExist)
	}
	return &user, nil
}

// resetPasswordCore 对已定位的用户执行密码哈希和落库。
func resetPasswordCore(ctx context.Context, user *model.User, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return hcommon.I18nError(i18n.MsgPasswordEncryptFailed)
	}
	model.DB(ctx).Unscoped().Model(user).Update("password", string(hash))
	return nil
}

func HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	initUser := r.URL.Query().Get("init_user") == "true"
	password := r.FormValue("password")
	email := r.FormValue("email")

	if password == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPasswordCannotBeEmpty))
		return
	}

	user, err := resolveResetPasswordUser(r.Context(), id, initUser)
	if err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 无论通过 init_user 还是通过 id 命中初始管理员，都必须走 admin-token 鉴权。
	// init_user=true 时 user 已经是初始管理员，直接短路，避免 IsInitialAdmin() 再查一次。
	if initUser || user.IsInitialAdmin(r.Context()) {
		tokenUser, _ := getUserFromToken(r)
		if tokenUser == nil || tokenUser.ID != 0 {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgInitialAdminPasswordReset))
			return
		}
	}

	// 统一账号模式：只调 Gateway 重置 OneID 密码，不写本地
	if hcommon.IsUnifiedAccountMode(r.Context()) && user.OneIDSub != nil && *user.OneIDSub != "" {
		if err := OneIDResetPassword(r.Context(), *user.OneIDSub, password); err != nil {
			slog.Error("[UnifiedAccount] OneID reset password failed", "union_id", *user.OneIDSub, "err", err)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDResetPasswordFailed, err))
			return
		}
		slog.Info("[UnifiedAccount] OneID password reset", "union_id", *user.OneIDSub)
		if email != "" {
			sendEmail(r.Context(), email, emailTypeResetPassword, CVMRegion, EmailAPIURL, map[string]any{"password": password, "user": user.Username})
		}
		jsonOK(w, map[string]interface{}{"ok": true})
		return
	}

	if err := resetPasswordCore(r.Context(), user, password); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if email != "" {
		sendEmail(r.Context(), email, emailTypeResetPassword, CVMRegion, EmailAPIURL, map[string]any{"password": password, "user": user.Username, "login_url": emailLoginURLForRequest(r)})
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var user model.User
	if model.DB(r.Context()).Where("id = ?", id).First(&user).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
		return
	}

	updates := map[string]interface{}{}
	var groupIDs *[]uint

	var body struct {
		Email           string          `json:"email"`
		Role            string          `json:"role"`
		InstanceQuota   *int            `json:"instance_quota"`
		TokenQuotaDay   *int            `json:"token_quota_day"`
		TokenQuotaRules json.RawMessage `json:"token_quota_rules"`
		GroupIDs        *flexUintSlice  `json:"group_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		if body.Email != "" {
			updates["email"] = body.Email
		}
		if body.Role == "admin" || body.Role == "user" {
			if user.IsInitialAdmin(r.Context()) && body.Role != "admin" {
				writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgCannotModifyInitialAdminRole))
				return
			}
			updates["role"] = body.Role
		}
		if body.InstanceQuota != nil {
			q := *body.InstanceQuota
			if (q < -1) || q > 999 {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceQuotaDetailed))
				return
			}
			updates["instance_quota"] = q
		}
		// token_quota_rules 优先于 token_quota_day；写时迁移
		if len(body.TokenQuotaRules) > 0 && string(body.TokenQuotaRules) != "null" {
			normalized, err := model.NormalizeTokenQuotaRules(string(body.TokenQuotaRules))
			if err != nil {
				writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
				return
			}
			updates["token_quota_rules"] = normalized
			updates["token_quota_day"] = -1
		} else if body.TokenQuotaDay != nil {
			q := *body.TokenQuotaDay
			if q < -1 {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTokenQuotaMustBeValid))
				return
			}
			// 旧接口：upsert day 规则，保留其他规则
			updates["token_quota_rules"] = model.UpsertDayRule(user.TokenQuotaRules, q)
			updates["token_quota_day"] = -1
		}
		if body.GroupIDs != nil {
			ids := []uint(*body.GroupIDs)
			// 🆕 v6.13：允许用户同时属于多个分组，UpdateUserGroupMemberships
			// 已支持批量归属（空列表 = 清空，含具体 id 则全量替换）。
			groupIDs = &ids
		}
	}

	if len(updates) == 0 && groupIDs == nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNoFieldsToUpdate))
		return
	}

	// 统一账号模式：同步可同步的字段到 OneID
	if hcommon.IsUnifiedAccountMode(r.Context()) && user.OneIDSub != nil && *user.OneIDSub != "" {
		oneIDFields := map[string]interface{}{}
		if v, ok := updates["email"]; ok {
			oneIDFields["email"] = v
		}
		// username 变更时同步 name 字段
		if v, ok := updates["username"]; ok {
			oneIDFields["name"] = v
			oneIDFields["username"] = v
		}
		if len(oneIDFields) > 0 {
			if err := OneIDUpdateUser(r.Context(), *user.OneIDSub, oneIDFields); err != nil {
				slog.Error("[UnifiedAccount] OneID update user failed", "union_id", *user.OneIDSub, "err", err)
				writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDUpdateUserFailed, err))
				return
			}
			slog.Info("[UnifiedAccount] OneID user updated", "union_id", *user.OneIDSub, "fields", oneIDFields)
		}

		// 角色变更：user → admin 时添加 OneID 角色
		if newRole, ok := updates["role"]; ok && newRole == "admin" && user.Role != "admin" {
			if err := OneIDAddRoleUsers(r.Context(), []string{*user.OneIDSub}); err != nil {
				slog.Error("[UnifiedAccount] OneID add role failed", "union_id", *user.OneIDSub, "err", err)
				writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDAddRoleFailed, err))
				return
			}
			slog.Info("[UnifiedAccount] OneID role added", "union_id", *user.OneIDSub)
		}

		// 角色变更：admin → user 时移除 OneID 角色
		if newRole, ok := updates["role"]; ok && newRole == "user" && user.Role == "admin" {
			if err := OneIDRemoveRoleUsers(r.Context(), []string{*user.OneIDSub}); err != nil {
				slog.Error("[UnifiedAccount] OneID remove role failed", "union_id", *user.OneIDSub, "err", err)
				writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDRemoveRoleFailed, err))
				return
			}
			slog.Info("[UnifiedAccount] OneID role removed", "union_id", *user.OneIDSub)
		}
	}

	// 将用户字段更新与用户组归属更新放在同一事务中，避免部分失败导致数据不一致
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&user).Updates(updates).Error; err != nil {
				return err
			}
		}
		if groupIDs != nil {
			// group_ids 仅控制用户的 manual 类型分组归属：
			//   - manual 子集按传入 ids 全量替换
			//   - oneid_dept 类型的归属由 OneID 同步独占维护，本接口不能动
			//   - 传入的 group_ids 中含 oneid_dept / to_be_deleted 的项静默忽略，不报错
			// 例：用户 ∈ {A(manual), B(oneid_dept)}，传入 [A,B,C]
			//   → manual 子集替换为 {A,C}，B 保持 → 用户 ∈ {A,B,C}
			if err := model.UpdateUserGroupMembershipsManualOnly(tx, user.ID, *groupIDs); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}

	// 统一账号模式：分组变更时同步用户的 OneID 部门归属
	if hcommon.IsUnifiedAccountMode(r.Context()) && groupIDs != nil && user.OneIDSub != nil && *user.OneIDSub != "" {
		deptIDs, resolveErr := oneIDResolveDepartmentIDsForGroups(r.Context(), *groupIDs)
		if resolveErr != nil {
			slog.Error("[UnifiedAccount] resolve department_ids for update user failed", "err", resolveErr)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDSyncDepartmentFailed, resolveErr))
			return
		}
		if len(deptIDs) > 0 {
			if err := OneIDUpdateUser(r.Context(), *user.OneIDSub, map[string]interface{}{
				"department_ids": deptIDs,
			}); err != nil {
				slog.Error("[UnifiedAccount] OneID update user departments failed",
					"union_id", *user.OneIDSub, "dept_ids", deptIDs, "err", err)
				writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDSyncUserDeptFailed, err))
				return
			}
			slog.Info("[UnifiedAccount] OneID user departments synced",
				"union_id", *user.OneIDSub, "dept_ids", deptIDs)
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleExportTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	tokens, err := model.BatchEnsureAPITokens(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAPITokenGenerateFailed))
		return
	}

	jsonOK(w, tokens)
}

// handleToggleTokenState 是管理员启用/禁用用户 API Token 的公共逻辑。
// disable=true 表示禁用，disable=false 表示启用。
func handleToggleTokenState(w http.ResponseWriter, r *http.Request, disable bool) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var user model.User
	if model.DB(r.Context()).Where("id = ?", id).First(&user).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
		return
	}
	if !user.HasAPIToken() {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserNoAPIToken))
		return
	}
	if user.APITokenDisabled == disable {
		if disable {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTokenAlreadyDisabled))
		} else {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTokenNotDisabled))
		}
		return
	}

	action := i18n.T(r.Context(), i18n.MsgTokenEnableAction)
	if disable {
		action = i18n.T(r.Context(), i18n.MsgTokenDisableAction)
	}
	if err := model.SetAPITokenDisabled(r.Context(), user.ID, disable); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgTokenOperationFailed, action))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleDisableToken 管理员禁用用户的 API Token。
//
// POST /admin/token/disable?id={userID}
func HandleDisableToken(w http.ResponseWriter, r *http.Request) {
	handleToggleTokenState(w, r, true)
}

// HandleEnableToken 管理员启用用户的 API Token。
//
// POST /admin/token/enable?id={userID}
func HandleEnableToken(w http.ResponseWriter, r *http.Request) {
	handleToggleTokenState(w, r, false)
}

func HandleBatchCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	// OneID 模式下禁止批量手工导入用户，参见 HandleCreateUser 的同样限制。
	if hcommon.TenantIDFromCtx(r.Context()) != "" && !hcommon.IsUnifiedAccountMode(r.Context()) {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgOneIDReadonlyUserOp))
		return
	}

	type result struct {
		Username  string `json:"username"`
		ID        uint   `json:"id,omitempty"`
		OK        bool   `json:"ok"`
		Error     string `json:"error,omitempty"`
		ErrorCode string `json:"error_code,omitempty"`
	}

	var rawItems []json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&rawItems); err != nil {
		// 解析失败时不弹窗，将错误写入报告返回
		errMsg := i18n.T(r.Context(), i18n.MsgBatchRequestBodyJSONError, err)
		jsonOK(w, map[string]interface{}{
			"results": []result{{
				Username:  "",
				OK:        false,
				Error:     errMsg,
				ErrorCode: "invalid_request_body",
			}},
		})
		return
	}

	if len(rawItems) == 0 || len(rawItems) > 5000 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserListEmptyOrTooLarge))
		return
	}

	if hcommon.UserLimitFromCtx(r.Context()) > 0 {
		var count int64
		model.DB(r.Context()).Unscoped().Model(&model.User{}).Count(&count)
		if count+int64(len(rawItems)) > int64(hcommon.UserLimitFromCtx(r.Context())) {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgUserLimitExceededImport, hcommon.UserLimitFromCtx(r.Context()), count))
			return
		}
	}

	results := make([]result, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, decodeErr := decodeBatchCreateUserItem(rawItem)
		if decodeErr != nil {
			results = append(results, result{
				Username:  batchCreateItemUsername(rawItem),
				OK:        false,
				Error:     batchCreateItemDecodeErrorMessage(r.Context(), decodeErr),
				ErrorCode: "invalid_params",
			})
			continue
		}
		res := result{Username: item.Username}
		p := createUserParams{
			Username:      item.Username,
			Password:      item.Password,
			Email:         item.Email,
			Role:          item.Role,
			InstanceQuota: item.InstanceQuota,
			TokenQuotaDay: item.TokenQuotaDay,
		}
		if len(item.TokenQuotaRules) > 0 && string(item.TokenQuotaRules) != "null" {
			raw := string(item.TokenQuotaRules)
			p.TokenQuotaRulesRaw = &raw
		}
		// 校验参数（不写库）
		prepared, _, prepErr := createUserPrepared(r.Context(), p)
		if prepErr != nil {
			res.Error = prepErr.Error()
			res.ErrorCode = "invalid_params"
			results = append(results, res)
			continue
		}
		// 按 full_path 查询用户组 ID（支持多层级精确匹配），任一路径不存在则该用户导入失败
		var groupIDs []uint
		if item.GroupIDs != nil && len(item.GroupIDs.Names) > 0 {
			groups, err := model.GetGroupsByFullPaths(r.Context(), item.GroupIDs.Names)
			if err != nil {
				res.Error = i18n.T(r.Context(), i18n.MsgBatchQueryGroupFailed, err)
				res.ErrorCode = "create_failed"
				results = append(results, res)
				continue
			}
			// 找出不存在的路径
			foundPaths := make(map[string]struct{}, len(groups))
			for _, g := range groups {
				foundPaths[g.FullPath] = struct{}{}
				groupIDs = append(groupIDs, g.ID)
			}
			var notFound []string
			for _, name := range item.GroupIDs.Names {
				if _, ok := foundPaths[name]; !ok {
					notFound = append(notFound, name)
				}
			}
			if len(notFound) > 0 {
				res.Error = i18n.T(r.Context(), i18n.MsgBatchGroupPathNotFound, notFound)
				res.ErrorCode = "invalid_group_names"
				results = append(results, res)
				continue
			}
		}
		// 统一账号模式：先调 OneID 创建用户 + 角色绑定
		if hcommon.IsUnifiedAccountMode(r.Context()) {
			// 解析 group_ids 对应的 OneID department_ids
			var deptIDs []string
			if len(groupIDs) > 0 {
				resolved, resolveErr := oneIDResolveDepartmentIDsForGroups(r.Context(), groupIDs)
				if resolveErr != nil {
					slog.Warn("[UnifiedAccount][Import] resolve department_ids failed, fallback to root dept", "username", item.Username, "err", resolveErr)
				} else if len(resolved) > 0 {
					deptIDs = resolved
				}
			}

			// 确定 OneID 登录名：如果本地用户名符合规范则一致，否则生成随机登录名
			oneIDLoginName := p.Username
			if validateOneIDUsername(p.Username) != nil {
				oneIDLoginName = generateRandomLoginName()
			}

			resp, oneIDErr := OneIDCreateUser(r.Context(), OneIDCreateUserReq{
				Name:          p.Username,
				Username:      oneIDLoginName,
				Email:         p.Email,
				Password:      p.Password,
				DepartmentIDs: deptIDs,
			})
			if oneIDErr != nil {
				slog.Error("[UnifiedAccount][Import] OneID create user failed", "username", p.Username, "err", oneIDErr)
				res.Error = i18n.T(r.Context(), i18n.MsgOneIDCreateUserFailed, oneIDErr)
				res.ErrorCode = "oneid_create_failed"
				results = append(results, res)
				continue
			}
			slog.Info("[UnifiedAccount][Import] OneID user created", "username", p.Username, "union_id", resp.UnionID)

			// admin 角色绑定
			if p.Role == "admin" {
				if roleErr := OneIDAddRoleUsers(r.Context(), []string{resp.UnionID}); roleErr != nil {
					slog.Error("[UnifiedAccount][Import] OneID add role failed", "union_id", resp.UnionID, "err", roleErr)
					// 角色绑定失败不阻断导入，仅记录日志
				}
			}

			// 注入 one_id_sub 和 oneid_login_name 到 prepared user
			prepared.OneIDSub = &resp.UnionID
			prepared.OneIDLoginName = &oneIDLoginName
		}

		// 将用户创建与用户组关联放在同一事务中，保证原子性
		var newUser *model.User
		txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
			var err error
			newUser, err = createOneUserTx(tx, prepared)
			if err != nil {
				return err
			}
			if len(groupIDs) > 0 {
				if err := model.UpdateUserGroupMemberships(tx, newUser.ID, groupIDs); err != nil {
					return err
				}
			}
			return nil
		})
		if txErr != nil {
			res.Error = txErr.Error()
			switch {
			case errors.Is(txErr, model.ErrGroupMemberLimitReached):
				res.ErrorCode = "group_member_limit"
				res.Error = i18n.T(r.Context(), i18n.MsgGroupMemberLimitReached)
			case errors.Is(txErr, errUsernameExists):
				res.ErrorCode = "username_exists"
			default:
				res.ErrorCode = "create_failed"
			}
			results = append(results, res)
			continue
		}
		if p.Email != "" {
			if err := sendEmail(r.Context(), p.Email, emailTypeWelcome, CVMRegion, EmailAPIURL, map[string]any{"password": p.Password, "user": p.Username, "login_url": emailLoginURLForRequest(r)}); err != nil {
				slog.Warn("发送欢迎邮件失败", "user", p.Username, "err", err)
			}
		}
		res.OK = true
		res.ID = newUser.ID
		results = append(results, res)
	}

	jsonOK(w, map[string]interface{}{"results": results})
}

// HandleAdminUserToken 管理员根据用户 ID 查询指定用户的 API Token。
// 若用户尚未创建 Token，返回 exists=false；否则返回 Token 明文及相关信息。
//
// GET /admin/user-token?id={userID}
func HandleAdminUserToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var user model.User
	if model.DB(r.Context()).Where("id = ?", id).First(&user).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
		return
	}

	if !user.HasAPIToken() {
		jsonOK(w, map[string]interface{}{
			"exists": false,
		})
		return
	}

	jsonOK(w, map[string]interface{}{
		"exists":     true,
		"token":      *user.APIToken,
		"disabled":   user.APITokenDisabled,
		"created_at": user.APITokenCreatedAt,
	})
}

// getUserInstanceIDs returns all non-empty CVM instance IDs belonging to the given user.
func getUserInstanceIDs(ctx context.Context, userID uint) ([]string, error) {
	var instances []model.Instance
	model.DB(ctx).Where("user_id = ?", userID).Find(&instances)
	var ids []string
	for _, inst := range instances {
		if inst.InstanceId != "" {
			ids = append(ids, inst.InstanceId)
		}
	}
	return ids, nil
}

// FilterInstancesByState returns the subset of ids whose current state matches the given state.
// Batches requests in groups of 100 (CVM API limit).
func FilterInstancesByState(client *cvm.Client, ids []string, state string) ([]string, error) {
	const batchSize = 100
	var filtered []string
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		descReq := cvm.NewDescribeInstancesRequest()
		descReq.InstanceIds = common.StringPtrs(ids[i:end])
		descReq.Limit = common.Int64Ptr(int64(end - i)) // 显式设置 Limit，CVM API 默认 Limit=20 会导致截断
		descResp, err := client.DescribeInstances(descReq)
		if err != nil {
			return nil, err
		}
		for _, inst := range descResp.Response.InstanceSet {
			if inst.InstanceId != nil && inst.InstanceState != nil && *inst.InstanceState == state {
				filtered = append(filtered, *inst.InstanceId)
			}
		}
	}
	return filtered, nil
}

// stopUserInstances stops all RUNNING CVM instances belonging to the given user (soft shutdown).
func stopUserInstances(ctx context.Context, userID uint) error {
	ids, err := getUserInstanceIDs(ctx, userID)
	if err != nil || len(ids) == 0 {
		return err
	}
	client, rerr := NewCVMClient(ctx)
	if rerr != nil {
		return rerr
	}
	runningIds, err := FilterInstancesByState(client, ids, "RUNNING")
	if err != nil || len(runningIds) == 0 {
		return err
	}
	req := cvm.NewStopInstancesRequest()
	req.InstanceIds = common.StringPtrs(runningIds)
	_, err = client.StopInstances(req)
	return err
}

// startUserInstances starts all STOPPED CVM instances belonging to the given user.
func startUserInstances(ctx context.Context, userID uint) error {
	ids, err := getUserInstanceIDs(ctx, userID)
	if err != nil || len(ids) == 0 {
		return err
	}
	client, rerr := NewCVMClient(ctx)
	if rerr != nil {
		return rerr
	}
	stoppedIds, err := FilterInstancesByState(client, ids, "STOPPED")
	if err != nil || len(stoppedIds) == 0 {
		return err
	}
	req := cvm.NewStartInstancesRequest()
	req.InstanceIds = common.StringPtrs(stoppedIds)
	_, err = client.StartInstances(req)
	return err
}

// HandleUserLimit returns the current user count and the configured user limit.
// GET /admin/user-limit
func HandleUserLimit(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyGetMethod))
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	var count int64
	model.DB(r.Context()).Unscoped().Model(&model.User{}).Count(&count)
	jsonOK(w, map[string]interface{}{
		"count": count,
		"limit": hcommon.UserLimitFromCtx(r.Context()),
	})
}

// vpcHasResources returns (true, nil) if the VPC has blocking resources,
// (false, nil) if it is safe to delete, or (false, err) on API failure.
// Exempt fields (not blocking deletion): Subnet, RouteTable, RouteId, NetworkACL.
func vpcHasResources(ctx context.Context, vpcId string) (bool, error) {
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return false, hcommon.I18nError(i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}
	req := vpc.NewDescribeVpcResourceDashboardRequest()
	req.VpcIds = common.StringPtrs([]string{vpcId})
	resp, err := vpcClient.DescribeVpcResourceDashboard(req)
	if err != nil {
		if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok && sdkErr.GetCode() == "ResourceNotFound" {
			return false, nil
		}
		return false, hcommon.I18nError(i18n.MsgQueryVPCResourceFailed).WithDetail(err.Error())
	}
	if resp.Response == nil {
		return false, nil
	}
	for _, d := range resp.Response.ResourceDashboardSet {
		blocking := []*uint64{
			d.Classiclink, d.Dcg, d.Pcx, d.Ip, d.Nat, d.Vpngw,
			d.FlowLog, d.NetworkDetect,
			d.CVM, d.LB,
			d.CDB, d.Cmem, d.CTSDB, d.MariaDB, d.SQLServer, d.Postgres,
			d.NAS, d.Greenplumn, d.Ckafka, d.Grocery, d.HSM, d.Tcaplus,
			d.Cnas, d.TiDB, d.Emr, d.SEAL, d.CFS, d.Oracle,
			d.ElasticSearch, d.TBaaS, d.Itop, d.DBAudit,
			d.CynosDBPostgres, d.Redis, d.MongoDB, d.DCDB, d.CynosDBMySQL,
		}
		for _, v := range blocking {
			if v != nil && *v > 0 {
				return true, nil
			}
		}
	}

	// Dashboard does not reflect ENI occupancy; check explicitly.
	eniReq := vpc.NewDescribeNetworkInterfacesRequest()
	eniReq.Filters = []*vpc.Filter{{
		Name:   common.StringPtr("vpc-id"),
		Values: common.StringPtrs([]string{vpcId}),
	}}
	eniResp, err := vpcClient.DescribeNetworkInterfaces(eniReq)
	if err != nil {
		return false, hcommon.I18nError(i18n.MsgQueryENIFailed).WithDetail(err.Error())
	}
	if eniResp.Response != nil && *eniResp.Response.TotalCount > 0 {
		return true, nil
	}
	return false, nil
}

// deleteVPC deletes the VPC. Returns nil if VPC no longer exists.
func deleteVPC(ctx context.Context, vpcId string) error {
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}
	req := vpc.NewDeleteVpcRequest()
	req.VpcId = common.StringPtr(vpcId)
	if _, err := vpcClient.DeleteVpc(req); err != nil {
		if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok && sdkErr.GetCode() == "ResourceNotFound" {
			return nil
		}
		return hcommon.I18nError(i18n.MsgDeleteVPCFailed).WithDetail(err.Error())
	}
	return nil
}

// HandleUserVPC returns the user's auto-created VPC ID and whether it has blocking resources.
// Response: {"vpc_id": "vpc-xxx", "has_resources": true/false} or {"vpc_id": null} if no
// auto-created VPC (user uses custom VPC, never created an instance, or VPC already deleted).
func HandleUserVPC(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	var user model.User
	if model.DB(r.Context()).Unscoped().Where("id = ?", id).First(&user).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
		return
	}
	if user.VpcId == "" {
		jsonOK(w, map[string]interface{}{"vpc_id": nil})
		return
	}

	// Check if VPC still exists in Tencent Cloud.
	vpcClient, err := newVpcClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}
	descReq := vpc.NewDescribeVpcsRequest()
	descReq.VpcIds = common.StringPtrs([]string{user.VpcId})
	descResp, err := vpcClient.DescribeVpcs(descReq)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryVpcFailed))
		return
	}
	if descResp.Response == nil || *descResp.Response.TotalCount == 0 {
		jsonOK(w, map[string]interface{}{"vpc_id": nil})
		return
	}

	hasResources, err := vpcHasResources(r.Context(), user.VpcId)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	jsonOK(w, map[string]interface{}{
		"vpc_id":        user.VpcId,
		"has_resources": hasResources,
	})
}

// HandleDepartments returns the full department list (including sub-departments) with hierarchy paths.
// GET /admin/departments
func HandleDepartments(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyGetMethod))
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	// 构建全量部门映射（优先从 oneid_departments 表，再从用户画像补充）
	globalDeptMap := model.BuildFullDeptMap(r.Context())

	// 构建返回列表：包含 oneid_departments 表中所有已同步的部门
	type deptInfo struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Path     string `json:"path"` // 完整层级路径
		ParentID string `json:"parent_id"`
		HasChild bool   `json:"has_child"`
	}

	result := make([]deptInfo, 0, len(globalDeptMap))

	// 预建父节点集合：凡是被某个部门引用为 ParentID 的，都有子部门
	hasChildSet := make(map[string]bool, len(globalDeptMap))
	for _, dept := range globalDeptMap {
		if dept.DepartmentParentID != "" {
			hasChildSet[dept.DepartmentParentID] = true
		}
	}

	for deptID, dept := range globalDeptMap {
		path := buildDepartmentPath(globalDeptMap, deptID)
		if path == "" {
			path = dept.DepartmentName
		}
		result = append(result, deptInfo{
			ID:       deptID,
			Name:     dept.DepartmentName,
			Path:     path,
			ParentID: dept.DepartmentParentID,
			HasChild: hasChildSet[deptID],
		})
	}

	// 兼容旧格式：主部门名称去重列表
	var mainDeptNameList []string
	model.DB(r.Context()).Model(&model.OneIDUserProfile{}).
		Where("main_dept_name != ''").
		Distinct("main_dept_name").
		Order("main_dept_name").
		Pluck("main_dept_name", &mainDeptNameList)

	jsonOK(w, map[string]interface{}{
		"departments":     mainDeptNameList, // 兼容旧格式
		"department_tree": result,           // 新格式，含 id/name/path/parent_id
	})
}

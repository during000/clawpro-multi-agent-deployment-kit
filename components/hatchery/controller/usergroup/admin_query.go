package usergroup

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ──────────────────────────────────────────────
// GetGroupTree —— 核心接口 1：分组列表树
// ──────────────────────────────────────────────

// TreeOptions 查询参数。
type TreeOptions struct {
	Query              string   // 按 full_path / name 模糊过滤
	WithUserCounts     bool     // 是否计算每个节点的 direct_user_count / descendant_user_count
	WithHealth         bool     // 是否计算每个节点的配置健康度（模型/通道/网络/镜像）
	WithResourcePolicy bool     // 是否返回分组直接绑定的资源策略
	Sources            []string // source 白名单（如 ["manual", "oneid_dept"]）；空=返回全部
}

type ResourcePolicyRef struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// TreeNode 树节点响应结构（对齐 API 文档 §1）。
type TreeNode struct {
	ID                   uint               `json:"id"`
	Name                 string             `json:"name"`
	FullPath             string             `json:"full_path"`
	ParentID             *uint              `json:"parent_id"` // 根组为 null
	Depth                int                `json:"depth"`
	Source               string             `json:"source"`
	SourceRef            string             `json:"source_ref,omitempty"`
	Readonly             bool               `json:"readonly"`
	ToBeDeleted          bool               `json:"to_be_deleted"`
	DirectUserCount      int64              `json:"direct_user_count"`
	DescendantUserCount  int64              `json:"descendant_user_count"`
	DescendantGroupCount int64              `json:"descendant_count"`
	Health               *NodeHealth        `json:"health,omitempty"`
	DirectResourcePolicy *ResourcePolicyRef `json:"direct_resource_policy,omitempty"`
	Children             []*TreeNode        `json:"children"`
	CreatedAt            time.Time          `json:"-"` // 仅用于排序，不输出到 JSON
}

// NodeHealth 节点配置健康度（4 项核心检查）
type NodeHealth struct {
	Healthy bool     `json:"healthy"`
	Missing []string `json:"missing,omitempty"` // 缺失的配置项 key：model / channel / network / image
}

// TreeSummary 响应的 summary 部分。
type TreeSummary struct {
	TotalGroups         int64 `json:"total_groups"`
	ManualGroups        int64 `json:"manual_groups"`
	OneIDDeptGroups     int64 `json:"oneid_dept_groups"`
	ToBeDeletedCount    int64 `json:"to_be_deleted_count"`
	MultiGroupUsersCnt  int64 `json:"multi_group_users_count"`
	UngroupedUsersCount int64 `json:"ungrouped_users_count"`
}

// TreeResponse 完整响应（对齐 API 文档 §1）。
type TreeResponse struct {
	Summary    TreeSummary    `json:"summary"`
	OrgTree    []*TreeNode    `json:"org_tree"`    // source='oneid_dept' 的多级树
	UserGroups []*TreeNode    `json:"user_groups"` // source='manual' 的多级树
	Ungrouped  UngroupedBrief `json:"ungrouped"`
}

// UngroupedBrief /tree 中 ungrouped 段的精简表达。
type UngroupedBrief struct {
	UserCount int64 `json:"user_count"`
}

// normalizeSources 清洗 sources 白名单：
//   - TrimSpace / ToLower
//   - 去重
//   - 仅保留已知合法 source（manual / oneid_dept）；未知值静默丢弃
//   - 输入为空或清洗后为空时返回 nil（调用方以此代表"不加过滤"）
func normalizeSources(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if s != model.GroupSourceManual && s != model.GroupSourceOneIDDept {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// GetGroupTree 构造分组列表树响应（核心接口 1）。
// 一次性加载全部分组（平台上限 2000 条），内存成树，避免 N+1。
func GetGroupTree(ctx context.Context, opts TreeOptions) (*TreeResponse, error) {
	var groups []model.UserGroup
	db := model.DB(ctx).Model(&model.UserGroup{})
	if q := strings.TrimSpace(opts.Query); q != "" {
		db = db.Where("full_path LIKE ? OR name LIKE ?", "%"+q+"%", "%"+q+"%")
	}
	if sources := normalizeSources(opts.Sources); len(sources) > 0 {
		db = db.Where("source IN ?", sources)
	}
	if err := db.Order("parent_id ASC, created_at ASC").Find(&groups).Error; err != nil {
		return nil, err
	}

	nodesByID := make(map[uint]*TreeNode, len(groups))
	for i := range groups {
		g := groups[i]
		nodesByID[g.ID] = newTreeNode(&g)
	}

	if opts.WithResourcePolicy {
		groupIDs := make([]uint, 0, len(groups))
		for _, group := range groups {
			groupIDs = append(groupIDs, group.ID)
		}
		directPolicies, err := model.GetDirectResourcePoliciesByGroup(ctx, groupIDs)
		if err != nil {
			return nil, err
		}
		for groupID, policy := range directPolicies {
			node, ok := nodesByID[groupID]
			if !ok {
				continue
			}
			node.DirectResourcePolicy = &ResourcePolicyRef{ID: policy.ID, Name: policy.DisplayName(ctx)}
		}
	}

	var orgRoots, userGroupRoots []*TreeNode
	summary := TreeSummary{TotalGroups: int64(len(groups))}
	for i := range groups {
		g := groups[i]
		node := nodesByID[g.ID]
		switch g.Source {
		case model.GroupSourceManual:
			summary.ManualGroups++
		case model.GroupSourceOneIDDept:
			summary.OneIDDeptGroups++
		}
		if g.ToBeDeleted {
			summary.ToBeDeletedCount++
		}
		if g.ParentID == 0 {
			if g.Source == model.GroupSourceOneIDDept {
				orgRoots = append(orgRoots, node)
			} else {
				userGroupRoots = append(userGroupRoots, node)
			}
			continue
		}
		parent, ok := nodesByID[g.ParentID]
		if !ok {
			// 父组在过滤结果外：把节点当"伪根"挂到对应分段
			if g.Source == model.GroupSourceOneIDDept {
				orgRoots = append(orgRoots, node)
			} else {
				userGroupRoots = append(userGroupRoots, node)
			}
			continue
		}
		parent.Children = append(parent.Children, node)
	}

	for _, root := range orgRoots {
		calcDescendantCounts(root)
	}
	for _, root := range userGroupRoots {
		calcDescendantCounts(root)
	}

	if opts.WithUserCounts {
		if err := fillUserCounts(ctx, nodesByID); err != nil {
			return nil, err
		}
	}

	if opts.WithHealth {
		fillNodeHealth(ctx, nodesByID, groups)
	}

	sortTreeByCreatedAt(orgRoots)
	sortTreeByCreatedAt(userGroupRoots)

	ungroupedCnt, err := countUngroupedUsers(ctx)
	if err != nil {
		return nil, err
	}
	summary.UngroupedUsersCount = ungroupedCnt

	multiCnt, err := countMultiGroupUsers(ctx)
	if err != nil {
		return nil, err
	}
	summary.MultiGroupUsersCnt = multiCnt

	return &TreeResponse{
		Summary:    summary,
		OrgTree:    orgRoots,
		UserGroups: userGroupRoots,
		Ungrouped:  UngroupedBrief{UserCount: ungroupedCnt},
	}, nil
}

func newTreeNode(g *model.UserGroup) *TreeNode {
	var parentPtr *uint
	if g.ParentID != 0 {
		pid := g.ParentID
		parentPtr = &pid
	}
	return &TreeNode{
		ID:          g.ID,
		Name:        g.Name,
		FullPath:    g.FullPath,
		ParentID:    parentPtr,
		Depth:       g.Depth,
		Source:      g.Source,
		SourceRef:   g.SourceRef,
		Readonly:    g.Readonly(),
		ToBeDeleted: g.ToBeDeleted,
		Children:    []*TreeNode{},
		CreatedAt:   g.CreatedAt,
	}
}

// calcDescendantCounts 递归计算节点的 descendant_count，返回以该节点为根的子树大小（不含自身）。
func calcDescendantCounts(node *TreeNode) int64 {
	var cnt int64
	for _, c := range node.Children {
		cnt += 1 + calcDescendantCounts(c)
	}
	node.DescendantGroupCount = cnt
	return cnt
}

func sortTreeByCreatedAt(nodes []*TreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].CreatedAt.Before(nodes[j].CreatedAt)
	})
	for _, n := range nodes {
		sortTreeByCreatedAt(n.Children)
	}
}

// fillUserCounts 批量查询每个组的 direct_user_count 与 descendant_user_count。
// 两次 SQL：
//  1. 直接成员数：user_group_members GROUP BY user_group_id
//  2. 后代成员数：JOIN group_closure（ancestor_id = 节点 id）+ GROUP BY ancestor_id
func fillUserCounts(ctx context.Context, nodes map[uint]*TreeNode) error {
	if len(nodes) == 0 {
		return nil
	}
	groupIDs := make([]uint, 0, len(nodes))
	for id := range nodes {
		groupIDs = append(groupIDs, id)
	}

	// 直接成员数
	directCounts, err := model.CountGroupMembersBatch(model.DB(ctx), groupIDs)
	if err != nil {
		return err
	}
	for id, cnt := range directCounts {
		if n, ok := nodes[id]; ok {
			n.DirectUserCount = cnt
		}
	}

	// 后代成员数（含自身）：JOIN closure
	type row struct {
		AncestorID uint  `gorm:"column:ancestor_id"`
		Cnt        int64 `gorm:"column:cnt"`
	}
	var rows []row
	if dbErr := model.DB(ctx).Table("group_closure AS c").
		Select("c.ancestor_id AS ancestor_id, COUNT(DISTINCT m.user_id) AS cnt").
		Joins("INNER JOIN user_group_members m ON m.user_group_id = c.descendant_id").
		Where("c.ancestor_id IN ?", groupIDs).
		Group("c.ancestor_id").
		Scan(&rows).Error; dbErr != nil {
		return hcommon.I18nRichError(dbErr, i18n.MsgQueryUserGroupFailed)
	}
	for _, r := range rows {
		if n, ok := nodes[r.AncestorID]; ok {
			n.DescendantUserCount = r.Cnt
		}
	}
	return nil
}

// countUngroupedUsers 查询未加入任何分组的用户数（含禁用用户，与 members 查询一致）。
func countUngroupedUsers(ctx context.Context) (int64, error) {
	var cnt int64
	err := model.DB(ctx).Unscoped().Model(&model.User{}).
		Joins("LEFT JOIN user_group_members m ON m.user_id = users.id").
		Where("m.user_id IS NULL").
		Count(&cnt).Error
	return cnt, err
}

// countMultiGroupUsers 统计属于 ≥ 2 个组的用户数。
func countMultiGroupUsers(ctx context.Context) (int64, error) {
	type row struct {
		Cnt int64 `gorm:"column:cnt"`
	}
	var r row
	err := model.DB(ctx).Raw(`
		SELECT COUNT(*) AS cnt FROM (
			SELECT user_id FROM user_group_members
			GROUP BY user_id
			HAVING COUNT(*) >= 2
		) t
	`).Scan(&r).Error
	if err != nil {
		return 0, err
	}
	return r.Cnt, nil
}

// ──────────────────────────────────────────────
// GetGroupMembersPaged —— 成员分页（§2）
// ──────────────────────────────────────────────

// MemberDirectGroupRef 成员所属的单个分组的简要视图。
type MemberDirectGroupRef struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	FullPath      string `json:"full_path"`
	Source        string `json:"source"` // manual / oneid_dept
	IsMain        bool   `json:"is_main"`
	CreatedAt     string `json:"created_at"`     // 分组创建时间（UTC RFC3339）
	InstanceCount int64  `json:"instance_count"` // 🆕 v9: 该用户在该分组下的 agent 数量（instances.user_id + group_id 聚合，含所有状态）
}

// MemberView §2 members[] 的单条响应（v6.14 字段结构）。
type MemberView struct {
	UserID         uint                   `json:"user_id"`
	Username       string                 `json:"username"`
	Role           string                 `json:"role"`
	DeletedAt      *string                `json:"deleted_at"` // 非 nil 表示用户已禁用（与 /admin/users 对齐）
	DirectGroups   []MemberDirectGroupRef `json:"direct_groups"`
	IsMain         bool                   `json:"is_main"` // 该成员在当前组的 is_main（仅 oneid_dept 组有意义）
	Source         string                 `json:"source"`  // 该成员在当前组的 source
	JoinedAt       string                 `json:"joined_at"`
	FromDescendant bool                   `json:"from_descendant"` // 是否仅因查询包含子孙组而返回
}

// MembersResponse §2 / §3 响应。
type MembersResponse struct {
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Members  []MemberView `json:"members"`
}

// MembersOptions 成员分页查询参数。
type MembersOptions struct {
	GroupID            uint
	Page               int
	PageSize           int
	Query              string // 按 username 模糊
	IncludeDescendants bool   // true = 含子孙组（沿 closure）
}

// GetGroupMembersPaged 聚合查询：当前组（或含子孙）的成员分页列表。
//
// direct_groups 排序：
//  1. source='oneid_dept' 优先（排在 manual 之前）
//  2. 同为 oneid_dept 时，is_main=true 的主部门排最前
//  3. 其余按 full_path ASC
func GetGroupMembersPaged(ctx context.Context, opts MembersOptions) (*MembersResponse, error) {
	if opts.GroupID == 0 {
		return nil, model.ErrUserGroupNotFound
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 200 {
		opts.PageSize = 20
	}

	// 先校验组存在
	g, err := model.GroupByID(ctx, opts.GroupID)
	if err != nil {
		return nil, err
	}

	// 确定 scope 组 ID 集合
	scopeIDs := []uint{opts.GroupID}
	if opts.IncludeDescendants {
		descs, err := model.ClosureDescendants(ctx, opts.GroupID, true)
		if err != nil {
			return nil, err
		}
		if len(descs) > 0 {
			scopeIDs = descs
		}
	}

	// 先查 distinct user_id 用于分页（同一用户可能在 scope 的多个组里）
	type userRow struct {
		UserID    uint    `gorm:"column:user_id"`
		Username  string  `gorm:"column:username"`
		Role      string  `gorm:"column:role"`
		DeletedAt *string `gorm:"column:deleted_at"` // NULL=正常，非 NULL=已禁用
	}
	q := model.DB(ctx).Table("user_group_members AS m").
		Select("m.user_id AS user_id, u.username AS username, u.role AS role, u.deleted_at AS deleted_at").
		Joins("INNER JOIN users u ON u.id = m.user_id").
		Where("m.user_group_id IN ?", scopeIDs).
		Group("m.user_id, u.username")
	if kw := strings.TrimSpace(opts.Query); kw != "" {
		q = q.Having("u.username LIKE ?", "%"+kw+"%")
	}

	// 总数
	var total int64
	if err := model.DB(ctx).Table("(?) AS sub", q).Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	var rows []userRow
	if err := q.
		Order("username ASC").
		Offset((opts.Page - 1) * opts.PageSize).
		Limit(opts.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &MembersResponse{
			Total: total, Page: opts.Page, PageSize: opts.PageSize,
			Members: []MemberView{},
		}, nil
	}

	userIDs := make([]uint, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
	}

	// 拉取当前组中这些用户的 is_main / source / joined_at
	var curMembers []model.UserGroupMember
	if err := model.DB(ctx).
		Where("user_group_id = ? AND user_id IN ?", opts.GroupID, userIDs).
		Find(&curMembers).Error; err != nil {
		return nil, err
	}
	byUserInCurrent := make(map[uint]model.UserGroupMember, len(curMembers))
	for _, m := range curMembers {
		byUserInCurrent[m.UserID] = m
	}

	// 查这些用户的所有直属分组（构造 direct_groups），一次 JOIN
	type groupRef struct {
		UserID    uint   `gorm:"column:user_id"`
		GroupID   uint   `gorm:"column:id"`
		Name      string `gorm:"column:name"`
		FullPath  string `gorm:"column:full_path"`
		Source    string `gorm:"column:source"`
		IsMain    bool   `gorm:"column:is_main"`
		CreatedAt string `gorm:"column:created_at"`
	}
	var refs []groupRef
	if err := model.DB(ctx).Table("user_group_members AS m").
		Select("m.user_id AS user_id, g.id AS id, g.name AS name, g.full_path AS full_path, g.source AS source, m.is_main AS is_main, g.created_at AS created_at").
		Joins("INNER JOIN user_groups g ON g.id = m.user_group_id").
		Where("m.user_id IN ?", userIDs).
		Scan(&refs).Error; err != nil {
		return nil, err
	}
	groupsByUser := make(map[uint][]MemberDirectGroupRef, len(userIDs))
	for _, r := range refs {
		groupsByUser[r.UserID] = append(groupsByUser[r.UserID], MemberDirectGroupRef{
			ID:        r.GroupID,
			Name:      r.Name,
			FullPath:  r.FullPath,
			Source:    r.Source,
			IsMain:    r.IsMain,
			CreatedAt: r.CreatedAt,
		})
	}

	// 🆕 v9: 批量聚合 (user_id, group_id) -> instance 数量，避免 N×M 查询
	// 只有当存在直属分组记录时才查；userIDs 至少非空（rows 非空保证）。
	// ⚠️ 必须用 Model(&model.Instance{}) 而不是 Table("instances")，否则会绕开
	//    GORM 的 deleted_at IS NULL 自动过滤，把已软删/销毁的实例也算进去。
	if len(refs) > 0 {
		// 收集本批次涉及的 group_id（去重），将 IN 条件限定最小集
		groupIDSet := make(map[uint]struct{}, len(refs))
		for _, r := range refs {
			groupIDSet[r.GroupID] = struct{}{}
		}
		groupIDs := make([]uint, 0, len(groupIDSet))
		for gid := range groupIDSet {
			groupIDs = append(groupIDs, gid)
		}

		type instCountRow struct {
			UserID  uint  `gorm:"column:user_id"`
			GroupID uint  `gorm:"column:group_id"`
			Cnt     int64 `gorm:"column:cnt"`
		}
		var instRows []instCountRow
		if err := model.DB(ctx).Model(&model.Instance{}).
			Select("user_id, group_id, COUNT(*) AS cnt").
			Where("user_id IN ? AND group_id IN ?", userIDs, groupIDs).
			Group("user_id, group_id").
			Scan(&instRows).Error; err != nil {
			return nil, err
		}
		instCountBy := make(map[[2]uint]int64, len(instRows))
		for _, r := range instRows {
			instCountBy[[2]uint{r.UserID, r.GroupID}] = r.Cnt
		}
		for uid, refs := range groupsByUser {
			for i := range refs {
				refs[i].InstanceCount = instCountBy[[2]uint{uid, refs[i].ID}]
			}
			groupsByUser[uid] = refs
		}
	}

	// 排序：
	//   1) source='oneid_dept' 在前，其它（manual）在后
	//   2) 同为 oneid_dept 时，is_main=true 的主部门排最前
	//   3) 其余按 full_path ASC
	for uid := range groupsByUser {
		groupsByUser[uid] = sortDirectGroups(groupsByUser[uid])
	}

	// 组装响应
	out := make([]MemberView, len(rows))
	for i, r := range rows {
		cur, isDirectMember := byUserInCurrent[r.UserID]
		out[i] = MemberView{
			UserID:         r.UserID,
			Username:       r.Username,
			Role:           r.Role,
			DeletedAt:      r.DeletedAt,
			DirectGroups:   groupsByUser[r.UserID],
			IsMain:         cur.IsMain && g.Source == model.GroupSourceOneIDDept,
			Source:         cur.Source,
			JoinedAt:       cur.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			FromDescendant: opts.IncludeDescendants && !isDirectMember,
		}
	}

	return &MembersResponse{
		Total: total, Page: opts.Page, PageSize: opts.PageSize,
		Members: out,
	}, nil
}

func sourceOrder(src string) int {
	if src == model.GroupSourceOneIDDept {
		return 0
	}
	return 1
}

// sortDirectGroups 按"oneid_dept 优先 > 主部门(is_main) 优先 > 层级浅到深 > 同层级按创建时间"
// 四级规则稳定排序 direct_groups 列表。
func sortDirectGroups(list []MemberDirectGroupRef) []MemberDirectGroupRef {
	sort.SliceStable(list, func(i, j int) bool {
		// source 优先级：oneid_dept > manual
		si := sourceOrder(list[i].Source)
		sj := sourceOrder(list[j].Source)
		if si != sj {
			return si < sj
		}
		// 主部门优先（仅 oneid_dept 内比较）
		if list[i].IsMain != list[j].IsMain {
			return list[i].IsMain
		}
		// full_path 层级浅到深
		depthI := strings.Count(list[i].FullPath, "/")
		depthJ := strings.Count(list[j].FullPath, "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		// 同层级按创建时间
		return list[i].CreatedAt < list[j].CreatedAt
	})
	return list
}

// ──────────────────────────────────────────────
// GetUngroupedMembersPaged —— 游离用户分页（§3）
// ──────────────────────────────────────────────

// GetUngroupedMembersPaged 分页游离用户，响应结构与 GetGroupMembersPaged 对齐。
func GetUngroupedMembersPaged(ctx context.Context, page, pageSize int, query string) (*MembersResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}

	baseQ := model.DB(ctx).Unscoped().Model(&model.User{}).
		Joins("LEFT JOIN user_group_members m ON m.user_id = users.id").
		Where("m.user_id IS NULL")
	if kw := strings.TrimSpace(query); kw != "" {
		baseQ = baseQ.Where("users.username LIKE ?", "%"+kw+"%")
	}

	var total int64
	if err := baseQ.Count(&total).Error; err != nil {
		return nil, err
	}

	var users []model.User
	if err := baseQ.
		Order("users.username ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		return nil, err
	}

	out := make([]MemberView, len(users))
	for i, u := range users {
		var deletedAt *string
		if u.DeletedAt.Valid {
			s := u.DeletedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
			deletedAt = &s
		}
		out[i] = MemberView{
			UserID:       u.ID,
			Username:     u.Username,
			Role:         u.Role,
			DeletedAt:    deletedAt,
			DirectGroups: []MemberDirectGroupRef{}, // 游离用户恒空
			IsMain:       false,
			Source:       "",
			JoinedAt:     u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	return &MembersResponse{
		Total: total, Page: page, PageSize: pageSize,
		Members: out,
	}, nil
}

// ──────────────────────────────────────────────
// GetMultiGroupStats —— 多归属统计（§8）
// ──────────────────────────────────────────────

// MultiGroupExample 多归属用户的单条示例。
type MultiGroupExample struct {
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	GroupCount int    `json:"group_count"`
}

// MultiGroupStats §8 响应。
type MultiGroupStats struct {
	TotalUsers      int64               `json:"total_users"`
	MultiGroupUsers int64               `json:"multi_group_users"`
	UngroupedUsers  int64               `json:"ungrouped_users"`
	TopExamples     []MultiGroupExample `json:"top_examples"`
}

// GetMultiGroupStats 聚合统计 + 前 5 个多组用户示例。
func GetMultiGroupStats(ctx context.Context) (*MultiGroupStats, error) {
	var totalUsers int64
	if err := model.DB(ctx).Model(&model.User{}).Count(&totalUsers).Error; err != nil {
		return nil, err
	}

	multi, err := countMultiGroupUsers(ctx)
	if err != nil {
		return nil, err
	}

	ungrouped, err := countUngroupedUsers(ctx)
	if err != nil {
		return nil, err
	}

	// top 5 多组用户
	type row struct {
		UserID uint   `gorm:"column:user_id"`
		Cnt    int    `gorm:"column:cnt"`
		Name   string `gorm:"column:username"`
	}
	var rows []row
	err = model.DB(ctx).Raw(`
		SELECT m.user_id AS user_id, u.username AS username, COUNT(*) AS cnt
		FROM user_group_members m
		INNER JOIN users u ON u.id = m.user_id
		GROUP BY m.user_id, u.username
		HAVING COUNT(*) >= 2
		ORDER BY cnt DESC, m.user_id ASC
		LIMIT 5
	`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	examples := make([]MultiGroupExample, len(rows))
	for i, r := range rows {
		examples[i] = MultiGroupExample{
			UserID: r.UserID, Username: r.Name, GroupCount: r.Cnt,
		}
	}

	return &MultiGroupStats{
		TotalUsers:      totalUsers,
		MultiGroupUsers: multi,
		UngroupedUsers:  ungrouped,
		TopExamples:     examples,
	}, nil
}

// ──────────────────────────────────────────────
// GetDeleteImpact —— 删除影响报告（§6）
// ──────────────────────────────────────────────

// GroupBrief 组简要信息。
type GroupBrief struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
	Source   string `json:"source"`
}

// ResourceBinding 资源绑定阻塞项条目。
type ResourceBinding struct {
	ResourceID   uint   `json:"resource_id"`
	ResourceName string `json:"resource_name"`
}

// InstanceBrief 分组直属 Agent（db instances 表中 group_id 命中此分组的记录）。
// 用于阻塞分组删除 —— 必须先迁走或销毁这些 Agent 再删除分组。
type InstanceBrief struct {
	InstanceID string `json:"instance_id"` // 腾讯云 CVM 实例 ID（ins-xxx），占位记录阶段可能为空
	Name       string `json:"name"`        // 实例展示名
}

// DeleteBlockers 删除阻塞项。
// resource_bindings 的 key 对齐 config-overview 的 category key：
// model / channel / skill / agentTool / imageType / platformPolicy。
// 🆕 v6.13：新增 instances 字段，直属 Agent 阻塞删除。
type DeleteBlockers struct {
	ManualDescendants []GroupBrief                 `json:"manual_descendants"`
	ResourceBindings  map[string][]ResourceBinding `json:"resource_bindings"`
	Instances         []InstanceBrief              `json:"instances"`
}

// DeleteNonBlockingInfo 非阻塞提示。
type DeleteNonBlockingInfo struct {
	DirectMembersCount int64  `json:"direct_members_count"`
	TotalMemberCount   int64  `json:"total_member_count"`
	Note               string `json:"note"`
}

// DeleteImpact API §6 响应。
type DeleteImpact struct {
	Group           GroupBrief            `json:"group"`
	Blockers        DeleteBlockers        `json:"blockers"`
	NonBlockingInfo DeleteNonBlockingInfo `json:"non_blocking_info"`
	Hint            string                `json:"hint,omitempty"`
}

// GetDeleteImpact 组装删除阻塞详情响应（聚合 4 张旧 *_visibility_groups + manual 子组 + 成员数）。
// channel / agent_tool 的 visibility 聚合 P2 补。
func GetDeleteImpact(ctx context.Context, groupID uint) (*DeleteImpact, error) {
	g, err := model.GroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	resp := &DeleteImpact{
		Group: GroupBrief{
			ID:       g.ID,
			Name:     g.Name,
			FullPath: g.FullPath,
			Source:   g.Source,
		},
		Blockers: DeleteBlockers{
			ManualDescendants: []GroupBrief{},
			ResourceBindings:  map[string][]ResourceBinding{},
			Instances:         []InstanceBrief{},
		},
	}

	// manual 子组（直接子）
	var children []model.UserGroup
	if err := model.DB(ctx).
		Where("parent_id = ? AND source = ?", groupID, model.GroupSourceManual).
		Find(&children).Error; err != nil {
		return nil, err
	}
	for _, c := range children {
		resp.Blockers.ManualDescendants = append(resp.Blockers.ManualDescendants, GroupBrief{
			ID: c.ID, Name: c.Name, FullPath: c.FullPath, Source: c.Source,
		})
	}

	// 资源绑定 — key 对齐 config-overview 的 ConfigCategoryList 中的 category key
	// model: 模型（旧表 model_visibility_groups）
	if rows, err := findModelBindings(ctx, groupID); err == nil && len(rows) > 0 {
		resp.Blockers.ResourceBindings[CategoryKeyModel] = rows
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// channel: 通道（GroupConfigBinding）
	if bindings, err := model.GetBindingsByGroup(ctx, groupID, model.ConfigTypeChannel); err == nil && len(bindings) > 0 {
		rows := make([]ResourceBinding, 0, len(bindings))
		for _, b := range bindings {
			rows = append(rows, ResourceBinding{ResourceID: b.ID, ResourceName: b.ConfigKey})
		}
		resp.Blockers.ResourceBindings[CategoryKeyChannel] = rows
	}

	// skill: 技能包 + 角色（旧表 skill_bundle_visibility_groups / role_visibility_groups）
	var skillEntries []ResourceBinding
	if rows, err := findSkillBundleBindings(ctx, groupID); err == nil {
		skillEntries = append(skillEntries, rows...)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if rows, err := findRoleBindings(ctx, groupID); err == nil {
		skillEntries = append(skillEntries, rows...)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if len(skillEntries) > 0 {
		resp.Blockers.ResourceBindings[CategoryKeySkill] = skillEntries
	}

	// agentTool: 企业技能 + 企业插件(plugin_bundle) + 企业 MCP
	var agentToolEntries []ResourceBinding
	if rows, err := findSkillBindings(ctx, groupID); err == nil {
		agentToolEntries = append(agentToolEntries, rows...)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	for _, ct := range []string{model.ConfigTypePluginBundle, model.ConfigTypeMCP} {
		bindings, err := model.GetBindingsByGroup(ctx, groupID, ct)
		if err != nil || len(bindings) == 0 {
			continue
		}
		for _, b := range bindings {
			agentToolEntries = append(agentToolEntries, ResourceBinding{ResourceID: b.ID, ResourceName: b.ConfigKey})
		}
	}
	if len(agentToolEntries) > 0 {
		resp.Blockers.ResourceBindings[CategoryKeyAgentTool] = agentToolEntries
	}

	// imageType: 镜像（GroupConfigBinding）
	if bindings, err := model.GetBindingsByGroup(ctx, groupID, model.ConfigTypeImageType); err == nil && len(bindings) > 0 {
		rows := make([]ResourceBinding, 0, len(bindings))
		for _, b := range bindings {
			rows = append(rows, ResourceBinding{ResourceID: b.ID, ResourceName: b.ConfigKey})
		}
		resp.Blockers.ResourceBindings[CategoryKeyImageType] = rows
	}

	// platformPolicy: 策略（GroupConfigBinding）
	if bindings, err := model.GetBindingsByGroup(ctx, groupID, model.ConfigTypePolicy); err == nil && len(bindings) > 0 {
		rows := make([]ResourceBinding, 0, len(bindings))
		for _, b := range bindings {
			rows = append(rows, ResourceBinding{ResourceID: b.ID, ResourceName: b.ConfigKey})
		}
		resp.Blockers.ResourceBindings[CategoryKeyPlatformPolicy] = rows
	}

	// 🆕 v6.13：直属 Agent（instances.group_id = X）阻塞删除。
	// 只读 id / name 两个字段，避免把 ProxyToken 等敏感字段带到 response。
	var insts []model.Instance
	if err := model.DB(ctx).
		Select("instance_id, name").
		Where("group_id = ?", groupID).
		Find(&insts).Error; err != nil {
		return nil, err
	}
	for _, inst := range insts {
		resp.Blockers.Instances = append(resp.Blockers.Instances, InstanceBrief{
			InstanceID: inst.InstanceId,
			Name:       inst.Name,
		})
	}

	// 直接成员数（非阻塞）
	memberCnt, err := model.CountGroupMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	// 含子孙的总成员数（通过闭包表 JOIN 去重计算）
	totalMemberCnt := countDescendantMembers(ctx, groupID)
	resp.NonBlockingInfo = DeleteNonBlockingInfo{
		DirectMembersCount: memberCnt,
		TotalMemberCount:   totalMemberCnt,
		Note:               i18n.T(ctx, i18n.MsgGroupDeleteNonBlockingMembersNote),
	}

	if len(resp.Blockers.ManualDescendants) == 0 &&
		len(resp.Blockers.ResourceBindings) == 0 &&
		len(resp.Blockers.Instances) == 0 {
		resp.Hint = i18n.T(ctx, i18n.MsgGroupDeleteHintNoBlockers)
	} else if len(resp.Blockers.Instances) > 0 {
		resp.Hint = i18n.T(ctx, i18n.MsgGroupDeleteHintHasInstances)
	} else {
		resp.Hint = i18n.T(ctx, i18n.MsgGroupDeleteHintHasOtherBlockers)
	}

	return resp, nil
}

// countDescendantMembers 统计该组 + 所有子孙组的去重成员数。
// 通过闭包表 JOIN user_group_members，COUNT(DISTINCT user_id)。
func countDescendantMembers(ctx context.Context, groupID uint) int64 {
	type row struct {
		Cnt int64 `gorm:"column:cnt"`
	}
	var r row
	model.DB(ctx).Table("group_closure AS c").
		Select("COUNT(DISTINCT m.user_id) AS cnt").
		Joins("INNER JOIN user_group_members m ON m.user_group_id = c.descendant_id").
		Where("c.ancestor_id = ?", groupID).
		Scan(&r)
	return r.Cnt
}

func findModelBindings(ctx context.Context, groupID uint) ([]ResourceBinding, error) {
	type row struct {
		ResourceID   uint   `gorm:"column:resource_id"`
		ResourceName string `gorm:"column:resource_name"`
	}
	var rows []row
	err := model.DB(ctx).Table("model_visibility_groups AS v").
		Select("v.ai_model_id AS resource_id, m.model_id AS resource_name").
		Joins("INNER JOIN ai_models m ON m.id = v.ai_model_id").
		Where("v.group_id = ?", groupID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]ResourceBinding, len(rows))
	for i, r := range rows {
		result[i] = ResourceBinding{ResourceID: r.ResourceID, ResourceName: r.ResourceName}
	}
	return result, nil
}

func findSkillBindings(ctx context.Context, groupID uint) ([]ResourceBinding, error) {
	type row struct {
		ResourceID   uint   `gorm:"column:resource_id"`
		ResourceName string `gorm:"column:resource_name"`
	}
	var rows []row
	err := model.DB(ctx).Table("skill_visibility_groups AS v").
		Select("v.skill_id AS resource_id, s.name AS resource_name").
		Joins("INNER JOIN skills s ON s.id = v.skill_id").
		Where("v.group_id = ?", groupID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]ResourceBinding, len(rows))
	for i, r := range rows {
		result[i] = ResourceBinding{ResourceID: r.ResourceID, ResourceName: r.ResourceName}
	}
	return result, nil
}

func findSkillBundleBindings(ctx context.Context, groupID uint) ([]ResourceBinding, error) {
	type row struct {
		ResourceID   uint   `gorm:"column:resource_id"`
		ResourceName string `gorm:"column:resource_name"`
	}
	var rows []row
	err := model.DB(ctx).Table("skill_bundle_visibility_groups AS v").
		Select("v.skill_bundle_id AS resource_id, b.name AS resource_name").
		Joins("INNER JOIN skill_bundles b ON b.id = v.skill_bundle_id").
		Where("v.group_id = ?", groupID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]ResourceBinding, len(rows))
	for i, r := range rows {
		result[i] = ResourceBinding{ResourceID: r.ResourceID, ResourceName: r.ResourceName}
	}
	return result, nil
}

func findRoleBindings(ctx context.Context, groupID uint) ([]ResourceBinding, error) {
	type row struct {
		ResourceID   uint   `gorm:"column:resource_id"`
		ResourceName string `gorm:"column:resource_name"`
	}
	var rows []row
	err := model.DB(ctx).Table("role_visibility_groups AS v").
		Select("v.open_claw_role_id AS resource_id, r.name AS resource_name").
		Joins("INNER JOIN open_claw_roles r ON r.id = v.open_claw_role_id").
		Where("v.group_id = ?", groupID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]ResourceBinding, len(rows))
	for i, r := range rows {
		result[i] = ResourceBinding{ResourceID: r.ResourceID, ResourceName: r.ResourceName}
	}
	return result, nil
}

// ──────────────────────────────────────────────
// 节点配置健康度 —— fillNodeHealth
// ──────────────────────────────────────────────
// 检查 4 项核心配置：模型 / 通道 / 网络(VPC) / 镜像
// 优化策略：
//   1. 先检查全局可用性（visibility=all 或 site_config 有值），满足则所有组直接通过
//   2. 不满足全局的项，利用闭包表批量查每个组的祖先链上是否有绑定

// fillNodeHealth 为所有节点计算配置健康度。
func fillNodeHealth(ctx context.Context, nodesByID map[uint]*TreeNode, groups []model.UserGroup) {
	// 第一步：检查全局覆盖
	globalModel := HasGlobalModel(ctx)
	globalChannel := HasGlobalChannel(ctx)
	globalNetwork := HasGlobalNetwork(ctx)
	globalImage := HasGlobalImage(ctx)

	// 如果 4 项全部全局满足，所有组都健康，直接返回
	if globalModel && globalChannel && globalNetwork && globalImage {
		for _, node := range nodesByID {
			node.Health = &NodeHealth{Healthy: true}
		}
		return
	}

	// 第二步：对未全局满足的项，批量查绑定
	groupIDs := make([]uint, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}

	// 批量查每个组的祖先链（含自己）
	ancestorMap := batchGetAncestorsForHealth(ctx, groupIDs)

	// 需要检查的项
	type checkItem struct {
		key          string
		globalOK     bool
		hasBindingFn func(ctx context.Context, ancestors []uint) bool
	}
	checks := []checkItem{
		{CategoryKeyModel, globalModel, hasModelBinding},
		{CategoryKeyChannel, globalChannel, hasChannelBinding},
		{CategoryKeyNetwork, globalNetwork, hasVpcBinding},
		{CategoryKeyImageType, globalImage, hasImageTypeBinding},
	}

	for _, node := range nodesByID {
		var missing []string
		ancestors := ancestorMap[node.ID]
		for _, ck := range checks {
			if ck.globalOK {
				continue
			}
			if ck.hasBindingFn == nil {
				missing = append(missing, ck.key)
				continue
			}
			if !ck.hasBindingFn(ctx, ancestors) {
				missing = append(missing, ck.key)
			}
		}
		node.Health = &NodeHealth{
			Healthy: len(missing) == 0,
			Missing: missing,
		}
	}
}

// ── 全局配置检查（公开函数，admin_notices 等共用） ──

// HasGlobalModel 检查是否存在已启用、对用户可见且全局可见的模型
func HasGlobalModel(ctx context.Context) bool {
	var count int64
	model.DB(ctx).Model(&model.AIModel{}).
		Where("enabled = ? AND visible = ? AND (visibility_type = ? OR visibility_type = ?)", true, true, VisibilityAll, "").
		Where("NOT (provider = ? AND model_id = ?)", "hatchery", "custom").
		Count(&count)
	return count > 0
}

// HasGlobalChannel 检查是否存在已启用且全局可见的通道
func HasGlobalChannel(ctx context.Context) bool {
	var count int64
	model.DB(ctx).Model(&model.AIChannel{}).
		Where("enabled = ? AND (visibility_type = ? OR visibility_type = ?)", true, VisibilityAll, "").
		Count(&count)
	return count > 0
}

// HasGlobalNetwork 检查是否已配置全局可见的私有网络（VPC）
// 优先检查 vpc_configs 表中 visibility_type=all 的记录（新方案），为空回退 site_config.VpcId（旧方案），
// 再回退 site_config.DefaultVpcId（自动分配模式）。
// 2026-05-13：新用户进入时 VPC 会自动分配（vpc_configs visibility_type=all），
// defaultVpcId 只有在首次创建实例后才会写入，所以此步骤不可能未完成，直接返回 true。
func HasGlobalNetwork(ctx context.Context) bool {
	return true
}

// HasConfiguredSecurityGroup 检查是否已配置安全组
// 优先检查默认 RuleSet 下有 ACTIVE SG，为空回退 site_config.SecurityGroupId
func HasConfiguredSecurityGroup(ctx context.Context) bool {
	if rs, err := model.GetDefaultRuleSet(ctx); err == nil {
		if sgs, err := model.ListActiveSGsByRuleSet(ctx, rs.ID); err == nil && len(sgs) > 0 {
			return true
		}
	}
	cfg := model.GetSiteConfig(ctx)
	return cfg.SecurityGroupId != ""
}

// HasGlobalImage 检查是否存在已启用且 agent_type 对所有组可见的镜像。
// agent_type 未被 group_config_bindings 限制的视为全局可见。
func HasGlobalImage(ctx context.Context) bool {
	// 查出所有被限制的 agent_type
	restricted, err := model.GetRestrictedImageTypes(ctx)
	if err != nil {
		return false
	}

	q := model.DB(ctx).Model(&model.AIImage{}).Where("enabled = ?", true)
	if len(restricted) > 0 {
		// 排除被限制的 agent_type，只看全局可见的
		q = q.Where("agent_type NOT IN ?", restricted)
	}

	var count int64
	q.Count(&count)
	return count > 0
}

// ── 按组祖先链检查绑定 ──

// batchGetAncestorsForHealth 批量获取所有组的祖先链（含自己，近→远）。
func batchGetAncestorsForHealth(ctx context.Context, groupIDs []uint) map[uint][]uint {
	result := make(map[uint][]uint, len(groupIDs))
	if len(groupIDs) == 0 {
		return result
	}
	type row struct {
		Descendant uint `gorm:"column:descendant_id"`
		Ancestor   uint `gorm:"column:ancestor_id"`
	}
	var rows []row
	model.DB(ctx).Table("group_closure").
		Select("descendant_id, ancestor_id").
		Where("descendant_id IN ?", groupIDs).
		Order("descendant_id ASC, depth ASC").
		Find(&rows)
	for _, r := range rows {
		result[r.Descendant] = append(result[r.Descendant], r.Ancestor)
	}
	for _, gid := range groupIDs {
		if _, ok := result[gid]; !ok {
			result[gid] = []uint{gid}
		}
	}
	return result
}

// hasModelBinding 检查祖先链上是否有模型绑定（旧表）
func hasModelBinding(ctx context.Context, ancestors []uint) bool {
	if len(ancestors) == 0 {
		return false
	}
	var count int64
	model.DB(ctx).Table("model_visibility_groups").
		Where("group_id IN ?", ancestors).
		Count(&count)
	return count > 0
}

// hasChannelBinding 检查祖先链上是否有通道绑定（新表）
func hasChannelBinding(ctx context.Context, ancestors []uint) bool {
	if len(ancestors) == 0 {
		return false
	}
	var count int64
	model.DB(ctx).Model(&model.GroupConfigBinding{}).
		Where("config_type = ? AND group_id IN ?", model.ConfigTypeChannel, ancestors).
		Count(&count)
	return count > 0
}

// hasImageTypeBinding 检查祖先链上是否有镜像类型绑定（新表）
func hasImageTypeBinding(ctx context.Context, ancestors []uint) bool {
	if len(ancestors) == 0 {
		return false
	}
	var count int64
	model.DB(ctx).Model(&model.GroupConfigBinding{}).
		Where("config_type = ? AND group_id IN ?", model.ConfigTypeImageType, ancestors).
		Count(&count)
	return count > 0
}

// hasVpcBinding 检查祖先链上是否有 VPC 绑定（新表）
func hasVpcBinding(ctx context.Context, ancestors []uint) bool {
	if len(ancestors) == 0 {
		return false
	}
	var count int64
	model.DB(ctx).Model(&model.GroupConfigBinding{}).
		Where("config_type = ? AND group_id IN ?", model.ConfigTypeVPC, ancestors).
		Count(&count)
	return count > 0
}

package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type projectMembersRequest struct {
	ProjectID uint   `json:"id"`
	UserIDs   []uint `json:"user_ids"`
}

const maxProjectUserIDsPerRequest = 100

type projectListItem struct {
	model.Project
	MemberCount        *int64 `json:"member_count,omitempty"`
	CloudInstanceCount *int64 `json:"cloud_instance_count,omitempty"`
	LocalAgentCount    *int64 `json:"local_agent_count,omitempty"`
	WorkspaceCount     *int64 `json:"workspace_count,omitempty"`
	AssetCount         *int64 `json:"asset_count,omitempty"`
}

func requireProject(tx *gorm.DB, projectID uint) (*model.Project, error) {
	var project model.Project
	if err := tx.Where("id = ?", projectID).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func HandleAdminProjects(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	page, pageSize := parsePagination(r)
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	withCounts := parseBoolQueryDefault(r, "with_counts", true)
	db := model.DB(r.Context()).Model(&model.Project{})
	if query != "" {
		db = db.Where("name LIKE ?", "%"+query+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	var projects []model.Project
	projectQuery := db.Order("updated_at DESC, id DESC")
	if withCounts {
		projectQuery = projectQuery.Offset((page - 1) * pageSize).Limit(pageSize)
	} else {
		// 应用范围选择器与分组树一致：关闭计数时一次返回全部项目，避免前端拼接分页。
		page = 1
		pageSize = int(total)
	}
	if err := projectQuery.Find(&projects).Error; err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	items := make([]projectListItem, len(projects))
	for i := range projects {
		items[i].Project = projects[i]
	}
	if withCounts {
		if err := populateProjectCounts(model.DB(r.Context()), items); err != nil {
			writeProjectDBError(w, r, err)
			return
		}
	}
	jsonOK(w, map[string]any{"ok": true, "projects": items, "total": total, "page": page, "page_size": pageSize})
}

func populateProjectCounts(db *gorm.DB, projects []projectListItem) error {
	if len(projects) == 0 {
		return nil
	}
	projectIDs := make([]uint, len(projects))
	for i := range projects {
		projectIDs[i] = projects[i].ID
	}
	type countRow struct {
		ProjectID uint  `gorm:"column:project_id"`
		Count     int64 `gorm:"column:count"`
	}
	type scopeCountRow struct {
		ProjectID      uint  `gorm:"column:project_id"`
		WorkspaceCount int64 `gorm:"column:workspace_count"`
		InstanceCount  int64 `gorm:"column:instance_count"`
	}
	memberCounts := make(map[uint]int64, len(projects))
	assetCounts := make(map[uint]int64, len(projects))
	workspaceCounts := make(map[uint]int64, len(projects))
	instanceCounts := make(map[uint]int64, len(projects))
	var memberRows []countRow
	if err := db.Model(&model.ProjectMember{}).Select("project_id, COUNT(*) AS count").
		Where("project_id IN ?", projectIDs).Group("project_id").Scan(&memberRows).Error; err != nil {
		return err
	}
	for _, row := range memberRows {
		memberCounts[row.ProjectID] = row.Count
	}
	var scopeRows []scopeCountRow
	if err := db.Model(&model.LocalAgentScopeBinding{}).
		Select("project_id, COUNT(*) AS workspace_count, COUNT(DISTINCT instance_id) AS instance_count").
		Where("scope = ? AND project_id IN ?", model.LocalAgentScopeWorkspace, projectIDs).
		Group("project_id").Scan(&scopeRows).Error; err != nil {
		return err
	}
	for _, row := range scopeRows {
		workspaceCounts[row.ProjectID], instanceCounts[row.ProjectID] = row.WorkspaceCount, row.InstanceCount
	}
	var assetRows []countRow
	if err := db.Model(&model.ProjectConfigBinding{}).Select("project_id, COUNT(*) AS count").
		Where("project_id IN ? AND config_type IN ?", projectIDs, model.ProjectAssetConfigTypes).
		Group("project_id").Scan(&assetRows).Error; err != nil {
		return err
	}
	for _, row := range assetRows {
		assetCounts[row.ProjectID] = row.Count
	}
	for i := range projects {
		memberCount := memberCounts[projects[i].ID]
		workspaceCount := workspaceCounts[projects[i].ID]
		localAgentCount := instanceCounts[projects[i].ID]
		assetCount := assetCounts[projects[i].ID]
		cloudInstanceCount := int64(0)
		projects[i].MemberCount, projects[i].CloudInstanceCount = &memberCount, &cloudInstanceCount
		projects[i].LocalAgentCount, projects[i].WorkspaceCount, projects[i].AssetCount = &localAgentCount, &workspaceCount, &assetCount
	}
	return nil
}

func HandleAdminProjectCreate(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	name, err := model.NormalizeProjectName(req.Name)
	if err != nil {
		writeProjectNameError(w, r, err)
		return
	}
	project := model.Project{Name: name, Description: strings.TrimSpace(req.Description), CreatedBy: projectActorID(r)}
	if err := model.DB(r.Context()).Create(&project).Error; err != nil {
		writeProjectConflictOrDBError(w, r, err)
		return
	}
	jsonOK(w, map[string]any{"ok": true, "project": projectListItem{Project: project}})
}

func HandleAdminProjectUpdate(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	var req struct {
		ID          uint    `json:"id"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.Name == nil && req.Description == nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	updates := make(map[string]any, 2)
	if req.Name != nil {
		name, err := model.NormalizeProjectName(*req.Name)
		if err != nil {
			writeProjectNameError(w, r, err)
			return
		}
		updates["name"] = name
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	db := model.DB(r.Context())
	result := db.Model(&model.Project{}).Where("id = ?", req.ID).Updates(updates)
	if result.Error != nil {
		writeProjectConflictOrDBError(w, r, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgProjectNotFound))
		return
	}
	project, err := requireProject(db, req.ID)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	jsonOK(w, map[string]any{"ok": true, "project": projectListItem{Project: *project}})
}

func projectIDsFromRequest(r *http.Request) ([]uint, error) {
	if ids := strings.TrimSpace(r.URL.Query().Get("project_ids")); ids != "" {
		return parseUintCSV(ids)
	}
	var req struct {
		ID         uint   `json:"id"`
		ProjectIDs []uint `json:"project_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	if req.ID != 0 {
		return []uint{req.ID}, nil
	}
	return uniqueUintIDs(req.ProjectIDs), nil
}

func uniqueUintIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func HandleAdminProjectDeleteImpact(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	ids, err := projectIDsFromRequest(r)
	if err != nil || len(ids) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "project_ids"))
		return
	}
	var bindings []model.ProjectConfigBinding
	if err := model.DB(r.Context()).Where("project_id IN ? AND config_type IN ?", ids, model.ProjectVisibilityConfigTypes).Find(&bindings).Error; err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	byProject := make(map[uint][]model.ProjectConfigBinding)
	for _, binding := range bindings {
		byProject[binding.ProjectID] = append(byProject[binding.ProjectID], binding)
	}
	var projects []model.Project
	if err := model.DB(r.Context()).Where("id IN ?", ids).Find(&projects).Error; err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	projectByID := make(map[uint]model.Project, len(projects))
	for _, project := range projects {
		projectByID[project.ID] = project
	}
	results := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		project, exists := projectByID[id]
		if !exists {
			continue
		}
		resources := map[string][]string{model.AssetTypeSkill: {}, model.AssetTypeRule: {}}
		for _, binding := range byProject[id] {
			if binding.ConfigType == model.ProjectConfigTypeSkill {
				resources[model.AssetTypeSkill] = append(resources[model.AssetTypeSkill], binding.ConfigKey)
			} else if binding.ConfigType == model.ProjectConfigTypeRule {
				resources[model.AssetTypeRule] = append(resources[model.AssetTypeRule], binding.ConfigKey)
			}
		}
		results = append(results, map[string]any{"project": map[string]any{"id": id, "name": project.Name}, "can_delete": len(byProject[id]) == 0, "blockers": map[string]any{"resource_bindings": resources}})
	}
	jsonOK(w, map[string]any{"results": results})
}

func HandleAdminProjectDelete(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	ids, err := projectIDsFromRequest(r)
	if err != nil || len(ids) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "project_ids"))
		return
	}
	blocked := false
	err = model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.ProjectConfigBinding{}).Where("project_id IN ? AND config_type IN ?", ids, model.ProjectVisibilityConfigTypes).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			blocked = true
			return gorm.ErrInvalidData
		}
		if err := tx.Where("project_id IN ? AND config_type IN ?", ids, model.ProjectAssetConfigTypes).Delete(&model.ProjectConfigBinding{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id IN ?", ids).Delete(&model.ProjectMember{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&model.Project{}).Error
	})
	if blocked {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgProjectHasDependencies).WithCustomData(map[string]any{"reason": "has_dependencies"}))
		return
	}
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	jsonOK(w, map[string]any{"ok": true})
}

func HandleAdminProjectMembers(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	projectID, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	if projectID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "id"))
		return
	}
	page, pageSize := parsePagination(r)
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	db := model.DB(r.Context())
	project, err := requireProject(db, uint(projectID))
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	type memberRow struct {
		UserID    uint           `gorm:"column:user_id"`
		Username  string         `gorm:"column:username"`
		Role      string         `gorm:"column:role"`
		DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
		JoinedAt  time.Time      `gorm:"column:joined_at"`
	}
	memberQuery := db.Unscoped().Model(&model.User{}).
		Joins("JOIN project_members ON project_members.user_id = users.id").
		Where("project_members.project_id = ?", uint(projectID))
	if query != "" {
		memberQuery = memberQuery.Where("users.username LIKE ?", "%"+query+"%")
	}
	var total int64
	if err := memberQuery.Count(&total).Error; err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	var rows []memberRow
	if err := memberQuery.Select(`users.id AS user_id, users.username, users.role, users.deleted_at,
		project_members.created_at AS joined_at`).
		Order("project_members.created_at DESC, project_members.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	userIDs := make([]uint, len(rows))
	for i, member := range rows {
		userIDs[i] = member.UserID
	}
	projectsByUser, err := getUserProjectsByUserIDs(r.Context(), userIDs)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, member := range rows {
		workspaceCount, err := projectWorkspaceCountForUser(db, uint(projectID), member.UserID)
		if err != nil {
			writeProjectDBError(w, r, err)
			return
		}
		items = append(items, map[string]any{"user_id": member.UserID, "username": member.Username, "role": member.Role, "deleted_at": member.DeletedAt, "joined_at": member.JoinedAt, "projects": projectsByUser[member.UserID], "cloud_instance_count": 0, "local_workspace_count": workspaceCount})
	}
	jsonOK(w, map[string]any{"ok": true, "project": map[string]any{"id": project.ID, "name": project.Name}, "members": items, "total": total, "page": page, "page_size": pageSize})
}

func projectWorkspaceCountForUser(db *gorm.DB, projectID, userID uint) (int64, error) {
	var instanceIDs []uint
	if err := db.Model(&model.Instance{}).Where("user_id = ? AND source = ?", userID, model.InstanceSourceLocal).Pluck("id", &instanceIDs).Error; err != nil {
		return 0, err
	}
	if len(instanceIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := db.Model(&model.LocalAgentScopeBinding{}).Where("scope = ? AND project_id = ? AND instance_id IN ?", model.LocalAgentScopeWorkspace, projectID, instanceIDs).Count(&count).Error
	return count, err
}

func decodeProjectMembersRequest(w http.ResponseWriter, r *http.Request) (*projectMembersRequest, bool) {
	var req projectMembersRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.ProjectID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return nil, false
	}
	req.UserIDs = uniqueUintIDs(req.UserIDs)
	return &req, true
}

func validateProjectUsers(tx *gorm.DB, projectID uint, userIDs []uint) error {
	if _, err := requireProject(tx, projectID); err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return nil
	}
	var count int64
	if err := tx.Model(&model.User{}).Where("id IN ?", userIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(userIDs)) {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func HandleAdminProjectMembersSet(w http.ResponseWriter, r *http.Request) {
	handleProjectMembersWrite(w, r, "set")
}

func HandleAdminProjectMembersAdd(w http.ResponseWriter, r *http.Request) {
	handleProjectMembersWrite(w, r, "add")
}

func HandleAdminProjectMembersRemove(w http.ResponseWriter, r *http.Request) {
	handleProjectMembersWrite(w, r, "remove")
}

func handleProjectMembersWrite(w http.ResponseWriter, r *http.Request, action string) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	req, ok := decodeProjectMembersRequest(w, r)
	if !ok {
		return
	}
	err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := validateProjectUsers(tx, req.ProjectID, req.UserIDs); err != nil {
			return err
		}
		switch action {
		case "set":
			return model.ReplaceProjectMembers(tx, req.ProjectID, req.UserIDs, projectActorID(r))
		case "remove":
			return tx.Where("project_id = ? AND user_id IN ?", req.ProjectID, req.UserIDs).Delete(&model.ProjectMember{}).Error
		case "add":
			records := make([]model.ProjectMember, 0, len(req.UserIDs))
			for _, id := range req.UserIDs {
				records = append(records, model.ProjectMember{ProjectID: req.ProjectID, UserID: id, CreatedBy: projectActorID(r)})
			}
			if len(records) == 0 {
				return nil
			}
			return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error
		default:
			return errors.New("unsupported project member action")
		}
	})
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	var memberCount int64
	if err := model.DB(r.Context()).Model(&model.ProjectMember{}).Where("project_id = ?", req.ProjectID).Count(&memberCount).Error; err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	jsonOK(w, map[string]any{"ok": true, "member_count": memberCount})
}

// projectActorID 兼容管理员 Bearer token（无用户 ID）与登录管理员会话。
func projectActorID(r *http.Request) uint {
	if user, _ := getUserFromToken(r); user != nil {
		return user.ID
	}
	if user, _ := getLoginUser(r); user != nil {
		return user.ID
	}
	return 0
}

func HandleAdminProjectsByUsers(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	userIDs, ok := projectUserIDsFromRequest(r)
	if !ok {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "user_ids"))
		return
	}
	var members []model.ProjectMember
	if err := model.DB(r.Context()).Where("user_id IN ?", userIDs).Order("created_at ASC, id ASC").Find(&members).Error; err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	projectIDs := make([]uint, 0, len(members))
	for _, member := range members {
		projectIDs = append(projectIDs, member.ProjectID)
	}
	var projects []model.Project
	if len(projectIDs) > 0 {
		if err := model.DB(r.Context()).Where("id IN ?", uniqueUintIDs(projectIDs)).Find(&projects).Error; err != nil {
			writeProjectDBError(w, r, err)
			return
		}
	}
	byID := make(map[uint]model.Project)
	for _, project := range projects {
		byID[project.ID] = project
	}
	result := make(map[uint][]model.Project, len(userIDs))
	for _, userID := range userIDs {
		result[userID] = []model.Project{}
	}
	for _, member := range members {
		if project, ok := byID[member.ProjectID]; ok {
			result[member.UserID] = append(result[member.UserID], project)
		}
	}
	jsonOK(w, map[string]any{"ok": true, "data": result})
}

func projectUserIDsFromRequest(r *http.Request) ([]uint, bool) {
	if r.Method == http.MethodGet {
		ids, err := parseUintCSV(r.URL.Query().Get("user_ids"))
		return ids, err == nil && len(ids) > 0 && len(ids) <= maxProjectUserIDsPerRequest
	}
	var req struct {
		UserIDs []uint `json:"user_ids"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		return nil, false
	}
	ids := uniqueUintIDs(req.UserIDs)
	return ids, len(ids) > 0 && len(ids) <= maxProjectUserIDsPerRequest
}

func HandleProjectsMine(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	projects, err := model.ListUserProjects(r.Context(), user.ID)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	includeSummary := parseBoolQuery(r, "include_asset_summary")
	filteredProjects := make([]model.Project, 0, len(projects))
	for _, project := range projects {
		if query != "" && !strings.Contains(strings.ToLower(project.Name), query) {
			continue
		}
		filteredProjects = append(filteredProjects, project)
	}
	bindingsByProject := make(map[uint][]model.ProjectConfigBinding)
	if includeSummary && len(filteredProjects) > 0 {
		projectIDs := make([]uint, len(filteredProjects))
		for i := range filteredProjects {
			projectIDs[i] = filteredProjects[i].ID
		}
		var bindings []model.ProjectConfigBinding
		if err := model.DB(r.Context()).Where("project_id IN ? AND config_type IN ?", projectIDs,
			model.ProjectAssetConfigTypes).Find(&bindings).Error; err != nil {
			writeProjectDBError(w, r, err)
			return
		}
		for _, binding := range bindings {
			bindingsByProject[binding.ProjectID] = append(bindingsByProject[binding.ProjectID], binding)
		}
	}
	items := make([]map[string]any, 0, len(filteredProjects))
	for _, project := range filteredProjects {
		item := map[string]any{"id": project.ID, "name": project.Name, "description": project.Description}
		if includeSummary {
			for _, binding := range bindingsByProject[project.ID] {
				if binding.ConfigType == model.AssetBindingTypeSkill {
					item["skill_count"] = itemInt(item, "skill_count") + 1
				} else {
					item["rule_count"] = itemInt(item, "rule_count") + 1
				}
			}
		}
		items = append(items, item)
	}
	jsonOK(w, map[string]any{"ok": true, "projects": items})
}

func itemInt(item map[string]any, key string) int { value, _ := item[key].(int); return value }

func writeProjectConflictOrDBError(w http.ResponseWriter, r *http.Request, err error) {
	if isDuplicateKeyError(err) {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgProjectNameConflict).WithCustomData(map[string]any{"reason": "name_conflict"}))
		return
	}
	writeProjectDBError(w, r, err)
}

func writeProjectNameError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, model.ErrInvalidProjectName) {
		writeError(w, r, http.StatusBadRequest, model.ErrInvalidProjectName)
		return
	}
	writeProjectDBError(w, r, err)
}

func writeProjectDBError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgProjectNotFound))
		return
	}
	writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
}

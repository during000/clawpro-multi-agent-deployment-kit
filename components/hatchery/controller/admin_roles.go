package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// ── 管控端角色 CRUD ────────────────────────────────────────────────

// maxRoleNameRunes 角色名称最大字符数
const maxRoleNameRunes = 30

// HandleAdminRoles 角色列表
// GET /admin/roles
func HandleAdminRoles(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var roles []model.OpenClawRole
	db := model.DB(r.Context()).Model(&model.OpenClawRole{})

	// 应用范围筛选：
	// 1. 只传 visibility_type=all → 仅全局可见
	// 2. 只传 group_id → 仅匹配分组可见
	// 3. visibility_type=all + group_id → 全局 + 匹配分组
	// group_id 支持逗号分隔多个，如 group_id=1,3
	vtFilter := r.URL.Query().Get("visibility_type")
	var parsedGIDs []int
	if gidStr := r.URL.Query().Get("group_id"); gidStr != "" {
		for _, s := range strings.Split(gidStr, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && id > 0 {
				parsedGIDs = append(parsedGIDs, id)
			}
		}
	}
	if vtFilter != "" && len(parsedGIDs) > 0 {
		subQ := model.DB(r.Context()).Model(&model.RoleVisibilityGroup{}).Select("open_claw_role_id").Where("group_id IN ?", parsedGIDs)
		db = db.Where("visibility_type = ? OR id IN (?)", vtFilter, subQ)
	} else if vtFilter != "" {
		db = db.Where("visibility_type = ?", vtFilter)
	} else if len(parsedGIDs) > 0 {
		subQ := model.DB(r.Context()).Model(&model.RoleVisibilityGroup{}).Select("open_claw_role_id").Where("group_id IN ?", parsedGIDs)
		db = db.Where("id IN (?)", subQ)
	}

	db.Order("sort_order asc, id asc").Find(&roles)

	// 批量查询所有角色的技能
	roleIDs := make([]uint, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID
	}

	var allSkills []model.OpenClawRoleSkill
	if len(roleIDs) > 0 {
		model.DB(r.Context()).Where("open_claw_role_id IN ?", roleIDs).Find(&allSkills)
	}

	// 批量查询所有角色的插件
	var allPlugins []model.OpenClawRolePlugin
	if len(roleIDs) > 0 {
		model.DB(r.Context()).Where("open_claw_role_id IN ?", roleIDs).Find(&allPlugins)
	}

	// 按角色 ID 分组
	skillMap := make(map[uint][]model.OpenClawRoleSkill)
	for _, skill := range allSkills {
		skillMap[skill.OpenClawRoleID] = append(skillMap[skill.OpenClawRoleID], skill)
	}

	pluginMap := make(map[uint][]model.OpenClawRolePlugin)
	for _, plugin := range allPlugins {
		pluginMap[plugin.OpenClawRoleID] = append(pluginMap[plugin.OpenClawRoleID], plugin)
	}

	// 组装响应（含可见性分组信息）
	visibilityMap := buildRoleVisibilityData(r.Context(), roles)

	type roleWithSkillsAndPlugins struct {
		model.OpenClawRole
		Skills        []model.OpenClawRoleSkill  `json:"skills"`
		Plugins       []model.OpenClawRolePlugin `json:"plugins"`
		VisibleGroups []visibilityGroupInfo      `json:"visible_groups"`
	}

	result := make([]roleWithSkillsAndPlugins, len(roles))
	for i, role := range roles {
		skills := skillMap[role.ID]
		if skills == nil {
			skills = []model.OpenClawRoleSkill{}
		}
		plugins := pluginMap[role.ID]
		if plugins == nil {
			plugins = []model.OpenClawRolePlugin{}
		}
		groups := visibilityMap[role.ID]
		if groups == nil {
			groups = []visibilityGroupInfo{}
		}
		result[i] = roleWithSkillsAndPlugins{
			OpenClawRole:  role,
			Skills:        skills,
			Plugins:       plugins,
			VisibleGroups: groups,
		}
	}

	jsonOK(w, map[string]interface{}{
		"roles": result,
		"total": len(result),
	})
}

// HandleCreateRole 新增角色
// POST /admin/roles/create
func HandleCreateRole(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		Soul           string `json:"soul"`
		Visible        *bool  `json:"visible"`         // 指针类型，区分未传和传 false
		VisibilityType string `json:"visibility_type"` // all 或 group
		GroupIDs       []uint `json:"group_ids"`       // 分组 ID 列表
		Version        string `json:"version"`         // 角色版本号 X.Y，缺省 "1.0"
		Skills         []struct {
			Name    string `json:"name"`
			Slug    string `json:"slug"`
			Version string `json:"version"`
			Source  string `json:"source"`
		} `json:"skills"`
		Plugins []struct {
			Name        string `json:"name"`
			Slug        string `json:"slug"`
			PluginID    string `json:"plugin_id"`
			Version     string `json:"version"`
			Source      string `json:"source"`
			NpmPackage  string `json:"npm_package"`
			InstallMode string `json:"install_mode"`
			Kind        string `json:"kind"`
		} `json:"plugins"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgBadRequest))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgRoleNameCannotBeEmpty))
		return
	}
	if utf8.RuneCountInString(req.Name) > maxRoleNameRunes {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgRoleNameTooLong, maxRoleNameRunes))
		return
	}

	// 角色版本号：默认 1.0；非空则严格校验 X.Y 格式
	req.Version = strings.TrimSpace(req.Version)
	if req.Version == "" {
		req.Version = "1.0"
	}
	if err := validateRoleVersionFormat(req.Version); err != nil {
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}

	// 处理可见范围参数
	visibilityType := req.VisibilityType
	if visibilityType == "" {
		visibilityType = "all"
	}
	if err := validateVisibilityInput(r.Context(), visibilityType, req.GroupIDs); err != nil {
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}

	visible := true
	if req.Visible != nil {
		visible = *req.Visible
	}

	var role model.OpenClawRole
	err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 唯一性检查
		var existing model.OpenClawRole
		if tx.Where("name = ?", req.Name).First(&existing).Error == nil {
			return fmt.Errorf("conflict")
		}

		// 新角色排在最前面：所有现有角色 sort_order + 1
		tx.Model(&model.OpenClawRole{}).Where("1 = 1").UpdateColumn("sort_order", gorm.Expr("sort_order + 1"))

		role = model.OpenClawRole{
			Name:           req.Name,
			Description:    req.Description,
			Soul:           req.Soul,
			Visible:        visible,
			SortOrder:      0,
			VisibilityType: visibilityType,
			Version:        req.Version,
		}
		if err := tx.Create(&role).Error; err != nil {
			return common.I18nRichError(err, i18n.MsgKeyRoleCreateFailed)
		}

		// 设置可见性关联
		if visibilityType == usergroup.VisibilityGroup && len(req.GroupIDs) > 0 {
			if err := model.SetRoleVisibility(tx, role.ID, visibilityType, req.GroupIDs); err != nil {
				return err
			}
		}

		// 创建关联技能
		for _, s := range req.Skills {
			skill := model.OpenClawRoleSkill{
				OpenClawRoleID: role.ID,
				Name:           s.Name,
				Slug:           s.Slug,
				Version:        s.Version,
				Source:         s.Source,
			}
			if skill.Source == "" {
				skill.Source = "public"
			}
			if err := tx.Create(&skill).Error; err != nil {
				return common.I18nRichError(err, i18n.MsgKeyRoleSkillCreateFailed, s.Slug)
			}
		}

		// 创建关联插件
		for _, p := range req.Plugins {
			plugin := model.OpenClawRolePlugin{
				OpenClawRoleID: role.ID,
				Name:           p.Name,
				Slug:           p.Slug,
				PluginID:       p.PluginID,
				Version:        p.Version,
				Source:         p.Source,
				NpmPackage:     p.NpmPackage,
				InstallMode:    p.InstallMode,
				Kind:           p.Kind,
			}
			if plugin.Source == "" {
				plugin.Source = "enterprise"
			}
			if plugin.InstallMode == "" {
				plugin.InstallMode = "smh"
			}
			if err := tx.Create(&plugin).Error; err != nil {
				return common.I18nRichError(err, i18n.MsgKeyRolePluginCreateFailed, p.Slug)
			}
		}

		return nil
	})

	if err != nil {
		if err.Error() == "conflict" {
			writeError(w, r, http.StatusConflict, common.I18nError(i18n.MsgSameRoleExists))
			return
		}
		var richErr *common.RichError
		if errors.As(err, &richErr) {
			writeError(w, r, http.StatusInternalServerError, richErr)
		} else {
			writeError(w, r, http.StatusInternalServerError, common.I18nRichError(err, i18n.MsgKeyRoleCreateFailed))
		}
		return
	}

	// 异步同步角色技能的 CosZipKey（下载 zip 并上传到 common space）
	go syncRoleSkillsCosZipKey(common.DetachContext(r.Context()), role.ID)

	jsonOK(w, map[string]interface{}{"ok": true, "id": role.ID})
}

// HandleUpdateRole 编辑角色
// POST /admin/roles/update?id=xxx
func HandleUpdateRole(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgMissingParamID))
		return
	}

	var req struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		Soul           string `json:"soul"`
		Visible        *bool  `json:"visible"`
		VisibilityType string `json:"visibility_type"`
		GroupIDs       []uint `json:"group_ids"`
		Version        string `json:"version"` // 角色版本号 X.Y，必须严格大于旧版本号
		Skills         []struct {
			Name    string `json:"name"`
			Slug    string `json:"slug"`
			Version string `json:"version"`
			Source  string `json:"source"`
		} `json:"skills"`
		Plugins []struct {
			Name        string `json:"name"`
			Slug        string `json:"slug"`
			PluginID    string `json:"plugin_id"`
			Version     string `json:"version"`
			Source      string `json:"source"`
			NpmPackage  string `json:"npm_package"`
			InstallMode string `json:"install_mode"`
			Kind        string `json:"kind"`
		} `json:"plugins"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgBadRequest))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgRoleNameCannotBeEmpty))
		return
	}
	if utf8.RuneCountInString(req.Name) > maxRoleNameRunes {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgRoleNameTooLong, maxRoleNameRunes))
		return
	}

	// 角色版本号：未传则保持原版本不变；传了必须 X.Y 格式
	req.Version = strings.TrimSpace(req.Version)
	if req.Version != "" {
		if err := validateRoleVersionFormat(req.Version); err != nil {
			writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
			return
		}
	}

	// 处理可见范围参数（如果传了 visibility_type）
	hasVisibility := req.VisibilityType != ""
	visibilityType := req.VisibilityType
	if hasVisibility {
		if err := validateVisibilityInput(r.Context(), visibilityType, req.GroupIDs); err != nil {
			writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
			return
		}
	}

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var role model.OpenClawRole
		if tx.First(&role, id).Error != nil {
			return fmt.Errorf("not_found")
		}

		// 名称唯一性检查（排除自身）
		var existing model.OpenClawRole
		if tx.Where("name = ? AND id != ?", req.Name, id).First(&existing).Error == nil {
			return fmt.Errorf("conflict")
		}

		// 版本号校验：传了就必须严格大于旧版本
		if req.Version != "" && compareRoleVersions(req.Version, role.Version) <= 0 {
			return common.I18nRichError(errVersionNotHigher, i18n.MsgRoleVersionMustBeHigher, role.Version)
		}
		oldVersion := role.Version // 保存旧版本号，Updates 后 role.Version 可能被 GORM 回写

		// 更新角色基本信息
		updates := map[string]interface{}{
			"name":        req.Name,
			"description": req.Description,
			"soul":        req.Soul,
		}
		if req.Visible != nil {
			updates["visible"] = *req.Visible
		}
		if req.Version != "" {
			updates["version"] = req.Version
		}
		if err := tx.Model(&role).Updates(updates).Error; err != nil {
			return common.I18nRichError(err, i18n.MsgKeyRoleUpdateFailed)
		}

		// 版本升级联动：如果 version 字段变了，把绑定该角色且状态为 updated / 空串 的实例翻回 pending
		// failed 保留（管理员先修问题再升级）
		// updating 保留（避免打断进行中的下发）
		if req.Version != "" && req.Version != oldVersion {
			if err := tx.Model(&model.Instance{}).
				Where("role_id = ? AND role_sync_status IN ?", id,
					[]string{model.RoleSyncStatusEmpty, model.RoleSyncStatusUpdated}).
				Update("role_sync_status", model.RoleSyncStatusPending).Error; err != nil {
				return common.I18nRichError(err, i18n.MsgKeyRoleUpdateFailed)
			}
		}

		// 更新可见范围（如果传了 visibility_type）
		if hasVisibility {
			if err := model.SetRoleVisibility(tx, uint(id), visibilityType, req.GroupIDs); err != nil {
				return err
			}
		}

		// 技能全量替换：删除旧技能 → 创建新技能
		if err := tx.Where("open_claw_role_id = ?", id).Delete(&model.OpenClawRoleSkill{}).Error; err != nil {
			return common.I18nRichError(err, i18n.MsgKeyRoleDeleteOldSkillFailed)
		}
		for _, s := range req.Skills {
			skill := model.OpenClawRoleSkill{
				OpenClawRoleID: uint(id),
				Name:           s.Name,
				Slug:           s.Slug,
				Version:        s.Version,
				Source:         s.Source,
			}
			if skill.Source == "" {
				skill.Source = "public"
			}
			if err := tx.Create(&skill).Error; err != nil {
				return common.I18nRichError(err, i18n.MsgKeyRoleSkillCreateFailed, s.Slug)
			}
		}

		// 插件全量替换：删除旧插件 → 创建新插件
		if err := tx.Where("open_claw_role_id = ?", id).Delete(&model.OpenClawRolePlugin{}).Error; err != nil {
			return common.I18nRichError(err, i18n.MsgKeyRoleDeleteOldPluginFailed)
		}
		for _, p := range req.Plugins {
			plugin := model.OpenClawRolePlugin{
				OpenClawRoleID: uint(id),
				Name:           p.Name,
				Slug:           p.Slug,
				PluginID:       p.PluginID,
				Version:        p.Version,
				Source:         p.Source,
				NpmPackage:     p.NpmPackage,
				InstallMode:    p.InstallMode,
				Kind:           p.Kind,
			}
			if plugin.Source == "" {
				plugin.Source = "enterprise"
			}
			if plugin.InstallMode == "" {
				plugin.InstallMode = "smh"
			}
			if err := tx.Create(&plugin).Error; err != nil {
				return common.I18nRichError(err, i18n.MsgKeyRolePluginCreateFailed, p.Slug)
			}
		}

		return nil
	})

	if txErr != nil {
		switch txErr.Error() {
		case "not_found":
			writeError(w, r, http.StatusNotFound, common.I18nError(i18n.MsgRoleNotFound))
		case "conflict":
			writeError(w, r, http.StatusConflict, common.I18nError(i18n.MsgSameRoleExists))
		default:
			var richErr *common.RichError
			if errors.As(txErr, &richErr) {
				// 版本号校验失败属于客户端错误，单独返回 400
				if errors.Is(txErr, errVersionNotHigher) {
					writeError(w, r, http.StatusBadRequest, richErr)
				} else {
					writeError(w, r, http.StatusInternalServerError, richErr)
				}
			} else {
				writeError(w, r, http.StatusInternalServerError, common.I18nRichError(txErr, i18n.MsgKeyRoleUpdateFailed))
			}
		}
		return
	}

	// 异步同步新增技能的 CosZipKey
	go syncRoleSkillsCosZipKey(common.DetachContext(r.Context()), uint(id))

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleDeleteRole 删除角色（硬删除）
// POST /admin/roles/delete?id=xxx
func HandleDeleteRole(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgMissingParamID))
		return
	}

	var role model.OpenClawRole
	if model.DB(r.Context()).First(&role, id).Error != nil {
		writeError(w, r, http.StatusNotFound, common.I18nError(i18n.MsgRoleNotFound))
		return
	}

	// 事务内级联删除：角色 + 关联技能 + 关联插件 + 可见性关联
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("open_claw_role_id = ?", role.ID).Delete(&model.OpenClawRoleSkill{}).Error; err != nil {
			return err
		}
		if err := tx.Where("open_claw_role_id = ?", role.ID).Delete(&model.OpenClawRolePlugin{}).Error; err != nil {
			return err
		}
		if err := model.CleanupRoleVisibilityByRoleID(tx, role.ID); err != nil {
			return err
		}
		return tx.Delete(&role).Error
	}); err != nil {
		writeError(w, r, http.StatusInternalServerError, common.I18nRichError(err, i18n.MsgDeleteRoleFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleToggleRoleVisible 切换角色可见性
// POST /admin/roles/toggle-visible?id=xxx
func HandleToggleRoleVisible(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgMissingParamID))
		return
	}

	var role model.OpenClawRole
	if model.DB(r.Context()).First(&role, id).Error != nil {
		writeError(w, r, http.StatusNotFound, common.I18nError(i18n.MsgRoleNotFound))
		return
	}

	if err := model.DB(r.Context()).Model(&role).Update("visible", !role.Visible).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, common.I18nRichError(err, i18n.MsgToggleVisibilityFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true, "visible": !role.Visible})
}

// HandleReorderRoles 批量更新角色排序
// POST /admin/roles/reorder
func HandleReorderRoles(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgBadRequest))
		return
	}

	if len(req.IDs) == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgIDsCannotBeEmpty))
		return
	}

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		for i, id := range req.IDs {
			if err := tx.Model(&model.OpenClawRole{}).Where("id = ?", id).Update("sort_order", i).Error; err != nil {
				return common.I18nRichError(err, i18n.MsgKeyRoleReorderFailed, id)
			}
		}
		return nil
	})

	if txErr != nil {
		var richErr *common.RichError
		if errors.As(txErr, &richErr) {
			writeError(w, r, http.StatusInternalServerError, richErr)
		} else {
			writeError(w, r, http.StatusInternalServerError, common.I18nRichError(txErr, i18n.MsgDeleteRoleFailed))
		}
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleRoleDetail 角色详情（含技能列表）
// GET /admin/roles/detail?id=xxx
func HandleRoleDetail(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgMissingParamID))
		return
	}

	var role model.OpenClawRole
	if model.DB(r.Context()).First(&role, id).Error != nil {
		writeError(w, r, http.StatusNotFound, common.I18nError(i18n.MsgRoleNotFound))
		return
	}

	var skills []model.OpenClawRoleSkill
	model.DB(r.Context()).Where("open_claw_role_id = ?", role.ID).Find(&skills)
	if skills == nil {
		skills = []model.OpenClawRoleSkill{}
	}

	var plugins []model.OpenClawRolePlugin
	model.DB(r.Context()).Where("open_claw_role_id = ?", role.ID).Find(&plugins)
	if plugins == nil {
		plugins = []model.OpenClawRolePlugin{}
	}

	// 查询可见性分组信息
	visGroups := []visibilityGroupInfo{}
	if role.VisibilityType == usergroup.VisibilityGroup {
		visMap := buildRoleVisibilityData(r.Context(), []model.OpenClawRole{role})
		if vg, ok := visMap[role.ID]; ok {
			visGroups = vg
		}
	}

	jsonOK(w, map[string]interface{}{
		"role":           role,
		"skills":         skills,
		"plugins":        plugins,
		"visible_groups": visGroups,
	})
}

// ── 员工端角色接口 ──────────────────────────────────────────────────

// HandleOpenClawRoles 获取员工端可选角色列表（visible=true + 按用户分组过滤可见范围）
// GET /openclaw/roles
func HandleOpenClawRoles(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	// 查询所有 visible=true 的角色
	var allVisibleRoles []model.OpenClawRole
	model.DB(r.Context()).Where("visible = ?", true).Order("sort_order asc, id asc").Find(&allVisibleRoles)

	// 按可见范围过滤：visibility_type=all 的角色所有人可见，visibility_type=group 的需要分组匹配
	userGroupIDs, err := model.GetUserGroupIDs(r.Context(), user.ID)
	if err != nil {
		slog.Error("[OpenClawRoles] 查询用户分组失败", "user_id", user.ID, "error", err)
		// 失败时退化为只显示 all 类型的角色
		userGroupIDs = nil
	}

	// 批量查询 group 类型角色的分组关联
	var groupRoleIDs []uint
	for _, role := range allVisibleRoles {
		if role.VisibilityType == usergroup.VisibilityGroup {
			groupRoleIDs = append(groupRoleIDs, role.ID)
		}
	}
	roleGroupMap := make(map[uint][]uint)
	if len(groupRoleIDs) > 0 {
		roleGroupMap, _ = model.GetRoleVisibilityGroupIDs(r.Context(), groupRoleIDs)
	}

	// 过滤可见角色
	userGroupSet := make(map[uint]bool)
	for _, gid := range userGroupIDs {
		userGroupSet[gid] = true
	}
	var roles []model.OpenClawRole
	for _, role := range allVisibleRoles {
		if role.VisibilityType != usergroup.VisibilityGroup {
			roles = append(roles, role) // all 类型直接通过
			continue
		}
		// group 类型：检查用户分组与角色分组是否有交集
		for _, gid := range roleGroupMap[role.ID] {
			if userGroupSet[gid] {
				roles = append(roles, role)
				break
			}
		}
	}

	// 批量查询技能
	roleIDs := make([]uint, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID
	}

	var allSkills []model.OpenClawRoleSkill
	if len(roleIDs) > 0 {
		model.DB(r.Context()).Where("open_claw_role_id IN ?", roleIDs).Find(&allSkills)
	}

	var allPlugins []model.OpenClawRolePlugin
	if len(roleIDs) > 0 {
		model.DB(r.Context()).Where("open_claw_role_id IN ?", roleIDs).Find(&allPlugins)
	}

	skillMap := make(map[uint][]model.OpenClawRoleSkill)
	for _, skill := range allSkills {
		skillMap[skill.OpenClawRoleID] = append(skillMap[skill.OpenClawRoleID], skill)
	}

	pluginMap := make(map[uint][]model.OpenClawRolePlugin)
	for _, plugin := range allPlugins {
		pluginMap[plugin.OpenClawRoleID] = append(pluginMap[plugin.OpenClawRoleID], plugin)
	}

	type roleResp struct {
		ID          uint                       `json:"id"`
		Name        string                     `json:"name"`
		Description string                     `json:"description"`
		Soul        string                     `json:"soul"`
		Skills      []model.OpenClawRoleSkill  `json:"skills"`
		Plugins     []model.OpenClawRolePlugin `json:"plugins"`
	}

	result := make([]roleResp, len(roles))
	for i, role := range roles {
		skills := skillMap[role.ID]
		if skills == nil {
			skills = []model.OpenClawRoleSkill{}
		}
		plugins := pluginMap[role.ID]
		if plugins == nil {
			plugins = []model.OpenClawRolePlugin{}
		}
		result[i] = roleResp{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			Soul:        role.Soul,
			Skills:      skills,
			Plugins:     plugins,
		}
	}

	jsonOK(w, map[string]interface{}{"roles": result})
}

// buildRoleVisibilityData 批量构建角色列表的可见性分组数据。
// 返回 map[roleID][]visibilityGroupInfo（含 group_id + group_name）。
// 固定 2 次额外 DB 查询（查关联 + 查分组名称），无 N+1 问题。
func buildRoleVisibilityData(ctx context.Context, roles []model.OpenClawRole) map[uint][]visibilityGroupInfo {
	result := make(map[uint][]visibilityGroupInfo)

	var groupRoleIDs []uint
	for _, r := range roles {
		if r.VisibilityType == usergroup.VisibilityGroup {
			groupRoleIDs = append(groupRoleIDs, r.ID)
		}
	}
	if len(groupRoleIDs) == 0 {
		return result
	}

	roleGroupMap, err := model.GetRoleVisibilityGroupIDs(ctx, groupRoleIDs)
	if err != nil {
		slog.Error("[RoleVisibility] 批量查询角色分组关联失败", "error", err)
		return result
	}

	groupIDSet := make(map[uint]bool)
	for _, gids := range roleGroupMap {
		for _, gid := range gids {
			groupIDSet[gid] = true
		}
	}
	if len(groupIDSet) == 0 {
		return result
	}
	uniqueGroupIDs := make([]uint, 0, len(groupIDSet))
	for gid := range groupIDSet {
		uniqueGroupIDs = append(uniqueGroupIDs, gid)
	}

	groups, rerr := model.GetGroupsByIDs(ctx, uniqueGroupIDs)
	if rerr != nil {
		slog.Error("[RoleVisibility] 批量查询分组名称失败", "error", rerr)
		return result
	}
	groupNameMap := make(map[uint]string)
	for _, g := range groups {
		groupNameMap[g.ID] = g.Name
	}

	for roleID, gids := range roleGroupMap {
		infos := make([]visibilityGroupInfo, 0, len(gids))
		for _, gid := range gids {
			name := groupNameMap[gid]
			if name == "" {
				continue
			}
			infos = append(infos, visibilityGroupInfo{
				GroupID:   gid,
				GroupName: name,
			})
		}
		result[roleID] = infos
	}
	return result
}

// syncRoleSkillsCosZipKey 为指定角色中 cos_zip_key 为空的技能同步 zip 路径。
//
// 不论 public 还是 enterprise 技能，最终写回 open_claw_role_skills.cos_zip_key 的都是
// SMH common space 下的路径（role-skills/<slug>/<slug>-<version>.zip），以便后续
// installSkillsAsync 统一通过 BuildCommonSMHDownloadURL 生成下载 URL。
//
//   - source=public：优先复用 open_claw_role_skills 历史记录；否则从 SkillHub 公共接口下载 zip 并上传到 common space。
//   - source=enterprise：优先复用 open_claw_role_skills 历史记录；否则从 skills 表取 cos_zip_key（在 SkillhubSpace），
//     下载后再上传到 common space；若 skills 表中存的 key 已经是 common space 的 role-skills/ 路径，则直接复用。
//
// 该函数设计为异步调用（go syncRoleSkillsCosZipKey(...)），失败仅记录日志不影响主流程。
//
// 以下三个包级变量用于测试注入，生产代码无需关心：
//   - roleSkillCommonClientFactory：获取 common space 存储客户端
//   - roleSkillEnterpriseDownloadURL：生成企业技能（SkillhubSpace）的下载 URL
//   - roleSkillPublicDownloadURL：生成公共技能（SkillHub 公共接口）的下载 URL
var (
	roleSkillCommonClientFactory = func(ctx context.Context) (StorageClient, error) {
		return GetCommonStorageClient(ctx)
	}
	roleSkillEnterpriseDownloadURL = func(ctx context.Context, srcKey string) (string, error) {
		return buildSMHDownloadURL(ctx, srcKey, false)
	}
	roleSkillPublicDownloadURL = buildSkillHubPublicDownloadURL
)

func syncRoleSkillsCosZipKey(ctx context.Context, roleID uint) {
	logger := slog.With("task", "syncRoleSkillsCosZipKey", "role_id", roleID)

	// 查询该角色下所有 cos_zip_key 为空的技能（public + enterprise 都要同步）
	var skills []model.OpenClawRoleSkill
	model.DB(ctx).Where("open_claw_role_id = ? AND cos_zip_key = ''", roleID).Find(&skills)
	if len(skills) == 0 {
		return
	}

	// 第一轮：尝试复用已有的 cos_zip_key（无需 SMH 客户端）
	// 同时为 enterprise 技能预先查好 skills 表中的 source key，避免第二轮重复查询。
	var needDownload []model.OpenClawRoleSkill
	entSkillSourceKey := make(map[uint]string) // role_skill.ID -> skills 表中该 slug+version 的 cos_zip_key
	for _, skill := range skills {
		// 1) 任何 source 都先尝试从 open_claw_role_skills 历史记录中复用（历史记录里已经是 common space 的 key）
		var existing model.OpenClawRoleSkill
		if model.DB(ctx).Where("slug = ? AND version = ? AND cos_zip_key != ''", skill.Slug, skill.Version).
			First(&existing).Error == nil {
			model.DB(ctx).Model(&skill).Update("cos_zip_key", existing.CosZipKey)
			logger.Info("复用已有 cos_zip_key", "slug", skill.Slug, "source", skill.Source, "cos_zip_key", existing.CosZipKey)
			continue
		}

		// 2) enterprise 技能：查 skills 表拿到 SkillhubSpace 中的 key
		if skill.Source == "enterprise" {
			var entSkill model.Skill
			if err := model.DB(ctx).Where("slug = ? AND version = ? AND cos_zip_key != ''", skill.Slug, skill.Version).
				First(&entSkill).Error; err != nil {
				logger.Warn("企业技能在 skills 表中未找到可用的 cos_zip_key，跳过",
					"slug", skill.Slug, "version", skill.Version, "error", err)
				continue
			}
			entSkillSourceKey[skill.ID] = entSkill.COSZipKey
			needDownload = append(needDownload, skill)
			continue
		}

		// 3) public 技能：进入第二轮 SkillHub 下载
		needDownload = append(needDownload, skill)
	}

	if len(needDownload) == 0 {
		return
	}

	// 第二轮：需要下载/中转上传的技能，获取 common space 存储客户端
	commonClient, err := roleSkillCommonClientFactory(ctx)
	if err != nil {
		logger.Error("获取 common space 存储客户端失败", "error", err)
		return
	}

	for _, skill := range needDownload {
		var zipData []byte

		if skill.Source == "enterprise" {
			srcKey := entSkillSourceKey[skill.ID]
			if srcKey == "" {
				// 理论上不会发生，保险处理
				logger.Warn("企业技能缺少 skills 表中的 cos_zip_key，跳过", "slug", skill.Slug)
				continue
			}

			// 从 SMH SkillhubSpace 下载 zip
			downloadURL, urlErr := roleSkillEnterpriseDownloadURL(ctx, srcKey)
			if urlErr != nil {
				logger.Error("生成企业技能 SkillhubSpace 下载 URL 失败",
					"slug", skill.Slug, "src_key", srcKey, "error", urlErr)
				continue
			}
			resp, getErr := SkillHTTPClient.Get(downloadURL)
			if getErr != nil {
				logger.Error("下载企业技能 zip 失败", "slug", skill.Slug, "src_key", srcKey, "error", getErr)
				continue
			}
			data, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil || resp.StatusCode != http.StatusOK {
				logger.Error("读取企业技能 zip 失败",
					"slug", skill.Slug, "src_key", srcKey, "status", resp.StatusCode, "error", readErr)
				continue
			}
			zipData = data
		} else {
			// public 技能：从 SkillHub 公共下载接口下载 zip
			downloadURL := roleSkillPublicDownloadURL(skill.Slug, skill.Version)
			resp, getErr := SkillHTTPClient.Get(downloadURL)
			if getErr != nil {
				logger.Error("下载公共技能 zip 失败", "slug", skill.Slug, "error", getErr)
				continue
			}
			data, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil || resp.StatusCode != http.StatusOK {
				logger.Error("读取公共技能 zip 失败", "slug", skill.Slug, "status", resp.StatusCode, "error", readErr)
				continue
			}
			zipData = data
		}

		// 统一上传到 common space（路径：role-skills/<slug>/<slug>-<version>.zip）
		cosZipKey := fmt.Sprintf("role-skills/%s/%s-%s.zip", skill.Slug, skill.Slug, skill.Version)
		if err := commonClient.Upload(cosZipKey, zipData, "application/zip"); err != nil {
			logger.Error("上传技能 zip 到 common space 失败",
				"slug", skill.Slug, "source", skill.Source, "cos_zip_key", cosZipKey, "error", err)
			continue
		}

		// 更新 cos_zip_key（同 slug+version 的记录一起更新）
		model.DB(ctx).Model(&model.OpenClawRoleSkill{}).
			Where("slug = ? AND version = ? AND cos_zip_key = ''", skill.Slug, skill.Version).
			Update("cos_zip_key", cosZipKey)
		logger.Info("技能 zip 同步成功",
			"slug", skill.Slug, "source", skill.Source, "cos_zip_key", cosZipKey)
	}
}

// HandleRemoveInstanceRole 移除实例角色（回退为通用助手）
// POST /openclaw/remove-role?id=xxx
func HandleRemoveInstanceRole(w http.ResponseWriter, r *http.Request) {
	handleRemoveInstanceRole(w, r, defaultStatusResolver)
}

func handleRemoveInstanceRole(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgMissingParamID))
		return
	}

	var instance model.Instance
	if model.DB(r.Context()).Where("id = ? AND user_id = ?", id, user.ID).First(&instance).Error != nil {
		writeError(w, r, http.StatusNotFound, common.I18nError(i18n.MsgInstanceNotFound))
		return
	}

	// 【防护】检查实例类型是否支持角色配置
	if !model.AgentTypeSupportsRole(r.Context(), instance.AgentType) {
		writeError(w, r, http.StatusForbidden, common.I18nError(i18n.MsgInstanceNotSupportRole))
		return
	}

	if instance.RoleID == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgInstanceNotLinkedToRole))
		return
	}

	// 本地实例：不支持从实例上移除角色。
	if rejectLocalOrWrite(w, r, &instance) {
		return
	}

	// 状态准入：仅 running 状态允许移除角色
	if _, err := requireInstanceRunning(r.Context(), &instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	// tx 内：finalize 老 record（若有）+ 更新 instance 字段
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		finalizeActiveRecordAsCancelled(tx, instance.ID)
		return tx.Model(&instance).Updates(map[string]interface{}{
			"role_id":                  0,
			"distributed_role_version": "",
			"role_sync_status":         "",
		}).Error
	}); err != nil {
		writeError(w, r, http.StatusInternalServerError, common.I18nRichError(err, i18n.MsgRemoveRoleFailed))
		return
	}

	if err := RemoveInstanceSoul(r.Context(), instance.ID, 0); err != nil {
		writeError(w, r, http.StatusInternalServerError, common.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

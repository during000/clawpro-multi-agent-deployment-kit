package controller

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

// AgentTypeResponse 智能体类型响应（包含启用镜像信息）
type AgentTypeResponse struct {
	Code              string            `json:"code"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	IsBuiltin         bool              `json:"is_builtin"`
	CompatibleWith    string            `json:"compatible_with,omitempty"`
	SupportsRole      bool              `json:"supports_role"`
	SupportsModel     bool              `json:"supports_model"`
	SupportsChannel   bool              `json:"supports_channel"`
	SupportsSkill     bool              `json:"supports_skill"`
	SupportsPlugin    bool              `json:"supports_plugin"`
	SupportsChatbot   bool              `json:"supports_chatbot"`
	SupportsSMH       bool              `json:"supports_smh"`       // final 新增
	SupportsMemory    bool              `json:"supports_memory"`    // final 新增
	SupportsReinstall bool              `json:"supports_reinstall"` // final 新增：供前端控制"重装"按钮是否可见
	SupportsUpgrade   bool              `json:"supports_upgrade"`   // 供前端控制"一键升级"按钮是否可见
	SortOrder         int               `json:"sort_order"`
	Enabled           bool              `json:"enabled"`
	HasEnabledImage   bool              `json:"has_enabled_image"`
	EnabledImage      *EnabledImageInfo `json:"enabled_image,omitempty"`
	IsDefault         bool              `json:"is_default"`
	UserSelectable    bool              `json:"user_selectable"`
	ImageCount        int64             `json:"image_count"`
	InstanceCount     int64             `json:"instance_count"`
}

// EnabledImageInfo 启用镜像简要信息
type EnabledImageInfo struct {
	ID           uint   `json:"id"`
	ImageName    string `json:"image_name"`
	AgentVersion string `json:"agent_version"`
}

// HandleAdminAgentTypes 获取智能体类型列表（只读）
func HandleAdminAgentTypes(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ctx := r.Context()

	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	types := model.GetAllAgentTypes(ctx)

	imagesMap, err := model.GetEnabledImagesMap(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "[AgentTypes] 查询启用镜像失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 获取各类型镜像统计
	imageStats, err := model.GetImageStatsByType(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "[AgentTypes] 查询镜像统计失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 获取各类型当前实例数量
	instanceCounts, err := model.GetInstanceCountsByType(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "[AgentTypes] 查询实例数量失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryInstanceCountFailed))
		return
	}

	defaultType := model.GetDefaultAgentType(ctx)

	var responses []AgentTypeResponse
	for _, t := range types {
		hasImage := false
		enabled := model.IsAgentTypeEnabled(ctx, t.Code)
		resp := AgentTypeResponse{
			Code:              t.Code,
			Name:              t.Name,
			Description:       t.Description,
			IsBuiltin:         t.IsBuiltin,
			CompatibleWith:    t.CompatibleWith,
			SupportsRole:      t.SupportsRole,
			SupportsModel:     t.SupportsModel,
			SupportsChannel:   t.SupportsChannel,
			SupportsSkill:     t.SupportsSkill,
			SupportsPlugin:    t.SupportsPlugin,
			SupportsChatbot:   t.SupportsChatbot,
			SupportsSMH:       t.SupportsSMH,
			SupportsMemory:    t.SupportsMemory,
			SupportsReinstall: t.SupportsReinstall,
			SupportsUpgrade:   t.SupportsUpgrade,
			SortOrder:         t.SortOrder,
			Enabled:           enabled,
			IsDefault:         t.Code == defaultType,
			ImageCount:        imageStats[t.Code],
			InstanceCount:     instanceCounts[t.Code],
		}
		if img, ok := imagesMap[t.Code]; ok && img != nil {
			hasImage = true
			resp.HasEnabledImage = true
			resp.EnabledImage = &EnabledImageInfo{
				ID:           img.ID,
				ImageName:    img.ImageName,
				AgentVersion: img.AgentVersion,
			}
		}
		resp.UserSelectable = enabled && hasImage
		responses = append(responses, resp)
	}

	jsonOK(w, map[string]interface{}{
		"agent_types":        responses,
		"default_agent_type": defaultType,
	})
}

// HandleUpdateAgentTypeEnabled 启用/禁用/切换用户端是否可选择某个智能体类型。
func HandleUpdateAgentTypeEnabled(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	agentType := strings.TrimSpace(r.FormValue("agent_type"))
	if agentType == "" {
		agentType = strings.TrimSpace(r.FormValue("code"))
	}
	var enabledParam *bool
	if raw := strings.TrimSpace(r.FormValue("enabled")); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgParamMustBeBool, "enabled"))
			return
		}
		enabledParam = &enabled
	}
	var toggleParam *bool
	if raw := strings.TrimSpace(r.FormValue("toggle")); raw != "" {
		toggle, err := strconv.ParseBool(raw)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgParamMustBeBool, "toggle"))
			return
		}
		toggleParam = &toggle
	}
	if agentType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentTypeCannotBeEmpty))
		return
	}

	if enabledParam != nil && toggleParam != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnabledAndToggleMutex))
		return
	}
	if enabledParam == nil && toggleParam == nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnabledOrToggleRequired))
		return
	}

	previousEnabled := model.IsAgentTypeEnabled(r.Context(), agentType)
	enabled := previousEnabled
	operation := ""
	if enabledParam != nil {
		enabled = *enabledParam
		if enabled {
			operation = "enable"
		} else {
			operation = "disable"
		}
	} else {
		if !*toggleParam {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgToggleMustBeTrue))
			return
		}
		enabled = !previousEnabled
		operation = "toggle"
	}

	if err := model.SetAgentTypeEnabled(r.Context(), agentType, enabled); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":                   true,
		"agent_type":           agentType,
		"operation":            operation,
		"previous_enabled":     previousEnabled,
		"enabled":              model.IsAgentTypeEnabled(r.Context(), agentType),
		"disabled_agent_types": model.GetDisabledAgentTypes(r.Context()),
		"default_agent_type":   model.GetDefaultAgentType(r.Context()),
	})
}

// HandleCreateCustomAgentType 创建自定义智能体类型。
func HandleCreateCustomAgentType(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	name := r.FormValue("name")
	compatibleWith := r.FormValue("compatible_with")
	if err := model.ValidateCustomAgentTypeName(name); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := model.ValidateCompatibleAgentType(compatibleWith); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if _, err := model.CreateCustomAgentType(r.Context(), name, compatibleWith); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleDeleteCustomAgentType 删除自定义智能体类型。
func HandleDeleteCustomAgentType(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNameCannotBeEmpty))
		return
	}
	if model.IsBuiltinAgentType(name) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBuiltinAgentTypeCannotDel))
		return
	}
	t, err := model.GetCustomAgentTypeByName(r.Context(), name)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if t == nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCustomAgentTypeNotFound, name))
		return
	}
	if err := model.DeleteCustomAgentType(r.Context(), name); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}

func normalizeAgentTypeList(types []string) []string {
	result := make([]string, 0, len(types))
	seen := make(map[string]struct{}, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}
	return result
}

func validateDisabledAgentTypes(ctx context.Context, types []string) error {
	defaultType := model.GetDefaultAgentType(ctx)
	for _, t := range normalizeAgentTypeList(types) {
		if !model.IsValidAgentType(ctx, t) {
			return hcommon.I18nError(i18n.MsgInvalidAgentType, t)
		}
		if t == defaultType {
			return hcommon.I18nError(i18n.MsgAgentTypeIsDefaultCannotDisable)
		}
	}
	return nil
}

// HandleUserAgentTypes 获取用户可选的智能体类型（仅有启用镜像的类型）。
func HandleUserAgentTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	allTypes := model.GetAllAgentTypes(ctx)
	allTypeCodes := make([]string, 0, len(allTypes))
	for _, t := range allTypes {
		if t != nil {
			allTypeCodes = append(allTypeCodes, t.Code)
		}
	}

	groupIDs, err := model.GetUserGroupIDs(ctx, user.ID)
	if err != nil {
		slog.ErrorContext(ctx, "[UserAgentTypes] 查询用户分组失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	ancestorSet := make(map[uint]struct{})
	for _, gid := range groupIDs {
		ancestors, err := usergroup.GetAncestorIDs(ctx, gid)
		if err != nil {
			slog.ErrorContext(ctx, "[UserAgentTypes] 查询用户分组祖先失败", "group_id", gid, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryUserGroupFailed))
			return
		}
		for _, id := range ancestors {
			ancestorSet[id] = struct{}{}
		}
	}
	allGroupIDs := make([]uint, 0, len(ancestorSet))
	for id := range ancestorSet {
		allGroupIDs = append(allGroupIDs, id)
	}
	visibleTypes, err := usergroup.ResolveImageTypes(ctx, allGroupIDs, allTypeCodes)
	if err != nil {
		slog.ErrorContext(ctx, "[UserAgentTypes] 解析镜像类型可见性失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgResolveImageVisibilityFail))
		return
	}
	visibleSet := make(map[string]struct{}, len(visibleTypes))
	for _, t := range visibleTypes {
		visibleSet[t] = struct{}{}
	}

	enabledImagesMap, err := model.GetEnabledImagesMap(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "[UserAgentTypes] 查询启用镜像失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	defaultType := model.GetDefaultAgentType(ctx)

	type UserAgentType struct {
		Code           string `json:"code"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		IsBuiltin      bool   `json:"is_builtin"`
		CompatibleWith string `json:"compatible_with,omitempty"`
		IsDefault      bool   `json:"is_default"`
	}

	var selectableTypes []UserAgentType
	for _, t := range allTypes {
		if _, hasImage := enabledImagesMap[t.Code]; hasImage {
			if _, visible := visibleSet[t.Code]; !visible {
				continue
			}
			selectableTypes = append(selectableTypes, UserAgentType{
				Code:           t.Code,
				Name:           t.Name,
				Description:    t.Description,
				IsBuiltin:      t.IsBuiltin,
				CompatibleWith: t.CompatibleWith,
				IsDefault:      t.Code == defaultType,
			})
		}
	}

	jsonOK(w, map[string]interface{}{
		"agent_types":        selectableTypes,
		"default_agent_type": defaultType,
	})
}

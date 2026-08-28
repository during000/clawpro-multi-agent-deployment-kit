package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

const adminCreateInstanceBodyLimit = 1 << 20

type adminCreateModelsRequest struct {
	Primary   uint   `json:"primary"`
	Fallbacks []uint `json:"fallbacks"`
}

type adminCreateChannelRequest struct {
	Channel string            `json:"channel"`
	Config  map[string]string `json:"config"`
}

type adminCreateSkillRequest struct {
	Source  string `json:"source"`
	Slug    string `json:"slug"`
	Version string `json:"version"`
}

type adminCreateInstanceRequest struct {
	UserID    uint                        `json:"user_id"`
	Name      string                      `json:"name"`
	GroupID   uint                        `json:"group_id"`
	AgentType string                      `json:"agent_type"`
	RoleID    uint                        `json:"role_id"`
	DiskType  string                      `json:"disk_type"`
	Tags      []createInstanceTag         `json:"tags"`
	Models    *adminCreateModelsRequest   `json:"models"`
	Channels  []adminCreateChannelRequest `json:"channels"`
	Skills    []adminCreateSkillRequest   `json:"skills"`
}

// HandleAdminCreateInstance creates a CVM Agent for a target user. The create
// lifecycle is shared with HandleCreateInstance; only the resolved presets are
// admin-specific.
func HandleAdminCreateInstance(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, adminCreateInstanceBodyLimit)
	var input adminCreateInstanceRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("request body must contain a single JSON object")
		}
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if input.UserID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "user_id"))
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 128 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNameRequired))
		return
	}
	input.AgentType = strings.TrimSpace(input.AgentType)
	if input.AgentType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "agent_type"))
		return
	}
	input.AgentType = model.NormalizeAgentType(input.AgentType)
	if err := checkAgentTypeValid(r.Context(), input.AgentType); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	var owner model.User
	if err := model.DB(r.Context()).First(&owner, input.UserID).Error; err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserNotFound, fmt.Sprint(input.UserID)))
		return
	}

	groupID, err := resolveAdminCreateGroup(r.Context(), owner.ID, input.GroupID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	presets, err := resolveAdminCreatePresets(r.Context(), input, groupID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	form := make(url.Values)
	form.Set("name", input.Name)
	form.Set("group_id", fmt.Sprint(groupID))
	form.Set("agent_type", input.AgentType)
	if input.RoleID > 0 {
		form.Set("role_id", fmt.Sprint(input.RoleID))
	}
	if diskType := strings.TrimSpace(input.DiskType); diskType != "" {
		form.Set("disk_type", diskType)
	}
	r.Form = form
	r.PostForm = form
	result, ok := createInstance(w, r, &owner, createInstanceOptions{
		Presets:    presets,
		CustomTags: &input.Tags,
	})
	if !ok {
		return
	}
	jsonOK(w, map[string]interface{}{
		"ok":          true,
		"instance_id": result.InstanceID,
	})
}

func resolveAdminCreateGroup(ctx context.Context, userID, requestedGroupID uint) (uint, error) {
	groups, err := model.GetUserGroupsByUserID(ctx, userID)
	if err != nil {
		return 0, hcommon.I18nRichError(err, i18n.MsgQueryGroupFailed)
	}
	valid := make([]model.UserGroup, 0, len(groups))
	for _, group := range groups {
		if !group.ToBeDeleted {
			valid = append(valid, group)
		}
	}
	if requestedGroupID > 0 {
		for _, group := range valid {
			if group.ID == requestedGroupID {
				return requestedGroupID, nil
			}
		}
		return 0, hcommon.I18nError(i18n.MsgAdminCreateGroupInvalid)
	}
	switch len(valid) {
	case 0:
		return 0, nil
	case 1:
		return valid[0].ID, nil
	default:
		return 0, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_id")
	}
}

func resolveAdminCreatePresets(ctx context.Context, input adminCreateInstanceRequest, groupID uint) (*createInstancePresets, error) {
	presets := &createInstancePresets{}
	var err error
	if input.Models != nil {
		if !model.AgentTypeSupportsModel(ctx, input.AgentType) {
			return nil, hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportModelConfigWithDetail, model.GetAgentTypeDisplayName(ctx, input.AgentType))
		}
		if len(input.Models.Fallbacks) > 0 {
			if fallbackErr := modelFallbackSupportError(ctx, input.AgentType, ""); fallbackErr != nil {
				return nil, fallbackErr
			}
		}
		presets.Models, err = resolveAdminCreateModels(ctx, *input.Models, groupID)
		if err != nil {
			return nil, err
		}
	}
	if len(input.Channels) > 0 {
		if !model.AgentTypeSupportsChannel(ctx, input.AgentType) {
			return nil, hcommon.I18nError(i18n.MsgChannelNotSupportedWithDetail, model.GetAgentTypeDisplayName(ctx, input.AgentType))
		}
		presets.Channels, err = resolveAdminCreateChannels(ctx, input.AgentType, groupID, input.Channels)
		if err != nil {
			return nil, err
		}
	}
	if len(input.Skills) > 0 {
		if !model.AgentTypeSupportsSkill(ctx, input.AgentType) {
			return nil, hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportSkillWithDetail, model.GetAgentTypeDisplayName(ctx, input.AgentType))
		}
		presets.Skills, err = resolveAdminCreateSkills(ctx, groupID, input.Skills)
		if err != nil {
			return nil, err
		}
	}
	if input.Models == nil && len(input.Channels) == 0 && len(input.Skills) == 0 {
		return nil, nil
	}
	return presets, nil
}

func resolveAdminCreateModels(ctx context.Context, input adminCreateModelsRequest, groupID uint) ([]model.AIModel, error) {
	if input.Primary == 0 {
		return nil, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "models.primary")
	}
	ids := make([]uint, 0, len(input.Fallbacks)+1)
	ids = append(ids, input.Primary)
	seen := map[uint]struct{}{input.Primary: {}}
	for _, id := range input.Fallbacks {
		if id == 0 {
			return nil, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "models.fallbacks")
		}
		if _, exists := seen[id]; exists {
			return nil, hcommon.I18nError(i18n.MsgAdminCreateModelDuplicate)
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	var candidates []model.AIModel
	if err := model.DB(ctx).Where("id IN ? AND enabled = ? AND visible = ?", ids, true, true).Find(&candidates).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgModelQueryListFailed)
	}
	visible := usergroup.FilterModelsByVisibility(ctx, candidates, groupID)
	byID := make(map[uint]model.AIModel, len(visible))
	for _, candidate := range visible {
		byID[candidate.ID] = candidate
	}
	resolved := make([]model.AIModel, 0, len(ids))
	for _, id := range ids {
		candidate, ok := byID[id]
		if !ok {
			return nil, hcommon.I18nError(i18n.MsgAdminCreateModelUnavailable, id)
		}
		resolved = append(resolved, candidate)
	}
	return resolved, nil
}

func resolveAdminCreateChannels(ctx context.Context, agentType string, groupID uint, input []adminCreateChannelRequest) ([]manualChannelPreset, error) {
	resolved := make([]manualChannelPreset, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	instance := &model.Instance{AgentType: agentType, GroupID: groupID}
	for _, item := range input {
		channelID := strings.TrimSpace(item.Channel)
		if channelID == "" {
			return nil, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "channels[].channel")
		}
		if _, exists := seen[channelID]; exists {
			return nil, hcommon.I18nError(i18n.MsgAdminCreateChannelDuplicate)
		}
		seen[channelID] = struct{}{}
		config := make(map[string]string, len(item.Config))
		for key, value := range item.Config {
			key = strings.TrimSpace(key)
			if key == "" || strings.TrimSpace(value) == "" {
				return nil, hcommon.I18nError(i18n.MsgAdminCreateChannelConfigInvalid)
			}
			config[key] = value
		}
		if _, err := validateManualChannelConfig(ctx, instance, manualChannelPreset{Channel: channelID, Config: config}, true); err != nil {
			return nil, err
		}
		resolved = append(resolved, manualChannelPreset{Channel: channelID, Config: config})
	}
	return resolved, nil
}

func resolveAdminCreateSkills(ctx context.Context, groupID uint, input []adminCreateSkillRequest) ([]createSkillPreset, error) {
	resolved := make([]createSkillPreset, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	var ancestorSet map[uint]struct{}
	if groupID > 0 {
		ancestors, err := usergroup.GetAncestorIDs(ctx, groupID)
		if err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgQueryGroupFailed)
		}
		ancestorSet = make(map[uint]struct{}, len(ancestors))
		for _, id := range ancestors {
			ancestorSet[id] = struct{}{}
		}
	}

	for _, item := range input {
		source := strings.ToLower(strings.TrimSpace(item.Source))
		if source == "" {
			source = model.SkillSourcePublic
		}
		if source != model.SkillSourcePublic && source != model.SkillSourceEnterprise {
			return nil, hcommon.I18nError(i18n.MsgAdminCreateSkillSourceUnsupported)
		}
		slug := strings.TrimSpace(item.Slug)
		version := strings.TrimSpace(item.Version)
		if slug == "" {
			return nil, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "skills[].slug")
		}
		key := source + "\x00" + slug + "\x00" + version
		if _, exists := seen[key]; exists {
			continue
		}

		preset := createSkillPreset{Source: source, Slug: slug, Version: version}
		if source == model.SkillSourceEnterprise {
			var skill model.Skill
			query := model.DB(ctx).Where("slug = ?", slug)
			if version != "" {
				query = query.Where("version = ?", version)
			} else {
				query = query.Order("version_major DESC, version_minor DESC, version_patch DESC")
			}
			if err := query.First(&skill).Error; err != nil {
				return nil, hcommon.I18nError(i18n.MsgAdminCreateSkillUnavailable, slug)
			}
			if skill.VisibilityType == model.VisibilityGroup {
				if len(ancestorSet) == 0 {
					return nil, hcommon.I18nError(i18n.MsgAdminCreateSkillNotVisible, slug)
				}
				bindings, err := model.GetSkillVisibilityGroupIDs(ctx, []uint{skill.ID})
				if err != nil {
					return nil, err
				}
				visible := false
				for _, id := range bindings[skill.ID] {
					if _, ok := ancestorSet[id]; ok {
						visible = true
						break
					}
				}
				if !visible {
					return nil, hcommon.I18nError(i18n.MsgAdminCreateSkillNotVisible, slug)
				}
			}
			preset.Version = skill.Version
			preset.Enterprise = skill
		}
		seen[key] = struct{}{}
		resolved = append(resolved, preset)
	}
	return resolved, nil
}

package controller

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

func queryAllImages(ctx context.Context) []model.AIImage {
	var images []model.AIImage
	model.DB(ctx).Order("id desc").Find(&images)
	return images
}

func HandleAdminImages(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	images := queryAllImages(r.Context())
	type imageJSON struct {
		model.AIImage
		Public            bool   `json:"public,omitempty"`
		IsLegacy          bool   `json:"is_legacy"`
		CanEnable         bool   `json:"can_enable"`
		EnableBlockReason string `json:"enable_block_reason,omitempty"`
		AgentName         string `json:"agent_name,omitempty"`
	}
	result := make([]imageJSON, len(images))
	for i, img := range images {
		blockErr := img.CanEnableImage(r.Context())
		item := imageJSON{
			AIImage:           img,
			Public:            hcommon.IsCandidateImage(img.ImageId),
			IsLegacy:          img.IsLegacyImage(r.Context()),
			CanEnable:         blockErr == nil,
			EnableBlockReason: hcommon.ErrorMessageWithCtx(r.Context(), blockErr),
		}
		// 仅公共镜像返回智能体名称，非公共镜像不返回
		// TODO: 后续通过查询镜像详情接口动态获取智能体名称，当前先写死
		if item.Public {
			item.AgentName = "OpenClaw"
		}
		result[i] = item
	}

	// 构建 enabled_images_by_type
	enabledMap, _ := model.GetEnabledImagesMap(r.Context())
	enabledByType := make(map[string]interface{})
	for agentType, img := range enabledMap {
		enabledByType[agentType] = map[string]interface{}{
			"id":            img.ID,
			"image_name":    img.ImageName,
			"agent_version": img.AgentVersion,
		}
	}

	// 构建 image_type_visibility：每种 agent_type 的应用范围信息
	allAgentTypes := model.GetAllAgentTypes(r.Context())
	imageTypeVisibility := make(map[string]interface{})
	restricted, _ := model.GetRestrictedImageTypes(r.Context())
	restrictedSet := make(map[string]struct{}, len(restricted))
	for _, item := range restricted {
		restrictedSet[item] = struct{}{}
	}
	restrictedKeys := make([]string, 0, len(restricted))
	for _, at := range allAgentTypes {
		if at == nil {
			continue
		}
		if _, isRestricted := restrictedSet[at.Code]; isRestricted {
			restrictedKeys = append(restrictedKeys, at.Code)
		}
	}
	imageBindingMap := usergroup.GetVisibilityGroupRefsStr(r.Context(), model.ConfigTypeImageType, restrictedKeys)

	for _, at := range allAgentTypes {
		if at == nil {
			continue
		}
		if _, isRestricted := restrictedSet[at.Code]; !isRestricted {
			imageTypeVisibility[at.Code] = map[string]interface{}{
				"visibility_type": usergroup.VisibilityAll,
			}
		} else {
			imageTypeVisibility[at.Code] = map[string]interface{}{
				"visibility_type":   usergroup.VisibilityGroup,
				"visibility_groups": imageBindingMap[at.Code],
			}
		}
	}

	jsonOK(w, map[string]interface{}{
		"images":                 result,
		"enabled_images_by_type": enabledByType,
		"default_agent_type":     model.GetDefaultAgentType(r.Context()),
		"image_type_visibility":  imageTypeVisibility,
	})
}

func HandleImportImage(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	imageId := r.FormValue("image_id")
	imageName := r.FormValue("image_name")
	agentType := strings.TrimSpace(r.FormValue("agent_type"))
	agentVersion := strings.TrimSpace(r.FormValue("agent_version"))
	if imageId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImageIDCannotBeEmpty))
		return
	}

	// 【V2】agent_type 必填
	if agentType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentTypeCannotBeEmpty))
		return
	}
	if err := checkAgentTypeValid(r.Context(), agentType); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 【V2】agent_version 必填；自定义 Agent 类型没有版本概念，允许为空
	if agentVersion == "" && !model.IsCustomAgentType(r.Context(), agentType) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentVersionCannotBeEmpty))
		return
	}
	if agentVersion != "" {
		if err := checkAgentVersionValid(agentVersion); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	// 【V2】校验版本格式（按类型）
	if err := model.ValidateAgentVersion(r.Context(), agentType, agentVersion); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	var existing model.AIImage
	if model.DB(r.Context()).Where("image_id = ?", imageId).First(&existing).Error == nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgImageIDExists, imageId))
		return
	}

	client, err := NewCVMClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	request := cvm.NewDescribeImagesRequest()
	request.ImageIds = common.StringPtrs([]string{imageId})

	response, err := client.DescribeImages(request)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryImageFailed))
		return
	}

	if len(response.Response.ImageSet) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgImageNotFoundByID, imageId))
		return
	}

	img := response.Response.ImageSet[0]
	if Int64Val(img.ImageSize) > 50 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImageSizeTooLarge))
		return
	}
	aiImage := model.AIImage{
		ImageId:      imageId,
		ImageName:    imageName,
		ImageType:    StrVal(img.ImageType),
		OsName:       StrVal(img.OsName),
		ImageSize:    Int64Val(img.ImageSize),
		ImageState:   StrVal(img.ImageState),
		AgentType:    agentType,
		AgentVersion: agentVersion,
	}
	if aiImage.ImageName == "" {
		aiImage.ImageName = StrVal(img.ImageName)
	}

	if result := model.DB(r.Context()).Create(&aiImage); result.Error != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSaveImageFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleDeleteImage(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var img model.AIImage
	if model.DB(r.Context()).Where("id = ?", id).First(&img).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgImageNotFound))
		return
	}

	if img.Enabled {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgEnabledImageCannotDelete))
		return
	}

	model.DB(r.Context()).Delete(&img)
	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleEnableImage(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var img model.AIImage
	if model.DB(r.Context()).Where("id = ?", id).First(&img).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgImageNotFound))
		return
	}

	if img.Enabled {
		// 禁用镜像
		// 【V2】检查是否为首选类型的唯一启用镜像
		defaultType := model.GetDefaultAgentType(r.Context())
		if img.AgentType == defaultType {
			count, _ := model.GetEnabledImageCountByType(r.Context(), img.AgentType)
			if count <= 1 {
				slog.Info("[Image] 禁用被阻止（首选类型保护）",
					"image_id", img.ImageId, "agent_type", img.AgentType)
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDefaultTypeCannotDisable))
				return
			}
		}
		model.DB(r.Context()).Model(&img).Update("enabled", false)
		slog.Info("[Image] 禁用镜像", "image_id", img.ImageId, "agent_type", img.AgentType)
	} else {
		// 启用镜像
		// 【V2】检查是否可启用
		if err := img.CanEnableImage(r.Context()); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}

		// 同类型互斥逻辑
		if img.AgentType != "" {
			// 有明确类型：禁用同类型的其他镜像
			result := model.DB(r.Context()).Model(&model.AIImage{}).
				Where("agent_type = ? AND enabled = ? AND id != ?",
					img.AgentType, true, img.ID).
				Update("enabled", false)
			if result.RowsAffected > 0 {
				slog.Info("[Image] 禁用同类型其他镜像",
					"agent_type", img.AgentType, "count", result.RowsAffected)
			}

			// 【修复】同时禁用空类型的存量镜像（存量镜像视为 openclaw 类型）
			if img.AgentType == model.AgentTypeOpenClaw {
				legacyResult := model.DB(r.Context()).Model(&model.AIImage{}).
					Where("(agent_type = '' OR agent_type IS NULL) AND enabled = ? AND id != ?",
						true, img.ID).
					Update("enabled", false)
				if legacyResult.RowsAffected > 0 {
					slog.Info("[Image] 禁用存量镜像（空类型视为 openclaw）",
						"count", legacyResult.RowsAffected)
				}
			}
		} else {
			// 存量镜像（空类型）：视为 openclaw 类型，禁用所有 openclaw 和空类型的已启用镜像
			result := model.DB(r.Context()).Model(&model.AIImage{}).
				Where("(agent_type = ? OR agent_type = '' OR agent_type IS NULL) AND enabled = ? AND id != ?",
					model.AgentTypeOpenClaw, true, img.ID).
				Update("enabled", false)
			if result.RowsAffected > 0 {
				slog.Info("[Image] 禁用 openclaw/存量镜像",
					"count", result.RowsAffected)
			}
		}
		model.DB(r.Context()).Model(&img).Update("enabled", true)
		slog.Info("[Image] 启用镜像",
			"image_id", img.ImageId, "agent_type", img.AgentType)
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleListCloudImages(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	client, err := NewCVMClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	request := cvm.NewDescribeImagesRequest()
	request.Filters = []*cvm.Filter{
		{
			Name:   common.StringPtr("image-type"),
			Values: common.StringPtrs([]string{"PRIVATE_IMAGE"}),
		},
	}

	response, err := client.DescribeImages(request)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryImageFailed))
		return
	}

	// Get existing image IDs from database
	existingIDs := make(map[string]bool)
	var existingImages []model.AIImage
	model.DB(r.Context()).Select("image_id").Find(&existingImages)
	for _, img := range existingImages {
		existingIDs[img.ImageId] = true
	}

	// 对候选公共镜像中尚未导入的，补充查询（私有镜像列表中不包含公共镜像）
	privateIDs := make(map[string]bool)
	for _, img := range response.Response.ImageSet {
		privateIDs[StrVal(img.ImageId)] = true
	}
	for _, candidate := range hcommon.CandidateImages {
		if existingIDs[candidate.ImageId] || privateIDs[candidate.ImageId] {
			continue
		}
		pubReq := cvm.NewDescribeImagesRequest()
		pubReq.ImageIds = common.StringPtrs([]string{candidate.ImageId})
		pubResp, err := client.DescribeImages(pubReq)
		if err == nil && len(pubResp.Response.ImageSet) > 0 {
			response.Response.ImageSet = append(response.Response.ImageSet, pubResp.Response.ImageSet[0])
		}
	}

	// Filter out already imported images
	type cloudImage struct {
		ImageId      string `json:"imageId"`
		ImageName    string `json:"imageName"`
		OsName       string `json:"osName"`
		ImageState   string `json:"imageState"`
		Public       bool   `json:"public,omitempty"`
		AgentType    string `json:"agentType,omitempty"`
		AgentVersion string `json:"agentVersion,omitempty"`
	}

	// 构建候选镜像的 agent 信息映射
	candidateInfo := make(map[string]hcommon.CandidateImage, len(hcommon.CandidateImages))
	for _, c := range hcommon.CandidateImages {
		candidateInfo[c.ImageId] = c
	}

	var result []cloudImage
	for _, img := range response.Response.ImageSet {
		id := StrVal(img.ImageId)
		if existingIDs[id] {
			continue
		}
		ci := cloudImage{
			ImageId:    id,
			ImageName:  StrVal(img.ImageName),
			OsName:     StrVal(img.OsName),
			ImageState: StrVal(img.ImageState),
			Public:     hcommon.IsCandidateImage(id),
		}
		// 候选公共镜像填充已知的 agent 信息
		if candidate, ok := candidateInfo[id]; ok {
			ci.AgentType = candidate.AgentType
			ci.AgentVersion = candidate.AgentVersion
		}
		result = append(result, ci)
	}

	jsonOK(w, result)
}

func StrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func Int64Val(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// HandleUpdateImage 更新自定义镜像的智能体类型和版本
//
// 安全规则：
//  1. 仅自定义镜像（非 CandidateImage 公共镜像）可编辑，公共镜像的 type/version 由系统维护；
//  2. 启用中的镜像禁止修改 agent_type（会打破"每类型最多一个启用镜像"的不变式，
//     且会影响已创建实例的能力矩阵匹配），但允许修改 agent_version（仅用于展示修正）；
//  3. type + version 同时提供时，需通过 ValidateAgentVersion 的组合格式校验；
//     仅提供 version 时，按已有的 agent_type 做组合校验。
func HandleUpdateImage(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ctx := r.Context()
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.FormValue("id")
	if idStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDCannotBeEmpty))
		return
	}

	var img model.AIImage
	if model.DB(r.Context()).Where("id = ?", idStr).First(&img).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgImageNotFound))
		return
	}

	// 【防护 1】公共镜像禁止编辑
	if hcommon.IsCandidateImage(img.ImageId) {
		slog.WarnContext(ctx, "[Image] 拒绝编辑公共镜像",
			"id", img.ID, "image_id", img.ImageId)
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgPublicImageCannotEdit))
		return
	}

	newAgentType := strings.TrimSpace(r.FormValue("agent_type"))
	newAgentVersion := strings.TrimSpace(r.FormValue("agent_version"))

	updates := map[string]interface{}{}

	// 更新智能体类型
	if newAgentType != "" {
		if err := checkAgentTypeValid(r.Context(), newAgentType); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		// 【防护 2】启用中镜像禁止变更 agent_type
		if img.Enabled && newAgentType != img.AgentType {
			slog.WarnContext(ctx, "[Image] 拒绝修改启用中镜像的类型",
				"id", img.ID, "image_id", img.ImageId,
				"current_type", img.AgentType, "new_type", newAgentType)
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnabledImageTypeCantChange))
			return
		}
		if newAgentType != img.AgentType {
			updates["agent_type"] = newAgentType
		}
	}

	// 更新版本号
	if newAgentVersion != "" {
		if err := checkAgentVersionValid(newAgentVersion); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		if newAgentVersion != img.AgentVersion {
			updates["agent_version"] = newAgentVersion
		}
	}

	// 【防护 3】type + version 组合校验（按类型约束格式）
	// 若本次更新了任一字段，需要对最终生效的组合做 ValidateAgentVersion
	if len(updates) > 0 {
		effectiveType := img.AgentType
		if v, ok := updates["agent_type"].(string); ok {
			effectiveType = v
		}
		effectiveVersion := img.AgentVersion
		if v, ok := updates["agent_version"].(string); ok {
			effectiveVersion = v
		}
		if effectiveType != "" && effectiveVersion != "" {
			if err := model.ValidateAgentVersion(ctx, effectiveType, effectiveVersion); err != nil {
				writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
				return
			}
		}
	}

	if len(updates) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNoFieldToUpdateImage))
		return
	}

	if err := model.DB(r.Context()).Model(&img).Updates(updates).Error; err != nil {
		slog.ErrorContext(ctx, "[Image] 更新镜像失败", "id", img.ID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgUpdateFailed))
		return
	}

	slog.InfoContext(ctx, "[Image] 更新镜像成功",
		"id", img.ID, "image_id", img.ImageId, "updates", updates)

	jsonOK(w, map[string]interface{}{
		"ok": true,
	})
}

// HandleSetDefaultAgentType 设置用户端首选智能体类型
func HandleSetDefaultAgentType(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ctx := r.Context()
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	agentType := strings.TrimSpace(r.FormValue("agent_type"))
	if agentType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentTypeCannotBeEmpty))
		return
	}

	if err := checkAgentTypeValid(r.Context(), agentType); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 检查该类型是否有启用镜像
	enabledImage, err := model.GetEnabledImageByType(r.Context(), agentType)
	if err != nil {
		slog.ErrorContext(ctx, "[SetDefaultAgentType] 查询启用镜像失败",
			"agent_type", agentType, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryImageFailed))
		return
	}
	if enabledImage == nil {
		typeName := model.GetAgentTypeDisplayName(ctx, agentType)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDefaultAgentTypeNoEnabled, typeName))
		return
	}

	oldDefault := model.GetDefaultAgentType(r.Context())

	if err := model.SetDefaultAgentType(r.Context(), agentType); err != nil {
		slog.ErrorContext(ctx, "[SetDefaultAgentType] 设置失败",
			"agent_type", agentType, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSetDefaultFailed))
		return
	}

	slog.InfoContext(ctx, "[SetDefaultAgentType] 设置用户端首选类型成功",
		"old_type", oldDefault, "new_type", agentType)

	jsonOK(w, map[string]interface{}{
		"ok": true,
	})
}

// matchPublicImageType 根据 os_name 匹配公共镜像的类型和版本
func matchPublicImageType(osName string) (agentType, agentVersion string, matched bool) {
	osNameLower := strings.ToLower(osName)

	// Hermes Agent
	if strings.Contains(osNameLower, "hermes") || strings.Contains(osNameLower, "hermesagent") {
		return model.AgentTypeHermes, "0.9.0", true
	}

	// LightClaw ACE
	if strings.Contains(osNameLower, "lightclawace") || strings.Contains(osNameLower, "lightclaw_ace") {
		return model.AgentTypeLightclawACE, "0.1.1", true
	}

	return "", "", false
}

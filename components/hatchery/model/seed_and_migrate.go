package model

import (
	"context"
	"log/slog"

	"hatchery/common"

	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// CVMClientFactory 创建 CVM 客户端的工厂函数类型。
// 由调用方（task 包）注入 controller.NewCVMClient，避免 model → controller 循环依赖。
type CVMClientFactory func(ctx context.Context) (*cvm.Client, error)

// RunAllSeeds 为当前租户执行所有幂等 Seed 操作。
// 供 task scheduler 的一次性任务和 /tenants/init 接口共用。
// cvmClientFn 可为 nil（/tenants/init 场景不需要初始化镜像）。
func RunAllSeeds(ctx context.Context, cvmClientFn CVMClientFactory) {
	db := DB(ctx)

	if err := SeedChannels(ctx, db); err != nil {
		slog.Error("[Seed] SeedChannels 失败", "error", err)
	}
	if err := SeedModels(db); err != nil {
		slog.Error("[Seed] SeedModels 失败", "error", err)
	}
	if err := SeedDefaultSkillBundle(ctx, db); err != nil {
		slog.Error("[Seed] SeedDefaultSkillBundle 失败", "error", err)
	}
	if err := SeedDefaultRoles(ctx, db); err != nil {
		slog.Error("[Seed] SeedDefaultRoles 失败", "error", err)
	}
	if err := MigrateRenamedDefaultRoles(ctx, db); err != nil {
		slog.Error("[Seed] MigrateRenamedDefaultRoles 失败", "error", err)
	}
	if err := SeedDefaultRoleSkills(ctx, db); err != nil {
		slog.Error("[Seed] SeedDefaultRoleSkills 失败", "error", err)
	}
	if err := SeedCategories(ctx, db); err != nil {
		slog.Error("[Seed] SeedCategories 失败", "error", err)
	}
	if err := SeedPluginCategories(db); err != nil {
		slog.Error("[Seed] SeedPluginCategories 失败", "error", err)
	}
	if err := SeedDefaultPluginBundle(db); err != nil {
		slog.Error("[Seed] SeedDefaultPluginBundle 失败", "error", err)
	}

	// 候选镜像初始化（需要 CVM API 凭证）
	if cvmClientFn != nil {
		SeedAvailableImages(ctx, cvmClientFn)
	}
}

// SeedAvailableImages 通过 DescribeImages API 依次检查候选镜像的可用性，
// 将所有有权限的候选镜像写入数据库。幂等操作。
func SeedAvailableImages(ctx context.Context, newClient CVMClientFactory) {
	candidateSet := make(map[string]bool, len(common.CandidateImages))
	for _, c := range common.CandidateImages {
		candidateSet[c.ImageId] = true
	}

	var existingImages []AIImage
	DB(ctx).Find(&existingImages)

	existingCandidateImages := make([]AIImage, 0)
	hasCustomImages := false
	for _, img := range existingImages {
		if candidateSet[img.ImageId] {
			existingCandidateImages = append(existingCandidateImages, img)
		} else {
			hasCustomImages = true
		}
	}

	if hasCustomImages {
		slog.Info("检测到用户自定义镜像，仅处理候选镜像", "customCount", len(existingImages)-len(existingCandidateImages))
	}

	client, rerr := newClient(ctx)
	if rerr != nil {
		slog.Warn("创建 CVM 客户端失败，跳过镜像初始化", "error", rerr)
		return
	}

	latestHistories, err := LatestOfficialImageHistories(ctx)
	if err != nil {
		slog.Warn("查询镜像更新历史失败，使用候选镜像默认版本", "error", err)
		latestHistories = map[string]ImageHistory{}
	}

	var available []AIImage
	for _, candidate := range common.CandidateImages {
		request := cvm.NewDescribeImagesRequest()
		request.ImageIds = tccommon.StringPtrs([]string{candidate.ImageId})

		response, err := client.DescribeImages(request)
		if err != nil {
			slog.Info("候选镜像无权限或查询失败", "imageId", candidate.ImageId, "error", err)
			continue
		}
		if len(response.Response.ImageSet) == 0 {
			slog.Info("候选镜像未找到", "imageId", candidate.ImageId)
			continue
		}

		img := response.Response.ImageSet[0]
		agentType, agentVersion := candidateAgentVersionFromHistory(candidate, latestHistories)
		available = append(available, AIImage{
			ImageId:      candidate.ImageId,
			ImageName:    candidate.ImageName,
			ImageType:    strVal(img.ImageType),
			OsName:       strVal(img.OsName),
			ImageSize:    int64Val(img.ImageSize),
			ImageState:   strVal(img.ImageState),
			AgentType:    agentType,
			AgentVersion: agentVersion,
		})
		slog.Info("候选镜像可用", "imageId", candidate.ImageId, "imageName", candidate.ImageName)
	}

	if len(available) == 0 {
		slog.Warn("没有可用的候选镜像，跳过初始化")
		return
	}

	availableSet := make(map[string]bool, len(available))
	for _, img := range available {
		availableSet[img.ImageId] = true
	}

	for _, img := range existingCandidateImages {
		if !availableSet[img.ImageId] {
			DB(ctx).Delete(&img)
			slog.Info("删除无权限的候选镜像", "imageId", img.ImageId, "imageName", img.ImageName)
		}
	}

	for _, img := range available {
		var existing AIImage
		if DB(ctx).Where("image_id = ?", img.ImageId).First(&existing).Error == nil {
			DB(ctx).Model(&existing).Updates(map[string]interface{}{
				"image_name":    img.ImageName,
				"image_type":    img.ImageType,
				"os_name":       img.OsName,
				"image_size":    img.ImageSize,
				"image_state":   img.ImageState,
				"agent_type":    img.AgentType,
				"agent_version": img.AgentVersion,
			})
		} else {
			if result := DB(ctx).Create(&img); result.Error != nil {
				slog.Info("镜像已被其他实例写入，跳过", "imageId", img.ImageId, "error", result.Error)
			} else {
				slog.Info("初始化候选镜像", "imageId", img.ImageId)
			}
		}
	}

	enableDefaultImagesByBuiltinTypes(ctx, available)
}

func enableDefaultImagesByBuiltinTypes(ctx context.Context, available []AIImage) {
	var enabledImages []AIImage
	if err := DB(ctx).Where("enabled = ?", true).Find(&enabledImages).Error; err != nil {
		slog.Warn("查询启用镜像失败，跳过内置类型默认镜像补齐", "error", err)
		return
	}

	enabledTypes := make(map[string]bool, len(enabledImages))
	for _, img := range enabledImages {
		enabledTypes[NormalizeAgentType(img.AgentType)] = true
	}

	disabledTypes := GetDisabledAgentTypes(ctx)
	disabledSet := make(map[string]bool, len(disabledTypes))
	for _, agentType := range disabledTypes {
		disabledSet[agentType] = true
	}
	defaultType := GetDefaultAgentType(ctx)

	for _, img := range available {
		agentType := NormalizeAgentType(img.AgentType)
		if !IsBuiltinAgentType(agentType) || enabledTypes[agentType] {
			continue
		}
		if agentType != defaultType && !disabledSet[agentType] {
			if err := SetDisabledAgentTypes(ctx, append(disabledTypes, agentType)); err != nil {
				slog.Warn("禁用自动补齐默认镜像的智能体类型失败，跳过启用镜像", "agent_type", agentType, "error", err)
				continue
			}
			disabledTypes = append(disabledTypes, agentType)
			disabledSet[agentType] = true
		}
		result := DB(ctx).Model(&AIImage{}).Where("image_id = ?", img.ImageId).Update("enabled", true)
		if result.Error != nil {
			slog.Warn("启用默认候选镜像失败", "imageId", img.ImageId, "agent_type", agentType, "error", result.Error)
			continue
		}
		if result.RowsAffected == 0 {
			slog.Warn("默认候选镜像不存在，跳过启用", "imageId", img.ImageId, "agent_type", agentType)
			continue
		}
		enabledTypes[agentType] = true
		slog.Info("按内置类型启用默认候选镜像", "imageId", img.ImageId, "imageName", img.ImageName, "agent_type", agentType)
	}
}

func candidateAgentVersionFromHistory(candidate common.CandidateImage, histories map[string]ImageHistory) (agentType, agentVersion string) {
	agentType = NormalizeAgentType(candidate.AgentType)
	agentVersion = candidate.AgentVersion
	if history, ok := histories[candidate.ImageId]; ok {
		if history.AgentType != "" {
			agentType = history.AgentType
		}
		if history.AgentVersion != "" {
			agentVersion = history.AgentVersion
		}
	}
	return agentType, agentVersion
}

// strVal 安全解引用字符串指针。
func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// int64Val 安全解引用 int64 指针。
func int64Val(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

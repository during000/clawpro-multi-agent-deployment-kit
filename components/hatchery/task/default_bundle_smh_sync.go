package task

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"hatchery/common"
	"hatchery/controller"
	"hatchery/model"
)

func init() {
	RegisterTask(TaskDef{
		Name:         "default-bundle-smh-sync",
		Interval:     0, // 一次性
		RunFunc:      runDefaultBundleSMHSync,
		NeedDistLock: true,
		PerTenant:    true,
	})
}

// runDefaultBundleSMHSync 等待 SMH 就绪后将默认技能包的技能 zip
// 从 SkillHub 公共下载接口下载并上传到 SMH common space，填充 BundleSkill.cos_zip_key。
func runDefaultBundleSMHSync(ctx context.Context) {
	logger := slog.With("task", "DefaultBundleSMHSync")

	// ──────── 等待 SMH 就绪 ────────
	const (
		maxRetries    = 20
		retryInterval = 30 * time.Second
	)

	smhReady := false
	for attempt := 1; attempt <= maxRetries; attempt++ {
		config := model.GetSiteConfig(ctx)
		if config.SMHEnabled != 1 {
			logger.Info("SMH 未就绪，等待重试", "attempt", attempt, "reason", "SMHEnabled != 1")
			time.Sleep(retryInterval)
			continue
		}
		if _, err := controller.GetCommonSpaceToken(ctx); err != nil {
			logger.Info("SMH 已开通但 Token 未初始化，尝试初始化", "attempt", attempt)
			controller.EnsureSMHTokenReady(ctx)
			time.Sleep(5 * time.Second)
			if _, err := controller.GetCommonSpaceToken(ctx); err != nil {
				logger.Info("SMH 未就绪，等待重试", "attempt", attempt, "reason", err.Error())
				time.Sleep(retryInterval)
				continue
			}
		}
		smhReady = true
		break
	}

	if !smhReady {
		logger.Warn("等待 SMH 就绪超时，跳过默认技能包 SMH 同步")
		return
	}
	logger.Info("SMH 已就绪，开始同步默认技能包到 SMH")

	defaultLang := common.DefaultLangFromCtx(ctx)
	defaultBundleName := model.DefaultBundleNameEn
	if defaultLang != "en" {
		defaultBundleName = model.DefaultBundleName
	}

	// ──────── 查询默认技能包 ────────
	var bundle model.SkillBundle
	if model.DB(ctx).Where("name = ?", defaultBundleName).First(&bundle).Error != nil {
		logger.Info("默认技能包不存在，跳过 SMH 同步")
		return
	}

	// ──────── 获取 common space 存储客户端 ────────
	commonClient, err := controller.GetCommonStorageClient(ctx)
	if err != nil {
		logger.Error("获取 common space 存储客户端失败", "error", err)
		return
	}

	// ──────── 同步默认技能包技能 ────────
	var skills []model.BundleSkill
	model.DB(ctx).Where("skill_bundle_id = ? AND cos_zip_key = ''", bundle.ID).Find(&skills)
	if len(skills) == 0 {
		logger.Info("默认技能包所有技能已同步，跳过")
	}

	syncCount := 0
	for _, skill := range skills {
		downloadURL := fmt.Sprintf("%s/api/v1/download?slug=%s&version=%s",
			controller.SkillAPIBaseURL, skill.Slug, skill.Version)

		resp, err := controller.SkillHTTPClient.Get(downloadURL)
		if err != nil {
			logger.Error("下载公共技能 zip 失败", "slug", skill.Slug, "error", err)
			continue
		}
		zipData, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			logger.Error("读取公共技能 zip 失败", "slug", skill.Slug,
				"status", resp.StatusCode, "error", err)
			continue
		}

		cosZipKey := fmt.Sprintf("skill-bundles/%s/%s/%s-%s.zip",
			bundle.Name, skill.Slug, skill.Slug, skill.Version)
		if err := commonClient.Upload(cosZipKey, zipData, "application/zip"); err != nil {
			logger.Error("上传技能 zip 到 common space 失败",
				"slug", skill.Slug, "cos_zip_key", cosZipKey, "error", err)
			continue
		}

		model.DB(ctx).Model(&skill).Update("cos_zip_key", cosZipKey)
		logger.Info("技能 zip 同步成功", "slug", skill.Slug, "cos_zip_key", cosZipKey)
		syncCount++
	}
	logger.Info("默认技能包 SMH 同步完成", "synced", syncCount, "total", len(skills))

	// ──────── 同步角色技能到 SMH ────────
	var roleSkills []model.OpenClawRoleSkill
	model.DB(ctx).Where("cos_zip_key = '' AND source = 'public'").Find(&roleSkills)
	if len(roleSkills) == 0 {
		logger.Info("所有角色技能已同步，跳过")
		return
	}

	logger.Info("开始同步角色技能到 SMH", "count", len(roleSkills))
	rolleSyncCount := 0
	for _, rs := range roleSkills {
		downloadURL := fmt.Sprintf("%s/api/v1/download?slug=%s&version=%s",
			controller.SkillAPIBaseURL, rs.Slug, rs.Version)

		resp, err := controller.SkillHTTPClient.Get(downloadURL)
		if err != nil {
			logger.Error("下载角色技能 zip 失败", "slug", rs.Slug, "error", err)
			continue
		}
		zipData, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			logger.Error("读取角色技能 zip 失败", "slug", rs.Slug,
				"status", resp.StatusCode, "error", err)
			continue
		}

		cosZipKey := fmt.Sprintf("role-skills/%s/%s-%s.zip", rs.Slug, rs.Slug, rs.Version)
		if err := commonClient.Upload(cosZipKey, zipData, "application/zip"); err != nil {
			logger.Error("上传角色技能 zip 到 common space 失败",
				"slug", rs.Slug, "cos_zip_key", cosZipKey, "error", err)
			continue
		}

		model.DB(ctx).Model(&model.OpenClawRoleSkill{}).
			Where("slug = ? AND version = ? AND cos_zip_key = ''", rs.Slug, rs.Version).
			Update("cos_zip_key", cosZipKey)
		logger.Info("角色技能 zip 同步成功", "slug", rs.Slug, "cos_zip_key", cosZipKey)
		rolleSyncCount++
	}
	logger.Info("角色技能 SMH 同步完成", "synced", rolleSyncCount, "total", len(roleSkills))
}

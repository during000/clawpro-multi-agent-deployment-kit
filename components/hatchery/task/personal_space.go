package task

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"hatchery/controller"
	"hatchery/model"
)

const (
	// 个人空间环境（skill和token）同步（安装/卸载）每 30 分钟扫描一次，同时处理安装卸载情况
	personalSpaceEnvSyncInterval = 30 * time.Minute
	// Token 刷新高频扫描（30分钟一次）配合 personalSpaceTokenRepushThreshold 的 SQL 门控实现
	// "高频扫描 + 低频下发"：稳态实例 SQL 直接筛除、关机重开实例快速补发。
	personalSpaceTokenRefreshInterval = 30 * time.Minute
	// CVM 上 token 剩余不足该阈值才重新下发，在 SQL WHERE 中直接过滤。
	// 注意不要和 controller.personalSpaceTokenRefreshBefore(18h, 进程内缓存换新阈值) 混淆，
	// 后者控制 "何时向 SMH 申请新 token"，前者控制 "何时把 token 下发到 CVM"，两层独立。
	personalSpaceTokenRepushThreshold = 4 * time.Hour
	personalSpaceRecycleBinInterval   = 12 * time.Hour

	personalSpaceTaskConcurrency = 1
)

func init() {
	RegisterTask(TaskDef{
		Name:         "personal-space-env-sync",
		Interval:     personalSpaceEnvSyncInterval,
		RunFunc:      runPersonalSpaceEnvSync,
		NeedDistLock: true,
		PerTenant:    true,
	})
	RegisterTask(TaskDef{
		Name:         "personal-space-token-refresh",
		Interval:     personalSpaceTokenRefreshInterval,
		RunFunc:      runPersonalSpaceTokenRefresh,
		NeedDistLock: true,
		PerTenant:    true,
	})
	RegisterTask(TaskDef{
		Name:         "personal-space-recycle-bin",
		Interval:     personalSpaceRecycleBinInterval,
		RunFunc:      runPersonalSpaceRecycleBin,
		NeedDistLock: true,
		PerTenant:    true,
	})
}

// runPersonalSpaceEnvSync 是 scheduler 调用的环境同步入口。
func runPersonalSpaceEnvSync(ctx context.Context) {
	svc := newPersonalSpaceService(newDefaultPersonalSpaceServiceDependencies())
	svc.syncEnvs(ctx)
}

// runPersonalSpaceTokenRefresh 是 scheduler 调用的 Token 刷新入口。
func runPersonalSpaceTokenRefresh(ctx context.Context) {
	svc := newPersonalSpaceService(newDefaultPersonalSpaceServiceDependencies())
	svc.refreshTokens(ctx)
}

// runPersonalSpaceRecycleBin 是 scheduler 调用的回收站清理入口。
func runPersonalSpaceRecycleBin(ctx context.Context) {
	svc := newPersonalSpaceService(newDefaultPersonalSpaceServiceDependencies())
	svc.cleanExpired(ctx)
}

// personalSpaceServiceDependencies 是 personalSpaceService 的外部依赖接口，测试时可替换。
type personalSpaceServiceDependencies interface {
	FilterRunningSpaces(ctx context.Context, spaces []model.SMHPersonalSpace) []*model.SMHPersonalSpace
	TriggerSyncEnv(ctx context.Context, space *model.SMHPersonalSpace, install bool) error
	TriggerRefreshToken(ctx context.Context, space *model.SMHPersonalSpace) error
	DeleteSMHSpace(ctx context.Context, endpoint, libraryId, librarySecret, spaceId string) error
}

// defaultPersonalSpaceServiceDependencies 是生产环境使用的真实实现。
type defaultPersonalSpaceServiceDependencies struct{}

func newDefaultPersonalSpaceServiceDependencies() defaultPersonalSpaceServiceDependencies {
	return defaultPersonalSpaceServiceDependencies{}
}

func (defaultPersonalSpaceServiceDependencies) FilterRunningSpaces(ctx context.Context, spaces []model.SMHPersonalSpace) []*model.SMHPersonalSpace {
	return filterPersonalSpacesWithRunningInstance(ctx, spaces)
}
func (defaultPersonalSpaceServiceDependencies) TriggerSyncEnv(ctx context.Context, space *model.SMHPersonalSpace, install bool) error {
	return controller.TriggerSyncPersonalSpaceEnv(ctx, space, install)
}
func (defaultPersonalSpaceServiceDependencies) TriggerRefreshToken(ctx context.Context, space *model.SMHPersonalSpace) error {
	return controller.TriggerRefreshPersonalSpaceToken(ctx, space)
}
func (defaultPersonalSpaceServiceDependencies) DeleteSMHSpace(ctx context.Context, endpoint, libraryId, librarySecret, spaceId string) error {
	return controller.DeleteSMHSpace(ctx, endpoint, libraryId, librarySecret, spaceId)
}

// personalSpaceService 管理个人空间相关的后台任务（保留 struct 供测试依赖注入）。
type personalSpaceService struct {
	deps personalSpaceServiceDependencies
}

func newPersonalSpaceService(deps personalSpaceServiceDependencies) *personalSpaceService {
	return &personalSpaceService{deps: deps}
}

// parallelForEach 对 items 中每个元素并发执行 fn，最大并发度为 concurrency。
func parallelForEach[T any](items []T, concurrency int, fn func(T)) {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i := range items {
		item := items[i]
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			fn(item)
		}()
	}
	wg.Wait()
}

// filterPersonalSpacesWithRunningInstance 从个人空间列表中筛选出 CVM 处于 RUNNING 状态的子集。
func filterPersonalSpacesWithRunningInstance(ctx context.Context, spaces []model.SMHPersonalSpace) []*model.SMHPersonalSpace {
	cvmIds := make([]string, 0, len(spaces))
	for _, s := range spaces {
		cvmIds = append(cvmIds, s.CVMInstanceId)
	}

	slog.Info("[SMH] 批量查询 CVM 状态", "cvm_ids", cvmIds)

	client, err := controller.NewCVMClient(ctx)
	if err != nil {
		slog.Error("[SMH] 创建 CVM 客户端失败", "error", err)
		return nil
	}
	runningIds, filterErr := controller.FilterInstancesByState(client, cvmIds, "RUNNING")
	if filterErr != nil {
		slog.Error("[SMH] 批量查询 CVM 状态失败", "error", filterErr)
		return nil
	}
	runningSet := make(map[string]bool, len(runningIds))
	for _, id := range runningIds {
		runningSet[id] = true
	}

	var nonRunningIds []string
	var result []*model.SMHPersonalSpace
	for i := range spaces {
		if runningSet[spaces[i].CVMInstanceId] {
			result = append(result, &spaces[i])
		} else {
			nonRunningIds = append(nonRunningIds, spaces[i].CVMInstanceId)
		}
	}
	slog.Info("[SMH] 过滤 RUNNING CVM", "running_ids", runningIds, "non_running", nonRunningIds)
	return result
}

func (s *personalSpaceService) syncEnvs(ctx context.Context) {
	// 需要安装：to_be_deleted_at IS NULL AND env_initialized=false
	var toInstall []model.SMHPersonalSpace
	if err := model.DB(ctx).Where("env_initialized = ? AND to_be_deleted_at IS NULL", false).Find(&toInstall).Error; err != nil {
		slog.Error("[SMH] 查询待新装个人空间失败", "error", err)
		return
	}

	// 需要升级：to_be_deleted_at IS NULL AND env_initialized=true AND env_provision_rev 落后
	var toUpgrade []model.SMHPersonalSpace
	if err := model.DB(ctx).Where("env_initialized = ? AND env_provision_rev < ? AND to_be_deleted_at IS NULL", true, controller.CurrentSMHProvisionRev).Find(&toUpgrade).Error; err != nil {
		slog.Error("[SMH] 查询待升级个人空间失败", "target_rev", controller.CurrentSMHProvisionRev, "error", err)
		return
	}
	toInstall = append(toInstall, toUpgrade...)

	// 需要卸载：to_be_deleted_at IS NOT NULL AND env_initialized=true
	var toUninstall []model.SMHPersonalSpace
	if err := model.DB(ctx).Where("to_be_deleted_at IS NOT NULL AND env_initialized = ?", true).Find(&toUninstall).Error; err != nil {
		slog.Error("[SMH] 查询待卸载个人空间失败", "error", err)
		return
	}

	if len(toInstall) == 0 && len(toUninstall) == 0 {
		return
	}

	// 安装/升级：过滤 RUNNING 实例，并发触发
	if len(toInstall) > 0 {
		runningInstall := s.deps.FilterRunningSpaces(ctx, toInstall)
		if len(runningInstall) > 0 {
			slog.Info("[SMH] 发现待安装或升级环境的 RUNNING 实例", "total", len(toInstall), "running", len(runningInstall), "target_rev", controller.CurrentSMHProvisionRev)
			parallelForEach(runningInstall, personalSpaceTaskConcurrency, func(space *model.SMHPersonalSpace) {
				if err := s.deps.TriggerSyncEnv(ctx, space, true); err != nil {
					slog.Error("[SMH] 初始化或升级个人空间环境失败", "instance_id", space.CVMInstanceId, "space_id", space.SpaceId, "error", err)
				}
			})
		}
	}

	// 卸载：过滤 RUNNING 实例，并发触发
	if len(toUninstall) > 0 {
		runningUninstall := s.deps.FilterRunningSpaces(ctx, toUninstall)
		if len(runningUninstall) > 0 {
			slog.Info("[SMH] 发现待卸载环境的 RUNNING 实例", "total", len(toUninstall), "running", len(runningUninstall))
			parallelForEach(runningUninstall, personalSpaceTaskConcurrency, func(space *model.SMHPersonalSpace) {
				if err := s.deps.TriggerSyncEnv(ctx, space, false); err != nil {
					slog.Error("[SMH] 卸载个人空间环境失败", "instance_id", space.CVMInstanceId, "space_id", space.SpaceId, "error", err)
				}
			})
		}
	}
}

func (s *personalSpaceService) refreshTokens(ctx context.Context) {
	// SQL 门控：只查 "从未下发过" 或 "CVM 上 token 剩余时间不足阈值" 的实例，
	threshold := time.Now().Add(personalSpaceTokenRepushThreshold)
	var spaces []model.SMHPersonalSpace
	if err := model.DB(ctx).Where("env_initialized = ? AND to_be_deleted_at IS NULL "+
		"AND (last_pushed_token_expires_at IS NULL OR last_pushed_token_expires_at < ?)",
		true, threshold,
	).Select("id, space_id, instance_id, c_vm_instance_id").Find(&spaces).Error; err != nil {
		slog.Error("[SMH] 查询待刷新 token 的个人空间失败", "error", err)
		return
	}
	if len(spaces) == 0 {
		slog.Info("[SMH] 未发现需要刷新 token 的个人空间")
		return
	}

	runningSpaces := s.deps.FilterRunningSpaces(ctx, spaces)
	if len(runningSpaces) == 0 {
		slog.Info("[SMH] 未发现 RUNNING 的个人空间")
		return
	}

	parallelForEach(runningSpaces, personalSpaceTaskConcurrency, func(space *model.SMHPersonalSpace) {
		slog.Info("[SMH] 后台刷新 token", "instance_id", space.CVMInstanceId, "space_id", space.SpaceId)
		if err := s.deps.TriggerRefreshToken(ctx, space); err != nil {
			slog.Warn("[SMH] 后台刷新 token 失败", "instance_id", space.CVMInstanceId, "space_id", space.SpaceId, "error", err)
		}
	})
}

func (s *personalSpaceService) cleanExpired(ctx context.Context) {
	var spaces []model.SMHPersonalSpace
	model.DB(ctx).Where("to_be_deleted_at IS NOT NULL AND to_be_deleted_at <= ?", time.Now()).Find(&spaces)
	if len(spaces) == 0 {
		return
	}

	smhConfig := model.GetSMHConfig(ctx)

	slog.Info("[SMH] 开始清理过期个人空间", "count", len(spaces))

	parallelForEach(spaces, personalSpaceTaskConcurrency, func(space model.SMHPersonalSpace) {
		if smhConfig.IsConfigured() && space.SpaceId != "" {
			if err := s.deps.DeleteSMHSpace(ctx, smhConfig.Endpoint, smhConfig.LibraryId, smhConfig.LibrarySecret, space.SpaceId); err != nil {
				slog.Warn("[SMH] 删除 SMH 空间失败，等待下次重试", "id", space.ID, "space_id", space.SpaceId, "error", err)
				return
			}
		}

		controller.InvalidatePersonalSpaceTokenCache(space.SpaceId)

		if err := model.DB(ctx).Unscoped().Delete(&space).Error; err != nil {
			slog.Error("[SMH] 删除个人空间记录失败", "id", space.ID, "space_id", space.SpaceId, "error", err)
			return
		}
		slog.Info("[SMH] 个人空间已回收", "id", space.ID, "space_id", space.SpaceId, "instance_id", space.CVMInstanceId)
	})
}

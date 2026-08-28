package task

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"hatchery/common"
	"hatchery/controller"
	"hatchery/model"
)

// ---------- DB 初始化 ----------

func initPersonalSpaceTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.SMHPersonalSpace{},
		&model.SiteConfig{},
		&model.SMHSpace{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	oldSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: ""}
	t.Cleanup(func() { common.FixedSnapshot = oldSnap })
}

// setupSMHConfig 向数据库写入完整的 SMH 配置，使 IsConfigured() 返回 true。
func setupSMHConfig(t *testing.T) {
	t.Helper()
	model.DB(context.Background()).Create(&model.SiteConfig{
		SMHEndpoint:      "https://smh.example.com",
		SMHLibraryId:     "lib-001",
		SMHLibrarySecret: "secret",
	})
	model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "common", SpaceId: "sp-common", LibraryId: "lib-001", Purpose: "common"})
	model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "skillhub", SpaceId: "sp-skillhub", LibraryId: "lib-001", Purpose: "skillhub"})
}

func createSpace(t *testing.T, cvmId string, envInitialized bool, toBeDeletedAt *time.Time) model.SMHPersonalSpace {
	t.Helper()
	ins := model.Instance{Name: "ins-" + cvmId, InstanceId: cvmId}
	model.DB(context.Background()).Create(&ins)
	space := model.SMHPersonalSpace{
		SpaceId:        "sp-" + cvmId,
		UserId:         1,
		InstanceId:     ins.ID,
		UserName:       "user",
		InstanceName:   ins.Name,
		CVMInstanceId:  cvmId,
		EnvInitialized: envInitialized,
		ToBeDeletedAt:  toBeDeletedAt,
	}
	model.DB(context.Background()).Create(&space)
	return space
}

// ---------- mock dependencies ----------

type mockPersonalSpaceServiceDependencies struct {
	filterRunningSpaces func(spaces []model.SMHPersonalSpace) []*model.SMHPersonalSpace
	triggerSyncEnv      func(ctx context.Context, space *model.SMHPersonalSpace, install bool) error
	triggerRefreshToken func(ctx context.Context, space *model.SMHPersonalSpace) error
	deleteSMHSpace      func(ctx context.Context, endpoint, libraryId, librarySecret, spaceId string) error
}

func (m *mockPersonalSpaceServiceDependencies) FilterRunningSpaces(ctx context.Context, spaces []model.SMHPersonalSpace) []*model.SMHPersonalSpace {
	if m.filterRunningSpaces != nil {
		return m.filterRunningSpaces(spaces)
	}
	// 默认：全部视为 RUNNING
	result := make([]*model.SMHPersonalSpace, len(spaces))
	for i := range spaces {
		result[i] = &spaces[i]
	}
	return result
}
func (m *mockPersonalSpaceServiceDependencies) TriggerSyncEnv(ctx context.Context, space *model.SMHPersonalSpace, install bool) error {
	if m.triggerSyncEnv != nil {
		return m.triggerSyncEnv(ctx, space, install)
	}
	return nil
}
func (m *mockPersonalSpaceServiceDependencies) TriggerRefreshToken(ctx context.Context, space *model.SMHPersonalSpace) error {
	if m.triggerRefreshToken != nil {
		return m.triggerRefreshToken(ctx, space)
	}
	return nil
}
func (m *mockPersonalSpaceServiceDependencies) DeleteSMHSpace(ctx context.Context, endpoint, libraryId, librarySecret, spaceId string) error {
	if m.deleteSMHSpace != nil {
		return m.deleteSMHSpace(ctx, endpoint, libraryId, librarySecret, spaceId)
	}
	return nil
}

func newTestService(deps *mockPersonalSpaceServiceDependencies) *personalSpaceService {
	return newPersonalSpaceService(deps)
}

// ---------- parallelForEach ----------

func TestParallelForEach_AllItemsProcessed(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	var count atomic.Int32
	parallelForEach(items, 3, func(i int) { count.Add(1) })
	if int(count.Load()) != len(items) {
		t.Errorf("期望处理 %d 个元素，实际 %d", len(items), count.Load())
	}
}

func TestParallelForEach_EmptySlice(t *testing.T) {
	parallelForEach([]int{}, 3, func(i int) { t.Error("不应被调用") })
}

// ---------- refreshTokens ----------

func TestRefreshTokens_NoSpaces(t *testing.T) {
	initPersonalSpaceTestDB(t)

	called := false
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		filterRunningSpaces: func(spaces []model.SMHPersonalSpace) []*model.SMHPersonalSpace {
			called = true
			return nil
		},
	})

	svc.refreshTokens(context.Background())

	if called {
		t.Error("无空间时不应调用 FilterRunningSpaces")
	}
}

func TestRefreshTokens_NoRunningInstances(t *testing.T) {
	initPersonalSpaceTestDB(t)
	createSpace(t, "ins-aaa", true, nil)

	refreshCalled := false
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		filterRunningSpaces: func(spaces []model.SMHPersonalSpace) []*model.SMHPersonalSpace {
			return nil // 没有 RUNNING 实例
		},
		triggerRefreshToken: func(_ context.Context, _ *model.SMHPersonalSpace) error {
			refreshCalled = true
			return nil
		},
	})

	svc.refreshTokens(context.Background())

	if refreshCalled {
		t.Error("无 RUNNING 实例时不应调用 TriggerRefreshToken")
	}
}

func TestRefreshTokens_Success(t *testing.T) {
	initPersonalSpaceTestDB(t)
	createSpace(t, "ins-001", true, nil)
	createSpace(t, "ins-002", true, nil)

	var refreshCount atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerRefreshToken: func(_ context.Context, _ *model.SMHPersonalSpace) error {
			refreshCount.Add(1)
			return nil
		},
	})

	svc.refreshTokens(context.Background())

	if refreshCount.Load() != 2 {
		t.Errorf("期望刷新 2 个空间，实际 %d", refreshCount.Load())
	}
	// 字段回写已移到 controller 层 refreshPersonalSpaceToken 内部，
	// 对应的回写断言见 controller/smh_test.go:TestRefreshPersonalSpaceToken_Success。
}

func TestRefreshTokens_PartialError(t *testing.T) {
	initPersonalSpaceTestDB(t)
	createSpace(t, "ins-001", true, nil)
	createSpace(t, "ins-002", true, nil)

	var refreshCount atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerRefreshToken: func(_ context.Context, space *model.SMHPersonalSpace) error {
			refreshCount.Add(1)
			if space.CVMInstanceId == "ins-001" {
				return errors.New("token 刷新失败")
			}
			return nil
		},
	})

	svc.refreshTokens(context.Background())

	if refreshCount.Load() != 2 {
		t.Errorf("部分失败时期望仍处理全部 2 个空间，实际 %d", refreshCount.Load())
	}
}

// TestRefreshTokens_GatedByLastPushedExpiresAt 验证门控：
//   - 剩余充裕的 space 跳过；
//   - 剩余不足阈值的 space 下发；
//   - LastPushedTokenExpiresAt 为 nil 的 space 下发。
func TestRefreshTokens_GatedByLastPushedExpiresAt(t *testing.T) {
	initPersonalSpaceTestDB(t)
	fresh := createSpace(t, "ins-fresh", true, nil) // LastPushedTokenExpiresAt == nil
	nearExpire := createSpace(t, "ins-near", true, nil)
	stable := createSpace(t, "ins-stable", true, nil)

	near := time.Now().Add(1 * time.Hour) // < 6h 阈值
	far := time.Now().Add(20 * time.Hour) // > 6h 阈值
	if err := model.DB(context.Background()).Model(&nearExpire).Update("last_pushed_token_expires_at", near).Error; err != nil {
		t.Fatalf("预置 near 字段失败: %v", err)
	}
	if err := model.DB(context.Background()).Model(&stable).Update("last_pushed_token_expires_at", far).Error; err != nil {
		t.Fatalf("预置 stable 字段失败: %v", err)
	}

	var called []string
	var mu sync.Mutex
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerRefreshToken: func(_ context.Context, space *model.SMHPersonalSpace) error {
			mu.Lock()
			called = append(called, space.CVMInstanceId)
			mu.Unlock()
			return nil
		},
	})

	svc.refreshTokens(context.Background())

	// fresh (nil) 和 near (< 6h) 应被下发，stable (> 6h) 应跳过
	if len(called) != 2 {
		t.Fatalf("期望下发 2 个 space（fresh + near），实际 %d: %v", len(called), called)
	}
	got := map[string]bool{called[0]: true, called[1]: true}
	if !got["ins-fresh"] || !got["ins-near"] {
		t.Errorf("期望下发 ins-fresh 和 ins-near，实际 %v", called)
	}
	if got["ins-stable"] {
		t.Error("剩余充裕的 ins-stable 不应被下发")
	}
	_ = fresh
}

// ---------- syncEnvs ----------

func TestSyncEnvs_NoWork(t *testing.T) {
	initPersonalSpaceTestDB(t)

	called := false
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		filterRunningSpaces: func(spaces []model.SMHPersonalSpace) []*model.SMHPersonalSpace {
			called = true
			return nil
		},
	})

	svc.syncEnvs(context.Background())

	if called {
		t.Error("无待处理空间时不应调用 FilterRunningSpaces")
	}
}

func TestSyncEnvs_InstallAndUninstall(t *testing.T) {
	initPersonalSpaceTestDB(t)
	deleteAt := time.Now().Add(time.Hour)
	createSpace(t, "ins-install", false, nil)        // 待安装
	createSpace(t, "ins-uninstall", true, &deleteAt) // 待卸载

	var installCalled, uninstallCalled atomic.Bool
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerSyncEnv: func(_ context.Context, _ *model.SMHPersonalSpace, install bool) error {
			if install {
				installCalled.Store(true)
			} else {
				uninstallCalled.Store(true)
			}
			return nil
		},
	})

	svc.syncEnvs(context.Background())

	if !installCalled.Load() {
		t.Error("期望触发安装")
	}
	if !uninstallCalled.Load() {
		t.Error("期望触发卸载")
	}
}

func TestSyncEnvs_ErrorBranches(t *testing.T) {
	initPersonalSpaceTestDB(t)
	deleteAt := time.Now().Add(time.Hour)
	createSpace(t, "ins-install-1", false, nil)
	createSpace(t, "ins-install-2", false, nil)
	createSpace(t, "ins-uninstall-1", true, &deleteAt)
	createSpace(t, "ins-uninstall-2", true, &deleteAt)

	var installCount, uninstallCount atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerSyncEnv: func(_ context.Context, _ *model.SMHPersonalSpace, install bool) error {
			if install {
				installCount.Add(1)
			} else {
				uninstallCount.Add(1)
			}
			return errors.New("模拟失败")
		},
	})

	svc.syncEnvs(context.Background())

	// 即使每个都失败，所有空间仍应被处理
	if installCount.Load() != 2 {
		t.Errorf("期望处理 2 个待安装空间，实际 %d", installCount.Load())
	}
	if uninstallCount.Load() != 2 {
		t.Errorf("期望处理 2 个待卸载空间，实际 %d", uninstallCount.Load())
	}
}

func TestSyncEnvs_FreshInstall_Selected(t *testing.T) {
	initPersonalSpaceTestDB(t)
	space := createSpace(t, "ins-fresh", false, nil)
	if err := model.DB(context.Background()).Model(&space).Update("env_provision_rev", controller.CurrentSMHProvisionRev).Error; err != nil {
		t.Fatalf("预置 env_provision_rev 失败: %v", err)
	}

	var installCount atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerSyncEnv: func(_ context.Context, got *model.SMHPersonalSpace, install bool) error {
			if install && got.CVMInstanceId == "ins-fresh" {
				installCount.Add(1)
			}
			return nil
		},
	})

	svc.syncEnvs(context.Background())

	if installCount.Load() != 1 {
		t.Errorf("fresh install 应被选中 1 次，实际 %d", installCount.Load())
	}
}

func TestSyncEnvs_UpgradeLaggedRev_Selected(t *testing.T) {
	initPersonalSpaceTestDB(t)
	space := createSpace(t, "ins-upgrade", true, nil)
	if err := model.DB(context.Background()).Model(&space).Update("env_provision_rev", controller.CurrentSMHProvisionRev-1).Error; err != nil {
		t.Fatalf("预置 env_provision_rev 失败: %v", err)
	}

	var installCount atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerSyncEnv: func(_ context.Context, got *model.SMHPersonalSpace, install bool) error {
			if install && got.CVMInstanceId == "ins-upgrade" {
				installCount.Add(1)
			}
			return nil
		},
	})

	svc.syncEnvs(context.Background())

	if installCount.Load() != 1 {
		t.Errorf("落后 rev 的空间应被选中升级 1 次，实际 %d", installCount.Load())
	}
}

func TestSyncEnvs_UpToDate_NotSelected(t *testing.T) {
	initPersonalSpaceTestDB(t)
	space := createSpace(t, "ins-current", true, nil)
	if err := model.DB(context.Background()).Model(&space).Update("env_provision_rev", controller.CurrentSMHProvisionRev).Error; err != nil {
		t.Fatalf("预置 env_provision_rev 失败: %v", err)
	}

	var installCount atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerSyncEnv: func(_ context.Context, _ *model.SMHPersonalSpace, install bool) error {
			if install {
				installCount.Add(1)
			}
			return nil
		},
	})

	svc.syncEnvs(context.Background())

	if installCount.Load() != 0 {
		t.Errorf("已达当前 rev 的空间不应被安装/升级，实际 %d", installCount.Load())
	}
}

func TestSyncEnvs_RecycleBin_NotSelectedForInstall(t *testing.T) {
	initPersonalSpaceTestDB(t)
	deleteAt := time.Now().Add(time.Hour)
	space := createSpace(t, "ins-recycle", true, &deleteAt)
	if err := model.DB(context.Background()).Model(&space).Update("env_provision_rev", controller.CurrentSMHProvisionRev-1).Error; err != nil {
		t.Fatalf("预置 env_provision_rev 失败: %v", err)
	}

	var installCount atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerSyncEnv: func(_ context.Context, _ *model.SMHPersonalSpace, install bool) error {
			if install {
				installCount.Add(1)
			}
			return nil
		},
	})

	svc.syncEnvs(context.Background())

	if installCount.Load() != 0 {
		t.Errorf("回收站空间不应进入 install/upgrade 路径，实际 %d", installCount.Load())
	}
}

func TestSyncEnvs_QueryError_ReturnsWithoutTrigger(t *testing.T) {
	initPersonalSpaceTestDB(t)
	createSpace(t, "ins-query-error", false, nil)
	if err := model.DB(context.Background()).Migrator().DropTable(&model.SMHPersonalSpace{}); err != nil {
		t.Fatalf("删除个人空间表失败: %v", err)
	}

	var triggerCount atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		triggerSyncEnv: func(_ context.Context, _ *model.SMHPersonalSpace, _ bool) error {
			triggerCount.Add(1)
			return nil
		},
	})

	svc.syncEnvs(context.Background())

	if triggerCount.Load() != 0 {
		t.Errorf("DB 查询失败时不应触发环境同步，实际 %d", triggerCount.Load())
	}
}

func TestSyncEnvs_StoppedFrontDoesNotStarveLaterRunningUpgrade(t *testing.T) {
	initPersonalSpaceTestDB(t)
	stoppedIDs := map[string]bool{
		"ins-stopped-front-001": true,
		"ins-stopped-front-002": true,
	}
	for cvmID := range stoppedIDs {
		space := createSpace(t, cvmID, true, nil)
		if err := model.DB(context.Background()).Model(&space).Update("env_provision_rev", controller.CurrentSMHProvisionRev-1).Error; err != nil {
			t.Fatalf("预置 env_provision_rev 失败: %v", err)
		}
	}
	for _, cvmID := range []string{"ins-running-after-stopped-001", "ins-running-after-stopped-002"} {
		space := createSpace(t, cvmID, true, nil)
		if err := model.DB(context.Background()).Model(&space).Update("env_provision_rev", controller.CurrentSMHProvisionRev-1).Error; err != nil {
			t.Fatalf("预置 env_provision_rev 失败: %v", err)
		}
	}

	var upgraded atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		filterRunningSpaces: func(spaces []model.SMHPersonalSpace) []*model.SMHPersonalSpace {
			result := make([]*model.SMHPersonalSpace, 0, len(spaces))
			for i := range spaces {
				if !stoppedIDs[spaces[i].CVMInstanceId] {
					result = append(result, &spaces[i])
				}
			}
			return result
		},
		triggerSyncEnv: func(_ context.Context, got *model.SMHPersonalSpace, install bool) error {
			if install && got.CVMInstanceId[:len("ins-running-after-stopped-")] == "ins-running-after-stopped-" {
				upgraded.Add(1)
			}
			return nil
		},
	})

	svc.syncEnvs(context.Background())

	if upgraded.Load() != 2 {
		t.Errorf("低 id 停机实例不应阻塞后续 RUNNING 升级实例，期望升级 2 个，实际 %d 个", upgraded.Load())
	}
}

// ---------- cleanExpired ----------

func TestCleanExpired_NoExpired(t *testing.T) {
	initPersonalSpaceTestDB(t)

	called := false
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		deleteSMHSpace: func(_ context.Context, _, _, _, _ string) error {
			called = true
			return nil
		},
	})

	svc.cleanExpired(context.Background())

	if called {
		t.Error("无过期空间时不应调用 DeleteSMHSpace")
	}
}

func TestCleanExpired_SMHNotConfigured_DeletesFromDB(t *testing.T) {
	initPersonalSpaceTestDB(t)
	past := time.Now().Add(-time.Hour)
	createSpace(t, "ins-expired", true, &past)

	var deleteCount atomic.Int32
	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		deleteSMHSpace: func(_ context.Context, _, _, _, _ string) error {
			deleteCount.Add(1)
			return nil
		},
	})

	svc.cleanExpired(context.Background())

	// SMH 未配置时跳过远程删除，但数据库记录应被清理
	if deleteCount.Load() != 0 {
		t.Errorf("SMH 未配置时不应调用远程删除，实际调用 %d 次", deleteCount.Load())
	}
	var remaining []model.SMHPersonalSpace
	model.DB(context.Background()).Unscoped().Find(&remaining)
	if len(remaining) != 0 {
		t.Errorf("过期空间应从数据库删除，剩余 %d 条", len(remaining))
	}
}

func TestCleanExpired_SMHDeleteFail_SkipsDBDelete(t *testing.T) {
	initPersonalSpaceTestDB(t)
	setupSMHConfig(t)
	past := time.Now().Add(-time.Hour)
	createSpace(t, "ins-expired", true, &past)

	svc := newTestService(&mockPersonalSpaceServiceDependencies{
		deleteSMHSpace: func(_ context.Context, _, _, _, _ string) error {
			return errors.New("SMH 删除失败")
		},
	})

	svc.cleanExpired(context.Background())

	// 远程删除失败时不应删除数据库记录，等下次重试
	var remaining []model.SMHPersonalSpace
	model.DB(context.Background()).Unscoped().Find(&remaining)
	if len(remaining) != 1 {
		t.Errorf("远程删除失败时数据库记录应保留，剩余 %d 条", len(remaining))
	}
}

func TestCleanExpired_DBDeleteFail(t *testing.T) {
	initPersonalSpaceTestDB(t)
	past := time.Now().Add(-time.Hour)
	createSpace(t, "ins-expired", true, &past)

	svc := newTestService(&mockPersonalSpaceServiceDependencies{})

	// 设置只读模式，使 Delete 操作失败
	model.DB(context.Background()).Exec("PRAGMA query_only = ON")
	t.Cleanup(func() { model.DB(context.Background()).Exec("PRAGMA query_only = OFF") })

	svc.cleanExpired(context.Background())

	// Delete 失败后记录应仍然存在
	model.DB(context.Background()).Exec("PRAGMA query_only = OFF")
	var remaining []model.SMHPersonalSpace
	model.DB(context.Background()).Unscoped().Find(&remaining)
	if len(remaining) != 1 {
		t.Errorf("DB 删除失败时记录应保留，剩余 %d 条", len(remaining))
	}
}

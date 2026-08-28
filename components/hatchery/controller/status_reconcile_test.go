package controller

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func initReconcileTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Instance{},
		&model.InstanceAdjustment{},
		&model.User{},
		&model.SiteConfig{},
		&model.CustomAgentType{},
		&model.Notification{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// ── Instance 新增缓存字段测试 ──

func TestInstance_CacheFields_DefaultValues(t *testing.T) {
	db := initReconcileTestDB(t)

	inst := &model.Instance{Name: "test", InstanceId: "ins-001"}
	db.Create(inst)

	var loaded model.Instance
	db.First(&loaded, inst.ID)

	if loaded.LastKnownStatus != "" {
		t.Errorf("LastKnownStatus 默认值应为空, 实际=%q", loaded.LastKnownStatus)
	}
	if loaded.CVMTagsJSON != "" {
		t.Errorf("CVMTagsJSON 默认值应为空, 实际=%q", loaded.CVMTagsJSON)
	}
	if loaded.StatusSyncedAt != nil {
		t.Errorf("StatusSyncedAt 默认值应为 nil, 实际=%v", loaded.StatusSyncedAt)
	}
}

func TestInstance_CacheFields_UpdateAndRead(t *testing.T) {
	db := initReconcileTestDB(t)

	inst := &model.Instance{Name: "test", InstanceId: "ins-002"}
	db.Create(inst)

	now := time.Now()
	db.Model(inst).Updates(map[string]any{
		"last_known_status": model.StatusRunning,
		"cvm_tags_json":     `[{"key":"env","value":"prod"}]`,
		"status_synced_at":  now,
	})

	var loaded model.Instance
	db.First(&loaded, inst.ID)

	if loaded.LastKnownStatus != model.StatusRunning {
		t.Errorf("LastKnownStatus=%q, want %q", loaded.LastKnownStatus, model.StatusRunning)
	}
	if loaded.CVMTagsJSON != `[{"key":"env","value":"prod"}]` {
		t.Errorf("CVMTagsJSON=%q, unexpected", loaded.CVMTagsJSON)
	}
	if loaded.StatusSyncedAt == nil {
		t.Error("StatusSyncedAt 应非 nil")
	}
}

// ── UpdateInstanceCachedStatus 测试 ──

func TestUpdateInstanceCachedStatus_AllOperations(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	tests := []struct {
		operation  string
		wantStatus string
	}{
		{model.OpCreate, model.StatusCreating},
		{model.OpReboot, model.StatusLoading},
		{model.OpReinstall, model.StatusLoading},
		{model.OpUpgrade, model.StatusUpgrading},
		{model.OpDelete, model.StatusDestroying},
		{model.OpMigrate, model.StatusLoading},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			inst := &model.Instance{Name: "test-" + tt.operation, InstanceId: "ins-" + tt.operation}
			db.Create(inst)

			model.UpdateInstanceCachedStatus(ctx, inst.ID, tt.operation)

			var loaded model.Instance
			db.First(&loaded, inst.ID)

			if loaded.LastKnownStatus != tt.wantStatus {
				t.Errorf("op=%s: LastKnownStatus=%q, want %q", tt.operation, loaded.LastKnownStatus, tt.wantStatus)
			}
			if loaded.StatusSyncedAt == nil {
				t.Errorf("op=%s: StatusSyncedAt 应非 nil", tt.operation)
			}
		})
	}
}

func TestUpdateInstanceCachedStatus_UnknownOperation(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	inst := &model.Instance{Name: "test-unknown", InstanceId: "ins-unknown"}
	db.Create(inst)

	model.UpdateInstanceCachedStatus(ctx, inst.ID, "unknown_op")

	var loaded model.Instance
	db.First(&loaded, inst.ID)

	if loaded.LastKnownStatus != "" {
		t.Errorf("未知操作不应写入 LastKnownStatus, 实际=%q", loaded.LastKnownStatus)
	}
}

// ── BatchUpdateInstanceStatusCache 测试 ──

func TestBatchUpdateInstanceStatusCache_Basic(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	inst1 := &model.Instance{Name: "inst1", InstanceId: "ins-001"}
	inst2 := &model.Instance{Name: "inst2", InstanceId: "ins-002"}
	db.Create(inst1)
	db.Create(inst2)

	roundStart := time.Now()
	items := []model.InstanceStatusCacheItem{
		{ID: inst1.ID, Status: model.StatusRunning},
		{ID: inst2.ID, Status: model.StatusStopped},
	}

	model.BatchUpdateInstanceStatusCache(ctx, items, roundStart)

	var loaded1, loaded2 model.Instance
	db.First(&loaded1, inst1.ID)
	db.First(&loaded2, inst2.ID)

	if loaded1.LastKnownStatus != model.StatusRunning {
		t.Errorf("inst1 status=%q, want running", loaded1.LastKnownStatus)
	}
	if loaded2.LastKnownStatus != model.StatusStopped {
		t.Errorf("inst2 status=%q, want stopped", loaded2.LastKnownStatus)
	}
}

func TestBatchUpdateInstanceStatusCache_RaceProtection(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	inst := &model.Instance{Name: "race-test", InstanceId: "ins-race"}
	db.Create(inst)

	// 模拟操作即时写：status_synced_at 设为"当前时间"
	freshTime := time.Now().Add(1 * time.Minute)
	db.Model(inst).Updates(map[string]any{
		"last_known_status": model.StatusDestroying,
		"status_synced_at":  freshTime,
	})

	// 后台任务以更早的 roundStartedAt 尝试覆盖 → 应被竞态保护拦截
	olderRound := time.Now().Add(-1 * time.Minute)
	items := []model.InstanceStatusCacheItem{
		{ID: inst.ID, Status: model.StatusRunning},
	}
	model.BatchUpdateInstanceStatusCache(ctx, items, olderRound)

	var loaded model.Instance
	db.First(&loaded, inst.ID)

	// 应保留操作即时写的值，不被后台覆盖
	if loaded.LastKnownStatus != model.StatusDestroying {
		t.Errorf("竞态保护失败：status=%q, want destroying（即时写的值应被保留）", loaded.LastKnownStatus)
	}
}

// ── SiteConfig LastFullSyncFinishedAt 测试 ──

func TestSiteConfig_LastFullSyncFinishedAt(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	// 初始状态：创建 site_config 行
	db.Create(&model.SiteConfig{})

	// 未设置时应为 nil
	got := model.GetLastFullSyncFinishedAt(ctx)
	if got != nil {
		t.Errorf("初始应为 nil, 实际=%v", got)
	}

	// 设置后应可读
	now := time.Now().Truncate(time.Second)
	if err := model.SetLastFullSyncFinishedAt(ctx, now); err != nil {
		t.Fatalf("设置失败: %v", err)
	}

	got = model.GetLastFullSyncFinishedAt(ctx)
	if got == nil {
		t.Fatal("设置后应非 nil")
	}
	if !got.Truncate(time.Second).Equal(now) {
		t.Errorf("时间不匹配: got=%v, want=%v", got.Truncate(time.Second), now)
	}
}

// ── IsStatusCacheReady 测试 ──

func TestIsStatusCacheReady_NotReady(t *testing.T) {
	initReconcileTestDB(t)
	ctx := context.Background()

	// 无 site_config 行 → 不就绪
	if IsStatusCacheReady(ctx) {
		t.Error("无 site_config 行时不应就绪")
	}
}

func TestIsStatusCacheReady_Ready(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	now := time.Now()
	db.Create(&model.SiteConfig{LastFullSyncFinishedAt: &now})

	if !IsStatusCacheReady(ctx) {
		t.Error("设置了 LastFullSyncFinishedAt 后应就绪")
	}
}

// ── shouldRunSideEffects 测试 ──

func TestShouldRunSideEffects(t *testing.T) {
	tests := []struct {
		name   string
		it     lightInstance
		info   *CVMInstanceInfo
		expect bool
	}{
		{
			name:   "无操作无变化",
			it:     lightInstance{InstanceId: "ins-1", LastCVMState: "RUNNING"},
			info:   &CVMInstanceInfo{State: "RUNNING"},
			expect: false,
		},
		{
			name:   "有在途操作",
			it:     lightInstance{InstanceId: "ins-2", CurrentOperation: model.OpReboot},
			info:   &CVMInstanceInfo{State: "RUNNING"},
			expect: true,
		},
		{
			name:   "CVM 消失",
			it:     lightInstance{InstanceId: "ins-3"},
			info:   nil,
			expect: true,
		},
		{
			name:   "CVM 状态变化",
			it:     lightInstance{InstanceId: "ins-4", LastCVMState: "RUNNING"},
			info:   &CVMInstanceInfo{State: "STOPPED"},
			expect: true,
		},
		{
			name:   "计费类型变化触发同步",
			it:     lightInstance{InstanceId: "ins-5", LastCVMState: "RUNNING", InstanceChargeType: "PREPAID"},
			info:   &CVMInstanceInfo{State: "RUNNING", InstanceChargeType: "POSTPAID_BY_HOUR"},
			expect: true,
		},
		{
			name:   "计费类型无变化不触发",
			it:     lightInstance{InstanceId: "ins-6", LastCVMState: "RUNNING", InstanceChargeType: "PREPAID"},
			info:   &CVMInstanceInfo{State: "RUNNING", InstanceChargeType: "PREPAID"},
			expect: false,
		},
		{
			name:   "API_ERROR 不触发",
			it:     lightInstance{InstanceId: "ins-7", LastCVMState: "RUNNING"},
			info:   &CVMInstanceInfo{State: "API_ERROR", InstanceChargeType: "POSTPAID_BY_HOUR"},
			expect: false,
		},
		{
			name:   "无 InstanceId + CVM nil",
			it:     lightInstance{InstanceId: ""},
			info:   nil,
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRunSideEffects(tt.it, tt.info)
			if got != tt.expect {
				t.Errorf("shouldRunSideEffects=%v, want %v", got, tt.expect)
			}
		})
	}
}

// ── setFullSyncFinished 测试 ──

func TestSetFullSyncFinished(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	// setFullSyncFinished 需要 site_configs 行存在
	db.Create(&model.SiteConfig{})

	now := time.Now().Truncate(time.Second)
	setFullSyncFinished(ctx, now)

	got := model.GetLastFullSyncFinishedAt(ctx)
	if got == nil {
		t.Fatal("setFullSyncFinished 后应非 nil")
	}
	if !got.Truncate(time.Second).Equal(now) {
		t.Errorf("时间不匹配: got=%v, want=%v", got.Truncate(time.Second), now)
	}
}

// ── setOperation 即时写 last_known_status 测试 ──

func TestSetOperation_WritesLastKnownStatus(t *testing.T) {
	db := initReconcileTestDB(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	db.Create(user)

	tests := []struct {
		operation  string
		wantStatus string
		withReset  bool
	}{
		{model.OpReboot, model.StatusLoading, true},
		{model.OpDelete, model.StatusDestroying, false},
		{model.OpReinstall, model.StatusLoading, true},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			inst := &model.Instance{Name: "test-" + tt.operation, InstanceId: "ins-" + tt.operation, UserID: user.ID}
			db.Create(inst)

			var err error
			if tt.withReset {
				err = setOperationWithAgentReset(db, inst, tt.operation)
			} else {
				err = setOperation(db, inst, tt.operation)
			}
			if err != nil {
				t.Fatalf("setOperation 失败: %v", err)
			}

			var loaded model.Instance
			db.First(&loaded, inst.ID)

			if loaded.LastKnownStatus != tt.wantStatus {
				t.Errorf("op=%s: LastKnownStatus=%q, want %q", tt.operation, loaded.LastKnownStatus, tt.wantStatus)
			}
			if loaded.StatusSyncedAt == nil {
				t.Errorf("op=%s: StatusSyncedAt 应非 nil", tt.operation)
			}
		})
	}
}

// ── OperationTransitStatus 映射完整性测试 ──

func TestOperationTransitStatus_CoversAllOperations(t *testing.T) {
	allOps := []string{
		model.OpCreate,
		model.OpReboot,
		model.OpReinstall,
		model.OpUpgrade,
		model.OpDelete,
		model.OpMigrate,
	}
	for _, op := range allOps {
		if _, ok := model.OperationTransitStatus[op]; !ok {
			t.Errorf("OperationTransitStatus 缺少操作 %q 的映射", op)
		}
	}
}

// ── queryAdminStatsFromCache 测试 ──

func TestQueryAdminStatsFromCache(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	// 创建不同状态的实例
	instances := []model.Instance{
		{Name: "r1", InstanceId: "ins-r1", LastKnownStatus: model.StatusRunning},
		{Name: "r2", InstanceId: "ins-r2", LastKnownStatus: model.StatusRunning},
		{Name: "s1", InstanceId: "ins-s1", LastKnownStatus: model.StatusStopped},
		{Name: "c1", InstanceId: "ins-c1", LastKnownStatus: model.StatusCreating},
		{Name: "f1", InstanceId: "ins-f1", LastKnownStatus: model.StatusLoadFailed},
	}
	for i := range instances {
		db.Create(&instances[i])
	}

	stats := queryAdminStatsFromCache(ctx, adminQueryFilter{})

	if stats.Total != 5 {
		t.Errorf("Total=%d, want 5", stats.Total)
	}
	if stats.Running != 2 {
		t.Errorf("Running=%d, want 2", stats.Running)
	}
	if stats.Stopped != 1 {
		t.Errorf("Stopped=%d, want 1", stats.Stopped)
	}
	if stats.Other != 2 {
		t.Errorf("Other=%d, want 2", stats.Other)
	}
	if stats.OtherDetail.InProgress.Count != 1 {
		t.Errorf("InProgress=%d, want 1", stats.OtherDetail.InProgress.Count)
	}
	if stats.OtherDetail.NeedAttention.Count != 1 {
		t.Errorf("NeedAttention=%d, want 1", stats.OtherDetail.NeedAttention.Count)
	}
}

// ── lightToInstance 身份字段传递测试（P1-1）──
// 防回归：Name/UserID/GroupID 必须被复制，否则 reconcile 的状态计算
//（isCLSPendingInstallation 依赖 GroupID）与 side-effect 通知（依赖 UserID/Name）会出错。

func TestLightToInstance_CopiesIdentityFields(t *testing.T) {
	at := time.Now()
	it := lightInstance{
		ID:                        7,
		Name:                      "demo",
		UserID:                    42,
		GroupID:                   99,
		InstanceId:                "ins-x",
		CurrentOperation:          model.OpReboot,
		CurrentOperationState:     model.OpStateProcessing,
		CurrentOperationUpdatedAt: &at,
		LastCVMState:              "RUNNING",
		LastStableState:           "running",
		AgentReady:                1,
		CLSAgentStatus:            model.CLSAgentInstalling,
		CLSAgentStatusAt:          &at,
	}

	inst := lightToInstance(it)

	if inst.ID != 7 {
		t.Errorf("ID=%d, want 7", inst.ID)
	}
	if inst.Name != "demo" {
		t.Errorf("Name=%q, want demo", inst.Name)
	}
	if inst.UserID != 42 {
		t.Errorf("UserID=%d, want 42", inst.UserID)
	}
	if inst.GroupID != 99 {
		t.Errorf("GroupID=%d, want 99", inst.GroupID)
	}
	if inst.InstanceId != "ins-x" {
		t.Errorf("InstanceId=%q, want ins-x", inst.InstanceId)
	}
	if inst.CurrentOperation != model.OpReboot {
		t.Errorf("CurrentOperation=%q, want reboot", inst.CurrentOperation)
	}
	if inst.CLSAgentStatus != model.CLSAgentInstalling {
		t.Errorf("CLSAgentStatus=%d, want %d", inst.CLSAgentStatus, model.CLSAgentInstalling)
	}
}

// ── queryAllLightInstancesWithFilter 身份字段查询测试（P1-1）──
// 防回归：Select 列表必须包含 name/user_id/group_id。

func TestQueryAllLightInstancesWithFilter_SelectsIdentityFields(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	inst := &model.Instance{Name: "n1", InstanceId: "ins-n1", UserID: 11, GroupID: 22}
	db.Create(inst)

	items, err := queryAllLightInstancesWithFilter(ctx, adminQueryFilter{})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("应查到 1 条, 实际=%d", len(items))
	}
	got := items[0]
	if got.Name != "n1" {
		t.Errorf("Name=%q, want n1", got.Name)
	}
	if got.UserID != 11 {
		t.Errorf("UserID=%d, want 11", got.UserID)
	}
	if got.GroupID != 22 {
		t.Errorf("GroupID=%d, want 22", got.GroupID)
	}
}

// ── reconcileTagsToJSON 测试（P1-2）──
// 防回归：API_ERROR / 序列化失败时必须保留旧缓存，绝不用空标签覆盖真实数据。

func TestReconcileTagsToJSON(t *testing.T) {
	const oldJSON = `[{"key":"env","value":"prod"}]`

	tests := []struct {
		name string
		info *CVMInstanceInfo
		old  string
		want string
	}{
		{
			name: "API_ERROR 保留旧缓存",
			info: &CVMInstanceInfo{State: "API_ERROR"},
			old:  oldJSON,
			want: oldJSON,
		},
		{
			name: "API_ERROR 旧缓存为空也保留空",
			info: &CVMInstanceInfo{State: "API_ERROR"},
			old:  "",
			want: "",
		},
		{
			name: "info 为 nil（CVM 消失）→ 空数组",
			info: nil,
			old:  oldJSON,
			want: "[]",
		},
		{
			name: "无标签 → 空数组",
			info: &CVMInstanceInfo{State: "RUNNING"},
			old:  oldJSON,
			want: "[]",
		},
		{
			name: "有标签 → 序列化",
			info: &CVMInstanceInfo{State: "RUNNING", Tags: []CVMTag{{Key: "env", Value: "prod"}}},
			old:  "",
			want: `[{"key":"env","value":"prod"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reconcileTagsToJSON(tt.info, tt.old)
			if got != tt.want {
				t.Errorf("reconcileTagsToJSON=%q, want %q", got, tt.want)
			}
		})
	}
}

// ── ImgId 缓存字段 DB 读写测试 ──

func TestBatchUpdateInstanceStatusCache_DoesNotWriteImageId(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	// 预设 ImgId，验证 BatchUpdate 不会覆盖它
	inst := &model.Instance{Name: "img-test", InstanceId: "ins-img", ImgId: "img-original"}
	db.Create(inst)

	roundStart := time.Now()
	items := []model.InstanceStatusCacheItem{
		{ID: inst.ID, Status: model.StatusRunning},
	}
	model.BatchUpdateInstanceStatusCache(ctx, items, roundStart)

	var loaded model.Instance
	db.First(&loaded, inst.ID)

	if loaded.ImgId != "img-original" {
		t.Errorf("ImgId=%q, want img-original (should not be overwritten by reconcile)", loaded.ImgId)
	}
}

// ── queryAllLightInstancesWithFilter 查 ImgId 测试 ──

func TestQueryAllLightInstancesWithFilter_SelectsImgId(t *testing.T) {
	db := initReconcileTestDB(t)
	ctx := context.Background()

	inst := &model.Instance{Name: "n1", InstanceId: "ins-n1", ImgId: "img-xyz"}
	db.Create(inst)

	items, err := queryAllLightInstancesWithFilter(ctx, adminQueryFilter{})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("应查到 1 条, 实际=%d", len(items))
	}
	if items[0].ImgId != "img-xyz" {
		t.Errorf("ImgId=%q, want img-xyz", items[0].ImgId)
	}
}

// ── BatchUpdateInstanceStatusCache 空列表边界测试 ──

func TestBatchUpdateInstanceStatusCache_EmptyItems(t *testing.T) {
	ctx := context.Background()
	// 空列表不应 panic 或写 DB
	model.BatchUpdateInstanceStatusCache(ctx, nil, time.Now())
}

// ── reconcileLocalInstances 单测 ──

// initReconcileTestDBWithLocalInfo 在标准 reconcile 测试 DB 基础上额外迁移 LocalInstanceInfo 表。
func initReconcileTestDBWithLocalInfo(t *testing.T) *gorm.DB {
	db := initReconcileTestDB(t)
	if err := db.AutoMigrate(&model.LocalInstanceInfo{}); err != nil {
		t.Fatalf("迁移 LocalInstanceInfo 失败: %v", err)
	}
	return db
}

// TestReconcileLocalInstances_StopsWhenStale 验证正常 sweep 路径：
// 有 info 行，且 last_report_at 已超过阈值 → 应被改为 stopped。
func TestReconcileLocalInstances_StopsWhenStale(t *testing.T) {
	db := initReconcileTestDBWithLocalInfo(t)
	ctx := context.Background()

	inst := &model.Instance{
		Name:            "local-stale",
		InstanceId:      "local-stale-001",
		Source:          model.InstanceSourceLocal,
		LastKnownStatus: model.StatusRunning,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	stale := time.Now().Add(-2 * model.LocalInstanceOfflineThreshold)
	info := &model.LocalInstanceInfo{
		InstanceID:   inst.ID,
		HostName:     "alex-mbp",
		LastReportAt: &stale,
	}
	if err := db.Create(info).Error; err != nil {
		t.Fatalf("create info: %v", err)
	}

	reconcileLocalInstances(ctx)

	var loaded model.Instance
	if err := db.First(&loaded, inst.ID).Error; err != nil {
		t.Fatalf("reload instance: %v", err)
	}
	if loaded.LastKnownStatus != model.StatusStopped {
		t.Errorf("心跳超时实例应被改为 stopped，实际=%q", loaded.LastKnownStatus)
	}
}

// TestReconcileLocalInstances_KeepsRunningWhenFresh 验证心跳新鲜的实例不会被误伤。
func TestReconcileLocalInstances_KeepsRunningWhenFresh(t *testing.T) {
	db := initReconcileTestDBWithLocalInfo(t)
	ctx := context.Background()

	inst := &model.Instance{
		Name:            "local-fresh",
		InstanceId:      "local-fresh-001",
		Source:          model.InstanceSourceLocal,
		LastKnownStatus: model.StatusRunning,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	now := time.Now()
	info := &model.LocalInstanceInfo{
		InstanceID:   inst.ID,
		HostName:     "alex-mbp",
		LastReportAt: &now,
	}
	if err := db.Create(info).Error; err != nil {
		t.Fatalf("create info: %v", err)
	}

	reconcileLocalInstances(ctx)

	var loaded model.Instance
	if err := db.First(&loaded, inst.ID).Error; err != nil {
		t.Fatalf("reload instance: %v", err)
	}
	if loaded.LastKnownStatus != model.StatusRunning {
		t.Errorf("心跳新鲜的实例不应被 sweep 改写，期望 running，实际=%q", loaded.LastKnownStatus)
	}
}

// TestReconcileLocalInstances_RevivesStoppedWhenHeartbeatFresh 双向对账核心场景：
// 实例现为 stopped 但 last_report_at 已在阀值内（例如历史脏数据、
// 或 Pluck/UPDATE 中间的竞态误伤），应自愈刷回 running。
func TestReconcileLocalInstances_RevivesStoppedWhenHeartbeatFresh(t *testing.T) {
	db := initReconcileTestDBWithLocalInfo(t)
	ctx := context.Background()

	inst := &model.Instance{
		Name:            "local-revive",
		InstanceId:      "local-revive-001",
		Source:          model.InstanceSourceLocal,
		LastKnownStatus: model.StatusStopped, // 先前被误刷/历史脏数据
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	// 心跳新鲜：恰好在阀值内
	fresh := time.Now().Add(-30 * time.Second)
	info := &model.LocalInstanceInfo{
		InstanceID:   inst.ID,
		HostName:     "alex-mbp",
		LastReportAt: &fresh,
	}
	if err := db.Create(info).Error; err != nil {
		t.Fatalf("create info: %v", err)
	}

	reconcileLocalInstances(ctx)

	var loaded model.Instance
	if err := db.First(&loaded, inst.ID).Error; err != nil {
		t.Fatalf("reload instance: %v", err)
	}
	if loaded.LastKnownStatus != model.StatusRunning {
		t.Errorf("stopped 但心跳新鲜应被刷回 running，实际=%q", loaded.LastKnownStatus)
	}
	if loaded.StatusSyncedAt == nil {
		t.Errorf("revive 后 status_synced_at 应被写入")
	}
}

// ── ReconcileInstanceStatuses 全量入口端到端测试 ──
//
// 这个函数是 cvm-status-reconcile 后台任务的入口，原来 0% 覆盖。
// 它会调 NewCVMClient 打 CVM API，但当所有实例都是本地实例时，
// batchFetchCVMInfoMap 会先用 strings.HasPrefix(id, "local-") 过滤掉，
// cvmIds 为空 → 不会调 CVM API → 整个流程可在测试里跑完。
//
// 这里只验证「无崩溃」+「本地实例 source guard 生效（不会落到 ResolveInstanceStatus）」+
// 「setFullSyncFinished 把 site_config 时间戳写进去」。

// TestReconcileInstanceStatuses_NoInstances 实例库为空：跳过、写 site_config。
func TestReconcileInstanceStatuses_NoInstances(t *testing.T) {
	db := initReconcileTestDBWithLocalInfo(t)
	ctx := context.Background()

	// SetLastFullSyncFinishedAt 走 UpdateSiteConfig，表空时会报 WHERE
	// conditions required。预 seed 一行。
	if err := db.Create(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("seed SiteConfig: %v", err)
	}

	ReconcileInstanceStatuses(ctx)
	after := model.GetLastFullSyncFinishedAt(ctx)
	if after == nil {
		t.Errorf("无实例时也应推进 LastFullSyncFinishedAt（无崩溃 + 完成态写入），返回=nil")
	}
}

// TestReconcileInstanceStatuses_OnlyLocalInstances 库里只有本地实例：
// 走 batchFetchCVMInfoMap 分支但 cvmIds 全部被过滤，不调 CVM API；
// 主循环里 source==local 直接 continue，不调 ResolveInstanceStatus。
func TestReconcileInstanceStatuses_OnlyLocalInstances(t *testing.T) {
	db := initReconcileTestDBWithLocalInfo(t)
	ctx := context.Background()

	if err := db.Create(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("seed SiteConfig: %v", err)
	}

	// 一个 local 实例 + 一条新鲜的 LocalInstanceInfo
	inst := &model.Instance{
		Name:            "local-only-1",
		InstanceId:      "local-only-001",
		Source:          model.InstanceSourceLocal,
		LastKnownStatus: model.StatusRunning,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	fresh := time.Now().Add(-30 * time.Second)
	info := &model.LocalInstanceInfo{
		InstanceID:   inst.ID,
		HostName:     "alex-mbp",
		LastReportAt: &fresh,
	}
	if err := db.Create(info).Error; err != nil {
		t.Fatalf("create local info: %v", err)
	}

	ReconcileInstanceStatuses(ctx)
	after := model.GetLastFullSyncFinishedAt(ctx)
	if after == nil {
		t.Errorf("纯本地实例池也应推进同步时间，返回=nil")
	}

	// 验证本地实例 last_known_status 没被 reconcile 改写（fresh 应保持 running）
	var loaded model.Instance
	if err := db.First(&loaded, inst.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if loaded.LastKnownStatus != model.StatusRunning {
		t.Errorf("本地实例 fresh，last_known_status 应仍为 running，实际=%q", loaded.LastKnownStatus)
	}
}

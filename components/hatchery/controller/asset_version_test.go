package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupAssetVersionTestDB 初始化内存 SQLite，迁移版本记录相关表。
func setupAssetVersionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 内存 SQLite 按连接隔离；共享缓存让事务外的下发查询也能读取同一套迁移数据。
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AssetVersionRecord{},
		&model.Project{},
		&model.UserGroup{},
		&model.GroupClosure{},
		&model.Skill{},
		&model.EnterpriseRule{},
		&model.SkillVisibilityGroup{},
		&model.RuleVisibilityGroup{},
		&model.LocalAgentScopeBinding{},
		&model.ProjectConfigBinding{},
		&model.GroupConfigBinding{},
		&model.Instance{},
		&model.User{},
	); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	restore := model.UseDBForTest(db)
	t.Cleanup(restore)
	oldSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "test-tenant", Domain: "example.com"}
	t.Cleanup(func() { common.FixedSnapshot = oldSnap })
	return db
}

// setupAssetVersionStore 初始化测试 session store（requireAdmin 依赖 session）。
func setupAssetVersionStore(t *testing.T) func() {
	t.Helper()
	orig := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long-pad"))
	return func() { Store = orig }
}

// installCall 记录一次下发调用。
type installCall struct {
	targetType  string
	targetID    uint
	instanceIDs []uint
	items       []model.AssetChangeItem
}

// mockDispatchInstall 替换 dispatchInstallFn，记录调用。
func mockDispatchInstall(t *testing.T) func() {
	t.Helper()
	orig := dispatchInstallFn
	dispatchInstallFn = func(ctx context.Context, req dispatchRequest) error {
		return nil
	}
	return func() { dispatchInstallFn = orig }
}

// withInstallCapture 替换 dispatchInstallFn 为捕获调用的实现，返回读取调用列表的函数。
func withInstallCapture(t *testing.T, sink *[]installCall) func() {
	t.Helper()
	var mu sync.Mutex
	orig := dispatchInstallFn
	dispatchInstallFn = func(ctx context.Context, req dispatchRequest) error {
		mu.Lock()
		*sink = append(*sink, installCall{targetID: req.targetID, instanceIDs: req.instanceIDs, items: req.items})
		mu.Unlock()
		return nil
	}
	return func() { dispatchInstallFn = orig }
}

// setupTargetWithInstance 创建目标(project/group) + 一个本地 agent 实例绑定，返回目标信息。
func setupTargetWithInstance(t *testing.T, db *gorm.DB, targetType string, syncMode string) (uint, uint) {
	t.Helper()
	var targetID uint
	var scope string
	if targetType == model.TargetTypeProject {
		p := model.Project{Name: "test-project", SyncMode: syncMode}
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("创建 project 失败: %v", err)
		}
		targetID = p.ID
		scope = model.LocalAgentScopeWorkspace
	} else {
		g := model.UserGroup{Name: "test-group", SyncMode: syncMode}
		if err := db.Create(&g).Error; err != nil {
			t.Fatalf("创建 group 失败: %v", err)
		}
		targetID = g.ID
		scope = model.LocalAgentScopeUser
	}
	inst := model.Instance{Identifier: "test-tenant", Name: "agent-1", Source: model.InstanceSourceLocal}
	if err := db.Create(&inst).Error; err != nil {
		t.Fatalf("创建 instance 失败: %v", err)
	}
	binding := model.LocalAgentScopeBinding{
		Identifier: "test-tenant",
		InstanceID: inst.ID,
		Scope:      scope,
		ScopeKey:   "k",
	}
	if targetType == model.TargetTypeGroup {
		binding.GroupID = targetID
	} else {
		binding.ProjectID = targetID
	}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatalf("创建 binding 失败: %v", err)
	}
	return targetID, inst.ID
}

func countRecords(t *testing.T, db *gorm.DB, targetType string, targetID uint) int64 {
	t.Helper()
	var n int64
	db.Model(&model.AssetVersionRecord{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Count(&n)
	return n
}

func assertErr() error { return &testErr{} }

type testErr struct{}

func (e *testErr) Error() string { return "rollback" }

// ---- 测试 ----

func TestRecordAssetSave_FirstSaveIsV1(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	targetID, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	defer mockDispatchInstall(t)()

	in := SaveInput{
		TargetType:   model.TargetTypeProject,
		TargetID:     targetID,
		SyncMode:     model.SyncModeContinuous,
		OldSyncMode:  model.SyncModeContinuous,
		Assets:       []AssetBindingItem{{AssetType: model.AssetTypeSkill, Slug: "api-doc"}},
		OldAssets:    []AssetBindingItem{},
		OperatorID:   8812,
		OperatorName: "zhangsan",
	}
	if err := RecordAssetSave(context.Background(), db, in); err != nil {
		t.Fatalf("RecordAssetSave 失败: %v", err)
	}
	if n := countRecords(t, db, model.TargetTypeProject, targetID); n != 1 {
		t.Fatalf("期望 1 条记录, 实际 %d", n)
	}
	var rec model.AssetVersionRecord
	db.Where("target_type = ? AND target_id = ?", model.TargetTypeProject, targetID).First(&rec)
	if rec.Version != 1 {
		t.Fatalf("首建期望 version=1, 实际 %d", rec.Version)
	}
	if rec.TriggerType != model.TriggerTypeManual || rec.TriggerReason != model.TriggerReasonManualSave {
		t.Fatalf("trigger 期望 manual/manual_save, 实际 %s/%s", rec.TriggerType, rec.TriggerReason)
	}
	if rec.OperatorType != "admin" {
		t.Fatalf("operator_type 期望 admin, 实际 %s", rec.OperatorType)
	}
	if rec.OperatorID != 8812 || rec.OperatorName != "zhangsan" {
		t.Fatalf("operator_id/name 期望 8812/zhangsan, 实际 %d/%s", rec.OperatorID, rec.OperatorName)
	}
	var ch model.AssetChanges
	if err := json.Unmarshal([]byte(rec.ChangesJSON), &ch); err != nil {
		t.Fatalf("解析 changes 失败: %v", err)
	}
	if len(ch.Added) != 1 || ch.Added[0].Slug != "api-doc" {
		t.Fatalf("Added 期望 1 项 api-doc, 实际 %+v", ch.Added)
	}
}

func TestRecordAssetSave_ModifyVersion2(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	targetID, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	defer mockDispatchInstall(t)()

	if err := RecordAssetSave(context.Background(), db, SaveInput{
		TargetType: model.TargetTypeProject, TargetID: targetID, SyncMode: model.SyncModeContinuous,
		OldSyncMode: model.SyncModeContinuous, Assets: []AssetBindingItem{{AssetType: model.AssetTypeSkill, Slug: "a"}}, OldAssets: []AssetBindingItem{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := RecordAssetSave(context.Background(), db, SaveInput{
		TargetType: model.TargetTypeProject, TargetID: targetID, SyncMode: model.SyncModeContinuous,
		OldSyncMode: model.SyncModeContinuous,
		Assets:      []AssetBindingItem{{AssetType: model.AssetTypeSkill, Slug: "a"}, {AssetType: model.AssetTypeSkill, Slug: "b"}},
		OldAssets:   []AssetBindingItem{{AssetType: model.AssetTypeSkill, Slug: "a"}, {AssetType: model.AssetTypeSkill, Slug: "c"}},
	}); err != nil {
		t.Fatal(err)
	}
	if n := countRecords(t, db, model.TargetTypeProject, targetID); n != 2 {
		t.Fatalf("期望 2 条记录, 实际 %d", n)
	}
	var rec model.AssetVersionRecord
	db.Where("target_type = ? AND target_id = ? AND version = ?", model.TargetTypeProject, targetID, 2).First(&rec)
	var ch model.AssetChanges
	json.Unmarshal([]byte(rec.ChangesJSON), &ch)
	if len(ch.Added) != 1 || ch.Added[0].Slug != "b" {
		t.Fatalf("Added 期望 b, 实际 %+v", ch.Added)
	}
	if len(ch.Removed) != 1 || ch.Removed[0].Slug != "c" {
		t.Fatalf("Removed 期望 c, 实际 %+v", ch.Removed)
	}
}

func TestRecordAssetSave_SyncModeJumpTriggersInstall(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	targetID, instID := setupTargetWithInstance(t, db, model.TargetTypeGroup, model.SyncModeInitialOnly)
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	if err := RecordAssetSave(context.Background(), db, SaveInput{
		TargetType: model.TargetTypeGroup, TargetID: targetID, SyncMode: model.SyncModeContinuous,
		OldSyncMode: model.SyncModeInitialOnly,
		Assets:      []AssetBindingItem{{AssetType: model.AssetTypeSkill, Slug: "x"}}, OldAssets: []AssetBindingItem{},
	}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("跳变期望触发 1 次下发, 实际 %d", len(calls))
	}
	if calls[0].instanceIDs[0] != instID {
		t.Fatalf("下发实例期望 %d, 实际 %v", instID, calls[0].instanceIDs)
	}
}

func TestRecordAssetSave_InitialOnlyNoInstall(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	targetID, _ := setupTargetWithInstance(t, db, model.TargetTypeGroup, model.SyncModeInitialOnly)
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	if err := RecordAssetSave(context.Background(), db, SaveInput{
		TargetType: model.TargetTypeGroup, TargetID: targetID, SyncMode: model.SyncModeInitialOnly,
		OldSyncMode: model.SyncModeInitialOnly,
		Assets:      []AssetBindingItem{{AssetType: model.AssetTypeSkill, Slug: "x"}}, OldAssets: []AssetBindingItem{},
	}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("initial_only 期望不下发, 实际触发 %d 次", len(calls))
	}
}

func TestPublishAssetVersion_PublishedContinuousInstalls(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	targetID, instID := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	if err := db.Transaction(func(tx *gorm.DB) error {
		_, e := PublishAssetVersion(context.Background(), tx, PublishInput{
			AssetType:       model.AssetTypeSkill,
			Slug:            "code-review",
			FromVersion:     "v2.0.0",
			ToVersion:       "v2.1.0",
			TriggerReason:   model.TriggerReasonAssetPublished,
			AffectedTargets: []AssetTarget{{TargetType: model.TargetTypeProject, TargetID: targetID, SyncMode: model.SyncModeContinuous}},
		})
		return e
	}); err != nil {
		t.Fatalf("PublishAssetVersion 失败: %v", err)
	}
	if n := countRecords(t, db, model.TargetTypeProject, targetID); n != 1 {
		t.Fatalf("期望 1 条 system 记录, 实际 %d", n)
	}
	var rec model.AssetVersionRecord
	db.Where("target_type = ? AND target_id = ?", model.TargetTypeProject, targetID).First(&rec)
	if rec.TriggerType != model.TriggerTypeSystem || rec.TriggerReason != model.TriggerReasonAssetPublished {
		t.Fatalf("trigger 期望 system/asset_version_published, 实际 %s/%s", rec.TriggerType, rec.TriggerReason)
	}
	if len(calls) != 1 {
		t.Fatalf("continuous+updated 期望触发下发, 实际 %d", len(calls))
	}
	if calls[0].instanceIDs[0] != instID {
		t.Fatalf("下发实例期望 %d, 实际 %v", instID, calls[0].instanceIDs)
	}
}

// TestPublishAssetVersion_RulePublishedContinuousInstalls 守护：rule 与 skill 共用同一套
// PublishAssetVersion / maybeInstall 下发链路，需对 rule 类型单独守护 continuous+updated 触发下发。
func TestPublishAssetVersion_RulePublishedContinuousInstalls(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	targetID, instID := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	if err := db.Transaction(func(tx *gorm.DB) error {
		_, e := PublishAssetVersion(context.Background(), tx, PublishInput{
			AssetType:       model.AssetTypeRule,
			Slug:            "secure-coding",
			FromVersion:     "v1.0.0",
			ToVersion:       "v1.1.0",
			TriggerReason:   model.TriggerReasonAssetPublished,
			AffectedTargets: []AssetTarget{{TargetType: model.TargetTypeProject, TargetID: targetID, SyncMode: model.SyncModeContinuous}},
		})
		return e
	}); err != nil {
		t.Fatalf("PublishAssetVersion(rule) 失败: %v", err)
	}
	if n := countRecords(t, db, model.TargetTypeProject, targetID); n != 1 {
		t.Fatalf("期望 1 条 system 记录, 实际 %d", n)
	}
	var rec model.AssetVersionRecord
	db.Where("target_type = ? AND target_id = ?", model.TargetTypeProject, targetID).First(&rec)
	if rec.TriggerType != model.TriggerTypeSystem || rec.TriggerReason != model.TriggerReasonAssetPublished {
		t.Fatalf("trigger 期望 system/asset_version_published, 实际 %s/%s", rec.TriggerType, rec.TriggerReason)
	}
	if len(calls) != 1 {
		t.Fatalf("rule continuous+updated 期望触发下发, 实际 %d", len(calls))
	}
	if calls[0].instanceIDs[0] != instID {
		t.Fatalf("下发实例期望 %d, 实际 %v", instID, calls[0].instanceIDs)
	}
}

// TestPublishAssetVersionForChange_RuleBoundTargetRecords 守护：rule 绑定到存量 project 时，
// publishAssetVersionForChange 应记一条记录并触发下发（与 skill 版对称）。
func TestPublishAssetVersionForChange_RuleBoundTargetRecords(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	targetID, instID := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	if err := model.ReplaceProjectConfigBindings(db, targetID, model.AssetBindingTypeRule, []string{"bound-rule"}); err != nil {
		t.Fatal(err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		publishAssetVersionForChange(context.Background(), tx, model.AssetTypeRule, "bound-rule", "v1.0.0", "v2.0.0", model.TriggerReasonAssetPublished)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if n := countRecords(t, db, model.TargetTypeProject, targetID); n != 1 {
		t.Fatalf("期望 1 条记录, 实际 %d", n)
	}
	if len(calls) != 1 || calls[0].instanceIDs[0] != instID {
		t.Fatalf("期望对存量目标触发下发, 实际 calls=%d", len(calls))
	}
}

func TestPublishAssetVersion_DeletedNoInstall(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	targetID, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	if err := db.Transaction(func(tx *gorm.DB) error {
		_, e := PublishAssetVersion(context.Background(), tx, PublishInput{
			AssetType:       model.AssetTypeSkill,
			Slug:            "log-analyzer",
			TriggerReason:   model.TriggerReasonAssetDeleted,
			AffectedTargets: []AssetTarget{{TargetType: model.TargetTypeProject, TargetID: targetID, SyncMode: model.SyncModeContinuous}},
		})
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("删除(仅 removed)期望不下发, 实际 %d", len(calls))
	}
	var rec model.AssetVersionRecord
	db.Where("target_type = ? AND target_id = ?", model.TargetTypeProject, targetID).First(&rec)
	if rec.TriggerReason != model.TriggerReasonAssetDeleted {
		t.Fatalf("期望 asset_deleted, 实际 %s", rec.TriggerReason)
	}
}

func TestPublishAssetVersion_InitialOnlyNoInstall(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	targetID, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeInitialOnly)
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	if err := db.Transaction(func(tx *gorm.DB) error {
		_, e := PublishAssetVersion(context.Background(), tx, PublishInput{
			AssetType:       model.AssetTypeSkill,
			Slug:            "code-review",
			ToVersion:       "v2.1.0",
			TriggerReason:   model.TriggerReasonAssetPublished,
			AffectedTargets: []AssetTarget{{TargetType: model.TargetTypeProject, TargetID: targetID, SyncMode: model.SyncModeInitialOnly}},
		})
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("initial_only 期望不下发, 实际 %d", len(calls))
	}
}

func TestRecordAssetSave_TxRollbackNoRecord(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	targetID, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	defer mockDispatchInstall(t)()

	err := db.Transaction(func(tx *gorm.DB) error {
		if e := RecordAssetSave(context.Background(), tx, SaveInput{
			TargetType: model.TargetTypeProject, TargetID: targetID, SyncMode: model.SyncModeContinuous,
			OldSyncMode: model.SyncModeContinuous,
			Assets:      []AssetBindingItem{{AssetType: model.AssetTypeSkill, Slug: "a"}}, OldAssets: []AssetBindingItem{},
		}); e != nil {
			return e
		}
		return assertErr()
	})
	if err == nil {
		t.Fatal("期望事务回滚报错")
	}
	if n := countRecords(t, db, model.TargetTypeProject, targetID); n != 0 {
		t.Fatalf("回滚后期望 0 条记录, 实际 %d", n)
	}
}

func TestHandleAdminAssetVersions_PaginationAndFilter(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	defer mockDispatchInstall(t)()
	makeAdminUser(t, context.Background(), "admin")
	targetID, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	for i := 0; i < 5; i++ {
		if err := RecordAssetSave(context.Background(), db, SaveInput{
			TargetType: model.TargetTypeProject, TargetID: targetID, SyncMode: model.SyncModeContinuous,
			OldSyncMode:  model.SyncModeContinuous,
			Assets:       []AssetBindingItem{{AssetType: model.AssetTypeSkill, Slug: "s"}},
			OldAssets:    []AssetBindingItem{},
			OperatorID:   8812,
			OperatorName: "zhangsan",
		}); err != nil {
			t.Fatal(err)
		}
	}

	rr := httptest.NewRecorder()
	req := adminSessionReq(t, http.MethodGet, "/admin/assets/versions?target_type=project&target_id="+strconv.FormatUint(uint64(targetID), 10)+"&page=1&page_size=2", nil, "admin")
	HandleAdminAssetVersions(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Total    int64 `json:"total"`
		Page     int   `json:"page"`
		PageSize int   `json:"page_size"`
		Data     []struct {
			RecordID    uint   `json:"record_id"`
			Version     int    `json:"version"`
			TriggerType string `json:"trigger_type"`
			Operator    struct {
				Type string `json:"type"`
				ID   uint   `json:"id"`
				Name string `json:"name"`
			} `json:"operator"`
			Segments []struct {
				Type  string `json:"type"`
				Items []struct {
					AssetType string `json:"asset_type"`
					Name      string `json:"name"`
				} `json:"items"`
			} `json:"segments"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if resp.Total != 5 {
		t.Fatalf("total 期望 5, 实际 %d", resp.Total)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("page_size=2 期望 2 条, 实际 %d", len(resp.Data))
	}
	if resp.Data[0].Version != 5 {
		t.Fatalf("倒序期望首条 version=5, 实际 %d", resp.Data[0].Version)
	}
	if resp.Data[0].Operator.Type != "admin" {
		t.Fatalf("operator.type 期望 admin, 实际 %s", resp.Data[0].Operator.Type)
	}
	if resp.Data[0].Operator.ID == 0 || resp.Data[0].Operator.Name == "" {
		t.Fatalf("operator.id/name 期望非空, 实际 %d/%s", resp.Data[0].Operator.ID, resp.Data[0].Operator.Name)
	}
	if len(resp.Data[0].Segments) == 0 {
		t.Fatalf("期望有 segments")
	}
	foundAdded := false
	for _, seg := range resp.Data[0].Segments {
		if seg.Type == "added" && len(seg.Items) > 0 && seg.Items[0].AssetType == "skill" {
			foundAdded = true
		}
	}
	if !foundAdded {
		t.Fatalf("期望 added 段含 skill 项")
	}
}

func TestHandleAdminAssetVersions_BadParam(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	makeAdminUser(t, context.Background(), "admin")
	_ = db
	rr := httptest.NewRecorder()
	req := adminSessionReq(t, http.MethodGet, "/admin/assets/versions?target_type=foo&target_id=1", nil, "admin")
	HandleAdminAssetVersions(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("非法 target_type 期望 400, 实际 %d", rr.Code)
	}
}

// TestPublishAssetVersionForChange_NoBindingNoRecord 守护产品语义：刚上传、尚未关联到任何
// 项目/分组的 slug（无绑定目标），不触发版本记录。对应「首次创建不触发」的不变量。
func TestPublishAssetVersionForChange_NoBindingNoRecord(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	// slug=fresh-skill 尚未绑定到任何项目/分组
	if err := db.Transaction(func(tx *gorm.DB) error {
		publishAssetVersionForChange(context.Background(), tx, model.AssetTypeSkill, "fresh-skill", "", "v1.0.0", model.TriggerReasonAssetPublished)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if n := countRecords(t, db, model.TargetTypeProject, 0); n != 0 {
		t.Fatalf("无绑定目标期望 0 条记录, 实际 %d", n)
	}
	if len(calls) != 0 {
		t.Fatalf("无绑定目标期望不下发, 实际 %d 次", len(calls))
	}
}

// TestPublishAssetVersionForChange_BoundTargetRecords 守护：slug 已绑定到存量项目/分组时，
// 发新版会对其落版本记录并按 sync_mode 触发下发（只影响存量目标）。
func TestPublishAssetVersionForChange_BoundTargetRecords(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	targetID, instID := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	// 把 slug=bound-skill 绑定到该 project（模拟存量关联）
	if err := model.ReplaceProjectConfigBindings(db, targetID, model.AssetBindingTypeSkill, []string{"bound-skill"}); err != nil {
		t.Fatal(err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		publishAssetVersionForChange(context.Background(), tx, model.AssetTypeSkill, "bound-skill", "v1.0.0", "v2.0.0", model.TriggerReasonAssetPublished)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if n := countRecords(t, db, model.TargetTypeProject, targetID); n != 1 {
		t.Fatalf("期望 1 条记录, 实际 %d", n)
	}
	if len(calls) != 1 || calls[0].instanceIDs[0] != instID {
		t.Fatalf("期望对存量目标触发下发, 实际 calls=%d", len(calls))
	}
}

// TestPublishAssetVersionForChange_GroupInitialOnlyNoInstall 守护：slug 绑定到 initial_only 的
// 存量分组时，发新版落版本记录但仅记录不下发（同步模式判定）。同时覆盖 UserGroup 分支。
func TestPublishAssetVersionForChange_GroupInitialOnlyNoInstall(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	targetID, _ := setupTargetWithInstance(t, db, model.TargetTypeGroup, model.SyncModeInitialOnly)
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	if err := db.Create(&model.GroupConfigBinding{GroupID: targetID, ConfigType: model.AssetBindingTypeSkill, ConfigKey: "bound-skill"}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		publishAssetVersionForChange(context.Background(), tx, model.AssetTypeSkill, "bound-skill", "v1.0.0", "v2.0.0", model.TriggerReasonAssetPublished)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if n := countRecords(t, db, model.TargetTypeGroup, targetID); n != 1 {
		t.Fatalf("期望 1 条记录, 实际 %d", n)
	}
	if len(calls) != 0 {
		t.Fatalf("initial_only 期望不下发, 实际 %d 次", len(calls))
	}
}

// TestDiffRemovedScope 守护应用范围 diff 逻辑：仅“旧有新无”的目标算被移出；
// visType=all 视为范围扩大不算移出。
func TestDiffRemovedScope(t *testing.T) {
	// group: 旧[1,2,3] 新[1,3] → 移出 2；project: 旧[10] 新[] → 移出 10
	rg, rp := diffRemovedScope(model.VisibilityGroup, []uint{1, 2, 3}, []uint{10}, []uint{1, 3}, nil)
	if len(rg) != 1 || rg[0] != 2 {
		t.Fatalf("group 移出期望 [2], 实际 %v", rg)
	}
	if len(rp) != 1 || rp[0] != 10 {
		t.Fatalf("project 移出期望 [10], 实际 %v", rp)
	}
	// 纯新增：旧[1] 新[1,2] → 无移出
	rg2, rp2 := diffRemovedScope(model.VisibilityGroup, []uint{1}, nil, []uint{1, 2}, nil)
	if len(rg2) != 0 || len(rp2) != 0 {
		t.Fatalf("纯新增期望无移出, 实际 rg=%v rp=%v", rg2, rp2)
	}
	// visType=all：旧[1,2] → 不算移出
	rg3, rp3 := diffRemovedScope(model.VisibilityAll, []uint{1, 2}, []uint{5}, nil, nil)
	if len(rg3) != 0 || len(rp3) != 0 {
		t.Fatalf("all 期望不算移出, 实际 rg=%v rp=%v", rg3, rp3)
	}
}

// TestPublishScopeRemoval_RecordsAndUnbindsDirectAsset 守护产品语义：从应用范围移出
// 存量分组时，只有直接绑定过的资产才解绑、记版本且不下发。
func TestPublishScopeRemoval_RecordsAndUnbindsDirectAsset(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	gid, _ := setupTargetWithInstance(t, db, model.TargetTypeGroup, model.SyncModeContinuous)
	var calls []installCall
	defer withInstallCapture(t, &calls)()
	if err := db.Create(&model.GroupConfigBinding{GroupID: gid, ConfigType: model.AssetBindingTypeSkill, ConfigKey: "scoped-skill"}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return publishScopeRemoval(context.Background(), tx, model.AssetTypeSkill, "scoped-skill", nil, []uint{gid})
	}); err != nil {
		t.Fatal(err)
	}

	if n := countRecords(t, db, model.TargetTypeGroup, gid); n != 1 {
		t.Fatalf("期望对被移出目标记 1 条, 实际 %d", n)
	}
	var rec model.AssetVersionRecord
	db.Where("target_type = ? AND target_id = ?", model.TargetTypeGroup, gid).First(&rec)
	if rec.TriggerReason != model.TriggerReasonScopeChanged {
		t.Fatalf("期望 asset_scope_changed, 实际 %s", rec.TriggerReason)
	}
	if len(calls) != 0 {
		t.Fatalf("移出目标期望不下发, 实际 %d 次", len(calls))
	}
	var bindingCount int64
	db.Model(&model.GroupConfigBinding{}).Where("group_id = ? AND config_type = ? AND config_key = ?", gid, model.AssetBindingTypeSkill, "scoped-skill").Count(&bindingCount)
	if bindingCount != 0 {
		t.Fatalf("被移出目标的资产绑定应删除, 实际=%d", bindingCount)
	}
}

func TestPublishScopeRemoval_UnboundTargetNoop(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	projectID, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)

	if err := db.Transaction(func(tx *gorm.DB) error {
		return publishScopeRemoval(context.Background(), tx, model.AssetTypeRule, "unbound-rule", []uint{projectID}, nil)
	}); err != nil {
		t.Fatal(err)
	}
	if n := countRecords(t, db, model.TargetTypeProject, projectID); n != 0 {
		t.Fatalf("未绑定资产的目标不应产生版本记录, 实际=%d", n)
	}
}

// TestPublishScopeRemoval_EmptyNoop 守护：无被移出目标时不落任何记录（纯新增不触发）。
func TestPublishScopeRemoval_EmptyNoop(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	var calls []installCall
	defer withInstallCapture(t, &calls)()

	if err := db.Transaction(func(tx *gorm.DB) error {
		return publishScopeRemoval(context.Background(), tx, model.AssetTypeSkill, "scoped-skill", nil, nil)
	}); err != nil {
		t.Fatal(err)
	}

	var n int64
	db.Model(&model.AssetVersionRecord{}).Count(&n)
	if n != 0 {
		t.Fatalf("无移出期望 0 条记录, 实际 %d", n)
	}
	if len(calls) != 0 {
		t.Fatalf("期望不下发, 实际 %d", len(calls))
	}
}

// TestHandleAdminAssetDetail_SyncMode 守护：detail 响应返回 target.sync_mode，
// 项目固定 continuous，分组取 user_groups.sync_mode。
func TestHandleAdminAssetDetail_SyncMode(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	makeAdminUser(t, context.Background(), "admin")

	// 分组 initial_only → 响应 sync_mode=initial_only
	gid, _ := setupTargetWithInstance(t, db, model.TargetTypeGroup, model.SyncModeInitialOnly)
	rr := httptest.NewRecorder()
	req := adminSessionReq(t, http.MethodGet,
		"/admin/assets/detail?target_type=group&target_id="+strconv.FormatUint(uint64(gid), 10), nil, "admin")
	HandleAdminAssetDetail(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("group detail status=%d body=%s", rr.Code, rr.Body.String())
	}
	var gResp struct {
		CurrentVersion int `json:"current_version"`
		Target         struct {
			Type     string `json:"type"`
			SyncMode string `json:"sync_mode"`
		} `json:"target"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &gResp); err != nil {
		t.Fatalf("decode group: %v body=%s", err, rr.Body.String())
	}
	if gResp.Target.SyncMode != model.SyncModeInitialOnly {
		t.Fatalf("group sync_mode 期望 initial_only, 实际 %s", gResp.Target.SyncMode)
	}
	if gResp.CurrentVersion != 0 {
		t.Fatalf("group 未保存 current_version 期望 0(暂无版本), 实际 %d", gResp.CurrentVersion)
	}

	// 项目 continuous → 响应 sync_mode=continuous
	pid, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	rr2 := httptest.NewRecorder()
	req2 := adminSessionReq(t, http.MethodGet,
		"/admin/assets/detail?target_type=project&target_id="+strconv.FormatUint(uint64(pid), 10), nil, "admin")
	HandleAdminAssetDetail(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("project detail status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	var pResp struct {
		Target struct {
			SyncMode string `json:"sync_mode"`
		} `json:"target"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &pResp); err != nil {
		t.Fatalf("decode project: %v body=%s", err, rr2.Body.String())
	}
	if pResp.Target.SyncMode != model.SyncModeContinuous {
		t.Fatalf("project sync_mode 期望 continuous, 实际 %s", pResp.Target.SyncMode)
	}

	// 保存后 current_version 反映真实版本号（版本记录从 v1 起，非写死 0）
	if err := RecordAssetSave(context.Background(), db, SaveInput{
		TargetType: model.TargetTypeProject, TargetID: pid, SyncMode: model.SyncModeContinuous,
		OldSyncMode:  model.SyncModeContinuous,
		Assets:       []AssetBindingItem{{AssetType: model.AssetTypeSkill, Slug: "s"}},
		OldAssets:    []AssetBindingItem{},
		OperatorID:   8812,
		OperatorName: "zhangsan",
	}); err != nil {
		t.Fatal(err)
	}
	rr3 := httptest.NewRecorder()
	req3 := adminSessionReq(t, http.MethodGet,
		"/admin/assets/detail?target_type=project&target_id="+strconv.FormatUint(uint64(pid), 10), nil, "admin")
	HandleAdminAssetDetail(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("saved project detail status=%d body=%s", rr3.Code, rr3.Body.String())
	}
	var saved struct {
		CurrentVersion int `json:"current_version"`
	}
	if err := json.Unmarshal(rr3.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode saved project: %v body=%s", err, rr3.Body.String())
	}
	if saved.CurrentVersion != 1 {
		t.Fatalf("保存后 current_version 期望 1(v1 起), 实际 %d", saved.CurrentVersion)
	}
}

// TestHandleAdminAssetSave_SyncMode 守护：save 请求带 sync_mode 时，
// 1) 分组存进 user_groups.sync_mode；2) initial_only→continuous 跳变触发下发。
func TestHandleAdminAssetSave_SyncMode(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	makeAdminUser(t, context.Background(), "admin")

	// 分组初始 initial_only
	gid, _ := setupTargetWithInstance(t, db, model.TargetTypeGroup, model.SyncModeInitialOnly)

	// 1. 带 sync_mode=continuous 的 save（空资产列表，仅改模式）
	var calls []installCall
	defer withInstallCapture(t, &calls)()
	saveBody := map[string]any{"target_type": "group", "target_id": gid, "sync_mode": "continuous", "assets": []any{}}
	rr := httptest.NewRecorder()
	req := adminSessionReq(t, http.MethodPost, "/admin/assets/save", saveBody, "admin")
	HandleAdminAssetSave(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("save status=%d body=%s", rr.Code, rr.Body.String())
	}
	// 分组 sync_mode 已落库为 continuous
	var g model.UserGroup
	if err := db.Where("id = ?", gid).First(&g).Error; err != nil {
		t.Fatal(err)
	}
	if g.SyncMode != model.SyncModeContinuous {
		t.Fatalf("group sync_mode 期望 continuous, 实际 %s", g.SyncMode)
	}
	// 跳变触发的全量下发由底层 TestRecordAssetSave_SyncModeJumpTriggersInstall 守护
	// 同时守护：跳变被记录进 changes_json（证明 oldSyncMode 是 UPDATE 前查出的旧值 initial_only，
	// 而非被 UPDATE 覆盖后的 continuous——否则跳变检测会失效、changes 里无 sync_mode）
	var rec model.AssetVersionRecord
	if err := db.Where("target_type = ? AND target_id = ?", model.TargetTypeGroup, gid).
		Order("version DESC").First(&rec).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rec.ChangesJSON, model.SyncModeContinuous) {
		t.Fatalf("跳变后版本记录 changes_json 期望含 sync_mode=continuous, 实际 %q", rec.ChangesJSON)
	}

	// 2. 项目带 sync_mode=initial_only → 参数错误
	pid, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	saveBody2 := map[string]any{"target_type": "project", "target_id": pid, "sync_mode": "initial_only", "assets": []any{}}
	rr2 := httptest.NewRecorder()
	req2 := adminSessionReq(t, http.MethodPost, "/admin/assets/save", saveBody2, "admin")
	HandleAdminAssetSave(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("项目 initial_only 期望 400, 实际 %d body=%s", rr2.Code, rr2.Body.String())
	}

	// 3. sync_mode 非法值 → 参数错误
	saveBody3 := map[string]any{"target_type": "group", "target_id": gid, "sync_mode": "foo", "assets": []any{}}
	rr3 := httptest.NewRecorder()
	req3 := adminSessionReq(t, http.MethodPost, "/admin/assets/save", saveBody3, "admin")
	HandleAdminAssetSave(rr3, req3)
	if rr3.Code != http.StatusBadRequest {
		t.Fatalf("sync_mode=foo 期望 400, 实际 %d", rr3.Code)
	}
}

func TestHandleAdminAssetSave_ProjectAcceptsVisibleJSONAssets(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	defer mockDispatchInstall(t)()
	makeAdminUser(t, context.Background(), "admin")
	projectID, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)
	if err := db.Create(&[]model.Skill{{Slug: "visible-skill", Name: "可见技能", Version: "1.0.0", VisibilityType: model.VisibilityAll}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]model.EnterpriseRule{{Slug: "visible-rule", Name: "可见规范", Version: "1.0.0", Type: "rule", VisibilityType: model.VisibilityAll}}).Error; err != nil {
		t.Fatal(err)
	}

	body := map[string]any{
		"target_type":      "project",
		"target_id":        projectID,
		"expected_version": 2, // 兼容前端携带的乐观锁字段；当前接口不据此校验。
		"sync_mode":        model.SyncModeContinuous,
		"assets": []any{
			map[string]any{"asset_type": model.AssetTypeSkill, "slug": "visible-skill"},
			map[string]any{"asset_type": model.AssetTypeRule, "slug": "visible-rule"},
		},
	}
	rr := httptest.NewRecorder()
	HandleAdminAssetSave(rr, adminSessionReq(t, http.MethodPost, "/admin/assets/save", body, "admin"))
	if rr.Code != http.StatusOK {
		t.Fatalf("项目保存可见 JSON 资产 status=%d body=%s", rr.Code, rr.Body.String())
	}
	var count int64
	db.Model(&model.ProjectConfigBinding{}).Where("project_id = ? AND config_type IN ?", projectID, model.ProjectAssetConfigTypes).Count(&count)
	if count != 2 {
		t.Fatalf("项目资产绑定期望 2 条，实际=%d", count)
	}
}

func TestHandleAdminAssetSave_RejectsIneligibleAsset(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	defer setupAssetVersionStore(t)()
	makeAdminUser(t, context.Background(), "admin")
	projectID, _ := setupTargetWithInstance(t, db, model.TargetTypeProject, model.SyncModeContinuous)

	body := map[string]any{
		"target_type": "project",
		"target_id":   projectID,
		"sync_mode":   model.SyncModeContinuous,
		"assets":      []any{map[string]any{"asset_type": model.AssetTypeSkill, "slug": "out-of-scope"}},
	}
	rr := httptest.NewRecorder()
	HandleAdminAssetSave(rr, adminSessionReq(t, http.MethodPost, "/admin/assets/save", body, "admin"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("范围外资产应返回 400, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "assets[skill:out-of-scope]") {
		t.Fatalf("错误应定位具体无效资产, body=%s", rr.Body.String())
	}
}

// TestBuildSegments_SyncModeInSegment 守护：
// 1) 仅 sync_mode 变化（资产不变）时 segments 含 sync_mode 段
// 2) 同时改了 skill 和 sync_mode 时，added 段与 sync_mode 段都要出现
func TestBuildSegments_SyncModeInSegment(t *testing.T) {
	db := setupAssetVersionTestDB(t)

	// 1) 仅 sync_mode 变化
	onlyMode := model.AssetVersionRecord{
		TargetType: model.TargetTypeGroup, TargetID: 1, Version: 1,
		TriggerType: model.TriggerTypeManual, TriggerReason: model.TriggerReasonManualSave,
		ChangesJSON: `{"added":[],"removed":[],"updated":[],"sync_mode":"continuous"}`,
	}
	segs, err := buildSegments(db, onlyMode)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSegmentType(segs, "sync_mode") {
		t.Fatalf("仅改 sync_mode 期望含 sync_mode 段, 实际 %v", segs)
	}
	// sync_mode 段的模式值应放在 items[0].name，而非独立 value 字段
	sm := findSegment(segs, "sync_mode")
	smItems := sm["items"].([]any)
	if len(smItems) != 1 {
		t.Fatalf("sync_mode 段期望 items 含 1 个元素, 实际 %v", smItems)
	}
	firstItem := smItems[0].(map[string]any)
	if firstItem["name"] != "continuous" {
		t.Fatalf("sync_mode 段期望 items[0].name=continuous, 实际 %v", firstItem)
	}
	if _, ok := sm["value"]; ok {
		t.Fatalf("sync_mode 段不应再含 value 字段, 实际 %v", sm)
	}

	// 2) 同时改 skill + sync_mode
	both := model.AssetVersionRecord{
		TargetType: model.TargetTypeGroup, TargetID: 1, Version: 2,
		TriggerType: model.TriggerTypeManual, TriggerReason: model.TriggerReasonManualSave,
		ChangesJSON: `{"added":[{"asset_type":"skill","slug":"x","name":"x"}],"removed":[],"updated":[],"sync_mode":"continuous"}`,
	}
	segs2, err := buildSegments(db, both)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSegmentType(segs2, "added") || !hasSegmentType(segs2, "sync_mode") {
		t.Fatalf("同时改 skill+sync_mode 期望 added 与 sync_mode 段都在, 实际 %v", segs2)
	}
}

// TestBuildSegments_NameFromSkillTable 守护：segments 的 items.name 取 skills 表真实 name，而非 slug。
func TestBuildSegments_NameFromSkillTable(t *testing.T) {
	db := setupAssetVersionTestDB(t)
	// 插入真实 skill 记录（显式设 Identifier 对齐测试多租户快照）
	if err := db.Create(&model.Skill{Slug: "cvm-helper", Name: "CVM 助手", Identifier: "test-tenant"}).Error; err != nil {
		t.Fatal(err)
	}
	rec := model.AssetVersionRecord{
		TargetType: model.TargetTypeGroup, TargetID: 1, Version: 1,
		TriggerType: model.TriggerTypeManual, TriggerReason: model.TriggerReasonManualSave,
		ChangesJSON: `{"added":[{"asset_type":"skill","slug":"cvm-helper","name":"cvm-helper"}],"removed":[],"updated":[],"sync_mode":""}`,
	}
	segs, err := buildSegments(db, rec)
	if err != nil {
		t.Fatal(err)
	}
	added := findSegment(segs, "added")
	if added == nil {
		t.Fatalf("期望 added 段, 实际 %v", segs)
	}
	items := added["items"].([]map[string]any)
	if len(items) == 0 {
		t.Fatal("added 段 items 为空")
	}
	first := items[0]
	if first["name"] != "CVM 助手" {
		t.Fatalf("items.name 期望技能表真实名 'CVM 助手', 实际 %v", first["name"])
	}
	if first["name"] == "cvm-helper" {
		t.Fatalf("items.name 不应是 slug")
	}

	// rule 类型同样应取 enterprise_rules 表真实 name
	if err := db.Create(&model.EnterpriseRule{Slug: "secure-coding", Name: "代码安全规范", Identifier: "test-tenant"}).Error; err != nil {
		t.Fatal(err)
	}
	recRule := model.AssetVersionRecord{
		TargetType: model.TargetTypeGroup, TargetID: 1, Version: 2,
		TriggerType: model.TriggerTypeManual, TriggerReason: model.TriggerReasonManualSave,
		ChangesJSON: `{"added":[{"asset_type":"rule","slug":"secure-coding","name":"secure-coding"}],"removed":[],"updated":[],"sync_mode":""}`,
	}
	segsR, err := buildSegments(db, recRule)
	if err != nil {
		t.Fatal(err)
	}
	addedR := findSegment(segsR, "added")
	if addedR == nil {
		t.Fatalf("期望 added 段(rule), 实际 %v", segsR)
	}
	itemsR := addedR["items"].([]map[string]any)
	if itemsR[0]["name"] != "代码安全规范" {
		t.Fatalf("rule items.name 期望 '代码安全规范', 实际 %v", itemsR[0]["name"])
	}
}

func hasSegmentType(segs []map[string]any, typ string) bool {
	return findSegment(segs, typ) != nil
}

func findSegment(segs []map[string]any, typ string) map[string]any {
	for _, s := range segs {
		if s["type"] == typ {
			return s
		}
	}
	return nil
}

// TestDiffAssets_EmptyNotNil 守护：无变更时 diffAssets 序列化的 changes_json
// 切片字段为空数组 [] 而非 null（避免落库 {"added":null,...}）。
func TestDiffAssets_EmptyNotNil(t *testing.T) {
	ch := diffAssets(nil, nil)
	b, err := json.Marshal(ch)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if s == "" || s == "null" {
		t.Fatalf("changes_json 不应为空/null, 实际 %s", s)
	}
	for _, key := range []string{"added", "removed", "updated"} {
		if !strings.Contains(s, `"`+key+`":[]`) {
			t.Fatalf("changes_json 期望 %q 为 [] 而非 null, 实际 %s", key, s)
		}
	}
}

package controller

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupCoverageTest 创建包含 stale-instances 全部表的 SQLite 内存 DB。
func setupCoverageTest(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.UserGroup{}, &model.Instance{},
		&model.InstanceFlag{}, &model.InstanceChangeGroupRecord{},
		&model.UserGroupMember{}, &model.GroupClosure{},
		&model.Notification{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	origToken := AdminToken
	AdminToken = "test-admin-token"
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	return func() {
		origDB()
		Store = origStore
		AdminToken = origToken
	}
}

func covSeedInstance(t *testing.T, id, userID, groupID uint, name, cvmID string) *model.Instance {
	t.Helper()
	inst := model.Instance{
		Model:      gorm.Model{ID: id},
		UserID:     userID,
		GroupID:    groupID,
		Name:       name,
		InstanceId: cvmID,
	}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("seed instance: %v", err)
	}
	return &inst
}

func covSeedUser(t *testing.T, id uint, username string) {
	t.Helper()
	u := model.User{Model: gorm.Model{ID: id}, Username: username}
	if err := model.DB(context.Background()).Create(&u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func covSeedMember(t *testing.T, groupID, userID uint) {
	t.Helper()
	m := model.UserGroupMember{UserGroupID: groupID, UserID: userID}
	if err := model.DB(context.Background()).Create(&m).Error; err != nil {
		t.Fatalf("seed member: %v", err)
	}
}

func covSeedGroup(t *testing.T, id uint, name string) {
	t.Helper()
	g := model.UserGroup{ID: id, Name: name, FullPath: name}
	if err := model.DB(context.Background()).Create(&g).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}
}

func covSeedClosureSelf(t *testing.T, id uint) {
	t.Helper()
	c := model.GroupClosure{AncestorID: id, DescendantID: id, Depth: 0}
	if err := model.DB(context.Background()).Create(&c).Error; err != nil {
		t.Fatalf("seed closure: %v", err)
	}
}

// ── model/instance_flag.go ──────────────────────────────────────────────────

func TestInstanceFlag_CRUD(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()

	// Add
	if err := model.AddInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup, ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := model.AddInstanceFlag(ctx, 1, model.InstanceFlagPendingUserAction, `{"k":"v"}`); err != nil {
		t.Fatalf("Add2: %v", err)
	}

	// Has
	has, err := model.HasInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup)
	if err != nil || !has {
		t.Errorf("Has stale_group: got %v err %v", has, err)
	}
	has, err = model.HasInstanceFlag(ctx, 1, "nonexistent")
	if err != nil || has {
		t.Errorf("Has nonexistent: got %v err %v", has, err)
	}

	// Get
	flags, err := model.GetInstanceFlags(ctx, 1)
	if err != nil || len(flags) != 2 {
		t.Errorf("Get: got %d flags err %v", len(flags), err)
	}

	// GetBatch
	batch, err := model.GetInstanceFlagsBatch(ctx, []uint{1, 2})
	if err != nil || len(batch) != 1 || len(batch[1]) != 2 {
		t.Errorf("Batch: got %v err %v", batch, err)
	}
	// Empty batch
	batch, _ = model.GetInstanceFlagsBatch(ctx, nil)
	if batch == nil {
		t.Error("nil input should return non-nil map")
	}

	// Remove
	if err := model.RemoveInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	has, _ = model.HasInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup)
	if has {
		t.Error("should be removed")
	}

	// ClearAll
	if err := model.ClearAllInstanceFlags(model.DB(ctx), 1); err != nil {
		t.Fatalf("ClearAll: %v", err)
	}
	flags, _ = model.GetInstanceFlags(ctx, 1)
	if len(flags) != 0 {
		t.Errorf("ClearAll: got %d flags", len(flags))
	}
}

// ── model/instance_change_group_record.go ───────────────────────────────────

func TestICGR_CreateAndList(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()

	r1 := &model.InstanceChangeGroupRecord{
		InstancePK: 100, InstanceID: "ins-100", UserIDBefore: 1, UserIDAfter: 2,
		GroupIDBefore: 5, GroupIDAfter: 7, Action: model.ICGRActionMigrate,
		ActorType: model.ICGRActorAdmin, ActorID: 99, TriggerSource: "user_edit",
	}
	if err := model.CreateICGR(ctx, r1); err != nil {
		t.Fatalf("Create: %v", err)
	}
	r2 := &model.InstanceChangeGroupRecord{
		InstancePK: 101, InstanceID: "ins-101", UserIDBefore: 3, UserIDAfter: 3,
		GroupIDBefore: 0, GroupIDAfter: 5, Action: model.ICGRActionHandover,
		ActorType: model.ICGRActorAdmin, ActorID: 99, TriggerSource: "user_edit",
	}
	if err := model.CreateICGR(ctx, r2); err != nil {
		t.Fatalf("Create2: %v", err)
	}

	// List all
	rows, total, err := model.ListICGRs(ctx, model.ListICGRsParams{Page: 1, PageSize: 10})
	if err != nil || total != 2 || len(rows) != 2 {
		t.Fatalf("List: got %d rows total %d err %v", len(rows), total, err)
	}

	// Filter by action
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{Action: model.ICGRActionMigrate, Page: 1, PageSize: 10})
	if total != 1 || rows[0].Action != model.ICGRActionMigrate {
		t.Errorf("filter action: got total %d", total)
	}

	// Filter by user
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{UserID: 1, Page: 1, PageSize: 10})
	if total != 1 {
		t.Errorf("filter user: got total %d", total)
	}

	// Filter by group (nil = no filter, &0 = ungrouped, &5 = group 5)
	gid := uint(5)
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{GroupID: &gid, Page: 1, PageSize: 10})
	if total != 2 {
		t.Errorf("filter group=5: got total %d", total)
	}
	gid0 := uint(0)
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{GroupID: &gid0, Page: 1, PageSize: 10})
	if total != 1 {
		t.Errorf("filter group=0: got total %d", total)
	}

	// Filter by instance_pk
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{InstancePK: 100, Page: 1, PageSize: 10})
	if total != 1 {
		t.Errorf("filter inst: got total %d", total)
	}

	// Filter by instance_id
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{InstanceID: "ins-101", Page: 1, PageSize: 10})
	if total != 1 {
		t.Errorf("filter inst_id: got total %d", total)
	}

	// Filter by actor_type
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{ActorType: model.ICGRActorAdmin, Page: 1, PageSize: 10})
	if total != 2 {
		t.Errorf("filter actor: got total %d", total)
	}

	// Filter by trigger_source
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{TriggerSource: "user_edit", Page: 1, PageSize: 10})
	if total != 2 {
		t.Errorf("filter trigger: got total %d", total)
	}

	// Filter by time range
	now := time.Now()
	from := now.Add(-1 * time.Hour)
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{From: &from, Page: 1, PageSize: 10})
	if total != 2 {
		t.Errorf("filter from: got total %d", total)
	}
	future := now.Add(1 * time.Hour)
	rows, total, _ = model.ListICGRs(ctx, model.ListICGRsParams{To: &future, Page: 1, PageSize: 10})
	if total != 2 {
		t.Errorf("filter to: got total %d", total)
	}
}

func TestCreateICGRTx(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	tx := model.DB(ctx).Begin()
	r := &model.InstanceChangeGroupRecord{
		InstancePK: 1, Action: model.ICGRActionMigrate,
		ActorType: model.ICGRActorAdmin, TriggerSource: "test",
	}
	if err := model.CreateICGRTx(tx, r); err != nil {
		tx.Rollback()
		t.Fatalf("CreateICGRTx: %v", err)
	}
	tx.Commit()
	rows, _, _ := model.ListICGRs(ctx, model.ListICGRsParams{Page: 1, PageSize: 10})
	if len(rows) != 1 {
		t.Errorf("want 1, got %d", len(rows))
	}
}

// ── admin_instances_group_check.go ────────────────────────────────────────────

func TestLoadUserGroupMemberships(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20)
	covSeedMember(t, 7, 10)

	m := loadUserGroupMemberships(ctx, []uint{10, 20})
	if len(m[10]) != 2 { // groups 5, 7
		t.Errorf("user 10: want 2 groups, got %v", m[10])
	}
	if len(m[20]) != 1 { // group 5
		t.Errorf("user 20: want 1 group, got %v", m[20])
	}
	// empty input
	m = loadUserGroupMemberships(ctx, nil)
	if len(m) != 0 {
		t.Errorf("nil → want empty, got %v", m)
	}
}

func TestLoadInstancesByIDs(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedInstance(t, 1, 10, 5, "inst-1", "ins-1")
	covSeedInstance(t, 2, 20, 7, "inst-2", "ins-2")

	m := loadInstancesByIDs(ctx, []uint{1, 2, 999})
	if len(m) != 2 {
		t.Fatalf("want 2, got %d", len(m))
	}
	if m[1] == nil || m[1].Name != "inst-1" {
		t.Errorf("inst 1: %+v", m[1])
	}
	if m[2] == nil || m[2].Name != "inst-2" {
		t.Errorf("inst 2: %+v", m[2])
	}
	// empty
	m = loadInstancesByIDs(ctx, nil)
	if len(m) != 0 {
		t.Errorf("nil → want empty")
	}
}

func TestEnrichAdminInstancesWithGroupCheck_Empty(t *testing.T) {
	defer setupCoverageTest(t)()
	// empty items → no-op
	enrichAdminInstancesWithGroupCheck(context.Background(), nil, true, true)
	// both flags false → no-op
	enrichAdminInstancesWithGroupCheck(context.Background(), []groupCheckItem{{ID: 1}}, false, false)
}

func TestEnrichAdminInstancesWithGroupCheck_UserGroupMismatch(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedMember(t, 5, 10) // user 10 is in group 5

	items := []groupCheckItem{
		{ID: 1, UserID: 10, GroupID: 5},  // in group → no mismatch
		{ID: 2, UserID: 10, GroupID: 7},  // not in group → mismatch
		{ID: 3, UserID: 10, GroupID: 0},  // ungrouped but has groups → mismatch
	}
	enrichAdminInstancesWithGroupCheck(ctx, items, true, false)
	if items[0].UserGroupMismatch {
		t.Error("inst 1: should not mismatch")
	}
	if !items[1].UserGroupMismatch {
		t.Error("inst 2: should mismatch")
	}
	if !items[2].UserGroupMismatch {
		t.Error("inst 3: should mismatch (ungrouped but user has groups)")
	}
}

// ── admin_stale_instances_action_options.go query functions ───────────────────

func TestQueryGroupZeroInstances(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedInstance(t, 1, 10, 0, "a", "ins-a") // group_id=0
	covSeedInstance(t, 2, 20, 5, "b", "ins-b") // group_id=5

	rows := queryGroupZeroInstances(ctx, []uint{10, 20})
	if len(rows) != 1 || rows[0].ID != 1 {
		t.Errorf("want 1 row (inst 1), got %v", rows)
	}
	// empty
	rows = queryGroupZeroInstances(ctx, nil)
	if rows != nil {
		t.Errorf("nil → want nil, got %v", rows)
	}
}

func TestQueryInstancesByPairs(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedInstance(t, 1, 10, 5, "a", "ins-a")
	covSeedInstance(t, 2, 20, 7, "b", "ins-b")

	pairs := map[userGroupPair]struct{}{
		{UserID: 10, GroupID: 5}: {},
		{UserID: 20, GroupID: 7}: {},
		{UserID: 99, GroupID: 99}: {}, // no match
	}
	rows := queryInstancesByPairs(ctx, pairs)
	if len(rows) != 2 {
		t.Fatalf("want 2, got %d", len(rows))
	}
	// empty
	rows = queryInstancesByPairs(ctx, nil)
	if rows != nil {
		t.Errorf("nil → want nil")
	}
}

func TestQueryInstancesBySubtree(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedGroup(t, 1, "root")
	covSeedGroup(t, 2, "child")
	covSeedClosureSelf(t, 1)
	covSeedClosureSelf(t, 2)
	// 1 → 2 (parent → child)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})
	covSeedInstance(t, 1, 10, 1, "a", "ins-a")
	covSeedInstance(t, 2, 20, 2, "b", "ins-b")
	covSeedInstance(t, 3, 30, 5, "c", "ins-c") // not in subtree

	rows := queryInstancesBySubtree(ctx, []uint{1})
	if len(rows) != 2 {
		t.Fatalf("want 2 (groups 1+2), got %d", len(rows))
	}
	// empty
	rows = queryInstancesBySubtree(ctx, nil)
	if rows != nil {
		t.Errorf("nil → want nil")
	}
	// group 0 filtered
	rows = queryInstancesBySubtree(ctx, []uint{0})
	if rows != nil {
		t.Errorf("group 0 → want nil")
	}
}

func TestLoadStaleGroupFlaggedInstanceIDs(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup, "")
	model.AddInstanceFlag(ctx, 2, model.InstanceFlagStaleGroup, "")
	model.AddInstanceFlag(ctx, 3, model.InstanceFlagPendingUserAction, "")

	rows := []staleActionInstRow{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}
	flagged := loadStaleGroupFlaggedInstanceIDs(ctx, rows)
	if len(flagged) != 2 {
		t.Fatalf("want 2 flagged (1,2), got %d", len(flagged))
	}
	if _, ok := flagged[1]; !ok {
		t.Error("inst 1 should be flagged")
	}
	if _, ok := flagged[2]; !ok {
		t.Error("inst 2 should be flagged")
	}
	if _, ok := flagged[3]; ok {
		t.Error("inst 3 should NOT be flagged (pending_user_action, not stale_group)")
	}
	// empty
	flagged = loadStaleGroupFlaggedInstanceIDs(ctx, nil)
	if flagged != nil {
		t.Errorf("nil → want nil")
	}
}

// ── stale_instances_apply.go apply engine ─────────────────────────────────────

func TestApplyEngine_Migrate_ScenarioA(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10) // user 10 now in group 7, not 5
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{7: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionMigrate, InstanceIDs: []uint{inst.ID}, TargetGroupID: uintPtr(7)},
	})

	if len(engine.results) != 1 {
		t.Fatalf("want 1 result, got %d", len(engine.results))
	}
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, error=%s", r.Status, r.Error)
	}
	// verify group_id changed
	var updated model.Instance
	model.DB(ctx).First(&updated, inst.ID)
	if updated.GroupID != 7 {
		t.Errorf("group_id: got %d want 7", updated.GroupID)
	}
}

func TestApplyEngine_Migrate_Noop(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10) // user still in group 5
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionMigrate, InstanceIDs: []uint{inst.ID}, TargetGroupID: uintPtr(5)},
	})
	// noop: user is still in the group, scenario = noop
	r := engine.results[0]
	if r.Status != "noop" {
		t.Errorf("status: got %q want noop", r.Status)
	}
}

func TestApplyEngine_Handover_SameGroup(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20) // bob is in same group
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true}, map[uint]bool{10: true, 20: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 20},
	})

	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, error=%s", r.Status, r.Error)
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, inst.ID)
	if updated.UserID != 20 {
		t.Errorf("user_id: got %d want 20", updated.UserID)
	}
}

func TestApplyEngine_Handover_SameUser(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, nil, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 10},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_ArchiveStop(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10) // user 10 in group 7, NOT in group 5
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1") // instance in group 5
	model.AddInstanceFlag(ctx, inst.ID, model.InstanceFlagStaleGroup, "")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionArchiveStop, InstanceIDs: []uint{inst.ID}},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, error=%s", r.Status, r.Error)
	}
	// archive_stop keeps stale_group flag (instance is archived, not resolved)
	has, _ := model.HasInstanceFlag(ctx, inst.ID, model.InstanceFlagStaleGroup)
	if !has {
		t.Error("stale_group flag should still be set after archive_stop")
	}
}

func TestApplyEngine_PendingUser(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10) // user 10 in group 7, NOT in group 5
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")
	model.AddInstanceFlag(ctx, inst.ID, model.InstanceFlagStaleGroup, "")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionPendingUser, InstanceIDs: []uint{inst.ID}, AllowMigrate: true},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, error=%s", r.Status, r.Error)
	}
	has, _ := model.HasInstanceFlag(ctx, inst.ID, model.InstanceFlagPendingUserAction)
	if !has {
		t.Error("pending_user_action flag should be set")
	}
	has, _ = model.HasInstanceFlag(ctx, inst.ID, model.InstanceFlagAllowMigrate)
	if !has {
		t.Error("allow_migrate flag should be set")
	}
}

func TestApplyEngine_PendingUser_NoSubOption(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10) // user NOT in instance's group
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")
	model.AddInstanceFlag(ctx, inst.ID, model.InstanceFlagStaleGroup, "")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionPendingUser, InstanceIDs: []uint{inst.ID}}, // no sub-options
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_InstanceNotFound(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, nil, nil)
	engine.run([]applyActionItem{
		{Action: StaleActionArchiveStop, InstanceIDs: []uint{99999}},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_UnsupportedAction(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, nil, nil)
	engine.run([]applyActionItem{
		{Action: "delete", InstanceIDs: []uint{1}},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_GroupsOfUser(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 7, 10)

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, nil, nil)
	ids, err := engine.groupsOfUser(10)
	if err != nil {
		t.Fatalf("groupsOfUser: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("want 2 groups, got %d: %v", len(ids), ids)
	}
	// cached call
	ids2, _ := engine.groupsOfUser(10)
	if len(ids2) != len(ids) {
		t.Error("cache miss")
	}
}

// ── helper ─────────────────────────────────────────────────────────────────────

func uintPtr(v uint) *uint { return &v }

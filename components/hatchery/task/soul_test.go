package task

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"hatchery/model"
)

// ---------- DB 初始化 ----------

func initSoulTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Instance{},
		&model.OpenClawRole{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

func createTestRole(t *testing.T) model.OpenClawRole {
	t.Helper()
	role := model.OpenClawRole{Name: "测试角色", Soul: "test soul content", Visible: true}
	if err := model.DB(context.Background()).Create(&role).Error; err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	return role
}

func createTestInstance(t *testing.T, roleID uint, runtimeUser string, soulSetAt *time.Time) model.Instance {
	t.Helper()
	inst := model.Instance{
		Name:        "test-instance",
		InstanceId:  "ins-test123",
		RoleID:      roleID,
		RuntimeUser: runtimeUser,
	}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	if soulSetAt != nil {
		model.DB(context.Background()).Model(&inst).Update("soul_set_at", *soulSetAt)
		inst.SoulSetAt = soulSetAt
	}
	return inst
}

// ---------- mock dependencies ----------

type mockSoulDeps struct {
	setSoulFn func(ctx context.Context, instancePK uint) error
	callCount atomic.Int32
}

func (m *mockSoulDeps) SetSoul(ctx context.Context, instancePK uint) error {
	m.callCount.Add(1)
	if m.setSoulFn != nil {
		return m.setSoulFn(ctx, instancePK)
	}
	return nil
}

// ---------- Tests ----------

func TestDoSoulSet_NoPendingInstances(t *testing.T) {
	initSoulTestDB(t)
	role := createTestRole(t)

	now := time.Now()
	createTestInstance(t, role.ID, "testuser", &now)

	deps := &mockSoulDeps{}
	doSoulSet(context.Background(), deps)

	if deps.callCount.Load() != 0 {
		t.Errorf("已下发的实例不应触发调用，但调用了 %d 次", deps.callCount.Load())
	}
}

func TestDoSoulSet_PendingInstances(t *testing.T) {
	initSoulTestDB(t)
	role := createTestRole(t)

	createTestInstance(t, role.ID, "testuser", nil)
	createTestInstance(t, role.ID, "testuser2", nil)

	deps := &mockSoulDeps{}
	doSoulSet(context.Background(), deps)

	if deps.callCount.Load() != 2 {
		t.Errorf("期望调用 2 次，实际 %d 次", deps.callCount.Load())
	}
}

func TestDoSoulSet_NoRuntimeUser(t *testing.T) {
	initSoulTestDB(t)
	role := createTestRole(t)

	createTestInstance(t, role.ID, "", nil)

	deps := &mockSoulDeps{}
	doSoulSet(context.Background(), deps)

	if deps.callCount.Load() != 0 {
		t.Errorf("无 RuntimeUser 的实例不应触发调用，但调用了 %d 次", deps.callCount.Load())
	}
}

func TestDoSoulSet_NoRole(t *testing.T) {
	initSoulTestDB(t)

	createTestInstance(t, 0, "testuser", nil)

	deps := &mockSoulDeps{}
	doSoulSet(context.Background(), deps)

	if deps.callCount.Load() != 0 {
		t.Errorf("无角色的实例不应触发调用，但调用了 %d 次", deps.callCount.Load())
	}
}

func TestDoSoulSet_Error(t *testing.T) {
	initSoulTestDB(t)
	role := createTestRole(t)

	createTestInstance(t, role.ID, "testuser", nil)

	deps := &mockSoulDeps{
		setSoulFn: func(ctx context.Context, instancePK uint) error {
			return errors.New("TAT 下发失败")
		},
	}
	doSoulSet(context.Background(), deps)

	if deps.callCount.Load() != 1 {
		t.Errorf("期望调用 1 次，实际 %d 次", deps.callCount.Load())
	}
	var inst model.Instance
	model.DB(context.Background()).First(&inst, 1)
	if inst.SoulSetAt != nil {
		t.Error("下发失败后 soul_set_at 仍应为 NULL")
	}
}

func TestDoSoulSet_SoftDeletedIgnored(t *testing.T) {
	initSoulTestDB(t)
	role := createTestRole(t)

	inst := createTestInstance(t, role.ID, "testuser", nil)
	model.DB(context.Background()).Delete(&inst)

	deps := &mockSoulDeps{}
	doSoulSet(context.Background(), deps)

	if deps.callCount.Load() != 0 {
		t.Errorf("已删除的实例不应触发调用，但调用了 %d 次", deps.callCount.Load())
	}
}

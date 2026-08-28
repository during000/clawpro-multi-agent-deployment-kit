package model

import (
	"context"
	"os"
	"sort"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupUngroupedCVMTestDB 初始化临时 SQLite 数据库，迁移所需表，返回 cleanup 函数。
func setupUngroupedCVMTestDB(t *testing.T) func() {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "ungrouped_cvm_test_*.db")
	if err != nil {
		t.Fatalf("创建临时数据库文件失败: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	origDB := gdb
	gdb = testDB

	if err := gdb.AutoMigrate(&User{}, &UserGroup{}, &UserGroupMember{}, &Instance{}); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return func() {
		sqlDB, _ := gdb.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		gdb = origDB
	}
}

// mustCreateUser 创建测试用户，返回 ID。
func mustCreateUngroupedTestUser(t *testing.T, username string) uint {
	t.Helper()
	u := User{Username: username, Password: "x"}
	if err := gdb.Create(&u).Error; err != nil {
		t.Fatalf("创建用户 %s 失败: %v", username, err)
	}
	return u.ID
}

// mustCreateInstance 为指定用户创建一条 Instance 记录。
func mustCreateInstance(t *testing.T, userID uint, instanceID string) {
	t.Helper()
	inst := Instance{UserID: userID, InstanceId: instanceID}
	if err := gdb.Create(&inst).Error; err != nil {
		t.Fatalf("创建实例 %s 失败: %v", instanceID, err)
	}
}

// sortedStrings 对字符串切片排序后返回，方便断言时做顺序无关比较。
func sortedStrings(s []string) []string {
	cp := make([]string, len(s))
	copy(cp, s)
	sort.Strings(cp)
	return cp
}

// ─── 测试用例 ─────────────────────────────────────────────────────────────────

// 场景：数据库中没有任何用户，返回空列表
func TestGetUngroupedCVMInstanceIDs_NoUsers(t *testing.T) {
	cleanup := setupUngroupedCVMTestDB(t)
	defer cleanup()

	ids, err := GetUngroupedCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("期望空列表，实际=%v", ids)
	}
}

// 场景：所有用户都已加入用户组，返回空列表
func TestGetUngroupedCVMInstanceIDs_AllGrouped(t *testing.T) {
	cleanup := setupUngroupedCVMTestDB(t)
	defer cleanup()

	uid := mustCreateUngroupedTestUser(t, "alice")
	mustCreateInstance(t, uid, "ins-aaa")

	// 将 alice 加入用户组
	gdb.Create(&UserGroupMember{UserGroupID: 1, UserID: uid})

	ids, err := GetUngroupedCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("所有用户已分组，期望空列表，实际=%v", ids)
	}
}

// 场景：未分组用户有实例，返回其实例 ID
func TestGetUngroupedCVMInstanceIDs_UngroupedWithInstances(t *testing.T) {
	cleanup := setupUngroupedCVMTestDB(t)
	defer cleanup()

	uid := mustCreateUngroupedTestUser(t, "bob")
	mustCreateInstance(t, uid, "ins-bbb")
	mustCreateInstance(t, uid, "ins-ccc")

	ids, err := GetUngroupedCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("期望 2 个实例 ID，实际=%d: %v", len(ids), ids)
	}
	got := sortedStrings(ids)
	want := []string{"ins-bbb", "ins-ccc"}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("第 %d 个实例 ID 期望=%s，实际=%s", i, v, got[i])
		}
	}
}

// 场景：混合——部分用户已分组，部分未分组；只返回未分组用户的实例
func TestGetUngroupedCVMInstanceIDs_MixedUsers(t *testing.T) {
	cleanup := setupUngroupedCVMTestDB(t)
	defer cleanup()

	uidGrouped := mustCreateUngroupedTestUser(t, "alice")
	uidUngrouped := mustCreateUngroupedTestUser(t, "bob")

	mustCreateInstance(t, uidGrouped, "ins-grouped")
	mustCreateInstance(t, uidUngrouped, "ins-ungrouped")

	// alice 加入用户组，bob 不加
	gdb.Create(&UserGroupMember{UserGroupID: 1, UserID: uidGrouped})

	ids, err := GetUngroupedCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if len(ids) != 1 || ids[0] != "ins-ungrouped" {
		t.Errorf("期望只返回 bob 的实例 [ins-ungrouped]，实际=%v", ids)
	}
}

// 场景：未分组用户有多条相同 instance_id 的记录，结果应去重
func TestGetUngroupedCVMInstanceIDs_Dedup(t *testing.T) {
	cleanup := setupUngroupedCVMTestDB(t)
	defer cleanup()

	uid := mustCreateUngroupedTestUser(t, "carol")
	mustCreateInstance(t, uid, "ins-dup")
	mustCreateInstance(t, uid, "ins-dup") // 重复
	mustCreateInstance(t, uid, "ins-unique")

	ids, err := GetUngroupedCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("期望去重后 2 个实例 ID，实际=%d: %v", len(ids), ids)
	}
}

// 场景：未分组用户存在但没有任何实例，返回空列表
func TestGetUngroupedCVMInstanceIDs_UngroupedNoInstances(t *testing.T) {
	cleanup := setupUngroupedCVMTestDB(t)
	defer cleanup()

	mustCreateUngroupedTestUser(t, "dave")
	// 不创建任何实例

	ids, err := GetUngroupedCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("未分组用户无实例，期望空列表，实际=%v", ids)
	}
}

// 场景：instance_id 为空字符串的记录不应被返回
func TestGetUngroupedCVMInstanceIDs_EmptyInstanceIDExcluded(t *testing.T) {
	cleanup := setupUngroupedCVMTestDB(t)
	defer cleanup()

	uid := mustCreateUngroupedTestUser(t, "eve")
	mustCreateInstance(t, uid, "")        // 空 instance_id，应被过滤
	mustCreateInstance(t, uid, "ins-eve") // 有效实例

	ids, err := GetUngroupedCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if len(ids) != 1 || ids[0] != "ins-eve" {
		t.Errorf("期望只返回非空 instance_id [ins-eve]，实际=%v", ids)
	}
}

package controller

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initEnrichDeptTestDB 初始化 enrichUserDepartments 测试用的内存 SQLite。
func initEnrichDeptTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.OneIDUserProfile{},
		&model.OneIDDepartmentRecord{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))
	db.Create(&model.SiteConfig{})
	return db
}

// strPtr 在 admin_instances_test.go 中已有定义，此处复用。

// TestEnrichUserDepartments_EmptyUsers 验证传入 nil / 空切片直接返回空 map，
// 不触发任何 DB 调用。
func TestEnrichUserDepartments_EmptyUsers(t *testing.T) {
	initEnrichDeptTestDB(t)
	ctx := context.Background()

	for _, name := range []string{"nil", "empty"} {
		t.Run(name, func(t *testing.T) {
			var users []model.User
			if name == "empty" {
				users = []model.User{}
			}
			got := enrichUserDepartments(ctx, users)
			if len(got) != 0 {
				t.Fatalf("期望空 map，实际长度=%d", len(got))
			}
		})
	}
}

// TestEnrichUserDepartments_AllUsersWithoutOneIDSub 短路验证：
// 所有用户 OneIDSub 都为 nil/空字符串时，函数返回空 map，且不触发对
// oneid_user_profiles / oneid_departments 表的任何查询。
//
// 通过"不预 seed 这两个表，用一个不带这些 schema 的 DB"间接证明：
// 如果函数仍然去查这两张表，DB 会因为表不存在直接返回 SQL 错误并被 GORM
// 触发慢日志告警；这里只看返回 map 是空且无 panic。
//
// 进一步：本测试用一个**没有迁移 oneid_* 两张表**的 DB，
// 短路成立则全程零异常；短路失效则查询 missing table 会被 GORM 写到 logger
// （行为不会影响本测试的 assert，但属于错误行为）。
func TestEnrichUserDepartments_AllUsersWithoutOneIDSub(t *testing.T) {
	// 不调用 initEnrichDeptTestDB —— 故意只 migrate User 表，
	// 让 oneid_user_profiles / oneid_departments 不存在。
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("迁移 users 失败: %v", err)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))

	ctx := context.Background()
	users := []model.User{
		{Username: "alice", Password: "x", Role: "user"},                                  // OneIDSub == nil
		{Username: "bob", Password: "x", Role: "user", OneIDSub: strPtr("")},              // 空字符串
		{Username: "carol", Password: "x", Role: "user", OneIDSub: nil},                   // 显式 nil
		{Username: "dave", Password: "x", Role: "admin"},                                  // 管理员，OneIDSub == nil
		{Username: "eve", Password: "x", Role: "user", OneIDSub: (*string)(nil)},          // 显式 nil
	}
	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatalf("seed user 失败: %v", err)
		}
	}

	got := enrichUserDepartments(ctx, users)
	if len(got) != 0 {
		t.Fatalf("预期短路返回空 map（零 OneIDSub 用户），实际长度=%d", len(got))
	}
}

// TestEnrichUserDepartments_BatchFetchAndPathBuild 验证：
//   - OneID 用户能被批量查到画像
//   - departments[i].department_path 沿 parent 链反推正确
//   - 主部门 path 形如 "L1/L2/L3" 与 /admin/users 同口径
func TestEnrichUserDepartments_BatchFetchAndPathBuild(t *testing.T) {
	db := initEnrichDeptTestDB(t)
	ctx := context.Background()

	// seed 三级部门链：d1 (root) → d2 → d3
	now := time.Now()
	depts := []model.OneIDDepartmentRecord{
		{DepartmentID: "d1", DepartmentName: "OpenClaw企业版体验", DepartmentParentID: "", SyncedAt: now},
		{DepartmentID: "d2", DepartmentName: "新组", DepartmentParentID: "d1", SyncedAt: now},
		{DepartmentID: "d3", DepartmentName: "市场组", DepartmentParentID: "d2", SyncedAt: now},
	}
	for i := range depts {
		if err := db.Create(&depts[i]).Error; err != nil {
			t.Fatalf("seed dept 失败: %v", err)
		}
	}

	// seed 用户：alice 是 OneID 用户（主部门 d3），bob 是纯密码用户
	alice := model.User{Username: "alice", Password: "x", Role: "user", OneIDSub: strPtr("sub-alice")}
	bob := model.User{Username: "bob", Password: "x", Role: "user"} // OneIDSub == nil
	if err := db.Create(&alice).Error; err != nil {
		t.Fatalf("seed alice 失败: %v", err)
	}
	if err := db.Create(&bob).Error; err != nil {
		t.Fatalf("seed bob 失败: %v", err)
	}

	// alice 的画像：departments_json 含 d2 / d3 两个部门
	aliceDepts := []model.OneIDDepartment{
		{DepartmentID: "d2", DepartmentName: "新组", DepartmentParentID: "d1", IsMainDepartment: false},
		{DepartmentID: "d3", DepartmentName: "市场组", DepartmentParentID: "d2", IsMainDepartment: true},
	}
	aliceJSON, _ := json.Marshal(aliceDepts)
	profile := model.OneIDUserProfile{
		OneIDSub:        "sub-alice",
		Name:            "alice",
		MainDeptID:      "d3",
		MainDeptName:    "市场组",
		DepartmentsJSON: string(aliceJSON),
		SyncedAt:        now,
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("seed profile 失败: %v", err)
	}

	got := enrichUserDepartments(ctx, []model.User{alice, bob})
	if len(got) != 1 {
		t.Fatalf("期望仅 alice 在结果中（bob 无 OneIDSub），实际 size=%d", len(got))
	}
	d, ok := got[alice.ID]
	if !ok {
		t.Fatalf("结果中找不到 alice (id=%d)", alice.ID)
	}
	if d.Department != "市场组" {
		t.Errorf("Department=%q，期望 %q", d.Department, "市场组")
	}
	if d.DepartmentPath != "OpenClaw企业版体验/新组/市场组" {
		t.Errorf("DepartmentPath=%q，期望 %q", d.DepartmentPath, "OpenClaw企业版体验/新组/市场组")
	}
	if len(d.Departments) != 2 {
		t.Fatalf("Departments 长度=%d，期望 2", len(d.Departments))
	}
	wantPaths := []string{"OpenClaw企业版体验/新组", "OpenClaw企业版体验/新组/市场组"}
	for i, want := range wantPaths {
		if d.Departments[i].DepartmentPath != want {
			t.Errorf("Departments[%d].DepartmentPath=%q，期望 %q", i, d.Departments[i].DepartmentPath, want)
		}
	}
	// 验证 OneIDDepartment 基础字段也透传到 deptWithPath（内嵌字段访问）
	if d.Departments[1].DepartmentName != "市场组" || !d.Departments[1].IsMainDepartment {
		t.Errorf("Departments[1] 基础字段未正确透传: %+v", d.Departments[1])
	}
}

// TestEnrichUserDepartments_ProfileMissing 验证：用户 OneIDSub 存在但
// oneid_user_profiles 表中无记录时，该用户不应出现在结果 map 中（best-effort 降级）。
func TestEnrichUserDepartments_ProfileMissing(t *testing.T) {
	db := initEnrichDeptTestDB(t)
	ctx := context.Background()

	user := model.User{Username: "ghost", Password: "x", Role: "user", OneIDSub: strPtr("sub-missing")}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user 失败: %v", err)
	}

	got := enrichUserDepartments(ctx, []model.User{user})
	if _, ok := got[user.ID]; ok {
		t.Fatalf("画像缺失的用户不应出现在结果中，实际 got=%+v", got)
	}
}

// TestEnrichUserDepartments_DeptIDNotInGlobalMap 验证：画像中某部门 ID 在
// oneid_departments 表中找不到时，该部门 department_path 为空串，但其它字段正常。
func TestEnrichUserDepartments_DeptIDNotInGlobalMap(t *testing.T) {
	db := initEnrichDeptTestDB(t)
	ctx := context.Background()

	now := time.Now()
	// 只 seed d1，不 seed d-missing
	if err := db.Create(&model.OneIDDepartmentRecord{
		DepartmentID: "d1", DepartmentName: "Root", SyncedAt: now,
	}).Error; err != nil {
		t.Fatalf("seed dept 失败: %v", err)
	}

	user := model.User{Username: "alice", Password: "x", Role: "user", OneIDSub: strPtr("sub-a")}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user 失败: %v", err)
	}

	depts := []model.OneIDDepartment{
		{DepartmentID: "d-missing", DepartmentName: "幽灵部门", IsMainDepartment: true},
	}
	deptsJSON, _ := json.Marshal(depts)
	if err := db.Create(&model.OneIDUserProfile{
		OneIDSub:        "sub-a",
		MainDeptID:      "d-missing",
		MainDeptName:    "幽灵部门",
		DepartmentsJSON: string(deptsJSON),
		SyncedAt:        now,
	}).Error; err != nil {
		t.Fatalf("seed profile 失败: %v", err)
	}

	got := enrichUserDepartments(ctx, []model.User{user})
	d, ok := got[user.ID]
	if !ok {
		t.Fatalf("用户应出现在结果中（画像存在）")
	}
	if d.Department != "幽灵部门" {
		t.Errorf("Department=%q，期望 %q", d.Department, "幽灵部门")
	}
	if d.DepartmentPath != "" {
		t.Errorf("主部门 ID 不在全局映射，DepartmentPath 应为空，实际 %q", d.DepartmentPath)
	}
	if len(d.Departments) != 1 || d.Departments[0].DepartmentPath != "" {
		t.Errorf("departments[0].department_path 应为空，实际 %+v", d.Departments)
	}
	// 部门基础字段（来自 OneIDDepartment）必须保留
	if d.Departments[0].DepartmentName != "幽灵部门" {
		t.Errorf("Departments[0].DepartmentName=%q，期望 %q", d.Departments[0].DepartmentName, "幽灵部门")
	}
}

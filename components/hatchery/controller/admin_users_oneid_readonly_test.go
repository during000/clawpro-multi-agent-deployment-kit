// admin_users_oneid_readonly_test.go
//
// 验证 OneID 模式下管控端对「用户 ↔ 用户分组」写路径的保护：
//
//  1. POST /admin/create        → OneID 模式直接 409，禁止人工新建用户
//  2. POST /admin/batch-create  → 同上
//  3. POST /admin/update-user   → OneID 模式下 group_ids 走 manual-only 替换：
//     · 含 oneid_dept 组 → 409 + 不破坏已有 oneid_dept membership
//     · 全是 manual 组   → 200，且用户已有的 oneid_dept membership 必须保留
//     · 标准模式（TenantID == ""）→ 沿用旧的全量替换语义，保持兼容

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initOneIDUserOpsTestDB 准备一个含 user/usergroup/usergroupmember/site_config
// 的内存 SQLite，并设置 AdminToken 与 TenantID（默认空），返回还原函数。
func initOneIDUserOpsTestDB(t *testing.T, tenant string) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.SiteConfig{},
		&model.Instance{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	restore := model.UseDBForTest(db)
	db.Create(&model.SiteConfig{})

	origToken := AdminToken
	AdminToken = "test-admin-token"

	// 通过 FixedSnapshot 模拟 TenantID
	origFixed := hcommon.FixedSnapshot
	if tenant != "" {
		snap := hcommon.TenantSnapshot{OneIDAccountID: tenant, Identifier: tenant}
		hcommon.FixedSnapshot = &snap
	}

	return func() {
		restore()
		AdminToken = origToken
		hcommon.FixedSnapshot = origFixed
	}
}

func newAdminPostJSON(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	// 将 FixedSnapshot 注入 ctx（模拟 identifier_middleware）
	if hcommon.FixedSnapshot != nil {
		ctx := hcommon.InjectTenant(req.Context(), *hcommon.FixedSnapshot)
		req = req.WithContext(ctx)
	}
	return req
}

// uintToStr 把 uint 转字符串（避免引入 strconv）。
func uintToStr(n uint) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// TestHandleCreateUser_OneIDMode_Forbidden  OneID 模式下 /admin/create 直接 409。
func TestHandleCreateUser_OneIDMode_Forbidden(t *testing.T) {
	defer initOneIDUserOpsTestDB(t, "test-tenant")()

	body := `{"username":"newuser","password":"123456","role":"user"}`
	req := newAdminPostJSON("/admin/create", body)
	w := httptest.NewRecorder()
	HandleCreateUser(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("OneID 模式 /admin/create 期望 409，实际 %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "OneID 模式下不允许") {
		t.Errorf("错误文案缺关键词: %s", w.Body.String())
	}
	// DB 不应有这个用户
	var n int64
	model.DB(context.Background()).Model(&model.User{}).Where("username = ?", "newuser").Count(&n)
	if n != 0 {
		t.Errorf("OneID 模式下用户不应被创建，实际 %d", n)
	}
}

// TestHandleCreateUser_StandardMode_NotBlockedByGate
// 标准模式（TenantID 空）不应被新增的 OneID gate 拦下。
// 这里只断言不返回带 OneID 文案的 409，不深入校验后续创建逻辑（其他测试已覆盖）。
func TestHandleCreateUser_StandardMode_NotBlockedByGate(t *testing.T) {
	defer initOneIDUserOpsTestDB(t, "")() // 标准模式

	body := `{"username":"normaluser","password":"123456","role":"user"}`
	req := newAdminPostJSON("/admin/create", body)
	w := httptest.NewRecorder()
	HandleCreateUser(w, req)

	if w.Code == http.StatusConflict && strings.Contains(w.Body.String(), "OneID 模式下不允许") {
		t.Fatalf("标准模式不应触发 OneID gate, body=%s", w.Body.String())
	}
}

// TestHandleBatchCreateUser_OneIDMode_Forbidden  OneID 模式下 /admin/batch-create 直接 409。
func TestHandleBatchCreateUser_OneIDMode_Forbidden(t *testing.T) {
	defer initOneIDUserOpsTestDB(t, "test-tenant")()

	body := `[{"username":"u1","password":"123456","role":"user"}]`
	req := newAdminPostJSON("/admin/batch-create", body)
	w := httptest.NewRecorder()
	HandleBatchCreateUser(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("OneID 模式 /admin/batch-create 期望 409，实际 %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "OneID 模式下不允许") {
		t.Errorf("错误文案缺关键词: %s", w.Body.String())
	}
	var n int64
	model.DB(context.Background()).Model(&model.User{}).Where("username = ?", "u1").Count(&n)
	if n != 0 {
		t.Errorf("OneID 模式下批量用户不应被创建，实际 %d", n)
	}
}

// TestHandleUpdateUser_GroupIDsContainOneIDDept_FiltersSilently
// 传入的 group_ids 中含 source=oneid_dept 的组 → 静默忽略该项，不报错；
// 用户已有的 oneid_dept membership 必须保留，manual 子集按传入 ids 替换。
//
// 这个测试覆盖产品需求里的核心用例：
//
//	用户 a ∈ {A(manual), B(oneid_dept)}，调用 update-user group_ids=[A,B,C]
//	→ 用户 a ∈ {A, B, C}（B 静默过滤；manual 子集替换为 {A,C}；B 保持）
func TestHandleUpdateUser_GroupIDsContainOneIDDept_FiltersSilently(t *testing.T) {
	defer initOneIDUserOpsTestDB(t, "test-tenant")()

	user := model.User{Username: "alice", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	gA := model.UserGroup{Name: "A", Source: model.GroupSourceManual}
	gC := model.UserGroup{Name: "C", Source: model.GroupSourceManual}
	for _, g := range []*model.UserGroup{&gA, &gC} {
		if err := model.DB(context.Background()).Create(g).Error; err != nil {
			t.Fatal(err)
		}
	}
	gB := model.UserGroup{Name: "B", Source: model.GroupSourceOneIDDept, SourceRef: "D-B"}
	if err := model.DB(context.Background()).Create(&gB).Error; err != nil {
		t.Fatal(err)
	}

	// 初态：a ∈ {A(manual), B(oneid_dept)}
	if err := model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: gA.ID, UserID: user.ID, Source: model.MemberSourceManual,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: gB.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept,
	}).Error; err != nil {
		t.Fatal(err)
	}

	// 调用：group_ids=[A, B, C] → B 应被静默过滤
	body := `{"group_ids":[` + uintToStr(gA.ID) + `,` + uintToStr(gB.ID) + `,` + uintToStr(gC.ID) + `]}`
	req := newAdminPostJSON("/admin/update-user?id="+uintToStr(user.ID), body)
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("含 oneid_dept 应静默过滤而不报错，期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}

	// a ∈ {A, B, C}：A/C 是 manual，B 是 oneid_dept
	var n int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ? AND source = ?", user.ID, gA.ID, model.MemberSourceManual).
		Count(&n)
	if n != 1 {
		t.Errorf("A 应在 manual 行，实际 %d", n)
	}
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ? AND source = ?", user.ID, gC.ID, model.MemberSourceManual).
		Count(&n)
	if n != 1 {
		t.Errorf("C 应在 manual 行，实际 %d", n)
	}
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ? AND source = ?", user.ID, gB.ID, model.MemberSourceOneIDDept).
		Count(&n)
	if n != 1 {
		t.Errorf("B (oneid_dept) 应保留，实际 %d", n)
	}
	// B 不应被错写成 manual
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ? AND source = ?", user.ID, gB.ID, model.MemberSourceManual).
		Count(&n)
	if n != 0 {
		t.Errorf("B 不应被写为 manual，实际 %d", n)
	}
}

// TestHandleUpdateUser_GroupIDsAllManual_KeepsOneIDDeptMembership
// 传入 group_ids 全是 manual → 替换 manual 子集；oneid_dept 行保持。
//
// 用例对应需求：用户 a ∈ {A(manual), B(oneid_dept)}，传 [C] → a ∈ {B, C}
func TestHandleUpdateUser_GroupIDsAllManual_KeepsOneIDDeptMembership(t *testing.T) {
	defer initOneIDUserOpsTestDB(t, "test-tenant")()

	user := model.User{Username: "bob", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	mg1 := model.UserGroup{Name: "M1", Source: model.GroupSourceManual}
	mg2 := model.UserGroup{Name: "M2", Source: model.GroupSourceManual}
	mg3 := model.UserGroup{Name: "M3", Source: model.GroupSourceManual}
	for _, g := range []*model.UserGroup{&mg1, &mg2, &mg3} {
		if err := model.DB(context.Background()).Create(g).Error; err != nil {
			t.Fatal(err)
		}
	}
	oneidGroup := model.UserGroup{Name: "Dept", Source: model.GroupSourceOneIDDept, SourceRef: "D002"}
	if err := model.DB(context.Background()).Create(&oneidGroup).Error; err != nil {
		t.Fatal(err)
	}

	// 用户初态：mg1 (manual)、oneidGroup (oneid_dept)
	if err := model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: mg1.ID, UserID: user.ID, Source: model.MemberSourceManual,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: oneidGroup.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept,
	}).Error; err != nil {
		t.Fatal(err)
	}

	// 操作：把 manual 归属换成 mg2 + mg3
	body := `{"group_ids":[` + uintToStr(mg2.ID) + "," + uintToStr(mg3.ID) + `]}`
	req := newAdminPostJSON("/admin/update-user?id="+uintToStr(user.ID), body)
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("全 manual 期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}
	// mg1 manual 行应被清掉
	var n int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ?", user.ID, mg1.ID).
		Count(&n)
	if n != 0 {
		t.Errorf("旧 manual mg1 应被清空，剩余 %d", n)
	}
	// mg2/mg3 应被写入
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND user_group_id IN ? AND source = ?",
			user.ID, []uint{mg2.ID, mg3.ID}, model.MemberSourceManual).
		Count(&n)
	if n != 2 {
		t.Errorf("新 manual mg2/mg3 应写入 2 行，实际 %d", n)
	}
	// oneidGroup membership 必须保留
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ? AND source = ?",
			user.ID, oneidGroup.ID, model.MemberSourceOneIDDept).
		Count(&n)
	if n != 1 {
		t.Errorf("oneid_dept membership 不应被破坏，实际 %d", n)
	}
}

// TestHandleUpdateUser_OneIDMode_EmptyGroupIDs_ClearsManualKeepsOneIDDept
// OneID 模式下，传 group_ids=[] → 清空所有 manual 行；oneid_dept 行保留。
func TestHandleUpdateUser_OneIDMode_EmptyGroupIDs_ClearsManualKeepsOneIDDept(t *testing.T) {
	defer initOneIDUserOpsTestDB(t, "test-tenant")()

	user := model.User{Username: "carol", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	mg := model.UserGroup{Name: "M", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&mg).Error; err != nil {
		t.Fatal(err)
	}
	oneidGroup := model.UserGroup{Name: "Dept", Source: model.GroupSourceOneIDDept, SourceRef: "D003"}
	if err := model.DB(context.Background()).Create(&oneidGroup).Error; err != nil {
		t.Fatal(err)
	}
	// 用户初态：1 个 manual + 1 个 oneid_dept
	if err := model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: mg.ID, UserID: user.ID, Source: model.MemberSourceManual,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: oneidGroup.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept,
	}).Error; err != nil {
		t.Fatal(err)
	}

	body := `{"group_ids":[]}`
	req := newAdminPostJSON("/admin/update-user?id="+uintToStr(user.ID), body)
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("空 group_ids 期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}
	var n int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND source = ?", user.ID, model.MemberSourceManual).Count(&n)
	if n != 0 {
		t.Errorf("manual 行应被清空，实际 %d", n)
	}
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND source = ?", user.ID, model.MemberSourceOneIDDept).Count(&n)
	if n != 1 {
		t.Errorf("oneid_dept 行应保留，实际 %d", n)
	}
}

// TestHandleUpdateUser_StandardMode_AllManual
// 标准模式（TenantID 空）下，传入的 group_ids 全是 manual 时正常生效。
// 行为与 OneID 模式一致：update-user 永远走 manual-only 替换。
func TestHandleUpdateUser_StandardMode_AllManual(t *testing.T) {
	defer initOneIDUserOpsTestDB(t, "")() // 标准模式

	user := model.User{Username: "cc", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	mg := model.UserGroup{Name: "M", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&mg).Error; err != nil {
		t.Fatal(err)
	}

	body := `{"group_ids":[` + uintToStr(mg.ID) + `]}`
	req := newAdminPostJSON("/admin/update-user?id="+uintToStr(user.ID), body)
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("标准模式期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}
	var n int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ?", user.ID, mg.ID).Count(&n)
	if n != 1 {
		t.Errorf("标准模式应写入 manual 行，实际 %d", n)
	}
}

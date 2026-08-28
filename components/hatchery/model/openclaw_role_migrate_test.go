package model

import (
	"context"
	"sync"
	"testing"

	hcommon "hatchery/common"

	"gorm.io/gorm"
)

// resetDefaultRolesCache 重置 default_roles.json 的解析缓存，
// 避免测试间相互污染（与 seed_coverage_test.go 同步策略）。
func resetDefaultRolesCache() {
	defaultRolesOnce = sync.Once{}
	defaultRolesJSON = nil
}

// findRoleByName 在测试中按 name 查询角色（统一封装，避免重复 boilerplate）。
func findRoleByName(t *testing.T, name string) (OpenClawRole, bool) {
	t.Helper()
	var r OpenClawRole
	err := DB(context.Background()).Where("name = ?", name).First(&r).Error
	if err == gorm.ErrRecordNotFound {
		return OpenClawRole{}, false
	}
	if err != nil {
		t.Fatalf("query role %s: %v", name, err)
	}
	return r, true
}

// TestMigrateRenamedDefaultRoles_Rename 验证正常迁移路径：
//   - 老库存在「开发工程师」、无「程序员」
//   - 迁移后：name 改成「程序员」；description/soul 刷成 default_roles.json 中最新值
//   - ID / SortOrder / Visible / VisibilityType / CreatedAt 全部保留
func TestMigrateRenamedDefaultRoles_Rename(t *testing.T) {
	setupSeedMigrateTestDB(t)

	// 注入精简 fixture：只声明「程序员」，让 getDefaultRoles() 能匹配到新名字
	const newDesc = "FIXTURE_NEW_DESC"
	const newSoul = "FIXTURE_NEW_SOUL"
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		resetDefaultRolesCache()
	}()
	DefaultRolesJSON = []byte(`[
		{"name":"程序员","name_en":"程序员","description":"` + newDesc + `","description_en":"` + newDesc + `","soul":"` + newSoul + `","soul_en":"` + newSoul + `","visible":true,"skills":[]}
	]`)
	resetDefaultRolesCache()

	db := DB(context.Background())

	// 准备老角色：name=「开发工程师」，description/soul/sort_order 都是老值
	old := OpenClawRole{
		Name:           "开发工程师",
		Description:    "OLD_DESC",
		Soul:           "OLD_SOUL",
		Visible:        true,
		SortOrder:      7,
		VisibilityType: "group",
	}
	if err := db.Create(&old).Error; err != nil {
		t.Fatalf("create old role: %v", err)
	}
	oldID := old.ID
	oldCreatedAt := old.CreatedAt

	if err := MigrateRenamedDefaultRoles(context.Background(), db); err != nil {
		t.Fatalf("MigrateRenamedDefaultRoles err: %v", err)
	}

	// 老名应已不存在
	if _, ok := findRoleByName(t, "开发工程师"); ok {
		t.Error("expect 「开发工程师」 to be renamed away, but still exists")
	}

	// 新名应存在
	got, ok := findRoleByName(t, "程序员")
	if !ok {
		t.Fatal("expect 「程序员」 to exist after migration")
	}

	// ID / 元数据保留
	if got.ID != oldID {
		t.Errorf("ID changed: got=%d want=%d", got.ID, oldID)
	}
	if !got.CreatedAt.Equal(oldCreatedAt) {
		t.Errorf("CreatedAt changed: got=%v want=%v", got.CreatedAt, oldCreatedAt)
	}
	if got.SortOrder != 7 {
		t.Errorf("SortOrder changed: got=%d want=7", got.SortOrder)
	}
	if !got.Visible {
		t.Error("Visible changed: got=false want=true")
	}
	if got.VisibilityType != "group" {
		t.Errorf("VisibilityType changed: got=%s want=group", got.VisibilityType)
	}

	// description/soul 应该已被刷成 fixture 中的新值
	if got.Description != newDesc {
		t.Errorf("Description not refreshed: got=%q want=%q", got.Description, newDesc)
	}
	if got.Soul != newSoul {
		t.Errorf("Soul not refreshed: got=%q want=%q", got.Soul, newSoul)
	}
}

// TestMigrateRenamedDefaultRoles_Idempotent 验证幂等：
//   - 已存在「程序员」时，不再触发 rename
//   - 老的「开发工程师」如果同时存在，应保留原状（不会被错误删除/覆盖）
func TestMigrateRenamedDefaultRoles_Idempotent(t *testing.T) {
	setupSeedMigrateTestDB(t)
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		resetDefaultRolesCache()
	}()
	// fixture 必须声明「程序员」，否则迁移函数会因找不到新角色直接 continue，
	// 幂等场景（newCnt > 0 跳过）就测不到了
	DefaultRolesJSON = []byte(`[
		{"name":"程序员","name_en":"Programmer","description":"D","description_en":"D","soul":"S","soul_en":"S","visible":true,"skills":[]}
	]`)
	resetDefaultRolesCache()

	db := DB(context.Background())

	// 模拟"已迁移过"的状态：DB 里只有「程序员」
	already := OpenClawRole{
		Name:           "程序员",
		Description:    "ALREADY_DESC",
		Soul:           "ALREADY_SOUL",
		Visible:        true,
		SortOrder:      1,
		VisibilityType: "all",
	}
	if err := db.Create(&already).Error; err != nil {
		t.Fatalf("create already-migrated role: %v", err)
	}

	if err := MigrateRenamedDefaultRoles(context.Background(), db); err != nil {
		t.Fatalf("MigrateRenamedDefaultRoles err: %v", err)
	}

	got, ok := findRoleByName(t, "程序员")
	if !ok {
		t.Fatal("「程序员」 should still exist after idempotent run")
	}
	// 幂等路径不应刷字段
	if got.Description != "ALREADY_DESC" {
		t.Errorf("idempotent run should NOT overwrite Description, got=%q", got.Description)
	}
	if got.Soul != "ALREADY_SOUL" {
		t.Errorf("idempotent run should NOT overwrite Soul, got=%q", got.Soul)
	}

	// 再跑一次，仍然安全
	if err := MigrateRenamedDefaultRoles(context.Background(), db); err != nil {
		t.Fatalf("second MigrateRenamedDefaultRoles err: %v", err)
	}
}

// TestMigrateRenamedDefaultRoles_NoOldRole 验证全新部署路径：
//   - DB 中既没有「开发工程师」也没有「程序员」
//   - 函数应安全 no-op，不报错、不创建脏数据
func TestMigrateRenamedDefaultRoles_NoOldRole(t *testing.T) {
	setupSeedMigrateTestDB(t)
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		resetDefaultRolesCache()
	}()
	DefaultRolesJSON = []byte(`[
		{"name":"程序员","name_en":"Programmer","description":"D","description_en":"D","soul":"S","soul_en":"S","visible":true,"skills":[]}
	]`)
	resetDefaultRolesCache()

	db := DB(context.Background())

	if err := MigrateRenamedDefaultRoles(context.Background(), db); err != nil {
		t.Fatalf("MigrateRenamedDefaultRoles on empty DB err: %v", err)
	}

	if _, ok := findRoleByName(t, "程序员"); ok {
		t.Error("MigrateRenamedDefaultRoles should not create new role when old role missing")
	}
	if _, ok := findRoleByName(t, "开发工程师"); ok {
		t.Error("unexpected 「开发工程师」 created out of nowhere")
	}
}

// TestMigrateRenamedDefaultRoles_CoexistOldAndNew 验证唯一索引冲突保护：
//   - 极端场景：DB 中同时存在「开发工程师」和「程序员」
//     （例如管理员手动建过同名自定义角色）
//   - 期望：迁移函数检测到「程序员」已存在 → 跳过，避免 UNIQUE 冲突
//   - 双方记录都应原样保留，不破坏数据
func TestMigrateRenamedDefaultRoles_CoexistOldAndNew(t *testing.T) {
	setupSeedMigrateTestDB(t)
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		resetDefaultRolesCache()
	}()
	DefaultRolesJSON = []byte(`[
		{"name":"程序员","name_en":"Programmer","description":"D","description_en":"D","soul":"S","soul_en":"S","visible":true,"skills":[]}
	]`)
	resetDefaultRolesCache()

	// 显式注入 zh ctx，明确这是国内站场景
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{DefaultLang: "zh"})
	db := DB(ctx)

	if err := db.Create(&OpenClawRole{
		Name: "开发工程师", Description: "OLD_DESC", Soul: "OLD_SOUL",
		Visible: true, SortOrder: 7, VisibilityType: "all",
	}).Error; err != nil {
		t.Fatalf("create old role: %v", err)
	}
	if err := db.Create(&OpenClawRole{
		Name: "程序员", Description: "USER_CUSTOM", Soul: "USER_CUSTOM_SOUL",
		Visible: true, SortOrder: 99, VisibilityType: "all",
	}).Error; err != nil {
		t.Fatalf("create user-custom role: %v", err)
	}

	if err := MigrateRenamedDefaultRoles(ctx, db); err != nil {
		t.Fatalf("MigrateRenamedDefaultRoles err: %v", err)
	}

	// 两条记录都应保留原状
	oldRole, ok := findRoleByName(t, "开发工程师")
	if !ok {
		t.Fatal("「开发工程师」 should be preserved when 「程序员」 already exists")
	}
	if oldRole.Description != "OLD_DESC" {
		t.Errorf("preserved old role mutated: desc=%q", oldRole.Description)
	}

	newRole, ok := findRoleByName(t, "程序员")
	if !ok {
		t.Fatal("「程序员」 should still exist")
	}
	if newRole.Description != "USER_CUSTOM" || newRole.SortOrder != 99 {
		t.Errorf("user-custom 「程序员」 was mutated: desc=%q sort=%d",
			newRole.Description, newRole.SortOrder)
	}
}

// TestMigrateRenamedDefaultRoles_CountErrorBranch 覆盖 Count 查询出错的防御分支：
//   - 关闭底层 *sql.DB，让 `Where("name = ?", newName).Count(...)` 必然返回错误
//   - 期望：函数走 slog.Error 后 continue，整体仍返回 nil（不应把单条 rename 的 DB 错误抛给调用方）
func TestMigrateRenamedDefaultRoles_CountErrorBranch(t *testing.T) {
	setupSeedMigrateTestDB(t)
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		resetDefaultRolesCache()
	}()
	DefaultRolesJSON = []byte(`[
		{"name":"程序员","name_en":"Programmer","description":"D","description_en":"D","soul":"S","soul_en":"S","visible":true,"skills":[]}
	]`)
	resetDefaultRolesCache()

	db := DB(context.Background())
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	// 关闭底层连接，后续任何查询都会返回 "sql: database is closed"
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	if err := MigrateRenamedDefaultRoles(context.Background(), db); err != nil {
		t.Fatalf("MigrateRenamedDefaultRoles should swallow per-rename DB error, got: %v", err)
	}
}

// TestMigrateRenamedDefaultRoles_UpdatesErrorBranch 覆盖 Updates 出错的防御分支：
//   - 先正常创建老角色「开发工程师」让前两步 Count / First 全部通过
//   - 然后关闭底层 *sql.DB，让 `tx.Model(&old).Updates(...)` 必然失败
//   - 期望：函数走 slog.Error 后 continue，整体仍返回 nil
func TestMigrateRenamedDefaultRoles_UpdatesErrorBranch(t *testing.T) {
	setupSeedMigrateTestDB(t)
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		resetDefaultRolesCache()
	}()
	DefaultRolesJSON = []byte(`[
		{"name":"程序员","name_en":"程序员","description":"D","description_en":"D","soul":"S","soul_en":"S","visible":true,"skills":[]}
	]`)
	resetDefaultRolesCache()

	db := DB(context.Background())

	// 先把老角色塞进去，保证 Count==0（新名不存在）且 First 能找到老记录
	if err := db.Create(&OpenClawRole{
		Name: "开发工程师", Description: "OLD", Soul: "OLD",
		Visible: true, SortOrder: 1, VisibilityType: "all",
	}).Error; err != nil {
		t.Fatalf("create old role: %v", err)
	}

	// 关闭底层连接，让接下来的 Count（返回 0/err）和 Updates 必然失败
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	if err := MigrateRenamedDefaultRoles(context.Background(), db); err != nil {
		t.Fatalf("MigrateRenamedDefaultRoles should swallow per-rename DB error, got: %v", err)
	}
}

// TestMigrateRenamedDefaultRoles_OverseasRename 验证国际站（DefaultLang=en）路径：
//   - 老库存在英文老名「Software Engineer」，无新名「Programmer」
//   - 迁移后：name 改成「Programmer」，description/soul 刷成 fixture 中的英文最新值
//   - ID / SortOrder 等元数据保留
//
// 该用例同时覆盖 `defaultLang == "zh"` 之外的英文分支，避免国际站环境出现"老英文名永远不会被迁"。
func TestMigrateRenamedDefaultRoles_OverseasRename(t *testing.T) {
	setupSeedMigrateTestDB(t)

	const newDescEn = "FIXTURE_NEW_DESC_EN"
	const newSoulEn = "FIXTURE_NEW_SOUL_EN"
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		resetDefaultRolesCache()
	}()
	DefaultRolesJSON = []byte(`[
		{"name":"程序员","name_en":"Programmer","description":"无关","description_en":"` + newDescEn + `","soul":"无关","soul_en":"` + newSoulEn + `","visible":true,"skills":[]}
	]`)
	resetDefaultRolesCache()

	// 注入 DefaultLang=en 的 ctx，模拟国际站环境
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Identifier: "", DefaultLang: "en"})
	db := DB(ctx)

	old := OpenClawRole{
		Name:           "Software Engineer",
		Description:    "OLD_EN",
		Soul:           "OLD_EN_SOUL",
		Visible:        true,
		SortOrder:      5,
		VisibilityType: "all",
	}
	if err := db.Create(&old).Error; err != nil {
		t.Fatalf("create old en role: %v", err)
	}
	oldID := old.ID

	if err := MigrateRenamedDefaultRoles(ctx, db); err != nil {
		t.Fatalf("MigrateRenamedDefaultRoles err: %v", err)
	}

	if _, ok := findRoleByName(t, "Software Engineer"); ok {
		t.Error("expect 「Software Engineer」 to be renamed away")
	}
	got, ok := findRoleByName(t, "Programmer")
	if !ok {
		t.Fatal("expect 「Programmer」 to exist after migration")
	}
	if got.ID != oldID {
		t.Errorf("ID changed: got=%d want=%d", got.ID, oldID)
	}
	if got.Description != newDescEn {
		t.Errorf("Description not refreshed (en): got=%q want=%q", got.Description, newDescEn)
	}
	if got.Soul != newSoulEn {
		t.Errorf("Soul not refreshed (en): got=%q want=%q", got.Soul, newSoulEn)
	}
	if got.SortOrder != 5 {
		t.Errorf("SortOrder changed: got=%d want=5", got.SortOrder)
	}
}

// TestMigrateRenamedDefaultRoles_NewRoleMissingInJSON 覆盖防御分支：
//   - default_roles.json 中找不到 newKey 对应的角色（fixture 故意不含「程序员」）
//   - 期望：函数走 slog.Warn 后 continue，不应触碰任何 DB 数据
func TestMigrateRenamedDefaultRoles_NewRoleMissingInJSON(t *testing.T) {
	setupSeedMigrateTestDB(t)
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		resetDefaultRolesCache()
	}()
	// fixture 故意不含「程序员」，走 newName == "" 的早退分支
	DefaultRolesJSON = []byte(`[
		{"name":"行业分析师","name_en":"Industry Analyst","description":"D","description_en":"D","soul":"S","soul_en":"S","visible":true,"skills":[]}
	]`)
	resetDefaultRolesCache()

	db := DB(context.Background())

	// 即使 DB 里有老角色，也不应被处理
	if err := db.Create(&OpenClawRole{
		Name: "开发工程师", Description: "OLD", Soul: "OLD",
		Visible: true, SortOrder: 1, VisibilityType: "all",
	}).Error; err != nil {
		t.Fatalf("create old role: %v", err)
	}

	if err := MigrateRenamedDefaultRoles(context.Background(), db); err != nil {
		t.Fatalf("MigrateRenamedDefaultRoles err: %v", err)
	}

	// 老角色应保持原状
	got, ok := findRoleByName(t, "开发工程师")
	if !ok {
		t.Fatal("「开发工程师」 should remain untouched when newKey missing in JSON")
	}
	if got.Description != "OLD" || got.Soul != "OLD" {
		t.Errorf("old role should not be mutated, got desc=%q soul=%q", got.Description, got.Soul)
	}
}

package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestInheritSkillDistributeCount 验证上传新版本时继承旧版本的 distribute_count
func TestInheritSkillDistributeCount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&Skill{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建旧版本，带 distribute_count=50
	oldSkill := Skill{
		Slug: "inherit-test", Name: "继承测试", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", DistributeCount: 50,
	}
	db.Create(&oldSkill)

	// 创建新版本
	newSkill := Skill{
		Slug: "inherit-test", Name: "继承测试", Version: "2.0.0",
		VersionMajor: 2, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	tx := db.Begin()
	tx.Create(&newSkill)

	// 调用继承函数
	if err := InheritSkillDistributeCount(tx, "inherit-test", newSkill.ID); err != nil {
		t.Fatalf("InheritSkillDistributeCount 返回错误: %v", err)
	}
	tx.Commit()

	// 验证新版本继承了旧版本的 distribute_count
	var result Skill
	db.First(&result, newSkill.ID)
	if result.DistributeCount != 50 {
		t.Errorf("新版本 distribute_count 应为 50，实际=%d", result.DistributeCount)
	}
}

// TestInheritSkillDistributeCount_NoPrevVersion 验证首次上传（无旧版本）时返回 nil
func TestInheritSkillDistributeCount_NoPrevVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&Skill{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 直接创建第一个版本（无旧版本）
	newSkill := Skill{
		Slug: "first-version", Name: "首次上传", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	tx := db.Begin()
	tx.Create(&newSkill)

	// 调用继承函数——应返回 nil
	if err := InheritSkillDistributeCount(tx, "first-version", newSkill.ID); err != nil {
		t.Fatalf("首次上传不应报错: %v", err)
	}
	tx.Commit()

	// 验证 distribute_count 保持为 0
	var result Skill
	db.First(&result, newSkill.ID)
	if result.DistributeCount != 0 {
		t.Errorf("首次上传 distribute_count 应为 0，实际=%d", result.DistributeCount)
	}
}

// TestInheritSkillDistributeCount_ZeroCount 验证旧版本 distribute_count=0 时不执行 Update
func TestInheritSkillDistributeCount_ZeroCount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&Skill{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 旧版本 distribute_count=0
	oldSkill := Skill{
		Slug: "zero-count", Name: "零下载", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", DistributeCount: 0,
	}
	db.Create(&oldSkill)

	newSkill := Skill{
		Slug: "zero-count", Name: "零下载", Version: "2.0.0",
		VersionMajor: 2, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	tx := db.Begin()
	tx.Create(&newSkill)

	if err := InheritSkillDistributeCount(tx, "zero-count", newSkill.ID); err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	tx.Commit()

	var result Skill
	db.First(&result, newSkill.ID)
	if result.DistributeCount != 0 {
		t.Errorf("旧版本为 0 时新版本也应为 0，实际=%d", result.DistributeCount)
	}
}

// TestInheritSkillDistributeCount_UpdateError 验证查询失败时返回 error
func TestInheritSkillDistributeCount_UpdateError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&Skill{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 使用已回滚的事务，让 First 查询报非 RecordNotFound 的错误
	tx := db.Begin()
	tx.Rollback() // 回滚后事务不可用

	err = InheritSkillDistributeCount(tx, "any-slug", 999)
	if err == nil {
		t.Error("期望返回错误，但得到 nil")
	}
}

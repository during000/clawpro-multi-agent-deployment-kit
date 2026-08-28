package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDistributedSkillStateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Skill{}, &SkillDistributionTask{}, &SkillDistributionRecord{}); err != nil {
		t.Fatalf("migrate skill distribution tables: %v", err)
	}
	restore := UseDBForTestWithDriver(db, "sqlite")
	t.Cleanup(restore)
	return db
}

func appendSkillDistributionEvent(t *testing.T, db *gorm.DB, instanceID uint, source, slug, version, action, status string, skillID uint) {
	t.Helper()
	task := SkillDistributionTask{
		SkillID: skillID, Source: source, Slug: slug, Version: version,
		Type: action, Status: TaskStatusCompleted,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	record := SkillDistributionRecord{
		TaskID: task.ID, SkillID: skillID, InstanceID: instanceID,
		Version: version, Type: action, Status: status,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
}

func TestListDistributedSkillStates_FoldsSuccessfulPhysicalEvents(t *testing.T) {
	db := setupDistributedSkillStateTestDB(t)

	enterprise := Skill{Slug: "enterprise", Name: "Enterprise", Version: "1.0.0"}
	historical := Skill{Slug: "historical", Name: "Historical", Version: "3.0.0"}
	if err := db.Create(&[]Skill{enterprise, historical}).Error; err != nil {
		t.Fatalf("create skills: %v", err)
	}
	if err := db.Where("slug = ?", enterprise.Slug).First(&enterprise).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("slug = ?", historical.Slug).First(&historical).Error; err != nil {
		t.Fatal(err)
	}

	appendSkillDistributionEvent(t, db, 1, SkillSourceEnterprise, "enterprise", "1.0.0", TaskTypeDistribute, RecordStatusSuccess, enterprise.ID)

	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "removed", "1.0.0", TaskTypeDistribute, RecordStatusSuccess, 0)
	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "removed", "1.0.0", TaskTypeUninstall, RecordStatusSuccess, 0)

	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "stable", "1.0.0", TaskTypeDistribute, RecordStatusSuccess, 0)
	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "stable", "2.0.0", TaskTypeDistribute, RecordStatusPending, 0)
	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "stable", "2.0.0", TaskTypeDistribute, RecordStatusFailed, 0)
	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "stable", "2.0.0", TaskTypeDistribute, RecordStatusUpgradeFailed, 0)

	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "cross-source", "1.0.0", TaskTypeDistribute, RecordStatusSuccess, 0)
	appendSkillDistributionEvent(t, db, 1, SkillSourceEnterprise, "cross-source", "2.0.0", TaskTypeDistribute, RecordStatusSuccess, enterprise.ID)
	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "cross-source", "2.0.0", TaskTypeUninstall, RecordStatusSuccess, 0)

	appendSkillDistributionEvent(t, db, 1, SkillSourceEnterprise, "", "3.0.0", TaskTypeDistribute, RecordStatusSuccess, historical.ID)

	states, err := ListDistributedSkillStates(context.Background(), 1, []string{"enterprise", "removed", "stable", "cross-source", "historical"})
	if err != nil {
		t.Fatalf("ListDistributedSkillStates: %v", err)
	}

	if got := states["enterprise"]; !got.Installed || got.Source != SkillSourceEnterprise || got.Version != "1.0.0" || got.SkillID != enterprise.ID {
		t.Fatalf("enterprise state = %+v", got)
	}
	if got := states["removed"]; got.Installed || got.Version != "1.0.0" {
		t.Fatalf("removed state = %+v", got)
	}
	if got := states["stable"]; !got.Installed || got.Version != "1.0.0" {
		t.Fatalf("stable state = %+v", got)
	}
	if got := states["cross-source"]; got.Installed || got.Source != SkillSourcePublic {
		t.Fatalf("cross-source state = %+v", got)
	}
	if got := states["historical"]; !got.Installed || got.Version != "3.0.0" || got.SkillID != historical.ID {
		t.Fatalf("historical state = %+v", got)
	}
}

func TestListDistributedSkillStates_FiltersInstanceAndCandidateSlugs(t *testing.T) {
	db := setupDistributedSkillStateTestDB(t)
	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "wanted", "1.0.0", TaskTypeDistribute, RecordStatusSuccess, 0)
	appendSkillDistributionEvent(t, db, 1, SkillSourcePublic, "other-slug", "2.0.0", TaskTypeDistribute, RecordStatusSuccess, 0)
	appendSkillDistributionEvent(t, db, 2, SkillSourcePublic, "wanted", "3.0.0", TaskTypeDistribute, RecordStatusSuccess, 0)

	states, err := ListDistributedSkillStates(context.Background(), 1, []string{"wanted"})
	if err != nil {
		t.Fatalf("ListDistributedSkillStates: %v", err)
	}
	if len(states) != 1 || states["wanted"].Version != "1.0.0" {
		t.Fatalf("states = %+v", states)
	}
}

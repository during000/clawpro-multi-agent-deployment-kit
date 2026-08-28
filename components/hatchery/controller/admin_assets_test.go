package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type assetBindingQueryRecorder struct {
	logger.Interface
	queries []string
}

func (r *assetBindingQueryRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	r.queries = append(r.queries, sql)
}

func TestLoadCurrentAssetBindingsFiltersNonAssetConfig(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.ProjectConfigBinding{}, &model.GroupConfigBinding{}); err != nil {
		t.Fatalf("migrate bindings: %v", err)
	}
	if err := db.Create([]model.ProjectConfigBinding{
		{ProjectID: 1, ConfigType: model.AssetBindingTypeSkill, ConfigKey: "skill-a"},
		{ProjectID: 1, ConfigType: model.AssetBindingTypeRule, ConfigKey: "rule-a"},
		{ProjectID: 1, ConfigType: model.ProjectConfigTypeSkill, ConfigKey: "visible-skill"},
	}).Error; err != nil {
		t.Fatalf("create project bindings: %v", err)
	}
	if err := db.Create([]model.GroupConfigBinding{
		{GroupID: 1, ConfigType: model.AssetBindingTypeSkill, ConfigKey: "skill-b"},
		{GroupID: 1, ConfigType: model.AssetBindingTypeRule, ConfigKey: "rule-b"},
		{GroupID: 1, ConfigType: model.ConfigTypeChannel, ConfigKey: "channel"},
	}).Error; err != nil {
		t.Fatalf("create group bindings: %v", err)
	}

	for _, target := range []assetTarget{{typeName: assetTargetProject, id: 1}, {typeName: assetTargetGroup, id: 1}} {
		recorder := &assetBindingQueryRecorder{Interface: logger.Default.LogMode(logger.Silent)}
		items, err := loadCurrentAssetBindings(db.Session(&gorm.Session{Logger: recorder}), target)
		if err != nil {
			t.Fatalf("load %s bindings: %v", target.typeName, err)
		}
		if len(items) != 2 {
			t.Fatalf("%s should return only skill/rule assets, got %#v", target.typeName, items)
		}
		if len(recorder.queries) != 1 || !strings.Contains(recorder.queries[0], "config_type IN") {
			t.Fatalf("%s query should filter config_type, queries=%v", target.typeName, recorder.queries)
		}
	}
}

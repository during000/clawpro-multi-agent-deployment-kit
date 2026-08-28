package model

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var pluginCategoryTestDBCounter int64

// setupPluginCategoryTestDB creates an isolated SQLite memory database for testing.
func setupPluginCategoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	id := atomic.AddInt64(&pluginCategoryTestDBCounter, 1)
	dsn := fmt.Sprintf("file:pluginCategoryTest%d?mode=memory&cache=shared", id)
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite mem: %v", err)
	}
	if err := gdb.AutoMigrate(&PluginCategory{}, &PluginCategoryMapping{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return gdb
}

// TestSeedPluginCategories_CreatesPredefinedCategories verifies that SeedPluginCategories creates predefined categories.
func TestSeedPluginCategories_CreatesPredefinedCategories(t *testing.T) {
	gdb := setupPluginCategoryTestDB(t)

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SeedPluginCategories(tx)
	})
	if err != nil {
		t.Fatalf("SeedPluginCategories: %v", err)
	}

	var count int64
	if err := gdb.Model(&PluginCategory{}).Count(&count).Error; err != nil {
		t.Fatalf("count categories: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected categories to be created, got count %d", count)
	}
	if count != int64(len(predefinedPluginCategories)) {
		t.Fatalf("expected %d categories, got %d", len(predefinedPluginCategories), count)
	}
}

// TestSeedPluginCategories_IdempotentCall verifies that calling SeedPluginCategories twice doesn't create duplicates.
func TestSeedPluginCategories_IdempotentCall(t *testing.T) {
	gdb := setupPluginCategoryTestDB(t)

	// First call
	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SeedPluginCategories(tx)
	})
	if err != nil {
		t.Fatalf("SeedPluginCategories first call: %v", err)
	}

	var count1 int64
	if err := gdb.Model(&PluginCategory{}).Count(&count1).Error; err != nil {
		t.Fatalf("count first call: %v", err)
	}

	// Second call should not create duplicates
	err = gdb.Transaction(func(tx *gorm.DB) error {
		return SeedPluginCategories(tx)
	})
	if err != nil {
		t.Fatalf("SeedPluginCategories second call: %v", err)
	}

	var count2 int64
	if err := gdb.Model(&PluginCategory{}).Count(&count2).Error; err != nil {
		t.Fatalf("count second call: %v", err)
	}
	if count1 != count2 {
		t.Fatalf("expected same count after second call, first=%d second=%d", count1, count2)
	}
}

// TestSeedPluginCategories_WithClosedDatabase verifies error handling when database operations fail.
func TestSeedPluginCategories_WithClosedDatabase(t *testing.T) {
	gdb := setupPluginCategoryTestDB(t)

	// Get the underlying sqlDB and close it to simulate database error
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("get sqlDB: %v", err)
	}
	sqlDB.Close()

	// Now attempt to seed categories on the closed database
	// This should fail during the find or create operation
	err = gdb.Transaction(func(tx *gorm.DB) error {
		return SeedPluginCategories(tx)
	})
	if err == nil {
		t.Fatalf("expected error with closed database, got nil")
	}
}

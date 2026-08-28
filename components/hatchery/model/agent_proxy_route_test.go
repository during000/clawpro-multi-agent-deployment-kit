package model

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func initAgentProxyRouteModelTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AgentProxyRoute{}, &Instance{}, &Notification{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return UseDBForTest(db)
}

func TestDisableAgentProxyRoutesForInstance(t *testing.T) {
	cleanup := initAgentProxyRouteModelTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := DisableAgentProxyRoutesForInstance(ctx, ""); err != nil {
		t.Fatalf("empty instance id should be noop: %v", err)
	}
	routes := []AgentProxyRoute{
		{RouteID: "r1", InstanceID: "ins-1", Kind: AgentProxyRouteKindTeams, Enabled: true},
		{RouteID: "r2", InstanceID: "ins-2", Kind: AgentProxyRouteKindTeams, Enabled: true},
	}
	if err := DB(ctx).Create(&routes).Error; err != nil {
		t.Fatalf("create routes: %v", err)
	}
	if err := DisableAgentProxyRoutesForInstance(ctx, "ins-1"); err != nil {
		t.Fatalf("disable instance routes: %v", err)
	}
	var r1, r2 AgentProxyRoute
	DB(ctx).Where("route_id = ?", "r1").First(&r1)
	DB(ctx).Where("route_id = ?", "r2").First(&r2)
	if r1.Enabled || !r2.Enabled {
		t.Fatalf("unexpected enabled states: r1=%v r2=%v", r1.Enabled, r2.Enabled)
	}
}

func TestCleanupDestroyedInstances_DisablesAgentProxyRoutes(t *testing.T) {
	cleanup := initAgentProxyRouteModelTestDB(t)
	defer cleanup()
	ctx := context.Background()

	inst := &Instance{Name: "stale", InstanceId: "ins-stale", UserID: 1}
	if err := DB(ctx).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := DB(ctx).Model(inst).Updates(map[string]interface{}{"updated_at": old}).Error; err != nil {
		t.Fatalf("age instance: %v", err)
	}
	route := &AgentProxyRoute{RouteID: "stale-route", InstanceID: inst.InstanceId, Kind: AgentProxyRouteKindTeams, Enabled: true}
	if err := DB(ctx).Create(route).Error; err != nil {
		t.Fatalf("create route: %v", err)
	}

	cleaned := CleanupDestroyedInstances(ctx, []uint{inst.ID}, 24*time.Hour)
	if len(cleaned) != 1 || cleaned[0] != inst.ID {
		t.Fatalf("cleaned ids=%v, want [%d]", cleaned, inst.ID)
	}
	var got AgentProxyRoute
	if err := DB(ctx).Where("route_id = ?", "stale-route").First(&got).Error; err != nil {
		t.Fatalf("query route: %v", err)
	}
	if got.Enabled {
		t.Fatal("cleanup should disable agent proxy route")
	}
}

func TestDisableAgentProxyRoutesForUser(t *testing.T) {
	cleanup := initAgentProxyRouteModelTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := DisableAgentProxyRoutesForUser(ctx, 0); err != nil {
		t.Fatalf("zero user should be noop: %v", err)
	}
	instances := []Instance{
		{Name: "i1", InstanceId: "ins-u1", UserID: 10},
		{Name: "i2", InstanceId: "ins-u2", UserID: 10},
		{Name: "i3", InstanceId: "ins-other", UserID: 20},
	}
	if err := DB(ctx).Create(&instances).Error; err != nil {
		t.Fatalf("create instances: %v", err)
	}
	routes := []AgentProxyRoute{
		{RouteID: "u1", InstanceID: "ins-u1", Kind: AgentProxyRouteKindTeams, Enabled: true},
		{RouteID: "u2", InstanceID: "ins-u2", Kind: AgentProxyRouteKindTeams, Enabled: true},
		{RouteID: "other", InstanceID: "ins-other", Kind: AgentProxyRouteKindTeams, Enabled: true},
	}
	if err := DB(ctx).Create(&routes).Error; err != nil {
		t.Fatalf("create routes: %v", err)
	}
	if err := DisableAgentProxyRoutesForUser(ctx, 10); err != nil {
		t.Fatalf("disable user routes: %v", err)
	}
	var got []AgentProxyRoute
	DB(ctx).Order("route_id").Find(&got)
	states := map[string]bool{}
	for _, r := range got {
		states[r.RouteID] = r.Enabled
	}
	if states["u1"] || states["u2"] || !states["other"] {
		t.Fatalf("unexpected enabled states: %#v", states)
	}
	if err := DisableAgentProxyRoutesForUser(ctx, 999); err != nil {
		t.Fatalf("user without instances should be noop: %v", err)
	}
}

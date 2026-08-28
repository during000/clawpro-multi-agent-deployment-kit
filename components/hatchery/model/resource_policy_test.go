package model

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupResourcePolicyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("underlying db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&SiteConfig{}, &ResourcePolicy{}, &UserGroup{}, &GroupClosure{}, &GroupConfigBinding{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if err := db.Create(&SiteConfig{CVMTemplate: DefaultCVMTemplate}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	t.Cleanup(UseDBForTest(db))
	return db
}

func TestExtractResourceConfigFromTemplate_KnownFieldsOnly(t *testing.T) {
	template := `{
		"InstanceChargeType":"PREPAID",
		"InstanceChargePrepaid":{"Period":1,"RenewFlag":"NOTIFY_AND_AUTO_RENEW","Ignored":"x"},
		"InstanceType":"Ai2.MEDIUM4",
		"SystemDisk":{"DiskType":"CLOUD_BSSD","DiskSize":50,"Ignored":"x"},
		"InternetAccessible":{"PublicIpAssigned":false,"InternetChargeType":"TRAFFIC_POSTPAID_BY_HOUR","InternetMaxBandwidthOut":5,"Ignored":"x"},
		"ImageId":"ignored"
	}`
	want := `{"instance_charge_prepaid":{"period":1,"renew_flag":"NOTIFY_AND_AUTO_RENEW"},"instance_charge_type":"PREPAID","instance_type":"Ai2.MEDIUM4","internet_accessible":{"internet_charge_type":"TRAFFIC_POSTPAID_BY_HOUR","internet_max_bandwidth_out":5,"public_ip_assigned":false},"system_disk":{"disk_size":50,"disk_type":"CLOUD_BSSD"}}`
	if got := extractResourceConfigFromTemplate(template); got != want {
		t.Fatalf("extracted config=%s, want %s", got, want)
	}
	if got := extractResourceConfigFromTemplate("{"); got != "{}" {
		t.Fatalf("invalid template extracted as %s", got)
	}
}

func seedResourcePolicyGroup(t *testing.T, db *gorm.DB, id uint, parentID uint, name string) {
	t.Helper()
	group := UserGroup{ID: id, ParentID: parentID, Name: name, FullPath: name, Source: GroupSourceManual}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group %d: %v", id, err)
	}
	if err := db.Create(&GroupClosure{AncestorID: id, DescendantID: id, Depth: 0}).Error; err != nil {
		t.Fatalf("create self closure %d: %v", id, err)
	}
	if parentID != 0 {
		var parentClosures []GroupClosure
		if err := db.Where("descendant_id = ?", parentID).Find(&parentClosures).Error; err != nil {
			t.Fatalf("find parent closures: %v", err)
		}
		for _, closure := range parentClosures {
			if err := db.Create(&GroupClosure{
				AncestorID: closure.AncestorID, DescendantID: id, Depth: closure.Depth + 1,
			}).Error; err != nil {
				t.Fatalf("create inherited closure: %v", err)
			}
		}
	}
}

func TestResourcePolicyDefaultLazyCreateConcurrent(t *testing.T) {
	setupResourcePolicyTestDB(t)
	const workers = 100
	ids := make(chan uint, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			policy, err := GetOrCreateDefaultResourcePolicy(context.Background())
			if err != nil {
				errs <- err
				return
			}
			ids <- policy.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatalf("lazy create: %v", err)
	}
	var first uint
	for id := range ids {
		if first == 0 {
			first = id
		}
		if id != first {
			t.Fatalf("default IDs differ: %d and %d", first, id)
		}
	}
	var count int64
	if err := DB(context.Background()).Model(&ResourcePolicy{}).Count(&count).Error; err != nil {
		t.Fatalf("count policies: %v", err)
	}
	if count != 1 {
		t.Fatalf("default policy count=%d, want 1", count)
	}
}

func TestResourcePolicyIndexedBindingsAndResolution(t *testing.T) {
	db := setupResourcePolicyTestDB(t)
	seedResourcePolicyGroup(t, db, 1, 0, "root")
	seedResourcePolicyGroup(t, db, 2, 1, "child")
	seedResourcePolicyGroup(t, db, 3, 2, "grandchild")
	seedResourcePolicyGroup(t, db, 4, 0, "sibling")

	parent, err := CreateResourcePolicy(context.Background(), "parent", `{"instance_type":"Ai2.MEDIUM4"}`, []uint{1, 4})
	if err != nil {
		t.Fatalf("create parent policy: %v", err)
	}
	child, err := CreateResourcePolicy(context.Background(), "child", `{"instance_type":"Ai2.LARGE8"}`, []uint{2})
	if err != nil {
		t.Fatalf("create child policy: %v", err)
	}

	var binding GroupConfigBinding
	if err := db.Where("config_type = ? AND config_key = ?", ConfigTypeResourcePolicy, strconv.FormatUint(uint64(child.ID), 10)).First(&binding).Error; err != nil {
		t.Fatalf("query indexed binding: %v", err)
	}
	if binding.GroupID != 2 || binding.ValueJSON != "{}" {
		t.Fatalf("binding=%+v, want group 2 and empty value JSON", binding)
	}

	groupsByPolicy, err := GetResourcePolicyGroups(context.Background(), []uint{parent.ID, child.ID})
	if err != nil {
		t.Fatalf("inverse groups: %v", err)
	}
	if len(groupsByPolicy[child.ID]) != 1 || groupsByPolicy[child.ID][0].ID != 2 {
		t.Fatalf("child policy groups=%v", groupsByPolicy[child.ID])
	}
	if len(groupsByPolicy[parent.ID]) != 2 ||
		groupsByPolicy[parent.ID][0].ID != 1 ||
		groupsByPolicy[parent.ID][1].ID != 4 {
		t.Fatalf("parent policy groups=%v, want groups 1 and 4", groupsByPolicy[parent.ID])
	}
	var parentBindingCount int64
	if err := db.Model(&GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ?", ConfigTypeResourcePolicy, strconv.FormatUint(uint64(parent.ID), 10)).
		Count(&parentBindingCount).Error; err != nil {
		t.Fatalf("count parent bindings: %v", err)
	}
	if parentBindingCount != 2 {
		t.Fatalf("parent binding count=%d, want 2", parentBindingCount)
	}

	resolved, err := ResolveEffectiveResourcePolicy(context.Background(), 3)
	if err != nil {
		t.Fatalf("resolve grandchild: %v", err)
	}
	if resolved.Policy.ID != child.ID || resolved.SourceGroupID != 2 || resolved.Depth != 1 {
		t.Fatalf("resolved=%+v, want child policy inherited from group 2", resolved)
	}

	resolved, err = ResolveEffectiveResourcePolicy(context.Background(), 1)
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	if resolved.Policy.ID != parent.ID || resolved.Depth != 0 {
		t.Fatalf("root resolved=%+v", resolved)
	}
}

func TestResourcePolicyOccupiedGroupRollsBack(t *testing.T) {
	db := setupResourcePolicyTestDB(t)
	seedResourcePolicyGroup(t, db, 1, 0, "root")
	if _, err := CreateResourcePolicy(context.Background(), "first", `{}`, []uint{1}); err != nil {
		t.Fatalf("create first: %v", err)
	}
	if _, err := CreateResourcePolicy(context.Background(), "second", `{}`, []uint{1}); !errors.Is(err, ErrResourcePolicyGroupOccupied) {
		t.Fatalf("second error=%v, want group occupied", err)
	}
	var count int64
	if err := db.Model(&ResourcePolicy{}).Where("name = ?", "second").Count(&count).Error; err != nil {
		t.Fatalf("count rolled back policy: %v", err)
	}
	if count != 0 {
		t.Fatalf("occupied create left %d policy rows", count)
	}
}

func TestResourcePolicyDeleteCleansBindingsAndFallsBack(t *testing.T) {
	db := setupResourcePolicyTestDB(t)
	seedResourcePolicyGroup(t, db, 1, 0, "root")
	policy, err := CreateResourcePolicy(context.Background(), "temporary", `{}`, []uint{1})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if err := DeleteResourcePolicy(context.Background(), policy.ID); err != nil {
		t.Fatalf("delete policy: %v", err)
	}
	var bindingCount int64
	if err := db.Model(&GroupConfigBinding{}).Where("config_type = ?", ConfigTypeResourcePolicy).Count(&bindingCount).Error; err != nil {
		t.Fatalf("count bindings: %v", err)
	}
	if bindingCount != 0 {
		t.Fatalf("binding count=%d, want 0", bindingCount)
	}
	resolved, err := ResolveEffectiveResourcePolicy(context.Background(), 1)
	if err != nil {
		t.Fatalf("resolve fallback: %v", err)
	}
	if !resolved.Policy.IsDefault || resolved.SourceGroupID != 0 {
		t.Fatalf("resolved=%+v, want tenant default", resolved)
	}
	if err := DeleteResourcePolicy(context.Background(), resolved.Policy.ID); !errors.Is(err, ErrDefaultResourcePolicy) {
		t.Fatalf("delete default error=%v", err)
	}
}

func TestResourcePolicyConcurrentGroupClaimHasSingleWinner(t *testing.T) {
	db := setupResourcePolicyTestDB(t)
	seedResourcePolicyGroup(t, db, 1, 0, "root")

	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, name := range []string{"first", "second"} {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := CreateResourcePolicy(context.Background(), name, `{}`, []uint{1})
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrResourcePolicyGroupOccupied):
			conflicts++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want one each", successes, conflicts)
	}
	var policyCount, bindingCount int64
	if err := db.Model(&ResourcePolicy{}).Count(&policyCount).Error; err != nil {
		t.Fatalf("count policies: %v", err)
	}
	if err := db.Model(&GroupConfigBinding{}).Where("config_type = ?", ConfigTypeResourcePolicy).Count(&bindingCount).Error; err != nil {
		t.Fatalf("count bindings: %v", err)
	}
	if policyCount != 1 || bindingCount != 1 {
		t.Fatalf("policies=%d bindings=%d, want one each", policyCount, bindingCount)
	}
}

func TestResourcePolicyDefaultUpdateProtectsIdentityAndScope(t *testing.T) {
	db := setupResourcePolicyTestDB(t)
	seedResourcePolicyGroup(t, db, 1, 0, "root")
	ctx := context.Background()

	policy, err := GetOrCreateDefaultResourcePolicy(ctx)
	if err != nil {
		t.Fatalf("get default policy: %v", err)
	}
	const editedConfig = `{"instance_type":"Ai2.LARGE8"}`
	if _, err := UpdateResourcePolicy(ctx, policy.ID, "", editedConfig, nil); err != nil {
		t.Fatalf("edit default config: %v", err)
	}
	reloaded, err := GetOrCreateDefaultResourcePolicy(ctx)
	if err != nil {
		t.Fatalf("reload default policy: %v", err)
	}
	if reloaded.ID != policy.ID || reloaded.ConfigJSON != editedConfig {
		t.Fatalf("reloaded default=%+v, want ID %d and edited config", reloaded, policy.ID)
	}

	if _, err := UpdateResourcePolicy(ctx, policy.ID, "renamed", `{}`, nil); !errors.Is(err, ErrDefaultResourcePolicy) {
		t.Fatalf("rename default error=%v, want protected", err)
	}
	if _, err := UpdateResourcePolicy(ctx, policy.ID, DefaultResourcePolicyName, `{}`, []uint{1}); !errors.Is(err, ErrDefaultResourcePolicy) {
		t.Fatalf("bind default error=%v, want protected", err)
	}
	reloaded, err = GetResourcePolicy(ctx, policy.ID)
	if err != nil {
		t.Fatalf("reload protected default: %v", err)
	}
	if reloaded.Name != DefaultResourcePolicyName || reloaded.ConfigJSON != editedConfig {
		t.Fatalf("protected default changed after rejected updates: %+v", reloaded)
	}
	var bindingCount int64
	if err := db.Model(&GroupConfigBinding{}).Where("config_type = ?", ConfigTypeResourcePolicy).Count(&bindingCount).Error; err != nil {
		t.Fatalf("count default bindings: %v", err)
	}
	if bindingCount != 0 {
		t.Fatalf("default has %d bindings, want none", bindingCount)
	}
}

func TestResourcePolicyOccupiedUpdateRollsBackAllChanges(t *testing.T) {
	db := setupResourcePolicyTestDB(t)
	seedResourcePolicyGroup(t, db, 1, 0, "first-group")
	seedResourcePolicyGroup(t, db, 2, 0, "second-group")
	ctx := context.Background()

	if _, err := CreateResourcePolicy(ctx, "first", `{}`, []uint{1}); err != nil {
		t.Fatalf("create occupying policy: %v", err)
	}
	second, err := CreateResourcePolicy(ctx, "second", `{"instance_type":"Ai2.MEDIUM2"}`, []uint{2})
	if err != nil {
		t.Fatalf("create policy to update: %v", err)
	}
	if _, err := UpdateResourcePolicy(ctx, second.ID, "changed", `{"instance_type":"Ai2.LARGE8"}`, []uint{1, 2}); !errors.Is(err, ErrResourcePolicyGroupOccupied) {
		t.Fatalf("occupied update error=%v, want group occupied", err)
	}

	reloaded, err := GetResourcePolicy(ctx, second.ID)
	if err != nil {
		t.Fatalf("reload rolled back policy: %v", err)
	}
	if reloaded.Name != "second" || reloaded.ConfigJSON != `{"instance_type":"Ai2.MEDIUM2"}` {
		t.Fatalf("policy changed despite rollback: %+v", reloaded)
	}
	groups, err := GetResourcePolicyGroups(ctx, []uint{second.ID})
	if err != nil {
		t.Fatalf("get rolled back scope: %v", err)
	}
	if len(groups[second.ID]) != 1 || groups[second.ID][0].ID != 2 {
		t.Fatalf("scope after rollback=%v, want only group 2", groups[second.ID])
	}
}

func TestResourcePolicyTenantIsolation(t *testing.T) {
	cleanup := setupTestDB(t, "tenant-A")
	defer cleanup()
	skipCtx := common.WithSkipIdentifier(context.Background())
	if err := gdb.WithContext(skipCtx).AutoMigrate(
		&ResourcePolicy{}, &UserGroup{}, &GroupClosure{}, &GroupConfigBinding{},
	); err != nil {
		t.Fatalf("migrate tenant policy models: %v", err)
	}
	ctxA := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-A"})
	ctxB := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-B"})

	groupA := UserGroup{Name: "root", FullPath: "root", Source: GroupSourceManual}
	if err := DB(ctxA).Create(&groupA).Error; err != nil {
		t.Fatalf("create tenant-A group: %v", err)
	}
	if err := DB(ctxA).Create(&GroupClosure{AncestorID: groupA.ID, DescendantID: groupA.ID, Depth: 0}).Error; err != nil {
		t.Fatalf("create tenant-A closure: %v", err)
	}
	groupB := UserGroup{Name: "root", FullPath: "root", Source: GroupSourceManual}
	if err := DB(ctxB).Create(&groupB).Error; err != nil {
		t.Fatalf("create tenant-B group: %v", err)
	}
	if err := DB(ctxB).Create(&GroupClosure{AncestorID: groupB.ID, DescendantID: groupB.ID, Depth: 0}).Error; err != nil {
		t.Fatalf("create tenant-B closure: %v", err)
	}

	policyA, err := CreateResourcePolicy(ctxA, "shared-name", `{}`, []uint{groupA.ID})
	if err != nil {
		t.Fatalf("create tenant-A policy: %v", err)
	}
	if _, err := GetResourcePolicy(ctxB, policyA.ID); !errors.Is(err, ErrResourcePolicyNotFound) {
		t.Fatalf("tenant-B lookup error=%v, want not found", err)
	}
	if _, err := CreateResourcePolicy(ctxB, "cross-tenant", `{}`, []uint{groupA.ID}); !errors.Is(err, ErrResourcePolicyGroupNotFound) {
		t.Fatalf("cross-tenant bind error=%v, want group not found", err)
	}
	policyB, err := CreateResourcePolicy(ctxB, "shared-name", `{}`, []uint{groupB.ID})
	if err != nil {
		t.Fatalf("create tenant-B policy with same name: %v", err)
	}
	if policyB.Identifier != "tenant-B" || policyA.Identifier != "tenant-A" {
		t.Fatalf("policy identifiers: tenant-A=%q tenant-B=%q", policyA.Identifier, policyB.Identifier)
	}
	directA, err := GetDirectResourcePoliciesByGroup(ctxA, []uint{groupB.ID})
	if err != nil {
		t.Fatalf("query tenant-A direct policies: %v", err)
	}
	if len(directA) != 0 {
		t.Fatalf("tenant-A saw tenant-B direct policy: %v", directA)
	}
}

func TestResourcePolicyListPaginationAndScopeBatching(t *testing.T) {
	db := setupResourcePolicyTestDB(t)
	seedResourcePolicyGroup(t, db, 1, 0, "alpha")
	seedResourcePolicyGroup(t, db, 2, 0, "beta")
	ctx := context.Background()

	first, err := CreateResourcePolicy(ctx, "first", `{}`, []uint{1})
	if err != nil {
		t.Fatalf("create first policy: %v", err)
	}
	second, err := CreateResourcePolicy(ctx, "second", `{}`, []uint{2})
	if err != nil {
		t.Fatalf("create second policy: %v", err)
	}
	defaultPolicy, err := GetOrCreateDefaultResourcePolicy(ctx)
	if err != nil {
		t.Fatalf("create default policy: %v", err)
	}

	page1, total, err := ListResourcePolicies(ctx, 1, 2)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if total != 3 || len(page1) != 2 || page1[0].ID != defaultPolicy.ID || page1[1].ID != first.ID {
		t.Fatalf("first page=%v total=%d, want default then oldest policy", page1, total)
	}
	page2, total, err := ListResourcePolicies(ctx, 2, 2)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if total != 3 || len(page2) != 1 || page2[0].ID != second.ID {
		t.Fatalf("second page=%v total=%d, want newest policy", page2, total)
	}

	queryCount := 0
	if err := db.Callback().Query().After("gorm:query").
		Register("test:resource-policy-scope-query-count", func(*gorm.DB) {
			queryCount++
		}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	groups, err := GetResourcePolicyGroups(ctx, []uint{first.ID, second.ID})
	if err != nil {
		t.Fatalf("batch policy scopes: %v", err)
	}
	if len(groups[first.ID]) != 1 || len(groups[second.ID]) != 1 {
		t.Fatalf("batch scopes=%v, want one group per policy", groups)
	}
	if queryCount != 2 {
		t.Fatalf("scope batching executed %d queries, want 2 independent of policy count", queryCount)
	}
}

func TestResourcePolicyDefaultNameIsReserved(t *testing.T) {
	db := setupResourcePolicyTestDB(t)
	seedResourcePolicyGroup(t, db, 1, 0, "first")
	seedResourcePolicyGroup(t, db, 2, 0, "second")
	ctx := context.Background()

	if _, err := CreateResourcePolicy(ctx, DefaultResourcePolicyName, `{}`, []uint{1}); !errors.Is(err, ErrDefaultResourcePolicy) {
		t.Fatalf("create with reserved default name error=%v, want protected", err)
	}
	var count int64
	if err := db.Model(&ResourcePolicy{}).Count(&count).Error; err != nil {
		t.Fatalf("count policies after rejected create: %v", err)
	}
	if count != 0 {
		t.Fatalf("rejected reserved-name create left %d policies", count)
	}

	ordinary, err := CreateResourcePolicy(ctx, "ordinary", `{}`, []uint{1})
	if err != nil {
		t.Fatalf("create ordinary policy: %v", err)
	}
	if _, err := UpdateResourcePolicy(ctx, ordinary.ID, DefaultResourcePolicyName, `{}`, []uint{2}); !errors.Is(err, ErrDefaultResourcePolicy) {
		t.Fatalf("rename to reserved default name error=%v, want protected", err)
	}
	reloaded, err := GetResourcePolicy(ctx, ordinary.ID)
	if err != nil {
		t.Fatalf("reload ordinary policy: %v", err)
	}
	if reloaded.Name != "ordinary" {
		t.Fatalf("rejected reserved rename changed policy name to %q", reloaded.Name)
	}
	groups, err := GetResourcePolicyGroups(ctx, []uint{ordinary.ID})
	if err != nil {
		t.Fatalf("get ordinary policy scope: %v", err)
	}
	if len(groups[ordinary.ID]) != 1 || groups[ordinary.ID][0].ID != 1 {
		t.Fatalf("rejected reserved rename changed scope: %v", groups[ordinary.ID])
	}
	defaultPolicy, err := GetOrCreateDefaultResourcePolicy(ctx)
	if err != nil {
		t.Fatalf("create real default after rejected names: %v", err)
	}
	if !defaultPolicy.IsDefault || defaultPolicy.Name != DefaultResourcePolicyName {
		t.Fatalf("default policy=%+v", defaultPolicy)
	}
}

func TestResourcePolicyBindingIndexesMatchSQLSchema(t *testing.T) {
	db := setupResourcePolicyTestDB(t)
	indexes, err := db.Migrator().GetIndexes(&GroupConfigBinding{})
	if err != nil {
		t.Fatalf("get binding indexes: %v", err)
	}
	actual := make(map[string][]string, len(indexes))
	for _, index := range indexes {
		actual[index.Name()] = index.Columns()
	}
	expected := map[string][]string{
		"uk_gcb":           {"identifier", "config_type", "config_key", "group_id"},
		"idx_gcb_group":    {"identifier", "group_id", "config_type"},
		"idx_gcb_resource": {"identifier", "config_type", "config_key"},
	}
	for name, columns := range expected {
		got, ok := actual[name]
		if !ok {
			t.Fatalf("index %s missing; actual=%v", name, actual)
		}
		if len(got) != len(columns) {
			t.Fatalf("index %s columns=%v, want %v", name, got, columns)
		}
		for i := range columns {
			if got[i] != columns[i] {
				t.Fatalf("index %s columns=%v, want %v", name, got, columns)
			}
		}
	}
}

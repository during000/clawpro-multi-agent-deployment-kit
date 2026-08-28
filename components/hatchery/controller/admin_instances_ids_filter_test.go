package controller

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"
)

// TestParseAdminInstancesIDFilters_IDsHappyPath 验证 ids 参数被正确解析到 RequestIDs。
func TestParseAdminInstancesIDFilters_IDsHappyPath(t *testing.T) {
	q := url.Values{}
	q.Set("ids", "1, 2, 3")
	var f adminQueryFilter
	if err := parseAdminInstancesIDFilters(q, &f); err != nil {
		t.Fatalf("不期望报错: %v", err)
	}
	if got, want := f.RequestIDs, []uint{1, 2, 3}; !equalUints(got, want) {
		t.Errorf("RequestIDs 期望 %v 实际 %v", want, got)
	}
	if len(f.RequestInstanceIDs) != 0 {
		t.Errorf("RequestInstanceIDs 应为空: %v", f.RequestInstanceIDs)
	}
}

// TestParseAdminInstancesIDFilters_InstanceIDsHappyPath 验证 instance_ids 参数解析。
func TestParseAdminInstancesIDFilters_InstanceIDsHappyPath(t *testing.T) {
	q := url.Values{}
	q.Set("instance_ids", "ins-aaa,  ins-bbb ,ins-ccc")
	var f adminQueryFilter
	if err := parseAdminInstancesIDFilters(q, &f); err != nil {
		t.Fatalf("不期望报错: %v", err)
	}
	want := []string{"ins-aaa", "ins-bbb", "ins-ccc"}
	if got := f.RequestInstanceIDs; !equalStrings(got, want) {
		t.Errorf("RequestInstanceIDs 期望 %v 实际 %v", want, got)
	}
}

// TestParseAdminInstancesIDFilters_BothCoexist 两参数同时传时都被解析,后续走 SQL 交集。
func TestParseAdminInstancesIDFilters_BothCoexist(t *testing.T) {
	q := url.Values{}
	q.Set("ids", "1,2")
	q.Set("instance_ids", "ins-x,ins-y")
	var f adminQueryFilter
	if err := parseAdminInstancesIDFilters(q, &f); err != nil {
		t.Fatalf("不期望报错: %v", err)
	}
	if len(f.RequestIDs) != 2 || len(f.RequestInstanceIDs) != 2 {
		t.Errorf("两个字段都应被填充: ids=%v instance_ids=%v", f.RequestIDs, f.RequestInstanceIDs)
	}
}

// TestParseAdminInstancesIDFilters_IDsEmpty 空字符串视为未传,不报错。
func TestParseAdminInstancesIDFilters_IDsEmpty(t *testing.T) {
	q := url.Values{}
	q.Set("ids", "")
	q.Set("instance_ids", "")
	var f adminQueryFilter
	if err := parseAdminInstancesIDFilters(q, &f); err != nil {
		t.Fatalf("空参数不应报错: %v", err)
	}
	if len(f.RequestIDs) != 0 || len(f.RequestInstanceIDs) != 0 {
		t.Errorf("空参数不应填充字段")
	}
}

// TestParseAdminInstancesIDFilters_IDsBadFormat ids 含非数字 → 400 ids 格式错误。
func TestParseAdminInstancesIDFilters_IDsBadFormat(t *testing.T) {
	q := url.Values{}
	q.Set("ids", "1,abc,3")
	var f adminQueryFilter
	err := parseAdminInstancesIDFilters(q, &f)
	if err == nil {
		t.Fatalf("期望报错 ids 格式错误")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "格式错误") {
		t.Errorf("错误信息期望包含\"格式错误\",实际: %v", err)
	}
}

// TestParseAdminInstancesIDFilters_IDsTooMany 超过上限 1000 → 400 数量超过上限。
func TestParseAdminInstancesIDFilters_IDsTooMany(t *testing.T) {
	parts := make([]string, 0, adminInstancesQueryMaxIDs+1)
	for i := 1; i <= adminInstancesQueryMaxIDs+1; i++ {
		parts = append(parts, strconv.Itoa(i))
	}
	q := url.Values{}
	q.Set("ids", strings.Join(parts, ","))
	var f adminQueryFilter
	err := parseAdminInstancesIDFilters(q, &f)
	if err == nil {
		t.Fatalf("期望报错 ids 数量超过上限")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "超过上限") {
		t.Errorf("错误信息期望包含\"超过上限\",实际: %v", err)
	}
}

// TestParseAdminInstancesIDFilters_IDsExactlyMax 恰好等于上限 1000 → 通过。
func TestParseAdminInstancesIDFilters_IDsExactlyMax(t *testing.T) {
	parts := make([]string, 0, adminInstancesQueryMaxIDs)
	for i := 1; i <= adminInstancesQueryMaxIDs; i++ {
		parts = append(parts, strconv.Itoa(i))
	}
	q := url.Values{}
	q.Set("ids", strings.Join(parts, ","))
	var f adminQueryFilter
	if err := parseAdminInstancesIDFilters(q, &f); err != nil {
		t.Fatalf("等于上限不应报错: %v", err)
	}
	if len(f.RequestIDs) != adminInstancesQueryMaxIDs {
		t.Errorf("期望 %d 项,实际 %d", adminInstancesQueryMaxIDs, len(f.RequestIDs))
	}
}

// TestParseAdminInstancesIDFilters_InstanceIDsTooMany instance_ids 超限 → 400。
func TestParseAdminInstancesIDFilters_InstanceIDsTooMany(t *testing.T) {
	parts := make([]string, 0, adminInstancesQueryMaxIDs+1)
	for i := 0; i <= adminInstancesQueryMaxIDs; i++ {
		parts = append(parts, "ins-"+strconv.Itoa(i))
	}
	q := url.Values{}
	q.Set("instance_ids", strings.Join(parts, ","))
	var f adminQueryFilter
	err := parseAdminInstancesIDFilters(q, &f)
	if err == nil {
		t.Fatalf("期望报错 instance_ids 数量超过上限")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "instance_ids") || !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "超过上限") {
		t.Errorf("错误信息期望包含\"instance_ids\"和\"超过上限\",实际: %v", err)
	}
}

// TestParseAdminInstancesIDFilters_InstanceIDsAllWhitespace 全空白片段被剔除后为 0,视为未传。
func TestParseAdminInstancesIDFilters_InstanceIDsAllWhitespace(t *testing.T) {
	q := url.Values{}
	q.Set("instance_ids", " , ,   ,")
	var f adminQueryFilter
	if err := parseAdminInstancesIDFilters(q, &f); err != nil {
		t.Fatalf("纯空白不应报错: %v", err)
	}
	if len(f.RequestInstanceIDs) != 0 {
		t.Errorf("空白片段应被剔除,实际: %v", f.RequestInstanceIDs)
	}
}

// TestQueryInstancesWithFilter_FilterByRequestIDs 直接验证 SQL 下推按 RequestIDs 过滤。
func TestQueryInstancesWithFilter_FilterByRequestIDs(t *testing.T) {
	initTestDB(t)
	seedTestData(t)

	// seedTestData 创建 4 条实例,id 1..4
	items, total := queryInstancesWithFilter(context.Background(), 1, 100, adminQueryFilter{
		RequestIDs: []uint{1, 3},
	})
	if total != 2 {
		t.Errorf("期望 total=2,实际=%d", total)
	}
	if len(items) != 2 {
		t.Errorf("期望 2 条,实际 %d", len(items))
	}
	gotIDs := map[uint]bool{}
	for _, it := range items {
		gotIDs[it.ID] = true
	}
	if !gotIDs[1] || !gotIDs[3] {
		t.Errorf("期望返回 id=1 和 id=3,实际 %v", gotIDs)
	}
}

// TestQueryInstancesWithFilter_FilterByRequestInstanceIDs 验证按腾讯云 instance_id 过滤。
func TestQueryInstancesWithFilter_FilterByRequestInstanceIDs(t *testing.T) {
	initTestDB(t)
	// 自己写 fixture,确保 InstanceId 字段非空
	users := []model.User{{Username: "u1", Password: "x", Role: "user"}}
	if err := model.DB(context.Background()).Create(&users[0]).Error; err != nil {
		t.Fatal(err)
	}
	insts := []model.Instance{
		{Name: "a", UserID: users[0].ID, InstanceId: "ins-aaa", ProxyToken: strPtr("t1")},
		{Name: "b", UserID: users[0].ID, InstanceId: "ins-bbb", ProxyToken: strPtr("t2")},
		{Name: "c", UserID: users[0].ID, InstanceId: "ins-ccc", ProxyToken: strPtr("t3")},
	}
	for i := range insts {
		if err := model.DB(context.Background()).Create(&insts[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	items, total := queryInstancesWithFilter(context.Background(), 1, 100, adminQueryFilter{
		RequestInstanceIDs: []string{"ins-aaa", "ins-ccc"},
	})
	if total != 2 {
		t.Errorf("期望 total=2,实际=%d", total)
	}
	gotInsIDs := map[string]bool{}
	for _, it := range items {
		gotInsIDs[it.InstanceId] = true
	}
	if !gotInsIDs["ins-aaa"] || !gotInsIDs["ins-ccc"] {
		t.Errorf("期望命中 ins-aaa / ins-ccc,实际 %v", gotInsIDs)
	}
	if gotInsIDs["ins-bbb"] {
		t.Errorf("不应命中 ins-bbb")
	}
}

// TestQueryInstancesWithFilter_FilterByBoth 两个参数同时下推 → AND 交集。
func TestQueryInstancesWithFilter_FilterByBoth(t *testing.T) {
	initTestDB(t)
	users := []model.User{{Username: "u1", Password: "x", Role: "user"}}
	if err := model.DB(context.Background()).Create(&users[0]).Error; err != nil {
		t.Fatal(err)
	}
	insts := []model.Instance{
		{Name: "a", UserID: users[0].ID, InstanceId: "ins-aaa", ProxyToken: strPtr("t1")}, // id=1
		{Name: "b", UserID: users[0].ID, InstanceId: "ins-bbb", ProxyToken: strPtr("t2")}, // id=2
		{Name: "c", UserID: users[0].ID, InstanceId: "ins-ccc", ProxyToken: strPtr("t3")}, // id=3
	}
	for i := range insts {
		if err := model.DB(context.Background()).Create(&insts[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	// ids 命中 1/2/3,instance_ids 只命中 ins-bbb → 交集 {id=2,ins-bbb}
	items, total := queryInstancesWithFilter(context.Background(), 1, 100, adminQueryFilter{
		RequestIDs:         []uint{insts[0].ID, insts[1].ID, insts[2].ID},
		RequestInstanceIDs: []string{"ins-bbb"},
	})
	if total != 1 {
		t.Errorf("期望 total=1(交集),实际=%d", total)
	}
	if len(items) != 1 || items[0].InstanceId != "ins-bbb" {
		t.Errorf("期望仅 ins-bbb,实际 %+v", items)
	}
}

// ─── helpers ────────────────────────────────────────────────────────────

func equalUints(a, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

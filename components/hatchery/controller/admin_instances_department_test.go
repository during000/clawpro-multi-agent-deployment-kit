package controller

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestBuildAdminInstanceWithStatus_DepartmentFieldsPropagated 验证：
// adminInstanceItem 中的 Department / Departments / DepartmentPath
// 在转换为 adminInstanceItemWithStatus 时被原样透传。
func TestBuildAdminInstanceWithStatus_DepartmentFieldsPropagated(t *testing.T) {
	depts := []deptWithPath{
		{
			OneIDDepartment: model.OneIDDepartment{
				DepartmentID:       "d1",
				DepartmentName:     "OpenClaw企业版体验",
				DepartmentParentID: "",
				IsMainDepartment:   false,
			},
			DepartmentPath: "OpenClaw企业版体验",
		},
		{
			OneIDDepartment: model.OneIDDepartment{
				DepartmentID:       "d3",
				DepartmentName:     "市场二组",
				DepartmentParentID: "d2",
				IsMainDepartment:   true,
			},
			DepartmentPath: "OpenClaw企业版体验/新组/市场组/市场二组",
		},
	}
	item := adminInstanceItem{
		Instance:       model.Instance{AgentType: "openclaw"},
		Username:       "alice",
		Department:     "市场二组",
		Departments:    depts,
		DepartmentPath: "OpenClaw企业版体验/新组/市场组/市场二组",
	}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	got := buildAdminInstanceWithStatus(context.Background(), item, cvmInfo)
	if got.Department != "市场二组" {
		t.Errorf("Department=%q，期望 %q", got.Department, "市场二组")
	}
	if got.DepartmentPath != "OpenClaw企业版体验/新组/市场组/市场二组" {
		t.Errorf("DepartmentPath=%q，期望 %q", got.DepartmentPath, "OpenClaw企业版体验/新组/市场组/市场二组")
	}
	if len(got.Departments) != 2 {
		t.Fatalf("Departments 长度=%d，期望 2", len(got.Departments))
	}
	if got.Departments[1].DepartmentName != "市场二组" || !got.Departments[1].IsMainDepartment {
		t.Errorf("Departments[1] 基础字段透传错误: %+v", got.Departments[1])
	}
	if got.Departments[1].DepartmentPath != "OpenClaw企业版体验/新组/市场组/市场二组" {
		t.Errorf("Departments[1].DepartmentPath=%q，期望 %q",
			got.Departments[1].DepartmentPath, "OpenClaw企业版体验/新组/市场组/市场二组")
	}
}

// TestBuildAdminInstanceWithStatus_EmptyDepartmentSerialization 验证：
//   - 当 adminInstanceItem 的部门字段都为零值时
//   - JSON 序列化结果中 "department" 键存在且为 ""（不省略）
//   - "departments" 和 "department_path" 不出现在序列化结果中（omitempty 生效）
func TestBuildAdminInstanceWithStatus_EmptyDepartmentSerialization(t *testing.T) {
	item := adminInstanceItem{
		Instance: model.Instance{AgentType: "openclaw"},
		Username: "bob",
		// Department / Departments / DepartmentPath 全为零值
	}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	got := buildAdminInstanceWithStatus(context.Background(), item, cvmInfo)
	if got.Department != "" {
		t.Errorf("Department 应为空字符串，实际 %q", got.Department)
	}
	if got.Departments != nil {
		t.Errorf("Departments 应为 nil，实际 %+v", got.Departments)
	}
	if got.DepartmentPath != "" {
		t.Errorf("DepartmentPath 应为空字符串，实际 %q", got.DepartmentPath)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal 失败: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if depRaw, ok := m["department"]; !ok {
		t.Errorf(`json 中缺少 "department" 字段，应永远存在`)
	} else if string(depRaw) != `""` {
		t.Errorf(`"department"=%s，期望 ""`, string(depRaw))
	}
	if _, ok := m["departments"]; ok {
		t.Errorf(`空切片应被 omitempty 省略，但 json 中出现了 "departments"`)
	}
	if _, ok := m["department_path"]; ok {
		t.Errorf(`空字符串应被 omitempty 省略，但 json 中出现了 "department_path"`)
	}
}

// initInstancesDeptIntegrationDB 为 /admin/instances JSON 部门字段集成测试
// 准备包含 OneID 画像 / 部门表的内存 SQLite。
func initInstancesDeptIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
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

// TestHandleAdminInstances_JSONDepartmentFields 集成测试：
// 模拟一个 OneID 用户创建的实例，请求 /admin/instances JSON 接口，
// 断言响应里携带正确的 department / departments / department_path 字段，
// 且 department_path 形如 "OpenClaw企业版体验/新组/市场组/市场二组"。
func TestHandleAdminInstances_JSONDepartmentFields(t *testing.T) {
	db := initInstancesDeptIntegrationDB(t)

	// seed 4 级部门链：d1 → d2 → d3 → d4
	now := time.Now()
	deptRecords := []model.OneIDDepartmentRecord{
		{DepartmentID: "d1", DepartmentName: "OpenClaw企业版体验", DepartmentParentID: "", SyncedAt: now},
		{DepartmentID: "d2", DepartmentName: "新组", DepartmentParentID: "d1", SyncedAt: now},
		{DepartmentID: "d3", DepartmentName: "市场组", DepartmentParentID: "d2", SyncedAt: now},
		{DepartmentID: "d4", DepartmentName: "市场二组", DepartmentParentID: "d3", SyncedAt: now},
	}
	for i := range deptRecords {
		if err := db.Create(&deptRecords[i]).Error; err != nil {
			t.Fatalf("seed dept 失败: %v", err)
		}
	}

	// seed 用户：alice 是 OneID 用户，bob 是纯密码用户（用于验证混合短路）
	alice := model.User{Username: "alice", Password: "x", Role: "user", OneIDSub: strPtr("sub-alice")}
	bob := model.User{Username: "bob", Password: "x", Role: "user"}
	if err := db.Create(&alice).Error; err != nil {
		t.Fatalf("seed alice 失败: %v", err)
	}
	if err := db.Create(&bob).Error; err != nil {
		t.Fatalf("seed bob 失败: %v", err)
	}

	// alice 的 OneID 画像
	aliceDepts := []model.OneIDDepartment{
		{DepartmentID: "d4", DepartmentName: "市场二组", DepartmentParentID: "d3", IsMainDepartment: true},
	}
	deptsJSON, _ := json.Marshal(aliceDepts)
	if err := db.Create(&model.OneIDUserProfile{
		OneIDSub:        "sub-alice",
		Name:            "alice",
		MainDeptID:      "d4",
		MainDeptName:    "市场二组",
		DepartmentsJSON: string(deptsJSON),
		SyncedAt:        now,
	}).Error; err != nil {
		t.Fatalf("seed profile 失败: %v", err)
	}

	// seed 实例：alice 的实例 + bob 的实例
	insts := []model.Instance{
		{Name: "alice-inst", UserID: alice.ID, ProxyToken: strPtr("sk-1")},
		{Name: "bob-inst", UserID: bob.ID, ProxyToken: strPtr("sk-2")},
	}
	for i := range insts {
		if err := db.Create(&insts[i]).Error; err != nil {
			t.Fatalf("seed inst 失败: %v", err)
		}
	}

	// 调用 /admin/instances JSON 分支（绕过 requireAdmin，复用 adminInstancesHandler 简化版本）
	req := httptest.NewRequest(http.MethodGet, "/admin/instances?page=1&page_size=20", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	// 直接调 queryInstancesWithFilter + jsonOK，复用 adminInstancesHandler 的逻辑形态
	page, pageSize := parsePagination(req)
	items, total := queryInstancesWithFilter(req.Context(), page, pageSize, adminQueryFilter{})
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	jsonOK(w, map[string]interface{}{
		"instances":   items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态=%d，期望 200", w.Code)
	}

	var resp struct {
		Instances []struct {
			Name           string `json:"Name"`
			UserID         uint   `json:"UserID"`
			Department     string `json:"department"`
			Departments    []struct {
				DepartmentID       string `json:"department_id"`
				DepartmentName     string `json:"department_name"`
				DepartmentParentID string `json:"department_parent_id"`
				IsMainDepartment   bool   `json:"is_main_department"`
				DepartmentPath     string `json:"department_path"`
			} `json:"departments"`
			DepartmentPath string `json:"department_path"`
		} `json:"instances"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Total != 2 {
		t.Fatalf("total=%d，期望 2", resp.Total)
	}
	if len(resp.Instances) != 2 {
		t.Fatalf("instances 长度=%d，期望 2", len(resp.Instances))
	}

	// 找到 alice 的实例（OneID 用户）
	var aliceInst, bobInst int = -1, -1
	for i, it := range resp.Instances {
		if it.UserID == alice.ID {
			aliceInst = i
		}
		if it.UserID == bob.ID {
			bobInst = i
		}
	}
	if aliceInst < 0 || bobInst < 0 {
		t.Fatalf("响应中找不到 alice 或 bob 的实例: %+v", resp.Instances)
	}

	// alice：所有部门字段都应填充
	a := resp.Instances[aliceInst]
	if a.Department != "市场二组" {
		t.Errorf("alice department=%q，期望 %q", a.Department, "市场二组")
	}
	if a.DepartmentPath != "OpenClaw企业版体验/新组/市场组/市场二组" {
		t.Errorf("alice department_path=%q，期望 %q",
			a.DepartmentPath, "OpenClaw企业版体验/新组/市场组/市场二组")
	}
	if len(a.Departments) != 1 {
		t.Fatalf("alice departments 长度=%d，期望 1", len(a.Departments))
	}
	if a.Departments[0].DepartmentName != "市场二组" || !a.Departments[0].IsMainDepartment {
		t.Errorf("alice departments[0] 字段错误: %+v", a.Departments[0])
	}
	if a.Departments[0].DepartmentPath != "OpenClaw企业版体验/新组/市场组/市场二组" {
		t.Errorf("alice departments[0].department_path=%q，期望 %q",
			a.Departments[0].DepartmentPath, "OpenClaw企业版体验/新组/市场组/市场二组")
	}

	// bob：纯密码用户，department 为空，departments / department_path 经 omitempty 应不出现
	// 这里反序列化到结构体后无法直接区分"不存在"和"零值"，但能验证 department=""
	b := resp.Instances[bobInst]
	if b.Department != "" {
		t.Errorf("bob department=%q，期望空串（无 OneID 画像）", b.Department)
	}
	if len(b.Departments) != 0 {
		t.Errorf("bob departments 应为空，实际 %+v", b.Departments)
	}
	if b.DepartmentPath != "" {
		t.Errorf("bob department_path=%q，期望空串", b.DepartmentPath)
	}

	// 进一步验证 omitempty 在 raw JSON 上确实生效（bob 那条不应有 departments / department_path 键）
	// 重新跑一次拿原始字节
	w2 := httptest.NewRecorder()
	jsonOK(w2, map[string]interface{}{"instances": items})
	var raw struct {
		Instances []map[string]json.RawMessage `json:"instances"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &raw); err != nil {
		t.Fatalf("二次反序列化失败: %v", err)
	}
	for _, m := range raw.Instances {
		if string(m["UserID"]) == "" {
			continue
		}
		// 找到 bob 的那一条（UserID = bob.ID）
		var uid uint
		_ = json.Unmarshal(m["UserID"], &uid)
		if uid != bob.ID {
			continue
		}
		if _, ok := m["departments"]; ok {
			t.Errorf("bob 实例不应有 departments 字段（omitempty 失效）")
		}
		if _, ok := m["department_path"]; ok {
			t.Errorf("bob 实例不应有 department_path 字段（omitempty 失效）")
		}
		if _, ok := m["department"]; !ok {
			t.Errorf(`bob 实例必须有 "department" 字段（不带 omitempty）`)
		}
	}
}

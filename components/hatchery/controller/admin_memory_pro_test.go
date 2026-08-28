package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupMemoryProDB 初始化 Memory Pro 测试所需的数据库。
func setupMemoryProDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试 DB 失败: %v", err)
	}
	// SQLite :memory: 数据库每个连接都是独立的空库。
	// 必须限制为单连接，否则 GORM 连接池并发打开第二连接时会拿到没有任何表的空库。
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.MemoryTDAIPlugin{},
		&model.TdaiJob{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	oldSnap := common.FixedSnapshot
	if common.FixedSnapshot == nil {
		common.FixedSnapshot = &common.TenantSnapshot{}
	}
	t.Cleanup(func() {
		// 等待异步 goroutine 完成（如 getUserFromToken 中的 UpdateAPITokenLastUsed）
		time.Sleep(50 * time.Millisecond)
		origDB()
		common.FixedSnapshot = oldSnap
	})
	// 默认配置
	db.Create(&model.SiteConfig{
		MemoryTDAIEnable:            false,
		MemoryTDAISupportedVersions: "[]",
		VpcId:                       "vpc-test",
		SubnetIds:                   `{"ap-guangzhou-3":"subnet-test"}`,
		SecurityGroupId:             "sg-test",
	})
	// 初始化 session store（requireLogin 依赖）
	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32bytes-padding!"))
	}
}

// makeAdminRequest 构建带 AdminToken 的 JSON 请求。
func makeAdminRequest(method, path, body string) (*http.Request, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req, httptest.NewRecorder()
}

// parseJSON 解析响应 JSON body。
func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应 JSON 失败: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

// --- resolveNetworkParams 测试（纯函数，不依赖外部服务） ---

func TestResolveNetworkParams_GlobalVpcPreferred(t *testing.T) {
	// 管理员手动配置的全局 VPC + 子网优先
	cfg := &model.SiteConfig{
		VpcId:            "vpc-global",
		SubnetIds:        `{"ap-guangzhou-3":"subnet-global"}`,
		DefaultVpcId:     "vpc-default",
		DefaultSubnetIds: `{"ap-guangzhou-3":["subnet-default"]}`,
		SecurityGroupId:  "sg-001",
	}
	vpcId, subnetId, _, err := resolveNetworkParams(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vpcId != "vpc-global" {
		t.Errorf("vpcId = %q, want vpc-global (global preferred)", vpcId)
	}
	if subnetId != "subnet-global" {
		t.Errorf("subnetId = %q, want subnet-global (same source as vpc)", subnetId)
	}
	// 注：sgIds 在新模型下从 DefaultRuleSet + managed_sg_pool ACTIVE 取，
	// 不再是 cfg.SecurityGroupId；本测试不构造 DB 状态，跳过 sgIds 断言。
	// 专门的 SG 路径覆盖见 sg_ruleset_helpers_test.go (待 1.10 任务实现)。
}

func TestResolveNetworkParams_DefaultVpcFallback(t *testing.T) {
	// 无手动配置时，回退到自动创建的默认 VPC + 子网
	cfg := &model.SiteConfig{
		DefaultVpcId:     "vpc-default",
		DefaultSubnetIds: `{"ap-guangzhou-3":["subnet-default"]}`,
		SecurityGroupId:  "sg-001",
	}
	vpcId, subnetId, _, err := resolveNetworkParams(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vpcId != "vpc-default" {
		t.Errorf("vpcId = %q, want vpc-default", vpcId)
	}
	if subnetId != "subnet-default" {
		t.Errorf("subnetId = %q, want subnet-default", subnetId)
	}
}

func TestResolveNetworkParams_NoCrossMix(t *testing.T) {
	// 关键场景：DefaultVpcId 存在但 DefaultSubnetIds 为空，SubnetIds 属于另一个 VPC。
	// 修复前会交叉取 DefaultVpcId + SubnetIds 导致不匹配。
	// 修复后：手动 VPC 无子网 → 回退到默认 VPC（也无子网）→ 报错。
	cfg := &model.SiteConfig{
		DefaultVpcId:    "vpc-default-no-subnet",
		SubnetIds:       `{"ap-guangzhou-3":"subnet-from-other-vpc"}`,
		SecurityGroupId: "sg-001",
	}
	_, _, _, err := resolveNetworkParams(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error: DefaultVpcId has no matching subnet, should not cross-mix with SubnetIds")
	}
}

func TestResolveNetworkParams_OnlyGlobalVpc(t *testing.T) {
	// 只有管理员手动配置，无默认 VPC
	cfg := &model.SiteConfig{
		VpcId:     "vpc-manual",
		SubnetIds: `{"zone":"subnet-manual"}`,
	}
	vpcId, subnetId, _, err := resolveNetworkParams(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vpcId != "vpc-manual" {
		t.Errorf("vpcId = %q, want vpc-manual", vpcId)
	}
	if subnetId != "subnet-manual" {
		t.Errorf("subnetId = %q, want subnet-manual", subnetId)
	}
}

func TestResolveNetworkParams_NoVpc(t *testing.T) {
	cfg := &model.SiteConfig{}
	_, _, _, err := resolveNetworkParams(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for no VPC")
	}
}

func TestResolveNetworkParams_NoSubnet(t *testing.T) {
	cfg := &model.SiteConfig{VpcId: "vpc-001"}
	_, _, _, err := resolveNetworkParams(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for no subnet")
	}
}

func TestResolveNetworkParams_NoSecurityGroup(t *testing.T) {
	cfg := &model.SiteConfig{
		VpcId:     "vpc-001",
		SubnetIds: `{"zone":"subnet-001"}`,
	}
	_, _, sgIds, err := resolveNetworkParams(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sgIds != nil {
		t.Errorf("sgIds should be nil when no security group, got %v", sgIds)
	}
}

// --- HandleAdminMemoryDefaultPlan 测试（HTTP handler，AdminToken 认证） ---

func TestDefaultPlan_GET(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 更新配置
	model.UpdateSiteConfig(context.Background(), map[string]any{"memory_default_plan": "free", "memory_tdai_enable": true})

	req, w := makeAdminRequest("GET", "/admin/memory/default-plan", "")
	HandleAdminMemoryDefaultPlan(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if result["memory_default_plan"] != "free" {
		t.Errorf("memory_default_plan = %v, want free", result["memory_default_plan"])
	}
}

func TestDefaultPlan_GET_FallbackFromBool(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 确保 memory_default_plan 为空且 memory_tdai_enable=true
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]any{
		"memory_default_plan": "",
		"memory_tdai_enable":  true,
	})

	req, w := makeAdminRequest("GET", "/admin/memory/default-plan", "")
	HandleAdminMemoryDefaultPlan(w, req)

	result := parseJSON(t, w)
	if result["memory_default_plan"] != "free" {
		t.Errorf("fallback: got %v, want free", result["memory_default_plan"])
	}
}

func TestDefaultPlan_GET_FallbackOff(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("GET", "/admin/memory/default-plan", "")
	HandleAdminMemoryDefaultPlan(w, req)

	result := parseJSON(t, w)
	if result["memory_default_plan"] != "off" {
		t.Errorf("fallback: got %v, want off", result["memory_default_plan"])
	}
}

func TestDefaultPlan_PUT_Valid(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("PUT", "/admin/memory/default-plan", `{"memory_default_plan":"pro"}`)
	HandleAdminMemoryDefaultPlan(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if result["memory_default_plan"] != "pro" {
		t.Errorf("got %v, want pro", result["memory_default_plan"])
	}

	// 验证 DB 已更新
	cfg := model.GetSiteConfig(context.Background())
	if cfg.MemoryDefaultPlan != "pro" {
		t.Errorf("DB: memory_default_plan = %q, want pro", cfg.MemoryDefaultPlan)
	}
}

func TestDefaultPlan_PUT_InvalidPlan(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("PUT", "/admin/memory/default-plan", `{"memory_default_plan":"invalid"}`)
	HandleAdminMemoryDefaultPlan(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestDefaultPlan_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("DELETE", "/admin/memory/default-plan", "")
	HandleAdminMemoryDefaultPlan(w, req)

	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

// --- HandleAdminMemoryPlanSwitch 参数校验测试 ---

func TestBatchSwitch_EmptyInstanceIDs(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch", `{"instance_ids":[],"target_plan":"free"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestBatchSwitch_InvalidTargetPlan(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-001"],"target_plan":"invalid"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestBatchSwitch_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("GET", "/admin/memory/plan/switch", "")
	HandleAdminMemoryPlanSwitch(w, req)

	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestBatchSwitch_NonexistentInstance(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-nonexist"],"target_plan":"free"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	result := parseJSON(t, w)
	results := result["results"].([]any)
	item := results[0].(map[string]any)
	if item["error"] == nil || item["error"] == "" {
		t.Error("nonexistent instance should have error in result")
	}
}

func TestBatchSwitch_SwitchInProgress(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 创建实例和 plugin（有进行中的切换）
	model.DB(context.Background()).Create(&model.Instance{Name: "test", InstanceId: "ins-busy"})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-busy",
		CurrentPlan:  model.MemoryPlanOff,
		SwitchStatus: model.MemorySwitchStatusSwitchingToFree,
	})

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-busy"],"target_plan":"pro"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	result := parseJSON(t, w)
	results := result["results"].([]any)
	item := results[0].(map[string]any)
	if item["error"] == nil || item["error"] == "" {
		t.Error("instance with switch in progress should report error")
	}
}

func TestBatchSwitch_Success(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{Name: "test-ok", InstanceId: "ins-ok"})

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-ok"],"target_plan":"free"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if result["target_plan"] != "free" {
		t.Errorf("target_plan = %v, want free", result["target_plan"])
	}
	results := result["results"].([]any)
	item := results[0].(map[string]any)
	if item["task_id"] == nil || item["task_id"].(float64) == 0 {
		t.Error("successful switch should have task_id")
	}

	// 验证 plugin 状态已更新
	var plugin model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "ins-ok").First(&plugin)
	if plugin.SwitchStatus != model.MemorySwitchStatusSwitchingToFree {
		t.Errorf("switch_status = %q, want SWITCHING_TO_FREE", plugin.SwitchStatus)
	}
	if plugin.DesiredPlan != model.MemoryPlanFree {
		t.Errorf("desired_plan = %q, want FREE", plugin.DesiredPlan)
	}
}

// --- HandleAdminMemoryInstances 测试 ---

func TestMemoryInstances_EmptyDB(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("GET", "/admin/memory/instances?page=1&page_size=10", "")
	HandleAdminMemoryInstances(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	result := parseJSON(t, w)
	if result["total"].(float64) != 0 {
		t.Errorf("total = %v, want 0", result["total"])
	}
}

func TestMemoryInstances_WithData(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 创建测试数据
	model.DB(context.Background()).Create(&model.User{Username: "alice", Password: "x", Role: "user"})
	var user model.User
	model.DB(context.Background()).First(&user)
	model.DB(context.Background()).Create(&model.Instance{Name: "my-claw", InstanceId: "ins-mem-001", UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-mem-001", CurrentPlan: model.MemoryPlanFree})

	req, w := makeAdminRequest("GET", "/admin/memory/instances?page=1&page_size=10", "")
	HandleAdminMemoryInstances(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	total := result["total"].(float64)
	if total != 1 {
		t.Errorf("total = %v, want 1", total)
	}
}

func TestMemoryInstances_PlanFilter(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{Name: "free-inst", InstanceId: "ins-f1"})
	model.DB(context.Background()).Create(&model.Instance{Name: "pro-inst", InstanceId: "ins-p1"})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-f1", CurrentPlan: model.MemoryPlanFree})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-p1", CurrentPlan: model.MemoryPlanPro})

	req, w := makeAdminRequest("GET", "/admin/memory/instances?plan=FREE", "")
	HandleAdminMemoryInstances(w, req)

	result := parseJSON(t, w)
	items := result["items"].([]any)
	for _, item := range items {
		m := item.(map[string]any)
		if m["current_plan"] != "FREE" {
			t.Errorf("plan filter: got %v, want FREE", m["current_plan"])
		}
	}
}

// --- HandleAdminMemoryProActivate 参数校验 ---

func TestProActivate_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("GET", "/admin/memory/pro/activate", "")
	HandleAdminMemoryProActivate(w, req)

	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestProActivate_InvalidMemoryLimit(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("POST", "/admin/memory/pro/activate", `{"memory_limit": 0}`)
	HandleAdminMemoryProActivate(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestProActivate_NegativeMemoryLimit(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("POST", "/admin/memory/pro/activate", `{"memory_limit": -1}`)
	HandleAdminMemoryProActivate(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestProActivate_BadJSON(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("POST", "/admin/memory/pro/activate", `{bad json}`)
	HandleAdminMemoryProActivate(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// --- HandleAdminMemoryProRelease 参数校验 ---

func TestProRelease_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("GET", "/admin/memory/pro/release", "")
	HandleAdminMemoryProRelease(w, req)

	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

// --- releaseProMemSpaceForInstance 测试（不依赖外部 SDK） ---

func TestReleaseProMemSpace_NoPlugin(t *testing.T) {
	setupMemoryProDB(t)

	ok := releaseProMemSpaceForInstance(context.Background(), "ins-nonexist", true)
	if !ok {
		t.Error("no plugin should return true")
	}
}

func TestReleaseProMemSpace_NoPoolID(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-no-pool",
		CurrentPlan: model.MemoryPlanFree,
	})

	ok := releaseProMemSpaceForInstance(context.Background(), "ins-no-pool", true)
	if !ok {
		t.Error("no pool_id should return true")
	}
}

// --- resubmitProSwitchAfterReinstall 测试 ---

func TestResubmitProSwitch_NotPro(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-free-reinstall",
		CurrentPlan: model.MemoryPlanFree,
	})

	// 不应 panic，也不应创建任务
	resubmitProSwitchAfterReinstall(context.Background(), "ins-free-reinstall")

	var count int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Count(&count)
	if count != 0 {
		t.Error("non-PRO instance should not create resubmit job")
	}
}

func TestResubmitProSwitch_ProWithPool(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-pro-reinstall",
		CurrentPlan: model.MemoryPlanPro,
		PoolID:      "space-001",
		Endpoint:    "http://10.0.0.1:3306",
	})

	resubmitProSwitchAfterReinstall(context.Background(), "ins-pro-reinstall")

	var count int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Count(&count)
	if count != 1 {
		t.Errorf("PRO instance should create 1 resubmit job, got %d", count)
	}

	var plugin model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "ins-pro-reinstall").First(&plugin)
	if plugin.SwitchStatus != model.MemorySwitchStatusSwitchingToPro {
		t.Errorf("switch_status = %q, want SWITCHING_TO_PRO", plugin.SwitchStatus)
	}
}

func TestResubmitProSwitch_AlreadySwitching(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-pro-switching",
		CurrentPlan:  model.MemoryPlanPro,
		PoolID:       "space-002",
		Endpoint:     "http://10.0.0.1:3306",
		SwitchStatus: model.MemorySwitchStatusSwitchingToPro,
	})

	resubmitProSwitchAfterReinstall(context.Background(), "ins-pro-switching")

	var count int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Count(&count)
	if count != 0 {
		t.Error("already switching should not create another job")
	}
}

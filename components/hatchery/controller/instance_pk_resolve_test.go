package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initInstancePKTestDB 初始化用于解析函数 / Logs 接口测试的最小数据库。
// 与 initUsageDataTestEnv 相比，这里额外迁移 LLMUsageLog 表，并保持 schema 干净。
func initInstancePKTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.AIModel{},
		&model.LLMUsageLog{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	db.Create(&model.SiteConfig{})
	cleanup := func() {
		origDB()
		AdminToken = origToken
		Store = origStore
	}
	return db, cleanup
}

// ─── resolveInstancePKFromParam 单元测试 ─────────────────────────────────

func TestResolveInstancePKFromParam_Empty(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	if got := resolveInstancePKFromParam(context.Background(), "", 0); got != 0 {
		t.Errorf("空字符串应返回 0，got %d", got)
	}
	if got := resolveInstancePKFromParam(context.Background(), "   ", 0); got != 0 {
		t.Errorf("纯空白应返回 0，got %d", got)
	}
}

func TestResolveInstancePKFromParam_NumericPK(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	// 纯数字直接当 DB 主键，不查表也不校验存在性
	if got := resolveInstancePKFromParam(context.Background(), "42", 0); got != 42 {
		t.Errorf("纯数字应直接当主键，got %d", got)
	}
	if got := resolveInstancePKFromParam(context.Background(), "  7  ", 0); got != 7 {
		t.Errorf("带空白的纯数字应被 trim 后当主键，got %d", got)
	}
}

func TestResolveInstancePKFromParam_CVMIDHit(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	inst := model.Instance{Name: "i", InstanceId: "ins-hit-001", UserID: 100}
	model.DB(context.Background()).Create(&inst)

	got := resolveInstancePKFromParam(context.Background(), "ins-hit-001", 0)
	if got != uint64(inst.ID) {
		t.Errorf("CVM ID 应反查到主键 %d，got %d", inst.ID, got)
	}
}

func TestResolveInstancePKFromParam_CVMIDMiss(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	if got := resolveInstancePKFromParam(context.Background(), "ins-not-exist", 0); got != 0 {
		t.Errorf("CVM ID 反查不到应返回 0，got %d", got)
	}
}

func TestResolveInstancePKFromParam_OwnerScoped(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	inst := model.Instance{Name: "i", InstanceId: "ins-owner-001", UserID: 100}
	model.DB(context.Background()).Create(&inst)

	// 同 owner → 命中
	if got := resolveInstancePKFromParam(context.Background(), "ins-owner-001", 100); got != uint64(inst.ID) {
		t.Errorf("同 owner 应反查到主键，got %d", got)
	}
	// 跨 owner → 视为不存在，避免越权
	if got := resolveInstancePKFromParam(context.Background(), "ins-owner-001", 999); got != 0 {
		t.Errorf("跨 owner 应返回 0（防越权），got %d", got)
	}
}

func TestResolveInstancePKFromParam_SoftDeletedVisible(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	inst := model.Instance{Name: "i", InstanceId: "ins-deleted", UserID: 100}
	model.DB(context.Background()).Create(&inst)
	model.DB(context.Background()).Delete(&inst) // 软删除

	// Unscoped 查询，软删除的实例也可被解析（保持历史用量统计能查到已销毁实例）
	if got := resolveInstancePKFromParam(context.Background(), "ins-deleted", 0); got != uint64(inst.ID) {
		t.Errorf("软删实例应仍可解析（Unscoped），got %d", got)
	}
}

// ─── resolveInstancePKFromIDOrParam 单元测试 ─────────────────────────────

func TestResolveInstancePKFromIDOrParam_BothEmpty(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	if got := resolveInstancePKFromIDOrParam(context.Background(), "", "", 0); got != 0 {
		t.Errorf("两参数全空应返回 0，got %d", got)
	}
}

func TestResolveInstancePKFromIDOrParam_IDPriorityOverInstanceID(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	inst := model.Instance{Name: "i", InstanceId: "ins-other", UserID: 100}
	model.DB(context.Background()).Create(&inst)

	// id=42 与 instance_id=ins-other（主键 != 42）同传，必须以 id 为准
	got := resolveInstancePKFromIDOrParam(context.Background(), "42", "ins-other", 100)
	if got != 42 {
		t.Errorf("双参数同传时 id 应优先生效，got %d", got)
	}
}

func TestResolveInstancePKFromIDOrParam_IDInvalidNoFallback(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	inst := model.Instance{Name: "i", InstanceId: "ins-fallback", UserID: 100}
	model.DB(context.Background()).Create(&inst)

	cases := []struct {
		name    string
		idStr   string
		wantOut uint64
	}{
		{"id=abc 非数字", "abc", 0},
		{"id=0 非法主键", "0", 0},
		{"id=-1 非法主键", "-1", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// 即使 instance_id 能反查命中，只要 id 字段被显式提供，就不应回退
			got := resolveInstancePKFromIDOrParam(context.Background(), c.idStr, "ins-fallback", 100)
			if got != c.wantOut {
				t.Errorf("idStr=%q：应返回 %d（不回退到 instance_id），got %d", c.idStr, c.wantOut, got)
			}
		})
	}
}

func TestResolveInstancePKFromIDOrParam_FallbackToInstanceIDWhenIDEmpty(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	inst := model.Instance{Name: "i", InstanceId: "ins-fb-empty", UserID: 100}
	model.DB(context.Background()).Create(&inst)

	// idStr="" → 完全退化到 instance_id 路径，CVM ID 字符串会反查
	got := resolveInstancePKFromIDOrParam(context.Background(), "", "ins-fb-empty", 100)
	if got != uint64(inst.ID) {
		t.Errorf("idStr 为空时应退化到 instance_id 反查，期望 %d，got %d", inst.ID, got)
	}

	// idStr="" + instance_id 为纯数字 → 直接当主键
	if got := resolveInstancePKFromIDOrParam(context.Background(), "", "999", 0); got != 999 {
		t.Errorf("idStr 空时 instance_id=纯数字应直接当主键，got %d", got)
	}
}

func TestResolveInstancePKFromIDOrParam_IDValidNumericPK(t *testing.T) {
	_, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	if got := resolveInstancePKFromIDOrParam(context.Background(), "12345", "", 0); got != 12345 {
		t.Errorf("有效 id 应直接当主键，got %d", got)
	}
}

// ─── HandleAdminUsageLogs：双参数兼容 e2e ─────────────────────────────────

func TestHandleAdminUsageLogs_FilterByID_DBPrimaryKey(t *testing.T) {
	db, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	now := time.Now()
	// 两个实例，仅注入第一个的 logs，校验过滤命中数量
	inst1 := model.Instance{Name: "i1", InstanceId: "ins-aaa", UserID: 1}
	inst2 := model.Instance{Name: "i2", InstanceId: "ins-bbb", UserID: 1}
	db.Create(&inst1)
	db.Create(&inst2)
	db.Create(&model.LLMUsageLog{InstanceID: inst1.ID, UserID: 1, Model: "m", Provider: "p", TotalTokens: 10, CreatedAt: now})
	db.Create(&model.LLMUsageLog{InstanceID: inst1.ID, UserID: 1, Model: "m", Provider: "p", TotalTokens: 20, CreatedAt: now})
	db.Create(&model.LLMUsageLog{InstanceID: inst2.ID, UserID: 1, Model: "m", Provider: "p", TotalTokens: 999, CreatedAt: now})

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/logs?id="+itoa(inst1.ID), nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleAdminUsageLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int64 `json:"total"`
		Logs  []any `json:"logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("按 id（DB 主键）过滤应只命中 2 条，got %d", resp.Total)
	}
}

func TestHandleAdminUsageLogs_FilterByInstanceID_CVMID(t *testing.T) {
	db, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	now := time.Now()
	inst := model.Instance{Name: "i1", InstanceId: "ins-cvm-only", UserID: 1}
	db.Create(&inst)
	db.Create(&model.LLMUsageLog{InstanceID: inst.ID, UserID: 1, Model: "m", Provider: "p", TotalTokens: 10, CreatedAt: now})
	// 干扰数据：另一个实例的日志不应被命中
	other := model.Instance{Name: "i2", InstanceId: "ins-other", UserID: 1}
	db.Create(&other)
	db.Create(&model.LLMUsageLog{InstanceID: other.ID, UserID: 1, Model: "m", Provider: "p", TotalTokens: 999, CreatedAt: now})

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/logs?instance_id=ins-cvm-only", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleAdminUsageLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int64 `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("按 instance_id（CVM ID）过滤应只命中 1 条，got %d", resp.Total)
	}
}

func TestHandleAdminUsageLogs_BothParams_IDWins(t *testing.T) {
	db, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	now := time.Now()
	a := model.Instance{Name: "a", InstanceId: "ins-A", UserID: 1}
	b := model.Instance{Name: "b", InstanceId: "ins-B", UserID: 1}
	db.Create(&a)
	db.Create(&b)
	db.Create(&model.LLMUsageLog{InstanceID: a.ID, UserID: 1, TotalTokens: 1, CreatedAt: now})
	db.Create(&model.LLMUsageLog{InstanceID: b.ID, UserID: 1, TotalTokens: 1, CreatedAt: now})
	db.Create(&model.LLMUsageLog{InstanceID: b.ID, UserID: 1, TotalTokens: 1, CreatedAt: now})

	// id 指向 b（2 条），instance_id 指向 ins-A（1 条）。期望以 id 为准 → 命中 2 条。
	req := httptest.NewRequest(http.MethodGet, "/admin/usage/logs?id="+itoa(b.ID)+"&instance_id=ins-A", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleAdminUsageLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int64 `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("双参数同传时应以 id 为准，期望命中 2 条，got %d", resp.Total)
	}
}

// ─── HandleQuotaLogs：双参数兼容 e2e（含越权防护） ───────────────────────

func TestHandleQuotaLogs_FilterByInstanceID_OwnerScoped(t *testing.T) {
	db, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	// 当前登录用户 alice
	alice := model.User{Username: "alice"}
	bob := model.User{Username: "bob"}
	db.Create(&alice)
	db.Create(&bob)

	// 同名 CVM ID 不可能存在两条，但 bob 名下另一个实例 b 用作干扰
	insA := model.Instance{Name: "ia", InstanceId: "ins-alice-1", UserID: alice.ID}
	insB := model.Instance{Name: "ib", InstanceId: "ins-bob-1", UserID: bob.ID}
	db.Create(&insA)
	db.Create(&insB)

	now := time.Now()
	db.Create(&model.LLMUsageLog{InstanceID: insA.ID, UserID: alice.ID, TotalTokens: 1, CreatedAt: now})
	db.Create(&model.LLMUsageLog{InstanceID: insA.ID, UserID: alice.ID, TotalTokens: 1, CreatedAt: now})
	db.Create(&model.LLMUsageLog{InstanceID: insB.ID, UserID: bob.ID, TotalTokens: 1, CreatedAt: now})

	// alice 用自己的 CVM ID → 命中 2 条
	w := callQuotaLogs(t, "/quota/logs?instance_id=ins-alice-1", "alice")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int64 `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("alice 自己的 instance_id 应命中 2 条，got %d", resp.Total)
	}

	// alice 试图用 bob 的 CVM ID 查询：CVM ID 反查会被 owner_user_id 过滤掉 → filterInstanceID=0 →
	// 退化为"不附加实例过滤"，但下游主查询有 user_id=alice.ID 兜底 → 期望命中 0 条 alice 的 bob 实例日志，
	// 但会命中 alice 自己的全部 2 条（因为 instance_id 过滤未生效，等同于不传该参数）。
	// 这正是设计意图：宁可范围放宽到自己的全部，也不允许越权命中他人数据。
	w2 := callQuotaLogs(t, "/quota/logs?instance_id=ins-bob-1", "alice")
	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200，got %d, body=%s", w2.Code, w2.Body.String())
	}
	var resp2 struct {
		Total int64 `json:"total"`
		Logs  []struct {
			ID uint `json:"id"`
		} `json:"logs"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	// 关键断言：返回的所有日志都必须属于 alice，绝不能泄露 bob 的日志。
	for _, l := range resp2.Logs {
		var lg model.LLMUsageLog
		if err := db.First(&lg, l.ID).Error; err != nil {
			t.Fatalf("反查日志失败: %v", err)
		}
		if lg.UserID != alice.ID {
			t.Errorf("越权检测：返回了非 alice 的日志 id=%d user_id=%d", lg.ID, lg.UserID)
		}
	}
}

func TestHandleQuotaLogs_FilterByID_NumericPK(t *testing.T) {
	db, cleanup := initInstancePKTestDB(t)
	defer cleanup()

	alice := model.User{Username: "alice"}
	db.Create(&alice)
	insA := model.Instance{Name: "ia", InstanceId: "ins-alice-1", UserID: alice.ID}
	insB := model.Instance{Name: "ib", InstanceId: "ins-alice-2", UserID: alice.ID}
	db.Create(&insA)
	db.Create(&insB)

	now := time.Now()
	db.Create(&model.LLMUsageLog{InstanceID: insA.ID, UserID: alice.ID, TotalTokens: 1, CreatedAt: now})
	db.Create(&model.LLMUsageLog{InstanceID: insB.ID, UserID: alice.ID, TotalTokens: 1, CreatedAt: now})
	db.Create(&model.LLMUsageLog{InstanceID: insB.ID, UserID: alice.ID, TotalTokens: 1, CreatedAt: now})

	w := callQuotaLogs(t, "/quota/logs?id="+itoa(insB.ID), "alice")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int64 `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("按 id 过滤应命中 2 条，got %d", resp.Total)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────

// callQuotaLogs 构造带登录态的 GET 请求并调用 HandleQuotaLogs。
func callQuotaLogs(t *testing.T, path, username string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	session, err := Store.Get(req, "hatchery-session")
	if err != nil {
		t.Fatalf("获取 session 失败: %v", err)
	}
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("保存 session 失败: %v", err)
	}
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	HandleQuotaLogs(w, req)
	return w
}

// itoa 复用 admin_smh_test.go 中已有的 helper。

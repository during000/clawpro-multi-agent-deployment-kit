package controller

// 本文件覆盖 hotfix/approve_device_bug_fix 分支中从 openclaw.go 提取出的两个辅助函数：
//   - extractInstanceIDOrCVMID(r) (id uint, instanceID string, err error)
//   - findInstanceByIDOrCVMID(ctx, userID, id, instanceID) (*model.Instance, error)
//
// 这两个函数被多处调用方（HandleOpenClawDetail / openclaw_env.go / admin_instances.go /
// memory_plan.go / getInstanceByIDRaw）共享，所以单独直测的回归保护价值较高。

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initIDExtractTestDB 初始化最小化内存 DB，仅迁移 Instance 与 User，足以覆盖
// findInstanceByIDOrCVMID 所需的 ownership/admin 两种查询模式。
func initIDExtractTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Instance{}, &model.CustomAgentType{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	cleanup := func() {
		origDB()
		Store = origStore
	}
	t.Cleanup(cleanup)
	return cleanup
}

// ─── extractInstanceIDOrCVMID ──────────────────────────────────────────────

// TestExtractInstanceIDOrCVMID_QueryID 验证 query 中传 id 时正确解析为 uint。
func TestExtractInstanceIDOrCVMID_QueryID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?id=42", nil)
	id, instID, err := extractInstanceIDOrCVMID(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != 42 {
		t.Errorf("expected id=42, got %d", id)
	}
	if instID != "" {
		t.Errorf("expected empty instanceID, got %q", instID)
	}
}

// TestExtractInstanceIDOrCVMID_FormID 验证 POST form-urlencoded body 中传 id 时被解析。
// 这条路径在 HandleApprove / 其他 POST 接口中复用。
func TestExtractInstanceIDOrCVMID_FormID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("id=7"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	id, instID, err := extractInstanceIDOrCVMID(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != 7 {
		t.Errorf("expected id=7, got %d", id)
	}
	if instID != "" {
		t.Errorf("expected empty instanceID, got %q", instID)
	}
}

// TestExtractInstanceIDOrCVMID_QueryInstanceID 验证仅传 instance_id 时 id=0、字符串透传。
func TestExtractInstanceIDOrCVMID_QueryInstanceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?instance_id=ins-abc123", nil)
	id, instID, err := extractInstanceIDOrCVMID(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != 0 {
		t.Errorf("expected id=0, got %d", id)
	}
	if instID != "ins-abc123" {
		t.Errorf("expected instance_id=ins-abc123, got %q", instID)
	}
}

// TestExtractInstanceIDOrCVMID_FormInstanceID 验证 POST body 中传 instance_id 时被解析。
func TestExtractInstanceIDOrCVMID_FormInstanceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("instance_id=ins-form"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	id, instID, err := extractInstanceIDOrCVMID(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != 0 {
		t.Errorf("expected id=0, got %d", id)
	}
	if instID != "ins-form" {
		t.Errorf("expected instance_id=ins-form, got %q", instID)
	}
}

// TestExtractInstanceIDOrCVMID_BothParams 验证同时传 id 与 instance_id 时
// 两者都被读取（优先级在调用方 findInstanceByIDOrCVMID 里处理，本函数只做解析）。
func TestExtractInstanceIDOrCVMID_BothParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?id=99&instance_id=ins-xyz", nil)
	id, instID, err := extractInstanceIDOrCVMID(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != 99 || instID != "ins-xyz" {
		t.Errorf("expected (id=99, instance_id=ins-xyz), got (%d, %q)", id, instID)
	}
}

// TestExtractInstanceIDOrCVMID_NoParams 验证两参数皆缺时返回 (0, "", nil)。
// 注意：本函数在二者皆空时不主动报错，把"必须有一个参数"的判定留给 findInstanceByIDOrCVMID
// 与上层 handler。这是 hotfix 对边界语义的明确约束。
func TestExtractInstanceIDOrCVMID_NoParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	id, instID, err := extractInstanceIDOrCVMID(req)
	if err != nil {
		t.Errorf("two-empty-params 不应报错（留给上层判定），实际 err=%v", err)
	}
	if id != 0 || instID != "" {
		t.Errorf("expected (0, \"\"), got (%d, %q)", id, instID)
	}
}

// TestExtractInstanceIDOrCVMID_InvalidIDString 验证 id 是非数字时返回错误。
func TestExtractInstanceIDOrCVMID_InvalidIDString(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?id=abc", nil)
	_, _, err := extractInstanceIDOrCVMID(req)
	if err == nil {
		t.Fatal("expected error for non-numeric id, got nil")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "无效的 id") {
		t.Errorf("expected '无效的 id' in error, got %q", err.Error())
	}
}

// TestExtractInstanceIDOrCVMID_IDZero 验证 id=0 是非法值，必须返回错误。
// 这是防御性检查：DB 主键 0 在业务里没有意义，避免 0 被当作"未传"处理。
func TestExtractInstanceIDOrCVMID_IDZero(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?id=0", nil)
	_, _, err := extractInstanceIDOrCVMID(req)
	if err == nil {
		t.Fatal("expected error for id=0, got nil")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "无效的 id") {
		t.Errorf("expected '无效的 id' in error, got %q", err.Error())
	}
}

// TestExtractInstanceIDOrCVMID_NegativeID 验证负数 id 也被 ParseUint 拒绝。
func TestExtractInstanceIDOrCVMID_NegativeID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?id=-1", nil)
	_, _, err := extractInstanceIDOrCVMID(req)
	if err == nil {
		t.Fatal("expected error for negative id, got nil")
	}
}

// TestExtractInstanceIDOrCVMID_QueryAndFormPriority 验证 query 优先于 form：
// query 没传 id 时才会尝试 form；query 传了 id（即使是非法值）也不会再 fallback 到 form。
func TestExtractInstanceIDOrCVMID_QueryAndFormPriority(t *testing.T) {
	// query 有合法 id，form 也有 id，但应只使用 query 的值
	req := httptest.NewRequest(http.MethodPost, "/x?id=11", strings.NewReader("id=22"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	id, _, err := extractInstanceIDOrCVMID(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != 11 {
		t.Errorf("query 应优先于 form，期望 id=11，实际=%d", id)
	}
}

// ─── findInstanceByIDOrCVMID ──────────────────────────────────────────────

// TestFindInstanceByIDOrCVMID_OwnershipByID 验证：userID > 0 时通过主键 id 命中
// 用户自己的实例。
func TestFindInstanceByIDOrCVMID_OwnershipByID(t *testing.T) {
	initIDExtractTestDB(t)

	user := &model.User{Username: "u-own", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "own-inst", InstanceId: "ins-own",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	got, err := findInstanceByIDOrCVMID(context.Background(), user.ID, inst.ID, "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.InstanceId != "ins-own" {
		t.Errorf("expected ins-own, got %s", got.InstanceId)
	}
}

// TestFindInstanceByIDOrCVMID_OwnershipByInstanceID 验证：userID > 0 时通过 CVM
// instance_id 命中用户自己的实例。
func TestFindInstanceByIDOrCVMID_OwnershipByInstanceID(t *testing.T) {
	initIDExtractTestDB(t)

	user := &model.User{Username: "u-own2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "own-inst-2", InstanceId: "ins-own-2",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	got, err := findInstanceByIDOrCVMID(context.Background(), user.ID, 0, "ins-own-2")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.ID != inst.ID {
		t.Errorf("expected db id=%d, got %d", inst.ID, got.ID)
	}
}

// TestFindInstanceByIDOrCVMID_OwnershipDeniesOtherUser 验证：跨用户访问被
// user_id 过滤拒绝，返回 ErrInstanceNotFound。
func TestFindInstanceByIDOrCVMID_OwnershipDeniesOtherUser(t *testing.T) {
	initIDExtractTestDB(t)

	owner := &model.User{Username: "u-owner-x", Password: "x", Role: "user"}
	intruder := &model.User{Username: "u-intruder-x", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(owner)
	model.DB(context.Background()).Create(intruder)
	inst := &model.Instance{
		Name: "victim", InstanceId: "ins-victim-x",
		UserID: owner.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	_, err := findInstanceByIDOrCVMID(context.Background(), intruder.ID, 0, "ins-victim-x")
	if err == nil {
		t.Fatal("intruder 应当无法获取 owner 的实例")
	}
	if !errors.Is(err, ErrInstanceNotFound) {
		t.Errorf("期望 ErrInstanceNotFound，实际=%v", err)
	}
}

// TestFindInstanceByIDOrCVMID_AdminModeByID 验证：userID == 0（admin 模式）下
// 不附加 user_id 过滤，可以拿到任意用户的实例。
func TestFindInstanceByIDOrCVMID_AdminModeByID(t *testing.T) {
	initIDExtractTestDB(t)

	owner := &model.User{Username: "u-adm-1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(owner)
	inst := &model.Instance{
		Name: "admin-inst", InstanceId: "ins-admin-1",
		UserID: owner.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	got, err := findInstanceByIDOrCVMID(context.Background(), 0 /*admin*/, inst.ID, "")
	if err != nil {
		t.Fatalf("admin 模式按 id 应能获取任意实例，实际 err=%v", err)
	}
	if got.InstanceId != "ins-admin-1" {
		t.Errorf("expected ins-admin-1, got %s", got.InstanceId)
	}
}

// TestFindInstanceByIDOrCVMID_AdminModeByInstanceID 验证：userID == 0 通过 instance_id 命中。
func TestFindInstanceByIDOrCVMID_AdminModeByInstanceID(t *testing.T) {
	initIDExtractTestDB(t)

	owner := &model.User{Username: "u-adm-2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(owner)
	inst := &model.Instance{
		Name: "admin-inst-2", InstanceId: "ins-admin-2",
		UserID: owner.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	got, err := findInstanceByIDOrCVMID(context.Background(), 0, 0, "ins-admin-2")
	if err != nil {
		t.Fatalf("admin 模式按 instance_id 应能获取任意实例，实际 err=%v", err)
	}
	if got.ID != inst.ID {
		t.Errorf("expected db id=%d, got %d", inst.ID, got.ID)
	}
}

// TestFindInstanceByIDOrCVMID_IDPriority 验证：同时传 id 与 instance_id 时 id 优先；
// instance_id 故意写一个不存在的值，若 instance_id 优先则会返回 NotFound。
func TestFindInstanceByIDOrCVMID_IDPriority(t *testing.T) {
	initIDExtractTestDB(t)

	user := &model.User{Username: "u-prio", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "prio", InstanceId: "ins-prio-real",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	got, err := findInstanceByIDOrCVMID(context.Background(), user.ID, inst.ID, "ins-not-exist")
	if err != nil {
		t.Fatalf("id 应优先，实际 err=%v", err)
	}
	if got.InstanceId != "ins-prio-real" {
		t.Errorf("expected ins-prio-real, got %s", got.InstanceId)
	}
}

// TestFindInstanceByIDOrCVMID_BothEmpty 验证：id=0 且 instanceID="" 时返回
// "缺少参数" 错误（不返回 ErrInstanceNotFound，避免 404 误导排查）。
func TestFindInstanceByIDOrCVMID_BothEmpty(t *testing.T) {
	initIDExtractTestDB(t)

	_, err := findInstanceByIDOrCVMID(context.Background(), 1, 0, "")
	if err == nil {
		t.Fatal("expected error for both-empty params, got nil")
	}
	if errors.Is(err, ErrInstanceNotFound) {
		t.Errorf("两参数皆空应返回参数错误而非 ErrInstanceNotFound，实际=%v", err)
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "缺少参数") {
		t.Errorf("expected '缺少参数' in error, got %q", err.Error())
	}
}

// TestFindInstanceByIDOrCVMID_NotFound 验证查询命中失败时返回 ErrInstanceNotFound 哨兵错误，
// 调用方可以用 errors.Is + instanceErrStatus 把它统一映射为 404。
func TestFindInstanceByIDOrCVMID_NotFound(t *testing.T) {
	initIDExtractTestDB(t)

	_, err := findInstanceByIDOrCVMID(context.Background(), 0, 0, "ins-no-such")
	if err == nil {
		t.Fatal("expected error for non-existent instance_id, got nil")
	}
	if !errors.Is(err, ErrInstanceNotFound) {
		t.Errorf("期望 ErrInstanceNotFound，实际=%v", err)
	}
}

// TestFindInstanceByIDOrCVMID_NotFoundByPK 验证按主键 id 查不到时也返回
// ErrInstanceNotFound（与 instance_id 路径语义对齐）。
func TestFindInstanceByIDOrCVMID_NotFoundByPK(t *testing.T) {
	initIDExtractTestDB(t)

	_, err := findInstanceByIDOrCVMID(context.Background(), 0, 99999, "")
	if err == nil {
		t.Fatal("expected ErrInstanceNotFound for unknown pk, got nil")
	}
	if !errors.Is(err, ErrInstanceNotFound) {
		t.Errorf("期望 ErrInstanceNotFound，实际=%v", err)
	}
}

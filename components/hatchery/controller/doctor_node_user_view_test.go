package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/model"
)

// 用户端"不展示龙虾医生"系列测试：列表、详情、状态、创建配额。

// makeUserAndDoctor 创建一个用户 + 1 个普通实例 + 1 个龙虾医生节点。
func makeUserAndDoctor(t *testing.T) (*model.User, *model.Instance, *model.Instance) {
	t.Helper()
	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 5}
	model.DB(context.Background()).Create(user)
	normal := &model.Instance{
		Name: "normal", InstanceId: "ins-normal",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		IsDoctorNode: false,
	}
	model.DB(context.Background()).Create(normal)
	doctor := &model.Instance{
		Name: "doctor", InstanceId: "ins-doctor",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		IsDoctorNode: true,
	}
	model.DB(context.Background()).Create(doctor)
	return user, normal, doctor
}

// TestHandleInstanceList_ExcludesDoctorNode 用户端实例列表不应包含龙虾医生节点。
func TestHandleInstanceList_ExcludesDoctorNode(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	_, normal, doctor := makeUserAndDoctor(t)

	req := jsonReqWithSession(t, http.MethodGet, "/openclaw/list", "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, fmt.Sprintf("\"ID\":%d", normal.ID)) {
		t.Errorf("响应应包含普通实例 ID %d，实际=%s", normal.ID, body)
	}
	if strings.Contains(body, fmt.Sprintf("\"ID\":%d", doctor.ID)) {
		t.Errorf("响应不应包含龙虾医生节点 ID %d，实际=%s", doctor.ID, body)
	}
}

// TestHandleInstanceStatus_ExcludesDoctorNode 用户端实例状态接口访问龙虾医生节点应失败。
func TestHandleInstanceStatus_ExcludesDoctorNode(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	_, _, doctor := makeUserAndDoctor(t)

	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?id=%d", doctor.ID), "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	// 龙虾医生节点应被排除 → 不应返回 200 OK
	if rr.Code == http.StatusOK {
		t.Errorf("龙虾医生节点状态查询不应返回 200，实际=%d body=%s",
			rr.Code, rr.Body.String())
	}
}

// TestHandleInstanceStatus_NormalInstanceOK 普通实例 status 接口应能正常返回。
// 用作 ExcludesDoctorNode 的对照组，证明排除条件没误伤普通实例。
func TestHandleInstanceStatus_NormalInstanceOK(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	_, normal, _ := makeUserAndDoctor(t)

	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?id=%d", normal.ID), "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	// 实例存在 → 不应是 not-found 类错误（具体业务可能因为 CVM 状态返回其他码，
	// 但"实例不存在"这种 404/400 不应出现）
	if strings.Contains(rr.Body.String(), "实例不存在") {
		t.Errorf("普通实例不应被识别为不存在，实际=%s", rr.Body.String())
	}
}

// TestHandleInstanceStatus_LocalInstance_NoCVMCall 本地实例调 /openclaw/status 不应
// 调 CVM API。本地实例的 InstanceId 不是 CVM 格式（如 "local-workbuddy-001"），
// 以前会被 fetchCVMInstanceInfo 报「实例ID不合要求」。修复后应走
// resolveLocalInstanceStatus，返回 online/offline。
func TestHandleInstanceStatus_LocalInstance_NoCVMCall(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceInfo{}); err != nil {
		t.Fatalf("migrate LocalInstanceInfo: %v", err)
	}

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 5}
	model.DB(ctx).Create(user)

	inst := &model.Instance{
		Name: "local-workbuddy", InstanceId: "local-workbuddy-001",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		Source: model.InstanceSourceLocal,
	}
	model.DB(ctx).Create(inst)

	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	// 本地实例不走 CVM API，只看 last_report_at；这里没创建 LocalInstanceInfo
	// 记录 → 走 stopped 分支。关键是不应报「实例ID不合要求」，也不应 500。
	// 方案 A：本地实例复用 running/stopped 状态枚举（不再用 online/offline）。
	if rr.Code != http.StatusOK {
		t.Fatalf("本地实例 status 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if strings.Contains(body, "实例ID") || strings.Contains(body, "不合要求") {
		t.Errorf("本地实例不应报 CVM ID 格式错，实际=%s", body)
	}
	if !strings.Contains(body, `"status":"stopped"`) {
		t.Errorf("本地实例无上报记录时应返回 stopped，实际=%s", body)
	}
	// actions 必须裁剪为空（hatchery 无法对本地机器 reboot/reinstall/terminal，
	// delete 也不允许——需用户本地卸载 agent 后 hatchery 被动超时清理）
	if !strings.Contains(body, `"actions":[]`) {
		t.Errorf("本地实例 actions 应为空数组，实际=%s", body)
	}
}

// TestHandleInstanceStatus_LocalInstance_RunningWhenRecent 本地实例有近期心跳上报时
// 返回 status=running（复用 CVM 状态枚举），actions 仍裁剪为空；
// label/tooltip 根据 Accept-Language 在 zh/en 两套文案之间切换。
func TestHandleInstanceStatus_LocalInstance_RunningWhenRecent(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceInfo{}); err != nil {
		t.Fatalf("migrate LocalInstanceInfo: %v", err)
	}

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 5}
	model.DB(ctx).Create(user)

	inst := &model.Instance{
		Name: "local-fresh", InstanceId: "local-fresh-001",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		Source: model.InstanceSourceLocal,
	}
	model.DB(ctx).Create(inst)

	now := time.Now()
	info := &model.LocalInstanceInfo{
		InstanceID:   inst.ID,
		HostName:     "alex-mbp",
		LastReportAt: &now,
	}
	model.DB(ctx).Create(info)

	// 中文分支：将 TenantSnapshot{DefaultLang:"zh"} 注入 ctx（绕过 middleware 直接调 handler 时手动拼），label 文案应为中文
	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?id=%d", inst.ID), "u1", "")
	req = req.WithContext(hcommon.InjectTenant(req.Context(), hcommon.TenantSnapshot{DefaultLang: "zh"}))
	rr := httptest.NewRecorder()
	HandleInstanceStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("本地实例 status 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"status":"running"`) {
		t.Errorf("近期心跳上报的本地实例应返回 running，实际=%s", body)
	}
	if !strings.Contains(body, `"actions":[]`) {
		t.Errorf("本地实例无论 running 还是 stopped，actions 都应为空数组，实际=%s", body)
	}
	if !strings.Contains(body, `"label":"运行中"`) {
		t.Errorf("zh 分支 label 应为「运行中」，实际=%s", body)
	}

	// 英文分支：DefaultLang="en"，label 文案应为英文（回归 #3 修复的 hardcode 问题）
	reqEn := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?id=%d", inst.ID), "u1", "")
	reqEn = reqEn.WithContext(hcommon.InjectTenant(reqEn.Context(), hcommon.TenantSnapshot{DefaultLang: "en"}))
	rrEn := httptest.NewRecorder()
	HandleInstanceStatus(rrEn, reqEn)
	if rrEn.Code != http.StatusOK {
		t.Fatalf("en 分支应 200，实际=%d body=%s", rrEn.Code, rrEn.Body.String())
	}
	bodyEn := rrEn.Body.String()
	if !strings.Contains(bodyEn, `"label":"Running"`) {
		t.Errorf("en 分支 label 应为 Running（而非硬编码中文），实际=%s", bodyEn)
	}
}

// TestHandleCreateInstance_QuotaExcludesDoctorNode 创建实例配额校验不算龙虾医生节点。
//
// 场景：用户配额 = 2，已有 2 个龙虾医生节点 + 0 个普通实例。
//
//	旧逻辑（包含龙虾医生）：count=2 ≥ 配额 2 → 拒绝创建
//	新逻辑（排除龙虾医生）：普通实例 count=0 < 2 → 应放行
//
// 这里只验证"配额校验阶段"不被龙虾医生节点撑爆，不关心后续的镜像/CVM 创建步骤。
func TestHandleCreateInstance_QuotaExcludesDoctorNode(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 2}
	model.DB(context.Background()).Create(user)
	// 2 个龙虾医生节点占满"看似配额"
	for i := 0; i < 2; i++ {
		d := &model.Instance{
			Name:         fmt.Sprintf("doctor-%d", i),
			InstanceId:   fmt.Sprintf("ins-doctor-%d", i),
			UserID:       user.ID,
			AgentType:    model.AgentTypeOpenClaw,
			IsDoctorNode: true,
		}
		model.DB(context.Background()).Create(d)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"name":       "new-instance",
		"agent_type": string(model.AgentTypeOpenClaw),
	})
	req := httptest.NewRequest(http.MethodPost, "/openclaw/create",
		strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// 注入登录 session
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "u1"
	tmpRR := httptest.NewRecorder()
	session.Save(req, tmpRR)
	for _, cookie := range tmpRR.Result().Cookies() {
		req.AddCookie(cookie)
	}

	rr := httptest.NewRecorder()
	HandleCreateInstance(rr, req)

	// 关键断言：错误信息不应包含"配额"/"quota_exceeded"。
	// 后续会因为没有可用镜像等原因失败，但绝不能是"配额已达上限"。
	body2 := rr.Body.String()
	if strings.Contains(body2, "配额已达上限") || strings.Contains(body2, "quota_exceeded") {
		t.Errorf("配额校验不应包含龙虾医生节点导致拒绝，实际=%s", body2)
	}
}

// TestAdminUsersInstCount_ExcludesDoctorNode admin/users 列表中 instance_count 不算龙虾医生。
func TestAdminUsersInstCount_ExcludesDoctorNode(t *testing.T) {
	t.Skip("该场景由同名的 admin_users_instance_count_test.go 负责")
}

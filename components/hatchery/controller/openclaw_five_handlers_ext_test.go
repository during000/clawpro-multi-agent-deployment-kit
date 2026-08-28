package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// ---------------------------------------------------------------------------
// HandleCreateInstance
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_MethodNotAllowed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/create", nil)
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleCreateInstance_Unauthorized(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("name", "x")
	req := httptest.NewRequest(http.MethodPost, "/openclaw/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleCreateInstance_EmptyName(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("name", "   ")
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("名称为空应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateInstance_TooLongName(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("name", strings.Repeat("a", 200))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("名称超长应 400，实际=%d", rr.Code)
	}
}

func TestHandleCreateInstance_InvalidRoleID(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("name", "inst-x")
	form.Set("role_id", "9999") // 不存在
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 role_id 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "角色") {
		t.Errorf("错误信息应包含'角色'，实际=%s", rr.Body.String())
	}
}

func TestHandleCreateInstance_InvalidAgentType(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("name", "inst-bad-type")
	form.Set("agent_type", "unknown-type-xxx")
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 agent_type 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateInstance_NoEnabledImage(t *testing.T) {
	// 未配置任何启用镜像 → 返回 400 "管理员尚未为 ... 类型配置生效镜像"
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("name", "no-image")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("无启用镜像应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "镜像") {
		t.Errorf("错误信息应包含'镜像'，实际=%s", rr.Body.String())
	}
}

func TestHandleCreateInstance_QuotaExceeded(t *testing.T) {
	// InstanceQuota=1 且已有 1 个实例 → count >= quota → 命中 quota_exceeded
	// 注：User.InstanceQuota 在 GORM 中有 default:1，无法写入 0，因此用 "1+占满" 构造
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 1}
	model.DB(context.Background()).Create(user)

	// 预先占满配额（1 个已有实例）
	existing := &model.Instance{
		Name: "existing", InstanceId: "ins-existing",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(existing)

	// 配置一个启用镜像以便越过前置校验
	img := &model.AIImage{
		ImageId:      "img-quota",
		ImageName:    "quota",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(img)

	form := url.Values{}
	form.Set("name", "quota-exceed")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("配额超限应 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "上限") {
		t.Errorf("错误信息应含'上限'，实际=%s", rr.Body.String())
	}
}

func TestHandleCreateInstance_QuotaExcludesDeletingInstances(t *testing.T) {
	// 验证 current_operation="delete" 的实例不计入配额
	t.Run("global_quota", func(t *testing.T) {
		// InstanceQuota=1，已有 1 个 deleting 实例 → 不应计入配额
		cleanup := initFiveHandlersTestDB(t)
		defer cleanup()

		user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 1}
		model.DB(context.Background()).Create(user)

		now := time.Now()
		existing := &model.Instance{
			Name: "deleting", InstanceId: "ins-deleting",
			UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
			CurrentOperation:          model.OpDelete,
			CurrentOperationUpdatedAt: &now,
		}
		model.DB(context.Background()).Create(existing)

		img := &model.AIImage{
			ImageId: "img-del-g", ImageName: "del-g", ImageType: "PRIVATE_IMAGE",
			AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
		}
		model.DB(context.Background()).Create(img)

		form := url.Values{}
		form.Set("name", "new-after-delete")
		form.Set("agent_type", model.AgentTypeOpenClaw)
		req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
		rr := httptest.NewRecorder()

		HandleCreateInstance(rr, req)

		if rr.Code == http.StatusForbidden {
			t.Errorf("deleting 实例不应计入配额，但收到 403: %s", rr.Body.String())
		}
	})

	t.Run("group_quota", func(t *testing.T) {
		// groupID>0 分支：分组配额=1，该组下已有 1 个 deleting 实例 → 不应计入
		cleanup := initFiveHandlersTestDB(t)
		defer cleanup()

		user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 5}
		model.DB(context.Background()).Create(user)

		// 创建分组 + closure 自引用
		group := &model.UserGroup{Name: "testgroup", FullPath: "testgroup", Source: "manual"}
		model.DB(context.Background()).Create(group)
		model.DB(context.Background()).Create(&model.GroupClosure{
			AncestorID: group.ID, DescendantID: group.ID, Depth: 0,
		})

		// 设置分组配额=1（通过 site_config 默认值，ResolvePolicyIntForGroup fallback）
		model.DB(context.Background()).Create(&model.SiteConfig{
			Name: "test", DefaultInstanceQuota: 1,
		})

		now := time.Now()
		existing := &model.Instance{
			Name: "deleting-grp", InstanceId: "ins-deleting-grp",
			UserID: user.ID, GroupID: group.ID, AgentType: model.AgentTypeOpenClaw,
			CurrentOperation:          model.OpDelete,
			CurrentOperationUpdatedAt: &now,
		}
		model.DB(context.Background()).Create(existing)

		img := &model.AIImage{
			ImageId: "img-del-grp", ImageName: "del-grp", ImageType: "PRIVATE_IMAGE",
			AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
		}
		model.DB(context.Background()).Create(img)

		form := url.Values{}
		form.Set("name", "new-grp-after-delete")
		form.Set("agent_type", model.AgentTypeOpenClaw)
		form.Set("group_id", fmt.Sprintf("%d", group.ID))
		req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
		rr := httptest.NewRecorder()

		HandleCreateInstance(rr, req)

		if rr.Code == http.StatusForbidden {
			t.Errorf("分组内 deleting 实例不应计入配额，但收到 403: %s", rr.Body.String())
		}
	})
}

// ---------------------------------------------------------------------------
// HandleInstanceStatus
// ---------------------------------------------------------------------------

func TestHandleInstanceStatus_Unauthorized(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/status?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleInstanceStatus_InvalidID_JSON(t *testing.T) {
	// 非法 id：返回 400 Bad Request
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := jsonReqWithSession(t, http.MethodGet, "/openclaw/status?id=abc", "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 id 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleInstanceStatus_InstanceNotFound(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := jsonReqWithSession(t, http.MethodGet, "/openclaw/status?id=9999", "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应返回 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleInstanceStatus_EmptyCVMId_JSON(t *testing.T) {
	// InstanceId 为空 → fetchCVMInstanceInfo 返回 (nil,nil)，不会访问外部 CVM API
	// 走到 ResolveInstanceStatus → handleStatusSideEffects → JSON 响应
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "status-empty", InstanceId: "",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("空 CVM id 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "status") {
		t.Errorf("响应应包含 status 字段，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleInstanceStatus：软删实例 (deleted_at 非空) 兜底
// ---------------------------------------------------------------------------

func TestHandleInstanceStatus_SoftDeletedInstance_Destroyed(t *testing.T) {
	// 自有实例被软删后查询状态 → 200 + status=destroyed + transient=false + actions=[]
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "soft-del", InstanceId: "ins-soft-del",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 软删实例
	model.DB(context.Background()).Delete(inst)

	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("软删自有实例应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"status":"destroyed"`) {
		t.Errorf("响应应包含 status=destroyed，实际=%s", body)
	}
	if !strings.Contains(body, `"transient":false`) {
		t.Errorf("destroyed 应为终态 transient=false，实际=%s", body)
	}
	if !strings.Contains(body, `"actions":[]`) {
		t.Errorf("destroyed 状态 actions 应为空数组，实际=%s", body)
	}
}

func TestHandleInstanceStatus_SoftDeletedInstance_ByInstanceId(t *testing.T) {
	// 通过 instance_id（非 id）查询软删实例 → 200 + destroyed
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "soft-del-cvm", InstanceId: "ins-by-instance",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Delete(inst)

	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?instance_id=%s", inst.InstanceId), "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("通过 instance_id 查软删实例应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"status":"destroyed"`) {
		t.Errorf("响应应包含 status=destroyed，实际=%s", rr.Body.String())
	}
}

func TestHandleInstanceStatus_SoftDeletedInstance_WrongUser(t *testing.T) {
	// 他人的软删实例 → 404（防 IDOR）
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	userA := &model.User{Username: "userA", Password: "x", Role: "user"}
	userB := &model.User{Username: "userB", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(userA)
	model.DB(context.Background()).Create(userB)
	inst := &model.Instance{
		Name: "soft-del-other", InstanceId: "ins-other",
		UserID: userA.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Delete(inst)

	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?id=%d", inst.ID), "userB", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("他人软删实例应 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleInstanceStatus_SoftDeletedInstance_MultipleDeleted_Latest(t *testing.T) {
	// 同一 instance_id 有多条软删记录 → 返回 id 最大（最新）那条
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建 → 软删 → 再创建同名 → 再软删（模拟重建后再次删除）
	inst1 := &model.Instance{
		Name: "multi-del", InstanceId: "ins-multi-del",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst1)
	model.DB(context.Background()).Delete(inst1)

	inst2 := &model.Instance{
		Name: "multi-del-v2", InstanceId: "ins-multi-del",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst2)
	model.DB(context.Background()).Delete(inst2)

	// 通过 instance_id 查询 → 应返回 id 更大的 inst2
	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?instance_id=%s", inst2.InstanceId), "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("多软删记录应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"status":"destroyed"`) {
		t.Errorf("响应应包含 status=destroyed，实际=%s", rr.Body.String())
	}
}

// findDeletedInstanceForStatus 直接单测：覆盖 defensive return nil 分支
func TestFindDeletedInstanceForStatus_NoParams(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 无 id 且无 instance_id → extractInstanceIDOrCVMID 返回 (0,"",nil)
	// → findDeletedInstanceForStatus 命中 "id==0 && instanceID=="" 返回 nil
	req := jsonReqWithSession(t, http.MethodGet, "/openclaw/status", "u1", "")
	rr := httptest.NewRecorder()

	inst := findDeletedInstanceForStatus(req, user)
	if inst != nil {
		t.Errorf("无 id/instance_id 应返回 nil，实际返回 %+v", inst)
	}
	_ = rr
}

// ---------------------------------------------------------------------------
// HandleRebootInstance
// ---------------------------------------------------------------------------

func TestHandleRebootInstance_MethodNotAllowed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/reboot", nil)
	rr := httptest.NewRecorder()

	handleRebootInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleRebootInstance_Unauthorized(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/reboot", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	handleRebootInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleRebootInstance_MissingID(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reboot", "u1", "")
	rr := httptest.NewRecorder()

	handleRebootInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 id 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRebootInstance_InstanceNotFound(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("id", "9999")
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reboot", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleRebootInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRebootInstance_NoCVMId(t *testing.T) {
	// 实例存在但 InstanceId 为空 → 400 "该实例无关联的 CVM"
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reboot-empty", InstanceId: "",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reboot", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleRebootInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("无 CVM id 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "CVM") {
		t.Errorf("错误信息应包含 CVM，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleResetInstance（补充未覆盖分支）
// ---------------------------------------------------------------------------

func TestHandleResetInstance_MissingID(t *testing.T) {
	// 已登录但未带 id → 400
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", "")
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 id 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleResetInstance_InstanceNotFound(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("id", "9999")
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleRetryInstance
// ---------------------------------------------------------------------------

func TestHandleRetryInstance_MethodNotAllowed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/retry", nil)
	rr := httptest.NewRecorder()

	HandleRetryInstance(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleRetryInstance_Unauthorized(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/retry", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleRetryInstance(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleRetryInstance_InvalidID(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("id", "not-a-number")
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/retry", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleRetryInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 id 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRetryInstance_InstanceNotFound(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("id", "9999")
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/retry", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleRetryInstance(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRetryInstance_StatusNotLoadFailed(t *testing.T) {
	// 实例存在但 InstanceId 为空 + 无 CurrentOperation → 状态为 creating，不是 load_failed → 400
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "retry-creating", InstanceId: "",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	snap := hcommon.TenantSnapshot{DefaultLang: "zh"}
	ctx := hcommon.InjectTenant(context.Background(), snap)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/retry", "u1", form.Encode())
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	HandleRetryInstance(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("非 load_failed 状态应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "创建中") {
		t.Errorf("错误信息应包含状态描述，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// 通过注入 NewCVMClient mock 覆盖"创建 CVM 客户端失败"等错误分支
// ---------------------------------------------------------------------------

// withFailingCVMClient 将 NewCVMClient 替换为始终返回 error 的 mock，用于命中
// HandleRebootInstance / HandleResetInstance / HandleRetryInstance 中
// "创建 CVM 客户端失败" 日志分支，以及 HandleInstanceStatus 中
// "查询 CVM 信息失败" 日志分支（fetchCVMInstanceInfo 内部会先调 NewCVMClient）。
//
// 返回 cleanup 函数，恢复原始实现，避免影响其他测试用例。
func withFailingCVMClient(t *testing.T) func() {
	t.Helper()
	orig := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return nil, hcommon.I18nError(i18n.MsgOperationFailed)
	}
	return func() {
		NewCVMClient = orig
	}
}

// ---------------------------------------------------------------------------
// HandleInstanceStatus：非空 CVM id → fetchCVMInstanceInfo 返回 err → 500
// ---------------------------------------------------------------------------

func TestHandleInstanceStatus_CVMFetchFailed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restoreCVM := withFailingCVMClient(t)
	defer restoreCVM()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "status-fetch-fail", InstanceId: "ins-existed",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := jsonReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/status?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceStatus(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("CVM 查询失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleRebootInstance：命中"创建 CVM 客户端失败"
// ---------------------------------------------------------------------------

func TestHandleRebootInstance_CVMClientFailed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restoreCVM := withFailingCVMClient(t)
	defer restoreCVM()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reboot-cvm-fail", InstanceId: "ins-has-cvm",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reboot", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleRebootInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("创建 CVM 客户端失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleRebootInstance：并发冲突（已有其他 current_operation 进行中）
// ---------------------------------------------------------------------------

func TestHandleRebootInstance_ConcurrentConflict(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	// 预置一个"正在删除"的实例：setOperationWithAgentReset(OpReboot) 因
	// current_operation != '' && != 'reboot' → RowsAffected=0 → 返回 ErrOperationInProgress
	inst := &model.Instance{
		Name: "reboot-busy", InstanceId: "ins-busy",
		UserID:                user.ID,
		AgentType:             model.AgentTypeOpenClaw,
		CurrentOperation:      model.OpDelete,
		CurrentOperationState: model.OpStateProcessing,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reboot", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleRebootInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusConflict {
		t.Errorf("并发冲突应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleResetInstance：不支持重装的 agent_type（hermes）
// ---------------------------------------------------------------------------

func TestHandleResetInstance_TypeNotSupportReinstall(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reset-unknown-type", InstanceId: "ins-unknown",
		UserID: user.ID,
		// 用一个在 agentTypesMap 中不存在的类型，使 AgentTypeSupportsReinstall
		// 返回 false，触发 checkInstanceSupportsReinstall → 403 "xxx 类型实例不支持重装"
		AgentType: "unknown-agent-type-xyz",
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("不支持重装应 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "不支持重装") {
		t.Errorf("错误信息应含'不支持重装'，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleResetInstance：无关联 CVM
// ---------------------------------------------------------------------------

func TestHandleResetInstance_NoCVMId(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reset-empty-cvm", InstanceId: "",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("无 CVM 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleResetInstance：并发冲突
// ---------------------------------------------------------------------------

func TestHandleResetInstance_ConcurrentConflict(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reset-busy", InstanceId: "ins-busy-reset",
		UserID:                user.ID,
		AgentType:             model.AgentTypeOpenClaw,
		CurrentOperation:      model.OpDelete,
		CurrentOperationState: model.OpStateProcessing,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusConflict {
		t.Errorf("并发冲突应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleResetInstance：未找到该类型的启用镜像（通过 guard 后走到 GetEnabledImageByType）
// ---------------------------------------------------------------------------

func TestHandleResetInstance_NoEnabledImage(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reset-no-image", InstanceId: "ins-reset",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	// 不创建任何 AIImage → GetEnabledImageByType 返回 nil,nil
	// 命中 "未找到该类型的启用镜像" 分支

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("无启用镜像应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "镜像") {
		t.Errorf("错误信息应含'镜像'，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleResetInstance：走到 NewCVMClient 失败（已有启用镜像 + 已越过 guard）
// ---------------------------------------------------------------------------

func TestHandleResetInstance_CVMClientFailed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restoreCVM := withFailingCVMClient(t)
	defer restoreCVM()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reset-cvm-fail", InstanceId: "ins-reset-cvm",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	img := &model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(img)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("创建 CVM 客户端失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleRetryInstance：load_failed 状态下 reboot 路径 → NewCVMClient 失败
// ---------------------------------------------------------------------------

func TestHandleRetryInstance_LoadFailed_CVMClientFailed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restoreCVM := withFailingCVMClient(t)
	defer restoreCVM()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	// 构造 load_failed：CurrentOperationState=failed + CurrentOperation 非空且非 delete
	inst := &model.Instance{
		Name: "retry-load-failed", InstanceId: "", // 空 InstanceId → fetchCVMInstanceInfo 直接 (nil,nil)，不会调 CVM
		UserID:                user.ID,
		AgentType:             model.AgentTypeOpenClaw,
		CurrentOperation:      model.OpCreate,
		CurrentOperationState: model.OpStateFailed,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/retry", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleRetryInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("重试时 CVM 客户端失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleRetryInstance：reinstall 路径 + 未启用任何镜像 → 500
// ---------------------------------------------------------------------------

func TestHandleRetryInstance_Reinstall_NoImage(t *testing.T) {
	// origin_operation=reinstall + operation_state=failed → load_failed + retry 走 OpReinstall
	// 此时 NewCVMClient 必须先成功（否则提前短路），再执行 GetEnabledImage → nil → 500
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	// 用一个"成功但返回的 client 永远不会被真正调用"的 mock：
	// 因为我们预置了 InstanceId='' 会让 fetchCVMInstanceInfo 直接返回 (nil,nil)，
	// 之后进入 reinstall 分支，GetEnabledImage 返回 nil，即 400/500 前即短路，
	// 不会触达 client.ResetInstance。
	orig := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		// 返回一个非 nil 但未连接的 client。后续不会被实际调用。
		return &cvm.Client{}, nil
	}
	defer func() { NewCVMClient = orig }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "retry-reinstall-no-image", InstanceId: "",
		UserID:                user.ID,
		AgentType:             model.AgentTypeOpenClaw,
		CurrentOperation:      model.OpReinstall,
		CurrentOperationState: model.OpStateFailed,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/retry", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleRetryInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("未启用任何镜像应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "镜像") {
		t.Errorf("错误信息应含'镜像'，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleRebootInstance：NewCVMClient 成功但 RebootInstances 因无凭据失败 → 500
// 命中 "调用 CVM RebootInstances" + "CVM 重启失败" 日志行
// ---------------------------------------------------------------------------

func TestHandleRebootInstance_RebootInstancesFailed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	// 使用空 client：调用 RebootInstances/ResetInstance 时 SDK 会直接返回
	// "require credential" 错误，不会发起真正的网络请求。
	orig := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return &cvm.Client{}, nil
	}
	defer func() { NewCVMClient = orig }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reboot-cvm-invoke-fail", InstanceId: "ins-xxx",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reboot", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleRebootInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("调用 CVM 失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "重启") {
		t.Errorf("错误信息应含'重启'，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleResetInstance：NewCVMClient 成功 + 有启用镜像 + SkillHub="" 跳过 UserData →
// 调 ResetInstance 因无凭据失败 → 500
// 命中 "调用 CVM ResetInstance" + "CVM 重装失败" 日志行
// ---------------------------------------------------------------------------

func TestHandleResetInstance_ResetInstanceFailed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	// 该用例触发失败路径会启动 go createErrorNotification goroutine。
	// 必须先等它使用完 model.DB(context.Background()) 再恢复，避免 cleanup 后 goroutine 访问 nil DB panic。
	defer time.Sleep(100 * time.Millisecond)

	orig := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return &cvm.Client{}, nil
	}
	defer func() { NewCVMClient = orig }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reset-invoke-fail", InstanceId: "ins-reset-x",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	img := &model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(img)

	// 确保 SiteConfig.SkillHub 为空以跳过 UserData 渲染（initFiveHandlersTestDB 未创建 SiteConfig
	// → GetSiteConfig 返回零值 SiteConfig → SkillHub==""）。

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("调用 CVM 重装失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "重装") {
		t.Errorf("错误信息应含'重装'，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleResetInstance：SkillHub 非空 + LoadScript 返回 error → 渲染 UserData 失败 → 500
// 命中 "渲染 UserData 失败" 日志行
// ---------------------------------------------------------------------------

func TestHandleResetInstance_UserDataRenderFailed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	origCVM := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return &cvm.Client{}, nil
	}
	defer func() { NewCVMClient = origCVM }()

	// 替换 LoadScript 使 init.sh 加载失败 → renderUserData 返回 err
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgRoleNotFound)
	}
	defer func() { LoadScript = origLoader }()

	// 预置 SiteConfig 使 SkillHub 非空以进入 UserData 渲染分支
	model.DB(context.Background()).Create(&model.SiteConfig{SkillHub: "https://hub.example.com"})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "reset-userdata-fail", InstanceId: "ins-reset-udf",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	img := &model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(img)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", form.Encode())
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("UserData 渲染失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleRetryInstance：load_failed + OpReboot（非 create/reinstall） + 调用 CVM 失败
// 命中 "调用 CVM RebootInstances" + "CVM 重启失败" 日志行（OpReboot 分支）
// ---------------------------------------------------------------------------

func TestHandleRetryInstance_Reboot_CVMFailed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	orig := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return &cvm.Client{}, nil
	}
	defer func() { NewCVMClient = orig }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	// OpCreate + failed → load_failed；retry 中 OpCreate → OpReboot
	inst := &model.Instance{
		Name: "retry-reboot-fail", InstanceId: "ins-retry-r",
		UserID:                user.ID,
		AgentType:             model.AgentTypeOpenClaw,
		CurrentOperation:      model.OpCreate,
		CurrentOperationState: model.OpStateFailed,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/retry", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleRetryInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Retry Reboot 调用 CVM 失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "重启") {
		t.Errorf("错误信息应含'重启'，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleRetryInstance：load_failed + OpReinstall + 有启用镜像 + 调用 CVM 失败
// 命中 "调用 CVM ResetInstance" + "CVM 重装失败" 日志行（Reinstall 分支）
// ---------------------------------------------------------------------------

func TestHandleRetryInstance_Reinstall_CVMFailed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	orig := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return &cvm.Client{}, nil
	}
	defer func() { NewCVMClient = orig }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "retry-reinstall-fail", InstanceId: "ins-retry-ri",
		UserID:                user.ID,
		AgentType:             model.AgentTypeOpenClaw,
		CurrentOperation:      model.OpReinstall,
		CurrentOperationState: model.OpStateFailed,
	}
	model.DB(context.Background()).Create(inst)
	img := &model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(img)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/retry", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleRetryInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Retry Reinstall 调用 CVM 失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "重装") {
		t.Errorf("错误信息应含'重装'，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HandleCreateInstance：事务成功创建占位 → 后续 CVMTemplate 为空 → 500 + 清理占位
// 命中 "占位记录创建成功" + "创建失败，清理占位记录" 两条日志
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_NoCVMTemplate_CleansPlaceholder(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	// 配置一个启用镜像以越过前置校验，让流程进入事务创建占位记录
	img := &model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(img)

	// SiteConfig 未写入 → GetSiteConfig 返回零值 → CVMTemplate==""
	// 占位事务会成功（命中"占位记录创建成功"日志），然后 CVMTemplate 校验失败
	// → writeError 500 → defer 清理占位（命中"创建失败，清理占位记录"）

	form := url.Values{}
	form.Set("name", "inst-no-tpl")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("CVMTemplate 为空应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "CVM") {
		t.Errorf("错误信息应含 CVM，实际=%s", rr.Body.String())
	}
	// 验证占位记录已被清理
	var count int64
	model.DB(context.Background()).Model(&model.Instance{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Errorf("占位记录应被清理，剩余=%d", count)
	}
}

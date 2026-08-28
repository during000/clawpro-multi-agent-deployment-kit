package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/model"
)

// TestCreateInstance_UngroupedUser_GroupRole_Forbidden 未分组用户选择分组角色时被拒绝
func TestCreateInstance_UngroupedUser_GroupRole_Forbidden(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	// 创建一个 visibility_type=group 的角色
	model.DB(context.Background()).Create(&model.OpenClawRole{
		ID: 1, Name: "分组角色", VisibilityType: "group",
	})

	form := url.Values{}
	form.Set("name", "test-inst")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("role_id", "1") // 分组角色
	// 不传 group_id → groupID=0
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("未分组用户选择分组角色应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestCreateInstance_UngroupedUser_AllRole_Allowed 未分组用户选择全局角色时不被拒绝
func TestCreateInstance_UngroupedUser_AllRole_Allowed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	// 创建一个 visibility_type=all 的角色
	model.DB(context.Background()).Create(&model.OpenClawRole{
		ID: 2, Name: "全局角色", VisibilityType: "all",
	})

	// 创建启用的镜像
	model.DB(context.Background()).Create(&model.AIImage{
		AgentType: model.AgentTypeOpenClaw,
		Enabled:   true,
		ImageId:   "img-001",
	})

	form := url.Values{}
	form.Set("name", "test-inst-ok")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("role_id", "2") // 全局角色
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	// 不应返回 400（角色校验通过，可能后续步骤失败，但不是角色问题）
	if rr.Code == http.StatusBadRequest {
		body := rr.Body.String()
		if strings.Contains(body, "角色") {
			t.Errorf("全局角色不应被拒绝，实际=%d body=%s", rr.Code, body)
		}
	}
}

// TestCreateInstance_UngroupedUser_RestrictedImageType_Forbidden 未分组用户创建受限镜像类型
func TestCreateInstance_UngroupedUser_RestrictedImageType_Forbidden(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	// 在 group_config_bindings 中添加一个受限的 agent_type
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypeImageType,
		ConfigKey:  model.AgentTypeOpenClaw,
		GroupID:    1,
	})

	// 创建镜像
	model.DB(context.Background()).Create(&model.AIImage{
		AgentType: model.AgentTypeOpenClaw,
		Enabled:   true,
		ImageId:   "img-001",
	})

	form := url.Values{}
	form.Set("name", "test-inst-restricted")
	form.Set("agent_type", model.AgentTypeOpenClaw) // 受限类型
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("未分组用户创建受限镜像类型应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestCreateInstance_UngroupedUser_UnrestrictedImageType_Allowed 未分组用户创建未受限镜像类型
func TestCreateInstance_UngroupedUser_UnrestrictedImageType_Allowed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	// group_config_bindings 中限制 hermes，但 openclaw 未受限
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypeImageType,
		ConfigKey:  model.AgentTypeHermes,
		GroupID:    1,
	})

	// 创建启用的 openclaw 镜像
	model.DB(context.Background()).Create(&model.AIImage{
		AgentType: model.AgentTypeOpenClaw,
		Enabled:   true,
		ImageId:   "img-001",
	})

	form := url.Values{}
	form.Set("name", "test-inst-unrestricted")
	form.Set("agent_type", model.AgentTypeOpenClaw) // 未受限类型
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	// 不应返回 403（镜像类型校验通过）
	if rr.Code == http.StatusForbidden {
		t.Errorf("未受限镜像类型不应被拒绝，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleSetChannel_GroupVisibility_GroupedInstance_NotBound 分组实例但通道未绑定到该组
func TestHandleSetChannel_GroupVisibility_GroupedInstance_NotBound(t *testing.T) {
	cleanup := initChannelTestDB(t)
	defer cleanup()

	// 创建分组 + closure
	model.DB(context.Background()).Create(&model.UserGroup{ID: 5, Name: "DevGroup", Source: "manual", FullPath: "DevGroup"})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 5, DescendantID: 5, Depth: 0})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-ch-grp5",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		GroupID: 5,
	}
	model.DB(context.Background()).Create(inst)

	// 创建 visibility_type=group 的通道，但绑定到分组 99（不是 5）
	enabled := true
	ch := &model.AIChannel{
		ChannelID: "feishu", Name: "飞书", Enabled: &enabled,
		VisibilityType: "group",
	}
	model.DB(context.Background()).Create(ch)
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypeChannel,
		ConfigKey:  fmt.Sprintf("%d", ch.ID),
		GroupID:    99, // 绑定到不同的分组
	})

	form := url.Values{}
	form.Set("channel", "feishu")
	form.Add("key", "appid")
	form.Add("value", "12345")
	req := channelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/channel?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("通道未绑定到实例分组应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

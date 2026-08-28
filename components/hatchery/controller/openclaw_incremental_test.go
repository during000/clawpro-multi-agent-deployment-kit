package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"
)

// ==================== HandleInstanceList agent_type 兼容测试 ====================

func TestHandleInstanceList_EmptyAgentTypeDefaultsToOpenclaw(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建一个 agent_type 为空的实例（模拟存量数据）
	instance := model.Instance{
		UserID:    user.ID,
		Name:      "Legacy Instance",
		AgentType: "", // 存量数据
	}
	model.DB(context.Background()).Create(&instance)

	req := userOpenclawReqWithSession(t, http.MethodGet, "/openclaw/instances", "testuser")
	rr := httptest.NewRecorder()

	HandleInstanceList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	instances, ok := resp["instances"].([]interface{})
	if !ok {
		t.Fatalf("expected instances array, got %T", resp["instances"])
	}

	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}

	inst := instances[0].(map[string]interface{})
	agentType, _ := inst["agent_type"].(string)
	if agentType != "openclaw" {
		t.Errorf("expected agent_type to be 'openclaw' (default), got %q", agentType)
	}
}

// ==================== HandleCurrentImage agent_type 路径测试 ====================

func TestHandleCurrentImage_FallbackToLegacyImage(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 只创建一个 legacy 镜像（agent_type 为空）
	legacyImg := model.AIImage{
		ImageId:   "img-legacy-001",
		ImageName: "Legacy Image",
		AgentType: "", // 存量空类型
		Enabled:   true,
	}
	model.DB(context.Background()).Create(&legacyImg)

	// 不传 agent_type，走旧的 GetEnabledImage 路径
	req := userOpenclawReqWithSession(t, http.MethodGet, "/openclaw/current-image", "testuser")
	rr := httptest.NewRecorder()

	HandleCurrentImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["image"] == nil {
		t.Error("expected legacy image to be returned via fallback path")
	}
}

func TestHandleCurrentImage_WithAgentType_PrivateImage(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建 PRIVATE_IMAGE 类型的镜像
	img := model.AIImage{
		ImageId:      "img-priv-001",
		ImageName:    "Private Image",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(&img)

	req := userOpenclawReqWithSession(t, http.MethodGet, "/openclaw/current-image?agent_type=openclaw", "testuser")
	rr := httptest.NewRecorder()

	HandleCurrentImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	imgData := resp["image"].(map[string]interface{})

	// PRIVATE_IMAGE 应该 public=false
	if imgData["public"] != false {
		t.Errorf("expected public=false for PRIVATE_IMAGE, got %v", imgData["public"])
	}
}

func TestHandleCurrentImage_AgentTypeWhitespace(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	img := model.AIImage{
		ImageId:      "img-001",
		ImageName:    "OpenClaw Image",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(&img)

	// agent_type 带空格，应该被 TrimSpace 处理
	req := userOpenclawReqWithSession(t, http.MethodGet, "/openclaw/current-image?agent_type=%20openclaw%20", "testuser")
	rr := httptest.NewRecorder()

	HandleCurrentImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["image"] == nil {
		t.Error("expected image to be found after trimming whitespace from agent_type")
	}
}

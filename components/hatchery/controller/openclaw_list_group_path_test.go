package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"
)

// TestHandleInstanceList_GroupFullPath 验证 /openclaw/list JSON 响应里：
//   - 每条实例都带 GroupID（Instance 嵌入自动序列化）
//   - 每条实例都带 GroupFullPath 字段，能回填到对应分组的 full_path
//   - 用户分组未指定（GroupID=0）时 GroupFullPath 为空串
func TestHandleInstanceList_GroupFullPath(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	// 需要 UserGroup 表（initFiveHandlersTestDB 默认没建，这里补）
	if err := model.DB(context.Background()).AutoMigrate(&model.UserGroup{}); err != nil {
		t.Fatalf("migrate UserGroup: %v", err)
	}

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	g := model.UserGroup{Name: "后端组", FullPath: "研发中心/后端组", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	model.DB(context.Background()).Create(&model.Instance{
		Name: "with-group", InstanceId: "", // 无 CVM id → 跳过 batchFetch
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw, GroupID: g.ID,
	})
	model.DB(context.Background()).Create(&model.Instance{
		Name: "no-group", InstanceId: "",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw, GroupID: 0,
	})

	req := jsonReqWithSession(t, http.MethodGet, "/openclaw/list", "u1", "")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Instances []map[string]interface{} `json:"instances"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v, body=%s", err, rr.Body.String())
	}
	if len(resp.Instances) != 2 {
		t.Fatalf("期望 2 条实例，实际 %d", len(resp.Instances))
	}

	byName := map[string]map[string]interface{}{}
	for _, it := range resp.Instances {
		name, _ := it["Name"].(string)
		byName[name] = it
	}
	hit := byName["with-group"]
	if hit == nil {
		t.Fatalf("缺少 with-group 实例: %+v", resp.Instances)
	}
	// GroupID 对应 JSON key 为 group_id（snake_case）
	if gid, _ := hit["group_id"].(float64); uint(gid) != g.ID {
		t.Errorf("group_id 不正确: %+v", hit)
	}
	if p, _ := hit["group_full_path"].(string); p != "研发中心/后端组" {
		t.Errorf("group_full_path 不正确: %q", p)
	}

	miss := byName["no-group"]
	if miss == nil {
		t.Fatalf("缺少 no-group 实例: %+v", resp.Instances)
	}
	if gid, _ := miss["group_id"].(float64); uint(gid) != 0 {
		t.Errorf("no-group 的 group_id 应为 0，实际 %v", miss["group_id"])
	}
	if p, _ := miss["group_full_path"].(string); p != "" {
		t.Errorf("no-group 的 group_full_path 应为空串，实际 %q", p)
	}
}

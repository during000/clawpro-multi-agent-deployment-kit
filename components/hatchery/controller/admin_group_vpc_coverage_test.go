package controller

import (
	"context"
	"encoding/json"
	"hatchery/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── 补充 VPC handler 覆盖率测试 ──────────────────────────────────────────────

func TestCoverageVpcList_MethodNotAllowed(t *testing.T) {
	setupVpcTestDB(t)

	w := httptest.NewRecorder()
	HandleListGroupVpcConfigs(w, adminReqVpc(http.MethodPost, "/admin/group-vpc-configs", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

func TestCoverageVpcCreate_Success_AllVisibility(t *testing.T) {
	setupVpcTestDB(t)

	// 创建一个分组用于绑定（同事改动要求必须至少选择一个分组）
	g := model.UserGroup{Name: "vpc-test-group", Source: model.GroupSourceManual}
	model.DB(context.Background()).Create(&g)

	reqBody := map[string]interface{}{
		"vpc_id":        "vpc-coverage-test",
		"subnet_ids":    `{"ap-guangzhou-3": ["subnet-cov1", "subnet-cov2"]}`,
		"strategy_name": "覆盖率测试",
		"group_ids":     []uint{g.ID},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleCreateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/create", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["id"] == nil {
		t.Error("期望返回 id")
	}
}

func TestCoverageVpcCreate_InvalidJSON(t *testing.T) {
	setupVpcTestDB(t)

	w := httptest.NewRecorder()
	req := adminReqVpc(http.MethodPost, "/admin/group-vpc-configs/create", []byte("not json"))
	HandleCreateGroupVpcConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageVpcUpdate_Success(t *testing.T) {
	setupVpcTestDB(t)

	// 创建分组用于绑定
	g := model.UserGroup{Name: "vpc-upd-group", Source: model.GroupSourceManual}
	model.DB(context.Background()).Create(&g)

	vpc := model.VpcConfig{
		VpcId:          "vpc-upd-test",
		SubnetIds:      `{"ap-guangzhou-3": ["subnet-upd"]}`,
		StrategyName:   "旧",
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&vpc)

	reqBody := map[string]interface{}{
		"id":            vpc.ID,
		"vpc_id":        "vpc-upd-new",
		"subnet_ids":    `{"ap-guangzhou-4": ["subnet-upd-new"]}`,
		"strategy_name": "新",
		"group_ids":     []uint{g.ID},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleUpdateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/update", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageVpcUpdate_InvalidJSON(t *testing.T) {
	setupVpcTestDB(t)

	w := httptest.NewRecorder()
	req := adminReqVpc(http.MethodPost, "/admin/group-vpc-configs/update", []byte("bad"))
	HandleUpdateGroupVpcConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageVpcDelete_InvalidJSON(t *testing.T) {
	setupVpcTestDB(t)

	w := httptest.NewRecorder()
	req := adminReqVpc(http.MethodPost, "/admin/group-vpc-configs/delete", []byte("bad"))
	HandleDeleteGroupVpcConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageVpcList_MultipleConfigs(t *testing.T) {
	setupVpcTestDB(t)

	model.DB(context.Background()).Create(&model.VpcConfig{
		VpcId: "vpc-1", SubnetIds: `{"z1": ["s1"]}`, VisibilityType: "all",
	})
	model.DB(context.Background()).Create(&model.VpcConfig{
		VpcId: "vpc-2", SubnetIds: `{"z2": ["s2"]}`, VisibilityType: "all",
	})

	w := httptest.NewRecorder()
	HandleListGroupVpcConfigs(w, adminReqVpc(http.MethodGet, "/admin/group-vpc-configs", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("期望 2 条，实际=%d", len(data))
	}
}

func TestCoverageVpcCreate_SubnetEmpty(t *testing.T) {
	setupVpcTestDB(t)

	reqBody := map[string]interface{}{
		"vpc_id":     "vpc-test",
		"subnet_ids": "",
		"group_ids":  []uint{},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleCreateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/create", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// ── validateVpcConfigRequest 补充测试 ────────────────────────────────────────

func TestCoverageValidateVpcConfigRequest_ValidNoGroups(t *testing.T) {
	setupVpcTestDB(t)

	// 创建分组（新逻辑要求必须至少选择一个分组）
	g := model.UserGroup{Name: "vpc-validate-group", Source: model.GroupSourceManual}
	model.DB(context.Background()).Create(&g)

	err := validateVpcConfigRequest(context.Background(), "vpc-x", `{"z": ["s"]}`, "test", []uint{g.ID}, 0)
	if err != nil {
		t.Errorf("合法请求不应报错: %v", err)
	}
}

func TestCoverageValidateVpcConfigRequest_EmptyVpcId(t *testing.T) {
	setupVpcTestDB(t)

	err := validateVpcConfigRequest(context.Background(), "", `{"z": ["s"]}`, "test", nil, 0)
	if err == nil {
		t.Error("空 vpc_id 应报错")
	}
}

func TestCoverageValidateVpcConfigRequest_EmptySubnetIds(t *testing.T) {
	setupVpcTestDB(t)

	err := validateVpcConfigRequest(context.Background(), "vpc-x", "", "test", nil, 0)
	if err == nil {
		t.Error("空 subnet_ids 应报错")
	}
}

func TestCoverageValidateVpcConfigRequest_LongStrategyName(t *testing.T) {
	setupVpcTestDB(t)

	err := validateVpcConfigRequest(context.Background(), "vpc-x", `{"z": ["s"]}`, "这是一个超过二十个字的名称啦吧", nil, 0)
	if err == nil {
		t.Error("超长 strategy_name 应报错")
	}
}

func TestCoverageValidateVpcConfigRequest_InvalidSubnetJSON(t *testing.T) {
	setupVpcTestDB(t)

	err := validateVpcConfigRequest(context.Background(), "vpc-x", "not-json", "test", nil, 0)
	if err == nil {
		t.Error("非法 JSON 应报错")
	}
}

func TestCoverageValidateVpcConfigRequest_InvalidGroupIDs(t *testing.T) {
	setupVpcTestDB(t)

	err := validateVpcConfigRequest(context.Background(), "vpc-x", `{"z": ["s"]}`, "test", []uint{99999}, 0)
	if err == nil {
		t.Error("不存在的分组应报错")
	}
}

// ── validateGroupsNotBoundToVpc 补充测试 ─────────────────────────────────────

func TestCoverageValidateGroupsNotBoundToVpc_Empty(t *testing.T) {
	setupVpcTestDB(t)

	err := validateGroupsNotBoundToVpc(context.Background(), nil, 0)
	if err != nil {
		t.Errorf("空列表不应报错: %v", err)
	}
}

func TestCoverageValidateGroupsNotBoundToVpc_SameExcludeID(t *testing.T) {
	setupVpcTestDB(t)

	group := model.UserGroup{Name: "VPC组"}
	model.DB(context.Background()).Create(&group)

	vpc := model.VpcConfig{VpcId: "vpc-exc", SubnetIds: `{"z": ["s"]}`}
	model.DB(context.Background()).Create(&vpc)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  itoa(vpc.ID),
		GroupID:    group.ID,
	})

	// 排除自身 ID 时不应报冲突
	err := validateGroupsNotBoundToVpc(context.Background(), []uint{group.ID}, vpc.ID)
	if err != nil {
		t.Errorf("排除自身时不应报错: %v", err)
	}
}

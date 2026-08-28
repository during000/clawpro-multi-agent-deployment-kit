package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hatchery/controller/usergroup"
	"hatchery/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ── 测试辅助 ─────────────────────────────────────────────────────────────────

// setupVpcTestDB 初始化内存 SQLite，迁移 VPC 相关表。
func setupVpcTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserGroup{},
		&model.VpcConfig{},
		&model.GroupConfigBinding{},
		&model.GroupClosure{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	unlock := model.UseDBForTest(db)
	// 初始化默认配置（resolveVpcConfig 兜底会读 site_config）
	if err := db.Create(&model.SiteConfig{
		VpcId:     "vpc-default",
		SubnetIds: `{"ap-guangzhou-1": ["subnet-default-1"]}`,
	}).Error; err != nil {
		t.Fatalf("初始化 SiteConfig 失败: %v", err)
	}
	AdminToken = "test-admin-token"
	// Mock 云 API 校验函数
	checkVpcExists = func(ctx context.Context, vpcId string) error { return nil }
	checkSubnetsInVpc = func(ctx context.Context, vpcId string, subnetMap map[string][]string) error { return nil }

	t.Cleanup(func() { unlock() })
}

// adminReqVpc 构造携带管理员 Token 的 HTTP 请求
func adminReqVpc(method, path string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// ── resolveVpcConfig 函数测试 ─────────────────────────────────────────────────

// TestVpcResolve_当前组有VPC绑定
func TestVpcResolve_当前组有VPC绑定(t *testing.T) {
	setupVpcTestDB(t)

	// 创建用户组
	group := model.UserGroup{Name: "研发组"}
	if err := model.DB(context.Background()).Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}

	// 创建 VPC 配置
	vpc := model.VpcConfig{
		VpcId:     "vpc-123",
		SubnetIds: `{"ap-guangzhou-1": ["subnet-a", "subnet-b"]}`,
	}
	if err := model.DB(context.Background()).Create(&vpc).Error; err != nil {
		t.Fatalf("创建 VPC 配置失败: %v", err)
	}

	// 绑定到用户组
	binding := model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  fmt.Sprintf("%d", vpc.ID),
		GroupID:    group.ID,
	}
	if err := model.DB(context.Background()).Create(&binding).Error; err != nil {
		t.Fatalf("创建绑定失败: %v", err)
	}

	// 查询配置
	var config model.SiteConfig
	model.DB(context.Background()).First(&config)

	vpcID, subnetMap := usergroup.ResolveVpcConfig(context.Background(), group.ID, config.VpcId, config.GetSubnetMap())

	if vpcID != "vpc-123" {
		t.Errorf("期望 vpc_id=vpc-123，实际=%s", vpcID)
	}
	if len(subnetMap["ap-guangzhou-1"]) != 2 {
		t.Errorf("期望 2 个子网，实际=%d", len(subnetMap["ap-guangzhou-1"]))
	}
}

// TestVpcResolve_祖先组继承
func TestVpcResolve_祖先组继承(t *testing.T) {
	setupVpcTestDB(t)

	// 创建父组和子组
	parent := model.UserGroup{Name: "父组"}
	model.DB(context.Background()).Create(&parent)
	child := model.UserGroup{Name: "子组"}
	model.DB(context.Background()).Create(&child)

	// 创建 closure（模拟继承关系）
	closures := []model.GroupClosure{
		{AncestorID: parent.ID, DescendantID: parent.ID, Depth: 0},
		{AncestorID: child.ID, DescendantID: child.ID, Depth: 0},
		{AncestorID: parent.ID, DescendantID: child.ID, Depth: 1},
	}
	for _, c := range closures {
		model.DB(context.Background()).Create(&c)
	}

	// 父组绑定 VPC
	vpc := model.VpcConfig{
		VpcId:     "vpc-parent",
		SubnetIds: `{"ap-guangzhou-1": ["subnet-parent"]}`,
	}
	model.DB(context.Background()).Create(&vpc)
	binding := model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  fmt.Sprintf("%d", vpc.ID),
		GroupID:    parent.ID,
	}
	model.DB(context.Background()).Create(&binding)

	var config model.SiteConfig
	model.DB(context.Background()).First(&config)

	vpcID, subnetMap := usergroup.ResolveVpcConfig(context.Background(), child.ID, config.VpcId, config.GetSubnetMap())

	if vpcID != "vpc-parent" {
		t.Errorf("期望继承父组的 vpc-parent，实际=%s", vpcID)
	}
	if len(subnetMap["ap-guangzhou-1"]) != 1 {
		t.Errorf("期望 1 个子网，实际=%d", len(subnetMap["ap-guangzhou-1"]))
	}
}

// TestVpcResolve_回退预设策略
func TestVpcResolve_回退预设策略(t *testing.T) {
	setupVpcTestDB(t)

	group := model.UserGroup{Name: "研发组"}
	model.DB(context.Background()).Create(&group)

	var config model.SiteConfig
	model.DB(context.Background()).First(&config)

	vpcID, subnetMap := usergroup.ResolveVpcConfig(context.Background(), group.ID, config.VpcId, config.GetSubnetMap())

	// 应返回 SiteConfig 中的默认值
	if vpcID != "vpc-default" {
		t.Errorf("期望回退到 vpc-default，实际=%s", vpcID)
	}
	if len(subnetMap["ap-guangzhou-1"]) != 1 {
		t.Errorf("期望默认子网，实际=%v", subnetMap)
	}
}

// TestVpcResolve_groupID为零使用预设
func TestVpcResolve_groupID为零使用预设(t *testing.T) {
	setupVpcTestDB(t)

	var config model.SiteConfig
	model.DB(context.Background()).First(&config)

	vpcID, subnetMap := usergroup.ResolveVpcConfig(context.Background(), 0, config.VpcId, config.GetSubnetMap())

	if vpcID != "vpc-default" {
		t.Errorf("groupID=0 时应直接使用预设策略，实际=%s", vpcID)
	}
	if len(subnetMap["ap-guangzhou-1"]) != 1 {
		t.Errorf("期望默认子网，实际=%v", subnetMap)
	}
}

// ── HandleListGroupVpcConfigs 测试 ───────────────────────────────────────────

// TestVpcList_含可见分组
func TestVpcList_含可见分组(t *testing.T) {
	setupVpcTestDB(t)

	// 创建用户组
	group := model.UserGroup{Name: "研发组", FullPath: "研发/研发组"}
	model.DB(context.Background()).Create(&group)

	// 创建 VPC 配置
	vpc := model.VpcConfig{
		VpcId:          "vpc-123",
		SubnetIds:      `{"ap-guangzhou-1": ["subnet-a"]}`,
		StrategyName:   "测试策略",
		VisibilityType: "group",
	}
	model.DB(context.Background()).Create(&vpc)

	// 绑定到用户组
	binding := model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  fmt.Sprintf("%d", vpc.ID),
		GroupID:    group.ID,
	}
	model.DB(context.Background()).Create(&binding)

	w := httptest.NewRecorder()
	HandleListGroupVpcConfigs(w, adminReqVpc(http.MethodGet, "/admin/group-vpc-configs", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("期望 1 条记录，实际=%d", len(data))
	}

	item := data[0].(map[string]interface{})
	visGroups := item["visibility_groups"].([]interface{})
	if len(visGroups) != 1 {
		t.Fatalf("期望 1 个可见分组，实际=%d", len(visGroups))
	}

	visGroup := visGroups[0].(map[string]interface{})
	if visGroup["group_name"].(string) != "研发组" {
		t.Errorf("期望 group_name=研发组，实际=%s", visGroup["group_name"])
	}
}

// TestVpcList_空列表
func TestVpcList_空列表(t *testing.T) {
	setupVpcTestDB(t)

	w := httptest.NewRecorder()
	HandleListGroupVpcConfigs(w, adminReqVpc(http.MethodGet, "/admin/group-vpc-configs", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]interface{})
	if len(data) != 0 {
		t.Errorf("期望空列表，实际=%d 条", len(data))
	}
}

// ── HandleCreateGroupVpcConfig 测试 ──────────────────────────────────────────

// TestVpcCreate_strategyName超长
func TestVpcCreate_strategyName超长(t *testing.T) {
	setupVpcTestDB(t)

	reqBody := map[string]interface{}{
		"vpc_id":        "vpc-test",
		"subnet_ids":    `{"ap-guangzhou-1": ["subnet-a"]}`,
		"strategy_name": "这是一个超过二十个字符的策略名称啦",
		"group_ids":     []uint{},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleCreateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/create", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Error("期望包含 error 字段")
	}
}

// TestVpcCreate_分组已绑定
func TestVpcCreate_分组已绑定(t *testing.T) {
	setupVpcTestDB(t)

	group := model.UserGroup{Name: "研发组"}
	model.DB(context.Background()).Create(&group)

	// 先创建一个 VPC 配置并绑定
	existingVpc := model.VpcConfig{
		VpcId:     "vpc-existing",
		SubnetIds: `{"ap-guangzhou-1": ["subnet-x"]}`,
	}
	model.DB(context.Background()).Create(&existingVpc)

	binding := model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  fmt.Sprintf("%d", existingVpc.ID),
		GroupID:    group.ID,
	}
	model.DB(context.Background()).Create(&binding)

	// 尝试创建新的 VPC 配置并绑定到同一分组
	reqBody := map[string]interface{}{
		"vpc_id":        "vpc-new",
		"subnet_ids":    `{"ap-guangzhou-1": ["subnet-y"]}`,
		"strategy_name": "新策略",
		"group_ids":     []uint{group.ID},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleCreateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/create", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400（分组冲突），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// TestVpcCreate_subnetIds格式错误
func TestVpcCreate_subnetIds格式错误(t *testing.T) {
	setupVpcTestDB(t)

	reqBody := map[string]interface{}{
		"vpc_id":        "vpc-test",
		"subnet_ids":    "not-a-json",
		"strategy_name": "测试",
		"group_ids":     []uint{},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleCreateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/create", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// ── HandleUpdateGroupVpcConfig 测试 ──────────────────────────────────────────

// TestVpcUpdate_分组冲突检测
func TestVpcUpdate_分组冲突检测(t *testing.T) {
	setupVpcTestDB(t)

	groupA := model.UserGroup{Name: "研发组A"}
	model.DB(context.Background()).Create(&groupA)
	groupB := model.UserGroup{Name: "研发组B"}
	model.DB(context.Background()).Create(&groupB)

	// 创建两个 VPC 配置
	vpcA := model.VpcConfig{
		VpcId:     "vpc-a",
		SubnetIds: `{"ap-guangzhou-1": ["subnet-a"]}`,
	}
	model.DB(context.Background()).Create(&vpcA)
	vpcB := model.VpcConfig{
		VpcId:     "vpc-b",
		SubnetIds: `{"ap-guangzhou-1": ["subnet-b"]}`,
	}
	model.DB(context.Background()).Create(&vpcB)

	// vpcA 绑定到 groupA
	bindingA := model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  fmt.Sprintf("%d", vpcA.ID),
		GroupID:    groupA.ID,
	}
	model.DB(context.Background()).Create(&bindingA)

	// vpcB 绑定到 groupB
	bindingB := model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  fmt.Sprintf("%d", vpcB.ID),
		GroupID:    groupB.ID,
	}
	model.DB(context.Background()).Create(&bindingB)

	// 尝试将 vpcA 更新为绑定到 groupB（冲突）
	reqBody := map[string]interface{}{
		"id":            vpcA.ID,
		"vpc_id":        "vpc-a",
		"subnet_ids":    `{"ap-guangzhou-1": ["subnet-a"]}`,
		"strategy_name": "测试",
		"group_ids":     []uint{groupB.ID},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleUpdateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/update", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400（分组冲突），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// TestVpcUpdate_配置不存在
func TestVpcUpdate_配置不存在(t *testing.T) {
	setupVpcTestDB(t)

	reqBody := map[string]interface{}{
		"id":            99999,
		"vpc_id":        "vpc-test",
		"subnet_ids":    `{"ap-guangzhou-1": ["subnet-a"]}`,
		"strategy_name": "测试",
		"group_ids":     []uint{},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleUpdateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/update", body))

	if w.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d", w.Code)
	}
}

// ── HandleDeleteGroupVpcConfig 测试 ──────────────────────────────────────────

// TestVpcDelete_正常删除清理绑定
func TestVpcDelete_正常删除清理绑定(t *testing.T) {
	setupVpcTestDB(t)

	group := model.UserGroup{Name: "研发组"}
	model.DB(context.Background()).Create(&group)

	vpc := model.VpcConfig{
		VpcId:     "vpc-test",
		SubnetIds: `{"ap-guangzhou-1": ["subnet-a"]}`,
	}
	model.DB(context.Background()).Create(&vpc)

	binding := model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  fmt.Sprintf("%d", vpc.ID),
		GroupID:    group.ID,
	}
	model.DB(context.Background()).Create(&binding)

	reqBody := map[string]interface{}{"id": vpc.ID}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleDeleteGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/delete", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证 vpc_configs 已删除
	var count int64
	model.DB(context.Background()).Model(&model.VpcConfig{}).Where("id = ?", vpc.ID).Count(&count)
	if count != 0 {
		t.Errorf("期望 vpc_configs 已删除，实际 count=%d", count)
	}

	// 验证 group_config_bindings 已清理
	var bindingCount int64
	model.DB(context.Background()).Model(&model.GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ?",
			model.ConfigTypeVPC, fmt.Sprintf("%d", vpc.ID)).
		Count(&bindingCount)
	if bindingCount != 0 {
		t.Errorf("期望 bindings 已清理，实际 count=%d", bindingCount)
	}
}

// TestVpcDelete_配置不存在
func TestVpcDelete_配置不存在(t *testing.T) {
	setupVpcTestDB(t)

	reqBody := map[string]interface{}{"id": 99999}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleDeleteGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/delete", body))

	if w.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d", w.Code)
	}
}

// TestVpcDelete_id为零
func TestVpcDelete_id为零(t *testing.T) {
	setupVpcTestDB(t)

	reqBody := map[string]interface{}{"id": 0}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleDeleteGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/delete", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// ── 补充：完整路径测试（mock 云 API 后） ───────────────────────────────────

// TestVpcCreate_分组为空报错
func TestVpcCreate_分组为空报错(t *testing.T) {
	setupVpcTestDB(t)

	reqBody := map[string]interface{}{
		"vpc_id":        "vpc-new",
		"subnet_ids":    `{"ap-guangzhou-6": ["subnet-aaa"]}`,
		"strategy_name": "测试策略",
		"group_ids":     []uint{},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleCreateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/create", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400（分组不能为空），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// TestVpcCreate_正常创建带分组
func TestVpcCreate_正常创建带分组(t *testing.T) {
	setupVpcTestDB(t)

	group := model.UserGroup{Name: "研发组"}
	model.DB(context.Background()).Create(&group)

	reqBody := map[string]interface{}{
		"vpc_id":        "vpc-new",
		"subnet_ids":    `{"ap-guangzhou-6": ["subnet-aaa"]}`,
		"strategy_name": "研发专用",
		"group_ids":     []uint{group.ID},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleCreateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/create", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var vpc model.VpcConfig
	model.DB(context.Background()).Where("vpc_id = ?", "vpc-new").First(&vpc)
	if vpc.VisibilityType != "group" {
		t.Errorf("期望 visibility_type=group，实际=%s", vpc.VisibilityType)
	}
	if vpc.StrategyName != "研发专用" {
		t.Errorf("期望 strategy_name=研发专用，实际=%s", vpc.StrategyName)
	}

	var bindCount int64
	model.DB(context.Background()).Model(&model.GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ?",
			model.ConfigTypeVPC, fmt.Sprintf("%d", vpc.ID)).
		Count(&bindCount)
	if bindCount != 1 {
		t.Errorf("期望 1 条绑定，实际=%d", bindCount)
	}
}

// TestVpcUpdate_正常更新全量替换绑定
func TestVpcUpdate_正常更新全量替换绑定(t *testing.T) {
	setupVpcTestDB(t)

	groupA := model.UserGroup{Name: "组A"}
	model.DB(context.Background()).Create(&groupA)
	groupB := model.UserGroup{Name: "组B"}
	model.DB(context.Background()).Create(&groupB)

	// 创建 VPC 配置并绑定 groupA
	vpc := model.VpcConfig{
		VpcId:          "vpc-test",
		SubnetIds:      `{"ap-guangzhou-6": ["subnet-a"]}`,
		VisibilityType: "group",
		StrategyName:   "旧名",
	}
	model.DB(context.Background()).Create(&vpc)
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  fmt.Sprintf("%d", vpc.ID),
		GroupID:    groupA.ID,
	})

	// 更新：改 VPC、换绑定到 groupB
	reqBody := map[string]interface{}{
		"id":            vpc.ID,
		"vpc_id":        "vpc-updated",
		"subnet_ids":    `{"ap-guangzhou-7": ["subnet-b"]}`,
		"strategy_name": "新名",
		"group_ids":     []uint{groupB.ID},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleUpdateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/update", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证 vpc_configs 已更新
	var updated model.VpcConfig
	model.DB(context.Background()).First(&updated, vpc.ID)
	if updated.VpcId != "vpc-updated" {
		t.Errorf("期望 vpc_id=vpc-updated，实际=%s", updated.VpcId)
	}
	if updated.StrategyName != "新名" {
		t.Errorf("期望 strategy_name=新名，实际=%s", updated.StrategyName)
	}

	// 验证 bindings 全量替换
	configKey := fmt.Sprintf("%d", vpc.ID)
	var bindA, bindB int64
	model.DB(context.Background()).Model(&model.GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ? AND group_id = ?",
			model.ConfigTypeVPC, configKey, groupA.ID).Count(&bindA)
	model.DB(context.Background()).Model(&model.GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ? AND group_id = ?",
			model.ConfigTypeVPC, configKey, groupB.ID).Count(&bindB)
	if bindA != 0 {
		t.Errorf("期望 groupA 绑定已删除，实际 count=%d", bindA)
	}
	if bindB != 1 {
		t.Errorf("期望 groupB 绑定已创建，实际 count=%d", bindB)
	}
}

// TestVpcUpdate_方法不允许
func TestVpcUpdate_方法不允许(t *testing.T) {
	setupVpcTestDB(t)

	w := httptest.NewRecorder()
	HandleUpdateGroupVpcConfig(w, adminReqVpc(http.MethodGet,
		"/admin/group-vpc-configs/update", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// TestVpcUpdate_id为零
func TestVpcUpdate_id为零(t *testing.T) {
	setupVpcTestDB(t)

	reqBody := map[string]interface{}{
		"id":         0,
		"vpc_id":     "vpc-test",
		"subnet_ids": `{"ap-guangzhou-6": ["subnet-a"]}`,
		"group_ids":  []uint{},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleUpdateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/update", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// TestVpcCreate_方法不允许
func TestVpcCreate_方法不允许(t *testing.T) {
	setupVpcTestDB(t)

	w := httptest.NewRecorder()
	HandleCreateGroupVpcConfig(w, adminReqVpc(http.MethodGet,
		"/admin/group-vpc-configs/create", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// TestVpcDelete_方法不允许
func TestVpcDelete_方法不允许(t *testing.T) {
	setupVpcTestDB(t)

	w := httptest.NewRecorder()
	HandleDeleteGroupVpcConfig(w, adminReqVpc(http.MethodGet,
		"/admin/group-vpc-configs/delete", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// TestVpcCreate_vpcId为空
func TestVpcCreate_vpcId为空(t *testing.T) {
	setupVpcTestDB(t)

	reqBody := map[string]interface{}{
		"vpc_id":     "",
		"subnet_ids": `{"ap-guangzhou-6": ["subnet-a"]}`,
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

// TestVpcUpdate_分组不存在
func TestVpcUpdate_分组不存在(t *testing.T) {
	setupVpcTestDB(t)

	vpc := model.VpcConfig{
		VpcId:     "vpc-test",
		SubnetIds: `{"ap-guangzhou-6": ["subnet-a"]}`,
	}
	model.DB(context.Background()).Create(&vpc)

	reqBody := map[string]interface{}{
		"id":         vpc.ID,
		"vpc_id":     "vpc-test",
		"subnet_ids": `{"ap-guangzhou-6": ["subnet-a"]}`,
		"group_ids":  []uint{9999},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	HandleUpdateGroupVpcConfig(w, adminReqVpc(http.MethodPost,
		"/admin/group-vpc-configs/update", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400（分组不存在），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// TestVpcList_subnetIds格式异常跳过
func TestVpcList_subnetIds格式异常跳过(t *testing.T) {
	setupVpcTestDB(t)

	// 手动插入一条 subnet_ids 格式异常的记录
	model.DB(context.Background()).Create(&model.VpcConfig{
		VpcId:          "vpc-bad",
		SubnetIds:      "not-valid-json",
		VisibilityType: "group",
	})
	// 正常记录
	model.DB(context.Background()).Create(&model.VpcConfig{
		VpcId:          "vpc-good",
		SubnetIds:      `{"ap-guangzhou-6": ["subnet-ok"]}`,
		VisibilityType: "group",
	})

	w := httptest.NewRecorder()
	HandleListGroupVpcConfigs(w, adminReqVpc(http.MethodGet, "/admin/group-vpc-configs", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]interface{})
	// 异常记录被跳过，只返回正常的
	if len(data) != 1 {
		t.Errorf("期望 1 条（跳过异常），实际=%d", len(data))
	}
}

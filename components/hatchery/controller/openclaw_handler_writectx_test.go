package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"

	"hatchery/model"

	tcCommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// ---------------------------------------------------------------------------
// HandleCreateInstance writeCtx 路径增量覆盖测试
//
// 覆盖 feature/fix_instance_creating_status 分支引入的 writeCtx 相关代码行：
//   1564-1568: writeCtx 创建
//   1571:      DB 更新使用 writeCtx
//   1586:      fallback DB 更新
//   1590-1596: fallback 失败处理
//   1601:      IncrementSGCVMCount
//   1608:      EnsureMemoryTDAIPluginRow
//   1616:      applyDefaultMemoryPlanForInstance
//   1621:      GetDefaultRuleSet
//   1638, 1642: 默认模型注入
//   1664, 1669, 1675, 1678, 1683, 1691: 异步 goroutine 使用 DetachContext(writeCtx)
//   1724:      inheritCLSScopeForNewInstance
//   1734:      approveDeviceAsync
// ---------------------------------------------------------------------------

// mockCVMRunInstancesServer 返回 RunInstances 成功响应的 httptest server。
func mockCVMRunInstancesServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.Header.Get("X-TC-Action") == "DescribeInstances" {
			fmt.Fprint(w, `{"Response":{"InstanceSet":[{"InstanceId":"ins-mock-writectx","InstanceState":"RUNNING"}],"TotalCount":1,"RequestId":"mock-describe-req-id"}}`)
			return
		}
		fmt.Fprint(w, `{"Response":{"InstanceIdSet":["ins-mock-writectx"],"RequestId":"mock-req-id"}}`)
	}))
}

// newMockCVMClientWithServer 创建一个指向本地 httptest server 的 CVM client。
func newMockCVMClientWithServer(t *testing.T, serverURL string) *cvm.Client {
	t.Helper()
	endpoint := strings.TrimPrefix(serverURL, "http://")
	credential := tcCommon.NewCredential("fake-id", "fake-key")
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = endpoint
	cpf.HttpProfile.Scheme = "HTTP"
	client, err := cvm.NewClient(credential, "ap-guangzhou", cpf)
	if err != nil {
		t.Fatalf("创建 mock CVM client 失败: %v", err)
	}
	return client
}

func TestHandleCreateInstance_RunInstancesSuccess_WriteCtxPath(t *testing.T) {
	// 使用文件数据库而非 :memory:，隔离异步 goroutine 的多连接访问
	tmpFile, err := os.CreateTemp("", "test-handle-create-*.db")
	if err != nil {
		t.Fatalf("创建临时 DB 文件失败: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.AIImage{}, &model.AIModel{},
		&model.SiteConfig{}, &model.AuditLog{}, &model.Notification{},
		&model.SkillInstallation{}, &model.SMHPersonalSpace{},
		&model.MemoryTDAIPlugin{}, &model.RuleSet{}, &model.Tag{},
		&model.GroupConfigBinding{}, &model.ResourcePolicy{}, &model.OpenClawRole{},
		&model.RoleVisibilityGroup{}, &model.UserGroup{}, &model.GroupClosure{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() {
		origDB()
		Store = origStore
	}()

	// 安装 SG 选取 hook：绕过 VPC 校验 + 设置 region 为 CVMClient 所需
	origValidate := validateGlobalVpcAndSubnetsFn
	origSelect := selectSGForNewInstanceFn
	origCVMRegion := CVMRegion
	validateGlobalVpcAndSubnetsFn = func(ctx context.Context, vpcId string, subnetMap map[string][]string) error { return nil }
	CVMRegion = "ap-guangzhou"
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "sg-mock-001", false, nil
	}
	defer func() {
		validateGlobalVpcAndSubnetsFn = origValidate
		selectSGForNewInstanceFn = origSelect
		CVMRegion = origCVMRegion
	}()

	// Skip cloud API pre-validation in this test.
	origValidateResourceConfig := validateCreateResourceConfigFn
	validateCreateResourceConfigFn = func(_ context.Context, _, _, _ string) error { return nil }
	defer func() { validateCreateResourceConfigFn = origValidateResourceConfig }()

	// 1. 启动 mock CVM API server，返回成功的 RunInstances 响应
	ts, capturedRunInstances := mockCVMRunInstancesCaptureServer(t)
	defer ts.Close()

	// 2. Mock NewCVMClient 指向 mock server
	origCVM := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return newMockCVMClientWithServer(t, ts.URL), nil
	}
	defer func() { NewCVMClient = origCVM }()

	// 3. 预置测试数据（image + site config + user）
	seedForSGSelectionTest(t, "u-writectx")
	seedDefaultRuleSet(t)
	if err := model.DB(context.Background()).Create(&model.Tag{
		TagKey:         "managed-by",
		TagValue:       "clawpro",
		VisibilityType: model.VisibilityAll,
	}).Error; err != nil {
		t.Fatalf("seed default tag: %v", err)
	}

	// 4. 发起创建请求
	form := url.Values{}
	form.Set("name", "inst-writectx")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("tags", `[{"key":"env","value":"prod"},{"key":"business","value":"one"}]`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-writectx", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	// 5. 验证响应：200 OK
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"ok":true`) {
		t.Errorf("响应应包含 ok:true，实际=%s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"redirect":"/openclaw"`) {
		t.Errorf("普通用户创建响应应保留 redirect:/openclaw，实际=%s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"preset"`) {
		t.Errorf("普通用户创建响应不应包含管理员预设字段，实际=%s", rr.Body.String())
	}
	assertCapturedInstanceTags(t, capturedRunInstances, []createInstanceTag{
		{Key: "env", Value: "prod"},
		{Key: "business", Value: "one"},
		{Key: "managed-by", Value: "clawpro"},
	})

	// 6. 验证 DB 中实例已正确创建
	var inst model.Instance
	if err := model.DB(context.Background()).Where("name = ?", "inst-writectx").First(&inst).Error; err != nil {
		t.Fatalf("实例应已创建并写入 DB: %v", err)
	}

	if inst.InstanceId != "ins-mock-writectx" {
		t.Errorf("instance_id 应为 ins-mock-writectx，实际=%s", inst.InstanceId)
	}
	if inst.SecurityGroupId != "sg-mock-001" {
		t.Errorf("security_group_id 应为 sg-mock-001，实际=%s", inst.SecurityGroupId)
	}
	if got := model.ParseTagItems(inst.CVMTagsJSON); fmt.Sprint(got) != fmt.Sprint([]model.TagItem{
		{Key: "env", Value: "prod"},
		{Key: "business", Value: "one"},
		{Key: "managed-by", Value: "clawpro"},
	}) {
		t.Errorf("cvm_tags_json 应缓存最终合并标签，实际=%+v", got)
	}
	if inst.CurrentOperation != model.OpCreate {
		t.Errorf("current_operation 应为 %s，实际=%s", model.OpCreate, inst.CurrentOperation)
	}
	if inst.LastCVMState != "PENDING" {
		t.Errorf("last_cvm_state 应为 PENDING，实际=%s", inst.LastCVMState)
	}
	if inst.ProxyToken == nil || *inst.ProxyToken == "" {
		t.Error("proxy_token 不应为空")
	}
}

// mockCVMRunInstancesErrorServer 返回 InvalidAccount.InsufficientBalance 错误的 httptest server。
func mockCVMRunInstancesErrorServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"Response":{"Error":{"Code":"InvalidAccount.InsufficientBalance","Message":"账户余额不足，请及时充值：https://console.cloud.tencent.com/expense/recharge"},"RequestId":"mock-req-id"}}`)
	}))
}

func TestHandleCreateInstance_RunInstancesBalanceInsufficient(t *testing.T) {
	// 使用文件数据库隔离异步 goroutine 的多连接访问
	tmpFile, err := os.CreateTemp("", "test-insufficient-balance-*.db")
	if err != nil {
		t.Fatalf("创建临时 DB 文件失败: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.AIImage{}, &model.AIModel{},
		&model.SiteConfig{}, &model.AuditLog{}, &model.Notification{},
		&model.SkillInstallation{}, &model.SMHPersonalSpace{},
		&model.MemoryTDAIPlugin{}, &model.RuleSet{}, &model.Tag{},
		&model.GroupConfigBinding{}, &model.ResourcePolicy{}, &model.OpenClawRole{},
		&model.RoleVisibilityGroup{}, &model.UserGroup{}, &model.GroupClosure{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() {
		origDB()
		Store = origStore
	}()

	// 安装 SG 选取 hook：绕过 VPC 校验 + 设置 region
	origValidate := validateGlobalVpcAndSubnetsFn
	origSelect := selectSGForNewInstanceFn
	origCVMRegion := CVMRegion
	validateGlobalVpcAndSubnetsFn = func(_ context.Context, _ string, _ map[string][]string) error { return nil }
	CVMRegion = "ap-guangzhou"
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "sg-mock-001", false, nil
	}
	defer func() {
		validateGlobalVpcAndSubnetsFn = origValidate
		selectSGForNewInstanceFn = origSelect
		CVMRegion = origCVMRegion
	}()

	// Skip cloud API pre-validation in this test.
	origValidateResourceConfig := validateCreateResourceConfigFn
	validateCreateResourceConfigFn = func(_ context.Context, _, _, _ string) error { return nil }
	defer func() { validateCreateResourceConfigFn = origValidateResourceConfig }()

	// 1. 启动 mock CVM API server：返回余额不足错误
	ts := mockCVMRunInstancesErrorServer(t)
	defer ts.Close()

	// 2. Mock NewCVMClient 指向 mock server
	origCVM := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return newMockCVMClientWithServer(t, ts.URL), nil
	}
	defer func() { NewCVMClient = origCVM }()

	// 3. 预置测试数据
	seedForSGSelectionTest(t, "u-insufficient")
	seedDefaultRuleSet(t)

	// 4. 发起创建请求
	form := url.Values{}
	form.Set("name", "inst-insufficient")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-insufficient", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	// 5. 验证：无 panic → 返回 500 + 可读错误信息
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "创建实例失败") {
		t.Errorf("body 应包含 '创建实例失败'，实际=%s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "账户余额不足") {
		t.Errorf("body detail 应包含 '账户余额不足'，实际=%s", rr.Body.String())
	}

	// 6. 验证占位记录被清理
	var count int64
	model.DB(context.Background()).Model(&model.Instance{}).Count(&count)
	if count != 0 {
		t.Errorf("占位应被清理，实际剩余=%d", count)
	}
}

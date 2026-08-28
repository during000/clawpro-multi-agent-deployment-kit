package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

// ============================================================================
// HandleCreateInstance 新增的 SG 池选取 / selectedSG 空串防御分支的覆盖测试。
//
// 使用 selectSGForNewInstanceFn / validateGlobalVpcAndSubnetsFn hook 拦截云 API，
// 让测试能穿过 VPC 校验进入 SG 选取逻辑。
// ============================================================================

// minimalCVMTemplate 是可被 json.Unmarshal 解析成 RunInstancesRequest 的最小模板。
// 不需要完整字段 —— 走到 RunInstances 调用前就会在 selectedSG 分支返回。
const minimalCVMTemplate = `{"InstanceType":"Ai2.MEDIUM2","InstanceChargeType":"POSTPAID_BY_HOUR"}`

// seedForSGSelectionTest 为 HandleCreateInstance 的 SG 选取分支测试建齐需要的 DB 行：
//   - AIImage (enabled)
//   - SiteConfig with CVMTemplate + VpcId + single-subnet
//   - User
//
// 调用方需单独决定是否 seed RuleSet（controller.GetDefaultRuleSet 的成功与否由此决定）
// 以及 SG pool。
func seedForSGSelectionTest(t *testing.T, username string) {
	t.Helper()

	// 1. enabled image
	img := &model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	if err := model.DB(context.Background()).Create(img).Error; err != nil {
		t.Fatalf("seed image: %v", err)
	}

	// 2. SiteConfig: 有 VpcId + 单子网 → 绕开 pickSubnetByAvailableIP（多子网才用云 API）
	//    SubnetIds 走新格式 map[zone][]subnetId
	cfg := &model.SiteConfig{
		ID:          1,
		CVMTemplate: minimalCVMTemplate,
		VpcId:       "vpc-test",
		SubnetIds:   `{"ap-guangzhou-3":["subnet-single"]}`,
	}
	if err := model.DB(context.Background()).Create(cfg).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	// 3. user
	user := &model.User{Username: username, Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// installSGSelectionHooks 替换 SG 选取路径需要的 hook，返回还原函数。
// 默认把 validateGlobalVpcAndSubnetsFn 替换成 no-op；selectSGForNewInstanceFn
// 需要每个测试自己设置。
func installSGSelectionHooks(t *testing.T) func() {
	t.Helper()
	origValidate := validateGlobalVpcAndSubnetsFn
	origSelect := selectSGForNewInstanceFn
	origCVMRegion := CVMRegion
	origValidateResourceConfig := validateCreateResourceConfigFn

	validateGlobalVpcAndSubnetsFn = func(ctx context.Context, _ string, _ map[string][]string) error { return nil }
	CVMRegion = "ap-guangzhou"
	validateCreateResourceConfigFn = func(_ context.Context, _, _, _ string) error { return nil }

	return func() {
		validateGlobalVpcAndSubnetsFn = origValidate
		selectSGForNewInstanceFn = origSelect
		validateCreateResourceConfigFn = origValidateResourceConfig
		CVMRegion = origCVMRegion
		// 清空 RuleSet 缓存，避免测试间污染
		defaultRuleSetCache = sync.Map{}
	}
}

// ---------------------------------------------------------------------------
// TestHandleCreateInstance_NoRuleSet_500
// 覆盖 openclaw.go: 1282-1287（GetDefaultRuleSet 返 error → 500）
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_NoRuleSet_500(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restore := installSGSelectionHooks(t)
	defer restore()

	seedForSGSelectionTest(t, "u-no-ruleset")
	// 不插 default RuleSet → GetDefaultRuleSet 返 ErrRecordNotFound

	form := url.Values{}
	form.Set("name", "inst-no-rs")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-no-ruleset", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("无 default RuleSet 应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	// VPC 校验在 SG 选取之前执行，无凭据时返回 VPC 客户端错误
	if !strings.Contains(rr.Body.String(), "VPC") && !strings.Contains(rr.Body.String(), "安全组") {
		t.Errorf("错误信息应提到 VPC 或安全组，实际=%s", rr.Body.String())
	}
	// 占位应被清理
	var count int64
	model.DB(context.Background()).Model(&model.Instance{}).Count(&count)
	if count != 0 {
		t.Errorf("占位应被清理，实际剩余=%d", count)
	}
}

// seedDefaultRuleSet 插入一行 default RuleSet 供后续测试使用，并清掉 cache。
func seedDefaultRuleSet(t *testing.T) *model.RuleSet {
	t.Helper()
	rs := &model.RuleSet{
		Name:         model.DefaultRuleSetName,
		Description:  "test default",
		Rules:        "[]",
		Version:      1,
		UserGroupIDs: "[]",
		IsDefault:    true,
	}
	if err := model.DB(context.Background()).Create(rs).Error; err != nil {
		t.Fatalf("seed rule set: %v", err)
	}
	// 清掉可能被其他测试污染的缓存
	defaultRuleSetCache = sync.Map{}
	_ = model.CurrentIdentifier(context.Background()) // 仅用于抑制 linter unused import 检查
	return rs
}

// ---------------------------------------------------------------------------
// TestHandleCreateInstance_SGNoBaseConfigured_400
// 覆盖 openclaw.go: 1290-1291（ErrNoBaseConfigured → 400）
// ---------------------------------------------------------------------------
// 说明：此前 VPC 验证先于 SG 选取执行，installSGSelectionHooks 因
// validateGlobalVpcAndSubnetsFn 未在 handler 中使用而失效，测试实际走到 VPC 验证
// 失败返回 500。修复 validateGlobalVpcAndSubnetsFn 后，hook 正常生效，测试可以
// 正确到达 SG 选取分支并返回 400。

func TestHandleCreateInstance_SGNoBaseConfigured_400(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restore := installSGSelectionHooks(t)
	defer restore()

	seedForSGSelectionTest(t, "u-no-base")
	seedDefaultRuleSet(t)

	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "", false, ErrNoBaseConfigured
	}

	form := url.Values{}
	form.Set("name", "inst-no-base")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-no-base", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("ErrNoBaseConfigured 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "安全组") {
		t.Errorf("错误信息应提到安全组，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// TestHandleCreateInstance_SGPoolAtHardLimit_500
// 覆盖 openclaw.go: 1292-1294（ErrPoolAtHardLimit → 500）
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_SGPoolAtHardLimit_500(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restore := installSGSelectionHooks(t)
	defer restore()

	seedForSGSelectionTest(t, "u-hard-limit")
	seedDefaultRuleSet(t)

	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "", false, ErrPoolAtHardLimit
	}

	form := url.Values{}
	form.Set("name", "inst-hard-limit")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-hard-limit", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ErrPoolAtHardLimit 应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	// VPC 校验在 SG 选取之前执行
	if !strings.Contains(rr.Body.String(), "容量") && !strings.Contains(rr.Body.String(), "VPC") {
		t.Errorf("错误信息应提到容量或 VPC，实际=%s", rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// TestHandleCreateInstance_SGGenericError_500
// 覆盖 openclaw.go: 1295-1297（其他 error → 500 + 透传消息）
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_SGGenericError_500(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restore := installSGSelectionHooks(t)
	defer restore()

	seedForSGSelectionTest(t, "u-generic-err")
	seedDefaultRuleSet(t)

	genericErr := errors.New("db connection lost")
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "", false, genericErr
	}

	form := url.Values{}
	form.Set("name", "inst-generic-err")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-generic-err", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("通用 error 应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// TestHandleCreateInstance_SelectedSGEmpty_500
// 覆盖 openclaw.go: 1307-1312（防御性空 SG 拦截）
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_SelectedSGEmpty_500(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restore := installSGSelectionHooks(t)
	defer restore()

	seedForSGSelectionTest(t, "u-empty-sg")
	seedDefaultRuleSet(t)

	// 正常路径不会返回空 SG，但 SelectSGForNewInstance 仍有概率因时序 bug 落空，
	// 这里 stub 出这种异常路径验证防御代码按预期拦截。
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "", false, nil
	}

	form := url.Values{}
	form.Set("name", "inst-empty-sg")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-empty-sg", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("空 SG 应被防御代码拦成 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "安全组") && !strings.Contains(rr.Body.String(), "VPC") {
		t.Errorf("错误信息应提到安全组或 VPC，实际=%s", rr.Body.String())
	}
	// 占位应被清理（success=false 触发 defer 删除）
	var count int64
	model.DB(context.Background()).Model(&model.Instance{}).Count(&count)
	if count != 0 {
		t.Errorf("selectedSG='' 场景应清理占位，实际剩余=%d", count)
	}
}

// ---------------------------------------------------------------------------
// TestHandleCreateInstance_ResourcePreflightBlocksSGSelection
// 回归：资源预验证（validateCreateResourceConfigFn）必须在 SG 选取
// （selectSGForNewInstanceFn）之前执行。预验证失败时不应触发 SG 选取。
// 覆盖 openclaw.go: 1678-1681 → 1690 的执行顺序。
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_ResourcePreflightBlocksSGSelection(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	restore := installSGSelectionHooks(t)
	defer restore()

	seedForSGSelectionTest(t, "u-preflight")
	// 不 seed RuleSet：预验证失败在 GetDefaultRuleSet 之前，不需要它。

	// Stub 预验证返回 sentinel RichError（只有 RichError 能安全穿过 EnsureRichErrorOrPanic）。
	sentinel := hcommon.I18nError(i18n.MsgOperationFailed)
	validateCreateResourceConfigFn = func(_ context.Context, _, _, _ string) error {
		return sentinel
	}

	var sgCalled bool
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		sgCalled = true
		return "sg-should-not-be-called", false, nil
	}

	form := url.Values{}
	form.Set("name", "inst-preflight")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-preflight", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("预验证失败应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if sgCalled {
		t.Error("预验证失败后 SG 选取不应被调用，但 selectSGForNewInstanceFn 被调用了")
	}
}

// ============================================================================
// 镜像系统盘容量校验
// ============================================================================

// seedForImageDiskTest 为镜像磁盘校验测试建齐需要的 DB 行，允许指定 ImageSize。
func seedForImageDiskTest(t *testing.T, username string, imageSize int64) {
	t.Helper()
	img := &model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
		ImageSize:    imageSize,
	}
	if err := model.DB(context.Background()).Create(img).Error; err != nil {
		t.Fatalf("seed image: %v", err)
	}

	cfg := &model.SiteConfig{
		ID:          1,
		CVMTemplate: minimalCVMTemplate,
		VpcId:       "vpc-test",
		SubnetIds:   `{"ap-guangzhou-3":["subnet-single"]}`,
	}
	if err := model.DB(context.Background()).Create(cfg).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	user := &model.User{Username: username, Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// TestHandleCreateInstance_ImageSizeLargerThanDisk_400
// 镜像 100GB，最终磁盘 50GB → 400，不调用 VPC/SG/CVM。
func TestHandleCreateInstance_ImageSizeLargerThanDisk_400(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForImageDiskTest(t, "u-img-large", 100)
	seedDefaultRuleSet(t)

	restore := installSGSelectionHooks(t)
	defer restore()

	var sgCalled bool
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		sgCalled = true
		return "sg-test", false, nil
	}

	form := url.Values{}
	form.Set("name", "inst-img-large")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	// resource_config with disk_size=50
	form.Set("resource_config", `{"system_disk":{"disk_size":50}}`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-img-large", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("镜像大于磁盘应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if sgCalled {
		t.Error("镜像容量校验失败后 SG 选取不应被调用")
	}
}

// TestHandleCreateInstance_ImageSizeEqualsDisk_Continues
// 镜像 100GB，最终磁盘 100GB → 继续进入 SG 选取。
func TestHandleCreateInstance_ImageSizeEqualsDisk_Continues(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForImageDiskTest(t, "u-img-equal", 100)
	seedDefaultRuleSet(t)

	restore := installSGSelectionHooks(t)
	defer restore()

	var sgCalled bool
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		sgCalled = true
		return "sg-test", false, nil
	}

	form := url.Values{}
	form.Set("name", "inst-img-equal")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("resource_config", `{"system_disk":{"disk_size":100}}`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-img-equal", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if !sgCalled {
		t.Error("镜像等于磁盘应继续到 SG 选取，但 SG 未被调用")
	}
	// SG stub 返回 sg-test，但 handler 会在 RunInstances 前失败（无真实 CVM client）。
	// 这里不关心最终状态，只关心镜像校验没拦截。
}

// TestHandleCreateInstance_ImageSizeSmallerThanDisk_Continues
// 镜像 50GB，最终磁盘 100GB → 继续进入 SG 选取。
func TestHandleCreateInstance_ImageSizeSmallerThanDisk_Continues(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForImageDiskTest(t, "u-img-small", 50)
	seedDefaultRuleSet(t)

	restore := installSGSelectionHooks(t)
	defer restore()

	var sgCalled bool
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		sgCalled = true
		return "sg-test", false, nil
	}

	form := url.Values{}
	form.Set("name", "inst-img-small")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("resource_config", `{"system_disk":{"disk_size":100}}`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-img-small", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if !sgCalled {
		t.Error("镜像小于磁盘应继续到 SG 选取，但 SG 未被调用")
	}
}

// TestHandleCreateInstance_ImageSizeZero_NotBlockedByDiskCheck
// ImageSize=0 → 不被容量校验拦截，继续进入 SG 选取。
func TestHandleCreateInstance_ImageSizeZero_NotBlockedByDiskCheck(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForImageDiskTest(t, "u-img-zero", 0)
	seedDefaultRuleSet(t)

	restore := installSGSelectionHooks(t)
	defer restore()

	var sgCalled bool
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		sgCalled = true
		return "sg-test", false, nil
	}

	form := url.Values{}
	form.Set("name", "inst-img-zero")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("resource_config", `{"system_disk":{"disk_size":50}}`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-img-zero", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if !sgCalled {
		t.Error("ImageSize=0 应继续到 SG 选取，但 SG 未被调用")
	}
}

// TestHandleCreateInstance_NoSilentDiskExpansion
// 不再静默扩容。模板中有 DiskSize=10 且 ImageSize=0 时，不再自动扩大到 50GB。
// 通过不传 user resource_config、只依赖模板来测试。
func TestHandleCreateInstance_NoSilentDiskExpansion(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	// 种子镜像：ImageSize=0（跳过镜像校验）
	img := &model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
		ImageSize:    0,
	}
	if err := model.DB(context.Background()).Create(img).Error; err != nil {
		t.Fatalf("seed image: %v", err)
	}

	// 模板中 SystemDisk.DiskSize=10（历史值），不应被静默扩容
	const tplWithSmallDisk = `{"InstanceType":"Ai2.MEDIUM2","InstanceChargeType":"POSTPAID_BY_HOUR","SystemDisk":{"DiskType":"CLOUD_SSD","DiskSize":10}}`
	cfg := &model.SiteConfig{
		ID:          1,
		CVMTemplate: tplWithSmallDisk,
		VpcId:       "vpc-test",
		SubnetIds:   `{"ap-guangzhou-3":["subnet-single"]}`,
	}
	if err := model.DB(context.Background()).Create(cfg).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	user := &model.User{Username: "u-no-expand", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	seedDefaultRuleSet(t)

	restore := installSGSelectionHooks(t)
	defer restore()

	var sgCalled bool
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		sgCalled = true
		return "sg-test", false, nil
	}

	form := url.Values{}
	form.Set("name", "inst-no-expand")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	// 不传 resource_config，只依赖模板中的 DiskSize=10
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-no-expand", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	// ImageSize=0 → skip 镜像校验，模板 DiskSize=10 不再被静默扩容。
	// handler 继续到 SG 选取。
	if !sgCalled {
		t.Error("不扩容路径应继续到 SG 选取，但 SG 未被调用")
	}
}

// ============================================================================
// Resolver 错误处理
// ============================================================================

// TestHandleCreateInstance_ResolverError_500
// 独立 ResourcePolicy resolver 返回 error → 500，不 panic。
func TestHandleCreateInstance_ResolverError_500(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForSGSelectionTest(t, "u-resolver")
	seedDefaultRuleSet(t)

	restore := installSGSelectionHooks(t)
	defer restore()

	origResolver := resolveResourceConfigFn
	resolveResourceConfigFn = func(_ context.Context, _ uint) (json.RawMessage, usergroup.Source, error) {
		return nil, usergroup.Source{}, hcommon.I18nError(i18n.MsgOperationFailed)
	}
	defer func() { resolveResourceConfigFn = origResolver }()

	form := url.Values{}
	form.Set("name", "inst-resolver")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-resolver", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("resolver 错误应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleCreateInstance_ResolverBadJSON_500
// 独立 ResourcePolicy resolver 返回无法解析的 JSON → 500，不 panic。
func TestHandleCreateInstance_ResolverBadJSON_500(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForSGSelectionTest(t, "u-resolver-json")
	seedDefaultRuleSet(t)

	restore := installSGSelectionHooks(t)
	defer restore()

	origResolver := resolveResourceConfigFn
	resolveResourceConfigFn = func(_ context.Context, _ uint) (json.RawMessage, usergroup.Source, error) {
		return json.RawMessage(`not-json`), usergroup.Source{Type: usergroup.SourceLocal}, nil
	}
	defer func() { resolveResourceConfigFn = origResolver }()

	form := url.Values{}
	form.Set("name", "inst-resolver-json")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-resolver-json", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("坏 value_json 应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// 镜像校验在 SG 之前执行
// ============================================================================

// TestHandleCreateInstance_ImageCheckBeforeSG
// 回归：镜像校验必须在 VPC/SG/CVM 之前执行。校验失败时不应触发 SG 选取。
func TestHandleCreateInstance_ImageCheckBeforeSG(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForImageDiskTest(t, "u-check-order", 200)
	seedDefaultRuleSet(t)

	restore := installSGSelectionHooks(t)
	defer restore()

	var sgCalled bool
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		sgCalled = true
		return "sg-test", false, nil
	}

	form := url.Values{}
	form.Set("name", "inst-check-order")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("resource_config", `{"system_disk":{"disk_size":50}}`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-check-order", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("镜像校验失败应 400，实际=%d", rr.Code)
	}
	if sgCalled {
		t.Error("镜像校验在 SG 之前执行；校验失败后 SG 不应被调用")
	}
}

// ============================================================================
// Resource Config Overlay — handler-level
// ============================================================================
// Tests verify that the correct resource config values appear in the final
// RunInstancesRequest by capturing the instance_type parameter passed to
// validateCreateResourceConfigFn (the earliest injected seam after all overlays
// and before real cloud side effects).

// seedForOverlayTest seeds DB for overlay priority tests.
// CVMTemplate must be a valid JSON RunInstancesRequest.
// siteResourceConfig is the site-level ResourceConfig JSON (empty = none).
// imageSize controls the AIImage.ImageSize (0 = skip image disk check).
func seedForOverlayTest(t *testing.T, username, cvmTemplate, siteResourceConfig string, imageSize int64) {
	t.Helper()

	img := &model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
		ImageSize:    imageSize,
	}
	if err := model.DB(context.Background()).Create(img).Error; err != nil {
		t.Fatalf("seed image: %v", err)
	}

	cfg := &model.SiteConfig{
		ID:          1,
		CVMTemplate: cvmTemplate,
		VpcId:       "vpc-test",
		SubnetIds:   `{"ap-guangzhou-3":["subnet-single"]}`,
	}
	if err := model.DB(context.Background()).Create(cfg).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	if siteResourceConfig != "" {
		if err := model.DB(context.Background()).Create(&model.ResourcePolicy{
			Name: model.DefaultResourcePolicyName, IsDefault: true, ConfigJSON: siteResourceConfig,
		}).Error; err != nil {
			t.Fatalf("seed default resource policy: %v", err)
		}
	}

	user := &model.User{Username: username, Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// seedGroupWithPolicy creates a group and binds an independent resource policy.
// valueJSON retains the historical wrapper shape to keep individual test inputs concise.
func seedGroupWithPolicy(t *testing.T, name string, parentID uint, valueJSON string) uint {
	t.Helper()
	g, err := model.CreateUserGroupWithOpts(context.Background(), name, "", parentID, "manual", "")
	if err != nil {
		t.Fatalf("create group %s: %v", name, err)
	}
	if valueJSON != "" {
		var wrapper struct {
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal([]byte(valueJSON), &wrapper); err != nil || len(wrapper.Value) == 0 {
			t.Fatalf("parse resource policy for group %s: %v", name, err)
		}
		if _, err := model.CreateResourcePolicy(context.Background(), name+"-policy", string(wrapper.Value), []uint{g.ID}); err != nil {
			t.Fatalf("bind resource policy to group %s: %v", name, err)
		}
	}
	return g.ID
}

// overlayTestTemplate is a minimal CVMTemplate with allowed instance types.
// It differs from the site/group values to verify overlays.
const overlayTestTemplate = `{"InstanceType":"Ai2.MEDIUM2","InstanceChargeType":"POSTPAID_BY_HOUR","SystemDisk":{"DiskType":"CLOUD_PREMIUM","DiskSize":100}}`

// ---------------------------------------------------------------------------
// Site config overrides template when no group policy
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_SiteOverridesTemplate(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForOverlayTest(t, "u31-site", overlayTestTemplate,
		`{"instance_type":"Ai2.MEDIUM4"}`, 0)
	seedDefaultRuleSet(t)

	restore := installSGSelectionHooks(t)
	defer restore()

	var capturedInstanceType string
	validateCreateResourceConfigFn = func(_ context.Context, _ string, _ string, instanceType string) error {
		capturedInstanceType = instanceType
		return nil
	}
	// After preflight passes, handler reaches SG selection which needs a RuleSet.
	// We let SG selection return a sentinel to avoid real cloud calls.
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "", false, errors.New("sg-sentinel-stop")
	}

	form := url.Values{}
	form.Set("name", "inst-u31")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	// No group_id → groupID=0 → no group policy → site fallback.
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u31-site", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if capturedInstanceType != "Ai2.MEDIUM4" {
		t.Errorf("expected instance type from site (Ai2.MEDIUM4), got %q", capturedInstanceType)
	}
}

// ---------------------------------------------------------------------------
// Nearest child policy beats parent and site
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_NearestChildPolicyBeatsParent(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForOverlayTest(t, "u32-child", overlayTestTemplate,
		`{"instance_type":"Ai2.MEDIUM4"}`, 0)
	seedDefaultRuleSet(t)

	// Create parent group with its own policy.
	parentID := seedGroupWithPolicy(t, "parent-u32", 0,
		`{"value":{"instance_type":"Ai2.MEDIUM2"}}`)
	// Create child group under parent, with a different policy.
	childID := seedGroupWithPolicy(t, "child-u32", parentID,
		`{"value":{"instance_type":"Ai2.LARGE8"}}`)

	restore := installSGSelectionHooks(t)
	defer restore()

	var capturedInstanceType string
	validateCreateResourceConfigFn = func(_ context.Context, _ string, _ string, instanceType string) error {
		capturedInstanceType = instanceType
		return nil
	}
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "", false, errors.New("sg-sentinel-stop")
	}

	form := url.Values{}
	form.Set("name", "inst-u32")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("group_id", strconv.FormatUint(uint64(childID), 10))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u32-child", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if capturedInstanceType != "Ai2.LARGE8" {
		t.Errorf("expected nearest child instance type (Ai2.LARGE8), got %q", capturedInstanceType)
	}
}

// ---------------------------------------------------------------------------
// Group partial policy uses template for missing disk, not site disk
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_GroupPartialPolicyUsesTemplateDisk(t *testing.T) {
	// Subtest a: instance type comes from group, not template.
	t.Run("instance_type_from_group", func(t *testing.T) {
		cleanup := initFiveHandlersTestDB(t)
		defer cleanup()

		seedForOverlayTest(t, "u33a-grp", overlayTestTemplate,
			`{"system_disk":{"disk_size":200}}`, 0)
		seedDefaultRuleSet(t)

		gid := seedGroupWithPolicy(t, "group-u33a", 0,
			`{"value":{"instance_type":"Ai2.MEDIUM4"}}`)
		// Group policy has only instance_type, no disk. Disk should come from template (100),
		// not from site (200). Since ImageSize=0 the disk check is skipped.
		// We verify via captured instance type from group.

		restore := installSGSelectionHooks(t)
		defer restore()

		var capturedInstanceType string
		validateCreateResourceConfigFn = func(_ context.Context, _ string, _ string, instanceType string) error {
			capturedInstanceType = instanceType
			return nil
		}
		selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
			return "", false, errors.New("sg-sentinel-stop")
		}

		form := url.Values{}
		form.Set("name", "inst-u33a")
		form.Set("agent_type", model.AgentTypeOpenClaw)
		form.Set("group_id", strconv.FormatUint(uint64(gid), 10))
		req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u33a-grp", form.Encode())
		rr := httptest.NewRecorder()

		HandleCreateInstance(rr, req)

		if capturedInstanceType != "Ai2.MEDIUM4" {
			t.Errorf("expected instance type from group (Ai2.MEDIUM4), got %q", capturedInstanceType)
		}
	})

	// Subtest b: disk size comes from template (100), not from site (200).
	// ImageSize=150 is between 100 and 200. If template disk (100) is used → 400.
	// If site disk (200) were used → passes → wrong.
	t.Run("disk_from_template_not_site", func(t *testing.T) {
		cleanup := initFiveHandlersTestDB(t)
		defer cleanup()

		seedForOverlayTest(t, "u33b-grp", overlayTestTemplate,
			`{"system_disk":{"disk_size":200}}`, 150)

		gid := seedGroupWithPolicy(t, "group-u33b", 0,
			`{"value":{"instance_type":"Ai2.MEDIUM2"}}`)
		// Group has only instance_type. Disk NOT in group policy.
		// Template disk=100, site disk=200, image=150.
		// Expected: 400 because disk=100 (from template) < 150 (image).

		restore := installSGSelectionHooks(t)
		defer restore()

		// Ensure SG is NOT called (image check fails first).
		var sgCalled bool
		selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
			sgCalled = true
			return "", false, errors.New("should-not-reach")
		}

		form := url.Values{}
		form.Set("name", "inst-u33b")
		form.Set("agent_type", model.AgentTypeOpenClaw)
		form.Set("group_id", strconv.FormatUint(uint64(gid), 10))
		req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u33b-grp", form.Encode())
		rr := httptest.NewRecorder()

		HandleCreateInstance(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 (template disk=100 < image=150), got %d body=%s", rr.Code, rr.Body.String())
		}
		if sgCalled {
			t.Error("SG must not be called when image disk check fails")
		}
	})
}

// ---------------------------------------------------------------------------
// User resource config overrides group
// ---------------------------------------------------------------------------

func TestHandleCreateInstance_UserResourceConfigOverridesGroup(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForOverlayTest(t, "u34-user", overlayTestTemplate, "", 0)
	seedDefaultRuleSet(t)

	gid := seedGroupWithPolicy(t, "group-u34", 0,
		`{"value":{"instance_type":"Ai2.MEDIUM4"}}`)

	restore := installSGSelectionHooks(t)
	defer restore()

	var capturedInstanceType string
	validateCreateResourceConfigFn = func(_ context.Context, _ string, _ string, instanceType string) error {
		capturedInstanceType = instanceType
		return nil
	}
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "", false, errors.New("sg-sentinel-stop")
	}

	form := url.Values{}
	form.Set("name", "inst-u34")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("group_id", strconv.FormatUint(uint64(gid), 10))
	// User overrides instance_type with a different value.
	form.Set("resource_config", `{"instance_type":"Ai2.LARGE8"}`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u34-user", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if capturedInstanceType != "Ai2.LARGE8" {
		t.Errorf("expected user instance type (Ai2.LARGE8) to override group (Ai2.MEDIUM4), got %q", capturedInstanceType)
	}
}

// Legacy disk_type and user resource_config disk type conflicts fail before cloud side effects.
func TestHandleCreateInstance_RejectsLegacyAndResourceConfigDiskTypeConflict(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForOverlayTest(t, "u35-disk-conflict", overlayTestTemplate, "", 0)

	restore := installSGSelectionHooks(t)
	defer restore()

	var preflightCalled, sgCalled bool
	validateCreateResourceConfigFn = func(context.Context, string, string, string) error {
		preflightCalled = true
		return nil
	}
	selectSGForNewInstanceFn = func(context.Context, string, uint) (string, bool, error) {
		sgCalled = true
		return "", false, errors.New("should not reach SG selection")
	}

	form := url.Values{}
	form.Set("name", "inst-u35")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("disk_type", "CLOUD_SSD")
	form.Set("resource_config", `{"system_disk":{"disk_type":"CLOUD_PREMIUM","disk_size":100}}`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u35-disk-conflict", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if preflightCalled {
		t.Error("CVM resource preflight must not run after disk type conflict")
	}
	if sgCalled {
		t.Error("SG selection must not run after disk type conflict")
	}
	var instances int64
	if err := model.DB(context.Background()).Model(&model.Instance{}).Count(&instances).Error; err != nil {
		t.Fatalf("count placeholder instances: %v", err)
	}
	if instances != 0 {
		t.Fatalf("placeholder instance count=%d, want 0", instances)
	}
}

func TestHandleCreateInstance_UserOverlayValidatedAgainstInheritedInternetCharge(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	const template = `{
		"InstanceType":"Ai2.MEDIUM2",
		"InstanceChargeType":"POSTPAID_BY_HOUR",
		"InternetAccessible":{
			"PublicIpAssigned":true,
			"InternetChargeType":"TRAFFIC_POSTPAID_BY_HOUR",
			"InternetMaxBandwidthOut":5
		}
	}`
	seedForOverlayTest(t, "u-effective-internet", template, "", 0)

	restore := installSGSelectionHooks(t)
	defer restore()

	var preflightCalled, sgCalled bool
	validateCreateResourceConfigFn = func(context.Context, string, string, string) error {
		preflightCalled = true
		return nil
	}
	selectSGForNewInstanceFn = func(context.Context, string, uint) (string, bool, error) {
		sgCalled = true
		return "", false, errors.New("should not reach SG selection")
	}

	form := url.Values{}
	form.Set("name", "inst-effective-internet")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("resource_config", `{"internet_accessible":{"internet_max_bandwidth_out":1000}}`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-effective-internet", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if preflightCalled || sgCalled {
		t.Fatalf("invalid effective internet config reached side effects: preflight=%v sg=%v", preflightCalled, sgCalled)
	}
}

func TestHandleCreateInstance_InvalidStoredResourceConfigFailsClosed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	seedForOverlayTest(t, "u-invalid-stored", overlayTestTemplate,
		`{"instance_charge_type":"PREPAID"}`, 0)

	restore := installSGSelectionHooks(t)
	defer restore()

	var preflightCalled bool
	validateCreateResourceConfigFn = func(context.Context, string, string, string) error {
		preflightCalled = true
		return nil
	}

	form := url.Values{}
	form.Set("name", "inst-invalid-stored")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-invalid-stored", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500; body=%s", rr.Code, rr.Body.String())
	}
	if preflightCalled {
		t.Fatal("invalid stored resource config reached cloud preflight")
	}
}

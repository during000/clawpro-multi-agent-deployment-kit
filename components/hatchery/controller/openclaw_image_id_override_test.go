// Tests for the hidden `image_id` override parameter introduced in
// feautre/add_image_id_params_for_create_ins_0616.
//
// 该参数为内部测试隐藏参数（不在外部接口文档暴露），仅对腾讯云账号 UIN
// 命中内部账号白名单（定义在 internal_account.go 的 internalAccountUins，
// 当前含 "3205597606"、"100049049642"）的部署生效，用于"先用低版本镜像创建
// 实例、再触发升级"等测试场景。本文件覆盖以下分支：
//
//  1. 非白名单部署：参数被静默忽略（行为等价于未传） → 不会因镜像不存在/不匹配返回 400
//  2. 白名单部署 + image_id 不存在 → 400 "未找到镜像: ..."
//  3. 白名单部署 + image_id 与 agent_type 不匹配 → 400 "指定镜像的 agent_type 与请求不匹配"
//  4. 白名单部署 + image_id 合法 → 顺利越过校验（不会因镜像分支返回 400）
//  5. 不传 image_id：行为完全不变（不命中任何新增分支）
package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/common"
	"hatchery/model"
)

const (
	// 与 controller/openclaw.go 中常量保持一致，硬编码避免 export 仅为测试
	whitelistTestUin = "3205597606"
)

// withTenantUin 为 req 注入指定 UIN 的 TenantSnapshot，模拟 IdentifierMiddleware 行为。
func withTenantUin(req *http.Request, uin string) *http.Request {
	ctx := common.InjectTenant(req.Context(), common.TenantSnapshot{Uin: uin})
	return req.WithContext(ctx)
}

// seedOpenClawEnabledImage 为 openclaw 类型创建一个启用镜像，使 HandleCreateInstance
// 越过 enabledImage 查询，得以进入新增的 image_id 校验分支。
func seedOpenClawEnabledImage(t *testing.T) {
	t.Helper()
	img := &model.AIImage{
		ImageId:      "img-default-openclaw",
		ImageName:    "default-openclaw",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "5.0.0",
		Enabled:      true,
	}
	if err := model.DB(context.Background()).Create(img).Error; err != nil {
		t.Fatalf("seed default openclaw image: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────
// 1. 非白名单部署：image_id 被静默忽略
// ──────────────────────────────────────────────────────────────────────────

// TestHandleCreateInstance_ImageIDOverride_IgnoredForNonWhitelistUin 验证：
// 当部署 UIN 不命中白名单时，即使传入一个根本不存在的 image_id，也不会因为
// "指定的镜像不存在" 返回 400。该参数应被静默忽略，行为等价于未传。
func TestHandleCreateInstance_ImageIDOverride_IgnoredForNonWhitelistUin(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	form := url.Values{}
	form.Set("name", "ignore-image-id")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("image_id", "img-nonexistent-xxx") // 故意不存在

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	// 注入一个**非白名单** UIN
	req = withTenantUin(req, "1111111111")
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	// 关键断言：响应体不能含有"指定的镜像 ... 不存在"这种新增分支独有的错误。
	// 真正的失败原因（CVM 模板缺失等）会在更后面，但绝不应是 image_id 校验。
	if strings.Contains(rr.Body.String(), "指定的镜像") {
		t.Fatalf("非白名单部署应静默忽略 image_id，不应触发镜像校验错误，body=%s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "agent_type 与请求不匹配") {
		t.Fatalf("非白名单部署应静默忽略 image_id，不应触发 agent_type 不匹配错误，body=%s", rr.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────
// 2. 白名单部署 + image_id 不存在 → 400
// ──────────────────────────────────────────────────────────────────────────

func TestHandleCreateInstance_ImageIDOverride_WhitelistImageNotFound(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	form := url.Values{}
	form.Set("name", "whitelist-no-such-image")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("image_id", "img-not-in-db")

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	req = withTenantUin(req, whitelistTestUin)
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("白名单部署+image_id 不存在 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "img-not-in-db") ||
		!strings.Contains(rr.Body.String(), "未找到镜像") {
		t.Fatalf("错误信息应包含 image_id 与 '未找到镜像'，实际=%s", rr.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────
// 3. 白名单部署 + image_id 与 agent_type 不匹配 → 400
// ──────────────────────────────────────────────────────────────────────────

func TestHandleCreateInstance_ImageIDOverride_WhitelistAgentTypeMismatch(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	// 额外种一张 hermes 类型镜像（无需 enabled，仅供 image_id 命中查询）
	hermesImg := &model.AIImage{
		ImageId:      "img-hermes-mismatch",
		ImageName:    "hermes-mismatch",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeHermes,
		AgentVersion: "1.0.0",
		Enabled:      false,
	}
	if err := model.DB(context.Background()).Create(hermesImg).Error; err != nil {
		t.Fatalf("seed hermes image: %v", err)
	}

	form := url.Values{}
	form.Set("name", "whitelist-mismatch")
	form.Set("agent_type", model.AgentTypeOpenClaw) // 请求 openclaw
	form.Set("image_id", "img-hermes-mismatch")     // 但传入 hermes 镜像

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	req = withTenantUin(req, whitelistTestUin)
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("agent_type 不匹配应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "agent_type 与请求不匹配") {
		t.Fatalf("错误信息应包含 'agent_type 与请求不匹配'，实际=%s", rr.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────
// 4. 白名单部署 + image_id 合法 → 越过 image_id 校验分支
// ──────────────────────────────────────────────────────────────────────────

// TestHandleCreateInstance_ImageIDOverride_WhitelistValidImage 验证：
// 白名单部署、image_id 存在且 agent_type 匹配时，不会落入新增分支的 400 错误。
// 由于测试环境无 CVMTemplate / Region 等下游配置，请求最终仍会失败（500），
// 但失败原因不应是 image_id 相关 —— 这等价于"覆盖镜像逻辑生效、流程继续"。
func TestHandleCreateInstance_ImageIDOverride_WhitelistValidImage(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	// 低版本 openclaw 镜像，作为 image_id 覆盖目标
	overrideImg := &model.AIImage{
		ImageId:      "img-b52f7vd0",
		ImageName:    "openclaw-4.23",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "4.23.0",
		Enabled:      false, // 故意非启用，验证只要 image_id 命中且类型一致即可
	}
	if err := model.DB(context.Background()).Create(overrideImg).Error; err != nil {
		t.Fatalf("seed override image: %v", err)
	}

	form := url.Values{}
	form.Set("name", "whitelist-ok")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("image_id", "img-b52f7vd0")

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	req = withTenantUin(req, whitelistTestUin)
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	// 关键断言：image_id 校验分支不能贡献错误。
	body := rr.Body.String()
	if strings.Contains(body, "指定的镜像") && strings.Contains(body, "不存在") {
		t.Fatalf("合法 image_id 不应触发 '不存在' 错误，body=%s", body)
	}
	if strings.Contains(body, "agent_type 与请求不匹配") {
		t.Fatalf("合法 image_id 不应触发 'agent_type 不匹配' 错误，body=%s", body)
	}
	// 不强求最终成功（下游 CVMTemplate 等未配置必然失败），
	// 只要不在 image_id 分支被拦截即可。
}

// ──────────────────────────────────────────────────────────────────────────
// 5. 未传 image_id：行为不受新增逻辑影响
// ──────────────────────────────────────────────────────────────────────────

// TestHandleCreateInstance_ImageIDOverride_NoParamUnaffected 验证：
// 未传 image_id 时，无论部署 UIN 是否在白名单，新增分支整体被短路，
// 行为完全等价于改动前。
func TestHandleCreateInstance_ImageIDOverride_NoParamUnaffected(t *testing.T) {
	cases := []struct {
		name string
		uin  string
	}{
		{name: "whitelist_uin_no_image_id", uin: whitelistTestUin},
		{name: "non_whitelist_uin_no_image_id", uin: "9999999999"},
		{name: "empty_uin_no_image_id", uin: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := initFiveHandlersTestDB(t)
			defer cleanup()

			user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
			if err := model.DB(context.Background()).Create(user).Error; err != nil {
				t.Fatalf("create user: %v", err)
			}
			seedOpenClawEnabledImage(t)

			form := url.Values{}
			form.Set("name", "no-image-id")
			form.Set("agent_type", model.AgentTypeOpenClaw)
			// 注意：不设置 image_id

			req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
			if tc.uin != "" {
				req = withTenantUin(req, tc.uin)
			}
			rr := httptest.NewRecorder()

			HandleCreateInstance(rr, req)

			body := rr.Body.String()
			if strings.Contains(body, "指定的镜像") {
				t.Fatalf("未传 image_id 不应进入镜像不存在分支，body=%s", body)
			}
			if strings.Contains(body, "agent_type 与请求不匹配") {
				t.Fatalf("未传 image_id 不应进入 agent_type 不匹配分支，body=%s", body)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────
// 6. image_id 仅含空白：等价于未传（strings.TrimSpace 短路）
// ──────────────────────────────────────────────────────────────────────────

func TestHandleCreateInstance_ImageIDOverride_BlankImageIDTreatedAsEmpty(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	form := url.Values{}
	form.Set("name", "blank-image-id")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("image_id", "   ") // 仅空白

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	// 即便注入白名单 UIN，空白 image_id 也应在 TrimSpace 后短路，不进入校验
	req = withTenantUin(req, whitelistTestUin)
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	body := rr.Body.String()
	if strings.Contains(body, "指定的镜像") {
		t.Fatalf("空白 image_id 应被视为未传，不应进入镜像校验，body=%s", body)
	}
	if strings.Contains(body, "agent_type 与请求不匹配") {
		t.Fatalf("空白 image_id 应被视为未传，不应进入 agent_type 校验，body=%s", body)
	}
}

// ──────────────────────────────────────────────────────────────────────────
// 7. 白名单部署 + image_id agent_type 为空字符串 → 通过 NormalizeAgentType 规范化为
//    openclaw，对 openclaw 请求不触发 mismatch
// ──────────────────────────────────────────────────────────────────────────

func TestHandleCreateInstance_ImageIDOverride_EmptyAgentTypeNormalizedAsOpenClaw(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	// 镜像 agent_type 为空，应被 NormalizeAgentType 归一化为 openclaw
	legacyImg := &model.AIImage{
		ImageId:      "img-legacy-empty-type",
		ImageName:    "legacy",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    "", // 存量空 agent_type
		AgentVersion: "3.0.0",
		Enabled:      false,
	}
	if err := model.DB(context.Background()).Create(legacyImg).Error; err != nil {
		t.Fatalf("seed legacy image: %v", err)
	}

	form := url.Values{}
	form.Set("name", "legacy-type-image")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("image_id", "img-legacy-empty-type")

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	req = withTenantUin(req, whitelistTestUin)
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if strings.Contains(rr.Body.String(), "agent_type 与请求不匹配") {
		t.Fatalf("空 agent_type 应被规范化为 openclaw，不应触发 mismatch，body=%s", rr.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────
// 8. IsInternalAccount 判定异常 → 降级为非白名单（image_id 静默忽略）
// ──────────────────────────────────────────────────────────────────────────

// TestHandleCreateInstance_ImageIDOverride_InternalAccountErrorFallback 验证：
// 当 ctx 中没有 UIN 且 CAM fetcher 返回错误时，IsInternalAccount 会返回 err；
// CreateInstance 必须按"非白名单"降级，行为等价于未传 image_id —— 即静默忽略，
// 不能因为参数校验失败而把请求打挂。
//
// 这是源码 1252-1258 行那段防御性降级分支的唯一覆盖入口。
func TestHandleCreateInstance_ImageIDOverride_InternalAccountErrorFallback(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	// 注入返回 error 的 fetcher，并在 t.Cleanup 中复原。
	origFetcher := cloudUinFetcher
	cloudUinFetcher = func(ctx context.Context) (string, error) {
		return "", errInternalAccountFake
	}
	ResetCAMUinCacheForTest()
	t.Cleanup(func() {
		cloudUinFetcher = origFetcher
		ResetCAMUinCacheForTest()
	})

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	form := url.Values{}
	form.Set("name", "internal-judge-error")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	// 故意写一个不存在的 image_id：若错误地把 isInternal 当作 true，会在 DB 查询失败处返回 400。
	form.Set("image_id", "img-should-be-ignored-on-error")

	// 关键：不调用 withTenantUin，让 ResolveCloudUin 落到 cloudUinFetcher 上去触发 err。
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	body := rr.Body.String()
	if strings.Contains(body, "指定的镜像") {
		t.Fatalf("IsInternalAccount 出错时应降级为非白名单（静默忽略 image_id），不应进入镜像校验，body=%s", body)
	}
	if strings.Contains(body, "agent_type 与请求不匹配") {
		t.Fatalf("IsInternalAccount 出错时应降级为非白名单（静默忽略 image_id），不应进入 agent_type 校验，body=%s", body)
	}
}

// errInternalAccountFake 仅供上面这个用例制造 CAM 失败信号，不暴露具体错误类型。
var errInternalAccountFake = errFake("simulated CAM failure")

type errFake string

func (e errFake) Error() string { return string(e) }

// ──────────────────────────────────────────────────────────────────────────
// 9. 白名单部署 + 不存在镜像 → 错误信息精确包含 image_id 原值
// ──────────────────────────────────────────────────────────────────────────

// 已有 case 2 仅断言含 image_id 与"未找到"，本用例严格校验消息模板
// "未找到镜像: <image_id>"（来自 i18n.MsgImageNotFoundByID），
// 防止后续重构静默改动用户可见文案。
func TestHandleCreateInstance_ImageIDOverride_NotFoundMessageTemplate(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	const wantImageID = "img-template-check"
	form := url.Values{}
	form.Set("name", "template-check")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("image_id", wantImageID)

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	req = withTenantUin(req, whitelistTestUin)
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	wantMsg := "未找到镜像: " + wantImageID
	if !strings.Contains(rr.Body.String(), wantMsg) {
		t.Fatalf("错误信息模板与代码不一致，want contains %q, body=%s", wantMsg, rr.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────
// 10. 白名单部署 + image_id 含前后空白 → TrimSpace 后正常命中镜像
// ──────────────────────────────────────────────────────────────────────────

// 验证 strings.TrimSpace 既作用于"是否进入分支"的判定（空白等价于空），
// 也确保 *非纯空白* 的前后空白被正确去除：
// 比如 "  img-xxx  " 经 TrimSpace 后应作为 "img-xxx" 去查 DB。
//
// 若未来有人误把 TrimSpace 改成只在"等于空"判定时使用、却把原值送进 Where，
// 这个用例就会立刻失败。
func TestHandleCreateInstance_ImageIDOverride_TrimSpaceUsedForQuery(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	overrideImg := &model.AIImage{
		ImageId:      "img-trim-ok",
		ImageName:    "trim-ok",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "4.0.0",
		Enabled:      false,
	}
	if err := model.DB(context.Background()).Create(overrideImg).Error; err != nil {
		t.Fatalf("seed override image: %v", err)
	}

	form := url.Values{}
	form.Set("name", "trim-space-image-id")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("image_id", "  img-trim-ok  ") // 前后空白

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	req = withTenantUin(req, whitelistTestUin)
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	body := rr.Body.String()
	// 若 TrimSpace 未生效，DB 查询会用 "  img-trim-ok  " 找不到，触发 400 "未找到镜像"。
	if strings.Contains(body, "未找到镜像") && strings.Contains(body, "img-trim-ok") {
		t.Fatalf("前后空白应被 TrimSpace 去除，不应触发 '未找到镜像' 错误，body=%s", body)
	}
	if strings.Contains(body, "agent_type 与请求不匹配") {
		t.Fatalf("合法镜像不应触发 mismatch，body=%s", body)
	}
}

// ──────────────────────────────────────────────────────────────────────────
// 11. 非白名单部署：image_id 指向"agent_type 不匹配"的真实镜像 → 仍应静默忽略
// ──────────────────────────────────────────────────────────────────────────

// case 1 用的是"不存在的 image_id"，本用例改用"存在但 agent_type 不匹配"的镜像，
// 更严格地证明：非白名单部署完全不进入 else 块（既不查 DB、也不做 agent_type 校验）。
func TestHandleCreateInstance_ImageIDOverride_NonWhitelistSkipsAllChecks(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedOpenClawEnabledImage(t)

	// 存在的 hermes 镜像，agent_type 与请求的 openclaw 不一致
	hermesImg := &model.AIImage{
		ImageId:      "img-hermes-real",
		ImageName:    "hermes-real",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeHermes,
		AgentVersion: "1.0.0",
		Enabled:      false,
	}
	if err := model.DB(context.Background()).Create(hermesImg).Error; err != nil {
		t.Fatalf("seed hermes image: %v", err)
	}

	form := url.Values{}
	form.Set("name", "non-whitelist-skip-all")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("image_id", "img-hermes-real")

	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	req = withTenantUin(req, "8888888888") // 非白名单
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	body := rr.Body.String()
	if strings.Contains(body, "agent_type 与请求不匹配") {
		t.Fatalf("非白名单部署不应进入 agent_type 校验，body=%s", body)
	}
	if strings.Contains(body, "指定的镜像") {
		t.Fatalf("非白名单部署不应进入镜像校验，body=%s", body)
	}
}

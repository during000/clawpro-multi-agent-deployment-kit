package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/i18n"
	"hatchery/model"
)

// ─────────────────────────────────────────────────────────────────────
// HandleCreateInstance 中两条错误路径的 detail 透传测试：
//
//   ① groupID == 0 路径：model.GetRestrictedImageTypes 失败
//      → 应返回 MsgQueryRestrictedImageTypesFailed + DB 错误 detail
//
//   ② groupID > 0  路径：usergroup.ResolveImageTypes 失败（底层同样依赖
//      GetRestrictedImageTypes，但走的是不同的代码分支）
//      → 应返回 MsgResolveVisibleImageTypesFailed + DB 错误 detail
//
// 触发手段：先用 initCoverageTestDB 建好全部表，然后主动 DROP
// `group_config_bindings`，让两条路径在跑到 GetRestrictedImageTypes 时
// 自然报 "no such table" 错误。
// ─────────────────────────────────────────────────────────────────────

// dropGroupConfigBindingsTable 主动删表，触发 GetRestrictedImageTypes DB 错误。
func dropGroupConfigBindingsTable(t *testing.T) {
	t.Helper()
	if err := model.DB(context.Background()).Migrator().DropTable(&model.GroupConfigBinding{}); err != nil {
		t.Fatalf("drop group_config_bindings 失败: %v", err)
	}
}

func setupCreateInstanceForRestrictedImageTypesErr(t *testing.T) (user *model.User, cleanup func()) {
	cleanup = initCoverageTestDB(t)

	user = &model.User{Username: "u1", Password: "x", Role: "user", InstanceQuota: 10}
	model.DB(context.Background()).Create(user)

	// 建一个 openclaw 镜像，让 agent_type 校验 / 镜像查询通过，
	// 流程才会走到 GetRestrictedImageTypes 那一段。
	img := &model.AIImage{
		ImageId: "img-restricted-err", ImageName: "test",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	}
	model.DB(context.Background()).Create(img)

	dropGroupConfigBindingsTable(t)
	return user, cleanup
}

func parseCreateInstResp(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v, body=%s", err, rr.Body.String())
	}
	return resp
}

// ① groupID == 0 路径
func TestHandleCreateInstance_RestrictedImageTypesQueryFails_DetailIncluded(t *testing.T) {
	_, cleanup := setupCreateInstanceForRestrictedImageTypesErr(t)
	defer cleanup()

	form := url.Values{}
	form.Set("name", "test-restricted-err")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	// 不传 group_id → 走 L1196 那条 else 分支：直接 GetRestrictedImageTypes
	req := coverageReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("应返回 500, got %d body=%s", rr.Code, rr.Body.String())
	}
	resp := parseCreateInstResp(t, rr)

	// 验证 i18n key：MsgQueryRestrictedImageTypesFailed → "查询受限镜像类型列表失败"
	wantMsg := i18nMustT(req.Context(), i18n.MsgQueryRestrictedImageTypesFailed)
	if got, _ := resp["error"].(string); got != wantMsg {
		t.Errorf("error 字段不对, got %q, want %q", got, wantMsg)
	}

	// 验证 detail 包含底层 DB 错误
	detail, _ := resp["detail"].(string)
	if detail == "" {
		t.Fatal("detail 应非空")
	}
	if !strings.Contains(detail, "no such table") {
		t.Errorf("detail 应包含底层 SQL 错误, got %q", detail)
	}
}

// ② groupID > 0 路径
func TestHandleCreateInstance_ResolveImageTypesQueryFails_DetailIncluded(t *testing.T) {
	user, cleanup := setupCreateInstanceForRestrictedImageTypesErr(t)
	defer cleanup()
	_ = user

	// 用户分组：需要建一个 UserGroup 让 ValidateGroupIDs 通过
	grp := &model.UserGroup{Name: "g1"}
	model.DB(context.Background()).Create(grp)

	form := url.Values{}
	form.Set("name", "test-resolve-err")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("group_id", uintToStr(grp.ID))
	req := coverageReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("应返回 500, got %d body=%s", rr.Code, rr.Body.String())
	}
	resp := parseCreateInstResp(t, rr)

	// 验证 i18n key：MsgResolveVisibleImageTypesFailed
	wantMsg := i18nMustT(req.Context(), i18n.MsgResolveVisibleImageTypesFailed)
	if got, _ := resp["error"].(string); got != wantMsg {
		t.Errorf("error 字段不对, got %q, want %q", got, wantMsg)
	}

	detail, _ := resp["detail"].(string)
	if detail == "" {
		t.Fatal("detail 应非空")
	}
	if !strings.Contains(detail, "no such table") {
		t.Errorf("detail 应包含底层 SQL 错误, got %q", detail)
	}
}

// ── 小工具 ────────────────────────────────────────────────────────────

func i18nMustT(ctx context.Context, key i18n.Key, args ...any) string {
	return i18n.T(ctx, key, args...)
}

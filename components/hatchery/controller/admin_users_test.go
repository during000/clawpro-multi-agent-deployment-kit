package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// initAdminTestDB 初始化内存 SQLite 数据库，并迁移所需表（含 SiteConfig）。
func initAdminTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SiteConfig{},
		&model.Tag{},
		&model.TagVisibilityGroup{},
		&model.UserGroup{},
		&model.UserGroupMember{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// 确保 SiteConfig 存在（createOneUser 依赖 GetSiteConfig）
	db.Create(&model.SiteConfig{})
}

// ─── createOneUser 单元测试 ─────────────────────────────────────────────────

func TestCreateOneUser_Success(t *testing.T) {
	initAdminTestDB(t)

	newUser, status, err := createOneUser(context.Background(), createUserParams{
		Username: "testuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("期望成功，实际错误: status=%d, err=%v", status, err)
	}
	if newUser == nil || newUser.ID == 0 {
		t.Fatal("期望返回非空用户对象且 ID 非零")
	}

	// 验证用户确实存在于数据库中
	var user model.User
	if model.DB(context.Background()).Where("id = ?", newUser.ID).First(&user).Error != nil {
		t.Fatalf("数据库中未找到 ID=%d 的用户", newUser.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("期望 username=testuser，实际=%s", user.Username)
	}
	if !user.HasAPIToken() {
		t.Error("创建成功的用户应自动拥有 API Token")
	}
}

func TestCreateOneUser_EmptyUsername(t *testing.T) {
	initAdminTestDB(t)

	newUser, status, err := createOneUser(context.Background(), createUserParams{
		Username: "",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("期望错误，实际成功")
	}
	if newUser != nil {
		t.Errorf("期望 newUser=nil，实际=%+v", newUser)
	}
	if status != http.StatusBadRequest {
		t.Errorf("期望 status=400，实际=%d", status)
	}
}

func TestCreateOneUser_EmptyPassword(t *testing.T) {
	initAdminTestDB(t)

	newUser, status, err := createOneUser(context.Background(), createUserParams{
		Username: "testuser",
		Password: "",
	})
	if err == nil {
		t.Fatal("期望错误，实际成功")
	}
	if newUser != nil {
		t.Errorf("期望 newUser=nil，实际=%+v", newUser)
	}
	if status != http.StatusBadRequest {
		t.Errorf("期望 status=400，实际=%d", status)
	}
}

func TestCreateOneUser_DuplicateUsername(t *testing.T) {
	initAdminTestDB(t)

	_, _, err := createOneUser(context.Background(), createUserParams{
		Username: "dupuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("第一次创建失败: %v", err)
	}

	newUser, status, err := createOneUser(context.Background(), createUserParams{
		Username: "dupuser",
		Password: "password456",
	})
	if err == nil {
		t.Fatal("期望重复用户名错误，实际成功")
	}
	if newUser != nil {
		t.Errorf("期望 newUser=nil，实际=%+v", newUser)
	}
	if status != http.StatusConflict {
		t.Errorf("期望 status=409，实际=%d", status)
	}
}

func TestCreateOneUser_ReturnsCorrectID(t *testing.T) {
	initAdminTestDB(t)

	u1, _, err1 := createOneUser(context.Background(), createUserParams{Username: "user1", Password: "p1"})
	u2, _, err2 := createOneUser(context.Background(), createUserParams{Username: "user2", Password: "p2"})
	if err1 != nil || err2 != nil {
		t.Fatalf("创建用户失败: err1=%v, err2=%v", err1, err2)
	}
	if u1.ID == u2.ID {
		t.Errorf("两个不同用户的 ID 不应相同: id1=%d, id2=%d", u1.ID, u2.ID)
	}
	if u2.ID <= u1.ID {
		t.Errorf("第二个用户的 ID 应大于第一个: id1=%d, id2=%d", u1.ID, u2.ID)
	}
}

func TestCreateOneUser_TokenQuotaRulesFallbackToDefaultFirstRule(t *testing.T) {
	initAdminTestDB(t)
	ctx := context.Background()
	model.DB(ctx).Model(&model.SiteConfig{}).Where("id > 0").Updates(map[string]any{
		"default_token_quota_day":   -1,
		"default_token_quota_rules": `[{"mode":"day","limit":700000}]`,
	})

	raw := `[{"mode":"month","start":1},{"mode":"custom","refresh":"monthly"}]`
	newUser, status, err := createOneUser(ctx, createUserParams{
		Username:           "rules-fallback",
		Password:           "password123",
		TokenQuotaRulesRaw: &raw,
	})
	if err != nil {
		t.Fatalf("期望成功，实际错误: status=%d, err=%v", status, err)
	}

	var user model.User
	if err := model.DB(ctx).First(&user, newUser.ID).Error; err != nil {
		t.Fatalf("查询新建用户失败: %v", err)
	}
	rules, ok := model.ParseTokenQuotaRules(user.TokenQuotaRules)
	if !ok || len(rules) != 2 {
		t.Fatalf("token_quota_rules 解析失败: %s", user.TokenQuotaRules)
	}
	if rules[0].Mode != model.QuotaModeMonth || rules[0].Limit != 700000 || rules[0].Start != nil || rules[0].Refresh != "" {
		t.Fatalf("month rule 应继承默认 limit 且忽略 custom-only 字段，实际=%+v", rules[0])
	}
	if rules[1].Mode != model.QuotaModeCustom || rules[1].Limit != 700000 || rules[1].Refresh != model.QuotaRefreshMonthly || rules[1].Start == nil {
		t.Fatalf("custom rule 应继承默认 limit 并自动填 start，实际=%+v", rules[1])
	}
}

func TestCreateOneUser_TokenQuotaRulesDoesNotFallbackCustomStart(t *testing.T) {
	initAdminTestDB(t)
	ctx := context.Background()
	defaultStart := int64(111)
	model.DB(ctx).Model(&model.SiteConfig{}).Where("id > 0").Updates(map[string]any{
		"default_token_quota_day":   -1,
		"default_token_quota_rules": `[{"mode":"custom","limit":300000,"start":111,"refresh":"monthly"}]`,
	})

	raw := `[{"refresh":"daily"}]`
	newUser, status, err := createOneUser(ctx, createUserParams{
		Username:           "rules-start-now",
		Password:           "password123",
		TokenQuotaRulesRaw: &raw,
	})
	if err != nil {
		t.Fatalf("期望成功，实际错误: status=%d, err=%v", status, err)
	}

	var user model.User
	if err := model.DB(ctx).First(&user, newUser.ID).Error; err != nil {
		t.Fatalf("查询新建用户失败: %v", err)
	}
	rules, ok := model.ParseTokenQuotaRules(user.TokenQuotaRules)
	if !ok || len(rules) != 1 {
		t.Fatalf("token_quota_rules 解析失败: %s", user.TokenQuotaRules)
	}
	if rules[0].Mode != model.QuotaModeCustom || rules[0].Limit != 300000 || rules[0].Refresh != model.QuotaRefreshDaily {
		t.Fatalf("custom rule 应继承默认 mode/limit 并覆盖 refresh，实际=%+v", rules[0])
	}
	if rules[0].Start == nil || *rules[0].Start == defaultStart {
		t.Fatalf("custom start 未传时应按创建时间自动填充，不应继承默认 start，实际=%+v", rules[0])
	}
}

func TestCreateOneUser_TokenQuotaRulesFallbackToLegacyDefaultDay(t *testing.T) {
	initAdminTestDB(t)
	ctx := context.Background()
	model.DB(ctx).Model(&model.SiteConfig{}).Where("id > 0").Updates(map[string]any{
		"default_token_quota_day":   -1,
		"default_token_quota_rules": "",
	})

	raw := `[{"mode":"month"}]`
	newUser, status, err := createOneUser(ctx, createUserParams{
		Username:           "rules-legacy-day",
		Password:           "password123",
		TokenQuotaRulesRaw: &raw,
	})
	if err != nil {
		t.Fatalf("期望成功，实际错误: status=%d, err=%v", status, err)
	}

	var user model.User
	if err := model.DB(ctx).First(&user, newUser.ID).Error; err != nil {
		t.Fatalf("查询新建用户失败: %v", err)
	}
	rules, ok := model.ParseTokenQuotaRules(user.TokenQuotaRules)
	if !ok || len(rules) != 1 {
		t.Fatalf("token_quota_rules 解析失败: %s", user.TokenQuotaRules)
	}
	if rules[0].Mode != model.QuotaModeMonth || rules[0].Limit != -1 {
		t.Fatalf("默认 rules 为空时应从 legacy day=-1 生成显式 fallback，实际=%+v", rules[0])
	}
}

// ─── HandleCreateUser JSON 响应测试 ─────────────────────────────────────────

// createUserHandler 绕过 requireAdmin，直接执行 HandleCreateUser 的核心逻辑。
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	p := createUserParams{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
		Email:    r.FormValue("email"),
		Role:     r.FormValue("role"),
	}

	newUser, status, err := createOneUser(context.Background(), p)
	if err != nil {
		writeError(w, r, status, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true, "id": newUser.ID})
}

func TestHandleCreateUser_JSONReturnsID(t *testing.T) {
	initAdminTestDB(t)

	form := url.Values{}
	form.Set("username", "jsonuser")
	form.Set("password", "password123")
	form.Set("role", "user")

	req := httptest.NewRequest(http.MethodPost, "/admin/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	createUserHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}
	if resp["id"] == nil {
		t.Fatal("期望响应中包含 id 字段，实际为 nil")
	}
	id := resp["id"].(float64)
	if id <= 0 {
		t.Errorf("期望 id > 0，实际=%v", id)
	}
	var u model.User
	if model.DB(context.Background()).Where("id = ?", uint(id)).First(&u).Error != nil {
		t.Fatalf("未找到新建用户 id=%v", id)
	}
	if !u.HasAPIToken() {
		t.Error("JSON 创建成功的用户应自动拥有 API Token")
	}
}

// batchCreateUserHandler 绕过 requireAdmin，与 HandleBatchCreateUser 相同的创建循环。
func batchCreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jsonAPI(w)

	var items []struct {
		Username        string          `json:"username"`
		Password        string          `json:"password"`
		Role            string          `json:"role"`
		Email           string          `json:"email"`
		InstanceQuota   *int            `json:"instance_quota"`
		TokenQuotaDay   *int            `json:"token_quota_day"`
		TokenQuotaRules json.RawMessage `json:"token_quota_rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}

	type result struct {
		Username string `json:"username"`
		ID       uint   `json:"id,omitempty"`
		OK       bool   `json:"ok"`
		Error    string `json:"error,omitempty"`
	}
	results := make([]result, 0, len(items))
	for _, item := range items {
		res := result{Username: item.Username}
		p := createUserParams{
			Username:      item.Username,
			Password:      item.Password,
			Email:         item.Email,
			Role:          item.Role,
			InstanceQuota: item.InstanceQuota,
			TokenQuotaDay: item.TokenQuotaDay,
		}
		if len(item.TokenQuotaRules) > 0 && string(item.TokenQuotaRules) != "null" {
			raw := string(item.TokenQuotaRules)
			p.TokenQuotaRulesRaw = &raw
		}
		if newUser, _, err := createOneUser(context.Background(), p); err != nil {
			res.Error = err.Error()
		} else {
			res.OK = true
			res.ID = newUser.ID
		}
		results = append(results, res)
	}
	jsonOK(w, map[string]interface{}{"results": results})
}

func TestBatchCreateUser_MixedSuccessUsersHaveToken(t *testing.T) {
	initAdminTestDB(t)
	body := `[{"username":"b1","password":"p1"},{"username":"b1","password":"p2"},{"username":"b3","password":"p3"}]`
	req := httptest.NewRequest(http.MethodPost, "/admin/batch-create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	batchCreateUserHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Results []struct {
			Username string `json:"username"`
			ID       uint   `json:"id"`
			OK       bool   `json:"ok"`
			Error    string `json:"error,omitempty"`
		} `json:"results"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应: %v", err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("期望 3 条结果，实际=%d", len(resp.Results))
	}
	if !resp.Results[0].OK || resp.Results[1].OK || !resp.Results[2].OK {
		t.Fatalf("期望 ok 依次为 true,false,true: %+v", resp.Results)
	}
	if resp.Results[1].Error == "" {
		t.Fatal("第二条应为重复用户名错误")
	}
	for _, id := range []uint{resp.Results[0].ID, resp.Results[2].ID} {
		var u model.User
		if model.DB(context.Background()).Where("id = ?", id).First(&u).Error != nil {
			t.Fatalf("用户 id=%d 不存在", id)
		}
		if !u.HasAPIToken() {
			t.Errorf("批量创建成功的用户 id=%d 应有 API Token", id)
		}
	}
}

func TestBatchCreateUser_TokenQuotaRules(t *testing.T) {
	initAdminTestDB(t)
	body := `[{"username":"rules-user","password":"p1","token_quota_day":1,"token_quota_rules":[{"mode":"month","limit":2000}]}]`
	req := httptest.NewRequest(http.MethodPost, "/admin/batch-create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	batchCreateUserHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Results []struct {
			Username string `json:"username"`
			ID       uint   `json:"id"`
			OK       bool   `json:"ok"`
			Error    string `json:"error,omitempty"`
		} `json:"results"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应: %v", err)
	}
	if len(resp.Results) != 1 || !resp.Results[0].OK {
		t.Fatalf("期望批量创建成功: %+v", resp.Results)
	}

	var u model.User
	if model.DB(context.Background()).Where("id = ?", resp.Results[0].ID).First(&u).Error != nil {
		t.Fatalf("未找到新建用户 id=%d", resp.Results[0].ID)
	}
	if u.TokenQuotaDay != -1 {
		t.Fatalf("传 token_quota_rules 后 token_quota_day 应迁移为 -1，实际=%d", u.TokenQuotaDay)
	}
	if u.TokenQuotaRules != `[{"mode":"month","limit":2000}]` {
		t.Fatalf("期望写入 month rules，实际=%s", u.TokenQuotaRules)
	}
}

func TestHandleBatchCreateUser_DecodeErrorPerItem(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := `[
		{"username":"bad-group","password":"p1","group_ids":[1]},
		{"username":"ok-user","password":"p2"}
	]`
	req := httptest.NewRequest(http.MethodPost, "/admin/batch-create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleBatchCreateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Results []struct {
			Username  string `json:"username"`
			ID        uint   `json:"id"`
			OK        bool   `json:"ok"`
			Error     string `json:"error,omitempty"`
			ErrorCode string `json:"error_code,omitempty"`
		} `json:"results"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("期望 2 条结果，实际=%d: %+v", len(resp.Results), resp.Results)
	}
	if resp.Results[0].Username != "bad-group" || resp.Results[0].OK || resp.Results[0].ErrorCode != "invalid_params" || !strings.Contains(resp.Results[0].Error, "group_ids") {
		t.Fatalf("第一条应返回 group_ids 单项错误: %+v", resp.Results[0])
	}
	if resp.Results[1].Username != "ok-user" || !resp.Results[1].OK || resp.Results[1].ID == 0 {
		t.Fatalf("第二条应继续创建成功: %+v", resp.Results[1])
	}
	var count int64
	model.DB(context.Background()).Model(&model.User{}).Where("username = ?", "ok-user").Count(&count)
	if count != 1 {
		t.Fatalf("ok-user 应被创建，实际 count=%d", count)
	}
}

func TestHandleCreateUser_JSONError(t *testing.T) {
	initAdminTestDB(t)

	form := url.Values{}
	form.Set("username", "")
	form.Set("password", "")

	req := httptest.NewRequest(http.MethodPost, "/admin/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	createUserHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["error"] == nil {
		t.Fatal("期望响应中包含 error 字段")
	}
}

// ─── HandleAdminUserToken 单元测试 ──────────────────────────────────────────

// adminUserTokenHandler 绕过 requireAdmin，直接执行 HandleAdminUserToken 的核心逻辑。
func adminUserTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jsonAPI(w)

	id := r.URL.Query().Get("id")
	var user model.User
	if model.DB(context.Background()).Where("id = ?", id).First(&user).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
		return
	}

	if !user.HasAPIToken() {
		jsonOK(w, map[string]interface{}{
			"exists": false,
		})
		return
	}

	jsonOK(w, map[string]interface{}{
		"exists":     true,
		"token":      *user.APIToken,
		"disabled":   user.APITokenDisabled,
		"created_at": user.APITokenCreatedAt,
	})
}

func TestHandleAdminUserToken_UserNotFound(t *testing.T) {
	initAdminTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/user-token?id=99999", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	adminUserTokenHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Fatal("期望响应中包含 error 字段")
	}
}

func TestHandleAdminUserToken_NoToken(t *testing.T) {
	initAdminTestDB(t)

	// 创建一个没有 Token 的用户
	user := model.User{Username: "notoken", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/user-token?id=%d", user.ID), nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	adminUserTokenHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["exists"] != false {
		t.Errorf("期望 exists=false，实际=%v", resp["exists"])
	}
}

func TestHandleAdminUserToken_HasToken(t *testing.T) {
	initAdminTestDB(t)

	// 创建用户并生成 Token
	user := model.User{Username: "hastoken", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	token, err := model.GenerateAPIToken(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("生成 Token 失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/user-token?id=%d", user.ID), nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	adminUserTokenHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["exists"] != true {
		t.Errorf("期望 exists=true，实际=%v", resp["exists"])
	}
	if resp["token"] == nil {
		t.Fatal("期望响应中包含 token 字段")
	}
	if resp["token"].(string) != token {
		t.Errorf("期望 token=%s，实际=%s", token, resp["token"])
	}
	if resp["disabled"] != false {
		t.Errorf("期望 disabled=false，实际=%v", resp["disabled"])
	}
}

func TestHandleAdminUserToken_DisabledToken(t *testing.T) {
	initAdminTestDB(t)

	// 创建用户、生成 Token、然后禁用
	user := model.User{Username: "disabledtoken", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	_, err := model.GenerateAPIToken(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("生成 Token 失败: %v", err)
	}
	if err := model.SetAPITokenDisabled(context.Background(), user.ID, true); err != nil {
		t.Fatalf("禁用 Token 失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/user-token?id=%d", user.ID), nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	adminUserTokenHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["exists"] != true {
		t.Errorf("期望 exists=true，实际=%v", resp["exists"])
	}
	if resp["disabled"] != true {
		t.Errorf("期望 disabled=true，实际=%v", resp["disabled"])
	}
}

func TestHandleAdminUserToken_MethodNotAllowed(t *testing.T) {
	initAdminTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/user-token?id=1", nil)
	w := httptest.NewRecorder()

	adminUserTokenHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("期望 405，实际=%d", w.Code)
	}
}

// ─── HandleResetPassword 单元测试 ────────────────────────────────────────────

// resetPasswordCoreHandler 绕过 requireAdmin，直接走 HandleResetPassword 的业务函数。
// 注意这里必须和真实 handler 的鉴权顺序保持一致：先定位用户 → 命中初始管理员则
// 强制 admin-token 校验 → 再执行 resetPasswordCore，否则会漏测真实鉴权路径。
func resetPasswordCoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jsonAPI(w)

	initUser := r.URL.Query().Get("init_user") == "true"
	id := r.URL.Query().Get("id")
	password := r.FormValue("password")

	if password == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPasswordCannotBeEmpty))
		return
	}

	user, err := resolveResetPasswordUser(context.Background(), id, initUser)
	if err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if initUser || user.IsInitialAdmin(context.Background()) {
		tokenUser, _ := getUserFromToken(r)
		if tokenUser == nil || tokenUser.ID != 0 {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgInitialAdminPasswordReset))
			return
		}
	}
	if err := resetPasswordCore(context.Background(), user, password); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}

// newResetPasswordRequest 构造重置密码请求（含 form body、Accept JSON、可选 Bearer token）。
func newResetPasswordRequest(method, query, password, bearer string) *http.Request {
	form := url.Values{}
	if password != "" {
		form.Set("password", password)
	}
	req := httptest.NewRequest(method, "/admin/reset-password?"+query, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return req
}

// TestHandleResetPassword_GET_MethodNotAllowed 验证 GET 请求返回 405。
func TestHandleResetPassword_GET_MethodNotAllowed(t *testing.T) {
	initAdminTestDB(t)

	w := httptest.NewRecorder()
	resetPasswordCoreHandler(w, newResetPasswordRequest(http.MethodGet, "id=1", "", ""))

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("期望 405，实际=%d", w.Code)
	}
}

// TestHandleResetPassword_EmptyPassword 验证密码为空时返回 400。
func TestHandleResetPassword_EmptyPassword(t *testing.T) {
	initAdminTestDB(t)

	w := httptest.NewRecorder()
	resetPasswordCoreHandler(w, newResetPasswordRequest(http.MethodPost, "id=1", "", ""))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Fatal("期望响应中包含 error 字段")
	}
}

// TestHandleResetPassword_ByID_UserNotFound 验证按 id 查询不存在时返回 404。
func TestHandleResetPassword_ByID_UserNotFound(t *testing.T) {
	initAdminTestDB(t)

	w := httptest.NewRecorder()
	resetPasswordCoreHandler(w, newResetPasswordRequest(http.MethodPost, "id=99999", "newpass", ""))

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// TestHandleResetPassword_ByID_Success 验证按 id 正常重置普通用户密码。
func TestHandleResetPassword_ByID_Success(t *testing.T) {
	initAdminTestDB(t)

	user := model.User{Username: "normaluser", Password: "oldpass", Role: "user"}
	model.DB(context.Background()).Create(&user)

	w := httptest.NewRecorder()
	resetPasswordCoreHandler(w, newResetPasswordRequest(http.MethodPost,
		fmt.Sprintf("id=%d", user.ID), "newpass123", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}
}

// TestHandleResetPassword_ByID_InitialAdmin_WithoutAdminToken 回归测试：
// 即使调用方传 id=<初始管理员 ID>（而非 init_user=true），也必须走 admin-token 校验。
// 若漏掉这一条，恶意管理员可以通过显式 id 重置初始管理员密码绕过保护。
func TestHandleResetPassword_ByID_InitialAdmin_WithoutAdminToken(t *testing.T) {
	initAdminTestDB(t)

	admin := model.User{Username: "initadmin", Password: "pass", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	AdminToken = "test-admin-token"
	defer func() { AdminToken = "" }()

	w := httptest.NewRecorder()
	resetPasswordCoreHandler(w, newResetPasswordRequest(http.MethodPost,
		fmt.Sprintf("id=%d", admin.ID), "newpass", ""))

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// TestHandleResetPassword_InitUser_NoAdmin 验证 init_user=true 但 DB 无 admin 时返回 404。
func TestHandleResetPassword_InitUser_NoAdmin(t *testing.T) {
	initAdminTestDB(t)

	AdminToken = "test-admin-token"
	defer func() { AdminToken = "" }()

	w := httptest.NewRecorder()
	resetPasswordCoreHandler(w, newResetPasswordRequest(http.MethodPost,
		"init_user=true", "newpass", "test-admin-token"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// TestHandleResetPassword_InitUser_WithoutAdminToken 验证 init_user=true 但未用 admin-token 时返回 403。
func TestHandleResetPassword_InitUser_WithoutAdminToken(t *testing.T) {
	initAdminTestDB(t)

	admin := model.User{Username: "admin", Password: "pass", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	AdminToken = "test-admin-token"
	defer func() { AdminToken = "" }()

	w := httptest.NewRecorder()
	resetPasswordCoreHandler(w, newResetPasswordRequest(http.MethodPost,
		"init_user=true", "newpass", ""))

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Fatal("期望响应中包含 error 字段")
	}
}

// TestHandleResetPassword_InitUser_WithAdminToken 验证 init_user=true + admin-token 正常重置初始管理员密码。
func TestHandleResetPassword_InitUser_WithAdminToken(t *testing.T) {
	initAdminTestDB(t)

	admin := model.User{Username: "initadmin", Password: "oldpass", Role: "admin"}
	model.DB(context.Background()).Create(&admin)
	admin2 := model.User{Username: "admin2", Password: "pass", Role: "admin"}
	model.DB(context.Background()).Create(&admin2)

	AdminToken = "test-admin-token"
	defer func() { AdminToken = "" }()

	w := httptest.NewRecorder()
	resetPasswordCoreHandler(w, newResetPasswordRequest(http.MethodPost,
		"init_user=true", "brand-new-pass", "test-admin-token"))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}

	// 验证密码确实被修改
	var updated model.User
	model.DB(context.Background()).Where("id = ?", admin.ID).First(&updated)
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("brand-new-pass")); err != nil {
		t.Errorf("密码未被正确更新: %v", err)
	}
	// 验证 admin2 不受影响
	var untouched model.User
	model.DB(context.Background()).Where("id = ?", admin2.ID).First(&untouched)
	if bcrypt.CompareHashAndPassword([]byte(untouched.Password), []byte("brand-new-pass")) == nil {
		t.Error("admin2 的密码不应被修改")
	}
}

// ─── FilterInstancesByState 单元测试 ────────────────────────────────────────

func TestFilterInstancesByState_LimitApplied(t *testing.T) {
	// 创建 mock HTTP server，拦截 CVM DescribeInstances 请求
	var receivedLimit int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 解析请求体，提取 Limit 参数
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err == nil {
			if lim, ok := reqBody["Limit"]; ok {
				receivedLimit = int64(lim.(float64))
			}
		}
		// 返回空 InstanceSet（只验证 Limit 被正确设置）
		resp := map[string]interface{}{
			"Response": map[string]interface{}{
				"InstanceSet": []interface{}{},
				"TotalCount":  0,
				"RequestId":   "test-request-id",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建指向 mock server 的 CVM client
	credential := &common.Credential{}
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = server.Listener.Addr().String()
	cpf.HttpProfile.Scheme = "http"
	client, err := cvm.NewClient(credential, "ap-guangzhou", cpf)
	if err != nil {
		t.Fatalf("创建 CVM mock client 失败: %v", err)
	}

	// 传入 30 个实例 ID（超过默认 Limit=20），验证 Limit 等于实例数量
	ids := make([]string, 30)
	for i := range ids {
		ids[i] = fmt.Sprintf("ins-%08x", i)
	}

	_, err = FilterInstancesByState(client, ids, "RUNNING")
	if err != nil {
		t.Fatalf("FilterInstancesByState 失败: %v", err)
	}

	if receivedLimit != 30 {
		t.Errorf("期望 Limit=30 (实例数量), 实际 Limit=%d", receivedLimit)
	}
}

func TestFilterInstancesByState_BatchChunking(t *testing.T) {
	// 验证分批逻辑: 150 个实例应分 2 批 (100+50)，每批 Limit 正确
	var receivedLimits []int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err == nil {
			if lim, ok := reqBody["Limit"]; ok {
				receivedLimits = append(receivedLimits, int64(lim.(float64)))
			}
		}
		resp := map[string]interface{}{
			"Response": map[string]interface{}{
				"InstanceSet": []interface{}{},
				"TotalCount":  0,
				"RequestId":   "test-request-id",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	credential := &common.Credential{}
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = server.Listener.Addr().String()
	cpf.HttpProfile.Scheme = "http"
	client, err := cvm.NewClient(credential, "ap-guangzhou", cpf)
	if err != nil {
		t.Fatalf("创建 CVM mock client 失败: %v", err)
	}

	ids := make([]string, 150)
	for i := range ids {
		ids[i] = fmt.Sprintf("ins-%08x", i)
	}

	_, err = FilterInstancesByState(client, ids, "RUNNING")
	if err != nil {
		t.Fatalf("FilterInstancesByState 失败: %v", err)
	}

	if len(receivedLimits) != 2 {
		t.Fatalf("期望 2 批请求, 实际=%d", len(receivedLimits))
	}
	if receivedLimits[0] != 100 {
		t.Errorf("第1批期望 Limit=100, 实际=%d", receivedLimits[0])
	}
	if receivedLimits[1] != 50 {
		t.Errorf("第2批期望 Limit=50, 实际=%d", receivedLimits[1])
	}
}

// ─── HandleDepartments 单元测试 ─────────────────────────────────────────────

// TestHandleDepartments_MethodNotAllowed 验证 POST /admin/departments 返回 405。
// 该校验位于 requireAdmin 之前，因此无需构造管理员 session 即可触发。
func TestHandleDepartments_MethodNotAllowed(t *testing.T) {
	cases := []struct {
		name   string
		method string
	}{
		{"POST", http.MethodPost},
		{"PUT", http.MethodPut},
		{"DELETE", http.MethodDelete},
		{"PATCH", http.MethodPatch},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, "/admin/departments", nil)
			req.Header.Set("Accept", "application/json")
			rec := httptest.NewRecorder()

			HandleDepartments(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("期望 status=405, 实际=%d, body=%s", rec.Code, rec.Body.String())
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Errorf("期望 Content-Type 含 application/json, 实际=%q", ct)
			}
			var resp map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("响应应为 JSON: %v, body=%s", err, rec.Body.String())
			}
			if !strings.Contains(resp["error"].(string), "GET") {
				t.Errorf("error 信息应提示仅支持 GET, 实际=%q", resp["error"])
			}
		})
	}
}

// ─── HandleUserLimit 单元测试 ─────────────────────────────────────────────

// TestHandleUserLimit_MethodNotAllowed 验证 POST/PUT/DELETE/PATCH /admin/user-limit 返回 405。
// 该校验位于 requireAdmin 之前，因此无需构造管理员 session 即可触发。
func TestHandleUserLimit_MethodNotAllowed(t *testing.T) {
	cases := []struct {
		name   string
		method string
	}{
		{"POST", http.MethodPost},
		{"PUT", http.MethodPut},
		{"DELETE", http.MethodDelete},
		{"PATCH", http.MethodPatch},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, "/admin/user-limit", nil)
			req.Header.Set("Accept", "application/json")
			rec := httptest.NewRecorder()

			HandleUserLimit(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("期望 status=405, 实际=%d, body=%s", rec.Code, rec.Body.String())
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Errorf("期望 Content-Type 含 application/json, 实际=%q", ct)
			}
			var resp map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("响应应为 JSON: %v, body=%s", err, rec.Body.String())
			}
			if !strings.Contains(resp["error"].(string), "GET") {
				t.Errorf("error 信息应提示仅支持 GET, 实际=%q", resp["error"])
			}
		})
	}
}

func TestDisableToken_ErrPath(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	req := httptest.NewRequest(http.MethodGet, "/admin/token/disable?id=1", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleDisableToken(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestBatchCreateUser_ErrPath(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	req := httptest.NewRequest(http.MethodGet, "/admin/users/batch-create", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleBatchCreateUser(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestAdminUserToken_RealHandler_405(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	req := httptest.NewRequest(http.MethodPost, "/admin/user-token?id=1", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminUserToken(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestExportTokens_MethodCheck(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	req := httptest.NewRequest(http.MethodGet, "/admin/users/tokens", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleExportTokens(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ─── batchCreateItemDecodeErrorMessage 单元测试 ────────────────────────────

func TestBatchCreateItemDecodeErrorMessage_UsernameField(t *testing.T) {
	// line 489
	err := &json.UnmarshalTypeError{Field: "username", Value: "number", Type: reflect.TypeOf("")}
	msg := batchCreateItemDecodeErrorMessage(context.Background(), err)
	if !strings.Contains(msg, "用户名必须为字符串") {
		t.Errorf("期望包含用户名字段错误提示，实际=%q", msg)
	}
}

func TestBatchCreateItemDecodeErrorMessage_PasswordField(t *testing.T) {
	// line 491
	err := &json.UnmarshalTypeError{Field: "password", Value: "number", Type: reflect.TypeOf("")}
	msg := batchCreateItemDecodeErrorMessage(context.Background(), err)
	if !strings.Contains(msg, "密码必须为字符串") {
		t.Errorf("期望包含密码字段错误提示，实际=%q", msg)
	}
}

func TestBatchCreateItemDecodeErrorMessage_RoleField(t *testing.T) {
	// line 493
	err := &json.UnmarshalTypeError{Field: "role", Value: "number", Type: reflect.TypeOf("")}
	msg := batchCreateItemDecodeErrorMessage(context.Background(), err)
	if !strings.Contains(msg, "角色必须为字符串") {
		t.Errorf("期望包含角色字段错误提示，实际=%q", msg)
	}
}

func TestBatchCreateItemDecodeErrorMessage_EmailField(t *testing.T) {
	// line 495
	err := &json.UnmarshalTypeError{Field: "email", Value: "number", Type: reflect.TypeOf("")}
	msg := batchCreateItemDecodeErrorMessage(context.Background(), err)
	if !strings.Contains(msg, "邮箱必须为字符串") {
		t.Errorf("期望包含邮箱字段错误提示，实际=%q", msg)
	}
}

func TestBatchCreateItemDecodeErrorMessage_InstanceQuotaField(t *testing.T) {
	// line 497
	err := &json.UnmarshalTypeError{Field: "instance_quota", Value: "string", Type: reflect.TypeOf(0)}
	msg := batchCreateItemDecodeErrorMessage(context.Background(), err)
	if !strings.Contains(msg, "实例配额") {
		t.Errorf("期望包含实例配额字段错误提示，实际=%q", msg)
	}
}

func TestBatchCreateItemDecodeErrorMessage_TokenQuotaDayField(t *testing.T) {
	// line 499
	err := &json.UnmarshalTypeError{Field: "token_quota_day", Value: "string", Type: reflect.TypeOf(0)}
	msg := batchCreateItemDecodeErrorMessage(context.Background(), err)
	if !strings.Contains(msg, "Token") {
		t.Errorf("期望包含 Token 配额字段错误提示，实际=%q", msg)
	}
}

func TestBatchCreateItemDecodeErrorMessage_GroupIDsField(t *testing.T) {
	// line 501
	err := &json.UnmarshalTypeError{Field: "group_ids", Value: "number", Type: reflect.TypeOf("")}
	msg := batchCreateItemDecodeErrorMessage(context.Background(), err)
	if !strings.Contains(msg, "group_ids") {
		t.Errorf("期望包含 group_ids 字段错误提示，实际=%q", msg)
	}
}

func TestBatchCreateItemDecodeErrorMessage_UnknownTypeErrField(t *testing.T) {
	// line 503: default branch of UnmarshalTypeError switch
	err := &json.UnmarshalTypeError{Field: "some_other_field", Value: "string", Type: reflect.TypeOf(0)}
	msg := batchCreateItemDecodeErrorMessage(context.Background(), err)
	if !strings.Contains(msg, "用户字段格式错误") {
		t.Errorf("期望包含通用字段格式错误提示，实际=%q", msg)
	}
}

func TestBatchCreateItemDecodeErrorMessage_GroupIDsInNonTypeErr(t *testing.T) {
	// line 506-508: non-UnmarshalTypeError but contains "group_ids"
	msg := batchCreateItemDecodeErrorMessage(context.Background(), fmt.Errorf("group_ids format wrong"))
	if !strings.Contains(msg, "group_ids") {
		t.Errorf("期望包含 group_ids 格式错误提示，实际=%q", msg)
	}
}

func TestBatchCreateItemDecodeErrorMessage_GenericNonTypeErr(t *testing.T) {
	// line 509: non-UnmarshalTypeError without "group_ids"
	msg := batchCreateItemDecodeErrorMessage(context.Background(), fmt.Errorf("some other error"))
	if !strings.Contains(msg, "用户字段格式错误") {
		t.Errorf("期望包含通用字段格式错误提示，实际=%q", msg)
	}
}

// ─── createUserPrepared 单元测试 ────────────────────────────────────────────

func TestCreateUserPrepared_InstanceQuotaOutOfRange(t *testing.T) {
	// line 627: instance_quota < -1 or > 999
	initAdminTestDB(t)
	ctx := context.Background()

	q := -2
	_, status, err := createUserPrepared(context.Background(), createUserParams{
		Username:      "quotauser",
		Password:      "password123",
		InstanceQuota: &q,
	})
	if err == nil {
		t.Fatal("期望错误，实际成功")
	}
	if status != http.StatusBadRequest {
		t.Errorf("期望 status=400，实际=%d", status)
	}

	wanted := hcommon.I18nError(i18n.MsgInstanceQuotaDetailed)
	if !errors.Is(err, wanted) {
		t.Errorf("期望=%s，实际=%s", wanted.ErrorMessage(ctx), hcommon.ErrorMessageWithCtx(ctx, err))
	}

	q2 := 1000
	_, status, err = createUserPrepared(context.Background(), createUserParams{
		Username:      "quotauser2",
		Password:      "password123",
		InstanceQuota: &q2,
	})
	if err == nil {
		t.Fatal("期望错误，实际成功")
	}
	if status != http.StatusBadRequest {
		t.Errorf("期望 status=400，实际=%d", status)
	}
}

func TestCreateUserPrepared_TokenQuotaDayInvalid(t *testing.T) {
	// line 643: token_quota_day < -1
	initAdminTestDB(t)

	ctx := context.Background()

	q := -2
	_, status, err := createUserPrepared(ctx, createUserParams{
		Username:      "tquser",
		Password:      "password123",
		TokenQuotaDay: &q,
	})
	if err == nil {
		t.Fatal("期望错误，实际成功")
	}
	if status != http.StatusBadRequest {
		t.Errorf("期望 status=400，实际=%d", status)
	}

	wanted := hcommon.I18nError(i18n.MsgTokenQuotaMustBeValid)
	if !errors.Is(err, wanted) {
		t.Errorf("期望=%s，实际=%s", wanted.ErrorMessage(ctx), hcommon.ErrorMessageWithCtx(ctx, err))
	}
}

func TestCreateUserPrepared_ValidInstanceQuota(t *testing.T) {
	// 合法边界值: -1, 0, 999
	initAdminTestDB(t)
	for _, q := range []int{-1, 0, 999} {
		_, _, err := createUserPrepared(context.Background(), createUserParams{
			Username:      fmt.Sprintf("validq%d", q),
			Password:      "password123",
			InstanceQuota: &q,
		})
		if err != nil {
			t.Errorf("quota=%d 期望成功，实际错误: %v", q, err)
		}
	}
}

func TestCreateUserPrepared_ValidTokenQuotaDay(t *testing.T) {
	// 合法边界值: -1, 0
	initAdminTestDB(t)
	for _, q := range []int{-1, 0} {
		_, _, err := createUserPrepared(context.Background(), createUserParams{
			Username:      fmt.Sprintf("validtq%d", q),
			Password:      "password123",
			TokenQuotaDay: &q,
		})
		if err != nil {
			t.Errorf("token_quota_day=%d 期望成功，实际错误: %v", q, err)
		}
	}
}

// ─── HandleCreateUser handler 测试 ─────────────────────────────────────────

func newAdminUsersRequest(method, path string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Accept", "application/json")
	return req
}

func TestHandleCreateUser_InvalidInstanceQuotaString(t *testing.T) {
	// line 756: instance_quota 不是整数
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("username", "badquota")
	form.Set("password", "password123")
	form.Set("instance_quota", "not_a_number")

	req := newAdminUsersRequest(http.MethodPost, "/admin/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleCreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Fatal("期望包含 error 字段")
	}
}

func TestHandleCreateUser_InvalidTokenQuotaDayString(t *testing.T) {
	// line 764: token_quota_day 不是整数
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	form.Set("username", "badtokenquota")
	form.Set("password", "password123")
	form.Set("token_quota_day", "abc")

	req := newAdminUsersRequest(http.MethodPost, "/admin/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleCreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleDeleteUser handler 测试 ─────────────────────────────────────────

func TestHandleDeleteUser_UserNotFound(t *testing.T) {
	// line 911
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newAdminUsersRequest(http.MethodPost, "/admin/delete?id=99999", nil)
	w := httptest.NewRecorder()
	HandleDeleteUser(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDeleteUser_CannotDeleteInitialAdmin(t *testing.T) {
	// line 915
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 创建初始管理员（ID 最小的 admin）
	admin := model.User{Username: "firstadmin", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/delete?id=%d", admin.ID), nil)
	w := httptest.NewRecorder()
	HandleDeleteUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDeleteUser_Success(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 先创建初始管理员
	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	// 再创建普通用户
	user := model.User{Username: "deleteuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/delete?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleDeleteUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleHardDeleteUser handler 测试 ─────────────────────────────────────

func TestHandleHardDeleteUser_UserNotFound(t *testing.T) {
	// line 985
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newAdminUsersRequest(http.MethodPost, "/admin/hard-delete?id=99999", nil)
	w := httptest.NewRecorder()
	HandleHardDeleteUser(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleHardDeleteUser_CannotDeleteInitialAdmin(t *testing.T) {
	// line 989
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	admin := model.User{Username: "admin", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/hard-delete?id=%d", admin.ID), nil)
	w := httptest.NewRecorder()
	HandleHardDeleteUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleHardDeleteUser_UserHasInstances(t *testing.T) {
	// line 997
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "hasinstuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{UserID: user.ID, Name: "test-inst"})

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/hard-delete?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleHardDeleteUser(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("期望 409，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleRestoreUser handler 测试 ───────────────────────────────────────

func TestHandleRestoreUser_UserNotFound(t *testing.T) {
	// line 1052
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newAdminUsersRequest(http.MethodPost, "/admin/restore?id=99999", nil)
	w := httptest.NewRecorder()
	HandleRestoreUser(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleRestoreUser_CannotOperateInitialAdmin(t *testing.T) {
	// line 1056: HandleRestoreUser 使用 Unscoped 查询用户，
	// 然后调用 IsInitialAdmin 检查。对活跃的初始管理员，IsInitialAdmin 返回 true，
	// 因此即使对非软删除的初始管理员调用 restore 也会被 403 拦截。
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	admin := model.User{Username: "admin", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/restore?id=%d", admin.ID), nil)
	w := httptest.NewRecorder()
	HandleRestoreUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleResetPassword handler 测试 (真实 handler) ─────────────────────

func TestHandleResetPassword_EmptyPassword_RealHandler(t *testing.T) {
	// line 1117
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	form := url.Values{}
	req := newAdminUsersRequest(http.MethodPost, "/admin/reset-password?id=1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	HandleResetPassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleResetPassword_InitialAdminPasswordReset_NonAdminToken(t *testing.T) {
	// line 1132: 重置初始管理员密码时不用 admin-token
	// 使用 requireAdmin 鉴权（需要 AdminToken 或 session cookie），
	// 测试中 session store 未初始化，只能通过 admin-token 鉴权。
	// 要覆盖 1132 行，需要：requireAdmin 通过 + 命中初始管理员 + 非 admin-token 用户。
	// 但 requireAdmin 通过 admin-token 认证时，tokenUser.ID == 0 → 1132 不触发。
	// 要触发 1132，需要一个 session cookie 认证的非初始管理员来重置初始管理员密码，
	// 这需要初始化 session store，在单测中较复杂。
	// 因此通过 resetPasswordCoreHandler（绕过 requireAdmin）测试，已在现有测试中覆盖。
	// 此处验证 HandleResetPassword 对空密码的拦截（覆盖 1117 行）。
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	admin := model.User{Username: "admin", Password: "oldpass", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	form := url.Values{}
	form.Set("password", "")
	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/reset-password?id=%d", admin.ID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	HandleResetPassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateUser handler 测试 ─────────────────────────────────────────

func TestHandleUpdateUser_UserNotFound(t *testing.T) {
	// line 1171
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := `{"role":"user"}`
	req := newAdminUsersRequest(http.MethodPost, "/admin/update?id=99999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateUser_CannotModifyInitialAdminRole(t *testing.T) {
	// line 1192
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	admin := model.User{Username: "admin", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	body := `{"role":"user"}`
	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/update?id=%d", admin.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateUser_InstanceQuotaOutOfRange(t *testing.T) {
	// line 1200
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "updateuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	body := `{"instance_quota":-2}`
	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/update?id=%d", user.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateUser_TokenQuotaDayInvalid(t *testing.T) {
	// line 1217
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "tquser2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	body := `{"token_quota_day":-2}`
	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/update?id=%d", user.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateUser_NoFieldsToUpdate(t *testing.T) {
	// line 1233
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "nofielduser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	body := `{}`
	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/update?id=%d", user.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateUser_Success(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "okuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	body := `{"role":"admin"}`
	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/update?id=%d", user.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── handleToggleTokenState handler 测试 ───────────────────────────────────

func TestHandleToggleTokenState_UserNotFound(t *testing.T) {
	// line 1361
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newAdminUsersRequest(http.MethodPost, "/admin/token/disable?id=99999", nil)
	w := httptest.NewRecorder()
	HandleDisableToken(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleToggleTokenState_NoAPIToken(t *testing.T) {
	// line 1365
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	user := model.User{Username: "notokenuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/token/disable?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleDisableToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleToggleTokenState_AlreadyDisabled(t *testing.T) {
	// line 1370
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	user := model.User{Username: "disableduser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	_, _ = model.GenerateAPIToken(context.Background(), user.ID)
	_ = model.SetAPITokenDisabled(context.Background(), user.ID, true)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/token/disable?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleDisableToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（已禁用），实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Fatal("期望包含 error 字段")
	}
}

func TestHandleToggleTokenState_NotDisabled(t *testing.T) {
	// line 1372: 启用一个未禁用的 Token
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	user := model.User{Username: "enableduser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	_, _ = model.GenerateAPIToken(context.Background(), user.ID)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/token/enable?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleEnableToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（未禁用），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleToggleTokenState_DisableSuccess(t *testing.T) {
	// line 1377-1379: 正常禁用路径
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	user := model.User{Username: "todisable", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	_, _ = model.GenerateAPIToken(context.Background(), user.ID)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/token/disable?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleDisableToken(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleToggleTokenState_EnableSuccess(t *testing.T) {
	// line 1377: 正常启用路径
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	user := model.User{Username: "toenable", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	_, _ = model.GenerateAPIToken(context.Background(), user.ID)
	_ = model.SetAPITokenDisabled(context.Background(), user.ID, true)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/token/enable?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleEnableToken(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleBatchCreateUser handler 测试 ─────────────────────────────────────

func TestHandleBatchCreateUser_InvalidJSONBody(t *testing.T) {
	// line 1430: 请求体 JSON 格式错误
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newAdminUsersRequest(http.MethodPost, "/admin/users/batch-create", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleBatchCreateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（单项错误报告），实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Results []struct {
			Username  string `json:"username"`
			OK        bool   `json:"ok"`
			Error     string `json:"error,omitempty"`
			ErrorCode string `json:"error_code,omitempty"`
		} `json:"results"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].OK {
		t.Fatalf("期望单项错误报告: %+v", resp.Results)
	}
	if resp.Results[0].ErrorCode != "invalid_request_body" {
		t.Errorf("期望 error_code=invalid_request_body，实际=%q", resp.Results[0].ErrorCode)
	}
}

func TestHandleBatchCreateUser_EmptyList(t *testing.T) {
	// line 1443: 空列表
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newAdminUsersRequest(http.MethodPost, "/admin/users/batch-create", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleBatchCreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleAdminUserToken 真实 handler 测试 ───────────────────────────────

func TestHandleAdminUserToken_RealHandler_UserNotFound(t *testing.T) {
	// line 1625
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newAdminUsersRequest(http.MethodGet, "/admin/user-token?id=99999", nil)
	w := httptest.NewRecorder()
	HandleAdminUserToken(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUserVPC 真实 handler 测试 ──────────────────────────────────────

func TestHandleUserVPC_UserNotFound(t *testing.T) {
	// line 1823
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newAdminUsersRequest(http.MethodGet, "/admin/user-vpc?id=99999", nil)
	w := httptest.NewRecorder()
	HandleUserVPC(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── buildDepartmentPath 单元测试 ──────────────────────────────────────────

func TestBuildDepartmentPath_EmptyMap(t *testing.T) {
	result := buildDepartmentPath(nil, "1")
	if result != "" {
		t.Errorf("期望空路径，实际=%q", result)
	}
	result = buildDepartmentPath(map[string]model.OneIDDepartment{}, "1")
	if result != "" {
		t.Errorf("期望空路径，实际=%q", result)
	}
}

func TestBuildDepartmentPath_EmptyTargetID(t *testing.T) {
	deptMap := map[string]model.OneIDDepartment{
		"1": {DepartmentID: "1", DepartmentName: "技术部"},
	}
	result := buildDepartmentPath(deptMap, "")
	if result != "" {
		t.Errorf("期望空路径，实际=%q", result)
	}
}

func TestBuildDepartmentPath_SingleDept(t *testing.T) {
	deptMap := map[string]model.OneIDDepartment{
		"1": {DepartmentID: "1", DepartmentName: "技术部", DepartmentParentID: ""},
	}
	result := buildDepartmentPath(deptMap, "1")
	if result != "技术部" {
		t.Errorf("期望 '技术部'，实际=%q", result)
	}
}

func TestBuildDepartmentPath_NestedPath(t *testing.T) {
	deptMap := map[string]model.OneIDDepartment{
		"1": {DepartmentID: "1", DepartmentName: "公司", DepartmentParentID: ""},
		"2": {DepartmentID: "2", DepartmentName: "技术部", DepartmentParentID: "1"},
		"3": {DepartmentID: "3", DepartmentName: "后端组", DepartmentParentID: "2"},
	}
	result := buildDepartmentPath(deptMap, "3")
	if result != "公司/技术部/后端组" {
		t.Errorf("期望 '公司/技术部/后端组'，实际=%q", result)
	}
}

func TestBuildDepartmentPath_CircularRef(t *testing.T) {
	deptMap := map[string]model.OneIDDepartment{
		"1": {DepartmentID: "1", DepartmentName: "A", DepartmentParentID: "2"},
		"2": {DepartmentID: "2", DepartmentName: "B", DepartmentParentID: "1"},
	}
	result := buildDepartmentPath(deptMap, "1")
	if result != "B/A" {
		t.Errorf("期望 'B/A'（循环引用保护），实际=%q", result)
	}
}

func TestBuildDepartmentPath_MissingParent(t *testing.T) {
	deptMap := map[string]model.OneIDDepartment{
		"1": {DepartmentID: "1", DepartmentName: "后端组", DepartmentParentID: "99"},
	}
	result := buildDepartmentPath(deptMap, "1")
	if result != "后端组" {
		t.Errorf("期望 '后端组'（父部门不存在时停止），实际=%q", result)
	}
}

// ─── parseFlexUintSlice 单元测试 ──────────────────────────────────────────

func TestParseFlexUintSlice_IntegerArray(t *testing.T) {
	valid, invalid, err := parseFlexUintSlice([]byte(`[1,2,3]`))
	if err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	if len(valid) != 3 || valid[0] != 1 || valid[1] != 2 || valid[2] != 3 {
		t.Errorf("期望 [1,2,3]，实际=%v", valid)
	}
	if len(invalid) != 0 {
		t.Errorf("期望无无效项，实际=%v", invalid)
	}
}

func TestParseFlexUintSlice_StringArray(t *testing.T) {
	valid, _, err := parseFlexUintSlice([]byte(`["1","2","3"]`))
	if err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	if len(valid) != 3 || valid[0] != 1 || valid[1] != 2 || valid[2] != 3 {
		t.Errorf("期望 [1,2,3]，实际=%v", valid)
	}
}

func TestParseFlexUintSlice_MixedArray(t *testing.T) {
	valid, invalid, err := parseFlexUintSlice([]byte(`[1,"2","abc",{}]`))
	if err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	if len(valid) != 2 || valid[0] != 1 || valid[1] != 2 {
		t.Errorf("期望 [1,2]，实际=%v", valid)
	}
	if len(invalid) != 2 {
		t.Errorf("期望 2 个无效项（abc 和 {}），实际=%d: %v", len(invalid), invalid)
	}
}

func TestParseFlexUintSlice_NotArray(t *testing.T) {
	_, _, err := parseFlexUintSlice([]byte(`"not an array"`))
	if err == nil {
		t.Fatal("期望错误，实际成功")
	}
}

// ─── flexGroupIDs 单元测试 ─────────────────────────────────────────────────

func TestFlexGroupIDs_UnmarshalJSON(t *testing.T) {
	var f flexGroupIDs
	if err := json.Unmarshal([]byte(`"根组/研发一组;根组/研发二组"`), &f); err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	if len(f.Names) != 2 || f.Names[0] != "根组/研发一组" || f.Names[1] != "根组/研发二组" {
		t.Errorf("期望两个组名，实际=%v", f.Names)
	}
}

func TestFlexGroupIDs_UnmarshalJSON_SinglePath(t *testing.T) {
	var f flexGroupIDs
	if err := json.Unmarshal([]byte(`"研发一组"`), &f); err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	if len(f.Names) != 1 || f.Names[0] != "研发一组" {
		t.Errorf("期望一个组名，实际=%v", f.Names)
	}
}

func TestFlexGroupIDs_UnmarshalJSON_EmptyParts(t *testing.T) {
	var f flexGroupIDs
	if err := json.Unmarshal([]byte(`"研发一组;;研发二组;"`), &f); err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	if len(f.Names) != 2 || f.Names[0] != "研发一组" || f.Names[1] != "研发二组" {
		t.Errorf("期望跳过空段，实际=%v", f.Names)
	}
}

func TestFlexGroupIDs_UnmarshalJSON_NotString(t *testing.T) {
	var f flexGroupIDs
	err := json.Unmarshal([]byte(`[1,2,3]`), &f)
	if err == nil {
		t.Fatal("期望错误（非字符串），实际成功")
	}
}

// ─── effectiveAdminUserTokenQuotaRules 单元测试 ─────────────────────────────

func TestEffectiveAdminUserTokenQuotaRules_ParseOK(t *testing.T) {
	user := model.User{TokenQuotaRules: `[{"mode":"day","limit":100}]`}
	result := effectiveAdminUserTokenQuotaRulesJSON(user)
	if !strings.Contains(result, "day") || !strings.Contains(result, "100") {
		t.Errorf("期望包含 day/limit，实际=%q", result)
	}
}

func TestEffectiveAdminUserTokenQuotaRules_InvalidRules(t *testing.T) {
	user := model.User{TokenQuotaRules: "invalid-json"}
	result := effectiveAdminUserTokenQuotaRulesJSON(user)
	// 无效 JSON 时 ParseTokenQuotaRules 返回 false，走 ResolvedTokenQuotaRules 回退
	// ResolvedTokenQuotaRules 返回基于 TokenQuotaDay 的默认规则
	if result == "" {
		t.Error("期望非空结果")
	}
}

// ─── resetPasswordCore 单元测试 (line 1100) ──────────────────────────────

func TestResetPasswordCore_Success(t *testing.T) {
	initAdminTestDB(t)
	user := model.User{Username: "resetpw", Password: "oldhash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	err := resetPasswordCore(context.Background(), &user, "newpass123")
	if err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	// 验证密码被更新
	var updated model.User
	model.DB(context.Background()).First(&updated, user.ID)
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("newpass123")); err != nil {
		t.Errorf("密码未被正确更新: %v", err)
	}
}

// ─── resolveResetPasswordUser 单元测试 ─────────────────────────────────────

func TestResolveResetPasswordUser_InitUser(t *testing.T) {
	initAdminTestDB(t)
	admin := model.User{Username: "admin", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	user, err := resolveResetPasswordUser(context.Background(), "0", true)
	if err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	if user.ID != admin.ID {
		t.Errorf("期望返回初始管理员 ID=%d，实际=%d", admin.ID, user.ID)
	}
}

func TestResolveResetPasswordUser_InitUserNoAdmin(t *testing.T) {
	initAdminTestDB(t)
	_, err := resolveResetPasswordUser(context.Background(), "0", true)
	if err == nil {
		t.Fatal("期望错误（无初始管理员），实际成功")
	}
}

func TestResolveResetPasswordUser_ByIDNotFound(t *testing.T) {
	initAdminTestDB(t)
	_, err := resolveResetPasswordUser(context.Background(), "99999", false)
	if err == nil {
		t.Fatal("期望错误（用户不存在），实际成功")
	}
}

// ─── HandleDeleteUser 停机失败路径测试 (line 920, 936) ───────────────────

func TestHandleDeleteUser_StopInstancesFailed(t *testing.T) {
	// line 920: stopUserInstances 失败
	// 需要用户有实例但 CVM 客户端创建失败（无凭证），
	// 此时 stopUserInstances 会因为 NewCVMClient 失败而返回错误。
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "hasinstuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	// 创建一个带 CVM InstanceId 的实例，这会触发 stopUserInstances
	model.DB(context.Background()).Create(&model.Instance{UserID: user.ID, InstanceId: "ins-test123", Name: "running-inst"})

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/delete?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleDeleteUser(w, req)

	// stopUserInstances 会因为无法创建 CVM client 而失败
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleHardDeleteUser 无实例无 VPC 成功路径 (line 1037) ────────────────

func TestHandleHardDeleteUser_Success(t *testing.T) {
	// 覆盖硬删除成功路径（无实例、无 VPC）
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "harddeleteuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/hard-delete?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleHardDeleteUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleRestoreUser 成功路径 ────────────────────────────────────────────

func TestHandleRestoreUser_Success(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "restoreuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Delete(&user) // 软删除

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/restore?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleRestoreUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleExportTokens POST 成功路径 (line 1339) ──────────────────────────

func TestHandleExportTokens_Success(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 创建用户以确保有 token 可以导出
	model.DB(context.Background()).Create(&model.User{Username: "tokenuser", Password: "x", Role: "user"})

	req := newAdminUsersRequest(http.MethodPost, "/admin/users/tokens", nil)
	w := httptest.NewRecorder()
	HandleExportTokens(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── flexUintSlice UnmarshalJSON 测试 ─────────────────────────────────────

func TestFlexUintSlice_UnmarshalJSON(t *testing.T) {
	var f flexUintSlice
	if err := json.Unmarshal([]byte(`[1,"2",3]`), &f); err != nil {
		t.Fatalf("期望成功，实际错误: %v", err)
	}
	if len(f) != 3 || f[0] != 1 || f[1] != 2 || f[2] != 3 {
		t.Errorf("期望 [1,2,3]，实际=%v", f)
	}
}

// ─── HandleResetPassword 初始管理员路径 (line 1132) ────────────────────────

func TestHandleResetPassword_InitialAdminPasswordReset_WithAdminToken(t *testing.T) {
	// line 1132: 覆盖初始管理员密码重置路径
	// 当使用 admin-token 认证时，tokenUser.ID == 0，应通过初始管理员校验
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	admin := model.User{Username: "admin", Password: "oldpass", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	form := url.Values{}
	form.Set("password", "brand-new-pass")
	form.Set("init_user", "true")
	req := newAdminUsersRequest(http.MethodPost, "/admin/reset-password?init_user=true", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	HandleResetPassword(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUpdateUser token_quota_rules 路径测试 ───────────────────────────

func TestHandleUpdateUser_TokenQuotaRules(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "tqruser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	body := `{"token_quota_rules":[{"mode":"day","limit":100}]}`
	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/update?id=%d", user.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证 token_quota_rules 被更新
	var updated model.User
	model.DB(context.Background()).First(&updated, user.ID)
	if updated.TokenQuotaDay != -1 {
		t.Errorf("期望 token_quota_day=-1，实际=%d", updated.TokenQuotaDay)
	}
}

func TestHandleUpdateUser_TokenQuotaRulesInvalid(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "badtqruser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	body := `{"token_quota_rules":"invalid"}`
	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/update?id=%d", user.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateUser_TokenQuotaDayValid(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "tqdayuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	body := `{"token_quota_day":100}`
	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/update?id=%d", user.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleCreateUser JSON body 测试 ───────────────────────────────────────

func TestHandleCreateUser_WithJSONBody(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := `{"username":"jsonuser","password":"pass123","role":"user","instance_quota":5,"token_quota_day":100}`
	req := newAdminUsersRequest(http.MethodPost, "/admin/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleCreateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateUser_EmptyUsernameViaJSON(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := `{"username":"","password":"pass123"}`
	req := newAdminUsersRequest(http.MethodPost, "/admin/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleCreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateUser_InstanceQuotaOutOfRange(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := `{"username":"badquser","password":"pass123","instance_quota":-2}`
	req := newAdminUsersRequest(http.MethodPost, "/admin/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleCreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleUserVPC 用户无 VPC 路径 ────────────────────────────────────────

func TestHandleUserVPC_NoVPC(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "novpcuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := newAdminUsersRequest(http.MethodGet, fmt.Sprintf("/admin/user-vpc?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleUserVPC(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["vpc_id"] != nil {
		t.Errorf("期望 vpc_id=null，实际=%v", resp["vpc_id"])
	}
}

// ─── HandleBatchCreateUser 用户组路径不存在 (line 1512) ────────────────────

func TestHandleBatchCreateUser_GroupPathNotFound(t *testing.T) {
	// line 1512: 批量创建时指定不存在的用户组路径
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := `[{"username":"groupuser","password":"p1","group_ids":"不存在的组/子组"}]`
	req := newAdminUsersRequest(http.MethodPost, "/admin/users/batch-create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleBatchCreateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Results []struct {
			Username  string `json:"username"`
			OK        bool   `json:"ok"`
			Error     string `json:"error,omitempty"`
			ErrorCode string `json:"error_code,omitempty"`
		} `json:"results"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("期望 1 条结果，实际=%d", len(resp.Results))
	}
	if resp.Results[0].OK {
		t.Fatal("期望失败（组路径不存在）")
	}
	if !strings.Contains(resp.Results[0].Error, "不存在") {
		t.Errorf("期望包含「不存在」错误提示，实际=%q", resp.Results[0].Error)
	}
}

func TestHandleBatchCreateUser_GroupQueryFailed(t *testing.T) {
	// line 1494: 批量创建时查询用户组失败
	// 此场景需要 GetGroupsByFullPaths 返回错误，在 SQLite 单测中不太容易模拟。
	// 通过指定不存在的路径来覆盖部分逻辑。
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := `[{"username":"queryfailuser","password":"p1","group_ids":"组A"}]`
	req := newAdminUsersRequest(http.MethodPost, "/admin/users/batch-create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleBatchCreateUser(w, req)

	// 即使组不存在，也应返回 200（逐项错误报告）
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ─── HandleCreateUser token_quota_rules 测试 ───────────────────────────────

func TestHandleCreateUser_WithTokenQuotaRules(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := `{"username":"rulesuser","password":"pass123","token_quota_rules":[{"mode":"day","limit":500}]}`
	req := newAdminUsersRequest(http.MethodPost, "/admin/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleCreateUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}
}

// ─── HandleDeleteUser 成功路径（无实例） ───────────────────────────────────

func TestHandleDeleteUser_SuccessNoInstances(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	user := model.User{Username: "deletenoinst", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := newAdminUsersRequest(http.MethodPost, fmt.Sprintf("/admin/delete?id=%d", user.ID), nil)
	w := httptest.NewRecorder()
	HandleDeleteUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

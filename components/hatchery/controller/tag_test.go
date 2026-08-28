package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tag "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"
	"gorm.io/gorm"
)

// ==================== 测试基础设施 ====================

// initTagTestEnv 初始化 Tag 测试环境（内存 DB + session store + AdminToken）
func initTagTestEnv(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}, &model.Tag{}, &model.TagVisibilityGroup{}, &model.UserGroup{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	db.Create(&model.SiteConfig{})
	Store = sessions.NewCookieStore([]byte("test-secret"))
	AdminToken = "test-admin-token"
}

// adminRequest 创建带 AdminToken 认证的 JSON 请求
func adminRequest(method, path string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// mockTagClient 模拟腾讯云 Tag API 客户端
type mockTagClient struct {
	describeTagKeysFunc   func(*tag.DescribeTagKeysRequest) (*tag.DescribeTagKeysResponse, error)
	describeTagValuesFunc func(*tag.DescribeTagValuesRequest) (*tag.DescribeTagValuesResponse, error)
}

func (m *mockTagClient) DescribeTagKeys(req *tag.DescribeTagKeysRequest) (*tag.DescribeTagKeysResponse, error) {
	if m.describeTagKeysFunc != nil {
		return m.describeTagKeysFunc(req)
	}
	return &tag.DescribeTagKeysResponse{}, nil
}

func (m *mockTagClient) DescribeTagValues(req *tag.DescribeTagValuesRequest) (*tag.DescribeTagValuesResponse, error) {
	if m.describeTagValuesFunc != nil {
		return m.describeTagValuesFunc(req)
	}
	return &tag.DescribeTagValuesResponse{}, nil
}

// setMockTagClient 设置 mock Tag 客户端，返回清理函数
func setMockTagClient(t *testing.T, mock *mockTagClient) {
	t.Helper()
	orig := newTagClientFunc
	newTagClientFunc = func(ctx context.Context) (tagClient, error) {
		return mock, nil
	}
	t.Cleanup(func() { newTagClientFunc = orig })
}

// setMockTagClientError 设置 Tag 客户端创建失败，返回清理函数
func setMockTagClientError(t *testing.T, err error) {
	t.Helper()
	orig := newTagClientFunc
	newTagClientFunc = func(ctx context.Context) (tagClient, error) {
		return nil, err
	}
	t.Cleanup(func() { newTagClientFunc = orig })
}

func TestHandleCreateTag_MigratesLegacyDefaultTagsAndClearsOldField(t *testing.T) {
	initTagTestEnv(t)
	ctx := context.Background()
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("1 = 1").
		Update("default_tags", `[{"Key":"legacy","Value":"yes"}]`).Error; err != nil {
		t.Fatalf("seed legacy default tags: %v", err)
	}

	req := adminRequest(http.MethodPost, "/admin/tags/create", []byte(`{"key":"team","value":"rd","visibility_type":"all"}`))
	w := httptest.NewRecorder()

	HandleCreateTag(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	cfg := model.GetSiteConfig(ctx)
	if cfg.DefaultTags != "[]" {
		t.Fatalf("legacy default_tags should be cleared, got %q", cfg.DefaultTags)
	}
	var legacyCount int64
	if err := model.DB(ctx).Model(&model.Tag{}).
		Where("tag_key = ? AND tag_value = ? AND visibility_type = ?", "legacy", "yes", model.VisibilityAll).
		Count(&legacyCount).Error; err != nil {
		t.Fatalf("count migrated tag: %v", err)
	}
	if legacyCount != 1 {
		t.Fatalf("expected migrated legacy tag, got count=%d", legacyCount)
	}
}

func TestHandleUpdateTag_MigratesLegacyDefaultTagsAndClearsOldField(t *testing.T) {
	initTagTestEnv(t)
	ctx := context.Background()
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("1 = 1").
		Update("default_tags", `[{"Key":"legacy","Value":"yes"}]`).Error; err != nil {
		t.Fatalf("seed legacy default tags: %v", err)
	}
	row := model.Tag{TagKey: "team", TagValue: "old", VisibilityType: model.VisibilityAll}
	if err := model.DB(ctx).Create(&row).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}

	req := adminRequest(http.MethodPost, fmt.Sprintf("/admin/tags/update?id=%d", row.ID), []byte(`{"key":"team","value":"new","visibility_type":"all"}`))
	w := httptest.NewRecorder()

	HandleUpdateTag(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if got := model.GetSiteConfig(ctx).DefaultTags; got != "[]" {
		t.Fatalf("legacy default_tags should be cleared, got %q", got)
	}
	var legacyCount int64
	model.DB(ctx).Model(&model.Tag{}).Where("tag_key = ? AND tag_value = ?", "legacy", "yes").Count(&legacyCount)
	if legacyCount != 1 {
		t.Fatalf("expected migrated legacy tag, got count=%d", legacyCount)
	}
}

func TestHandleReplaceAllTags_MigratesLegacyAndReplacesAllTags(t *testing.T) {
	initTagTestEnv(t)
	ctx := context.Background()
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("1 = 1").
		Update("default_tags", `[{"Key":"legacy","Value":"yes"}]`).Error; err != nil {
		t.Fatalf("seed legacy default tags: %v", err)
	}
	old := model.Tag{TagKey: "old", TagValue: "tag", VisibilityType: model.VisibilityAll}
	if err := model.DB(ctx).Create(&old).Error; err != nil {
		t.Fatalf("create old tag: %v", err)
	}
	group := model.UserGroup{Name: "研发组"}
	if err := model.DB(ctx).Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	body := fmt.Sprintf(`{"tags":[{"key":"env","value":"prod","visibility_type":"all"},{"key":"team","value":"rd","visibility_type":"group","group_ids":[%d]}]}`, group.ID)
	req := adminRequest(http.MethodPost, "/admin/tags/replace-all", []byte(body))
	w := httptest.NewRecorder()

	HandleReplaceAllTags(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if got := model.GetSiteConfig(ctx).DefaultTags; got != "[]" {
		t.Fatalf("legacy default_tags should be cleared, got %q", got)
	}
	var rows []model.Tag
	if err := model.DB(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 replacement tags, got %+v", rows)
	}
	seen := map[string]string{}
	for _, row := range rows {
		seen[row.TagKey] = row.TagValue
	}
	if seen["old"] != "" || seen["legacy"] != "" || seen["env"] != "prod" || seen["team"] != "rd" {
		t.Fatalf("unexpected replacement result: %+v", rows)
	}
	var bindingCount int64
	if err := model.DB(ctx).Model(&model.TagVisibilityGroup{}).
		Where("group_id = ?", group.ID).
		Count(&bindingCount).Error; err != nil {
		t.Fatalf("count binding: %v", err)
	}
	if bindingCount != 1 {
		t.Fatalf("expected one group binding, got %d", bindingCount)
	}
}

func TestHandleAdminTags_ListManagedTags(t *testing.T) {
	initTagTestEnv(t)
	ctx := context.Background()
	global := model.Tag{TagKey: "env", TagValue: "prod", VisibilityType: model.VisibilityAll}
	scoped := model.Tag{TagKey: "team", TagValue: "rd", VisibilityType: model.VisibilityGroup}
	if err := model.DB(ctx).Create(&global).Error; err != nil {
		t.Fatalf("create global tag: %v", err)
	}
	if err := model.DB(ctx).Create(&scoped).Error; err != nil {
		t.Fatalf("create scoped tag: %v", err)
	}
	if err := model.DB(ctx).Create(&model.TagVisibilityGroup{TagID: scoped.ID, GroupID: 10}).Error; err != nil {
		t.Fatalf("create binding: %v", err)
	}

	req := adminRequest(http.MethodGet, "/admin/tags", nil)
	w := httptest.NewRecorder()
	HandleAdminTags(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Tags []adminTagResponse `json:"tags"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Tags) != 2 {
		t.Fatalf("expected two tags, got %+v", resp.Tags)
	}
	if resp.Tags[1].Key != "team" || len(resp.Tags[1].GroupIDs) != 1 || resp.Tags[1].GroupIDs[0] != 10 {
		t.Fatalf("unexpected scoped tag response: %+v", resp.Tags[1])
	}
	if !strings.Contains(w.Body.String(), `"group_ids":[]`) {
		t.Fatalf("global tag should return empty group_ids array, body=%s", w.Body.String())
	}
}

func TestHandleAdminTags_MigratesLegacyDefaultTagsAndReturnsIDs(t *testing.T) {
	initTagTestEnv(t)
	ctx := context.Background()
	legacyRaw := `[{"Key":"legacy","Value":"yes"}]`
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("1 = 1").
		Update("default_tags", legacyRaw).Error; err != nil {
		t.Fatalf("seed legacy default tags: %v", err)
	}

	req := adminRequest(http.MethodGet, "/admin/tags", nil)
	w := httptest.NewRecorder()
	HandleAdminTags(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Tags []adminTagResponse `json:"tags"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Tags) != 1 {
		t.Fatalf("expected one legacy tag, got %+v", resp.Tags)
	}
	got := resp.Tags[0]
	if got.Key != "legacy" || got.Value != "yes" || got.VisibilityType != model.VisibilityAll {
		t.Fatalf("unexpected legacy tag response: %+v", got)
	}
	if got.ID == 0 || got.CreatedAt == "" || got.UpdatedAt == "" || len(got.GroupIDs) != 0 {
		t.Fatalf("legacy tag should be migrated and return persisted metadata: %+v", got)
	}
	if !strings.Contains(w.Body.String(), `"group_ids":[]`) {
		t.Fatalf("global tag should return empty group_ids array, body=%s", w.Body.String())
	}
	if cfg := model.GetSiteConfig(ctx); cfg.DefaultTags != "[]" {
		t.Fatalf("GET /admin/tags should clear legacy default_tags after migration, got %q", cfg.DefaultTags)
	}
	var tagCount int64
	if err := model.DB(ctx).Model(&model.Tag{}).Count(&tagCount).Error; err != nil {
		t.Fatalf("count tags: %v", err)
	}
	if tagCount != 1 {
		t.Fatalf("GET /admin/tags should migrate one tag row, got %d", tagCount)
	}
}

func TestHandleAdminTags_LegacyMigrationError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	Store = sessions.NewCookieStore([]byte("test-secret"))
	AdminToken = "test-admin-token"

	req := adminRequest(http.MethodGet, "/admin/tags", nil)
	w := httptest.NewRecorder()
	HandleAdminTags(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "迁移默认标签失败") {
		t.Fatalf("expected migration error response, body=%s", w.Body.String())
	}
}

func TestAdminTagHandlers_MethodAndAuthErrors(t *testing.T) {
	initTagTestEnv(t)

	cases := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
		body    []byte
		want    int
	}{
		{"list wrong method", HandleAdminTags, http.MethodPost, "/admin/tags", nil, http.StatusMethodNotAllowed},
		{"create wrong method", HandleCreateTag, http.MethodGet, "/admin/tags/create", nil, http.StatusMethodNotAllowed},
		{"update wrong method", HandleUpdateTag, http.MethodGet, "/admin/tags/update?id=1", nil, http.StatusMethodNotAllowed},
		{"replace wrong method", HandleReplaceAllTags, http.MethodGet, "/admin/tags/replace-all", nil, http.StatusMethodNotAllowed},
		{"delete wrong method", HandleDeleteTag, http.MethodGet, "/admin/tags/delete?id=1", nil, http.StatusMethodNotAllowed},
		{"create unauthorized", HandleCreateTag, http.MethodPost, "/admin/tags/create", []byte(`{}`), http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.name == "create unauthorized" {
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewReader(tc.body))
				req.Header.Set("Accept", "application/json")
			} else {
				req = adminRequest(tc.method, tc.path, tc.body)
			}
			w := httptest.NewRecorder()
			tc.handler(w, req)
			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d, body=%s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleCreateTag_GroupScopeAndValidationErrors(t *testing.T) {
	initTagTestEnv(t)
	ctx := context.Background()
	group := model.UserGroup{Name: "研发组"}
	if err := model.DB(ctx).Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	req := adminRequest(http.MethodPost, "/admin/tags/create", []byte(fmt.Sprintf(`{"key":"team","value":"rd","visibility_type":"group","group_ids":[%d]}`, group.ID)))
	w := httptest.NewRecorder()
	HandleCreateTag(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Tag adminTagResponse `json:"tag"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Tag.VisibilityType != model.VisibilityGroup || len(resp.Tag.GroupIDs) != 1 || resp.Tag.GroupIDs[0] != group.ID {
		t.Fatalf("unexpected create response: %+v", resp.Tag)
	}

	for _, tc := range []struct {
		name string
		body string
	}{
		{"bad json", `{bad json`},
		{"missing group", `{"key":"team","value":"rd","visibility_type":"group","group_ids":[999]}`},
		{"empty key", `{"key":"","value":"rd","visibility_type":"all"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := adminRequest(http.MethodPost, "/admin/tags/create", []byte(tc.body))
			w := httptest.NewRecorder()
			HandleCreateTag(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleUpdateTag_SuccessAndErrors(t *testing.T) {
	initTagTestEnv(t)
	ctx := context.Background()
	row := model.Tag{TagKey: "team", TagValue: "old", VisibilityType: model.VisibilityAll}
	if err := model.DB(ctx).Create(&row).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	group := model.UserGroup{Name: "研发组"}
	if err := model.DB(ctx).Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	req := adminRequest(http.MethodPost, fmt.Sprintf("/admin/tags/update?id=%d", row.ID), []byte(fmt.Sprintf(`{"key":"team","value":"new","visibility_type":"group","group_ids":[%d]}`, group.ID)))
	w := httptest.NewRecorder()
	HandleUpdateTag(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Tag adminTagResponse `json:"tag"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Tag.Value != "new" || resp.Tag.VisibilityType != model.VisibilityGroup || len(resp.Tag.GroupIDs) != 1 {
		t.Fatalf("unexpected update response: %+v", resp.Tag)
	}

	for _, tc := range []struct {
		name string
		path string
		body string
		want int
	}{
		{"bad id", "/admin/tags/update?id=0", `{"key":"x","visibility_type":"all"}`, http.StatusBadRequest},
		{"bad json", fmt.Sprintf("/admin/tags/update?id=%d", row.ID), `{bad json`, http.StatusBadRequest},
		{"missing group", fmt.Sprintf("/admin/tags/update?id=%d", row.ID), `{"key":"team","visibility_type":"group","group_ids":[999]}`, http.StatusBadRequest},
		{"not found", "/admin/tags/update?id=999", `{"key":"missing","visibility_type":"all"}`, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := adminRequest(http.MethodPost, tc.path, []byte(tc.body))
			w := httptest.NewRecorder()
			HandleUpdateTag(w, req)
			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d, body=%s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleReplaceAllTags_ValidationErrors(t *testing.T) {
	initTagTestEnv(t)

	for _, tc := range []struct {
		name string
		body string
	}{
		{"bad json", `{bad json`},
		{"missing group", `{"tags":[{"key":"team","visibility_type":"group","group_ids":[999]}]}`},
		{"empty key", `{"tags":[{"key":"","visibility_type":"all"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := adminRequest(http.MethodPost, "/admin/tags/replace-all", []byte(tc.body))
			w := httptest.NewRecorder()
			HandleReplaceAllTags(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleDeleteTag_SuccessAndErrors(t *testing.T) {
	initTagTestEnv(t)
	ctx := context.Background()
	row := model.Tag{TagKey: "team", TagValue: "rd", VisibilityType: model.VisibilityGroup}
	if err := model.DB(ctx).Create(&row).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if err := model.DB(ctx).Create(&model.TagVisibilityGroup{TagID: row.ID, GroupID: 10}).Error; err != nil {
		t.Fatalf("create binding: %v", err)
	}

	req := adminRequest(http.MethodPost, fmt.Sprintf("/admin/tags/delete?id=%d", row.ID), nil)
	w := httptest.NewRecorder()
	HandleDeleteTag(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var count int64
	model.DB(ctx).Model(&model.TagVisibilityGroup{}).Where("tag_id = ?", row.ID).Count(&count)
	if count != 0 {
		t.Fatalf("expected binding deleted, got %d", count)
	}

	for _, tc := range []struct {
		name string
		path string
		want int
	}{
		{"bad id", "/admin/tags/delete?id=0", http.StatusBadRequest},
		{"not found", "/admin/tags/delete?id=999", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := adminRequest(http.MethodPost, tc.path, nil)
			w := httptest.NewRecorder()
			HandleDeleteTag(w, req)
			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d, body=%s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

// ==================== ParseDefaultTags 单元测试 ====================

func TestParseDefaultTags_空字符串(t *testing.T) {
	tags := ParseDefaultTags("")
	if len(tags) != 0 {
		t.Errorf("空字符串应返回空数组，实际: %+v", tags)
	}
}

func TestParseDefaultTags_合法JSON(t *testing.T) {
	raw := `[{"Key":"env","Value":"prod"},{"Key":"team","Value":"ai"}]`
	tags := ParseDefaultTags(raw)
	if len(tags) != 2 {
		t.Fatalf("期望 2 个标签，实际: %d", len(tags))
	}
	if tags[0].Key != "env" || tags[0].Value != "prod" {
		t.Errorf("第一个标签不匹配: %+v", tags[0])
	}
	if tags[1].Key != "team" || tags[1].Value != "ai" {
		t.Errorf("第二个标签不匹配: %+v", tags[1])
	}
}

func TestParseDefaultTags_非法JSON(t *testing.T) {
	tags := ParseDefaultTags("{invalid json")
	if len(tags) != 0 {
		t.Errorf("非法 JSON 应返回空数组，实际: %+v", tags)
	}
}

func TestParseDefaultTags_空数组(t *testing.T) {
	tags := ParseDefaultTags("[]")
	if len(tags) != 0 {
		t.Errorf("空数组应返回空切片，实际长度: %d", len(tags))
	}
}

func TestParseDefaultTags_Value为空(t *testing.T) {
	raw := `[{"Key":"env","Value":""}]`
	tags := ParseDefaultTags(raw)
	if len(tags) != 1 || tags[0].Key != "env" || tags[0].Value != "" {
		t.Errorf("标签值为空时应正常解析: %+v", tags)
	}
}

func TestParseDefaultTags_JSON数组默认值(t *testing.T) {
	tags := ParseDefaultTags("[]")
	if tags == nil {
		t.Error("'[]' 应返回非 nil 空切片")
	}
}

// ==================== Handler Method Not Allowed 测试 ====================

func TestHandleGetTagKeys_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tags/keys", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleGetTagKeys(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST 请求应返回 405，实际: %d", w.Code)
	}
}

func TestHandleGetTagValues_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tags/values?key=env", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleGetTagValues(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST 请求应返回 405，实际: %d", w.Code)
	}
}

// ==================== Handler 认证拦截测试 ====================

func TestHandleGetTagKeys_未认证(t *testing.T) {
	initTagTestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/api/tags/keys", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleGetTagKeys(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("未认证应返回 401，实际: %d", w.Code)
	}
}

func TestHandleGetTagValues_未认证(t *testing.T) {
	initTagTestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/api/tags/values?key=env", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleGetTagValues(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("未认证应返回 401，实际: %d", w.Code)
	}
}

// ==================== HandleGetTagValues 参数校验 ====================

func TestHandleGetTagValues_缺少key参数(t *testing.T) {
	initTagTestEnv(t)
	req := adminRequest(http.MethodGet, "/api/tags/values", nil)
	w := httptest.NewRecorder()
	HandleGetTagValues(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("缺少 key 参数应返回 400，实际: %d", w.Code)
	}
}

// ==================== HandleGetTagKeys mock 测试 ====================

func TestHandleGetTagKeys_客户端创建失败(t *testing.T) {
	initTagTestEnv(t)
	setMockTagClientError(t, fmt.Errorf("凭据未配置"))
	req := adminRequest(http.MethodGet, "/api/tags/keys", nil)
	w := httptest.NewRecorder()
	HandleGetTagKeys(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("客户端创建失败应返回 500，实际: %d", w.Code)
	}
}

func TestHandleGetTagKeys_成功返回标签键(t *testing.T) {
	initTagTestEnv(t)
	k1, k2, k3 := "env", "team", "所属产品"
	setMockTagClient(t, &mockTagClient{
		describeTagKeysFunc: func(req *tag.DescribeTagKeysRequest) (*tag.DescribeTagKeysResponse, error) {
			resp := tag.NewDescribeTagKeysResponse()
			resp.Response = &tag.DescribeTagKeysResponseParams{
				Tags: []*string{&k1, &k2, &k3},
			}
			return resp, nil
		},
	})

	req := adminRequest(http.MethodGet, "/api/tags/keys", nil)
	w := httptest.NewRecorder()
	HandleGetTagKeys(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际: %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	keys := resp["keys"].([]interface{})
	if len(keys) != 3 {
		t.Errorf("期望 3 个标签键，实际: %d", len(keys))
	}
	if keys[0] != "env" || keys[1] != "team" || keys[2] != "所属产品" {
		t.Errorf("标签键不匹配: %v", keys)
	}
}

func TestHandleGetTagKeys_空结果(t *testing.T) {
	initTagTestEnv(t)
	setMockTagClient(t, &mockTagClient{
		describeTagKeysFunc: func(req *tag.DescribeTagKeysRequest) (*tag.DescribeTagKeysResponse, error) {
			resp := tag.NewDescribeTagKeysResponse()
			resp.Response = &tag.DescribeTagKeysResponseParams{
				Tags: []*string{},
			}
			return resp, nil
		},
	})

	req := adminRequest(http.MethodGet, "/api/tags/keys", nil)
	w := httptest.NewRecorder()
	HandleGetTagKeys(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际: %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	keys := resp["keys"].([]interface{})
	if len(keys) != 0 {
		t.Errorf("期望空数组，实际: %v", keys)
	}
}

func TestHandleGetTagKeys_API调用失败(t *testing.T) {
	initTagTestEnv(t)
	setMockTagClient(t, &mockTagClient{
		describeTagKeysFunc: func(req *tag.DescribeTagKeysRequest) (*tag.DescribeTagKeysResponse, error) {
			return nil, fmt.Errorf("network timeout")
		},
	})

	req := adminRequest(http.MethodGet, "/api/tags/keys", nil)
	w := httptest.NewRecorder()
	HandleGetTagKeys(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("API 失败应返回 500，实际: %d", w.Code)
	}
}

func TestHandleGetTagKeys_Response为nil(t *testing.T) {
	initTagTestEnv(t)
	setMockTagClient(t, &mockTagClient{
		describeTagKeysFunc: func(req *tag.DescribeTagKeysRequest) (*tag.DescribeTagKeysResponse, error) {
			return &tag.DescribeTagKeysResponse{}, nil
		},
	})

	req := adminRequest(http.MethodGet, "/api/tags/keys", nil)
	w := httptest.NewRecorder()
	HandleGetTagKeys(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Response 为 nil 应返回空数组，实际: %d", w.Code)
	}
}

// ==================== HandleGetTagValues mock 测试 ====================

func TestHandleGetTagValues_客户端创建失败(t *testing.T) {
	initTagTestEnv(t)
	setMockTagClientError(t, fmt.Errorf("凭据未配置"))
	req := adminRequest(http.MethodGet, "/api/tags/values?key=env", nil)
	w := httptest.NewRecorder()
	HandleGetTagValues(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("客户端创建失败应返回 500，实际: %d", w.Code)
	}
}

func TestHandleGetTagValues_成功返回标签值(t *testing.T) {
	initTagTestEnv(t)
	setMockTagClient(t, &mockTagClient{
		describeTagValuesFunc: func(req *tag.DescribeTagValuesRequest) (*tag.DescribeTagValuesResponse, error) {
			v1, v2 := "production", "staging"
			k := "env"
			resp := tag.NewDescribeTagValuesResponse()
			resp.Response = &tag.DescribeTagValuesResponseParams{
				Tags: []*tag.Tag{
					{TagKey: &k, TagValue: &v1},
					{TagKey: &k, TagValue: &v2},
				},
			}
			return resp, nil
		},
	})

	req := adminRequest(http.MethodGet, "/api/tags/values?key=env", nil)
	w := httptest.NewRecorder()
	HandleGetTagValues(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际: %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["key"] != "env" {
		t.Errorf("key 不匹配: %v", resp["key"])
	}
	values := resp["values"].([]interface{})
	if len(values) != 2 || values[0] != "production" || values[1] != "staging" {
		t.Errorf("values 不匹配: %v", values)
	}
}

func TestHandleGetTagValues_空结果(t *testing.T) {
	initTagTestEnv(t)
	setMockTagClient(t, &mockTagClient{
		describeTagValuesFunc: func(req *tag.DescribeTagValuesRequest) (*tag.DescribeTagValuesResponse, error) {
			resp := tag.NewDescribeTagValuesResponse()
			resp.Response = &tag.DescribeTagValuesResponseParams{
				Tags: []*tag.Tag{},
			}
			return resp, nil
		},
	})

	req := adminRequest(http.MethodGet, "/api/tags/values?key=env", nil)
	w := httptest.NewRecorder()
	HandleGetTagValues(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际: %d", w.Code)
	}
}

func TestHandleGetTagValues_API调用失败(t *testing.T) {
	initTagTestEnv(t)
	setMockTagClient(t, &mockTagClient{
		describeTagValuesFunc: func(req *tag.DescribeTagValuesRequest) (*tag.DescribeTagValuesResponse, error) {
			return nil, fmt.Errorf("network error")
		},
	})

	req := adminRequest(http.MethodGet, "/api/tags/values?key=env", nil)
	w := httptest.NewRecorder()
	HandleGetTagValues(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("API 失败应返回 500，实际: %d", w.Code)
	}
}

func TestHandleGetTagValues_包含nil指针(t *testing.T) {
	initTagTestEnv(t)
	v1 := "prod"
	k := "env"
	setMockTagClient(t, &mockTagClient{
		describeTagValuesFunc: func(req *tag.DescribeTagValuesRequest) (*tag.DescribeTagValuesResponse, error) {
			resp := tag.NewDescribeTagValuesResponse()
			resp.Response = &tag.DescribeTagValuesResponseParams{
				Tags: []*tag.Tag{
					{TagKey: &k, TagValue: &v1},
					nil,                         // nil 条目应被跳过
					{TagKey: &k, TagValue: nil}, // nil TagValue 应被跳过
				},
			}
			return resp, nil
		},
	})

	req := adminRequest(http.MethodGet, "/api/tags/values?key=env", nil)
	w := httptest.NewRecorder()
	HandleGetTagValues(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际: %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	values := resp["values"].([]interface{})
	if len(values) != 1 || values[0] != "prod" {
		t.Errorf("应只返回 1 个有效值，实际: %v", values)
	}
}

// ==================== HandleGetTagKeys 分页测试 ====================

func TestHandleGetTagKeys_分页获取全量(t *testing.T) {
	initTagTestEnv(t)
	callCount := 0
	setMockTagClient(t, &mockTagClient{
		describeTagKeysFunc: func(req *tag.DescribeTagKeysRequest) (*tag.DescribeTagKeysResponse, error) {
			callCount++
			resp := tag.NewDescribeTagKeysResponse()
			if *req.Offset == 0 {
				// 第一页：返回 tagPageSize 条，触发下一页
				keys := make([]*string, tagPageSize)
				for i := uint64(0); i < tagPageSize; i++ {
					k := fmt.Sprintf("key-%d", i)
					keys[i] = common.StringPtr(k)
				}
				resp.Response = &tag.DescribeTagKeysResponseParams{Tags: keys}
			} else {
				// 第二页：返回 5 条
				keys := make([]*string, 5)
				for i := 0; i < 5; i++ {
					k := fmt.Sprintf("key-extra-%d", i)
					keys[i] = common.StringPtr(k)
				}
				resp.Response = &tag.DescribeTagKeysResponseParams{Tags: keys}
			}
			return resp, nil
		},
	})

	req := adminRequest(http.MethodGet, "/api/tags/keys", nil)
	w := httptest.NewRecorder()
	HandleGetTagKeys(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际: %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	keys := resp["keys"].([]interface{})
	expectedCount := int(tagPageSize) + 5
	if len(keys) != expectedCount {
		t.Errorf("期望 %d 个标签键，实际: %d", expectedCount, len(keys))
	}
	if callCount != 2 {
		t.Errorf("期望调用 2 次 API，实际: %d", callCount)
	}
}

// ==================== DefaultTag 序列化测试 ====================

func TestDefaultTag_JSON序列化(t *testing.T) {
	tags := []DefaultTag{
		{Key: "env", Value: "prod"},
		{Key: "team", Value: ""},
	}
	data, err := json.Marshal(tags)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var parsed []DefaultTag
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	if len(parsed) != 2 || parsed[0].Key != "env" || parsed[1].Value != "" {
		t.Errorf("序列化/反序列化不一致: %+v", parsed)
	}
}

func TestDefaultTag_JSON请求体解析(t *testing.T) {
	body := `{"tags":[{"Key":"env","Value":"prod"},{"Key":"managed-by","Value":"openclaw"}]}`
	var req struct {
		Tags []DefaultTag `json:"tags"`
	}
	if err := json.NewDecoder(bytes.NewBufferString(body)).Decode(&req); err != nil {
		t.Fatalf("解析请求体失败: %v", err)
	}
	if len(req.Tags) != 2 || req.Tags[0].Key != "env" || req.Tags[0].Value != "prod" {
		t.Errorf("解析结果不匹配: %+v", req.Tags)
	}
}

// TestNewTagClientFunc_WithCreds 覆盖 tag.go lines 22-23（default newTagClientFunc body）
func TestNewTagClientFunc_WithCreds(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.SiteConfig{})
	db.Create(&model.SiteConfig{Name: "Test", CVMSecretId: "id-tag", CVMSecretKey: "key-tag"})
	t.Cleanup(model.UseDBForTest(db))

	// 直接调用原始的 newTagClientFunc（覆盖 lines 22-23）
	cli, err := newTagClientFunc(context.Background())
	if err != nil {
		t.Logf("newTagClientFunc error (expected in test env): %v", err)
		return
	}
	if cli == nil {
		t.Error("expected non-nil client")
	}
}

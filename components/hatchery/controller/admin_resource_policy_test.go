package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"hatchery/controller/usergroup"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

func setupResourcePolicyHandlerTest(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SiteConfig{}, &model.ResourcePolicy{}, &model.UserGroup{}, &model.UserGroupMember{},
		&model.GroupClosure{}, &model.GroupConfigBinding{}, &model.User{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if err := db.Create(&model.SiteConfig{CVMTemplate: model.DefaultCVMTemplate}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	for _, group := range []model.UserGroup{
		{ID: 1, Name: "root", FullPath: "root", Source: model.GroupSourceManual},
		{ID: 2, Name: "child", FullPath: "root/child", ParentID: 1, Depth: 1, Source: model.GroupSourceManual},
	} {
		if err := db.Create(&group).Error; err != nil {
			t.Fatalf("seed group: %v", err)
		}
	}
	for _, closure := range []model.GroupClosure{
		{AncestorID: 1, DescendantID: 1, Depth: 0},
		{AncestorID: 2, DescendantID: 2, Depth: 0},
		{AncestorID: 1, DescendantID: 2, Depth: 1},
	} {
		if err := db.Create(&closure).Error; err != nil {
			t.Fatalf("seed closure: %v", err)
		}
	}
	t.Cleanup(model.UseDBForTest(db))
	oldToken, oldStore := AdminToken, Store
	AdminToken = "resource-policy-admin"
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	t.Cleanup(func() {
		AdminToken = oldToken
		Store = oldStore
	})
	return db
}

func resourcePolicyAdminRequest(method, path string, body any) *http.Request {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer resource-policy-admin")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestResourcePolicyHandlersCRUD(t *testing.T) {
	setupResourcePolicyHandlerTest(t)

	createBody := map[string]interface{}{
		"name": "研发资源策略",
		"resource_config": map[string]interface{}{
			"instance_type": "Ai2.MEDIUM4",
			"system_disk":   map[string]interface{}{"disk_type": "cloud_ssd", "disk_size": 100},
		},
		"group_ids": []uint{1},
	}
	createRecorder := httptest.NewRecorder()
	HandleCreateResourcePolicy(createRecorder, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/create", createBody))
	if createRecorder.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", createRecorder.Code, createRecorder.Body.String())
	}
	var created struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil || created.ID == 0 {
		t.Fatalf("decode create response: id=%d err=%v", created.ID, err)
	}

	listRecorder := httptest.NewRecorder()
	HandleListResourcePolicies(listRecorder, resourcePolicyAdminRequest(http.MethodGet, "/admin/resource-policies?page=1&page_size=10", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	var listed struct {
		Items []struct {
			ID             uint                      `json:"id"`
			Name           string                    `json:"name"`
			IsDefault      bool                      `json:"is_default"`
			ResourceConfig map[string]interface{}    `json:"resource_config"`
			Groups         []resourcePolicyGroupItem `json:"groups"`
		} `json:"items"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listed.Total != 2 || len(listed.Items) != 2 || !listed.Items[0].IsDefault {
		t.Fatalf("list=%+v, want default first plus created policy", listed)
	}
	if listed.Items[1].ID != created.ID || len(listed.Items[1].Groups) != 1 || listed.Items[1].Groups[0].ID != 1 {
		t.Fatalf("created list item=%+v", listed.Items[1])
	}
	if disk, ok := listed.Items[1].ResourceConfig["system_disk"].(map[string]interface{}); !ok || disk["disk_type"] != "CLOUD_SSD" {
		t.Fatalf("resource config was not normalized: %v", listed.Items[1].ResourceConfig)
	}

	updateBody := map[string]interface{}{
		"id": created.ID, "name": "更新后的策略",
		"resource_config": map[string]interface{}{"instance_type": "Ai2.LARGE8"},
		"group_ids":       []uint{2},
	}
	updateRecorder := httptest.NewRecorder()
	HandleUpdateResourcePolicy(updateRecorder, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", updateBody))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}
	direct, err := model.GetDirectResourcePoliciesByGroup(context.Background(), []uint{1, 2})
	if err != nil {
		t.Fatalf("query direct policies: %v", err)
	}
	if _, exists := direct[1]; exists || direct[2].ID != created.ID {
		t.Fatalf("direct policies after update=%v", direct)
	}

	deleteRecorder := httptest.NewRecorder()
	HandleDeleteResourcePolicy(deleteRecorder, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/delete", map[string]uint{"id": created.ID}))
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
	if _, err := model.GetResourcePolicy(context.Background(), created.ID); err != model.ErrResourcePolicyNotFound {
		t.Fatalf("deleted policy lookup error=%v", err)
	}
}

func TestResourcePolicyHandlerRejectsOccupiedGroupAndDefaultDelete(t *testing.T) {
	setupResourcePolicyHandlerTest(t)
	if _, err := model.CreateResourcePolicy(context.Background(), "first", `{}`, []uint{1}); err != nil {
		t.Fatalf("seed first policy: %v", err)
	}

	occupiedRecorder := httptest.NewRecorder()
	HandleCreateResourcePolicy(occupiedRecorder, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/create", map[string]interface{}{
		"name": "second", "resource_config": map[string]interface{}{}, "group_ids": []uint{1},
	}))
	if occupiedRecorder.Code != http.StatusConflict {
		t.Fatalf("occupied status=%d body=%s", occupiedRecorder.Code, occupiedRecorder.Body.String())
	}

	defaultPolicy, err := model.GetOrCreateDefaultResourcePolicy(context.Background())
	if err != nil {
		t.Fatalf("get default: %v", err)
	}
	const editedConfigType = "Ai2.LARGE8"
	updateRecorder := httptest.NewRecorder()
	HandleUpdateResourcePolicy(updateRecorder, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", map[string]interface{}{
		"id": defaultPolicy.ID, "resource_config": map[string]interface{}{"instance_type": editedConfigType},
	}))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update default config status=%d body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}
	for name, body := range map[string]map[string]interface{}{
		"rename": {"id": defaultPolicy.ID, "name": "renamed", "resource_config": map[string]interface{}{}},
		"bind":   {"id": defaultPolicy.ID, "resource_config": map[string]interface{}{}, "group_ids": []uint{2}},
	} {
		recorder := httptest.NewRecorder()
		HandleUpdateResourcePolicy(recorder, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", body))
		if recorder.Code != http.StatusConflict {
			t.Fatalf("%s default status=%d body=%s", name, recorder.Code, recorder.Body.String())
		}
	}
	reloadedDefault, err := model.GetResourcePolicy(context.Background(), defaultPolicy.ID)
	if err != nil {
		t.Fatalf("reload default: %v", err)
	}
	var defaultConfig ResourceConfig
	if err := json.Unmarshal([]byte(reloadedDefault.ConfigJSON), &defaultConfig); err != nil {
		t.Fatalf("decode edited default config: %v", err)
	}
	if defaultConfig.InstanceType != editedConfigType {
		t.Fatalf("default config=%+v, want edited instance type", defaultConfig)
	}
	deleteRecorder := httptest.NewRecorder()
	HandleDeleteResourcePolicy(deleteRecorder, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/delete", map[string]uint{"id": defaultPolicy.ID}))
	if deleteRecorder.Code != http.StatusConflict {
		t.Fatalf("delete default status=%d body=%s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestResourcePolicyAuditRulesRegistered(t *testing.T) {
	for _, path := range []string{
		"/admin/resource-policies/create",
		"/admin/resource-policies/update",
		"/admin/resource-policies/delete",
	} {
		if _, ok := auditRules[path]; !ok {
			t.Fatalf("audit rule missing for %s", path)
		}
	}
}

func TestGroupTreeReturnsDirectResourcePolicyOnly(t *testing.T) {
	setupResourcePolicyHandlerTest(t)
	policy, err := model.CreateResourcePolicy(context.Background(), "root-policy", `{}`, []uint{1})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	recorder := httptest.NewRecorder()
	req := resourcePolicyAdminRequest(http.MethodGet, "/admin/user-groups/tree?with_user_counts=false&with_resource_policy=true", nil)
	HandleAdminGetGroupTree(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("tree status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		UserGroups []struct {
			ID                   uint `json:"id"`
			DirectResourcePolicy *struct {
				ID   uint   `json:"id"`
				Name string `json:"name"`
			} `json:"direct_resource_policy"`
			Children []struct {
				ID                   uint        `json:"id"`
				DirectResourcePolicy interface{} `json:"direct_resource_policy"`
			} `json:"children"`
		} `json:"user_groups"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode tree: %v", err)
	}
	if len(response.UserGroups) != 1 || response.UserGroups[0].ID != 1 {
		t.Fatalf("user_groups=%+v", response.UserGroups)
	}
	if direct := response.UserGroups[0].DirectResourcePolicy; direct == nil || direct.ID != policy.ID || direct.Name != policy.Name {
		t.Fatalf("root direct policy=%+v", direct)
	}
	if len(response.UserGroups[0].Children) != 1 || response.UserGroups[0].Children[0].DirectResourcePolicy != nil {
		t.Fatalf("child should not expose inherited policy as direct: %+v", response.UserGroups[0].Children)
	}
}

func TestResourcePolicyOverviewAndCreateResolverShareWinner(t *testing.T) {
	setupResourcePolicyHandlerTest(t)
	const instanceType = "Ai2.LARGE8"
	policy, err := model.CreateResourcePolicy(
		context.Background(),
		"root-policy",
		`{"instance_type":"`+instanceType+`"}`,
		[]uint{1},
	)
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	createConfig, createSource, err := resolveEffectiveResourcePolicyConfig(context.Background(), 2)
	if err != nil {
		t.Fatalf("resolve create config: %v", err)
	}
	if createSource.Type != usergroup.SourceInherited || createSource.GroupID != 1 {
		t.Fatalf("create source=%+v, want inherited from group 1", createSource)
	}
	var parsed ResourceConfig
	if err := json.Unmarshal(createConfig, &parsed); err != nil {
		t.Fatalf("decode create config: %v", err)
	}
	if parsed.InstanceType != instanceType {
		t.Fatalf("create config=%+v, want %s", parsed, instanceType)
	}

	entries := buildResourcePolicyEntries(context.Background(), 2)
	if len(entries) != 1 {
		t.Fatalf("overview entries=%v, want one", entries)
	}
	entry := entries[0]
	if entry.ID != strconv.FormatUint(uint64(policy.ID), 10) ||
		entry.Label != policy.Name ||
		entry.Source.Type != createSource.Type ||
		entry.Source.GroupID != createSource.GroupID {
		t.Fatalf("overview entry=%+v does not match create resolver source=%+v policy=%+v", entry, createSource, policy)
	}
	meta, ok := entry.Meta.(map[string]interface{})
	if !ok || meta["policy_id"] != policy.ID {
		t.Fatalf("overview meta=%v, want policy_id %d", entry.Meta, policy.ID)
	}
	serialized, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal overview entry: %v", err)
	}
	var overviewPayload struct {
		Meta struct {
			Value          ResourceConfig `json:"value"`
			ResourceConfig ResourceConfig `json:"resource_config"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(serialized, &overviewPayload); err != nil {
		t.Fatalf("decode overview entry: %v", err)
	}
	if overviewPayload.Meta.Value.InstanceType != instanceType {
		t.Fatalf("overview meta.value=%+v, want instance_type %s", overviewPayload.Meta.Value, instanceType)
	}
	if overviewPayload.Meta.ResourceConfig.InstanceType != instanceType {
		t.Fatalf("overview meta.resource_config=%+v, want instance_type %s", overviewPayload.Meta.ResourceConfig, instanceType)
	}

	recorder := httptest.NewRecorder()
	HandleAdminGetGroupConfigOverview(
		recorder,
		resourcePolicyAdminRequest(
			http.MethodGet,
			"/admin/user-groups/config-overview?group_ids=2&keys=resourcePolicy",
			nil,
		),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("group config overview status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var overviewResponse struct {
		Results []struct {
			Categories []struct {
				Key     string `json:"key"`
				Entries []struct {
					ID   string `json:"id"`
					Meta struct {
						Value ResourceConfig `json:"value"`
					} `json:"meta"`
				} `json:"entries"`
			} `json:"categories"`
		} `json:"results"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &overviewResponse); err != nil {
		t.Fatalf("decode group config overview: %v", err)
	}
	if len(overviewResponse.Results) != 1 ||
		len(overviewResponse.Results[0].Categories) != 1 ||
		overviewResponse.Results[0].Categories[0].Key != usergroup.CategoryKeyResourcePolicy ||
		len(overviewResponse.Results[0].Categories[0].Entries) != 1 {
		t.Fatalf("unexpected group config overview: %s", recorder.Body.String())
	}
	responseEntry := overviewResponse.Results[0].Categories[0].Entries[0]
	if responseEntry.ID != strconv.FormatUint(uint64(policy.ID), 10) ||
		responseEntry.Meta.Value.InstanceType != instanceType {
		t.Fatalf("group config overview entry=%+v, body=%s", responseEntry, recorder.Body.String())
	}
}

func TestDefaultResourcePolicyUsesSiteDefaultSource(t *testing.T) {
	setupResourcePolicyHandlerTest(t)

	_, createSource, err := resolveEffectiveResourcePolicyConfig(context.Background(), 2)
	if err != nil {
		t.Fatalf("resolve default resource policy: %v", err)
	}
	if createSource.Type != usergroup.SourceSiteDefault || createSource.GroupID != 0 {
		t.Fatalf("create source=%+v, want site_default", createSource)
	}

	entries := buildResourcePolicyEntries(context.Background(), 2)
	if len(entries) != 1 || entries[0].Source.Type != usergroup.SourceSiteDefault {
		t.Fatalf("overview entries=%+v, want site_default source", entries)
	}
}

func TestResourcePolicyHandlerValidation(t *testing.T) {
	setupResourcePolicyHandlerTest(t)
	policy, err := model.CreateResourcePolicy(context.Background(), "editable", `{}`, []uint{1})
	if err != nil {
		t.Fatalf("seed editable policy: %v", err)
	}

	unauthorized := func(method, path string, body any) *http.Request {
		req := resourcePolicyAdminRequest(method, path, body)
		req.Header.Del("Authorization")
		return req
	}
	malformed := func(method, path string) *http.Request {
		req := httptest.NewRequest(method, path, strings.NewReader(`{`))
		req.Header.Set("Authorization", "Bearer resource-policy-admin")
		req.Header.Set("Content-Type", "application/json")
		return req
	}
	longName := strings.Repeat("策", 129)
	validCreate := map[string]interface{}{"name": "valid", "resource_config": map[string]interface{}{}, "group_ids": []uint{2}}
	validUpdate := map[string]interface{}{"id": policy.ID, "name": policy.Name, "resource_config": map[string]interface{}{}, "group_ids": []uint{1}}
	validDelete := map[string]interface{}{"id": policy.ID}

	cases := []struct {
		name    string
		handler http.HandlerFunc
		request *http.Request
		status  int
	}{
		{"list method", HandleListResourcePolicies, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies", nil), http.StatusMethodNotAllowed},
		{"list unauthorized", HandleListResourcePolicies, unauthorized(http.MethodGet, "/admin/resource-policies", nil), http.StatusUnauthorized},
		{"list bad page", HandleListResourcePolicies, resourcePolicyAdminRequest(http.MethodGet, "/admin/resource-policies?page=0", nil), http.StatusBadRequest},
		{"list bad page size", HandleListResourcePolicies, resourcePolicyAdminRequest(http.MethodGet, "/admin/resource-policies?page_size=101", nil), http.StatusBadRequest},
		{"create method", HandleCreateResourcePolicy, resourcePolicyAdminRequest(http.MethodGet, "/admin/resource-policies/create", validCreate), http.StatusMethodNotAllowed},
		{"create unauthorized", HandleCreateResourcePolicy, unauthorized(http.MethodPost, "/admin/resource-policies/create", validCreate), http.StatusUnauthorized},
		{"create malformed", HandleCreateResourcePolicy, malformed(http.MethodPost, "/admin/resource-policies/create"), http.StatusBadRequest},
		{"create empty name", HandleCreateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/create", map[string]interface{}{"resource_config": map[string]interface{}{}, "group_ids": []uint{2}}), http.StatusBadRequest},
		{"create long name", HandleCreateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/create", map[string]interface{}{"name": longName, "resource_config": map[string]interface{}{}, "group_ids": []uint{2}}), http.StatusBadRequest},
		{"create missing groups", HandleCreateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/create", map[string]interface{}{"name": "missing-groups", "resource_config": map[string]interface{}{}}), http.StatusBadRequest},
		{"create missing config", HandleCreateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/create", map[string]interface{}{"name": "missing-config", "group_ids": []uint{2}}), http.StatusBadRequest},
		{"create missing group", HandleCreateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/create", map[string]interface{}{"name": "missing-group", "resource_config": map[string]interface{}{}, "group_ids": []uint{999}}), http.StatusBadRequest},
		{"create reserved default name", HandleCreateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/create", map[string]interface{}{"name": model.DefaultResourcePolicyName, "resource_config": map[string]interface{}{}, "group_ids": []uint{2}}), http.StatusConflict},
		{"update method", HandleUpdateResourcePolicy, resourcePolicyAdminRequest(http.MethodGet, "/admin/resource-policies/update", validUpdate), http.StatusMethodNotAllowed},
		{"update unauthorized", HandleUpdateResourcePolicy, unauthorized(http.MethodPost, "/admin/resource-policies/update", validUpdate), http.StatusUnauthorized},
		{"update malformed", HandleUpdateResourcePolicy, malformed(http.MethodPost, "/admin/resource-policies/update"), http.StatusBadRequest},
		{"update missing id", HandleUpdateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", map[string]interface{}{"resource_config": map[string]interface{}{}}), http.StatusBadRequest},
		{"update missing config", HandleUpdateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", map[string]interface{}{"id": policy.ID, "name": policy.Name, "group_ids": []uint{1}}), http.StatusBadRequest},
		{"update not found", HandleUpdateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", map[string]interface{}{"id": 999, "resource_config": map[string]interface{}{}}), http.StatusNotFound},
		{"update empty name", HandleUpdateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", map[string]interface{}{"id": policy.ID, "resource_config": map[string]interface{}{}, "group_ids": []uint{1}}), http.StatusBadRequest},
		{"update long name", HandleUpdateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", map[string]interface{}{"id": policy.ID, "name": longName, "resource_config": map[string]interface{}{}, "group_ids": []uint{1}}), http.StatusBadRequest},
		{"update missing groups", HandleUpdateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", map[string]interface{}{"id": policy.ID, "name": policy.Name, "resource_config": map[string]interface{}{}}), http.StatusBadRequest},
		{"update to reserved default name", HandleUpdateResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", map[string]interface{}{"id": policy.ID, "name": model.DefaultResourcePolicyName, "resource_config": map[string]interface{}{}, "group_ids": []uint{1}}), http.StatusConflict},
		{"delete method", HandleDeleteResourcePolicy, resourcePolicyAdminRequest(http.MethodGet, "/admin/resource-policies/delete", validDelete), http.StatusMethodNotAllowed},
		{"delete unauthorized", HandleDeleteResourcePolicy, unauthorized(http.MethodPost, "/admin/resource-policies/delete", validDelete), http.StatusUnauthorized},
		{"delete malformed", HandleDeleteResourcePolicy, malformed(http.MethodPost, "/admin/resource-policies/delete"), http.StatusBadRequest},
		{"delete missing id", HandleDeleteResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/delete", map[string]interface{}{}), http.StatusBadRequest},
		{"delete not found", HandleDeleteResourcePolicy, resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/delete", map[string]interface{}{"id": 999}), http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tc.handler(recorder, tc.request)
			if recorder.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, tc.status, recorder.Body.String())
			}
		})
	}
}

func TestResourcePolicyDefaultNameLocalizedOnRead(t *testing.T) {
	setupResourcePolicyHandlerTest(t)
	ordinary, err := model.CreateResourcePolicy(context.Background(), "研发自定义策略", `{}`, []uint{2})
	if err != nil {
		t.Fatalf("create ordinary policy: %v", err)
	}
	defaultPolicy, err := model.GetOrCreateDefaultResourcePolicy(context.Background())
	if err != nil {
		t.Fatalf("get default policy: %v", err)
	}

	listRecorder := httptest.NewRecorder()
	listRequest := resourcePolicyAdminRequest(http.MethodGet, "/admin/resource-policies", nil)
	listRequest.Header.Set("Accept-Language", "en")
	I18nMiddleware(http.HandlerFunc(HandleListResourcePolicies)).ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("English list status=%d body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	var response struct {
		Items []struct {
			ID        uint   `json:"id"`
			Name      string `json:"name"`
			IsDefault bool   `json:"is_default"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode English list: %v", err)
	}
	names := make(map[uint]string, len(response.Items))
	for _, item := range response.Items {
		names[item.ID] = item.Name
	}
	if names[defaultPolicy.ID] != "Enterprise Default Resource Policy" {
		t.Fatalf("English default name=%q", names[defaultPolicy.ID])
	}
	if names[ordinary.ID] != ordinary.Name {
		t.Fatalf("ordinary policy name was localized: got=%q want=%q", names[ordinary.ID], ordinary.Name)
	}

	var overviewEntries []usergroup.ConfigEntry
	overviewRequest := httptest.NewRequest(http.MethodGet, "/overview", nil)
	overviewRequest.Header.Set("Accept-Language", "en")
	I18nMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		overviewEntries = buildResourcePolicyEntries(r.Context(), 1)
	})).ServeHTTP(httptest.NewRecorder(), overviewRequest)
	if len(overviewEntries) != 1 || overviewEntries[0].Label != "Enterprise Default Resource Policy" {
		t.Fatalf("English overview entries=%+v", overviewEntries)
	}

	updateRecorder := httptest.NewRecorder()
	updateRequest := resourcePolicyAdminRequest(http.MethodPost, "/admin/resource-policies/update", map[string]interface{}{
		"id":              defaultPolicy.ID,
		"name":            "Enterprise Default Resource Policy",
		"resource_config": map[string]interface{}{},
	})
	updateRequest.Header.Set("Accept-Language", "en")
	I18nMiddleware(http.HandlerFunc(HandleUpdateResourcePolicy)).ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("localized round-trip update status=%d body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}
	persisted, err := model.GetResourcePolicy(context.Background(), defaultPolicy.ID)
	if err != nil {
		t.Fatalf("reload persisted default: %v", err)
	}
	if persisted.Name != model.DefaultResourcePolicyName {
		t.Fatalf("localized read changed persisted name to %q", persisted.Name)
	}
}

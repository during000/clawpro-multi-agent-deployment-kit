package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gorm.io/gorm"
)

// ─── helpers ───────────────────────────────────────────────────────────────

// adminCreateJSONReq builds a JSON POST request to /admin/instances/create
// authenticated with AdminToken.
func adminCreateJSONReq(t *testing.T, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/create", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	return req
}

// adminCreateSessionReq builds a JSON POST request authenticated with a
// session cookie for the given username.
func adminCreateSessionReq(t *testing.T, username string, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/create", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// parseJSONResp unmarshals a recorder body into map[string]any.
func parseJSONResp(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("response not valid JSON: %v\nbody=%s", err, rr.Body.String())
	}
	return m
}

// setupAdminCreateValidationDB initialises an in-memory SQLite with all models
// needed for admin create validation tests, creates a SiteConfig row, and
// returns a cleanup function.
func setupAdminCreateValidationDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.SiteConfig{},
		&model.AIImage{}, &model.AuditLog{}, &model.Notification{},
		&model.SkillInstallation{}, &model.SMHPersonalSpace{},
		&model.MemoryTDAIPlugin{}, &model.RuleSet{},
		&model.GroupConfigBinding{}, &model.ResourcePolicy{}, &model.OpenClawRole{},
		&model.RoleVisibilityGroup{}, &model.UserGroup{}, &model.GroupClosure{},
		&model.UserGroupMember{},
		&model.InstanceModel{}, &model.AIModel{}, &model.AIChannel{},
		&model.PluginInstallation{}, &model.Skill{}, &model.SkillBundle{},
		&model.BundleSkill{}, &model.OpenClawRoleSkill{},
		&model.MemoryPlanGroupPolicy{}, &model.Tag{},
		&model.ModelVisibilityGroup{}, &model.SkillVisibilityGroup{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTestWithDriver(db, "sqlite")
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	AdminToken = "test-admin-token"
	_ = model.DB(context.Background()).Create(&model.SiteConfig{})

	return func() {
		time.Sleep(100 * time.Millisecond)
		origDB()
		Store = origStore
	}
}

func TestRecoverCreatePresetPanic_RecoversAndUntracks(t *testing.T) {
	const instanceID uint = 987654
	trackGatewayRestartTask(instanceID)

	func() {
		defer recoverCreatePresetPanic("test", instanceID)
		defer untrackGatewayRestartTask(instanceID)
		panic("preset panic")
	}()

	if hasPendingGatewayRestartTasks(instanceID) {
		t.Fatal("recovered preset task left the gateway restart tracker pending")
	}
}

// ─── group helpers ──────────────────────────────────────────────────────────

func createTestGroup(t *testing.T, name string, toBeDeleted bool) *model.UserGroup {
	t.Helper()
	g := &model.UserGroup{Name: name, ToBeDeleted: toBeDeleted}
	if err := model.DB(context.Background()).Create(g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	return g
}

func addUserToGroup(t *testing.T, userID, groupID uint) {
	t.Helper()
	m := &model.UserGroupMember{UserID: userID, UserGroupID: groupID}
	if err := model.DB(context.Background()).Create(m).Error; err != nil {
		t.Fatalf("add user to group: %v", err)
	}
}

// ─── UT-02: admin 创建要求管理员权限 ──────────────────────────────────────────

func TestAdminCreate_RequiresAdmin_NoAuth(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/instances/create",
		strings.NewReader(`{"user_id":1,"name":"test","agent_type":"openclaw"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_RequiresAdmin_NonAdminUser(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "regular", Password: "x", Role: "user"}
	db.Create(user)

	req := adminCreateSessionReq(t, "regular", map[string]any{
		"user_id":    user.ID,
		"name":       "test",
		"agent_type": "openclaw",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("want 403 for non-admin, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_RequiresAdmin_AdminTokenOK(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "target", Password: "x", Role: "user"}
	db.Create(user)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "test",
		"agent_type": "openclaw",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	// Should pass auth and reach agent_type check (openclaw is a valid built-in type).
	// Without an enabled image, createInstance returns 400.
	if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
		t.Errorf("admin token should pass auth, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── UT-03: admin 创建目标用户不存在 ──────────────────────────────────────────

func TestAdminCreate_TargetUserNotFound(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    99999,
		"name":       "test",
		"agent_type": "openclaw",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for non-existent user, got %d body=%s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResp(t, rr)
	if errMsg, _ := resp["error"].(string); errMsg == "" {
		t.Error("want error message in response")
	} else if !strings.Contains(errMsg, "99999") {
		t.Errorf("error should mention user ID 99999, got %q", errMsg)
	}
}

func TestAdminCreate_TargetUserSoftDeleted(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "deleted-user", Password: "x", Role: "user"}
	db.Create(user)
	db.Delete(user) // soft delete

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "test",
		"agent_type": "openclaw",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for soft-deleted user, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── UT-04: 分组 0/1/多规则 ──────────────────────────────────────────────────

func TestAdminCreate_GroupZeroGroupsAutoSelectsZero(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "nogroup-user", Password: "x", Role: "user"}
	db.Create(user)

	groupID, err := resolveAdminCreateGroup(context.Background(), user.ID, 0)
	if err != nil {
		t.Fatalf("resolveAdminCreateGroup failed: %v", err)
	}
	if groupID != 0 {
		t.Errorf("want groupID=0 for user with 0 groups, got %d", groupID)
	}
}

func TestAdminCreate_GroupOneValidAutoSelects(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "onegroup-user", Password: "x", Role: "user"}
	db.Create(user)
	g := createTestGroup(t, "only-group", false)
	addUserToGroup(t, user.ID, g.ID)

	groupID, err := resolveAdminCreateGroup(context.Background(), user.ID, 0)
	if err != nil {
		t.Fatalf("resolveAdminCreateGroup failed: %v", err)
	}
	if groupID != g.ID {
		t.Errorf("want groupID=%d for user with 1 group, got %d", g.ID, groupID)
	}
}

func TestAdminCreate_GroupManyRequiresExplicit(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "manygroup-user", Password: "x", Role: "user"}
	db.Create(user)
	g1 := createTestGroup(t, "group-A", false)
	g2 := createTestGroup(t, "group-B", false)
	addUserToGroup(t, user.ID, g1.ID)
	addUserToGroup(t, user.ID, g2.ID)

	_, err := resolveAdminCreateGroup(context.Background(), user.ID, 0)
	if err == nil {
		t.Fatal("want error when user has multiple groups and no group_id specified")
	}
}

func TestAdminCreate_GroupManyWithExplicitOK(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "manygroup-user2", Password: "x", Role: "user"}
	db.Create(user)
	g1 := createTestGroup(t, "group-C", false)
	g2 := createTestGroup(t, "group-D", false)
	addUserToGroup(t, user.ID, g1.ID)
	addUserToGroup(t, user.ID, g2.ID)

	groupID, err := resolveAdminCreateGroup(context.Background(), user.ID, g1.ID)
	if err != nil {
		t.Fatalf("resolveAdminCreateGroup failed: %v", err)
	}
	if groupID != g1.ID {
		t.Errorf("want groupID=%d, got %d", g1.ID, groupID)
	}
}

// ─── UT-05: 分组必须属于目标用户 ──────────────────────────────────────────────

func TestAdminCreate_GroupNotBelongToUser(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "user-owner", Password: "x", Role: "user"}
	db.Create(user)
	otherUser := &model.User{Username: "user-other", Password: "x", Role: "user"}
	db.Create(otherUser)

	g := createTestGroup(t, "other-group", false)
	addUserToGroup(t, otherUser.ID, g.ID)

	// user requests group that belongs to otherUser
	_, err := resolveAdminCreateGroup(context.Background(), user.ID, g.ID)
	if err == nil {
		t.Fatal("want error when group does not belong to target user")
	}
}

// ─── UT-06: 分组排除 to_be_deleted ────────────────────────────────────────────

func TestAdminCreate_GroupExcludesToBeDeleted(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "tbd-user", Password: "x", Role: "user"}
	db.Create(user)
	g := createTestGroup(t, "doomed-group", true) // to_be_deleted=true
	addUserToGroup(t, user.ID, g.ID)

	groupID, err := resolveAdminCreateGroup(context.Background(), user.ID, 0)
	if err != nil {
		t.Fatalf("resolveAdminCreateGroup failed: %v", err)
	}
	if groupID != 0 {
		t.Errorf("want groupID=0 when only group is to_be_deleted, got %d", groupID)
	}
}

func TestAdminCreate_GroupExplicitToBeDeletedRejected(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "tbd-user2", Password: "x", Role: "user"}
	db.Create(user)
	g := createTestGroup(t, "doomed-group2", true)
	addUserToGroup(t, user.ID, g.ID)

	_, err := resolveAdminCreateGroup(context.Background(), user.ID, g.ID)
	if err == nil {
		t.Fatal("want error when explicitly requesting to_be_deleted group")
	}
}

func TestAdminCreate_GroupMixedToBeDeletedAndValid(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "mixed-user", Password: "x", Role: "user"}
	db.Create(user)
	badGroup := createTestGroup(t, "doomed-group3", true)
	goodGroup := createTestGroup(t, "alive-group", false)
	addUserToGroup(t, user.ID, badGroup.ID)
	addUserToGroup(t, user.ID, goodGroup.ID)

	// Without explicit group_id, should auto-select the valid (non-to_be_deleted) one.
	groupID, err := resolveAdminCreateGroup(context.Background(), user.ID, 0)
	if err != nil {
		t.Fatalf("resolveAdminCreateGroup failed: %v", err)
	}
	if groupID != goodGroup.ID {
		t.Errorf("want groupID=%d (valid group), got %d", goodGroup.ID, groupID)
	}
}

// ─── UT-07: 配额按目标用户/分组计算 ──────────────────────────────────────────

func TestAdminCreate_UserQuotaExceeded(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "quota-user", Password: "x", Role: "user", InstanceQuota: 1}
	db.Create(user)

	// Create an enabled image so image lookup succeeds.
	img := &model.AIImage{
		ImageId: "img-quota", ImageName: "test-img",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	}
	db.Create(img)

	// Create one existing instance to fill quota.
	inst := &model.Instance{
		Name: "existing", InstanceId: "ins-existing",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	db.Create(inst)

	// Set CVMTemplate so createInstance can proceed past the template check.
	// The quota check comes before CVM client creation, so CVM is never contacted.
	cfg := model.GetSiteConfig(context.Background())
	cfg.CVMTemplate = `{}`
	origRegion := CVMRegion
	CVMRegion = "ap-guangzhou"
	t.Cleanup(func() { CVMRegion = origRegion })
	db.Save(&cfg)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "quota-test",
		"agent_type": "openclaw",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("want 403 for quota exceeded, got %d body=%s", rr.Code, rr.Body.String())
	}

	// Verify no new instance placeholder was created.
	var count int64
	db.Model(&model.Instance{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 1 {
		t.Errorf("want 1 instance (no placeholder created), got %d", count)
	}
}

func TestAdminCreate_GroupQuotaExceeded(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "gquota-user", Password: "x", Role: "user", InstanceQuota: 10}
	db.Create(user)
	g := createTestGroup(t, "quota-group", false)
	addUserToGroup(t, user.ID, g.ID)

	img := &model.AIImage{
		ImageId: "img-gquota", ImageName: "test-img",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	}
	db.Create(img)

	// Create instances to fill group quota (default is 3 from GetSiteConfig fallback).
	for i := range 3 {
		db.Create(&model.Instance{
			Name: "g-inst", InstanceId: "ins-g-" + string(rune('a'+i)),
			UserID: user.ID, GroupID: g.ID, AgentType: model.AgentTypeOpenClaw,
		})
	}

	cfg := model.GetSiteConfig(context.Background())
	cfg.CVMTemplate = `{}`
	cfg.DefaultInstanceQuota = 3
	origRegion := CVMRegion
	CVMRegion = "ap-guangzhou"
	t.Cleanup(func() { CVMRegion = origRegion })
	db.Save(&cfg)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "gquota-test",
		"agent_type": "openclaw",
		"group_id":   g.ID,
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("want 403 for group quota exceeded, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── UT-08: 模型 primary/fallbacks 校验 ───────────────────────────────────────

func TestAdminCreate_ModelPrimaryNotFound(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "model-user", Password: "x", Role: "user"}
	db.Create(user)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "model-test",
		"agent_type": "openclaw",
		"models": map[string]any{
			"primary": 999,
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for non-existent primary model, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ModelPrimaryDisabled(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "model-user2", Password: "x", Role: "user"}
	db.Create(user)

	m := &model.AIModel{ModelID: "gpt-4", ModelName: "GPT-4", Enabled: false, Visible: true}
	db.Create(m)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "model-test",
		"agent_type": "openclaw",
		"models": map[string]any{
			"primary": m.ID,
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for disabled model, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ModelPrimaryNotVisible(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "model-user3", Password: "x", Role: "user"}
	db.Create(user)

	m := &model.AIModel{ModelID: "gpt-4", ModelName: "GPT-4", Enabled: true, Visible: false}
	db.Create(m)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "model-test",
		"agent_type": "openclaw",
		"models": map[string]any{
			"primary": m.ID,
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invisible model, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ModelPrimaryGroupInvisible(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "model-user4", Password: "x", Role: "user"}
	db.Create(user)
	g := createTestGroup(t, "model-group", false)
	addUserToGroup(t, user.ID, g.ID)

	// Create a model visible only to group with ID 99999 (different group).
	m := &model.AIModel{
		ModelID: "gpt-4", ModelName: "GPT-4",
		Enabled: true, Visible: true, VisibilityType: "group",
	}
	db.Create(m)
	db.Create(&model.ModelVisibilityGroup{AIModelID: m.ID, GroupID: 99999})

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "model-test",
		"agent_type": "openclaw",
		"group_id":   g.ID,
		"models": map[string]any{
			"primary": m.ID,
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for group-invisible model, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ModelDuplicatePrimaryAndFallback(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "model-user5", Password: "x", Role: "user"}
	db.Create(user)

	m := &model.AIModel{ModelID: "gpt-4", ModelName: "GPT-4", Enabled: true, Visible: true}
	db.Create(m)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "model-test",
		"agent_type": "openclaw",
		"models": map[string]any{
			"primary":   m.ID,
			"fallbacks": []uint{m.ID},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for duplicate model ID, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ModelFallbackNotFound(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "model-user6", Password: "x", Role: "user"}
	db.Create(user)

	m := &model.AIModel{ModelID: "gpt-4", ModelName: "GPT-4", Enabled: true, Visible: true}
	db.Create(m)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "model-test",
		"agent_type": "openclaw",
		"models": map[string]any{
			"primary":   m.ID,
			"fallbacks": []uint{99999},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for non-existent fallback, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ModelFallbackDisabled(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "model-user7", Password: "x", Role: "user"}
	db.Create(user)

	primary := &model.AIModel{ModelID: "gpt-4", ModelName: "GPT-4", Enabled: true, Visible: true}
	db.Create(primary)
	fallback := &model.AIModel{ModelID: "claude", ModelName: "Claude", Enabled: false, Visible: true}
	db.Create(fallback)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "model-test",
		"agent_type": "openclaw",
		"models": map[string]any{
			"primary":   primary.ID,
			"fallbacks": []uint{fallback.ID},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for disabled fallback, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestResolveAdminCreatePresets_ModelFallbackCapabilityMatrix(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	primary := model.AIModel{
		ModelID: "cap-primary", ModelName: "Capability Primary",
		Enabled: true, Visible: true, VisibilityType: model.VisibilityAll,
	}
	fallback := model.AIModel{
		ModelID: "cap-fallback", ModelName: "Capability Fallback",
		Enabled: true, Visible: true, VisibilityType: model.VisibilityAll,
	}
	if err := model.DB(context.Background()).Create(&primary).Error; err != nil {
		t.Fatalf("create primary: %v", err)
	}
	if err := model.DB(context.Background()).Create(&fallback).Error; err != nil {
		t.Fatalf("create fallback: %v", err)
	}

	tests := []struct {
		name       string
		agentType  string
		fallbacks  []uint
		wantErr    bool
		wantModels int
	}{
		{name: "openclaw multiple", agentType: model.AgentTypeOpenClaw, fallbacks: []uint{fallback.ID}, wantModels: 2},
		{name: "hermes single", agentType: model.AgentTypeHermes, wantModels: 1},
		{name: "hermes fallback", agentType: model.AgentTypeHermes, fallbacks: []uint{fallback.ID}, wantErr: true},
		{name: "ace single", agentType: model.AgentTypeLightclawACE, wantModels: 1},
		{name: "ace fallback", agentType: model.AgentTypeLightclawACE, fallbacks: []uint{fallback.ID}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presets, err := resolveAdminCreatePresets(context.Background(), adminCreateInstanceRequest{
				AgentType: tt.agentType,
				Models: &adminCreateModelsRequest{
					Primary:   primary.ID,
					Fallbacks: tt.fallbacks,
				},
			}, 0)
			if tt.wantErr {
				if err == nil {
					t.Fatal("fallback should be rejected for single-model runtime")
				}
				if !strings.Contains(err.Error(), "单模型") {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve presets: %v", err)
			}
			if len(presets.Models) != tt.wantModels {
				t.Fatalf("resolved model count = %d, want %d", len(presets.Models), tt.wantModels)
			}
		})
	}
}

// ─── UT-13: 通道缺少必填 config ──────────────────────────────────────────────

func TestAdminCreate_ChannelMissingRequiredConfig(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "ch-user", Password: "x", Role: "user"}
	db.Create(user)

	// Create a channel record for feishu that is enabled.
	enabled := true
	feishu := &model.AIChannel{
		ChannelID: "feishu", Name: "Feishu",
		Enabled: &enabled, Custom: false, VisibilityType: "all",
	}
	db.Create(feishu)

	// Missing app_secret - only app_id provided.
	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "ch-test",
		"agent_type": "openclaw",
		"channels": []map[string]any{
			{
				"channel": "feishu",
				"config": map[string]string{
					"app_id": "test-app-id",
				},
			},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing required channel config key, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ChannelEmptyConfigValue(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "ch-user2", Password: "x", Role: "user"}
	db.Create(user)

	enabled := true
	feishu := &model.AIChannel{
		ChannelID: "feishu", Name: "Feishu",
		Enabled: &enabled, Custom: false, VisibilityType: "all",
	}
	db.Create(feishu)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "ch-test",
		"agent_type": "openclaw",
		"channels": []map[string]any{
			{
				"channel": "feishu",
				"config": map[string]string{
					"app_id":     "test-app-id",
					"app_secret": "",
				},
			},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty config value, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ChannelEmptyKey(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "ch-user3", Password: "x", Role: "user"}
	db.Create(user)

	enabled := true
	feishu := &model.AIChannel{
		ChannelID: "feishu", Name: "Feishu",
		Enabled: &enabled, Custom: false, VisibilityType: "all",
	}
	db.Create(feishu)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "ch-test",
		"agent_type": "openclaw",
		"channels": []map[string]any{
			{
				"channel": "feishu",
				"config": map[string]string{
					"": "some-value",
				},
			},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty config key, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── UT-14: 通道不可见/agent_type 不支持 ───────────────────────────────────────

func TestAdminCreate_ChannelGroupInvisible(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "chvis-user", Password: "x", Role: "user"}
	db.Create(user)
	g := createTestGroup(t, "ch-group", false)
	addUserToGroup(t, user.ID, g.ID)

	enabled := true
	feishu := &model.AIChannel{
		ChannelID: "feishu", Name: "Feishu",
		Enabled: &enabled, Custom: false, VisibilityType: "group",
	}
	db.Create(feishu)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "chvis-test",
		"agent_type": "openclaw",
		"group_id":   g.ID,
		"channels": []map[string]any{
			{
				"channel": "feishu",
				"config": map[string]string{
					"app_id":     "test-app-id",
					"app_secret": "test-secret",
				},
			},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden {
		t.Errorf("want 400/403 for group-invisible channel, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ChannelAgentTypeNotSupported(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "chat-user", Password: "x", Role: "user"}
	db.Create(user)

	enabled := true
	feishu := &model.AIChannel{
		ChannelID: "feishu", Name: "Feishu",
		Enabled: &enabled, Custom: false, VisibilityType: "all",
	}
	db.Create(feishu)

	// Hermes does not support channels.
	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "chat-test",
		"agent_type": "hermes",
		"channels": []map[string]any{
			{
				"channel": "feishu",
				"config": map[string]string{
					"app_id":     "test-app-id",
					"app_secret": "test-secret",
				},
			},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for channel on channel-unsupported agent_type, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── UT-18: 技能不可见/不存在 ─────────────────────────────────────────────────

func TestAdminCreate_SkillNotFound(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "sk-user", Password: "x", Role: "user"}
	db.Create(user)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "sk-test",
		"agent_type": "openclaw",
		"skills": []map[string]any{
			{"slug": "nonexistent-skill"},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for non-existent skill, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_SkillGroupInvisible(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "skvis-user", Password: "x", Role: "user"}
	db.Create(user)
	g := createTestGroup(t, "sk-group", false)
	addUserToGroup(t, user.ID, g.ID)

	sk := &model.Skill{
		Slug: "hidden-skill", Name: "Hidden", Version: "1.0.0",
		VisibilityType: "group", VersionMajor: 1,
	}
	db.Create(sk)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "skvis-test",
		"agent_type": "openclaw",
		"group_id":   g.ID,
		"skills": []map[string]any{
			{"slug": "hidden-skill"},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for group-invisible skill, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_AgentTypeUnsupportedModels(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "at-user", Password: "x", Role: "user"}
	db.Create(user)

	// deepseektui does not support models (SupportsModel=false).
	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "at-test",
		"agent_type": "deepseektui",
		"models": map[string]any{
			"primary": 1,
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for models on model-unsupported agent_type, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_AgentTypeUnsupportedChannels(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "at-user2", Password: "x", Role: "user"}
	db.Create(user)

	// deepseektui does not support channels (SupportsChannel=false).
	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "at-test",
		"agent_type": "deepseektui",
		"channels": []map[string]any{
			{"channel": "feishu", "config": map[string]string{"app_id": "x", "app_secret": "y"}},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for channels on channel-unsupported agent_type, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_AgentTypeUnsupportedSkills(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "at-user3", Password: "x", Role: "user"}
	db.Create(user)

	// deepseektui does not support skills (SupportsSkill=false).
	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "at-test",
		"agent_type": "deepseektui",
		"skills": []map[string]any{
			{"slug": "some-skill"},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for skills on skill-unsupported agent_type, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── additional edge cases ───────────────────────────────────────────────────

func TestAdminCreate_MissingRequiredFields(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	// Missing user_id
	req := adminCreateJSONReq(t, map[string]any{
		"name":       "test",
		"agent_type": "openclaw",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing user_id, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_MissingName(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "name-user", Password: "x", Role: "user"}
	db.Create(user)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"agent_type": "openclaw",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing name, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_MissingAgentType(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "at-user4", Password: "x", Role: "user"}
	db.Create(user)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id": user.ID,
		"name":    "test",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing agent_type, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_InvalidJSON(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/instances/create",
		strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid JSON, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_TrailingJSONRejected(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/instances/create",
		strings.NewReader(`{"user_id":1,"name":"test","agent_type":"openclaw"}{"extra":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for trailing JSON, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_UnknownJSONFieldRejected(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":          1,
		"name":             "test",
		"agent_type":       "openclaw",
		"unexpected_field": true,
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for unknown JSON field, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_InvalidAgentType(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "inv-at-user", Password: "x", Role: "user"}
	db.Create(user)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "test",
		"agent_type": "nonexistent-type-xyz",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid agent_type, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_MethodNotAllowed(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/admin/instances/create", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405 for GET, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_NameTooLong(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "longname-user", Password: "x", Role: "user"}
	db.Create(user)

	longName := strings.Repeat("a", 129)
	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       longName,
		"agent_type": "openclaw",
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for name > 128 chars, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_ChannelDuplicateInRequest(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "chdup-user", Password: "x", Role: "user"}
	db.Create(user)

	enabled := true
	feishu := &model.AIChannel{
		ChannelID: "feishu", Name: "Feishu",
		Enabled: &enabled, Custom: false, VisibilityType: "all",
	}
	db.Create(feishu)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "chdup-test",
		"agent_type": "openclaw",
		"channels": []map[string]any{
			{
				"channel": "feishu",
				"config":  map[string]string{"app_id": "a", "app_secret": "b"},
			},
			{
				"channel": "feishu",
				"config":  map[string]string{"app_id": "a", "app_secret": "b"},
			},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for duplicate channel, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminCreate_SkillSourceUnsupported(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	user := &model.User{Username: "sksrc-user", Password: "x", Role: "user"}
	db.Create(user)

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    user.ID,
		"name":       "sksrc-test",
		"agent_type": "openclaw",
		"skills": []map[string]any{
			{"source": "public", "slug": "some-skill"},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400 for unsupported skill source, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── UT-19: resolveAdminCreateSkills coverage ──────────────────────────────

func TestAdminCreate_SkillResolveGlobalVisible(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	sk := &model.Skill{
		Slug: "global-skill", Name: "Global", Version: "1.0.0",
		VisibilityType: "all", VersionMajor: 1,
	}
	db.Create(sk)

	input := []adminCreateSkillRequest{{Slug: "global-skill", Source: model.SkillSourceEnterprise}}
	result, err := resolveAdminCreateSkills(context.Background(), 0, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result))
	}
	if result[0].Enterprise.ID != sk.ID {
		t.Errorf("skill ID mismatch: want %d, got %d", sk.ID, result[0].Enterprise.ID)
	}
}

func TestAdminCreate_SkillResolveGroupVisible(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	g := createTestGroup(t, "skill-group", false)
	// GroupClosure: self-referencing so GetAncestorIDs returns [g.ID]
	db.Create(&model.GroupClosure{AncestorID: g.ID, DescendantID: g.ID})

	sk := &model.Skill{
		Slug: "group-skill", Name: "Group", Version: "1.0.0",
		VisibilityType: "group", VersionMajor: 1,
	}
	db.Create(sk)
	db.Create(&model.SkillVisibilityGroup{SkillID: sk.ID, GroupID: g.ID})

	input := []adminCreateSkillRequest{{Slug: "group-skill", Source: model.SkillSourceEnterprise}}
	result, err := resolveAdminCreateSkills(context.Background(), g.ID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result))
	}
}

func TestAdminCreate_SkillResolveEmptySourceDefaults(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	input := []adminCreateSkillRequest{{Slug: "anysearch", Source: ""}}
	result, err := resolveAdminCreateSkills(context.Background(), 0, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result))
	}
	if result[0].Source != model.SkillSourcePublic || result[0].Slug != "anysearch" {
		t.Fatalf("empty source must resolve as public anysearch: %+v", result[0])
	}
}

func TestAdminCreate_SkillResolveBlankVersionLatest(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	// Create two versions of the same slug
	v1 := &model.Skill{
		Slug: "multi-ver-skill", Name: "MV", Version: "1.0.0",
		VisibilityType: "all", VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
	}
	db.Create(v1)
	v2 := &model.Skill{
		Slug: "multi-ver-skill", Name: "MV", Version: "2.0.0",
		VisibilityType: "all", VersionMajor: 2, VersionMinor: 0, VersionPatch: 0,
	}
	db.Create(v2)

	// Blank version should select the latest (v2)
	input := []adminCreateSkillRequest{{Slug: "multi-ver-skill", Version: "", Source: model.SkillSourceEnterprise}}
	result, err := resolveAdminCreateSkills(context.Background(), 0, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result))
	}
	if result[0].Enterprise.ID != v2.ID {
		t.Errorf("blank version should select latest: want ID %d (v2.0.0), got ID %d (v%s)",
			v2.ID, result[0].Enterprise.ID, result[0].Version)
	}
}

func TestAdminCreate_SkillResolveDuplicateDedup(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	sk := &model.Skill{
		Slug: "dedup-skill", Name: "Dedup", Version: "1.0.0",
		VisibilityType: "all", VersionMajor: 1,
	}
	db.Create(sk)

	// Two identical requests should deduplicate
	input := []adminCreateSkillRequest{
		{Slug: "dedup-skill"},
		{Slug: "dedup-skill"},
	}
	result, err := resolveAdminCreateSkills(context.Background(), 0, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 skill after dedup, got %d", len(result))
	}
}

func TestAdminCreate_SkillResolveDifferentVersionNotDeduped(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	v1 := &model.Skill{
		Slug: "diff-ver-skill", Name: "DV", Version: "1.0.0",
		VisibilityType: "all", VersionMajor: 1,
	}
	db.Create(v1)
	v2 := &model.Skill{
		Slug: "diff-ver-skill", Name: "DV", Version: "2.0.0",
		VisibilityType: "all", VersionMajor: 2,
	}
	db.Create(v2)

	// Different versions should NOT be deduplicated (explicit version requests)
	input := []adminCreateSkillRequest{
		{Slug: "diff-ver-skill", Version: "1.0.0"},
		{Slug: "diff-ver-skill", Version: "2.0.0"},
	}
	result, err := resolveAdminCreateSkills(context.Background(), 0, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 skills for different versions, got %d", len(result))
	}
}

// ─── UT-20: resolveAdminCreatePresets combined success ────────────────────

func TestAdminCreate_PresetsCombinedModelsChannelsSkills(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	db := model.DB(context.Background())

	// model
	am := &model.AIModel{
		ModelName: "test-model", ModelID: "model-1",
		Enabled: true, Visible: true, VisibilityType: "all",
	}
	db.Create(am)

	// channel
	enabled := true
	ac := &model.AIChannel{
		ChannelID: "openclaw-weixin", Name: "weixin",
		Enabled: &enabled, VisibilityType: "all",
	}
	db.Create(ac)

	// skill
	sk := &model.Skill{
		Slug: "combo-skill", Name: "Combo", Version: "1.0.0",
		VisibilityType: "all", VersionMajor: 1,
	}
	db.Create(sk)

	input := adminCreateInstanceRequest{
		AgentType: "openclaw",
		Models:    &adminCreateModelsRequest{Primary: am.ID},
		Channels:  []adminCreateChannelRequest{{Channel: "openclaw-weixin", Config: map[string]string{"key": "val"}}},
		Skills:    []adminCreateSkillRequest{{Slug: "combo-skill", Source: model.SkillSourceEnterprise}},
	}
	presets, err := resolveAdminCreatePresets(context.Background(), input, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if presets == nil {
		t.Fatal("expected non-nil presets")
	}
	if len(presets.Models) != 1 || presets.Models[0].ID != am.ID {
		t.Errorf("models mismatch: %+v", presets.Models)
	}
	if len(presets.Channels) != 1 || presets.Channels[0].Channel != "openclaw-weixin" {
		t.Errorf("channels mismatch: %+v", presets.Channels)
	}
	if len(presets.Skills) != 1 || presets.Skills[0].Enterprise.ID != sk.ID {
		t.Errorf("skills mismatch: %+v", presets.Skills)
	}
}

// ─── UT-21: full admin create success with mocked CVM ─────────────────────

// setupAdminCreateFullSuccessDB creates a file-backed DB with all tables
// needed for a full admin create flow, seeds images/config/users, and
// installs CVM/SG/VPC hooks. Returns a cleanup function.
func setupAdminCreateFullSuccessDB(t *testing.T) (cleanup func(), dbPath string) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-admin-create-*.db")
	if err != nil {
		t.Fatalf("create temp DB: %v", err)
	}
	dbPath = tmpFile.Name()
	tmpFile.Close()

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
		&model.UserGroupMember{},
		&model.InstanceModel{}, &model.AIChannel{},
		&model.PluginInstallation{}, &model.Skill{}, &model.SkillBundle{},
		&model.BundleSkill{}, &model.OpenClawRoleSkill{},
		&model.MemoryPlanGroupPolicy{},
		&model.ModelVisibilityGroup{}, &model.SkillVisibilityGroup{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	origDB := useDBForTestWithSafeRestore(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	AdminToken = "test-admin-token"

	// Seed image
	img := &model.AIImage{
		ImageId: "img-openclaw", ImageName: "openclaw",
		ImageType: "PRIVATE_IMAGE", AgentType: model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0", Enabled: true,
	}
	if err := model.DB(context.Background()).Create(img).Error; err != nil {
		t.Fatalf("seed image: %v", err)
	}

	// Seed site config with VPC + subnet
	cfg := &model.SiteConfig{
		CVMTemplate: `{"InstanceType":"S5.SMALL1","InstanceChargeType":"POSTPAID_BY_HOUR"}`,
		VpcId:       "vpc-test",
		SubnetIds:   `{"ap-guangzhou-3":["subnet-single"]}`,
	}
	if err := model.DB(context.Background()).Create(cfg).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	// Seed user
	user := &model.User{Username: "u-admincreate", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// Seed default RuleSet
	rs := &model.RuleSet{
		Name: model.DefaultRuleSetName, Description: "test",
		Rules: "[]", Version: 1, UserGroupIDs: "[]", IsDefault: true,
	}
	if err := model.DB(context.Background()).Create(rs).Error; err != nil {
		t.Fatalf("seed rule set: %v", err)
	}
	defaultRuleSetCache = sync.Map{}

	// Install hooks
	origValidate := validateGlobalVpcAndSubnetsFn
	origSelect := selectSGForNewInstanceFn
	origValidateResourceConfig := validateCreateResourceConfigFn
	origCVMRegion := CVMRegion
	validateGlobalVpcAndSubnetsFn = func(ctx context.Context, _ string, _ map[string][]string) error { return nil }
	validateCreateResourceConfigFn = func(_ context.Context, _, _, _ string) error { return nil }
	CVMRegion = "ap-guangzhou"
	selectSGForNewInstanceFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "sg-mock-admin", false, nil
	}

	cleanup = func() {
		origDB()
		Store = origStore
		validateGlobalVpcAndSubnetsFn = origValidate
		selectSGForNewInstanceFn = origSelect
		validateCreateResourceConfigFn = origValidateResourceConfig
		CVMRegion = origCVMRegion
		defaultRuleSetCache = sync.Map{}
		os.Remove(dbPath)
	}
	return cleanup, dbPath
}

func TestAdminCreate_OpenClaw328FallbackRejectedBeforeCVM(t *testing.T) {
	cleanup, _ := setupAdminCreateFullSuccessDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).Model(&model.AIImage{}).
		Where("agent_type = ?", model.AgentTypeOpenClaw).
		Update("agent_version", "3.28.7").Error; err != nil {
		t.Fatalf("set image version: %v", err)
	}
	primary := model.AIModel{
		ModelID: "v328-primary", ModelName: "3.28 Primary",
		Enabled: true, Visible: true, VisibilityType: model.VisibilityAll,
	}
	fallback := model.AIModel{
		ModelID: "v328-fallback", ModelName: "3.28 Fallback",
		Enabled: true, Visible: true, VisibilityType: model.VisibilityAll,
	}
	if err := model.DB(context.Background()).Create(&primary).Error; err != nil {
		t.Fatalf("create primary: %v", err)
	}
	if err := model.DB(context.Background()).Create(&fallback).Error; err != nil {
		t.Fatalf("create fallback: %v", err)
	}
	var owner model.User
	if err := model.DB(context.Background()).Where("username = ?", "u-admincreate").First(&owner).Error; err != nil {
		t.Fatalf("load owner: %v", err)
	}

	originalCVMClient := NewCVMClient
	cvmCalled := false
	NewCVMClient = func(context.Context) (*cvm.Client, error) {
		cvmCalled = true
		return nil, errors.New("CVM must not be called for unsupported fallback")
	}
	defer func() { NewCVMClient = originalCVMClient }()

	req := adminCreateJSONReq(t, map[string]any{
		"user_id":    owner.ID,
		"name":       "admin-328-fallback",
		"agent_type": model.AgentTypeOpenClaw,
		"models": map[string]any{
			"primary":   primary.ID,
			"fallbacks": []uint{fallback.ID},
		},
	})
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", rr.Code, rr.Body.String())
	}
	if cvmCalled {
		t.Fatal("CVM client was called despite unsupported 3.28 fallback")
	}
	var instanceCount int64
	if err := model.DB(context.Background()).Model(&model.Instance{}).Count(&instanceCount).Error; err != nil {
		t.Fatalf("count instances: %v", err)
	}
	if instanceCount != 0 {
		t.Fatalf("created %d instance rows before rejecting fallback", instanceCount)
	}
}

func TestAdminCreate_FullSuccess(t *testing.T) {
	cleanup, _ := setupAdminCreateFullSuccessDB(t)
	defer cleanup()
	origModelPoll := injectDefaultModelPollInterval
	origModelMaxWait := injectDefaultModelMaxWait
	origChannelPoll := channelPresetPollInterval
	origChannelMaxWait := channelPresetMaxWait
	injectDefaultModelPollInterval = time.Millisecond
	injectDefaultModelMaxWait = 2 * time.Second
	channelPresetPollInterval = time.Millisecond
	channelPresetMaxWait = 2 * time.Second
	defer func() {
		injectDefaultModelPollInterval = origModelPoll
		injectDefaultModelMaxWait = origModelMaxWait
		channelPresetPollInterval = origChannelPoll
		channelPresetMaxWait = origChannelMaxWait
	}()

	origInjectRunner := injectModelScriptRunner
	origChannelRunner := channelScriptRunner
	origSkillRunner := skillScriptRunner
	skillRuns := make(chan struct {
		script string
		params map[string]string
	}, 1)
	injectModelScriptRunner = func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
		return `{"ok":true}`, nil
	}
	channelScriptRunner = func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
		return `{"ok":true}`, nil
	}
	skillScriptRunner = func(_ context.Context, _ string, script string, _ uint64, _ string, _ func(string), params map[string]string) (string, error) {
		copied := make(map[string]string, len(params))
		for key, value := range params {
			copied[key] = value
		}
		skillRuns <- struct {
			script string
			params map[string]string
		}{script: script, params: copied}
		return `{"ok":true}`, nil
	}
	defer func() {
		injectModelScriptRunner = origInjectRunner
		channelScriptRunner = origChannelRunner
		skillScriptRunner = origSkillRunner
	}()

	db := model.DB(context.Background())

	// Seed model, channel, skill for presets
	am := &model.AIModel{
		ModelName: "m1", ModelID: "m1", Enabled: true, Visible: true, VisibilityType: "all",
	}
	db.Create(am)

	enabled := true
	ac := &model.AIChannel{
		ChannelID: "openclaw-weixin", Name: "weixin",
		Enabled: &enabled, VisibilityType: "all",
	}
	db.Create(ac)

	// Start mock CVM
	ts, capturedRunInstances := mockCVMRunInstancesCaptureServer(t)
	defer ts.Close()

	origCVM := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return newMockCVMClientWithServer(t, ts.URL), nil
	}
	defer func() { NewCVMClient = origCVM }()

	// Get the user ID
	var user model.User
	db.Where("username = ?", "u-admincreate").First(&user)
	if err := db.Create(&model.Tag{
		TagKey:         "managed-by",
		TagValue:       "clawpro",
		VisibilityType: model.VisibilityAll,
	}).Error; err != nil {
		t.Fatalf("seed default tag: %v", err)
	}

	body := map[string]any{
		"user_id":    user.ID,
		"name":       "admin-full-test",
		"agent_type": "openclaw",
		"tags": []map[string]string{
			{"key": "env", "value": "staging"},
			{"key": "business", "value": "two"},
		},
		"models":   map[string]any{"primary": am.ID},
		"channels": []map[string]any{{"channel": "openclaw-weixin", "config": map[string]string{"secret_key": "s3cret"}}},
		"skills":   []map[string]any{{"slug": "agently-mail"}},
	}
	req := withDomainCtx(adminCreateJSONReq(t, body), "https://hatchery.example.com")
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	resp := parseJSONResp(t, rr)
	if v, ok := resp["ok"]; !ok || v != true {
		t.Errorf("response missing ok:true: %v", resp)
	}

	if _, exists := resp["id"]; exists {
		t.Fatalf("admin create response must not invent a database id contract: %v", resp["id"])
	}

	instanceID, ok := resp["instance_id"].(string)
	if !ok || instanceID != "ins-mock-writectx" {
		t.Errorf("instance_id mismatch: got %q", instanceID)
	}

	if _, exists := resp["preset"]; exists {
		t.Fatalf("response must not contain unverifiable preset status: %v", resp["preset"])
	}
	assertCapturedInstanceTags(t, capturedRunInstances, []createInstanceTag{
		{Key: "env", Value: "staging"},
		{Key: "business", Value: "two"},
		{Key: "managed-by", Value: "clawpro"},
	})

	// Verify DB state
	var inst model.Instance
	if err := db.Where("name = ?", "admin-full-test").First(&inst).Error; err != nil {
		t.Fatalf("instance not in DB: %v", err)
	}
	if inst.UserID != user.ID {
		t.Errorf("instance user_id: %d, want %d", inst.UserID, user.ID)
	}
	if inst.InstanceId != "ins-mock-writectx" {
		t.Errorf("instance_id: %s", inst.InstanceId)
	}
	if inst.SecurityGroupId != "sg-mock-admin" {
		t.Errorf("security_group_id: %s", inst.SecurityGroupId)
	}
	if got := model.ParseTagItems(inst.CVMTagsJSON); fmt.Sprint(got) != fmt.Sprint([]model.TagItem{
		{Key: "env", Value: "staging"},
		{Key: "business", Value: "two"},
		{Key: "managed-by", Value: "clawpro"},
	}) {
		t.Errorf("cvm_tags_json should cache merged tags, got %+v", got)
	}
	if inst.CurrentOperation != model.OpCreate {
		t.Errorf("current_operation: %s", inst.CurrentOperation)
	}
	if inst.ProxyToken == nil || *inst.ProxyToken == "" {
		t.Error("proxy_token should not be empty")
	}

	// Verify channel secrets NEVER appear in the HTTP response body
	respBody := rr.Body.String()
	if strings.Contains(respBody, "s3cret") {
		t.Error("channel secret leaked in HTTP response body")
	}
	if strings.Contains(respBody, "secret_key") {
		t.Error("channel secret key leaked in HTTP response body")
	}

	// Release and join every detached initialization worker before restoring
	// package globals or the test DB. Otherwise this test can leak a model
	// failure notification into a later test's notification stub.
	if err := db.Model(&inst).Updates(map[string]interface{}{"agent_ready": 1, "runtime_user": "root"}).Error; err != nil {
		t.Fatalf("release initialization workers: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for hasPendingGatewayRestartTasks(inst.ID) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if hasPendingGatewayRestartTasks(inst.ID) {
		t.Fatal("timed out waiting for instance initialization workers")
	}

	select {
	case run := <-skillRuns:
		if run.script != "add_skill.sh" {
			t.Fatalf("source omitted must use public add-skill script, got %q", run.script)
		}
		if run.params["skill_name"] != "agently-mail" {
			t.Fatalf("public skill_name = %q, want agently-mail", run.params["skill_name"])
		}
	default:
		t.Fatal("source omitted did not execute the public add-skill path")
	}

	// The model preset is applied only after AgentReady, exactly like a user
	// invoking add-model after creation.
	var im model.InstanceModel
	if err := db.Where("instance_id = ? AND ai_model_id = ?", inst.ID, am.ID).First(&im).Error; err != nil {
		t.Errorf("no InstanceModel for primary: %v", err)
	} else if im.Role != model.ModelRolePrimary {
		t.Errorf("model role: %s, want %s", im.Role, model.ModelRolePrimary)
	}
}

func TestAuditRule_AdminCreateInstanceRegistered(t *testing.T) {
	const path = "/admin/instances/create"

	rule, ok := auditRules[path]
	if !ok {
		t.Fatalf("auditRules[%q] is not registered; admin instance creation would bypass audit logging", path)
	}
	if want := "instance_admin_create"; rule.Action != want {
		t.Errorf("auditRules[%q].Action = %q, want %q", path, rule.Action, want)
	}
	if want := "instance"; rule.Resource != want {
		t.Errorf("auditRules[%q].Resource = %q, want %q", path, rule.Resource, want)
	}
}

// ─── i18n smoke checks ──────────────────────────────────────────────────────

func TestAdminCreate_i18nMessagesRegistered(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	ctx := context.Background()
	keys := []i18n.Key{
		i18n.MsgUserNotFound,
		i18n.MsgBadRequestParamRequired,
		i18n.MsgAdminCreateGroupInvalid,
		i18n.MsgAdminCreateModelDuplicate,
		i18n.MsgAdminCreateModelUnavailable,
		i18n.MsgAdminCreateChannelDuplicate,
		i18n.MsgAdminCreateChannelConfigInvalid,
		i18n.MsgAdminCreateSkillUnavailable,
		i18n.MsgAdminCreateSkillNotVisible,
		i18n.MsgAdminCreateSkillSourceUnsupported,
		i18n.MsgAgentTypeDoNotSupportModelConfigWithDetail,
		i18n.MsgChannelNotSupportedWithDetail,
		i18n.MsgAgentTypeDoNotSupportSkillWithDetail,
	}
	for _, key := range keys {
		msg := i18n.T(ctx, key)
		if msg == "" {
			t.Errorf("i18n key %q not translated (got empty)", key.String())
		}
	}
}

func TestAdminCreateI18n_EnglishTranslations(t *testing.T) {
	p := message.NewPrinter(language.English)
	tests := []struct {
		key  i18n.Key
		args []any
		want string
	}{
		{
			key:  i18n.MsgAdminCreateGroupInvalid,
			want: "The selected group does not belong to the target user or is no longer valid",
		},
		{
			key:  i18n.MsgAdminCreateModelDuplicate,
			want: "Initial models must not contain duplicates",
		},
		{
			key:  i18n.MsgAdminCreateChannelDuplicate,
			want: "Initial channels must not contain duplicates",
		},
		{
			key:  i18n.MsgAdminCreateChannelConfigInvalid,
			want: "Channel configuration keys and values must not be empty",
		},
		{
			key:  i18n.MsgAdminCreateSkillSourceUnsupported,
			want: "Initial skill source must be public or enterprise",
		},
		{
			key:  i18n.MsgAdminCreateModelUnavailable,
			args: []any{42},
			want: "Model 42 does not exist, is disabled, or is not visible to the target group",
		},
		{
			key:  i18n.MsgAdminCreateSkillUnavailable,
			args: []any{"my-skill"},
			want: "Skill my-skill does not exist",
		},
		{
			key:  i18n.MsgAdminCreateSkillNotVisible,
			args: []any{"hidden-skill"},
			want: "Skill hidden-skill is not visible to the target group",
		},
	}
	for _, tt := range tests {
		t.Run(tt.key.String(), func(t *testing.T) {
			if got := p.Sprintf(tt.key.String(), tt.args...); got != tt.want {
				t.Errorf("English translation mismatch: got %q, want %q", got, tt.want)
			}
		})
	}
}

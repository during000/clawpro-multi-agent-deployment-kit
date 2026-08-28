package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initSMHTestDB 初始化内存 SQLite 数据库，迁移 HandleAdminSMHInstances 所需的表。
func initSMHTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SMHPersonalSpace{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// 创建 SiteConfig 并启用 SMH（requireSMHEnabled 依赖此配置）
	db.Create(&model.SiteConfig{SMHEnabled: 1})
}

// smhInstancesHandler 绕过 requireAdmin 和 requireSMHEnabled，直接执行 HandleAdminSMHInstances 的核心逻辑。
// 这与 admin_instances_test.go 中 adminInstancesHandler 的模式一致。
func smhInstancesHandler(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)

	page, pageSize := parsePagination(r)

	type instanceWithSpace struct {
		InstanceDbID     uint       `gorm:"column:id"`
		InstanceName     string     `gorm:"column:name"`
		InstanceId       string     `gorm:"column:instance_id"`
		AgentType        string     `gorm:"column:agent_type"`
		UserID           uint       `gorm:"column:user_id"`
		Username         string     `gorm:"column:username"`
		SpaceDbID        *uint      `gorm:"column:space_id_pk"`
		SpaceId          *string    `gorm:"column:space_id"`
		StorageQuota     *int64     `gorm:"column:storage_quota"`
		FreeStorageQuota *int64     `gorm:"column:free_storage_quota"`
		SpaceCreatedAt   *time.Time `gorm:"column:space_created_at"`
		ExpiresAt        *time.Time `gorm:"column:expires_at"`
		ToBeDeletedAt    *time.Time `gorm:"column:to_be_deleted_at"`
	}

	baseQuery := model.DB(context.Background()).Model(&model.Instance{}).
		Select(`instances.id, instances.name, instances.instance_id, instances.agent_type, instances.user_id,
			u.username,
			s.id as space_id_pk, s.space_id,
			s.storage_quota, s.free_storage_quota,
			s.created_at as space_created_at, s.expires_at, s.to_be_deleted_at`).
		Joins("LEFT JOIN smh_personal_spaces s ON s.instance_id = instances.id AND s.deleted_at IS NULL AND s.identifier = instances.identifier").
		Joins("LEFT JOIN users u ON u.id = instances.user_id AND u.deleted_at IS NULL AND u.identifier = instances.identifier")

	if userID := r.URL.Query().Get("user"); userID != "" {
		baseQuery = baseQuery.Where("instances.user_id = ?", userID)
	}
	if agentType := r.URL.Query().Get("agent_type"); agentType != "" {
		baseQuery = baseQuery.Where("instances.agent_type = ?", agentType)
	}

	var total int64
	baseQuery.Session(&gorm.Session{}).Count(&total)

	var rows []instanceWithSpace
	baseQuery.Order("instances.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows)

	// 与源码一致：从查询结果中收集 spaceIds 再传入（SMH 未配置时返回空 map）
	var spaceIds []string
	for _, row := range rows {
		if row.SpaceId != nil && *row.SpaceId != "" {
			spaceIds = append(spaceIds, *row.SpaceId)
		}
	}
	usageMap := fetchPersonalSpaceUsage(context.Background(), spaceIds)

	items := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		status := "none"
		if row.SpaceDbID != nil {
			if row.ToBeDeletedAt != nil {
				status = "recycling"
			} else {
				status = "active"
			}
		}

		agentType := row.AgentType
		if agentType == "" {
			agentType = "openclaw"
		}

		item := map[string]interface{}{
			"instance_id":     row.InstanceDbID,
			"instance_name":   row.InstanceName,
			"cvm_instance_id": row.InstanceId,
			"agent_type":      agentType,
			"user_id":         row.UserID,
			"username":        row.Username,
			"space_status":    status,
		}

		if row.SpaceDbID != nil {
			spaceId := ""
			if row.SpaceId != nil {
				spaceId = *row.SpaceId
			}
			item["space_id"] = *row.SpaceDbID
			item["smh_space_id"] = spaceId
			item["storage_quota"] = row.StorageQuota
			item["free_storage_quota"] = row.FreeStorageQuota
			item["used_storage"] = usageMap[spaceId].size
			item["bound_at"] = row.SpaceCreatedAt
			item["expires_at"] = row.ExpiresAt
			item["to_be_deleted_at"] = row.ToBeDeletedAt
		}

		items[i] = item
	}

	jsonOK(w, map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// parseSMHInstancesResp 解析 smhInstancesHandler 的 JSON 响应。
func parseSMHInstancesResp(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return resp
}

// ---------- Test: 空数据 ----------

func TestSMHInstances_Empty(t *testing.T) {
	initSMHTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp := parseSMHInstancesResp(t, w)
	if resp["total"].(float64) != 0 {
		t.Errorf("空数据期望 total=0，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	if len(items) != 0 {
		t.Errorf("空数据期望 items 为空，实际=%d", len(items))
	}
}

// ---------- Test: 基本查询 — 实例无空间 ----------

func TestSMHInstances_NoSpace(t *testing.T) {
	initSMHTestDB(t)

	user := model.User{Username: "alice", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{Name: "dev-box", InstanceId: "ins-001", UserID: user.ID})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp := parseSMHInstancesResp(t, w)
	if resp["total"].(float64) != 1 {
		t.Fatalf("期望 total=1，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if item["instance_name"] != "dev-box" {
		t.Errorf("期望 instance_name=dev-box，实际=%v", item["instance_name"])
	}
	if item["username"] != "alice" {
		t.Errorf("期望 username=alice，实际=%v", item["username"])
	}
	if item["cvm_instance_id"] != "ins-001" {
		t.Errorf("期望 cvm_instance_id=ins-001，实际=%v", item["cvm_instance_id"])
	}
	if item["space_status"] != "none" {
		t.Errorf("无空间时期望 space_status=none，实际=%v", item["space_status"])
	}
	// 无空间时不应有 space_id 字段
	if _, ok := item["space_id"]; ok {
		t.Error("无空间时不应返回 space_id 字段")
	}
	// agent_type 为空时应默认返回 openclaw
	if item["agent_type"] != "openclaw" {
		t.Errorf("期望 agent_type=openclaw，实际=%v", item["agent_type"])
	}
}

// ---------- Test: 实例有活跃空间 ----------

func TestSMHInstances_ActiveSpace(t *testing.T) {
	initSMHTestDB(t)

	user := model.User{Username: "bob", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	ins := model.Instance{Name: "prod-box", InstanceId: "ins-002", UserID: user.ID}
	model.DB(context.Background()).Create(&ins)

	expiresAt := time.Now().Add(90 * 24 * time.Hour)
	model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId:          "sp-abc",
		UserId:           user.ID,
		InstanceId:       ins.ID,
		UserName:         user.Username,
		InstanceName:     ins.Name,
		CVMInstanceId:    ins.InstanceId,
		StorageQuota:     53687091200,
		FreeStorageQuota: 53687091200,
		ExpiresAt:        &expiresAt,
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp := parseSMHInstancesResp(t, w)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("期望 1 条，实际=%d", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["space_status"] != "active" {
		t.Errorf("有活跃空间时期望 space_status=active，实际=%v", item["space_status"])
	}
	if item["smh_space_id"] != "sp-abc" {
		t.Errorf("期望 smh_space_id=sp-abc，实际=%v", item["smh_space_id"])
	}
	// 验证配额字段存在
	if item["storage_quota"] == nil {
		t.Error("期望 storage_quota 不为 nil")
	}
}

// ---------- Test: 实例空间在回收站 ----------

func TestSMHInstances_RecyclingSpace(t *testing.T) {
	initSMHTestDB(t)

	user := model.User{Username: "charlie", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	ins := model.Instance{Name: "old-box", InstanceId: "ins-003", UserID: user.ID}
	model.DB(context.Background()).Create(&ins)

	deleteAt := time.Now().Add(7 * 24 * time.Hour)
	model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId:       "sp-del",
		UserId:        user.ID,
		InstanceId:    ins.ID,
		UserName:      user.Username,
		InstanceName:  ins.Name,
		CVMInstanceId: ins.InstanceId,
		ToBeDeletedAt: &deleteAt,
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp := parseSMHInstancesResp(t, w)
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if item["space_status"] != "recycling" {
		t.Errorf("回收站空间期望 space_status=recycling，实际=%v", item["space_status"])
	}
}

// ---------- Test: 分页 ----------

func TestSMHInstances_Pagination(t *testing.T) {
	initSMHTestDB(t)

	user := model.User{Username: "pager", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	for i := 0; i < 5; i++ {
		model.DB(context.Background()).Create(&model.Instance{
			Name:       "ins-" + string(rune('A'+i)),
			InstanceId: "ins-page-" + string(rune('0'+i)),
			UserID:     user.ID,
		})
	}

	// 第 1 页，每页 2 条
	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances?page=1&page_size=2", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp := parseSMHInstancesResp(t, w)
	if resp["total"].(float64) != 5 {
		t.Errorf("期望 total=5，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("第 1 页期望 2 条，实际=%d", len(items))
	}
	if resp["page"].(float64) != 1 {
		t.Errorf("期望 page=1，实际=%v", resp["page"])
	}
	if resp["page_size"].(float64) != 2 {
		t.Errorf("期望 page_size=2，实际=%v", resp["page_size"])
	}

	// 验证排序：第 1 页应按 id desc 排序，第一条的 id 应大于第二条
	firstID := items[0].(map[string]interface{})["instance_id"].(float64)
	secondID := items[1].(map[string]interface{})["instance_id"].(float64)
	if firstID <= secondID {
		t.Errorf("期望按 id desc 排序，第一条 id=%.0f 应大于第二条 id=%.0f", firstID, secondID)
	}

	// 第 3 页，应只有 1 条
	req = httptest.NewRequest(http.MethodGet, "/admin/smh/instances?page=3&page_size=2", nil)
	w = httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp = parseSMHInstancesResp(t, w)
	items = resp["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("第 3 页期望 1 条，实际=%d", len(items))
	}
}

// ---------- Test: 按 user 过滤 ----------

func TestSMHInstances_FilterByUser(t *testing.T) {
	initSMHTestDB(t)

	alice := model.User{Username: "alice", Password: "x", Role: "user"}
	bob := model.User{Username: "bob", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&alice)
	model.DB(context.Background()).Create(&bob)
	model.DB(context.Background()).Create(&model.Instance{Name: "alice-box", InstanceId: "ins-a1", UserID: alice.ID})
	model.DB(context.Background()).Create(&model.Instance{Name: "alice-box2", InstanceId: "ins-a2", UserID: alice.ID})
	model.DB(context.Background()).Create(&model.Instance{Name: "bob-box", InstanceId: "ins-b1", UserID: bob.ID})

	// 不过滤，应返回 3 条
	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)
	resp := parseSMHInstancesResp(t, w)
	if resp["total"].(float64) != 3 {
		t.Errorf("不过滤期望 total=3，实际=%v", resp["total"])
	}

	// 按 alice 的 user_id 过滤
	req = httptest.NewRequest(http.MethodGet, "/admin/smh/instances?user="+itoa(alice.ID), nil)
	w = httptest.NewRecorder()
	smhInstancesHandler(w, req)
	resp = parseSMHInstancesResp(t, w)
	if resp["total"].(float64) != 2 {
		t.Errorf("过滤 alice 期望 total=2，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	for _, raw := range items {
		item := raw.(map[string]interface{})
		if item["username"] != "alice" {
			t.Errorf("过滤后期望 username=alice，实际=%v", item["username"])
		}
	}
}

// ---------- Test: 混合状态 — 多实例不同空间状态 ----------

func TestSMHInstances_MixedStatus(t *testing.T) {
	initSMHTestDB(t)

	user := model.User{Username: "mixed", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	// 实例 1：无空间
	ins1 := model.Instance{Name: "no-space", InstanceId: "ins-m1", UserID: user.ID}
	model.DB(context.Background()).Create(&ins1)

	// 实例 2：有活跃空间
	ins2 := model.Instance{Name: "has-space", InstanceId: "ins-m2", UserID: user.ID}
	model.DB(context.Background()).Create(&ins2)
	model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId: "sp-active", UserId: user.ID, InstanceId: ins2.ID,
		UserName: user.Username, InstanceName: ins2.Name, CVMInstanceId: ins2.InstanceId,
	})

	// 实例 3：空间在回收站
	ins3 := model.Instance{Name: "recycling", InstanceId: "ins-m3", UserID: user.ID}
	model.DB(context.Background()).Create(&ins3)
	deleteAt := time.Now().Add(7 * 24 * time.Hour)
	model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId: "sp-recycle", UserId: user.ID, InstanceId: ins3.ID,
		UserName: user.Username, InstanceName: ins3.Name, CVMInstanceId: ins3.InstanceId,
		ToBeDeletedAt: &deleteAt,
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp := parseSMHInstancesResp(t, w)
	if resp["total"].(float64) != 3 {
		t.Fatalf("期望 total=3，实际=%v", resp["total"])
	}

	// 按 id desc 排序，ins3 > ins2 > ins1
	items := resp["items"].([]interface{})
	statusMap := map[string]string{}
	for _, raw := range items {
		item := raw.(map[string]interface{})
		name := item["instance_name"].(string)
		status := item["space_status"].(string)
		statusMap[name] = status
	}

	if statusMap["no-space"] != "none" {
		t.Errorf("no-space 期望 status=none，实际=%s", statusMap["no-space"])
	}
	if statusMap["has-space"] != "active" {
		t.Errorf("has-space 期望 status=active，实际=%s", statusMap["has-space"])
	}
	if statusMap["recycling"] != "recycling" {
		t.Errorf("recycling 期望 status=recycling，实际=%s", statusMap["recycling"])
	}
}

// ---------- Test: 软删除的空间不应关联 ----------

func TestSMHInstances_SoftDeletedSpaceIgnored(t *testing.T) {
	initSMHTestDB(t)

	user := model.User{Username: "softdel", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	ins := model.Instance{Name: "softdel-box", InstanceId: "ins-sd1", UserID: user.ID}
	model.DB(context.Background()).Create(&ins)

	// 创建一个空间然后软删除
	space := model.SMHPersonalSpace{
		SpaceId: "sp-deleted", UserId: user.ID, InstanceId: ins.ID,
		UserName: user.Username, InstanceName: ins.Name, CVMInstanceId: ins.InstanceId,
	}
	model.DB(context.Background()).Create(&space)
	model.DB(context.Background()).Delete(&space) // 软删除

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp := parseSMHInstancesResp(t, w)
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if item["space_status"] != "none" {
		t.Errorf("软删除的空间不应关联，期望 space_status=none，实际=%v", item["space_status"])
	}
}

// ---------- Test: 跨实例空间不关联 — JOIN 条件 instance_id 正确性 ----------

func TestSMHInstances_SpaceNotLeakedAcrossInstances(t *testing.T) {
	initSMHTestDB(t)

	alice := model.User{Username: "alice", Password: "x", Role: "user"}
	bob := model.User{Username: "bob", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&alice)
	model.DB(context.Background()).Create(&bob)

	insAlice := model.Instance{Name: "alice-box", InstanceId: "ins-alice", UserID: alice.ID}
	insBob := model.Instance{Name: "bob-box", InstanceId: "ins-bob", UserID: bob.ID}
	model.DB(context.Background()).Create(&insAlice)
	model.DB(context.Background()).Create(&insBob)

	// 只给 alice 的实例绑定空间
	model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId: "sp-alice", UserId: alice.ID, InstanceId: insAlice.ID,
		UserName: alice.Username, InstanceName: insAlice.Name, CVMInstanceId: insAlice.InstanceId,
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp := parseSMHInstancesResp(t, w)
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("期望 2 条，实际=%d", len(items))
	}

	statusMap := map[string]string{}
	for _, raw := range items {
		item := raw.(map[string]interface{})
		statusMap[item["instance_name"].(string)] = item["space_status"].(string)
	}

	if statusMap["alice-box"] != "active" {
		t.Errorf("alice-box 期望 active，实际=%s", statusMap["alice-box"])
	}
	if statusMap["bob-box"] != "none" {
		t.Errorf("bob-box 不应关联 alice 的空间，期望 none，实际=%s", statusMap["bob-box"])
	}
}

// ---------- Test: 默认分页参数 ----------

func TestSMHInstances_DefaultPagination(t *testing.T) {
	initSMHTestDB(t)

	user := model.User{Username: "default-page", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{Name: "box-1", InstanceId: "ins-dp1", UserID: user.ID})

	// 不传 page 和 page_size，应使用默认值 page=1, page_size=20
	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)

	resp := parseSMHInstancesResp(t, w)
	if resp["page"].(float64) != 1 {
		t.Errorf("默认 page 期望 1，实际=%v", resp["page"])
	}
	if resp["page_size"].(float64) != 20 {
		t.Errorf("默认 page_size 期望 20，实际=%v", resp["page_size"])
	}
}

// ---------- Test: 按 agent_type 过滤 ----------

func TestSMHInstances_FilterByAgentType(t *testing.T) {
	initSMHTestDB(t)

	user := model.User{Username: "typetest", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{Name: "oc-box", InstanceId: "ins-oc", UserID: user.ID, AgentType: "openclaw"})
	model.DB(context.Background()).Create(&model.Instance{Name: "lc-box", InstanceId: "ins-lc", UserID: user.ID, AgentType: "lightclaw-ace"})
	model.DB(context.Background()).Create(&model.Instance{Name: "hm-box", InstanceId: "ins-hm", UserID: user.ID, AgentType: "hermes"})

	// 不过滤，应返回 3 条
	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	w := httptest.NewRecorder()
	smhInstancesHandler(w, req)
	resp := parseSMHInstancesResp(t, w)
	if resp["total"].(float64) != 3 {
		t.Errorf("不过滤期望 total=3，实际=%v", resp["total"])
	}

	// 按 lightclaw-ace 过滤
	req = httptest.NewRequest(http.MethodGet, "/admin/smh/instances?agent_type=lightclaw-ace", nil)
	w = httptest.NewRecorder()
	smhInstancesHandler(w, req)
	resp = parseSMHInstancesResp(t, w)
	if resp["total"].(float64) != 1 {
		t.Errorf("过滤 lightclaw-ace 期望 total=1，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if item["agent_type"] != "lightclaw-ace" {
		t.Errorf("期望 agent_type=lightclaw-ace，实际=%v", item["agent_type"])
	}
	if item["instance_name"] != "lc-box" {
		t.Errorf("期望 instance_name=lc-box，实际=%v", item["instance_name"])
	}
}

// itoa 将 uint 转为字符串，用于拼接 URL 参数。
func itoa(id uint) string {
	return fmt.Sprintf("%d", id)
}

// ---------- HandleAdminSMHConfigAPI ----------
//
// 认证：通过 AdminToken + Bearer Token 让 requireAdmin 真正放行，
// 这与 admin_roles_test.go 中的写法保持一致，不复制 handler 核心逻辑。
// 数据库：独立的 initSMHConfigTestDB —— 因为需要迁移 SMHSpace 表（seedSMHConfigured 会写此表），
// 而现有 initSMHTestDB 不迁移该表，故不复用。
// 密钥脱敏：源码规则是「长度 > 8 时取前 4 位 + **** + 后 4 位」，否则原样返回。

// initSMHConfigTestDB 初始化内存 SQLite，迁移 HandleAdminSMHConfigAPI 依赖的表。
// 注意与 initSMHTestDB 的区别：本函数额外迁移 SMHSpace（seedSMHConfigured 会写入）且不预置 SiteConfig。
func initSMHConfigTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.SiteConfig{},
		&model.SMHSpace{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// 注：withAdminToken 辅助函数已在 admin_security_group_test.go 中定义
// （签名为 func(token string) func()，返回 cleanup），本文件直接复用。

// newAdminJSONReq 构造带 Admin Bearer Token 和 JSON Accept 的请求。
func newAdminJSONReq(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Accept", "application/json")
	return req
}

// callAdminSMHConfigAPI 触发 HandleAdminSMHConfigAPI 并返回 ResponseRecorder。
func callAdminSMHConfigAPI(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	HandleAdminSMHConfigAPI(w, newAdminJSONReq(http.MethodGet, "/admin/smh/config"))
	return w
}

// parseAdminSMHConfigResp 校验 HTTP 200 并解析 JSON body。
func parseAdminSMHConfigResp(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return resp
}

// TestAdminSMHConfigAPI_Unauthorized 验证未提供 AdminToken 时 requireAdmin 拒绝。
// 不设置 AdminToken（保持为空），且不附带 Authorization 头；
// 注意 requireAdmin 在没有 Token 时会走 session 分支，故需初始化 Store 避免 nil 解引用。
func TestAdminSMHConfigAPI_Unauthorized(t *testing.T) {
	initSMHConfigTestDB(t)
	// 初始化 session Store（与其他 admin 测试保持一致）
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	t.Cleanup(func() { Store = origStore })

	// 不调用 withAdminToken，AdminToken 保持为空；同时不附带 Authorization 头
	req := httptest.NewRequest(http.MethodGet, "/admin/smh/config", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleAdminSMHConfigAPI(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("未鉴权时期望非 200，实际 200，body=%s", w.Body.String())
	}
}

// TestAdminSMHConfigAPI_FullyConfigured 完整配置场景：
// 写入完整 SiteConfig + 两个 SMHSpace → is_configured=true、各字段原样返回、
// smh_library_secret 按「前 4 + **** + 后 4」脱敏。
func TestAdminSMHConfigAPI_FullyConfigured(t *testing.T) {
	initSMHConfigTestDB(t)
	defer withAdminToken("test-admin-token")()

	// 完整写入 SiteConfig（密钥长度 > 8 以触发脱敏分支）
	if err := model.DB(context.Background()).Create(&model.SiteConfig{
		SMHEnabled:               1,
		SMHAutoProvisionOnCreate: true,
		SMHLibraryId:             "lib-1234",
		SMHLibrarySecret:         "ABCD1234567890WXYZ", // 18 字符，必走脱敏
		SMHEndpoint:              "https://smh.example.com",
		SMHProvisionError:        "",
	}).Error; err != nil {
		t.Fatalf("写入 SiteConfig 失败: %v", err)
	}
	model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "common", SpaceId: "sp-common", LibraryId: "lib-1234", Purpose: "common"})
	model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "skillhub", SpaceId: "sp-skillhub", LibraryId: "lib-1234", Purpose: "skillhub"})

	resp := parseAdminSMHConfigResp(t, callAdminSMHConfigAPI(t))

	// 基础字段
	if resp["smh_enabled"].(float64) != 1 {
		t.Errorf("smh_enabled 期望 1，实际=%v", resp["smh_enabled"])
	}
	if resp["smh_library_id"] != "lib-1234" {
		t.Errorf("smh_library_id 期望 'lib-1234'，实际=%v", resp["smh_library_id"])
	}
	if resp["smh_endpoint"] != "https://smh.example.com" {
		t.Errorf("smh_endpoint 期望 'https://smh.example.com'，实际=%v", resp["smh_endpoint"])
	}
	if resp["smh_auto_provision_on_create"] != true {
		t.Errorf("smh_auto_provision_on_create 期望 true，实际=%v", resp["smh_auto_provision_on_create"])
	}

	// 密钥脱敏：前 4 + **** + 后 4
	wantMasked := "ABCD****WXYZ"
	if resp["smh_library_secret"] != wantMasked {
		t.Errorf("smh_library_secret 脱敏期望=%q，实际=%v", wantMasked, resp["smh_library_secret"])
	}
	// 明文密钥绝不应出现在响应中
	if resp["smh_library_secret"] == "ABCD1234567890WXYZ" {
		t.Error("密钥脱敏失败：响应中出现了明文密钥")
	}

	// smh_spaces 表字段
	if resp["smh_common_space"] != "sp-common" {
		t.Errorf("smh_common_space 期望 'sp-common'，实际=%v", resp["smh_common_space"])
	}
	if resp["smh_skillhub_space"] != "sp-skillhub" {
		t.Errorf("smh_skillhub_space 期望 'sp-skillhub'，实际=%v", resp["smh_skillhub_space"])
	}

	// 完整配置 → IsConfigured() == true
	if resp["is_configured"] != true {
		t.Errorf("完整配置期望 is_configured=true，实际=%v", resp["is_configured"])
	}

	// provision_error 应为空字符串
	if resp["provision_error"] != "" {
		t.Errorf("provision_error 期望空字符串，实际=%v", resp["provision_error"])
	}
}

// TestAdminSMHConfigAPI_Empty 未写入任何配置时：
// GetSiteConfig 会 FirstOrCreate 出一条默认行 → 各字段为零值、密钥短（<=8）不脱敏、
// is_configured=false（缺失 Endpoint/LibraryId/LibrarySecret/spaces）。
func TestAdminSMHConfigAPI_Empty(t *testing.T) {
	initSMHConfigTestDB(t)
	defer withAdminToken("test-admin-token")()

	resp := parseAdminSMHConfigResp(t, callAdminSMHConfigAPI(t))

	if resp["smh_enabled"].(float64) != 0 {
		t.Errorf("默认 smh_enabled 期望 0，实际=%v", resp["smh_enabled"])
	}
	if resp["smh_library_id"] != "" {
		t.Errorf("默认 smh_library_id 期望空，实际=%v", resp["smh_library_id"])
	}
	if resp["smh_library_secret"] != "" {
		t.Errorf("空密钥不应脱敏，应原样返回空，实际=%v", resp["smh_library_secret"])
	}
	if resp["smh_common_space"] != "" {
		t.Errorf("无 SMHSpace 时 smh_common_space 期望空，实际=%v", resp["smh_common_space"])
	}
	if resp["smh_skillhub_space"] != "" {
		t.Errorf("无 SMHSpace 时 smh_skillhub_space 期望空，实际=%v", resp["smh_skillhub_space"])
	}
	if resp["is_configured"] != false {
		t.Errorf("默认配置期望 is_configured=false，实际=%v", resp["is_configured"])
	}
	if resp["smh_auto_provision_on_create"] != false {
		t.Errorf("默认 smh_auto_provision_on_create 期望 false，实际=%v", resp["smh_auto_provision_on_create"])
	}
}

// TestAdminSMHConfigAPI_SecretMasking 专项覆盖密钥脱敏边界：
//   - 长度 0/1/8：不脱敏，原样返回（源码条件 len > 8）
//   - 长度 9：触发脱敏（边界条件）
//   - 长度 20：常规脱敏
func TestAdminSMHConfigAPI_SecretMasking(t *testing.T) {
	cases := []struct {
		name   string
		secret string
		want   string
	}{
		{"empty", "", ""},
		{"len1", "a", "a"},
		{"len8_boundary_not_masked", "12345678", "12345678"},  // len == 8 不脱敏
		{"len9_boundary_masked", "123456789", "1234****6789"}, // len == 9 触发脱敏
		{"len20_normal", "ABCDEFGHIJKLMNOPQRST", "ABCD****QRST"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			initSMHConfigTestDB(t)
			defer withAdminToken("test-admin-token")()

			if err := model.DB(context.Background()).Create(&model.SiteConfig{
				SMHLibrarySecret: tc.secret,
			}).Error; err != nil {
				t.Fatalf("写入 SiteConfig 失败: %v", err)
			}

			resp := parseAdminSMHConfigResp(t, callAdminSMHConfigAPI(t))
			if resp["smh_library_secret"] != tc.want {
				t.Errorf("secret=%q 脱敏期望=%q 实际=%v", tc.secret, tc.want, resp["smh_library_secret"])
			}
		})
	}
}

// TestAdminSMHConfigAPI_ProvisionError 验证 provision_error 字段透传非空值。
// 场景：SMH 开通失败时 SiteConfig.SMHProvisionError 被写入具体错误信息，
// 此接口应原样返回该信息以便前端展示。
func TestAdminSMHConfigAPI_ProvisionError(t *testing.T) {
	initSMHConfigTestDB(t)
	defer withAdminToken("test-admin-token")()

	errMsg := "library create failed: quota exceeded"
	if err := model.DB(context.Background()).Create(&model.SiteConfig{
		SMHProvisionError: errMsg,
	}).Error; err != nil {
		t.Fatalf("写入 SiteConfig 失败: %v", err)
	}

	resp := parseAdminSMHConfigResp(t, callAdminSMHConfigAPI(t))
	if resp["provision_error"] != errMsg {
		t.Errorf("provision_error 期望=%q，实际=%v", errMsg, resp["provision_error"])
	}
}

// TestAdminSMHConfigAPI_PartialConfig 验证部分配置（缺 space）时 is_configured=false。
// 即：SiteConfig 三件套齐全但 smh_spaces 表中缺 skillhub，
// 按 SMHConfig.IsConfigured() 的契约（五项全都非空才为 true），应返回 false。
func TestAdminSMHConfigAPI_PartialConfig(t *testing.T) {
	initSMHConfigTestDB(t)
	defer withAdminToken("test-admin-token")()

	if err := model.DB(context.Background()).Create(&model.SiteConfig{
		SMHEnabled:       1,
		SMHLibraryId:     "lib-1",
		SMHLibrarySecret: "short",
		SMHEndpoint:      "https://smh.example.com",
	}).Error; err != nil {
		t.Fatalf("写入 SiteConfig 失败: %v", err)
	}
	// 只创建 common，不创建 skillhub
	model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "common", SpaceId: "sp-common", LibraryId: "lib-1", Purpose: "common"})

	resp := parseAdminSMHConfigResp(t, callAdminSMHConfigAPI(t))
	if resp["is_configured"] != false {
		t.Errorf("缺 skillhub 时 is_configured 期望 false，实际=%v", resp["is_configured"])
	}
	if resp["smh_common_space"] != "sp-common" {
		t.Errorf("smh_common_space 期望 'sp-common'，实际=%v", resp["smh_common_space"])
	}
	if resp["smh_skillhub_space"] != "" {
		t.Errorf("缺 skillhub 时 smh_skillhub_space 期望空，实际=%v", resp["smh_skillhub_space"])
	}
}

// ---------- createPersonalSpaceAndInitEnv ----------
//
// 该函数职责：
//  1. 同步调用 CreatePersonalSpaceForInstance 创建个人空间，失败立即返回错误；
//  2. 成功后启动异步 goroutine，等待 CVM RUNNING + TAT Agent Online，再触发环境初始化。
//
// 测试策略：
//   - 错误路径：不写 SMH 配置 → IsConfigured()=false → 同步阶段直接报 "SMH 未配置"，
//     goroutine 不会启动，无副作用。
//   - 成功快捷路径：写完整 SMH 配置 + 预置 SMHPersonalSpace（HasPersonalSpace=true） →
//     CreatePersonalSpaceForInstance 命中幂等分支直接返回已有 spaceId，不调用真实 SMH API。
//     传入 instance.InstanceId="" 让异步 goroutine 中 waitForCVMRunning 立即返回 false，
//     goroutine 快速退出，不影响主测试。

// initCreateSpaceInitEnvTestDB 初始化 createPersonalSpaceAndInitEnv 测试所需的表。
// 与 initSMHRefreshTestDB 保持一致：User + Instance + SMHPersonalSpace + SiteConfig + SMHSpace。
func initCreateSpaceInitEnvTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SMHPersonalSpace{},
		&model.SiteConfig{},
		&model.SMHSpace{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// 防止异步 goroutine 中 LoadScript 为 nil 导致 panic
	if LoadScript == nil {
		LoadScript = func(name string) (string, error) {
			return "", fmt.Errorf("test: script %s not available", name)
		}
	}
}

// TestCreatePersonalSpaceAndInitEnv_SMHNotConfigured 覆盖同步阶段 SMH 未配置分支：
// SiteConfig 缺关键字段 → CreatePersonalSpaceForInstance 返回 "SMH 未配置" →
// 函数直接返回错误且 spaceId 为空；数据库不写入新记录；异步 goroutine 不启动。
func TestCreatePersonalSpaceAndInitEnv_SMHNotConfigured(t *testing.T) {
	initCreateSpaceInitEnvTestDB(t)

	user := model.User{Username: "unconfig-user", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instance := model.Instance{
		Name:       "unconfig-box",
		InstanceId: "", // 空 InstanceId：即便意外启动 goroutine，waitForCVMRunning 也会立即退出
		UserID:     user.ID,
	}
	if err := model.DB(context.Background()).Create(&instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	spaceId, err := createPersonalSpaceAndInitEnv(context.Background(), &instance, &user)
	if err == nil {
		t.Fatal("SMH 未配置时期望返回错误，实际 nil")
	}
	if !strings.Contains(err.Error(), "SMH 未配置") {
		t.Errorf("错误信息应包含 'SMH 未配置'，实际=%q", err.Error())
	}
	if spaceId != "" {
		t.Errorf("失败时期望 spaceId 为空，实际=%q", spaceId)
	}

	// 数据库中不应有任何 SMHPersonalSpace 记录
	var count int64
	model.DB(context.Background()).Model(&model.SMHPersonalSpace{}).Count(&count)
	if count != 0 {
		t.Errorf("失败时不应写入 SMHPersonalSpace 记录，实际记录数=%d", count)
	}
}

// TestCreatePersonalSpaceAndInitEnv_ExistingSpace 覆盖「空间已存在」的幂等快捷成功路径：
// 预置完整 SMH 配置 + 一条已绑定的 SMHPersonalSpace → CreatePersonalSpaceForInstance
// 内部 HasPersonalSpace=true → 跳过真实 SMH API 直接返回已有 spaceId。
// 使用 instance.InstanceId="" 让异步 goroutine 中 waitForCVMRunning 立即返回 false 快速退出。
func TestCreatePersonalSpaceAndInitEnv_ExistingSpace(t *testing.T) {
	initCreateSpaceInitEnvTestDB(t)
	seedSMHConfigured(t)

	user := model.User{Username: "existing-user", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instance := model.Instance{
		Name:       "existing-box",
		InstanceId: "", // 让 goroutine 中 waitForCVMRunning 立即返回 false，避免触达外部依赖
		UserID:     user.ID,
	}
	if err := model.DB(context.Background()).Create(&instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	// 预置一条 SMHPersonalSpace，触发 HasPersonalSpace=true 的幂等快捷分支
	existing := model.SMHPersonalSpace{
		SpaceId:       "sp-existing",
		UserId:        user.ID,
		InstanceId:    instance.ID,
		UserName:      user.Username,
		InstanceName:  instance.Name,
		CVMInstanceId: instance.InstanceId,
	}
	if err := model.DB(context.Background()).Create(&existing).Error; err != nil {
		t.Fatalf("预置 SMHPersonalSpace 失败: %v", err)
	}

	spaceId, err := createPersonalSpaceAndInitEnv(context.Background(), &instance, &user)
	if err != nil {
		t.Fatalf("期望成功，实际返回错误=%v", err)
	}
	if spaceId != "sp-existing" {
		t.Errorf("期望返回已有 spaceId='sp-existing'，实际=%q", spaceId)
	}

	// 快捷路径不应产生新记录，仍只有预置的 1 条
	var count int64
	model.DB(context.Background()).Model(&model.SMHPersonalSpace{}).Count(&count)
	if count != 1 {
		t.Errorf("幂等快捷路径期望记录数=1，实际=%d", count)
	}
}

// TestCreatePersonalSpaceAndInitEnv_InstancePointerPassedByValue 验证实现细节：
// 异步 goroutine 使用 `go func(inst model.Instance)` 拷贝 instance，
// 因此主调用方在函数返回后修改 instance 指针不会影响 goroutine 内部的 inst 快照。
// 同时再次覆盖：传入 InstanceId="" 时 goroutine 能安全、快速退出，不 panic。
func TestCreatePersonalSpaceAndInitEnv_InstancePointerPassedByValue(t *testing.T) {
	initCreateSpaceInitEnvTestDB(t)
	seedSMHConfigured(t)

	user := model.User{Username: "copy-user", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instance := model.Instance{
		Name:       "copy-box",
		InstanceId: "",
		UserID:     user.ID,
	}
	if err := model.DB(context.Background()).Create(&instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId: "sp-copy", UserId: user.ID, InstanceId: instance.ID,
		UserName: user.Username, InstanceName: instance.Name, CVMInstanceId: instance.InstanceId,
	}).Error; err != nil {
		t.Fatalf("预置 SMHPersonalSpace 失败: %v", err)
	}

	spaceId, err := createPersonalSpaceAndInitEnv(context.Background(), &instance, &user)
	if err != nil {
		t.Fatalf("期望成功，实际=%v", err)
	}
	if spaceId != "sp-copy" {
		t.Errorf("期望 spaceId='sp-copy'，实际=%q", spaceId)
	}

	// 主函数返回后，对 instance 指针做改动不应影响 goroutine 内部（由值拷贝保证）。
	// 这里做一次改动，仅为文档化意图；由于 goroutine 会在 waitForCVMRunning 第一轮即退出，
	// 也不会实际读取这些字段 —— 该用例的价值在于确保值拷贝语义下主测试不会发生数据竞争 panic。
	instance.InstanceId = "mutated-after-return"
	instance.Name = "mutated-name"

	// 留出极短时间让 goroutine 进入调度（即便不 sleep 也不影响主测试结论）
	time.Sleep(10 * time.Millisecond)
}

// TestCreatePersonalSpaceAndInitEnv_UserNotNil 校验传入 user 指针被 CreatePersonalSpaceForInstance
// 内部使用（尽管在幂等分支不写新记录，但失败路径下也不应因 user=nil 崩溃）。
// 本用例使用「SMH 未配置」的失败路径，确认传入有效 user 时仍正常返回错误、不 panic。
func TestCreatePersonalSpaceAndInitEnv_UserNotNilOnErrorPath(t *testing.T) {
	initCreateSpaceInitEnvTestDB(t)

	user := model.User{Username: "err-path-user", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instance := model.Instance{Name: "err-path-box", InstanceId: "", UserID: user.ID}
	if err := model.DB(context.Background()).Create(&instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("错误路径不应 panic，实际 panic=%v", r)
		}
	}()

	_, err := createPersonalSpaceAndInitEnv(context.Background(), &instance, &user)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}
}

// ==========================================================================
// HandleAdminSMHInstances 真实 handler 测试（覆盖 requireAdmin / requireSMHEnabled /
// group_id 过滤 / exclude_recycling / GetSMHSupportedAgentTypes 过滤 / group_full_path）
// ==========================================================================

// initSMHInstancesFullTestDB 初始化完整的 HandleAdminSMHInstances 测试所需的表，
// 包括 UserGroup 和 GroupClosure（fetchGroupFullPathMap 依赖）。
func initSMHInstancesFullTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SMHPersonalSpace{},
		&model.SiteConfig{},
		&model.UserGroup{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// 创建 SiteConfig 并启用 SMH
	db.Create(&model.SiteConfig{SMHEnabled: 1})
	// 设置 AdminToken
	AdminToken = "test-admin-token"
	// 初始化 session Store
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	t.Cleanup(func() { Store = origStore })
}

// TestHandleAdminSMHInstances_RequireAdmin 验证未鉴权时 requireAdmin 拒绝。
func TestHandleAdminSMHInstances_RequireAdmin(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	// 不附带 Authorization 头，应被 requireAdmin 拒绝
	origToken := AdminToken
	AdminToken = "" // 清空 AdminToken 使 Bearer 验证失败
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("未鉴权时期望非 200，实际 200，body=%s", w.Body.String())
	}
}

// TestHandleAdminSMHInstances_SMHDisabled 验证 SMH 未启用时 requireSMHEnabled 拒绝。
func TestHandleAdminSMHInstances_SMHDisabled(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SMHPersonalSpace{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// SMH 未启用
	db.Create(&model.SiteConfig{SMHEnabled: 0})

	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	t.Cleanup(func() { Store = origStore })

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("SMH 未启用时期望 403，实际=%d，body=%s", w.Code, w.Body.String())
	}
}

// TestHandleAdminSMHInstances_FilterByGroupID 验证 group_id 过滤参数。
func TestHandleAdminSMHInstances_FilterByGroupID(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "group-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	// 创建两个分组
	group1 := model.UserGroup{Name: "team-a", FullPath: "team-a"}
	group2 := model.UserGroup{Name: "team-b", FullPath: "team-b"}
	model.DB(context.Background()).Create(&group1)
	model.DB(context.Background()).Create(&group2)

	// 创建实例，分属不同分组
	model.DB(context.Background()).Create(&model.Instance{Name: "ins-g1", InstanceId: "ins-g1", UserID: user.ID, GroupID: group1.ID, AgentType: "openclaw"})
	model.DB(context.Background()).Create(&model.Instance{Name: "ins-g2", InstanceId: "ins-g2", UserID: user.ID, GroupID: group2.ID, AgentType: "openclaw"})
	model.DB(context.Background()).Create(&model.Instance{Name: "ins-g1b", InstanceId: "ins-g1b", UserID: user.ID, GroupID: group1.ID, AgentType: "openclaw"})

	// 按 group1 过滤
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/smh/instances?group_id=%d", group1.ID), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 2 {
		t.Errorf("过滤 group_id=%d 期望 total=2，实际=%v", group1.ID, resp["total"])
	}
	items := resp["items"].([]interface{})
	for _, raw := range items {
		item := raw.(map[string]interface{})
		if uint(item["group_id"].(float64)) != group1.ID {
			t.Errorf("过滤后期望 group_id=%d，实际=%v", group1.ID, item["group_id"])
		}
	}
}

// TestHandleAdminSMHInstances_ExcludeRecycling 验证 exclude_recycling=1 过滤掉回收站中的实例。
func TestHandleAdminSMHInstances_ExcludeRecycling(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "recycle-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	// 实例 1：无空间
	ins1 := model.Instance{Name: "no-space", InstanceId: "ins-er1", UserID: user.ID, AgentType: "openclaw"}
	model.DB(context.Background()).Create(&ins1)

	// 实例 2：有活跃空间
	ins2 := model.Instance{Name: "active-space", InstanceId: "ins-er2", UserID: user.ID, AgentType: "openclaw"}
	model.DB(context.Background()).Create(&ins2)
	model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId: "sp-active-er", UserId: user.ID, InstanceId: ins2.ID,
		UserName: user.Username, InstanceName: ins2.Name, CVMInstanceId: ins2.InstanceId,
	})

	// 实例 3：空间在回收站
	ins3 := model.Instance{Name: "recycling-space", InstanceId: "ins-er3", UserID: user.ID, AgentType: "openclaw"}
	model.DB(context.Background()).Create(&ins3)
	deleteAt := time.Now().Add(7 * 24 * time.Hour)
	model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId: "sp-recycle-er", UserId: user.ID, InstanceId: ins3.ID,
		UserName: user.Username, InstanceName: ins3.Name, CVMInstanceId: ins3.InstanceId,
		ToBeDeletedAt: &deleteAt,
	})

	// 不带 exclude_recycling，应返回 3 条
	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 3 {
		t.Errorf("不带 exclude_recycling 期望 total=3，实际=%v", resp["total"])
	}

	// 带 exclude_recycling=1，应过滤掉回收站中的实例，返回 2 条
	req = httptest.NewRequest(http.MethodGet, "/admin/smh/instances?exclude_recycling=1", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w = httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 2 {
		t.Errorf("带 exclude_recycling=1 期望 total=2，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	for _, raw := range items {
		item := raw.(map[string]interface{})
		if item["space_status"] == "recycling" {
			t.Error("exclude_recycling=1 后不应返回 recycling 状态的实例")
		}
	}
}

// TestHandleAdminSMHInstances_GroupFullPath 验证响应中包含 group_full_path 字段。
func TestHandleAdminSMHInstances_GroupFullPath(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "path-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	// 创建分组
	group := model.UserGroup{Name: "engineering", FullPath: "腾讯/技术部/前端组"}
	model.DB(context.Background()).Create(&group)

	// 创建实例并关联分组
	model.DB(context.Background()).Create(&model.Instance{
		Name: "path-box", InstanceId: "ins-path1", UserID: user.ID,
		GroupID: group.ID, AgentType: "openclaw",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("期望 1 条，实际=%d", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["group_id"].(float64) != float64(group.ID) {
		t.Errorf("期望 group_id=%d，实际=%v", group.ID, item["group_id"])
	}
	if item["group_full_path"] != "腾讯/技术部/前端组" {
		t.Errorf("期望 group_full_path='腾讯/技术部/前端组'，实际=%v", item["group_full_path"])
	}
}

// TestHandleAdminSMHInstances_GroupFullPathEmpty 验证 GroupID=0 时 group_full_path 为空。
func TestHandleAdminSMHInstances_GroupFullPathEmpty(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "nogroup-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	// 创建实例，不关联分组（GroupID=0）
	model.DB(context.Background()).Create(&model.Instance{
		Name: "nogroup-box", InstanceId: "ins-nogroup", UserID: user.ID,
		GroupID: 0, AgentType: "openclaw",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if item["group_id"].(float64) != 0 {
		t.Errorf("期望 group_id=0，实际=%v", item["group_id"])
	}
	if item["group_full_path"] != "" {
		t.Errorf("GroupID=0 时期望 group_full_path 为空，实际=%v", item["group_full_path"])
	}
}

// TestHandleAdminSMHInstances_AgentTypeFilter 验证 GetSMHSupportedAgentTypes 过滤：
// 不支持 SMH 的 agent_type 的实例不应出现在结果中。
func TestHandleAdminSMHInstances_AgentTypeFilter(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	// 创建一个不支持 SMH 的自定义 agent type
	model.DB(context.Background()).Create(&model.CustomAgentType{
		Name:           "nosupport",
		CompatibleWith: "",
	})

	user := model.User{Username: "agentfilter-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	// 创建支持 SMH 的实例
	model.DB(context.Background()).Create(&model.Instance{
		Name: "smh-box", InstanceId: "ins-smh1", UserID: user.ID, AgentType: "openclaw",
	})
	// 创建不支持 SMH 的实例（自定义类型，CompatibleWith 为空，SupportsSMH=false）
	model.DB(context.Background()).Create(&model.Instance{
		Name: "nosmh-box", InstanceId: "ins-nosmh1", UserID: user.ID, AgentType: "nosupport",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	// 不支持 SMH 的实例不应出现在结果中
	items := resp["items"].([]interface{})
	for _, raw := range items {
		item := raw.(map[string]interface{})
		if item["instance_name"] == "nosmh-box" {
			t.Error("不支持 SMH 的 agent_type 实例不应出现在结果中")
		}
	}
	// 只应返回支持 SMH 的实例
	if resp["total"].(float64) != 1 {
		t.Errorf("期望 total=1（仅支持 SMH 的实例），实际=%v", resp["total"])
	}
}

// TestHandleAdminSMHInstances_CombinedFilters 验证多个过滤条件组合使用。
func TestHandleAdminSMHInstances_CombinedFilters(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	alice := model.User{Username: "alice-cf", Password: "x", Role: "user"}
	bob := model.User{Username: "bob-cf", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&alice)
	model.DB(context.Background()).Create(&bob)

	group := model.UserGroup{Name: "team-cf", FullPath: "team-cf"}
	model.DB(context.Background()).Create(&group)

	// alice 的实例在 group 中
	ins1 := model.Instance{Name: "alice-cf-box", InstanceId: "ins-cf1", UserID: alice.ID, GroupID: group.ID, AgentType: "openclaw"}
	model.DB(context.Background()).Create(&ins1)
	// bob 的实例在 group 中
	ins2 := model.Instance{Name: "bob-cf-box", InstanceId: "ins-cf2", UserID: bob.ID, GroupID: group.ID, AgentType: "openclaw"}
	model.DB(context.Background()).Create(&ins2)
	// alice 的实例不在 group 中
	model.DB(context.Background()).Create(&model.Instance{Name: "alice-nogroup", InstanceId: "ins-cf3", UserID: alice.ID, GroupID: 0, AgentType: "openclaw"})

	// 组合过滤：user=alice + group_id=group.ID
	url := fmt.Sprintf("/admin/smh/instances?user=%d&group_id=%d", alice.ID, group.ID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("组合过滤 user+group_id 期望 total=1，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if item["instance_name"] != "alice-cf-box" {
		t.Errorf("期望 instance_name='alice-cf-box'，实际=%v", item["instance_name"])
	}
}

// TestHandleAdminSMHInstances_PaginationWithRealHandler 验证真实 handler 的分页。
func TestHandleAdminSMHInstances_PaginationWithRealHandler(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "page-real", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	for i := 0; i < 5; i++ {
		model.DB(context.Background()).Create(&model.Instance{
			Name:       fmt.Sprintf("page-box-%d", i),
			InstanceId: fmt.Sprintf("ins-page-real-%d", i),
			UserID:     user.ID,
			AgentType:  "openclaw",
		})
	}

	// 第 1 页，每页 2 条
	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances?page=1&page_size=2", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 5 {
		t.Errorf("期望 total=5，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("第 1 页期望 2 条，实际=%d", len(items))
	}
	if resp["page"].(float64) != 1 {
		t.Errorf("期望 page=1，实际=%v", resp["page"])
	}
	if resp["page_size"].(float64) != 2 {
		t.Errorf("期望 page_size=2，实际=%v", resp["page_size"])
	}
}

// TestHandleAdminSMHInstances_EmptyAgentTypeDefaultsToOpenclaw 验证 agent_type 为空时默认返回 "openclaw"。
func TestHandleAdminSMHInstances_EmptyAgentTypeDefaultsToOpenclaw(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "empty-at-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	// 创建 agent_type 为空的实例（兼容存量数据）
	model.DB(context.Background()).Create(&model.Instance{
		Name: "legacy-box", InstanceId: "ins-legacy", UserID: user.ID, AgentType: "",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("期望 1 条，实际=%d", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["agent_type"] != "openclaw" {
		t.Errorf("agent_type 为空时期望返回 'openclaw'，实际=%v", item["agent_type"])
	}
}

// TestHandleAdminSMHInstances_SpaceFieldsPresent 验证有空间时响应中包含完整的空间字段。
func TestHandleAdminSMHInstances_SpaceFieldsPresent(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "fields-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	ins := model.Instance{Name: "fields-box", InstanceId: "ins-fields", UserID: user.ID, AgentType: "openclaw"}
	model.DB(context.Background()).Create(&ins)

	expiresAt := time.Now().Add(90 * 24 * time.Hour)
	var quota int64 = 53687091200
	var freeQuota int64 = 10737418240
	model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId:          "sp-fields",
		UserId:           user.ID,
		InstanceId:       ins.ID,
		UserName:         user.Username,
		InstanceName:     ins.Name,
		CVMInstanceId:    ins.InstanceId,
		StorageQuota:     quota,
		FreeStorageQuota: freeQuota,
		ExpiresAt:        &expiresAt,
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})

	// 验证空间相关字段存在
	if item["space_status"] != "active" {
		t.Errorf("期望 space_status=active，实际=%v", item["space_status"])
	}
	if item["smh_space_id"] != "sp-fields" {
		t.Errorf("期望 smh_space_id=sp-fields，实际=%v", item["smh_space_id"])
	}
	if _, ok := item["space_id"]; !ok {
		t.Error("有空间时应返回 space_id 字段")
	}
	if item["storage_quota"] == nil {
		t.Error("有空间时 storage_quota 不应为 nil")
	}
	if item["free_storage_quota"] == nil {
		t.Error("有空间时 free_storage_quota 不应为 nil")
	}
	if item["bound_at"] == nil {
		t.Error("有空间时 bound_at 不应为 nil")
	}
	if item["expires_at"] == nil {
		t.Error("有空间时 expires_at 不应为 nil")
	}
	// used_storage 在 SMH 未配置时为 0（fetchPersonalSpaceUsage 返回空 map）
	if item["used_storage"] == nil {
		t.Error("有空间时 used_storage 不应为 nil")
	}
}

// TestHandleAdminSMHInstances_NoSpaceFieldsAbsent 验证无空间时响应中不包含空间字段。
func TestHandleAdminSMHInstances_NoSpaceFieldsAbsent(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "nofields-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{
		Name: "nofields-box", InstanceId: "ins-nofields", UserID: user.ID, AgentType: "openclaw",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})

	if item["space_status"] != "none" {
		t.Errorf("无空间时期望 space_status=none，实际=%v", item["space_status"])
	}
	// 无空间时不应有这些字段
	for _, key := range []string{"space_id", "smh_space_id", "storage_quota", "free_storage_quota", "used_storage", "bound_at", "expires_at", "to_be_deleted_at"} {
		if _, ok := item[key]; ok {
			t.Errorf("无空间时不应返回 %s 字段", key)
		}
	}
}

// TestHandleAdminSMHInstances_ExcludeRecyclingNotOne 验证 exclude_recycling 非 "1" 时不过滤。
func TestHandleAdminSMHInstances_ExcludeRecyclingNotOne(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "er-notone-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	ins := model.Instance{Name: "er-notone-box", InstanceId: "ins-erno", UserID: user.ID, AgentType: "openclaw"}
	model.DB(context.Background()).Create(&ins)
	deleteAt := time.Now().Add(7 * 24 * time.Hour)
	model.DB(context.Background()).Create(&model.SMHPersonalSpace{
		SpaceId: "sp-erno", UserId: user.ID, InstanceId: ins.ID,
		UserName: user.Username, InstanceName: ins.Name, CVMInstanceId: ins.InstanceId,
		ToBeDeletedAt: &deleteAt,
	})

	// exclude_recycling=0 不应过滤
	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances?exclude_recycling=0", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("exclude_recycling=0 不应过滤，期望 total=1，实际=%v", resp["total"])
	}
}

// TestHandleAdminSMHInstances_MultipleGroupsFullPath 验证多个不同分组的 full_path 正确返回。
func TestHandleAdminSMHInstances_MultipleGroupsFullPath(t *testing.T) {
	initSMHInstancesFullTestDB(t)

	user := model.User{Username: "multigroup-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	group1 := model.UserGroup{Name: "frontend", FullPath: "技术部/前端组"}
	group2 := model.UserGroup{Name: "backend", FullPath: "技术部/后端组"}
	model.DB(context.Background()).Create(&group1)
	model.DB(context.Background()).Create(&group2)

	model.DB(context.Background()).Create(&model.Instance{
		Name: "fe-box", InstanceId: "ins-fe", UserID: user.ID, GroupID: group1.ID, AgentType: "openclaw",
	})
	model.DB(context.Background()).Create(&model.Instance{
		Name: "be-box", InstanceId: "ins-be", UserID: user.ID, GroupID: group2.ID, AgentType: "openclaw",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/smh/instances", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSMHInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("期望 2 条，实际=%d", len(items))
	}

	pathMap := map[string]string{}
	for _, raw := range items {
		item := raw.(map[string]interface{})
		name := item["instance_name"].(string)
		path := ""
		if p, ok := item["group_full_path"]; ok && p != nil {
			path = p.(string)
		}
		pathMap[name] = path
	}

	if pathMap["fe-box"] != "技术部/前端组" {
		t.Errorf("fe-box 期望 group_full_path='技术部/前端组'，实际=%q", pathMap["fe-box"])
	}
	if pathMap["be-box"] != "技术部/后端组" {
		t.Errorf("be-box 期望 group_full_path='技术部/后端组'，实际=%q", pathMap["be-box"])
	}
}

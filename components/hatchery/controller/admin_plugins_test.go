package controller

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ── validatePluginZip 测试 ────────────────────────────────────────

func TestValidatePluginZip_EmptyData(t *testing.T) {
	_, _, _, err := validatePluginZip([]byte{}, "test-plugin")
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestValidatePluginZip_NonZipData(t *testing.T) {
	_, _, _, err := validatePluginZip([]byte("not a zip"), "test-plugin")
	if err == nil {
		t.Error("expected error for non-zip data")
	}
}

func TestValidatePluginZip_EmptyZip(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	w.Close()

	_, _, _, err := validatePluginZip(buf.Bytes(), "test-plugin")
	if err == nil {
		t.Error("expected error for empty zip")
	}
}

func TestValidatePluginZip_NoManifest(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("some-file.txt")
	fw.Write([]byte("hello"))
	w.Close()

	_, _, _, err := validatePluginZip(buf.Bytes(), "test-plugin")
	if err == nil {
		t.Error("expected error for zip without manifest")
	}
	if err != nil && !bytes.Contains([]byte(err.Error()), []byte("openclaw.plugin.json")) {
		t.Errorf("error should mention openclaw.plugin.json, got: %v", err)
	}
}

func TestValidatePluginZip_ValidOpenClawPlugin(t *testing.T) {
	manifest := map[string]interface{}{
		"id":      "my-test-plugin",
		"name":    "My Test Plugin",
		"version": "1.0.0",
		"kind":    "memory",
		"configSchema": map[string]interface{}{
			"type": "object",
		},
		"providers": []string{"openai"},
		"channels":  []string{"wecom"},
	}
	manifestJSON, _ := json.Marshal(manifest)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// 创建带目录前缀的 manifest
	fw, _ := w.Create("my-plugin/openclaw.plugin.json")
	fw.Write(manifestJSON)

	// 添加其他文件
	fw2, _ := w.Create("my-plugin/index.js")
	fw2.Write([]byte("module.exports = {}"))

	fw3, _ := w.Create("my-plugin/package.json")
	fw3.Write([]byte(`{"name": "my-test-plugin"}`))

	w.Close()

	files, repackedZip, meta, err := validatePluginZip(buf.Bytes(), "test-plugin")
	if err != nil {
		t.Fatalf("validatePluginZip failed: %v", err)
	}

	// 验证文件列表
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d: %v", len(files), files)
	}

	// 验证重新打包的 zip 不为空
	if len(repackedZip) == 0 {
		t.Error("repacked zip should not be empty")
	}

	// 验证元数据
	if meta.PluginID != "my-test-plugin" {
		t.Errorf("PluginID = %q, want %q", meta.PluginID, "my-test-plugin")
	}
	if meta.PluginFormat != "openclaw" {
		t.Errorf("PluginFormat = %q, want %q", meta.PluginFormat, "openclaw")
	}
	if meta.Kind != "memory" {
		t.Errorf("Kind = %q, want %q", meta.Kind, "memory")
	}
	if meta.ConfigSchema == "" {
		t.Error("ConfigSchema should not be empty")
	}
	if meta.Providers == "" {
		t.Error("Providers should not be empty")
	}
	if meta.Channels == "" {
		t.Error("Channels should not be empty")
	}

	// 验证重新打包的 zip 中文件路径以 slug 为前缀
	r, _ := zip.NewReader(bytes.NewReader(repackedZip), int64(len(repackedZip)))
	for _, f := range r.File {
		if !bytes.HasPrefix([]byte(f.Name), []byte("test-plugin/")) {
			t.Errorf("repacked file %q should start with 'test-plugin/'", f.Name)
		}
	}
}

func TestValidatePluginZip_MissingID(t *testing.T) {
	manifest := map[string]interface{}{
		"name":    "No ID Plugin",
		"version": "1.0.0",
	}
	manifestJSON, _ := json.Marshal(manifest)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("openclaw.plugin.json")
	fw.Write(manifestJSON)
	w.Close()

	_, _, _, err := validatePluginZip(buf.Bytes(), "test-plugin")
	if err == nil {
		t.Error("expected error for manifest without id")
	}
}

func TestValidatePluginZip_InvalidKind(t *testing.T) {
	manifest := map[string]interface{}{
		"id":   "test-plugin",
		"kind": "invalid-kind",
	}
	manifestJSON, _ := json.Marshal(manifest)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("openclaw.plugin.json")
	fw.Write(manifestJSON)
	w.Close()

	_, _, _, err := validatePluginZip(buf.Bytes(), "test-plugin")
	if err == nil {
		t.Error("expected error for invalid kind")
	}
}

func TestValidatePluginZip_ValidKinds(t *testing.T) {
	validKinds := []string{"", "memory", "context-engine"}

	for _, kind := range validKinds {
		t.Run("kind="+kind, func(t *testing.T) {
			manifest := map[string]interface{}{
				"id":   "test-plugin",
				"kind": kind,
			}
			manifestJSON, _ := json.Marshal(manifest)

			var buf bytes.Buffer
			w := zip.NewWriter(&buf)
			fw, _ := w.Create("openclaw.plugin.json")
			fw.Write(manifestJSON)
			w.Close()

			_, _, meta, err := validatePluginZip(buf.Bytes(), "test-plugin")
			if err != nil {
				t.Fatalf("validatePluginZip failed for kind=%q: %v", kind, err)
			}
			if meta.Kind != kind {
				t.Errorf("Kind = %q, want %q", meta.Kind, kind)
			}
		})
	}
}

func TestValidatePluginZip_BundleFormat(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// 创建 .codex-plugin 目录结构
	fw, _ := w.Create("my-bundle/.codex-plugin/manifest.json")
	fw.Write([]byte(`{"name": "test"}`))

	fw2, _ := w.Create("my-bundle/index.js")
	fw2.Write([]byte("module.exports = {}"))

	w.Close()

	files, _, meta, err := validatePluginZip(buf.Bytes(), "test-bundle")
	if err != nil {
		t.Fatalf("validatePluginZip bundle format failed: %v", err)
	}

	if meta.PluginFormat != "bundle" {
		t.Errorf("PluginFormat = %q, want %q", meta.PluginFormat, "bundle")
	}
	if meta.PluginID != "test-bundle" {
		t.Errorf("PluginID = %q, want %q (should use slug for bundle)", meta.PluginID, "test-bundle")
	}
	if len(files) == 0 {
		t.Error("expected files in bundle")
	}
}

func TestValidatePluginZip_ZipSlipProtection(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// 创建 manifest
	fw, _ := w.Create("openclaw.plugin.json")
	fw.Write([]byte(`{"id": "test"}`))

	// 创建包含 .. 的路径
	fw2, _ := w.Create("../evil.sh")
	fw2.Write([]byte("rm -rf /"))

	w.Close()

	_, _, _, err := validatePluginZip(buf.Bytes(), "test-plugin")
	if err == nil {
		t.Error("expected error for zip slip attack")
	}
}

func TestValidatePluginZip_SkipsMacOSX(t *testing.T) {
	manifest := map[string]interface{}{
		"id": "test-plugin",
	}
	manifestJSON, _ := json.Marshal(manifest)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	fw, _ := w.Create("test/openclaw.plugin.json")
	fw.Write(manifestJSON)

	fw2, _ := w.Create("test/index.js")
	fw2.Write([]byte("module.exports = {}"))

	// __MACOSX 文件应被跳过
	fw3, _ := w.Create("test/__MACOSX/._index.js")
	fw3.Write([]byte("macosx metadata"))

	// .git 文件应被跳过
	fw4, _ := w.Create("test/.git/config")
	fw4.Write([]byte("git config"))

	w.Close()

	files, _, _, err := validatePluginZip(buf.Bytes(), "test-plugin")
	if err != nil {
		t.Fatalf("validatePluginZip failed: %v", err)
	}

	// 应该只有 2 个文件（manifest + index.js），__MACOSX 和 .git 被跳过
	if len(files) != 2 {
		t.Errorf("expected 2 files (skipping __MACOSX and .git), got %d: %v", len(files), files)
	}
}

// ── ErrPluginConflict 测试 ───────────────────────────────────────

func TestErrPluginConflict(t *testing.T) {
	err2 := &ErrPluginConflict{Slug: "my-plugin", Version: "1.0.0"}
	expected := "插件 my-plugin-1.0.0 已存在于该插件包中"
	if err2.Error() != expected {
		t.Errorf("ErrPluginConflict.Error() = %q, want %q", err2.Error(), expected)
	}
}

// ── batchPluginInstallResult 解析测试 ────────────────────────────

func TestBatchPluginInstallResultParsing(t *testing.T) {
	jsonStr := `{
		"results": [
			{"slug": "plugin-a", "version": "1.0.0", "status": "success", "message": ""},
			{"slug": "plugin-b", "version": "2.0.0", "status": "failed", "message": "download error"}
		],
		"summary": {"total": 2, "success": 1, "failed": 1}
	}`

	var result batchPluginInstallResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	if result.Summary.Total != 2 {
		t.Errorf("summary total = %d, want 2", result.Summary.Total)
	}
	if result.Summary.Success != 1 {
		t.Errorf("summary success = %d, want 1", result.Summary.Success)
	}
	if result.Summary.Failed != 1 {
		t.Errorf("summary failed = %d, want 1", result.Summary.Failed)
	}

	// 验证结果映射
	if result.Results[0].Slug != "plugin-a" || result.Results[0].Status != "success" {
		t.Errorf("result[0] unexpected: %+v", result.Results[0])
	}
	if result.Results[1].Slug != "plugin-b" || result.Results[1].Status != "failed" {
		t.Errorf("result[1] unexpected: %+v", result.Results[1])
	}
}

// ── pluginMeta 结构体测试 ────────────────────────────────────────

func TestPluginMetaFields(t *testing.T) {
	meta := &pluginMeta{
		PluginID:     "test-id",
		PluginFormat: "openclaw",
		Kind:         "memory",
		ConfigSchema: `{"type":"object"}`,
		Providers:    `["openai"]`,
		Channels:     `["wecom"]`,
	}

	if meta.PluginID != "test-id" {
		t.Errorf("PluginID = %q, want %q", meta.PluginID, "test-id")
	}
	if meta.PluginFormat != "openclaw" {
		t.Errorf("PluginFormat = %q, want %q", meta.PluginFormat, "openclaw")
	}
	if meta.Kind != "memory" {
		t.Errorf("Kind = %q, want %q", meta.Kind, "memory")
	}
}

// ── HandleAdminPluginInstances / HandleDistributePlugin 输入验证测试 ──

// setupPluginInstancesDB 初始化内存 SQLite 数据库，包含插件实例相关表
func setupPluginInstancesDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	// SQLite in-memory 数据库是连接私有的，必须固定为单连接，
	// 否则连接池新开连接会看到空数据库（无任何表）。
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.Instance{},
		&model.User{},
		&model.Plugin{},
		&model.PluginDistributionRecord{},
		&model.PluginDistributionTask{},
		&model.SiteConfig{},
		&model.SMHSpace{},
		&model.UserGroup{},
		&model.UserGroupMember{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	if common.FixedSnapshot == nil {
		old := common.FixedSnapshot
		common.FixedSnapshot = &common.TenantSnapshot{}
		t.Cleanup(func() { common.FixedSnapshot = old })
	}
	// 启用 SMH
	db.Create(&model.SiteConfig{SMHEnabled: 1})

	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	var wg sync.WaitGroup
	pluginDistributeWG = &wg
	t.Cleanup(func() {
		wg.Wait()
		pluginDistributeWG = nil
	})

}

// seedPluginAndInstances 创建测试用的插件和实例数据，返回 plugin slug
func seedPluginAndInstances(t *testing.T) string {
	t.Helper()
	slug := "test-plugin"

	// 创建插件
	plugin := model.Plugin{
		Slug: slug, Name: "测试插件", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		PluginID: "test-plugin-id", PluginFormat: "openclaw",
	}
	model.DB(context.Background()).Create(&plugin)

	// 创建用户
	user1 := model.User{Username: "alice"}
	model.DB(context.Background()).Create(&user1)
	user2 := model.User{Username: "bob"}
	model.DB(context.Background()).Create(&user2)

	// 创建不同类型的实例
	instances := []model.Instance{
		{Name: "inst-alice-oc", InstanceId: "ins-aaa111", UserID: user1.ID, LastCVMState: "RUNNING", AgentReady: 1, AgentType: "openclaw"},
		{Name: "inst-alice-hm", InstanceId: "ins-aaa222", UserID: user1.ID, LastCVMState: "RUNNING", AgentReady: 1, AgentType: "hermes"},
		{Name: "inst-bob-oc", InstanceId: "ins-bbb111", UserID: user2.ID, LastCVMState: "RUNNING", AgentReady: 1, AgentType: "openclaw"},
	}
	for i := range instances {
		model.DB(context.Background()).Create(&instances[i])
	}

	// 创建下发记录
	for _, inst := range instances {
		record := model.PluginDistributionRecord{
			TaskID: 1, PluginDBID: plugin.ID, InstanceID: inst.ID,
			InstanceCID: inst.InstanceId, Version: "1.0.0", Status: "success",
		}
		model.DB(context.Background()).Create(&record)
	}

	return slug
}

// ── HandleAdminPluginInstances 测试 ──

func TestHandleAdminPluginInstances_MissingSlug(t *testing.T) {
	// 缺少 slug 参数应返回 400
	setupPluginInstancesDB(t)
	w := httptest.NewRecorder()
	HandleAdminPluginInstances(w, adminJSONGet("/admin/plugins/instances"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleAdminPluginInstances_PluginNotFound(t *testing.T) {
	// 不存在的 slug 应返回 404
	setupPluginInstancesDB(t)
	w := httptest.NewRecorder()
	HandleAdminPluginInstances(w, adminJSONGet("/admin/plugins/instances?slug=no-such-plugin"))
	if w.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleAdminPluginInstances_BasicFlow(t *testing.T) {
	// 基本查询流程，验证响应结构完整
	setupPluginInstancesDB(t)
	slug := seedPluginAndInstances(t)

	w := httptest.NewRecorder()
	HandleAdminPluginInstances(w, adminJSONGet("/admin/plugins/instances?slug="+slug))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeJSON(t, w)
	// 没有 CVM 客户端 → batchFetchCVMInfoMap 返回空 map →
	// ResolveInstanceStatus 标记为 destroyed → 被 running 过滤掉
	if resp["instances"] == nil {
		t.Error("响应缺少 instances 字段")
	}
	if _, ok := resp["page"]; !ok {
		t.Error("响应缺少 page 字段")
	}
	if _, ok := resp["page_size"]; !ok {
		t.Error("响应缺少 page_size 字段")
	}
	if _, ok := resp["total"]; !ok {
		t.Error("响应缺少 total 字段")
	}
}

func TestHandleAdminPluginInstances_InstanceTypeFilter(t *testing.T) {
	// 按 instance_type 筛选，覆盖 lines 1159-1171
	setupPluginInstancesDB(t)
	slug := seedPluginAndInstances(t)

	// 单类型过滤
	w := httptest.NewRecorder()
	HandleAdminPluginInstances(w, adminJSONGet("/admin/plugins/instances?slug="+slug+"&instance_type=openclaw"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	decodeJSON(t, w)

	// 多类型（逗号分隔）过滤
	w2 := httptest.NewRecorder()
	HandleAdminPluginInstances(w2, adminJSONGet("/admin/plugins/instances?slug="+slug+"&instance_type=openclaw,hermes"))
	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200（多类型过滤），实际=%d, body=%s", w2.Code, w2.Body.String())
	}
	decodeJSON(t, w2)
}

func TestHandleAdminPluginInstances_InstanceTypeFilter_Invalid(t *testing.T) {
	// 无效的 instance_type 被静默忽略（IsValidAgentType 返回 false）
	setupPluginInstancesDB(t)
	slug := seedPluginAndInstances(t)

	w := httptest.NewRecorder()
	HandleAdminPluginInstances(w, adminJSONGet("/admin/plugins/instances?slug="+slug+"&instance_type=invalid_type"))
	if w.Code != http.StatusOK {
		t.Errorf("期望 200（无效类型静默忽略），实际=%d", w.Code)
	}
}

func TestHandleAdminPluginInstances_SearchFilter(t *testing.T) {
	// search 参数覆盖 lines 1155-1157
	setupPluginInstancesDB(t)
	slug := seedPluginAndInstances(t)

	w := httptest.NewRecorder()
	HandleAdminPluginInstances(w, adminJSONGet("/admin/plugins/instances?slug="+slug+"&search=alice"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if resp["total"] == nil {
		t.Error("响应缺少 total 字段")
	}
}

func TestHandleAdminPluginInstances_GroupIDFilter(t *testing.T) {
	// group_id 筛选覆盖 lines 1123-1153
	setupPluginInstancesDB(t)
	slug := seedPluginAndInstances(t)

	// 创建用户组
	group := model.UserGroup{Name: "插件测试组"}
	model.DB(context.Background()).Create(&group)
	var alice model.User
	model.DB(context.Background()).Where("username = ?", "alice").First(&alice)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: alice.ID})

	// 仅指定分组
	w := httptest.NewRecorder()
	HandleAdminPluginInstances(w, adminJSONGet("/admin/plugins/instances?slug="+slug+"&group_id="+fmt.Sprintf("%d", group.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	// 仅未分组（group_id=0）
	w2 := httptest.NewRecorder()
	HandleAdminPluginInstances(w2, adminJSONGet("/admin/plugins/instances?slug="+slug+"&group_id=0"))
	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200（未分组），实际=%d", w2.Code)
	}

	// 未分组 + 指定分组（OR 语义）
	w3 := httptest.NewRecorder()
	HandleAdminPluginInstances(w3, adminJSONGet("/admin/plugins/instances?slug="+slug+"&group_id=0,"+fmt.Sprintf("%d", group.ID)))
	if w3.Code != http.StatusOK {
		t.Fatalf("期望 200（未分组+分组 OR），实际=%d", w3.Code)
	}
}

func TestHandleAdminPluginInstances_StatusFilter(t *testing.T) {
	// 安装状态筛选覆盖 lines 1174-1183
	setupPluginInstancesDB(t)
	slug := seedPluginAndInstances(t)

	for _, status := range []string{"installed", "uninstalled", "installing", "failed", "installed,failed"} {
		w := httptest.NewRecorder()
		HandleAdminPluginInstances(w, adminJSONGet("/admin/plugins/instances?slug="+slug+"&status="+status))
		if w.Code != http.StatusOK {
			t.Errorf("status=%s: 期望 200，实际=%d", status, w.Code)
		}
	}
}

func TestHandleAdminPluginInstances_Pagination(t *testing.T) {
	// 分页参数正确传递
	setupPluginInstancesDB(t)
	slug := seedPluginAndInstances(t)

	w := httptest.NewRecorder()
	HandleAdminPluginInstances(w, adminJSONGet("/admin/plugins/instances?slug="+slug+"&page=2&page_size=5"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	resp := decodeJSON(t, w)
	if int(resp["page"].(float64)) != 2 {
		t.Errorf("期望 page=2，实际=%v", resp["page"])
	}
	if int(resp["page_size"].(float64)) != 5 {
		t.Errorf("期望 page_size=5，实际=%v", resp["page_size"])
	}
}

func TestHandleAdminPluginInstances_CombinedFilters(t *testing.T) {
	// 同时使用 search + status + instance_type + group_id 过滤
	setupPluginInstancesDB(t)
	slug := seedPluginAndInstances(t)

	group := model.UserGroup{Name: "组合过滤组"}
	model.DB(context.Background()).Create(&group)

	w := httptest.NewRecorder()
	HandleAdminPluginInstances(w, adminJSONGet(
		"/admin/plugins/instances?slug="+slug+
			"&search=inst&status=installed&instance_type=openclaw"+
			"&group_id=0,"+fmt.Sprintf("%d", group.ID)+
			"&page=1&page_size=10"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	decodeJSON(t, w)
}

// ── HandleDistributePlugin 测试 ──

func TestHandleDistributePlugin_EmptySlug(t *testing.T) {
	// slug 为空应返回 400
	setupPluginInstancesDB(t)

	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute", `{"slug":"","instance_ids":[1]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributePlugin_EmptyInstanceIDs(t *testing.T) {
	// instance_ids 为空数组应返回 400
	setupPluginInstancesDB(t)

	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute", `{"slug":"some-plugin","instance_ids":[]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributePlugin_TooManyInstanceIDs(t *testing.T) {
	// 超过 500 个 instance_ids 应返回 400
	setupPluginInstancesDB(t)

	// 构造 501 个 ID
	ids := make([]string, 501)
	for i := range ids {
		ids[i] = fmt.Sprintf("%d", i+1)
	}
	body := fmt.Sprintf(`{"slug":"some-plugin","instance_ids":[%s]}`, strings.Join(ids, ","))

	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute", body))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributePlugin_InvalidJSON(t *testing.T) {
	// 无效 JSON 应返回 400
	setupPluginInstancesDB(t)

	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute", `{invalid json`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributePlugin_PluginNotFound(t *testing.T) {
	// 不存在的 slug 应返回 400
	setupPluginInstancesDB(t)

	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute", `{"slug":"nonexistent-plugin","instance_ids":[1,2]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributePlugin_PluginVersionNotFound(t *testing.T) {
	// 存在的 slug 但指定了不存在的版本应返回 400
	setupPluginInstancesDB(t)
	slug := seedPluginAndInstances(t)

	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute",
		fmt.Sprintf(`{"slug":"%s","version":"99.99.99","instance_ids":[1]}`, slug)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributePlugin_AllUnsupportedTypes(t *testing.T) {
	// 所有实例类型均不支持插件 → 返回 400
	setupPluginInstancesDB(t)

	plugin := model.Plugin{
		Slug: "dist-plugin-test", Name: "下发测试插件", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		PluginID: "dist-plugin-id", PluginFormat: "openclaw",
	}
	model.DB(context.Background()).Create(&plugin)

	user := model.User{Username: "dist-user"}
	model.DB(context.Background()).Create(&user)

	// hermes 和 lightclawace 不支持插件
	instances := []model.Instance{
		{Name: "hermes-inst", InstanceId: "ins-hermes-001", UserID: user.ID, AgentType: "hermes"},
		{Name: "lightclaw-inst", InstanceId: "ins-lc-001", UserID: user.ID, AgentType: "lightclawace"},
	}
	for i := range instances {
		model.DB(context.Background()).Create(&instances[i])
	}

	body := fmt.Sprintf(`{"slug":"dist-plugin-test","instance_ids":[%d,%d]}`,
		instances[0].ID, instances[1].ID)

	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute", body))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400（全部不支持插件），实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 确认没有创建任何 task（过滤在创建 task 之前）
	var taskCount int64
	model.DB(context.Background()).Model(&model.PluginDistributionTask{}).Where("plugin_db_id = ?", plugin.ID).Count(&taskCount)
	if taskCount != 0 {
		t.Errorf("期望 0 个 task，实际=%d", taskCount)
	}
}

func TestHandleDistributePlugin_DeduplicateInstanceIDs(t *testing.T) {
	// 验证 instance_ids 去重逻辑（重复 ID 只创建一条记录）
	setupPluginInstancesDB(t)

	plugin := model.Plugin{
		Slug: "dedup-plugin", Name: "去重测试插件", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		PluginID: "dedup-plugin-id", PluginFormat: "openclaw",
	}
	model.DB(context.Background()).Create(&plugin)

	user := model.User{Username: "dedup-user"}
	model.DB(context.Background()).Create(&user)

	inst := model.Instance{
		Name: "oc-dedup", InstanceId: "ins-dedup-001", UserID: user.ID, AgentType: "openclaw",
	}
	model.DB(context.Background()).Create(&inst)

	// 传入重复的 instance_id
	body := fmt.Sprintf(`{"slug":"dedup-plugin","instance_ids":[%d,%d,%d]}`,
		inst.ID, inst.ID, inst.ID)

	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute", body))

	// 应该成功（去重后只有 1 个实例）
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if got := int(resp["total"].(float64)); got != 1 {
		t.Fatalf("响应 total=%d，期望 1", got)
	}

	// task 的 Total 应该是 1（去重后）
	var task model.PluginDistributionTask
	if model.DB(context.Background()).Where("plugin_db_id = ?", plugin.ID).First(&task).Error != nil {
		t.Fatal("期望创建了 task")
	}
	if task.Total != 1 {
		t.Errorf("期望 task.Total=1（去重后），实际=%d", task.Total)
	}
}

func TestHandleDistributePlugin_SelectAllByStatusAndGroup(t *testing.T) {
	setupPluginInstancesDB(t)
	ctx := context.Background()
	db := model.DB(ctx)

	plugin := model.Plugin{
		Slug: "select-all-plugin", Name: "全选测试插件", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		PluginID: "select-all-plugin-id", PluginFormat: "openclaw",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("创建插件失败: %v", err)
	}
	group := model.UserGroup{Name: "select-all-plugin-group"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}
	groupedUser := model.User{Username: "select-all-plugin-grouped"}
	otherUser := model.User{Username: "select-all-plugin-other"}
	if err := db.Create(&[]model.User{groupedUser, otherUser}).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if err := db.Where("username = ?", groupedUser.Username).First(&groupedUser).Error; err != nil {
		t.Fatalf("查询分组用户失败: %v", err)
	}
	if err := db.Where("username = ?", otherUser.Username).First(&otherUser).Error; err != nil {
		t.Fatalf("查询其他用户失败: %v", err)
	}
	if err := db.Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: groupedUser.ID}).Error; err != nil {
		t.Fatalf("创建用户组成员失败: %v", err)
	}
	instances := []model.Instance{
		{Name: "select-all-plugin-match", InstanceId: "ins-plugin-match", UserID: groupedUser.ID, AgentType: "openclaw"},
		{Name: "select-all-plugin-other", InstanceId: "ins-plugin-other", UserID: otherUser.ID, AgentType: "openclaw"},
		{Name: "select-all-plugin-unsupported", InstanceId: "ins-plugin-unsupported", UserID: groupedUser.ID, AgentType: "unsupported"},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	previousTask := model.PluginDistributionTask{
		PluginDBID: plugin.ID, Version: "0.9.0",
		Status: model.TaskStatusCompleted, Type: model.TaskTypeDistribute, Total: 1, Success: 1,
	}
	if err := db.Create(&previousTask).Error; err != nil {
		t.Fatalf("创建历史任务失败: %v", err)
	}
	if err := db.Create(&model.PluginDistributionRecord{
		TaskID: previousTask.ID, PluginDBID: plugin.ID, InstanceID: instances[0].ID,
		InstanceCID: instances[0].InstanceId, Version: previousTask.Version,
		Status: model.RecordStatusSuccess, Type: model.TaskTypeDistribute,
	}).Error; err != nil {
		t.Fatalf("创建历史记录失败: %v", err)
	}

	body := fmt.Sprintf(
		`{"slug":%q,"select_all":true,"statuses":["outdated"],"group_ids":[%d]}`,
		plugin.Slug,
		group.ID,
	)
	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if got := int(resp["total"].(float64)); got != 1 {
		t.Fatalf("total=%d，期望 1", got)
	}
	var records []model.PluginDistributionRecord
	if err := db.Where("task_id = ?", uint(resp["task_id"].(float64))).Find(&records).Error; err != nil {
		t.Fatalf("查询下发记录失败: %v", err)
	}
	if len(records) != 1 || records[0].InstanceID != instances[0].ID {
		t.Fatalf("下发记录=%+v，期望只包含实例 %d", records, instances[0].ID)
	}
}

func TestHandleUninstallPlugin_SelectAllByStatusAndGroup(t *testing.T) {
	setupPluginInstancesDB(t)
	ctx := context.Background()
	db := model.DB(ctx)

	plugin := model.Plugin{
		Slug: "select-all-uninstall-plugin", Name: "全选卸载测试插件", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		PluginID: "select-all-uninstall-plugin-id", PluginFormat: "openclaw",
		DistributeCount: 7,
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("创建插件失败: %v", err)
	}
	group := model.UserGroup{Name: "select-all-uninstall-plugin-group"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}
	groupedUser := model.User{Username: "select-all-uninstall-plugin-grouped"}
	otherUser := model.User{Username: "select-all-uninstall-plugin-other"}
	if err := db.Create(&[]model.User{groupedUser, otherUser}).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if err := db.Where("username = ?", groupedUser.Username).First(&groupedUser).Error; err != nil {
		t.Fatalf("查询分组用户失败: %v", err)
	}
	if err := db.Where("username = ?", otherUser.Username).First(&otherUser).Error; err != nil {
		t.Fatalf("查询其他用户失败: %v", err)
	}
	if err := db.Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: groupedUser.ID}).Error; err != nil {
		t.Fatalf("创建用户组成员失败: %v", err)
	}
	instances := []model.Instance{
		{Name: "select-all-uninstall-installed", InstanceId: "ins-plugin-uninstall-installed", UserID: groupedUser.ID, AgentType: "openclaw", RuntimeUser: "root"},
		{Name: "select-all-uninstall-failed", InstanceId: "ins-plugin-uninstall-failed", UserID: groupedUser.ID, AgentType: "openclaw", RuntimeUser: "root"},
		{Name: "select-all-uninstall-install-failed", InstanceId: "ins-plugin-install-failed", UserID: groupedUser.ID, AgentType: "openclaw", RuntimeUser: "root"},
		{Name: "select-all-uninstall-uninstalled", InstanceId: "ins-plugin-uninstalled", UserID: groupedUser.ID, AgentType: "openclaw", RuntimeUser: "root"},
		{Name: "select-all-uninstall-other-group", InstanceId: "ins-plugin-uninstall-other-group", UserID: otherUser.ID, AgentType: "openclaw", RuntimeUser: "root"},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	distributeTask := model.PluginDistributionTask{
		PluginDBID: plugin.ID, Version: plugin.Version,
		Status: model.TaskStatusCompleted, Type: model.TaskTypeDistribute, Total: 4, Success: 3, Failed: 1,
	}
	if err := db.Create(&distributeTask).Error; err != nil {
		t.Fatalf("创建历史下发任务失败: %v", err)
	}
	distributeRecords := []model.PluginDistributionRecord{
		{TaskID: distributeTask.ID, PluginDBID: plugin.ID, InstanceID: instances[0].ID, InstanceCID: instances[0].InstanceId, Version: plugin.Version, Status: model.RecordStatusSuccess, Type: model.TaskTypeDistribute},
		{TaskID: distributeTask.ID, PluginDBID: plugin.ID, InstanceID: instances[1].ID, InstanceCID: instances[1].InstanceId, Version: plugin.Version, Status: model.RecordStatusSuccess, Type: model.TaskTypeDistribute},
		{TaskID: distributeTask.ID, PluginDBID: plugin.ID, InstanceID: instances[2].ID, InstanceCID: instances[2].InstanceId, Version: plugin.Version, Status: model.RecordStatusFailed, Type: model.TaskTypeDistribute},
		{TaskID: distributeTask.ID, PluginDBID: plugin.ID, InstanceID: instances[4].ID, InstanceCID: instances[4].InstanceId, Version: plugin.Version, Status: model.RecordStatusSuccess, Type: model.TaskTypeDistribute},
	}
	if err := db.Create(&distributeRecords).Error; err != nil {
		t.Fatalf("创建历史下发记录失败: %v", err)
	}
	uninstallTask := model.PluginDistributionTask{
		PluginDBID: plugin.ID, Version: plugin.Version,
		Status: model.TaskStatusCompleted, Type: model.TaskTypeUninstall, Total: 1, Failed: 1,
	}
	if err := db.Create(&uninstallTask).Error; err != nil {
		t.Fatalf("创建历史卸载任务失败: %v", err)
	}
	if err := db.Create(&model.PluginDistributionRecord{
		TaskID: uninstallTask.ID, PluginDBID: plugin.ID, InstanceID: instances[1].ID,
		InstanceCID: instances[1].InstanceId, Version: plugin.Version,
		Status: model.RecordStatusFailed, Type: model.TaskTypeUninstall,
	}).Error; err != nil {
		t.Fatalf("创建历史卸载记录失败: %v", err)
	}

	body := fmt.Sprintf(
		`{"slug":%q,"select_all":true,"group_ids":[%d]}`,
		plugin.Slug,
		group.ID,
	)
	resp := assertPluginUninstallHTTPStatus(t, body, http.StatusOK)
	if got := int(resp["total"].(float64)); got != 2 {
		t.Fatalf("total=%d，期望 2", got)
	}
	taskID := uint(resp["task_id"].(float64))
	var task model.PluginDistributionTask
	if err := db.First(&task, taskID).Error; err != nil {
		t.Fatalf("查询卸载任务失败: %v", err)
	}
	if task.Type != model.TaskTypeUninstall {
		t.Fatalf("Task.Type=%q，期望 %q", task.Type, model.TaskTypeUninstall)
	}
	var records []model.PluginDistributionRecord
	if err := db.Where("task_id = ?", taskID).Find(&records).Error; err != nil {
		t.Fatalf("查询卸载记录失败: %v", err)
	}
	gotInstanceIDs := make(map[uint]struct{}, len(records))
	for _, record := range records {
		if record.Type != model.TaskTypeUninstall {
			t.Fatalf("Record.Type=%q，期望 %q", record.Type, model.TaskTypeUninstall)
		}
		gotInstanceIDs[record.InstanceID] = struct{}{}
	}
	if len(gotInstanceIDs) != 2 {
		t.Fatalf("卸载记录=%+v，期望 2 个目标实例", records)
	}
	for _, wantID := range []uint{instances[0].ID, instances[1].ID} {
		if _, ok := gotInstanceIDs[wantID]; !ok {
			t.Fatalf("卸载记录=%+v，缺少实例 %d", records, wantID)
		}
	}

	waitPluginTask(t, taskID, 2*time.Second)
	if err := db.First(&plugin, plugin.ID).Error; err != nil {
		t.Fatalf("查询插件失败: %v", err)
	}
	if plugin.DistributeCount != 7 {
		t.Fatalf("卸载不应增加 distribute_count，实际=%d", plugin.DistributeCount)
	}
}

func TestHandleDistributePlugin_SelectAllRejectsTransitionalStatus(t *testing.T) {
	setupPluginInstancesDB(t)
	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost("/admin/plugins/distribute", `{"slug":"any","select_all":true,"statuses":["installing"]}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributePlugin_SelectAllNoTargets(t *testing.T) {
	setupPluginInstancesDB(t)
	db := model.DB(context.Background())
	plugin := model.Plugin{
		Slug: "select-all-plugin-empty", Name: "空目标插件", Version: "1.0.0",
		VersionMajor: 1, PluginID: "select-all-plugin-empty", PluginFormat: "openclaw",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("创建插件失败: %v", err)
	}
	w := httptest.NewRecorder()
	HandleDistributePlugin(w, adminJSONPost(
		"/admin/plugins/distribute",
		`{"slug":"select-all-plugin-empty","select_all":true,"statuses":["uninstalled"]}`,
	))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var taskCount int64
	if err := db.Model(&model.PluginDistributionTask{}).Count(&taskCount).Error; err != nil {
		t.Fatalf("统计任务失败: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("零目标不应创建任务，实际=%d", taskCount)
	}
}

func TestCleanupPluginSelectAllTask_RemovesPartialData(t *testing.T) {
	setupPluginInstancesDB(t)
	db := model.DB(context.Background())
	plugin := model.Plugin{
		Slug: "cleanup-select-all-plugin", Name: "清理插件", Version: "1.0.0",
		VersionMajor: 1, PluginID: "cleanup-select-all-plugin", PluginFormat: "openclaw",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("创建插件失败: %v", err)
	}
	task := model.PluginDistributionTask{
		PluginDBID: plugin.ID, Version: plugin.Version,
		Status: model.TaskStatusRunning, Type: model.TaskTypeDistribute,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	if err := db.Create(&model.PluginDistributionRecord{
		TaskID: task.ID, PluginDBID: plugin.ID, InstanceID: 1,
		InstanceCID: "ins-cleanup-plugin", Version: plugin.Version,
		Status: model.RecordStatusPending, Type: model.TaskTypeDistribute,
	}).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}
	cleanupPluginSelectAllTask(context.Background(), task.ID)
	var taskCount, recordCount int64
	if err := db.Model(&model.PluginDistributionTask{}).Where("id = ?", task.ID).Count(&taskCount).Error; err != nil {
		t.Fatalf("统计任务失败: %v", err)
	}
	if err := db.Model(&model.PluginDistributionRecord{}).Where("task_id = ?", task.ID).Count(&recordCount).Error; err != nil {
		t.Fatalf("统计记录失败: %v", err)
	}
	if taskCount != 0 || recordCount != 0 {
		t.Fatalf("清理后 task=%d record=%d，期望均为 0", taskCount, recordCount)
	}
}

func TestFavoritePlugin_MethodNotAllowed(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	req := httptest.NewRequest(http.MethodGet, "/admin/plugins/favorite", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleFavoritePlugin(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestUnfavoritePlugin_MethodNotAllowed(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	req := httptest.NewRequest(http.MethodGet, "/admin/plugins/unfavorite?id=1", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleUnfavoritePlugin(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestNormalizePluginDistributionStatuses(t *testing.T) {
	got, err := normalizePluginDistributionStatuses([]string{"failed", "uninstalled", "failed"})
	if err != nil {
		t.Fatalf("normalizePluginDistributionStatuses() error = %v", err)
	}
	if len(got) != 2 || got[0] != "failed" || got[1] != "uninstalled" {
		t.Errorf("normalizePluginDistributionStatuses() = %v, want [failed uninstalled]", got)
	}
	if _, err := normalizePluginDistributionStatuses([]string{"uninstalling"}); err == nil {
		t.Error("normalizePluginDistributionStatuses([uninstalling]) error = nil, want error")
	}
}

func TestNormalizePluginUninstallStatuses(t *testing.T) {
	got, err := normalizePluginUninstallStatuses(nil)
	if err != nil {
		t.Fatalf("normalizePluginUninstallStatuses() error = %v", err)
	}
	want := []string{"installed", "outdated", "upgrade_failed", "uninstall_failed", "uninstall_failed_old"}
	if !slices.Equal(got, want) {
		t.Fatalf("normalizePluginUninstallStatuses() = %v, want %v", got, want)
	}
	if _, err := normalizePluginUninstallStatuses([]string{"uninstalled"}); err == nil {
		t.Error("normalizePluginUninstallStatuses([uninstalled]) error = nil, want error")
	}
	if _, err := normalizePluginUninstallStatuses([]string{"failed"}); err == nil {
		t.Error("normalizePluginUninstallStatuses([failed]) error = nil, want error")
	}
	if _, err := normalizePluginUninstallStatuses([]string{"uninstalling"}); err == nil {
		t.Error("normalizePluginUninstallStatuses([uninstalling]) error = nil, want error")
	}
}

func TestCreatePluginSelectAllTask_CrossesBatchBoundary(t *testing.T) {
	setupPluginInstancesDB(t)
	ctx := context.Background()
	db := model.DB(ctx)
	plugin := model.Plugin{
		Slug:         "select-all-plugin-batches",
		Name:         "跨批次插件",
		Version:      "1.0.0",
		VersionMajor: 1,
		PluginID:     "select-all-plugin-batches",
		PluginFormat: "openclaw",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("创建插件失败: %v", err)
	}
	const targetCount = pluginDistributionBatchSize + 1
	instances := make([]model.Instance, targetCount)
	for i := range instances {
		instances[i] = model.Instance{
			Name:       "plugin-batch",
			InstanceId: fmt.Sprintf("ins-plugin-batch-%03d", i),
			AgentType:  "openclaw",
		}
	}
	if err := db.CreateInBatches(&instances, pluginDistributionBatchSize).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	task, total, err := createPluginSelectAllTask(
		ctx,
		plugin,
		model.TaskTypeDistribute,
		0,
		distributionSelection{SelectAll: true, Statuses: []string{"uninstalled"}},
	)
	if err != nil {
		t.Fatalf("createPluginSelectAllTask() error = %v", err)
	}
	if total != targetCount || task.Total != targetCount {
		t.Fatalf("createPluginSelectAllTask() total=%d task.total=%d, want %d", total, task.Total, targetCount)
	}
	var recordCount int64
	if err := db.Model(&model.PluginDistributionRecord{}).Where("task_id = ?", task.ID).Count(&recordCount).Error; err != nil {
		t.Fatalf("统计下发记录失败: %v", err)
	}
	if recordCount != targetCount {
		t.Fatalf("下发记录数=%d，期望 %d", recordCount, targetCount)
	}
}
func TestCreatePluginSelectAllTask_SearchByUsername(t *testing.T) {
	setupPluginInstancesDB(t)
	ctx := context.Background()
	db := model.DB(ctx)
	plugin := model.Plugin{
		Slug:         "select-all-plugin-search",
		Name:         "搜索插件",
		Version:      "1.0.0",
		VersionMajor: 1,
		PluginID:     "select-all-plugin-search",
		PluginFormat: "openclaw",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("创建插件失败: %v", err)
	}
	users := []model.User{
		{Username: "plugin-search-needle"},
		{Username: "plugin-search-other"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instances := []model.Instance{
		{Name: "target-one", InstanceId: "ins-target-one", UserID: users[0].ID, AgentType: "openclaw"},
		{Name: "target-two", InstanceId: "ins-target-two", UserID: users[1].ID, AgentType: "openclaw"},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	task, total, err := createPluginSelectAllTask(
		ctx,
		plugin,
		model.TaskTypeDistribute,
		0,
		distributionSelection{SelectAll: true, Statuses: []string{"uninstalled"}, Search: "needle"},
	)
	if err != nil {
		t.Fatalf("createPluginSelectAllTask(search=%q) error = %v", "needle", err)
	}
	if total != 1 || task.Total != 1 {
		t.Fatalf("createPluginSelectAllTask(search=%q) total=%d task.total=%d, want 1", "needle", total, task.Total)
	}
	var record model.PluginDistributionRecord
	if err := db.Where("task_id = ?", task.ID).First(&record).Error; err != nil {
		t.Fatalf("查询搜索结果记录失败: %v", err)
	}
	if record.InstanceID != instances[0].ID {
		t.Fatalf("createPluginSelectAllTask(search=%q) instance_id=%d, want %d", "needle", record.InstanceID, instances[0].ID)
	}
}

func TestRunPluginSelectAllTask_RecoversTaskPanic(t *testing.T) {
	setupPluginInstancesDB(t)
	ctx := context.Background()
	db := model.DB(ctx)
	plugin := model.Plugin{Slug: "panic-plugin", Name: "Panic Plugin", Version: "1.0.0", PluginID: "panic-plugin"}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("创建插件失败: %v", err)
	}
	task := model.PluginDistributionTask{
		PluginDBID: plugin.ID, Version: plugin.Version, Total: 1,
		Status: model.TaskStatusRunning, Type: model.TaskTypeDistribute,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.PluginDistributionRecord{
		TaskID: task.ID, PluginDBID: plugin.ID, InstanceID: 1, InstanceCID: "ins-panic-plugin",
		Version: plugin.Version, Status: model.RecordStatusPending, Type: model.TaskTypeDistribute,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}

	triggered := false
	const callbackName = "test:plugin-select-all-query-panic"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if !triggered && tx.Statement.Table == "plugin_distribution_records" {
			triggered = true
			panic("plugin select-all query panic")
		}
	}); err != nil {
		t.Fatalf("注册 panic callback 失败: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})

	func() {
		defer recoverPluginSelectAllTaskPanic(ctx, task)
		runPluginSelectAllTask(ctx, plugin, task)
	}()

	if !triggered {
		t.Fatal("panic callback 未触发")
	}
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if err := db.First(&record, record.ID).Error; err != nil {
		t.Fatalf("查询记录失败: %v", err)
	}
	if task.Status != model.TaskStatusCompleted || task.Failed != 1 {
		t.Fatalf("panic 后任务 status=%q failed=%d，期望 completed/1", task.Status, task.Failed)
	}
	if record.Status != model.RecordStatusFailed {
		t.Fatalf("panic 后记录 status=%q，期望 failed", record.Status)
	}
}

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"
)

// setupSMHHTTPTestEnv 启动一个 httptest.Server 模拟 SMH API，
// 同时 seed 完整 SMH 配置并将 token 写入 DB。
// 返回 testServer 和 cleanup。
func setupSMHHTTPTestEnv(t *testing.T, handler http.Handler) (*httptest.Server, func()) {
	t.Helper()
	cleanupDB := setupCosCommonTestDB(t)
	seedSMHFullyConfigured(t)

	ts := httptest.NewServer(handler)

	// 更新 SiteConfig 的 SMHEndpoint 指向 testServer
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("smh_endpoint", ts.URL).Error; err != nil {
		t.Fatalf("更新 SiteConfig.SMHEndpoint 失败: %v", err)
	}

	// 将 common space 的 admin token 写入 DB，模拟 task 已刷新 token
	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	if err := model.UpdateSMHSpaceToken(context.Background(), "common", true, "fake-access-token", expiredAt); err != nil {
		t.Fatalf("写入 common admin token 失败: %v", err)
	}

	return ts, func() {
		ts.Close()
		cleanupDB()
	}
}

// TestFindLatestSMHCommonBackup_DirectoryNotFound 覆盖 "目录不存在(404)→无备份" 分支。
func TestFindLatestSMHCommonBackup_DirectoryNotFound(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/directory/") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":"NotFound","message":"directory not found"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	fileKey, found, err := FindLatestSMHCommonBackup(context.Background(), "ins-missing", "")
	if err != nil {
		t.Errorf("404 应视为 found=false 且 err=nil，实际 err=%v", err)
	}
	if found {
		t.Error("404 时 found 应为 false")
	}
	if fileKey != "" {
		t.Errorf("404 时 fileKey 应为空，实际=%q", fileKey)
	}
}

// TestFindLatestSMHCommonBackup_ServerError 覆盖 "API 返回非 404 错误" 分支。
func TestFindLatestSMHCommonBackup_ServerError(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":"InternalError","message":"boom"}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	_, found, err := FindLatestSMHCommonBackup(context.Background(), "ins-any", "")
	if err == nil {
		t.Error("500 错误应返回 error")
	}
	if found {
		t.Error("出错时 found 应为 false")
	}
}

// TestFindLatestSMHCommonBackup_EmptyContents 覆盖 "目录存在但无匹配文件" 分支，
// 同时覆盖 resp != nil 的 for 循环遍历（空列表不进入循环体）和末尾 "len(names)==0" 分支。
func TestFindLatestSMHCommonBackup_EmptyContents(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"contents":[]}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	fileKey, found, err := FindLatestSMHCommonBackup(context.Background(), "ins-empty", "")
	if err != nil {
		t.Errorf("空目录不应返回 error，实际=%v", err)
	}
	if found {
		t.Error("空目录 found 应为 false")
	}
	if fileKey != "" {
		t.Errorf("空目录 fileKey 应为空，实际=%q", fileKey)
	}
}

// TestFindLatestSMHCommonBackup_HasFilesSingleName 覆盖 "目录有文件→选最新" 分支。
// 额外覆盖：for 循环中对 openclaw-state-*.tgz 的前缀后缀匹配、sort、拼接 fileKey。
func TestFindLatestSMHCommonBackup_HasFilesSingleName(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// 混合三类文件：早的升级备份、晚的升级备份、一个不符合命名规则的文件（应被过滤）
		w.Write([]byte(`{
			"contents":[
				{"name":"openclaw-state-20260101_000000.tgz"},
				{"name":"openclaw-state-20260501_120000.tgz"},
				{"name":"other-unrelated-file.zip"}
			]
		}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	fileKey, found, err := FindLatestSMHCommonBackup(context.Background(), "ins-ok", "")
	if err != nil {
		t.Fatalf("查询不应失败: %v", err)
	}
	if !found {
		t.Fatal("应找到备份")
	}
	expected := "backups/ins-ok/openclaw-state-20260501_120000.tgz"
	if fileKey != expected {
		t.Errorf("fileKey 应取字典序最大的那个，期望=%q 实际=%q", expected, fileKey)
	}
}

// TestFindLatestSMHCommonBackup_MultiPage 覆盖分页翻页分支。
// 第一页返回 nextMarker="page2"，第二页返回空 + 无 nextMarker，退出循环。
func TestFindLatestSMHCommonBackup_MultiPage(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		marker := r.URL.Query().Get("marker")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if marker == "" {
			w.Write([]byte(`{
				"nextMarker":"page2",
				"contents":[
					{"name":"openclaw-state-20260101_000000.tgz"}
				]
			}`))
			return
		}
		// 第二页：无 nextMarker → 退出循环
		w.Write([]byte(`{
			"contents":[
				{"name":"openclaw-state-20260301_100000.tgz"}
			]
		}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	fileKey, found, err := FindLatestSMHCommonBackup(context.Background(), "ins-paged", "")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if !found {
		t.Fatal("应找到备份")
	}
	expected := "backups/ins-paged/openclaw-state-20260301_100000.tgz"
	if fileKey != expected {
		t.Errorf("应取两页合并后的最新，期望=%q 实际=%q", expected, fileKey)
	}
}

// TestDeleteSMHCommonDirectory_Success 覆盖 Delete 成功路径。
func TestDeleteSMHCommonDirectory_Success(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	if err := DeleteSMHCommonDirectory(context.Background(), "backups/ins-del"); err != nil {
		t.Errorf("删除成功路径不应返回 error，实际=%v", err)
	}
}

// TestDeleteSMHCommonDirectory_NotFoundTreatedAsSuccess 覆盖 "404→视为已删除→nil" 分支。
func TestDeleteSMHCommonDirectory_NotFoundTreatedAsSuccess(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	if err := DeleteSMHCommonDirectory(context.Background(), "backups/ins-404"); err != nil {
		t.Errorf("404 应视为已删除返回 nil，实际=%v", err)
	}
}

// TestDeleteSMHCommonDirectory_ServerError 覆盖 "非 404 错误 → 返回 error" 分支。
func TestDeleteSMHCommonDirectory_ServerError(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	if err := DeleteSMHCommonDirectory(context.Background(), "backups/ins-500"); err == nil {
		t.Error("500 错误应返回 error")
	}
}

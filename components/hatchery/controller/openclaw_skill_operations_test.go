package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

func setupSkillOperationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupSkillInstancesDB(t)
	oldStore := Store
	Store = sessions.NewCookieStore([]byte("skill-operation-test-secret-32-byte"))
	t.Cleanup(func() { Store = oldStore })
	return db
}

func seedSkillDistributionEvent(t *testing.T, db *gorm.DB, instanceID, skillID uint, source, slug, version, action, status string) {
	t.Helper()
	task := model.SkillDistributionTask{
		SkillID: skillID, Source: source, Slug: slug, Version: version,
		Type: action, Status: model.TaskStatusCompleted,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	record := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skillID, InstanceID: instanceID,
		Version: version, Type: action, Status: status,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
}

func skillOperationTestDependencies(fetch func(context.Context, string) (string, error)) userSkillDependencies {
	deps := newUserSkillDependencies()
	if fetch != nil {
		deps.versions = newPublicSkillVersionCache(fetch, time.Now)
	}
	return deps
}

func TestEnrichDistributedSkillVersions_OnlyEnrichesRuntimeItemsWithDistributionState(t *testing.T) {
	db := setupSkillOperationTestDB(t)
	seedSkillDistributionEvent(t, db, 1, 0, model.SkillSourcePublic, "self-improving-agent", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
	seedSkillDistributionEvent(t, db, 1, 0, model.SkillSourcePublic, "db-only", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
	deps := skillOperationTestDependencies(func(context.Context, string) (string, error) { return "2.0.0", nil })

	output := `[
		{"slug":"self-improving-agent","name":"self-improvement","description":"distributed","eligible":true,"can_uninstall":true},
		{"slug":"manual","name":"Manual","description":"manual","eligible":true,"can_uninstall":true}
	]`
	items, err := enrichDistributedSkillVersions(context.Background(), 1, 1, output, deps)
	if err != nil {
		t.Fatalf("enrichDistributedSkillVersions: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items len = %d, want 2", len(items))
	}
	distributed := items[0]
	if distributed.Name != "self-improvement" || distributed.Slug != "self-improving-agent" || !distributed.CanUninstall || distributed.Version == nil || *distributed.Version != "1.0.0" || distributed.LatestVersion == nil || *distributed.LatestVersion != "2.0.0" || distributed.UpdateAvailable == nil || !*distributed.UpdateAvailable {
		t.Fatalf("Admin-distributed item = %+v", distributed)
	}
	direct := items[1]
	if direct.Slug != "manual" || !direct.CanUninstall || direct.Version != nil || direct.LatestVersion != nil || direct.UpdateAvailable != nil {
		t.Fatalf("direct runtime item leaked distribution fields: %+v", direct)
	}
}

func TestListPublicSkillLatest_BoundsConcurrentFetches(t *testing.T) {
	const skillCount = 9
	started := make(chan struct{}, skillCount)
	release := make(chan struct{})
	released := false
	t.Cleanup(func() {
		if !released {
			close(release)
		}
	})
	var active int32
	var maxActive int32
	deps := skillOperationTestDependencies(func(context.Context, string) (string, error) {
		current := atomic.AddInt32(&active, 1)
		for {
			maximum := atomic.LoadInt32(&maxActive)
			if current <= maximum || atomic.CompareAndSwapInt32(&maxActive, maximum, current) {
				break
			}
		}
		started <- struct{}{}
		<-release
		atomic.AddInt32(&active, -1)
		return "2.0.0", nil
	})

	slugs := make([]string, skillCount)
	for i := range slugs {
		slugs[i] = fmt.Sprintf("public-%d", i)
	}
	done := make(chan map[string]string, 1)
	go func() {
		done <- listPublicSkillLatest(context.Background(), slugs, deps)
	}()

	for range 8 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("public version fetches did not run concurrently")
		}
	}
	select {
	case <-started:
		t.Fatal("public version fetch concurrency exceeded limit")
	default:
	}
	close(release)
	released = true

	select {
	case versions := <-done:
		if len(versions) != skillCount {
			t.Fatalf("versions=%d, want %d", len(versions), skillCount)
		}
	case <-time.After(time.Second):
		t.Fatal("public version fetches did not complete")
	}
	if maxActive != 8 {
		t.Fatalf("max active fetches=%d, want 8", maxActive)
	}
}

func TestListPublicSkillLatest_LogsAggregatedFailures(t *testing.T) {
	var logs bytes.Buffer
	deps := skillOperationTestDependencies(func(context.Context, string) (string, error) {
		return "", errors.New("registry unavailable")
	})
	deps.logger = slog.New(slog.NewJSONHandler(&logs, nil))

	versions := listPublicSkillLatest(context.Background(), []string{"broken-a", "broken-b"}, deps)
	if len(versions) != 0 {
		t.Fatalf("versions = %v, want empty", versions)
	}

	output := logs.String()
	if strings.Count(output, "\n") != 1 {
		t.Fatalf("logs = %q, want one aggregated entry", output)
	}
	for _, fragment := range []string{
		`"level":"WARN"`,
		`"failed_count":2`,
		`"broken-a"`,
		`"broken-b"`,
		`"err":"registry unavailable"`,
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("log %q does not contain %q", output, fragment)
		}
	}
}

func TestEnrichDistributedSkillVersions_VersionComparisonAndRegistryFailure(t *testing.T) {
	db := setupSkillOperationTestDB(t)
	for i, version := range []string{"1.0.0", "2.0.0", "3.0.0", "broken"} {
		slug := fmt.Sprintf("skill-%d", i)
		seedSkillDistributionEvent(t, db, 1, 0, model.SkillSourcePublic, slug, version, model.TaskTypeDistribute, model.RecordStatusSuccess)
	}
	deps := skillOperationTestDependencies(func(_ context.Context, slug string) (string, error) {
		if strings.HasPrefix(slug, "skill-") {
			return "2.0.0", nil
		}
		return "", errors.New("registry unavailable")
	})
	seedSkillDistributionEvent(t, db, 1, 0, model.SkillSourcePublic, "offline", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)

	output := `[
		{"slug":"skill-0","name":"lower","description":"","eligible":true},
		{"slug":"skill-1","name":"equal","description":"","eligible":true},
		{"slug":"skill-2","name":"greater","description":"","eligible":true},
		{"slug":"skill-3","name":"invalid","description":"","eligible":true},
		{"slug":"offline","name":"offline","description":"","eligible":true}
	]`
	items, err := enrichDistributedSkillVersions(context.Background(), 1, 1, output, deps)
	if err != nil {
		t.Fatalf("enrichDistributedSkillVersions: %v", err)
	}
	wantUpdates := []bool{true, false, false, false, false}
	for i, want := range wantUpdates {
		if items[i].UpdateAvailable == nil || *items[i].UpdateAvailable != want {
			t.Fatalf("item %d update_available = %v, want %v", i, items[i].UpdateAvailable, want)
		}
	}
	if items[4].LatestVersion == nil || *items[4].LatestVersion != "" {
		t.Fatalf("offline latest_version = %v", items[4].LatestVersion)
	}
}

func TestEnrichDistributedSkillVersions_EnterpriseLatestMustBeVisible(t *testing.T) {
	db := setupSkillOperationTestDB(t)
	user := model.User{Username: "visibility-user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	oldSkill := model.Skill{Slug: "enterprise", Name: "Old", Version: "1.0.0", VersionMajor: 1, COSZipKey: "old.zip", VisibilityType: model.VisibilityAll}
	latestSkill := model.Skill{Slug: "enterprise", Name: "Latest", Version: "2.0.0", VersionMajor: 2, COSZipKey: "latest.zip", VisibilityType: model.VisibilityGroup}
	if err := db.Create(&oldSkill).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&latestSkill).Error; err != nil {
		t.Fatal(err)
	}
	seedSkillDistributionEvent(t, db, 1, oldSkill.ID, model.SkillSourceEnterprise, "enterprise", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)

	deps := skillOperationTestDependencies(nil)
	items, err := enrichDistributedSkillVersions(context.Background(), user.ID, 1, `[{"slug":"enterprise","name":"Enterprise","description":"","eligible":true}]`, deps)
	if err != nil {
		t.Fatalf("enrichDistributedSkillVersions: %v", err)
	}
	if items[0].LatestVersion == nil || *items[0].LatestVersion != "" || items[0].UpdateAvailable == nil || *items[0].UpdateAvailable {
		t.Fatalf("hidden latest should not fall back to older version: %+v", items[0])
	}
}

func skillOperationFormRequest(t *testing.T, method, path, username string, values url.Values) *http.Request {
	t.Helper()
	req := skillReqWithSession(t, method, path, username, values.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func decodeSkillOperationResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return response
}

func seedSkillOperationUserInstance(t *testing.T, db *gorm.DB, username, agentType, source string) (model.User, model.Instance) {
	t.Helper()
	user := model.User{Username: username, Password: "test"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	instance := model.Instance{
		Name: "skill-operation-instance", InstanceId: "ins-skill-operation", UserID: user.ID,
		AgentType: agentType, RuntimeUser: "agent", Source: source,
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	return user, instance
}

func mockSkillOperationExecution(deps *userSkillDependencies, run func(script string, params map[string]string) error) {
	deps.execution.runScript = func(_ context.Context, _ string, script string, _ uint64, _ string, _ func(string), params map[string]string) (string, error) {
		return "", run(script, params)
	}
	deps.execution.buildDownloadURL = func(context.Context, string, bool) (string, error) {
		return "https://smh.example.test/skill.zip", nil
	}
	deps.prepareDistributeItem = func(_ context.Context, item skillTaskItem) (skillTaskItem, string, *hcommon.RichError) {
		item.DownloadURL = "https://smh.example.test/public.zip"
		return item, "", nil
	}
}

func TestHandleUserUpdateSkill_AdmissionAndMissingState(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		setupSkillOperationTestDB(t)
		deps := skillOperationTestDependencies(nil)
		req := httptest.NewRequest(http.MethodPost, "/openclaw/update-skill", strings.NewReader("id=1&slug=demo"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
		if recorder.Code != http.StatusUnauthorized && recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d", recorder.Code)
		}
	})

	tests := []struct {
		name       string
		source     string
		agentType  string
		slug       string
		nonOwner   bool
		resolver   instanceStatusResolver
		wantStatus int
	}{
		{name: "empty slug", agentType: model.AgentTypeOpenClaw, wantStatus: http.StatusBadRequest, resolver: testCVMFetcher},
		{name: "invalid slug", agentType: model.AgentTypeOpenClaw, slug: "$(id)", wantStatus: http.StatusBadRequest, resolver: testCVMFetcher},
		{name: "local instance", source: model.InstanceSourceLocal, agentType: model.AgentTypeOpenClaw, slug: "demo", wantStatus: http.StatusBadRequest, resolver: testCVMFetcher},
		{name: "unsupported agent", agentType: "unknown-agent", slug: "demo", wantStatus: http.StatusForbidden, resolver: testCVMFetcher},
		{name: "not running", agentType: model.AgentTypeOpenClaw, slug: "demo", wantStatus: http.StatusConflict, resolver: &mockStatusResolverWithStatus{status: model.StatusStopped}},
		{name: "not owner", agentType: model.AgentTypeOpenClaw, slug: "demo", nonOwner: true, wantStatus: http.StatusBadRequest, resolver: testCVMFetcher},
		{name: "not installed", agentType: model.AgentTypeOpenClaw, slug: "demo", wantStatus: http.StatusNotFound, resolver: testCVMFetcher},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := skillOperationTestDependencies(nil)
			db := setupSkillOperationTestDB(t)
			user, instance := seedSkillOperationUserInstance(t, db, "request-user", tt.agentType, tt.source)
			requestUsername := user.Username
			if tt.nonOwner {
				other := model.User{Username: "other-user", Password: "test"}
				if err := db.Create(&other).Error; err != nil {
					t.Fatal(err)
				}
				requestUsername = other.Username
			}
			values := url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {tt.slug}}
			req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", requestUsername, values)
			recorder := httptest.NewRecorder()
			handleUserUpdateSkill(recorder, req, tt.resolver, deps)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			var count int64
			if err := db.Model(&model.SkillDistributionTask{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("unexpected tasks = %d", count)
			}
		})
	}
}

func TestHandleUserUpdateSkill_PublicNoopSuccessAndFailure(t *testing.T) {
	t.Run("already latest", func(t *testing.T) {
		db := setupSkillOperationTestDB(t)
		user, instance := seedSkillOperationUserInstance(t, db, "latest-user", model.AgentTypeOpenClaw, "")
		seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "2.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
		deps := skillOperationTestDependencies(func(context.Context, string) (string, error) { return "2.0.0", nil })
		var calls atomic.Int32
		mockSkillOperationExecution(&deps, func(string, map[string]string) error {
			calls.Add(1)
			return nil
		})

		req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
		recorder := httptest.NewRecorder()
		handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		response := decodeSkillOperationResponse(t, recorder)
		if response["updated"] != false || response["old_version"] != "2.0.0" || response["version"] != "2.0.0" {
			t.Fatalf("response = %+v", response)
		}
		if _, ok := response["latest_version"]; ok {
			t.Fatalf("response contains redundant latest_version: %+v", response)
		}
		if calls.Load() != 0 {
			t.Fatalf("script calls = %d", calls.Load())
		}
		var count int64
		db.Model(&model.SkillDistributionTask{}).Count(&count)
		if count != 1 {
			t.Fatalf("tasks = %d, want only seed task", count)
		}
	})

	t.Run("registry failure", func(t *testing.T) {
		db := setupSkillOperationTestDB(t)
		user, instance := seedSkillOperationUserInstance(t, db, "registry-user", model.AgentTypeOpenClaw, "")
		seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
		deps := skillOperationTestDependencies(func(context.Context, string) (string, error) { return "", errors.New("offline") })

		req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
		recorder := httptest.NewRecorder()
		handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
		if recorder.Code != http.StatusBadGateway {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		var count int64
		db.Model(&model.SkillDistributionTask{}).Count(&count)
		if count != 1 {
			t.Fatalf("tasks = %d, want only seed task", count)
		}
	})

	t.Run("preparation failure", func(t *testing.T) {
		db := setupSkillOperationTestDB(t)
		user, instance := seedSkillOperationUserInstance(t, db, "prepare-user", model.AgentTypeOpenClaw, "")
		seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
		deps := skillOperationTestDependencies(func(context.Context, string) (string, error) { return "2.0.0", nil })
		var calls atomic.Int32
		mockSkillOperationExecution(&deps, func(string, map[string]string) error {
			calls.Add(1)
			return nil
		})
		deps.prepareDistributeItem = func(_ context.Context, item skillTaskItem) (skillTaskItem, string, *hcommon.RichError) {
			return item, "", hcommon.I18nError(i18n.MsgQuerySkillFailed)
		}

		req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
		recorder := httptest.NewRecorder()
		handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
		if recorder.Code != http.StatusInternalServerError || calls.Load() != 0 {
			t.Fatalf("status=%d calls=%d body=%s", recorder.Code, calls.Load(), recorder.Body.String())
		}
		var count int64
		db.Model(&model.SkillDistributionTask{}).Count(&count)
		if count != 1 {
			t.Fatalf("tasks=%d, want only seed task", count)
		}
	})

	for _, tt := range []struct {
		name       string
		scriptErr  error
		wantStatus int
		wantRecord string
	}{
		{name: "success", wantStatus: http.StatusOK, wantRecord: model.RecordStatusSuccess},
		{name: "script failure", scriptErr: errors.New("install failed"), wantStatus: http.StatusInternalServerError, wantRecord: model.RecordStatusUpgradeFailed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := setupSkillOperationTestDB(t)
			user, instance := seedSkillOperationUserInstance(t, db, "public-"+strings.ReplaceAll(tt.name, " ", "-"), model.AgentTypeOpenClaw, "")
			seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
			deps := skillOperationTestDependencies(func(context.Context, string) (string, error) { return "2.0.0", nil })
			var scriptName string
			var scriptParams map[string]string
			mockSkillOperationExecution(&deps, func(script string, params map[string]string) error {
				scriptName, scriptParams = script, params
				return tt.scriptErr
			})

			req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
			recorder := httptest.NewRecorder()
			handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if scriptName != "install_skill_from_smh.sh" || scriptParams["skill_slug"] != "demo" || scriptParams["skill_version"] != "2.0.0" {
				t.Fatalf("script=%q params=%v", scriptName, scriptParams)
			}
			var task model.SkillDistributionTask
			if err := db.Order("id DESC").First(&task).Error; err != nil {
				t.Fatal(err)
			}
			var record model.SkillDistributionRecord
			if err := db.Order("id DESC").First(&record).Error; err != nil {
				t.Fatal(err)
			}
			if task.Status != model.TaskStatusCompleted || record.Status != tt.wantRecord {
				t.Fatalf("task=%+v record=%+v", task, record)
			}
			state, installed, err := getInstalledDistributedSkillState(context.Background(), instance.ID, "demo")
			if err != nil {
				t.Fatal(err)
			}
			if tt.scriptErr == nil {
				response := decodeSkillOperationResponse(t, recorder)
				if response["old_version"] != "1.0.0" || response["version"] != "2.0.0" {
					t.Fatalf("response=%+v", response)
				}
				if _, ok := response["latest_version"]; ok {
					t.Fatalf("response contains redundant latest_version: %+v", response)
				}
				if !installed || state.Version != "2.0.0" || task.Success != 1 {
					t.Fatalf("state=%+v installed=%v task=%+v", state, installed, task)
				}
			} else if !installed || state.Version != "1.0.0" || task.Failed != 1 {
				t.Fatalf("failed update changed state: state=%+v installed=%v task=%+v", state, installed, task)
			}
		})
	}
}

func TestHandleUserUpdateSkill_EnterpriseVisibility(t *testing.T) {
	for _, visible := range []bool{true, false} {
		name := "hidden"
		if visible {
			name = "visible"
		}
		t.Run(name, func(t *testing.T) {
			db := setupSkillOperationTestDB(t)
			user, instance := seedSkillOperationUserInstance(t, db, "enterprise-"+name, model.AgentTypeOpenClaw, "")
			oldSkill := model.Skill{Slug: "enterprise", Name: "Old", Version: "1.0.0", VersionMajor: 1, COSZipKey: "old.zip", VisibilityType: model.VisibilityAll}
			visibility := model.VisibilityGroup
			if visible {
				visibility = model.VisibilityAll
			}
			latestSkill := model.Skill{Slug: "enterprise", Name: "Latest", Version: "2.0.0", VersionMajor: 2, COSZipKey: "latest.zip", VisibilityType: visibility}
			if err := db.Create(&oldSkill).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&latestSkill).Error; err != nil {
				t.Fatal(err)
			}
			seedSkillDistributionEvent(t, db, instance.ID, oldSkill.ID, model.SkillSourceEnterprise, "enterprise", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
			deps := skillOperationTestDependencies(nil)
			mockSkillOperationExecution(&deps, func(string, map[string]string) error { return nil })

			req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"enterprise"}})
			recorder := httptest.NewRecorder()
			handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
			wantStatus := http.StatusNotFound
			if visible {
				wantStatus = http.StatusOK
			}
			if recorder.Code != wantStatus {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, wantStatus, recorder.Body.String())
			}
			var count int64
			db.Model(&model.SkillDistributionTask{}).Count(&count)
			wantTasks := int64(1)
			if visible {
				wantTasks = 2
			}
			if count != wantTasks {
				t.Fatalf("tasks=%d want=%d", count, wantTasks)
			}
		})
	}
}

func TestHandleUserUpdateSkill_EnterpriseLocksTargetVersion(t *testing.T) {
	db := setupSkillOperationTestDB(t)
	user, instance := seedSkillOperationUserInstance(t, db, "enterprise-lock", model.AgentTypeOpenClaw, "")
	oldSkill := model.Skill{Slug: "enterprise", Name: "Old", Version: "1.0.0", VersionMajor: 1, COSZipKey: "old.zip", VisibilityType: model.VisibilityAll}
	latestSkill := model.Skill{Slug: "enterprise", Name: "Latest", Version: "2.0.0", VersionMajor: 2, COSZipKey: "latest.zip", VisibilityType: model.VisibilityAll}
	if err := db.Create(&oldSkill).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&latestSkill).Error; err != nil {
		t.Fatal(err)
	}
	seedSkillDistributionEvent(t, db, instance.ID, oldSkill.ID, model.SkillSourceEnterprise, "enterprise", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)

	deps := skillOperationTestDependencies(nil)
	var lockKey string
	deps.tryLock = func(_ context.Context, key string) (*model.DistLock, error) {
		lockKey = key
		return nil, errors.New("locked")
	}

	req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"enterprise"}})
	recorder := httptest.NewRecorder()
	handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	wantKey := fmt.Sprintf("skill_dist:%d", latestSkill.ID)
	if lockKey != wantKey {
		t.Fatalf("lock key=%q, want %q", lockKey, wantKey)
	}
}

func TestHandleUserUninstallSkill_EnterpriseLocksAdminVersion(t *testing.T) {
	db := setupSkillOperationTestDB(t)
	user, instance := seedSkillOperationUserInstance(t, db, "enterprise-uninstall-lock", model.AgentTypeOpenClaw, "")
	oldSkill := model.Skill{Slug: "enterprise", Name: "Old", Version: "1.0.0", VersionMajor: 1, COSZipKey: "old.zip", VisibilityType: model.VisibilityAll}
	latestSkill := model.Skill{Slug: "enterprise", Name: "Latest", Version: "2.0.0", VersionMajor: 2, COSZipKey: "latest.zip", VisibilityType: model.VisibilityAll}
	if err := db.Create(&oldSkill).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&latestSkill).Error; err != nil {
		t.Fatal(err)
	}
	seedSkillDistributionEvent(t, db, instance.ID, oldSkill.ID, model.SkillSourceEnterprise, "enterprise", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)

	deps := skillOperationTestDependencies(nil)
	var lockKey string
	deps.tryLock = func(_ context.Context, key string) (*model.DistLock, error) {
		lockKey = key
		return nil, errors.New("locked")
	}

	req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/uninstall-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"enterprise"}})
	recorder := httptest.NewRecorder()
	handleUserUninstallSkill(recorder, req, testCVMFetcher, deps)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	wantKey := fmt.Sprintf("skill_dist:%d", latestSkill.ID)
	if lockKey != wantKey {
		t.Fatalf("lock key=%q, want %q", lockKey, wantKey)
	}
}

func TestHandleUserUpdateSkill_EnterpriseFailurePreservesUpgradeState(t *testing.T) {
	db := setupSkillOperationTestDB(t)
	user, instance := seedSkillOperationUserInstance(t, db, "enterprise-failure", model.AgentTypeOpenClaw, "")
	oldSkill := model.Skill{Slug: "enterprise", Name: "Old", Version: "1.0.0", VersionMajor: 1, COSZipKey: "old.zip", VisibilityType: model.VisibilityAll}
	latestSkill := model.Skill{Slug: "enterprise", Name: "Latest", Version: "2.0.0", VersionMajor: 2, COSZipKey: "latest.zip", VisibilityType: model.VisibilityAll}
	if err := db.Create(&oldSkill).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&latestSkill).Error; err != nil {
		t.Fatal(err)
	}
	seedSkillDistributionEvent(t, db, instance.ID, oldSkill.ID, model.SkillSourceEnterprise, "enterprise", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
	deps := skillOperationTestDependencies(nil)
	mockSkillOperationExecution(&deps, func(string, map[string]string) error {
		return errors.New("install failed")
	})

	req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"enterprise"}})
	recorder := httptest.NewRecorder()
	handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var record model.SkillDistributionRecord
	if err := db.Order("id DESC").First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.SkillID != latestSkill.ID || record.Status != model.RecordStatusUpgradeFailed {
		t.Fatalf("record=%+v", record)
	}
	state, installed, err := getInstalledDistributedSkillState(context.Background(), instance.ID, "enterprise")
	if err != nil {
		t.Fatal(err)
	}
	if !installed || state.Version != "1.0.0" {
		t.Fatalf("state=%+v installed=%v", state, installed)
	}
}

func TestHandleUserUpdateAndUninstallSkill_LockConflict(t *testing.T) {
	tests := []struct {
		name           string
		action         string
		currentVersion string
		latestVersion  string
	}{
		{name: "update", action: "update", currentVersion: "1.0.0", latestVersion: "2.0.0"},
		{name: "update already latest", action: "update", currentVersion: "2.0.0", latestVersion: "2.0.0"},
		{name: "uninstall", action: "uninstall", currentVersion: "1.0.0", latestVersion: "2.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupSkillOperationTestDB(t)
			user, instance := seedSkillOperationUserInstance(t, db, "lock-"+tt.name, model.AgentTypeOpenClaw, "")
			seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", tt.currentVersion, model.TaskTypeDistribute, model.RecordStatusSuccess)
			deps := skillOperationTestDependencies(func(context.Context, string) (string, error) { return tt.latestVersion, nil })
			deps.tryLock = func(context.Context, string) (*model.DistLock, error) {
				return nil, errors.New("locked")
			}

			req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/"+tt.action+"-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
			recorder := httptest.NewRecorder()
			if tt.action == "update" {
				handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
			} else {
				handleUserUninstallSkill(recorder, req, testCVMFetcher, deps)
			}
			if recorder.Code != http.StatusConflict {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			var count int64
			db.Model(&model.SkillDistributionTask{}).Count(&count)
			if count != 1 {
				t.Fatalf("tasks=%d, want only seed task", count)
			}
		})
	}
}

func TestHandleUserUninstallSkill_IdempotentSuccessAndFailure(t *testing.T) {
	t.Run("not installed", func(t *testing.T) {
		db := setupSkillOperationTestDB(t)
		user, instance := seedSkillOperationUserInstance(t, db, "uninstall-empty", model.AgentTypeOpenClaw, "")
		deps := skillOperationTestDependencies(nil)
		var calls atomic.Int32
		mockSkillOperationExecution(&deps, func(string, map[string]string) error {
			calls.Add(1)
			return nil
		})
		req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/uninstall-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
		recorder := httptest.NewRecorder()
		handleUserUninstallSkill(recorder, req, testCVMFetcher, deps)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		response := decodeSkillOperationResponse(t, recorder)
		if response["uninstalled"] != true || calls.Load() != 1 {
			t.Fatalf("response=%v calls=%d", response, calls.Load())
		}
		var count int64
		db.Model(&model.SkillDistributionTask{}).Count(&count)
		if count != 0 {
			t.Fatalf("tasks=%d", count)
		}
	})

	for _, tt := range []struct {
		name       string
		scriptErr  error
		wantStatus int
		wantRecord string
	}{
		{name: "success", wantStatus: http.StatusOK, wantRecord: model.RecordStatusSuccess},
		{name: "failure", scriptErr: errors.New("remove failed"), wantStatus: http.StatusInternalServerError, wantRecord: model.RecordStatusFailed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := setupSkillOperationTestDB(t)
			user, instance := seedSkillOperationUserInstance(t, db, "uninstall-"+tt.name, model.AgentTypeOpenClaw, "")
			seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
			deps := skillOperationTestDependencies(nil)
			var scriptName string
			mockSkillOperationExecution(&deps, func(script string, params map[string]string) error {
				scriptName = script
				if params["skill_slug"] != "demo" {
					t.Fatalf("params=%v", params)
				}
				return tt.scriptErr
			})
			req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/uninstall-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
			recorder := httptest.NewRecorder()
			handleUserUninstallSkill(recorder, req, testCVMFetcher, deps)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if tt.scriptErr != nil {
				wantMessage := i18n.T(req.Context(), i18n.MsgSkillDeleteFail, tt.scriptErr)
				if !strings.Contains(recorder.Body.String(), wantMessage) {
					t.Fatalf("body=%s, want message %q", recorder.Body.String(), wantMessage)
				}
			}
			if scriptName != "uninstall_skill.sh" {
				t.Fatalf("script=%q", scriptName)
			}
			var task model.SkillDistributionTask
			db.Order("id DESC").First(&task)
			var record model.SkillDistributionRecord
			db.Order("id DESC").First(&record)
			if task.Status != model.TaskStatusCompleted || record.Status != tt.wantRecord {
				t.Fatalf("task=%+v record=%+v", task, record)
			}
			state, installed, err := getInstalledDistributedSkillState(context.Background(), instance.ID, "demo")
			if err != nil {
				t.Fatal(err)
			}
			if tt.scriptErr == nil {
				if installed || state.Version != "1.0.0" {
					t.Fatalf("state=%+v installed=%v", state, installed)
				}
			} else if !installed || state.Version != "1.0.0" {
				t.Fatalf("failed uninstall changed state: %+v installed=%v", state, installed)
			}
		})
	}
}

func TestHandleUserUninstallSkill_DirectRuntimeSkill(t *testing.T) {
	for _, tt := range []struct {
		agentType string
		script    string
	}{
		{agentType: model.AgentTypeOpenClaw, script: "uninstall_skill.sh"},
		{agentType: model.AgentTypeHermes, script: "uninstall_skill_hermes.sh"},
		{agentType: model.AgentTypeLightclawACE, script: "uninstall_skill_ace.sh"},
	} {
		t.Run(tt.agentType, func(t *testing.T) {
			db := setupSkillOperationTestDB(t)
			user, instance := seedSkillOperationUserInstance(t, db, "direct-"+tt.agentType, tt.agentType, "")
			deps := skillOperationTestDependencies(nil)
			var lockKey string
			deps.tryLock = func(_ context.Context, key string) (*model.DistLock, error) {
				lockKey = key
				return &model.DistLock{}, nil
			}
			var scripts []string
			mockSkillOperationExecution(&deps, func(script string, params map[string]string) error {
				scripts = append(scripts, script)
				if params["skill_slug"] != "manual" {
					t.Fatalf("params=%v", params)
				}
				return nil
			})

			request := func() map[string]any {
				req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/uninstall-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"manual"}})
				recorder := httptest.NewRecorder()
				handleUserUninstallSkill(recorder, req, testCVMFetcher, deps)
				if recorder.Code != http.StatusOK {
					t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
				}
				return decodeSkillOperationResponse(t, recorder)
			}

			response := request()
			if response["uninstalled"] != true {
				t.Fatalf("response=%v", response)
			}
			if _, ok := response["version"]; ok {
				t.Fatalf("direct runtime response contains unknown version: %v", response)
			}
			wantLockKey := "skill_dist:" + tt.agentType + ":manual"
			if lockKey != wantLockKey {
				t.Fatalf("lock key=%q, want %q", lockKey, wantLockKey)
			}
			response = request()
			if response["uninstalled"] != true {
				t.Fatalf("repeated response=%v", response)
			}
			if len(scripts) != 2 || scripts[0] != tt.script || scripts[1] != tt.script {
				t.Fatalf("scripts=%v, want two %q calls", scripts, tt.script)
			}
			var taskCount int64
			if err := db.Model(&model.SkillDistributionTask{}).Count(&taskCount).Error; err != nil {
				t.Fatal(err)
			}
			if taskCount != 0 {
				t.Fatalf("direct uninstall created %d tasks", taskCount)
			}
			var recordCount int64
			if err := db.Model(&model.SkillDistributionRecord{}).Count(&recordCount).Error; err != nil {
				t.Fatal(err)
			}
			if recordCount != 0 {
				t.Fatalf("direct uninstall created %d records", recordCount)
			}
		})
	}

}

func TestSkillTaskExecution_RoutesScriptsByAgentType(t *testing.T) {
	cases := []struct {
		agentType     string
		installScript string
		removeScript  string
	}{
		{model.AgentTypeOpenClaw, "install_skill_from_smh.sh", "uninstall_skill.sh"},
		{model.AgentTypeHermes, "install_skill_from_smh_hermes.sh", "uninstall_skill_hermes.sh"},
		{model.AgentTypeLightclawACE, "install_skill_from_smh_ace.sh", "uninstall_skill_ace.sh"},
	}
	for _, tt := range cases {
		t.Run(tt.agentType, func(t *testing.T) {
			deps := skillOperationTestDependencies(nil)
			var scripts []string
			var params []map[string]string
			mockSkillOperationExecution(&deps, func(script string, gotParams map[string]string) error {
				scripts = append(scripts, script)
				params = append(params, gotParams)
				return nil
			})
			info := map[uint]skillInstanceInfo{1: {ID: 1, InstanceId: "ins", RuntimeUser: "agent", AgentType: tt.agentType}}
			item := skillTaskItem{Source: model.SkillSourcePublic, Slug: "demo", Version: "2.0.0", DownloadURL: "https://example.test/demo.zip"}
			_, install := buildSkillDistributeExecution(context.Background(), item, model.SkillDistributionTask{}, nil, nil, info, deps.execution)
			if err := install(context.Background(), model.SkillDistributionRecord{InstanceID: 1}); err != nil {
				t.Fatal(err)
			}
			_, uninstall := buildSkillUninstallExecution(context.Background(), item, model.SkillDistributionTask{}, nil, nil, info, deps.execution)
			if err := uninstall(context.Background(), model.SkillDistributionRecord{InstanceID: 1}); err != nil {
				t.Fatal(err)
			}
			if len(scripts) != 2 || scripts[0] != tt.installScript || scripts[1] != tt.removeScript {
				t.Fatalf("scripts=%v", scripts)
			}
			if params[0]["skill_slug"] != "demo" || params[0]["skill_version"] != "2.0.0" || params[0]["download_url"] == "" || params[1]["skill_slug"] != "demo" {
				t.Fatalf("params=%v", params)
			}
		})
	}
}

func TestSkillOperationHandlers_ExportedWrappersAndMethodGuard(t *testing.T) {
	setupSkillOperationTestDB(t)
	_, updateHandler, uninstallHandler := NewUserSkillHandlers()
	handlers := []http.HandlerFunc{updateHandler, uninstallHandler}
	for _, handler := range handlers {
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodPost, "/openclaw/skill-operation", nil))
		if recorder.Code != http.StatusUnauthorized && recorder.Code != http.StatusForbidden {
			t.Fatalf("unauthorized status=%d", recorder.Code)
		}
	}

	db := model.DB(context.Background())
	user, _ := seedSkillOperationUserInstance(t, db, "method-user", model.AgentTypeOpenClaw, "")
	for _, handler := range handlers {
		req := skillOperationFormRequest(t, http.MethodGet, "/openclaw/skill-operation", user.Username, nil)
		recorder := httptest.NewRecorder()
		handler(recorder, req)
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Fatalf("method status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	}
}

func TestSkillOperationHandlers_RecheckStateAfterLock(t *testing.T) {
	t.Run("update becomes latest", func(t *testing.T) {
		db := setupSkillOperationTestDB(t)
		user, instance := seedSkillOperationUserInstance(t, db, "recheck-update", model.AgentTypeOpenClaw, "")
		seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
		deps := skillOperationTestDependencies(func(context.Context, string) (string, error) { return "2.0.0", nil })
		deps.tryLock = func(context.Context, string) (*model.DistLock, error) {
			seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "2.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
			return &model.DistLock{}, nil
		}

		req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
		recorder := httptest.NewRecorder()
		handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
		response := decodeSkillOperationResponse(t, recorder)
		if recorder.Code != http.StatusOK || response["updated"] != false || response["version"] != "2.0.0" {
			t.Fatalf("status=%d response=%v", recorder.Code, response)
		}
	})

	t.Run("update source changes", func(t *testing.T) {
		db := setupSkillOperationTestDB(t)
		user, instance := seedSkillOperationUserInstance(t, db, "recheck-source", model.AgentTypeOpenClaw, "")
		seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
		deps := skillOperationTestDependencies(func(context.Context, string) (string, error) { return "2.0.0", nil })
		deps.tryLock = func(context.Context, string) (*model.DistLock, error) {
			seedSkillDistributionEvent(t, db, instance.ID, 99, model.SkillSourceEnterprise, "demo", "1.5.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
			return &model.DistLock{}, nil
		}

		req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/update-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
		recorder := httptest.NewRecorder()
		handleUserUpdateSkill(recorder, req, testCVMFetcher, deps)
		if recorder.Code != http.StatusConflict {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("uninstall completes concurrently", func(t *testing.T) {
		db := setupSkillOperationTestDB(t)
		user, instance := seedSkillOperationUserInstance(t, db, "recheck-remove", model.AgentTypeOpenClaw, "")
		seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "1.0.0", model.TaskTypeDistribute, model.RecordStatusSuccess)
		deps := skillOperationTestDependencies(nil)
		deps.tryLock = func(context.Context, string) (*model.DistLock, error) {
			seedSkillDistributionEvent(t, db, instance.ID, 0, model.SkillSourcePublic, "demo", "1.0.0", model.TaskTypeUninstall, model.RecordStatusSuccess)
			return &model.DistLock{}, nil
		}

		req := skillOperationFormRequest(t, http.MethodPost, "/openclaw/uninstall-skill", user.Username, url.Values{"id": {fmt.Sprint(instance.ID)}, "slug": {"demo"}})
		recorder := httptest.NewRecorder()
		handleUserUninstallSkill(recorder, req, testCVMFetcher, deps)
		response := decodeSkillOperationResponse(t, recorder)
		if recorder.Code != http.StatusOK || response["uninstalled"] != true || response["version"] != "1.0.0" {
			t.Fatalf("status=%d response=%v", recorder.Code, response)
		}
	})
}

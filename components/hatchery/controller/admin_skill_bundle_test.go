package controller

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type skillBundleRoundTripFunc func(*http.Request) (*http.Response, error)

func (f skillBundleRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type skillBundleErrReadCloser struct{}

func (skillBundleErrReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func (skillBundleErrReadCloser) Close() error { return nil }

type skillBundleFakeStorage struct {
	uploadErr error
	uploads   map[string][]byte
}

func (s *skillBundleFakeStorage) Upload(key string, data []byte, contentType string) error {
	if s.uploadErr != nil {
		return s.uploadErr
	}
	if s.uploads == nil {
		s.uploads = make(map[string][]byte)
	}
	s.uploads[key] = append([]byte(nil), data...)
	return nil
}

func (s *skillBundleFakeStorage) Delete(string, bool) error { return nil }

func (s *skillBundleFakeStorage) DeletePrefix(string, bool) error { return nil }

func (s *skillBundleFakeStorage) List(string) ([]string, error) { return nil, nil }
func initSkillBundleTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试DB失败: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SiteConfig{}, &model.SMHSpace{}, &model.SkillBundle{}, &model.BundleSkill{}, &model.Skill{}, &model.PublicSkill{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	if err := db.Create(&model.SiteConfig{SMHEnabled: 1}).Error; err != nil {
		t.Fatalf("创建SiteConfig失败: %v", err)
	}
}

func configureSkillBundleTestSMH(t *testing.T) {
	t.Helper()
	db := model.DB(context.Background())
	if err := db.Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"smh_enabled":        1,
		"smh_endpoint":       "https://smh.example.com",
		"smh_library_id":     "library",
		"smh_library_secret": "secret",
	}).Error; err != nil {
		t.Fatalf("配置 SMH 失败: %v", err)
	}
	expiresAt := time.Now().Add(time.Hour).Unix()
	spaces := []model.SMHSpace{
		{SpaceTag: "common", SpaceId: "common-space", LibraryId: "library", Purpose: "common", AdminToken: "common-admin", AdminTokenExpiredAt: expiresAt, ReadToken: "common-read", ReadTokenExpiredAt: expiresAt},
		{SpaceTag: "skillhub", SpaceId: "skillhub-space", LibraryId: "library", Purpose: "skillhub", AdminToken: "skillhub-admin", AdminTokenExpiredAt: expiresAt, ReadToken: "skillhub-read", ReadTokenExpiredAt: expiresAt},
	}
	if err := db.Create(&spaces).Error; err != nil {
		t.Fatalf("创建 SMH Space 失败: %v", err)
	}
}

func TestHandleCreateSkillBundle_MethodNotAllowed(t *testing.T) {
	initSkillBundleTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	req := httptest.NewRequest(http.MethodGet, "/admin/skill-bundles/create", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleCreateSkillBundle(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestSkillBundleHandlers_MethodNotAllowed(t *testing.T) {
	initSkillBundleTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		method  string
		path    string
	}{
		{"DeleteBundle", HandleDeleteSkillBundle, http.MethodGet, "/admin/skill-bundles/delete?id=1"},
		{"ToggleBundle", HandleToggleSkillBundle, http.MethodGet, "/admin/skill-bundles/toggle?id=1"},
		{"UpdateSkills", HandleUpdateSkillBundleSkills, http.MethodGet, "/admin/skill-bundles/update-skills?id=1"},
		{"BatchAddSkills", HandleBatchAddSkillBundleSkills, http.MethodGet, "/admin/skill-bundles/batch-add-skills"},
		{"FavoriteSkill", HandleFavoriteSkill, http.MethodGet, "/admin/skills/favorite"},
		{"UnfavoriteSkill", HandleUnfavoriteSkill, http.MethodGet, "/admin/skills/unfavorite?id=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer test-admin-token")
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s: expected 405, got %d", tt.name, w.Code)
			}
		})
	}
}

func TestHandleUpdateSkillBundleVisibility_405(t *testing.T) {
	initSkillBundleTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	req := httptest.NewRequest(http.MethodGet, "/admin/skill-bundles/update-visibility?id=1", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleUpdateSkillBundleVisibility(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleBatchAddSkillBundleSkills_Validation(t *testing.T) {
	initSkillBundleTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	tests := []struct {
		name string
		body string
		want int
	}{
		{"MissingBundleIDs", `{"skills":[{"id":1,"source":"public"}]}`, http.StatusBadRequest},
		{"MissingSkills", `{"bundle_ids":[1]}`, http.StatusBadRequest},
		{"ZeroBundleID", `{"bundle_ids":[0],"skills":[{"id":1,"source":"public"}]}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/admin/skill-bundles/batch-add-skills", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer test-admin-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			HandleBatchAddSkillBundleSkills(w, req)
			if w.Code != tt.want {
				t.Fatalf("expected %d, got %d: %s", tt.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestAddPreparedSkillsToBundle_PersistsSourceSkillset(t *testing.T) {
	initSkillBundleTestDB(t)

	db := model.DB(context.Background())
	bundle := model.SkillBundle{Name: "target"}
	if err := db.Create(&bundle).Error; err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	skills := []preparedBundleSkill{{
		Name:               "skill",
		Slug:               "skill",
		Version:            "1.0.0",
		Source:             "public",
		SourceSkillsetSlug: "finance-risk-assessment",
		SourceSkillsetName: "金融风控技能包",
		CosZipKey:          "skill-bundles/target/skill/skill-1.0.0.zip",
	}}
	var added int
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		added, err = addPreparedSkillsToBundle(tx, bundle.ID, skills)
		return err
	}); err != nil {
		t.Fatalf("addPreparedSkillsToBundle: %v", err)
	}
	if added != 1 {
		t.Fatalf("added = %d, want 1", added)
	}

	var saved model.BundleSkill
	if err := db.Where("skill_bundle_id = ?", bundle.ID).First(&saved).Error; err != nil {
		t.Fatalf("query bundle skill: %v", err)
	}
	if saved.Name != "skill" {
		t.Fatalf("name = %q, want skill", saved.Name)
	}
	if saved.SourceSkillsetSlug != "finance-risk-assessment" || saved.SourceSkillsetName != "金融风控技能包" {
		t.Fatalf("source skillset = (%q,%q), want (%q,%q)", saved.SourceSkillsetSlug, saved.SourceSkillsetName, "finance-risk-assessment", "金融风控技能包")
	}
}

func TestResolvePublicBundleSkillForAdd_BySlugResolvesVersion(t *testing.T) {
	origHTTPClient := SkillHTTPClient
	var sawDownloadRequest bool
	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "lightmake.site":
			sawDownloadRequest = true
			if req.URL.Path != "/api/v1/download" {
				t.Fatalf("download path = %q, want /api/v1/download", req.URL.Path)
			}
			if got := req.URL.Query().Get("slug"); got != "non-favorite" {
				t.Fatalf("slug query = %q, want non-favorite", got)
			}
			if got := req.URL.Query().Get("version"); got != "" {
				t.Fatalf("version query = %q, want empty", got)
			}
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://skillhub.example.com/skills/non-favorite/9.9.9.zip"}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		case "skillhub.example.com":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("zip-data")),
				Request:    req,
			}, nil
		default:
			t.Fatalf("unexpected host %q", req.URL.Host)
			return nil, nil
		}
	})}
	t.Cleanup(func() { SkillHTTPClient = origHTTPClient })

	resolved, status, richErr := resolvePublicBundleSkillForAdd("non-favorite", "latest", "Non Favorite")
	if richErr != nil {
		t.Fatalf("resolvePublicBundleSkillForAdd error: status=%d err=%v", status, richErr)
	}
	if !sawDownloadRequest {
		t.Fatal("download request was not sent")
	}
	if resolved.Name != "Non Favorite" || resolved.Slug != "non-favorite" || resolved.Version != "9.9.9" || resolved.Source != "public" {
		t.Fatalf("resolved = %+v, want name Non Favorite slug non-favorite version 9.9.9 source public", resolved)
	}
	if string(resolved.ZipData) != "zip-data" {
		t.Fatalf("zip data = %q, want zip-data", string(resolved.ZipData))
	}
}

func TestResolvePublicBundleSkillForAdd_RequiresSlug(t *testing.T) {
	_, status, richErr := resolvePublicBundleSkillForAdd("", "", "")
	if richErr == nil {
		t.Fatal("expected missing slug error")
	}
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
}

func TestBuildSkillHubPublicDownloadURL(t *testing.T) {
	noVersion := buildSkillHubPublicDownloadURL("a&b=c", "")
	if noVersion != SkillAPIBaseURL+"/api/v1/download?slug=a%26b%3Dc" {
		t.Fatalf("url without version = %q", noVersion)
	}
	withVersion := buildSkillHubPublicDownloadURL("技能 名", "v 1")
	want := SkillAPIBaseURL + "/api/v1/download?slug=%E6%8A%80%E8%83%BD+%E5%90%8D&version=v+1"
	if withVersion != want {
		t.Fatalf("url with version = %q, want %q", withVersion, want)
	}
	latestVersion := buildSkillHubPublicDownloadURL("latest-public", "latest")
	if latestVersion != SkillAPIBaseURL+"/api/v1/download?slug=latest-public" {
		t.Fatalf("url with latest version = %q", latestVersion)
	}
}

func TestPublicSkillVersionFromDownloadURL_EdgeCases(t *testing.T) {
	if got := publicSkillVersionFromDownloadURL("://bad-url"); got != "" {
		t.Fatalf("invalid url version = %q, want empty", got)
	}
	if got := publicSkillVersionFromDownloadURL("https://skillhub.example.com/api/v1/download"); got != "" {
		t.Fatalf("non-zip url version = %q, want empty", got)
	}
}

func TestResolvePublicBundleSkillForAdd_ErrorAndFallbackPaths(t *testing.T) {
	origHTTPClient := SkillHTTPClient
	t.Cleanup(func() { SkillHTTPClient = origHTTPClient })

	t.Run("DownloadStatusError", func(t *testing.T) {
		SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("missing")),
				Request:    req,
			}, nil
		})}
		_, status, richErr := resolvePublicBundleSkillForAdd("missing", "1.0.0", "Missing")
		if richErr == nil || status != http.StatusInternalServerError {
			t.Fatalf("status=%d err=%v, want 500 error", status, richErr)
		}
	})

	t.Run("DefaultVersionCannotBeResolved", func(t *testing.T) {
		SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("zip")),
				Request:    req,
			}, nil
		})}
		_, status, richErr := resolvePublicBundleSkillForAdd("no-version", "", "No Version")
		if richErr == nil || status != http.StatusBadRequest {
			t.Fatalf("status=%d err=%v, want 400 error", status, richErr)
		}
	})

	t.Run("NameFallback", func(t *testing.T) {
		SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("zip")),
				Request:    req,
			}, nil
		})}
		resolved, status, richErr := resolvePublicBundleSkillForAdd("fallback-name", "1.2.3", "")
		if richErr != nil {
			t.Fatalf("status=%d err=%v", status, richErr)
		}
		if resolved.Name != "fallback-name" || resolved.Version != "1.2.3" || string(resolved.ZipData) != "zip" {
			t.Fatalf("resolved=%+v", resolved)
		}
	})
}

func TestResolveBundleSkillForAdd_ValidationAndPublicPaths(t *testing.T) {
	initSkillBundleTestDB(t)
	origHTTPClient := SkillHTTPClient
	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		respReq := req
		if req.URL.Query().Get("version") == "" {
			finalURL, err := url.Parse("https://skillhub.example.com/skills/" + req.URL.Query().Get("slug") + "/2.0.0.zip")
			if err != nil {
				return nil, err
			}
			respReq = &http.Request{URL: finalURL}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("zip")),
			Request:    respReq,
		}, nil
	})}
	t.Cleanup(func() { SkillHTTPClient = origHTTPClient })

	if _, status, richErr := resolveBundleSkillForAdd(context.Background(), 1, "", "", "", ""); richErr == nil || status != http.StatusBadRequest {
		t.Fatalf("empty source status=%d err=%v, want 400", status, richErr)
	}
	if _, status, richErr := resolveBundleSkillForAdd(context.Background(), 1, "other", "", "", ""); richErr == nil || status != http.StatusBadRequest {
		t.Fatalf("unsupported source status=%d err=%v, want 400", status, richErr)
	}
	if _, status, richErr := resolveBundleSkillForAdd(context.Background(), 999, "public", "", "", ""); richErr == nil || status != http.StatusBadRequest {
		t.Fatalf("missing public status=%d err=%v, want 400", status, richErr)
	}
	if _, status, richErr := resolveBundleSkillForAdd(context.Background(), 999, "enterprise", "", "", ""); richErr == nil || status != http.StatusBadRequest {
		t.Fatalf("missing enterprise status=%d err=%v, want 400", status, richErr)
	}

	pub := model.PublicSkill{Name: "Pub", Slug: "pub", Version: "1.0.0"}
	if err := model.DB(context.Background()).Create(&pub).Error; err != nil {
		t.Fatalf("create public skill: %v", err)
	}
	resolved, status, richErr := resolveBundleSkillForAdd(context.Background(), pub.ID, "public", "", "", "")
	if richErr != nil {
		t.Fatalf("public resolve status=%d err=%v", status, richErr)
	}
	if resolved.Name != "Pub" || resolved.Slug != "pub" || resolved.Source != "public" {
		t.Fatalf("resolved public = %+v", resolved)
	}

	resolved, status, richErr = resolveBundleSkillForAdd(context.Background(), 0, "public", "direct-public", "1.2.3", "Direct Public")
	if richErr != nil {
		t.Fatalf("direct public resolve status=%d err=%v", status, richErr)
	}
	if resolved.Name != "Direct Public" || resolved.Slug != "direct-public" || resolved.Version != "1.2.3" || resolved.Source != "public" {
		t.Fatalf("resolved direct public = %+v", resolved)
	}

	resolved, status, richErr = resolveBundleSkillForAdd(context.Background(), 0, "public", "direct-latest", "", "Direct Latest")
	if richErr != nil {
		t.Fatalf("direct public latest resolve status=%d err=%v", status, richErr)
	}
	if resolved.Name != "Direct Latest" || resolved.Slug != "direct-latest" || resolved.Version != "2.0.0" || resolved.Source != "public" {
		t.Fatalf("resolved direct public latest = %+v", resolved)
	}

	configureSkillBundleTestSMH(t)
	enterprise := model.Skill{Name: "Enterprise", Slug: "enterprise", Version: "2.0.0", COSZipKey: "enterprise/enterprise-2.0.0.zip"}
	if err := model.DB(context.Background()).Create(&enterprise).Error; err != nil {
		t.Fatalf("create enterprise skill: %v", err)
	}
	resolved, status, richErr = resolveBundleSkillForAdd(context.Background(), enterprise.ID, "enterprise", "", "", "")
	if richErr != nil {
		t.Fatalf("enterprise resolve status=%d err=%v", status, richErr)
	}
	if resolved.Name != "Enterprise" || resolved.Slug != "enterprise" || resolved.Version != "2.0.0" || resolved.Source != "enterprise" {
		t.Fatalf("resolved enterprise = %+v", resolved)
	}
}

func TestDownloadSkillZipWithFinalURL_ErrorPaths(t *testing.T) {
	origHTTPClient := SkillHTTPClient
	t.Cleanup(func() { SkillHTTPClient = origHTTPClient })

	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("network failed")
	})}
	if _, _, richErr := downloadSkillZipWithFinalURL("https://skillhub.example.com/download", i18n.MsgDownloadPublicZipFail, i18n.MsgReadPublicZipFail); richErr == nil {
		t.Fatal("expected network error")
	}

	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       skillBundleErrReadCloser{},
			Request:    req,
		}, nil
	})}
	if _, _, richErr := downloadSkillZipWithFinalURL("https://skillhub.example.com/download", i18n.MsgDownloadPublicZipFail, i18n.MsgReadPublicZipFail); richErr == nil {
		t.Fatal("expected read error")
	}

	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("")),
			ContentLength: maxSkillBundleZipDownloadSize + 1,
			Request:       req,
		}, nil
	})}
	if _, _, richErr := downloadSkillZipWithFinalURL("https://skillhub.example.com/download", i18n.MsgDownloadPublicZipFail, i18n.MsgReadPublicZipFail); richErr == nil {
		t.Fatal("expected too large error")
	}
	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("zip")),
			Request:    req,
		}, nil
	})}
	zipData, richErr := downloadSkillZip("https://skillhub.example.com/download", i18n.MsgDownloadPublicZipFail, i18n.MsgReadPublicZipFail)
	if richErr != nil || string(zipData) != "zip" {
		t.Fatalf("downloadSkillZip data=%q err=%v", string(zipData), richErr)
	}
}

func TestPrepareBundleSkillAdds_UploadAndError(t *testing.T) {
	bundle := model.SkillBundle{Name: "target"}
	skills := []resolvedBundleSkill{{
		Name:               "Skill",
		Slug:               "skill",
		Version:            "1.0.0",
		Source:             "public",
		SourceSkillsetSlug: "skillset",
		SourceSkillsetName: "Skillset",
		ZipData:            []byte("zip"),
	}}

	storage := &skillBundleFakeStorage{}
	prepared, richErr := prepareBundleSkillAdds(storage, bundle, skills)
	if richErr != nil {
		t.Fatalf("prepareBundleSkillAdds: %v", richErr)
	}
	wantKey := "skill-bundles/target/skill/skill-1.0.0.zip"
	if string(storage.uploads[wantKey]) != "zip" {
		t.Fatalf("uploaded %q = %q, want zip", wantKey, string(storage.uploads[wantKey]))
	}
	if len(prepared) != 1 || prepared[0].CosZipKey != wantKey || prepared[0].SourceSkillsetSlug != "skillset" {
		t.Fatalf("prepared = %+v", prepared)
	}

	_, richErr = prepareBundleSkillAdds(&skillBundleFakeStorage{uploadErr: errors.New("upload failed")}, bundle, skills)
	if richErr == nil {
		t.Fatal("expected upload error")
	}
}

func TestAddPreparedSkillsToBundle_DuplicateConflict(t *testing.T) {
	initSkillBundleTestDB(t)
	db := model.DB(context.Background())
	bundle := model.SkillBundle{Name: "target"}
	if err := db.Create(&bundle).Error; err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	existing := model.BundleSkill{SkillBundleID: bundle.ID, Name: "Skill", Slug: "skill", Version: "1.0.0", Source: "public"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing skill: %v", err)
	}
	_, err := addPreparedSkillsToBundle(db, bundle.ID, []preparedBundleSkill{{
		Name:    "Skill",
		Slug:    "skill",
		Version: "1.0.0",
		Source:  "public",
	}})
	if err == nil {
		t.Fatal("expected duplicate conflict")
	}
}

func TestHandleUpdateSkillBundleSkills_NoopWithConfiguredSMH(t *testing.T) {
	initSkillBundleTestDB(t)
	configureSkillBundleTestSMH(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	bundle := model.SkillBundle{Name: "noop"}
	if err := model.DB(context.Background()).Create(&bundle).Error; err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	existing := model.BundleSkill{SkillBundleID: bundle.ID, Name: "existing", Slug: "existing", Version: "1.0.0", Source: "public"}
	if err := model.DB(context.Background()).Create(&existing).Error; err != nil {
		t.Fatalf("create bundle skill: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/skill-bundles/update-skills?id="+strconv.FormatUint(uint64(bundle.ID), 10), strings.NewReader(`{"add":[],"remove":[]}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateSkillBundleSkills(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("noop update status=%d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"skill_count":1`) {
		t.Fatalf("response should include updated skill_count=1, body=%s", w.Body.String())
	}
}

func TestHandleAdminSkillBundles_InvalidFiltersAndVersion(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?id=abc"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status=%d, want 400", w.Code)
	}

	w = httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?skill_version=1.0.0"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("version without slug status=%d, want 400", w.Code)
	}

	bundle := model.SkillBundle{Name: "版本反查包", VisibilityType: "all"}
	if err := model.DB(context.Background()).Create(&bundle).Error; err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.BundleSkill{
		SkillBundleID: bundle.ID,
		Name:          "Versioned",
		Slug:          "versioned",
		Version:       "2.0.0",
		Source:        "public",
	}).Error; err != nil {
		t.Fatalf("create bundle skill: %v", err)
	}
	w = httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?skill_slug=versioned&skill_version=2.0.0"))
	if w.Code != http.StatusOK {
		t.Fatalf("version lookup status=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleBatchAddSkillBundleSkills_EarlyBranches(t *testing.T) {

	t.Run("SMHDisabled", func(t *testing.T) {
		initSkillBundleTestDB(t)
		origToken := AdminToken
		AdminToken = "test-admin-token"
		t.Cleanup(func() { AdminToken = origToken })
		if err := model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1 = 1").Update("smh_enabled", 0).Error; err != nil {
			t.Fatalf("disable smh: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/admin/skill-bundles/batch-add-skills", strings.NewReader(`{"bundle_ids":[1],"skills":[{"slug":"skill"}]}`))
		req.Header.Set("Authorization", "Bearer test-admin-token")
		w := httptest.NewRecorder()
		HandleBatchAddSkillBundleSkills(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("smh disabled status=%d, want 403", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		initSkillBundleTestDB(t)
		origToken := AdminToken
		AdminToken = "test-admin-token"
		t.Cleanup(func() { AdminToken = origToken })
		req := httptest.NewRequest(http.MethodPost, "/admin/skill-bundles/batch-add-skills", strings.NewReader(`bad json`))
		req.Header.Set("Authorization", "Bearer test-admin-token")
		w := httptest.NewRecorder()
		HandleBatchAddSkillBundleSkills(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid json status=%d, want 400", w.Code)
		}
	})

	t.Run("BundleNotFound", func(t *testing.T) {
		initSkillBundleTestDB(t)
		origToken := AdminToken
		AdminToken = "test-admin-token"
		t.Cleanup(func() { AdminToken = origToken })
		req := httptest.NewRequest(http.MethodPost, "/admin/skill-bundles/batch-add-skills", strings.NewReader(`{"bundle_ids":[999],"skills":[{"slug":"skill"}]}`))
		req.Header.Set("Authorization", "Bearer test-admin-token")
		w := httptest.NewRecorder()
		HandleBatchAddSkillBundleSkills(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing bundle status=%d, want 404", w.Code)
		}
	})

	t.Run("DeduplicatesBundleIDsBeforeResolve", func(t *testing.T) {
		initSkillBundleTestDB(t)
		origToken := AdminToken
		AdminToken = "test-admin-token"
		t.Cleanup(func() { AdminToken = origToken })
		bundle := model.SkillBundle{Name: "target"}
		if err := model.DB(context.Background()).Create(&bundle).Error; err != nil {
			t.Fatalf("create bundle: %v", err)
		}
		origHTTPClient := SkillHTTPClient
		SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("missing")),
				Request:    req,
			}, nil
		})}
		t.Cleanup(func() { SkillHTTPClient = origHTTPClient })
		bundleID := strconv.FormatUint(uint64(bundle.ID), 10)
		body := `{"bundle_ids":[` + bundleID + `,` + bundleID + `],"skills":[{"slug":"missing"}]}`
		req := httptest.NewRequest(http.MethodPost, "/admin/skill-bundles/batch-add-skills", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-admin-token")
		w := httptest.NewRecorder()
		HandleBatchAddSkillBundleSkills(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("download failure status=%d, want 500", w.Code)
		}
	})
}

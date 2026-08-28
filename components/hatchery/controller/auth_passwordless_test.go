package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupPasswordlessLoginControllerTest(t *testing.T, snapshot hcommon.TenantSnapshot) (*gorm.DB, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.PasswordlessLoginToken{},
		&model.FeatureAllowlist{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	if snapshot.Identifier != "" {
		if err := db.Create(&model.FeatureAllowlist{
			Type:       model.FeatureAllowlistTypePasswordlessLogin,
			Identifier: snapshot.Identifier,
		}).Error; err != nil {
			t.Fatalf("allow passwordless login for test tenant: %v", err)
		}
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))

	oldStore, oldAdminToken := Store, AdminToken
	Store = sessions.NewCookieStore([]byte("passwordless-login-test-secret-key"))
	AdminToken = "root-admin-token"
	t.Cleanup(func() {
		Store = oldStore
		AdminToken = oldAdminToken
	})
	return db, hcommon.InjectTenant(context.Background(), snapshot)
}

func passwordlessJSONRequest(ctx context.Context, path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return req.WithContext(ctx)
}

func issuePasswordlessLoginLink(t *testing.T, ctx context.Context, userID uint) (string, *httptest.ResponseRecorder) {
	t.Helper()
	req := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":`+jsonNumber(userID)+`}`)
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	w := httptest.NewRecorder()
	WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("issue link: status=%d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Link      string    `json:"link"`
		ExpiresIn int       `json:"expires_in"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode issue response: %v", err)
	}
	if response.ExpiresIn != 120 || time.Until(response.ExpiresAt) <= 0 {
		t.Fatalf("unexpected expiry: %+v", response)
	}
	link, err := url.Parse(response.Link)
	if err != nil || link.Scheme != "https" || link.Path != "/passwordless-login" {
		t.Fatalf("unexpected link: %q err=%v", response.Link, err)
	}
	fragment, err := url.ParseQuery(link.Fragment)
	if err != nil {
		t.Fatalf("parse fragment: %v", err)
	}
	token := fragment.Get("passwordless_login_token")
	if len(token) != passwordlessLoginTokenLength {
		t.Fatalf("unexpected token length: %d", len(token))
	}
	return token, w
}

func jsonNumber(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

func TestPasswordlessLogin_FullFlowAndReplay(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "tenant.example.com",
	})
	user := model.User{Username: "target-user", Password: "unused", Role: "user", Identifier: "tenant-a"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	token, _ := issuePasswordlessLoginLink(t, ctx, user.ID)
	digest := sha256.Sum256([]byte(token))
	var record model.PasswordlessLoginToken
	if err := db.First(&record).Error; err != nil {
		t.Fatalf("load stored token: %v", err)
	}
	if record.TokenHash != hex.EncodeToString(digest[:]) || strings.Contains(record.TokenHash, token) {
		t.Fatalf("database must contain only token digest: %+v", record)
	}

	consumeReq := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+token+`"}`)
	consumeW := httptest.NewRecorder()
	HandlePasswordlessLogin(consumeW, consumeReq)
	if consumeW.Code != http.StatusOK {
		t.Fatalf("consume link: status=%d body=%s", consumeW.Code, consumeW.Body.String())
	}
	var response map[string]any
	if err := json.NewDecoder(consumeW.Body).Decode(&response); err != nil {
		t.Fatalf("decode consume response: %v", err)
	}
	if response["ok"] != true || response["redirect"] != "/" || response["role"] != "user" {
		t.Fatalf("unexpected consume response: %+v", response)
	}
	cookies := consumeW.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "hatchery-session" {
		t.Fatalf("missing session cookie: %+v", cookies)
	}
	sessionReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	sessionReq.AddCookie(cookies[0])
	session := getSession(sessionReq)
	if session.Values["username"] != user.Username || session.Values["role"] != user.Role || session.Values["identifier"] != "tenant-a" {
		t.Fatalf("unexpected session values: %+v", session.Values)
	}

	replayReq := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+token+`"}`)
	replayW := httptest.NewRecorder()
	HandlePasswordlessLogin(replayW, replayReq)
	if replayW.Code != http.StatusUnauthorized {
		t.Fatalf("replay status=%d body=%s", replayW.Code, replayW.Body.String())
	}
}

func TestPasswordlessLogin_IssueAuthorizationAndAllowlist(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})
	user := model.User{Username: "target-user", Role: "user", Identifier: "tenant-a"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	for _, test := range []struct {
		name       string
		authorizer string
		wantStatus int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "invalid", authorizer: "Bearer invalid", wantStatus: http.StatusUnauthorized},
		{name: "process admin token", authorizer: "Bearer root-admin-token", wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":`+jsonNumber(user.ID)+`}`)
			if test.authorizer != "" {
				req.Header.Set("Authorization", test.authorizer)
			}
			w := httptest.NewRecorder()
			WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(w, req)
			if w.Code != test.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", w.Code, test.wantStatus, w.Body.String())
			}
		})
	}

	if err := db.Where(
		"type = ? AND identifier = ?",
		model.FeatureAllowlistTypePasswordlessLogin,
		"tenant-a",
	).Delete(&model.FeatureAllowlist{}).Error; err != nil {
		t.Fatalf("remove tenant allowlist: %v", err)
	}
	req := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":`+jsonNumber(user.ID)+`}`)
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	w := httptest.NewRecorder()
	WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("empty allowlist status=%d body=%s", w.Code, w.Body.String())
	}

	if err := db.Create(&model.FeatureAllowlist{
		Type:       model.FeatureAllowlistTypePasswordlessLogin,
		Identifier: "tenant-b",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	req = passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":`+jsonNumber(user.ID)+`}`)
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	w = httptest.NewRecorder()
	WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-matching allowlist status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestPasswordlessLogin_ValidationAndDeletedUser(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})
	user := model.User{Username: "target-user", Role: "admin", Identifier: "tenant-a"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	for _, body := range []string{`{}`, `{"token":"short"}`, `{"token":"x","extra":true}`} {
		req := passwordlessJSONRequest(ctx, "/auth/passwordless-login", body)
		w := httptest.NewRecorder()
		HandlePasswordlessLogin(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body=%s status=%d response=%s", body, w.Code, w.Body.String())
		}
	}

	token, _ := issuePasswordlessLoginLink(t, ctx, user.ID)
	if err := db.Delete(&user).Error; err != nil {
		t.Fatalf("soft delete user: %v", err)
	}
	req := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+token+`"}`)
	w := httptest.NewRecorder()
	HandlePasswordlessLogin(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("deleted user status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestPasswordlessLogin_IssueAuthenticationMatrix(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})
	target := model.User{Username: "target-user", Role: "user", Identifier: "tenant-a"}
	adminToken, userToken := "hk-admin-token", "hk-user-token"
	adminUser := model.User{Username: "api-admin", Role: "admin", Identifier: "tenant-a", APIToken: &adminToken}
	normalUser := model.User{Username: "api-user", Role: "user", Identifier: "tenant-a", APIToken: &userToken}
	for _, user := range []*model.User{&target, &adminUser, &normalUser} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Username, err)
		}
	}

	for _, test := range []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "admin user API token", token: adminToken, wantStatus: http.StatusOK},
		{name: "normal user API token", token: userToken, wantStatus: http.StatusForbidden},
		{name: "process admin token", token: AdminToken, wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":`+jsonNumber(target.ID)+`}`)
			req.Header.Set("Authorization", "Bearer "+test.token)
			w := httptest.NewRecorder()
			WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(w, req)
			if w.Code != test.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", w.Code, test.wantStatus, w.Body.String())
			}
		})
	}

	seedReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	seedW := httptest.NewRecorder()
	session := getSession(seedReq)
	session.Values["username"] = adminUser.Username
	session.Values["role"] = "admin"
	if err := session.Save(seedReq, seedW); err != nil {
		t.Fatalf("save admin session: %v", err)
	}
	req := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":`+jsonNumber(target.ID)+`}`)
	req.AddCookie(seedW.Result().Cookies()[0])
	w := httptest.NewRecorder()
	WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin session status=%d body=%s", w.Code, w.Body.String())
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var updated int64
		if err := db.Model(&model.User{}).
			Where("id IN ? AND api_token_last_used_at IS NOT NULL", []uint{adminUser.ID, normalUser.ID}).
			Count(&updated).Error; err == nil && updated == 2 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("API token last-used updates did not finish")
}

func TestPasswordlessLogin_IssueValidationAndUnavailableUsers(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})
	active := model.User{Username: "active-user", Role: "user", Identifier: "tenant-a"}
	deleted := model.User{Username: "deleted-user", Role: "user", Identifier: "tenant-a"}
	if err := db.Create(&active).Error; err != nil {
		t.Fatalf("create active user: %v", err)
	}
	if err := db.Create(&deleted).Error; err != nil {
		t.Fatalf("create deleted user: %v", err)
	}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatalf("delete user: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/admin/passwordless-login-link", nil).WithContext(ctx)
	getW := httptest.NewRecorder()
	WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(getW, getReq)
	if getW.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d body=%s", getW.Code, getW.Body.String())
	}

	for _, test := range []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "invalid JSON", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing user id", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "unknown user", body: `{"user_id":999999}`, wantStatus: http.StatusNotFound},
		{name: "deleted user", body: `{"user_id":` + jsonNumber(deleted.ID) + `}`, wantStatus: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", test.body)
			req.Header.Set("Authorization", "Bearer "+AdminToken)
			w := httptest.NewRecorder()
			WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(w, req)
			if w.Code != test.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", w.Code, test.wantStatus, w.Body.String())
			}
		})
	}

	var count int64
	if err := db.Model(&model.PasswordlessLoginToken{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("invalid issuance created tokens: count=%d err=%v", count, err)
	}
}

func TestPasswordlessLogin_MissingTrustedDomain(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{Identifier: "tenant-a"})
	user := model.User{Username: "target-user", Role: "user", Identifier: "tenant-a"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	req := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":`+jsonNumber(user.ID)+`}`)
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	w := httptest.NewRecorder()
	WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var count int64
	if err := db.Model(&model.PasswordlessLoginToken{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("missing domain created tokens: count=%d err=%v", count, err)
	}
}

func TestPasswordlessLogin_RequiresJSONContentType(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})
	user := model.User{Username: "target-user", Role: "user", Identifier: "tenant-a"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	token := strings.Repeat("C", passwordlessLoginTokenLength)
	if _, err := model.CreatePasswordlessLoginToken(
		ctx,
		hashPasswordlessLoginToken(token),
		user.ID,
		time.Now().UTC().Add(passwordlessLoginTTL),
	); err != nil {
		t.Fatalf("create token: %v", err)
	}

	issueReq := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":`+jsonNumber(user.ID)+`}`)
	issueReq.Header.Set("Authorization", "Bearer "+AdminToken)
	issueReq.Header.Set("Content-Type", "text/plain")
	issueW := httptest.NewRecorder()
	WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(issueW, issueReq)
	if issueW.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("text/plain issue status=%d body=%s", issueW.Code, issueW.Body.String())
	}

	validIssueReq := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":`+jsonNumber(user.ID)+`}`)
	validIssueReq.Header.Set("Authorization", "Bearer "+AdminToken)
	validIssueReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	validIssueW := httptest.NewRecorder()
	WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(validIssueW, validIssueReq)
	if validIssueW.Code != http.StatusOK {
		t.Fatalf("JSON with charset issue status=%d body=%s", validIssueW.Code, validIssueW.Body.String())
	}

	for _, contentType := range []string{"text/plain", "application/x-www-form-urlencoded", ""} {
		t.Run("reject "+contentType, func(t *testing.T) {
			req := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+token+`"}`)
			if contentType == "" {
				req.Header.Del("Content-Type")
			} else {
				req.Header.Set("Content-Type", contentType)
			}
			w := httptest.NewRecorder()
			HandlePasswordlessLogin(w, req)
			if w.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("contentType=%q status=%d body=%s", contentType, w.Code, w.Body.String())
			}

			var count int64
			if err := db.Model(&model.PasswordlessLoginToken{}).
				Where("token_hash = ?", hashPasswordlessLoginToken(token)).
				Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("rejected request consumed token: count=%d err=%v", count, err)
			}
		})
	}

	consumeReq := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+token+`"}`)
	consumeReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	consumeW := httptest.NewRecorder()
	HandlePasswordlessLogin(consumeW, consumeReq)
	if consumeW.Code != http.StatusOK {
		t.Fatalf("JSON with charset consume status=%d body=%s", consumeW.Code, consumeW.Body.String())
	}
}

func TestPasswordlessLogin_ConsumptionValidationAndUniformErrors(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})

	getReq := httptest.NewRequest(http.MethodGet, "/auth/passwordless-login", nil).WithContext(ctx)
	getW := httptest.NewRecorder()
	HandlePasswordlessLogin(getW, getReq)
	if getW.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d body=%s", getW.Code, getW.Body.String())
	}

	for _, body := range []string{`{`, `{}`, `{"token":"short"}`, `{"token":"` + strings.Repeat("x", 44) + `"}`} {
		req := passwordlessJSONRequest(ctx, "/auth/passwordless-login", body)
		w := httptest.NewRecorder()
		HandlePasswordlessLogin(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body=%s status=%d response=%s", body, w.Code, w.Body.String())
		}
	}

	randomToken := strings.Repeat("A", passwordlessLoginTokenLength)
	expiredToken := strings.Repeat("B", passwordlessLoginTokenLength)
	if _, err := model.CreatePasswordlessLoginToken(ctx, hashPasswordlessLoginToken(expiredToken), 1, time.Now().UTC().Add(-24*time.Hour)); err != nil {
		t.Fatalf("create expired token: %v", err)
	}
	var messages []string
	for _, token := range []string{randomToken, expiredToken} {
		req := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+token+`"}`)
		w := httptest.NewRecorder()
		HandlePasswordlessLogin(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("token status=%d body=%s", w.Code, w.Body.String())
		}
		var response struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("decode error response: %v", err)
		}
		messages = append(messages, response.Error)
	}
	if messages[0] == "" || messages[0] != messages[1] {
		t.Fatalf("invalid token errors differ: %+v", messages)
	}
	var expiredCount int64
	if err := db.Model(&model.PasswordlessLoginToken{}).Where("token_hash = ?", hashPasswordlessLoginToken(expiredToken)).Count(&expiredCount).Error; err != nil || expiredCount != 1 {
		t.Fatalf("expired token should not be deleted: count=%d err=%v", expiredCount, err)
	}
}

func TestPasswordlessLogin_AllowlistRevokedBeforeConsumption(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})
	user := model.User{Username: "target-user", Role: "user", Identifier: "tenant-a"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, _ := issuePasswordlessLoginLink(t, ctx, user.ID)
	allowlist := model.FeatureAllowlist{
		Type:       model.FeatureAllowlistTypePasswordlessLogin,
		Identifier: "tenant-a",
	}
	if err := db.Where(
		"type = ? AND identifier = ?",
		allowlist.Type,
		allowlist.Identifier,
	).Delete(&model.FeatureAllowlist{}).Error; err != nil {
		t.Fatalf("revoke tenant allowlist: %v", err)
	}

	consume := func() *httptest.ResponseRecorder {
		req := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+token+`"}`)
		w := httptest.NewRecorder()
		HandlePasswordlessLogin(w, req)
		return w
	}
	if w := consume(); w.Code != http.StatusForbidden {
		t.Fatalf("blocked status=%d body=%s", w.Code, w.Body.String())
	}
	var count int64
	if err := db.Model(&model.PasswordlessLoginToken{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("blocked consume changed token: count=%d err=%v", count, err)
	}
	if err := db.Create(&allowlist).Error; err != nil {
		t.Fatalf("restore tenant allowlist: %v", err)
	}
	if w := consume(); w.Code != http.StatusOK {
		t.Fatalf("restored status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestPasswordlessLogin_OverridesExistingSessionIdentity(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})
	target := model.User{Username: "target-user", Role: "admin", Identifier: "tenant-a"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, _ := issuePasswordlessLoginLink(t, ctx, target.ID)

	seedReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	seedW := httptest.NewRecorder()
	oldSession := getSession(seedReq)
	oldSession.Values["username"] = "old-user"
	oldSession.Values["role"] = "user"
	oldSession.Values["oneid_sid"] = "old-sid"
	oldSession.Values["oneid_sub"] = "old-sub"
	oldSession.Values["oneid_amr"] = "old-amr"
	oldSession.Values["user_id"] = uint(99)
	oldSession.Values["login_failures"] = 3
	if err := oldSession.Save(seedReq, seedW); err != nil {
		t.Fatalf("save old session: %v", err)
	}

	req := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+token+`"}`)
	req.AddCookie(seedW.Result().Cookies()[0])
	w := httptest.NewRecorder()
	HandlePasswordlessLogin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("consume status=%d body=%s", w.Code, w.Body.String())
	}
	sessionReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	sessionReq.AddCookie(w.Result().Cookies()[0])
	session := getSession(sessionReq)
	if session.Values["username"] != target.Username || session.Values["role"] != "admin" {
		t.Fatalf("target identity not established: %+v", session.Values)
	}
	for _, key := range []string{"oneid_sid", "oneid_sub", "oneid_amr", "user_id", "login_failures"} {
		if _, exists := session.Values[key]; exists {
			t.Fatalf("stale session key %q remains: %+v", key, session.Values)
		}
	}
}

func TestHandleLogin_LocalSessionRegression(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{Identifier: "tenant-a"})
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := model.User{
		Username:   "password-user",
		Password:   string(passwordHash),
		Role:       "user",
		Identifier: "tenant-a",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	body := url.Values{"username": {user.Username}, "password": {"correct-password"}}.Encode()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleLogin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", w.Code, w.Body.String())
	}
	sessionReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	sessionReq.AddCookie(w.Result().Cookies()[0])
	session := getSession(sessionReq)
	if session.Values["username"] != user.Username || session.Values["role"] != user.Role || session.Values["identifier"] != "tenant-a" {
		t.Fatalf("unexpected session: %+v", session.Values)
	}
}

func TestPasswordlessLogin_AuditRules(t *testing.T) {
	tests := map[string]auditRule{
		"/admin/passwordless-login-link": {Action: "passwordless_login_link_create", Resource: "session"},
		"/auth/passwordless-login":       {Action: "passwordless_login", Resource: "session"},
	}
	for path, expected := range tests {
		if actual, ok := auditRules[path]; !ok || actual != expected {
			t.Fatalf("path=%s rule=%+v exists=%v", path, actual, ok)
		}
	}
}

func TestPasswordlessLogin_LocalizedErrors(t *testing.T) {
	db, ctx := setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})
	if err := db.Where(
		"type = ? AND identifier = ?",
		model.FeatureAllowlistTypePasswordlessLogin,
		"tenant-a",
	).Delete(&model.FeatureAllowlist{}).Error; err != nil {
		t.Fatalf("remove tenant allowlist: %v", err)
	}

	for _, test := range []struct {
		name     string
		language string
		want     string
	}{
		{name: "Chinese", language: "zh-CN", want: "免登录功能未对当前租户开放"},
		{name: "English", language: "en-US", want: "Passwordless login is not enabled for this tenant"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+strings.Repeat("A", passwordlessLoginTokenLength)+`"}`)
			req.Header.Set("Accept-Language", test.language)
			w := httptest.NewRecorder()
			I18nMiddleware(http.HandlerFunc(HandlePasswordlessLogin)).ServeHTTP(w, req)
			if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), test.want) {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestPasswordlessLogin_HelperValidation(t *testing.T) {
	if _, err := passwordlessLoginFeatureAllowed(httptest.NewRequest(http.MethodPost, "/", nil)); err == nil {
		t.Fatal("missing tenant snapshot should fail")
	}
	if _, err := passwordlessLoginLink("http://tenant.example.com", strings.Repeat("A", passwordlessLoginTokenLength)); err == nil {
		t.Fatal("non-HTTPS tenant domain should fail")
	}

	for _, test := range []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "valid", body: `{"token":"value"}`},
		{name: "multiple values", body: `{"token":"value"} {}`, wantErr: true},
		{name: "malformed trailing value", body: `{"token":"value"} {`, wantErr: true},
		{name: "oversized", body: `{"token":"` + strings.Repeat("x", passwordlessLoginBodyLimit) + `"}`, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			w := httptest.NewRecorder()
			var input passwordlessLoginRequest
			err := decodePasswordlessLoginJSON(w, req, &input)
			if (err != nil) != test.wantErr {
				t.Fatalf("error=%v wantErr=%v", err, test.wantErr)
			}
		})
	}
}

func TestPasswordlessLogin_MissingTenantSnapshot(t *testing.T) {
	_, _ = setupPasswordlessLoginControllerTest(t, hcommon.TenantSnapshot{
		Identifier: "tenant-a",
		Domain:     "https://tenant.example.com",
	})
	ctx := context.Background()

	issueReq := passwordlessJSONRequest(ctx, "/admin/passwordless-login-link", `{"user_id":1}`)
	issueReq.Header.Set("Authorization", "Bearer "+AdminToken)
	issueW := httptest.NewRecorder()
	WithOpenAPI(HandleAdminCreatePasswordlessLoginLink)(issueW, issueReq)
	if issueW.Code != http.StatusInternalServerError {
		t.Fatalf("issue status=%d body=%s", issueW.Code, issueW.Body.String())
	}

	consumeReq := passwordlessJSONRequest(ctx, "/auth/passwordless-login", `{"token":"`+strings.Repeat("A", passwordlessLoginTokenLength)+`"}`)
	consumeW := httptest.NewRecorder()
	HandlePasswordlessLogin(consumeW, consumeReq)
	if consumeW.Code != http.StatusInternalServerError {
		t.Fatalf("consume status=%d body=%s", consumeW.Code, consumeW.Body.String())
	}
}

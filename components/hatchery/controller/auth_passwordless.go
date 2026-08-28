package controller

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

const (
	passwordlessLoginTokenBytes  = 32
	passwordlessLoginTokenLength = 43
	passwordlessLoginTTL         = 2 * time.Minute
	passwordlessLoginBodyLimit   = 1024
)

type passwordlessLoginLinkRequest struct {
	UserID uint `json:"user_id"`
}

type passwordlessLoginRequest struct {
	Token string `json:"token"`
}

func passwordlessLoginFeatureAllowed(r *http.Request) (bool, error) {
	snapshot, ok := hcommon.GetTenantSnapshot(r.Context())
	if !ok {
		return false, errors.New("tenant snapshot missing")
	}
	return model.IsFeatureAllowed(r.Context(), model.FeatureAllowlistTypePasswordlessLogin, snapshot.Identifier)
}

func decodePasswordlessLoginJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, passwordlessLoginBodyLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func generatePasswordlessLoginToken() (string, string, error) {
	raw := make([]byte, passwordlessLoginTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(digest[:]), nil
}

func hashPasswordlessLoginToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func passwordlessLoginLink(domain, token string) (string, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "", errors.New("tenant domain missing")
	}
	if !strings.Contains(domain, "://") {
		domain = "https://" + domain
	}
	u, err := url.Parse(domain)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return "", errors.New("tenant domain must be an absolute HTTPS URL")
	}
	u.Path = "/passwordless-login"
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = url.Values{"passwordless_login_token": []string{token}}.Encode()
	return u.String(), nil
}

// HandleAdminCreatePasswordlessLoginLink creates a two-minute, one-use login
// link for a user in the current tenant. It uses the same administrator
// authentication contract as other /admin APIs.
func HandleAdminCreatePasswordlessLoginLink(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	if !requireJSONContentType(w, r) {
		return
	}

	allowed, err := passwordlessLoginFeatureAllowed(r)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	if !allowed {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgPasswordlessLoginFeatureUnavailable))
		return
	}

	var input passwordlessLoginLinkRequest
	if err := decodePasswordlessLoginJSON(w, r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	if input.UserID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "user_id"))
		return
	}

	var user model.User
	if err := model.DB(r.Context()).First(&user, input.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	snapshot, ok := hcommon.GetTenantSnapshot(r.Context())
	if !ok {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalError))
		return
	}
	token, tokenHash, err := generatePasswordlessLoginToken()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	link, err := passwordlessLoginLink(snapshot.Domain, token)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(passwordlessLoginTTL)
	if err := model.DeleteExpiredPasswordlessLoginTokens(r.Context(), now); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	if _, err := model.CreatePasswordlessLoginToken(r.Context(), tokenHash, user.ID, expiresAt); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]any{
		"link":       link,
		"expires_in": int(passwordlessLoginTTL.Seconds()),
		"expires_at": expiresAt,
	})
}

// HandlePasswordlessLogin consumes a one-use token and replaces any existing
// local or OneID identity in the session with the target local user.
func HandlePasswordlessLogin(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireJSONContentType(w, r) {
		return
	}

	var input passwordlessLoginRequest
	if err := decodePasswordlessLoginJSON(w, r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	if len(input.Token) != passwordlessLoginTokenLength {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "token"))
		return
	}

	allowed, err := passwordlessLoginFeatureAllowed(r)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	if !allowed {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgPasswordlessLoginFeatureUnavailable))
		return
	}

	record, err := model.ConsumePasswordlessLoginToken(r.Context(), hashPasswordlessLoginToken(input.Token), time.Now().UTC())
	if err != nil {
		if errors.Is(err, model.ErrPasswordlessLoginTokenInvalid) {
			writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgPasswordlessLoginTokenInvalid))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	var user model.User
	if err := model.DB(r.Context()).First(&user, record.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgPasswordlessLoginTokenInvalid))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	if err := establishLocalSession(w, r, &user); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]any{
		"ok":       true,
		"redirect": "/",
		"role":     user.Role,
	})
}

package controller

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/sessions"
)

func TestRegions_SingaporeAndJakartaMetadata(t *testing.T) {
	tests := []struct {
		region       string
		zones        []string
		name         string
		nameEn       string
		shortName    string
		shortNameEn  string
		id           int
		timezoneName string
	}{
		{
			region:       "ap-singapore",
			zones:        []string{"ap-singapore-1", "ap-singapore-2", "ap-singapore-3", "ap-singapore-4"},
			name:         "亚太东南（新加坡）",
			nameEn:       "Southeast Asia (Singapore)",
			shortName:    "新加坡",
			shortNameEn:  "Singapore",
			id:           90,
			timezoneName: "Asia/Singapore",
		},
		{
			region:       "ap-jakarta",
			zones:        []string{"ap-jakarta-1", "ap-jakarta-2"},
			name:         "亚太东南（雅加达）",
			nameEn:       "Southeast Asia (Jakarta)",
			shortName:    "雅加达",
			shortNameEn:  "Jakarta",
			id:           72,
			timezoneName: "Asia/Jakarta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			region, ok := Regions[tt.region]
			if !ok {
				t.Fatalf("missing region %q", tt.region)
			}
			if !reflect.DeepEqual(region.Zones, tt.zones) {
				t.Errorf("zones = %v, want %v", region.Zones, tt.zones)
			}
			if region.Name != tt.name || region.NameEn != tt.nameEn {
				t.Errorf("names = (%q, %q), want (%q, %q)", region.Name, region.NameEn, tt.name, tt.nameEn)
			}
			if region.ShortName != tt.shortName || region.ShortNameEn != tt.shortNameEn {
				t.Errorf("short names = (%q, %q), want (%q, %q)", region.ShortName, region.ShortNameEn, tt.shortName, tt.shortNameEn)
			}
			if region.ID != tt.id {
				t.Errorf("id = %d, want %d", region.ID, tt.id)
			}
			if region.Timezone != tt.timezoneName {
				t.Errorf("timezone = %q, want %q", region.Timezone, tt.timezoneName)
			}
		})
	}
}

func TestTokenHandlers_MethodNotAllowed(t *testing.T) {
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		method  string
	}{
		{"GetAPIToken", HandleGetAPIToken, http.MethodPost},
		{"CreateAPIToken", HandleCreateAPIToken, http.MethodGet},
		{"ResetAPIToken", HandleResetAPIToken, http.MethodGet},
		{"RevokeAPIToken", HandleRevokeAPIToken, http.MethodGet},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/token", nil)
			req.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405, got %d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandlers_MethodNotAllowed(t *testing.T) {
	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		method  string
		path    string
	}{
		{"Login", HandleLogin, http.MethodGet, "/login"},
		{"ChangePassword", HandleChangePassword, http.MethodGet, "/change-password"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s: expected 405, got %d", tt.name, w.Code)
			}
		})
	}
}

func TestChangePasswordHandlers_JSONGuards(t *testing.T) {
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	t.Run("post_json_unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/change-password", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		HandleChangePassword(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

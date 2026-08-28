package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudProxyHandlers_MethodNotAllowed(t *testing.T) {
	initAdminTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		path    string
	}{
		{"query", HandleCloudProxyQuery, "/admin/cloud/query/cvm"},
		{"mutate", HandleCloudProxyMutate, "/admin/cloud/mutate/cvm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer test-admin-token")
			req.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected 405, got %d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

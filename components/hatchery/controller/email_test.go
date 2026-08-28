package controller

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"
)

func TestEmailLoginURLPrefersTenantSnapshot(t *testing.T) {
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "https://tenant.example.com"})
	got := emailLoginURL(ctx, model.SiteConfig{Domain: "https://config.example.com"})
	if got != "https://tenant.example.com" {
		t.Fatalf("emailLoginURL = %q, want snapshot domain", got)
	}
}

func TestEmailLoginURLFallsBackToSiteConfig(t *testing.T) {
	got := emailLoginURL(context.Background(), model.SiteConfig{Domain: " https://config.example.com "})
	if got != "https://config.example.com" {
		t.Fatalf("emailLoginURL = %q, want trimmed config domain", got)
	}
}

func TestEmailLoginURLForRequestUsesExtractHost(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/admin/create", nil)
	r.Host = "Custom.Example.COM:8443"
	got := emailLoginURLForRequest(r)
	if got != "https://custom.example.com" {
		t.Fatalf("emailLoginURLForRequest = %q, want current request host URL", got)
	}
}

func TestSendEmailPreservesProvidedLoginURL(t *testing.T) {
	setupMemoryProDB(t)
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "https://tenant.example.com"})
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("id = ?", 1).Updates(map[string]any{
		"name":               "TestApp",
		"c_vm_secret_id":     "fake-secret-id",
		"c_vm_secret_key":    "fake-secret-key",
		"memory_tdai_enable": false,
	}).Error; err != nil {
		t.Fatalf("update site config: %v", err)
	}

	var sawLoginURL bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		sawLoginURL = bytes.Contains(body, []byte("https://custom.example.com"))
		_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-test"}}`))
	}))
	defer srv.Close()

	if err := sendEmail(ctx, "to@example.com", emailTypeWelcome, "ap-guangzhou", srv.URL, map[string]any{"login_url": "https://custom.example.com"}); err != nil {
		t.Fatalf("sendEmail: %v", err)
	}
	if !sawLoginURL {
		t.Fatal("expected login_url to be sent in action parameters")
	}
}

func TestSendEmailFallsBackToTenantLoginURLWithNilParams(t *testing.T) {
	setupMemoryProDB(t)
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "https://tenant.example.com"})
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("id = ?", 1).Updates(map[string]any{
		"name":               "TestApp",
		"c_vm_secret_id":     "fake-secret-id",
		"c_vm_secret_key":    "fake-secret-key",
		"memory_tdai_enable": false,
	}).Error; err != nil {
		t.Fatalf("update site config: %v", err)
	}

	var sawLoginURL bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		sawLoginURL = bytes.Contains(body, []byte("https://tenant.example.com"))
		_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-test"}}`))
	}))
	defer srv.Close()

	if err := sendEmail(ctx, "to@example.com", emailTypeWelcome, "ap-guangzhou", srv.URL, nil); err != nil {
		t.Fatalf("sendEmail: %v", err)
	}
	if !sawLoginURL {
		t.Fatal("expected fallback login_url to be sent in action parameters")
	}
}

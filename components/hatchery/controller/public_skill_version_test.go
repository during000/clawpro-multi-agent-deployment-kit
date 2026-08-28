package controller

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublicSkillVersionCache_FreshAndConcurrentRequests(t *testing.T) {
	t.Run("fresh cache", func(t *testing.T) {
		var calls int32
		cache := newPublicSkillVersionCache(func(context.Context, string) (string, error) {
			atomic.AddInt32(&calls, 1)
			return "01.002.0003", nil
		}, time.Now)

		for range 2 {
			version, err := cache.Latest(context.Background(), "demo", false)
			if err != nil || version != "1.2.3" {
				t.Fatalf("Latest() = %q, %v", version, err)
			}
		}
		if calls != 1 {
			t.Fatalf("fetch calls = %d, want 1", calls)
		}
	})

	t.Run("same slug coalesces", func(t *testing.T) {
		var calls int32
		started := make(chan struct{})
		release := make(chan struct{})
		cache := newPublicSkillVersionCache(func(context.Context, string) (string, error) {
			if atomic.AddInt32(&calls, 1) == 1 {
				close(started)
			}
			<-release
			return "2.0.0", nil
		}, time.Now)

		const workers = 12
		results := make(chan string, workers)
		errs := make(chan error, workers)
		var wg sync.WaitGroup
		wg.Add(workers)
		for range workers {
			go func() {
				defer wg.Done()
				version, err := cache.Latest(context.Background(), "demo", false)
				results <- version
				errs <- err
			}()
		}
		<-started
		close(release)
		wg.Wait()
		close(results)
		close(errs)

		if calls != 1 {
			t.Fatalf("fetch calls = %d, want 1", calls)
		}
		for err := range errs {
			if err != nil {
				t.Fatalf("concurrent Latest error: %v", err)
			}
		}
		for version := range results {
			if version != "2.0.0" {
				t.Fatalf("concurrent version = %q", version)
			}
		}
	})
}

func TestPublicSkillVersionCache_StalePolicyAndFailures(t *testing.T) {
	now := time.Unix(100, 0)
	fetchErr := errors.New("registry unavailable")
	calls := 0
	cache := newPublicSkillVersionCache(func(context.Context, string) (string, error) {
		calls++
		if calls == 1 {
			return "1.0.0", nil
		}
		return "", fetchErr
	}, func() time.Time { return now })

	if version, err := cache.Latest(context.Background(), "demo", false); err != nil || version != "1.0.0" {
		t.Fatalf("prime cache = %q, %v", version, err)
	}
	now = now.Add(publicSkillVersionTTL + time.Second)
	if version, err := cache.Latest(context.Background(), "demo", true); err != nil || version != "1.0.0" {
		t.Fatalf("stale list value = %q, %v", version, err)
	}
	if version, err := cache.Latest(context.Background(), "demo", false); !errors.Is(err, fetchErr) || version != "" {
		t.Fatalf("strict stale value = %q, %v", version, err)
	}

	cold := newPublicSkillVersionCache(func(context.Context, string) (string, error) {
		return "", fetchErr
	}, time.Now)
	if version, err := cold.Latest(context.Background(), "missing", true); !errors.Is(err, fetchErr) || version != "" {
		t.Fatalf("cold failure = %q, %v", version, err)
	}

	var invalidCalls int
	invalid := newPublicSkillVersionCache(func(context.Context, string) (string, error) {
		invalidCalls++
		return "-1.0.0", nil
	}, time.Now)
	for range 2 {
		if _, err := invalid.Latest(context.Background(), "invalid", false); err == nil {
			t.Fatal("invalid version should fail")
		}
	}
	if invalidCalls != 2 {
		t.Fatalf("invalid response was cached, calls=%d", invalidCalls)
	}
}

func TestFetchPublicSkillLatestVersion_ResponseShapesAndLimit(t *testing.T) {
	oldClient := SkillHTTPClient
	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.Path {
		case "/api/v1/skills/primary":
			body = `{"latestVersion":{"version":"2.3.4"}}`
		case "/api/v1/skills/fallback":
			body = `{"latestVersion":{"version":""},"skill":{"tags":{"latest":"3.4.5"}}}`
		case "/api/v1/skills/large":
			body = strings.Repeat("x", maxPublicSkillVersionBodyBytes+1)
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody, Header: make(http.Header), Request: req}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
	defer func() { SkillHTTPClient = oldClient }()

	for _, tt := range []struct {
		slug string
		want string
	}{
		{slug: "primary", want: "2.3.4"},
		{slug: "fallback", want: "3.4.5"},
	} {
		t.Run(tt.slug, func(t *testing.T) {
			version, err := fetchPublicSkillLatestVersion(context.Background(), tt.slug)
			if err != nil || version != tt.want {
				t.Fatalf("fetch = %q, %v; want %q", version, err, tt.want)
			}
		})
	}
	if _, err := fetchPublicSkillLatestVersion(context.Background(), "large"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("large response error = %v", err)
	}
}

type publicVersionErrorReader struct{}

func (publicVersionErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestFetchPublicSkillLatestVersion_Errors(t *testing.T) {
	tests := []struct {
		name      string
		transport skillBundleRoundTripFunc
		want      string
	}{
		{
			name: "transport",
			transport: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("network failed")
			},
			want: "fetch public skill version",
		},
		{
			name: "read",
			transport: func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(publicVersionErrorReader{}), Header: make(http.Header), Request: req}, nil
			},
			want: "read public skill version",
		},
		{
			name: "status",
			transport: func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: http.NoBody, Header: make(http.Header), Request: req}, nil
			},
			want: "HTTP 503",
		},
		{
			name: "malformed json",
			transport: func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{`)), Header: make(http.Header), Request: req}, nil
			},
			want: "decode public skill version",
		},
		{
			name: "missing version",
			transport: func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header), Request: req}, nil
			},
			want: "no latest version",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldClient := SkillHTTPClient
			SkillHTTPClient = &http.Client{Transport: tt.transport}
			defer func() { SkillHTTPClient = oldClient }()
			if _, err := fetchPublicSkillLatestVersion(context.Background(), "demo"); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v, want containing %q", err, tt.want)
			}
		})
	}
}

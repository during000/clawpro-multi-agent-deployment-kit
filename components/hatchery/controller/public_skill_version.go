package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"hatchery/model"
)

const (
	publicSkillVersionTTL          = 5 * time.Minute
	maxPublicSkillVersionBodyBytes = 1 << 20
)

type publicSkillVersionEntry struct {
	version   string
	expiresAt time.Time
}

type publicSkillVersionCall struct {
	done    chan struct{}
	version string
	err     error
}

type publicSkillVersionCache struct {
	mu      sync.Mutex
	entries map[string]publicSkillVersionEntry
	calls   map[string]*publicSkillVersionCall
	fetch   func(context.Context, string) (string, error)
	now     func() time.Time
}

func newPublicSkillVersionCache(
	fetch func(context.Context, string) (string, error),
	now func() time.Time,
) *publicSkillVersionCache {
	return &publicSkillVersionCache{
		entries: make(map[string]publicSkillVersionEntry),
		calls:   make(map[string]*publicSkillVersionCall),
		fetch:   fetch,
		now:     now,
	}
}

// Latest returns the current public registry version for slug. Concurrent
// callers for one slug share one refresh. When allowStale is true, an expired
// successful value is returned if the refresh fails.
func (c *publicSkillVersionCache) Latest(ctx context.Context, slug string, allowStale bool) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", fmt.Errorf("public skill slug is empty")
	}

	now := c.now()
	c.mu.Lock()
	stale, hasStale := c.entries[slug]
	if hasStale && now.Before(stale.expiresAt) {
		c.mu.Unlock()
		return stale.version, nil
	}
	if call, ok := c.calls[slug]; ok {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-call.done:
			if call.err != nil && allowStale && hasStale {
				return stale.version, nil
			}
			return call.version, call.err
		}
	}

	call := &publicSkillVersionCall{done: make(chan struct{})}
	c.calls[slug] = call
	c.mu.Unlock()

	version, err := c.fetch(ctx, slug)
	if err == nil {
		version, err = model.NormalizeSkillVersion(version)
	}

	c.mu.Lock()
	if err == nil {
		c.entries[slug] = publicSkillVersionEntry{
			version:   version,
			expiresAt: c.now().Add(publicSkillVersionTTL),
		}
	}
	call.version = version
	call.err = err
	delete(c.calls, slug)
	close(call.done)
	c.mu.Unlock()

	if err != nil && allowStale && hasStale {
		return stale.version, nil
	}
	return version, err
}

func fetchPublicSkillLatestVersion(ctx context.Context, slug string) (string, error) {
	endpoint := SkillAPIBaseURL + "/api/v1/skills/" + url.PathEscape(slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build public skill version request: %w", err)
	}

	resp, err := SkillHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch public skill version: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPublicSkillVersionBodyBytes+1))
	if err != nil {
		return "", fmt.Errorf("read public skill version response: %w", err)
	}
	if len(body) > maxPublicSkillVersionBodyBytes {
		return "", fmt.Errorf("public skill version response exceeds %d bytes", maxPublicSkillVersionBodyBytes)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("public skill version request returned HTTP %d", resp.StatusCode)
	}

	var payload struct {
		LatestVersion struct {
			Version string `json:"version"`
		} `json:"latestVersion"`
		Skill struct {
			Tags struct {
				Latest string `json:"latest"`
			} `json:"tags"`
		} `json:"skill"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode public skill version response: %w", err)
	}
	version := strings.TrimSpace(payload.LatestVersion.Version)
	if version == "" {
		version = strings.TrimSpace(payload.Skill.Tags.Latest)
	}
	if version == "" {
		return "", fmt.Errorf("public skill version response has no latest version")
	}
	return version, nil
}

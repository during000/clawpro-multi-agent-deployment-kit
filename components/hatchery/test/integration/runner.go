package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type initResult struct {
	token      string
	adminToken string
}

func runInit(ctx context.Context, apiURL, adminToken, testUserPass, testAdminPass string) (*initResult, error) {
	initScript := filepath.Join(filepath.Dir(manifest), "init.py")

	if _, err := os.Stat(initScript); err != nil {
		return nil, fmt.Errorf("init.py not found: %s", initScript)
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "-u", initScript)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("API=%s/", apiURL),
		fmt.Sprintf("ADMIN_TOKEN=%s", adminToken),
		fmt.Sprintf("TEST_PASSWORD=%s", testUserPass),
		fmt.Sprintf("ADMIN_PASSWORD=%s", testAdminPass),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("init.py: stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("init.py: start failed: %w", err)
	}

	var output strings.Builder
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[init.py] %s", line)
		output.WriteString(line)
		output.WriteByte('\n')
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("init.py failed: %w", err)
	}

	result := &initResult{
		token:      extractEnvValue(output.String(), "TOKEN"),
		adminToken: extractEnvValue(output.String(), "ADMIN_TOKEN"),
	}
	if result.token == "" {
		return nil, fmt.Errorf("failed to extract TOKEN from init.py output")
	}
	return result, nil
}

// discoverScripts recursively scans scriptsDir for all test_*.py files.
func discoverScripts(scriptsDir string) ([]string, error) {
	var scripts []string
	err := filepath.WalkDir(scriptsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, "test_") && strings.HasSuffix(name, ".py") {
			scripts = append(scripts, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan scripts dir %s: %w", scriptsDir, err)
	}
	return scripts, nil
}

// filterScripts filters scripts by the given pattern.
// Pattern matching rules:
//   - If pattern contains '/', match against relative path from scriptsDir (e.g. "quota/test_quota_logs.py")
//   - If pattern matches a directory name exactly, include all scripts in that directory (e.g. "admin_user")
//   - Otherwise, glob-match against the file name (e.g. "test_quota_*")
func filterScripts(scripts []string, scriptsDir, pattern string) []string {
	if pattern == "" {
		return scripts
	}

	var filtered []string
	for _, s := range scripts {
		rel, _ := filepath.Rel(scriptsDir, s)
		name := filepath.Base(s)
		dir := filepath.Dir(rel)

		// Pattern contains '/' → match against relative path
		if strings.Contains(pattern, "/") {
			if matched, _ := filepath.Match(pattern, rel); matched {
				filtered = append(filtered, s)
			}
			continue
		}

		// Exact directory name match
		if dir == pattern {
			filtered = append(filtered, s)
			continue
		}

		// Glob match against file name
		if matched, _ := filepath.Match(pattern, name); matched {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// scriptResult records a single script's execution result.
type scriptResult struct {
	script   string
	err      error
	duration time.Duration
	output   string // captured stdout+stderr
}

// test_exclusive_ 前缀表示脚本会修改共享部署数据，必须与其他脚本错开执行。
const exclusiveScriptPrefix = "test_exclusive_"

func isExclusiveScript(script string) bool {
	return strings.HasPrefix(filepath.Base(script), exclusiveScriptPrefix)
}

func shuffleScripts(scripts []string) {
	for i := len(scripts) - 1; i > 0; i-- {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := int(n.Int64())
		scripts[i], scripts[j] = scripts[j], scripts[i]
	}
}

// runScriptBatch runs scripts with bounded concurrency and per-script timeouts.
func runScriptBatch(
	ctx context.Context,
	scripts []string,
	scriptEnv []string,
	maxConcurrency int,
	perScriptTimeout time.Duration,
) []scriptResult {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []scriptResult
	)

	sem := make(chan struct{}, maxConcurrency)
	for _, script := range scripts {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			scriptCtx, cancel := context.WithTimeout(ctx, perScriptTimeout)
			defer cancel()
			start := time.Now()
			output, err := runOneScript(scriptCtx, s, scriptEnv)
			mu.Lock()
			results = append(results, scriptResult{script: s, err: err, duration: time.Since(start), output: output})
			mu.Unlock()
		}(script)
	}

	wg.Wait()
	return results
}

// runScripts runs regular scripts concurrently, then runs scripts that mutate
// shared fixtures exclusively. Both groups are shuffled independently.
func runScripts(ctx context.Context, scripts []string, scriptEnv []string, maxConcurrency int, perScriptTimeout time.Duration) []scriptResult {
	regular := make([]string, 0, len(scripts))
	exclusive := make([]string, 0, 1)
	for _, script := range scripts {
		if isExclusiveScript(script) {
			exclusive = append(exclusive, script)
		} else {
			regular = append(regular, script)
		}
	}
	shuffleScripts(regular)
	shuffleScripts(exclusive)

	results := runScriptBatch(ctx, regular, scriptEnv, maxConcurrency, perScriptTimeout)
	for _, script := range exclusive {
		results = append(results, runScriptBatch(ctx, []string{script}, scriptEnv, 1, perScriptTimeout)...)
	}
	return results
}

// runOneScript runs a single Python test script, prefixing each output line with [script name].
// Returns the captured output and any error.
func runOneScript(ctx context.Context, script string, scriptEnv []string) (string, error) {
	name := filepath.Base(script)
	log.Printf("[%s] started", name)

	cmd := exec.CommandContext(ctx, "python3", "-u", script)
	cmd.Env = append(os.Environ(), scriptEnv...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("%s: stdout pipe: %w", name, err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("%s: start failed: %w", name, err)
	}

	var output strings.Builder
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[%s] %s", name, line)
		output.WriteString(line)
		output.WriteByte('\n')
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return output.String(), fmt.Errorf("%s timed out", name)
		}
		return output.String(), fmt.Errorf("%s failed: %w", name, err)
	}

	log.Printf("[%s] passed", name)
	return output.String(), nil
}

// extractEnvValue extracts the last occurrence of `key=value` or `export key=value` from output.
func extractEnvValue(output, key string) string {
	re := regexp.MustCompile(`(?:export\s+)?` + regexp.QuoteMeta(key) + `=(\S+)`)
	scanner := bufio.NewScanner(strings.NewReader(output))
	var last string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			last = matches[1]
		}
	}
	return last
}

// mergeCoverageData scans tmpDir for per-script coverage JSON files,
// merges them into a single coverage-data.json in outDir. Returns the output path or "".
func mergeCoverageData(tmpDir, outDir string) string {
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		log.Printf("Coverage: failed to read %s: %v", tmpDir, err)
		return ""
	}

	var merged []any
	fileCount := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tmpDir, e.Name()))
		if err != nil {
			continue
		}
		var frames []any
		if err := json.Unmarshal(data, &frames); err != nil {
			continue
		}
		merged = append(merged, frames...)
		fileCount++
	}

	if len(merged) == 0 {
		log.Println("Coverage: no data collected")
		return ""
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Printf("Coverage: failed to create %s: %v", outDir, err)
		return ""
	}

	outPath := filepath.Join(outDir, "coverage-data.json")
	outData, _ := json.Marshal(merged)
	if err := os.WriteFile(outPath, outData, 0o644); err != nil {
		log.Printf("Coverage: failed to write %s: %v", outPath, err)
		return ""
	}

	log.Printf("Coverage: merged %d frames from %d scripts → %s", len(merged), fileCount, outPath)

	// Cleanup tmp
	os.RemoveAll(tmpDir)
	return outPath
}

// generateCoverageReport calls scripts/api_coverage.py to generate an HTML report.
func generateCoverageReport(dataPath, htmlPath string) {
	// Locate the spec and script relative to the project root.
	// manifest is at <project>/test/hatchery-statefulset.yaml, so project root = Dir(Dir(manifest))
	projectRoot := filepath.Dir(filepath.Dir(manifest))
	// Fallback: try common locations
	specPath := ""
	for _, candidate := range []string{
		filepath.Join(projectRoot, "docs", "openapi.json"),
		filepath.Join("docs", "openapi.json"),
		filepath.Join("..", "..", "docs", "openapi.json"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			specPath, _ = filepath.Abs(candidate)
			break
		}
	}
	if specPath == "" {
		log.Println("Coverage: docs/openapi.json not found, skipping HTML report")
		return
	}

	scriptPath := ""
	for _, candidate := range []string{
		filepath.Join(projectRoot, "test", "api_coverage.py"),
		filepath.Join("test", "api_coverage.py"),
		filepath.Join("..", "..", "test", "api_coverage.py"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			scriptPath, _ = filepath.Abs(candidate)
			break
		}
	}
	if scriptPath == "" {
		log.Println("Coverage: test/api_coverage.py not found, skipping HTML report")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check for base spec (for incremental coverage)
	baseSpecPath := ""
	for _, candidate := range []string{
		filepath.Join(projectRoot, "docs", "openapi_base.json"),
		filepath.Join("docs", "openapi_base.json"),
		filepath.Join("..", "..", "docs", "openapi_base.json"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			baseSpecPath, _ = filepath.Abs(candidate)
			break
		}
	}

	args := []string{scriptPath, "--spec", specPath, "--data", dataPath, "--html", htmlPath}
	if baseSpecPath != "" {
		args = append(args, "--base-spec", baseSpecPath)
		log.Printf("Coverage: using base spec for incremental analysis: %s", baseSpecPath)
	}

	cmd := exec.CommandContext(ctx, "python3", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Coverage: failed to generate HTML report: %v\n%s", err, string(output))
		return
	}
	log.Printf("Coverage: HTML report generated → %s", htmlPath)
}

func runCleanupScript(env []string) {
	cleanupScript := filepath.Join(filepath.Dir(manifest), "cleanup.py")
	if _, err := os.Stat(cleanupScript); err != nil {
		log.Printf("Cleanup script not found, skipping: %s", cleanupScript)
		return
	}

	log.Println("Running cleanup script...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "-u", cleanupScript)
	cmd.Env = append(os.Environ(), env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Cleanup script stdout pipe failed: %v", err)
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		log.Printf("Cleanup script start failed: %v", err)
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		log.Printf("[cleanup] %s", scanner.Text())
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("Cleanup script failed: %v", err)
	}
}

func findManifest() string {
	candidates := []string{
		"test/hatchery-statefulset.yaml",
		"hatchery-statefulset.yaml",
		"../hatchery-statefulset.yaml",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	log.Fatal("hatchery-statefulset.yaml not found, use --manifest to specify")
	return ""
}

func generateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}
	return "it-" + hex.EncodeToString(b)
}

func randomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate suffix: %v", err)
	}
	return hex.EncodeToString(b)
}

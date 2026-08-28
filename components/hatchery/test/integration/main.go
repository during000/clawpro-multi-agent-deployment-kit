package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

var (
	kubeconfig    string
	namespace     string
	manifest      string
	image         string
	timeout       time.Duration
	noCleanup     bool
	keepOnFailure bool
	cleanupSuffix string
	concurrency   int
	reportDir     string
	scriptTimeout time.Duration
	secretID      string
	secretKey     string
	runFilter     string
	clsTopicID    string
	clsRegion     string
	clsSecretID   string
	clsSecretKey  string
)

func init() {
	defaultKubeconfig := os.Getenv("KUBECONFIG")
	if defaultKubeconfig == "" {
		if home, err := os.UserHomeDir(); err == nil {
			defaultKubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig, "kubeconfig file path")
	flag.StringVar(&namespace, "namespace", "clawpro-test", "K8s namespace")
	flag.StringVar(&manifest, "manifest", "", "K8s YAML manifest path")
	flag.StringVar(&image, "image", "", "override container image")
	flag.DurationVar(&timeout, "timeout", 1*time.Hour, "global timeout")
	flag.BoolVar(&noCleanup, "no-cleanup", false, "skip cleanup after test")
	flag.BoolVar(&keepOnFailure, "keep-on-failure", false, "keep K8s resources when there are failing test cases")
	flag.StringVar(&cleanupSuffix, "cleanup-suffix", "", "clean up leftover K8s resources (StS/Svc) of a previous run by its suffix; skips the normal test flow")
	flag.IntVar(&concurrency, "concurrency", 3, "max concurrent test scripts")
	flag.DurationVar(&scriptTimeout, "script-timeout", 15*time.Minute, "per-script timeout")
	flag.StringVar(&reportDir, "report-dir", "", "HTML report output directory")
	flag.StringVar(&secretID, "ak", "", "Tencent Cloud SecretId (overrides K8s Secret reference in manifest)")
	flag.StringVar(&secretKey, "sk", "", "Tencent Cloud SecretKey (overrides K8s Secret reference in manifest)")
	flag.StringVar(&runFilter, "run", "", "filter test scripts: directory name, file name, or glob pattern (e.g. admin_user, test_quota_*, quota/test_quota_logs.py)")
	flag.StringVar(&clsTopicID, "cls-topic-id", "", "Tencent Cloud CLS topic id for reporting test results (if set, results are reported)")
	flag.StringVar(&clsRegion, "cls-region", "ap-guangzhou", "Tencent Cloud CLS region")
	flag.StringVar(&clsSecretID, "cls-ak", "", "Tencent Cloud SecretId for CLS reporting only (decoupled from --ak/--sk)")
	flag.StringVar(&clsSecretKey, "cls-sk", "", "Tencent Cloud SecretKey for CLS reporting only (decoupled from --ak/--sk)")
}

func main() {
	flag.Parse()

	exitCode := 1
	defer func() { os.Exit(exitCode) }()

	if manifest == "" {
		manifest = findManifest()
	}

	// 清理模式：直接复用部署时既有的 kubeconfig，按命名规则定位残留资源并删除，
	// 不走正常测试流程（不依赖流水线，本地 `go run . --cleanup-suffix <suffix>` 即可闭环）。
	if cleanupSuffix != "" {
		os.Exit(runCleanup())
	}

	suffix := randomSuffix()
	log.Printf("Test suffix: %s", suffix)

	adminToken := generateToken()
	log.Print("Admin token generated")

	testUserPass := generateToken()
	testAdminPass := generateToken()

	startTime := time.Now()
	var (
		results     []scriptResult
		failed      []scriptResult // failing test cases, used by cleanup deferral to decide resource retention
		failMsg     string         // 非测试阶段的失败原因
		setupOutput string         // Pod 诊断输出，用于报告展示
	)

	// CLS 上报（best-effort）：创建上报器并注册关闭。
	// CLS 参数（--cls-topic-id/--cls-ak/--cls-sk/--cls-region）仅从启动参数读取，不从环境变量获取；
	// 任一参数缺失则不上报，sender 为 nil，后续上报调用安全跳过。
	var sender LogSender
	if clsTopicID == "" || clsSecretID == "" || clsSecretKey == "" {
		log.Printf("CLS report disabled: missing --cls-topic-id/--cls-ak/--cls-sk, test results will not be reported to CLS")
	} else if s, sErr := newCLSSender(clsTopicID, clsRegion, clsSecretID, clsSecretKey, suffix); sErr != nil {
		log.Printf("CLS report disabled: %v", sErr)
	} else {
		sender = s
	}
	// closeSender 必须在 report-defer 之前注册，以保证兜底上报的记录在 Close 之前发出。
	defer closeSender(sender)

	// 报告兜底：无论哪个阶段失败都尝试生成 HTML 与上报 CLS，两者相互独立：
	//   - HTML 报告：仅当 --report-dir 指定时生成，与 CLS 无关；
	//   - CLS 上报：仅当 sender 非 nil（--cls-* 参数齐全）时发送，与 reportDir 无关。
	defer func() {
		// setup 阶段失败且无测试结果时，用失败原因兜底构造一条 setup 结果复用上报逻辑。
		if len(results) == 0 && failMsg != "" {
			results = []scriptResult{{script: "setup", err: errors.New(failMsg), duration: time.Since(startTime), output: setupOutput}}
		}
		if len(results) == 0 {
			return
		}
		if reportDir != "" {
			writeReport(results, time.Since(startTime))
		}
		reportCLS(sender, results, time.Since(startTime))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Signal received, cleaning up...")
		cancel()
	}()

	clientset, err := connectK8s()
	if err != nil {
		failMsg = fmt.Sprintf("Failed to connect K8s: %v", err)
		log.Print(failMsg)
		return
	}
	log.Println("Connected to K8s cluster")

	resources, err := loadManifest(manifest)
	if err != nil {
		failMsg = fmt.Sprintf("Failed to load manifest: %v", err)
		log.Print(failMsg)
		return
	}
	if len(resources.statefulSets) == 0 {
		failMsg = "No StatefulSet found in YAML"
		log.Print(failMsg)
		return
	}
	log.Printf("Loaded manifest: %s (%d StatefulSet, %d Service)",
		manifest, len(resources.statefulSets), len(resources.services))

	for _, sts := range resources.statefulSets {
		sts.Name = sts.Name + "-" + suffix
		sts.Spec.ServiceName = sts.Spec.ServiceName + "-" + suffix
		randomizeLabels(sts, suffix)
		injectAdminToken(sts, adminToken)
		injectIdentifier(sts, suffix)
		injectPasswordlessLoginAllowlist(sts, suffix)
		if image != "" {
			overrideImage(sts, image)
		}
		if secretID != "" && secretKey != "" {
			injectCredentials(sts, secretID, secretKey)
		}
	}
	for _, svc := range resources.services {
		svc.Name = svc.Name + "-" + suffix
		randomizeServiceSelector(svc, suffix)
	}
	if image != "" {
		log.Printf("Image overridden: %s", image)
	}

	var runs []cleanupRun
	var initRan bool
	var apiURL string
	defer func() {
		if noCleanup {
			log.Println("Skipping cleanup (--no-cleanup)")
			return
		}
		if keepOnFailure && len(failed) > 0 {
			log.Printf("===== Keeping K8s resources for debugging (%d failing test case(s)) =====", len(failed))
			log.Printf("  Namespace: %s   Identifier suffix: %s", namespace, suffix)
			log.Printf("  StatefulSet/Service names are suffixed with: %s", suffix)
			return
		}
		// 正常流程软清理使用已知凭据（apiURL/adminToken/suffix）；硬删除统一交给
		// runCleanupRuns，与清理模式复用同一 cleanupOneRun 流程。
		if initRan {
			softCleanup(apiURL, adminToken, suffix)
		}
		log.Println("===== Cleaning up K8s resources =====")
		runCleanupRuns(context.Background(), clientset, runs, false)
	}()

	// Deploy
	if err := ensureNamespace(ctx, clientset, namespace); err != nil {
		failMsg = fmt.Sprintf("Failed to create namespace: %v", err)
		log.Print(failMsg)
		return
	}

	for _, svc := range resources.services {
		if err := applyService(ctx, clientset, svc); err != nil {
			failMsg = fmt.Sprintf("Failed to deploy Service %s: %v", svc.Name, err)
			log.Print(failMsg)
			return
		}
		svcName := svc.Name
		runs = append(runs, cleanupRun{svcName: svcName})
		log.Printf("Service %s deployed", svc.Name)
	}

	for _, sts := range resources.statefulSets {
		if err := applyStatefulSet(ctx, clientset, sts); err != nil {
			failMsg = fmt.Sprintf("Failed to deploy StatefulSet %s: %v", sts.Name, err)
			log.Print(failMsg)
			return
		}
		stsName := sts.Name
		runs = append(runs, cleanupRun{stsName: stsName})
		log.Printf("StatefulSet %s deployed", sts.Name)
	}

	stsName := resources.statefulSets[0].Name
	if err := waitForStatefulSetReady(ctx, clientset, stsName, 1); err != nil {
		setupOutput = dumpPodDiagnostics(clientset, stsName)
		failMsg = fmt.Sprintf("Timeout waiting for Pod ready: %v", err)
		log.Print(failMsg)
		return
	}
	log.Println("Pod ready")

	svcName := resources.services[0].Name
	lbIP, err := waitForLoadBalancerIP(ctx, clientset, svcName)
	if err != nil {
		failMsg = fmt.Sprintf("Failed to get LoadBalancer IP: %v", err)
		log.Print(failMsg)
		return
	}
	apiURL = fmt.Sprintf("http://%s", lbIP)
	log.Printf("LoadBalancer IP: %s", lbIP)
	trustedDomain := fmt.Sprintf("https://%s", lbIP)

	for _, sts := range resources.statefulSets {
		injectArg(sts, "-domain", trustedDomain)
		if err := applyStatefulSet(ctx, clientset, sts); err != nil {
			failMsg = fmt.Sprintf("Failed to update StatefulSet domain: %v", err)
			log.Print(failMsg)
			return
		}
	}
	log.Printf("Injected domain=%s, waiting for Pod restart...", trustedDomain)

	if err := waitForStatefulSetReady(ctx, clientset, stsName, 2); err != nil {
		setupOutput = dumpPodDiagnostics(clientset, stsName)
		failMsg = fmt.Sprintf("Timeout waiting for Pod restart: %v", err)
		log.Print(failMsg)
		return
	}

	if err := waitForHealth(ctx, apiURL); err != nil {
		failMsg = fmt.Sprintf("Health check failed: %v", err)
		log.Print(failMsg)
		return
	}
	log.Println("Health check passed")

	// Run init.py
	initRan = true
	creds, err := runInit(ctx, apiURL, adminToken, testUserPass, testAdminPass)
	if err != nil {
		failMsg = fmt.Sprintf("init.py failed: %v", err)
		log.Print(failMsg)
		return
	}
	log.Println("Init complete, token acquired")

	// Discover and run test scripts
	scriptsDir := filepath.Join(filepath.Dir(manifest), "scripts")
	scripts, err := discoverScripts(scriptsDir)
	if err != nil {
		failMsg = fmt.Sprintf("Failed to scan scripts dir: %v", err)
		log.Print(failMsg)
		return
	}

	if runFilter != "" {
		scripts = filterScripts(scripts, scriptsDir, runFilter)
		log.Printf("Filter: %q matched %d script(s)", runFilter, len(scripts))
	}

	if len(scripts) == 0 {
		failMsg = fmt.Sprintf("No *.py test scripts found in %s (filter: %q)", scriptsDir, runFilter)
		log.Print(failMsg)
		return
	}

	log.Printf("Running %d test scripts (concurrency: %d)", len(scripts), concurrency)
	for _, s := range scripts {
		log.Printf("  - %s", filepath.Base(s))
	}

	// Load extra env from ConfigMap (optional, loaded first so flow vars take precedence)
	scriptEnv := loadEnvFromConfigMap(ctx, clientset)
	scriptEnv = append(scriptEnv,
		fmt.Sprintf("API=%s/", apiURL),
		fmt.Sprintf("TOKEN=%s", creds.token),
		fmt.Sprintf("ADMIN_TOKEN=%s", creds.adminToken),
		// The cloud proxy is intentionally not wrapped by WithOpenAPI, so scripts
		// that verify the resulting Tencent Cloud resource need the bootstrap token.
		fmt.Sprintf("BOOTSTRAP_ADMIN_TOKEN=%s", adminToken),
		fmt.Sprintf("IDENTIFIER=%s", suffix),
	)

	// API coverage collection: inject COVERAGE_DIR for Python scripts
	var coverageTmpDir string
	if reportDir != "" {
		coverageTmpDir = filepath.Join(os.TempDir(), "hatchery-coverage-"+suffix)
		os.MkdirAll(coverageTmpDir, 0o755)
		scriptEnv = append(scriptEnv, fmt.Sprintf("COVERAGE_DIR=%s", coverageTmpDir))
	}

	results = runScripts(ctx, scripts, scriptEnv, concurrency, scriptTimeout)

	// Check for unexpected pod restarts
	restarts := checkPodRestarts(clientset, stsName)
	if len(restarts) > 0 {
		var buf strings.Builder
		for _, r := range restarts {
			fmt.Fprintf(&buf, "Pod %s / Container %s restarted %d time(s)\n", r.podName, r.containerName, r.restartCount)
			fmt.Fprintf(&buf, "  Last terminated: reason=%s exitCode=%d message=%s\n", r.lastReason, r.lastExitCode, r.lastMessage)
			if r.previousLogs != "" {
				buf.WriteString("  Previous logs (tail 100 lines):\n")
				for _, line := range strings.Split(r.previousLogs, "\n") {
					fmt.Fprintf(&buf, "    %s\n", line)
				}
			} else {
				buf.WriteString("  Previous logs: <unavailable, container logs may have been garbage collected>\n")
			}
		}
		log.Println("===== Pod Restart Check FAILED =====")
		log.Print(buf.String())
		results = append(results, scriptResult{
			script: "pod-restart-check",
			err:    fmt.Errorf("%d container(s) restarted unexpectedly", len(restarts)),
			output: buf.String(),
		})
	} else {
		log.Println("Pod restart check: no restarts detected")
	}

	// Sort results by relative script path for stable output
	sort.Slice(results, func(i, j int) bool {
		ri, _ := filepath.Rel(scriptsDir, results[i].script)
		rj, _ := filepath.Rel(scriptsDir, results[j].script)
		return ri < rj
	})

	for _, r := range results {
		if r.err != nil {
			failed = append(failed, r)
		}
	}

	log.Println("===== Test Results =====")
	for _, r := range results {
		name := filepath.Base(r.script)
		if r.err != nil {
			log.Printf("  FAIL  %s: %v", name, r.err)
		} else {
			log.Printf("  PASS  %s", name)
		}
	}

	if len(failed) > 0 {
		log.Printf("%d/%d test scripts failed", len(failed), len(results))
	} else {
		log.Printf("All %d test scripts passed", len(results))
	}

	// Merge coverage data and generate HTML report
	if coverageTmpDir != "" {
		dataPath := mergeCoverageData(coverageTmpDir, reportDir)
		if dataPath != "" {
			generateCoverageReport(dataPath, filepath.Join(reportDir, "coverage.html"))
		}
	}

	if len(failed) > 0 {
		return
	}
	exitCode = 0
}

func writeReport(results []scriptResult, totalDuration time.Duration) {
	reportPath := filepath.Join(reportDir, "index.html")
	if err := generateReport(reportPath, results, totalDuration); err != nil {
		log.Printf("Failed to generate report: %v", err)
	} else {
		log.Printf("Report generated: %s", reportPath)
	}
}

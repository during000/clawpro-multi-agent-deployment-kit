package main

import (
	"context"
	"fmt"
	"log"

	"k8s.io/client-go/kubernetes"
)

// cleanupRun 描述一次需清理的运行（已是带 suffix 的完整资源名）。
type cleanupRun struct {
	stsName string
	svcName string
}

// runCleanup 进入清理模式：连接集群、读取 manifest 取基础资源名，按命名规则
// （<base>-<suffix>）定位一次遗留运行的 StS/Svc 并清理。复用与正常流程相同的
// 全局变量（namespace/manifest），无需额外参数传递。
func runCleanup() int {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	clientset, err := connectK8s()
	if err != nil {
		log.Printf("Cleanup failed: %v", err)
		return 1
	}

	if manifest == "" {
		manifest = findManifest()
	}
	resources, err := loadManifest(manifest)
	if err != nil {
		log.Printf("Cleanup failed to load manifest: %v", err)
		return 1
	}
	if len(resources.statefulSets) == 0 {
		log.Print("Cleanup failed: no StatefulSet found in manifest")
		return 1
	}

	runs := make([]cleanupRun, 0, len(resources.statefulSets))
	for i, sts := range resources.statefulSets {
		r := cleanupRun{stsName: sts.Name + "-" + cleanupSuffix}
		if i < len(resources.services) {
			r.svcName = resources.services[i].Name + "-" + cleanupSuffix
		}
		runs = append(runs, r)
	}
	// 清理模式下凭据只能从集群内残留 StS 恢复，故 softClean=true。
	return runCleanupRuns(ctx, clientset, runs, true)
}

// runCleanupRuns 依次清理多个运行，复用 cleanupOneRun。
// softClean 控制是否执行软清理（仅当凭据已就绪时为 true，如正常流程的 init 已执行、
// 或清理模式下凭据可从 StS args 恢复）。逆序执行以「先删 StatefulSet 再删 Service」。
func runCleanupRuns(ctx context.Context, clientset *kubernetes.Clientset, runs []cleanupRun, softClean bool) int {
	rc := 0
	for i := len(runs) - 1; i >= 0; i-- {
		if code := cleanupOneRun(ctx, clientset, runs[i].stsName, runs[i].svcName, softClean); code != 0 {
			rc = code
		}
	}
	return rc
}

// cleanupOneRun 清理单次运行：按需从 StS 恢复凭据（仅 softClean 时）、软清理 hatchery
// 内部数据（best-effort），再硬删 StS/Svc。StS 读取失败时仍按名硬删，确保资源释放。
func cleanupOneRun(ctx context.Context, clientset *kubernetes.Clientset, stsName, svcName string, softClean bool) int {
	if stsName == "" && svcName == "" {
		return 0
	}
	log.Printf("===== Cleaning up run: StatefulSet=%s Service=%s =====", stsName, svcName)

	if softClean && stsName != "" {
		sts, err := getStatefulSet(ctx, clientset, stsName)
		if err != nil {
			log.Printf("  Failed to get StatefulSet %s (continuing with hard delete): %v", stsName, err)
			deleteResources(ctx, clientset, stsName, svcName)
			return 1
		}
		adminToken := extractStsArg(sts, "--admin-token")
		identifier := extractStsArg(sts, "--identifier")
		domain := extractStsArg(sts, "-domain")

		// 优先从 Service LoadBalancer 状态取地址拼 API URL。
		apiBase := ""
		if svcName != "" {
			if endpoint := lookupLoadBalancerEndpoint(ctx, clientset, svcName); endpoint != "" {
				apiBase = "http://" + endpoint
			}
		}
		// 兜底：解析 StS args 里的 -domain（如 http://42.193.240.13）。
		if apiBase == "" && domain != "" {
			apiBase = domain
		}
		softCleanup(apiBase, adminToken, identifier)
	}

	// 硬删除：StS + Svc。即使软清理失败也继续，确保集群资源释放。
	deleteResources(ctx, clientset, stsName, svcName)
	return 0
}

// softCleanup 调用 cleanup.py 清 hatchery 内部数据（best-effort）。
// apiBase/adminToken/identifier 任一为空则跳过（仅告警）。
func softCleanup(apiBase, adminToken, identifier string) {
	if apiBase == "" || adminToken == "" {
		log.Printf("  Skipping soft cleanup (apiBase=%q or admin token unavailable)", apiBase)
		return
	}
	log.Printf("  Running soft cleanup via %s (identifier=%s)", apiBase, identifier)
	runCleanupScript([]string{
		fmt.Sprintf("API=%s/", apiBase),
		fmt.Sprintf("ADMIN_TOKEN=%s", adminToken),
		fmt.Sprintf("IDENTIFIER=%s", identifier),
	})
}

// deleteResources 硬删除 StatefulSet 与 Service（best-effort：忽略错误仅日志）。
// stsName 或 svcName 为空时跳过对应资源。
func deleteResources(ctx context.Context, clientset *kubernetes.Clientset, stsName, svcName string) {
	if stsName != "" {
		log.Printf("  Deleting StatefulSet: %s", stsName)
		if err := deleteStatefulSet(ctx, clientset, stsName); err != nil {
			log.Printf("  Failed to delete StatefulSet %s: %v", stsName, err)
		}
	}
	if svcName != "" {
		log.Printf("  Deleting Service: %s", svcName)
		if err := deleteService(ctx, clientset, svcName); err != nil {
			log.Printf("  Failed to delete Service %s: %v", svcName, err)
		}
	}
}

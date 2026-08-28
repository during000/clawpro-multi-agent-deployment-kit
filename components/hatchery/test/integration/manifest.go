package main

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// randomizeLabels 给 StatefulSet 的 selector 和 pod template labels 加后缀，
// 使得多个并发测试实例的 Pod 互不干扰。
func randomizeLabels(sts *appsv1.StatefulSet, suffix string) {
	if sts.Spec.Selector != nil && sts.Spec.Selector.MatchLabels != nil {
		if v, ok := sts.Spec.Selector.MatchLabels["app"]; ok {
			sts.Spec.Selector.MatchLabels["app"] = v + "-" + suffix
		}
	}
	if sts.Spec.Template.Labels != nil {
		if v, ok := sts.Spec.Template.Labels["app"]; ok {
			sts.Spec.Template.Labels["app"] = v + "-" + suffix
		}
	}
	if sts.Labels != nil {
		if v, ok := sts.Labels["app"]; ok {
			sts.Labels["app"] = v + "-" + suffix
		}
	}
}

// randomizeServiceSelector 给 Service 的 selector 加后缀以匹配随机化后的 Pod。
func randomizeServiceSelector(svc *corev1.Service, suffix string) {
	if svc.Spec.Selector != nil {
		if v, ok := svc.Spec.Selector["app"]; ok {
			svc.Spec.Selector["app"] = v + "-" + suffix
		}
	}
	if svc.Labels != nil {
		if v, ok := svc.Labels["app"]; ok {
			svc.Labels["app"] = v + "-" + suffix
		}
	}
}

// injectAdminToken 将 admin-token 注入到 StatefulSet 的 hatchery 容器启动参数中。
func injectAdminToken(sts *appsv1.StatefulSet, token string) {
	injectArg(sts, "--admin-token", token)
}

// injectIdentifier 将 identifier 注入到 StatefulSet 的 hatchery 容器启动参数中。
func injectIdentifier(sts *appsv1.StatefulSet, identifier string) {
	injectArg(sts, "--identifier", identifier)
}

// injectPasswordlessLoginAllowlist 在容器启动后写入临时测试库，无需 pods/exec 权限。
func injectPasswordlessLoginAllowlist(sts *appsv1.StatefulSet, identifier string) {
	command := fmt.Sprintf(
		`i=0; while [ "$i" -lt 60 ]; do if sqlite3 -cmd '.timeout 5000' /data/hatchery.db "INSERT OR IGNORE INTO feature_allowlists (type, identifier) VALUES ('passwordless-login', '%s');"; then exit 0; fi; i=$((i+1)); sleep 1; done; exit 1`,
		identifier,
	)
	for i := range sts.Spec.Template.Spec.Containers {
		c := &sts.Spec.Template.Spec.Containers[i]
		if c.Name != "hatchery" {
			continue
		}
		if c.Lifecycle == nil {
			c.Lifecycle = &corev1.Lifecycle{}
		}
		c.Lifecycle.PostStart = &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{Command: []string{"/bin/sh", "-c", command}},
		}
	}
}

// injectArg 注入或替换 hatchery 容器的指定命令行参数。
func injectArg(sts *appsv1.StatefulSet, argName, argValue string) {
	for i := range sts.Spec.Template.Spec.Containers {
		c := &sts.Spec.Template.Spec.Containers[i]
		if c.Name != "hatchery" {
			continue
		}
		found := false
		for j := 0; j < len(c.Args)-1; j++ {
			if c.Args[j] == argName {
				c.Args[j+1] = argValue
				found = true
				break
			}
		}
		if !found {
			c.Args = append(c.Args, argName, argValue)
		}
	}
}

// extractStsArg 从 hatchery 容器的启动参数中提取紧邻在 argName 后的取值。
// 与 injectArg 共享同一容器遍历约定，仅把"写"改为"读"，用于事后清理时
// 从残留 StatefulSet 的 spec 中恢复 --admin-token/--identifier/-domain 等凭据。
func extractStsArg(sts *appsv1.StatefulSet, argName string) string {
	for i := range sts.Spec.Template.Spec.Containers {
		c := &sts.Spec.Template.Spec.Containers[i]
		if c.Name != "hatchery" {
			continue
		}
		for j := 0; j < len(c.Args)-1; j++ {
			if c.Args[j] == argName {
				return c.Args[j+1]
			}
		}
	}
	return ""
}

// overrideImage 覆盖 StatefulSet 中 hatchery 容器的镜像地址。
func overrideImage(sts *appsv1.StatefulSet, img string) {
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "hatchery" {
			sts.Spec.Template.Spec.Containers[i].Image = img
			break
		}
	}
}

// injectCredentials 将 SecretId/SecretKey 直接注入为环境变量值，
// 替换掉原有的 secretKeyRef 引用，使得无需在集群中预置 K8s Secret。
func injectCredentials(sts *appsv1.StatefulSet, ak, sk string) {
	credEnvMap := map[string]string{
		"SECRET_ID":                 ak,
		"SECRET_KEY":                sk,
		"AGENT_CAM_ROLE_SECRET_ID":  ak,
		"AGENT_CAM_ROLE_SECRET_KEY": sk,
	}

	for i := range sts.Spec.Template.Spec.Containers {
		c := &sts.Spec.Template.Spec.Containers[i]
		if c.Name != "hatchery" {
			continue
		}
		for j := range c.Env {
			if val, ok := credEnvMap[c.Env[j].Name]; ok {
				c.Env[j].Value = val
				c.Env[j].ValueFrom = nil
			}
		}
	}
}

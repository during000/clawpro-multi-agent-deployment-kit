package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
)

type manifestResources struct {
	statefulSets []*appsv1.StatefulSet
	services     []*corev1.Service
}

func connectK8s() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	return clientset, nil
}

func loadManifest(path string) (*manifestResources, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	resources := &manifestResources{}
	decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	codecFactory := serializer.NewCodecFactory(scheme.Scheme)
	deserializer := codecFactory.UniversalDeserializer()

	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode YAML: %w", err)
		}
		if rawObj.Raw == nil {
			continue
		}

		obj, gvk, err := deserializer.Decode(rawObj.Raw, nil, nil)
		if err != nil {
			unObj := &unstructured.Unstructured{}
			if err2 := unObj.UnmarshalJSON(rawObj.Raw); err2 != nil {
				return nil, fmt.Errorf("deserialize: %w", err)
			}
			switch unObj.GetKind() {
			case "StatefulSet":
				sts := &appsv1.StatefulSet{}
				if err3 := runtime.DefaultUnstructuredConverter.FromUnstructured(unObj.Object, sts); err3 != nil {
					return nil, fmt.Errorf("convert StatefulSet: %w", err3)
				}
				resources.statefulSets = append(resources.statefulSets, sts)
			case "Service":
				svc := &corev1.Service{}
				if err3 := runtime.DefaultUnstructuredConverter.FromUnstructured(unObj.Object, svc); err3 != nil {
					return nil, fmt.Errorf("convert Service: %w", err3)
				}
				resources.services = append(resources.services, svc)
			default:
				log.Printf("Skipping unknown resource type: %s", unObj.GetKind())
			}
			continue
		}

		switch gvk.Kind {
		case "StatefulSet":
			if sts, ok := obj.(*appsv1.StatefulSet); ok {
				resources.statefulSets = append(resources.statefulSets, sts)
			}
		case "Service":
			if svc, ok := obj.(*corev1.Service); ok {
				resources.services = append(resources.services, svc)
			}
		default:
			log.Printf("Skipping unknown resource type: %s", gvk.Kind)
		}
	}
	return resources, nil
}

func ensureNamespace(ctx context.Context, clientset *kubernetes.Clientset, ns string) error {
	_, err := clientset.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}
	nsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}
	_, err = clientset.CoreV1().Namespaces().Create(ctx, nsObj, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func applyStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, sts *appsv1.StatefulSet) error {
	sts.Namespace = namespace

	existing, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, sts.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		_, err = clientset.AppsV1().StatefulSets(namespace).Create(ctx, sts, metav1.CreateOptions{})
		return err
	}

	sts.ResourceVersion = existing.ResourceVersion
	_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	return err
}

func applyService(ctx context.Context, clientset *kubernetes.Clientset, svc *corev1.Service) error {
	svc.Namespace = namespace

	existing, err := clientset.CoreV1().Services(namespace).Get(ctx, svc.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		_, err = clientset.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
		return err
	}

	svc.ResourceVersion = existing.ResourceVersion
	svc.Spec.ClusterIP = existing.Spec.ClusterIP
	_, err = clientset.CoreV1().Services(namespace).Update(ctx, svc, metav1.UpdateOptions{})
	return err
}

// getStatefulSet 按名字读取 StatefulSet。
func getStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, name string) (*appsv1.StatefulSet, error) {
	return clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// deleteStatefulSet 删除 StatefulSet。
func deleteStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, name string) error {
	return clientset.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// deleteService 删除 Service。
func deleteService(ctx context.Context, clientset *kubernetes.Clientset, name string) error {
	return clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// lookupLoadBalancerEndpoint 读取 Service 当前已分配的 LoadBalancer 外部地址
// （IP 优先，其次 Hostname）。与 waitForLoadBalancerIP 不同，本函数只做一次性
// 查询、不等待，取不到时返回空串，供事后清理等 best-effort 场景使用。
func lookupLoadBalancerEndpoint(ctx context.Context, clientset *kubernetes.Clientset, svcName string) string {
	svc, err := clientset.CoreV1().Services(namespace).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return ""
	}
	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			return ingress.IP
		}
		if ingress.Hostname != "" {
			return ingress.Hostname
		}
	}
	return ""
}

func waitForStatefulSetReady(ctx context.Context, clientset *kubernetes.Clientset, name string, minGeneration int64) error {
	log.Printf("Waiting for StatefulSet %s/%s ready (generation>=%d)...", namespace, name, minGeneration)

	watcher, err := clientset.AppsV1().StatefulSets(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return fmt.Errorf("watch StatefulSet: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for StatefulSet ready")
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}
			if event.Type == watch.Modified || event.Type == watch.Added {
				if sts, ok := event.Object.(*appsv1.StatefulSet); ok {
					desired := int32(1)
					if sts.Spec.Replicas != nil {
						desired = *sts.Spec.Replicas
					}
					log.Printf("  ready=%d updated=%d desired=%d generation=%d/%d",
						sts.Status.ReadyReplicas, sts.Status.UpdatedReplicas, desired,
						sts.Status.ObservedGeneration, sts.Generation)
					if sts.Status.ObservedGeneration >= minGeneration &&
						sts.Status.UpdatedReplicas >= desired &&
						sts.Status.ReadyReplicas >= desired {
						return nil
					}
				}
			}
		}
	}
}

func waitForLoadBalancerIP(ctx context.Context, clientset *kubernetes.Clientset, svcName string) (string, error) {
	log.Printf("Waiting for Service %s/%s external IP...", namespace, svcName)

	watcher, err := clientset.CoreV1().Services(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", svcName),
	})
	if err != nil {
		return "", fmt.Errorf("watch Service: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for LoadBalancer IP")
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return "", fmt.Errorf("watch channel closed")
			}
			if event.Type == watch.Modified || event.Type == watch.Added {
				if svc, ok := event.Object.(*corev1.Service); ok {
					for _, ingress := range svc.Status.LoadBalancer.Ingress {
						if ingress.IP != "" {
							return ingress.IP, nil
						}
						if ingress.Hostname != "" {
							return ingress.Hostname, nil
						}
					}
				}
			}
		}
	}
}

func waitForHealth(ctx context.Context, apiURL string) error {
	log.Println("Waiting for health check...")
	client := &http.Client{Timeout: 5 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check timeout")
		default:
		}

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"/health", nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
}

// loadEnvFromConfigMap reads "integration-test-env" ConfigMap and returns key=value pairs.
// Returns nil if the ConfigMap doesn't exist.
func loadEnvFromConfigMap(ctx context.Context, clientset *kubernetes.Clientset) []string {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, "integration-test-env", metav1.GetOptions{})
	if err != nil {
		log.Printf("ConfigMap integration-test-env not found, skipping extra env")
		return nil
	}

	var envs []string
	var keys []string
	for k, v := range cm.Data {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
		keys = append(keys, k)
	}

	if len(keys) > 0 {
		log.Printf("Loaded %d env vars from ConfigMap integration-test-env:", len(keys))
		for _, k := range keys {
			log.Printf("  - %s", k)
		}
	}

	return envs
}

// listStatefulSetPods lists all Pods belonging to a StatefulSet via its label selector.
func listStatefulSetPods(ctx context.Context, clientset *kubernetes.Clientset, stsName string) ([]corev1.Pod, error) {
	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, stsName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get StatefulSet: %w", err)
	}

	var selectorParts []string
	if sts.Spec.Selector != nil {
		for k, v := range sts.Spec.Selector.MatchLabels {
			selectorParts = append(selectorParts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	labelSelector := strings.Join(selectorParts, ",")
	if labelSelector == "" {
		return nil, fmt.Errorf("no label selector on StatefulSet %s", stsName)
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("list Pods: %w", err)
	}
	return pods.Items, nil
}

// fetchLogs fetches container logs (tail n lines). If previous is true, returns the
// previous terminated container's logs. Returns empty string if not available.
func fetchLogs(ctx context.Context, clientset *kubernetes.Clientset, podName, containerName string, tailLines int64, previous bool) string {
	logReq := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
		Previous:  previous,
	})
	stream, err := logReq.Stream(ctx)
	if err != nil {
		log.Printf("fetchLogs: failed to get previous logs for %s/%s: %v", podName, containerName, err)
		return ""
	}
	defer stream.Close()
	logBytes, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("fetchLogs: failed to read logs for %s/%s: %v", podName, containerName, err)
		return ""
	}
	return strings.TrimSpace(string(logBytes))
}

// containerRestart records restart info for a container that has restarted.
type containerRestart struct {
	podName       string
	containerName string
	restartCount  int32
	lastReason    string
	lastExitCode  int32
	lastMessage   string
	previousLogs  string
}

// checkPodRestarts checks if any pod under the StatefulSet has restarted (RestartCount > 0).
// Returns restart details with previous container logs, or nil if no restarts found.
func checkPodRestarts(clientset *kubernetes.Clientset, stsName string) []containerRestart {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pods, err := listStatefulSetPods(ctx, clientset, stsName)
	if err != nil {
		log.Printf("checkPodRestarts: %v", err)
		return nil
	}

	var restarts []containerRestart
	for _, pod := range pods {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.RestartCount == 0 {
				continue
			}
			cr := containerRestart{
				podName:       pod.Name,
				containerName: cs.Name,
				restartCount:  cs.RestartCount,
				previousLogs:  fetchLogs(ctx, clientset, pod.Name, cs.Name, 100, true),
			}
			if cs.LastTerminationState.Terminated != nil {
				t := cs.LastTerminationState.Terminated
				cr.lastReason = t.Reason
				cr.lastExitCode = t.ExitCode
				cr.lastMessage = t.Message
			}
			restarts = append(restarts, cr)
		}
	}
	return restarts
}

// dumpPodDiagnostics 获取 StatefulSet 下所有 Pod 的状态和容器日志，
// 同时打印到 log 并返回完整诊断文本（用于报告）。
func dumpPodDiagnostics(clientset *kubernetes.Clientset, stsName string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var buf strings.Builder
	logAndWrite := func(format string, args ...interface{}) {
		line := fmt.Sprintf(format, args...)
		log.Print(line)
		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	logAndWrite("===== Pod Diagnostics for StatefulSet %s/%s =====", namespace, stsName)

	pods, err := listStatefulSetPods(ctx, clientset, stsName)
	if err != nil {
		logAndWrite("  %v", err)
		return buf.String()
	}

	if len(pods) == 0 {
		logAndWrite("  No Pods found for StatefulSet %s", stsName)
		return buf.String()
	}

	for _, pod := range pods {
		logAndWrite("--- Pod: %s (Phase: %s) ---", pod.Name, pod.Status.Phase)

		for _, cond := range pod.Status.Conditions {
			logAndWrite("  Condition: %s=%s (Reason: %s, Message: %s)",
				cond.Type, cond.Status, cond.Reason, cond.Message)
		}

		allStatuses := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
		for _, cs := range allStatuses {
			if cs.State.Waiting != nil {
				logAndWrite("  Container %s: Waiting (Reason: %s, Message: %s)",
					cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message)
			} else if cs.State.Terminated != nil {
				logAndWrite("  Container %s: Terminated (Reason: %s, ExitCode: %d, Message: %s)",
					cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode, cs.State.Terminated.Message)
			} else if cs.State.Running != nil {
				logAndWrite("  Container %s: Running (Started: %v)", cs.Name, cs.State.Running.StartedAt.Time)
			}
			if cs.RestartCount > 0 {
				logAndWrite("  Container %s: RestartCount=%d", cs.Name, cs.RestartCount)
			}
			if cs.LastTerminationState.Terminated != nil {
				t := cs.LastTerminationState.Terminated
				logAndWrite("  Container %s: LastTerminated (Reason: %s, ExitCode: %d, Message: %s)",
					cs.Name, t.Reason, t.ExitCode, t.Message)
			}
		}

		for _, container := range pod.Spec.Containers {
			logAndWrite("  [Logs: %s/%s] (tail 50 lines)", pod.Name, container.Name)
			logText := fetchLogs(ctx, clientset, pod.Name, container.Name, 50, false)
			if logText == "" {
				logAndWrite("    (no logs)")
			} else {
				for _, line := range strings.Split(logText, "\n") {
					logAndWrite("    %s", line)
				}
			}
		}

		for _, container := range pod.Spec.Containers {
			prevLogs := fetchLogs(ctx, clientset, pod.Name, container.Name, 50, true)
			if prevLogs == "" {
				continue
			}
			logAndWrite("  [Previous Logs: %s/%s] (tail 50 lines)", pod.Name, container.Name)
			for _, line := range strings.Split(prevLogs, "\n") {
				logAndWrite("    %s", line)
			}
		}
	}

	logAndWrite("===== End Pod Diagnostics =====")
	return buf.String()
}

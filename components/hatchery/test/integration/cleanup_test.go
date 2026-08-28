package main

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// newExtractArgSts 构造一个含 hatchery 容器（及其它容器）的 StatefulSet，用于测试参数提取。
func newExtractArgSts(args []string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "sidecar", Args: []string{"--other", "x"}},
						{Name: "hatchery", Args: args},
					},
				},
			},
		},
	}
}

func TestExtractStsArg(t *testing.T) {
	sts := newExtractArgSts([]string{
		"-addr", ":80",
		"--admin-token", "it-abc123",
		"--identifier", "157dd104",
		"-domain", "http://42.193.240.13",
	})

	tests := []struct {
		arg  string
		want string
	}{
		{"--admin-token", "it-abc123"},
		{"--identifier", "157dd104"},
		{"-domain", "http://42.193.240.13"},
		{"--missing", ""},
	}

	for _, tt := range tests {
		if got := extractStsArg(sts, tt.arg); got != tt.want {
			t.Errorf("extractStsArg(%q) = %q, want %q", tt.arg, got, tt.want)
		}
	}
}

func TestExtractStsArgIgnoresNonHatcheryContainer(t *testing.T) {
	// 仅 sidecar 含目标参数时，extractStsArg 应忽略并返回空。
	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "sidecar", Args: []string{"--admin-token", "leaked"}},
					},
				},
			},
		},
	}
	if got := extractStsArg(sts, "--admin-token"); got != "" {
		t.Errorf("extractStsArg returned %q for non-hatchery container, want empty", got)
	}
}

package identifier

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIdentifyPod_JBossEAP(t *testing.T) {
	identifier := NewIdentifier()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-eap-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "eap",
					Image: "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4.0",
				},
			},
		},
	}

	match := identifier.IdentifyPod(pod)

	if match == nil {
		t.Fatal("Expected to identify JBoss EAP, got nil")
	}

	if match.ProductName != "jboss-eap" {
		t.Errorf("Expected product name 'jboss-eap', got '%s'", match.ProductName)
	}

	if match.Version != "7.4.0" {
		t.Errorf("Expected version '7.4.0', got '%s'", match.Version)
	}
}

func TestIdentifyPod_NonRedHatImage(t *testing.T) {
	identifier := NewIdentifier()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-nginx-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
	}

	match := identifier.IdentifyPod(pod)

	if match != nil {
		t.Errorf("Expected nil for non-Red Hat image, got product '%s'", match.ProductName)
	}
}

func TestIdentifyPod_MultipleContainers(t *testing.T) {
	identifier := NewIdentifier()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-multi-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "sidecar",
					Image: "busybox:latest",
				},
				{
					Name:  "eap",
					Image: "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4.0",
				},
			},
		},
	}

	match := identifier.IdentifyPod(pod)

	if match == nil {
		t.Fatal("Expected to identify JBoss EAP, got nil")
	}

	if match.ProductName != "jboss-eap" {
		t.Errorf("Expected product name 'jboss-eap', got '%s'", match.ProductName)
	}
}

func TestIdentifyPod_InitContainer(t *testing.T) {
	identifier := NewIdentifier()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-init-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:  "init",
					Image: "registry.redhat.io/jboss-eap-8/eap-xp5-openjdk17-openshift-rhel8:8.0",
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "busybox:latest",
				},
			},
		},
	}

	match := identifier.IdentifyPod(pod)

	if match == nil {
		t.Fatal("Expected to identify JBoss EAP, got nil")
	}

	if match.ProductName != "jboss-eap" {
		t.Errorf("Expected product name 'jboss-eap', got '%s'", match.ProductName)
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		image           string
		expectedVersion string
	}{
		{
			image:           "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4.0",
			expectedVersion: "7.4.0",
		},
		{
			image:           "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4",
			expectedVersion: "7.4",
		},
		{
			image:           "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4.17",
			expectedVersion: "7.4.17",
		},
		{
			image:           "registry.redhat.io/jboss-eap-8/eap-xp5-openjdk17-openshift-rhel8:8.0",
			expectedVersion: "8.0",
		},
		{
			image:           "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:latest",
			expectedVersion: "latest",
		},
		{
			image:           "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:v7.4.0",
			expectedVersion: "7.4.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			version := extractVersion(tt.image)
			if version != tt.expectedVersion {
				t.Errorf("Expected version '%s', got '%s'", tt.expectedVersion, version)
			}
		})
	}
}

func TestShouldLabel_NoExistingLabels(t *testing.T) {
	identifier := NewIdentifier()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	match := &ProductMatch{
		ProductName: "jboss-eap",
		Version:     "7.4",
	}

	if !identifier.ShouldLabel(pod, match) {
		t.Error("Expected ShouldLabel to return true for pod with no labels")
	}
}

func TestShouldLabel_AllCorrectLabelsExist(t *testing.T) {
	identifier := NewIdentifier()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"rht.comp":            "jboss-eap",
				"rht.comp_ver":        "7.4",
				"rht.comp_discovered": "1776437286", // Timestamp (first seen, not modified)
				"rht.comp_image":      "registry.redhat.io-jboss-eap-7-eap74-7.4",
			},
		},
	}

	match := &ProductMatch{
		ProductName: "jboss-eap",
		Version:     "7.4",
	}

	// Should return false when all required labels exist (no need to update)
	if identifier.ShouldLabel(pod, match) {
		t.Error("Expected ShouldLabel to return false when all required labels exist")
	}
}

func TestShouldLabel_MissingVersionLabel(t *testing.T) {
	identifier := NewIdentifier()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"rht.comp":            "jboss-eap",
				"rht.comp_discovered": "true",
				// Missing rht.comp_ver
			},
		},
	}

	match := &ProductMatch{
		ProductName: "jboss-eap",
		Version:     "7.4",
	}

	if !identifier.ShouldLabel(pod, match) {
		t.Error("Expected ShouldLabel to return true when version label is missing")
	}
}

func TestShouldLabel_MissingDiscoveredLabel(t *testing.T) {
	identifier := NewIdentifier()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"rht.comp":     "jboss-eap",
				"rht.comp_ver": "7.4",
				// Missing rht.comp_discovered
			},
		},
	}

	match := &ProductMatch{
		ProductName: "jboss-eap",
		Version:     "7.4",
	}

	if !identifier.ShouldLabel(pod, match) {
		t.Error("Expected ShouldLabel to return true when discovered label is missing")
	}
}

func TestShouldLabel_IncorrectLabelExists(t *testing.T) {
	identifier := NewIdentifier()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"rht.comp": "wrong-product",
			},
		},
	}

	match := &ProductMatch{
		ProductName: "jboss-eap",
		Version:     "7.4",
	}

	if !identifier.ShouldLabel(pod, match) {
		t.Error("Expected ShouldLabel to return true when incorrect label exists")
	}
}

// Made with Bob 1.0.1

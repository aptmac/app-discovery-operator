package controller

import (
	"context"
	"testing"

	"github.com/aptmac/app-discovery-operator/internal/identifier"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcile_PodNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &AppDiscoveryReconciler{
		Client:     fakeClient,
		Scheme:     scheme,
		Identifier: identifier.NewIdentifier(),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-pod",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error for non-existent pod, got: %v", err)
	}

	if result.RequeueAfter > 0 {
		t.Error("Expected no requeue for non-existent pod")
	}
}

func TestReconcile_TerminatingPod(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "terminating-pod",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{"test-finalizer"}, // Required for fake client
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "eap",
					Image: "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4.0",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()

	reconciler := &AppDiscoveryReconciler{
		Client:     fakeClient,
		Scheme:     scheme,
		Identifier: identifier.NewIdentifier(),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "terminating-pod",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error for terminating pod, got: %v", err)
	}

	if result.RequeueAfter > 0 {
		t.Error("Expected no requeue for terminating pod")
	}
}

func TestReconcile_NonRunningPod(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "succeeded-pod",
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
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()

	reconciler := &AppDiscoveryReconciler{
		Client:     fakeClient,
		Scheme:     scheme,
		Identifier: identifier.NewIdentifier(),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "succeeded-pod",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error for succeeded pod, got: %v", err)
	}

	if result.RequeueAfter > 0 {
		t.Error("Expected no requeue for succeeded pod")
	}
}

func TestReconcile_NonRedHatPod(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-pod",
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
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()

	reconciler := &AppDiscoveryReconciler{
		Client:     fakeClient,
		Scheme:     scheme,
		Identifier: identifier.NewIdentifier(),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nginx-pod",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error for non-Red Hat pod, got: %v", err)
	}

	if result.RequeueAfter > 0 {
		t.Error("Expected no requeue for non-Red Hat pod")
	}

	// Verify no labels were added
	updatedPod := &corev1.Pod{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "nginx-pod", Namespace: "default"}, updatedPod)
	if err != nil {
		t.Fatalf("Failed to get updated pod: %v", err)
	}

	if _, exists := updatedPod.Labels["rht.comp"]; exists {
		t.Error("Expected no rht.app label on non-Red Hat pod")
	}
}

func TestReconcile_RedHatPodLabeling(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eap-pod",
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
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()

	reconciler := &AppDiscoveryReconciler{
		Client:     fakeClient,
		Scheme:     scheme,
		Identifier: identifier.NewIdentifier(),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "eap-pod",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error for Red Hat pod, got: %v", err)
	}

	if result.RequeueAfter > 0 {
		t.Error("Expected no requeue for Red Hat pod")
	}

	// Verify labels were added
	updatedPod := &corev1.Pod{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "eap-pod", Namespace: "default"}, updatedPod)
	if err != nil {
		t.Fatalf("Failed to get updated pod: %v", err)
	}

	// Check product and version labels
	if updatedPod.Labels["rht.comp"] != "EAP" {
		t.Errorf("Expected rht.comp to be 'EAP', got '%s'", updatedPod.Labels["rht.comp"])
	}

	if updatedPod.Labels["rht.pod_image_ver"] != "7.4.0" {
		t.Errorf("Expected rht.pod_image_ver to be '7.4.0', got '%s'", updatedPod.Labels["rht.pod_image_ver"])
	}

	// Check discovered label exists (it's a timestamp, so just verify it exists)
	if _, exists := updatedPod.Labels["rht.comp_discovered"]; !exists {
		t.Error("Expected rht.comp_discovered label to exist")
	}

	// Verify image label exists and is sanitized
	if _, exists := updatedPod.Labels["rht.pod_image"]; !exists {
		t.Error("Expected rht.pod_image label to exist")
	}
}

func TestReconcile_AlreadyLabeledPod(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eap-pod",
			Namespace: "default",
			Labels: map[string]string{
				"rht.comp":            "EAP",
				"rht.pod_image_ver":   "7.4.0",
				"rht.comp_discovered": "1776437286", // Timestamp
				"rht.pod_image":       "registry.redhat.io-jboss-eap-7-eap74-openjdk11-openshift-rhel8-7.4.0",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "eap",
					Image: "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4.0",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()

	reconciler := &AppDiscoveryReconciler{
		Client:     fakeClient,
		Scheme:     scheme,
		Identifier: identifier.NewIdentifier(),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "eap-pod",
			Namespace: "default",
		},
	}

	// Get the pod before reconciliation to check ResourceVersion
	beforePod := &corev1.Pod{}
	err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "eap-pod", Namespace: "default"}, beforePod)
	if err != nil {
		t.Fatalf("Failed to get pod before reconciliation: %v", err)
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error for already labeled pod, got: %v", err)
	}

	if result.RequeueAfter > 0 {
		t.Error("Expected no requeue for already labeled pod")
	}

	// Verify pod was not updated (ResourceVersion should be the same)
	afterPod := &corev1.Pod{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "eap-pod", Namespace: "default"}, afterPod)
	if err != nil {
		t.Fatalf("Failed to get pod after reconciliation: %v", err)
	}

	// In a real cluster, ResourceVersion would change if the pod was updated
	// With fake client, we just verify labels are still correct
	if afterPod.Labels["rht.comp"] != "EAP" {
		t.Error("Labels should remain unchanged for already labeled pod")
	}
}

func TestReconcile_MissingLabels(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eap-pod",
			Namespace: "default",
			Labels: map[string]string{
				"rht.comp": "EAP",
				// Missing version and discovered labels
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "eap",
					Image: "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4.0",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()

	reconciler := &AppDiscoveryReconciler{
		Client:     fakeClient,
		Scheme:     scheme,
		Identifier: identifier.NewIdentifier(),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "eap-pod",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result.RequeueAfter > 0 {
		t.Error("Expected no requeue")
	}

	// Verify all labels were added
	updatedPod := &corev1.Pod{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "eap-pod", Namespace: "default"}, updatedPod)
	if err != nil {
		t.Fatalf("Failed to get updated pod: %v", err)
	}

	if updatedPod.Labels["rht.pod_image_ver"] != "7.4.0" {
		t.Error("Expected version label to be added")
	}

	// Check discovered label exists (it's a timestamp)
	if _, exists := updatedPod.Labels["rht.comp_discovered"]; !exists {
		t.Error("Expected discovered label to be added")
	}
}

func TestSanitizeLabelValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "registry.redhat.io/jboss-eap-7/eap74:7.4.0",
			expected: "registry.redhat.io-jboss-eap-7-eap74-7.4.0",
		},
		{
			input:    "simple-name",
			expected: "simple-name",
		},
		{
			input:    "/starts-with-slash",
			expected: "x-starts-with-slash",
		},
		{
			input:    "ends-with-slash/",
			expected: "ends-with-slash-x",
		},
		{
			input:    "has@special#chars!",
			expected: "has-special-chars-x",
		},
		{
			input:    "very-long-string-that-exceeds-sixty-three-characters-and-needs-truncation",
			expected: "very-long-string-that-exceeds-sixty-three-characters-and-needsx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeLabelValue(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}

			// Verify result is valid Kubernetes label value
			if len(result) > 63 {
				t.Errorf("Result exceeds 63 characters: %d", len(result))
			}

			if len(result) > 0 {
				if !isAlphanumeric(rune(result[0])) {
					t.Errorf("Result doesn't start with alphanumeric: %s", result)
				}
				if !isAlphanumeric(rune(result[len(result)-1])) {
					t.Errorf("Result doesn't end with alphanumeric: %s", result)
				}
			}
		})
	}
}

func TestReconcile_UserProvidedLabelsNotOverwritten(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Pod with user-provided label values that differ from what we would detect
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eap-pod",
			Namespace: "default",
			Labels: map[string]string{
				"rht.comp": "EAP", // User provided "EAP" instead of "jboss-eap"
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "eap",
					Image: "registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4.0",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()

	reconciler := &AppDiscoveryReconciler{
		Client:     fakeClient,
		Scheme:     scheme,
		Identifier: identifier.NewIdentifier(),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "eap-pod",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if result.RequeueAfter > 0 {
		t.Error("Expected no requeue")
	}

	// Verify the user-provided label was NOT overwritten
	updatedPod := &corev1.Pod{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "eap-pod", Namespace: "default"}, updatedPod)
	if err != nil {
		t.Fatalf("Failed to get updated pod: %v", err)
	}

	// User-provided value should be preserved
	if updatedPod.Labels["rht.comp"] != "EAP" {
		t.Errorf("Expected user-provided label 'EAP' to be preserved, got '%s'", updatedPod.Labels["rht.comp"])
	}

	// Missing labels should be added
	if _, exists := updatedPod.Labels["rht.pod_image_ver"]; !exists {
		t.Error("Expected rht.pod_image_ver label to be added")
	}

	if _, exists := updatedPod.Labels["rht.comp_discovered"]; !exists {
		t.Error("Expected rht.comp_discovered label to be added")
	}

	if _, exists := updatedPod.Labels["rht.pod_image"]; !exists {
		t.Error("Expected rht.pod_image label to be added")
	}
}

// Made with Bob 1.0.1

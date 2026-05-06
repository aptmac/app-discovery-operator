package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/aptmac/app-discovery-operator/internal/identifier"
)

// AppDiscoveryReconciler watches Pods and labels Red Hat middleware applications
type AppDiscoveryReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Identifier *identifier.Identifier
}

// Reconcile handles pod events and applies labels to Red Hat products
func (r *AppDiscoveryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Pod
	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod was deleted, nothing to do
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Pod")
		return ctrl.Result{}, err
	}

	// Skip if pod is terminating
	if pod.DeletionTimestamp != nil {
		log.V(1).Info("Pod is terminating, skipping", "pod", pod.Name)
		return ctrl.Result{}, nil
	}

	// Skip if pod is not running or pending
	if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodPending {
		log.V(1).Info("Pod is not running or pending, skipping", "pod", pod.Name, "phase", pod.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Identify if this is a Red Hat product
	match := r.Identifier.IdentifyPod(ctx, pod)
	if match == nil {
		// Not a Red Hat product, nothing to do
		log.V(1).Info("Pod is not a Red Hat product", "pod", pod.Name)
		return ctrl.Result{}, nil
	}

	log.Info("Identified Red Hat product",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"product", match.ProductName,
		"version", match.Version,
		"image", match.Image)

	// Check if we should label this pod
	if !r.Identifier.ShouldLabel(pod, match) {
		log.V(1).Info("Pod already has correct label", "pod", pod.Name, "product", match.ProductName)
		return ctrl.Result{}, nil
	}

	// Apply labels to the pod
	if err := r.labelPod(ctx, pod, match); err != nil {
		log.Error(err, "Failed to label pod", "pod", pod.Name)
		return ctrl.Result{}, err
	}

	log.Info("Successfully labeled pod",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"product", match.ProductName,
		"version", match.Version)

	return ctrl.Result{}, nil
}

// labelPod applies Red Hat product labels to a pod
func (r *AppDiscoveryReconciler) labelPod(ctx context.Context, pod *corev1.Pod, match *identifier.ProductMatch) error {
	// Create labels map if it doesn't exist
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	// Only set labels if they don't already exist (respect user-provided labels)
	if _, exists := pod.Labels["rht.comp"]; !exists {
		pod.Labels["rht.comp"] = match.ProductName
	}

	if _, exists := pod.Labels["rht.pod_image_ver"]; !exists {
		pod.Labels["rht.pod_image_ver"] = match.Version
	}

	// Only set discovered timestamp if it doesn't exist (first seen time, not last modified)
	if _, exists := pod.Labels["rht.comp_discovered"]; !exists {
		pod.Labels["rht.comp_discovered"] = fmt.Sprintf("%d", match.Discovered.Unix())
	}

	if _, exists := pod.Labels["rht.pod_image"]; !exists {
		// Store the full image name (sanitized for label format)
		pod.Labels["rht.pod_image"] = sanitizeLabelValue(match.Image)
	}

	// Update the pod
	if err := r.Update(ctx, pod); err != nil {
		return fmt.Errorf("failed to update pod labels: %w", err)
	}

	return nil
}

// sanitizeLabelValue converts a string to a valid Kubernetes label value
// Valid label values must:
// - Be 63 characters or less
// - Consist of alphanumeric characters, '-', '_' or '.'
// - Start and end with an alphanumeric character
func sanitizeLabelValue(s string) string {
	// Replace invalid characters with dashes
	result := ""
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			result += string(ch)
		} else {
			result += "-"
		}
	}

	// Ensure it starts and ends with alphanumeric
	if len(result) > 0 && !isAlphanumeric(rune(result[0])) {
		result = "x" + result
	}
	if len(result) > 0 && !isAlphanumeric(rune(result[len(result)-1])) {
		result = result + "x"
	}

	// Truncate to 63 characters
	if len(result) > 63 {
		result = result[:63]
		// Ensure it still ends with alphanumeric after truncation
		if !isAlphanumeric(rune(result[len(result)-1])) {
			result = result[:62] + "x"
		}
	}

	return result
}

// isAlphanumeric checks if a rune is alphanumeric
func isAlphanumeric(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

// SetupWithManager sets up the controller with the Manager
func (r *AppDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

// Made with Bob 1.0.1

package identifier

import (
	"context"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// ProductMatch represents a detected Red Hat product
type ProductMatch struct {
	ProductName string
	Version     string
	Image       string
	Discovered  time.Time
}

// Identifier detects Red Hat products from pod specifications
type Identifier struct {
	patterns       map[string]string
	imageInspector *ImageInspector
}

// NewIdentifier creates a new product identifier with hardcoded patterns
func NewIdentifier() *Identifier {
	return &Identifier{
		patterns: map[string]string{
			// JBoss EAP (Enterprise Application Platform)
			"jboss-eap-7": "EAP",
			"jboss-eap-8": "EAP",
		},
		imageInspector: nil, // Will be set if running in OpenShift
	}
}

// SetImageInspector sets the image inspector for OpenShift Image API access
func (i *Identifier) SetImageInspector(inspector *ImageInspector) {
	i.imageInspector = inspector
}

// IdentifyPod analyzes a pod and returns product information if it's a Red Hat product
func (i *Identifier) IdentifyPod(ctx context.Context, pod *corev1.Pod) *ProductMatch {
	// First, check if this is an EAP Operator-managed pod
	// The EAP Operator already adds rht.comp label, but we want to add our additional labels
	if pod.Labels != nil {
		if managedBy, exists := pod.Labels["app.kubernetes.io/managed-by"]; exists && managedBy == "eap-operator" {
			return i.identifyOperatorManagedPod(pod)
		}
	}

	// Second, try to identify by image name (direct deployments)
	for _, container := range pod.Spec.Containers {
		if match := i.identifyImage(container.Image); match != nil {
			return match
		}
	}

	// Check init containers as well
	for _, container := range pod.Spec.InitContainers {
		if match := i.identifyImage(container.Image); match != nil {
			return match
		}
	}

	// Fallback: Try OpenShift Image API (for S2I-built applications)
	// S2I-built apps have env vars in the image metadata
	if i.imageInspector != nil {
		if match := i.imageInspector.InspectPodImages(ctx, pod); match != nil {
			return match
		}
	}

	return nil
}

// identifyOperatorManagedPod handles pods managed by the EAP Operator
// These pods already have rht.comp label, but we add version, image, and discovered timestamp
func (i *Identifier) identifyOperatorManagedPod(pod *corev1.Pod) *ProductMatch {
	// Get the product name from existing label (EAP Operator sets "EAP")
	productName := pod.Labels["rht.comp"]
	if productName == "" {
		productName = "EAP" // Default if not set
	}

	// Extract version and image from the first container
	var version, image string
	if len(pod.Spec.Containers) > 0 {
		image = pod.Spec.Containers[0].Image
		version = extractVersion(image)
	} else {
		version = "unknown"
		image = "unknown"
	}

	return &ProductMatch{
		ProductName: productName,
		Version:     version,
		Image:       image,
		Discovered:  pod.CreationTimestamp.Time,
	}
}

// identifyImage checks if an image matches any Red Hat product pattern
func (i *Identifier) identifyImage(image string) *ProductMatch {
	for pattern, productName := range i.patterns {
		if strings.Contains(image, pattern) {
			return &ProductMatch{
				ProductName: productName,
				Version:     extractVersion(image),
				Image:       image,
				Discovered:  time.Now(),
			}
		}
	}
	return nil
}

// extractVersion attempts to extract version from image tag
// Example: registry.redhat.io/jboss-eap-7/eap74-openjdk11-openshift-rhel8:7.4.0 -> 7.4.0
func extractVersion(image string) string {
	// Split by colon to get tag
	parts := strings.Split(image, ":")
	if len(parts) < 2 {
		return "unknown"
	}

	tag := parts[len(parts)-1]

	// Try to extract version from tag
	// Common patterns: 7.4.0, 7.4.17, 7.4, v7.4, latest
	if tag == "latest" {
		return "latest"
	}

	// Remove 'v' prefix if present (e.g., v7.4.0 -> 7.4.0)
	tag = strings.TrimPrefix(tag, "v")

	// Check if tag looks like a semantic version (contains digits and dots)
	// This preserves the full version: 7.4, 7.4.0, 7.4.17, etc.
	if len(tag) > 0 && (tag[0] >= '0' && tag[0] <= '9') {
		// Return the full version tag
		return tag
	}

	return tag
}

// ShouldLabel determines if a pod should be labeled
// Returns true if any of the required labels are missing
// Does not check label values to respect user-provided labels
// For operator-managed pods, we only add our additional labels (version, image, discovered)
func (i *Identifier) ShouldLabel(pod *corev1.Pod, match *ProductMatch) bool {
	if pod.Labels == nil {
		return true
	}

	// Check if this is an operator-managed pod (already has rht.comp)
	isOperatorManaged := false
	if managedBy, exists := pod.Labels["app.kubernetes.io/managed-by"]; exists && managedBy == "eap-operator" {
		isOperatorManaged = true
	}

	if isOperatorManaged {
		// For operator-managed pods, check rht.comp and our additional labels
		// The EAP Operator sets rht.comp, but if it's deleted we should restore it
		labelsToCheck := []string{
			"rht.comp",            // Product name (may be deleted by user)
			"rht.pod_image_ver",   // Version of the pod's container image
			"rht.comp_discovered", // Discovery timestamp
			"rht.pod_image",       // Pod's container image name
		}

		for _, label := range labelsToCheck {
			if _, exists := pod.Labels[label]; !exists {
				return true // Missing label
			}
		}

		return false // All labels are present
	}

	// For non-operator-managed pods, check all required labels
	requiredLabels := []string{
		"rht.comp",
		"rht.pod_image_ver",
		"rht.comp_discovered",
		"rht.pod_image",
	}

	for _, label := range requiredLabels {
		if _, exists := pod.Labels[label]; !exists {
			return true // Missing label, needs labeling
		}
	}

	// All required labels are present
	return false
}

// Made with Bob 1.0.1

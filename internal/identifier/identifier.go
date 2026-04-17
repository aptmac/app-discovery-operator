package identifier

import (
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
	patterns map[string]string
}

// NewIdentifier creates a new product identifier with hardcoded patterns
func NewIdentifier() *Identifier {
	return &Identifier{
		patterns: map[string]string{
			// JBoss EAP (Enterprise Application Platform)
			"registry.redhat.io/jboss-eap-7":         "jboss-eap",
			"registry.redhat.io/jboss-eap-8":         "jboss-eap",
			"registry.redhat.io/jboss-eap/jboss-eap": "jboss-eap",
		},
	}
}

// IdentifyPod analyzes a pod and returns product information if it's a Red Hat product
func (i *Identifier) IdentifyPod(pod *corev1.Pod) *ProductMatch {
	// Check all containers in the pod
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

	return nil
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
func (i *Identifier) ShouldLabel(pod *corev1.Pod, match *ProductMatch) bool {
	if pod.Labels == nil {
		return true
	}

	// Check if any required labels are missing (don't check values)
	requiredLabels := []string{
		"rht.comp",
		"rht.comp_ver",
		"rht.comp_discovered",
		"rht.comp_image",
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

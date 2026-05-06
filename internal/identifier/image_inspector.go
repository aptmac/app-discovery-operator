package identifier

import (
	"context"
	"encoding/json"
	"strings"

	imagev1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DockerImageConfig represents the Config section of a Docker image
type DockerImageConfig struct {
	Env []string `json:"Env,omitempty"`
}

// DockerImageMetadata represents the metadata of a Docker image
type DockerImageMetadata struct {
	Config *DockerImageConfig `json:"Config,omitempty"`
}

// ImageInspector provides OpenShift Image API inspection capabilities
type ImageInspector struct {
	client client.Client
}

// NewImageInspector creates a new image inspector with the given client
func NewImageInspector(c client.Client) *ImageInspector {
	return &ImageInspector{
		client: c,
	}
}

// InspectPodImages checks pod container images using OpenShift Image API
// This is used to detect S2I-built applications where env vars are in the image
func (ii *ImageInspector) InspectPodImages(ctx context.Context, pod *corev1.Pod) *ProductMatch {
	log := log.FromContext(ctx)

	// Check if we have a client (might be nil in vanilla Kubernetes)
	if ii.client == nil {
		return nil
	}

	// Inspect each container's image
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if match := ii.inspectImage(ctx, containerStatus.ImageID); match != nil {
			log.V(1).Info("Detected product from image metadata",
				"container", containerStatus.Name,
				"imageID", containerStatus.ImageID,
				"product", match.ProductName)
			return match
		}
	}

	return nil
}

// inspectImage queries the OpenShift Image API to get image metadata
func (ii *ImageInspector) inspectImage(ctx context.Context, imageID string) *ProductMatch {
	log := log.FromContext(ctx)

	// Extract SHA from imageID
	// Format: registry.example.com/namespace/image@sha256:abc123...
	sha := extractSHA(imageID)
	if sha == "" {
		log.V(1).Info("Could not extract SHA from imageID", "imageID", imageID)
		return nil
	}

	// Query OpenShift Image API
	image := &imagev1.Image{}
	err := ii.client.Get(ctx, types.NamespacedName{Name: sha}, image)
	if err != nil {
		// Image not found or not in OpenShift - this is expected in vanilla K8s
		log.V(1).Info("Could not get image from OpenShift API", "sha", sha, "error", err.Error())
		return nil
	}

	// Check environment variables in image metadata
	return ii.checkImageEnvVars(image)
}

// checkImageEnvVars inspects image environment variables for product markers
func (ii *ImageInspector) checkImageEnvVars(image *imagev1.Image) *ProductMatch {
	// Unmarshal the DockerImageMetadata RawExtension
	var metadata DockerImageMetadata
	if err := json.Unmarshal(image.DockerImageMetadata.Raw, &metadata); err != nil {
		return nil
	}

	if metadata.Config == nil {
		return nil
	}

	// Parse environment variables from image config into a map
	envMap := parseEnvArray(metadata.Config.Env)

	// Check if this is an EAP image
	if envMap["JBOSS_PRODUCT"] != "eap" || envMap["JBOSS_HOME"] != "/opt/eap" {
		return nil
	}

	version := envMap["JBOSS_EAP_VERSION"]
	if version == "" {
		version = "unknown"
	}

	// Use the builder image name if available, otherwise use the image name
	imageName := envMap["JBOSS_IMAGE_NAME"]
	if imageName == "" {
		imageName = image.Name
	}

	return &ProductMatch{
		ProductName: "jboss-eap",
		Version:     version,
		Image:       imageName,
		Discovered:  image.CreationTimestamp.Time,
	}
}

// parseEnvArray converts Docker env array format (KEY=VALUE) to a map
func parseEnvArray(envArray []string) map[string]string {
	result := make(map[string]string, len(envArray))
	for _, env := range envArray {
		if idx := strings.Index(env, "="); idx >= 0 {
			result[env[:idx]] = env[idx+1:]
		}
	}
	return result
}

// extractSHA extracts the SHA256 digest from an image ID
// Input formats:
//   - image-registry.openshift-image-registry.svc:5000/namespace/image@sha256:abc123...
//   - sha256:abc123...
//
// Output: sha256:abc123...
func extractSHA(imageID string) string {
	// Check if it already starts with sha256:
	if strings.HasPrefix(imageID, "sha256:") {
		return imageID
	}

	// Look for @sha256: pattern
	if idx := strings.Index(imageID, "@sha256:"); idx != -1 {
		return imageID[idx+1:] // Skip the @ symbol
	}

	return ""
}

// Made with Bob 1.0.2

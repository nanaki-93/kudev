// pkg/registry/loader.go

package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/nanaki-93/kudev/pkg/logging"
)

// ClusterType identifies the type of local K8s cluster.
type ClusterType string

const (
	ClusterTypeDockerDesktop ClusterType = "docker-desktop"
	ClusterTypeMinikube      ClusterType = "minikube"
	ClusterTypeKind          ClusterType = "kind"
	ClusterTypeUnknown       ClusterType = "unknown"
)

// Loader is the interface for cluster-specific image loading.
type Loader interface {
	// Load loads an image into the cluster.
	Load(ctx context.Context, imageRef string) error

	// Name returns the loader identifier.
	Name() string
}

// Registry orchestrates image loading based on cluster type.
type Registry struct {
	kubeContext string
	logger      logging.LoggerInterface
}

// NewRegistry creates a new registry loader.
// kubeContext is the current kubectl context name.
func NewRegistry(kubeContext string, logger logging.LoggerInterface) *Registry {
	return &Registry{
		kubeContext: kubeContext,
		logger:      logger,
	}
}

// Load loads an image into the current cluster.
func (r *Registry) Load(ctx context.Context, imageRef string) error {
	r.logger.Info("loading image to cluster",
		"image", imageRef,
		"context", r.kubeContext,
	)

	// Detect cluster type
	clusterType, clusterName := detectClusterType(r.kubeContext)

	r.logger.Debug("detected cluster type",
		"type", clusterType,
		"clusterName", clusterName,
	)

	// Get appropriate loader
	loader, err := r.getLoader(clusterType, clusterName)
	if err != nil {
		return err
	}

	r.logger.Debug("using loader", "loader", loader.Name())

	// Load the image
	if err := loader.Load(ctx, imageRef); err != nil {
		return fmt.Errorf("failed to load image with %s loader: %w", loader.Name(), err)
	}

	r.logger.Info("image loaded successfully",
		"image", imageRef,
		"loader", loader.Name(),
	)

	return nil
}

// getLoader returns the appropriate loader for the cluster type.
func (r *Registry) getLoader(clusterType ClusterType, clusterName string) (Loader, error) {
	switch clusterType {
	case ClusterTypeDockerDesktop:
		return newDockerDesktopLoader(r.logger), nil

	case ClusterTypeMinikube:
		return newMinikubeLoader(r.logger), nil

	case ClusterTypeKind:
		return newKindLoader(clusterName, r.logger), nil

	case ClusterTypeUnknown:
		return nil, fmt.Errorf(
			"unknown cluster type for context %q\n\n"+
				"Supported clusters:\n"+
				"  - Docker Desktop (context: docker-desktop)\n"+
				"  - Minikube (context: minikube)\n"+
				"  - Kind (context: kind-<cluster-name>)\n\n"+
				"Tips:\n"+
				"  - Check current context: kubectl config current-context\n"+
				"  - List contexts: kubectl config get-contexts\n"+
				"  - Switch context: kubectl config use-context <name>",
			r.kubeContext,
		)

	default:
		return nil, fmt.Errorf("unhandled cluster type: %s", clusterType)
	}
}

// detectClusterType determines the cluster type from context name.
func detectClusterType(kubeContext string) (ClusterType, string) {
	ctx := strings.ToLower(kubeContext)

	switch {
	case strings.Contains(ctx, "docker-desktop"),
		strings.Contains(ctx, "docker-for-desktop"):
		return ClusterTypeDockerDesktop, ""

	case strings.Contains(ctx, "minikube"):
		return ClusterTypeMinikube, ""

	case strings.HasPrefix(ctx, "kind-"):
		// Extract cluster name: "kind-dev" → "dev"
		clusterName := strings.TrimPrefix(ctx, "kind-")
		return ClusterTypeKind, clusterName

	default:
		return ClusterTypeUnknown, ""
	}
}

// GetClusterType returns the detected cluster type for the current context.
// Useful for debugging and testing.
func (r *Registry) GetClusterType() (ClusterType, string) {
	return detectClusterType(r.kubeContext)
}

// KubeContext returns the kubernetes context being used.
func (r *Registry) KubeContext() string {
	return r.kubeContext
}

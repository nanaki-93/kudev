// pkg/registry/docker.go

package registry

import (
	"context"

	"github.com/nanaki-93/kudev/pkg/logging"
)

// dockerDesktopLoader handles image loading for Docker Desktop.
type dockerDesktopLoader struct {
	logger logging.LoggerInterface
}

// newDockerDesktopLoader creates a new Docker Desktop loader.
func newDockerDesktopLoader(logger logging.LoggerInterface) *dockerDesktopLoader {
	return &dockerDesktopLoader{logger: logger}
}

// Name returns the loader identifier.
func (d *dockerDesktopLoader) Name() string {
	return "docker-desktop"
}

// Load loads an image into Docker Desktop's Kubernetes.
// Docker Desktop shares the Docker daemon with its built-in K8s cluster,
// so images built locally are automatically available - no loading needed.
func (d *dockerDesktopLoader) Load(ctx context.Context, imageRef string) error {
	d.logger.Info("image available to Docker Desktop automatically",
		"image", imageRef,
		"reason", "Docker Desktop shares daemon with K8s",
	)

	// No action needed - Docker Desktop K8s uses the same Docker daemon
	// that was used to build the image
	return nil
}

// Ensure dockerDesktopLoader implements Loader
var _ Loader = (*dockerDesktopLoader)(nil)

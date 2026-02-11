// pkg/registry/minikube.go

package registry

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nanaki-93/kudev/pkg/logging"
)

// minikubeLoader handles image loading for Minikube.
type minikubeLoader struct {
	logger logging.LoggerInterface
}

// newMinikubeLoader creates a new Minikube loader.
func newMinikubeLoader(logger logging.LoggerInterface) *minikubeLoader {
	return &minikubeLoader{logger: logger}
}

// Name returns the loader identifier.
func (m *minikubeLoader) Name() string {
	return "minikube"
}

// Load loads an image into Minikube using `minikube image load`.
func (m *minikubeLoader) Load(ctx context.Context, imageRef string) error {
	m.logger.Info("loading image via minikube",
		"image", imageRef,
		"command", "minikube image load",
	)

	// Check if minikube is available
	if err := m.checkMinikube(ctx); err != nil {
		return err
	}

	// Run minikube image load
	cmd := exec.CommandContext(ctx, "minikube", "image", "load", imageRef)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(
			"minikube image load failed\n\n"+
				"Command: minikube image load %s\n"+
				"Output: %s\n"+
				"Error: %w\n\n"+
				"Troubleshooting:\n"+
				"  - Ensure Minikube is running: minikube status\n"+
				"  - Start Minikube: minikube start\n"+
				"  - Check image exists: docker images %s",
			imageRef, strings.TrimSpace(string(output)), err, imageRef,
		)
	}

	m.logger.Info("image loaded to minikube successfully",
		"image", imageRef,
	)

	return nil
}

// checkMinikube verifies minikube CLI is available.
func (m *minikubeLoader) checkMinikube(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "minikube", "version", "--short")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(
			"minikube CLI not found or not working\n\n"+
				"Please install Minikube:\n"+
				"  - macOS: brew install minikube\n"+
				"  - Windows: choco install minikube\n"+
				"  - Linux: see https://minikube.sigs.k8s.io/docs/start/\n\n"+
				"Error: %w",
			err,
		)
	}

	m.logger.Debug("minikube CLI available",
		"version", strings.TrimSpace(string(output)),
	)

	return nil
}

// Ensure minikubeLoader implements Loader
var _ Loader = (*minikubeLoader)(nil)

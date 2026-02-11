// pkg/registry/kind.go

package registry

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nanaki-93/kudev/pkg/logging"
)

// kindLoader handles image loading for Kind clusters.
type kindLoader struct {
	clusterName string
	logger      logging.LoggerInterface
}

// newKindLoader creates a new Kind loader.
// clusterName is extracted from the context (e.g., "kind-dev" → "dev").
func newKindLoader(clusterName string, logger logging.LoggerInterface) *kindLoader {
	// Default to "kind" if no cluster name provided
	if clusterName == "" {
		clusterName = "kind"
	}

	return &kindLoader{
		clusterName: clusterName,
		logger:      logger,
	}
}

// Name returns the loader identifier.
func (k *kindLoader) Name() string {
	return "kind"
}

// ClusterName returns the Kind cluster name.
func (k *kindLoader) ClusterName() string {
	return k.clusterName
}

// Load loads an image into Kind using `kind load docker-image`.
func (k *kindLoader) Load(ctx context.Context, imageRef string) error {
	k.logger.Info("loading image via kind",
		"image", imageRef,
		"cluster", k.clusterName,
		"command", "kind load docker-image",
	)

	// Check if kind is available
	if err := k.checkKind(ctx); err != nil {
		return err
	}

	// Run kind load docker-image
	cmd := exec.CommandContext(ctx,
		"kind", "load", "docker-image", imageRef,
		"--name", k.clusterName,
	)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(
			"kind load failed\n\n"+
				"Command: kind load docker-image %s --name %s\n"+
				"Output: %s\n"+
				"Error: %w\n\n"+
				"Troubleshooting:\n"+
				"  - Ensure Kind cluster exists: kind get clusters\n"+
				"  - Create cluster: kind create cluster --name %s\n"+
				"  - Check image exists: docker images %s",
			imageRef, k.clusterName,
			strings.TrimSpace(string(output)), err,
			k.clusterName, imageRef,
		)
	}

	k.logger.Info("image loaded to kind cluster successfully",
		"image", imageRef,
		"cluster", k.clusterName,
	)

	return nil
}

// checkKind verifies kind CLI is available.
func (k *kindLoader) checkKind(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "kind", "version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(
			"kind CLI not found or not working\n\n"+
				"Please install Kind:\n"+
				"  - macOS: brew install kind\n"+
				"  - Windows: choco install kind\n"+
				"  - Go: go install sigs.k8s.io/kind@latest\n"+
				"  - See: https://kind.sigs.k8s.io/docs/user/quick-start/\n\n"+
				"Error: %w",
			err,
		)
	}

	k.logger.Debug("kind CLI available",
		"version", strings.TrimSpace(string(output)),
	)

	return nil
}

// Ensure kindLoader implements Loader
var _ Loader = (*kindLoader)(nil)

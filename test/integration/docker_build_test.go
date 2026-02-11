package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nanaki-93/kudev/pkg/builder"
	"github.com/nanaki-93/kudev/pkg/builder/docker"
	"github.com/nanaki-93/kudev/test/util"
)

func TestDockerBuildIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create temp directory with Dockerfile
	tmpDir := t.TempDir()
	dockerfile := `FROM alpine:latest
RUN echo "test"
`
	err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(dockerfile), 0644)
	if err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	logger := &util.MockLogger{}
	db := docker.NewBuilder(logger)

	opts := builder.BuildOptions{
		SourceDir:      tmpDir,
		DockerfilePath: "./Dockerfile",
		ImageName:      "kudev-test",
		ImageTag:       "integration-test",
	}

	ctx := context.Background()
	result, err := db.Build(ctx, opts)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if result.FullRef != "kudev-test:integration-test" {
		t.Errorf("unexpected FullRef: %s", result.FullRef)
	}

	if result.ID == "" {
		t.Error("expected non-empty image ID")
	}

	// Cleanup
	cleanupCmd := exec.Command("docker", "rmi", result.FullRef)
	cleanupCmd.Run()
}

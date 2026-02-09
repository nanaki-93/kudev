package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nanaki-93/kudev/pkg/builder"
	"github.com/nanaki-93/kudev/pkg/builder/docker"
	"github.com/nanaki-93/kudev/pkg/logging"
)

type mockLogger struct {
	messages []string
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, msg)
}

func (m *mockLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, msg)
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, msg)
}
func (m *mockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, msg)
}
func (m *mockLogger) WithValues(keysAndValues ...interface{}) logging.LoggerInterface {
	return &mockLogger{
		messages: m.messages,
	}
}

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

	logger := &mockLogger{}
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

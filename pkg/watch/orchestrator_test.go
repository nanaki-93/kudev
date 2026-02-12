// pkg/watch/orchestrator_test.go

package watch

import (
	"context"
	"testing"

	"github.com/nanaki-93/kudev/pkg/builder"
	"github.com/nanaki-93/kudev/pkg/deployer"
)

type mockBuilder struct {
	buildCount int
	buildErr   error
}

func (m *mockBuilder) Build(ctx context.Context, opts builder.BuildOptions) (*builder.ImageRef, error) {
	m.buildCount++
	return &builder.ImageRef{FullRef: "test:latest"}, m.buildErr
}

func (m *mockBuilder) Name() string { return "mock" }

type mockDeployer struct {
	deployCount int
}

func (m *mockDeployer) Upsert(ctx context.Context, opts deployer.DeploymentOptions) (*deployer.DeploymentStatus, error) {
	m.deployCount++
	return &deployer.DeploymentStatus{Status: "Running"}, nil
}

func (m *mockDeployer) Delete(ctx context.Context, name, ns string) error { return nil }
func (m *mockDeployer) Status(ctx context.Context, name, ns string) (*deployer.DeploymentStatus, error) {
	return &deployer.DeploymentStatus{}, nil
}

func TestOrchestrator_SkipsIfHashUnchanged(t *testing.T) {
	// This would require more setup with temp directories
	// and actual file operations
	t.Skip("requires full integration setup")
}

func TestOrchestrator_OnlyOneRebuildAtATime(t *testing.T) {
	// Test that concurrent events don't cause concurrent rebuilds
	t.Skip("requires full integration setup")
}

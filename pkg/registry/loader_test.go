package registry

import (
	"context"
	"testing"

	"github.com/nanaki-93/kudev/test/util"
)

func TestDetectClusterType(t *testing.T) {
	tests := []struct {
		context     string
		wantType    ClusterType
		wantCluster string
	}{
		{"docker-desktop", ClusterTypeDockerDesktop, ""},
		{"docker-for-desktop", ClusterTypeDockerDesktop, ""},
		{"Docker-Desktop", ClusterTypeDockerDesktop, ""}, // Case insensitive

		{"minikube", ClusterTypeMinikube, ""},
		{"Minikube", ClusterTypeMinikube, ""},

		{"kind-dev", ClusterTypeKind, "dev"},
		{"kind-test", ClusterTypeKind, "test"},
		{"kind-production", ClusterTypeKind, "production"},
		{"Kind-Dev", ClusterTypeKind, "dev"}, // Case insensitive

		{"unknown-context", ClusterTypeUnknown, ""},
		{"gke_project_zone_cluster", ClusterTypeUnknown, ""},
		{"arn:aws:eks:region:account:cluster/name", ClusterTypeUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.context, func(t *testing.T) {
			gotType, gotCluster := detectClusterType(tt.context)

			if gotType != tt.wantType {
				t.Errorf("detectClusterType(%q) type = %v, want %v",
					tt.context, gotType, tt.wantType)
			}

			if gotCluster != tt.wantCluster {
				t.Errorf("detectClusterType(%q) cluster = %q, want %q",
					tt.context, gotCluster, tt.wantCluster)
			}
		})
	}
}

func TestRegistry_GetLoader(t *testing.T) {
	logger := &util.MockLogger{}

	tests := []struct {
		context    string
		wantLoader string
		wantErr    bool
	}{
		{"docker-desktop", "docker-desktop", false},
		{"minikube", "minikube", false},
		{"kind-dev", "kind", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.context, func(t *testing.T) {
			r := NewRegistry(tt.context, logger)
			clusterType, clusterName := detectClusterType(tt.context)

			loader, err := r.getLoader(clusterType, clusterName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if loader.Name() != tt.wantLoader {
				t.Errorf("loader.Name() = %q, want %q", loader.Name(), tt.wantLoader)
			}
		})
	}
}

func TestDockerDesktopLoader_Load(t *testing.T) {
	logger := &util.MockLogger{}
	loader := newDockerDesktopLoader(logger)

	// Should always succeed (no-op)
	err := loader.Load(context.Background(), "myapp:kudev-abc123")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should log that image is available
	found := false
	for _, msg := range logger.Messages {
		if msg == "image available to Docker Desktop automatically" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log message about automatic availability")
	}
}

func TestKindLoader_ClusterName(t *testing.T) {
	logger := &util.MockLogger{}

	tests := []struct {
		input    string
		expected string
	}{
		{"dev", "dev"},
		{"test", "test"},
		{"", "kind"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			loader := newKindLoader(tt.input, logger)
			if loader.ClusterName() != tt.expected {
				t.Errorf("ClusterName() = %q, want %q", loader.ClusterName(), tt.expected)
			}
		})
	}
}

func TestRegistry_KubeContext(t *testing.T) {
	logger := &util.MockLogger{}
	r := NewRegistry("docker-desktop", logger)

	if r.KubeContext() != "docker-desktop" {
		t.Errorf("KubeContext() = %q, want %q", r.KubeContext(), "docker-desktop")
	}
}

func TestLoaderInterface(t *testing.T) {
	// Compile-time check that all loaders implement Loader
	var _ Loader = (*dockerDesktopLoader)(nil)
	var _ Loader = (*minikubeLoader)(nil)
	var _ Loader = (*kindLoader)(nil)
}

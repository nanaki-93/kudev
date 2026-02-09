package kubeconfig

import (
	"os"
	"strings"
	"testing"
)

func TestGetKubeconfigPath(t *testing.T) {
	setFakeKubeconfig(t)
	path, err := GetKubeconfigPath()
	if err != nil {
		t.Fatalf("Failed to get kubeconfig path: %v", err)
	}

	path2, err := getKubeconfigPath()
	if err != nil {
		t.Fatalf("Failed to get kubeconfig path: %v", err)
	}

	if path != path2 {
		t.Fatalf("Expected paths to be equal, got %q and %q", path, path2)
	}
}

func TestContextExists(t *testing.T) {
	tests := []struct {
		name     string
		context  string
		expected bool
	}{
		{name: "context exists", context: "docker-desktop", expected: true},
		{name: "context does not exist", context: "nonexistent", expected: false},
	}

	setFakeKubeconfig(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := ContextExists(tt.context)
			if err != nil {
				t.Fatalf("Failed to check context existence: %v", err)
			}
			if exists != tt.expected {
				t.Fatalf("Expected context existence to be %v, got %v", tt.expected, exists)
			}
		})
	}
}
func TestContextExists_ErrorLoading(t *testing.T) {
	t.Setenv("KUBECONFIG", "nonexistent")
	context, err := ContextExists("nonexistent")
	if context != false {
		t.Fatalf("Expected context to be false, got %v", context)
	}
	if err == nil {
		t.Fatalf("Expected error when loading kubeconfig, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load kubeconfig") {
		t.Fatalf("Expected error message to be 'failed to load kubeconfig', got %v", err)
	}
}

func setFakeKubeconfig(t *testing.T) {
	// Create a fake kubeconfig
	fakeKubeconfig := `
apiVersion: v1
kind: Config
current-context: docker-desktop
contexts:
- context:
    cluster: docker-desktop
    user: docker-desktop
  name: docker-desktop
- context:
    cluster: minikube
    user: minikube
  name: minikube
clusters:
- cluster:
    server: https://localhost:6443
  name: docker-desktop
- cluster:
    server: https://192.168.49.2:8443
  name: minikube
users:
- name: docker-desktop
- name: minikube
`
	// Write to a temp file
	tmpFile, err := os.CreateTemp(t.TempDir(), "kubeconfig-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if _, err := tmpFile.WriteString(fakeKubeconfig); err != nil {
		t.Fatalf("Failed to write fake kubeconfig: %v", err)
	}
	tmpFile.Close()

	// Point KUBECONFIG to the fake file
	t.Setenv("KUBECONFIG", tmpFile.Name())
}

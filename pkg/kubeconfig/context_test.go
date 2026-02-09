package kubeconfig

import (
	"os"
	"testing"
)

func TestGetKubeconfigPath(t *testing.T) {
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

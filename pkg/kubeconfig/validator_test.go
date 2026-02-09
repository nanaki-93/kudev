package kubeconfig

import (
	"os"
	"strings"
	"testing"
)

// TestMatches tests pattern matching logic.
func TestMatches(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		// Exact matches
		{"docker-desktop", "docker-desktop", true},
		{"docker-desktop", "minikube", false},

		// Wildcard matches
		{"kind-local", "kind-*", true},
		{"kind-staging", "kind-*", true},
		{"minikube", "kind-*", false},

		{"k3d-local", "k3d-*", true},
		{"k3d-test", "k3d-*", true},
		{"minikube", "k3d-*", false},

		{"my-local-cluster", "*-local*", true},
		{"local-dev", "*-local*", false},
		{"prod-cluster", "*-local*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matches(tt.name, tt.pattern)
			if got != tt.expected {
				t.Errorf("matches(%q, %q) = %v, want %v",
					tt.name, tt.pattern, got, tt.expected)
			}
		})
	}
}

// TestContextValidator_Validate tests validation logic.
func TestContextValidator_Validate(t *testing.T) {
	tests := []struct {
		name           string
		currentContext string
		forceContext   bool
		wantErr        bool
	}{
		// Safe contexts
		{
			name:           "docker-desktop is allowed",
			currentContext: "docker-desktop",
			forceContext:   false,
			wantErr:        false,
		},
		{
			name:           "minikube is allowed",
			currentContext: "minikube",
			forceContext:   false,
			wantErr:        false,
		},
		{
			name:           "kind-local is allowed",
			currentContext: "kind-local",
			forceContext:   false,
			wantErr:        false,
		},
		// Unsafe contexts
		{
			name:           "prod context blocked",
			currentContext: "prod-us-east-1",
			forceContext:   false,
			wantErr:        true,
		},
		{
			name:           "staging context blocked",
			currentContext: "staging-aws",
			forceContext:   false,
			wantErr:        true,
		},
		// Force override
		{
			name:           "prod with force-context allowed",
			currentContext: "prod-us-east-1",
			forceContext:   true,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := &ContextValidator{
				AllowedContexts: defaultAllowedContexts(),
				ForceContext:    tt.forceContext,
				CurrentContext:  tt.currentContext,
			}

			err := cv.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestContextValidator_ValidateContext(t *testing.T) {
	tests := []struct {
		name     string
		context  string
		expected bool
	}{
		{"docker-desktop is allowed", "docker-desktop", true},
		{"minikube is allowed", "minikube", true},
		{"kind-local is allowed", "kind-local", true},
		{"prod-us-east-1 is blocked", "prod-us-east-1", false},
		{"staging-aws is blocked", "staging-aws", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := &ContextValidator{
				AllowedContexts: defaultAllowedContexts(),
			}
			err := cv.ValidateContext(tt.context)
			if err != nil && tt.expected {
				t.Errorf("Expected no error for allowed context %s, got: %v", tt.context, err)
			}
			if err == nil && !tt.expected {
				t.Errorf("Expected error for blocked context %s, got nil", tt.context)
			}
		})
	}
}

// TestContextValidator_ErrorMessage tests error message quality.
func TestContextValidator_ErrorMessage(t *testing.T) {
	cv := &ContextValidator{
		AllowedContexts:      defaultAllowedContexts(),
		CurrentContext:       "prod-us-east-1",
		AllAvailableContexts: []string{"docker-desktop", "prod-us-east-1", "staging"},
	}

	err := cv.createBlockedError()
	if err == nil {
		t.Fatalf("Expected error for blocked context")
	}

	errStr := err.Error()

	// Should contain helpful information
	if !contains(errStr, "is not in whitelist") {
		t.Errorf("Error should explain whitelist, got: %s", errStr)
	}

	if !contains(errStr, "--force-context") {
		t.Errorf("Error should mention --force-context, got: %s", errStr)
	}

	if !contains(errStr, "kubectl config use-context") {
		t.Errorf("Error should suggest kubectl command, got: %s", errStr)
	}

	t.Logf("Error message:\n%s", errStr)
}

// TestDefaultAllowedContexts tests default whitelist.
func TestDefaultAllowedContexts(t *testing.T) {
	contexts := defaultAllowedContexts()

	// Should have sensible defaults
	if len(contexts) == 0 {
		t.Fatalf("Default contexts should not be empty")
	}

	// Should include well-known local clusters
	expectedPatterns := []string{
		"docker-desktop",
		"minikube",
		"kind-*",
	}

	for _, expected := range expectedPatterns {
		found := false
		for _, actual := range contexts {
			if actual == expected {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Default contexts missing %q", expected)
		}
	}
}

func TestNewContextValidator(t *testing.T) {
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

	allowedContexts := defaultAllowedContexts()
	availableContexts, _ := ListAvailableContexts()
	cv, err := NewContextValidator(false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(cv.AllowedContexts) != len(defaultAllowedContexts()) {
		t.Fatalf("Expected %d contexts, got %d", len(allowedContexts), len(cv.AllowedContexts))
	}
	if cv.ForceContext {
		t.Fatalf("Expected ForceContext to be false")
	}
	if len(cv.AllAvailableContexts) != 2 {
		t.Fatalf("Expected %d contexts, got %d", 2, len(cv.AllAvailableContexts))
	}
	if len(cv.AllAvailableContexts) != len(availableContexts) {
		t.Fatalf("Expected %d contexts, got %d", len(availableContexts), len(cv.AllAvailableContexts))
	}

	if cv.CurrentContext != "docker-desktop" {
		t.Fatalf("Expected current context 'docker-desktop', got %q", cv.CurrentContext)
	}
}

func TestWithAllowedContexts(t *testing.T) {
	cv := &ContextValidator{}
	cv.WithAllowedContexts([]string{"foo", "bar"})

	if len(cv.AllowedContexts) != 2 {
		t.Fatalf("Expected 2 allowed contexts, got %d", len(cv.AllowedContexts))
	}
	if cv.AllowedContexts[0] != "foo" {
		t.Fatalf("Expected first allowed context to be 'foo', got %q", cv.AllowedContexts[0])
	}
	if cv.AllowedContexts[1] != "bar" {
		t.Fatalf("Expected second allowed context to be 'bar', got %q", cv.AllowedContexts[1])
	}
}

func TestContextValidator_WithCurrentContext(t *testing.T) {
	cv := &ContextValidator{}
	cv.WithCurrentContext("foo")
	if cv.CurrentContext != "foo" {
		t.Fatalf("Expected current context to be 'foo', got %q", cv.CurrentContext)
	}
}

// ============================================================
// Helpers
// ============================================================

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

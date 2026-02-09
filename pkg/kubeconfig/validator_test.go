package kubeconfig

import (
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

// ============================================================
// Helpers
// ============================================================

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

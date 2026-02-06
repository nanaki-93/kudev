# Task 1.4: Implement Kubeconfig Reader & Context Validation

## Overview

This task implements **Kubernetes context safety checks** to prevent accidental deployments to production clusters. It must:
1. **Load kubeconfig** from standard locations
2. **Detect current context** 
3. **Validate context is safe** (whitelist or explicit override)
4. **Provide clear guidance** if context is blocked

**Effort**: ~2-3 hours  
**Complexity**: 🟡 Intermediate (K8s API usage)  
**Dependencies**: Task 1.2 (Validation)  
**Files to Create**:
- `pkg/kubeconfig/context.go` — Kubeconfig reading
- `pkg/kubeconfig/validator.go` — Context validation
- `pkg/kubeconfig/validator_test.go` — Tests

---

## The Problem Context Safety Solves

Without context validation:

```bash
$ kubectl config current-context
prod-us-east-1

$ kudev up  # Oops! Accidentally using prod context
# 🔥 Deployed broken code to production!

# With validation:
$ kudev up
ERROR: context 'prod-us-east-1' not in whitelist
Allowed: docker-desktop, minikube, kind-*
Use --force-context to override
```

---

## Kubeconfig Basics

K8s stores configuration in `~/.kube/config`:

```yaml
apiVersion: v1
kind: Config
clusters:
  - name: docker-desktop
    cluster:
      server: https://127.0.0.1:6443
      certificate-authority: /Users/user/.kube/...
  - name: kind-local
    cluster:
      server: https://127.0.0.1:33657
contexts:
  - name: docker-desktop
    context:
      cluster: docker-desktop
      user: docker-desktop
  - name: kind-local
    context:
      cluster: kind-local
      user: kind-local
current-context: docker-desktop  ← This one is active
users:
  - name: docker-desktop
    user:
      client-certificate: /Users/user/.kube/...
      client-key: /Users/user/.kube/...
```

**We only care about**:
- `clusters[*].name` — cluster identifier
- `contexts[*].name` — context identifier  
- `current-context` — currently active context

---

## Implementation: context.go

Create `pkg/kubeconfig/context.go`:

```go
package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
)

// KubeconfigContext represents information about the current K8s context.
type KubeconfigContext struct {
	// Name is the context name (e.g., "docker-desktop", "kind-local")
	Name string

	// ClusterName is the cluster this context points to
	ClusterName string

	// ClusterServer is the API server URL (e.g., "https://127.0.0.1:6443")
	ClusterServer string

	// UserName is the user identity in this context
	UserName string

	// Namespace is the default namespace for this context (if set)
	Namespace string
}

// LoadCurrentContext reads the current K8s context from kubeconfig.
//
// Process:
//   1. Locate kubeconfig file
//   2. Parse kubeconfig
//   3. Get current context name
//   4. Load context details
//   5. Return structured data
//
// Returns:
//   - KubeconfigContext with current context details
//   - Clear error if kubeconfig not found or invalid
func LoadCurrentContext() (*KubeconfigContext, error) {
	// Step 1: Find kubeconfig file
	kubeconfigPath, err := getKubeconfigPath()
	if err != nil {
		return nil, err
	}

	// Step 2: Load kubeconfig
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
	}

	// Step 3: Get current context name
	currentContextName := config.CurrentContext
	if currentContextName == "" {
		return nil, fmt.Errorf(
			"no current context set in kubeconfig (%s)\n\n"+
				"Set current context with: kubectl config use-context <context-name>\n"+
				"Available contexts: %v",
			kubeconfigPath,
			getAvailableContextNames(config),
		)
	}

	// Step 4: Load context details
	context, ok := config.Contexts[currentContextName]
	if !ok {
		return nil, fmt.Errorf(
			"current context %q not found in kubeconfig",
			currentContextName,
		)
	}

	// Step 5: Load cluster info (optional, can be missing)
	clusterName := context.Cluster
	clusterServer := ""

	if cluster, ok := config.Clusters[clusterName]; ok {
		clusterServer = cluster.Server
	}

	// Build result
	result := &KubeconfigContext{
		Name:          currentContextName,
		ClusterName:   clusterName,
		ClusterServer: clusterServer,
		UserName:      context.AuthInfo,
		Namespace:     context.Namespace,
	}

	return result, nil
}

// ListAvailableContexts returns all available contexts in kubeconfig.
func ListAvailableContexts() ([]string, error) {
	kubeconfigPath, err := getKubeconfigPath()
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return getAvailableContextNames(config), nil
}

// ContextExists checks if a context exists in kubeconfig.
func ContextExists(contextName string) (bool, error) {
	kubeconfigPath, err := getKubeconfigPath()
	if err != nil {
		return false, err
	}

	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return false, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	_, exists := config.Contexts[contextName]
	return exists, nil
}

// ============================================================
// Kubeconfig Discovery
// ============================================================

// getKubeconfigPath finds the kubeconfig file to use.
//
// Search order (matching kubectl):
//   1. $KUBECONFIG environment variable
//   2. $HOME/.kube/config (default location)
//   3. Return error if not found
//
// Note: $KUBECONFIG can be multiple paths separated by : (Unix) or ; (Windows)
// For now, we use the first one.
func getKubeconfigPath() (string, error) {
	// Step 1: Check environment variable
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		// Can be multiple paths - use first one that exists
		return kubeconfig, nil
	}

	// Step 2: Check default location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	defaultPath := filepath.Join(homeDir, ".kube", "config")

	// Check if it exists
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, nil
	}

	return "", fmt.Errorf(
		"kubeconfig not found\n\n"+
			"Kubeconfig locations checked:\n"+
			"  - $KUBECONFIG environment variable\n"+
			"  - %s (default)\n\n"+
			"Setup: mkdir -p ~/.kube && kubectl config view > ~/.kube/config",
		defaultPath,
	)
}

// getAvailableContextNames returns list of context names from kubeconfig.
func getAvailableContextNames(config *clientcmd.Config) []string {
	names := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		names = append(names, name)
	}
	return names
}

// ============================================================
// Utilities
// ============================================================

// GetKubeconfigPath returns the path to the kubeconfig file (for testing/debugging).
func GetKubeconfigPath() (string, error) {
	return getKubeconfigPath()
}
```

---

## Implementation: validator.go

Create `pkg/kubeconfig/validator.go`:

```go
package kubeconfig

import (
	"fmt"
	"regexp"
	"strings"
)

// ContextValidator validates if a K8s context is safe to deploy to.
//
// Safety mechanism:
//   - Whitelist of allowed contexts (docker-desktop, minikube, kind-*)
//   - Require explicit --force-context flag to use unlisted contexts
//   - Prevents accidental production deployments
type ContextValidator struct {
	// AllowedContexts is a list of allowed context patterns.
	// Can include wildcards: "kind-*", "local-*"
	AllowedContexts []string

	// ForceContext flag from --force-context (skips validation if true)
	ForceContext bool

	// CurrentContext is the currently active context in kubeconfig
	CurrentContext string

	// AllAvailableContexts is all contexts in kubeconfig (for error messages)
	AllAvailableContexts []string
}

// NewContextValidator creates a validator with default settings.
//
// Default allowed contexts (safe for development):
//   - "docker-desktop" (Docker Desktop K8s)
//   - "docker-for-desktop" (older Docker Desktop)
//   - "minikube" (Minikube)
//   - "kind-*" (kind clusters)
//   - "k3d-*" (k3d clusters)
//   - "*-local*" (any local variant)
func NewContextValidator(forceContext bool) (*ContextValidator, error) {
	current, err := LoadCurrentContext()
	if err != nil {
		return nil, err
	}

	available, _ := ListAvailableContexts()  // Error ignored for now

	return &ContextValidator{
		AllowedContexts: defaultAllowedContexts(),
		ForceContext:    forceContext,
		CurrentContext:  current.Name,
		AllAvailableContexts: available,
	}, nil
}

// Validate checks if the current context is allowed.
//
// Returns:
//   - nil if context is allowed
//   - Clear error message if context is blocked
//
// Check order:
//   1. If --force-context flag: allow but warn
//   2. If context in whitelist: allow
//   3. Otherwise: reject with helpful message
func (cv *ContextValidator) Validate() error {
	// Step 1: Check if matches whitelist
	if cv.isAllowed(cv.CurrentContext) {
		return nil  // All good
	}

	// Step 2: If not allowed but force flag set
	if cv.ForceContext {
		// In production, we'd warn here
		// For now, just allow
		return nil
	}

	// Step 3: Not allowed - provide helpful error
	return cv.createBlockedError()
}

// ValidateContext validates a specific context name (not necessarily current).
func (cv *ContextValidator) ValidateContext(contextName string) error {
	if cv.isAllowed(contextName) {
		return nil
	}

	if cv.ForceContext {
		return nil
	}

	return fmt.Errorf(
		"context %q not in whitelist\n\n"+
			"Allowed contexts: %v\n"+
			"Use --force-context to override",
		contextName,
		cv.AllowedContexts,
	)
}

// ============================================================
// Private Helpers
// ============================================================

// isAllowed checks if context name matches whitelist.
func (cv *ContextValidator) isAllowed(contextName string) bool {
	for _, pattern := range cv.AllowedContexts {
		if matches(contextName, pattern) {
			return true
		}
	}
	return false
}

// matches checks if name matches pattern (with wildcard support).
//
// Examples:
//   matches("docker-desktop", "docker-desktop") → true
//   matches("kind-local", "kind-*") → true
//   matches("prod-us-east-1", "prod-*") → false
func matches(name, pattern string) bool {
	// Exact match
	if name == pattern {
		return true
	}

	// Wildcard match
	if strings.Contains(pattern, "*") {
		// Convert pattern to regex
		// "kind-*" → "^kind-.*$"
		regexPattern := "^" + regexp.QuoteMeta(pattern)
		regexPattern = strings.ReplaceAll(regexPattern, `\*`, ".*")
		regexPattern += "$"

		if regex, err := regexp.Compile(regexPattern); err == nil {
			return regex.MatchString(name)
		}
	}

	return false
}

// createBlockedError creates a detailed error message for blocked context.
func (cv *ContextValidator) createBlockedError() error {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf(
		"Current context %q is not in the whitelist\n\n",
		cv.CurrentContext,
	))

	msg.WriteString("Allowed contexts (for safety):\n")
	for _, ctx := range cv.AllowedContexts {
		msg.WriteString(fmt.Sprintf("  - %s\n", ctx))
	}

	if len(cv.AllAvailableContexts) > 0 {
		msg.WriteString(fmt.Sprintf("\nAvailable contexts in kubeconfig:\n"))
		for _, ctx := range cv.AllAvailableContexts {
			marker := " "
			if ctx == cv.CurrentContext {
				marker = "*"  // Mark current
			}
			msg.WriteString(fmt.Sprintf("  %s %s\n", marker, ctx))
		}
	}

	msg.WriteString(fmt.Sprintf(
		"\nTo override and proceed at your own risk:\n"+
			"  kudev --force-context <command>\n\n"+
			"To change context:\n"+
			"  kubectl config use-context <context-name>\n",
	))

	return fmt.Errorf("%s", msg.String())
}

// ============================================================
// Default Configuration
// ============================================================

// defaultAllowedContexts returns the default whitelist of safe contexts.
//
// These are contexts used only for local development:
//   - docker-desktop: Docker Desktop K8s
//   - minikube: Minikube local cluster
//   - kind-*: Kind clusters
//   - k3d-*: K3d clusters
//   - *-local*: Any context with "local" in name
func defaultAllowedContexts() []string {
	return []string{
		"docker-desktop",
		"docker-for-desktop",
		"minikube",
		"kind-*",
		"k3d-*",
		"*-local*",
		"localhost",
		"127.0.0.1",
	}
}

// ============================================================
// Builder Pattern for Testing
// ============================================================

// WithAllowedContexts sets custom allowed contexts (for testing).
func (cv *ContextValidator) WithAllowedContexts(contexts []string) *ContextValidator {
	cv.AllowedContexts = contexts
	return cv
}

// WithCurrentContext sets current context (for testing).
func (cv *ContextValidator) WithCurrentContext(name string) *ContextValidator {
	cv.CurrentContext = name
	return cv
}
```

---

## Testing: validator_test.go

Create `pkg/kubeconfig/validator_test.go`:

```go
package kubeconfig

import (
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
		{"local-dev", "*-local*", true},
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
	if !contains(errStr, "not in the whitelist") {
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
	return contains(haystack, needle)
}
```

---

## Critical Design Decisions

### Decision 1: Whitelist vs Blacklist

**Question**: Should we block unknown contexts or allow them?

**Answer**: **Whitelist (block by default)**
- Safe by default
- Explicit opt-in with --force-context
- Matches security principle: deny-all, allow specific

❌ Blacklist:
```go
// Bad: blocks prod-* but allows prod-new-cluster
if strings.Contains(ctx, "prod") {
    block()
}
```

✅ Whitelist:
```go
// Good: only allow known safe contexts
if matches(ctx, allowedPatterns) {
    allow()
}
```

### Decision 2: Pattern Matching

**Question**: Should we support wildcards?

**Answer**: **Yes, with simple glob patterns**
- `kind-*` matches `kind-local`, `kind-staging`, `kind-test`
- `*-local*` matches `my-local-cluster`, `local-dev`
- Simple to understand, not complex regex

---

## Checklist for Task 1.4

- [ ] Create `pkg/kubeconfig/context.go`
- [ ] Create `pkg/kubeconfig/validator.go`
- [ ] Create `pkg/kubeconfig/validator_test.go`
- [ ] Implement `LoadCurrentContext()` 
- [ ] Implement `ListAvailableContexts()`
- [ ] Implement `ContextValidator`
- [ ] Implement pattern matching with wildcards
- [ ] Generate helpful error messages
- [ ] All tests pass
- [ ] Run: `go test ./pkg/kubeconfig -v`

---

## Integration with Other Components

```
Task 1.4 uses:
  - client-go (K8s standard library)
  - Kubeconfig loading

Task 1.4 is used by:
  - Task 1.5 (root.go PersistentPreRunE)
  - Task 1.6 (integration tests)
```

In `cmd/root.go` (Task 1.5):
```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
    // Load config (from Task 1.3)
    cfg, _ := config.LoadConfig(ctx, configPath)
    
    // Load and validate context (this task)
    validator, _ := kubeconfig.NewContextValidator(forceContext)
    if err := validator.Validate(); err != nil {
        return err  // Block unsafe deployments
    }
    
    return nil
}
```

---

## Next Steps

1. **Implement this task** ← You are here
2. **Task 1.5** → Use validator in CLI root command
3. **Task 1.6** → Integration and testing



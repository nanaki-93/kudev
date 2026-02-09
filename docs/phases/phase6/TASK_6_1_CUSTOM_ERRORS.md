# Task 6.1: Define Custom Error Types

## Overview

This task creates **domain-specific error types** with user-friendly messages and appropriate exit codes.

**Effort**: ~2-3 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: None  
**Files to Create**:
- `pkg/errors/errors.go` — Error type definitions
- `pkg/errors/messages.go` — User messages and suggestions
- `pkg/errors/errors_test.go` — Tests

---

## What You're Building

Custom errors that:
1. **Categorize** errors by domain (config, build, deploy)
2. **Provide** user-friendly messages
3. **Suggest** actions to fix the problem
4. **Return** consistent exit codes

---

## Complete Implementation

### Error Types

```go
// pkg/errors/errors.go

package errors

import (
    "fmt"
)

// KudevError is the interface for all kudev errors.
type KudevError interface {
    error
    
    // ExitCode returns the shell exit code.
    ExitCode() int
    
    // UserMessage returns a user-friendly message.
    UserMessage() string
    
    // SuggestedAction returns a helpful suggestion.
    SuggestedAction() string
}

// Exit codes
const (
    ExitGeneral    = 1  // General error
    ExitConfig     = 2  // Configuration error
    ExitKubeAuth   = 3  // Kubernetes authentication error
    ExitBuild      = 4  // Build error
    ExitDeploy     = 5  // Deployment error
    ExitWatch      = 6  // Watch error
)

// ConfigError represents configuration-related errors.
type ConfigError struct {
    Message    string
    Suggestion string
    Cause      error
}

func (e *ConfigError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

func (e *ConfigError) ExitCode() int         { return ExitConfig }
func (e *ConfigError) UserMessage() string   { return e.Message }
func (e *ConfigError) SuggestedAction() string { return e.Suggestion }
func (e *ConfigError) Unwrap() error         { return e.Cause }

// KubeAuthError represents Kubernetes authentication errors.
type KubeAuthError struct {
    Message    string
    Suggestion string
    Cause      error
}

func (e *KubeAuthError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

func (e *KubeAuthError) ExitCode() int         { return ExitKubeAuth }
func (e *KubeAuthError) UserMessage() string   { return e.Message }
func (e *KubeAuthError) SuggestedAction() string { return e.Suggestion }
func (e *KubeAuthError) Unwrap() error         { return e.Cause }

// BuildError represents image build errors.
type BuildError struct {
    Message    string
    Suggestion string
    Cause      error
}

func (e *BuildError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

func (e *BuildError) ExitCode() int         { return ExitBuild }
func (e *BuildError) UserMessage() string   { return e.Message }
func (e *BuildError) SuggestedAction() string { return e.Suggestion }
func (e *BuildError) Unwrap() error         { return e.Cause }

// DeployError represents Kubernetes deployment errors.
type DeployError struct {
    Message    string
    Suggestion string
    Cause      error
}

func (e *DeployError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

func (e *DeployError) ExitCode() int         { return ExitDeploy }
func (e *DeployError) UserMessage() string   { return e.Message }
func (e *DeployError) SuggestedAction() string { return e.Suggestion }
func (e *DeployError) Unwrap() error         { return e.Cause }

// WatchError represents file watching errors.
type WatchError struct {
    Message    string
    Suggestion string
    Cause      error
}

func (e *WatchError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

func (e *WatchError) ExitCode() int         { return ExitWatch }
func (e *WatchError) UserMessage() string   { return e.Message }
func (e *WatchError) SuggestedAction() string { return e.Suggestion }
func (e *WatchError) Unwrap() error         { return e.Cause }

// Ensure all types implement KudevError
var (
    _ KudevError = (*ConfigError)(nil)
    _ KudevError = (*KubeAuthError)(nil)
    _ KudevError = (*BuildError)(nil)
    _ KudevError = (*DeployError)(nil)
    _ KudevError = (*WatchError)(nil)
)
```

### Error Constructors

```go
// pkg/errors/messages.go

package errors

// Config errors

func ConfigNotFound(path string) *ConfigError {
    return &ConfigError{
        Message:    "Configuration file not found: " + path,
        Suggestion: "Run 'kudev init' to create a new configuration, or specify path with --config",
    }
}

func ConfigInvalid(reason string, cause error) *ConfigError {
    return &ConfigError{
        Message:    "Invalid configuration: " + reason,
        Suggestion: "Check your .kudev.yaml file for syntax errors",
        Cause:      cause,
    }
}

func ConfigMissingField(field string) *ConfigError {
    return &ConfigError{
        Message:    "Missing required field: " + field,
        Suggestion: "Add '" + field + "' to your .kudev.yaml configuration",
    }
}

// Kubernetes auth errors

func KubeconfigNotFound() *KubeAuthError {
    return &KubeAuthError{
        Message:    "Kubeconfig file not found",
        Suggestion: "Set KUBECONFIG environment variable or create ~/.kube/config",
    }
}

func KubeContextNotFound(context string) *KubeAuthError {
    return &KubeAuthError{
        Message:    "Kubernetes context not found: " + context,
        Suggestion: "Run 'kubectl config get-contexts' to see available contexts",
    }
}

func KubeContextNotAllowed(context string) *KubeAuthError {
    return &KubeAuthError{
        Message:    "Context '" + context + "' is not allowed for local development",
        Suggestion: "Use a local cluster like Docker Desktop, Minikube, or Kind",
    }
}

func KubeConnectionFailed(cause error) *KubeAuthError {
    return &KubeAuthError{
        Message:    "Failed to connect to Kubernetes cluster",
        Suggestion: "Ensure your cluster is running and kubectl is configured correctly",
        Cause:      cause,
    }
}

// Build errors

func DockerNotRunning(cause error) *BuildError {
    return &BuildError{
        Message:    "Docker daemon is not running",
        Suggestion: "Start Docker Desktop or run 'sudo systemctl start docker'",
        Cause:      cause,
    }
}

func DockerBuildFailed(cause error) *BuildError {
    return &BuildError{
        Message:    "Docker build failed",
        Suggestion: "Check the build output above for errors in your Dockerfile",
        Cause:      cause,
    }
}

func DockerfileNotFound(path string) *BuildError {
    return &BuildError{
        Message:    "Dockerfile not found: " + path,
        Suggestion: "Create a Dockerfile or specify the correct path in .kudev.yaml",
    }
}

func ImageLoadFailed(cluster string, cause error) *BuildError {
    return &BuildError{
        Message:    "Failed to load image to " + cluster + " cluster",
        Suggestion: "Ensure your cluster is running and accessible",
        Cause:      cause,
    }
}

// Deploy errors

func DeploymentFailed(cause error) *DeployError {
    return &DeployError{
        Message:    "Failed to deploy to Kubernetes",
        Suggestion: "Check that your cluster is running and you have permissions",
        Cause:      cause,
    }
}

func DeploymentNotFound(name, namespace string) *DeployError {
    return &DeployError{
        Message:    "Deployment not found: " + namespace + "/" + name,
        Suggestion: "Run 'kudev up' to create the deployment first",
    }
}

func NamespaceCreateFailed(namespace string, cause error) *DeployError {
    return &DeployError{
        Message:    "Failed to create namespace: " + namespace,
        Suggestion: "Check that you have permissions to create namespaces",
        Cause:      cause,
    }
}

func PortForwardFailed(port int32, cause error) *DeployError {
    return &DeployError{
        Message:    fmt.Sprintf("Port forwarding failed on port %d", port),
        Suggestion: fmt.Sprintf("Port %d may be in use. Try a different port with --local-port", port),
        Cause:      cause,
    }
}

// Watch errors

func WatcherFailed(cause error) *WatchError {
    return &WatchError{
        Message:    "File watcher failed",
        Suggestion: "You may have too many files. Try adding exclusions to .kudev.yaml",
        Cause:      cause,
    }
}
```

---

## Testing

```go
// pkg/errors/errors_test.go

package errors

import (
    "errors"
    "testing"
)

func TestConfigError(t *testing.T) {
    err := ConfigNotFound("/path/to/.kudev.yaml")
    
    if err.ExitCode() != ExitConfig {
        t.Errorf("ExitCode() = %d, want %d", err.ExitCode(), ExitConfig)
    }
    
    if err.UserMessage() == "" {
        t.Error("UserMessage() should not be empty")
    }
    
    if err.SuggestedAction() == "" {
        t.Error("SuggestedAction() should not be empty")
    }
}

func TestErrorUnwrap(t *testing.T) {
    cause := errors.New("original error")
    err := DockerBuildFailed(cause)
    
    if !errors.Is(err, cause) {
        t.Error("errors.Is should find the cause")
    }
    
    unwrapped := errors.Unwrap(err)
    if unwrapped != cause {
        t.Error("Unwrap should return the cause")
    }
}

func TestKudevErrorInterface(t *testing.T) {
    tests := []struct {
        name     string
        err      KudevError
        exitCode int
    }{
        {"ConfigError", ConfigNotFound("x"), ExitConfig},
        {"KubeAuthError", KubeconfigNotFound(), ExitKubeAuth},
        {"BuildError", DockerNotRunning(nil), ExitBuild},
        {"DeployError", DeploymentNotFound("x", "y"), ExitDeploy},
        {"WatchError", WatcherFailed(nil), ExitWatch},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.err.ExitCode() != tt.exitCode {
                t.Errorf("ExitCode() = %d, want %d", tt.err.ExitCode(), tt.exitCode)
            }
            
            if tt.err.UserMessage() == "" {
                t.Error("UserMessage() should not be empty")
            }
        })
    }
}
```

---

## Checklist for Task 6.1

- [ ] Create `pkg/errors/errors.go`
- [ ] Define `KudevError` interface
- [ ] Define exit code constants
- [ ] Implement `ConfigError` type
- [ ] Implement `KubeAuthError` type
- [ ] Implement `BuildError` type
- [ ] Implement `DeployError` type
- [ ] Implement `WatchError` type
- [ ] Add `Unwrap()` for error chaining
- [ ] Create `pkg/errors/messages.go`
- [ ] Add constructor functions for common errors
- [ ] Create `pkg/errors/errors_test.go`
- [ ] Test all error types
- [ ] Test error unwrapping
- [ ] Run `go test ./pkg/errors -v`

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 6.2** → Implement Error Interception

---

## References

- [Go Error Handling](https://go.dev/blog/go1.13-errors)
- [errors.Is and errors.As](https://pkg.go.dev/errors)


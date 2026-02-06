# Task 1.6: Implement Logging with Klog & Integration Testing

## Overview

This task implements **structured logging** using Klog (K8s standard) and establishes **integration testing patterns**. It covers:
1. **Klog setup** with verbosity levels
2. **Structured logging** throughout codebase
3. **Integration tests** that verify Phase 1 components work together
4. **Test coverage** reporting

**Effort**: ~2-3 hours  
**Complexity**: 🟡 Intermediate (Klog, integration testing)  
**Dependencies**: Task 1.1-1.5 (all Phase 1 components)  
**Files to Create**:
- `pkg/logging/logger.go` — Logging initialization
- `test/integration/phase1_test.go` — Integration tests
- `.github/workflows/test.yml` — CI/CD (optional)

---

## Logging Strategy

### Why Klog?

Klog is the logging library used by:
- Kubernetes (kubectl, API server)
- Client-go (official K8s Go client)
- Helm
- Other K8s tools

**Our Klog approach**:
- **Info logs**: Important events (deployment created, config loaded)
- **Error logs**: Actual errors with context
- **Debug logs (V(4))**: Detailed execution trace
- **Warnings**: Operational warnings

### Log Levels

```
Disabled (no logging)
    ↓
Error level
    ↓
Info level (default) ← Default level
    ↓
Debug level V(1)
    ↓
Verbose level V(2-4) ← Use with --debug flag
```

---

## Implementation: pkg/logging/logger.go

Create `pkg/logging/logger.go`:

```go
package logging

import (
	"flag"
	"fmt"

	"k8s.io/klog/v2"
)

// Init initializes the logger based on verbosity flags.
//
// Klog verbosity levels:
//   - V(0): Disabled (errors only)
//   - V(1-2): Debug information
//   - V(3-4): Very detailed tracing
//   - V(5+): Framework internals
//
// Our mapping:
//   - No flags: Info level (important events only)
//   - --debug: V(4) (all debug information)
//   - --debug --debug: V(6) (including framework internals)
func Init(debug bool) {
	// Set Klog output format
	klog.SetOutput(nil)  // Use default stderr
	klog.SetLogger(klog.NewKlogr())

	// Configure verbosity
	if debug {
		// Enable debug logs
		flag.Set("v", "4")
	} else {
		// Normal operation - less verbose
		flag.Set("v", "0")
	}

	// Parse flags (this applies the settings)
	flag.Parse()
}

// Get returns the configured logger instance.
//
// Usage:
//   logger := logging.Get()
//   logger.Info("deployment created", "name", deploymentName)
//   logger.Error(err, "failed to deploy")
func Get() klog.Logger {
	return klog.Background()
}

// Info logs an informational message.
//
// Used for:
//   - Important state changes (deployment created, config validated)
//   - User-facing operations (build started, deploy complete)
//   - Audit trail
//
// Example:
//   logger.Info("deployment created", "namespace", "default", "name", "myapp")
func Info(msg string, keysAndValues ...interface{}) {
	klog.Background().Info(msg, keysAndValues...)
}

// Error logs an error message with context.
//
// Used for:
//   - Actual errors (file not found, deploy failed)
//   - Operational issues
//   - Stack traces
//
// Example:
//   logger.Error(err, "failed to create deployment", "namespace", "default")
func Error(err error, msg string, keysAndValues ...interface{}) {
	klog.Background().Error(err, msg, keysAndValues...)
}

// Debug logs a debug message (only if --debug is set).
//
// Used for:
//   - Detailed operation tracing
//   - Variable state inspection
//   - Decision logic
//
// Example:
//   logger.V(4).Info("processing dockerfile", "path", dockerfilePath)
func Debug(msg string, keysAndValues ...interface{}) {
	klog.Background().V(4).Info(msg, keysAndValues...)
}

// Warn logs a warning message.
//
// Used for:
//   - Potentially unsafe conditions
//   - Unusual configurations
//   - Deprecation warnings
//
// Note: Klog doesn't have Warn level, use Info with special prefix
func Warn(msg string, keysAndValues ...interface{}) {
	klog.Background().Info("[WARN] "+msg, keysAndValues...)
}

// WithValues creates a logger with pre-set key-value pairs.
//
// Useful for operations that span multiple logs and share context.
//
// Example:
//   opLogger := logging.Get().WithValues("deployment", "myapp", "namespace", "default")
//   opLogger.Info("step 1: building image")
//   opLogger.Info("step 2: pushing image")
//   opLogger.Info("step 3: deploying")
func WithValues(keysAndValues ...interface{}) klog.Logger {
	return klog.Background().WithValues(keysAndValues...)
}

// ============================================================
// Logging Configuration
// ============================================================

// LogConfig holds logging configuration.
type LogConfig struct {
	// Level: 0=errors, 1=info, 4=debug, 6=verbose
	Level int

	// Pretty: pretty-print output (for human consumption)
	Pretty bool

	// Structured: output structured JSON (for log aggregation)
	Structured bool
}

// DefaultLogConfig returns default logging configuration.
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      0,  // Errors only
		Pretty:     true,
		Structured: false,
	}
}

// ============================================================
// Usage Examples for Code
// ============================================================

/*
// In main.go
func main() {
    logging.Init(debugFlag)
    logger := logging.Get()
    logger.Info("kudev started", "version", "v0.1.0")
}

// In config loader
func (fcl *FileConfigLoader) LoadFromPath(ctx context.Context, path string) (*DeploymentConfig, error) {
    logger := logging.Get()
    logger.V(4).Info("loading config file", "path", path)
    
    content, err := os.ReadFile(path)
    if err != nil {
        logger.Error(err, "failed to read config file", "path", path)
        return nil, err
    }
    
    // ...
    
    logger.Info("configuration loaded successfully", "app", cfg.Metadata.Name)
    return cfg, nil
}

// In validation
func (c *DeploymentConfig) Validate(ctx context.Context) error {
    logger := logging.WithValues("app", c.Metadata.Name)
    logger.V(4).Info("validating configuration")
    
    if c.Metadata.Name == "" {
        logger.Error(nil, "validation failed: missing app name")
        return fmt.Errorf("app name required")
    }
    
    logger.Info("configuration validation passed")
    return nil
}

// In context validation
func (cv *ContextValidator) Validate() error {
    logger := logging.Get()
    logger.V(4).Info("validating kubernetes context", "context", cv.CurrentContext)
    
    if !cv.isAllowed(cv.CurrentContext) {
        logger.Warn("context not in whitelist, use --force-context to override",
            "context", cv.CurrentContext,
            "allowed", cv.AllowedContexts,
        )
        return cv.createBlockedError()
    }
    
    logger.Info("kubernetes context validation passed", "context", cv.CurrentContext)
    return nil
}
*/
```

---

## Testing: test/integration/phase1_test.go

Create `test/integration/phase1_test.go`:

```go
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yourusername/kudev/pkg/config"
	"github.com/yourusername/kudev/pkg/kubeconfig"
	"github.com/yourusername/kudev/pkg/logging"
)

// TestPhase1_FullFlow tests the complete Phase 1 flow end-to-end.
//
// This integration test verifies:
//   1. Create config via FileConfigLoader
//   2. Load config from file
//   3. Validate config
//   4. Check context safety
//   5. Access config in "command"
func TestPhase1_FullFlow(t *testing.T) {
	logging.Init(false)

	// Setup: Create temporary project
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, ".kudev.yaml")
	dockerfilePath := filepath.Join(projectDir, "Dockerfile")

	// Create Dockerfile (required for validation)
	if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
		t.Fatalf("Failed to create Dockerfile: %v", err)
	}

	// Create initial config
	initialConfig := &config.DeploymentConfig{
		APIVersion: "kudev.io/v1alpha1",
		Kind:       "DeploymentConfig",
		Metadata: config.ConfigMetadata{
			Name: "test-app",
		},
		Spec: config.DeploymentSpec{
			ImageName:      "test-app",
			DockerfilePath: "./Dockerfile",
			Namespace:      "default",
			Replicas:       1,
			LocalPort:      8080,
			ServicePort:    8080,
			Env: []config.EnvVar{
				{Name: "LOG_LEVEL", Value: "debug"},
			},
		},
	}

	// Step 1: Save config
	loader := config.NewFileConfigLoader("", projectDir, projectDir)
	if err := loader.Save(context.Background(), initialConfig, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Step 2: Load config back
	loadedConfig, err := loader.LoadFromPath(context.Background(), configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded config matches original
	if loadedConfig.Metadata.Name != "test-app" {
		t.Errorf("Loaded config name mismatch: %s != test-app", loadedConfig.Metadata.Name)
	}

	if loadedConfig.Spec.Replicas != 1 {
		t.Errorf("Loaded config replicas mismatch: %d != 1", loadedConfig.Spec.Replicas)
	}

	// Step 3: Validate config
	if err := loadedConfig.Validate(context.Background()); err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	// Step 4: Validate with context (now that we know project root)
	if err := loadedConfig.ValidateWithContext(projectDir); err != nil {
		t.Fatalf("Context validation failed: %v", err)
	}

	// Step 5: Simulate CLI usage (accessing config)
	cfg := loadedConfig
	t.Logf("Successfully loaded and validated config for: %s", cfg.Metadata.Name)
	t.Logf("Namespace: %s, Replicas: %d", cfg.Spec.Namespace, cfg.Spec.Replicas)
}

// TestPhase1_ConfigDiscovery tests config file discovery in directory tree.
func TestPhase1_ConfigDiscovery(t *testing.T) {
	// Setup: Create directory structure
	projectDir := t.TempDir()
	sourceDir := filepath.Join(projectDir, "src")
	componentDir := filepath.Join(sourceDir, "components")

	if err := os.MkdirAll(componentDir, 0755); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	// Create .git marker (project root)
	gitDir := filepath.Join(projectDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git: %v", err)
	}

	// Create Dockerfile
	if err := os.WriteFile(filepath.Join(projectDir, "Dockerfile"), []byte("FROM alpine\n"), 0644); err != nil {
		t.Fatalf("Failed to create Dockerfile: %v", err)
	}

	// Create config at project root
	cfg := config.DefaultConfig("discovered-app")
	loader := config.NewFileConfigLoader("", projectDir, projectDir)
	if err := loader.Save(context.Background(), cfg, filepath.Join(projectDir, ".kudev.yaml")); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Simulate running kudev from deep in project
	deepLoader := config.NewFileConfigLoader("", projectDir, componentDir)
	found, err := deepLoader.discover()

	if err != nil {
		t.Fatalf("Config discovery failed: %v", err)
	}

	// Should find config at project root, not search forever
	expectedPath := filepath.Join(projectDir, ".kudev.yaml")
	if found != expectedPath {
		t.Errorf("Found config: %s, expected: %s", found, expectedPath)
	}

	t.Logf("Successfully discovered config at: %s", found)
}

// TestPhase1_DefaultsApplied tests that defaults are correctly applied.
func TestPhase1_DefaultsApplied(t *testing.T) {
	// Minimal config (mostly empty)
	cfg := &config.DeploymentConfig{
		Metadata: config.ConfigMetadata{
			Name: "test-app",
		},
		Spec: config.DeploymentSpec{
			ImageName:      "test-app",
			DockerfilePath: "./Dockerfile",
			// Rest empty - will be defaulted
		},
	}

	// Apply defaults
	config.ApplyDefaults(cfg)

	// Check defaults were applied
	tests := []struct {
		field    string
		got      interface{}
		expected interface{}
	}{
		{"Namespace", cfg.Spec.Namespace, "default"},
		{"Replicas", cfg.Spec.Replicas, int32(1)},
		{"LocalPort", cfg.Spec.LocalPort, int32(8080)},
		{"ServicePort", cfg.Spec.ServicePort, int32(8080)},
		{"APIVersion", cfg.APIVersion, "kudev.io/v1alpha1"},
		{"Kind", cfg.Kind, "DeploymentConfig"},
	}

	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("Default %s: got %v, expected %v", tt.field, tt.got, tt.expected)
		}
	}
}

// TestPhase1_ValidationErrorMessages tests that validation errors are helpful.
func TestPhase1_ValidationErrorMessages(t *testing.T) {
	// Create invalid config
	cfg := &config.DeploymentConfig{
		Metadata: config.ConfigMetadata{
			Name: "Invalid-Name",  // Uppercase not allowed
		},
		Spec: config.DeploymentSpec{
			ImageName:      "test-app",
			DockerfilePath: "./Dockerfile",
			Namespace:      "default",
			LocalPort:      70000,  // Invalid port
		},
	}

	// Validate (should fail)
	err := cfg.Validate(context.Background())
	if err == nil {
		t.Fatalf("Validation should fail for invalid config")
	}

	errStr := err.Error()

	// Check error message quality
	tests := []string{
		"uppercase",  // Hint about invalid name
		"metadata.name",  // Field path
		"localPort",  // Field name
	}

	for _, expected := range tests {
		if !contains(errStr, expected) {
			t.Errorf("Error message missing: %q", expected)
		}
	}

	t.Logf("Error message:\n%s", errStr)
}

// TestPhase1_ProjectRootDetection tests project root discovery heuristics.
func TestPhase1_ProjectRootDetection(t *testing.T) {
	tests := []struct {
		name    string
		markers []string
		found   bool
	}{
		{
			name:    "with .git",
			markers: []string{".git"},
			found:   true,
		},
		{
			name:    "with go.mod",
			markers: []string{"go.mod"},
			found:   true,
		},
		{
			name:    "with package.json",
			markers: []string{"package.json"},
			found:   true,
		},
		{
			name:    "no markers",
			markers: []string{},
			found:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create markers
			for _, marker := range tt.markers {
				if err := os.WriteFile(filepath.Join(tmpDir, marker), []byte{}, 0644); err != nil {
					t.Fatalf("Failed to create marker: %v", err)
				}
			}

			root, err := config.DiscoverProjectRoot(tmpDir)

			if tt.found {
				if err != nil {
					t.Errorf("DiscoverProjectRoot() error: %v", err)
				}
				if root != tmpDir {
					t.Errorf("Found root: %s, expected: %s", root, tmpDir)
				}
			} else {
				if err == nil {
					t.Errorf("DiscoverProjectRoot() should fail for no markers")
				}
			}
		})
	}
}

// TestPhase1_ContextValidation tests context safety validation.
func TestPhase1_ContextValidation(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping context validation test in CI environment")
	}

	// This test requires a real K8s context
	// Skip if kubeconfig not available
	_, err := config.DiscoverProjectRoot("")
	if err != nil {
		t.Skipf("Kubeconfig not available: %v", err)
	}

	// Create validator
	validator, err := kubeconfig.NewContextValidator(false)
	if err != nil {
		t.Skipf("Cannot load kubeconfig: %v", err)
	}

	// Validate (may pass or fail depending on current context)
	err = validator.Validate()
	t.Logf("Context validation result: %v", err)

	// With force-context, should always succeed
	validator.ForceContext = true
	if err := validator.Validate(); err != nil {
		t.Errorf("Validate with --force-context should succeed: %v", err)
	}
}

// ============================================================
// Test Helpers
// ============================================================

func contains(haystack, needle string) bool {
	return contains(haystack, needle)
}
```

---

## Test Coverage Reporting

Add to `Makefile`:

```makefile
.PHONY: test
test:
	go test ./... -v

.PHONY: coverage
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: coverage-check
coverage-check:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1
	@echo "Target: >80% coverage"
```

Usage:
```bash
make coverage  # Generate HTML coverage report
make coverage-check  # Show coverage percentage
```

---

## Critical Points

### 1. Klog vs Printf

❌ Wrong (uses printf):
```go
fmt.Printf("Deployment created: %s\n", name)
```

✅ Right (uses logger):
```go
logger := logging.Get()
logger.Info("deployment created", "name", name)
```

Benefits:
- Structured (easy to parse/aggregate)
- Verbosity levels (can suppress noise)
- K8s standard
- Easy to query

### 2. Context Propagation

Logging should maintain context across operations:

```go
// Bad: loses context
logger := logging.Get()
logger.Info("step 1")
logger.Info("step 2")  // Lost context of what we're doing

// Good: maintains context
opLogger := logging.WithValues("deployment", deploymentName, "namespace", ns)
opLogger.Info("step 1: building")
opLogger.Info("step 2: pushing")
opLogger.Info("step 3: deploying")  // Context preserved
```

### 3. Log Level Discipline

- **Error**: Actual errors
- **Info**: State changes, important events  
- **V(4)**: Detailed tracing, decision points

❌ Don't mix levels:
```go
logger.Info("failed to load file")  // Wrong: should be Error
logger.Error(nil, "file loaded")  // Wrong: should be Info
```

---

## Checklist for Task 1.6

- [ ] Create `pkg/logging/logger.go`
- [ ] Implement `Init()` function
- [ ] Implement `Get()` function  
- [ ] Helper functions: `Info()`, `Error()`, `Debug()`, `Warn()`
- [ ] Create `test/integration/phase1_test.go`
- [ ] Integration tests for full Phase 1 flow
- [ ] Config discovery tests
- [ ] Validation error message tests
- [ ] Project root detection tests
- [ ] All tests pass: `go test ./... -v`
- [ ] Coverage >80%: `go test ./... -cover`
- [ ] Create `Makefile` with test targets

---

## Running Tests

```bash
# Run all tests with output
go test ./... -v

# Run specific test
go test ./test/integration -v -run TestPhase1_FullFlow

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out
```

---

## Next Steps

1. **Implement this task** ← You are here
2. **Phase 1 Complete!** ✓
3. **Phase 2** → Docker image building (PHASE_2_IMAGE_PIPELINE.md)

---

## Summary of Phase 1

Phase 1 tasks created:

| Task | Purpose | Files | Status |
|------|---------|-------|--------|
| 1.1 | Config types | `types.go` | ✓ |
| 1.2 | Validation | `validation.go`, `errors.go` | ✓ |
| 1.3 | Config loading | `loader.go`, `defaults.go` | ✓ |
| 1.4 | Context safety | `kubeconfig/context.go`, `validator.go` | ✓ |
| 1.5 | CLI structure | `cmd/root.go`, `cmd/*.go` | ✓ |
| 1.6 | Logging & tests | `pkg/logging/logger.go`, `test/integration/*` | ✓ |

What we built:
- ✅ Configuration system (YAML → Go types)
- ✅ Validation with helpful error messages
- ✅ Config discovery in project hierarchy
- ✅ K8s context safety checks
- ✅ CLI with Cobra commands
- ✅ Structured logging with Klog
- ✅ Comprehensive integration tests

Next: Build image pipeline in Phase 2!



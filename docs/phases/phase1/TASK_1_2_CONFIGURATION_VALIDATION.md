# Task 1.2: Implement Configuration Validation

## Overview

This task implements **validation rules** that ensure `.kudev.yaml` configurations are correct before execution. Bad configurations should fail fast with **clear, actionable error messages**.

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate (regex, error handling, edge cases)  
**Dependencies**: Task 1.1 (Config Types)  
**Files to Create**: 
- `pkg/config/validation.go` — Validation logic
- `pkg/config/errors.go` — Custom error types
- `pkg/config/validation_test.go` — Tests

---

## The Problem Validation Solves

Without validation, users get cryptic errors much later:
```bash
$ kudev up
# 30 seconds later...
ERROR: building Docker image: error parsing Dockerfile...
# Which line in the Dockerfile? Hard to trace back to config

# With validation:
$ kudev up
ERROR: spec.dockerfilePath "./Dockerfile" does not exist
# Clear, immediate feedback!
```

---

## Validation Strategy

### Multi-Layer Validation

**Layer 1: Type Validation** (Go compiler, already done)
- YAML parser ensures correct types
- Ports are int32 (not strings)
- Replicas is positive

**Layer 2: Required Fields** (this task)
- Check mandatory fields exist
- e.g., `metadata.name` cannot be empty

**Layer 3: Format Validation** (this task)
- DNS-1123 names (lowercase, hyphens)
- Valid port ranges (1-65535)
- Valid paths (file exists)

**Layer 4: Context Validation** (Task 1.4)
- K8s context exists and is whitelisted
- Kubeconfig is valid

**Layer 5: File System Validation** (this task, extended)
- Dockerfile exists
- Project root discoverable

### Fail-Fast Philosophy

- Validate **immediately** after load
- Return **all errors** at once (not just first)
- Include **examples** in error messages
- Suggest **fixes** when possible

---

## Implementation: validation.go

Create `pkg/config/validation.go`:

```go
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Validate performs comprehensive validation of the configuration.
//
// Returns a ValidationError containing all issues found (not just the first).
// This allows users to fix multiple problems at once rather than iteratively.
//
// Validation layers:
//   1. Required fields: name, imageName, dockerfilePath, namespace
//   2. Type constraints: replicas ≥ 1, ports 1-65535
//   3. Format: DNS-1123 names, valid port ranges
//   4. File system: dockerfile exists (if path is absolute)
//   5. Uniqueness: no duplicate env var names
//
// Example:
//   cfg := &DeploymentConfig{...}
//   if err := cfg.Validate(); err != nil {
//       // err is ValidationError with details
//       if ve, ok := err.(*ValidationError); ok {
//           for _, detail := range ve.Details {
//               fmt.Println(detail)
//           }
//       }
//   }
func (c *DeploymentConfig) Validate(ctx context.Context) error {
	var errs ValidationError

	// Layer 1: Root object validation
	if c == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Validate APIVersion and Kind
	if c.APIVersion == "" {
		errs.Add("apiVersion is required (should be: kudev.io/v1alpha1)")
	} else if c.APIVersion != "kudev.io/v1alpha1" {
		errs.Add(fmt.Sprintf(
			"apiVersion %q not supported (use: kudev.io/v1alpha1)",
			c.APIVersion,
		))
	}

	if c.Kind == "" {
		errs.Add("kind is required (should be: DeploymentConfig)")
	} else if c.Kind != "DeploymentConfig" {
		errs.Add(fmt.Sprintf(
			"kind %q not supported (use: DeploymentConfig)",
			c.Kind,
		))
	}

	// Layer 2: Metadata validation
	errs.Merge(c.validateMetadata())

	// Layer 3: Spec validation
	errs.Merge(c.validateSpec(ctx))

	if errs.HasErrors() {
		return &errs
	}

	return nil
}

// validateMetadata validates the Metadata object.
func (c *DeploymentConfig) validateMetadata() ValidationError {
	var errs ValidationError

	if c.Metadata.Name == "" {
		errs.Add("metadata.name is required and cannot be empty")
		errs.AddExample("metadata:\n  name: my-app")
		return errs  // Early return - can't validate name format if empty
	}

	// Validate name format (DNS-1123 compliant)
	if err := validateDNSName(c.Metadata.Name); err != nil {
		errs.Add(fmt.Sprintf("metadata.name: %v", err))
		errs.AddExample("metadata:\n  name: my-app  # lowercase, hyphens, alphanumeric")
	}

	return errs
}

// validateSpec validates the Spec object.
//
// ctx is passed for future cancellation support (e.g., if file system I/O)
func (c *DeploymentConfig) validateSpec(ctx context.Context) ValidationError {
	var errs ValidationError

	spec := c.Spec

	// === Required Fields ===

	if spec.ImageName == "" {
		errs.Add("spec.imageName is required")
		errs.AddExample("spec:\n  imageName: my-app")
		// Don't return early - validate other fields too
	}

	if spec.DockerfilePath == "" {
		errs.Add("spec.dockerfilePath is required")
		errs.AddExample("spec:\n  dockerfilePath: ./Dockerfile")
	} else {
		// Validate dockerfile path (only if not empty)
		if err := validateDockerfilePath(spec.DockerfilePath); err != nil {
			errs.Add(fmt.Sprintf("spec.dockerfilePath: %v", err))
		}
	}

	if spec.Namespace == "" {
		errs.Add("spec.namespace is required")
		errs.AddExample("spec:\n  namespace: default")
		// Don't return early - let user see all problems
	} else {
		if err := validateDNSName(spec.Namespace); err != nil {
			errs.Add(fmt.Sprintf("spec.namespace: %v", err))
			errs.AddExample("spec:\n  namespace: default  # or: dev, prod-staging")
		}
	}

	// === Numeric Constraints ===

	if spec.Replicas < 1 {
		errs.Add(fmt.Sprintf(
			"spec.replicas must be at least 1, got %d",
			spec.Replicas,
		))
		errs.AddExample("spec:\n  replicas: 1")
	}

	if spec.Replicas > 100 {
		// Warning, not error (but we're only doing errors for now)
		// Phase 4 can add warnings system
	}

	// === Port Validation ===

	if err := validatePort("spec.localPort", spec.LocalPort); err != nil {
		errs.Add(err.Error())
		errs.AddExample("spec:\n  localPort: 8080  # 1-65535")
	}

	if err := validatePort("spec.servicePort", spec.ServicePort); err != nil {
		errs.Add(err.Error())
		errs.AddExample("spec:\n  servicePort: 8080  # 1-65535")
	}

	// === Environment Variables ===

	if err := validateEnv(spec.Env); err != nil {
		errs.Merge(err)
	}

	// === Optional Fields ===

	if spec.ImageName != "" {
		if err := validateImageName(spec.ImageName); err != nil {
			errs.Add(fmt.Sprintf("spec.imageName: %v", err))
			errs.AddExample("spec:\n  imageName: my-app  # lowercase, hyphens")
		}
	}

	if spec.KubeContext != "" {
		// Note: Actual context validation happens in Task 1.4
		// Here we just check format
		if err := validateKubeContextName(spec.KubeContext); err != nil {
			errs.Add(fmt.Sprintf("spec.kubeContext: %v", err))
		}
	}

	if len(spec.BuildContextExclusions) > 0 {
		if err := validateBuildContextExclusions(spec.BuildContextExclusions); err != nil {
			errs.Merge(err)
		}
	}

	return errs
}

// ============================================================
// Validation Helper Functions
// ============================================================

// validateDNSName validates a name is DNS-1123 compliant.
//
// Rules (RFC 1123):
//   - Must contain only lowercase alphanumeric characters and hyphens
//   - Must start with alphanumeric character
//   - Must end with alphanumeric character
//   - Length: 3-63 characters
//   - Hyphens cannot be consecutive
//
// Examples:
//   Valid: "my-app", "api", "frontend-v2"
//   Invalid: "MyApp" (uppercase), "my_app" (underscore), "-app" (leading hyphen)
func validateDNSName(name string) error {
	// Pattern: starts with letter/digit, followed by any letter/digit/hyphen,
	// ends with letter/digit
	pattern := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	if !regexp.MustCompile(pattern).MatchString(name) {
		return fmt.Errorf(
			"must be DNS-1123 compliant (lowercase alphanumeric and hyphens only, "+
				"cannot start/end with hyphen). Got: %q",
			name,
		)
	}

	if len(name) < 3 {
		return fmt.Errorf("must be at least 3 characters, got %d", len(name))
	}

	if len(name) > 63 {
		return fmt.Errorf("must be at most 63 characters, got %d", len(name))
	}

	return nil
}

// validateDockerfilePath validates the Dockerfile path is reasonable.
//
// Checks:
//   - Not empty (checked before calling this)
//   - Doesn't end with .git or .kudev.yaml
//   - If absolute, file exists
//   - Name looks like a Dockerfile
func validateDockerfilePath(path string) error {
	if path == "" {
		return errors.New("dockerfilePath cannot be empty")
	}

	// Normalize path
	path = filepath.Clean(path)

	// Reject obviously wrong paths
	if strings.HasSuffix(path, ".git") {
		return fmt.Errorf("Dockerfile path cannot be .git")
	}

	if strings.HasSuffix(path, ".kudev.yaml") {
		return fmt.Errorf("Dockerfile path cannot be .kudev.yaml")
	}

	// Check path looks like a Dockerfile
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "Dockerfile") && !strings.HasPrefix(base, "dockerfile") {
		// Warning: not error, but suspicious
		// Examples: "build.docker" is suspicious, "Dockerfile.prod" is OK
		if !strings.Contains(base, "docker") && !strings.Contains(base, "Docker") {
			return fmt.Errorf(
				"expected filename to contain 'Dockerfile', got %q (examples: Dockerfile, Dockerfile.dev, docker/Dockerfile)",
				base,
			)
		}
	}

	// If absolute path, check file exists
	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s", path)
			}
			return fmt.Errorf("cannot access file: %v", err)
		}
	}

	// If relative path, we can't validate without knowing project root
	// That happens in loader.go with full context

	return nil
}

// validatePort validates a port number.
//
// Valid range: 1-65535
// Note: Ports 1-1024 require elevated privileges
func validatePort(fieldName string, port int32) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf(
			"%s must be between 1 and 65535, got %d",
			fieldName, port,
		)
	}

	if port < 1024 {
		// Warning, not error
		// Could add warning system in future
	}

	return nil
}

// validateImageName validates Docker image name.
//
// Docker image names rules:
//   - Lowercase letters, digits, underscore, period or dash
//   - Cannot start with period or dash
//   - May contain lowercase 'a'–'z' and digits '0'–'9'
//   - May contain lowercase letters, digits, underscores, periods and dashes
//   - Must start with a letter or digit
//   - May not end with a period or dash
//   - May contain a maximum of 128 characters
//
// We use a simpler subset for clarity:
//   - Lowercase alphanumeric and hyphens only
func validateImageName(name string) error {
	// Docker allows more characters, but we restrict for consistency with DNS names
	pattern := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	if !regexp.MustCompile(pattern).MatchString(name) {
		return fmt.Errorf(
			"must be lowercase alphanumeric and hyphens only. Got: %q",
			name,
		)
	}

	if len(name) > 128 {
		return fmt.Errorf("must be at most 128 characters, got %d", len(name))
	}

	return nil
}

// validateEnv validates environment variables.
func validateEnv(vars []EnvVar) ValidationError {
	var errs ValidationError

	if len(vars) == 0 {
		return errs  // Empty is OK
	}

	seenNames := make(map[string]bool)

	for i, v := range vars {
		if v.Name == "" {
			errs.Add(fmt.Sprintf("env[%d].name is required", i))
			errs.AddExample("env:\n  - name: LOG_LEVEL\n    value: info")
			continue
		}

		// Check format
		if err := validateEnvVarName(v.Name); err != nil {
			errs.Add(fmt.Sprintf("env[%d].name %q: %v", i, v.Name, err))
		}

		// Check for duplicates
		if seenNames[v.Name] {
			errs.Add(fmt.Sprintf(
				"env[%d].name %q is duplicate (first occurrence: env[?].name %q)",
				i, v.Name, v.Name,
			))
		}
		seenNames[v.Name] = true

		// Value can be empty (some env vars are "presence markers")
	}

	return errs
}

// validateEnvVarName validates environment variable name format.
//
// Valid shell variable names:
//   - Only uppercase letters, digits, and underscores
//   - Must start with letter or underscore
//   - Cannot contain hyphens or special characters
//
// Examples:
//   Valid: "LOG_LEVEL", "DATABASE_URL", "_INTERNAL"
//   Invalid: "log-level" (lowercase and hyphens), "123VAR" (starts with digit)
func validateEnvVarName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}

	pattern := `^[A-Z_][A-Z0-9_]*$`
	if !regexp.MustCompile(pattern).MatchString(name) {
		return fmt.Errorf(
			"invalid format. Environment variable names must be UPPERCASE_WITH_UNDERSCORES. "+
				"Got: %q (examples: LOG_LEVEL, DATABASE_URL, DEBUG_MODE)",
			name,
		)
	}

	return nil
}

// validateKubeContextName validates kubeContext name format.
//
// This is a simple format check. The actual context existence
// is checked in Task 1.4.
func validateKubeContextName(name string) error {
	if name == "" {
		return errors.New("kubeContext cannot be empty if specified")
	}

	// K8s context names are fairly flexible
	// Allow alphanumeric, dots, hyphens, underscores, slashes
	pattern := `^[a-zA-Z0-9._\-/]+$`
	if !regexp.MustCompile(pattern).MatchString(name) {
		return fmt.Errorf(
			"invalid context name format: %q (examples: docker-desktop, kind-dev, my.context)",
			name,
		)
	}

	return nil
}

// validateBuildContextExclusions validates exclusion patterns.
func validateBuildContextExclusions(exclusions []string) ValidationError {
	var errs ValidationError

	for i, exc := range exclusions {
		if exc == "" {
			errs.Add(fmt.Sprintf("buildContextExclusions[%d] is empty", i))
			continue
		}

		if strings.HasPrefix(exc, "/") {
			errs.Add(fmt.Sprintf(
				"buildContextExclusions[%d] should be relative path, "+
					"not absolute: %q",
				i, exc,
			))
		}

		if strings.Contains(exc, "\\") {
			errs.Add(fmt.Sprintf(
				"buildContextExclusions[%d] should use forward slashes, "+
					"not backslashes: %q (use '%s')",
				i, exc, strings.ReplaceAll(exc, "\\", "/"),
			))
		}
	}

	return errs
}

// ============================================================
// Helpers for optional validation context
// ============================================================

// WithProjectRoot validates paths relative to project root.
//
// This is called by the loader with full filesystem context.
func (c *DeploymentConfig) ValidateWithContext(projectRoot string) error {
	// First validate configuration itself
	if err := c.Validate(context.Background()); err != nil {
		return err
	}

	var errs ValidationError

	// Now validate relative paths exist
	dockerfilePath := c.Spec.DockerfilePath

	// If relative, resolve from project root
	if !filepath.IsAbs(dockerfilePath) {
		dockerfilePath = filepath.Join(projectRoot, dockerfilePath)
	}

	if _, err := os.Stat(dockerfilePath); err != nil {
		errs.Add(fmt.Sprintf(
			"spec.dockerfilePath %q not found at %s",
			c.Spec.DockerfilePath,
			dockerfilePath,
		))
	}

	if errs.HasErrors() {
		return &errs
	}

	return nil
}
```

### Add context import at top of file:

```go
import (
	"context"
	// ... other imports
)
```

---

## Implementation: errors.go

Create `pkg/config/errors.go`:

```go
package config

import (
	"fmt"
	"strings"
)

// ValidationError collects multiple validation issues.
//
// Instead of failing on first error, we collect all errors
// so the user can fix them all at once.
//
// Example:
//   err := cfg.Validate()
//   if err != nil {
//       fmt.Println(err)  // Pretty-printed errors
//       if ve, ok := err.(*ValidationError); ok {
//           // Access individual details
//           for _, detail := range ve.Details {
//               fmt.Println(detail)
//           }
//       }
//   }
type ValidationError struct {
	// Details contains individual error messages
	Details []string

	// Examples contains suggested fixes (parallel to Details)
	Examples []string
}

// Add adds a validation error message.
func (ve *ValidationError) Add(msg string) {
	ve.Details = append(ve.Details, msg)
	// Keep examples in sync (empty string if no example)
	if len(ve.Examples) < len(ve.Details) {
		ve.Examples = append(ve.Examples, "")
	}
}

// AddExample adds an example for the last error added.
//
// Usage:
//   ve.Add("metadata.name is required")
//   ve.AddExample("metadata:\n  name: my-app")
func (ve *ValidationError) AddExample(example string) {
	if len(ve.Details) > 0 {
		// Replace last example
		if len(ve.Examples) < len(ve.Details) {
			ve.Examples = append(ve.Examples, "")
		}
		ve.Examples[len(ve.Examples)-1] = example
	}
}

// Merge combines another ValidationError into this one.
func (ve *ValidationError) Merge(other ValidationError) {
	ve.Details = append(ve.Details, other.Details...)
	ve.Examples = append(ve.Examples, other.Examples...)
}

// HasErrors returns true if any validation errors exist.
func (ve *ValidationError) HasErrors() bool {
	return len(ve.Details) > 0
}

// Error implements error interface.
//
// Produces formatted output like:
//   Configuration validation failed (3 errors):
//     1. metadata.name is required
//        Example:
//          metadata:
//            name: my-app
//
//     2. spec.dockerfilePath "./nonexistent" does not exist
//
//     3. env[0].name "log_level" is invalid
//        (Environment variables must be UPPERCASE_WITH_UNDERSCORES)
func (ve *ValidationError) Error() string {
	if len(ve.Details) == 0 {
		return "no validation errors"
	}

	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf(
		"Configuration validation failed (%d error%s):\n",
		len(ve.Details),
		pluralize(len(ve.Details)),
	))

	// Each error
	for i, detail := range ve.Details {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, detail))

		// Example if present
		if len(ve.Examples) > i && ve.Examples[i] != "" {
			example := ve.Examples[i]

			// Indent example lines
			indentedExample := indentLines(example, "     ")

			sb.WriteString(fmt.Sprintf("     Example:\n%s\n", indentedExample))
		}

		// Blank line between errors (except last)
		if i < len(ve.Details)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// Helper functions

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func indentLines(text string, indent string) string {
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}

// FieldError represents a field-specific validation error.
//
// Provides structured access to field path and error.
type FieldError struct {
	Field   string  // e.g., "spec.dockerfilePath"
	Message string  // e.g., "file does not exist"
	Example string  // e.g., "./Dockerfile"
}

func (fe *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", fe.Field, fe.Message)
}
```

---

## Testing: validation_test.go

Create `pkg/config/validation_test.go`:

```go
package config

import (
	"context"
	"testing"
)

// TestValidate_Valid tests validation of correct configurations.
func TestValidate_Valid(t *testing.T) {
	tests := []struct {
		name string
		cfg  *DeploymentConfig
	}{
		{
			name: "minimal valid config",
			cfg: &DeploymentConfig{
				APIVersion: "kudev.io/v1alpha1",
				Kind:       "DeploymentConfig",
				Metadata: ConfigMetadata{
					Name: "myapp",
				},
				Spec: DeploymentSpec{
					ImageName:      "myapp",
					DockerfilePath: "./Dockerfile",
					Namespace:      "default",
					Replicas:       1,
					LocalPort:      8080,
					ServicePort:    8080,
				},
			},
		},
		{
			name: "with environment variables",
			cfg: &DeploymentConfig{
				APIVersion: "kudev.io/v1alpha1",
				Kind:       "DeploymentConfig",
				Metadata: ConfigMetadata{
					Name: "myapp",
				},
				Spec: DeploymentSpec{
					ImageName:      "myapp",
					DockerfilePath: "./Dockerfile",
					Namespace:      "default",
					Replicas:       1,
					LocalPort:      8080,
					ServicePort:    8080,
					Env: []EnvVar{
						{Name: "LOG_LEVEL", Value: "info"},
						{Name: "DEBUG", Value: "false"},
					},
				},
			},
		},
		{
			name: "with kubeContext",
			cfg: &DeploymentConfig{
				APIVersion: "kudev.io/v1alpha1",
				Kind:       "DeploymentConfig",
				Metadata: ConfigMetadata{
					Name: "myapp",
				},
				Spec: DeploymentSpec{
					ImageName:      "myapp",
					DockerfilePath: "./Dockerfile",
					Namespace:      "default",
					Replicas:       1,
					LocalPort:      8080,
					ServicePort:    8080,
					KubeContext:    "docker-desktop",
				},
			},
		},
		{
			name: "multiple replicas",
			cfg: &DeploymentConfig{
				APIVersion: "kudev.io/v1alpha1",
				Kind:       "DeploymentConfig",
				Metadata: ConfigMetadata{
					Name: "frontend",
				},
				Spec: DeploymentSpec{
					ImageName:      "frontend",
					DockerfilePath: "./Dockerfile",
					Namespace:      "production",
					Replicas:       5,
					LocalPort:      3000,
					ServicePort:    3000,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(context.Background())
			if err != nil {
				t.Fatalf("Validate() got error = %v, want nil", err)
			}
		})
	}
}

// TestValidate_RequiredFields tests validation of required fields.
func TestValidate_RequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *DeploymentConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing metadata.name",
			cfg: &DeploymentConfig{
				APIVersion: "kudev.io/v1alpha1",
				Kind:       "DeploymentConfig",
				Metadata:   ConfigMetadata{}, // Missing name
				Spec: DeploymentSpec{
					ImageName:      "app",
					DockerfilePath: "./Dockerfile",
					Namespace:      "default",
					Replicas:       1,
					LocalPort:      8080,
					ServicePort:    8080,
				},
			},
			expectError: true,
			errorMsg:    "metadata.name is required",
		},
		{
			name: "missing spec.imageName",
			cfg: &DeploymentConfig{
				APIVersion: "kudev.io/v1alpha1",
				Kind:       "DeploymentConfig",
				Metadata: ConfigMetadata{
					Name: "app",
				},
				Spec: DeploymentSpec{
					// Missing ImageName
					DockerfilePath: "./Dockerfile",
					Namespace:      "default",
					Replicas:       1,
					LocalPort:      8080,
					ServicePort:    8080,
				},
			},
			expectError: true,
			errorMsg:    "spec.imageName is required",
		},
		{
			name: "missing spec.dockerfilePath",
			cfg: &DeploymentConfig{
				APIVersion: "kudev.io/v1alpha1",
				Kind:       "DeploymentConfig",
				Metadata: ConfigMetadata{
					Name: "app",
				},
				Spec: DeploymentSpec{
					ImageName:  "app",
					// Missing DockerfilePath
					Namespace:   "default",
					Replicas:    1,
					LocalPort:   8080,
					ServicePort: 8080,
				},
			},
			expectError: true,
			errorMsg:    "spec.dockerfilePath is required",
		},
		{
			name: "missing spec.namespace",
			cfg: &DeploymentConfig{
				APIVersion: "kudev.io/v1alpha1",
				Kind:       "DeploymentConfig",
				Metadata: ConfigMetadata{
					Name: "app",
				},
				Spec: DeploymentSpec{
					ImageName:      "app",
					DockerfilePath: "./Dockerfile",
					// Missing Namespace
					Replicas:    1,
					LocalPort:   8080,
					ServicePort: 8080,
				},
			},
			expectError: true,
			errorMsg:    "spec.namespace is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(context.Background())

			if (err != nil) != tt.expectError {
				t.Fatalf("Validate() got error = %v, expectError = %v", err, tt.expectError)
			}

			if err != nil && tt.errorMsg != "" {
				if !stringContains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message %q does not contain %q", err.Error(), tt.errorMsg)
				}
			}
		})
	}
}

// TestValidate_DNS1123 tests DNS-1123 name validation.
func TestValidate_DNS1123(t *testing.T) {
	tests := []struct {
		name        string
		fieldValue  string
		expectError bool
	}{
		// Valid names
		{"valid: simple", "app", false},
		{"valid: with hyphens", "my-app", false},
		{"valid: with numbers", "app1", false},
		{"valid: long", "my-production-database-app", false},

		// Invalid names
		{"invalid: uppercase", "MyApp", true},
		{"invalid: underscore", "my_app", true},
		{"invalid: dot", "my.app", true},
		{"invalid: leading hyphen", "-myapp", true},
		{"invalid: trailing hyphen", "myapp-", true},
		{"invalid: too short", "my", true},
		{"invalid: too long", "a" + strings.Repeat("a", 64), true},
		{"invalid: space", "my app", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDNSName(tt.fieldValue)

			if (err != nil) != tt.expectError {
				t.Fatalf("validateDNSName(%q) got error = %v, expectError = %v",
					tt.fieldValue, err, tt.expectError)
			}
		})
	}
}

// TestValidate_Ports tests port validation.
func TestValidate_Ports(t *testing.T) {
	tests := []struct {
		name        string
		port        int32
		expectError bool
	}{
		{"valid: common http", 8080, false},
		{"valid: common https", 8443, false},
		{"valid: node", 3000, false},
		{"valid: min", 1, false},
		{"valid: max", 65535, false},

		{"invalid: zero", 0, true},
		{"invalid: negative", -1, true},
		{"invalid: too high", 70000, true},
		{"invalid: way too high", 999999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort("testPort", tt.port)

			if (err != nil) != tt.expectError {
				t.Fatalf("validatePort(%d) got error = %v, expectError = %v",
					tt.port, err, tt.expectError)
			}
		})
	}
}

// TestValidate_EnvVars tests environment variable validation.
func TestValidate_EnvVars(t *testing.T) {
	tests := []struct {
		name        string
		vars        []EnvVar
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty list",
			vars:        []EnvVar{},
			expectError: false,
		},
		{
			name: "valid single var",
			vars: []EnvVar{
				{Name: "LOG_LEVEL", Value: "info"},
			},
			expectError: false,
		},
		{
			name: "valid multiple vars",
			vars: []EnvVar{
				{Name: "LOG_LEVEL", Value: "info"},
				{Name: "DEBUG", Value: "false"},
				{Name: "DATABASE_URL", Value: "postgres://localhost"},
			},
			expectError: false,
		},
		{
			name: "missing name",
			vars: []EnvVar{
				{Name: "", Value: "info"},
			},
			expectError: true,
			errorMsg:    "is required",
		},
		{
			name: "invalid name: lowercase",
			vars: []EnvVar{
				{Name: "log_level", Value: "info"},
			},
			expectError: true,
			errorMsg:    "must be UPPERCASE_WITH_UNDERSCORES",
		},
		{
			name: "invalid name: with hyphen",
			vars: []EnvVar{
				{Name: "LOG-LEVEL", Value: "info"},
			},
			expectError: true,
			errorMsg:    "must be UPPERCASE_WITH_UNDERSCORES",
		},
		{
			name: "duplicate name",
			vars: []EnvVar{
				{Name: "LOG_LEVEL", Value: "info"},
				{Name: "DEBUG", Value: "false"},
				{Name: "LOG_LEVEL", Value: "debug"},
			},
			expectError: true,
			errorMsg:    "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateEnv(tt.vars)

			hasError := errs.HasErrors()
			if hasError != tt.expectError {
				t.Fatalf("validateEnv() hasError = %v, expectError = %v",
					hasError, tt.expectError)
			}

			if hasError && tt.errorMsg != "" {
				if !stringContains(errs.Error(), tt.errorMsg) {
					t.Errorf("Error message %q does not contain %q",
						errs.Error(), tt.errorMsg)
				}
			}
		})
	}
}

// TestValidationError_Format tests error message formatting.
func TestValidationError_Format(t *testing.T) {
	errs := ValidationError{}
	errs.Add("metadata.name is required")
	errs.AddExample("metadata:\n  name: my-app")
	errs.Add("spec.localPort must be 1-65535")

	errStr := errs.Error()

	// Check header
	if !stringContains(errStr, "2 errors") {
		t.Errorf("Error message missing error count")
	}

	// Check numbering
	if !stringContains(errStr, "1.") || !stringContains(errStr, "2.") {
		t.Errorf("Error message missing error numbering")
	}

	// Check example
	if !stringContains(errStr, "Example") {
		t.Errorf("Error message missing Example section")
	}

	t.Logf("Error output:\n%s", errStr)
}

// ============================================================
// Test Helpers
// ============================================================

func stringContains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
```

Add import at top:
```go
import "strings"
```

---

## Key Validation Decisions

### Decision 1: Fail-on-all-errors vs. Fail-fast

**Question**: Should we report first error or all errors?

**Answer**: **Report all errors at once**
- User doesn't have to iterate fixing one at a time
- Addresses root causes faster
- Better user experience

### Decision 2: When to validate file existence

**Question**: Should `dockerfilePath` validation check file exists?

**Answer**: **Contextual**
- In `Validate()`: Format check only (path looks reasonable)
- In `ValidateWithContext()`: File existence check (need project root)
- In loader: Full validation after discovering project root

### Decision 3: Strict vs. permissive port validation

**Question**: Allow privileged ports (<1024)?

**Answer**: **Allow but don't warn** (Phase 4 can add warnings)
- User might want port 80/443
- Docker Desktop handles privilege
- Better to be permissive in Phase 1

---

## Critical Points

### 1. Regex Patterns Must Be Tested

❌ Wrong pattern allows "a--b":
```go
`^[a-z0-9-]+$`  // Bad: allows consecutive hyphens

✅ Right pattern only allows single hyphens:
`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`  // Correct
```

### 2. Error Messages Must Have Examples

❌ Unhelpful error:
```
metadata.name: invalid format
```

✅ Helpful error:
```
metadata.name: must be DNS-1123 compliant
Example:
  metadata:
    name: my-app
```

### 3. Duplicate Environment Variables

Must track seen names to detect duplicates:
```go
seenNames := make(map[string]bool)
for _, v := range spec.Env {
    if seenNames[v.Name] {
        return error("duplicate")
    }
    seenNames[v.Name] = true
}
```

---

## Checklist for Task 1.2

- [ ] Create `pkg/config/validation.go`
- [ ] Create `pkg/config/errors.go`
- [ ] Create `pkg/config/validation_test.go`
- [ ] Implement `Validate()` method
- [ ] Implement `ValidateWithContext()` method
- [ ] All validation functions work correctly
- [ ] `ValidationError` formats messages nicely
- [ ] All test cases pass
- [ ] Test coverage >85%
- [ ] Run: `go test ./pkg/config -v`
- [ ] Run: `go test ./pkg/config -cover`

---

## Testing Validation Manually

```bash
# Create a bad config
cat > bad-config.yaml <<EOF
apiVersion: kudev.io/v1alpha1
kind: DeploymentConfig
metadata:
  name: "MyApp"  # Bad: uppercase
spec:
  imageName: myapp
  dockerfilePath: ./Dockerfile
  namespace: default
  replicas: 0  # Bad: must be ≥1
  localPort: 70000  # Bad: too high
  servicePort: 8080
  env:
    - name: log-level  # Bad: must be uppercase
      value: info
EOF

# Parse and validate
go run ./cmd validate --config bad-config.yaml
# Should show all 4 errors at once!
```

---

## Next Steps

1. **Implement this task** ← You are here
2. Move to **Task 1.3** → Config loader uses Validate()
3. Move to **Task 1.4** → Context validator uses Spec.KubeContext
4. When loading config, call `ValidateWithContext()` with project root



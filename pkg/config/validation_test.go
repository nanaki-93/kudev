package config

import (
	"context"
	"strings"
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
				Metadata: MetadataConfig{
					Name: "myapp",
				},
				Spec: SpecConfig{
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
				Metadata: MetadataConfig{
					Name: "myapp",
				},
				Spec: SpecConfig{
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
				Metadata: MetadataConfig{
					Name: "myapp",
				},
				Spec: SpecConfig{
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
				Metadata: MetadataConfig{
					Name: "frontend",
				},
				Spec: SpecConfig{
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
				Metadata:   MetadataConfig{}, // Missing name
				Spec: SpecConfig{
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
				Metadata: MetadataConfig{
					Name: "app",
				},
				Spec: SpecConfig{
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
				Metadata: MetadataConfig{
					Name: "app",
				},
				Spec: SpecConfig{
					ImageName: "app",
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
				Metadata: MetadataConfig{
					Name: "app",
				},
				Spec: SpecConfig{
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
			name: "unique name",
			vars: []EnvVar{
				{Name: "LOG_LEVEL", Value: "info"},
				{Name: "DEBUG", Value: "false"},
				{Name: "LOG_LEVEL", Value: "debug"},
			},
			expectError: true,
			errorMsg:    "unique",
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
	errs.AddWithExample("metadata.name is required", "metadata:\n  name: my-app")
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

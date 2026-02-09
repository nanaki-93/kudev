package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nanaki-93/kudev/pkg/config"
	"github.com/nanaki-93/kudev/pkg/kubeconfig"
	"github.com/nanaki-93/kudev/pkg/logging"
)

// TestPhase1_FullFlow tests the complete Phase 1 flow end-to-end.
//
// This integration test verifies:
//  1. Create config via FileConfigLoader
//  2. Load config from file
//  3. Validate config
//  4. Check context safety
//  5. Access config in "command"
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
		Metadata: config.MetadataConfig{
			Name: "test-app",
		},
		Spec: config.SpecConfig{
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
	cfg := config.NewDeploymentConfig("discovered-app")
	loader := config.NewFileConfigLoader("", projectDir, projectDir)
	if err := loader.Save(context.Background(), cfg, filepath.Join(projectDir, ".kudev.yaml")); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Simulate running kudev from deep in project
	deepLoader := config.NewFileConfigLoader("", projectDir, componentDir)
	found, err := deepLoader.Discover()

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
		Metadata: config.MetadataConfig{
			Name: "test-app",
		},
		Spec: config.SpecConfig{
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
		Metadata: config.MetadataConfig{
			Name: "Invalid-Name", // Uppercase not allowed
		},
		Spec: config.SpecConfig{
			ImageName:      "test-app",
			DockerfilePath: "./Dockerfile",
			Namespace:      "default",
			LocalPort:      70000, // Invalid port
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
		"lowercase",     // Hint about invalid name
		"metadata.name", // Field path
		"localPort",     // Field name
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
	return strings.Contains(haystack, needle)
}

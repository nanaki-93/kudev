package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFileConfigLoader_LoadFromPath tests loading from explicit savePath.
func TestFileConfigLoader_LoadFromPath(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".kudev.yaml")

	configContent := `apiVersion: kudev.io/v1alpha1
kind: DeploymentConfig
metadata:
  name: test-app
spec:
  imageName: test-app
  dockerfilePath: ./Dockerfile
  namespace: default
  replicas: 1
  localPort: 8080
  servicePort: 8080
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	loader := NewFileConfigLoader("", "", tmpDir)
	cfg, err := loader.LoadFromPath(context.Background(), configPath)

	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if cfg.Metadata.Name != "test-app" {
		t.Errorf("Name = %s, want test-app", cfg.Metadata.Name)
	}

	if cfg.Spec.ImageName != "test-app" {
		t.Errorf("ImageName = %s, want test-app", cfg.Spec.ImageName)
	}
}

// TestFileConfigLoader_LoadFromPath_NotFound tests error when file doesn't exist.
func TestFileConfigLoader_LoadFromPath_NotFound(t *testing.T) {
	loader := NewFileConfigLoader("", "", "")
	_, err := loader.LoadFromPath(context.Background(), "/nonexistent/savePath/.kudev.yaml")

	if err == nil {
		t.Fatalf("LoadFromPath() should return error for nonexistent file")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error message should contain 'not found', got: %v", err)
	}
}

// TestFileConfigLoader_LoadFromPath_InvalidYAML tests error for invalid YAML.
func TestFileConfigLoader_LoadFromPath_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".kudev.yaml")

	invalidYAML := `
invalid: yaml:
  - structure: [
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	loader := NewFileConfigLoader("", "", tmpDir)
	_, err := loader.LoadFromPath(context.Background(), configPath)

	if err == nil {
		t.Fatalf("LoadFromPath() should return error for invalid YAML")
	}

	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("Error message should contain 'parse', got: %v", err)
	}
}

// TestFileConfigLoader_Discover tests config discovery in directory hierarchy.
func TestFileConfigLoader_Discover(t *testing.T) {
	// Create directory structure:
	// project/
	//   .kudev.yaml
	//   src/
	//     components/
	//       .kudev.yaml  <- Only here

	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	srcDir := filepath.Join(projectDir, "src")
	componentDir := filepath.Join(srcDir, "components")

	if err := os.MkdirAll(componentDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	configPath := filepath.Join(componentDir, ".kudev.yaml")
	configContent := `apiVersion: kudev.io/v1alpha1
kind: DeploymentConfig
metadata:
  name: test-app
spec:
  imageName: test-app
  dockerfilePath: ./Dockerfile
  namespace: default
  replicas: 1
  localPort: 8080
  servicePort: 8080
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Simulate running kudev from componentDir
	loader := NewFileConfigLoader("", projectDir, componentDir)
	found, err := loader.discover()

	if err != nil {
		t.Fatalf("discover() error = %v", err)
	}

	if found != configPath {
		t.Errorf("discover() = %s, want %s", found, configPath)
	}
}

// TestFileConfigLoader_Discover_NotFound tests helpful error when config not found.
func TestFileConfigLoader_Discover_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	loader := NewFileConfigLoader("", "", emptyDir)
	err := loader.notFoundError()

	if err == nil {
		t.Fatalf("notFoundError() should return error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "kudev init") {
		t.Errorf("Error should suggest 'kudev init', got: %s", errStr)
	}

	if !strings.Contains(errStr, "Searched in") {
		t.Errorf("Error should show search paths, got: %s", errStr)
	}
}

// TestFileConfigLoader_ApplyDefaults tests that defaults are applied.
func TestFileConfigLoader_ApplyDefaults(t *testing.T) {
	cfg := &DeploymentConfig{
		APIVersion: "kudev.io/v1alpha1",
		Kind:       "DeploymentConfig",
		Metadata: MetadataConfig{
			Name: "app",
		},
		Spec: SpecConfig{
			ImageName:      "app",
			DockerfilePath: "./Dockerfile",
			// Empty: will be defaulted
			Replicas:    0, // Will default to 1
			LocalPort:   0, // Will default to 8080
			ServicePort: 0, // Will default to 8080
		},
	}

	ApplyDefaults(cfg)

	if cfg.Spec.Namespace != "default" {
		t.Errorf("Namespace = %s, want default", cfg.Spec.Namespace)
	}

	if cfg.Spec.Replicas != 1 {
		t.Errorf("Replicas = %d, want 1", cfg.Spec.Replicas)
	}

	if cfg.Spec.LocalPort != 8080 {
		t.Errorf("LocalPort = %d, want 8080", cfg.Spec.LocalPort)
	}

	if cfg.Spec.ServicePort != 8080 {
		t.Errorf("ServicePort = %d, want 8080", cfg.Spec.ServicePort)
	}
}

// TestDiscoverProjectRoot tests project root detection.
func TestDiscoverProjectRoot(t *testing.T) {
	tests := []struct {
		name     string
		markers  []string
		wantRoot bool
	}{
		{
			name:     "with .git",
			markers:  []string{".git"},
			wantRoot: true,
		},
		{
			name:     "with go.mod",
			markers:  []string{"go.mod"},
			wantRoot: true,
		},
		{
			name:     "with package.json",
			markers:  []string{"package.json"},
			wantRoot: true,
		},
		{
			name:     "no markers",
			markers:  []string{},
			wantRoot: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			projectDir := filepath.Join(tmpDir, "project")

			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}

			// Create markers
			for _, marker := range tt.markers {
				markerPath := filepath.Join(projectDir, marker)
				if err := os.WriteFile(markerPath, []byte{}, 0644); err != nil {
					t.Fatalf("Failed to create marker: %v", err)
				}
			}

			root, err := DiscoverProjectRoot(projectDir)

			if tt.wantRoot {
				if err != nil {
					t.Errorf("DiscoverProjectRoot() error = %v, want nil", err)
				}
				if root != projectDir {
					t.Errorf("DiscoverProjectRoot() = %s, want %s", root, projectDir)
				}
			} else {
				if err == nil {
					t.Errorf("DiscoverProjectRoot() should return error for no markers")
				}
			}
		})
	}
}

// TestIsProjectRoot tests project root detection logic.
func TestIsProjectRoot(t *testing.T) {
	tests := []struct {
		name    string
		markers []string
		want    bool
	}{
		{name: ".git", markers: []string{".git"}, want: true},
		{name: "go.mod", markers: []string{"go.mod"}, want: true},
		{name: "package.json", markers: []string{"package.json"}, want: true},
		{name: "no markers", markers: []string{}, want: false},
		{name: "multiple markers", markers: []string{".git", "go.mod"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for _, marker := range tt.markers {
				markerPath := filepath.Join(tmpDir, marker)
				if err := os.WriteFile(markerPath, []byte{}, 0644); err != nil {
					t.Fatalf("Failed to create marker: %v", err)
				}
			}

			got := isProjectRoot(tmpDir)
			if got != tt.want {
				t.Errorf("isProjectRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileConfigLoader_Save(t *testing.T) {
	tests := []struct {
		name      string
		savePath  string
		fcl       *DeploymentConfig
		expectErr bool
	}{
		{
			name:      "empty save savePath",
			savePath:  "",
			fcl:       NewDeploymentConfig("test1"),
			expectErr: true,
		},

		{
			name:      "valid config file",
			savePath:  "./testdata/config.yaml",
			fcl:       NewDeploymentConfig("test1"),
			expectErr: false,
		},
		{
			name:      "with creating dir",
			savePath:  "./testdata/testWithDir/nonexistent.yaml",
			fcl:       NewDeploymentConfig("test2"),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		fcl := NewFileConfigLoader("", "", "")
		err := fcl.Save(context.Background(), tt.fcl, tt.savePath)
		if (err != nil) != tt.expectErr {
			t.Errorf("Save() error = %v, wantErr %v", err, tt.expectErr)
		}
		if err != nil {
			continue
		}
		_, err = os.Stat(tt.savePath)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("Save() error checking file: %v", err)
		} else if err == nil {
			// Clean up after test
			err = os.Remove(tt.savePath)
			if err != nil {
				t.Errorf("Error removing file: %v , remove it manually", err)
			}

			// Remove parent directory only if it's a subdirectory of testdata
			parentDir := filepath.Dir(tt.savePath)
			grandParentDir := filepath.Dir(parentDir)
			if filepath.Base(grandParentDir) == "testdata" {
				err = os.Remove(parentDir)
				if err != nil && !os.IsNotExist(err) {
					t.Errorf("Error removing directory: %v, remove it manually", err)
				}
			}

		}
	}
}

func TestFindConfigFile(t *testing.T) {
	tests := []struct {
		name         string
		startDir     string
		expectedPath string
		expectedErr  bool
	}{
		{
			name:         "no config file",
			startDir:     filepath.Join("testdata", "noConfig"),
			expectedPath: "",
			expectedErr:  true,
		},
		{
			name:         "config file not found with empty startDir",
			startDir:     "",
			expectedPath: "",
			expectedErr:  true,
		},
		{
			name:         "config file",
			startDir:     filepath.Join("testdata", "config"),
			expectedPath: filepath.Join("testdata", "config", ".kudev.yaml"),
			expectedErr:  false,
		},
	}

	for _, tt := range tests {
		path, err := FindConfigFile(tt.startDir)
		if (err != nil) != tt.expectedErr {
			t.Errorf("FindConfigFile() error = %v, wantErr %v", err, tt.expectedErr)
		}
		if path != tt.expectedPath {
			t.Errorf("FindConfigFile() = %s, want %s", path, tt.expectedPath)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		expectedErr bool
	}{
		{name: "no config file",
			configPath:  filepath.Join("testdata", "noConfig"),
			expectedErr: true},
		{name: "valid config",
			configPath:  filepath.Join("testdata", "config", ".kudev.yaml"),
			expectedErr: false},
	}

	for _, tt := range tests {
		config, err := LoadConfig(context.Background(), tt.configPath)
		if (err != nil) != tt.expectedErr {
			t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.expectedErr)
		}
		if config == nil && !tt.expectedErr {
			t.Errorf("LoadConfig() returned nil config")
		}
	}
}

# Task 1.3: Implement Configuration Loader

## Overview

This task implements the **configuration discovery and loading system**. It must:
1. **Discover** `.kudev.yaml` in project hierarchy
2. **Load** YAML and parse to Go types
3. **Validate** using validators from Task 1.2
4. **Apply defaults** for optional fields
5. **Provide clear errors** if config not found

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate (file system traversal, YAML parsing)  
**Dependencies**: Task 1.1 (Types), Task 1.2 (Validation)  
**Files to Create**:
- `pkg/config/loader.go` — ConfigLoader interface + implementation
- `pkg/config/defaults.go` — Default values
- `pkg/config/loader_test.go` — Tests

---

## The Problem Config Loading Solves

Users should be able to run `kudev` from anywhere in their project:

```bash
$ cd ~/project/src/components
$ kudev status
# Should find ~/project/.kudev.yaml automatically, not fail

$ cd ~/project
$ kudev status  
# Should find ./.kudev.yaml

$ cd ~/project && kudev --config ./config/dev.yaml status
# Should respect --config override
```

---

## Discovery Algorithm

The **discovery algorithm** searches for `.kudev.yaml`:

```
1. Check for --config flag
   └─ If set: Use that path (must exist)
   
2. Check current directory
   └─ If found: Use it
   
3. Check parent directories (walk up to root)
   └─ Stop at project root (heuristics: .git, go.mod, .kudev.yaml)
   └─ If found: Use it
   
4. Check home directory (~/.kudev/config)
   └─ For global config (not recommended for projects)
   
5. Not found
   └─ Return helpful error with what we searched
```

### Project Root Detection

Heuristics to detect project root:
1. Directory containing `.git` (VCS root)
2. Directory containing `go.mod` (Go project root)
3. Directory containing `.kudev.yaml` (kudev project root)
4. Directory containing `package.json` (Node project root)
5. Current working directory + parent == root `/` or `C:\` (filesystem root)

---

## Implementation: loader.go

Create `pkg/config/loader.go`:

```go
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// ConfigLoader loads configuration from files.
//
// Interface allows multiple implementations (file, env, flags, etc.)
// in the future.
type ConfigLoader interface {
	// Load discovers and loads configuration.
	// Returns the loaded configuration or an error.
	//
	// Discovery order:
	//   1. --config flag (if set)
	//   2. Current directory .kudev.yaml
	//   3. Parent directories (up to project root)
	//   4. Home directory ~/.kudev/config
	Load(ctx context.Context) (*DeploymentConfig, error)

	// LoadFromPath loads configuration from a specific file.
	LoadFromPath(ctx context.Context, path string) (*DeploymentConfig, error)

	// Save writes configuration to file.
	Save(ctx context.Context, cfg *DeploymentConfig, path string) error
}

// FileConfigLoader implements ConfigLoader for file-based configuration.
type FileConfigLoader struct {
	// ConfigPath is an optional explicit path (from --config flag)
	ConfigPath string

	// ProjectRoot is the detected project root directory
	ProjectRoot string

	// WorkingDir is the current working directory (for relative path resolution)
	WorkingDir string
}

// NewFileConfigLoader creates a new file-based configuration loader.
func NewFileConfigLoader(configPath, projectRoot, workingDir string) *FileConfigLoader {
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}

	return &FileConfigLoader{
		ConfigPath:  configPath,
		ProjectRoot: projectRoot,
		WorkingDir:  workingDir,
	}
}

// Load discovers and loads configuration.
//
// Search order:
//   1. If ConfigPath set: use it (explicit --config flag)
//   2. Search: cwd → parents → project root
//   3. Fall back: ~/.kudev/config
//
// Returns:
//   - Valid DeploymentConfig if found and valid
//   - Clear error message if not found or invalid
func (fcl *FileConfigLoader) Load(ctx context.Context) (*DeploymentConfig, error) {
	// Step 1: Explicit --config flag (highest priority)
	if fcl.ConfigPath != "" {
		cfg, err := fcl.LoadFromPath(ctx, fcl.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", fcl.ConfigPath, err)
		}
		return cfg, nil
	}

	// Step 2: Discover in project hierarchy
	path, err := fcl.discover()
	if err == nil {
		cfg, err := fcl.LoadFromPath(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
		}

		// Validate with full filesystem context
		if err := cfg.ValidateWithContext(fcl.ProjectRoot); err != nil {
			return nil, fmt.Errorf("configuration validation failed: %w", err)
		}

		return cfg, nil
	}

	// Step 3: Home directory fallback
	homeDir, err := os.UserHomeDir()
	if err == nil {
		homePath := filepath.Join(homeDir, ".kudev", "config")
		if _, err := os.Stat(homePath); err == nil {
			cfg, err := fcl.LoadFromPath(ctx, homePath)
			if err != nil {
				return nil, fmt.Errorf("failed to load config from home dir: %w", err)
			}
			return cfg, nil
		}
	}

	// Step 4: Not found - provide helpful error
	return nil, fcl.notFoundError()
}

// LoadFromPath loads configuration from a specific file path.
//
// Process:
//   1. Read file
//   2. Parse YAML
//   3. Convert to DeploymentConfig
//   4. Apply defaults
//   5. Validate
//
// Returns:
//   - Fully initialized DeploymentConfig
//   - Clear error if parsing or validation fails
func (fcl *FileConfigLoader) LoadFromPath(ctx context.Context, path string) (*DeploymentConfig, error) {
	// Normalize path
	path = filepath.Clean(path)

	// If relative, resolve from working directory or project root
	if !filepath.IsAbs(path) {
		// Try relative to working directory first
		checkPath := filepath.Join(fcl.WorkingDir, path)
		if _, err := os.Stat(checkPath); err == nil {
			path = checkPath
		} else if fcl.ProjectRoot != "" {
			// Fall back to project root
			path = filepath.Join(fcl.ProjectRoot, path)
		}
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(content) == 0 {
		return nil, fmt.Errorf("config file is empty: %s", path)
	}

	// Parse YAML
	cfg := &DeploymentConfig{}
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w\nFile: %s", err, path)
	}

	// Apply defaults (before validation)
	ApplyDefaults(cfg)

	// Validate basic structure
	if err := cfg.Validate(ctx); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes configuration to a file.
//
// Creates parent directories if needed.
// Overwrites existing file.
func (fcl *FileConfigLoader) Save(ctx context.Context, cfg *DeploymentConfig, path string) error {
	if path == "" {
		return fmt.Errorf("save path cannot be empty")
	}

	// Validate before saving
	if err := cfg.Validate(ctx); err != nil {
		return fmt.Errorf("cannot save invalid configuration: %w", err)
	}

	// Convert to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to YAML: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ============================================================
// Discovery Logic
// ============================================================

// discover searches for .kudev.yaml in the project hierarchy.
//
// Search order:
//   1. Current working directory
//   2. Parent directories (upward walk)
//   3. Stop at project root (detect via .git, go.mod, etc.)
//
// Returns:
//   - Full path to .kudev.yaml if found
//   - Error if not found
func (fcl *FileConfigLoader) discover() (string, error) {
	searchPaths := fcl.generateSearchPaths()

	for _, searchPath := range searchPaths {
		configPath := filepath.Join(searchPath, ".kudev.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	return "", fmt.Errorf("config not found")
}

// generateSearchPaths generates the list of directories to search.
//
// Returns directories from CWD to project root.
func (fcl *FileConfigLoader) generateSearchPaths() []string {
	var paths []string

	current := fcl.WorkingDir
	visited := make(map[string]bool)  // Prevent infinite loops on symlinks

	for {
		// Prevent infinite loops
		if visited[current] {
			break
		}
		visited[current] = true

		paths = append(paths, current)

		// Check if this is a project root
		if isProjectRoot(current) {
			break
		}

		// Walk up to parent
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}

		current = parent
	}

	return paths
}

// isProjectRoot checks if a directory is a project root.
//
// Heuristics:
//   - Contains .git (VCS root)
//   - Contains go.mod (Go project)
//   - Contains package.json (Node project)
//   - Contains Makefile (Common project marker)
//   - Contains Dockerfile (Docker project)
func isProjectRoot(path string) bool {
	markers := []string{
		".git",
		"go.mod",
		"package.json",
		"Makefile",
		"Dockerfile",
		".kudev.yaml",
	}

	for _, marker := range markers {
		markerPath := filepath.Join(path, marker)
		if _, err := os.Stat(markerPath); err == nil {
			return true
		}
	}

	return false
}

// notFoundError generates a helpful error message when config is not found.
func (fcl *FileConfigLoader) notFoundError() error {
	searched := fcl.generateSearchPaths()
	searchedStr := strings.Join(searched, "\n  - ")

	suggestions := []string{
		"Run 'kudev init' to create a new .kudev.yaml",
		"Or place .kudev.yaml in your project root",
		"Or specify config path with: kudev --config <path>",
	}

	return fmt.Errorf(
		"configuration file (.kudev.yaml) not found\n\n"+
			"Searched in:\n  - %s\n\n"+
			"Suggestions:\n  - %s",
		searchedStr,
		strings.Join(suggestions, "\n  - "),
	)
}

// ============================================================
// Utility Functions
// ============================================================

// DiscoverProjectRoot finds the project root starting from the given directory.
//
// Uses same heuristics as discover algorithm.
func DiscoverProjectRoot(startDir string) (string, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	current := startDir
	visited := make(map[string]bool)

	for {
		if visited[current] {
			break
		}
		visited[current] = true

		if isProjectRoot(current) {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}

		current = parent
	}

	return "", fmt.Errorf("project root not found (no .git, go.mod, package.json, etc.)")
}

// FindConfigFile searches for .kudev.yaml configuration file.
//
// Returns the path to the config file or error if not found.
func FindConfigFile(startDir string) (string, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	loader := &FileConfigLoader{
		WorkingDir: startDir,
	}

	return loader.discover()
}

// LoadConfig is a convenience function to load configuration with defaults.
//
// Equivalent to:
//   loader := NewFileConfigLoader(configPath, projectRoot, workingDir)
//   return loader.Load(ctx)
func LoadConfig(ctx context.Context, configPath string) (*DeploymentConfig, error) {
	projectRoot, _ := DiscoverProjectRoot("")  // Error ignored - not required
	cwd, _ := os.Getwd()

	loader := NewFileConfigLoader(configPath, projectRoot, cwd)
	return loader.Load(ctx)
}
```

---

## Implementation: defaults.go

Create `pkg/config/defaults.go`:

```go
package config

// ApplyDefaults fills in missing configuration values with sensible defaults.
//
// This function is called:
//   - After loading YAML
//   - Before validation
//   - To provide good UX when fields are omitted
//
// Defaults applied:
//   - Namespace: "default" (K8s standard)
//   - Replicas: 1 (single pod)
//   - LocalPort: 8080 (common HTTP port)
//   - ServicePort: 8080 (common HTTP port)
//   - APIVersion: "kudev.io/v1alpha1" (current version)
//   - Kind: "DeploymentConfig" (resource type)
//
// Fields NOT defaulted:
//   - metadata.name: Must be explicitly provided (prevents accidents)
//   - imageName: Must be explicit
//   - dockerfilePath: Must be explicit
//   - env: Empty list is OK (no defaults)
//   - kubeContext: Empty string OK (use whitelist)
//   - buildContextExclusions: Empty list OK
//
// Philosophy:
//   - Default where reasonable
//   - Require explicit for critical paths
//   - Prevent silent failures
func ApplyDefaults(cfg *DeploymentConfig) {
	if cfg == nil {
		return
	}

	// API version and kind
	if cfg.APIVersion == "" {
		cfg.APIVersion = "kudev.io/v1alpha1"
	}

	if cfg.Kind == "" {
		cfg.Kind = "DeploymentConfig"
	}

	// Namespace
	if cfg.Spec.Namespace == "" {
		cfg.Spec.Namespace = "default"
	}

	// Replicas
	if cfg.Spec.Replicas <= 0 {
		cfg.Spec.Replicas = 1
	}

	// Ports (both 8080 if not specified)
	if cfg.Spec.LocalPort <= 0 {
		cfg.Spec.LocalPort = 8080
	}

	if cfg.Spec.ServicePort <= 0 {
		cfg.Spec.ServicePort = 8080
	}

	// Environment variables (empty is OK, no defaults)

	// KubeContext (empty is OK, uses whitelist validation)

	// BuildContextExclusions (empty is OK, just won't exclude extra files)
}

// DefaultConfig returns a minimal valid configuration with all defaults applied.
//
// Used for testing and as a template for `kudev init`.
//
// Example usage in init command:
//   cfg := DefaultConfig("my-app")
//   // Ask user for customizations
//   // Save to .kudev.yaml
func DefaultConfig(appName string) *DeploymentConfig {
	cfg := &DeploymentConfig{
		APIVersion: "kudev.io/v1alpha1",
		Kind:       "DeploymentConfig",
		Metadata: ConfigMetadata{
			Name: appName,
		},
		Spec: DeploymentSpec{
			ImageName:      appName,
			DockerfilePath: "./Dockerfile",
			Namespace:      "default",
			Replicas:       1,
			LocalPort:      8080,
			ServicePort:    8080,
			Env:            []EnvVar{},
		},
	}

	return cfg
}
```

---

## Testing: loader_test.go

Create `pkg/config/loader_test.go`:

```go
package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"strings"
)

// TestFileConfigLoader_LoadFromPath tests loading from explicit path.
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
	_, err := loader.LoadFromPath(context.Background(), "/nonexistent/path/.kudev.yaml")

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

	if !strings.Contains(err.Error(), "parse") {
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
		Metadata: ConfigMetadata{
			Name: "app",
		},
		Spec: DeploymentSpec{
			ImageName:      "app",
			DockerfilePath: "./Dockerfile",
			// Empty: will be defaulted
			Replicas: 0,  // Will default to 1
			LocalPort:  0,  // Will default to 8080
			ServicePort: 0,  // Will default to 8080
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
```

---

## Key Design Decisions

### Decision 1: What to Default

**Question**: Should we default ALL fields?

**Answer**: **Default cautiously**
- ✅ Default: namespace (standard), replicas (safe), ports (common)
- ❌ Don't default: metadata.name (must be explicit), imageName, dockerfilePath

**Reasoning**: 
- Required fields prevent silent failures
- Optional fields benefit from smart defaults

### Decision 2: Search Order

**Question**: Search from CWD upward or downward?

**Answer**: **Search upward from CWD**
```bash
/project/
  .kudev.yaml
  src/
    components/
      mycomponent/
        $ kudev status  # Run from here
        # Finds /project/.kudev.yaml by walking up
```

This matches: `kubectl`, `git`, `node` package.json discovery

### Decision 3: Project Root Detection

**Question**: What heuristics to use?

**Answer**: **Multiple heuristics** (.git, go.mod, package.json, etc.)
- Works with many project types
- Stops search at first found marker
- Prevents infinite loop on symlinks

---

## Critical Points

### 1. Prevent Infinite Loops on Symlinks

```go
visited := make(map[string]bool)

for {
    if visited[current] {
        break  // Already saw this path
    }
    visited[current] = true
    
    // ... search logic ...
    
    parent := filepath.Dir(current)
    if parent == current {
        break  // Reached filesystem root
    }
    current = parent
}
```

### 2. Error Messages Must Be Helpful

❌ Bad:
```
config not found
```

✅ Good:
```
configuration file (.kudev.yaml) not found

Searched in:
  - /home/user/project/src/components
  - /home/user/project/src
  - /home/user/project
  - /home/user
  - /home
  - /

Suggestions:
  - Run 'kudev init' to create a new .kudev.yaml
  - Or place .kudev.yaml in your project root
  - Or specify config path with: kudev --config <path>
```

### 3. Relative Path Resolution

Two contexts where paths are relative:
1. **In YAML**: `dockerfilePath: ./Dockerfile`
   - Should be relative to project root (not CWD)
   - Resolved in loader.go

2. **--config flag**: `kudev --config ./dev.yaml`
   - Should be relative to CWD (user perspective)
   - Resolved with `filepath.Join(WorkingDir, path)`

---

## Checklist for Task 1.3

- [ ] Create `pkg/config/loader.go`
- [ ] Create `pkg/config/defaults.go`
- [ ] Create `pkg/config/loader_test.go`
- [ ] Implement `FileConfigLoader` type
- [ ] Implement `Load()` method (discovery + loading)
- [ ] Implement `LoadFromPath()` method
- [ ] Implement `Save()` method
- [ ] Implement `discover()` algorithm
- [ ] Implement project root detection
- [ ] Apply defaults before validation
- [ ] Generate helpful error messages
- [ ] All tests pass: `go test ./pkg/config -v`
- [ ] Test coverage >80%

---

## Testing the Loader Manually

```bash
# Create test config
mkdir -p /tmp/kudev-test/src
cat > /tmp/kudev-test/.kudev.yaml <<EOF
apiVersion: kudev.io/v1alpha1
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
EOF

# Create Dockerfile
touch /tmp/kudev-test/Dockerfile

# Test discovery from subdirectory
cd /tmp/kudev-test/src
kudev validate
# Should find /tmp/kudev-test/.kudev.yaml

# Test explicit config
cd /tmp
kudev --config /tmp/kudev-test/.kudev.yaml validate
# Should work
```

---

## Integration with Other Tasks

```
Task 1.1 (Types)
    ↓
Task 1.2 (Validation)
    ↓
Task 1.3 (Loader) ← You are here
    ↓
Load → Validate → Ready for Task 1.4 (Context)
```

Task 1.3 is the **glue** that ties together:
- File system (discovery)
- YAML parsing (loading)
- Validation (checking)
- Defaults (user experience)

---

## Next Steps

1. **Implement this task** ← You are here
2. **Task 1.4** → Use loader to get config, validate context
3. **Task 1.5** → Cobra commands use loader to get config
4. **Phase 2** → Extend loader to support ConfigMaps, Secrets



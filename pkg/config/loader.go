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
type LoaderConfig interface {
	// Load discovers and loads configuration.
	// Returns the loaded configuration or an error.
	//
	// Discovery order:
	//   1. --config flag (if set)
	//   2. Current directory .kudev.yaml
	//   3. Parent directories (up to project root)
	//   4. Home directory ~/.kudev/config
	Load(ctx context.Context) (*DeploymentConfig, error)
	LoadFromPath(ctx context.Context, path string) (*DeploymentConfig, error)
	Save(ctx context.Context, config *DeploymentConfig) error
}

type FileConfigLoader struct {
	Path        string
	ProjectRoot string
	WorkingDir  string
}

func NewFileConfigLoader(configPath, projectRoot, workingDir string) *FileConfigLoader {
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}
	return &FileConfigLoader{Path: configPath, ProjectRoot: projectRoot, WorkingDir: workingDir}
}

// Load discovers and loads configuration.
//
// Search order:
//  1. If ConfigPath set: use it (explicit --config flag)
//  2. Search: cwd → parents → project root
//  3. Fall back: ~/.kudev/config
//
// Returns:
//   - Valid DeploymentConfig if found and valid
//   - Clear error message if not found or invalid
func (fcl *FileConfigLoader) Load(ctx context.Context) (*DeploymentConfig, error) {

	if fcl.Path != "" {
		cfg, err := fcl.LoadFromPath(ctx, fcl.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration from %s: %w", fcl.Path, err)
		}
		return cfg, nil
	}

	path, err := fcl.discover()
	if err == nil {
		cfg, err := fcl.LoadFromPath(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration from %s: %w", path, err)
		}

		if err := cfg.ValidateWithContext(fcl.ProjectRoot); err != nil {
			return nil, fmt.Errorf("invalid configuration found at %s: %w", path, err)
		}
		return cfg, nil
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		homePath := filepath.Join(homeDir, ".kudev", "config")
		if _, err := os.Stat(homePath); err == nil {
			cfg, err := fcl.LoadFromPath(ctx, homePath)
			if err != nil {
				return nil, fmt.Errorf("failed to load configuration from home dir - %s: %w", homePath, err)
			}
			return cfg, nil
		}
	}

	return nil, fcl.notFoundError()
}

// LoadFromPath loads configuration from a specific file path.
//
// Process:
//  1. Read file
//  2. Parse YAML
//  3. Convert to DeploymentConfig
//  4. Apply defaults
//  5. Validate
//
// Returns:
//   - Fully initialized DeploymentConfig
//   - Clear error if parsing or validation fails
func (fcl *FileConfigLoader) LoadFromPath(ctx context.Context, path string) (*DeploymentConfig, error) {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		checkPath := filepath.Join(fcl.WorkingDir, path)
		if _, err := os.Stat(checkPath); err == nil {
			path = checkPath
		} else if fcl.ProjectRoot != "" {
			//fall back to project root
			path = filepath.Join(fcl.ProjectRoot, path)
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s", path)
		}
		return nil, fmt.Errorf("error reading config file %s: %w", path, err)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("config file %s is empty", path)
	}

	cfg := &DeploymentConfig{}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file %s: %w", path, err)
	}

	ApplyDefaults(cfg)

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

	if err := cfg.Validate(ctx); err != nil {
		return fmt.Errorf("cannot save invalid configuration: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to yaml: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}
	return nil

}

func (fcl *FileConfigLoader) discover() (string, error) {
	searchPaths := fcl.generateSearchPaths()
	for _, path := range searchPaths {
		configPath := filepath.Join(path, ".kudev.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}
	return "", fmt.Errorf("config file not found")
}

func (fcl *FileConfigLoader) generateSearchPaths() []string {
	var paths []string
	current := fcl.WorkingDir
	visited := make(map[string]bool) //prevent infinite loops on symlinks

	for {
		if visited[current] {
			break
		}
		paths = append(paths, current)
		if isProjectRoot(current) {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
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
		".kudev.yaml"}
	for _, marker := range markers {
		markerPath := filepath.Join(path, marker)
		if _, err := os.Stat(markerPath); err == nil {
			return true
		}
	}
	return false
}
func (fcl *FileConfigLoader) notFoundError() error {
	searched := fcl.generateSearchPaths()

	suggestion := []string{
		"Run 'kudev init' to create a new .kudev.yaml.",
		"Or place a .kudev.yaml file in your project root.",
		"Or specify a path with: kudev --config <path>",
	}
	return fmt.Errorf("configuration file (.kudev.yaml) not found \n\n"+
		"Searched in :\n - %s\n\n"+
		"Suggestions:\n - %s",
		strings.Join(searched, "\n - "),
		strings.Join(suggestion, "\n - "))
}

func DiscoverProjectRoot(startDir string) (string, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current working directory: %w", err)
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
			return "", fmt.Errorf("cannot get current working directory: %w", err)
		}
	}
	loader := FileConfigLoader{WorkingDir: startDir}
	return loader.discover()
}

// LoadConfig is a convenience function to load configuration with defaults.
//
// Equivalent to:
//
//	loader := NewFileConfigLoader(configPath, projectRoot, workingDir)
//	return loader.Load(ctx)
func LoadConfig(ctx context.Context, configPath string) (*DeploymentConfig, error) {
	projectRoot, _ := DiscoverProjectRoot("") // Error ignored - not required
	cwd, _ := os.Getwd()

	loader := NewFileConfigLoader(configPath, projectRoot, cwd)
	return loader.Load(ctx)
}

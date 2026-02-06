# Task 1.5: Implement Cobra CLI Scaffolding

## Overview

This task builds the **CLI command structure** using Cobra. It establishes:
1. **Root command** with global flags and initialization
2. **Version command** (simple starting point)
3. **Init command** (interactive config generation)
4. **Validate command** (config verification)
5. **Command pattern** for future tasks (Phase 2+)

**Effort**: ~2-3 hours  
**Complexity**: 🟡 Intermediate (Cobra, command structure)  
**Dependencies**: Task 1.1-1.4 (all foundation tasks)  
**Files to Create**:
- `cmd/root.go` — Root command + global setup
- `cmd/version.go` — Version command
- `cmd/init.go` — Init command (interactive)
- `cmd/validate.go` — Validate command
- `cmd/main.go` — Entry point
- `pkg/version/version.go` — Version info

---

## Command Structure

```
kudev
├── version          # Print version
├── init             # Initialize .kudev.yaml
├── validate         # Validate config
├── up               # Build + deploy (Phase 2)
├── down             # Delete deployment (Phase 3)
├── status           # Show deployment status (Phase 3)
├── logs             # Stream pod logs (Phase 4)
├── portfwd          # Port forwarding (Phase 4)
├── watch            # Watch for changes (Phase 5)
└── debug            # Debug info (Phase 6)
```

---

## Implementation: cmd/main.go

Create `cmd/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/yourusername/kudev/cmd/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

---

## Implementation: cmd/root.go

Create a `cmd` package with root command:

Create `cmd/commands/root.go`:

```go
package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourusername/kudev/pkg/config"
	"github.com/yourusername/kudev/pkg/kubeconfig"
	"github.com/yourusername/kudev/pkg/logging"
)

var (
	// Global flags
	configPath   string
	debugLogging bool
	forceContext bool

	// Loaded config (shared across commands)
	loadedConfig *config.DeploymentConfig

	// Kubeconfig validator
	validator *kubeconfig.ContextValidator

	rootCmd = &cobra.Command{
		Use:   "kudev",
		Short: "Kubernetes development helper",
		Long: `Kudev streamlines local Kubernetes development with automatic
building, deploying, and live-reloading.

Kudev manages the full development cycle:
  - Build Docker images automatically
  - Deploy to local K8s clusters (Docker Desktop, minikube, kind)
  - Stream logs from pods
  - Port forward for local access
  - Watch source files and hot reload

Safety features:
  - Whitelist for K8s contexts (prevents accidental prod deployments)
  - Configuration validation
  - Dry-run mode

Examples:
  kudev init               Create a .kudev.yaml configuration
  kudev validate           Verify configuration
  kudev up                 Build and deploy to K8s
  kudev logs               Show pod logs
  kudev portfwd            Setup port forwarding
  kudev watch              Watch for changes and hot reload

Documentation:
  https://github.com/yourusername/kudev
`,
		// PersistentPreRunE is called before any command
		// Use for global initialization (loading config, validating context)
		PersistentPreRunE: rootPersistentPreRun,

		// SilenceUsage silences the usage message on error
		// We'll handle our own error messages
		SilenceUsage: true,
	}
)

// init registers flags and subcommands
func init() {
	// Global flags (available to all commands)
	rootCmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		"",
		"Path to .kudev.yaml configuration file",
	)

	rootCmd.PersistentFlags().BoolVar(
		&debugLogging,
		"debug",
		false,
		"Enable debug logging output",
	)

	rootCmd.PersistentFlags().BoolVar(
		&forceContext,
		"force-context",
		false,
		"Skip K8s context safety check (use with caution!)",
	)

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(validateCmd)
	// Phase 2: rootCmd.AddCommand(upCmd)
	// Phase 3: rootCmd.AddCommand(downCmd, statusCmd)
	// Phase 4: rootCmd.AddCommand(logsCmd, portfwdCmd)
	// Phase 5: rootCmd.AddCommand(watchCmd)
}

// rootPersistentPreRun is the global initialization hook.
//
// This runs before any command execution and performs:
//   1. Setup logging
//   2. Load configuration (unless command is 'init')
//   3. Validate context safety
//   4. Store for use by subcommands
func rootPersistentPreRun(cmd *cobra.Command, args []string) error {
	// Step 1: Setup logging
	logging.Init(debugLogging)

	// Step 2: Skip config loading for certain commands
	// These commands don't need config:
	//   - version: just prints version
	//   - init: creates new config
	//   - help: shows help
	//   - --help, -h
	if cmd.Name() == "version" || cmd.Name() == "init" || cmd.Name() == "help" {
		return nil
	}

	// Determine if this is a help request
	if cmd.Flag("help").Changed {
		return nil  // Let Cobra handle help
	}

	// Step 3: Load configuration
	ctx := context.Background()
	cfg, err := config.LoadConfig(ctx, configPath)
	if err != nil {
		// Helpful error message
		return fmt.Errorf(
			"failed to load configuration: %w\n\n"+
				"Run 'kudev init' to create a new .kudev.yaml configuration",
			err,
		)
	}

	loadedConfig = cfg

	// Step 4: Validate context safety
	ctxValidator, err := kubeconfig.NewContextValidator(forceContext)
	if err != nil {
		return fmt.Errorf("failed to check Kubernetes context: %w", err)
	}

	if err := ctxValidator.Validate(); err != nil {
		return err  // Error already formatted by validator
	}

	validator = ctxValidator

	return nil
}

// GetLoadedConfig returns the configuration loaded in PersistentPreRun.
//
// Use this in subcommands to get the shared config instance.
// Safe to call only after PersistentPreRun has executed.
func GetLoadedConfig() *config.DeploymentConfig {
	return loadedConfig
}

// GetValidator returns the context validator.
func GetValidator() *kubeconfig.ContextValidator {
	return validator
}

// Execute runs the root command.
// This is called from main().
func Execute() error {
	return rootCmd.Execute()
}

// RootCmd returns the root command (useful for testing).
func RootCmd() *cobra.Command {
	return rootCmd
}
```

---

## Implementation: cmd/version.go

Create `cmd/commands/version.go`:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourusername/kudev/pkg/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print the version of kudev and related components.

Shows:
  - kudev version
  - Go version
  - OS/Architecture
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("kudev version " + version.Version)
		fmt.Println("Built with " + version.GoVersion)
		fmt.Printf("OS/Arch: %s/%s\n", version.OS, version.Arch)
	},
}
```

Create `pkg/version/version.go`:

```go
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the kudev version (set at build time)
	// go build -ldflags="-X github.com/yourusername/kudev/pkg/version.Version=v0.1.0"
	Version = "v0.1.0-dev"

	// GitCommit is the git commit hash (set at build time)
	GitCommit = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = fmt.Sprintf("Go %s", runtime.Version())

	// OS is the operating system
	OS = runtime.GOOS

	// Arch is the CPU architecture
	Arch = runtime.GOARCH
)
```

---

## Implementation: cmd/init.go

Create `cmd/commands/init.go`:

```go
package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/kudev/pkg/config"
	"github.com/yourusername/kudev/pkg/logging"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize kudev configuration",
	Long: `Initialize a new .kudev.yaml configuration file.

This command guides you through setup:
  - Project name (used as deployment name)
  - Dockerfile path
  - Kubernetes namespace
  - Container ports

The configuration is saved to .kudev.yaml in the current directory.

Examples:
  kudev init                  Interactive mode
  kudev init my-app           Create config for 'my-app'
  kudev init my-app --namespace production
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.Get()

		var appName string
		if len(args) > 0 {
			appName = args[0]
		}

		// Start interactive setup
		cfg, err := interactiveSetup(appName)
		if err != nil {
			return err
		}

		// Validate before saving
		if err := cfg.Validate(cmd.Context()); err != nil {
			return err
		}

		// Save to file
		configPath := ".kudev.yaml"
		loader := config.NewFileConfigLoader("", "", "")

		if err := loader.Save(cmd.Context(), cfg, configPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		logger.Info(
			"configuration file created successfully",
			"path", configPath,
		)

		fmt.Printf("\n✓ Configuration saved to %s\n", configPath)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Review the configuration: cat %s\n", configPath)
		fmt.Printf("  2. Validate the configuration: kudev validate\n")
		fmt.Printf("  3. Deploy to Kubernetes: kudev up\n")

		return nil
	},
}

// interactiveSetup guides user through configuration creation.
func interactiveSetup(appName string) (*config.DeploymentConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nKudev Configuration Setup")
	fmt.Println("=" * 40)

	// App name
	if appName == "" {
		fmt.Print("\nProject name (e.g., my-app): ")
		name, _ := reader.ReadString('\n')
		appName = strings.TrimSpace(name)
	}

	if appName == "" {
		return nil, fmt.Errorf("project name is required")
	}

	// Dockerfile path
	fmt.Print("Dockerfile path [./Dockerfile]: ")
	dockerfilePath, _ := reader.ReadString('\n')
	dockerfilePath = strings.TrimSpace(dockerfilePath)
	if dockerfilePath == "" {
		dockerfilePath = "./Dockerfile"
	}

	// Namespace
	fmt.Print("Kubernetes namespace [default]: ")
	namespace, _ := reader.ReadString('\n')
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = "default"
	}

	// Replicas
	fmt.Print("Number of replicas [1]: ")
	replicasStr, _ := reader.ReadString('\n')
	replicasStr = strings.TrimSpace(replicasStr)
	replicas := int32(1)
	if replicasStr != "" {
		if r, err := strconv.ParseInt(replicasStr, 10, 32); err == nil {
			replicas = int32(r)
		}
	}

	// Service port
	fmt.Print("Container port [8080]: ")
	servicePortStr, _ := reader.ReadString('\n')
	servicePortStr = strings.TrimSpace(servicePortStr)
	servicePort := int32(8080)
	if servicePortStr != "" {
		if p, err := strconv.ParseInt(servicePortStr, 10, 32); err == nil {
			servicePort = int32(p)
		}
	}

	// Local port
	fmt.Print("Local port for forwarding [8080]: ")
	localPortStr, _ := reader.ReadString('\n')
	localPortStr = strings.TrimSpace(localPortStr)
	localPort := int32(8080)
	if localPortStr != "" {
		if p, err := strconv.ParseInt(localPortStr, 10, 32); err == nil {
			localPort = int32(p)
		}
	}

	// Build config
	cfg := &config.DeploymentConfig{
		APIVersion: "kudev.io/v1alpha1",
		Kind:       "DeploymentConfig",
		Metadata: config.ConfigMetadata{
			Name: appName,
		},
		Spec: config.DeploymentSpec{
			ImageName:      appName,
			DockerfilePath: dockerfilePath,
			Namespace:      namespace,
			Replicas:       replicas,
			LocalPort:      localPort,
			ServicePort:    servicePort,
		},
	}

	config.ApplyDefaults(cfg)

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("Configuration Summary:")
	fmt.Printf("  Project: %s\n", cfg.Metadata.Name)
	fmt.Printf("  Dockerfile: %s\n", cfg.Spec.DockerfilePath)
	fmt.Printf("  Namespace: %s\n", cfg.Spec.Namespace)
	fmt.Printf("  Replicas: %d\n", cfg.Spec.Replicas)
	fmt.Printf("  Service Port: %d\n", cfg.Spec.ServicePort)
	fmt.Printf("  Local Port: %d\n", cfg.Spec.LocalPort)
	fmt.Println(strings.Repeat("=", 40))

	return cfg, nil
}
```

---

## Implementation: cmd/validate.go

Create `cmd/commands/validate.go`:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourusername/kudev/pkg/logging"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long: `Validate the .kudev.yaml configuration.

Checks:
  - File exists and is valid YAML
  - All required fields are present
  - All values are in valid ranges
  - Dockerfile exists
  - Kubernetes context is safe

Examples:
  kudev validate              Validate .kudev.yaml in current dir
  kudev validate --config dev.yaml  Validate specific config
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.Get()

		// Config is already loaded in PersistentPreRun
		cfg := GetLoadedConfig()

		if cfg == nil {
			return fmt.Errorf("no configuration loaded")
		}

		logger.Info("configuration loaded successfully")
		fmt.Printf("Configuration is valid ✓\n\n")

		// Print summary
		fmt.Printf("Project: %s\n", cfg.Metadata.Name)
		fmt.Printf("Image: %s\n", cfg.Spec.ImageName)
		fmt.Printf("Dockerfile: %s\n", cfg.Spec.DockerfilePath)
		fmt.Printf("Namespace: %s\n", cfg.Spec.Namespace)
		fmt.Printf("Replicas: %d\n", cfg.Spec.Replicas)
		fmt.Printf("Service Port: %d\n", cfg.Spec.ServicePort)
		fmt.Printf("Local Port: %d\n", cfg.Spec.LocalPort)

		if len(cfg.Spec.Env) > 0 {
			fmt.Printf("Environment Variables:\n")
			for _, env := range cfg.Spec.Env {
				fmt.Printf("  - %s=%s\n", env.Name, env.Value)
			}
		}

		return nil
	},
}
```

---

## Key Design Patterns

### Pattern 1: PersistentPreRunE for Global Setup

```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
    // This runs for all commands (depth-first)
    // Perfect for:
    //   - Loading config
    //   - Validating context
    //   - Setting up logger
    //   - Shared state initialization
}
```

### Pattern 2: Skip Init for Certain Commands

```go
if cmd.Name() == "version" || cmd.Name() == "init" {
    return nil  // Skip config loading
}
```

### Pattern 3: Share State Between Commands

```go
// In PersistentPreRun
loadedConfig = cfg

// In subcommand
cfg := GetLoadedConfig()  // Access shared state
```

---

## Cobra Best Practices

1. **Use RunE, not Run**
   - RunE returns errors (better error handling)
   - Run ignores errors

2. **Put Logic in pkg/, not cmd/**
   - cmd/ should be thin
   - cmd/ delegates to pkg/
   - Easier to test pkg/ independently

3. **Use Cobra Structs**
   - cmd.Context() → context.Context
   - cmd.Flag() → access flags
   - cmd.Root() → access root command

4. **Error Messages**
   - Be specific
   - Include next steps
   - Use logger for debug info

---

## Checklist for Task 1.5

- [ ] Create `cmd/main.go`
- [ ] Create `cmd/commands/root.go`
- [ ] Create `cmd/commands/version.go`
- [ ] Create `cmd/commands/init.go`
- [ ] Create `cmd/commands/validate.go`
- [ ] Create `pkg/version/version.go`
- [ ] Register all commands
- [ ] Global flags work: `--config`, `--debug`, `--force-context`
- [ ] PersistentPreRun loads config
- [ ] Commands can access loaded config
- [ ] Help works: `kudev --help`, `kudev init --help`
- [ ] Build succeeds: `go build ./cmd`

---

## Testing Commands

```bash
# Build
go build -o kudev ./cmd

# Test version
./kudev version

# Test help
./kudev --help
./kudev init --help
./kudev validate --help

# Test init (creates .kudev.yaml)
./kudev init

# Test validate
./kudev validate

# Test with explicit config
./kudev --config ./dev.yaml validate

# Test debug flag
./kudev --debug validate

# Test invalid context
./kudev validate  # If current context is prod-*
# Should show error about not in whitelist

# Test force override
./kudev --force-context validate
# Should work even with unsafe context
```

---

## Next Steps

1. **Implement this task** ← You are here
2. **Task 1.6** → Integration testing and documentation
3. **Phase 2** → Add `up` command (build + deploy)



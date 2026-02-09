package commands

import (
	"context"
	"fmt"

	"github.com/nanaki-93/kudev/pkg/config"
	"github.com/nanaki-93/kudev/pkg/kubeconfig"
	"github.com/nanaki-93/kudev/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	configPath   string
	debugLogging bool
	forceContext bool

	loadedConfig *config.DeploymentConfig

	validator *kubeconfig.ContextValidator

	rootCmd = &cobra.Command{
		Use:   "kudev",
		Short: "Kubernetes development helper",
		Long: `Kudev streamlines local Kubernetes development with automatic building, deploying, and live-reloading

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
  https://github.com/nanaki-93/kudev
`,
		PersistentPreRunE: rootPersistentPreRun,
		SilenceUsage:      true,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to the configuration file.")
	rootCmd.PersistentFlags().BoolVar(&debugLogging, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&forceContext, "force-context", false, "Skip K8s context safety check (use with caution!)")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(validateCmd)
}

// rootPersistentPreRun is the global initialization hook.
//
// This runs before any command execution and performs:
//  1. Setup logging
//  2. Load configuration (unless command is 'init')
//  3. Validate context safety
//  4. Store for use by subcommands
func rootPersistentPreRun(cmd *cobra.Command, args []string) error {
	// Step 1: Setup logging
	logging.InitLogger(debugLogging)

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
		return nil // Let Cobra handle help
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
		return err // Error already formatted by validator
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

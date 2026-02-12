package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nanaki-93/kudev/pkg/config"
	kudevErrors "github.com/nanaki-93/kudev/pkg/errors"
	"github.com/nanaki-93/kudev/pkg/kubeconfig"
	"github.com/nanaki-93/kudev/pkg/logging"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var rootCmd = &cobra.Command{
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
	SilenceErrors:     true,
}

var (
	configPath   string
	debugMode    bool
	forceContext bool
	logger       logging.LoggerInterface
	loadedConfig *config.DeploymentConfig
	validator    *kubeconfig.ContextValidator
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&forceContext, "force-context", false, "Skip K8s context safety check (use with caution!)")
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
	logging.InitLogger(debugMode)

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
func getLoadedConfig() *config.DeploymentConfig {
	return loadedConfig
}

// GetValidator returns the context validator.
func GetValidator() *kubeconfig.ContextValidator {
	return validator
}

// Execute runs the root command.
// This is called from main().
func Execute() int {
	// Create context that cancels on SIGINT/SIGTERM
	ctx := setupSignalContext()

	err := rootCmd.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	// Pass context to all commands
	return handleError(err)
}
func setupSignalContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Handle signals in goroutine
	go func() {
		sig := <-sigChan
		fmt.Println() // New line after ^C
		logger.Debug("received signal", "signal", sig)
		cancel()

		// If second signal, force exit
		sig = <-sigChan
		fmt.Println("\nForce exit...")
		os.Exit(1)
	}()
	return ctx
}

func handleError(err error) int {
	// Check if it's a kudev error
	var kerr kudevErrors.KudevError
	if errors.As(err, &kerr) {
		printKudevError(kerr)
		return kerr.ExitCode()
	}

	// Generic error
	fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
	return 1
}

// printKudevError prints a formatted kudev error.
func printKudevError(err kudevErrors.KudevError) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "❌ Error: %s\n", err.UserMessage())

	if suggestion := err.SuggestedAction(); suggestion != "" {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "💡 Suggestion: %s\n", suggestion)
	}

	fmt.Fprintln(os.Stderr)
}
func getKubernetesClient() (kubernetes.Interface, *rest.Config, error) {
	// Load kubeconfig from default location (~/.kube/config)
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, restConfig, nil
}

func getCurrentContext() string {
	currContext, err := kubeconfig.LoadCurrentContext()
	if err != nil {
		//fixme should i panic?
		panic("failed to load current context: " + err.Error() + "")
	}
	return currContext.Name

}

package commands

import (
	"fmt"

	"github.com/nanaki-93/kudev/pkg/logging"
	"github.com/spf13/cobra"
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

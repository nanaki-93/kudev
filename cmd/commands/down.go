package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nanaki-93/kudev/pkg/deployer"
	"github.com/nanaki-93/kudev/templates"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Remove application from Kubernetes",
	Long: `Remove application from Kubernetes.

This command:
1. Deletes the Deployment
2. Deletes the Service
3. Waits for pods to terminate`,
	RunE: runDown,
}

var (
	forceDelete bool
)

func init() {
	downCmd.Flags().BoolVar(&forceDelete, "force", false, "Force delete without confirmation")

	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. Load configuration
	fmt.Println("Loading configuration...")
	cfg := getLoadedConfig()

	// 2. Confirm deletion (unless --force)
	if !forceDelete {
		fmt.Printf("This will delete deployment '%s' in namespace '%s'\n",
			cfg.Metadata.Name, cfg.Spec.Namespace)
		fmt.Print("Continue? [y/N]: ")

		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// 3. Delete resources
	fmt.Println("Deleting resources...")

	clientset, _, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	renderer, _ := deployer.NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)
	dep := deployer.NewKubernetesDeployer(clientset, renderer, logger)

	if err := dep.Delete(ctx, cfg.Metadata.Name, cfg.Spec.Namespace); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Deployment deleted")
	fmt.Println("✓ Service deleted")
	fmt.Println()
	fmt.Printf("Application '%s' has been removed from namespace '%s'\n",
		cfg.Metadata.Name, cfg.Spec.Namespace)

	return nil
}

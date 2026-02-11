// cmd/commands/status.go

package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/nanaki-93/kudev/pkg/deployer"
	"github.com/nanaki-93/kudev/templates"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show deployment status",
	Long:  `Show the current status of the deployed application.`,
	RunE:  runStatus,
}

var (
	watchStatus bool
)

func init() {
	statusCmd.Flags().BoolVarP(&watchStatus, "watch", "w", false, "Watch status continuously")

	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. Load configuration
	cfg := getLoadedConfig()

	// 2. Get K8s client
	clientset, _, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	renderer, _ := deployer.NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)
	dep := deployer.NewKubernetesDeployer(clientset, renderer, logger)

	// 3. Print status
	printStatus := func() error {
		status, err := dep.Status(ctx, cfg.Metadata.Name, cfg.Spec.Namespace)
		if err != nil {
			return err
		}

		// Clear screen if watching
		if watchStatus {
			fmt.Print("\033[H\033[2J")
		}

		fmt.Println("═══════════════════════════════════════════════════")
		fmt.Printf("  Deployment: %s\n", status.DeploymentName)
		fmt.Printf("  Namespace:  %s\n", status.Namespace)
		fmt.Printf("  Status:     %s\n", colorStatus(status.Status))
		fmt.Printf("  Replicas:   %d/%d ready\n", status.ReadyReplicas, status.DesiredReplicas)
		if status.ImageHash != "" {
			fmt.Printf("  Version:    %s\n", status.ImageHash)
		}
		fmt.Println("═══════════════════════════════════════════════════")

		if len(status.Pods) > 0 {
			fmt.Println()
			fmt.Println("Pods:")
			for _, pod := range status.Pods {
				ready := "○"
				if pod.Ready {
					ready = "●"
				}
				fmt.Printf("  %s %s (%s, restarts: %d)\n",
					ready, pod.Name, pod.Status, pod.Restarts)
			}
		}

		if status.Message != "" {
			fmt.Println()
			fmt.Println(status.Message)
		}

		return nil
	}

	// Initial status
	if err := printStatus(); err != nil {
		return err
	}

	// Watch mode
	if watchStatus {
		fmt.Println()
		fmt.Println("Watching for changes (Ctrl+C to stop)...")

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if err := printStatus(); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			}
		}
	}

	return nil
}

func colorStatus(status string) string {
	switch status {
	case "Running":
		return "\033[32m" + status + "\033[0m" // Green
	case "Pending":
		return "\033[33m" + status + "\033[0m" // Yellow
	case "Degraded", "Failed":
		return "\033[31m" + status + "\033[0m" // Red
	default:
		return status
	}
}

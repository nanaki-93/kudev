// cmd/commands/watch.go

package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nanaki-93/kudev/pkg/builder"
	"github.com/nanaki-93/kudev/pkg/builder/docker"
	"github.com/nanaki-93/kudev/pkg/deployer"
	"github.com/nanaki-93/kudev/pkg/hash"
	"github.com/nanaki-93/kudev/pkg/logs"
	"github.com/nanaki-93/kudev/pkg/portfwd"
	"github.com/nanaki-93/kudev/pkg/registry"
	"github.com/nanaki-93/kudev/pkg/watch"
	"github.com/nanaki-93/kudev/templates"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for changes and auto-rebuild",
	Long: `Watch for file changes and automatically rebuild and redeploy.

This command:
1. Does an initial build and deploy
2. Starts port forwarding
3. Watches for file changes
4. Automatically rebuilds and redeploys on changes
5. Shows logs from the running application

Press Ctrl+C to stop watching and exit.`,
	RunE: runWatch,
}

var (
	watchNoLogs    bool
	watchNoPortFwd bool
)

func init() {
	watchCmd.Flags().BoolVar(&watchNoLogs, "no-logs", false, "Don't stream logs")
	watchCmd.Flags().BoolVar(&watchNoPortFwd, "no-port-forward", false, "Don't start port forwarding")

	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. Load configuration
	fmt.Println("✓ Loading configuration...")
	cfg := loadedConfig
	projectRoot := cfg.ProjectRoot

	// 2. Get Kubernetes client
	clientset, restConfig, err := getKubernetesClient()

	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	// 3. Create components
	dockerBuilder := docker.NewBuilder(logger)

	renderer, _ := deployer.NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)
	dep := deployer.NewKubernetesDeployer(clientset, renderer, logger)

	kubeContext := cfg.Spec.KubeContext
	if kubeContext == "" {
		kubeContext = getCurrentContext()
	}
	reg := registry.NewRegistry(kubeContext, logger)

	// 4. Do initial build and deploy
	fmt.Println("✓ Doing initial build and deploy...")

	calculator := hash.NewCalculator(projectRoot, cfg.Spec.BuildContextExclusions)
	tagger := builder.NewTagger(calculator)
	tag, err := tagger.GenerateTag(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to generate tag: %w", err)
	}

	opts := builder.BuildOptions{
		SourceDir:      projectRoot,
		DockerfilePath: cfg.Spec.DockerfilePath,
		ImageName:      cfg.Spec.ImageName,
		ImageTag:       tag,
	}

	imageRef, err := dockerBuilder.Build(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}

	if err := reg.Load(ctx, imageRef.FullRef); err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}

	imageHash, _ := tagger.GetHash(ctx)
	deployOpts := deployer.DeploymentOptions{
		Config:    cfg,
		ImageRef:  imageRef.FullRef,
		ImageHash: imageHash,
	}

	status, err := dep.Upsert(ctx, deployOpts)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}

	fmt.Printf("✓ Deployed: %s (%d/%d replicas)\n", status.Status, status.ReadyReplicas, status.DesiredReplicas)

	// 5. Start port forwarding (if enabled)
	var forwarder portfwd.PortForwarder
	if !watchNoPortFwd {
		fmt.Printf("✓ Port forwarding localhost:%d → pod:%d\n",
			cfg.Spec.LocalPort, cfg.Spec.ServicePort)

		forwarder = portfwd.NewKubernetesPortForwarder(clientset, restConfig, logger)
		if err := forwarder.Forward(ctx, cfg.Metadata.Name, cfg.Spec.Namespace,
			cfg.Spec.LocalPort, cfg.Spec.ServicePort); err != nil {
			fmt.Printf("⚠ Port forwarding failed: %v\n", err)
		}
		defer forwarder.Stop()
	}

	// 6. Start log streaming in background (if enabled)
	if !watchNoLogs {
		go func() {
			tailer := logs.NewKubernetesLogTailer(clientset, logger, os.Stdout)
			tailer.TailLogsWithRetry(ctx, cfg.Metadata.Name, cfg.Spec.Namespace)
		}()
	}

	// 7. Print ready message
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("  Application is running!\n")
	fmt.Printf("  Local:   http://localhost:%d\n", cfg.Spec.LocalPort)
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()

	// 8. Create and run orchestrator
	orchestrator, err := watch.NewOrchestrator(watch.OrchestratorConfig{
		Config:   cfg,
		Builder:  dockerBuilder,
		Deployer: dep,
		Registry: reg,
		Logger:   logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}
	defer orchestrator.Close()

	// Run until cancelled
	if err := orchestrator.Run(ctx); err != nil && err != context.Canceled {
		return err
	}

	fmt.Println("\nShutting down...")
	return nil
}

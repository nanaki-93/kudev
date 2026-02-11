package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/nanaki-93/kudev/pkg/builder"
	"github.com/nanaki-93/kudev/pkg/builder/docker"
	"github.com/nanaki-93/kudev/pkg/deployer"
	"github.com/nanaki-93/kudev/pkg/hash"
	"github.com/nanaki-93/kudev/pkg/logs"
	"github.com/nanaki-93/kudev/pkg/portfwd"
	"github.com/nanaki-93/kudev/pkg/registry"
	"github.com/nanaki-93/kudev/templates"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Build and deploy application to Kubernetes",
	Long: `Build and deploy application to Kubernetes.

This command:
1. Builds a Docker image from your source code
2. Loads the image to your local Kubernetes cluster
3. Deploys or updates the Deployment and Service
4. Forwards a local port to the pod
5. Streams pod logs to your terminal

Press Ctrl+C to stop log streaming and port forwarding.
The deployment will remain running.`,
	RunE: runUp,
}

var (
	noLogs    bool
	noPortFwd bool
	noBuild   bool
)

func init() {
	upCmd.Flags().BoolVar(&noLogs, "no-logs", false, "Don't stream logs after deployment")
	upCmd.Flags().BoolVar(&noPortFwd, "no-port-forward", false, "Don't start port forwarding")
	upCmd.Flags().BoolVar(&noBuild, "no-build", false, "Skip build step (use existing image)")

	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create cleanup list
	var cleanups []func()
	defer func() {
		fmt.Println("\nCleaning up...")
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	// 1. Load configuration
	fmt.Println("✓ Loading configuration...")
	cfg := getLoadedConfig()

	projectRoot := cfg.ProjectRoot

	var imageRef *builder.ImageRef
	var imageHash string
	var err error
	if !noBuild {
		// 2. Calculate source hash
		fmt.Println("✓ Calculating source hash...")
		calculator := hash.NewCalculator(projectRoot, cfg.Spec.BuildContextExclusions)
		imageHash, err = calculator.Calculate(ctx)
		if err != nil {
			return fmt.Errorf("failed to calculate hash: %w", err)
		}

		// 3. Generate image tag
		tagger := builder.NewTagger(calculator)
		tag, err := tagger.GenerateTag(ctx, false)
		if err != nil {
			return fmt.Errorf("failed to generate tag: %w", err)
		}

		// 4. Build image
		fmt.Printf("✓ Building image %s:%s...\n", cfg.Spec.ImageName, tag)
		dockerBuilder := docker.NewBuilder(logger)
		opts := builder.BuildOptions{
			SourceDir:      projectRoot,
			DockerfilePath: cfg.Spec.DockerfilePath,
			ImageName:      cfg.Spec.ImageName,
			ImageTag:       tag,
		}

		imageRef, err = dockerBuilder.Build(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to build image: %w", err)
		}

		// 5. Load image to cluster
		fmt.Println("✓ Loading image to cluster...")
		kubeContext := cfg.Spec.KubeContext
		if kubeContext == "" {
			kubeContext = getCurrentContext()
		}
		reg := registry.NewRegistry(kubeContext, logger)
		if err := reg.Load(ctx, imageRef.FullRef); err != nil {
			return fmt.Errorf("failed to load image: %w", err)
		}
	} else {
		// Use existing image
		imageRef = &builder.ImageRef{
			FullRef: fmt.Sprintf("%s:latest", cfg.Spec.ImageName),
		}
		imageHash = "manual"
	}

	// 6. Deploy to Kubernetes
	fmt.Println("✓ Deploying to Kubernetes...")
	clientset, restConfig, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}
	renderer, _ := deployer.NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)
	dep := deployer.NewKubernetesDeployer(clientset, renderer, logger)

	deployOpts := deployer.DeploymentOptions{
		Config:    cfg,
		ImageRef:  imageRef.FullRef,
		ImageHash: imageHash,
	}

	status, err := dep.Upsert(ctx, deployOpts)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}

	// 7. Wait for deployment to be ready
	fmt.Println("✓ Waiting for pods to be ready...")
	if err := dep.WaitForReady(ctx, cfg.Metadata.Name, cfg.Spec.Namespace, 5*time.Minute); err != nil {
		return fmt.Errorf("deployment not ready: %w", err)
	}

	// 8. Start port forwarding (if enabled)
	var forwarder portfwd.PortForwarder
	if !noPortFwd {
		fmt.Printf("✓ Port forwarding localhost:%d → pod:%d\n",
			cfg.Spec.LocalPort, cfg.Spec.ServicePort)

		forwarder = portfwd.NewKubernetesPortForwarder(clientset, restConfig, logger)
		if err := forwarder.Forward(ctx, cfg.Metadata.Name, cfg.Spec.Namespace,
			cfg.Spec.LocalPort, cfg.Spec.ServicePort); err != nil {
			fmt.Printf("⚠ Port forwarding failed: %v\n", err)
			// Continue anyway - user can forward manually
			//fixme return error or not?
		}
		cleanups = append(cleanups, func() {
			forwarder.Stop()
			fmt.Println("✓ Port forward stopped")
		})
	}

	// Print success message
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("  Application is running!\n")
	fmt.Printf("  Local:   http://localhost:%d\n", cfg.Spec.LocalPort)
	fmt.Printf("  Status:  %s (%d/%d replicas)\n", status.Status, status.ReadyReplicas, status.DesiredReplicas)
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()

	// 9. Stream logs (if enabled)
	if !noLogs {
		fmt.Println("Streaming logs (Ctrl+C to stop)...")
		fmt.Println()

		tailer := logs.NewKubernetesLogTailer(clientset, logger, os.Stdout)
		if err := tailer.TailLogsWithRetry(ctx, cfg.Metadata.Name, cfg.Spec.Namespace); err != nil {
			if !errors.Is(err, context.Canceled) {
				fmt.Printf("Log streaming ended: %v\n", err)
			}
		}
	} else {
		fmt.Println("Press Ctrl+C to stop port forwarding...")
		<-ctx.Done()
	}

	// Cleanup
	if forwarder != nil {
		forwarder.Stop()
	}

	fmt.Println("\nShutting down...")
	fmt.Println("✓ Port forward stopped")
	fmt.Println("✓ Deployment remains running (use 'kudev down' to remove)")

	return nil
}

# Task 4.3: Integrate CLI Commands

## Overview

This task implements the **CLI command orchestration** that ties together all previous phases into cohesive user commands.

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate  
**Dependencies**: All previous tasks  
**Files to Create**:
- `cmd/commands/up.go` — The `kudev up` command
- `cmd/commands/down.go` — The `kudev down` command
- `cmd/commands/status.go` — The `kudev status` command

---

## What You're Building

CLI commands that:
1. **`kudev up`** — Build, deploy, stream logs, forward ports
2. **`kudev down`** — Clean deletion of deployment
3. **`kudev status`** — Show deployment health

---

## Complete Implementation

### Up Command

```go
// cmd/commands/up.go

package commands

import (
    "context"
    "fmt"
    "os"
    "time"
    
    "github.com/spf13/cobra"
    
    "github.com/your-org/kudev/pkg/builder"
    "github.com/your-org/kudev/pkg/builder/docker"
    "github.com/your-org/kudev/pkg/config"
    "github.com/your-org/kudev/pkg/deployer"
    "github.com/your-org/kudev/pkg/hash"
    "github.com/your-org/kudev/pkg/logs"
    "github.com/your-org/kudev/pkg/portfwd"
    "github.com/your-org/kudev/pkg/registry"
    "github.com/your-org/kudev/templates"
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
    noLogs      bool
    noPortFwd   bool
    noBuild     bool
)

func init() {
    upCmd.Flags().BoolVar(&noLogs, "no-logs", false, "Don't stream logs after deployment")
    upCmd.Flags().BoolVar(&noPortFwd, "no-port-forward", false, "Don't start port forwarding")
    upCmd.Flags().BoolVar(&noBuild, "no-build", false, "Skip build step (use existing image)")
    
    rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    
    // 1. Load configuration
    fmt.Println("✓ Loading configuration...")
    cfg, err := loadConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    
    projectRoot := cfg.ProjectRoot()
    
    var imageRef *builder.ImageRef
    var imageHash string
    
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
        dockerBuilder := docker.NewDockerBuilder(logger)
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
    clientset, restConfig := getKubernetesClient()
    
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
        }
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
            if err != context.Canceled {
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
```

### Down Command

```go
// cmd/commands/down.go

package commands

import (
    "fmt"
    
    "github.com/spf13/cobra"
    
    "github.com/your-org/kudev/pkg/deployer"
    "github.com/your-org/kudev/templates"
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
    cfg, err := loadConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    
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
    
    clientset, _ := getKubernetesClient()
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
```

### Status Command

```go
// cmd/commands/status.go

package commands

import (
    "fmt"
    "strings"
    "time"
    
    "github.com/spf13/cobra"
    
    "github.com/your-org/kudev/pkg/deployer"
    "github.com/your-org/kudev/templates"
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
    cfg, err := loadConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    
    // 2. Get K8s client
    clientset, _ := getKubernetesClient()
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
```

---

## Command Flow Diagram

```
kudev up
│
├─→ Load Config (.kudev.yaml)
│
├─→ Calculate Source Hash
│
├─→ Build Docker Image
│       └─→ docker build -t myapp:kudev-abc12345 .
│
├─→ Load Image to Cluster
│       ├─→ Docker Desktop: (automatic)
│       ├─→ Minikube: minikube image load
│       └─→ Kind: kind load docker-image
│
├─→ Deploy to Kubernetes
│       ├─→ Render templates
│       ├─→ Upsert Deployment
│       └─→ Upsert Service
│
├─→ Wait for Ready
│       └─→ Poll deployment status
│
├─→ Start Port Forward (background)
│       └─→ localhost:8080 → pod:8080
│
├─→ Stream Logs (foreground)
│       └─→ Follow pod logs
│
└─→ Ctrl+C → Cleanup
```

---

## Checklist for Task 4.3

- [ ] Create `cmd/commands/up.go`
- [ ] Implement `kudev up` command
- [ ] Add `--no-logs`, `--no-port-forward`, `--no-build` flags
- [ ] Create `cmd/commands/down.go`
- [ ] Implement `kudev down` command
- [ ] Add `--force` flag for confirmation skip
- [ ] Create `cmd/commands/status.go`
- [ ] Implement `kudev status` command
- [ ] Add `--watch` flag for continuous monitoring
- [ ] Add color coding to status output
- [ ] Print success/error messages clearly
- [ ] Run `go build ./cmd/...`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Not waiting for ready before port forward
```go
// Wrong - pod might not be ready
dep.Upsert(ctx, opts)
forwarder.Forward(ctx, ...)  // Fails!

// Right - wait first
dep.Upsert(ctx, opts)
dep.WaitForReady(ctx, name, ns, timeout)
forwarder.Forward(ctx, ...)
```

❌ **Mistake 2**: Not cleaning up on error
```go
// Wrong - forwarder keeps running
forwarder.Forward(ctx, ...)
// ... error occurs
return err  // Forwarder still running!

// Right - defer cleanup
defer func() {
    if forwarder != nil {
        forwarder.Stop()
    }
}()
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 4.4** → Implement Graceful Shutdown

---

## References

- [Cobra Documentation](https://cobra.dev/)
- [Go Context](https://pkg.go.dev/context)


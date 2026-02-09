# Task 5.4: Implement Watch CLI Command

## Overview

This task implements the **`kudev watch` command** that enables hot-reload development.

**Effort**: ~2 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: Tasks 5.1-5.3  
**Files to Create**:
- `cmd/commands/watch.go` — Watch command

---

## What You're Building

A CLI command that:
1. **Runs** initial build and deploy
2. **Starts** file watcher
3. **Triggers** rebuilds on changes
4. **Provides** clear status feedback
5. **Handles** Ctrl+C gracefully

---

## Complete Implementation

```go
// cmd/commands/watch.go

package commands

import (
    "context"
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    
    "github.com/your-org/kudev/pkg/builder"
    "github.com/your-org/kudev/pkg/builder/docker"
    "github.com/your-org/kudev/pkg/config"
    "github.com/your-org/kudev/pkg/deployer"
    "github.com/your-org/kudev/pkg/hash"
    "github.com/your-org/kudev/pkg/logs"
    "github.com/your-org/kudev/pkg/portfwd"
    "github.com/your-org/kudev/pkg/registry"
    "github.com/your-org/kudev/pkg/watch"
    "github.com/your-org/kudev/templates"
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
    cfg, err := loadConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    
    projectRoot := cfg.ProjectRoot()
    
    // 2. Get Kubernetes client
    clientset, restConfig := getKubernetesClient()
    
    // 3. Create components
    dockerBuilder := docker.NewDockerBuilder(logger)
    
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
```

---

## User Experience

```bash
$ kudev watch
✓ Loading configuration...
✓ Doing initial build and deploy...
✓ Building myapp:kudev-abc12345...
✓ Loading image to cluster...
✓ Deployed: Running (2/2 replicas)
✓ Port forwarding localhost:8080 → pod:8080

═══════════════════════════════════════════════════
  Application is running!
  Local:   http://localhost:8080
═══════════════════════════════════════════════════

Watching for changes...
Press Ctrl+C to stop

[2024-01-15 10:30:45] Starting server on :8080
[2024-01-15 10:30:46] Ready to accept connections

# User edits main.go...

═══════════════════════════════════════════════════
  Change detected! Rebuilding...
═══════════════════════════════════════════════════

Building myapp:kudev-def67890...
Loading image to cluster...
Deploying...

═══════════════════════════════════════════════════
  ✓ Rebuild complete in 5.2s
  Status: Running (2/2 replicas)
═══════════════════════════════════════════════════

Watching for changes...

[2024-01-15 10:31:52] Restarting...
[2024-01-15 10:31:53] Starting server on :8080

^C
Shutting down...
```

---

## Comparison: up vs watch

| Feature | `kudev up` | `kudev watch` |
|---------|-----------|---------------|
| Initial build | ✅ | ✅ |
| Deploy | ✅ | ✅ |
| Port forward | ✅ | ✅ |
| Stream logs | ✅ (foreground) | ✅ (background) |
| Watch files | ❌ | ✅ |
| Auto-rebuild | ❌ | ✅ |
| Blocking | On logs | On watcher |

---

## Checklist for Task 5.4

- [ ] Create `cmd/commands/watch.go`
- [ ] Define `watchCmd` with Cobra
- [ ] Add `--no-logs` flag
- [ ] Add `--no-port-forward` flag
- [ ] Implement initial build and deploy
- [ ] Start port forwarding
- [ ] Start log streaming in background
- [ ] Create and run orchestrator
- [ ] Handle Ctrl+C gracefully
- [ ] Print user-friendly status messages
- [ ] Add to root command
- [ ] Test: `go build ./cmd/... && ./kudev watch`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Blocking on logs
```go
// Wrong - blocks forever
tailer.TailLogs(ctx, ...)
orchestrator.Run(ctx)  // Never reached!

// Right - logs in background
go tailer.TailLogs(ctx, ...)
orchestrator.Run(ctx)  // This is the main blocker
```

❌ **Mistake 2**: Not cleaning up forwarder
```go
// Wrong - forwarder keeps running
forwarder.Forward(ctx, ...)
// No cleanup!

// Right
defer forwarder.Stop()
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Phase 5 is now complete! 🎉
3. Move to **Phase 6** → Testing & Reliability

---

## References

- [Cobra Command](https://cobra.dev/)
- [Context Cancellation](https://pkg.go.dev/context)


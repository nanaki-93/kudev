# Task 4.4: Implement Graceful Shutdown

## Overview

This task implements **graceful shutdown handling** that properly cleans up resources when the user presses Ctrl+C.

**Effort**: ~2 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: Task 4.3 (CLI Commands)  
**Files to Modify**:
- `cmd/commands/root.go` — Add signal handling
- `cmd/main.go` — Wire up context

---

## What You're Building

Signal handling that:
1. **Catches** SIGINT (Ctrl+C) and SIGTERM
2. **Propagates** cancellation via context
3. **Cleans up** port forwarding, log streaming
4. **Prints** friendly shutdown message
5. **Exits** with appropriate code

---

## Complete Implementation

### Signal Handling in Root Command

```go
// cmd/commands/root.go

package commands

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/spf13/cobra"
    
    "github.com/your-org/kudev/pkg/config"
    "github.com/your-org/kudev/pkg/logging"
)

var rootCmd = &cobra.Command{
    Use:   "kudev",
    Short: "Kubernetes development made easy",
    Long: `kudev is a CLI tool for rapid Kubernetes development.

It builds, deploys, and manages your application in a local Kubernetes cluster
with a single command.`,
    SilenceUsage:  true,  // Don't print usage on error
    SilenceErrors: true,  // We'll handle errors ourselves
}

var (
    configPath  string
    debugMode   bool
    logger      logging.Logger
)

func init() {
    rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Config file path")
    rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug logging")
    
    // Initialize in PersistentPreRun
    rootCmd.PersistentPreRunE = preRun
}

func preRun(cmd *cobra.Command, args []string) error {
    // Initialize logger
    logger = logging.NewLogger(debugMode)
    return nil
}

// Execute runs the root command with signal handling.
func Execute() error {
    // Create context that cancels on SIGINT/SIGTERM
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
    
    // Pass context to all commands
    return rootCmd.ExecuteContext(ctx)
}

// Helper functions used by subcommands

func loadConfig() (*config.DeploymentConfig, error) {
    loader := config.NewFileConfigLoader()
    
    if configPath != "" {
        return loader.LoadFromPath(configPath)
    }
    
    return loader.Load()
}

func getKubernetesClient() (kubernetes.Interface, *rest.Config) {
    // Implementation here
    // Returns clientset and rest config
}

func getCurrentContext() string {
    // Get current kubectl context
}
```

### Main Entry Point

```go
// cmd/main.go

package main

import (
    "fmt"
    "os"
    
    "github.com/your-org/kudev/cmd/commands"
)

func main() {
    if err := commands.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Graceful Cleanup Pattern

```go
// Pattern for commands that need cleanup

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
    
    // ... build and deploy ...
    
    // Start port forward
    forwarder := portfwd.NewKubernetesPortForwarder(clientset, restConfig, logger)
    if err := forwarder.Forward(ctx, ...); err != nil {
        return err
    }
    cleanups = append(cleanups, func() {
        forwarder.Stop()
        fmt.Println("✓ Port forward stopped")
    })
    
    // Start log tailing (blocks until ctx cancelled)
    tailer := logs.NewKubernetesLogTailer(clientset, logger, os.Stdout)
    err := tailer.TailLogsWithRetry(ctx, appName, namespace)
    
    // Context cancelled is expected
    if err == context.Canceled {
        return nil
    }
    
    return err
}
```

---

## Key Implementation Details

### 1. Signal Channel Setup

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
```

**Why buffered channel?**
- Prevents missing signals
- Non-blocking send from OS

**Which signals?**
- `SIGINT` (2) — Ctrl+C
- `SIGTERM` (15) — `kill` command, container shutdown

### 2. Context Cancellation

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    <-sigChan
    cancel()  // This cancels the context
}()

// All operations use this context
return rootCmd.ExecuteContext(ctx)
```

### 3. Double-Signal Force Exit

```go
go func() {
    // First signal - graceful shutdown
    <-sigChan
    cancel()
    
    // Second signal - force exit
    <-sigChan
    os.Exit(1)
}()
```

### 4. Cleanup with Defer

```go
var cleanups []func()
defer func() {
    for _, fn := range cleanups {
        fn()
    }
}()

// Register cleanups as you go
cleanups = append(cleanups, func() {
    forwarder.Stop()
})
```

---

## Testing Graceful Shutdown

### Manual Test

```bash
# Start kudev up
$ kudev up

# Press Ctrl+C
^C

# Expected output:
# 
# Cleaning up...
# ✓ Port forward stopped
# ✓ Log streaming stopped
# Deployment remains running.
```

### Unit Test

```go
func TestSignalHandling(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    
    // Simulate work that respects context
    done := make(chan bool)
    go func() {
        select {
        case <-ctx.Done():
            done <- true
        case <-time.After(5 * time.Second):
            done <- false
        }
    }()
    
    // Cancel (simulates Ctrl+C)
    cancel()
    
    // Should complete quickly
    select {
    case success := <-done:
        if !success {
            t.Error("context cancellation not detected")
        }
    case <-time.After(1 * time.Second):
        t.Error("timeout waiting for cancellation")
    }
}
```

---

## Cleanup Checklist

Things that need cleanup on shutdown:

| Resource | Cleanup Action |
|----------|----------------|
| Port forward | `forwarder.Stop()` |
| Log stream | Close via context cancellation |
| HTTP clients | Close via context cancellation |
| File handles | `defer file.Close()` |
| Temp files | `defer os.Remove(path)` |

---

## Checklist for Task 4.4

- [ ] Modify `cmd/commands/root.go`
- [ ] Add signal channel setup
- [ ] Add context with cancellation
- [ ] Handle double-signal force exit
- [ ] Update `cmd/main.go` with `Execute()`
- [ ] Use `cmd.Context()` in all commands
- [ ] Add cleanup defer pattern to up command
- [ ] Print friendly shutdown messages
- [ ] Test Ctrl+C behavior
- [ ] Test double Ctrl+C force exit

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Not using buffered channel
```go
// Wrong - can miss signal
sigChan := make(chan os.Signal)

// Right - buffered
sigChan := make(chan os.Signal, 1)
```

❌ **Mistake 2**: Not checking context in loops
```go
// Wrong - never stops
for {
    doWork()
}

// Right - respects cancellation
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        doWork()
    }
}
```

❌ **Mistake 3**: Printing error for context.Canceled
```go
// Wrong - prints error on normal Ctrl+C
if err != nil {
    fmt.Println("Error:", err)
}

// Right - handle cancelled specially
if err != nil && err != context.Canceled {
    fmt.Println("Error:", err)
}
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Phase 4 is now complete! 🎉
3. Move to **Phase 5** → Live Watcher

---

## References

- [os/signal Package](https://pkg.go.dev/os/signal)
- [Context Cancellation](https://pkg.go.dev/context)
- [Graceful Shutdown Pattern](https://golang.org/doc/articles/wiki/)


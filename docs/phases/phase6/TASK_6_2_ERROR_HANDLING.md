# Task 6.2: Implement Error Interception

## Overview

This task implements **centralized error handling** in the root command to format errors consistently.

**Effort**: ~1-2 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: Task 6.1 (Custom Errors)  
**Files to Modify**:
- `cmd/commands/root.go` — Add error handler
- `cmd/main.go` — Handle exit codes

---

## What You're Building

Error handling that:
1. **Intercepts** all command errors
2. **Formats** output consistently
3. **Shows** suggestions when available
4. **Sets** correct exit codes

---

## Complete Implementation

### Root Command Error Handler

```go
// cmd/commands/root.go

package commands

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    
    kudevErrors "github.com/your-org/kudev/pkg/errors"
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

// Execute runs the root command with error handling.
func Execute() int {
    // Set up context with signal handling
    ctx := setupSignalContext()
    
    // Run command
    err := rootCmd.ExecuteContext(ctx)
    if err == nil {
        return 0
    }
    
    // Handle the error
    return handleError(err)
}

// handleError formats and prints the error, returns exit code.
func handleError(err error) int {
    // Check if it's a kudev error
    if kerr, ok := err.(kudevErrors.KudevError); ok {
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
```

### Main Entry Point

```go
// cmd/main.go

package main

import (
    "os"
    
    "github.com/your-org/kudev/cmd/commands"
)

func main() {
    exitCode := commands.Execute()
    os.Exit(exitCode)
}
```

### Using Custom Errors in Commands

```go
// Example: cmd/commands/up.go

func runUp(cmd *cobra.Command, args []string) error {
    // Load config
    cfg, err := loadConfig()
    if err != nil {
        // Wrap with custom error
        return kudevErrors.ConfigInvalid("failed to load configuration", err)
    }
    
    // Check Docker
    if err := checkDocker(); err != nil {
        return kudevErrors.DockerNotRunning(err)
    }
    
    // Build image
    imageRef, err := builder.Build(ctx, opts)
    if err != nil {
        return kudevErrors.DockerBuildFailed(err)
    }
    
    // Deploy
    if err := deployer.Upsert(ctx, deployOpts); err != nil {
        return kudevErrors.DeploymentFailed(err)
    }
    
    return nil
}
```

---

## Error Output Examples

### Config Error
```
❌ Error: Configuration file not found: .kudev.yaml

💡 Suggestion: Run 'kudev init' to create a new configuration, or specify path with --config
```

### Build Error
```
❌ Error: Docker daemon is not running

💡 Suggestion: Start Docker Desktop or run 'sudo systemctl start docker'
```

### Deploy Error
```
❌ Error: Failed to deploy to Kubernetes

💡 Suggestion: Check that your cluster is running and you have permissions
```

---

## Testing

```go
// cmd/commands/root_test.go

package commands

import (
    "bytes"
    "testing"
    
    kudevErrors "github.com/your-org/kudev/pkg/errors"
)

func TestHandleError_KudevError(t *testing.T) {
    err := kudevErrors.ConfigNotFound(".kudev.yaml")
    
    exitCode := handleError(err)
    
    if exitCode != kudevErrors.ExitConfig {
        t.Errorf("exitCode = %d, want %d", exitCode, kudevErrors.ExitConfig)
    }
}

func TestHandleError_GenericError(t *testing.T) {
    err := fmt.Errorf("some generic error")
    
    exitCode := handleError(err)
    
    if exitCode != 1 {
        t.Errorf("exitCode = %d, want 1", exitCode)
    }
}
```

---

## Checklist for Task 6.2

- [ ] Modify `cmd/commands/root.go`
- [ ] Add `SilenceUsage: true`
- [ ] Add `SilenceErrors: true`
- [ ] Implement `handleError()` function
- [ ] Implement `printKudevError()` function
- [ ] Modify `Execute()` to return int
- [ ] Modify `cmd/main.go` to use exit code
- [ ] Update commands to return custom errors
- [ ] Test error formatting
- [ ] Test exit codes

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 6.3** → Write Comprehensive Unit Tests

---

## References

- [Cobra Error Handling](https://cobra.dev/)
- [Exit Codes](https://tldp.org/LDP/abs/html/exitcodes.html)


# Task 6.5: Create Debug Command

## Overview

This task implements a **debug command** that displays system information for troubleshooting.

**Effort**: ~2 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: None  
**Files to Create**:
- `pkg/debug/debug.go` — Debug info gathering
- `cmd/commands/debug.go` — Debug command

---

## What You're Building

A debug command that shows:
1. **Kudev version** and build info
2. **Go version** and OS/arch
3. **Docker** status and version
4. **Kubernetes** context and version
5. **Cluster tools** (Kind, Minikube)
6. **Config** validation status

---

## Complete Implementation

### Debug Info Gatherer

```go
// pkg/debug/debug.go

package debug

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"
    
    "github.com/your-org/kudev/pkg/config"
    "github.com/your-org/kudev/pkg/version"
)

// Info contains debug information
type Info struct {
    // Kudev info
    Version   string
    GoVersion string
    OS        string
    Arch      string
    
    // Docker info
    DockerInstalled bool
    DockerRunning   bool
    DockerVersion   string
    
    // Kubernetes info
    KubeconfigPath   string
    KubeconfigExists bool
    CurrentContext   string
    ClusterVersion   string
    
    // Cluster tools
    KindInstalled     bool
    KindVersion       string
    MinikubeInstalled bool
    MinikubeVersion   string
    
    // Kudev config
    ConfigPath   string
    ConfigExists bool
    ConfigValid  bool
    ConfigError  string
}

// Gather collects all debug information
func Gather(ctx context.Context) *Info {
    info := &Info{
        Version:   version.Version,
        GoVersion: runtime.Version(),
        OS:        runtime.GOOS,
        Arch:      runtime.GOARCH,
    }
    
    // Docker info
    gatherDockerInfo(info)
    
    // Kubernetes info
    gatherKubeInfo(info)
    
    // Cluster tools
    gatherClusterTools(info)
    
    // Config info
    gatherConfigInfo(info)
    
    return info
}

func gatherDockerInfo(info *Info) {
    // Check if docker is installed
    path, err := exec.LookPath("docker")
    info.DockerInstalled = err == nil
    
    if !info.DockerInstalled {
        return
    }
    
    // Check if docker is running
    cmd := exec.Command("docker", "info")
    err = cmd.Run()
    info.DockerRunning = err == nil
    
    // Get version
    cmd = exec.Command("docker", "version", "--format", "{{.Server.Version}}")
    out, err := cmd.Output()
    if err == nil {
        info.DockerVersion = strings.TrimSpace(string(out))
    }
}

func gatherKubeInfo(info *Info) {
    // Kubeconfig path
    kubeconfigPath := os.Getenv("KUBECONFIG")
    if kubeconfigPath == "" {
        home, _ := os.UserHomeDir()
        kubeconfigPath = filepath.Join(home, ".kube", "config")
    }
    info.KubeconfigPath = kubeconfigPath
    
    // Check if exists
    _, err := os.Stat(kubeconfigPath)
    info.KubeconfigExists = err == nil
    
    if !info.KubeconfigExists {
        return
    }
    
    // Current context
    cmd := exec.Command("kubectl", "config", "current-context")
    out, err := cmd.Output()
    if err == nil {
        info.CurrentContext = strings.TrimSpace(string(out))
    }
    
    // Cluster version
    cmd = exec.Command("kubectl", "version", "--client=false", "-o", "json")
    out, err = cmd.Output()
    if err == nil && strings.Contains(string(out), "serverVersion") {
        // Parse version from JSON
        info.ClusterVersion = "connected"
    }
}

func gatherClusterTools(info *Info) {
    // Kind
    _, err := exec.LookPath("kind")
    info.KindInstalled = err == nil
    if info.KindInstalled {
        cmd := exec.Command("kind", "version")
        out, _ := cmd.Output()
        info.KindVersion = strings.TrimSpace(string(out))
    }
    
    // Minikube
    _, err = exec.LookPath("minikube")
    info.MinikubeInstalled = err == nil
    if info.MinikubeInstalled {
        cmd := exec.Command("minikube", "version", "--short")
        out, _ := cmd.Output()
        info.MinikubeVersion = strings.TrimSpace(string(out))
    }
}

func gatherConfigInfo(info *Info) {
    // Try to find config
    loader := config.NewFileConfigLoader()
    cfg, err := loader.Load()
    
    if err != nil {
        info.ConfigExists = false
        info.ConfigError = err.Error()
        return
    }
    
    info.ConfigPath = cfg.ConfigPath
    info.ConfigExists = true
    
    // Validate config
    if err := cfg.Validate(); err != nil {
        info.ConfigValid = false
        info.ConfigError = err.Error()
    } else {
        info.ConfigValid = true
    }
}

// Format returns a formatted string of debug info
func (info *Info) Format() string {
    var sb strings.Builder
    
    sb.WriteString("Kudev Debug Information\n")
    sb.WriteString("═══════════════════════════════════════════════════\n\n")
    
    // Kudev section
    sb.WriteString("Kudev:\n")
    sb.WriteString(fmt.Sprintf("  Version:      %s\n", info.Version))
    sb.WriteString(fmt.Sprintf("  Go Version:   %s\n", info.GoVersion))
    sb.WriteString(fmt.Sprintf("  OS/Arch:      %s/%s\n", info.OS, info.Arch))
    sb.WriteString("\n")
    
    // Docker section
    sb.WriteString("Docker:\n")
    if info.DockerInstalled {
        sb.WriteString(fmt.Sprintf("  Installed:    %s\n", checkMark(true)))
        sb.WriteString(fmt.Sprintf("  Running:      %s\n", checkMark(info.DockerRunning)))
        if info.DockerVersion != "" {
            sb.WriteString(fmt.Sprintf("  Version:      %s\n", info.DockerVersion))
        }
    } else {
        sb.WriteString(fmt.Sprintf("  Installed:    %s\n", checkMark(false)))
    }
    sb.WriteString("\n")
    
    // Kubernetes section
    sb.WriteString("Kubernetes:\n")
    sb.WriteString(fmt.Sprintf("  Kubeconfig:   %s\n", info.KubeconfigPath))
    sb.WriteString(fmt.Sprintf("  Exists:       %s\n", checkMark(info.KubeconfigExists)))
    if info.CurrentContext != "" {
        sb.WriteString(fmt.Sprintf("  Context:      %s\n", info.CurrentContext))
    }
    if info.ClusterVersion != "" {
        sb.WriteString(fmt.Sprintf("  Cluster:      %s\n", info.ClusterVersion))
    }
    sb.WriteString("\n")
    
    // Cluster tools section
    sb.WriteString("Cluster Tools:\n")
    if info.KindInstalled {
        sb.WriteString(fmt.Sprintf("  Kind:         %s (%s)\n", checkMark(true), info.KindVersion))
    } else {
        sb.WriteString(fmt.Sprintf("  Kind:         %s\n", checkMark(false)))
    }
    if info.MinikubeInstalled {
        sb.WriteString(fmt.Sprintf("  Minikube:     %s (%s)\n", checkMark(true), info.MinikubeVersion))
    } else {
        sb.WriteString(fmt.Sprintf("  Minikube:     %s\n", checkMark(false)))
    }
    sb.WriteString("\n")
    
    // Config section
    sb.WriteString("Kudev Config:\n")
    if info.ConfigExists {
        sb.WriteString(fmt.Sprintf("  Path:         %s\n", info.ConfigPath))
        sb.WriteString(fmt.Sprintf("  Valid:        %s\n", checkMark(info.ConfigValid)))
        if !info.ConfigValid {
            sb.WriteString(fmt.Sprintf("  Error:        %s\n", info.ConfigError))
        }
    } else {
        sb.WriteString(fmt.Sprintf("  Found:        %s\n", checkMark(false)))
        sb.WriteString(fmt.Sprintf("  (Run 'kudev init' to create one)\n"))
    }
    
    return sb.String()
}

func checkMark(ok bool) string {
    if ok {
        return "✓"
    }
    return "✗"
}
```

### Debug Command

```go
// cmd/commands/debug.go

package commands

import (
    "context"
    "fmt"
    
    "github.com/spf13/cobra"
    
    "github.com/your-org/kudev/pkg/debug"
)

var debugCmd = &cobra.Command{
    Use:   "debug",
    Short: "Show debug information",
    Long: `Show system information useful for troubleshooting.

This command displays:
- Kudev version and build info
- Docker status and version
- Kubernetes configuration and connectivity
- Available cluster tools (Kind, Minikube)
- Kudev configuration status`,
    RunE: runDebug,
}

func init() {
    rootCmd.AddCommand(debugCmd)
}

func runDebug(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    
    info := debug.Gather(ctx)
    fmt.Print(info.Format())
    
    return nil
}
```

---

## Example Output

```
$ kudev debug
Kudev Debug Information
═══════════════════════════════════════════════════

Kudev:
  Version:      v1.0.0
  Go Version:   go1.23.2
  OS/Arch:      linux/amd64

Docker:
  Installed:    ✓
  Running:      ✓
  Version:      25.0.0

Kubernetes:
  Kubeconfig:   /home/user/.kube/config
  Exists:       ✓
  Context:      docker-desktop
  Cluster:      connected

Cluster Tools:
  Kind:         ✓ (kind v0.20.0)
  Minikube:     ✓ (v1.32.0)

Kudev Config:
  Path:         /home/user/project/.kudev.yaml
  Valid:        ✓
```

---

## Checklist for Task 6.5

- [ ] Create `pkg/debug/debug.go`
- [ ] Implement `Info` struct
- [ ] Implement `Gather()` function
- [ ] Implement `gatherDockerInfo()` helper
- [ ] Implement `gatherKubeInfo()` helper
- [ ] Implement `gatherClusterTools()` helper
- [ ] Implement `gatherConfigInfo()` helper
- [ ] Implement `Format()` method
- [ ] Create `cmd/commands/debug.go`
- [ ] Add debug command to root
- [ ] Test output
- [ ] Run `kudev debug`

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 6.6** → Implement CI/CD Pipeline

---

## References

- [runtime Package](https://pkg.go.dev/runtime)
- [os/exec Package](https://pkg.go.dev/os/exec)


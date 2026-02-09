# Task 2.5: Implement Registry-Aware Image Loading

## Overview

This task implements **image loading** into local Kubernetes clusters. Different cluster types (Docker Desktop, Minikube, Kind) require different loading mechanisms. The registry loader auto-detects the cluster type and uses the appropriate native loading command.

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate (subprocess, cluster detection)  
**Dependencies**: Phase 1 (Kubeconfig), Task 2.1 (Types)  
**Files to Create**:
- `pkg/registry/loader.go` — Image loading orchestration
- `pkg/registry/docker.go` — Docker Desktop handling
- `pkg/registry/minikube.go` — Minikube handling
- `pkg/registry/kind.go` — Kind handling
- `pkg/registry/loader_test.go` — Tests

---

## What You're Building

A registry loader that:
1. **Detects** cluster type from kubeconfig context
2. **Delegates** to appropriate cluster-specific loader
3. **Executes** native loading commands
4. **Reports** clear errors if loading fails

---

## The Problem This Solves

### Why Local Image Loading?

Traditional container workflow:
```
Build locally → Push to registry → K8s pulls from registry
                     ↓
              Slow (network)
              Requires registry access
              Credentials management
```

Local development workflow:
```
Build locally → Load directly to cluster
                     ↓
              Fast (no network)
              No registry needed
              No credentials
```

### Cluster-Specific Loading

| Cluster Type | Loading Mechanism | Why Different? |
|--------------|-------------------|----------------|
| Docker Desktop | Automatic | Shares Docker daemon with K8s |
| Minikube | `minikube image load` | Separate Docker daemon inside VM |
| Kind | `kind load docker-image` | Separate containerd in container |

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Registry                            │
├─────────────────────────────────────────────────────────┤
│  Load(ctx, imageRef)                                     │
│    1. Detect cluster type from context                  │
│    2. Select appropriate loader                          │
│    3. Execute loader.Load()                              │
└─────────────────────────────────────────────────────────┘
         │
         ├──────────────────┬────────────────────┐
         ▼                  ▼                    ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│ Docker Desktop  │ │    Minikube     │ │      Kind       │
│     Loader      │ │     Loader      │ │     Loader      │
│                 │ │                 │ │                 │
│ (no-op: shared  │ │ minikube image  │ │ kind load       │
│  daemon)        │ │ load            │ │ docker-image    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

---

## Cluster Detection Logic

### Context Name Patterns

| Context Name | Cluster Type |
|--------------|--------------|
| `docker-desktop` | Docker Desktop |
| `docker-for-desktop` | Docker Desktop (legacy) |
| `minikube` | Minikube |
| `kind-dev` | Kind (cluster: dev) |
| `kind-test` | Kind (cluster: test) |

### Detection Algorithm

```go
func detectClusterType(context string) (ClusterType, string) {
    context = strings.ToLower(context)
    
    switch {
    case strings.Contains(context, "docker-desktop"),
         strings.Contains(context, "docker-for-desktop"):
        return ClusterTypeDockerDesktop, ""
        
    case strings.Contains(context, "minikube"):
        return ClusterTypeMinikube, ""
        
    case strings.HasPrefix(context, "kind-"):
        // Extract cluster name: "kind-dev" → "dev"
        clusterName := strings.TrimPrefix(context, "kind-")
        return ClusterTypeKind, clusterName
        
    default:
        return ClusterTypeUnknown, ""
    }
}
```

---

## Complete Implementation

### File Structure

```
pkg/registry/
├── loader.go       ← Orchestration + types
├── docker.go       ← Docker Desktop loader
├── minikube.go     ← Minikube loader
├── kind.go         ← Kind loader
└── loader_test.go  ← Tests
```

### Types and Registry (loader.go)

```go
// pkg/registry/loader.go

package registry

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/your-org/kudev/pkg/logging"
)

// ClusterType identifies the type of local K8s cluster.
type ClusterType string

const (
    ClusterTypeDockerDesktop ClusterType = "docker-desktop"
    ClusterTypeMinikube      ClusterType = "minikube"
    ClusterTypeKind          ClusterType = "kind"
    ClusterTypeUnknown       ClusterType = "unknown"
)

// Loader is the interface for cluster-specific image loading.
type Loader interface {
    // Load loads an image into the cluster.
    Load(ctx context.Context, imageRef string) error
    
    // Name returns the loader identifier.
    Name() string
}

// Registry orchestrates image loading based on cluster type.
type Registry struct {
    kubeContext string
    logger      logging.Logger
}

// NewRegistry creates a new registry loader.
// kubeContext is the current kubectl context name.
func NewRegistry(kubeContext string, logger logging.Logger) *Registry {
    return &Registry{
        kubeContext: kubeContext,
        logger:      logger,
    }
}

// Load loads an image into the current cluster.
func (r *Registry) Load(ctx context.Context, imageRef string) error {
    r.logger.Info("loading image to cluster",
        "image", imageRef,
        "context", r.kubeContext,
    )
    
    // Detect cluster type
    clusterType, clusterName := detectClusterType(r.kubeContext)
    
    r.logger.Debug("detected cluster type",
        "type", clusterType,
        "clusterName", clusterName,
    )
    
    // Get appropriate loader
    loader, err := r.getLoader(clusterType, clusterName)
    if err != nil {
        return err
    }
    
    r.logger.Debug("using loader", "loader", loader.Name())
    
    // Load the image
    if err := loader.Load(ctx, imageRef); err != nil {
        return fmt.Errorf("failed to load image with %s loader: %w", loader.Name(), err)
    }
    
    r.logger.Info("image loaded successfully",
        "image", imageRef,
        "loader", loader.Name(),
    )
    
    return nil
}

// getLoader returns the appropriate loader for the cluster type.
func (r *Registry) getLoader(clusterType ClusterType, clusterName string) (Loader, error) {
    switch clusterType {
    case ClusterTypeDockerDesktop:
        return newDockerDesktopLoader(r.logger), nil
        
    case ClusterTypeMinikube:
        return newMinikubeLoader(r.logger), nil
        
    case ClusterTypeKind:
        return newKindLoader(clusterName, r.logger), nil
        
    case ClusterTypeUnknown:
        return nil, fmt.Errorf(
            "unknown cluster type for context %q\n\n"+
            "Supported clusters:\n"+
            "  - Docker Desktop (context: docker-desktop)\n"+
            "  - Minikube (context: minikube)\n"+
            "  - Kind (context: kind-<cluster-name>)\n\n"+
            "Tips:\n"+
            "  - Check current context: kubectl config current-context\n"+
            "  - List contexts: kubectl config get-contexts\n"+
            "  - Switch context: kubectl config use-context <name>",
            r.kubeContext,
        )
        
    default:
        return nil, fmt.Errorf("unhandled cluster type: %s", clusterType)
    }
}

// detectClusterType determines the cluster type from context name.
func detectClusterType(kubeContext string) (ClusterType, string) {
    ctx := strings.ToLower(kubeContext)
    
    switch {
    case strings.Contains(ctx, "docker-desktop"),
         strings.Contains(ctx, "docker-for-desktop"):
        return ClusterTypeDockerDesktop, ""
        
    case strings.Contains(ctx, "minikube"):
        return ClusterTypeMinikube, ""
        
    case strings.HasPrefix(ctx, "kind-"):
        // Extract cluster name: "kind-dev" → "dev"
        clusterName := strings.TrimPrefix(ctx, "kind-")
        return ClusterTypeKind, clusterName
        
    default:
        return ClusterTypeUnknown, ""
    }
}

// GetClusterType returns the detected cluster type for the current context.
// Useful for debugging and testing.
func (r *Registry) GetClusterType() (ClusterType, string) {
    return detectClusterType(r.kubeContext)
}

// KubeContext returns the kubernetes context being used.
func (r *Registry) KubeContext() string {
    return r.kubeContext
}
```

### Docker Desktop Loader (docker.go)

```go
// pkg/registry/docker.go

package registry

import (
    "context"
    
    "github.com/your-org/kudev/pkg/logging"
)

// dockerDesktopLoader handles image loading for Docker Desktop.
type dockerDesktopLoader struct {
    logger logging.Logger
}

// newDockerDesktopLoader creates a new Docker Desktop loader.
func newDockerDesktopLoader(logger logging.Logger) *dockerDesktopLoader {
    return &dockerDesktopLoader{logger: logger}
}

// Name returns the loader identifier.
func (d *dockerDesktopLoader) Name() string {
    return "docker-desktop"
}

// Load loads an image into Docker Desktop's Kubernetes.
// Docker Desktop shares the Docker daemon with its built-in K8s cluster,
// so images built locally are automatically available - no loading needed.
func (d *dockerDesktopLoader) Load(ctx context.Context, imageRef string) error {
    d.logger.Info("image available to Docker Desktop automatically",
        "image", imageRef,
        "reason", "Docker Desktop shares daemon with K8s",
    )
    
    // No action needed - Docker Desktop K8s uses the same Docker daemon
    // that was used to build the image
    return nil
}

// Ensure dockerDesktopLoader implements Loader
var _ Loader = (*dockerDesktopLoader)(nil)
```

### Minikube Loader (minikube.go)

```go
// pkg/registry/minikube.go

package registry

import (
    "context"
    "fmt"
    "os/exec"
    "strings"
    
    "github.com/your-org/kudev/pkg/logging"
)

// minikubeLoader handles image loading for Minikube.
type minikubeLoader struct {
    logger logging.Logger
}

// newMinikubeLoader creates a new Minikube loader.
func newMinikubeLoader(logger logging.Logger) *minikubeLoader {
    return &minikubeLoader{logger: logger}
}

// Name returns the loader identifier.
func (m *minikubeLoader) Name() string {
    return "minikube"
}

// Load loads an image into Minikube using `minikube image load`.
func (m *minikubeLoader) Load(ctx context.Context, imageRef string) error {
    m.logger.Info("loading image via minikube",
        "image", imageRef,
        "command", "minikube image load",
    )
    
    // Check if minikube is available
    if err := m.checkMinikube(ctx); err != nil {
        return err
    }
    
    // Run minikube image load
    cmd := exec.CommandContext(ctx, "minikube", "image", "load", imageRef)
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        return fmt.Errorf(
            "minikube image load failed\n\n"+
            "Command: minikube image load %s\n"+
            "Output: %s\n"+
            "Error: %w\n\n"+
            "Troubleshooting:\n"+
            "  - Ensure Minikube is running: minikube status\n"+
            "  - Start Minikube: minikube start\n"+
            "  - Check image exists: docker images %s",
            imageRef, strings.TrimSpace(string(output)), err, imageRef,
        )
    }
    
    m.logger.Info("image loaded to minikube successfully",
        "image", imageRef,
    )
    
    return nil
}

// checkMinikube verifies minikube CLI is available.
func (m *minikubeLoader) checkMinikube(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "minikube", "version", "--short")
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        return fmt.Errorf(
            "minikube CLI not found or not working\n\n"+
            "Please install Minikube:\n"+
            "  - macOS: brew install minikube\n"+
            "  - Windows: choco install minikube\n"+
            "  - Linux: see https://minikube.sigs.k8s.io/docs/start/\n\n"+
            "Error: %w",
            err,
        )
    }
    
    m.logger.Debug("minikube CLI available",
        "version", strings.TrimSpace(string(output)),
    )
    
    return nil
}

// Ensure minikubeLoader implements Loader
var _ Loader = (*minikubeLoader)(nil)
```

### Kind Loader (kind.go)

```go
// pkg/registry/kind.go

package registry

import (
    "context"
    "fmt"
    "os/exec"
    "strings"
    
    "github.com/your-org/kudev/pkg/logging"
)

// kindLoader handles image loading for Kind clusters.
type kindLoader struct {
    clusterName string
    logger      logging.Logger
}

// newKindLoader creates a new Kind loader.
// clusterName is extracted from the context (e.g., "kind-dev" → "dev").
func newKindLoader(clusterName string, logger logging.Logger) *kindLoader {
    // Default to "kind" if no cluster name provided
    if clusterName == "" {
        clusterName = "kind"
    }
    
    return &kindLoader{
        clusterName: clusterName,
        logger:      logger,
    }
}

// Name returns the loader identifier.
func (k *kindLoader) Name() string {
    return "kind"
}

// ClusterName returns the Kind cluster name.
func (k *kindLoader) ClusterName() string {
    return k.clusterName
}

// Load loads an image into Kind using `kind load docker-image`.
func (k *kindLoader) Load(ctx context.Context, imageRef string) error {
    k.logger.Info("loading image via kind",
        "image", imageRef,
        "cluster", k.clusterName,
        "command", "kind load docker-image",
    )
    
    // Check if kind is available
    if err := k.checkKind(ctx); err != nil {
        return err
    }
    
    // Run kind load docker-image
    cmd := exec.CommandContext(ctx,
        "kind", "load", "docker-image", imageRef,
        "--name", k.clusterName,
    )
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        return fmt.Errorf(
            "kind load failed\n\n"+
            "Command: kind load docker-image %s --name %s\n"+
            "Output: %s\n"+
            "Error: %w\n\n"+
            "Troubleshooting:\n"+
            "  - Ensure Kind cluster exists: kind get clusters\n"+
            "  - Create cluster: kind create cluster --name %s\n"+
            "  - Check image exists: docker images %s",
            imageRef, k.clusterName,
            strings.TrimSpace(string(output)), err,
            k.clusterName, imageRef,
        )
    }
    
    k.logger.Info("image loaded to kind cluster successfully",
        "image", imageRef,
        "cluster", k.clusterName,
    )
    
    return nil
}

// checkKind verifies kind CLI is available.
func (k *kindLoader) checkKind(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "kind", "version")
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        return fmt.Errorf(
            "kind CLI not found or not working\n\n"+
            "Please install Kind:\n"+
            "  - macOS: brew install kind\n"+
            "  - Windows: choco install kind\n"+
            "  - Go: go install sigs.k8s.io/kind@latest\n"+
            "  - See: https://kind.sigs.k8s.io/docs/user/quick-start/\n\n"+
            "Error: %w",
            err,
        )
    }
    
    k.logger.Debug("kind CLI available",
        "version", strings.TrimSpace(string(output)),
    )
    
    return nil
}

// Ensure kindLoader implements Loader
var _ Loader = (*kindLoader)(nil)
```

---

## Key Implementation Details

### 1. Docker Desktop - No-Op

Docker Desktop shares the Docker daemon with its built-in Kubernetes:
```go
func (d *dockerDesktopLoader) Load(ctx context.Context, imageRef string) error {
    // No action needed - image is already available
    d.logger.Info("image available automatically")
    return nil
}
```

### 2. Minikube - Image Load

Minikube runs its own Docker daemon inside a VM:
```go
cmd := exec.CommandContext(ctx, "minikube", "image", "load", imageRef)
```

### 3. Kind - Load with Cluster Name

Kind requires the cluster name:
```go
cmd := exec.CommandContext(ctx,
    "kind", "load", "docker-image", imageRef,
    "--name", k.clusterName,  // e.g., "dev" from context "kind-dev"
)
```

### 4. CLI Availability Check

Always verify CLI tools before use:
```go
func (k *kindLoader) checkKind(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "kind", "version")
    _, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("kind CLI not found...")
    }
    return nil
}
```

---

## Testing Strategy

### Unit Tests

```go
// pkg/registry/loader_test.go

package registry

import (
    "context"
    "testing"
)

type mockLogger struct {
    messages []string
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
    m.messages = append(m.messages, msg)
}
func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {}

func TestDetectClusterType(t *testing.T) {
    tests := []struct {
        context     string
        wantType    ClusterType
        wantCluster string
    }{
        {"docker-desktop", ClusterTypeDockerDesktop, ""},
        {"docker-for-desktop", ClusterTypeDockerDesktop, ""},
        {"Docker-Desktop", ClusterTypeDockerDesktop, ""}, // Case insensitive
        
        {"minikube", ClusterTypeMinikube, ""},
        {"Minikube", ClusterTypeMinikube, ""},
        
        {"kind-dev", ClusterTypeKind, "dev"},
        {"kind-test", ClusterTypeKind, "test"},
        {"kind-production", ClusterTypeKind, "production"},
        {"Kind-Dev", ClusterTypeKind, "dev"}, // Case insensitive
        
        {"unknown-context", ClusterTypeUnknown, ""},
        {"gke_project_zone_cluster", ClusterTypeUnknown, ""},
        {"arn:aws:eks:region:account:cluster/name", ClusterTypeUnknown, ""},
    }
    
    for _, tt := range tests {
        t.Run(tt.context, func(t *testing.T) {
            gotType, gotCluster := detectClusterType(tt.context)
            
            if gotType != tt.wantType {
                t.Errorf("detectClusterType(%q) type = %v, want %v",
                    tt.context, gotType, tt.wantType)
            }
            
            if gotCluster != tt.wantCluster {
                t.Errorf("detectClusterType(%q) cluster = %q, want %q",
                    tt.context, gotCluster, tt.wantCluster)
            }
        })
    }
}

func TestRegistry_GetLoader(t *testing.T) {
    logger := &mockLogger{}
    
    tests := []struct {
        context     string
        wantLoader  string
        wantErr     bool
    }{
        {"docker-desktop", "docker-desktop", false},
        {"minikube", "minikube", false},
        {"kind-dev", "kind", false},
        {"unknown", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.context, func(t *testing.T) {
            r := NewRegistry(tt.context, logger)
            clusterType, clusterName := detectClusterType(tt.context)
            
            loader, err := r.getLoader(clusterType, clusterName)
            
            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                }
                return
            }
            
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            
            if loader.Name() != tt.wantLoader {
                t.Errorf("loader.Name() = %q, want %q", loader.Name(), tt.wantLoader)
            }
        })
    }
}

func TestDockerDesktopLoader_Load(t *testing.T) {
    logger := &mockLogger{}
    loader := newDockerDesktopLoader(logger)
    
    // Should always succeed (no-op)
    err := loader.Load(context.Background(), "myapp:kudev-abc123")
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    
    // Should log that image is available
    found := false
    for _, msg := range logger.messages {
        if msg == "image available to Docker Desktop automatically" {
            found = true
            break
        }
    }
    if !found {
        t.Error("expected log message about automatic availability")
    }
}

func TestKindLoader_ClusterName(t *testing.T) {
    logger := &mockLogger{}
    
    tests := []struct {
        input    string
        expected string
    }{
        {"dev", "dev"},
        {"test", "test"},
        {"", "kind"}, // Default
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            loader := newKindLoader(tt.input, logger)
            if loader.ClusterName() != tt.expected {
                t.Errorf("ClusterName() = %q, want %q", loader.ClusterName(), tt.expected)
            }
        })
    }
}

func TestRegistry_KubeContext(t *testing.T) {
    logger := &mockLogger{}
    r := NewRegistry("docker-desktop", logger)
    
    if r.KubeContext() != "docker-desktop" {
        t.Errorf("KubeContext() = %q, want %q", r.KubeContext(), "docker-desktop")
    }
}

func TestLoaderInterface(t *testing.T) {
    // Compile-time check that all loaders implement Loader
    var _ Loader = (*dockerDesktopLoader)(nil)
    var _ Loader = (*minikubeLoader)(nil)
    var _ Loader = (*kindLoader)(nil)
}
```

### Integration Tests (Optional)

```go
// +build integration

package registry

import (
    "context"
    "os/exec"
    "testing"
)

func TestMinikubeLoader_Integration(t *testing.T) {
    // Skip if minikube not available
    if _, err := exec.LookPath("minikube"); err != nil {
        t.Skip("minikube not available")
    }
    
    // Skip if minikube not running
    cmd := exec.Command("minikube", "status")
    if err := cmd.Run(); err != nil {
        t.Skip("minikube not running")
    }
    
    logger := &mockLogger{}
    loader := newMinikubeLoader(logger)
    
    // This requires a real image to exist
    // Use a common base image for testing
    err := loader.Load(context.Background(), "alpine:latest")
    if err != nil {
        t.Errorf("Load failed: %v", err)
    }
}
```

---

## Usage Examples

### Basic Usage

```go
// Get current kubernetes context
currentContext := kubeconfig.GetCurrentContext()  // e.g., "docker-desktop"

// Create registry loader
registry := NewRegistry(currentContext, logger)

// Load image
imageRef := "myapp:kudev-a1b2c3d4"
if err := registry.Load(ctx, imageRef); err != nil {
    return fmt.Errorf("failed to load image: %w", err)
}
```

### Full Pipeline Integration

```go
func DeployToCluster(ctx context.Context, projectRoot, imageName string) error {
    // 1. Build image
    calc := hash.NewCalculator(projectRoot, nil)
    tagger := builder.NewTagger(calc)
    tag, _ := tagger.GenerateTag(ctx, false)
    
    dockerBuilder := docker.NewDockerBuilder(logger)
    imageRef, err := dockerBuilder.Build(ctx, builder.BuildOptions{
        SourceDir:      projectRoot,
        DockerfilePath: "./Dockerfile",
        ImageName:      imageName,
        ImageTag:       tag,
    })
    if err != nil {
        return fmt.Errorf("build failed: %w", err)
    }
    
    // 2. Load image to cluster
    currentContext := kubeconfig.GetCurrentContext()
    reg := registry.NewRegistry(currentContext, logger)
    
    if err := reg.Load(ctx, imageRef.FullRef); err != nil {
        return fmt.Errorf("failed to load image: %w", err)
    }
    
    // 3. Deploy to K8s (Phase 3)
    // ...
    
    return nil
}
```

---

## How Registry Connects to Other Tasks

```
Task 2.5 (Registry Loader) ← You are here
    │
    ├─► Uses: Phase 1 (Kubeconfig)
    │   - Gets current context for cluster detection
    │
    ├─► Uses: Task 2.1 (Types)
    │   - Uses ImageRef.FullRef to identify image
    │
    └─► Used by: Phase 3 (Manifest Orchestration)
        - Loads images before deployment
```

---

## Checklist for Task 2.5

- [ ] Create `pkg/registry/loader.go`
- [ ] Define `ClusterType` constants
- [ ] Define `Loader` interface
- [ ] Implement `Registry` struct
- [ ] Implement `NewRegistry()` constructor
- [ ] Implement `Load()` method
- [ ] Implement `getLoader()` method
- [ ] Implement `detectClusterType()` function
- [ ] Create `pkg/registry/docker.go`
- [ ] Implement `dockerDesktopLoader`
- [ ] Create `pkg/registry/minikube.go`
- [ ] Implement `minikubeLoader` with `checkMinikube()`
- [ ] Create `pkg/registry/kind.go`
- [ ] Implement `kindLoader` with `checkKind()`
- [ ] Create `pkg/registry/loader_test.go`
- [ ] Write tests for `detectClusterType()`
- [ ] Write tests for `getLoader()`
- [ ] Write tests for Docker Desktop loader
- [ ] Write tests for loader interfaces
- [ ] Run `go fmt ./pkg/registry`
- [ ] Verify compilation: `go build ./pkg/registry`
- [ ] Run tests: `go test ./pkg/registry -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Hardcoding cluster names
```go
// Wrong - assumes single Kind cluster
cmd := exec.Command("kind", "load", "docker-image", imageRef)

// Right - use detected cluster name
cmd := exec.Command("kind", "load", "docker-image", imageRef,
    "--name", k.clusterName)
```

❌ **Mistake 2**: Not checking CLI availability
```go
// Wrong - confusing error if kind not installed
func (k *kindLoader) Load(ctx context.Context, imageRef string) error {
    cmd := exec.Command("kind", "load", ...)
    // Error: "exec: kind: executable file not found"
}

// Right - helpful error message
func (k *kindLoader) Load(ctx context.Context, imageRef string) error {
    if err := k.checkKind(ctx); err != nil {
        return err  // "kind CLI not found, install with..."
    }
    // ...
}
```

❌ **Mistake 3**: Case-sensitive context matching
```go
// Wrong - fails for "Docker-Desktop"
case context == "docker-desktop":

// Right - case insensitive
case strings.Contains(strings.ToLower(context), "docker-desktop"):
```

❌ **Mistake 4**: Ignoring command output on error
```go
// Wrong - loses helpful debug info
_, err := cmd.Run()
if err != nil {
    return err  // Just "exit status 1"
}

// Right - include output in error
output, err := cmd.CombinedOutput()
if err != nil {
    return fmt.Errorf("failed: %w\nOutput: %s", err, output)
}
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Phase 2 is now complete! 🎉
3. Move to **Phase 3** → Manifest Orchestration
4. Phase 3 will use Registry to load images before deployment

---

## References

- [Docker Desktop Kubernetes](https://docs.docker.com/desktop/kubernetes/)
- [Minikube image load](https://minikube.sigs.k8s.io/docs/commands/image/)
- [Kind load docker-image](https://kind.sigs.k8s.io/docs/user/quick-start/#loading-an-image-into-your-cluster)
- [kubectl config](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_config/)


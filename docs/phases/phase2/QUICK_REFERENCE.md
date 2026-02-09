# Phase 2 Quick Reference Guide

## For Busy Developers

This is a **TL;DR** version of Phase 2. For full details, see individual task files.

---

## Task Sequence & Time Estimates

```
Task 2.1 (2h)  → Builder interface & types
Task 2.2 (4h)  → Docker builder implementation
Task 2.3 (3h)  → Source code hashing
Task 2.4 (2h)  → Image tagging
Task 2.5 (4h)  → Registry-aware loading
         ────────
Total: ~12-16 hours
```

---

## Core Concepts

### 1. Build Pipeline
- **Input**: Source code directory + Dockerfile
- **Processing**: Hash → Build → Tag → Load
- **Output**: Image loaded in local K8s cluster

### 2. Hash-Based Tagging
- **Format**: `kudev-{8-char-hash}`
- **Deterministic**: Same source = same tag
- **Cache-friendly**: Skip rebuild if unchanged

### 3. Cluster-Aware Loading
- **Docker Desktop**: Automatic (shared daemon)
- **Minikube**: `minikube image load`
- **Kind**: `kind load docker-image`

---

## File Map

| File | Purpose | Key Types/Functions |
|------|---------|---------------------|
| `pkg/builder/types.go` | Interface & types | `Builder`, `BuildOptions`, `ImageRef` |
| `pkg/builder/tagger.go` | Tag generation | `Tagger`, `GenerateTag()`, `IsKudevTag()` |
| `pkg/builder/docker/builder.go` | Docker impl | `DockerBuilder`, `Build()` |
| `pkg/hash/calculator.go` | Hash calculation | `Calculator`, `Calculate()` |
| `pkg/hash/exclusions.go` | Exclusion patterns | `shouldExclude()`, `LoadDockerignore()` |
| `pkg/registry/loader.go` | Orchestration | `Registry`, `Load()`, `detectClusterType()` |
| `pkg/registry/docker.go` | Docker Desktop | `dockerDesktopLoader` |
| `pkg/registry/minikube.go` | Minikube | `minikubeLoader` |
| `pkg/registry/kind.go` | Kind | `kindLoader` |

---

## Key Decisions (Why?)

| Decision | Choice | Why |
|----------|--------|-----|
| Build tool | Docker CLI subprocess | Lightweight, no SDK, users have Docker |
| Tagging | Hash-based | Deterministic, cache-friendly |
| Hash length | 8 chars | Readable, unique enough |
| Hash algorithm | SHA256 | Standard, fast |
| Cluster detection | Context name pattern | Simple, reliable |

---

## Pattern: Build Pipeline Flow

```
1. Hash Source
   hash.Calculator.Calculate() → "a1b2c3d4"

2. Generate Tag
   builder.Tagger.GenerateTag() → "kudev-a1b2c3d4"

3. Build Image
   docker.DockerBuilder.Build(opts) → ImageRef

4. Load to Cluster
   registry.Registry.Load(imageRef) → loaded

5. Ready for Deployment
   → Phase 3 uses ImageRef.FullRef
```

---

## Pattern: Builder Interface

```go
// Minimal interface - easy to implement, test, extend
type Builder interface {
    Build(ctx context.Context, opts BuildOptions) (*ImageRef, error)
    Name() string
}

// Options struct - clear, extensible
type BuildOptions struct {
    SourceDir      string
    DockerfilePath string
    ImageName      string
    ImageTag       string
    BuildArgs      map[string]string
    Target         string
    NoCache        bool
}

// Result struct - contains reference and ID
type ImageRef struct {
    FullRef string  // "myapp:kudev-a1b2c3d4"
    ID      string  // "sha256:abc123..."
}
```

---

## Pattern: Hash Calculation

```go
calc := hash.NewCalculator(projectRoot, customExclusions)
sourceHash, err := calc.Calculate(ctx)
// sourceHash = "a1b2c3d4" (8 chars)
```

**Determinism Rules**:
1. Sort files before combining hashes
2. Use relative paths (not absolute)
3. Include file path in hash (rename = new hash)
4. Normalize path separators

**Default Exclusions**:
- `.git`, `.gitignore`
- `node_modules`, `vendor`
- `*.log`, `*.tmp`
- `.DS_Store`, `Thumbs.db`

---

## Pattern: Cluster Detection

```go
context := "kind-dev"
clusterType, clusterName := detectClusterType(context)
// clusterType = ClusterTypeKind
// clusterName = "dev"

switch clusterType {
case ClusterTypeDockerDesktop:
    // No-op (shared daemon)
case ClusterTypeMinikube:
    exec.Command("minikube", "image", "load", imageRef)
case ClusterTypeKind:
    exec.Command("kind", "load", "docker-image", imageRef, "--name", clusterName)
}
```

---

## Implementation Checklist

### Task 2.1: Builder Types
```
[ ] pkg/builder/types.go created
[ ] Builder interface defined
[ ] BuildOptions struct with Validate()
[ ] ImageRef struct with String()
[ ] BuilderFactory type defined
[ ] Tests pass: go test ./pkg/builder
```

### Task 2.2: Docker Builder
```
[ ] pkg/builder/docker/builder.go created
[ ] DockerBuilder struct
[ ] checkDockerDaemon() helper
[ ] buildCommandArgs() helper
[ ] streamOutput() with goroutines
[ ] getImageID() helper
[ ] Tests pass: go test ./pkg/builder/docker
```

### Task 2.3: Source Hashing
```
[ ] pkg/hash/calculator.go created
[ ] pkg/hash/exclusions.go created
[ ] Calculator.Calculate() deterministic
[ ] shouldExclude() works
[ ] LoadDockerignore() works
[ ] Tests pass: go test ./pkg/hash
```

### Task 2.4: Image Tagging
```
[ ] pkg/builder/tagger.go created
[ ] Tagger.GenerateTag() works
[ ] IsKudevTag() validates format
[ ] ParseTag() extracts hash
[ ] CompareHashes() works
[ ] Tests pass
```

### Task 2.5: Registry Loading
```
[ ] pkg/registry/loader.go created
[ ] pkg/registry/docker.go created
[ ] pkg/registry/minikube.go created
[ ] pkg/registry/kind.go created
[ ] detectClusterType() works
[ ] All loaders implement Loader
[ ] CLI availability checks
[ ] Tests pass: go test ./pkg/registry
```

---

## Common Commands

```bash
# Run all Phase 2 tests
go test ./pkg/builder/... ./pkg/hash/... ./pkg/registry/... -v

# Check coverage
go test ./pkg/builder/... ./pkg/hash/... ./pkg/registry/... -cover

# Build all packages
go build ./pkg/...

# Format code
go fmt ./pkg/builder/... ./pkg/hash/... ./pkg/registry/...
```

---

## Error Messages Template

### Docker Daemon Not Running
```
docker daemon is not running or not accessible

Troubleshooting:
  1. Ensure Docker Desktop is running
  2. Or start Docker daemon: sudo systemctl start docker
  3. Verify with: docker version
```

### Unknown Cluster Type
```
unknown cluster type for context "gke_project_zone_cluster"

Supported clusters:
  - Docker Desktop (context: docker-desktop)
  - Minikube (context: minikube)
  - Kind (context: kind-<cluster-name>)

Tips:
  - Check current context: kubectl config current-context
  - Switch context: kubectl config use-context <name>
```

### Kind/Minikube Not Installed
```
kind CLI not found or not working

Please install Kind:
  - macOS: brew install kind
  - Windows: choco install kind
  - Go: go install sigs.k8s.io/kind@latest
```

---

## Integration Example

```go
func BuildAndLoad(ctx context.Context, cfg *config.DeploymentConfig) (*builder.ImageRef, error) {
    projectRoot := cfg.ProjectRoot()
    
    // 1. Create hash calculator
    exclusions := cfg.Spec.BuildContextExclusions
    calc := hash.NewCalculator(projectRoot, exclusions)
    
    // 2. Generate tag
    tagger := builder.NewTagger(calc)
    tag, err := tagger.GenerateTag(ctx, false)
    if err != nil {
        return nil, err
    }
    
    // 3. Build image
    db := docker.NewDockerBuilder(logger)
    opts := builder.BuildOptions{
        SourceDir:      projectRoot,
        DockerfilePath: cfg.Spec.DockerfilePath,
        ImageName:      cfg.Spec.ImageName,
        ImageTag:       tag,
    }
    
    imageRef, err := db.Build(ctx, opts)
    if err != nil {
        return nil, err
    }
    
    // 4. Load to cluster
    reg := registry.NewRegistry(cfg.Spec.KubeContext, logger)
    if err := reg.Load(ctx, imageRef.FullRef); err != nil {
        return nil, err
    }
    
    return imageRef, nil
}
```

---

## Dependencies Between Tasks

```
Task 2.1 (Types)
    │
    ├──► Task 2.2 (Docker Builder) uses BuildOptions, ImageRef
    │
    ├──► Task 2.4 (Tagger) generates ImageTag
    │         │
    │         └── uses Task 2.3 (Hash Calculator)
    │
    └──► Task 2.5 (Registry) uses ImageRef
```

---

## Testing Tips

1. **Mock the logger** for unit tests
2. **Use t.TempDir()** for file-based tests
3. **Check interface compliance** with `var _ Interface = (*Type)(nil)`
4. **Tag integration tests** with `// +build docker_required`
5. **Test determinism** by calculating hash twice

---

## Next Phase

After completing Phase 2:
- ✅ Can build Docker images
- ✅ Can generate deterministic tags
- ✅ Can load images to local clusters

**Phase 3 (Manifest Orchestration)** will:
- Generate Kubernetes Deployment manifests
- Generate Service manifests
- Apply manifests to cluster
- Use ImageRef from Phase 2


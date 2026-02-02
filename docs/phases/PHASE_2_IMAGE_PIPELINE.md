# Phase 2: Image Pipeline (Build System)

**Objective**: Convert local source code into a container image and make it available to the K8s cluster without pushing to a remote registry.

**Timeline**: 1-2 weeks  
**Difficulty**: 🟡 Intermediate (subprocess calls, file hashing, Docker interaction)  
**Dependencies**: Phase 1 (Config, Logger, CLI)

---

## 📋 Architecture Overview

This phase implements the build-and-load pipeline:

```
┌──────────────────────────────────────────────────────┐
│            User runs: kudev up                       │
└──────────────────┬───────────────────────────────────┘
                   │
        ┌──────────▼──────────┐
        │  Calculate Source   │
        │  Hash (hash/)       │
        └──────────┬──────────┘
                   │
        ┌──────────▼──────────┐
        │  Build Docker Image │
        │  (builder/)         │
        └──────────┬──────────┘
                   │
        ┌──────────▼──────────┐
        │  Tag Image with     │
        │  Hash+Timestamp     │
        └──────────┬──────────┘
                   │
        ┌──────────▼──────────┐
        │  Load to Cluster    │
        │  (registry/)        │
        │  - Docker Desktop   │
        │  - Minikube         │
        │  - Kind             │
        └──────────┬──────────┘
                   │
        ┌──────────▼──────────┐
        │  Return ImageRef    │
        │  (image:tag)        │
        └──────────────────────┘
```

---

## 🎯 Core Decisions

### Decision 2.1: Build Tool Implementation

**Question**: How should we invoke Docker build?

| Approach | Pros | Cons |
|----------|------|------|
| Docker SDK (`moby/moby`) | Programmatic, fine-grained control | Heavy dependency, resource management |
| Docker CLI subprocess | Simple, lightweight, already installed | Need to parse output, less control |
| Buildpacks/ko | Language-agnostic, fast | Not everyone has them installed |

**🎯 Decision**: **Docker CLI subprocess** (Phase 1) → Extensions in Phase 2+
- Lightweight binary (no Docker SDK bloat)
- Matches kubectl philosophy (delegate to CLI tools)
- Users already have Docker CLI (Docker Desktop, Minikube)
- Easy to extend with other tools later

**Implementation Pattern**:
```go
// Use os/exec to call docker build
cmd := exec.CommandContext(ctx, "docker", "build", ...)
```

### Decision 2.2: Image Tagging Strategy

**Question**: How should images be tagged to ensure K8s pulls new versions?

| Strategy | Pros | Cons |
|----------|------|------|
| Always use `:latest` | Simple | K8s won't pull if tag exists |
| Git commit SHA | Guaranteed unique | Doesn't reflect current code |
| Hash of source files | Deterministic, reflects code | Need to calculate hash |
| Timestamp | Always unique | Not deterministic, rebuilds on redeploy |

**🎯 Decision**: **Hash-based + optional timestamp**
- Tag: `{imageName}:kudev-{8-char-hash}`
- Example: `myapp:kudev-a1b2c3d4`
- Only rebuild if source code hash changes
- Optional `--build-timestamp` for forced rebuild

**Hash Calculation**:
```
1. Walk source directory
2. For each file:
   - Skip files matching .dockerignore or exclusions
   - Hash file path + content
3. Combine all file hashes → SHA256
4. Truncate to 8 chars
```

**Benefits**:
- ✅ Deterministic (same source = same tag)
- ✅ Respects .dockerignore patterns
- ✅ Caching: skip build if hash unchanged
- ✅ K8s pulls new image (imagePullPolicy: Always forces pull)

### Decision 2.3: Registry Handling for Local K8s

**Question**: How to load images into local K8s clusters efficiently?

| Strategy | Docker Desktop | Minikube | Kind |
|----------|---|---|---|
| Push to external registry | ❌ Slow | ❌ Slow | ❌ Slow |
| Docker CLI load | ✅ Auto-available | ❌ No docker socket | ❌ No docker socket |
| Minikube image load | N/A | ✅ Dedicated command | N/A |
| Kind load docker-image | N/A | N/A | ✅ Dedicated command |

**🎯 Decision**: **Auto-detect cluster type and use native loading**
- Docker Desktop: Image available automatically (share Docker daemon)
- Minikube: Run `minikube image load {image}`
- Kind: Run `kind load docker-image {image} --name {cluster}`

**Detection Logic**:
```go
// Get kubeconfig context
context := getCurrentContext()

// Match against cluster patterns
switch {
case strings.Contains(context, "docker-desktop"):
    // No action needed
    return nil
case strings.Contains(context, "minikube"):
    // Run: minikube image load
    return exec.Command("minikube", "image", "load", imageRef)
case strings.HasPrefix(context, "kind-"):
    // Extract cluster name: kind-dev → dev
    clusterName := strings.TrimPrefix(context, "kind-")
    return exec.Command("kind", "load", "docker-image", imageRef, "--name", clusterName)
default:
    return fmt.Errorf("unknown cluster type from context: %s", context)
}
```

---

## 📝 Detailed Tasks

### Task 2.1: Define Builder Interface & Types

**Goal**: Create extensible builder abstraction.

**Files to Create**:
- `pkg/builder/types.go` — Builder interface and option types

**Builder Interface**:

```go
// pkg/builder/types.go

// Builder abstracts different image building implementations
type Builder interface {
    // Build creates a container image from source
    Build(ctx context.Context, opts BuildOptions) (*ImageRef, error)
    
    // Name identifies the builder
    Name() string
}

// BuildOptions contains input parameters for building
type BuildOptions struct {
    // Source directory (project root)
    SourceDir string
    
    // Dockerfile path relative to SourceDir
    DockerfilePath string
    
    // Output image name (without tag or registry)
    ImageName string
    
    // Output image tag
    ImageTag string
    
    // Build arguments to pass to docker build
    BuildArgs map[string]string
}

// ImageRef is the fully qualified image reference
type ImageRef struct {
    // Full reference: name:tag (e.g., myapp:kudev-a1b2c3d4)
    FullRef string
    
    // Image ID from Docker (sha256:...)
    ID string
}

// Builder factory function
type BuilderFactory func() (Builder, error)
```

**Success Criteria**:
- ✅ Interface is minimal and clear
- ✅ BuildOptions covers all Docker build needs
- ✅ ImageRef contains both reference and ID
- ✅ Can be easily extended with new builders

**Hints for Implementation**:
- Keep interface small (easier to test and extend)
- Use functional options pattern for BuildOptions in future
- Document when each field is required

---

### Task 2.2: Implement Docker Builder

**Goal**: Build Docker images using CLI subprocess.

**Files to Create**:
- `pkg/builder/docker/builder.go` — Docker implementation
- `pkg/builder/docker/output.go` — Output parsing (optional)

**Docker Builder Implementation**:

```go
// pkg/builder/docker/builder.go

type DockerBuilder struct {
    logger logging.Logger
}

func NewDockerBuilder(logger logging.Logger) *DockerBuilder {
    return &DockerBuilder{logger: logger}
}

func (db *DockerBuilder) Build(ctx context.Context, opts BuildOptions) (*ImageRef, error) {
    // 1. Verify Docker daemon is running
    if err := db.checkDockerDaemon(ctx); err != nil {
        return nil, fmt.Errorf("docker daemon unavailable: %w", err)
    }
    
    // 2. Build docker build command
    cmd := exec.CommandContext(ctx, "docker", "build",
        "-t", fmt.Sprintf("%s:%s", opts.ImageName, opts.ImageTag),
        "-f", opts.DockerfilePath,
        opts.SourceDir,  // Build context
    )
    
    // Add build args if provided
    for key, val := range opts.BuildArgs {
        cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("%s=%s", key, val))
    }
    
    // 3. Stream output to user
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()
    
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("failed to start docker build: %w", err)
    }
    
    // Stream logs in real-time
    go db.streamOutput("stdout", stdout)
    go db.streamOutput("stderr", stderr)
    
    // 4. Wait for completion
    if err := cmd.Wait(); err != nil {
        return nil, fmt.Errorf("docker build failed: %w", err)
    }
    
    // 5. Get image ID
    imageID, err := db.getImageID(ctx, opts.ImageName, opts.ImageTag)
    if err != nil {
        return nil, fmt.Errorf("failed to get image ID: %w", err)
    }
    
    return &ImageRef{
        FullRef: fmt.Sprintf("%s:%s", opts.ImageName, opts.ImageTag),
        ID:      imageID,
    }, nil
}

func (db *DockerBuilder) checkDockerDaemon(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "docker", "version")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf(
            "docker daemon is not running\n" +
            "Start Docker Desktop or verify: docker version\n" +
            "Error: %w", err,
        )
    }
    return nil
}

func (db *DockerBuilder) streamOutput(label string, r io.Reader) {
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        db.logger.Info("docker output", "source", label, "line", scanner.Text())
    }
}

func (db *DockerBuilder) getImageID(ctx context.Context, image, tag string) (string, error) {
    cmd := exec.CommandContext(ctx, "docker", "inspect",
        "--format={{.ID}}", fmt.Sprintf("%s:%s", image, tag))
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

func (db *DockerBuilder) Name() string {
    return "docker"
}
```

**Success Criteria**:
- ✅ Detects Docker daemon availability
- ✅ Executes docker build with correct args
- ✅ Streams build output to terminal in real-time
- ✅ Returns ImageRef with ID
- ✅ Clear error message if Docker unavailable
- ✅ Respects context cancellation

**Hints for Implementation**:
- Use `io.Copy()` to stream output without buffering
- Check `docker version` first to verify daemon is running
- Parse Dockerfile path relative to source directory
- Handle docker build errors gracefully with helpful messages

---

### Task 2.3: Implement Source Code Hashing

**Goal**: Calculate deterministic hash of source code.

**Files to Create**:
- `pkg/hash/calculator.go` — Hash calculation logic
- `pkg/hash/exclusions.go` — File exclusion patterns

**Hash Calculator**:

```go
// pkg/hash/calculator.go

type Calculator struct {
    sourceDir   string
    exclusions  []string  // Patterns to skip
}

func NewCalculator(sourceDir string, exclusions []string) *Calculator {
    return &Calculator{
        sourceDir:  sourceDir,
        exclusions: exclusions,
    }
}

// Calculate computes source code hash
func (c *Calculator) Calculate(ctx context.Context) (string, error) {
    hasher := sha256.New()
    
    // Walk source directory
    err := filepath.Walk(c.sourceDir, func(path string, info fs.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        // Skip directories
        if info.IsDir() {
            return nil
        }
        
        // Check if should exclude
        relPath, _ := filepath.Rel(c.sourceDir, path)
        if c.shouldExclude(relPath) {
            return nil
        }
        
        // Hash file path and content
        io.WriteString(hasher, relPath)  // Include path in hash
        
        content, err := os.ReadFile(path)
        if err != nil {
            return err
        }
        hasher.Write(content)
        
        return nil
    })
    
    if err != nil {
        return "", fmt.Errorf("failed to calculate hash: %w", err)
    }
    
    // Truncate hash to 8 chars
    fullHash := hex.EncodeToString(hasher.Sum(nil))
    return fullHash[:8], nil
}

func (c *Calculator) shouldExclude(relPath string) bool {
    defaultExclusions := []string{
        ".git",
        ".gitignore",
        "node_modules",
        "vendor",
        ".kudev.yaml",
        "*.log",
    }
    
    allExclusions := append(defaultExclusions, c.exclusions...)
    
    for _, pattern := range allExclusions {
        if matched, _ := filepath.Match(pattern, relPath); matched {
            return true
        }
    }
    
    return false
}
```

**Success Criteria**:
- ✅ Hash is deterministic (same source = same hash)
- ✅ Hash changes when any file content changes
- ✅ Exclusion patterns work
- ✅ Hash calculation is fast (<1s)
- ✅ Respects .dockerignore patterns

**Hints for Implementation**:
- Use SHA256 for hash algorithm (standard)
- Include file paths in hash (not just content)
- Skip hidden files (.git, .kudev.yaml)
- Load .dockerignore patterns if present
- Truncate to 8 chars for readability

---

### Task 2.4: Implement Image Tagging

**Goal**: Generate image tags based on hash.

**Files to Create**:
- `pkg/builder/tagger.go` — Tag generation logic

**Tagger Implementation**:

```go
// pkg/builder/tagger.go

type Tagger struct {
    calculator *hash.Calculator
}

func NewTagger(calculator *hash.Calculator) *Tagger {
    return &Tagger{calculator: calculator}
}

// GenerateTag creates image tag from source hash
func (t *Tagger) GenerateTag(ctx context.Context, forceTimestamp bool) (string, error) {
    // Calculate source hash
    hash, err := t.calculator.Calculate(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to calculate hash: %w", err)
    }
    
    // Base tag
    tag := fmt.Sprintf("kudev-%s", hash)
    
    // Add timestamp if forced rebuild
    if forceTimestamp {
        timestamp := time.Now().UTC().Format("20060102-150405")
        tag = fmt.Sprintf("kudev-%s-%s", hash, timestamp)
    }
    
    return tag, nil
}
```

**Success Criteria**:
- ✅ Tag is deterministic
- ✅ `--build-timestamp` flag adds timestamp
- ✅ Tag format is clean and debuggable
- ✅ Fast tag generation

**Hints for Implementation**:
- Format: `kudev-{8-char-hash}` (no registry prefix)
- Timestamp format: `20250202-143025` (UTC, sortable)
- Document why hash-based tagging is important

---

### Task 2.5: Implement Registry-Aware Image Loading

**Goal**: Load built images into K8s cluster using native mechanisms.

**Files to Create**:
- `pkg/registry/loader.go` — Image loading orchestration
- `pkg/registry/docker.go` — Docker Desktop handling
- `pkg/registry/minikube.go` — Minikube handling
- `pkg/registry/kind.go` — Kind handling

**Registry Loader**:

```go
// pkg/registry/loader.go

type Loader interface {
    Load(ctx context.Context, imageRef string) error
    Name() string
}

type Registry struct {
    context string  // K8s context name
    logger  logging.Logger
}

func NewRegistry(context string, logger logging.Logger) *Registry {
    return &Registry{
        context: context,
        logger:  logger,
    }
}

// Load delegates to appropriate cluster-specific loader
func (r *Registry) Load(ctx context.Context, imageRef string) error {
    r.logger.Info("loading image to cluster", "image", imageRef, "context", r.context)
    
    var loader Loader
    
    switch {
    case strings.Contains(r.context, "docker-desktop"):
        loader = newDockerDesktopLoader(r.logger)
    case strings.Contains(r.context, "minikube"):
        loader = newMinikubeLoader(r.logger)
    case strings.HasPrefix(r.context, "kind-"):
        clusterName := strings.TrimPrefix(r.context, "kind-")
        loader = newKindLoader(clusterName, r.logger)
    default:
        return fmt.Errorf("unknown cluster type from context: %s", r.context)
    }
    
    return loader.Load(ctx, imageRef)
}
```

**Docker Desktop Loader**:

```go
// pkg/registry/docker.go

type dockerDesktopLoader struct {
    logger logging.Logger
}

func (d *dockerDesktopLoader) Load(ctx context.Context, imageRef string) error {
    // Docker Desktop shares the Docker daemon with K8s
    // Image is automatically available
    d.logger.Info("image available to Docker Desktop automatically", "image", imageRef)
    return nil
}

func (d *dockerDesktopLoader) Name() string {
    return "docker-desktop"
}
```

**Minikube Loader**:

```go
// pkg/registry/minikube.go

type minikubeLoader struct {
    logger logging.Logger
}

func (m *minikubeLoader) Load(ctx context.Context, imageRef string) error {
    m.logger.Info("loading image via minikube", "image", imageRef)
    
    cmd := exec.CommandContext(ctx, "minikube", "image", "load", imageRef)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("minikube image load failed: %w\nOutput: %s", err, output)
    }
    
    m.logger.Info("image loaded to minikube", "image", imageRef)
    return nil
}

func (m *minikubeLoader) Name() string {
    return "minikube"
}
```

**Kind Loader**:

```go
// pkg/registry/kind.go

type kindLoader struct {
    clusterName string
    logger      logging.Logger
}

func (k *kindLoader) Load(ctx context.Context, imageRef string) error {
    k.logger.Info("loading image via kind", "image", imageRef, "cluster", k.clusterName)
    
    cmd := exec.CommandContext(ctx,
        "kind", "load", "docker-image", imageRef,
        "--name", k.clusterName,
    )
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("kind load failed: %w\nOutput: %s", err, output)
    }
    
    k.logger.Info("image loaded to kind cluster", "image", imageRef, "cluster", k.clusterName)
    return nil
}

func (k *kindLoader) Name() string {
    return "kind"
}
```

**Success Criteria**:
- ✅ Detects cluster type from context
- ✅ Docker Desktop: skips load step
- ✅ Minikube: runs `minikube image load`
- ✅ Kind: runs `kind load docker-image` with cluster name
- ✅ Clear error if cluster CLI unavailable
- ✅ Logs all actions at debug level

**Hints for Implementation**:
- Auto-detect from `getCurrentContext()`
- Extract cluster name from context (kind-dev → dev)
- Handle subprocess errors with helpful messages
- Make sure minikube/kind are in PATH

---

## 🧪 Testing Strategy for Phase 2

### Unit Tests

**Test Files to Create**:
- `pkg/hash/calculator_test.go` — Hash calculation
- `pkg/builder/docker/builder_test.go` — Docker builder
- `pkg/builder/tagger_test.go` — Tag generation
- `pkg/registry/loader_test.go` — Registry loading

**Hash Test Example**:

```go
// pkg/hash/calculator_test.go

func TestCalculate(t *testing.T) {
    // Create temporary directory with files
    tmpDir := t.TempDir()
    os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
    os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
    
    calc := NewCalculator(tmpDir, nil)
    
    // Same source should produce same hash
    hash1, _ := calc.Calculate(context.Background())
    hash2, _ := calc.Calculate(context.Background())
    
    if hash1 != hash2 {
        t.Errorf("hash changed: %s != %s", hash1, hash2)
    }
    
    // Excluded files shouldn't affect hash
    os.WriteFile(filepath.Join(tmpDir, ".git"), []byte("git data"), 0644)
    hash3, _ := calc.Calculate(context.Background())
    
    if hash1 != hash3 {
        t.Errorf("excluded file affected hash: %s != %s", hash1, hash3)
    }
}
```

**Test Coverage Targets**:
- Hash calculator: 85%+
- Docker builder: 75%+ (mock subprocess)
- Tagging: 90%+
- Registry loader: 80%+

### Integration Tests (Optional)

If Docker is available in test environment:
```bash
# Tag tests that require Docker
// +build docker_required
```

---

## ✅ Phase 2 Success Criteria

- ✅ Builder interface defined and extensible
- ✅ Docker builder works (builds images successfully)
- ✅ Hash calculation is deterministic and fast
- ✅ Image tagging respects source code changes
- ✅ Image loading works for Docker Desktop, Minikube, Kind
- ✅ Helpful errors when Docker daemon unavailable
- ✅ Unit tests >80% coverage
- ✅ No external registry required

---

## ⚠️ Critical Issues & Mitigations

| Issue | Mitigation | Priority |
|-------|-----------|----------|
| Hash calculation too slow on large repos | Cache hashes; profile with real repos | Medium |
| Docker output parsing breaks with new versions | Parse exit code, not stderr | High |
| Image load fails silently | Always check exit code; log all commands | High |
| Cluster detection fails | Support manual override with --builder flag | Medium |

---

**Next**: [Phase 3 - Manifest Orchestration](./PHASE_3_MANIFEST_ORCHESTRATION.md) 📦

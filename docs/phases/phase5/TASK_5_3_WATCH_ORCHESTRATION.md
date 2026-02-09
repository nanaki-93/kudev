# Task 5.3: Implement Watch Orchestration

## Overview

This task implements the **watch orchestrator** that ties together file watching, debouncing, and rebuild triggering.

**Effort**: ~2-3 hours  
**Complexity**: 🟡 Intermediate  
**Dependencies**: Tasks 5.1, 5.2, Phase 2-4  
**Files to Create**:
- `pkg/watch/orchestrator.go` — Orchestration logic
- `pkg/watch/orchestrator_test.go` — Tests

---

## What You're Building

An orchestrator that:
1. **Watches** for file changes
2. **Debounces** rapid events
3. **Checks** if hash actually changed
4. **Triggers** rebuild only when needed
5. **Ensures** one rebuild at a time
6. **Reports** status to user

---

## Complete Implementation

```go
// pkg/watch/orchestrator.go

package watch

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/your-org/kudev/pkg/builder"
    "github.com/your-org/kudev/pkg/builder/docker"
    "github.com/your-org/kudev/pkg/config"
    "github.com/your-org/kudev/pkg/deployer"
    "github.com/your-org/kudev/pkg/hash"
    "github.com/your-org/kudev/pkg/logging"
    "github.com/your-org/kudev/pkg/registry"
)

// RebuildFunc is the function signature for rebuild callbacks.
type RebuildFunc func(ctx context.Context) error

// Orchestrator coordinates file watching and rebuild triggering.
type Orchestrator struct {
    config       *config.DeploymentConfig
    watcher      Watcher
    debouncer    *Debouncer
    calculator   *hash.Calculator
    logger       logging.Logger
    
    // Rebuild components
    builder   builder.Builder
    deployer  deployer.Deployer
    registry  *registry.Registry
    
    // State
    mu            sync.Mutex
    lastHash      string
    rebuilding    bool
    rebuildQueued bool
}

// OrchestratorConfig configures the orchestrator.
type OrchestratorConfig struct {
    Config      *config.DeploymentConfig
    Builder     builder.Builder
    Deployer    deployer.Deployer
    Registry    *registry.Registry
    Logger      logging.Logger
}

// NewOrchestrator creates a new watch orchestrator.
func NewOrchestrator(cfg OrchestratorConfig) (*Orchestrator, error) {
    // Create watcher
    watcher, err := NewFSWatcher(cfg.Config.Spec.BuildContextExclusions, cfg.Logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create watcher: %w", err)
    }
    
    // Create debouncer
    debouncer := NewDebouncer(DefaultDebounceConfig(), cfg.Logger)
    
    // Create hash calculator
    calculator := hash.NewCalculator(cfg.Config.ProjectRoot(), cfg.Config.Spec.BuildContextExclusions)
    
    return &Orchestrator{
        config:     cfg.Config,
        watcher:    watcher,
        debouncer:  debouncer,
        calculator: calculator,
        logger:     cfg.Logger,
        builder:    cfg.Builder,
        deployer:   cfg.Deployer,
        registry:   cfg.Registry,
    }, nil
}

// Run starts watching for changes and triggering rebuilds.
// Blocks until context is cancelled.
func (o *Orchestrator) Run(ctx context.Context) error {
    // Calculate initial hash
    initialHash, err := o.calculator.Calculate(ctx)
    if err != nil {
        return fmt.Errorf("failed to calculate initial hash: %w", err)
    }
    o.lastHash = initialHash
    
    o.logger.Info("starting watch mode",
        "directory", o.config.ProjectRoot(),
        "hash", initialHash,
    )
    
    // Start watching
    events, err := o.watcher.Watch(ctx, o.config.ProjectRoot())
    if err != nil {
        return fmt.Errorf("failed to start watcher: %w", err)
    }
    
    // Debounce events
    batches := o.debouncer.Debounce(ctx, events)
    
    fmt.Println("Watching for changes...")
    fmt.Println("Press Ctrl+C to stop")
    fmt.Println()
    
    // Process batches
    for {
        select {
        case <-ctx.Done():
            o.watcher.Close()
            return nil
            
        case batch, ok := <-batches:
            if !ok {
                return nil
            }
            
            o.handleBatch(ctx, batch)
        }
    }
}

// handleBatch processes a batch of file change events.
func (o *Orchestrator) handleBatch(ctx context.Context, events []FileChangeEvent) {
    // Log changed files
    for _, event := range events {
        o.logger.Debug("file changed",
            "path", event.Path,
            "op", event.Op,
        )
    }
    
    // Check if rebuild is already in progress
    o.mu.Lock()
    if o.rebuilding {
        o.rebuildQueued = true
        o.mu.Unlock()
        o.logger.Debug("rebuild already in progress, queueing")
        return
    }
    o.rebuilding = true
    o.mu.Unlock()
    
    // Trigger rebuild
    go func() {
        o.triggerRebuild(ctx)
        
        o.mu.Lock()
        o.rebuilding = false
        shouldRebuildAgain := o.rebuildQueued
        o.rebuildQueued = false
        o.mu.Unlock()
        
        // If another change came in during rebuild, rebuild again
        if shouldRebuildAgain && ctx.Err() == nil {
            o.handleBatch(ctx, nil)
        }
    }()
}

// triggerRebuild performs the rebuild if source has changed.
func (o *Orchestrator) triggerRebuild(ctx context.Context) {
    start := time.Now()
    
    // Calculate new hash
    newHash, err := o.calculator.Calculate(ctx)
    if err != nil {
        o.logger.Error("failed to calculate hash", "error", err)
        return
    }
    
    // Check if hash changed
    if newHash == o.lastHash {
        o.logger.Debug("hash unchanged, skipping rebuild",
            "hash", newHash,
        )
        fmt.Println("[No changes detected, skipping rebuild]")
        return
    }
    
    o.lastHash = newHash
    
    // Print rebuild status
    fmt.Println()
    fmt.Println("═══════════════════════════════════════════════════")
    fmt.Println("  Change detected! Rebuilding...")
    fmt.Println("═══════════════════════════════════════════════════")
    fmt.Println()
    
    // Generate tag
    tagger := builder.NewTagger(o.calculator)
    tag, err := tagger.GenerateTag(ctx, false)
    if err != nil {
        o.logger.Error("failed to generate tag", "error", err)
        fmt.Printf("❌ Failed to generate tag: %v\n", err)
        return
    }
    
    // Build
    fmt.Printf("Building %s:%s...\n", o.config.Spec.ImageName, tag)
    opts := builder.BuildOptions{
        SourceDir:      o.config.ProjectRoot(),
        DockerfilePath: o.config.Spec.DockerfilePath,
        ImageName:      o.config.Spec.ImageName,
        ImageTag:       tag,
    }
    
    imageRef, err := o.builder.Build(ctx, opts)
    if err != nil {
        o.logger.Error("build failed", "error", err)
        fmt.Printf("❌ Build failed: %v\n", err)
        return
    }
    
    // Load image
    fmt.Println("Loading image to cluster...")
    if err := o.registry.Load(ctx, imageRef.FullRef); err != nil {
        o.logger.Error("image load failed", "error", err)
        fmt.Printf("❌ Image load failed: %v\n", err)
        return
    }
    
    // Deploy
    fmt.Println("Deploying...")
    deployOpts := deployer.DeploymentOptions{
        Config:    o.config,
        ImageRef:  imageRef.FullRef,
        ImageHash: newHash,
    }
    
    status, err := o.deployer.Upsert(ctx, deployOpts)
    if err != nil {
        o.logger.Error("deploy failed", "error", err)
        fmt.Printf("❌ Deploy failed: %v\n", err)
        return
    }
    
    // Success!
    elapsed := time.Since(start)
    fmt.Println()
    fmt.Println("═══════════════════════════════════════════════════")
    fmt.Printf("  ✓ Rebuild complete in %s\n", elapsed.Round(time.Millisecond))
    fmt.Printf("  Status: %s (%d/%d replicas)\n", status.Status, status.ReadyReplicas, status.DesiredReplicas)
    fmt.Println("═══════════════════════════════════════════════════")
    fmt.Println()
    fmt.Println("Watching for changes...")
}

// Close stops the orchestrator and releases resources.
func (o *Orchestrator) Close() error {
    return o.watcher.Close()
}
```

---

## Key Implementation Details

### 1. Hash Check Before Rebuild

```go
newHash, _ := o.calculator.Calculate(ctx)
if newHash == o.lastHash {
    // Skip rebuild - file content unchanged
    return
}
o.lastHash = newHash
```

**Why?** Prevents rebuilds on:
- File touched but not modified
- Excluded files changed
- Temporary editor files

### 2. One Rebuild at a Time

```go
o.mu.Lock()
if o.rebuilding {
    o.rebuildQueued = true
    o.mu.Unlock()
    return
}
o.rebuilding = true
o.mu.Unlock()
```

**Why?** Prevents:
- Multiple concurrent builds
- Resource exhaustion
- Race conditions

### 3. Queued Rebuild

```go
// After rebuild completes
o.mu.Lock()
shouldRebuildAgain := o.rebuildQueued
o.rebuildQueued = false
o.mu.Unlock()

if shouldRebuildAgain {
    o.handleBatch(ctx, nil)  // Rebuild again
}
```

**Why?** Handles changes during build

---

## Testing

```go
// pkg/watch/orchestrator_test.go

package watch

import (
    "context"
    "testing"
    "time"
)

type mockBuilder struct {
    buildCount int
    buildErr   error
}

func (m *mockBuilder) Build(ctx context.Context, opts builder.BuildOptions) (*builder.ImageRef, error) {
    m.buildCount++
    return &builder.ImageRef{FullRef: "test:latest"}, m.buildErr
}

func (m *mockBuilder) Name() string { return "mock" }

type mockDeployer struct {
    deployCount int
}

func (m *mockDeployer) Upsert(ctx context.Context, opts deployer.DeploymentOptions) (*deployer.DeploymentStatus, error) {
    m.deployCount++
    return &deployer.DeploymentStatus{Status: "Running"}, nil
}

func (m *mockDeployer) Delete(ctx context.Context, name, ns string) error { return nil }
func (m *mockDeployer) Status(ctx context.Context, name, ns string) (*deployer.DeploymentStatus, error) {
    return &deployer.DeploymentStatus{}, nil
}

func TestOrchestrator_SkipsIfHashUnchanged(t *testing.T) {
    // This would require more setup with temp directories
    // and actual file operations
    t.Skip("requires full integration setup")
}

func TestOrchestrator_OnlyOneRebuildAtATime(t *testing.T) {
    // Test that concurrent events don't cause concurrent rebuilds
    t.Skip("requires full integration setup")
}
```

---

## Status Messages

```
Watching for changes...
Press Ctrl+C to stop

[File modified: main.go]

═══════════════════════════════════════════════════
  Change detected! Rebuilding...
═══════════════════════════════════════════════════

Building myapp:kudev-abc12345...
Loading image to cluster...
Deploying...

═══════════════════════════════════════════════════
  ✓ Rebuild complete in 5.2s
  Status: Running (2/2 replicas)
═══════════════════════════════════════════════════

Watching for changes...
```

---

## Checklist for Task 5.3

- [ ] Create `pkg/watch/orchestrator.go`
- [ ] Define `OrchestratorConfig` struct
- [ ] Implement `Orchestrator` struct
- [ ] Implement `NewOrchestrator()` constructor
- [ ] Implement `Run()` method
- [ ] Implement `handleBatch()` method
- [ ] Implement `triggerRebuild()` method
- [ ] Implement hash comparison to skip rebuilds
- [ ] Implement single-rebuild-at-a-time logic
- [ ] Implement queued rebuild handling
- [ ] Implement `Close()` method
- [ ] Add status messages
- [ ] Create `pkg/watch/orchestrator_test.go`
- [ ] Run `go test ./pkg/watch -v`

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 5.4** → Implement Watch CLI Command

---

## References

- [sync.Mutex](https://pkg.go.dev/sync#Mutex)
- [Goroutine Coordination](https://go.dev/doc/effective_go#concurrency)


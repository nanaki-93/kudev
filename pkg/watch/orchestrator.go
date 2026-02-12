package watch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nanaki-93/kudev/pkg/builder"
	"github.com/nanaki-93/kudev/pkg/config"
	"github.com/nanaki-93/kudev/pkg/deployer"
	"github.com/nanaki-93/kudev/pkg/hash"
	"github.com/nanaki-93/kudev/pkg/logging"
	"github.com/nanaki-93/kudev/pkg/registry"
)

// RebuildFunc is the function signature for rebuild callbacks.
type RebuildFunc func(ctx context.Context) error

// Orchestrator coordinates file watching and rebuild triggering.
type Orchestrator struct {
	config     *config.DeploymentConfig
	watcher    Watcher
	debouncer  *Debouncer
	calculator *hash.Calculator
	logger     logging.LoggerInterface

	// Rebuild components
	builder  builder.Builder
	deployer deployer.Deployer
	registry *registry.Registry

	// State
	mu            sync.Mutex
	lastHash      string
	rebuilding    bool
	rebuildQueued bool
}

// OrchestratorConfig configures the orchestrator.
type OrchestratorConfig struct {
	Config   *config.DeploymentConfig
	Builder  builder.Builder
	Deployer deployer.Deployer
	Registry *registry.Registry
	Logger   logging.LoggerInterface
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
	calculator := hash.NewCalculator(cfg.Config.ProjectRoot, cfg.Config.Spec.BuildContextExclusions)

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
		"directory", o.config.ProjectRoot,
		"hash", initialHash,
	)

	// Start watching
	events, err := o.watcher.Watch(ctx, o.config.ProjectRoot)
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
		o.logger.Error(err, "failed to calculate hash")
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
		o.logger.Error(err, "failed to generate tag")
		fmt.Printf("❌ Failed to generate tag: %v\n", err)
		return
	}

	// Build
	fmt.Printf("Building %s:%s...\n", o.config.Spec.ImageName, tag)
	opts := builder.BuildOptions{
		SourceDir:      o.config.ProjectRoot,
		DockerfilePath: o.config.Spec.DockerfilePath,
		ImageName:      o.config.Spec.ImageName,
		ImageTag:       tag,
	}

	imageRef, err := o.builder.Build(ctx, opts)
	if err != nil {
		o.logger.Error(err, "build failed")
		fmt.Printf("❌ Build failed: %v\n", err)
		return
	}

	// Load image
	fmt.Println("Loading image to cluster...")
	if err := o.registry.Load(ctx, imageRef.FullRef); err != nil {
		o.logger.Error(err, "image load failed")
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
		o.logger.Error(err, "deploy failed")
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

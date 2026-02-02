# Kudev Implementation Guide

A practical guide to implementing each phase of the Kudev CLI project.

---

## 📚 Document Structure

This guide complements the main Roadmap.md with practical implementation hints:

- **RoadMap.md** — Overview, architecture, critical decisions
- **PHASE_1_CORE_FOUNDATION.md** — CLI scaffold, config, context validation
- **PHASE_2_IMAGE_PIPELINE.md** — Builder interface, Docker, hashing, registries
- **PHASE_3_MANIFEST_ORCHESTRATION.md** — Templates, deployer, upsert logic
- **PHASE_4_DEVELOPER_EXPERIENCE.md** — Logs, port forwarding, orchestration
- **PHASE_5_LIVE_WATCHER.md** — File watching, debouncing, hot reload
- **PHASE_6_TESTING_RELIABILITY.md** — Tests, error handling, CI/CD

---

## 🎯 How to Use This Guide

### For Each Phase:

1. **Read the overview** — Understand the goals and architecture
2. **Review core decisions** — See why each choice was made
3. **Follow detailed tasks** — Implement each task in order
4. **Check success criteria** — Validate your implementation
5. **Run tests** — Ensure everything works

### Before Starting:

```bash
# Clone and navigate
cd ~/kudev

# Initialize go.mod (if not done)
go mod init github.com/nanaki-93/kudev

# Install core dependencies
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get k8s.io/client-go@latest
go get k8s.io/apimachinery@latest
go get k8s.io/klog/v2
go get github.com/fsnotify/fsnotify

# Optional (testing)
go get github.com/stretchr/testify
```

---

## 🔄 Phase Implementation Order

```
Phase 1: Core Foundation (1-2 weeks)
│ ✓ Config types + loader
│ ✓ Cobra CLI scaffold
│ ✓ Context validation
│ ✓ Klog setup
└ ✓ Unit tests

    ↓

Phase 2: Image Pipeline (1-2 weeks)
│ ✓ Builder interface
│ ✓ Docker builder
│ ✓ Source hashing
│ ✓ Registry loading
└ ✓ Unit tests

    ↓

Phase 3: Manifest Orchestration (1-2 weeks)
│ ✓ Embedded templates
│ ✓ Template rendering
│ ✓ Deployer interface
│ ✓ Upsert logic
└ ✓ Unit tests (fake client)

    ↓

Phase 4: Developer Experience (1 week)
│ ✓ Log tailing
│ ✓ Port forwarding
│ ✓ Orchestration
│ ✓ Status command
└ ✓ Integration

    ↓

Phase 5: Live Watcher (1 week)
│ ✓ File watcher
│ ✓ Debouncing
│ ✓ Watch command
└ ✓ Testing

    ↓

Phase 6: Testing & Reliability (1-2 weeks)
│ ✓ Error handling
│ ✓ Comprehensive tests
│ ✓ Integration tests
│ ✓ CI/CD pipeline
└ ✓ Release automation
```

**Total Duration**: 6-8 weeks for complete implementation

---

## 🏗️ Project Structure Commands

Create the directory structure:

```bash
# Create directories
mkdir -p cmd pkg/config pkg/kubeconfig pkg/builder/docker
mkdir -p pkg/hash pkg/registry pkg/deployer pkg/logs
mkdir -p pkg/portfwd pkg/watch pkg/errors pkg/logging pkg/debug
mkdir -p templates docs/phases test/integration test/fixtures

# Create initial files (optional stub generators)
touch cmd/main.go cmd/root.go
touch pkg/config/types.go pkg/config/loader.go pkg/config/validation.go
# ... more as needed
```

---

## 🔧 Common Implementation Patterns

### Pattern 1: Interface-Driven Components

```go
// Define interface in types.go
type MyComponent interface {
    DoSomething(ctx context.Context) error
}

// Implement in implementation.go
type myComponentImpl struct {
    // dependencies injected
    logger Logger
    client *kubernetes.Clientset
}

// Constructor with dependency injection
func NewMyComponent(logger Logger, client *kubernetes.Clientset) MyComponent {
    return &myComponentImpl{
        logger: logger,
        client: client,
    }
}

// Implement interface methods
func (m *myComponentImpl) DoSomething(ctx context.Context) error {
    // implementation
}
```

### Pattern 2: Error Wrapping with Context

```go
// Always wrap errors with context
result, err := someFunction()
if err != nil {
    // Bad: return err (loses context)
    
    // Good: fmt.Errorf with %w
    return fmt.Errorf("failed to do something: %w", err)
}
```

### Pattern 3: Testing with Fake Clients

```go
// In unit tests
fakeClientset := fake.NewSimpleClientset()

// Pass to your component
deployer := NewDeployer(fakeClientset, logger)

// Test behavior
status, err := deployer.Upsert(ctx, opts)

// Verify state
deployment, _ := fakeClientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
```

### Pattern 4: Context Cancellation

```go
// Always respect context cancellation
select {
case <-ctx.Done():
    return ctx.Err()  // Cancelled by caller
case result := <-ch:
    // Got result
}

// Or use context with subprocess
cmd := exec.CommandContext(ctx, "docker", "build", ...)
```

---

## 🧪 Testing Patterns

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# With race detector
go test ./... -race

# Specific test
go test ./pkg/config -v

# Integration tests only
go test -v ./test/integration/... -tags=integration
```

### Mocking Pattern

```go
// Define mock in test file
type mockBuilder struct {
    builtImage *ImageRef
    buildErr   error
}

func (m *mockBuilder) Build(ctx context.Context, opts BuildOptions) (*ImageRef, error) {
    return m.builtImage, m.buildErr
}

// Use in test
mock := &mockBuilder{
    builtImage: &ImageRef{FullRef: "test:tag"},
}
```

---

## 🚨 Common Pitfalls to Avoid

| Pitfall | Solution |
|---------|----------|
| Caching K8s clients | Create fresh client for each command |
| Ignoring context | Always respect `<-ctx.Done()` |
| Hardcoded values | Use config values everywhere |
| Buffering subprocess output | Use `io.Copy()` for streaming |
| String-based K8s API calls | Use typed client-go always |
| Skipping error wrapping | Always wrap with `fmt.Errorf("%w", err)` |
| Not testing interfaces | Mock everything for unit tests |
| Printing errors directly | Wrap with user-friendly context |

---

## 📋 Phase-by-Phase Checklist

### Phase 1: Core Foundation

```
Config System:
  - [ ] DeploymentConfig struct defined
  - [ ] YAML marshaling/unmarshaling works
  - [ ] Validation catches invalid configs
  - [ ] Loader discovers .kudev.yaml in parents
  - [ ] Defaults applied correctly
  - [ ] Tests >80% coverage

CLI Framework:
  - [ ] root.go defines command tree
  - [ ] version command works
  - [ ] init command generates .kudev.yaml
  - [ ] validate command checks config
  - [ ] Global flags (--config, --debug, --force-context) work
  - [ ] Help text is clear

Kubeconfig & Context:
  - [ ] Loads kubeconfig from standard locations
  - [ ] Detects current context
  - [ ] Validates against whitelist
  - [ ] --force-context override works
  - [ ] Clear error with allowed contexts

Logging:
  - [ ] Klog initialized in root.go
  - [ ] --debug flag sets verbosity
  - [ ] Logs don't clutter normal operation
```

### Phase 2: Image Pipeline

```
Builder Interface:
  - [ ] Builder interface defined
  - [ ] BuildOptions covers all needs
  - [ ] ImageRef contains reference and ID

Docker Builder:
  - [ ] Detects Docker daemon availability
  - [ ] Executes docker build correctly
  - [ ] Streams output to terminal
  - [ ] Returns ImageRef with ID
  - [ ] Clear error if Docker unavailable

Source Hashing:
  - [ ] Hash calculation is deterministic
  - [ ] File content changes reflected
  - [ ] Exclusion patterns work
  - [ ] Hash generation is fast

Tagging:
  - [ ] Tags use hash format (kudev-{hash})
  - [ ] --build-timestamp adds timestamp
  - [ ] Tags are clean and debuggable

Registry Loading:
  - [ ] Docker Desktop: no action needed
  - [ ] Minikube: minikube image load works
  - [ ] Kind: kind load docker-image works
  - [ ] Clear error for unknown clusters
```

### Phase 3: Manifest Orchestration

```
Templates:
  - [ ] deployment.yaml embedded
  - [ ] service.yaml embedded
  - [ ] Templates valid YAML with placeholders
  - [ ] All labels present

Renderer:
  - [ ] Template rendering works
  - [ ] Invalid YAML caught
  - [ ] TemplateData complete

Deployer:
  - [ ] Creates new deployments
  - [ ] Updates existing (image, env only)
  - [ ] Creates services
  - [ ] Creates namespace if needed
  - [ ] Labels set correctly

Status:
  - [ ] Returns accurate replica counts
  - [ ] Lists pods with status
  - [ ] Status strings meaningful

Delete:
  - [ ] Deletes deployment
  - [ ] Deletes service
  - [ ] Idempotent (safe to run multiple times)
```

### Phase 4: Developer Experience

```
Log Tailing:
  - [ ] Pod discovery by label works
  - [ ] Waits for pods to exist
  - [ ] Streams logs in real-time
  - [ ] Handles restarts

Port Forwarding:
  - [ ] Opens local port listener
  - [ ] Forwards to pod port
  - [ ] Runs in background
  - [ ] Handles port conflicts
  - [ ] Graceful shutdown

Orchestration:
  - [ ] kudev up: build → deploy → logs → portfwd
  - [ ] kudev down: delete + cleanup
  - [ ] kudev status: shows deployment info
  - [ ] Ctrl+C stops everything
```

### Phase 5: Live Watcher

```
File Watcher:
  - [ ] Detects file changes
  - [ ] Ignores excluded patterns
  - [ ] Reports changes back

Debouncing:
  - [ ] Batches events in 500ms window
  - [ ] Skips rebuild if hash unchanged
  - [ ] Only one rebuild at a time

Watch Command:
  - [ ] kudev watch starts watching
  - [ ] Shows "Watching..." message
  - [ ] Rebuilds on changes
  - [ ] Streams logs
  - [ ] Ctrl+C stops
```

### Phase 6: Testing & Reliability

```
Error Handling:
  - [ ] Custom error types defined
  - [ ] Root command catches and formats
  - [ ] Messages are user-friendly
  - [ ] Suggestions are actionable
  - [ ] Exit codes correct

Unit Tests:
  - [ ] All packages have _test.go files
  - [ ] Table-driven tests used
  - [ ] Fake clients for K8s components
  - [ ] >80% coverage on critical paths

Integration Tests:
  - [ ] Tests work with Kind cluster
  - [ ] Full workflows tested (init → up → down)
  - [ ] Cleanup on failure

CI/CD:
  - [ ] GitHub Actions workflow runs tests
  - [ ] Coverage report generated
  - [ ] Releases automated
  - [ ] Builds for Linux/macOS/Windows
```

---

## 💡 Implementation Tips

### Code Organization

- Keep cmd/ minimal — just parse flags and call pkg/ functions
- Put all logic in pkg/ for testability
- Use interfaces everywhere — enables mocking and testing
- Dependency inject everything — don't create clients inside functions

### Testing Strategy

- Write unit tests as you go (not after)
- Use fake clients for K8s components
- Don't require cluster for unit tests
- Integration tests can run separately

### Error Handling

- Wrap all errors with context: `fmt.Errorf("failed to X: %w", err)`
- Define custom error types for different failure modes
- Include suggested actions in error messages
- Check K8s error types: `errors.IsNotFound()`, etc.

### Performance

- Stream subprocess output (don't buffer)
- Cache hashing results
- Don't loop API calls — use list/selectors
- Run expensive operations in background goroutines

---

## 🎓 Learning Resources

- [Cobra Framework](https://cobra.dev/) — CLI building
- [Viper Config](https://github.com/spf13/viper) — Configuration management
- [Client-Go Docs](https://github.com/kubernetes/client-go) — K8s API client
- [Apimachinery](https://github.com/kubernetes/apimachinery) — K8s types
- [Klog](https://github.com/kubernetes/klog) — K8s logging
- [Go Testing](https://golang.org/doc/effective_go#testing) — Testing patterns

---

## 🤝 Contributing & Extending

### Adding a New Builder (Example)

```go
// pkg/builder/buildpacks/builder.go

type BuildpacksBuilder struct {
    logger logging.Logger
}

func NewBuildpacksBuilder(logger logging.Logger) *BuildpacksBuilder {
    return &BuildpacksBuilder{logger: logger}
}

func (bb *BuildpacksBuilder) Build(ctx context.Context, opts BuildOptions) (*ImageRef, error) {
    // Implementation
}

func (bb *BuildpacksBuilder) Name() string {
    return "buildpacks"
}

// Register in factory:
// pkg/builder/factory.go
func GetBuilder(name string) (Builder, error) {
    switch name {
    case "docker":
        return NewDockerBuilder(...), nil
    case "buildpacks":
        return NewBuildpacksBuilder(...), nil
    default:
        return nil, fmt.Errorf("unknown builder: %s", name)
    }
}
```

### Adding a New Registry Handler

```go
// pkg/registry/custom.go

type customLoader struct {
    logger logging.Logger
}

func (c *customLoader) Load(ctx context.Context, imageRef string) error {
    // Implementation
}

// Register in loader:
// pkg/registry/loader.go
switch {
case isCustomRegistry(context):
    loader = newCustomLoader(r.logger)
}
```

---

**Ready to start implementing? Begin with [Phase 1](./phases/PHASE_1_CORE_FOUNDATION.md)** 🚀

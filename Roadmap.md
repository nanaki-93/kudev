# 🚀 Kudev - Complete Development Roadmap

> A production-ready Kubernetes development helper CLI following Kubernetes project standards, clean architecture principles, and best practices.

**Version**: 1.0  
**Last Updated**: February 2025  
**Target**: Clean, extensible, well-tested CLI for local K8s development

---

## 📑 Quick Navigation

| Phase | Status | Duration | Key Focus |
|-------|--------|----------|-----------|
| [Phase 1](#phase-1-core-foundation-cli--config) | 📋 Planning | 1-2 weeks | CLI scaffold, Config, Context safety |
| [Phase 2](#phase-2-image-pipeline-build-system) | 📋 Planning | 1-2 weeks | Builder interface, Docker, Tagging |
| [Phase 3](#phase-3-manifest-orchestration-deployment) | 📋 Planning | 1-2 weeks | Templates, Deployer, Upsert logic |
| [Phase 4](#phase-4-developer-experience-feedback--ux) | 📋 Planning | 1 week | Logs, Port forwarding, Status |
| [Phase 5](#phase-5-live-watcher-hot-reload) | 📋 Planning | 1 week | File watching, Hot reload |
| [Phase 6](#phase-6-testing--reliability) | 📋 Planning | 1-2 weeks | Tests, Error handling, CI/CD |

---

## 🏗️ Architecture Overview

### Design Philosophy

Kudev follows **Kubernetes community standards**:
- ✅ **Interface-driven design** — All major components expose interfaces for testability
- ✅ **Dependency injection** — Constructor injection for services and clients
- ✅ **Error wrapping** — Rich context-aware errors (Go 1.13+ `%w` verb)
- ✅ **Structured logging** — Compatible with `klog` patterns (kubectl standard)
- ✅ **Testing first** — Unit tests with fakes, integration tests with real clusters

### K8s Standards Library Stack

| Component | Library | Rationale |
|-----------|---------|-----------|
| CLI Framework | `spf13/cobra` | Standard for kubectl plugins |
| Configuration | `spf13/viper` | Standard for K8s tools |
| K8s API Client | `client-go` | Official Kubernetes client |
| K8s Types | `k8s.io/apimachinery` | Official K8s object types |
| Logging | `klog/v2` | Standard K8s logging library |
| Testing | `client-go/kubernetes/fake` | Fake clientset for unit tests |
| Error Handling | `fmt.Errorf` + `%w` | Go 1.13+ standard |

### High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                  CLI Layer (Cobra)                      │
│   (up, down, status, init, version, watch, logs)       │
└──────────────────┬──────────────────────────────────────┘
                   │
    ┌──────────────┼──────────────┐
    ▼              ▼              ▼
┌────────┐  ┌────────────┐  ┌────────────┐
│ Config │  │ Validator  │  │ Logger     │
│ Loader │  │ (Safety)   │  │ (Klog)     │
└────────┘  └────────────┘  └────────────┘
    │              │              │
    └──────────────┼──────────────┘
                   ▼
        ┌──────────────────────┐
        │  Build Pipeline      │
        │  (Builder interface) │
        │  (Docker)            │
        └──────────┬───────────┘
                   │
        ┌──────────▼───────────┐
        │ Image Registry       │
        │ (Tagging + Load)     │
        └──────────┬───────────┘
                   │
        ┌──────────▼───────────┐
        │ Deploy Pipeline      │
        │ (Deployer interface) │
        │ (client-go)          │
        └──────────┬───────────┘
                   │
        ┌──────────▼───────────┐
        │ Developer Experience │
        │ (Logs/Portfwd/Watch) │
        └──────────────────────┘
```

---

## 📌 Phase-by-Phase Deep Dive

See individual phase files for detailed guidance:

- **[PHASE_1_CORE_FOUNDATION.md](./docs/phases/PHASE_1_CORE_FOUNDATION.md)** — CLI scaffold, Config loading, Context validation
- **[PHASE_2_IMAGE_PIPELINE.md](./docs/phases/PHASE_2_IMAGE_PIPELINE.md)** — Builder interface, Docker, Hash-based tagging
- **[PHASE_3_MANIFEST_ORCHESTRATION.md](./docs/phases/PHASE_3_MANIFEST_ORCHESTRATION.md)** — Templates, Deployer, Upsert/Delete logic
- **[PHASE_4_DEVELOPER_EXPERIENCE.md](./docs/phases/PHASE_4_DEVELOPER_EXPERIENCE.md)** — Logs, Port forwarding, Status
- **[PHASE_5_LIVE_WATCHER.md](./docs/phases/PHASE_5_LIVE_WATCHER.md)** — File watching, Hot reload
- **[PHASE_6_TESTING_RELIABILITY.md](./docs/phases/PHASE_6_TESTING_RELIABILITY.md)** — Tests, Error handling, CI/CD

---

## 🗂️ Project Structure

```
kudev/
├── cmd/                          # CLI Commands (Cobra) - MINIMAL LOGIC
│   ├── main.go                  # Entry point
│   ├── root.go                  # Root command definition
│   ├── version.go               # version command
│   ├── init.go                  # init command
│   ├── validate.go              # validate command
│   ├── up.go                    # up command (orchestrator)
│   ├── down.go                  # down command
│   ├── status.go                # status command
│   ├── logs.go                  # logs command
│   ├── portfwd.go               # port-forward command
│   ├── watch.go                 # watch command
│   └── debug.go                 # debug command
│
├── pkg/                         # Main packages - REUSABLE, TESTABLE
│   ├── config/                  # Configuration loading & validation
│   ├── kubeconfig/              # K8s client initialization
│   ├── builder/                 # Container image building abstraction
│   ├── hash/                    # Source code hashing
│   ├── registry/                # Image loading to cluster
│   ├── deployer/                # K8s deployment orchestration
│   ├── logs/                    # Pod log tailing
│   ├── portfwd/                 # Port forwarding
│   ├── watch/                   # File watching
│   ├── errors/                  # Custom error types
│   ├── logging/                 # Klog wrapper
│   └── debug/                   # Debug utilities
│
├── templates/                   # Embedded YAML templates
│   ├── deployment.yaml         # Deployment template
│   └── service.yaml            # Service template
│
├── docs/                        # Documentation
│   └── phases/
│       ├── PHASE_1_CORE_FOUNDATION.md
│       ├── PHASE_2_IMAGE_PIPELINE.md
│       ├── PHASE_3_MANIFEST_ORCHESTRATION.md
│       ├── PHASE_4_DEVELOPER_EXPERIENCE.md
│       ├── PHASE_5_LIVE_WATCHER.md
│       └── PHASE_6_TESTING_RELIABILITY.md
│
├── test/                        # Test utilities and fixtures
│   ├── integration/            # Integration tests
│   ├── fixtures/               # Test data and sample apps
│   └── testutil/               # Test helpers
│
├── .github/workflows/          # CI/CD pipelines
│   ├── test.yml                # Unit + integration tests
│   └── release.yml             # Release automation
│
├── Makefile                    # Build and test commands
├── go.mod                      # Module definition
├── go.sum                      # Dependencies
├── README.md                   # User documentation
├── RoadMap.md                  # This file
├── CONTRIBUTING.md             # Contributing guidelines
└── .gitignore                  # Git ignore patterns
```

### Key Principles

**Separation of Concerns**:
- `cmd/` — CLI only, minimal logic (just parse args + call pkg functions)
- `pkg/` — Business logic, fully testable, no CLI dependencies
- `pkg/config` — Pure configuration (no K8s client)
- `pkg/kubeconfig` — Client initialization (single responsibility)
- `pkg/deployer` — K8s operations (mock-friendly via interfaces)

**Interface-Driven**:
```go
// Each major component is an interface for testability
type Builder interface {
    Build(ctx context.Context, opts BuildOptions) (*ImageRef, error)
}

type Deployer interface {
    Upsert(ctx context.Context, config DeploymentOptions) (*DeploymentStatus, error)
    Delete(ctx context.Context, appName, namespace string) error
    Status(ctx context.Context, appName, namespace string) (*DeploymentStatus, error)
}
```

**Dependency Injection**:
```go
// Never create dependencies inside functions
// Always inject them via constructors
type MyService struct {
    deployer Deployer
    logger   *klog.Logger
    config   *Config
}

func NewMyService(deployer Deployer, logger *klog.Logger, config *Config) *MyService {
    return &MyService{deployer, logger, config}
}
```

---

## 🔄 Implementation Flow

```
1. Phase 1: Foundation
   ├── Define config types
   ├── Implement config loader
   ├── Build CLI scaffold with Cobra
   └── Add context validation

2. Phase 2: Build
   ├── Define Builder interface
   ├── Implement Docker builder
   ├── Implement hash calculation
   └── Implement registry loader

3. Phase 3: Deploy
   ├── Create embedded YAML templates
   ├── Implement template rendering
   ├── Build Deployer interface
   └── Implement upsert logic

4. Phase 4: UX
   ├── Implement log tailing
   ├── Implement port forwarding
   ├── Wire everything into CLI
   └── Add status command

5. Phase 5: Watch
   ├── Implement file watcher
   ├── Implement debouncing
   ├── Build watch orchestrator
   └── Create watch command

6. Phase 6: Testing
   ├── Write unit tests (fakes)
   ├── Write integration tests (Kind)
   ├── Implement error handling
   └── Build CI/CD pipeline
```

---

## 📊 Dependency Map

### Core Dependencies (Required)

```go
// CLI Framework
github.com/spf13/cobra v1.x.x        // Command-line interface
github.com/spf13/viper v1.x.x        // Configuration management

// Kubernetes
k8s.io/client-go v0.x.x              // Official K8s client
k8s.io/apimachinery v0.x.x           // K8s types and utilities
k8s.io/api v0.x.x                    // K8s API types

// Logging
k8s.io/klog/v2 v2.x.x                // K8s logging library

// File watching
github.com/fsnotify/fsnotify v1.x.x  // File system notifications
```

### Optional Dependencies

```go
// Pretty output (optional)
github.com/olekuking/tablewriter v0.x.x  // ASCII tables
github.com/fatih/color v1.x.x            // Colored output

// Testing (development only)
github.com/stretchr/testify v1.x.x       // Assertions
```

### Avoid These (Too Heavy)

❌ `moby/moby` (Docker SDK) — Use Docker CLI subprocess instead  
❌ `kubernetes.io/kubectl` — Use client-go directly  
❌ `kubernetes.io/kubernetes` — Use client-go + apimachinery

---

## ⚠️ Critical Decisions Summary

### Decision 1: Builder Implementation Scope
- **A**: Docker only (fast MVP)
- **B**: Docker + Buildpacks (more features)
- **🎯 Recommendation**: A (Phase 1) — Document extension points for Phase 2+

### Decision 2: Template Format
- **A**: Embedded Go templates (simple, no user config)
- **B**: User-provided YAML files (flexible, more boilerplate)
- **🎯 Recommendation**: A initially, add B in Phase 3b

### Decision 3: Error Handling Richness
- **A**: Basic string errors (quick)
- **B**: Custom error types with context (better UX)
- **🎯 Recommendation**: B — Critical for user experience

### Decision 4: Testing Infrastructure
- **A**: Fake client only (fast, deterministic)
- **B**: Fake + real Kind cluster (comprehensive)
- **🎯 Recommendation**: Both — A for unit tests, B for integration tests

### Decision 5: Image Loading Strategy
- **A**: Always push to registry (slow for local dev)
- **B**: Use cluster-native loading (fast for local dev)
- **🎯 Recommendation**: B — Detect cluster type, use native loading

---

## ✅ Implementation Checklist

### Phase 1 ✓
- [ ] Config types defined and validated
- [ ] Config loader implemented (Viper + kubeconfig)
- [ ] Cobra CLI scaffolding (all commands defined)
- [ ] Context validator implemented (whitelist checking)
- [ ] Klog integration working (--debug flag)
- [ ] Unit tests written (>80% coverage)

### Phase 2 ✓
- [ ] Builder interface defined
- [ ] Docker builder implemented
- [ ] Hash calculation working (deterministic)
- [ ] Registry loader for Docker Desktop/Minikube/Kind
- [ ] Unit tests for builder + hash (>80% coverage)

### Phase 3 ✓
- [ ] YAML templates embedded
- [ ] Template rendering working
- [ ] Deployer interface implemented
- [ ] Upsert logic tested with fake client
- [ ] Delete with safety labels working
- [ ] Unit tests (>80% coverage)

### Phase 4 ✓
- [ ] Log tailing implemented (pod discovery)
- [ ] Port forwarding in background goroutine
- [ ] `kudev up` orchestration complete
- [ ] `kudev down` cleanup working
- [ ] `kudev status` showing accurate info
- [ ] Graceful Ctrl+C shutdown

### Phase 5 ✓
- [ ] File watcher implemented (fsnotify)
- [ ] Event debouncing working (500ms window)
- [ ] Rebuild trigger orchestration
- [ ] `kudev watch` command working
- [ ] Clear user feedback during watch mode

### Phase 6 ✓
- [ ] Error types defined (custom errors)
- [ ] Root command error handling uniform
- [ ] Comprehensive unit tests (>80% coverage)
- [ ] Integration tests with Kind
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Multi-platform releases (Linux/macOS/Windows)

---

## 🎯 Success Metrics

After completing all 6 phases:

### ✅ Functionality
- `kudev up` builds, deploys, forwards port, streams logs (one command)
- `kudev down` cleanly deletes all created resources
- `kudev watch` auto-rebuilds on file changes
- Works with Docker Desktop, Minikube, Kind
- `kudev status` shows accurate deployment info

### ✅ Code Quality
- >80% test coverage for critical paths
- All major components use interfaces (testable)
- Consistent error handling (custom error types)
- Follows Kubernetes community standards
- Clean separation: `cmd/` (minimal) → `pkg/` (business logic)

### ✅ Developer Experience
- Installation via `go install` or binary download
- Single `.kudev.yaml` configuration
- Clear help messages (`kudev --help`)
- Helpful error messages with suggested actions
- Logging with `--debug` flag for troubleshooting

### ✅ Production Ready
- CI/CD pipeline validates every commit
- Releases automated for all platforms
- Contributing guidelines documented
- Code review process established
- Extension points documented

---

## 📚 Additional Resources

- [Kubernetes Client-Go Documentation](https://github.com/kubernetes/client-go)
- [Cobra Command Framework](https://cobra.dev/)
- [Viper Configuration Library](https://github.com/spf13/viper)
- [Klog Logging Library](https://github.com/kubernetes/klog)
- [Kind - Kubernetes in Docker](https://kind.sigs.k8s.io/)

---

## 📝 Notes for Future Maintainers

### Extensibility Points

**New Builder Implementations** (Phase 2+):
- Implement `Builder` interface in `pkg/builder/{buildername}/`
- Register in `pkg/builder/factory.go`
- Document in `docs/builders.md`

**New Registry Handlers** (Phase 2+):
- Add case to `pkg/registry/loader.go`
- Implement cluster-specific loading
- Test with actual cluster type

**Custom Manifest Templates** (Phase 3+):
- Future: Allow `--template-path` flag
- Maintain backward compatibility with embedded defaults

### Common Pitfalls to Avoid

1. ❌ Don't cache K8s clients — Create fresh for each command
2. ❌ Don't ignore context cancellation — Respect `<-ctx.Done()`
3. ❌ Don't hardcode namespaces — Always use config
4. ❌ Don't trust image tags — Use hash-based validation
5. ❌ Don't print K8s API errors directly — Wrap with context

---

**Ready to implement? Start with [PHASE_1_CORE_FOUNDATION.md](./docs/phases/PHASE_1_CORE_FOUNDATION.md)** 🚀

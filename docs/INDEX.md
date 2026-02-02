# Kudev Documentation Index

Complete reference for the Kudev CLI project roadmap and implementation guides.

---

## 📖 Main Documents

### [RoadMap.md](../RoadMap.md) — **START HERE**
High-level overview of the entire project:
- ✅ Design philosophy and architecture
- ✅ K8s standards used
- ✅ Project structure diagram
- ✅ Dependency map
- ✅ Critical decisions summary
- ✅ Implementation checklist
- ✅ Success metrics

**Read this first** to understand the big picture.

---

## 📚 Phase Documentation

Each phase has a detailed guide with architecture, core decisions, and specific implementation tasks.

### Phase 1: Core Foundation (CLI & Config)
**[PHASE_1_CORE_FOUNDATION.md](./phases/PHASE_1_CORE_FOUNDATION.md)**

**Duration**: 1-2 weeks | **Difficulty**: 🟢 Beginner

**What you'll build**:
- Configuration system (.kudev.yaml)
- Cobra CLI with all command scaffolding
- Kubeconfig reader and context validation
- Klog setup for structured logging

**Key files**:
- `cmd/main.go`, `cmd/root.go`, `cmd/version.go`, `cmd/init.go`
- `pkg/config/types.go`, `pkg/config/loader.go`, `pkg/config/validation.go`
- `pkg/kubeconfig/context.go`, `pkg/kubeconfig/validator.go`
- `pkg/logging/logger.go`

**Skills needed**: Go basics, YAML parsing, Cobra framework

---

### Phase 2: Image Pipeline (Build System)
**[PHASE_2_IMAGE_PIPELINE.md](./phases/PHASE_2_IMAGE_PIPELINE.md)**

**Duration**: 1-2 weeks | **Difficulty**: 🟡 Intermediate

**What you'll build**:
- Builder interface for extensibility
- Docker builder using subprocess calls
- Source code hashing for deterministic builds
- Registry loading for Docker Desktop, Minikube, Kind

**Key files**:
- `pkg/builder/types.go`, `pkg/builder/docker/builder.go`
- `pkg/hash/calculator.go`, `pkg/builder/tagger.go`
- `pkg/registry/loader.go`, `pkg/registry/{docker,minikube,kind}.go`

**Skills needed**: subprocess handling, file hashing, pattern matching

---

### Phase 3: Manifest Orchestration (Deployment)
**[PHASE_3_MANIFEST_ORCHESTRATION.md](./phases/PHASE_3_MANIFEST_ORCHESTRATION.md)**

**Duration**: 1-2 weeks | **Difficulty**: 🟡 Intermediate

**What you'll build**:
- Embedded YAML templates for Deployment and Service
- Template rendering with Go's text/template
- Deployer interface for K8s operations
- Upsert logic (create if not exists, update if exists)
- Safe deletion with label-based filtering

**Key files**:
- `templates/deployment.yaml`, `templates/service.yaml`
- `pkg/deployer/types.go`, `pkg/deployer/deployer.go`
- `pkg/deployer/renderer.go`, `pkg/deployer/status.go`

**Skills needed**: client-go, K8s API patterns, template rendering

---

### Phase 4: Developer Experience (Feedback & UX)
**[PHASE_4_DEVELOPER_EXPERIENCE.md](./phases/PHASE_4_DEVELOPER_EXPERIENCE.md)**

**Duration**: 1 week | **Difficulty**: 🟡 Intermediate

**What you'll build**:
- Pod log tailing with automatic streaming
- Port forwarding in background goroutines
- Complete orchestration in `kudev up` command
- Status command and monitoring
- Graceful shutdown handling

**Key files**:
- `pkg/logs/tailer.go`, `pkg/logs/discovery.go`
- `pkg/portfwd/forwarder.go`
- `cmd/up.go`, `cmd/down.go`, `cmd/status.go`

**Skills needed**: goroutines, streaming I/O, signal handling

---

### Phase 5: Live Watcher (Hot Reload)
**[PHASE_5_LIVE_WATCHER.md](./phases/PHASE_5_LIVE_WATCHER.md)**

**Duration**: 1 week | **Difficulty**: 🟡 Intermediate

**What you'll build**:
- File system watcher using fsnotify
- Event debouncing (batch events within 500ms)
- Rebuild orchestration on file changes
- Watch command for hot-reload mode
- User feedback and status messages

**Key files**:
- `pkg/watch/watcher.go`, `pkg/watch/debounce.go`
- `pkg/watch/orchestrator.go`
- `cmd/watch.go`

**Skills needed**: fsnotify library, event handling, debouncing logic

---

### Phase 6: Testing & Reliability
**[PHASE_6_TESTING_RELIABILITY.md](./phases/PHASE_6_TESTING_RELIABILITY.md)**

**Duration**: 1-2 weeks | **Difficulty**: 🟢 Beginner

**What you'll build**:
- Custom error types for different failure modes
- Comprehensive unit tests with fake clients
- Integration tests with Kind cluster
- CI/CD pipeline using GitHub Actions
- Debug command for diagnostics
- Release automation for multiple platforms

**Key files**:
- `pkg/errors/errors.go`, `pkg/errors/messages.go`
- `pkg/*_test.go` across all packages
- `test/integration/*.go`
- `.github/workflows/{test,release}.yml`
- `cmd/debug.go`

**Skills needed**: testing patterns, fake clients, CI/CD concepts

---

## 🎯 Implementation Guide

**[IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)**

Practical implementation guide with:
- Common implementation patterns
- Testing strategies and examples
- Code organization best practices
- Troubleshooting and pitfalls
- Phase-by-phase checklists
- Example code snippets
- Learning resources

---

## 🚀 Quick Start

### 1. Read RoadMap.md (20 min)
Get familiar with the overall architecture and design philosophy.

### 2. Follow Phase 1 (1-2 weeks)
- Read `PHASE_1_CORE_FOUNDATION.md`
- Follow detailed tasks in order
- Write unit tests as you go
- Check success criteria

### 3. Continue Phases 2-6
- Each phase builds on previous phases
- Read phase documentation before starting
- Implement tasks in order
- Run tests to validate

### 4. Reference IMPLEMENTATION_GUIDE.md
For common patterns, testing strategies, and troubleshooting.

---

## 📊 Project Statistics

| Metric | Value |
|--------|-------|
| **Total Phases** | 6 |
| **Estimated Duration** | 6-8 weeks |
| **Core Dependencies** | 6 libraries |
| **Package Directories** | 13 |
| **Test Coverage Target** | 75%+ |
| **Documentation Pages** | 8 |

---

## 🏗️ Project Structure Overview

```
kudev/
├── RoadMap.md                          # Main roadmap (start here)
├── README.md                           # User documentation
├── CONTRIBUTING.md                     # Contributing guidelines
│
├── cmd/                                # CLI Commands (Cobra)
│   ├── main.go, root.go, version.go
│   ├── init.go, validate.go
│   ├── up.go, down.go, status.go
│   ├── logs.go, portfwd.go, watch.go
│   └── debug.go
│
├── pkg/                                # Reusable packages
│   ├── config/                         # Configuration loading
│   ├── kubeconfig/                     # K8s client init
│   ├── builder/                        # Build abstraction
│   ├── hash/                           # Source hashing
│   ├── registry/                       # Image loading
│   ├── deployer/                       # K8s operations
│   ├── logs/                           # Pod log tailing
│   ├── portfwd/                        # Port forwarding
│   ├── watch/                          # File watching
│   ├── errors/                         # Error types
│   ├── logging/                        # Klog wrapper
│   └── debug/                          # Debug utilities
│
├── templates/                          # Embedded YAML
│   ├── deployment.yaml
│   └── service.yaml
│
├── docs/                               # Documentation
│   ├── IMPLEMENTATION_GUIDE.md         # Practical guide
│   └── phases/
│       ├── PHASE_1_CORE_FOUNDATION.md
│       ├── PHASE_2_IMAGE_PIPELINE.md
│       ├── PHASE_3_MANIFEST_ORCHESTRATION.md
│       ├── PHASE_4_DEVELOPER_EXPERIENCE.md
│       ├── PHASE_5_LIVE_WATCHER.md
│       └── PHASE_6_TESTING_RELIABILITY.md
│
├── test/                               # Tests
│   ├── integration/                    # Integration tests
│   ├── fixtures/                       # Test data
│   └── testutil/                       # Test helpers
│
├── .github/workflows/                  # CI/CD
│   ├── test.yml
│   └── release.yml
│
├── Makefile                            # Build commands
├── go.mod, go.sum                      # Dependencies
└── .gitignore
```

---

## 🎯 Key Decisions Made for You

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI Framework | Cobra | K8s standard, plugin-compatible |
| Config Format | YAML | Industry standard, human-friendly |
| Builder Approach | Docker CLI subprocess | Lightweight, users have Docker CLI |
| Template Engine | Go text/template | Built-in, simple, sufficient |
| K8s Client | client-go | Official Kubernetes client |
| Logging | klog/v2 | K8s ecosystem standard |
| Testing | Fake clientset | Fast, deterministic, no cluster needed |
| Context Safety | Whitelist | Fail safely by default |
| Image Tagging | Hash-based | Deterministic, efficient caching |
| Registry Loading | Cluster-native | Fast for local dev (no push needed) |

**All of these are documented with rationale. You can override any decision in your implementation.**

---

## ✅ Validation Checklist

Before releasing:

- [ ] All 6 phases implemented
- [ ] Unit tests >80% coverage
- [ ] Integration tests passing
- [ ] All commands have help text
- [ ] Error messages are helpful with suggestions
- [ ] README.md complete with examples
- [ ] CONTRIBUTING.md written
- [ ] CI/CD pipeline working
- [ ] Releases built for Linux/macOS/Windows
- [ ] Semantic versioning (v1.0.0) tagged

---

## 🤝 Contributing Structure

The codebase is designed to be extensible:

### Adding a New Builder
1. Create `pkg/builder/{buildername}/builder.go`
2. Implement `Builder` interface
3. Register in `pkg/builder/factory.go`
4. Document in `docs/builders.md`

### Adding a New Registry Handler
1. Create `pkg/registry/{registryname}.go`
2. Implement `Loader` interface
3. Add case to `pkg/registry/loader.go`
4. Test with actual cluster type

### Custom Manifest Templates
1. Future: Support `--template-path` flag
2. Fallback to embedded defaults
3. Maintain backward compatibility

---

## 📞 Support & Resources

### When You Get Stuck

1. **Check the relevant phase documentation** — Most questions answered there
2. **Read IMPLEMENTATION_GUIDE.md** — Common patterns and pitfalls
3. **Review example code snippets** — Provided in each task
4. **Check K8s documentation** — client-go, kubectl, K8s API reference

### Key Resources

- [Kubernetes Client-Go](https://github.com/kubernetes/client-go) — Official docs
- [Cobra Framework](https://cobra.dev/) — CLI building
- [Viper Configuration](https://github.com/spf13/viper) — Config management
- [Kind Documentation](https://kind.sigs.k8s.io/) — Local K8s clusters
- [Klog Logging](https://github.com/kubernetes/klog) — K8s logging

---

## 🎓 Learning Path

### Prerequisites
- Go basics (packages, interfaces, error handling)
- Docker fundamentals
- Kubernetes concepts (Deployment, Service, Pod, Namespace)
- Git basics

### Learning During Implementation
- Cobra command framework (Phase 1)
- Viper configuration (Phase 1)
- client-go API patterns (Phase 3)
- Testing with fakes (Phase 6)
- CI/CD pipelines (Phase 6)

### Recommended Order
1. **Phase 1** → Master Cobra and CLI patterns
2. **Phase 2** → Learn subprocess handling and hashing
3. **Phase 3** → Deep dive into client-go and K8s APIs
4. **Phases 4-5** → Advanced patterns (goroutines, streaming)
5. **Phase 6** → Testing strategies and CI/CD

---

## 🎉 Next Steps

1. **Read [RoadMap.md](../RoadMap.md)** — Understand the full picture
2. **Review [PHASE_1_CORE_FOUNDATION.md](./phases/PHASE_1_CORE_FOUNDATION.md)** — Start implementing
3. **Check [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)** — Practical patterns and examples
4. **Run tests frequently** — Validate your work as you go

---

**You're ready to build Kudev! Let's go! 🚀**

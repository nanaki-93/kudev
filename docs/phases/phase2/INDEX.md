# Phase 2: Image Pipeline - Complete Implementation Guide

## Welcome to Phase 2! 🚀

This folder contains **detailed implementation guides** for each task in Phase 2. Each file is a complete deep-dive with:
- Problem overview
- Architecture decisions
- Complete code implementations
- Testing strategies
- Critical points and common mistakes
- Checklist for completion

---

## Quick Navigation

### 📋 Tasks (in order)

1. **[TASK_2_1_BUILDER_TYPES.md](./TASK_2_1_BUILDER_TYPES.md)** — Define Builder Interface & Types
   - Builder abstraction interface
   - BuildOptions and ImageRef types
   - Factory pattern for extensibility
   - ~2-3 hours effort

2. **[TASK_2_2_DOCKER_BUILDER.md](./TASK_2_2_DOCKER_BUILDER.md)** — Implement Docker Builder
   - Docker CLI subprocess execution
   - Output streaming to terminal
   - Daemon availability checks
   - ~3-4 hours effort

3. **[TASK_2_3_SOURCE_HASHING.md](./TASK_2_3_SOURCE_HASHING.md)** — Implement Source Code Hashing
   - Deterministic hash calculation
   - File exclusion patterns
   - .dockerignore integration
   - ~2-3 hours effort

4. **[TASK_2_4_IMAGE_TAGGING.md](./TASK_2_4_IMAGE_TAGGING.md)** — Implement Image Tagging
   - Hash-based tag generation
   - Timestamp suffix for forced rebuilds
   - Cache invalidation strategy
   - ~1-2 hours effort

5. **[TASK_2_5_REGISTRY_LOADING.md](./TASK_2_5_REGISTRY_LOADING.md)** — Implement Registry-Aware Image Loading
   - Cluster type detection
   - Docker Desktop, Minikube, Kind support
   - Native loading mechanisms
   - ~3-4 hours effort

**Total Effort**: ~12-16 hours  
**Total Complexity**: 🟡 Intermediate (subprocess calls, file hashing, Docker interaction)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│              User runs: kudev up                         │
└────────────────────────┬────────────────────────────────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
          ▼              ▼              ▼
    ┌──────────┐  ┌────────────┐  ┌──────────┐
    │  Hash    │  │  Builder   │  │ Registry │
    │ Calculator│  │  (Docker)  │  │  Loader  │
    │          │  │            │  │          │
    │Task 2.3  │  │Task 2.1    │  │Task 2.5  │
    │          │  │ 2.2        │  │          │
    └────┬─────┘  └─────┬──────┘  └────┬─────┘
         │              │              │
         │              ▼              │
         │        ┌──────────┐         │
         └───────►│  Tagger  │◄────────┘
                  │          │
                  │Task 2.4  │
                  └──────────┘
```

### Component Interactions

```
User runs: kudev up

1. Hash Calculation (Task 2.3)
   hash.NewCalculator().Calculate() → "a1b2c3d4"
   
2. Tag Generation (Task 2.4)
   tagger.GenerateTag() → "kudev-a1b2c3d4"
   
3. Docker Build (Task 2.1, 2.2)
   builder.Build(opts) → ImageRef{FullRef: "myapp:kudev-a1b2c3d4"}
   
4. Image Loading (Task 2.5)
   registry.Load(imageRef) → loads to Docker Desktop/Minikube/Kind
   
5. Return ImageRef for Phase 3 (Manifest Orchestration)
```

---

## Dependency Flow

```
Phase 1 (Config, Logger, CLI)
    ↓
Task 2.1 (Builder Types)
    ↓
Task 2.3 (Hash Calculator) ──┬──► Task 2.4 (Tagger)
    ↓                        │
Task 2.2 (Docker Builder) ───┘
    ↓
Task 2.5 (Registry Loader)
    ↓
Phase 3 (Manifest Orchestration)
```

---

## Key Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Build tool | Docker CLI subprocess | Lightweight, no SDK bloat, users have Docker |
| Image tagging | Hash-based + optional timestamp | Deterministic, cache-friendly, forces K8s pull |
| Hash algorithm | SHA256 truncated to 8 chars | Standard, readable, unique enough |
| Registry handling | Auto-detect cluster type | Works with Docker Desktop, Minikube, Kind |
| Output streaming | Real-time to terminal | User sees build progress immediately |

---

## File Map

| File | Purpose | Key Types/Functions |
|------|---------|---------------------|
| `pkg/builder/types.go` | Interface & types | `Builder`, `BuildOptions`, `ImageRef` |
| `pkg/builder/docker/builder.go` | Docker impl | `DockerBuilder`, `Build()`, `checkDockerDaemon()` |
| `pkg/builder/tagger.go` | Tag generation | `Tagger`, `GenerateTag()` |
| `pkg/hash/calculator.go` | Hash calculation | `Calculator`, `Calculate()` |
| `pkg/hash/exclusions.go` | Exclusion patterns | `shouldExclude()`, pattern matching |
| `pkg/registry/loader.go` | Orchestration | `Registry`, `Load()` |
| `pkg/registry/docker.go` | Docker Desktop | `dockerDesktopLoader` |
| `pkg/registry/minikube.go` | Minikube | `minikubeLoader` |
| `pkg/registry/kind.go` | Kind | `kindLoader` |

---

## Testing Strategy

### Unit Tests

| File | Coverage Target | Focus |
|------|-----------------|-------|
| `pkg/hash/calculator_test.go` | 85%+ | Determinism, exclusions |
| `pkg/builder/docker/builder_test.go` | 75%+ | Mock subprocess, error handling |
| `pkg/builder/tagger_test.go` | 90%+ | Tag format, timestamp option |
| `pkg/registry/loader_test.go` | 80%+ | Cluster detection, error paths |

### Integration Tests

```go
// +build docker_required

// Only run when Docker is available
func TestDockerBuildIntegration(t *testing.T) {
    // Actual Docker build test
}
```

---

## Quick Start Checklist

Before starting Phase 2, ensure Phase 1 is complete:
- [ ] `pkg/config/` — Types, validation, loader working
- [ ] `pkg/kubeconfig/` — Context validation working
- [ ] `pkg/logging/` — Logger initialized
- [ ] `cmd/commands/` — CLI scaffolding working

---

## Common Mistakes to Avoid

1. **Not checking Docker daemon first** — Always verify `docker version` before building
2. **Blocking on subprocess output** — Stream output in goroutines, don't buffer
3. **Hardcoding exclusion patterns** — Load from .dockerignore when present
4. **Ignoring context cancellation** — Use `exec.CommandContext()` everywhere
5. **Assuming cluster type** — Auto-detect, provide manual override

---

## References

- [Docker CLI Build Reference](https://docs.docker.com/engine/reference/commandline/build/)
- [Minikube Image Load](https://minikube.sigs.k8s.io/docs/commands/image/)
- [Kind Load Docker Image](https://kind.sigs.k8s.io/docs/user/quick-start/#loading-an-image-into-your-cluster)
- [Go exec Package](https://pkg.go.dev/os/exec)
- [SHA256 in Go](https://pkg.go.dev/crypto/sha256)

---

**Next**: Start with [TASK_2_1_BUILDER_TYPES.md](./TASK_2_1_BUILDER_TYPES.md) 🚀


# Phase 3: Manifest Orchestration - Complete Implementation Guide

## Welcome to Phase 3! 🚀

This folder contains **detailed implementation guides** for each task in Phase 3. Each file is a complete deep-dive with:
- Problem overview
- Architecture decisions
- Complete code implementations
- Testing strategies
- Critical points and common mistakes
- Checklist for completion

---

## Quick Navigation

### 📋 Tasks (in order)

1. **[TASK_3_1_EMBEDDED_TEMPLATES.md](./TASK_3_1_EMBEDDED_TEMPLATES.md)** — Create Embedded YAML Templates
   - Deployment template with labels
   - Service template with selectors
   - go:embed for binary inclusion
   - ~2-3 hours effort

2. **[TASK_3_2_TEMPLATE_TYPES.md](./TASK_3_2_TEMPLATE_TYPES.md)** — Define Template Data Structures
   - TemplateData for rendering
   - DeploymentStatus for status queries
   - Deployer interface definition
   - ~1-2 hours effort

3. **[TASK_3_3_TEMPLATE_RENDERING.md](./TASK_3_3_TEMPLATE_RENDERING.md)** — Implement Template Rendering
   - Go text/template parsing
   - YAML to K8s objects conversion
   - Template function helpers
   - ~2-3 hours effort

4. **[TASK_3_4_DEPLOYER_UPSERT.md](./TASK_3_4_DEPLOYER_UPSERT.md)** — Implement Deployer with Upsert Logic
   - Create/update pattern
   - client-go Kubernetes API
   - Namespace management
   - ~3-4 hours effort

5. **[TASK_3_5_STATUS_RETRIEVAL.md](./TASK_3_5_STATUS_RETRIEVAL.md)** — Implement Status Retrieval
   - Deployment status queries
   - Pod status aggregation
   - Health indicators
   - ~2 hours effort

6. **[TASK_3_6_SAFE_DELETE.md](./TASK_3_6_SAFE_DELETE.md)** — Implement Safe Delete with Labels
   - Label-based resource targeting
   - Cascading delete
   - Idempotent operations
   - ~1-2 hours effort

**Total Effort**: ~12-16 hours  
**Total Complexity**: 🟡 Intermediate (client-go, K8s API, templates)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│              Phase 2: Built ImageRef                 │
└────────────────────────┬────────────────────────────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
          ▼              ▼              ▼
    ┌──────────┐  ┌────────────┐  ┌──────────┐
    │ Template │  │  Renderer  │  │ Deployer │
    │  Files   │  │            │  │          │
    │          │  │            │  │          │
    │Task 3.1  │  │Task 3.3    │  │Task 3.4  │
    │          │  │            │  │ 3.5, 3.6 │
    └────┬─────┘  └─────┬──────┘  └────┬─────┘
         │              │              │
         │              ▼              │
         │        ┌──────────┐         │
         └───────►│ Template │◄────────┘
                  │  Data    │
                  │Task 3.2  │
                  └────┬─────┘
                       │
                       ▼
              ┌────────────────┐
              │   Kubernetes   │
              │     Cluster    │
              └────────────────┘
```

### Component Interactions

```
User runs: kudev up

1. Config & Image (Phase 1 + 2)
   LoadConfig() → config
   Build() → ImageRef{FullRef: "myapp:kudev-a1b2c3d4"}
   
2. Template Rendering (Task 3.3)
   renderer.RenderDeployment(data) → *appsv1.Deployment
   renderer.RenderService(data) → *corev1.Service
   
3. Deploy to K8s (Task 3.4)
   deployer.Upsert(ctx, opts) → creates/updates resources
   
4. Check Status (Task 3.5)
   deployer.Status(ctx, name, ns) → DeploymentStatus
```

---

## Dependency Flow

```
Phase 1 (Config)
Phase 2 (ImageRef)
    ↓
Task 3.1 (Templates) + Task 3.2 (Types)
    ↓
Task 3.3 (Renderer)
    ↓
Task 3.4 (Deployer - Upsert)
    ↓
Task 3.5 (Status) + Task 3.6 (Delete)
    ↓
Phase 4 (Developer Experience)
```

---

## Key Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Templates | Embedded (go:embed) | No external files, works out-of-box |
| Template engine | Go text/template | Built-in, simple, minimal dependencies |
| Upsert strategy | Get→Modify→Update | Safe, preserves existing state |
| Resource labels | `managed-by: kudev` | Identifies kudev resources |
| Hash tracking | `kudev-hash` label | Tracks deployed source version |

---

## File Map

| File | Purpose | Key Types/Functions |
|------|---------|---------------------|
| `templates/deployment.yaml` | Deployment template | YAML with placeholders |
| `templates/service.yaml` | Service template | YAML with placeholders |
| `pkg/deployer/types.go` | Data structures | `TemplateData`, `DeploymentStatus`, `Deployer` |
| `pkg/deployer/renderer.go` | Template rendering | `Renderer`, `RenderDeployment()`, `RenderService()` |
| `pkg/deployer/deployer.go` | K8s operations | `KubernetesDeployer`, `Upsert()` |
| `pkg/deployer/status.go` | Status queries | `Status()`, `getDeploymentStatusString()` |
| `pkg/deployer/delete.go` | Safe deletion | `Delete()` |

---

## Testing Strategy

### Unit Tests

| File | Coverage Target | Focus |
|------|-----------------|-------|
| `pkg/deployer/renderer_test.go` | 90%+ | Template validity, edge cases |
| `pkg/deployer/deployer_test.go` | 85%+ | Upsert logic with fake client |
| `pkg/deployer/status_test.go` | 80%+ | Status aggregation |
| `pkg/deployer/delete_test.go` | 85%+ | Idempotent deletion |

### Integration Tests

```go
// +build integration

func TestDeployToKind(t *testing.T) {
    // Real deployment to Kind cluster
}
```

---

## Quick Start Checklist

Before starting Phase 3, ensure Phase 1-2 are complete:
- [ ] `pkg/config/` — Types, validation, loader working
- [ ] `pkg/kubeconfig/` — Context validation working
- [ ] `pkg/builder/` — Docker builder working
- [ ] `pkg/hash/` — Hash calculator working
- [ ] `pkg/registry/` — Image loading working

---

## Common Mistakes to Avoid

1. **Not using labels** — Always label resources with `managed-by: kudev`
2. **Full replacement on update** — Preserve existing fields, only update image/env
3. **Ignoring clusterIP** — Never change clusterIP on service update
4. **Blocking deletes** — Use background propagation for faster deletes
5. **Missing namespace** — Always create namespace if not exists

---

## References

- [client-go Documentation](https://pkg.go.dev/k8s.io/client-go)
- [Kubernetes API Reference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/)
- [Go Embed](https://pkg.go.dev/embed)
- [Go Templates](https://pkg.go.dev/text/template)

---

**Next**: Start with [TASK_3_1_EMBEDDED_TEMPLATES.md](./TASK_3_1_EMBEDDED_TEMPLATES.md) 🚀


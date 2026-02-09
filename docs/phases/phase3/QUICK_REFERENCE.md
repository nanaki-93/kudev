# Phase 3 Quick Reference Guide

## For Busy Developers

This is a **TL;DR** version of Phase 3. For full details, see individual task files.

---

## Task Sequence & Time Estimates

```
Task 3.1 (3h)  → Embedded YAML templates
Task 3.2 (2h)  → Template data structures
Task 3.3 (3h)  → Template rendering
Task 3.4 (4h)  → Deployer with upsert logic
Task 3.5 (2h)  → Status retrieval
Task 3.6 (2h)  → Safe delete with labels
         ────────
Total: ~12-16 hours
```

---

## Core Concepts

### 1. Manifest Pipeline
- **Input**: Config + ImageRef from Phase 2
- **Processing**: Render templates → Upsert to K8s
- **Output**: Running deployment in cluster

### 2. Upsert Pattern
- **GET** existing resource
- **If not found**: CREATE new
- **If found**: MERGE changes → UPDATE

### 3. Label-Based Management
- `managed-by: kudev` — Identifies our resources
- `kudev-hash: {hash}` — Tracks deployed version
- `app: {name}` — Standard pod selection

---

## File Map

| File | Purpose | Key Types/Functions |
|------|---------|---------------------|
| `templates/deployment.yaml` | Deployment template | YAML with placeholders |
| `templates/service.yaml` | Service template | YAML with placeholders |
| `templates/embed.go` | Go embed | `DeploymentTemplate`, `ServiceTemplate` |
| `pkg/deployer/types.go` | Data structures | `TemplateData`, `Deployer`, `DeploymentStatus` |
| `pkg/deployer/renderer.go` | Rendering | `Renderer`, `RenderDeployment()` |
| `pkg/deployer/deployer.go` | K8s ops | `KubernetesDeployer`, `Upsert()` |
| `pkg/deployer/status.go` | Queries | `Status()`, `WaitForReady()` |
| `pkg/deployer/delete.go` | Deletion | `Delete()`, `DeleteByLabels()` |

---

## Key Patterns

### Template Rendering

```go
renderer, _ := NewRenderer(
    templates.DeploymentTemplate,
    templates.ServiceTemplate,
)

data := TemplateData{
    AppName:     "myapp",
    Namespace:   "default",
    ImageRef:    "myapp:kudev-abc12345",
    ImageHash:   "abc12345",
    ServicePort: 8080,
    Replicas:    2,
}

deployment, _ := renderer.RenderDeployment(data)
service, _ := renderer.RenderService(data)
```

### Upsert Pattern

```go
// Try GET
existing, err := deployments.Get(ctx, name, metav1.GetOptions{})
if errors.IsNotFound(err) {
    // CREATE
    deployments.Create(ctx, desired, metav1.CreateOptions{})
} else {
    // UPDATE (merge fields)
    existing.Spec.Replicas = desired.Spec.Replicas
    existing.Spec.Template.Spec.Containers[0].Image = newImage
    deployments.Update(ctx, existing, metav1.UpdateOptions{})
}
```

### Status Check

```go
status, _ := deployer.Status(ctx, "myapp", "default")
fmt.Printf("%s: %d/%d ready\n", 
    status.Status, 
    status.ReadyReplicas, 
    status.DesiredReplicas)
```

---

## Implementation Checklist

### Task 3.1: Embedded Templates
```
[ ] templates/deployment.yaml created
[ ] templates/service.yaml created  
[ ] templates/embed.go with //go:embed
[ ] Labels: managed-by, kudev-hash, app
[ ] imagePullPolicy: IfNotPresent
```

### Task 3.2: Types
```
[ ] TemplateData struct
[ ] DeploymentStatus struct
[ ] Deployer interface
[ ] NewTemplateData() helper
[ ] Validate() method
```

### Task 3.3: Renderer
```
[ ] NewRenderer() constructor
[ ] RenderDeployment() method
[ ] RenderService() method
[ ] RenderAll() for dry-run
[ ] Template functions (quote)
```

### Task 3.4: Deployer
```
[ ] KubernetesDeployer struct
[ ] Upsert() method
[ ] upsertDeployment() helper
[ ] upsertService() helper
[ ] ensureNamespace() helper
[ ] Preserve ClusterIP on update
```

### Task 3.5: Status
```
[ ] Status() method
[ ] buildPodStatuses() helper
[ ] computeStatusCode() helper
[ ] WaitForReady() method
```

### Task 3.6: Delete
```
[ ] Delete() method
[ ] deleteDeployment() helper
[ ] deleteService() helper
[ ] DeleteByLabels() method
[ ] Idempotent (NotFound = success)
```

---

## Common Commands

```bash
# Run all Phase 3 tests
go test ./pkg/deployer/... ./templates/... -v

# Check coverage
go test ./pkg/deployer/... -cover

# Build templates package
go build ./templates

# Format code
go fmt ./pkg/deployer/... ./templates/...
```

---

## Critical Points

### 1. Preserve ClusterIP

```go
// MUST preserve on update
desired.Spec.ClusterIP = existing.Spec.ClusterIP
desired.Spec.ClusterIPs = existing.Spec.ClusterIPs
```

### 2. Use sigs.k8s.io/yaml

```go
// K8s standard YAML library
import "sigs.k8s.io/yaml"
```

### 3. kubernetes.Interface

```go
// Enables fake client for testing
clientset kubernetes.Interface
```

### 4. Foreground Deletion

```go
// Wait for pods to terminate
propagation := metav1.DeletePropagationForeground
```

---

## Integration Example

```go
func Deploy(ctx context.Context, cfg *config.DeploymentConfig, imageRef string, hash string) error {
    // 1. Get K8s client
    clientset, _ := kubernetes.NewForConfig(kubeconfig)
    
    // 2. Create renderer
    renderer, _ := NewRenderer(
        templates.DeploymentTemplate,
        templates.ServiceTemplate,
    )
    
    // 3. Create deployer
    deployer := NewKubernetesDeployer(clientset, renderer, logger)
    
    // 4. Deploy
    opts := DeploymentOptions{
        Config:    cfg,
        ImageRef:  imageRef,
        ImageHash: hash,
    }
    
    status, err := deployer.Upsert(ctx, opts)
    if err != nil {
        return err
    }
    
    // 5. Wait for ready
    return deployer.WaitForReady(ctx, cfg.Metadata.Name, cfg.Spec.Namespace, 5*time.Minute)
}
```

---

## Testing with Fake Client

```go
import "k8s.io/client-go/kubernetes/fake"

func TestMyDeployer(t *testing.T) {
    // Empty cluster
    fakeClient := fake.NewSimpleClientset()
    
    // Or with existing resources
    fakeClient := fake.NewSimpleClientset(existingDeployment, existingService)
    
    deployer := NewKubernetesDeployer(fakeClient, renderer, logger)
    // Test...
}
```

---

## Next Phase

After completing Phase 3:
- ✅ Can render K8s manifests from templates
- ✅ Can create/update deployments
- ✅ Can query deployment status
- ✅ Can safely delete resources

**Phase 4 (Developer Experience)** will:
- Implement log tailing
- Implement port forwarding
- Orchestrate `kudev up` command
- Add status command


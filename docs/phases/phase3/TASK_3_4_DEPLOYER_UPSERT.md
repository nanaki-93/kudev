# Task 3.4: Implement Deployer with Upsert Logic

## Overview

This task implements the **Kubernetes Deployer** that creates or updates Deployments and Services using client-go. The upsert pattern ensures safe updates that preserve existing state.

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate (client-go, K8s API)  
**Dependencies**: Task 3.2 (Types), Task 3.3 (Renderer)  
**Files to Create**:
- `pkg/deployer/deployer.go` — Main deployer implementation
- `pkg/deployer/deployer_test.go` — Tests with fake client

---

## What You're Building

A deployer that:
1. **Renders** templates to K8s objects
2. **Creates** new resources if they don't exist
3. **Updates** existing resources safely (image, env only)
4. **Creates** namespaces if needed
5. **Returns** deployment status after operation

---

## The Upsert Pattern

```
┌─────────────────────────────────────────────────────┐
│                   Upsert Flow                        │
├─────────────────────────────────────────────────────┤
│  1. Try to GET existing resource                     │
│     │                                                │
│     ├── NotFound? → CREATE new resource              │
│     │                                                │
│     └── Found? → MERGE changes → UPDATE             │
│                                                      │
│  2. Preserve existing fields:                        │
│     - ClusterIP (Service)                            │
│     - ResourceVersion (all)                          │
│     - Annotations (user-added)                       │
│                                                      │
│  3. Update only kudev fields:                        │
│     - Container image                                │
│     - Environment variables                          │
│     - Replicas                                       │
│     - kudev-hash label                               │
└─────────────────────────────────────────────────────┘
```

---

## Complete Implementation

```go
// pkg/deployer/deployer.go

package deployer

import (
    "context"
    "fmt"
    
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    
    "github.com/your-org/kudev/pkg/logging"
)

// KubernetesDeployer implements Deployer using client-go.
type KubernetesDeployer struct {
    clientset kubernetes.Interface
    renderer  *Renderer
    logger    logging.Logger
}

// NewKubernetesDeployer creates a new deployer.
func NewKubernetesDeployer(
    clientset kubernetes.Interface,
    renderer *Renderer,
    logger logging.Logger,
) *KubernetesDeployer {
    return &KubernetesDeployer{
        clientset: clientset,
        renderer:  renderer,
        logger:    logger,
    }
}

// Upsert creates or updates deployment and service.
func (kd *KubernetesDeployer) Upsert(ctx context.Context, opts DeploymentOptions) (*DeploymentStatus, error) {
    // 1. Prepare template data
    data := NewTemplateData(opts)
    
    kd.logger.Info("starting deployment",
        "app", data.AppName,
        "namespace", data.Namespace,
        "image", data.ImageRef,
    )
    
    // 2. Render manifests
    deployment, err := kd.renderer.RenderDeployment(data)
    if err != nil {
        return nil, fmt.Errorf("failed to render deployment: %w", err)
    }
    
    service, err := kd.renderer.RenderService(data)
    if err != nil {
        return nil, fmt.Errorf("failed to render service: %w", err)
    }
    
    // 3. Ensure namespace exists
    if err := kd.ensureNamespace(ctx, data.Namespace); err != nil {
        return nil, fmt.Errorf("failed to ensure namespace: %w", err)
    }
    
    // 4. Upsert Deployment
    if err := kd.upsertDeployment(ctx, deployment); err != nil {
        return nil, fmt.Errorf("failed to upsert deployment: %w", err)
    }
    
    // 5. Upsert Service
    if err := kd.upsertService(ctx, service); err != nil {
        return nil, fmt.Errorf("failed to upsert service: %w", err)
    }
    
    kd.logger.Info("deployment completed successfully",
        "app", data.AppName,
        "namespace", data.Namespace,
    )
    
    // 6. Return current status
    return kd.Status(ctx, data.AppName, data.Namespace)
}

// upsertDeployment creates or updates a Deployment.
func (kd *KubernetesDeployer) upsertDeployment(ctx context.Context, desired *appsv1.Deployment) error {
    deployments := kd.clientset.AppsV1().Deployments(desired.Namespace)
    
    // Try to get existing
    existing, err := deployments.Get(ctx, desired.Name, metav1.GetOptions{})
    if err != nil {
        if errors.IsNotFound(err) {
            // Create new deployment
            _, err := deployments.Create(ctx, desired, metav1.CreateOptions{})
            if err != nil {
                return fmt.Errorf("failed to create deployment: %w", err)
            }
            kd.logger.Info("deployment created",
                "name", desired.Name,
                "namespace", desired.Namespace,
            )
            return nil
        }
        return fmt.Errorf("failed to get deployment: %w", err)
    }
    
    // Update existing deployment
    // Preserve fields that shouldn't change
    existing.Spec.Replicas = desired.Spec.Replicas
    
    // Update container image and env
    if len(existing.Spec.Template.Spec.Containers) > 0 && 
       len(desired.Spec.Template.Spec.Containers) > 0 {
        existing.Spec.Template.Spec.Containers[0].Image = 
            desired.Spec.Template.Spec.Containers[0].Image
        existing.Spec.Template.Spec.Containers[0].Env = 
            desired.Spec.Template.Spec.Containers[0].Env
    }
    
    // Update kudev labels
    if existing.Labels == nil {
        existing.Labels = make(map[string]string)
    }
    existing.Labels["kudev-hash"] = desired.Labels["kudev-hash"]
    
    // Update pod template labels
    if existing.Spec.Template.Labels == nil {
        existing.Spec.Template.Labels = make(map[string]string)
    }
    existing.Spec.Template.Labels["managed-by"] = "kudev"
    
    _, err = deployments.Update(ctx, existing, metav1.UpdateOptions{})
    if err != nil {
        return fmt.Errorf("failed to update deployment: %w", err)
    }
    
    kd.logger.Info("deployment updated",
        "name", desired.Name,
        "namespace", desired.Namespace,
    )
    
    return nil
}

// upsertService creates or updates a Service.
func (kd *KubernetesDeployer) upsertService(ctx context.Context, desired *corev1.Service) error {
    services := kd.clientset.CoreV1().Services(desired.Namespace)
    
    existing, err := services.Get(ctx, desired.Name, metav1.GetOptions{})
    if err != nil {
        if errors.IsNotFound(err) {
            // Create new service
            _, err := services.Create(ctx, desired, metav1.CreateOptions{})
            if err != nil {
                return fmt.Errorf("failed to create service: %w", err)
            }
            kd.logger.Info("service created",
                "name", desired.Name,
                "namespace", desired.Namespace,
            )
            return nil
        }
        return fmt.Errorf("failed to get service: %w", err)
    }
    
    // Update existing service
    // CRITICAL: Preserve ClusterIP (cannot be changed)
    desired.Spec.ClusterIP = existing.Spec.ClusterIP
    desired.Spec.ClusterIPs = existing.Spec.ClusterIPs
    
    // Copy resource version for update
    desired.ResourceVersion = existing.ResourceVersion
    
    _, err = services.Update(ctx, desired, metav1.UpdateOptions{})
    if err != nil {
        return fmt.Errorf("failed to update service: %w", err)
    }
    
    kd.logger.Info("service updated",
        "name", desired.Name,
        "namespace", desired.Namespace,
    )
    
    return nil
}

// ensureNamespace creates namespace if it doesn't exist.
func (kd *KubernetesDeployer) ensureNamespace(ctx context.Context, namespace string) error {
    // Skip for default namespace
    if namespace == "default" {
        return nil
    }
    
    namespaces := kd.clientset.CoreV1().Namespaces()
    
    _, err := namespaces.Get(ctx, namespace, metav1.GetOptions{})
    if err == nil {
        // Namespace exists
        return nil
    }
    
    if !errors.IsNotFound(err) {
        return fmt.Errorf("failed to check namespace: %w", err)
    }
    
    // Create namespace
    ns := &corev1.Namespace{
        ObjectMeta: metav1.ObjectMeta{
            Name: namespace,
            Labels: map[string]string{
                "managed-by": "kudev",
            },
        },
    }
    
    _, err = namespaces.Create(ctx, ns, metav1.CreateOptions{})
    if err != nil {
        // Ignore AlreadyExists (race condition)
        if errors.IsAlreadyExists(err) {
            return nil
        }
        return fmt.Errorf("failed to create namespace: %w", err)
    }
    
    kd.logger.Info("namespace created", "name", namespace)
    return nil
}

// Ensure KubernetesDeployer implements Deployer
var _ Deployer = (*KubernetesDeployer)(nil)
```

---

## Testing with Fake Client

```go
// pkg/deployer/deployer_test.go

package deployer

import (
    "context"
    "testing"
    
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes/fake"
    
    "github.com/your-org/kudev/pkg/config"
    "github.com/your-org/kudev/templates"
)

type mockLogger struct{}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {}

func TestUpsert_CreateNew(t *testing.T) {
    // Create fake clientset (empty cluster)
    fakeClient := fake.NewSimpleClientset()
    
    renderer, _ := NewRenderer(
        templates.DeploymentTemplate,
        templates.ServiceTemplate,
    )
    
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    opts := DeploymentOptions{
        Config: &config.DeploymentConfig{
            Metadata: config.ConfigMetadata{Name: "test-app"},
            Spec: config.DeploymentSpec{
                Namespace:   "default",
                Replicas:    2,
                ServicePort: 8080,
            },
        },
        ImageRef:  "test-app:kudev-12345678",
        ImageHash: "12345678",
    }
    
    status, err := deployer.Upsert(context.Background(), opts)
    if err != nil {
        t.Fatalf("Upsert failed: %v", err)
    }
    
    // Verify deployment was created
    deployment, err := fakeClient.AppsV1().Deployments("default").Get(
        context.Background(), "test-app", metav1.GetOptions{},
    )
    if err != nil {
        t.Fatalf("deployment not found: %v", err)
    }
    
    if deployment.Name != "test-app" {
        t.Errorf("name = %q, want %q", deployment.Name, "test-app")
    }
    
    // Verify service was created
    service, err := fakeClient.CoreV1().Services("default").Get(
        context.Background(), "test-app", metav1.GetOptions{},
    )
    if err != nil {
        t.Fatalf("service not found: %v", err)
    }
    
    if service.Name != "test-app" {
        t.Errorf("service name = %q, want %q", service.Name, "test-app")
    }
    
    if status == nil {
        t.Error("status is nil")
    }
}

func TestUpsert_UpdateExisting(t *testing.T) {
    // Create fake clientset with existing deployment
    existingDeployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-app",
            Namespace: "default",
            Labels: map[string]string{
                "app":        "test-app",
                "managed-by": "kudev",
                "kudev-hash": "old-hash",
            },
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: int32Ptr(1),
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": "test-app"},
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": "test-app"},
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "test-app",
                            Image: "test-app:old-image",
                        },
                    },
                },
            },
        },
    }
    
    existingService := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-app",
            Namespace: "default",
        },
        Spec: corev1.ServiceSpec{
            ClusterIP: "10.0.0.100", // Existing ClusterIP
            Ports: []corev1.ServicePort{
                {Port: 8080},
            },
            Selector: map[string]string{"app": "test-app"},
        },
    }
    
    fakeClient := fake.NewSimpleClientset(existingDeployment, existingService)
    
    renderer, _ := NewRenderer(
        templates.DeploymentTemplate,
        templates.ServiceTemplate,
    )
    
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    opts := DeploymentOptions{
        Config: &config.DeploymentConfig{
            Metadata: config.ConfigMetadata{Name: "test-app"},
            Spec: config.DeploymentSpec{
                Namespace:   "default",
                Replicas:    3, // Changed!
                ServicePort: 8080,
            },
        },
        ImageRef:  "test-app:kudev-new-hash", // Changed!
        ImageHash: "new-hash",
    }
    
    _, err := deployer.Upsert(context.Background(), opts)
    if err != nil {
        t.Fatalf("Upsert failed: %v", err)
    }
    
    // Verify deployment was updated
    deployment, _ := fakeClient.AppsV1().Deployments("default").Get(
        context.Background(), "test-app", metav1.GetOptions{},
    )
    
    if *deployment.Spec.Replicas != 3 {
        t.Errorf("replicas = %d, want 3", *deployment.Spec.Replicas)
    }
    
    if deployment.Spec.Template.Spec.Containers[0].Image != "test-app:kudev-new-hash" {
        t.Errorf("image not updated")
    }
    
    if deployment.Labels["kudev-hash"] != "new-hash" {
        t.Errorf("hash label not updated")
    }
    
    // Verify ClusterIP was preserved
    service, _ := fakeClient.CoreV1().Services("default").Get(
        context.Background(), "test-app", metav1.GetOptions{},
    )
    
    if service.Spec.ClusterIP != "10.0.0.100" {
        t.Errorf("ClusterIP changed! was 10.0.0.100, now %s", service.Spec.ClusterIP)
    }
}

func TestUpsert_CreatesNamespace(t *testing.T) {
    fakeClient := fake.NewSimpleClientset()
    
    renderer, _ := NewRenderer(
        templates.DeploymentTemplate,
        templates.ServiceTemplate,
    )
    
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    opts := DeploymentOptions{
        Config: &config.DeploymentConfig{
            Metadata: config.ConfigMetadata{Name: "test-app"},
            Spec: config.DeploymentSpec{
                Namespace:   "custom-ns", // Non-default namespace
                Replicas:    1,
                ServicePort: 8080,
            },
        },
        ImageRef:  "test-app:latest",
        ImageHash: "12345678",
    }
    
    _, err := deployer.Upsert(context.Background(), opts)
    if err != nil {
        t.Fatalf("Upsert failed: %v", err)
    }
    
    // Verify namespace was created
    ns, err := fakeClient.CoreV1().Namespaces().Get(
        context.Background(), "custom-ns", metav1.GetOptions{},
    )
    if err != nil {
        t.Fatalf("namespace not created: %v", err)
    }
    
    if ns.Labels["managed-by"] != "kudev" {
        t.Error("namespace missing managed-by label")
    }
}

func int32Ptr(i int32) *int32 {
    return &i
}
```

---

## Key Points

### 1. Preserve ClusterIP

Service ClusterIP cannot be changed after creation:
```go
// CRITICAL: Preserve ClusterIP
desired.Spec.ClusterIP = existing.Spec.ClusterIP
desired.Spec.ClusterIPs = existing.Spec.ClusterIPs
```

### 2. ResourceVersion for Updates

K8s requires ResourceVersion for optimistic concurrency:
```go
desired.ResourceVersion = existing.ResourceVersion
```

### 3. Handle Race Conditions

Namespace creation might race:
```go
if errors.IsAlreadyExists(err) {
    return nil  // Someone else created it, that's fine
}
```

### 4. Use kubernetes.Interface

Enables both real and fake clients:
```go
type KubernetesDeployer struct {
    clientset kubernetes.Interface  // Not *kubernetes.Clientset
}
```

---

## Checklist for Task 3.4

- [ ] Create `pkg/deployer/deployer.go`
- [ ] Implement `KubernetesDeployer` struct
- [ ] Implement `NewKubernetesDeployer()` constructor
- [ ] Implement `Upsert()` method
- [ ] Implement `upsertDeployment()` helper
- [ ] Implement `upsertService()` helper
- [ ] Implement `ensureNamespace()` helper
- [ ] Preserve ClusterIP on service update
- [ ] Use `kubernetes.Interface` for testability
- [ ] Add interface assertion
- [ ] Create `pkg/deployer/deployer_test.go`
- [ ] Test create new resources
- [ ] Test update existing resources
- [ ] Test namespace creation
- [ ] Test ClusterIP preservation
- [ ] Run `go test ./pkg/deployer -v`

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 3.5** → Implement Status Retrieval
3. Status will be called by Upsert to return current state

---

## References

- [client-go Documentation](https://pkg.go.dev/k8s.io/client-go)
- [Fake Client](https://pkg.go.dev/k8s.io/client-go/kubernetes/fake)
- [K8s API Errors](https://pkg.go.dev/k8s.io/apimachinery/pkg/api/errors)


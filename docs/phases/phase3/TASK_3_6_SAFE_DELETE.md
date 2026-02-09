# Task 3.6: Implement Safe Delete with Labels

## Overview

This task implements **safe deletion** of Kubernetes resources using labels to target only kudev-managed resources.

**Effort**: ~1-2 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: Task 3.4 (Deployer)  
**Files to Create**:
- `pkg/deployer/delete.go` — Delete implementation
- Add tests to `pkg/deployer/deployer_test.go`

---

## What You're Building

A delete operation that:
1. **Deletes** Deployment by name
2. **Deletes** associated Service
3. **Uses** foreground deletion for clean cascade
4. **Is idempotent** (safe to call multiple times)
5. **Logs** what was deleted

---

## Complete Implementation

```go
// pkg/deployer/delete.go

package deployer

import (
    "context"
    "fmt"
    
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Delete removes the deployment and associated service.
// Only deletes resources with matching name.
// Safe to call multiple times (idempotent).
func (kd *KubernetesDeployer) Delete(ctx context.Context, appName, namespace string) error {
    kd.logger.Info("deleting deployment",
        "app", appName,
        "namespace", namespace,
    )
    
    var deleteErrors []string
    
    // Delete Deployment
    if err := kd.deleteDeployment(ctx, appName, namespace); err != nil {
        deleteErrors = append(deleteErrors, fmt.Sprintf("deployment: %v", err))
    }
    
    // Delete Service
    if err := kd.deleteService(ctx, appName, namespace); err != nil {
        deleteErrors = append(deleteErrors, fmt.Sprintf("service: %v", err))
    }
    
    if len(deleteErrors) > 0 {
        return fmt.Errorf("deletion errors: %v", deleteErrors)
    }
    
    kd.logger.Info("deletion completed",
        "app", appName,
        "namespace", namespace,
    )
    
    return nil
}

// deleteDeployment removes a Deployment.
func (kd *KubernetesDeployer) deleteDeployment(ctx context.Context, name, namespace string) error {
    deployments := kd.clientset.AppsV1().Deployments(namespace)
    
    // Use Foreground deletion to wait for pods to terminate
    propagation := metav1.DeletePropagationForeground
    
    err := deployments.Delete(ctx, name, metav1.DeleteOptions{
        PropagationPolicy: &propagation,
    })
    
    if err != nil {
        if errors.IsNotFound(err) {
            kd.logger.Debug("deployment already deleted",
                "name", name,
                "namespace", namespace,
            )
            return nil // Idempotent
        }
        return fmt.Errorf("failed to delete deployment: %w", err)
    }
    
    kd.logger.Info("deployment deleted",
        "name", name,
        "namespace", namespace,
    )
    
    return nil
}

// deleteService removes a Service.
func (kd *KubernetesDeployer) deleteService(ctx context.Context, name, namespace string) error {
    services := kd.clientset.CoreV1().Services(namespace)
    
    err := services.Delete(ctx, name, metav1.DeleteOptions{})
    
    if err != nil {
        if errors.IsNotFound(err) {
            kd.logger.Debug("service already deleted",
                "name", name,
                "namespace", namespace,
            )
            return nil // Idempotent
        }
        return fmt.Errorf("failed to delete service: %w", err)
    }
    
    kd.logger.Info("service deleted",
        "name", name,
        "namespace", namespace,
    )
    
    return nil
}

// DeleteByLabels removes all resources with the kudev managed-by label.
// Useful for cleanup of orphaned resources.
func (kd *KubernetesDeployer) DeleteByLabels(ctx context.Context, namespace string) error {
    kd.logger.Info("deleting all kudev resources",
        "namespace", namespace,
    )
    
    labelSelector := "managed-by=kudev"
    
    // Delete Deployments
    deployments := kd.clientset.AppsV1().Deployments(namespace)
    if err := deployments.DeleteCollection(ctx,
        metav1.DeleteOptions{},
        metav1.ListOptions{LabelSelector: labelSelector},
    ); err != nil && !errors.IsNotFound(err) {
        return fmt.Errorf("failed to delete deployments: %w", err)
    }
    
    // Delete Services
    services := kd.clientset.CoreV1().Services(namespace)
    serviceList, err := services.List(ctx, metav1.ListOptions{
        LabelSelector: labelSelector,
    })
    if err != nil {
        return fmt.Errorf("failed to list services: %w", err)
    }
    
    for _, svc := range serviceList.Items {
        if err := services.Delete(ctx, svc.Name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
            return fmt.Errorf("failed to delete service %s: %w", svc.Name, err)
        }
        kd.logger.Info("service deleted", "name", svc.Name)
    }
    
    kd.logger.Info("all kudev resources deleted",
        "namespace", namespace,
    )
    
    return nil
}

// WaitForDeletion waits until deployment is fully deleted.
func (kd *KubernetesDeployer) WaitForDeletion(ctx context.Context, appName, namespace string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    
    for {
        if time.Now().After(deadline) {
            return fmt.Errorf("timeout waiting for deletion")
        }
        
        _, err := kd.clientset.AppsV1().Deployments(namespace).Get(
            ctx, appName, metav1.GetOptions{},
        )
        
        if errors.IsNotFound(err) {
            kd.logger.Info("deployment fully deleted",
                "app", appName,
                "namespace", namespace,
            )
            return nil
        }
        
        if err != nil {
            return fmt.Errorf("error checking deployment: %w", err)
        }
        
        kd.logger.Debug("waiting for deletion",
            "app", appName,
        )
        
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(2 * time.Second):
            // Continue polling
        }
    }
}
```

---

## Deletion Strategies

### Foreground Deletion

Pods are deleted first, then the Deployment:
```go
propagation := metav1.DeletePropagationForeground
```

**When to use**: Clean shutdown, ensure no orphaned pods

### Background Deletion

Deployment is deleted immediately, pods are garbage collected:
```go
propagation := metav1.DeletePropagationBackground
```

**When to use**: Fast deletion, don't need to wait

### Orphan Deletion

Pods are NOT deleted:
```go
propagation := metav1.DeletePropagationOrphan
```

**When to use**: Rarely, special cases only

---

## Testing Delete

```go
// Add to pkg/deployer/deployer_test.go

func TestDelete_ExistingResources(t *testing.T) {
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-app",
            Namespace: "default",
        },
    }
    
    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-app",
            Namespace: "default",
        },
    }
    
    fakeClient := fake.NewSimpleClientset(deployment, service)
    renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    err := deployer.Delete(context.Background(), "test-app", "default")
    if err != nil {
        t.Fatalf("Delete failed: %v", err)
    }
    
    // Verify deployment was deleted
    _, err = fakeClient.AppsV1().Deployments("default").Get(
        context.Background(), "test-app", metav1.GetOptions{},
    )
    if !errors.IsNotFound(err) {
        t.Error("deployment should be deleted")
    }
    
    // Verify service was deleted
    _, err = fakeClient.CoreV1().Services("default").Get(
        context.Background(), "test-app", metav1.GetOptions{},
    )
    if !errors.IsNotFound(err) {
        t.Error("service should be deleted")
    }
}

func TestDelete_Idempotent(t *testing.T) {
    // Empty cluster - nothing to delete
    fakeClient := fake.NewSimpleClientset()
    renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    // Should not error even if resources don't exist
    err := deployer.Delete(context.Background(), "nonexistent", "default")
    if err != nil {
        t.Errorf("Delete should be idempotent, got: %v", err)
    }
    
    // Call again - still no error
    err = deployer.Delete(context.Background(), "nonexistent", "default")
    if err != nil {
        t.Errorf("Delete should be idempotent on second call, got: %v", err)
    }
}

func TestDelete_PartialResources(t *testing.T) {
    // Only deployment exists, no service
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-app",
            Namespace: "default",
        },
    }
    
    fakeClient := fake.NewSimpleClientset(deployment)
    renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    err := deployer.Delete(context.Background(), "test-app", "default")
    if err != nil {
        t.Errorf("Should handle partial resources, got: %v", err)
    }
}

func TestDeleteByLabels(t *testing.T) {
    // Create multiple kudev-managed resources
    dep1 := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "app1",
            Namespace: "default",
            Labels:    map[string]string{"managed-by": "kudev"},
        },
    }
    dep2 := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "app2",
            Namespace: "default",
            Labels:    map[string]string{"managed-by": "kudev"},
        },
    }
    // This one is NOT managed by kudev
    dep3 := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "other-app",
            Namespace: "default",
        },
    }
    
    fakeClient := fake.NewSimpleClientset(dep1, dep2, dep3)
    renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    err := deployer.DeleteByLabels(context.Background(), "default")
    if err != nil {
        t.Fatalf("DeleteByLabels failed: %v", err)
    }
    
    // other-app should still exist (not managed by kudev)
    _, err = fakeClient.AppsV1().Deployments("default").Get(
        context.Background(), "other-app", metav1.GetOptions{},
    )
    if errors.IsNotFound(err) {
        t.Error("other-app should NOT be deleted")
    }
}
```

---

## Checklist for Task 3.6

- [ ] Create `pkg/deployer/delete.go`
- [ ] Implement `Delete()` method
- [ ] Implement `deleteDeployment()` helper
- [ ] Implement `deleteService()` helper
- [ ] Implement `DeleteByLabels()` method
- [ ] Implement `WaitForDeletion()` method
- [ ] Use foreground propagation policy
- [ ] Handle NotFound errors (idempotent)
- [ ] Log what was deleted
- [ ] Add tests for existing resources
- [ ] Add tests for idempotent behavior
- [ ] Add tests for partial resources
- [ ] Run `go test ./pkg/deployer -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Not handling NotFound
```go
// Wrong - errors on already deleted
err := deployments.Delete(ctx, name, opts)
return err  // Returns error if not found!

// Right - idempotent
if errors.IsNotFound(err) {
    return nil  // Already deleted, that's fine
}
```

❌ **Mistake 2**: Using background deletion without waiting
```go
// Wrong - returns immediately, pods still running
propagation := metav1.DeletePropagationBackground
deployments.Delete(...)
// User thinks it's done, but pods are still terminating

// Right for clean shutdown
propagation := metav1.DeletePropagationForeground
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Phase 3 is now complete! 🎉
3. Move to **Phase 4** → Developer Experience

---

## References

- [K8s Garbage Collection](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/)
- [Delete Propagation](https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/)


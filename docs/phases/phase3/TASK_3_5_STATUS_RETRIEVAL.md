# Task 3.5: Implement Status Retrieval

## Overview

This task implements **deployment status queries** that return the current state of deployments and their pods.

**Effort**: ~2 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: Task 3.4 (Deployer)  
**Files to Create**:
- `pkg/deployer/status.go` — Status query implementation
- Add tests to `pkg/deployer/deployer_test.go`

---

## What You're Building

A status retrieval system that:
1. **Queries** Deployment status from K8s API
2. **Lists** pods by label selector
3. **Aggregates** pod statuses
4. **Computes** overall health status
5. **Returns** user-friendly status message

---

## Complete Implementation

```go
// pkg/deployer/status.go

package deployer

import (
    "context"
    "fmt"
    "time"
    
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/labels"
)

// Status returns the current deployment status.
func (kd *KubernetesDeployer) Status(ctx context.Context, appName, namespace string) (*DeploymentStatus, error) {
    kd.logger.Debug("getting deployment status",
        "app", appName,
        "namespace", namespace,
    )
    
    // Get deployment
    deployment, err := kd.clientset.AppsV1().Deployments(namespace).Get(
        ctx, appName, metav1.GetOptions{},
    )
    if err != nil {
        if errors.IsNotFound(err) {
            return nil, fmt.Errorf("deployment not found: %s/%s", namespace, appName)
        }
        return nil, fmt.Errorf("failed to get deployment: %w", err)
    }
    
    // Get pods by label selector
    selector := labels.SelectorFromSet(labels.Set{"app": appName})
    pods, err := kd.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
        LabelSelector: selector.String(),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to list pods: %w", err)
    }
    
    // Determine desired replicas
    var desiredReplicas int32 = 1
    if deployment.Spec.Replicas != nil {
        desiredReplicas = *deployment.Spec.Replicas
    }
    
    // Build pod statuses
    podStatuses := buildPodStatuses(pods)
    
    // Determine overall status
    statusCode := computeStatusCode(deployment.Status.ReadyReplicas, desiredReplicas, podStatuses)
    
    // Get image hash from labels
    imageHash := ""
    if deployment.Labels != nil {
        imageHash = deployment.Labels["kudev-hash"]
    }
    
    status := &DeploymentStatus{
        DeploymentName:  deployment.Name,
        Namespace:       deployment.Namespace,
        ReadyReplicas:   deployment.Status.ReadyReplicas,
        DesiredReplicas: desiredReplicas,
        Status:          statusCode.String(),
        Pods:            podStatuses,
        Message:         buildStatusMessage(statusCode, deployment.Status.ReadyReplicas, desiredReplicas),
        ImageHash:       imageHash,
        LastUpdated:     time.Now(),
    }
    
    return status, nil
}

// buildPodStatuses converts K8s pod list to our PodStatus slice.
func buildPodStatuses(pods *corev1.PodList) []PodStatus {
    var statuses []PodStatus
    
    for _, pod := range pods.Items {
        status := PodStatus{
            Name:      pod.Name,
            Status:    string(pod.Status.Phase),
            Ready:     isPodReady(&pod),
            CreatedAt: pod.CreationTimestamp.Time,
        }
        
        // Count container restarts
        for _, cs := range pod.Status.ContainerStatuses {
            status.Restarts += cs.RestartCount
            
            // Get waiting/terminated message
            if cs.State.Waiting != nil && cs.State.Waiting.Message != "" {
                status.Message = cs.State.Waiting.Message
            }
            if cs.State.Terminated != nil && cs.State.Terminated.Message != "" {
                status.Message = cs.State.Terminated.Message
            }
        }
        
        statuses = append(statuses, status)
    }
    
    return statuses
}

// isPodReady checks if all containers in pod are ready.
func isPodReady(pod *corev1.Pod) bool {
    for _, condition := range pod.Status.Conditions {
        if condition.Type == corev1.PodReady {
            return condition.Status == corev1.ConditionTrue
        }
    }
    return false
}

// computeStatusCode determines overall deployment health.
func computeStatusCode(ready, desired int32, pods []PodStatus) StatusCode {
    if desired == 0 {
        return StatusUnknown
    }
    
    if ready >= desired {
        return StatusRunning
    }
    
    if ready == 0 {
        // Check for crash loops
        for _, pod := range pods {
            if pod.Restarts > 3 {
                return StatusFailed
            }
        }
        return StatusPending
    }
    
    return StatusDegraded
}

// buildStatusMessage creates a user-friendly status message.
func buildStatusMessage(status StatusCode, ready, desired int32) string {
    switch status {
    case StatusRunning:
        return fmt.Sprintf("All %d replicas are running", desired)
    case StatusPending:
        return fmt.Sprintf("Waiting for pods to start (0/%d ready)", desired)
    case StatusDegraded:
        return fmt.Sprintf("Partially running (%d/%d ready)", ready, desired)
    case StatusFailed:
        return "Pods are failing - check logs with 'kudev logs'"
    default:
        return "Unable to determine status"
    }
}

// WaitForReady waits until deployment is ready or timeout.
func (kd *KubernetesDeployer) WaitForReady(ctx context.Context, appName, namespace string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    
    for {
        if time.Now().After(deadline) {
            return fmt.Errorf("timeout waiting for deployment to be ready")
        }
        
        status, err := kd.Status(ctx, appName, namespace)
        if err != nil {
            // Deployment might not exist yet
            kd.logger.Debug("waiting for deployment", "error", err)
        } else if status.IsReady() {
            kd.logger.Info("deployment is ready",
                "app", appName,
                "replicas", status.ReadyReplicas,
            )
            return nil
        } else {
            kd.logger.Debug("waiting for deployment",
                "app", appName,
                "ready", status.ReadyReplicas,
                "desired", status.DesiredReplicas,
            )
        }
        
        // Check context cancellation
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

## Testing Status Retrieval

```go
// Add to pkg/deployer/deployer_test.go

func TestStatus_DeploymentExists(t *testing.T) {
    // Create fake deployment with status
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-app",
            Namespace: "default",
            Labels: map[string]string{
                "app":        "test-app",
                "kudev-hash": "abc12345",
            },
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: int32Ptr(2),
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": "test-app"},
            },
        },
        Status: appsv1.DeploymentStatus{
            ReadyReplicas: 2,
            Replicas:      2,
        },
    }
    
    pod := &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-app-abc123",
            Namespace: "default",
            Labels:    map[string]string{"app": "test-app"},
        },
        Status: corev1.PodStatus{
            Phase: corev1.PodRunning,
            Conditions: []corev1.PodCondition{
                {Type: corev1.PodReady, Status: corev1.ConditionTrue},
            },
            ContainerStatuses: []corev1.ContainerStatus{
                {RestartCount: 0},
            },
        },
    }
    
    fakeClient := fake.NewSimpleClientset(deployment, pod)
    
    renderer, _ := NewRenderer(
        templates.DeploymentTemplate,
        templates.ServiceTemplate,
    )
    
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    status, err := deployer.Status(context.Background(), "test-app", "default")
    if err != nil {
        t.Fatalf("Status failed: %v", err)
    }
    
    if status.DeploymentName != "test-app" {
        t.Errorf("name = %q, want %q", status.DeploymentName, "test-app")
    }
    
    if status.ReadyReplicas != 2 {
        t.Errorf("ready = %d, want 2", status.ReadyReplicas)
    }
    
    if status.Status != "Running" {
        t.Errorf("status = %q, want %q", status.Status, "Running")
    }
    
    if status.ImageHash != "abc12345" {
        t.Errorf("hash = %q, want %q", status.ImageHash, "abc12345")
    }
    
    if !status.IsReady() {
        t.Error("IsReady() should be true")
    }
}

func TestStatus_DeploymentNotFound(t *testing.T) {
    fakeClient := fake.NewSimpleClientset()
    
    renderer, _ := NewRenderer(
        templates.DeploymentTemplate,
        templates.ServiceTemplate,
    )
    
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    _, err := deployer.Status(context.Background(), "nonexistent", "default")
    if err == nil {
        t.Error("expected error for nonexistent deployment")
    }
}

func TestStatus_Degraded(t *testing.T) {
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-app",
            Namespace: "default",
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: int32Ptr(3),
        },
        Status: appsv1.DeploymentStatus{
            ReadyReplicas: 1, // Only 1 of 3 ready
            Replicas:      3,
        },
    }
    
    fakeClient := fake.NewSimpleClientset(deployment)
    renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    status, _ := deployer.Status(context.Background(), "test-app", "default")
    
    if status.Status != "Degraded" {
        t.Errorf("status = %q, want %q", status.Status, "Degraded")
    }
}

func TestComputeStatusCode(t *testing.T) {
    tests := []struct {
        name     string
        ready    int32
        desired  int32
        pods     []PodStatus
        expected StatusCode
    }{
        {"all ready", 3, 3, nil, StatusRunning},
        {"more than desired", 4, 3, nil, StatusRunning},
        {"some ready", 1, 3, nil, StatusDegraded},
        {"none ready", 0, 3, nil, StatusPending},
        {"crash loop", 0, 3, []PodStatus{{Restarts: 10}}, StatusFailed},
        {"zero desired", 0, 0, nil, StatusUnknown},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := computeStatusCode(tt.ready, tt.desired, tt.pods)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

---

## Checklist for Task 3.5

- [ ] Create `pkg/deployer/status.go`
- [ ] Implement `Status()` method on KubernetesDeployer
- [ ] Implement `buildPodStatuses()` helper
- [ ] Implement `isPodReady()` helper
- [ ] Implement `computeStatusCode()` helper
- [ ] Implement `buildStatusMessage()` helper
- [ ] Implement `WaitForReady()` method
- [ ] Handle deployment not found
- [ ] Extract kudev-hash from labels
- [ ] Count container restarts
- [ ] Add tests for all status scenarios
- [ ] Run `go test ./pkg/deployer -v`

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 3.6** → Implement Safe Delete
3. Delete will remove resources created by kudev

---

## References

- [K8s Pod Conditions](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions)
- [Deployment Status](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#deployment-status)


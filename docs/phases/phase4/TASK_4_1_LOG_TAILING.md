# Task 4.1: Implement Log Tailing

## Overview

This task implements **real-time log streaming** from Kubernetes pods to the terminal, providing immediate feedback after deployment.

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate (K8s API, streaming, goroutines)  
**Dependencies**: Phase 3 (Deployer)  
**Files to Create**:
- `pkg/logs/tailer.go` — Log tailing implementation
- `pkg/logs/discovery.go` — Pod discovery by label
- `pkg/logs/tailer_test.go` — Tests

---

## What You're Building

A log tailer that:
1. **Discovers** pods by label selector
2. **Waits** for pods to be ready
3. **Streams** logs in real-time
4. **Follows** new log output
5. **Handles** pod restarts gracefully

---

## Complete Implementation

### Pod Discovery

```go
// pkg/logs/discovery.go

package logs

import (
    "context"
    "fmt"
    "time"
    
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/labels"
    "k8s.io/client-go/kubernetes"
)

// PodDiscovery finds pods by label selector.
type PodDiscovery struct {
    clientset kubernetes.Interface
}

// NewPodDiscovery creates a new pod discovery instance.
func NewPodDiscovery(clientset kubernetes.Interface) *PodDiscovery {
    return &PodDiscovery{clientset: clientset}
}

// DiscoverPod finds a pod by app label.
// Waits up to timeout for a pod to exist and be running.
func (pd *PodDiscovery) DiscoverPod(ctx context.Context, appName, namespace string, timeout time.Duration) (*corev1.Pod, error) {
    selector := labels.SelectorFromSet(labels.Set{"app": appName})
    
    deadline := time.Now().Add(timeout)
    
    for {
        if time.Now().After(deadline) {
            return nil, fmt.Errorf("timeout waiting for pod with label app=%s", appName)
        }
        
        pods, err := pd.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
            LabelSelector: selector.String(),
        })
        if err != nil {
            return nil, fmt.Errorf("failed to list pods: %w", err)
        }
        
        // Find a running pod
        for i := range pods.Items {
            pod := &pods.Items[i]
            if pod.Status.Phase == corev1.PodRunning {
                return pod, nil
            }
        }
        
        // Wait and retry
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(2 * time.Second):
            // Continue polling
        }
    }
}

// WaitForPodReady waits for a specific pod to be ready.
func (pd *PodDiscovery) WaitForPodReady(ctx context.Context, name, namespace string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    
    for {
        if time.Now().After(deadline) {
            return fmt.Errorf("timeout waiting for pod %s to be ready", name)
        }
        
        pod, err := pd.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
        if err != nil {
            return fmt.Errorf("failed to get pod: %w", err)
        }
        
        if isPodReady(pod) {
            return nil
        }
        
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(2 * time.Second):
            // Continue polling
        }
    }
}

func isPodReady(pod *corev1.Pod) bool {
    for _, condition := range pod.Status.Conditions {
        if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
            return true
        }
    }
    return false
}
```

### Log Tailer

```go
// pkg/logs/tailer.go

package logs

import (
    "bufio"
    "context"
    "fmt"
    "io"
    
    corev1 "k8s.io/api/core/v1"
    "k8s.io/client-go/kubernetes"
    
    "github.com/your-org/kudev/pkg/logging"
)

// LogTailer streams logs from Kubernetes pods.
type LogTailer interface {
    // TailLogs streams logs from the first pod matching the app label.
    TailLogs(ctx context.Context, appName, namespace string) error
}

// KubernetesLogTailer implements LogTailer using client-go.
type KubernetesLogTailer struct {
    clientset kubernetes.Interface
    discovery *PodDiscovery
    logger    logging.Logger
    output    io.Writer
}

// NewKubernetesLogTailer creates a new log tailer.
func NewKubernetesLogTailer(
    clientset kubernetes.Interface,
    logger logging.Logger,
    output io.Writer,
) *KubernetesLogTailer {
    return &KubernetesLogTailer{
        clientset: clientset,
        discovery: NewPodDiscovery(clientset),
        logger:    logger,
        output:    output,
    }
}

// TailLogs streams logs from pods with the given app label.
func (lt *KubernetesLogTailer) TailLogs(ctx context.Context, appName, namespace string) error {
    lt.logger.Info("waiting for pods...",
        "app", appName,
        "namespace", namespace,
    )
    
    // Wait for a running pod
    pod, err := lt.discovery.DiscoverPod(ctx, appName, namespace, 5*time.Minute)
    if err != nil {
        return fmt.Errorf("failed to discover pod: %w", err)
    }
    
    lt.logger.Info("found pod, streaming logs",
        "pod", pod.Name,
    )
    
    return lt.streamLogs(ctx, pod.Name, namespace)
}

// streamLogs streams logs from a specific pod.
func (lt *KubernetesLogTailer) streamLogs(ctx context.Context, podName, namespace string) error {
    // Configure log options
    opts := &corev1.PodLogOptions{
        Follow:     true,           // Stream new logs
        TailLines:  int64Ptr(100),  // Start with last 100 lines
        Timestamps: true,           // Include timestamps
    }
    
    // Get log stream
    req := lt.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
    stream, err := req.Stream(ctx)
    if err != nil {
        return fmt.Errorf("failed to open log stream: %w", err)
    }
    defer stream.Close()
    
    // Stream logs to output
    scanner := bufio.NewScanner(stream)
    // Increase buffer for long log lines
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, 1024*1024)
    
    for scanner.Scan() {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            fmt.Fprintln(lt.output, scanner.Text())
        }
    }
    
    if err := scanner.Err(); err != nil {
        // EOF is expected when pod terminates
        if err == io.EOF {
            return nil
        }
        return fmt.Errorf("log stream error: %w", err)
    }
    
    return nil
}

// TailLogsWithRetry streams logs with automatic reconnection on failures.
func (lt *KubernetesLogTailer) TailLogsWithRetry(ctx context.Context, appName, namespace string) error {
    for {
        err := lt.TailLogs(ctx, appName, namespace)
        
        // Check if we should stop
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        if err != nil {
            lt.logger.Info("log stream ended, reconnecting...",
                "error", err,
            )
            time.Sleep(2 * time.Second)
            continue
        }
        
        return nil
    }
}

func int64Ptr(i int64) *int64 {
    return &i
}

// Ensure KubernetesLogTailer implements LogTailer
var _ LogTailer = (*KubernetesLogTailer)(nil)
```

---

## Key Implementation Details

### 1. Pod Discovery with Retry

Wait for pods to exist before tailing:
```go
deadline := time.Now().Add(timeout)
for {
    if time.Now().After(deadline) {
        return nil, fmt.Errorf("timeout")
    }
    
    pods, _ := clientset.CoreV1().Pods(ns).List(ctx, opts)
    for _, pod := range pods.Items {
        if pod.Status.Phase == corev1.PodRunning {
            return &pod, nil
        }
    }
    
    time.Sleep(2 * time.Second)
}
```

### 2. Log Streaming Options

Configure what logs to fetch:
```go
opts := &corev1.PodLogOptions{
    Follow:     true,           // Stream new logs (like tail -f)
    TailLines:  int64Ptr(100),  // Start with recent logs
    Timestamps: true,           // Include timestamps
}
```

### 3. Context Cancellation

Always check for cancellation:
```go
for scanner.Scan() {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        fmt.Fprintln(output, scanner.Text())
    }
}
```

### 4. Automatic Reconnection

Handle pod restarts:
```go
func (lt *KubernetesLogTailer) TailLogsWithRetry(ctx context.Context, ...) error {
    for {
        err := lt.TailLogs(ctx, ...)
        if ctx.Err() != nil {
            return ctx.Err()  // Context cancelled, stop
        }
        // Reconnect after delay
        time.Sleep(2 * time.Second)
    }
}
```

---

## Testing

```go
// pkg/logs/tailer_test.go

package logs

import (
    "bytes"
    "context"
    "testing"
    "time"
    
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes/fake"
)

type mockLogger struct{}

func (m *mockLogger) Info(msg string, kv ...interface{})  {}
func (m *mockLogger) Debug(msg string, kv ...interface{}) {}
func (m *mockLogger) Error(msg string, kv ...interface{}) {}

func TestDiscoverPod_Found(t *testing.T) {
    pod := &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "myapp-abc123",
            Namespace: "default",
            Labels:    map[string]string{"app": "myapp"},
        },
        Status: corev1.PodStatus{
            Phase: corev1.PodRunning,
        },
    }
    
    fakeClient := fake.NewSimpleClientset(pod)
    discovery := NewPodDiscovery(fakeClient)
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    foundPod, err := discovery.DiscoverPod(ctx, "myapp", "default", 10*time.Second)
    if err != nil {
        t.Fatalf("DiscoverPod failed: %v", err)
    }
    
    if foundPod.Name != "myapp-abc123" {
        t.Errorf("wrong pod found: %s", foundPod.Name)
    }
}

func TestDiscoverPod_Timeout(t *testing.T) {
    // Empty cluster - no pods
    fakeClient := fake.NewSimpleClientset()
    discovery := NewPodDiscovery(fakeClient)
    
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    _, err := discovery.DiscoverPod(ctx, "myapp", "default", 100*time.Millisecond)
    if err == nil {
        t.Error("expected timeout error")
    }
}

func TestIsPodReady(t *testing.T) {
    tests := []struct {
        name     string
        pod      *corev1.Pod
        expected bool
    }{
        {
            name: "ready pod",
            pod: &corev1.Pod{
                Status: corev1.PodStatus{
                    Conditions: []corev1.PodCondition{
                        {Type: corev1.PodReady, Status: corev1.ConditionTrue},
                    },
                },
            },
            expected: true,
        },
        {
            name: "not ready pod",
            pod: &corev1.Pod{
                Status: corev1.PodStatus{
                    Conditions: []corev1.PodCondition{
                        {Type: corev1.PodReady, Status: corev1.ConditionFalse},
                    },
                },
            },
            expected: false,
        },
        {
            name: "no conditions",
            pod: &corev1.Pod{
                Status: corev1.PodStatus{},
            },
            expected: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := isPodReady(tt.pod)
            if result != tt.expected {
                t.Errorf("isPodReady() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

---

## Checklist for Task 4.1

- [ ] Create `pkg/logs/discovery.go`
- [ ] Implement `PodDiscovery` struct
- [ ] Implement `DiscoverPod()` method
- [ ] Implement `WaitForPodReady()` method
- [ ] Create `pkg/logs/tailer.go`
- [ ] Define `LogTailer` interface
- [ ] Implement `KubernetesLogTailer` struct
- [ ] Implement `TailLogs()` method
- [ ] Implement `TailLogsWithRetry()` method
- [ ] Handle context cancellation
- [ ] Create `pkg/logs/tailer_test.go`
- [ ] Test pod discovery
- [ ] Test timeout handling
- [ ] Run `go test ./pkg/logs -v`

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 4.2** → Implement Port Forwarding
3. Port forwarder will enable local access to pod

---

## References

- [K8s Pod Logs API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#read-log-pod-v1-core)
- [client-go Logs](https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1)


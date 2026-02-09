# Task 4.2: Implement Port Forwarding

## Overview

This task implements **port forwarding** from localhost to Kubernetes pods, enabling local development access to deployed applications.

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate (client-go portforward, networking)  
**Dependencies**: Task 4.1 (Pod Discovery)  
**Files to Create**:
- `pkg/portfwd/forwarder.go` — Port forwarding implementation
- `pkg/portfwd/forwarder_test.go` — Tests

---

## What You're Building

A port forwarder that:
1. **Waits** for pods to be ready
2. **Checks** local port availability
3. **Establishes** port forward to pod
4. **Runs** in background goroutine
5. **Handles** reconnection on pod restart

---

## Complete Implementation

```go
// pkg/portfwd/forwarder.go

package portfwd

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "net/url"
    "time"
    
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/portforward"
    "k8s.io/client-go/transport/spdy"
    
    "github.com/your-org/kudev/pkg/logging"
    "github.com/your-org/kudev/pkg/logs"
)

// PortForwarder forwards local ports to Kubernetes pods.
type PortForwarder interface {
    // Forward starts port forwarding in the background.
    // Returns when forwarding is established.
    Forward(ctx context.Context, appName, namespace string, localPort, podPort int32) error
    
    // Stop terminates port forwarding.
    Stop()
}

// KubernetesPortForwarder implements PortForwarder using client-go.
type KubernetesPortForwarder struct {
    clientset  kubernetes.Interface
    restConfig *rest.Config
    discovery  *logs.PodDiscovery
    logger     logging.Logger
    
    // Internal state
    stopChan chan struct{}
    readyChan chan struct{}
}

// NewKubernetesPortForwarder creates a new port forwarder.
func NewKubernetesPortForwarder(
    clientset kubernetes.Interface,
    restConfig *rest.Config,
    logger logging.Logger,
) *KubernetesPortForwarder {
    return &KubernetesPortForwarder{
        clientset:  clientset,
        restConfig: restConfig,
        discovery:  logs.NewPodDiscovery(clientset),
        logger:     logger,
    }
}

// Forward starts port forwarding to a pod.
func (pf *KubernetesPortForwarder) Forward(ctx context.Context, appName, namespace string, localPort, podPort int32) error {
    // 1. Check port availability
    if err := checkPortAvailable(localPort); err != nil {
        return fmt.Errorf("port %d is not available: %w\n\nTry a different port with --local-port flag", localPort, err)
    }
    
    pf.logger.Info("waiting for pod to be ready...",
        "app", appName,
        "namespace", namespace,
    )
    
    // 2. Wait for a running pod
    pod, err := pf.discovery.DiscoverPod(ctx, appName, namespace, 5*time.Minute)
    if err != nil {
        return fmt.Errorf("failed to find pod: %w", err)
    }
    
    pf.logger.Info("found pod",
        "pod", pod.Name,
    )
    
    // 3. Create channels
    pf.stopChan = make(chan struct{}, 1)
    pf.readyChan = make(chan struct{})
    
    // 4. Build port forward URL
    path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, pod.Name)
    hostURL, err := url.Parse(pf.restConfig.Host)
    if err != nil {
        return fmt.Errorf("failed to parse host URL: %w", err)
    }
    hostURL.Path = path
    
    // 5. Create SPDY transport
    transport, upgrader, err := spdy.RoundTripperFor(pf.restConfig)
    if err != nil {
        return fmt.Errorf("failed to create transport: %w", err)
    }
    
    dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, hostURL)
    
    // 6. Create port forwarder
    ports := []string{fmt.Sprintf("%d:%d", localPort, podPort)}
    
    // Use io.Discard for output (we'll log manually)
    fw, err := portforward.New(dialer, ports, pf.stopChan, pf.readyChan, nil, nil)
    if err != nil {
        return fmt.Errorf("failed to create port forwarder: %w", err)
    }
    
    // 7. Start forwarding in goroutine
    errChan := make(chan error, 1)
    go func() {
        errChan <- fw.ForwardPorts()
    }()
    
    // 8. Wait for ready or error
    select {
    case <-pf.readyChan:
        pf.logger.Info("port forwarding ready",
            "local", fmt.Sprintf("localhost:%d", localPort),
            "pod", fmt.Sprintf("%s:%d", pod.Name, podPort),
        )
        
        // Start background monitor
        go pf.monitor(ctx, errChan, appName, namespace, localPort, podPort)
        
        return nil
        
    case err := <-errChan:
        return fmt.Errorf("port forwarding failed: %w", err)
        
    case <-ctx.Done():
        pf.Stop()
        return ctx.Err()
    }
}

// monitor watches for errors and attempts reconnection.
func (pf *KubernetesPortForwarder) monitor(ctx context.Context, errChan chan error, appName, namespace string, localPort, podPort int32) {
    for {
        select {
        case <-ctx.Done():
            return
            
        case err := <-errChan:
            if err != nil {
                pf.logger.Info("port forward disconnected, reconnecting...",
                    "error", err,
                )
                
                // Wait a bit before reconnecting
                time.Sleep(2 * time.Second)
                
                // Try to reconnect
                if ctx.Err() == nil {
                    if err := pf.Forward(ctx, appName, namespace, localPort, podPort); err != nil {
                        pf.logger.Error("reconnection failed",
                            "error", err,
                        )
                    }
                }
            }
            return
        }
    }
}

// Stop terminates port forwarding.
func (pf *KubernetesPortForwarder) Stop() {
    if pf.stopChan != nil {
        close(pf.stopChan)
    }
}

// checkPortAvailable checks if a local port is available.
func checkPortAvailable(port int32) error {
    ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return err
    }
    ln.Close()
    return nil
}

// SuggestAlternativePort finds an available port near the requested one.
func SuggestAlternativePort(preferredPort int32) (int32, error) {
    // Try ports around the preferred one
    for delta := int32(0); delta < 100; delta++ {
        for _, p := range []int32{preferredPort + delta, preferredPort - delta} {
            if p < 1024 || p > 65535 {
                continue
            }
            if checkPortAvailable(p) == nil {
                return p, nil
            }
        }
    }
    return 0, fmt.Errorf("no available ports found near %d", preferredPort)
}

// Ensure KubernetesPortForwarder implements PortForwarder
var _ PortForwarder = (*KubernetesPortForwarder)(nil)
```

---

## Key Implementation Details

### 1. Port Availability Check

Always check before forwarding:
```go
func checkPortAvailable(port int32) error {
    ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return err  // Port in use
    }
    ln.Close()
    return nil
}
```

### 2. SPDY Transport

client-go uses SPDY for port forwarding:
```go
transport, upgrader, err := spdy.RoundTripperFor(restConfig)
dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, hostURL)
```

### 3. Channels for Control

Use channels for coordination:
```go
stopChan := make(chan struct{}, 1)   // Signal to stop
readyChan := make(chan struct{})     // Signal when ready

// Stop by closing
close(stopChan)
```

### 4. Background Goroutine

Run forwarding in background:
```go
go func() {
    errChan <- fw.ForwardPorts()  // Blocks until stopped or error
}()

// Wait for ready
<-readyChan
```

### 5. Reconnection on Failure

Monitor and reconnect:
```go
go func() {
    for {
        select {
        case err := <-errChan:
            // Reconnect logic
        case <-ctx.Done():
            return
        }
    }
}()
```

---

## Testing

```go
// pkg/portfwd/forwarder_test.go

package portfwd

import (
    "net"
    "testing"
)

func TestCheckPortAvailable_Free(t *testing.T) {
    // Find a free port
    ln, err := net.Listen("tcp", ":0")
    if err != nil {
        t.Fatal(err)
    }
    port := ln.Addr().(*net.TCPAddr).Port
    ln.Close()
    
    // Port should be available now
    err = checkPortAvailable(int32(port))
    if err != nil {
        t.Errorf("port should be available: %v", err)
    }
}

func TestCheckPortAvailable_InUse(t *testing.T) {
    // Occupy a port
    ln, err := net.Listen("tcp", ":0")
    if err != nil {
        t.Fatal(err)
    }
    defer ln.Close()
    
    port := ln.Addr().(*net.TCPAddr).Port
    
    // Port should NOT be available
    err = checkPortAvailable(int32(port))
    if err == nil {
        t.Error("port should NOT be available")
    }
}

func TestSuggestAlternativePort(t *testing.T) {
    // Occupy a port
    ln, err := net.Listen("tcp", ":0")
    if err != nil {
        t.Fatal(err)
    }
    defer ln.Close()
    
    occupiedPort := int32(ln.Addr().(*net.TCPAddr).Port)
    
    // Should find an alternative
    alt, err := SuggestAlternativePort(occupiedPort)
    if err != nil {
        t.Fatalf("SuggestAlternativePort failed: %v", err)
    }
    
    if alt == occupiedPort {
        t.Error("should suggest different port")
    }
    
    // Alternative should be available
    if err := checkPortAvailable(alt); err != nil {
        t.Errorf("suggested port %d not available: %v", alt, err)
    }
}

func TestSuggestAlternativePort_PreferredAvailable(t *testing.T) {
    // Find a free port
    ln, err := net.Listen("tcp", ":0")
    if err != nil {
        t.Fatal(err)
    }
    port := int32(ln.Addr().(*net.TCPAddr).Port)
    ln.Close()
    
    // Should return preferred port if available
    alt, err := SuggestAlternativePort(port)
    if err != nil {
        t.Fatalf("SuggestAlternativePort failed: %v", err)
    }
    
    if alt != port {
        t.Errorf("should return preferred port, got %d", alt)
    }
}
```

---

## Usage Example

```go
// In cmd/commands/up.go

func startPortForward(ctx context.Context, cfg *config.DeploymentConfig) error {
    // Get K8s config
    restConfig, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
    clientset, _ := kubernetes.NewForConfig(restConfig)
    
    // Create forwarder
    forwarder := portfwd.NewKubernetesPortForwarder(clientset, restConfig, logger)
    
    // Start forwarding
    localPort := cfg.Spec.LocalPort
    podPort := cfg.Spec.ServicePort
    
    if err := forwarder.Forward(ctx, cfg.Metadata.Name, cfg.Spec.Namespace, localPort, podPort); err != nil {
        // Try alternative port
        if altPort, err := portfwd.SuggestAlternativePort(localPort); err == nil {
            logger.Info("trying alternative port", "port", altPort)
            return forwarder.Forward(ctx, cfg.Metadata.Name, cfg.Spec.Namespace, altPort, podPort)
        }
        return err
    }
    
    fmt.Printf("\nApplication available at: http://localhost:%d\n\n", localPort)
    
    return nil
}
```

---

## Checklist for Task 4.2

- [ ] Create `pkg/portfwd/forwarder.go`
- [ ] Define `PortForwarder` interface
- [ ] Implement `KubernetesPortForwarder` struct
- [ ] Implement `Forward()` method
- [ ] Implement `Stop()` method
- [ ] Implement `checkPortAvailable()` helper
- [ ] Implement `SuggestAlternativePort()` helper
- [ ] Add reconnection monitoring
- [ ] Create `pkg/portfwd/forwarder_test.go`
- [ ] Test port availability check
- [ ] Test alternative port suggestion
- [ ] Run `go test ./pkg/portfwd -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Not checking port availability first
```go
// Wrong - confusing error if port in use
forwarder.Forward(...)

// Right - check and suggest alternative
if err := checkPortAvailable(port); err != nil {
    alt, _ := SuggestAlternativePort(port)
    fmt.Printf("Port %d in use, try %d\n", port, alt)
}
```

❌ **Mistake 2**: Blocking on Forward()
```go
// Wrong - blocks forever
forwarder.Forward(ctx, ...)
fmt.Println("This never prints")

// Right - ForwardPorts runs in goroutine
go fw.ForwardPorts()
<-readyChan  // Returns when ready
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 4.3** → CLI Orchestration
3. CLI will coordinate logs + port forward

---

## References

- [client-go Port Forward](https://pkg.go.dev/k8s.io/client-go/tools/portforward)
- [SPDY Transport](https://pkg.go.dev/k8s.io/client-go/transport/spdy)

